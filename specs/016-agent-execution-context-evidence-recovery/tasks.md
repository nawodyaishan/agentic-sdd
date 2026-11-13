# Feature 016 — Tasks (r1, 2026-11-13)

**Parents:** [`spec.md`](spec.md) (DRAFT r1) + [`plan.md`](plan.md) (r1).
**Approval:** Combined spec/plan/tasks **approved by the owner on 2026-11-13**; B1–B3 are authorized under the dependency-safe wave schedule below. Approval record: [spec §14](spec.md#14-approval). No real-home, product model/network/MCP, or release action is authorized.
**Execution rule:** One integrated code state per wave; parallel writers only on disjoint files. After a wave, record evidence and continue only within the authorized schedule. Do not start a real-home apply, product model/network call, or release.
**Baseline:** commit `a8f1c1c` (2026-11-13); re-verify in T01. The working tree contains only this feature folder.
**Boundary:** Offline, isolated fixtures only (`t.TempDir()`, disposable Git repositories, `GOPROXY=off`, `GOWORK=off`, `GOTOOLCHAIN=local`). No real `~/.config/opencode`, real home, private Git, model/network/MCP call, paid inference, trusted-CI adoption, tag, cask/tap push, or release.
**Rule:** `excluded` and `not_run` are not `pass`. A skip, filter, tolerance, or exclusion is never added to make an aggregate suite pass. A gate conflict stops for a human decision.

## Batch ledger

| Batch | Included tasks | Scope | State | Next action |
|---|---|---|---|---|
| **B1** | T01–T07 | Typed execution foundation, private workspace, run lifecycle, task briefs, CLI/help/JSON, gate registration | `awaiting human review — full 11-package gate green` | Review the integrated CLI/workspace/brief result. |
| **B2** | T08–T11 | `run check`, full log capture, limits/signals, checkpoint integration, stale/interrupted recovery | `awaiting human review — full gate and race green` | Review check evidence, interruption, and stale handling. |
| **B3** | T12–T17 | Review package, severity/request-only reviewer guidance, docs, six-group verification, AC report, handover | `awaiting human review — implementation complete` | Review the acceptance report and the actual diff; no acceptance inferred. |

## Parallel execution authorization (2026-11-13)

The owner directed implementation of all batches using Gemini Flash, DeepSeek max, and free implement agents. This is reconciled with the hard dependency graph:

- **B1 is a hard prerequisite for B2/B3 code.** All three batches share `internal/execution` and the CLI; starting three simultaneous writers in one checkout would conflict and violates the repository's no-parallel-writers rule.
- **Parallelism is by disjoint files, not by simultaneous writers on one file.** The parent owns integration and the shared `types.go` contract.
- **Wave 1 (completed):**
  - DeepSeek max (`sdd-implement-2`): T02–T04 — `internal/execution/{repository,workspace,checkpoint,record}.go` and tests. It may extend `types.go` additively; no other agent edits it.
  - Gemini Flash (`sdd-implement-9r-1`): T05 — `internal/execution/taskbrief.go`, its tests and `testdata/`; it reads `types.go` and must not edit it.
  - Free (`sdd-implement-free-1`): T14/T15 skill/docs work — `skills-files/`, `opencode-pack/skills/agentic-sdd-orchestrate/`, `docs/execution-runs.md`, `README.md`, `AGENTS.md` run-artifact note, `cmd/agentic-sdd/helptext.go` topics. It must not edit Go service or CLI dispatch files.
- **Wave 2 (completed):** B1 CLI T06–T07 and B2 T08–T11. DeepSeek implemented `checks.go`; the parent wired `run check` and the 11-package gate; Gemini implemented `reviewpackage.go`.
- **Wave 3 (completed):** B3 T12–T13 and T16–T17; the parent wired `review package`, consolidated the acceptance report, and closed the handover.
- **Current state:** B1–B3 implementation complete; **awaiting human review**. No acceptance, commit, or release is inferred.
- **Every agent:** no commit, push, release, real-home apply, product model/network/MCP call, or edits outside its owned files. Report changed paths, commands/exits, deviations, and compile blockers.

## B1 — Typed execution foundation, run lifecycle, and task briefs

**Batch ID:** B1
**Included tasks:** T01, T02, T03, T04, T05, T06, T07
**Outcome:** A reusable `internal/execution` service and public `run init|status|record|clean` / `task brief` commands exist with private Git-invisible workspaces, safe identity/locking/cleanup, a versioned checkpoint, fail-closed task extraction, coherent help, stable text/JSON output, and focused safety tests. No check execution or review packaging is included.
**State:** `awaiting human review — full 11-package gate green`
**Next action:** Human review of B1; no real-home/model/network/release action.

### T01 — Repository-first inventory and reconciliation

- [x] **T01** Confirm the live baseline and record any deviation before production edits.
  - **Objective:** Verify the plan's §1 findings: command names, dispatch/help structure, `--format` versus `--json`, Go version/dependencies, `internal/harnesscheck` and `internal/harnessevidence` APIs, `internal/appops` boundary, orchestration/review skill files, supported-core gate lists, and the self-ignoring `.agentic-sdd/runs/.gitignore` behavior.
  - **Files:** Read-only. May update `plan.md` §1 and this task's completion note. No production, gate, skill, or docs edits.
  - **Acceptance:** Every §1 row is confirmed or corrected with a path/line reference; `run`, `task`, and `review` are confirmed free; any material conflict is reported and B1 stops for reconciliation. No unapproved scope or real-home/model/network action.
  - **Verification:** `git status --short`; targeted reads/`go list`; record the commit and findings.
  - **Completion (2026-11-13):** baseline inventory confirmed; command names, Go 1.23/dependency-free module, helper APIs, gate lists, and the self-ignoring run root were verified. No production edit at this step.

### T02 — Execution types, repository identity, and path safety

- [x] **T02** Add `internal/execution/types.go` and `repository.go` with schema-v1 types, Git observations, source digests, and containment helpers.
  - **Objective:** Define the typed request/result model, error sentinels, limits, repo-root resolution, `git rev-parse` identity, worktree fingerprint, file/scope digests, relative-path validation, and symlink refusal. No CLI wiring yet.
  - **Files:** `internal/execution/types.go`, `internal/execution/repository.go`, `internal/execution/repository_test.go`, optional small testdata fixtures under `internal/execution/testdata/`.
  - **Acceptance:** AC-02/AC-03/AC-13 partial; repository root and HEAD are resolved exactly; a non-repo or unreadable repo fails with a clear error; path traversal and symlinks are refused; no import of `cmd/agentic-sdd`, `experiments/tui`, or `internal/appops`; no new dependency; no network/model call.
  - **Verification:** `go test ./internal/execution -run 'TestRepository|TestPathIdentity'`; `go list -deps ./internal/execution` review; `go mod verify`.
  - **Completion (2026-11-13, DeepSeek):** repository/Git identity, worktree fingerprint, source digests, path containment, and symlink refusal implemented; focused tests and dependency check pass.

### T03 — Private workspace, atomic writes, locking, and clean

- [x] **T03** Add `internal/execution/workspace.go` with run identity, root resolution, layout, permissions, per-run locking, atomic writes, and scoped cleanup.
  - **Objective:** Create `run-<UTC>-<hex>` IDs; resolve `<repo>/.agentic-sdd/runs` or an explicit `--run-root`; create the self-ignoring `runs/.gitignore` safely; write `run.json` atomically; enforce `0700`/`0600`; implement collision handling, lock acquire/reclaim/release, and `run clean` preview/apply/absent behavior.
  - **Files:** `internal/execution/workspace.go`, `internal/execution/workspace_test.go`, optional Unix/other build-tag helpers for process/lock checks.
  - **Acceptance:** AC-03; `git status --short` stays clean after workspace creation; no root `.gitignore` edit; an existing non-ignoring `runs/.gitignore` fails closed rather than leaving visible scratch; collisions, symlinks, traversal, concurrent writers, stale locks, and already-absent runs are handled; `run clean` refuses a live lock; an explicit root inside a Git worktree reports visibility honestly; no delete outside the owned run directory; no hidden prompt.
  - **Verification:** `go test ./internal/execution -run 'TestWorkspace|TestRunID|TestLock|TestClean'`; a temporary-repo test that creates a run and asserts empty `git status --porcelain`.
  - **Completion (2026-11-13, DeepSeek):** run IDs, private layout, self-ignoring root, atomic writes, locks, clean, and visibility reporting implemented; focused tests pass.

### T04 — Checkpoint state machine, status, and untrusted records

- [x] **T04** Add `internal/execution/checkpoint.go` and `record.go` with observed state, freshness/reconciliation, status results, and strict record ingestion.
  - **Objective:** Implement the state precedence and transition table from plan §8; compute freshness reasons from source/Git/worktree digests; expose a status result with `next_permitted_investigation`; parse strict JSON records with size/duplicate/unknown-field/path validation; persist raw records and digests; ensure a `passed` claim yields `awaiting-verification`, never `verified-evidence`.
  - **Files:** `internal/execution/checkpoint.go`, `internal/execution/record.go`, `internal/execution/checkpoint_test.go`, `internal/execution/record_test.go`, `internal/execution/testdata/records/`.
  - **Acceptance:** AC-07/AC-08 partial/AC-13; every state and freshness reason is covered; no `completed` or `approved` state; no auto-resume; source/tasks/batch scope changes and worktree changes become `stale`, while checkbox/status-only progress is recorded as `progress_updated` and does not stale the run; a record cannot verify a task; malformed/oversized/duplicate/unknown input exits as a validation error with no partial artifact.
  - **Verification:** `go test ./internal/execution -run 'TestCheckpoint|TestStatus|TestRecord'`.
  - **Completion (2026-11-13, DeepSeek/parent):** state precedence, freshness/scope digests, status, strict records, and record-as-claim persistence implemented; parent fixed record visibility in status and added assertions. Focused tests pass.

### T05 — Deterministic task brief extraction

- [x] **T05** Add `internal/execution/taskbrief.go` with batch/task section scanning, exact ID matching, reference resolution, and brief rendering.
  - **Objective:** Implement the plan §5 algorithm: locate the selected batch, find exactly one task in `### T...` or bold-checkbox form, honor fenced code blocks and nested headings, extract the exact section and line range, resolve FR/AC references in `spec.md`, fail closed on missing/duplicate/ambiguous IDs, and write an idempotent brief with `--force` semantics.
  - **Files:** `internal/execution/taskbrief.go`, `internal/execution/taskbrief_test.go`, `internal/execution/testdata/{tasks,spec}/*.md`.
  - **Acceptance:** AC-04; `T1` never matches `T10`; duplicate IDs, missing tasks, wrong-batch IDs, malformed fences, and unresolved required references fail with exit `2` and no brief; CRLF, nested `####` headings, and checkbox forms work; a non-task `###` boundary stops the section; the full-file and normalized scope digests are produced, with checkbox/status-only changes ignored by the scope digest; the brief excludes subsequent tasks and unrelated spec text; no LLM.
  - **Verification:** `go test ./internal/execution -run TestTaskBrief`.
  - **Completion (2026-11-13, Gemini/parent):** 18 scenarios pass; the parent hardened the run-directory symlink check, reused the shared atomic writer, and removed the duplicate bounded-read implementation. Full-package rerun passes; review is pending.

### T06 — CLI adapters, help, JSON, and exit codes for run/task

- [x] **T06** Wire `run init|status|record|clean` and `task brief` through thin CLI adapters.
  - **Objective:** Add `cmd/agentic-sdd/run.go`, `task.go`, and `execution_cli.go`; dispatch `run`/`task` from `main.go`; add help topics and usage lines; implement `--repo`, `--run-root`, `--format text|json`, `--yes`, `--force`, and exit-code mapping; render stable text/JSON from typed results.
  - **Files:** `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/helptext.go`, `cmd/agentic-sdd/run.go`, `cmd/agentic-sdd/task.go`, `cmd/agentic-sdd/execution_cli.go`, corresponding tests.
  - **Acceptance:** AC-01/AC-02/AC-12/AC-13 partial; old commands/output/exit codes unchanged; `--help` on each family prints to stdout and exits `0`; unknown flags and missing required arguments exit `2`; JSON has `schemaVersion`/`kind`; no ANSI/decoration in JSON or non-TTY; no hidden prompt; `run clean` without `--yes` is a preview; a direct run is visibly marked and `task brief` refuses it with a clear message; direct `run record` defaults the task label to `direct`.
  - **Verification:** `go test ./cmd/agentic-sdd -run 'TestExecutionCLI|TestRunCLI|TestTaskBriefCLI|TestUsage'`; existing CLI tests.
  - **Completion (2026-11-13, parent):** `run init|status|record|clean` and `task brief` are wired with text/JSON, help/usage, and exit-code mapping; direct scope, JSON schema, preview/clean, and record-as-claim tests pass. `run check` is wired via T10 and `review package` via T13. `go test ./internal/execution ./cmd/agentic-sdd` (focused) and `go vet` are green.

### T07 — B1 gate registration and focused verification

- [x] **T07** Register `internal/execution` in the supported-core gate and complete B1 evidence.
  - **Objective:** Add `./internal/execution` to `Taskfile.yml` (`test`, `vet`), `harness/checks.json` (`test`, `vet` argv), `tests/supported-core-go.sh` (required dir plus package list), and `AGENTS.md` (Go layout, supported-core list/count). Run the focused B1 suite and record exact commands/exits.
  - **Files:** `Taskfile.yml`, `harness/checks.json`, `tests/supported-core-go.sh`, `AGENTS.md`, B1 test files, and a short B1 evidence note in this file.
  - **Acceptance:** AC-03/AC-04/AC-12 partial; 11-package gate is consistent across all four lists; `go mod verify` unchanged; no new dependency; `task test`, `task vet`, and `git diff --check` pass; no real home, model, network, or live apply; any aggregate-gate conflict is reported as `not_run`/`blocked`, not hidden.
  - **Verification:** `task test`; `task vet`; `git diff --check`; `go mod verify`; record commit, commands, environment, and exits.
  - **Completion (2026-11-13):** all four gate lists, the four Python parity tests, `CONTRIBUTING.md`, and the harness README are updated to eleven packages; parity tests pass. `task test`, `task vet`, `git diff --check`, and `go mod verify` pass in the integrated tree.

## B2 — Truthful check evidence and recovery

**Batch ID:** B2
**Included tasks:** T08, T09, T10, T11
**Outcome:** `run check` executes an explicit argv without a shell, captures complete-so-far stdout/stderr with truthful pass/fail/timeout/interruption/limit state, records exact provenance, integrates with the checkpoint, and survives interruption without claiming success. No review packaging is included.
**State:** `awaiting human review — full gate and race green`
**Next action:** Human review of B2 check evidence; no acceptance inferred.

### T08 — Check execution, capped logs, and truthful status

- [x] **T08** Add `internal/execution/checks.go` and platform helpers for process execution and evidence capture.
  - **Objective:** Implement plan §6: require an argv after `--`; no shell; repo-root working directory; timeout; process-group cancellation on Unix; capped stdout/stderr; start/end/exit/signal/truncation metadata; private `argv.json`; summary extraction with an explicit excerpt label; exact result states.
  - **Files:** `internal/execution/checks.go`, optional `checks_unix.go`/`checks_other.go`, `internal/execution/checks_test.go`, helper-process fixtures.
  - **Acceptance:** AC-05; pass/fail/timeout/signal/output cap/missing program/disk-error paths are covered; a failure far from the tail is surfaced; full logs are retrievable by exact path; no secret-like argv is echoed in the summary/JSON by default; no environment dump; no model/network call.
  - **Verification:** `go test ./internal/execution -run 'TestCheck|TestCommand|TestSummary'`.
  - **Completion (2026-11-13, DeepSeek):** `checks.go` + Unix/other process helpers + 19 scenarios implemented; focused tests, `-race`, `-count=3`, vet, and cross-OS builds pass. `CheckRunning` is a check-local in-progress result, replaced by the final result; an interrupted writer can leave it visible. JSON envelope is wrapped by the CLI T10.

### T09 — Checkpoint integration and interrupted/stale recovery

- [x] **T09** Integrate checks with the checkpoint state machine and per-run lock.
  - **Objective:** Record `running` at check start, update on completion/interruption, detect a dead/stale lock and report `interrupted_check`, compare the worktree observation at start/end, mark `worktree_changed_during_check` when needed, and prevent two concurrent checks on one run.
  - **Files:** `internal/execution/checkpoint.go`, `internal/execution/checks.go`, `internal/execution/checkpoint_test.go`, `internal/execution/checks_test.go`.
  - **Acceptance:** AC-05/AC-07/AC-08; a passing check can reach `verified-evidence` only when the end observation matches the current source/worktree and no unresolved claim/blocker contradicts it; a record alone cannot; a killed process or stale lock yields `observed-partial`, not success; `run status` never auto-resumes.
  - **Verification:** `go test ./internal/execution -run 'TestCheckpointCheck|TestInterrupted|TestStale|TestLock'`.
  - **Completion (2026-11-13, DeepSeek):** start/end observations, lock hold, interrupted writer, worktree-change downgrade, and checkpoint state mapping are covered; focused tests pass.

### T10 — CLI `run check`

- [x] **T10** Wire `run check` through the CLI with stable output and exit codes.
  - **Objective:** Add the `run check` adapter, `--timeout`, `--max-output-bytes`, `--task`, common flags, help, text/JSON rendering, and exit mapping (`0` pass, `1` fail/interrupted, `2` usage, `124` timeout). The actual command exit/signal stays in the result.
  - **Files:** `cmd/agentic-sdd/run.go`, `cmd/agentic-sdd/execution_cli.go`, `cmd/agentic-sdd/helptext.go`, corresponding tests.
  - **Acceptance:** AC-01/AC-05/AC-12; `--` is required and the command vector is preserved exactly; no implicit shell; JSON is valid and versioned, including a failed/timed-out/interrupted check result; text gives exact log paths; no raw argv echo by default; no prompt; old commands unchanged; a direct run defaults the task label to `direct` when `--task` is omitted.
  - **Verification:** `go test ./cmd/agentic-sdd -run 'TestRunCheck|TestExecutionCLI'`.
  - **Completion (2026-11-13, parent):** `run check` wired with the `schemaVersion`/`kind`/`check` JSON envelope, `--timeout`, `--max-output-bytes`, `--task`, `--` boundary, pass/fail/timeout exit mapping, and missing-run exit 1. Focused CLI tests pass; full gate pending T11/T16.

### T11 — B2 verification and evidence

- [x] **T11** Complete the B2 safety matrix and record evidence.
  - **Objective:** Run the group-4 tests plus concurrency, timeout, interruption, output-cap, and failure-summary cases; record exact commands/exits, fixture setup, and any untested claim. No production changes unless a focused in-scope correction is authorized.
  - **Files:** B2 test files and a short B2 evidence note in this file; no product scope expansion.
  - **Acceptance:** AC-05/AC-07/AC-08; no real home, model, network, or MCP; no skip/filter/tolerance added; `task test`, `task vet`, and `git diff --check` pass; unresolved issues remain visible.
  - **Verification:** `go test ./internal/execution ./cmd/agentic-sdd`; `task test`; `task vet`; `git diff --check`; record commit/commands/env/exits.
  - **Completion (2026-11-13):** check/checkpoint/CLI tests pass; `task test`, `task vet`, `git diff --check`, `go mod verify`, and the B2 race run are green. Evidence is consolidated in [`b3-acceptance.md`](b3-acceptance.md).

## B3 — Review packages, severity guidance, and closeout

**Batch ID:** B3
**Included tasks:** T12, T13, T14, T15, T16, T17
**Outcome:** `review package` produces a truthful committed/staged/unstaged/untracked package with explicit binary/large/sensitive/conflict handling; existing review guidance distinguishes severity from validity and keeps fresh review request-only; docs/help describe the contract; the six test groups and AC-01–AC-13 report are complete; the session handover records `awaiting human review`.
**State:** `awaiting human review — implementation complete`
**Next action:** Human review of the B3 review package, skill/docs, and acceptance report; no acceptance claim.

### T12 — Git change collection and manifest types

- [x] **T12** Add `internal/execution/reviewpackage.go` Git collection primitives.
  - **Objective:** Implement read-only Git inspection with `-z` output, `--no-ext-diff --no-textconv`, `GIT_OPTIONAL_LOCKS=0`, disabled fsmonitor, base/head resolution, committed/staged/unstaged/untracked enumeration, rename/delete/mode/binary/submodule/conflict detection, and path containment. No package assembly yet.
  - **Files:** `internal/execution/reviewpackage.go`, `internal/execution/reviewpackage_test.go`, disposable-repo helpers.
  - **Acceptance:** AC-06 partial; no index/worktree mutation; external diff drivers cannot execute; a missing/invalid base exits `2`; `--head` mismatch and conflicted index are detected and reported; paths are parsed safely.
  - **Verification:** `go test ./internal/execution -run 'TestReviewCollect|TestReviewGit'`.
  - **Completion (2026-11-13, Gemini):** review collection and manifest primitives implemented; 18 review scenarios pass, including committed/staged/unstaged/untracked, rename, delete, mode, binary, large, sensitive, symlink, submodule, conflict, head mismatch, invalid base, and no mutation.

### T13 — Review package assembly, partiality, and CLI

- [x] **T13** Assemble the review package, manifest, omissions, and `review package` CLI.
  - **Objective:** Implement per-file patch generation, untracked copies, binary/large/sensitive handling, total caps, manifest fields, package layout, `complete`/`partial_reasons`, and exit `0`/`3`/`1`/`2`. Wire `cmd/agentic-sdd/review.go`, help, and text/JSON.
  - **Files:** `internal/execution/reviewpackage.go`, `cmd/agentic-sdd/review.go`, `cmd/agentic-sdd/execution_cli.go`, `cmd/agentic-sdd/helptext.go`, corresponding tests.
  - **Acceptance:** AC-01/AC-06/AC-12; committed, staged, unstaged, and untracked changes are represented as applicable; binary/rename/delete/mode/submodule/conflict/sensitive/large cases have explicit statuses; no false complete-diff claim; no redaction claim; no symlink follow; partial package exits `3` and still emits its JSON result object when requested.
  - **Verification:** `go test ./internal/execution -run 'TestReviewPackage|TestReviewPartial'; go test ./cmd/agentic-sdd -run 'TestReviewCLI'`.
  - **Completion (2026-11-13, parent):** `review package` is wired with text/JSON, partial exit 3, exact omission reasons, and manifest schema; CLI tests cover a complete package and a sensitive partial package.

### T14 — Severity, request-only reviewer, and orchestration guidance

- [x] **T14** Integrate severity and request-only review into the existing skill sources.
  - **Objective:** Add the Critical/Important/Minor severity dimension separate from `valid`/`disputed`/`needs evidence`; update verification-review verdict rules; add request-only fresh per-task reviewer language; add the five failure categories and bounded-correction/no-auto-escalation language; add CLI usage and artifact-reference guidance to the orchestration skill and its dispatch/failure references.
  - **Files:** `skills-files/agentic-sdd-router/references/review-resolution.md`, `skills-files/agentic-sdd-router/references/independent-review.md`, `skills-files/agentic-sdd-verification-review/SKILL.md`, `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`, `opencode-pack/skills/agentic-sdd-orchestrate/references/dispatch-contract.md`, `opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`, `internal/skillsync` asset tests as needed.
  - **Acceptance:** AC-08/AC-09/AC-10/AC-11; severity is distinct from validity; no mandatory second review; fresh reviewer is request-only and refused when isolation/permissions are unverified; no new skill name/profile/routing/MCP permission; existing asset tests pass.
  - **Verification:** `go test ./internal/skillsync`; targeted text assertions for the severity/request-only phrases.
  - **Completion (2026-11-13, Free agent):** severity table and Ready rule added; request-only reviewer/IR-06-01 language added; orchestration commands, five failure categories, bounded correction, and checkpoint rules added. `go test ./internal/skillsync` exit 0; `go test ./cmd/agentic-sdd` exit 0; `git diff --check` exit 0. Review pending.

### T15 — Public docs, help, and run-artifact guidance

- [x] **T15** Document the CLI and artifact contract without overclaiming.
  - **Objective:** Add `docs/execution-runs.md` with examples, artifact layout, exit codes, stale/reconciliation semantics, review partial reasons, privacy limits, and the no-sandbox/no-approval boundary. Add a short README pointer and the `run`/`task`/`review` help topics; add the run-artifact note to `AGENTS.md`.
  - **Files:** `docs/execution-runs.md`, `README.md`, `cmd/agentic-sdd/helptext.go`, `AGENTS.md`, help/docs tests.
  - **Acceptance:** AC-01/AC-12; commands and flags match the frozen contract; examples are runnable in a disposable repo; exit codes and partial semantics are accurate; no token-savings, sandbox, approval, or release claim; existing docs untouched.
  - **Verification:** `go test ./cmd/agentic-sdd -run TestHelp`; `git diff --check`; manual read of examples.
  - **Completion (2026-11-13, Free agent):** `docs/execution-runs.md`, README pointer, AGENTS.md run-artifact note, and `helptext.go` topics added; `docs/README.md` index pointer added by the parent. `gofmt` and `go test ./cmd/agentic-sdd -run TestHelp` exit 0. Help-topic dispatch/usage wiring remains T06; review pending.

### T16 — Six-group verification and AC report

- [x] **T16** Run the complete deterministic verification matrix and write the acceptance report.
  - **Objective:** Execute the six groups from plan §11 with isolated fixtures; record commands, exits, environment, and evidence for AC-01–AC-13; verify old-command regression; run the 11-package gate; record `pass`/`fail`/`blocked`/`not_run` honestly.
  - **Files:** `specs/016-agent-execution-context-evidence-recovery/b3-acceptance.md`; test files as needed; no production scope expansion.
  - **Acceptance:** AC-01–AC-13 are each accounted for; no real home/model/network/MCP; no skip/filter/tolerance; `task test`, `task vet`, `git diff --check`, `go mod verify` pass; untested claims are explicit.
  - **Verification:** `go test ./internal/execution ./cmd/agentic-sdd`; `go test ./internal/skillsync`; `task test`; `task vet`; `git diff --check`; `go mod verify`.
  - **Completion (2026-11-13):** [`b3-acceptance.md`](b3-acceptance.md) records AC-01–AC-13, exact commands/exits, not-run items, and residual limits. The full gate, race run, and Python parity checks pass.

### T17 — Handover and batch closeout

- [x] **T17** Write the session handover and set the batch state truthfully.
  - **Objective:** Create `docs/handovers/<YYYY-MM-DD>_016.md` (append a letter for a second same-day session) with batch position/next authorized batch, what changed, decisions, open decisions, verified facts, blockers, next permitted action, guardrails, and first commands. Update this ledger to `awaiting human review`; do not call the feature accepted.
  - **Files:** `docs/handovers/<YYYY-MM-DD>_016.md`, this `tasks.md`.
  - **Acceptance:** No acceptance/approval claim; links to spec/plan/tasks/evidence by path; no credentials or sensitive traces; next action is explicit; the batch remains `awaiting human review`.
  - **Verification:** `git diff --check`; read the handover against the repository's handover structure.
  - **Completion (2026-11-13):** [`handover`](../../docs/handovers/2026-11-13_016.md) updated to B1–B3 implementation complete, human review pending; no acceptance, commit, or release claim.

## Continuation state

- **Current batch:** B1–B3 implementation complete; **awaiting human review**.
- **Approval:** approved 2026-11-13 in [spec §14](spec.md#14-approval).
- **Next permitted action:** human review of the integrated result and [`b3-acceptance.md`](b3-acceptance.md). Do not start another feature batch.
- **Do not:** run real-home `apply`, execute product model/network/MCP calls, dispatch fresh reviewers, commit/push/tag/release, or treat a green suite as acceptance.
