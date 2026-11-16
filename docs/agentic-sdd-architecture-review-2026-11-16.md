# Agentic-SDD architecture and feature review — 2026-11-16

**Status:** advisory technical review. It records no approval, changes no batch
state and is not an Opus architecture verdict or a release certification. Under
repository policy a consequential architecture verdict still requires Opus and a
human decision; the five lanes below used one non-Opus model family.

**Baseline:** `HEAD` `c542b4a` (v0.13.0), 2026-11-16, 13 tags, 220 commits,
21 feature records. During the review the working tree also carried approved
Feature 020 §12 / B9 corrective work (016 lock heartbeat and interrupt handling,
execution documentation, fixtures). Lanes that needed a stable snapshot used a
`git archive` of `c542b4a`; findings describe that commit unless they explicitly
concern the in-flight B9 tree.

**Method:** five read-only lanes partitioned by surface, each on
`openai/gpt-6.1-sol` at high effort. No lane edited the repository, ran
`apply`/`restore`, published, or touched a real user home. Evidence included
CodeGraph discovery, isolated-home tests, race-enabled coverage, cross-builds,
and isolated reproduction probes. The lane reports were consolidated here; every
finding is reproduced with its evidence anchor.

## 1. Executive assessment

**Overall: a sound engineering core wrapped in a fragile assurance layer.**
Four lanes graded their surface *Sound with risks*; the feature-record lane
graded the process *Fragile*. No lane found a Blocker, a general authorization
bypass or an approved-scope violation. The weaknesses cluster at boundaries:
engine versus UI, shipped defaults versus host policy, canonical records versus
delivery history, and what the gates test versus what users run.

The most consequential gap is not code quality. It is that **the required gates
certify the eleven root packages, not the product that ships** — the TUI, the
four-binary archive and the Python regression suites are outside them.

**Bottom line:** fix the reproduced integrity bugs and close the
TUI/archive/Python gate gap before the next release is treated as qualified;
only then reconcile the records that already describe it as qualified.

### Ranked risks

| # | Risk | Findings | Why it matters | Effort |
|---|---|---|---|---|
| 1 | Check truthfulness and lock ownership | A-01–A-04 | A run can be recorded `verified-evidence` while a child it started is still alive; a symlinked lock truncates its referent; installer and fleet writes exclude each other only by convention; elapsed age alone can evict a live lock. Parts of A-01/A-04 are in B9; A-02/A-03 are not. | M |
| 2 | Gates miss the shipped surface | E-01, E-02 | Required checks select 11 root packages only. No TUI test, no four-binary/archive validation, no maintained Python suite, no race lane. `build-darwin` builds 2 of the 4 shipped applications. The TUI has 22.9k production lines. | M |
| 3 | Consent can dispatch off-screen | B-01, B-02 | At 100×16 the MCP confirmation accepts `y` with its decision controls outside the viewport. The advertised focus trap is absent in the composed app. Fleet has a related case behind its live-capability gate. | M |
| 4 | Assurance records are unreconciled | D-01, D-07, D-02–D-04 | 008 published v0.5.0 with `G-008-01` still open (owner disclosed the exception); 014 has committed code but no integrated closeout; 016/021 record a release and `not_run` in the same feature. | M |
| 5 | Two authorities for routing and tools | C-01, C-02 | The shipped pack still pins 9Router Claude routes and grants MCP/code-execution to reviewers, while the host policy says no Claude through 9Router and the independent-review contract forbids MCP/code execution. | M |
| 6 | Release supply chain and determinism | E-04, E-05, E-06, E-09, E-10 | Mutable action tags, persisted checkout credentials, unused OIDC scope, no release concurrency, wall-clock `.Date` in binaries, a release observation that `GITHUB_TOKEN` suppresses, and a docs smoke URL that repeats the base path. | S–M |
| 7 | Identity and bounded-input gaps | A-06, A-08, B-05 | The backup tree digest is ambiguous before SHA-256; manifest and `--result` reads can block or allocate before validation. | S–M |
| 8 | Record entropy | D-05, D-06, D-10–D-14 | Stale "pending approval" flags, dead handover links, accepted work with open headings, and release pointers lagging v0.13.0. | S |

### What is strong

- **Preview-first and reversible by construction.** `apply` is opt-in, existing
  skills are backed up before replacement, unchanged skills are no-ops, and
  tests use isolated homes rather than the real one.
- **Clean dependency direction.** No core package imports CLI or TUI code; the
  root module is stdlib-only; the TUI stays in its own Go 1.26 module.
- **Typed observations instead of prose.** Checkpoints, receipts and review
  packages distinguish blocked, incomplete, not-run and stale, and the receipt
  validator checks consistency rather than claiming approval.
- **A deliberately hardened PR boundary.** Trusted-base tooling, exact contract
  comparison, pinned networkless containers, independent probes and retained
  evidence are stronger than typical CI.
- **Entrypoint coherence.** Four named applications, a paired archive, link-time
  legacy wrappers, and an identity-checked `asdx-core tui` handshake are
  consistently implemented and documented.
