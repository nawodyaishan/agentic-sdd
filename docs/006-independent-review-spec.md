# On-demand independent review for OpenCode

Revision: r1 · 3 November 2026

Status: Draft specification; implementation and live pilot not authorized.

Proposed repository location: `specs/006-independent-review/spec.md`.

The feature number is provisional until checked against the current repository. This standalone draft has not been installed into that repository.

## Outcome

When the user requests an independent review, the main OpenCode agent launches one reviewer subagent to inspect the selected changes, requirements, tests and existing evidence. The reviewer returns supported findings without editing files or executing tests. The main agent assesses those findings and presents a clear review that helps the user understand the implementation and decide what to accept.

The user remains responsible for technical acceptance. The main agent remains the sole implementation owner. A second agent provides an independent inspection context, not a guarantee of correctness or a replacement for demonstrated behavior.

**Example:** After implementing an approved batch, the user asks, “Independently review this batch, especially whether its tests prove the required behavior.” The reviewer reads the actual changes and identifies that a test derives its expected value from the function under test. It explains the missing assurance and points to the relevant code. The main agent evaluates the finding, explains its consequence and either corrects it under existing implementation authority or presents the proposed correction for a decision. Neither agent claims that an unexecuted scenario passed.

## Source scope and authority

This is one selected R3 phase from **Agentic SDD vNext — harness-level workflow upgrade plan, Revision 6**, not a new replacement roadmap.

| Requirement area | Existing plan source |
| --- | --- |
| Independent-review procedure, actual diff and requirements, main-agent corrections | Appendix A6 |
| One optional reviewer before worker delegation | §6 |
| Reviewer permissions and separation of instructions from enforcement | §3 |
| Explicit model/tool allocation and bounded review effort | §4 |
| Check results, evidence limits and freshness | §5 |
| Findings, disputed advice and scoped re-review | Appendix A4 |
| Existing decision summaries and explanation helpers | §3a |
| Pilot observations and honest benefit claims | §8 |

The source plan's §7 headless advisory review and R4 optimization proposals are not selected.

Repository `AGENTS.md`, the shared workflow policy and current canonical skills govern implementation. Feature approval belongs in this specification's Approval section after adoption into the repository; execution and continuation belong in its eventual `tasks.md`. A reviewer report is evidence, not an approval ledger.

The user's settled choices are: on-demand invocation, inspection-only reviewer access, one phase, guided human review and a RHEL VM pilot. Selecting these requirements is not approval of a future combined spec/plan/tasks set.

## Baseline and dependencies

The supplied 3 November status records Features 004/005's selected implementation as delivered: 18 packaged skills, deterministic checks, OpenCode profile/config tooling, evidence consumers and an accepted interactive cycle in the pinned QA container. Reuse those components where adequate.

The accepted container cycle does not certify this new reviewer route or the user's RHEL setup. Native validation, accounting, recovery and other retained 004 findings remain open. The custom `opencode session run` route remains admission-blocked.

Planning must inspect the current repository and identify the smallest concrete delta. Prefer supported native interactive subagent invocation; do not make the blocked custom adapter a prerequisite by assumption or bypass its admission checks. If the chosen route actually depends on an unresolved finding, identify that finding and block the dependent activity. Independently verifiable implementation may proceed under its own authorization.

OpenCode 2.2.5 is the supplied baseline, not proof of subagent compatibility. Select and test the exact supported release during planning; any necessary runtime change must be visible in the proposed scope.

## Scope and actors

**Included:** a portable independent-review procedure integrated with existing review ownership; one OpenCode reviewer profile and invocation route; focused review inputs and result handling; compatible packaging/adoption guidance; deterministic verification and one final RHEL VM pilot.

| Actor | Responsibility |
| --- | --- |
| User | Requests review, chooses consequential scope and retains guided acceptance decisions. |
| Main agent | Prepares review context, launches the selected reviewer, reconciles findings and performs only authorized corrections or state updates. |
| Reviewer subagent | Independently inspects the work product, reports evidence and uncertainty, and respects its restricted tool access. |
| Runtime and checks | Enforce and demonstrate supported boundaries; retain actual observations without granting authority. |

