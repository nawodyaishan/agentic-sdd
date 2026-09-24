---
name: agentic-sdd-plan
description: "Draft a feature plan from the draft spec with specialist and tool assignments, then continue to tasks for one combined approval."
---

# Agentic SDD Plan

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for `specs/<nnn-slug>/plan.md`, drafted from this feature's `spec.md` — the draft spec is sufficient input; no separate spec approval is required first. A sufficient top-level technical specification informs this plan; it does not replace the feature design.

Read the spec draft, relevant `Docs/High Level Spec.md` sections, affected architecture/ADRs and targeted code/tests. Record the spec revision used and source links. Cover the approach, affected components, design and contract decisions, and the risks that actually apply. For a bounded established-pattern change, keep it short; add alternatives, migration, security, rollout and failure-mode detail in proportion to real risk.

Assign specialists and tools here, once, for the whole feature: the actual installed skill name, its availability in the active client, and why it applies — or an explicit "no additional specialist needed" with a one-line reason when repository guidance is adequate. Name the tools the work will actually need (local code discovery, versioned library docs, external search, browser checks, repository gates) as conditional selections, not mandatory calls. A role label in a plan is not an invocation. If a named skill or tool is missing, restricted or unsuitable, record that gap instead of silently substituting or inventing an identifier. Task-specific exceptions belong in `tasks.md`, not repeated here.

Surface material conflicts with approved upstream decisions rather than changing them. Continue to `agentic-sdd-tasks`; the spec, plan and tasks are then presented together for ONE combined human approval. Do not stop here for a separate plan approval, and do not implement. If the user asked only for a plan, deliver only the plan.

Use `agentic-sdd-architecture-review` as an extra risk review when warranted; its verdict never replaces the human approval. Draft a contract, ADR, data model, test plan or runbook separately only for an independent consumer or explicit request. Designing infrastructure changes does not authorize applying them; live operations need their own execution authorization.

Output design decisions, affected components, assigned specialists and tools, risks, checks and the next document to draft.
