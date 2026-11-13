# Current state: research skills and subagents

**Date:** 2026-11-12
**Scope:** how research is represented, routed, delegated and evidenced in this
repository and on the current machine.
**Status:** study and advisory report only. It creates no approval, batch
authorization, installation permission or live-apply authorization.

## 1. Summary

Research is not a separate product feature here. It is a **conditional,
evidence-producing step** that must return to whichever SDD stage was blocked.
The logic exists in four layers, and only some of them are installed:

| Layer | Artifact | Current machine state |
|---|---|---|
| Portable core | `skills-files/agentic-sdd-research-spec/SKILL.md` + router references | Installed in all four client skill directories |
| Optional orchestration pack | `opencode-pack/agents/sdd-research-librarian.md`, `agentic-sdd-orchestrate` | Source present in the repo; **not installed** in `~/.config/opencode` |
| Installed fleet | `~/.config/opencode/agents/sdd-lightweight.md` | Installed, but the older local-only variant |
| Global third-party skill | `~/.config/opencode/skills/mattpocock-skills/skills/engineering/research/SKILL.md` | Present and available as `research` |

The repository therefore contains **two delegated-research designs** and the
machine exposes **two skills named around research**, but the routing rules
between them are only partly written down.

**Short answer on stage fit:** current logic addresses the *need* for research
at spec, plan, tasks, verification and blocker resolution through one
cross-cutting "named uncertainty" route. It does **not** assign a research
agent to those stages. Only `agentic-sdd-debugging` names research inline;
only the opt-in orchestration pack defines a delegated research role; and on
this machine only `sdd-lightweight` is actually installed, with local-only
permissions.

## 2. What exists, by layer

### 2.1 Portable core skill: `agentic-sdd-research-spec`

File: `skills-files/agentic-sdd-research-spec/SKILL.md`

- **Trigger:** an explicit research request, or one named uncertainty that
  actually blocks drafting, implementation, review or a direct fix.
- **Method:** state the question, the decision it affects, the relevant
  version/environment and the evidence threshold. Read local source and
  versioned documentation first. Only then select an external tool.
- **Tool selection is need-based, not identifier-based:** local code discovery,
  a versioned documentation tool, or a web search/fetch tool. The skill
  explicitly says not to assume an MCP server is installed or reachable, and to
  report an unavailable capability and use a transparent fallback.
- **Output location:** where the work lives. Feature work records the answer in
  the relevant `spec.md` or `plan.md` section; a direct fix records it in the
  response and the change itself. `research.md` is allowed only for a separate
  consumer, repository convention or explicit request.
- **Approval behaviour:** material new evidence that changes approved
  requirements or design returns the affected changes for human review.
- **Delegation:** do not spawn subagents unless the user requests them.
- **Prohibitions:** do not turn research into an alternate spec, silently add
  scope, or claim resolution without evidence.

### 2.2 Router integration

Files:

- `skills-files/agentic-sdd-router/SKILL.md`
- `skills-files/agentic-sdd-router/references/workflow-policy.md`
- `skills-files/agentic-sdd-router/references/specialists.md`
- `skills-files/agentic-sdd-router/references/human-input.md`

The router classifies the requested action before feature intake. Its action
table has a dedicated row: **"Research for a named uncertainty"** →
`agentic-sdd-research-spec`, then return to the invoking stage. The same route
applies inside an approved feature when a named factual question blocks
drafting, implementation, review or a direct fix.

Relevant constraints:

- Research is conditional; it does not require a particular MCP or external
  search.
- Research artifacts are conditional, like clarification documents, ADRs and
  verification reports: a distinct consumer, actual risk, repository convention
  or explicit request. "No automatic extras."
- Human questions cover intent, unavailable stakeholder knowledge or
  consequential tradeoffs — **no implicit research delegation** for facts that
  the main agent can discover itself.
- The specialist map assigns a versioned documentation tool (Context7-style)
  for uncertain library behaviour, a web search/fetch tool (Exa-style) for
  external technical research, and CodeGraph for local cross-module discovery.
  These are candidate selections resolved from the active client, never assumed
  identifiers.
- `agentic-sdd-teach` cross-references research permissions but does not grant
  automatic delegation.

`agentic-sdd-questionnaire` is adjacent but distinct: it is for knowledge held
by a person, explicitly not a way to shift routine research to the user.

### 2.3 Opt-in orchestration: `agentic-sdd-orchestrate` and `sdd-research-librarian`

Files:

- `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`
- `opencode-pack/skills/agentic-sdd-orchestrate/references/context-and-cost.md`
- `opencode-pack/agents/sdd-research-librarian.md`
- `opencode-pack/routing.md`

