---
title: Files and backups
description: Client skill destinations, backup and manifest formats, digests, restore behavior, and the boundaries separating each store.
---

Source-derived filesystem, backup, and manifest facts for Agentic-SDD. See [CLI reference](./cli.md) and [TUI reference](./tui.md) for the commands and screens that read or write these locations.

## Integrity and store boundaries

### Digests are not signatures

SHA-256 tree and file digests in skill manifests, fleet manifests, and MCP plans detect accidental truncation, corruption, or file drift between a preview and a later apply or restore. They do **not** sign binaries or manifests cryptographically and do not verify authorship if another local process already has write access to the destination or backup directory.

### Stores are not interchangeable

Agentic-SDD keeps four separate state stores with distinct roots, lock files, and schemas:

- **Skill backups** (`internal/skillsync`): `manifest.json` under `<home>/.agentic-sdd/backups/<id>/` or `<repo>/backups/<id>/`, read by `asdx-core backups` and `asdx-core restore`.
- **Fleet sidecars** (`internal/fleet`): `fleet-manifest.json` inside fleet backup directories, paired with `asdx-core fleet` and Fleet Manager in `asdx`.
- **MCP ownership state** (`internal/skillsync` MCP-B): `<home>/.agentic-sdd/mcp/manifest.json`, `backups/<id>/`, `.lock`, `snapshot.json`, and `intent.json`, used only for entry-level edits to existing global OpenCode config (`~/.config/opencode/opencode.json` or `.jsonc`). See [MCP reference](./mcp.md).
- **Execution run scratch** (`internal/execution`): `<repo>/.agentic-sdd/runs/<run-id>/` (or explicit `--run-root`). See [Execution reference](./execution.md).

Backup IDs and manifest files from one store cannot be inspected, applied, or restored by another store's commands.

### Embedded versus explicit `--repo` backup roots

Skill install and restore choose their backup root from how skill sources are loaded:

- **Embedded assets (no `--repo`)**: backups are written under `<home>/.agentic-sdd/backups`.
- **Explicit `--repo PATH`**: backups are written under `<repo>/backups`.

Because `asdx-core backups` and `asdx-core restore` resolve the backup root using the same `--repo` rule, a backup created with `--repo` is not listed when `--repo` is omitted, and vice versa. Keep `<repo>/backups/` local and out of Git, as it may contain prior local skill content.

### Which files get backed up

Which files get backed up is narrow by design: a skill backup copies the managed `agentic-sdd-*` skill directories that are about to be replaced, plus the `manifest.json` that records their paths and SHA-256 digests. Unrelated skills and user configuration files such as `opencode.json` or `AGENTS.md` stay outside the skill backup, and operation logs remain under `<home>/.agentic-sdd/logs/`.

### Skill formats 1–3 versus fleet format 4

Skill `manifest.json` formats 1–3 are additive revisions of the installer backup schema (`Format < 2` uses the legacy format 1 reader without per-entry records; format 2 records tool metadata and per-entry `entries`; format 3 adds `opencode_targets` and `opencode-agent` entries when OpenCode agent profiles change).

Fleet `fleet-manifest.json` uses `format: 4` (`internal/fleet/manifest.go`). It is a separate sidecar schema for fleet instance provenance, source/config/desired/installed digests, and rollback metadata — not an upgrade to skill `manifest.json` — and `asdx-core restore` cannot read it.

### Private execution scratch

`<repo>/.agentic-sdd/runs/` holds private execution workspaces (`0700` run directories, `0600` JSON artifacts, and a self-ignoring `.gitignore` containing `*` so `git status` stays clean without editing the repository's root `.gitignore`). Run artifacts are observations, not approvals, verification gates, or OS sandboxes. Keep them out of Git, backups, and uploads, and remove a run explicitly with `asdx-core run clean --run ID --yes`.

## Related reference pages

- [Configuration reference](./configuration.md) — configuration files, environment variables, and per-subsystem precedence.
- [Fleet reference](./fleet.md) — `fleet.json` schema and format 4 sidecar fields.
- [MCP reference](./mcp.md) — MCP target resolution and ownership store details.
- [Execution reference](./execution.md) — run workspace files, locks, and review packages.

<!-- generated:reference/files-and-backups:start -->
Source-derived destination and backup facts for the current checkout.

## Client skill destinations

| Client | Destination |
| --- | --- |
| `agents` | `<home>/.agents/skills` |
| `codex` | `<home>/.codex/skills` |
| `claude` | `<home>/.claude/skills` |
| `agy` | `<home>/.gemini/antigravity-cli/skills` |

The restore reader accepts only saved `agentic-sdd-*` skill trees; other skill names are refused.

## OpenCode agent profiles

| Item | Contract |
| --- | --- |
| Client name | `opencode-agents` |
| Target | `<home>/.config/opencode/agents` |
| Manifest entry kind | `opencode-agent` |

## Backup roots

| Source selection | Backup root |
| --- | --- |
| embedded skills (no explicit --repo) | `<home>/.agentic-sdd/backups` |
| explicit --repo checkout | `<repo>/backups` |

Backup IDs use the UTC layout `20060102T150405.000000000Z`. Each backup directory contains `manifest.json`.

## `manifest.json` formats

| Format | Fields |
| --- | --- |
| Format 1 (original) | `created`, `source`, `targets`, `skills` |
| Formats 2 and 3 (additive) | `format`, `id`, `created_at`, `operation`, `restored_from`, `tool`, `home`, `backup_root`, `opencode_targets`, `entries` |
| Format 2 baseline | Used when the transaction changes only skill directories |
| Format 3 file changes | `opencode_targets` is recorded; OpenCode agent entries use kind `opencode-agent` |

| Record part | Fields |
| --- | --- |
| Tool metadata | `version`, `commit`, `date`, `go_version` |
| Per-entry | `kind`, `client`, `skill`, `path`, `action`, `backup`, `sha256` |
| Entry actions | `replaced`, `installed` |

## Digests and restore

- Tree digest: relative paths, entry kinds (file or directory) and file contents in stable lexical order
- File digest: raw bytes of one regular file; special files are refused
- A restored backup must match the current home identity and its saved content is validated before use
- Format 1 is legacy: `format < 2` keeps the older reader path without per-entry records

## Separate stores

| Store | Owner | Identity |
| --- | --- | --- |
| Skill backup `manifest.json` | internal/skillsync | Formats 1-3 |
| Fleet sidecar `fleet-manifest.json` | internal/fleet | Format 4 |
| MCP ownership `manifest.json` | internal/skillsync MCP-B | Format 1, manifest_version 1 |
| Execution run scratch | internal/execution | schemaVersion 1 |

These files live in different roots and must not be read as one schema. The MCP ownership manifest records only managed entry names and installed digests; the fleet sidecar records per-file provenance; the skill backup manifest records skill destinations and saved copies.
<!-- generated:reference/files-and-backups:end -->
