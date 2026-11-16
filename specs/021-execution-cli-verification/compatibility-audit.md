# Feature 021 — pre-spec compatibility audit

**Status:** read-only audit record; not approval, acceptance or test evidence.
**Method:** four parallel read-only Muse Spark 1.3 subagent audits (016, 017, 018, 019), each checking implementation, tests, cross-feature compatibility, and skill/doc reflection. One 016/019 attempt hit a provider `tool_choice` limitation and was retried with a different read-only agent type on the same model. No files were edited and no mutating commands were run by the auditors.

## Verdicts

| Feature | Verdict | Summary |
|---|---|---|
| 016 execution CLI | **Compatible — minor drift** | All seven command actions exist and route through `internal/execution`; service tests are thorough. Public CLI gaps are the `G1`–`G12` list; the orchestration skill omits `run record`/`run clean` and exact exit semantics. |
| 017 hardening | **Compatible — minor drift** | Foreign-checkpoint refusal, brief registration/digest, lock and legacy-reuse behavior are implemented and tested. The orchestration skill does not reflect the 017 recovery rules. |
| 018 research handoff | **Compatible** | The research workflow, stage pointers, policy precedence and librarian parity are present and statically tested. No stale CLI names or evidence-bypass path. |
| 019 rename/entrypoint | **Substantially executed; honest `not_run`** | Canonical `asdx`/`asdx-core`, wrappers, protocol identity and four-name packaging config are present. Active surfaces use the canonical name; one dated research snapshot still shows `agentic-sdd preview` and is classified historical. Archive/cask membership remains `not_run`. |

## Findings and disposition

| ID | Feature | Severity | Finding | Evidence | 021 disposition |
|---|---|---|---|---|---|
| C-01 | 016 | Minor | CLI-boundary gaps: timeout `124`, failure, output cap, signal, lock conflict, clean refusals, review partial reasons, strict record input, stale status, flag edges | `cmd/agentic-sdd/execution_cli_test.go`, `execution_integrity_test.go`; service-only tests in `checks_test.go`/`reviewpackage_test.go` | Add `G1`–`G12`; no production change |
| C-02 | 016 | Minor | Orchestration skill table lacks `run record`/`run clean`; omits `--`/no-shell, partial `3`, timeout `124`, usage/lock/conflict `2`, status exit `0` for stale/blocked | `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` §4 | `G14` + named skill edits |
| C-03 | 016 | Info | `b3-acceptance.md` records v0.10.0 release evidence but its "not run" list says no tag/release/push | `specs/016-agent-execution-context-evidence-recovery/b3-acceptance.md` | Historical-record inconsistency; do not edit without a 016 decision; note as `not_run`/finding |
| C-04 | 017 | Medium | Skill lacks foreign-checkpoint conflict, brief-drift reasons, byte-identical reissue vs `--force`, record/clean refusals | `references/failure-and-reconciliation.md`, `SKILL.md` | `G14` + named recovery edits |
| C-05 | 017 | Info | Matrix under-claims 017 transaction coverage | `taskbrief_transaction_test.go` | Matrix cites `TestTaskBrief_Transaction_*` cases |
| C-06 | 018 | Info | No incompatibility; research is static/instruction coverage | `skills-files/agentic-sdd-research-spec`, five stage pointers, policy, librarian | Add 018 relationship note; no re-test |
| C-07 | 019 | Low | Dated research snapshot uses `agentic-sdd preview` as current guidance | `docs/research-skills-and-subagents-current-state.md:290,329` | Classify historical in 021; do not rewrite |
| C-08 | 019 | Info | Two binary names during the deprecation window (`task build` → `asdx-core`; `build-darwin`/harness → `agentic-sdd_darwin_*`) | `Taskfile.yml`, `harness/checks.json`, 019 tasks | 021 names which binary each test exercises; no gate change |
| C-09 | 019 | Info | Four-name archive/cask membership and live macOS behavior are `not_run` (goreleaser absent; no live release) | 019 tasks/evidence | Preserve as `not_run`; no archive claim |

## Cross-feature compatibility

| Claim | Post-019 status | Evidence |
|---|---|---|
| 016 commands and flags | Migrated to `asdx-core` in help, usage and `docs/execution-runs.md` | `cmd/agentic-sdd/{main,run,task,review}.go`; `docs/execution-runs.md` |
| 016 run scratch `.agentic-sdd/runs/` | Preserved (019 D-09 state/env freeze) | `docs/execution-runs.md`, `AGENTS.md` |
| 017 help/checkpoint/brief integrity | Migrated; canonical names; wrapper preserves stdout/JSON/exit | 017 help text, integration tests |
| 018 research skills/references | No stale CLI examples; capability-based MCP wording | `skills-files/agentic-sdd-research-spec`, router policy |
| Pre-019 `agentic-sdd run|task|review` in 016/017 specs | Correctly frozen historical examples | 019 D-07; matrix `not_run` note |
| 019 archive/cask/live macOS | `not_run` | 019 tasks/evidence |

## Reflection status

| Surface | Implemented | Tested | Reflected in skills | Consistent |
|---|---|---|---|---|
| 016 execution CLI | Yes | Service: yes; CLI gaps: partial | Partial — `run record`/`run clean`/exit semantics missing | Yes |
| 017 hardening | Yes | Yes | Partial — recovery rules missing | Yes |
| 018 research | Yes | Static only | Yes (research skill, stages, policy, librarian) | Yes |
| 019 rename | Yes (config/code) | Yes where claimed; archive/cask `not_run` | Mostly; one dated research snapshot historical | Yes |

## Not verified

- Live test/gate runs; the auditors were read-only and cited recorded results.
- Real legacy wrapper execution, built-binary `os.Exit`/EPIPE, PTY, ENOSPC, macOS/Windows runtime.
- Archive/cask membership, live Homebrew/macOS quarantine, registry/trademark completion.
- Runtime skill/model compliance, MCP/network behavior, sandbox/reviewer isolation (`IR-06-01`).
- 018's live retrieval/activation; its suite is static by design.

## Skill-gap closure review (DeepSeek)

A read-only DeepSeek study reviewed how to close the skill gaps. Verdict: `READY_WITH_DECISIONS`. Resolved decisions:

- Sandbox assertions cover only affirmative claims (`is a sandbox`, `acts as a sandbox`); the required denial `not a sandbox` stays.
- Exit `2`/`3`/`124` are asserted once each, not per cause.
- The four brief-drift reasons are a subset; the eight-state list and full freshness vocabulary stay in `docs/execution-runs.md`.
- `routing.md` gets a prose-only cross-reference; no table row or backticked model (`internal/appops/routing_test.go` parses the tables).
- `dispatch-contract.md` needs no edit; its handoff fields are `covered-existing`.

The exact draft table rows, exit bullet and recovery block are recorded in `spec.md` §6.2 as implementation guidance, not a second contract.

## Effect on Feature 021

- Confirms the closed `G1`–`G14` scope; no broad re-test is warranted.
- Adds the skill-alignment requirement (`G14`) with named files.
- Explicitly excludes historical 016/017 records, the dated research snapshot, and live archive/cask behavior.
- Does not create approval, batch authorization or acceptance for any feature.
