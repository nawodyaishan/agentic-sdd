# Tasks: CLI quality and usability

Status: approved for implementation; see the combined approval record in `spec.md`.

## Batch B1 — usable and testable CLI

State: **awaiting human review**. T1 through T3 are implemented and verified. Next action: review this batch; no real-home apply has been run.

### T1 — CLI behavior

Add command parsing, help, clear usage errors, and explicit exit behavior in `cmd/agentic-sdd`; retain legacy flags and preview default. Add focused tests using temporary homes. Done when the command acceptance cases in `spec.md` pass.

Result: complete. Explicit commands, legacy flags, help, usage exit status, and isolated-home tests are in `cmd/agentic-sdd/main.go` and `main_test.go`.

### T2 — sync clarity and documentation

Refactor `Sync` into small discovery/planning/execution steps without changing the safety contract; add focused regression coverage where the extraction exposes a meaningful edge case. Update README and Makefile to show new commands. Done when existing safety tests and required gates pass.

Result: complete. `Sync` now delegates skill discovery, change planning, and installation. Existing backup, rollback, no-op, and refusal tests pass. README and Makefile show the new commands.

### T3 — Docker end to end test

Add a container based test command that compiles and runs the CLI using a temporary repository and home inside Docker. Verify preview, apply, backup, no-op repeat apply, and invalid input. Document how to run it before applying to a real home. Done when the script is reviewable and, if Docker is available, passes.

Result: complete. `tests/docker-e2e.sh` runs `tests/e2e-in-container.sh` in a Go container, with the repository mounted read-only and the test home created inside the container. `make docker-e2e` passes.

## Verification and handoff

Principal specialist for all tasks: `golang-pro`, assigned in `plan.md`. Run command tests, `make test`, `make vet`, `git diff --check`, Docker end to end, and read-only `make preview`. Record changed paths and check results here after implementation, then set this batch to `awaiting human review`. Do not run an apply against the real user home.

Verification for this code state: `make test`, `make vet`, `go test -race ./...`, `git diff --check`, Go formatting, `make docker-e2e`, and shell syntax checks passed. `make preview` also completed read-only. Changed paths: `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/main_test.go`, `internal/skillsync/sync.go`, `Makefile`, `README.md`, `tests/docker-e2e.sh`, and `tests/e2e-in-container.sh`. No destination skill directories were modified by this batch.
