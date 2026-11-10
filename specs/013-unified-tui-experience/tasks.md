# Tasks: Feature 013 — Unified TUI Experience and Animated Branding

**Revision:** r3 — review reconciliation, 2026-11-10  
**Inputs:** [spec.md](spec.md) r3; [plan.md](plan.md) r3.  
**Approval pointer:** [spec.md §Approval](spec.md#approval), the only combined approval record. Specialist/tool assignments are in [plan §5](plan.md#5-specialist-and-tool-assignments). Owner requested B2 parallel free, DeepSeek Flash, MiMo Flash, MiniMax Flash and Gemini Flash lanes. These own gate coverage, metadata extraction, pure spacing/Home helpers, acceptance tests and catalog enrichment respectively. The coordinator owns root integration, gates and ledger; worker file ownership is disjoint.

## 1. Batch ledger

Every batch requires applicable combined approval, explicit authorization and acceptance of dependencies. Owner approved r3, accepted DEP-012 and authorized D03 via “all 1 2 3 approved accepted”. B1 is authorized with parallel independent subagents; later batches retain their review dependencies. On completion record checks, set `awaiting human review`, and stop.

| Batch | Tasks | Outcome / completion condition | Verification | State / next action |
|---|---|---|---|---|
| B1 | T01–T03 | Persistent responsive shell and accessible animated branding; existing Explore remains usable; AC-01/02/09 | Model/layout/fake-clock tests and owner visual walkthrough; root/nested gates | `accepted by owner` — user praised B1 and requested spacing refinement during next work |
| B2 | T04–T05 | Shared metadata/gate coverage, categorized rich catalog, local dashboard, text/line spacing, agent routing details and high-contrast shell | Metadata/inventory/search/status/resize/spacing/routing/contrast tests, no-network assertions and package-selector audit | `completed — accepted by owner 2026-11-10` — changes committed and pushed; B3 requires separate authorization |
| B3 | T06–T08 | Shared stdlib operation seam, validated client subsets, unchanged explicit CLI; AC-04/10 engine/seam coverage | Characterization, isolated-home subset/default/refusal tests, core and container gates | `accepted by owner` — 2026-11-10; commit/push authorized |
| B4 | T09–T10 | Preview-first install journey with safe async execution; AC-04/05 and write part of AC-08 | Fake-operation state-machine tests, isolated-home install/rollback/no-op tests, PTY checks | `authorized — starting` 2026-11-10 in parallel with B5; disjoint screen/host lanes |
| B5 | T11–T13 | Backup/restore, activity/help, automatic main-menu update check, browser guide and onboarding; AC-06/07/08/12/13 | Restore drift/refusal tests, bounded log/network and injected opener tests, full supported route walkthrough | `authorized — starting` 2026-11-10 in parallel with B4; screen integration depends on the B4 operation host |
| B6 | T15 → T14 → T16 | Paired package/marker proof first, then Homebrew-default launcher and final acceptance; AC-01–14 | Filtered-archive/universal/marker snapshot proof before launcher; dispatch/handshake/PTY tests and scoped gates | `not started` — accept B5, confirm r3 and authorize B6; D01 direction resolved |

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
| T09 | Client choice → optional pack/preset → source/home review → inspectable exact preview → default-No confirmation in nested screens; integrate optional Huh only if pinned integration is suitable. Do not add per-skill installation. | B3 acceptance, B4 authorization | Cancel/back/no-client/invalid-preset tests; changing settings invalidates confirmation; focus/resize retains safe No; AC-04/05 | pending |
| T10 | Async operations, single-write exclusion, Ctrl+C and OS SIGINT/SIGTERM/SIGHUP host policy with default Bubble Tea signal handling replaced explicitly. Defer exit during writes, flush results/logs, restore terminal idempotently; document wedged-write/SIGKILL limitation. | T09 | Deterministic delayed operation tests and named Python PTY cases for idle/in-flight/repeated signals; selected temporary-home no-op/replacement/failure/rollback; no root PTY dependency; AC-05/08 | pending |

### B5 — Recovery, activity and onboarding

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T11 | Backup list/details → restore preview → independent default-No confirmation → async restore and results. | B4 acceptance, B5 authorization | Enter on backup cannot write; corrupt/stale/absent/refused backup states; exact removal/overwrite preview; drift invalidates consent; temporary-home restore safety; AC-06 | pending |
| T12 | Activity, About, help, automatic first-Home update check and Retry; use T06 context-aware checker, once-per-session state and offline/`AGENTIC_SDD_NO_UPDATE_CHECK=1` controls. | T11 | Immediate local rendering, one automatic attempt across revisits, disabled zero calls, actual transport cancellation plus stale suppression, retry; timeout/offline/rate-limit/dev fixtures; no downloads/browser/other startup network; AC-07 | pending |
| T13 | Skippable shared onboarding; hyperlinked README/nawodyaishan profile with explicit keyboard opener; audit supported route matrix. Use fixed macOS opener or once-resolved Linux opener under documented user-PATH trust. | T12 | Fixed HTTPS links, plain/OSC8 fixtures, injected absolute-path resolution and no-shell launch; missing/SSH/failure fallback; no automatic clipboard/agent/browser; excluded controls non-executable; AC-12/13 | pending |

### B6 — Launch, distribution readiness and acceptance

| ID | Task and likely components | Dependency | Completion / verification | Status |
|---|---|---|---|---|
| T15 | **Execute first in B6.** Establish generic CLI/Homebrew-marked CLI/companion build IDs, per-app universal entries and filtered generic/Homebrew paired archives; inject shared release/dev content build identity. Configure explicit reviewed Go patch for nested release build with local toolchain policy, separate from core verification. Define direct companion flags/help and protocol-info response. | B5 acceptance, r3 confirmation and B6 authorization | Local nonpublishing snapshot proves archive names/contents/Mach-O slices, formula selection/renames, marker only in Homebrew CLI, matching identities, no formula compilation/post-install launch; missing tooling blocks proof; AC-11/14 | pending |
| T14 | After T15 package proof, implement explicit `tui` and Homebrew real-interactive bare default against its concrete marker/companion protocol. Add enumerated TUI help/usage/topic lines with reviewed fixture diffs. Preserve other bytes/flags/defaults/trace and forward signals without abandoning child writes. Update actual README/Taskfile commands. | T15 proof, B6 authorization | Distribution×args×TTY/forced-mode matrix; missing/mismatch/unknown-dev ID errors, direct companion gates and parent-child PTY signals; no shell/PATH companion lookup/fallback; additive-help-only diff review; AC-10/11/14 | pending |
| T16 | Final scoped regression and owner walkthrough across update/offline/browser/branding/Homebrew/resize/accessibility/error/terminal lifecycle journeys; finish architecture support/phase-note reconciliation and document limitations. | T14 | AC-01–14 matrix with code state and actual evidence; unrun checks blocked/not_run; no publish/support claim from unrun platform proof; state `awaiting human review` | pending |

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

- **Current position:** B1, B2 and B3 are accepted by the owner. B3 is being committed and pushed with owner authorization. B4 and B5 are authorized and starting in parallel with free, DeepSeek and Gemini Flash lanes; B6 remains unauthorized.
- **Approval:** See [spec.md §Approval](spec.md#approval); r3 approved, DEP-012 accepted, D03 authorized, B1/B2/B3 accepted and B4/B5 authorized by owner on 2026-11-10. B3's “first commit and push current changes” authorizes publishing the reviewed B3 tree only.
- **Changed implementation paths (B3):** `internal/skillsync/{clients,result,identity}.go` plus `sync.go`, `restore.go`, `inventory.go` and their tests; `internal/appops/{requests,ops,update}.go` and tests; `cmd/agentic-sdd/{logs,updates,guide,guide_presentation}.go` plus `b3_compat_test.go`, `logs_test.go`, `updates_test.go`; `docs/tui-architecture.md`. No TUI screen, source-skill, release, harness-selector or real-home changes.
- **Evidence code state:** HEAD `ad00ac0` plus working-tree B3 SHA256 `cbfad743445c0523c07483764758fc40d26e8bbe655337eeb9f25b11b3ac735a` over 26 changed/untracked paths under `internal/skillsync`, `internal/appops`, `cmd/agentic-sdd` and `docs/tui-architecture.md`, each path+NUL+contents+NUL.
- **B3 checks passed on this state:** `task test` and `task vet` (8 core packages incl. appops); `git diff --check`; `sh tests/guide-e2e.sh`; `sh tests/docker-e2e.sh` (preloaded `golang:1.25-alpine`); offline `task -t Taskfile.experimental.yml tui:test` (nested suite executed, not skipped); nested `go vet ./...` and `go mod tidy -diff`; `task -t Taskfile.experimental.yml tui:pty-test` (4 lifecycle cases). Focused `./internal/skillsync`, `./internal/appops` and `./cmd/agentic-sdd` also pass with `-count=1`.
- **Red/green evidence:** The read-only audit recorded the pre-B3 gaps — no old/new equivalence harness, no selection tests, no manifest canonical-target assertion, no subset restore round trip, no exact apply/restore/backups byte fixtures and no structured backup identity in logging. The new tests cover those gaps; the CLI byte fixtures characterize pre-B3 behavior and pass after the change. No separate failing-before-fix run was recorded for every case.
- **Actual delegation:** B3 free `opencode/deepseek-v4.1-flash-free` (gate-coverage audit; CLI byte characterization); DeepSeek `opencode-go/deepseek-v4.1-flash` max lane (client selection, structured engine results, source identity, appops requests/tokens, logging integration) and high lane (context-aware update checker with compatible CLI wrapper); Muse Spark `opencode-go/muse-spark-1.3-contributor` (architecture-note reconciliation). The owner's muse-spark request succeeded in this session; the earlier privacy block did not recur. The coordinator supplied the guide structured-result cleanup, token/selection review, gate runs and ledger reconciliation.
- **Limitations / open review:** B3 is awaiting owner review and has no commit/push authorization. The appops token pins the proposed change set and source identity, not destination bytes: a content edit to a destination already planned for replace/remove does not invalidate consent, while any drift that can introduce or remove a write does. Revalidation is pre-execution drift detection, not isolation from hostile concurrent writers. The explicit guide flow still previews then applies without consuming the new token (existing CLI behavior preserved); B4/T09 owns the confirmation-token journey. Trusted-CI adoption of the amended `harness/checks.json` remains separate: no `tests/harness` run was made under the B2 steering, and PR-path trusted-gate coverage is not claimed. macOS PTY, SSH/tmux, reduced-color/truecolor and screen-reader checks remain not_run.
- **Next permitted action:** Commit and push the accepted B3 tree, then start B4 and B5 with disjoint implementation lanes. B6 still requires separate authorization.

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
