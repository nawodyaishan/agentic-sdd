# Tasks: Feature 013 — Unified TUI Experience and Animated Branding

**Revision:** r4 — B7 styling amendment approved 2026-11-10; existing task states/history preserved.
**Inputs:** [spec.md](spec.md) approved r3 plus approved r4 FR-10/AC-15/16; [plan.md](plan.md) r3 plus approved r4 §8.
**Approval pointer:** [spec.md §Approval](spec.md#approval), the only combined approval record. Specialist/tool assignments are in [plan §5](plan.md#5-specialist-and-tool-assignments). Owner requested B2 parallel free, DeepSeek Flash, MiMo Flash, MiniMax Flash and Gemini Flash lanes. These own gate coverage, metadata extraction, pure spacing/Home helpers, acceptance tests and catalog enrichment respectively. The coordinator owns root integration, gates and ledger; worker file ownership is disjoint.

## 1. Batch ledger

Every batch requires applicable combined approval, explicit authorization and acceptance of dependencies. Owner approved r3, accepted DEP-012 and authorized D03 via “all 1 2 3 approved accepted”. B1 is authorized with parallel independent subagents; later batches retain their review dependencies. On completion record checks, set `awaiting human review`, and stop.

| Batch | Tasks | Outcome / completion condition | Verification | State / next action |
|---|---|---|---|---|
| B1 | T01–T03 | Persistent responsive shell and accessible animated branding; existing Explore remains usable; AC-01/02/09 | Model/layout/fake-clock tests and owner visual walkthrough; root/nested gates | `accepted by owner` — user praised B1 and requested spacing refinement during next work |
| B2 | T04–T05 | Shared metadata/gate coverage, categorized rich catalog, local dashboard, text/line spacing, agent routing details and high-contrast shell | Metadata/inventory/search/status/resize/spacing/routing/contrast tests, no-network assertions and package-selector audit | `completed — accepted by owner 2026-11-10` — changes committed and pushed; B3 requires separate authorization |
| B3 | T06–T08 | Shared stdlib operation seam, validated client subsets, unchanged explicit CLI; AC-04/10 engine/seam coverage | Characterization, isolated-home subset/default/refusal tests, core and container gates | `accepted by owner` — 2026-11-10; commit/push authorized |
| B4 | T09–T10 | Preview-first install journey with safe async execution; AC-04/05 and write part of AC-08 | Fake-operation state-machine tests, isolated-home install/rollback/no-op tests, PTY checks | `accepted by owner` — 2026-11-10; committed and pushed |
| B5 | T11–T13 | Backup/restore, activity/help, automatic main-menu update check, browser guide and onboarding; AC-06/07/08/12/13 | Restore drift/refusal tests, bounded log/network and injected opener tests, full supported route walkthrough | `accepted by owner` — 2026-11-10; committed and pushed |
| B6 | T15 → T14 → T16 | Paired package/marker proof first, then Homebrew-default launcher and final acceptance; AC-01–14 | Filtered-archive/universal/marker snapshot proof before launcher; dispatch/handshake/PTY tests and scoped gates | `authorized — starting` 2026-11-10 in parallel with B7; T15 before T14; local nonpublishing proof only |
| B7 | T17 → T18 → T19 → T20 | Styling-only semantic/adaptive theme, chrome/list/glyph refinement, help/confirmation modals and truthful feedback; FR-10.1–10.5, AC-15/16 with affected AC-01/02/05/08/09/10/11/13 regressions | Token/contrast/profile/cell/layout/modal/phase tests, existing safety/CLI/nested/root/PTY gates and owner visual walkthrough; stop for review | `authorized — starting` 2026-11-10 in parallel with B6; T17 → T20 |

B7 is **approved by the owner 2026-11-10 and authorized for implementation**. B1–B6 rows and task states are historical/current records. B6 and B7 start in parallel with disjoint ownership; B6 executes T15 before T14, B7 executes T17 → T20. There is no automatic numerical-batch progression, and B6/T16 and B7/T20 remain separate acceptance stops. B7/T17 has no dependency on B6 package/launcher proof.

## 2. Ordered tasks

Task state is recorded below. Write focused failing tests before behavioral implementation. Existing operation fixtures stay unchanged; only enumerated additive `tui` help/usage/topic fixture diffs may change, with explicit review (FR-09). Task IDs remain stable; B6 intentionally executes T15 before T14.

### B1 — Navigation and animated identity

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T01 | Characterize nested launch and CLI operation/plain/numeric contracts; define root route/focus model and four destinations with one contextual footer. Preserve standalone opt-in entry; establish the named Python PTY harness/prerequisites from plan §6. | DEP-012, applicable approval and B1 authorization | Root model/filter/navigation tests; 4 Linux PTY cases pass (quit, early source error, non-TTY stdin, forced script mode); CLI compatibility/core/container checks unchanged | implemented — awaiting human review |
| T02 | Add cell-aware sidebar/list/detail layout and scroll/focus preservation at width/height breakpoints; safe too-small view. | T01 | Breakpoint/short-height tests plus parent regressions for reachable long Unicode paths, bottom footer, per-route scroll persistence, last-item visibility and overscroll recovery; ANSI/grapheme-aware wrapping | implemented — awaiting human review |
| T03 | **Improve branding with animated ASCII logo, colors, icons and emoji, and personal attribution “by nawodyaishan.”** Implement compact mark/reveal, violet/cyan tokens, labelled status styles, static/reduced-motion setting, Unicode/ASCII session option and reused environment controls; preserve attribution in About when narrow. | T02 | Pure 10-frame/static branding and root tick/input tests pass; visible motion/symbol settings and ASCII data escaping; D02 owner visual acceptance remains pending | implemented — awaiting human visual review |

### B2 — Explore and local Home

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T04 | Create `internal/appops` with pure curated teaching metadata (B2 owns package creation); enrich catalog. With explicit D03 permission, add the package to Taskfile/harness/core-shell test/vet and required-directory lists; audit Lefthook and package-count guidance. Do not duplicate metadata or defer coverage to B3. | B1 acceptance, D03 decision, r3 confirmation and B2 authorization | Metadata extracted with 43/43 curated entries preserved and zero CLI prose changes; categorized catalog counts derived from Kind+Origin; unknown items neutral; search covers ID/name/purpose/category; stale async results discarded by generation+category; selection kept by kind+ID; agent routing table drift-tested against routing.md and preset frontmatter, with manual-failover/no-runtime claims; all prior core packages retained plus appops; AC-03 | completed — accepted by owner |
| T05 | Home local overview/recommended installation action and rich catalog detail panes; navigation into installation configuration placeholder without writes. | T04 | Inventory counts/client states from snapshot only; word-boundary wrapping, 2-space section indents, single blank separators and one-cell pane gutters; rich detail shows routing, destination paths/status and next action with full scroll reach; category focus/Tab/[ ]/c controls; long paths reachable, footer retained, bounds at 40/60/79/80/109/110 and h8/12/24; high-contrast Midnight Lagoon palette (AAA text, 3:1 borders), colored padded top bar, compact short-height header and color-off plain output; read-only home/source snapshots unchanged with proxy denied; AC-03/07/09 local scope | completed — accepted by owner |

### B3 — Shared operation boundary and client selection

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T06 | Extend B2's existing `internal/appops` with shared orchestration; add sync/restore structured-result engine variants plus exact-compatibility wrappers, replacing backup-text parsing in logging. Extract context-aware update checker with compatible CLI wrapper. Audit T04 gate coverage again. | B2 acceptance, r3 confirmation, B3 authorization | Byte/error equivalence; structured backup/outcome/no-op/pre/post-backup failure/rollback tests; private log policy and context cancellation tests; all core/container gates; AC-10/11 seam | implemented — awaiting human review |
| T07 | Add optional explicit client-selection value: nil retains default-all, non-nil empty fails engine/adapter validation. Narrow only changes/entries; always record full canonical core target identity in backup manifest, optional profiles separately. | T06 | Subset replacement→backup→supported restore round trip; unselected client files untouched; omitted/empty/unknown selections, preview/apply/inventory parity, pack/preset validation; AC-04/06 | implemented — awaiting human review |
| T08 | Define immutable typed requests/results, generation IDs and precise preview/revalidation tokens per plan §1. Reconcile architecture note's current-support header, mutation table, guide-wiring proposal and root migration prerequisites while preserving 012 history. | T07 | Source/selected-path/absence/new-write/options/backup-payload drift requires re-preview; unrelated unselected edits do not; stale/double result tests, existing safety checks retained; AC-05/06 foundations | implemented — awaiting human review |

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
| T15 | **Execute first in B6.** Establish generic CLI/Homebrew-marked CLI/companion build IDs, per-app universal entries and filtered generic/Homebrew paired archives; inject shared release/dev content build identity. Configure explicit reviewed Go patch for nested release build with local toolchain policy, separate from core verification. Define direct companion flags/help and protocol-info response. | B5 acceptance, r3 confirmation and B6 authorization | Local nonpublishing snapshot proves archive names/contents/Mach-O slices, formula selection/renames, marker only in Homebrew CLI, matching identities, no formula compilation/post-install launch; missing tooling blocks proof; AC-11/14 | pending |
| T14 | After T15 package proof, implement explicit `tui` and Homebrew real-interactive bare default against its concrete marker/companion protocol. Add enumerated TUI help/usage/topic lines with reviewed fixture diffs. Preserve other bytes/flags/defaults/trace and forward signals without abandoning child writes. Update actual README/Taskfile commands. | T15 proof, B6 authorization | Distribution×args×TTY/forced-mode matrix; missing/mismatch/unknown-dev ID errors, direct companion gates and parent-child PTY signals; no shell/PATH companion lookup/fallback; additive-help-only diff review; AC-10/11/14 | pending |
| T16 | Final scoped regression and owner walkthrough across update/offline/browser/branding/Homebrew/resize/accessibility/error/terminal lifecycle journeys; finish architecture support/phase-note reconciliation and document limitations. | T14 | AC-01–14 matrix with code state and actual evidence; unrun checks blocked/not_run; no publish/support claim from unrun platform proof; state `awaiting human review` | pending |

### B7 — Semantic styling refinement (approved)

**Boundary:** one approved execution batch, T17 → T18 → T19 → T20. B1–B5 acceptance is the baseline; D04 and DEP-B7 are resolved. B7 runs in parallel with B6 on disjoint files: B6 owns release/launcher paths, B7 owns nested TUI styling/presentation paths. B6/T16 and B7/T20 are separate acceptance stops. After T20 verification, stop `awaiting human review`; no following batch is implicitly authorized.

| ID | Task and likely components | Dependency | Completion / per-task verification | Status |
|---|---|---|---|---|
| T17 | Define semantic tokens/profile maps in nested `theme.go`; root asynchronous background-color handling and presentation-only Theme propagation to all children/delegate/filter styles. Preserve deterministic standalone constructor, NO_COLOR/plain and root isolation. No local raw colors. | B1–B5 acceptance; D04 light-floor/reduced-color decision; combined B7 approval and explicit B7 authorization | Regression-first tests for every actual dark/light surface/selected/modal role pair (>=7:1 text / >=3:1 non-text under proposed floor), forced TrueColor/ANSI-256/ANSI-16 reference profiles and remapping limits; missing/late background messages do not block, mutate consent or reset selection/filter/scroll; color-off no queries/paint/SGR; palette/component-default audit; focused nested tests; FR-10.1, AC-15 | authorized — starting |
| T18 | Refine `viewport.go`, `app_screens.go`, catalog/list and row renderers: derived breadcrumbs, existing-route pills, factual status cluster, padded top bar, stage-accurate styled footer hints, fixed header/list/gutter geometry and filter focus. Replace decorative emoji with centralized safe Unicode/ASCII markers, preserving data and links. | T17; approved D04 glyph decision | Bounds/geometry fixtures at 40/60/79/80/109/110 plus 100/130 tier edges and h8/12/16/24; stable marker/selected row widths, focus/filter/category/navigation/last-row/scroll preservation; update/offline flags reflect existing state only; long/wide/combining data, OSC 8 visible URLs, NO_COLOR/ASCII/plain; focused nested tests and reviewed presentation-only assertion changes; FR-10.2/5, AC-15 | authorized — starting |
| T19 | Add common centered modal compositor in nested viewport/helper; supply existing help and install/restore confirmation content with dimmed inactive backdrop, bounded scroll body, pinned policy/keys and full-screen tiny-terminal fallback. Retain existing handlers/tokens; onboarding shares Theme but stays its existing journey. | T18 | Center/fit/cell/ANSI/OSC and focus/scroll restoration tests, all warnings/paths reachable; shrink/grow/help/background reply never confirms; default No/inert Enter/affirmative key/cancel semantics unchanged; stale preview requires re-preview, duplicate submit zero extra writes; no underlying hyperlink/cursor leak or busy-policy overlay; focused safety/integration tests; FR-10.3, AC-16 | authorized — starting |
| T20 | Integrate root-owned motion-aware phase spinner and persistent inline status/toast styling using existing host messages; optional native indeterminate progress only if explicitly selected and supported at the pin (no ProgramStatus). Finish AC-15/16 regression evidence and owner walkthrough preparation. | T19; approved D04 phase-spinner exception/native-progress choice | Fake-clock/host starting→writing→flushing/result/error/deferred-exit/static/stale-tick assertions; no fabricated percentages or phase advances, timer stops and optional native state clears, full error/backup/log references retained; focused tests then root/nested/race/PTY and unchanged CLI gates per plan §§6/8.7; audit root manifests/dependency/operation paths unchanged; AC-15/16 and affected prior AC evidence with not_run limits; stop awaiting human review after authorized implementation | authorized — starting |

Assignments are inherited once from plan §8.6; no task-specific specialist/tool exceptions. Review presentation assertion diffs narrowly (including current emoji expectations), keeping all behavior/safety tests and CLI byte fixtures. B7 cannot change root CLI/engine/appops, signals, Taskfiles, harness, release files or root dependencies to make a check pass. Any necessary scope expansion stops for a human decision.

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
- T15 finalizes the pinned build-toolchain/GoReleaser commands and nonpublishing snapshot before T14. No release tags, formula pushes, dependency downloads, real-home applies or live operations pre-authorized.

## 4. Continuation

- **Current position:** B1–B5 are accepted by the owner. B3 is published at `22f0318`; B4/B5 are being committed and pushed with owner authorization. B7 drafting is requested; B6 and B7 implementation remain unauthorized.
- **Approval:** See [spec.md §Approval](spec.md#approval); r3 approved, DEP-012 accepted, D03 authorized, B1–B5 accepted and B7 drafting requested by owner on 2026-11-10. The B4/B5 “commit and push” authorization covers the accepted B4/B5 tree only.
- **Changed implementation paths (B4/B5):** `experiments/tui/{ops_host,signals_unix,signals_other,install,restore,activity,update_state,onboarding,opener,app_screens,app}.go`, `experiments/tui/cmd/tui-spike/main.go`, their tests, `experiments/tui/integration_b4b5_test.go` and `tests/tui/test_terminal_lifecycle.py`. No root CLI, engine, source-skill, release or harness-selector changes.
- **Evidence code state:** HEAD `22f0318` plus working-tree B4/B5 SHA256 `836f6df04b64075a937850704b2ef1dbb3d5c5464e07b7abe01c3c881532b891` over 23 changed/untracked paths under `experiments/tui` plus the named PTY test, each path+NUL+contents+NUL.
- **B4/B5 checks passed on this state:** `task test` and `task vet` (8 core packages); `git diff --check`; `sh tests/guide-e2e.sh`; `sh tests/docker-e2e.sh`; offline `task -t Taskfile.experimental.yml tui:test`; nested `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...` and `go mod tidy -diff`; `task -t Taskfile.experimental.yml tui:pty-test` (10 lifecycle cases, including idle/in-flight SIGINT/SIGTERM/SIGHUP). Root and nested gates were re-run after the integration lane.
- **Red/green evidence:** Each lane added focused tests for its own state machine; the integration lane added hub/journey/single-write/deferred-exit/update/onboarding/opener and bounds tests. The tests pass on the integrated tree, but no separate failing-before-fix run was recorded for every assertion.
- **Actual delegation:** B4/B5 fleet — DeepSeek `opencode-go/deepseek-v4.1-flash` max for the operation host and final integration; Gemini `9router/ag/gemini-3.8-flash-high` for the install journey; DeepSeek high for backup/restore; Gemini `9router/ag/gemini-3.8-flash-medium` for Activity/update; free `opencode/deepseek-v4.1-flash-free` for onboarding/opener. The Gemini Flash lanes ran without fallback. The coordinator reviewed the integration, formatted tests and re-ran the gates.
- **Limitations / open review:** B4/B5 are awaiting owner visual and route walkthrough; macOS PTY, SSH/tmux, reduced-color/truecolor and screen-reader checks remain not_run. The update check was exercised with injected checks only — no live GitHub request or browser launch. A real post-backup TUI failure/rollback was not provoked; rollback states are covered by engine tests and fake host results. `docs/subagent-fleet-comparison.md` is an untracked file that this batch did not create and does not own. B6 packaging/launcher and release remain unauthorized.
- **B7 approval (r4):** Research input is [b7-styling-research.md](b7-styling-research.md); FR-10/AC-15/16 and plan §8 plus T17–T20 are approved by the owner on 2026-11-10. B7 implementation is authorized and starts in parallel with B6 on disjoint files. The B4/B5 evidence/limitations above remain historical assertions.
- **B7 drafting model:** Owner-selected OpenAI GPT-6.1 Sol (`openai/gpt-6.1-sol`), not Claude/Sonnet and not a claimed automatic fallback. No fallback failure reason is known; parent owns fallback handling and the human handoff.
- **B6/B7 parallel start:** B6 `sdd` lanes own `.goreleaser.yml`, release/launcher paths, version/protocol identity and packaging proof; B7 lanes own `experiments/tui` theme/viewport/app presentation. T15 must prove packaging before T14; B7 runs T17 → T18 → T19 → T20. No live tag, formula publication or real-home operation.
- **Next permitted action:** Commit the approved B7 documents, then start B6 T15 and B7 T17 in parallel. Stop each batch for its own review; B6/T16 and B7/T20 must not claim cross-batch final acceptance until both reviews complete.

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
