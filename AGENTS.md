# dua Agent Guide

This file is the shared source of truth for any AI agent working on this repo (Claude Code, Codex, etc.). `CLAUDE.md` is a symlink to this file. Put machine-specific or personal overrides in `AGENTS.local.md` / `CLAUDE.local.md`; both are gitignored.

## Project

dua is a Linux disk analysis and system status toolkit, forked from the macOS Mole CLI's `analyze` and `status` commands and re-scoped to an independent project. It is written in Go (two small binaries) behind a thin bash router. It is **read-only**: it inspects and reports, never deletes or modifies user data.

## Product Direction

dua is a terminal-first Linux maintenance inspection toolkit. Its core job is to help power users see where disk space went and how healthy the machine is, from a CLI, script, or compact TUI.

### What dua Should Do

- Keep `analyze` a visual disk explorer: an overview of system roots plus hidden-space insights, drill-down navigation, a Top-files view, live scan feedback, caching, and `--json` output for automation.
- Keep `status` a compact read-only health dashboard (CPU, memory, disk, thermal, hardware, top processes) plus stable JSON/NDJSON output (`--json`, `--watch`).
- Keep command UX dense and terminal-native: short labels, stable alignment, predictable shortcuts, one-screen summaries, then optional drill-down.
- Keep both binaries safe by construction: no deletion path exists anywhere in the codebase.

### What dua Should Not Do

- Do not add cleanup, uninstall, Trash routing, or any modification surface. The project deliberately deleted the macOS Mole cleanup commands; do not re-introduce them.
- Do not add background agents, persistent monitoring, notifications, schedulers, or menu bar behavior unless explicitly requested and justified as CLI scope.
- Do not turn `status` into a noisy dashboard. Extra rows, live alerts, and tuning controls need a common user action, not just an available metric.
- Do not add prompts, preferences, or output modes to solve every edge case. Prefer quieter defaults, preview/read-only guidance, or declining unsupported operations.
- Do not broaden cache/large-file matching from exact evidence into vendor-wide or generic-name globs. Hint markers must point at real, rebuildable paths.

### Product Decision Filter

Before accepting a new feature, answer these questions in the PR, issue, or review notes when the fit is not obvious:

1. Does it clearly belong to analyze, status, update, or the router?
2. Is it read-only, previewable, testable, and explainable in one terminal screen?
3. Can the user verify what dua reports before trusting it?
4. Is the target data locally rebuildable, disposable, or backed by exact path evidence?
5. Would this be better as documentation, a warning, or an explicit "not supported" answer?

If the answer is no or unclear, decline the feature, narrow it, or park it until the product value beats the added surface area.

## Repository Map

- `AGENTS.md` is the cross-agent source of truth. `CLAUDE.md` must remain a symlink to it so Claude and Codex receive the same project contract.
- `dua` - the CLI entrypoint. It is a **router only**: it parses args and dispatches to the Go binaries (or runs shell logic for `update`). Business logic does not belong here.
- `cmd/analyze/` - the disk-analysis Go binary (TUI + `--json`). `main.go` is bootstrap only; `model.go` holds types and accessor methods; `update.go` holds the Bubble Tea Update chain; `scanner.go` owns traversal and concurrency budgets.
- `cmd/status/` - the system status Go binary (TUI, `--json`, `--watch`). `view.go` renders only; collection and JSON/NDJSON contracts live in the other `cmd/status/` files.
- `internal/units/` - shared byte formatting.
- `bin/` - build output directory (gitignored); `dua-analyze` and `dua-status` live here after `make build`.
- `Makefile` - Linux-only build/test entrypoints. `CGO_ENABLED=0`, `-trimpath`, `-s -w`.
- `go.mod` - pins `toolchain go1.25.10`. Both workflows read `go-version-file: go.mod`, so a toolchain bump here updates CI and release builds together.
- `.github/workflows/` - CI and release pipelines. `test.yml` runs vet + tests on Linux-DEV/main pushes and pull requests. `release.yml` is tag-driven (see Versioning). `upload-darwin.yml` is a one-off dispatch pipeline that cross-compiles darwin/arm64, smoke-tests it on a macos-14 runner, and attaches the tarball to an existing release; it exists until a regular release folds darwin into the main matrix (see Next Steps) and should be deleted afterwards.
- `.github/dependabot.yml` - weekly dependency-update PRs for `gomod` + `github-actions`; minor/patch updates are grouped to reduce noise.
- `README.md` (English) and `README.zh-TW.md` (Traditional Chinese) are the user-facing docs; keep them in sync when behaviour or install instructions change — the two are translations of the same document, so headings and section order must match.