- **Content discipline.** 18 skills with explicit authority owners, no broken
  internal Markdown targets, all 39 required sitemap pages present, and the known
  021 orchestration omissions fixed.
- **Scale.** 57.6k production / 71.7k test lines and ~1,266 test functions, with
  race coverage of 78.9–95.0% on the reviewed core packages.

## 2. System map

| Plane | Main code | Owner records | Assessment |
|---|---|---|---|
| Instruction | `skills-files/` (18 skills, 15,371 words, 110.7 KiB), `opencode-pack/` (26 files, 81.7 KiB), `integrations/`, adapters | 009, 017, 018, 020 | Coherent authority split; risks at the upstream/host boundary and in mandatory transitive reading |
| Control | `cmd/agentic-sdd` (8.1k production lines), `cmd/sdd-check`, `cmd/internal/tuirun` | 010–013, 019 | Sound execution model; flat package and ambient globals raise test-coupling cost |
| Interface | `experiments/tui` (22.3k production lines, nested module) | 013, 015 | Instance-local state and one write host are good; newer MCP/Fleet screens bypass shell-level confirmation geometry |
| State and domain | `internal/skillsync` (6.7k), `internal/fleet` (4.7k), `internal/appops` (3.0k), `internal/opencodeadapter` | 003, 009, 014 | Useful deep modules; skillsync mixes MCP transaction work with skill-tree ownership |
| Execution and evidence | `internal/execution` (6.3k), `harnesscheck` (1.4k), `harnessevidence` (3.1k), `buildinfo` | 004, 016, 021 | Strong typed facade; completion, lock ownership and identity edge cases remain |
| Delivery | `Taskfile.yml`, `harness/checks.json`, `.github/workflows/`, `.goreleaser.yml`, website | 008, 020 | Trusted PR lane is strong; release and nested-module coverage trail it |

The recurring pattern: each plane is well built on its own, and the failures
appear where one plane hands authority to another — UI to engine, defaults to
host, record to reality, gate to artifact.

## 3. Findings by lane

Severity is the lane's assessment, not an independent re-grade. Full evidence
anchors are reproduced because the raw lane reports live only in the session
workspace.

### 3.1 Core Go libraries — Sound with risks

Nine requested packages plus the embed roots were reviewed at `c542b4a`.
Race-enabled tests and vet passed on the nine-package selection; five probes
reproduced real failures despite green tests. Coverage: skillsync 78.9%, fleet
82.0%, execution 81.1%, appops 85.9%, harnesscheck 81.8%, harnessevidence 80.3%,
buildinfo 95.0%.

| ID | Severity | Finding | Evidence | Recommendation | Effort |
|---|---|---|---|---|---|
| A-01 | High | A successful check leader can leave ordinary descendants running while the checkpoint becomes `verified-evidence`; a redirected background `sleep` survives a pass. | `internal/execution/checks.go:274–344`, `checks_unix.go:15–39`; probe `TestArchitectureProbeLeaderExitLeavesOrdinaryChild` | Detect remaining owned-group activity on leader exit, terminate/join within a bound, and classify uncertainty as partial. `harnesscheck/process_unix.go:60–71` already models the distinction. | M |
| A-02 | High | The fleet lock leaf can be a symlink; acquiring and releasing the lock destroys the referent's bytes. | `internal/fleet/plan.go:355–410`; probe `TestFleetLockSymlinkTruncatesExternalFile` | Refuse non-regular lock leaves before mutation; open no-follow with descriptor/path identity checks. Preserves a stable lock inode. | S |
| A-03 | High | Installer and fleet writes to the same agent directory do not share writer exclusion; an embedded OpenCode install commits while the fleet lock is held. | `internal/skillsync/sync.go:90–294,432–695`, `internal/fleet/apply.go:70–75`; probe `TestInstallerWriterIgnoresFleetLock` | One mutation-owner seam over overlapping targets, acquired before planning and held through provenance persistence. | M |
| A-04 | High | Lock age can evict a live owner, and stale reclamation unlinks by pathname without ownership comparison. Execution waits 30 minutes; MCP waits 30 seconds. | `internal/execution/workspace.go:709–765,798–830`; `internal/skillsync/mcp_transaction.go:1231–1278`; probe `TestArchitectureProbeLiveExpiredLock` | Prefer kernel-held stable-inode locking; otherwise require renewable ownership and race-safe reclamation. Elapsed age is not proof of abandonment. | M |
| A-05 | Medium | Run cleanup inspects a lock but never acquires exclusion before `RemoveAll`. | `internal/execution/workspace.go:184–251` | Serialize cleanup with writers using an owner outside the directory being deleted; add a barrier-controlled clean-versus-writer test. | M |
| A-06 | Medium | Backup tree encoding is ambiguous before SHA-256: one file `a` containing `f b\nchanged` hashes identically to empty `a` plus `b` containing `changed`. | `internal/skillsync/manifest.go:94–120`; probe `TestDifferentTreesHaveSameBackupDigest` | Versioned, length-framed records or per-file content digests, with explicit legacy verification. | M |
| A-07 | Medium | Fleet provenance finalization is outside the installer transaction and not uniformly crash-durable; sidecars use truncating writes and a stale live pointer is not rebuilt. | `internal/fleet/apply.go:260–303`, `restore.go:272–315`, `manifest.go:219–224` | Durable transaction intent, atomic sidecars, pointer/journal reconciliation, and a distinct "files committed, metadata incomplete" outcome. | M |
| A-08 | Medium | Manifest safety checks occur after, or are absent from, unbounded reads: a FIFO `manifest.json` can hang discovery; fleet follows symlink leaves. | `internal/skillsync/restore.go:149,192–198`, `manifest.go:77–85`; `internal/fleet/manifest.go:101–116` | Share bounded regular-file opening with identity verification; add FIFO, symlink-leaf, oversize and replacement tests. | S |
| A-09 | Medium | Transaction interfaces expose ownership obligations: `AgentFileChange.Old` trusts caller state, and exported `WriteCheckpoint` requires a lock callers cannot acquire. | `internal/skillsync/agents.go:11–19,31–40,78–84`; `internal/execution/checkpoint.go:270–272` | Let mutation methods own locking and existence checks; make unchecked helpers private. | M |

