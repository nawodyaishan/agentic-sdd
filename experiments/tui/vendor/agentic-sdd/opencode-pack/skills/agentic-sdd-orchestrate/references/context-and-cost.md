# Context and cost

Delegation earns its cost only when the child's context is smaller and better targeted than doing the work inline, and when the parent can still verify the result. This reference defines the classification that decides whether to delegate at all, and how to keep context small without hiding evidence.

## Delegation classes

| Class | Situation | Default action |
|---|---|---|
| `D0` direct | A single read, a short search, a known answer, one small understood fix, or the human asked for no delegation. | Main agent uses its own tools. No child. |
| `D1` cheap read-only | Several searches or file reads would consume substantial main context, and the child needs no write access. | Delegate one bounded read-only question to a free or lightweight profile. |
| `D2` authorized writer | An approved batch or direct fix genuinely needs isolated implementation, tests or documentation work. | Delegate to a writer profile with an explicit brief and allowed paths; one writer at a time. |
| `D3` advisor / research / vision | A decision needs architecture advice, external documentation research, or reading an image the main model cannot see. | Delegate to the advisor, librarian or visual profile; keep the external-query rules. |
| `D4` critical-main | The work changes approvals, permissions, destructive operations, releases, credentials, or anything the human must decide. | Keep it in the main session. Delegate only the read-only evidence gathering. |
| `blocked` | Approval, scope, path isolation, data policy, network or cost authority, or current state is missing or unsafe. | Do not dispatch. Ask one material question through the parent, or stop. |

Classification is the main agent's decision, and it must be stated in the report. A child cannot reclassify its own task upward to gain permissions.

## Passing minimal context

- Point at sources: file paths with line ranges, spec section numbers, batch IDs, commit hashes. Do not paste files the child can read.
- State the goal, the boundary, the expected evidence and the stop conditions — nothing else is required.
- Keep full private evidence (raw logs, long diffs, complete test output) in the main session and pass accurate excerpts or a pointer to where it lives.
- Pass the *question*, not the reasoning that produced it, for read-only discovery. The child's independence is worth more than your hypothesis.
- Never pass secrets, credentials, tokens or unrelated private data to any child, and never include them in an external search query.

## Separating main-session usage from subagent usage

Report cost and context honestly and separately:

- What the **main session** consumed is not the same as what a **child** consumed. Do not present a child's small context as if the whole task were cheap, and do not hide the main session's usage.
- Name the model that actually ran for each side when it is observable; "the configured model" is not evidence.
- Free models are free to the wallet, not free in rate limits, latency or privacy terms. Paid lanes may draw on an allowance or incur charges.
- A provider's catalogue listing, a configured model and a successful call are three different claims. Only the last one is evidence of availability.

## Never claim unmeasured savings

Do not state or imply a measured token, cost or subscription saving unless the run actually measured it and the comparison is stated (same task, same code state, same checks). Otherwise describe the mechanism — smaller briefs, fewer redundant calls, bounded read-only children — and say the saving is unmeasured. Avoid percentage claims, "N× faster" claims and "zero cost" claims that were not observed.

## Prefer the cheaper correct action

When the same evidence is available inline, use the main agent. When one targeted question unblocks a decision, delegate once rather than exploring broadly. When a child's answer is insufficient, refine the brief instead of starting several children in parallel. Additional agents are a cost, not a quality signal.
