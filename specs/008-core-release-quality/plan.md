# Core release quality — plan

Revision: r4 approved, 2026-11-07; amended 2026-11-07 for core contract approval, commit/push authorization, GoReleaser snapshot, and Linux CI runner / isolated Linux evidence sufficiency for AC-14/T10. Input: [spec r2](spec.md). Single approval owner: [spec §Approval](spec.md#approval). B4a–B5 executed; commit and push completed at checkpoint 7d3e99b (7d3e99b083f094ab4aba17e835bdcca39ee4abd5); tagging/releasing and trusted reference designation remain separate decisions.

## 1. Approach and inspected baseline

Deliver one release-readiness outcome, guided by [vision H0](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#10-roadmap-by-evidence-gates-not-hype-or-deadlines). Preserve B1–B3's delivered scope/evidence. Split the unstarted B4 into B4a (core boundary), B4b (core gate/adoption) and B4c (release QA), then retain B5's identified-candidate assessment. Each batch stops for human review. This is not implementation of H1–H3, an orchestration project or a broad repository reorganization.

Scope constraint confirmed by the user during drafting: strictly follow Feature 008 and exclude its out-of-scope components. Work on 004/005/006/007 is limited to the current-facing reconciliation, retained findings, distribution integrity and supported-core verification explicitly required by 008. No experimental implementation, repair, pilot, native dispatch, C2 certification or retrospective completion of those features is included.

### Deprecated-feature work fence (r2)

The owner accepted B1 and explicitly directed that **deprecated Feature 006 and Feature 007 scopes are excluded before B2**. Their historical records remain evidence, not work queues.

- Do not resume, implement, repair, complete, certify or execute either feature. Do not investigate 006 pilot/mitigation/runtime/provider controls, run its experimental fixtures, or reopen 007's old-base adoption/transition work. B1's limited status/warning reconciliation is delivered, not an ongoing assignment to those features.
- Saving the owner's existing 006/007 changes as WIP commits is a separately authorized preservation action. It does not approve their behavior, include them in the supported product, waive findings, or authorize more work.
- Preserve all 18 portable skill IDs and packaged references. Deprecating feature work does not authorize deleting shipped portable guidance, changing the installer, or removing its package-integrity checks.
- Validate the **maintained supported core** directly under 008 R6/R9/R10: current commands, installer/restore, packaged skills, check/evidence code and reviewed-base trust boundary. This is not continuation of 007. Retain six required check roles and meaningful anti-weakening coverage; the proposed C4 selection changes need reviewed adoption, not a silent substitution or deprecated-feature tests.
- Future source/fixture selection must exclude 006 native/profile/dispatch/pilot suites and 007 historical migration/transition exercises. Core Taskfile regression tests remain eligible where they verify current supported commands, not old-feature completion. Historical findings can be cited only to explain exclusions; no mitigation design or additional experimental research is implied.

Removing required core Taskfile/trust verification would conflict with the approved spec; this clarification excludes deprecated feature work, not those release-critical core checks. Any broader removal needs an explicit spec-scope decision rather than silent weakening.

### Historical B3 blocked-test selection addition (r3)

The owner extended the temporary exclusion to blocked parts of 005 as well as 006/007 and requested a test-selection task before/during B3. Do not select blocked native ACP/admission/runtime certification, native/profile/dispatch/pilot or historical migration/adoption validation. Offline shipped core installer, package, check/evidence and current-command/trust coverage remain required; old feature numbers alone do not exclude supported core code.

T05a adds explicit local `test-core`/`vet-core` entry points with an allowlisted package/file selection and failure controls. Core CLI tests may compile the existing helper as a production dependency while excluding its experimental test file; compilation is not runtime validation. No native client or provider is launched. Keep this profile distinct from the unchanged aggregate `task test` / required `go test ./...` and trusted contract. Do not silently rewrite required argv, weaken equality or claim a scoped pass is an aggregate pass. If the exclusion prevents an aggregate run, record that row not_run/blocked pending a human gate decision; core-only evidence cannot silently waive it. B4 still owns the complete release-QA matrix and any separately reviewed trust-contract adoption.

### Selected core-first amendment (r4)

The owner chose core-only default commands and the minimum technical boundary needed to keep experiments inactive. The r3 paragraph describes B3's delivered state, not a permanent requirement to retain `./...`. Proposed C4/C6 supersede future design only after this amendment is reviewed and a batch is authorized. G-008-01's policy direction is selected; its machine-gate/evidence conflict is not yet resolved.

Bounded source observations at amendment HEAD `0bab31e`: `main.go` dispatches through `runOpenCode` and `printOpenCodeUsage` and advertises the helper in ordinary help; `opencode.go` is their single implementation and imports the experimental adapter. `adapterfiles.go` embeds inert configuration/catalog resources and the receipt schema. The trusted bootstrap compares complete required records, wraps their steps in isolated execution and injects an independent core probe. These observations support narrow CLI/build and gate edits—not experimental runtime investigation.

There is no top-level product SRS or technical specification. The canonical vision is long-lived strategy, not implementation approval; its [principles](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#6-non-negotiable-engineering-principles) and [document ownership](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#13-relationship-to-other-repository-documents) inform this amendment. The following table preserves the 2026-11-05 discovery baseline; current delivery/evidence is in tasks, not inferred from those historical findings.

| Observed source | Design consequence |
|---|---|
| HEAD `0ee5aaf65df7485d872d5784b17d6435175a185f`; unrelated modified 006/integration/test files and untracked docs | Not a clean release candidate. Preserve all work; inclusion needs provenance and review, not automatic staging. |
| [007 approval](../007-taskfile-migration/spec.md#approval) records direct trusted-branch adoption; [tasks](../007-taskfile-migration/tasks.md#continuation-record) retains contradictory earlier summaries | Reconcile current-facing state; retain 007-F1 as accepted adoption evidence gap, not a passing old-base PR gate. Hosted result still needs observation. |
| [006 tasks](../006-independent-review/tasks.md) retains deferred B4 and IR-06-01 | Disclose unsupported generic native reviewer; do not resume 006 or certify the proposed C2 export. |
| [source.go](../../internal/skillsync/source.go) compares structure/content, ignores modes; requests 0755 directories/0644 files on installation; [restore.go](../../internal/skillsync/restore.go) uses the same comparison/staged path | Prefer documentation and regression-test correction, not permission behavior changes. Account for umask; inspect backup/digest/rollback details before making mode guarantees. |
| [README safety model](../../README.md#safety-model) claims permissions participate in exact change detection | PQ-04 is a demonstrated documentation contradiction. |
| [checks.json](../../harness/checks.json) defines six required Go/build/diff checks; [CI](../../.github/workflows/ci.yml) calls the base Taskfile/isolated bootstrap | Supplement with separately owned release QA, not an unreviewed expansion of the trusted required contract. |
| [.goreleaser.yml](../../.goreleaser.yml) distributes only the Darwin universal installer, with embedded skills and version ldflags | `sdd-check` cross-builds do not make it a published executable. Native macOS/Homebrew evidence is a separate gate. |
| `git ls-files -- backups bin` returns no tracked paths; `git check-ignore -v backups bin` identifies existing ignore rules | Retain ignored local content untouched. This does not establish its sensitivity or retention eligibility; do not read backup payloads to draft a plan. |
| Existing untracked [docs/README.md](../../docs/README.md) already indexes current/history material | Inspect and extend only with authorized provenance; do not overwrite it with a second index. |

CodeGraph reported no usable project index; use targeted local reads/search. No production checks, native probes or hosted run were executed for this plan.

## 2. Design contracts

### C1 — Authority, scope and current-state ownership

Keep the 18 portable identities and existing Go/skill/spec/harness/integration boundaries. Feature 008 owns only supported-core quality and release evidence. `spec.md#approval` owns combined approval; `tasks.md` owns batch state. Current-facing summaries link to the owning 004–007 approval/continuation, not a new status ledger. Mark old statements historical or add dated reconciliation without erasing approvals, accepted partial 005 delivery, findings or original test commands. Source-only/native/platform limits must be consistent across README, CLI help where relevant, integration docs and draft release notes.

B1 inventories tracked, ignored and untracked paths by metadata and provenance. It classifies docs/history/test entry points and checks references before proposing moves. Default: no moves/deletions, preserve backups/bin and existing local docs. A justified move needs consumer/link/embed inventory, compatibility note, focused tests and a scoped reverse patch. Private handovers/raw traces do not enter release inputs. If ownership is uncertain, leave the path unchanged and record the blocker.

### C2 — Existing instructions and user experiences, not another workflow

Compare the canonical router/human-input, implement, debugging, verification-review and helper instructions against R2/R3 before editing. `human-input.md` already contains ready-frontier, fact-discovery and three-question rules; avoid restating them in the router. Refine only demonstrated missing/contradictory branches. Implementation retains one completion-review owner; verification supplies findings and no inferred approval.

Add bounded fixture/source coverage to the existing harness scenario/test ownership, with independent expected outcomes for:

- Incomplete prompts with discoverable facts, unavailable stakeholder intent, dependent questions and already settled answers; a clear request with no mandatory interview.
- Direct fixes without feature paperwork; feature drafting versus batch execution; research and limited-document side effects; optional teaching; review-only without mutations.
- Bare `continue`, stale handover, agent `Ready`, accepted partial delivery, green checks and changed scope: none creates missing batch/live/spend authority. A clear named batch authorization is a positive control, not something to reject universally.
- Completion output covering changed behavior, actual checks, limitations, risk-relevant review points and next permitted action, without a second mandatory review loop.

Static/source checks establish instruction coverage only. Scripted fixtures establish consumer behavior only. Keep actual-model observations `not_run`; no comparative cost/reliability claim or model call is needed for this feature. Walk through the three compact R8 examples against these contracts and link examples to the canonical rules rather than copying full skills. Teaching remains opt-in.

### C3 — Accurate existing installer permission semantics

Do not change default mode/no-op/restore behavior to match the inaccurate README. Specify structure/content equality, permission-only no-op (including no new backup), requested installation modes and umask effects. Trace backup `copyTree`, manifest digest, staged install, rollback and restore separately: backup copying mode bits is not a promise of exact restoration, and root/child mode handling may differ.

Use isolated test homes for embedded and `--repo` sources; cover differing modes with identical content, actual content replacement, install/restore idempotence, backup-before-write, failed staging/replacement and unchanged unrelated skills. Add missing assertions test-first and reuse existing valid failure tests. Do not weaken permissions or add chmod-on-no-op. A demonstrated safety defect requiring changed behavior returns for compatibility/scope review before implementation; document-only intent is not permission to redesign filesystem semantics.

### C4 — Core-only default gate, separate release QA, reviewed adoption

After approval, normal `task test`/`task vet` target the core; `task verify`, configured hooks, CI and release tasks agree. Retain `test-core`/`vet-core` as compatible aliases or parity-enforced wrappers, not independent scope definitions. `harness/checks.json` remains the owner of mandatory selection. Preserve its version/status schema and six roles (`mod-verify`, `tidy-check`, `vet`, `test`, `build-darwin`, `diff-check`); revise test/vet argv/env/local recipes deliberately rather than preserving literal `./...` breadth.

Proposed core Go selection: module root `.`, `./internal/version`, `./internal/skillsync`, `./internal/harnesscheck`, `./internal/harnessevidence`, `./cmd/agentic-sdd` and `./cmd/sdd-check`. Root/version compile/vet integrity is not mislabeled as behavioral test coverage; CLI/package and later artifact checks assert metadata/embeds. These are the initial explicit reviewed candidates, not a feature-number filter. T06a must confirm their dependency/capability mapping and reconcile a material change before implementation. `internal/opencodeadapter` behavioral suites and experimental CLI tests remain outside default selection.

Use literal, reviewed Go package selections in the canonical required records. Local wrappers mirror/derive that selection with enforced parity and independent expected-core assertions; do not create another gate catalog. Do not replace trusted CI with execution of a candidate's Taskfile or `supported-core-go.sh` on the host. Preserve full trusted/candidate contract equality, base-owned bootstrap/selection/probes, isolated candidate execution and failed/incomplete/missing-prerequisite semantics. Keep the current independent no-op probe and extend the existing reviewed-base probe mechanism narrowly for default-inactive boundary protection. Deleting candidate tests, emptying tasks or forging a receipt must not make a core defect/activation mutant pass.

Probe coverage is for named defects/activation cases and execution boundaries, not universal adequacy or arbitrary conditional cheating. Observe that the independent checks execute, and retain human review of changed gate/test paths. Do not turn a successful execution result into technical Ready or human acceptance automatically.

Adoption sequence: review the exact new definitions and bootstrap/probe changes; validate positive/negative cases in labeled disposable development-reference fixtures; obtain actual human approval of the trusted revision/scope; separately authorize any live trusted-branch adoption/push. The old base is expected to reject unadopted definitions. Never disable comparison or treat a candidate-local copy as an already trusted live base. G-008-01 remains open for actual gate claims until matching reviewed-base/candidate definitions and fresh core evidence exist. Simulated adoption is not hosted observation. This is forward 008 adoption, not replaying 007's historical transition.

Remove experimental targets such as the current helper entry from the ordinary Task interface/dependency graph. If retained, place them in an explicit `Taskfile.experimental.yml` that the root Taskfile does not automatically include. No default task, hook, build, verification or release target invokes it. Creating opt-in instructions does not authorize invocation; 008 does not execute those suites. Update AGENTS' temporary r3 wording only with the approved implementation, so it accurately describes the adopted policy rather than contradicting active commands.

Keep complete release QA distinct from the required Go gate. Explicitly select safe core Python cases, installer E2E, packaging and native first-use in the candidate matrix; do not use blanket discovery, run historical adoption cases, or blindly insert Python into a Go-only worker image. Developer-local trusted tests may use sanitized environments/temporary homes, but those are not an OS sandbox. Untrusted candidates run only in the reviewed credential-free non-root/no-egress boundary without host home/socket/privileged mounts. Missing safe tools/images block their rows, never trigger installation or fallback.

The matrix remains in `release-readiness.md`, not another approval ledger. Each row identifies suite/command, CI vs isolated QA vs manual, exact candidate/tree/tool/config identity, result/evidence and limits. Excluded behavior is listed as out of scope/not_run, not pass; failed supported-core checks cannot be reclassified as experimental. New/unclassified packages, tests, assets or runtime effects stop for scope review. Existing B3 results retain their original selection/identity and cannot certify this new gate.

| Surface | Planned verification / owner |
|---|---|
| Go/module/format and required gate | Aligned core `task test`/`task vet`/`task verify`, compatible core aliases, module checks and non-mutating gofmt. Six required roles remain; new definitions need adoption and fresh evidence. |
| 18 skills, references and embeds | `internal/skillsync/package_test.go` and focused canonical-reference checks; exact expected ID set, missing-reference negative control, embedded/on-disk installs. |
| Installer/restore and permission contract | Focused skillsync/CLI tests plus `task docker-e2e` with available local image, disposable homes and negative/no-op/rollback checks. |
| Taskfile/workflow | Explicit current-command/selection tests, excluding historical migration/adoption cases; verify ordinary help/dependency graph, aliases, hooks and trusted-base entry. Core versus experimental selection must not drift. |
| Runner/receipts/recovery | Go harnesscheck/harnessevidence tests; selected `test_receipts.py`, `test_recovery.py`, `test_connected.py`; `validate_contracts.py` only with its existing dependency. |
| Shipped portable method and exclusion warnings | Core workflow/reference checks and changed guidance scenarios from C2 only. Do not select 006 experimental profile/native/dispatch/pilot suites or use them as a method-validation project. Package-integrity checks retain shipped references without certifying deprecated functionality. |
| Trusted isolation/anti-weakening | Explicit core policy/container cases in the reviewed boundary: contract weakening, empty candidate task, core-test deletion, forged receipt, no-op defect and default-activation mutant. Do not select old 007 exercises. Reference fixtures do not prove actual trusted adoption. |
| Packaging/metadata | Default core-only source/snapshot/release profile; inspect archive/checksums/dependencies/embeds/ldflags and disabled helper behavior on the identical artifact. No publishing token or local artifact overwrite. |
| First run and public route | Isolated Linux source smoke and isolated first-run lifecycle; macOS universal architecture cross-build and packaging integrity verified. Per 2026-11-07 amendment, Linux CI runner / isolated Linux execution evidence is sufficient for candidate release qualification; native macOS/Homebrew runtime execution remains unobserved (not pass). Linux local QA is NOT hosted CI observation. |
| Hosted CI | Read exact SHA/run results for the maintained core candidate if available; distinguish main push, same-repo PR, divergent/fork PR and release workflow. Do not reopen 007 adoption/transition work. Unobserved paths stay unobserved; no branch-rule/PR/push creation is implicit. |
| Online model behavior | `not_run`; separate future authorization only. No model-quality or savings inference. |

B4a–c finalize/verify these definitions in separate reviewed batches. Missing prerequisites or failed relevant rows block their claims; all promised-core release-critical rows are needed for go. A core gate passing is not itself the release go/no-go or publication permission.

### C6 — Minimum default-inactive CLI/build boundary

Prefer a narrow namespaced build-tag pair (`agentic_sdd_experimental`) around the existing CLI helper, not a new runtime engine. Proposed files: the existing `cmd/agentic-sdd/opencode.go` and its experimental test file become opt-in; a small default-built `opencode_disabled.go` supplies the existing `runOpenCode`/`printOpenCodeUsage` signatures. Default experimental actions return exit 2 with a clear unavailable diagnostic before any configuration, credential, provider or process work. Targeted help may explain unavailability without claiming a working capability. Adapt ordinary `printUsage` to show only supported commands by default. Core flags, destinations, preview/apply/restore and all 18 portable identities remain unchanged.

This gates the installer's experimental helper, not OpenCode's use of the portable skills: the shared skill destination and ordinary client workflow remain core.

Default package builds/tests must not import the experimental backend. Boundary tests use synthetic canaries and denial/help/dependency observations only—no native executable, provider, credentials, model or experimental suite. Also verify root Task discovery/dependencies and release/source build commands cannot enable the tag through inherited `GOFLAGS`, foreign workspace/config or an automatic include. Experimental source may remain opt-in, but 008 does not build/run its behavioral suites or certify it.

**Do not reuse B3's explicit CLI file list unchanged:** named `.go` files ignore build constraints. Replace it with approved core package selection after tagging experimental CLI tests and adding default-boundary tests; prove the selected core file/dependency set excludes the experimental implementation/tests. Update the selector regression tests intentionally, retaining independent coverage/failure oracles rather than deleting assertions to obtain green.

Keep root embed paths and necessary receipt schema/reference assets. `AdapterFiles` is inert by declaration; confirm no core operation extracts/activates experimental resources or writes client discovery configuration. Inventory retained inert resources in the artifact. If the narrow CLI split does not cut an actual activation/import path, return that concrete mapping for a small revised boundary before broader changes. No separate module/package relocation is authorized merely to tidy the tree.

Classify changed availability of the experimental CLI/task as a compatibility change with release/migration notes. Default builds, source usage, GoReleaser snapshots and published artifacts all use the same core profile; Linux CI runner / isolated Linux evidence is sufficient for candidate release qualification, Darwin cross-build integrity remains verified, and native macOS execution remains unobserved. Runtime configuration toggles are not a substitute for the proposed compile-time default or for technical containment of any future experiment.

### C5 — Candidate identity, compatibility and go/no-go

After B1–B4 review, request owner disposition of unrelated dirty/untracked inputs and a clean committed candidate. Do not stage, commit, stash, remove or reset them under this plan. A sanitized export may support preliminary QA, with explicit inclusion/exclusion manifest and digest, but cannot be presented as the final clean Git candidate. Commit/push/tag/publication retain their separate authorization requirements.

Freeze the candidate commit/tree and source/config manifest, trusted base identity, tool versions and snapshot version. Freeze the release payload separately from the later report so adding evidence does not create a self-referential hash. Any code, skill, build/config or packaging-input change invalidates affected evidence and requires a new identity and reruns; report-only edits must disclose the payload/report distinction.

Use existing GoReleaser and Homebrew configuration, no new distribution channel. Validate hooks/tool requirements before invoking snapshot; it runs `go mod tidy` and `--clean` can remove output, so use disposable candidate storage and no publishing token. Native macOS availability is not established: per 2026-11-07 owner authorization, native macOS runtime evidence is dropped as a mandatory release qualification requirement in favor of Linux CI runner / isolated Linux evidence sufficiency. Darwin cross-build compilation and packaging integrity (universal binary, embedded skills, ldflags, no experimental dependencies) remain required; native macOS/Homebrew execution remains unobserved, not pass. Linux local QA is NOT hosted CI observation (hosted CI remains unobserved until run on GitHub Actions). Never represent the previous published binary as testing this candidate. No actual release workflow dispatch or tag is authorized.

Prepare `release-readiness.md` for the maintainer's decision: bounded before/after summary, compatibility/path inventory, candidate verification matrix, known blockers and excluded experimental findings, justified next `0.x` recommendation, draft release notes and go/no-go against spec §7. The Task migration's breaking developer-command change needs an explicit compatibility note even if runtime interfaces remain unchanged. A go verdict is technical readiness, not acceptance/publication authority. Per the 2026-11-07 amendment, Linux CI runner evidence qualifies candidate first use, while Darwin cross-build packaging integrity remains verified and native macOS execution is recorded as unobserved (not pass).

## 3. Affected components and rollback

- Guidance: targeted `skills-files/agentic-sdd-*/` references/instructions; preserve IDs, linked assets/notices and conditional disclosure. Existing instructions may need no change where evidence shows no gap.
- Navigation/state: README/help, AGENTS, docs index, `docs/getting-started.md` and CONTRIBUTING. Documented source commands must use the same core profile and flag protection as Task/release commands. B1's older-feature summaries are delivered history, not further 006/007 assignments. Security-reporting guidance stays accurate; no ceremonial new policy is needed.
- Installer: README/help and focused `internal/skillsync`/CLI tests; permission/install/restore behavior unchanged absent reviewed escalation. C6 changes only experimental CLI availability, not core filesystem behavior.
- Verification/release: normal/opt-in Taskfiles, selectors/tests/docs, `harness/checks.json`, configured hooks, reviewed-base probes/bootstrap where needed, CLI build-tag boundary and GoReleaser core profile. Workflow code changes only if necessary; preserve existing base/candidate separation and privileges. Implementation/adoption remain unauthorized until reviewed.
- Evidence: this feature's `release-readiness.md`; existing tasks and one session handover own continuation. No automatic extra ADR, roadmap or status ledger.

Migration is narrow core-default command/gate/build alignment, not a repository move or real-user upgrade. Rollback must restore a mutually consistent reviewed contract, local commands/hooks and core-inactive build boundary; never restore an active experimental default merely to obtain green. Live trusted-base rollback is separately authorized. Use a scoped reverse diff, preserving inherited work and historical evidence. Real-home apply/rollback and any broad path move remain outside this amendment.

## 4. Specialist and tool assignments

All named skills are available in the active catalog; load guidance in the main agent only when its scope begins. Only the bounded B1 and later explicit B3 delegations recorded in spec §Approval are permitted; no automatic product orchestration, model selection or installation. B3's parent also loads agent-writing guidance for the requested instruction scope and keeps runtime/CI trust boundaries unchanged.

| Work | Principal guidance | Tools and reason |
|---|---|---|
| Agent-consumed workflow/state docs (B1/B2) | `agentic-sdd-writing-for-agents` | Targeted local reads/search and branch/reference checks; refine existing owners without duplicate rules. |
| Installer contract/tests (B3) | `golang-pro`; `test-driven-development` before test/behavior implementation | Repository Go/Task commands, isolated test homes; preserve standard-library architecture. |
| Core CLI/build boundary (B4a) | `golang-pro`; `test-driven-development` | Narrow build-tag/stub/dependency checks, synthetic no-effect boundary cases; no experimental execution. |
| Required gate/hook/adoption changes (B4b) | `security-auditor`; `agentic-sdd-architecture-review` as consequential review guidance | Verify base-owned selection/probes, unchanged privilege/isolation guarantees and explicit adoption; technical verdict is not human approval. |
| Complete core release QA (B4c) | `test-automator`; `test-driven-development` for new executable coverage | Explicit suites, Docker only within reviewed boundary, meaningful negative controls and candidate identity. |
| Inventory, public onboarding, final assessment (B1/B5) | No additional specialist needed beyond repository/SDD guidance unless editing agent instructions | Git metadata/provenance, link checks, isolated CLI/package smoke and evidence comparison. |

Context7 is conditional for exact Task/GoReleaser/GitHub/Homebrew API/config questions; Exa/official sources only for named gaps it cannot answer. Discover actual tool identifiers at use. GitHub CLI/API is read-only for hosted evidence when permitted; no auth setup or credential reads. Missing local tools/images/native macOS remain explicit blockers, not installation authority. Browser tools are unnecessary unless a concrete public-navigation check requires them.

## 5. Risks and decisions deferred to their ready checkpoint

| Risk / unresolved decision | Mitigation / blocking point |
|---|---|
| Dirty 006 work or private docs silently shipped | B1 provenance and B5 candidate gate; owner resolves uncertain inclusion, not the drafting agent. |
| Historical acceptance upgraded to safety certification | Keep 004/006 findings and 005 partial scope; state reconciliation is not new approval. |
| Permission claims exceed umask/restore behavior | B3 source/test trace, documentation-first; behavioral change requires review. |
| Core-only selection weakens coverage/trust | Preserve core safety/module/build coverage and equality/independent probes; change exact argv only through reviewed adoption. |
| Default tags/file-list mode reactivate experiments | C6 package selection, sanitized profile and positive/negative default-boundary checks on actual artifacts. |
| Partial adoption creates local/CI/hook mismatch | B4a leaves the old aggregate blocked; B4b aligns/adopts definitions before claiming required success. No hook bypass or candidate-as-trusted shortcut. |
| Static checks called behavioral success | Label evidence classes; no online inference; model cases remain `not_run`. |
| macOS native runtime unavailable | Per 2026-11-07 amendment, dropped as release blocker; Linux CI runner / isolated Linux evidence is sufficient; Darwin cross-build verified; native macOS execution unobserved (not pass). |
| Version or release chosen prematurely | Recommend only after compatibility/evidence review; human selects version and separately authorizes live actions. |

No human clarification blocked drafting. The core-first direction and exact core contract are approved in spec §Approval. B4a–B5 executed. G-008-01 remains open until a clean candidate commit is designated as the reviewed trusted reference base. Commit and push authorized by the owner; tagging and release publication remain separate decisions.