**Cohesion note.** Production/test lines are skillsync 6,745/8,664, fleet
4,703/5,635, execution 6,258/5,390, appops 3,048/4,135. Skillsync is the clearest
mixed-responsibility module: 3,802 production lines concern MCP configuration and
runtime probing rather than skill trees. The demonstrated seam is MCP document
planning/ownership sharing one mutation primitive — not another transaction
engine. Fleet's parser/resolver/renderer and deployment/provenance are cohesive;
execution's natural seams are repository observation, workspace ownership, pure
task extraction and evidence persistence. Do not merge manifest schemas, digest
encodings or checkpoint/receipt vocabularies: they represent different identities.

### 3.2 CLI and TUI — Sound with risks

The process and module boundaries, preview-first engine reuse, identity-checked
launcher and single-write host are sound. Production size is smaller than the
raw totals suggest: CLI 8,092 production/16,151 test lines across 26/38 files;
TUI 22,266/26,620 across 39/54 files. CLI/checker tests (706 pass events), scoped
vet and both binary builds passed. Nested tests passed in 255s; earlier 120s/70s
runs timed out but no deadlock was established.

| ID | Severity | Finding | Evidence | Recommendation | Effort |
|---|---|---|---|---|---|
| B-01 | High | Confirmation visibility is not enforced consistently. At 100×16 the MCP decision controls are offscreen but `y` emits `StartWriteMsg`; a Fleet restore prompt can remain invisible after PageDown while `y` dispatches (Fleet requires live capability; shipping defaults are preview-only). | `experiments/tui/modal.go:52`; `mcp.go:621–643`; `fleet.go:1032–1068`; `fleet_recovery.go:253–262` | Pinned policy/decision geometry and root ownership for every write confirmation; block affirmative input when essential content cannot fit; integrated height/scroll tests. | M |
| B-02 | Medium | MCP's advertised focus trap is absent in the composed application: Tab moves shell focus to navigation, and `1` escapes the confirmation. | `mcp.go:269,621–643`; `app.go:745–749,845–847`; `mcp_screens.go:370` | Route confirmation keys before global navigation; distinguish text/decision focus; test shell × child × key behavior. | M |
| B-03 | Medium | MCP result rendering bypasses the common truthful-outcome projection: nil-error completion is called successful, a backup is claimed when absent, and log failure/no-change distinctions are lost. | `mcp.go:405–427`; `ops_host.go:695–751,1203` | Preserve structured results on errors and use one summary projection for no-op, partial/rollback, backup and log failure. | M |
| B-04 | Medium | Flat packages permit ambient dependencies and process-global substitution in tests; multi-boolean TUI modes encode implicit precedence. | CLI `mcp.go:42`, `mcp_write.go:42–52`, `tui.go:50`, `tui_test.go:984`; TUI `app.go:57–139`, `fleet.go:48–153` | Inject per-invocation services/streams; extract bounded command/journey packages with explicit messages. Do not split by file count. | M |
| B-05 | Medium | `--result FILE` uses `os.ReadFile` before checking size; stdin is bounded, file input is not, and special files can block. | `cmd/agentic-sdd/execution_cli.go:118–131` | Open/check the file and reuse the bounded reader; decide explicitly whether non-regular inputs are supported. | S |
| B-06 | Medium | Root help materially understates accepted flags and write surfaces: it calls MCP read-only and omits apply/restore; pack flags and several help topics are undiscoverable. | `main.go:695–746`; `mcp.go:159–178`; `documentation_reference_test.go:1130–1136` | Reconcile root help with dispatch; retain topic detail; require reviewed help-contract changes. | S |
| B-07 | Low | Help/validation ordering has inconsistent exceptions: `run check --help` exits 2 without `--`; `preview --version nonsense` exits 0. | `cmd/agentic-sdd/run.go` (`runRunCheck`); `main.go:275–305` | Choose a compatibility policy and test exceptional grammar explicitly. | S |
| B-08 | Low | Wrapper tests fake Exec or merge streams; they cannot independently prove byte-identical JSON stdout, deprecation-only stderr and nonzero exit forwarding together. | `tui_test.go:991–1110`; `legacy_exec_posix.go:13`; `tests/tui/test_launcher_lifecycle.py:167–180,429–502` | Add subprocess tests with separate pipes for link-time legacy builds. | M |
| B-09 | Low | Startup performs synchronous inventory and log reads before rendering large/slow homes. | `cmd/internal/tuirun/run.go:136–151`; `activity.go:271–302` | Explicit loading state; move initial reads behind generation-bound commands; measure first. | M |
| B-10 | Low | Release orientation lags the baseline: README evidence names v0.11.0 and the TUI architecture text mixes v0.9/r6-era statuses. | `README.md:44,104–108`; `docs/tui-architecture.md:16–34,404–406` | Link one release-specific evidence owner; keep historical notes subordinate. | S |

