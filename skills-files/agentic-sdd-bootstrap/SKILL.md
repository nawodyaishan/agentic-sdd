---
name: agentic-sdd-bootstrap
description: "Manually establish minimal SDD pointers, feature conventions and the single combined-approval location using existing repository files."
---

# Agentic SDD Bootstrap

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Run only when explicitly requested. Reuse existing repository instructions, governance and toolchain.

1. Discover the actual top-level SRS, technical specification and roadmap, commonly `Docs/SRS.md`, `Docs/High Level Spec.md` and `Docs/Tasks.md`. Respect existing names and case. Discover `specs/` and any feature index or tracker.
2. In an existing instruction entry point, add only the missing pointers: canonical top-level sources; `specs/<nnn-slug>/` as the feature location for spec, plan and tasks; the single location recording the combined human approval of those three; where batch state and continuation notes live; relevant build/test commands; and the boundary between developing infrastructure code and executing it live. Create `AGENTS.md` only if a durable instruction entry point is needed and none exists.
3. Preserve existing features and approval records, including approvals recorded under an earlier per-stage convention — note their actual scope rather than rewriting them. If no feature index exists, use the existing roadmap or directory names for discovery; create a separate index only when finding features is a real problem.
4. For a new project, establish only the structure and toolchain needed for the first feature within the user's setup scope. Do not create a feature folder until one is selected.

Do not materialize generic constitutions, templates, empty directories, checklists or role charters. Reusable guidance stays with the skill package. Keep client invocation controls, permissions and MCP settings in client-specific configuration, outside portable skill text, and do not change installed clients here.

Report actual changes, canonical source paths and any unresolved setup question. A second run with unchanged needs should be a no-op.
