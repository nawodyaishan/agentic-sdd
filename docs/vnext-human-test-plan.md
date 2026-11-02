# vNext human hands-on test plan

**Status: draft for discussion; not executed.** This plan covers the full vision
in [Agentic SDD vNext](Agentic-SDD-vNext-Adoption-Plan.md), including the initial
R0–R2 release, future R3/R4 extensions and Revision 6 human-input proposals.
It distinguishes delivered work from future design; it does not authorize a new
batch, model request, provider spend, installation, deployment or publication.

## Direction and roles

The user directed that full CI testing and local containerized real-AI testing
stop: prepare the full plan and test by human hands-on review instead. The user
will operate their **normal setup** and decide when to perform each test. The
assistant may help explain steps and inspect evidence the user chooses to share;
it must not run the tests, access credentials or change the live setup without
separate authorization. No existing CI workflow is disabled by this decision.

Human acceptance of feature 004 is recorded in
[spec §Approval](../specs/004-vnext-harness/spec.md#approval). This plan does
not reverse it, waive findings or imply successful manual tests. Current status,
evidence and finding IDs are in [tasks](../specs/004-vnext-harness/tasks.md#continuation-record)
and [acceptance](../harness/acceptance.md). Any future change in scope requires
the applicable feature/batch approval, not just this draft.

## Before any hands-on session

1. Choose a representative, non-sensitive task and a disposable project or
   explicit safe workspace in the normal setup. Record what was selected; do not
   use production data or alter an unrelated working tree.
2. Inspect current skill/configuration versions and any uncommitted work. Keep
   credentials and account details private. Decide explicitly whether a change
   to the normal setup is needed; preview and back up before any separately
   authorized installation. The default is observation only.
3. If the human chooses to use a real model, decide the exact model, data scope,
   request/token/deadline limits and spending controls before the session.
   Existing conditional allocations and P1's spent attempt do not authorize a
   new run. Treat missing costs as unknown; do not copy primary auth stores,
   silently switch providers or perform automatic retries. Strict pre-send
   guarantees remain unresolved for the previously proposed automated pilot;
   an independently human-operated session needs its own explicit safety and
   scope decision, not an assumed exemption.
4. Agree on a short session timebox and stop on unexpected writes, permissions,
   costs, missing evidence or unsafe behavior. Record an **unobserved** result
   rather than repeatedly rerunning a case. The user chooses whether to proceed.

## Hands-on test sequence

These are proposed human scenarios, not a mandatory one-sitting checklist. For
each, record the task, actual setup/version, action, expected and observed result,
relevant evidence, issues, and **pass / fail / blocked / not run**. A pass applies
only to the route and setup actually exercised; no small sample proves reliability.

| Order | Area and current state | Human action and expected observation | Scope and finding |
| --- | --- | --- | --- |
| 1 | **R0 baseline — delivered** | Choose a small real task; note starting files, instructions and current workflow. Verify the task can be described and its expected result is independent of the implementation. | Establish a comparison starting point; no claim of improvement. |
| 2 | **R1 portable workflow — skills delivered** | Ask for a direct fix, a spec-only request and a review-only request in separate safe examples. Confirm the assistant chooses the requested mode, does not create unnecessary feature docs and does not edit in review-only mode. | Evidence for routing/authority (AC-01); separate scenarios, no assumed blanket compliance. |
| 3 | **R1 draft and approval — delivered guidance** | On one small feature, inspect spec, plan and tasks together. Confirm the requirements, design, checks and one-batch scope are understandable. Give or withhold the combined approval yourself. | Human decision is recorded in the prescribed location; no implementation before approval. |
| 4 | **R1 runtime and boundaries — template/fixtures only** | Review the actual effective instructions, selected model and permissions on the intended setup. In a safe test workspace, explicitly choose a harmless denied action and check whether the tested route blocks it. | B3-F1–F4; individual routes only. Instructions and a worktree are not an OS sandbox. Never test destructive or credential-access actions on real data. |
| 5 | **R2 one batch and review — guidance/fixtures delivered** | Authorize one named batch, observe implementation and request a single meaningful review. Check results against the acceptance criteria and inspect changes before deciding what to accept. | B5-F1/B8-F1; passing tests do not settle unresolved review findings or authorize another batch. |
| 6 | **R2 deterministic checks and evidence — delivered** | Inspect or, if you choose and authorize an appropriate safe context, run relevant existing project checks. Review actual results and evidence, including failures and missing prerequisites. | AC-06/09; prior hosted and offline checks remain historical scoped evidence. Do not initiate a full CI run or PR experiment under this plan. |
| 7 | **R2 interrupt and resume — offline fixtures only** | For a deliberately harmless test task, interrupt at a user-chosen point. Inspect partial work and request a resume; verify the assistant re-reads authority, preserves your work and does not start an unapproved batch. Test compaction/restart/move only when independently safe and explicitly selected. | B6-F1/B8-F1; no external write is retried until its outcome is reconciled. Native recovery is not yet proven. |
| 8 | **R2 integrated result — not proven** | Review the whole small feature from request through approval, one batch, review and chosen recovery scenario. Check the final changes and outstanding findings together. | B8-F1; mark incomplete if a stage was skipped, failed or could not be safely observed. |
| 9 | **Benefit — not demonstrated** | If you want a comparison, choose an appropriate baseline and candidate before testing. Keep model, inputs, tools and environment fixed; note failures, elapsed time and your interventions for both. | B8-F2; avoid claiming savings or reliability from unmatched setups or a small sample. A genuinely unseen holdout remains a separate decision. |
| 10 | **R3 extensions — future, not delivered** | Decide whether a specific need justifies an advisory agent, optional reviewer, worktree pilot or plugin. First review a separate proposal, boundaries and rollback; manually exercise only what is later approved and built. | No delegation, plugin or extension is enabled by this plan. |
| 11 | **R4 optimization/portability — future, not delivered** | Decide which measured bottleneck or additional adapter merits a separate proposal. Compare only after baseline behavior is understood and keep failure modes visible. | No automatic model routing, concurrency or second-adapter claim. |
| 12 | **Revision 6 human-input proposals — advisory, not implemented** | Review proposed clarification, questionnaire, handoff, explanation and learning flows in §3a/A8 of the vNext plan. Choose which, if any, deserve a separately specified implementation and human scenario. | No new helpers, teaching workspace or behavioral claim is implied. |

## How to capture a manual result

Use a short entry in the appropriate existing feature task continuation, or share
notes for the assistant to reconcile when authorized:

```text
Scenario and date:
Setup/version and safe workspace:
Expected / observed:
Result: pass | fail | blocked | not run
Evidence location (sanitized):
Finding IDs / next decision:
```

Do not include keys, tokens, account records, private prompts, hidden reasoning or
raw sensitive traces. A screenshot or narrative is an observation, not proof of
untested execution boundaries. Use the existing spec for human approval and tasks
for continuation; a test note cannot grant authorization.

## Decisions before starting

- Which small task and workspace will the user use for the **first** manual
  session, and which of scenarios 1–12 should it cover?
- Does the user want **planning only** for now, or to personally conduct a
  selected session? A real-home change, paid/model experiment or external write
  still needs its own clear scope and limits.
- If a problem appears, should the assistant explain it, draft a fix, or request
  a separately scoped implementation decision? Do not silently expand the test.

**Next:** review this draft with the user, pick a first small session and identify
its prerequisites. Until then, preserve existing work, the spent P1 allocation,
all open findings and the stop on full CI/local real-AI container testing.
