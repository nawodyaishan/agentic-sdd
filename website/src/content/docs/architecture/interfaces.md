---
title: Architecture interfaces
description: How the CLI dispatches, how the TUI hosts one write, the typed seams, and the fail-closed pair handshake.
---

Two installed binaries share one engine: `asdx-core` (CLI, `cmd/agentic-sdd`) and `asdx` (TUI, `experiments/tui`). They are separate Go modules and separate processes. This page describes the interfaces between them and the packages underneath, including where the CLI bypasses the typed seam on purpose.

If you want the flag table instead of the call graph, use [`reference/cli`](../reference/cli.md).

## CLI dispatch is standard-library `flag`, per handler

`cmd/agentic-sdd/main.go` is the whole entry story:

1. `main` first calls `runLegacyCompatibility`. In a canonical `asdx-core` build both `legacyTarget` and `legacyName` are empty, so it returns `handled=false` and the normal CLI continues (`TestCanonicalBuildHasNoLegacyIdentity` in `cmd/agentic-sdd/tui_test.go`).
2. `resolveEnv` decides terminal facts once (`cmd/agentic-sdd/env.go`); `AGENTIC_SDD_TRACE` optionally attaches a tracer.
3. `runEnv` switches on the first argument: `preview`, `apply`, `version`, `help`, `backups`, `logs`, `restore`, `guide`, `tui`, `fleet`, `mcp`, `run`, `task`, `review`. Anything that does not start with `-` is an unknown command with exit `2`.
4. Every flag-parsing handler owns a `flag.NewFlagSet(..., flag.ContinueOnError)` whose output is `io.Discard`; `flag.ErrHelp` is turned into usage plus exit `0`, and every other parse error into exit `2`. There is no global flag registry and no command that silently accepts another command's flag.

The execution family repeats that shape in `newExecutionFlagSet` (`cmd/agentic-sdd/execution_cli.go`) with `--repo`, `--run-root` and `--format`, and maps typed service errors to public exit codes in `executionErrorCode` (usage/unsafe/conflict `2`, missing run for `status` `1`, partial package `3`, anything else `1`).

Bare `asdx-core` is preview: `command` starts as `"preview"` and only the literal `apply` word or the legacy `--apply` flag changes it.

## The call graph is mixed, not a façade

Some families go through `internal/appops`; others call their engine directly. That is the current architecture, and pretending otherwise would mislead a maintainer debugging a stack trace.

| CLI family | Actual path | Source |
|---|---|---|
| `preview` (default) | `skillsync.SyncWithOptionsResult(..., false, ...)` directly | `cmd/agentic-sdd/main.go` |
| `apply` | `runApplyWithLog` → `appops.NewInstallRequest` → `appops.PreviewInstall` → `appops.ApplyInstall(token)` | `cmd/agentic-sdd/logs.go` |
| `restore` | `appops.NewRestoreRequest` → `PreviewRestore` → `ApplyRestore(token)` | `cmd/agentic-sdd/logs.go` |
| `mcp status/doctor/preview` | `appops.MCPStatus`, `appops.MCPDoctor`, `appops.MCPPreview` | `cmd/agentic-sdd/mcp.go` |
| `mcp plan/apply/restore` | `appops.NewMCP*Request` then `MCPPlanPreview`, `MCPApply`, `MCPRestore` | `cmd/agentic-sdd/mcp_write.go` |
| `fleet status/plan/apply/restore/doctor` | `internal/fleet` directly (`BuildStatus`, `BuildPlan`, `LoadPlan`, `ApplyPlan`, `RunDoctor`) | `cmd/agentic-sdd/fleet*.go` |
| `run` / `task` / `review` | `internal/execution` directly (`InitRun`, `RunCheck`, `IngestRecord`, `BuildReviewPackage`) | `cmd/agentic-sdd/run.go`, `task.go`, `review.go` |
| `backups` / `logs` | `skillsync.ListBackups` and the local log store | `cmd/agentic-sdd/*.go` |
| `guide` | interactive wizard; on confirmation calls `skillsync.SyncWithOptionsResult(..., true, ...)` directly | `cmd/agentic-sdd/guide.go` |
| `tui` | resolve and start the sibling `asdx` | `cmd/agentic-sdd/tui.go` |

The TUI uses the seam for every write it offers, which is why install, restore, fleet and MCP all appear in `tui.OpsHostOptions` (`experiments/tui/ops_host.go`). The CLI's direct fleet and execution calls are intentional: those families are standalone, and `internal/fleet`/`internal/execution` expose typed errors the CLI maps to exit codes.

## Typed seams: immutable request, preview token, stale result

`internal/appops` is not a universal façade, but its request types are the contract both front-ends share:

