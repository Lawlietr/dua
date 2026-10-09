# dua Upstream Backlog (TODO)

Live backlog of upstream (`tw93/Mole`) commits worth porting to dua.

- **Fork base**: `b56f7561`.
- **Source of truth**: `git log b56f7561..upstream/main -- cmd/analyze/ cmd/status/` → **140** commits (135 non-merge + 5 merge commits) as of the 2026-10-09 S0-1 audit; `git log --full-history` reports 165 because it also walks merge ancestors. Quote the method with the number: the earlier "151" in this file was an unqualified count of a different query.
- **Upstream head**: `2e8ea7f3` (2026-10-09, `fix(optimize): …`), which sits a few commits past tag `V1.58.0` (`8710fdad`, 2026-10-05). The newest commit touching `cmd/analyze`/`cmd/status` is `1aad186b` (2026-10-08), already listed under SKIP — nothing new to triage since the previous check.
- **Open upstream PRs**: 0 (`gh pr list --repo tw93/Mole --state open` → `[]`, 2026-10-09) — only merged upstream commits need tracking.
- **Hash validity**: all hashes in this file (the 13-commit P0 chain plus P1/P2/P4/P5/P3) were checked with `git merge-base --is-ancestor <h> upstream/main` on 2026-10-09 — **all still live**, no re-derivation needed.
- **Branch for porting**: `port/a-layer-upstream` (A-layer + UI/UX batches done). **Never port directly to `main`.**
- Upstream rewrote history since the fork, so re-derive hashes against current `upstream/main` before porting. Every entry below was re-verified as a live `upstream/main` ancestor on 2026-10-09.

