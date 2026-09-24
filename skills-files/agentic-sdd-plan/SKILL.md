---
name: agentic-sdd-plan
description: "Draft a packet plan from an approved spec, assign actual specialist skills and obtain human plan signoff."
---

# Agentic SDD Plan

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for `specs/<nnn-slug>/plan.md` after the packet's `spec.md` has applicable human approval. A sufficient top-level technical specification informs this plan; it does not replace the packet design.

Read the approved spec, relevant `Docs/High Level Spec.md` sections, affected architecture/ADRs and targeted code/tests. Record the selected spec revision and source links. For a bounded established-pattern change, keep the plan short: affected modules, design/contract boundaries, specialist assignment, focused checks and recovery considerations. Add alternatives, migration, security, rollout and failure-mode detail in proportion to actual risk.

Assign the architect or domain specialist(s) by actual installed skill name and client availability. State why each applies. One main agent will load the principal skill; a second skill only for a real boundary crossing. A role label in a plan is not an invocation. If a named skill is missing or unsuitable in the active client, record the gap rather than silently substituting.

Create or refine `plan.md` and mark its substantive revision draft with a distinct human plan-approval state in its approval section or established tracker. Do not write `tasks.md` before that decision unless prior explicit authorization covers the exact plan scope/stage. If design changes materially after approval, seek review only for the affected revision.

Use `agentic-sdd-architecture-review` as an additional risk review when warranted; it does not replace the ordinary human plan signoff. Draft a contract, ADR, data model, test plan or runbook separately only for an independent consumer or explicit request. Live operations need their own execution authorization.

Output design decisions, assigned specialist(s), checks, risks and approval status. Stop at the reviewable plan when human signoff is still needed.
