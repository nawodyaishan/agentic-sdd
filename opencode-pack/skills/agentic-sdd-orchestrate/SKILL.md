---
name: agentic-sdd-orchestrate
description: Use when the human explicitly requests bounded delegation or enables the optional Agentic-SDD orchestration experience. The main agent classifies the request, issues compact task briefs, supervises children, reconciles evidence and reports back, while retaining human approval and the existing Agentic-SDD authority. Not for ordinary direct fixes or single-tool actions.
---

# Agentic-SDD — Orchestrate (opt-in)

**Role:** instructions for the **main human-facing agent**, not for a child. Loading this skill creates no authority to call a model, spend credits, inspect secrets, write global configuration or start an SDD batch.

## 1. Preserve the existing owners

Read the shared [workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context, and consult [human input](../agentic-sdd-router/references/human-input.md) before asking the human anything this policy already answers.

- Do not invent a second approval process. For features, `spec.md` §Approval owns combined approval and `tasks.md` owns the selected batch and continuation.
- `agentic-sdd-implement` owns implementation and its completion review; `agentic-sdd-verification-review` owns an explicitly requested review. Delegation never reassigns those responsibilities to a child.
- The optional `sdd-orchestrator` profile does not choose the effective main-session model. Read the actual model when capability, privacy or spend matters.

## 2. Classify before you delegate

Use the five labels in [context and cost](references/context-and-cost.md): `D0` direct, `D1` cheap read-only, `D2` authorized writer, `D3` advisor/research/vision, `D4` critical-main, and `blocked`.

Never delegate merely because another agent exists or a free model is offered. Prefer one suitable child at a time; never run parallel writers in one checkout and never allow nested delegation.

## 3. Before dispatch

1. Inspect the requested scope, the existing approval and the current worktree identity; reread only the source you need.
2. State the child role, objective, why a child is useful, scope and exclusions, expected evidence, and the permissions involved.
3. Check the **effective** model, context and modality limits, privacy and network eligibility, allowance or charges, tool permissions, and whether a partial task already exists.
4. Require the corresponding explicit authorization for writers, side-effecting tests, paid or cross-lane model selection, external research, image disclosure and independent review.
5. Issue the brief defined in [the dispatch contract](references/dispatch-contract.md): a small self-contained packet built from source links and precise file or revision identifiers, not a full conversation transcript.

## 4. During delegated work

- A child follows its **own** effective permission rules; prompt constraints are not an OS sandbox.
- A child may return `needs_input` to the parent but never bypasses the parent to obtain human authority.
- No child may commit, merge, push, deploy, change credentials, start new paid inference, spawn further children, or begin another SDD batch merely from assignment wording.
- Use [failure and reconciliation](references/failure-and-reconciliation.md) for model errors, partial writes, contradictory evidence or unclear permissions. Fallback is manual by default; a written ladder is not an automatic failover engine.
- When direct tools suffice, use them. Keep full private evidence for long logs and pass only accurate excerpts and references.

## 5. When the child finishes

1. Treat child output as a **claim**. Independently inspect the relevant source and diff, exit codes, failure paths and fresh evidence.
2. Compare the actual work with the approved scope and the brief; mark missing or unsupported claims as unknown or blocked.
3. Reconcile shared-state or partial-write effects. Do not automatically trigger another model, another reviewer or another batch.
4. Report **one concise human-facing result**: what changed, what was actually verified, what failed, observed agent and model identities, usage and cost where known, risks, and the single next action the human may authorize.
5. Stop at the existing human review boundary. A model's `Ready` verdict is not approval, and completion review is a single pass, not a second implementation.

## 6. Never assume

- That the latest default model is the model running in this session.
- That free models guarantee privacy, zero cost to the workflow, or enough output capacity.
- That a connected MCP entry is available or permitted for this task.
- That configured permission denials prevent symlink or path escapes.
- That a second model's review is independent evidence of correctness.
- That compaction summaries, handovers or child logs supersede canonical approvals.

**Success:** deliberate delegation, narrowly sufficient context, verifiable outputs, minimal redundant model calls, and clear human decisions — not maximum agent count.
