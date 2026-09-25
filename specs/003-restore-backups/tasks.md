# Tasks: Restore skill backups

Status: **draft, awaiting combined human approval** of `spec.md` / `plan.md` / `tasks.md` r2. The approval record lives only in `spec.md` under `## Approval`. No batch is authorized yet.

Default specialist and tools come from `plan.md` (`golang-pro` for Go; targeted local search; repository gates). The exceptions are listed per task.

## Batch B1: self-describing backups (format 2 records written by `apply`)

Outcome: every `apply` that changes anything writes a format 2 `manifest.json`, with location, tool version, timestamps and per-skill records, through a generalized install path. It adds no new command. This batch changes a documented on-disk contract and the safety-critical install path, so it is reviewed on its own.

### T1: generalize the install path without changing behavior

Add the `src` and `remove` fields to `change`. Lift the backup-root rule and the four-target list into shared helpers, and factor the per-destination check-and-compare step out of `planChanges`. Make `installChanges` take an operation descriptor (operation, `restored_from`, source label, the format 1 `skills` list, and the wording), and support `remove` changes: back up, rename to an undo dir, delete on success, and rename back on rollback. Depends on nothing.

Done when:
- `Sync` output and the on-disk results are unchanged;
- all existing `internal/skillsync` and `cmd/agentic-sdd` tests pass with unchanged expectations;
- a new unit test proves that a `remove` change is rolled back (the undo dir is renamed back) when a later change fails;
- `make test`, `make vet` and `git diff --check` are clean.

### T2: format 2 manifest and digest (`internal/skillsync/manifest.go`)

Add the manifest/entry types (format 1 fields keep their JSON names and values; add `format`, `id`, `created_at`, `operation`, `restored_from`, `tool`, `home`, `backup_root`, `entries`), `treeDigest`, and `readManifest` (which detects format 1 versus format 2). Have `installChanges` write one entry per change (`replaced` or `installed`, from `old`), with `backup` and `sha256` for replaced entries. Depends on T1.

Done when isolated-home tests show:
- an apply that both replaces and installs writes correct entries, paths, `home`, `backup_root`, and `tool` values from `internal/version`;
- `created_at` parses as RFC 3339 and matches `id`;
- each `sha256` matches a recomputed digest of the saved tree;
- the format 1 fields hold the same values as before this change;
- a no-op apply still creates no backup;
- `readManifest` loads both a format 1 fixture and a format 2 manifest.

**B1 verification:** `make test`, `make vet`, `git diff --check`, all in temporary homes only.

## Batch B1 result — state: awaiting human review

Implemented per the user's "implement" authorization following the r2 redraft; combined approval recorded in `spec.md`.

Changed paths:
- `internal/skillsync/sync.go` — `change` gained `src` (source-relative path; `Sync` sets it to the skill name) and `remove` (no stage; backup, rename to undo, delete on success, rename back on rollback — the rename-phase `os.Rename(c.stage, path)` step is now skipped for `remove` changes, and `rollback`/the success cleanup needed no change since `os.RemoveAll` on an already-absent path is a no-op); added the `operation` type (`kind`, `restoredFrom`); `Sync` now passes `home` and `operation{kind: "apply"}` through; `installChanges` takes `home` and `op operation`, builds one `entry` per change (`replaced` when `old`, with `backup` and a `treeDigest` `sha256`; `installed` otherwise) and writes them into a format 2 manifest alongside the unchanged format 1 fields; the summary line's verb switches to "Restored" for `op.kind == "restore"` (unused until B2/B3, but exercised by the new unit test).
- `internal/skillsync/manifest.go` (new) — the `manifest` struct extended additively with `format`, `id`, `created_at`, `operation`, `restored_from`, `tool`, `home`, `backup_root`, `entries` (all `omitempty`, so a legacy manifest round-trips unchanged); `toolInfo` and `entry` types; `isLegacy()` (`Format < 2`); `readManifest`; `treeDigest` (SHA-256 over an `fs.FS` subtree's relative paths, entry kinds and file contents, walked in `fs.WalkDir`'s stable lexical order).
- `internal/skillsync/manifest_test.go` (new) — `TestApplyWritesFormat2Manifest` (a mixed replace+install apply; checks every format 2 field, the four-target entry set, and that the recorded `sha256` matches a recomputed `treeDigest` of the saved backup tree) and `TestReadManifestDetectsLegacyVersusFormat2` (a hand-built format 1 JSON object with the format 2 keys stripped, and a format 2 manifest, both round-tripped through `readManifest`).
- `internal/skillsync/sync_test.go` — added `TestInstallChangesRollsBackRemoveOnLaterFailure`: a `remove` change is committed (renamed to its undo dir) before a second, unrelated change fails at its own rename step (forced by a pre-existing non-empty directory at that destination); asserts the removed skill's original content is restored via `rollback`, the failed change's original directory is untouched, and no `.agentic-sdd-stage-`/`.agentic-sdd-undo-` directories are left behind.

