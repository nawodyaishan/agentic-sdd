# Agentic SDD

### Build software with AI. Stay the engineer.

[![Release](https://img.shields.io/github/v/release/nawodyaishan/agentic-sdd)](https://github.com/nawodyaishan/agentic-sdd/releases/latest)
[![CI](https://github.com/nawodyaishan/agentic-sdd/actions/workflows/ci.yml/badge.svg)](https://github.com/nawodyaishan/agentic-sdd/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Turn an idea into changes you can understand, review, and maintain—not just code an agent generated.**

Agentic SDD is an open-source toolkit of **18 engineering skills for coding agents**. It gives you a practical way to scope work, plan meaningful changes, debug failures, and verify results—with you in charge of the decisions.

Use it with **OpenCode, Codex, Claude Code, or Antigravity CLI**. Start with one agent. Add teaching or specialist reviews when you need them. No agent fleet or new runtime required.

[Get started](#quick-start) · [See the workflow](#the-workflow) · [Read the examples](docs/getting-started.md) · [Explore the vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md)

## Why Agentic SDD?

Writing code is getting faster. Knowing **what to build, why it works, and whether it's ready** still takes engineering judgment.

Agentic SDD helps you bring that judgment into your work with AI:

- **Agree on the outcome before the implementation.** Make requirements and important trade-offs visible instead of letting the agent guess.
- **Keep changes reviewable.** Work in manageable batches, with a clear summary of what changed and what was checked.
- **Skip unnecessary ceremony.** A small bug fix stays a small bug fix. Larger features get a connected specification, plan, and task list.
- **Learn without switching workflows.** Ask for explanations, hints, or debugging practice when a topic is unfamiliar. Teaching is optional.
- **Pick up where you left off.** Keep decisions and progress in your repository rather than relying only on a long chat.

> **Learning first when developing competence. Delegation first when applying established competence. Verification always.**

## Who is it for?

| If you're… | Use Agentic SDD to… |
|---|---|
| Building solo with a coding agent | Deliver useful changes without losing track of scope or review. |
| A student, intern, or self-taught developer | Practice professional engineering while building something real. |
| An experienced engineer | Delegate familiar work while keeping technical decisions and acceptance in your hands. |
| Working in backend, platform, DevOps, or SRE | Make risks, recovery steps, and permission for live operations explicit. |
| Maintaining open source or a small team | Give contributors and their agents a shared way to prepare and explain changes. |

One methodology, not a separate process for every experience level.

## Quick start

### 1. Install the toolkit

**macOS with Homebrew:**

```sh
brew install nawodyaishan/tap/agentic-sdd
agentic-sdd preview
```

Preview shows which skills would be installed or replaced. **It does not write changes.** When you're happy with the plan:

```sh
agentic-sdd apply
```

Already installed? Run `brew upgrade nawodyaishan/tap/agentic-sdd`, then preview and apply to update your installed skills.

<details>
<summary><strong>Linux or a source checkout</strong></summary>

With Go 1.23 or newer installed:

```sh
git clone https://github.com/nawodyaishan/agentic-sdd.git
cd agentic-sdd
go run ./cmd/agentic-sdd preview
```

After reviewing the preview, install the skills:

```sh
go run ./cmd/agentic-sdd apply
```

These commands install skills into your home directory. To try a read-only preview with a disposable home first, follow the [first-five-minutes guide](docs/getting-started.md#first-five-minutes-preview-then-stop).

</details>

### 2. Start in your project

Open your coding agent in the repository you want to work on. Ask it to load the router skill:

> Use `agentic-sdd-router`. Help me plan a feature for this project. Inspect what's already here, ask about important decisions you can't resolve, and stop before implementation.

The router helps choose the right next step. You don't need to memorize all 18 skills or invoke them in sequence.

### 3. Try a real task

**Fix something small**

> Use `agentic-sdd-implement` to fix this validation bug. Keep the change focused, add a regression test, and report what you checked.

**Plan a larger change**

> Use `agentic-sdd-router` to draft a spec, plan, and tasks for CSV export. Preserve the existing output format. Present the three documents together for my review; don't implement yet.

**Learn an unfamiliar part of your code**

> Use `agentic-sdd-teach` to help me understand this retry loop. Ask me to reason through a failure case before showing the answer.

**Get a review without edits**

> Use `agentic-sdd-verification-review` to review this diff against the requirements. Report findings only; don't change files.

See the [worked examples](docs/getting-started.md) for a full feature, a direct fix, and resuming work safely.

## The workflow

**Understand → Scope → Plan → Implement → Verify → Review**

For a feature that needs planning, your agent prepares three connected documents:

| Document | The question it answers |
|---|---|
| `spec.md` | What are we building, what's out of scope, and how will we know it works? |
| `plan.md` | How should we build it, and what risks or trade-offs matter? |
| `tasks.md` | What do we do first, how do we check it, and where do we stop? |

Review them together and give **one combined approval**. Then authorize one implementation batch. The agent works on that batch, runs relevant checks, explains the result, and stops for your review.

You decide whether to request corrections or start the next batch. Passing tests isn't permission to keep going, and approving code isn't permission to deploy it.

**For a clear, small fix:** go straight to implementation and verification. No feature paperwork required.

## What comes in the toolkit?

The skills cover the engineering work around writing code—not just generating it.

| What you need | Skills to reach for |
|---|---|
| Start or resume work | Router, bootstrap, handoff |
| Shape a feature | Spec, plan, tasks, focused research |
| Make and check changes | Implementation, debugging, verification review, architecture review |
| Think through a decision | Grilling, questionnaires, plain-language clarification |
| Learn and improve the process | Teaching, drift/retrospective, writing for agents |

Browse the [18 skills and their instructions](skills-files/). The helpers include adaptations of [Matt Pocock's productivity principles](skills-files/agentic-sdd-router/references/matt-pocock-notice.md).

## Where the skills go

The installer copies the same toolkit to all four destinations:

| Client | Skill directory |
|---|---|
| OpenCode and shared agents | `~/.agents/skills` |
| Codex | `~/.codex/skills` |
| Claude Code | `~/.claude/skills` |
| Antigravity CLI | `~/.gemini/antigravity-cli/skills` |

Only `agentic-sdd-*` skills are managed; unrelated skills are left alone. Client discovery and invocation can differ—portable instructions don't guarantee identical behavior across every client or model.

## Safety model

The installer **previews by default**, backs up existing skills before replacing them, and provides a preview-first restore command. Repeating an apply with unchanged skill content leaves those skills alone. It rejects symlinks and special files in managed skill trees.

```sh
agentic-sdd backups             # list saved installations
agentic-sdd restore BACKUP_ID   # preview a restore
```

To carry out a reviewed restore, use `agentic-sdd restore apply BACKUP_ID`.

Backups recover skill content, not exact original permissions or ownership. For flags, storage locations, permission behavior, and recovery details, see the [installer reference](docs/installer-reference.md).

**The workflow is guidance, not a security sandbox.** Your agent client's permissions, execution environment, and your review remain important. No prompt can guarantee correct code, prevent every unwanted action, or enforce a spending limit.

## Support and release scope

- **Available today:** 18 portable skills, the preview-first installer, backups, and restore. No premium model or multiple subagents are required by the workflow; your chosen agent still needs its own setup.
- **Distribution:** macOS universal binary through Homebrew or [GitHub Releases](https://github.com/nawodyaishan/agentic-sdd/releases/latest); Go source usage on Linux. Linux CI and isolated installer tests provide execution coverage. Native macOS/Homebrew runtime testing remains unobserved; Windows runtime support is not claimed.
- **Not part of the default product:** autonomous orchestration, automatic model routing, or native independent-review execution. Experimental OpenCode runtime tooling is separate and disabled in the default CLI; you do **not** need it to use the portable skills with OpenCode. See [integration limits](integrations/opencode/usage.md).

We don't promise zero hallucinations, guaranteed cost savings, or identical results across models. The goal is a clearer engineering process and evidence you can inspect.

## Explore further

- **[Getting started](docs/getting-started.md)** — first-use walkthrough and worked examples.
- **[Installer reference](docs/installer-reference.md)** — commands, backups, restore, and file behavior.
- **[Product vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md)** — why the project exists and where it could go.
- **[Documentation](docs/README.md)** — guides and technical background.
- **[Contributing](CONTRIBUTING.md)** — development setup, tests, and how to propose improvements.
- **[Check runner](harness/README.md)** — source-only verification tooling for contributors.

## Help make it better

Tried it on a real project? Tell us where it helped—and where it got in your way. Bug reports, clearer examples, skill improvements, and tests are welcome.

[Open an issue](https://github.com/nawodyaishan/agentic-sdd/issues) or read the [contribution guide](CONTRIBUTING.md). For security concerns, follow the [reporting guidance](CONTRIBUTING.md#reporting-bugs-and-security-concerns); don't post credentials or private project details publicly.

Open source under the [MIT License](LICENSE).
