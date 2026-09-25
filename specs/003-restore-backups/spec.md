# Restore skill backups

Revision: r1 (draft, 2026-09-25) — first draft of spec, plan and tasks together.

## Outcome and sources

Let a user who ran `agentic-sdd apply` put their previous skills back. They list the backups the installer has made, pick one from the CLI (by ID, or from a numbered list when running interactively), preview what the restore would change, and restore it with an explicit `--apply`. The restore follows the same safety contract as install: it previews by default, backs up what it replaces, stages each copy before touching an installed skill, rolls back on failure, and leaves unrelated skills alone.

Sources:
- `AGENTS.md`: Safe changes (preview default, `--apply` is the only installing mode, back up before replacing, reject symlinks/special files, no-op behavior, keep README and Makefile in step with the CLI), Verification, and the preview-versus-apply dev/live boundary.
- `README.md`: Safety model (backup location and layout, `manifest.json`), CLI reference (commands, flags, exit codes), and Where the skills go (the four client targets).
- `internal/skillsync/sync.go`: `Sync`, `installChanges` (backup root selection, timestamp format `20060102T150405.000000000Z`, `manifest{Created, Source, Targets, Skills}`, backup layout `<stamp>/<client>/<skill>/`), `rollback`.
- `cmd/agentic-sdd/main.go`: the command switch, flag handling and exit-code conventions.
- `specs/001-cli-quality/spec.md` (the CLI command and exit-code contract) and `specs/002-homebrew-publishing/spec.md` (the embedded source, and backups under `<home>/.agentic-sdd/backups` when `--repo` is unset).

This repository has no top-level product SRS or roadmap, so this spec is the requirement source for this feature.

## What a backup contains today (the existing contract)

An apply that replaces at least one existing skill creates `<backup root>/<stamp>/`. The backup root is `<repo>/backups` with `--repo`, and `<home>/.agentic-sdd/backups` otherwise. The backup holds:

- `manifest.json`, which records `created` (the stamp), `source` (`embedded` or the `skills-files` path), `targets` (the absolute paths of the four client skill directories at the time) and `skills` (every source skill name in that run, not only those backed up).
- `<client>/<skill>/`, a copy of each skill as it was before that apply replaced it. `<client>` is one of `agents`, `codex`, `claude` or `agy`. Skills that apply installed fresh have no entry, because nothing existed to back up.

This feature reads that format and does not change it.

## Scope

- **List backups.** A read-only `agentic-sdd backups` command shows the backups in the backup root (the same root `apply` would use for the given `--repo`/`--home`), newest first. Each row shows a selection number, the backup ID (the stamp directory name), the recorded source, and what the backup holds (the number of skill directories and the clients they came from). A directory that is not a well-formed backup is listed as unusable with a reason and cannot be selected. An empty or missing root prints a clear "no backups" message and exits 0.
- **Select a backup.** `agentic-sdd restore <BACKUP_ID>` selects a backup by its exact ID. `agentic-sdd restore` with no ID, when stdin is a terminal, shows the same list and asks for a number. A blank answer cancels. When stdin is not a terminal, a missing ID is a usage error that points to `agentic-sdd backups`.
- **Preview a restore (default).** Restore lists one line per backed-up skill: `restore <client>/<skill>` (the skill is missing now, so it is installed from the backup), `replace + back up <client>/<skill>` (the current skill differs from the backup), or `up to date <client>/<skill>` (identical). It then stops without writing, as `preview` does.
- **Apply a restore.** `agentic-sdd restore <BACKUP_ID> --apply` does the same planning, then:
  - backs up every current skill it will replace into a new timestamped backup in the same root, in the same format, with `source` recording the restored backup (for example `restore:<BACKUP_ID>`), so a restore can itself be undone by restoring that new backup;
  - stages each restored copy beside its destination before touching an installed skill, installs by rename, and rolls back fully on failure, as `apply` does.

  If nothing differs, the restore prints that everything is already up to date and creates no backup. In interactive selection with `--apply`, the plan is shown and the user must confirm with `y` before anything is written. Any other answer cancels with no changes.
- **Restore is scoped to the backup's contents.** Only the `<client>/<skill>` directories in the selected backup are touched. Skills that apply installed fresh (so they have no backup entry), non-`agentic-sdd-*` skills, and clients the backup does not mention stay as they are. The backup being restored is never modified or deleted.
- Update the CLI help, the `README.md` CLI reference and safety model, the `Makefile` (a read-only `backups` target), `AGENTS.md` Go layout (if a new file is added), and the Docker end-to-end script so they cover restore.

## Safety and validation

