---
title: Execution run reference
description: Field contracts for private run scratch, task briefs, check observations, caller records, and review packages, with states, limits, and boundaries.
---

Source-derived execution facts. A run or check is observation, never approval, verification or a sandbox.

For the step-by-step workflow, follow [Use execution runs for one task](../how-to/execution.md); this page lists the exact field, state, and limit contracts that workflow relies on.

<!-- generated:reference/execution:start -->
Source-derived execution-run facts for the current checkout. A run, checkpoint, check or record is an observation: it is never approval, batch state, verification or a sandbox.

## Schema

All emitted records use `schemaVersion` 1.

| Record kind | Value |
| --- | --- |
| Run identity | `execution.run.identity` |
| Task brief | `execution.task.brief` |
| Review package | `execution.review.package` |

## Workspace

The default root is `<repo>/.agentic-sdd/runs`; `--run-root` selects another private root. A generated `.gitignore` containing `*` keeps the run tree out of Git without editing the tracked root ignore file.

| Artifact | Path |
| --- | --- |
| Run identity | `run.json` |
| Checkpoint | `checkpoint.json` |
| Writer lock (kernel guard, under the run root) | `.locks/<run-id>` |
| Writer lock (legacy heartbeat) | `.lock` |
| Task brief | `briefs/<task>.md` |
| Check directory | `checks/<check-id>/` |
| Check metadata | `meta.json`, `argv.json`, `stdout.log`, `stderr.log` |
| Caller record | `records/<rec-id>.json` |
| Review package | `review/<pkg-id>/` |
| Review manifest | `review.json` |
| Review patch | `review.patch` |

Run IDs match `^run-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{6}$`. A newly created run root and each run directory use mode 0700 (an existing run root keeps its current mode), identity and checkpoint files use mode 0600, and those two writes are atomic.

## Observed state

- Checkpoint states: `prepared`, `running`, `observed-partial`, `observed-failed`, `awaiting-verification`, `verified-evidence`, `stale`, `blocked`
- Caller claim results: `passed`, `failed`, `partial`, `blocked`, `needs_input`
- Observed check results: `running`, `passed`, `failed`, `timed_out`, `interrupted`, `output_limit`, `not_started`
- Scope kinds: `feature`, `direct`

Precedence, severity and reconciliation belong to `run status`; a reported state is an observation even when it is stale or blocked.

## Limits

| Limit | Default |
| --- | --- |
| tasks.md read | 4194304 bytes (4 MiB) |
| spec.md read | 8388608 bytes (8 MiB) |
| Caller record | 1048576 bytes (1 MiB) |
| run check wall clock | 900 seconds (15 minutes) |
| Per-stream stdout/stderr | 67108864 bytes (64 MiB) |
| Review package per file | 2097152 bytes (2 MiB) |
| Review package total | 33554432 bytes (32 MiB) |
| Legacy lock heartbeat window | 1800 seconds (30 minutes); age alone never reclaims a lock |

## CLI

Common flags are `--repo`, `--run-root`, `--format`; `--format` defaults to `text` and accepts text or json. The command families are `run init|status|record|clean|check`, `task brief` and `review package`.

Documented exit codes are `0`, `1`, `2`, `3`, `124`. `124` is the check-timeout status, and a partial review package uses `3` while still emitting its result object.

## Boundary

The service observes local state and writes only private run artifacts. It does not approve work, call a model, or act as a sandbox; the repository, batch authorization and human review remain authoritative.
<!-- generated:reference/execution:end -->

## Curated notes on the generated facts

These hand-written notes add the operational meaning the generated fields do not carry.

- **Modes.** The run directory is created `0700` by the service, and identity, checkpoint, lock, record, brief and check files are `0600`. The run root is created `0700` only when it is newly created; an existing root keeps its own mode, including a user-supplied `--run-root`.
- **Check results.** `running` is the in-progress marker written when a check starts; the final outcome replaces it. A record that still reads `running` means no final outcome was written, and `run status` reports it as `observed-partial` with `interrupted_check`, never as a pass.
- **Lock conflicts.** Per-run kernel guards live under `<run-root>/.locks/<run-id>`, alongside the legacy `<run-id>/.lock` heartbeat. A lock conflict exits `2`. Age never reclaims a lock: a legacy `.lock` is reclaimed only when its recorded PID is dead on this host, and a malformed, unreadable or unknown legacy `.lock` blocks. `run clean --yes` takes the lock before deleting.
- **Output and environment.** A check's checkpoint record and printed result carry a bounded summary of its output (up to 40 failure-marker lines from stdout and stderr together, or the last 20 lines when no line matches), and the summary is not redacted. The child process inherits the full environment. A check must not print secrets into a run that agents will read; see the [how-to's warning](../how-to/execution.md#3-run-one-authorized-check).
- **Limits of interruption and process-group completion.** SIGINT and SIGTERM terminate the child's process group after a bounded grace period and record `interrupted`. On Linux and Darwin, a surviving child process in the owned group makes `run check` fail (Darwin has classification only without post-reap group signaling, a documented residual); on Windows, `run check` fails closed because owned process-group guarantees are unavailable. An uncatchable `SIGKILL`, detached child, or host crash leaves the child running; a same-host dead-PID legacy `.lock` then permits cleanup while that orphan may still execute (a malformed or unknown legacy `.lock` blocks). The [architecture evidence page](../architecture/evidence.md) and the how-to describe this in full.

## Cleanup

`asdx-core run clean --run RUN_ID` previews the deletion of one private run directory, and adding `--yes` deletes only that run's scratch. It refuses a live writer lock, never follows a symlink, and treats an already-absent run as success; there is no bulk cleanup command and no background purging.
