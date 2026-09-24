---
name: agentic-sdd-implement
description: "Implement one approved batch, or one direct fix, with assigned specialist guidance loaded in the main agent and focused verification."
---

# Agentic SDD Implement

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Two entry paths, with different required inputs and different outputs. The same rules apply whether this skill is reached through `agentic-sdd-router` or invoked directly.

| | Feature batch | Direct fix |
|---|---|---|
| Precondition | Combined approval of `spec.md`, `plan.md`, `tasks.md` recorded, and this batch authorized | Clear scope, understood consequences, meaningful verification available |
| Inputs | Batch task entries, governing spec/plan sections, approval reference, affected code/tests | The request, relevant code and constraints |
| Specialist source | `plan.md` defaults plus any `tasks.md` exception | Request, repository context and available catalog, when useful |
| On completion | Set batch state `awaiting human review` with next action | Report the diff and evidence in the response |

## Feature batch

Read the batch's task entries, the spec and plan sections that govern them, the combined approval reference, relevant linked top-level source sections, and affected code/tests. Do not reload unrelated sections or require optional documents. Confirm the batch is bounded, verifiable and still matches the approved revisions; if a material source or design change invalidates approval, revise and return the affected changes for review rather than proceeding.

Before loading any guidance, resolve the assignments: take the feature's default specialist and tool assignments from `plan.md`, apply any task-specific exception recorded in `tasks.md` — the exception overrides the default for that task — then load only what the current task actually needs. Add a second specialist only for a real domain boundary. Where the assignment is "no additional specialist needed", proceed on repository guidance.

Implement exactly one batch within its boundaries, following established project patterns. Run focused checks plus repository-required gates, diagnose in-scope failures and record commands and results against this code state. Summarize the diff, deviations, checks and remaining issues, then stop: set the batch state to `awaiting human review` in `tasks.md` with the next action, and update the global roadmap only at a meaningful milestone. Do not start the next batch because this one passed its checks; wait for the user to authorize it.

## Direct fix

Read the relevant code, tests and repository constraints only. Do not look for or create `specs/`, `plan.md` or `tasks.md`; a direct fix has no batch ID and writes no batch state, and a missing `specs/` directory is not a blocker. Choose specialist guidance from the request, repository context and the available catalog where it would genuinely help; otherwise work from repository guidance alone.

A bounded bug fix in production code is an ordinary direct fix. Stop and route to feature planning and its own authorization when the change is consequential — permissions or access control, data integrity, public or cross-service contracts, migrations, or operations against live systems — or when scope, consequences or verification are unclear. Size is not the test.

Implement the fix, run focused checks plus repository-required gates, and report the request addressed, the diff, the checks and their results, and any remaining risk in the response.

## Both paths

Load specialist guidance into this main agent; a task label is not loaded guidance. If an assigned specialist or tool is unavailable or unsuitable, report that gap and use an adequate fallback transparently; do not silently substitute or invent an identifier. Use MCPs and CLIs only for a concrete need.

Honor earlier authorization for the same scope; do not re-ask at each phase, but seek new authorization for changed consequences. Do not weaken tests. Drafting a migration or Terraform change does not authorize executing it against real systems. Use `agentic-sdd-verification-review` for completion review.

Use one main agent. Do not spawn subagents unless the user explicitly requests them. No routine completion or drift document.
