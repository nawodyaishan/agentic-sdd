# Agentic SDD Workflow Audit

Date: 24 September 2026. Audience: Nawodya. Scope: audit and proposed redesign; the uploaded skills have not been changed.

## Decision

Keep the chat-to-repository handoff and one primary coding agent. Replace artifact-driven phase routing with routing based on the actual readiness, scope and risk of the requested change. Load existing specialist skills on demand into the main agent. Preserve meaningful verification and human ownership of consequential decisions.

The immediate objective is more accepted changes per subscription allowance, with fewer context compactions and less review effort. Buying additional API capacity or changing models is secondary.

## Evidence and limits

I read all ten uploaded SKILL.md files in full. They total 3,691 whitespace-separated words and 26,163 characters. This is a file-size measurement, not a model-token measurement. Ten installed skills do not mean ten full skill bodies enter every prompt.

Exa searches requested 18 results across six queries covering four topics: skill/role mechanisms, context management, Spec Kit adoption, and Zed integrations. Search counts include overlapping results. Selected official pages were fetched and checked; the references below support the product claims.

Not available: actual generated project documents, usage transcripts, installed client versions/configuration, specialist skill contents, the precise Codegraph implementation, or whether MCPs are global or repository-local. Consequently, the audit establishes instructions that can cause overhead, not a measured allocation of your subscription consumption or a guaranteed savings percentage.

Your answer “No” to the upstream-document approval question is ambiguous. This proposal provisionally treats SRS/technical-spec/task documents as drafts: preserve their content and flag material gaps rather than silently accepting or rewriting them. If they are already approved, keep decisions locked and ask only about conflicts or proposed departures.

## What the supplied skills actually require

| Skill | Concrete finding | Recommended change |
|---|---|---|
| `agentic-sdd-router` | Routes on missing `spec.md`, `plan.md`, and `tasks.md`; does not explicitly recognize your `Docs/SRS.md`, `High Level Spec.md`, or `Tasks.md`. Discovery expects lowercase `docs/`. It routes by workflow phase but has no specialist skill selection step. | Resolve the actual repository paths and canonical sources first. Determine whether the requested slice is ready, then choose only missing work and its domain skill. Retain a full route for changes that warrant it. |
| `agentic-sdd-bootstrap` | Calls for 12 templates, two review checklists, a constitution, and AGENTS.md: 16 file creations/updates if each template is a file, before optional client shims. Directories are additional but do not themselves consume model tokens. | Default to discovery and a compact instruction entry point. Reuse existing governance. Keep reusable templates with the skill package and materialize a template only for a real need. |
| `agentic-sdd-spec` | Reads `existing specs/*/spec.md` without a relevance bound; always creates a requirements checklist alongside the specification. | Inspect an index or targeted search first. Read related specifications only. For a small change, put acceptance checks in the existing task rather than a second document. |
| `agentic-sdd-plan` | Requires an 18-part plan structure and can add data-model, contract, test-plan and ADR artifacts when relevant. | A short technical approach is enough for many changes. Separate artifacts only when they have an independent consumer or lifecycle. Preserve real API contracts and migration plans. |
| `agentic-sdd-architecture-review` | Mandatory gate between planning and code; writes approval into review.md, but updates approval in plan.md only if the user explicitly asks to maintain artifacts. | Make review proportional to architectural risk. Store human authorization once, separately from the agent's review verdict, and reference that state consistently. |
| `agentic-sdd-tasks` | Requires spec, plan, review and constitution; repeats up to 12 fields per task plus verification/dependency summaries. | Refine the relevant section of your existing Tasks.md. Put shared boundaries at feature level; repeat only task-specific exceptions. Do not require invented file ownership before a new repository has a design. |
| `agentic-sdd-implement` | Requires the full spec/plan/tasks/constitution packet and optional review/test-plan for each invocation. It already prohibits unrequested delegation and preserves scope. | Keep those good implementation boundaries. Read the current task, its source sections, relevant code/tests and actual constraints; expand context only when a concrete question requires it. Honor authorization already granted for the same scope. |
| `agentic-sdd-verification-review` | Overlaps implement's verification and drift-retro's drift checks. Its artifact writes are already conditional on being requested. | Retain it as the main completion check. Reuse evidence for the same code state; rerun checks affected by subsequent edits or required by CI. Report concrete defects and commands rather than broad reassurance. |
| `agentic-sdd-drift-retro` | Always creates/updates both a drift report and retrospective, even if the finding is “no drift”; adds release notes for user-facing work. | Check drift during verification. Write a separate report only for a meaningful discrepancy; write a retrospective after a useful incident/lesson or milestone; batch release notes by release. |
| `agentic-sdd-research-spec` | Broad trigger overlaps spec/plan/tasks; workflow includes Exa research and a large document structure. It is absent from the router's phase table. | Turn it into an optional research operation driven by named unresolved questions. Integrate decision-relevant evidence into the current artifact instead of creating another complete spec. |

