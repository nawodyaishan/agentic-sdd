---
title: Supported clients and destinations
description: Authoritative matrix of skill destinations, OpenCode pack paths, client selection, and verified versus portable support boundaries.
---

This page is the single source of truth for where Agentic-SDD writes files, how client selection differs between the CLI and TUI, and what is verified versus merely portable.

## Skill and profile destinations

Agentic-SDD manages only `agentic-sdd-*` skill directories and the shipped OpenCode pack `.md` profile files. Unrelated skills and user configuration files (`opencode.json`, `AGENTS.md`, or custom local agents outside the pack list) are left untouched.

Treat every destination above as a separate per-client skills folder under the resolved `--home`: the Claude Code folder is `~/.claude/skills`, the Codex folder is `~/.codex/skills`, the Antigravity CLI folder is `~/.gemini/antigravity-cli/skills`, and the shared `agents` folder for OpenCode and other agents is `~/.agents/skills`.

| Target ID | Client / surface | Destination under `--home` | Default core | With `--with-opencode` |
|---|---|---|---|---|
| `agents` | OpenCode and shared agents | `<home>/.agents/skills` | 18 core skills | + `agentic-sdd-orchestrate` |
| `codex` | Codex | `<home>/.codex/skills` | 18 core skills | + `agentic-sdd-orchestrate` |
| `claude` | Claude Code | `<home>/.claude/skills` | 18 core skills | + `agentic-sdd-orchestrate` |
| `agy` | Antigravity CLI | `<home>/.gemini/antigravity-cli/skills` | 18 core skills | + `agentic-sdd-orchestrate` |
| `opencode-agents` | OpenCode v2 agent profiles | `<home>/.config/opencode/agents` | Not touched | 14 `.md` profiles (+ 4 presets with `--opencode-free-first`) |

Destination rules enforced by `internal/skillsync/sync.go` and `internal/skillsync/clients.go`:

- **No duplicate OpenCode skill directory:** `agentic-sdd-orchestrate` installs into the selected client skill directories (`<home>/.agents/skills`, etc.), never into `<home>/.config/opencode/skills`, so skill IDs do not collide and sibling links such as `../agentic-sdd-router/references/workflow-policy.md` resolve.
- **Explicit `--home` derivation:** `<home>/.config/opencode/agents` is always derived from the resolved `--home` directory, not `XDG_CONFIG_HOME`.
- **Two fleet adapters excluded:** The 2 profiles in `opencode-pack/adapters/` (`sdd-plan-9r.md`, `sdd-spec-9r.md`) are fleet-render sources only and are never installed by `--with-opencode`.

For backup roots, manifest formats (`1`–`3` and `4`), and SHA-256 verification details, see [Files and backups](../reference/files-and-backups.md).

## Client selection: TUI only, no CLI `--clients` flag

The CLI and TUI share the same installation engine (`internal/skillsync`), but client filtering is exposed only in the terminal UI:

- **CLI (`asdx-core`):** There is **no `--clients` flag** on `asdx-core preview` or `asdx-core apply`. Running the CLI targets **all four** skill destinations (`agents`, `codex`, `claude`, `agy`).
- **TUI (`asdx` / `asdx-core tui`):** The **Install & Manage** screen lets you toggle individual client IDs (`agents`, `codex`, `claude`, `agy`), defaulting to all four selected and requiring at least one. See the [TUI reference](../reference/tui.md).

See [Manage skills](../how-to/skills.md) for practical install and update steps.

## Core vs optional OpenCode pack

| Layer | Source in repo | What it adds | Client scope |
|---|---|---|---|
| **Core skills** | `skills-files/` | 18 portable `SKILL.md` directories | All four skill destinations |
| **Optional OpenCode pack** | `opencode-pack/skills/`, `opencode-pack/agents/` | `agentic-sdd-orchestrate` + 14 OpenCode v2 `.md` profiles | Skill destinations + OpenCode `~/.config/opencode/agents` |
| **Optional Free-first presets** | `opencode-pack/presets/free-first/` | 4 implement/test profile presets (`sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, `sdd-test-implement-2`) | OpenCode `~/.config/opencode/agents` (requires `--with-opencode`) |

- OpenCode profiles, MCP config helpers (`asdx-core mcp`), and fleet tools (`asdx-core fleet`) apply only to OpenCode-oriented configurations; they do not configure Codex, Claude Code, or Antigravity CLI agents.
- Bare `asdx-core` defaults to **core-only** read-only preview. Bare `asdx` opens the TUI with `--with-opencode=true` by default (`--with-opencode=false` disables the pack). See [Choose your setup](../getting-started/choose-setup.md).

## Verified support vs portable instructions

**Portable skill files do not establish cross-client parity.** Agentic-SDD draws a strict line between what is verified in tests and what depends on an external client or operating system:

| Surface | Actual status | What is verified vs unverified |
|---|---|---|
| **Skill files and installer transactions** | Verified in Go unit/static tests (`internal/skillsync`) | Directory layout, `SKILL.md` frontmatter and relative links, symlink/special-file refusal, atomic staging, SHA-256 backup/restore manifests, and adapter exclusion are tested offline on temporary homes. |
| **Codex, Claude Code, Antigravity CLI, shared `~/.agents/skills`** | Portable Markdown skills only | The installer writes standard `SKILL.md` directories to those paths. Live behavior—how each client discovers skills, honors `disable-model-invocation: true`, enforces permissions, or handles models—is **not** runtime-certified. |
| **OpenCode runtime integration** | Historical probes uncertified | `integrations/opencode/supported-version.md` records offline/source-level checks against `opencode` `2.2.0`/`2.2.5`; live native runtime certification remains blocked/uncertified, and experimental `opencode`/`opencode-harness` CLI commands exit `2` in default core builds (`cmd/agentic-sdd/opencode_disabled.go`). |
| **macOS release (`v0.11.0` / Homebrew cask)** | Public metadata verified only | Release tag `v0.11.0` (`agentic-sdd_0.11.0_darwin_all.tar.gz`) and cask metadata are publicly observed; archive binary contents, runtime identity, and live macOS installation are unverified here. |
| **Linux** | Source-only | No prebuilt Linux release archive ships; build from source with Go (`cmd/agentic-sdd` and `experiments/tui`). |
| **Windows** | Unsupported / unknown | Not tested or packaged. |

## Next steps

- Ready to pick an installation mode? Continue to [Choose your setup](../getting-started/choose-setup.md).
- Ready to install or build from source? Go to [Installation](../getting-started/installation.md).
