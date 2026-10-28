# Agentic SDD vNext — harness-level workflow upgrade plan
Revision 5 · 28 October 2026  
Status: researched proposal; no project implementation or runtime benchmark performed.

## Decision and scope

Evolve Agentic SDD into a **portable, measurable development harness**, with OpenCode V2 as its first runtime adapter. Skills remain the workflow and knowledge layer. The deliverable now also includes runtime configuration, policy enforcement, model/tool allocation, executable verification, session recovery, isolated execution, headless runs and evaluation.

Revision 4 added harness features to a skills-focused plan. Revision 5 reorganizes the architecture and delivery around the complete lifecycle. It replaces the previous A/H implementation sequence; Appendix A preserves the detailed Superpowers-inspired workflow improvements and source-based file changes.

The attached “AI-Harness-Configuration-Areas-OpenCode-V2-2026-10(1).md” is the coverage checklist, not a directive to enable everything. The original ten skills and two references remain the workflow baseline. Current local OpenCode settings and actual provider access have not been inspected. Findings identify absent or incomplete integration contracts, not proven defects on your machine.

**Constraints retained:** OpenCode 2.x first; up to US$10/month additional API spending; one main agent and one authorized batch by default; combined spec/plan/tasks approval; direct fixes stay lightweight; review-only does not mutate project state; no implicit delegation, provider spending, deployment or next-batch authorization.

**First release outcome:** one feature can be drafted, approved, implemented, deterministically checked, reviewed, interrupted and resumed with traceable evidence and unchanged authorization scope. The same check contract also runs without an LLM.

## 1. Target architecture and ownership

| Layer | Responsibility | Deliverable | Authority |
|---|---|---|---|
| Workflow | Intent, design, task dependencies, review criteria, approval semantics | Existing skills, shared policy, spec/plan/tasks | Canonical requirements and task state |
| Runtime adapter | Entry modes, model selection, effective configuration, context and tool access | OpenCode configuration, commands, agent profiles | Runtime settings within user/system policy |
| Execution boundary | Files/processes/network/credentials, workspace lifecycle and cancellation | Host/container/VM/CI configuration plus native permissions/policies | Actual execution rights |
| Verification | Formatting checks, compiler/typecheck, lint, tests, feature acceptance | Existing project scripts plus a thin check contract | Observed deterministic results |
| Evidence and recovery | Run identity, changed-code identity, results, costs and continuation pointers | Minimal run receipts and sanitized native exports | Observations only; no new approval ledger |
| Evaluation | Compare outcomes across instructions, models and adapters | Small fixed fixtures and comparison report | Evidence for keeping/removing changes |

Use OpenCode's existing agent loop. Do not build a new orchestration engine, universal configuration compiler, dashboard or custom MCP server for the first release.

A run receipt may reference an approval recorded in tasks.md, but cannot create it. Test output establishes what was checked, not what the user authorized. A technical Ready verdict does not imply merge/deploy permission. Tool permission approval and approval of an SDD batch are distinct decisions.

### Proposed run contract

At run entry, resolve: request mode; repository/workspace; feature and batch if applicable; actual authorization reference; allowed side effects; model/profile; tool set; checks; deadline/attempt/spend limits. Record available values, leaving unknowns explicit. Reuse native session IDs and existing task identifiers.

At exit, distinguish completed, incomplete, blocked, cancelled and failed execution, alongside Ready/Needs changes/Blocked technical review where applicable. This is a proposed adapter result contract, not a claim about native OpenCode event names. Include affected files, evidence references, unresolved findings and next permitted action. Direct fixes need no feature documents; review receipts belong outside the target repository under the host's normal logging policy.

## 2. Coverage of every attached area

P0 means foundation; P1 means targeted extension; P2 means situational. Priority does not authorize an action.

| Attached area | Revision 5 decision | Stage |
|---|---|---|
| Configuration architecture | Ownership map, effective-config audit, version pin and reversible installation | R0–R1 |
| Instructions and skills | Lean repository guidance, progressive loading, duplicate/conflict tests | R1 |
| Models/providers | Capability checks, explicit task profiles, bounded fallback and data-egress policy | R1–R3 |
| Permissions/policies | Separate tool decisions, hard restrictions and OS isolation | R1 |
| Verification gates | Executable check contract reused locally and in CI | R1–R2 |
| Context lifecycle | Budget, retained state, evidence freshness and recovery tests | R2 |
| Observability/evaluation | Baseline before tuning; minimal run metrics and fixed fixtures | R0 onward |
| Agents/subagents | Primary profiles first; one optional reviewer, workers later | R3 |
| MCP design | Selective exposure, authentication, timeout and trust contracts | R1–R2 |
| Plugins/hooks | One measured continuity/evidence extension if native behavior is insufficient | R3 |
| Commands/orchestration | Explicit entry modes and one-batch completion/recovery lifecycle | R1–R2 |
| Parallel work/isolation | Single-workspace ownership first; worktree protocol before concurrency | R3 |
| Tool design | Validated inputs, clear effects, bounded output, retries/idempotency | R1–R3 |
| Headless/CI/service | Deterministic CI first; bounded advisory agent job second | R2–R3 |
| Warming/caching | Stable prompts and measured native cache use; warming initially off | R3/P2 |
| Networking/credentials | Correct service environment now; proxies/CAs only if needed | R1/P2 |
| Hygiene/TUI | Output/watcher/snapshot policy; cosmetic extensions deferred | R1/P2 |
| Language intelligence | External lint/typecheck/build; no assumed V2 LSP diagnostics | R1 |

Provider-level compatibility, budgets, privacy and evaluation are part of these contracts. Cross-harness portability is specified in section 9; it is not a promise of identical extension APIs.

## 3. Configuration, instructions and enforcement

### Configuration ownership

Version-control project defaults, reusable profiles, commands, verification scripts and a supported-version record. Keep personal accounts, secrets and machine-specific paths outside Git. Place restrictions that must outrank repository configuration in the appropriate global/system/managed policy scope; do not deploy enterprise management for a personal setup merely because it exists.

Choose one project configuration location consistently. OpenCode merges direct and .opencode configuration differently; policies use broader-authority precedence rather than normal nearest-config precedence. Inspect the effective outcome, not only the checked-in file. Pin a tested exact V2 release and plugin/server revisions; retain a known-good configuration for rollback. [O1, N1]

