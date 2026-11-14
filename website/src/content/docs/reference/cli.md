---
title: CLI reference
description: Command families, flags, defaults, exit codes, and output contracts for asdx-core, asdx, and the guide, fleet, MCP, and execution subcommands.
---

This page combines curated orientation prose with source-derived command tables and captured help output from `asdx-core`.
See [TUI reference](./tui.md) for the interactive `asdx` terminal interface and [Files and backups](./files-and-backups.md) for storage roots and manifest formats.

## Command family overview

`asdx-core` groups commands into four operational areas, each with its own `flag.FlagSet` rather than universal global flags:

- **Core installer and recovery**: `preview` (the default when `asdx-core` runs bare), `apply`, `backups`, `logs`, `restore`, `version`, and `help`. These manage portable skills under client destinations (`--home`) from embedded assets or an explicit checkout (`--repo`).
- **Interactive entrypoints**: `guide` (numbered terminal wizard) and `tui` (compatibility launcher for the sibling `asdx` binary).
- **Subsystem commands**: `fleet` (personal `fleet.json` agent plans and sidecar backups), `mcp` (existing global OpenCode MCP configuration inspection, plans, and entry-level restore), and the execution family (`run`, `task`, `review`) for private run scratch under `.agentic-sdd/runs/`. See [Fleet reference](./fleet.md), [MCP reference](./mcp.md), and [Execution reference](./execution.md).
- **Excluded experimental commands**: `opencode` and `opencode-harness` require the `-tags agentic_sdd_experimental` build tag and exit `2` in default core builds.

## Interactive entrypoints: `guide`, `tui`, and `asdx`

- **Bare `asdx-core` vs `asdx`**: running `asdx-core` with no command always runs a read-only skill `preview` and never opens the full terminal UI. Run `asdx` directly (or `asdx-core tui`) for the Bubble Tea terminal UI.
- **`asdx-core guide` (numbered wizard)**: a stdin/stdout numbered-prompt flow over skill `preview`, `apply`, `restore`, and `logs`. It accepts only `--repo` and `--home`, refuses non-interactive runs (`exit 2`, while honoring `AGENTIC_SDD_FORCE_INTERACTIVE=1`), and on interactive terminal stdout (`StdoutTTY`) also exposes a GitHub CLI release check (choice `5`) and a read-only skill/profile explorer (choice `6`). It defaults to core skills unless you confirm the OpenCode pack prompts inside the wizard.
- **`asdx-core tui` (companion launcher)**: verifies the same-install `asdx` binary over a bounded `--protocol-info` handshake (protocol revision `2`, matching revision, commit, and source digest) without searching `PATH` or invoking a shell. Unlike bare `asdx-core` or `apply`, `asdx` and `asdx-core tui` default `--with-opencode` to `true` (pass `--with-opencode=false` for core skills only), require real stdin and stdout TTYs, and reject `AGENTIC_SDD_FORCE_INTERACTIVE=1` with exit `2`.
- **Legacy wrappers**: during the deprecation window, `agentic-sdd` forwards to same-directory `asdx-core` and `agentic-sdd-tui` forwards to same-directory `asdx` after a stderr-only notice (bare `--help`, `-h`, and `--version` are answered locally).

## Exit-code and structured-output caveats

Exit codes are family-specific rather than shared across all commands:

- **`0`**: success, a read-only preview, or a produced diagnostic/status report — including `fleet doctor` when model routes are unavailable, `mcp status`/`doctor` when prerequisites are missing, or a real `mcp` no-op.
- **`1`**: runtime or I/O error, missing or unreadable log (`logs`), missing or mismatched `asdx` sibling (`tui`), missing run (`run status`), held lock, stale plan, or unmanaged conflict (`fleet`, `mcp`), or a failing check command (`run check`).
- **`2`**: usage, flag, or unknown-command/topic error, non-TTY refusal (`guide`, `tui`, `restore` without ID), disabled experimental command (`opencode`, `opencode-harness`), deferred `mcp doctor --network` / `--connect`, invalid configuration, or execution scope refusal (for example `task brief` on a `--direct` run).
- **`3`**: `review package` partial package (`execution.ErrPartial`) when explicit omissions occur; the manifest output is still written.
- **`124`**: `run check` timed out before the wrapped command exited.

