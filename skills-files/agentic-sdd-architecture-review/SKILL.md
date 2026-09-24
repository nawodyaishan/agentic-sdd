---
name: agentic-sdd-architecture-review
description: "Review consequential packet architecture and operational risks without replacing human spec, plan or task signoffs."
---

# Agentic SDD Architecture Review

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use when a packet introduces new architecture, sensitive permissions, consequential API or data change, migration, production operation, or when explicitly requested. Routine packets still receive the user's ordinary spec, plan and task signoffs; they do not require this extra review.

Read relevant approved `spec.md`, draft/approved `plan.md`, top-level technical decisions, affected contracts/code and recovery notes. Evaluate the changed decision rather than reloading an entire packet or all upstream documents. Check actual compatibility, data ownership, trust boundaries, dependencies, failure modes, tests, rollout and practical recovery risks.

Return `Ready`, `Needs changes`, or `Blocked` as the agent's technical verdict with concrete findings. It is never a human signature. Preserve the plan-stage human approval evidence in its single location; a separate review record can point to it without maintaining a second approval flag. A preexisting genuine human approval for the same revision/scope remains valid. A legacy agent-only `Approved` label does not.

If a consequential decision changes after signoff, prepare the revised design and seek human review for that affected stage. Keep drafting permission separate from authorization to execute on real data or infrastructure. Create `review.md` only for an independent audit need, established convention or explicit request.

Output verdict, affected plan decision, approval reference and remaining execution boundary. Do not spawn an independent reviewer unless the user requests delegation.
