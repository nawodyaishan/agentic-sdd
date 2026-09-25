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
**B2 state:** not started. **Next action:** after B1's human review, wait for explicit authorization to implement B2.

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
**B3 state:** not started. **Next action:** after B2's human review, wait for explicit authorization to implement B3.

## Continuation

- Approval reference: `spec.md` `## Approval` (currently draft/pending, r2).
- Current batch: none. First ready batch after approval: **B1**.
- Record changed paths, the checks and the code state they ran against, and set `awaiting human review` at each batch boundary. Passing checks never start the next batch.
