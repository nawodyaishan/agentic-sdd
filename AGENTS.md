# Repository guidance

## Scope

This repository holds the canonical Agentic SDD skill files and a Go CLI that installs them into user skill directories. Treat `skills-files/` as the source of truth. Keep `backups/` local and out of Git; it may contain private previous skill content.

## Go layout

- `cmd/agentic-sdd/main.go`: CLI flags, home-directory default, and process exit behavior.
- `internal/skillsync/sync.go`: planning, backup, installation, and rollback.
- `internal/skillsync/files.go`: path checks, tree comparison, and copying.
- `internal/skillsync/sync_test.go`: isolated-home tests; never write to real user skill directories in tests.

Keep client destinations and skill selection together in `internal/skillsync`. Prefer Go standard library code and small, explicit functions. Do not add a package or interface solely to move a few lines of code.

## Safe changes

- Preview must remain the default. `--apply` is the only mode that installs skills.
- Back up every existing skill that will be replaced before modifying destinations.
- Keep unrelated skills untouched and reject symlinks or special files in managed skill trees.
- Preserve the no-op behavior: identical skills should not be replaced or backed up again.
- Update README commands and the Makefile whenever the CLI path or flags change.

## Verification

Run `make test`, `make vet`, and `git diff --check` after Go changes. Use a temporary home in tests. `make preview` is a useful read-only check against the local installation. Install the repository's Lefthook checks with `make hooks-install`.
