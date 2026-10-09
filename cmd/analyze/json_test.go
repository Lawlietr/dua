package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPerformScanForJSONIncludesAllEntriesAndLargeFiles(t *testing.T) {
	skipIfBlockAccountingUnreliable(t)
	root := t.TempDir()

	totalFiles := maxEntries + 6
	for i := 0; i < totalFiles-1; i++ {
		path := filepath.Join(root, fmt.Sprintf("small-%02d.txt", i))
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write small file %d: %v", i, err)
		}
	}

	hugeFile := filepath.Join(root, "huge.bin")
	if err := os.WriteFile(hugeFile, make([]byte, 2<<20), 0o644); err != nil {
		t.Fatalf("write huge file: %v", err)
	}

	result := performScanForJSON(root, false)

	if result.Overview {
		t.Fatalf("expected non-overview JSON result")
	}
	if got := len(result.Entries); got != totalFiles {
		t.Fatalf("expected %d entries, got %d", totalFiles, got)
	}
	if result.TotalFiles != int64(totalFiles) {
		t.Fatalf("expected %d total files, got %d", totalFiles, result.TotalFiles)
	}
	if len(result.LargeFiles) == 0 {
		t.Fatalf("expected large_files to include the large file")
	}

	foundHuge := false
	for _, file := range result.LargeFiles {
		if file.Name == "huge.bin" && file.Path == hugeFile {
			foundHuge = true
			break
		}
	}
	if !foundHuge {
		t.Fatalf("expected huge.bin in large_files, got %#v", result.LargeFiles)
	}
}

func TestJSONEntriesFromDirEntriesIncludesMetadata(t *testing.T) {
	oldAccess := time.Now().AddDate(0, 0, -120)

	entries := jsonEntriesFromDirEntries([]dirEntry{
		{
			Name:       "old.bin",
			Path:       "/tmp/old.bin",
			Size:       42,
			IsDir:      false,
			LastAccess: oldAccess,
		},
		{
			Name:  "node_modules",
			Path:  "/tmp/project/node_modules",
			Size:  128,
			IsDir: true,
		},
	}, false, nil)

	if entries[0].LastAccess == "" {
		t.Fatalf("expected last_access to be populated")
	}
	if entries[1].Cleanable != true {
		t.Fatalf("expected node_modules entry to be marked cleanable")
	}
}

func TestJSONOverviewEntriesKeepPartialMeasurement(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, "target")
	writeFileWithSize(t, filepath.Join(target, "file"), 4096)

	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "du"), []byte("#!/bin/sh\nprintf '8\tpartial\\n'\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write du stub: %v", err)
	}
	t.Setenv("PATH", stubDir)

	entries := measureOverviewEntriesForJSON([]dirEntry{{Name: "target", Path: target, Size: -1, IsDir: true}}, nil)
	if len(entries) != 1 {
		t.Fatalf("expected one measured entry, got %#v", entries)
	}
	if entries[0].Size != 8192 {
		t.Fatalf("JSON output dropped a partial measurement: got %d, want 8192", entries[0].Size)
	}
	// Keeping the bytes is only half of it: the row has to say it kept only some
	// of them, or the same total reads as a finished measurement downstream.
	if entries[0].State != scanPartial {
		t.Fatalf("a partial du measurement lost its coverage: %#v", entries[0])
	}
}

// jsonDocumentProbe reads the published wire form back, so a rename that leaves
// the Go fields alone still shows up as a missing key here.
type jsonDocumentProbe struct {
	ScanStatus string `json:"scan_status"`
	TotalSize  int64  `json:"total_size"`
	Entries    []struct {
		Path       string `json:"path"`
		Size       int64  `json:"size"`
		ScanStatus string `json:"scan_status"`
	} `json:"entries"`
}

func decodeScanDocument(t *testing.T, result jsonOutput) jsonDocumentProbe {
	t.Helper()
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal JSON document: %v", err)
	}
	var document jsonDocumentProbe
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode JSON document: %v", err)
	}
	return document
}

func sumProbeSizes(document jsonDocumentProbe) int64 {
	var total int64
	for _, entry := range document.Entries {
		total += entry.Size
	}
	return total
}

// A clean scan must say so. Without the field, or with a document that is always
// "partial", automation cannot tell a complete total from a lower bound - which
// is the only reason to publish it at all.
func TestAnalyzeJSONCarriesCoverageOnTheDocumentAndEachEntry(t *testing.T) {
	root := t.TempDir()
	writeFileWithSize(t, filepath.Join(root, "readable"), 4096)

	document := decodeScanDocument(t, performDirectoryScanForJSON(root))
	if document.ScanStatus != "complete" {
		t.Fatalf("a clean scan must report complete coverage, got %q", document.ScanStatus)
	}
	if len(document.Entries) == 0 {
		t.Fatal("expected the scanned entry in the document")
	}
	for _, entry := range document.Entries {
		if entry.ScanStatus != "complete" {
			t.Fatalf("a measured entry must not report %q: %#v", entry.ScanStatus, entry)
		}
	}
	if got, want := document.TotalSize, sumProbeSizes(document); got != want {
		t.Fatalf("total_size %d does not match the listed entries %d", got, want)
	}
}

