<div align="center">
  <h1>dua</h1>
  <p><em>Disk analysis and system status for Linux, from the terminal.</em></p>
  <p>English | <a href="README.zh-TW.md">繁體中文</a></p>
</div>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL_v3-blue.svg?style=flat-square" alt="License"></a>
</p>

dua is a terminal-first Linux inspection toolkit with two read-only commands, forked from the macOS Mole CLI's `analyze` and `status` and re-scoped as an independent project. It shows where disk space went and how healthy the machine is — from a CLI, script, or compact TUI. It **never deletes or modifies user data**: there is no cleanup surface anywhere in the codebase.

## Installation

### Binary install (curl)

Prebuilt Linux binaries (amd64 and arm64) are built by GitHub Actions. No sudo needed — installed to `~/.local/bin`:

```bash
curl -fsSL "https://raw.githubusercontent.com/Lawlietr/dua/main/scripts/install.sh" | bash
```

Add `~/.local/bin` to your `PATH` (usually in `~/.bashrc` or `~/.zshrc`) to make it permanent.

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

## Features

- **`dua analyze`** — visual disk explorer:
  - System overview (`/usr`, `/opt`, `/var`, `/home`) plus hidden-space insights (npm, Go build, pip, Gradle, JetBrains caches, Trash, old Downloads)
  - Drill-down navigation with per-directory bars and last-access hints
  - Top-files (T) view, live scan feedback with cursor sort modes, and a disk-usage cache
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

Environment: `DUA_ANALYZE_PATH` sets the scan target when no PATH is given; `DUA_ANALYZE_LIVE_SORT` selects the live-scan sort mode.

### `dua version`

Shows version and commit information.

### `dua update`

Checks GitHub for a newer release and updates in-place (downloads tarball, verifies SHA-256, replaces binaries). Run with `sudo` if installed in a system directory.

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
