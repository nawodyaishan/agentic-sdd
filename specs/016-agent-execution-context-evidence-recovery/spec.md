# Feature 016 — Agent Execution Context, Evidence & Recovery

**Status:** APPROVED r1 (2026-11-13) — combined spec/plan/tasks approval recorded in §14. B1–B3 implementation is complete and **awaiting human review**; a green suite is not acceptance.
**Location:** `specs/016-agent-execution-context-evidence-recovery/spec.md`
**Prepared:** 2026-11-13, canonicalized from the 2026-11-12 owner draft.
**Supersedes:** [`016-agent-execution-context-evidence-recovery-spec.md`](016-agent-execution-context-evidence-recovery-spec.md) (historical input retained beside this file; not authority).
**Target:** root Go 1.23+ `agentic-sdd` CLI and a reusable standard-library execution service. No TUI, Fleet Manager, MCP, model-routing, installer, or release change is required.
**Owner decisions:** **D-01 = 1B** (complete public CLI on reusable Go services), **D-02 = 2A** (private temporary disk checkpoints), **D-03 = 3A** (review packages and severity classification), **D-04 = 3B only on explicit request** (fresh per-task reviewer, never automatic).

> **Authority notice.** This is a proposed feature specification, not authority to write repository source, execute checks against a real user environment, call paid models, dispatch agents, change global OpenCode configuration, approve Feature 015's MCP amendment, release, tag, push, deploy, or begin implementation. Canonical Agentic-SDD spec → plan → tasks, **one combined human approval**, one explicitly authorized batch, and human review before continuation remain in force.

## 1. Executive outcome

Give the existing main orchestrator a fully usable, scriptable public CLI for preparing small task briefs, capturing truthful test evidence, producing complete review-change packages, and safely recovering intra-batch execution context. Human authority stays in the existing SDD records; model, fleet, and MCP management stay in Features 009/014/015.

The implementation is **CLI-first and library-first**: public commands call a cohesive root-Go execution service, not four shell scripts and not logic embedded in the TUI. Existing `agentic-sdd` commands remain compatible, and a future UI can call typed APIs rather than shelling out or parsing CLI output.

**One-sentence success:** after a permitted engineering task, the main agent can point a bounded child at a small brief, preserve full test/diff evidence on disk, return a concise verifiable summary, recover from compaction or interruption without trusting stale progress, and stop at the existing human checkpoint.

### 1.1 What this feature does not replace

| Existing capability | Boundary preserved | Gap 016 addresses |
|---|---|---|
| 009 `agentic-sdd-orchestrate`, profiles and routing | One human-facing main; opt-in delegation; provider/spend policy; no uncontrolled swarms | Deterministic task/evidence preparation and recoverable runtime artifacts, without another model router |
| 010/011 `guide`, CLI UX and flow trace | Existing help, stdout/stderr, trace, flags and non-TTY contracts | New command families with plain, machine-consumable responses |
| 012/013 TUI and `internal/appops` | UI owns display/draft state; backend owns verified facts | Future reuse through typed execution APIs; **no new TUI screen in 016** |
| 014 `internal/fleet` and 015 Fleet Manager | User model bindings, installed-file provenance, save/apply consent | Execution run workspace/checkpoint, not fleet configuration, provenance, or MCP setup |
| `sdd-check`, `internal/harnesscheck`, `internal/harnessevidence` | Deterministic check contracts and evidence helpers; experimental native harness boundaries | Reuse compatible building blocks; do not expand experimental runtime admission |
| Existing review-resolution and one completion self-review | Risk-based review, one batch, human stop | Complete review artifact packaging and severity classification; request-only fresh reviewer |

## 2. Owner decisions and binding rules