Structured JSON output is also subsystem-specific:

- `--json` belongs to the `mcp` family (`status`, `doctor`, `preview`, `apply`, `restore list|preview|apply`). Flag/usage errors still emit plain text on `stderr`, and `mcp restore apply` still prompts `[y/N]` interactively.
- `--format json` (default `text`) belongs to the execution family (`run`, `task`, `review`).
- Core installer commands (`preview`, `apply`, `backups`, `logs`, `restore`, `guide`, `version`) and `fleet` subcommands have no `--json` flag; `fleet plan --output PATH` writes a canonical JSON plan file, `asdx --protocol-info` emits private pairing JSON, and `AGENTIC_SDD_TRACE` appends `0600` JSONL trace events.

## Related reference pages

- [TUI reference](./tui.md) — `asdx` routes, keys, child screens, and update/browser effects.
- [Files and backups](./files-and-backups.md) — destination paths, backup roots, and manifest formats.
- [Configuration reference](./configuration.md) — per-subsystem flags, files, and environment precedence.

<!-- generated:reference/cli:start -->
Facts below were captured by running the candidate `asdx-core` binary at commit `2fc31e9f9c4d019e2b9f3b4f004978028808c74b` (version `dev`, built `2026-11-14T00:00:00Z` with `go1.27.1`). The expected inspected-revision commit `2fc31e9f9c4d019e2b9f3b4f004978028808c74b` was verified before capture.

## Entrypoints and compatibility wrappers

- Canonical commands: `asdx-core` (CLI) and `asdx` (TUI companion), defined in `internal/buildinfo/buildinfo.go` and `cmd/agentic-sdd/main.go`.
- Legacy compatibility wrappers (one deprecation window): `agentic-sdd` forwards to same-directory `asdx-core` after a stderr notice (only exact `--help`, `-h`, and `--version` are answered locally); `agentic-sdd-tui` wraps `asdx`.
- Bare `asdx-core` defaults to read-only `preview` and never opens the terminal UI (`cmd/agentic-sdd/main.go`).
- Experimental commands `opencode` and `opencode-harness` require `-tags agentic_sdd_experimental` and exit `2` in default core builds (`cmd/agentic-sdd/opencode_disabled.go`).

## Core installer, launcher, and utility commands

Flags are command-specific (standard-library `flag.FlagSet` per handler, not universal global flags):

| Command | Forms / subcommands | Accepted flags | Write behavior & exit notes | Source |
|---|---|---|---|---|
| `preview` (default) | `asdx-core [preview]` | `--repo PATH`, `--home PATH`, `--with-opencode`, `--opencode-free-first`, `--apply` (default form only), `--version` (root flag), `-h`, `--help` | Read-only. `--version` is a root flag: it is accepted in the default, `preview`, and `apply` forms and short-circuits to version output (`0`); it is not bare-only. `--opencode-free-first` requires `--with-opencode` (`2`); `--apply` combined with an explicit `preview`/`apply` command (`2`); unknown flag/argument (`2`). Exits `0`, `1` (runtime/I/O error), or `2` (usage/flag error). | `cmd/agentic-sdd/main.go` |
| `apply` | `asdx-core apply` | `--repo PATH`, `--home PATH`, `--with-opencode`, `--opencode-free-first`, `--version` (root flag), `-h`, `--help` | Writes backups + skills + log under `<home>/.agentic-sdd/logs`. Exits `0` success, `1` runtime error, `2` usage/flag error. | `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/logs.go` |
| `version` | `asdx-core version` | None (extra args including `--help` exit `2`; use `asdx-core help version`) | Read-only. Prints version, commit, date, and Go version (`0`). | `cmd/agentic-sdd/main.go` |
| `backups` | `asdx-core backups` | `--repo PATH`, `--home PATH`, `-h`, `--help` | Read-only; identical listing to `restore list`. Exits `0`, `1`, or `2`. | `cmd/agentic-sdd/main.go` |
| `logs` | `asdx-core logs` | `--home PATH`, `-h`, `--help` | Read-only; prints latest apply/restore log. Exits `1` if missing/unreadable, `2` usage. | `cmd/agentic-sdd/logs.go` |
| `restore` | `asdx-core restore [ID]` / `restore list\|preview\|apply [ID]` | `--repo PATH`, `--home PATH`, `--apply` (bare form only; rejected on sub-words), `-h`, `--help` | Read-only without `--apply`/`apply`. Interactive mode prompts when no `ID`; non-TTY without `ID` exits `2`. | `cmd/agentic-sdd/main.go` |
| `guide` | `asdx-core guide` | `--repo PATH`, `--home PATH`, `-h`, `--help` | Interactive numbered wizard; refuses non-TTY with exit `2`. Writes only after confirmation. | `cmd/agentic-sdd/guide.go` |
| `tui` | `asdx-core tui` | `--home PATH`, `--repo PATH`, `--fleet-config PATH`, `--with-opencode` (default `true`, `--with-opencode=false` to disable), `--opencode-free-first`, `-h`, `--help` | Requires real stdin+stdout TTY and rejects `AGENTIC_SDD_FORCE_INTERACTIVE=1` (`2`). Missing/mismatched `asdx` exits `1`. | `cmd/agentic-sdd/tui.go` |
| `help` | `asdx-core help [topic]` | Topics: `preview`, `apply`, `backups`, `logs`, `restore`, `guide`, `version`, `tui`, `orchestrate`, `fleet`, `mcp`, `run`, `task`, `review`; experimental-build `opencode` and `opencode-harness` are accepted and print the unavailable note in default builds | Read-only (`0`); unknown topic or extra args exits `2`. `help orchestrate`, `help fleet`, `help mcp` print dedicated blocks without repeating root usage; the default-build `help opencode`/`help opencode-harness` topics describe commands that require `-tags agentic_sdd_experimental`. | `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/helptext.go` |

