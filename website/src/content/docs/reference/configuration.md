---
title: Configuration reference
description: Configuration locations, parser contracts and precedence.
---

Source-derived configuration facts. Subsystems keep separate schemas; this page does not define a unified runtime configuration.

<!-- generated:reference/configuration:start -->
Source-derived configuration facts for the current checkout. They do not certify runtime behavior, provider availability or release support.

## Precedence by subsystem

| Subsystem | Selector | Default | Boundary |
| --- | --- | --- | --- |
| skill installation | `--repo` | `skills-files embedded in the binary` | an explicit checkout reads the skills-files/ directory under the repository root |
| skill installation and fleet | `--home` | `the OS user home directory (os.UserHomeDir)` | all client, agent, backup and MCP state paths derive from this value |
| fleet | `--config PATH` | `none: personal fleet.json is required` | no product environment variable or implicit default path selects it |
| MCP | `--config PATH` | `<home>/.config/opencode/opencode.json or <home>/.config/opencode/opencode.jsonc` | an explicit path wins; both default files present is an ambiguity conflict; neither present is format none and no file is created |
| execution | `--run-root DIR` | `<repo>/.agentic-sdd/runs` | --repo defaults to the current working tree |
| skill backups | `--repo` | `<home>/.agentic-sdd/backups when assets are embedded` | an explicit checkout keeps backups under its own backups/ directory |

No product-specific environment variable in the inspected CLI and TUI sources overrides these selectors. The only implicit home source is the OS user-home lookup behind `--home`.

## Environment variables

| Variable | Accepted values | Effect | Scope |
| --- | --- | --- | --- |
| `AGENTIC_SDD_TRACE` | `<path>` | append the opt-in JSON-Lines flow trace; unset or empty disables it | CLI |
| `AGENTIC_SDD_FORCE_INTERACTIVE` | `exactly 1` | treat stdin, stdout and stderr as terminals for scripted terminal behavior | CLI presentation and TUI launch gate |
| `AGENTIC_SDD_ANIMATE` | `exactly 1` | opt in to animation; further terminal gates apply | CLI guide presentation |
| `AGENTIC_SDD_ASCII` | `exactly 1` | use ASCII borders and symbols | TUI companion |
| `AGENTIC_SDD_REDUCED_MOTION` | `exactly 1` | disable TUI animation | TUI companion |
| `NO_COLOR` | `any non-empty value` | disable ANSI color | CLI and TUI presentation |
| `TERM` | `dumb` | disable color and select ASCII-only glyphs | CLI presentation |
| `COLUMNS` | `positive integer` | set terminal width; otherwise 80 | CLI presentation |

These variables affect presentation, tracing or the TUI launch gate. They do not change the fleet config path, the MCP config path, the run root or a backup root.

## Output contracts

| Subsystem | Contract |
| --- | --- |
| Execution | `--format text\|json`; JSON carries `schemaVersion` 1 and a kind value |
| MCP | `--json` emits stable, script-compatible JSON instead of text |
| Fleet | `--output PATH` writes the canonical plan JSON; `fleet.json` remains the input contract |

## Boundary

Configuration is read from explicit or documented locations scoped by subsystem. There is no single unified configuration schema: each subsystem owns its parser, constants and defaults, and the reference pages keep those contracts separate.
<!-- generated:reference/configuration:end -->
