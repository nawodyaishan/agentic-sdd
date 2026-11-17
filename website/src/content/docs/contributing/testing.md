---
title: Testing and verification
description: Test organization, explicit selections, fixtures, CI, and how to read results versus coverage.
---

Contributor testing has two layers. Public guidance on this page tells you exactly which commands to run and how to interpret their output. The detailed engineering inventory lives in `docs/testing/test-cases-report.md` (snapshot commit `f559872`); that report lists declared test cases and is not a passing run, a coverage percentage, or an approval.

## How to read test evidence

Five kinds of evidence exist. Do not confuse them:

| Evidence kind | What it is | Where it lives |
|---|---|---|
| Declared cases | Test functions found by static listing (`go test -list`, Python AST) | `docs/testing/test-cases-report.md` at snapshot `f559872` |
| Executed runs | An actual command run with revision, toolchain, selection, exit code, and log | Your terminal output or `task verify` evidence directory |
| Go statement coverage | Optional measured profile from the contract below | Private local file only, never a site asset |
| Manual walkthroughs | A human following steps and recording observations | Session notes, labeled with participant and date |
| Model behavior | What an agent did on one occasion | Illustrative only, never proof of enforcement |

A passing static text check does not prove model compliance. A declaration count is not a coverage percentage. An author-run walkthrough is labeled `author_walkthrough`, not unfamiliar-user evidence. Anything unavailable is reported as `not_run` or `blocked` with a reason; those words are not `pass`.

## Test organization

Tests are organized by module and selection lane, not by who wrote them:

- **Root Go module** (`go.mod`): CLI, installer, fleet engine, execution engine, check runner, and their package tests. Run with `task test` and `task vet`.
- **Nested TUI Go module** (`experiments/tui/go.mod`): terminal UI tests. These never run from `task test`; they are opt-in through `Taskfile.experimental.yml`.
- **Python harness suites** (`tests/harness/`): stdlib `unittest` static and contract checks. Each suite is selected explicitly by file pattern, never by broad directory discovery.
- **Shell and container drivers** (`tests/*.sh`, `tests/docker-e2e.sh`): environment-gated or containerized journeys. They need their named prerequisite (Docker, a built binary pair) and are not part of the routine contributor loop.
- **Check declarations** (`harness/checks.json`): check-level entries consumed by `cmd/sdd-check` through `task verify`. They declare checks, not individual test cases.

The Feature 008 fence still applies: fenced Feature 005 experimental adapter work, deferred Feature 006 native/profile/dispatch/pilot work, and historical Feature 007 migration exercises stay excluded. Their files may appear in the inventory as historical evidence, but they are never selected by the commands below. See [Development setup](./development.md) for the toolchain and module layout behind these lanes.

## Default core gate

After any Go, skill, or repository edit, run the same three routine checks the pre-commit hook runs (`Taskfile.yml`, `lefthook.yml`, `CONTRIBUTING.md`):

```sh
task test
task vet
git diff --check
```

`task test` and `task vet` (aliases `task test-core` and `task vet-core` via `tests/supported-core-go.sh`) select exactly these 11 supported-core packages with a sanitized offline environment (`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off`, empty `GOFLAGS`):

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

Never run `go test ./...` as a substitute: that wildcard would select fenced `internal/opencodeadapter` tests while still omitting the separate `experiments/tui` module. The same package list appears in `harness/checks.json` for the `vet` and `test` checks and in `tests/harness/README.md`; if those three sources ever disagree, stop and report the drift to the testing initiative instead of picking one silently.

For the full six-check `required` contract (`mod-verify`, `tidy-check`, `vet`, `test`, `build-darwin`, `diff-check`) with private external logs, run `task verify`, which drives `cmd/sdd-check` with `--scope required --workspace . --contract ./harness/checks.json --output <owned-directory>`.

## Explicit Python selections

Python suites always run one file pattern at a time with the stdlib runner. The two routine suites for skill and documentation edits are:

```sh
python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py
python3 -B -m unittest discover -s tests/harness -p test_research_workflow.py
```

These are static text and fixture checks: they read canonical instructions, scenario JSON, and links, and they detect in-memory clause mutations. They launch no client, provider, container, network call, or subprocess, and they prove nothing about model behavior.

Other suites follow the same explicit pattern with their own prerequisites: `tests/harness/README.md` documents receipt, recovery, trusted-CI, and connected suites, several of which need `TMPDIR=/tmp/opencode` or an explicit opt-in variable. Broad discovery across `tests/harness/` is not an approved selection because it mixes explicit static suites, environment-gated fixtures, and historical or fenced files. Do not add Python cases to `harness/checks.json` or count a passing static check as runtime evidence.

## Fixtures and isolation

