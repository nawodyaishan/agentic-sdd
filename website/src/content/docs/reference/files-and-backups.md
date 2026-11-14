---
title: Files and backups
description: Managed destinations, formats and recovery stores.
---

Source-derived filesystem and manifest facts. Checksums detect corruption; they are not signatures.

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
