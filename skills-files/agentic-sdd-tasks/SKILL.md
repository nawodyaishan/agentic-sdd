---
name: agentic-sdd-tasks
description: "Draft ordered feature tasks and explicit execution batches, then present spec, plan and tasks for one combined approval."
---

# Agentic SDD Tasks

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for `specs/<nnn-slug>/tasks.md`, drafted from this feature's `spec.md` and `plan.md` drafts — no separate plan approval is required first. The top-level `Docs/Tasks.md` remains the project roadmap, not this feature's execution ledger.

Read the spec and plan drafts, the relevant roadmap item and the repository's verification conventions. Write ordered, independently verifiable tasks: objective, acceptance result, dependencies/order, status and focused verification. Identify likely modules; do not invent exact file ownership before design makes it knowable. Specialist and tool assignments live once in `plan.md`; record only task-specific exceptions here, including an explicit "no additional specialist needed" where that differs from the plan. An exception overrides the feature default for that task alone, so state which default it replaces and why.

## Batches

Group tasks into explicit execution batches sized for focused execution, verification and comfortable human review — not by file, line or token limits, and not by splitting the feature itself. Each batch records:

- Batch ID and the task IDs it includes
- Outcome and completion conditions
- Verification for that batch
- State, including `awaiting human review`
- Next action

After combined approval and authorization to implement, exactly one batch executes, is verified, and then stops for the user's review. Completed checks never advance the batch automatically.

## Approval and continuation

Reconcile the three documents, then present spec, plan and tasks together for ONE combined human approval. Record it once in this feature's agreed location (by convention `spec.md`) or an established tracker entry, tied to the reviewed revisions and scope; reference it from here rather than keeping a second approval flag. Do not implement before that approval. Approval of these documents does not authorize a live deployment, Terraform apply or migration execution.

At batch boundaries, keep compact continuation state here: current batch and task IDs, approval reference, changed paths, checks with the code state they were run against, current state and next action. Update the global roadmap at meaningful feature status changes, not after every command. Verify current code state before trusting these notes.

Output the task path, batch list, the first ready batch, specialist/tool exceptions, verification and approval status.
