---
name: agentic-sdd-verification-review
description: "Review a batch, direct fix or whole feature against requirements and engineering evidence; resolve findings within authority, and keep review-only requests non-mutating."
---

# Agentic SDD Verification Review

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use after implementation or for an explicit review. Read the [review-scope and finding-resolution guidance](../agentic-sdd-router/references/review-resolution.md). The same rules apply whether reached through the router or invoked directly. Implementation invokes one completion review for its current result; the router consumes that result, not another unchanged review.

| | Feature batch | Direct fix |
|---|---|---|
| Inputs | The diff, approved `spec.md`/`plan.md` sections, the batch's `tasks.md` entries, the combined approval reference, applicable contracts and test evidence | The request, relevant code and constraints, the diff and verification evidence |
| Compare against | The feature's acceptance criteria, design boundaries and batch outcome | The stated request and the constraints the change touches |
| State writer | Return findings/verdict; implementation owns the permitted batch-state write | None — evidence goes in the response |

For **feature scope**, use the approved aggregate outcome, cross-batch diff, acceptance/integration evidence and consequential open findings. Select it for an explicit feature-review request or planned final-batch validation, not every task. Passing individual batches while their connected behavior fails keeps the feature unready. An old-batch issue may block the feature without authorizing an old-batch edit.

For a direct fix, do not require, look for or create feature documents; a missing `specs/` directory, plan assignment, task entry or batch ID is not a gap to report.

Review the actual relevant staged, unstaged and untracked changes and deletions, not HEAD alone. Treat an old summary as a pointer; refresh changed/uncertain inputs before relying on its verdict.

Consult upstream source sections only where relevant. Cover **requirements** (missing, extra, misunderstood or unverified behavior) and **engineering** (correctness, compatibility, permissions/security, errors, meaningful tests and relevant operational/maintenance risk). Use the [testing guidance](../agentic-sdd-router/references/testing.md) to catch tautological expectations or setup-only regression failures. Diagnose a named uncertainty with `agentic-sdd-debugging` through permitted observation; review-only diagnosis recommends instrumentation/checks rather than adding logging or running write-producing tests.

Reuse a prior check only if code, relevant dependencies/configuration, inputs and environment still match. Rerun affected, failed or repository-required checks. Choose focused tests and static checks for changed behavior; use rendering, schema or policy validation, or an authorized `terraform plan`, for infrastructure. Do not claim success for checks that were not run, and keep static checks distinct from executed tests.

Return `Ready`, `Needs changes` or `Blocked`, labelled **self-review** when the implementation and review use this same agent. Give material findings stable IDs, locations, impact, evidence, disposition (`valid`, `disputed`, `needs evidence`) and resolution/next action. Check reviewer advice against actual requirements/code; reject incorrect advice with evidence rather than implementing it mechanically. Keep failed integration, uncertain evidence and unresolved consequential findings visible. This technical verdict neither supplies human approval nor authorizes integration, publication or deployment.

**Review-only requests change nothing.** When the user asked for a review, make no edits to code, documentation, approval records, task status or batch state; report the findings and the recommended next action in the response and let the user decide. Fix in-scope defects, and write workflow state, only when the surrounding request already authorizes implementation, fixes or that state change — for example a verification run that follows an authorized batch.

Route material requirement, design or batch-scope departures back for human review and reconciliation of the dependent documents.

When corrections are authorized, resolve valid in-scope findings and re-review the changed result, affected dependencies and new regressions within the current effort budget. Preserve finding IDs. Scope-expanding changes need a decision; exhausted effort leaves unresolved findings, not automatic escalation or Ready. The implementation/lifecycle owner persists any authorized task update and `awaiting human review`; for a direct fix the response is the record. Do not add a second ledger or routine verification/completion file.
