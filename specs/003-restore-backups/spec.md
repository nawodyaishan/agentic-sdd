# Restore skill backups

Revision: r2 (draft, 2026-09-25). This revision applies the user's decisions on r1's open questions:
- Q1: backups now record location, tool version and timestamps, and each skill's action.
- Q2: the list shows each backup's ID; the user picks one from the list or passes the ID.
- Q3: both command shapes are provided.

r1 was never approved, so r2 replaces it entirely.

## Outcome and sources

Let a user who ran `agentic-sdd apply` put their previous skills back easily and exactly. Every backup records what it holds, where each skill came from, which version of `agentic-sdd` wrote it and when. The user lists the backups, sees each one's ID, picks one from the list (or passes its ID directly), previews the restore and applies it. The restore follows the same safety contract as install: it previews by default, backs up what it changes, stages each copy before touching an installed skill, rolls back on failure and leaves unrelated skills alone.

Sources:
- `AGENTS.md`: Safe changes (preview default, back up before replacing, reject symlinks/special files, no-op behavior, keep README and Makefile in step with the CLI), Verification, and the preview-versus-apply dev/live boundary.
- `README.md`: Safety model (backup location and layout, `manifest.json`), CLI reference (commands, flags, exit codes), and Where the skills go (the four client targets).
- `internal/skillsync/sync.go`: `Sync`, `installChanges` (backup root selection, stamp format `20060102T150405.000000000Z`, current `manifest{Created, Source, Targets, Skills}`, layout `<stamp>/<client>/<skill>/`), `rollback`.
- `internal/version/version.go`: `Version`, `Commit`, `Date` and `GoVersion`, injected at release build time.
- `cmd/agentic-sdd/main.go`: the command switch, flag handling and exit-code conventions.
- `specs/001-cli-quality/spec.md` (the CLI command and exit-code contract) and `specs/002-homebrew-publishing/spec.md` (the embedded source, and backups under `<home>/.agentic-sdd/backups` when `--repo` is unset).

This repository has no top-level product SRS or roadmap, so this spec is the requirement source for this feature.

## Backup records

### What exists today (format 1)

An apply that changes anything creates `<backup root>/<stamp>/`. The backup root is `<repo>/backups` with `--repo`, and `<home>/.agentic-sdd/backups` otherwise. The backup holds:

- `manifest.json`, with `created` (the stamp), `source`, `targets` (the four client directories as absolute paths) and `skills` (every source skill name in the run);
- a `<client>/<skill>/` copy of each skill that apply replaced.

Format 1 does not record which skills that apply installed fresh, which version of the tool wrote it, or a readable time.

### What this feature adds (format 2)

Every new backup, whether written by `apply` or by `restore --apply`, keeps the same directory layout. It writes a `manifest.json` that keeps every format 1 field unchanged and adds these:

| Field | Meaning |
| :--- | :--- |
| `format` | `2` |
| `id` | The backup ID, the same as the directory name and `created` stamp |
| `created_at` | The creation time in RFC 3339 UTC, with nanoseconds |
| `operation` | `apply` or `restore` |
| `restored_from` | For `operation: restore`, the ID of the backup that was restored |
| `tool` | `version`, `commit`, `date` and `go_version` of the `agentic-sdd` binary that wrote it (`dev`/`none`/`unknown` for local builds) |
| `home` | The absolute home directory the operation ran against |
| `backup_root` | The absolute backup root the backup was written to |
| `entries` | One record per skill directory the operation changed (see below) |

Each entry records these fields:
- `client`: `agents`, `codex`, `claude` or `agy`.
- `skill`: the `agentic-sdd-*` name.
- `path`: the absolute destination that was changed.
- `action`: either `replaced` (the skill existed, and its previous tree is in the backup) or `installed` (the skill did not exist before this operation, so nothing is backed up).
- For `replaced` entries: `backup`, the relative path of the saved copy (`<client>/<skill>`), and `sha256`, a digest over the saved tree's relative paths and file contents.

Format 1 backups stay listable and restorable, with the limits described under Restore.

## Scope

- **Record** format 2 manifests on every `apply` and `restore --apply` that changes anything. Backup layout, backup root rules and the no-op rule (nothing changed means no backup) stay as they are.
- **List backups.** Two equivalent read-only commands, `agentic-sdd backups` and `agentic-sdd restore list`, list the backups in the backup root for the given `--repo`/`--home`, newest first. Each row shows:
  - a selection number and the backup ID;
  - `created_at` (or the stamp, for format 1);
  - the operation (`apply` or `restore from <ID>`);
  - the tool version;
  - the source;
  - a summary such as `3 replaced, 2 installed across claude, codex`.

  Format 1 backups are marked `legacy`. A directory that is not a usable backup is listed as unusable, with a reason, and cannot be selected. An empty or missing root prints a clear "no backups" message and exits 0.
