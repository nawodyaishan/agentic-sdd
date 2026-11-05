# Feature 008 — Core Product Quality, Repository Maintainability & Release Readiness

**Status:** Combined r1 approved; B1 accepted; B2 authorized under the r2 scope fence; later batches not authorized

**Date:** 2026-11-05

**Revision:** r1 — approved combined spec/plan/tasks scope; execution updates reference the approval record.

**Proposed repository path:** `specs/008-core-release-quality/spec.md`

**Baseline:** Public `v0.4.0`; November 5 read-only assessment of `main` at `0ee5aaf`; owner-supplied repository tree

**Next release:** Undecided `v0.x` version, subject to review and verification

**Authority:** [Approval](#approval) records combined r1 approval, accepted B1 and B2-only execution under the r2 scope fence. Prior delegation and commit authority were bounded to B1 and the named WIP preservation. No experimental model evaluation, local configuration change, real-home apply, release tag or publication is authorized.

## 1. Product vision and outcome

Make the existing Agentic-SDD **supported core** a coherent, easy-to-adopt, maintainable and verifiably release-ready open-source product. Agentic-SDD is an **engineering learning-and-delivery framework** for users of coding agents: it improves requirements, implementation, debugging, verification and review while retaining human understanding and control. It is not a general-purpose autonomous multi-agent engine.

The intended audiences are (1) solo developers using OpenCode, Codex and Claude Code; (2) interns, undergraduates, junior and self-taught developers who want to learn professional engineering practices while delivering work; and (3) experienced backend, infrastructure, platform and SRE engineers and maintainers. These are **target-audience hypotheses**, not demonstrated adoption.

**Release outcome:** A new user can safely preview/install, distinguish direct fixes from features, establish scope on an incomplete request, review an authorized batch, understand verification, and find the documented support limits. Maintainers can understand the active status without reconciling stale summaries, run release-critical checks on a selected candidate and determine a justified release go/no-go.

The feature is successful only with evidence, not because a release was desired: a **documented no-go** is the correct outcome if any supported-core blocker remains. Publication always needs a separate human decision.

## 2. Scope boundary and existing architecture

**Retain as supported-core candidates, subject to fresh release checks:**

- All **18 existing portable skills** and their conditional references, routing, direct fixes, optional teaching, debugging and reviews.
- One main agent; one combined spec/plan/tasks approval; one explicitly authorized batch; one completion review; stop for human continuation.
- `cmd/agentic-sdd`, `internal/skillsync`, embedded assets, preview-first installer, backup/staged replacement/rollback and restore.
- The **documented supported portion** of `sdd-check`, `harness/` check/evidence contracts and trusted CI, with platform and trust limitations accurately declared.
- Existing `Taskfile.yml`, `AGENTS.md`, source/release tooling and GitHub/Homebrew distribution paths.

**Experimental/unsupported in this release:** native OpenCode ACP model execution, automatic multi-model routing or billing firewall, general native independent reviewer filesystem isolation, automatic subagent orchestration, native macOS check-runner guarantees and cross-client behavioral equivalence. Feature 006 B4 remains blocked/deferred by the demonstrated `IR-06-01` symlink escape; no documentation improvement can certify that boundary.

Do not duplicate the existing `human-input.md`, `grilling`, `questionnaire`, `teach`, `implement`, `verification-review` or approval contracts. Keep portable methodology separate from host-specific runtime enforcement.

## 3. Source-grounded problems

The owner's November 5 read-only report recorded:

| ID | Finding | Release disposition |
|---|---|---|
| **PQ-01 / IR-06-01** | Native read-only reviewer escaped an intended exclusion through an allowed symlink | Keep generic native reviewer unsupported; disclose explicitly |
| **PQ-02** | Stale current-status statements in `AGENTS.md`, feature 005/007 tasks and 006 runbook | Reconcile current-facing text without rewriting historic decisions |
| **PQ-03** | Required six checks omit relevant Python/Taskfile/trusted-CI regression suites | Explicit release verification and safe test ownership |
| **PQ-04** | README's file-permission comparison language contradicts installer mode semantics | Establish and verify accurate user-facing contract |
| **PQ-05** | Budget-model reliability/cost advantage not measured; behavioral cases `not_run` | Do not advertise unproven benefits; use bounded scenario evidence |
| **PQ-06** | Newcomers lack one concise end-to-end worked path | Provide examples without creating a second workflow |

The submitted tree also shows many historical `docs/handovers/`, `audits/`, `docs/006-independent-review-spec.md`, `backups/`, `bin/`, mixed Python/shell tests, `007-taskfile-migration`, and root embed files. **Presence in `tree` does not establish tracked status, duplicate content, user ownership or safe removability.** The read-only assessment did not rerun tests or establish a clean candidate tree.

## 4. Functional requirements

### R1 — Clear supported product and release contract

Publish an accurate, easy-to-find support matrix distinguishing portable skills, CLI/restore, source-only tools, supported operating systems, GitHub/Homebrew artifacts and inactive OpenCode experiments. Installation into four client directories is **not** evidence of identical behavioral compliance. Document what is actually supported at `0.x`, not an unapproved `1.0.0` API promise. README, help, installed assets, docs and release notes must agree.

### R2 — Better human–agent interaction using existing skills

For incomplete prompts, the current router/human-input procedure must first inspect discoverable repository facts, reuse settled answers and isolate **consequential** human decisions. Ask at most three ready independent questions in a round; present recommendation, plausible alternatives and consequences where appropriate. Do not force an interview for a clear request. Distinguish established fact, assumption, decision, approval and authorization. Explain scope, exclusions and acceptance conditions at the appropriate checkpoint. Implement improvements by refining **existing** instructions where there is a demonstrated gap, not by introducing another mandatory skill, approval, form or ledger.

### R3 — Proportional implementation, debugging and review

Preserve the direct-fix route without feature paperwork; feature route with reconciled spec/plan/tasks; review-only route without mutation; optional teaching without turning a checkout into a course. Ensure implementation output communicates changed behavior, actual checks, limitations, risk-based review points and the next permitted action. One principal implementation completion review remains the owner; do not duplicate it in the router. No automatic review/fix loops or model escalation.

### R4 — Correct authority and state navigation

Reconcile misleading **current-facing** status statements in canonical project guidance and feature records, including 005/006/007, while preserving dated history and genuine approvals. Canonical combined approval stays in `spec.md` §Approval; batch continuation stays in `tasks.md`. Handovers and summaries are secondary pointers. A model's `Ready`, an accepted partial delivery, a green test, or bare `continue` must not authorize another batch or live operation. Existing 004/006 unresolved findings remain visible.

### R5 — Structure, navigation and repository hygiene

Perform a tracked/ignored/untracked path and provenance inspection **before changing** structure. Retain `cmd/`, `internal/`, `skills-files/`, `specs/`, `harness/` and `integrations/opencode/` as principal ownership boundaries. Improve discoverability of active guides versus historical reports, handovers and audits; a concise `docs/README.md` index or equivalent is preferable to a bulk move. Identify actual duplicate documents before consolidation. Review `tests/harness/` ownership and entry points; reorganize only if it yields defensible maintenance value with path/reference regression coverage.

Review `backups/` and `bin/` for tracked/generated/sensitive content and `.gitignore` correctness; **never discard, publish, or alter local user backups/artifacts merely because they appear in the tree**. Preserve all 18 skill names and linked references, root Go embed assets, import paths, installation destinations and release artifact behavior. Document any migration/path impact and rollback. Cosmetic conformity to a large Go template is not a sufficient reason for change.

### R6 — Complete and trustworthy release verification

Create a candidate-specific verification matrix: Go tests/vet/format, skill/embed/reference integrity, installer/restore E2E, Taskfile regressions, relevant Python harness tests, evidence/check-runner tests, trusted-CI anti-weakening, release packaging and user first-run smoke. State exactly which checks run as required CI, separately isolated release QA, manual walkthrough, or `not_run`.

Preserve **trusted-base ownership** of CI bootstrap/check contracts. Do not run untrusted candidate scripts with host secrets, privileged tokens or dangerous mounts. Do not blindly move all Python suites into `harness/checks.json`; preserve the check-runner's distinct trust/scope contract. Negative cases must show failure is detected. Report test identity, scope, environment, result and limitations, including unobserved fork/PR and native-platform paths.

### R7 — Installer permission/restore contract

Resolve PQ-04 by agreeing the user-facing semantics for file modes, no-op comparison, mode normalization, backup and restore. Prefer clarifying documentation to changing tested behavior unless a demonstrated correctness/safety issue requires implementation changes. If behavior changes, assess compatibility and verify idempotence, permission-only differences, failed writes and restore. Never claim exact mode preservation when the implementation normalizes modes. Preserve preview-first and backup-before-write invariants.

### R8 — Public onboarding and contribution readiness

Present a minimal safe first-five-minutes entry point and **three compact worked experiences**: (a) a direct fix with no feature paperwork; (b) a feature with short spec/plan/tasks, combined human approval and one batch; (c) a review/resume where acceptance does not authorize continuation. Explain the same workflow to learners and professionals, with teaching optional. Advanced runtime experiments belong behind explicit warnings. Ensure CONTRIBUTING and security-reporting guidance are navigable, accurate and proportional; do not add ceremonial files without a real use.

### R9 — Integrated Taskfile, release candidate and decision

Inspect Feature 007's actual Taskfile state before relying on it. Ensure documented developer commands, test invocations, CI, GoReleaser/release metadata, generated artifacts and Homebrew instructions refer to the maintained implementation. Choose a **clean, identified release candidate**. Validate packaging, embedded skills, metadata and the advertised macOS/Homebrew install path; distinguish Linux source usage and the unpublished `sdd-check` executable. Prepare factual release notes, known limitations, an exact candidate identification, compatibility disposition and a justified go/no-go. Version selection follows changes and remains `0.x` unless a separately approved compatibility contract exists.

## 5. Explicit non-goals

- Completing Feature 006 B4 or treating the proposed sanitized-export C2 as certified.
- Enabling native ACP inference, launching paid/free model tests without separate authorization, general model routing, mandatory subagents or prompt-warming/caching infrastructure.
- Replacing Agentic-SDD with Superpowers, Spec Kit or a different development methodology.
- Creating a separate learner product, compulsory teaching, more approval records or a new current-state ledger.
- Promising budget-model reliability, lower token usage or human productivity gains without measured comparable observations.
- Certifying macOS native check execution, all client runtimes, hosted PR/fork isolation or enterprise security without corresponding evidence.
- Deleting historical evidence, unrelated work, backups or generated outputs without provenance and explicit authority.
- Tagging, pushing, publishing, updating a Homebrew tap, changing credential stores or applying skills to a real home merely because the feature is approved.

## 6. Acceptance criteria

| ID | Required observable result |
|---|---|
| **AC-01** | Current supported vs experimental/platform/artifact matrix is prominent, internally consistent, and grounded in source and actual candidate evidence. |
| **AC-02** | Existing feature, direct-fix, review-only, research, limited-document and teaching routes retain their proper authority and side-effect boundaries. |
| **AC-03** | Tested/inspected incomplete-request examples show fact discovery, scope establishment, relevant human questions, settled-answer reuse and no mandatory interrogation for clear requests. Static tests and actual model behavior are labeled separately. |
| **AC-04** | Negative scenarios for bare `continue`, stale handoff, `Ready` verdict, approval vs batch authorization and live/spend permission do not grant new work authority. |
| **AC-05** | Implementation/debugging/review guidance has one completion-review owner and concise, risk-relevant results, actual checks, known unknowns and next allowed action. |
| **AC-06** | PQ-02 current-status contradictions are removed or marked historical in their owning locations; old approvals/findings remain intact, including accepted partial 005 and open 004/006 findings. |
| **AC-07** | IR-06-01, unfinished 006 B4 and experimental OpenCode restrictions are visible in appropriate user-facing docs and candidate release notes; generic native review is not claimed safe. |
| **AC-08** | README/help, installer source and focused tests agree on permission comparison, normalization, no-op, backup and restore semantics; any behavioral change is explicitly reviewed for compatibility. |
| **AC-09** | A tracked/ignored/untracked structure inventory establishes the disposition of `bin/`, `backups/`, historical docs and mixed tests. No unreviewed loss of local or historical information. |
| **AC-10** | All 18 skill IDs and local references remain valid in source and embedded/installed distributions; any path moves have passing dependency/link checks and rollback information. |
| **AC-11** | A release verification matrix records exact suite, command, environment, candidate identity, result and whether CI/manual/not-run. Critical changed surfaces have fresh, relevant evidence; exceptions are explicit. |
| **AC-12** | Trusted CI invariants and negative/anti-weakening checks still hold; no candidate-controlled code is executed in a privileged trusted-base context. |
| **AC-13** | First-use installation and all three worked examples are repeatable with documented prerequisites and correct human stop points. Teaching remains opt-in. |
| **AC-14** | Release-candidate packaging, CLI metadata, macOS/Homebrew route, source-only tooling and distribution claims are verified for the **same exact candidate**. |
| **AC-15** | Model scenario `not_run` statuses and unmeasured cost/quality claims remain honest; evidence types are not conflated. |
| **AC-16** | Release report records exact commit/tree, known blockers, nonblocking experimental findings, supported compatibility scope, verification evidence, version recommendation, draft release notes and a reasoned **go/no-go**. |
| **AC-17** | Compatibility-sensitive changes are classified and have migration notes or are excluded; no automatic `v1.0.0` promise is made. |
| **AC-18** | Human acceptance of these artifacts does **not** start a later batch, spend models, modify real-user installs or authorize publication. Release action gets its own explicit approval. |

## 7. Release gate and defect disposition

**Release-blocking for the promised core:** broken or incomplete packaged skills, unsafe installer/restore regressions, materially misleading support claims, missing fresh verification of changed release-critical behavior, compromised reviewed-base CI boundaries, invalid release artifact/version metadata, or contradictory live-looking approval/continuation guidance with security impact.

**May remain open if prominently excluded:** IR-06-01 and 006 B4 generic native reviewer; disabled native ACP admission; unresolved hosted token/billing or compaction questions; budget-model comparative metrics; cross-client behavior equivalence; native macOS runner; unproven divergent fork/PR route. A known critical defect in a shipped supported surface **cannot** be reclassified as experimental merely to obtain green status.

**Not a release gate:** cosmetic reorganization; learner adoption metrics; adding autonomous agents; completion of all historical feature 004/006 research. No finding may be silently waived; accepted residual risk must have a named scope and human decision.

## 8. Structural target (illustrative, not a mandatory migration)

```text
README.md                   # concise public entry point
AGENTS.md                   # current repository engineering authority
CONTRIBUTING.md             # contributor entry point
Taskfile.yml                # developer commands / explicit release QA
cmd/                        # existing Go CLI entry points
internal/                   # existing Go implementation
skills-files/               # 18 portable skill directories; retain identities
specs/                      # feature approval and continuation owners
harness/                    # deterministic contract and trusted CI
integrations/opencode/      # clearly experimental native material
tests/                      # documented owners and run commands
docs/
  README.md                 # optional index: onboarding, reference, history
  handovers/                # historical pointers, not permission
```

Go's official module guide supports the current `cmd/` + `internal/` organization. A structure change needs a concrete benefit, actual path-impact map, proof of working Go embeds/imports and link resolution, and a safe recovery route; **renaming everything to fit a template is prohibited scope creep**.

## 9. Validation method and evidence integrity

- Start by observing the real Git status and current canonical `spec.md` approvals / latest `tasks.md` continuations. Preserve dirty and untracked work; don't assume the provided tree is the release source.
- Use model-free source and fixture checks for static rule coverage, failure behavior and package integrity. These do **not** establish that budget models follow instructions.
- Separate Go/Task/Python deterministic checks, script/native probes, manual first-run evidence and online model evaluations. An online comparison requires separate exact model, provider, privacy, token/request/time and credentials authorization, with no silent fallback or retry.
- Test bad inputs and negative paths, particularly unauthorized continuation, direct-fix ceremony regression, review-only mutations, broken links, restore failures and untrusted CI candidate weakening.
- Validate packaged and installed outputs using safe isolated homes and appropriate CI isolation. A home directory or Git worktree alone is not a security sandbox.
- Present `pass / fail / blocked / not_run` explicitly. Failing or unavailable relevant evidence blocks the corresponding *claim*, not automatically unrelated offline scope.

## 10. Design dependencies, unresolved decisions and risks

**For subsequent `plan.md` / `tasks.md` design:**

1. Reinspect Feature 007's actual delivered/accepted state and Taskfile/CI/release invocation, rather than trusting stale heading rows.
2. Establish tracked vs untracked provenance of `backups/`, `bin/`, docs duplicates and build outputs.
3. Decide PQ-04 by focused source/test analysis: documentation correction vs a separately justified behavioral change.
4. Map exact build, Homebrew tap, check-runner support and required released artifacts at the chosen release candidate.
5. Determine which additional checks are safe in ordinary CI and which belong to isolated release QA without compromising trusted-base isolation.
6. Decide any docs/test path moves only after referencing every consumer and migration impact.
7. Determine appropriate next `0.x` version from actual changes and compatibility; do not preselect it.

**Risks to mitigate:** accidental scope expansion into an orchestration rewrite; history presented as new consent; claim inflation from static test results; inadvertently weakening CI or permissions; unsafe cleanup of backups; broken installed skills due to path changes; release from a dirty/unidentified tree; premature 1.0 promise.

## 11. Completion, handoff, and release authority

At feature completion, provide a bounded change summary, user-facing before/after examples, path/compatibility inventory, candidate-specific test matrix, open findings and exclusions, exact release candidate, draft release notes, and a go/no-go judgment. Stop for human review. The owner may approve a narrowly supported `0.x` release **after** independently authorizing release/tag/tap actions; green CI or feature acceptance alone is not publication permission.

## Approval

This is section 12 and the sole combined-approval record for [spec r1](spec.md), [plan r1](plan.md) and [tasks r1](tasks.md).

- **Decision:** Approved, 2026-11-05.
- **Approver:** The repository owner in this conversation (no additional identity asserted).
- **Approved revisions/scope:** Combined r1 spec, plan and tasks as presented, retaining the owner's explicit strict-008-only scope and exclusions.
- **Decision evidence:** The owner said “i approve all 3 plan spec tasks” and requested use of `sdd-implement-1` and `sdd-implement-2`. After the response proposed recording approval and implementing B1 only, the owner switched out of Plan mode and said “start”. This authorizes that presented B1 scope, not B2–B5.
- **Authorized implementation batch:** B1 (T01–T02) is accepted. On 2026-11-06, after plan/tasks r2 exclusions were presented and committed, the owner said “start b2”: authorize B2 (T03–T04) only under that fence, verify and stop for human review. B3–B5 remain unauthorized.
- **B1 acceptance (2026-11-06):** the owner said “accept b1” after the delivered work, checks and residual issues were presented. This accepts the bounded documentation/source-inventory scope, not release readiness or excluded-feature behavior.
- **Post-B1 scope direction:** the owner required plan/tasks updates before B2 to exclude deprecated 006/007 scopes. Plan/tasks r2 make that fence explicit; the subsequent “start b2” applies to that presented scope. Core Taskfile/trust verification required by R6/R9 remains within 008; no old-feature resumption is authorized.
- **Commit authority (2026-11-06):** the owner requested existing uncommitted 006/007 changes be saved as WIP and selected “Commit B1 only” for the separate reviewed B1 delivery. Use scoped separate commits, preserve unrelated/private work, and do not push. Saving WIP does not approve, test, certify or resume those features.
- **B2 commit authority (2026-11-06):** the owner said “commit b2 and before start b3, give me tasks in b3”. Commit B2 only, preserve unrelated changes, and present B3's planned tasks. This does not authorize a push or B3 implementation, nor supply a separate B2 acceptance or release-readiness decision.
- **Authorized delegation:** `sdd-implement-1` and `sdd-implement-2` for bounded, non-overlapping B1 work using configured models; parent integrates and verifies. This is development assistance, not authorization for experimental product model evaluations or native reviewers.
- **Authorized model/network/credential/live actions:** Only the explicitly requested development-agent delegation above; no product model evaluation, native dispatch, credential/configuration change, real-home apply or other live action.
- **Authorized tag, push, GitHub Release, or Homebrew tap change:** None.

The recorded decision covers all three reconciled r1 documents. Later batches and live operations require their own actual authorization. B1 completion review does not start B2.

## 13. References

**Primary project evidence:** owner-provided `agentic-sdd-context-pack.md` (2026-11-05); read-only product-quality assessment at `0ee5aaf` with PQ-01–PQ-06; owner-provided `tree -L 6` output; canonical repository files including `AGENTS.md`, `skills-files/agentic-sdd-router/references/{workflow-policy,human-input}.md`, `harness/{checks.json,acceptance.md,trusted-ci.md}`, `specs/004-*` through `007-*`, and `integrations/opencode/{usage,independent-review,safety}.md`. Snapshot descriptions are not fresh test runs.

**External best-practice references (guidance, not verification of this repository):**

- Go module organization: https://go.dev/doc/modules/layout
- Go Task ordering and task dependencies: https://taskfile.dev/docs/guide/dependencies
- GitHub repository best practices: https://docs.github.com/en/repositories/creating-and-managing-repositories/best-practices-for-repositories
- Semantic Versioning: https://semver.org/
- Matt Pocock's productivity skills: https://github.com/mattpocock/skills/tree/main/skills/productivity
