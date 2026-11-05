# Core release quality — plan

Revision: r2, 2026-11-06 — owner-requested post-B1 scope clarification. Input: [spec r1](spec.md). Combined r1 approval, B1 acceptance and commit authority: [spec §Approval](spec.md#approval). B2 is not authorized; review this clarification before authorizing it. This design does not itself authorize publication or later batches.

## 1. Approach and inspected baseline

Deliver one release-readiness outcome in five reviewable batches: establish provenance/state; close demonstrated portable-guidance gaps; align installer semantics; integrate safe release QA; assess one frozen candidate. A supported-core no-go is a valid result. Do not expand into another runtime/orchestration project.

Scope constraint confirmed by the user during drafting: strictly follow Feature 008 and exclude its out-of-scope components. Work on 004/005/006/007 is limited to the current-facing reconciliation, retained findings, distribution integrity and supported-core verification explicitly required by 008. No experimental implementation, repair, pilot, native dispatch, C2 certification or retrospective completion of those features is included.

### Deprecated-feature work fence (r2)

The owner accepted B1 and explicitly directed that **deprecated Feature 006 and Feature 007 scopes are excluded before B2**. Their historical records remain evidence, not work queues.

- Do not resume, implement, repair, complete, certify or execute either feature. Do not investigate 006 pilot/mitigation/runtime/provider controls, run its experimental fixtures, or reopen 007's old-base adoption/transition work. B1's limited status/warning reconciliation is delivered, not an ongoing assignment to those features.
- Saving the owner's existing 006/007 changes as WIP commits is a separately authorized preservation action. It does not approve their behavior, include them in the supported product, waive findings, or authorize more work.
- Preserve all 18 portable skill IDs and packaged references. Deprecating feature work does not authorize deleting shipped portable guidance, changing the installer, or removing its package-integrity checks.
- Validate the **maintained supported core** directly under 008 R6/R9: current Taskfile commands, installer/restore, packaged skills, the check runner and reviewed-base trust boundary. This is not continuation of 007. Retain the six required-check contract and meaningful anti-weakening coverage; do not replace them with deprecated-feature tests.
- Future source/fixture selection must exclude 006 native/profile/dispatch/pilot suites and 007 historical migration/transition exercises. Core Taskfile regression tests remain eligible where they verify current supported commands, not old-feature completion. Historical findings can be cited only to explain exclusions; no mitigation design or additional experimental research is implied.

Removing required core Taskfile/trust verification would conflict with the approved spec; this clarification excludes deprecated feature work, not those release-critical core checks. Any broader removal needs an explicit spec-scope decision rather than silent weakening.

There is no top-level SRS, technical specification or global roadmap; repository `AGENTS.md` and the feature records are the established sources. The owner-supplied audit is input, not fresh verification. Findings below are source observations on 2026-11-05, not acceptance passes:

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

### C4 — Release QA distinct from the trusted required gate

Keep the six required IDs, argv, scope and equality contract unchanged by default. Build an explicit allowlisted model-free release-QA entry in `Taskfile.yml` and existing test tooling after inspecting each suite's subprocess/network/file effects. Do not use blanket Python discovery or execute opt-in native/online scripts by filename. Document which suite owns which surface, prerequisites, exclusions and failure semantics in `tests/harness/README.md` and contributor guidance.

For trusted local development, selected synthetic suites may run with sanitized environments and temporary homes; that is not a sandbox claim. For untrusted candidates, scripts/tests execute only inside an independently reviewed, credential-free, non-root/no-egress boundary with no host home/socket or privileged mounts. The host runs reviewed-base bootstrap only. New QA needing a different Python/image capability is separately isolated and reviewed; absence blocks that row, never triggers download/fallback or host execution. No new required contract/CI trust adoption is implicit in this plan.

The release matrix is a section of `release-readiness.md` in this feature, not another approval or task ledger. Record per row: suite/command, required CI vs isolated release QA vs manual, exact candidate/tree and relevant tool/config identities, environment/platform, evidence path/digest, `pass / fail / blocked / not_run`, negative controls and limits. Retain raw logs externally and privately; include only sanitized observations. Treat output generated by a test as evidence, never consent.

| Surface | Planned verification / owner |
|---|---|
| Go/module/format and required gate | `task test`, `task vet`, `task verify`; non-mutating gofmt check; existing module checks. Required gate remains six checks, not complete release QA. |
| 18 skills, references and embeds | `internal/skillsync/package_test.go` and focused canonical-reference checks; exact expected ID set, missing-reference negative control, embedded/on-disk installs. |
| Installer/restore and permission contract | Focused skillsync/CLI tests plus `task docker-e2e` with available local image, disposable homes and negative/no-op/rollback checks. |
| Taskfile/workflow | Explicit `test_taskfile.py`; inspect workflow pins and trusted-base entry; seeded failure/effect-boundary regressions. |
| Runner/receipts/recovery | Go harnesscheck/harnessevidence tests; selected `test_receipts.py`, `test_recovery.py`, `test_connected.py`; `validate_contracts.py` only with its existing dependency. |
| Shipped portable method and exclusion warnings | Core workflow/reference checks and changed guidance scenarios from C2 only. Do not select 006 experimental profile/native/dispatch/pilot suites or use them as a method-validation project. Package-integrity checks retain shipped references without certifying deprecated functionality. |
| Trusted isolation/anti-weakening | `test_trusted_ci.py` and explicitly opted-in `test_trusted_ci_container.py` using the documented wrapper and reviewed fixtures; required-contract deletion/replacement, empty candidate task, test deletion, forged receipt and seeded no-op regression. Frozen fixtures do not prove current base/candidate transition. |
| Packaging/metadata | Nonpublishing GoReleaser validation/snapshot in disposable candidate storage; inspect archives/checksums/embedded content/ldflags; no overwrite of local `bin/` or `dist/`. |
| First run and public route | Isolated Linux source smoke; macOS universal architecture and actual native Homebrew/formula/help/version/preview plus isolated-home install/restore. Cross-build/inspection is not native smoke. |
| Hosted CI | Read exact SHA/run results for the maintained core candidate if available; distinguish main push, same-repo PR, divergent/fork PR and release workflow. Do not reopen 007 adoption/transition work. Unobserved paths stay unobserved; no branch-rule/PR/push creation is implicit. |
| Online model behavior | `not_run`; separate future authorization only. No model-quality or savings inference. |

B4 finalizes exact safe commands/allowlist from the actual suites. A missing prerequisite or failed relevant row blocks the corresponding claim; all promised-core release-critical rows must be satisfied for go. Ordinary QA must fail nonzero on required missing/failing rows, not silently skip them into green.

### C5 — Candidate identity, compatibility and go/no-go

After B1–B4 review, request owner disposition of unrelated dirty/untracked inputs and a clean committed candidate. Do not stage, commit, stash, remove or reset them under this plan. A sanitized export may support preliminary QA, with explicit inclusion/exclusion manifest and digest, but cannot be presented as the final clean Git candidate. Commit/push/tag/publication retain their separate authorization requirements.

Freeze the candidate commit/tree and source/config manifest, trusted base identity, tool versions and snapshot version. Freeze the release payload separately from the later report so adding evidence does not create a self-referential hash. Any code, skill, build/config or packaging-input change invalidates affected evidence and requires a new identity and reruns; report-only edits must disclose the payload/report distinction.

Use existing GoReleaser and Homebrew configuration, no new distribution channel. Validate hooks/tool requirements before invoking snapshot; it runs `go mod tidy` and `--clean` can remove output, so use disposable candidate storage and no publishing token. Native macOS availability is not established: obtain manual/authorized CI evidence for the identical payload, or report blocked/no-go for the advertised route. Never represent the previous published binary as testing this candidate. No actual release workflow dispatch is authorized.

Prepare `release-readiness.md` for the maintainer's decision: bounded before/after summary, compatibility/path inventory, candidate verification matrix, known blockers and excluded experimental findings, justified next `0.x` recommendation, draft release notes and go/no-go against spec §7. The Task migration's breaking developer-command change needs an explicit compatibility note even if runtime interfaces remain unchanged. A go verdict is technical readiness, not acceptance/publication authority. Missing native packaging/first-run proof cannot be waived by calling an existing supported surface experimental.

## 3. Affected components and rollback

- Guidance: targeted `skills-files/agentic-sdd-*/` references/instructions; preserve IDs, linked assets/notices and conditional disclosure. Existing instructions may need no change where evidence shows no gap.
- Navigation/state: README, AGENTS, existing docs index and CONTRIBUTING. B1's 005/006/007 current-facing summary and exclusion-warning edits are delivered historical work; no further 006/007 assignment remains. Security-reporting guidance must be accurate; add a dedicated policy only if it has a real maintained owner/channel.
- Installer: README/help and focused `internal/skillsync`/CLI tests; production behavior unchanged absent reviewed escalation.
- Verification/release: Taskfile, test entry points/ownership docs, focused fixtures; `.github`/GoReleaser edits only for demonstrated release-critical gaps, with trust/compatibility review before changed execution boundaries.
- Evidence: this feature's `release-readiness.md`; existing tasks and one session handover own continuation. No automatic extra ADR, roadmap or status ledger.

Default migration is documentation/test/task additions, not a repository move or real-user upgrade. Rollback uses a reviewed reverse diff limited to Feature 008-owned hunks; never overwrite inherited edits. Temporary-home tests prove installer backup/restore independently; real-home rollback/apply is not exercised. Any path move needs its own explicit impact/recovery mapping before execution.

## 4. Specialist and tool assignments

All named skills are available in the active catalog; load guidance in the main agent only when its scope begins. Delegation is permitted only within the owner's explicit B1 request recorded in spec §Approval; no automatic product orchestration, model selection or installation.

| Work | Principal guidance | Tools and reason |
|---|---|---|
| Agent-consumed workflow/state docs (B1/B2) | `agentic-sdd-writing-for-agents` | Targeted local reads/search and branch/reference checks; refine existing owners without duplicate rules. |
| Installer contract/tests (B3) | `golang-pro`; `test-driven-development` before test/behavior implementation | Repository Go/Task commands, isolated test homes; preserve standard-library architecture. |
| Release QA and negative fixtures (B4) | `test-automator`; `test-driven-development` for new executable coverage | Explicit suite execution, Docker only within reviewed boundary, diff and failure probes. |
| Any CI privilege/isolation change | `security-auditor`, crossing the B4 trust boundary; use `agentic-sdd-architecture-review` before consequential redesign | Inspect base/candidate execution, pins, mounts and secrets; no weakening/adoption without scoped review. |
| Inventory, public onboarding, final assessment (B1/B5) | No additional specialist needed beyond repository/SDD guidance unless editing agent instructions | Git metadata/provenance, link checks, isolated CLI/package smoke and evidence comparison. |

Context7 is conditional for exact Task/GoReleaser/GitHub/Homebrew API/config questions; Exa/official sources only for named gaps it cannot answer. Discover actual tool identifiers at use. GitHub CLI/API is read-only for hosted evidence when permitted; no auth setup or credential reads. Missing local tools/images/native macOS remain explicit blockers, not installation authority. Browser tools are unnecessary unless a concrete public-navigation check requires them.

## 5. Risks and decisions deferred to their ready checkpoint

| Risk / unresolved decision | Mitigation / blocking point |
|---|---|
| Dirty 006 work or private docs silently shipped | B1 provenance and B5 candidate gate; owner resolves uncertain inclusion, not the drafting agent. |
| Historical acceptance upgraded to safety certification | Keep 004/006 findings and 005 partial scope; state reconciliation is not new approval. |
| Permission claims exceed umask/restore behavior | B3 source/test trace, documentation-first; behavioral change requires review. |
| QA breadth weakens trusted-base isolation | B4 separate allowlisted QA; preserve required contract; stop on unavailable safe runtime. |
| Static checks called behavioral success | Label evidence classes; no online inference; model cases remain `not_run`. |
| macOS or exact packaging prerequisites unavailable | B5 blocked/no-go, not cross-build substitution. |
| Version or release chosen prematurely | Recommend only after compatibility/evidence review; human selects version and separately authorizes live actions. |

No human clarification blocked drafting. Current combined approval and B1-only authority are recorded in spec §Approval; this plan does not independently grant them. Each batch stops for review; later batches and live operations need their own authorization.
