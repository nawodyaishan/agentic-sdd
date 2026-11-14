---
title: Use execution runs for one task
description: Create private run scratch, extract an exact task brief, capture check evidence and caller claims, assemble a review package, and recover from stale state.
---

`asdx-core run`, `asdx-core task` and `asdx-core review` keep one task's assignment, local check evidence, caller-reported claims and Git change package in a private run directory. The families are ordinary offline CLI commands: explicit arguments, no prompt, no model call, no network call and no telemetry. Their implementation is `internal/execution` with thin command adapters in `cmd/agentic-sdd/run.go`, `cmd/agentic-sdd/task.go` and `cmd/agentic-sdd/review.go`.

Run scratch is private by default: `<repo>/.agentic-sdd/runs/<run-id>/`. The generated `runs/.gitignore` keeps the tree out of Git without editing the tracked root ignore file. Keep it out of backups and uploads too, and remove it with `run clean` when it is no longer needed.

> **Observation, not authority.** A run, checkpoint, check log or review package is an observation of local state. It is never approval, batch state, verification, reviewer isolation or a sandbox. `spec.md` §Approval and `tasks.md` remain authoritative, and every command you run keeps your own permissions and side effects.

## When this workflow helps

Use a run when one task needs a durable local trail that survives a restart or context compaction:

- the exact task section and its requirement references, extracted without editing `tasks.md`;
- what a local command actually printed and how it exited;
- what an agent or human *reported*, kept distinct from what was observed;
- the actual Git change set, with explicit omissions, for a bounded review.

Synthetic, inert shape examples for the run identity, status, checkpoint, check metadata, caller claim, task brief and partial review package live under `website/tests/examples/execution/`. They are hand-written, no command was executed for them, and they are not captured evidence.

A feature run is bound to a feature directory, `spec.md`, `tasks.md` and one selected batch ID. A direct-scope run is bound to an optional label and is useful when no feature documents exist. Both are references to existing work: creating a run authorizes nothing.

## Before you start

- Work in the repository checkout that owns the files you will inspect. Git identity is read from that checkout.
- The canonical command is `asdx-core`; `agentic-sdd` forwards to it during the deprecation window. `asdx-core help run`, `help task` and `help review` print the same contract as the command families below.
- A feature run requires the feature's `spec.md` and `tasks.md` to exist and the selected batch heading to be unambiguous. Missing or duplicate batches fail closed.
- None of the execution commands themselves installs skills, changes your user home, calls a provider or contacts a network endpoint. A command you hand to `run check` can still have its own effects.

## 1. Create the run

For feature-backed work, name the feature directory and the already-authorized batch. Replace the example path, batch and task identifiers with the ones you are authorized to use:

```sh
asdx-core run init \
  --feature specs/016-agent-execution-context-evidence-recovery \
  --batch B3
```

For direct-scope work, omit the feature and batch and add an optional label:

```sh
asdx-core run init --direct --label "typo fix in guide"
```

Illustrative output shape (not captured evidence):

```text
run: run-20261113T114500Z-a1b2c3
scope: batch B3 (specs/016-agent-execution-context-evidence-recovery)
state: prepared (observation only; not approval)
run dir: /path/to/repo/.agentic-sdd/runs/run-20261113T114500Z-a1b2c3
next: asdx-core task brief --run run-20261113T114500Z-a1b2c3 --task T01
```

The run ID is the handle for every later command. `run.json` is the immutable identity: repository root, scope, source digests, Git baseline, creation time and tool version. `checkpoint.json` is the mutable observation file.

`--format json` emits the same identity with `schemaVersion: 1` and `kind: execution.run.identity`.

### Where the scratch lives

- The default root is `<repo>/.agentic-sdd/runs`. `run init` refuses the default root when it cannot prove the tree is Git-ignored or when the root contains tracked files; pass an explicit `--run-root` in that case.
- `--run-root DIR` selects another private root. When it is inside the repository but not ignored, `run init` reports the workspace as Git-visible instead of pretending otherwise; a root outside the repository is simply outside that worktree.
- Run directories are mode `0700`; identity and checkpoint files are mode `0600`. An interrupted checkpoint write leaves the previous valid file in place.

## 2. Extract the exact task brief

```sh
asdx-core task brief --run run-20261113T114500Z-a1b2c3 --task T05
```

The extractor recognizes numeric task headings such as `T05`. Lane or delegation labels are not task IDs; extract the owning numeric task instead of inventing an identifier. Missing, duplicate or ambiguous task headings fail closed.

The brief is written to `briefs/<task>.md` and registered in `checkpoint.json` with its path and SHA-256 digest. It contains:

