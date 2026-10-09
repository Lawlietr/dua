package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// permissionFixtureChildEnv marks the re-executed child, which already runs
// without privileges. It also stops runWithoutPrivileges from recursing.
const permissionFixtureChildEnv = "DUA_TEST_UNPRIVILEGED_CHILD"

// runTestWithoutPrivileges makes the calling test run in a process that really
// cannot read a mode-0000 directory, and reports whether the caller must stop.
//
// A chmod-0 fixture proves nothing to the root user, and both our CI and most
// containers run tests as root, so such tests used to skip themselves and the
// permission-denied code path went untested. Instead, re-run this package's own
// test binary inside a private user namespace: outside the namespace we become
// the overflow uid, DAC checks apply, and the fixture behaves like it does on a
// normal user's machine. The parent reports the child's failure (or skip).
//
// Call it as the first line of the test, then keep writing the body as usual:
//
//	if runTestWithoutPrivileges(t) {
//		return // the re-executed child runs the body
//	}
//
// A test that already runs as a non-root user continues in process, so the
// helper never weakens what the test asserts - it only moves it to a process
// where the assertion means something.
func runTestWithoutPrivileges(t *testing.T) bool {
	t.Helper()

	if os.Getenv(permissionFixtureChildEnv) == "1" {
		if os.Geteuid() == 0 {
			t.Fatal("re-executed for the permission fixture but still root")
		}
		return false
	}
	if os.Geteuid() != 0 {
		return false
	}
	if runtime.GOOS != "linux" {
		t.Skipf("permission fixture needs Linux user namespaces (running as root on %s)", runtime.GOOS)
	}
	if _, err := exec.LookPath("unshare"); err != nil {
		t.Skipf("cannot drop privileges for the permission fixture: %v", err)
		return false
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}
	// The go tool creates its build temp dir 0700; a process outside the user
	// namespace could not traverse it to reach the binary. Widen only the dirs
	// inside TMPDIR, and only the ones above the already-world-readable binary.
	for dir := filepath.Dir(exe); dir != "/" && isUnderTempDir(dir); dir = filepath.Dir(dir) {
		if info, err := os.Stat(dir); err == nil && info.Mode().Perm()&0o055 != 0o055 {
			if err := os.Chmod(dir, info.Mode().Perm()|0o055); err != nil {
				t.Skipf("cannot make the test binary reachable to an unprivileged process: %v", err)
			}
		}
	}

	cmd := exec.Command("unshare", "-U", exe, "-test.run", "^"+regexp.QuoteMeta(t.Name())+"$", "-test.timeout=120s")
	cmd.Env = append(os.Environ(), permissionFixtureChildEnv+"=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unprivileged run of %s failed: %v\n%s", t.Name(), err, output)
	}
	if bytes.Contains(output, []byte("--- SKIP")) {
		t.Skip("the unprivileged run skipped itself")
	}
	return true
}

func isUnderTempDir(path string) bool {
	tmp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(tmp, resolved)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "..")
}

// lockDirFromReader makes a directory unreadable while it is still addressable
// by name, and restores it when the test ends. Use it in a test guarded by
// runTestWithoutPrivileges; as root the mode bits would block nothing.
func lockDirFromReader(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o755) })
}

// skipIfBlockAccountingUnreliable guards size assertions that rely on a file's
// st_blocks matching its logical size. Some containerized filesystems (ZFS in
// dev environments) report a single block per file regardless of content, which
// collapses sparse/actual-usage accounting and breaks large-file detection.
func skipIfBlockAccountingUnreliable(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "probe.bin")
	if err := os.WriteFile(path, make([]byte, 4<<20), 0o644); err != nil {
		t.Fatalf("write probe: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat probe: %v", err)
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if stat.Blocks*512 < info.Size()/2 {
			t.Skip("filesystem under-reports allocated blocks; size accounting unreliable here")
		}
	}
}

func skipIfFinderUnavailable(t *testing.T) {
	t.Helper()

	if os.Getenv("CI") != "" {
		t.Skip("Skipping Finder-dependent test in CI")
	}
	if os.Getenv("MOLE_SKIP_FINDER_TESTS") == "1" {
		t.Skip("Skipping Finder-dependent test via MOLE_SKIP_FINDER_TESTS")
	}
	if _, err := exec.LookPath("osascript"); err != nil {
		t.Skipf("Skipping Finder-dependent test, osascript unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "osascript", "-e", `tell application "Finder" to get name`)
	output, err := cmd.CombinedOutput()
	text := strings.ToLower(string(output))
	if ctx.Err() == context.DeadlineExceeded {
		t.Skip("Skipping Finder-dependent test, Finder probe timed out")
	}
	if strings.Contains(text, "connection invalid") || strings.Contains(text, "can’t get application \"finder\"") || strings.Contains(text, "can't get application \"finder\"") {
		t.Skipf("Skipping Finder-dependent test, Finder probe indicates unavailable session: %s", strings.TrimSpace(string(output)))
	}
	if err != nil {
		reason := strings.TrimSpace(string(output))
		if reason == "" {
			reason = err.Error()
		}
		t.Skipf("Skipping Finder-dependent test, Finder unavailable: %s", reason)
	}
}

// A mode-0000 directory proves nothing to the root user, so a permission test
// that runs as root asserts nothing. This test pins both halves of the
// mechanism, and it is expected to run (not skip) on a root CI runner: the
// parent shows that root reads the fixture anyway, the re-executed child shows
// that it cannot.
func TestPermissionFixtureDropsReadAccess(t *testing.T) {
	if os.Geteuid() == 0 && os.Getenv(permissionFixtureChildEnv) != "1" {
		dir := t.TempDir()
		locked := filepath.Join(dir, "locked")
		if err := os.Mkdir(locked, 0o755); err != nil {
			t.Fatal(err)
		}
		lockDirFromReader(t, locked)
		if _, err := os.ReadDir(locked); err != nil {
			t.Fatalf("root is expected to read a mode-0000 directory, got %v", err)
		}
	}
	if runTestWithoutPrivileges(t) {
		return
	}

	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "inner"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	lockDirFromReader(t, locked)

	if _, err := os.Stat(locked); err != nil {
		t.Fatalf("the directory must still be statable by name: %v", err)
	}
	if _, err := os.ReadDir(locked); !errors.Is(err, syscall.EACCES) {
		t.Fatalf("expected permission-denied listing, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(locked, "inner")); !errors.Is(err, syscall.EACCES) {
		t.Fatalf("expected permission-denied stat through the directory, got %v", err)
	}
}
