# Core release quality — tasks and continuation

Revision: r2, 2026-11-06 — owner-requested post-B1 exclusion clarification. Inputs: [spec r1](spec.md), [plan r2](plan.md). Single approval owner: [spec §Approval](spec.md#approval). Shared contracts C1–C5 and specialist/tool assignments live in the plan. B1 r1 delivery is accepted; B2 remains unauthorized pending review of this clarification and explicit batch authorization.

## Execution rules

Execute only the batch authorized in spec §Approval, verify it, record `awaiting human review` and stop. Acceptance, `Ready`, bare `continue`, green checks and prior-feature authority do not start later batches. The owner's explicit delegation request covered B1 only. Separate post-B1 authority covers the WIP and B1 commits recorded below, not a push. No product model evaluation, native dispatch, credential/config change, real-home apply, tag/publication or branch-rule change is included.

Task/batch progress is recorded below; unstarted tasks remain pending. Existing dirty/untracked work is preserved. Every completion records changed paths, exact source/evidence identity, actual commands/results, omissions and next permitted action. Changes to requirements, compatibility or trust boundaries return the affected package for human review.

The user's drafting correction explicitly requires strict Feature 008 scope and exclusion of out-of-scope components. References to 004–007 below are dependencies/status and supported-core evidence only, not authorization to resume or complete them. In particular, exclude 006 B4, native ACP/reviewer execution, C2 certification, orchestration and model evaluations.

**Post-B1 owner direction:** deprecated 006/007 feature work is excluded from every remaining batch. B1's status reconciliation is historical completed work, not a standing task to revisit those features. Do not investigate or run 006 experiments or repair/replay 007 migration/adoption transitions. Separate WIP commits preserve existing changes only. Current core Taskfile commands, package/installer integrity and reviewed-base trust verification remain required under 008 R6/R9; verify them as core surfaces, not as old-feature completion. The plan's deprecated-feature fence governs all future task/test selection.

## Ordered tasks

### T01 — Establish provenance, candidate blockers and test ownership

- **Status:** delivered; B1 accepted by the owner on 2026-11-06. See spec §Approval and completion checks below.
- **Contract / coverage:** C1/C4/C5; AC-09/10/11/14/17. **Dependencies:** none.
- Inspect Git status/HEAD, tracked/ignored/untracked metadata for docs/history/audits/tests/backups/bin and root embed consumers. Do not read sensitive backup payloads or delete local artifacts. Compare suspected duplicate docs before proposing consolidation; inventory test entry points/effects, not just filenames.
- Establish exact 18-ID/reference/embedded ownership and release/tool inputs; identify inherited 006 hunks and private docs that cannot be silently included. Record findings in the provenance/ownership sections of `release-readiness.md`, with links rather than copied history.
- **Verify / done when:** metadata commands reproduce dispositions; every proposed changed path has an owner and every proposed move has consumers, migration/recovery and link coverage. Default no moves. List missing candidate/macOS/tool prerequisites with the claim each blocks. Inventory does not certify the candidate or mark suites passed.

### T02 — Reconcile current state and support/navigation contracts

- **Status:** delivered; B1 accepted by the owner on 2026-11-06. Only core status/navigation and exclusion-warning documentation changed; no experimental route was repaired or tested. No further 006/007 reconciliation work is scheduled.
- **Contract / coverage:** C1; AC-01/06/07/09/18. **Dependencies:** T01.
- Reconcile misleading current-facing 005/006/007 summaries and AGENTS against actual approvals/continuations and Git history. Retain original dated evidence, accepted partial 005 scope, 004 findings, 006 B4 deferral/IR-06-01 and 007-F1 direct-adoption gap.
- Create/update a concise support/platform/artifact matrix and navigation in existing public docs. Reuse the inherited docs index only after provenance inspection; no overwrite. Make unsupported native review and inactive ACP limits prominent. Do not settle other features' open decisions.
- **Verify / done when:** each support/status claim cites its owner/source; inspect current headings and links for contradiction; all findings and actual human decisions remain traceable. No claim of hosted/native success from static text.

### T03 — Cover authority and incomplete-request branches

- **Contract / coverage:** C2; AC-02/03/04/05/10/15/18. **Dependencies:** B1 reviewed; B2 authorized.
- Inspect canonical instructions and existing scenario/test ownership; make a gap-to-change map. Add bounded independent cases for incomplete versus clear prompts, settled answers, ready questions, all routing/side-effect branches and the positive/negative authority cases in C2. Refine existing instructions only where a real gap is demonstrated.
- Preserve one implementation completion-review owner and portable/host separation. No new skill, approval record, mandatory interview or automatic review loop.
- Use only core workflow/reference cases; do not expand into 006 profile/pilot/native/dispatch fixtures or 007 historical migration exercises. Preserve shipped reference links without claiming deprecated-feature behavior is verified.
- **Verify / done when:** targeted source/fixture tests detect seeded missing-rule/reference/authority regressions; exact 18 IDs and relevant references survive; review-only and direct-fix cases retain correct effects. Report source inspection, scripted consumer checks and actual-model `not_run` separately. New executable tests use the plan's test-first guidance.

### T04 — Deliver concise first use and three worked experiences

- **Contract / coverage:** C1/C2; AC-01/02/04/05/07/13/15/18. **Dependencies:** T03.
- Update README/current docs and CONTRIBUTING with safe prerequisites and a first-five-minutes path. Provide compact direct-fix, combined-approval/one-batch feature, and review/resume examples with exact stop points; link canonical rules and keep teaching optional.
- Make security-reporting guidance navigable using a real maintained channel. Do not invent private reporting availability or add ceremonial files. Public install instructions may show explicit opt-in apply, but validation uses isolated homes only.
- **Verify / done when:** repeatable model-free walkthroughs and link/command inspection match documented effects; negative resume examples do not grant continuation. Label instruction walkthroughs versus unrun real-agent behavior. Candidate-binary/macOS first-run checks remain T10, not claimed here.

### T05 — Establish and verify permission/no-op/restore semantics

- **Contract / coverage:** C3; AC-08/10/17. **Dependencies:** B2 reviewed; B3 authorized.
- Trace comparison, requested install modes, root/child umask effects, backup copy/digest, stage/rollback and restore. Add missing focused assertions before changes for embedded and on-disk sources using temporary homes; document exact semantics in README/help where relevant.
- Cover permission-only no-op/no backup, content replacement and repeat apply/restore, failed writes and backup-before-write. Reuse unchanged valid failure tests and add assertions only where coverage is absent.
- **Verify / done when:** focused skillsync/CLI tests, `task test`, `task vet`, non-mutating formatting inspection, `git diff --check` and available isolated installer E2E agree with documentation. No exact-mode-preservation claim beyond observations. Any needed behavioral/compatibility change blocks that portion pending scope review; no silent chmod redesign.

### T06 — Define the complete release verification matrix and safe entry point

- **Contract / coverage:** C4; AC-10/11/12/15/17. **Dependencies:** B3 reviewed; B4 authorized.
- Finalize exact suite allowlist, commands, environments and ownership from T01/C4. Separate ordinary six-check CI, isolated release QA, manual/native checks and excluded model/native experiments. Include all changed release-critical surfaces and negative controls; explain any excluded historical test.
- Apply the r2 deprecated-feature fence: no 006 WIP/native/profile/dispatch/pilot suites or 007 historical migration/adoption exercises. Current-command Taskfile and trusted-core checks remain required; preservation commits are not verification evidence.
- Add an explicit model-free Task release-QA entry and focused entry-point tests, using existing scripts/fixtures where appropriate. Inspect subprocess/environment effects before inclusion. Update test ownership/entry docs and contributor commands. Required missing/failing rows must fail, not disappear into skips.
- **Verify / done when:** demonstrate the new entry's failure for a seeded failing suite/missing required prerequisite and no execution of native/online routes; command arguments/effects and external evidence paths are tested. Existing six-check equality stays intact. Any new image or trust design is reviewed before candidate execution; unavailable prerequisites remain blocked.

### T07 — Verify maintained core Task and trusted-CI boundaries

- **Contract / coverage:** C1/C4; AC-11/12/15/18. **Dependencies:** T06. AC-06/07 status/exclusion work was delivered in T02, not reopened here.
- Run relevant current-command Taskfile/trusted-CI units and explicit core container anti-weakening fixtures in the approved boundary. Compare reviewed-base and candidate identities for the maintained core, not old 007 migration/adoption transitions. Inspect current release workflow/tool pins and record the scope of each observation. No deprecated feature or experimental fixture work is included.
- Obtain read-only hosted evidence for the exact core candidate SHA/run if accessible. Keep main-push, fork/PR and release-workflow outcomes separate; no 007 adoption project, new PR, push, workflow dispatch, branch-rule administration or credential setup.
- **Verify / done when:** negative controls detect required-contract replacement, candidate task/test weakening, forged evidence and known no-op regression; no candidate code executes on the privileged host. Report fixture success and hosted/transition gaps separately. Preserve 007-F1's historical disposition even if new main CI evidence passes.

### T08 — Run aggregate development QA and prepare candidate prerequisites

- **Contract / coverage:** C4/C5; AC-10/11/12/15/17. **Dependencies:** T06/T07.
- Run the safe selected Go/Python/reference/embed/installer/Task suites on an identified development tree; retain logs externally and update matrix results. Perform one batch completion review, without claiming final-candidate coverage from a dirty development run.
- Classify compatibility changes and path impacts, required reruns and prerequisites for clean-candidate/macOS validation. Identify the exact owner decisions needed for dirty-input disposition, candidate commit and native evidence.
- **Verify / done when:** every matrix row has a command/owner/class and a truthful result or blocked/not-run reason; actual failures remain open. Missing safe environments stop affected executions. B4 can be delivered with explicit blockers; no-go conditions remain for B5.

### T09 — Freeze a clean release payload and packaging inputs

- **Contract / coverage:** C5; AC-09/11/14/16/17/18. **Dependencies:** B4 reviewed; B5 authorized; owner-provided clean candidate/disposition.
- Confirm clean commit/tree, exact selected inputs, trusted base, versions and manifest; preserve unrelated work outside this candidate without deleting/stashing/resetting it. An export supports preliminary checks only, not a claim of clean final Git identity.
- Validate nonpublishing packaging config/snapshot in disposable storage; inspect archives, checksums, universal architectures, embedded 18 skills/references and metadata. GoReleaser hooks/cleanup must not overwrite local artifacts or use a publishing credential.
- **Verify / done when:** exact payload identity and build settings are reproducible and every artifact maps to it; `sdd-check` remains honestly source-only. Missing clean candidate, packaging tool or safe execution prerequisites blocks T09 rather than authorizing commits/downloads or bypasses.

### T10 — Verify the identical candidate and first-run paths

- **Contract / coverage:** C2/C3/C4/C5; AC-01/08/10/11/12/13/14/15/17. **Dependencies:** T09.
- Rerun release-critical matrix rows against the frozen payload, including required/isolated QA and Linux source first run. Obtain actual macOS/Homebrew candidate evidence (help/version/preview and isolated-home install/no-op/backup/restore) without publishing a formula or substituting an older released binary.
- Repeat all three worked paths as documented; distinguish manual/scripted instruction walkthrough from model compliance. Record platform/tool/config/evidence identity and all limits. No real-home skill apply or native check-runner certification.
- **Verify / done when:** required candidate evidence is fresh and identity-consistent, or the report explicitly records fail/blocked/not-run and resulting no-go. Cross-compilation is not native execution; subsequent payload changes invalidate affected evidence.

### T11 — Produce the compatibility, release-notes and go/no-go decision package

- **Contract / coverage:** C1/C5; AC-01/06/07/11/14/15/16/17/18. **Dependencies:** T09/T10 results, including blocked results.
- Complete `release-readiness.md`: before/after summary, path/compatibility inventory, exact payload versus report identity, verification matrix, supported-core blockers, excluded experimental findings, justified next `0.x` recommendation and draft release notes.
- Apply spec §7 release gates; unresolved supported-core blockers require no-go. Distinguish technical readiness, human acceptance and separately authorized publication. Version selection stays with the owner; no tag or live release is part of B5.
- **Verify / done when:** each AC is traced to actual evidence or a named unmet condition; report and packaged claims agree; risk acceptance has an actual named human decision or remains pending. Return one completion review and stop. A completed no-go report does not mark unmet release ACs passed or claim full feature acceptance.

## Execution batches

| Batch / included tasks | Outcome and completion verification | State / next action |
|---|---|---|
| **B1 — T01–T02** | Provenance/ownership and reconciled support/current-state navigation. Verify metadata, source-backed claims, retained approvals/findings and links; no moves/deletions by default. | Delivered; **accepted by the owner, 2026-11-06** (spec §Approval). Technical verdict: Ready for this documentation/source-inventory scope, not release readiness. Separate WIP/B1 commits and r2 exclusions were requested; B2 remains unauthorized. |
| **B2 — T03–T04** | Bounded existing-guidance improvements, authority scenarios and three onboarding experiences. Focused instruction/reference/fixture checks and model-free walkthroughs; model behavior not_run. | Not started. Requires B1 review and B2 authorization; then stop awaiting human review. |
| **B3 — T05** | Accurate permission/restore contract with focused safety/idempotence evidence. Repository Go gates and isolated installer tests; behavioral escalation reviewed before action. | Not started. Requires B2 review and B3 authorization; then stop awaiting human review. |
| **B4 — T06–T08** | Explicit safe release QA, trusted-CI negative evidence and candidate prerequisite list. Run selected suites and entry-point failures; preserve base-owned isolation and declare gaps. | Not started. Requires B3 review and B4 authorization; then stop awaiting human review. |
| **B5 — T09–T11** | Same-candidate packaging/first use, final matrix, compatibility/notes and technical go/no-go. Clean-candidate and native prerequisites gate dependent claims; honest no-go allowed. | Not started. Requires B4 review and B5 authorization plus candidate prerequisites. Stop for human review; never publish automatically. |

## Acceptance reconciliation

| Acceptance criteria | Owning tasks |
|---|---|
| AC-01 | T02/T04/T10/T11 |
| AC-02/03/04/05 | T03/T04 (AC-03 principally T03) |
| AC-06/07 | T02/T07/T11 |
| AC-08 | T05/T10 |
| AC-09 | T01/T02/T09 |
| AC-10 | T01/T03/T05/T06/T08/T09/T10 |
| AC-11/12 | T06/T07/T08/T10/T11 (AC-12 T07/T10) |
| AC-13 | T04/T10 |
| AC-14 | T01/T09/T10/T11 |
| AC-15 | T03/T04/T06–T08/T10/T11 |
| AC-16/17 | T09–T11; T05/T06/T08 supply compatibility inputs |
| AC-18 | All batch stops; T02/T03/T07/T09/T11 |

No specialist/tool exceptions to the plan. No automatic subagents, model evaluations, hosted experiments or repository-wide cleanup. A material path/trust/permission behavior change needs affected-document reconciliation before execution.

## Continuation record

2026-11-05 — Drafting only. Read spec and targeted source/feature records; drafted plan/tasks r1 and normalized the spec's sole approval anchor without changing requirements or recording consent. HEAD observed: `0ee5aaf65df7485d872d5784b17d6435175a185f`; inherited 006/integration/test changes and untracked docs remain outside this session's edits. Sources and planning observations are in plan §1; these are not release test results.

**Historical drafting checkpoint:** the package was presented for combined review with no batch started. The approval and execution update below supersedes that checkpoint. Later candidate inclusion, native evidence, version selection and live operations remain separate decisions at their ready checkpoints. Drafting verification results follow; no production/model/packaging suite is claimed run.

Draft verification: `git diff --check` passed; a non-mutating local Markdown check passed for spec/plan/tasks and the original drafting handover `docs/handovers/2026-11-05_008.md` (subsequently privately archived by owner-requested cleanup), checking local file/heading links and trailing whitespace, including untracked drafts that Git's ordinary diff does not inspect. The [current handover](../../docs/handovers/2026-11-05_008b.md) links the active continuation. Acceptance coverage was reconciled against AC-01–AC-18 and the user's strict-scope correction. These checks establish draft consistency only. Go, Python production suites, container/native probes, packaging and model behavior remain `not_run` in this drafting session.

2026-11-05 — B1 start: the owner approved the combined r1 package, requested `sdd-implement-1` and `sdd-implement-2`, then said “start” after B1-only execution was presented. Actual authority is recorded once in spec §Approval. T01 is delegated to the first implementation agent; the second prepares T02 independently without file edits until the inventory prerequisite is available. Parent owns public support/navigation integration, approval, this ledger, handover and final verification. Non-overlapping write scopes preserve existing dirty work. B1 is in progress; B2–B5 remain unstarted and unauthorized. No product inference/native probe or live action is included.

Owner steering during B1: the user flagged agent 2 investigating contexts related to excluded components. Parent immediately narrowed both workers: no further experimental-context investigation, mitigation/pilot analysis or experimental test execution. T02 remains only minimal current-facing state reconciliation required by R4 and required unsupported warnings; experimental code, routes and findings stay untouched. Remaining verification is bounded core text/link/provenance inspection, not experimental certification.

### B1 completion — 2026-11-05

- **Authority:** spec §Approval; B1/T01–T02 only. **State at delivery:** `awaiting human review` (superseded by the 2026-11-06 acceptance below). No B2–B5 implementation or live operation follows from the completion verdict.
- **Delegation:** `sdd-implement-1` produced the T01 inventory; its session reported `opencode-go/minimax-m3.1-flash`. `sdd-implement-2` made bounded T02 edits and reported `opencode-go/deepseek-v4.1-flash`. Configured primaries were requested; no model override or fallback was attempted. Parent reviewed and corrected report claims, integrated public docs, and owns the single completion review. Development delegation is not a product model evaluation.
- **B1 paths:** `README.md`, inherited `docs/README.md` (extended, not overwritten), `AGENTS.md`, current-facing 005/006/007 task summaries, entry warnings in `integrations/opencode/{independent-review,safety}.md`, `harness/trusted-ci.md`, and this feature's report/approval/continuation plus current handover. No Go, skill payload, native/test implementation, build/trust input or installer behavior change. No repository structure move.
- **Results:** prominent support/platform/artifact matrix; active 008 pointers; source-only versus published CLI distinction; original decisions and dated evidence retained; deferred/unsafe native review remains unsupported. T01 found an actual duplicate audit pair (retained), ignored local backups/bin (untouched), missing local GoReleaser and unobserved native macOS evidence. Those are later candidate prerequisites, not new implementation scope.
- **Checks:** `git diff --check` and scoped local Markdown/link/heading/whitespace inspection passed, including untracked docs. Independent source assertions passed for the exact 18 IDs/definitions, unchanged six required checks, unchanged embed/build/CI/release inputs, audit duplicate equality, no tracked backups/bin and one remaining handover. Baseline digest comparison confirmed all 8 protected inherited experimental source/test/design files unchanged; other baseline handover removals were separately authorized and privately archived. Machine-readable check results and exact checked-document hashes: `/tmp/opencode/008-b1-jys_ckvu/{verification.json,core-source-checks.json}`; these identify the checked working tree, not a clean release candidate.
- **Not run:** Go/Python production suites, experimental/native/container/model tests, packaging, macOS/Homebrew smoke and hosted CI. This batch changed documentation only; no claim of those checks passing or of behavioral compliance. The two workers ran metadata/source/text checks only; parent did not expand into experimental test execution after the owner's steering.
- **Post-cleanup link check:** 87 nonignored tracked/untracked Markdown files, 394 local targets, zero missing; scoped heading/whitespace and inherited-file checks also passed. Results: `/tmp/opencode/008-b1-jys_ckvu/all-markdown-links.json`. The original historical recovery source remains accessible through a commit-pinned Git link, not a dead working-tree path.
- **Residual limits:** 006 T03/T04 historical review-scope detail remains in its owner; no new acceptance is inferred or required to certify its unsupported native route. Accepted partial 005 scope, 004 findings, IR-06-01 and 007-F1 remain visible. Final candidate identity/version, permission-contract verification (B3), complete release QA (B4), same-candidate packaging/native evidence and go/no-go (B5) remain pending.
- **Separate owner-requested cleanup:** after the user selected “only keep latest one”, privately archived/verified 33 older handovers, retained `docs/handovers/2026-11-05_008b.md`, and removed the older files. Cleanup-only commit `5f2da3b` contains 22 tracked deletions plus two commit-pinned historical-link repairs; 11 untracked old handovers were removed only after private archival. Minimal repairs to untracked draft links stay uncommitted. B1 and unrelated edits were excluded from the commit; no push occurred. The retained current handover remains untracked with B1 until separately authorized commit disposition. Private archive/manifest are linked by path in that handover, not published. T01 counts describe the pre-cleanup snapshot.

### B1 acceptance and scope clarification — 2026-11-06

The owner accepted B1, requested existing 006/007 changes be saved separately as WIP, selected a B1-only commit, and required strict exclusion of deprecated 006/007 scopes before B2. Actual decision evidence and authority are recorded once in spec §Approval. Plan/tasks r2 add that fence without dropping 008's required maintained-core Taskfile/trust verification. WIP preservation does not reopen those features or establish passing behavior. No further experimental investigation or execution is authorized.

**Preservation commits:** `4b2f99f` saves only the inherited pre-B1 006 bytes and four existing new WIP files; `8696926` saves 007's status reconciliation as WIP. They do not reopen either feature. Parent staged the 006 baseline separately so B1's later warning/status changes were not silently included in that WIP checkpoint. No experimental suite, native operation or product model call was performed to save WIP.

**B1 commit scope:** the accepted documentation/inventory delivery, its approval/continuation/current handover, and owner-requested plan/tasks r2 exclusions only. Unrelated inherited removal of AGENTS' spending-policy section and the pre-existing untracked `docs/vnext-project-status.md` draft are excluded; their disposition remains with the owner. This package's resulting commit is identified by Git history rather than embedding its own hash in these files. No push is authorized.

**Commit validation:** staged whitespace checking caught six Markdown hard-break space suffixes in the previously untracked spec header. Replaced them with blank-line formatting without changing requirements or approval meaning; staged whitespace, local-reference and r2 scope-fence checks must pass before committing. This is documentation validation, not a production or experimental test run.

**Current position:** B1 accepted; 006/007 WIP preserved separately; r2 scope fence prepared for the authorized B1 documentation commit. **Next permitted action:** review r2 scope clarification and request explicit B2 authorization. B2–B5 remain unstarted; no delegation for a later batch follows from the prior B1 request. Scoped text/link/whitespace checks verify the clarification; no production, experimental or model behavior is claimed verified by those checks.
