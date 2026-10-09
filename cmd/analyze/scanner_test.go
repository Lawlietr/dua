package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
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

	total, err := getDirectoryLogicalSizeWithExclude(context.Background(), base, "", nil)
	if err != nil {
		t.Fatalf("getDirectoryLogicalSizeWithExclude (no exclude) error: %v", err)
	}
	if total != 600 {
		t.Fatalf("expected total 600 bytes, got %d", total)
	}

	excluding, err := getDirectoryLogicalSizeWithExclude(context.Background(), base, filepath.Join(base, "Library"), nil)
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
	size, err := getDirectorySizeFromDuSkippingImmediateChild(context.Background(), base, excluded, func(path string) (int64, error) {
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
		return getDirectorySizeFromDuSkippingImmediateChild(context.Background(), base, excluded, func(path string) (int64, error) {
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

// The overview path is the one place a partial total used to be thrown away and
// replaced by a second, much slower measurement of the same tree.
func TestOverviewMeasurementKeepsPartialDuResult(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "root")
	writeFileWithSize(t, filepath.Join(root, "readable"), 4096)

	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "du"), []byte("#!/bin/sh\nprintf '8\tpartial\\n'\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write du stub: %v", err)
	}
	t.Setenv("PATH", stubDir)

	size, err := measureOverviewSize(context.Background(), root)
	if size != 8192 {
		t.Fatalf("overview dropped the partial du total: size=%d err=%v", size, err)
	}
	if err == nil {
		t.Fatal("a partial overview total must not be reported as complete")
	}
	if got := measurementState(size, err); got != scanPartial {
		t.Fatalf("expected a partial classification, got %s", got)
	}

	// A cached overview number carries no coverage marker, so a partial total
	// must never overwrite the complete snapshot that is already on disk.
	if cached, cacheErr := loadOverviewCachedSize(root); cacheErr == nil && cached == 8192 {
		t.Fatal("partial overview total was published as a complete cache entry")
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := measureOverviewSize(cancelCtx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was not propagated through the measurement helpers: %v", err)
	}
}

// The fallback walk is only reached when du produced nothing usable, so it has
// to report its own gaps instead of returning a quietly smaller total.
func TestLogicalSizeFallbackReportsItsOwnGaps(t *testing.T) {
	root := t.TempDir()
	writeFileWithSize(t, filepath.Join(root, "readable"), 4096)

	total, err := getDirectoryLogicalSizeWithExclude(context.Background(), root, "", nil)
	if err != nil {
		t.Fatalf("clean fallback walk failed: %v", err)
	}
	if got := measurementState(total, nil); got != scanComplete {
		t.Fatalf("a walk over a readable tree must classify as complete, got %s", got)
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := getDirectoryLogicalSizeWithExclude(cancelCtx, root, "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("the fallback walk ignored its deadline: %v", err)
	}
}

// The Downloads insight used to measure directories with its own du call that
// dropped every failure. It now shares the overview runner, so a partial read is
// reported instead of silently shrinking the total.
func TestOldDownloadsReportsPartialMeasurement(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().AddDate(0, 0, -120)
	subdir := filepath.Join(dir, "old-dir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chtimes(subdir, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "du"), []byte("#!/bin/sh\nprintf '8\tpartial\\n'\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write du stub: %v", err)
	}
	t.Setenv("PATH", stubDir)

	size, err := measureOldDownloads(context.Background(), dir, 90)
	if size != 8192 {
		t.Fatalf("the Downloads insight dropped the partial du total: size=%d err=%v", size, err)
	}
	if err == nil {
		t.Fatal("a partial Downloads measurement must not be reported as complete")
	}
	if got := measurementState(size, err); got != scanPartial {
		t.Fatalf("expected a partial classification, got %s", got)
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

// A readable root must not turn an unreadable descendant into a measured zero,
// and a partial scan must not overwrite the good cache entry with a lower bound.
func TestScanUnreadableDescendantPreservesCoverageAndGoodCache(t *testing.T) {
	if runTestWithoutPrivileges(t) {
		return // the re-executed child runs the body as a non-root uid
	}
	// The fixture compares measured bytes across two scans of the same tree, so
	// it needs a filesystem that reports real allocation for its files.
	skipIfBlockAccountingUnreliable(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "root")
	child := filepath.Join(root, "child")
	locked := filepath.Join(child, "locked")
	writeFileWithSize(t, filepath.Join(child, "readable"), 4096)
	writeFileWithSize(t, filepath.Join(locked, "hidden"), 1<<20)

	scan := func() scanResult {
		t.Helper()
		var files, dirs, bytes int64
		current := &atomic.Value{}
		current.Store("")
		result, err := scanPathConcurrentAllEntries(context.Background(), root, &files, &dirs, &bytes, current)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}

	good := scan()
	if good.State != scanComplete {
		t.Fatalf("initial scan state = %s, want complete", good.State)
	}
	if err := saveCacheToDisk(root, good); err != nil {
		t.Fatal(err)
	}

	lockDirFromReader(t, locked)

	partial := scan()
	// The exact bytes lost depend on the filesystem's block accounting, so the
	// invariant is the one that matters: the unreadable subtree is gone, the
	// readable bytes survive, and the result says it is a lower bound.
	if partial.State != scanPartial {
		t.Fatalf("partial scan must be marked partial: state=%s", partial.State)
	}
	if partial.TotalSize <= 0 || partial.TotalSize >= good.TotalSize {
		t.Fatalf("partial result lost coverage or readable bytes: good=%d partial=%d", good.TotalSize, partial.TotalSize)
	}
	if len(partial.Entries) == 0 {
		t.Fatal("partial scan produced no rows")
	}
	for _, entry := range partial.Entries {
		if entry.Path == child && entry.State != scanPartial {
			t.Fatalf("child coverage not propagated: state=%s", entry.State)
		}
	}

	if err := saveCacheToDisk(root, partial); err != nil {
		t.Fatal(err)
	}
	cached, err := loadCacheFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	if cached.TotalSize != good.TotalSize {
		t.Fatalf("partial scan replaced a good cache entry: got %d, want %d", cached.TotalSize, good.TotalSize)
	}

	if err := os.Chmod(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	recovered := scan()
	if recovered.State != scanComplete || recovered.TotalSize != good.TotalSize {
		t.Fatalf("recovery after the permission came back: state=%s total=%d", recovered.State, recovered.TotalSize)
	}
}

// A folded directory is sized with du instead of being walked, so its coverage
// has to come from the du result itself: the row keeps the bytes du could
// measure and says they are a lower bound, and the tree it belongs to is not
// reported as complete.
func TestScanFoldedDirWithUnreadableDescendantKeepsCoverage(t *testing.T) {
	if runTestWithoutPrivileges(t) {
		return // the re-executed child runs the body as a non-root uid
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "root")
	folded := filepath.Join(root, ".git")
	locked := filepath.Join(folded, "objects")
	writeFileWithSize(t, filepath.Join(folded, "HEAD"), 4096)
	writeFileWithSize(t, filepath.Join(locked, "pack"), 1<<20)
	lockDirFromReader(t, locked)

	var files, dirs, bytes int64
	current := &atomic.Value{}
	current.Store("")
	result, err := scanPathConcurrentAllEntries(context.Background(), root, &files, &dirs, &bytes, current)
	if err != nil {
		t.Fatal(err)
	}

	if result.State != scanPartial {
		t.Fatalf("tree coverage = %s, want partial", result.State)
	}
	var row *dirEntry
	for i := range result.Entries {
		if result.Entries[i].Path == folded {
			row = &result.Entries[i]
		}
	}
	if row == nil {
		t.Fatalf("the folded directory produced no row: %+v", result.Entries)
	}
	if row.State != scanPartial {
		t.Fatalf("folded row coverage = %s, want partial", row.State)
	}
	if row.Size <= 0 {
		t.Fatalf("folded row lost the bytes du could measure: %d", row.Size)
	}
}