## `fleet` family

Exit codes (`cmd/agentic-sdd/fleet.go`, `cmd/agentic-sdd/fleet_plan.go`, `cmd/agentic-sdd/fleet_doctor.go`): `0` success/report produced, `1` runtime error / lock held / stale plan / unmanaged conflict, `2` usage / flag / invalid configuration / malformed plan / invalid backup ID.

| Subcommand | Accepted flags | Write & contract notes | Source |
|---|---|---|---|
| `fleet status` | `--config PATH` (required), `--home PATH`, `--repo PATH` | Read-only, offline inspection of `fleet.json` and installed targets. | `cmd/agentic-sdd/fleet.go` |
| `fleet plan` | `--config PATH` (required), `--output PATH`, `--adopt`, `--home PATH`, `--repo PATH` | Read-only on destinations; `--output PATH` writes the canonical plan JSON file. Exits `1` when plan has conflicts. | `cmd/agentic-sdd/fleet_plan.go` |
| `fleet apply` | `--plan PATH` (required), `--home PATH` | Applies a saved plan under operation lock. | `cmd/agentic-sdd/fleet.go` |
| `fleet restore` | `--backup ID` (required), `--home PATH` | Flag form only; restores managed agents from backup ID. | `cmd/agentic-sdd/fleet.go` |
| `fleet doctor` | `--opencode-bin PATH`, `--config PATH` | Offline model-route diagnostic; exits `0` when report is produced even if routes are unavailable (`2` on invalid config). | `cmd/agentic-sdd/fleet_doctor.go` |

## `mcp` family (including incomplete top-level help note)

> **Known top-level help discrepancy (`cmd/agentic-sdd/main.go` vs `cmd/agentic-sdd/mcp.go`, `cmd/agentic-sdd/mcp_write.go`):** Top-level `asdx-core --help` summarizes `mcp` as read-only (`status|doctor|preview`) and omits `apply` and `restore`. Actual `mcp help` and Go dispatch implement `status`, `doctor`, `preview`, `apply`, and entry-level `restore list|preview|apply`.

Exit codes (`cmd/agentic-sdd/mcp.go`, `cmd/agentic-sdd/mcp_write.go`): `0` success / produced report / real no-op, `1` stale plan / conflict / unsupported runtime / rollback / engine or I/O error, `2` usage / flag / validation / unsafe target / unknown subcommand / deferred `--network` or `--connect`.

