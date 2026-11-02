# Final human cycle — session run sheet

**Status: blank worksheet; not started.** Use with the
[final human usage plan](vnext-human-test-plan.md). Fill the fields before
launch; a blank or unchecked row is **not run**, never a pass. This sheet does
not authorize a model call, installation, adoption or any live operation.

## 1. Session identity (selected setup, 2026-11-02)

| Field | Value |
| --- | --- |
| Date / operator | _fill in_ |
| Safe workspace (absolute path) | **Container `/work`**, backed by a fresh host scratch directory. Record the host path outside Git. |
| Agent/client and version | _fill in_ |
| Installed skills state / backup path | CLI binary built from the delivered identity; installations stay inside the container unless separately authorized. |
| OpenCode CLI and service version | _fill in if `OPENCODE_BINARY` is mounted_ |
| Adoption choice | **Preview only.** Applying to the scratch workspace needs a separate explicit go-ahead; the normal setup is out of scope. |
| Model choice | **None.** No native OpenCode/ACP session or provider request; inference stays out of scope. |
| If a model: exact id/provider, data scope, request/token/deadline limits, spend control | _not applicable in this cycle_ |
| Timebox and stop conditions | **90 minutes, one sitting.** Stop on the first unexpected write, permission prompt, missing prerequisite or evidence inconsistency. |
| Evidence location (sanitized, outside Git) | _fill in_ |

Container command (from the repository root):

```sh
make build
mkdir -p /tmp/opencode/vnext-cycle-work
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work \
  tests/vnext-container.sh
# Optional read-only mount of the host binary, only if you need it:
# OPENCODE_BINARY=$(command -v opencode) SDD_CONTAINER_WORKSPACE=... tests/vnext-container.sh
```

The recipe never pulls an image, uses `--network=none`, drops capabilities,
has a read-only root and repo, mounts no host home or credentials, and mounts
only the chosen scratch directory read-write. Inside `/work`, run only the
model-free scenarios below (installer preview/no-op/backup/restore, candidate →
preflight → config preview, evidence snapshot/validate/export/receipt on a
synthetic project). AI-driven skill behavior is exercised host-side with this
assistant; the container cannot substitute for a model session.

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
