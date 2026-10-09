package main

import (
	"context"
	"errors"
	"io/fs"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// scanState describes measurement coverage within dua's scan filters, not an
// atomic filesystem snapshot. The zero value represents a complete measurement,
// so a cache entry or scan result written by an older binary (which never had a
// state) still reads as complete.
type scanState uint8

const (
	scanComplete scanState = iota
	scanPartial
	scanUnavailable
)

func (s scanState) String() string {
	switch s {
	case scanPartial:
		return "partial"
	case scanUnavailable:
		return "unavailable"
	default:
		return "complete"
	}
}

// measurementState preserves the distinction between a useful partial size and
// a failed probe that measured nothing. Callers must retain the returned bytes:
// a partial measurement is only partial because some bytes were measured, and
// reporting it as complete would let the TUI and the cache present an
// under-counted directory as if the subtree were fully known.
func measurementState(size int64, err error) scanState {
	if err == nil {
		return scanComplete
	}
	if size > 0 {
		return scanPartial
	}
	return scanUnavailable
}

// isPermissionFailure reports a denial that will repeat on every scan until the
// permissions change (a chmod 0 directory, or a path this process may not
// enter). A partial result caused only by those is as current as a complete
// one, so re-measuring it on every visit only burns the same answer again.
// Anything else - a timeout, a cancellation, a vanished file, an I/O error -
// may clear on the next attempt, so its missing bytes are worth another try.
func isPermissionFailure(err error) bool {
	return err != nil && (errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM))
}

func isTransientFailure(err error) bool {
	return err != nil && !isPermissionFailure(err)
}

// partialFromPermissionErrors reports whether any of the errors that made a
// measurement incomplete were permission denials, i.e. whether some of the
// missing bytes are worth keeping as a cached lower bound. A tree with a
// chmod-0 subdirectory measures the same way on every visit, so its partial
// total is the current answer; a tree that lost bytes to a timeout has a
// different answer next time, and caching the old one would hide that.
func partialFromPermissionErrors(errs ...error) bool {
	for _, err := range errs {
		if isPermissionFailure(err) {
			return true
		}
	}
	return false
}

type dirEntry struct {
	State      scanState
	Name       string
	Path       string
	Size       int64
	IsDir      bool
	LastAccess time.Time
}

type fileEntry struct {
	Name string
	Path string
	Size int64
}

type scanResult struct {
	State      scanState
	Entries    []dirEntry
	LargeFiles []fileEntry
	TotalSize  int64
	TotalFiles int64
	// dedupedHardlink is true when a hardlinked file in this subtree was
	// counted as zero because another link was seen earlier in the same
	// scan. Such a result is scan-order dependent and must not be written
	// to the on-disk cache. In-memory only; never serialized to cacheEntry.
	dedupedHardlink bool
	// transientFailure is true when part of the coverage was lost to a failure
	// that may not recur (see isPermissionFailure). A row like that is always
	// re-measured on the next visit (see shouldRefreshExistingResult): its missing
	// bytes are an arbitrary amount frozen at an arbitrary moment. In-memory only.
	transientFailure bool
	// partial is true when at least one permission denial contributed to the
	// missing bytes, so the bytes that *were* measured are the same ones the next
	// scan would find, and the total may be cached as a lower bound. A tree can
	// set both flags - it lost bytes to a denial and to a timeout - and stays
	// cacheable: the floor is still honest, and pressing R retries the temporary
	// half. `persistable` is what decides reuse. In-memory only.
	partial bool
}

// persistable reports whether a result may be reused without rescanning. It
// answers "is this the best answer a rescan could produce?", not "is this
// complete?": a partial whose gaps include a permission denial is the former,
// because those bytes stay unreadable until access changes, and the coverage
// marker keeps it honest in the TUI. A partial whose gaps are all temporary
// (a timeout, a cancellation) is not: rescanning can recover those bytes, and
// reusing the frozen lower bound would report them as measured.
func (r scanResult) persistable() bool {
	return r.State == scanComplete || (r.State == scanPartial && r.partial)
}

// overviewMeasurementStorable keeps an overview measurement whose only failures
// were permission denials, like a clean one: rescanning cannot recover those
// bytes until access changes. Transient ones stay unstored.
func overviewMeasurementStorable(size int64, err error) bool {
	return size > 0 && (err == nil || isPermissionFailure(err))
}

type cacheEntry struct {
	// State records the coverage the totals were measured under, so a cached
	// partial keeps reading as partial after a reload. Entries written by an
	// older binary decode this as scanComplete, which matches what they were.
	State        scanState
	Entries      []dirEntry
	LargeFiles   []fileEntry
	TotalSize    int64
	TotalFiles   int64
	ModTime      time.Time
	ScanTime     time.Time
	NeedsRefresh bool
	// SchemaVersion guards against reusing cache written by an older binary
	// with different sizing semantics. Entries not at cacheSchemaVersion are
	// rejected on load. Old caches decode this as 0.
	SchemaVersion int
}