These are mostly process-design issues, not excessively long individual SKILL.md files. Their sizes are approximately 292–544 words each.

### Specific coordination defects

1. **Duplicate source risk.** A usable upstream technical spec can exist while the router still sees “no plan.md.” The agent is then directed to recreate a technical plan under a new name.
2. **Approval state mismatch.** The router checks approval in plan.md while architecture-review primarily records it in review.md. A later invocation can believe the review is still missing. This is a possible loop established by the instructions, not evidence that it happened in your sessions.
3. **Repeated reading.** Several stages independently require overlapping packets. Instructions do not distinguish content already read and unchanged from content that needs refreshing.
4. **No specialist selection mechanism.** The router selects a phase, not Go/backend/React/platform expertise. Role names in an upstream task document do not by themselves load an installed skill.
5. **Nonportable reference.** The router names `/Users/nawodyaishan/Downloads/production-spec-driven-agentic-coding-guide.md`. This is a provenance pointer, not an explicit mandatory read, but another machine cannot rely on it being available.
6. **Client-specific tool assumptions.** Some frontmatter names `web_search_advanced_exa` and `web_fetch_exa` directly. Actual MCP tool identifiers can differ by server/client. Declare intent in portable instructions and verify native tool mappings locally.
7. **Risk labels are not permissions.** Fields such as `risk: safe` are metadata unless a particular host interprets them. Several documentation skills also list Write, Edit and unrestricted Bash. In Claude Code, `allowed-tools` preapproves access; it is not a universal sandbox or exclusive tool allowlist. Use runtime permission configuration for real enforcement. [4][6]

Do not erase permissions to reduce friction. Distinguish permission to draft a migration or Terraform change from permission to execute it against real data or infrastructure. A single explicit scope approval should not need repeating at every workflow phase.

## Proposed operating model

### 1. Preserve and map the handoff

Keep the documents produced in Claude/ChatGPT when they are useful. In the existing repository instruction file, add a short mapping to actual paths:

- Product intent: SRS.md or the product document the user designates.
- Technical decisions: High Level Spec.md, plus relevant ADRs.
- Work/status: Tasks.md.
- Implementation evidence: source, tests, CI configuration and current diff.

Respect `Docs/` versus `docs/`; this distinction matters on case-sensitive systems. Do not rename files just to satisfy a skill's convention.

Read sufficient upstream context during intake to understand the project. Later, retrieve the relevant sections for the current slice. Long documents are not inherently wrong; repeatedly loading unrelated sections is the concern.

For conflicting documents, flag the conflict and resolve it. Do not create another canonical document as an automatic response.

### 2. Route by readiness and risk

The router should answer, briefly:

1. What observable change is requested?
2. Which existing source defines it, and is it sufficiently clear?
3. Which architectural or operational boundaries does it affect?
4. What is the smallest independently reviewable slice?
5. Which existing specialist skill and verification steps apply?

Then choose one of these proposed defaults:

| Change | Planning/documentation | Review |
|---|---|---|
| Small reversible fix or mechanical edit | Existing task and a few acceptance bullets; usually no new MD file | Focused regression check and diff review |
| Bounded feature using established architecture | One task section containing acceptance criteria, short approach and verification | Integration/contract checks where relevant |
| New architecture, sensitive permissions, consequential API change, migration or production operation | Explicit design/decision record and execution/rollback scope; dedicated artifacts where justified | Deliberate architecture and operational review |

These are recommended policies, not vendor requirements. File count and elapsed time alone are poor risk classifiers: a two-line permission change can deserve more scrutiny than a large mechanical refactor.

For greenfield work, establish just enough structure and toolchain for the first end-to-end slice. Authorize required project dependencies as part of that setup scope. For existing systems, use actual architecture/test conventions and verify compatibility. Current Spec Kit guidance also says not to reconstruct an entire existing system before taking on a bounded change. [8]

### 3. Use domain skills inside the main agent

Keep phase and expertise separate. “Implement” is a phase; “Go backend” is the expertise needed during that phase. Start with one relevant phase procedure and one principal specialist skill. Load a second specialist only when the task crosses a real boundary.

