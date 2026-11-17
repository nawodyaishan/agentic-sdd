---
title: Architecture evidence
description: Run persistence, per-run locks, freshness, check evidence and review-package recovery.
---

The `run`, `task` and `review` families exist so an agent can leave a truthful local trace instead of a persuasive summary. Everything they write is private scratch under a run root. None of it is approval, a batch ledger, a sandbox or proof of authenticity.

`internal/execution` implements this; `cmd/agentic-sdd/execution_cli.go` and `run.go`/`task.go`/`review.go` add parsing and exit codes. [`reference/execution`](../reference/execution.md) has the schema tables.

## Where a run lives

`ResolveRunRoot` defaults to `<repo>/.agentic-sdd/runs` and accepts `--run-root` for another private root. `InitRun` then:

- writes a self-ignoring `.gitignore` containing `*` for the default root, so `git status` stays clean without editing the tracked root ignore file; an explicit `--run-root` must already be ignored, or the run records that the workspace is visible to Git;
- creates the run directory and its subdirectories `0700`, files `0600`;
- refuses a symlinked scratch parent or run root, an unborn HEAD, or a scope root above the repository;
- writes `run.json` atomically (`writeFileAtomic`).

`run.json` (`RunIdentity`) is immutable. It records the run ID shape `run-YYYYMMDDTHHMMSSZ-xxxxxx`, the scope (feature directory, spec/tasks digests, batch ID and batch scope digest), the baseline Git observation, an optional approval *reference*, and whether the workspace is visible to Git.

`checkpoint.json` (`Checkpoint`) is mutable and explicitly disposable: observed state, task states, checks, claims, blockers and review packages. `WriteCheckpoint` is atomic, so an interrupted write leaves the previous valid document.

### Diagram D05 — MCP and execution are separate stores

![Diagram D05: two flows with a dotted connector labelled "separate stores" between them. The top MCP flow runs plan, lock, apply and verify, with a rollback branch on verification failure. The bottom execution flow runs run init, task brief, run check, run record, run status, review package and run clean in that order.](/diagrams/d05-evidence.png)

*Figure D05 — MCP and execution flows. Source: `assets/diagrams/D05-evidence.mmd`; the PNG is its rendered projection.*

- The top flow is the MCP transaction: `MCPPlan`, `acquireMCPLock`, `ApplyMCPPlan`, `mcpVerifySnapshotEntries`, with `mcpApplyRollback` on failure. Its lock and snapshot and intent records live under `~/.agentic-sdd/mcp/` and its target is `~/.config/opencode/opencode.json`; the figure does not draw those paths.
- The bottom flow is one execution run: `InitRun`, `registerTaskBrief`, `RunCheck` (the child command, capped logs and the checkpoint), `IngestRecord`, `StatusRun` with `evaluateFreshness`, `BuildReviewPackage` and `CleanRun`. Its files are `<run-root>/.locks/<run-id>` (kernel guard), `<run>/.lock` (legacy heartbeat), `run.json`, `checkpoint.json` and the `briefs/`, `checks/`, `records/` and `review/` directories under the run root; the figure does not draw those paths.
- The dotted connector is a boundary marker, not a data path. The two flows share no store: the MCP engine never reads a checkpoint, and the execution package never opens a client config or writes outside a run root. A check command it runs is arbitrary and executes with the invoker's own permissions.
- Both are opt-in: MCP writes only through `appops.MCPApply`/`MCPRestore` after a digest check, and execution writes only when a `run`-family command is invoked.

## Locks and interruption state

Each run has one writer at a time. `acquireRunLock` acquires a nonblocking kernel guard under `<run-root>/.locks/<run-id>` (outside the directory `clean` removes) and creates the legacy `<run>/.lock` with `O_CREATE|O_EXCL`, token, PID, hostname, start time, operation and a heartbeat timestamp refreshed every five minutes for older binaries. Age never reclaims a lock: a legacy `.lock` is reclaimed only when its recorded PID is dead on this host (after a file-identity recheck), and a malformed, unreadable or unknown-owner legacy `.lock` blocks as a conflict. `run clean --yes` acquires the same lock before deleting, and its preview stays read-only. `run check` turns SIGINT and SIGTERM into a child-process-group termination, a bounded grace period, an `interrupted` record and a released lock; it exits `1`, adding no exit-code class. On Linux and Darwin a surviving child process in the owned group makes `run check` fail (Darwin has classification only, a documented residual); on Windows `run check` fails closed.

