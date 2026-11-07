# Failure and reconciliation

A child's report is evidence to check, not a result to accept. This reference covers the three failure shapes that matter most: a failed or unavailable model, an interrupted write, and a claim the parent cannot verify.

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

1. Inspect the child session's recorded tool results to see which writes completed.
2. Inspect the working tree with `git status --short` and `git diff` — read the actual additions and deletions, not the summary.
3. Preserve completed work; resume the same child session when the client supports safe continuation.
4. Never replay a whole write because part of it is missing; patch only the missing section.
5. If the state is ambiguous, stop and return `blocked` with the paths and the uncertainty. Duplicated writes are worse than a stopped task.

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
