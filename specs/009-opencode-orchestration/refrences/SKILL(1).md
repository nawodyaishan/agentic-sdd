---
name: agentic-sdd-orchestrate
description: Use when the human explicitly requests bounded delegation or optional agent orchestration of engineering work, or enables the Agentic-SDD orchestration experience. The main agent scopes specialist tasks, supervises subagents, reconciles evidence, and reports back while retaining human approval and existing Agentic-SDD authority. Not for ordinary direct fixes or single-tool actions.
---

# Agentic-SDD — Orchestrate (DRAFT, OPT-IN)

**Role:** Instructions for the **main human-facing agent**, not for a child. No authority to call a model, spend credits, inspect secrets, write global configuration, or execute a batch is created by loading this skill.

## 1. Preserve the existing owners

- Load the current Agentic-SDD router and its workflow-policy/human-input references. **Do not invent a new approval process.**
- For features, `spec.md` §Approval owns combined approval, while `tasks.md` owns the selected batch and continuation. The human explicitly authorizes a batch and reviews its outcome before another begins.
- `agentic-sdd-implement` owns completion review; `verification-review`, `debugging`, `teach`, and other existing skills retain their existing responsibilities.
- An optional OpenCode `sdd-orchestrator` *agent profile* does not choose the effective main-session model on its own. Check the actual model if capability, privacy, or spend matters.

## 2. Determine whether delegation is appropriate

First classify the human's intent:

- **Direct:** One-file read, short search, simple Git/terminal action, small understood fix, or no delegation requested. The main uses direct tools and does not spawn a child.
- **Assist:** The human has opted into optional orchestration; one **read-only** specialist may inspect bounded non-sensitive context when a separate child is genuinely worthwhile.
- **Delegate:** The human requests specialist implementation, tests, documentation, advice, research, or review. Verify the exact existing authorization and effective permissions before dispatch; do not infer authority from the presence of a role.
- **Learn:** Preserve the user's intended hands-on work. Prefer questions, hints or bounded evidence over implementation unless the human changes the requested mode.
- **Blocked:** Missing approval, unsafe path isolation, incompatible model/data policy, unknown partial edits, disallowed network/cost, or otherwise insufficient authority: stop or ask one material question through the parent.

**Do not delegate merely because another agent exists or a free model is offered.** Prefer one suitable child at a time; never start parallel writers in one checkout or allow nested delegation. Read-only parallel work is deferred until the runtime and task independence are tested and explicitly approved.

## 3. Before dispatch

1. Inspect the actual requested scope, approval and current worktree/commit identity; reread only necessary source.
2. State the intended child role, objective, why the child is useful, scope/exclusions, expected evidence, and the required human permissions.
3. Check **effective** model/provider/variant, context and modality limits, privacy and network eligibility, allowance/charges, tool permissions and whether a partial task already exists.
4. For **writers, tests with side effects, paid/cross-lane selection, external research, image disclosure, or independent review**, require the corresponding explicit authorization. Feature 006's unsafe native isolated reviewer must not be invoked as a generally safe reviewer.
5. Load [dispatch contract](references/dispatch-contract.md) and issue a **small self-contained packet** using source links and precise file/revision identifiers, not a full conversation history.

## 4. During delegated work

- Child follows its **own** effective permission rules; prompt constraints are not an OS sandbox.
- Child may return `needs_input` to the parent, never bypass the parent to request human authority.
- No child may commit, merge, push, deploy, change credentials, run new paid inference, spawn further children, or begin another SDD batch merely from assignment wording.
- If model/provider errors, partial writes, contradictory evidence or unclear permissions occur, use [failure and reconciliation](references/failure-and-reconciliation.md). **Fallback is manual by default**; a written model ladder is not an auto-failover engine.
- If direct tools are sufficient, use them. For long logs, retain full private evidence and pass only accurate excerpts/references. See [context and cost](references/context-and-cost.md).

## 5. When the child finishes

1. Treat child output as a **claim**. Independently inspect relevant source/diff, exit codes, failure paths and fresh evidence.
2. Compare actual work with the original approved scope and task packet; mark missing/unsupported claims unknown or blocked.
3. Reconcile shared-state or partial-write effects. Do **not** automatically trigger another model, another reviewer, or another batch.
4. Report **one concise human-facing result**: what changed, what was actually verified, what failed, agent/model identities where observed, usage and cost where known, risks, and the next action that the human may authorize.
5. Stop at the existing human review boundary. A model's `Ready` verdict is not approval.

## 6. Never assume

- That the latest default model is the model running in this session.
- That free models guarantee privacy, zero cost to the overall workflow, or sufficient output capacity.
- That a connected MCP entry is available or permitted for this task.
- That source-configured permission denials prevent symlink/path escapes.
- That a second model's review is independent evidence of correctness.
- That compaction summaries, handoffs or subagent logs supersede canonical approvals.

**Success:** Deliberate delegation, narrowly sufficient context, verifiable outputs, minimal redundant model calls, and clear human decisions—not maximum agent count.
