---
title: MCP reference
description: Managed MCP configuration, previews and recovery.
---

Source-derived MCP configuration facts. Configuration does not authenticate, connect or prove a server works.

<!-- generated:reference/mcp:start -->
Source-derived MCP facts for the current checkout. Status, doctor and preview are read-only and offline; apply and restore require the expected base digest from a reviewed preview before they write.

## Target selection

| Case | Result |
| --- | --- |
| Explicit `--config` | Must end in `.json` or `.jsonc`; an explicit missing path is a missing target, not a new file |
| Default JSON | `<home>/.config/opencode/opencode.json` |
| Default JSONC | `<home>/.config/opencode/opencode.jsonc` |
| Both default files | Ambiguity conflict; no target is chosen |
| Neither default file | none (no file is created) |

A symlink, special file or out-of-scope target is refused. The configuration read is bounded to 1048576 bytes (1 MiB).

## Schema and format

- Accepted target formats: `json`, `jsonc`
- Observed schema values: `none`, `v2`
- Managed entries live under `mcp.servers` in the OpenCode V2 shape
- legacy v1 mcp.<name> entries are refused and never migrated to mcp.servers; there is no automatic rewrite
- A JSONC edit is refused when comment and formatting preservation cannot be proven

## Default managed servers

| Server | Kind | Purpose | Prerequisite |
| --- | --- | --- | --- |
| `codegraph` | local | repository code intelligence | codegraph CLI |
| `context7` | remote | hosted library documentation | none |
| `exa` | remote | hosted web research | none |

## Plan, apply and restore

Plan identity is `mcp-plan-v1`. A plan is bound to the base configuration digest and its own plan digest; an apply must present the expected base digest or fail as stale.

- Plan operations: `add`, `change`, `leave`, `disabled`, `conflict`, `restore`, `remove`, `missing`
- Observed server states: `not-configured`, `configured`, `disabled`, `missing-prerequisite`, `auth-not-checked`, `conflict`, `schema-mismatch`, `unsafe-target`, `unknown`
- Authentication modes: `oauth`, `api-key`, `keyless`
- Snapshot kinds: `apply`, `restore`
- A missing configuration is never created; there is no replace or force option
- No command performs a network call, OAuth flow, package installation or model validation

## Ownership and recovery store

| Item | Contract |
| --- | --- |
| Ownership manifest | `<home>/.agentic-sdd/mcp/manifest.json` |
| Manifest format | 1 (manifest_version 1) |
| Manifest fields | `format`, `manifest_version`, `tool_version`, `created`, `target`, `target_format`, `schema`, `base_digest`, `installed_digest`, `backup_id`, `backup_path`, `runtime_version`, `runtime_evidence`, `actions`, `entries` |
| Per-entry fields | `name`, `installed_digest`, `backup_id` |
| Backups | `<home>/.agentic-sdd/mcp/backups` |
| Lock | `<home>/.agentic-sdd/mcp/.lock` |
| Snapshot | `snapshot.json` format 1 |
| Intent record | `intent.json` |
| Intent phases | `prepared`, `renamed`, `manifest-written`, `complete`, `rolled-back`, `needs-recovery` |
| Runtime evidence | `local-probe` |

The MCP ownership manifest is a separate store, not the skill backup manifest and not the fleet sidecar. Restore is entry-level only; whole-file replacement is not part of this build.

## CLI

- `mcp status`
- `mcp doctor`
- `mcp preview`
- `mcp apply`
- `mcp restore list`
- `mcp restore preview`
- `mcp restore apply`

Documented exit codes are 0 (success, a produced report or a real no-op), 1 (stale plan, conflict, unsupported runtime, rollback or engine/I/O error) and 2 (usage, flags, validation, unsafe target or unknown subcommand).

## Boundary

The manifest copies only managed-entry text and hashes. Unrelated servers, comments where preservation is unproven and credentials remain outside the managed write; remote entries use environment placeholders and OpenCode owns OAuth credentials.
<!-- generated:reference/mcp:end -->