Checks (all against this code state, in temporary directories only):
- `go build ./...`, `go vet ./...`, `gofmt -l .` (no output) — clean.
- `go test ./...` — all pass, including the three new tests; every pre-existing `internal/skillsync` and `cmd/agentic-sdd` test passes with unchanged expectations, confirming `Sync`'s output and on-disk results are unaffected by the generalization.
- `git diff --check` — clean.
- `make build-darwin` — cross-compiles cleanly (darwin amd64+arm64), confirming the new `internal/version` import and manifest code build for the release target.
- Manual smoke test: built the CLI and ran `apply --home <tmp>` with one pre-existing skill; inspected the written `manifest.json` — format 2 fields, four-target entry set (one `replaced` with `backup`/`sha256`, three `installed`) all correct; `tool` fields read `dev`/`none`/`unknown` as expected for an unldflagged local build.

Deviations from `plan.md`: none of substance. `rollback` needed no source change, as anticipated in `plan.md`'s risk note — proven by the new test rather than assumed.

Next action: human review of the manifest-format extension and the install-path generalization (including the new `remove`/`src` fields, unused by any command yet) before authorizing Batch B2 (the restore core in `internal/skillsync`).

## Batch B2: restore core in `internal/skillsync`

Depends on B1 being human-reviewed. Outcome: `skillsync` can list backups, validate a selected one, and preview or apply its restore. For format 2 that is an exact inverse; for format 1 it puts back the saved trees only. Both cases carry the full backup-before-change, staging and rollback contract. It is exercised through Go tests only, with no CLI surface yet.

### T3: backup discovery and validation (`internal/skillsync/restore.go`)

Implement:
- `ListBackups`, newest first, with the summary counts, `legacy` for format 1, and unusable entries flagged with a reason;
- `validateBackupID`;
- `loadBackup`: the top-level allowlist, `Lstat` on the client and skill dirs, `checkTreeFS`, a regular `SKILL.md`, the one-to-one match between format 2 `replaced` entries and saved trees, an entry `path` equal to the computed destination, and a matching `id`;
- digest verification;
- the home-match check.

Depends on B1.

Done when tests cover:
- an empty or missing root, correct ordering, legacy marking, and the summary text;
- rejections for: a bad ID syntax, `..`/separator IDs, a missing manifest, unparseable JSON, a mismatched `id`, an unknown top-level entry, a non-`agentic-sdd-*` dir, a symlinked backup/client/skill dir, a symlink inside a skill, a missing `SKILL.md`, an entry without a saved tree, a saved tree without an entry, an entry `path` outside the home, a digest mismatch, and a home mismatch.

### T4: `Restore` preview and apply

Plan from the records:
- format 2: `replaced` → `restore` / `replace + back up` / `up to date`; `installed` → `remove + back up` / `up to date` (already absent);
- format 1: the saved trees only, plus a legacy notice.

Preview prints the plan and the `Preview only…` line. Apply verifies the digests, then calls `installChanges` with `operation: restore` and `restored_from: <ID>`. If there are no changes, it prints that everything is up to date and creates no backup. Depends on T3.

Done when isolated-home tests show:
- preview writes nothing;
- a format 2 restore of an apply's backup returns replaced skills to their saved trees and removes the installed ones, leaving unrelated skills, skills without a record, and the selected backup untouched;
- the pre-restore backup's entries record exactly what the restore changed, and restoring it returns the post-apply state (the removed skills come back, and the restored skills revert);
- a format 1 restore puts back the saved trees only and prints the legacy notice;
- a repeated restore is a no-op with no new backup;
- an injected mid-install failure (with both replace and remove changes) rolls every destination back;
- destination refusals happen before any write.

