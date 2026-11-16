# Feature 021 — Execution CLI contract verification

**Revision:** APPROVED r1 (spec), 2026-11-16 — the owner approved `spec.md`; `plan.md` and `tasks.md` are pending combined approval. No implementation authority.
**Parents:** [Feature 016](../016-agent-execution-context-evidence-recovery/spec.md) (execution service and CLI), [Feature 017](../017-agent-context-upgrade/spec.md) (checkpoint/brief integrity), [Feature 019](../019-cli-tui-entrypoint/spec.md) D-03/D-04/D-10 (canonical `asdx-core` CLI).
**Companion:** [test-case-matrix.md](test-case-matrix.md) — closed gap list, exact test names and traceability matrix. [compatibility-audit.md](compatibility-audit.md) records the 016–019 audit findings.
**Status:** draft only. Combined spec/plan/tasks approval and explicit batch authorization are still required.
**Provenance:** drafted after read-only reviews: an architecture/testability review (DeepSeek), a test-technique/codebase-compatibility review (9Router Gemini Hydrogen), four Muse Spark 1.3 compatibility audits (016–019), and a DeepSeek skill-gap-closure study. A Gemini Argon Medium spec review was applied as **provisional** input (non-Opus); per the model-routing policy it does not replace an Opus verdict or human approval. Verdicts are recorded in [the session handover](../../docs/handovers/2026-11-16_021.md) and [compatibility-audit.md](compatibility-audit.md); none is approval.

## 1. Decision and rationale

**Decision:** Close a **named, closed list of public-CLI behavioral gaps** in Feature 016's execution surface — `asdx-core run init|status|check|record|clean`, `asdx-core task brief`, and `asdx-core review package` — as amended by Feature 017 and renamed by Feature 019. Add one traceability matrix mapping the 016/017/019 contract clauses to existing tests, new cases, or explicit `not_run`, and align the skill surface that tells agents how to use the CLI.

**Why this shape:** the repository already has 123 `Test*` functions in `internal/execution` (including one helper-process entry point) and 7 execution CLI tests in `cmd/agentic-sdd`. The public CLI shape is also pinned by dispatch/FlagSet/help inventories in `documentation_reference_test.go` and generated-reference parity. A broad "test everything" feature would duplicate that coverage and become a third contract source. The defensible gap is the **public CLI boundary**:

- timeout exit `124` never appears in `cmd/agentic-sdd/*_test.go`;
- failure, output-limit, signal, and lock-conflict paths are service-only;
- `run clean` refusals, `review package` `head_not_current`/`index_conflict`, strict record input, stale status from HEAD/worktree, and several adapter flags are not exercised at the CLI;
- no single checked-in matrix proves which 016/017 clauses are covered and which are not.

**Core focus:** the four scriptable capabilities 016 delivered and 017 hardened:

1. Preparing **small task briefs** for bounded handoff.
2. Capturing **truthful test evidence** without turning claims into verification.
3. Producing **complete review-change packages** with explicit omissions.
4. **Safely recovering intra-batch execution context** without trusting stale state.

Every gap case serves one of those capabilities or the scriptable CLI contract they depend on. 021 also aligns the skill files that instruct agents to use the CLI correctly. It is not a generic CLI test expansion.

**Not:** a reimplementation of 016, a new command/flag, a rewrite of existing tests, a security or sandbox certification, a reviewer-isolation claim, an acceptance of 016/017/018/019, or a release action.

## 2. Relationship to existing work

| Feature | Relationship |
|---|---|
| 016 | Source of the execution commands, flags, JSON kinds, exit codes and state machine. 021 tests the delivered public contract; it does not reopen 016's approval. |
| 017 | Foreign-checkpoint refusal, brief provenance and conservative schema-v1 reuse are current behavior. Existing 017 CLI tests are referenced; 021 adds only uncovered CLI edges. |
| 018 | Research evidence handoff is instruction/static coverage. Research never runs tests, and a docs citation never substitutes for executed check evidence. 021 references 018 and does not re-test its fixture suite. |
| 019 | Canonical CLI is `asdx-core`; legacy `agentic-sdd` is a link-time wrapper. 021 tests canonical help and CLI behavior; real legacy execution parity is `not_run` because in-process tests never enter the wrapper. Pre-019 examples in frozen 016/017 records and the dated research snapshot are historical, not active guidance. |
| 008 | The supported-core gate stays at 11 packages. 021 adds no package, dependency or Taskfile entry. |