Preflight must identify effective skill paths/revisions, active commands/agents, models, MCP connections and plugins without exposing credentials. Check startup logs for ignored/invalid settings. Current policy documentation says invalid statements can be dropped with a warning: a syntactically plausible deny is not sufficient evidence that a restriction works. [N1]

### Instructions

Keep AGENTS.md focused on repository structure, authoritative commands, invariants and where the SDD workflow lives. Skills own reusable methods; scoped instructions own package-specific conventions. Avoid copying the same policy into global instructions, profiles, commands and every skill.

V2 combines instruction sources and does not resolve conflicts for you. It recognizes AGENTS.md, not a CLAUDE.md fallback. Nested guidance is discovered during exploration; already-loaded nested edits are not automatically refreshed like ambient instructions. Test a new session after changing those instructions. Check sibling references and duplicate skill precedence. [N2, O11]

### Three distinct controls

| Control | Use | Limit |
|---|---|---|
| Workflow instruction | One authorized batch, scope decisions and meaningful review | Model behavior must be evaluated; prose is not access control |
| Permissions and policies | Agent allow/ask/deny; hard provider/tool/integration restrictions | Only covers managed execution paths and checks actually performed |
| OS/container/VM boundary | Limit filesystem, processes, credentials and outbound effects where required | Must be separately configured and tested |

V2 policies support provider.use, tool.use and integration.use. A tool policy denial overrides normal permissions and saved approvals; policy allow merely removes a policy restriction and does not grant tool permission. Policies can block loading an MCP server, skill or plugin, but do not sandbox executable plugin code. [N1]

Define draft, build/direct-fix and review profiles. Draft may write agreed feature documents but not production code; built-in Plan is not automatically that profile. Review must consider shell, MCP, browser and Code Mode paths, not just edit denial. Start with restrictive observation access, then enable tested read-only routes. A child agent has its own permissions; it does not inherit a safe intersection with the parent's. [O2, O6, O8]

Path permissions cannot distinguish B1 from B2 when both modify the same file. Preserve approval semantics in workflow state and test them. Do not claim a static path allowlist completely enforces task intent.

Treat repository code, retrieved pages and tool results as data, not authority to weaken policy or disclose credentials. Make provider/remote-tool data destinations explicit. For an untrusted repository or test program, choose an appropriate isolated execution environment before running its code. A worktree alone does not supply that boundary.

## 4. Model, tool and context allocation

### Model policy

Use logical Economy, Standard and Deep profiles, initially backed by one default model and at most one alternative. Map each to an available provider/model/variant during implementation; do not guess names or prices.

| Work | Initial allocation | Escalation condition |
|---|---|---|
| Bounded discovery/mechanical edits | Cheapest model that passes tool/contract fixtures | Observed capability failure after context/diagnosis checks |
| Ordinary design, implementation and debugging | Tested standard profile | Material unresolved uncertainty or consequential design |
| High-risk design/review | Explicit deep profile or preauthorized trigger | Return unresolved work when budget is exhausted |

Verify tool calling, structured result support where needed, context/input/output limits, streaming/error behavior, reasoning variants and provider data handling. An OpenAI-compatible endpoint is not proof of identical capabilities. Run a tiny capability smoke test before benchmarking.

Use command-level selection for predictable transitions: command model → command agent model → current session model. Merely changing primary agent does not change the selected session model. Root defaults currently do not retain variants; per-agent request overlays are documented as not yet sent. Keep active request settings in provider/model/variant configuration. [O2–O4]

Separate availability fallback from reasoning escalation. Rate limits do not justify silent expensive routing. Allow fallback only within the agreed provider/model/budget set, otherwise stop with an explicit incomplete/blocked outcome. Test native default and compaction fallback as well. Frequent model switches may alter caching or restore original history from provider-specific compaction. [O3, O7]

### Budget

Retain the US$10/month additional ceiling. Suggested envelopes remain US$6 routine work, US$3 difficult diagnosis/review and US$1 comparisons; these are allocations, not price forecasts. Track aggregate paid model and tool use, including retries, summaries and in-flight requests. Unknown cost is unknown, not zero. Use provider-side spend/credit controls where available; local estimates and step limits are not hard billing guarantees.

Keep warming, speculative background compaction, repeated independent reviews and automatic escalation off initially. Reuse suitable existing access without assuming a subscription includes API credit. No paid benchmark is authorized merely by this plan.

### Tools and MCP

Keep Context7 for version-sensitive library uncertainty, Exa for source discovery/current external evidence, CodeGraph for difficult relationships checked against current source, and browser automation for actual UI behavior. Do not call all four on every task. Prefer existing repository tools/scripts when they answer the question.

For each integration define: purpose/trigger, side effects, permitted modes, data destination, credential source, actual tool IDs, startup/catalog/execution timeout, cancellation behavior, bounded output and fallback. Pin local packages. Configured V2 MCP servers connect automatically; denied access differs from a disabled connection, and a project override replaces the complete same-named server object. [O5]

Start with native Code Mode for relevant integrations; it can process intermediate results without sending them all to the model. Compare direct exposure only for demonstrated compatibility problems. Nested tools enforce their own checks, and custom tools must implement the permissions they claim. MCP resource-read access needs its own rules. [O5, O8]

A proposed custom tool must have a narrow purpose, validated arguments, explicit effect, understandable errors, paging/truncation markers and observable success. For writes, define idempotency or reconciliation before retry. A timeout does not prove that an external operation failed; check the resulting state before repeating it. Introduce no wrapper that merely renames an adequate built-in. This adopts the tool-design/evaluation principles in N9, not a new tool catalogue.

### Context and session memory

Use on-demand skills, source pointers and bounded outputs; preserve full logs when verification needs them. Keep critical requirement/authorization pointers available across compaction. Evidence reuse requires the same relevant code, configuration, environment and check scope.

Resume re-reads canonical task state and current diff, including uncommitted changes. It validates workspace, actual authorization, blockers, evidence freshness, model/tools and next action. A summary cannot invent permission or turn an unfinished check into a pass. Keep native compaction initially; test a cheaper summary model separately. Stable instruction prefixes and measured native caching come before warming. [O3, O7]

