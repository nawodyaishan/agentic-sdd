# Installer reference

The `agentic-sdd` CLI installs the portable skill toolkit. It does not run your coding agent. For installation and your first task, start with the [README](../README.md#quick-start).

## Commands

| Command | Behavior |
|---|---|
| `agentic-sdd` or `agentic-sdd preview` | Show planned installs, replacements, and unchanged skills without writing. |
| `agentic-sdd apply` | Back up existing skills that need replacement, stage copies, and install. Identical skill content is a no-op. |
| `agentic-sdd version` | Show version, commit, build date, and Go version. `--version` also works. |
| `agentic-sdd logs` | Show the latest apply or restore log and its path. |
| `agentic-sdd backups` | List backups newest first, including their ID, operation, source, and contents. |
| `agentic-sdd restore BACKUP_ID` | Preview what restoring the selected backup would change. |
| `agentic-sdd restore apply BACKUP_ID` | Back up the affected current skills, then restore the selected backup. |
| `agentic-sdd help orchestrate` | Show usage for opt-in OpenCode orchestration options. |
| `agentic-sdd --help` | Show commands, flags, and defaults. |

`restore preview BACKUP_ID` is another preview form; `restore BACKUP_ID --apply` is another writing form. With no backup ID on a terminal, restore lists available backups and offers interactive selection. Writing through interactive selection asks for confirmation.

## Flags

Put flags after the command, for example:

```sh
agentic-sdd preview --home /path/to/disposable-home
```

| Flag | Meaning |
|---|---|
| `--home PATH` | Select the home directory containing the four client skill destinations; defaults to your home. |
| `--repo PATH` | Read skills from a checkout containing `skills-files/` instead of the skills embedded in the binary. Also selects the checkout's `backups/` directory for backups. |
| `--apply` | Legacy flag-only installation form, or a writing flag for restore. Do not combine flag-only installation with a named command. |
| `--with-opencode` | Opt-in flag: install `agentic-sdd-orchestrate` across the four client skill directories and install/update OpenCode v2 agent profiles under `<home>/.config/opencode/agents/`. |
| `--opencode-free-first` | Opt-in preset flag: preset OpenCode writer and test profiles to prefer free models first (requires `--with-opencode`). |

The binary embeds the skills, so ordinary installed-binary use needs no repository checkout. Editing a checkout does not update skills already installed in your home; review a preview and explicitly apply when you want that update.

## Destinations and managed files

Every install targets all four directories listed in [Where the skills go](../README.md#where-the-skills-go). The installer manages only `agentic-sdd-*` directories and copies each complete directory, including `SKILL.md` and supporting references. Unrelated skills are left untouched.

When `--with-opencode` is passed, the installer also installs `agentic-sdd-orchestrate` across all four client skill directories (never in `~/.config/opencode/skills`) and manages the 14 OpenCode v2 agent profiles under `<home>/.config/opencode/agents/` (`sdd-orchestrator.md`, the six task specialists, the two optional OpenAI specialists `sdd-openai-architect.md` and `sdd-openai-implement.md`, and the five MCP-updated review/discovery profiles; derived strictly from `--home`). Staged `.md` agent files use `.agentic-sdd-stage-*` temporary files (never ending in `.md`) with mode `0644`.

Before writing, it rejects symlinks, special files, conflicting non-directory path components, and source skills without a regular `SKILL.md`. It also refuses a source with no matching skills.

## Change detection and permissions

Comparison uses directory structure, entry kinds, file sizes, and SHA-256 content hashes—not permission bits. A permission-only difference is reported as up to date: no rewrite, chmod, or backup. Restore uses the same comparison.

Apply and restore request directory mode `0755` and file mode `0644`, subject to your umask during copying. The top-level staged skill directory is explicitly set to `0755` after staging, so it can be more permissive than its children under a restrictive umask. An unchanged skill keeps its existing modes.

Backup copying preserves ordinary read/write/execute mode bits subject to umask, but restoring uses the install modes above. These are content-recovery backups, not guarantees of original ownership, ACLs, extended metadata, or exact permission restoration. Focused permission tests cover Linux temporary directories with umasks `022` and `077`, not every filesystem policy.

## Backups and restore

Backups are stored under:

- `<home>/.agentic-sdd/backups/` when using embedded skills.
- `<repo>/backups/` when `--repo` selects a checkout.

Each backup has a UTC timestamp ID, saved skill trees where applicable, and a `manifest.json`. The manifest records the operation, tool metadata, home, backup root, and per-skill entries: `replaced` with a saved copy and digest, or `installed` when there was no previous copy to save. Restore records the source backup ID as well. When `--with-opencode` is used, backups record Format 3 manifests with `opencode_targets` and `kind: "opencode-agent"` for individual agent files, using dedicated single-file SHA-256 digests (`fileDigest`). Restore handles Format 3 backups automatically without requiring extra flags.

Backup data is complete before replacement begins. Each new skill tree is staged beside its destination, then installed by rename. If installation fails partway, rollback restores changes already made and leaves the backup for recovery.

A restore first backs up what it will change. It puts back entries recorded as replaced and removes skills recorded as freshly installed, allowing a restore to undo an installation. The new backup can undo the restore itself. Restore derives destinations from the current `--home`, refuses a backup for a different home, and validates saved content before using it.

Digests cover relative paths, entry kinds, and file contents, not permissions. They detect content corruption but are not cryptographic signatures proving who created a backup. Older backups are marked `[legacy]` and restored on a best-effort basis; without fresh-install records, they cannot remove skills that were newly installed by a later apply.

Backups can contain private previous skill content. Repository-local `backups/` is Git-ignored; retain or prune your backups deliberately rather than publishing them with an issue.

## Logs and exit codes

Apply and writing restore operations create private timestamped text logs under `<home>/.agentic-sdd/logs/`, including actions, results, and errors. This location uses the home directory even when `--repo` selects a checkout backup root. Preview writes no log. `agentic-sdd logs --home PATH` reads the latest log for the selected home.

- **0:** successful command, preview, or completed cancellation.
- **2:** usage error, invalid flags/arguments/backup ID syntax, or an unavailable experimental command.
- **1:** filesystem, logging, sync, or backup-validation failure, including a missing backup ID.

Errors go to stderr; action output goes to stdout. A preview listing replacements exits successfully—it has not installed them.

## Contributor tooling

See [Contributing](../CONTRIBUTING.md) for source setup and tests. [Check-runner documentation](../harness/README.md) covers source-only deterministic verification, and [trusted CI](../harness/trusted-ci.md) explains isolated checks and their limits. Neither the installer nor agent-readable skills provide an operating-system security sandbox.
