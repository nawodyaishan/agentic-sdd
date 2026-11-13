# Execution runs — task briefs, check evidence and review packages

**Status:** the approved contract for [Feature 016](../specs/016-agent-execution-context-evidence-recovery/spec.md), whose combined approval is recorded once in [spec §14](../specs/016-agent-execution-context-evidence-recovery/spec.md#14-approval). Implementation batches B1–B3 are complete and awaiting human review. [Feature 017](../specs/017-agent-context-upgrade/spec.md) adds run-bound brief provenance and conservative reuse of schema-v1 runs without changing the other command families; its B1 implementation awaits human review. This page describes the frozen public contract: it is not a shipped-capability claim, a security certification or an approval.

The `run`, `task` and `review` command families give the main agent one private local place for a task brief, truthful check evidence, a review package and a recoverable checkpoint. They are ordinary CLI commands: offline, explicit arguments, no prompt, no model call, no network call, no telemetry, and no change to an existing command's output or exit codes.

## What a run is — and is not

A run is bound to an explicit repository root, a feature and batch identity (or a `--direct` label), a unique collision-resistant run ID, and observed Git identity. It records observations only:

- a run is **not** an approval, a batch ledger or a completion claim; `spec.md` §Approval and `tasks.md` stay authoritative;
- a run is **not** a sandbox — it inherits the invoker's permissions and every side effect of the command it runs;
- a run is **not** a second copy of the specification; it stores a source reference and digest, so a changed task makes the checkpoint stale instead of silently diverging.

`run init` is a local scratch write. It is never permission to implement, to start another batch, to call a paid model or to dispatch a reviewer.

## Commands

```sh
# Create a private task-run workspace; no model calls, no user-home changes.
agentic-sdd run init --feature DIR --batch ID [--repo DIR] [--run-root DIR] [--format text|json]
agentic-sdd run init --direct [--label TEXT] [--repo DIR] [--run-root DIR] [--format text|json]

# Extract exactly one authorized task into the run.
agentic-sdd task brief --run RUN_ID --task T01 [--force] [--repo DIR] [--run-root DIR] [--format text|json]

# Execute one already-authorized local check; preserve complete stdout/stderr.
agentic-sdd run check --run RUN_ID [--task T01] [--timeout DURATION] [--max-output-bytes N] [--repo DIR] [--run-root DIR] [--format text|json] -- COMMAND [ARG...]

# Package actual relevant Git changes and provenance for a bounded review.
agentic-sdd review package --run RUN_ID --base COMMIT [--head COMMIT] [--max-file-bytes N] [--max-total-bytes N] [--repo DIR] [--run-root DIR] [--format text|json]

# Read observed run status; never claim approval or completion.
agentic-sdd run status --run RUN_ID [--repo DIR] [--run-root DIR] [--format text|json]

# Attach a caller-supplied result as a CLAIM, not verified truth.
agentic-sdd run record --run RUN_ID [--task T01] --result FILE|- [--repo DIR] [--run-root DIR] [--format text|json]

# Preview or remove precisely this run's scratch files.
agentic-sdd run clean --run RUN_ID [--yes] [--repo DIR] [--run-root DIR] [--format text|json]

agentic-sdd help run|task|review
```

Defaults and limits fixed by the plan:

| Flag | Default and range |
|---|---|
| `--format` | `text`; `json` adds a versioned object with `schemaVersion: 1` and a `kind` value, exact artifact paths and `unknown` where a value is not observable |
| `--timeout` (`run check`) | `15m`; `0` disables the wall-clock limit |
| `--max-output-bytes` (`run check`) | `64 MiB` per stream; `1 KiB`–`1 GiB` |
| `--max-file-bytes` (`review package`) | `2 MiB` per file |
| `--max-total-bytes` (`review package`) | `32 MiB` total |

`run check` requires the `--` separator and a non-empty command vector. There is no implicit shell, glob expansion or variable interpolation; invoke `sh -c '...'` explicitly if that is the intended command. The command runs in the repository root with the invoker's permissions. `run clean` without `--yes` is a preview; `--yes` acknowledges deletion of that run's scratch only, never SDD or batch authorization.

`--repo` defaults to the current working tree. `--run-root` selects an explicit private root, including when the repository is not writable; the resolved path and whether it is Git-visible are reported, never asserted. A feature run stores the `spec.md`/`tasks.md` paths and digests plus the selected batch's normalized scope digest; a direct run stores a label and fabricates no feature files.

## Artifact layout

Default root: `<repo>/.agentic-sdd/runs`, with a generated `runs/.gitignore` containing `*` so the whole run tree stays invisible to Git without editing the tracked root `.gitignore`. An existing ignore file is never rewritten. If run scratch cannot be kept invisible and no explicit `--run-root` is given, `run init` fails rather than leaving visible files.

```text
<run-root>/<run-id>/
  run.json                 immutable identity, 0600
  checkpoint.json          mutable observed state, atomic, 0600
  .lock                    per-run writer lock, 0600
  briefs/<task>.md         generated task brief, 0600
  checks/<check-id>/
    meta.json              argv digest, paths, timestamps, exit/signal, limits
    argv.json              full argv, private, 0600
    stdout.log             capped complete-so-far stdout
    stderr.log             capped complete-so-far stderr
  records/<record-id>.json untrusted claim, 0600
  review/<package-id>/
    review.json            manifest
    review.patch           committed/staged/unstaged sections
    untracked/<path>       verbatim copies or symlink metadata
```

Run, check, record and package IDs are path-safe and deterministic in shape (`run-20261113T114500Z-a1b2c3`, `chk-…`, `rec-…`, `pkg-…`). Directories are `0700` and files `0600`; checkpoint writes are atomic, so an interrupted write leaves the previous valid file. One writer at a time per run: concurrent checks or run-backed brief mutations on one run fail on the lock instead of interleaving, and a run-ID collision is rejected rather than reused.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Command completed successfully. For `run status`, the observation was read successfully even when the reported state is `stale` or `blocked` — inspect the state field. For `run check`, the supplied command passed. |
| `1` | Runtime, I/O, interrupted or failed check. For `run check`, the command failed, was interrupted or exceeded a limit; its own exit code and signal are in the result, not the wrapper's exit code. |
| `2` | Usage, invalid arguments, ambiguous or missing task, unsafe path, an unresolved required reference, or a conflicting run/checkpoint identity. Text on stderr; no partial JSON object. |
| `3` | `review package` wrote a partial package. The artifact and omissions are real, and the package must not be called complete. |
| `124` | `run check` timed out. |

A failed, timed-out or interrupted `run check` and a partial `review package` still emit their versioned result object, so CI can consume the evidence. Usage and validation errors do not.

## Observed state, stale runs and reconciliation

`run status` reports an observation, never a verdict. The states are `prepared`, `running`, `observed-partial`, `observed-failed`, `awaiting-verification`, `verified-evidence`, `stale` and `blocked`; precedence is `blocked`, `running`, `stale`, `observed-failed`, `observed-partial`, `awaiting-verification`, `verified-evidence`, `prepared`, and the run-level state is the highest-severity task state.

- `verified-evidence` means a passing check matches the current source and worktree with no contradicting claim or blocker. It is still not human approval.
- An interrupted `run check` can leave a checkpoint record with `result: running`; `run status` reports `observed-partial` with `interrupted_check`, never a pass.
- A `run record` entry is a **claim**. A `passed` claim yields `awaiting-verification`; only a fresh check in the current code state, or the parent's own inspection, supports a verified observation.
- Freshness reasons are explicit: `spec_changed`, `tasks_changed`, `batch_changed`, `head_changed`, `worktree_changed`, `scope_mismatch`, `missing_artifact`, `brief_digest_mismatch`, `unregistered_brief`, `brief_inspection_failed`, `interrupted_check`, `stale_lock`, `contradictory_evidence`. A checkbox or status-line-only edit is reported as `progress_updated` and does not stale the run.
- `run status` also reports a next permitted investigation — `inspect_changed_paths`, `rerun_authorized_check`, `reconcile_source_change`, `resolve_blocker` or `none`. It never prints “resume”, “start next batch” or “approved”.

After compaction, restart or interruption, read `run status` before acting. A `stale` or `blocked` run stays stopped until the changed source, batch or worktree is reconciled; reconcile, re-observe, then decide with the human. No writer, check, reviewer or next batch resumes automatically because a checkpoint looks nearly done.

## Task briefs: provenance and recovery

A run-backed brief belongs to exactly one run. `task brief` requires that run's validated identity, and a `checkpoint.json` naming a different — even syntactically valid — run ID is a conflict: `run status` and every checkpoint-consuming mutation exit `2` with no status JSON and no changed run content. A foreign checkpoint is preserved for investigation, never presented as this run's normal observation. Direct-scope runs have no task brief.

Creating, replacing or re-registering a run-backed brief is one mutation under the per-run writer lock. On success the checkpoint records the task, the path of the contained `briefs/<task>.md` artifact and its digest, preserving existing task state and check, claim and blocker history. A persisted brief never verifies evidence, clears a failure or becomes approval; a failed checkpoint write is reported as an error rather than success.

`run status` compares every registered brief with the file on disk. A missing file reports `missing_artifact`; different bytes report `brief_digest_mismatch`; a regular file under `briefs/` with no registered provenance reports `unregistered_brief`. An unreadable, symlinked or over-budget `briefs/` directory reports `brief_inspection_failed` instead of a clean observation. All of these are blocking stale reasons with `reconcile_source_change` as the next permitted investigation.

Schema-v1 runs stay readable without migration. A run with no brief artifacts is not invalid for predating provenance. A run holding an unregistered brief is `stale` until someone reconciles it: status never adopts, rewrites or deletes the file. An explicit `task brief` reissue may register that artifact only when the rendered content is byte-identical; different bytes still require `--force`, and `--force` never bypasses a changed source scope.

A brief, check log or record is an observation about local state. It is not a sandbox, not cryptographic authenticity or tamper-proof storage, and it never repairs a damaged run automatically; reconciliation and any replacement stay explicit, human-authorized actions.

## Review packages and partial reasons

`review package` records the actual selected scope: committed changes for `BASE..HEAD`, plus staged, unstaged and untracked changes when `--head` is the current HEAD. It never treats `BASE..HEAD` as the whole dirty worktree. Git inspection is read-only — no stage, commit, reset, checkout or clean — and external diff drivers and textconv are disabled.

`review.json` reports `complete` (content coverage, not evidence freshness), `partial_reasons`, per-file entries, omissions and counts. A written but incomplete package exits `3` and names every reason. Examples of honest partiality:

- `head_not_current` — `--head` differs from the current HEAD, so staged/unstaged/untracked sections are omitted;
- `index_conflict` — a conflicted index, so the index section is omitted;
- `base_not_ancestor` — the base is not an ancestor of the head; the diff is still produced but the relationship is called out;
- `sensitive_path` — a default sensitive path (`.env`, `*.pem`, `*.key`, `id_rsa`, `credentials*`, `secrets*`, and similar) is omitted and listed;
- `max_file_bytes_exceeded` / `max_total_bytes_exceeded` — a file or package over the configured caps is omitted with its size, digest and reason;
- `unreadable_path` — an unreadable or missing path is recorded as omitted with the exact error.

Tracked binary changes keep their binary patch and are flagged `binary: true`. Untracked files are copied up to the caps; untracked symlinks are recorded as link metadata and never followed. Binary, renamed, deleted, mode-changed and submodule-pointer entries carry explicit statuses. A package whose run evidence is stale records that freshness warning; it never becomes approval or a verified task. Preparing a package does not dispatch a reviewer.

## Privacy and safety limits

- No network, model, MCP, telemetry or credential access in the execution service. It works completely offline.
- No environment dump. Command arguments stay private run metadata by default: summaries show the program basename, argument count and an argv digest, not raw arguments, so a token embedded in a command is not echoed.
- Review packaging screens sensitive paths by pattern. That is omission, not redaction, and it is not a guarantee that a package is safe to share; the manifest records what was left out.
- Run scratch is private but not encrypted. Treat `.agentic-sdd/runs/` as local, disposable project data: keep it out of Git (it self-ignores), out of backups and out of any upload, and remove it with `run clean` when it is no longer needed.
- `run clean` removes only the owned run directory, refuses a live lock, never follows a symlink and treats an already-absent run as success. There is no “clean all” and no background purging.
- Permission and path checks reduce mistakes between cooperative local tools; they are not protection against another process running as the same user.

## Boundaries: no sandbox, no approval, no model calls

- **Not a sandbox.** `run check` runs a command the invoker supplied, with the invoker's permissions and side effects. Tests can mutate files, reach the network or cost money; they need the same authorization as running them directly. A preview or dry run never executes anything.
- **Not an approval.** A run, a checkpoint, a passing check or a review package is evidence about a local observation. Human approval lives in `spec.md` §Approval; batch state and continuation live in `tasks.md`. Nothing here authorizes another batch, a merge, a publish or a release.
- **Not a reviewer dispatch.** A fresh per-task reviewer exists only when the human explicitly asks for one, and is refused when isolation or permissions cannot be verified. Feature 006 `IR-06-01` remains open; a private run workspace is not reviewer isolation.
- **No hidden behaviour.** No prompt, model request, authentication check, network call, auto-install or auto-resume. Existing commands, flags, exit codes and traces are unchanged.

## Related

- [Feature 016 specification](../specs/016-agent-execution-context-evidence-recovery/spec.md) — requirements, acceptance criteria and the approval record.
- [Plan](../specs/016-agent-execution-context-evidence-recovery/plan.md) — workspace, parser, check, package and state-machine design.
- [Tasks](../specs/016-agent-execution-context-evidence-recovery/tasks.md) — batch ledger and the current authorization.
- `agentic-sdd help run`, `agentic-sdd help task`, `agentic-sdd help review` — the same contract, one command family at a time.
