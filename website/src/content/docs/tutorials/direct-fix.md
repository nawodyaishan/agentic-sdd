---
title: Direct fix tutorial
description: Walk through a small, bounded bug fix with focused checks, explicit stop points, and an optional explanation step.
---

Use the **direct-fix** path when a change is clearly scoped, its consequences are understood, and meaningful verification is available. A direct fix skips `spec.md`, `plan.md`, `tasks.md`, and batch state completely—even when it touches production code.

> **Scope and verification notice:** Commands using `asdx-core` run locally against a disposable `--home` directory with no real-home writes. Any step that sends prompts to an AI coding client requires your own configured client and provider/model connection; those steps are **optional and unverified here**. Illustrative AI responses below are examples of expected structure, not proof of model behavior. Agentic SDD skills are workflow guidance, not an OS security sandbox.

## When to use a direct fix vs. feature planning

Size alone does not decide the route. Use the consequence test from `skills-files/agentic-sdd-router/references/workflow-policy.md`:

| Situation | Route | Why |
|---|---|---|
| Bounded bug fix with clear expected behavior and a focused regression check | Direct fix (`agentic-sdd-implement`) | Scope, consequences, and verification are already clear. |
| Change affects permissions, data integrity, public/cross-service contracts, migrations, or live-system operations | Feature planning ([First feature](./first-feature.md)) | Needs documented scope, design review, and explicit combined approval before edits. |
| Cause, blast radius, or verification is unclear | Diagnose first (`agentic-sdd-debugging`), then decide | Separating evidence from hypotheses prevents guessing. |

## Example scenario

Suppose a small CLI project has a status formatter that repeats a prefix label (`status: status: ready`) instead of printing `status: ready`. The flags, exit codes, and filesystem behavior must stay unchanged.

## Step 1: Verify your isolated CLI setup

Before working in a project, confirm that your canonical `asdx-core` binary works and preview the core skills in a disposable directory so your real home directory stays untouched. (If you already completed [First five minutes](../getting-started/first-five-minutes.md), you can skip to Step 2.)

```sh
asdx-core version
demo_home=$(mktemp -d)
asdx-core preview --home "$demo_home"
```

- **Prerequisites:** `asdx-core` installed and on `PATH` (see [Installation](../getting-started/installation.md)). No AI provider credentials or network connection are needed.
- **Effects:** Read-only. `asdx-core version` prints build metadata; `asdx-core preview --home "$demo_home"` inspects the empty temporary directory without creating skill files, backups, or logs.
- **Expected result:** `version` exits `0` with version, commit, date, and Go version. `preview` exits `0` and lists the 18 core skills planned for the four client directories under `"$demo_home"` without writing files.
- **Cleanup:** Remove the empty temporary directory:

  ```sh
  rmdir "$demo_home"
  ```

  `rmdir` succeeds only if `"$demo_home"` remained empty.

## Step 2: Send a bounded direct-fix prompt (optional client step)

Open your project in an AI coding client that has the Agentic SDD core skills installed (for example via [Installing skills](../how-to/skills.md)) and send a single bounded prompt:

> Use `agentic-sdd-implement` to fix the repeated `"status: "` label in the status formatter. Keep flags, exit codes, and filesystem behavior unchanged. Add a focused regression test, run the focused test suite, and report what changed.

**Asserted workflow behavior (`skills-files/agentic-sdd-implement/SKILL.md`):**

- The skill inspects only the relevant formatter code, existing tests, and repository constraints. It does **not** look for or create `specs/`, `plan.md`, `tasks.md`, a batch ID, or a batch-state entry.
- If the cause is unclear, the agent follows `skills-files/agentic-sdd-debugging/SKILL.md` to separate evidence from hypotheses before editing.
- If inspecting the code reveals a consequential change (for example, changing a public JSON schema or permissions boundary), the agent **stops** and recommends routing to feature planning instead of stretching the direct-fix path.
- After applying the bounded fix, the agent runs focused checks plus repository-required gates, then performs one completion review via `skills-files/agentic-sdd-verification-review/SKILL.md` (labelled **self-review** when the same agent runs both implementation and review).

