# Feature 017 — Tasks

**Revision:** r1, 2026-11-13.
**Parents:** [spec.md r1](spec.md), [plan.md r1](plan.md).
**Approval reference:** [spec §Approval](spec.md#approval) — combined approval recorded 2026-11-13; B1 authorized. This file is the batch ledger, not an approval record.
**Assignments/tools:** plan §§5–6; no task-specific specialist exceptions.

## Batch ledger

| Batch | Tasks | Outcome | State | Next action |
|---|---|---|---|---|
| B1 | T01–T05 | Run-bound, serialized briefs and truthful recovery with compatible diagnostics | `accepted — 2026-11-13` | Feature 017 is accepted for the B1 scope. Follow-on entrypoint/help/docs work is a separate feature; no next batch exists here. |

## B1 — Execution context integrity

**Completion:** AC-01–05 met with isolated fixture evidence and required gates; changed paths and unresolved risks reported. Never mark accepted from tests alone.
**Verification:** T02/T03/T04 focused checks; T05 integration, race and repository gates.
**Stop:** Persist `awaiting human review` after verified implementation; no automatic continuation.

### T01 — Revalidate baseline and freeze the shared seam

- [x] **T01** Parent confirms F-01–03 against current checkout and freezes plan §2 helper contracts.
  - Dependencies: combined approval and explicit B1 authorization.
  - Allowed files/operations: read source; update this feature's continuation notes; parent-only minimal shared-contract edits if actually necessary. No production fix before a reproducing regression test.
  - Acceptance: current HEAD/dirty state recorded, unrelated work preserved, exclusive lane ownership confirmed, existing task fields reused where possible; no unapproved schema or scope expansion.
  - Verification: `git status --short`, `git rev-parse HEAD`, CodeGraph source inspection; baseline focused test command with actual exit recorded.
  - Completion (2026-11-13): baseline `05c67dff31d9f4a023675bc56b269623961da1d8`; only the feature records were untracked. `go test ./internal/execution ./cmd/agentic-sdd` passed. Frozen contract: `registerTaskBrief(checkpoint, runDir, brief)` verifies designated path/digest and preserves task state; `inspectBriefArtifacts(runDir, checkpoint)` returns sorted reasons or an error. Parent added `internal/execution/briefintegrity.go` with both helpers and the three additive reason constants; Lane A may correct the helper and owns checkpoint integration/tests. Existing `TaskState` fields reused; no schema change.
  - Stop condition: changed requirements, unavailable requested route, overlapping ownership or unsafe fixture environment.

### T02 — Enforce checkpoint ownership and brief freshness

- [x] **T02** Lane A implements common checkpoint ownership validation and plan §2 brief-integrity helpers.
  - Dependencies: T01; concurrent with T03/T04.
  - Allowed files: exactly lane A paths in plan §5. Other production/tests are read-only.
  - Acceptance: AC-01/04; registered missing/mismatched and legacy unregistered briefs are diagnosed; registration helper preserves existing task state/history. Matching old runs and absent checkpoints retain specified behavior.
  - Verification: regression-first tests for foreign checkpoint ID, no-write refusal, matching/absent checkpoint, matching/missing/changed brief, unregistered legacy artifact, no-brief legacy run, symlink/escape/unreadable/entry-budget refusal. Run new tests plus existing checkpoint tests.
  - Completion (2026-11-13, DeepSeek Flash max): `checkpoint.go` now validates the owning run ID in `LoadCheckpoint`, inspects brief provenance in `evaluateFreshness`, and maps the additive reasons to `reconcile_source_change`. New `checkpoint_identity_test.go` and `briefintegrity_test.go` cover foreign/matching/missing checkpoints, no-write refusal, registered/missing/changed/unregistered/symlink/unreadable/over-budget briefs, and status mapping. Parent added record/review-package conflict coverage during integration. Red evidence against the pre-017 checkpoint (`-overlay`) failed as expected; green after the change. Full package, race and vet passed.
  - Stop condition: need for new schema, unbounded traversal or changes outside ownership; return blocker to parent.

### T03 — Make brief creation a locked, recoverable transaction

- [x] **T03** Lane B implements plan §4 in the typed service, retaining pure extraction helpers.
  - Dependencies: T01; compiled integration depends on T02's declared helper, but work can proceed concurrently.
  - Allowed files: exactly lane B paths in plan §5, including the existing `taskbrief_test.go` fixture update required by the stricter owning-run contract.
  - Acceptance: AC-02/03; lock blocks mutation, source/run overrides fail closed, brief registration succeeds truthfully, idempotence and force work without erasing failures/claims/blockers; failed checkpoint writes do not report success.
  - Verification: red then green tests for live lock, identity/source/batch mismatch, missing run identity, create/repeat/force registration, interrupted artifact-first write, preservation of prior evidence, progress-only versus scope edits, lock release on all error paths. Existing parser tests stay green.
  - Completion (2026-11-13, Gemini Flash High): run-backed `TaskBrief` now validates the owning identity and explicit fields, acquires the per-run writer lock, rejects spec or substantive batch drift, writes artifact-first, registers path/digest, and reports checkpoint-write failure without claiming success. `taskbrief_transaction_test.go` covers lock exclusion, contradictions, registration, history preservation, legacy repair, force, source drift and lock release; the existing idempotence fixture now uses a real disposable run. Parent added explicit release-error propagation during integration. Focused `TestTaskBrief` and full package passed.
  - Stop condition: cannot express transaction via frozen helper contract; do not edit checkpoint/types/CLI files owned elsewhere.

### T04 — Document recovery and add disjoint CLI regressions

- [x] **T04** Lane C documents conservative legacy reuse and exercises public CLI behavior.
  - Dependencies: T01; concurrent with T02/T03; final green requires their integration.
  - Allowed files: exactly lane C paths in plan §5. No help/parser/production or feature-ledger edits.
  - Acceptance: AC-05 plus CLI coverage of AC-01–04; documents distinguish observation, reconciliation, explicit retry and force from approval/resume. No instruction to trust/migrate old briefs automatically.
  - Verification: fixture CLI tests for wrong checkpoint ID (established conflict mapping, no normal JSON), deleted brief status, legacy unregistered diagnostic, repeat/force and existing direct-scope refusal. No test writes into a real home. Run named new CLI tests.
  - Completion (2026-11-13, OpenCode free DeepSeek): `docs/execution-runs.md` documents run-bound provenance, lock participation, stale reasons, conservative schema-v1 reuse and explicit force/registration limits without claiming a sandbox or approval. `cmd/agentic-sdd/execution_integrity_test.go` adds four CLI regressions covering foreign checkpoint refusal, deleted registered brief, legacy unregistered brief and repeat/force. Three were expected red before Lanes A/B and passed after integration; the repeat/force guard passed throughout.
  - Stop condition: missing/inaccessible test helper, model limit or production changes required; return blocker without crossing ownership.

### T05 — Integrate, verify and stop for review

- [x] **T05** Parent integrates all lanes, resolves contract-only issues, performs one completion review and records evidence.
  - Dependencies: T02, T03, T04 complete; serial, no active worker writing.
  - Allowed files: integrated diff and necessary bounded glue; parent-only shared paths. Scope expansion returns for decision.
  - Acceptance: AC-01–05 demonstrated, no new dependency/package/gate exclusion, no false verification, unchanged Feature 016 approval/state. Record actual models/fallback reasons, changes, omissions and commands/exits.
  - Verification: sanitized offline `go test ./internal/execution ./cmd/agentic-sdd`; `go test -race ./internal/execution`; `task test`; `task vet`; `git diff --check`; source/diff review. Run `codegraph sync` and verify status before final review. Aggregate conflicts are blocked/not_run, never hidden.
  - Completion (2026-11-13, parent): integrated the three lanes without changing schema, gate lists or dependencies. Added explicit lock-release error propagation in `TaskBrief`, plus record/review-package foreign-checkpoint coverage in `checkpoint_identity_test.go`. `go test ./internal/execution ./cmd/agentic-sdd` passed; `go test -race ./internal/execution` passed; `task test`, `task vet`, `go mod verify` and `git diff --check` passed; `codegraph sync` reports up to date. Completion review verdict: **Ready** (technical assessment, not human acceptance). Residual risks recorded below. B1 set to `awaiting human review`; no commit/push/tag/release/apply.
  - Stop condition/completion: set B1 to `awaiting human review`, write session handover and present exact evidence. No commit/push/tag/release/apply or additional batch.

## Continuation

- Current stage: B1 implementation complete and **accepted by the owner on 2026-11-13**. Combined approval is recorded in [spec §Approval](spec.md#approval); B1 was explicitly authorized on 2026-11-13.
- Acceptance evidence: the owner said “accepted” in the session following the B1 completion report. This accepts the recorded B1 scope and evidence only; it does not authorize commit, release, deployment or a follow-on feature.
- Current batch/tasks: B1 T01–T05 complete and accepted. No next batch exists or is authorized in Feature 017.
- Design decisions: conservative legacy reuse and one parallel batch selected by the owner; see spec §Owner decisions. Implemented lanes used DeepSeek Flash max (A), Gemini Flash High (B) and OpenCode free DeepSeek (C); no fallback was needed.
- Baseline: `05c67dff31d9f4a023675bc56b269623961da1d8`. The working tree contains Feature 017 plus unrelated untracked user work that appeared during the session (`docs/snap-ubuntu-packaging-study.md` earlier, `specs/018/` at final check); Feature 017 did not create or modify it.
- Changed implementation paths: `internal/execution/{checkpoint,taskbrief,briefintegrity}.go`, focused tests `internal/execution/{checkpoint_identity,briefintegrity,taskbrief_transaction,taskbrief}_test.go`, `cmd/agentic-sdd/execution_integrity_test.go`, `docs/execution-runs.md`, and this feature's records. `go.mod`/`go.sum`, gate lists and Feature 016 records are unchanged.
- Checks (integrated code state): `go test ./internal/execution ./cmd/agentic-sdd` pass; `go test -race ./internal/execution` pass; `task test` pass; `task vet` pass; `go mod verify` pass; `git diff --check` pass; CodeGraph up to date.
- Residual risks: the artifact-first/checkpoint-second transaction can be interrupted between writes; status detects the unregistered/mismatched artifact and never auto-repairs. Path/lock protections are local-user containment, not a sandbox; Feature 006 `IR-06-01` remains unresolved. Legacy unregistered briefs require explicit reconciliation. Platform validation was Linux-only this session. `LoadCheckpoint` now requires a readable owning `run.json`; production callers already load identity first.
- Next permitted action: none in Feature 017. Follow-on work requires a separate feature with its own combined approval; do not reopen this accepted scope merely to bundle entrypoint, help or TUI changes.