| Work | Main-agent role | Existing specialist skill to map | Verification examples, adapted to the repo |
|---|---|---|---|
| Unresolved architecture or requirements | Technical architect | Architecture / specification skill | Acceptance criteria, compatibility and failure-mode review |
| Go service behavior | Go backend engineer | Your Go quality / architecture skill | Formatting and focused Go tests; race checks when concurrency warrants them |
| TypeScript API | TypeScript backend engineer | Your TS backend skill | Typecheck, focused tests, validation/error/contract checks |
| React interaction | Frontend engineer | Your React / accessibility skill | Component or interaction tests; browser check when needed |
| Kubernetes manifests | Platform engineer | Your Kubernetes skill | Render/schema/policy checks appropriate to the repository |
| Terraform module/configuration | Infrastructure engineer | Your Terraform skill | Formatting/validation and plan review in the authorized environment |

Exact skill names and paths must come from your installed catalog; they were not included in these attachments. Map by task semantics plus repository paths, not extension alone. React and a TS backend both use TypeScript; arbitrary YAML is not necessarily Kubernetes.

Put this small map in an existing instruction file or router reference. Do not create a role charter and agent directory for every discipline. At task start the agent can emit one line such as: `Role: Go backend; skill: <actual installed name>; scope: order cancellation; checks: focused service tests.` This makes selection observable without another document.

### Native mechanisms and limits

- **Claude Code:** skills can run inline in the main conversation. Use concise descriptions for selection and supporting references for infrequent detail. Path-scoped `.claude/rules` can add small invariants when matching files are read. Do not put a full engineering handbook into always-loaded rules. [4][5]
- **Codex:** skill descriptions support implicit selection; the body is loaded when selected. Current docs also expose invocation policy through `agents/openai.yaml`. Keep common procedure text portable and client-specific configuration separate. [1]
- **Bootstrap:** because you already invoke it manually, keep it explicitly initiated. Claude's `disable-model-invocation: true` and Codex's `allow_implicit_invocation: false` are different native controls. Disabling automatic invocation may also prevent a router from invoking it automatically; have the router recommend the explicit bootstrap command instead. [1][4]
- **Role switching is not context isolation.** Skill text already loaded into the main thread can remain there. Current Claude docs describe skill reinjection after compaction as well. On-demand loading avoids unnecessary initial work; it does not erase old instructions. Use a fresh task session after a completed slice when accumulated context becomes irrelevant. [2][4]
- **Optional independent review:** a separate reviewer can help for difficult/high-risk changes, but switching the same model's persona is not independent verification. Do not make a second agent mandatory for ordinary tasks.
- **Zed:** native Zed Agent profiles do not automatically govern external Claude/Codex agents or terminal threads. Those usually have their own tools and configuration; Zed can forward MCP servers via ACP. Check the actual runtime before changing profiles. [9][10]

## MCP policy for the tools you named

| Tool | Trigger | Bound the returned context |
|---|---|---|
| Context7 | A precise question about a library/API/version needed for the task | Resolve the correct library/version; request only the relevant topic; stop when answered |
| Exa | External architecture evidence, service behavior, alternatives, or information absent from local/versioned docs | Start with a small relevant result set and snippets; fetch the best primary pages |
| Codegraph | Callers/callees, dependencies or change impact that direct search does not answer efficiently | Request relevant symbols/relationships first; expand depth/source only for unresolved questions |
| Local search/read | Existing implementation and tests | Scope paths and patterns, inspect relevant matches |

Do not force both Exa and Context7 on every task. Do not repeat a recent lookup unless the version, question or evidence changed. A backend role does not require a standing research session.

“Codegraph” identifies multiple projects, so exact flags and indexing advice require your server's package/repository name. Backend indexing work outside the model is different from text returned into its context.

My earlier suggestion that all enabled MCP schemas necessarily dominate every session was too broad. Current Claude configurations support tool search and deferred definitions; availability varies by client/model/provider configuration. Returned documentation and graph content still enters the conversation. Verify actual usage before adopting aggressive server toggling or more tooling. [2][7]

A skill can guide tool selection; it cannot universally turn servers on/off or create permissions just by declaring a role. Runtime access controls remain authoritative. [6]

## Documentation and completion policy

Suggested default policy to incorporate into a future revision:

- Reuse the existing canonical docs and current task section.
- Create a new document only when it serves a distinct long-term need or is requested.
- Keep templates with reusable skills; instantiate only needed artifacts.
- No separate “no drift” document, per-task retrospective or routine status report.
- Record acceptance criteria and verification where the task already lives.
- Keep real ADRs, API contracts and migration/rollback documentation when warranted.
- Read related source sections instead of every spec in the repository.
- Review the current diff and actual test outcomes; do not regenerate plans merely because execution started.
- Preserve authorization for the same approved scope, and seek new authorization when the scope or consequences materially change.

Keep a compact continuation state in the existing task, containing the current slice, decisions, changed paths, checks and unresolved issue. Update it at a milestone, not after every command. This gives a fresh session enough context without adding another permanent handoff document.

## Subscriptions and measurement

Your limiting resource is subscription allowance and usable context, so optimize those before paying for another provider. API price per million tokens is not the same as marginal cost inside an existing subscription. Prompt caching can lower billed API input cost without freeing context-window space.

Use the existing primary subscription for implementation. Start with its ordinary reasoning setting for routine work and increase effort for actual difficulty. Reserve a second provider for a concrete failure, difficult review or allowance fallback. Cheap API summaries may help on selected tasks, but introduce another bill and can discard important details; do not make them a required step.

The earlier specific model/price shortlist is not a sufficient basis for a purchase decision; this audit does not revalidate it or recommend those exact routes.

Run a small before/after comparison using similar task types:

1. Record startup context before work and after router/bootstrap.
2. Record new/updated document count, compactions, elapsed time, your review minutes and acceptance outcome.
3. In Claude use `/context` to inspect categories and `/usage` for plan usage/activity. The displayed dollar estimate is not a subscription invoice. [2][3]
4. For Codex use the status/usage information available in your installed client; do not assume identical metering across providers.
5. Change bootstrap and routing first; keep the model stable so the comparison is interpretable.
6. Then add the existing specialist skill map; check whether correct skills load without requiring manual reminder.
7. Test one small fix, one normal feature and one infrastructure task. Audit missing verification as well as token reduction.

Proposed acceptance checks for the revised workflow:

- Existing usable source documents do not trigger duplicate specs due to filename mismatch.
- A small fix creates zero new planning MD files by default.
- Bootstrap does not instantiate 12 templates automatically.
- The selected specialist skill matches the task and is visible in the initial status.
- One consistent authorization state prevents needless repeated approval requests.
- Completion includes relevant verification with no false success claims.
- Review time and defects do not worsen while context/allowance consumption improves.

No numeric savings promise is justified until real sessions are measured.

## Prioritized revision order

1. Router: existing-document recognition, risk/readiness classification and specialist selection.
2. Bootstrap: minimal, idempotent setup with templates kept outside each project.
3. Approval state: fix review.md/plan.md disagreement without weakening the human gate.
4. Spec/plan/tasks: permit a single existing task section for ordinary changes.
5. Implement/verify: selective context and reuse of still-valid verification evidence.
6. Drift/retro/research: invoke only when there is useful work to do.

You can retain the ten skill names for compatibility. The important change is making the expensive branches conditional, not replacing ten files with ten new files.

## Source references

1. [OpenAI: Agent Skills](https://developers.openai.com/codex/skills) — progressive disclosure, implicit/explicit activation and invocation policy.
2. [Anthropic: Explore the context window](https://code.claude.com/docs/en/context-window) — startup context, skill/rule behavior after compaction and `/context`.
3. [Anthropic: Manage costs effectively](https://code.claude.com/docs/en/costs) — usage tracking and distinction between API estimates and subscription billing.
4. [Anthropic: Extend Claude with skills](https://code.claude.com/docs/en/skills) — inline/forked skills, invocation settings and allowed-tools semantics.
5. [Anthropic: Store instructions and memories](https://code.claude.com/docs/en/memory) — AGENTS.md import and path-scoped rules.
6. [Anthropic: Configure permissions](https://code.claude.com/docs/en/permissions) — runtime enforcement versus instruction text.
7. [Anthropic: Connect Claude Code to tools via MCP](https://code.claude.com/docs/en/mcp) — discovery/tool search, configuration-dependent behavior and server controls.
8. [GitHub Spec Kit: Adopting in an existing project](https://github.com/github/spec-kit/blob/main/docs/guides/existing-projects.md) — bounded changes without documenting the entire existing system first.
9. [Zed: Agent Profiles](https://zed.dev/docs/ai/agent-profiles) — profile scope and permission distinction.
10. [Zed: External Agents](https://zed.dev/docs/ai/external-agents) — external runtimes and MCP/configuration boundaries.

Primary local evidence: all ten user-supplied skills, identified by their frontmatter names above. The original files were read as audit material, not adopted as operating instructions for this review.
