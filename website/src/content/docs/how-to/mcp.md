---
title: Manage MCP servers
description: Inspect, plan, apply and recover the three managed OpenCode MCP servers without treating configuration as proof of connectivity.
---

This guide walks through the MCP Tools surface for the three default servers:
CodeGraph (local repository code intelligence), Context7 (hosted library
documentation) and Exa (hosted web research). The CLI is `asdx-core mcp`; the
TUI path is **Install & Manage → MCP Tools**. Both use the same engine and the
same global OpenCode configuration.

**Configuration is not a connection.** Every command here reads or edits the
OpenCode configuration. None logs in to a provider, opens a socket, installs a
package or runs a model. A server can be configured and still be unauthenticated,
unreachable or broken; the tool reports what it inspected, not what a provider
would do at request time.

## Before you start

- **An existing global OpenCode configuration.** The managed target is
  `opencode.json` or `opencode.jsonc` under `<home>/.config/opencode/`. This
  build never creates either file.
- **OpenCode V2 for `apply`.** The write path runs a bounded local
  `opencode --version` probe and requires a major version of 2 or later. Pass
  `--opencode-bin PATH` if the binary is not on `PATH`. Inspection does not run
  it.
- **The CodeGraph CLI for a working CodeGraph server.** A missing `codegraph`
  executable is reported as `missing-prerequisite`; nothing is downloaded or
  installed. The tool looks up the binary and, when given `--project`, project
  index evidence; it does not run `codegraph init`.
- **An environment placeholder if you choose an API key.** The plan writes
  `{env:CONTEXT7_API_KEY}` or `{env:EXA_API_KEY}`; set the variable yourself in
  the environment OpenCode uses. OAuth is delegated to OpenCode's own credential
  store and is never performed by this tool.
- **A JSON target for writes.** A `.jsonc` target is inspected, but this build
  refuses to rewrite it because it cannot prove comment and formatting
  preservation. It prints a copyable snippet instead.

You can inspect an isolated home with `--home PATH`. Use that only for a
directory you intend to inspect or test.

## Inspect the current state

```sh
asdx-core mcp status
asdx-core mcp status --json
asdx-core mcp doctor
asdx-core mcp doctor --project . --json
```

`status` is the default when no subcommand is given. Both commands are
read-only and offline. `doctor` adds a fixed check list: target configuration,
schema, CodeGraph binary lookup, project-index evidence, Context7
authentication, Exa authentication, runtime version and remote connectivity.

Doctor checks report `pass`, `warn`, `blocked` or `not-checked`. A
`not-checked` check means no evidence was collected; it is not a pass. Useful
status lines:

| Line | Meaning |
| --- | --- |
| `Configuration: not configured` | Neither default file exists; nothing will be created |
| `Configuration: ambiguous` | Both default files exist; no target is chosen |
| `Format: jsonc` | Inspection works; this build will not write or restore it |
| `Schema: v1` | Legacy `mcp.<name>` entries; refused and never migrated |
| `prereq: codegraph CLI (missing)` | Install the CLI yourself; no installer runs |
| `Not checked: remote connectivity, authentication, runtime version` | No network or login was attempted |

## Plan the change

```sh
asdx-core mcp preview
asdx-core mcp preview --servers codegraph,exa
asdx-core mcp preview --servers context7 --auth context7=api-key
asdx-core mcp preview --servers exa --auth exa=oauth --json
```

Selection defaults to all three servers. The plan prints the exact target,
format, schema, base digest, plan digest, one entry per selected server and any
conflicts. Operation tokens are `add`, `change`, `leave`, `disabled` and
`conflict`; restore views also use `restore` and `remove`.

Review these before applying:

- **`leave`** — the entry already matches the managed recipe; nothing to write.
- **`add`** — the managed entry is absent and would be added.
- **`change`** — an entry the ownership manifest proves Agentic-SDD installed
  differs from the current recipe. An entry you created yourself is not changed.
- **`disabled`** — an existing `disabled: true` server is preserved and is never
  re-enabled by this surface.
- **`conflict`** — the plan is not applicable. A conflicted preview exits `1`,
  and `apply` refuses. Version 1 has no replace, adopt or force option.

