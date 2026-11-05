# 006 supervised synthetic pilot — proposed exception, not executable

**Update, 5 November 2026:** The user subsequently authorized this one
exploratory observation and the scoped exception is now recorded in
[spec §Pilot prerequisites](spec.md#pilot-prerequisites-and-retained-findings),
[plan C4](plan.md#c4--proposed-pilot-bounds-and-stop-contract) and
[safety](../../integrations/opencode/safety.md#before-any-separately-authorized-online-test).
The earlier policy-blocked statements below describe the proposal's historical
state. Actual route and fixture availability remain to be established, and the
original acceptance criteria are unchanged.

## Status and scope

On 5 November 2026 the user requested one supervised synthetic-fixture pilot using
`openai/gpt-6-luna-fast#high`, one parent and one inspection-only reviewer, no
nested delegation or automatic retries, and a 20-minute deadline. The user
authorized preparing a narrow policy exception and the live test subject to
higher-priority and external restrictions. This document is a proposal, not an
assertion that those restrictions have been lifted or that B4 has passed.

## Proposed repository-only exception

For this one observation only, replace the C4 requirement to demonstrate an
enforced aggregate request/token ceiling before observation with an explicit
disclosure that **exact aggregate request and token ceilings are neither verified
nor enforced**. Do not infer them from steps or partial usage records. Replace
C4's three-case/600-second proposal with one supervised observation and a
20-minute total deadline. Do not treat this as a relaxation of reviewer
permissions, credential isolation, no-fallback or no-nesting controls. Preserve
all unrelated findings and their status. This proposal does not change
`AGENTS.md`, existing safety policy, the 006 approval, or T05 acceptance.

The exception cannot override the session's higher-priority requirement for
explicit bounded request and token limits, or the requirement for a fresh
credential-free evaluation environment and a user-supplied dedicated scoped
credential for any online fixture. No numeric aggregate request/input/output
limits were supplied for this new model in the current authorization. The old
C4 numbers were proposed for a different pilot and do not establish authorization
or enforcement here. Consequently this proposal is **blocked**, not an active
permission to launch. Any applicable external subscription terms and runtime
policies must also be observed; none has been shown to be an independent ban.

## Proposed procedure if all applicable gates are met

1. Use a disposable synthetic-only fixture, with answer key outside both agents'
   readable scope and an independently recorded baseline. Keep all secrets and
   existing work out. Identify the exact isolated service, scoped secret reference,
   user-owned cleanup and selected subscription route without copying the primary
   credential store. Confirm model and variant selection without inference.
2. Configure at most one parent and one child, no additional user turns or
   re-review. Candidate step limits: parent 6, reviewer 6. The previous 4-step
   parent failed to launch a child; 6 once launched one. Version 2.2.5 has prior
   observed 4- and 6-step parent behavior, but reviewer limit enforcement on the
   proposed route remains unverified. `steps` is not a request/token bound.
3. Keep deny-first inspection-only permissions: reviewer may read only the
   approved fixture and necessary method; deny editing, shell, execution, web,
   integrations, secrets, external directories and any subagent launch. Parent
   may launch only that reviewer. Check effective policy and positive/negative
   permission evaluation, then observe actual dispatch without treating evaluation
   as proof. The existing packaged-reference denial remains unresolved.
4. Before launch verify a real no-automatic-retry mechanism on the pinned binary,
   not merely an instruction. Disable warming and compaction, and account for
   any unavoidable title/summary calls. Current documentation describes a retry
   hook, but its availability and effectiveness on 2.2.5 are not established;
   the existing template denies plugins. Stop if this cannot be reconciled without
   weakening admission or permissions.
5. Record the owned parent and reviewer IDs, deadline and available usage sources.
   At deadline, breach or unexpected activity interrupt both owned sessions, child
   and parent, using the supported per-session route, and inspect active state and
   terminal events. Stream closure or local process exit alone does not establish
   provider cancellation. Current API documentation lists
   `POST /api/session/{sessionID}/interrupt` and active sessions, but actual 2.2.5
   behavior, child discovery and bounded termination have not been verified.
6. Inspect parent and child session messages/logs and available usage records; the
   documented experimental session statistics and usage-limit endpoints are
   candidates, not verified provider accounting. Include auxiliary and in-flight
   work if available; otherwise report unknown, not zero. Report each actual
   observation separately from offline checks and do not claim 006 acceptance.

## Historical paid attempts

The prior OpenCode Go and OpenRouter attempts and their child/auxiliary use remain
unreconciled; see [the runbook](../../integrations/opencode/independent-review.md)
and [safety](../../integrations/opencode/safety.md). They do not, by themselves,
prove a separate financial blocker for a genuinely distinct subscription-only
fixture with no paid fallback or shared allocation. However, the requested route's
credential, billing relationship, subscription allowance and isolation are not
established. A shared allowance or paid route could make the old uncertainty
material; do not assume either separation or a paid-key cap. Repository safety's
provider-reconciliation rule remains in force unless separately changed within
applicable authority. No historical finding is closed by this assessment.

## Evidence collected in this session

- Local binary reports `opencode v2.2.5` and SHA-256
  `8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050`.
- `sh tests/harness/native-independent-review.sh`: 6/6 pass in the pinned,
  network-disabled fixture; permission evaluation only, no model calls or tool
  dispatch. No new runtime test of step enforcement, cancellation or usage occurred.
- Current [agents documentation](https://opencode.ai/v2/docs/agents) describes
  positive `steps` and notes user input resets the allowance. Current
  [API documentation](https://opencode.ai/v2/docs/api) lists interruption,
  session logs, statistics and usage-limit routes. Documentation is not proof of
  behavior in the pinned release or of provider-side accounting.
