---
title: First feature tutorial
description: Take one practical feature from spec, plan, and tasks through one combined approval, one implementation batch, review, and explicit resume.
---

Use the **feature workflow** when a change affects public contracts, data integrity, permissions, migrations, or multiple coordinated components. You review `spec.md`, `plan.md`, and `tasks.md` together once, authorize one batch at a time, and inspect real check results at each stop boundary.

> **Scope and verification notice:** Commands using `asdx-core` run locally against a disposable `--home` directory with no real-home writes. Any step that sends prompts to an AI coding client requires your own configured client and provider/model connection; those steps are **optional and unverified here**. Illustrative AI responses and draft excerpts below are examples only—they never grant approval or verify model compliance. Agentic SDD skills are workflow guidance, not an OS security sandbox.

## Example scenario

Suppose a project has a read-only `report` command that prints plain text. You want to add `--format json` output while keeping text output unchanged and excluding any file writes or live network calls. Because `--format json` defines a public output contract and error semantics across multiple files, it belongs in a feature folder (`specs/001-json-report/`) rather than a [Direct fix](./direct-fix.md).

## Step 1: Verify isolated CLI prerequisites

Confirm your local `asdx-core` installation works without touching your real home directory. (Skip this if you already verified your setup in [First five minutes](../getting-started/first-five-minutes.md).)

```sh
asdx-core version
demo_home=$(mktemp -d)
asdx-core preview --home "$demo_home"
```

- **Prerequisites:** `asdx-core` installed (see [Installation](../getting-started/installation.md)). No provider credentials or network access are required for this CLI check.
- **Effects:** Read-only. Prints version metadata and previews installing the 18 core skills into `"$demo_home"` without writing files.
- **Expected result:** Both commands exit `0`; `"$demo_home"` remains empty.
- **Cleanup:** Remove the temporary directory:

  ```sh
  rmdir "$demo_home"
  ```

## Step 2: Draft `spec.md`, `plan.md`, and `tasks.md` (Stop 1: before implementation)

In your AI coding client, ask `agentic-sdd-router` to inspect the repository and draft the three connected documents in order (`agentic-sdd-spec` → `agentic-sdd-plan` → `agentic-sdd-tasks`), then stop before editing production code:

> Use `agentic-sdd-router`. Inspect this repository, ask at most the consequential questions you can't resolve from the code, and draft `spec.md`, `plan.md`, and `tasks.md` for adding `--format json` to the `report` command. Present them for my combined approval and stop before implementing.

**Asserted workflow behavior (`skills-files/agentic-sdd-router/SKILL.md` and `workflow-policy.md`):**

- `agentic-sdd-spec` defines the outcome, scope, explicit exclusions, observable acceptance criteria, and a single approval section initialized to `draft/pending`—never a pre-accepted approval.
- `agentic-sdd-plan` consumes the draft spec directly (no separate spec approval needed), defines the shared JSON/error contract once, names risks, and records specialist/tool selections once (for example, `"no additional specialist needed"` with a one-line reason when repository conventions suffice).
- `agentic-sdd-tasks` orders tasks into explicit execution batches (here, **B1** for JSON formatting/schema tests and **B2** for CLI flag wiring and integration checks), runs one readiness reconciliation, and **stops** for combined human approval.

If an external fact (for example a renamed option in a pinned dependency) blocks the draft, pause and resolve that one question with the [Research and debugging tutorial](./research-and-debugging.md) before inventing scope.

