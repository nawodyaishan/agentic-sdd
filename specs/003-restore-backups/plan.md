# Plan: Restore skill backups

Spec revision used: `spec.md` r2 (draft/pending, 2026-09-25). Combined approval is recorded only in `spec.md` under `## Approval`.

## Approach

There are three steps, each built on the one before.

1. **Make backups self-describing.** `installChanges` already creates the backup for every change set. It will also write a format 2 manifest that keeps the format 1 fields and adds `format`, `id`, `created_at`, `operation`, `restored_from`, `tool`, `home`, `backup_root` and per-skill `entries` (`client`, `skill`, `path`, `action`, `backup`, `sha256`). The entry `action` follows from the existing `change.old` flag: `replaced` if the skill existed, `installed` if it did not. The records therefore come from data the planner already has, not from a second scan.
2. **Treat a selected backup as one more skill source.** A restore is an install whose source is `os.DirFS(<backup root>/<ID>)`, whose source path per change is `<client>/<skill>`, and whose change set comes from the backup's records. The one new kind of change is **remove**, for format 2 `installed` entries. Remove reuses the existing backup-then-rename-away-then-rollback mechanics without a staged copy. Staging, the pre-change backup, the rename and rollback all stay in one code path.
3. **Expose it in the CLI.** Add `backups`, plus `restore` with two equivalent shapes: `restore [ID] [--apply]`, and `restore list|preview [ID]|apply [ID]`. Add an injectable stdin for the numbered prompt and the apply confirmation.

## Affected components

- `internal/skillsync/sync.go`:
  - `change` gains `src` (the source-relative path; `Sync` sets it to the skill name) and `remove` (no stage; the current tree is backed up, renamed to an undo dir, and deleted after success);
  - the per-destination check-and-compare step is factored out of `planChanges`;
  - the backup-root rule and the four-target list are lifted out of `Sync` into helpers shared with restore;
  - `installChanges` takes an operation descriptor (operation, `restored_from`, source label, the skill list for the format 1 `skills` field, and the preview/summary wording), and it writes the format 2 manifest with one entry per change, including the `sha256` of each saved tree, computed from the backup copy just written;
  - `rollback` already handles "the path is absent, rename the undo back", which covers a remove, because `os.RemoveAll` on a missing path is a no-op. A test will prove this, not an assumption.
- `internal/skillsync/manifest.go` (new, small): the format 2 `manifest` and `entry` types (format 1 fields keep their JSON names), `readManifest`, which detects format 1 (no `format` field) versus format 2, and `treeDigest(fs.FS, root)`. The digest is SHA-256 over the tree's sorted relative paths, entry types and file contents, reusing the walk pattern of `sameSourceTarget`. It lives in its own file because the manifest is now a documented on-disk contract that both apply and restore read and write. The tool fields come from `agentic-sdd/internal/version`.
- `internal/skillsync/restore.go` (new; the restore counterpart of `sync.go`, in the same package per `AGENTS.md`):
  - `ListBackups(repo, home string) ([]Backup, error)`, newest first; unusable entries are returned flagged with a reason, and format 1 entries are flagged `legacy`;
  - `validateBackupID`: the exact stamp syntax, a single path element, and an `os.Lstat` result that is a real directory;
  - `loadBackup`: the manifest, the top-level allowlist (`manifest.json` plus `agents`, `codex`, `claude` and `agy`), `Lstat` on each client and skill dir (because `fs.WalkDir` over `os.DirFS` follows a symlinked *root*), `checkTreeFS`, a regular `SKILL.md`, the one-to-one match between `replaced` entries and saved trees for format 2, and each entry's `path` matching the destination computed for its client and skill;
  - the home match: `targets` (and the format 2 `home`) compared with those computed from the current `--home`;
  - `Restore(repo, home, id string, apply bool, out io.Writer) error`, which plans from the records, prints the preview (with a legacy notice for format 1), and on apply verifies each `sha256` and then calls `installChanges` with the operation set to `restore` and `restored_from: <ID>`.
- `internal/skillsync/*_test.go`: isolated-home tests for every spec acceptance item that `skillsync` owns.
- `cmd/agentic-sdd/main.go`:
  - add `backups` and `restore` to the command switch;
  - `restore` checks for a `list`/`preview`/`apply` sub-word first. A backup ID can never equal one of these words because IDs must match the stamp syntax, so this needs no disambiguation;
  - each command gets its own `flag.FlagSet` (`--repo`, `--home`, plus `--apply` for the bare `restore` form only; `--apply` with `restore preview` or `restore apply` is a usage error, as with the top-level commands);
  - interspersed parsing: parse flags, take at most one positional ID, then parse the rest, so that `restore ID --apply` and `restore --apply ID` both work;
  - `run` gains an `io.Reader` stdin and an interactivity flag. `main` passes `os.Stdin` and a `Stat().Mode()&os.ModeCharDevice` check; tests pass a `strings.Reader` and a fixed bool, and the existing `run(...)` call sites in `main_test.go` are updated mechanically;
  - the interactive flow is: list, then `Select a backup [1-N] (blank to cancel):`, then preview. For the apply forms, the plan is followed by `Restore these skills? [y/N]:` and then the apply;
  - usage text, and `help [preview|apply|backups|restore]`.