**Entrypoint note.** The four-name mapping is coherent, `asdx` rejects
CLI-shaped arguments, and legacy modes are link-time selections rather than
basename guesses. `asdx-core tui` verifies protocol and build identity with no
PATH fallback. Legacy CLI forwarding deliberately lacks that handshake, so a
manually mixed same-directory core can run without compatibility verification —
an accepted distinction, not authentication.

### 3.3 Skills and documentation — Sound with risks

All 18 portable skill bodies, their references, the opt-in pack, selected public
guides and configuration inventories were reviewed. No broken local Markdown
targets were found in `skills-files/`, `opencode-pack/`, `docs/` or
`integrations/`; all 39 required sitemap routes exist; the two permitted static
suites passed (33 tests). The known 021 orchestration omissions are fixed.

| ID | Severity | Finding | Evidence | Recommendation | Effort |
|---|---|---|---|---|---|
| C-01 | High | Shipped Claude routing contradicts current host policy and configured profiles: the pack says Claude and Gemini/Argon live under 9Router, while the host says no Claude model routes through 9Router. Host agents were also renamed during review, leaving stale fallback references. | `opencode-pack/routing.md:11,25–30,58–60`; `opencode-pack/adapters/sdd-plan-9r.md:4`; host `AGENTS.md:56,91` | Decide supported upstream defaults separately from personal overrides; reconcile provider/role tables and name their authority. Do not copy personal policy wholesale upstream. | M |
| C-02 | High | Independent-review contracts prescribe incompatible tools: the portable reviewer "must not … use remote, MCP or code-execution routes", while the shipped review profile instructs CodeGraph, Context7, `execute` and Exa. | `skills-files/agentic-sdd-router/references/independent-review.md:35`; `opencode-pack/agents/sdd-code-review-9r.md:2,21–35,59–65` | Distinguish ordinary advisory review from the strict independent route; give each a compatible tool contract and truthful label. | M |
| C-03 | Medium | Shared policy orders `codegraph init -y` without conditioning on write authorization, so a read-only review of an unindexed checkout can mutate it. | `skills-files/agentic-sdd-router/references/workflow-policy.md:75,79`; `research-spec/references/search-mcp-workflow.md:13` | Condition bootstrap/sync on write authority; read-only branches use an existing index or targeted reads. | S |
| C-04 | Medium | Documentation index still calls migrated guides active and site content pending, although destination pages exist; orchestration/installer stubs give bare route names rather than working links. | `docs/README.md:8–12,25–36`; `docs/orchestration-guide.md:3,7–9`; `docs/installer-reference.md:3,7–9` | Finish migration pointers; make the site the current entry; label engineering stubs historical. | S |
| C-05 | Medium | A dated research study is promoted as current guidance although it says the librarian is "not installed" and "No written precedence rule" exists. | `docs/README.md:45–47`; `docs/research-skills-and-subagents-current-state.md:3,18,309` | Preserve the dated study; relabel its index entry and link current normative owners. | S |
| C-06 | Medium | Correction/re-review authority differs between core and pack: core allows scoped re-review within the current budget; the pack says a second review is a new authorization. | `router/references/review-resolution.md:43–47`; `verification-review/SKILL.md:41`; `opencode-pack/.../failure-and-reconciliation.md:80–82` | Let core own ordinary scoped re-review; reserve separate authorization for a new independent/model review. | S |
| C-07 | Low | Shared-policy loading dilutes lightweight branches: a 160-word explanation requires 2,008 policy words. | `wait-what/SKILL.md:9–11`; `router/references/workflow-policy.md:23–101` | Consider a small authority kernel with conditional lifecycle references; retain entry-point prohibitions. | M |
| C-08 | Low | Research trigger exception is ambiguous: the description excludes generic research, but the body permits broad explicit research. | `research-spec/SKILL.md:3,10,17` | State whether explicit broad requests must still name the affected SDD decision. | S |
| C-09 | Low | Pack safety inventory omits its OpenAI writer, so the read-only/writer list is inaccurate. | `opencode-pack/README.md:80`; `opencode-pack/agents/sdd-openai-implement.md:6–8` | Include the request-only OpenAI writer; derive the list from profile metadata where practical. | S |
| C-10 | Low | Public release qualifier lags the released baseline: README foregrounds v0.11.0 while v0.13.0 is released. | `README.md:44–45,104–109` | Link dated release evidence and identify newer releases without implying unperformed macOS/runtime verification. | S |