### 2.1 Pre-spec compatibility audit

Four read-only Muse Spark 1.3 audits covered 016, 017, 018 and 019. Verdicts:

- **016:** compatible with minor drift. All seven command actions exist and are service-tested; the public CLI gaps are `G1`–`G12`; the orchestration skill omits `run record`/`run clean` and exact exit semantics.
- **017:** compatible with minor drift. Foreign-checkpoint, brief-registration and lock behavior are implemented and tested; the orchestration skill does not reflect the 017 recovery rules.
- **018:** compatible. Research remains instruction/static coverage; no stale CLI names or evidence-bypass path; no 021 re-test is needed.
- **019:** substantially executed with honest `not_run` boundaries. Active surfaces use `asdx-core`; a dated research snapshot still shows `agentic-sdd preview` and is classified historical; archive/cask membership remains `not_run`.

The findings are recorded in [compatibility-audit.md](compatibility-audit.md). They justify `G14` but create no approval, acceptance or runtime claim.

## 3. Scope

### In scope

- The gap cases `G1`–`G12` in [test-case-matrix.md](test-case-matrix.md), the `G13` matrix existence check and the `G14` skill-alignment check.
- Public CLI assertions: exit code, stdout/stderr destination, JSON `schemaVersion`/`kind`, required fields, ID grammar, ordered reasons, no half-JSON on usage errors.
- 017 integrity edges at the CLI: writer-lock conflict, clean refusals, brief/checkpoint interaction where a new case touches it.
- Adapter wiring that is not otherwise covered: `--direct` conflicts, `--run-root`, feature vs direct `--task` defaults, review `--base`/cap flags.
- **Skill alignment:** update the existing orchestration skill and its references so agents use the verified CLI correctly — exhaustive command coverage, exact exit/partial/status semantics, 017 foreign-checkpoint/brief-drift recovery, and the observation/authority boundary. No new skill and no duplicate state/approval logic.
- A traceability matrix covering 016 FR-01–FR-12 and AC-01–AC-13, 017 AC-01–AC-05, and 019 D-03/D-04/D-10.
- One structural existence test that fails if a matrix row names a missing top-level test or an unknown status, plus one static skill-alignment check.

### Out of scope

- Production Go code changes. A failing case is a finding, not permission to change behavior; fixes need a separate decision.
- New skills, changes to non-orchestration skill authority, or historical skill/doc rewrites. Alignment edits are limited to the named orchestration-skill surface.
- Replacing, deleting or weakening existing 016/017 tests.
- New packages, dependencies, Taskfile entries, gate changes or harness frameworks.
- PTY/TTY testing, static golden JSON, a built-binary process test, or re-testing `os.Exit`.
- Real legacy `agentic-sdd` execution-command parity (link-time wrapper; `not_run`).
- Rewriting historical 016/017 records, the dated `docs/research-skills-and-subagents-current-state.md` snapshot, or their pre-019 examples. The 016 `b3-acceptance.md` release-vs-`not_run` inconsistency is recorded as an informational finding, not edited here.
- Security/sandbox/reviewer isolation, `IR-06-01`, live model/MCP/network, paid inference.
- Real-home apply, backup/restore, release/tag/tap/cask, macOS/Windows runtime claims.
- Performance, fuzz, the TUI, fleet, MCP, installer and documentation features.

## 4. Gap inventory

The new behavior cases and alignment checks are:

| ID | Gap | Expected |
|---|---|---|
| G1 | `run check` timeout | exit 124; `timed_out`; partial state; log paths |
| G2 | `run check` failure | exit 1; `failed`; command exit recorded, not wrapper exit |
| G3 | `run check` output cap | exit 1; `output_limit`; `stdout_truncated`; partial state |
| G4 | `run check` signal/interruption | exit 1; `interrupted`; partial state; lock released (Unix) |
| G5 | CLI writer-lock conflict | second `run check`/`task brief` exits 2; no artifact change |
| G6 | `run clean` refusals/absent run | unsafe/live-lock exit 2, no deletion; absent run exit 0 |
| G7 | `review package` partial reasons | `head_not_current`/`index_conflict` exit 3; no complete claim |
| G8 | `run record` strict input | invalid/oversize exit 2, no artifact; `--result -` success |
| G9 | `run status` stale from HEAD/worktree | exit 0; `stale`; `head_changed`/`worktree_changed`; next investigation |
| G10 | Flag edges | missing `--`, empty vector, negative timeout: exit 2 |
| G11 | Adapter wiring | `--direct` conflict, `--run-root`, feature/direct `--task`, invalid `--base`, cap flags |
| G12 | CLI service edges | `not_started` exit 1; worktree change; `spec_changed` vs `progress_updated`; record blocker |
| G13 | Traceability matrix | every named test exists; 016 FR-01–FR-12, 016 AC-01–AC-13, 017 AC-01–AC-05 and 019 D-03/D-04/D-10 exactly once; valid status |
| G14 | Skill alignment | orchestration skill and references cover all seven command families, exit/partial/status semantics, 017 recovery, and the observation boundary; static check |

Existing coverage is referenced, not repeated: direct-scope refusal, foreign checkpoint, deleted/legacy brief, repeat/force, help/format, sensitive partial, and all typed-service branches. The matrix records each as `covered-existing`.

## 5. Test architecture

- **Location:** a small new suite under `cmd/agentic-sdd/execution_cli_contract_*_test.go` (package `main`). The plan may split the suite across a few disjoint files in the same package to enable parallel lanes; it remains one suite, one package, with no new gate or dependency. Typed-service cases are added only if a gap cannot be reached through the CLI; otherwise existing `internal/execution` tests are referenced.
- **Reuse:** `executionTestRepo`, `runExecutionCLI`, `executionGitOutput`, `decodeRunIdentity`, `initExecutionIntegrityRun`. No second Git repo builder.
- **No new package/dependency/Taskfile entry:** `task test` already selects `cmd/agentic-sdd` and `internal/execution`; the 11-package fence, `harness/checks.json`, `tests/supported-core-go.sh` and its Python parity tests stay unchanged.
- **Hermeticity:** disposable Git repo per case; `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_SYSTEM=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_OPTIONAL_LOCKS=0`, fixed identities, `LC_ALL=C`; temp run root; no real home, network, model or MCP.
- **Timeout case:** use `--timeout 250ms`–`300ms` with `sleep 5`, not a tight 50ms margin. The check timer starts before `cmd.Start()`, so a tight margin can yield `not_started`/exit 1 instead of the required `timed_out`/exit 124 under load or `-race`.
- **Output-cap case:** use a POSIX `sh` builtin loop, and assert `output_limit == true` and `stdout_truncated == true`. `CheckRecord` has no single `truncated` field.
- **Signal case:** gate on `runtime.GOOS != "windows"` (or a `//go:build unix` file) **and** `exec.LookPath("sh")`; use `kill -TERM $$`. A skip is `not_run`, never a pass.
- **Shell lookup:** every shell-invoking case (`G1`–`G6`, `G12` where applicable) resolves `sh` with `exec.LookPath("sh")` and skips as `not_run` if absent; do not hard-code `/bin/sh`.
- **Lock cases:** set Git isolation once on the test goroutine. Start a check that blocks deterministically (`sh -c 'sleep 2'`, or a FIFO gate) in a goroutine returning results over a buffered channel. Poll until `.lock` exists **and** `checkpoint.json` reports state `running`, issue the competing `run check`, `task brief` or `run clean --yes` while the holder is still sleeping, then join the holder. Do not rely on polling `running` alone: `RunCheck` writes the final state before releasing the lock, so the holder must remain alive until the competing call returns. No `t.Fatal` in a worker goroutine; always join before returning. For `run clean`'s live-lock refusal, trigger the lock with a concurrent check rather than forging `.lock`.
- **Stdin case:** call `run(args, &stdout, &stderr, strings.NewReader(payload), false)` directly with Git isolation set; `runExecutionCLI`/`runNonInteractive` hard-code empty stdin.
- **Flag edges:** `run check` without `--`, an empty vector after `--`, and a negative `--timeout` all return exit 2 with a single `error:` line on stderr and empty stdout. They do not print the long usage block. The negative-timeout case must first resolve a valid run.
- **Process boundary:** `runNonInteractive` returns the exact code `main` exits with, so exit assertions are in-process. Real `os.Exit`, `EPIPE`, and built-binary behavior are `not_run`.
- **JSON:** structural assertions only. No golden files: IDs, timestamps and temp paths are not deterministic. Assert `schemaVersion == 1`, exact `kind`, ID grammar, required non-empty fields, ordered reason sets, and no JSON object on stderr-path usage errors.
- **Skill alignment (G14):** read the orchestration skill and its references plus `docs/execution-runs.md` as ground truth. Assert the seven canonical command families, the required denial wording `not a sandbox`, `run record` as a claim, and `stale` guidance; assert the absence of **affirmative** sandbox claims (`is a sandbox`, `acts as a sandbox`), active legacy execution command examples, and **affirmative** approval/resume phrases attached to a run (`run init is permission`, `resume the batch`, `run status approves`). Denial wording (`not a sandbox`, `never resume`, `not approval`) is required or allowed and must not trip the absence check. The documented `"agentic-sdd" wraps "asdx-core"` note is allowed. Assert exit tokens `2`, `3`, `124` once each, not per cause; assert the four brief-drift reasons as a subset, not the full freshness list or state precedence. Do not assert full sentences, section titles, line numbers, flag defaults, link depth or the eight-state list. The check is structural alignment only.
- **Budget:** under ~4s added; no sleep over 2s in the lock case; `harness/checks.json` gives `test` 180s for all 11 packages.

