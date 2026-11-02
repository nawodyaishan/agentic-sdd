# Human test record — feature 005

Use with the accepted-container [human test guide](vnext-human-test-guide.md).
A blank row is **not run**, not a pass. Do not change your normal setup; the
accepted environment is the pinned image.

## Session

| Field | Value |
| --- | --- |
| Date / your name | |
| Image id (`docker image inspect agentic-sdd-qa:005 --format '{{.Id}}'`) | expected `sha256:35ed49cd…` |
| OpenCode version / hash | expected `2.2.5` / `8d060e1d…` |
| Workspace (`SDD_QA_WORKSPACE`) | |
| Login/model/provider used | |
| Timebox | 90 minutes suggested |
| Notes shared with the assistant? | yes / no |

## Results

| # | Task | Result | Notes / evidence |
| --- | --- | --- | --- |
| 0 | Image build + `--check` | | Expected image id above and `14 passed, 0 failed, 0 skipped` |
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
| 11 | Online login + `vnext-online-check.sh` | | Expected 3 pass / 0 fail / 1 skip |
| 12 | Optional: one `--hi` model round trip | | Record model id and reply |
| 13 | SDD router inside OpenCode uses packaged skills | | Expect spec/plan/tasks, no implementation |
| 14 | Config preview wrote nothing | | `online_ready: false`, no `.opencode` |
| 15 | Evidence snapshot observed/refused correctly | | `sha256-v1:` diff, 40-hex base |

Allowed results: **pass**, **fail**, **blocked**, **not run**. Add one short
note per row: what you asked, what you expected, what happened. Do not paste
credentials, account details or raw private logs.

## Closeout

```text
Sitting/date:
Image id / OpenCode version / model:
Passed / failed / blocked / not run:
Most useful or confusing behavior:
Finding IDs or next decision:
Feature status: implementation delivered; human usage validation not run
```

Send the completed rows or a short summary back for reconciliation. This record
applies only to this image; it does not make the feature Ready, supported or
beneficial, and it does not authorize a normal-setup install, adoption, release
or any further model call.
