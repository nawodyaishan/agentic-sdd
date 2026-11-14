---
title: MCP reference
description: MCP server configuration, preview and apply behavior, ownership manifests, diagnostics, and recovery for optional tool integrations.
---

Curated contracts for the `mcp` family. Configuration does not authenticate,
connect or prove that a server works. The generated inventory below is the
compact summary; this section adds the decision points a maintainer needs
before running a write.

Sources: command dispatch and renderers in `cmd/agentic-sdd/mcp.go` and
`cmd/agentic-sdd/mcp_write.go`; target validation, planning, transaction and
ownership manifest in `internal/skillsync/mcp.go`,
`internal/skillsync/mcp_config.go` and `internal/skillsync/mcp_transaction.go`;
typed operation seams in `internal/appops/mcp.go` and
`internal/appops/mcp_write.go`.

## Command surface

| Command | Writes | Contract |
| --- | --- | --- |
| `asdx-core mcp` / `status` | no | Default when no subcommand is given. Reports target, format, schema, digest and the three server states. |
| `asdx-core mcp doctor` | no | Adds eight offline checks. `--network` and `--connect` are rejected as deferred (exit `2`). |
| `asdx-core mcp preview` | no | Builds the digest-bound plan and token; exits `1` when the plan has conflicts. |
| `asdx-core mcp apply` | yes | Requires `--expect-config`; rewrites managed entries only after revalidation and a local runtime probe. |
| `asdx-core mcp restore list` | no | Lists backups from the MCP store only. |
| `asdx-core mcp restore preview ID` | no | Compares the current file with one backup, entry by entry. |
| `asdx-core mcp restore apply ID` | yes | Requires `--expect-config`; restores recorded managed entries only. |
| `asdx-core mcp help` | no | Family help. This is authoritative over the shorter root `--help` summary; see [CLI reference](./cli.md). |

| Flag | Applies to | Contract |
| --- | --- | --- |
| `--home PATH` | all | Resolves every state path and the default target. Defaults to the OS user home. |
| `--config PATH` | all (`restore list` parses but ignores it) | Must end in `.json` or `.jsonc`. For plan/apply/restore it must also stay under `<home>/.config/opencode`; an explicit missing path is a missing target, never a create request. |
| `--project PATH` | `status`, `doctor` | Project path for CodeGraph index evidence. |
| `--servers NAME,...` | `preview`, `apply` | Default is all three. Unknown names are usage errors; output follows the canonical `codegraph`, `context7`, `exa` order. |
| `--auth SERVER=MODE` | `preview`, `apply` | Repeatable. Values are validated even for an unselected server. |
| `--expect-config SHA256` | `apply`, `restore apply` | Base digest copied from the reviewed preview. Empty or mismatched values never write. |
| `--opencode-bin PATH` | `doctor`, `apply` | Doctor looks up only; apply executes a bounded `--version` probe when a write is planned. |
| `--json` | all | One indented document with lower-case keys. Preview entry text is limited to managed entries; no credential value is emitted. |

## Target and schema

The only supported write target is the global OpenCode configuration.
Project-local scope is deferred, and a target outside
`<home>/.config/opencode` is refused as unsafe for plan/apply/restore.

- Default target: `opencode.json`, or `opencode.jsonc` when it is the only one
  of the two files.
- Both default files present: ambiguity conflict; neither is chosen.
- Neither default file: missing target; no file or directory is created.
- The target must be a regular file, not a symlink or special file. Reads are
  bounded to 1048576 bytes, require valid UTF-8, and reject duplicate keys,
  trailing data and a non-object top level.
- An explicit missing `.json` or `.jsonc` target is reported as missing;
  `apply` and `restore apply` refuse instead of creating it.
- A `.jsonc` target can be inspected, but this build refuses to rewrite it
  because it cannot prove comment and formatting preservation. The plan exits
  `1` with a copyable snippet.

Status schema values are `none` (no usable configuration), `v2`
(`mcp.servers`), and the error paths `v1` (legacy `mcp.<name>`), `invalid` and
`unknown`. Legacy V1 is reported as a schema mismatch; it is never migrated or
rewritten. JSON and JSONC are parsed separately; comments are not silently
discarded.

## Server schema and authentication

Only `mcp.servers.<name>` entries for `codegraph`, `context7` and `exa` are
managed. Unrelated servers and top-level keys are carried through unchanged.
For a managed entry, this build validates:

| Field | Contract |
| --- | --- |
| `type` | `local` or `remote` when present |
| `command` | For `local`, a non-empty array of strings |
| `url` | For `remote`, a non-empty string |
| `disabled` | Boolean; `true` is preserved and never re-enabled |
| `headers` | Object of string values |
| `oauth`, `codemode` | Boolean fields written by the managed recipes |

Credentials are never written as values. `--auth` selects a recipe whose
headers use `{env:NAME}` placeholders; OpenCode owns any OAuth credentials. A
literal credential in a managed entry's `headers` or `environment` is a
conflict: the plan refuses before rendering a diff, and the value is not echoed.
Unrelated entries are not scanned for credentials and are not rewritten.

| Server | Kind | Accepted auth | Default | Placeholder |
| --- | --- | --- | --- | --- |
| `codegraph` | local | none | none | none |
| `context7` | remote | `oauth`, `api-key` | `oauth` | `{env:CONTEXT7_API_KEY}` for API key |
| `exa` | remote | `keyless`, `oauth`, `api-key` | `keyless` | `{env:EXA_API_KEY}` for API key |

