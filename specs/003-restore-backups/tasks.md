# Tasks: Restore skill backups

Status: **draft, awaiting combined human approval** of `spec.md` / `plan.md` / `tasks.md` r1. The approval record lives only in `spec.md` under `## Approval`. No batch is authorized yet.

Default specialist and tools come from `plan.md` (`golang-pro` for Go; targeted local search; repository gates). The exceptions are listed per task.

## Batch B1: restore core in `internal/skillsync`

Outcome: `skillsync` can list the backups in a root, validate a selected backup, and preview or apply a restore with the full backup-before-replace, staged-install and rollback contract. It is exercised only through Go tests in temporary homes, with no CLI surface yet. This batch carries the safety-critical logic, so it is reviewed on its own before any user-facing command exists.

### T1: generalize the install path without changing `Sync` behavior

Add a source-relative `src` path to `change` and use it in `installChanges`. Let callers supply the manifest's skill list and source label, and the verbs/summary (or print them at the caller). Lift the backup-root rule and the four-target list out of `Sync` into helpers, and factor the per-destination check-and-compare step out of `planChanges`. Depends on nothing.

Done when `Sync` output and on-disk results are unchanged, all existing `internal/skillsync` and `cmd/agentic-sdd` tests pass without edits to their expectations, and `make test`, `make vet` and `git diff --check` are clean.

### T2: backup discovery and validation (`internal/skillsync/restore.go`)

Implement `ListBackups` (newest first, with malformed entries flagged with a reason), `validateBackupID` (the exact stamp syntax, a single path element, `os.Lstat` directory), and `loadBackup`. `loadBackup` parses the manifest, allowlists `manifest.json` and the four client dirs and `agentic-sdd-*` skill dirs, calls `Lstat` on each client and skill dir, then runs `checkTreeFS` and the regular `SKILL.md` check. Add the home-match check against the current `--home`'s targets. Depends on T1 (the shared target/backup-root helpers).

Done when tests cover an empty or missing root, correct ordering, and a malformed-entry reason. The rejection tests must cover a bad ID syntax, `..`/separator IDs, a missing manifest, unparseable JSON, an unknown top-level entry, a non-`agentic-sdd-*` skill dir, a symlinked backup/client/skill dir, a symlink inside a skill, a missing `SKILL.md` and a home mismatch.

### T3: `Restore` preview and apply

Plan one change per `<client>/<skill>` in the backup against the current home's destinations (`restore` / `replace + back up` / `up to date`). Preview prints the plan and `Preview only. Run with --apply to restore.`. Apply routes through `installChanges` with `os.DirFS(<backup>)` as source and `restore:<ID>` as the manifest source. If there are no changes, it prints that everything is up to date and creates no backup. Depends on T1 and T2.

Done when isolated-home tests show the following:
- preview writes nothing;
- apply restores exactly the backed-up trees and leaves fresh installs, unrelated skills and the selected backup untouched;
- the new pre-restore backup's manifest names the restored ID, and restoring that backup returns the pre-restore state;
- a repeated restore is a no-op with no new backup;
- an injected mid-install failure rolls every destination back (reuse the existing rollback test pattern);
- destination refusals (a non-directory target, a symlink in the installed tree, a bad parent component) happen before any write.

`make test`, `make vet` and `git diff --check` must be clean.

**B1 verification:** `make test`, `make vet`, `git diff --check`, all in temporary homes only.
**B1 state:** not started. **Next action:** wait for combined approval of r1, then explicit authorization to implement B1.

## Batch B2: CLI selection, docs and end-to-end check

Depends on B1 being human-reviewed. Outcome: users can run `agentic-sdd backups`, `agentic-sdd restore <ID> [--apply]`, and the interactive `agentic-sdd restore [--apply]` numbered selection with confirmation. Help, README, Makefile, AGENTS.md and the Docker e2e all match.

### T4: `backups` and `restore` commands

Add both commands to `run`'s switch, with their own flag sets (`--repo`, `--home`; plus `--apply` for `restore`) and interspersed positional parsing for `restore`. Add an `io.Reader` stdin and an interactivity flag to `run`, wired in `main` from `os.Stdin`, and update the existing test call sites. Implement the numbered prompt (blank input cancels; out-of-range or non-numeric input exits 2) and the `y/N` confirmation for `--apply`. A missing ID on a non-interactive stdin exits 2 and names `agentic-sdd backups`. Also extend usage and `help [preview|apply|backups|restore]`. Depends on B1.

Done when `main_test.go` covers:
- `backups` output and an empty root;
- `restore <ID>` preview;
- `restore <ID> --apply` and `restore --apply <ID>`;
- stray args and bad flags (2);
- a not-found ID and a home mismatch (1);
- interactive select → preview; interactive select → `y` → applied; interactive select → `n` → unchanged (0); blank → cancel (0); a bad number (2);
- a non-interactive missing ID (2).

### T5: docs, Makefile and Docker e2e

Update `README.md`: CLI reference rows, the restore `--apply` note, exit codes, and a "Restoring a backup" paragraph that states restore is not a full undo of fresh installs. Add a read-only `backups` Makefile target and no `restore --apply` target. Add `restore.go` to `AGENTS.md`'s Go layout. Extend `tests/e2e-in-container.sh` with apply → `backups` → `restore` preview (no change) → `restore --apply` (old content back, pre-restore backup created) → repeat (no new backup) → restore of the pre-restore backup (new content back). Depends on T4. Task exception: no additional specialist is needed for the doc, Makefile and shell edits, matching `plan.md`'s default for these files.

Done when docs match the implemented help text, and `make docker-e2e` passes. If Docker is unavailable, record that and run `tests/e2e-in-container.sh` with a temporary home instead.

**B2 verification:** `make test`, `make vet`, `git diff --check`, `make docker-e2e` (or the stated fallback). No `restore --apply` or `apply` against a real home.
**B2 state:** not started. **Next action:** after B1's human review, wait for explicit authorization to implement B2.

## Continuation

- Approval reference: `spec.md` `## Approval` (currently draft/pending, r1).
- Current batch: none. First ready batch after approval: **B1**.
- Record changed paths, the checks and the code state they ran against, and set `awaiting human review` at each batch boundary. Passing checks never start the next batch.