Anthropic's long-running-agent research supports incremental work, reproducible environment setup and durable handoffs. Reuse Agentic SDD's current documents instead of importing another progress ledger or its autonomous feature-selection policy. Those findings motivate the pilot; they do not establish gains for your project. [N8]

## 5. Executable verification and evidence

Make verification a runnable repository contract, rather than a paragraph that asks an agent to test.

For each existing check record: command/working directory; affected scope; prerequisites; timeout; whether it writes files/uses network; expected result; and evidence location. Use the project's existing task runner. Add only a thin wrapper where this materially improves reproducibility. No application-specific commands can be chosen until the target repository is inspected.

| Point | Required behavior |
|---|---|
| Before editing | Check environment and relevant baseline; distinguish pre-existing failures |
| During implementation | Focused checks for changed behavior and regressions |
| Batch completion | Required checks plus acceptance coverage and structured review |
| Feature completion | Connected behavior across batches and unresolved findings |
| Pre-merge | Trusted deterministic gate against the final candidate revision |

Do not run every suite after every edit. Reuse valid evidence and invalidate affected checks after changes, merges or relevant environment drift. A formatter may change the diff, so check the post-format state.

**Verified V2 limitation:** accepted LSP configuration currently starts no LSP runtime and supplies no built-in diagnostics. Use actual compiler/typecheck/lint/test commands. Formatters are opt-in and may depend on installed project binaries; formatting success does not imply test success. [N3, N4]

A receipt should include run/session/task IDs, effective harness/config revision, actual model/variant, base revision and relevant diff identity, commands, exit/result, timestamp, evidence links and remaining gaps. Keep native logs when adequate. Do not create a second mutable completion ledger. Do not export secrets or hidden reasoning; collect observable actions and results.

A zero process exit or a positive LLM summary cannot alone set Ready. CI must independently execute the trusted required checks. Review changes to tests/check configuration themselves; an agent must not pass by deleting coverage or weakening the gate. Missing prerequisites, timeouts, cancellation and unavailable evidence remain incomplete/blocked.

For strict read-only review, reuse existing evidence or run permitted read-only observation. Tests that write caches, start services or modify a database require a separately authorized verification context, even if the source checkout is disposable.

## 6. Workspace isolation, interruption and recovery

Default to one implementation owner per workspace. Record existing dirty files; preserve user work. Before parallel writers, define task boundaries, source revision, separate worktrees, shared-resource ownership and the one integration owner.

Worktrees isolate source edits, not databases, ports, secrets, network or external services. Assign distinct test databases/ports/temp directories where necessary. Cross-worktree tasks exchange contract/evidence references, not overlapping uncontrolled edits. A worker result is a claim; integration reruns affected checks on the merged candidate.

OpenCode worktree create/refresh uses configuration/plugins from the project's saved canonical checkout. Removal uses recorded strategy ownership. Do not assume a worker directory's config owns these operations. [N6]

On cancellation or timeout: stop new work, request cancellation of tracked sessions/children, terminate only owned processes, inspect uncertain side effects, preserve partial diffs and record the next safe action. Closing an event stream is not proof that the agent or an external operation stopped. Never automatically replay a non-idempotent action.

Snapshot undo is best effort and scope-limited. It does not reverse commits, database changes, processes or external actions; it can overwrite later edits to affected files. Use Git/backup and explicit reconciliation for important recovery. Never force-remove a dirty worktree as routine cleanup. [N7]

Pilot one independent reviewer only after the single-agent lifecycle works and only when explicitly requested or separately authorized. Start at one optional child with no nested delegation; choose concurrency from dependency and budget evidence. This plan does not authorize spawning agents now.

## 7. Headless execution, services and extensibility

### Automation progression

1. Run deterministic verification in CI without an LLM.
2. Add an optional advisory agent review of an explicit revision/diff.
3. Consider bounded code-changing automation only after policy, isolation, cancellation and scope tests pass.

Use a pinned runtime, explicit project/model/profile, known environment and a fresh run identity. V2 supports standalone runs and newline-delimited JSON; standalone gives a private server, not a security sandbox. The shared service otherwise owns sessions, configuration and tool execution. [N5]

Set a deadline and define how pending permission requests become blocked outcomes or reach an authorized human. Do not solve noninteractive prompts with blanket auto-approval. V2 run --auto approves actions not explicitly denied; it is not the default for this design. Parse structured events with a version-tested adapter and distinguish transport completion from verified task success.

For CI, use least-privilege short-lived credentials where available; do not expose secrets to untrusted contributions or execute their changed scripts with privileged tokens. Keep the deterministic gate trusted. Agent review remains advisory initially and cannot merge/deploy. Runtime config alone cannot enforce a remote branch's merge rules: configure required checks in the repository host if that guarantee is wanted.

Use the CLI first. Add the network client/API for structured session control when needed. Use the embedded SDK only when embedding the host is the actual requirement; current V2 SDK and network client are different surfaces. Test lifecycle cleanup, permissions and cancellation for the chosen transport rather than transplanting ACP cancellation semantics into another API. [N5, N6]

### Hooks and plugins

Prefer native features. Add one optional server plugin only for a measured gap:

| Hook/API | Candidate responsibility | Prohibited shortcut |
|---|---|---|
| ctx.session.hook context | Deduplicated pointer to canonical state/mode | Reinjecting all skills or hiding tools as sole enforcement |
| ctx.session.hook compaction | Include actual batch/authorization/finding references | Replacing native summary with incomplete state |
| ctx.tool.hook execute.after | Observe check successes/failures and correlate evidence | Automatic Ready from exit zero |
| ctx.tool.hook execute.before | Diagnostics around attempted actions | Assuming universal veto/sandbox semantics |
| ctx.session.hook retry | Bound agreed transient retries | Silent paid model/provider escalation |
| Session/events API | Status, owned-child tracking and minimal telemetry | Auto-approval or an endless continuation loop |

Prompt hooks can modify persisted user input: never manufacture approval there. Plugin ordering, duplicate discovery, reload, per-session location, failure visibility and cleanup all need tests. Keep optional plugin failure compatible with manual native workflow. Native policies/isolation must remain independent. Pin packages/SDK, inspect code and permissions, and provide rollback. No custom TUI is needed for initial delivery. [O9]

## 8. Observability and evaluation

