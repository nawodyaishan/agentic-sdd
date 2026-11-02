# Final human cycle — session run sheet

**Status: blank worksheet; not started.** Use with the
[final human usage plan](vnext-human-test-plan.md). Fill the fields before
launch; a blank or unchecked row is **not run**, never a pass. This sheet does
not authorize a model call, installation, adoption or any live operation.

## 1. Session identity (fill in)

| Field | Value |
| --- | --- |
| Date / operator | |
| Safe workspace (absolute path) | |
| Agent/client and version | |
| Installed skills state / backup path | |
| OpenCode CLI and service version | |
| Adoption choice: observe / preview / separately authorized apply | |
| Model choice (or **none**) | |
| If a model: exact id/provider, data scope, request/token/deadline limits, spend control | |
| Timebox and stop conditions | |
| Evidence location (sanitized, outside Git) | |

Stop before starting if any of these is unknown: workspace safety, installed
version/backup, rollback path, model scope/limits (when a model is used), or
whether an operation is authorized.

## 2. Pre-flight checks

- [ ] Safe/disposable workspace confirmed; unrelated worktree untouched.
- [ ] Current instructions/skills and runtime identity recorded.
- [ ] No credentials, auth stores, environment dumps or raw logs to be copied.
- [ ] Rollback path known: skill backup/restore and/or config backup + digest
      check.
- [ ] Model-free first sitting selected (recommended), or model scope/limits
      explicitly agreed.
- [ ] Timebox and stop conditions agreed.

## 3. Recommended first sitting — no model, no live adoption

Run the plan's scenarios that fit; record each row. Suggested order:

| # | Plan scenario | Result (pass/fail/blocked/not run) | Evidence / finding |
| --- | --- | --- | --- |
| 1 | Baseline observation | | |
| 2 | Direct fix (no feature docs) | | |
| 3 | Spec-only + combined approval | | |
| 4 | One implementation batch + review | | |
| 5 | Review-only (no edits) | | |
| 6 | Interruption, resume/handoff | | |
| 7 | Bug investigation / diagnosis | | |
| 8 | Clarification / grilling | | |
| 9 | Questionnaire draft / late answer (optional) | | |
| 10 | Explanation (“wait, what?”) | | |
| 11 | Teaching (focused; persistent only if chosen) | | |
| 12 | Inactive candidate + config **preview** | | |
| 15 | Deterministic check + receipt export/reload | | |
| 16 | Whole-cycle reconciliation | | |

Scenario 12's preview writes nothing; do not apply. Scenarios 9 and 11 are
optional and may be skipped without failing the sitting.

## 4. Separate later sitting — only with new authorization

- [ ] Scenario 13 — adoption/rollback on a chosen setup (preview, backup,
      apply, verify, rollback; preserve unrelated config).
- [ ] Scenario 14 — real native pending-human behavior. **Skip while admission
      remains blocked**; record `not run — native admission blocked`.

## 5. Closeout

```text
Sitting and date:
Setup/version and safe workspace:
Scenarios run / skipped:
Observed results and evidence (sanitized):
Finding IDs / next decision:
Feature status: implementation delivered; human usage validation not run
```

- Reconcile notes into the relevant feature continuation when authorized; do
  not mark feature 005 `Ready`, certified or beneficial from this sheet alone.
- Preserve open 004 findings and the native-admission block unless separately
  observed and accepted.
- Any fix/retest cycle is a new user-directed decision, not an automatic loop.