| Subcommand | Accepted flags | Write & contract notes | Source |
|---|---|---|---|
| `mcp` / `mcp status` | `--home PATH`, `--config PATH`, `--project PATH`, `--json` | Read-only, offline; default when no subcommand or a flag is passed after `mcp`. | `cmd/agentic-sdd/mcp.go` |
| `mcp doctor` | `--home PATH`, `--config PATH`, `--project PATH`, `--opencode-bin PATH`, `--codegraph-bin PATH`, `--json` | Offline diagnostics; `--network` and `--connect` are explicitly rejected with exit `2`. | `cmd/agentic-sdd/mcp.go` |
| `mcp preview` | `--servers NAME,...`, `--auth SERVER=MODE`, `--home PATH`, `--config PATH`, `--json` | Read-only structured plan + digest/token (`context7=oauth|api-key`, `exa=keyless|oauth|api-key`). Exits `1` if `Conflicts` is non-empty. | `cmd/agentic-sdd/mcp.go` |
| `mcp apply` | `--expect-config SHA256` (required), `--opencode-bin PATH`, `--servers NAME,...`, `--auth SERVER=MODE`, `--home PATH`, `--config PATH`, `--json` | Writes managed entries into existing global OpenCode config only when digest matches and local runtime probe succeeds; never creates missing config. | `cmd/agentic-sdd/mcp_write.go` |
| `mcp restore list` | `--home PATH`, `--config PATH` (parsed, unused by list), `--json` | Read-only listing of managed MCP backups. | `cmd/agentic-sdd/mcp_write.go` |
| `mcp restore preview ID` | `--home PATH`, `--config PATH`, `--json` | Read-only entry-level restore preview for positional `ID`; exits `1` on conflicts. | `cmd/agentic-sdd/mcp_write.go` |
| `mcp restore apply ID` | `--expect-config SHA256` (required), `--home PATH`, `--config PATH`, `--json` | Entry-level restore for positional `ID`; prompts `[y/N]` when `env.Interactive` is true. | `cmd/agentic-sdd/mcp_write.go` |

## Execution family (`run`, `task`, `review`)

Shared execution flags (`cmd/agentic-sdd/execution_cli.go`): `--repo DIR` (default: current working tree), `--run-root DIR` (default: `<repo>/.agentic-sdd/runs`), `--format text|json` (default `text`).

Execution flag defaults (`internal/execution/types.go`): `run check --timeout 15m` (`0` or negative disables the wall-clock limit); `run check --max-output-bytes 67108864` per stream (`0` or negative selects the default; effective range `1024`–`1073741824`); `review package --max-file-bytes 2097152` and `--max-total-bytes 33554432` (`0` or negative selects the default).

| Subcommand | Additional flags / positional contract | Distinct exit codes & contract notes | Source |
|---|---|---|---|
| `run init` | `--feature DIR` + `--batch ID`, or `--direct` + optional `--label TEXT` | `--direct` cannot combine with `--feature`/`--batch` (`2`). Exits `0`, `1` runtime, `2` usage/unsafe/conflict/missing scope. | `cmd/agentic-sdd/run.go` |
| `run status` | `--run ID` | Missing run for `status` exits `1` (unlike `init` missing batch `2`). | `cmd/agentic-sdd/run.go`, `cmd/agentic-sdd/execution_cli.go` |
| `run record` | `--run ID` (required), `--task T01` (required for feature runs, defaults to `direct` for `--direct`), `--result FILE\|-` (required) | Ingests untrusted claim record; `0` recorded, `1` runtime, `2` usage/validation. | `cmd/agentic-sdd/run.go` |
| `run clean` | `--run ID`, `--yes` | Without `--yes`, previews without deleting; with `--yes`, deletes only that run directory. | `cmd/agentic-sdd/run.go` |
| `run check` | `--run ID` (required), `--task T01`, `--timeout DURATION` (default `15m`; `0` disables), `--max-output-bytes N` (default `67108864` per stream, clamped to `1024`–`1073741824`), `-- COMMAND [ARG...]` | `--` separator before command is required (even `--help` without `--` exits `2`). Exits `0` pass, `1` fail/error, `124` timed out, `2` usage. | `cmd/agentic-sdd/run.go` |
| `task brief` | `--run ID` (required), `--task T01` (required), `--force` | Refuses `--direct` runs (`2`) and foreign checkpoints (`2`); `--force` refreshes differing brief bytes without bypassing changed scope. | `cmd/agentic-sdd/task.go` |
| `review package` | `--run ID` (required), `--base COMMIT` (required), `--head COMMIT` (defaults to current `HEAD`), `--max-file-bytes N` (default `2097152`), `--max-total-bytes N` (default `33554432`) | Exits `0` complete package, `3` partial package with explicit omissions, `1` runtime/missing run, `2` usage/unsafe. | `cmd/agentic-sdd/review.go` |

