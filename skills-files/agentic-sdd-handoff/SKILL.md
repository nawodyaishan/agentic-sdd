---
name: agentic-sdd-handoff
description: Prepare a sanitized canonical-first transfer when the user requests a handoff or repository guidance requires a session handover.
disable-model-invocation: true
---

# Agentic SDD Handoff

Read the [shared policy](../agentic-sdd-router/references/workflow-policy.md) once
per unchanged context. Use actual repository recovery/storage conventions, not
the temporary-only storage rule from an upstream example.

1. Resolve current mode, requested next-session focus, actual authorization and
   the current staged/unstaged/untracked work. Preserve partial and unrelated work.
2. Within existing write authority, reconcile canonical approval/continuation
   and evidence first. A review-only/explanation mode must not change them;
   report stale state and proposed corrections in the response instead.
3. Locate the consuming repository's handover convention. Use its required
   location/structure and a non-colliding session filename. If no canonical
   location/write scope is established, return a response handoff or seek the
   material location decision; do not invent a new progress ledger.
4. Write concise pointers to batch position and next authorized work, actions
   taken, actual decisions/open choices, verified facts, finding IDs, next
   permitted action, guardrails and first safe commands. Link specs/tasks/diffs
   and evidence rather than copying their bodies or the conversation history.
5. Check that a fresh receiver can distinguish delivered/incomplete/blocked
   work, evidence freshness and actual consent. Redact sensitive data; never
   copy keys, account records, raw private traces or hidden reasoning. A
   suggested-skill list, if useful, is advisory and cannot grant new authority.

Temporary transfer copies may supplement an authorized canonical handover, not
replace it. A receiver re-reads actual files and reconciles authority before
continuing; neither handoff nor compaction supplies a new batch or live permission.

Adaptation: [source and license](../agentic-sdd-router/references/matt-pocock-notice.md).