- `InstallRequest` and `RestoreRequest` have unexported fields. `NewInstallRequest`/`NewRestoreRequest` validate home, absolute paths, pack/preset combinations, the client selection and the backup ID before anything touches the filesystem; accessors return copies.
- `PreviewInstall` and `ApplyInstall` are a pair. `ApplyInstall` re-runs the preview into `io.Discard`, recomputes `installPreviewToken(req, current)` and refuses with `ErrNeedsPreview` if the token no longer matches. The token version string is `appops-preview-v1`. Nothing is written before that check.
- A refused apply carries the fresh plan (`refusedApply`) so a caller can show what changed; it carries no token.
- `OperationID.StaleFor` gives a slow earlier result no authority over newer state: a different request or an older generation is stale (`internal/appops/requests.go`).

### Diagram D02 — the pairing arrow

The overview carries D02's full text alternative. The interface-relevant part is the parent-side sequence in `launchTUI`:

1. `resolveTUICompanion` resolves the running executable through `filepath.EvalSymlinks`, then requires a same-directory `asdx` that is a regular, non-symlinked, executable file. There is no `PATH` search and no shell.
2. `queryTUICompanionProtocol` runs `asdx --protocol-info` with no stdin, a bounded stdout/stderr buffer and `context.WithTimeout`. A timeout, non-zero exit or over-budget response fails.
3. `buildinfo.ParseProtocolInfo` rejects unknown fields and trailing data.
4. `buildinfo.VerifyCompanion` checks product, protocol revision `2`, both distribution markers, both identities for placeholders, then identity equality. Distribution is deliberately excluded from the match.

Every failure exits `1` through `failTUILaunch`, which never substitutes preview. The handshake is a compatibility check, not authentication or write authorization; `buildinfo.go` says so in the function comment.

## TUI state and write host

`tui.App` owns presentation state; `tui.OpsHost` owns the single write slot. That split is the reason a busy screen cannot be bypassed by a key sequence:

- `BeginWrite` takes `OpsHostOptions` seams, claims the slot, and returns a `tea.Cmd` that runs outside `Update`/`View`. `ErrWriteInProgress` is returned, not queued, when a write is active (`TestOpsHostRefusesSecondConcurrentWriteWithoutQueueing`).
- `WriteKind` enumerates what may run: install, restore, fleet apply, fleet restore, source save, MCP apply, MCP restore. A nil optional seam is a refusal (`ErrFleetUnavailable`, `ErrMCPUnavailable`), never a fake success.
- The result message is emitted only after the operation log is flushed and closed, so the root model can defer exit until the record exists.
- `WritePolicyText` is one constant shown before confirmation; `TestAppBusyFooterReplacesQuitHintAndPolicyIsShownBeforeExecution` asserts the policy is shown and the busy footer carries `WRITE-IN-PROGRESS`.
- `experiments/tui/ops_host.go` production seams default to `appops.ApplyInstall`, `appops.ApplyRestore`, `appops.ApplyFleet`, `appops.ApplyFleetRestore`, `appops.MCPApply` and `appops.MCPRestore`; tests inject fakes.

The launch gates are part of the interface too. `tuirun.Run` (`experiments/tui/cmd/internal/tuirun/run.go`) rejects a non-terminal stdin/stdout and an `AGENTIC_SDD_FORCE_INTERACTIVE=1` scripted run before a program is created. The CLI repeats the gate in `runTUICommand` using `RealStdinTTY`/`RealStdoutTTY`, which ignore the force flag (`cmd/agentic-sdd/env.go`).

## Pairing, legacy names and forwarding

- Legacy CLI `agentic-sdd` is the same source dual-built with `-X main.legacyTarget=asdx-core -X main.legacyName=agentic-sdd`. Only exact `--help`, `-h` and `--version` are answered locally; every other invocation is replaced with the canonical sibling after a stderr-only notice (`runLegacyCompatibility`, `cmd/agentic-sdd/main.go`). A missing, symlinked or non-executable target is a closed failure with rebuild guidance.
- Legacy TUI `agentic-sdd-tui` mirrors that in `experiments/tui/cmd/tui-companion/main.go`. `--protocol-info` is forwarded as pure JSON with no notice, so an older parent can still verify or reject the pair cleanly.
- The parent registers shutdown signals before starting the child and forwards them, so a signal during launch cannot orphan a child holding the alternate screen (`launchTUI`, `TestLaunchTUIQueuesSignalBeforeChildStarts`). The parent returns the child's exit code; a signalled child becomes exit `1` with an abnormal-termination message rather than an invented code.
- `tuiCompanionArgs` forwards only flags the user actually supplied, so the companion keeps its documented defaults, including `--with-opencode=true`. The CLI's own default remains pack-off.

## Capability markers are an interface, not a preference

`internal/buildinfo` also exposes `FleetCapability`. `CurrentFleetCapability` reads the link-time marker and falls back to `preview` for an absent, unknown or malformed value; there is no environment, config, CLI or key override. `TestFleetCapabilityDefaultsToPreviewAndFailsSafe` and `TestReleaseConfigKeepsFleetPreviewOnly` prove the shipped default. That is why the TUI's live fleet controls are described as build-gated, not as a user setting.

Related: [`architecture/workflows`](workflows.md) for what these interfaces do over time, and the route and key sources under `experiments/tui/app_screens.go`.
