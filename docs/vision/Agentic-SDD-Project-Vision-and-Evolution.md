# Agentic-SDD — Product Vision, Audiences, and Evolution Strategy

> **Purpose:** Define why Agentic-SDD exists, whom it serves, the engineering principles it preserves, and how it should evolve as AI-assisted software engineering changes.\
> **Status:** Living product strategy; **not** a feature specification, implementation approval, or release commitment.\
> **Canonical path:** `docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md`\
> **Established:** November 2026\
> **Applies to:** Public open-source product strategy, contributor decisions, and future roadmap selection.

## 1. Mission

**Agentic-SDD helps people learn to engineer software and deliver it responsibly with AI—without surrendering understanding, verification, or control.**

It is an open-source **engineering learning-and-delivery framework** for people using coding agents. It combines a portable, specification-driven engineering methodology with explicit human decision points, evidence-based verification, proportional processes, and optional assistance for learning, debugging, and review.

The project is designed for a world in which agents can generate and modify code quickly, but the responsibility for requirements, architecture, security, reliability, maintenance, and operational consequences remains real.

Its measure of success is **reliable, understandable engineering progress**, not the volume of generated code, the number of agents, or the length of an autonomous run.

### Vision statement

> Any developer—from a student building their first real system to an experienced engineer maintaining production infrastructure—should be able to use AI assistance in a disciplined way: understand the problem, establish a reviewable scope, create and verify useful changes, learn from mistakes, and choose deliberately what to delegate.

### Short public positioning

**Spec-driven engineering for coding agents, with human control, meaningful verification, and optional learning support.**

## 2. The problem we address

Coding agents remove friction from writing software, but their speed can create new failure modes:

- **Unclear intent becomes plausible code.** An incomplete request may cause an agent to invent requirements rather than identify consequential decisions.
- **Implementation outruns review.** Large or unbounded changes are difficult for a person to understand and approve.
- **A green check can be misunderstood.** Tests, static checks, model reviews, receipts, and production observations prove different things.
- **Users may stop learning.** Delegating unfamiliar work can reduce opportunities to develop debugging intuition, code comprehension, and technical mental models.
- **More agents can increase complexity.** Delegation adds context handoffs, cost, permissions, integration work, and additional review obligations.
- **Workflow ceremony can defeat its purpose.** Unnecessary documents, repeated questions, and duplicate reviews may cost more than the change warrants.
- **Prompt rules are not a sandbox.** Agent-readable instructions can express an intended workflow but cannot alone enforce filesystem, network, spending, or execution boundaries.

Agentic-SDD seeks to make these trade-offs visible and manageable, rather than claiming to eliminate them.

## 3. Who Agentic-SDD serves

The order below reflects **strategic priority**, not proven usage or product-market fit.

| Audience | Primary goal | Product experience to optimize |
|---|---|---|
| **1. Individual developers using coding agents** | Ship understandable, maintainable changes without uncontrolled scope | Fast entry, safe installation, concise scopes, bounded implementation, direct fixes, useful review summaries |
| **2. Interns, undergraduates, self-taught learners, and junior engineers** | Learn professional engineering practice while building real software | Plain-language explanation, actionable questions, optional tutoring, intentional hands-on work, learning through debugging and review |
| **3. Experienced software engineers** | Increase throughput while retaining technical and architectural judgment | Minimal ceremony, risk-based review, meaningful testing, high-quality handoffs, optional delegation |
| **4. Backend, DevOps, Platform, Infrastructure, and SRE engineers** | Manage systems with consequential failure modes | Explicit blast radius, runtime and environment boundaries, recovery and verification evidence, separate authorization for live changes |
| **5. Open-source maintainers and small teams** | Review AI-assisted changes and maintain consistent engineering expectations | Portable procedures, contributor onboarding, transparent release boundaries, evidence that reviewers can inspect |

**A single core methodology should support all these audiences.** Learners do not need a separate mandatory workflow, and experienced users should not be forced into a tutorial. Accessibility is a product property; autonomous orchestration is an optional capability.

### Illustrative user outcomes

**Student or intern:** “I can explain the requirements, inspect the design, diagnose failures, and understand the code I submitted.”

**Solo engineer:** “My agent can implement useful bounded work, and I can review a concise account of what changed, what was checked, and what remains uncertain.”

**Platform/SRE engineer:** “I can use the same discipline for automation and infrastructure code while retaining distinct approval of deployment, destructive effects, and real-world operational actions.”

