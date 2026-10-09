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

Keep incomplete / partially-sized measurements visible across navigation, live scans, JSON, and caches. **Port as one unit** (13 commits; not cherry-pickable individually). Estimated ~20–35h for someone familiar with the codebase — that figure was 2–3× high: the decomposition below summed to ~11h of carded work, **all 15 cards are closed**, and what is left is not a card: merge the branch and push the `v0.3.0` tag (see **v0.3.0 — release decision, procedure and notes draft**); see the calibration note at the end of this section.

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
| `f5b4130c` | 1 | 1 | `B-3` | **two halves**: the guard *and* its test switching the scan to `scanCacheBypass`; without the policy switch the branch is never reached — see the `B-3` row |
| `50c89d9b` | **9** | **43** | `B-1`/`B-2` | the heaviest commit in the chain by a wide margin |
| `20ac485b` | 7 | 13 | `B-2` | 606 of its changed lines are tests |
| `53c4d362` | 6 | 7 | `C-1` | |
| `a4350259` | 4 | 4 | `C-1` | |
| `1cbaad5e` | 2 | 2 | `C-2` | 3 json.go hunks + 1 test; a 4th (`model.go` `MarshalText`) is **not portable** — see the `C-2` row |
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
| `S0-2` ☑ | Test fixture helper for unreadable dirs that does **not** rely on `chmod 000` + root skip (our CI runs as root, so upstream's fixtures self-skip) | none — `318ee925` turned out to carry **no** infra, it uses the same `os.Geteuid() == 0` skip | 1h | ✅ `runTestWithoutPrivileges` + `lockDirFromReader`; `TestPermissionFixtureDropsReadAccess` runs (not skips) under root CI and both directions are verified; the five existing permission tests now run as root and `skipIfRoot` is deleted | `fac63f15`, single commit |
| `A-1` ☑ | Add `scanState` type (`scanComplete`/`scanPartial`/…), `measurementState()`, unit tests. **Not wired into the model yet.** | `cf6165b2` (type part) | 45m | ✅ `go vet` clean, 4 tests pass (3 mutations checked), zero behavior change | done in `A-1` commit; A-3 will consume it |
| `A-2a` ☑ | Make `du` keep "size + error = partial" instead of dropping the size: runner semantics, `scanFailures`, the three measurement guards, and the skip-immediate-child accumulator | `155a0cca` | 60m | ✅ partial `du` bytes survive at the du boundary, at a folded scan and in live scan; nothing re-walks | done; unblocks A-3 |
| `A-2b` ☑ | Thread `context` through the measure helpers (`measureOverviewSize`, `getDirectoryLogicalSizeWithExclude`, `getDirectorySizeFromDuSkippingImmediateChild`, `measureInsightSize`, `measureOldDownloads`) and stop `measureOverviewSize` from re-walking on a partial du | `d04453f1` | 60m | ✅ overview keeps partial bytes with one measurement; cancellation propagates | done; see the deviation note in the work log |
| `A-3` ☑ | Carry the state into scan results and the model (`m.scanState`, per-entry state); keep `NeedsRefresh` semantics consistent | `cf6165b2`, `580b8eb0` (model part) | 90m | ✅ `dirEntry.State` / `scanResult.State` / `m.scanState` wired end-to-end; a partial scan keeps its bytes, the header marks the lower bound, and row states survive a cache round-trip; A-2b's interim `size > 0` widening replaced by a real marker in the TUI | done; `A-4`, `C-1`, `B-2` read this state |
| `A-4` ☑ | Carry partial coverage through live scan events | `13f068ac` | 45m | ✅ `readLiveScanInitialEntries` returns a `scanResult` and tracks its own coverage; `runLiveScan` keeps an unscannable target as an `scanUnavailable` row instead of dropping it, copies each target's state onto its entry, and reports the final state; `liveScanStartMsg.state` reaches `m.scanState` | done; `B-2` reads the result state |
| `B-1` ☑ | `historyEntry` gains state; bump `cacheSchemaVersion` 3 → 4 so stale caches are rejected | `50c89d9b` (cache part) | 45m | ✅ landed early inside `A-3`: `historyEntry.State` + schema v4 + a partial total is never written to disk. Only the release-note duty remains → moved into `C-3` | done; `A-3` raised the schema to **v4**; `B-2` raised it to **v5** for the overview coverage marker, and both bumps land in the same unreleased range, so they cost one re-scan, not two |
| `B-2` ☑ | Cache partial scans correctly: partial results are reusable, a complete scan may overwrite them, stale writes are rejected, changed sizes invalidate | `50c89d9b`, `20ac485b` | 90–120m | ✅ a permission-shaped partial survives navigation and relaunch without being re-scanned forever, and a temporary one is never reused: `scanResult.partial` is set per cause at every failure site (5 in the walk via `recordScanFailure`/`recordChildFailure`, 5 in the live scan), `persistable()` now asks “did a denial explain part of the gap?” instead of “was nothing else wrong?”, `shouldPersistSubdirCache` applies the same gate, `loadCachedSubdirResult` restores the stored `State` (it used to drop it, so every cached partial read back as *complete*), and `du`'s stderr is captured (`boundedSnippetWriter`, 64 KiB) so a `du` failure reads as a denial rather than a timeout. `cacheSchemaVersion` 4 → 5 (the overview store is checked against the same constant, and a pre-v5 snapshot can hold an unmarked lower bound). Overwrite/stale/size-change behaviour already existed (TTL + mtime + grace + schema in `loadCacheFromDisk`) and stays pinned by `TestCacheKeepsCoverageContract` | done; the reuse *policy* decision is written up below — it was the hazard this card flagged. New tests: `TestShouldPersistSubdirCacheKeepsOnlyReusablePartials`, `TestCachedPartialSubtreeReadsBackAsPartial`, `TestDuStderrNamesUnreadableDirectories`, `TestDuOnUnreadableSubtreeKeepsBytesAndNamesTheDenial` (runs under root CI on the `S0-2` fixture). Three mutations verified red, one per gate: drop the read-side `State` carry → `TestCachedPartialSubtreeReadsBackAsPartial`; drop `persistable()` from `shouldPersistSubdirCache` → the transient case in `TestShouldPersistSubdirCacheKeepsOnlyReusablePartials`; drop the `partial` basis from `persistable()` → both. New tests (8): `TestClassifyPermissionVsTransientFailure`, `TestCacheRejectsTransientPartial`, `TestNeedsRefreshFollowsFailureKind`, `TestShouldPersistSubdirCacheKeepsOnlyReusablePartials`, `TestCachedPartialSubtreeReadsBackAsPartial`, `TestOverviewMeasurementKeepsPartialFlag`, `TestDuStderrNamesUnreadableDirectories`, `TestDuOnUnreadableSubtreeKeepsBytesAndNamesTheDenial` (runs under root CI on the `S0-2` fixture) |
| `B-3` ☑ | Keep complete child caches after a partial parent refresh | `f5b4130c` | 45m (sized as probe + one fix) | child caches survive a partial refresh and a finished refresh still drops the entry — `TestRefreshOverUnreadableSubtreeKeepsItsMeasuredCache`, which runs under root CI and goes red when the guard is removed | ✅ ran ~55m. The re-derivation showed the card has **two halves**: the guard, and upstream switching its own test scan to `scanCacheBypass` — under the reuse policy the drop branch never runs, so a port that copies only the source line lands with a test that cannot fail. Confirmed with a throwaway probe that dua reaches the branch (`err == nil` + `state=partial`, and the entry file was deleted), so this is a real fix here rather than a no-op. Landed: `cachePolicy == scanCacheBypass && result.State == scanComplete` + `TestRefreshOverUnreadableSubtreeKeepsItsMeasuredCache`. Two fixture choices follow from the code: it sits two levels down because `R` calls `invalidateCacheTree`, which already clears the scanned directory and its **direct** children, so a nested subtree is the only depth a refresh can destroy; and it asserts the cache file's presence instead of byte counts, so it does not inherit the `skipIfBlockAccountingUnreliable` skip that makes the coverage tests above no-op on this filesystem. 4 mutation probes (reverted guard, inverted guard, never-drop, drop-for-unavailable) — all caught |
| `C-1` ☑ | TUI labels for partial/incomplete sizes (`measuredSizeLabel`, `scanSummary`) and pending-size column alignment. Both specified halves were already closed (`A-3` labels, fork-history pending rows), so the card ran as a re-derivation. Result: **not a no-op** — `53c4d362` has four parts, two landed via `A-3`, two were genuinely missing (a short reason for a failed measurement, and a header that survives that reason) | `53c4d362` (2 of 4 parts), `a4350259` (**not applicable**), `b77a48d7` (**test + README only** → belongs to `D-2`) | 30m → likely <15m | ✅ ran ~50m, over the box: the re-derivation was cheap, the two missing halves were real work. Now: a failed overview measurement renders `Partial size: Variables (var) (access denied)` — row name, not full path — via `measurementErrorReason`, and `writeOverviewStatus` keeps the header on one line (`TestOverviewStatusLineStaysInsideTheTerminal`); `duError` carries du's first stderr line as a display reason with `Unwrap` preserving classification (`TestDuErrorKeepsClassificationUnderUnwrap`) | `a4350259` is dua's filter-space key, and the multi-select/Trash surface was removed at the fork. `b77a48d7` changes no source file — but see `D-2`: its test asserts `document.ScanStatus`, so it needs `C-2` first |
| `C-2` ☑ | JSON contract: `scan_status` on the document and on entries + tests | `1cbaad5e` | 45m | ✅ ran ~70m, over the box: `dua analyze --json` now reports `scan_status` at both levels, and unmeasurable overview rows stay visible | two deviations from upstream were needed (below); additive wire field only |
| `C-3` ☑ | Docs: `README.md` + `README.zh-TW.md` in sync, version decision (**minor bump → v0.3.0**, behavior change), release note about the one-time cache invalidation | — | 45m | ✅ ran ~40m, inside the box: both READMEs document the TUI markers (`+`, `--`) that the chain shipped but never described, plus the one-time cache re-scan under `dua update`; structural parity verified by parser (12/12 headings, bullets/code fences/paragraphs equal). Version decision, tag procedure and the release-notes draft are in the release section below | the chain's only docs card; the acceptance check had to be made mechanical, because "in sync" between two translations cannot be eyeballed |
| `D-1` ☑ | Real-machine A/B: run the same path before/after, diff the totals and the new markers | — | 30m | ✅ ran ~30m, done: the measured diff is written up in the **D-1 measured A/B** section below — totals and file counts byte-identical on `/etc`, `/var`, `/var/lib`, `/usr`, `/usr/share`; the only change is that lower bounds are now labelled (`Total: 739.6 MB+`, `--` instead of a percentage, `/var/lib/nginx` reads `partial` instead of looking empty); pre-coverage caches are rejected once and downgrading is safe; no measurable perf cost. | no code; the write-up is the deliverable |
| `D-2` ☑ | Port the navigation-regression test suite on top of the `S0-2` fixture. When porting, replace upstream's `os.Geteuid() == 0 { t.Skip… }` with `runTestWithoutPrivileges(t)`. **Unblocked: `C-2` landed**, so `b77a48d7`'s `TestPartialScanCoverageSurvivesEntryLimit` can assert `document.ScanStatus` / `entry.ScanStatus` as written — but its *fixture expectation* cannot be copied: it expects the unreadable directory to report size 0 + `unavailable`, while a dua walk can stat the directory itself and reports its own blocks + `partial`. Port the assertions about coverage, not the byte counts. | `318ee925`, `b77a48d7` (source half of the latter is README only) | 45m | ✅ ran ~55m. Both tests ported and both **run** under root CI through `runTestWithoutPrivileges`. The card's fixture warning did not apply to the path it names: probed first, and the directory-listing paths (live scan, `performDirectoryScanForJSON`) report the unreadable directory as `size 0 + unavailable`, exactly as upstream wrote it — the own-blocks shape belongs to the *overview measurement* path (`C-2`). So upstream's assertions are kept verbatim; only the byte arithmetic changed (`st_blocks` is unreliable here, so the expected aggregate is summed with `getActualFileSize`). One real divergence found and pinned: upstream's `goBack` re-scans any partial parent, dua treats a permission denial as durable (`isPermissionFailure`), so returning reuses the measured floor and only `R` re-reads. README half ported in both docs. Eight mutations probed: seven red; see the notes below for the one that stayed green. | test-only |

Porting notes learned while doing `A-1` (cheap risk reductions for `A-3`/`B-2`):

- Upstream's coverage tests call `scanPathConcurrentAllEntries`, `calculateDirSizeFast`, `saveCacheToDisk`, `loadCacheFromDisk`, `writeFileWithSize` and `skipIfRoot` — **all of these already exist in `cmd/analyze/` with the same names**, so the tests in `cf6165b2`/`20ac485b` port almost verbatim once the `State` fields exist. The blocker is the fixture, not the API surface → that is exactly why `S0-2` gates `D-2`.
- Upstream's `scanState` uses `scanComplete = iota` deliberately: cache files written before coverage tracking decode `State` as 0, so the zero value must mean complete. `TestScanStateZeroValueMeansComplete` pins this; `B-2` (cache semantics) must not re-order those constants.
- `scan_state_test.go` exists but nothing calls `measurementState` yet — `go vet` does not flag unused package-level functions, and CI runs vet + tests only. If `A-3` slips, this type stays dead code on the branch; that is the accepted cost of the slicing rule. (Closed by `A-3`: `measurementState` is now called from the scanner, live scan and the overview row handler.)

Porting notes learned while doing `C-2` (the two things that made a 45m card take ~70m):

- **Upstream's json.go hunks assume its own cache API.** Its overview read returns bytes only, so `item.State = measurementState(size, err)` after a cache hit is free. dua's `loadOverviewCachedMeasurement` (since `B-2`) returns the stored *state*, and deriving a new one from `(bytes, no error)` promotes a cached lower bound to measured. The port must reuse `cachedState`. This is the same read-side bug class `B-2` removed in `loadCachedSubdirResult`, arriving through a different door — the pattern to check on every path that reads a measurement back.
- **A display-only type is not display-only here.** Upstream publishes the word by giving `scanState` `MarshalText`. dua's `scanState` is also the on-disk cache type (`cacheEntry.State`, `dirEntry.State` inside cached entry lists), so that hunk rewrites the storage format (`"State":1` → `"State":"partial"`) and, with no matching `UnmarshalText`, *every* cache read then fails to decode — a silently dead cache, no test failure, no error. Probed standalone before choosing. Keep the string at the JSON structs and the stored form numeric: byte-identical wire, storage untouched, and no cache-schema bump needed.
- **Fixtures encode the other project's semantics — and which *path* builds the row decides what the row means.** The upstream form of the permission test asserts the unreadable directory appears as `size 0 / unavailable`. In the **overview measurement** path (`C-2`) a dua row does carry the directory's own blocks and `partial`, because that path measures the directory itself. In the **directory listing** path (`D-2`, live scan and `performDirectoryScanForJSON`) the row is built from the listing that failed, so it is `0 + unavailable` and upstream's expectation holds verbatim. Verified by probe in both, not by reasoning from the diff. Rule for test ports: port the *invariant*, and measure the expected values instead of importing them from either project's other path.

Porting notes learned while doing `D-2` (what a test-only card still costs):

- **A test-only card still has a contract to re-derive.** `b77a48d7` and `318ee925` change no source file, so the card looked like a copy job. It was two decisions: which upstream expectations describe dua (the entry-cap and JSON-listing halves do, verbatim — including that the TUI total and the JSON total are the same number), and which encode a policy dua does not have (upstream re-scans a partial parent on `goBack`; dua does not, on purpose: a permission denial is `durable`, so a returning user is shown the floor already measured and `R` is the retry). A port that copied the upstream body would have failed here, and a port that only "adjusted" it without naming why would have hidden a real product difference.
- **Exact byte arithmetic cannot survive this container.** Upstream writes `msg.result.TotalSize != int64(maxEntries*4096)`; files here report 512 bytes for a 4 KiB write (`st_blocks` under-reports, see `skipIfBlockAccountingUnreliable`). The expected aggregate is summed from the filesystem with `getActualFileSize`, which keeps the assertion exact on every machine instead of weakening it to `> 0`.
- **A live-scan fixture must stay under the cache gate.** Growing the tree past `subdirCacheMinFiles` made the test fail on `TempDir RemoveAll cleanup: … /home/.cache/dua/analyzer: directory not empty`: the live scan publishes its result while sibling goroutines are still writing cache files, so the scan's last writes race the cleanup. That is why `TestPartialNavigationRefreshRecoversCoverage` keeps a small fixture, and why "`R` does not serve a stored partial" stays owned by the synchronous cache tests (`TestCachedPartialSubtreeReadsBackAsPartial`, `TestRefreshOverUnreadableSubtreeKeepsItsMeasuredCache`). Same trap applies to any future card that drives the live scan and wants a cached subtree.
- **Two guards, so one mutation cannot fail the test.** Dropping `invalidateCacheTree` from `R`, *or* switching `R` to the reuse policy, leaves the recovery test green — either guard alone is enough to re-read. Removing both is still green at this fixture size, because nothing was stored to reuse. Recorded rather than papered over: it says what the test does *not* pin, which is the input the next cache card needs.

Porting notes learned while doing `A-3` (cheap risk reductions for `A-4`/`B-2`/`D-2`):

- **Permission fixtures now run under root**: `runTestWithoutPrivileges(t)` (`cmd/analyze/test_helpers_test.go`, from `S0-2`) re-executes the test binary under `unshare -U`, where the uid becomes the overflow uid and DAC checks apply; the parent relays the child's failure and output. Use it as the first statement of any test that needs a directory to be unreadable, with `lockDirFromReader(t, path)` for the mode change. `skipIfRoot` no longer exists. In `A-3` the manual version of this (build with `go test -c`, then run the binary under `unshare -U`) caught a real bug — an `A-2b` test that asserted exact byte arithmetic and had never actually run here — which is why `S0-2` made it a routine call rather than a trick.
- Display decisions in this area are row-level, not list-level: the percent column and the zero-row filter key off the *row's* state, and only the header total carries the lower-bound marker. Blanketing the whole list because one root is unreadable reads as a broken screen. Keep `A-4`/`C-1`/`C-2` consistent with that split.
- Still uncovered by any test, as root or non-root (known gaps, do not add a seam to `scanner.go` just to reach them): `scanSubdirWithCache`'s sizing fallback state, the two `result.State != scanComplete → parent incomplete` propagations in `scanPathConcurrentWithLimiter`, and the three `child.Info()` error sites.

Porting notes learned while doing `B-2` (the product decision this card warned about, and the two deviations from upstream):

- **The reuse rule.** A partial is reusable when a permission denial explains part of its gap (`scanResult.partial`), *not* when nothing else went wrong. A tree that lost bytes to both a `chmod 0` directory and a timeout therefore keeps its measured floor: that floor is the same number next time, and the temporary half is what `R` re-measures. This is upstream's choice in `50c89d9b` and dua takes it, because the alternative — one locked subdirectory anywhere in `/usr` disabling caching for that tree forever — is the case that matters on Linux. It is only safe because two other things exist: the coverage marker (so a cached lower bound still *reads* as partial) and `R` (an explicit retry).
- **Cacheability and re-measurement are separate questions.** `historyEntry.NeedsRefresh` is `needsRefresh || transientFailure || !persistable()`: a mixed-cause partial is *both* stored (fast first paint) *and* refreshed (its recoverable gap gets another look). Do not merge those two conditions back into one — that coupling is what made `TestNeedsRefreshFollowsFailureKind` contradict the cache gate mid-session.
- **Two gates, and the read gate held the real bug.** The write gate already existed in `saveCacheToDiskWithOptions` (`!persistable()`); what was missing was that `loadCachedSubdirResult` rebuilt the result without `State`, so any cached partial was served as a *complete* measurement — exactly the overstatement `A-2b` refused to widen for. Removing the read-side carry reproduces `cached partial read back as complete`; removing the `partial` basis of `persistable()` reproduces `partial with a permission cause and a timeout → false`. Both directions verified by mutation, since neither was otherwise observable.
- **`du`'s failure kind lives only in its stderr.** The process just exits non-zero; `du: cannot read directory 'x': Permission denied` is the only signal. Without capturing it (bounded at 64 KiB, then joined onto the error as `fs.ErrPermission` so the existing classifier sees it), every large tree — which is measured with `du`, not the walk — classified as transient, and the feature would have been dead in practice while passing its unit tests. `TestDuOnUnreadableSubtreeKeepsBytesAndNamesTheDenial` is a test that could not exist before `S0-2`.
- **Deviation 1: no `dirScanCacheEntry.Partial`.** Upstream introduces a separate on-disk subtree type with a `Partial` string; dua's subtree cache *is* `cacheEntry`, which has carried `State` since `A-3`. Reuse is decided from `State` plus the write gate, so a second marker would be a second source of truth. `cacheSchemaVersion` does go to **5**, and only because of the overview store: `loadOverviewSnapshotStore` compares `SchemaVersion` against the same constant, and a pre-B-2 snapshot can hold a lower bound with nothing marking it, which would read back as an exact figure. The subtree format itself did not change (a v4 file can only contain complete totals, so it is still interpretable), so the bump costs one extra re-scan that `A-3`'s v4 bump already charged in the same unreleased release — which is also why dua keeps one shared counter instead of versioning the two stores separately.
- **Deviation 2: no test-mode cache invalidation from `20ac485b`.** Upstream's `cachePolicyFor("test")` forces a fresh measurement inside tests; dua has no `test` cache policy, and its tests deliberately assert the opposite (child caches survive, pinned since `A-1`). Porting it would have undone a shipped behaviour for a test-harness convenience.
- Still uncovered, unchanged from the `A-3` list: `scanSubdirWithCache`'s sizing fallback and the three `child.Info()` sites have no direct test. `B-3` now exercises the child-cache side of that function; the sizing fallback and the `Info()` sites still do not.

Suggested session plan: `B-2` ✅ landed inside its box (~65m against 90–120m). `C-1` ✅ landed over its box (~50m against a 30m re-derivation) — the re-derivation itself was ~15m, but it found two real halves in `53c4d362`, which is the opposite failure from a stale-hash no-op and the same lesson read backwards: a re-derivation card is a *probe*, not a cheap card. `C-2` ✅ landed over its box (~70m against 45m): the port itself was ~25m, and the rest went into two decisions the card text did not anticipate (the cache-hit state, and whether to copy upstream's `MarshalText`) — both needed a standalone probe before choosing, which is the same unmodelled cost recorded for `A-2a`/`A-2b`. `B-3` ✅ landed just over its box (~55m against 45m): the source change is one guard, and the rest went into deriving which path can reach that branch at all — the same probe-first cost recorded for `C-1`/`C-2`. `D-2` ✅ landed just over its box (~55m against 45m), and it is the clearest case yet that a test-only card still carries a contract: the two commits touch no source file, and the work went into measuring which upstream expectations dua shares (all of the entry-cap/JSON-listing ones, verbatim) versus which encode a refresh policy dua rejects (`goBack` re-scans any partial; dua treats a permission denial as durable and lets `R` retry). `D-1` ✅ ran inside its box (~30m): it is the only card in the chain with no code, and it produced the release-note evidence (measured A/B below) plus one follow-up (`F-1`), found by `D-1` and closed after `C-3` corrected its premise. `C-3` ✅ landed inside its box (~40m against 45m): it was a docs card, and the work was finding what was *undocumented* rather than writing more — the TUI `+` / `--` markers had shipped in `A-3`/`C-1` with no user-facing explanation, and the v3→v5 cache bump had no upgrade note. What is left is not a card: merge `port/a-layer-upstream` to `main` (0 behind, so a fast-forward) and push the `v0.3.0` tag — procedure, pitfalls and the notes draft in the release section below. For scale, the rest of the backlog is P1 1.5h, P2 3h, P4 6.3h, P5 5.5h, plus one open follow-up (`F-1`, re-scoped to 10m). Closed so far: `S0-1`, `S0-2`, `A-1`, `A-2a`, `A-2b`, `A-3`, `A-4`, `B-1`, `B-2`, `B-3`, `C-1`, `C-2`, `C-3`, `D-1`, `D-2` — the whole P0 batch. Estimates were re-sized after `A-2a` ran 60m→~95m and `A-2b` 60m→~85m: the unmodelled cost in both was the per-site mutation probing needed to prove a behaviour change is pinned, and in `A-2b` also a consumer-side decision the card text did not anticipate. `A-3` stayed in its box only because that display decision had already been written down in the `A-2b` log; `B-2` carries the same hazard (partial-cache reuse *policy* is a product decision, not a port) — write that decision into the card text before starting it. Calibration as of `C-3`, over the twelve closed cards that recorded both numbers (`A-1`, `A-2a`, `A-2b`, `A-3`, `A-4`, `B-2`, `B-3`, `C-1`, `C-2`, `C-3`, `D-1`, `D-2`): **660m planned, 710m actual (+8% in aggregate)** — while the per-card spread is 0.54× (`B-2`, 120m → 65m) to 1.67× (`C-1`, 30m → 50m). Method, so the number is recomputable: planned = each card's own Est column, a range taken at its upper bound (`B-2` 120m) and a re-sized card counted at its **original** figure (`C-1` 30m, not the 15m its text was amended to after the re-derivation); actual = the last figure in that card's work-log row. `B-1` has no log row (it landed inside `A-3`) and `S0-1`/`S0-2` never recorded an estimate, so they are excluded; an earlier version of this paragraph quoted 660m/720m without stating the method and could not be reproduced from the log — that is why the method is now part of the claim. Aggregate totals are therefore usable and a single card is not: plan ±50% per card, and trust the sum. The cards that went over (`C-1`, `A-2a`, `C-2`, `A-2b`) all paid for a *decision* or a *probe* the card text did not name; the cards that came in under (`B-2`, `A-1`, `S0-2`) all had that decision written down in the card text before starting, or were mechanical. So the sizing rule for `C-3`/`D-1` is not "add 50%" — it is "name the decision in the card text, or size the card as a probe". A second calibration point, at the other end of the plan: the chain was originally estimated at **20–35h** for someone familiar with the codebase; the twelve cards that implemented it summed to 11h of carded work and actually took 12h. The original figure was 2–3× high, and the difference was not speed — it was that the backlog was decomposed against the real tree (via `S0-1`) instead of against upstream's diff.

Hard ordering (everything else is free-choice):

1. `A-1` before `A-2a`/`A-3` — the type has to exist before anything can carry it.
2. `A-3` before `A-4` — live-scan events read the model state. `A-2a` before `A-3` as well: `A-3`'s acceptance criterion (a partial scan still renders correct totals) needs a scanner that actually produces partials, which is what `A-2a` added. (A earlier note in this session suggested doing `A-4` right after `A-1` — that was wrong on both counts.)
3. `A-2a` before `B-2` — the cache cannot persist a state the scanner never produces.
4. `B-1` and `B-2` in the same release — a half-bumped cache schema misreads old entries. `A-3` took the schema to v4 and `B-2` to v5, both inside this same unreleased range, so pre-coverage caches are rejected once rather than twice: anyone building this branch pays the one-time re-scan now, and `C-3` must say so. `B-2` changed *what* gets written and how a stored partial reads back.
5. `S0-2` before `D-2` — the navigation tests need a fixture that is not skipped under root CI.
6. `A-*` + `B-*` + `C-*` all present before tagging; `C-3` was the last card of the release — ✅ done, so the release gate is now the merge + tag procedure.
7. Stop-anywhere cards (safe to close a session on): none left — the chain is closed. The remaining step is the merge + tag procedure, which is safe to leave unrun: the branch builds and passes `make check` as it stands, and nothing in it half-works.

### D-1 measured A/B (real machine, 2026-10-09) — the text to paste into the PR and the release notes

**Method.** Two builds of the same machine: **before** = `43872222` (the tree immediately before `A-1`), **after** = branch head. Both built with `make build`, run as uid 65534 under `unshare -U` so the permission denials are real, each cold run with a fresh `$HOME` (empty cache) and a warm run reusing it. Paths are the machine's own — `/etc`, `/var`, `/var/lib`, `/usr`, `/usr/share` — not fixtures.

**The numbers did not move. What was invisible is now marked.**

| path | build | cold | `total_size` | files | rows | document `scan_status` |
|---|---|---|---|---|---|---|
| `/etc` (fully readable) | before | 0.05s | 1,394,335 | 796 | 194 | *no field* |
| `/etc` | after | 0.07s | **1,394,335** | **796** | **194** | `complete` |
| `/var` (permission gaps) | before | 0.14s | 739,638,981 | 11,035 | 13 | *no field* |
| `/var` | after | 0.12s | **739,638,981** | **11,035** | **13** | `partial` |
| `/var/lib` | before | — | 629,873,134 | 10,397 | 38 | *no field* |
| `/var/lib` | after | — | **629,873,134** | **10,397** | **38** | `partial` on 5 rows |

Byte-identical totals and file counts on every path, including `/usr` (1,715,417,822) and `/usr/share` (384,755,746). Nothing in the chain changed a measurement; it changed what the measurement claims.

**TUI, same machine, same pty, `/var`:**

```
before:  Analyze Disk  /var  |  Total: 739.6 MB          ○  1. ████…  85.2%  |  📁 lib   629.9 MB
after:   Analyze Disk  /var  |  Total: 739.6 MB+         ▶ ○  1. ████…    --  |  📁 lib   629.9 MB+
```

The `+` and the suppressed percentage are the whole product: a share *of a lower bound* is not a share. The bars still scale proportionally, so the ranking is unchanged — only the false precision goes away. Rows that are complete still show their percentage (`log 22.8 MB 3.1%`).

**The two examples to put in the release notes** (real paths, no fixture):
- `/var/lib/nginx` measures **0 B in both builds**. Before, that was indistinguishable from an empty directory; after, the row is `partial` — there are files in there, dua cannot read them.
- `/var/lib/docker` keeps its **486 MB** measured floor and is marked `partial`, instead of a silent under-count presented as a total.

**Cache store across the schema bump** (this is the release-note claim, measured rather than asserted): before writes 58 store entries for `/var`; **after over that store keeps 0 and rewrites everything**, no error, same totals → pre-coverage caches are rejected exactly once, as `cacheSchemaVersion` 3 → 5 intends. **After over its own store: 0 written, 54 kept** → the partial floor is reused across runs, which is `B-2`'s policy seen from outside. The old build over the new store also just re-scans, so **downgrading is safe too**. Fresh-`HOME` entry counts match build for build (58/58 `/var`, 2/2 `/etc`, 50/50 `/var/lib`, 4/4 `/usr/share/doc`); one run over a foreign store ended at 54 rather than 58, which was not chased — totals were unaffected either way.

**Perf** (the chain threads `ctx` through five measure helpers and records state at every failure site): median cold over three fresh-cache runs — `/usr`: **385ms before / 387ms after**; `/usr/share`: **182ms / 173ms**. No measurable cost.

**Two things the run surfaced that are not chain bugs:**
1. **Withdrawn as written.** The A/B reported `total_files: -1` for a path with no entries (`/boot`) while `scan_status` was `complete`. Re-checked against the raw document: dua emits **no `total_files` key at all** in that case (`json.go` tags the field `omitempty`, and the count is a plain non-negative counter — no `-1` exists anywhere in the scan path). The `-1` was the probe script's own default (`d.get("total_files", -1)`), which the A/B printed as if it were observed. What is real is smaller: because of `omitempty`, a genuinely measured `0` and "not reported" are the same absent key. Still a contract wart in the field set this release documents, so `F-1` is re-scoped to that; kept because it is cheap and the field set is the release's subject.
2. The `/var` `lib` row size equals the `/var/lib` document total (629,873,134) — the two paths agree. Worth stating because it is the first cross-check a reviewer will run.

## v0.3.0 — release decision, procedure, and notes draft

Closed by `C-3` (2026-10-09). This is the only P0 work left, and it is procedure, not code.

### Why a minor bump

`AGENTS.md` → Versioning: features or behavior changes bump the minor; patch is for fixes, docs, and chores. This range changes observable behavior in three user-visible ways — JSON gains `scan_status` at two levels, the interactive view labels lower bounds (`+`, `--`), and the cache schema goes 3 → 5 — so **v0.3.0**, not v0.2.1. Nothing here is a stability promise, so v1.0.0 stays unused.

### The version string needs no code change

`Makefile` derives `main.version` from `git describe --tags --always --dirty` and `main.commit` from `git rev-parse --short HEAD`; `dua version`, the TUI footer, and the JSON `version`/`commit` fields all read those two. **There is no version constant to edit — the tag is the version.** Confirmed from the branch build: it reports `v0.2.0-<n>-g<sha>-dirty` because v0.2.0 is the newest tag, and will report `v0.3.0` as soon as that tag exists. A future agent should not go looking for a `version.go`.

### Before tagging: nothing

`F-1` (10m) is **closed** (2026-10-09), so no code item gates the tag. It removed `omitempty` from `total_files`: a scan that counted zero files now says `total_files: 0` instead of dropping the key — the same false silence this chain removed for sizes, sitting in the field next to `total_size` (which never had `omitempty`). Verified on the shipped binary: an empty directory yields `total_files: 0` with `scan_status: complete`, and `/etc` still yields 796 files / 1,394,335 B — identical to the `D-1` measurement, so the change is additive only.

### Procedure — executed 2026-10-09

1. `make check` on the branch head → green (`cmd/analyze`, `cmd/status`, `internal/units`), plus `make build`.
2. `git checkout main && git merge --ff-only port/a-layer-upstream` → fast-forward, 40 commits, `main` = `e71986ea`. Verified 0 behind first, so no merge commit was needed.
3. `git push origin main`, then polled `test.yml` on `main` until `completed / success` (3 polls, ~1 min). This was the first CI pass over these commits — checked, not assumed: no PR exists for the branch and `test.yml` triggers on `Linux-DEV`/`main`/PRs only, so a topic branch never gets CI by itself. Local `make check` is the same command set on a different toolchain image, not a substitute.
4. `git tag -a v0.3.0 -m "dua v0.3.0" && git push origin v0.3.0` → annotated tag (same shape as `v0.2.0`), tag object `47c9d208` → target `e71986ea`. `release.yml` created release id 407826231 with the workflow's fixed body plus the auto `**Full Changelog**: .../compare/v0.2.0...v0.3.0` link, and 4 assets (linux amd64/arm64 tarball + `.sha256`).
5. **Still manual:** paste the notes below into the release body. The workflow body is version-agnostic on purpose, so version-specific text can only be added after the release exists — that is how v0.2.0's darwin section got there. Chosen approach is *manual per release* (no automation): auto-generated GitHub notes list merged PRs, and this repo pushes to branches directly, so auto-notes would have produced an empty list.
6. **Still manual:** dispatch `upload-darwin.yml` with `version=v0.3.0`, `confirm_tag=v0.3.0`, `ref=main`. Until it runs, the macOS install line published in both READMEs 404s — see the pitfall below.

Four pitfalls, each checked rather than assumed:

- This repo carries inherited upstream tags (`V1.58.0`, `v1.30.0-windows`, …). `release.yml` triggers on lowercase `v*`, so pushing upstream tags to origin would fire release builds for upstream versions. Never bulk-push upstream tags.
- `releases/latest` — the install URL in both READMEs and inside `dua update` — resolves among **published releases**, not tags, so those upstream tags do not shadow dua's tarballs.
- `releases/latest/download/<asset>` resolves **inside the newest release only**; there is no fallback to an older release that had the asset. Measured: `latest/download/dua-darwin-arm64.tar.gz` → 200 while v0.2.0 is newest, but a name absent from the newest release redirects to `releases/download/v0.2.0/<name>` and 404s. So a release without darwin assets **breaks the macOS install command on `main`** until `upload-darwin` runs for it. This is why `e71986ea` parameterized that workflow: it had v0.2.0 hard-coded in the upload guard, the `gh release` commands and the notes rebuild, so it could only ever serve v0.2.0.
- `main` was 10 commits ahead of `v0.2.0` before this merge (the darwin/platform work: `eee2f39f`…`b56f7561`), and those were never released. So the published range `v0.2.0..v0.3.0` is **50 commits**, not the 40 in the merge, and the release notes must cover that older work too. A release notes list derived only from "what this branch did" would have been incomplete.

### Release record (as published)

Verified against the **downloaded artifacts**, not the local build: both tarballs match their published `.sha256`; `dua version` from the linux/amd64 tarball reports `v0.3.0` + commit `e71986ea`; `dua status --json` and `dua analyze --json` carry `"version": "v0.3.0"`; `/etc` → `total_files: 796`, `total_size: 1394335` (byte-identical to the `D-1` baseline, so `F-1` is additive); an empty directory → `total_files: 0`. Checksums: `664319a3…4e46` amd64, `f379555b…5e5dc` arm64.

The body GitHub generated is the template + compare link only; the human notes below are what to paste.

### Release notes draft (paste into the release body after tagging)

> **dua v0.3.0 — measurements now say what they covered.**
>
> - `dua analyze --json` reports `scan_status` (`complete` / `partial` / `unavailable`) on the document and on every entry, so a `total_size` that skipped an unreadable subtree reads as a lower bound instead of a total.
> - The interactive view says the same thing: a lower-bound size carries a `+` (`739.6 MB+`), and a row that could not be fully measured shows `--` where its share would be — a percentage of a lower bound is not a percentage. Rows that were measured fully keep their exact numbers and shares.
> - A measurement that failed now explains itself: `Partial size: Variables (var) (access denied)`, with `du`'s first stderr line kept as the reason.
> - `--json` lists every entry the scan saw; the interactive view keeps the 30 largest. An entry dua could not measure may sit outside that list while the total still reads `partial`.
> - `--json` always reports `total_files` now. It used to disappear when the count was zero, which made "this tree is empty" indistinguishable from "no count was reported".
>
> **The numbers did not move.** Measured against the pre-release build on the same machine, unprivileged: `/etc` 1,394,335 B / 796 files, `/var` 739,638,981 B / 11,035 files, `/var/lib` 629,873,134 B, `/usr` 1,715,417,822 B, `/usr/share` 384,755,746 B — byte-identical totals and file counts. What changed is what those numbers claim. No measurable cost either (`/usr` 385ms → 387ms).
>
> **Upgrading:** the `analyze` cache format went from v3 to v5 in this range, so the first scan of a tree after updating re-reads it — once; later scans are served from the new cache again. Downgrading stays safe: an older binary refuses the newer cache instead of misreading it.

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
| `d534c30b` | animate pending overview rows | **Already in the tree** — its twin on the history line upstream actually publishes is `53383db2`, an ancestor of the fork base. `d534c30b` itself is on an abandoned line. Verified 2026-10-09: nothing to port. |
| `8a2b84b1` | keep braille spinner frames UTF-8 safe | **Skip** — it re-encodes `lib/core/ui.sh` (bash), which dua does not have; the frames live in Go. |
| `99a9471d` | align pending overview sizes with the numeric column | **Already in the tree** — twin `1cef46a9`, also an ancestor of the fork base. Its `--` placeholder was superseded by `53383db2`'s animated `scanning` row, which dua has; `TestOverviewPendingSizeUsesScanningSpinner` pins it. Nothing to port. |

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

### Follow-ups found while measuring (not upstream ports)

| Card | Origin | What | Est. | Notes |
|---|---|---|---|---|
| `F-1` ☑ | found by `D-1`, **premise corrected by `C-3`** | ~~`total_files: -1`~~ — that `-1` came from the probe script's default; the shipped binary omits the key entirely. The real wart: `TotalFiles` was tagged `json:"total_files,omitempty"`, so a measured count of `0` was indistinguishable from "the field is not reported". | 10m | ✅ ran ~20m: `omitempty` removed, so the count is always present (`/tmp/f1empty` → `total_files: 0`, `scan_status: complete`), and `jsonDocumentProbe.TotalFiles` became a `*int64` so the test can tell an absent key from a zero. Mutation-checked: putting `omitempty` back fails the new test on the exact assertion. Cross-check against `D-1`: `/etc` still reports 796 files / 1,394,335 B — the change is additive, no measurement moved Why it was worth doing before the tag: was worth doing before the tag because nothing in the repo reads the field today (no test, no doc referenced it), so the fix is free now and a breaking change after v0.3.0. Note `total_size` never had `omitempty` — the document already always declared one number and not the other|

## SKIP — Mole-specific / macOS / cleanup (not dua scope)

dua is read-only and Linux-first. Do NOT port these:

| Commit | Message | Why skip |
|--------|---------|----------|
| `9235bf7e` | fix(analyze): move to Trash via trash(8) | deletion surface |
| `f899cbf8` | fix(analyze): bypass nested caches on manual refresh | cleanup |
| `e8968b4b` | List uv cache in 'mole analyze' | mole-specific cache |
| `0b174ffa` | feat: recognize CACHEDIR.TAG cache markers | mole-specific cache |
| `794d2233` / `7f94ad35` | battery health | macOS-only |

[Showing lines 1-295 of 337 (50.0KB limit). Use offset=296 to continue.]