Establish the baseline before tuning. Use three small representative tasks: a feature crossing a real contract, a regression fix and a review-only case. Add cheap scripted negative tests for policy/config/recovery paths; do not spend model tokens on every static check.

| Measure | Interpretation |
|---|---|
| Acceptance checks and regressions | Primary verified outcome |
| Scope/permission violations | Release blocker for a claimed boundary |
| Review finding validity and missed defects | Quality of review, assessed against known cases |
| Model/provider/variant; tokens/cache where exposed | Explain context and routing changes |
| Latency, retries, tool failures and human interventions | Locate friction and avoid hiding supervision cost |
| Total known/estimated cost, including failed attempts | Cost of achieving a verified result |
| Cancellation/recovery behavior | Whether partial work resumes safely |

Use native stats/exports first; add minimal correlation only where unavailable. Mark unavailable cache/cost data as unknown. Sanitize stored traces and set a retention limit. Aggregate all paid attempts, including failures, when comparing cost per verified success; zero successes is failure, not zero cost.

Freeze input revision, fixtures, check environment, permissions and model while testing a harness change. Then compare model profiles separately. Repeat critical cases within budget and retain an unseen variant to catch overfitting. Report observations and sample size, not a reliability percentage inferred from a few runs. Tool-design research supports realistic outcomes and trace/error metrics, rather than rewarding a prescribed call sequence. [N9]

**Adoption gate:** keep a change only if boundary/acceptance tests pass and it shows a useful reduction in cost, delay or intervention without losing quality. Remove unnecessary plugins/tools/configuration. Keep the first paid comparison within the proposed US$1 envelope; current prices/access must be checked before execution.

## 9. Portability and operational maintenance

Keep specs, acceptance criteria, skills, check scripts, result meanings and evaluation fixtures portable. Put model selectors, permissions, commands, hooks and service/API details in separate adapters. OpenCode is the only implementation target for this release.

Codex and Claude Code are future adapters, not presumed compatible configs. Each must demonstrate skill discovery, instruction scope, equivalent action boundaries, model availability, headless completion/cancellation and evidence export. Mark unsupported guarantees explicitly; fail the requested mode or offer a clear manual alternative rather than silently weakening it. A common conceptual vocabulary does not imply interchangeable plugin/hook formats. The Context Course is a useful cross-harness learning source, not a substitute for current product docs. [N10]

For your Mac/remote-lab use, distinguish CLI environment from the process actually running the service. Proxy variables need loopback exclusions; managed-service environment differs from standalone process environment. Apply custom CAs only where required; never disable certificate verification as routine repair. Credential rotation and account changes should invalidate assumptions and trigger connection checks. Do not log service environment values that may contain secrets. [N11]

Tune watcher ignores for generated output and tool-result caps without hiding evidence. Treat native session/snapshot stores as potentially sensitive. Notifications and TUI customization are optional convenience work. Warming needs measured net benefit after its requests are counted.

After an OpenCode/model/provider/plugin change, rerun selection, policy, tool, compaction and cancellation smoke tests before adopting it. Maintain a known-good version/config and migration note. Do not silently update pinned dependencies in the middle of a run.

## 10. Deliverables and staged implementation

These are proposed repository locations, not files created by this review. Adapt to the actual tree before implementation. Reuse existing installer backup/manifest behavior; avoid an unrelated CLI rewrite.

| Deliverable | Proposed location | Purpose |
|---|---|---|
| Harness contract and supported versions | harness/README.md | Ownership, modes, results and compatibility |
| OpenCode configuration/profile/commands | integrations/opencode/ | Native adapter, credential-free examples |
| Portable check contract | Existing task runner plus scripts/verify if necessary | Same deterministic checks locally/CI |
| Receipt schema and fixtures | harness/evidence/ and tests/harness/ | Observable results and scenario verification |
| Headless runner | integrations/opencode/run, only when needed | Explicit lifecycle/status/budget mapping |
| CI job | Existing CI configuration | Trusted deterministic gate; optional advisory agent |
| Optional server plugin | integrations/opencode/plugin/ | Only a measured continuity/evidence need |
| Workflow improvements | Existing skill package | Appendix A changes |

Preserve project ownership of code/checks, user ownership of provider/credential/global settings, and adapter ownership of runtime glue. Install project defaults through an explicit opt-in with a preview/backup; do not overwrite unrelated configuration.

| Release batch | Scope | Exit evidence |
|---|---|---|
| R0 — baseline and contract | Inspect exact runtime/effective setup; define task fixtures and check/result contract | Recorded baseline/unknowns; reproducible representative checks |
| R1 — foundation | Config/instructions/policies, primary profiles, model/tool compatibility, verifier; Appendix A handoff/task-contract fixes | Draft/build/review paths work; claimed denials tested; checks runnable without LLM |
| R2 — complete single-agent lifecycle | Implementation/debug/review improvements, receipts, compaction/recovery, deterministic CI | One authorized batch completes or resumes with fresh evidence; no invented approval |
| R3 — targeted extension | Select only demonstrated needs: bounded headless advisory review, optional reviewer, isolated worktree pilot or one plugin | Each selected extension passes its own failure/cost tests |
| R4 — optional optimization/portability | Compare routing/cache/concurrency; consider another adapter | Evidence of useful benefit; no weakened guarantees |

Each release batch is scoped implementation work under the existing approval model. Draft one coherent spec/plan/tasks set for the chosen release, obtain the combined decision, then execute only the authorized batch. Do not require all R3/R4 features to finish R2.

**Minimum initial release:** R0–R2. It includes native configuration, controls, checks, evidence and recovery as well as skill improvements. It does not depend on multi-agent work or a custom plugin.

### Acceptance and failure scenarios

Keep Appendix A's workflow scenarios and add:

- Conflicting config/skill/instruction copies resolve to known effective sources.
- An invalid policy statement is detected; a project allow cannot bypass a higher-authority denial.
- Primary-agent/model selection and unavailable-model fallback follow the declared contract.
- A custom/local model with incompatible tools fails capability checks before implementation.
- Review writes attempted through editor/shell/MCP/browser/Code Mode are blocked on all advertised paths.
- An unavailable MCP, malformed tool arguments or truncated output cannot become successful research.
- Non-idempotent action timeout produces reconciliation, not blind retry.
- Compaction, restart or location move preserves actual authorization and relevant uncommitted state.
- Missing typecheck executable, formatter failure or unavailable LSP diagnostics cannot become a pass.
- Passing batch tests but failing connected feature keeps the feature unready.
- Agent changes a check to suppress a failure: trusted gate/review detects the weakened criterion.
- Headless permission request, deadline or provider error yields blocked/incomplete output, never an unattended hang or false success.
- Cancellation stops owned activity; uncertain external effects remain visible.
- Worktree tasks conflict on shared service/files: stop integration and reverify the final candidate.
- Plugin reload/failure or duplicate registration adds no duplicate state/approval/requests.
- Cost data absent or delayed never implies zero cost or a guaranteed cap.

## 11. Evidence, limits and decisions still needed

Context7 was used for V2 policies, headless lifecycle and worktrees. Exa Full Suite discovered and fetched official sources and independent engineering references. Mixed old/new Context7 snippets were checked against current V2 pages; for example, top-level policies is current while older experimental.policies examples remain in indexed material.

New verified facts driving this revision: policy precedence and non-sandbox limits; instruction discovery/update rules; absent V2 LSP runtime; opt-in formatting; standalone/JSON CLI behavior; embedded SDK ownership; canonical-checkout worktree behavior; snapshot limits; service environment differences. [N1–N7, N11]

Proposed profile names, run receipts, release batches, budget envelopes and isolation/verification contracts are design recommendations. They are not existing project features or measured improvements.

Before R1 implementation, inspect the exact installed V2 release, sanitized effective config, active plugins/MCP tools, available provider/models and target project's real verification commands. Those details choose settings, not the architectural direction. No further decision is needed to complete this plan revision.


## Appendix A — retained workflow engineering changes

These techniques remain part of the upgrade. They now supply the portable workflow layer of the harness, rather than defining the entire implementation scope. Runtime delivery follows section 10.

### A1. Clarify handoffs before adding capabilities

These are instruction ambiguities inferred from the text, not confirmed defects.

#### A. Route by requested action before feature intake

The router's intake section asks it to resolve top-level documents and select/create a feature before the routing table presents direct-fix and review-only exceptions.

**Change:** classify the request first: explicitly limited document request, review-only, direct fix, research, resume, or feature drafting. Then load the documents that mode needs. Only feature intake selects or creates a feature folder.

**Why:** a review or bounded fix should not accidentally inherit the full feature setup procedure.

#### B. Give completion review one owner

The router says implement then verification-review. Implementation itself says to use verification-review and also contains its own completion/state-write instructions.

**Change:** implementation owns the complete authorized batch lifecycle and invokes verification-review once for its current result. The router invokes implementation and consumes its returned result; it does not independently repeat the review. Standalone verification-review remains manually invocable.

A correction changes the code and can justify scoped re-review. That is distinct from repeating the same review because two skills each schedule it.

#### C. Reconcile stored state with the latest authorization

The router's awaiting-review row says to report the state and wait, while other paragraphs permit explicitly authorized continuation.

**Change:** reconcile current user authorization with recorded state before choosing the route. A new explicit B2 authorization can advance past the previous B1 review checkpoint when it clearly covers that continuation. A vague “continue” must be interpreted from context, not treated automatically as authorization for every remaining batch.

#### D. Make the router's research route match the research skill

The router's row describes a fact blocking drafting. Research-spec explicitly supports uncertainty blocking a review, batch or direct fix as well.

**Change:** route a named blocking uncertainty from any supported stage to research, then return to that same stage. Research is conditional; no mandatory Exa/Context7 call.

#### E. Separate technical outcome from workflow position

Verification can return Ready, Needs changes or Blocked, while completion wording commonly writes awaiting human review.

**Change:** retain a batch's state and next action, and record the technical verdict separately when review occurs. A stopped batch can be awaiting the user's decision with a Needs changes verdict; this is not a successfully finished batch. A technical blocker must remain visible. No stale Ready verdict survives relevant changes.

**Boundary:** none of these changes adds approval, invents consent, or alters the one-batch default.

### A2. Core capability: designs and tasks that can be implemented without guessing

#### Ownership

- `agentic-sdd-spec`: observable requirements and material unresolved intent.
- `agentic-sdd-plan`: chosen design, invariants and shared contracts.
- `agentic-sdd-tasks`: implementation/verification ownership, dependency order and readiness reconciliation.
- `agentic-sdd-architecture-review`: deeper scrutiny of consequential choices, conditionally.

Do not run architecture review for every routine feature, and do not merge its risk assessment with ordinary plan completeness.

#### Small additions to the spec

Preserve its existing content. Add a concrete acceptance example where prose allows materially different interpretations. Mark a consequential assumption as unresolved or as the agent's proposal, rather than presenting it as a user decision.

Use stable acceptance labels when several tasks need to reference requirements; for a tiny feature, existing headings or bullet references are sufficient. No mandatory anti-goal, assumption, or risk table when it would be empty.

#### Strengthen design contracts

Where tasks cross a meaningful boundary, specify the contract once in the plan or the repository's existing canonical API/schema source:

- input/output and relevant error semantics;
- invariant that must remain true;
- state transition or side effect;
- important units, versions or concurrency assumptions;
- why the selected approach fits the requirements.

Alternatives remain proportional. Established patterns need a short justification, not a forced architecture debate.

#### Make task entries executable

Keep the existing objective, acceptance result, dependencies, status and checks. Add only what the task needs:

| Addition | Rule |
|---|---|
| Acceptance reference | Identify the requirement(s) covered |
| Contract reference | Link the shared definition; do not copy it into multiple tasks |
| Consumes/produces | Include for real inter-task interfaces, not every single-file task |
| Verification scenario | State input/condition, expected behavior and applicable check |
| Integration responsibility | Identify which task/batch proves the connected behavior |

Determine target files from code discovery. Do not require guessed file paths, invented interfaces or brittle source line numbers. The plan should decide requirements and boundaries, not contain the whole production implementation.

**Illustrative task entry, not a mandatory template:**

