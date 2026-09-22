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

- The test suite runs as the invoking user. Tests that depend on permission-denied semantics (`chmod 000`) call `skipIfRoot` so they skip under root CI. Tests that depend on `st_blocks` size accounting call `skipIfBlockAccountingUnreliable` (some containerized filesystems, e.g. ZFS, report one block per file and collapse sparse/actual-usage accounting).
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
- `cmd/analyze/cache.go` owns analyze cache schema, expiry, load/save, invalidation, and cacheability decisions. Computation changes must invalidate stale persisted data in the same change. Run `go test ./cmd/analyze`.
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

# 4. Cherry-pick selected commits
git cherry-pick <commit-hash>

# 5. Run tests after each cherry-pick
go test ./cmd/analyze ./cmd/status

# 6. Commit or amend as appropriate
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

As of the latest check, upstream has **32 commits** affecting `cmd/analyze/` and `cmd/status/` since the fork point:

| Category | Count | Action |
|----------|-------|--------|
| Already ported by dua | ~5 | N/A |
| Mole-specific (cleanup/uninstall/etc.) | ~15 | **SKIP** |
| macOS-specific (battery/ioreg/Spotlight) | ~8 | **SKIP** |
| Potentially useful for review | ~4 | **REVIEW** |

#### Potentially Useful Commits (Review Needed)

**Status process metrics improvements:**
- `b3dc7f31` fix(status): mark stale process data in the TUI
- `a0fd0dc1` fix(status): retain stale process samples
- `c034c3fc` fix(status): preserve multiword process owner names
- `0b1e34d5` fix(status): expose process sample freshness
- `f92133a4` fix(status): surface zombie processes and parent owners
- `0be91406` fix(status): read the core topology by level order
- `455e6627` fix(status): stop reporting an active tunnel interface as a proxy

**Analyze snapshot improvements:**
- `a300f00c` fix(analyze): reject stale snapshot probes
- `76214aac` fix(analyze): refresh local snapshot count
- `608d5d04` fix(analyze): surface local snapshot space

**UI/UX improvements:**
- `785b84c4` fix(ui): keep braille spinner frames UTF-8 safe
- `2057e1ea` fix(analyze): measure the whole bar in eighths of a cell
- `6e4c7455` fix(analyze): quiet the sub-cell bars and fit the scan path
- `1e0d23c9` feat(status): balance two-column layout without growing dashboard

**Performance:**
- Scan cancellation improvements (大部分已由 dua port 過)

### Cherry-pick Decision Log

| Date | Category | Commits | Decision | Reason |
|------|----------|---------|----------|--------|
| 2025-08-28 | A: Status process metrics | `b3dc7f31`, `a0fd0dc1`, `c034c3fc`, `0b1e34d5`, `f92133a4`, `0be91406`, `455e6627` | **SKIP** | Requires `ZombieParent`, `summarizeZombies`, `processSample`, `parseProcessOutputStrict` and other types/functions not in HEAD. Not self-contained. |
| 2025-08-28 | B: Analyze snapshot | `a300f00c`, `76214aac`, `608d5d04` | **SKIP** | Depends on snapshot infrastructure changes in upstream not present in HEAD. |
| 2025-08-28 | C: UI/UX | `785b84c4`, `2057e1ea`, `6e4c7455`, `1e0d23c9` | **REVIEW LATER** | Low priority, needs TUI compatibility check. |

**Current policy**: Skip A/B category commits. They require significant porting effort (multiple new types, functions, and struct changes) and are not critical for dua's Linux target users. Re-evaluate if upstream refactors into smaller, self-contained patches.

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
- [ ] Fold darwin/arm64 into `release.yml`'s build matrix (with macos smoke gate) and delete `upload-darwin.yml`.
