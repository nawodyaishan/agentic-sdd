---
title: Development setup
description: Toolchains, two-module Go layout, local builds, hooks, and routine contributor checks.
---

Agentic-SDD separates the root Go CLI/installer module, a nested terminal UI Go module, optional Python stdlib harness suites, and an isolated Node.js documentation workspace. Install only the toolchain required for the surface you are changing.

## Toolchains by contribution surface

| Surface | Required tools | Version / pin | Source of truth |
|---|---|---|---|
| Root CLI, installer, and skills | Git, Go, Go Task, Lefthook | Go `1.23` directive; Task `3.54.0` in CI | `go.mod`, `CONTRIBUTING.md`, `Taskfile.yml`, `.github/workflows/ci.yml` |
| Nested TUI companion (`experiments/tui`) | Go (newer directive), Git | Go `1.26` directive (`go1.27.1` in release/snapshot tasks) | `experiments/tui/go.mod`, `Taskfile.experimental.yml`, `.github/workflows/release.yml` |
| Harness static & contract suites | Python 3 (optional `jsonschema` 4.x for receipt/contract JSON suites) | Python `3.9+` (`unittest` stdlib) | `tests/harness/README.md`, `AGENTS.md` |
| Website & docs tooling (`website/`) | Node.js and npm (only for site/docs work) | Node `24.22.3` (`>=24.22.3 <25.0.0`) | `website/.node-version`, `website/package.json` |
| Container E2E checks (optional) | Docker with preloaded `golang:1.25-alpine` | Offline `--pull=never --network none` | `Taskfile.yml`, `tests/docker-e2e.sh`, `harness/checks.json` |
| Release snapshot build (optional) | GoReleaser plus `go1.27.1` and SHA-256 tool | GoReleaser `v2.14.0` | `Taskfile.experimental.yml`, `.github/workflows/release.yml` |

Only Go is needed for root Go and skill work. Node.js is never required unless you are building or testing the documentation site under `website/`.

## Clone, inspect, and install hooks

Clone the repository and verify your local Go and Task binaries before running checks:

```sh
git clone https://github.com/nawodyaishan/agentic-sdd.git
cd agentic-sdd
go version
task --version
task hooks-install
```

`task hooks-install` runs `lefthook install` (`Taskfile.yml`). On commit, `lefthook.yml` checks `gofmt -l` on staged `*.go` files, runs `task test` and `task vet`, and runs `git diff --cached --check`.

Run `task help` (or `task`) to list root tasks (`Taskfile.yml`). Experimental and TUI tasks live separately in `Taskfile.experimental.yml` so they do not run during routine CLI/skill checks.

## Two-module layout and repository map

The repository uses two separate Go modules rather than a Go workspace (`GOWORK=off` is enforced in check runners):

- **Root module (`go.mod`, `module agentic-sdd`, `go 1.23`)** builds `asdx-core` (`cmd/agentic-sdd`) and `sdd-check` (`cmd/sdd-check`), and embeds `skills-files/` via `skillsfiles.go` (`//go:embed all:skills-files`) so built binaries work without a repository checkout.
- **Nested TUI module (`experiments/tui/go.mod`, `module agentic-sdd/experiments/tui`, `go 1.26`)** depends on the root module through `replace agentic-sdd => ../..` and builds the `asdx` companion (`experiments/tui/cmd/tui-companion`).

Key directories and owning files (`AGENTS.md`, `CONTRIBUTING.md`):

- `skills-files/agentic-sdd-*/`: canonical portable skill instructions (source of truth; see [Skills reference](../reference/skills.md)).
- `opencode-pack/`: optional OpenCode orchestration skill (`skills/agentic-sdd-orchestrate/`), 14 shipped agent profiles (`agents/`), four free-first presets (`presets/free-first/`), and two non-default fleet adapters (`adapters/`).
- `cmd/agentic-sdd/`: `asdx-core` CLI entrypoint (`main.go`), stdlib `flag.FlagSet` handlers (`fleet.go`, `mcp.go`, `mcp_write.go`, `run.go`, `task.go`, `review.go`, `guide.go`, `tui.go`), and default experimental compile guard (`opencode_disabled.go`, `//go:build !agentic_sdd_experimental`).
- `cmd/sdd-check/`: deterministic check-runner binary used by `task verify`.
- `internal/version/`: build-time version, commit, and date metadata (`dev`/`none`/`unknown` defaults for local builds).
- `internal/skillsync/`: planning, backup, installation, rollback, client target paths, source loading (`embed.FS` or `--repo`), and MCP config transactions.
- `internal/appops/`: typed seams shared by CLI/TUI workflows (`ops.go`, `fleet.go`, `mcp.go`, `mcp_write.go`, `routing.go`, `teaching.go`, `update.go`).
- `internal/buildinfo/`: canonical/legacy command names, protocol revision, distribution markers, and companion handshake validation.
- `internal/fleet/`: offline fleet parser (`fleet.go`, `parse.go`), source resolver, deterministic model renderer, plan/lock, format-4 sidecar manifest, and offline doctor.
- `internal/execution/`: typed run workspace, locks, checkpoint freshness, task-brief extraction, record ingestion, check runner, and review-package assembly.
- `internal/harnesscheck/` and `internal/harnessevidence/`: contract runner, Linux process-group isolation, and receipt snapshot/validation/export.
- `internal/opencodeadapter/`: fenced Feature 005 experimental adapter (excluded from default builds and the core test gate).
- `experiments/tui/`: canonical TUI shell, routes, companion entrypoint (`cmd/tui-companion`), and companion launcher (`cmd/internal/tuirun`).
- `tests/` and `harness/`: `tests/supported-core-go.sh`, `tests/harness/` Python suites, `tests/tui/` PTY suites, `harness/checks.json`, and shell/container drivers.
- `website/`: isolated Starlight documentation project (`package.json`, `docs-manifest.json`, `astro.config.mjs`, `plugins/`, `scripts/`, `src/content/docs/`, `tests/`).
- `docs/` and `specs/`: engineering documentation, testing inventory (`docs/testing/test-cases-report.md`), CodeGraph guide (`docs/codegraph.md`), product vision (`docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md`), and feature specs.