> T2 — Reject an overlapping active reservation  
> Covers: AC-2. Depends on: T1 storage contract.  
> Contract: plan §ReservationStore; consumes its atomic claim operation.  
> Outcome: the losing concurrent request receives the specified conflict result.  
> Check: integration scenario uses two independent transactions against the agreed database. Assert exactly one successful claim and the agreed loser behavior.  
> Integration owner: T3 verifies the service/API mapping.  
> Target files: identify from the repository before finalizing the task.

The database, behavior and contract in this example are hypothetical; no SeatGuard spec was inspected.

#### One readiness reconciliation, not a new approval gate

Tasks owns the final reconciliation of all three drafts:

1. Each acceptance criterion has an implementation owner and a suitable verification owner.
2. Shared producers and consumers agree on types, values, errors and units.
3. Dependencies and shared-file changes have a workable order.
4. Tests check required behavior and relevant failure cases.
5. Material assumptions have been resolved or surfaced for the combined human review.

Use the “could a competent builder proceed without consequential guessing?” question as a check, not another mandatory review document. Record only useful gaps/corrections. Do not import Superpowers' table for every possible pair of tasks; inspect actual dependency edges.

At implementation start, refresh affected assumptions against current code. Do not rerun the full planning process when nothing relevant changed.

**Upstream inspiration:** builder-check, writing-plans and preflight. [S1–S3]

### A3. Core capability: meaningful testing and systematic diagnosis

#### Improve test quality as well as test ordering

The previous plan emphasized red–green–refactor but underweighted Superpowers' `writing-good-tests.md`. Its useful contribution is asking which real defect a test would catch.

Add concise guidance to implementation:

- Derive expectations independently of the implementation under test.
- Assert observable behavior, boundary contracts or side effects.
- A regression test should demonstrate the target defect, not merely a setup/import failure.
- Use doubles where appropriate, but ensure they preserve the behavior the test depends on. Interaction assertions are valid when calls/order are part of the contract.
- Do not test that a skill contains particular wording as proof the agent will follow it.
- Do not create low-value tests solely to satisfy the workflow.

Default to test-first where useful for new behavior and bug fixes. Use characterization tests before changing poorly understood legacy behavior. Use schema/policy/rendering or integration validation where those prove the relevant outcome. This selection is an implementation decision within scope, not another universal approval checkpoint.

A mental mutation check can identify weak tests; it is not evidence that automated mutation testing was executed.

#### Add one standalone debugging skill

Add `agentic-sdd-debugging` because diagnosis is a reusable capability with its own trigger and procedure. It is process guidance, not a domain specialist; it should not consume the “one principal specialist” slot.

**Entry contract:** symptom, expected behavior, known environment, current authorization and affected scope. No feature documents are required for a direct-fix investigation.

**Procedure:**
1. Reproduce, or collect enough evidence to characterize the intermittent failure.
2. Separate baseline/environment problems from introduced behavior.
3. Trace the relevant data/control boundary.
4. State one supported hypothesis and a discriminating check.
5. Correct within authorized scope and verify the original failure plus relevant regressions.
6. Return the finding, evidence, remaining uncertainty and next action to the invoking stage.

Read-only review may diagnose through permitted observation; it must not add instrumentation or change files unless separately authorized. Investigation does not authorize a production mitigation, deployment or expanded refactor.

When repeated attempts produce no new evidence, change the investigation: inspect missing context, revisit assumptions or seek help. Do not copy a rule that three failures prove bad architecture, or automatically change models.

**Upstream inspiration:** test-driven-development, writing-good-tests and systematic-debugging. [S4–S6]

### A4. Core capability: review that produces resolved findings

#### Review at the right scope

Retain one batch completion review by default. Use focused task checks during implementation and an additional task review only for a concrete risky boundary or an explicitly selected delegated workflow.

Return two dimensions inside the existing verdict:

- **Requirements:** missing, extra, misunderstood or unverified behavior.
- **Engineering:** correctness, compatibility, error handling, test adequacy and relevant maintainability/operational risk.

Use the existing Ready / Needs changes / Blocked vocabulary. Do not introduce another “Approved” label that can be confused with human approval. Label same-agent review as self-review; adding a second checklist does not make it independent.

#### Resolve findings through a bounded process

For each material finding, record a stable ID, location, impact and evidence. Classify it as valid, disputed or needing more evidence.

- Fix valid in-scope findings when implementation is authorized.
- Reject incorrect advice with a technical explanation and evidence.
- Seek a decision for changes that alter approved scope.
- Keep relevant unresolved issues visible; do not downgrade severity to finish.
- Keep non-blocking suggestions from expanding the batch.

Re-review the addressed findings and the changed code's relevant dependencies, plus new regressions the fix may introduce. Do not repeat the whole review unnecessarily.

Choose a review-effort budget appropriate to the batch. If it is exhausted, return the unresolved findings. The budget limits work; it cannot turn an unresolved correctness problem into success.

#### Important differences from upstream

Superpowers' current fix-loop reference uses fixed rounds, model escalation and “parking” findings. Its final-review reference limits the final fix wave. Borrow scoped re-review and persistent findings, not those rules wholesale.

Also avoid an absolute rule that anything outside the fix diff is non-blocking: a newly discovered critical vulnerability or cross-cutting contract failure can affect readiness even if its source predates the latest patch. Report its impact without silently expanding implementation.

**Upstream inspiration:** task-reviewer, receiving-code-review, fix-loop and handling-reports. [S7–S10]

### A5. Core capability: integrated feature completion

Add a `feature` review scope alongside the existing `batch` and `direct fix` scopes. Plan feature-wide validation as part of the final authorized batch, or perform it on an explicit feature-review request.

Check the feature's aggregate changes, acceptance coverage, interactions between batches, relevant regression evidence and open findings. Include rollout/recovery/observability only when the change warrants them; architecture-review already supplies the domain questions.

Respect review-only mode. Discovering an old-batch issue does not automatically authorize changing that batch; use existing authorization or request a decision for expanded work.

Pass only actual approved drift or useful lessons to drift-retro. Keep current records; do not automatically delete ledgers or create a completion document.

**Completion output:** observed feature outcome, evidence, requirements not yet proven, unresolved findings and the proposed integration action. Ready does not mean merged or deployed.

**Upstream inspiration:** whole-branch final review and finishing-a-development-branch. [S11–S12]

### A6. Optional capability: independent review before delegated implementation

The attached policy already allows delegation when explicitly requested. Do not turn a plan's model/role label into authorization.