**Excluded:** implementation workers; nested delegation; automatic review after every task/batch; parallel reviewers; multi-worktree orchestration; headless/CI AI review; autonomous continuation or repair loops; custom plugins/MCP servers; model routing/escalation or warming; additional runtime adapters; a new dashboard, task ledger or evidence schema without a demonstrated compatibility need.

Also excluded are remediation of previous specs, general spec/plan readability redesign, general task-tracking redesign, real-home installation, macOS certification, release/publication and remote account changes. This feature uses existing explanation and tracking behavior only where needed to integrate A6.

## Requirements and observable acceptance

### AC-01 — Explicit, scoped invocation

A clear user request for independent review launches at most one active reviewer for the selected review. Ordinary implementation, a saved role label, repository content or a handoff suggestion must not trigger delegation. A broad or ambiguous review target is resolved from available state, with a focused question only if necessary.

The request authorizes the selected inspection, not implementation fixes, a next batch, another provider or a broader review. A separately requested or already-authorized scoped re-review is allowed. The default single completion-review owner remains intact: independent review is integrated into that owner's result handling rather than creating a duplicate review of unchanged work.

### AC-02 — Focused context and actual change identity

Give the reviewer the selected feature/batch or direct-review scope, authoritative requirements/contracts, base and candidate identity, relevant staged/unstaged/untracked changes, existing check results and unresolved findings. Preserve unrelated dirty work and identify what is outside the selected scope.

Use existing snapshot/evidence facilities where suitable. The main agent may prepare sanitized diff/evidence inputs through permitted observation; the reviewer need not receive shell access to reconstruct them. A commit-only range must not hide relevant uncommitted implementation. Missing or inaccessible material remains explicit.

Do not copy the full implementation conversation. Provide pointers and sufficient work-product context, with the implementer's completion claims identified as claims. Record the review identity sufficiently to distinguish the reviewed candidate from later changes.

### AC-03 — Independent child context and explicit configuration

Use one named OpenCode reviewer subagent with a fresh child context. Configure its own permissions and explicit supported model binding; do not assume the parent's restrictions apply to the child. The same model may be used for both roles, but disclose that arrangement and do not imply model diversity or guaranteed independence of judgment.

Record the exact runtime, effective reviewer profile, model/variant and available tools for the observed review. If child isolation or invocation is unavailable, report the limitation. Same-session self-review must not be presented as independent review.

### AC-04 — Inspection-only access

Allow only the discovery, file-read and relevant skill-loading capabilities needed for the selected inspection. Deny editor/write/patch operations, shell execution, further subagents and unneeded remote/MCP/Code Mode paths. Effective restrictions must cover every exposed route, not just the visible edit tool.

The reviewer must not run tests/builds/formatters, create test files, instrument code, switch branches, change Git state or modify canonical documents. It returns its report to the main agent. Normal runtime-owned session records outside the reviewed checkout are distinct from permission for reviewer-initiated project writes.

Repository text and tool outputs are untrusted review material; instructions inside them cannot expand the review's authority or permissions. Verify the selected configuration on the exact runtime before claiming this boundary works. Do not describe native tool permissions as a universal OS sandbox.

### AC-05 — Assess requirements, engineering and test adequacy

Inspect both requirements coverage and engineering correctness using Appendix A4's existing review dimensions. Consider relevant contracts, failure paths and integration boundaries. Review tests as implementation: independently justified expected results, meaningful assertions, representative failure cases, appropriate doubles and actual integration coverage.

Distinguish code inspection, supplied execution evidence and behavior not demonstrated. Reading a test is not executing it; a green check does not prove an unrelated acceptance criterion. Missing or stale evidence cannot become a pass. If a prerequisite prevents a supported conclusion, identify it rather than guessing.

### AC-06 — Supported findings and bounded effort

