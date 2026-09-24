---
name: agentic-sdd-implement
description: "Implement one approved packet task with the assigned specialist in the main agent and focused verification."
---

# Agentic SDD Implement

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use after applicable human signoffs for the selected packet's `spec.md`, `plan.md` and `tasks.md`. The exception is an explicitly requested trivial direct fix outside the packet workflow.

## Before editing

Read the current approved task section, packet spec/plan sections that govern it, their approval references, relevant linked top-level source sections, affected code/tests and repository constraints. Do not require every optional packet document or reload unrelated sections. Confirm the task is bounded, verifiable and still matches the approved revisions. If a material source/design change invalidates approval, revise and re-review only the affected stage.

Resolve the task's principal specialist against the active client's installed catalog and load its skill into this main agent. Add a second specialist only for a real domain boundary. If unavailable or unsuitable, report that gap and use repository guidance only when adequate; do not silently substitute. A task label is not a loaded skill.

## Execute and verify

Implement one ready task or a small approved group within boundaries. Follow established project patterns. Honor earlier authorization for the same scope; do not ask again at each phase. Seek new authorization for changed consequences. Drafting a migration or Terraform change does not authorize executing it against real systems.

Run focused checks plus repository-required gates, diagnose in-scope failures and record commands/results for this code state. Do not weaken tests. Summarize the diff, deviations, checks and remaining issues; update the packet task status and, at a meaningful milestone, the global roadmap. Use `agentic-sdd-verification-review` for completion review. Do not start the next task merely because the current one passed checks; honor the user's requested review boundary.

Use one main agent. Do not delegate without the user's explicit request. No routine completion or drift document.
