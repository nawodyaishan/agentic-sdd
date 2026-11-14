---
title: Install and manage skills
description: Preview, install and update core skills across supported clients using the CLI, TUI client selection or guided wizard.
---

Install or update the 18 core Agentic-SDD skills using the actual entrypoints available in `asdx-core` and `asdx`.

## Core unit and entrypoints

Agentic-SDD installs the **18 core skills** (`skills-files/`) as a complete unit across any selected destination—there is no per-skill install flag. See the [Skill catalog](../reference/skills.md) for the full list and [Supported clients](../introduction/clients.md) for the four destination paths (`agents`, `codex`, `claude`, `agy`).

How you select clients and defaults depends on which entrypoint you run:

| Entrypoint | Client selection | `--with-opencode` default | Behavior |
|---|---|---|---|
| `asdx-core` (CLI) | **All four clients** (`--clients` does not exist on the CLI) | Off (core skills only) | Previews by default; writes only on `apply` or `--apply`. |
| `asdx-core guide` | **All four clients** | Prompts in wizard | Numbered interactive prompts on a real TTY; writes only after confirmation (default **No**). |
| `asdx` / `asdx-core tui` (TUI) | **Per-client checkboxes** in Install & Manage | On (`--with-opencode=false` disables) | Full-screen terminal UI on a real TTY; writes only after `y` confirmation (default **No**). |

If you are unsure whether you want the optional OpenCode orchestration pack (`--with-opencode` / `--opencode-free-first`), read [Choose your setup](../getting-started/choose-setup.md) first.

## Install or update all four clients with the CLI

1. **Preview changes first (read-only):**

   ```sh
   asdx-core preview
   ```

   To read skills from a local repository checkout instead of the binary's embedded copy, add `--repo /path/to/agentic-sdd`. To target a separate home directory, add `--home /path/to/home`. For the full `--repo` layout rules and model-override boundaries, see [Customization and local sources](./customization.md).

2. **Review the plan:** Check every `install`, `replace + back up`, and `up to date` line. Nothing has been written yet.

3. **Apply the changes:**

   ```sh
   asdx-core apply
   ```

   Pass the same `--repo` or `--home` flags if you used them in `preview`.

**Effects and safety guarantees (`internal/skillsync/sync.go`):**

- **Backups before replacement:** Every existing `agentic-sdd-*` directory that differs from source is copied to `<home>/.agentic-sdd/backups/<UTC-ID>/` (or `<repo>/backups/<UTC-ID>/` when `--repo` is set) with a `manifest.json` SHA-256 digest before anything in the destination is replaced.
- **Atomic staging:** Each skill is staged in a temporary directory beside its destination and moved into place; partial failures roll back already-renamed skills.
- **No-op when unchanged:** Re-running `asdx-core apply` when files match prints `All skills are up to date.` and writes no new backup.
- **Unrelated skills untouched:** Only `agentic-sdd-*` directories are managed. Any symlinks or special files in a target path or parent directory cause the installer to refuse before writing.
- **Operation log:** `apply` writes a timestamped log under `<home>/.agentic-sdd/logs/` (even when `--repo` is used).

## Select specific clients in the TUI

Only the packaged TUI (`asdx`, or its launcher `asdx-core tui`) lets you install to a subset of the four supported clients.

1. Launch `asdx` in an interactive terminal. Pass `--with-opencode=false` if you want the core-only launch configuration, and `--home PATH` or `--repo PATH` if needed:

   ```sh
   asdx --with-opencode=false
   ```

   Note that on interactive startup, the Home screen performs one bounded release check against GitHub unless `AGENTIC_SDD_NO_UPDATE_CHECK=1` is set (see the [TUI reference](../reference/tui.md)).

2. Open **Install & Manage** (`2`), then select **Install**.
3. **Clients step:** Use `↑`/`↓` (`k`/`j`) to highlight a client (`agents`, `codex`, `claude`, `agy`), press `Space` to toggle selection, and press `Enter` to continue. At least one client must remain selected.
4. **Options step:** Confirm whether to include the optional OpenCode pack and Free-first preset (`Space` to toggle, `Enter` to continue).
5. **Review and Preview steps:** Inspect the resolved source, home, selected destinations, and planned changes (`Enter` advances from Review to Preview, then from a non-empty Preview to Confirm).
6. **Confirm step:** The default selection is **No**. Press `y` (never `Enter`) only if you accept the previewed writes, or press `n` / `Esc` / `b` to go back without writing.

> **Manifest identity note:** Even when you select a subset of clients in the TUI, `manifest.json` still records all four canonical client paths for `--home` in `targets`, while `entries` lists only the selected destinations that changed (`internal/skillsync/sync.go`).

## Guided terminal fallback

If `asdx` is not built or you prefer numbered text prompts on a real TTY:

```sh
asdx-core guide
```

Choose option **1** (preview and install), pick core-only or `--with-opencode` options, inspect the full preview paths and counts, and answer `[y/N]` (default **No**). Like `asdx-core apply`, the wizard manages all four client destinations.

## Verify installed skills and logs

- Run `asdx-core preview` (with matching `--home` / `--repo` / `--with-opencode` flags). Every managed destination should report `up to date`.
- Run `asdx-core logs` (or `asdx-core logs --home PATH`) to view the latest apply transcript and backup path.
- In `asdx`, open **Explore** (`3`) to inspect per-destination status across the catalog.

## Next steps

- Learn how to update skills, restore backups, and clean up installations in [Updates and recovery](./updates-and-recovery.md).
- Inspect backup formats and manifest details in [Files and backups](../reference/files-and-backups.md).
- Check full CLI and TUI options in the [CLI reference](../reference/cli.md) and [TUI reference](../reference/tui.md).