- **Prerequisites:** An AI coding client with Agentic SDD core skills installed and a working provider/model connection (**optional and unverified here**), plus a project checkout where the formatter bug exists.
- **Effects:** Modifies only the affected formatter source file and its focused test file in your working tree; runs the project's focused test and repository check commands. Creates no `specs/` files, no Git commits, and no real-home skill writes.
- **Expected result (illustrative AI response — not executed output):**

  ```text
  Scope: Direct fix (no specs/ or batch records created).
  Changed files:
  - internal/status/format.go: remove duplicate "status: " prefix in FormatLine.
  - internal/status/format_test.go: add TestFormatLineSinglePrefix regression check.

  Checks run against current working tree:
  - go test ./internal/status -run TestFormatLineSinglePrefix -> PASS
  - go vet ./internal/status -> PASS

  Completion review (self-review): Ready.
  - Requirements: FormatLine now outputs "status: ready"; flags and exit codes unchanged.
  - Engineering: No unrelated files touched; test fails before fix and passes after.
  - Limitations: Full repository suite was not run unless requested.

  Next permitted action: Review the diff and commit when ready, or ask for a broader test run.
  ```

- **Cleanup:** If you are practicing on a disposable sample repo or temporary branch, inspect and discard the practice edit with `git diff` and `git checkout -- .` (only after verifying you want to drop the changes).

## Step 3: Check human approval and stop boundaries

Notice how authority works across this small fix:

1. **Where the human approves:** Your initial prompt authorized the bounded fix to the status formatter and its focused test. If the agent discovered that the fix required changing a public contract or migrating stored data, it would stop and ask you before proceeding.
2. **Where the agent stops:** After reporting the diff, actual check commands/results, self-review verdict, and limitations, the agent **stops**. It does not expand into unrelated cleanup, run `asdx-core apply`, or commit and push changes on its own.
3. **What `Ready` means:** A `Ready` self-review verdict is a technical assessment by the same agent, not independent review and not permission to merge or deploy.

- **Prerequisites:** The completion summary and working-tree diff from Step 2.
- **Effects:** Read-only human inspection (`git diff`, `git status`).
- **Expected result:** You confirm the diff is minimal, the regression check tests the actual bug rather than tautological code, and no feature paperwork was invented.
- **Cleanup:** None for read-only inspection.

## Step 4: Ask for an optional teaching explanation

When a bug involves an unfamiliar language feature or test pattern, you can invoke `agentic-sdd-teach` explicitly after the fix without turning your repository into a learning workspace:

> Use `agentic-sdd-teach` to explain why the regression test checks the formatted string directly instead of calling the helper under test. Keep it brief and do not create any learning files.

**Asserted workflow behavior (`skills-files/agentic-sdd-teach/SKILL.md`):**

- `agentic-sdd-teach` declares `disable-model-invocation: true`, so it runs only when you explicitly request it.
- Its lightweight default provides a short explanation and one example or practice question without creating lesson files, opening browsers, or advancing any workflow state.

- **Prerequisites:** An active client session with `agentic-sdd-teach` installed (**optional and unverified here**).
- **Effects:** Response-only explanation; no workspace files, specs, or commits are written.
- **Expected result (illustrative AI response — not executed output):**

  ```text
  If a test computes its expected string by calling the same formatter helper, a bug
  inside that helper can pass the test too. Comparing against an independently written
  literal ("status: ready") catches prefix duplication.

  Quick check: What input would test that an already-empty message does not emit a
  dangling "status: " prefix?
  ```

- **Cleanup:** None; no files were created.

## Next steps

- Ready for multi-step work that needs upfront design? Continue to the [First feature tutorial](./first-feature.md).
- Need to understand the core terms? Read [Core concepts](../introduction/concepts.md).
- Want to inspect exact CLI flags or skill IDs? See the [CLI reference](../reference/cli.md) and [Skill catalog](../reference/skills.md).
