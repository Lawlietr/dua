<div align="center">
  <h1>dua</h1>
  <p><em>Disk analysis and system status for Linux, from the terminal.</em></p>
  <p>English | <a href="README.zh-TW.md">繁體中文</a></p>
</div>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL_v3-blue.svg?style=flat-square" alt="License"></a>
</p>

dua is a terminal-first Linux inspection toolkit with two read-only commands, forked from the macOS Mole CLI's `analyze` and `status` and re-scoped as an independent project. Linux is the primary platform; macOS arm64 is supported on a secondary, best-effort basis. It shows where disk space went and how healthy the machine is — from a CLI, script, or compact TUI. It **never deletes or modifies user data**: there is no cleanup surface anywhere in the codebase.

## Installation

### Binary install (curl)

Prebuilt Linux binaries (amd64 and arm64) are built by GitHub Actions. No sudo needed — installed to `~/.local/bin`:

```bash
curl -fsSL "https://raw.githubusercontent.com/Lawlietr/dua/main/scripts/install.sh" | bash
```

Add `~/.local/bin` to your `PATH` (usually in `~/.bashrc` or `~/.zshrc`) to make it permanent.

#### macOS (Apple Silicon)

Linux is the primary platform. A macOS arm64 build is also published as secondary support (since v0.2.0):

```bash
mkdir -p ~/.local/bin
curl -fsSL "https://github.com/Lawlietr/dua/releases/latest/download/dua-darwin-arm64.tar.gz" | tar -xz -C ~/.local/bin
```

Notes:
- Requires Apple Silicon; Intel Macs are not built.
- `dua update` works on macOS when using the router shipped in the darwin tarball.
- Opening files and folders (O / P / F keys) uses the macOS `open` command.
- Insight paths target XDG caches, so the insights list is sparser on macOS than on Linux.

### Build from source

Requires Go 1.25+:

```bash
make build
./dua status --json        # machine-readable status
./dua analyze --json /some/path
./dua analyze              # overview TUI (no PATH scans the machine)
./dua status               # TUI dashboard
```

Install the binaries to a location on your `PATH`:

```bash
cp bin/dua-analyze bin/dua-status /usr/local/bin/
```

`dua status --json` is resilient: if one collector fails it still prints the metrics that were collected, reports the failure on stderr, and exits 0 — the same way `--watch` keeps streaming. It exits 1 only when none of CPU, memory, disk, or process metrics are available, or when JSON encoding fails.

## Features

- **`dua analyze`** — visual disk explorer:
  - System overview (`/usr`, `/opt`, `/var`, `/home`) plus hidden-space insights (npm, Go build, pip, Gradle, JetBrains caches, Trash, old Downloads)
  - Drill-down navigation with per-directory bars and last-access hints
  - Top-files (T) view, live scan feedback with cursor sort modes, and a disk-usage cache
  - Coverage-aware sizes: a measurement that skipped unreadable subtrees is labelled a lower bound instead of a total, in the TUI and in `--json`
  - `--json` output for automation
- **`dua status`** — compact health dashboard:
  - CPU, memory, disk, thermal (hwmon), hardware model, OS info, and top processes
  - High-CPU process alerts
  - `--json` for one-shot automation and `--watch` for NDJSON streams

## Commands

### `dua analyze [PATH]`

- No PATH: system overview (system roots + insight directories), then drill in with Enter.
- `PATH`: scans that directory.
- Keys: `↑↓←→` navigate, `Enter` drill in, `Esc` back, `R` refresh, `/` filter, `T` Top files, `O` open with `xdg-open`, `P` preview, `F` reveal in file manager, `Q`/`Ctrl+C` quit.
- Flags: `--json` prints the scan result as JSON.
- Coverage: the JSON output carries `scan_status` (`complete`, `partial`, `unavailable`) on the document and on each entry, so a `total_size` that skipped an unreadable subtree reads as a lower bound instead of a total. In the TUI a lower-bound size carries a `+` (e.g. `739.6 MB+`) and a partially measured row shows `--` where its share would be, because a percentage of a lower bound is not a percentage; rows that were measured fully keep their exact numbers and shares. The interactive list keeps the 30 largest entries, so an entry dua could not measure may fall outside it while the total still reads `partial`; `--json` lists every entry the scan saw. After you restore access, `R` re-reads the directory and the missing bytes appear.

Environment: `DUA_ANALYZE_PATH` sets the scan target when no PATH is given; `DUA_ANALYZE_LIVE_SORT` selects the live-scan sort mode.

### `dua version`

Shows version and commit information.

### `dua update`

Checks GitHub for a newer release and updates in-place (downloads tarball, verifies SHA-256, replaces binaries). Run with `sudo` if installed in a system directory.

dua v0.3.0 changed the `analyze` cache format so a cached measurement keeps its coverage marker. Updating from an earlier release therefore costs one re-scan per tree: the first scan of a directory after updating re-reads it, later scans are served from the new cache again.

## How it Works

- `dua` is a thin bash router that dispatches to two Go binaries (`dua-analyze`, `dua-status`).
- Sizes come from a bounded concurrent walk; `du -skPx` (with platform-aware exclude flags) measures folded directories, and a cache avoids re-scanning unchanged trees.
- Status metrics come from `/proc`, `/sys/class/hwmon`, DMI, and `/etc/os-release`; no privileged access is required.
- Analysis is read-only by construction: no deletion, truncation, or modification code path exists.

## Development

```bash
make build     # build bin/dua-analyze and bin/dua-status
make check     # go vet ./... + go test ./...
go test ./...  # full test suite
```

See `AGENTS.md` for the project contract, hotspot ownership, and testing notes.

## License

GPL v3. See [LICENSE](LICENSE).
