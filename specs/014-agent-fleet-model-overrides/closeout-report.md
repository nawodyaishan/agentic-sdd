# Feature 014 — consolidated closeout report (AC-29 candidate)

**Status:** report prepared 2026-11-18. **Owner decision pending.** Every row below
carries the status recorded in [the 022 A5 draft](../022-architecture-remediation/a5-closeout-014-draft.md)
and its `Evidence update (2026-11-17)` section; no status was upgraded, and no test was
re-run while writing this document. Feature 014 is not cleared for release, and its
gates are unchanged by this file.

**Purpose:** give the owner one consolidated AC-01…AC-29 view with an evidence pointer
per row, so the closeout decision can be made in one place instead of across
`spec.md` §12/§13, [`tasks.md`](tasks.md) and the draft's two tables.

## 1. Scope

In scope: the acceptance criteria of [spec §7](spec.md), the single NFR row the draft
dispositions (NFR-02), FR-14, and FR-15. Finding-level rows the draft also carries
(H1–H3, M1–M10, L1–L9, §8 negative-matrix rows 1–20, trusted-base adoption) are **not**
restated here; they stay in [the draft](../022-architecture-remediation/a5-closeout-014-draft.md),
whose status values this report reuses unchanged. Rows the owner must decide are listed
in §4.

Out of scope: prompts, permission design, CLI grammar, code changes. This is a record.

### 1.1 Status vocabulary

Exactly one value per row:

- `observed` — the named test or record exists and the package passes as reported by the
  2026-11-17 runs. It does not mean each row was individually re-traced.
- `not_run` — no run, or no evidence, was produced.
- `blocked` — the requirement cannot be evidenced with the current authority or inputs.
- `owner-pending` — a human decision, not more evidence, is what remains.

No row currently uses `not_run`: the rows the earlier draft marked `not_run` (AC-24,
AC-27, AC-28, L1–L3/L5–L8) were re-evidenced in the 2026-11-17 pass. Two of them carry a
`Partial:` note and keep that qualification.

### 1.2 Historical record used by this report

The 2026-11-11 handover `docs/handovers/2026-11-11_014b.md` is **not** in the working
tree at HEAD: the object survives only at commit `d5eaa7a` and was removed in `4c2b9bd`.
This report did not read that Git object; the facts below are the reproduction in
[the 015 r2 UX document §1.1](../015-fleet-manager-tui/015-fleet-manager-tui-experience-r2-ux-optimized.md),
which attributes them to the handover. Quoted **as historical batch state from
2026-11-11**, not as current evidence:

- Combined spec/plan/tasks decision recorded 2026-11-11; B1–B5 and B6–B9 implemented,
  integration green, awaiting human review. No further batch, merge/release, real-home
  apply or trusted-base adoption recorded.
- `internal/fleet/` written in dependency-free Go 1.23; 18-name resolve/render tests
  reported as passing; 009 `--with-opencode` still installs 14 profiles.
- Plans integrity-bound, source/location handling documented, plan file mode `0600`;
  lock via OS `flock`; exit `2` for validation and `1` for runtime; 18-operation plan
  ≈852 µs and 100-operation plan ≈4.08 ms **in those tests**, not general guarantees.
- Option-A adapters under `opencode-pack/adapters/`; `vision` aliases
  `sdd-visual-inspector`; FR-15 user-driven import deferred.
- The `authenticated` doctor label derived from a CLI dump is an unresolved scope
  choice; authentication otherwise reads as unknown/not verified.
- Final architecture/verification verdict (Opus), trusted-base adoption and a runtime
  release exercise remain open prerequisites.

The handover's `flock` wording matches the code (`internal/fleet/lock_posix.go` uses
`syscall.Flock`), while [plan §2.3](plan.md) still describes `O_CREATE|O_EXCL`. That
text difference is recorded here so it is not re-derived as a contradiction.

## 2. Acceptance criteria and requirement rows