**Maintainer:** “Contributors and their agents have a clear shared process, without pretending that a prompt alone guarantees compliance.”

## 4. Two outcomes, not one

Agentic-SDD optimizes for **engineering mastery** and **engineering leverage**. These are complementary but sometimes competing objectives.

| Dimension | What success means | Anti-pattern |
|---|---|---|
| **Engineering mastery** | The developer increasingly understands, investigates, changes, reviews, and maintains their systems independently | The agent completes work that the owner cannot explain or diagnose |
| **Engineering leverage** | The developer can deliver more reliable, appropriately scoped work at acceptable time, cost, and risk | Faster code generation produces more rework, review burden, or instability |

Neither outcome should be inferred from the other. Correct generated code is not proof that a student learned. High task throughput is not proof that an implementation is maintainable or safely operable.

### The guiding rule

> **Learning first when developing competence. Delegation first when applying established competence. Verification always.**

This is a product principle, **not** a mandate that the agent must teach on every request.

## 5. Four ways of working

These are **interaction modes**, not four always-running agents or a new mandatory runtime architecture.

| Mode | Appropriate when | Human role | AI role |
|---|---|---|---|
| **Learn** | The developer is acquiring an unfamiliar concept or technique | Investigate, implement key steps, explain decisions | Tutor, provide hints, pose questions, review reasoning |
| **Pair** | The developer knows part of the domain and wants active collaboration | Lead decisions, inspect changes, practice debugging | Suggest or implement small changes, explain trade-offs |
| **Delegate** | Requirements and technology are familiar enough for bounded execution | Own scope, acceptance, review, and important technical judgments | Implement a defined task, run relevant checks, report evidence |
| **Orchestrate** | Work has separable responsibilities, credible verification, and justified overhead | Own architecture, integration, risk, and final decisions | Coordinate optional scoped assistants or subagents under explicit authority |

A developer may use **Delegate** for a familiar Go API while using **Learn** to investigate unfamiliar Kubernetes networking. Modes reflect *task-specific competence and risk*, not a user's permanent seniority label.

The current portable Agentic-SDD core does **not** need to implement these as automated switches. Existing teaching, debugging, implementation, and review skills may support them through explicit user intent.

## 6. Non-negotiable engineering principles

1. **Human authority is explicit.** Understanding a request, recommending an action, approving feature documents, authorizing a batch, allowing model spending, and permitting live infrastructure operations are distinct decisions.
2. **One main agent by default.** Specialist knowledge can be loaded as skills. Separate agents are optional, never a prerequisite for ordinary workflows.
3. **Reviewable scope beats unlimited execution.** A coherent feature may span sessions; an implementation batch stays bounded by dependencies, blast radius, and a person's ability to verify it.
4. **Proportional process.** A small, well-understood direct fix need not create `spec.md`, `plan.md`, and `tasks.md`. Consequential design or operational changes deserve fuller review.
5. **Ask necessary questions, not all possible questions.** Inspect available facts first, reuse settled decisions, and ask humans about consequential intent or choices the repository cannot answer.
6. **Tests and evidence outrank confident narration.** A model's approval, another model's agreement, a static skill check, or a receipt is not automatically proof of correct runtime behavior.
7. **Risk-based review is a human skill.** Encourage users to understand architectural contracts, failure paths, security boundaries, and meaningful changes—not mechanically inspect every trivial line.
8. **Keep learning optional and genuine.** Teach when requested or when the user chooses a learning-oriented interaction; do not silently replace their problem-solving with an answer.
9. **Security requires technical controls.** Prompt-level restrictions are not filesystem isolation, network containment, credential boundaries, or billing enforcement.
10. **Model and client neutrality is an aspiration verified per surface.** Portable instructions do not establish identical compliance across OpenCode, Codex, Claude Code, operating systems, or models.
11. **Prefer simple, reversible architecture.** Protect canonical data and historical records. Avoid duplicate ledgers, mandatory subagent fleets, and framework rewrites without evidence.
12. **Be honest about support.** Supported, experimental, blocked, historically observed, and not-yet-tested are separate statuses.

## 7. Product architecture and ownership boundaries

Agentic-SDD is **not** one monolithic agent engine. It is a set of complementary layers with distinct responsibilities.

