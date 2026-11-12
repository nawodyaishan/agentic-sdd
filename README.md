# Agentic SDD

### Build software with AI. Stay the engineer.

[![Release](https://img.shields.io/github/v/release/nawodyaishan/agentic-sdd)](https://github.com/nawodyaishan/agentic-sdd/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/nawodyaishan/agentic-sdd/ci.yml?label=CI)](https://github.com/nawodyaishan/agentic-sdd/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Turn an idea into changes you can understand, review, and maintain—not just code an agent generated.**

Agentic SDD is an open-source toolkit of **18 core engineering skills** plus an **opt-in OpenCode Orchestration Pack (`--with-opencode`)** for coding agents. It gives you a practical way to scope work, plan meaningful changes, delegate bounded tasks to specialized subagents, debug failures, and verify results—with you in charge of the decisions.

Use it with **OpenCode, Codex, Claude Code, or Antigravity CLI**. Explore and install skills through the guided terminal wizard (`agentic-sdd guide`) or the packaged terminal UI (`agentic-sdd tui`), or keep using the explicit `preview`/`apply` commands for scripts and CI. Start with one agent, or opt into multi-model specialist orchestration (`sdd-orchestrator`, OpenAI `gpt-6.1-sol` / `gpt-6.1-terra`, Claude Opus/Sonnet, Gemini Flash, DeepSeek, and Console Free models) when you want it.

[Quick start](#quick-start) · [What's new](#whats-new-feature-010013) · [Usage guide](#complete-usage-guide) · [OpenCode orchestration (`--with-opencode`)](#opencode-orchestration-pack---with-opencode) · [Full orchestration docs](docs/orchestration-guide.md) · [Installer reference](docs/installer-reference.md) · [Product vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md)

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
brew install --cask nawodyaishan/tap/agentic-sdd
agentic-sdd preview
```

Preview shows which skills would be installed or replaced. **It does not write changes.** When you're happy with the plan:

```sh
agentic-sdd apply
```

In a real interactive terminal, a bare `agentic-sdd` from the Homebrew cask opens the terminal UI instead. `agentic-sdd preview` always keeps the explicit, write-free CLI, so scripts, pipes and `AGENTIC_SDD_FORCE_INTERACTIVE=1` are unaffected.

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
<summary><strong>Upgrading an earlier formula installation</strong></summary>

Earlier releases used a Homebrew formula with the same name. The cask ships with the first cask-bearing release; use `brew install nawodyaishan/tap/agentic-sdd` for older formula releases. When the tap publishes the formula-to-cask migration, Homebrew 5.0.6 or newer migrates automatically on `brew update && brew upgrade nawodyaishan/tap/agentic-sdd`; finish with `brew uninstall --formula agentic-sdd` if Homebrew asks. On older Homebrew, run `brew uninstall --formula agentic-sdd` first, then the `brew install --cask` command above.

</details>

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

### Try the interactive CLI locally

From a repository checkout with Go and Task installed:

```sh
task run
```

This opens the animated guide in your terminal with a fresh temporary home, removed on exit. Installs affect that temporary home, not your real agent directories. Animation respects `TERM`, `NO_COLOR`, and `COLUMNS`; inherited flow tracing and forced-interactive settings are disabled for the demo. Enter `q` at the menu to quit. This is not an OS sandbox; explicitly choosing the update check can contact GitHub.

### Try the interactive CLI in Docker

From a repository checkout, with Docker, Task, and the `golang:1.25-alpine` image already available locally:

```sh
task docker-guide
```

Explore the numbered menu and optional animated intro; enter `q` at the menu to quit. The repository is mounted read-only, installs target only `/tmp/demo-home` inside the disposable container, and your real home is never mounted. Networking is disabled, so checking for updates shows the offline fallback. Animation respects your terminal's `TERM`, `NO_COLOR`, and `COLUMNS` settings. All container changes disappear on exit.

For automated container checks instead, run `task docker-guide-e2e`.

### Guided CLI (`agentic-sdd guide`)

`agentic-sdd guide` is the zero-dependency, numbered-prompt path through the same installer. It needs an interactive terminal; with piped or `/dev/null` stdin it exits `2` without reading input.

On terminal stdout it shows:

- a branded introduction with the tagline, creator, installed version and documentation URL;
- a grouped main menu: **1** preview and install, **2** restore from a backup, **3** view the latest install or restore log, **4** About and getting started, **5** check for CLI updates (contacts GitHub), **6** explore skills and agent profiles, and **q** to quit;
- a staged flow — choose options, review exact changes, then confirm — with complete paths, per-destination counts and a preview restoring command.

Every confirmation defaults to **No**; EOF cancels without writing, and `q` exits the menu. Installing through the wizard performs the same `apply` operation as the CLI, including backups and the operation log. `task run` and `task docker-guide` are disposable demo wrappers for this flow.

Scripted use is unaffected: with non-terminal stdout, the guide and every other command keep their historical plain output and exit codes.

### Terminal UI (`agentic-sdd tui`)

In a real terminal, `agentic-sdd tui` opens the packaged terminal UI:

```sh
agentic-sdd tui
agentic-sdd tui --home ~ --with-opencode=false   # core skills only
agentic-sdd tui --opencode-free-first            # Free-first profile presets
```

It provides four destinations: **Home** (setup status and next actions), **Install & Manage** (client selection, optional OpenCode pack and Free-first preset, preview, install, backups and restore), **Explore** (searchable catalog of core skills and agent profiles, with purpose, per-destination status and profile routing details) and **Activity & Help** (bounded logs, quick start, shortcuts, commands, troubleshooting, diagnostics and version).

Presentation is responsive: sidebar/list/details at 110+ columns, a compact layout at 80–109, a single list with separate details at 60–79, and explicit resize guidance below that. A single high-contrast Agentic Dark palette (with TrueColor, ANSI-256 and ANSI-16 fallbacks) keeps text and controls readable, while `NO_COLOR`, `AGENTIC_SDD_ASCII=1` and `AGENTIC_SDD_REDUCED_MOTION=1` select plain, ASCII and static modes. Status is always labelled, never conveyed by color alone.

The TUI never writes before an explicit confirmation (default **No**). During an apply or restore, `q`/Esc/navigation cannot abandon the transaction; Ctrl+C requests exit after the write finishes or rolls back. A missing, mismatched or non-executable companion is a clear error, never a silent fallback to another write workflow.

The Home screen checks GitHub once per session for a newer CLI release without blocking local work; set `AGENTIC_SDD_NO_UPDATE_CHECK=1` to suppress it, or retry explicitly from the UI. The online guide and author links open a browser only when you choose the action.

The TUI includes the OpenCode profile pack by default; pass `--with-opencode=false` for the core skills only. It needs a real input and output terminal, plus the paired `agentic-sdd-tui` binary next to `agentic-sdd`. The CLI checks the companion's build identity before starting it; a missing, non-executable or mismatched companion is a clear error with rebuild guidance, never a silent switch to another workflow. Pipes, scripts, `AGENTIC_SDD_FORCE_INTERACTIVE=1` and every explicit command or flag keep the standard CLI. `agentic-sdd preview` is always the write-free escape hatch.

From a source checkout, build a matching local pair (same revision, commit and source-content digest, so the handshake passes without a release tag):

```sh
task -t Taskfile.experimental.yml tui:pair
bin/tui-pair/agentic-sdd tui
```

The task needs Go 1.26 or newer for the nested TUI module. It prints the injected revision and source digest for both binaries.

## What's new (feature 010–013)

These features added the interactive layers on top of the same preview-first installer. Explicit commands, flags, exit codes and non-terminal output remain compatible.

| Feature | What it added |
|---|---|
| **010 — Guided CLI & flow tracing** | `agentic-sdd guide`, grouped previews, per-topic help, `hint:` remediation, a non-TTY `restore` fix, and the opt-in `AGENTIC_SDD_TRACE` JSON Lines flow trace. |
| **011 — Branded CLI** | ASCII wordmark with tagline, creator, version and documentation URL; context-correct commands; review-with-`b` navigation; aligned change and recovery summaries; menu choice **6** to explore skills and profiles; and an explicit, bounded update check (choice **5**) with copyable Homebrew instructions. |
| **012 — Readable catalog** | Definition-list wrapping that stays inside narrow widths, role-grouped profile lists with stable numbers, `b`/`B`/`Enter` back navigation, concise safety badges, status labels and next-action hints; plus an [isolated Charm v2 prototype](docs/tui-architecture.md) in `experiments/tui` that leaves the root module dependency-free. |
| **013 — Unified TUI** | The paired `agentic-sdd-tui` application: Home, Install & Manage, Explore and Activity & Help; searchable catalog; client selection; preview-first install, backup browsing and restore; bounded logs, help, onboarding and diagnostics; a responsive high-contrast theme; once-per-session update check; Homebrew bare-command launch and paired cask packaging. |

Specs: [010](specs/010-guided-cli-ux/spec.md), [011](specs/011-cli-brand-styling/spec.md), [012](specs/012-cli-tui-remediation/spec.md), [013](specs/013-unified-tui-experience/spec.md) — [tasks](specs/013-unified-tui-experience/tasks.md) records the remaining Feature 013 review state.

---

## Complete usage guide

### CLI commands at a glance

| Command | What it does |
|---|---|
| `agentic-sdd` or `agentic-sdd preview` | Preview installing/updating the 18 core skills across all 4 client directories (no files written). A bare `agentic-sdd` opens the TUI only for the Homebrew package in a real interactive terminal; `preview` always stays on the CLI. |
| `agentic-sdd apply` | Back up changed existing skills, then install the 18 core skills. |
| `agentic-sdd preview --with-opencode` | Preview installing the 18 core skills, `agentic-sdd-orchestrate`, and 14 OpenCode v2 agent profiles in `~/.config/opencode/agents/`. |
| `agentic-sdd apply --with-opencode` | Back up any replaced skills/profiles and install the core skills + OpenCode orchestration pack. |
| `agentic-sdd apply --with-opencode --opencode-free-first` | Above + back up and preset the 4 routine implement/test profiles (`sdd-implement-1/-2`, `sdd-test-implement-1/-2`) to use the Free-first model ladder. |
| `agentic-sdd guide` | Interactive numbered-prompt wizard: install/update, restore, view logs, About, explicit update check, and the skills/profiles explorer. Requires a terminal; writes nothing before confirmation. |
| `agentic-sdd tui` | Open the packaged terminal UI in a real terminal (needs the paired `agentic-sdd-tui` binary). Supports `--home`, `--repo`, `--with-opencode` (default on) and `--opencode-free-first`. |
| `agentic-sdd backups` | List all saved backups newest first (both skill directories and `opencode-agents` profiles). |
| `agentic-sdd restore [BACKUP_ID]` | Preview restoring a backup (or select one interactively on a terminal). |
| `agentic-sdd restore apply [BACKUP_ID]` | Back up current state and restore skills and OpenCode agent profiles from `BACKUP_ID`. |
| `agentic-sdd logs` | Show the latest `apply` or `restore` log under `~/.agentic-sdd/logs/`. |
| `agentic-sdd version` | Print version, commit, build date, and Go version. |
| `agentic-sdd help <topic>` | Show focused help for `preview`, `apply`, `backups`, `logs`, `restore`, `guide`, `version`, `tui`, or `orchestrate`. |
| `agentic-sdd help orchestrate` | Show usage and flags for the OpenCode orchestration pack. |

All commands accept `--home PATH` (to target an isolated home directory) and `--repo PATH` (to read `skills-files/` and `opencode-pack/` from a local checkout instead of the embedded binary assets).

### Environment and accessibility switches

| Variable | Applies to | Effect |
|---|---|---|
| `NO_COLOR` (non-empty) | CLI and TUI | Disable ANSI styling. Text, structure and status words are unchanged. |
| `TERM=dumb` | CLI | Disable color and use ASCII glyphs. |
| `COLUMNS` | CLI | Set presentation width; otherwise 80. |
| `AGENTIC_SDD_ANIMATE=1` | `guide` | Opt in to the 1.5-second two-pass brand highlight on a capable terminal. Static by default; `NO_COLOR`, dumb, narrow, or non-terminal output suppresses it. |
| `AGENTIC_SDD_TRACE=<path>` | CLI | Append a privacy-safe JSON Lines flow trace (resolved environment, screens, prompts, plan counts, result and exit). Unset or empty disables it. |
| `AGENTIC_SDD_FORCE_INTERACTIVE=1` | CLI tests | Treat stdin, stdout and stderr as terminals for container tests. It never answers or skips a confirmation. |
| `AGENTIC_SDD_ASCII=1` | TUI | Use ASCII borders and symbols. |
| `AGENTIC_SDD_REDUCED_MOTION=1` | TUI | Disable animation and show static rendering. |
| `AGENTIC_SDD_NO_UPDATE_CHECK=1` | TUI | Suppress the automatic once-per-session release check; explicit retry remains available. |

The trace schema and journey catalog live in [guide-flows.md](docs/guide-flows.md). A trace is diagnostic evidence, not the apply log, approval, or proof of model/client behavior.

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

Backups and operation logs live under `~/.agentic-sdd/` (`backups/` and `logs/`). The TUI is distributed as a paired `agentic-sdd-tui` companion binary; the Homebrew cask installs both, and the CLI verifies their build identity at launch.

## Safety model

The installer **previews by default**, backs up existing skills and `.md` agent profiles with SHA-256 digests before replacing them, stages updates atomically, and provides a preview-first `restore` command. Repeating an `apply` with unchanged content is a clean no-op. It rejects symlinks and special files in managed targets.

The guide and TUI are alternate front ends over that same engine. Neither writes before an explicit confirmation (default **No**). During a TUI write, `q`/Esc/navigation cannot abandon the transaction; Ctrl+C requests exit after the operation finishes or rolls back. A missing, mismatched or non-executable TUI companion is an error, never a silent fallback to another write workflow.

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
- **[Guided CLI journeys and flow tracing](docs/guide-flows.md)** — screen and prompt catalog, trace schema, and `AGENTIC_SDD_TRACE` events.
- **[TUI architecture](docs/tui-architecture.md)** — packaged companion design, experimental nested module, and toolchain migration blueprint.
- **[Fleet Manager (terminal UI)](docs/fleet-manager-tui.md)** — explicit local `fleet.json` source, staged model changes, source save pending separate approval, preview-only deployment and managed-backup restore.
- **[OpenCode pack reference](opencode-pack/README.md)** & **[Model routing policy](opencode-pack/routing.md)** — specialist profiles, permissions, and fallback ladders.
- **[Product vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md)** — why the project exists and where it is headed.
- **[Documentation index](docs/README.md)** & **[Contributing](CONTRIBUTING.md)**.

Open source under the [MIT License](LICENSE).
