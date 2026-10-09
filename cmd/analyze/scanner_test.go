package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func writeFileWithSize(t testing.TB, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	content := make([]byte, size)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestGetDirectoryLogicalSizeWithExclude(t *testing.T) {
	base := t.TempDir()
	homeFile := filepath.Join(base, "fileA")
	libFile := filepath.Join(base, "Library", "fileB")
	projectLibFile := filepath.Join(base, "Projects", "Library", "fileC")

	writeFileWithSize(t, homeFile, 100)
	writeFileWithSize(t, libFile, 200)
	writeFileWithSize(t, projectLibFile, 300)

	total, err := getDirectoryLogicalSizeWithExclude(base, "")
	if err != nil {
		t.Fatalf("getDirectoryLogicalSizeWithExclude (no exclude) error: %v", err)
	}
	if total != 600 {
		t.Fatalf("expected total 600 bytes, got %d", total)
	}

	excluding, err := getDirectoryLogicalSizeWithExclude(base, filepath.Join(base, "Library"))
	if err != nil {
		t.Fatalf("getDirectoryLogicalSizeWithExclude (exclude Library) error: %v", err)
	}
	if excluding != 400 {
		t.Fatalf("expected 400 bytes when excluding top-level Library, got %d", excluding)
	}
}

func TestGetDirectorySizeFromDuSkippingImmediateChildDoesNotMeasureExcludedPath(t *testing.T) {
	base := t.TempDir()
	excluded := filepath.Join(base, "Library")
	included := filepath.Join(base, "Documents")
	if err := os.MkdirAll(excluded, 0o755); err != nil {
		t.Fatalf("mkdir excluded: %v", err)
	}
	if err := os.MkdirAll(included, 0o755); err != nil {
		t.Fatalf("mkdir included: %v", err)
	}

	var measured []string
	size, err := getDirectorySizeFromDuSkippingImmediateChild(base, excluded, func(path string) (int64, error) {
		measured = append(measured, path)
		return 100, nil
	})
	if err != nil {
		t.Fatalf("getDirectorySizeFromDuSkippingImmediateChild: %v", err)
	}
	if size < 100 {
		t.Fatalf("expected included directory size in total, got %d", size)
	}
	if len(measured) != 1 || measured[0] != included {
		t.Fatalf("expected to measure only %s, measured %#v", included, measured)
	}
}

// du exits non-zero when part of a tree is unreadable, but it still prints the
// aggregate for the part it could read. Both halves have to survive: the bytes,
// so no caller re-walks the same tree, and the error, so the result is reported
// as partial rather than as a full measurement.
func TestDuKeepsMeasuredBytesWhenItExitsNonZero(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "node_modules")
	writeFileWithSize(t, filepath.Join(target, "file"), 1)

	stubDir := t.TempDir()
	// Exercise the real external-command boundary: a subtotal on stdout plus a
	// non-zero exit, which is what GNU du does for an unreadable descendant.
	stub := "#!/bin/sh\nprintf '8\tpartial\\n'\nexit 1\n"
	if err := os.WriteFile(filepath.Join(stubDir, "du"), []byte(stub), 0o755); err != nil {
		t.Fatalf("write du stub: %v", err)
	}
	t.Setenv("PATH", stubDir)

	size, err := getDirectorySizeFromDu(context.Background(), target)
	if size != 8192 {
		t.Fatalf("partial du output was dropped: size=%d err=%v", size, err)
	}
	if err == nil {
		t.Fatal("a failing du must not report a complete measurement")
	}
	if got := measurementState(size, err); got != scanPartial {
		t.Fatalf("a du run with bytes and a failure must classify as partial, got %s", got)
	}
}

// One unreadable child must not discard the sizes already measured for its siblings.
func TestSizeSkippingImmediateChildKeepsPartialTotals(t *testing.T) {
	base := t.TempDir()
	excluded := filepath.Join(base, "Library")
	broken := filepath.Join(base, "Broken")
	intact := filepath.Join(base, "Documents")
	for _, dir := range []string{excluded, broken, intact} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	measure := func(brokenErr error) (int64, error) {
		return getDirectorySizeFromDuSkippingImmediateChild(base, excluded, func(path string) (int64, error) {
			switch path {
			case broken:
				// 4096 bytes were measured before the failure, and they count.
				return 4096, brokenErr
			case intact:
				return 8192, nil
			default:
				t.Errorf("unexpected measurement target: %s", path)
				return 0, nil
			}
		})
	}

	complete, err := measure(nil)
	if err != nil {
		t.Fatalf("clean measurement failed: %v", err)
	}

	partial, err := measure(errors.New("permission denied"))
	if err == nil {
		t.Fatal("the child failure was dropped, so the caller would cache an incomplete tree as complete")
	}
	if partial != complete {
		t.Fatalf("the failing child discarded its siblings' bytes: complete=%d partial=%d", complete, partial)
	}
	if got := measurementState(partial, err); got != scanPartial {
		t.Fatalf("partial child run must classify as partial, got %s", got)
	}
}