## 6. Traceability matrix

[test-case-matrix.md](test-case-matrix.md) contains:

| Source | Clause | Evidence | Origin | Runtime status |
|---|---|---|---|---|
| 016 | FR-01..FR-12 (one row each) | exact `Test*` name(s) | existing-service / existing-cli / new-cli / static / static-check | covered / covered-static / not_run |
| 016 | AC-01..AC-13 (one row each) | ... | ... | ... |
| 017 | AC-01..AC-05 (one row each) | ... | ... | ... |
| 019 | D-03/D-04/D-10 relevant clauses | ... | existing-cli / static | covered / covered-static / not_run |

- AC-09 (severity) and AC-10 (request-only fresh reviewer) remain `not_run` for runtime behavior. AC-11 (orchestration prose) is `covered-static` through G14; 021 adds no second prose suite.
- Every 016 FR-01–FR-12, 016 AC-01–AC-13, 017 AC-01–AC-05, and 019 D-03/D-04/D-10 clause appears exactly once as a row. Origin cells may combine tokens with `/` (for example `existing-service / existing-cli`, `existing-cli (stub)`); G13 splits on `/` and tolerates the `(stub)` qualifier. Every `new-cli` row names an exact top-level `Test*` function. An unknown status or missing test fails the existence check.
- `G13` is **structural traceability only**: it proves a named test exists, not that the test is non-empty, unskipped, or selected by a particular run. That limit is stated in the matrix.
- The parser scans `./*_test.go`, `../../internal/execution/*_test.go`, and `../../internal/skillsync/*_test.go` with `go/parser`. It resolves `../../specs/021-execution-cli-verification/test-case-matrix.md` relative to the package directory and never writes.

### 6.1 Skill alignment scope

