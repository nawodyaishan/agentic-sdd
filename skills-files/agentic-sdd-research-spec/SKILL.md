---
name: agentic-sdd-research-spec
description: "Resolve a named fact blocking the current packet spec or plan and keep evidence in that packet."
---

# Agentic SDD Research Spec

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use only for an explicit research request or named uncertainty that affects the current packet's requirements, design, review or task. Return to the interrupted stage after answering it.

State the question, affected packet decision, relevant version/environment and evidence threshold. Read relevant local source and versioned documentation first. If still unresolved, use an available documentation or search tool appropriate to the concrete need. Prefer primary sources and fetch only decision-relevant material; do not assume a particular MCP identifier is installed or reachable.

Record the answer, source/version and remaining uncertainty in the relevant `spec.md` or `plan.md` section. Respect the stage's draft/approval state: material new evidence that changes approved requirements or design needs review of that affected stage. Create `research.md` only for a separate consumer, repository convention or explicit request.

Do not turn research into an alternate complete spec, silently add scope or claim resolution without evidence. Output the finding, source pointers, packet impact and next stage.
