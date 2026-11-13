# Feature 016 — Plan (r1, 2026-11-13)

**Parent spec:** [`spec.md`](spec.md) (DRAFT r1, owner decisions 1B / 2A / 3A / 3B-on-request).
**Status:** APPROVED r1 (2026-11-13) — combined approval in [spec §14](spec.md#14-approval); B1–B3 implementation is complete and **awaiting human review**. The full 11-package gate, race run, and parity checks pass; acceptance is not inferred.
**Target:** root Go 1.23+ CLI and a new `internal/execution` standard-library package. No TUI, Fleet Manager, MCP, routing, installer, or release change.
**Baseline inspected:** commit `a8f1c1c` on 2026-11-13; working tree clean except this feature folder.
**Historical input:** [`016-agent-execution-context-evidence-recovery-spec.md`](016-agent-execution-context-evidence-recovery-spec.md). It is advisory and is superseded by this plan and the canonical spec.

## 1. Repository-first inspection (T01 must confirm before code)

The implementation starts with a read-only inventory. Do not trust this list blindly; recheck the live files and record deviations in this section or in the implementing task's note.

| Area | Verified baseline | Consequence |
|---|---|---|
| CLI dispatch | `cmd/agentic-sdd/main.go` dispatches `preview`, `apply`, `version`, `opencode`, `help`, `backups`, `logs`, `restore`, `guide`, `tui`, `fleet`, `mcp`. `run`, `task`, and `review` are free. | Add three dispatch cases and help topics without renaming existing commands. |
| Help | `cmd/agentic-sdd/helptext.go` holds `topicHelp`; `printUsage` and `isHelpTopic` list commands in `main.go`. | Add `run`, `task`, `review` topics and usage lines; update exact-output tests only where additive lines require it. |
| Output conventions | `Env` in `cmd/agentic-sdd/env.go` owns TTY/color/width; `NO_COLOR` and `TERM=dumb` are honored. `mcp` uses `--json`; the new draft uses `--format text\|json`. | New families use `--format`; existing `--json` behavior is unchanged. New output is plain and deterministic. |
| Go module | `go.mod` is `go 1.23` with no `require`. | Standard library only; no new dependency. |
| Check/evidence helpers | `internal/harnesscheck` provides a contract-driven runner, bounded logs, and exit mapping. `internal/harnessevidence` provides Git scope snapshots and strict receipt/duplicate-key parsing. | Reuse compatible patterns; `run check` needs arbitrary argv and full logs, so it gets a small dedicated runner rather than a second contract engine. |
| Operation seam | `internal/appops` owns immutable typed requests for install/restore/fleet. | No `appops` change in 016. A future TUI calls `internal/execution` only in a separately approved feature. |
| Orchestration skill | `opencode-pack/skills/agentic-sdd-orchestrate/{SKILL.md,references/*}` and `skills-files/agentic-sdd-router/references/{review-resolution,independent-review}.md` exist. | Integrate CLI usage, severity, and request-only reviewer wording there; no new skill or profile. |
| Supported-core gate | `Taskfile.yml`, `harness/checks.json`, `tests/supported-core-go.sh`, and `AGENTS.md` list ten packages (`.`, `internal/version`, `internal/skillsync`, `internal/harnesscheck`, `internal/harnessevidence`, `internal/appops`, `internal/buildinfo`, `internal/fleet`, `cmd/agentic-sdd`, `cmd/sdd-check`). Lefthook calls `task test` / `task vet`. | Add `./internal/execution` to the gate and update the AGENTS.md list/count. This is a scoped 008 fence update included in the combined approval. |
| Run workspace visibility | A `.agentic-sdd/runs/.gitignore` containing `*` makes the entire run tree ignored, including the ignore file itself; `git status --short` stays empty (verified in a disposable repository). | Repo-local scratch is possible without editing the root `.gitignore`. The tool creates the nested ignore file, never rewrites an existing one blindly. |
| Feature states | 009 shipped behavior (verify its live ledger); 013 accepted/published; 014 implementation awaiting human/Opus acceptance; 015 MCP accepted and `v0.9.0` authorized. | 016 depends on 009/013 behavior, not on 014/015 acceptance. Do not reopen or restate their gates. |
| Independent review | Feature 006 `IR-06-01` remains open. | No reviewer sandbox claim; request-only review is proposed, not dispatched by default. |

## 2. Architecture

```text
cmd/agentic-sdd/
  main.go                 add run/task/review dispatch (thin)
  execution_cli.go        shared flags, text/JSON rendering, exit-code mapping
  run.go                  run init|status|record|clean|check adapters
  task.go                 task brief adapter
  review.go               review package adapter
  helptext.go             help topics for the three families
        |
        v
internal/execution/       typed, reusable, standard-library service
  types.go                schema-v1 types, enums, errors, limits
  repository.go           repo root, Git observations, source digests, path containment
  workspace.go            run identity/root, layout, atomic writes, lock, clean
  checkpoint.go           observed state, freshness, reconciliation, status
  taskbrief.go            tasks.md/spec.md extraction and brief rendering
  checks.go               argv execution, capped logs, process status, summary
  reviewpackage.go        Git collection, manifest, package assembly, partiality
  record.go               strict JSON claim parsing and persistence
        |
        +--> read-only Git and repository files
        +--> private run artifacts only
        +--> reuse harness/evidence helpers only where their contracts fit
```

`internal/execution` must not import `cmd/agentic-sdd`, `experiments/tui`, or any UI package. The CLI may import `internal/execution`. The service exposes typed requests/results; rendering and flag parsing stay in `cmd/agentic-sdd`. No business logic is duplicated in skill prose.

### 2.1 Package responsibilities

| File | Responsibility |
|---|---|
| `types.go` | `RunID`, `Scope`, `GitObservation`, `CheckRecord`, `ClaimRecord`, `ReviewManifest`, state enums, schema versions, error sentinels (`ErrUsage`, `ErrUnsafePath`, `ErrStale`, `ErrPartial`). |
| `repository.go` | Canonical repo root; `git rev-parse --show-toplevel`, `--verify HEAD`; branch and worktree fingerprint; source-file digests; relative-path validation; symlink refusal; bounded read helpers. |
| `workspace.go` | Run ID creation, root resolution, `run.json`/`checkpoint.json` layout, `0700`/`0600` permissions, atomic writes, per-run lock, clean with ownership checks. |
| `checkpoint.go` | State precedence, transitions, freshness reasons, status result, next-permitted-investigation values; never approves or resumes. |
| `taskbrief.go` | Batch/task section scanner with fenced-code awareness; exact ID matching; FR/AC reference resolution; brief rendering and idempotence. |
| `checks.go` | `exec.CommandContext` argv execution; no shell; process group cancellation; capped stdout/stderr; timestamps, exit/signal, truncation, summary extraction; metadata. |
| `reviewpackage.go` | Git status/diff collection; committed/staged/unstaged/untracked sections; binary/rename/mode/submodule/conflict handling; sensitive/large screening; manifest and patch artifacts. |
| `record.go` | Strict JSON schema, size cap, duplicate-key rejection, path validation, untrusted claim persistence. |

No new root dependency. No new package solely for a few lines; if `repository.go` or `reviewpackage.go` grows too broad during implementation, split only along a real seam and record it.

## 3. Frozen public contract

The grammar in [spec §5](spec.md#5-public-cli-contract) is frozen with these additions:

- Common flags on every new command: `--repo DIR`, `--run-root DIR`, `--format text|json` (default `text`).
- `run init` requires exactly one of `--feature DIR --batch ID` or `--direct`; `--batch` is invalid with `--direct`.
- `task brief` additionally accepts `--force` to replace a differing brief; without it, a differing existing brief is an error.
- `run check` additionally accepts `--timeout DURATION` (default `15m`; `0` disables) and `--max-output-bytes N` (default `64 MiB` per stream; range `1 KiB`–`1 GiB`). The command vector is everything after `--`.
- `run record` requires `--result FILE|-`; `--task` is required for feature scope and optional for direct scope (default `direct`).
- `review package` additionally accepts `--max-file-bytes N` (default `2 MiB`) and `--max-total-bytes N` (default `32 MiB`).
- `run clean` without `--yes` is a preview; `--yes` is the only deletion authorization. There is no prompt.
- `help run|task|review` prints the topic; unknown flags exit `2`.

**Exit codes:** use the table in [spec §5.1](spec.md#51-exit-codes). `run check` exits `0` only when the supplied command passed; `1` for a failed/interrupted/limit-exceeded command; `2` for usage/validation; `124` for timeout. The actual command exit code and signal stay in the result, not in the wrapper's exit code.

**JSON:** every success or evidence-result object has `"schemaVersion": 1`, `"kind": "execution.<family>.<action>"`, and stable field names. A failed/timed-out/interrupted `run check` and a partial `review package` still emit their result object. Usage and validation errors go to stderr as text and use exit `2`; do not emit a half-valid JSON object for those errors. `text` mode is plain ASCII where practical, uses no color, and prints exact artifact paths.

**Backward compatibility:** existing command output, exit codes, flags, trace events, backup manifests, fleet manifests, MCP output, and TUI boundaries stay unchanged. Additive help lines may update exact-output tests; do not silently change old text.

## 4. Run workspace and identity

### 4.1 Root and layout

Default root: `<repo>/.agentic-sdd/runs`. The service creates `runs/.gitignore` with the single line `*` if absent; it never rewrites an existing ignore file. This makes the entire run tree Git-invisible without touching the root `.gitignore`. If the ignore file exists but does not make the run tree invisible, `run init` fails with exit `2` and asks for an explicit `--run-root`; it does not leave visible scratch. If the repository is not writable, the command fails with exit `2` and tells the user to pass an explicit `--run-root DIR`; there is no silent fallback to a user cache. An explicit run root outside the repository is created as a private directory only, with no ignore file. If an explicit root is inside a Git worktree and is not ignored, the output reports `workspace_git_visible: true`; it never claims invisibility it does not have.

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

Run ID format: `run-<UTC compact>-<6 random hex>`, for example `run-20261113T114500Z-a1b2c3`. It is path-safe and collision-resistant. `run init` uses `os.Mkdir` exclusivity and retries a bounded number of times on collision. Check, record, and package IDs use the same shape (`chk-...`, `rec-...`, `pkg-...`) so artifact paths are deterministic and path-safe.

### 4.2 Identity

`run.json` records:

- schema version, run ID, creation time, tool version;
- scope kind (`feature` or `direct`), repo root, feature directory, `spec.md`/`tasks.md` paths and SHA-256 digests, batch ID, selected batch-section digest, optional direct-scope label; task IDs are recorded per brief/check/record rather than fixed at init;
- baseline Git HEAD, branch if available, and a worktree fingerprint;
- the authoritative approval-reference path (`spec.md` §Approval) but **not** an inferred approval status;
- the resolved run root and artifact root.

The feature directory and both source files must be regular files inside the repository; symlinks, missing files, and paths outside the repo are refused. `tasks.md` is read-only to this feature. `run init` resolves the batch section through the same deterministic parser, requires exactly one match, and records its normalized scope digest; a missing or duplicate batch exits `2` before any workspace is created.

Two `tasks.md` identities are stored: the full-file SHA-256 for provenance, and a normalized **scope digest** over the selected batch header and task section. The scope digest normalizes `- [x]`/`- [X]` to `- [ ]` and ignores lines beginning with `**Status:**`, `**State:**`, `**Next action:**`, or `**Completion:**` (case-insensitive). Checking a task off or adding a progress note therefore does not make the checkpoint stale; changing the task's requirements, boundaries, or batch does.

The worktree fingerprint is a SHA-256 over `git status --porcelain=v1 -z` plus the HEAD ID, a digest of `git diff HEAD --binary --no-ext-diff --no-textconv` (staged and unstaged changes relative to HEAD), and a bounded content digest for untracked files (path, size, and bytes up to 2 MiB per file / 32 MiB total; larger files use path, size, and modification time). Ignored files are excluded. This is an observation, not a security boundary. An unborn HEAD (a repository with no commits) fails `run init` with exit `2` because Git identity and review bases cannot be resolved.

### 4.3 Safety and concurrency

- Every artifact path is resolved under the run directory; reject absolute paths, `..`, NUL, and symlinks in any component.
- Directories `0700`; files `0600`; atomic writes use a same-directory temp file, `Sync`, `Rename`, and directory `Sync`.
- Per-run lock: `O_CREATE|O_EXCL`; content includes a random token, PID, hostname, start time, and operation. Release only removes a lock whose token matches. A stale lock may be reclaimed only when the PID is not alive on the same host or the age exceeds 30 minutes; otherwise fail with a clear conflict.
- `run clean` validates the run root and ownership marker, refuses symlinks, refuses while a live lock exists, removes only the run directory, and reports already-absent runs as success. No recursive delete of the parent.
- Concurrent checks on one run fail on the lock; no interleaved checkpoint write.

## 5. Task brief extraction

The parser is deterministic and fail-closed:

1. Read `tasks.md` (max 4 MiB, valid UTF-8) and locate the batch section by exact heading token (`## B1`, `## B1 — ...`, `## Batch B1 ...`). The section ends at the next `##` heading or EOF.
2. Within the batch, locate exactly one task entry: `### T01 ...`, `### T1: ...`, `### T1 — ...`, or `- [ ] **T01** ...` / `- [x] **T01** ...`. Token matching is exact, so `T1` never matches `T10`.
3. Track fenced code blocks (backtick or tilde fences, up to three leading spaces); headings inside a fence are content, not boundaries. The task section runs until the next `###` or `##` heading, or EOF, and includes nested `####` headings and list content. A `###` heading that is not the selected task ends the section; a `###` heading that looks like a task ID but cannot be parsed is an error, not content.
4. Fail with exit `2` on a missing batch, duplicate batch, missing task, duplicate task in the batch, malformed fence, or a task ID that only appears in another batch. Do not silently choose the first match.
5. Scan the task section for `FR-xx`, `AC-xx`, and explicit `spec.md`/`tasks.md` links. Support comma/slash lists and ranges (`AC-01–AC-13`) by expanding them; resolve every referenced FR/AC row uniquely in `spec.md`; an unresolved or duplicate required reference is an error, not an omission.
6. Render the brief with: run/scope identity; exact source paths, line ranges, and digests; the verbatim task section; the batch header's outcome/verification/state lines when present; the resolved FR/AC rows; and a standard authority/stop block. Do not inline unrelated tasks or the whole spec.
7. Write `briefs/<task>.md` atomically. If it exists with the same digest, no-op; if it differs, fail unless `--force`. The main agent may append a human comment afterward; the source contract remains the canonical section.

No LLM is used to fill gaps. A direct-scope run cannot call `task brief`; it uses a caller-supplied bounded brief in `run record` or the existing SDD text.

## 6. Check execution

- Require `--` and a non-empty argv. Do not invoke a shell, expand globs, or interpolate variables. The invoker may explicitly run `sh -c '...'` if that is the intended command.
- Working directory is the repo root; record the exact absolute path in private metadata. No `--cwd` in v1 unless the repository-first inspection proves it necessary.
- `exec.CommandContext` with a timeout; on Unix set `Setpgid` and cancel the whole process group; on other platforms cancel the process. Set `WaitDelay` to bound orphaned I/O.
- Stream stdout/stderr to separate files with per-stream caps. When a cap is hit, cancel the process, mark `truncated` and `output_limit`, and record the partial logs truthfully.
- Record start/end UTC timestamps, duration, exit code, signal, timeout, output limit, byte counts, log paths, working directory, argv digest, and argv path. Do not dump the environment.
- Text/JSON summary shows the program basename, argument count, argv SHA-256, and the private argv path; it does not echo raw arguments by default. This avoids leaking tokens embedded in commands while preserving exact provenance.
- Salient summary: stream the full log up to the output cap for case-insensitive failure markers (`fail`, `error`, `panic`, `not ok`, `fatal`, `cannot`, `undefined`), deduplicate, cap at 40 lines, and set `summary_truncated` if more matches exist. Append the last 20 lines when no marker is found. Label it as an excerpt; the full log is authoritative. Never claim the excerpt is complete.
- Result states: `passed`, `failed`, `timed_out`, `interrupted`, `output_limit`, `not_started`. A missing program, permission error, or disk-full condition is `failed`/`not_started` with the exact error; it is never a pass.
- One check at a time per run. Update the checkpoint at start (`running`), on completion, and on interruption. Record the worktree observation at start and end; if the check mutates the worktree, mark `worktree_changed_during_check` and do not treat the check as verified evidence.

## 7. Review package

### 7.1 Git collection

Use the `git` CLI, read-only, with `GIT_OPTIONAL_LOCKS=0`, `LC_ALL=C`, and `--no-ext-diff --no-textconv --no-color` so user-configured external diff drivers cannot execute. Disable `core.fsmonitor` for the inspection (`-c core.fsmonitor=false`). Do not stage, commit, reset, checkout, or clean.

Resolve `--base` with `git rev-parse --verify <base>^{commit}`. `--head` defaults to current HEAD; if it differs, the committed section is produced but staged/unstaged/untracked sections are omitted with `head_not_current` and the package is partial. Record whether the base is an ancestor of head; a non-ancestor base is a warning in the manifest, not a silent rewrite.

Sections:

- **committed:** `git diff --binary --find-renames --find-copies base..head -- <paths>`;
- **staged:** `git diff --binary --cached head -- <paths>` only when head is current (head..index);
- **unstaged:** `git diff --binary -- <paths>` only when head is current (index..worktree);
- **untracked:** `git ls-files -z --others --exclude-standard`, copied verbatim under `untracked/`, never following symlinks.

Parse `git status --porcelain=v1 -z` for status, staged/unstaged flags, renames, deletions, mode changes, and unmerged entries. Never parse human-formatted quoted output. Generate per-file diffs in sorted path order; for a rename, pass both the old and new paths so the rename header is preserved.

### 7.2 Manifest and partiality

`review.json` contains schema version, package ID, run ID, creation time, repo root, base/head IDs, branch, Git version, scope identity, `complete`, `partial_reasons`, file entries, omissions, counts, and artifact paths. File entries include path, old path for renames, status, staged/unstaged/untracked flags, binary/submodule flags, mode, size, digest, included, and reason. The manifest also records the run state and stale reasons; those are evidence-freshness warnings, not content omissions. `complete` refers to content coverage, not whether the run's earlier evidence is fresh. Packaging records the current observation and a package entry in the checkpoint; it never marks a task verified or approved.

Default sensitive path patterns: `.env`, `.env.*`, `*.pem`, `*.key`, `*.p12`, `*.jks`, `*.keystore`, `id_rsa`, `id_ed25519`, `.netrc`, `.npmrc`, `*.pypirc`, `credentials*`, `secrets*`. A match omits content, records the path and reason, and makes the package partial. There is no override in v1; the reviewer can inspect locally.

Large/binary/conflicted handling:

- Per-file diff cap `--max-file-bytes`; total cap `--max-total-bytes`. Exceeding a cap omits the file, records size/digest/reason, and marks partial.
- Tracked binary changes use `git diff --binary`; the manifest flags `binary: true`. This preserves a complete patch while telling the reviewer it is not text.
- Untracked files are copied up to the caps; untracked symlinks are represented by a small metadata file containing the link target and are never followed.
- Deleted/renamed/mode changes/submodule pointers are recorded from `--summary`/`--raw` and included where the diff supports them.
- A conflicted index (`u` status or unmerged codes) omits the index section, records `index_conflict`, and makes the package partial. Do not run `git diff --cached` on a conflicted index.
- A missing or unreadable path is recorded as omitted with the exact error; never silently drop it.

Exit `0` only when `complete` is true; exit `3` when a package was written but is partial; exit `1` when no package could be written. The text/JSON output names every partial reason.

## 8. Checkpoint and state machine

`checkpoint.json` is versioned and additive. It stores the run ID, update time, current state, last observation, per-task brief/check/record references, checks, records, blockers, and review packages. It never stores a competing approval or batch ledger.

State precedence (highest to lowest): `blocked`, `running`, `stale`, `observed-failed`, `observed-partial`, `awaiting-verification`, `verified-evidence`, `prepared`. The run-level state is the highest-severity task state; per-task states are also stored. A live check therefore reports `running`; once it ends, the resulting failure/partial/stale state takes over.

| State | Meaning | Exit from it |
|---|---|---|
| `prepared` | Run created; no evidence yet. | A check/record starts or fails. |
| `running` | A check is executing and the lock/heartbeat is live. | Check completes, times out, or is detected interrupted. |
| `observed-partial` | Check evidence exists but is incomplete: timeout, output limit, interrupted check, or a missing artifact. A record claiming `partial` stays `awaiting-verification`. | A fresh complete check, or a human-authorized disposition. |
| `observed-failed` | A check observed a failure. A record that merely claims failure stays `awaiting-verification` until the parent verifies it. | A later passing check on the same observed state. |
| `awaiting-verification` | A claim or check needs parent inspection; not verified. | Parent runs an authorized check or records a bounded disposition. |
| `verified-evidence` | A passing check matches the current source/worktree and no unresolved claim/blocker contradicts it. Still **not** human approval. | Any source/worktree change makes it `stale`. |
| `stale` | Source, batch, HEAD, or worktree changed since the last observation; or an artifact is missing/contradictory. | Reconcile and re-observe; never auto-resume. |
| `blocked` | Explicit blocker (permission, billing, data policy, partial write, unsafe path, ambiguous source). | Human decision or authorized remediation. |

Freshness reasons are explicit strings: `spec_changed`, `tasks_changed`, `batch_changed`, `head_changed`, `worktree_changed`, `scope_mismatch`, `missing_artifact`, `interrupted_check`, `stale_lock`, `contradictory_evidence`, and the informational `progress_updated` (checkbox/status-only change, not stale). `run status` prints them and a `next_permitted_investigation` value (`inspect_changed_paths`, `rerun_authorized_check`, `reconcile_source_change`, `resolve_blocker`, `none`). It never prints `resume`, `start next batch`, or `approved`.

`run record` accepts strict JSON only:

```json
{
  "schemaVersion": 1,
  "taskId": "T01",
  "result": "passed|failed|partial|blocked|needs_input",
  "summary": "bounded text",
  "changedPaths": ["relative/path"],
  "checks": [{"command": "text", "exitCode": 0, "logPath": "optional"}],
  "blockers": [{"kind": "permission|billing|data_policy|transport|test|partial_write|unknown", "detail": "text"}],
  "agent": {"role": "optional", "model": "optional"},
  "unknowns": ["text"]
}
```

Reject unknown fields, duplicate keys, more than 1 MiB, absolute/traversal paths, and NUL bytes. Bound `summary` to 8 KiB, `changedPaths` to 1,000 entries, and `checks`/`blockers`/`unknowns` to 100 entries each; each string is length-bounded. Persist the raw file and its digest. A `passed` claim yields `awaiting-verification`, never `verified-evidence`. A blocker yields `blocked`. Token/spend data is not observed by this utility; status reports `unknown` unless a claim supplies a value, and any supplied value stays labeled `reported`.

## 9. Security and privacy

- No network, model, MCP, telemetry, or credential access in `internal/execution`.
- No environment dump. Command arguments are private metadata by default; summaries use the program basename, arg count, and digest.
- Review packaging disables external diff drivers/textconv and does not mutate the index/worktree.
- Symlinks are refused or recorded, never followed, in run artifacts and untracked review copies.
- Sensitive paths are omitted with an explicit reason; this is pattern screening, not redaction or sandboxing.
- `run clean` is scoped to the owned run directory and refuses symlinked roots.
- Feature 006 `IR-06-01` remains open; no general reviewer isolation claim is made.

## 10. Integration with existing skills

Only these skill/reference files change, and only where the CLI/severity/reviewer contract requires it:

| File | Change |
|---|---|
| `skills-files/agentic-sdd-router/references/review-resolution.md` | Add the Critical/Important/Minor severity table, separate from `valid`/`disputed`/`needs evidence`; state that severity does not bypass required checks or add an approval gate. |
| `skills-files/agentic-sdd-verification-review/SKILL.md` | Findings carry severity; `Ready` requires no unresolved Critical and an authorized disposition for Important items. |
| `skills-files/agentic-sdd-router/references/independent-review.md` | Fresh per-task reviewer is request-only; propose role/package/model/privacy; refuse when isolation/permission is unverified. |
| `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` | Add when to use `run init`, `task brief`, `run check`, `review package`, `run status`; pass artifact references, not transcripts. |
| `opencode-pack/skills/agentic-sdd-orchestrate/references/dispatch-contract.md` | Define the brief packet: exact task, source/digest references, allowed paths, expected checks, stop conditions. |
| `opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md` | Add the five failure categories (provider/transport, model quality/wrong implementation, test/command failure, partial-write/unknown side effects, permission/billing/data-policy denial), checkpoint/stale/interrupted rules, and the bounded-correction/no-auto-escalation boundary; a record is a claim; never auto-resume. |

No new skill name, profile, installer path, routing lane, MCP permission, reviewer role, or mandatory delegation. `internal/skillsync` asset tests must still pass.

## 11. Test strategy

Fixtures use `t.TempDir()` and disposable Git repositories. Set `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, `GOENV=off`, `GOWORK=off`, `GOFLAGS=`; isolate Git config with `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_SYSTEM=/dev/null`, deterministic author/committer values, and `GIT_OPTIONAL_LOCKS=0`. No real home, network, model, or MCP call.

| Group | Coverage | Primary tests |
|---|---|---|
| 1 — CLI/API compatibility | Command syntax/help, text/JSON schema, exit codes, old-command regression, no `cmd`/TUI import inversion | `cmd/agentic-sdd` table tests; `go list -deps` check |
| 2 — Workspace/resume safety | New run, collisions, symlink/traversal, concurrent writers, stale source/worktree/batch, clean, no auto-continuation | `internal/execution` workspace/checkpoint tests |
| 3 — Task extraction | `T1`/`T01`/checkbox forms, duplicate IDs, nested headings, fences, missing task, line ranges | `taskbrief` fixture tests |
| 4 — Truthful check evidence | Pass/fail/timeout/signal/output cap/missing program, failure far from tail, large logs, full-log retrieval, no secret echo | `checks` helper-process tests |
| 5 — Change package accuracy | Committed/staged/unstaged/untracked, binary, rename/delete/mode, submodule, large/sensitive, conflict, `--head` mismatch, no index mutation | `reviewpackage` Git fixture tests |
| 6 — Review/authority | Severity/validity matrix, no default reviewer, request-only gating, failed worker/partial edit/unauthorized escalation negatives, record cannot verify | Skill asset tests + status/record tests |

**Gate updates:** add `./internal/execution` to `Taskfile.yml` (`test`, `vet`), `harness/checks.json` (`test`, `vet` argv), `tests/supported-core-go.sh` (required dir and package list), and `AGENTS.md` (Go layout plus the supported-core package list/count). This is the scoped 008 fence update referenced by the spec approval section. Do not add skips, filters, or tolerances to make an aggregate suite pass.

**Commands:**

```sh
go test ./internal/execution ./cmd/agentic-sdd
go test ./internal/skillsync
task test
task vet
git diff --check
go mod verify
```

No `task preview`/`apply` against a real home; no `task release`; no model call.

## 12. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Git CLI differences across macOS/Linux | Use plumbing/`-z` output, record Git version, test on the repository's Linux CI; do not claim Windows support. |
| Symlink/TOCTOU races | `Lstat` checks, `O_NOFOLLOW` where the platform supports it, private `0700` roots, and a documented residual local-user race. Do not call it a sandbox. |
| Huge logs/diffs exhaust disk | Per-stream/per-file/total caps; terminate and mark partial; exact paths and sizes. |
| Checkpoint state complexity | One table-driven transition test matrix; precedence documented; no `completed` state. |
| Stale observation hides a real change | Recompute source/Git/worktree digests on every status; stale is blocking; no auto-resume. |
| Sensitive content in review packages | Default deny patterns, partial package, no override in v1, no redaction claim. |
| Skill text drifts from CLI behavior | Keep commands in one help source and test named strings; no duplicated state logic in skill prose. |
| Gate/fence conflict | Explicitly approve the 11-package update with this feature; change all four gate lists in one batch; report any aggregate-gate conflict rather than weakening it. |

## 13. Specialist and tool assignments

| Work | Specialist | Tools / why |
|---|---|---|
| Go service, Git/process handling, tests | `golang-pro` (installed) | Principal Go guidance; standard library, `os/exec`, `syscall`, testing. |
| Public CLI/JSON contract | `api-design-principles` (installed) | Stable command grammar, versioned schema, error/exit semantics. |
| Completion review | `agentic-sdd-verification-review` (installed) | One completion-review invocation for the authorized batch; verdict is technical, not human approval. |
| Code discovery | CodeGraph MCP/CLI if `.codegraph/` exists | Read-only symbol/call-path context; no secrets in queries. |
| Library docs | Context7 only if a concrete stdlib/API question needs it | No third-party dependency is planned; do not browse by default. |
| Verification | Repository shell/Git/Go gates | Offline, isolated fixtures; no model or network calls. |

No additional specialist is needed for Markdown skill edits; repository guidance and the SDD skills cover them. Subagent dispatch was authorized by the owner on 2026-11-13; implementation uses the named implement agents under the disjoint file-ownership schedule in `tasks.md`. No subagent may start a real-home apply, product model/network/MCP call, reviewer dispatch, commit, release, or another task outside its assignment.

## 14. Batches (authorized 2026-11-13; dependency-safe waves)

The owner authorized B1–B3 on 2026-11-13 and requested parallel execution with Gemini Flash, DeepSeek max, and free implement agents. The hard dependency graph is preserved: B1 is a prerequisite for B2/B3 code, and parallel writers are limited to disjoint files. One integrated code state per wave; no two agents edit the same file.

| Batch | Tasks | Outcome | Wave / gate |
|---|---|---|---|
| **B1** | T01–T07 | Typed `internal/execution` foundation, private workspace, run identity, checkpoint/status/record/clean, task brief, CLI/help/JSON, gate registration, focused tests. | Wave 1 foundation (T02–T05) + Wave 2 CLI/gate (T06–T07); review before B2 code. |
| **B2** | T08–T11 | `run check`, full log capture, limits/signals, checkpoint integration, stale/interrupted recovery, focused tests. | Wave 2, after the B1 foundation compiles and integrates. |
| **B3** | T12–T17 | Review package/manifest/partial semantics, severity and request-only reviewer skill integration, docs, six-group verification, AC report, handover. | T14–T15 in Wave 1 (disjoint docs/skills); T12–T13 after B1; T16–T17 after B2. |

The exact file ownership and current state are in `tasks.md` §“Parallel execution authorization”. At each wave boundary the parent integrates, runs the named checks, updates the ledger, and continues within the authorized schedule. Human acceptance remains separate; a green wave is not feature acceptance.

## 15. Plan decisions (confirmed with the combined approval)

1. **Three batches, not two.** B1 is already large; separating check evidence (B2) from diff packaging (B3) keeps each reviewable. The spec's proposed grouping is updated accordingly.
2. **`--format text|json` for the new families.** Existing `mcp` keeps `--json`; do not add a second alias to the new commands.
3. **Repo-local workspace with an explicit `--run-root` alternative.** No silent user-cache fallback. The self-ignoring `runs/.gitignore` avoids a root `.gitignore` change.
4. **Wrapper exit codes, not raw command exit codes, for `run check`.** `0` pass, `1` fail/interrupted, `2` usage, `124` timeout; the actual command exit/signal is in the result. This avoids collisions with the wrapper's usage code.
5. **No sensitive-path override in v1.** A sensitive path makes the review package partial; the reviewer inspects locally.
6. **No `--cwd` for checks in v1.** The repo root is the working directory; add it only if T01 proves a concrete need.
7. **No new trace events or TUI/appops code.** The existing `run.start`/`run.end` trace covers command names; the typed service is available for a later approved UI.

If T01 finds a material conflict (for example, an existing helper already owns run state, or the gate cannot accept an 11th package without an owner decision), stop and record it for reconciliation rather than silently changing this plan.
