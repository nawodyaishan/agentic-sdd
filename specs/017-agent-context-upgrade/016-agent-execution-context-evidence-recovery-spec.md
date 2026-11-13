# Feature 016 — Agent Execution Context, Evidence & Recovery

**Status:** DRAFT r1 — awaiting combined specification/plan/tasks approval and explicit implementation-batch authorization  
**Prepared:** 2026-11-12  
**Proposed canonical path:** `specs/016-agent-execution-context-evidence-recovery/spec.md` (confirm `016` is unused in the live checkout)  
**Primary user decisions:** **Q1=B** (complete public CLI commands on a reusable Go architecture), **Q2=A** (temporary disk checkpoints), **Q3=A with B on explicit request** (review packages + severity classification by default; optional fresh per-task reviewer **only** on specific human request).  
**Product:** Agentic-SDD (OpenCode-first orchestration; portable engineering process retained)  
**Prerequisites:** Review the currently delivered/accepted Feature 009 and Feature 013 contracts, the separately gated 014 work and Feature 015/MCP amendment status. Do not treat incomplete or historically approved work as accepted without checking live `tasks.md`/Git state.

> **Authority notice:** This is a proposed feature specification, not authority to write repository source, call paid models, dispatch agents, execute tests against the user's real environment, change global OpenCode configuration, approve Feature 015's MCP amendment, release, tag, push, deploy or start implementation. Canonical Agentic-SDD spec → plan → tasks; **one combined human approval**, one explicitly authorized batch, and mandatory human review before continuation remain in force.

## 1. Executive outcome

**Give the existing main orchestrator a fully usable, scriptable public CLI for preparing small task briefs, capturing truthful test evidence, producing complete review-change packages, and safely recovering intra-batch execution context.** Human authority stays in existing SDD records; model and fleet management stay in Features 009/014/015.

The implementation must be **CLI-first and library-first**, not four isolated shell scripts or logic embedded in the TUI. It adds a cohesive root-Go execution/evidence service and public CLI command families that call it. Existing `agentic-sdd` commands remain compatible, and future UI integration uses typed APIs rather than shelling out or parsing CLI output.

**One-sentence success:** After a permitted engineering task, the main agent can point a bounded child at a small brief, preserve full test/diff evidence on disk, return a concise verifiable summary, recover from compaction/interruption without trusting stale progress, and stop at the existing human checkpoint.

### 1.1 What is new versus what is already delivered

| Existing capability | Boundary this feature preserves | Gap Feature 016 addresses |
|---|---|---|
| 009 `agentic-sdd-orchestrate`, `opencode-pack/`, specialist profiles and routing | One human-facing main; opt-in delegation; provider/spend policy; no uncontrolled swarms | Deterministic task/evidence preparation and recoverable runtime artifacts, *without another model router* |
| 010/011 `guide`, CLI UX and opt-in flow trace | Existing help, stdout/stderr, trace, flags and non-TTY contracts | New CLI subcommands with plain, machine-consumable responses |
| 012/013 TUI architecture and `internal/appops` | UI owns display/draft state; backend owns verified facts | Future reuse through typed execution APIs; **no new TUI screen in 016** |
| 014 `internal/fleet` Git-managed profiles/restore and 015 Fleet Manager | User model bindings and installed-file provenance; separate save/apply consent | **Execution** run workspace/checkpoint, not fleet config/provenance or MCP setup |
| Existing `sdd-check`, `internal/harnesscheck` and `internal/harnessevidence` | Deterministic verification/evidence, with experimental native harness boundaries | Reuse useful building blocks if compatible; do not expand experimental runtime admission or misstate supported status |
| Existing review-resolution and one completion self-review | Risk-based review, one batch, human stop | Complete review artifact packaging and severity classification; request-only fresh reviewer |

**Source basis:** Seven attached feature specifications (009–015), the read-only November 9 Superpowers comparison, and subsequent owner decisions. The sources are snapshots with different dates and acceptance states; inspect the current checkout before selecting the implementation batch. This document makes no new claim that missing helpers or runtime hooks have been verified on the owner's machine.