| Layer | Responsibility | Long-term boundary |
|---|---|---|
| **Portable engineering skills** | How the agent scopes, plans, implements, debugs, verifies, explains, and reviews | Canonical reusable methodology; works without advanced runtime integration |
| **Feature artifacts** | Durable requirements, design, tasks, approval, and continuation state | `spec.md` approval and `tasks.md` continuation remain the source of authority |
| **Go CLI and distribution** | Preview-first skill installation, safe replacement, backup, restore, release metadata | Focus on correct, reversible file operations rather than becoming an agent runtime |
| **Deterministic validation** | Execute and report specifically supported checks; preserve evidence and trust boundaries | Tests are bounded proof of named behavior, not blanket AI certification |
| **Client-specific integrations** | Optional adapters, configuration examples, and provider-specific behavior | Clearly separated and graduated to support only after runtime validation |
| **Community and learning material** | Quick starts, worked examples, contribution guidance, examples of good review | Help users enter the same workflow without creating a competing approval process |

The existing `cmd/`, `internal/`, `skills-files/`, `specs/`, `harness/`, and `integrations/` separation is broadly appropriate. Future changes should optimize clarity and ownership—not conform to an oversized directory template for appearance alone.

## 8. Core experience we want users to recognize

A well-functioning Agentic-SDD interaction should feel like this:

1. **Understand:** Classify the request before starting paperwork. Inspect relevant facts, and surface material uncertainty without inventing requirements.
2. **Scope:** Describe the intended outcome, exclusions, acceptance evidence, and any decisions the human must make.
3. **Prepare:** For a consequential feature, draft and reconcile `spec.md` → `plan.md` → `tasks.md` for **one combined human approval**. A direct fix remains direct.
4. **Execute:** Work on **one explicitly authorized batch**, with relevant tests and precise change boundaries.
5. **Verify:** Report actual checks, observed results, limitations, and unresolved issues rather than declaring success by confidence.
6. **Review:** Summarize what matters to a human reviewer and stop at the established decision point; do not auto-start the next batch.
7. **Learn, when requested:** Explain difficult decisions, test comprehension, and help the user practice independent diagnosis without changing implementation authority.

The desired interaction minimizes **unnecessary interruption**, not meaningful human control.

## 9. How the project should evolve with AI software engineering

This is a **conditional direction**, not a date-based promise that AI will inevitably replace particular engineering jobs or that all proposed capabilities will be built.

### 9.1 Changes the project should prepare for

**More capable coding agents.** As implementation becomes cheaper and more automated, requirements quality, design constraints, test selection, and reviewable changes become more important. The framework should remain useful even if an agent can produce a complete feature quickly.

**More diverse model economics.** Users will continue to combine free, inexpensive, and powerful models with different context limits, privacy terms, tool reliability, and costs. Product decisions should optimize *successful verified outcomes per total effort and cost*, not assume one best provider.

**Greater use of autonomous workflows.** Some engineering tasks may benefit from longer bounded execution or multiple agents. Agentic-SDD should evaluate those patterns without making them the default or bypassing established human approvals.

**More generated code and more review work.** Better automation should move effort toward early defect detection, reproducible evidence, architecture understanding, and focused risk review—not simply multiply the number of AI reviewers.

**Risk of reduced developer learning.** Students and developing engineers need opportunities to practice reasoning, code comprehension, debugging, and independent change-making. Optional teaching should support active learning, not merely explain completed agent work after the fact.

**Growing security and operational stakes.** Agent access to repositories, credentials, networks, CI, and production infrastructure requires real containment, execution policies, attribution, and audit trails. Instruction documents must never be sold as equivalent to enforced security boundaries.

### 9.2 Long-term aspirations (not current release guarantees)

- **Context-efficient execution:** conditional references, purposeful handoffs, source-of-truth navigation, and measured reduction of irrelevant context.
- **Budget-aware engineering quality:** compare total retries, failure rate, meaningful tests, review time, and cost across model choices; prefer configuration transparency over opaque routing.
- **Optional independent verification:** fresh-context review and adversarial checks where they detect material defects, with certified permissions and actual isolation rather than assumed safeguards.
- **Learning-aware interactions:** user-requested explanation, diagnosis practice, review exercises, and feedback based on demonstrated understanding, without mandatory coursework.
- **Risk-calibrated delegation:** allow more autonomy for verified low-risk scopes only when explicit authority, environment controls, observability, and rollback are in place.
- **Portable engineering contracts:** preserve common methodology while documenting and testing real differences between agent clients and model behaviors.
- **Sustainable OSS collaboration:** accessible onboarding, precise issue reports, contributor tests, clear version boundaries, and reproducible releases.

