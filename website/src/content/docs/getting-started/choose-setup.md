---
title: Choose your setup
description: Decide between the asdx-core CLI and asdx TUI, core skills versus the optional OpenCode pack, and check model/provider prerequisites.
---

Before installing, make two quick choices: **how you want to run the installer** (CLI vs TUI) and **which layer you need** (core skills only vs the optional OpenCode orchestration pack).

For exact destination paths, client selection rules, and platform support status, see [Supported clients and destinations](../introduction/clients.md).

## 1. CLI (`asdx-core`) vs TUI (`asdx`)

The two canonical binaries share the same preview-first installer engine, **but their bare defaults differ**:

| Command | Interface | Default pack behavior | Client filtering | Best for |
|---|---|---|---|---|
| `asdx-core` | Read-only CLI (`preview`) | **Core skills only** (`--with-opencode` is off) | Always targets all 4 client directories | Scripts, CI, pipes, quick previews, zero-dependency Go builds |
| `asdx` | Interactive TUI | **OpenCode pack on** (`--with-opencode=true`) | Interactive selection on **Install & Manage** | Interactive setup, choosing specific clients, browsing skills and profiles |
| `asdx-core guide` | Numbered terminal wizard | Prompts interactively (core or pack) | Targets all 4 client directories | Terminal users without the `asdx` TUI companion |

Key entrypoint facts from `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/tui.go`, and `experiments/tui/cmd/tui-companion/main.go`:

- **Bare `asdx-core` never opens the TUI** and writes nothing; it runs `asdx-core preview` with core skills only.
- **Bare `asdx` opens the TUI with `--with-opencode=true`.** To use the TUI for core skills only, run `asdx --with-opencode=false`.
- **`asdx-core tui`** launches the verified sibling `asdx` binary from the same installation (it needs a real stdin/stdout terminal and fails closed if `asdx` is missing or mismatched).
- **Legacy wrappers:** `agentic-sdd` forwards to `asdx-core` (never the TUI) and `agentic-sdd-tui` forwards to `asdx` during the deprecation window.

See the [CLI reference](../reference/cli.md) and [TUI reference](../reference/tui.md) for all flags, routes, and exit codes.

## 2. Core skills only vs optional OpenCode pack

| Setup choice | What gets installed | Choose this if… |
|---|---|---|
| **Core skills only** | 18 portable skills (`skills-files/`) | You use Codex, Claude Code, Antigravity CLI, or OpenCode with **one main coding agent** and want minimal setup. |
| **Core + OpenCode pack** (`--with-opencode`) | 18 core skills + `agentic-sdd-orchestrate` + 14 OpenCode v2 profiles (`opencode-pack/agents/`) | You use OpenCode and want optional subagent roles (review, diagnosis, test design, research, implementation). |
| **Core + OpenCode pack + Free-first presets** (`--with-opencode --opencode-free-first`) | Everything above + 4 Free-first implement/test preset profiles (`opencode-pack/presets/free-first/`) | You want the pack and prefer the 4 routine implement/test profiles preset to try `opencode/deepseek-v4.1-flash-free` first. |

- You can start with **core skills only** and add `--with-opencode` later at any time.
- For local `--repo` checkouts, supported model overrides, and the rules preserving unrelated files, see [Customization and local sources](../how-to/customization.md).
- If you do not use OpenCode, stick to **core skills only**—OpenCode agent profiles do not configure Codex, Claude Code, or Antigravity CLI.
- Remember the CLI vs TUI default:
  - CLI core-only preview: `asdx-core preview`
  - CLI with pack preview: `asdx-core preview --with-opencode`
  - TUI core-only: `asdx --with-opencode=false`
  - TUI with pack (default): `asdx`

## 3. Model and provider prerequisites

- **Installing or previewing requires no API keys:** Neither `asdx-core` nor `asdx` calls an LLM, reads provider tokens, or requires credentials to preview, install, back up, or restore skills.
- **Core skills use your existing agent session:** Whatever model and provider your coding client already runs is sufficient; Agentic-SDD does not lock core skills to a specific provider.
- **Pack profiles are configuration, not access guarantees:** Most of the 14 OpenCode profiles pin specific provider/model IDs (for example `openai/...`, `9router/ag/...`, `opencode-go/...`, or `opencode/...-free`), while `sdd-orchestrator` inherits your active session model. A pinned model ID—including `--opencode-free-first`—does **not** guarantee that your OpenCode setup has credentials, quota, routing, or availability for that model.
- **Safe offline trial:** You can verify your setup right away using an isolated temporary `--home` directory with zero provider credentials and zero real-home writes.

## Next steps

- Install the binary or build from source in [Installation](installation.md).
- Run a safe, write-free preview in [First five minutes](first-five-minutes.md).
- Jump straight to [Manage skills](../how-to/skills.md) or [Updates and recovery](../how-to/updates-and-recovery.md) if you already have the binaries.