## 2. Product decisions and non-negotiables

| ID | Selected decision | Binding rule |
|---|---|---|
| **D-01** | **Public CLI (Q1=B)** | Ship documented commands for creating/inspecting runs, extracting briefs, capturing check results, producing review packages and cleanly removing a chosen run. All operational logic in reusable Go packages. |
| **D-02** | **Disk checkpoint (Q2=A)** | Store temporary, private, Git-ignored *execution observations* with run identity, source digest and verification evidence; never store a second authoritative approval/batch ledger. Reconcile before resume. |
| **D-03** | **Evidence + severity (Q3=A)** | Produce review packages; classify findings as Critical, Important or Minor, separately from evidentiary status (`valid`, `disputed`, `needs evidence`). Keep current optional independent review and completion-review policy. |
| **D-04** | **Requested per-task reviewer (Q3=B conditionally)** | The user may explicitly request a fresh reviewer for a named task or bounded set; it is **off by default**, not implied by package creation, and blocked if effective permissions/isolation cannot satisfy the intended review. |
| **D-05** | **Architecture** | Root Go 1.23+ standard-library-first, typed service/API shared by CLI and eventual TUI, no root external dependencies unless separately approved, no new autonomous daemon/agent runtime. |
| **D-06** | **Efficiency and validation** | Minimal targeted tests of correctness/safety; no blanket multi-agent benchmarking, full regression-suite expansion or paid live model calls by default. |

## 3. Actors, journeys and boundaries

| Actor | Journey | Expected outcome |
|---|---|---|
| Human developer | Approves feature/batch, selects direct/delegated work and optional review | Retains final control; one batch at a time |
| Main intelligent agent | Creates run, extracts brief, optionally delegates, records truthful evidence, summarizes | Context-efficient handoff and independently verified result |
| Specialist child | Reads only authorized brief/source, returns observations/changes | No power to approve tasks, select paid routes, or invoke next batch |
| CI / CLI automation | Invokes same commands with explicit args and JSON output | Stable exit codes and machine-readable status; no prompts/network inference |
| Future Fleet/TUI consumer | Calls shared typed execution service when separately authorized | No direct file mutation or CLI-output parsing from a screen |

### 3.1 Execution scope

Feature 016 **does not** automatically invoke an OpenCode subagent or decide what model to use. It prepares/records the execution context. The existing Feature 009 orchestration skill decides whether to call any agent, subject to prior human approval and host/runtime permissions.

The CLI's `run check` executes a supplied **local command** only when a human or authorized orchestrator invokes it. It is **not a sandbox**. Tests can have side effects and therefore require the same authorization as direct test execution. A dry-run/preview must never run the command.

## 4. Public CLI contract (proposed)

The following command grammar is the preferred public design. The plan must check existing command collisions and installed CLI parser behavior before freezing exact flags. The command family should be complete and compatible with existing explicit flags, help, non-TTY behavior, exit statuses, and JSON consumers.

```text
# Create a private task-run workspace; no model calls, no user-home changes.
agentic-sdd run init --feature specs/016-... --batch B1 [--repo DIR] [--format text|json]

# Extract an authorized task; artifact is created inside that run.
agentic-sdd task brief --run RUN_ID --task T01 [--format text|json]

# Execute one already-authorized check, persist full stdout/stderr and result.
agentic-sdd run check --run RUN_ID --task T01 -- go test ./internal/example/...

# Package actual relevant Git changes and provenance for a bounded review.
agentic-sdd review package --run RUN_ID --base COMMIT [--head COMMIT] [--format text|json]

# Read observation/checkpoint status; do not claim approval or completion.
agentic-sdd run status --run RUN_ID [--format text|json]

# Attach a result supplied by the caller as a CLAIM (not verified truth).
agentic-sdd run record --run RUN_ID --task T01 --result RESULT_FILE [--format text|json]

# Remove precisely this run's scratch files, with explicit confirmation/flag.
agentic-sdd run clean --run RUN_ID [--yes]
```

