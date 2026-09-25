# Repository guidance

## Scope

This repository holds the canonical Agentic SDD skill files and a Go CLI that installs them into user skill directories. Treat `skills-files/` as the source of truth. Keep `backups/` local and out of Git; it may contain private previous skill content.

## Go layout

- `cmd/agentic-sdd/main.go`: CLI flags, home-directory default, process exit behavior, and interactive backup selection (`backups`, `restore`).
- `internal/skillsync/sync.go`: planning, backup, installation, and rollback (shared by apply and restore).
- `internal/skillsync/files.go`: path checks, tree comparison, and copying.
- `internal/skillsync/manifest.go`: the backup manifest format (additive: format 1 fields plus format 2's location, tool version, timestamps and per-skill entries) and its tree digest.
- `internal/skillsync/restore.go`: listing and validating backups, and restore's planning/apply.
- `internal/skillsync/sync_test.go`, `manifest_test.go`, `restore_test.go`: isolated-home tests; never write to real user skill directories in tests.
- `internal/skillsync/source.go`: `fs.FS`-based source resolution (embedded `skills-files/` by default, `--repo` on disk otherwise).
- `internal/version/version.go`: build-time version metadata, injected via `-ldflags` at release build time; `dev`/`none`/`unknown` defaults for local builds.
- `skillsfiles.go`: root-level `go:embed all:skills-files`, so an installed binary (e.g. via Homebrew) needs no repository checkout.

Keep client destinations and skill selection together in `internal/skillsync`. Prefer Go standard library code and small, explicit functions. Do not add a package or interface solely to move a few lines of code.

## Safe changes

- Preview must remain the default. `--apply` is the only mode that installs skills.
- Back up every existing skill that will be replaced before modifying destinations.
- Keep unrelated skills untouched and reject symlinks or special files in managed skill trees.
- Preserve the no-op behavior: identical skills should not be replaced or backed up again.
- Update README commands and the Makefile whenever the CLI path or flags change.

## Verification

Run `make test`, `make vet`, and `git diff --check` after Go changes. Use a temporary home in tests. `make preview` is a useful read-only check against the local installation. Install the repository's Lefthook checks with `make hooks-install`.

## SDD feature conventions

This repository has no top-level product SRS or technical spec. The global roadmap is `ROADMAP.md`: select the next feature from its "Recommended roadmap" section (Now / Next / Later / Explicitly not doing) and check its "Kill criteria and signals to watch" before starting one. Each feature lives in `specs/<nnn-slug>/` (`spec.md`, `plan.md`, `tasks.md`); the latest is `specs/003-restore-backups/`, and the next number is 004. The combined human approval for a feature's spec/plan/tasks is recorded once, in that feature's `spec.md` under `## Approval`. Batch state and continuation notes live in that feature's `tasks.md`.

`preview` (default) versus `apply`/`--apply` is this repo's dev-versus-live boundary: approval to develop or test installer code never authorizes running `apply` against a real user home; only an explicit, separately authorized real-home apply does.

## Release and publish

`.goreleaser.yml` builds darwin (amd64+arm64, merged into a universal binary) and publishes a Homebrew formula to `nawodyaishan/homebrew-tap`, driven by `.github/workflows/release.yml` on a `v*` tag push. `make tag V=vX.Y.Z MSG="..."` creates the tag; `make release` runs a local snapshot build (no publish, no tag required) to verify the pipeline. Writing a real `v*` tag, pushing it, or generating/rotating `HOMEBREW_TAP_TOKEN` are live-system operations requiring their own authorization — developing or testing this pipeline's code does not authorize them.
