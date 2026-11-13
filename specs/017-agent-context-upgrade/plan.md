# Feature 017 — Plan

**Revision:** r1, 2026-11-13. Input: [spec.md r1](spec.md).
**Approval:** [spec §Approval](spec.md#approval) — combined approval recorded 2026-11-13; B1 authorized.

## 1. Architecture review and adjustments

**Technical verdict: Needs changes in the existing implementation; proposed residual design ready for combined human review.** This is the main agent's source-backed review, not an independent Opus verdict or human approval.

The supplied draft's CLI-first, typed-service, stdlib architecture fits the current codebase, but its greenfield premise is obsolete. Reuse `internal/execution`, existing command adapters, lock, atomic writer and `TaskState.BriefPath/BriefDigest`. Do not introduce another run store, parser, service package or migration subsystem.

| Decision | Reason / failure addressed |
|---|---|
| Check checkpoint ownership on load, not only write | F-02 must fail before normal status or artifact creation. |
| Make run-backed brief creation one locked service operation | F-03; typed callers require the same protections as CLI. |
| Persist existing brief fields, verify path + digest on status | F-01; private scratch is not inherently trustworthy. |
| Keep schema v1 and diagnose unregistered artifacts | Owner selected conservative compatibility, not automatic migration. |
| One batch; parallel disjoint files after interface freeze | Owner selected this over dependent concurrent batches. |

Alternatives rejected: rebuilding 016 (duplicate scope), trusting filenames as provenance (unverified), silently adopting old briefs (hides drift), changing all schemas/new dependencies (unnecessary), multi-batch concurrency on shared files (integration and authority risk).

## 2. Shared contract and ownership freeze

The parent first confirms the existing types/helpers and writes the bounded task packets. Reuse schema-v1 task fields; avoid editing `types.go` unless the approved requirements cannot be expressed. Any necessary contract change is parent-owned and frozen before parallel dispatch; material expansion returns for human decision.

Proposed package-private helper seam, implemented in a new `briefintegrity.go` by the checkpoint lane:

- `registerTaskBrief(checkpoint *Checkpoint, brief Brief) error`: update only the selected task's brief path/digest; preserve existing state/history; initialize an otherwise absent task to `prepared`. Caller holds the run lock and persists the checkpoint. No disk write or new authority in the helper.
- `inspectBriefArtifacts(runDir string, checkpoint *Checkpoint) ([]string, error)`: bounded, read-only inspection of registered path/digest and unregistered `briefs/` entries. Reuse contained safe reads. Return deterministic reason strings; any unreadable/unsafe/over-limit inspection must lead to reconciliation or an error, never clean status.

Exact helper signatures are implementation details to confirm in T01; the semantic contract above is fixed. The Gemini lane may call the declared helper while the DeepSeek lane implements it; parent integrates only once both compile together. No worker edits the other's files.

## 3. Checkpoint identity and freshness lane

Compare loaded checkpoint ID with validated owning run identity in the common loader, so status, record, checks and review packaging inherit the check. Reject with existing `ErrConflict`. Preserve missing-file semantics, existing validation and write-time equality.

Extend freshness to inspect brief integrity, using existing `missing_artifact` where applicable and additive `brief_digest_mismatch` / `unregistered_brief` reasons. These are blocking stale reasons, not approval states. Reuse `next_permitted_investigation` with an existing investigation-only value; do not add a resume action.

Bound discovery to the dedicated `briefs/` directory: no recursive arbitrary-tree walk, symlink following or unbounded reads. Use existing source-size limits and an explicit documented finite entry budget. On budget exhaustion, fail closed with a specific diagnostic. A zero-brief legacy run remains readable. A live writer must not cause status to claim a clean verified snapshot during a two-file transition; preserve existing lock/running diagnostics and conservatively label transient integrity observations.

## 4. Brief transaction lane

Keep pure `ExtractTask`/reference resolution/rendering usable without run artifacts. For run-backed operations:

1. Validate/load owning identity; reject contradictory explicit request fields and direct scope.
2. Acquire the existing run lock; reload identity/checkpoint under lock, including checkpoint ownership.
3. Resolve contained canonical source paths and validate source identity. Scope-changing source edits require reconciliation; progress-only task edits retain current semantics. Do not use `--force` to bypass changed scope.
4. Extract/render; handle existing content via current idempotence/force contract.
5. Atomically write the brief if needed, register its reference, then atomically persist checkpoint. Report success only after both succeed.
6. Release the lock on every path; preserve prior task/check/claim/blocker state.

Two files cannot be atomically committed with the current helpers. Use artifact-first, checkpoint-second ordering. On checkpoint failure, return error; leave a discoverable unregistered/mismatched artifact, not false success. Status detects it without repair. Do not delete/restore over another actor's data. Retrying explicitly can reconcile byte-identical content under lock; force remains required for differing bytes.

## 5. Parallel dispatch design

After approval and explicit B1 authorization, T01 interface freeze precedes three concurrent workstreams:

| Lane | Implement subagent / requested model | Exclusive write ownership |
|---|---|---|
| A — recovery/identity | `sdd-implement-2`, configured DeepSeek Flash `#max`; catalog confirms `opencode-go/deepseek-v4.1-flash` with `max` | `internal/execution/checkpoint.go`, new `briefintegrity.go`, `briefintegrity_test.go`, new `checkpoint_identity_test.go` |
| B — brief transaction | `sdd-implement-9r-1`, Gemini Flash High; catalog confirms `9router/ag/gemini-3.8-flash-high` | `internal/execution/taskbrief.go`, `internal/execution/taskbrief_test.go`, new `taskbrief_transaction_test.go` |
| C — docs/CLI regressions | `sdd-implement-free-1`, free Console DeepSeek; catalog confirms `opencode/deepseek-v4.1-flash-free` | `docs/execution-runs.md`, new `cmd/agentic-sdd/execution_integrity_test.go` |

Catalog checks establish identifiers, not current endpoint availability or successful inference. Check configured primary and native capability before launch; report the actual model. Use only authorized provider-specific fallback chains from host policy. Never silently substitute paid models for the free lane. Paid DeepSeek/9Router implementation calls are not authorized by drafting. No implementation subagents launched here.

Parent exclusively owns shared types/helper-contract changes, CLI production adjustments if required, feature records, handover, integration and final gates. Workers return bounded diffs, exact commands/exits, blockers and untouched ownership boundaries. Tests may initially fail while another lane's declared helper is pending; do not broaden ownership to fix it.

This is one integrated batch, not three independently accepted batches. No next batch or automatic reviewer dispatch. An independent consequential review, if later requested, must use the authorized review role rather than an implementation-model verdict.

## 6. Guidance, tools and verification

- Principal specialist: installed `golang-pro` for Go/stdlib safety and concurrency. Load in the parent before implementation.
- Installed `test-driven-development` for regression-first work; load when implementing. Existing repository patterns otherwise suffice.
- Docs lane: no additional specialist needed; this is user documentation, not agent-policy editing.
- Installed `agentic-sdd-verification-review` for one completion review; no implicit independent subagent.
- CodeGraph first for indexed source; targeted reads/grep for gaps; patch for edits; shell for isolated Go/Git fixtures and repository gates. Context7 only for a concrete uncertain standard-library/tool API, not this local architecture audit. No browser/external research needed.

Verification: focused API/CLI tests per lane; integrated `go test ./internal/execution ./cmd/agentic-sdd`; `go test -race ./internal/execution`; required `task test`, `task vet`, `git diff --check`. Use sanitized offline environment and temporary HOME/Git config per 016 plan §11; no real-home apply/preview, provider smoke or excluded experimental suites. Do not add packages or alter supported-core gate selection.

Risks: two-file interrupted writes (detect, no false success); legacy ambiguous artifacts (stale, explicit reconciliation); digest-read cost (bounded); shared helper integration (freeze before dispatch); stale test evidence (tie results to integrated code state). New findings beyond this narrow scope require reconciliation, not opportunistic implementation.