type historyEntry struct {
	State         scanState
	Path          string
	Entries       []dirEntry
	LargeFiles    []fileEntry
	TotalSize     int64
	TotalFiles    int64
	Selected      int
	EntryOffset   int
	LargeSelected int
	LargeOffset   int
	NeedsRefresh  bool
	IsOverview    bool
}

type scanResultMsg struct {
	path   string
	result scanResult
	err    error
	stale  bool
}

type liveScanStartMsg struct {
	state scanState
	// transient says whether the listing's own gaps may clear on a retry, so
	// the model can decide later whether this view is worth re-scanning.
	transient bool
	// partial says the listing's bytes are a reusable floor: at least one gap is
	// a permission denial, so it will still be there next time. Only used to
	// carry coverage into the scan that follows, never rendered on its own.
	partial       bool
	id            int64
	path          string
	entries       []dirEntry
	totalSize     int64
	totalFiles    int64
	largeFiles    []fileEntry
	scanningPaths []string
	events        <-chan liveScanEventMsg
	cancel        context.CancelFunc
	err           error
}

type liveScanEventKind int

const (
	liveScanChildProgress liveScanEventKind = iota + 1
	liveScanChildDone
	liveScanComplete
	liveScanFailed
)

type liveScanEventMsg struct {
	id     int64
	path   string
	kind   liveScanEventKind
	entry  dirEntry
	result scanResult
	err    error
}

type liveSortMode int

const (
	liveSortContinuous liveSortMode = iota
	liveSortFreezeOnMove
)

type overviewSizeMsg struct {
	Path  string
	Index int
	Size  int64
	Err   error
}

type initializeMsg struct{}

type tickMsg time.Time

type model struct {
	scanState           scanState
	scanTransient       bool // scanState != scanComplete because of a transient failure
	path                string
	history             []historyEntry
	entries             []dirEntry
	largeFiles          []fileEntry
	selected            int
	offset              int
	status              string
	totalSize           int64
	scanning            bool
	spinner             int
	tickRunning         bool // one tickCmd loop is already re-arming itself
	filesScanned        *int64
	dirsScanned         *int64
	bytesScanned        *int64
	currentPath         *atomic.Value
	showLargeFiles      bool
	isOverview          bool
	cache               map[string]historyEntry
	largeSelected       int
	largeOffset         int
	overviewSizeCache   map[string]int64
	overviewScanning    bool
	overviewScanningSet map[string]bool // Track which paths are currently being scanned
	width               int             // Terminal width
	height              int             // Terminal height
	totalFiles          int64           // Total files found in current/last scan
	lastTotalFiles      int64           // Total files from previous scan (for progress bar)
	diskFree            int64           // Free disk space for the analyzed volume
	viewNeedsRefresh    bool
	// Top-files (T) view incremental filter. largeFilesAll is the full,
	// size-ranked list; largeFiles is the view actually rendered and acted on,
	// which equals largeFilesAll when no filter is set and the matching subset
	// otherwise. largeFiltering is true only while the user is typing a query.
	largeFilesAll  []fileEntry
	largeFilter    string
	largeFiltering bool
	// Directory (drill-down) view incremental filter, mirroring the Top-files
	// one. entriesAll is the full non-empty entry list; entries is the rendered,
	// possibly filtered view. Disabled in overview mode.
	entriesAll          []dirEntry
	entryFilter         string
	entryFiltering      bool
	liveScanID          int64
	liveScanCancel      context.CancelFunc
	liveScanEvents      <-chan liveScanEventMsg
	liveScanningPaths   map[string]bool
	autoSortLiveEntries bool
	liveSortMode        liveSortMode
}

func (m model) inOverviewMode() bool {
	return m.isOverview && m.path == "/"
}

// entryScanState reports the coverage of a rendered list: a row that is still
// pending, or that could only be measured partially, makes the total a lower
// bound rather than a complete measurement.
func entryScanState(entries []dirEntry) scanState {
	for _, entry := range entries {
		if entry.Size < 0 || entry.State != scanComplete {
			return scanPartial
		}
	}
	return scanComplete
}

