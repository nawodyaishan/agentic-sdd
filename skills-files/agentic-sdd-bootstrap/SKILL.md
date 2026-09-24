---
name: agentic-sdd-bootstrap
description: "Manually establish minimal SDD pointers, packet conventions and approval location using existing repository files."
---

# Agentic SDD Bootstrap

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Run only when explicitly requested. Reuse existing repository instructions, governance and toolchain.

1. Discover the actual top-level SRS, technical specification and roadmap, commonly `Docs/SRS.md`, `Docs/High Level Spec.md` and `Docs/Tasks.md`. Respect `Docs/` case and existing names. Discover `specs/` and any packet index or tracker.
2. In an existing instruction entry point, add only missing pointers: canonical top-level sources; `specs/<nnn-slug>/` as the bounded packet location; spec, plan and task approval evidence location; relevant build/test commands and consequential execution boundaries. Create `AGENTS.md` only if a durable instruction entry point is needed and none exists.
3. Preserve existing packets and approval records. If no packet index exists, use the existing roadmap or directory names for discovery; create a separate index only when finding packets is a real problem.
4. For a new project, establish only the structure/toolchain needed for the first bounded workload within the user's setup scope. Do not create a packet until a workload is selected.

Do not materialize generic constitutions, twelve templates, empty directories, checklists or role charters. Reusable guidance stays with the skill package. Keep client invocation controls, permissions and MCP settings outside portable skill text.

Report actual changes, canonical source paths and any unresolved setup question. A second run with unchanged needs should be a no-op.
