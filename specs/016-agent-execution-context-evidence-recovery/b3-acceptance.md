# Feature 016 — B3 acceptance evidence (2026-11-13)

**Status:** Implementation complete for B1–B3; **awaiting human review**. This is not feature acceptance, a release verdict, a security certification, or a reviewer-isolation claim.
**Baseline:** commit `a8f1c1c` plus the reviewed Feature 016 working-tree changes. No commit, tag, push, or release was performed.
**Approval:** combined spec/plan/tasks approval is recorded in [spec §14](spec.md#14-approval). The owner authorized B1–B3 and the dependency-safe parallel waves in [tasks.md](tasks.md#parallel-execution-authorization-2026-11-13).

## Commands run and exits

All Go commands use the repository's sanitized environment (`GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, `GOENV=off`, `GOWORK=off`, `GOFLAGS=`).

| Command | Exit | Result |
|---|---|---|
| `go test ./internal/execution ./cmd/agentic-sdd -count=1` | 0 | Both packages pass. |
| `go test -race ./internal/execution ./cmd/agentic-sdd -count=1` | 0 | Race detector clean. |
| `task test` | 0 | All 11 supported-core packages pass. |
| `task vet` | 0 | All 11 supported-core packages pass static analysis. |
| `git diff --check` | 0 | No whitespace errors. |
| `go mod verify` | 0 | Module cache verified; `go.mod` remains dependency-free. |
| `python3 -B -m unittest tests.harness.test_appops_core_selection tests.harness.test_supported_core_go tests.harness.test_taskfile` | 0 | 28 tests pass, including 11-package parity. |
| `PYTHONPATH=tests/harness python3 -B -m unittest tests.harness.test_trusted_ci` | 0 | 14 tests pass. |
| Disposable-repo CLI smoke | 0 | `run init` → `task brief` → `run check` → `run record` → `run status` → `run clean`; `git status` stayed clean; the default run root was Git-invisible. |

## Acceptance criteria

| ID | Verdict | Evidence |
|---|---|---|
| **AC-01** | **Pass** | `run init|status|check|record|clean`, `task brief`, and `review package` are dispatched with family help; `cmd/agentic-sdd` tests pass. `agentic-sdd help run|task|review` prints the topics. |
| **AC-02** | **Pass** | All commands call `internal/execution`; `go list -deps ./internal/execution` has no `cmd/agentic-sdd`, `internal/appops`, or `experiments/tui` imports. CLI parsing/rendering stays in `cmd/agentic-sdd`. |
| **AC-03** | **Pass** | Workspace tests cover run IDs, 0700/0600 layout, self-ignoring `runs/.gitignore`, collisions, symlinks, traversal, locking, stale-lock reclaim, and scoped clean. The smoke test left `git status` clean. |
| **AC-04** | **Pass** | 18 task-brief scenarios cover `T1` vs `T10`, checkbox tasks, fenced blocks, nested `####`, non-task boundaries, CRLF, duplicate/missing IDs, FR/AC ranges, and scope-digest progress-only changes. |
| **AC-05** | **Pass** | Check tests cover pass/fail/timeout/signal/output cap/missing program/failure far from tail/summary truncation; the CLI test asserts full log paths and result mapping. |
| **AC-06** | **Pass** | 18 review-package tests cover committed/staged/unstaged/untracked, rename/delete/mode, binary, large, sensitive, symlink, submodule gitlink, conflicted index, head mismatch, invalid base, no mutation, and checkpoint integration. The CLI test asserts a partial package exits 3 with `sensitive_path`. |
| **AC-07** | **Pass** | Checkpoint tests cover state precedence, source/tasks/batch freshness, `progress_updated`, missing artifacts, interrupted checks, stale locks, and no auto-resume. A record alone yields `awaiting-verification`. |
| **AC-08** | **Pass** | Failure tests and skill references distinguish provider/transport, model quality, test/command, partial-write, and permission/billing/data-policy failures; no automatic model/provider/reviewer/batch escalation is implemented. |
| **AC-09** | **Pass** | `review-resolution.md` and the verification-review skill define Critical/Important/Minor separately from `valid`/`disputed`/`needs evidence`; `internal/skillsync` asset tests pass. |
| **AC-10** | **Pass** | The independent-review reference states request-only fresh review, the proposal fields, refusal when isolation/permissions are unverified, and the open `IR-06-01` boundary. No automatic reviewer dispatch exists in the code. |
| **AC-11** | **Pass** | The orchestration skill and its dispatch/failure references describe the run/task/review commands, exact brief/artifact references, five failure categories, bounded correction, and checkpoint/stale rules; no new skill name or profile was added. |
| **AC-12** | **Pass** | The full 11-package suite, `task test`, `task vet`, existing CLI tests, help-topic tests, JSON schema tests, and `git diff --check` pass. Existing commands and output paths were not rewritten. |
| **AC-13** | **Pass** | CLI tests create a direct run, assert `ScopeDirect`, refuse `task brief` with a clear message, and default `run record`/`run check` to `direct`; no feature files are fabricated. |

## Explicitly not run / not claimed

- No real-home `apply`, no real `~/.config/opencode` write, no backup restore against a user home.
- No product model, network, MCP, OAuth, telemetry, or paid-inference call.
- No fresh reviewer dispatch or reviewer-isolation certification; `IR-06-01` remains open.
- No tag, release, push, deployment, or Homebrew tap change.
- No model-based subagent smoke test; it remains optional and separately authorized.

## Residual limits

- `run check` is not a sandbox; the invoker's permissions and command side effects apply.
- Symlink checks and private permissions reduce cooperative-tool mistakes; they do not protect against another process running as the same user or close all TOCTOU windows.
- Review-package sensitive-path handling is pattern-based omission, not redaction or a shareability guarantee.
- A checkpoint is an observation; `spec.md` §Approval and `tasks.md` remain authoritative.

## Next action

Human review of the B1–B3 implementation and this evidence. Do not treat a green suite as acceptance or authorize a commit, release, or real-home operation from it.
