---
title: Configuration reference
description: Where fleet, MCP, execution, profile, and presentation settings live, how each parser resolves values, and which precedence rules apply.
---

Agentic-SDD has no single configuration file and no single precedence rule. Fleet instances, MCP servers, execution runs, agent-profile frontmatter and terminal presentation are separate contracts, and each owns its own location, parser and failure mode. This page explains where each one reads, how each resolves a value, and where the formats genuinely differ. It does not define a unified runtime configuration.

The generated block at the end is derived from the current source checkout. The explanation and the fixtures below are curated additions; they were checked against the parsers named here, not against a running provider, client or release.

For one subsystem's full contract use [fleet](fleet.md), [MCP](mcp.md), [execution](execution.md), [profiles](profiles.md) and [files and backups](files-and-backups.md). Command flags and exit codes stay in the [CLI reference](cli.md).

## Where each subsystem reads

| Subsystem | Read from | Format | Owning source |
| --- | --- | --- | --- |
| Fleet instances | the path passed to `--config`; no default path and no environment variable | strict JSON, 1 MiB maximum | `internal/fleet/parse.go`, `internal/fleet/fleet.go` |
| MCP servers | `--config` inside `<home>/.config/opencode`, otherwise `opencode.json`, otherwise `opencode.jsonc` | JSON or JSONC, 1 MiB maximum | `internal/skillsync/mcp_config.go` |
| Execution runs | no configuration file; `--run-root`, defaulting to `<repo>/.agentic-sdd/runs` | JSON artifacts at `schemaVersion` 1 | `internal/execution/workspace.go`, `internal/execution/record.go` |
| Agent profiles | frontmatter in the canonical Markdown under `opencode-pack/`, installed to `<home>/.config/opencode/agents` | `---`-fenced frontmatter, preserved byte-for-byte outside `model` | `internal/fleet/render.go`, `internal/skillsync/sync.go` |
| Terminal presentation | the process environment | exact string comparison | `cmd/agentic-sdd/env.go`, `experiments/tui/cmd/internal/tuirun/run.go` |

Table 1. Where each subsystem reads. No two rows share a file.

There is no user profile configuration file. Profiles come from the pack (`opencode-pack/agents/`, `opencode-pack/adapters/` and `opencode-pack/presets/free-first/`), and the only supported per-instance overrides are the fleet `model` and `enabled` fields (`internal/fleet/resolve.go`, `internal/fleet/render.go`).

## How each subsystem resolves a value

The generated precedence table lists each selector and its default. What differs between them is the behaviour around those selectors.

- **Fleet** reads exactly the `--config` path on the commands that require one (`fleet status`, `fleet plan`). Nothing searches for `fleet.json`, and no product environment variable names it, so a missing or mistyped path is a usage error rather than a silent fallback. `--home` changes where instances are installed and inspected; it never changes which configuration is read. `fleet doctor` accepts `--config` as optional input to its local-only diagnostics. (`cmd/agentic-sdd/fleet.go`)
- **MCP** honours an explicit `--config` only while it stays inside `<home>/.config/opencode`; a path outside that directory is refused as an unsafe target instead of accepted. Without `--config`, exactly one of `opencode.json` and `opencode.jsonc` must exist. Both present is an ambiguity conflict, and neither present reports format `none` with no target bytes, because this build never creates the file. This is a two-candidate ambiguity check, not a first-match search list. (`mcpResolvePlanTarget` in `internal/skillsync/mcp_config.go`)
- **Execution** derives its workspace from `--repo` (the current working tree by default) and then from `--run-root`. An empty `--run-root` selects `<repo>/.agentic-sdd/runs` and writes a self-ignoring `.gitignore` there; an explicit root is used as given and its Git visibility is reported instead of forced. (`ResolveRunRoot` and `prepareRunRoot` in `internal/execution/workspace.go`)
- **Profiles** have no runtime lookup at all. The canonical pack file is the source, and a fleet `instances.<id>.model` value replaces only that file's top-level `model:` scalar while every other frontmatter field and the prompt body stay byte-identical. An omitted or empty fleet model leaves the canonical value untouched. (`internal/fleet/render.go`)
- **Backups** are the one case where `--repo` changes a write location rather than a read location; [files and backups](files-and-backups.md) owns that split.

Because the layers are unequal, no global order applies. Fleet and MCP have no environment layer, execution has no configuration file, and profile selection has no user file. A "flag, then environment, then file" rule carried from one subsystem to another will be wrong.

## Formats: JSON, JSONC and frontmatter

**Fleet is strict JSON.** `fleet.json` is decoded with `encoding/json` while unknown fields are disallowed, after a separate pass rejects duplicate object keys. Comments, trailing commas and trailing data are invalid, as are input over 1 MiB, invalid UTF-8 and nesting deeper than 32 levels (`internal/fleet/parse.go`). A field outside the schema is rejected, not ignored; a `clients` array is one example, since only the TUI selects clients.

**MCP accepts JSON or JSONC, and refuses to edit JSONC.** The extension picks the format, and only `.json` and `.jsonc` are supported (`mcpFormatForPath`). For reading, `stripJSONC` blanks comments and trailing commas while preserving line offsets (`internal/skillsync/mcp.go`), so a `.jsonc` file with comments is readable. Writing is different: this build cannot prove lossless comment and formatting preservation, so it refuses the edit and prints a copyable `mcp.servers` snippet instead (`internal/skillsync/mcp_config.go`). A `//` comment inside a `.json` target is simply invalid JSON. The document schema is versioned too: `mcp.servers` is required, a legacy `mcp.<name>` document is refused and never migrated, and a managed entry must keep a diffable shape: `local` with a non-empty `command` array of strings, `remote` with a non-empty `url`, `disabled` as a boolean, `headers` as a string map. Unrelated servers and unknown fields elsewhere are carried through untouched.