## Presentation and trace environment variables

- `NO_COLOR`: non-empty disables ANSI color styling (`cmd/agentic-sdd/env.go`).
- `TERM=dumb`: switches to ASCII glyphs and disables color (`cmd/agentic-sdd/env.go`).
- `COLUMNS`: positive integer sets wrap width (defaults to `80`, `cmd/agentic-sdd/env.go`).
- `AGENTIC_SDD_ANIMATE=1` / `AGENTIC_SDD_REDUCED_MOTION=1`: CLI animation opt-in (`cmd/agentic-sdd/env.go`) and TUI reduced-motion control (`experiments/tui/cmd/internal/tuirun/run.go`).
- `AGENTIC_SDD_FORCE_INTERACTIVE=1`: enables interactive CLI prompts on scripted streams, while `asdx-core tui` refuses it with exit `2` (`cmd/agentic-sdd/env.go`, `cmd/agentic-sdd/tui.go`).
- `AGENTIC_SDD_TRACE`: path to append JSONL trace events (`0600` regular file, `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/trace.go`).

## Captured help inventory

| ID | Argv | Exit | Flags observed in help | Primary source |
|---|---|---|---|---|
| `top` | `asdx-core --help` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/main.go` |
| `help-preview` | `asdx-core help preview` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `help-apply` | `asdx-core help apply` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `help-backups` | `asdx-core help backups` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `help-restore` | `asdx-core help restore` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `help-logs` | `asdx-core help logs` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `help-guide` | `asdx-core help guide` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `help-version` | `asdx-core help version` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `help-tui` | `asdx-core help tui` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `tui` | `asdx-core tui --help` | `0` | `--fleet-config`, `--help`, `--home`, `--opencode-free-first`, `--repo`, `--with-opencode`, `-h` | `cmd/agentic-sdd/tui.go` |
| `orchestrate` | `asdx-core help orchestrate` | `0` | `--home`, `--opencode-free-first`, `--repo`, `--with-opencode` | `cmd/agentic-sdd/main.go` |
| `fleet` | `asdx-core fleet --help` | `0` | `--adopt`, `--backup`, `--config`, `--home`, `--opencode-bin`, `--output`, `--plan`, `--repo` | `cmd/agentic-sdd/fleet.go` |
| `fleet-status` | `asdx-core fleet status --help` | `0` | `--config`, `--home`, `--repo` | `cmd/agentic-sdd/fleet.go` |
| `fleet-plan` | `asdx-core fleet plan --help` | `0` | `--adopt`, `--config`, `--home`, `--output`, `--repo` | `cmd/agentic-sdd/fleet_plan.go` |
| `fleet-apply` | `asdx-core fleet apply --help` | `0` | `--home`, `--plan` | `cmd/agentic-sdd/fleet.go` |
| `fleet-restore` | `asdx-core fleet restore --help` | `0` | `--backup`, `--home` | `cmd/agentic-sdd/fleet.go` |
| `fleet-doctor` | `asdx-core fleet doctor --help` | `0` | `--config`, `--opencode-bin` | `cmd/agentic-sdd/fleet_doctor.go` |
| `mcp` | `asdx-core mcp help` | `0` | `--auth`, `--codegraph-bin`, `--config`, `--expect-config`, `--home`, `--json`, `--opencode-bin`, `--project`, `--servers` | `cmd/agentic-sdd/mcp.go` |
| `mcp-status` | `asdx-core mcp status --help` | `0` | `--config`, `--home`, `--json`, `--project` | `cmd/agentic-sdd/mcp.go` |
| `mcp-doctor` | `asdx-core mcp doctor --help` | `0` | `--codegraph-bin`, `--config`, `--home`, `--json`, `--opencode-bin`, `--project` | `cmd/agentic-sdd/mcp.go` |
| `mcp-preview` | `asdx-core mcp preview --help` | `0` | `--auth`, `--config`, `--home`, `--json`, `--servers` | `cmd/agentic-sdd/mcp.go` |
| `mcp-apply` | `asdx-core mcp apply --help` | `0` | `--auth`, `--config`, `--expect-config`, `--home`, `--json`, `--opencode-bin`, `--servers` | `cmd/agentic-sdd/mcp_write.go` |
| `mcp-restore` | `asdx-core mcp restore help` | `0` | `--config`, `--expect-config`, `--home`, `--json` | `cmd/agentic-sdd/mcp_write.go` |
| `help-run` | `asdx-core help run` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `run` | `asdx-core run help` | `0` | `--batch`, `--direct`, `--feature`, `--format`, `--label`, `--max-output-bytes`, `--repo`, `--result`, `--run`, `--run-root`, `--task`, `--timeout`, `--yes` | `cmd/agentic-sdd/run.go` |
| `help-task` | `asdx-core help task` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `task` | `asdx-core task help` | `0` | `--force`, `--format`, `--repo`, `--run`, `--run-root`, `--task` | `cmd/agentic-sdd/task.go` |
| `help-review` | `asdx-core help review` | `0` | `--apply`, `--help`, `--home`, `--repo`, `--version`, `-h` | `cmd/agentic-sdd/helptext.go` |
| `review` | `asdx-core review help` | `0` | `--base`, `--format`, `--head`, `--max-file-bytes`, `--max-total-bytes`, `--repo`, `--run`, `--run-root` | `cmd/agentic-sdd/review.go` |

## Top-level help excerpt (`asdx-core --help`)

```text
Usage: asdx-core [preview|apply] [--repo PATH] [--home PATH]
       asdx-core tui [--repo PATH] [--home PATH]
       asdx-core version
       asdx-core backups [--repo PATH] [--home PATH]
       asdx-core logs [--home PATH]
       asdx-core restore [ID] [--apply] [--repo PATH] [--home PATH]
       asdx-core restore list|preview|apply [ID] [--repo PATH] [--home PATH]
       asdx-core guide [--repo PATH] [--home PATH]
       asdx-core fleet status|plan|apply|restore|doctor [--config PATH] [--home PATH]
       asdx-core mcp status|doctor|preview [--config PATH] [--home PATH] [--json]
       asdx-core run init|status|check|record|clean [--run ID] [--repo PATH]
       asdx-core task brief --run ID --task T01 [--repo PATH]
       asdx-core review package --run ID --base COMMIT [--repo PATH]
       asdx-core help [preview|apply|backups|logs|restore|tui|fleet|mcp|run|task|review]

