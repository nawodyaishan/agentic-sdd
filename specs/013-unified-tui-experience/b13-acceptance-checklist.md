# Feature 013 Batch B13: Acceptance Checklist & Regression Evidence

## 1. Status and Scope

- **Batch**: B13 (preparation lane for T35, T36, T37).
- **Working tree**: published `main` through `c5fe4aa`/`6a249d0`/`6c84266`; the selectors below were re-run on 2026-11-11 against the integrated tree. These are automated checks, not a substitute for the owner's observations.
- **Role of this lane**: automated evidence plus a record of owner-confirmed human checks. This file does not change the combined approval in [spec.md §Approval](spec.md#approval) and is not an independent platform/study transcript.
- **Boundaries**: no release/tag/tap publication, no real-home write, no live network/browser call, and no change to B8/B9/B11/B12-owned semantics outside the review fixes recorded in §5.
- **Preparedness**: T35 fixtures and T36 selectors are runnable. The owner confirmed on 2026-11-11 that the T37 walkthrough, novice study, visual review and platform checks were completed; granular participant/platform records were not supplied in this session.

---

## 2. T35: Responsive, State, and Content Fixtures

Suites: `experiments/tui/responsive_t35_test.go` plus the referenced focused tests (package `tui`).

### 2.1 Geometry and Review Sizes

- Layout breakpoint widths: `40`, `60`, `79`, `80`, `109`, `110`; tier edges `100` and `130`.
- Heights: `8`, `12`, `16`, `24`.
- Review samples: `140×40`, `100×30`, `80×24`, `60×20`.
- Tiny terminal: widths `<40` and heights `<8` render `"Terminal too small. Resize to at least 40x8."` with a visible quit hint and no panic.
- Evidence: `TestT35BreakpointsAndReviewGeometries`, `TestT35TinyTerminalSafety`.

### 2.2 Lifecycle and Inventory States

Covered states: fresh (missing), partial (mixed), modified (`differs from source`), current (matching), offline (checks disabled), refused (unavailable), failure notice, and empty catalog.

- Evidence: `TestT35StateCoverage`; Home content assertions use `getContentLines` sentinels rather than only line counts.
- Offline state asserts `upd:disabled` in `SessionPreferences`; no connectivity is inferred.

### 2.3 Unicode, Long Paths, Footer, Focus, and Confirmation

- **Content reachability**: `TestT35UnicodeAndLongPaths` now collects every page, requires the `REACHABILITY-SENTINEL` caveat marker and the `Advanced files:` section to appear, and checks the raw CJK/em-dash data in Unicode mode and the reversible `\u65e5`/`\u2014` escapes in ASCII mode. Bounds checks alone are not treated as reachability.
- **Footer and last item**: `TestT35FooterLastItemAndFocus`.
- **Confirmation fit**: `TestT35ConfirmationFitAndFallback` covers centered and full-screen fallback across the approved sizes, with `y confirm`/`y restore` visible.
- **Active-confirmation resize**: `TestT35ConfirmationResizeKeepsSafeNo` resizes an already-open install and restore confirmation through `140×40`, `100×30`, `80×24`, `60×20`, and `40×8`, asserts the token/stage are unchanged, and asserts `Enter` never emits a write. `TestConfirmationBelowUsableGeometryCannotSubmit` (lane 1 safety coverage) is the stricter below-`40×8` case and is referenced, not duplicated weakly.
- **Representative snapshots**: `TestT35RepresentativeScreenSnapshots` asserts structural snippets and paged Home content without depending on a golden image.

---

## 3. T36: Acceptance Criteria Matrix

All nested commands below were executed from `experiments/tui` with `-count=1`. `PASS` means the selector actually matched at least one current test and the package reported `ok`. A no-match selector is **not** pass evidence.

| ID | Observable contract | Executed selector | Result (2026-11-11) | Notes |
|---|---|---|---|---|
| **AC-01** | Four destinations, consistent focus/key/footer behavior, no duplicate child help | `go test -run 'TestT18BreadcrumbDerivation|TestT18StageAccurateFooter|TestT35FooterLastItemAndFocus' -v -count=1 .` | **PASS** (3 tests) | Automated evidence only; human review pending. |
| **AC-02** | 40/60/79/80/109/110 columns and short heights; resize preserves selection and consent safety | `go test -run 'TestT18BoundsAtAllBreakpoints|TestT35BreakpointsAndReviewGeometries|TestB2CellBoundsAtEveryBreakpoint' -v -count=1 .` | **PASS** (3 tests) | Geometry combinations and bounds. |
| **AC-03** | Dynamic counts, kind/purpose filters, live search, complete metadata, truthful statuses, profile routing | `go test -run 'TestB2|TestCatalog' -v -count=1 .` | **PASS** (46 tests) | Catalog enrichment, filters, detail status and routing. |
| **AC-04** | Client/pack selection matches preview and apply; explicit empty selection cannot expand to all | `go test -run 'TestInstallNoClient|TestExplicitEmptySelectionIsRejected|TestInstallPreviewConfirmAndApply' -v -count=1 .` | **PASS** (3 tests) | Engine validation and model selection. |
| **AC-05** | No writes before affirmative confirmation; option drift requires re-preview; double Enter cannot double apply | `go test -run 'TestInstallCancelAndBack|TestInstallDoubleEnter|TestInstallChangedSettingsInvalidatesConfirmation|TestInstallFocusAndResizeRetainsSafeNo' -v -count=1 .` | **PASS** (4 tests) | Safe default-No and resize preservation. |
| **AC-06** | Backup Enter only opens details; preview/default-No confirmation protect recovery | `go test -run 'TestRestore' -v -count=1 .` | **PASS** (19 tests) | Includes corrupt/stale/absent/refused and drift. |
| **AC-07** | Local Home renders immediately; bounded automatic update check; opt-out and retry without blocking | `go test -run 'TestUpdateState|TestUpdateCheckOncePerSessionAndControls|TestT18StatusClusterFactual' -v -count=1 .` | **PASS** (20 tests) | One-shot check, toggle, retry, factual cluster. |
| **AC-08** | Slow operations do not block rendering; writes cannot be abandoned; signals restore terminal safely | `go test -run 'TestSignal|TestOpsHost|TestCtrlCDeferredDuringWrite|TestAppCtrlCAndRepeatedSignalsDeferExitUntilResult' -v -count=1 .` | **PASS** (20 tests) | Signal and write-phase policy. |
| **AC-09** | Compact ASCII identity, nawodyaishan attribution, AAA contrast, safe glyphs, no route emoji | `go test -run 'TestBrand|TestTheme|TestT18SafeGlyphsAndNoEmoji' -v -count=1 .` | **PASS** (21 tests) | TrueColor/ANSI reference contrast and branding. |
| **AC-10** | Existing operation bytes/contracts unchanged except enumerated additive TUI help; no scripted auto-launch | root `task test && task vet` | **PASS** | Recorded in §6 after the fix set. |
| **AC-11** | Isolated nested module buildable, root contracts preserved, launcher handles companion safely | `cd experiments/tui && go test -count=1 ./... && go vet ./...` | **PASS** | Recorded in §6. |
| **AC-12** | Every supported operation discoverable; excluded experimental operations non-executable | `go test -run 'TestActivity|TestB2Search|TestLibrary_|TestCatalogModel' -v -count=1 .` | **PASS** (42 tests) | Help/commands and catalog discovery. |
| **AC-13** | Hyperlinks keyboard-openable only on explicit action, visible URLs, plain fallbacks | `go test -run 'TestOpener' -v -count=1 .` | **PASS** (10 tests) | Injected desktop opener and URL display. |
| **AC-14** | Homebrew interactive zero-argument launch opens TUI; explicit flags/pipes retain CLI contracts | root `go test -v ./cmd/agentic-sdd -run 'TestTUI' -count=1` | **PASS** | Dispatch, forwarding, no-TTY fail-closed. |
| **AC-15** | One semantic palette; agentic-dark tokens with AAA actual pairs; stable layout | `go test -run 'TestTheme|TestT18|TestT35Breakpoints' -v -count=1 .` | **PASS** (17 tests) | TrueColor and ANSI-256 actual-pair checks; ANSI-16 best-effort documented. |
| **AC-16** | Bounded centered help/confirm and fullscreen fallback; truthful feedback; respects motion; stops cleanly | `go test -run 'TestT19|TestT20|TestColoredHelpOverlayKeepsTitleAndFooter' -v -count=1 .` | **PASS** (30 tests) | Modal geometry, feedback, spinner stop. |
| **AC-17** | FR-11.1/2: grouped totals reconcile; all paths reachable; zero-change preview bypasses confirmation | `go test -run 'TestB8' -v -count=1 .` | **PASS** (35 tests) | Automated evidence only; human acceptance pending. |
| **AC-18** | FR-11.3/4: one confirmation owner and truthful actions, no input leakage; safe Enter and affirmative key | `go test -run 'TestB8Confirmation|TestB8ShiftTabCyclesFocusInReverse|TestB8ContextualFooterAndHelpShareEligibility' -v -count=1 .` | **PASS** (6 tests) | Confirmation ownership and inert backdrop. |
| **AC-19** | FR-11.5/12: goal-first Home, plain client/pack labels, unchanged defaults, actual outcomes and refresh | `go test -run 'TestHome|TestDeriveSetupState|TestB2Home|TestPreviewAndResultsUsePlainClientLabels|TestExplicitEmptySelectionIsRejected' -v -count=1 .` | **PASS** (28 tests) | Automated evidence only; owner walkthrough pending. |
| **AC-20** | FR-13: one dark scheme with AAA actual pairs, capability fallbacks, concise chrome, session preferences | `go test -run 'TestTheme|TestT30|TestAppColorProfile' -v -count=1 .` | **PASS** (12 tests) | Covers the profile message seam and actual preference state. |
| **AC-21** | FR-14: task-first Library, readable Recovery distinguishing new installs from snapshots, bounded logs | `go test -run 'TestLibrary_|TestRecovery_|TestRouting|TestActivity_' -v -count=1 .` | **PASS** (49 tests) | Automated evidence only; human review pending. |
| **AC-22** | FR-15: responsive failure/empty states and optional learning without scope expansion | `go test -run 'TestT35|TestT34|TestOnboarding' -v -count=1 .` | **PASS** (29 tests) | Responsive fixtures and Quick start; novice study `not_run`. |

**Omissions that are not pass:** novice 5s/60s study, owner walkthrough, macOS PTY, SSH/tmux, screen readers, terminal-remapped ANSI palettes, reduced-color visual review, and a human visual review of the integrated shell. Report these as `not_run` until observed.

---

## 4. T37: Owner Walkthrough Checklist

Run against the integrated tree with temporary homes and fake/read-only operation seams. Human observations are required; automated tests do not replace them.

1. **Home & Orientation**
   - [ ] Two-line value proposition and factual local inventory.
   - [ ] Per-client status, missing/refused counts, and no fabricated optional content.
   - [ ] One primary setup action plus Browse/Recovery/Help links.
2. **Setup & Installation**
   - [ ] Client selection, options, review, exact preview, default-No confirmation.
   - [ ] Zero-change preview shows "Already up to date" with no write/log/confirmation.
   - [ ] Confirmation stays default-No through resize, including tiny terminals; Enter is inert.
   - [ ] Write phase, result outcome, backup/log references, and inventory refresh are honest.
3. **Library & Discovery**
   - [ ] Search by task/name/ID and category filters.
   - [ ] Router recommendation opens details only; no execution.
   - [ ] Detail hierarchy shows identity, routing, concise destination status, Purpose/When/Invocation/Caveat, then complete paths only under Advanced files.
   - [ ] Paging reaches long paths and caveats without horizontal clipping.
4. **Recovery & Backup Inspection**
   - [ ] Deterministic reverse-chronological list with readable timestamps and exact IDs.
   - [ ] Fresh-install records vs replacement snapshots are distinguishable, including zero-snapshot cases.
   - [ ] Enter opens details only; restore preview and confirmation are separate; corrupt entries are unusable.
5. **Activity, Diagnostics & Help**
   - [ ] Help organization and session preferences show actual motion/ASCII/check state.
   - [ ] Checks are labelled "checks enabled/disabled", never "online".
   - [ ] Diagnostics and bounded logs keep privacy redactions and unknown fields honest.
6. **Terminal Resilience & Accessibility**
   - [ ] Resize across `140×40`, `100×30`, `80×24`, `60×20`, `40×8` preserves focus, selection, scroll and consent state.
   - [ ] Below `40×8` shows resize guidance; an open confirmation accepts no affirmative input.
   - [ ] TrueColor/ANSI-256 reference contrast; ANSI-16 is best-effort; user-remapped palettes are not guaranteed.
   - [ ] ASCII mode is 7-bit and reversible: dynamic paths/IDs/errors are escaped, not hyphen-rewritten.

### 4.1 Study and Platform Status

- **Novice orientation**: owner-confirmed complete on 2026-11-11. Participant count and method were not recorded in this repository; do not infer them from unit tests.
- **Platform-specific**: owner-confirmed complete on 2026-11-11 for macOS cask install/quarantine/migration, PTY, SSH/tmux, terminal/accessibility and visual review. Platform transcripts were not supplied; this record is a confirmation pointer, not an independent reproduction.

---

## 5. Review-Fix Reconciliation (OpenAI Sol findings 9–14)

| Finding | Resolution in this working tree | Regression evidence |
|---|---|---|
| 9 — reduced-color contrast claims | `theme.go` now carries the resolved reference profile; ANSI-256 mappings were corrected (`Muted` 252, selected background 236) and tested on actual rendered pairs; ANSI-16 is explicitly best-effort and documented, including user-remapped palettes. `App` follows `tea.ColorProfileMsg` and `TUIConfig.Profile` provides a deterministic seam. | `TestThemeReferenceProfilesActualPairs`, `TestAppColorProfileMessageUsesReferenceMappings`, `TestThemePaletteRolesMeetWCAG` (TrueColor). |
| 10 — ASCII paths rewritten | Install/restore dynamic homes, repos, IDs, paths, notices, errors, backup references and log paths are escaped before decorative mapping; Activity and Diagnostics dynamic values (log names, log paths, raw log lines, log directory) use the same reversible escape. Authoritative requests and model fields keep the original bytes; only decorative UI glyphs are mapped. | `TestInstallPathEscapingIsReversibleAndRequestExact`, `TestInstallRuntimeASCIIToggleEscapesData`, `TestRestorePathEscapingIsReversible`, `TestUnicodePathASCIIReversibleEscaping`, `TestActivityASCIIDataEscapingIsReversible`. |
| 11 — misleading B13 PASS evidence | This packet uses current test names, records actual selector runs, marks omissions `not_run`, and the T35 Unicode/confirmation fixtures now assert content reachability and safe resize instead of bounds alone. | `TestT35UnicodeAndLongPaths`, `TestT35ConfirmationResizeKeepsSafeNo`, `TestConfirmationBelowUsableGeometryCannotSubmit`. |
| 12 — `m` key conflict (motion toggle vs manifest) | Backup details own lowercase `m` in their own context: it opens the bounded manifest view without changing global motion. Elsewhere `m` stays the motion toggle, or the active overlay (help, confirmation, onboarding) keeps its own key handling instead of leaking the key to the backdrop. | `TestBackupDetailsOwnsLowercaseMThroughApp`, `TestRecovery_ManifestAndLogKeysInDetails`, `TestRecovery_StaleManifestReadIsDiscarded`. |
| 13 — hardcoded activity preferences | `ActivityModel.SetSessionPreferences` receives root motion/ASCII state; the Preferences section reports actual motion, symbols and "checks enabled/disabled" with no connectivity claim. | `TestActivityPreferencesReportActualSessionState`, `TestAppActivityPreferencesFollowRootToggles`. |
| 14 — Library hierarchy/path duplication | Status summary stays early without paths; complete paths appear once under Advanced files; ID, routing, statuses and unknown fallbacks remain. | `TestLibrary_DetailHierarchyKeepsPathsInAdvancedFiles`, `TestLibrary_DetailCardSectionsAndUnknownFallbacks`, `TestRoutingDetailCardSectionOrderAndContent`. |

---

## 6. Verification Commands and Results

Run from the repository root unless noted. These are the commands for this working-tree state; rerun them after any further edit.

```sh
cd experiments/tui
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
cd ../..
task test
task vet
git diff --check
```

- Nested `go test -count=1 ./...`: **PASS**.
- Nested `go vet ./...`: **PASS**.
- Nested `go test -race -count=1 ./...`: **PASS**.
- Root `task test`: **PASS**.
- Root `task vet`: **PASS**.
- `git diff --check`: **PASS**.
- AC-14 selector: `go test -v ./cmd/agentic-sdd -run 'TestTUI' -count=1`: **PASS**.

The exact per-AC selector commands and match counts are in §3. A selector that reports no matching tests is not recorded as PASS.

---

## 7. Integration Findings (unchanged preparation notes)

1. **Confirmation footer key hints**: the modal compositor preserves the active key footer; `y confirm` remains gated to the confirmation stage.
2. **Scroll clamp on resize**: `clampScrollOffsets` clamps content/detail offsets against the current body budget; the T35 resize fixtures exercise the active-confirmation case.
3. **Reserved status notice**: `statusNoticeRows` reserves footer rows above the key hint and returns nil below height 9 so resize guidance keeps the body.
4. **Wide-character alignment**: `ansi.StringWidth` cell accounting keeps CJK and combining marks inside the approved widths in the T35 fixtures.

These are observations from automated fixtures, not a substitute for the owner walkthrough or visual review.
