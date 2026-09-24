---
name: agentic-sdd-research-spec
description: "Resolve a named fact blocking a feature draft or a direct fix, and keep the evidence where that work lives."
---

# Agentic SDD Research Spec

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use only for an explicit research request or a named uncertainty that actually blocks the current work — a feature's spec, plan, tasks, review or batch, or a direct fix. Return to the interrupted stage after answering it.

State the question, the decision it affects, the relevant version/environment and the evidence threshold. Read relevant local source and versioned documentation first. If still unresolved, select an available tool by concrete need: local code discovery for cross-module questions, a versioned documentation tool for library behavior, a web search/fetch tool for external research. Prefer primary sources, fetch only decision-relevant material, and do not assume a particular MCP identifier is installed or reachable — report an unavailable capability and use an adequate fallback transparently.

Record the answer, source and version, and remaining uncertainty where the work lives: the relevant `spec.md` or `plan.md` section for feature work, or the response and the change itself for a direct fix. Never create feature documents just to hold research for a direct fix.

Respect the approval state: material new evidence that changes approved requirements or design returns the affected changes for human review and reconciliation of dependent documents. Create `research.md` only for a separate consumer, repository convention or explicit request.

Do not turn research into an alternate spec, silently add scope, or claim resolution without evidence. Output the finding, source pointers, impact and the next stage. Do not spawn subagents unless the user requests them.