After the core workflow works:

1. Add a small independent-review procedure: task/batch scope, requirement/contract references, actual diff including relevant uncommitted changes, evidence and a read-only finding contract.
2. Confirm the available reviewer/harness supports the requested separation. If unavailable, report that; do not call self-review independent.
3. Let the same main agent decide and implement authorized corrections, then obtain scoped re-review where useful.
4. Add worker delegation later only if concrete tasks benefit from separate context.

For workers, distinguish completed, concerns, missing context and blocked outcomes. A worker report is a claim to verify. Child agents cannot authorize further work. Model escalation requires actual available capability and authorization; do not invent tiers or switch providers silently.

Avoid copying whole conversation histories. Do not force a fresh worker per tiny task or parallelize overlapping edits merely to use subagents.

This capability remains in the roadmap; it is not required for the first release.

### A7. Concrete file change map

All paths below are relative to the supplied skill package root.

| File | Planned change |
|---|---|
| `agentic-sdd-router/references/workflow-policy.md` | Clarify lifecycle ownership, technical verdict versus approval, and process guidance versus domain specialists; keep shared rules here |
| `agentic-sdd-router/SKILL.md` | Classify mode before intake; reconcile new authorization; broaden research routing; remove duplicate review scheduling; route feature review |
| `agentic-sdd-spec/SKILL.md` | Add conditional acceptance examples/labels and explicit treatment of material assumptions |
| `agentic-sdd-plan/SKILL.md` | Define shared contracts/invariants once; make unresolved design choices visible |
| `agentic-sdd-tasks/SKILL.md` | Add contract/acceptance references, integration ownership and the concrete readiness check |
| `agentic-sdd-implement/SKILL.md` | Add meaningful test/diagnosis loop; own one completion review; return verdict/state consistently |
| `agentic-sdd-verification-review/SKILL.md` | Add review scopes, two dimensions, finding resolution and scoped re-review |
| `agentic-sdd-research-spec/SKILL.md` | Retain existing procedure; optionally align description with its already broader body |
| `agentic-sdd-architecture-review/SKILL.md` | Retain current conditional role; avoid duplicating the plan readiness check |
| `agentic-sdd-drift-retro/SKILL.md` | Retain current role; receive actual unresolved drift/lessons from completion |
| `agentic-sdd-bootstrap/SKILL.md` | Keep explicit setup; link the harness preflight/check contract when requested, without duplicating runtime configuration |
| `agentic-sdd-router/references/specialists.md` | Retain conditional expertise; point to tool relevance/availability contracts; concrete provider/model configuration belongs to adapters |
| `agentic-sdd-debugging/SKILL.md` | New standalone diagnosis capability with feature/direct-fix/review boundaries |

Add supporting references only when they keep the entry skill concise: one for task/readiness contracts, one for testing, and one for review resolution. Give each technique one owner; do not repeat it in every skill. An optional delegation reference belongs to a later increment.

If a new skill is added, update the package's explicit ten-skill count and other inventories. The attached files do not include packaging code, so no claim is made about which packaging changes are required.

### A8. Workflow behavior scenarios

Use small scenario fixtures alongside each batch, not a new benchmarking product. The assertions below test behavior rather than the presence of phrases in Markdown.

| Scenario | Expected observable behavior |
|---|---|
| Request a direct fix in a repo without specs | No feature documents created or required |
| Resume B1 awaiting review with explicit B2 authorization | Correctly considers the new scoped authorization; does not wait mechanically or run B3 |
| Request spec only | Delivers spec only; no downstream drafting or implementation |
| Two tasks disagree on a shared API/error/unit | Detects and reconciles the mismatch before implementation |
| Acceptance criterion has no test/integration owner | Identifies the gap and assigns suitable verification |
| Test computes expected output using the same implementation | Identifies the tautology and chooses an independent expectation |
| Regression test fails due to missing setup rather than target behavior | Does not claim red–green proof |
| Intermittent failure with insufficient evidence | Gathers discriminating evidence; does not invent certainty |
| Review-only request suggests adding diagnostic logging | Returns the recommendation; makes no edits |
| Reviewer proposes an incorrect change | Checks reality and rejects or qualifies the suggestion with evidence |
| Fix introduces a new dependent failure | Scoped re-review catches the regression |
| Batch checks pass but connected feature fails | Feature review detects the integration defect |
| Router invokes implementation | One completion review on unchanged result, not two |
| Relevant code changes after Ready | Affected evidence and verdict are refreshed |

Run representative scenarios on the existing and proposed instructions with the same model, permissions, fixtures and tools. Repeat critical cases to expose variability. Inspect actual diffs, tool traces and results. Report observations without claiming a reliability percentage from a few trials.

A useful pilot includes one multi-component feature, one bug investigation and one review-only request. Choose a real feature whose decisions are still open; do not invent a large demo project solely to exercise the framework.

### A9. Workflow exclusions

- Repeated per-stage approvals or forced brainstorming questions.
- Mandatory exact file paths before discovery or full implementation code in plans.
- Mandatory fresh agents or task-level reviews for every tiny step.
- Universal red–green ritual for human prose or configuration.
- Deleting good code because it was written before tests.
- Unconditional full-suite reruns when evidence remains valid.
- Automatic five-round repair loops or model escalation.
- Treating deferred critical findings as successful completion.
- Automatic continuation through all batches.
- A second authoritative ledger or automatic deletion of review history.
- CLI/plugin changes without a demonstrated delivery need.

These exclusions preserve your existing economy of documents and human control while adopting the engineering mechanisms that add capability.


## Appendix B — baseline evidence

I read all **12 supplied file bodies: ten skills and two shared references**. The attachment is the authoritative Agentic SDD baseline for this plan, ahead of earlier summaries or the current GitHub branch.

Attachment: `Pasted text(20261027-041158).txt`  
SHA-256: `1e73d301bc6ac39145d9fb276fe185fb8ecd2b2a1c6bdf652573b76fd6cb07dd`

Locations below refer to line numbers in that combined attachment. They identify evidence, not observed runtime failures.