The orchestration skill is for the human-facing main agent, not a child. It
classifies work as `D0` direct, `D1` cheap read-only, `D2` authorized writer,
`D3` advisor/research/vision, `D4` critical-main, or `blocked`. External
documentation research sits in **D3**. Before dispatch the parent must state the
role, objective, scope, evidence and permissions, and must obtain explicit
authorization for **external research** among other side effects. No nested
delegation; one child at a time when possible.

`sdd-research-librarian` is the pack's research specialist:

- **Mode:** subagent; no repository writes.
- **Pinned model:** `opencode/deepseek-v4.1-flash-free`; documented alternatives
  `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high`.
- **Allowed tools:** read, glob, grep, skill, execute, CodeGraph, Context7,
  `webfetch`, `websearch`, and the two Exa tools (`exa_web_search_exa`,
  `exa_web_fetch_exa`). Denies edit, shell, subagent and `.env` reads.
- **Method:** loads `agentic-sdd-research-spec`. Official documentation through
  Context7 first, then vendor pages/specifications via Exa or web fetch. Prefer
  the version matching this project and state the version checked.
- **Return contract:** answer, exact source URLs, the supporting section or
  quote, source date, applicability to the project version, contradictions, and
  what the evidence does not settle. Never fabricate a URL, version or quote,
  and never present a search snippet as verified content.
- **Privacy rule:** no private repository code, file contents, internal
  identifiers, customer data, credentials, secrets or infrastructure details in
  an external query.

`routing.md` names five "high-intensity" roles allowed to use Exa:
`sdd-architecture-review-9r`, `sdd-code-review-9r`,
`sdd-architecture-advisor`, `sdd-research-librarian`, `sdd-openai-architect`.
`internal/appops/routing.go` and `internal/skillsync/opencode_pack_test.go`
encode the same model pins and role sets as machine-checked data.

### 2.4 Installed local discovery: `sdd-lightweight`

File: `~/.config/opencode/agents/sdd-lightweight.md` (machine)
File: `opencode-pack/agents/sdd-lightweight.md` (repo pack version)

The installed profile is the one this session can actually delegate to. It is
much narrower than the pack source:

- Deny-by-default; allows `read`, `glob`, `grep` and
  `codegraph_codegraph_explore`.
- **No `execute`**, therefore no Context7 or CodeGraph CLI through Code Mode.
- **No Exa or web search/fetch.**
- Denies `edit`, `shell` and `subagent`. `.env`, `*secret*` and
  `*credential*` reads are denied.
- `steps: 10`; model `opencode/beam-free`.
- Returns paths and line references, scope inspected, uncertainty, and exact
  error text. The parent validates consequential claims.

This makes it a **local, bounded discovery helper**, not an external research
agent. The repo pack version adds `execute`, `context7_*` and an explicit
`exa_*` deny, but that version is not the installed one.

### 2.5 Global third-party `research` skill

File: `~/.config/opencode/skills/mattpocock-skills/skills/engineering/research/SKILL.md`

This is separate from the Agentic-SDD portable skills and is currently
available in the session as `research`.

- **Trigger:** the user wants a topic researched, docs/API facts gathered, or
  reading legwork delegated.
- **Method:** spin up a **background agent** with fresh context, investigate
  primary sources, follow every claim to the owning source.
- **Output:** write findings to a **single Markdown file**, citing each claim,
  in the repo's existing notes convention; if none exists, choose a sensible
  location and say where.

It has no model pin, no permission model and no SDD approval/evidence rules.
It is a generic engineering skill, not an SDD phase.

### 2.6 Tool layer actually available

From `~/.config/opencode/opencode.json` and repository guidance:

- **CodeGraph:** local indexed code discovery (`codegraph_codegraph_explore`,
  or CLI fallback); repository-bootstrap rules live in `AGENTS.md` and
  `docs/codegraph.md`.
- **Context7:** versioned library/framework documentation.
- **Exa:** remote MCP with `web_search_exa`, `web_fetch_exa`, `agent_run` and
  `web_search_advanced_exa`; API key from
  `~/.config/opencode/secrets/exa-api-key`.
- **Built-in** `websearch` / `webfetch` where the active profile allows them.

The librarian profile allows only the two Exa search/fetch tools, not
`agent_run`; it additionally allows built-in web search/fetch. Other profiles
are inconsistent only in surface form: non-high-intensity profiles declare
`exa_*` with `effect: deny`, which has the same result.

## 3. Current routing logic end to end

1. **Classify the request.** Is it an explicit research request, or a named
   factual uncertainty that blocks current work? If neither, do not research.
