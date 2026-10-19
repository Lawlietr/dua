package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// Ported from upstream 6307d735's cmd/analyze/directory_browsing_test.go.
// Three deviations, all deliberate:
//   - upstream gates the file on //go:build darwin. dua's primary platform is
//     Linux and its CI runs the test suite on Linux only, so an OS gate would
//     mean these never run anywhere in this repo. The gate is dropped.
//   - the deletion case is dropped, along with the delete-confirmation steps in
//     the navigation test: dua removed the delete surface at the fork. What
//     remains of that test still pins the point - a warmed, bounded child listing
//     must expand to the complete listing on entry, and the last child must be
//     reachable through the filter.
//   - upstream's m.cancelBackgroundCacheWrites has no dua counterpart (dua's
//     finishLiveScan hands the write to a plain goroutine), so each case waits
//     for the listing to reach disk instead. The temp HOME is removed at cleanup,
//     which would otherwise race the writer.

func TestDirectoryRefreshKeepsSelectedPathVisible(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(fmt.Sprintf("live=%t", live), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := filepath.Join(home, "root")
			if err := os.Mkdir(root, 0o755); err != nil {
				t.Fatal(err)
			}
			const count = 80
			for i := range count {
				child := filepath.Join(root, fmt.Sprintf("entry-%02d", i))
				if err := os.Mkdir(child, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(child, "data"), []byte("data"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			m := newModel(root, false)
			m.height, m.width = 24, 120
			updated, _ := m.Update(runScanResultCmd(t, m.scanFreshCmd(root)))
			m = updated.(model)

			// Walk to the last row, then grow THAT child so the next measurement
			// moves it to the top of the sort. Upstream hardcodes entry-79, which
			// only holds because macOS readdir returns creation order; on Linux the
			// order is hash based, so the path is read back from the model instead.
			// The invariant is the same: the cursor follows its path across a
			// re-sort, it does not stay on the row index.
			for range count - 1 {
				m, _ = filterKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
			}
			target := m.selectedEntryPath()
			if target == "" {
				t.Fatal("no row under the cursor after walking the listing")
			}
			if err := os.WriteFile(filepath.Join(target, "data"), []byte(strings.Repeat("x", 8192)), 0o644); err != nil {
				t.Fatal(err)
			}

			if live {
				start := m.scanFreshCmd(root)().(liveScanStartMsg)
				defer cancelAndDrainLiveScan(start)
				updated, _ = m.Update(start)
				m = updated.(model)
				result := drainLiveScanToResultMsg(t, start)
				if result.err != nil {
					t.Fatal(result.err)
				}
				updated, _ = m.Update(liveScanEventMsg{
					id: start.id, path: root, kind: liveScanComplete, result: result.result,
				})
				waitForTestCondition(t, "completed listing to reach disk cache", func() bool {
					_, err := loadCacheFromDisk(root)
					return err == nil
				})
			} else {
				updated, _ = m.Update(runScanResultCmd(t, m.scanFreshCmd(root)))
			}
			m = updated.(model)
			name := filepath.Base(target)
			for line := range strings.SplitSeq(ansi.Strip(m.View()), "\n") {
				if strings.Contains(line, "\u25b6") && strings.Contains(line, name) {
					return
				}
			}
			t.Fatalf("refresh hid or changed the selected path %s: %s", name, ansi.Strip(m.View()))
		})
	}
}

func TestDirectoryNavigationExpandsWarmedSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "root")
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range subdirCacheMinFiles {
		if err := os.WriteFile(filepath.Join(child, fmt.Sprintf("entry-%03d", i)), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(root, false)
	m.height, m.width = 24, 120
	start := m.scanFreshCmd(root)().(liveScanStartMsg)
	defer cancelAndDrainLiveScan(start)
	updated, _ := m.Update(start)
	m = updated.(model)
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for m.scanning {
		select {
		case event, ok := <-start.events:
			if !ok {
				t.Fatal("live scan closed without completing")
			}
			updated, _ = m.Update(event)
			m = updated.(model)
		case <-deadline.C:
			t.Fatal("live scan did not complete")
		}
	}
	var cmd tea.Cmd
	m, cmd = filterKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.path != child || !m.scanning || len(m.entries) > maxEntries {
		t.Fatalf("expected the warmed child summary to refresh on entry: %s, %d rows", m.path, len(m.entries))
	}
	updated, _ = m.Update(runScanResultCmd(t, cmd))
	m = updated.(model)
	name := fmt.Sprintf("entry-%03d", subdirCacheMinFiles-1)
	m, _ = filterRune(t, m, '/')
	m = filterType(t, m, name)
	// Upstream reaches for the delete confirmation here to show the last entry is
	// addressable; dua has no delete surface, so the filter result is the
	// assertion: the last child of a listing that was warmed as a bounded summary
	// is still a row.
	if len(m.entries) != 1 || m.entries[0].Path != filepath.Join(child, name) {
		t.Fatalf("filter lost the late entry of a warmed listing: %+v", m.entries)
	}
	m, _ = filterKey(t, m, tea.KeyMsg{Type: tea.KeyEsc}) // Clear the filter.
	m, _ = filterRune(t, m, 'h')
	if m.path != root || len(m.entries) != 1 {
		t.Fatalf("back navigation lost the parent: %s", m.path)
	}
	m, cmd = filterKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.path != child || len(m.entries) != subdirCacheMinFiles || m.scanning || cmd != nil {
		t.Fatalf("in-memory revisit lost the complete child listing: %s, %d rows", m.path, len(m.entries))
	}
	waitForTestCondition(t, "expanded child listing to reach disk cache", func() bool {
		cached, err := loadCacheFromDisk(child)
		return err == nil && cached.TotalFiles == int64(subdirCacheMinFiles)
	})
}

func TestDirectoryRowsStayAlignedBeyondTwoDigitIndexes(t *testing.T) {
	m := model{path: "/fixture", width: 120, height: 8, selected: -1, totalSize: 10000}
	for i := range 10000 {
		name := fmt.Sprintf("entry-%05d", i)
		m.entries = append(m.entries, dirEntry{Name: name, Path: "/fixture/" + name, Size: 1})
	}
	for _, withHints := range []bool{false, true} {
		if withHints {
			for i := range m.entries {
				m.entries[i].LastAccess = time.Now().Add(-120 * 24 * time.Hour)
			}
		}
		for _, width := range []int{80, 120, 140} {
			m.width = width
			short := m
			short.entries = m.entries[:2]
			short.offset = 0
			rowWidth := ansi.StringWidth(rowContaining(ansi.Strip(short.View()), short.entries[0].Name))
			for _, offset := range []int{98, 998, 9998} {
				m.offset = offset
				view := ansi.Strip(m.View())
				first := rowContaining(view, m.entries[offset].Name)
				second := rowContaining(view, m.entries[offset+1].Name)
				firstPrefix, _, firstOK := strings.Cut(first, "  |  ")
				secondPrefix, _, secondOK := strings.Cut(second, "  |  ")
				if !firstOK || !secondOK || ansi.StringWidth(firstPrefix) != ansi.StringWidth(secondPrefix) {
					t.Fatalf("row numbers shifted the columns at %d:\n%s\n%s", offset+1, first, second)
				}
				if ansi.StringWidth(first) > rowWidth || ansi.StringWidth(second) > rowWidth {
					t.Fatalf("row numbers widened the rows at %d:\n%s\n%s", offset+1, first, second)
				}
			}
		}
	}
}

func BenchmarkDirectoryBrowsing(b *testing.B) {
	m := model{path: "/fixture", width: 120, height: 24}
	for i := range 10000 {
		m.entriesAll = append(m.entriesAll, dirEntry{
			Name: fmt.Sprintf("entry-%05d", i),
			Path: fmt.Sprintf("/fixture/entry-%05d", i),
			Size: 1,
		})
	}
	m.entries = m.entriesAll
	m.totalSize = int64(len(m.entries))
	b.Run("render", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = m.View()
		}
	})
	b.Run("filter_last", func(b *testing.B) {
		m.entryFilter = "entry-09999"
		b.ReportAllocs()
		for b.Loop() {
			m.applyEntryFilter()
			_ = m.View()
		}
	})
}