**Trigger and duplication note.** `grill-me` versus `grilling` is an intentional
alias with `disable-model-invocation: true`; plan versus tasks name different
outputs. The router owns 44.6% of portable bytes chiefly as shared contracts;
the burden is mandatory transitive reading, not individual skill length. The
canonical owners are `workflow-policy.md` (approval/batch), `review-resolution.md`
(findings) and `task-contracts.md` (readiness). Do not export the dated local
skill catalog as portable truth.

### 3.4 Feature records and delivery state — Fragile

All 21 features record a combined approval, and acceptance remains distinct from
technical findings and live authorization. Git chronology gaps for 001/002 are a
provenance limitation, not proof of unauthorized implementation. The fragility is
in reconciliation: readers must arbitrate contradictory canonical text and
reconstruct delivery from Git.

| Feature | Recorded state | Delivery / assurance issue |
|---|---|---|
| 001 CLI quality | B1 awaiting review | Delivered in v0.1.0; closure absent |
| 002 publishing | B1/B2 awaiting review | Delivered in v0.2.0; plan still says pending |
| 003 restore | B1–B3 awaiting review | Delivered in v0.3.0; tasks also say no approval/batch |
| 004 harness | Accepted; technical blocked | Deliberately retains failures; partial against full native outcome |
| 005 completion | Closed | Delivered; plan header stale |
| 006 independent review | B4 deferred | Partial; IR-06-01 remains; WIP shipped, not certified |
| 007 Taskfile | Delivered/adopted | Old-base gap explicitly accepted, not passed |
| 008 core quality | B1 accepted; later review pending | **G-008-01 open at v0.5.0 release**; no final qualified-core verdict |
| 009 orchestration | Complete/accepted | No material conflict found |
| 010 guided CLI | Accepted | Fresh-user AC-12 remains `not_run` |
| 011 branding | Delivered | T16 verification text stale |
| 012 remediation | Awaiting review | Delivered; 013 later accepts DEP-012 |
| 013 unified TUI | B1–B14 accepted | Stale pending leaves and headings contradict closure |
| 014 fleet core | B6/B7 in progress | **Committed code, no integrated closeout**; only pre-integration B5 report; T01–T12 unchecked |
| 015 Fleet/MCP TUI | W3 pending; B4 unstarted | MCP accepted, Fleet partial; headings still say MCP awaiting |
| 016 execution | B1–B3 awaiting review | Release recorded and `not_run` in the same evidence; B9 corrective in progress |
| 017 context integrity | Accepted/closed | No new implementation required |
| 018 research handoff | Accepted | Spec still says uncommitted |
| 019 entrypoint | B0–B3 awaiting review | Archive evidence not reconciled; one aggregate `go test ./...` record conflicts with the 008 fence |
| 020 documentation | B1–B7 accepted; B8 awaiting; B9 active | B8 evidence claims no commit after commit; AR-1–AR-10 open/fixing |
| 021 CLI verification | Complete; awaiting acceptance | Draft headers and `not_run` lines contradict the release |