**Design clarifications:**

1. `--repo` defaults to the current working tree **only for local read-only inspection**; commands that write or execute must surface the resolved target and require a valid run identity. Never infer a feature approval from a directory name, `RUN_ID`, or a `--yes` flag.
2. `run init` is a local scratch write, not permission to implement; it must be explicit. `task brief`, `run check`, `review package`, and `run record` may write **only their designated private run artifacts** (apart from side effects of an explicitly authorized check command).
3. `--format json` is machine-readable, with stable schema/version, exact evidence paths and `unknown` where needed. Human `text` mode prints compact status plus precise log/artifact pointers. Do not mix decorative colors/progress bars into JSON or piped output.
4. All commands work unattended with explicit arguments. No hidden prompt, model request, authentication check, network call, or auto-install. When user confirmation is required in a non-interactive context, reject unless an existing authorization/explicit confirmation mechanism is satisfied; `--yes` for cleaning only acknowledges deleting that run's scratch, **not** SDD/batch authorization.
5. The CLI must provide `help` for the new groups and keep old commands/exit codes/output unchanged. New commands may use additive JSON fields but should version their own contracts rather than mutating existing fleet/trace schemas.
6. `run record` accepts a child/tool report as **untrusted input**, labels it *reported*, and does not set human approval, verification, or next-batch authority. Prefer a small validated JSON schema over arbitrary YAML execution.
7. `run status` displays *observed* run artifacts, original feature/batch source identities, worktree freshness, and whether the checkpoint is **stale** or `needs reconciliation`. It must never declare a task verified solely from a `record` claim.
8. `run clean` removes only the owned run directory, is idempotent for an already-absent run where safe, and must not follow symlinks or delete unrelated files. No automatic background purging, no destructive global “clean all” command in this feature.

### 4.1 Shared engine/API architecture

```text
cmd/agentic-sdd/
  existing root parser + new public command adapters
        |  parse arguments, format text/JSON, return exit statuses
        v
internal/execution/                  # proposed root standard-library service
  repository.go                      # repo identity, exact safe paths, Git state
  workspace.go                       # private run lifecycle + atomic file writes
  taskbrief.go                       # deterministic task extraction and references
  checks.go                          # command execution + full log capture
  reviewpackage.go                   # committed/staged/unstaged/untracked coverage
  checkpoint.go                      # observed state + freshness/reconciliation
  types.go                           # typed requests/results, versioned contracts
        |
        +--> reuse existing harness/evidence/check helper code where safe
        +--> existing canonical specs/tasks and Git (read-only inspection)
        +--> local private run artifacts only

opencode-pack/skills/agentic-sdd-orchestrate/
  SKILL.md + references/dispatch-contract.md
  references/context-and-cost.md
  references/failure-and-reconciliation.md
       | use public CLI or typed service indirectly; no new authority

experiments/tui/ or 015 Fleet Manager
  NOT MODIFIED by 016; future typed adapter through internal/appops
```

The file names above are illustrative design boundaries rather than an instruction to add one file per bullet. The implementation plan should favor a **small set of cohesive Go packages**, reuse current repo conventions and avoid importing `cmd/agentic-sdd` into UI packages. No duplicate task parser in CLI, TUI and skills. All behavior remains available through the public CLI regardless of whether a TUI is installed.

## 5. Functional requirements

### FR-01 — Source of authority and run identity

A run must be bound to an explicit repository root, feature `spec.md` and `tasks.md` identity where applicable, authorized batch identifier, task ID and a unique collision-resistant run ID. Record Git HEAD and tracked/untracked worktree fingerprint **as observations**, not as authorizations. For direct-fix mode, support a clearly identified direct-scope operation without fabricating feature files. Never silently start another batch after the run ends.

