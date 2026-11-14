# Feature 020 — Audiences and documentation journeys

**Status:** supporting analysis for draft r2, 2026-11-14; not measured product-market fit.
**Authority:** [vision](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md) §3 explicitly orders strategic audiences, not proven usage; §§6 and 11 reject unmeasured neutrality, cost and learning claims.

## Evidence and prioritization

**Observed product fit:** plain skill files, four installer destinations, optional OpenCode profiles, safe preview and proportional direct-fix/feature procedures exist in source. Optional teaching, debugging and human-review guidance exist as instructions. They do not establish runtime compliance or educational benefit.

**Hypotheses:** audience problems, adoption motives, discovery channels and success targets below are research-informed design hypotheses, not interviews, analytics or observed conversion. Priority considers capability fit, onboarding accessibility and support burden, not audience size. Keep learners in the first tier as the vision requires; do not bury them behind an advanced configuration manual.

- **P1:** solo users and learners; small safe examples, little setup, one-main-agent path.
- **P2:** experienced/OpenCode/backend users; optional depth and accurate references.
- **P3:** maintainers, small teams, infrastructure/custom-workflow users; reuse core guidance with stronger risk/recovery notes. No team administration or production-policy enforcement promise.

## Audience-to-documentation matrix

| Audience / priority | Problem; prior knowledge | Adoption hypothesis; expected workflow | Onboarding obstacle | Documentation and depth | Discovery hypothesis | Observable success to evaluate |
|---|---|---|---|---|---|---|
| Solo developer using AI / P1 | Scope and review drift; Git, shell and one client | Bounded engineering without ceremony; preview → direct fix → feature when consequential | “SDD” sounds bureaucratic | Overview/first five minutes/direct-fix tutorial; shallow entry, optional intermediate feature tutorial | GitHub, client communities, package search | Completes isolated preview and explains scope/checks/unknowns of one change |
| OpenCode power user / P2 | Many agents/routes but unclear ownership; client/provider familiarity | Optional pack/profile configuration with explicit consent | Mistakes installed machine fleet for shipped pack; provider prefixes | Setup choice, profiles, routing, MCP and fleet; intermediate/deep reference | OpenCode community and skill catalogs | Identifies shipped versus local profiles and understands fallback is manual |
| Codex, Claude Code, Antigravity user / P1–P2 | Wants portable process; knows own harness | Skills-only install → one-agent workflow | Assumes OpenCode-specific profiles/MCP apply to every client | Client/destination matrix and shared tutorials; shallow then per-client caveats | Client documentation/community, portable-skills searches | Finds correct destination and can name what has not been runtime-verified in that client |
| Backend/full-stack engineer / P2 | Public/API/data changes need review; tests, services and Git | Small direct fixes; feature documents for consequential contracts | Believes all changes require three documents | Choosing scope, lifecycle, debugging, evidence; intermediate | Engineering articles and OSS examples | Chooses the correct branch and reviews a bounded feature with meaningful checks |
| Cloud/DevOps/platform/SRE / P3 | Automation can affect real environments; IaC and operational risk | Same methodology for code with separate live authorization | Equates prompt controls or preview with sandbox/policy enforcement | Security, recovery, failure modes, trust diagrams; deep caveats within shared pages | Infrastructure communities, plan/diff analogies | Distinguishes approval, batch execution and live change; avoids real-home/production actions in demo |
| OSS maintainer / P3 | Inconsistent contributor-agent output; PR review experience | Adopt pointers/procedures, review changes and evidence | New workflow could add process overhead and rewrite policy | Contributor setup/testing, skills authoring, ADR/release guidance; intermediate/deep | GitHub, maintainer communities | Can submit/review a focused PR and locate canonical acceptance/evidence |
| Small engineering team / P3 | Shared expectations without new platform; Git/PR practices | Use common feature/handoff conventions | Assumes organization management, CI enforcement or a daemon exists | Shared lifecycle/contributor/handoff sections; intermediate, not a team product manual | Team leads evaluating SDD approaches | Two collaborators agree on scope and next authorized batch; no inferred enforcement |
| University student/undergraduate / P1 | Needs engineering practice, not only answers; basic programming, variable Git knowledge | Guided setup → tiny direct fix → optional teach/debug → first feature | Jargon, prerequisites, unclear safe stopping point | Glossary-in-concepts, screenshot if useful, bounded worked example; beginner with expandable depth | Instructors, student repos, course communities | Explains a change and independently modifies one small requirement; do not claim learning from installation |
| Junior developer/intern / P1 | Cannot defend generated work; some stack knowledge | Pair/learn first, delegate familiar work only | Fear of asking questions, unclear review expectations | Same learner entry plus debugging/verification and review checklist; beginner/intermediate | Mentors and onboarding material | States what changed, what was tested and what remains unknown |
| Advanced custom-workflow builder / P3 | Needs composable instructions and boundaries; models, MCP, profiles | Inspect skills/profile sources → configure fleet → bounded delegation | Assumes adapters/local profiles/experimental runtime are installed defaults | Architecture, catalogs, parser contracts, customization, evidence; deep | Source code, advanced user communities | Customizes supported configuration without changing routing semantics or relying on undocumented agents |

