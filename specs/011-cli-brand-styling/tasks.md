# Tasks: Feature 011 — Branded CLI Styling

**Revision:** Approved r4 with review clarifications, 2026-11-09. **Inputs:** [spec](spec.md) approved r6, [plan](plan.md) r4. Initial B1/r5 history retained; B2–B4 implementation and final verification delegation authorized at spec §Approval.
**Approval pointer:** [spec §Approval](spec.md#approval), the single approval record.
**Shared assignments:** plan §§4–5 own historical B1 lanes; plan §7 owns proposed r6 boundaries and §7.6 its specialist/tool assignments. No task-specific specialist exceptions for B2–B4.

## Batch B1 — Complete branded guide and safe update affordances

- **Tasks:** T01–T08. This was the original approved batch; its parallel lanes were not separately authorized batches. Draft B2–B4 below expand the same feature without changing B1's historical state.
- **Outcome:** All spec FR-01–08 and AC-01–10 addressed, with explicit manual-check status.
- **Completion conditions:** Integrated tree has no unexpected scope/dependency/gate changes; required automated checks pass; independent review has no unresolved blocking findings; human results/omissions are recorded without invented approval. Human acceptance remains separate.
- **State:** `awaiting human review — r5 UX corrections verified, 2026-11-09`.
- **Next action:** Owner reviews the corrected menu/build/update-result experience and pending visual/link checks. No next batch or live operation is authorized. Original and amendment approvals are recorded only in spec §Approval.
- **Execution order:** `T01 → [T02 || T03 || T04] → T05 → [T06 || T07] → T08`. All implementation writers stop before parallel verification starts. T07 may consume T06's completed evidence afterwards without changing that snapshot.

## Ordered tasks — initial delivery history

The completed rows below describe the original implementation. The affected animation checks (including T02's former 400 ms/raw-byte constraints) are superseded by spec r4 and the current amendment section; the old passing review is not evidence for changed animation.

| ID | Objective and assigned lane | Dependencies | Acceptance result and focused verification | Status |
|---|---|---|---|---|
| T01 | Coordinator pins current tree, interfaces and output baseline; loads assigned implementation guidance and sends bounded work packets | Combined approval and B1 authorization | Record `git status`, base commit plus diff/untracked-content fingerprint; capture existing plain guide (`stdin` interactive, stdout non-TTY), preview/help/version/error bytes using fixtures; run current supported-core `task test` and `task vet`. Resolve unexpected pre-existing failures without weakening gates. Confirm leaf interfaces/ownership from plan §§2/4; no live commands | completed; baseline evidence below |
| T02 | Free lane builds ASCII branding, semantic styles, narrow layout helpers and deterministic optional reveal; tests first, then implementation | T01 | Own only `branding.go`/`branding_test.go`. AC-01–04/08 renderer-level coverage: color/no-color/dumb, widths 40/60/80, version/dev, full URL, static/animated equality, at most 4 chunks/400 ms, no waits when disabled, reset before waits and no forbidden controls. Focused `go test ./cmd/agentic-sdd -run '^TestBranding'` | completed |
| T03 | DeepSeek lane builds bounded read-only updater, stable-version comparison and fixed plain update instructions; tests first | T01 | Own only `updates.go`/`updates_test.go`. AC-09/10: numeric comparison incl. 0.9/0.10 and huge components; invalid/dev/suffix/control tags; fixed endpoint/headers, timeout/redirect/body bounds and body close; equal/ahead/newer/unavailable/failed; non-200, 404, 403/429, timeout, malformed/oversized/malicious JSON, draft/prerelease. Validate exact unstyled Homebrew script, no untrusted interpolation. Fake transport only. Focused `go test ./cmd/agentic-sdd -run '^TestUpdates'` | completed |
| T04 | Gemini Flash lane drafts reference/flow changes and extends network-less container assertions | T01; finalized against T05 | Own only `docs/installer-reference.md`, `docs/guide-flows.md`, `tests/guide-e2e-in-container.sh`. Cover FR-04/06–08 and `updates.result`; clarify installer command equivalents versus guide-only update check. Add static/color/narrow/animation/no-color and graceful offline update assertions; unset inherited animation env before journeys. Preserve all prior tests and `--network none`. Run `sh -n tests/guide-e2e-in-container.sh`; full container test waits for T05. No production code or ledger edits | completed; coordinator reconciliation described below |
| T05 | DeepSeek integrates shared environment/guide/summary call sites and adds cross-component regression tests, test-first | T02, T03, T04 writer completion | Sole shared-Go writer per plan §4. Resolve `Animate` once; inject version/waits/checker; intro once; accept menu `5` only with terminal stdout; print/update trace result and return to menu without automatic retries. Keep defaults/EOF/legacy bytes/logs/schema. Add `guide_branding_test.go`, `guide_updates_test.go`, `branding_summary_test.go`; assert network spy zero/one calls, new choice rejected in legacy branch, install/decline/repeat/restore, full paths/commands and NO_COLOR. Run focused new tests and existing guide/plain/log/trace tests. Coordinator reconciles any final docs/shell edits only after lane C stops | completed |
| T06 | Free validation-only worker runs deterministic regression gates against frozen integrated tree | T05; all writers stopped | No edits. Run commands below, return exit codes, relevant output, precise tree fingerprint, scope/diff summary and omissions. Check no automatic network invocation; all public-network outcomes use fakes. Verify task test/vet stay on the seven supported-core packages. Failed checks are findings, not permission to change tests or filters | completed; all automated gates PASS |
| T07 | Fresh Gemini reviewer inspects production Go/tests against spec and security/compatibility boundaries | T05; frozen tree; final verdict consumes T06 evidence | Read-only generic Gemini assignment per plan §5. Review AC-01–06/08–10, bounded HTTP path, enum/trace compatibility, no startup network, no executable updater or clipboard controls, preserved commands/logs/non-TTY. Report findings with paths/lines, severity, acceptance IDs, evidence gaps and verdict. No fixes or ledger edits; do not claim manual AC-07 passed | completed; Ready, no blocking findings |
| T08 | Coordinator independently reviews Gemini-authored docs/shell, reconciles T06/T07 and prepares human checkpoint | T06 and T07 | Verify docs/scripts against integrated code without relaxing assertions; record manual walkthrough results or `not_run`. Address blockers through original authors and repeat affected gates/review on any changed snapshot. Write continuation/handover, set B1 `awaiting human review`, report actual models/fallbacks and request human acceptance. Never mark accepted without the owner | completed; human checks remain not_run |

## Verification commands and evidence

Focused names are reserved for the new tests above; preserve existing test names/assertions. Worker test-first failures must be reported as expected red-stage evidence, not passing completion. Use Go formatting on changed Go files before the frozen integration snapshot.

```sh
go test -count=1 ./cmd/agentic-sdd -run '^(TestBranding|TestUpdates|TestGuideBranding|TestGuideUpdates|TestBrandingSummary)'
task test
task vet
git diff --check
sh tests/guide-e2e.sh
sh tests/docker-e2e.sh
```

The task-defined aggregate is the supported-core selection, not `go test ./...`. Preserve sanitized environment settings. Containers use local images/read-only source and `--network none`; no public release API request, real-home guide/apply/restore, install/update script execution, live model/runtime evaluation, commit/push/tag or release. Unavailable gates are `blocked`/`not_run`; excluded validation is not a pass. No gate edits, skips or tolerances.

On the frozen tree, T06 and T07 can run in parallel because both are read-only; the final verdict references completed gate results. Fingerprint includes untracked new files, not just HEAD. Any changes invalidate affected evidence and require rerun/review before the checkpoint.

## Acceptance coverage

| Criteria | Delivery / verification |
|---|---|
| AC-01–04 | T02 helpers → T05 integration → T06 gates / T07 review |
| AC-05–06 | T01 baseline → T05 regressions → T04 container assertions → T06/T07 |
| AC-07 | T08 owner walkthrough, never automated or agent-invented |
| AC-08 | T02 entry → T05 integration → T06/T07; T08 human URL-following check |
| AC-09–10 | T03 leaf tests → T05 menu/trace integration → T06/T07; T04 offline container assertion |

## Human walkthrough record

- **Status:** `not_run`.
- **Disposable home / command:** pending; use an explicitly temporary `--home`, then quit without installation.
- **Variants:** static color; `NO_COLOR=1`; opt-in `AGENTIC_SDD_ANIMATE=1`; width 40; light/dark terminal themes; follow complete docs URL via terminal/browser.
- **Observed readability / motion / link limitations:** pending.
- No need to call GitHub or execute the displayed update script for this walkthrough; mocked automated tests prove the update result branches.

## Continuation

- **Current implemented batch:** B1; original T01–T08 and r5 corrections completed, awaiting human review. Draft expansion B2–B4 below has no execution authorization.
- **Approval reference:** spec §Approval; original B1 authorization 2026-11-08 and scoped animation amendment accepted 2026-11-09.
- **Changed paths:** plan §2's Go leaf/shared/test files, the two reference docs and guide container script; coordinator added `cmd/agentic-sdd/guide_plain_compatibility_test.go` to pin the captured pre-011 guide transcript. Spec approval/status, task ledger and session handover updated; no dependencies/gates/installer producers changed.
- **Checks:** T01 `task test` and `task vet` PASS against initial production HEAD `5ba1704a21a5244987dcce60308c19ae0f28a6e3`. Baseline binary/plain outputs and SHA-256 manifest: `/tmp/opencode/011-baseline/baseline-state.json`; full bytes captured for preview/help/version/usage error and real-PTY stdin with non-terminal guide stdout (`4`, `q`). Final automated gates and independent review PASS/Ready on the frozen snapshot below. Temporary files are local evidence, not portable shipped fixtures; human acceptance remains pending.
- **State:** awaiting human review — r5 UX correction verified.
- **Next permitted action:** Human combined review of spec r6 / plan r4 / tasks r4 and selection of a proposed batch; B1 visual/link checks remain pending. Do not implement the expansion, commit/push, install into a real home, run update instructions or publish without separate authorization.

## B1 execution evidence (2026-11-08)

- **Models:** T02 `opencode/deepseek-v4.1-flash-free`; T03/T05 `opencode-go/deepseek-v4.1-flash` (configured max effort); T04 `9router/ag/gemini-3.8-flash-high`; T06 `opencode/deepseek-v4.1-flash-free`; T07 fresh `general` review session using `9router/ag/gemini-3.8-flash-high`. No fallback used.
- **Test-first evidence:** T02/T03/T05 authored new tests before implementations and observed missing-symbol/field failures for the new APIs, then focused green runs. These initial compile failures establish missing APIs, not independent behavioral mutation testing. Existing tests were not weakened. Coordinator's legacy-output characterization test passed against the baseline source before integration.
- **Coordinator reconciliation:** corrected new width-40 shell assertions to require both wrapped tagline lines and absence of art; explicitly set width 80 for wide static checks; clarified update-trace privacy wording so existing `run.start` version metadata is not incorrectly described as removed. No existing journey assertion weakened.
- **Frozen verification snapshot:** `01501dcdff85bed4024f2f3d21208a80a057ea0d6395b2659c017ea5409d16e5`, 33 CLI/source/test/doc/gate/script files including untracked new Go files. Manifest `/tmp/opencode/011-final/frozen-state.json`; all hashes match before and after gates. Workflow ledgers and unrelated user ZIP excluded. This is a local content fingerprint, not a release or commit ID.
- **T06 gates PASS:** 36 matching focused tests, `task test`, supplementary uncached selection of the same seven core packages, `task vet`, `git diff --check`, `sh tests/guide-e2e.sh`, `sh tests/docker-e2e.sh`. Containers retained local image, read-only source and `--network none`; J7 exercised graceful offline failure only. Logs under `/tmp/opencode/011-final/`; coordinator checked `frozen-final.log` and `scope-diffstat.log`.
- **Compatibility PASS:** final binary versus pre-011 capture: preview/help/version/usage-error and real-PTY-stdin/non-terminal-stdout guide (`4`, `q`) have byte-identical stdout/stderr and identical exits. `/tmp/opencode/011-baseline/compare-plain.py` also asserts the disposable home was not created.
- **Environment limits:** local Go `1.27.1`, container Go `1.25`, module floor remains `1.23`; this run did not execute a Go 1.23 toolchain. No dependency or toolchain download. Human AC-07 and URL-following remain `not_run`; no public release API, actual update script, real-home operation, commit/push/tag or publication.
- **Automated smoke only:** coordinator captured static/narrow/animated guide output under `/tmp/opencode/011-final/{static,narrow,animated}.txt` using disposable homes and quit-only input. This is not the human walkthrough or browser/assistive-technology evidence.
- **T07 review:** Ready on the same frozen snapshot, incorporating completed T06 evidence; no blocking/high/medium findings. Source review covered tracked and untracked Go/tests without writes. Worker session `ses_e43d045bcfferYjWo7VwLa7VkJ` contains the verdict; validation session `ses_e43d086adffeUwCzS6vWpxYXfY` contains gate results. This is a Gemini technical review, not an Opus or runtime-certification claim. Existing run-start tool-version metadata is preserved; the review's privacy statement applies to newly added update-result events, not removal of pre-existing metadata.

## Current animation amendment (2026-11-09)

- **Owner feedback:** The local demo was run and the owner reported that the logo was not visibly animated. Screenshot: `/tmp/herdr-clipboard-images-1000/client-2-clipboard-1794185579165351404-0.png`. This is partial human feedback, not evidence of the complete light/dark/browser walkthrough. Initial 200 ms reveal met its timing contract but did not meet the owner's visual expectation.
- **Authorized correction:** One-second color sweep, then static logo before menu, retaining all suppression rules. Narrow exception permits carriage returns solely on the name line. Spec r4 / plan r2 reconcile the affected constraints; original scope and approval remain preserved.
- **Execution:** Main coordinator handles this bounded follow-up, rather than claiming the original parallel agents authored or independently reviewed it. No new delegation/model invocation. This task-specific follow-up assignment replaces the initial T02 implementation-agent assignment for the animation correction only; Go/TDD guidance remains applicable.
- **Changed implementation/tests:** `branding.go`, `branding_test.go`, `guide_branding_test.go`; canonical animation descriptions and container frame assertions updated. No other CLI behavior, dependency, gate selection or release operation changed.
- **Red/green:** Before implementation, the updated renderer and guide tests both failed because the old implementation produced two waits instead of ten; after implementation, focused branding/guide tests pass. Revised tests assert ten distinct advancing full-width highlights, exactly one second planned delay, SGR resets before waits, only name-line CR, unchanged settled display, unchanged suppressed bytes and no redraw after tagline/menu.
- **Current verification:** focused branding/guide tests (`-count=1`), `task test`, `task vet`, `git diff --check`, both offline container suites and original binary plain-output comparisons PASS. No excluded validation, real-home action, public release API request or update-script execution. The original snapshot/Gemini review above predates these changes (and subsequent Taskfile demo-task additions); it is not current-animation independent-review evidence. This bounded follow-up received coordinator-focused code/test review, not a fresh Gemini/Opus verdict.
- **Observed PTY timing:** 10 real highlight frames; first-to-last frame 0.904 s, first frame to menu 1.005 s, 10 name-line CR redraws, none after the tagline; exit 0 on `q`, no home created. Raw capture/result: `/tmp/opencode/011-sweep/animated-pty.bin`, `pty-result.json`. This measures emitted frames and timing, not a human aesthetic/accessibility verdict.
- **Current content fingerprint:** `035f8ba237117a290d6a63337051a23175bf88a593f2f05d30acad5b580b0751`, 34 CLI/test/doc/README/Taskfile/gate/script files, manifest `/tmp/opencode/011-sweep/frozen-state.json`; workflow ledgers excluded. This includes the separately requested local/container demo tasks already in the tree. Do not confuse it with the initial November 8 snapshot.
- **Human status:** Revised animation inspection and actual URL following remain pending; no human acceptance is recorded.

## Current UX amendment (2026-11-09)

- **Authority:** spec §Approval r5; owner's request to add the proposed fixes to the current spec and begin implementation, with Exa research.
- **Assignment:** Main coordinator, replacing the original parallel-lane assignments only for this bounded correction. Preserve all unrelated/uncommitted work; no commit/push or public API call.
- **Work:** one accurate safety notice; menu boundaries and descriptive labels; precise EOF/Ctrl+C text; development-build explanation; About first-use and restore/upgrade guidance; restore-impact warning; update progress and accurate fetched-metadata/unavailable/failure results. Retain owner-requested branding/animation; preserve legacy plain bytes, HTTP bounds, traces and write semantics.
- **Red stage:** `go test ./cmd/agentic-sdd -run '^TestGuideUX' -count=1` failed before production changes on missing menu/build/update explanation, absent restore warning, repeated broad no-write text and missing menu boundaries.
- **Verification:** `go test ./cmd/agentic-sdd -count=1`, `task test`, `task vet`, `git diff --check`, `sh tests/guide-e2e.sh` and `sh tests/docker-e2e.sh` PASS on the amended working tree. Existing renderer-label tests changed only for the specific copy replaced by r5; original plain compatibility tests unchanged. Added `guide_ux_test.go`; changed `guide.go`, `branding.go`, `updates.go`, three replaced-copy test expectations and the two reference docs. Installer producers, gate selection, dependencies and animation algorithm unchanged. No public API request or real-home operation.
- **Human checks:** remain pending; no fresh independent reviewer has assessed this amendment.

## Authorized r6 expansion — B2 through B4

**Execution override (2026-11-09):** Owner authorized all three batches with parallel Free/DeepSeek leaves and independent verification afterwards. Earlier draft-only/dependency wording below is retained as proposal history but does not block this explicit authorization. Current state: implementation starting; no batch delivery accepted yet. Coordinator owns the ledger. B2 inventory/shared integration: DeepSeek; B3 branding and B4 teaching catalog leaves: Free, disjoint paths; shared guide and explorer integration is serialized after leaves. Independent verification agent runs after writers stop. Clarifications: T09 typed missing-pack error vs invalid source; T11 specialized review-with-back helper; T10 always carries resolved home as already required.

Combined approval and the owner's explicit all-batches execution instruction are recorded only in [spec §Approval](spec.md#approval), tied to spec r6 / plan r4 / tasks r4 with review clarifications. The draft dependency rows below are superseded by the explicit execution override. Parallel leaf execution does not constitute batch completion or acceptance; shared integration is serialized and final verification follows it.

Assignments for the new work live in plan §7.6 (main agent by default, Go/TDD guidance, no new delegation). There are no task-specific specialist exceptions. Original B1 model assignments/evidence must not be presented as authorship or review of this draft.

### Batch B2 — Correct installation context, inventory and recovery

- **Tasks:** T09–T11.
- **Outcome:** Every target/command is context-correct; preview/receipt distinguishes desired inventory from actual changes; informed options and review-back work without new mutation paths.
- **Dependencies:** Combined r6 approval and explicit B2 authorization. Reconfirm B1/r5 tree and preserve existing WIP; B1 human visual checks are not fabricated or silently marked complete.
- **State:** `awaiting human review — r6 correction integrated and checked, 2026-11-10`.
- **Completion:** AC-12–14 and affected AC-05/06/11 tests pass, as do `task test`, `task vet`, `git diff --check` and affected existing offline journeys. No source asset/dependency/gate change. Finish with `awaiting human review` and exact tree/check evidence.
- **Next action:** Independent verification review, then owner review of the delivered B2–B4 experience. No next batch, commit/push or live operation is authorized.

| Task | Objective / likely modules | Dependencies | Acceptance and focused verification | Status |
|---|---|---|---|---|
| T09 | Pin current tracked/untracked baseline; add read-only selection inventory seam in existing `internal/skillsync/inventory.go` through existing source/target/validation/comparison helpers | B2 authorization | AC-12/14/15 foundation: fixture core/pack/preset/custom sources; matching/different/missing/refused target states; core works without optional pack; no writes, unrelated-file enumeration or symlink following. Test before helper implementation; `go test ./internal/skillsync` | completed (`sdd-implement-2`) |
| T10 | Add actual destinations, friendly labels, 80-column table/stacked fallback, source-derived unique/repeated counts and successful-result-based receipt/recovery in terminal guide; reuse context/quoting logic | T09 | AC-12/14: 19×4+18 embedded example plus one custom source, unchanged/replaced/no-op cases, spaces/apostrophes/non-ASCII paths; suggested read-only recovery preview has real ID/home/repo and never `--apply`; preview retains selected pack flags. Log/no-op rules unchanged. `go test ./cmd/agentic-sdd` | completed (`sdd-implement-1`) |
| T11 | Explain pack/Free-first options; add terminal-only `b` at final review and truthful log/error outcomes; label first-use examples; amend context/review/receipt docs and targeted offline script assertions | T10 | AC-13/17: `1 → options → b → changed options → y` writes only after second review; b/decline/EOF produce no installer writes; unsafe log paths stay errors. Legacy exact-output assertions unchanged. Decode b trace (no write approval). `task test`, `task vet`, both offline suites, `git diff --check` | completed (`sdd-implement-1`, coordinator docs/script) |

### Batch B3 — Stronger optional brand choreography

- **Tasks:** T12–T13.
- **Outcome:** Legible name and workflow motif, opt-in 1.5-second two-pass highlight, preserved static/plain compatibility, explicit demo cleanup disclosure.
- **Dependencies:** Combined r6 approval, B2 completion and explicit B3 authorization (serial integration avoids overlapping guide/branding/doc writers).
- **State:** `in progress — branding leaf delegated to Free`.
- **Completion:** AC-16 and affected AC-01–05/08/11/17 automated checks pass. Owner aesthetic/light-dark/link checks remain `not_run` until actually performed. Both offline suites and core gates pass; no release operation.
- **Next action:** Stop for B2 review before requesting/using B3 authorization.

| Task | Objective / likely modules | Dependencies | Acceptance and focused verification | Status |
|---|---|---|---|---|
| T12 | Test first, then change `branding.go` frame generator/renderer and static logo/motif; update only superseded animation assertions in branding and guide tests | B3 authorization | AC-16: 20×75 ms, forward/return movement, full-width frames, resets before waits, static/animated settled equivalence, no post-prompt redraw; disabled/nil wait yields zero waits. Check widths 40/60/80 and NO_COLOR/dumb/plain cases. No real sleeps in unit tests | completed (`sdd-implement-free-1`) |
| T13 | Disclose temporary-home/container cleanup in existing Taskfile demo tasks; reconcile animation/demo docs and offline frame assertions; capture optional real-PTY timing with disposable home and quit-only input | T12 | AC-16/17: local fixture task output makes cleanup explicit without CLI `/tmp` inference; static demo is understandable without color; scripts remain local-image/network-less. Core gates and both offline suites; PTY unavailable = `not_run`, not pass | completed (`sdd-implement-free-1`, coordinator PTY evidence) |

### Batch B4 — Optional beginner explorer and final integration

- **Tasks:** T14–T16.
- **Outcome:** Optional local skills/profile/concepts explorer and grouped installation explanation help beginners understand the toolkit without invoking models or changing files.
- **Dependencies:** Combined r6 approval, B2/B3 completion and explicit B4 authorization. Existing shared guide files have one writer; no mandatory parallel subagent work.
- **State:** `awaiting human review — r6 correction integrated and checked, 2026-11-10`.
- **Completion:** AC-15/17 and all applicable earlier acceptance checks pass; examples/roles match actual managed asset IDs; all core and offline gates pass on the final tree. Record human visual/learning walkthrough honestly; do not claim improved learning from passing static tests.
- **Next action:** Independent verification review, then owner learning/visual review. No next batch, commit/push or live operation is authorized.

| Task | Objective / likely modules | Dependencies | Acceptance and focused verification | Status |
|---|---|---|---|---|
| T14 | Add a small curated teaching registry (`guide_catalog.go`): all shipped skills/base/preset IDs, purpose groups, when/example/declared-intent caveats; generic unknown-ID fallback and concepts | B4 authorization | AC-15: asset-set equality catches missing/stale entries; exact IDs come from inventory, not registry enumeration. All 14+4 current profiles covered, no hardcoded count or permission/runtime/free claim. Custom source gets honest generic descriptions | completed (`sdd-implement-free-2`) |
| T15 | Add terminal-only menu 6 and plain category/list/detail state machine (`guide_explorer.go`); source/target statuses from inventory, receipt role-group explanation and context-preserving re-entry suggestion | T14 | AC-15/17: numbered selection/back/Enter/quit/EOF/invalid sequences; core-only, absent pack, custom ID and refused target fixtures; unknown installed files not listed. No compulsory post-success prompt, HTTP/model/shell action or source/user-config parsing beyond managed inventory. Existing guide sequences unchanged except explicit new choices | completed (`sdd-implement-free-1`, coordinator menu wiring) |
| T16 | Reconcile schema-1 additive IDs and learning/docs examples; extend one offline explorer journey; run final frozen-tree checks and completion review, then human checkpoint/handover | T15; all writers stopped | AC-12–17 plus original compatibility, logs and trace tests. Parse new traces and assert no profile IDs/content/paths/free text. Run `task test`, `task vet`, `git diff --check`, both offline suites. Review via main agent unless separately delegated; state whether independent review occurred. Leave manual learning/aesthetic/URL outcomes pending if not performed | awaiting independent verification |

## r6 acceptance map and continuation

| Acceptance | Proposed owner |
|---|---|
| AC-12 — actual target/context/recovery commands | T09/T10, integration T16 |
| AC-13 — informed options/back/error handling | T11 |
| AC-14 — distinct inventory/change counts and receipt | T09/T10 |
| AC-15 — optional learner profiles/skills/status | T09 read-only foundation; T14/T15 content/navigation |
| AC-16 — revised choreography/static fallbacks | T12/T13 |
| AC-17 — traces, demos, compatibility and integration | T11/T13/T15/T16 |

- **Approval:** r6 combined approval and all-batches execution recorded only at spec §Approval (2026-11-09).
- **Current code:** B1/r5 working tree retained without production changes in this drafting session. Original B1 and r5 check evidence does not prove proposed B2–B4 behavior.
- **Current action:** Execute disjoint Free/DeepSeek leaves, serialize shared guide integration, then dispatch the requested independent verification agent.
- **Product decisions:** the 1.5-second two-pass choreography, menu 6 explorer and final-review b navigation are approved for implementation via spec §Approval; final human review of the resulting experience remains pending.
- **Future continuation record:** after an authorized batch, record task IDs, exact changed tree, commands/exits/omissions, remaining findings and `awaiting human review`. Do not copy a prior model verdict or human acceptance onto a changed tree.
human review`. Do not copy a prior model verdict or human acceptance onto a changed tree.

## r6 execution evidence (2026-11-10)

- **Delegation actually used:** `sdd-implement-2` (DeepSeek) for read-only inventory T09; `sdd-implement-1` (DeepSeek) for terminal presentation/review T10–T11; `sdd-implement-free-1` (Console Free) for branding T12–T13 and the explorer leaf T15 (after completing T12 in an earlier assignment); `sdd-implement-free-2` (Console Free) for the teaching registry T14. No model fallback was used. The coordinator wired menu `6`, reconciled superseded terminal-copy assertions, owned docs/scripts/ledger and did not delegate approval.
- **Review-clarification implementation:** `skillsync.ErrPackMissing` distinguishes an absent optional pack from invalid/unreadable pack sources; the explorer offers Back for either, while installation still propagates the failure. A specialized `confirmWithBack` returns yes/no/back/EOF only for the final terminal review, leaving `prompter.confirm` and legacy non-terminal input semantics unchanged. Suggested commands keep the explicit resolved `--home` as FR-09 requires.
- **Automated checks on the integrated tree:** `go test ./cmd/agentic-sdd -count=1` PASS; `task test` PASS across the seven supported-core packages; `task vet` PASS; `git diff --check` PASS; `sh tests/guide-e2e.sh` PASS including new J8 (learn), J9 (review-back/decline) and J10 (context-preserving recovery preview) journeys; `sh tests/docker-e2e.sh` PASS.
- **Real-PTY smoke:** disposable home, quit-only input, opt-in animation; 20 name-line redraws, no redraw after the tagline/menu, exit 0, no home writes. This measures emitted frames/timing, not aesthetics or accessibility.
- **Known limits:** `gofmt -l` still reports the pre-existing unrelated `internal/skillsync/opencode_sync_test.go`; it was not reformatted. Human light/dark, narrow-width, URL-following and beginner-learning reviews remain `not_run`. The provisional Gemini architecture review predates this implementation tree and is not verification evidence for it. No public release API request, real-home installation, provider/model execution or commit/push occurred.
- **Remaining action:** independent read-only verification review, then human review. Delivered work is not accepted by this record.

## Post-verification corrections (2026-11-10)

Independent verification returned `Needs changes` with three small conformance gaps plus one cosmetic note. All were fixed and re-checked:

1. **Stage 3 heading (Medium):** `3. Installation result` now precedes success, no-op and failure result screens; new assertions cover it.
2. **About → explorer (Low):** terminal About now says to choose `6`; asserted.
3. **Quoting evidence (Low):** added an independent POSIX-style tokenizer test proving `guidePreviewCommand`/`guideApplyCommand` parse back to the exact home/repo values for spaces, apostrophes and non-ASCII paths, including selected pack flags. It does not execute the commands.
4. **Highlight padding (Info):** the sweep now stops within the name glyphs; the frame test asserts the highlight never leaves the name bounds.

Re-verified after these edits: `go test ./cmd/agentic-sdd`, `go test ./internal/skillsync`, `task test`, `task vet`, `git diff --check`, `sh tests/guide-e2e.sh`, `sh tests/docker-e2e.sh` PASS; real-PTY smoke again shows 20 redraws, no post-intro redraw and no home write. Human visual/link/learning review is still `not_run`; no commit/push, real-home install or public API call occurred.