2. **Preserve the mode.** Feature drafting, implementation, review and direct
   fixes keep their existing restrictions; research returns to that stage.
3. **Read local evidence first.** CodeGraph for local symbols and dependencies;
   targeted source reads otherwise.
4. **Escalate only for a concrete gap.** Context7 for versioned library
   behaviour; Exa/web fetch for external technical facts. Prefer primary
   sources, fetch only decision-relevant material, record the version checked.
5. **Apply the evidence threshold.** State what the evidence does and does not
   settle. Never present a snippet as verified content.
6. **Record where the work lives.** `spec.md`/`plan.md` for feature work; the
   response and change for a direct fix; `research.md` only for a separate
   consumer or explicit request.
7. **Return to the invoking stage.** Material new evidence that changes
   approved requirements goes back to human review.
8. **Delegation is a separate decision.** The core skill says no subagents
   unless requested. The orchestration pack allows a bounded `D1` child and a
   `D3` librarian, but only with the parent's explicit authorization for
   external research. On this machine, automatic delegation is authorized only
   for `sdd-lightweight`, and it is local-only.

## 4. Stage-level coverage: where research agents help and what the current logic does

You identified five points where research agents are useful: **spec, plan,
tasks, verification and bug-fix blocker resolution**. The current logic treats
research as a cross-cutting route for a named blocker, not as a stage-owned
delegation. It therefore supports research *content* at every stage, but
supports a research *agent* at almost none of them.

### 4.1 Coverage matrix

| Stage | Why research helps | What the stage skill actually says | Research agent available today | Coverage |
|---|---|---|---|---|
| **Spec** | External constraints, standards/API facts, feasibility and prior art that change acceptance criteria or scope. | Read relevant SRS/roadmap; "do not silently assume risky facts"; use human-input, grilling or questionnaire for material choices. No research trigger and no delegation. `agentic-sdd-research-spec` is reachable through router classification or direct invocation, not through this skill. | None stage-owned. The user may explicitly ask for one; `sdd-lightweight` can only do local discovery. | **Partial** — the need is covered by the cross-cutting route, but there is no stage-local trigger, handoff or evidence slot. |
| **Plan** | Library/API behaviour, version compatibility, design alternatives and operational constraints that affect the chosen approach. | Assigns specialists and tools once, including "versioned library docs, external search" as **conditional selections, not mandatory calls**. Optional `agentic-sdd-architecture-review`. No research-agent role. | External tools run in the main agent. The librarian is available only if the pack is installed **and** the user explicitly authorizes external research. | **Partial** — tool assignment covers the need, but there is no delegated research handoff. |
| **Tasks** | Verification design, real test commands, dependency/environment facts and task decomposition. | Derives tasks from the spec and plan; records only specialist/tool **exceptions**. No research reference. | None. Research inputs must already exist in the spec/plan or be obtained before drafting. | **Not addressed** — no research step is designed into this stage. |
| **Verify** | Whether a finding is real, what the versioned contract actually says, and whether tests are meaningful. | A named uncertainty is diagnosed with `agentic-sdd-debugging` "through permitted observation"; review-only diagnosis recommends instrumentation/checks. Independent review only on explicit request; otherwise self-review. No research handoff. | None stage-owned. Explicit user request could add one, but it is not part of the review contract. | **Indirect only** — the debugging pointer carries the fact-finding need; review remains non-mutating. |
| **Bug-fix blocker resolution** | Root-cause evidence, exact error semantics, version-specific behaviour and workaround facts. | `agentic-sdd-debugging` runs in **this same main agent**; step 3 says "use versioned research only for a named factual uncertainty, then return here." `agentic-sdd-implement` says: "Use one main agent. Do not spawn subagents unless the user explicitly requests them." | No. A research agent can supply a named external fact, but diagnosis and the fix stay in the main agent. | **Partial and deliberately constrained** — blocker-resolution logic exists and is strong, but it is not a research-agent path. |
| **Any stage: named blocker** | One blocking fact outside local knowledge. | Router classifies it and routes to `agentic-sdd-research-spec`, then returns to the invoking stage. Evidence goes into `spec.md`/`plan.md` or the direct-fix response; `research.md` only for a separate consumer or explicit request. Delegation is not automatic. | Only if the user requests it, or through the opt-in orchestration pack with explicit authorization. The librarian is not installed. | **Yes at routing level** — operationally limited by installation and authorization. |

### 4.2 What the current logic actually addresses

- **It addresses research as a cross-cutting blocker route, not as a stage
  role.** The router has an explicit research row, and the workflow policy says
  a named factual uncertainty may block any stage. That is the only place where
  all five stages converge.