- **Prerequisites:** An AI coding client with Agentic SDD core skills installed and a working provider/model connection (**optional and unverified here**).
- **Effects:** Creates `specs/001-json-report/spec.md`, `plan.md`, and `tasks.md` in the sample project. Does **not** modify application code or tests.
- **Expected result (illustrative draft summary and AI response — not executed output):**

  | Draft file | Illustrative contents |
  |---|---|
  | `specs/001-json-report/spec.md` | Outcome, exclusions (no writes/network), acceptance labels (`AC-1` JSON output, `AC-2` unchanged text default, `AC-3` invalid `--format` error), and `## Approval` set to `draft/pending`. |
  | `specs/001-json-report/plan.md` | Shared output contract, exit-code rules, affected modules, specialist assignment (`no additional specialist needed`), and focused test gates. |
  | `specs/001-json-report/tasks.md` | Ordered tasks grouped into **B1** (JSON serializer + unit checks) and **B2** (CLI flag wiring + CLI integration checks), both `pending` behind combined approval. |

  ```text
  Action: Feature drafting (specs/001-json-report/)
  Stage: Combined approval checkpoint (drafts only; no implementation started)
  Files created:
  - specs/001-json-report/spec.md (r1) — Approval: draft/pending
  - specs/001-json-report/plan.md (r1) — Specialist: none needed (standard Go CLI patterns)
  - specs/001-json-report/tasks.md (r1) — Batches: B1 (T01-T02), B2 (T03-T04)

  Blocking questions: None (or at most 3 consequential questions if a contract detail was unresolved).
  Next decision needed: Combined human approval of spec.md r1, plan.md r1, and tasks.md r1,
  plus explicit authorization if you want to start Batch B1.
  ```

- **Cleanup:** If practicing on a scratch repo, you can delete the draft folder with `rm -r specs/001-json-report` after inspecting it.

## Step 3: Record one combined approval and authorize Batch B1 only

Read `spec.md`, `plan.md`, and `tasks.md` together. Only after you actually review and accept them, give **one combined approval** and explicitly name the first batch to execute:

> I approve `specs/001-json-report/spec.md` r1, `plan.md` r1, and `tasks.md` r1. Implement Batch B1 only under that scope, verify it, and stop for my review. Do not start B2, apply to a real home, or publish.

**Asserted workflow rules (`skills-files/agentic-sdd-router/references/human-input.md`):**

- Combined document approval and batch authorization are distinct decisions (though one message can state both clearly). Approving the three documents alone does **not** authorize the agent to pick and start B1.
- Approval is recorded **once** in `spec.md` (`## Approval`), tied to the reviewed revisions and human actor. Never fabricate an approver, timestamp, or consent in template text:

  ```markdown
  ## Approval

  - **Decision:** [template placeholder — pending actual human review]
  - **Reviewed scope and revisions:** `spec.md` r1, `plan.md` r1, `tasks.md` r1
  - **Actor:** `<replace with actual human reviewer>`
  - **Evidence:** `<replace with link/note of actual human approval>`
  ```

- **Prerequisites:** Reviewed drafts from Step 2 and an active client session (**optional and unverified here**).
- **Effects:** Updates the single approval record in `spec.md` to reflect your explicit decision and authorizes execution of **B1 only**.
- **Expected result:** The agent confirms the recorded approval reference and begins only Batch B1.
- **Cleanup:** None if continuing to Step 4; otherwise revert the sample folder.

## Step 4: Execute Batch B1 and stop at `awaiting human review` (Stop 2)

During B1, `agentic-sdd-implement` reads the B1 tasks, governing `spec.md`/`plan.md` sections, approval reference, and affected code, then implements only B1.

**Asserted workflow behavior (`skills-files/agentic-sdd-implement/SKILL.md`):**

- Executes **only** tasks in B1 (`T01–T02`). It does not start B2 even if all checks pass.
- Runs focused checks plus repository-required gates against the current working tree.
- Invokes `agentic-sdd-verification-review` **once** for the B1 diff and evidence, labelled **self-review** when the same agent performs it.
- Records compact continuation state in `tasks.md` and sets B1's state to `awaiting human review` with the next action, then **stops**.