## Three journeys, one content set

1. **Beginner:** introduction → prerequisites/client choice → isolated-home preview → skills-only direct fix. Define spec, plan, task, approval and evidence in one concepts page. Optional teaching links do not become mandatory coursework.
2. **Intermediate:** reusable example → consequential feature → combined approval → one authorized batch → checks → review/resume. Research/debugging tutorials reuse the example and evidence explanation. Avoid a second learner-only methodology.
3. **Advanced:** optional pack → shipped profile catalog → model availability/manual fallback → fleet/MCP configuration → execution/recovery/architecture. Second-client or second-model behavior is a separately observed experiment, never a required onboarding success or parity claim.

At the front door answer what/why/who/how, when a direct fix or an existing process makes Agentic-SDD unnecessary, and what can safely be tried without provider credentials or live writes.

## Comparable-product lessons, not superiority claims

Public primary sources were inspected by the audience research worker on 2026-11-14:

| Reference | Useful documentation pattern | Application / limit |
|---|---|---|
| [OpenCode skills](https://opencode.ai/docs/skills/) | Discovery/location and client-native usage reference | Separate skill availability from runtime/model behavior; upstream details require version-aware recheck |
| [OpenSpec](https://github.com/Fission-AI/OpenSpec) | Change-oriented artifact lifecycle | Explain Agentic-SDD's actual approval/batch contract; do not claim peers lack human control |
| [Superpowers](https://github.com/obra/superpowers) | Worked engineering skills and optional agent workflows | Explain one-main-agent default and explicit delegation without declaring a universal differentiator |
| [BMAD Method](https://github.com/bmad-code-org/BMAD-METHOD) | Progressive method/workflow discovery | Offer a short setup chooser rather than mirror a larger role taxonomy |
| [Spec Kit](https://github.com/github/spec-kit) | Entry points and workflow documentation | Compare approach/scope only, not untested effectiveness or client superiority |
| [Kubernetes declarative configuration](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/) | Task-focused guides and clear operational prerequisites | Good how-to structure; Kubernetes-scale docs infrastructure would be excessive |
| [OpenTofu core workflow](https://opentofu.org/docs/intro/core-workflow/) | Preview/plan/apply concepts and explicit effects | An analogy, not equivalent safety enforcement |
| [Grafana as code](https://grafana.com/docs/grafana/latest/as-code/) | User-goal organization with underlying reference links | Reuse task/reference separation, not a product capability comparison |

No peer star count, marketing self-description or source study proves adoption, cost savings, learning or review effectiveness. A public comparison page should state its review date, link both implementations and avoid exclusive “only tool that…” claims.

## Evaluation plan

T12 uses plan §7's L1 learner, U1 experienced-user and M1 maintainer task cards, fixed20 search queries and keyboard/viewport protocol. Record prerequisites, completion, wrong turns, unclear terms, safety misunderstanding and time with consent and no raw private sessions. L1/U1 must be unfamiliar non-authors; author-only walkthroughs remain separately labeled.

Start with observable targets: users find setup choice, preview without real-home writes and distinguish core from optional orchestration. If participants are unavailable, report `not_run` and “fresh-user onboarding unvalidated”; affected AC-01/02 usability remains partial unless the owner explicitly accepts that limitation. Establish completion/time baselines before improvement percentages. Discovery/adoption hypotheses remain unproven until voluntary feedback or actual evidence supports them; no analytics service is required.