Keep local generated state out of Git (`AGENTS.md`, `CONTRIBUTING.md`): `backups/`, `.agentic-sdd/runs/` (self-ignoring execution scratch), `.codegraph/`, `bin/`, and `dist/`.

## Build and preview locally

Build the canonical `asdx-core` binary for your host OS (`Taskfile.yml`):

```sh
task build
```

This writes `bin/asdx-core` with `-trimpath`. To cross-compile macOS (`darwin/amd64` and `darwin/arm64`) binaries for `agentic-sdd` and `sdd-check` into `bin/`, run `task build-darwin`.

To build a local identity-matched four-binary pair (`asdx`, `asdx-core`, `agentic-sdd`, `agentic-sdd-tui`) into `bin/tui-pair/` without a release tag (requires Go `1.26` or newer):

```sh
task -t Taskfile.experimental.yml tui:pair
```

To inspect how canonical `skills-files/` changes compare against your local skill directories without writing any files:

```sh
task preview
```

`task preview` executes `go run ./cmd/agentic-sdd preview` (`Taskfile.yml`). Preview is read-only and is the safe default. Avoid running `task apply` during development unless you explicitly intend to modify your real user skill directories and write a local backup; use `task run` to try the interactive numbered CLI guide inside an auto-deleted temporary `--home`, or pass explicit `--home` temporary directories when testing CLI commands manually. See [CLI reference](../reference/cli.md) and [Files and backups](../reference/files-and-backups.md) for command flags and store locations.

## Routine checks and explicit selections

Run the three routine repository checks after Go, skill, or repository edits (`AGENTS.md`, `CONTRIBUTING.md`):

```sh
task test
task vet
git diff --check
```

`task test` and `task vet` (and their `test-core` / `vet-core` aliases via `tests/supported-core-go.sh`) run Go with a sanitized offline environment (`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS=`) against the exact 11 supported-core packages:

```text
.
./internal/version
./internal/skillsync
./internal/harnesscheck
./internal/harnessevidence
./internal/appops
./internal/buildinfo
./internal/fleet
./internal/execution
./cmd/agentic-sdd
./cmd/sdd-check
```

Never run `go test ./...` on the root module as a substitute for `task test`: that wildcard would select fenced `internal/opencodeadapter` tests while still omitting the separate `experiments/tui` module.

When editing skills or documentation, also run the two explicitly selected Python static suites by file pattern (`AGENTS.md`, `tests/harness/README.md`):

```sh
python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py
python3 -B -m unittest discover -s tests/harness -p test_research_workflow.py
```

Always select Python suites individually with `-p <suite>.py`. Broad discovery (`python3 -m unittest discover -s tests/harness`) is not an approved selection because it mixes explicit static suites, environment-gated fixtures, and historical/fenced files.

For style, impact review, and page-ownership rules, follow [Contributing docs and changes](./docs-and-changes.md).

To run the full six-check `required` contract (`mod-verify`, `tidy-check`, `vet`, `test`, `build-darwin`, `diff-check`) with private external logs via `cmd/sdd-check`, run `task verify`. For TUI, PTY, container, website, and coverage commands, see [Testing and verification](./testing.md).

## Source discovery with CodeGraph

CodeGraph is optional local repository infrastructure (`AGENTS.md`, `docs/codegraph.md`) for tracing symbols, call paths, and blast radius without manual grep/read loops. It does not change product behavior, approve features, or replace tests.

1. Check index status in an ordinary repository checkout (never in a home directory, filesystem root, `backups/`, or private vault):

```sh
codegraph status --json
```

2. If no `.codegraph/` index exists yet and `codegraph` is installed, initialize it once:

```sh
codegraph init -y
codegraph status --json
```

3. Query via the MCP `codegraph_explore` tool (or `codegraph explore`, `codegraph node`, `codegraph callers`, `codegraph impact` on the CLI) before editing indexed code.
4. After bulk edits, branch switches, or before a review, sync the local index:

```sh
codegraph sync
codegraph status
```

If `codegraph` is unavailable or returns a staleness banner, read the affected source files directly and never put secrets or private paths into queries.

## Safe change boundaries for Go and skills

- Keep `skills-files/` as the canonical source of skill instructions and `opencode-pack/` as the optional pack source (`AGENTS.md`, `CONTRIBUTING.md`).
- Preserve `preview` as the default CLI mode; `--apply` is the only install mode, and existing changed skills must be backed up before replacement.
- Keep unrelated files untouched, preserve no-op idempotence, and reject symlinks or special files in managed skill trees.
- Keep all automated tests and manual exercises isolated to temporary directories (`t.TempDir()`, temporary `--home`, synthetic `--repo`); never write into real user skill directories from tests.
- Keep CLI flag parsing and exit behavior in `cmd/agentic-sdd/`, installer/backup/MCP transaction mechanics in `internal/skillsync/`, and small explicit stdlib functions without speculative wrapper interfaces.
- When `cmd/agentic-sdd` commands, flags, or `skills-files/` / `opencode-pack/` metadata change, note that `website/scripts/reference.mjs` and `cmd/agentic-sdd/documentation_reference_test.go` validate reference parity (`go test ./cmd/agentic-sdd -run Documentation` runs independently without Node.js).