- **Select a backup** by passing its exact ID, or by choosing its number from the list:
  - `agentic-sdd restore <ID>` and `agentic-sdd restore preview <ID>` preview the restore of that backup.
  - `agentic-sdd restore <ID> --apply` and `agentic-sdd restore apply <ID>` apply it.
  - Without an ID (`restore`, `restore preview`, `restore --apply`, `restore apply`), when stdin is a terminal, the CLI shows the list and asks for a number. A blank answer cancels. When stdin is not a terminal, a missing ID is a usage error that points to `agentic-sdd backups`.
  - As with the top-level commands, `--apply` cannot be combined with `restore preview` or `restore apply`.
- **Restore returns the recorded skills to their pre-operation state.** For a format 2 backup:
  - a `replaced` entry puts the saved tree back;
  - an `installed` entry removes the skill that operation installed fresh, if it is still present.

  For a format 1 backup, only the saved `<client>/<skill>` trees are put back. Fresh installs are not recorded, so they are left in place. Preview says this for legacy backups. Unrelated skills, skills the backup has no record of, and the selected backup itself are never modified.
- **Preview a restore (default).** Restore prints one line per affected skill, then stops without writing:
  - `restore <client>/<skill>`: the skill is missing now and will be installed from the backup;
  - `replace + back up <client>/<skill>`: the current skill differs from the saved tree;
  - `remove + back up <client>/<skill>`: the skill was installed fresh by the recorded operation and will be removed;
  - `up to date <client>/<skill>`: already identical, or already absent for a removal.
- **Apply a restore.**
  1. Verify each `replaced` entry's `sha256` against the saved tree.
  2. Back up every current skill that will be replaced or removed into a new format 2 backup (`operation: restore`, `restored_from: <ID>`), whose entries record exactly what this restore changed. Restoring that new backup undoes the restore.
  3. Stage each restored copy beside its destination, then install by rename, removing by rename-away as well, and roll back fully on failure, as `apply` does.

  If nothing differs, the restore prints that everything is already up to date and creates no backup. With interactive selection and apply, the plan is shown and the user must confirm with `y` before anything is written. Any other answer cancels with no changes.
- Update the CLI help, the `README.md` CLI reference and safety model (including the format 2 manifest), the `Makefile` (a read-only `backups` target), the `AGENTS.md` Go layout, and the Docker end-to-end script.

## Safety and validation

- `backups` and `restore list` never write. `restore` previews unless `--apply` or `restore apply` is used.
- A backup ID must match the stamp format exactly, and it must name a direct child directory of the backup root. Path separators, `..`, absolute paths, and symlinked or non-directory entries are refused before anything is read from them.
- A backup is usable only when all of the following hold:
  - its `manifest.json` parses, and a format 2 manifest's `id` equals its directory name;
  - its top-level entries are `manifest.json` plus known client directories;
  - every saved tree is an `agentic-sdd-*` directory with a regular `SKILL.md` that passes the existing symlink/special-file check;
  - for format 2, the saved trees and the `replaced` entries match one to one, each entry's `client`/`skill` is valid, and its `path` equals the destination computed for that client and skill.
- A `sha256` mismatch is reported in the list (as unusable) and refuses the restore.
- **Home match.** The manifest's recorded `targets` (and, for format 2, `home`) must match those computed from the current `--home`. Otherwise the restore is refused with both paths named. Destinations always come from the current `--home` plus the client and skill name, never from paths in the manifest, so a restore cannot write or remove outside the current home's four client directories.
- Current destinations get the same checks as `apply`: parent components, non-directory targets, and symlinks or special files in an installed skill tree (including a skill about to be removed) are all refused before any change.
- Restored files get the installer's normal modes (`0755` directories, `0644` files). Tree comparison uses structure and content, as the embedded-source comparison does today.
- Running `restore --apply`/`restore apply` or `apply` against a real user home is a live operation. Approving this feature authorizes developing and testing in temporary homes only (see `AGENTS.md`).

## Acceptance

