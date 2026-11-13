# Failure and reconciliation

A child's report is evidence to check, not a result to accept. This reference covers the five failure categories, manual model fallback, bounded correction, checkpoint and stale state, interrupted writes, and a claim the parent cannot verify.

## Failure categories

Classify the failure before reacting: the category decides who may act.

| Category | Looks like | First move |
| --- | --- | --- |
| **Provider/transport** | Unavailable model, quota, repeated transient errors, a dropped stream. | Read the real error, then the manual fallback rules below. |
| **Model quality or wrong implementation** | Plausible output that does not implement the brief, invents requirements, or edits the wrong files. | Compare against the brief and the actual diff; stop and re-scope instead of adding more prose. |
| **Test or command failure** | The child's command or the repository's check genuinely fails. | Capture command, exit code and full log; decide whether the cause is inside the authorized scope. |
| **Partial write or unknown side effects** | Interrupted write, half-applied edit, unclear working-tree state. | Inspect the actual tree before any further write; reconcile or return `blocked`. |
| **Permission, billing or data-policy denial** | Denied tool or path, spend limit, private-source or network restriction. | Stop. This is an authorization question, not a model problem; never reroute it. |

A failure in one category is not evidence for another. A failing test is not a reason to switch models, and a denied permission is not a transport error.

## Bounded correction

Within the authorized batch, the main may diagnose and make a focused correction. Keep the existing limited correction passes, then stop for a human decision.

- Correct the named defect, then re-run the check that exposed it plus a regression the change could touch.
- Never silently re-run a partly successful edit, and never replay a write whose state is unknown.
- Never escalate to a more expensive model, a different provider, a fresh reviewer or another task or batch because code or tests failed. A human may authorize another agent, model or task; that request still passes the existing privacy, cost and role gates.
- Exhausted passes leave the finding open as `Needs changes` or `blocked`, never as a success claim.

## Checkpoints, stale state and interruption

`agentic-sdd run status` reports observed state, not progress toward approval. Read it before continuing after compaction, restart or interruption.

- States are observations: `prepared`, `running`, `observed-partial`, `observed-failed`, `awaiting-verification`, `verified-evidence`, `stale`, `blocked`. None is human approval, and even `verified-evidence` is not acceptance.
- A `stale` or `blocked` run stays stopped until the changed source, batch or worktree is reconciled. The reported next permitted investigation names an investigation, never a resume.
- A `run record` entry is a **claim**. A `passed` claim yields `awaiting-verification`; only a fresh check in the current code state, or the parent's own inspection, supports a verified observation.
- Never resume a writer, command, reviewer or the next batch because a checkpoint says the work was nearly done. Reconcile, re-observe, then decide with the human.

## Model failure and manual fallback

Fallback is **manual by default**. A written model ladder in `opencode-pack/routing.md` documents permitted substitutes; it is not an automatic failover engine, and no child may switch provider or spend more on its own.

When a child fails for model unavailability, quota or repeated transient provider errors:

1. Read the child's actual output and error text; do not guess the cause.
2. Try each documented fallback at most once, in order, and only within the lane rules in `routing.md` (Free Console, Go, or 9Router/Claude).
3. Check compatibility before switching: context size, output ceiling, and input modality. A smaller output ceiling can truncate a write; a text-only model cannot review an image.
4. Treat permission denials, invalid requests and missing tools as **task errors**, not model failures. Never retry them on another model.
5. Report the model actually used and the reason for the switch. A specification, plan or review produced by a fallback stays provisional where the governing policy says so.

Never expose credentials in a retry, and never route a paid lane through an endpoint that was not already configured for it.

## Interrupted writes

Before retrying any write task that may have been interrupted:

1. Read `agentic-sdd run status` when a run exists; it reports the checkpoint, freshness and blockers but replaces neither this inspection nor a human decision.
2. Inspect the child session's recorded tool results to see which writes completed.
3. Inspect the working tree with `git status --short` and `git diff` — read the actual additions and deletions, not the summary.
4. Preserve completed work; resume the same child session when the client supports safe continuation.
5. Never replay a whole write because part of it is missing; patch only the missing section.
6. If the state is ambiguous, stop and return `blocked` with the paths and the uncertainty. Duplicated writes are worse than a stopped task.

## Rejecting unverified claims

For every claim in `findings` or "checks passed":

- Confirm the file or line it cites actually contains what was claimed.
- Re-run or read the reported command output; a quoted command without its result is not evidence.
- Treat a claim about behavior as unproven unless a check exercised it in this code state.
- Treat "all tests pass" without the command, package selection and result as unknown.
- Compare scope: files outside `allowed_paths`, a different batch, or extra edits are deviations even when they look helpful.

Record what remains unverified rather than upgrading it to verified.

## One completion review

When the batch is done, run a **single** completion review through `agentic-sdd-implement`'s completion step, or `agentic-sdd-verification-review` when an independent review was explicitly requested. Do not chain reviewer after reviewer, and do not use a review failure as a reason to silently start another implementation pass. A second review is a new, explicitly authorized decision.

## Reporting a failure

Report, in this order: what failed, the exact error or observable state, which paths were affected, what was preserved, what is uncertain, and the one next action the human can authorize. Do not bury a failed check under a summary of successful ones.