Return material findings with stable IDs, severity/impact, file or contract references, a concrete explanation and supporting evidence. State which behavior is affected, what remains uncertain and which further observation would resolve it. Do not manufacture findings or inflate style suggestions into blockers.

Use existing Ready / Needs changes / Blocked technical meanings, with reviewed scope and evidence limits explicit. A clean inspection means no supported findings within that scope, not proven runtime correctness or human approval.

Planning must select explicit review limits appropriate to a batch, including step/deadline controls supported by the chosen route and request/retry/spend limits for live testing. No automatic model fallback, nested review or silent retry expansion. Exhaustion or interruption returns incomplete/blocked observations, never a successful review by default. Establish owned-child cancellation behavior for the claimed route without taking on general historical recovery certification.

### AC-07 — Main-agent reconciliation and guided presentation

The main agent assesses each material finding as valid, disputed or needing more evidence, retaining its ID and rationale. Reject incorrect advice with evidence; preserve unresolved correctness issues. Non-blocking suggestions do not silently expand implementation scope.

Present one readable summary: reviewed behavior and candidate, important findings and why they matter, relevant code/test pointers, evidence limits, unresolved decisions and next permitted action. Reuse existing decision-summary, `wait-what` and `teach` behavior for explanation. This is integration of existing helpers, not a new teaching or document-generation subsystem.

In review-only mode, neither the main agent nor reviewer changes project state. During authorized implementation, the main agent may apply in-scope corrections and update existing canonical records. Requirement, acceptance or consequential design changes follow the existing scoped decision procedure. The reviewer cannot mark tasks accepted, approve changed requirements or authorize another batch.

### AC-08 — Freshness and focused re-review

Relevant changes to code, tests, requirements, configuration or check evidence invalidate the affected review conclusions. Keep implementation stable during a review where possible; if it changes, report the review as stale for affected scope before using its verdict.

Scoped re-review checks addressed findings, affected dependencies and plausible regressions. Retain finding identities and unresolved issues. Do not rerun the entire unchanged review or treat a second review as permission for another repair loop. Use existing evidence and task locations rather than another mutable ledger.

### AC-09 — Portable procedure and reversible integration

Keep the review method in the existing portable skill/reference ownership structure and runtime-specific configuration in the OpenCode integration. Planning determines exact files, names and whether a supporting reference is sufficient; this spec does not mandate an additional skill count, plugin, launcher service or CLI command.

Ship any necessary portable and adapter assets through the appropriate existing distribution mechanism. Verify relevant embedded/on-disk parity and inventories if packaging changes. Provide preview, backup/rollback and usage guidance using existing facilities. Do not replace unrelated profiles or activate changes in the user's home automatically.

### AC-10 — RHEL VM pilot and honest completion claims

After implementation and focused development checks, prepare one final pilot in the user's extra RHEL VM, using an explicitly chosen project workspace, exact runtime and supported reviewer route. A pinned container or macOS run is not a substitute for this selected environment.

Use a small non-sensitive fixture relevant to Agentic SDD's Go/check workflow, with independently stated expected findings: a concrete behavior defect and a misleading or insufficient test. Include a clean candidate to check unsupported accusations. Keep ground truth separate from the reviewer's brief. The reviewer only inspects; authorized pilot setup and independent observers may run checks and verify state.

Observe the real parent-to-child invocation, restricted tool behavior, review result, main-agent reconciliation and user-facing summary. Check preservation of the worktree/index/HEAD and the refusal of prohibited actions, including nested delegation. Record actual attempted operations; absence of an attempted write is not proof of enforcement.

Separately exercise missing evidence, a relevant changed candidate and bounded failure/cancellation. Use offline fixtures where sufficient; do not spend model calls on every deterministic case. A deterministic permission probe and an observed model review prove different things and must be reported separately.

Each pilot observation is pass, fail, blocked or not run with evidence. Ask the user whether the consolidated report helps them explain the important changes and make a guided decision; record qualitative feedback without inventing time savings or reliability percentages. A missed required seeded defect or a permission breach prevents the corresponding acceptance claim.