| ID | Severity | Finding | Evidence | Recommendation | Effort |
|---|---|---|---|---|---|
| D-01 | High | 008 published v0.5.0 without closing its retained assurance gate. Owner authorization disclosed the exception; this is not an unauthorized-release allegation. | `specs/008-core-release-quality/spec.md:243–245`, `tasks.md:151,357–364` | Obtain an explicit current qualified-core designation and review disposition; preserve historical failures. | M |
| D-02 | High | Publication authority is incompletely reconciled for shipped 014/016/019/018; later release decisions live in evidence or handovers, not the owning spec's approval section. | `014/spec.md:467`; `016/spec.md:335`, `b3-acceptance.md:5`; `019/spec.md:140`; `018/spec.md:202–206` | Record actual later decisions and scope in the owning record; never invent consent. | M |
| D-03 | Medium | Known 016 release/`not_run` contradiction repeats in 021. | `016/b3-acceptance.md:28–35,60`; `021/tasks.md:248,256` | Time-scope the old `not_run` statements. | S |
| D-04 | Medium | 020 B8 handover/evidence still says uncommitted and unauthorized for publication despite commits in v0.13.0. | `docs/handovers/2026-11-16_020.md:7–8,40–46`; `020/tasks.md:670,689–693` | Reconcile delivery with Git and live authorization; acceptance stays pending. | S |
| D-05 | Medium | Pending-approval declarations survive actual combined approval in 003, 005, 018, 019 and 021. | `003/tasks.md:3,182–184`; `005/plan.md:4`; `018/plan.md:58`; `019/plan.md:4`, `tasks.md:5`; `021/spec.md:3–6,238` | Replace current flags with approval pointers; label preserved drafts historical. | S |
| D-06 | Medium | Accepted TUI work has stale task-level and summary states in 011, 013 and 015. | `013/tasks.md:7,27,112–113,220,226–231`; `015/spec.md:4,538,554–571`; `011/tasks.md:158,189–196` | Reconcile leaves and continuations without re-accepting scope. | S |
| D-07 | High | 014 claims a consolidated acceptance report, but the only verification report is explicitly pre-integration B5; B6/B7 remain active and T01–T12 unchecked. | `014/tasks.md:16–17,25–88,109`; `b5-verification-report.md:9–15,115–142` | Produce integrated AC/finding dispositions, then human/required architecture review before Fleet-live expansion. | M |
| D-08 | Medium | 020 AR-1–AR-10 remain "open/fixing" after conditional acceptance; later policy decisions are not finding closure evidence. | `020/tasks.md:298–321`; `020/spec.md:241–244` | Map each AR ID to fix evidence or accepted deferral. | M |
| D-09 | Medium | 019 records unrestricted root `go test ./...` despite the 008 exclusion fence. | `019/tasks.md:168`; `AGENTS.md:120–125` | Reconcile actual selection/logs; report inadmissible evidence rather than weakening selectors. | S |
| D-12 | Medium | 021 says independent reviewers reran groups, while the evidence says both were shell-denied. | `021/tasks.md:246,263`; `test-case-matrix.md:119` | Separate static review, parent execution and structural traceability. | S |
| D-13 | Medium | Earliest committed approval documents follow some implementation commits. This is a provenance gap, not unauthorized implementation. | Git order `0ea53f7`,`aec99e7` → `6a3449a`; `209bd0a`,`65bb267` → `f5b2711` | Retain genuine conversation evidence or state its absence. | M |
| D-10 | Low | Index remains "integration-pending"; 018 handover blocks on resolved approval; 021 handover lists fixed CLI/skill gaps as current facts. | `docs/README.md:11–12,25–28`; `docs/handovers/2026-11-13_018.md:5,26–44`; `2026-11-16_021.md:56–61` | Fold resolved handovers; time-scope baseline facts. | S |
| D-11 | Low | Local-link scan found missing destinations, including deleted handovers. | `008/tasks.md:183`; `013/spec.md:6,212`; `018/plan.md:93` | Use surviving canonical references or commit-qualified historical links. | S |
| D-14 | Low | Release-facing snapshot still highlights v0.11.0 while 021 inspected four-name v0.13.0 archives. | `README.md:44–45,104–109`; `021/spec.md:250–252` | Refresh exact-version evidence links; retain live-macOS limitations. | S |

**Cross-feature sequencing.** 013 and 019 are complementary, not duplicate:
013 owns the unified TUI, 019 supersedes naming/default-entry contracts. 016 →
017 → 021 are service, integrity correction and CLI verification; accepting 017
does not accept 016. 014 is the engine and 015 the UI, but 015 also absorbed MCP
scope — separate accepted MCP delivery from pending Fleet W3/B4. 004–008 need
their findings archived, not reopened.

### 3.5 Build, verification and release — Sound with risks

`task test` and `task vet` passed against an isolated export of HEAD. The PR
boundary is unusually deliberate: base-owned tooling, exact contract comparison,
pinned networkless containers, independent probes and retained observations.
The weaknesses are at the release boundary and in what the required set omits.

