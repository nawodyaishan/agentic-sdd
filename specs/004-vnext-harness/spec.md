# Agentic SDD vNext: verifiable single-agent lifecycle

Revision: r1 (draft, 2026-10-28).

## Outcome and sources

Evolve Agentic SDD from a portable skill package into a portable, measurable development harness, with OpenCode 2.x as the first runtime adapter. A user can draft and approve a feature, authorize one implementation batch, obtain deterministic verification and meaningful review, interrupt execution, and resume with traceable evidence and unchanged authorization scope. The same deterministic check contract runs without an LLM.

This feature selects the adoption plan's **minimum initial release, R0–R2**, as one coherent outcome. R3/R4 remain future work in that source. This is a proposed scope for human review, not implementation authorization.

Primary source: [Agentic SDD vNext Adoption Plan, revision 5](../../docs/Agentic-SDD-vNext-Adoption-Plan.md):

- Decision and scope; §§1–3: outcome, ownership, modes, configuration and enforcement.
- §§4–6: model/tool allocation, budget, verification, evidence and recovery.
- §§7–9: deterministic CI, evaluation, portability and maintenance.
- §10: minimum release and acceptance/failure scenarios; §11: unverified environment details.
- Appendix A1–A5, A7–A9: workflow handoffs, implementable contracts, debugging, testing and review.

Repository constraints: [AGENTS.md](../../AGENTS.md), [README.md](../../README.md), and the canonical [workflow policy](../../skills-files/agentic-sdd-router/references/workflow-policy.md). There is no top-level product SRS or roadmap. The adoption document supplies this feature's intent; this spec defines its proposed acceptance boundary.

The adoption plan §1 mentions approval recorded in `tasks.md`. For this repository, `spec.md` under `## Approval` remains the single combined-approval location; tasks and receipts reference it. This preserves the repository convention rather than introducing another approval record.

## Current baseline and evidence limits

- `skills-files/` is the canonical workflow package. Its current inventory describes ten skills, with shared policy and specialist references.
- The Go installer already provides preview, backup, install and restore behavior. Existing safety and compatibility guarantees remain requirements.
- The repository has an existing Makefile verification gate and isolated installer tests. `make verify` includes commands that write caches, build outputs and potentially module files; it is not a read-only review operation.
- Existing features occupy `specs/001-cli-quality/` through `specs/003-restore-backups/`. This feature does not reopen their approvals.
- The adoption document is researched proposal evidence, not an observed runtime baseline. Its original snapshot and separate coverage-checklist attachment have not been independently inspected here. Current repository files control implementation discovery.
- Exact installed OpenCode release, effective configuration, provider access, model compatibility, policy behavior and performance remain unverified. No runtime or benchmark result is asserted by this spec.

## Actors and authority

| Actor or source | Authority and limits |
| --- | --- |
| Human user | Approves the combined feature documents, authorizes a selected batch and decides material scope changes. Separately authorizes paid runs and live operations. |
| Main agent | Performs the requested mode within actual authorization; one implementation owner and one authorized batch by default. |
| Project | Owns requirements, source, checks and canonical task state. |
| User/runtime administrator | Owns provider accounts, credentials, machine configuration and higher-authority restrictions. |
| Runtime and execution environment | Enforce their tested tool, filesystem, process and network boundaries; workflow prose does not grant rights. |
| Checks, receipts and reviewers | Supply observations and technical verdicts; cannot create approval, authorize another batch, merge or deploy. |

## In scope

1. **Portable lifecycle contract.** Explicit draft, build/direct-fix, review and resume behavior, run-entry context, exit outcomes, allowed effects and authorization references. Direct fixes remain document-light; explicitly limited document requests remain limited.
2. **OpenCode foundation.** Credential-free project defaults, primary profiles and entry commands; a tested exact runtime version; sanitized effective-configuration discovery; tested policy boundaries; reversible, explicitly opted-in installation and a known-good rollback path.
3. **Bounded model and tool access.** Logical Economy, Standard and Deep profiles initially mapped to one default model and at most one alternative; capability checks; explicit fallback, destination, timeout and cost contracts. Select integrations only when relevant and available.
4. **Executable verification.** Existing project checks described through a reusable contract, with baseline/focused/completion scopes and a trusted deterministic CI gate independent of an LLM.
5. **Evidence and recovery.** Minimal receipts or adequate native exports, evidence freshness, current-diff awareness, interruption/cancellation handling, and resumption after restart, compaction or workspace changes.
6. **Workflow improvements.** Action-first routing; one completion-review owner; reconciliation of latest authorization; research returning to its invoking stage; contract-aware planning and tasks; meaningful testing; standalone `agentic-sdd-debugging`; bounded finding resolution; aggregate feature review.
7. **Evaluation and maintenance.** A small fixed feature/regression/review fixture set, cheap deterministic negative scenarios, baseline comparison, sanitized retention and compatibility smoke checks. Update skill inventories and distribution validation for the added debugging skill.

