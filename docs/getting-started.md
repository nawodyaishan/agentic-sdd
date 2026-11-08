# Getting started with the supported core

One workflow for learners and experienced engineers: choose the right scope,
verify the result, and keep the human stop points. These fictional examples are
not model-compliance evidence. Their approval lines illustrate decisions; they
do not authorize work in this checkout.

## First five minutes: preview, then stop

With the CLI already installed, run `agentic-sdd version` and
`agentic-sdd preview`. Preview reads the selected installation but does not
install skills or create an apply log. Review the destinations and proposed changes.
See the [support matrix](../README.md#support-and-release-scope) for platforms,
published artifacts and exclusions; installation is not cross-client certification.

On an interactive terminal, `agentic-sdd guide` walks through the same
preview-first flow step by step with numbered prompts; see
[guided CLI journeys](guide-flows.md) for its screens, journeys, and optional
flow trace.

For a source checkout, with **Go 1.23+ already available**, try a disposable home:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go version
demo_home=$(mktemp -d)
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  go run ./cmd/agentic-sdd preview --home "$demo_home"
rmdir "$demo_home"  # succeeds only if preview left the new home empty
```

Run this from the repository root. Go may compile and write its build cache;
the preview must leave the selected home unchanged. Missing prerequisites stop
the exercise rather than triggering installation. Run only a checkout you trust:
temporary homes and Go environment settings are not a security sandbox.
Task is not needed for this
preview; [contributor checks](../CONTRIBUTING.md) use Task and optional Python.

**Stop here:** an empty test home predicts fresh installs, not changes to your real
installation. Real-home `apply` is an explicit later choice after preview, not part
of this exercise. Editing `skills-files/` does not update already installed skills.
Do not enable native runtime experiments to use the portable workflow.

## Direct fix: no feature paperwork

Suppose an existing formatter repeats a label and its expected output is clear:

> Fix the repeated label in the status formatter. Keep flags and filesystem
> behavior unchanged. Add/run the focused regression check and report the result.

Expected path: inspect the formatter and existing test, make the bounded fix,
verify it, then summarize changed behavior, actual checks, limitations and the next
allowed action. **No `spec.md`, `plan.md`, `tasks.md`, batch ID or batch-state write.**
If the change instead affects permissions, data integrity or a public contract,
stop and use feature planning rather than stretching the direct-fix route.
See the [direct-fix contract](../skills-files/agentic-sdd-router/references/workflow-policy.md#direct-fixes).

## Feature: one combined approval and one batch

Suppose a project needs JSON output for an existing read-only report command:

> Draft the spec, plan and tasks for JSON report output. Inspect the repository
> first, reuse settled answers, and stop for one combined review. Do not implement.

Keep the three documents short but connected, for example:

| Document | Illustrative contents |
|---|---|
| `spec.md` | Add JSON output; preserve text output; exclude writes and runtime integration; acceptance covers valid/invalid selection and unchanged default output. |
| `plan.md` | Reuse the report data; define the output contract and error handling once; name actual guidance/tools and verification. |
| `tasks.md` | B1: output selection, serialization and focused checks; explicit completion conditions and stop for review. |

For an incomplete request, the agent first separates repository facts from
assumptions, proposes scope/exclusions/acceptance conditions, and asks only ready
consequential questions—at most three per round. A clear request needs no interview.

After actually reviewing the connected documents, the human can say:

> I approve these spec/plan/tasks revisions. Implement B1 only under that scope,
> verify it, and stop for my review. No real-home apply, publishing or model spend.

The agent records the combined approval **once** in the project's established
approval location. It implements only B1, performs one completion review and records
`awaiting human review` in tasks. Approval of the documents without a selected,
authorized batch is not permission to choose and start one.
See [approval](../skills-files/agentic-sdd-router/references/workflow-policy.md#drafting-sequence-and-one-combined-approval)
and [batch/resume rules](../skills-files/agentic-sdd-router/references/workflow-policy.md#execution-batches-and-resume).

## Review and resume: acceptance is not continuation

At the B1 checkpoint, a findings-only request stays read-only:

> Review the B1 diff and evidence against its acceptance criteria. Report findings
> only; do not edit source, documentation, approvals or task state.

“Ready” is a technical verdict, not approval. A human's “I accept B1” accepts that
delivery; it does not authorize B2. Bare “continue”, green tests or a stale handover
also do not create missing authority. The agent checks current files and the owning
approval/continuation, reports the next permitted action and waits when necessary.
Acceptance with open findings keeps those findings visible.

When the human wants the already-approved next batch, a clear instruction is valid:

> I authorize B2 only under the approved scope. Verify it and stop for my review.

The agent reconciles that instruction with current state; it must not reject clear
unchanged authority merely because an old checkpoint exists. It does not start B3.
Research, diagnosis and a requested explanation preserve the invoking mode's
restrictions; they do not turn a review into a fix.
See [decision meaning](../skills-files/agentic-sdd-router/references/human-input.md#decision-meaning).

## Optional learning and useful reports

**Teaching is optional**, for example: “Explain this change with one example;
do not create learning files.” The same engineering workflow still applies; a
checkout does not become a course. See [teaching](../skills-files/agentic-sdd-teach/SKILL.md).

For bugs, contribution checks and safe handling of sensitive reports, use
[CONTRIBUTING](../CONTRIBUTING.md#reporting-bugs-and-security-concerns).
Do not paste credentials, private backups or raw session traces into public issues.
