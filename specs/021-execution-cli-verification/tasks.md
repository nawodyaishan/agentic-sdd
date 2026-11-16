# Feature 021 — Tasks

**Revision:** r1, 2026-11-16 — draft for combined spec/plan/tasks approval.
**Parents:** [spec.md](spec.md) r1 (spec approved 2026-11-16), [plan.md](plan.md) r1.
**Approval reference:** [spec §11](spec.md#11-approval) — spec approved 2026-11-16; **combined spec/plan/tasks approval and Wave 1 authorization recorded 2026-11-16**. This file is the execution ledger, not an approval record.
**Assignments/tools:** plan §8; explicit owner-requested parallel fleet. No task-specific specialist exception.

## Batch ledger

| Batch | Tasks | Outcome | State | Next action |
|---|---|---|---|---|
| B1 | T01–T02 | Check/evidence CLI tests (G1–G4, G12) | `complete — awaiting human review` | Human review |
| B2 | T03–T04 | Recovery/clean/status tests (G5, G6, G9) | `complete — awaiting human review` | Human review |
| B3 | T05–T06 | Review package tests (G7) | `complete — awaiting human review` | Human review |
| B4 | T07–T08 | Record/flag/adapter tests (G8, G10, G11) | `complete — awaiting human review` | Human review |
| B5 | T09–T11 | Skill alignment + G14 | `complete — awaiting human review` | Human review |
| B6 | T12–T13 | Traceability matrix + G13 | `complete — awaiting human review` | Human review |
| B7 | T14–T17 | Integration, independent verification, evidence | `complete — awaiting human review` | Human review |

**Execution rule:** owner instruction 2026-11-16 authorizes parallel **disjoint-file** batches once the combined approval and explicit batch authorization are recorded. Wave 1 = B1–B6 simultaneously; Wave 2 = B7 serially after every Wave 1 lane stops. No two agents write the same file. A green check never advances a batch or creates approval.

## Fleet

| Lane | Agent | Primary model | Fallback | Owns |
|---|---|---|---|---|
| L1 | `sdd-implement-2` | `opencode-go/deepseek-v4.1-flash#max` | MiMo V2.6 Flash, MiniMax M3.1 Flash | `execution_cli_contract_check_test.go` |
| L2 | `sdd-implement-9r-2` | `9router/ag/gemini-4-hydrogen-xhigh` | Hydrogen High, then Go DeepSeek | `execution_cli_contract_recovery_test.go` |
| L3 | `sdd-implement-9r-1` | `9router/ag/gemini-4-hydrogen-high` | Gemini 4 Argon Medium | `execution_cli_contract_review_test.go` |
| L4 | `sdd-implement-1` | `opencode-go/muse-spark-1.3-contributor` | MiMo V2.6 Flash, MiniMax M3.1 Flash | `execution_cli_contract_record_test.go` |
| L5a | `sdd-docs-worker` | `opencode/deepseek-v4.1-flash-free` | `opencode/glm-5.3-flash-free` | `SKILL.md` |
| L5b | `sdd-implement-free-1` | `opencode/glm-5.3-flash-free` | `opencode/mimo-v2.6-flash-free` | `failure-and-reconciliation.md`, `routing.md` |
| L5c | `sdd-test-implement-2` | `opencode/beam-free` | GLM free, DeepSeek free | `execution_skill_alignment_test.go` |
| L6 | `sdd-implement-free-2` | `opencode/beam-free` | GLM free, MiMo free | `execution_contract_traceability_test.go`, matrix |
| V1 | `sdd-review-verification-1` | configured GLM | DeepSeek, MiniMax | Read-only independent verification |
| V2 | `sdd-code-review-9r` | `9router/ag/gemini-4-hydrogen-xhigh` | Hydrogen High, then Go DeepSeek | Read-only independent review (provisional) |

The parent confirms each child's actual model before launch and reports fallbacks. V2 is non-Opus and therefore provisional per the model-routing policy; human review remains required.

## B1 — Check/evidence CLI tests

**Completion:** G1–G4 and G12 pass at the CLI boundary; no production change; focused evidence recorded.

### T01 — Author check/evidence contract tests

- **Dependencies:** combined approval and explicit Wave 1 authorization.
- **Files:** new `cmd/agentic-sdd/execution_cli_contract_check_test.go` only.
- **Acceptance reference:** spec FR-02/FR-05, AC-02/AC-05; matrix G1–G4, G12.
- **Work:** implement `TestExecutionContractCheckTimeout`, `TestExecutionContractCheckFailure`, `TestExecutionContractCheckOutputLimit`, `TestExecutionContractCheckInterrupted`, `TestExecutionContractServiceEdges` using existing helpers. Timeout `250–300ms` with `sleep 5`; output cap asserts `output_limit`/`stdout_truncated`; signal Unix-gated; shell cases resolve `sh` via `exec.LookPath("sh")`; JSON structural assertions only.
- **Verification:** `go test ./cmd/agentic-sdd -run 'TestExecutionContract(CheckTimeout|CheckFailure|CheckOutputLimit|CheckInterrupted|ServiceEdges)' -count=1`.
- **Completion:** changed paths, commands/exits, any `not_run` recorded.
- **Stop:** a production fix is needed, a case cannot run hermetically, or the lock/check contract is ambiguous.

### T02 — Run focused check/evidence tests

- **Dependencies:** T01.
- **Files:** none.
- **Work:** rerun the T01 command with `-count=1`, capture exact exits, and report any flakes with the actual state.
- **Verification:** T01 command; no skip-to-pass.
- **Completion:** evidence note in the lane report; parent integrates in B7.
- **Stop:** unexplained flake or production defect.

## B2 — Recovery/clean/status tests

**Completion:** G5, G6 and G9 pass; deterministic lock/clean/status evidence recorded.

### T03 — Author recovery contract tests

- **Dependencies:** combined approval and explicit Wave 1 authorization.
- **Files:** new `cmd/agentic-sdd/execution_cli_contract_recovery_test.go` only.
- **Acceptance reference:** spec FR-01/FR-04/FR-05, AC-01/AC-04; matrix G5, G6, G9.
- **Work:** implement `TestExecutionContractWriterLock`, `TestExecutionContractCleanRefusals`, `TestExecutionContractStatusStale`. Lock holder uses `sh -c 'sleep 2'` or a FIFO; poll `.lock` + checkpoint `running`; issue the competing call while the holder is alive; join. Clean covers unowned/symlinked/moved/live-lock/absent. Status covers `head_changed`/`worktree_changed` and exit `0`.
- **Verification:** `go test ./cmd/agentic-sdd -run 'TestExecutionContract(WriterLock|CleanRefusals|StatusStale)' -count=1`.
- **Completion:** changed paths, commands/exits, `not_run` for absent `sh`/symlink setup.
- **Stop:** lock/clean behavior requires production change or the test cannot be made deterministic.

### T04 — Run focused recovery tests

- **Dependencies:** T03.
- **Files:** none.
- **Verification:** T03 command; repeat once to check determinism.
- **Completion:** evidence note; parent integrates in B7.
- **Stop:** repeated flake or production defect.

## B3 — Review package tests

**Completion:** G7 passes; partial reasons and no-false-complete behavior proven.

### T05 — Author review package contract test

- **Dependencies:** combined approval and explicit Wave 1 authorization.
- **Files:** new `cmd/agentic-sdd/execution_cli_contract_review_test.go` only.
- **Acceptance reference:** spec FR-03, AC-03; matrix G7.
- **Work:** implement `TestExecutionContractReviewPartialReasons` with `head_not_current` and `index_conflict` fixtures; assert exit `3`, `complete=false`, exact reasons, and no complete claim. Do not duplicate the existing sensitive-partial or complete-package cases.
- **Verification:** `go test ./cmd/agentic-sdd -run 'TestExecutionContractReviewPartialReasons' -count=1`.
- **Completion:** changed paths/commands/exits.
- **Stop:** requires a production change or unsafe Git fixture.

### T06 — Run focused review test

- **Dependencies:** T05.
- **Files:** none.
- **Verification:** T05 command.
- **Completion:** evidence note; parent integrates in B7.

## B4 — Record/flag/adapter tests

**Completion:** G8, G10 and G11 pass; strict claim, flag edges and adapter wiring proven.

### T07 — Author record/flag/adapter tests

- **Dependencies:** combined approval and explicit Wave 1 authorization.
- **Files:** new `cmd/agentic-sdd/execution_cli_contract_record_test.go` only.
- **Acceptance reference:** spec FR-01/FR-02/FR-05, AC-01/AC-02/AC-05; matrix G8, G10, G11.
- **Work:** implement `TestExecutionContractRecordValidation`, `TestExecutionContractFlagEdges`, `TestExecutionContractAdapterWiring`. `--result -` uses direct `run(..., strings.NewReader(...), false)`; invalid/oversize exit 2 with no artifact; flag edges assert exit 2, empty stdout, `error:` stderr; adapters cover `--direct` conflict, `--run-root`, feature/direct `--task`, invalid `--base`, review caps.
- **Verification:** `go test ./cmd/agentic-sdd -run 'TestExecutionContract(RecordValidation|FlagEdges|AdapterWiring)' -count=1`.
- **Completion:** changed paths/commands/exits.
- **Stop:** requires production change or a non-hermetic fixture.

### T08 — Run focused record/flag/adapter tests

- **Dependencies:** T07.
- **Files:** none.
- **Verification:** T07 command.
- **Completion:** evidence note; parent integrates in B7.

## B5 — Skill alignment

**Completion:** named skill edits applied; G14 passes; skillsync link test passes; no authority/state duplication.

### T09 — Align the orchestration skill table and exit bullet

- **Dependencies:** combined approval and explicit Wave 1 authorization.
- **Files:** `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` only.
- **Acceptance reference:** spec FR-06, AC-06; plan §9; matrix G14.
- **Work:** add `run record`/`run clean` rows (claim wording; preview/`--yes`); add one `--`/no-shell + exit `0/1/2/3/124` bullet; cite `docs/execution-runs.md` as backticked path text, **not** a markdown link. Do not add schema/flag-default/state-list detail.
- **Verification:** `go test ./internal/skillsync -run 'TestOpenCodePackAssetsAndPermissions' -count=1`.
- **Completion:** diff limited to the named file; link test passes.
- **Stop:** link test requires a non-skill change or the edit expands into a second contract.

### T10 — Align recovery and routing references

- **Dependencies:** combined approval and explicit Wave 1 authorization.
- **Files:** `opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`, `opencode-pack/routing.md` only.
- **Acceptance reference:** spec FR-06, AC-06; plan §9.
- **Work:** add the recovery block (foreign checkpoint, four drift reasons, byte-identical reissue vs `--force`, record/clean refusals) in “Checkpoints, stale state and interruption”; add one prose execution cross-reference to `routing.md`; no table row/backticked model there.
- **Verification:** `go test ./internal/appops -run 'TestAgentRoutingMatchesDocumentedPolicy' -count=1`; `go test ./internal/skillsync -run 'TestOpenCodePackAssetsAndPermissions' -count=1`.
- **Completion:** named files only; both focused tests pass.
- **Stop:** routing parser or skillsync link test fails beyond the allowed prose edit.

### T11 — Author and run the G14 skill-alignment check

- **Dependencies:** T09/T10 for the green run; may author the test in parallel.
- **Files:** new `cmd/agentic-sdd/execution_skill_alignment_test.go` only.
- **Acceptance reference:** spec FR-06/FR-07, AC-06/AC-07; matrix G14.
- **Work:** implement `TestExecutionContractSkillAlignment` with presence assertions (seven commands, `run record` claim, `not a sandbox`, exit tokens once, four drift reasons subset, `stale`/`blocked`) and absence assertions for affirmative sandbox/approval/resume forms and active legacy execution examples. Allow the wrapper note and denial wording.
- **Verification:** `go test ./cmd/agentic-sdd -run 'TestExecutionContractSkillAlignment' -count=1`; run after T09/T10; expected red before, green after.
- **Completion:** focused test green; no writes outside the test file.
- **Stop:** the assertion becomes brittle (full sentences/state list) or the skill edit is out of scope.

## B6 — Traceability

**Completion:** G13 validates matrix grammar, exact-once rows and real test names.

### T12 — Author the traceability matrix check

- **Dependencies:** combined approval and explicit Wave 1 authorization; test names frozen in plan §3.
- **Files:** new `cmd/agentic-sdd/execution_contract_traceability_test.go` only.
- **Acceptance reference:** spec FR-07, AC-07; matrix G13.
- **Work:** implement `TestExecutionContractTraceabilityMatrix` with `go/parser` over `./*_test.go`, `../../internal/execution/*_test.go`, `../../internal/skillsync/*_test.go`; read `../../specs/021-execution-cli-verification/test-case-matrix.md`; check exact test names, exact-once clause rows, slash-separated origin/`(stub)`, status vocabulary, no writes. Expected red until all names land.
- **Verification:** `go test ./cmd/agentic-sdd -run 'TestExecutionContractTraceabilityMatrix' -count=1`.
- **Completion:** test file written; red evidence recorded; B7 reruns green.
- **Stop:** parser requires a new package/dependency or writes.

### T13 — Update the matrix rows and statuses

- **Dependencies:** T12 (same lane; matrix is one writer).
- **Files:** `specs/021-execution-cli-verification/test-case-matrix.md` only.
- **Work:** verify every new test name, origin/status, exact-once 016 FR/AC, 017 AC, 019 clauses; mark G13/G14 `covered-static`/`new` as appropriate; keep `not_run` reasons.
- **Verification:** reread against `plan.md` §3 and T01–T12; G13 rerun in B7.
- **Completion:** matrix consistent; no duplicate/missing clause.
- **Stop:** a clause needs a design decision or a test name changes.

## B7 — Integration, verification and evidence

**Completion:** full gates green; independent reviews recorded; batch state updated; no acceptance inferred.

### T14 — Integrate and run the full gates

- **Dependencies:** B1–B6 complete; all Wave 1 lanes stopped.
- **Files:** none beyond scoped fixes required by integration; no new production behavior.
- **Work:** integrate lanes, resolve compile/format issues without changing contract, run:
  `go test ./internal/execution ./cmd/agentic-sdd -count=1`,
  `task test`, `task vet`,
  `go test -race ./internal/execution ./cmd/agentic-sdd -count=1`,
  `git diff --check`.
  Run `task verify` if the reviewed toolchain is available; otherwise record `not_run`.
- **Verification:** exact commands/exits, code state, changed paths, skips.
- **Completion:** B7 evidence note.
- **Stop:** gate conflict, production defect, or a lane's file ownership overlapped.

### T15 — Independent verification lane 1

- **Dependencies:** T14.
- **Agent/model:** `sdd-review-verification-1` (configured GLM; fallbacks per plan §8).
- **Files:** read-only.
- **Work:** run the check/evidence and recovery focused groups read-only; inspect the diff for contract accuracy, test weakness, hermeticity and false claims; report findings.
- **Verification:** commands/exits and findings in its report.
- **Completion:** verdict recorded; no edits.
- **Stop:** blocker/major finding returns to T14.

### T16 — Independent verification lane 2 (provisional)

- **Dependencies:** T14.
- **Agent/model:** `sdd-code-review-9r` with `9router/ag/gemini-4-hydrogen-xhigh` (provisional; fallback Hydrogen High).
- **Files:** read-only.
- **Work:** review the review/record/adapter tests, skill alignment and G13/G14; check compatibility, claims and `not_run` accounting.
- **Verification:** read-only commands/findings.
- **Completion:** provisional verdict recorded; Opus review remains required if treated as final.
- **Stop:** blocker/major finding returns to T14.

### T17 — Record completion evidence and state

- **Dependencies:** T15/T16.
- **Files:** `specs/021-execution-cli-verification/tasks.md`, `docs/handovers/2026-11-16_021.md` (and spec approval state only if the owner records it).
- **Work:** record commands/exits, changed paths, executed/`covered-static`/`not_run`, review verdicts, residual risks, and the next permitted action. Set batches to `awaiting human review`; no commit/release/apply.
- **Verification:** evidence cross-checked against T14–T16; no acceptance inferred.
- **Completion:** human review requested.
- **Completion (2026-11-16):** all gates green after review fixes; V2 provisional `Ready`, V1 `Ready with findings` with all findings resolved/dispositioned; B1–B7 set to `awaiting human review`. No commit/push/release/apply.
- **Stop:** any missing evidence or unresolved blocker.

## B7 evidence (2026-11-16)

**Wave 1 result:** all six lanes stopped; six new test files plus the named skill edits are present. No production Go code changed. L5c's file was repaired by the parent after the lane left trailing analysis text and an unsupported lookbehind; L4's direct-record fixture was corrected from `taskId: T01` to `taskId: direct` (test-construction bug). A generated untracked root binary from a lane's `go build` was removed.

| Check | Exit | Result |
|---|---|---|
| `go test ./internal/execution ./cmd/agentic-sdd -count=1` | 0 | Both packages pass |
| `go test ./cmd/agentic-sdd -run 'TestExecutionContract' -count=1` | 0 | All G1–G14 tests pass |
| `task test` | 0 | 11 supported-core packages pass |
| `task vet` | 0 | 11 packages clean |
| `go test -race ./internal/execution ./cmd/agentic-sdd -count=1` | 0 | Race clean |
| `git diff --check` | 0 | Clean |
| `go mod verify` | 0 | Modules verified |
| `task verify SDD_OUTPUT=/tmp/opencode/021-verify-2` | 0 | Six required checks pass incl. `build-darwin`; evidence `/tmp/opencode/021-verify-2` |

**Independent reviews:** V1 (`sdd-review-verification-1`, GLM) and V2 (`sdd-code-review-9r`, Gemini 4 Hydrogen XHigh, provisional) returned findings; both had shell permission denied and could not independently rerun tests. **V2 re-review after fixes: Ready (provisional); F1–F3 resolved with no remaining static findings.** **V1 re-review: Ready with findings; R1-021-1 and R1-021-2 fixed; R1-021-5 (matrix lock wording) resolved by the matrix edit; R1-021-3/R1-021-4 dispositioned.**

**Review fixes applied (2026-11-16):**
- G14 now asserts the 017 recovery rules against `failure-and-reconciliation.md` alone (not `docs`), asserts the dispatch handoff fields, scopes `run record`/`claim` to its table row, and scopes exit tokens to the exit bullet rather than headings.
- The record adapter test uses its own repo for the inside-repository run root and resolves `sh` via `exec.LookPath("sh")`, asserting the direct-check `TaskID`.
- The lock holder now sleeps 3s with a 2s observation deadline; G1/G3 skip on Windows with an explicit `not_run` reason.
- Gates rerun after the fixes: `go test ./internal/execution ./cmd/agentic-sdd`, `task test`, `task vet`, `go test -race`, `git diff --check` and `task verify` all pass.

**Not_run:** real `os.Exit`/built-binary/EPIPE, legacy `agentic-sdd` wrapper execution, ENOSPC, PTY/macOS/Windows runtime, live model/MCP/network, real-home apply, release/tag/tap/cask.

## Readiness reconciliation

1. **AC ownership:** AC-01 → existing extraction tests + T03 + T09; AC-02 → T01/T07; AC-03 → T05/T07; AC-04 → T03; AC-05 → T07; AC-06 → T09–T11; AC-07 → T12/T13; AC-08 → all lanes + T14; AC-09 → T14; AC-10 → T14/T17.
2. **Shared contract:** plan §3 freezes test names, exit/JSON invariants and skill constraints once; tasks reference it.
3. **Dependencies:** B1–B6 are disjoint-file parallel lanes; B7 is serial and depends on all. No shared writer.
4. **Tests:** each lane runs focused cases; B7 runs the full 11-package gate, race and diff check; V1/V2 independently rerun groups.
5. **Assumptions:** no production Go change; defects are findings; model fallbacks recorded; a skip is `not_run`.

## Continuation

- **Current stage:** B1–B7 implementation, integration and independent review complete; **awaiting human review**. A green suite and reviewer verdicts are not acceptance.
- **Baseline:** `ed68941a91c7a9aec4c6642ca8f85142c0a83fc2` plus unrelated Feature 020 working-tree changes.
- **Changed paths:** six new `cmd/agentic-sdd/execution_*_test.go` files; `opencode-pack/skills/agentic-sdd-orchestrate/{SKILL.md,references/failure-and-reconciliation.md}`, `opencode-pack/routing.md`; `specs/021-execution-cli-verification/{spec,plan,tasks,test-case-matrix,compatibility-audit}.md`; `docs/handovers/2026-11-16_021.md`. No production Go code.
- **Checks:** see §B7 evidence — all integrated gates and `task verify` passed after the review fixes.
- **Next permitted action:** human review/acceptance of Feature 021. No commit/push/tag/release/real-home/apply.
- **Guardrails:** no commit/push/tag/release/real-home/apply; no production Go change; named skill files only; keep 016/017/018/019 records unchanged; never add a skip or gate exclusion to make the suite pass.
