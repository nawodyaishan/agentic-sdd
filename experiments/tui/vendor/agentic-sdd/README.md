# Agentic SDD

**Build software with AI. Stay the engineer.**

[![Release](https://img.shields.io/github/v/release/nawodyaishan/agentic-sdd)](https://github.com/nawodyaishan/agentic-sdd/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/nawodyaishan/agentic-sdd/ci.yml?label=CI)](https://github.com/nawodyaishan/agentic-sdd/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

Agentic SDD is an open-source toolkit of **18 portable core engineering
skills** plus an optional OpenCode pack, delivered by a preview-first Go CLI
(`asdx-core`) and a terminal UI companion (`asdx`). It helps one main coding
agent scope work, keep human approval explicit, verify with real checks, and
learn as you build. Small fixes skip feature paperwork; larger features get a
connected spec, plan, and task list for one combined approval.

**[Documentation](https://nawodyaishan.github.io/agentic-sdd/)** — set up
Agentic SDD, follow a tutorial, and look up commands, references, and known
limits. If the site is unavailable, use the
[documentation source index](website/src/content/docs/index.md) instead.

## Why Agentic SDD?

Knowing **what to build, why it works, and whether it is ready** still takes
engineering judgment.

- **Agree on the outcome first.** Make requirements and trade-offs visible.
- **Keep changes reviewable.** Work in bounded batches and summarize checks.
- **Skip unnecessary ceremony.** Small fixes stay small; larger features get
  a spec, plan, and tasks.
- **Delegate deliberately.** Scope subagent tasks, check diffs and tests, and
  stop for review.
- **Learn in place.** Ask for explanations or debugging practice when useful;
  teaching is optional.

## Quick start

### 1. Get the binaries

Binary acquisition is separate from installing skills. None of these commands
writes to your client skill directories or `~/.agentic-sdd/`.

**macOS (Homebrew cask).** Requires Homebrew. The cask puts `asdx-core` and
`asdx` on `PATH`, fetches the release archive over the network, and writes
inside your Homebrew prefix. The `v0.11.0` tag and cask metadata are verified; a
live macOS install is not.

```sh
brew install --cask nawodyaishan/tap/agentic-sdd
asdx-core version
```

**Linux or source checkout (CLI only).** Requires **Go 1.23 or newer** and
`git`. The clone uses the network, the build writes `bin/asdx-core` plus Go
build caches, and the root module has no external dependencies.

```sh
git clone https://github.com/nawodyaishan/agentic-sdd.git
cd agentic-sdd
go build -trimpath -o bin/asdx-core ./cmd/agentic-sdd
./bin/asdx-core version
```

The source build does **not** add `bin/` to `PATH`; use `./bin/asdx-core` in
place of `asdx-core` below. The paired `asdx` TUI needs Go 1.26+, Task, and
`sha256sum` or `shasum`; see [Installation](https://nawodyaishan.github.io/agentic-sdd/getting-started/installation/).

### 2. Preview core skills in a disposable home

`asdx-core preview` is the core-only, scriptable path: no OpenCode pack, no
provider credentials, no model calls. Point `--home` at an empty directory you
own so the plan cannot touch your real client directories.

```sh
preview_home=$(mktemp -d "${TMPDIR:-/tmp}/asdx-preview.XXXXXX")
asdx-core preview --home "$preview_home"
rmdir "$preview_home"
```

If you built from source, run the command above with `./bin/asdx-core` instead.
Expected: **18 core skills** for each of the four client destinations, a
`Preview only.` notice, and an optional TTY summary; nothing is created inside
`$preview_home`, so `rmdir` succeeds only while it is empty. **Stop here** —
nothing was installed. To install into your own client directories, use the
preview-and-confirm recipe in [Install and manage skills](https://nawodyaishan.github.io/agentic-sdd/how-to/skills/)
or [First five minutes](https://nawodyaishan.github.io/agentic-sdd/getting-started/first-five-minutes/).

### 3. Run your first workflow

Decide between CLI, TUI, core skills, and the optional pack in
[Choose your setup](https://nawodyaishan.github.io/agentic-sdd/getting-started/choose-setup/),
then follow one website tutorial:

- [Direct fix](https://nawodyaishan.github.io/agentic-sdd/tutorials/direct-fix/) —
  a small bug fix with no `specs/` paperwork.
- [First feature](https://nawodyaishan.github.io/agentic-sdd/tutorials/first-feature/) —
  `spec.md`, `plan.md`, and `tasks.md` for one combined approval, then one
  authorized batch.
- [Research and debugging](https://nawodyaishan.github.io/agentic-sdd/tutorials/research-and-debugging/)
  or [Specialists](https://nawodyaishan.github.io/agentic-sdd/tutorials/specialists/) —
  optional depth for investigation or bounded roles.

## Support and release scope

These instructions describe the current checkout (`dev`, `unreleased`). A
release tag is not a support guarantee: for `v0.11.0`, the tag, release
metadata, and cask declaration are verified, while archive contents, binary
execution, and a live macOS install are not. The documentation site was observed
reachable on 2026-11-16; that is not full-site, accessibility, release, or
runtime verification.

- **macOS** is the only prebuilt path (Homebrew cask
  `nawodyaishan/tap/agentic-sdd`); formula-to-cask migration is manual — see
  [Updates and recovery](https://nawodyaishan.github.io/agentic-sdd/how-to/updates-and-recovery/).
- **Linux** is source-only: CLI with Go 1.23+, paired TUI with Go 1.26+ and
  Task. No prebuilt package exists.
- **Windows** is unsupported and behavior is unknown.
- **Clients** receive installed files at four destinations; installation does
  not establish identical behavior across clients, models, or providers.
- **No built-in uninstall.** `asdx-core backups` lists; `asdx-core restore
  BACKUP_ID` previews.

See [Compatibility and support](https://nawodyaishan.github.io/agentic-sdd/releases/compatibility/),
[Migration](https://nawodyaishan.github.io/agentic-sdd/releases/migration/), and
[Known limitations](https://nawodyaishan.github.io/agentic-sdd/operations/known-limitations/).

## Where the skills go

Core skills install into `~/.agents/skills` (OpenCode and shared agents),
`~/.codex/skills`, `~/.claude/skills`, and `~/.gemini/antigravity-cli/skills`;
the pack also writes profiles under `~/.config/opencode/agents/`. The TUI
selects clients; scripted runs use the fixed set. See
[Clients and destinations](https://nawodyaishan.github.io/agentic-sdd/introduction/clients/)
and the [Skills reference](https://nawodyaishan.github.io/agentic-sdd/reference/skills/).

## Where the files go

Operation logs live under `~/.agentic-sdd/logs/`. Backups live under
`~/.agentic-sdd/backups/` for the embedded source and the checkout's `backups/`
directory for explicit `--repo`; `asdx-core backups` lists newest first, and
SHA-256 digests let `restore` preview exactly what would change.

Execution commands keep private scratch under `.agentic-sdd/runs/`; a run,
checkpoint, or review package is an observation, not approval, batch state,
verification, reviewer isolation, or a sandbox. See [Files and backups](https://nawodyaishan.github.io/agentic-sdd/reference/files-and-backups/) and
[Execution and review evidence](https://nawodyaishan.github.io/agentic-sdd/how-to/execution/).

## OpenCode Orchestration Pack (`--with-opencode`)

Opt in when one agent is not enough and you want bounded specialist help behind
a single orchestrator. The pack adds the `agentic-sdd-orchestrate` skill and 14
agent profiles. `--opencode-free-first` applies Free-first model-ladder
overrides to four existing implement/test profiles rather than adding new
profiles.

Core-only is the CLI default (`asdx-core preview` without `--with-opencode`);
the `asdx` TUI enables the pack by default, and `asdx --with-opencode=false`
selects core only. Profile names, permissions, and routing are owned by
[Orchestration](https://nawodyaishan.github.io/agentic-sdd/how-to/orchestration/),
[Profiles](https://nawodyaishan.github.io/agentic-sdd/reference/profiles/), and
[Fleet](https://nawodyaishan.github.io/agentic-sdd/how-to/fleet/); this README
does not duplicate them. Model IDs are configuration, not availability, quota,
cost, or behavior guarantees.

The pack's real-world companion is [`agentic-sdd-fleet`](https://github.com/nawodyaishan/agentic-sdd-fleet),
an external personal repository (observed 2026-11-16): a `schemaVersion: 1`
`fleet.json` with 27 instances over 20 canonical sources, an `agents/` snapshot,
and a `manifest.json` pinning one pack commit. Configuration only, no
credentials, not part of Agentic-SDD or upstream `opencode-pack/`; that pin can
drift, so preview under a disposable `--home` or read-only `asdx --fleet-config`.

## The workflow

**Understand → Scope → Plan → Implement → Verify → Review**

For a feature that needs planning, your agent prepares three connected
documents for **one combined approval**:

- `spec.md` — what are we building, what is out of scope, how will we know it works?
- `plan.md` — how should we build it, and what risks or trade-offs matter?
- `tasks.md` — what do we do first, how do we check it, and where do we stop?

Then authorize one implementation batch. The agent works that batch, runs the
relevant checks, explains the result, and stops for your review. **For a clear,
small fix,** go straight to implementation and verification; no feature
paperwork is required. The [First feature](https://nawodyaishan.github.io/agentic-sdd/tutorials/first-feature/)
tutorial owns the full loop.

### Everyday prompts in your project

1. **Start or plan a feature (with combined human approval):** Use `agentic-sdd-router`. Inspect this repository, ask at most the consequential questions you can't resolve from the code, and draft `spec.md`, `plan.md`, and `tasks.md` for my combined approval. Stop before implementing.
2. **Fix a small bug directly (no `specs/` paperwork):** Use `agentic-sdd-implement` to fix this nil-pointer bug. Keep the diff minimal, run the focused test suite, and report what changed.

## Safety model

The installer **previews by default**, backs up existing skills and agent
profiles with SHA-256 digests before replacing them, and provides a
preview-first restore. Repeating an `apply` unchanged is a clean no-op. It
rejects symlinks and special files in managed targets.

```sh
asdx-core backups                   # list saved installations
asdx-core restore BACKUP_ID         # preview a restore; writes nothing
```

Effects are command-scoped. `asdx-core preview` writes nothing to managed
destinations and creates no installer backups or operation logs (an explicitly
enabled trace is separate); `apply` writes backups, selected skills or profiles,
and one operation log, and applying a restore writes destinations and a log. The
`asdx` TUI makes one bounded read-only HTTPS update check per session unless
`AGENTIC_SDD_NO_UPDATE_CHECK=1` is set. Fleet and execution are not globally
read-only: `fleet plan --output` writes the named file, and execution writes
private scratch. `run check` runs the command you supply with your own user
authority and side effects; it is not a sandbox. `--home` is not an OS sandbox.

The guide and TUI are alternate front ends over the same engine; neither writes
before explicit confirmation (default **No**). Profile permissions and workflow
guidance are not containment, credential boundaries, or proof that a model
follows instructions. MCP configuration proves neither authentication nor
connectivity. Read [Trust boundaries](https://nawodyaishan.github.io/agentic-sdd/security/trust/),
[MCP configuration](https://nawodyaishan.github.io/agentic-sdd/how-to/mcp/), and
[Updates and recovery](https://nawodyaishan.github.io/agentic-sdd/how-to/updates-and-recovery/).

## Explore further

- [Documentation website](https://nawodyaishan.github.io/agentic-sdd/) — primary
  current reference.
- [Core concepts](https://nawodyaishan.github.io/agentic-sdd/introduction/concepts/) —
  skills, profiles, features, direct fixes, batches, and consent; see the
  [CLI reference](https://nawodyaishan.github.io/agentic-sdd/reference/cli/) and
  [TUI reference](https://nawodyaishan.github.io/agentic-sdd/reference/tui/) for commands.
- [Architecture overview](https://nawodyaishan.github.io/agentic-sdd/architecture/overview/),
  [Testing](https://nawodyaishan.github.io/agentic-sdd/contributing/testing/), and
  [Development](https://nawodyaishan.github.io/agentic-sdd/contributing/development/) —
  design, contribution workflow, and checks.
- [GitHub repository](https://github.com/nawodyaishan/agentic-sdd),
  [public releases](https://github.com/nawodyaishan/agentic-sdd/releases),
  [CONTRIBUTING.md](CONTRIBUTING.md), and the [MIT License](LICENSE).
- [Real-world fleet example](https://github.com/nawodyaishan/agentic-sdd-fleet) — external, configuration only, no credentials.

**Retained historical worked example:** the [original `docs/getting-started.md`](docs/getting-started.md) — a pre-website walkthrough of a direct fix and a feature, kept for continuity, not current onboarding.