Authentication choices are explicit:

| Server | Accepted `--auth` values | Default | Written credential |
| --- | --- | --- | --- |
| `codegraph` | none | none | none (local command) |
| `context7` | `oauth`, `api-key` | `oauth` | `{env:CONTEXT7_API_KEY}` for API key |
| `exa` | `keyless`, `oauth`, `api-key` | `keyless` | `{env:EXA_API_KEY}` for API key |

`exa=keyless` is rate-limited; the plan does not upgrade it automatically. No
plaintext key is written. An existing managed entry containing a literal
credential is refused before any diff is rendered, and the value is not echoed.

## Apply the reviewed plan

```sh
asdx-core mcp apply --expect-config <base-digest-from-preview>
asdx-core mcp apply --expect-config <base-digest-from-preview> --opencode-bin /path/to/opencode
```

`apply` rebuilds the plan from the current files, then refuses unless:

1. the plan has no conflicts;
2. the target exists and is a regular `.json` file in the global scope;
3. `--expect-config` equals the current base digest;
4. the shared engine lock is acquired. A lock older than 30 seconds is reclaimed
   once as abandoned;
5. under the lock the file is re-read, the digest must still match, and a
   bounded local `opencode --version` probe reports V2 or later.

The digest is the coordination point: copy it from the preview you reviewed. An
empty or mismatched digest writes nothing. The command does not prompt; the
explicit digest and the revalidated plan are the confirmation.

A write creates a private entry-level backup under
`<home>/.agentic-sdd/mcp/backups/<id>/`, writes an intent record, replaces the
target atomically and records the ownership manifest. Only managed entries are
saved; unrelated servers, comments elsewhere in the file and credentials are
not copied into the backup. A second identical apply reports a no-op, creates no
backup and does not run the write path.

See the [MCP reference](../reference/mcp.md) for the manifest fields,
transaction phases and JSON result keys.

## Recover a managed entry

```sh
asdx-core mcp restore list
asdx-core mcp restore preview <backup-id>
asdx-core mcp restore apply <backup-id> --expect-config <digest-from-restore-preview>
```

The MCP store is separate from the skill backup store and the fleet manifest;
restoring one never restores another. Restore is **entry-level**:

- it compares only the entries recorded in that backup;
- a managed entry changed after the apply is a conflict, not an overwrite;
- an entry you removed is not silently recreated;
- an unrelated server or field is left alone;
- a whole-file restore and a missing-file recreation are not part of this build.

`restore apply` is interactive by default and asks `[y/N]`; only `y` or `yes`
proceeds, and every other answer writes nothing. A non-interactive run has no
prompt but still needs the backup id and the digest from `restore preview`. A
stale digest, a missing target, a conflict or a JSONC backup refuses with exit
`1`.

If an apply reports `Rollback: INCOMPLETE`, the managed entries may be partially
written. Keep the backup path from the output, re-run `status` and `preview` to
see the current state, and do not assume a retry repairs it. Restore the
recorded backup only after reviewing its preview.

## What the offline checks do not prove

`status`, `doctor` and `preview` do not prove:

- that a remote server is reachable or that a request would succeed;
- that Context7 or Exa authentication is configured, valid or accepted;
- that an OAuth flow completed — no flow is started;
- that the CodeGraph index is fresh or that its MCP server is connected;
- that `codegraph` or `opencode` has a particular version. The only version
  probe is the bounded local `opencode --version` check on a planned apply.

`doctor --network` and `doctor --connect` are rejected as unavailable in this
build. Treat any green-looking state as configuration evidence only.

## TUI route

Open **Install & Manage → MCP Tools** (or press `t` in the Install & Manage
hub) to see the same three servers and the same read-only diagnostics and
preview. `s` selects servers, `v` previews, `d` opens diagnostics, `e` opens
recovery, and `Esc` goes back. The TUI writes through the same engine and the
same digest checks; it has no separate network or OAuth test.

For command flags and exits, see the [MCP reference](../reference/mcp.md) and
the MCP family entry in the [CLI reference](../reference/cli.md). Managed paths
are listed in [Files and backups](../reference/files-and-backups.md); target
resolution is summarized in [Configuration](../reference/configuration.md).