| File | Alignment intent |
|---|---|
| `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` | Add `run record` and `run clean` **table rows** (not just prose); the `run record` row must say **claim**, never verification. Add one §4 bullet for `--`/no-shell, exit `0/1/2/3/124`, and `run status` exit `0` for `stale`/`blocked`. Cite `docs/execution-runs.md` as **backticked path text**; do not add a `../../../docs/` markdown link in `SKILL.md`, because `TestOpenCodePackAssetsAndPermissions` resolves `SKILL.md` links inside `opencode-pack/`/`skills-files/` only. |
| `references/dispatch-contract.md` | Handoff fields (run ID, task ID, brief path + digest, check-log and package paths) are already present; G14 asserts them as `covered-existing`; no edit (resolved by the skill-gap review). |
| `references/failure-and-reconciliation.md` | Add a recovery-section block: foreign-checkpoint conflict (exit `2`, no status JSON, preserved), the four brief-drift reasons as a **subset** → `reconcile_source_change`, byte-identical reissue vs `--force` (never bypasses changed scope), and record/clean refusals. Point to `docs/execution-runs.md`; do not add the eight-state list or full freshness vocabulary. |
| `references/context-and-cost.md` | No execution-CLI change; G14 asserts it does not drift into a second contract. |
| `opencode-pack/routing.md` | One **prose** cross-reference: `run`/`task`/`review` live in `docs/execution-runs.md`; this section covers the installer only. Do not add a table row or backticked model name: `internal/appops/routing_test.go` parses the tables. |
| Do not touch | `agentic-sdd-implement`, `agentic-sdd-verification-review`, `agentic-sdd-debugging`, `workflow-policy.md`, `independent-review.md` authority rules, or historical records. |

### 6.2 Draft skill edits (implementation guidance, not a second contract)

These are the intended shapes; the plan may polish wording but must keep the constraints.

**`SKILL.md` table rows:**

| Command | Use when | What it leaves behind |
|---|---|---|
| `run record` | A child or caller supplies a result you must keep as a **claim**, not verified truth. | `records/<record-id>.json` plus digest; a `passed` claim yields `awaiting-verification`, never acceptance. |
| `run clean` | The run scratch is finished with and must be removed. | Nothing after `--yes`; a preview otherwise. Refuses a live lock or unowned run (exit `2`); an absent run is success. |

**`SKILL.md` §4 bullet (one bullet, not a subsection):**

> `run check` needs `--` and a non-empty command vector: no implicit shell, glob or variable expansion. Exit `0` passed, `1` failed/interrupted/over limit, `124` timed out, `2` usage/lock/conflicting identity, `3` a partial `review package`. `run status` exits `0` even when the reported state is `stale` or `blocked`. Exact flags and reasons are in `docs/execution-runs.md`; cite it as path text, not a markdown link.

**`failure-and-reconciliation.md` recovery block (pointers only):**

- Foreign checkpoint: status and checkpoint-consuming mutations exit `2`, emit no status JSON, change nothing, and preserve the file for investigation.
- Brief drift: `missing_artifact`, `brief_digest_mismatch`, `unregistered_brief`, `brief_inspection_failed` are blocking stale reasons with `reconcile_source_change`. Reissue adopts only byte-identical content; different bytes need `--force`, and `--force` never bypasses a changed source scope.
- Record/clean/brief refusals are state/authorization questions: unowned, symlinked, moved or live-locked runs are refused with exit `2` and nothing deleted. Never hand-edit run files to manufacture a pass.

Do not add JSON schema values, flag defaults, artifact layout, the partial-reason taxonomy, or the eight-state precedence to the skill; those stay in `docs/execution-runs.md` and the CLI help.

G14 is a static content check; it does not prove model compliance or runtime skill activation. FR-10 and AC-11 become `covered-static` through the alignment check; the remaining runtime claims stay `not_run`.