- **Prerequisites:** Combined approval recorded in `spec.md` and explicit B1 authorization from Step 3 (**optional and unverified here**).
- **Effects:** Adds/updates only the B1 source and test files plus the B1 continuation row in `tasks.md` (`awaiting human review`). No B2 files, commits, or real-home writes.
- **Expected result (illustrative AI response — not executed output):**

  ```text
  Batch B1 (T01-T02): implementation complete; workflow state set to awaiting human review.
  Approval reference: specs/001-json-report/spec.md#approval (r1)

  Changed files:
  - internal/report/json.go: marshal ReportSummary with stable field names
  - internal/report/json_test.go: unit checks for valid report and empty-item edge case
  - specs/001-json-report/tasks.md: B1 -> awaiting human review

  Checks run against current diff:
  - go test ./internal/report -> PASS
  - go vet ./internal/report -> PASS

  Completion review (self-review): Ready for B1 scope.
  - Requirements: AC-1 serializer matches plan.md contract; AC-2 text formatter untouched.
  - Limitations: CLI flag wiring (AC-3) is in B2 and not yet implemented.

  Next permitted action: Review B1 (or request a read-only review), then explicitly authorize B2 if ready.
  ```

- **Cleanup:** Wait until after Step 5 if you want to try review and resume; otherwise discard practice changes with `git status` inspection and targeted checkout/removal.

## Step 5: Perform a read-only review, then explicitly resume to Batch B2 (Stop 3)

At the `awaiting human review` checkpoint, separate **inspecting**, **accepting**, and **authorizing the next batch**:

1. **Optional read-only review:** Ask for findings without changing files or state:

   > Use `agentic-sdd-verification-review` to review the B1 diff and test evidence against `AC-1` and `AC-2`. Report findings only; do not edit code, docs, approvals, or `tasks.md`.

   **Asserted rule (`skills-files/agentic-sdd-verification-review/SKILL.md`):** Review-only requests change nothing. Even a `Ready` verdict is technical evidence, not human approval or permission to begin B2.

2. **Explicit resume to B2:** A bare `"continue"` or `"I accept B1"` does **not** authorize B2. To advance, authorize B2 explicitly under the approved scope:

   > I accept B1. Authorize Batch B2 only under the approved `spec.md` r1 / `plan.md` r1 / `tasks.md` r1 scope, verify it, and stop for my review.

   **Asserted workflow behavior (`workflow-policy.md`):** On resume, the agent reconciles `tasks.md` and the latest explicit authorization with current code state (never trusting stale notes blindly), implements only B2, runs focused checks and connected integration checks, performs one completion review, sets B2 to `awaiting human review`, and **stops**.

- **Prerequisites:** Completed B1 checkpoint (`tasks.md` showing `awaiting human review`) and an active client session (**optional and unverified here**).
- **Effects:** Part 1 (review-only) has **zero file effects**. Part 2 updates only B2 files, tests, and `tasks.md` (`B2 -> awaiting human review`).
- **Expected result:** Part 1 returns findings (`Critical`/`Important`/`Minor` if any) and a technical verdict without mutating files. Part 2 implements B2, reports B2 + connected check results, and stops for final human review.
- **Cleanup:** If practicing in a disposable checkout, clean up practice commits/files once finished; never run real-home `asdx-core apply` as part of a tutorial cleanup.

## Summary of stop boundaries

| Checkpoint | Who decides | What is authorized | Where the agent stops |
|---|---|---|---|
| Drafts (`spec.md`, `plan.md`, `tasks.md`) | Human | Drafting the three files only | Before any implementation edit |
| Combined approval + B1 start | Human | Only named batch (B1) under reviewed revisions | After B1 checks + one completion review (`awaiting human review`) |
| Read-only review | Agent reports; human decides | Inspection only | After returning findings and verdict (no edits) |
| Resume to B2 | Human | Only B2 under approved scope | After B2 checks + completion review (`awaiting human review`) |

## Next steps

- Need a tiny fix with no `specs/` directory? See the [Direct fix tutorial](./direct-fix.md).
- Keep a durable local trail for one batch with [execution runs](../how-to/execution.md), which record the brief, check evidence, and review package without changing approvals.
- Review the core terms in [Core concepts](../introduction/concepts.md).
- Browse all 18 core skills in the [Skill catalog](../reference/skills.md) or command contracts in the [CLI reference](../reference/cli.md).