- run, task and batch identity;
- the tasks source path, line range, full tasks digest, normalized batch scope digest and spec digest;
- the batch context and the exact task definition;
- the resolved requirement and acceptance references from `spec.md`;
- explicit authority and stop conditions.

The brief copies requirements; it does not invent them and does not grant permission beyond the task text. It is scoped assignment, not approval.

Re-issuing `task brief` registers a byte-identical brief. If the rendered bytes differ, the command fails unless you pass `--force`. `--force` replaces differing brief bytes; it never bypasses a changed spec or batch scope. Direct-scope runs have no task brief and refuse the command.

## 3. Run one authorized check

`run check` executes exactly the argument vector you supply. The `--` separator is required and there is no implicit shell, glob expansion or variable interpolation:

```sh
asdx-core run check \
  --run run-20261113T114500Z-a1b2c3 \
  --task T05 \
  -- go test ./internal/execution
```

If a shell really is the intended command, say so explicitly with `sh -c '...'`; otherwise the arguments are passed as-is.

What is observed:

- The command runs in the repository root recorded by the run identity, with the invoker's environment and permissions.
- `--timeout` defaults to 15 minutes; `0` disables the wall-clock limit.
- `--max-output-bytes` caps each of stdout and stderr at 64 MiB by default and accepts 1 KiB to 1 GiB per stream. Reaching the cap cancels the command and marks the result `output_limit` and the stream truncated.
- The full-to-cap `stdout.log` and `stderr.log` are kept under `checks/<check-id>/`. `argv.json` keeps the exact vector privately; the run-level record stores the program name, argument count and an argv digest instead of the full vector.
- Exit `0` means the supplied command passed. Exit `1` means it failed, was interrupted or exceeded a limit. Exit `124` means the wall-clock limit fired. The command's own exit code and signal, when observed, are inside the record rather than the wrapper's exit code.
- A worktree change during the check, or a missing end observation, downgrades the observation to `observed-partial`; a passing process exit alone does not justify `verified-evidence`.

A check is not a sandbox. Tests can write files, reach the network or cost money. Run only commands whose side effects are already authorized.

## 4. Attach a caller record (a claim)

`run record` ingests a caller-supplied JSON document and persists it verbatim with its digest:

```sh
asdx-core run record \
  --run run-20261113T114500Z-a1b2c3 \
  --task T05 \
  --result ./record.json
```

Use `--result -` to read from standard input. Feature-scope runs require `--task`; direct-scope runs default the task label to `direct` for both `run record` and `run check`. The parser rejects unknown fields, duplicate keys, invalid UTF-8, NUL bytes, unsafe relative paths, oversized documents and the wrong `schemaVersion`. A record is at most 1 MiB.

Illustrative synthetic record (a partial report, not a passing claim):

```json
{
  "schemaVersion": 1,
  "result": "partial",
  "summary": "Example only: one focused check observed; integration checks not run.",
  "changedPaths": ["internal/example/example.go"],
  "checks": [
    {
      "command": "go test ./internal/example",
      "exitCode": 0
    }
  ],
  "agent": {
    "role": "implementation",
    "model": "example-model-name"
  },
  "unknowns": ["Integration behavior was not exercised."]
}
```

Allowed `result` values are `passed`, `failed`, `partial`, `blocked` and `needs_input`. A non-blocked record sets the task to `awaiting-verification`; a `blocked` result or any blocker sets `blocked`. Neither outcome proves the work. The optional `agent.role` and `agent.model` are reported provenance, not verified identity. `checks` inside a record are claims about commands, not observed process evidence — only `run check` produces the latter.

Allowed blocker kinds are `permission`, `billing`, `data_policy`, `transport`, `test`, `partial_write` and `unknown`.

## 5. Assemble a review package

```sh
asdx-core review package \
  --run run-20261113T114500Z-a1b2c3 \
  --base <base-commit>
```

The package is written under `review/<package-id>/` with `review.json` and `review.patch`. Packaging reads Git objects and the worktree; it does not stage, commit, reset, checkout, clean or dispatch a reviewer.

- Committed changes cover `BASE..HEAD`.
- Staged, unstaged and untracked changes are included only when `--head` is the current HEAD. `--head` defaults to the current HEAD; a different head produces `head_not_current` and omits those sections.
- `review.json` records base, head, branch, scope, counts, per-file status and digests, omissions, `complete`, `partial_reasons`, the run state and freshness warnings.
- `complete` means content coverage for the selected range. It does not mean the run's evidence is fresh and it does not mean the change is reviewed or approved.
- A partial package is still written. The command exits `3`, `partial_reasons` names why coverage is partial, and each omitted path is listed with its reason. The partial reasons are `base_not_ancestor`, `head_not_current`, `index_conflict`, `sensitive_path`, `max_file_bytes_exceeded`, `max_total_bytes_exceeded` and `unreadable_path`.
- Sensitive-path screening (`.env`, `*.pem`, `*.key`, SSH private keys, `credentials*`, `secrets*` and similar) omits files from the package. That is omission, not redaction, and it is not a guarantee that the remaining package is safe to share.

