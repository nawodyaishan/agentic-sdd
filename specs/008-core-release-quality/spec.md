# Feature 008 — Core Product Quality, Repository Maintainability & Release Readiness

**Status:** Core-first amendment approved; macOS native runtime requirement dropped in favor of Linux CI / isolated Linux runner sufficiency; commit and push completed at checkpoint 7d3e99b (7d3e99b083f094ab4aba17e835bdcca39ee4abd5); tagging/releasing and trusted-reference designation remain separate decisions

**Date:** 2026-11-05

**Revision:** r2 approved, 2026-11-07 — Option 1 plus minimum Option 2; amended 2026-11-07 for exact core contract approval, commit/push authorization, GoReleaser snapshot, and Linux CI runner evidence sufficiency for AC-14/T10. Historical r1 approval and B1–B3 delivery remain scoped to their recorded revisions.

**Proposed repository path:** `specs/008-core-release-quality/spec.md`

**Baseline:** Public `v0.4.0`; November 5 read-only assessment of `main` at `0ee5aaf`; owner-supplied repository tree

**Next release:** Undecided `v0.x` version, subject to review and verification

**Authority:** [Approval](#approval) preserves prior batch/commit decisions and records the owner's combined approval of spec r2/plan r4/tasks r4, execution of B4a–B5, approval of the exact core contract, authorization of commit and push (completed at checkpoint 7d3e99b), authorization of GoReleaser install/snapshot, and the amendment dropping native macOS/Homebrew runtime evidence as a release qualification requirement (Linux CI runner / isolated Linux evidence sufficiency). Tagging, release publication, and trusted reference designation (G-008-01) remain separate decisions.

## 1. Product vision and outcome

Make the existing Agentic-SDD **supported core** a coherent, easy-to-adopt, maintainable and verifiably release-ready open-source product. Agentic-SDD is an **engineering learning-and-delivery framework** for users of coding agents: it improves requirements, implementation, debugging, verification and review while retaining human understanding and control. It is not a general-purpose autonomous multi-agent engine.

The intended audiences are (1) solo developers using OpenCode, Codex and Claude Code; (2) interns, undergraduates, junior and self-taught developers who want to learn professional engineering practices while delivering work; and (3) experienced backend, infrastructure, platform and SRE engineers and maintainers. These are **target-audience hypotheses**, not demonstrated adoption.

**Release outcome:** A new user can safely preview/install, distinguish direct fixes from features, establish scope on an incomplete request, review an authorized batch, understand verification, and find the documented support limits. Maintainers can understand the active status without reconciling stale summaries, run release-critical checks on a selected candidate and determine a justified release go/no-go.

The feature is successful only with evidence, not because a release was desired: a **documented no-go** is the correct outcome if any supported-core blocker remains. Publication always needs a separate human decision.

The canonical [vision §6](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#6-non-negotiable-engineering-principles), [§7](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#7-product-architecture-and-ownership-boundaries) and [§10/H0](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#10-roadmap-by-evidence-gates-not-hype-or-deadlines) guide this release-first amendment. Conditional H1–H3 aspirations are not acceptance scope; the vision does not override feature approval or execution authority.

## 2. Scope boundary and existing architecture

**Retain as supported-core candidates, subject to fresh release checks:**

- All **18 existing portable skills** and their conditional references, routing, direct fixes, optional teaching, debugging and reviews.
- One main agent; one combined spec/plan/tasks approval; one explicitly authorized batch; one completion review; stop for human continuation.
- `cmd/agentic-sdd`, `internal/skillsync`, embedded assets, preview-first installer, backup/staged replacement/rollback and restore.
- The **documented supported portion** of `sdd-check`, `harness/` check/evidence contracts and trusted CI, with platform and trust limitations accurately declared.
- Existing `Taskfile.yml`, `AGENTS.md`, source/release tooling and GitHub/Homebrew distribution paths.

**Experimental/unsupported in this release:** native OpenCode ACP model execution, automatic multi-model routing or billing firewall, general native independent reviewer filesystem isolation, automatic subagent orchestration, native macOS check-runner guarantees and cross-client behavioral equivalence. Feature 006 B4 remains blocked/deferred by the demonstrated `IR-06-01` symlink escape; no documentation improvement can certify that boundary.

**Amended default:** normal Task commands, hooks, required CI and release builds target the working supported core. Experimental features and their behavioral suites are inactive/excluded by default and separated from that command/dependency graph. Keep their source/history rather than deleting it blindly; an explicitly opt-in experimental task/build path is not approval to execute it. Classify by capability and effect, not old feature number: retained installer, check/evidence, metadata and current Taskfile integrity remain core even if introduced in 005 or 007. Core boundary tests may prove that experiments cannot activate; they do not run or certify the experiments.

Do not duplicate the existing `human-input.md`, `grilling`, `questionnaire`, `teach`, `implement`, `verification-review` or approval contracts. Keep portable methodology separate from host-specific runtime enforcement.

## 3. Source-grounded problems

The owner's November 5 read-only report recorded:

| ID | Finding | Release disposition |
|---|---|---|
| **PQ-01 / IR-06-01** | Native read-only reviewer escaped an intended exclusion through an allowed symlink | Keep generic native reviewer unsupported; disclose explicitly |
| **PQ-02** | Stale current-status statements in `AGENTS.md`, feature 005/007 tasks and 006 runbook | Reconcile current-facing text without rewriting historic decisions |
| **PQ-03** | Required six checks omit relevant Python/Taskfile/trusted-CI regression suites | Explicit release verification and safe test ownership |
| **PQ-04** | README's file-permission comparison language contradicts installer mode semantics | Establish and verify accurate user-facing contract |
| **PQ-05** | Budget-model reliability/cost advantage not measured; behavioral cases `not_run` | Do not advertise unproven benefits; use bounded scenario evidence |
| **PQ-06** | Newcomers lack one concise end-to-end worked path | Provide examples without creating a second workflow |

The submitted tree also shows many historical `docs/handovers/`, `audits/`, `docs/006-independent-review-spec.md`, `backups/`, `bin/`, mixed Python/shell tests, `007-taskfile-migration`, and root embed files. **Presence in `tree` does not establish tracked status, duplicate content, user ownership or safe removability.** The read-only assessment did not rerun tests or establish a clean candidate tree.

## 4. Functional requirements

### R1 — Clear supported product and release contract

Publish an accurate, easy-to-find support matrix distinguishing portable skills, CLI/restore, source-only tools, supported operating systems, GitHub/Homebrew artifacts and inactive OpenCode experiments. Installation into four client directories is **not** evidence of identical behavioral compliance. Document what is actually supported at `0.x`, not an unapproved `1.0.0` API promise. README, help, installed assets, docs and release notes must agree.

### R2 — Better human–agent interaction using existing skills

For incomplete prompts, the current router/human-input procedure must first inspect discoverable repository facts, reuse settled answers and isolate **consequential** human decisions. Ask at most three ready independent questions in a round; present recommendation, plausible alternatives and consequences where appropriate. Do not force an interview for a clear request. Distinguish established fact, assumption, decision, approval and authorization. Explain scope, exclusions and acceptance conditions at the appropriate checkpoint. Implement improvements by refining **existing** instructions where there is a demonstrated gap, not by introducing another mandatory skill, approval, form or ledger.

### R3 — Proportional implementation, debugging and review

Preserve the direct-fix route without feature paperwork; feature route with reconciled spec/plan/tasks; review-only route without mutation; optional teaching without turning a checkout into a course. Ensure implementation output communicates changed behavior, actual checks, limitations, risk-based review points and the next permitted action. One principal implementation completion review remains the owner; do not duplicate it in the router. No automatic review/fix loops or model escalation.

### R4 — Correct authority and state navigation

Reconcile misleading **current-facing** status statements in canonical project guidance and feature records, including 005/006/007, while preserving dated history and genuine approvals. Canonical combined approval stays in `spec.md` §Approval; batch continuation stays in `tasks.md`. Handovers and summaries are secondary pointers. A model's `Ready`, an accepted partial delivery, a green test, or bare `continue` must not authorize another batch or live operation. Existing 004/006 unresolved findings remain visible.

### R5 — Structure, navigation and repository hygiene

Perform a tracked/ignored/untracked path and provenance inspection **before changing** structure. Retain `cmd/`, `internal/`, `skills-files/`, `specs/`, `harness/` and `integrations/opencode/` as principal ownership boundaries. Improve discoverability of active guides versus historical reports, handovers and audits; a concise `docs/README.md` index or equivalent is preferable to a bulk move. Identify actual duplicate documents before consolidation. Review `tests/harness/` ownership and entry points; reorganize only if it yields defensible maintenance value with path/reference regression coverage.

Review `backups/` and `bin/` for tracked/generated/sensitive content and `.gitignore` correctness; **never discard, publish, or alter local user backups/artifacts merely because they appear in the tree**. Preserve all 18 skill names and linked references, necessary root Go embed assets, import paths, installation destinations and supported-core artifact behavior. R11's deliberate experimental-availability change needs its explicit compatibility/asset disposition rather than silent removal. Document migration/path impact and rollback; cosmetic conformity to a large Go template is not sufficient reason for change.

### R6 — Complete and trustworthy release verification

Create a candidate-specific verification matrix: Go tests/vet/format, skill/embed/reference integrity, installer/restore E2E, Taskfile regressions, relevant Python harness tests, evidence/check-runner tests, trusted-CI anti-weakening, release packaging and user first-run smoke. State exactly which checks run as required CI, separately isolated release QA, manual walkthrough, or `not_run`.

Preserve **trusted-base ownership** of CI bootstrap/check contracts. Align the six existing check roles to the reviewed supported-core selection rather than requiring literal `./...` breadth. Changes to required definitions need reviewed-base adoption; retain exact trusted/candidate comparison and meaningful independent core probes. Do not run candidate-controlled scripts on a privileged host or blindly move Python suites into the Go check contract. Separate isolated release QA from the mandatory CI gate. Negative cases must show real failures, missing coverage and candidate weakening are detected. A scoped developer pass is not an adopted required-gate pass.

### R7 — Installer permission/restore contract

Resolve PQ-04 by agreeing the user-facing semantics for file modes, no-op comparison, mode normalization, backup and restore. Prefer clarifying documentation to changing tested behavior unless a demonstrated correctness/safety issue requires implementation changes. If behavior changes, assess compatibility and verify idempotence, permission-only differences, failed writes and restore. Never claim exact mode preservation when the implementation normalizes modes. Preserve preview-first and backup-before-write invariants.

### R8 — Public onboarding and contribution readiness

Present a minimal safe first-five-minutes entry point and **three compact worked experiences**: (a) a direct fix with no feature paperwork; (b) a feature with short spec/plan/tasks, combined human approval and one batch; (c) a review/resume where acceptance does not authorize continuation. Explain the same workflow to learners and professionals, with teaching optional. Advanced runtime experiments belong behind explicit warnings. Ensure CONTRIBUTING and security-reporting guidance are navigable, accurate and proportional; do not add ceremonial files without a real use.

### R9 — Integrated Taskfile, release candidate and decision

Reuse B1's Taskfile provenance and verify the current maintained command/CI/release inputs; do not reopen 007's historical migration work. Ensure developer commands, test invocations, CI, GoReleaser metadata, generated artifacts and Homebrew instructions agree with the supported core profile. Choose a **clean, identified release candidate**. Validate packaging, embedded skills, metadata and the advertised macOS/Homebrew install path; distinguish Linux source usage and the unpublished `sdd-check` executable. Prepare factual release notes, limitations, exact candidate identity, compatibility disposition and a justified go/no-go. Version selection remains `0.x` unless a separately approved compatibility contract exists.

### R10 — Core-only defaults and separated experimental commands

`task test`, `task vet`, `task verify`, configured hooks and normal build/release commands must agree on the supported core. Keep `test-core`/`vet-core` only as compatible aliases or parity-enforced entry points, not competing scope owners. The existing required check contract owns mandatory selection; preserve the six roles, accurate evidence/failure semantics and reviewed-base trust protections while updating test/vet definitions deliberately. Experimental tasks belong in a separately invoked task file, not automatically included, discovered as ordinary core commands or depended on by core validation/release. No blanket Python discovery, skip-to-green behavior or hidden relaxation of core assertions. New/unclassified surfaces require scope review.

### R11 — Minimum technical default-inactive release boundary

Default core binaries must not activate experimental OpenCode operations, load their runtime backend or expose them as supported functionality. Attempted experimental actions return an explicit unavailable/non-success response before configuration, credential, provider or native-process effects; targeted help may explain unavailability without advertising a working integration. Prefer a narrow compile-time CLI boundary over a runtime/orchestration rewrite. Normal build/test/release commands must not inherit flags that silently enable experiments. Preserve core command behavior, all 18 skill identities and necessary inert schema/embed assets; no active experimental configuration may be emitted to client discovery. Retained inert resources require an explicit artifact disposition. Any future experimental build/task invocation remains separately authorized and unsupported, not part of 008's validation.

## 5. Explicit non-goals

- Completing Feature 006 B4 or treating the proposed sanitized-export C2 as certified.
- Running experimental suites merely because they are offline, re-opening old 005/006/007 implementation/adoption work, or implementing vision H1–H3. Only the minimal core-default disable/separation boundary and its core safety checks are added here.
- Enabling native ACP inference, launching paid/free model tests without separate authorization, general model routing, mandatory subagents or prompt-warming/caching infrastructure.
- Replacing Agentic-SDD with Superpowers, Spec Kit or a different development methodology.
- Creating a separate learner product, compulsory teaching, more approval records or a new current-state ledger.
- Promising budget-model reliability, lower token usage or human productivity gains without measured comparable observations.
- Certifying macOS native check execution, all client runtimes, hosted PR/fork isolation or enterprise security without corresponding evidence.
- Deleting historical evidence, unrelated work, backups or generated outputs without provenance and explicit authority.
- Tagging, pushing, publishing, updating a Homebrew tap, changing credential stores or applying skills to a real home merely because the feature is approved.

## 6. Acceptance criteria

| ID | Required observable result |
|---|---|
| **AC-01** | Current supported vs experimental/platform/artifact matrix is prominent, internally consistent, and grounded in source and actual candidate evidence. |
| **AC-02** | Existing feature, direct-fix, review-only, research, limited-document and teaching routes retain their proper authority and side-effect boundaries. |
| **AC-03** | Tested/inspected incomplete-request examples show fact discovery, scope establishment, relevant human questions, settled-answer reuse and no mandatory interrogation for clear requests. Static tests and actual model behavior are labeled separately. |
| **AC-04** | Negative scenarios for bare `continue`, stale handoff, `Ready` verdict, approval vs batch authorization and live/spend permission do not grant new work authority. |
| **AC-05** | Implementation/debugging/review guidance has one completion-review owner and concise, risk-relevant results, actual checks, known unknowns and next allowed action. |
| **AC-06** | PQ-02 current-status contradictions are removed or marked historical in their owning locations; old approvals/findings remain intact, including accepted partial 005 and open 004/006 findings. |
| **AC-07** | IR-06-01, unfinished 006 B4 and experimental OpenCode restrictions are visible in appropriate user-facing docs and candidate release notes; generic native review is not claimed safe. |
| **AC-08** | README/help, installer source and focused tests agree on permission comparison, normalization, no-op, backup and restore semantics; any behavioral change is explicitly reviewed for compatibility. |
| **AC-09** | A tracked/ignored/untracked structure inventory establishes the disposition of `bin/`, `backups/`, historical docs and mixed tests. No unreviewed loss of local or historical information. |
| **AC-10** | All 18 skill IDs and local references remain valid in source and embedded/installed distributions; any path moves have passing dependency/link checks and rollback information. |
| **AC-11** | A release verification matrix records exact suite, command, environment, candidate identity, result and whether CI/manual/not-run. Critical changed surfaces have fresh, relevant evidence; exceptions are explicit. |
| **AC-12** | Trusted CI invariants and negative/anti-weakening checks still hold; no candidate-controlled code is executed in a privileged trusted-base context. |
| **AC-13** | First-use installation and all three worked examples are repeatable with documented prerequisites and correct human stop points. Teaching remains opt-in. |
| **AC-14** | Release-candidate packaging, CLI metadata, Linux CI / isolated Linux first-run smoke, Darwin cross-build/package integrity, source-only tooling and distribution claims are verified for the **same exact candidate**; Linux CI runner / isolated Linux execution evidence is sufficient for candidate release qualification, while native macOS/Homebrew runtime execution remains unobserved (not pass). |
| **AC-15** | Model scenario `not_run` statuses and unmeasured cost/quality claims remain honest; evidence types are not conflated. |
| **AC-16** | Release report records exact commit/tree, known blockers, nonblocking experimental findings, supported compatibility scope, verification evidence, version recommendation, draft release notes and a reasoned **go/no-go**. |
| **AC-17** | Compatibility-sensitive changes are classified and have migration notes or are excluded; no automatic `v1.0.0` promise is made. |
| **AC-18** | Human acceptance of these artifacts does **not** start a later batch, spend models, modify real-user installs or authorize publication. Release action gets its own explicit approval. |
| **AC-19** | Default test/vet/verify, configured hooks and release commands select the same approved supported core; experimental tasks/suites are not selected directly or transitively. Core aliases cannot drift; excluded suites are not reported as passes. |
| **AC-20** | The default source build and release artifact have a verified inactive experimental CLI boundary: unsupported requests cause no config/credential/provider/native effects, ordinary help describes core support, and inherited build flags cannot silently enable experiments. All core commands remain functional. |
| **AC-21** | G-008-01 is closed only by fresh evidence for the reviewed/adopted core gate on identified inputs—not by this policy decision, a local alias, relaxed comparison or re-labeled old results. Base-owned negative controls still detect core no-op defects, missing/altered required selection and attempted default experimental activation, including candidate test/task weakening. |
| **AC-22** | Experimental commands/tests have an explicit non-default location and no dependency from normal tasks, hooks, CI or release. Preserved WIP/history is not activated, validated or advertised as core; boundary safety tests do not invoke experimental behavior. |
| **AC-23** | Core/experimental capability and dependency mapping covers compiled dependencies, assets and CLI availability. Compatibility/migration notes explain changed experimental availability; required core schemas/references, embeds and installation destinations remain intact. No broad package/module reorganization is required without new reviewed evidence. |

## 7. Release gate and defect disposition

**Release-blocking for the promised core:** broken or incomplete packaged skills, unsafe installer/restore regressions, materially misleading support claims, missing fresh verification of changed release-critical behavior, compromised reviewed-base CI boundaries, invalid release artifact/version metadata, or contradictory live-looking approval/continuation guidance with security impact.

**May remain open if prominently excluded:** IR-06-01 and 006 B4 generic native reviewer; disabled native ACP admission; unresolved hosted token/billing or compaction questions; budget-model comparative metrics; cross-client behavior equivalence; native macOS runner and native Homebrew runtime execution (per 2026-11-07 amendment, Linux CI runner / isolated Linux evidence is sufficient for candidate release qualification; Darwin cross-compilation and package integrity remain required, while native macOS execution remains unobserved, not pass; Linux local QA is NOT hosted CI observation); unproven divergent fork/PR route. A known critical defect in a shipped supported surface **cannot** be reclassified as experimental merely to obtain green status.

**Not a release gate:** cosmetic reorganization; learner adoption metrics; adding autonomous agents; completion of all historical feature 004/006 research. No finding may be silently waived; accepted residual risk must have a named scope and human decision.

G-008-01's policy direction is selected, but the existing machine gate remains unchanged and blocked until the reviewed implementation/adoption is verified. The release gate is the complete supported-core contract, not all experimental validations. A working experimental activation path in the default artifact, missing core coverage or a weakened trust boundary remains a core blocker. A green core gate is necessary, not sufficient: candidate-specific packaging, platform/first-use and other promised-core evidence are still required.

## 8. Structural target (illustrative, not a mandatory migration)

```text
README.md                   # concise public entry point
AGENTS.md                   # current repository engineering authority
CONTRIBUTING.md             # contributor entry point
Taskfile.yml                # developer commands / explicit release QA
cmd/                        # existing Go CLI entry points
internal/                   # existing Go implementation
skills-files/               # 18 portable skill directories; retain identities
specs/                      # feature approval and continuation owners
harness/                    # deterministic contract and trusted CI
integrations/opencode/      # clearly experimental native material
tests/                      # documented owners and run commands
docs/
  README.md                 # optional index: onboarding, reference, history
  handovers/                # historical pointers, not permission
```

Go's official module guide supports the current `cmd/` + `internal/` organization. A structure change needs a concrete benefit, actual path-impact map, proof of working Go embeds/imports and link resolution, and a safe recovery route; **renaming everything to fit a template is prohibited scope creep**.

## 9. Validation method and evidence integrity

- Start by observing the real Git status and current canonical `spec.md` approvals / latest `tasks.md` continuations. Preserve dirty and untracked work; don't assume the provided tree is the release source.
- Use model-free source and fixture checks for static rule coverage, failure behavior and package integrity. These do **not** establish that budget models follow instructions.
- Separate Go/Task/Python deterministic checks, script/native probes, manual first-run evidence and online model evaluations. An online comparison requires separate exact model, provider, privacy, token/request/time and credentials authorization, with no silent fallback or retry.
- Test bad inputs and negative paths, particularly unauthorized continuation, direct-fix ceremony regression, review-only mutations, broken links, restore failures and untrusted CI candidate weakening.
- Validate packaged and installed outputs using safe isolated homes and appropriate CI isolation. A home directory or Git worktree alone is not a security sandbox.
- Present `pass / fail / blocked / not_run` explicitly. Failing or unavailable relevant evidence blocks the corresponding *claim*, not automatically unrelated offline scope.

## 10. Design dependencies, unresolved decisions and risks

**Retained discovery decisions and remaining amendment dependencies; do not redo completed work:**

1. Reuse B1's provenance and verify changed current-core Taskfile/CI/release inputs, not old 007 transition records.
2. Reuse the tracked/ignored/untracked inventory and refresh only changed candidate inputs; never inspect private backup payloads to regenerate it.
3. Preserve B3's documentation-first permission contract/test evidence and obtain its review disposition; any behavioral change still needs separate justification/approval.
4. Map exact build, Homebrew tap, check-runner support and required released artifacts at the chosen release candidate.
5. Determine which additional checks are safe in ordinary CI and which belong to isolated release QA without compromising trusted-base isolation.
6. Decide any docs/test path moves only after referencing every consumer and migration impact.
7. Determine appropriate next `0.x` version from actual changes and compatibility; do not preselect it.
8. For this amendment, map default CLI/backend dependencies and task/hook/release graphs without running excluded suites. Verify the smallest compile-time disable boundary and the reviewed core-gate adoption sequence; do not use a candidate-controlled selector as trusted CI authority.

**Risks to mitigate:** accidental scope expansion into an orchestration rewrite; history presented as new consent; claim inflation from static test results; inadvertently weakening CI or permissions; unsafe cleanup of backups; broken installed skills due to path changes; release from a dirty/unidentified tree; premature 1.0 promise.

## 11. Completion, handoff, and release authority

At feature completion, provide a bounded change summary, user-facing before/after examples, path/compatibility inventory, candidate-specific test matrix, open findings and exclusions, exact release candidate, draft release notes, and a go/no-go judgment. Stop for human review. The owner may approve a narrowly supported `0.x` release **after** independently authorizing release/tag/tap actions; green CI or feature acceptance alone is not publication permission.

## Approval

This is section 12 and the sole combined-approval record for [spec r1](spec.md), [plan r1](plan.md) and [tasks r1](tasks.md).

**Amendment checkpoint:** spec r2 + plan r4 + tasks r4 combined review completed and approved on 2026-11-07; B4a–B5 executed. The 2026-11-07 documentation amendment records the owner's approval of the exact core contract, authorization to commit and push all reviewed changes, authorization to install GoReleaser and run snapshot, and the decision dropping native macOS/Homebrew runtime evidence as an AC-14/T10 release qualification requirement in favor of Linux CI / isolated Linux runner evidence sufficiency.

- **Historical combined r1 decision:** Approved, 2026-11-05.
- **Amendment decision:** Approved, 2026-11-07 (combined spec r2, plan r4, tasks r4); documentation amendment approved 2026-11-07.
- **Approver:** The repository owner in this conversation (no additional identity asserted).
- **Approved revisions/scope:** Combined r1 spec, plan and tasks as presented, retaining the owner's explicit strict-008-only scope and exclusions.
- **Decision evidence:** The owner said “i approve all 3 plan spec tasks” and requested use of `sdd-implement-1` and `sdd-implement-2`. After the response proposed recording approval and implementing B1 only, the owner switched out of Plan mode and said “start”. This authorizes that presented B1 scope, not B2–B5.
- **Authorized implementation batch:** B1 (T01–T02) is accepted; B2 (T03–T04) was authorized and committed. On 2026-11-06, after B3 tasks were presented, the owner said “also add task to ingore excluded scopes (005 006 007 blocked parts) tests validations in test runs” and “start B3”. Authorize T05 plus bounded T05a test selection under that exclusion fence, verify and stop. No separate B2 acceptance is inferred; this is actual new B3 authority. B4/B5 remain unauthorized.
- **B1 acceptance (2026-11-06):** the owner said “accept b1” after the delivered work, checks and residual issues were presented. This accepts the bounded documentation/source-inventory scope, not release readiness or excluded-feature behavior.
- **Post-B1 scope direction:** the owner required plan/tasks updates before B2 to exclude deprecated 006/007 scopes. Plan/tasks r2 make that fence explicit; the subsequent “start b2” applies to that presented scope. Core Taskfile/trust verification required by R6/R9 remains within 008; no old-feature resumption is authorized.
- **Commit authority (2026-11-06):** the owner requested existing uncommitted 006/007 changes be saved as WIP and selected “Commit B1 only” for the separate reviewed B1 delivery. Use scoped separate commits, preserve unrelated/private work, and do not push. Saving WIP does not approve, test, certify or resume those features.
- **B2 commit authority (2026-11-06):** the owner said “commit b2 and before start b3, give me tasks in b3”. Commit B2 only, preserve unrelated changes, and present B3's planned tasks. This does not authorize a push or B3 implementation, nor supply a separate B2 acceptance or release-readiness decision.
- **B3 scope/delegation (2026-11-06):** the subsequent “start B3” authorizes the presented installer contract work and the requested excluded-test selection task. The owner then requested parallel implementation and another subagent for temporary excluded scope in AGENTS.md: permit three bounded workers (isolated permission tests, README contract, append-only AGENTS scope). Parent owns test-selection implementation/integration/state and completion review. No B3 commit, push, experimental evaluation or live apply is authorized. Do not weaken the unchanged trusted required contract; record aggregate-gate conflicts explicitly rather than hiding excluded validations behind green skips.
- **Authorized delegation:** bounded, non-overlapping B1 work and the later three explicitly requested B3 workers described above, using configured models; parent integrates and verifies. This is development assistance, not authorization for experimental product model evaluations or native reviewers.
- **Authorized model/network/credential/live actions:** Only the explicitly requested development-agent delegation above; no product model evaluation, native dispatch, credential/configuration change, real-home apply or other live action.
- **Scoped commit and push authority (2026-11-06):** the owner said “and commit changes and push”. Commit and push only this session's reviewed Feature 008 scope: B3 permission/selection delivery, the core-first amendment drafts, and the latest-only handover documentation. Preserve unrelated work (including the inherited AGENTS spending-policy deletion and untracked `docs/vnext-project-status.md`). No tag, GitHub Release or Homebrew tap change.
- **Authorized tag, push, GitHub Release, or Homebrew tap change:** No tag, GitHub Release or Homebrew tap change. Destination/scope of the recorded push is the current `main` and its existing upstream only.
- **Core-first direction and document authority (2026-11-06):** after prioritizing the working release scope and canonical vision, the owner selected “Option 1 as the main approach, with the minimum parts of Option 2 needed to ensure experiments are actually inactive in the release” and requested edits to spec/plan/tasks. This selects the proposed policy and authorizes this reconciliation, not implementation or retrospective B3 acceptance. The later “resume” continues that document work only. No new delegation, commit, live trusted-base adoption or release action is authorized.

- **Amendment approval and B4a–B5 authorization (2026-11-07):** after the handover and the pending-review state were presented, the owner chose “Approve and start B4a and B4b” and followed up with “start again b4a using gemini flash subagent / also start b4b b4c b4d using same type of gemini flash agents or deepseek go agents / also if for less demand work use free model subagents” (the remaining batches in `tasks.md` are B4a, B4b, B4c and B5). This records combined approval of spec r2, plan r4 and tasks r4 and authorizes B4a (T06a/T06b), B4b (T06c/T07), B4c (T06/T07a/T08) and B5 (T09–T11) using `sdd-implement-9r-1` (Gemini 3.8 Flash High) or `sdd-implement-1`/`sdd-implement-2` (DeepSeek Go) for core code/test batches and `sdd-implement-free-1` / `sdd-implement-free-2` for lower-demand documentation/matrix work, respecting task dependencies and leaving any unmet live/clean-candidate/native/designated-reference prerequisites explicitly blocked rather than simulated green. No commit, push, tag, release, live trusted-base push, hook bypass, experimental suite/opt-in build, real-home apply, credential change or model evaluation is authorized. G-008-01 stays open until fresh core evidence and an actual designated trusted reference exist.

- **Core contract, commit/push, GoReleaser, and release requirement amendment (2026-11-07):** Condensed owner direction (bracketed text supplies context for “do it”):
  > *"i approve exact core contract / commmit changes / all changes approved to commit and push / [install GoReleaser and snapshot] do it / drop Native macOS / Homebrew evidence — AC-14 / T10 as a requirment, linux ci runner is enough"*

  This records actual human owner authorization with the following explicit scope and boundaries:
  1. **Core contract approval:** Approves the exact supported-core contract delivered in B4b (7-package selection in `harness/checks.json` with SHA-256 `f42851cc793fb384ce26ea3ecc428caf748a454396a2017892ab7e5029d49b17`, Taskfile core alignment, and default-inactive experimental CLI boundary).
  2. **Commit and push authority:** Authorizes committing all reviewed Feature 008 changes and pushing them to upstream `main`. Unrelated dirty/untracked files remain preserved as instructed.
  3. **GoReleaser snapshot authority:** Authorizes installing GoReleaser in the host environment and executing non-publishing snapshot validation (`task release`).
  4. **Release requirement amendment (AC-14 / T10):** Drops native macOS / Homebrew runtime execution evidence as a mandatory blocking requirement for candidate release qualification; Linux CI runner / isolated Linux execution evidence is sufficient. Darwin cross-build compilation and package integrity (Mach-O binaries, universal architecture build, embedded skills, ldflags, no experimental dependencies) remain strictly required. Native macOS runtime execution and Homebrew formula execution remain unobserved (not pass). Historical reports remain dated history and are not retroactively green. Linux local QA is NOT hosted CI observation; hosted CI observation on GitHub Actions remains required before public release.
  5. **No release or tag authority:** No authorization is granted to create a Git tag (`v*`), publish a GitHub Release, or push to the Homebrew tap.
  6. **Trusted reference designation:** Gate `G-008-01` remains open until a clean candidate commit is created and designated by the human owner as the reviewed trusted reference (the exact candidate SHA is not yet chosen).

- **Release authorization (2026-11-07):** The owner requested “release the version using tag” and selected `v0.5.0` after being informed that pushing a `v*` tag triggers GitHub Release and Homebrew tap publication and that cleanup is uncommitted, trusted-reference designation remains open, and review verdicts remain provisional. This authorizes committing the final cleanup, creating and pushing `v0.5.0`, and the resulting existing release workflow publication. It does not retrospectively mark missing checks, Opus review, independent reference designation, native macOS execution, or full-feature acceptance complete. Prior no-publication statements above describe earlier authorization boundaries.

The recorded decisions cover their stated revisions and scope. B1 completion review does not start B2.

## 13. References

**Primary project evidence:** owner-provided `agentic-sdd-context-pack.md` (2026-11-05); read-only product-quality assessment at `0ee5aaf` with PQ-01–PQ-06; owner-provided `tree -L 6` output; canonical repository files including `AGENTS.md`, `skills-files/agentic-sdd-router/references/{workflow-policy,human-input}.md`, `harness/{checks.json,acceptance.md,trusted-ci.md}`, `specs/004-*` through `007-*`, and `integrations/opencode/{usage,independent-review,safety}.md`. Snapshot descriptions are not fresh test runs.

**Core-first amendment sources:** canonical vision §§6/7/10/13; current CLI dispatch/help in `cmd/agentic-sdd/main.go`, the two helper entry points in `cmd/agentic-sdd/opencode.go`, inert `adapterfiles.go` embeds, `Taskfile.yml`, `lefthook.yml`, and trusted contract comparison/execution in `tests/harness/trusted_ci.py`. These support the narrow design; they do not establish runtime safety or adoption without fresh tests.

**External best-practice references (guidance, not verification of this repository):**

- Go module organization: https://go.dev/doc/modules/layout
- Go Task ordering and task dependencies: https://taskfile.dev/docs/guide/dependencies
- GitHub repository best practices: https://docs.github.com/en/repositories/creating-and-managing-repositories/best-practices-for-repositories
- Semantic Versioning: https://semver.org/
- Matt Pocock's productivity skills: https://github.com/mattpocock/skills/tree/main/skills/productivity
