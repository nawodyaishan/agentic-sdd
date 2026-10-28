# vNext harness tasks and continuation

Revision: r1 (draft, 2026-10-28). Inputs: [spec.md r1](spec.md) and [plan.md r1](plan.md). Single approval reference: [spec.md §Approval](spec.md#approval). Specialist/tool assignments are in the plan; no task-specific exceptions.

## Execution rules

- All tasks below are **not started**. Planning and documentation checks are not implementation evidence.
- Execute one selected batch only after combined approval and its authorization. At its boundary, record checks, technical verdict, unresolved findings and `awaiting human review`; do not start the next batch automatically.
- Runtime inspection is sanitized observation; installation, paid comparisons, credential changes and real-home apply require separate authorization. An unavailable prerequisite remains blocked.
- Refresh source/diff and assignments before each batch. Material contract or scope changes return the affected draft sections for review.

## Ordered tasks

### T01 — Freeze baseline, runtime inventory and representative scenarios

- Status: **complete within scope (2026-10-28)**; baseline behavioral observations blocked (B1-F1), recorded honestly in [harness/baseline.md](../../harness/baseline.md).
- Covers: AC-02, AC-04, AC-11; establishes prerequisites for AC-03/10/12.
- Depends on: combined approval and B1 authorization.
- Contract: plan C2 and C6.
- Targets: new `harness/README.md`, `integrations/opencode/README.md`, `integrations/opencode/supported-version.md`, `tests/harness/` scenario inputs; sanitized baseline references in this task.
- Objective: record CLI/service exact versions and configuration provenance; identify installed skill precedence, actual model/tool availability, check prerequisites and isolation capabilities without copying secrets. Select exact fixture revisions and one unseen variant before tuning.
- Produces: pinned compatibility record, baseline input identities, scenario expectations, available/unavailable capability matrix and explicit authorization/cost gaps.
- Verification: reproduce source/config identity, inspect startup warnings, test a conflicting config/skill fixture, and run relevant existing deterministic checks in disposable storage. Record preexisting failures. Obtain authorized baseline behavioral observations before modifying the skills; if unavailable, freeze inputs and record the blocked observation rather than claim a baseline.
- Completion: knowns/unknowns are explicit; fixtures have independent expected outcomes; no live config is changed. Missing paid-run permission is not a request to run a benchmark implicitly.

### T02 — Freeze portable contracts and failure fixtures

- Status: **complete within scope (2026-10-28)**; schema/negative cases validated, semantic/runtime cases defined but NOT RUN (owned by T07/T08).
- Covers: AC-01, AC-05, AC-06, AC-09, AC-10, AC-11.
- Depends on: T01's identities and environment facts.
- Contract: plan C1–C6.
- Targets: `harness/README.md`, `harness/checks.json`, `harness/evidence/`, `tests/harness/` (new).
- Objective: define versioned check/receipt schemas and synthetic valid/invalid examples; document modes, authority, result meanings, retention and freshness. Capture check commands/prerequisites/effects, without creating an orchestrator.
- Consumes: T01's real checks and environment; produces: contracts used by T03–T11.
- Verification: reconcile command/result semantics, nullable observations, diff identity including untracked files, and external evidence paths. Fixtures distinguish missing prerequisites, failure, timeout, cancellation, skipped checks, unknown cost and stale Ready. Review fixtures for secrets and independent expectations.
- Completion: producers and consumers share one contract; unresolved runtime details have explicit blocked predicates. If T01 requires a material departure from the plan, revise and seek review before dependent implementation.

### T03 — Correct routing and executable planning handoffs

- Status: **complete within scope (2026-10-28)**. Canonical skills changed; controlled old-vs-candidate fixtures observed. No measured behavioral benefit was demonstrated, so the final adoption claim remains open (see `harness/baseline.md`).
- Covers: AC-01, AC-05, AC-08 (review ownership), AC-12.
- Depends on: T02 and T01's baseline observation prerequisite.
- Contract: C1; shared planning contracts from C6 scenarios.
- Targets: `skills-files/agentic-sdd-router/SKILL.md`, its `references/workflow-policy.md`, and spec/plan/tasks/research-spec skills; optional concise task-contract reference under the router.
- Objective: classify action before intake; reconcile latest authorization; return research to its invoking mode; make implementation the single completion-review owner; add acceptance/contract/integration ownership and readiness reconciliation.
- Verification scenarios: direct fix with no specs creates none; spec-only stops at spec; explicit B2 authorization crosses B1's review checkpoint but never enters B3; ambiguous continue does not invent scope; shared error/unit disagreement and missing verification owner are surfaced. Inspect actual outputs/diffs, not phrase matching.
- Completion: routing and task-contract scenarios are observed or honestly blocked; no new approval step or source-of-truth duplicate.
- Completion review: **Ready for batch review** (self-review). The controlled strong-model fixtures showed both old and candidate skills passing the tested authorization/reconciliation cases; no measured candidate benefit was observed, so final adoption remains open. The weaker model exposed non-deterministic bare-continuation/tool-schema behavior. One review correction made feature-scoped spec-only folder creation explicit.

### T04 — Add diagnosis and bounded review resolution

- Covers: AC-07, AC-08, AC-10, AC-12.
- Depends on: T03's lifecycle ownership and T02's result contract.
- Contract: C1, C5 and C6.
- Targets: implement and verification-review skills; new debugging skill; relevant drift-retro/bootstrap links; optional testing/review references; `README.md`, shared skill inventory, installer tests.
- Objective: meaningful testing, systematic diagnosis, feature review, finding disposition and scoped re-review; distribute the debugging skill with existing embedding/discovery.
- Verification scenarios: tautological expectation, setup-only regression failure, intermittent uncertainty, review-only logging suggestion, incorrect reviewer advice, a correction causing dependent regression, and passing batches with failing feature integration. Trace exactly one completion review for unchanged implementation.
- Packaging checks: embedded/on-disk discovery includes debugging, sibling links resolve, temporary-home install/restore/no-op preserves unrelated skills. Update explicit ten-skill inventories without falsifying historical audit documents.
- Completion: eleven canonical skills are discoverable, observed review/diagnosis behavior matches the contracts and existing installer compatibility holds.

### T05 — Build native profiles and reversible configuration adoption

- Covers: AC-02, AC-03, AC-12.
- Depends on: T01 pinned runtime, T02 contracts, T03 routing.
- Contract: C2.
- Targets: new `integrations/opencode/` native JSONC template and adoption/rollback instructions; `tests/harness/` configuration fixtures.
- Objective: define namespaced draft/build/review primary profiles and prompt-only current-session commands. Keep the template inactive until explicitly adopted. Implement the documented manual preview/backup/rollback procedure in temporary fixture exercises.
- Verification: config discovery/duplicates and ignored settings; allowed draft document edit versus denied production edit; review mutation attempts through editor, shell, MCP/resource access, browser and Code Mode (disabled routes must actually be unavailable); denied child launch; project allow versus global denial; invalid policy warning; no shell blocks in discovered commands. Static lint complements actual runtime denial probes, never replaces them.
- Adoption checks: existing config with unrelated fields survives; identical proposal does nothing; backups restore changed files; new files can be removed; symlink/special-file refusal; intervening edits prevent blind rollback.
- Completion: every advertised mode path has an observed boundary result. Unsupported routes remain unavailable; clean runtime/isolation prerequisites are documented.

### T06 — Validate model, tool and budget compatibility

- Covers: AC-04, AC-03, AC-11.
- Depends on: T05, T01's access inventory and any separately required paid-run authorization.
- Contract: C2; emits C4 capability observations.
- Targets: `integrations/opencode/` compatibility/tool records and `tests/harness/` negative scenarios.
- Objective: map Economy/Standard/Deep to verified access, test actual command/agent/session selection and bound fallback, expose only needed integrations with explicit destinations/effects/timeouts.
- Verification: incompatible tool model fails before use; unavailable model stays blocked or uses only the permitted alternative; no expensive fallback; missing MCP, malformed arguments and truncated output remain unresolved; non-idempotent timeout requests reconciliation. Cost absent/delayed is unknown, not zero; limit exhaustion stops new paid work.
- Completion: tiny authorized smoke tests establish capabilities before comparisons. No model/provider has been selected by guesswork, no server connected implicitly by a template, and unknown billing cannot produce a claimed hard cap.

### T07 — Implement the deterministic check runner

- Covers: AC-06, AC-09, AC-10 (owned checks).
- Depends on: T02's check/result contracts and T01's baseline commands.
- Contract: C3 and C4.
- Targets: new `cmd/sdd-check/`, `internal/harnesscheck/`, `harness/checks.json`; `Makefile`; Go tests using subprocess fixtures.
- Objective: validate and execute argv-based scoped checks with explicit prerequisites, timeouts, full logs, structured outcomes and owned-process cancellation. Reuse authoritative existing commands and retain direct Make targets.
- Verification: a passing fixture, actual nonzero failure, missing executable, invalid schema/duplicate IDs/path escape, skipped remaining checks, output truncation markers, deadline and process-tree cancellation. A focused success must never claim required completion. Test Linux/macOS behavior or mark an untested host unsupported.
- Gates: `make test`, `make vet`, `git diff --check`, followed by one required-contract run against the relevant final state. No recursive runner invocation from its own Go test suite.
- Completion: same check contract can execute without any model/provider access; baseline/focused/required evidence is distinguishable; the runner never changes approval or sets Ready.

### T08 — Validate receipts, redaction and freshness

- Covers: AC-09, AC-10, AC-04 (unknown cost), AC-06 (reuse).
- Depends on: T07 outputs and T02 schema; T06 supplies actual model metadata when available.
- Contract: C4.
- Targets: `harness/evidence/`, `internal/harnesscheck/` receipt validation/output as needed, `tests/harness/` evidence fixtures and adapter export instructions.
- Objective: correlate native observations and check output without a second ledger. Validate schema and identity; keep logs external and sanitize before export.
- Verification: missing session/cost values carry reasons; direct fix has no invented batch; staged, unstaged and untracked changes invalidate affected evidence; changed config/environment invalidates reuse; post-format output is reidentified. Secret-canary input is absent from exported artifacts. Review receipts do not touch the target checkout. Invalid receipt claims cannot manufacture approval or a check result.
- Completion: evidence can trace a result to actual changed content; reusable evidence has explicit freshness scope; retention and manual cleanup preserve retained acceptance artifacts.

### T09 — Prove interruption, compaction and resume

- Covers: AC-01, AC-03, AC-08, AC-09, AC-10.
- Depends on: T04–T08 and authorized observed runtime access.
- Contract: C1, C2, C4 and C5.
- Targets: adapter recovery procedure, `tests/harness/` recovery scenarios, canonical skill pointers only where needed.
- Objective: connect native lifecycle behavior to current task state and evidence using one main agent, without a new headless runner/plugin.
- Verification: interrupt a permitted fixture check; request native cancellation and observe owned activity stopping; preserve unrelated processes and partial files. Resume after compaction/restart/location move with dirty files and stale evidence. An explicit B2 authorization is respected, while no authorization blocks further writing. Uncertain external effects remain visible and are not replayed.
- Completion: original authorization survives recovery, no unfinished check becomes a pass, no force cleanup or duplicate review occurs. Stream closure alone is rejected as cancellation evidence.

### T10 — Connect trusted deterministic CI and weaken-the-gate tests

- Covers: AC-06, AC-03, AC-11, AC-12.
- Depends on: T07/T08; T01 identifies the trusted baseline and CI bootstrap constraints.
- Contract: C3.
- Targets: `.github/workflows/ci.yml`, `Makefile`, `tests/harness/`, CI trust documentation in `harness/README.md`.
- Objective: execute trusted required commands against a disposable candidate without LLM/provider credentials or publish privileges; retain existing Go/build checks.
- Verification: candidate empties a Makefile target, removes a required check, changes a receipt to pass, or deletes the known defect's coverage. The trusted gate or explicit review fixture detects the weakening. Candidate network/credential access is constrained; missing prerequisites fail visibly. Compare local/CI contract outcomes on the same candidate.
- Completion: the CI bootstrap/reference choice is explicit and tested. If the trusted branch does not yet contain the runner, record the required adoption action; do not represent an untrusted candidate's runner as the trusted gate. No branch-rule administration or publication occurs.

### T11 — Connected feature validation and adoption comparison

- Covers: all ACs; integration owner for AC-01–AC-12.
- Depends on: T01–T10, fresh required evidence, any separate comparison authorization and trusted CI bootstrap.
- Contract: C1–C6.
- Targets: `tests/harness/` scenario outcomes; sanitized baseline/comparison summary in `harness/`; README/version/rollback pointers; this task's continuation evidence.
- Objective: show one coherent feature lifecycle from draft through approval, one authorized batch, checks, review, interruption and safe resume; compare against the frozen baseline.
- Verification: rerun affected acceptance scenarios and the unseen variant, include critical repetitions within limits, aggregate failed attempts/cost/interventions, and check all twelve ACs against observed evidence. Run final installer compatibility/container checks and trusted deterministic gate on the final candidate.
- Completion: feature-wide self-review returns Ready only with fresh acceptance/boundary evidence and useful observed benefit without lost quality. Lack of runtime access, cost authorization, trustworthy baseline or CI adoption leaves the applicable acceptance blocked, not waived. Report unresolved findings and proposed integration action; do not merge/deploy.

## Execution batches

The adoption plan's R0–R2 are release stages, not a requirement to fit this work into three oversized execution batches.

| Batch | Stage / tasks | Outcome and completion evidence | State | Next action |
| --- | --- | --- | --- | --- |
| B1 | R0: T01–T02 | Frozen baseline, supported-runtime facts/unknowns, check/result contracts and fixture expectations. No canonical skill tuning before baseline observation. | **approved by human** (2026-10-28). Contract artifacts delivered; technical verdict **Blocked** on missing behavioral baseline, isolated native fixtures and CI bootstrap (B1-F1/F2/F4). | B1 findings remain visible; bounded provider access and Docker image pull authorized. B2 authorized after collecting the baseline. |
| B2 | R1: T03 | Action-first routing and implementable planning contracts; observed scope/handoff scenarios. | **awaiting human review** (2026-10-28). Canonical skills changed; controlled comparison completed. Technical verdict **Ready for batch review**, with an open adoption finding: no measured candidate benefit. | Human review; authorize B3 separately if desired. B3 native-profile work is not authorized. |
| B3 | R1: T05–T06 | Native modes, reversible adoption and model/tool compatibility; observed denials and selection tests. | not started | After B2 review, authorize B3; obtain any needed runtime/paid smoke-test permission separately. |
| B4 | R1: T07 | Runnable deterministic check contract, negative execution tests and preserved Make entry points. | not started | After B3 review, authorize B4. |
| B5 | R2: T04 | Diagnosis, meaningful testing and bounded review; debugging distribution and behavior evidence. | not started | After B4 review, authorize B5. |
| B6 | R2: T08–T09 | Receipts/freshness plus observed interruption, native cancellation and resumption. | not started | After B5 review, authorize B6. |
| B7 | R2: T10 | Trusted CI contract and weakened-gate detection; bootstrap dependency explicit. | not started | After B6 review, authorize B7. |
| B8 | R2: T11 | Connected feature acceptance, baseline comparison and final self-review. | not started | After B7 review, authorize B8 and separately authorize paid comparisons if needed. |

Every batch runs focused checks plus repository-required gates for its changes, reports omissions and stops at `awaiting human review` when its authorized work ends. A technical Blocked/Needs changes verdict is retained alongside that state. The default order above avoids overlapping ownership; dependency edges do not authorize skipping human batch boundaries.

## Acceptance ownership and readiness reconciliation

| Criterion | Implementation owners | Verification owners / decisive scenarios |
| --- | --- | --- |
| AC-01 | T02, T03, T09 | T03 action/authorization cases; T09 resume; T11 connected lifecycle |
| AC-02 | T01, T05 | T01 provenance; T05 conflict/adoption/rollback; T11 |
| AC-03 | T05, T06, T10 | T05 all advertised routes and isolation; T09 cancellation; T10 CI boundary; T11 |
| AC-04 | T01, T06, T08 | T06 capability/fallback/cost; T08 unknown observations; T11 |
| AC-05 | T02, T03 | T03 incompatible contracts and missing owner; T11 |
| AC-06 | T02, T07, T10 | T07 subprocess failures; T08 freshness; T10 weakened gate; T11 |
| AC-07 | T04 | T04 test-quality and diagnosis scenarios; T11 |
| AC-08 | T03, T04, T09 | T04 one review/finding resolution; T09 stale verdict; T11 feature integration |
| AC-09 | T02, T07, T08 | T08 identity/redaction/no-ledger cases; T09 recovery; T11 |
| AC-10 | T04, T07, T08, T09 | T07 owned processes; T08 dirty identity; T09 native lifecycle; T11 |
| AC-11 | T01, T06, T10, T11 | T01 frozen baseline; T06 limits; T11 comparison and compatibility evidence |
| AC-12 | T04, T05, T10 | T04 distribution; T05 adapter separation; T10 existing gates; T11 |

Draft reconciliation completed:

- All criteria have implementation and verification owners. T11 owns connected behavior, not merely aggregation of batch pass labels.
- C1 defines outcomes once; C3 produces check observations consumed by C4/T08 and C5/T09. No result can create approval.
- Skill edits are ordered T03 before T04; adapter depends on T03. T08 follows check output; T09 follows both runtime and workflow changes. CI follows the stable check/evidence contract.
- Native adapter configuration is a separate opt-in manual procedure, not an extension of skill-only backup manifests. No production config is installed while developing it.
- Behavioral evidence and static validation are explicitly different. Baseline, billing authorization, host isolation and CI trust are unresolved execution prerequisites with owners, not invented successful results.
- Remaining design discoveries that alter the contracts must return for review. Exact IDs/versions are implementation discoveries in B1, not a license to change authority or scope.

## Continuation record

- Current stage: B2 executed 2026-10-28; **awaiting human review** at the batch boundary. B1 was approved by the human before B2 began.
- Current batch/tasks: B2 / T03 complete within scope. B3 (T05–T06) not started and not authorized. Technical verdict: **Ready for batch review** — the batch outcome is implemented and verified; open adoption finding: controlled fixtures showed no measured improvement, and the weaker model did not consistently follow bare-continuation semantics. Neither finding blocks human review of B2 but both remain visible for final adoption.
- Approval reference: [spec.md §Approval](spec.md#approval), covering spec r1/plan r1/tasks r1; B1 and B2 authorized; B2 only most recently.
- Changed paths in B2: `skills-files/agentic-sdd-router/SKILL.md`, `skills-files/agentic-sdd-router/references/workflow-policy.md`, `skills-files/agentic-sdd-router/references/task-contracts.md` (new), `skills-files/agentic-sdd-spec/SKILL.md`, `skills-files/agentic-sdd-plan/SKILL.md`, `skills-files/agentic-sdd-tasks/SKILL.md`, `skills-files/agentic-sdd-research-spec/SKILL.md`, `tests/harness/workflow-scenarios.json` (new), `harness/baseline.md`, `integrations/opencode/README.md`, plus this file and the spec approval section.
- Preexisting user material: untracked `docs/Agentic-SDD-vNext-Adoption-Plan.md`; preserved unchanged (SHA-256 `b1eb0e75138acb6e1781086e183a456e4eb9f08a22cbc802e8b62da9451649cc`).
- Checks against this code state (2026-10-28): `make test` PASS; `make vet` PASS; `python3 tests/harness/validate_contracts.py` PASS; skill Markdown local-link/whitespace check PASS (13 skill Markdown files, 3 router references); `git diff --check` clean. `make docker-e2e` PASS after the explicitly authorized image pull. Controlled provider fixture evidence and observed cost are recorded in [harness/baseline.md](../../harness/baseline.md); fixture definitions: [tests/harness/workflow-scenarios.json](../../tests/harness/workflow-scenarios.json).
- Open findings carried forward: B1-F3 truncated native skill catalog; B1-F4 CI bootstrap trusted reference; B1-F5 runtime capability/cancellation untested; native policy/config boundary fixtures remain not run; no measured benefit from B2; weak-model workflow/tool-schema failures. B1-F1 behavioral baseline and B1-F2 Docker e2e prerequisite are resolved within their stated limits.
- Next permitted action: human review of B2. Do not start B3, install runtime config, spend further, or apply to a real home without explicit authorization.