- **Spec and tasks have no research language at all.** Spec mentions risky
  assumptions but routes them to human input/grilling/questionnaire, not a
  research agent. Tasks inherits its research inputs from the spec and plan.
- **Plan addresses research as a tool assignment, not an agent.** It names
  versioned documentation and external search as possible tools; the main agent
  is expected to use them. Nothing says a child should do the reading.
- **Verification addresses research indirectly through debugging.** It does not
  define a research step, and its review-only mode is explicitly non-mutating.
- **Blocker resolution is intentionally a main-agent activity.** Debugging is
  "reusable process guidance, not a domain specialist or permission to
  delegate." Implementation repeats the one-main-agent rule. A research agent
  can answer a named external fact; it is not designed to take over diagnosis.
- **Delegated research is an opt-in exception.** The orchestration pack puts
  external documentation research in `D3` and requires explicit authorization.
  Without that pack and an explicit request, the stage skills keep research in
  the main agent.
- **The only automatic delegation currently authorized is local.** Global
  `AGENTS.md` permits automatic delegation to `sdd-lightweight`; it is
  read/glob/grep/CodeGraph only, with no web, Context7 or `execute`. It can
  support local discovery for spec/plan/tasks/verification, but cannot resolve
  an external fact.

### 4.3 Verdict

Current logic is **strong on blocker handling and weak on stage-owned research
delegation**:

- Strong: one canonical research route, evidence placed where the work lives,
  approval preservation, explicit no-fabrication sources, and a clear
  main-agent diagnosis loop.
- Weak: no stage-level contract for invoking a research agent from spec, plan,
  tasks or verification; no installed external librarian; no rule for how a
  stage consumes a research agent's output; no precedence over the global
  `research` skill.
- Bug-fix blocker resolution is the clearest example: the workflow wants
  research *evidence* there, but deliberately keeps diagnosis and correction in
  the main agent.

### 4.4 If research agents should be used at those stages

Advisory options only; each is a change decision, not an authorized edit.

1. **Keep the current main-agent model.** Use CodeGraph/Context7/Exa directly
   and record evidence in `spec.md`/`plan.md` or the direct-fix response. This
   needs no installation and preserves the existing stage contracts.
2. **Install the opt-in research specialist.** Preview
   `agentic-sdd preview --with-opencode`; only after separate authorization,
   `apply --with-opencode` installs `sdd-research-librarian` and
   `sdd-orchestrator`. Installation alone is not enough: the stage skills still
   need a handoff sentence saying when to invoke it and where its output goes.
3. **Add stage handoff language deliberately.** For example: plan assigns the
   librarian for named external questions; tasks references the resulting
   evidence; verification may request a read-only fact check; debugging may use
   the librarian for a named fact while retaining diagnosis in the main agent.
   This is feature/document work and needs its own combined approval.
4. **Use the global `research` skill only outside SDD work**, or write a
   precedence rule that its background-agent and always-write-a-file behaviour
   does not apply inside an SDD stage unless the user explicitly asks.

## 5. Current-state gaps and risks

| # | Observation | Consequence |
|---|---|---|
| 1 | The repo contains `sdd-research-librarian` and `sdd-orchestrator`, but `~/.config/opencode/agents` has 18 files and neither profile. | No on-machine external-research specialist. Research delegation currently means `sdd-lightweight` (local) or main-agent tools. |
| 2 | The installed `sdd-lightweight` lacks `execute`, Context7 and Exa; the repo pack version is more capable. | Local discovery works through direct read/grep/CodeGraph, but the child cannot consult library docs or run CodeGraph CLI. This matches the older machine fleet documented in `docs/subagent-fleet-comparison.md`. |
| 3 | Two different research patterns exist: the SDD skill (inline, blocker-driven, no artifact by default, no subagents by default) and the global `research` skill (background agent, always a Markdown file). | In an SDD session the two can conflict on delegation and artifact creation. No written precedence rule currently says which wins. |
| 4 | Naming is crowded: `agentic-sdd-research-spec`, `sdd-research-librarian`, `sdd-lightweight`, and the global `research` skill. | Users and agents can route "research" to the wrong layer. The router table is the only disambiguator. |
| 5 | The global `AGENTS.md` authorizes automatic delegation only to `sdd-lightweight`; the orchestration pack requires explicit authorization for external research. | Consistent, but it means external research is never automatic and cannot be assumed available. |
| 6 | `docs/subagent-fleet-comparison.md` is a dated snapshot (2026-11-10); `docs/mcp-default-integration-study.md` is current for MCP integration. | Fleet counts and permission claims can drift after any install. Treat the comparison as historical evidence, not live state. |
| 7 | The core skill says a `research.md` file is conditional; the global `research` skill says to always write a Markdown file. | An SDD invocation that loads both could create an unrequested artifact, violating the "no automatic extras" policy. |
| 8 | The Exa MCP exposes `agent_run`, but no research profile allows it. | Multi-step Exa research is not available to subagents; only search/fetch are. Intentional-looking, but undocumented as a decision. |
| 9 | Spec, plan, tasks and verification do not define a research-agent handoff; only debugging names research inline, and implementation requires "one main agent." | Even after installing the librarian, those stages have no written trigger, brief or output-consumption rule. Installation alone would not produce stage-level research delegation. |
| 10 | The global `research` skill is stage-unaware and always writes a Markdown file. | It cannot serve the stage contracts as-is: it bypasses the conditional-artifact rule and does not return to an SDD stage with a named evidence boundary. |

