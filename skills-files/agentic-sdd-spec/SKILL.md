---
name: agentic-sdd-spec
description: "Draft or refine a bounded packet spec from the relevant SRS and roadmap sections, then obtain human signoff."
---

# Agentic SDD Spec

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for a selected new packet or to repair a material requirement gap in an existing one. The global SRS is product intent; `specs/<nnn-slug>/spec.md` is the approval contract for this workload.

Read the relevant SRS and global-roadmap sections, applicable existing decisions and related packet summaries. Do not read every packet or the whole SRS. Reuse rather than duplicate source text. If the workload size or ownership is unclear, propose its boundary before creating the folder.

Create or refine `spec.md` with a concise objective, source-section references, in/out scope, actors or permissions where relevant, observable acceptance criteria, important edge cases and unresolved material questions. Include only slice-specific requirements. Mark the substantive revision draft and record the packet-spec approval state in its own approval section or established tracker. A separate `clarify.md` or requirements checklist needs a distinct consumer or explicit request.

Ask only questions that change behavior, scope, acceptance, permissions, data or compatibility; at most three at once. Do not silently assume risky facts. If user-visible intent changes from the SRS, flag it for a canonical-source decision.

Stop after the reviewable draft for the user's spec decision. Do not draft `plan.md` or `tasks.md` until this spec revision/scope has human approval, unless a prior explicit authorization covers that exact stage. If an approved spec is unchanged, reuse its evidence without another request.

Output the path, source pointers, acceptance summary, material open questions and human-approval status. Do not call an agent's clarity assessment approval.
