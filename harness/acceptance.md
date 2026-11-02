# Feature 004 acceptance and adoption assessment — B8/T11

Scope: R0–R2 in [spec](../specs/004-vnext-harness/spec.md), contracts C1–C6
in [plan](../specs/004-vnext-harness/plan.md). This is a sanitized evidence map,
not approval or a completion ledger. [Tasks continuation](../specs/004-vnext-harness/tasks.md#continuation-record)
owns actual commands, input identities, results and current workflow position.

**Human acceptance approved; technical validation remains Blocked.** The human
decision is recorded only in [spec §Approval](../specs/004-vnext-harness/spec.md#approval).
It closes the review checkpoint without supplying missing evidence or resolving
findings. The B7 review checkpoint was advanced and
B8 authorized, but native/behavioral findings and the benefit requirement were
not waived. No final adoption, inference allocation, publication or live install
follows from B8 authorization or a passing deterministic check.

The user subsequently directed that full CI testing and local containerized
real-AI testing stop in favor of a full plan and hands-on human manual testing.
This is a validation-direction decision, not evidence that manual tests passed
or that the remaining findings are waived; existing CI is not disabled by it.
See [spec §Approval](../specs/004-vnext-harness/spec.md#approval) and tasks for
the decision and current next action.

Permitted offline validation passed: **3 connected tests (8 actual check runs)**,
**6/6 local required checks**, **6/6 isolated reviewed-base checks**, installer
Docker e2e and Go race tests. The connected runs include expected failure,
cancellation and deadline outcomes; their fixture assertions passing does not
turn those executions into successes. See tasks for the exact evidence identities.

Separately authorized [offline S0](../tests/harness/S0.md) was subsequently
delivered: byte-budget/HTTP controls, native **fake-provider** request construction,
nine credential-free compatibility probes on exact 2.2.5 and a separate worker.
This adds no real-model/native-lifecycle or benefit observation. Actual token
accounting and online egress/credential/allocation provenance remain S0-F1/F2;
the full feature verdict is unchanged. S0 was subsequently accepted and exact
bounded S1 conditionally authorized in spec §Approval; its original accounting
conditions remain unmet. A corrected, explicit **P1** exception subsequently
authorized one synthetic text-only observation with input tokens measured only
afterward. P1's 17 offline tests and native fake topology passed, then its single
real upstream attempt returned **HTTP 429** and stopped without retry/fallback.
**1/6 attempts consumed; 5 remain conditional.** Usage/billing and a usable model
response remain unknown. Driver completion is not successful generation: native
exit was 1, gate result `incomplete`. P1-only egress/allocation evidence narrows
S0-F2 but proves neither the original S1 token contract nor native tool/lifecycle
acceptance or useful benefit. The full-feature **technical verdict remains Blocked**,
separate from the subsequent human acceptance decision.

## Connected behavior: what is and is not observed

B8 joins the actual deterministic runner, result persistence, receipt validation,
export/reload and fresh workspace observations. The fixture uses synthetic
project/task sentinels, known independent expectations and a fresh temporary
Git repository. It checks actual process outcomes, not hand-authored passing
result JSON. Signal/deadline paths each have two deterministic repetitions.

The scripted location move is reconciled with independently captured identity
and a new actual check run; it is **not** a native session move or agent resume.
Failed assertions, stopped checks, stale evidence and unresolved findings must
not be promoted to completed/Ready. Receipt consumption must not alter source,
synthetic approval/task sentinels or grant authority for another batch.

These tests **do not demonstrate** an agent drafting a coherent feature,
obeying a human approval, executing one batch, invoking exactly one review,
cancelling native inference or surviving compaction/restart. They neither
drive OpenCode nor load a model. The real development thread is narrow
prospective evidence of human-selected batch progression, not a controlled,
native-correlated baseline/candidate experiment. No saved narrative or fixture
authority string substitutes for real user consent.

## All-criterion evidence reconciliation

“Partial” means some independent deterministic/source evidence exists but the
**whole criterion has not been demonstrated**. No missing native behavior is
turned into a pass by aggregation of earlier batch labels.

| Criterion | Available evidence | Remaining decisive gap / finding |
| --- | --- | --- |
| AC-01 routing/authority | B2 observed explicit-batch and ambiguous-continue comparisons; current canonical approval/task references; B8 receipt rejection of missing/changed caller-supplied authority | Partial: no controlled connected native draft → approval → one batch → review → resume lifecycle; no research-return behavioral case. B8-F1; B6-F1 |
| AC-02 effective config/adoption | B3 credential-free API collision/precedence/invalid-policy probes and synthetic manual backup/no-op/rollback; installer isolated-home tests | Partial: complete live discovery, true tool/canonical-path boundary and supported adoption/rollback remain unverified. B3-F1/F4; live install is not authorized |
| AC-03 mode/execution boundaries | B3 permission/API and plugin/MCP startup controls; B7 worker isolation and anti-weakening fixtures | Partial: permission evaluation is not a real native model-originated tool invocation. Strict native review routes stay disabled; no sandbox guarantee from prose/worktree. B3-F1 |
| AC-04 models/tools/budget | Inactive exact-ID catalog, provider-denied templates, no fallback; explicit safety prerequisites; P1-only exact text route/current official free-price excerpt, offline single-dispatch/egress proof and one upstream 429 | Blocked: original S1 token bounds, full model/tool/session selection and native usage/billing correlation remain unverified. B3-F2/F3; one failed text attempt is not a measured zero session cost |
| AC-05 implementable contracts | Canonical C1–C6 and AC ownership; B2 readiness reconciliation comparison | Partial: observed comparison had no demonstrated improvement; connected native implementation against the shared contracts is missing. B8-F1/F2 |
| AC-06 deterministic verification | B4 runner negative/process cases, B6 freshness, B7 local/CI consistency and weakened-gate detection; hosted main-push runs passed; B8 real result-to-receipt connection; user-supplied push artifact ZIP matched GitHub's digest and its six passing check results/log digests were inspected | Deterministic Linux subset evidenced; whole criterion partial pending connected native completion/freshness and B7-F1 PR-path observation. Push artifact's trusted/candidate revisions are identical; target-event listing returned zero runs. Remote branch-rule administration is excluded, not a new mandatory implementation task |
| AC-07 tests/diagnosis | B5 independent fault oracles and eleven-skill packaging; B8 actual assertion failure, correction and no Ready with open findings | Partial: all eight B5 diagnosis agent cases remain `not_run`, including intermittent uncertainty and read-only logging. B5-F1 |
| AC-08 one review/resolution | Single-owner workflow contract; B5 dependent-regression oracle; B8 green checks cannot waive an unresolved finding | Partial: review-count, disputed advice and correction/re-review behavior of an actual agent still unobserved; deterministic receipt consistency does not prove engineering review. B5-F1; B8-F1 |
| AC-09 traceable results | Versioned check/receipt contract, scoped dirty identities, redaction/unknowns, external owner-only evidence; B8 actual result/export/reload | Partial: real native session/model/task/billing correlation and a trusted native exporter remain unverified. B6-F2; no caller-supplied string certifies provenance |
| AC-10 interruption/resume | Linux owned-process cancellation/deadline and partial-file preservation; B7 owned-CID cleanup; B8 partial receipt reload/location reconciliation | Partial: native cancellation, real compaction/restart/move, resumed authority obedience and duplicate-review avoidance remain `not_run`. B6-F1 |
| AC-11 measured adoption/maintenance | Frozen pre-tuning inputs, retained B2 comparison (no observed benefit), version/rollback and installer compatibility coverage | Blocked: no valid vNext connected comparison/critical native repetitions, no genuinely undisclosed holdout, no useful benefit observed. B8-F2; B3-F2/F3/F4 |
| AC-12 portable package | Eleven skills plus sibling references embedded/on-disk install/preview/no-op/restore; separated native adapter and deterministic contracts | Portable distribution subset evidenced; release support is not certified while native boundary/selection/recovery acceptance is missing. B3/B6 findings apply; Darwin cross-build is not execution evidence |

Sources: [frozen baseline and B2 observations](baseline.md),
[native API evidence/limits](../integrations/opencode/verification.md),
[diagnosis cases](../tests/harness/diagnosis-scenarios.json),
[receipt trust boundary](evidence/README.md),
[native recovery gaps](../integrations/opencode/recovery.md),
[trusted CI](trusted-ci.md), and [B8 fixture scope](../tests/harness/README.md#b8-connected-offline-components-and-feature-acceptance).
Reuse earlier observations only within their recorded scope; they are not new
or current observations of changed runtime/provider environments.

## Baseline, holdout and benefit decision

- Preserve `scenarios.json` and `diagnosis-scenarios.json`. Do not silently change
  their original statuses or prompts to manufacture an outcome; later evidence
  is linked by scenario/finding ID from tasks.
- B1/B2 has actual pre-tuning observations, but a documented spend incident,
  provider switches and failed feature attempts limit what can be compared.
  Detailed incident records stay outside Git; do not replay its credential-copy
  procedure. Failed/tool-schema attempts count as failures, not free successes.
- The retained controlled B2 pairs showed no useful reduction in cost/delay or
  intervention with unchanged quality. They are not a vNext final comparison.
  Do not compare new deterministic suite duration to old model costs, or a new
  model to the old model, and call that an instruction improvement.
- B8 makes **zero additional model requests**. Added inference cost is not a
  measured billing observation; main-session/subscription use and remaining
  allowance are unknown. No new behavioral sample size, latency distribution,
  intervention count or reliability/savings percentage is available.
- `holdout-restore-legacy` is a reserved high-level variant, not a genuinely
  unseen concrete case. Existing format-1 restore tests prove compatibility,
  not an undisclosed evaluation. No new holdout is executed or revealed/tuned
  here; a future evaluator must supply the concrete case within authorization.
- **B8-F2:** comparison/useful benefit remains unobserved. Adoption requires
  unchanged model/variant, input revisions, permissions, tools, environment and
  full failed/retried/auxiliary usage accounting, plus a trustworthy baseline.
  If these cannot be matched safely, report non-comparability, not a pass.

## Integration and rollback decision

Keep shipping portable skills and deterministic tooling separately from the
inactive native template. Continue preview by default and use backup/restore
for installed skill changes; no development test authorizes a real-home apply.
Use [manual native adoption/rollback](../integrations/opencode/adoption.md)
only after a separate reviewed opt-in. Native config/provider changes, known-good
runtime upgrades and required-check administration are not performed by B8.

**B8-F1:** no connected native acceptance observation exists. Before requesting
such execution, prepare one bounded proposal naming the exact model/variant,
synthetic data scope, all auxiliary calls/repetitions, request/token/deadline
limits and stop conditions, with verified nonsecret privacy/price/cap/remaining
allowance controls. The user supplies any dedicated secret through a secret
reference outside chat. Neither “free” nor the previously reported capped-key
creation establishes these prerequisites. Do not enable providers, copy auth
stores, probe alternatives or relax fixture isolation to get acceptance.

The next decision is review of the permitted B8 delivery and retained gaps,
not an automatic release. Green checks plus accepted partial batches do not
satisfy connected acceptance or establish useful adoption benefit.