Commands:
  preview        Show planned changes without writing files (default).
  apply          Back up changed existing skills, then install them.
  version        Print version, commit, and build date information.
  backups        List backups newest first (same as "restore list").
  logs           Show the latest apply or restore log.
  restore        Preview or restore a backup, selected by id or number.
  guide          Interactive step-by-step wizard for terminal users.
  tui            Open the terminal UI in a real terminal (packaged asdx companion).
  fleet          Manage agent fleet profiles and model overrides.
  mcp            Inspect the default MCP tools read-only (status, doctor, preview).
  run            Create or inspect a private execution run.
  task           Extract one authorized task brief into a run.
  review         Package real Git changes for a bounded review.

A bare "asdx-core" always previews and never opens the terminal UI. Run the
separate "asdx" command for the terminal UI; "asdx-core tui" is the explicit
compatibility launcher for that identity-verified sibling. "asdx-core preview"
always previews without writing.

restore forms:
  restore                   Interactively list backups and pick one to preview.
  restore ID                Preview restoring backup ID.
  restore ID --apply        Restore backup ID.
  restore list               Same listing as "backups".
  restore preview [ID]       Same as "restore [ID]" without --apply.
  restore apply [ID]         Same as "restore ID --apply".
With no ID and no --apply/apply, restoring prompts for confirmation before
writing anything. Without a terminal, an ID is required.

