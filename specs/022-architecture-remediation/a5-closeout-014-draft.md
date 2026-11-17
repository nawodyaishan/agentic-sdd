# Feature 014 closeout table — DRAFT (022 A5)

**Status:** draft for the owner; nothing here is accepted. Acceptance, exceptions and
blockers are the owner's decisions (D-07). Produced 2026-11-17 by a read-only Sonnet
research pass; the parent then re-ran `go test -count=1 ./internal/fleet ./cmd/agentic-sdd`
(985 passed, working tree after A6 changes) and confirmed `TestFleetExitCodesTable`
(`cmd/agentic-sdd/fleet_test.go:171`) and zero `require` lines in `go.mod`. An
`observed` row means the named test or record exists and the package passes; individual
rows were not re-traced. The B5 disposition report under `/tmp/opencode/022-impl/B5/`
no longer exists. Parent: [plan §13.4](plan.md#134-r4--014-closeout-table-d-07).

Status vocabulary: `observed`, `not_run`, `blocked`, `owner-pending`.
Tests live in `internal/fleet/*_test.go` unless stated.

| ID | Requirement | Evidence | Status | Note |
|---|---|---|---|---|
| AC-01 | 014 directory unoccupied; 009 and 013 untouched | `b5-verification-report.md` AC-01 row | observed | Historical record |
| AC-02 | All 18 names represented, no duplicated prompts | `TestFleetResolve`, `TestFleetResolveRealPack`, `TestFleetRender/real_pack_render_all_14` | observed | `vision` maps to `agent:sdd-visual-inspector` |
| AC-03 | Identical re-render; body and permissions unchanged | `TestFleetRender`, `TestFleetPermissions` | observed | |
| AC-04 | Personal config in a separate private repo | B5 report | observed | Design property, no test |
| AC-05 | Invalid config, keys and sources fail clearly | `TestFleetParse`, `TestFleetResolve` | observed | |
| AC-06 | `--with-opencode`, `--opencode-free-first` unchanged | B5 report (6 skillsync tests), Row 5 test | observed | Skillsync tests not re-located |
| AC-07 | No new dependencies, prompt tree or runtime | `go.mod` has 0 `require` (re-checked) | observed | `go mod verify` not re-run |
| AC-08 | `status`/`plan` read-only, offline | `TestFleetStatusZeroOneAndReadOnly`, `TestFleetDoctorOfflineByDefault` | observed | CLI-level tests not enumerated |
| AC-09 | Plan reports every operation and conflict | `TestFleetPlan`, `TestFleetPlanFiles` | observed | |
| AC-10 | Stale plan refused | `TestB6ExpectedDigestIsCheckedUnderTheGuard`, `TestB6NewerTargetBetweenPlanAndApplyIsRefused` | observed | |
| AC-11 | Unmanaged files and `AGENTS.md` never overwritten | `TestFleetApply`, `TestFleetStatusUnmanagedLegacyAndUnsafe` | observed | No explicit `AGENTS.md` assertion found |
| AC-12 | Backups; interrupted apply recoverable | `TestFleetApplyIntegrity`, `TestB6MetadataFailureAfterSidecarIsIncompleteNotSuccess` | observed | A6 now pins the conflicted-transaction recovery message |
| AC-13 | No-op reapply; drift not overwritten | `TestFleetApplyIntegrity`, `TestFleetStatusManagedPendingAndDrifted` | observed | |
| AC-14 | Restore preserves user files, rejects modified target | `TestFleetRestore`, `TestFleetRestoreIntegrity` | observed | |
| AC-15 | Format-3 readable; format-4 desired digest | `TestFleetRestore/…legacy_skillsync_coexistence` | observed | |
| AC-16 | `enabled:false` removes agent safely | `TestFleetStatusDisabled`, `TestFleetApplyIntegrity` | observed | Native `disabled` unproven |
| AC-17 | Model-only change cannot widen permissions | `TestFleetPermissions` (static) | blocked | Runtime half needs a pinned OpenCode binary, never exercised |
| AC-18 | Spec/plan adapters non-writing; IR-06-01 explicit | `TestFleetNegativeMatrixRow12…` | observed | Limitation retained, not fixed |
| AC-19 | Provider/model grammar | `TestFleetDoctorModelRouteGrammar` | observed | |
| AC-20 | `openai/*` is not ChatGPT auth | `TestFleetDoctorOpenAIRouteOnly` | observed | |
| AC-21 | `status` separates configured/rendered/installed/enabled/selected | `TestFleetStatus*` | observed | Runtime availability labelled not checked |
| AC-22 | Orchestrator install does not change default | `TestFleetNegativeMatrixRow18…` | observed | B5 called it blocked at the time |
| AC-23 | Model change leaves 009 fallback/quota alone | Row 5 test, design | observed | No dedicated test |
| AC-24 | New user can follow the walkthrough | `fleet-worked-guide.md` | not_run | No recorded end-to-end walkthrough |
| AC-25 | 18 and 100-instance timings recorded | `TestFleetBuildPlanScaleRecordsTimings`, `scale_015_test.go` | observed | "Designated CI runner" undefined |
| AC-26 | Package in Taskfile, CI, harness | `Taskfile.yml`; `harness/checks.json` not opened | observed | |
| AC-27 | Existing installer tests pass; no TUI dependency | aggregate `task test` | not_run | Not re-run |
| AC-28 | Release packaging includes templates and metadata | none | not_run | No packaging evidence found |
| AC-29 | Consolidated final report | `tasks.md` T14 | blocked | Report and 014 handover missing |
| NFR-02 | Pipeline under 2 s | scale tests | observed | Recorded, not asserted on a named runner |
| FR-14 | Pinned OpenCode version exercised | none | blocked | No binary exercised |
| FR-15 / H3 | Import preview or deferral | `spec.md` §9, §13 | owner-pending | Deferral recorded 2026-11-11; reconfirm |
| §8 rows 1–20 | Negative matrix | Rows 3, 5, 12, 18, 19 named; 4, 7–10, 13 mapped by inference | observed | Inferred rows unchecked |
| §8 row 17 | Reviewer path escape | none | blocked | IR-06-01 policy |
| H1, H2 | Adapters/all-18 mapping; real-plan timings | as AC-02, AC-25 | observed | |
| M1 | PID lock, atomic reclaim | `TestFleetLockAtomic`, `TestB1FleetLock*` | observed | |
| M2–M5, M9 | Drift at apply; manifest/live pointer; status labels; `restored`; reserved names | `TestFleetApplyIntegrity`, `TestB6*`, `TestFleetStatus*`, `TestFleetRestoreIntegrity`, `TestFleetPlanReservedAndRouteWarnings` | observed | |
| M6 | Plan ID drops config/repo paths | `TestFleetPlan` | observed | Sub-case not confirmed |
| M7 | Size limit, non-regular file rejection | `TestB6ManifestReader*` | observed | Config and plan loaders not individually checked |
| M8 | Exit-code mapping | `TestFleetExitCodesTable` | observed | Confirmed present |
| M10 | Ledger consistency | `tasks.md` | observed | |
| L1–L3, L5–L8 | Hardening (CRLF, symlinks, block scalars, 0700, doctor cap) | `TestFleetPlanFilesSymlinkRefusal` only | not_run | No per-item definitions mapped |
| L4 | Legacy `loadBackup` ignores fleet manifest | legacy-coexistence restore test | observed | |
| L9 | Guide placement | `tasks.md` T14 | owner-pending | |
| Trusted base | Adopt the 10-package `harness/checks.json` | plan §5 | owner-pending | Live adoption, see A2 |
| Consent | 014 release/acceptance consent | `spec.md` §13 | owner-pending | D-02; tag ancestry is not consent |

## Rows the owner must decide

- FR-15/H3 deferral; L9 guide placement; trusted-base adoption; consent (D-02).
- AC-17 runtime half, AC-24, AC-28, AC-29: accept as `blocked`/`not_run` exceptions or require evidence.
- The Opus verdict required by the 014 plan, which does not exist.

## Rows with no evidence found

AC-28, AC-29, AC-17 runtime half, FR-14, AC-24 as an executed walkthrough, L1–L3 and
L5–L8 as individual items, and any 014 closeout handover.
