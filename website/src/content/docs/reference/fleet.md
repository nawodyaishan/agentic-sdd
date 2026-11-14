---
title: Fleet reference
description: Fleet configuration, plans, drift and recovery.
---

Source-derived fleet facts. The shipped TUI fleet capability remains preview-only unless a reviewed build enables live dispatch.

<!-- generated:reference/fleet:start -->
Source-derived fleet facts for the current checkout. Preview planning is read-only; apply and restore remain explicit write operations.

## `fleet.json` schema

Only schemaVersion 1 is accepted. The document is strict JSON, bounded to 1048576 bytes (1 MiB).

| Root field | Contract |
| --- | --- |
| schemaVersion | Must equal 1 |
| fleet | Optional fleet label; no path separators or traversal |
| preset | Optional preset label; no path separators or traversal |
| instances | Map of instance ID to instance object |

| Instance field | Contract |
| --- | --- |
| source | Required one-hop reference matching `^(agent\|preset):([a-z0-9][a-z0-9-]*)$` |
| model | Optional override matching `^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$` |
| enabled | Optional boolean; omitted or null defaults to true |

Instance IDs match `^[a-z0-9][a-z0-9-]{0,63}$`. Reserved instance names are `spec`, `plan`, `vision`.

- valid UTF-8 required
- duplicate JSON object keys rejected
- unknown fields rejected
- trailing data rejected
- JSON nesting depth limited to 32

## Model grammar

`^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$`

An empty or omitted model leaves the canonical source model untouched. The optional `#variant` suffix is part of the same model token, and empty path segments are rejected.

## Source references

- Allowed namespaces: `agent` and `preset`
- Walked source roots: `agents`, `adapters`, `presets/free-first`
- Exactly one hop is resolved; instance-to-instance aliases and cycles are rejected
- A name may exist in exactly one walked root; duplicate agent/adapter names fail resolution
- Unknown sources fail instead of rendering an empty agent

## Target and sidecar

| Item | Contract |
| --- | --- |
| Scope | `global` |
| Target root | `<home>/.config/opencode/agents` |
| Sidecar file | `fleet-manifest.json` inside the backup directory |
| Sidecar format | format 4 |
| Transaction states | `applying`, `applied`, `failed` |
| Per-file provenance | `managed`, `adopted`, `restored`, `legacy`, `unknown`, `removed` |

The sidecar fields are `format`, `id`, `operation`, `restoredFrom`, `state`, `createdAt`, `home`, `backupRoot`, `backupPath`, `targetRoot`, `scope`, `sourceRev`, `configDigest`, `rendererVersion`, `rollback`, `files`. Per-file records carry `instanceID`, `sourceRef`, `sourceDigest`, `configDigest`, `rendererVersion`, `model`, `desiredDigest`, `installedDigest`, `backupRef`, `backupDigest`, `scope`, `target`, `provenance`. Fleet never writes the user's `AGENTS.md`; names are bare `.md` file names under the target root.

## CLI

| Command | Flags | Required |
| --- | --- | --- |
| `fleet status` | `--config`, `--home`, `--repo` | `--config` |
| `fleet plan` | `--config`, `--output`, `--adopt`, `--home`, `--repo` | `--config` |
| `fleet apply` | `--plan`, `--home` | `--plan` |
| `fleet restore` | `--backup`, `--home` | `--backup` |
| `fleet doctor` | `--opencode-bin`, `--config` | none |

Documented exit codes are 0 (success), 1 (runtime, lock, stale plan or unmanaged conflict) and 2 (usage, flags, invalid configuration or malformed plan).

## TUI build capability

The Fleet Manager live controls are a build-time marker, not a user preference. Valid markers are `preview` and `live`; the shipped default is `preview`, and an absent, malformed or unknown marker fails safe to preview. The marker is injected with `-X agentic-sdd/internal/buildinfo.FleetCapabilityMarker`; there is no environment, config, CLI or key override.

## Separate contracts

The `fleet.json` schema (version 1) and the `fleet-manifest.json` sidecar (format 4) are distinct from the skill backup `manifest.json` formats 1-3 and from the MCP ownership store. They do not share a schema version.
<!-- generated:reference/fleet:end -->
