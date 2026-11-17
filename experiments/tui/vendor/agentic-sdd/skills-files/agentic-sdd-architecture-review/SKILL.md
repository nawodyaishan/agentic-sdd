---
name: agentic-sdd-architecture-review
description: "Review consequential feature architecture and operational risk as a technical verdict, without replacing the human combined approval."
---

# Agentic SDD Architecture Review

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use when a feature introduces new architecture, sensitive permissions, a consequential API or data change, a migration, a production operation, or when explicitly requested. Routine features go straight to the user's combined approval; they do not need this extra review.

Read the relevant `spec.md` and `plan.md` sections (draft or approved), top-level technical decisions, affected contracts and code, and recovery notes. Evaluate the changed decision rather than reloading the whole feature or all upstream documents. Check actual compatibility, data ownership, trust boundaries, dependencies, failure modes, tests, rollout and practical recovery risk.

Return `Ready`, `Needs changes` or `Blocked` as the agent's technical verdict with concrete findings. It is never a human signature. This review informs the single combined human approval of spec, plan and tasks; it does not add an approval gate of its own and does not create a second approval flag — reference the one approval record. A preexisting genuine human approval for the same revisions and scope remains valid; a legacy agent-only `Approved` label does not.

An explicit review request returns findings and a recommended next action, and changes nothing — no edits to code, documents, approval records, task status or batch state. Revise the design or write workflow state only when the surrounding request already authorizes it. If a consequential decision changes after approval, prepare the revised design, reconcile the dependent documents and return the affected changes for human review. Keep drafting permission separate from authorization to execute against real data or infrastructure. Create `review.md` only for an independent audit need, established convention or explicit request.

Output the verdict, the affected design decision, the approval reference and the remaining execution boundary. Do not spawn an independent reviewer unless the user requests delegation.
