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
**B1 state:** not started. **Next action:** wait for combined approval of r2, then explicit authorization to implement B1.

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
