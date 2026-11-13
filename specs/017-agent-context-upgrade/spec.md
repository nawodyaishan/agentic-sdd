# Feature 017 — Execution context integrity hardening

**Revision:** ACCEPTED r1, 2026-11-13 — combined approval in §Approval; B1 implementation and human review are complete. Acceptance evidence is recorded in `tasks.md`.
**Input:** [owner-supplied historical draft](016-agent-execution-context-evidence-recovery-spec.md). Preserve that file unchanged; its Feature 016 name, pending approvals and proposed CLI are historical, not this feature's authority.
**Baseline:** `05c67dff31d9f4a023675bc56b269623961da1d8`.

## Outcome and compatibility

Close the reproduced recovery and concurrency gaps in the existing execution service, rather than rebuild Feature 016. A generated task brief must be attributable to its run, registered for recovery, and protected by the existing writer lock. A foreign checkpoint must never appear to be this run's normal status.

Feature 016 already implements the seven command operations in the supplied draft. Its [approved spec](../016-agent-execution-context-evidence-recovery/spec.md#14-approval) and [batch ledger](../016-agent-execution-context-evidence-recovery/tasks.md) remain unchanged: implementation awaits human review, not acceptance. This feature neither accepts 016 nor reopens its approval.

See [compatibility evidence](compatibility.md) for coverage, reproduced findings and audit limitations. Product basis: [vision](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md) §§2, 6 (truthful evidence, explicit human authority, proportionate process). This repository has no top-level SRS.

## Scope

- Existing `internal/execution` task-brief creation, checkpoint loading and freshness evaluation.
- Focused typed-service and CLI regression tests; recovery documentation.
- Schema-v1 compatibility, existing commands, exit conventions, private artifact permissions and offline operation.

Excluded: rebuilding checks/review packaging, TUI, fleet/MCP/model routing, agent dispatch, installer changes, new root dependencies, gate-list expansion, experimental 005/006/007 validation, real-home apply, publication and releases. No general sandbox, cryptographic authenticity or comprehensive security-audit claim.

## Requirements and acceptance criteria

### FR-01 — Bind persisted observations to the owning run

Loading an existing checkpoint for status or mutation must compare its run ID with the containing run's validated identity. A syntactically valid foreign ID is still a conflict. Refuse before presenting normal status or writing artifacts; preserve both files for investigation. Missing checkpoints retain the existing explicitly synthesized-prepared behavior, but any discovered unregistered brief remains subject to FR-04.

**AC-01:** A fixture checkpoint with another valid run ID is rejected by status and every checkpoint-consuming mutation, with the established conflict exit mapping, no normal-status JSON and no changed run content. Matching schema-v1 runs still work.

### FR-02 — Serialize and bind run-backed brief creation

A persisted brief requires a valid owning run. Explicit request identity, repository, source and batch fields must agree with that run when supplied. Direct-scope runs remain unsupported for task briefs. All run-backed brief mutation, including replacement and checkpoint registration, participates in the existing per-run writer lock. Pure extraction/rendering remains available without a run.

**AC-02:** A live writer lock prevents brief creation/replacement/registration without changing artifacts. Matching requests succeed; conflicting source/run/batch overrides, invalid/missing identity and symlink/path escapes fail closed through typed APIs as well as CLI.

### FR-03 — Register brief provenance truthfully

Successful persisted brief creation records its task, contained artifact path and digest in the existing checkpoint task metadata. Registration preserves check/claim/blocker history and task state; merely extracting a brief cannot verify evidence or clear a failure. Idempotent creation and `--force` replacement obey the same rules.

**AC-03:** Successful create/repeat/replace leaves a matching brief and checkpoint reference. After deletion, replacement with different bytes, or an interrupted two-file update, status reports stale/reconciliation evidence instead of silently trusting the brief. Checkpoint-write failure never returns success. Neither automatic rollback over unrelated work nor automatic repair is allowed.

### FR-04 — Conservatively reuse schema-v1 runs

Keep old runs readable without automatic migration. A discovered brief without registered provenance is an observation requiring reconciliation, not trusted evidence. Diagnose without changing files. Explicitly reissuing `task brief` may register a byte-identical artifact only after current source/run validation under lock; differing bytes still require `--force`. Status never performs that repair.

**AC-04:** An otherwise clean old run containing an unregistered brief reports `stale` with a specific reconciliation reason and an investigation-only next action. Registered missing/changed briefs are also detected. Directory inspection is bounded, contained and symlink-safe; inability to inspect cannot yield a clean result. An old run with no brief artifacts is not invalid merely because it predates 017.

### FR-05 — Preserve existing authority and interfaces

Reuse typed services, schema-v1 fields and existing CLI grammar. Additive diagnostic reasons are permitted; no new verified/approved/completed meaning, prompt, network/model call or canonical ledger is introduced.

**AC-05:** Focused tests demonstrate current help/text/JSON/exit behavior, direct scope, source staleness and record-as-claim behavior remain intact. Supported-core `task test`, `task vet` and `git diff --check` are reported honestly; no exclusions are added to obtain a pass.

## Owner decisions

- 2026-11-13: audit implemented 016 for evidence-backed residual gaps; use 017 only for those fixes.
- Conservative reuse of old schema-v1 runs: unregistered briefs require reconciliation; no silent migration.
- One implementation batch with three disjoint parallel workstreams and serial integration/review, rather than artificial concurrent batches.
- Requested implementation lanes: Gemini Flash High, DeepSeek Flash max and OpenCode free. Drafting these lanes did not launch them; B1 was separately authorized and executed on 2026-11-13.

## Approval

**Combined decision:** **Approved, 2026-11-13.** The owner approved the reviewed `spec.md` r1, `plan.md` r1 and `tasks.md` r1 and authorized implementation of B1 in this session.

- **Approver:** The repository owner in this conversation (no additional identity asserted).
- **Approved revisions/scope:** `spec.md` r1, `plan.md` r1 and `tasks.md` r1 as presented. This covers AC-01–AC-05 and B1 T01–T05 only.
- **Decision evidence:** the owner said “i accept the docs , start implementation” and subsequently “using sub agehts” on 2026-11-13.
- **Authorized implementation:** B1 only, using the named implement subagents under the dependency-safe, disjoint-file schedule in `plan.md` §5 and `tasks.md`. One integrated batch; human review remains required afterward.
- **Still not authorized:** real-home writes or `apply`, product model/network/MCP calls, fresh reviewer dispatch, Feature 015 MCP amendment approval, tag, release, push or deployment.
- **Reference preserved:** Feature 016's own approval and pending B1–B3 human review are unchanged.

Approval is recorded once, here, tied to the reviewed revisions. `plan.md` and `tasks.md` reference this section; they do not maintain competing approval flags.
