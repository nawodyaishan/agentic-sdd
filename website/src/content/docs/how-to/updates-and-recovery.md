---
title: Updates and recovery
description: Update skills and binaries, use embedded versus --repo backup stores, restore snapshots, roll back binaries and uninstall cleanly.
---

Keep installed skills current, switch between embedded and `--repo` backup roots safely, restore previous snapshots, roll back binaries, and remove Agentic-SDD without touching unrelated files.

## Update installed skills and binaries

**Upgrading the `asdx-core` / `asdx` binaries or pulling a `--repo` checkout does not update already-installed skills or OpenCode agent profiles.** You update the binaries (or checkout) first, then preview and apply the updated skills.

### 1. Check for a newer release (optional)

There is no `asdx-core update` command and no automatic self-updater (`internal/appops/update.go`).

- **TUI (`asdx`):** Checks `https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest` once per session (read-only HTTPS GET, 5s timeout, 64 KiB limit, redirects refused, no local inventory sent). Suppress with `AGENTIC_SDD_NO_UPDATE_CHECK=1`.
- **`asdx-core guide`:** Menu choice **5** runs the same bounded read-only check on demand and prints manual upgrade commands.

### 2. Upgrade the binaries

- **macOS (Homebrew cask):**

  ```sh
  brew update
  brew upgrade --cask nawodyaishan/tap/agentic-sdd
  ```

  If you previously installed an older Homebrew **formula** named `agentic-sdd` on older Homebrew, uninstall the formula first (`brew uninstall --formula agentic-sdd`) and then run `brew install --cask nawodyaishan/tap/agentic-sdd`.
- **Source build (Linux / macOS):** Check out the target tag or commit and rebuild the matched pair (`task -t Taskfile.experimental.yml tui:pair`) or CLI (`go build -trimpath -o bin/asdx-core ./cmd/agentic-sdd`). See [Installation](../getting-started/installation.md).

### 3. Preview and apply skill updates

Run `preview` with the same options you use for your installation (`--with-opencode`, `--opencode-free-first`, `--repo`, `--home`), inspect the diff, then run `apply` (or use `asdx` / `asdx-core guide`):

```sh
asdx-core preview
asdx-core apply
```

Every replaced skill or `.md` agent profile is backed up before overwriting.

## Backup and configuration stores

Where `asdx-core` looks for backups depends on `--repo` (`internal/skillsync/sync.go`):

| Source flag | Backup root | Operation logs |
|---|---|---|
| Embedded skills (no `--repo`) | `<home>/.agentic-sdd/backups/<UTC-ID>/` | `<home>/.agentic-sdd/logs/` |
| Explicit `--repo PATH` | `<repo>/backups/<UTC-ID>/` | `<home>/.agentic-sdd/logs/` |

> **Match `--repo` when listing or restoring:** `asdx-core backups` and `asdx-core restore` resolve the backup root from `--repo` and `--home`. If you installed with `--repo /path/to/checkout`, you must pass `--repo /path/to/checkout` to see or restore those backups. Logs always stay under `<home>/.agentic-sdd/logs/`.

Other Agentic-SDD subsystems keep **separate** backup and state stores; do not mix backup IDs or manifests across them (see [Files and backups](../reference/files-and-backups.md)):

- **Skill and `--with-opencode` backups (`internal/skillsync`):** `manifest.json` Formats 1–3, restored by `asdx-core restore` or TUI `Install & Manage` → `Backups`.
- **Fleet backups (`internal/fleet`):** `fleet-manifest.json` (Format 4) sidecar under `<home>/.agentic-sdd/backups/<UTC-ID>/` (never uses `--repo/backups`), restored only by `asdx-core fleet restore --backup ID`.
- **MCP entries (`internal/skillsync` MCP-B):** `<home>/.agentic-sdd/mcp/` (`manifest.json` and `backups/`) for managed server entries inside `<home>/.config/opencode/opencode.json` or `.jsonc`, restored only by `asdx-core mcp restore apply ID --expect-config SHA256`.
- **Execution run scratch (`internal/execution`):** `<repo>/.agentic-sdd/runs/<run-id>/`, inspected or removed with `asdx-core run status` and `asdx-core run clean --run ID --yes`.

## Restore from a backup

1. **List available backups** (newest first; add `--repo PATH` or `--home PATH` if used during install):

   ```sh
   asdx-core backups
   ```

2. **Preview the restore** (read-only; writes nothing):

   ```sh
   asdx-core restore BACKUP_ID
   ```

   For Format 2 and Format 3 backups, restore both puts back items recorded as `replaced` and **removes** items recorded as `installed` fresh by that apply (`internal/skillsync/restore.go`). Older `[legacy]` (Format 1) backups restore saved items best-effort and leave newly installed items in place. Restore refuses backups created for a different `--home` path and verifies SHA-256 digests before writing.