- `cmd/agentic-sdd/main_test.go`: parsing, alias equivalence, exit codes and the interactive flows.
- `tests/e2e-in-container.sh`: extend with the spec's apply → `backups` → `restore preview` → `restore apply` → repeat → restore of the pre-restore backup sequence, asserting on the manifest fields with `grep`, since the container has no `jq`.
- `Makefile`: a read-only `backups` target. Deliberately no restore-apply target.
- `README.md`: CLI reference rows for `backups` and every `restore` form, exit codes, the format 2 manifest description, and a "Restoring a backup" paragraph (format 2 restores exactly; format 1 puts back saved trees only). `AGENTS.md`: Go layout lines for `manifest.go` and `restore.go`.

## Design decisions

1. **Extend the manifest additively; don't change the layout.** The format 1 fields keep their names and meaning, so existing backups and any tool reading them keep working. `format: 2` is the only thing restore uses to tell the formats apart. Existing format 1 manifests are never rewritten.
2. **Entries record what the operation changed, not what exists.** `action` comes from the planner's `old` flag. This makes restore an exact inverse: `replaced` → put the saved tree back, `installed` → remove. The pre-restore backup written by a restore follows the same rule, so restoring it undoes the restore. For example, a skill the restore removed was `old`, so it is recorded as `replaced` and comes back.
3. **Remove is a variant of the existing change, not a second installer.** It is backed up, renamed to an undo dir, deleted on success and renamed back on rollback. The rejected alternative, a separate removal pass, would split the all-or-nothing rollback into two transactions.
4. **The digest guards the saved copy, not the current install.** `sha256` is computed from the backup tree right after it is written, and it is verified before a restore uses it. That detects a truncated or edited backup. Comparing current installs still uses the existing structure-and-content comparison.
5. **Destinations come from `--home` and the client/skill name; manifest paths are only checked.** A tampered or foreign manifest cannot redirect a write or a removal. A mismatch is a refusal.
6. **Both command shapes route to one implementation.** `backups` and `restore list` call the same function; `restore <ID>` and `restore preview <ID>` are identical; `restore <ID> --apply` and `restore apply <ID>` are identical. Tests assert that each pair produces the same output.
7. **List numbers are prompt answers, never arguments.** The numbers shift as soon as a new backup exists, so scripts use the stable ID.
8. **Interactivity is detected, never assumed.** The prompt appears only for a character-device stdin. Otherwise a missing ID exits 2, so CI and pipes never hang.
9. **Exit codes follow the 001 contract.** Code 2 is for malformed input. Code 1 is for state and filesystem failures (not found, malformed backup, digest mismatch, home mismatch, unsafe trees, I/O). Code 0 covers previews, successes, cancels and an empty list.

## Risks

- **Rollback-sensitive refactor (medium, testable).** B1 changes the shared install path that protects real homes. Mitigation: B1's first task is the refactor alone, and the existing `Sync` and rollback tests must pass with unchanged expectations before the manifest fields are added.
- **Manifest compatibility (low, testable).** Adding fields could break format 1 readers only if field names changed. Tests assert that the format 1 fields are byte-for-byte the same values as before and that format 1 fixtures still load.
- **Remove semantics (medium, testable).** Removing skills is new and destructive. It is mitigated by four things together: a removal happens only for a recorded `installed` entry, only after that skill is backed up, only within the current home's client dirs, and it is rolled back on failure. Tests cover a skill that was removed and later restored from the pre-restore backup.
- **Symlink or traversal via backup content (medium, testable).** Mitigation: `Lstat` on the ID, client and skill dirs before any `DirFS` walk; strict ID syntax; allowlisted entries; manifest paths checked, never trusted. Tests cover a symlinked backup, client and skill dir, a symlink inside a skill, and a manifest `path` pointing outside the home.
- **Interactive I/O in tests (low).** Handled with an injected reader and an interactivity flag. No pseudo-terminal dependency.
- **Live-home misuse (process risk).** No Makefile target applies a restore. Go tests and the Docker e2e use temporary homes only.

`agentic-sdd-architecture-review` is optional but reasonable before approval, because this feature changes a documented on-disk format and adds a destructive remove path. Your combined approval is still the decision.

## Specialists and tools

- **`golang-pro`** (installed in this client as `anthropic-skills:golang-pro`): all Go work in B1–B3. That covers the install-path generalization, the manifest and digest, `restore.go`, flag parsing and the stdin injection. It is standard-library Go CLI and filesystem work in this repo's existing patterns.
- **No additional specialist** for `README.md`, `Makefile`, `AGENTS.md` and the e2e shell script. Repository guidance and the existing files are enough.
- **Code discovery**: this checkout has no `.codegraph/`, so use targeted `Grep`/`Read` over `internal/skillsync` and `cmd/agentic-sdd`. The call graph is small: `Sync` → `planChanges`/`installChanges` → `rollback`.
- **Repository gates**: `make test`, `make vet` and `git diff --check` for every batch; `make docker-e2e` for B3 when Docker is available (if it is not, say so and run `tests/e2e-in-container.sh` directly with a temporary home).
- No Context7 or web research is needed. Everything here is Go standard library (`flag`, `io/fs`, `os`, `encoding/json`, `crypto/sha256`, `time`) already used in the repo.

## Next document

`tasks.md`.
