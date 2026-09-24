---
name: agentic-sdd-spec
description: "Draft a feature spec from the relevant SRS and roadmap sections as the first of three documents for one combined approval."
---

# Agentic SDD Spec

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for a selected new feature or to repair a material requirement gap in an existing one. The global SRS is product intent; `specs/<nnn-slug>/spec.md` is the outcome contract for this feature.

Read the relevant SRS and global-roadmap sections, applicable existing decisions and related feature summaries. Do not read every feature folder or the whole SRS. Reuse rather than duplicate source text. If the feature's boundary or ownership is unclear, propose it before creating the folder.

Create or refine `spec.md` with the outcome, in-scope and explicitly excluded work, observable acceptance criteria, source-section references, actors or permissions where relevant, important edge cases and unresolved material questions. Include only this feature's requirements; keep design, tasks and batches out of it. Mark the substantive revision.

Ask only questions that change behavior, scope, acceptance, permissions, data or compatibility; at most three at once. Do not silently assume risky facts. If user-visible intent conflicts with the SRS, surface it for a canonical-source decision rather than changing an approved decision silently.

Once the draft is sufficiently clear, continue to `agentic-sdd-plan` and then `agentic-sdd-tasks`: all three are presented together for ONE combined human approval. Do not stop here for a separate spec approval, and do not implement. If the user asked only for a spec, deliver only the spec. If a material unanswered question genuinely blocks the plan, record the assumption or the blocking question and say so honestly.

Record the combined approval once, in this file's approval section or one established tracker entry, tied to the reviewed revisions and scope; the other documents reference it.

Output the path, source pointers, acceptance summary, material open questions and the next document to draft. Do not call an agent's clarity assessment approval.
