package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestSavePrefPreservesOtherKeys is the whole point of the refactor: writing one
// preference must not clobber another. The previous single-string store failed
// exactly this scenario. t.Setenv redirects HOME to a temp dir so getConfigPath
// resolves inside it and the test never touches the user's real config.
func TestSavePrefPreservesOtherKeys(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	savePref("ui_theme", "dark")
	savePref("cpu_cores", "8")

	prefs := loadPrefs()
	if got := prefs["ui_theme"]; got != "dark" {
		t.Errorf("ui_theme = %q, want %q (clobbered by the second write)", got, "dark")
	}
	if got := prefs["cpu_cores"]; got != "8" {
		t.Errorf("cpu_cores = %q, want %q", got, "8")
	}
}

// TestLoadPrefsIgnoresBlanksAndComments keeps the file hand-editable.
func TestLoadPrefsIgnoresBlanksAndComments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Seed a file with a comment, a blank line, and a malformed line.
	savePref("ui_theme", "dark")
	prefs := loadPrefs()
	if len(prefs) != 1 || prefs["ui_theme"] != "dark" {
		t.Fatalf("unexpected prefs after seed: %#v", prefs)
	}
}

// Staggered process starts do not prove anything here: each writer finishes its
// whole read-modify-write before the next one is scheduled, so an unlocked
// implementation passes too. Every helper must sit on a barrier and enter
// savePref together. Without the flock, this loses keys.
func TestSavePrefConcurrentProcessesPreserveEveryKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	const writers = 16
	barrier := filepath.Join(home, "barrier")
	readyDir := filepath.Join(home, "ready")
	if err := os.MkdirAll(readyDir, 0o700); err != nil {
		t.Fatalf("create ready dir: %v", err)
	}

	commands := make([]*exec.Cmd, 0, writers)
	for i := range writers {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSavePrefProcessHelper$")
		cmd.Env = append(os.Environ(),
			"HOME="+home,
			"MOLE_PREFS_TEST_HELPER=1",
			fmt.Sprintf("MOLE_PREFS_TEST_KEY=key_%02d", i),
			"MOLE_PREFS_TEST_VALUE="+strconv.Itoa(i),
			"MOLE_PREFS_TEST_BARRIER="+barrier,
			fmt.Sprintf("MOLE_PREFS_TEST_READY=%s", filepath.Join(readyDir, strconv.Itoa(i))),
		)
		if err := cmd.Start(); err != nil {
			t.Fatalf("start writer %d: %v", i, err)
		}
		commands = append(commands, cmd)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		entries, err := os.ReadDir(readyDir)
		if err != nil {
			t.Fatalf("read ready dir: %v", err)
		}
		if len(entries) == writers {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d writers reached the barrier", len(entries), writers)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if err := os.WriteFile(barrier, []byte("go"), 0o600); err != nil {
		t.Fatalf("release barrier: %v", err)
	}

	for i, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("writer %d failed: %v", i, err)
		}
	}

	prefs := loadPrefs()
	for i := range writers {
		key := fmt.Sprintf("key_%02d", i)
		if got := prefs[key]; got != strconv.Itoa(i) {
			t.Errorf("%s = %q, want %q", key, got, strconv.Itoa(i))
		}
	}
}

func TestSavePrefProcessHelper(t *testing.T) {
	if os.Getenv("MOLE_PREFS_TEST_HELPER") != "1" {
		return
	}
	if barrier := os.Getenv("MOLE_PREFS_TEST_BARRIER"); barrier != "" {
		if ready := os.Getenv("MOLE_PREFS_TEST_READY"); ready != "" {
			if err := os.WriteFile(ready, []byte("1"), 0o600); err != nil {
				t.Fatalf("signal ready: %v", err)
			}
		}
		deadline := time.Now().Add(30 * time.Second)
		for {
			if _, err := os.Stat(barrier); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("barrier %s never opened", barrier)
			}
			time.Sleep(time.Millisecond)
		}
	}
	savePref(os.Getenv("MOLE_PREFS_TEST_KEY"), os.Getenv("MOLE_PREFS_TEST_VALUE"))
}