// A folded directory measured by a partially failing du must keep that number.
// The old "any error means re-measure" guard threw away 8 KB of real
// measurement and replaced it with a slow recursive walk of the same tree.
func TestFoldedDirectoryKeepsPartialDuSize(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "root")
	folded := filepath.Join(root, "node_modules")
	writeFileWithSize(t, filepath.Join(folded, "file"), 1)

	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "du"), []byte("#!/bin/sh\nprintf '8\tpartial\\n'\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write du stub: %v", err)
	}
	t.Setenv("PATH", stubDir)

	var files, dirs, bytes int64
	current := &atomic.Value{}
	current.Store("")
	result, err := scanPathConcurrentAllEntries(context.Background(), root, &files, &dirs, &bytes, current)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	for _, entry := range result.Entries {
		if filepath.Clean(entry.Path) == filepath.Clean(folded) {
			if entry.Size != 8192 {
				t.Fatalf("partial du size was replaced by a re-walk for %s: got %d, want 8192 (%+v)", entry.Path, entry.Size, result)
			}
			return
		}
	}
	t.Fatalf("folded directory missing from the scan result: %+v", result)
}

// The live-scan entry point takes the same folded shortcut, so it must keep the
// same partial number as the full scan does.
func TestLiveScanFoldedDirectoryKeepsPartialDuSize(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	folded := filepath.Join(home, "root", "node_modules")
	writeFileWithSize(t, filepath.Join(folded, "file"), 1)

	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "du"), []byte("#!/bin/sh\nprintf '8\tpartial\\n'\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write du stub: %v", err)
	}
	t.Setenv("PATH", stubDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	publication := newScanPublication(ctx, cancel)
	limiter := newScanLimiter(1)
	largeFileMinSize := int64(largeFileWarmupMinSize)
	var files, dirs, bytes int64
	current := &atomic.Value{}
	current.Store("")

	result, err := scanLiveTarget(ctx,
		liveScanTarget{name: "folded", path: folded, kind: liveScanTargetFoldedDirectory},
		make(chan fileEntry, 16), &largeFileMinSize, limiter,
		&files, &dirs, &bytes, current, scanCacheBypass, publication)
	if err != nil {
		t.Fatalf("live scan: %v", err)
	}
	if result.TotalSize != 8192 {
		t.Fatalf("live scan dropped the partial du measurement: got %d, want 8192 (%+v)", result.TotalSize, result)
	}
}

func TestGetDirectorySizeFromDuWithIgnoresSkipsCloudPlaceholderTree(t *testing.T) {
	base := t.TempDir()
	writeFileWithSize(t, filepath.Join(base, "Application Support", "state.dat"), 4096)
	writeFileWithSize(t, filepath.Join(base, "Mobile Documents", "cloud.dat"), 1024*1024)

	withoutIgnore, err := getDirectorySizeFromDuWithExcludeAndIgnores(context.Background(), base, "", nil)
	if err != nil {
		t.Fatalf("getDirectorySizeFromDuWithExcludeAndIgnores without ignore: %v", err)
	}
	withIgnore, err := getDirectorySizeFromDuWithExcludeAndIgnores(context.Background(), base, "", []string{"Mobile Documents"})
	if err != nil {
		t.Fatalf("getDirectorySizeFromDuWithExcludeAndIgnores with ignore: %v", err)
	}
	if withIgnore >= withoutIgnore {
		t.Fatalf("expected ignored Mobile Documents to reduce size, got ignored=%d without=%d", withIgnore, withoutIgnore)
	}
	if withIgnore <= 0 {
		t.Fatalf("expected non-zero size for included files, got %d", withIgnore)
	}
}

func TestValidateDuIgnoreNameRejectsPathPatterns(t *testing.T) {
	for _, name := range []string{"", "../Library", "Library/Developer", "bad\x00name"} {
		if err := validateDuIgnoreName(name); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
	if err := validateDuIgnoreName("Mobile Documents"); err != nil {
		t.Fatalf("expected basename ignore to be accepted: %v", err)
	}
}

func BenchmarkGetDirectorySizeFromDuWithExcludeHomeLibrary(b *testing.B) {
	base := b.TempDir()
	libraryDir := filepath.Join(base, "Library")
	for dirIdx := range 250 {
		for fileIdx := range 20 {
			writeFileWithSize(
				b,
				filepath.Join(libraryDir, "bulk", fmt.Sprintf("dir-%03d", dirIdx), "bucket", fmt.Sprintf("file-%03d.dat", fileIdx)),
				16,
			)
		}
	}
	writeFileWithSize(b, filepath.Join(base, "Documents", "keep.dat"), 4096)

	excludePath := filepath.Join(base, "Library")
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		size, err := getDirectorySizeFromDuWithExclude(context.Background(), base, excludePath)
		if err != nil {
			b.Fatalf("getDirectorySizeFromDuWithExclude: %v", err)
		}
		if size <= 0 {
			b.Fatalf("expected non-zero size, got %d", size)
		}
	}
}
