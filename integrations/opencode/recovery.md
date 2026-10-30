# Recovery procedure — offline B6 subset

This is a manual reconciliation procedure, **not native recovery certification**.
The [support record](supported-version.md) and [B3 findings](verification.md#open-acceptance-findings)
still control. No service/config/credential change, model call or native
compaction/restart experiment is authorized by this document.

## Before any resumed action

1. Resolve the physical workspace, actual requested mode and latest explicit
   authorization. Reopen the feature's `spec.md` approval and `tasks.md` current
   continuation when feature work applies. A direct fix has no invented feature
   records. Treat old summaries and [handovers](../../docs/handovers/) as pointers,
   not permission. Explicit new batch authority may advance a checkpoint; bare
   “continue” cannot authorize the next batch.
2. Inspect staged, unstaged and relevant untracked changes/deletions. Preserve
   dirty and partial files. Do not reset, remove a workspace, restore a snapshot
   or replay a command merely to make old evidence match. A location move needs
   current identity/permission reconciliation, not a rewritten old receipt.
3. Reconcile current runtime/config/skills, model and tools, allowed effects,
   limits and open findings. User-reported rotation/cap completion is not current
   price/privacy/allowance verification or online-run permission. See [safety](safety.md).
4. Recover observed results, not claimed success: distinguish missing evidence,
   blocked prerequisites, actual failure, cancellation and incomplete deadlines.
   Use [offline receipt checks](../../harness/evidence/README.md) against fresh
   independently observed identity/result/log digests. Unknown scope or changed
   relevant inputs invalidates reuse/Ready. Rerun only authorized affected checks.
5. Reconcile uncertain external/non-idempotent effects before considering retry.
   Ending a stream, disconnecting a UI, cancelling a check, undoing files or
   deleting a sandbox does not prove an agent stopped or reverse external effects.
6. Continue only the actual authorized work. Consume an unchanged completion
   review rather than invoking another; corrections warrant scoped re-review.
   The implementation owner writes permitted continuation state and stops at
   its batch review boundary. A receipt never changes canonical state.

## Deterministic check interruption (observed offline)

The fixture in `tests/harness/test_recovery.py` builds the existing Linux
`sdd-check` runner, starts a synthetic writing check, then signals only that
owned runner or lets its deadline expire. It observes the writer stopping,
an unrelated fixture process surviving, partial files remaining, the later
check not running, private logs and cancelled/incomplete output surviving a
JSON save/reload. It never starts OpenCode or uses a provider credential.
Do not infer native session cancellation or process isolation from this result.

## Native cancellation, compaction and restart (not observed)

Use only a separately authorized isolated session after the native/tool/model
and spending prerequisites are resolved. Use the pinned runtime's supported
cancellation control; this offline procedure intentionally supplies no untested
API endpoint, CLI flag or version-specific keybinding. Observe session/tool
activity actually stopping and reconcile any remaining process or external
effects; do not trust stream closure alone. Preserve partial results.

For compaction/restart/location-move cases, independently compare pre/post
session identity, authorized action, dirty content, open findings and evidence
freshness. Inspect the actual next actions for scope and duplicate-review
violations. A saved JSON reload or static instruction cannot demonstrate that
an agent recalled or obeyed any of these rules. All such agent observations
remain **not_run** under offline B6; T09 completion stays blocked.

Never clone/export the primary native database or auth store to construct this
test. A future online fixture needs its own scoped secret reference, exact
model/data scope and bounded requests/tokens/deadline, with no paid fallback,
delegation or background inference. Do not restart or cancel the user's shared
service or current session while developing the fixture.
