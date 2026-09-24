# Plan: CLI quality and usability

Status: approved for implementation; see the combined approval record in `spec.md`.

## Approach

- Keep `cmd/agentic-sdd` as the CLI boundary. Use a small `run(args, stdout, stderr)` path with `flag.FlagSet` and explicit exit codes; keep `main` limited to process exit. Recognize a leading `preview`, `apply`, or `help` command, then parse flags. Preserve flag-only legacy invocations and reject ambiguous command/`--apply` combinations.
- Keep client destinations and skill discovery in `internal/skillsync`. Extract the current scan and change calculation from `Sync` into small functions, preserving deterministic output order and the current backup/stage/rollback sequence. Avoid new packages or interfaces.
- Use temporary repository and home directories for CLI and sync tests. Check read-only preview, command compatibility, help/errors, no-op apply, backups, and refusal paths. Keep the existing safety tests.
- Update README examples and Makefile preview/apply recipes to use explicit commands while documenting legacy forms.
- Add a Docker test entry point that runs the compiled CLI against a temporary source and home inside a Go container. It must check preview leaves files unchanged, apply creates backups and installs, a second apply is a no-op, and invalid input fails. This test never mounts the real home.

## Evidence and tools

- CodeGraph explored `main`, `Sync`, and filesystem helpers to identify the call path and blast radius. `cmd/agentic-sdd/main.go` and `internal/skillsync/sync_test.go` are the direct change/test points.
- Exa found the [official Go `flag` documentation](https://pkg.go.dev/flag), which supports independent `FlagSet` values for commands and `ContinueOnError` for testable parsing.
- Context7 was requested for planning but its plugin is not installed or callable in this session. The Go standard library docs provide the needed API contract. No third-party dependency is planned.
- Principal specialist: installed `golang-pro`, for Go CLI design and safety tests. Repository instructions override its general advice where the existing small program needs simpler code.

## Checks and recovery

Run focused command tests, `make test`, `make vet`, `git diff --check`, and the Docker end to end command before any real-home apply; use `make preview` as a read-only smoke check. Apply tests use temporary homes only. If Docker is unavailable, record that gate as unrun and do not claim container verification. If a regression appears, keep the public flags and sync contract intact while correcting the narrow code path; no real user installation is part of this packet.

## Design boundary

The command layer decides mode and exit status; `skillsync` owns filesystem actions. No change to managed targets, manifest schema, backup location, or skill selection. Current backup and rollback guarantees are acceptance gates, not optional cleanup.