3. **Apply the restore:**

   ```sh
   asdx-core restore apply BACKUP_ID
   ```

   (`asdx-core restore BACKUP_ID --apply` is equivalent.) Before modifying any destination, `restore apply` saves a **new** backup recording `operation: "restore"` and `restored_from: BACKUP_ID`, and writes a log under `<home>/.agentic-sdd/logs/`, so a restore can itself be undone.

In an interactive terminal, `asdx-core restore --apply` with no `BACKUP_ID` lists backups, lets you pick by number, previews the plan, and then prompts `[y/N]` (default **No**); plain `restore` stops after the preview. In `asdx`, open **Install & Manage** (`2`) → **Backups** (or press `r` on Home) for the same preview-first flow.

## Uninstall Agentic-SDD

**There is no built-in `asdx-core uninstall` command.** Do **not** delete your home directory or entire client folders such as `~/.agents/skills`, `~/.claude/skills`, or `~/.config/opencode/agents`—they may hold your own skills, configurations, or custom profiles.

Follow these selective steps instead:

1. **Optional — undo the first install with `restore`:** If you still have the backup ID from your first Format 2/3 `apply` (which recorded every skill as `installed` or `replaced`), `asdx-core restore apply <INITIAL_BACKUP_ID>` removes the skills installed by that run and restores any prior copies. (Remember that `restore` itself creates one new restore backup and log under `<home>/.agentic-sdd/`, which step 4 covers.)
2. **Optional — revert Fleet or MCP entries first:** If you used `asdx-core fleet apply` or `asdx-core mcp apply`, restore those managed entries via `asdx-core fleet restore` or `asdx-core mcp restore` while your backups are still intact.
3. **Selectively remove installed skill directories and pack profiles:**
   - Remove only directories matching `agentic-sdd-*` inside each client destination you installed to:
     - `<home>/.agents/skills/agentic-sdd-*`
     - `<home>/.codex/skills/agentic-sdd-*`
     - `<home>/.claude/skills/agentic-sdd-*`
     - `<home>/.gemini/antigravity-cli/skills/agentic-sdd-*`
   - If you installed `--with-opencode`, remove only the shipped `sdd-*.md` agent files under `<home>/.config/opencode/agents/` (the 14 pack profiles plus the 4 Free-first preset files if you used `--opencode-free-first`). Leave `<home>/.config/opencode/AGENTS.md`, `opencode.json` / `opencode.jsonc`, and any custom agent files intact.
4. **Remove Agentic-SDD state directories (optional):** Backups can contain copies of your earlier skills, so inspect them before deleting if you might need them later. When ready, remove:
   - `<home>/.agentic-sdd/` (skill/fleet backups, MCP ownership state, and operation logs)
   - `<repo>/backups/` (if you used `--repo`)
5. **Remove the binaries:**
   - **macOS (Homebrew cask):** `brew uninstall --cask nawodyaishan/tap/agentic-sdd` (removes `asdx`, `asdx-core`, `agentic-sdd`, and `agentic-sdd-tui`).
   - **Source build:** Delete `asdx`, `asdx-core`, `agentic-sdd`, and `agentic-sdd-tui` from `bin/tui-pair/` or whichever `PATH` directory you placed them in.

## Roll back the binaries

Rolling back the binaries does **not** revert already-installed skill files; run `asdx-core preview` / `asdx-core apply` or `asdx-core restore` afterwards if you also want to revert skills.

- **From source:** Check out the previous Git tag or commit (for example `git checkout v0.11.0`) and rebuild the matched pair (`task -t Taskfile.experimental.yml tui:pair`) or CLI (`go build -trimpath -o bin/asdx-core ./cmd/agentic-sdd`). Always rebuild `asdx` and `asdx-core` together so their build-identity handshake matches.
- **From a GitHub Release archive (macOS):** Download the desired release archive and `checksums.txt` from the [GitHub Releases](https://github.com/nawodyaishan/agentic-sdd/releases) page, verify the archive SHA-256 against `checksums.txt` before extracting, and place all four binaries (`asdx`, `asdx-core`, `agentic-sdd`, `agentic-sdd-tui`) together as regular files in the same directory. On macOS, check Gatekeeper quarantine attributes if launching an unnotarized downloaded archive outside Homebrew.

## Next steps

- Inspect manifest fields and backup formats in [Files and backups](../reference/files-and-backups.md).
- Review all command flags in the [CLI reference](../reference/cli.md) and TUI screens in the [TUI reference](../reference/tui.md).
