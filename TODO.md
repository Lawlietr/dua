# dua Upstream Backlog (TODO)

Live backlog of upstream (`tw93/Mole`) commits worth porting to dua.

- **Fork base**: `b56f7561`.
- **Source of truth**: `git log b56f7561..upstream/main -- cmd/analyze/ cmd/status/` (134 commits as of 2026-10-04).
- **Branch for porting**: `port/a-layer-upstream` (A-layer done). **Never port directly to `main`.**
- Upstream rewrote history since the fork, so re-derive hashes against current `upstream/main` before porting.

## P0 — Next batch: analyze partial-coverage chain

Keep incomplete / partially-sized measurements visible across navigation, live scans, JSON, and caches. **Port as one unit** (13 commits; not cherry-pickable individually). Estimated ~20–35h for someone familiar with the codebase.

| Commit | Message | Hotspot |
|--------|---------|---------|
| `cf6165b2` | Preserve coverage when directory scans fail | scanner.go |
| `155a0cca` | Retain partial du measurements without rescanning | scanner.go |
| `13f068ac` | Carry partial coverage through live scan events | live_scan.go |
| `580b8eb0` | Keep incomplete measurements visible in analyze | model.go / view.go |
| `1cbaad5e` | Expose scan coverage in analyze JSON | json.go |
| `f5b4130c` | Keep complete child caches after partial refreshes | cache.go |
| `a4350259` | Share partial selection accounting with confirmation | model.go |
| `b77a48d7` | Define partial coverage beyond the terminal entry limit | model.go |
| `318ee925` | Verify partial scan recovery through navigation | test |
| `50c89d9b` | fix(analyze): cache partial scans that only lack unreadable folders | cache.go |
| `20ac485b` | fix(analyze): invalidate changed sizes and reject stale cache writes | cache.go |
| `53c4d362` | fix(analyze): clarify incomplete size measurements | format.go |
| `d04453f1` | Preserve overview measurement failures and deadlines | scanner.go / update.go |

Hotspots touched: `scanner.go`, `model.go`, `cache.go`, `live_scan.go`, `json.go`, `view.go`, `format.go`, `insights.go`, `update.go`.
dua has no coverage symbols yet — this is **new functionality, not a refactor**.

## P1 — Status: stale process data (small, self-contained)

| Commit | Message |
|--------|---------|
| `390294d5` | fix(status): mark stale process data in the TUI |
| `eb20146b` | fix(status): retain stale process samples |

## P2 — Analyze: local snapshots

| Commit | Message |
|--------|---------|
| `a6d59d49` | fix(analyze): reject stale snapshot probes |
| `03311d45` | fix(analyze): refresh local snapshot count (#1469) |
| `e9f52994` | fix(analyze): surface local snapshot space (#1467) |

## P3 — UI/UX: pending overview animation + spinner

| Commit | Message |
|--------|---------|
| `99a9471d` | fix(analyze): align pending overview sizes with the numeric column |
| `d534c30b` | fix(analyze): animate pending overview rows with the scanning spinner |
| `8a2b84b1` | fix(ui): keep braille spinner frames UTF-8 safe |

## P4 — Status: health diagnosis, --watch, locale, CPU card

| Commit | Message |
|--------|---------|
| `4ee83df7` | feat(status): add inline health diagnosis |
| `825ef91b` | refactor(status): split diagnosis helpers into diagnosis.go |
| `cd95c912` | fix(status): keep health score monotonic past CPU and memory high thresholds |
| `bfbe240d` | feat(status): add --watch streaming NDJSON mode (#1138) |
| `afc83fc5` | fix(status): force C locale for all metric subprocesses |
| `b5fc149a` | fix(status): collect processes under comma-decimal locales |
| `4cbab499` | feat(status): let the CPU card show a configurable number of cores |
| `00a42fd9` | fix(status): keep the CPU card inside the window and document the key |

## P5 — Analyze: cache robustness + live scan rows

| Commit | Message |
|--------|---------|
| `0dc42987` | perf(analyze): cut the cache hot path and bound the overview store |
| `744e3a34` | fix(analyze): stop the disk cache from growing without bound |
| `194bdd8c` | fix(analyze): prune expired cache files (#902) |
| `6127d79c` | fix(analyze): serialize cache publication |
| `9fcbadf2` | feat(analyze): show live directory scan rows |

## SKIP — Mole-specific / macOS / cleanup (not dua scope)

dua is read-only and Linux-first. Do NOT port these:

| Commit | Message | Why skip |
|--------|---------|----------|
| `9235bf7e` | fix(analyze): move to Trash via trash(8) | deletion surface |
| `f899cbf8` | fix(analyze): bypass nested caches on manual refresh | cleanup |
| `e8968b4b` | List uv cache in 'mole analyze' | mole-specific cache |
| `0b174ffa` | feat: recognize CACHEDIR.TAG cache markers | mole-specific cache |
| `794d2233` / `7f94ad35` | battery health | macOS-only |
| `acd40d9b` / `cadbee22` | OrbStack + Claude cleanup | mole-specific cleanup |
| `4add6a48` / `bf08459d` | cleanup safeguards | deletion surface |
| … | ~80 more Mole-only commits | cleanup / uninstall / battery / Spotlight |

## Notes

- Verify each batch: `make check` (vet + tests), then run `./dua status` / `./dua analyze` from the repo.
- Test procedure (avoid confusing the stable install): always invoke via the repo path `./dua` or `/root/opencode-stuffs/dua/dua`; never run bare `dua` (that hits the stable version in `~/.local/bin`). Discriminate by `dua version` output: dev build shows `v0.2.0-<n>-g<short>` + commit, stable shows the release tag.
- Full upstream has 134 analyze/status commits since the fork; only the tables above are Linux-relevant and non-Mole.
