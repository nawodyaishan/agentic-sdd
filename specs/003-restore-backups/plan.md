# Plan: Restore skill backups

Spec revision used: `spec.md` r1 (draft/pending, 2026-09-25). Combined approval is recorded only in `spec.md` under `## Approval`.

## Approach

Treat a selected backup as one more `fs.FS` skill source, and reuse the install pipeline that already does staging, pre-replace backup, rename and rollback. A restore is an apply whose source is `os.DirFS(<backup root>/<ID>)`, whose skill paths are `<client>/<skill>` instead of `<skill>`, and whose change set comes from the backup's contents instead of from all four clients × all source skills. The code needs only two new things:

1. **Backup discovery and validation**: listing the root, ID and layout checks, manifest parsing, and the home match.
2. **CLI selection**: the `backups` and `restore` commands, and an injectable stdin for the numbered prompt and confirmation.

The rest is a small generalization of the existing planning and install functions, so restore gets the same safety behavior without a second copy of the staging and rollback code.

## Affected components

- `internal/skillsync/sync.go`:
  - add a `src` field (the source-relative path) to `change`, so `installChanges` copies from `c.src` instead of assuming `c.skill`. `Sync` sets `src = skill`, so its behavior is unchanged;
  - factor the per-destination check-and-compare step out of `planChanges`, so restore planning can call it for each `<client>/<skill>` in a backup;
  - make `installChanges`'s backup manifest take the skill list it should record (`Sync` passes all source names, as now; restore passes the restored names);
  - `installChanges` takes the preview/apply verbs and the final summary line as data, or its caller prints them, so restore can say `restore` and `Restored N skills`. Choose the smallest change at implementation time.
- `internal/skillsync/restore.go` (new; the restore counterpart of `sync.go`, kept in the same package per `AGENTS.md`):
  - `backupRoot(repo, home)`, lifted from `Sync`'s inline logic so `Sync`, `ListBackups` and `Restore` share one rule;
  - `ListBackups(repo, home string, out io.Writer) ([]Backup, error)`, newest first; malformed entries are returned flagged with a reason, not dropped;
  - `validateBackupID`, which requires the stamp format regexp derived from the existing `"20060102T150405.000000000Z"` layout, a single path element, and an `os.Lstat` result that is a real directory;
  - `loadBackup`, which parses the manifest, allowlists top-level entries (`manifest.json`, `agents`, `codex`, `claude`, `agy`) and `agentic-sdd-*` skill dirs, calls `os.Lstat` on each client and skill directory (because `fs.WalkDir` on `os.DirFS` follows a symlinked *root*), then runs `checkTreeFS` and the regular `SKILL.md` check on each tree;
  - `Restore(repo, home, id string, apply bool, out io.Writer) error`, which validates, runs the home-match check (manifest `targets` compared with the targets computed from `home`, using the same `targets` slice `Sync` builds, lifted into a helper), plans, previews, and applies through `installChanges` with the backup FS as source and `restore:<ID>` as the manifest source label.
- `internal/skillsync/sync_test.go` (or a new `restore_test.go`): isolated-home tests for every spec acceptance item that `skillsync` owns.
- `cmd/agentic-sdd/main.go`:
  - `backups` and `restore` cases in the command switch, each with its own `flag.FlagSet` (`--repo`, `--home`, plus `--apply` for `restore`);
  - interspersed parsing for `restore`, so that `restore ID --apply` and `restore --apply ID` both work. Go's `flag` stops at the first positional, so parse, take at most one positional, then parse the rest; more than one positional is a usage error;
  - `run` gains an `io.Reader` stdin plus an "is interactive" check. `main` passes `os.Stdin` and a `Stat().Mode()&os.ModeCharDevice` check; tests pass a `strings.Reader` and a fixed bool. Updating the existing `run(...)` call sites in `main_test.go` is mechanical;
  - the interactive flow: list, then `Select a backup [1-N] (blank to cancel):`, then preview; with `--apply`, the plan is printed, then `Restore these skills? [y/N]:`, then apply. The first version runs planning twice (once to show the plan, once inside apply). That is acceptable at this scale and avoids a new "plan then apply" API; revisit only if a test shows the double planning matters.
  - usage text, and `help [preview|apply|backups|restore]`.