| Recorded state | Lock observation | Effective state | Reason |
|---|---|---|---|
| `running` | absent | `observed-partial` | `interrupted_check` |
| `running` | present but stale | `observed-partial` | `interrupted_check`, `stale_lock` |
| any | — | `blocked` wins over everything | blocker recorded |
| any with source/identity/artifact drift | — | `stale` | e.g. `spec_changed`, `head_changed`, `missing_artifact` |

`effectiveState` applies that precedence, and `HighestState` orders `blocked`, `running`, `stale`, `observed-failed`, `observed-partial`, `awaiting-verification`, `verified-evidence`, `prepared`. `TestLockConflictAndRelease`, `TestLockStaleReclaim`, `TestLockHeartbeatPreventsAgeReclaim`, `TestCheckInterruptKillsProcessGroup`, `TestCleanApplyReclaimsStaleLockAndRemovesRun` and `TestStatusInterruptedCheck` pin the lock and interruption behavior. `run clean` previews or removes exactly one owned run, takes the lock for deletion and refuses a live lock, an unowned or symlinked run, or a moved run (`TestCleanRefusesLiveLock`, `TestCleanRefusesUnownedAndSymlinkedRuns`, `TestCleanRefusesMovedRun`).

**Limits after the corrective.** A live writer is protected by the kernel guard under `.locks/` and the legacy heartbeat, `run clean --yes` takes the lock, and SIGINT/SIGTERM terminate the child's process group and leave an `interrupted` record. On Linux and Darwin, a surviving child process in the owned group makes `run check` fail (`failed`, exit `1`; Darwin is classification-only without post-reap group signaling, a documented residual), and on Windows `run check` fails closed. The remaining gap is a signal the wrapper cannot catch: after `SIGKILL`, a detached/escaped child, or a host crash the child's process group may keep running, while the kernel guard is released by the OS and a same-host dead-PID legacy `.lock` is reclaimable at once (a malformed or unknown-owner legacy `.lock` blocks). `run clean` can then remove the run directory while that orphan still executes, and the check record stays `running`; `run status` reports `observed-partial` with `interrupted_check`. These are not sandbox or approval properties; they are local state limits to observe with `run status` before acting.

## Freshness is computed, not remembered

`run status` never trusts the checkpoint alone. `evaluateFreshness` compares the recorded identity with current state and re-observes the worktree through `ObserveRun`, which hashes HEAD, branch, status, diff, untracked content (bounded) and a worktree digest while excluding the run directory. It never mutates the checkpoint.

Reasons are explicit: `spec_changed`, `tasks_changed`, `batch_changed`, `head_changed`, `worktree_changed`, `scope_mismatch`, `missing_artifact`, `brief_digest_mismatch`, `unregistered_brief`, `brief_inspection_failed`, `interrupted_check`, `stale_lock`, `contradictory_evidence`. `progress_updated` is the only informational reason. It is reported when `tasks.md` no longer matches its recorded digest but the selected batch's normalized scope digest still matches: checkbox and status-line edits, and any edit outside the selected batch. `tasks_changed` is only reported together with `batch_changed`, and that pair stales the run.

The same evaluation also names one permitted next investigation: `inspect_changed_paths`, `rerun_authorized_check`, `reconcile_source_change`, `resolve_blocker` or `none`. It never says "resume", "approved" or "start next batch". `TestStatusSourceAndWorktreeFreshness`, `TestStatusMissingArtifact`, `TestStatusScopeMismatch`, `TestStatusContradictoryEvidence` and `TestStatusStateMachineTable` cover the matrix.

A foreign or mismatched checkpoint is a conflict, not data to adopt: a `checkpoint.json` naming a different run ID exits `2` and is preserved for investigation. Schema-v1 runs stay readable; a run with no brief artifacts is not invalid.

## Check evidence

`RunCheck` runs one explicit argv vector with no shell, no glob expansion and no variable interpolation, in the repository root recorded by the run identity, with the invoker's own operating-system permissions. It is not a sandbox and does not drop privileges.

