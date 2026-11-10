# Tasks: Feature 013 — Unified TUI Experience and Animated Branding

**Revision:** r5 — cask packaging amendment authorized 2026-11-10; existing task states/history preserved.
**Inputs:** [spec.md](spec.md) approved r3 plus approved r4 FR-10/AC-15/16; [plan.md](plan.md) r3 plus approved r4 §8.
**Approval pointer:** [spec.md §Approval](spec.md#approval), the only combined approval record. Specialist/tool assignments are in [plan §5](plan.md#5-specialist-and-tool-assignments). Owner requested B2 parallel free, DeepSeek Flash, MiMo Flash, MiniMax Flash and Gemini Flash lanes. These own gate coverage, metadata extraction, pure spacing/Home helpers, acceptance tests and catalog enrichment respectively. The coordinator owns root integration, gates and ledger; worker file ownership is disjoint.

**UX amendment:** r6 implementation is integrated in the main working tree and awaiting human review; approval is recorded once in [spec.md §Approval](spec.md#approval). B8–B13 are implemented; B11/B13 rows below reflect the final integrated state. Preserve r5 cask T21 and all earlier batch states. No commit, push, release, real-home apply or new gate change is authorized by this ledger.

## 1. Batch ledger

Every batch requires applicable combined approval, explicit authorization and acceptance of dependencies. Owner approved r3, accepted DEP-012 and authorized D03 via “all 1 2 3 approved accepted”. B1 is authorized with parallel independent subagents; later batches retain their review dependencies. On completion record checks, set `awaiting human review`, and stop.

| Batch | Tasks | Outcome / completion condition | Verification | State / next action |
|---|---|---|---|---|
| B1 | T01–T03 | Persistent responsive shell and accessible animated branding; existing Explore remains usable; AC-01/02/09 | Model/layout/fake-clock tests and owner visual walkthrough; root/nested gates | `accepted by owner` — user praised B1 and requested spacing refinement during next work |
| B2 | T04–T05 | Shared metadata/gate coverage, categorized rich catalog, local dashboard, text/line spacing, agent routing details and high-contrast shell | Metadata/inventory/search/status/resize/spacing/routing/contrast tests, no-network assertions and package-selector audit | `completed — accepted by owner 2026-11-10` — changes committed and pushed; B3 requires separate authorization |
| B3 | T06–T08 | Shared stdlib operation seam, validated client subsets, unchanged explicit CLI; AC-04/10 engine/seam coverage | Characterization, isolated-home subset/default/refusal tests, core and container gates | `accepted by owner` — 2026-11-10; commit/push authorized |
| B4 | T09–T10 | Preview-first install journey with safe async execution; AC-04/05 and write part of AC-08 | Fake-operation state-machine tests, isolated-home install/rollback/no-op tests, PTY checks | `accepted by owner` — 2026-11-10; committed and pushed |
| B5 | T11–T13 | Backup/restore, activity/help, automatic main-menu update check, browser guide and onboarding; AC-06/07/08/12/13 | Restore drift/refusal tests, bounded log/network and injected opener tests, full supported route walkthrough | `accepted by owner` — 2026-11-10; committed and pushed |
| B6 | T15 → T14 → T16, plus T21 | Paired package/marker proof first, then Homebrew-default launcher and final acceptance; AC-01–14 | Filtered-archive/universal/marker snapshot proof before launcher; dispatch/handshake/PTY tests and scoped gates | `in progress` — T15 proof and T14 review published at `974eeca`; T21 cask amendment locally implemented; T16 final acceptance/owner walkthrough pending |
| B7 | T17 → T18 → T19 → T20 | Styling-only semantic/adaptive theme, chrome/list/glyph refinement, help/confirmation modals and truthful feedback; FR-10.1–10.5, AC-15/16 with affected AC-01/02/05/08/09/10/11/13 regressions | Token/contrast/profile/cell/layout/modal/phase tests, existing safety/CLI/nested/root/PTY gates and owner visual walkthrough; stop for review | `accepted by owner` — 2026-11-10; published at `430b0e0` |

B7 is **approved by the owner 2026-11-10 and authorized for implementation**. B1–B6 rows and task states are historical/current records. B6 and B7 start in parallel with disjoint ownership; B6 executes T15 before T14, B7 executes T17 → T20. There is no automatic numerical-batch progression, and B6/T16 and B7/T20 remain separate acceptance stops. B7/T17 has no dependency on B6 package/launcher proof.

### r6 batch extension (approved; implementation awaiting human review)

All rows below were authorized and implemented. B8–B13 now rest in the integrated working tree; each still requires human acceptance, and no next batch starts automatically. Existing B6/T16/T21 obligations are separate and unchanged.

| Batch | Tasks / dependency | Outcome / completion condition | Verification | State / next action |
|---|---|---|---|---|
| B8 — P0 trust and interaction | T22 → T23 → T24; accepted B7 baseline, r6 approval + B8 authorization | Grouped exact impact, read-only no-op, one confirmation owner and shared contextual actions; AC-17/18 | Typed-plan reconciliation, 94-entry/unknown/refused/no-op fixtures; state×key/focus, drift/double-submit and fake-host zero-write checks; scoped gates | `accepted by owner` — 2026-11-11; DeepSeek max commit `2f82605`; commit/push authorized; human/platform checks remain `not_run` |
| B9 — P0 newcomer and truthful outcome | T25 → T26 → T27; r6 approval + B9 authorization | Goal-first Home/setup, plain client/pack labels and correct same-state refresh/results; AC-19 | Fresh/partial/modified/current/unknown fixtures, defaults/nonempty-client guards, same-home apply/no-op/restore and refresh failure tests | `accepted by owner` — 2026-11-11; DeepSeek Go high commit `166c678`; B9/B8 label reconciliation applied; commit/push authorized |
| B10 — P1 single dark identity | T28 → T29 → T30; r6 approval + B10 authorization, approved D06 | One Mocha-based agentic-dark with retained AAA, restrained chrome and accessible preferences; AC-20 | Actual pair/token/profile tests; background-reply safety, bounds/static/plain/ASCII/update/preference checks and owner visuals | `accepted by owner` — 2026-11-11; Gemini 3.8 Flash High commit `91c62ff`; commit/push authorized |
| B11 — P1 discovery and recovery | T31 → T32 → T33; r6 approval + B11 authorization | Purpose-first Library, clear backup facts, bounded Activity and organized Help; AC-21 | Search/back identity, zero/legacy/corrupt backups, zero-snapshot fresh install, inspect/preview zero writes, bounded logs and unknown fields | `accepted by owner` — 2026-11-11; Gemini 3.8 Flash High commit `fc6ca23`; recovery truthfulness hardened after review; commit/push authorized |
| B12 — P2 optional learning | T34; r6 approval + B12 authorization | Skippable/reopenable Quick start and first Router guidance without new persistence/runtime scope; learning part of AC-22 | Skip/reopen/back tests preserve options/consent, no clipboard/browser/agent/client calls | `accepted by owner` — 2026-11-11; DeepSeek Go high commit `d908f37`; B9 plain-label reconciliation applied; commit/push authorized |
| B13 — P1 integrated robustness and acceptance | T35 → T36 → T37; B13 authorization | Consolidated responsive/failure/compatibility evidence for AC-17–22 and affected AC-01–16; owner review packet | Breakpoint/review-size fixtures, root/nested/race/PTY/e2e as applicable; temporary-home walkthrough, explicit platform/user-study omissions | `accepted by owner` — 2026-11-11; packet at [b13-acceptance-checklist.md](b13-acceptance-checklist.md); T35 fixtures include content reachability and active-confirmation resize safety; T36 selectors matched and passed; T37 human walkthrough, novice study and platform checks remain `not_run` |

The review's robustness requirements apply in **every** earlier batch; B13 consolidates them, it does not postpone fixing unsafe clipping/focus. Its P1 label describes required quality, not permission to ship an incomplete P0 journey.

### Review-priority traceability

| Review finding / priority | Contract | Task owners |
|---|---|---|
| UX-01 P0 goal-first Home | FR-12 / AC-19 | B9 T25 |
| UX-02/03 P0 client/pack terminology | FR-12 / AC-19 | B9 T26 |
| UX-04 P0 long exact preview | FR-11.1 / AC-17 | B8 T22 |
| UX-05 P0 duplicate confirmation observation | FR-11.3 / AC-18 | B8 T24 reproduces before correcting |
| UX-06 P0 focus/keys | FR-11.4 / AC-18 | B8 T24 |
| UX-07 P0 no-op too late | FR-11.2 / AC-17 | B8 T23 |
| UX-08 P1 density/blank areas | FR-13/15 / AC-20/22 | B10 T30, B13 T35 |
| UX-09 P1 recovery facts/discovery | FR-14 / AC-21 | B11 T32 |
| UX-10 P1 task-first Library | FR-14 / AC-21 | B11 T31 |
| UX-11 P1 distracting status cluster | FR-13 / AC-20 | B10 T30; B11 T33 diagnostics |
| UX-12 P1 advanced paths | FR-11.1/12/14 / AC-17/19/21 | B8 T22, B9 T25/26, B11 T31/32 |
| UX-13 P1 development metadata | FR-14 / AC-21 | B11 T33 |
| UX-14 P1 footer overload | FR-11.4 / AC-18 | B8 T24; B10 T30 spacing |
| UX-15 P1 no-op warning color | FR-11.2/13 / AC-17/20 | B8 T23 semantics; B10 T30 styling |
| UX-16 P2 optional first-use learning | FR-12/15 / AC-19/22 | B9 T25/27, B12 T34 |
| Review §§7/9/11 palette/responsiveness/quality | FR-13/15 / AC-20/22 | B10 T28–30, B13 T35–37; regressions in every batch |
| Review P2 future Fleet | FR-15 exclusion / D06 | Deferred; no executable placeholder or implementation task |

## 2. Ordered tasks

Task state is recorded below. Write focused failing tests before behavioral implementation. Existing operation fixtures stay unchanged; only enumerated additive `tui` help/usage/topic fixture diffs may change, with explicit review (FR-09). Task IDs remain stable; B6 intentionally executes T15 before T14.

### B1 — Navigation and animated identity

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T01 | Characterize nested launch and CLI operation/plain/numeric contracts; define root route/focus model and four destinations with one contextual footer. Preserve standalone opt-in entry; establish the named Python PTY harness/prerequisites from plan §6. | DEP-012, applicable approval and B1 authorization | Root model/filter/navigation tests; 4 Linux PTY cases pass (quit, early source error, non-TTY stdin, forced script mode); CLI compatibility/core/container checks unchanged | accepted by owner |
| T02 | Add cell-aware sidebar/list/detail layout and scroll/focus preservation at width/height breakpoints; safe too-small view. | T01 | Breakpoint/short-height tests plus parent regressions for reachable long Unicode paths, bottom footer, per-route scroll persistence, last-item visibility and overscroll recovery; ANSI/grapheme-aware wrapping | accepted by owner |
| T03 | **Improve branding with animated ASCII logo, colors, icons and emoji, and personal attribution “by nawodyaishan.”** Implement compact mark/reveal, violet/cyan tokens, labelled status styles, static/reduced-motion setting, Unicode/ASCII session option and reused environment controls; preserve attribution in About when narrow. | T02 | Pure 10-frame/static branding and root tick/input tests pass; visible motion/symbol settings and ASCII data escaping; D02 owner visual acceptance remains pending | implemented — awaiting human visual review |

### B2 — Explore and local Home

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T04 | Create `internal/appops` with pure curated teaching metadata (B2 owns package creation); enrich catalog. With explicit D03 permission, add the package to Taskfile/harness/core-shell test/vet and required-directory lists; audit Lefthook and package-count guidance. Do not duplicate metadata or defer coverage to B3. | B1 acceptance, D03 decision, r3 confirmation and B2 authorization | Metadata extracted with 43/43 curated entries preserved and zero CLI prose changes; categorized catalog counts derived from Kind+Origin; unknown items neutral; search covers ID/name/purpose/category; stale async results discarded by generation+category; selection kept by kind+ID; agent routing table drift-tested against routing.md and preset frontmatter, with manual-failover/no-runtime claims; all prior core packages retained plus appops; AC-03 | completed — accepted by owner |
| T05 | Home local overview/recommended installation action and rich catalog detail panes; navigation into installation configuration placeholder without writes. | T04 | Inventory counts/client states from snapshot only; word-boundary wrapping, 2-space section indents, single blank separators and one-cell pane gutters; rich detail shows routing, destination paths/status and next action with full scroll reach; category focus/Tab/[ ]/c controls; long paths reachable, footer retained, bounds at 40/60/79/80/109/110 and h8/12/24; high-contrast Midnight Lagoon palette (AAA text, 3:1 borders), colored padded top bar, compact short-height header and color-off plain output; read-only home/source snapshots unchanged with proxy denied; AC-03/07/09 local scope | completed — accepted by owner |

### B3 — Shared operation boundary and client selection

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T06 | Extend B2's existing `internal/appops` with shared orchestration; add sync/restore structured-result engine variants plus exact-compatibility wrappers, replacing backup-text parsing in logging. Extract context-aware update checker with compatible CLI wrapper. Audit T04 gate coverage again. | B2 acceptance, r3 confirmation, B3 authorization | Byte/error equivalence; structured backup/outcome/no-op/pre/post-backup failure/rollback tests; private log policy and context cancellation tests; all core/container gates; AC-10/11 seam | accepted by owner |
| T07 | Add optional explicit client-selection value: nil retains default-all, non-nil empty fails engine/adapter validation. Narrow only changes/entries; always record full canonical core target identity in backup manifest, optional profiles separately. | T06 | Subset replacement→backup→supported restore round trip; unselected client files untouched; omitted/empty/unknown selections, preview/apply/inventory parity, pack/preset validation; AC-04/06 | accepted by owner |
| T08 | Define immutable typed requests/results, generation IDs and precise preview/revalidation tokens per plan §1. Reconcile architecture note's current-support header, mutation table, guide-wiring proposal and root migration prerequisites while preserving 012 history. | T07 | Source/selected-path/absence/new-write/options/backup-payload drift requires re-preview; unrelated unselected edits do not; stale/double result tests, existing safety checks retained; AC-05/06 foundations | accepted by owner |

### B4 — Safe installation workflow

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T09 | Client choice → optional pack/preset → source/home review → inspectable exact preview → default-No confirmation in nested screens; integrate optional Huh only if pinned integration is suitable. Do not add per-skill installation. | B3 acceptance, B4 authorization | Cancel/back/no-client/invalid-preset tests; changing settings invalidates confirmation; focus/resize retains safe No; AC-04/05 | accepted by owner |
| T10 | Async operations, single-write exclusion, Ctrl+C and OS SIGINT/SIGTERM/SIGHUP host policy with default Bubble Tea signal handling replaced explicitly. Defer exit during writes, flush results/logs, restore terminal idempotently; document wedged-write/SIGKILL limitation. | T09 | Deterministic delayed operation tests and named Python PTY cases for idle/in-flight/repeated signals; selected temporary-home no-op/replacement/failure/rollback; no root PTY dependency; AC-05/08 | accepted by owner |

### B5 — Recovery, activity and onboarding

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T11 | Backup list/details → restore preview → independent default-No confirmation → async restore and results. | B4 acceptance, B5 authorization | Enter on backup cannot write; corrupt/stale/absent/refused backup states; exact removal/overwrite preview; drift invalidates consent; temporary-home restore safety; AC-06 | accepted by owner |
| T12 | Activity, About, help, automatic first-Home update check and Retry; use T06 context-aware checker, once-per-session state and offline/`AGENTIC_SDD_NO_UPDATE_CHECK=1` controls. | T11 | Immediate local rendering, one automatic attempt across revisits, disabled zero calls, actual transport cancellation plus stale suppression, retry; timeout/offline/rate-limit/dev fixtures; no downloads/browser/other startup network; AC-07 | accepted by owner |
| T13 | Skippable shared onboarding; hyperlinked README/nawodyaishan profile with explicit keyboard opener; audit supported route matrix. Use fixed macOS opener or once-resolved Linux opener under documented user-PATH trust. | T12 | Fixed HTTPS links, plain/OSC8 fixtures, injected absolute-path resolution and no-shell launch; missing/SSH/failure fallback; no automatic clipboard/agent/browser; excluded controls non-executable; AC-12/13 | accepted by owner |

### B6 — Launch, distribution readiness and acceptance

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T15 | **Execute first in B6.** Establish generic CLI/Homebrew-marked CLI/companion build IDs, per-app universal entries and filtered generic/Homebrew paired archives; inject shared release/dev content build identity. Configure explicit reviewed Go patch for nested release build with local toolchain policy, separate from core verification. Define direct companion flags/help and protocol-info response. | B5 acceptance, r3 confirmation and B6 authorization | Local nonpublishing snapshot proves archive names/contents/Mach-O slices, Homebrew artifact selection, marker only in Homebrew CLI, matching identities, no install-time launch; missing tooling blocks proof; AC-11/14 | implemented — snapshot proof passed (GoReleaser v2.14.0, Go 1.27.1, no external lipo; paired archives/universal slices/identity verified); package form superseded by T21 |
| T14 | After T15 package proof, implement explicit `tui` and Homebrew real-interactive bare default against its concrete marker/companion protocol. Add enumerated TUI help/usage/topic lines with reviewed fixture diffs. Preserve other bytes/flags/defaults/trace and forward signals without abandoning child writes. Update actual README/Taskfile commands. | T15 proof, B6 authorization | Distribution×args×TTY/forced-mode matrix; missing/mismatch/unknown-dev ID errors, direct companion gates and parent-child PTY signals; no shell/PATH companion lookup/fallback; additive-help-only diff review; AC-10/11/14 | reviewed by owner — accepted |
| T16 | Final scoped regression and owner walkthrough across update/offline/browser/branding/Homebrew/cask/resize/accessibility/error/terminal lifecycle journeys; finish architecture support/phase-note reconciliation and document limitations. | T14, T21 | AC-01–14 matrix with code state and actual evidence; unrun checks blocked/not_run; no publish/support claim from unrun platform proof; state `awaiting human review` | pending — T14 review + owner walkthrough |
| T21 | **Cask migration amendment (D05).** Replace `brews` with a prebuilt-artifact `homebrew_casks` entry: name the Homebrew CLI artifact `agentic-sdd`, link `agentic-sdd` + `agentic-sdd-tui`, strip macOS quarantine from both unsigned binaries after install, and record the formula-to-cask tap migration sequence. Update static release-config assertions, README install/upgrade guidance, AGENTS release notes and stale architecture packaging text. | T15 proof and owner D05 authorization | `goreleaser check`; local nonpublishing snapshot writes `Casks/agentic-sdd.rb` with both `binary` stanzas and only `xattr` post-install actions; checksum matches the Homebrew archive; static test rejects `brews`/`Formula`; README uses `brew install --cask`; no live tap/release | implemented locally — awaiting review; tap publish not authorized |

### B7 — Semantic styling refinement (approved)

**Boundary:** one approved execution batch, T17 → T18 → T19 → T20. B1–B5 acceptance is the baseline; D04 and DEP-B7 are resolved. B7 runs in parallel with B6 on disjoint files: B6 owns release/launcher paths, B7 owns nested TUI styling/presentation paths. B6/T16 and B7/T20 are separate acceptance stops. After T20 verification, stop `awaiting human review`; no following batch is implicitly authorized.

| ID | Task and likely components | Dependency | Completion / per-task verification | Status |
|---|---|---|---|---|
| T17 | Define semantic tokens/profile maps in nested `theme.go`; root asynchronous background-color handling and presentation-only Theme propagation to all children/delegate/filter styles. Preserve deterministic standalone constructor, NO_COLOR/plain and root isolation. No local raw colors. | B1–B5 acceptance; D04 light-floor/reduced-color decision; combined B7 approval and explicit B7 authorization | Regression-first tests for every actual dark/light surface/selected/modal role pair (>=7:1 text / >=3:1 non-text under proposed floor), forced TrueColor/ANSI-256/ANSI-16 reference profiles and remapping limits; missing/late background messages do not block, mutate consent or reset selection/filter/scroll; color-off no queries/paint/SGR; palette/component-default audit; focused nested tests; FR-10.1, AC-15 | accepted by owner |
| T18 | Refine `viewport.go`, `app_screens.go`, catalog/list and row renderers: derived breadcrumbs, existing-route pills, factual status cluster, padded top bar, stage-accurate styled footer hints, fixed header/list/gutter geometry and filter focus. Replace decorative emoji with centralized safe Unicode/ASCII markers, preserving data and links. | T17; approved D04 glyph decision | Bounds/geometry fixtures at 40/60/79/80/109/110 plus 100/130 tier edges and h8/12/16/24; stable marker/selected row widths, focus/filter/category/navigation/last-row/scroll preservation; update/offline flags reflect existing state only; long/wide/combining data, OSC 8 visible URLs, NO_COLOR/ASCII/plain; focused nested tests and reviewed presentation-only assertion changes; FR-10.2/5, AC-15 | accepted by owner |
| T19 | Add common centered modal compositor in nested viewport/helper; supply existing help and install/restore confirmation content with dimmed inactive backdrop, bounded scroll body, pinned policy/keys and full-screen tiny-terminal fallback. Retain existing handlers/tokens; onboarding shares Theme but stays its existing journey. | T18 | Center/fit/cell/ANSI/OSC and focus/scroll restoration tests, all warnings/paths reachable; shrink/grow/help/background reply never confirms; default No/inert Enter/affirmative key/cancel semantics unchanged; stale preview requires re-preview, duplicate submit zero extra writes; no underlying hyperlink/cursor leak or busy-policy overlay; focused safety/integration tests; FR-10.3, AC-16 | accepted by owner |
| T20 | Integrate root-owned motion-aware phase spinner and persistent inline status/toast styling using existing host messages; optional native indeterminate progress only if explicitly selected and supported at the pin (no ProgramStatus). Finish AC-15/16 regression evidence and owner walkthrough preparation. | T19; approved D04 phase-spinner exception/native-progress choice | Fake-clock/host starting→writing→flushing/result/error/deferred-exit/static/stale-tick assertions; no fabricated percentages or phase advances, timer stops and optional native state clears, full error/backup/log references retained; focused tests then root/nested/race/PTY and unchanged CLI gates per plan §§6/8.7; audit root manifests/dependency/operation paths unchanged; AC-15/16 and affected prior AC evidence with not_run limits; stop awaiting human review after authorized implementation | accepted by owner |

Assignments are inherited once from plan §8.6; no task-specific specialist/tool exceptions. Review presentation assertion diffs narrowly (including current emoji expectations), keeping all behavior/safety tests and CLI byte fixtures. B7 cannot change root CLI/engine/appops, signals, Taskfiles, harness, release files or root dependencies to make a check pass. Any necessary scope expansion stops for a human decision.

### B8 — P0 grouped decisions and consistent focus (proposed)

| ID | Objective / likely components | Dependency | Completion / focused verification | Status |
|---|---|---|---|---|
| T22 | Derive immutable per-action/per-destination impact projection from existing typed preview; summary before expandable groups/paths in `install.go` and reusable restore presentation where applicable. Keep original item/token identity, stable detail state and complete path access. | Accepted B7, combined r6 approval, B8 authorization | Mixed client/action totals reconcile exactly; removal and refusal shown distinctly; unknown kind not addition; 94-entry stress case, Unicode/long paths, pinned summary and full detail reachability; no mutation or engine API changes; FR-11.1 / AC-17 | proposed |
| T23 | Add read-only Already up to date branch for complete successful zero-mutation preview; actions Home/Library/details/options, no confirm/apply/log/backup side effects. | T22 | Zero fake write-host calls including y/Enter/repeated input; temporary-home tree unchanged and no new backup/log; refused/error/incomplete/unknown never no-op; changed options/fresh later preview can require normal consent; explicit CLI unchanged; FR-11.2 / AC-17 | proposed |
| T24 | Reproduce screenshot duplicate confirmation deterministically, then ensure single render/focus owner for install/restore; unify action bindings/help/footer eligibility and reverse Tab across visible focus targets. Preserve existing affirmative handlers and aliases. | T23 | Input/list/checkbox/details/help/modal/write × Enter/Esc/q/Space/?/slash/j/k/arrows/Tab/Shift+Tab/Ctrl+C matrix; one copy of confirm facts, no backdrop handlers/links; inert Enter/default No/y-Y, preview drift, duplicate submit, focus/scroll restoration, resize/late message zero accidental writes; FR-11.3/4 / AC-18; applicable lifecycle and CLI gates; stop for B8 human review | proposed |

### B9 — P0 Home, setup language and outcomes (proposed)

| ID | Objective / likely components | Dependency | Completion / focused verification | Status |
|---|---|---|---|---|
| T25 | Recompose Home in `app_screens.go`/existing models: two-line purpose, evidence-based setup state, one primary action, Browse/Recovery links, secondary inventory and advanced details. | B8 human acceptance + B9 authorization | First-use/current/partial/modified/unknown/read-failure fixtures; no hardcoded counts/forced onboarding, paths reachable, keyboard primary action at each size; no extra network or persistent marker; FR-12 / AC-19 | proposed |
| T26 | Plain client/pack labels, explanations and source/home disclosure in `install.go`/onboarding; preserve IDs, current defaults, selection values and engine constraints. | T25 | Four supported display names, shared-directory explanation, optional-profile/free-preference caveat, pack dependency, explicit-empty rejection and source/path review; no executable/auth detection or pack-only plan; back navigation retains options; FR-12 / AC-19 | proposed |
| T27 | Verify then correct result/refresh presentation as necessary in install/restore/root: typed outcomes, real backup/log references, generation-bound local inventory refresh and honest refresh failure/stale status. | T26 | Same-home/source/options install → inventory → no-op → restore → inventory fixtures; stale results ignored; no-op does not write; failure/partial/rollback unknowns not optimistic success; optional Router next step; root/nested/PTY/e2e regressions as applicable; FR-11.5/12 / AC-19; stop for B9 human review | proposed |

### B10 — P1 single agentic-dark theme and hierarchy (proposed)

| ID | Objective / likely components | Dependency | Completion / focused verification | Status |
|---|---|---|---|---|
| T28 | Establish Mocha semantic role/surface pairing in `theme.go`; generate full contrast matrix before applying screen changes; resolve selected/border conflicts without lowering AAA. | B9 human acceptance, approved D06, B10 authorization | >=7:1 actual primary/secondary/muted/status text pairs on canvas/rail/surface/selection/modal, >=3:1 meaningful focus/borders; selected candidate primary on `#313244`; review selected `#45475A` pair not accepted as AAA; raw-token audit; FR-13 / AC-20 | proposed |
| T29 | Replace active adaptive light selection with one dark resolver; propagate styles without resetting children; preserve reference TrueColor/256/16, plain/NO_COLOR/ASCII and deterministic constructors. | T28 | No light theme/switcher; late/absent background replies cannot change scheme/routes/consent/filter/scroll; profile/downsampling pairs checked, user ANSI remapping limit stated; no-color no paint/query/style controls; retain relevant old safety tests and narrowly replace light-specific expectations; FR-13 / AC-20 | proposed |
| T30 | Restrained header/breadcrumb/panel spacing, neutral no-op, concise update notice and Help session preferences in existing viewport/App/Activity models; keep branding/reveal/phase spinner/attribution/opener contracts. | T29 | Preferences motion/symbol/offline reachable via keyboard/help; visible initial checking and actionable update, unavailable diagnostics not persistent clutter; no repeats/downloads; short/narrow safety budget and labelled selection, no decorative filler; owner visual examples and applicable gates; stop for B10 human review | proposed |

### B11 — P1 Library, Recovery, Activity and Help (proposed)

| ID | Objective / likely components | Dependency | Completion / focused verification | Status |
|---|---|---|---|---|
| T31 | Purpose/task-first Library rows/search and detail hierarchy, maintained category metadata and Router recommendation in existing catalog models; exact IDs/routing/advanced files retained. | B10 human acceptance + B11 authorization | Task/name/ID search, unknown metadata and truthful installed/modified/refused statuses; Router opens details only; category/query/kind+ID/scroll stable detail→back/resize; no second metadata registry or execution; FR-14 / AC-21 | proposed |
| T32 | Improve Recovery entry and validated backup summaries/details in `restore.go`/Home routing: readable time/exact ID, recorded new entries versus saved snapshots, known affected clients, bounded manifest/log disclosure. | T31 | Zero/one/many, corrupt/refused/legacy backups, unknown timestamp/log association, fresh install with recorded entries and zero snapshots; Enter inspect only, preview read-only, separate default-No restore with drift protection; exact removal/overwrite paths remain reachable; FR-14 / AC-21 | proposed |
| T33 | Bounded recent-operations summary and raw-log view, organize Help sections/Diagnostics/About/preferences in `activity.go` and existing route models. | T32 | Unknown structured fields not inferred from prose; bounded huge/missing/unreadable logs and privacy tests; Development build leads to exact metadata; Retry and URL open explicit/injected, excluded operations explanatory only; scope requires no schema/format changes; applicable gates and owner discovery/recovery walkthrough; stop for B11 human review | proposed |

### B12 — P2 optional learning (proposed)

| ID | Objective / likely components | Dependency | Completion / focused verification | Status |
|---|---|---|---|---|
| T34 | Refine existing onboarding into optional in-session Quick start/tour and post-success first Router prompt; reuse Home/setup/help, not a second wizard or persistent preferences service. | B11 human acceptance + B12 authorization | Skip/reopen/back retain choices, no confirmation/preview reuse after changed options, no client/clipboard/agent/browser launch; static/plain/narrow usability; future Fleet/profile editor absent; FR-15 / AC-22; stop for B12 human review | proposed |

### B13 — Integrated responsiveness, regression and human acceptance (proposed)

| ID | Objective / likely components | Dependency | Completion / focused verification | Status |
|---|---|---|---|---|
| T35 | Consolidate missing responsive/empty/failure fixtures and fix only approved presentation gaps in existing layouts/screens. Do not rewrite routing/framework or change breakpoints. | B12 human acceptance + B13 authorization | Widths 40/60/79/80/109/110, tier edges 100/130, heights 8/12/16/24; 140×40/100×30/80×24/60×20 review samples; fresh/partial/modified/current/offline/refused/failure/empty; Unicode/long paths/footer/last item/focus and confirmation fit; deterministic representative snapshots with reviewed diffs; FR-15 / AC-22 | proposed |
| T36 | Run scoped current-tree acceptance/regression matrix AC-17–22 plus affected AC-01–16; preserve CLI bytes, root isolation and terminal lifecycle. Record command, code revision/diff identity, actual result and omission reason. | T35 | `task test`, `task vet`, actually executed nested test/vet/race, named PTY lifecycle/launcher cases and unchanged guide/docker e2e where applicable; diff check; no excluded validation, selectors weakened, live home/network/browser use; evidence not_run/blocked where missing; prior packaging evidence referenced with its own code state | proposed |
| T37 | Owner walkthrough and documentation/architecture support reconciliation for r6, scoped to actual outcomes; novice orientation test if feasible; reconcile remaining B6/T16/T21 status without claiming acceptance. | T36; separate B6 closure required for whole-feature readiness | Walkthrough records Home/setup/no-op/mixed preview/cancel/confirm/result/Library/zero-snapshot recovery/preferences/tiny terminal; 5s/60s targets and 3–5 novices measured or explicitly not_run; platform limits listed; README/support note reflect only verified scope; batch `awaiting human review`, no auto release/next batch | proposed |

**Shared assignments:** plan §9.7; no task-specific specialist/tool exceptions or new agent authority. Each batch runs focused failing tests before behavioral changes and applicable plan §6/§9.6 gates. Shared files overlap, so sequence batches; no implied parallel lanes. New UX implementation may not alter root engine/appops APIs, manifest/log formats, source skills, release/cask files, root dependencies or maintained gate selectors. Return for scope review if a required fact cannot be exposed safely through current results.

## 3. Verification policy and commands

Commands below are planned implementation checks, **not results from this documentation session**.

```sh
git status --short
task test
task vet
git diff --check
task -t Taskfile.experimental.yml tui:test
sh tests/guide-e2e.sh
sh tests/docker-e2e.sh
```

- Run focused tests first, then plan §6 gates. Nested tests must actually execute; a skip is not_run. Existing seven core packages/sanitation stay selected; add `internal/appops` only after explicit D03 authorization, with every package/required-directory list and count reference reconciled. Stop on a gate-policy conflict.
- Use temporary homes, injected transports and deterministic clocks; no real client directories or live update requests. Use PTYs for cleanup and handled-interruption evidence, not headless snapshots alone.
- Preserve J1–J10 IDs/inputs/defaults and operation-output fixtures. Only FR-09's enumerated additive TUI help/usage/topic changes permit reviewed fixture updates. Inspect Git state before editing: r3 finds only untracked 013 files, not an existing e2e modification; historical handovers are not current-state evidence.
- Do not run broad Python discovery, excluded experimental OpenCode commands, or 005/006/007 exercises. If a required aggregate includes them, record it blocked/not_run and request a decision; no skips/tolerances to manufacture pass.
- **Owner B2 scope steering:** no further standalone `tests/harness` runs or harness/OpenCode-harness improvement work. Earlier additive package-list/gate fixture edits and runs predate this note and are frozen for scope review; do not describe them as UI acceptance or shipping harness capability. Focused UI/metadata tests continue. Existing generic core Go gates are not OpenCode improvements and retain their coverage; any aggregate actually selecting excluded validations remains blocked/not_run, never filtered.
- PTY harness: `python3 -B -m unittest discover -s tests/tui -p test_terminal_lifecycle.py` once implemented; explicit file selection only. No root Go dependency or assumed Alpine `script` binary. Missing prerequisites mean blocked/not_run.
- T15 finalizes the pinned build-toolchain/GoReleaser commands and nonpublishing snapshot before T14. No release tags, tap pushes, dependency downloads, real-home applies or live operations pre-authorized. T21 changes only the Homebrew package form and its documentation; it does not authorize a tap migration push.

## 4. Continuation

### r6 implementation continuation — accepted and awaiting publication

- **Position:** B8–B13/T22–T37 are implemented, integrated and **accepted by the owner on 2026-11-11**; commit and push are authorized. HEAD was `0f853b5` before this commit; the integrated tree is being published as one Feature 013 commit.
- **Approval pointer:** [spec.md §Approval](spec.md#approval), r6 combined approval and B8–B13 authorizations. This continuation records execution only; it does not create or extend approval.
- **Integrated code state:** 40 changed paths under `experiments/tui/**`; `git diff HEAD -- experiments/tui` digest `279d5dc0a8c828b227ec49ccd950d392816647d5ef0d6ce8f190e746d5b7fc36` (11,530 patch lines at this checkpoint). The B13 packet is at [b13-acceptance-checklist.md](b13-acceptance-checklist.md).
- **Review/fix evidence:** OpenAI Sol (`openai/gpt-6.1-sol#max`) architecture/code review returned `Needs changes` with 14 findings (8 major, 6 minor) in the categories recovery I/O/associations, hidden confirmation execution, stale filter/activity results, no-op refresh, Router detail state, reduced-color claims, ASCII path fidelity, acceptance evidence, key conflicts, preferences and Library hierarchy. Two DeepSeek fix lanes (`opencode-go/deepseek-v4.1-flash#max`/`#high`) addressed all 14; an independent DeepSeek verification returned `Ready with findings`, and a third DeepSeek lane closed the five residuals (Activity ASCII data, manifest stat/open race, B13 numbering, synchronous ActivityModel I/O, filter query validation).
- **Checks passed on the final tree:** nested `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...`; root `task test` and `task vet` (8 supported-core packages); `git diff --check`; B13 selectors matched and passed; AC-14 focused launcher test. These are automated checks, not human/platform acceptance.
- **Remaining limitations:** ANSI-16 accent/error roles are explicitly best-effort and do not claim AAA on the standard reference palette; TrueColor and ANSI-256 reference pairs are tested. Human walkthrough, novice 5s/60s study, macOS PTY/Homebrew, SSH/tmux, screen readers, terminal-remapped palettes and visual review remain `not_run`. B6/T16, cask T21 live publication and the `internal/buildinfo` gate question remain separate.
- **Next permitted action:** commit and push the accepted tree (authorized 2026-11-11), then report the published SHA. Do not release/tag, push the tap or run a real-home apply without separate authorization. B6/T16 remains a separate acceptance stop.

### Pre-r6 implementation continuation (historical evidence; not UX authorization)

- **Current position:** B1–B5 and B7 are accepted and published. B3 is at `22f0318`, B4/B5 at `c360586`, the approved B7 documents at `5051299`, the B7 implementation at `430b0e0`, and B6 T15/T14 at `974eeca`. T21 (cask migration amendment, D05) is implemented locally and **uncommitted, awaiting owner review**. T16 final acceptance/owner walkthrough remains pending.
- **Approval:** See [spec.md §Approval](spec.md#approval); r3 approved, DEP-012 accepted, D03 authorized, B1–B5 accepted, B7 approved, B6/B7 implementation authorized and D05 cask migration authorized on 2026-11-10. D05 authorizes implementation and local nonpublishing proof only; it does not authorize a tag, release, tap push, `tap_migrations.json` publication or live install.
- **Changed implementation paths (T21):** `.goreleaser.yml`, `internal/buildinfo/release_config_test.go`, `Taskfile.experimental.yml`, `README.md`, `AGENTS.md`, `docs/tui-architecture.md`, `specs/013-unified-tui-experience/{spec,plan,tasks,b6-packaging-research}.md`. Earlier B6/B7 paths remain as published at `974eeca`/`430b0e0`.
- **B6 checks passed on the published T15/T14 tree (`974eeca`):** `task test` and `task vet` (8 core packages); `git diff --check`; `sh tests/guide-e2e.sh`; `sh tests/docker-e2e.sh`; nested `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...` and `go mod tidy -diff`; `task -t Taskfile.experimental.yml tui:test`; `task -t Taskfile.experimental.yml tui:pty-test` (10 lifecycle + 13 launcher cases); focused `go test ./internal/buildinfo` and vet; `release:snapshot` pass.
- **T21 local checks:** `goreleaser check` passes; `PATH=/tmp/opencode/b6-tools:$PATH task -t Taskfile.experimental.yml release:snapshot` writes `dist/homebrew/Casks/agentic-sdd.rb`; the cask has `binary "agentic-sdd"` + `binary "agentic-sdd-tui"` and only quarantine-removal post-install actions; the Homebrew archive contains `agentic-sdd` + `agentic-sdd-tui`; `dist/checksums.txt` matches; focused `internal/buildinfo` release-config tests pass. Root/nested regression gates are `not_run` for the T21 diff.
- **Red/green evidence:** B7 lanes added regression-first theme/contrast, chrome/bounds, modal-safety and phase-spinner tests; B6 added build-identity/protocol/config tests. The T21 config test was updated before the snapshot proof; no separate failing-before-pass run is recorded for every assertion. B6 now has snapshot/Mach-O/archive/cask proof at the local, nonpublishing level.
- **Actual delegation:** B6 T15 DeepSeek `opencode-go/deepseek-v4.1-flash` max; B7 T17 Gemini `9router/ag/gemini-3.8-flash-high`; T18 Gemini `9router/ag/gemini-3.8-flash-medium`; T19 DeepSeek `opencode-go/deepseek-v4.1-flash` high; T20 free `opencode/deepseek-v4.1-flash-free`. All lanes ran without fallback. T21 research and implementation were coordinator-run with Exa/Context7 sources.
- **Limitations / open review:** T21 proof used GoReleaser `v2.14.0` installed outside the repository; the snapshot task requires that exact version on PATH and does not download it. The archive `ids` use universal artifact IDs because `replace: true` removes single-arch artifacts. The tap remains formula-only (`Formula/agentic-sdd.rb`); publishing `Casks/agentic-sdd.rb` and the same-commit `tap_migrations.json` migration are separate live operations requiring Homebrew `>=5.0.6`. The cask is unsigned, so its quarantine hook and real install behavior remain `not_run` until a macOS walkthrough. `internal/buildinfo` has focused tests but is not in the 8-package core selector; adding it is a gate-selection change for review. T16 remains the final AC-01–14 matrix, Homebrew/cask/owner walkthrough and architecture support reconciliation. macOS PTY, SSH/tmux, truecolor/ANSI and screen-reader checks remain `not_run`. No live GitHub request, browser launch, tap write or real-home write was performed. `docs/subagent-fleet-comparison.md` and `docs/agent-fleet-spec-review-2026-11-10.md` are untracked files this batch does not own.
- **Next permitted action:** Owner review of the T21 cask diff and local snapshot evidence. Do not commit/push, publish a release, push the tap or run a live Homebrew journey without explicit authorization. T16 final AC-01–14 acceptance and the macOS cask walkthrough remain pending; do not claim B6 batch acceptance before T16.

## 5. Architecture-review reconciliation

Source: owner-requested DeepSeek V4.1 Flash review, child session `ses_e3db3502effe5rkXV1pHs0osbG`, verdict Needs changes. This maps document remedies, not implementation or verified closure.

| Finding | r3 remedy / implementation owner |
|---|---|
| 1 subset backups rejected by restore | Full canonical target identity independent of selected changes; T07 round-trip restore test |
| 2 backup identity available only in prose | Structured engine variants/compatibility wrappers and logging integration; T06 |
| 3 incomplete marker/universal/toolchain/package strategy | Explicit build IDs, separate filtered paired archives, per-app universals, pinned build stage; T15 before T14 |
| 4 incompatible blanket plain-byte promise | Enumerated additive TUI help/usage/topic exceptions; T14 reviewed fixture diffs |
| 5 missing appops core-gate coverage | D03 explicit authorization and all selector/required-directory/count updates with package creation; T04/T06 |
| 6 T04 depended on future T06 package | B2 creates pure metadata package; B3 extends it |
| 7 no PTY mechanism | Explicit Python stdlib harness/opt-in task; T01/T10/T14 |
| 8 missing OS-signal/terminal lifecycle policy | Host signal ownership, deferred write shutdown, idempotent release, parent-child handling; T10/T14 |
| 9 empty selection means all | Nil-default versus explicit nonempty selection contract at engine seam; T07 |
| 10 vague preview identity scope | Exact selected/read-reviewed source/path set, proposed-plan comparison and backup payload identity; T08 |
| 11 companion/opener trust undefined | Direct companion public flags plus private protocol-info; absolute sibling and documented desktop opener trust; T13/T15/T14 |
| 12 historical architecture claims conflict | Explicit four-part historical/support/phase reconciliation; T08/T16 |
| 13 update cancellation lacks context | Context-aware checker and transport cancellation test; T06/T12 |
| 14 `dev` version handshake insufficient | Shared injected revision/content identity; unknown identity fails closed; T15/T14 |
| 15 stale working-tree instruction | Current-state inspection replaces historical e2e/uncommitted-work assumption |
