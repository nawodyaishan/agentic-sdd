---
title: Core concepts
description: Plain-language guide to skills, profiles, features, direct fixes, batches, evidence, consent, and optional teaching.
---

Agentic-SDD separates **how work is scoped and verified** from **which coding agent or model you run**. Understanding nine core concepts keeps the workflow lightweight and prevents unnecessary files or unintended writes.

## Skill

A **skill** is a portable folder containing a `SKILL.md` instruction file (plus optional `references/` files). The **18 core skills** live in `skills-files/` and cover request routing, specification/plan/task drafting, implementation, debugging, verification review, architecture review, handoffs, and optional teaching.

- Skills work with **one main coding agent** by default—no extra agent runtime or subagent fleet is required.
- Four core skills (`agentic-sdd-grill-me`, `agentic-sdd-handoff`, `agentic-sdd-teach`, and `agentic-sdd-wait-what`) set `disable-model-invocation: true`, meaning they run only when you explicitly ask for them.
- See the full list in the [Skill catalog](../reference/skills.md).

## Profile

A **profile** is an optional OpenCode v2 agent configuration file (`.md` with model and permission frontmatter) installed into `~/.config/opencode/agents/` only when you opt into the OpenCode orchestration pack (`--with-opencode`).

- The pack adds **14 agent profiles** from `opencode-pack/agents/` (including the human-facing `sdd-orchestrator` primary profile and 13 read-only or bounded subagent profiles) plus the `agentic-sdd-orchestrate` skill.
- Skills and profiles solve different problems: **skills** are portable engineering instructions for any supported destination; **profiles** are OpenCode-specific agent definitions.
- See [Supported clients and destinations](clients.md) and [Choose your setup](../getting-started/choose-setup.md) before enabling profiles.

## Feature

A **feature** is one coherent product or engineering outcome tracked under `specs/<nnn-slug>/` using three connected Markdown documents:

| Document | Question it answers |
|---|---|
| `spec.md` | What are we building, what is excluded, how will we verify it, and where is the single combined human approval recorded? |
| `plan.md` | What technical approach, affected components, design trade-offs, risks, and specialist skills apply? |
| `tasks.md` | In what order are tasks executed, how are they grouped into batches, and what is the current continuation state? |

A feature may span multiple sessions. Do not split a feature merely because it has several dependent tasks; split only for independent outcomes, unrelated architectures, or an unmanageable review burden. See the [First feature tutorial](../tutorials/first-feature.md).

## Direct fix

A **direct fix** is a small, clearly scoped change that bypasses `spec.md`, `plan.md`, and `tasks.md` completely. A task qualifies as a direct fix when three conditions hold:

1. The scope is clear.
2. The consequences are understood.
3. Meaningful verification is available.

Changes to permissions, access control, data integrity, public or cross-service contracts, migrations, or live infrastructure are **not** direct fixes—use feature planning instead. See the [Direct fix tutorial](../tutorials/direct-fix.md).

## Batch

A **batch** is a bounded group of tasks in `tasks.md` sized for focused implementation, verification, and comfortable human review.

- After combined feature approval and explicit batch authorization, the agent executes **one** batch, runs focused and repository-required checks, updates `tasks.md` to `awaiting human review` with the next action, and **stops**.
- Passing tests in Batch 1 never authorizes the agent to start Batch 2 automatically.

## Evidence

**Evidence** is what was actually observed—test output tied to the current diff, compiler or linter checks, file hashes, or inspected source paths—not confident narration.

- Static skill checks, green unit tests, a model's `Ready` review verdict, and installer checksums prove only what they directly test.
- An agent's `Ready` or `Needs changes` review verdict is a **technical assessment**, never human acceptance or approval.
- Installer SHA-256 digests (`manifest.json`) detect corrupted backups; they are not cryptographic signatures. See [Files and backups](../reference/files-and-backups.md).

## Consent and explicit authority

Agentic-SDD separates distinct human decisions instead of treating one chat message as blanket permission:

| Decision | What it authorizes | What it does **not** authorize |
|---|---|---|
| Answering clarifying questions | Requirement or design input | Implementing code or creating approvals |
| **Combined approval** (recorded once in `spec.md`) | The reviewed `spec.md` + `plan.md` + `tasks.md` scope | Picking and running a batch without authorization |
| **Batch authorization** | Implementing and verifying **one** named batch | Starting the next batch (`"continue"` alone is not enough) |
| **Live / spending / destructive consent** | Real-home `asdx-core apply`, deployments, releases, tags, or paid model calls | Inferred from drafting code or approving a feature spec |

The installer follows the same principle: `asdx-core` defaults to write-free `preview`, and interactive writes (`asdx`, `asdx-core guide`, `asdx-core restore`) default confirmations to **No**.

## Optional teaching

**Teaching is always optional.** When you want to learn while building—or want a plain-language recap of where things stand—you invoke an explicit-only skill:

- `agentic-sdd-teach` explains a requested concept with a short example or practice question (and maintains a separate learning workspace only if you explicitly ask for one).
- `agentic-sdd-wait-what` re-explains the current proposal, progress, and next decision in plain language without changing any files, tests, or approvals.

Teaching never turns a working repository into mandatory coursework and never advances feature approval or batch state.

## One main agent, not a hidden runtime

The Agentic-SDD vision describes four ways a human may interact with an AI assistant (**learn**, **pair**, **delegate**, and **orchestrate**). Those describe **how you choose to work on a given task**—not four background agents, required daemons, or an automated runtime engine:

- **One main agent is the default.** Specialist knowledge is loaded as skills inside that single session.
- **Subagent profiles are opt-in.** Enabling `--with-opencode` installs OpenCode profile templates, and `agentic-sdd-orchestrate` still delegates only when you explicitly request it.
- **Instructions are not a sandbox.** Skill text and profile frontmatter cannot enforce OS filesystem, network, credential, or billing boundaries on their own; your client's permissions and your own review remain essential.

## Next steps

- Check destination paths and client differences in [Supported clients and destinations](clients.md).
- Decide between CLI, TUI, core-only, and optional pack in [Choose your setup](../getting-started/choose-setup.md).