Flags:
  --repo PATH  Repository containing skills-files (default: skills embedded in this binary).
  --home PATH  User home for skill destinations (default: your home).
  --apply      Legacy form of the apply command; cannot be combined with a command.
  --version    Legacy form of the version command.
  -h, --help  Show this help.
```

## MCP family help excerpt (`asdx-core mcp help`)

```text
Usage: asdx-core mcp [status] [--home PATH] [--config PATH] [--project PATH] [--json]
       asdx-core mcp doctor [--home PATH] [--config PATH] [--project PATH]
                              [--opencode-bin PATH] [--codegraph-bin PATH] [--json]
       asdx-core mcp preview [--servers NAME,...] [--auth SERVER=MODE]
                               [--home PATH] [--config PATH] [--json]
       asdx-core mcp apply --expect-config SHA256 [--opencode-bin PATH]
                             [--servers NAME,...] [--auth SERVER=MODE]
                             [--home PATH] [--config PATH] [--json]
       asdx-core mcp restore list [--home PATH] [--json]
       asdx-core mcp restore preview ID [--home PATH] [--config PATH] [--json]
       asdx-core mcp restore apply ID --expect-config SHA256
                                  [--home PATH] [--config PATH] [--json]
       asdx-core mcp help

Commands:
  status   Report the OpenCode MCP configuration target and the three default
           servers: CodeGraph (local), Context7 and Exa (remote). Default when
           no subcommand is given.
  doctor   Offline diagnostics for the same configuration and prerequisites.
  preview  Print the structured configuration plan: target, base and plan
           digest, one entry per selected server, conflicts and a token.
           Nothing is written.
  apply    Write the selected managed entries when the current digest equals
           --expect-config and the local runtime probe succeeds.
  restore  list, preview or apply an entry-level restore of managed entries
           only. Whole-file restore is not part of this build.
  help     Show this help.

Legacy note: "agentic-sdd" wraps "asdx-core" during the deprecation window.

Scope of this build (MCP-A + MCP-B):
  MCP-A status, offline doctor and preview stay read-only. MCP-B adds apply and
  restore for the existing global OpenCode configuration only; a missing
  configuration is never created and there is no replace or force option.
  "doctor --network" and "doctor --connect" remain deferred and are rejected.
  No command performs a network call, OAuth flow, package installation or
  model call. A plaintext API key or token is never written; managed remote
  entries use {env:NAME} placeholders and OpenCode owns OAuth credentials.

Flags:
  --home PATH           User home directory (defaults to your home).
  --config PATH         Explicit OpenCode configuration file to inspect.
  --project PATH        Project path used for prerequisite checks (status/doctor).
  --servers NAME,...    Servers to configure (default: all default servers).
  --auth SERVER=MODE    context7=oauth|api-key; exa=keyless|oauth|api-key.
  --expect-config SHA256  Base digest the reviewed preview printed (apply/restore).
  --opencode-bin PATH   Local OpenCode binary (doctor lookup; apply runtime probe).
  --codegraph-bin PATH  Local CodeGraph binary to look up (doctor only).
  --json                Emit stable, script-compatible JSON instead of text.

Exit Codes:
  0  Success, a produced report, or a real no-op
  1  Stale plan, conflict, unsupported runtime, rollback or engine/I/O error
  2  Usage error, flag error, validation, unsafe target or unknown subcommand
```

## Evidence sources

- `cmd/agentic-sdd/env.go`
- `cmd/agentic-sdd/execution_cli.go`
- `cmd/agentic-sdd/fleet.go`
- `cmd/agentic-sdd/fleet_doctor.go`
- `cmd/agentic-sdd/fleet_plan.go`
- `cmd/agentic-sdd/guide.go`
- `cmd/agentic-sdd/helptext.go`
- `cmd/agentic-sdd/logs.go`
- `cmd/agentic-sdd/main.go`
- `cmd/agentic-sdd/mcp.go`
- `cmd/agentic-sdd/mcp_write.go`
- `cmd/agentic-sdd/opencode_disabled.go`
- `cmd/agentic-sdd/review.go`
- `cmd/agentic-sdd/run.go`
- `cmd/agentic-sdd/task.go`
- `cmd/agentic-sdd/trace.go`
- `cmd/agentic-sdd/tui.go`
- `internal/buildinfo/buildinfo.go`
<!-- generated:reference/cli:end -->
