# Feature 021 — Plan

**Revision:** r1, 2026-11-16 — draft for combined spec/plan/tasks approval.
**Spec:** [spec.md](spec.md) r1, approved 2026-11-16 (spec only). Plan/tasks are pending combined approval.
**Companions:** [test-case-matrix.md](test-case-matrix.md), [compatibility-audit.md](compatibility-audit.md).
**Authority:** no implementation, commit, release or real-home action is authorized by this plan alone.

## 1. Decision and inputs

**Approach:** add the closed `G1`–`G14` verification suite and the named orchestration-skill alignment, with **no Go production-code change**, no new package, dependency, Taskfile entry or 008 gate change. Reuse the existing `cmd/agentic-sdd` test helpers and fixtures. G13 is structural traceability; G14 is static skill alignment.

**Spec refinement for parallel execution:** the spec's initial “one new file” wording is refined to six disjoint test files in `cmd/agentic-sdd`: four `execution_cli_contract_*_test.go` files plus `execution_skill_alignment_test.go` and `execution_contract_traceability_test.go`, all in `package main`. This is a file-layout refinement to satisfy the owner's parallel-fleet constraint, not a scope or behavior change. It remains one suite, one package, no gate/dependency change, and is part of this plan for combined approval.

**Inputs:** 016 spec/plan/tasks and `b3-acceptance.md`; 017 spec/tasks; 019 D-03/D-04/D-10; `docs/execution-runs.md`; the orchestration skill and references; the four-feature compatibility audit; the DeepSeek skill-gap study; the Gemini Argon spec review (provisional).

**Baseline:** `ed68941a91c7a9aec4c6642ca8f85142c0a83fc2` plus the unrelated Feature 020 working-tree changes. Feature 021 files are new; do not touch 020.

## 2. Architecture and file ownership

| Lane | New/changed files | Cases | Writer |
|---|---|---|---|
| **L1 — check/evidence** | `cmd/agentic-sdd/execution_cli_contract_check_test.go` | G1–G4, G12 | `sdd-implement-2` |
| **L2 — recovery** | `cmd/agentic-sdd/execution_cli_contract_recovery_test.go` | G5, G6, G9 | `sdd-implement-9r-2` |
| **L3 — review package** | `cmd/agentic-sdd/execution_cli_contract_review_test.go` | G7 | `sdd-implement-9r-1` |
| **L4 — record/adapter** | `cmd/agentic-sdd/execution_cli_contract_record_test.go` | G8, G10, G11 | `sdd-implement-1` |
| **L5a — skill table** | `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` | G14 content | `sdd-docs-worker` |
| **L5b — skill recovery** | `opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`, `opencode-pack/routing.md` | G14 content | `sdd-implement-free-1` |
| **L5c — skill check** | `cmd/agentic-sdd/execution_skill_alignment_test.go` | G14 | `sdd-test-implement-2` |
| **L6 — traceability** | `cmd/agentic-sdd/execution_contract_traceability_test.go`, `specs/021-execution-cli-verification/test-case-matrix.md` | G13 | `sdd-implement-free-2` |
| **B7 — integration** | No product edits beyond scoped fixes; evidence in `tasks.md`/handover | all | parent + reviewers |

**Rules:** one writer per file; no agent edits another lane's file; every lane reuses existing helpers (`executionTestRepo`, `runExecutionCLI`, `executionGitOutput`, `decodeRunIdentity`, `initExecutionIntegrityRun`) rather than adding a second fixture builder. New test names are frozen in §3 and the matrix.

## 3. Shared contract

### 3.1 Exact test names

| Case | Test |
|---|---|
| G1 | `TestExecutionContractCheckTimeout` |
| G2 | `TestExecutionContractCheckFailure` |
| G3 | `TestExecutionContractCheckOutputLimit` |
| G4 | `TestExecutionContractCheckInterrupted` |
| G5 | `TestExecutionContractWriterLock` |
| G6 | `TestExecutionContractCleanRefusals` |
| G7 | `TestExecutionContractReviewPartialReasons` |
| G8 | `TestExecutionContractRecordValidation` |
| G9 | `TestExecutionContractStatusStale` |
| G10 | `TestExecutionContractFlagEdges` |
| G11 | `TestExecutionContractAdapterWiring` |
| G12 | `TestExecutionContractServiceEdges` |
| G13 | `TestExecutionContractTraceabilityMatrix` |
| G14 | `TestExecutionContractSkillAlignment` |

### 3.2 Behavioural invariants