**Tracking convention**: every backlog item is a card. `☐` = not started, `◐` = in progress (reverted at the end of the session, nothing on disk), `☑` = merged as its own commit. When a card closes, mark it and append a row to the [Work log](#work-log) with the real elapsed time — the estimates below are planning inputs, the log is what tells us whether a card size is realistic.

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

Measured porting cost (2026-10-09 S0-1 audit). Method, because the first attempt at this number was wrong: `git apply --check -3` mixes a failed plain apply with the 3-way fallback in its output, so its "hunk" count measured nothing useful. The number below comes from a real `git cherry-pick -x <commit>` onto the branch tip inside a throwaway `git worktree`, counted as conflicted files and `<<<<<<<` markers, then `git cherry-pick --abort`:

| Commit | Conflicted files | Markers | Card | Note |
|---|---|---|---|---|
| `cf6165b2` | 0 | 0 | `A-1` | applies clean — still has to compile and be wired by hand |
| `d04453f1` | 2 | 2 | `A-2` | small conflicts, but it is the context-threading half of `A-2` |
| `155a0cca` | 3 | 4 | `A-2` | the `du` error-semantics change |
| `13f068ac` | 1 | 1 | `A-4` | cheap mechanically |
| `580b8eb0` | 2 | 2 | `A-3` | |
| `f5b4130c` | 1 | 1 | `B-3` | |
| `50c89d9b` | **9** | **43** | `B-1`/`B-2` | the heaviest commit in the chain by a wide margin |
| `20ac485b` | 7 | 13 | `B-2` | 606 of its changed lines are tests |
| `53c4d362` | 6 | 7 | `C-1` | |
| `a4350259` | 4 | 4 | `C-1` | |
| `1cbaad5e` | 2 | 2 | `C-2` | |
| `b77a48d7` | 2 | 2 | `D-2` | test + README only |
| `318ee925` | 1 | 1 | `D-2` | test only |
| **Total** | **40** | **82** | | 12 of 13 conflict; `cf6165b2` is the only clean apply |

Context drift since the chain landed (`git diff d04453f1 upstream/main -- cmd/analyze/<file>`): `update.go` +189/−80, `scanner.go` +151/−36, `model.go` +119/−8, `cache.go` +106/−41, `live_scan.go` +65/−36, `json.go` +32/−26, `view.go` +29/−21 — ~960 changed lines around the code being ported. Port it by hand (rebase-style); a mechanical cherry-pick will not survive.

### Time-boxed work cards for P0

Slicing rules for this backlog — each card is a work-session unit:

- **One card ≤ 90 minutes.** If a card cannot be finished in the session, revert it and stop; never leave a half-wired card on disk.
- **Every card ends green**: `make check` passes and the result is committed on the porting branch. Inert additions (types + tests, not wired yet) are a legitimate card for exactly this reason.
- **One upstream commit per card where possible**, so a revert maps back to a known upstream change.
- Do not ship Stage A without Stage C in the *same release*: a partial marker that the TUI and JSON cannot show is dead weight.

| Card | Scope | Upstream | Est. | Done when | Safe stop |
|---|---|---|---|---|---|
| `S0-1` ☑ | Re-fetch upstream, re-derive the chain diff, refresh the hash table in this file | — | 0.5h | this section lists current hashes and hunk counts | always safe |
| `S0-2` ☐ | Test fixture helper for unreadable dirs that does **not** rely on `chmod 000` + root skip (our CI runs as root, so upstream's fixtures self-skip) | `318ee925` infra | 1h | helper + one self-verifying test runs under root CI | revert helper |
| `A-1` ☑ | Add `scanState` type (`scanComplete`/`scanPartial`/…), `measurementState()`, unit tests. **Not wired into the model yet.** | `cf6165b2` (type part) | 45m | ✅ `go vet` clean, 4 tests pass (3 mutations checked), zero behavior change | done in `A-1` commit; A-3 will consume it |
| `A-2a` ☑ | Make `du` keep "size + error = partial" instead of dropping the size: runner semantics, `scanFailures`, the three measurement guards, and the skip-immediate-child accumulator | `155a0cca` | 60m | ✅ partial `du` bytes survive at the du boundary, at a folded scan and in live scan; nothing re-walks | done; unblocks A-3 |
| `A-2b` ☑ | Thread `context` through the measure helpers (`measureOverviewSize`, `getDirectoryLogicalSizeWithExclude`, `getDirectorySizeFromDuSkippingImmediateChild`, `measureInsightSize`, `measureOldDownloads`) and stop `measureOverviewSize` from re-walking on a partial du | `d04453f1` | 60m | ✅ overview keeps partial bytes with one measurement; cancellation propagates | done; see the deviation note in the work log |
| `A-3` ☐ | Carry the state into scan results and the model (`m.scanState`, per-entry state); keep `NeedsRefresh` semantics consistent | `cf6165b2`, `580b8eb0` (model part) | 90m | a partial scan still renders correct totals and marks state; covers the untested `calculateDirSizeConcurrent` guard; replaces the interim `size > 0` display widening left by `A-2b` with a real marker | needs A-2a; revert after it |
| `A-4` ☐ | Carry partial coverage through live scan events | `13f068ac` | 45m | live scan does not overwrite partial rows with zeros | independent of B |
| `B-1` ☐ | `historyEntry` gains state; bump `cacheSchemaVersion` 3 → 4 so stale caches are rejected | `50c89d9b` (cache part) | 45m | old caches are ignored, not misread; note the one-time re-scan for release notes | must land with B-2 in the same release |
| `B-2` ☐ | Cache partial scans correctly: partial results are reusable, a complete scan may overwrite them, stale writes are rejected, changed sizes invalidate | `50c89d9b`, `20ac485b` | 90–120m | navigation + relaunch keeps partial without rescanning forever | second riskiest card |
| `B-3` ☐ | Keep complete child caches after a partial parent refresh | `f5b4130c` | 45m | child caches survive a partial refresh | low coupling |
| `C-1` ☐ | TUI labels for partial/incomplete sizes (`measuredSizeLabel`, `scanSummary`) and pending-size column alignment | `53c4d362`, `580b8eb0` (view), `99a9471d`, `a4350259`, `b77a48d7` | 60m | a partial row is visibly distinguishable and aligned | view-only |
| `C-2` ☐ | JSON contract: `scan_status` on the document and on entries + tests | `1cbaad5e` | 45m | `dua analyze --json` exposes coverage | additive field only |
| `C-3` ☐ | Docs: `README.md` + `README.zh-TW.md` in sync, version decision (**minor bump → v0.3.0**, behavior change), release note about the one-time cache invalidation | — | 45m | both READMEs match heading-for-heading | last card of the release |
| `D-1` ☐ | Real-machine A/B: run the same path before/after, diff the totals and the new markers | — | 30m | a short note in the PR with the observed diff | always safe |
| `D-2` ☐ | Port the navigation-regression test suite on top of the `S0-2` fixture | `318ee925`, `b77a48d7` | 45m | tests run (not skip) under our root CI | test-only |

Porting notes learned while doing `A-1` (cheap risk reductions for `A-3`/`B-2`):

- Upstream's coverage tests call `scanPathConcurrentAllEntries`, `calculateDirSizeFast`, `saveCacheToDisk`, `loadCacheFromDisk`, `writeFileWithSize` and `skipIfRoot` — **all of these already exist in `cmd/analyze/` with the same names**, so the tests in `cf6165b2`/`20ac485b` port almost verbatim once the `State` fields exist. The blocker is the fixture, not the API surface → that is exactly why `S0-2` gates `D-2`.
- Upstream's `scanState` uses `scanComplete = iota` deliberately: cache files written before coverage tracking decode `State` as 0, so the zero value must mean complete. `TestScanStateZeroValueMeansComplete` pins this; `B-2` (cache semantics) must not re-order those constants.
- `scan_state_test.go` exists but nothing calls `measurementState` yet — `go vet` does not flag unused package-level functions, and CI runs vet + tests only. If `A-3` slips, this type stays dead code on the branch; that is the accepted cost of the slicing rule.

Suggested session plan: `A-3` is the only card that can start next (90m); after it `A-4` (45m) and `C-1` (60m) are both open. Open cards total ~9–9.5h (`A-3` 90m, `A-4` 45m, `B-1` 45m, `B-2` 90–120m, `B-3` 45m, `C-1` 60m, `C-2` 45m, `C-3` 45m, `D-1` 30m, `D-2` 45m); `S0-1`, `A-1`, `A-2a` and `A-2b` are done. Estimates were re-sized after `A-2a` ran 60m→~95m and `A-2b` 60m→~85m: the unmodelled cost in both was the per-site mutation probing needed to prove a behaviour change is pinned, and in `A-2b` also a consumer-side decision the card text did not anticipate.

Hard ordering (everything else is free-choice):

1. `A-1` before `A-2a`/`A-3` — the type has to exist before anything can carry it.
2. `A-3` before `A-4` — live-scan events read the model state. `A-2a` before `A-3` as well: `A-3`'s acceptance criterion (a partial scan still renders correct totals) needs a scanner that actually produces partials, which is what `A-2a` added. (A earlier note in this session suggested doing `A-4` right after `A-1` — that was wrong on both counts.)
3. `A-2a` before `B-2` — the cache cannot persist a state the scanner never produces.
4. `B-1` and `B-2` in the same release — a half-bumped cache schema misreads old entries.
5. `S0-2` before `D-2` — the navigation tests need a fixture that is not skipped under root CI.
6. `A-*` + `B-*` + `C-*` all present before tagging; `C-3` is the last card of the release.
7. Stop-anywhere cards (safe to close a session on): `S0-1`, `A-1`, `A-4`, `B-3`, `C-1`, `C-2`, `D-1`, `D-2`.

## P1 — Status: stale process data (small, self-contained)

| Card | Commit | Message | Est. |
|---|---|---|---|
| `P1-1` ☐ | `eb20146b` | retain stale process samples (collection side) | 45m |
| `P1-2` ☐ | `390294d5` | mark stale process data in the TUI (`cmd/status/view.go`) | 45m |

`P1-1` must land first; `P1-2` depends on it but is a separate commit — both are safe to stop between.

## P2 — Analyze: local snapshots

| Card | Commit | Message | Est. |
|---|---|---|---|
| `P2-0` ☐ | — | spike: check whether the snapshot probe path exists in dua at all (Linux: `btrfs`/`zfs` snapshots, not Time Machine) before porting | 30m |
| `P2-1` ☐ | `a6d59d49` | reject stale snapshot probes | 45m |
| `P2-2` ☐ | `03311d45` | refresh local snapshot count (#1469) | 45m |
| `P2-3` ☐ | `e9f52994` | surface local snapshot space (#1467) | 60m |

Do `P2-0` first — if dua has no snapshot source on Linux, the whole P2 group becomes a documented skip instead of ~2.5h of porting.

## P3 — UI/UX: spinner glyphs + one animation loop (done 2026-10-09)

Ported on `port/a-layer-upstream`:

| Upstream | Message | dua commit | Note |
|---|---|---|---|
| `7d08959d` ☑ | fix(ui): use braille spinner frames for consistent stroke density | `9952bdde` | The A-layer log claimed this was done on 2026-10-04; it never was. |
| `6aa93263` ☑ | fix(analyze): keep one spinner loop while overview scans refill | `e2994ff1` | Includes `TestOverviewRefillsKeepOneTickLoop`; verified red against the pre-change code. |
| `19f14e40` ☑ | fix(analyze): keep one analyze animation loop across entry points | `e2994ff1` | `startTick` guard + `Init` → `initializeMsg` handoff; upstream's `scheduledTickCount` test helper and entry-point table test adapted to dua's model. |

Still open from the original P3 list:

| Upstream | Message | Status |
|---|---|---|
| `d534c30b` | animate pending overview rows | **Do not port** — upstream reverted it in favour of `6aa93263` / `19f14e40`. |
| `8a2b84b1` | keep braille spinner frames UTF-8 safe | **Skip** — it re-encodes `lib/core/ui.sh` (bash), which dua does not have; the frames live in Go. |
| `99a9471d` | align pending overview sizes with the numeric column | **Open** — the sizes half of the pair that `d534c30b`/`6aa93263` replaced; overlaps with the P0 chain, so fold it in there. |

## P4 — Status: health diagnosis, --watch, locale, CPU card

Cards follow the P0 slicing rules (≤ 90 min, end green, one commit each). The locale pair is the highest value per hour: it fixes wrong numbers on non-English systems.

| Card | Commit | Scope | Est. | Notes |
|---|---|---|---|---|
| `P4-L1` ☐ | `afc83fc5` | force C locale for every metric subprocess | 45m | bug fix, platform-agnostic, no surface area added |
| `P4-L2` ☐ | `b5fc149a` | collect processes under comma-decimal locales | 45m | lands after `P4-L1`, shares its tests |
| `P4-V1` ☐ | `bfbe240d` | verify whether `--watch` NDJSON is already in dua | 20m | dua ships `--watch` already — likely a documented skip, not a port |
| `P4-H1` ☐ | `cd95c912` | keep the health score monotonic past CPU/memory high thresholds | 45m | only if dua has the health score; check first |
| `P4-H2` ☐ | `4ee83df7` | inline health diagnosis | 90m | **feature**, needs the product filter + a minor version bump |
| `P4-H3` ☐ | `825ef91b` | split diagnosis helpers into `diagnosis.go` | 30m | only after `P4-H2` |
| `P4-C1` ☐ | `00a42fd9` | keep the CPU card inside the window | 45m | rendering only (`cmd/status/view.go`) |
| `P4-C2` ☐ | `4cbab499` | configurable number of cores on the CPU card | 60m | adds a control — needs the product filter |

## P5 — Analyze: cache robustness + live scan rows

These are independent of the P0 chain except for the `cache.go` hotspot; take them one card at a time.

| Card | Commit | Scope | Est. | Notes |
|---|---|---|---|---|
| `P5-1` ☐ | `744e3a34` | stop the disk cache growing without bound | 60m | standalone fix |
| `P5-2` ☐ | `194bdd8c` | prune expired cache files (#902) | 45m | pairs with `P5-1`, separate commit |
| `P5-3` ☐ | `6127d79c` | serialize cache publication | 45m | concurrency fix; re-run with `-race` |
| `P5-4` ☐ | `0dc42987` | cut the cache hot path, bound the overview store | 90m | measure before/after; treat `scanner.go` semaphores as independent budgets |
| `P5-5` ☐ | `9fcbadf2` | show live directory scan rows | 90m | **feature** — needs the product filter; overlaps with `A-4` |

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
| `d3826922` / `e3f2e3bd` / `1aad186b` | 2026-10-06…08: `~/Library` per-child du sizing, hardlink counting, dropping the Library du pool | macOS size source; dua's overview roots are `/usr`, `/opt`, `/var`, `/home` + XDG paths |
| `f2de1148` | 2026-10-06: collect hardware inside the concurrent burst | touches `mo-applets/hardware/paths.go` (`system_profiler`); dua's `cmd/status/metrics_hardware.go` is macOS-sourced too |
| … | ~80 more Mole-only commits | cleanup / uninstall / battery / Spotlight |

Skipping `d3826922` / `e3f2e3bd` / `1aad186b` still costs drift: they keep reshaping upstream `cmd/analyze/scanner.go`, which widens the diff for the P0 chain. Re-derive that diff right before starting the batch.

## Work log

One row per closed card. The `actual` column is the input for re-sizing future cards.

| Date | Card | Est. | Actual | Commit | Outcome / notes |
|---|---|---|---|---|---|
| 2026-10-09 | `A-2b` | 60m | ~85m | `00d0e973` | Ported `d04453f1`: ctx threaded through all five measure helpers, `measureOverviewSize` keeps a partial du total and only pays for the logical walk when du gave nothing usable, the walk reports its own gaps, `getDirSizeFast` deleted (the Downloads insight now shares the du runner). **Deviation from upstream**: upstream already had `State` plumbing, so its consumers could render partials; ours (the TUI row handler and the JSON entry) would have shown *nothing* for a partially readable root, so both were widened to accept `size > 0` with an error. That is a display decision, not a port of upstream code, and `A-3`/`A-4` must replace it with a real marker (`B-2` for JSON). 5 new tests + 6 mutation probes, all caught. Overran because the consumer-side decision was not in the card text. |
| 2026-10-09 | `A-2a` | 60m | ~95m | `4cf39ec5` | Split `A-2` into `A-2a`/`A-2b` (the context threading is a separate signature sweep). Ported upstream `155a0cca`: du returns bytes **and** error on a non-zero exit; `scanFailures` helper (from `cf6165b2`, scanner side); measurement guards flipped to `size <= 0 && err != nil` at all three sites; skip-immediate-child accumulates and returns `(total, firstFailure)`. 4 new tests; mutation-checked all reverts — 3 of the 4 guard/behaviour mutations are caught, the 4th (`calculateDirSizeConcurrent`, scanner.go:902) is **not covered by any test**, upstream has the same gap; cover it in `A-3`. `make check` green, binaries smoke-tested on a real tree (`--json` totals unchanged: 5.28 GB / 28 entries). Went 35m over: the call-site guards needed per-site mutation probes, which the estimate did not count. |
| 2026-10-09 | `A-1` | 45m | ~30m | `4d7f5b66` | `scanState` + `measurementState()` added to `cmd/analyze/model.go` at the upstream position (keeps later cherry-picks aligned), tests in new `cmd/analyze/scan_state_test.go`. Not wired into `dirEntry`/`scanResult` — that is `A-3`. 3 mutations verified to fail the tests (`size >= 0`, partial→unavailable, reordering the iota so zero ≠ complete). `make check` green. |
| 2026-10-09 | `S0-1` | 0.5h | ~35m | `43872222` | Re-fetched: `upstream/main` unchanged (`2e8ea7f3`), 0 open PRs, newest analyze/status commit `1aad186b` already triaged → no new work. All 13 chain hashes + every other backlog hash verified as live ancestors. Corrected the commit count 151 → **140** (135 non-merge + 5 merges) and replaced the bogus "62 hunks" figure with a cherry-pick probe: 40 conflicted files / 82 markers, `50c89d9b` worst at 9/43. Re-sized `B-2` to 90–120m. Lesson: measure with `git cherry-pick -x` in a scratch worktree, never with `git apply --check -3` output. |
| 2026-10-09 | P3 batch (pre-card era) | — | ~2h | `9952bdde`, `e2994ff1` | 3 upstream commits + 4 tests; mutation-checked the tick guard. `make check` green, pty smoke OK. Not yet merged to `main`. |
| 2026-10-09 | docs re-audit | — | ~1h | `67583cad`, `3d011733` | Corrected the A-layer upstream mapping, re-derived counts for `V1.58.0`, split the backlog into cards. |

## Notes

- Verify each batch: `make check` (vet + tests), then run `./dua status` / `./dua analyze` from the repo. For TUI changes, smoke-test through a pty (`timeout 6 script -qec './dua analyze <path>' /dev/null`).
- Work is scheduled by card, not by batch (see the P0 card table for the slicing rules): one card ≤ 90 minutes, ends green, ends in one commit. If a card does not fit the session, revert and stop.
- Test procedure (avoid confusing the stable install): always invoke via the repo path `./dua` or `/root/opencode-stuffs/dua/dua`; never run bare `dua` (that hits the stable version in `~/.local/bin`). Discriminate by `dua version` output: dev build shows `v0.2.0-<n>-g<short>` + commit, stable shows the release tag.
- Record **upstream hash → dua commit** mappings in both this file and the AGENTS.md decision log. Logging only local hashes is what made the 2026-10-04 A-layer row unreadable.
- An upstream change under `mo-applets/` is not automatically a skip: port the equivalent into `cmd/analyze/` / `cmd/status/`, since the two trees diverged and the upstream path is not the porting target (see `1a1c7300` → `b5608780`).
- Full upstream has 151 analyze/status commits since the fork; only the tables above are Linux-relevant and non-Mole.