- `cmd/agentic-sdd/main_test.go`: parsing, exit codes and interactive-flow tests.
- `tests/e2e-in-container.sh`: extend with apply → `backups` → `restore` preview → `restore --apply` → repeat restore → restore of the pre-restore backup.
- `Makefile`: a read-only `backups` target (`go run ./cmd/agentic-sdd backups`). Do not add a `restore --apply` make target, so that no one-keystroke path writes to the real home.
- `README.md`: CLI reference rows for `backups`/`restore`, the `--apply` meaning for restore, exit codes, and a "Restoring a backup" paragraph in the Safety model. `AGENTS.md` Go layout gets a line for `restore.go`.

## Design decisions

1. **Reuse `installChanges`; do not write a parallel restore installer.** The staging, pre-replace backup, rename and rollback path is the part most worth not duplicating, and the tests already cover it. The cost is a `src` field and a caller-supplied skill list and labels. The rejected alternative, a standalone `restore` function, would duplicate about 90 lines of rollback-sensitive code.
2. **Destinations come from `--home` and the client name; the manifest is only checked.** The manifest's absolute `targets` are used only for the home-match refusal, never as write paths. A tampered or foreign manifest therefore cannot redirect writes.
3. **The home match is a hard refusal, with no override flag.** Cross-home restore is out of scope. An override can be added later if a real need appears.
4. **The pre-restore backup uses the existing format.** Only `source` differs (`restore:<ID>`), so `backups` lists it and `restore` can read it. The spec forbids format changes.
5. **Well-formed backups are listed and selected by exact ID; the list number is only a prompt answer.** The numbers shift as soon as a restore creates a new backup, so they are not accepted as a CLI argument. This keeps scripted use stable.
6. **Interactivity is detected, never assumed.** The prompt appears only for a character-device stdin. Otherwise a missing ID is exit 2, so CI and pipes never hang.
7. **Exit codes follow the 001 contract.** Code 2 is for malformed input (bad ID syntax, stray args, bad flags, invalid prompt answer, missing ID when non-interactive). Code 1 is for state and filesystem failures (ID not found, malformed backup, home mismatch, unsafe trees, I/O). Code 0 covers preview, a successful restore, cancel and an empty list.

## Risks

- **Rollback-sensitive refactor (medium, testable).** Generalizing `change`/`installChanges` touches the path that protects real homes. Mitigation: B1 changes the shared code first, with the existing `Sync` tests (`TestSyncBacksUpAndReplaces`, the refusal tests and `TestRollbackRestoresOriginalAndRemovesNewSkill`) passing unchanged before any restore code is added.
- **Symlink or traversal via backup content (medium, testable).** `os.DirFS` follows symlinks when opening a root path. Mitigation: `os.Lstat` on the ID, client and skill dirs before any `DirFS` walk; strict ID syntax; allowlisted entries. Add tests for a symlinked backup dir, a symlinked client dir, a symlinked skill dir and a symlink inside a skill.
- **Semantics surprise: fresh installs survive a restore (accepted, documented; spec Q1).** The README states that restore puts back backed-up versions and is not a full undo.
- **Interactive I/O in tests (low).** Handled by injecting the reader and the interactive flag. No pseudo-terminal dependency.
- **Live-home misuse (process risk).** No Makefile target runs `restore --apply`. The docker e2e and Go tests use temporary homes only. Approval does not authorize a real-home restore.

`agentic-sdd-architecture-review` is optional here. The design reuses a proven path, and its risks are covered by tests. Run it before approval only if you want an independent check of the refactor in design decision 1.

## Specialists and tools

- **`golang-pro`** (installed in this client as `anthropic-skills:golang-pro`): all Go work in B1 and B2, meaning the `change`/`installChanges` generalization, `restore.go`, flag parsing and the stdin injection. It is ordinary standard-library Go CLI and filesystem work in this repo's existing patterns.
- **No additional specialist** for `README.md`, `Makefile`, `AGENTS.md` and the e2e shell script. Repository guidance and the existing files are enough.
- **Code discovery**: this checkout has no `.codegraph/`, so use targeted `Grep`/`Read` over `internal/skillsync` and `cmd/agentic-sdd`. The call graph is small: `Sync` → `planChanges`/`installChanges` → `rollback`.
- **Repository gates**: `make test`, `make vet` and `git diff --check` for every batch; `make docker-e2e` for B2 when Docker is available (if it is not, say so and run `tests/e2e-in-container.sh` directly with a temporary home).
- No Context7 or web research is needed. Everything here is Go standard library (`flag`, `io/fs`, `os`, `regexp`/`time.Parse`) already used in the repo.

## Next document

`tasks.md`.