| ID | Severity | Finding | Evidence | Recommendation | Effort |
|---|---|---|---|---|---|
| E-01 | High | Required verification misses the shipped TUI and nested dependency integrity. `build-darwin` builds the CLI and `sdd-check`, not the four shipped applications; no archive-member or pair-handshake smoke test. | `harness/checks.json:22–40`; `.goreleaser.yml:73–121`; `experiments/tui/go.mod:5–29` | Add an explicitly approved release-surface lane: nested tests/vet/module verification, archive members/slices and pair-handshake smoke tests. Preserve the 008 fence. | M |
| E-02 | High | Maintained Python regressions are not selected by core CI, release verification or hooks, so skill-rule and trusted-gate regressions can pass every gate. | `harness/checks.json:5–49`; `lefthook.yml:11–16`; `scripts/docs-ci.py:288–310`; `AGENTS.md:86–93` | Adopt individually named supported-core Python suites, including core/research workflow and trusted-gate regressions; never broad discovery. | M |
| E-04 | High | Privileged release preparation uses mutable checkout/setup-go action tags, retains checkout credentials, and holds unused OIDC permission across verification. | `.github/workflows/release.yml:8–28` | SHA-pin actions, disable persisted credentials, remove unused OIDC, separate read-only verification from publishing privileges. | S |
| E-03 | Medium | `verify`/`tidy-check` clear fewer Go controls than `test`/`vet`: GOFLAGS, GOENV and GOWORK remain inherited. | `Taskfile.yml:45–51,91,111`; `internal/harnesscheck/runner.go:191–210` | Share one explicit environment profile across bootstrap and executed checks; test hostile inherited values. | S |
| E-05 | Medium | Release publication has no explicit concurrency/timeout/recovery policy; token preflight proves readability, not tap write authority; verification evidence is not uploaded. | `.github/workflows/release.yml:12–57`; `Taskfile.yml:109–112` | Serialize publishing, bound the job, retain evidence, validate permission scope without writes, document asset/tap recovery. | M |
| E-06 | Medium | Release binaries embed wall-clock `.Date` despite deterministic paths and file timestamps. | `.goreleaser.yml:31–40,56–65` | Inject `.CommitDate` or a fixed source timestamp; compare independent rebuilds. | S |
| E-07 | Medium | Two snapshot recipes have materially different guarantees; public `task release` lacks the experimental source-content identity and tool checks. | `Taskfile.yml:137–140`; `Taskfile.experimental.yml:292–352`; `.goreleaser.yml:43` | Consolidate snapshot policy behind one entrypoint, retaining nonpublishing behavior and external output. | M |
| E-08 | Medium | Trusted-core staging ignores a small root set only, not ignored root `dist` or nested website dependencies/build output. | `tests/harness/trusted_ci.py:18,100–129`; `.gitignore:7–8`; `website/.gitignore:1–4` | Define reviewed generated-output exclusions or require a clean export; keep arbitrary candidate ignore rules untrusted. | S |
| E-09 | Medium | Release-doc observation relies on `release: published`, but publication uses `GITHUB_TOKEN`, which suppresses downstream workflow events. | `.github/workflows/docs.yml:137–138,742–778`; `release.yml:54–56` | Invoke the read-only observation explicitly or use an authenticated dispatch; do not add a broad PAT. | S |
| E-10 | Medium | Smoke asset URLs repeat `/agentic-sdd`: `SITE` already includes it and extracted CSS/JS paths include it again. | `.github/workflows/docs.yml:931,939–954` | Resolve paths against the origin and test URL construction; keep observation-only policy explicit. | S |
| E-11 | Low | Cold container work shares fixed 120/180-second budgets; timing fixtures use short wall-clock ceilings; no default race lane. | `harness/checks.json:6–40`; `tests/harness/trusted_ci.py:155–163,300–303` | Measure cold two-CPU runs, diagnose before changing budgets, add a supported-core race lane. | M |
| E-12 | Low | Receipt result parsing does not require EOF after the first JSON object; malformed trailing bytes are accepted. | `internal/harnessevidence/validate.go:357–370` | Require `io.EOF` with malformed-tail and duplicate-key regression tests. | S |

**Gate map (condensed).** Local hooks are bypassable and run working-tree
tests. `task test`/`vet` select 11 root packages with no nested-module, race or
Python lane. `task verify` adds six checks but inherits environment. CI runs the
trusted-base lane and probes for product paths, skipping `docs/**`/`specs/**`-only
changes. Release runs `task verify` and GoReleaser on any `v*` push with no
dependency on PR status. No skip, filter or weakened selector was found; excluded
005/006/007 work is not selected.

## 4. Cross-cutting themes

1. **Assurance follows the test seam, not the user surface.** The core has
   deliberate, well-evidenced contracts; the TUI, archive and instruction
   behavior are largely outside the required gates. The highest-value process fix
   is a scoped release-surface gate, not more core tests.
2. **There are two authorities for configuration.** Model routing, tool
   permissions and review contracts are duplicated across the shipped pack, the
   installed host configuration, the feature records and handovers. The copies
   drift independently; C-01/C-02 and D-02/D-04 are the same failure at different
   layers. Name one owner per concern and make the others point to it.
3. **Observations are typed, but completion and identity are still ambiguous.**
   Checkpoints distinguish outcomes well, yet A-01 can mark a run complete while
   work continues, and A-06's digest cannot distinguish two different trees.
   Truthful evidence needs both a completion rule and a collision-resistant
   identity.
4. **Enforcement is duplicated across four runners.** Taskfile, CI, Lefthook and
   `harness/checks.json` implement overlapping steps with different environment
   clearing, selection and timing. The divergence is small today but makes
   "green" mean different things in different places.
5. **Consent is a layout property, not just a policy property.** B-01/B-02 show
   that a correctly worded default-No confirmation can still dispatch while its
   decision content is invisible. Confirmation geometry belongs to the shell
   contract, not to individual screens.
6. **Records are mid-migration.** 020 is moving public guidance to the site while
   historical studies, handovers and specs remain in place. Until the migration
   closes, "current" is ambiguous and stale pointers accumulate (C-04/C-05,
   D-04/D-05/D-10).
7. **Scale is manageable; coupling is the cost.** Production code is 8.1k CLI and
   22.3k TUI lines, not the 24k/49k totals that include tests. Extracting one
   execution/MCP boundary and injecting per-invocation services would pay off
   sooner than a package-per-file reorganization.

## 5. Recommended sequence

No item below is authorized by this review. P0 means "fix before the next
release claim or live expansion", not "start now".