| Source | Attachment body begins | What already exists |
|---|---:|---|
| router/references/workflow-policy.md | 5 | Canonical sources, combined approval, batch authorization, direct fixes, evidence reuse, continuation |
| router/references/specialists.md | 83 | Conditional expertise/tools, availability checks, no automatic installation/model changes |
| router/SKILL.md | 133 | Feature selection, state-based routing, direct fixes, review-only path |
| verification-review/SKILL.md | 182 | Acceptance and quality checks, current-state evidence, technical verdict, read-only reviews |
| tasks/SKILL.md | 221 | Independently verifiable tasks, dependencies, batches, reconciliation, continuation |
| bootstrap/SKILL.md | 262 | Explicit, minimal, repeatable project setup |
| research-spec/SKILL.md | 290 | Named uncertainty, evidence threshold, versioned sources, return to interrupted stage |
| drift-retro/SKILL.md | 317 | Canonical reconciliation, useful lessons, conditional release preparation |
| spec/SKILL.md | 346 | Outcomes, exclusions, observable acceptance, relevant edge cases, material questions |
| architecture-review/SKILL.md | 377 | Compatibility, ownership, trust boundaries, failures, tests, rollout and recovery |
| plan/SKILL.md | 404 | Design/contracts, proportional alternatives and risks, specialist assignments |
| implement/SKILL.md | 433 | One authorized batch/direct fix, focused checks, diagnosis, completion review |

Paths in this table omit the common `agentic-sdd-` prefix for readability. Appendix A's change map uses full names.

Exa was used to inspect Superpowers source and supporting references; Context7 was queried for planning and review workflows. Context7 also retrieves historical design documents, so current source files control interpretation. No model behavior, runtime bug or performance improvement was measured in this review.

## References

The attached snapshot is the primary source for Agentic SDD findings. Superpowers URLs are moving branch references; pin the actual upstream revision and retain applicable notices when adapting material.

- S1: [Builder check](https://github.com/obra/superpowers/blob/main/skills/brainstorming/builder-check-prompt.md)
- S2: [Writing plans](https://github.com/obra/superpowers/blob/main/skills/writing-plans/SKILL.md)
- S3: [Pre-flight scan](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/references/preflight.md)
- S4: [Test-driven development](https://github.com/obra/superpowers/blob/main/skills/test-driven-development/SKILL.md)
- S5: [Writing good tests](https://github.com/obra/superpowers/blob/main/skills/test-driven-development/writing-good-tests.md)
- S6: [Systematic debugging](https://github.com/obra/superpowers/blob/main/skills/systematic-debugging/SKILL.md)
- S7: [Task reviewer](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/task-reviewer-prompt.md)
- S8: [Receiving code review](https://github.com/obra/superpowers/blob/main/skills/receiving-code-review/SKILL.md)
- S9: [Fix loop](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/references/fix-loop.md)
- S10: [Handling reports](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/references/handling-reports.md)
- S11: [Final review](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/references/final-review.md)
- S12: [Finishing a development branch](https://github.com/obra/superpowers/blob/main/skills/finishing-a-development-branch/SKILL.md)
- [Subagent-driven development](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/SKILL.md)
- [Resuming from a ledger](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/references/resuming.md)
- [Scoped re-review](https://github.com/obra/superpowers/blob/main/skills/subagent-driven-development/re-review-prompt.md)

### OpenCode references

Context7 supplied the initial lookup. Official V2 pages and current documentation source were read to resolve mixed V1/V2 snippets. Initial research: 27 October 2026. Revision 5 adds the sources below, checked 28 October 2026. Verify all runtime-dependent behavior against the installed release.

- O1: [V2 configuration](https://opencode.ai/v2/docs/config/)
- O2: [V2 agents](https://opencode.ai/v2/docs/agents/)
- O3: [V2 models, variants and caching](https://opencode.ai/v2/docs/models/)
- O4: [V2 commands/model precedence](https://opencode.ai/v2/docs/commands/)
- O5: [V2 MCP servers/resources](https://opencode.ai/v2/docs/mcp-servers/)
- O6: [V2 permissions](https://opencode.ai/v2/docs/permissions/)
- O7: [V2 compaction](https://opencode.ai/v2/docs/compaction/)
- O8: [V2 tools](https://opencode.ai/v2/docs/tools/)
- O9: [Plugin loading](https://opencode.ai/v2/docs/plugins/), [plugin overview source](https://github.com/anomalyco/opencode/blob/dev/services/www/src/docs/content/build/plugins/index.mdx), and [server hooks/API source](https://github.com/anomalyco/opencode/blob/dev/services/www/src/docs/content/build/plugins/server.mdx)
- O10: [Agentic SDD installer source inspected](https://github.com/nawodyaishan/agentic-sdd/blob/main/internal/skillsync/sync.go)
- O11: [V2 skill discovery](https://opencode.ai/v2/docs/skills/)


### Sources added for Revision 5

- N1: [OpenCode V2 policies](https://opencode.ai/v2/docs/policies/) — precedence, hard denials, invalid statements and plugin limits.
- N2: [V2 instructions](https://opencode.ai/v2/docs/instructions/) — discovery, scope and update behavior.
- N3: [V2 LSP status](https://opencode.ai/v2/docs/lsp/) — accepted configuration without a running integration.
- N4: [V2 formatters](https://opencode.ai/v2/docs/formatters/) — opt-in execution and failure behavior.
- N5: [V2 CLI commands](https://opencode.ai/v2/docs/cli/commands/) — standalone/JSON runs, auto approval, statistics and service controls.
- N6: [V2 embedded SDK](https://opencode.ai/v2/docs/build/sdk/) — host lifecycle, events and worktree configuration ownership.
- N7: [V2 snapshots](https://opencode.ai/v2/docs/snapshots/) — capture/restore limitations.
- N8: [Anthropic: Effective harnesses for long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) — incremental work and recoverable context; 26 November 2025.
- N9: [Anthropic: Writing effective tools for agents](https://www.anthropic.com/engineering/writing-tools-for-agents) — narrow tools, realistic evaluation and trace metrics; 11 September 2025.
- N10: [Hugging Face Context Course](https://huggingface.co/learn/context-course/en/unit0/introduction) — conceptual cross-harness coverage.
- N11: [V2 network configuration](https://opencode.ai/v2/docs/network/) — proxies, CAs and service environment.

The new attachment is a checklist input, not implementation evidence. Only this advisory plan was updated. The supplied documents, skills, project repository and runtime configuration were not changed.