- The check starts under the run lock and writes `checks/<check-id>/meta.json`, `argv.json`, `stdout.log` and `stderr.log`.
- Output is capped per stream; a capped result is `output_limit` with truncated logs, not a silent pass.
- Outcomes are `passed`, `failed`, `timed_out`, `interrupted`, `output_limit`, `not_started`; the in-progress marker `running` survives only when no final outcome could be written.
- Exit code, signal, timing and bounded summary lines are recorded. `argv_digest` and `argv_path` live in the checkpoint metadata; the full argv stays in the private run directory.
- A worktree change during the check is recorded as `worktree_changed_during_check`.
- The checkpoint and the printed result carry a bounded output summary: up to 40 failure-marker lines from stdout and stderr together, or the last 20 lines when no line matches. The child inherits the full environment, and the summary is not redacted. A check that prints secrets puts them into the run; see the [how-to's secret warning](../how-to/execution.md#3-run-one-authorized-check).
- Public exit codes: `0` passed (no surviving owned-group children), `124` timed out, `1` failed/interrupted/limited (including surviving owned-group children on Linux/Darwin and fail-closed refusal on Windows), `2` usage or validation.

`TestCheckPassReachesVerifiedEvidence`, `TestCheckArtifactsAndPrivateProvenance`, `TestCheckTimeout`, `TestCheckSignalIsInterrupted` and `TestCheckOutputLimitTerminatesAndTruncates` prove those outcomes.

## Claims are not evidence

`run record` stores a caller-supplied claim (`passed`, `failed`, `partial`, `blocked`, `needs_input`) under `records/`. A `passed` claim produces `awaiting-verification`; it never produces `verified-evidence`. Only a passing check observed against the current source and worktree, with no contradicting claim or blocker, reaches `verified-evidence` — and even that is not human approval. `TestRecordIngestPassedRemainsAwaitingVerification` and `TestStatusNeverUsesApprovalLanguage` protect the wording and the state.

## Review packages and recovery

`BuildReviewPackage` records the scope it actually read: committed `BASE..HEAD`, plus staged, unstaged and untracked changes when `--head` is the current HEAD. Git inspection is read-only; no stage, commit, checkout, reset or clean is issued, and external diff drivers are disabled (`TestReview_NoMutation`). It acquires the run lock, then writes:

- the bounded `untracked/` tree first (regular-file content and symlink metadata);
- `review.patch` — committed, staged and unstaged sections with honest omission markers;
- `review.json` — manifest with base, head, branch, counts, per-file digests, omissions, partial reasons, and the run state plus stale reasons observed at build time;
- an appended `ReviewPackageRecord` in the checkpoint, written last.

Recovery guidance follows from that order:

- A written but incomplete package exits `3` (`ErrPartial`) and names every reason: `head_not_current`, `index_conflict`, `base_not_ancestor`, `sensitive_path`, `max_file_bytes_exceeded`, `max_total_bytes_exceeded`, `unreadable_path`. `complete` describes content coverage, not evidence freshness.
- Every artifact is written with `writeFileAtomic`, so an interrupted build leaves either a previous complete file or no file, never a half-written one.
- A crash after the package files but before the checkpoint update leaves an unregistered directory. The checkpoint stays authoritative; the recovery is to re-observe with `run status` and rebuild the package, not to hand-edit the checkpoint. `TestReview_CheckpointIntegration` covers the normal append path.
- Sensitive paths (`.env`, `*.pem`, keys, credentials, secrets, and similar) are omitted and listed rather than copied. That screening is a convenience, not a guarantee: a package can still contain sensitive content from a path that does not match the default patterns.

## Inherited command permissions and same-user trust

Every artifact here is written by the user who ran the command, with that user's permissions. There is no privilege separation, no daemon, no signature and no tamper-proof store. `review.json` digests detect drift; they do not authenticate a reviewer or prove a command ran. A second process running as the same user can rewrite any of it, and `run clean` will remove only a run it can prove it owns.

That is also why run artifacts are private scratch: keep `.agentic-sdd/runs` out of Git, backups and uploads, and read `run status` before acting after an interruption rather than assuming a nearly complete checkpoint means the work is complete.

Related: [`architecture/workflows`](workflows.md) for install-time transactions, and [`reference/execution`](../reference/execution.md) for field-level schema.
