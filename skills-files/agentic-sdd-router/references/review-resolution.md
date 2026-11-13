# Review scope and finding resolution

Use when reviewing an authorized result or deciding what to do with review findings. Review supplies technical evidence, not approval or new execution authority.

## Scope and verdict

| Scope | Required comparison |
| --- | --- |
| Batch | Current batch's authorized outcome, acceptance/contract references, actual diff and relevant check evidence |
| Direct fix | The request, affected behavior, actual diff and focused/regression evidence; no invented feature/task records |
| Feature | Aggregate approved outcome, cross-batch interactions, acceptance coverage, relevant regressions and all consequential open findings |

Select feature scope for an explicit feature-review request or planned feature-wide validation in the final authorized batch, not automatically after every task. Passing batches do not prove the connected feature works. Include rollout/recovery/observability questions only where risk warrants them.

Cover two dimensions in the existing review, not two separate approval gates:

- **Requirements:** missing, extra, misunderstood or unverified behavior.
- **Engineering:** correctness, compatibility, failure handling, meaningful tests and relevant maintainability/operational risk.

Return `Ready`, `Needs changes` or `Blocked`. Label same-agent review **self-review**, not independent review. Unknown evidence stays unknown. A failed integration or unresolved consequential finding cannot become Ready because local checks passed or the agent stopped. `Ready` also requires no unresolved Critical finding and an authorized disposition for every Important one ([severity](#severity)). Ready does not mean merged, published or deployed.

## Severity

Severity is a second dimension of a finding and answers a different question from its disposition: disposition says whether the finding is true, severity says what leaving it open costs. Record both independently — a confirmed finding can be Minor, and a disputed finding can be Critical if it turns out true.

| Severity | Meaning | Consequence |
| --- | --- | --- |
| **Critical** | Security, data-loss or functional failure; the authorized outcome is not delivered. | Blocks acceptance until it is resolved or explicitly addressed by an authorized scope decision recorded in the canonical documents. |
| **Important** | Material requirement, correctness or robustness gap that does not yet break the outcome. | Resolve within the authorized scope before completion, or document an authorized, scoped disposition for accepting the gap. |
| **Minor** | Cosmetic, preference or low-impact hygiene with no hidden requirement or safety consequence. | Report as deferred and non-blocking; it does not expand scope. |

Severity ranks attention; it grants nothing. It never bypasses a spec-mandated check, never upgrades `needs evidence` to a pass, and never adds a second approval gate or an extra mandatory review. Do not lower a severity to reach `Ready`: only evidence, a correction, or an authorized disposition changes a finding's effect.

## Findings and corrections

Give each material finding a stable ID, location, impact, severity and evidence. Use `valid`, `disputed` or `needs evidence` for its disposition, then state its resolution or next action in the existing finding/task context:

- Correct a valid in-scope defect only when fixes are authorized. Retain the finding ID and show the new evidence.
- Check incorrect advice against the actual requirement/code and reject or qualify it with evidence. A confident reviewer is not a source of truth.
- For approved-scope changes or unclear authority, return the decision to the user. Non-blocking suggestions do not expand the batch.
- Keep unresolved findings visible. Preexisting or cross-cutting critical issues can affect readiness even when outside the latest diff; that does not authorize an unsolicited repair.

Re-review addressed findings and changed dependencies, including regressions introduced by the correction. Do not re-review an unchanged result just because routing changed. Use the current run's effort budget; choose a proportional bound when absent (for example, two corrective passes for a small fixture), without creating another approval gate. Exhaustion leaves `Needs changes`/`Blocked` and the next safe action, not parked critical findings, automatic model escalation or a success claim.

## Lifecycle ownership

Implementation owns one completion-review invocation for its current authorized result and any permitted batch-state write. The reviewer returns findings/verdict; the router consumes implementation's result. A correction can justify scoped re-review. An explicitly requested independent review follows the [independent review method](independent-review.md) and still feeds this same owner; it does not add a second completion review. Standalone verification review remains manually invocable and writes no project state. Feature review finding an old-batch problem does not automatically authorize editing that batch.