- Preview is the default for `restore`. Only `--apply` writes. `backups` never writes.
- A backup ID must match the stamp format exactly, and it must name a direct child directory of the backup root. Path separators, `..`, absolute paths, and symlinked or non-directory entries are refused before anything is read from them.
- A backup is usable only when its `manifest.json` parses, its top-level entries are `manifest.json` plus known client directories, every entry under a client directory is an `agentic-sdd-*` directory, and every tree passes the existing symlink/special-file check (with a regular `SKILL.md` in each skill).
- **Home match.** The manifest's recorded `targets` must match the client directories computed from the current `--home`. If they differ (for example, the backup was made with `--home /tmp/test-home` and is being restored into the real home), the restore is refused with both paths named. Destinations always come from the current `--home` and the client name, never from paths in the manifest, so a restore cannot write outside the current home's four client directories.
- Current destinations get the same checks as `apply`: parent components, non-directory targets, and symlinks or special files in an installed skill tree are all refused before any change.
- Restored files get the installer's normal modes (`0755` directories, `0644` files). Comparison uses structure and content, as the embedded-source comparison does today.
- Running `restore --apply` against a real user home is a live operation. Approving this feature authorizes developing and testing restore in temporary homes only (see `AGENTS.md`).

## Acceptance

1. `agentic-sdd backups` (with `--home` and optionally `--repo`) lists every well-formed backup newest first, with selection number, ID, source and contents. It marks malformed entries as unusable with a reason, prints a clear message for an empty or missing root, and writes nothing.
2. `agentic-sdd restore <ID>` prints the per-skill plan (`restore` / `replace + back up` / `up to date`) and then `Preview only. Run with --apply to restore.`. It changes no skill directory and creates no backup.
3. `agentic-sdd restore <ID> --apply` restores exactly the backed-up `<client>/<skill>` trees into the current home's client directories. It first creates a new backup of every current skill it replaces, and that new backup's `manifest.json` names the restored ID. Skills not in the backup, unrelated skills and the selected backup are unchanged. Restoring that new backup afterwards returns the skills to their pre-restore state.
4. Repeating the same `restore <ID> --apply` is a no-op: it prints that everything is up to date and creates no new backup.
5. A failure partway through an applied restore leaves every destination as it was before the restore, like `apply`'s rollback. The pre-restore backup remains for manual recovery.
6. Refusals exit before any write: an ID that does not match the stamp format or that traverses paths exits 2. A well-formed ID that does not exist, a malformed backup, a home mismatch, or symlinks/special files in the backup or destination exit 1. Each prints an actionable message to stderr.
7. With no ID, `restore` shows the numbered list and prompt on an interactive stdin. It accepts a valid number, treats blank input as a cancel (exit 0, no changes), and rejects an out-of-range or non-numeric answer (exit 2). With `--apply`, nothing is written unless the answer to the confirmation is `y`. On a non-interactive stdin, a missing ID exits 2 and names `agentic-sdd backups`. Tests drive this through injected input, not a real terminal.
8. `agentic-sdd help`, `--help`, `README.md` (CLI reference and safety model), and `Makefile` describe `backups` and `restore`. The Docker end-to-end script exercises apply → backups → restore preview → restore apply → repeat restore in an isolated home. `make test`, `make vet`, `git diff --check` and `make docker-e2e` (when Docker is available) pass.

## Outside scope

- Removing skills that a later apply installed fresh. Restore puts back the backed-up versions; it is not a full undo of an apply (see Q1).
- Restoring part of a backup (per client or per skill filters), restoring into a different home than the backup's recorded targets, and restoring across backup roots in one command.
- Pruning, deleting, renaming, exporting or compressing backups; retention policies.
- Changing the backup directory layout or `manifest.json` fields. The new pre-restore backup uses the existing format, with only the `source` value distinguishing it.
- A third-party TUI/prompt library, fuzzy selection, and a `latest` alias.
- Concurrency control between simultaneous `apply`/`restore` runs.
- Running `restore --apply` against any real user home. That needs separate, explicit authorization.

## Open questions (resolve at combined approval)

The draft proceeds on these defaults. Changing one changes behavior, so each is up to you:

- **Q1 — fresh installs after the backup.** Default: leave them in place, because the backup does not record which skills that apply installed fresh. The alternative is to remove `agentic-sdd-*` skills that are absent from the backup but listed in the manifest's `skills`. That is a destructive heuristic and could remove skills a later apply legitimately installed.
- **Q2 — selection UX.** Default: exact ID argument plus a numbered stdin prompt on a TTY, in the standard library only. The alternative is ID-only, with no interactive prompt, which removes the TTY and confirmation handling from B2.
- **Q3 — command shape.** Default: a separate `backups` list command, plus `restore [ID] [--apply]`, keeping `--apply` as the one writing switch per `AGENTS.md`. The alternative is `restore list` / `restore preview ID` / `restore apply ID`, mirroring the top-level `preview`/`apply` commands.

## Approval

Decision: **draft/pending**. No human approval has been recorded yet. Scope under review: `spec.md`, `plan.md` and `tasks.md` at revision r1 (2026-09-25), covering Batch B1 (restore core in `internal/skillsync`) and Batch B2 (CLI selection, docs, Makefile, Docker e2e). Approval would not authorize running `restore --apply` or `apply` against a real user home.