1. `apply` writes a format 2 `manifest.json` that keeps the format 1 fields plus `format`, `id`, `created_at`, `operation: apply`, `tool` (the values from `internal/version`), `home`, `backup_root` and one entry per changed skill. Each entry has the correct `action`, `path`, and (for `replaced`) `backup` and `sha256`. A no-op apply still creates no backup. Existing `apply` output and the install results are otherwise unchanged.
2. `agentic-sdd backups` and `agentic-sdd restore list` print the same listing, newest first. Each row has a selection number, ID, time, operation, tool version, source and change summary. Format 1 backups are marked `legacy`; unusable entries (including a digest mismatch) show a reason. An empty or missing root prints a clear message. Neither command writes anything.
3. `restore <ID>` and `restore preview <ID>` print the per-skill plan (`restore` / `replace + back up` / `remove + back up` / `up to date`, plus a legacy notice for format 1), then `Preview only. Run with --apply (or restore apply) to restore.`. They change no skill directory and create no backup.
4. `restore <ID> --apply` and `restore apply <ID>` behave identically on a format 2 backup of an apply. `replaced` skills get their saved trees back, and `installed` skills are removed. The first step is a new format 2 backup (`operation: restore`, `restored_from: <ID>`) holding every skill replaced or removed, with entries recording exactly what the restore changed. Unrelated skills and the selected backup are unchanged. Restoring the new backup then returns every affected skill to its pre-restore state, including reinstalling the removed ones and removing any that the restore installed.
5. For a format 1 backup, a restore puts back only the saved trees, leaves other skills in place, and says so in both preview and apply output.
6. Repeating the same restore apply is a no-op: it prints that everything is up to date and creates no new backup.
7. A failure partway through an applied restore leaves every destination as it was before the restore, like `apply`'s rollback. The pre-restore backup remains for manual recovery.
8. Refusals exit before any write:
   - exit 2: an ID that does not match the stamp format or that traverses paths; `--apply` combined with `restore preview`/`restore apply`; a stray argument;
   - exit 1: a well-formed ID that does not exist; a malformed backup; a digest mismatch; a home mismatch; symlinks or special files in the backup or destination.

   Each prints an actionable message to stderr.
9. Without an ID on an interactive stdin, `restore`/`restore preview`/`restore --apply`/`restore apply` show the numbered list and prompt. They accept a valid number, treat blank input as a cancel (exit 0, no changes) and reject an out-of-range or non-numeric answer (exit 2). The apply forms write nothing unless the confirmation answer is `y`. On a non-interactive stdin, a missing ID exits 2 and names `agentic-sdd backups`. Tests drive this through injected input.
10. `agentic-sdd help`, `--help`, `README.md` (CLI reference, safety model and format 2 manifest) and the `Makefile` describe `backups` and every `restore` form. The Docker end-to-end script runs this sequence in an isolated home:
    1. an `apply` that both replaces and installs;
    2. `backups`;
    3. `restore preview`;
    4. `restore apply`, which restores the replaced skills and removes the installed ones;
    5. a repeated restore (no-op);
    6. a restore of the pre-restore backup, which returns the post-apply state.

    `make test`, `make vet`, `git diff --check` and `make docker-e2e` (when Docker is available) pass.

## Outside scope

- Restoring part of a backup (per client or per skill filters), restoring into a different home than the backup's recorded one, and restoring across backup roots in one command.
- Pruning, deleting, renaming, exporting or compressing backups; retention policies.
- Rewriting or upgrading existing format 1 manifests. They are read as they are, never modified.
- Changing the backup directory layout or backup root rules.
- Accepting a list number as a command-line argument. List numbers change as soon as a new backup exists, so the number is only accepted as an answer to the interactive prompt; scripts pass the ID.
- A third-party TUI/prompt library, fuzzy selection, and a `latest` alias.
- Concurrency control between simultaneous `apply`/`restore` runs.
- Running `apply` or a restore apply against any real user home. That needs separate, explicit authorization.

## Resolved decisions (from r1)

- **Q1 → records.** Backups record location (`home`, `backup_root`, per-entry `path`), tool version (`tool`), timestamps (`id`, `created_at`) and each skill's `action`. With those records, restore can return fresh installs to their pre-operation state as well. Only format 1 backups keep the limited behavior.
- **Q2 → list or ID.** The CLI lists each backup's ID. The user picks a number from the list interactively or passes the ID to `restore`.
- **Q3 → both shapes.** `backups` plus `restore [ID] [--apply]`, and `restore list` / `restore preview [ID]` / `restore apply [ID]`, which are equivalent aliases.

## Approval

Decision: **approved for implementation**. Evidence: the user's "implement" message in this conversation, following the r2 redraft that incorporated their decisions on backup records, selection and command shape. Scope: this spec and its linked `plan.md`/`tasks.md` at revision r2 (2026-09-25), covering:
- Batch B1: format 2 backup records written by `apply`.
- Batch B2: the restore core in `internal/skillsync`.
- Batch B3: the CLI commands, selection, docs, Makefile and Docker e2e.

Batches execute one at a time, each stopping for human review before the next begins. No real `apply` or restore apply against a real user home is authorized.
