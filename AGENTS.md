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

1. Does it clearly belong to analyze, status, or the router?
2. Is it read-only, previewable, testable, and explainable in one terminal screen?
3. Can the user verify what dua reports before trusting it?
4. Is the target data locally rebuildable, disposable, or backed by exact path evidence?
5. Would this be better as documentation, a warning, or an explicit "not supported" answer?

If the answer is no or unclear, decline the feature, narrow it, or park it until the product value beats the added surface area.

## Repository Map

- `AGENTS.md` is the cross-agent source of truth. `CLAUDE.md` must remain a symlink to it so Claude and Codex receive the same project contract.
- `dua` - the CLI entrypoint. It is a **router only**: it parses args and dispatches to the Go binaries. Business logic does not belong here.
- `cmd/analyze/` - the disk-analysis Go binary (TUI + `--json`). `main.go` is bootstrap only; `model.go` holds types and accessor methods; `update.go` holds the Bubble Tea Update chain; `scanner.go` owns traversal and concurrency budgets.
- `cmd/status/` - the system status Go binary (TUI, `--json`, `--watch`). `view.go` renders only; collection and JSON/NDJSON contracts live in the other `cmd/status/` files.
- `internal/units/` - shared byte formatting.
- `bin/` - build output directory (gitignored); `dua-analyze` and `dua-status` live here after `make build`.
- `Makefile` - Linux-only build/test entrypoints. `CGO_ENABLED=0`, `-trimpath`, `-s -w`.

Platform-specific files in `cmd/analyze/` carry explicit build tags (`atime_linux.go`, `atime_darwin.go`). The darwin files are preserved so the binaries still cross-compile, but Linux is the supported runtime.

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
gofmt -l cmd/ internal/
```

Public docs and examples should prefer the installed `dua` command. Use `./dua` in this repository when verifying source-tree behavior before building. `analyze` and `analyse` are both accepted command spellings.

## Working Rules

- **dua is read-only.** Before adding any code path that deletes, truncates, moves, or modifies files, stop and reconsider: the deletion surface was deliberately removed during the Linux port.
- Keep analyze scan behavior conservative and bounded: every scan has a timeout budget and inner-loop checkpoints; a timed-out producer must not feed partial output into downstream consumers.
- `du` is invoked with `-skPx` plus exclude patterns. The exclude flag is platform-aware (`duExcludeArgs`): GNU du (Linux) uses `--exclude=NAME`, BSD du (macOS) uses `-I NAME`. Verify against the target platform's `du` before changing flags.
- `xdg-open` is the default file opener (O / P / F keys). Validate paths (`validatePath`) before passing them to any external command.
- Spotlight (`mdfind`) integration is macOS-only and guarded by `runtime.GOOS == "darwin"`. The walk-based collector is the Linux large-file source.
- Overview roots and insight paths are platform-aware (`systemOverviewRoots`, `createInsightEntries`). Linux uses `/usr`, `/opt`, `/var`, `/home` and XDG cache paths (`~/.cache`, `~/.npm`, `~/.local/share/Trash`, ...).
- Keep shell code minimal; the `dua` router is the only shell surface. Format with `./scripts/check.sh --format` only if scripts exist; there is none today.
- Do not add AI attribution trailers to commits.

### Testing Notes

- The test suite runs as the invoking user. Tests that depend on permission-denied semantics (`chmod 000`) call `skipIfRoot` so they skip under root CI. Tests that depend on `st_blocks` size accounting call `skipIfBlockAccountingUnreliable` (some containerized filesystems, e.g. ZFS, report one block per file and collapse sparse/actual-usage accounting).
- macOS-only behavior tests (Spotlight invocation, OrbStack insight, Trash paths) are gated on `runtime.GOOS == "darwin"` and skip on Linux.
- Never pipe a test, check, or CI run into `tail` or `head`. The pipeline reports the pager's exit code, so a red run reads green. Let it print in full, or capture to a file and check the status separately.
- Prefer targeted `go test ./cmd/analyze` / `go test ./cmd/status` during development; run `go test ./...` before committing.

## Hotspot Ownership

These files are intentionally large. Do not start by splitting them. Keep edits narrow and run the listed tests when touching each area.

- `cmd/analyze/update.go` owns the Bubble Tea `Update` chain and message handlers (Init, scanCmd, updateKey, goBack, switchToOverviewMode, enterSelectedDir). This is the largest file in `cmd/analyze/` and the natural landing spot for new key bindings, message types, or navigation behavior. Run `go test ./cmd/analyze`.
- `cmd/analyze/scanner.go` owns disk traversal, Spotlight integration (darwin-only), cancellation, and all scan concurrency budgets. Treat its semaphores as independent resource limits and measure before changing them. Run `go test ./cmd/analyze`.
- `cmd/analyze/cache.go` owns analyze cache schema, expiry, load/save, invalidation, and cacheability decisions. Computation changes must invalidate stale persisted data in the same change. Run `go test ./cmd/analyze`.
- `cmd/analyze/analyze_test.go` and `cmd/status/view_test.go` are test hotspots. Add new cases near related behavior; split later only when touching many adjacent cases. Run `go test ./cmd/...`.
- `cmd/status/view.go` owns status rendering only; collection and JSON/NDJSON contracts live elsewhere in `cmd/status/`. Keep narrow-terminal layout and automation output independent. Run `go test ./cmd/status`.

## Verification

- Go changes: run `gofmt -l cmd/ internal/` (must be empty), then `go test ./...` and `go vet ./...`.
- Cleanup behavior: not applicable — dua is read-only. If a change adds modification behavior, stop and re-read the Product Direction.
- Documentation-only changes: check links and commands.
- Manual verification: `./dua status --json`, `./dua analyze --json /some/path`, `./dua analyze` and `./dua status` TUI smoke tests (run under a pseudo-TTY if not attached).

`make build` and `make check` are wrappers around the commands above.
