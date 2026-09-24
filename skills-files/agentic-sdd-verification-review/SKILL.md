---
name: agentic-sdd-verification-review
description: "Verify a completed batch or a direct fix against acceptance criteria and evidence; review-only requests return findings without edits."
---

# Agentic SDD Verification Review

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use after implementation or for an explicit review. The same rules apply whether this skill is reached through `agentic-sdd-router` or invoked directly.

| | Feature batch | Direct fix |
|---|---|---|
| Inputs | The diff, approved `spec.md`/`plan.md` sections, the batch's `tasks.md` entries, the combined approval reference, applicable contracts and test evidence | The request, relevant code and constraints, the diff and verification evidence |
| Compare against | The feature's acceptance criteria, design boundaries and batch outcome | The stated request and the constraints the change touches |
| State written (when authorized) | Batch state `awaiting human review` with next action | None — evidence goes in the response |

For a direct fix, do not require, look for or create feature documents; a missing `specs/` directory, plan assignment, task entry or batch ID is not a gap to report.

Consult linked top-level source sections only where the change or a conflict requires them. In both paths check scope, compatibility, permissions and security, error handling, tests, and significant source or roadmap drift.

Reuse a prior check only if code, relevant dependencies/configuration, inputs and environment still match. Rerun affected, failed or repository-required checks. Choose focused tests and static checks for changed behavior; use rendering, schema or policy validation, or an authorized `terraform plan`, for infrastructure. Do not claim success for checks that were not run, and keep static checks distinct from executed tests.

Return `Ready`, `Needs changes` or `Blocked` with concrete file/line findings, checks and results, deviations from the approved scope and unresolved risk. This verdict is a technical assessment: it never replaces the human combined approval or the human batch review, and never authorizes merge, deployment or live infrastructure work.

**Review-only requests change nothing.** When the user asked for a review, make no edits to code, documentation, approval records, task status or batch state; report the findings and the recommended next action in the response and let the user decide. Fix in-scope defects, and write workflow state, only when the surrounding request already authorizes implementation, fixes or that state change — for example a verification run that follows an authorized batch.

Route material requirement, design or batch-scope departures back for human review and reconciliation of the dependent documents.

When the request authorizes it, summarize evidence in the current task entry and set the batch state to `awaiting human review` with its next action rather than continuing; for a direct fix, the response is the record. Create `verification.md` only for an independent audit record, established convention or explicit request; avoid a separate no-drift file.
