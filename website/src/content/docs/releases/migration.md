---
title: Migration guide
description: Canonical and legacy command names, preserved state, upgrade, uninstall, and rollback.
---

This page is the sole owner of the canonical-to-legacy migration table. Other
pages link here instead of maintaining a second table. For what each release
and platform supports, see [compatibility](compatibility.md).

## Canonical and legacy names

The table below is the complete mapping. The prose around it defines the
terms; the table summarizes them.

| Command | Role | Status |
|---|---|---|
| `asdx` | Canonical terminal UI for interactive sessions | Use this for guided install and management |
| `asdx-core` | Canonical CLI for scripts, pipes, and CI | Use this for `preview`, `apply`, and automation |
| `agentic-sdd` | Legacy CLI wrapper, forwards to `asdx-core` | Compatibility wrapper for one deprecation window |
| `agentic-sdd-tui` | Legacy TUI wrapper, opens `asdx` | Compatibility wrapper for one deprecation window |

### Which one do I run: asdx-core vs asdx?

`asdx-core` is the CLI: non-interactive, scriptable, and the name to use in
CI and pipes. `asdx` is the terminal UI: interactive screens for install,
fleet, MCP, and browsing the guide. Both read and write the same state; the
difference is the interface, not the engine.

Wrapper behavior is deterministic: the wrappers are built from the same
sources as their canonical targets, resolve that target inside the same
installation without a shell or `PATH` search, and forward streams, signals,
and exit codes unchanged. Standard output, JSON output, and exit codes are
never modified; the deprecation notice goes to standard error only. Exact
top-level `--help`, `-h`, and `--version` on a legacy name identify the
compatibility name and its canonical target; subcommand help and JSON commands
are untouched. Bare `asdx-core` defaults to a read-only `preview`, as listed
in the [CLI reference](../reference/cli.md).

No wrapper-removal release has been announced. Wrappers remain for at least
one minor release with release notes; do not treat any existing release as
the removal one, and do not remove the wrappers on your own schedule.

## Preserved state and environment

The rename migrates no data. `~/.agentic-sdd/` (backups, logs, manifests)
remains readable across canonical and legacy names, and `AGENTIC_SDD_*`
environment switches keep working. There is no forced data-directory or
environment-variable migration, and state remains stable after the wrappers
are eventually removed. Backup and managed-path details live in the
[files and backups reference](../reference/files-and-backups.md).

## Upgrade

- On macOS, upgrade the cask release and then run `asdx` to preview and apply
  the update, or use `asdx-core preview` and `asdx-core apply` for scripts.
  Earlier formula installations migrate to the cask as documented in
  `README.md`; finish with whatever uninstall step Homebrew requests for the
  old formula, if any.
- On Linux, rebuild from the new source checkout; there is no prebuilt
  package to upgrade in place.
- After upgrading, verify with `asdx-core version` and a read-only
  `asdx-core preview` before applying anything. Both commands are listed in
  the [CLI reference](../reference/cli.md).

## Uninstall and rollback

- Uninstall removes only product-owned executables. It leaves
  `~/.agentic-sdd/` untouched, so backups and logs survive for a later
  reinstall. Delete that directory only if the backups inside it are
  deliberately no longer wanted. Full backup, restore, and recovery steps
  belong to `how-to/updates-and-recovery`; the exact managed paths are in the
  [files and backups reference](../reference/files-and-backups.md).
- To roll back, reinstall the previous release tag from the
  [GitHub releases page](https://github.com/nawodyaishan/agentic-sdd/releases)
  and confirm the installed version before use. Because state was never
  migrated, the older release reads the same `~/.agentic-sdd/` content.