## Acceptance criteria

Acceptance labels are stable references for later design and verification ownership. Each claimed runtime guarantee requires observed evidence against the pinned release; unsupported paths must be disabled or the requested mode must fail explicitly. Passing static document checks alone does not establish agent behavior.

### AC-01 — Request routing and authorization

- Classify the requested action before feature intake. A direct fix in a repository without specs neither creates nor requires feature documents; a spec-only request produces only the spec.
- Feature drafting uses one combined spec/plan/tasks approval. Implementation starts only within actual authorization and stops after the selected batch for human review.
- Resume reconciles recorded state with the latest explicit authorization. For example, explicit B2 authorization after B1's review checkpoint permits B2, not B3. An ambiguous “continue” does not grant all remaining work.
- A named blocking uncertainty can enter research from drafting, implementation, direct fix or review and returns to that same stage with its mode restrictions intact.
- Technical verdict, execution result and workflow position remain distinct. `Needs changes` or `Blocked` cannot be represented as successful batch completion merely because the agent is awaiting a human response.

### AC-02 — Known effective configuration and reversible adoption

- Preflight identifies the exact runtime version, effective instruction/skill sources and revisions, active commands/profiles, configured models, integrations and plugins without exposing credentials.
- Conflicting or duplicate sources have an explainable effective result; invalid or ignored configuration, including dropped policy statements, is surfaced before relying on a claimed restriction.
- Project defaults use one documented location. Secrets and machine-specific configuration stay outside version control; higher-authority restrictions remain outside repository control where necessary.
- Adapter installation requires explicit opt-in, previews changes, backs up replaced content and supports restoring the known-good state. Unrelated configuration is preserved; identical content is a no-op.
- Existing skill install/restore guarantees remain intact, including preview by default, backups before replacement, rejection of symlinks/special files and isolated-home testing.

### AC-03 — Tested mode and execution boundaries

- Draft can write only agreed drafting artifacts; build/direct-fix can perform only authorized effects; review cannot mutate the target project's code, documents, approval or task state.
- Negative scenarios exercise every advertised review route through editor, shell, MCP, browser and Code Mode. A route not demonstrably read-only is unavailable in review mode. Tests that write caches, start services or modify databases require a separately authorized verification context.
- A higher-authority denial cannot be bypassed by project-level allow settings or saved tool approval. Invalid policy statements cannot silently count as enforcement.
- Documentation and evidence distinguish workflow intent, runtime policy and OS isolation. A worktree, standalone process or prose instruction is never presented as a sandbox.
- Untrusted executable fixtures run only in an identified, appropriate isolated environment. Repository/tool content cannot authorize weakening controls or disclosing credentials.

### AC-04 — Predictable models, tools and budget handling

- Actual provider/model/variant mappings and permitted data destinations are explicit and capability-tested before implementation. Incompatible tool support or unavailable models produce a clear blocked result or an explicitly permitted fallback.
- Agent/profile selection, command selection and fallback are checked against the supported runtime. Availability fallback and reasoning escalation remain separate; neither silently changes provider, cost class or authorization.
- Every enabled integration identifies purpose, permitted modes, effects, actual tool IDs, credential source, data destination, startup/catalog/execution limits, cancellation and fallback behavior. MCP resource access is included in the boundary.
- Missing integrations, malformed arguments, errors and truncated outputs cannot be recorded as successful research without sufficient evidence. A timed-out non-idempotent action is reconciled before retry.
- Additional paid model and tool usage stays within the user's authorized limits and the US$10/month ceiling. Account for failed attempts, retries, summaries and in-flight work; unknown or delayed cost remains unknown. Local estimates are not advertised as hard billing guarantees.
- No paid run is implied by document approval. Warming, speculative background work and automatic expensive escalation are off initially.

