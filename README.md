# Agentic SDD

### Build software with AI. Stay the engineer.

[![Release](https://img.shields.io/github/v/release/nawodyaishan/agentic-sdd)](https://github.com/nawodyaishan/agentic-sdd/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/nawodyaishan/agentic-sdd/ci.yml?label=CI)](https://github.com/nawodyaishan/agentic-sdd/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Turn an idea into changes you can understand, review, and maintain.**

Agentic SDD is an open-source toolkit of **18 core engineering skills** plus an
opt-in OpenCode orchestration pack for coding agents. It helps you scope work,
plan meaningful changes, delegate bounded tasks to specialist subagents, and
verify results — with you in charge of the decisions.

Two entry points, one installer engine: **`asdx`** opens the packaged terminal
UI (TUI); **`asdx-core`** is the scriptable command-line interface (CLI) used
for preview, apply, backups, and restore. Legacy names `agentic-sdd` (CLI) and
`agentic-sdd-tui` (TUI) remain wrappers during the deprecation window.

[Quick start](#quick-start) · [Support and release scope](#support-and-release-scope) · [Where the skills go](#where-the-skills-go) · [The workflow](#the-workflow) · [Safety model](#safety-model) · [Explore further](#explore-further)

## Why Agentic SDD?

Writing code is getting faster. Knowing **what to build, why it works, and
whether it is ready** still takes engineering judgment.

Agentic SDD brings that judgment into AI-assisted work:

- **Agree on the outcome before the implementation.** Make requirements and
  trade-offs visible instead of letting the agent guess.
- **Keep changes reviewable.** Work in bounded batches with a summary of what
  changed and what was checked.
- **Skip unnecessary ceremony.** A small fix stays a small fix. A larger
  feature gets a connected specification, plan, and task list.
- **Delegate deliberately.** Scope subagent tasks, check their diffs and test
  results, and stop for human review between batches.
- **Learn in place.** Ask for explanations or debugging practice when a topic
  is unfamiliar. Teaching is optional.

> Learning first when developing competence. Delegation first when applying
> established competence. Verification always.

## Quick start

### 1. Install

**macOS with Homebrew:**

```sh
brew install --cask nawodyaishan/tap/agentic-sdd
asdx
```

`asdx` opens the TUI on Home. Enter (or `s`) opens Install & Manage, which
previews the exact changes and writes only after you confirm (default
**No**). The pack is included by default; `asdx --with-opencode=false`
installs core skills only.

**Linux and source checkouts** have no prebuilt package; build from source
(see [Support and release scope](#support-and-release-scope)):

```sh
git clone https://github.com/nawodyaishan/agentic-sdd.git
cd agentic-sdd
go build -trimpath -o bin/asdx-core ./cmd/agentic-sdd
bin/asdx-core preview
```

### 2. Preview before you install

```sh
asdx-core preview --home /tmp/test-home
asdx-core apply --home /tmp/test-home
```

`preview` writes nothing. `apply` backs up replaced skills, then installs.
Omit `--home` to target your real agent directories.

### 3. Run your first workflow

Follow the worked paths in [docs/getting-started.md](docs/getting-started.md):
a direct fix with no feature paperwork, a feature with one combined approval
and one batch, and how review differs from continuation. Each path lists
prerequisites, expected results, and cleanup.

## Support and release scope

This section is the compatibility pointer; it replaces any per-feature
changelog here. For what changed in a specific version, read the notes on the
[public releases page](https://github.com/nawodyaishan/agentic-sdd/releases)
and the specs under `specs/` in your checkout.

- **Reviewed releases** are cut as `v*` tags with the paired `asdx`/`asdx-core`
  archive and the Homebrew cask. The README badge above links to the releases
  page; check there for the notes that apply to your checkout.
- **macOS** installs through the Homebrew cask (`nawodyaishan/tap/agentic-sdd`).
  Formula-to-cask migration is not automatic; if you installed the old formula,
  uninstall it explicitly as described in
  [Updates and recovery](website/src/content/docs/how-to/updates-and-recovery.md).
- **Linux** is source-only: build `asdx-core` with Go 1.23 or newer, or the
  paired TUI with Go 1.26 or newer and Task. No prebuilt Linux package is
  published.
- **Windows** is unsupported and its behavior is unknown.
- **Preview-first applies everywhere.** `asdx-core preview` is the write-free
  check; `guide` and the TUI confirm (default **No**) before any write.
- No built-in uninstall command. Restore from a listed backup
  (`asdx-core backups`, `asdx-core restore apply BACKUP_ID`).

## Where the skills go

Core skills install into per-client skill directories; the orchestration pack
adds one skill plus agent profiles. The TUI offers client selection during
install; scripted runs install the fixed destination set.

| Destination | Path | Installed by |
|---|---|---|
| OpenCode and shared agents | `~/.agents/skills` | Core and pack |
| Codex | `~/.codex/skills` | Core and pack |
| Claude Code | `~/.claude/skills` | Core and pack |
| Antigravity CLI | `~/.gemini/antigravity-cli/skills` | Core and pack |
| OpenCode agent profiles | `~/.config/opencode/agents/` | Pack only |

### Where the files go

Backups and operation logs live under `~/.agentic-sdd/` (`backups/` and
`logs/`). The backup store path differs for embedded versus explicit `--repo`
sources; `asdx-core backups` lists what exists on your machine, newest first.
Each backup carries SHA-256 digests so `restore` previews exactly what would
change before writing.

## OpenCode Orchestration Pack (`--with-opencode`)

Opt in when one agent is not enough and you want bounded specialist help behind
a single orchestrator. The pack adds the `agentic-sdd-orchestrate` skill to the
four client directories and places agent profiles under
`~/.config/opencode/agents/`.

```sh
asdx --with-opencode=false   # core skills only
asdx-core preview --with-opencode
asdx-core apply --with-opencode
```

Profile names, permissions, and routing live in
[`opencode-pack/README.md`](opencode-pack/README.md) and
[`opencode-pack/routing.md`](opencode-pack/routing.md); the README does not
duplicate them. Two fleet adapters ship in the repository but are not installed
by the ordinary pack flow.

## The workflow

**Understand → Scope → Plan → Implement → Verify → Review**

For a feature that needs planning, your agent prepares three connected
documents for **one combined approval**:

- `spec.md` — what are we building, what is out of scope, how will we know it
  works?
- `plan.md` — how should we build it, and what risks or trade-offs matter?
- `tasks.md` — what do we do first, how do we check it, and where do we stop?

Then authorize one implementation batch. The agent works that batch, runs the
relevant checks, explains the result, and stops for your review.

**For a clear, small fix:** go straight to implementation and verification. No
feature paperwork required.

### Everyday prompts in your project

1. **Start or plan a feature (with combined human approval):**
   > Use `agentic-sdd-router`. Inspect this repository, ask at most the consequential questions you can't resolve from the code, and draft `spec.md`, `plan.md`, and `tasks.md` for my combined approval. Stop before implementing.
2. **Fix a small bug directly (no `specs/` paperwork):**
   > Use `agentic-sdd-implement` to fix this nil-pointer bug. Keep the diff minimal, run the focused test suite, and report what changed.

## Safety model

The installer **previews by default**, backs up existing skills and agent
profiles with SHA-256 digests before replacing them, and provides a
preview-first `restore`. Repeating an `apply` with unchanged content is a clean
no-op. It rejects symlinks and special files in managed targets.

```sh
asdx-core backups                   # list saved installations
asdx-core restore BACKUP_ID         # preview a restore
asdx-core restore apply BACKUP_ID   # restore skills and agent profiles
```

The guide and TUI are alternate front ends over the same engine. Neither
writes before an explicit confirmation (default **No**). The workflow is
guidance for your review process, not an isolation boundary: your agent
client's permissions, execution environment, and your own review remain
essential.

## Explore further

- [Getting started](docs/getting-started.md) — first-use walkthrough and
  worked examples for the supported core.
- [Full usage and orchestration guide](docs/orchestration-guide.md) — CLI,
  orchestration, specialist, and MCP usage.
- [Installer reference](docs/installer-reference.md) — commands, backups,
  restore, and file behavior.
- [Guided CLI journeys](docs/guide-flows.md) — screens, prompts, and trace
  events for `asdx-core guide`.
- [OpenCode pack reference](opencode-pack/README.md) and
  [model routing policy](opencode-pack/routing.md).
- [Contributing](CONTRIBUTING.md) and the
  [public releases page](https://github.com/nawodyaishan/agentic-sdd/releases).
- Repository source on GitHub:
  [github.com/nawodyaishan/agentic-sdd](https://github.com/nawodyaishan/agentic-sdd).

Open source under the [MIT License](LICENSE).
