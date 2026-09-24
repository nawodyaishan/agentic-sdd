---
name: agentic-sdd-tasks
description: "Draft packet execution tasks from an approved plan, assign specialists and checks, then obtain human task signoff."
---

# Agentic SDD Tasks

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for `specs/<nnn-slug>/tasks.md` after the packet's `plan.md` has applicable human approval. The top-level `Docs/Tasks.md` remains the project roadmap, not the packet execution ledger.

Read the approved packet spec and plan, relevant roadmap item and verification conventions. Create or refine a sequence of small, independently verifiable tasks. Each task needs an objective, acceptance result, dependency/order, status, focused verification and actual installed specialist skill(s) with a reason. Put shared constraints, source/approval pointers and execution boundaries once at packet level; repeat only task-specific exceptions. Identify likely modules, but do not invent exact file ownership before design makes it knowable.

Choose the first ready task or small approved group that fits a focused session. Split a broad workload across packets or tasks when dependencies, unrelated areas or review cost warrant it. Do not generate parallel-agent groups by default. A role or skill name written in `tasks.md` does not load it; the implementer must invoke the relevant installed skill in the main agent.

Mark the substantive task-list revision draft and record its human approval once in `tasks.md` or an established tracker, separate from spec and plan signoffs. Stop for the user's task decision before implementation unless explicit prior authorization covers this exact stage/scope. Human approval for design does not authorize live deployment, Terraform apply or migration execution.

At milestones, keep compact continuation state in the packet task section: current task, source/approval pointers, changed paths, checks with code state and next issue. Update the global roadmap at meaningful packet status changes, not after every command.

Output task path, first ready task, specialist availability, verification and approval status.
