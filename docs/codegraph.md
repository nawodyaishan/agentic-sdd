# CodeGraph setup and continuous sync

CodeGraph is the repository's default code-discovery index. It gives agents and
maintainers one-call symbol source, call paths and blast-radius context without
a grep/read loop. It is local tooling: it does not change product behavior,
approve work or authorize a release.

## Prerequisites

- The `codegraph` CLI is installed and on `PATH`.
- The CodeGraph MCP server is configured in the active client if MCP tools are
  available. MCP configuration is machine-specific and stays out of portable
  skill text.

Check the current state:

```sh
command -v codegraph
codegraph status
codegraph status --json
```

`.codegraph/` is local state. It self-ignores and the repository `.gitignore`
also excludes it. Never commit the database, WAL, daemon socket or logs.

## Bootstrap when the index is absent

Run this only in an ordinary project checkout — not in a home directory, a
filesystem root, `backups/`, a private vault or an unrelated repository.

```sh
codegraph init -y
codegraph status --json
```

`-y` is non-interactive and takes the defaults. If initialization fails, keep
the exact error, do not delete an existing database blindly, and continue with
targeted `Read`/`Grep`. If the CLI is missing, do not install it silently;
report the missing prerequisite and use the fallback.

## Daily use

1. **Before searching or editing indexed code:** call the MCP
   `codegraph_explore` tool with the symbol, file or question. Treat its source
   as already read. Fall back to `codegraph explore`, `codegraph node`,
   `codegraph callers` or `codegraph impact` if MCP is unavailable.
2. **While editing:** use the returned call paths and blast radius to choose the
   files and tests to update. Do not re-grep code CodeGraph already returned.
3. **After a bulk edit, branch switch or before review/commit:** sync and check
   status.

```sh
codegraph sync
codegraph status
```

A normal MCP watcher indexes changes within about a second. If a CodeGraph
response carries a staleness banner, `Read` the flagged files before relying on
them. If auto-sync is disabled or recovering, read files directly until the
index catches up.

## Troubleshooting

| Symptom | Action |
|---|---|
| `codegraph status` says no index | `codegraph init -y`, then status |
| Stale lock blocks indexing | `codegraph unlock`, then `codegraph sync` |
| Index is corrupt or repeatedly inconsistent | `codegraph index` to rebuild from scratch |
| Background daemon is wedged | `codegraph daemon` to inspect/stop it, then retry |
| MCP tools unavailable but CLI works | Use CLI `explore`/`node`/`callers`/`impact` and sync before review |
| Neither MCP nor CLI available | Use targeted `Read`/`Grep`; say the index was unavailable; do not claim CodeGraph evidence |

## Continuity with MCP

Keep the MCP server and CLI pointed at the same checkout. The MCP server exposes
the current index; the CLI is the bootstrap, sync and recovery path. After a
large refactor, run `codegraph sync` before asking for a review so the reviewer
sees the new symbols rather than a stale graph. Do not index unrelated private
repositories, credentials or secret-bearing files. CodeGraph queries must never
contain secrets, tokens, private paths or private code from another project.

## Scope boundary

CodeGraph is infrastructure. It is not a source of human approval, a batch
ledger, a test result or release authorization. A green index proves only that
the local graph was built; it does not prove the code is correct.