- Exit: timeout `124`; failure/interrupted/output-limit/`not_started` `1`; usage/lock/conflict/unsafe-path `2`; partial `review package` `3`; `run status` `0` even when `stale`/`blocked`.
- JSON: `schemaVersion == 1`; exact `kind`; ID grammar; required non-empty fields; ordered reason sets; no JSON object on stderr-path usage errors.
- `run record` is a claim: `passed` yields `awaiting-verification`, never `verified-evidence`.
- `run status` is an observation: no approval, resume or next-batch meaning.
- `review package` is `complete: false` with named reasons when content is omitted.
- G13 is structural only (symbol existence + exact-once rows + vocabulary); G14 is static content only.
- No security/sandbox/reviewer-isolation, live model/network, real `os.Exit`, legacy wrapper, archive/cask or non-Linux claim.

### 3.3 Skill contract

- `SKILL.md` gains `run record` and `run clean` table rows, plus one exit/`--` bullet. `run record` must say **claim**.
- `failure-and-reconciliation.md` gains a recovery block for foreign checkpoint, four brief-drift reasons, byte-identical reissue vs `--force`, and record/clean refusals.
- `routing.md` gains a prose-only installer/execution cross-reference.
- Do not add a `../../../docs/` markdown link in `SKILL.md`: `internal/skillsync/opencode_pack_test.go` resolves `SKILL.md` links inside `opencode-pack/`/`skills-files/` only. Cite `docs/execution-runs.md` as backticked path text.
- No new skill, no state-machine/approval duplication, no non-orchestration skill edits, no historical-record rewrite.

## 4. Parallel execution model

- **Wave 0 — approval:** the owner approves `spec.md` + this plan + `tasks.md` together and authorizes the parallel wave. No agent starts before that.
- **Wave 1 — parallel disjoint lanes:** B1–B6 run concurrently because their files are disjoint. Each lane writes only its files, runs its focused cases, and reports changed paths/commands/exits.
- **Wave 2 — serial integration:** B7 runs after all Wave 1 lanes stop. The parent integrates, runs the full gate/race/diff checks, and dispatches two read-only verification lanes in parallel.
- **G13/G14 expected red:** G13 depends on all test names existing; G14 depends on the skill edits. They may be authored early and are expected to fail until their dependencies land. That is recorded, not treated as a lane failure.
- **No simultaneous writers on one file.** Parallelism is by file ownership, not by shared-edit merging.

## 5. Batches

| Batch | Tasks | Outcome | Completion condition |
|---|---|---|---|
| B1 | T01–T02 | Check/evidence CLI tests | G1–G4, G12 pass; no production change |
| B2 | T03–T04 | Recovery/clean/status tests | G5, G6, G9 pass |
| B3 | T05–T06 | Review package tests | G7 passes; partial reasons/no false complete |
| B4 | T07–T08 | Record/flag/adapter tests | G8, G10, G11 pass |
| B5 | T09–T11 | Skill alignment | Named edits + G14 pass; skillsync link test passes |
| B6 | T12–T13 | Traceability | G13 passes after all names/matrix rows land |
| B7 | T14–T17 | Integration, review, evidence | Full gates green; reviews recorded; batch state updated |

## 6. Verification

**Per-lane focused commands:**

```sh
go test ./cmd/agentic-sdd -run 'TestExecutionContract(CheckTimeout|CheckFailure|CheckOutputLimit|CheckInterrupted|ServiceEdges)' -count=1
go test ./cmd/agentic-sdd -run 'TestExecutionContract(WriterLock|CleanRefusals|StatusStale)' -count=1
go test ./cmd/agentic-sdd -run 'TestExecutionContractReviewPartialReasons' -count=1
go test ./cmd/agentic-sdd -run 'TestExecutionContract(RecordValidation|FlagEdges|AdapterWiring)' -count=1
go test ./internal/skillsync -run 'TestOpenCodePackAssetsAndPermissions' -count=1
go test ./cmd/agentic-sdd -run 'TestExecutionContractSkillAlignment' -count=1
go test ./cmd/agentic-sdd -run 'TestExecutionContractTraceabilityMatrix' -count=1
```

**B7 integrated:**

```sh
go test ./internal/execution ./cmd/agentic-sdd -count=1
task test
task vet
go test -race ./internal/execution ./cmd/agentic-sdd -count=1
git diff --check
```

`task verify` may be run when the reviewed toolchain is available; `task verify-trusted` is not required for this test-only change unless the owner requests it. A skip is `not_run`, not a pass. Evidence records commands/exits and the code state.

