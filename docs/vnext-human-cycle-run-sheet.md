# Human test record — feature 005

Use with the [human test guide](vnext-human-test-guide.md). A blank row is
**not run**, not a pass. Do not change your normal setup for this cycle.

## Session

| Field | Value |
| --- | --- |
| Date / your name | |
| Safe project (path or description) | |
| Assistant / model used | |
| Timebox | 90 minutes suggested |
| Notes shared with the assistant? | yes / no |

## Results

| # | Task | Result | Notes / evidence |
| --- | --- | --- | --- |
| 0 | Build check (`sh tests/vnext-model-free-checks.sh`, or `--batch --full`) | | Expected `14 passed, 0 failed, 0 skipped`; slim container is `12/0/2` |
| 1 | Direct fix stays direct | | |
| 2 | Spec/plan/tasks only, no implementation | | |
| 3 | One batch, then stop for review | | |
| 4 | Review-only made no edits | | |
| 5 | Interruption and resume preserved state | | |
| 6 | Bug diagnosed before fixing | | |
| 7 | Clarification asked only what was needed | | |
| 8 | Questionnaire drafted, nothing sent | | |
| 9 | Explanation changed nothing | | |
| 10 | Teaching stayed focused | | |
| 11 | Optional: OpenCode starts inside the container | | `opencode --version` / TUI; no provider or network |
| 12 | Optional: config preview wrote nothing | | |
| 13 | Optional: evidence snapshot observed/refused correctly | | |
| 14 | Optional: online login/chat, only if you chose `--online` | | Run `vnext-online-check.sh`; `--hi` is one live prompt |
| 15 | Optional: SDD router inside container uses packaged skills | | Prompt in guide §3; expect spec/plan/tasks, no implementation |

Allowed results: **pass**, **fail**, **blocked**, **not run**. Add one short
note per row: what you asked, what you expected, what happened. Do not paste
credentials, account details or raw private logs.

## Closeout

```text
Sitting/date:
Project and setup:
Passed / failed / blocked / not run:
Most useful or confusing behavior:
Finding IDs or next decision:
Feature status: implementation delivered; human usage validation not run
```

Send the completed rows or a short summary back for reconciliation. This record
does not make the feature Ready, supported or beneficial, and it does not
authorize a model call, install, adoption or release.