## 10. Roadmap by evidence gates, not hype or deadlines

These horizons are **decision areas**, not approved development phases. Any actual change needs the project's normal scope, planning, and authorization process.

| Horizon | Primary investment | Evidence needed to advance | Explicitly deferred until justified |
|---|---|---|---|
| **H0 — Trustworthy core** | Clear public scope, repo navigation, proportional interaction, documentation consistency, installer/restore correctness, release tests, verified `0.x` release | Clean candidate, passing relevant checks, accurate support claims, usable beginner-to-professional walkthroughs | Native autonomous orchestration, guaranteed learning/cost outcomes |
| **H1 — Engineering effectiveness** | Reproducible real-task evaluations for incomplete requests, debugging, implementation, verification and review; compare budget-model results | Comparable scenarios with failure/rework/review measurements and transparent test limitations | Extra agents or longer prompts adopted merely because they are available |
| **H2 — Optional depth and learning** | Improve explicit teaching, risk-based human review, context/handoff guidance, possibly certified independent review | Demonstrated benefit for learners or reviewers without losing safety, raising ceremony, or increasing cost without justification | Universal mandatory teaching or review loops |
| **H3 — Conditional autonomy** | Bounded model selection, subagents, or coordinated execution only for appropriate workloads | Proven incremental quality or productivity; complete accounting where claimed; technical containment; explicit user authority and failure recovery | Fully unattended changes to consequential environments |

**Feature 008** is the immediate release-focused effort under **H0**, not a request to implement all four horizons. Open findings from earlier experiments retain their separate owners; they do not disappear because the product strategy advances.

### Decision gate for new investments

Before adding an agent, tool, hook, model router, documentation step, or new directory, ask:

1. **Which observed user problem does this solve?** Is it reproducible, or merely a plausible future problem?
2. **What already solves part of it?** Reuse an existing skill, reference, contract, or test before creating another owner.
3. **What is the smallest useful change?** Prefer a local improvement over a sweeping rewrite.
4. **How will we know it helped?** Define a meaningful baseline, result, and regression check.
5. **What is the total cost?** Include model tokens, retries, human review, maintenance, security exposure, and context complexity.
6. **What new authority or trust boundary does it create?** Require separate approval and enforcement when necessary.
7. **Does it improve delivery, learning, or both?** State the intended outcome rather than assuming one implies the other.

A good roadmap includes **not building** features that fail these gates.

## 11. How success should be assessed

Avoid vanity metrics such as code lines produced, total prompts, agent count, or unqualified green badges.

| Product outcome | Example evaluation questions | Evidence discipline |
|---|---|---|
| **Correctness** | Does implementation match acceptance criteria? Can a real defect be reproduced and prevented? | Tests, traces, negative cases, checked diffs |
| **Scope control** | Did an agent change files or begin work beyond its actual authorization? | Approval and batch records, reviewed change set |
| **Human experience** | How many necessary vs repetitive questions? Can a reviewer tell what remains unresolved? | Scenario review, user feedback, time and friction observations |
| **Review effectiveness** | Are important problems found before acceptance? Does a second review find useful distinct defects? | Confirmed findings, false positives, review effort |
| **Learning effectiveness** | Can a learner explain, diagnose or modify the relevant behavior without the original agent? | Optional comprehension, diagnosis, and independent extension exercises |
| **Model economy** | How much total model usage, rework, and human effort is required per verified outcome? | Comparable recorded runs with known provider/model limitations |
| **Operational safety** | Are destructive, network, credential and live-system effects technically constrained? | Real enforcement observations in supported environments |
| **OSS usability** | Can an unfamiliar user install, understand the support boundary, and finish one representative change? | Fresh-user walkthroughs and actionable issue reports |

Do not publish numeric benchmarks or productivity promises without adequate data. Static checks of skill text and documentation are **not** proof that live models comply.

## 12. Open-source sustainability and public promises

### What we should promise

- Clear intent, documented scope, and candid limitations.
- A usable portable **core** that does not require a premium model or multiple subagents to understand its workflow.
- Safe-by-default, reviewable installation behavior within verified platform constraints.
- Honest evidence and explicit separation of supported and experimental surfaces.
- Accessible examples for learners and experienced maintainers.
- Respect for human decisions, user environments, and nontrivial operational risk.

