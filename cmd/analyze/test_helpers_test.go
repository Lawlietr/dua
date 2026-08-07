package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// skipIfRoot guards tests whose behavior depends on permission-denied
// semantics; the root user can read directories regardless of mode bits.
func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission-denied semantics unavailable")
	}
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