Run initialization must not assert the feature is approved just because `spec.md` contains a string; it may report canonical metadata and unresolved status. The parent agent must still inspect the actual approved scope and human authorization. `tasks.md` remains canonical for selected batch and continuation; execution checkpoints are disposable observations.

### FR-02 — Private, safely isolated scratch workspace

Create a run-specific directory with safe canonical root and normalized IDs. Prefer `.agentic-sdd/runs/<run-id>/` **only when it can be kept Git-ignored without modifying unrelated tracked files**; otherwise choose an explicit appropriate private user-cache location with clear path reporting. Use restrictive permissions (directories `0700`, files `0600` where applicable), reject symlinks and unsafe path traversal, guard against collision and concurrent writers, atomically write/rewrite checkpoint files, and retain exact provenance. Git-ignore alone is not privacy, permission containment or proof of sandboxing.

Do not silently modify existing `.gitignore` or tracked project files simply to create scratch. Establish a tested, supported convention for ignoring run artifacts (e.g. an internal self-ignoring `.gitignore` where supported). Avoid raw credentials, unnecessary environment dumps and automatic remote sync/telemetry. Provide explicit selective cleanup.

### FR-03 — Deterministic task brief extraction

Extract one **exact** task section from the canonical `tasks.md`, retaining its subtask details and necessary specification/acceptance references, without including subsequent unrelated tasks. Support real task IDs used by this repository, not only numeric `Task 1`; avoid substring matches (`T1` vs `T10`), mishandled nested headings, malformed code-fence swallowing, duplicate task IDs and missing tasks. Fail closed when the parser cannot uniquely identify the target or required constraints.

A brief contains task source/revision, authorized scope (as a claim/reference, not fresh authority), dependencies, allowed files/operations, expected checks and stop conditions. The CLI must **not** use an LLM to invent missing requirements. The main agent can add a concise human-readable comment afterward without changing the source contract.

### FR-04 — Tool/check output buffering with truthful results

`run check` executes an explicitly supplied executable/argument vector without implicit shell interpolation (`--` boundary). Capture **complete** stdout and stderr with an accurate process status, signal/cancellation observation where available, start/end timestamp, command/working-directory identity, and private full-log path. Return a compact summary containing salient failures and pointer(s) to the full log. Do not rely only on last three/five/fifteen lines; a failure could occur earlier in output.

Set sensible resource/time/output-storage limits or explicit configuration for huge/long-running checks. Handle zero/nonzero exits, timeouts, cancellation, binary output, missing programs, disk-full failures, and interrupted writes with truthful incomplete/failed state; never silently replace a failing check with success. A passing check is evidence about the executed command, not a feature approval. Never echo secrets from command arguments/environment into summary or JSON by default.

Existing check/evidence tools should be reused where practical; this feature **must not** activate the inactive experimental native ACP harness or change its admission restrictions.

### FR-05 — Complete Git change/review package

`review package` must identify the intended base/revision and reflect the **actual selected scope**: committed changes (where requested), index/staged changes, unstaged changes, plus untracked additions—without silently treating `BASE..HEAD` as a complete dirty-worktree diff. Handle nontext/binary, deleted/renamed, permission changes, submodules, huge files, conflicted indexes, path quoting and missing/unreadable content with explicit statuses.

Generate a review manifest including exact repo/revision identity, bounded change summaries, a complete or explicitly partial diff artifact, file paths and omissions. Prefer access to the minimum reviewed material instead of duplicating an entire repository into a review package. Screen or mask obvious sensitive paths and stop on prohibited inputs; do **not** claim automated redaction is perfect or that handing a reviewer a file provides filesystem isolation. If a complete and safe package cannot be built, fail or label it partial and demand reconciliation before review.

### FR-06 — Execution checkpoint and stale-state reconciliation

Maintain a versioned private checkpoint (`checkpoint.json` or equivalent) with run, feature/batch/task, current Git identity, observed command results, worker-reported status, path/digest references and open blockers. State must distinguish **prepared**, **running**, **observed-partial**, **observed-failed**, **awaiting-verification**, **verified-evidence**, **stale**, and **blocked** without implying human approval.

