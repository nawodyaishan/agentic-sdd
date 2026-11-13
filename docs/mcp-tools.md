# MCP Tools: CodeGraph, Context7 and Exa

The MCP Tools surface manages the configuration that makes three default
tool servers available to OpenCode: **CodeGraph** (local repository code
intelligence), **Context7** (hosted library documentation) and **Exa**
(hosted web research). It lives in two places that share one engine:

* the terminal UI: **Install & Manage → MCP Tools** (or press `t` from the
  Install & Manage hub);
* the CLI: `asdx-core mcp status|doctor|preview|apply|restore|help`.

Both read the same facts and both refuse the same unsafe operations. Nothing
here contacts a model, reads a provider credential, or changes which agent
may use a tool — configuring a server only makes it *available* to OpenCode.
Feature 009 remains the authority on agent permissions.

**Scope of this build:** MCP-A (read-only status, offline doctor, structured
preview), MCP-B (digest-checked apply and managed restore) and MCP-C (the TUI
MCP Tools child). MCP-D covers verification and this documentation. There is
no software installation, no OAuth login, no network connection test and no
release action anywhere in this surface. JSONC apply is refused with a manual
snippet rather than reserialized.

## The beginner journey

1. **Discover.** Open Install & Manage → MCP Tools, or run
   `asdx-core mcp status`. You see the three servers, whether each is
   configured, what is missing, and the next safe action. Nothing is written.
2. **Select.** In the TUI, press `s` on the Overview to pick servers.
   Context7 asks for OAuth (recommended, explicitly accepted) or an API key;
   Exa asks you to acknowledge its keyless rate limits and privacy terms, or
   to choose OAuth/API key instead.
3. **Preview.** Press `v` (or run `asdx-core mcp preview`). You see the
   exact target file and digest, which entries would be added, changed, left
   alone or refused as conflicts, and what backup and restore would cover.
   Previewing writes nothing.
4. **Confirm.** A separate confirmation screen defaults to **No**. Accepting
   applies the change through one shared write slot and reports the real
   result plus the backup identity.
5. **Restore.** Recovery lists only backups Agentic-SDD itself made. A
   restore previews the exact current-versus-restore effects and needs its
   own default-No confirmation.

## CLI equivalents

```sh
asdx-core mcp status                              # read-only overview (default; bare `mcp` also works)
asdx-core mcp doctor                              # offline prerequisite diagnostics, no execution
asdx-core mcp preview --servers codegraph,exa     # read-only plan; nothing is written
asdx-core mcp preview --servers context7 --auth context7=oauth
asdx-core mcp preview --servers exa --auth exa=keyless
asdx-core mcp apply --expect-config SHA256        # write the reviewed preview (digest required)
asdx-core mcp restore list                        # list managed backups
asdx-core mcp restore preview ID                  # preview one managed backup
asdx-core mcp restore apply ID --expect-config SHA256
asdx-core mcp help                                # this reference
```

Useful flags: `--home PATH` (inspect an isolated home instead of your own),
`--config PATH` (inspect an explicit file), `--project PATH` (project
prerequisite checks), `--json` (stable script output), `--opencode-bin PATH`
(explicit local binary for the apply-time runtime probe). Exit codes follow
the fleet convention: `2` usage/validation, `1` stale/conflict/runtime, `0`
report or real no-op.

## Read-only first (MCP-A)

`status`, offline `doctor` and `preview` create, modify and delete no file.
They invoke no model, package manager, vendor installer or remote endpoint,
and they execute no binary — prerequisite reporting uses a local executable
lookup only. A server that is merely configured is never labeled connected,
authenticated or working: without an explicit check the state reads
`authentication not checked`, `connection not tested` or `unknown`.

`doctor --network` and `doctor --connect` are deferred to a separately
approved capability and are rejected. The TUI has no connection test at all
in v1; Diagnostics shows `connection not tested` with CLI guidance.

## Applying and restoring (MCP-B)

Apply is opt-in and digest-checked:

* it requires `--expect-config` with the exact digest the preview printed;
  an empty value is a usage error and any mismatch refuses with no write;
* it runs a bounded local `opencode --version` probe (explicit
  `--opencode-bin` or one resolved executable — no shell, no network, no
  model) and fails closed on a missing or unsupported runtime;
* a shared engine lock covers the target and the ownership manifest while a
  competing writer is refused;
* a private backup lands under `<home>/.agentic-sdd/mcp/backups/<id>/`
  (mode `0700`/`0600`) holding only the managed-entry text plus metadata —
  never unrelated servers or their credentials — beside a separate ownership
  manifest;
