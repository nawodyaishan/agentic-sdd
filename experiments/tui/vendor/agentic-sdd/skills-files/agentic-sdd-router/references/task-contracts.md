# Task and readiness contracts

Use this reference when drafting a plan/tasks set or reconciling it for combined review. It owns task-entry and readiness mechanics; requirement ownership stays in `spec.md`, design contracts in `plan.md` or the repository's canonical API/schema source, and execution state in `tasks.md`.

## Task entries

Keep the existing objective, acceptance result, dependencies, status and checks. Add only what the task needs:

| Addition | Rule |
|---|---|
| Acceptance reference | Identify the requirement(s) covered. Use stable labels when several tasks refer to them; headings or bullets are enough for a tiny feature. |
| Contract reference | Link the one authoritative shared definition; do not copy it into multiple tasks. |
| Consumes / produces | Include for a real inter-task interface, not every single-file task. State values, errors and units when they are not obvious. |
| Verification scenario | State input or condition, expected behavior and the applicable check. Derive the expected result independently of the implementation under test. |
| Integration responsibility | Identify which task or batch proves the connected behavior. |
| Target files | Discover them from the repository before finalizing; do not guess brittle paths or source lines. |

A task is ready when a competent builder can proceed without consequential guessing. This question is a check, not another document or approval checkpoint.

## Shared boundary contracts

Where meaningful work crosses a boundary, define the contract once in `plan.md` or the repository's existing canonical API/schema source. Include only what consumers need:

- inputs and outputs;
- relevant error semantics;
- the invariant that must remain true;
- state transition or side effect;
- important units, versions or concurrency assumptions.

Explain briefly why the selected approach fits the requirements. Keep alternatives proportional. Do not turn every task pair into a contract table.

## One readiness reconciliation

`agentic-sdd-tasks` performs one reconciliation after the spec, plan and tasks drafts exist:

1. Every acceptance criterion has an implementation owner and a suitable verification owner.
2. Shared producers and consumers agree on types, values, errors and units.
3. Dependencies and shared-file changes have a workable order.
4. Tests check required behavior and relevant failure cases.
5. Material assumptions are resolved or surfaced for the combined human review.

Inspect actual dependency edges; do not import a table for every possible pair. Record only useful gaps and corrections. This reconciliation supports the one combined approval; it does not create another approval flag or require architecture review for routine work.

At implementation start, refresh affected assumptions against current code. Do not rerun the whole planning process when nothing relevant changed.