Automated tests and manual exercises must stay inside temporary directories: Go tests use `t.TempDir()`, CLI experiments use a temporary `--home` (or `task run`, which creates one and deletes it), and synthetic `--repo` directories stand in for real checkouts. Never write into real user skill directories from a test, and never run `task apply` to "check" installer behavior; `task preview` is the read-only way to inspect what installation would do. See [Files and backups](../reference/files-and-backups.md) for the managed stores these fixtures imitate, and [CLI reference](../reference/cli.md) for the flags that accept temporary paths. If your change touches public documentation, follow the style and impact rules in [Contributing docs and changes](./docs-and-changes.md) before opening a pull request.

## Opt-in lanes

Everything beyond the core gate is opt-in and labeled as such. A skip caused by a missing prerequisite (old Go, no Docker, no built binary) is `not_run`, never a pass:

| Lane | Command | Prerequisite |
|---|---|---|
| Nested TUI module tests | `task -t Taskfile.experimental.yml tui:test` | Go 1.26 or newer |
| TUI PTY suites | `task -t Taskfile.experimental.yml tui:pty-test` (plus `tui:fleet-pty`, `tui:mcp-pty`) | Built `asdx`/`asdx-core` pair in `bin/tui-pair/` |
| Container installer check | `task docker-e2e` | Docker with preloaded `golang:1.25-alpine`, offline flags |
| Guided container journeys | `task docker-guide-e2e` | Same container setup |
| TUI vendor drift | `task tui-vendor-check` (also a lefthook pre-commit job) | Go toolchain; read-only, offline; fails when the vendored copy or its stamp no longer matches `internal/skillsync` and `internal/fleet` |
| Docs CI policy contracts | `python3 -B -m unittest discover -s tests/harness -p test_docs_ci.py` | Python 3 only; fixture-based contract tests for the docs launcher and workflow policy; publishes nothing |
| Website and docs checks | `npm --prefix website run docs:check`; `docs:reference:check` with a candidate binary; `docs:check-built` after `docs:site`; `npm --prefix website test` | Node.js `24.22.3` (`website/package.json` engines). A missing toolchain or candidate binary is `not_run`, never a pass |

Hosted CI (`.github/workflows/ci.yml`) runs the trusted gate against candidate bytes as data. The docs workflow (`.github/workflows/docs.yml`) exposes a read-only, non-blocking post-publish observation through `workflow_call`, but no publishing workflow calls it yet, so no observation runs after a release today. On a `v*` tag push, `Release build` (`.github/workflows/release.yml`) builds and manifests artifacts without tokens, and `Release publish` (`.github/workflows/release-publish.yml`, run from the default branch via `workflow_run`) refuses unless the tagged commit is on the default branch, CI on that commit has passed (waiting up to 40 minutes), the manifest matches, `RELEASE_TOOLS_SHA` pins the tooling, and the tap has the cask with no formula. Local runs never reinterpret themselves as trusted-CI evidence. Local snapshot builds (`task release` and its experimental equivalent) build artifacts only: they never publish, tag or upload, and the two snapshot entrypoints (`task release` and `task -t Taskfile.experimental.yml release:snapshot`) report the same prerequisites.

## Optional coverage contract

Statement coverage is optional and has no threshold or target percentage. If you choose to measure it, use exactly this contract and keep every artifact private:

```sh
mkdir -p /tmp/opencode/coverage
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= go test -count=1 -covermode=set -coverprofile=/tmp/opencode/coverage/core.cover . ./internal/version ./internal/skillsync ./internal/harnesscheck ./internal/harnessevidence ./internal/appops ./internal/buildinfo ./internal/fleet ./internal/execution ./cmd/agentic-sdd ./cmd/sdd-check
go tool cover -func=/tmp/opencode/coverage/core.cover
```

Rules for this contract:

- Run `go tool cover` only if the test run succeeded.
- Never substitute `./...` or add `-coverpkg=./...`; keep the nested TUI module separate and opt-in.
- Keep profiles, HTML reports, and summaries in owned scratch such as `/tmp/opencode/` (honoring `TMPDIR`); never commit them or publish them as site assets, since profiles can reveal source.
- Report per-package denominators with `no test files` and not-in-profile states intact. Do not collapse the result into a single whole-repository percentage.

A measured result must record the exact revision and dirty state, Go version, OS and architecture, tool versions, selection and sanitized environment, UTC interval, full command and exit code, skips and testless or omitted packages, and profile or summary digests. Without that provenance, and without actually running the measurement, coverage for this feature is `not_measured`, with the reason (no authorized measurement run), the `f559872` declaration snapshot, and the owner (the testing initiative) stated explicitly.

## Drift belongs to the testing initiative

If you notice that a selector, required set, fence boundary, or test file changed underneath these pages — for example `tests/supported-core-go.sh`, `harness/checks.json`, `Taskfile.yml`, or a Go or Python test file was added, removed, or re-scoped — do not invent a new generator, gate, or task to fix the docs. Emit one drift observation naming the changed selector or path, the snapshot it contradicts, and the affected claim, and assign it to the testing initiative. Preserve the `f559872` snapshot wording; regenerating or editing historical counts is that initiative's supported method, and none was supplied at this snapshot.