Platform-specific files in `cmd/analyze/` carry explicit build tags (`atime_linux.go`, `atime_darwin.go`). Linux is the primary supported runtime; macOS arm64 ships as secondary, best-effort support (tarballs attached to releases since v0.2.0). The darwin build paths (`atime_darwin.go`, Spotlight, `open`) are real code paths exercised by CI smoke tests, not dead weight.

## Commands

```bash
make build          # builds bin/dua-analyze and bin/dua-status
make check          # go vet ./... + go test ./...
go test ./...       # full Go test suite
go test ./cmd/analyze
go test ./cmd/status
./dua status --json
./dua analyze --json /some/path
./dua analyze        # overview TUI (no PATH)
./dua status         # TUI dashboard
./dua update         # Check GitHub and update to the latest release
gofmt -l cmd/ internal/
```

Public docs and examples should prefer the installed `dua` command. Use `./dua` in this repository when verifying source-tree behavior before building. `analyze` and `analyse` are both accepted command spellings.

## Working Rules

- **dua is read-only.** Before adding any code path that deletes, truncates, moves, or modifies files, stop and reconsider: the deletion surface was deliberately removed during the Linux port.
- Keep analyze scan behavior conservative and bounded: every scan has a timeout budget and inner-loop checkpoints; a timed-out producer must not feed partial output into downstream consumers.
- `du` is invoked with `-skPx` plus exclude patterns. The exclude flag is platform-aware (`duExcludeArgs`): GNU du (Linux) uses `--exclude=NAME`, BSD du (macOS) uses `-I NAME`. Verify against the target platform's `du` before changing flags.
- The file opener (O / P / F keys) is platform-aware (`openCommandName`): `open` on macOS, `xdg-open` elsewhere. Validate paths (`validatePath`) before passing them to any external command.
- Spotlight (`mdfind`) integration is macOS-only and guarded by `runtime.GOOS == "darwin"`. The walk-based collector is the Linux large-file source.
- Overview roots and insight paths are platform-aware (`systemOverviewRoots`, `createInsightEntries`). Linux uses `/usr`, `/opt`, `/var`, `/home` and XDG cache paths (`~/.cache`, `~/.npm`, `~/.local/share/Trash`, ...).
- Keep shell code minimal; the `dua` router is the only shell surface. Format with `./scripts/check.sh --format` only if scripts exist; there is none today.
- Keep the module path as `github.com/Lawlietr/dua`, matching the GitHub repo. The supported install paths are the GitHub Release tarballs (curl) and `make build`. `go install github.com/Lawlietr/dua/cmd/analyze@<tag>` works but installs binaries named `analyze`/`status` (not `dua-analyze`/`dua-status`), so the curl tarball remains the canonical install for the `dua` router.
- Do not add AI attribution trailers to commits.

### Testing Notes

- The test suite runs as the invoking user. Tests that need permission-denied semantics (`chmod 000`) call `runTestWithoutPrivileges` (`cmd/analyze/test_helpers_test.go`), which re-runs the package's test binary inside a private user namespace (`unshare -U`) so the fixture applies even when the suite runs as root; `lockDirFromReader` sets the mode and restores it. Do not add root-skipping guards back — on root CI they silently drop the whole permission-denied path. `TestPermissionFixtureDropsReadAccess` pins the mechanism itself. Tests that depend on `st_blocks` size accounting call `skipIfBlockAccountingUnreliable` (some containerized filesystems, e.g. ZFS, report one block per file and collapse sparse/actual-usage accounting).
- macOS-only behavior tests (Spotlight invocation, OrbStack insight, Trash paths) are gated on `runtime.GOOS == "darwin"` and skip on Linux.
- Never pipe a test, check, or CI run into `tail` or `head`. The pipeline reports the pager's exit code, so a red run reads green. Let it print in full, or capture to a file and check the status separately.
- Prefer targeted `go test ./cmd/analyze` / `go test ./cmd/status` during development; run `go test ./...` before committing.