func (m *model) hydrateOverviewEntries() {
	m.entries = createOverviewEntries()
	if m.overviewSizeCache == nil {
		m.overviewSizeCache = make(map[string]int64)
	}
	for i := range m.entries {
		if size, ok := m.overviewSizeCache[m.entries[i].Path]; ok {
			m.entries[i].Size = size
			continue
		}
		if size, state, err := loadOverviewCachedMeasurement(m.entries[i].Path); err == nil {
			m.entries[i].Size = size
			m.entries[i].State = state
			// The in-memory map holds sizes only, so a partial one must not
			// enter it: it would come back on the next paint without its marker.
			if state == scanComplete {
				m.overviewSizeCache[m.entries[i].Path] = size
			}
		}
	}
	m.totalSize = sumKnownEntrySizes(m.entries)
	m.scanState = entryScanState(m.entries)
	m.scanTransient = false
}

func (m *model) sortOverviewEntriesBySize() {
	// Stable sort by size.
	sort.SliceStable(m.entries, func(i, j int) bool {
		return m.entries[i].Size > m.entries[j].Size
	})
}

func (m *model) getScanProgress() (files, dirs, bytes int64) {
	if m.filesScanned != nil {
		files = atomic.LoadInt64(m.filesScanned)
	}
	if m.dirsScanned != nil {
		dirs = atomic.LoadInt64(m.dirsScanned)
	}
	if m.bytesScanned != nil {
		bytes = atomic.LoadInt64(m.bytesScanned)
	}
	return
}

func (m *model) clampEntrySelection() {
	if len(m.entries) == 0 {
		m.selected = 0
		m.offset = 0
		return
	}
	if m.selected >= len(m.entries) {
		m.selected = len(m.entries) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	viewport := calculateViewport(m.height, false)
	maxOffset := max(len(m.entries)-viewport, 0)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+viewport {
		m.offset = m.selected - viewport + 1
	}
}

func (m *model) clampLargeSelection() {
	if len(m.largeFiles) == 0 {
		m.largeSelected = 0
		m.largeOffset = 0
		return
	}
	if m.largeSelected >= len(m.largeFiles) {
		m.largeSelected = len(m.largeFiles) - 1
	}
	if m.largeSelected < 0 {
		m.largeSelected = 0
	}
	viewport := calculateViewport(m.height, true)
	maxOffset := max(len(m.largeFiles)-viewport, 0)
	if m.largeOffset > maxOffset {
		m.largeOffset = maxOffset
	}
	if m.largeSelected < m.largeOffset {
		m.largeOffset = m.largeSelected
	}
	if m.largeSelected >= m.largeOffset+viewport {
		m.largeOffset = m.largeSelected - viewport + 1
	}
}

func fileEntryName(f fileEntry) string { return f.Name }
func fileEntryPath(f fileEntry) string { return f.Path }
func dirEntryName(e dirEntry) string   { return e.Name }
func dirEntryPath(e dirEntry) string   { return e.Path }

// filterMatches reports whether an item with the given name and path matches a
// case-insensitive substring query. Single source of truth for both the
// Top-files and directory filters so their match semantics cannot drift.
func filterMatches(name, path, query string) bool {
	needle := strings.ToLower(query)
	return strings.Contains(strings.ToLower(name), needle) ||
		strings.Contains(strings.ToLower(displayPath(path)), needle)
}

// filterByQuery returns the items matching query, or the original slice
// unchanged when the query is empty. nameOf/pathOf project the fields matched.
func filterByQuery[T any](all []T, query string, nameOf, pathOf func(T) string) []T {
	if query == "" {
		return all
	}
	out := make([]T, 0, len(all))
	for _, item := range all {
		if filterMatches(nameOf(item), pathOf(item), query) {
			out = append(out, item)
		}
	}
	return out
}

// applyLargeFilter rebuilds the rendered Top-files view from largeFilesAll
// using the current query. An empty query restores the full list.
func (m *model) applyLargeFilter() {
	m.largeFiles = filterByQuery(m.largeFilesAll, m.largeFilter, fileEntryName, fileEntryPath)
	m.clampLargeSelection()
}

// resetLargeFilter clears any active Top-files filter and restores the full
// list. Callers that leave the Top-files view use this so the next visit and
// the per-path navigation state start clean.
func (m *model) resetLargeFilter() {
	m.largeFilter = ""
	m.largeFiltering = false
	if m.largeFilesAll != nil {
		m.largeFiles = m.largeFilesAll
	}
}

// applyEntryFilter rebuilds the rendered directory view from entriesAll using
// the current query. The directory view is the drill-down list (m.entries) in
// non-overview mode.
func (m *model) applyEntryFilter() {
	m.entries = filterByQuery(m.entriesAll, m.entryFilter, dirEntryName, dirEntryPath)
	m.clampEntrySelection()
}

// resetEntryFilter clears any active directory filter and restores the full
// entry list.
func (m *model) resetEntryFilter() {
	m.entryFilter = ""
	m.entryFiltering = false
	if m.entriesAll != nil {
		m.entries = m.entriesAll
	}
}
