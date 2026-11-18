# Feature 022 — Tasks and execution ledger

**Revision:** r1, 2026-11-16. **Parents:** [spec r3](spec.md), [plan r1](plan.md).
**Approval authority:** [spec §9](spec.md#9-approval), currently draft/pending.
**Current state:** combined approval recorded 2026-11-16 (spec §9). Authorized now:
B0 preflight, B3 and B5 only. B1, B2 and B4 wait for the Opus verdict. This ledger is
the authoritative batch record.

**Owner approval 2026-11-17 (plan §12.9):** T11 digest codec approved; TUI lane vendoring approved (re-vendor after T12/T13). Release batch: review, apply, commit and push pending. TUI failures: restore-path test fix done (parent re-ran with a long TMPDIR: pass, vet clean), **awaiting human review**; T19 fixed as a stale test expectation (test-only change in modal_t19_test.go); parent re-ran the full nested TUI package: pass (73.6 s), vet clean. **Awaiting human review.**

**Owner acceptance 2026-11-17:** B1, B2, B3, B4 (T07, T09), B5 (records and
annotations), B7 (T15) and B9 are accepted as batch outcomes. Exceptions: B4's
Python/archive lane is **not** accepted as qualifying evidence while
`test_trusted_ci_container.py` fails on untouched `HEAD`; the nested TUI lane stays
blocked; owner decisions D-01, D-02 and D-07 stay open. Acceptance is not release,
adoption or commit.
Model aliases **H/D/G**, specialist guidance and exact tool/model constraints are
defined once in [plan §9](plan.md#9-agents-specialists-and-conditional-tools).

## 1. Execution and parallel authorization

The owner requested a design for parallel batches, not implementation. After
combined approval, an explicitly authorized wave may launch its named independent
batches together. Otherwise the default is one authorized batch, verify, stop.
No child starts another task outside its brief, changes the ledger/approval,
commits, publishes, switches provider or spawns agents.

Contracts [K1](plan.md#3-shared-contract-k1--mutation-ownership),
[K2](plan.md#4-shared-contract-k2--backup-identity-input-and-provenance),
[K3](plan.md#5-shared-contract-k3--execution-completion),
[K4](plan.md#6-shared-contract-k4--tui-cli-and-policy) and
[K5](plan.md#7-shared-contract-k5--release-and-gate-handoff) are authoritative
shared inputs. Do not restate or vary them in worker-specific briefs.

Each batch returns its result, parent integrates/checks it, then sets that
batch to `awaiting human review`. A wave checkpoint includes every authorized
member, partial/blocked members and the next *permitted*, not automatic, action.
The batch ledger below is the only 022 execution ledger.

## 2. Batch ledger and dependency waves

Current batch states below reflect parent integration and 2026-11-17 owner review;
“Eligible next” assumes explicit authority, dependency evidence, a frozen overlay
and the relevant O decisions/Opus review.

| Batch | Tasks | Priority / outcome | Dependencies and proposed worker(s) | Completion verification | State / next action |
|---|---|---|---|---|---|
| **B0** | T00 | Contract, overlay and authority freeze; no product implementation | Parent; required Opus review separately requested/authorized | Decision/source identities; ownership map; reviewed contracts and scope | frozen at B0 (2026-11-16, [plan §12](plan.md#12-opus-verdict-and-conditions-frozen-at-b0-r2-2026-11-16)); 2026-11-17 owner decisions in [plan §12.7–§12.9](plan.md#127-owner-decisions-recorded-2026-11-17) |
| **B1** | T01–T02 | P0 shared target guard and safe legacy fleet leaf | B0; H | AC-02, producer portion AC-03; skillsync/fleet subprocess exclusion and safe-path tests | **owner accepted 2026-11-17** (lock API approved as built) (parent re-ran `go test -race` for skillsync and fleet, plus vet on linux, darwin and windows: all pass. Lock files live at `<home>/.agentic-sdd/locks/`; B6 must confirm cleanup and restore never delete that directory.) |
| **B2** | T03–T05 | P0 process truthfulness/run ownership; P1 checkpoint ownership | B0; H, sequential tasks in one service-owned lane | AC-01, run portion AC-03/12; execution tests and public CLI regressions | **owner accepted 2026-11-17** (parent re-ran `go test -race ./internal/execution`, `go test ./cmd/agentic-sdd`, and vet on linux, darwin and windows: all pass). Residuals as recorded: a child that leaves the group before the scan is not observed; Darwin has classification only and no pin, so a late signal can reach a reused group id; on Windows every check now fails closed, which is a visible behavior change. Run locks live under `.locks/`, and malformed or unknown legacy `.lock` now blocks; manual removal may be needed. |
| **B3** | T06 | P0 composed confirmation visibility/focus | B0; H | AC-05, composed model tests with dispatch intercepted | **owner accepted 2026-11-17**; residuals as recorded (gap: the gate is in the composed App; direct model calls still dispatch; not checked in a live terminal; `q`, `?` and route jumps are inert during these confirmations). Follow-on TUI fixes (restore-path wrap-aware test fix, T19 stale expectation fix in `modal_t19_test.go`) and nested-module vendoring (~16 MB) are **awaiting human review**; the full nested TUI package passed in the parent's run (73.6 s), vet clean. |
| **B4** | T07–T09 | P0 additive qualification producer; independent P1 env/EOF fixes | B0; D for T07/T09, G for T08 on disjoint files | AC-04 producer, AC-17/18; gate negative fixtures and existing contract unchanged | T07 and T09 **owner accepted 2026-11-17**. T08 Python/archive: the container lane is **not** accepted as qualifying evidence (`test_trusted_ci_container.py` fails on untouched `HEAD` `e4b0d14`). Container overlay fix is **awaiting human review**. The help/probe conflict is **open**: the owner must choose among the options the parent reported (probe checks the experimental gate; reject the flags; or keep strict and block). Nested TUI lane: vendored (~16 MB, offline), **awaiting human review**; must be re-vendored after T12 and T13 land. |
| **B5** | T10 | P0 assurance evidence and decision reconciliation | B0; G; parent owns actual decision recording | AC-09; source-backed record diff and explicit unresolved gates | **owner accepted 2026-11-17** (disposition report at `/tmp/opencode/022-impl/B5/disposition.md`; factual annotations applied to 008, 014, 016, 018, 019, 021); D-01, D-02, D-07 owner decisions remain open. |
| **B6** | T11–T13 | P0 guard consumers, P1 digest/readers/provenance | B1; K1/K2/O-05 freeze; D for pure codec, then H engine + H fleet | AC-03/10/11/12; connected installer/fleet failures and original backups | T11 **owner approved 2026-11-17** (digest codec). T12 **awaiting human review** (MCP legacy lock is `.agentic-sdd/mcp/.lock`, owner option A). T13 (fleet) **in progress** (lock order: fleet's legacy lock, then the agent guard). |
| **B7** | T14–T16 | P0 release privilege, P1 snapshot/observations | B4 additive lane available; K5/O-01 freeze; H workflow owner + G fixed snapshot/observation lanes | AC-06/07/08; workflow negatives, external snapshots, URL/event fixtures | T14 **awaiting human review** (split into `Release build` and `Release publish` via `workflow_run`, pushed in `f0746f7` with `docs(agents)` at `e266ce6` on `origin/main`; `release` environment has owner as required reviewer, prevent-self-review OFF, admin bypass ON, `v*` deployment tags; checks tagged commit ancestry, CI wait up to 40 min, exact manifest, tap cask/no-formula, and `RELEASE_TOOLS_SHA`; own-run exclusion uses `check_suite.id`, unverified against a live GitHub publish run; until `RELEASE_TOOLS_SHA` is set, publish fails closed). T15 **owner accepted 2026-11-17** (parent re-ran B7/Taskfile tests and diff check: pass; `.Date` removed from ldflags; snapshot task never publishes; two-build sha256 identity shown for CLI binaries via `tests/reproducible-build-check.sh`; GoReleaser snapshot/archive `not_run`). T16 **delivered, awaiting human review (2026-11-17/18)**: observation/smoke clauses in `docs.yml` (`53c2162`) and the in-repo route as `docs-observe.yml` (`0b3a3d3`); live behaviour `not_run`; observer hardening after the Opus review landed in `13e60d3`; the re-check found its `created_at` window wrongly refuses real releases (GitHub documents `created_at` as the commit date); fixed afterwards: the window now uses `published_at` only, from one API call, with fixtures that model an old commit date. |
| **B8** | T17–T18 | P1 routing/research authority, P2 proportional context | B0 and explicit owner overlay adoption; G | AC-14/15/22; static rules, links, mandatory-word inventory | **awaiting human review (2026-11-17)** (`0a669d3`; Go DeepSeek resume after a free-quota stop). Static suites, two new B8 fixtures and `go test ./internal/skillsync ./internal/appops` pass. Fixtures not wired into `harness/checks.json`. Models unverified, no model calls. Root AGENTS.md paragraph proposed, not applied. |
| **B9** | T19–T21 | P1 bounded record/help, P2 invocation/wrapper contracts | B2 service interface; D | AC-11/13/19/20; separate-pipe real binaries and bounded inputs | **owner accepted 2026-11-17** (parent re-ran `go test ./cmd/agentic-sdd`, vet and the diff check: pass; child reported scoped race and gofmt clean). Deliberate omissions: the `help <unknown>` usage line still lists the older topic subset; website reference pages wait for B11. Removed unused preview seams; command names, exit codes and stream contracts unchanged. |
| **B10** | T22–T23 | P1 TUI results, P2 seams/startup | B3 consent frozen; B6/B9 relevant structured interfaces available; H | AC-13/19/21; child/root facts and generation-safe initial frame | **awaiting human review (2026-11-17)** (`32b0e10`; Go DeepSeek resume). Nested vet and tests pass (675 in the main package), vendor check ok. PTY launcher lane not run. Open: `internal/appops` drops engine rollback facts on MCP apply failure (root gap, no batch yet). |
| **B11** | T24 | P1 canonical docs/evidence summaries | B5, B6, B7, B8, B9; final B10 impact notes before closeout; G | AC-16 plus docs consumers of other ACs; links/reference/site-policy evidence | **awaiting human review (2026-11-17)** (`264b841`, `4dea573`, `4424b16`; Claude Haiku). Site checks pass on Node 24.21.0, not the pinned 24.22.3; pages stay `unreviewed`; fingerprints refreshed after a content pass, not human review; one built-site focus test fails on untouched HEAD; `real-guides.json` re-baselined (confirm). Not done: D-06, C-09 pack README line, T16 callsite. |
| **B12** | T25 | P2 measured race/timing; final candidate qualification proposal | B1–B11 stopped and integrated; D+G bounded read/test lanes | AC-23, final AC-04/07; cold runs, race, artifact/negative gate evidence | **development evidence recorded 2026-11-17, awaiting human review**, no qualified pass. Race (5 pkgs, 128 s), `task test-core` 50 s, vet-core 8 s, cold GOMAXPROCS=2 test-core 66 s / vet-core 31 s / race 113 s, nested TUI and vendor check all pass at `4424b16`. `harnessevidence`/`harnesscheck`/`buildinfo` race pass (parent). Two clean exports byte-identical on 8 pairs. **Update 2026-11-18:** goreleaser v2.14.0 installed to `~/.local/bin`; `task release` snapshot at `de3bcb2` built one archive with four universal Mach-O binaries, `sha256sum -c` OK, cask rendered, nothing published (AC-07 archive half observed on this host). Run needs a work dir on the repo's filesystem (`/dev/shm` fails with a hardlink error). Still blocked: container lane not run at this HEAD, trusted base undesignated (adoption blocked). Evidence in `/dev/shm/022-B12/`. |
| **B13** | T26 | Whole-feature integration and finding disposition | B12 and all applicable blockers dispositioned; parent | All 23 ACs and 55 findings; unchanged gates and adopted lanes, advisory review limits explicit | **in progress (parent), 2026-11-17**: review verdict `Blocked` on the IDs in the handover `docs/handovers/2026-11-17c_022.md`. Not accepted; no Opus verdict exists. |

W1 = B1/B2/B3/B4/B5. W2 = B6/B7/B8/B9. W3 = B10/B11.
W4 = B12 then B13. The [plan DAG and transfers](plan.md#10-parallel-execution-design)
control concurrency; wave approval does not override a blocked dependency.
Root `AGENTS.md`, 022 canonical docs/ledger and approval sections are parent-only.

## 3. Task briefs

All tasks below are unchecked / not started. New test/helper filenames are
proposed additions, not claims that these files already exist. Existing target
files were located through CodeGraph/config discovery. Refresh actual symbols
before editing; a changed path/scope needs parent agreement.

### B0 — T00: freeze contracts and authorize the selected frontier

- [ ] **T00 — Parent preflight.** Compare current tree to HEAD `e4b0d14`, retain
  the owner's pack/appops/fleet/TUI/website overlay, and record exactly what can
  be shared with workers. Adopt, isolate or block overlapping edits—never reset.
- **Inputs/outputs:** spec O-01–O-08 → accepted values or specific pending
  blockers; freeze K1–K5/API/metadata fields and worker path reservations in the
  plan before consumer launches. Record actual combined approval/wave authority
  only if supplied. Required consequential Opus verdict must be separately
  requested and obtained; Sonnet r1 advice is not a substitute.
- **Check:** `git status --short`, `git diff --name-only`, `git log -5 --oneline`,
  CodeGraph status/explore; static audit of proposed Python selections to ensure
  no excluded runtime validation. No product tests or writes needed for an
  unavailable owner decision. **Owner:** parent; no implement agent assignment.
- **Exit:** remaining unknowns block only affected batches; do not call every
  batch implementation-ready or authorize it by checking this box.

### B1 — T01–T02: guard producer

- [ ] **T01 — H: implement K1 non-execution guard.** Own new
  `internal/skillsync/mutation_lock*.go` and matching tests; include bounded
  regular-opening mechanics in a new package-local file usable by later readers.
  No existing transaction/manifest consumer edits belong to this task.
  Produce the opaque checked handle, home/domain key, refusal semantics and
  Linux/Darwin/non-supported behavior frozen at B0. Kernel-held files are not
  stale leases; release never unlinks a successor. **AC:** producer AC-03.
- [ ] **T02 — H: harden legacy fleet lock leaf.** Own
  `internal/fleet/{plan,lock_posix,lock_windows}.go` and existing/new lock tests,
  consuming T01 mechanics only where the frozen API justifies it. No `BuildPlan`
  rendering/policy rewrite. **AC:** AC-02.
- **Verification:** scoped `go test -race ./internal/skillsync ./internal/fleet`;
  helper processes hold/release descriptors and expose barriers. Sentinel
  referents unchanged for link/FIFO/replacement inputs; independent home targets
  proceed concurrently; unsupported mutation refuses, not silently falls back.
- **Integration:** parent accepts T01 API/tests before B6 consumers. B1 cannot
  close A-03 without the B6 connected writer tests.

### B2 — T03–T05: execution service owner

- [ ] **T03 — H: normal-exit group completion.** Own
  `internal/execution/checks*.go` and tests. Consume K3/O-06; reproduce leader
  exit 0 with surviving same-group child, then repair without changing argv,
  shell semantics, result vocabulary or exit classes. **AC:** AC-01.
  Opus-approved group-identity proof is mandatory; unresolved proof blocks this
  task, not permission to signal a reused PGID.
- [ ] **T04 — H: run lifetime/cleanup guard.** Own
  `internal/execution/workspace.go`, new run-lock OS helpers and related tests.
  Replace age-only reclaim and inside-deleted-directory exclusion with the
  reviewed per-run K1 protocol; revalidate after acquisition. Preserve B9
  fixes/legacy diagnostics without claiming mixed-version coordination.
  **AC:** execution portion AC-03.
- [ ] **T05 — H: checkpoint ownership.** Own
  `internal/execution/{checkpoint,taskbrief,record,reviewpackage}.go` and affected
  tests, with T03/T04 callsite changes owned by this same worker. Public writes
  self-acquire or require a checked opaque handle; private nested functions
  cannot deadlock or accept stale ownership. **AC:** execution portion AC-12.
- **Ordering:** T03/T04 can be designed independently but one service writer
  applies them sequentially; T05 follows their handle/outcome contract.
- **Verification:** `go test -race ./internal/execution ./cmd/agentic-sdd`,
  handshake-driven clean/init/check/record/package races, crash/dead holder and
  ordinary leader exit; no root aggregate selection. Existing timeout=124,
  interrupted=1 and conflict=2 stay truthful. No CLI production edits here;
  return any required CLI delta to B9.

### B3 — T06: composed confirmation

- [ ] **T06 — H: one geometry/focus contract.** Own
  `experiments/tui/{app,modal,mcp,mcp_screens,fleet,fleet_recovery,fleet_write}.go`
  and confirmation/geometry tests. Extend existing root modal routing so hidden
  essential content cannot dispatch; root shortcuts cannot steal confirmation
  focus. Produce K4 confirmation state/messages for B10. **AC:** AC-05.
- **Verification:** in nested module, focused confirmation/composition tests,
  then full nested test/vet. Assert intercepted write messages at 100×16 MCP,
  100×20 Fleet restore, smaller, resize and scroll layouts. Explicit Yes only
  when visible; No/Esc/default Enter behavior preserved; no real OpsHost writes.
- **Boundary:** results/seams/startup remain B10 after this P0 handoff, so P2
  refactoring cannot hold the consent fix hostage.

### B4 — T07–T09: additive verification producer

- [ ] **T07 — D: explicit runner environment.** Own
  `internal/harnesscheck/runner.go`, environment-specific tests and new fixtures.
  Preserve approved overrides, strip hostile Go selection controls and document
  the frozen environment projection for T08. Do not change the full inherited
  environment contract of user `run check`. **AC:** environment part AC-17.
- [ ] **T08 — G: reviewed additive qualification and staging.** Own
  `Taskfile.yml`, `tests/harness/trusted_ci.py`, `.github/workflows/ci.yml`,
  relevant supported-core/parity/trusted-launcher Python tests and a new fixed
  qualification selector/contract adjacent to existing `harness/checks.json`.
  **Do not modify the old required contract.** Select the exact audited suites
  in plan §8; generated directories are excluded by reviewed staging rules, not
  arbitrary candidate ignores. Produce a bounded command/result contract for
  B7 and B12; add negative failing-lane fixtures. **AC:** producer AC-04, staging
  and entrypoint AC-17. Existing core `test`/`vet` sanitization is retained, not
  counted as a new fix; verify/tidy/build/bootstrap gaps are made explicit.
- [ ] **T09 — D: strict result JSON.** Own
  `internal/harnessevidence/validate.go`, validation tests and malformed-tail/
  duplicate-key fixtures. Consume the existing schema; require EOF after the
  first document. **AC:** AC-18. No adapter tests or schema/approval expansion.
- **Parallel split:** T07/T09 share D sequentially or in separate bounded D
  sessions on disjoint packages; T08 owns all Python/Taskfile/CI edits.
- **Verification:** named Python suites only, explicit core harness Go tests,
  hostile-control inputs, exact argv/selector/staging negatives and old-gate
  anti-weakening fixtures. Missing Docker/tools block affected observations;
  configuration tests do not claim a container pass or adoption.

### B5 — T10: assurance records

- [ ] **T10 — G: source-backed disposition.** Own targeted evidence/task text
  under `specs/008-core-release-quality/`, `014-agent-fleet-model-overrides/`,
  `016-agent-execution-context-evidence-recovery/` and
  `021-execution-cli-verification/`. No 022 state writes or implicit acceptance.
  Produce current integrated 014 AC/finding evidence using explicit scoped
  fleet checks, or an honest blocker; date old `not_run` claims and point to
  actual release decisions. **AC:** AC-09.
- **Decision boundary:** draft changes to historical approval sections are
  returned to the parent, not applied by the child. Parent records only actual
  supplied decisions; missing provenance remains missing. G-008-01 stays open
  absent owner designation/adoption. No excluded experiments or Fleet-live work.
- **Verification:** compare cited source sections/commits; explicit
  `go test ./internal/fleet` where new integrated evidence is needed; Markdown
  links and source attribution. Engineering verification is not feature acceptance.
- **Output:** frozen record changes plus remaining owner decisions for B11
  summary consumers; no duplicate global status ledger.

### B6 — T11–T13: guard, identity and transaction consumers

- [ ] **T11 — D: pure framed digest codec.** Own new
  `internal/skillsync/tree_digest_v2.go` and tests/old-tree fixtures only.
  Implement the exact K2 stream and exports; do not edit `manifest.go`, restore
  dispatch or any legacy function. **AC:** codec part AC-10.
- [ ] **T12 — H: skillsync consumers/before-state/readers.** After T11 and B1,
  own `internal/skillsync/{sync,agents,restore,manifest,identity,mcp_transaction}.go`
  and related tests. Acquire the domain guard before authoritative
  revalidation/backup; guarded nested APIs do not re-lock. Add per-entry versions,
  preserve old readers and source identity; bound manifest reads; fix MCP
  age-only reclaim without merging its destination with global profiles.
  **AC:** skillsync portion AC-03/10/11/12.
- [x] **T13 — H: fleet consumer/provenance.** (awaiting human review; see §7) After T11/B1 and the frozen
  skillsync guarded API, own `internal/fleet/{apply,restore,manifest,plan}.go` and
  provenance/reader/transaction tests. Hold K1 through finalization, retain raw
  file digest semantics, atomically finalize/reconcile metadata and never
  overwrite newer targets. **AC:** fleet portion AC-03/10/11/12.
- **Internal schedule:** T11 first; T12 and T13 may run in parallel on the frozen
  API with separate workspaces/compile fixtures, then parent integrates them.
  If that API changed since B0, stop consumers until refrozen—no guessed stubs.
- **Verification:** `go test -race ./internal/skillsync ./internal/fleet`;
  ambiguous-tree fixture, legacy format-1/2/3 and fleet format-4 backup fixtures,
  mixed/unknown digest versions, reader type/limit cases, sidecar/pointer failure
  injection and two-process installer/fleet exclusion/rollback interleavings.
  A structured “files changed, metadata incomplete” result cannot be called
  successful rollback. Read-only status cannot silently repair metadata.

### B7 — T14–T16: release configuration and offline consumers

- [ ] **T14 — H: publisher privilege boundary.** Own
  `.github/workflows/release.yml` and dedicated release-permission/failure/
  artifact-handoff fixtures. Consume T08's fixed qualification contract, preserve
  already-pinned actions and pin the remaining mutable actions to verified
  full SHAs. Separate verifier and publisher credentials, serialize publishing,
  bound jobs and bind artifacts/source. **AC:** AC-06.
- [ ] **T15 — G: reproducible snapshot policy.** Own `.goreleaser.yml`,
  `Taskfile.experimental.yml`, snapshot-specific helper/config tests and, after
  B4 ownership transfer, only snapshot clauses of `Taskfile.yml`. Consume K5
  artifact identity; remove runtime date injection, refuse dirty reproducible
  snapshots, place outputs externally and consolidate entrypoint guarantees.
  **AC:** AC-07.
- [x] **T16 — G: explicit observation and URLs.** (awaiting human review; see §2 and §8) Own the observation/smoke
  clauses of `.github/workflows/docs.yml` and dedicated event/URL test fixtures.
  Consume T14's permission/trigger contract. Return any release-YAML callsite
  patch to H; do not edit T14's file concurrently. **AC:** AC-08.
- **Scheduling:** T15 and T14 may develop against frozen K5; T16 starts only
  after T14's trigger/permission handoff. H integrates its callsite and reviews
  all workflow permissions together. G returns recovery-manual content to B11.
- **Verification:** offline workflow negatives, suppressed-token-event route,
  relative/root asset paths, failure-before-publication, same-source/artifact
  validation and partial-release walkthrough. Two authorized nonpublishing clean
  exports test actual byte equality with installed pinned tools; absent tool/
  artifact-reuse capability is a blocker, not permission for live trial publish.

### B8 — T17–T18: authority and context

- [ ] **T17 — G: upstream inventory and advisory profiles.** Own
  `opencode-pack/{routing,README}.md`, affected pack agent/adapter frontmatter
  and policy references, plus their dedicated metadata fixtures. First reconcile
  the inherited owner overlay and O-02/O-03; preserve unadopted changes. C-09
  factual writer inventory may proceed independently. Do not edit appops/TUI
  routing tests over another session's work; return required projections to the
  parent for scoped handoff. **AC:** AC-14.
- [ ] **T18 — G: portable mode/research/context rules.** Own affected
  `skills-files/agentic-sdd-router/references/`, research-spec and lightweight
  teach/wait-what/handoff entries, and pack scoped-re-review wording after T17.
  Read-only discovery cannot bootstrap; research trigger/body agree; conditional
  references reduce mandatory word load with prohibitions preserved.
  **AC:** AC-15/22.
- **Verification:** both selected static suites, installed-link closure,
  inventory/profile projection checks and read-only scenario fixtures on an
  unindexed disposable tree. Word counts include transitive mandatory
  references; they are not runtime token/compliance evidence.
- **Boundary:** root `AGENTS.md` changes are parent proposals, not child writes.
  No global host edits, agent model calls or excluded independent-review tests.

### B9 — T19–T21: CLI contracts

- [ ] **T19 — D: bounded record file.** Own
  `cmd/agentic-sdd/execution_cli.go` and record-file reader tests. Consume
  K2's exact record limit and existing service ingestion contract. Support
  regular FILE/`-` stdin, reject special/oversized/replaced inputs safely.
  **AC:** record portion AC-11.
- [ ] **T20 — D: truthful help and explicit invocation dependencies.** Own
  `cmd/agentic-sdd/{main,helptext,mcp,mcp_write,tui}.go` and affected help/MCP/
  launcher tests, plus `run.go` only for an agreed B2 caller delta. Preserve
  command names/exit/stream contracts and current help-order exceptions. Remove
  ambient substitution in the touched invocation paths; expose actual write
  surfaces/pack flags. **AC:** CLI portions AC-13/19.
- [ ] **T21 — D: built wrapper evidence.** Own new separate-pipe binary tests
  alongside existing CLI launcher tests; no launcher lifecycle rewrite. Use
  actual link-time legacy builds, fixture companions and nonzero exits.
  **AC:** AC-20.
- **Ordering:** tasks are sequential in one CLI owner lane (shared test helpers/
  dispatch). No website or generated-reference edits; send impact notes to B11.
- **Verification:** `go test ./cmd/agentic-sdd`, scoped race where supported,
  record special-file/limit+1 cases, exact help ordering and separate streams;
  test binaries/cache/HOME under external disposable paths. Nested PTY evidence
  is complementary and not silently claimed if not run.

### B10 — T22–T23: TUI consumers and bounded refactoring

- [ ] **T22 — H: truthful outcome projection.** Own
  `experiments/tui/{mcp,mcp_adapter,ops_host}.go` and associated tests after B3.
  Preserve structured partial/rollback/log facts on errors; one projection for
  root/child, no invented backup on no-op. **AC:** TUI portion AC-13.
- [ ] **T23 — H: explicit instance seams and initial frame.** Own remaining
  touched MCP instance-state code, `experiments/tui/{app,activity}.go`,
  `experiments/tui/cmd/internal/tuirun/run.go` and startup/generation tests.
  Consume B3 consent/T22 results; no process-global registry/lock. Delay initial
  readers behind cancellable/generation-bound commands while rendering useful
  loading/empty content. **AC:** TUI portion AC-19, AC-21.
- **Verification:** nested model tests block a reader on a channel, observe
  first frame, change home/route and release stale result; assert stale suppression
  and errors. Run full nested tests/vet/module integrity; timing is recorded,
  not a short wall-clock assertion. Preserve integrated consent regressions.
- **Priority:** T22 correctness precedes T23 cleanup; do not reopen B3 merely to
  reorganize code. Return final doc impact to B11.

### B11 — T24: one documentation/reference owner

- [ ] **T24 — G: reconcile canonical navigation/records/reference.** Own
  README, docs index/migration stubs/dated-study labels, cited record summaries/
  links and required website pages/reference generators/freshness metadata.
  Consume T10 decisions/evidence and all producer impact notes; T17 profile
  inventory is an input, not host truth. Fix D-09 attribution here (FR/AC-16),
  not in environment code. Fold handovers only with preserved canonical/history
  pointers; keep 022's session handover parent-owned.
- **AC:** AC-16 and documentation consumers of AC-07/08/10–15/20/21.
  Update generated blocks only from a candidate-bound binary after B6/B7/B9
  interfaces freeze; preserve `unreviewed` metadata where applicable. Later TUI
  prose consumes B10 final behavior before closure.
- **Verification:** local files/headings/history links; selected static suites;
  applicable website source/reference/build/export and docs CI-policy checks
  with the reviewed toolchain. Node/version/container gaps remain `not_run`,
  never silently weakened. No deployment or change of historical approval.

### B12 — T25: measurements and same-candidate qualification

- [ ] **T25 — D+G: bounded verification lanes.** Stop writers, freeze the
  integrated tree and run plan §8 selections. D owns race/Go measurements; G
  owns cold-run/selector/archive evidence inspection on separate disposable
  outputs. No concurrent edits to production files or the candidate.
- **Owned changes:** only measured budget/selection proposals in the new
  additive qualification files/fixtures transferred from B4, if approved;
  never modify the old trusted contract or add green retries. Parent applies
  any agreed budget diff and reruns affected checks against the resulting state.
- **AC:** AC-23 and final evidence for AC-04/07. Compare two clean package
  exports, inspect all four applications/slices/checksums, run both module
  integrity checks and scoped race selections. Record two-CPU cold-cache wall
  times/ceilings separately from semantics. Missing native/Docker/snapshot
  prerequisites block only affected claims.
- **Adoption:** a separately designated trusted base may run the new lane only
  under actual authority. Without it report development evidence plus
  `adoption blocked`, not a qualified independent pass.

### B13 — T26: serial final integration and completion review

- [ ] **T26 — Parent: reconcile whole candidate.** Inspect every diff against
  adopted scope and owner work; preserve actual models/failures/partial states.
  Re-run unchanged required core gates, affected static/site checks and the
  appropriately adopted/additive qualification lanes. Each result is tied to
  the final candidate, not a mixture of worker snapshots.
- **Acceptance:** verify every AC in §5 and disposition every source finding
  through spec §6; preserve missing owner designation, acceptance or release
  observations explicitly. Check connected CLI→service→metadata and
  UI→adapter→engine failures, not only task checkboxes.
- **Review:** obtain the required consequential architecture verdict before
  treating affected designs ready; any final supplementary cross-model code
  inspection may use the requested H/D/G lanes with read-only briefs and a model
  different from the main implementer where practical. It is not Opus, human
  acceptance, a rerun of tests, or sandbox certification.
- **Exit:** report `Ready`/`Needs changes`/`Blocked` with remaining IDs; record
  completed/incomplete/blocked separately; put delivered batches at `awaiting
  human review`; write/update the session handover. No commit, push, tag,
  release, deployment, real-home apply or next batch inferred.

## 4. Acceptance producer/consumer reconciliation

This maps each AC to implementation and connected verification owners. Tasks
reference their contracts above; the spec remains the only requirements owner.

| AC | Implementation owner(s) | Connected verification owner |
|---|---|---|
| AC-01 | T03 | T03 + T26 public check evidence |
| AC-02 | T01–T02 | T02 sentinel/subprocess cases |
| AC-03 | T01, T04, T12–T13 | T12/T13 two-writer races; T26 clean/run integration |
| AC-04 | T08 | T25, T26; actual adoption remains owner decision |
| AC-05 | T06 | T06 composed app; T22/T23 preserve it |
| AC-06 | T14 | T14 offline privilege/artifact tests; T26 |
| AC-07 | T15 | T25 same-input package equality |
| AC-08 | T16 | T16 event/URL fixtures; T26 |
| AC-09 | T10 | Parent sources/decisions in T26; no invented closure |
| AC-10 | T11–T13 | T12/T13 original backup/sidecar fixtures; T26 |
| AC-11 | T12–T13, T19 | Each reader's negative fixtures; T26 CLI boundary |
| AC-12 | T05, T12–T13 | Transaction fault/ownership tests; T26 |
| AC-13 | T20, T22 | T22 root/child structured facts + T24 help parity |
| AC-14 | T17 | T17 projections; parent overlay/decision check |
| AC-15 | T18 | T18 read-only scenario and static instructions |
| AC-16 | T10, T24 | T24 links/source attribution; parent decision check |
| AC-17 | T07–T08 | T07 hostile env/T08 staging negatives; T25 |
| AC-18 | T09 | T09 malformed/duplicate/tail regressions |
| AC-19 | T20, T23 | Independent invocation/model cases; T26 |
| AC-20 | T21 | Built separate-pipe fixture processes |
| AC-21 | T23 | Stalled-source/stale-generation fixture |
| AC-22 | T18 | Transitive mandatory-word inventory + authority assertions |
| AC-23 | T25 | Final scoped measurements/race and budget evidence |

All 55 source IDs still map once in spec §6. Priority totals remain P0 15 /
P1 34 / P2 6. A task may implement multiple findings, but it cannot erase their
individual evidence/dispositions in the final report.

## 5. Launch brief and file-ownership checklist

Every worker receives: exact spec/plan/tasks revisions and approval/wave pointer;
task IDs; allowed paths and explicit exclusions; K-contract dependencies;
adopted base/overlay identity; model/variant; relevant specialist assignment;
allowed checks/effects; outputs and stop condition. Include unresolved O IDs.
Never give a blocked consumer a brief that invites it to choose its own API.

Before parallel launch the parent verifies:

1. No two active briefs own the same file, test helper or generated artifact.
   A new file added by a worker belongs to that brief, not an adjacent lane.
2. Producers are accepted/frozen before consumers compile against them; any
   isolated consumer workspace uses the same approved producer identity.
3. Worktree isolation preserves rather than discards the owner's overlay.
   No child stages/cherry-picks/commits. Parent integrates returned patches.
4. External output/HOME/cache paths are per worker; no generated root binary.
   Aggregate/measurement runs wait for writers; focused tests may run only on
   a clearly identified stable/disposable snapshot.
5. H/D/G assignments match plan §9; missing availability is reported, not a
   model switch. No automatic paid escalation for the free D lane.

## 6. Checkpoint continuation template

At each authorized batch/wave stop, parent records here:

- Actual authorization pointer, completed/blocked task IDs and next allowed action.
- Integrated code-state identity and inherited owner overlay disposition.
- Changed paths and any API/contract drift with its approved reconciliation.
- Actual models/variants, failures, partial writes and resumed sessions.
- Exact checks and whether they are worker, parent, trusted-base or not-run.
- Remaining finding/O IDs and required human decisions.
- State: `awaiting human review` for delivered batches; blocked/incomplete remains
  explicit. No checked task, review verdict or release is feature acceptance.

## 7. Readiness and continuation

**Reconciliation outcome:** all 23 ACs have implementation and verification
owners; all 55 findings remain mapped; shared-file paths have ordered transfers;
test scenarios include failure and concurrency boundaries. All batches have
dependencies and stop points. Parallelism does not weaken priority or authority.

**Current state (2026-11-17):** `origin/main` is at `e266ce6` (`docs(agents): record release and tap history`), with the split release workflows pushed beneath it in `f0746f7` (`ci(release): split build and publish with tag-trust guards`); all other 022 changes in the working tree are **uncommitted**.
- **Owner-accepted batch outcomes (2026-11-17):** B2, B3, B4 (T07, T09), B5, B7 (T15), and B9; T11 (digest codec) and B1's lock API as built are owner-approved.
- **Awaiting human review:** B1 (T01–T02), B4 container overlay fix and nested TUI vendoring (~16 MB; re-vendored 2026-11-17 after T12 and T13, go.mod/go.sum unchanged, vet clean), B6 T12 (MCP legacy lock `.agentic-sdd/mcp/.lock`), B7 T14 (release workflows pushed in `f0746f7`), and TUI test fixes (restore-path wrap-aware test fix and T19 stale expectation fix; full nested TUI package passed in parent's run at 73.6 s, vet clean).
- **Awaiting human review:** B6 T13 (fleet). Lock order (legacy, then agent guard) and journal order (live pointer, then sidecar) are amended in plan §12.10 item 13. Fail-closed sidecar reads and the missing conflict-resolution command are not yet ruled on. The parent re-ran `go test -race` for fleet and skillsync and vet: all pass.
- **Delivered, awaiting human review (superseded 2026-11-18):** B7 T16, B8, B10, B11 and B12 (development evidence); B13 reported `Blocked`/`Needs changes` (Opus review, see §8).
- **Open decisions / blockers:** T08 container lane is not accepted as qualifying evidence; help/probe conflict requires owner choice (probe checks the experimental gate, reject the flags, or keep strict and block); D-01, D-02, D-07 and real trusted-base adoption (O-01) remain open; `RELEASE_TOOLS_SHA` must be set by the owner on the `release` environment before any release publish can succeed.
- **Next permitted action:** review B1, B4 container overlay/TUI fixes, B6 T12, and B7 T14; resolve the help/probe conflict and container overlay decision; finish and review B6 T13, then re-vendor the nested TUI dependencies before launching remaining authorized batches. No commit, push, tag, release, real-home apply or next batch without explicit authorization.

## 8. Amendment r4 work items

Parents: [spec §9 amendment r4](spec.md#9-approval), [plan §13](plan.md#13-amendment-r4-open-item-resolutions-2026-11-17).
All items are **not started and not authorized**. Each needs an explicit owner go.
No task checked here is acceptance. Drafting items (R3, R4, R5 text) change docs
or tests only; R2 step 1 runs Docker; R7 edits `Taskfile.yml`.

| ID | Item | Owner decision needed | Verification | State |
|---|---|---|---|---|
| **A1** | R2 step 1: container suite re-run on HEAD, result recorded | go to run Docker | six of six tests with HEAD SHA, or the failures verbatim | **done 2026-11-17, awaiting human review:** 6 of 6 passed in 545 s at HEAD `a76d39b` plus an 11-path uncommitted working tree (`SDD_TRUSTED_CI_CONTAINERS=1`, log `/dev/shm/a1-container.log`). Not a clean-commit run: re-run after commit before citing it as qualifying evidence. |
| **A2** | R2 step 2: advance trusted base, drop overlay | base SHA and live-adoption go (after A1 passes) | fresh gate run on later candidate | blocked: owner chose (2026-11-17) to designate one SHA after B13 |
| **A3** | R1: Opus review request draft; later SHA designation | designation SHA | 008 AC-21 three conditions | blocked on A2 |
| **A4** | R3: consent-provenance template and per-record notes | quotes or "not recoverable" | records contain only owner-supplied text | not started |
| **A5** | R4: 014 per-AC/finding closeout table | accept / accept with exceptions / blocker | each row has an evidence pointer or `not_run` | **draft written, awaiting owner decision:** [a5-closeout-014-draft.md](a5-closeout-014-draft.md). Nothing accepted. |
| **A6** | R5: conflicted fleet transaction refusal message and stable test | none | `go test -race ./internal/fleet ./cmd/agentic-sdd`; vet | **done 2026-11-17, awaiting human review:** `ConflictRecoveryHint` in `internal/fleet/apply.go`; apply and restore tests pin it. `go test -race ./internal/fleet`, `go test -count=1 ./internal/fleet ./cmd/agentic-sdd` (985) and vet pass. |
| **A7** | R6: B8 routing/reviewer edits | adopt or discard pending pack overlay | AC-14/15/22 static checks | blocked on overlay decision |
| **A8** | R7: `task tui-vendor-check` and source-SHA stamp | none beyond the go | drift detected on a deliberate skillsync edit; clean tree passes | **done 2026-11-17, awaiting human review:** `tests/tui-vendor-check.sh`, `experiments/tui/vendor.stamp`, `task tui-vendor-check`. Verified: missing stamp, stale vendor and a deliberate edit all fail; a clean re-vendor passes. Nested TUI package (689 tests) passes on the re-vendored copy. Not yet wired into trusted CI (needs review first). |

Order: A4, A5, A6, A8 are independent and may run together after a go. A1 precedes
A2 and A3. A7 can follow the overlay decision. T16, B10–B13 are unchanged and still
need their own authorization. After any `skillsync`/`fleet` change, re-vendor and
re-run A8's check before the nested TUI gate.

**Notes from the first A-batch (2026-11-17):** (1) the vendored copy includes the
root-package non-Go files (for example `Taskfile.yml`, `README.md`), so the drift
check also fails after an edit to those; re-vendor and `--stamp` afterwards. If that
proves too noisy, narrowing the diff is a design change for the owner. (2)
`go mod verify` cannot check the path-replaced root module, so the check resolves
dependencies from `vendor/` instead. (3) A4, A2, A3 and A7 remain unstarted: they
wait for the owner's quotes, A1's clean-commit re-run plus a base SHA, and the pack
overlay decision.

**Update 2026-11-18:** (1) T16 callsite delivered as option 1: new `.github/workflows/docs-observe.yml` (`workflow_run` on `Release publish`, `contents: read` only, tag derived from the latest release, ancestry checked); `release-publish.yml` unchanged; T16 and the callsite are **awaiting human review**, live behaviour `not_run` (the workflow has never run). Calling `docs.yml` directly was rejected because its `deploy` job needs `pages: write` and `id-token: write`. (2) Node 24.22.3 is pinned in `website/.node-version`, `package.json` engines and the `docs.yml` image; the website checks were re-run on it (572 tests, 553 pass, the same single built-run focus failure). Node installed in `~/.local/node-v24.22.3`; signing keys came from GitHub main without a pinned fingerprint. (3) The 014 closeout report (AC-29) is at `specs/014-agent-fleet-model-overrides/closeout-report.md`, statuses unchanged, six exceptions proposed for the owner, nothing accepted. (4) B8's `AGENTS.md` paragraph is added.

**Opus verdict 2026-11-18 (real Opus 5.5, no fallback, candidate `95102d6`): Needs changes.** Findings: F-01 (appops drops MCP rollback facts; fix in progress), F-02 (observer tag is the latest release; hardened in `13e60d3`, re-check found a created_at window defect, since fixed), F-03 (this ledger's T16 rows, corrected), F-04 (comment overstated evidence parity, in progress), F-05 (workflow_run chaining limit note, in progress), F-06 (known gaps: execution comment, plan 8 vs qualification.json, CI wiring). Conditions before the verdict is final: container run on the final SHA 6/6; F-01 fixed or owner exception; F-02 hardened or owner-accepted; D-01 decided; owner accepts that T14/T16 live behaviour is `not_run`. The verdict is technical, not approval.

**Opus re-check 2026-11-18 (candidate `a4e95c0`): Needs changes for tagging.** F-01 resolved for the TUI (CLI still hides rollback facts on failure; Low, contract change not authorized); F-03/F-04/F-05/F-06 resolved apart from stale phrases; F-02 open with a NEW Medium defect: the `created_at` window check refuses every genuine release. The defect cannot affect publication (separate non-blocking workflow) but would waste the first live observation, so the tag waits for the fix. Low items: three separate `releases/latest` reads; `go mod verify` wording left in plan 13.7.

**Release v0.14.0 (2026-11-18):** tag `v0.14.0` on `effab6e` (CI and Docs green; Opus said Ready at `c571c80`; the later delta is website prose and fingerprints). `Release build` succeeded. First `Release publish` failed closed at the `release` environment (deployment ref `main` vs tag-only policy); policy amended (plan 12.9 item 14); the failed run was re-run and is waiting for the owner's environment approval. Not yet published: confirm with `gh release view v0.14.0` and the tap cask.

**v0.14.0 publish attempt 2 (2026-11-18):** refused at the CI gate. Docs workflow jobs
that are event-conditional (PR validate, scheduled observations, publish-time
observations) report `skipped` on a tag push, and the gate refused every non-success
conclusion. Fixed per the owner's approval: required checks are now an explicit
present-and-success set (`quality`, `verify`, `Trusted main validate and build`); named
non-applicable lanes may skip; everything else still refuses. `RELEASE_TOOLS_SHA` must
be re-set to the new tooling commit, then re-run the same publish run (no retag).

**v0.14.0 published 2026-11-18 (owner approved the deployment).** Release assets
(archive, `checksums.txt`, `manifest.json`) and tap cask 0.14.0 are live. First live
`Docs observe` run succeeded and matched the published revision: AC-08 live evidence
exists; the workflow_run chain (build -> publish -> observe) is observed. Auto-approval
for the owner's own publishes needs a UI-only step (remove the release environment's
required reviewer; no API exists). Still open: D-01 trusted-base designation, the 014
exceptions, B13 final integration closure, and the `real-guides.json` re-baseline
confirmation.