**Execution records are strict JSON as well.** A run root holds JSON artifacts at `schemaVersion` 1. The user-authored input is the record passed to `run record --result`, and ingestion rejects unknown fields, duplicate keys, oversize input, NUL bytes and unsafe relative paths (`internal/execution/record.go`, `internal/execution/types.go`).

**Profile frontmatter is not parsed as YAML.** `internal/fleet/render.go` reads it as a line state machine between two `---` delimiter lines and understands only what it needs: a top-level `model:` scalar and a `permissions:` list of `action`, `resource` and `effect` rules. Unknown top-level keys are preserved, but rendering fails closed on a duplicate `model`, a `|` or `>` block scalar for the model, a list item outside `permissions`, a malformed `action`, a rule missing any of its three fields, or an indented field outside the `permissions` block. Everything else in the file is interpreted by the OpenCode host that reads the installed profile, not by Agentic-SDD.

## Flags that look global but are not

Every command parses its own `flag.FlagSet`; there is no universal global flag set (`cmd/agentic-sdd/main.go`, `execution_cli.go`, `fleet.go`, `mcp.go`, `run.go`). Two traps follow.

- `--config` means `fleet.json` in the `fleet` family and the OpenCode configuration in the `mcp` family: same spelling, different schema and different default rule.
- `--home` appears on most state-touching commands but not on every command. `--run-root` exists only on `run`, `task brief` and `review package`; `--servers` and `--auth` only on `mcp preview` and `mcp apply`; `--json` on MCP is not the same switch as `--format text|json` on execution, and the `fleet` family writes its plan JSON to `--output`.

There is no `--clients` flag in the CLI; client destinations are chosen in the TUI. The [CLI reference](cli.md) owns the per-command flag table.

## Environment-variable semantics

The generated table below lists every variable read by the CLI presentation resolver and the TUI companion, and none of them selects a file, a run root or a profile. Comparisons are exact: the `AGENTIC_SDD_*` presentation switches must equal `1` (except `AGENTIC_SDD_TRACE`, which takes a path), `TERM=dumb` also selects ASCII glyphs, `COLUMNS` must parse as a positive integer or the width stays 80, and `NO_COLOR` disables color for any non-empty value (`cmd/agentic-sdd/env.go`). Because `AGENTIC_SDD_FORCE_INTERACTIVE=1` fakes the terminal facts, it cannot satisfy the TUI's real-terminal launch gate (`cmd/agentic-sdd/tui.go`).

Two variables sit outside that resolver. `AGENTIC_SDD_NO_UPDATE_CHECK=1` disables the TUI update check and reports offline mode (`experiments/tui/update_state.go`). `AGENTIC_SDD_TUI=1` is the legacy opt-in used by the experimental `tui-spike` path; the `asdx` companion path keeps the same safety rule without requiring it (`experiments/tui/config.go`, `experiments/tui/cmd/internal/tuirun/run.go`). Other `AGENTIC_SDD_*` names in the tree belong to test harnesses, such as the permission probes in `internal/skillsync/permissions_test.go`, and are not product configuration.

## Fixtures

`website/tests/examples/configuration/` holds inert fixtures for these contracts. Nothing there is installed, executed or read by a product command. The expected outcomes below were produced by a throwaway program that called the Go parsers directly, and the configuration targets keep the extension the parser uses to choose behaviour (`.json` or `.jsonc`).

| Fixture | Subsystem | Expected parser outcome |
| --- | --- | --- |
| `fleet.valid.json` | fleet | accepted |
| `fleet.invalid-unknown-field.json` | fleet | rejected: unknown field `clients` |
| `fleet.invalid-duplicate-key.json` | fleet | rejected: duplicate key |
| `fleet.invalid-model.json` | fleet | rejected: model without a provider segment |
| `mcp.valid.json` | MCP | accepted; plan reports no conflicts |
| `mcp.valid.jsonc` | MCP | parsed, then edit refused as unsupported |
| `mcp.invalid-comment-in-json.json` | MCP | rejected: comment in a `.json` target |
| `mcp.invalid-unclosed-jsonc.jsonc` | MCP | rejected: unclosed block comment |
| `mcp.invalid-legacy-schema.json` | MCP | rejected: legacy `mcp.<name>` schema |
| `mcp.invalid-managed-entry.json` | MCP | rejected: `command` is not an array of strings |
| `execution-record.valid.json` | execution | accepted |
| `execution-record.invalid-unknown-field.json` | execution | rejected: unknown field |
| `profile.valid.md` | profiles | accepted; renders with one model change |
| `profile.invalid-duplicate-model.md` | profiles | rejected: duplicate `model` |
| `profile.invalid-indented-field.md` | profiles | rejected: indented field outside `permissions` |
| `profile.invalid-permission-rule.md` | profiles | rejected: rule missing `effect` |

Table 2. Fixture intent. A rejected fixture is a negative case, not a supported configuration.

The JSON fixtures also load as YAML documents because JSON is a YAML subset. `mcp.valid.jsonc` does not, because `//` is not a YAML comment; JSONC is read by stripping comments and trailing commas, not by a YAML parser. The frontmatter block in `profile.valid.md` does parse as YAML, but Agentic-SDD never uses a YAML parser for it.

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