* a second identical apply reports `up to date`, creates no second backup
  and leaves the file bytes unchanged.

Restore touches only entries Agentic-SDD manages. Later edits to unrelated
fields, unrelated servers or a managed entry are preserved or reported as a
conflict — never silently erased. Whole-file restore is not part of v1.

## Global configuration only; nothing is created

The only managed target is the global OpenCode V2 configuration
(`~/.config/opencode/opencode.json`, or `.jsonc` if that is the only file).
Project-local files, system files and managed-preference layers are never
written. When neither `.json` nor `.jsonc` exists, the tool reports
`not configured` and prints a copyable manual snippet — it does not create
a file, a directory or a `--create` path. When both exist, it reports an
ambiguity and refuses rather than guessing. A V1 `mcp.<name>` shape is
reported as a schema mismatch with migration guidance, never silently
migrated.

## JSONC, conflicts and unsafe targets

* **JSONC** files are edited only as byte ranges when the tool can prove it
  owns every touched comment and delimiter; otherwise it refuses and prints
  the exact manual snippet. Comments and formatting are never silently
  discarded by reserialization.
* **Unmanaged conflicts** always refuse in v1: an unrelated `mcp.servers`
  entry, an unknown top-level field, an unknown field inside an unrelated
  entry and an existing `disabled: true` server all survive preview and
  apply unchanged, and a disabled server is never re-enabled. There is no
  `--replace` or force flag.
* **Unsafe targets** — symlinks, non-regular files, traversal paths, paths
  outside the global scope, changed modes, special files — are rejected
  without reading or writing outside the intended target.
* **Credentials** are placeholders, not values: managed remote entries use
  `{env:NAME}` (for example `{env:CONTEXT7_API_KEY}`) and OAuth stays in
  OpenCode's own store. A managed entry containing a literal secret is
  refused before any diff is rendered, and the value is never echoed, logged
  or copied into a backup.

## CodeGraph prerequisite and index

The Overview keeps five CodeGraph facts separate: CLI installed, entry
configured, project index initialized, index freshness, and connection
tested. Ordinary status and offline doctor execute nothing, so freshness
reads `freshness not checked` and the connection reads `not tested`.

* A missing `codegraph` binary is reported as `missing prerequisite` with
  supported guidance. Nothing is downloaded or installed.
* A missing project `.codegraph/` index is reported as `index not
  initialized`. Nothing runs `codegraph init` (or `init -y`) during global
  MCP setup, and project initialization never targets a home directory, the
  filesystem root, `backups/`, private vaults or unrelated repositories.
* `codegraph status` is an index check, not proof that the MCP server is
  connected.

## Context7 and Exa honesty

* **Context7:** OAuth is the recommended route and only applies after you
  explicitly accept it in the plan; the API-key route is a separate choice
  using an environment placeholder. No login is performed and no preference
  is stored.
* **Exa:** keyless is the default only after you explicitly acknowledge the
  visible rate limits and privacy terms. Keyless is rate-limited, not
  unlimited free use; OAuth or an API key are optional and user-controlled.
  Nothing automatically upgrades Exa to an authenticated or paid route.

Availability checks never transmit private source code, credentials or
unrelated repository contents.

## TUI keys

From the Install & Manage hub: `t` opens MCP Tools (`f` remains Fleet
Manager, `i`/`b` remain Install/Backups). Inside MCP Tools:

| Key | Action |
|---|---|
| `j`/`k`, arrows | Move the cursor |
| `s`, Enter (Overview) | Open server selection |
| `v` (Overview/Select) | Open the read-only configuration preview |
| Enter (Preview) | Open the default-No confirmation |
| `d` (any view except Confirm) | Open Diagnostics |
| `e` (Overview) | Open Recovery |
| Esc | Back one step: Confirm → Preview → Select → Overview → hub |
| `q` | Quit |

Resizing below 60 columns or 8 rows shows guidance instead of an
affirmative action, and a resize resets any confirmation to **No** — a
resize can never confirm a write. Footer text always shows the keys for the
current view, and `?` opens help.

## What this guide does not promise

No network or handshake check, no OAuth login, no package installation, no
`codegraph init`, no project-local management, no `--create`, no
`--replace`/force, no general uninstaller, no `mcp install-missing`, no
`opencode launch --free`, and no release or real-home operation. Those are
separately approved capabilities or explicit deferrals, not hidden features.
See [Feature 015 spec](../specs/015-fleet-manager-tui/spec.md) §16 for the
governing requirements and acceptance criteria.