Tests live in `internal/fleet/*_test.go` unless a path is given. Subtest names are quoted
as they appear in the Go source (`go test -run` uses underscores instead of spaces).

| ID | Requirement (short) | Evidence pointer | Status | Note |
|---|---|---|---|---|
| AC-01 | 014 free; 009/013 records untouched | [B5 report](b5-verification-report.md) AC-01 row | observed | Historical record |
| AC-02 | All 18 names represented; no duplicated prompts | `TestFleetResolve`, `TestFleetResolveRealPack`, `TestFleetRender` "real pack render all 14" | observed | `vision` maps to `agent:sdd-visual-inspector` |
| AC-03 | Identical re-render; body and permissions unchanged | `TestFleetRender`, `TestFleetPermissions` | observed | |
| AC-04 | Personal config kept in a separate private repo | [B5 report](b5-verification-report.md); design property | observed | No test by design |
| AC-05 | Invalid config, keys and sources fail clearly | `TestFleetParse`, `TestFleetResolve` | observed | |
| AC-06 | `--with-opencode`, `--opencode-free-first` unchanged | [B5 report](b5-verification-report.md) (6 skillsync tests); §8 row 5 test | observed | Skillsync tests not re-located |
| AC-07 | No new dependencies, prompt tree, ledger or runtime | `go.mod` (0 `require`), `go mod verify` | observed | Re-checked in the 2026-11-17 pass |
| AC-08 | `status`/`plan` read-only and offline | `TestFleetStatusZeroOneAndReadOnly`, `TestFleetDoctorOfflineByDefault` | observed | CLI-level tests not enumerated |
| AC-09 | Plan reports every operation and conflict | `TestFleetPlan`, `TestFleetPlanFiles` | observed | |
| AC-10 | Stale plan refused | `TestB6ExpectedDigestIsCheckedUnderTheGuard`, `TestB6NewerTargetBetweenPlanAndApplyIsRefused` | observed | |
| AC-11 | Unmanaged files and `AGENTS.md` never overwritten | `TestFleetApply`, `TestFleetStatusUnmanagedLegacyAndUnsafe` | observed | No explicit `AGENTS.md` assertion found |
| AC-12 | Backups; interrupted apply recoverable | `TestFleetApplyIntegrity`, `TestB6MetadataFailureAfterSidecarIsIncompleteNotSuccess` | observed | A6 pins the conflicted-transaction message |
| AC-13 | No-op reapply; drift not overwritten | `TestFleetApplyIntegrity`, `TestFleetStatusManagedPendingAndDrifted` | observed | |
| AC-14 | Restore preserves user files; modified target refused | `TestFleetRestore`, `TestFleetRestoreIntegrity` | observed | |
| AC-15 | Format-3 readable; format-4 desired digest; no fabricated clean state | `TestFleetRestore` "fleet-backup listing and legacy skillsync coexistence" | observed | |
| AC-16 | `enabled:false` removes an agent safely | `TestFleetStatusDisabled`, `TestFleetApplyIntegrity` | observed | Native `disabled` field unproven; omit path only |
| AC-17 | Model-only change cannot widen permissions | `TestFleetPermissions` (static only) | blocked | Static half observed; runtime half never exercised — see §4.1 |
| AC-18 | Spec/plan adapters non-writing; IR-06-01 explicit | `TestFleetNegativeMatrixRow12SpecPlanAdapterNonwritingPolicy` | observed | Limitation retained, not fixed |
| AC-19 | Provider/model grammar for the named lanes | `TestFleetDoctorModelRouteGrammar` | observed | |
| AC-20 | `openai/*` not reported as ChatGPT authentication | `TestFleetDoctorOpenAIRouteOnly` | observed | |
| AC-21 | Status separates configured/rendered/installed/enabled/selected/runtime | `TestFleetStatusManagedPendingAndDrifted`, `TestFleetStatusUnmanagedLegacyAndUnsafe`, `TestFleetStatusDisabled` | observed | Runtime availability labelled not checked |
| AC-22 | Orchestrator install does not change the default primary | `TestFleetNegativeMatrixRow18OrchestratorInstallDoesNotChangeDefault` | observed | B5 called it blocked at the time |
| AC-23 | Model change leaves 009 fallback/quota alone | §8 row 5 test; design | observed | No dedicated test |
| AC-24 | New user can follow the walkthrough on a fixture home | [worked guide](fleet-worked-guide.md); preview walkthrough on a `/dev/shm` home; `TestFleetApplyCLI` in `cmd/agentic-sdd/fleet_apply_test.go` | observed | **Partial:** preview only; apply/restore not run on a throwaway home |
| AC-25 | 18-agent and larger scale fixtures; timings recorded | `TestFleetBuildPlanScaleRecordsTimings`, `internal/fleet/scale_015_test.go` | observed | "Designated CI runner" undefined |
| AC-26 | Package registered in Taskfile, CI and harness | `Taskfile.yml`; `harness/checks.json` (10-package list) | observed | |
| AC-27 | Existing installer tests pass; no TUI dependency | `task test-core`, `task vet-core`, fleet and cmd suites | observed | Aggregate `task test` not re-run for this row |
| AC-28 | Release packaging includes templates and metadata | embed via `opencode_pack.go`; `TestFleetResolveRealPack` | observed | **Partial:** no packaged-archive inspection |
| AC-29 | Consolidated final report (this file plus spec §12 items) | this document; [`tasks.md`](tasks.md) T14 | blocked | Candidate report only; row stays blocked — see §4.2 |
| NFR-02 | 18-instance offline pipeline under 2 s, recorded | scale tests (median-of-5) | observed | Recorded, not asserted on a named runner |
| FR-14 | Pinned OpenCode version exercised for effective validation | none | blocked | No binary run — see §4.1 |
| FR-15 | Import/adoption preview, or a recorded deferral | [spec §9, §13](spec.md) | owner-pending | Deferral recorded 2026-11-11; reconfirm at closeout |