## Versioning

- Releases are tag-driven: pushing a `v*` tag triggers `.github/workflows/release.yml`, which builds `dua-analyze`/`dua-status` for linux/amd64 and linux/arm64, packages flat tarballs (`dua`, `dua-analyze`, `dua-status`) with sha256 checksums, and creates a GitHub Release. `releases/latest` serves the newest tag. Since v0.2.0, releases also carry a `dua-darwin-arm64.tar.gz` attached by the one-off `upload-darwin.yml` pipeline; this is transitional until darwin joins the regular matrix.
- Commits to `main`/`Linux-DEV` do NOT rebuild the distributed binaries: `releases/latest` only changes when a `v*` tag is pushed. A security fix must ship as a patch release (e.g. `v0.1.2`) so users downloading via curl get the fixed binaries.
- Version numbers: bump the patch (`v0.x.y`) for fixes, docs, and project-level chores; bump the minor (`v0.y.0`) only for new features or behavior changes; reserve `v1.0.0` for a stability promise. Do not bump the minor for chores.

## Hotspot Ownership

These files are intentionally large. Do not start by splitting them. Keep edits narrow and run the listed tests when touching each area.

- `cmd/analyze/update.go` owns the Bubble Tea `Update` chain and message handlers (Init, scanCmd, updateKey, goBack, switchToOverviewMode, enterSelectedDir). This is the largest file in `cmd/analyze/` and the natural landing spot for new key bindings, message types, or navigation behavior. Run `go test ./cmd/analyze`.
- `cmd/analyze/scanner.go` owns disk traversal, Spotlight integration (darwin-only), cancellation, and all scan concurrency budgets. Treat its semaphores as independent resource limits and measure before changing them. Run `go test ./cmd/analyze`.
- `cmd/analyze/cache.go` owns analyze cache schema, expiry, load/save, invalidation, and cacheability decisions. Cacheability is decided by `scanResult.persistable()`, which asks **why** coverage was lost, not how much: a permission-denial gap leaves a reusable lower bound, a timeout or cancellation does not — and that is a separate question from `historyEntry.NeedsRefresh`, so do not merge the two. Computation changes must invalidate stale persisted data in the same change. Run `go test ./cmd/analyze`.
- `cmd/analyze/analyze_test.go` and `cmd/status/view_test.go` are test hotspots. Add new cases near related behavior; split later only when touching many adjacent cases. Run `go test ./cmd/...`.
- `cmd/status/view.go` owns status rendering only; collection and JSON/NDJSON contracts live elsewhere in `cmd/status/`. Keep narrow-terminal layout and automation output independent. Run `go test ./cmd/status`.

## Context-Mode Knowledge Base

The four hotspot files below plus the user docs are pre-indexed into `ctx-index`. Use `ctx_search` to pull precise snippets instead of re-reading these large files:

- `cmd/analyze/scanner.go` — disk traversal, Spotlight (darwin-only), cancellation, concurrency budgets
- `cmd/analyze/update.go` — Bubble Tea `Update` chain and message handlers
- `cmd/analyze/cache.go` — cache schema, expiry, load/save, invalidation
- `cmd/status/view.go` — status rendering only
- `README.md` / `README.zh-TW.md` — user-facing usage docs (keep the two in sync)

This section is only the pointer; the indexed copy lives in the shared knowledge base. If the base is purged or the repo is cloned to a fresh environment, re-index with `ctx_index` on these paths.

## Architecture Note

dua currently ships as **three files**: a bash router (`dua`) plus two Go binaries (`dua-analyze`, `dua-status`). The router dispatches to the appropriate binary based on the subcommand. This is intentional and documented in the Repository Map above.

A single-binary design (one `dua` executable with subcommands) would be cleaner from a user's perspective, but would require merging `cmd/analyze` and `cmd/status` into a single `main.go` with subcommand handling (e.g. via `flag` or a CLI library). This is not currently planned but worth keeping in mind if the project grows.