### What we should not promise without proof

- Zero hallucinations, reliable obedience by every agent/client, or fully autonomous correctness.
- Guaranteed reduction in tokens, cost, review time, or human learning deficits.
- Security isolation based only on prompt permissions or nominal read-only agent roles.
- Universal support for platforms, models, providers, or CI paths never actually tested.
- Completion of experimental roadmap capabilities on specific dates.

### Governance and compatibility

- Keep the canonical skill identities, feature authority, and supported CLI behavior explicit.
- Maintain an auditable boundary between release-supported capabilities and research prototypes.
- Separate *acceptance of a feature*, *technical verdict*, *permission to execute*, and *permission to publish*.
- Use semantic versions consistently. A reliable pre-1.0 release is useful; `v1.0.0` should follow a deliberate public compatibility commitment.
- Contributions and design reviews should explain user value, evidence, migration effects, and security consequences; no change is inherently superior because it adds automation.

## 13. Relationship to other repository documents

This document is the **long-lived product compass**. It must not become a second approval or task ledger.

| Document | Owns | Does **not** own |
|---|---|---|
| `README.md` | Public introduction, quick start, current supported links | Full strategy or historical audit |
| `docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md` (**this document**) | Mission, audiences, principles, long-term conditional evolution | Feature approval, current batch, release go/no-go |
| `AGENTS.md` and canonical skills | Actual repository/agent workflow policy | Public product vision marketing |
| `specs/<feature>/spec.md` | Per-feature scope, acceptance, combined approval | Project-wide future aspirations |
| `specs/<feature>/plan.md` | Technical approach, trade-offs, tool and specialist assignments | New global authority |
| `specs/<feature>/tasks.md` | Implementation batches, status, continuation | Product strategy |
| `harness/` and its reports | Bounded verification contracts, evidence and trust boundaries | General certification of all agent behavior |
| `integrations/` | Client-specific supported/experimental guidance | Replacement for portable methodology |
| Release notes and release-readiness report | Exact version, tested surfaces, known limitations | Aspirational capabilities |

Changes to this strategy should be proposed with rationale and reviewed like any public-facing project decision. They should **not** automatically modify already-approved feature scopes or reopen unrelated implementation batches.

## 14. Research context and reference points

This section explains **why** the strategy emphasizes review, learning, and controlled autonomy. External observations inform the direction; they are not verification of Agentic-SDD's own effectiveness.

- [DORA — State of AI-assisted Software Development 2025](https://dora.dev/research/2025/dora-report/): AI acts as an amplifier of existing engineering practices rather than a substitute for them.
- [DORA — Balancing AI tensions (March 2026)](https://dora.dev/insights/balancing-ai-tensions/): code-generation gains can shift work toward verification; smaller changes and better review practices matter.
- [How AI Impacts Skill Formation (2026)](https://arxiv.org/abs/2601.20245): a bounded randomized study of developers learning an unfamiliar library found weaker understanding under some forms of AI assistance; it does not establish that every AI-assisted workflow harms learning.
- [(Im)Paired Programming (2026)](https://arxiv.org/abs/2607.26375): a small student study motivates explicit attention to comprehension and independent extension, not only task completion.
- [Anthropic — Demystifying evals for AI agents (2026)](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents): code-based, model-based, and human evaluation measure different aspects of agent behavior.
- [OpenAI — Running Codex safely (2026)](https://openai.com/index/running-codex-safely/): approvals and technical sandboxing have different roles in containing agent actions.
- [Anthropic — How we contain Claude across products (2026)](https://www.anthropic.com/engineering/how-we-contain-claude): approval fatigue and technical containment are material to safe autonomous workflows.
- [Go — Organizing a Go module](https://go.dev/doc/modules/layout): favor useful package boundaries and `internal/` rather than template-driven reorganization.
- [Superpowers](https://github.com/obra/superpowers), [GitHub Spec Kit](https://github.com/github/spec-kit), and [Matt Pocock's productivity skills](https://github.com/mattpocock/skills/tree/main/skills/productivity): adjacent approaches to composable skills, structured development, and focused user interaction. These are sources for selective comparison, **not** authorities over Agentic-SDD's approval contract.

---

### Project compass

> **Build a better engineer and better software—not simply a faster code generator.**
>
> Start with clarity. Delegate deliberately. Verify what matters. Keep human authority meaningful. Let advanced automation earn its place through evidence.
