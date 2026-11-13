# Feature 018 — Tasks

**Revision:** ACCEPTED r1, 2026-11-14 — human acceptance recorded; no further batch.
**Parents:** [spec.md](spec.md) (ACCEPTED v2; §6.1 evidence classification, §9 approval), [plan.md](plan.md) (ACCEPTED r1).
**Approval reference:** [spec §9](spec.md#9-approval) — **combined spec/plan/tasks approval recorded 2026-11-13; B1 and B2 authorized in parallel by owner decision; integrated result accepted 2026-11-14.** This file is the execution ledger, not an approval record.
**Execution rule:** the authorized B1/B2 lanes ran with disjoint file ownership, integrated in T04, verified, and stopped for human review; the owner accepted the integrated result on 2026-11-14. No commit, push, release, real-home apply, MCP install, permission change or paid model call is authorized by these documents.

## Batch ledger

| Batch | Tasks | Outcome | State | Next action |
|---|---|---|---|---|
| **B1** | T01–T02 | Compact research skill + one packaged reference; five stage pointers; precedence/privacy policy; body-only librarian parity | `accepted by owner 2026-11-14 — integrated gates green` | None for implementation; commit/release/live operations remain separately authorized. |
| **B2** | T03–T04 | Reference packaging assertion; dedicated static/fixture suite; integrated evidence and completion summary | `accepted by owner 2026-11-14 — integrated gates green` | None for implementation; commit/release/live operations remain separately authorized. |

**Why two batches:** B1 changes the portable instructions and policy prose; B2 tests the connected contract, adds the packaging assertion and records evidence. The owner authorized both in parallel on 2026-11-13 because their file ownership is disjoint. T03 authors tests against the frozen plan contract; T04 is the parent integration after the lanes.

## Execution lanes (2026-11-13)

These are the actual owner-authorized parallel lanes, with disjoint file ownership.

| Batch | Task | Executor used | Files |
|---|---|---|---|
| B1 | T01 | `sdd-implement-9r-2` with `9router/ag/gemini-4-hydrogen-xhigh` | research skill + reference |
| B1 | T02a | `sdd-implement-2` with `opencode-go/deepseek-v4.1-flash#max` | spec/plan/tasks pointers |
| B1 | T02b | `sdd-implement-free-1` (free model) | verification/debugging pointers |
| B1 | T02c | `sdd-implement-free-1` (free model) | workflow-policy + librarian body |
| B2 | T03a | `sdd-implement-free-1` (free model) | package assertion + README |
| B2 | T03b | `sdd-implement-free-2` (free Beam model) | Python suite + scenarios |
| T04 | Integration | Parent | evidence, ledger, handover |
| T04 | Review | `sdd-review-verification-1` read-only | no edits |

If a named lane is unavailable, report the gap; do not silently substitute a paid or different-provider model. The existing SDD implementation and review rules apply.

## B1 — Portable research methodology, stage pointers and policy precedence

**Completion condition:** the research contract is internally consistent; all five stage pointers return to their own stage without duplicating the method; the 008 exact clauses still pass; the 18-skill distribution, 009 permissions/model and 014/015/017/019 records are unchanged.

### T01 — Amend the research skill and add one packaged reference

- [x] **T01** Amend `agentic-sdd-research-spec` and add `references/search-mcp-workflow.md`.
  - **Dependencies:** combined spec/plan/tasks approval and explicit B1 authorization.
  - **Allowed files:** `skills-files/agentic-sdd-research-spec/SKILL.md`; new `skills-files/agentic-sdd-research-spec/references/search-mcp-workflow.md`.
  - **Acceptance reference:** FR-02, FR-03, FR-04, FR-08, FR-10, FR-11, FR-12; AC-03, AC-08, AC-10, AC-11, AC-14–AC-21 (static instruction coverage; runtime behavior remains `not_run`).
  - **Work:**
    - Keep `SKILL.md` compact (approximately 35–45 lines). Preserve verbatim: `Return to the interrupted stage after answering it` and `Never create feature documents just to hold research for a direct fix`.
    - Make the description trigger-focused on a named, consequential fact blocking an SDD stage; distinguish it from generic research, topic reports, background delegation and mandatory-file conventions.
    - Add the R0–R3 effort choice, bounded brief, local-first capability order, privacy/untrusted-content rule, seven-field handoff, approval preservation and no-automatic-artifact rule.
    - Add one conditional relative link to `references/search-mcp-workflow.md`.
    - Put the capability matrix, Exa/Context7 discipline, optional Tavily-style site tools, claim labels and stopping rules in the reference. Use capability-based wording; do not assert a provider, identifier, installation or permission.
  - **Verification:**
    - `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py` passes unchanged.
    - The two preserved clauses are present exactly; the new link resolves; no new skill directory or skill ID exists.
    - `git diff --check`.
  - **Integration responsibility:** T02 consumes the trigger and seven-field contract; T03 tests both.
  - **State:** complete 2026-11-13; accepted 2026-11-14.

### T02 — Wire five stages, policy precedence and optional librarian parity

- [x] **T02** Add the five stage pointers, the router-reference precedence/privacy rule and the librarian body note.
  - **Dependencies:** T01 (the trigger and handoff contract must be fixed first).
  - **Allowed files:** `skills-files/agentic-sdd-spec/SKILL.md`, `skills-files/agentic-sdd-plan/SKILL.md`, `skills-files/agentic-sdd-tasks/SKILL.md`, `skills-files/agentic-sdd-verification-review/SKILL.md`, `skills-files/agentic-sdd-debugging/SKILL.md`, `skills-files/agentic-sdd-router/references/workflow-policy.md`, `opencode-pack/agents/sdd-research-librarian.md` (body only).
  - **Acceptance reference:** FR-01, FR-05, FR-06; AC-01, AC-02, AC-05, AC-06, AC-07, AC-09, AC-12.
  - **Work:**
    - Add one or two sentences per stage, using relative links to the canonical research skill and the stage-specific return destination in plan §3.3. Do not repeat R0–R3, the matrix or the handoff.
    - Verification-review pointer: read-only; no file/state mutation; retrieved documentation is never executed test evidence.
    - Debugging pointer: named external/version uncertainty only; root-cause reasoning and the fix stay in the main agent.
    - Add two sentences to `workflow-policy.md` §Research routing: SDD precedence over generic `research` conventions; external-query privacy and untrusted fetched content.
    - Add the body-only librarian paragraph from plan §3.5. Do not change frontmatter, description, mode, model, `permissions:`, step count, fallback routing or the existing privacy rule. If the combined review rejects this paragraph, omit it and record that decision.
    - Do not edit `agentic-sdd-router/SKILL.md` or `references/specialists.md`.
  - **Verification:**
    - `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py` passes.
    - `go test ./internal/skillsync` passes after the reference link is in place (link/package checks).
    - Inspect the librarian diff: only body text changes; model/permissions/fallback are byte-identical.
    - `git diff --check`.
  - **Integration responsibility:** T03 validates the five pointers, policy sentences and librarian contract.
  - **State:** complete 2026-11-13; accepted 2026-11-14.

## B2 — Packaging assertion, deterministic fixtures and completion evidence

**Completion condition:** every AC has an owner and an evidence type; the dedicated suite fails when a required clause is removed or softened; the 008 manifest and core test remain untouched; no runtime, security-filter, model-routing or provider claim is made from static coverage.

### T03 — Add the packaging assertion and dedicated research suite

- [x] **T03** Add the reference packaging check and the explicitly selected Python static/fixture suite.
  - **Dependencies:** B2 authorization; the frozen contract in plan §3. T03 may author in parallel with T01–T02; parent reconciles wording at integration.
  - **Allowed files:** `internal/skillsync/package_test.go`; new `tests/harness/test_research_workflow.py`; new `tests/harness/research-workflow-scenarios.json`; `tests/harness/README.md`.
  - **Acceptance reference:** AC-01–AC-22 evidence coverage; especially AC-10, AC-11, AC-14–AC-18, AC-20, AC-21 and AC-22.
  - **Work:**
    - Add a focused package assertion that `agentic-sdd-research-spec/references/search-mcp-workflow.md` is embedded, regular, linked from the research `SKILL.md`, and included by the existing package snapshot. Keep the exact 18-skill list and all OpenCode profile/model/permission assertions unchanged.
    - Add a Python suite using only the standard library. Check required clauses and links in the research skill, reference, five stage skills, `workflow-policy.md` and librarian body; add mutation checks that removing or softening privacy, untrusted-content, snippet-non-promotion, approval-preservation, stage-return or docs-are-not-tests clauses fails.
    - Add the declarative scenario manifest: R0, R1, R2, version mismatch, contradictory/snippet-only claims, unavailable MCP, unauthorized `agent_run`/Tavily Research/hosted research, a mocked credential/private-path query, and a valid/invalid seven-field handoff. Every case records `agent_observation: "not_run"`.
    - Add the small positive/negative trigger fixture for the description, including a generic-`research` collision and a non-SDD request. Label it static lexical coverage with `runtime_activation: "not_run"`.
    - Document the explicit test command in `tests/harness/README.md`.
    - Do not edit `test_core_workflow.py` or `core-workflow-scenarios.json`; do not add an aggregate suite or a new dependency.
  - **Verification:**
    - `go test ./internal/skillsync`
    - `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py`
    - `python3 -B -m unittest discover -s tests/harness -p test_research_workflow.py`
    - `git diff --check`
  - **Integration responsibility:** T04 consumes the suite results and records the evidence.
  - **State:** complete 2026-11-13; accepted 2026-11-14.

### T04 — Integrated verification and completion evidence

- [x] **T04** Run the connected gates and record honest completion evidence.
  - **Dependencies:** T01–T03 integrated.
  - **Allowed files:** `specs/018-research-evidence-handoff/tasks.md` (ledger/evidence); `docs/handovers/2026-11-13_018b.md` (implementation-session handover); `internal/skillsync/mcp_compat_test.go` only for the integration correction below. No product edits in this task.
  - **Acceptance reference:** AC-22; plan §5 coverage matrix.
  - **Work:**
    - Run the §5.2 commands against the integrated state. Record each command, exit and evidence location.
    - Record changed paths, deviations, unresolved findings, and every `not_run` item (live MCP/model behavior, runtime redaction, skill activation, paid evaluation, real-home apply, release).
    - Apply the integration correction in `internal/skillsync/mcp_compat_test.go`: the no-MCP-state walk skips the managed client skill roots so the legitimate `search-mcp-workflow.md` reference is not treated as MCP state. Keep the seeded-config byte-identity and `.agentic-sdd/mcp` assertions intact; this is a false-positive fix, not a relaxation of the MCP contract.
    - Confirm unchanged surfaces: 18 skill IDs, 009 librarian model/permissions/fallback, 014/015/017/019 records, 008 package gate and check contract, Taskfiles.
    - Set the batch state to `awaiting human review` after implementation and the completion review; the owner then accepted the integrated result on 2026-11-14. A green check alone is not acceptance.
  - **Verification:** `task test`, `task vet`, the two Python suites, `go test ./internal/skillsync`, `git diff --check`, plus `sdd-review-verification-1` read-only review of the result. Re-review only changed/corrected work, not unchanged work.
  - **Integration responsibility:** parent owns the final evidence and next-action report.
  - **State:** complete 2026-11-13; accepted 2026-11-14.

## T04 evidence (2026-11-13)

**Integrated state:** all six implementation lanes landed; B1/B2 content and tests are present; the read-only review returned **Ready** on static content. No commit, push, release, real-home apply, MCP install or model call was made.

| Check | Exit | Result |
|---|---|---|
| `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py` | 0 | 11 tests OK; 008 clauses unchanged |
| `python3 -B -m unittest discover -s tests/harness -p test_research_workflow.py` | 0 | 22 tests OK; 19 scenarios + 6 trigger cases |
| `go test ./internal/skillsync` | 0 | package/link and MCP-compat tests pass |
| `task test` | 0 | all 11 supported-core packages pass |
| `task vet` | 0 | no findings |
| `git diff --check` | 0 | clean |
| `gofmt -l internal/skillsync/package_test.go internal/skillsync/mcp_compat_test.go` | 0 | empty output |
| `python3 -m json.tool tests/harness/research-workflow-scenarios.json` | 0 | valid JSON |

**Read-only review:** `sdd-review-verification-1` returned **Ready** on requirements and engineering content. Its shell access was denied, so its executed checks are `not_run`; the parent ran the commands above. Findings F-1 (ledger/handover) is addressed here and in `docs/handovers/2026-11-13_018b.md`; F-2 (optional narrower MCP-guard scoping) is accepted as-is because the seeded-config byte-identity and `.agentic-sdd/mcp` assertions remain; F-3/F-4 are informational; F-5 is resolved by the parent-run gates.

**Integration correction:** `internal/skillsync/mcp_compat_test.go` now skips the managed client skill roots in the no-MCP-state walk. The seeded OpenCode config byte-identity, `.agentic-sdd/mcp` non-existence and transcript/network tripwire assertions are unchanged. This is a false-positive fix for the legitimate `search-mcp-workflow.md` reference, not a relaxation of the MCP contract.

**Changed paths:** `skills-files/agentic-sdd-research-spec/SKILL.md`, `skills-files/agentic-sdd-research-spec/references/search-mcp-workflow.md` (new), the five stage `SKILL.md` files, `skills-files/agentic-sdd-router/references/workflow-policy.md`, `opencode-pack/agents/sdd-research-librarian.md`, `internal/skillsync/package_test.go`, `internal/skillsync/mcp_compat_test.go`, `tests/harness/README.md`, `tests/harness/test_research_workflow.py` (new), `tests/harness/research-workflow-scenarios.json` (new), and the Feature 018 records/handovers. No product Go code changed outside tests.

**Not run:** live model/MCP activation, live retrieval, dispatch-time redaction, provider permissions, paid evaluation, real-home `asdx-core apply`, release, commit and push. Static instruction/fixture coverage is not runtime proof.

**Unchanged:** 008 clauses and scenario manifest, 18-skill distribution, 009 librarian model/permissions/fallback, 014/015/017/019 records, Taskfile.yml and `harness/checks.json`.

**Residual risk:** the feature is verified as instruction and packaging coverage only. Live skill activation, MCP availability, retrieval and redaction remain `not_run` unless separately authorized.

## Human acceptance (2026-11-14)

The owner accepted the integrated B1/B2 result after reviewing the green gates and the read-only **Ready** verdict. Accepted scope: the research skill and packaged reference, five stage pointers, workflow-policy precedence/privacy, body-only librarian parity, packaging assertion, dedicated static/fixture suite, and the MCP-guard false-positive correction.

Residual limits remain explicit: live model/MCP activation, live retrieval, dispatch-time redaction, provider permissions, paid evaluation and real-home behavior are `not_run`. Acceptance does not authorize a commit, push, tag, release, real-home apply, MCP install or live model/MCP evaluation; those remain separate decisions.

## Readiness reconciliation

The single reconciliation required by the task contract is recorded here:

1. **AC ownership:** T01 owns the research-skill/reference behavior clauses; T02 owns stage routing, policy precedence, verification boundaries and approval preservation; T03 owns the deterministic evidence for all ACs; T04 owns the honest completion report. Every AC-01–AC-22 has a producer and a verifier.
2. **Shared contract:** the seven-field handoff is defined once in plan §3.2 and implemented in T01. T02 stage pointers reference it; T03 fixtures validate its shape. No second template is introduced.
3. **Dependencies:** T01/T02 produce the prose contract; T03 authors fixtures against the frozen plan contract in parallel; T04 integrates after all lanes. B1 and B2 were authorized in parallel by the owner on 2026-11-13; no further batch is authorized.
4. **Tests:** T03 checks required behavior, links, negative mutations, unavailable-capability handling and trigger boundaries. It does not alter the 008 core suite or claim runtime enforcement.
5. **Assumptions:** the body-only librarian paragraph is included; relative stage links are the selected format; no technical question remains.

## Open items

- **Human acceptance:** recorded 2026-11-14. No further implementation batch is authorized.
- **B1/B2 authorization:** granted in parallel by the owner on 2026-11-13; implementation accepted 2026-11-14.
- **Runtime evaluation:** any live model/MCP evaluation remains separately authorized and is `not_run` in this plan.
- **Residual risk:** static instruction/fixture coverage only; live activation, retrieval and redaction are unproven.
- **Commit/release/live operations:** not authorized by acceptance; each needs a separate decision.