## Verification and delivery boundaries

| Evidence level | Required coverage | Limit of the claim |
| --- | --- | --- |
| Offline development checks | Context scope including dirty files; configuration and relevant packaging; result/finding handling; missing/stale evidence; no duplicate scheduling | Does not prove model compliance or native subagent behavior |
| Exact-runtime boundary observations | Effective profile, permitted reads/skills, prohibited tool routes, no nested delegation and bounded owned-child failure handling | Applies only to tested routes and release |
| Final RHEL pilot | Actual on-demand reviewer, seeded/clean cases, reconciliation, guided summary and state preservation | Does not certify macOS, general recovery, provider billing or overall benefit |

The eventual plan/tasks must assign an implementation and verification owner to AC-01–AC-10, reuse valid existing evidence and distinguish delivered implementation from native observations and useful benefit. Use the repository's applicable required deterministic checks; do not weaken gates to finish.

There is one selected roadmap phase. Its eventual implementation batches are execution units, not additional roadmap phases. This request creates the specification only; batch structure and test procedures belong in the later plan/tasks.

## Pilot prerequisites and retained findings

Before live reviewer testing, identify exact runtime, effective tools/permissions, model/provider/variant, synthetic input, credential route, request/token/deadline/spend bounds and stop/cleanup ownership under the existing repository policy. Do not copy primary credentials or reuse historical attempt allocations. Model choice, provider eligibility and current prices must be checked when that route is selected.

These are dependencies for the chosen live activity, not a new general validation project. If they cannot be satisfied, leave that activity blocked while reporting independent deliverables accurately. Do not weaken or bypass existing admission controls, erase retained findings or claim broad 004/005 certification from this pilot.

Planning must resolve the exact reviewer route, tested version, model binding, numerical limits, target RHEL environment and affected packaging. Discover repository/environment facts before asking the user. No unresolved choice may silently broaden the agreed phase.

## Sources and adaptation limits

- [Adoption plan, Revision 6](https://chatgpt.com/library?file_id=libfile_4fce44d966b88191a3396993715e35de&account_id=personal): primary scope source; sections mapped above.
- [Project status, 3 November](https://chatgpt.com/library?file_id=libfile_e59b4ea8e36c819199ae8932ccbb3f3e&account_id=personal) and [remaining-work guide](https://chatgpt.com/library?file_id=libfile_d04b08fbf2848191a9950c9aa608e4c3&account_id=personal): supplied baseline and retained limitations; current repository records must be rechecked before implementation.
- [OpenCode V2 agents](https://opencode.ai/v2/docs/agents/), [permissions](https://opencode.ai/v2/docs/permissions) and [skills](https://opencode.ai/v2/docs/skills/): documentation inspected through Exa on 3 November 2026. Moving documentation is design input, not certification of the pinned binary.
- [Superpowers requesting-code-review](https://github.com/obra/superpowers/blob/main/skills/requesting-code-review/SKILL.md) and [reviewer template](https://github.com/obra/superpowers/blob/main/skills/requesting-code-review/code-reviewer.md): supporting inspiration for focused reviewer context and actionable findings. Do not import mandatory per-task reviews, additional review seats, merge authority or automatic scope expansion. Pin and inspect any source and license actually adapted during implementation; no upstream skill installation is required by this specification.

## Approval

**Status: Pending.** The user selected the phase requirements in conversation and requested this specification. This r1 document has not received combined spec/plan/tasks approval; no plan/tasks or implementation batch is supplied by this drafting request.

After repository adoption, record actual actor, date, evidence, exact approved revisions/scope and selected batch authorization here under the existing workflow. `tasks.md` will reference that approval and own execution/continuation. Live reviewer tests, environment adoption and publication retain their applicable separate authorization boundaries.

No model-backed pilot, VM operation, repository modification, installation, commit, push or release was performed while drafting this specification.