### AC-05 — Implementable requirements and task contracts

- Specs identify observable acceptance and material assumptions, adding examples where interpretations differ materially.
- Meaningful shared boundaries have one authoritative contract covering inputs, outputs, relevant errors, invariants, effects and necessary units/version/concurrency assumptions.
- Before combined approval, every acceptance criterion has implementation and verification ownership; dependencies, producers/consumers and connected-behavior responsibility agree.
- A fixture with incompatible shared error/type/unit expectations is reconciled or explicitly blocked before implementation. A criterion with no verification owner is identified and assigned suitable coverage.
- This readiness reconciliation adds neither another approval checkpoint nor mandatory architecture review for routine work.

### AC-06 — Deterministic verification independent of the agent

- Required checks specify command, working directory, scope, prerequisites, timeout, write/network effects, expected result and evidence location, using the existing task runner where adequate.
- The same contract executes locally and in CI without an LLM or provider credentials. CI independently runs trusted required checks against the final candidate; it does not infer success from an agent summary or exit alone.
- Baseline failures are distinguished from introduced failures. Changed behavior receives focused checks, batch completion covers applicable acceptance, and final feature review covers connected behavior.
- Missing executables, formatter failure, unavailable diagnostics, cancellation, timeout and missing evidence never count as passes. Formatting that changes the diff invalidates affected pre-format evidence.
- Relevant code/configuration/environment changes invalidate affected results. Unaffected valid evidence may be reused; unnecessary full-suite repetition is not required.
- A fixture that weakens or deletes a check to suppress a known failure is detected by the trusted gate or review and cannot establish Ready.

### AC-07 — Meaningful tests and systematic diagnosis

- Tests assert independently derived observable behavior or contracts. A test that computes its expectation using the implementation under test is identified as inadequate evidence.
- A regression's failing check demonstrates the target defect, not missing setup or an import failure. Test-first, characterization or integration/schema/policy validation is selected according to what proves the outcome.
- The new debugging skill accepts symptom, expected behavior, environment, authorization and scope without requiring feature documents for direct fixes.
- Diagnosis separates baseline/environment failures, traces the relevant boundary, uses a supported hypothesis and discriminating check, and reports remaining uncertainty. Repeated attempts without new evidence change the investigation rather than implying certainty or triggering automatic model escalation.
- Read-only diagnosis recommends instrumentation without adding it. Investigation does not authorize mitigation, deployment or a wider refactor.

### AC-08 — One review owner and resolved findings

- Implementation owns one completion review for its current result; the router consumes that result without scheduling a duplicate review of unchanged work.
- Review supports batch, direct-fix and feature scopes, covers requirements and engineering, and uses `Ready`, `Needs changes` or `Blocked`. Same-agent review is labelled self-review.
- Material findings retain stable identifiers, location, impact, evidence and disposition: valid, disputed or needing more evidence. Incorrect advice can be rejected with evidence; unresolved consequential findings remain visible.
- Authorized in-scope corrections receive scoped re-review, including affected dependencies and new regressions. Effort limits stop work with unresolved findings rather than converting them into success.
- Passing individual batch checks but failing connected feature behavior keeps the feature unready. Feature Ready does not authorize merge, release or deployment.

### AC-09 — Traceable results without another authority ledger

- Run entry resolves mode, workspace, relevant feature/batch, actual authorization reference, allowed effects, model/profile, tools, checks and deadline/attempt/spend limits. Unknown observations are explicit; missing required authority blocks the affected action.
- Run exit distinguishes `completed`, `incomplete`, `blocked`, `cancelled` and `failed`, independently of any technical review verdict. Results identify affected files, evidence, unresolved findings and next permitted action.
- Evidence correlates native session/run/task identifiers, effective harness revision, actual model/variant, base revision and relevant uncommitted diff identity, check commands/results/timestamps and remaining gaps.
- Receipts reference canonical approval and task state; they cannot create consent or become a second mutable completion ledger. Review receipts remain outside the target repository.
- Stored exports contain observable actions/results, exclude secrets and hidden reasoning, and follow an explicit retention policy. Unavailable cost/cache/token data is recorded as unknown.

### AC-10 — Safe interruption and resumption

