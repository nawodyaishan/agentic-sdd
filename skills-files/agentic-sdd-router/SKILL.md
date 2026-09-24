---
name: agentic-sdd-router
description: "Route SDD project intake and bounded work packets through spec, plan, task approvals and implementation."
---

# Agentic SDD Router

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for SDD intake, packet selection, or resumption. Preserve direct manual invocation of every phase.

## Intake versus packet work

Resolve the repository's actual product SRS, high-level technical specification and global roadmap, commonly `Docs/SRS.md`, `Docs/High Level Spec.md` and `Docs/Tasks.md`. Preserve spelling and case. Read their relevant headings/sections and existing packet index; do not read entire unrelated documents. The top-level documents define product intent, major decisions and roadmap. They do not replace a selected packet's files.

Choose the smallest reviewable workload whose dependencies, verification and review cost fit a focused implementation session. Create or reuse `specs/<nnn-slug>/` for a selected feature/workload. If size is uncertain, show its objective, source references, boundary and candidate role before creating files. Split a packet containing unrelated architectures or many dependent tasks.

## Route within a selected packet

| Readiness | Next action |
|---|---|
| No `spec.md` or slice-specific requirements unclear | `agentic-sdd-spec`: draft or refine `spec.md`; stop for human approval of that revision. |
| Spec approved, no adequate `plan.md` | `agentic-sdd-plan`: draft/refine design and actual specialist assignment; stop for human approval. |
| Plan approved, no adequate `tasks.md` | `agentic-sdd-tasks`: draft/refine bounded tasks, specialists and checks; stop for human approval. |
| Approved spec, plan and tasks; one task ready | `agentic-sdd-implement` on one task or a small approved group, then `agentic-sdd-verification-review`. |
| A consequential design/operation needs extra scrutiny | `agentic-sdd-architecture-review` alongside the applicable plan stage; its verdict never replaces human signoff. |
| Named factual question blocks the current stage | `agentic-sdd-research-spec`, then return to that stage. |
| Meaningful drift, useful lesson, release preparation or explicit request | `agentic-sdd-drift-retro` as needed. |

A missing packet file matters after a packet is selected. Determine stage from both the file and human approval evidence, not filename alone. An existing packet resumes at its first unapproved or materially changed stage. Preserve approval for the same document revision/scope; an agent's `Ready` is not human approval. If the user explicitly approved a defined set of stages in advance, honor its actual scope without repeated ceremony. Do not treat a plan signoff as permission for a live apply, migration or deployment.

Bootstrap is explicitly initiated through `agentic-sdd-bootstrap`, not triggered by missing convention files. For an explicitly requested trivial direct fix, bypass packet creation and use focused implementation and verification. Ordinary new features/workloads use the packet even when top-level documents are complete.

## Output

Briefly state packet or direct-fix scope, current stage, approval evidence or missing decision, selected principal specialist and checks. Stop at a needed human checkpoint after making the draft concrete. Continue authorized work when the relevant approval already exists.
