# Compatibility study — 017 input versus delivered 016

**Date/state:** 2026-11-13; checkout `05c67dff31d9f4a023675bc56b269623961da1d8`. Initial Git status: only the user-supplied untracked `specs/017-agent-context-upgrade/` directory. No production source changed.

## Reconciliation

The input is the November 12 Feature 016 DRAFT r1, not a distinct 017 capability specification. Canonical 016 now has approved spec/plan/tasks and B1–B3 implementation awaiting human review: [spec](../016-agent-execution-context-evidence-recovery/spec.md):3,326–337; [tasks](../016-agent-execution-context-evidence-recovery/tasks.md):10–30.

| Input requirement | Existing implementation / boundary | 017 treatment |
|---|---|---|
| FR-01/02 identity/private runs | `repository.go`, `workspace.go`, `record.go` | Retain; fix checkpoint ownership and brief lock participation. |
| FR-03 exact briefs | `taskbrief.go`, `cmd/agentic-sdd/task.go` | Retain parser; fix persisted provenance/recovery. |
| FR-04 checks | `checks.go`, platform helpers, `run.go` | Already present; regression only. |
| FR-05 review packages | `reviewpackage.go`, `review.go` | Already present; no new package engine. |
| FR-06 recovery | `checkpoint.go` | Retain; close observed integrity gaps. |
| FR-07–10 failure/severity/request-only review/orchestration | Existing portable review references and orchestration pack; 016 tasks T14/T15 | Preserve; no automatic reviewer or new dispatch policy. |
| FR-11 CLI/API | Existing `run`, `task`, `review` adapters, help and schema-v1 types | Preserve frozen 016 grammar, including direct scope and run roots absent from old examples. |
| FR-12 accounting/privacy | Private artifacts and existing execution docs | Preserve; no model/cost or sandbox claim. |

“Present” is not proof of every acceptance criterion. The initial discovery checked feature records; the focused audit below inspected actual source and reproduced selected defects. A complete review-package, process-runner, platform and security audit was not performed. Feature 006 IR-06-01 remains outside scope.

## Confirmed findings

### F-01 — Important: generated briefs are invisible to recovery

`internal/execution/taskbrief.go:138–199` writes/returns the artifact without updating the checkpoint. Existing fields already exist at `types.go:128–135`. `checkpoint.go:389–393` only checks brief paths that were registered, and only for existence, not digest agreement.

**Reproduction:** initialize a fixture feature run; extract T01; inspect checkpoint (`tasks` absent); delete returned `brief_path`; invoke status. **Observed:** exit 0, `state: prepared`, `next_permitted_investigation: none`, no missing-artifact reason. **Required:** registered provenance, then missing/changed artifact reconciliation. Maps to 016 FR-06 and plan §8.

### F-02 — Important: foreign checkpoint identity is accepted on reads

`checkpoint.go:234–257` validates ID syntax, not equality with `run.json`. `StatusRun:301–356` consumes it without comparing IDs. `WriteCheckpoint:270–272` already rejects mismatches, but only when writing.

**Reproduction:** replace checkpoint `run_id` with `run-20261113T000000Z-abcdef`, keeping the valid owning run unchanged; request status for the actual run ID. **Observed:** exit 0, normal status for the actual run, `prepared`, no reconciliation reason. **Required:** conflict before normal status or mutation. Maps to 016 FR-01/06.

### F-03 — Important: brief writes bypass the shared writer lock

`TaskBrief:59–199` never acquires the lock used by check/record/review. `workspace.go:709–767` already provides the shared lock mechanism.

**Reproduction:** install a valid `.lock` with the live fixture Python PID, matching hostname, current UTC timestamp and token; invoke `task brief`. **Observed:** exit 0, artifact created while lock remains. **Required:** conflict and no mutation. Maps to 016 FR-02/NFR-04.

### Additional source-backed edge to test, not a separate reproduced defect

`TaskBrief:60–86` fills only missing request fields from run identity and may continue after a missing identity file; explicit source/run overrides are not compared there. Test the typed-service boundary rather than relying on the CLI's prior identity load (`task.go:60–68`). Treat this as part of F-03's run-backed transaction boundary.

## Evidence and repeatability

Built the unchanged CLI successfully with:

```sh
env GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= \
  go build -o /tmp/opencode/agentic-sdd-017-audit ./cmd/agentic-sdd
```

Three fixture scenarios above ran via Python subprocesses against a disposable repository with isolated HOME, disabled global/system Git config, local fixture author identity and disabled Git commit hooks. Only `/tmp/opencode/` fixture data and binary were written. Fixture source: a committed `specs/fixture/{spec,tasks}.md`, with batch heading `## B1 — Fixture` and task heading `### T01 — First task`; no external requirements needed. Brief/status used explicit JSON output.

Ephemeral diagnostic paths: `/tmp/opencode/sdd017-audit-4r8kch_u/` and `/tmp/opencode/agentic-sdd-017-audit`; they are not durable acceptance evidence. Recreate scenarios in automated tests during B1.

A separate exploratory feature-scope `run check` call without `--task` correctly returned exit 2 and no check execution. No finding is claimed for that case. One exploratory Python printer failed with `AttributeError` before interpreting that response; its corrected invocation confirmed the rejection.

No full Go test/vet suite, race suite or aggregate acceptance gate was run in this drafting session. Existing test names cover parser, basic missing-artifact/state and locks elsewhere, but do not establish these three cases pass. The implementation tasks require regression tests and actual gates.