- Resume after compaction, restart or location move re-reads canonical state and the current diff, preserves existing user changes, and validates workspace, authorization, blockers, evidence freshness and required capabilities.
- An unfinished check is not promoted to a pass; relevant edits invalidate stale Ready. A summary cannot grant another batch or broaden allowed effects.
- Cancellation/timeout stops new work, requests cancellation of owned activity, terminates only owned processes and reports uncertain external effects with the next safe action. Closing a stream is not proof of cancellation.
- Partial diffs are preserved. Non-idempotent actions are not blindly replayed; snapshot undo is not claimed to reverse external effects, commits or processes. Dirty workspaces are not force-removed as cleanup.
- One implementation owner per workspace remains the default.

### AC-11 — Measured adoption and compatibility maintenance

- Establish baseline observations before tuning, using a feature crossing a real contract, a regression investigation and a review-only case; retain an unseen variant and repeat critical cases within authorized limits.
- Compare instruction changes with input revision, model, permissions, tools and check environment held fixed. Model comparisons are separate. Report actual sample sizes and observations, not unsupported reliability percentages or savings.
- Track acceptance/regressions, scope violations, finding validity, latency/retries/tool failures, human interventions, cancellation/recovery and total known/estimated cost including failures. Zero verified successes is failure, not zero cost.
- Adoption requires passing claimed boundary/acceptance tests and an observed useful reduction in cost, delay or intervention without losing quality. Missing comparison evidence remains an explicit open adoption condition.
- Runtime/provider/model/integration changes trigger applicable selection, policy, tool, compaction and cancellation smoke checks. Preserve a known-good version/configuration and migration notes.

### AC-12 — Portable workflow and package compatibility

- Portable requirements, skill methods, check contracts, result meanings and fixtures remain separate from OpenCode-specific configuration and API details.
- Existing client skill distribution continues to work. Adding debugging updates explicit inventories and packaging/discovery validation so the new skill and sibling references are available together.
- Only OpenCode's runtime adapter is claimed supported by this release. Unsupported guarantees are explicit and cannot silently weaken the requested mode.

## Excluded from this release

- R3/R4 extensions: independent reviewer agents, delegated workers, parallel writing, worktree concurrency pilots, custom continuity plugins, headless advisory LLM jobs, bounded autonomous code-changing jobs, model/cache optimization and additional runtime adapters.
- A replacement agent loop, orchestration engine, universal configuration compiler, dashboard, custom MCP server or custom TUI.
- Blanket tool auto-approval, automatic continuation through batches, automatic model escalation or additional approval ledgers.
- Enterprise management, proxies/custom CAs, credential rotation and account changes unless separately scoped from demonstrated need.
- An unrelated installer rewrite, real-home installation, publication, deployment, live migration, remote merge-rule administration or paid benchmark execution merely because this feature is drafted or approved.
- A universal red–green ritual for prose, phrase-presence tests as proof of agent compliance, mandatory per-task reviews or mandatory domain specialists for routine work.

Deterministic CI and tested cancellation/recovery are included even though autonomous/headless LLM jobs and concurrent workers are deferred. R3/R4 are not completion dependencies.

## Material unknowns and decisions for planning

These are not assumed user decisions and do not prevent drafting this outcome contract:

1. **Supported runtime and access:** determine the exact V2 version, effective configuration, available providers/models/variants and actual integration capabilities before selecting adapter settings. Verify version-sensitive source claims against that release.
2. **Execution environment and installation surface:** choose the isolated fixture environment, permitted review observation routes, opt-in configuration delivery and rollback mechanism. Approval of code development does not authorize changing the user's real setup.
3. **Pilot and evidence policy:** select the real cross-contract pilot, regression/review fixtures, evidence storage/retention, and check prerequisites. Set concrete run effort/deadline limits and cost controls from actual access; the proposed US$1 comparison envelope is not spending consent.

The later plan must make consequential design choices explicit; tasks must map all acceptance criteria to owners and executable or observed verification. If a required boundary cannot be demonstrated on the chosen runtime, surface the gap for scope review instead of silently waiving it.

## Approval

