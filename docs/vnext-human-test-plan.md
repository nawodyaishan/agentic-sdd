# vNext final human usage cycle — feature 005

**Status:** the user subsequently reported the cycle complete and accepted its
results for the pinned image; individual scenario outcomes are unspecified. See
the [continuation](../specs/005-vnext-completion/tasks.md#continuation-record).
**Step-by-step testing instructions are in the [final human test
guide](vnext-human-test-guide.md).** The fill-in [session run
sheet](vnext-human-cycle-run-sheet.md) records your results; a blank row is not
a pass. A model-free container recipe (`tests/vnext-container.sh`) hosts the
user-driven sandbox without network, credentials or host home. This document
replaced the 004-era full-vNext draft at 005/B5 integration.
It covers one integrated human cycle for the selected implementation
(core R0–R2 plus all seven helpers, OpenCode only). R3/R4 extensions remain
explicit non-goals. Preparing this plan is not a pass, not a model/usage test,
not approval of any live operation and not authorization to install or adopt a
runtime configuration.

Evidence, finding identities and current workflow position stay authoritative in
[005 tasks](../specs/005-vnext-completion/tasks.md#continuation-record),
[004 tasks](../specs/004-vnext-harness/tasks.md#continuation-record) and the
[004 acceptance map](../harness/acceptance.md). 004's B3-F1–F4, B5-F1,
B6-F1/F2, remaining B7-F1, B8-F1/F2 and S0-F1/F2 are carried; nothing here
closes them. Development checks are recorded in 005 tasks, not this plan.

## What is delivered, and what remains unobserved

Delivered at source level by 005 (development-verified, not runtime-certified):

- the canonical Agentic SDD workflow plus seven helper capabilities
  (`grill-me`, `grilling`, `questionnaire`, `handoff`, `wait-what`, `teach`,
  `writing-for-agents`) in the namespaced 18-ID skill package, with
  preview-by-default installation, backups, no-op and restore;
- a file-only inactive OpenCode candidate/preflight path and a separate,
  digest-checked config preview/apply/rollback transaction;
- a narrow ACP session client, strict-admission blocked by default, and
  pending-human stop normalization with private, review-only summaries;
- the production v1 evidence consumer (scoped identity, validation, safe
  export) and checks-derived receipt projection with optional native
  observations.

Explicitly **not** observed or certified: real model-originated tools,
permissions or canonical-path isolation; model/selection/price/privacy/
allowance compatibility; native cancellation, compaction, restart or location
move; actual adoption/rollback on the user's setup; macOS runtime; and any
cost, reliability or useful-benefit claim. Strict native prompt admission is
still denied because token/egress/accounting/credential controls are
unverified. Native recovery, not any fixture count, remains the gap.

## Roles and authority

- The **user** chooses the workspace, setup, whether to adopt configuration,
  whether to use a real model/provider, and every live operation. The user runs
   the hands-on scenarios in their selected environment. The accepted cycle used
   the pinned container, not the normal setup.
- The **assistant** may explain steps, inspect evidence the user chooses to
  share, and reconcile notes when authorized. It must not run the human tests,
  access credentials, copy auth stores, change the shared service, install to a
  real home or call a model unless a separate explicit authorization names the
  exact scope, model and limits.
- This plan creates no allocation. Existing conditional S1 attempts and the
  spent P1 attempt are not a budget. Missing cost is unknown, not zero.

## Before the session

1. Choose a representative, non-sensitive task and a **disposable project or
   explicit safe workspace**. Record what was selected. Do not use production
   data or alter an unrelated working tree.
2. Record current agent/skill versions, installed skill state and the intended
   OpenCode runtime identity. Keep credentials, account details and raw private
   evidence out of Git and out of this document.
3. Decide the adoption path: **observation only** (no config change), **preview**
   (default, read-only), or a separately authorized **apply**. If applying,
   preview and back up first; know the exact rollback command and backup path.
4. Decide whether a real model is used at all. If yes, select the exact
   model/variant, data scope, request/token/deadline limits and stop conditions
   before the session. Prefer no provider access for the first sitting; most
   scenarios below are workflow/evidence scenarios that do not require one.
5. Agree on a timebox and stop conditions: unexpected writes, permission
   prompts, costs, missing evidence, stale identity or unsafe behavior end the
   scenario; record it as **unobserved/blocked** rather than rerunning blindly.

## Integrated cycle

Run the scenarios that fit the chosen workspace; not every case must fit one
sitting. Each row is a separate result. A pass applies only to the route and
setup actually exercised.

| Order | Area | Human action and expected observation | Primary criteria / findings |
| --- | --- | --- | --- |
| 1 | Baseline | Pick a small real task. Record starting files/instructions and the expected result independently of the implementation. | Comparison starting point only; no improvement claim. B8-F2. |
| 2 | Direct fix | Ask for a bounded bug fix in a safe example. Confirm no feature interview or feature docs are created, and the change is focused. | AC-01 routing; no new spec tree. |
| 3 | Spec-only + combined approval | On one small feature, ask for spec/plan/tasks only. Inspect them together, edit if needed, then give or withhold the combined approval yourself. | AC-04/AC-05; approval is recorded only in the prescribed location. |
| 4 | One implementation batch | Authorize exactly one named batch. Observe implementation and ask for one meaningful review; check the changes and evidence before accepting. Do not let green tests alone authorize the next batch. | AC-06/AC-08; B5-F1, B8-F1. |
| 5 | Review-only | Request review of a small existing change. Confirm findings are reported without edits to code, docs, approval or task state. | AC-07/AC-08; no state change. |
| 6 | Interruption and resume | Interrupt a harmless task at a chosen point. Inspect partial work, then request a resume/handoff. Confirm authority is re-read, your files are preserved and no new batch starts implicitly. Test restart/move only if independently safe and explicitly chosen. | AC-06/AC-12; B6-F1. Native recovery remains unproven. |
| 7 | Bug investigation | Provide a failing test or reproduction and ask for diagnosis before a fix. Confirm the failure is reproduced, the cause is distinguished from the symptom, and no unrelated rewrite occurs. | AC-07/AC-13; diagnosis helper branch. |
| 8 | Clarification / grilling | Present one consequential decision with real alternatives. Confirm no more than three independent questions per round, settled answers are reused, and agreement does not start implementation. | AC-02; correctness over question count. |
| 9 | Questionnaire and late answer | Optionally draft a questionnaire for a person who holds missing knowledge. Check recipient/purpose fit; do not send it. If a late answer arrives, confirm only the affected scope returns for decision. | AC-03/AC-04; no sending without separate authorization. |
| 10 | Explanation | Ask "wait, what?" about the current status. Confirm a short explanation of where things are, what exists vs remains and the next decision, with no state change. | AC-07; clarity only. |
| 11 | Teaching | Ask for a focused explanation/example and feedback. If persistent learning files are wanted, confirm a separately chosen workspace; otherwise confirm none are created. | AC-08; learning evidence is not "material covered". |
| 12 | Inactive OpenCode candidate | Generate the candidate and inspect preflight/effective setup warnings. Preview config against a scratch workspace. Confirm no provider/model process starts and no config is written by preview. | AC-10; B3-F1/F4. No real-home write. |
| 13 | Adoption and rollback (optional) | Only with separate authorization: apply the reviewed candidate to a scratch or chosen setup after backup, verify the effective setup, then roll back and confirm unrelated configuration survives. Preserve the original config. | AC-10; B3-F4. A sandbox is not proven by prose. |
| 14 | Headless pending input (optional, separately authorized) | If a real native session is ever authorized, stop on one harmless consequential question. Confirm the result is bounded `blocked`/`pending_human` with the exact question and next action, not a guessed answer. Review the private summary without resuming. | AC-11; B3-F1/F2, B6-F1. Otherwise record **not run — native admission blocked**. |
| 15 | Deterministic evidence | Run a chosen existing project check on the safe workspace. Capture the scoped identity, result, log digest and evidence root separately; build/export a receipt only if the caller-supplied observations are complete. Reload it and confirm changed code/authority or stale identity blocks reuse. | AC-09/AC-12; B6-F2. A receipt is not certified provenance or approval. |
| 16 | Whole-cycle reconciliation | Review request → approval → one batch → review → chosen recovery → final changes and open findings together. Mark incomplete if a stage was skipped, failed or could not be safely observed. | AC-14/AC-15. |

## Recording a result

Use a short entry in the relevant feature task continuation, or share notes for
the assistant to reconcile when authorized:

```text
Scenario and date:
Setup/version and safe workspace:
Expected / observed:
Result: pass | fail | blocked | not run
Evidence location (sanitized):
Finding IDs / next decision:
```

Never copy credentials, raw auth databases, complete environment dumps or
private logs into Git. Sanitized portable receipts may be kept under the
retention policy in [evidence/README.md](../harness/evidence/README.md).

## Rollback and recovery

- **Skills:** preview first; `agentic-sdd apply` backs up replaced skills and
  `agentic-sdd restore` restores a chosen backup. Identical content is a no-op.
  Unrelated skills are preserved; symlinks/special files are refused.
- **OpenCode config:** use the separate digest-checked
  [adoption/rollback procedure](../integrations/opencode/adoption.md). An old
  backup is not authority to discard later edits; reconcile intervening changes
  instead of overwriting them.
- **Sessions:** admitted native execution is currently denied. A pending
  summary is an observation, never consent; reconcile workspace, session,
  authority and changed files before any separately authorized new preparation.
- **Evidence:** freshness failures require remeasurement/reconciliation, not
  reuse of an old receipt. No automatic retry of uncertain external effects.

## Honest completion

Status is **implementation delivered; human usage cycle reported complete and
accepted for the pinned container**, with individual scenario outcomes
unspecified. Completing this plan or any scenario is not required to
call the implementation delivered, and running scenarios does not by itself
certify support, safety, quality or benefit. Support remains **not certified**;
native admission, actual adoption, macOS behavior and all carried findings stay
explicit until separately observed and accepted by the user.