After compaction, session restart or process interruption, `run status` must re-check repository identity and changed-worktree/target digests, compare canonical task/batch authority, detect contradictory/stale entries and report the next permissible *investigation*, not automatically resume a writer, command, reviewer or paid model. Store a source reference/digest rather than replacing `tasks.md` with the scratch checkpoint. Avoid ambiguous “completed” states; a record of worker success is only a claim.

### FR-07 — Failure classification and bounded correction circuit breaker

The existing `agentic-sdd-orchestrate` failure guidance should distinguish: (a) provider/transport failure; (b) model quality or wrong implementation; (c) test/command failure; (d) partial-write/unknown side effects; (e) permission, billing, or data-policy denial. Capture evidence and stop on structural/authorization failures. Within an authorized batch, the main may diagnose and make a focused correction; default to the existing limited correction passes, then stop for a human decision if unresolved.

**Do not** automatically escalate to a more expensive model or a different provider because code/tests failed, silently rerun partially successful edits, call a blocked reviewer, or select another task/batch. Explicit human request may authorize a different agent/model/task, but it must pass the existing Feature 009 privacy/cost and role gates.

### FR-08 — Severity and evidentiary resolution for reviews

Extend existing review-resolution guidance with **severity** as a separate dimension from **validity/evidence**:

- **Critical:** Security/data-loss/functional failure; blocks acceptance until resolved or explicitly addressed by authorized scope decision.
- **Important:** Material requirement, correctness or robustness gap; address within scope before completion or document an authorized scoped disposition.
- **Minor:** Purely cosmetic/preference/low-impact hygiene; report as deferred/non-blocking **only if** there is no hidden requirement or safety consequence.

Existing validity states remain (`valid`, `disputed`, `needs evidence`); finding must cite file/line or reproducible evidence, affected requirement, and consequence. The parent verifies substantial findings before initiating fixes. Severity categories do **not** allow silently bypassing a spec-mandated check. No mandatory second independent review or extra human approval round.

### FR-09 — Request-only fresh reviewer per delegated task

A human may explicitly request *a fresh reviewer for Task Txx* or selected tasks. That opt-in causes the orchestrator to propose the reviewer role, input package, model/privacy cost and evidence policy; dispatch only after satisfying the relevant existing authorization and runtime security restrictions. **Never** activate a generic native independent-review profile against private source if `IR-06-01` remains applicable and unmitigated. A sanitized export with verified boundary, or a clearly labeled advisory alternative with appropriately limited source access, may be considered only when independently shown safe for that scope.

Request-only fresh review does not create compulsory per-task review, extra model calls, parallel writers, auto-acceptance or a new approval ledger. If the user does not request it, the existing single completion self-review remains the default. The orchestrator may always prepare a review package without dispatching an external reviewer.

### FR-10 — Integration with the existing orchestration skill

Update only necessary `opencode-pack/skills/agentic-sdd-orchestrate/` instructions and references so the main agent knows when to run `task brief`, `run check`, `review package`, `run status`, and when to use direct tools instead. The main should give subagents a short exact task packet and artifact references, never a full conversation dump by default. It must independently inspect consequential child claims and actual diffs/tests before reporting outcomes to the human.

No new skill name, new primary agent, agent-profile installer changes, model routing/fallback chains, changed MCP permissions, additional review role or mandatory delegation procedure is required by Feature 016.

### FR-11 — Public CLI compatibility and future consumers

New command families must be discoverable through `agentic-sdd help` and `--help`, usable by human shell and OpenCode agents, and behave predictably under pipes/CI. Reuse existing CLI environment/diagnostic conventions: plain stdout for data, stderr for errors, no spinners/ANSI under non-TTY, `NO_COLOR` honored, predictable EOF/cancel semantics and deterministic ordering where appropriate. Maintain backwards compatibility with `fleet status|plan|apply|restore|doctor`, `guide`, `preview`, `apply`, `restore`, `logs`, `version`, existing trace event schemas and optional `--with-opencode` presets.