## Fork Maintenance

### Upstream Repository

- **Upstream**: `https://github.com/tw93/Mole`
- **Upstream remote**: `upstream` (configured as `https://github.com/tw93/Mole.git`)
- **Origin remote**: `origin` (configured as `git@github.com:Lawlietr/dua.git`)

```bash
# Add upstream remote (if not already configured)
git remote add upstream https://github.com/tw93/Mole.git

# Fetch latest from upstream
git fetch upstream

# Check what's new upstream (analyze/status only)
git log --oneline upstream/main -- cmd/analyze/ cmd/status/

# Check merge base (split point)
git merge-base HEAD upstream/main
```

### Fork Strategy: Cherry-pick, Not Merge

two is an **independent fork**, not a tracked fork. Mole continues to develop macOS-specific features (cleanup, uninstall, optimize, purge, trash routes, battery health, etc.) that dua deliberately does not need. Direct merge of `upstream/main` will produce **~70+ conflicts** including:

- **~60 modify/delete conflicts**: Mole added/modified files that dua deleted (cleanup scripts, uninstall scripts, tests, Mole-specific shell scripts)
- **~13 content conflicts**: Both repos modified the same files differently

**Do NOT** do `git merge upstream/main`. Use cherry-pick instead.

### Cherry-pick Workflow

```bash
# 1. Fetch upstream
git fetch upstream

# 2. List upstream commits affecting analyze/status
git log --oneline upstream/main -- cmd/analyze/ cmd/status/

# 3. Filter out Mole-specific commits (cleanup, uninstall, trash, delete, battery, etc.)
#    Keep only: bug fixes, performance improvements, UI fixes, Linux-relevant changes

# 4. Cost-probe a candidate batch before scheduling it: cherry-pick each commit onto the
#    branch tip in a throwaway worktree, count conflicts, then abort
git worktree add --detach /tmp/probe <branch-tip>
cd /tmp/probe && git cherry-pick -x <hash>   # then: git diff --name-only --diff-filter=U
git cherry-pick --abort && git worktree remove --force /tmp/probe   # leave nothing behind
# Do NOT cost-probe with `git apply --check -3`: its output mixes a failed plain apply with
# the 3-way fallback, so it produces numbers that look like conflict counts but are not.

# 5. Cherry-pick selected commits (expect hand resolution: the P0 chain conflicts in 40 files
#    / 82 markers against current dua; per-commit table in TODO.md)
git cherry-pick <commit-hash>

# 6. Run tests after each cherry-pick
go test ./cmd/analyze ./cmd/status

# 7. Commit or amend as appropriate
```

### Commit Classification Guide

