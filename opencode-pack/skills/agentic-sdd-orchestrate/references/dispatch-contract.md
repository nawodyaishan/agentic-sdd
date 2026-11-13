# Dispatch contract

The brief tells a child exactly what to do, where the boundary is, and how its work will be judged. It is written by the main agent and passed as the child's first message. Keep it small: source pointers and identifiers, not a transcript.

## Task brief fields

| Field | Meaning |
|---|---|
| `task_id` | Stable identifier for this one bounded unit, e.g. `T03` or `fix-<slug>`. The child echoes it in its return. |
| `authorized_scope` | The approval the work rests on: feature approval plus batch ID, or the direct-fix request. State explicitly what is **not** authorized (other batches, commits, publishing, live operations). |
| `allowed_paths` | The exact files, directories or packages the child may read or modify. Anything else is out of scope; a child that needs more must return `needs_input`. |
| `expected_verification` | The commands or checks that must run, and the evidence to quote back — actual command and actual result, plus any check the child could not run. |
| `stop_conditions` | Conditions that end the task without continuing: failing pre-existing checks, missing approval, a path outside `allowed_paths`, an unavailable model that would change privacy or cost, an interrupted write with unknown state, or any destructive or publishing step. |

Add only what the child cannot look up for itself: the objective in one or two sentences, the relevant source pointers, the existing convention to follow, and the return format expected. Do not paste the whole conversation, the full spec, or unrelated files.

## Exact packet, not a transcript

When the task came from a generated brief, pass that artifact rather than your own retelling: run ID, task ID, brief path and digest, and the source path and line range the brief was extracted from. The child names the brief it worked from in its return, so the parent can show exactly which task text it saw.

Add the artifact references the child may use — check log paths, review package manifest and patch paths, and the one or two source files that actually matter — with a short accurate excerpt where a pointer alone is not enough.

A generated brief restates an authorized scope; it does not create one. If the brief digest no longer matches the canonical task, or the brief names a different batch, that is a stop condition, not a detail to smooth over. A full transcript or a duplicated log is not a brief: it hides the one requirement the child must follow and makes any claim about what it was told unverifiable.

## Child return schema

A child reports in this shape so the parent can reconcile instead of trusting prose.

```yaml
status: completed | blocked | needs_input | failed | partial
scope_observed: <what was actually touched or read, versus the brief>
changed_paths: [<every created, modified or deleted path>]
findings: [<what was established, with file/line references>]
source_references: [<files, sections, revision/commit identifiers relied on>]
checks_run: [<command and actual result, including failures>]
unverified_claims: [<statements the child did not prove>]
actual_model: <model identity observed, or "unknown">
questions_for_parent: [<only blocking questions, each with why it blocks>]
```

Use `partial` when the work is genuinely usable but incomplete, `blocked` when the child cannot proceed without authority or evidence, and `needs_input` when a specific answer would unblock it. `completed` asserts only that the assigned scope is done and the listed checks passed — not that a review, approval or another batch may proceed.

## Single-writer rule

One writer at a time per checkout. A second writer is allowed only after the first has stopped, its changes have been inspected, and the parent has confirmed the working tree is coherent. Read-only children may run in parallel only when their sources are independent, the runtime supports it, and the human has approved it.

## What the child must not infer from the brief

Assignment wording grants no permission to commit, push, deploy, migrate, rotate credentials, run paid inference beyond what was authorized, spawn further children, start another batch, or answer its own questions about human intent.