No other NFR has a per-item status in the draft, so none is tabled here; where an NFR is
evidenced it is through the AC row that carries it.

### 2.1 Coverage against spec §12

The §12 completion-report items map to this document as follows; gaps stay visible
rather than being filled in by inference.

- §12.1 revision/dirty state, §12.2 the 18-name mapping table, §12.3 manifest schema,
  §12.4 command resolution, §12.5 changed-path list and §12.8 documentation set are
  **not** reproduced here. They live in [spec §2.1](spec.md), [plan §2](plan.md),
  [`tasks.md`](tasks.md) batch records and the [B5 report](b5-verification-report.md).
- §12.6 test list with pass/fail/blocked/not-run is this table plus §4.
- §12.7 open risks and retained limitations is §4.
- §12.9 release recommendation is **not** given: a recommendation needs the Opus verdict
  and the owner decisions in §4.

## 3. What the evidence does and does not show

- The fleet package's own tests and the core task selections pass as reported. That is
  package-level evidence, not an integrated acceptance claim.
- Static frontmatter/permission equality is not evidence of effective runtime
  permissions or model behaviour (spec §7 preamble).
- Every runtime, entitlement and version statement in 014 remains unavailable while
  FR-14 is `blocked`.

## 4. Proposed exceptions for owner decision

Each item is a proposal to record a disposition, not a disposition. Choosing "record as
an exception" is itself an owner decision.

### 4.1 AC-17 runtime half and FR-14 — proposed `blocked` exception

- **What is missing:** no pinned OpenCode executable was exercised, so the runtime half
  of AC-17 and all of FR-14 have no evidence. Static permission preservation is covered
  by `TestFleetPermissions`.