Create **typed root package requests and results** with stable versioned JSON shape; keep command parsing/presentation thin. A later TUI can call the typed API through its supported `internal/appops` seam **only in a separately approved feature**; Feature 016 does not add a TUI page, another application shell, or embed an experimental TUI into the root Go 1.23 module.

### FR-12 — Honest accounting, provenance and security

Capture only actual observable metadata: check execution outcome, run artifact bytes, logged summary, source IDs, optionally model/role reported by the parent/child, and `unknown` for unsupported token/spend fields. A CLI file utility is not automatically capable of inspecting OpenCode live sessions or proving an agent actually used a model. Do not claim saved token percentages, zero duplicate work, automatic runtime containment, cost caps or successful review without supporting evidence.

Run files and review packages may include sensitive source; avoid uploading or external MCP access by default. `run clean` must remove only its own artifacts. Enforce no uncontrolled symlink reads/writes in helper path handling while preserving Feature 006's separate `IR-06-01` blocker; a safe local workspace implementation is **not** general reviewer sandbox certification.

## 6. Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-01** | **Root Go 1.23+**, standard-library-first; no new third-party root dependency without separate approval; nested Go 1.26 TUI untouched. |
| **NFR-02** | All CLI behaviors are exposed through reusable typed services; no business logic split between CLI and skill prose or copied into future TUI. |
| **NFR-03** | No model calls, MCP calls, network requests, background monitoring or credential reads by the execution utility. It must work completely offline. |
| **NFR-04** | Safe path normalization, symlink/path-escape refusal, atomic writes, file-permission restrictions, bounded processing, crash recovery and per-run concurrency control. |
| **NFR-05** | A failed command/report/partial disk write never produces a false `verified` result or hides an earlier failure. |
| **NFR-06** | Existing non-TTY output/exit behavior and current configuration/installation manifests remain unchanged for preexisting commands. |
| **NFR-07** | Versioned run/evidence records with explicitly documented compatibility/cleanup semantics; no automatic Git commit/branch/rebase. |
| **NFR-08** | No surprise profile activation, billed inference, changed permissions, blocked ACP admission bypass or release operation. |

## 7. Acceptance criteria

| ID | Acceptance condition |
|---|---|
| **AC-01** | `agentic-sdd run init`, `task brief`, `run check`, `review package`, `run status`, `run record`, `run clean` exist with coherent help and explicit-argument CI behavior; no collision/regression in existing CLI groups. |
| **AC-02** | All new commands route through tested reusable Go execution services; no TUI/agent instruction contains a second implementation of state or Git/package logic. |
| **AC-03** | A run workspace has correct identity, is private and Git-ignored without unintended tracked changes; collisions, symlinks, path traversal and concurrent writers are handled safely. |
| **AC-04** | Task extraction selects only the requested task, includes necessary scoped requirements, and fails on duplicate/ambiguous IDs and malformed section boundaries. |
| **AC-05** | `run check` records accurate exit status and full logs for passes, failures and interruption; summary reports meaningful errors and gives an exact retrieval path. |
| **AC-06** | Review package reflects committed/staged/unstaged/untracked changes as applicable and accurately identifies omissions/binary/large/secret-risk data; no false complete-diff claim. |
| **AC-07** | Checkpoint survives compaction/CLI restart but becomes `stale` after source/worktree/batch changes and cannot itself authorize resume/next batch. |
| **AC-08** | Failure handling stops at the authorized boundary; no automatic repeated writes, provider escalation, external reviewer, next batch, publish or paid call. |
| **AC-09** | Finding severity (`Critical`, `Important`, `Minor`) and evidence-validity classifications remain distinct and are integrated into current review-resolution wording. |
| **AC-10** | Fresh per-task reviewer is **not dispatched by default**, can be proposed only when specifically requested, and is refused when relevant permission/isolation conditions are not verified. |
| **AC-11** | Existing `agentic-sdd-orchestrate` references describe CLI use, context minimization and truthful parent verification without duplicating approvals or adding a new skill. |
| **AC-12** | Existing CLI commands retain output/exit/backup/fleet/TUI boundary behavior; new text/JSON outputs are valid, versioned, bounded, and usable offline under CI. |