Preparing a package is not reviewer isolation and not a review. A separate reviewer still needs an explicit human request and its own permission boundary.

## 6. Read status and recover

Start recovery by observing, not by resuming:

```sh
asdx-core run status --run run-20261113T114500Z-a1b2c3
```

`run status` reports the effective state, freshness reasons, counts, blockers, the last observation and one permitted next investigation. Its exit `0` means the observation was read successfully even when the reported state is `stale` or `blocked`; read the state field.

States are `prepared`, `running`, `observed-partial`, `observed-failed`, `awaiting-verification`, `verified-evidence`, `stale` and `blocked`. Precedence is `blocked`, `running`, `stale`, `observed-failed`, `observed-partial`, `awaiting-verification`, `verified-evidence`, `prepared`. Task states combine at the highest severity, and freshness can raise the reported state to `stale`.

`verified-evidence` means a passing check matches the current source and worktree with no contradicting claim or blocker. It is a local observation label, not human verification of the work.

| Symptom | What it means | Safe next action |
| --- | --- | --- |
| `observed-partial` with `interrupted_check` | The writer stopped before a final check outcome, or the run was marked running without a live lock. | Inspect the check logs; rerun an authorized check in the current worktree. Do not treat the partial record as a pass. |
| `stale` with `spec_changed`, `tasks_changed` or `batch_changed` | Source scope no longer matches the run's recorded digests. A checkbox-only or status-line-only edit appears as `progress_updated` and does not stale the run. | Reconcile the source with the human decision. Do not repoint or rewrite the run implicitly; start a new run only for a scope that is already authorized. |
| `stale` with `head_changed` or `worktree_changed` | Git identity or the worktree moved since the recorded observation. | Inspect the changed paths, then rerun the authorized check against the current state. |
| `stale` with `missing_artifact`, `brief_digest_mismatch`, `unregistered_brief` or `brief_inspection_failed` | A registered brief or check artifact is missing, changed or was never registered. | Inspect the artifact. Re-issuing `task brief` may register only byte-identical content; differing bytes require `--force` and a reconciled source scope. |
| Exit `2` naming a foreign checkpoint or run identity | The checkpoint belongs to a different run ID or the selected run root does not match. | Preserve it for investigation. Never edit it into agreement and never consume it as this run's state. |
| Lock conflict | Another writer owns the run, or a stale lock is not yet reclaimable. | Wait for the live writer to finish. A lock is reclaimed only when its PID is dead or it is older than 30 minutes. Repeated lock failures are a stop condition. |
| `blocked` | A claim reported a blocker or the checkpoint carries one. | Resolve the blocker with the responsible human; `run status` permits only `resolve_blocker`. |

The status output names one permitted next investigation: `inspect_changed_paths`, `rerun_authorized_check`, `reconcile_source_change`, `resolve_blocker` or `none`. It never says "resume", "approved" or "start the next batch". A stale or blocked run stays stopped until the source, batch or worktree is reconciled and a human decides what happens next.

## 7. Remove the scratch

Preview first:

```sh
asdx-core run clean --run run-20261113T114500Z-a1b2c3
```

Delete exactly that run's directory only after the preview is what you expect:

```sh
asdx-core run clean --run run-20261113T114500Z-a1b2c3 --yes
```

`run clean` removes only the owned run directory. It refuses a live writer lock, a symlink or a directory without matching ownership evidence, and it treats an already-absent run as success. There is no "clean all". Removing scratch changes no skill, backup, approval or batch state.

## Boundaries to keep

- **Not approval.** No run state, passing check, brief or review package replaces `spec.md` §Approval or a human decision.
- **Not batch state.** `tasks.md` remains the batch ledger. A run cannot authorize another batch, a merge, a publish or a release.
- **Not verification.** `run check` observes one local process. `run record` stores a claim. Neither proves that the task is correct or complete.
- **Not reviewer isolation.** The package is local data for a human-requested review. It isolates no process and grants no reviewer permissions.
- **Not a sandbox.** Commands inherit your environment, permissions and side effects; the private directory is not encryption, tamper-proof storage or protection from another process running as the same user.

For field-level contracts, see the [execution run reference](../reference/execution.md). For flags, exit codes and family defaults, see the [CLI reference](../reference/cli.md). For where run scratch sits among backups and managed stores, see [files and backups](../reference/files-and-backups.md).
