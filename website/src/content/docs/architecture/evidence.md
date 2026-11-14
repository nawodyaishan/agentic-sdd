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

D05's static projection is owned by the diagram lane and is not rendered here; its text alternative:

- On the left, the MCP transaction: lock at `~/.agentic-sdd/mcp/.lock`, snapshot and intent records under `~/.agentic-sdd/mcp/backups`, target `~/.config/opencode/opencode.json`.
- On the right, an execution run: lock at `<run>/.lock`, `run.json`, `checkpoint.json`, `briefs/`, `checks/`, `records/`, `review/` under `.agentic-sdd/runs`.
- There is no arrow between them. The MCP engine never reads a checkpoint. The execution package itself never opens a client config or writes outside a run root, though a check command it runs is arbitrary and executes with the invoker's own permissions.
- Both are opt-in: MCP writes only through `appops.MCPApply`/`MCPRestore` after a digest check, and execution writes only when a `run`-family command is invoked.

## Locks and interruption state

Each run has one writer at a time. `acquireRunLock` creates `.lock` with `O_CREATE|O_EXCL`, records token, PID, hostname, start time and operation, and releases by comparing its own token. A stale lock is reclaimed only when its PID is dead on this host or it is older than `LockStaleAfter` (30 minutes); any other existing lock is a conflict.

| Recorded state | Lock observation | Effective state | Reason |
|---|---|---|---|
| `running` | absent | `observed-partial` | `interrupted_check` |
| `running` | present but stale | `observed-partial` | `interrupted_check`, `stale_lock` |
| any | — | `blocked` wins over everything | blocker recorded |
| any with source/identity/artifact drift | — | `stale` | e.g. `spec_changed`, `head_changed`, `missing_artifact` |

`effectiveState` applies that precedence, and `HighestState` orders `blocked`, `running`, `stale`, `observed-failed`, `observed-partial`, `awaiting-verification`, `verified-evidence`, `prepared`. `TestLockConflictAndRelease`, `TestLockStaleReclaim` and `TestStatusInterruptedCheck` pin the lock and interruption behavior. `run clean` previews or removes exactly one owned run and refuses a live lock, an unowned or symlinked run, or a moved run (`TestCleanRefusesLiveLock`, `TestCleanRefusesUnownedAndSymlinkedRuns`, `TestCleanRefusesMovedRun`).

## Freshness is computed, not remembered

`run status` never trusts the checkpoint alone. `evaluateFreshness` compares the recorded identity with current state and re-observes the worktree through `ObserveRun`, which hashes HEAD, branch, status, diff, untracked content (bounded) and a worktree digest while excluding the run directory. It never mutates the checkpoint.

Reasons are explicit: `spec_changed`, `tasks_changed`, `batch_changed`, `head_changed`, `worktree_changed`, `scope_mismatch`, `missing_artifact`, `brief_digest_mismatch`, `unregistered_brief`, `brief_inspection_failed`, `interrupted_check`, `stale_lock`, `contradictory_evidence`. `progress_updated` is the only informational reason; a checkbox-only edit does not stale a run.

The same evaluation also names one permitted next investigation: `inspect_changed_paths`, `rerun_authorized_check`, `reconcile_source_change`, `resolve_blocker` or `none`. It never says "resume", "approved" or "start next batch". `TestStatusSourceAndWorktreeFreshness`, `TestStatusMissingArtifact`, `TestStatusScopeMismatch`, `TestStatusContradictoryEvidence` and `TestStatusStateMachineTable` cover the matrix.

A foreign or mismatched checkpoint is a conflict, not data to adopt: a `checkpoint.json` naming a different run ID exits `2` and is preserved for investigation. Schema-v1 runs stay readable; a run with no brief artifacts is not invalid.

## Check evidence

`RunCheck` runs one explicit argv vector with no shell, no glob expansion and no variable interpolation, in the repository root recorded by the run identity, with the invoker's own operating-system permissions. It is not a sandbox and does not drop privileges.

- The check starts under the run lock and writes `checks/<check-id>/meta.json`, `argv.json`, `stdout.log` and `stderr.log`.
- Output is capped per stream; a capped result is `output_limit` with truncated logs, not a silent pass.
- Outcomes are `passed`, `failed`, `timed_out`, `interrupted`, `output_limit`, `not_started`; the in-progress marker `running` survives only when no final outcome could be written.
- Exit code, signal, timing and bounded summary lines are recorded. `argv_digest` and `argv_path` live in the checkpoint metadata; the full argv stays in the private run directory.
- A worktree change during the check is recorded as `worktree_changed_during_check`.
- Public exit codes: `0` passed, `124` timed out, `1` failed/interrupted/limited, `2` usage or validation.

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