| Priority | Action | Findings | Effort |
|---|---|---|---|
| P0 | Preserve A-01/A-02/A-03 reproductions as regressions; fix symlinked-lock truncation and shared installer/fleet writer exclusion; confirm B9 covers the live-lock and leader-exit cases. | A-01–A-04 | M |
| P0 | Add a scoped release-surface gate for the nested TUI, the four-binary archive and named Python suites. Requires an explicit 008-fence decision. | E-01, E-02 | M |
| P0 | Enforce confirmation geometry in the shell (pinned decision area, block `y` when content cannot fit) and route confirmation keys before global navigation. | B-01, B-02 | M |
| P0 | Reconcile assurance records: 008 `G-008-01`, 014 integrated closeout, 016/021 release versus `not_run`, and later release consent in owning specs. Human decisions, not code. | D-01, D-02, D-03, D-07 | M |
| P0 | Harden release: SHA-pin actions, drop persisted credentials and unused OIDC, serialize publication, use `.CommitDate`, fix the docs smoke URL and release event. | E-04–E-06, E-09, E-10 | S |
| P1 | Name one owner for model routing/tool policy; split ordinary advisory review from the strict independent contract; condition CodeGraph bootstrap on write authority. | C-01, C-02, C-03 | M |
| P1 | Version the backup tree digest, bound manifest/`--result` reads before allocation, and add FIFO/symlink/oversize tests. | A-06, A-08, B-05 | S–M |
| P1 | Inject CLI services/streams and isolate MCP's model; extract the MCP document seam from skillsync. | A-09, B-04 | M |
| P1 | Finish the documentation migration: current entry points, historical labels, handover cleanup, approval-pointer hygiene. | C-04, C-05, D-04, D-05, D-10, D-11 | S–M |
| P2 | Deepen remaining transactions: cleanup exclusion, provenance durability, result projection, help parity, wrapper stream tests, startup loading state. | A-05, A-07, B-03, B-06–B-10 | M |
| P2 | Reduce context load and tighten research/trigger wording; fix pack inventories and release pointers. | C-07–C-10, D-06, D-14 | S |
| P2 | Unify verification environments, snapshot identity, receipt EOF and cold-run budgets. | E-03, E-07, E-08, E-11, E-12 | S–M |

### Decisions the owner has to make

1. **008 disposition:** designate the current qualified core and close
   `G-008-01`, or keep the claim explicitly open. A release does not close it.
2. **014/015 Fleet-live gate:** require integrated acceptance evidence (including
   an Opus verdict where consequential) before expansion, and decide whether B4
   is selected or remains unstarted.
3. **016/019/020 B8/021 acceptance:** accept, defer or reject after B9 completes;
   release alone closes none of them.
4. **Upstream versus host policy:** which model/tool routing is canonical for the
   shipped pack, and which parts are personal overrides that must not ship.
5. **Review contract:** whether the shipped reviewer is an ordinary advisory
   reviewer with MCP/Context7, or a strict independent reviewer with read-only
   local access. Both cannot be true under the current wording.
6. **Gate scope:** whether to amend the 008 fence to add TUI, archive and Python
   suites to the required set.

## 6. What this review does not establish

- No runtime, live-system, provider, deployment or Homebrew behavior was
  certified. Reports of releases, deployments and CI runs were attributed, not
  independently re-executed.
- No macOS or Windows native behavior, release-archive contents, Mach-O slices,
  signing/notarization or accessibility certification.
- No secret, credential or private-evidence inspection.
- Excluded Feature 005/006/007 experiments were not selected or validated.
- The five lanes share one model family; correlated blind spots are possible.
  There is no Opus verdict, so consequential architecture conclusions remain
  provisional under repository policy.
- The in-flight B9 working tree was not reviewed as shipped; findings describe
  `c542b4a` unless stated otherwise.
- A green check never proves an unrelated criterion, and no lane claims human
  acceptance.

## 7. Appendix — metrics and evidence

| Metric | Value |
|---|---|
| Baseline | `c542b4a`, v0.13.0, 13 tags, 220 commits |
| Go source | 304 files; 57,640 production / 71,689 test lines |
| Root module | 34,704 production / 44,477 test lines; stdlib-only |
| TUI module | 22,936 production / 27,212 test lines; Go 1.26 |
| Test functions | 738 root + 528 TUI (~1,266) |
| Core coverage (race) | skillsync 78.9%, fleet 82.0%, execution 81.1%, appops 85.9%, harnesscheck 81.8%, harnessevidence 80.3%, buildinfo 95.0% |
| Skills | 18 skills, 30 files, 15,371 words, 110.7 KiB; router owns 44.6% of bytes |
| OpenCode pack | 26 files, 81.7 KiB; 14 profiles plus 4 free-first presets |
| Documentation site | 39 required sitemap pages, all present |
| Feature records | 21 features, all with a combined approval record |
| Static content suites | 33 tests passed (11 core workflow + 22 research workflow) |
| Lane verification | Core race+vet passed; CLI 706 pass events; TUI nested run passed in 255s; `task test`/`task vet` passed on an isolated export |

**Reproductions preserved during the review (not committed):**

- `TestArchitectureProbeLeaderExitLeavesOrdinaryChild` (A-01)
- `TestFleetLockSymlinkTruncatesExternalFile` (A-02)
- `TestInstallerWriterIgnoresFleetLock` (A-03)
- `TestArchitectureProbeLiveExpiredLock` (A-04)
- `TestDifferentTreesHaveSameBackupDigest` (A-06)
- TUI height/focus model probes for B-01/B-02

The five lane reports contained additional narrative and method detail. This
document reproduces every finding with its evidence anchor; the raw files were
session workspace only and are not part of the repository.