### 7.1 Minimum critical test plan — lean and deterministic

**Use six focused test groups** with fixtures and temporary repository/workspace roots only; no real user home mutation, provider inference or broad expensive validation cycle:

1. **CLI/API compatibility:** command syntax/help/text/JSON; old CLI output unaffected; typed service reuse and no TUI import inversion.
2. **Workspace and resume safety:** new run, colliding IDs, unsafe symlink/path, concurrency, stale Git state, stale batch, safe cleanup and no automatic continuation.
3. **Task section extraction:** real task-ID formats, nested headings/fences, missing/duplicate IDs and precise source references.
4. **Truthful check evidence:** passing/failing/timeout/interrupted command, failure line far from tail, large output and secure full-log retrieval.
5. **Change package accuracy:** committed, staged, unstaged, untracked, binary, renames and sensitive/oversized exclusions; partial data labelled honestly.
6. **Review/authority:** severity/validity matrix; no default extra reviewer, explicit request remains gated; negative tests for failed worker, partial edits and unauthorized escalation.

**Optional documented manual smoke:** One real but harmless local test command and one package generation on a disposable Git fixture. A model-based subagent smoke test requires separate human authorization, known privacy and cost terms, and is **not** a default Feature 016 acceptance requirement.

## 8. Delivery grouping — proposed, not batch authorization

| Proposed batch | Scope | Deliverable/decision gate |
|---|---|---|
| **B1 — Typed execution foundation & public CLI** | Inspect existing helpers and parser; add shared run model/workspace, `run init/status/record/clean`, task brief extraction + `task brief`; CLI help/JSON and core safety tests | Read-only inventory/reconciliation first; demonstrate private workspace and accurate contract before moving to execution tooling |
| **B2 — Evidence, recovery and orchestration integration** | Add `run check` and complete `review package`; faithful output/diff manifests; checkpoint stale reconciliation; severity guidance and request-only reviewer language in existing skill; focused safety tests | Human reviews B1 before separate B2 authorization; demonstrate correctness and explicitly record unresolved runtime isolation limits |

Do not silently expand these batches with a provider monitor, custom plugin, worktree manager, MCP system, Fleet Manager TUI page, public agent executor or new portability framework. If B1 reveals existing deterministic services fully cover a requirement, reuse those rather than building another subsystem.

## 9. Dependencies, exclusions and explicit risks

### Dependencies

- Existing canonical SDD authority and `AGENTS.md` rules; `tasks.md` is the task/batch authority, scratch is not.
- Feature 009 shipped orchestration skill and current review-resolution guidance; verify accepted implementation rather than relying on older v6 draft status.
- Features 010–013 existing CLI conventions and optional TUI architectural boundary.
- Feature 014/015 may still have pending review/gated amendments; Feature 016 does **not** require accepting or completing them and must not conflict with their command grammar.
- Existing harness/check/evidence package APIs and disabled native adapter must be inspected and reused carefully; no claim of enabled native runtime.

### Explicit exclusions

- No new subagent profiles or model/provider routing and **no** agent-dispatch engine.
- No automatic provider failover, paid-token usage, model quality evaluation or quota monitoring.
- No automatic fresh reviewers, mandatory per-task review, duplicate completion self-review or multi-writer concurrency.
- No independent-review sandbox certification or closure of Feature 006 `IR-06-01`.
- No redefinition of Feature 015's MCP install/connection/permission amendment or Fleet Manager TUI UI.
- No worktree creation manager, task-scheduling daemon, watch mode or background automated retention sweeper.
- No changes to Feature 008 release claims, core skill count or existing user installation behavior.
- No release/tag/push/deploy without separate approval.