Historical-record findings (the 016 `b3-acceptance.md` release-vs-`not_run` inconsistency and the dated research snapshot's legacy preview lines) are recorded in [compatibility-audit.md](compatibility-audit.md) and are not edited. They do not block alignment.

## 7. Functional requirements

**FR-01 — Small task briefs.** Verify `task brief` selects exactly one authorized task, retains bounded context and source/digest, and obeys 017 registration, lock, foreign-checkpoint and legacy-reuse rules. Reference existing extraction tests; add only the CLI lock case (`G5`); align the orchestration skill's brief guidance (`G14`).

**FR-02 — Truthful test evidence.** Verify `run check` pass/fail/timeout/signal/output-cap (`G1`–`G4`), strict `run record` claims (`G8`), and `run status` claim/stale/blocker states (`G9`, `G12`). A passed claim never becomes verified evidence without a fresh check or parent inspection.

**FR-03 — Complete review packages.** Verify `review package` committed/staged/unstaged/untracked coverage, explicit omissions, `head_not_current`/`index_conflict` partial exit `3` (`G7`), and invalid base/cap flags (`G11`). Never claim complete when content was omitted.

**FR-04 — Safe intra-batch recovery.** Verify `run init`/`run status`/`run clean` and the 017 hardening: writer-lock conflict (`G5`), clean refusals and absent run (`G6`), stale HEAD/worktree (`G9`), spec/tasks vs progress-only changes, and blockers (`G12`); reference the existing foreign-checkpoint and legacy-brief tests. No auto-resume, no approval from a checkpoint.

**FR-05 — Scriptable CLI contract.** Verify exit `0/1/2/3/124` where covered, stdout/stderr separation, JSON `schemaVersion`/`kind`, ID grammar, ordered reasons, no half-JSON on usage errors, and canonical `asdx-core` names (`G10`, `G11`). Real legacy execution and built-binary process behavior are `not_run`.

**FR-06 — Skill alignment.** Update only the named orchestration skill files so agents use the verified CLI correctly: all seven command families, `--`/no-shell, `run record` claim, `run clean` scope, partial/timeout/usage exit codes, 017 foreign-checkpoint/brief-drift recovery, exact handoff fields, and installer/execution cross-reference. No new skill; no duplicate state machine, approval ledger or authority change; do not touch non-orchestration authority rules or historical records.

**FR-07 — Traceability and honest evidence.** `G13` validates the matrix (016 FR/AC, 017 AC, 019 clauses); `G14` verifies skill alignment. Evidence distinguishes `covered`, `covered-static` and `not_run`. G13/G14 are structural only; no acceptance, security, sandbox or live-provider claim.

**FR-08 — Compatibility.** No new package, dependency, Taskfile entry, gate change or production Go behavior change. `documentation_reference_test.go`, generated-reference parity, `tests/harness` expectations and the 11-package selection remain green and untouched. A discovered defect is a finding, not a silent fix.

## 8. Acceptance criteria

- **AC-01:** Existing extraction tests and `G5` prove `task brief` selects exactly one task, records its path/digest, enforces the lock, and refuses a foreign checkpoint or changed scope.
- **AC-02:** `G1`–`G4`, `G8`, `G9` and `G12` prove pass/fail/timeout/signal/output-cap, strict claim ingestion, claim-vs-verified, stale and blocker behavior.
- **AC-03:** `G7` and `G11` prove complete/partial review packages, `head_not_current`/`index_conflict` exit `3`, and invalid base/cap handling; no false complete claim.
- **AC-04:** `G5`, `G6`, `G9` and `G12` prove lock conflict, clean refusals/absent run, stale sources and no auto-resume; existing 017 foreign-checkpoint/legacy-brief cases are referenced.
- **AC-05:** `G10` and `G11` prove exit/stream/JSON exactness and canonical `asdx-core` names; legacy execution and built-binary behavior are `not_run`.
- **AC-06:** The named orchestration skill files are updated as scoped, `G14` passes, no active legacy execution command example remains (the documented wrapper note is allowed), and no new skill, state machine or approval ledger is introduced.
- **AC-07:** `G13` validates the matrix: 016 FR-01–FR-12, AC-01–AC-13, 017 AC-01–AC-05 and 019 D-03/D-04/D-10; every row has an origin, status and evidence or reason.
- **AC-08:** Existing tests are referenced, not duplicated; no 016/017 test is weakened or removed.
- **AC-09:** The 11-package gate, dependencies, Taskfile, `harness/checks.json`, docs/reference parity and harness expectations remain unchanged; `task test`, `task vet`, `go test -race` and `git diff --check` pass.
- **AC-10:** Evidence labels `covered`/`covered-static`/`not_run`, including legacy wrapper execution, real `os.Exit`/EPIPE, ENOSPC, runtime AC-08..AC-10 and non-Linux behavior; 016/017/018/019 approval and batch records remain unchanged.

## 9. Verification and evidence

- `go test ./cmd/agentic-sdd -run 'TestExecutionContract' -count=1`.
- `go test ./internal/execution ./cmd/agentic-sdd -count=1`.
- `task test` and `task vet` (11-package supported core).
- `go test -race ./internal/execution ./cmd/agentic-sdd -count=1` where available.
- `git diff --check`.
- Manual diff review of the named skill files: only the scoped alignment edits, no authority/state duplication.
- Completion report: commands/exits, changed paths, matrix statuses, executed vs `covered-static` vs `not_run`, residual risks, reviewer verdict.

## 10. Risks and exclusions

| Risk | Mitigation |
|---|---|
| Duplicating existing tests | Closed gap list; matrix references existing cases |
| Flaky timeout | 250–300ms margin with `sleep 5`; no tight deadline |
| Flaky lock case | Poll observed `running`; channel results; always join |
| `t.Setenv` in goroutines | Set once; no parallel env mutation |
| Docs/reference parity break | Test and named-skill changes only; no Go production/struct/constant/help changes |
| Gate expansion | No new package/dependency/Taskfile entry |
| Matrix rot | `G13` top-level existence check; structural only |
| Skill prose drift or a second contract | `G14` narrow canonical/boundary assertions; named edits only; manual diff review |
| Skill edit scope creep | Limit to the named orchestration files; no authority/state changes; do not touch non-orchestration skills or historical records |
| Green suite read as acceptance | Evidence labels; human review separate |

**Explicitly not claimed:** security/sandbox certification, reviewer isolation, `IR-06-01` closure, real `os.Exit`/EPIPE, built-binary behavior, legacy `agentic-sdd` execution, PTY/macOS/Windows runtime, ENOSPC, live model/MCP/network, real-home apply, release/packaging, performance or token savings.

## 11. Approval

**Spec approval:** Recorded 2026-11-16. The owner approved `spec.md` r1 and authorized drafting of `plan.md` and `tasks.md`. This is not combined approval and does not authorize implementation.

**Combined spec/plan/tasks approval:** Pending review of `plan.md` and `tasks.md`. No implementation batch, commit, release or real-home operation is authorized until that approval and an explicit batch authorization are recorded here.

## 12. Decisions and defaults

The following defaults are applied in §§3–8. The owner may override them at combined approval; they do not block `plan.md`/`tasks.md` drafting.

1. **Defect handling (resolved-by-default):** report-only; a failing case is a finding with a reproducible case, not a silent fix.
2. **G11/G12 inclusion (resolved-by-default):** keep both small adapter/CLI groups.
3. **Matrix enforcement (resolved-by-default):** one structural `G13` test, not doc-only.
4. **Traceability granularity (resolved-by-default):** FR/AC-level rows plus clause sub-rows only where currently unverified.
5. **G4 signal technique (resolved-by-default):** self-signal `kill -TERM $$`; a FIFO/helper alternative is allowed if it stays deterministic and Linux-first.

**Resolved by the skill-gap review:** `G14` lives in `execution_cli_contract_test.go`; non-orchestration skills stay CLI-free; `dispatch-contract.md` needs no edit; sandbox assertions cover only affirmative forms (the required denial `not a sandbox` stays); exit codes are asserted once each; drift reasons are a four-item subset; `routing.md` gets a prose-only cross-reference.