## 7. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Parallel file conflict | One writer per file; lanes list exact paths; parent integrates |
| G13/G14 red before dependencies land | Expected; author against frozen names/content, integrate in B7 |
| Skill link-test failure | No markdown link to `docs/`; backticked path text; skillsync focused test |
| Lock-test race | Deterministic `sleep`/FIFO holder; poll `.lock` + `running`; join |
| Timeout flakiness | 250–300ms margin with `sleep 5`; no tight deadline |
| `sh` absent | `exec.LookPath("sh")`; skip as `not_run` |
| Free-model limits | Keep files small; Beam Free fallback for long context |
| Model/provider failure | Use the documented fallback order; report actual model/reason; stop if blocked |
| Test reveals a defect | Record a finding with a reproducible case; no silent production fix |
| Gate/docs parity | No production/struct/help change; tests only plus named skill docs |

## 8. Fleet, specialists and tools

**Model routing:** explicit owner-requested overrides; the parent confirms each child's actual model before launching and reports fallbacks. Go agents use `opencode-go/`; free agents use `opencode/`; 9Router agents use `9router/`.

| Lane | Agent | Primary model | Fallback |
|---|---|---|---|
| L1 | `sdd-implement-2` | `opencode-go/deepseek-v4.1-flash#max` | MiMo V2.6 Flash, MiniMax M3.1 Flash |
| L2 | `sdd-implement-9r-2` | `9router/ag/gemini-4-hydrogen-xhigh` | Gemini 4 Hydrogen High, then Go DeepSeek (authorized exception) |
| L3 | `sdd-implement-9r-1` | `9router/ag/gemini-4-hydrogen-high` | Gemini 4 Argon Medium |
| L4 | `sdd-implement-1` | `opencode-go/muse-spark-1.3-contributor` | MiMo V2.6 Flash, MiniMax M3.1 Flash |
| L5a | `sdd-docs-worker` | `opencode/deepseek-v4.1-flash-free` | `opencode/glm-5.3-flash-free` |
| L5b | `sdd-implement-free-1` | `opencode/glm-5.3-flash-free` | `opencode/mimo-v2.6-flash-free` |
| L5c | `sdd-test-implement-2` | `opencode/beam-free` | GLM free, DeepSeek free |
| L6 | `sdd-implement-free-2` | `opencode/beam-free` | GLM free, MiMo free |
| B7 review 1 | `sdd-review-verification-1` | configured GLM | DeepSeek, MiniMax |
| B7 review 2 | `sdd-code-review-9r` | `9router/ag/gemini-4-hydrogen-xhigh` | Hydrogen High, then Go DeepSeek (authorized exception) |

B7 review 2 is **provisional** because it is non-Opus; it does not replace an Opus verdict or human acceptance.

**Specialists:** no additional domain specialist is required. The work is repository-local Go testing and agent-consumed Markdown; use `golang-pro` only if an installed copy is available and a Go design question arises. **Tools:** CodeGraph for source navigation; Git and Go test for fixtures; `task test`/`vet`; no external search, browser, Context7, MCP or network call. Use `agentic-sdd-research-spec` only if a new named factual blocker appears.

## 9. Skill alignment detail

- **SKILL.md:** add the two table rows from spec §6.2, the one exit/`--` bullet, and backticked `docs/execution-runs.md` path text. Keep §4 a “when to use” table; do not turn it into a spec.
- **failure-and-reconciliation.md:** insert the recovery block in “Checkpoints, stale state and interruption” (not “Failure categories”); pointers only; four drift reasons as a subset.
- **routing.md:** add one prose line after the CLI block; no table row or backticked model name.
- **No edit:** `dispatch-contract.md` (already covers handoff fields), `context-and-cost.md`, `agentic-sdd-implement`, `-verification-review`, `-debugging`, `workflow-policy.md`, `independent-review.md`, historical records.

## 10. Decisions and assumptions

- Defect handling: **report-only**; a failing case is a finding.
- G11/G12 stay in scope.
- G13 is a structural test, not doc-only.
- Traceability granularity: FR/AC rows plus clause sub-rows only where currently unverified.
- G4 uses self-signal `kill -TERM $$`; FIFO/helper is allowed if deterministic.
- G14 lives in `execution_skill_alignment_test.go`; G13 lives in `execution_contract_traceability_test.go`; all in `package main`.
- Non-orchestration skills stay CLI-free.
- `dispatch-contract.md` needs no edit.
- No production Go behavior change.

## 11. Next document

Draft `tasks.md` with the seven batches, exact task/file ownership, parallel wave schedule, fleet assignments, per-task verification and continuation state. Then present spec, plan and tasks together for one combined approval.
