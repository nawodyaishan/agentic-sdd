---
name: agentic-sdd-router
description: "Route SDD intake and feature work through combined spec/plan/tasks approval, batched implementation and direct fixes."
---

# Agentic SDD Router

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use for SDD intake, feature selection, or resumption. Preserve direct manual invocation of every phase.

## Classify the action first

Before feature intake, identify what the user requested:

| Requested action | Route |
|---|---|
| One named document | Draft only that document. If it is feature-scoped, select or create the feature folder needed to hold it; do not draft downstream documents. |
| Review only | `agentic-sdd-verification-review`; change no project state. |
| Diagnose an error or intermittent/unexpected behavior | `agentic-sdd-debugging`; retain the requested effects/mode and return evidence to the invoking stage. Do not assume diagnosis authorizes a fix. |
| Bounded fix with understood consequences and meaningful verification | Direct fix; create no feature documents or batch state. |
| Research for a named uncertainty | `agentic-sdd-research-spec`, then return to the invoking stage. |
| Resume or continue recorded work | Reconcile documents, current diff and the latest explicit authorization before choosing the next action. |
| A feature outcome | Feature intake below, then drafting. |

Do not make a limited document request, review or direct fix inherit feature setup. Only feature intake selects or creates `specs/<nnn-slug>/`.

## Intake for feature work

Resolve the repository's actual product SRS, high-level technical specification and global roadmap, commonly `Docs/SRS.md`, `Docs/High Level Spec.md` and `Docs/Tasks.md`. Preserve spelling and case. Read their relevant headings/sections and the existing feature index; do not read entire unrelated documents. The top-level documents define product intent, major decisions and roadmap. They do not replace a selected feature's files.

Select one coherent outcome as the next feature and create or reuse `specs/<nnn-slug>/`. A feature may span several sessions — size *implementation batches*, not the feature, for focused execution, verification and comfortable human review. Do not split a feature merely because it holds several dependent tasks; split for independent outcomes, unrelated architectures or unmanageable review burden. If scope is uncertain, show its objective, source references and boundary before creating files. Detail only this feature; leave future work in the roadmap.

## Route within a selected feature

| Readiness | Next action |
|---|---|
| New feature, or a material gap in its documents | `agentic-sdd-spec` → `agentic-sdd-plan` → `agentic-sdd-tasks` in that order, each using the preceding draft; reconcile, then stop for ONE combined human approval of all three. |
| Combined approval recorded, implementation authorized, a batch selected | `agentic-sdd-implement` on ONE batch. Implementation owns the batch lifecycle and invokes completion review once; consume its result and stop for human review. |
| `tasks.md` says `awaiting human review` | Reconcile that state with the latest explicit authorization. A clearly scoped new batch authorization may advance to that batch. A bare "continue" is not authorization for the next batch: report the next action and wait for an explicit decision. |
| A consequential design or operation needs extra scrutiny | `agentic-sdd-architecture-review` before or alongside the combined approval; its verdict never replaces human approval. |
| A named factual question blocks drafting, implementation, review or a direct fix | `agentic-sdd-research-spec`, then return to the invoking stage with its restrictions intact. |
| Explicit whole-feature review, or planned final-batch feature acceptance | `agentic-sdd-verification-review` at feature scope; compare connected behavior and open findings, not just batch pass labels. Review-only remains non-mutating. |
| Meaningful drift, useful lesson, release preparation or explicit request | `agentic-sdd-drift-retro` as needed. |
| Clear scope, understood consequences, meaningful verification available | Direct fix: focused implementation and verification. No `specs/` directory, plan assignment, task entry, batch ID or batch-state write — including in production code. |
| Consequential permissions, data integrity, public contract, migration or live-system change | Not a direct fix: draft the feature documents and obtain the combined approval, plus separate execution authorization. |
| Explicit review-only request | `agentic-sdd-verification-review` for findings and a recommended next action; no edits to code, documents, approval records, task status or batch state. |

Determine stage from the documents *and* the recorded approval and batch state, not filenames alone. An unapproved feature resumes at drafting or at the combined-approval checkpoint — never at implementation. An approved feature resumes at the recorded next action after reconciling the latest explicit authorization. A green test run, a completed batch or the existence of approval is not authorization to begin the next batch. Keep technical verdict, execution outcome and workflow position separate; a batch awaiting a user decision is not automatically successfully finished.

Preserve approval across ordinary progress updates; a material requirement, design or batch-scope change returns the affected changes for review and reconciliation of dependent documents, leaving unaffected approval intact. Approval to develop infrastructure code never authorizes a live apply, deployment or migration.

Bootstrap is explicitly initiated through `agentic-sdd-bootstrap`, not triggered by missing convention files. Routing through this skill and invoking a phase skill directly follow identical rules; the router adds no requirement that direct invocation lacks, and removes none. Verify current code state before trusting continuation notes or old test results.

## Output

Briefly state the classified action, feature or direct-fix scope, current stage, approval and batch state, the missing decision if any, the selected specialist (or "none needed") and the checks. Stop at a needed human checkpoint after making the draft concrete. Continue authorized work when the relevant approval and authorization already exist. Do not spawn subagents unless the user requests them.