| Category | Examples | Action |
|----------|---------|--------|
| **Mole-specific** | `clean`, `uninstall`, `purge`, `optimize`, `trash`, `delete`, `mo `, `mole ` | **SKIP** |
| **macOS-only** | `battery`, `ioreg`, `Apple Silicon`, `Spotlight`, `Parallels VM` | **SKIP** (Linux doesn't need these) |
| **Platform-specific shell** | `clean.sh`, `uninstall.sh`, `mole` script | **SKIP** |
| **Bug fixes (generic)** | `cancel scan`, `process owner`, `stale data`, `snapshot count` | **REVIEW** |
| **Performance** | `scan cancellation`, `cache optimization` | **REVIEW** |
| **UI/UX** | `spinner`, `layout`, `UTF-8 safe`, `terminal fit` | **REVIEW** |
| **Tests** | test improvements, race fixes | **REVIEW** |

### Upstream Commits Since Fork (Analysis)

As of the 2026-10-09 audit (`upstream/main` = `2e8ea7f3`, newest tag `V1.58.0` = `8710fdad`), `git log b56f7561..upstream/main -- cmd/analyze/ cmd/status/` holds **140 commits** (135 non-merge + 5 merge; `--full-history` reports 165), and there are **0 open PRs**. Always state the query alongside a count — an unqualified count of the same range has produced 151 in an earlier revision of these docs and was wrong. Upstream keeps moving, so re-derive candidates from that command before porting; the hashes in [TODO.md](TODO.md) were verified as live ancestors of `upstream/main` on 2026-10-09 and only need re-deriving if upstream rewrites history again.

The live, prioritized backlog lives in **[TODO.md](TODO.md)**; this table is a summary only:

| Category | Count | Action |
|----------|-------|--------|
| Already ported by dua | 8 (A-layer + UI/UX batch, branch `port/a-layer-upstream`) | N/A |
| Partial-coverage chain (analyze) | ~13 | **IN PROGRESS** — 12 of 13 cards landed (`A-1`…`A-4`, `B-1`–`B-3`, `C-1`, `C-2`); `C-3`, `D-1`, `D-2` remain |
| Status stale-process / snapshot / health / locale / CPU card | ~15 | **REVIEW** |
| UI/UX (remaining: pending-row size alignment) | 1 | **REVIEW** |
| Mole-specific (cleanup/uninstall/Trash/battery/OrbStack/Claude) | ~90 | **SKIP** |

#### Potentially Useful Commits (Review Needed)

Categories worth reviewing (full hashes and ordering in [TODO.md](TODO.md)); the old hashes `b3dc7f31`…`1e0d23c9` in the git-log history are stale:

- **Partial-coverage chain (analyze)** — `cf6165b2`…`53c4d362` (13 commits): keep partial du measurements and incomplete-size markers visible, carry coverage through live scan events, expose coverage in JSON, preserve child caches. **In progress as cards:** `A-1` (state type), `A-2a`/`A-2b` (du keeps partial bytes, measure helpers take a context), `A-3` (state carried into scan results, the model, the TUI and the cache; includes the `B-1` schema bump to v4), `A-4` (`13f068ac`: live-scan coverage), `B-2` (`50c89d9b`/`20ac485b`: partial results cached by cause, schema v5) and `C-1` (`53c4d362` display half only: a short reason for a failed measurement, `duError` + `measurementErrorReason`, and a one-line overview header; `a4350259` does not apply, `b77a48d7` is test+README only) are ported. Remaining: `B-3` (a re-derivation — `B-2` already pinned the cache gates), `C-2` (JSON `scan_status`; `D-2`'s test depends on it), `C-3` (docs + release note), `D-1`/`D-2` (verification).
- **Status stale-process** — `390294d5` mark stale process data in the TUI, `eb20146b` retain stale process samples.
- **Analyze snapshot** — `a6d59d49` reject stale snapshot probes, `03311d45` refresh local snapshot count, `e9f52994` surface local snapshot space.
- **UI/UX** — ported on 2026-10-09: `7d08959d` compact 2x2 spinner glyphs, `6aa93263` + `19f14e40` one animation loop per analyze session. **Nothing is open here**: `99a9471d` (align pending overview sizes) and `d534c30b` (animate pending rows) are on an upstream history line that was later rewritten; their published twins `1cef46a9` and `53383db2` are ancestors of dua's fork base, so dua has had that behaviour since the split. `8a2b84b1` is a no-op here — it re-encodes `lib/core/ui.sh`, which dua does not have.
- **Status health/locale/CPU card** — `4ee83df7`/`825ef91b`/`cd95c912` health diagnosis, `bfbe240d` `--watch` NDJSON, `afc83fc5`/`b5fc149a` C locale, `4cbab499`/`00a42fd9` CPU card.
- **Analyze cache robustness** — `0dc42987`/`744e3a34` bound cache growth, `194bdd8c` prune expired cache, `6127d79c` serialize cache publication.
- **Performance:** scan cancellation improvements are mostly already ported by dua.

### Cherry-pick Decision Log

| Date | Category | Commits | Decision | Reason |
|------|----------|---------|----------|--------|
| 2026-10-09 | Partial-coverage chain, card `C-2` (JSON coverage contract) | upstream `1cbaad5e` (3 of its 4 hunks) | **PORTED, one hunk declined** | dua `6312713b`: `scan_status` on the analyze JSON document and on each entry, using `scanState.String()` so the CLI, TUI and scripts branch on one vocabulary. Two deviations were required: the overview document derives coverage with `entryScanState()` (dua injects pending rows as `-1`, while upstream's `scanHasPartialEntries()` presumes measured sizes), and an overview cache hit reuses the **stored** state (upstream's cache read returns bytes only; recomputing from "bytes with no error" would undo `B-2` on that path). The `MarshalText` hunk is declined — probed first: `scanState` is also dua's on-disk cache type, so marshalling it rewrites the cache format and every cache read then fails to decode, silently. Upstream's fixture expectation for an unreadable directory (`0` / `unavailable`) is not what a dua walk produces (it can stat the directory itself → its own blocks + `partial`), so the ported test asserts the invariant rather than the borrowed numbers. |
| 2026-10-09 | Partial-coverage chain, card `B-3` (child caches under a refresh) | upstream `f5b4130c` (both of its halves) | **PORTED** | dua `1499e33e`: `scanSubdirWithCache` dropped a subtree's stored measurement whenever a cache-bypassing pass (the `R` key) could not finish that subtree, so a permission denial on refresh threw away the only measurement of bytes that were readable a moment ago. Porting note: upstream's change is a guard *and* a test that switches its scan to `scanCacheBypass` — the drop branch never runs under the reuse policy, so only copying the source line would have produced a test that cannot fail. The new test sits two levels deep because `R` invalidates the scanned directory and its direct children first, and it asserts cache-file presence rather than byte counts so it does not self-skip on filesystems that under-report allocation. |
| 2026-10-09 | Partial-coverage chain, card `C-1` (measurement-failure diagnostics) | upstream `53c4d362` (2 of its 4 parts) | **PORTED, narrowed by re-derivation** | dua `efe025a8`. Ran as a re-derivation and was **not** a no-op: `53c4d362`'s size labels had already landed via `A-3`, but its diagnostic half had not, so an overview failure printed `Partial size for /home/x: du failed …: exit status 1` — no actionable cause, and long enough to wrap the header. Landed `duError` (carries the strerror from the first `du:` stderr line; `Unwrap` returns the cause so classification and the `B-2` cache decision are unchanged), `measurementErrorReason` (timeout / cancelled / access denied classified first, du's own text only as fallback), row-name instead of full path, and `writeOverviewStatus` which keeps the header on one line. **Not ported:** `a4350259` (space as multi-select) — dua's `KeySpace` is the filter input and the multi-select/Trash surface was removed at the fork; `b77a48d7` — touches `README.md` + a test only, so it is not a display card; it moves to `D-2`, whose test asserts `document.ScanStatus` and therefore needs `C-2` first. 6 new tests, 5 mutation probes all caught, `make check` green, and an end-to-end probe under `unshare -U` at 70 columns. Ran ~50m against a 30m box: the re-derivation was ~15m, the work it found was not — a re-derivation card should be sized as probe **plus one fix**. |
| 2026-10-04 | A-layer: CLI help, status network/JSON, overview perf | dua `55d62acd`, `a9529a0f`, `b5608780`, `a4593718`, `676829c8` | **PORTED** | Self-contained, Linux-relevant fixes. Ported as a batch on branch `port/a-layer-upstream`; `make check` green; both binaries verified running. |
| 2026-10-09 | Correction of the A-layer mapping (see note below) | upstream `4e4646b2`, `4f7d853a`, `1a1c7300`, `f2527b02`, `a6f37ec1`, `7d08959d` | **DOC FIX** | The 2026-10-04 row listed **dua** hashes as if they were upstream ones. Actual mapping: `676829c8` ← `a6f37ec1`; `b5608780` ← `1a1c7300` + test `f2527b02` (upstream changed `mo-applets/metrics.go`, dua ported it into `cmd/status/metrics_network.go`); `4f7d853a` (`status --json` without a TTY) was **already** in dua; `55d62acd` (router help) and `a9529a0f`/`a4593718` have **no upstream twin** — the former is dua-router-only, the latter two were written against an upstream history that no longer contains them. |
| 2026-10-09 | UI/UX: spinner glyph set + single animation loop | dua `9952bdde`, `e2994ff1` ← upstream `7d08959d`, `6aa93263`, `19f14e40` | **PORTED** | `6aa93263`/`19f14e40` replace the dual animation loop; `startTick` + `tickRunning` guard, `Init` → `initializeMsg` handoff. Ported upstream's `scheduledTickCount`-based regression tests, adapted to dua's model; `make check` green; TUI smoke-tested through a pty. (The claim in this row's first edit — that `d534c30b` was "superseded" by these two — was wrong: `d534c30b` is about pending overview rows and has nothing to do with the loop count; see the 2026-10-09 fork-history row below.) |
| 2026-10-09 | Fork-history check on the "pending rows" pair | upstream `1cef46a9`, `53383db2` (published twins of `99a9471d`, `d534c30b`) | **ALREADY IN THE TREE — no port needed** | Both twins are ancestors of dua's fork base, so dua inherited the `--` placeholder and then the animated `scanning` row. Verified three ways: `git merge-base --is-ancestor` on all four hashes, `git log -S 'scanning", spinnerFrames' -- cmd/analyze/view.go` (lands on `53383db2`, not on any dua commit), and a block-level diff of `view.go` against `d534c30b` — the only difference is `A-3`'s `measuredSizeLabel`. `TestOverviewPendingSizeUsesScanningSpinner` already pins the behaviour. **Process lesson**: several hashes in the backlog predate an upstream history rewrite, so a card can look open purely because its hash is no longer on the published line. Before sizing a card, run the ancestry check and a `-S` payload search. |
| 2026-10-09 | Partial-coverage chain, card `B-2` (cache partial scans) | upstream `50c89d9b`, `20ac485b` | **PORTED, with two deliberate deviations** | dua `1a43a9c1`. Reuse is decided by the **cause** of a coverage gap: `scanResult.partial` (a permission denial explains part of it → the measured bytes are a reusable lower bound) vs `transientFailure` (a gap may clear → `NeedsRefresh`), with `scanResult.persistable()` as the single decision point for the cache write, the model snapshot and the overview snapshot. **Not ported:** (a) upstream's `dirScanCacheEntry.Partial` string — dua's subtree cache *is* `cacheEntry`, which has carried `State` since `A-3`, so a second marker would be a second source of truth; (b) upstream's `cachePolicyFor("test")` invalidation — dua has no `test` cache policy and its tests assert the opposite (child caches survive) as shipped behaviour. `cacheSchemaVersion` 4 → 5 for the overview store only (a pre-B-2 snapshot can hold an unmarked lower bound; the subtree format is unchanged). Found two real bugs on the way: `loadCachedSubdirResult` dropped the stored `State`, so every cached partial read back as **complete** (broken since `A-3` first wrote it), and `du`'s stderr was discarded, so a `du` failure — i.e. every large tree, which is measured with `du` — classified as transient and no partial would ever have been cached in practice. `make check` green; 8 new tests; three mutation probes, one per gate, each red; functional probes with `unshare -U` (no CAP_DAC_OVERRIDE) plus a strace-confirmed cache read. Card time ~65m against a 90–120m estimate, because the policy decision had been written into the card first — the `A-2b` lesson held. |
| 2026-10-09 | Partial-coverage chain, cards `A-4` (live-scan events) | upstream `13f068ac` | **PORTED** | `cbeb4769`: `readLiveScanInitialEntries` returns a `scanResult` and tracks its own coverage; `runLiveScan` keeps an unscannable target as an `scanUnavailable` row, copies target state onto entries, reports the final state; `liveScanStartMsg.state` feeds `m.scanState`. Deviation: the same commit's `largeFileMinSize` parameter removals are not ported (they belong to an unported chain; dua still passes it). The hard-error branch is ported for parity but no dua fixture reaches it — the measure helpers return typed partial/unavailable results rather than errors. 2 new tests, one of which runs under root CI; green as root and under `unshare -U`. |
| 2026-10-09 | Partial-coverage chain, cards `A-1`–`A-3` + the `B-1` cache half | upstream `cf6165b2` (state type + model), `155a0cca`, `d04453f1`, `580b8eb0` (model/view), `50c89d9b` (cache part) | **PARTLY PORTED** | Ported as time-boxed cards on `port/a-layer-upstream` (`4d7f5b66`, `4cf39ec5`, `00d0e973`, plus this one): `scanState`/`measurementState`, partial `du` bytes survive instead of being re-walked, `dirEntry.State`/`scanResult.State`/`m.scanState`, coverage-aware size labels, cache schema v4. Not ported: upstream already had this state, so its view/JSON halves assume it — dua adds the display decisions itself (see the card work log in TODO.md). `B-2` closed 2026-10-09 (see the row above); `B-3`/`C-*`/`D-*` remain. |
| 2025-08-28 | A: Status stale-process | `390294d5`, `eb20146b` (+ older hashes `b3dc7f31`…`455e6627` now stale) | **REVIEW** | Smaller/self-contained than the original A set; re-check against current upstream before porting. |
| 2025-08-28 | B: Analyze snapshot | `a6d59d49`, `03311d45`, `e9f52994` (+ older hashes `a300f00c`…`608d5d04` now stale) | **REVIEW** | May depend on snapshot infra; verify self-containment against current HEAD. |
| 2025-08-28 | C: UI/UX | `99a9471d`, `d534c30b`, `8a2b84b1` (+ older hashes `785b84c4`…`1e0d23c9` now stale) | **CLOSED 2026-10-09 — nothing to port** | `8a2b84b1` is a no-op for dua (bash `lib/core/ui.sh`); `99a9471d`/`d534c30b` were already inherited via their published twins `1cef46a9`/`53383db2` (see the fork-history row). |

**Current policy**: Prefer small, self-contained, Linux-relevant fixes ported as a batch on a feature branch (never directly to `main`). The partial-coverage chain is 12 of 13 cards landed; `C-3` (docs + release), `D-1` (real-machine A/B) and `D-2` (navigation tests) remain, worked as time-boxed cards (≤ 90 minutes each, each ending green and committed) rather than one sitting — the card breakdown, per-card status markers, hard ordering, and the actual-time work log are in [TODO.md](TODO.md). When closing a card, mark it there and log the real elapsed time; when the estimate and the log disagree, re-size the remaining cards of that stage. Re-derive all upstream hashes from current `upstream/main` before porting, and record the **upstream** hash next to the dua commit that ports it — the 2026-10-09 correction above shows what happens when only local hashes are logged. When an upstream change touches `mo-applets/`, port the equivalent into `cmd/analyze/` / `cmd/status/` instead of skipping it: the two trees diverged, so the upstream path is not the porting target.

### Decision Checklist for Cherry-picking

Before cherry-picking any upstream commit, answer:

1. **Is this a bug fix or performance improvement?** (skip feature additions)
2. **Is it platform-agnostic or Linux-relevant?** (skip macOS-only features)
3. **Does it touch only analyze/status code?** (skip shell scripts)
4. **Is the fix self-contained?** (skip commits that depend on other Mole features)
5. **Can we run `go test ./cmd/analyze ./cmd/status` after cherry-picking?** (verify it compiles)

If any answer is "no", skip the commit.

### Regular Maintenance

- **Frequency**: Review upstream monthly or after major releases
- **Command**: `git log --oneline upstream/main -- cmd/analyze/ cmd/status/` to see new commits
- **Documentation**: Update this section when new commits are cherry-picked
- **Tags**: Keep track of which upstream commits were cherry-picked in git log

## Next Steps

- [x] Include version in `dua status` and `dua analyze` main output (TUI).
- [x] Include version in `dua status` and `dua analyze` JSON output.
- [x] Add `dua update` command to check GitHub and update in-place.
- [x] Port A-layer upstream fixes (CLI help, braille spinner, default-route tunnel accounting, partial one-shot JSON, overview concurrency budget) — branch `port/a-layer-upstream`.
- [ ] Port the partial-coverage chain (analyze partial coverage tracking) — in progress, worked one time-boxed card at a time (`S0-1`…`D-2` in [TODO.md](TODO.md)); do not attempt it as a single sitting. Done: `S0-1`, `S0-2`, `A-1`, `A-2a`, `A-2b`, `A-3`, `A-4`, `B-1`, `B-2`, `C-1`. Open: `B-3` (re-derivation only), `C-2`, `C-3`, `D-1`, `D-2`.
- [ ] Fold darwin/arm64 into `release.yml`'s build matrix (with macos smoke gate) and delete `upload-darwin.yml`.