## 6. Advisory recommendations

These are recommendations only; none is an authorized change.

1. **Write down precedence.** State that inside SDD feature or direct-fix work,
   `agentic-sdd-research-spec` and the router's workflow policy govern, and the
   generic `research` skill's background-agent and always-write-a-file
   behaviours do not apply unless the user explicitly asks for them.
2. **Choose the delegated path deliberately.** Either keep research in the main
   agent with CodeGraph/Context7/Exa and record evidence in `spec.md`/`plan.md`,
   or install the pack's `sdd-research-librarian` with
   `agentic-sdd preview --with-opencode` and, only after explicit
   authorization, `apply --with-opencode`. A real-home `apply` is a live
   operation and needs its own decision.
3. **Do not silently upgrade the installed `sdd-lightweight`.** Its local-only
   scope is a real safety boundary, and editing it outside the installer would
   diverge from managed state.
4. **Refresh the fleet comparison after any install.** The authoritative live
   check is `ls ~/.config/opencode/agents` plus the installed profile bodies;
   `docs/subagent-fleet-comparison.md` is a snapshot.
5. **Keep the external-query privacy rule visible** wherever external research
   is enabled. It is currently stated in the librarian prompt, `routing.md` and
   the global `AGENTS.md` context, but not in the core research skill.
6. **Decide the stage handoff before installing anything.** If research agents
   should serve spec, plan, tasks or verification, define the trigger, the
   bounded question, the evidence location and the return-to-stage rule in
   those skills or in `plan.md`. A profile without a stage contract changes
   capability but not workflow.
7. **Keep bug-fix diagnosis in the main agent.** Use a research agent only to
   resolve a named external fact during blocker resolution; do not delegate
   reproduction, hypothesis selection or correction, which the debugging and
   implementation skills deliberately keep in the main session.

## 7. Evidence inspected

- `skills-files/agentic-sdd-research-spec/SKILL.md`
- `skills-files/agentic-sdd-router/SKILL.md`
- `skills-files/agentic-sdd-router/references/workflow-policy.md`
- `skills-files/agentic-sdd-router/references/specialists.md`
- `skills-files/agentic-sdd-router/references/human-input.md`
- `skills-files/agentic-sdd-spec/SKILL.md`
- `skills-files/agentic-sdd-plan/SKILL.md`
- `skills-files/agentic-sdd-tasks/SKILL.md`
- `skills-files/agentic-sdd-implement/SKILL.md`
- `skills-files/agentic-sdd-verification-review/SKILL.md`
- `skills-files/agentic-sdd-debugging/SKILL.md`
- `skills-files/agentic-sdd-architecture-review/SKILL.md`
- `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`
- `opencode-pack/skills/agentic-sdd-orchestrate/references/context-and-cost.md`
- `opencode-pack/agents/sdd-research-librarian.md`
- `opencode-pack/agents/sdd-lightweight.md`
- `opencode-pack/routing.md`
- `opencode-pack/README.md`
- `internal/appops/routing.go`
- `internal/skillsync/opencode_pack_test.go`
- `~/.config/opencode/opencode.json`
- `~/.config/opencode/agents/sdd-lightweight.md`
- `~/.config/opencode/skills/mattpocock-skills/skills/engineering/research/SKILL.md`
- `docs/subagent-fleet-comparison.md`
- `docs/mcp-default-integration-study.md`

## 8. What this report does not establish

- It does not verify that Exa, Context7 or CodeGraph are reachable in this
  session; it only records their configuration and permission surfaces.
- It does not authorize installing the opt-in pack, editing agent profiles,
  applying to a real home directory, or spending on external research.
- It does not approve any feature, batch or direct fix.