`make test`, `make vet` and `git diff --check` must be clean.

**B2 verification:** `make test`, `make vet`, `git diff --check`, all in temporary homes only.

## Batch B2 result — state: awaiting human review

Implemented per the user's "proceed" authorization following B1's review.

Changed paths:
- `internal/skillsync/sync.go` — lifted `clientTargets(home)` and `backupRootFor(repo, home)` out of `Sync` into shared helpers (`Sync`'s own behavior and output are unchanged; it now calls them instead of repeating the same logic inline).
- `internal/skillsync/restore.go` (new) — `ValidateBackupID` (exported; the stamp-format regexp itself rules out path separators and `..`, so there is no separate traversal check); `Backup`/`restoreRecord` types; `ListBackups` (newest first; a missing root reports zero backups, not an error; a malformed entry is included with `Unusable` set to why); `summarizeRecords`; `loadBackup` (the manifest read, id-match check, targets/home match, the top-level allowlist walk with an `Lstat` on every backup/client/skill directory before any `fs.WalkDir`, the one-to-one cross-check between format 2 `replaced` entries and saved trees including per-entry digest verification, and the legacy fallback that treats every saved tree as a replaceable record); `Restore` (plans one line per record — `restore`/`replace + back up`/`up to date` for a `replaced` record, `remove + back up`/`up to date` for an `installed` one, plus a legacy notice — then, on apply, routes the resulting changes through the same `installChanges` B1 already generalized, with `operation{kind: "restore", restoredFrom: id}` and the restored backup itself as the new backup's `source` label).
- `internal/skillsync/restore_test.go` (new) — `TestListBackupsEmptyOrMissingRoot`, `TestListBackupsOrderingLegacyAndSummary` (ordering, the `legacy` marker, and the summary text), `TestRestorePreviewWritesNothing`, `TestRestoreFormat2RoundTrip` (apply → restore reverts the replaced skill and removes the fresh installs → the pre-restore backup's entries are all `replaced` → a repeated restore is a no-op → restoring the pre-restore backup returns the exact post-apply state), `TestRestoreFormat1PutsBackOnlySavedTrees` (a hand-built legacy manifest; restore puts back only the saved tree and leaves an unrecorded fresh install alone), `TestRestoreInstallRollsBackMixedReplaceAndRemove` (a mixed replace+remove change set built from a real `loadBackup` result, with a deliberately engineered `installChanges` failure — see its doc comment for why this is exercised at that level rather than through `Restore` itself, which structurally cannot reach that state), `TestRestoreRefusesNonDirectoryTarget`, `TestRestoreRefusesSymlinkInInstalledTree`, and `TestLoadBackupRejections` (a table of 16 distinct malformed/hostile backups: bad ID syntax, ID traversal, home mismatch, missing manifest, unparseable JSON, mismatched `id`, an unknown top-level entry, a non-`agentic-sdd-*` entry, a symlinked backup/client/skill directory, a symlink inside a skill, a missing `SKILL.md`, an entry without a saved tree, a saved tree without an entry, an entry `path` mismatch, and a digest mismatch).

Deviation from `plan.md`/`tasks.md`, noted for review: `validateBackupID` is exported as `ValidateBackupID`. B3's CLI needs to run the ID-syntax check on its own, separately from a full `Restore`/`ListBackups` call, so it can give a bad ID its own exit-2 status (per spec Acceptance 8) while every other `loadBackup`/`Restore` failure exits 1. Everything else matches the plan as drafted.

Checks (all against this code state, in temporary directories only):
- `go build ./...`, `go vet ./...`, `gofmt -l .` (no output) — clean.
- `go test ./...` — all pass, including the 12 new B2 tests (2 of them table-driven, 16 sub-cases) and every pre-existing test with unchanged expectations.
- `git diff --check` — clean.
- `make build-darwin` — cross-compiles cleanly.

Next action: human review of the restore core (`ListBackups`, `loadBackup`'s validation, `Restore`'s planning and the `ValidateBackupID` export) before authorizing Batch B3 (the CLI commands, interactive selection, docs, Makefile and Docker e2e).

## Batch B3: CLI commands and selection, docs and end-to-end check

Depends on B2 being human-reviewed. Outcome: `backups`, `restore list`, `restore [preview|apply] [ID]` and `restore [ID] [--apply]` all work, including interactive numbered selection with apply confirmation. Help, README, Makefile, AGENTS.md and the Docker e2e all match.

### T5: `backups` and `restore` commands

Add `backups` and `restore` to `run`'s switch:
- the `restore` sub-word dispatch (`list`/`preview`/`apply`, otherwise a bare ID or none);
- per-command flag sets, with `--apply` allowed only on the bare form;
- interspersed ID parsing;
- an `io.Reader` stdin and an interactivity flag on `run`, wired in `main`, with the existing test call sites updated;
- the numbered prompt (blank cancels; out-of-range or non-numeric input exits 2) and the `y/N` confirmation for the apply forms;
- a missing ID on a non-interactive stdin exits 2 and names `agentic-sdd backups`;
- usage text and `help [preview|apply|backups|restore]`.

Depends on B2.

Done when `main_test.go` covers:
- `backups` equals `restore list`, including the empty root;
- `restore <ID>` equals `restore preview <ID>`;
- `restore <ID> --apply`, `restore --apply <ID>` and `restore apply <ID>` are equivalent;
- `--apply` with `preview` or `apply` exits 2; stray args and bad flags exit 2;
- a not-found ID, a digest mismatch and a home mismatch exit 1;
- interactive: select → preview; select → `y` → applied; select → `n` → unchanged (0); blank → cancel (0); a bad number (2);
- a non-interactive missing ID exits 2.

### T6: docs, Makefile and Docker e2e

Update `README.md`:
- CLI reference rows for `backups` and every `restore` form;
- exit codes;
- the Safety model, with the format 2 manifest table and a "Restoring a backup" paragraph (format 2 is exact; format 1 puts back saved trees only).

Add a read-only `backups` Makefile target and no restore-apply target. Add `manifest.go` and `restore.go` to `AGENTS.md`'s Go layout. Extend `tests/e2e-in-container.sh` with the spec's sequence (acceptance 10), asserting on the manifest fields with `grep`. Depends on T5. Task exception: no additional specialist is needed for the doc, Makefile and shell edits, matching `plan.md`'s default for these files.

Done when the docs match the implemented help text, and `make docker-e2e` passes. If Docker is unavailable, record that and run `tests/e2e-in-container.sh` with a temporary home instead.

**B3 verification:** `make test`, `make vet`, `git diff --check`, `make docker-e2e` (or the stated fallback). No `apply` or restore apply against a real home.

## Batch B3 result — state: awaiting human review

Implemented per the user's "proceed" authorization following B2's review.

Changed paths:
- `cmd/agentic-sdd/main.go` — added `backups` and `restore` to the command switch; `run` gained `stdin io.Reader` and `interactive bool` parameters (wired from `os.Stdin` and a `ModeCharDevice` check in `main`); `runBackups` (the read-only listing, shared by `agentic-sdd backups` and `agentic-sdd restore list`); `runRestore` (sub-word dispatch for `list`/`preview`/`apply`, interspersed flag/positional parsing so an ID and `--apply` can appear in either order, `--apply` rejected when combined with an explicit `preview`/`apply` sub-word, a non-interactive missing ID naming `agentic-sdd backups`, and the interactive numbered-list-then-confirm flow); `selectBackup` and `readLine` (a single shared `bufio.Reader` per invocation, so the selection prompt and the apply confirmation read from the same stream in order); `printBackups` (id, `created_at`/`created`, operation — `restore from <id>` when applicable — tool version, source, and the per-backup summary, with a `[legacy]` marker and an `unusable: <reason>` line); `resolveHome` (a small shared helper, also now used by the original preview/apply path); usage text and `help`'s topic check extended to `backups`/`restore`.
- `cmd/agentic-sdd/main_test.go` — added a `runNonInteractive` helper and updated the 7 pre-existing `run(...)` call sites to use it (no behavior change); added `setupBackupCLI`, `read`, and `normalizeBackupLine` test helpers; added `TestRunBackupsEqualsRestoreList`, `TestRunRestorePreviewEqualsBareID`, `TestRunRestoreApplyFormsEquivalent` (all three apply forms produce the same result and, once the backup path suffix is normalized away, the same output), `TestRunRestoreUsageErrors` (table: `--apply` combined with an explicit sub-word, a stray extra argument, an undefined flag, and a malformed id both bare and under `restore preview`), `TestRunRestoreStateErrorsExitOne` (not found, digest mismatch, home mismatch), `TestRunRestoreNonInteractiveMissingID`, and `TestRunRestoreInteractiveFlow` (select-then-preview, select-then-`y`-applies, select-then-`n`-cancels, blank-cancels, a bad number, and an empty backup list).
- `README.md` — CLI reference rows for `backups` and every `restore` form, updated exit codes, a format 2 manifest field table, and a "Restoring a backup" paragraph (exact inverse for a format 2 backup; saved-trees-only for a `[legacy]` one); repository layout and test-coverage paragraph updated for `manifest.go`/`restore.go`/`restore_test.go`/`manifest_test.go`.
- `AGENTS.md` — Go layout entries for `manifest.go` and `restore.go`.
- `Makefile` — a read-only `backups` target (`go run ./cmd/agentic-sdd backups`); deliberately no restore-apply target, so no one-keystroke path writes to a real home.
- `tests/e2e-in-container.sh` — extended with the spec's full sequence: apply (replace + fresh installs) → `backups` (asserted equal to `restore list`) → `restore preview` (no change) → `restore --apply` (reverts the replace, removes the fresh installs, asserts the new backup's `operation`/`restored_from` via `grep`) → repeated restore (no-op, no third backup) → `restore apply` of the pre-restore backup (returns to the post-apply state) → the existing invalid-command check.

Deviation from `plan.md`, noted for review: `plan.md` sketched a `printRestoreUsage` distinct from the top-level usage text; implemented instead by reusing `printUsage` everywhere `--help`/`-h` is requested (on `backups`, `restore`, and every existing command alike), matching the CLI's existing convention that any command's `--help` prints the same full usage rather than a per-command one. Everything else matches the plan as drafted.

Checks (all against this code state; `make docker-e2e` fell back to running the same script directly, noted below):
- `go build ./...`, `go vet ./...`, `gofmt -l .` (no output) — clean.
- `go test ./...` — all pass, including 12 new `cmd/agentic-sdd` tests (several table/sub-test driven) and every pre-existing test with unchanged expectations.
- `git diff --check` — clean.
- `make build-darwin` — cross-compiles cleanly.
- `make docker-e2e` — the Docker **daemon** is not running in this environment (client present, `dial unix /var/run/docker.sock: ... no such file or directory`), so per this task's stated fallback, `sh tests/e2e-in-container.sh` was run directly against a real built binary in a temporary repo/home instead; it printed `Docker end to end checks passed.`. The containerized run itself (`golang:1.25-alpine`, read-only mount, no network) has not been exercised in this session and should be re-checked wherever Docker is available before relying on it.
- Manual smoke test: built the CLI and ran `apply` → `backups` → `restore preview` → `restore --apply` → `backups` again, inspecting real output (shown in the batch's implementation transcript) — the second listing correctly shows the pre-restore backup as `restore from <id>` with `40 replaced, 0 installed`.

A pre-existing, unrelated issue was noticed and left untouched (not part of this feature): `make help`'s `awk` pattern (`[a-zA-Z_-]+`) excludes digits, so `docker-e2e` has always been silently missing from `make help`'s printed list even though the target itself works. Flagged for a separate direct fix; not corrected here.

Next action: human review of the CLI surface, docs, and the direct (non-Docker) e2e run before this feature is considered done. Re-run `make docker-e2e` in an environment with a running Docker daemon when convenient. No real `apply` or restore apply has been run against a real user home.

## Continuation

- Approval reference: `spec.md` `## Approval` (currently draft/pending, r2).
- Current batch: none. First ready batch after approval: **B1**.
- Record changed paths, the checks and the code state they ran against, and set `awaiting human review` at each batch boundary. Passing checks never start the next batch.
