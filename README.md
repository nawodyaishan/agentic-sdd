# Agentic SDD

### Build software with AI. Stay the engineer.

[![Release](https://img.shields.io/github/v/release/nawodyaishan/agentic-sdd)](https://github.com/nawodyaishan/agentic-sdd/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/nawodyaishan/agentic-sdd/ci.yml?label=CI)](https://github.com/nawodyaishan/agentic-sdd/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Turn an idea into changes you can understand, review, and maintain—not just code an agent generated.**

Agentic SDD is an open-source toolkit of **18 core engineering skills** plus an **opt-in OpenCode Orchestration Pack (`--with-opencode`)** for coding agents. It gives you a practical way to scope work, plan meaningful changes, delegate bounded tasks to specialized subagents, debug failures, and verify results—with you in charge of the decisions.

Use it with **OpenCode, Codex, Claude Code, or Antigravity CLI**. Start with one agent, or opt into multi-model specialist orchestration (`sdd-orchestrator`, OpenAI `gpt-6.1-sol` / `gpt-6.1-terra`, Claude Opus/Sonnet, Gemini Flash, DeepSeek, and Console Free models) when you want it.

[Quick start](#quick-start) · [Usage guide](#complete-usage-guide) · [OpenCode orchestration (`--with-opencode`)](#opencode-orchestration-pack---with-opencode) · [Full orchestration docs](docs/orchestration-guide.md) · [Installer reference](docs/installer-reference.md) · [Product vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md)

## Why Agentic SDD?

Writing code is getting faster. Knowing **what to build, why it works, and whether it's ready** still takes engineering judgment.

Agentic SDD helps you bring that judgment into your work with AI:

- **Agree on the outcome before the implementation.** Make requirements and important trade-offs visible instead of letting the agent guess.
- **Keep changes reviewable.** Work in manageable batches, with a clear summary of what changed and what was checked.
- **Skip unnecessary ceremony.** A small bug fix stays a small bug fix. Larger features get a connected specification, plan, and task list.
- **Delegate deliberately without losing control.** One human-facing orchestrator scopes subagent tasks, checks their diffs and test results, and reports back in one consolidated summary.
- **Learn without switching workflows.** Ask for explanations, hints, or debugging practice when a topic is unfamiliar. Teaching is optional.

> **Learning first when developing competence. Delegation first when applying established competence. Verification always.**

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

**Install with the OpenCode Orchestration Pack (`--with-opencode`):**

```sh
# Preview core skills + agentic-sdd-orchestrate + 14 OpenCode v2 agent profiles
agentic-sdd preview --with-opencode

# Apply (backs up any existing managed profiles first)
agentic-sdd apply --with-opencode

# Optional: also configure routine implement/test profiles to try Free models first
agentic-sdd apply --with-opencode --opencode-free-first
```

Already installed? Run `brew upgrade nawodyaishan/tap/agentic-sdd`, then preview and apply to update your installed skills and profiles.

<details>
<summary><strong>Linux or a source checkout</strong></summary>

With Go 1.23 or newer installed:

```sh
git clone https://github.com/nawodyaishan/agentic-sdd.git
cd agentic-sdd
go run ./cmd/agentic-sdd preview
go run ./cmd/agentic-sdd apply

# Or with the OpenCode orchestration pack:
go run ./cmd/agentic-sdd preview --with-opencode
go run ./cmd/agentic-sdd apply --with-opencode
```

To try a read-only preview with a disposable home first, pass `--home /tmp/test-home` or follow the [first-five-minutes guide](docs/getting-started.md#first-five-minutes-preview-then-stop).

</details>

---

## Complete usage guide

### CLI commands at a glance

| Command | What it does |
|---|---|
| `agentic-sdd` or `agentic-sdd preview` | Preview installing/updating the 18 core skills across all 4 client directories (no files written). |
| `agentic-sdd apply` | Back up changed existing skills, then install the 18 core skills. |
| `agentic-sdd preview --with-opencode` | Preview installing the 18 core skills, `agentic-sdd-orchestrate`, and 14 OpenCode v2 agent profiles in `~/.config/opencode/agents/`. |
| `agentic-sdd apply --with-opencode` | Back up any replaced skills/profiles and install the core skills + OpenCode orchestration pack. |
| `agentic-sdd apply --with-opencode --opencode-free-first` | Above + back up and preset the 4 routine implement/test profiles (`sdd-implement-1/-2`, `sdd-test-implement-1/-2`) to use the Free-first model ladder. |
| `agentic-sdd guide` | Interactive numbered-prompt wizard to preview, install, restore, or inspect skills on a terminal. |
| `agentic-sdd backups` | List all saved backups newest first (both skill directories and `opencode-agents` profiles). |
| `agentic-sdd restore [BACKUP_ID]` | Preview restoring a backup (or select one interactively on a terminal). |
| `agentic-sdd restore apply [BACKUP_ID]` | Back up current state and restore skills and OpenCode agent profiles from `BACKUP_ID`. |
| `agentic-sdd logs` | Show the latest `apply` or `restore` log under `~/.agentic-sdd/logs/`. |
| `agentic-sdd version` | Print version, commit, build date, and Go version. |
| `agentic-sdd help <topic>` | Show focused help for `preview`, `apply`, `backups`, `logs`, `restore`, `guide`, `version`, or `orchestrate`. |
| `agentic-sdd help orchestrate` | Show usage and flags for the OpenCode orchestration pack. |

All commands accept `--home PATH` (to target an isolated home directory) and `--repo PATH` (to read `skills-files/` and `opencode-pack/` from a local checkout instead of the embedded binary assets).

### Everyday prompts in your project

1. **Start or plan a feature (with combined human approval):**
   > Use `agentic-sdd-router`. Inspect this repository, ask at most the consequential questions you can't resolve from the code, and draft `spec.md`, `plan.md`, and `tasks.md` for my combined approval. Stop before implementing.
2. **Fix a small bug directly (no `specs/` paperwork):**
   > Use `agentic-sdd-implement` to fix this nil-pointer bug. Keep the diff minimal, run the focused test suite, and report what changed.
3. **Implement one approved batch with specialist subagents (`--with-opencode`):**
   > I approve `spec.md`, `plan.md`, and `tasks.md`. Use `agentic-sdd-orchestrate` to implement Batch B1 with `sdd-openai-implement` (or `sdd-implement-9r-1`), then verify the diff with `sdd-review-verification-2` and stop for my review.
4. **Run a high-depth architecture or specification review:**
   > Use `agentic-sdd-orchestrate` to dispatch `sdd-openai-architect` (or `sdd-architecture-review-9r`) on our draft `spec.md` and `plan.md`. Reconcile its findings against the codebase and present a ranked summary.
5. **Learn an unfamiliar concept or stress-test a design:**
   > Use `agentic-sdd-teach` to walk me through how `internal/skillsync/restore.go` validates backup digests, or use `agentic-sdd-grilling` to challenge my migration plan.

For complete walkthroughs, see the [Full Usage & Orchestration Guide](docs/orchestration-guide.md) and [Getting Started Examples](docs/getting-started.md).

---

## OpenCode Orchestration Pack (`--with-opencode`)

When you opt into `--with-opencode`, the CLI installs **`agentic-sdd-orchestrate`** into the 4 client skill directories and places **14 OpenCode v2 agent profiles** in `~/.config/opencode/agents/`:

| Profile | Mode & Role | Default Model | MCP Access (`execute` + tools) |
|---|---|---|---|
| `sdd-orchestrator` | Primary communicator (`mode: primary`) | Active session model | Inherits session tools |
| `sdd-openai-architect` | Read-only architecture, spec & critical decision advisor | `openai/gpt-6.1-sol` | `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-openai-implement` | Authorized complex implementation, debugging & refactoring | `openai/gpt-6.1-terra` | `codegraph_*`, `context7_*` (`exa_*` denied) |
| `sdd-architecture-advisor` | Read-only architecture & operational risk critique | `9router/ag/claude-sonnet-5-5-high` | `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-debug-diagnose` | Read-only root-cause hypotheses & triage | `opencode/deepseek-v4.1-flash-free` | `codegraph_*`, `context7_*` |
| `sdd-test-design` | Read-only test cases & negative-path design | `opencode/deepseek-v4.1-flash-free` | `codegraph_*`, `context7_*` |
| `sdd-research-librarian` | Read-only upstream docs & API version research | `opencode/deepseek-v4.1-flash-free` | `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-docs-worker` | Bounded documentation writer | `opencode/deepseek-v4.1-flash-free` | `codegraph_*`, `context7_*` |
| `sdd-visual-inspector` | Read-only screenshot, diagram & log image inspector | `opencode/glm-5.3-flash-free` | `codegraph_*`, `context7_*` |
| `sdd-architecture-review-9r` | Read-only Opus architecture review (backed up & updated) | `9router/ag/claude-opus-5-5-high` | `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-code-review-9r` | Read-only Opus code/verification review (backed up & updated) | `9router/ag/claude-opus-5-5-high` | `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-review-verification-1` | Read-only GLM verification review (backed up & updated) | `opencode-go/glm-5.3-flash` | `codegraph_*`, `context7_*` |
| `sdd-review-verification-2` | Read-only Beam Free supplementary review (backed up & updated) | `opencode/beam-free` | `codegraph_*`, `context7_*` |
| `sdd-lightweight` | Bounded read-only discovery (`steps: 10`, backed up & updated) | `opencode/beam-free` | `codegraph_*`, `context7_*` |

With `--opencode-free-first`, the installer also backs up and updates `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, and `sdd-test-implement-2` to start on `opencode/deepseek-v4.1-flash-free`. Your `opencode.json`, `AGENTS.md`, `sdd-spec-9r.md`, `sdd-plan-9r.md`, `sdd-implement-9r-*.md`, and `vision.md` remain untouched. See [`opencode-pack/README.md`](opencode-pack/README.md) and [`opencode-pack/routing.md`](opencode-pack/routing.md) for full details.

---

## The workflow

**Understand → Scope → Plan → Implement → Verify → Review**

For a feature that needs planning, your agent prepares three connected documents:

| Document | The question it answers |
|---|---|
| `spec.md` | What are we building, what's out of scope, and how will we know it works? |
| `plan.md` | How should we build it, and what risks or trade-offs matter? |
| `tasks.md` | What do we do first, how do we check it, and where do we stop? |

Review them together and give **one combined approval**. Then authorize one implementation batch. The agent works on that batch, runs relevant checks, explains the result, and stops for your review.

**For a clear, small fix:** go straight to implementation and verification. No feature paperwork required.

## What comes in the toolkit?

| What you need | Skills to reach for |
|---|---|
| Start, resume, or orchestrate work | `agentic-sdd-router`, `agentic-sdd-orchestrate` (`--with-opencode`), `agentic-sdd-bootstrap`, `agentic-sdd-handoff` |
| Shape a feature | `agentic-sdd-spec`, `agentic-sdd-plan`, `agentic-sdd-tasks`, `agentic-sdd-research-spec` |
| Make and check changes | `agentic-sdd-implement`, `agentic-sdd-debugging`, `agentic-sdd-verification-review`, `agentic-sdd-architecture-review` |
| Think through a decision | `agentic-sdd-grilling`, `agentic-sdd-grill-me`, `agentic-sdd-questionnaire`, `agentic-sdd-wait-what` |
| Learn and improve the process | `agentic-sdd-teach`, `agentic-sdd-drift-retro`, `agentic-sdd-writing-for-agents` |

Browse the [18 core skills](skills-files/) and the [opt-in OpenCode orchestration pack](opencode-pack/). The helpers include adaptations of [Matt Pocock's productivity principles](skills-files/agentic-sdd-router/references/matt-pocock-notice.md).

## Where the files go

| Target | Path | Managed by |
|---|---|---|
| OpenCode and shared agents | `~/.agents/skills` | Default & `--with-opencode` |
| Codex | `~/.codex/skills` | Default & `--with-opencode` |
| Claude Code | `~/.claude/skills` | Default & `--with-opencode` |
| Antigravity CLI | `~/.gemini/antigravity-cli/skills` | Default & `--with-opencode` |
| OpenCode v2 agent profiles | `~/.config/opencode/agents/*.md` | `--with-opencode` (`--opencode-free-first`) only |

## Safety model

The installer **previews by default**, backs up existing skills and `.md` agent profiles with SHA-256 digests before replacing them, stages updates atomically, and provides a preview-first `restore` command. Repeating an `apply` with unchanged content is a clean no-op. It rejects symlinks and special files in managed targets.

```sh
agentic-sdd backups                   # list saved installations
agentic-sdd restore BACKUP_ID         # preview a restore
agentic-sdd restore apply BACKUP_ID   # restore skills and OpenCode agent profiles
```

**The workflow is guidance, not an OS security sandbox.** Your agent client's permissions, execution environment, and your review remain essential.

## Explore further

- **[Full Usage & Orchestration Guide](docs/orchestration-guide.md)** — complete CLI, orchestration, OpenAI/Claude/Gemini/DeepSeek specialist, and MCP usage guide.
- **[Getting started](docs/getting-started.md)** — first-use walkthrough and worked examples.
- **[Installer reference](docs/installer-reference.md)** — commands, Format 2/3 backups, restore, and file behavior.
- **[OpenCode pack reference](opencode-pack/README.md)** & **[Model routing policy](opencode-pack/routing.md)** — specialist profiles, permissions, and fallback ladders.
- **[Product vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md)** — why the project exists and where it is headed.
- **[Documentation index](docs/README.md)** & **[Contributing](CONTRIBUTING.md)**.

Open source under the [MIT License](LICENSE).