- Decision: **approved** (2026-10-28).
- Scope: spec r1, plan r1 and tasks r1, selecting the R0–R2 minimum initial release.
- Human approver: the user in this conversation.
- Approval evidence: “Approve the drafts and implement B1.”
- Reviewed document SHA-256 identities, before this approval/state update: spec `cbf9342fe3e9043d7a1517a69b05b2c4b74d31626a377ccdd1f4f9e0ca21bc44`; plan `7c3d48b3222027b5a9ac84d79bd4d06258482ceea7e0c2f5be8ba33846c0f8e5`; tasks `a5a1bdcff77d4cbe7a92c2eb045a8f58f9ff9f0709ef4920b5d03e10729ca83f`.
- Implementation authorization: **B1 / T01–T02**, approved 2026-10-28; **B2 / T03**, authorized 2026-10-28; and **B3 / T05–T06**, authorized by “study /home/nawo-ubuntu1/github/INCIDENT-2026-10-28-openai-spend.md doc and make best effort to avoid sucj incidents future and proceed to B3”. This advances past B2's review checkpoint without inventing an additional human technical verdict. No B4, real-home installation, credential/account change or unrelated live operation is authorized.
- B3 model selection: user supplied DeepSeek Flash Free for everyday work, GLM Flash Free for quick actions, paid MiMo V2.6 Pro for harder work, and Nemotron 3 Ultra Free for synthetic experiments. User later permitted testing these named low-cost/free models. That permission does not establish current prices, private-data eligibility, remaining allowance or a dedicated paid-key cap. B3 used no additional model calls; provider execution stays disabled pending those prerequisites. The usage catalog does not expand the initial one-default/one-alternative deployment scope or authorize automatic escalation.
- B4 authorization: user said **“start b4”** after selecting all three pre-B4 remediation options. Scope is **B4 / T07 only**, deterministic runner development/verification. Sandbox access was restricted per the user's later **Restrict and defer** decision; dedicated-key cap/rotation remain pending user actions. This authorizes neither B5 nor additional model usage, account changes, runtime-config installation or sandbox deletion.
- B5 authorization (2026-10-29): user said **“commit push b4 and start b5”**, then **“resume”** while B5 was in progress. B4-only commit `1b50933193c622197eeab45fa0a8005b425d76b9` was pushed to `origin/main`; pending B3 work was excluded. Scope now is **B5 / T04 only**, skill/packaging/diagnostic verification. No B6, model execution, live installation, key/account change or deferred deletion is authorized.
- B5 human review (2026-10-29): user said **“B5 approved , if any blockers or decisions remainign , ask me”**. Approval is recorded for the presented partial delivery; it does not supply the missing B5-F1 observations, waive acceptance or authorize B6, model usage, publication or live operations. Remaining decisions are being returned to the user.
- B6 authorization (2026-10-29): user selected **“Offline B6 (Recommended)”**: only credential-free receipt/freshness work and offline recovery fixtures; model-backed/native acceptance stays blocked. User also selected **“Both completed”** for exposed OpenRouter-key rotation and creation of the dedicated evaluation credential with the proposed $0.25 lifetime cap. This is user-reported completion, not independent verification of scope/cap or permission to access the key, run a model, adopt config or delete old sandboxes. No B7 or publication is authorized.
- B6 review / B7 authorization (2026-10-30): user said **“accept the delivered offline B6 subset without waiving its findings, then authorize B7/T10 only”** and allowed B7 to consume the delivered offline receipt/check contracts with native correlation unresolved. B7 exit evidence must cover trusted bootstrap, candidate-weakened gates and local/CI consistency. B6-F1/F2 and earlier findings remain open. No B8, model run, live install, branch-rule administration, publication or credential operation is authorized.
- Publication authorization (2026-10-30): user said **“commit and push the stable changes”**, then selected **“All tested artifacts”** after clarification that this includes inactive B3 and B5–B7 deliverables and activates the proposed CI workflow on main. Publish verified source in logical commits, retain findings, and exclude private evidence/local forensic notes. This authorizes the commit/push and resulting deterministic CI, not B8, release tags, model/native experiments, live client installation or account/branch-rule administration. B7-F1 is not resolved by publication alone; hosted results still require observation.

Next action: human review of B7/T10 delivery and B7-F1. Local trusted-bootstrap/anti-weakening/CI-style consistency fixtures pass; trusted-branch adoption and hosted execution remain unperformed, so full CI activation is technically Blocked. B6/native/earlier findings are retained. B8 and online usage remain unauthorized. Continuation evidence is in `tasks.md`.
