# Feature 014 — B5 verification record (doctor + docs + gates + scale)

**Batch:** B5 (T09 + T10), authorized 2026-11-11 per `spec.md` §13; scope
statement in `tasks.md` B5 ledger row. Produced by the B5 subagent
(`sdd-test-implement-2` lane, model `opencode-go/qwen3.8-flash`) on
2026-11-11. **This record is evidence, not a ledger update**: batch state
`awaiting human review` is written to `tasks.md` by the parent.

**Base commit:** `dde41afc80f5d7f4a1a4cd25b8eac33722d19803` (working tree
carries concurrent B1–B4 subagent changes; B5 evidence below was taken
against the tree state at run time, which included B1 parse, B2 render,
B3 plan/CLI, plus B5 doctor. B4 apply/restore/manifest had not landed as
fleet tests at record time; the skillsync working-tree edits observed
(M `internal/skillsync/{manifest,source,sync}.go`) are B4 work in progress
and are **not** verified by this batch.)

**Environment:** linux/amd64, `go version go1.27.1 linux/amd64`,
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS=`,
all test fixtures under `t.TempDir()` disposable homes. No real user home
touched. No network access attempted. No model/provider call made.

## Changed / created by B5

| Path | Kind |
|---|---|
| `internal/fleet/doctor.go` | new — offline doctor (labels, dump/bin probes, config pre-read) |
| `internal/fleet/doctor_test.go` | new — `TestFleetDoctor*` × 14 |
| `internal/fleet/scale_test.go` | new — 18-name golden + 100-instance fixtures, NFR-02 recorded |
| `cmd/agentic-sdd/fleet_doctor.go` | new — doctor report wiring behind B3's runFleet |
| `cmd/agentic-sdd/fleet.go` | edited — placeholder `runFleetDoctor` tail replaced by one delegation to the B5 wiring (B3 owns the rest of this file) |
| `cmd/agentic-sdd/fleet_doctor_test.go` | new — `TestFleetDoctor*` CLI × 4 |
| `Taskfile.yml` | test/vet argv 9 → 10 packages (+`./internal/fleet`) |
| `harness/checks.json` | required `test`/`vet` argv 9 → 10 packages |
| `tests/supported-core-go.sh` | required-directory loop + selection (+`internal/fleet`) |
| `tests/harness/test_supported_core_go.py`, `test_appops_core_selection.py`, `test_trusted_ci.py`, `test_taskfile.py` | parity lists 9 → 10 (fleet inserted newest-last before CLI packages) |
| `AGENTS.md`, `CONTRIBUTING.md`, `tests/harness/README.md` | maintained count/list guidance 9 → 10 |
| `specs/014-agent-fleet-model-overrides/fleet-worked-guide.md` | new — FR-16 worked guide |
| `specs/014-agent-fleet-model-overrides/b5-verification-report.md` | new — this file |

`go.mod` unchanged, zero `require` lines (AC-07/NFR-04): `grep -c '^require'
go.mod` → `0`; `go mod verify` → `all modules verified`.

## Commands and results (recorded against the tree state above)

| Command | Result |
|---|---|
| `go test -count=1 ./internal/fleet -run TestFleetDoctor` | **PASS** — 15/15 tests, 0 fail |
| `go test -count=1 ./cmd/agentic-sdd -run TestFleetDoctor` | **Pass** — 4/4, 0 fail |
| `task test` (10 packages) | **PASS** all packages ok |
| `task vet` (10 packages) | **PASS** (no diagnostics) |
| `task test-core` → `sh tests/supported-core-go.sh test` | **PASS** (dir checks + selection) |
| `git diff --check` | **PASS** (clean) |
| `python3 -B -m unittest discover -s tests/harness -p test_supported_core_go.py` | **PASS** 8 tests |
| `… -p test_appops_core_selection.py` | **PASS** 9 tests |
| `… -p test_trusted_ci.py` | **PASS** 14 tests |
| `… -p test_taskfile.py` | **PASS** 11 tests |
| `go test -count=1 ./cmd/agentic-sdd ./internal/fleet` (full both) | **PASS** |
| `go test -count=1 ./internal/skillsync -run 'WithOpenCode\|OpenCodeFreeFirst\|TestWith'` (009 AC regression subset) | **PASS** 6 tests |

TDD note: `TestFleetDoctor*` was written first and run red (`undefined:
RunDoctor/DoctorEvidence/...` build failure), then implemented green. The
intermediate red run also surfaced B3's in-flight `plan.go`
(`undefined: isProcessDead`); that resolved when B3 landed their files and
was never edited by B5.

## NFR-02 recorded numbers (median-of-5, `-count=1` run, this machine)

Scope of these rows: fleet.json strict parse pre-read + doctor labeling only.

```
18-name golden offline:           median 4µs   worst 5µs    alloc ~1.6 KB
18-name golden dump-inspected:    median 114µs worst 129µs  alloc ~41 KB
100-instance synthetic offline:   median 9µs   worst 22µs   alloc ~15.7 KB
100-instance synthetic dump-inspected: median 169µs worst 170µs alloc ~63.5 KB
```

**Recorded, not asserted** (per spec §6 "design targets", plan §3). The full
NFR-02 pipeline (parse+resolve+render+plan ≤2 s on the designated CI runner)
is **BLOCKED** for B5 evidence: B1–B3 unit suites exist and pass, but the
end-to-end plan over the 18/100 fixtures belongs to the post-merge
integration run, and B4's apply path was still landing during this batch.

The 18-name fixture encodes the §2.1 reconciliation (5 overlaps `agent:`,
4 Free-first `preset:`, 9 absent names as instances/reuse incl. the R1
conditional adapters) and parses under the strict B1 validator; it shares 13
unique routes across 18 instances, which the scale test asserts.

## §8 negative matrix — B5 status map

| # | Scenario | Status | Evidence / blocked on |
|---|---|---|---|
| 1 | invalid/duplicate JSON, oversize, malformed alias | Partial pass | `TestFleetParse` (B1); B5 pre-read: `TestFleetDoctorModelsFromConfig` (oversize/UTF-8/shape) |
| 2 | policy keys smuggled (`permissions`…) | Pass | strict-unknown-field reject (B1) + CLI case `TestFleetDoctorCLIValidationAndUsage` (exit 2, no echo) |
| 3 | same installed name / alias→alias | Blocked | reject path is B1 resolve + B4 collision; no B5 surface |
| 4 | unmanaged collision | Blocked | B4 apply (not landed at run time) |
| 5 | 009 overlap vs fleet divergence | Partial | 009 regression pass (skillsync subset above); separate-mode override refusal needs B4 |
| 6 | file changed after plan → stale | Partial | `TestFleetPlan`/`Revalidate` (B3); live reject-on-apply blocked on B4 |
| 7 | mid-run fail 3-of-6 rollback | Blocked | B4 |
| 8 | legacy format-3 → no invented desired digest | Blocked | B4 sidecar/adopt |
| 9 | manual drift after apply | Blocked | B4 |
| 10 | disabled managed removal + backup | Blocked | B4 |
| 11 | permission-block change → stale reject | Partial | B2 perm-equality gate (`TestFleetPermissions`); plan-side covered by B3 `TestFleetPlan` |
| 12 | spec/plan adapter weak-prompt block | Blocked | R1 adapter decision/evidence (B2 conditional; not present in tree at run time) |
| 13 | symlink target escape refused | Blocked (fleet) | 009 symlink regression passes (`TestApplyWithOpenCodeSymlinkRejectionAndRollback`); fleet-target variant is B4 |
| 14 | 9Router + OpenAI routes: no auth inference, no network | **Pass** | `TestFleetDoctorOpenAIRouteOnly`, `TestFleetDoctorAuthMarkerRequiredForAuthenticated`, `TestFleetDoctorUnknownMarkerIsNotAuthentication`, `TestFleetDoctorOfflineByDefault` |
| 15 | plan-only: no real-home change/credentials/model | Partial | design holds (`--config` explicit; doctor proven in 14); status/plan proof is B3's isolated-home suites; final run recorded at integration |
| 16 | competing applies → lock | Pass (B3) | `TestFleetLockUnit`, `TestFleetLock` |
| 17 | reviewer path-escape isolation claim | Blocked by policy | IR-06-01 stays explicit; no claim made (guide §6) |
| 18 | orchestrator install ≠ default change | Blocked | B3/B4 status/default observation surface |
| 19 | Git revert of fleet.json still requires apply | Not run | design property (documented in guide §3); behavioral test belongs to integration run |
| 20 | `--with-opencode` unmodified vs 009 fixtures | **Pass** | 6 named skillsync tests + full cmd suite fresh (`-count=1`) |

## AC-01…AC-29 — B5-observable slice

Rows outside B5's evidence stay with their batches. Full disposition belongs
to the consolidated §12 completion report after review.

- **AC-19 Pass** (grammar incl. all lanes offline; `TestFleetDoctorModelRouteGrammar`, B1 tests).
- **AC-20 Pass** (`openai/*` route-only, marker-required authenticated; §8-14 tests above).
- **AC-21 Partial**: doctor labels `configured|catalog-visible|authenticated|unknown|unavailable` + `not checked (offline)` with tool attribution (NFR-05) pass; installed/enabled/selected-primary states belong to `fleet status` (B3) — integration verdict pending.
- **AC-22 Blocked** (no default-agent change observable without B3/B4 status surface).
- **AC-24 Partial**: worked guide delivered (`fleet-worked-guide.md`) covering private-Git setup, 1-agent change, preview/apply/drift/restore, multi-machine reproduce, provider-vs-entitlement, limitations; the end-to-end walkthrough on a fixture home is blocked until apply lands.
- **AC-25 Pass in B5 scope**: 18-name golden fixture + 100-instance synthetic exist offline; timings recorded, not asserted; pipeline-wide timing BLOCKED (see NFR-02 above).
- **AC-26 Pass**: `internal/fleet` registered in Taskfile, harness contract, core script, lefthook-inherited aggregates, and all four parity Python suites updated and green.
- **AC-27 Pass**: aggregate `task test`/`task vet` fresh green including installer/backup/embed tests; no dependency on `experiments/tui`.
- **AC-01 Pass**: 014 directory unoccupied before this batch (R10); 009/013 records untouched by B5.
- **AC-04/06/07 Pass**: private Git file is external and untouched by tests; 009 regression green; `go.mod` dep-free proven.
- **AC-28 Not run** (release packaging of fleet-visible templates/manifest metadata is a post-B4 packaging check).
- **AC-29** this record + parent review; final consolidated report after all batches.

## Deviations & observations (for human review)

1. **Guide placement:** `specs/014-agent-fleet-model-overrides/fleet-worked-guide.md` (feature record) rather than `docs/`, because `docs/` is maintained release guidance and 014 is unreleased; plan T10 permits either location. Move to `docs/` at release-readiness if directed.
2. **No new CLI flag:** B3's frozen grammar (`fleet doctor [--opencode-bin PATH] [--config PATH]`) was kept; the internal `CatalogDump` evidence seam (needed to demonstrate the `authenticated` label honestly) is exercised by unit tests only. Surfacing it via a flag would extend the approved grammar → parent decision.
3. **Edit inside B3's file:** one delegation line at the tail of B3's placeholder `runFleetDoctor` (`cmd/agentic-sdd/fleet.go`) — this is the assigned "fleet doctor wiring stub"; everything else in that file is B3's.
4. **B4 concurrent edits observed** in `internal/skillsync/{manifest,source,sync}.go` while R4 mandates manifest byte-identity; not B5's to judge — flagging for the parent's integration review.
5. **gofmt observations at run time:** B3's `internal/fleet/plan.go` was momentarily gofmt-dirty (their in-flight save); a pre-existing `cmd/agentic-sdd/guide_explorer_test.go` shows under `gofmt -l` on Go 1.27.1 unrelated to this batch (not touched; `task` gates do not run gofmt — Lefthook does at commit).
6. **Trusted-base adoption:** the candidate `harness/checks.json` now differs from the last reviewed base; per `tests/harness` adoption sequence, running `task verify-trusted` against a live reviewed base is a separately authorized operation and was **not** performed here.

## Blocked list (carried forward)

B1–B4 end-to-end integration items: §8 rows 3, 4, 7–10, 12, 13(fleet), 18, 19; AC-22; AC-24 walkthrough; AC-25 pipeline timing on designated CI runner; AC-28 packaging; native OpenCode-version validation (FR-14 — no pinned binary exercised in this batch, all runtime claims `not checked (offline)`); trusted-base contract adoption (live op).

**Next action:** parent records B5 `awaiting human review` in `tasks.md`;
human review of this batch, then B1–B5 integration verification in one tree.