| ID | Selected decision | Binding rule |
|---|---|---|
| **D-01** | Public CLI (Q1=B) | Ship documented commands for creating/inspecting runs, extracting briefs, capturing check results, producing review packages, and cleanly removing a chosen run. All operational logic lives in reusable Go packages. |
| **D-02** | Disk checkpoint (Q2=A) | Store temporary, private, Git-invisible **execution observations** with run identity, source digest, and verification evidence. Never store a second authoritative approval or batch ledger. Reconcile before resume. |
| **D-03** | Evidence + severity (Q3=A) | Produce review packages; classify findings as **Critical**, **Important**, or **Minor**, separately from evidentiary status (`valid`, `disputed`, `needs evidence`). Keep the current optional independent review and completion-review policy. |
| **D-04** | Requested per-task reviewer (Q3=B conditionally) | The user may explicitly request a fresh reviewer for a named task or bounded set. It is **off by default**, not implied by package creation, and refused when effective permissions or isolation cannot satisfy the intended review. |
| **D-05** | Architecture | Root Go 1.23+ standard-library-first, typed service/API shared by CLI and eventual UI, no root external dependency unless separately approved, no new autonomous daemon or agent runtime. |
| **D-06** | Efficiency and validation | Minimal targeted tests of correctness and safety; no blanket multi-agent benchmarking, full regression-suite expansion, or paid live model calls by default. |

## 3. Relationship to existing capabilities

- **009** is the shipped orchestration policy and profile source; verify its accepted behavior before relying on it. 016 supplies execution-context commands; it does not add a skill name, primary agent, profile installer, routing lane, fallback chain, or MCP permission.
- **013** owns the TUI and `internal/appops` seam. 016 adds no TUI page and does not import `cmd/agentic-sdd` into UI packages.
- **014/015** may still have pending review or gated amendments. 016 does not require accepting or completing them and must not conflict with their command grammar. Model bindings and Fleet Manager state are out of scope.
- **008** owns the supported-core test selection. Adding `./internal/execution` to that selection is a scoped, explicitly reviewed fence update in the plan; it is not a new release claim.
- **006 `IR-06-01`** remains unresolved. A safe local run workspace is **not** general reviewer sandbox certification.

## 4. Actors, journeys, and boundaries

| Actor | Journey | Expected outcome |
|---|---|---|
| Human developer | Approves feature/batch, selects direct or delegated work, optionally requests review | Retains final control; one batch at a time |
| Main intelligent agent | Creates a run, extracts a brief, optionally delegates, records truthful evidence, summarizes | Context-efficient handoff and independently verified result |
| Specialist child | Reads only the authorized brief/source, returns observations/changes | No power to approve tasks, select paid routes, or invoke the next batch |
| CI / CLI automation | Invokes the same commands with explicit arguments and JSON output | Stable exit codes and machine-readable status; no prompts, network inference, or hidden model calls |
| Future Fleet/TUI consumer | Calls the typed execution service when separately authorized | No direct file mutation or CLI-output parsing from a screen |

### 4.1 Execution scope

Feature 016 **does not** automatically invoke an OpenCode subagent or decide which model to use. It prepares and records execution context. The existing 009 orchestration skill decides whether to call any agent, subject to prior human approval and host/runtime permissions.

`run check` executes a supplied **local command** only when a human or authorized orchestrator invokes it. It is **not a sandbox**. Tests can have side effects and require the same authorization as direct test execution. A dry run or preview must never execute the command.

## 5. Public CLI contract

The command families are `run`, `task`, and `review`. Exact flag names are frozen by the plan after the repository-first inspection task; the following grammar is the intended public design.