`exa=keyless` is rate-limited; no command upgrades it automatically. Supplying
an auth choice for `codegraph` is a usage error.

## Plan and JSON states

Plan operations are `add`, `change`, `leave`, `disabled` and `conflict`; the
same vocabulary also defines `restore`, `remove` and `missing` for entry-level
backup views. Status states are
`not-configured`, `configured`, `disabled`, `missing-prerequisite`,
`auth-not-checked`, `conflict`, `schema-mismatch`, `unsafe-target` and
`unknown`. Doctor checks report `pass`, `warn`, `blocked` or `not-checked`;
`not-checked` is missing evidence, not a pass. Server evidence is `static`, or
`local-probe` for the CodeGraph prerequisite lookup.

`--json` document identities:

| Command | Top-level keys |
| --- | --- |
| `status` | `command`, `mode` (`offline`), `home`, `config`, `servers`, `codegraph_binary`, `codegraph_version`, `opencode_version`, `not_checked` |
| `doctor` | status keys plus `checks` and `offline: true` |
| `preview` | `command`, `mode` (`read-only`), `home`, `target`, `format`, `schema`, `base_digest`, `plan_digest`, `selected`, `auth`, `entries`, `conflicts`, `no_op`, `runtime_version`, `runtime_evidence`, `not_checked`, `token`, `preview_only: true`, `write: none` |
| `restore list` | `command`, `mode`, `home`, `count`, `backups` |
| `restore preview` | `command`, `mode`, `home`, `backup_id`, `target`, `base_digest`, `entries`, `conflicts`, `token`, `preview_only`, `write` |
| `apply` / `restore apply` | `command`, `mode` (`write`), `home`, `target`, `base_digest`, `plan_digest`, `no_op`, `applied`, `backup_id`, `backup_path`, `installed_digest`, `runtime_version`, `rollback` |

Each plan entry carries `name`, `op`, `reason`, `old_text`, `new_text`,
`old_hash` and `new_hash`. The write result's `rollback` object carries
`attempted`, `complete` and `error`.

## Apply preconditions and transaction

`apply` rebuilds the plan from the current file; a token from an earlier process
is not trusted. Before any write it requires:

1. selected names and auth values are valid;
2. the plan has no conflicts and no managed entry contains a literal credential;
3. the target is an existing regular `.json` file in the global scope;
4. `--expect-config` equals the current base digest;
5. the shared engine lock is acquired. A lock older than 30 seconds is treated
   as abandoned and reclaimed once;
6. under the lock the file is re-read, the digest must still match, and a
   bounded local `opencode --version` probe reports major version 2 or later
   (5-second timeout, no shell, no network).

A concurrent edit between preview and lock is a stale plan, not a lost update. A write saves an entry-level
backup under `<home>/.agentic-sdd/mcp/backups/<id>/`, writes `intent.json`,
replaces the target atomically and records the ownership manifest. Rollback
restores only the recorded managed entries; an incomplete rollback is reported
as `INCOMPLETE` and must not be read as success. A no-op apply creates no backup
and does not run the runtime probe.

## Restore preconditions

Restore uses the MCP ownership backup only; it never reads the skill backup
store or the fleet sidecar. `restore preview` compares recorded entries with the
current file and exits `1` on conflicts. `restore apply` additionally requires:

1. an existing backup ID whose characters are limited to `A-Z`, `a-z`, `0-9`,
   `.`, `-` and `_`;
2. the recorded target still equals the current target and exists;
3. a JSON target; a backup of a JSONC target is not restorable by this build;
4. every managed entry still matches the installed hash recorded at apply. A
   user edit, a missing entry the user removed, or a non-object entry is a
   conflict with no force option;
5. `--expect-config` equals the current base digest;
6. interactive confirmation by `y` or `yes`; every other answer, including end
   of input, writes nothing. A non-interactive run has no prompt but still needs
   the id and digest.

Restore is entry-level. It does not restore unrelated entries, does not
recreate a missing configuration and is not a whole-file rollback.

## Exits

| Exit | Meaning |
| --- | --- |
| `0` | A report was produced, a real no-op, or a completed apply/restore |
| `1` | Stale digest, conflict, unsupported runtime or schema, incomplete rollback, or engine/I/O error |
| `2` | Usage, flag, validation, unsafe target, unknown subcommand, or a deferred `doctor --network`/`--connect` flag |

A conflicted `preview` is a report and an error: it prints the plan and exits
`1`. A `status` or `doctor` report about a missing configuration still exits
`0` when the report itself was produced.

## Limits

- No network request, OAuth flow, package installation, model call or MCP
  server start. The only executed binary is the bounded local
  `opencode --version` probe on a planned apply.
- No configuration creation, no replace/adopt/force option, no project-local
  scope and no arbitrary target guarantee.
- No JSONC edit. No comment-preserving rewrite is promised.
- No whole-file restore and no restore of a backup whose target has moved.
- Offline status, doctor and preview do not prove connectivity, authentication,
  OAuth completion or CodeGraph index freshness. See the how-to for the
  operational sequence.
- Synthetic fixture shapes live under `website/tests/examples/mcp/`; they are
  inert documentation fixtures, not a supported target and not captured run
  output.

For the walkthrough, see [Manage MCP servers](../how-to/mcp.md). Managed paths
and stores are summarized in [Files and backups](./files-and-backups.md); the
flag inventory for every family is in [CLI reference](./cli.md).

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