// An overview row that could not be measured stays in the document: it is the
// only thing that explains why total_size is a lower bound. A path that does not
// exist is used here so the case runs under root CI without a permission fixture.
func TestJSONOverviewKeepsUnmeasurableRowsAndTheirCoverage(t *testing.T) {
	root := t.TempDir()
	writeFileWithSize(t, filepath.Join(root, "readable"), 4096)
	missing := filepath.Join(root, "gone")

	result := performOverviewScanForJSONWithEntries(root, nil, []dirEntry{
		{Name: "Gone", Path: missing, IsDir: true, Size: -1},
		{Name: "Root", Path: root, IsDir: true, Size: -1},
	})
	document := decodeScanDocument(t, result)

	if document.ScanStatus != "partial" {
		t.Fatalf("an unmeasurable row must make the document partial, got %q", document.ScanStatus)
	}
	var measured bool
	for _, entry := range document.Entries {
		if entry.Path == missing {
			if entry.ScanStatus != "unavailable" || entry.Size != 0 {
				t.Fatalf("an unknown size must not be presented as a number: %#v", entry)
			}
			measured = true
		}
	}
	if !measured {
		t.Fatalf("the unmeasurable row was dropped, hiding why the total is a floor: %#v", document.Entries)
	}
	// A pending row enters as -1; leaking that into the total would understate
	// the scan below zero and cancel out real bytes.
	if got, want := document.TotalSize, sumProbeSizes(document); got != want {
		t.Fatalf("total_size %d does not match the listed entries %d", got, want)
	}
}

// The overview cache stores coverage, and a cached lower bound must still read
// as one after a reload. Deriving the state from "bytes came back with no
// error" reports a cached partial as measured - the read-side bug B-2 removed in
// loadCachedSubdirResult, which this path would reintroduce.
func TestJSONOverviewReadsBackACachedPartialAsPartial(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, "target")
	writeFileWithSize(t, filepath.Join(target, "file"), 4096)

	if err := storeOverviewMeasurement(target, 4096, true); err != nil {
		t.Fatalf("store a partial overview measurement: %v", err)
	}

	entries := measureOverviewEntriesForJSON([]dirEntry{{Name: "target", Path: target, Size: -1, IsDir: true}}, nil)
	if len(entries) != 1 {
		t.Fatalf("expected one measured entry, got %#v", entries)
	}
	if entries[0].State != scanPartial {
		t.Fatalf("a cached partial was reported as %q: %#v", entries[0].State, entries[0])
	}
}

// The same coverage on a real permission denial, which is how a user reaches it.
// Both shapes are checked because they are produced by different code paths: the
// walk marks the scan it was asked to do, the overview marks each row separately.
func TestAnalyzeJSONReportsPartialCoverageAndUnavailableSizes(t *testing.T) {
	if runTestWithoutPrivileges(t) {
		return // the re-executed child runs the body as a non-root uid
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "root")
	locked := filepath.Join(root, "locked")
	writeFileWithSize(t, filepath.Join(root, "readable"), 4096)
	writeFileWithSize(t, filepath.Join(locked, "hidden"), 1<<20)
	lockDirFromReader(t, locked)

	for _, overview := range []bool{false, true} {
		t.Run(fmt.Sprintf("overview=%t", overview), func(t *testing.T) {
			var result jsonOutput
			if overview {
				result = performOverviewScanForJSONWithEntries(root, nil, []dirEntry{
					{Name: "locked", Path: locked, IsDir: true, Size: -1},
					{Name: "root", Path: root, IsDir: true, Size: -1},
				})
			} else {
				result = performDirectoryScanForJSON(root)
			}
			document := decodeScanDocument(t, result)

			if document.ScanStatus != "partial" {
				t.Fatalf("an unreadable subtree must make the document partial, got %q", document.ScanStatus)
			}
			if document.TotalSize != sumProbeSizes(document) {
				t.Fatalf("total_size %d does not match the listed entries %d", document.TotalSize, sumProbeSizes(document))
			}
			for _, entry := range document.Entries {
				if entry.Path != locked {
					continue
				}
				// Not "unavailable" with a zero size, as the upstream form of this
				// test expects: a walk can still stat the directory itself, so its
				// own blocks are measured and only its contents are unknown. That is
				// partial, and inventing a zero would understate the tree. What the
				// contract requires is that the row survives and never claims to be
				// finished. A size that is fully unknown is pinned separately by
				// TestJSONOverviewKeepsUnmeasurableRowsAndTheirCoverage.
				if entry.ScanStatus == "complete" {
					t.Fatalf("an unreadable directory claimed complete coverage: %#v", entry)
				}
				return
			}
			t.Fatalf("JSON omitted the unreadable entry: %#v", document.Entries)
		})
	}
}

func TestJSONEntriesFromDirEntriesMarksOverviewInsights(t *testing.T) {
	entry := dirEntry{
		Name:  "Old Downloads (90d+)",
		Path:  "/tmp/test-home/Downloads",
		Size:  256,
		IsDir: true,
	}

	entries := jsonEntriesFromDirEntries([]dirEntry{entry}, true, map[string]bool{
		entry.Path: true,
	})

	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if !entries[0].Insight {
		t.Fatalf("expected entry to be marked as insight")
	}
}

func TestPerformOverviewScanForJSONSchemaWithInjectedEntries(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "payload")
	if err := os.WriteFile(payload, []byte("overview"), 0o644); err != nil {
		t.Fatalf("write overview payload: %v", err)
	}

	result := performOverviewScanForJSONWithEntries("/", nil, []dirEntry{{
		Name:  "Fixture",
		Path:  root,
		IsDir: true,
		Size:  -1,
	}})

	if result.Path != "/" || !result.Overview {
		t.Fatalf("unexpected overview identity: path=%q overview=%v", result.Path, result.Overview)
	}
	if result.Entries == nil {
		t.Fatal("overview entries must be a JSON list, not nil")
	}
	if result.TotalSize <= 0 {
		t.Fatalf("expected measured overview size, got %d", result.TotalSize)
	}
}
