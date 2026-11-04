# vNext remaining work and adoption guide

**Status:** decision and planning guide, not authorization to run an experiment,
change configuration, deploy, publish or spend. Updated 2026-11-03. This guide
interprets the [revision 6 adoption plan](Agentic-SDD-vNext-Adoption-Plan.md)
against current feature records; the plan is advisory and its old eleven-skill
baseline is historical.

## 1. Where the project stands

- Feature 005's selected R0–R2 implementation and seven helper capabilities are
  delivered; B1–B5 and the B5 review are complete. The portable package contains
  18 skills. No further implementation batch is selected or authorized.
- The user reported the human cycle complete and working and accepted its results
  for the pinned `agentic-sdd-qa:005` image, using `opencode/big-pickle`.
  The user-provided image check showed 14/14 passes; the connectivity check showed
  3 passes and one optional `hi` skip. Individual workflow scenario outcomes and
  other skips were not specified; do not fill in passes. See the authoritative
  [005 continuation](../specs/005-vnext-completion/tasks.md#continuation-record).
- The result is scoped to the pinned container, **not** native adapter admission,
  normal-home setup, macOS, independent billing or measured benefit. The
  [004 acceptance assessment](../harness/acceptance.md) retains its **Blocked**
  technical verdict; human acceptance does not erase findings.
- The existing adapter has an inactive candidate, config transaction, bounded
  ACP implementation, pending-human normalization and evidence consumer, tested
  with offline/synthetic fixtures. `session run` remains admission-blocked while
  its safety prerequisites are unverified. Delivery is not runtime certification.

**Authority:** [repository guidance](../AGENTS.md), the respective feature's
`spec.md` approval and `tasks.md` continuation govern. Prior session handovers and
this guide do not create approval, waive findings or authorize a batch.

## 2. Choose an outcome before doing work

| Choice | Outcome | Decision or authorization required |
| --- | --- | --- |
| Native validation | Evidence for the exact adapter lifecycle and boundaries | First draft/review a scoped feature or validation proposal; actual online execution has separate exact limits and permissions. |
| Normal-setup adoption | Review and optionally use the installer/config in a named real environment | Separate permission for each real-home apply/config change and rollback scope. |
| Benefit evaluation | Compare workflow outcomes with a credible baseline | Decide comparison design and separately authorize any model-backed work. |
| Operational evidence | Address a specific CI, provenance or security gap | Specify the event, environment, side effects and owner; remote/account actions need separate authority. |
| Publication | Review and publish selected changes | Review exact commits/diffs, dirty work and CI effects; explicitly authorize push/release separately. |
| Stop | Preserve current result and limitations | No further action required. |

Do not start all tracks automatically. Select a coherent outcome; consequential
new work follows the SDD spec → plan → tasks sequence, one combined human approval
and one explicitly authorized batch. A bounded direct fix is possible only where
repository policy permits it. Approval to implement is not permission to run live
tests, install, change accounts, push or release.

## 3. Native adapter validation: proposed path

**Goal:** establish precisely which native OpenCode operations are supported and
safe for the pinned runtime, not merely that the generic TUI works in a container.
Current evidence and finding ownership: [004 acceptance](../harness/acceptance.md),
[005 acceptance table](../specs/005-vnext-completion/tasks.md#acceptance-and-verification-ownership),
[adapter verification](../integrations/opencode/verification.md) and
[recovery](../integrations/opencode/recovery.md).

### 3.1 Prepare without live inference

1. Identify exact implementation revision, binary/version/digest, configuration,
   workspace, candidate and current dirty files. Use synthetic, non-sensitive input.
2. Map each intended claim to retained finding IDs, an independently stated expected
   result, existing evidence and the missing observation. Reuse matching offline
   evidence; do not treat fake-peer success as native behavior.
3. Design a disposable execution boundary: filesystem and process ownership,
   restricted network/egress, clean credential-free baseline, scoped test secret
   only if separately authorized, deadlines, cancellation and cleanup ownership.
4. Review effective configuration and permission/policy paths, including editor,
   shell, MCP, browser and Code Mode where claimed. A policy or instruction alone
   is not an OS sandbox. Current [OpenCode policy documentation](https://opencode.ai/v2/docs/policies/)
   says invalid statements are dropped with a warning; test the exact pinned
   release rather than assuming today's documentation proves its behavior.
5. Identify a real safe rollback and the evidence to retain, with no credentials,
   full environment, private transcripts or hidden reasoning in the repository.

### 3.2 Propose distinct observations

These are **test designs, not permission to execute**:

| Observation | Pass needs | Stop / retain as unknown when |
| --- | --- | --- |
| Effective config and boundaries | Actual effective sources and tested allowed/denied routes; invalid policy and higher-authority behavior accounted for | A path cannot be tested or an unexpected route allows a side effect. |
| Connected lifecycle | Real draft, actual human combined approval, exactly one authorized batch, review and authorized resume; changed code and results independently checked | Scope changes, approval is ambiguous, or evidence cannot correlate the steps. |
| Pending input | A bounded native pending/blocked result with the actual question and no guessed consent; correct safe re-entry | It hangs, auto-answers, loses the question or resumes without authority. |
| Cancellation and recovery | Owned activity cancellation, preserved partial changes, explicit uncertain side effects and validated restart/move/compaction where claimed | Cancellation is unconfirmed, external effects are uncertain, or work resumes from stale authority. |
| Evidence and accounting | Correct session/model/location, code/config/check identity and freshness; separately verified usage/provenance where claimed | Source is caller assertion only, data is stale, or accounting is absent/delayed. |

The current [official ACP documentation](https://opencode.ai/v2/docs/cli/acp/)
describes cancellation and session-directory requirements. It is an input to
the test design, **not** proof of behavior in the pinned version or proof that
external side effects were reversed.

### 3.3 Gate any online observation

Before any agent-operated inference, document and obtain separate explicit user
authorization for the exact provider/model/variant, data, environment, total
requests including auxiliary work/retries, input/output token bounds, deadline,
stop conditions and effect scope. Apply [evaluation safety](../integrations/opencode/safety.md)
and `AGENTS.md`: no primary auth/database copying; use a user-provisioned dedicated
scoped secret reference, not a secret in chat; no fallback, warming or probing.
For paid work verify current price, remaining allowance and a dedicated provider-side
cap before starting. Free pricing, tool capability and private-data eligibility
are different checks; unknown or delayed usage stops work. Do not infer a new
allocation from a historical budget or previously conditional attempts.

The [OpenRouter guardrails documentation](https://openrouter.ai/docs/guides/features/guardrails)
describes scoped budgets and allowlists, while its [ZDR documentation](https://openrouter.ai/docs/guides/features/zdr)
distinguishes inference routing from third-party tools. These are provider-specific
examples, **not** claims about `opencode/big-pickle`, an account, eligibility or
current prices. Verify the actual chosen route and service independently.

## 4. Normal setup and reversible adoption

This is a separate outcome from the accepted container cycle. Decide target OS,
home, exact paths, runtime version and whether the desired action is observation,
preview, isolated scratch exercise or actual adoption. Inspect and follow
[adoption](../integrations/opencode/adoption.md) and [usage](../integrations/opencode/usage.md).

1. Inventory effective configuration and potential conflicts without reading or
   copying secrets. Record existing state and unknowns, not raw credential data.
2. Preview the exact candidate. Specify changed files, backups, unrelated settings,
   verification, rollback and responsibility for intervening edits.
3. Exercise apply/rollback first in a disposable, explicitly approved environment;
   confirm unrelated settings survive and a changed destination is not overwritten.
4. For real-home changes, obtain a distinct explicit approval for exact targets and
   effects. Verify the result; if anything differs, stop and reconcile before retry.
5. Test macOS only on an actual separately chosen Mac/runtime. A cross-build or
   Linux container is not macOS runtime evidence.

Do not activate the intentionally provider-denying candidate merely to make a
model work; an approved configuration is not automatically an approved provider
or data route. Never infer adoption from installer success in a scratch container.

## 5. Evaluate useful benefit, if desired

The [plan §8](Agentic-SDD-vNext-Adoption-Plan.md) proposes a baseline and outcome
comparison. Positive user feedback is useful qualitative evidence, but there is
no established cost, quality or time improvement. For a credible comparison:

1. Agree on a decision, representative feature/bug/review tasks, success criteria,
   meaningful human-effort measurements and a genuinely undisclosed holdout.
2. Freeze input revisions, model/variant, tools, permissions, environment and
   limits; change one harness variable at a time. Record failures, interventions,
   retries, elapsed time and *all* known usage, not just successful output.
3. Predefine acceptable quality and boundary checks, stop criteria, sample size and
   how unknown costs or unmatched baselines will be reported. Do not manufacture
   a percentage from a handful of runs or call non-comparable results an improvement.
4. Obtain separately scoped inference authorization and satisfy §3.3 before any
   model-backed comparison. Without safe comparable evidence, report **unknown**.

## 6. Operational gaps and ownership

- **CI PR path — B7-F1:** the deterministic Linux checks and a main-push artifact
  have evidence; a relevant PR-path observation is still missing. Before a PR or
  CI experiment, choose repository/event/revision and obtain separate permission.
  Preserve trusted checks and review changes to gates. GitHub's [secure-use
  guidance](https://docs.github.com/en/actions/reference/security/secure-use)
  warns against executing untrusted PR code in privileged workflows and recommends
  least-privilege tokens; verify actual workflow and runner configuration.
- **Provenance and billing — B6-F2, S0-F1/F2:** caller-supplied metadata and a
  session's `cost: 0` are not independent provider billing, a budget cap or trusted
  native provenance. Reconcile against separately available authoritative evidence
  only within approved access; treat unknown as unknown.
- **Credential hygiene:** [safety §User-controlled remediation](../integrations/opencode/safety.md#user-controlled-remediation-and-reported-status)
  records user-reported rotation and unresolved decisions about old credential-bearing
  sandboxes and other credentials. The user owns inspection, exact-path cleanup,
  revocation and account changes; do not read/copy stores or delete paths broadly.
- **Publication:** inspect `git status --short --branch`, the exact diff and commit
  range immediately before deciding. Preserve the pre-existing user change in
  `cmd/agentic-sdd/opencode_test.go` and local private handovers. Agree on staged
  scope and whether push-triggered CI is acceptable; commits, push and release tags
  require their own applicable authorization. Do not rely on an old ahead count.

## 7. Optional R3/R4 — not feature 005 work

Independent reviewers/workers, multi-worktree orchestration, advisory AI CI,
plugins, warming/routing/cache optimization and additional runtime adapters are
optional proposals. Choose one only for a demonstrated need, with its own scope,
isolation, cost/failure tests and approval. No subagent, model switch, plugin or
automation is authorized by this guide.

## 8. Review and evidence template

Use this in an existing feature document or review response, not a new mandatory
ledger. Keep finding IDs in their canonical locations.

```text
Proposed outcome and exclusions:
Finding IDs and evidence already available:
Exact revision / runtime / workspace / data scope:
Expected behavior and independently checkable evidence:
Permissions, side effects, owner and rollback:
Model/provider, request/token/deadline/spend bounds (if applicable):
Stop conditions and unknowns:
Human decision requested; separately authorized execution, if any:
Result: pass | fail | blocked | not run; evidence pointer:
Next permitted action:
```

Acceptance may be scoped or partial. Preserve technical verdicts, missing evidence
and retained findings; do not equate acceptance with merge, adoption or support.

## 9. Research and limits

Public primary sources were discovered using Exa on 2026-11-03; OpenCode V2
documentation was also consulted through Context7. Relevant sources are linked
in context above and include [permissions](https://opencode.ai/v2/docs/permissions),
[configuration](https://opencode.ai/v2/docs/config/), and
[GitHub's PR security guidance](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target).
Moving public documentation does not certify the installed 2.2.5 runtime; pin and
verify the version used in any later proposal. No account, credentials, provider
endpoint, inference request, CI experiment or live installation was accessed for
this guide.