### Risks and mitigations

| Risk | Mitigation |
|---|---|
| CLI proliferation and unstable grammar | Existing CLI conventions, coherent `task`, `run`, `review` families; freeze through one documented contract and typed engine; no UI rewrites |
| Stale checkpoints mislead main agent | Recheck repo+batch+worktree digest; never auto-resume; stale is a blocking state |
| Sensitive diff/log data leaks | Private workspace, no auto-upload, minimal output, explicit risk exclusions, user-controlled deletion |
| `BASE..HEAD` misses dirty files | Comprehensive staged/unstaged/untracked awareness and manifest of omissions |
| A tool wrapper seems to confer authority | Explicitly separate CLI command execution from feature/batch/model authorization; wrapper is not sandbox |
| False reviewer security claims | Existing Feature 006 block; reviewer source access requires separately verified safe boundary |
| Duplicating harness or fleet functionality | Source audit before implementation; reuse typed packages instead of copying functionality |
| Too many tests/new dependencies | Six targeted groups, fixture-only model-free verification, root standard-library-first |

## 10. Required completion and review report

After each authorized batch, report: exact changes and source paths, new CLI contract, reused versus new packages, unit/fixture checks actually run and actual exits, sample human/JSON output, any privacy/symlink/partial-state findings, old-command regression evidence, untested claims, model calls/spend if separately authorized, the current canonical approved batch and what must be reviewed before continuation. Do not call a batch `accepted` from automated tests or a model verdict alone.

## 11. Approval and decision record

### Owner scope answers (2026-11-12)

- **Q1 = B:** Complete, scriptable public CLI with robust engineering architecture; reusable Go engine and backward-compatible surface.
- **Q2 = A:** Private temporary disk checkpoint/journal with canonical `tasks.md` precedence and state reconciliation.
- **Q3 = A, plus B when requested:** Review packages and severity classification by default; fresh per-task reviewers only upon explicit human request, never automatic, subject to the unresolved safety boundaries.

### Required approvals still pending

- **Combined `spec.md`/`plan.md`/`tasks.md` approval:** **Not recorded**.
- **Feature 016 implementation batch:** **Not authorized**.
- **Real-home CLI use, code-executing checks, provider/model calls:** **Not authorized by this draft**.
- **Native independent-review activation:** **Not authorized** (Feature 006 `IR-06-01` unresolved).
- **Feature 015 MCP amendment:** **Independently pending**, not authorized by this draft.
- **Git push, tag, release, deployment:** **Not authorized**.

## 12. Evidence and research provenance

### User-supplied primary records

- `COMPARATIVE_ORCHESTRATION_ASSESSMENT(1).md` — November 9 comparative assessment: Superpowers task slicing, temporary workspace, diff packages, output capture, compaction survival and fix-loop; claims of exact token savings and “closed security boundary” are **not** treated as measured/proven.
- Feature 009–015 specs supplied November 12; especially 009 `opencode-pack/`, 013's TUI architecture, 014 `internal/fleet` ownership, and 015 `internal/appops`/MCP amendment boundaries.
- User's subsequent decision `1B, 2A, 3A + B only on explicit request`.

### External, advisory engineering references (previous Exa/Context7 checks)

- Superpowers [inline `executing-plans`](https://github.com/obra/superpowers/blob/main/skills/executing-plans/SKILL.md), [subagent-driven development](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/SKILL.md), [task-brief helper](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/scripts/task-brief), [scratch workspace helper](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/scripts/sdd-workspace).
- OpenCode [configuration](https://opencode.ai/docs/config/) and [compaction](https://opencode.ai/docs/compaction/) — tool-output and compaction controls; use native facilities when sufficient.

**Feature principle:** *A public CLI for truthful engineering execution evidence, a small reusable Go core, optional specialist reviews, and durable recovery without transferring human authority to the harness.*