- **Why:** the 2026-11-17 evidence pass reports the local `opencode` as **v2.3.0**, while
  the spec names v2.2.5 (historical) and v2.2.6 (machine snapshot) as candidate versions.
  Real-home inspection and model calls remain not granted under [spec §13](spec.md). That
  evidence report is outside the repository and was not re-run here.
- **Proposal:** record both rows as a `blocked` exception that expires when a pinned
  binary is available; no supported-version, entitlement or effective-permission claim
  is made until then.
- **Alternative:** obtain a pinned binary and run the version-scoped checks first.

### 4.2 AC-29 and this report — proposed `blocked` retention

- **Current state:** the draft records AC-29 `blocked` because no standalone consolidated
  report existed and the 014 handover is missing from HEAD.
- **This file:** answers the "write the report" branch of that blocker. It cannot change
  its own row's status, and spec §12 items listed in §2.1 remain absent.
- **Proposal:** decide whether this document satisfies AC-29 (with the §2.1 gaps
  recorded) or whether the remaining §12 items must be completed first. Until that
  decision, the row stays `blocked`.

### 4.3 014 Opus verdict — absent

- [plan.md](plan.md) requires a final consequential architecture/verification verdict
  from Opus; the 2026-11-17 search found none in the repository or its history.
- **Proposal:** commission that review, or record an explicit owner waiver. A waiver is
  a decision about the gate, not a substitute verdict, and this report cannot supply it.

### 4.4 L9 guide placement — proposed `owner-pending` retention

- The worked guide sits at [`fleet-worked-guide.md`](fleet-worked-guide.md) inside the
  feature record, still titled "draft, B5". The B5 report records the placement as a
  deviation with `docs/` deferred until release readiness.
- `docs/guides/` does not exist at HEAD, so a move would create it.
- **Proposal:** keep it in the feature record while 014 is unreleased, and treat the move
  as a release-readiness task; or direct the move now.
- Related but separate: the guide's §6 limitations must keep matching the `blocked` rows
  above.

### 4.5 FR-15 deferral — recorded 2026-11-11, reconfirm

- The deferral is already recorded as an owner decision in [spec §9 and §13](spec.md);
  the supported path for existing unmanaged profiles is an explicit
  `fleet plan --adopt`, and no import tool is in scope.
- The draft still records FR-15/H3 as `owner-pending` because a closeout record has not
  restated it.
- **Proposal:** copy the 2026-11-11 deferral into the closeout disposition so the row
  stops reading as open, and keep any import preview as separate future scope.

### 4.6 Release/acceptance consent (D-02) — not recoverable

- [spec §13](spec.md) records, dated 2026-11-17, that release consent for this feature is
  **not recoverable** and was not reconstructed from workflow runs or tags; commits in
  `v0.9.0`–`v0.13.0` do not establish it.
- **Proposal:** let that note stand as the record, or supply the original words and date
  to replace it. No consent exists in the repository today.

## 5. What this report is not

- **Not acceptance.** No row is promoted, no gate changes, and 014 is not cleared for
  release or for Fleet live capability.
- **Not approval evidence.** [spec §13](spec.md) is the only record of owner decisions;
  this file adds none. Any decision recorded here must be written there by the owner's
  own action.
- **Not batch state.** [`tasks.md`](tasks.md) owns batches; this report neither advances
  nor closes one.
- **Not a test run.** No command was executed to produce it; anything labelled
  `observed` is carried from the draft's reported runs.
- **Not a runtime, permission, isolation or performance certification.** Static prompt
  validation is not model behaviour, and the recorded timings are design-target
  observations on one machine.
- **Not a release or packaging sign-off.** AC-28 is partial and no artifact was
  inspected.
- **Not the Opus verdict** and not a substitute for human review of the §4 items.

## 6. Next action

Owner decision pending on §4.1–§4.6. Until that decision, Feature 014 keeps the status
already recorded in [spec §13](spec.md), [plan.md](plan.md) and [`tasks.md`](tasks.md),
and no further fleet batch, merge, release, real-home apply or trusted-base adoption
follows from this document.