```text
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

**Contract clarifications:**

1. `--repo` defaults to the current working tree for local read-only inspection. Commands that write or execute must surface the resolved target and require a valid run identity. A feature approval is never inferred from a directory name, `RUN_ID`, or `--yes`.
2. `run init` is a local scratch write, not permission to implement. `task brief`, `run check`, `review package`, and `run record` may write only their designated private run artifacts, apart from side effects of an explicitly authorized check command.
3. `--format json` is machine-readable with `schemaVersion` and `kind`, exact artifact paths, and `unknown` where a value is not observable. Human `text` mode prints compact status plus precise log/artifact pointers. Decorative colors, spinners, and progress bars never enter JSON or piped output. A failed, timed-out, or interrupted `run check` and a partial `review package` still emit their versioned result object so CI can consume the evidence; usage and validation errors are text on stderr with no partial JSON object.
4. All commands work unattended with explicit arguments. No hidden prompt, model request, authentication check, network call, or auto-install. `run clean` without `--yes` is a preview; `--yes` acknowledges deletion of that run's scratch only, **not** SDD/batch authorization.
5. New commands keep old commands, exit codes, and output unchanged. New JSON fields are additive and versioned; existing fleet/trace schemas are not mutated.
6. `run record` accepts a child or tool report as **untrusted input**, labels it *reported*, and never sets human approval, verification, or next-batch authority. The input is a small validated JSON document, not arbitrary YAML or executable content.
7. `run status` displays observed run artifacts, original feature/batch identities, worktree freshness, and whether the checkpoint is **stale** or `needs reconciliation`. It never declares a task verified solely from a `record` claim.
8. `run clean` removes only the owned run directory, is idempotent for an already-absent run, and never follows symlinks or deletes unrelated files. No automatic background purging and no destructive global "clean all" command.
9. `--direct` creates a clearly identified direct-scope run without fabricating feature files. `task brief` requires feature scope; direct runs use the other commands with an optional `--label`, and `run check`/`run record` default the task label to `direct`.

### 5.1 Exit codes

| Code | Meaning |
|---|---|
| `0` | Command completed successfully. For `run status`, the observation was read successfully even if the reported state is `stale` or `blocked`; callers inspect the state field. For `run check`, the supplied command passed. |
| `1` | Runtime, I/O, interrupted, or failed check. For `run check`, the supplied command failed, was interrupted, or exceeded a limit; the actual exit/signal is in the output and metadata. |
| `2` | Usage, invalid arguments, ambiguous/missing task, unsafe path, or unresolved required reference. |
| `3` | `review package` wrote a partial package. The artifact and omissions are real; the package must not be called complete. |
| `124` | `run check` timed out. |

`run status` reports `prepared`, `running`, `observed-partial`, `observed-failed`, `awaiting-verification`, `verified-evidence`, `stale`, or `blocked` as data, not as an approval. A failed or interrupted command never returns success.

## 6. Functional requirements

### FR-01 — Source of authority and run identity

A run is bound to an explicit repository root, feature `spec.md` and `tasks.md` identity where applicable, authorized batch identifier, any task IDs observed in its briefs/checks, and a unique collision-resistant run ID. It records Git HEAD and the tracked/untracked worktree fingerprint **as observations**, not authorizations. Direct-fix mode is supported as a clearly identified direct-scope operation without fabricating feature files. A run never silently starts another batch after it ends.

Run initialization does not assert that a feature is approved because `spec.md` contains a string. It may report canonical metadata and unresolved status. The parent agent must still inspect the approved scope and human authorization. `tasks.md` remains canonical for the selected batch and continuation; checkpoints are disposable observations.

### FR-02 — Private, safely isolated scratch workspace

Create a run-specific directory under a safe canonical root with normalized IDs. Repo-local `.agentic-sdd/runs/<run-id>/` is preferred because a self-ignoring `runs/.gitignore` keeps it invisible **without modifying tracked project files**; when the repository is not writable, an explicit `--run-root` private location is required and its exact path is reported. Use restrictive permissions (directories `0700`, files `0600` where applicable), reject symlinks and unsafe path traversal, guard against collision and concurrent writers, write checkpoint files atomically, and retain exact provenance.

Do not silently modify existing `.gitignore` or tracked project files merely to create scratch. Avoid raw credentials, unnecessary environment dumps, and automatic remote sync or telemetry. Provide explicit selective cleanup. Git-ignore alone is not privacy, permission containment, or proof of sandboxing.

### FR-03 — Deterministic task brief extraction

Extract one **exact** task section from the canonical `tasks.md`, retaining its subtask details and necessary specification/acceptance references, without including subsequent unrelated tasks. Support real task IDs used by this repository (for example `T01`, `T1`, and bold checkbox task entries), not only numeric `Task 1`; avoid substring matches (`T1` versus `T10`), mishandled nested headings, malformed code-fence swallowing, duplicate task IDs, and missing tasks. Fail closed when the parser cannot uniquely identify the target or required constraints.

A brief contains task source/revision, authorized scope as a reference/claim (not fresh authority), dependencies, allowed files/operations, expected checks, and stop conditions. The CLI must not use an LLM to invent missing requirements. The main agent may add a concise human-readable comment afterward without changing the source contract.

### FR-04 — Tool/check output buffering with truthful results

`run check` executes an explicitly supplied executable/argument vector without implicit shell interpolation (the `--` boundary is required). Capture **complete** stdout and stderr with an accurate process status, signal/cancellation observation where available, start/end timestamps, command/working-directory identity, and private full-log paths. Return a compact summary containing salient failures and pointers to the full logs. Do not rely only on the last three, five, or fifteen lines; a failure can occur earlier.

Set sensible resource/time/output-storage limits or explicit configuration for huge or long-running checks. Handle zero/nonzero exits, timeouts, cancellation, binary output, missing programs, disk-full failures, and interrupted writes with truthful incomplete/failed state; never silently replace a failing check with success. A passing check is evidence about the executed command, not feature approval. Never echo secrets from command arguments or environment into the summary or JSON by default.

Reuse existing check/evidence tools where practical; this feature must not activate the inactive experimental native ACP harness or change its admission restrictions.

### FR-05 — Complete Git change/review package

`review package` identifies the intended base/revision and reflects the **actual selected scope**: committed changes where requested, index/staged changes, unstaged changes, plus untracked additions, without silently treating `BASE..HEAD` as a complete dirty-worktree diff. Handle non-text/binary, deleted/renamed, permission changes, submodules, huge files, conflicted indexes, path quoting, and missing/unreadable content with explicit statuses.

Generate a review manifest including exact repository/revision identity, bounded change summaries, a complete or explicitly partial diff artifact, file paths, and omissions. Prefer access to the minimum reviewed material instead of duplicating an entire repository. Screen or mask obvious sensitive paths and stop on prohibited inputs; do not claim automated redaction is perfect or that handing a reviewer a file provides filesystem isolation. If a complete and safe package cannot be built, fail or label it partial and require reconciliation before review.

### FR-06 — Execution checkpoint and stale-state reconciliation

Maintain a versioned private checkpoint with run, feature/batch/task, current Git identity, observed command results, worker-reported status, path/digest references, and open blockers. State distinguishes **prepared**, **running**, **observed-partial**, **observed-failed**, **awaiting-verification**, **verified-evidence**, **stale**, and **blocked** without implying human approval.

After compaction, session restart, or process interruption, `run status` re-checks repository identity and changed-worktree/target digests, compares canonical task/batch authority, detects contradictory or stale entries, and reports the next permissible *investigation*, not an automatic resume of a writer, command, reviewer, or paid model. Store a source reference/digest rather than replacing `tasks.md` with the scratch checkpoint. Avoid ambiguous "completed" states; a record of worker success is only a claim.

### FR-07 — Failure classification and bounded correction circuit breaker

The existing `agentic-sdd-orchestrate` failure guidance distinguishes: (a) provider/transport failure; (b) model quality or wrong implementation; (c) test/command failure; (d) partial-write/unknown side effects; (e) permission, billing, or data-policy denial. Capture evidence and stop on structural or authorization failures. Within an authorized batch, the main may diagnose and make a focused correction; default to the existing limited correction passes, then stop for a human decision if unresolved.

Do not automatically escalate to a more expensive model or different provider because code or tests failed, silently rerun partially successful edits, call a blocked reviewer, or select another task/batch. An explicit human request may authorize a different agent/model/task, but it must pass the existing 009 privacy/cost and role gates.

### FR-08 — Severity and evidentiary resolution for reviews

Extend existing review-resolution guidance with **severity** as a separate dimension from **validity/evidence**:

- **Critical:** security/data-loss/functional failure; blocks acceptance until resolved or explicitly addressed by an authorized scope decision.
- **Important:** material requirement, correctness, or robustness gap; address within scope before completion or document an authorized scoped disposition.
- **Minor:** purely cosmetic/preference/low-impact hygiene; report as deferred/non-blocking **only if** there is no hidden requirement or safety consequence.

Existing validity states remain (`valid`, `disputed`, `needs evidence`). Each finding cites file/line or reproducible evidence, the affected requirement, and the consequence. The parent verifies substantial findings before initiating fixes. Severity categories do not allow silently bypassing a spec-mandated check. No mandatory second independent review or extra human approval round.

### FR-09 — Request-only fresh reviewer per delegated task

A human may explicitly request *a fresh reviewer for Task Txx* or selected tasks. That opt-in causes the orchestrator to propose the reviewer role, input package, model/privacy cost, and evidence policy; dispatch occurs only after satisfying the relevant existing authorization and runtime security restrictions. Never activate a generic native independent-review profile against private source if `IR-06-01` remains applicable and unmitigated. A sanitized export with a verified boundary, or a clearly labeled advisory alternative with appropriately limited source access, may be considered only when independently shown safe for that scope.

Request-only fresh review does not create compulsory per-task review, extra model calls, parallel writers, auto-acceptance, or a new approval ledger. If the user does not request it, the existing single completion self-review remains the default. The orchestrator may always prepare a review package without dispatching an external reviewer.

### FR-10 — Integration with the existing orchestration skill

Update only the necessary `opencode-pack/skills/agentic-sdd-orchestrate/` instructions and references so the main agent knows when to run `task brief`, `run check`, `review package`, and `run status`, and when direct tools suffice. The main gives subagents a short exact task packet and artifact references, never a full conversation dump by default. It independently inspects consequential child claims and actual diffs/tests before reporting outcomes.

No new skill name, primary agent, agent-profile installer change, model routing/fallback chain, MCP permission change, additional review role, or mandatory delegation procedure is required by Feature 016.

### FR-11 — Public CLI compatibility and future consumers

New command families are discoverable through `agentic-sdd help` and `--help`, usable by a human shell and OpenCode agents, and predictable under pipes/CI. Reuse existing CLI environment/diagnostic conventions: plain stdout for data, stderr for errors, no spinners or ANSI under non-TTY, `NO_COLOR` honored, predictable EOF/cancel semantics, and deterministic ordering where appropriate. Maintain backwards compatibility with `fleet status|plan|apply|restore|doctor`, `guide`, `preview`, `apply`, `restore`, `logs`, `version`, existing trace event schemas, and optional `--with-opencode` presets.

Create typed root package requests and results with a stable versioned JSON shape; keep command parsing and presentation thin. A later TUI may call the typed API through its supported `internal/appops` seam only in a separately approved feature. Feature 016 adds no TUI page, application shell, or experimental TUI code to the root Go module.

### FR-12 — Honest accounting, provenance, and security

Capture only actual observable metadata: check execution outcome, run artifact bytes, logged summary, source IDs, optionally model/role reported by the parent or child, and `unknown` for unsupported token/spend fields. A CLI file utility cannot inspect OpenCode live sessions or prove an agent used a model. Do not claim saved token percentages, zero duplicate work, automatic runtime containment, cost caps, or successful review without supporting evidence.

Run files and review packages may include sensitive source; avoid uploading or external MCP access by default. `run clean` removes only its own artifacts. Enforce no uncontrolled symlink reads/writes in helper path handling while preserving Feature 006's separate `IR-06-01` blocker; a safe local workspace implementation is not general reviewer sandbox certification.

## 7. Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-01** | Root **Go 1.23+**, standard-library-first; no new third-party root dependency without separate approval; nested Go 1.26 TUI untouched. |
| **NFR-02** | All CLI behavior is exposed through reusable typed services; no business logic split between CLI and skill prose or copied into a future TUI. |
| **NFR-03** | No model calls, MCP calls, network requests, background monitoring, or credential reads by the execution utility. It works completely offline. |
| **NFR-04** | Safe path normalization, symlink/path-escape refusal, atomic writes, restrictive file permissions, bounded processing, crash recovery, and per-run concurrency control. |
| **NFR-05** | A failed command, report, or partial disk write never produces a false `verified` result or hides an earlier failure. |
| **NFR-06** | Existing non-TTY output/exit behavior and current configuration/installation manifests remain unchanged for preexisting commands. |
| **NFR-07** | Versioned run/evidence records with documented compatibility and cleanup semantics; no automatic Git commit, branch, or rebase. |
| **NFR-08** | No surprise profile activation, billed inference, changed permissions, blocked ACP admission bypass, or release operation. |

## 8. Acceptance criteria

| ID | Acceptance condition |
|---|---|
| **AC-01** | `run init`, `task brief`, `run check`, `review package`, `run status`, `run record`, and `run clean` exist with coherent help and explicit-argument CI behavior; no collision or regression in existing CLI groups. |
| **AC-02** | All new commands route through tested reusable Go execution services; no TUI/agent instruction contains a second implementation of state or Git/package logic. |
| **AC-03** | A run workspace has correct identity, is private and Git-invisible without unintended tracked changes; collisions, symlinks, path traversal, and concurrent writers are handled safely. |
| **AC-04** | Task extraction selects only the requested task, includes necessary scoped requirements, and fails on duplicate/ambiguous IDs and malformed section boundaries. |
| **AC-05** | `run check` records accurate exit status and full logs for passes, failures, and interruption; the summary reports meaningful errors and gives an exact retrieval path. |
| **AC-06** | The review package reflects committed/staged/unstaged/untracked changes as applicable and accurately identifies omissions, binary/large/sensitive-risk data; no false complete-diff claim. |
| **AC-07** | The checkpoint survives compaction/CLI restart but becomes `stale` after source/worktree/batch changes and cannot itself authorize resume or the next batch. |
| **AC-08** | Failure handling stops at the authorized boundary; no automatic repeated writes, provider escalation, external reviewer, next batch, publish, or paid call. |
| **AC-09** | Finding severity (`Critical`, `Important`, `Minor`) and evidence validity remain distinct and are integrated into current review-resolution wording. |
| **AC-10** | A fresh per-task reviewer is **not dispatched by default**, can be proposed only when specifically requested, and is refused when relevant permission/isolation conditions are not verified. |
| **AC-11** | Existing `agentic-sdd-orchestrate` references describe CLI use, context minimization, and truthful parent verification without duplicating approvals or adding a new skill. |
| **AC-12** | Existing CLI commands retain output/exit/backup/fleet/TUI boundary behavior; new text/JSON outputs are valid, versioned, bounded, and usable offline under CI. |
| **AC-13** | A direct-scope run works without fabricating feature files, is visibly distinct from a feature run, and cannot be mistaken for feature/batch approval. |

## 9. Edge cases and failure semantics

| Situation | Required behavior |
|---|---|
| Run ID not found under the selected root | Exit `1` with a clear message; no run is created and no other run is inspected. |
| Run ID found with a mismatched repository root | Exit `2`; the command refuses the mismatch rather than reading an unrelated run. |
| Duplicate task ID in one batch | `task brief` exits `2`; no brief is written. |
| Task ID appears in another batch only | `task brief` exits `2`; the run's selected batch is authoritative. |
| Missing/ambiguous `spec.md` or `tasks.md` | `run init` exits `2`; no run is created. |
| Non-Git repository, unborn HEAD, or read-only repository without an explicit writable `--run-root` | `run init` exits `2` with a clear reason; no partial workspace is left behind. |
| Symlink in the run root, a parent, or an artifact path | Refuse the operation; never follow it. |
| Concurrent `run check` or `run init` on one run | Second writer fails with a lock error; no interleaved checkpoint write. |
| Check timeout | Process is terminated; state is `observed-partial`; exit `124`; log paths and signal are recorded. |
| Output cap exceeded | Process is terminated; log is marked truncated; state is `observed-partial`; exit `1`. |
| Command missing or not executable | Exit `1`; no false pass; exact error recorded. |
| Disk full or interrupted artifact write | Atomic write leaves the previous valid checkpoint or an explicit incomplete marker; never a half-valid JSON file. |
| Review base missing/invalid | Exit `2`; no package. |
| Review head differs from current HEAD | Committed section is produced; staged/unstaged/untracked sections are marked omitted with reason; package is partial (exit `3`). |
| Conflicted index | Index section is omitted and marked; package is partial (exit `3`). |
| Sensitive path changed | Content is omitted and listed; package is partial (exit `3`). No redaction claim. |
| Huge or binary file | Binary diff/metadata or omission is explicit; no silent content loss. |
| `run clean` without `--yes` | Preview only; exit `0`; nothing deleted. |
| `run clean` for an absent run | Report already absent; exit `0`; no error for an idempotent cleanup. |
| Unknown JSON field in `run record` | Exit `2`; record is rejected, not partially stored. |
| Pipe closes during output | Command stops cleanly; no panic; the exit code reflects the failure. |

## 10. Security, privacy, and authority

- Run artifacts are private scratch: directories `0700`, files `0600` where applicable, no automatic upload, no telemetry, no remote sync.
- The CLI does not read credentials, dump environment variables, or echo secret-like arguments in summaries/JSON by default. Full command metadata stays in the private run artifact.
- The CLI is not a sandbox. It runs a command the invoker explicitly supplies, with the invoker's permissions and side effects.
- Git inspection is read-only: no commit, branch, rebase, checkout, stage, reset, or clean. External diff drivers and textconv are disabled for review packaging.
- Feature/batch approval, model selection, paid inference, reviewer dispatch, and release authority remain outside this feature.
- `IR-06-01` remains open; no reviewer sandbox or isolation claim is created.

## 11. Test and evidence expectations

Use six focused deterministic groups with fixtures and temporary repository/workspace roots only; no real user home mutation, provider inference, or broad expensive validation cycle:

1. **CLI/API compatibility:** syntax/help/text/JSON; old CLI output unaffected; typed service reuse and no TUI import inversion.
2. **Workspace and resume safety:** new run, colliding IDs, unsafe symlink/path, concurrency, stale Git state, stale batch, safe cleanup, and no automatic continuation.
3. **Task section extraction:** real task-ID formats, nested headings/fences, missing/duplicate IDs, and precise source references.
4. **Truthful check evidence:** passing/failing/timeout/interrupted command, failure line far from the tail, large output, and secure full-log retrieval.
5. **Change package accuracy:** committed, staged, unstaged, untracked, binary, renames, and sensitive/oversized exclusions; partial data labeled honestly.
6. **Review/authority:** severity/validity matrix; no default extra reviewer; explicit request remains gated; negative tests for failed worker, partial edits, and unauthorized escalation.

An optional manual smoke may run one harmless local test command and generate one package in a disposable Git fixture. A model-based subagent smoke test requires separate human authorization, known privacy and cost terms, and is not a default acceptance requirement.

## 12. Delivery grouping (approved 2026-11-13; dependency-safe waves)

The owner approved B1–B3 and requested parallel execution with Gemini Flash, DeepSeek max, and free implement agents. This is reconciled with the hard dependency graph: B1 is a prerequisite for B2/B3 code, and parallel writers are limited to disjoint files. One integrated code state per wave; human acceptance remains separate.

| Batch | Scope | Wave / decision gate |
|---|---|---|
| **B1 — Typed execution foundation, run lifecycle, and task briefs** | Repository-first inventory; shared types, repository/Git identity, private workspace, checkpoint/status/record/clean, task-brief extraction, `run init|status|record|clean` and `task brief`, help/JSON, core safety tests | Wave 1 foundation and Wave 2 CLI/gate; demonstrate a private workspace, accurate contracts, and fail-closed brief extraction before B2 code |
| **B2 — Truthful check evidence and recovery** | `run check`, full log capture, limits/signals, checkpoint integration, stale/interrupted reconciliation, focused safety tests | Wave 2, after the B1 foundation integrates; demonstrate truthful pass/fail/interruption evidence |
| **B3 — Review packages, severity guidance, and closeout** | Git change collection, review package/manifest/partial semantics, severity and request-only reviewer wording in existing skills, docs, six-group verification, AC report | T14/T15 in Wave 1 (disjoint docs/skills); T12–T13 after B1; T16–T17 after B2; demonstrate complete/partial package accuracy and no default reviewer |

Do not silently expand these batches with a provider monitor, custom plugin, worktree manager, MCP system, Fleet Manager TUI page, public agent executor, or new portability framework. If B1 reveals that existing deterministic services fully cover a requirement, reuse them rather than building another subsystem.

## 13. Dependencies, exclusions, and explicit risks

### Dependencies

- Existing canonical SDD authority and `AGENTS.md` rules; `tasks.md` is the task/batch authority, scratch is not.
- Feature 009 shipped orchestration skill and current review-resolution guidance; verify the accepted implementation rather than relying on older draft status.
- Features 010–013 existing CLI conventions and the optional TUI architectural boundary.
- Features 014/015 may still have pending review or gated amendments; 016 does not require accepting or completing them and must not conflict with their command grammar.
- Existing `internal/harnesscheck` and `internal/harnessevidence` APIs and the disabled native adapter; inspect and reuse carefully, with no claim of enabled native runtime.
- The repository's supported-core gate and 008 fence; adding `./internal/execution` is a scoped fence update included in this feature's combined approval request.

### Explicit exclusions

- No new subagent profiles or model/provider routing and no agent-dispatch engine.
- No automatic provider failover, paid-token usage, model quality evaluation, or quota monitoring.
- No automatic fresh reviewers, mandatory per-task review, duplicate completion self-review, or multi-writer concurrency.
- No independent-review sandbox certification or closure of Feature 006 `IR-06-01`.
- No redefinition of Feature 015's MCP install/connection/permission amendment or Fleet Manager TUI.
- No worktree creation manager, task-scheduling daemon, watch mode, or background retention sweeper.
- No changes to Feature 008 release claims, core skill count, or existing user installation behavior.
- No release, tag, push, or deploy without separate approval.

### Risks and mitigations

| Risk | Mitigation |
|---|---|
| CLI proliferation and unstable grammar | Existing CLI conventions; coherent `run`, `task`, `review` families; one documented contract and typed engine; no UI rewrites. |
| Stale checkpoints mislead the main agent | Recheck repo + batch + worktree digest; never auto-resume; stale is a blocking state. |
| Sensitive diff/log data leaks | Private workspace, no auto-upload, minimal output, explicit risk exclusions, user-controlled deletion. |
| `BASE..HEAD` misses dirty files | Comprehensive staged/unstaged/untracked awareness and a manifest of omissions. |
| A tool wrapper seems to confer authority | Explicitly separate CLI command execution from feature/batch/model authorization; wrapper is not a sandbox. |
| False reviewer security claims | Existing Feature 006 block; reviewer source access requires a separately verified safe boundary. |
| Duplicating harness or fleet functionality | Source audit before implementation; reuse typed packages instead of copying functionality. |
| Too many tests or new dependencies | Six targeted groups, fixture-only model-free verification, root standard-library-first. |

## 14. Approval

**Combined decision:** **Approved, 2026-11-13.** The owner approved the reviewed `spec.md` r1, `plan.md` r1 and `tasks.md` r1 and authorized implementation of B1–B3 using the named implement subagents.

- **Approver:** The repository owner in this conversation (no additional identity asserted).
- **Approved revisions/scope:** the canonical `spec.md` r1, `plan.md` r1 and `tasks.md` r1 as presented in this session.
- **Decision evidence:** the owner said “i accept 3 docs” and “start implementation using gemini flash, deepseek max and free implement sub agents all batches parallylly” on 2026-11-13.
- **Authorized implementation:** B1–B3, using the specified implement agents, subject to the dependency-safe parallel schedule and file-ownership rules recorded in `tasks.md`. B1 is a hard prerequisite for B2/B3 code because the batches share `internal/execution`; parallel writers are limited to disjoint files. “All batches in parallel” is therefore reconciled as parallel disjoint work, not three simultaneous writers in one checkout.
- **Authorized fence update:** adding `./internal/execution` to the supported-core gate and the related `AGENTS.md` package list, as described in `plan.md` §11.
- **Still not authorized:** real-home writes or `apply`, code-executing checks against the owner's environment, product model/network/MCP calls, paid inference outside the specified implementation agents, fresh reviewer dispatch, Feature 015 MCP amendment approval, tag, release, push, or deployment.

Approval is recorded once, here, tied to the reviewed revisions. `plan.md` and `tasks.md` reference this section; they do not maintain competing approval flags.

## 15. Evidence and research provenance

### Primary records

- Owner draft: [`016-agent-execution-context-evidence-recovery-spec.md`](016-agent-execution-context-evidence-recovery-spec.md) (2026-11-12).
- Owner decisions in this conversation: **1B, 2A, 3A, and 3B only on explicit request**.
- Current repository inspection on 2026-11-13 at commit `a8f1c1c`: CLI dispatch/help conventions, `internal/harnesscheck`, `internal/harnessevidence`, `internal/appops`, the 009 orchestration skill, review-resolution references, and the 008 supported-core gate.
- Feature 009–015 specifications and task ledgers in `specs/`; their live state is authoritative for what is accepted or gated.

### External, advisory engineering references

- Superpowers [inline `executing-plans`](https://github.com/obra/superpowers/blob/main/skills/executing-plans/SKILL.md), [subagent-driven development](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/SKILL.md), [task-brief helper](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/scripts/task-brief), and [scratch workspace helper](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/scripts/sdd-workspace) — advisory comparisons, not measured proof.
- OpenCode [configuration](https://opencode.ai/docs/config/) and [compaction](https://opencode.ai/docs/compaction/) — advisory; use native facilities where sufficient.

**Feature principle:** *A public CLI for truthful engineering execution evidence, a small reusable Go core, optional specialist reviews, and durable recovery without transferring human authority to the harness.*
