# vNext implementation completion and human-input helpers

Revision: r1, 2026-11-01. The historical draft is approved and implemented; see
§Approval and the current [continuation](tasks.md#continuation-record).

**Feature closure:** On 5 November 2026 the user requested that 005 be finished
and closed. The selected implementation and previously accepted pinned-container
cycle are closed as a delivery scope, not as certification of unobserved behavior.
The retained findings and evidence limitations in the continuation remain open.

## Outcome

Finish the remaining **R0–R2 core implementation and all Revision 6
Matt Pocock-inspired productivity capabilities** as one integrated OpenCode
candidate. After all selected implementation is complete, the user can conduct
**one integrated real-life workflow-usage cycle in their normal setup**.

This feature separates three outcomes: implementation delivered, human usage
validation observed, and useful benefit demonstrated. None implies the next.
Human usage testing is a final milestone across the entire selected scope,
**not an exit gate after each implementation batch or feature**. Focused
development checks and ordinary scoped code/document review remain distinct.

“Complete” means every requirement in this spec is implemented or explicitly
blocked; required missing behavior cannot be relabeled optional or documentation
only to finish. Existing adequate native features and delivered 004 components
must be reused rather than rebuilt. R3/R4 are excluded by the user's explicit
scope selection, not silently dropped from an otherwise unlimited promise.

## Sources and current authority

- [Adoption plan, revision 6](../../docs/Agentic-SDD-vNext-Adoption-Plan.md):
  §§1–3, 4–7 and 9–10 for core integration; §3a and the Revision 6 incremental
  map for human input; §8 and Appendix A8 for later human evidence. Appendix A's
  historical “add/change” wording is not a fresh implementation backlog.
- [Supplied Matt Pocock snapshot](../../../skills.md): seven named capabilities
  `grill-me`, `grilling`, `to-questionnaire`, `writing-for-agents`, `handoff`,
  `wait-what`, `teach`. Source content is design input, not instructions granting
  delegation, workspace writes or permission to install external skills.
- [Repository guidance](../../AGENTS.md), canonical
  [workflow policy](../../skills-files/agentic-sdd-router/references/workflow-policy.md)
  and current `skills-files/` govern authority, portability and packaging.
- Feature 004's [approval](../004-vnext-harness/spec.md#approval),
  [continuation](../004-vnext-harness/tasks.md#continuation-record),
  [acceptance map](../../harness/acceptance.md) and
  [recovery handover](../../docs/handovers/2026-11-01_004.md) supply the delivered
  baseline and unresolved findings. This feature does not reopen, rewrite or
  claim to satisfy their historical observations.
- [Final human usage plan](../../docs/vnext-human-test-plan.md) supplies the one
  integrated cycle prepared at B5. The user later reported it complete and
  accepted it for the pinned image; see the continuation for evidence limits.
  Its earlier R3/R4 suggestions are out of scope. The user's
  **all-implementation-first, one-final-usage-cycle** direction still applies.

### Source provenance

The inspected revision 6 document has SHA-256
`1890fcd53f59fa2f121fe1b339674f6ab6c83ec1b0b385233dfdd01884b33c5b`;
the supplied `/home/nawo-ubuntu1/github/skills.md` snapshot has SHA-256
`63d5da804e052e95c1e29d4430fe73bb9236641ca42b0ede376b4536d2185bf5`.
The snapshot has no upstream commit identity.

Supplementary files were found and read in the local installed checkout
`/home/nawo-ubuntu1/.config/opencode/skills/mattpocock-skills`, whose reported HEAD
is `4ae2317ddb22b046e26455e4f9c9b3aaaf62212d`. This is supplementary evidence,
not proof that the supplied combined snapshot came from that commit, and not a
runtime dependency to introduce into portable instructions.

| Supplement inspected | SHA-256 of inspected bytes |
| --- | --- |
| `LICENSE` (MIT; copyright 2026 Matt Pocock) | `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5` |
| `skills/productivity/writing-for-agents/SKILL-MECHANICS.md` | `c768e6307c7c10728c401c213f2c4ba71c542127eeb7ad2956aabd15a0fa0059` |
| `skills/productivity/teach/MISSION-FORMAT.md` | `8cacbb3c0644d3ae0ea4965564797099401a6930a23f7cf462918576587f2418` |
| `skills/productivity/teach/RESOURCES-FORMAT.md` | `e9cacf34026e11a8d1c8f9de88abe5bcbf654f4ebdb25cae8c0de0d5f48f44ec` |
| `skills/productivity/teach/LEARNING-RECORD-FORMAT.md` | `701fa34b6748aa89e6c960ffb815257f481a7d77fb2900f9028f7edf3fdd6052` |

Before copying or substantially adapting source, implementation must retain
applicable notices and identify the exact material used. No wholesale dependency
on uninspected upstream files, frontmatter semantics or teaching assets is implied.

## Baseline: what is actually left

The current repository has eleven canonical skills and supporting planning,
testing and review references; a Go skill installer; a deterministic check runner;
inactive OpenCode config/lockdown/model templates; offline receipt/recovery
consumers and fixtures; a trusted CI implementation and inspected push artifact.
The seven productivity capabilities are not packaged as Agentic SDD helpers.
Other globally installed helpers do not make them portable package deliverables.

| Area | Delivered baseline to preserve | Remaining implementation or disposition |
| --- | --- | --- |
| Routing, planning, diagnosis and review | 004's canonical skill changes and eleven-skill package | Add the missing human-input/explanation/learning branches without rewriting working lifecycle rules. Behavioral gaps stay for the final usage cycle. |
| Effective setup and configuration | Inactive templates, collision probes, manual preview/rollback instructions | Complete a usable, version-aware, sanitized preflight/adoption interface and its integration with entry modes. Actual user-home adoption remains opt-in. |
| Model/tool controls | Disabled catalog, safety contracts, narrow lab evidence | Complete explicit configurable model/tool bindings and fail-closed admission reporting; no automatic enrollment, live capability probe or new inference broker. |
| Receipts/native observations | Existing schemas, deterministic results and offline consumers | Connect available supported native observations to the existing portable evidence contract; expose unknown/untrusted observations honestly. |
| Recovery/continuation | Deterministic cancellation, manual native reconciliation guidance | Complete coherent user-invoked entry/resume/handoff behavior using supported native facilities, with owned-activity and uncertain-effect handling. Native recovery itself remains unvalidated. |
| Pending human input | Existing human approval conventions; no delivered adapter pending-input contract | Provide a bounded headless pending/blocked outcome and resumable question summary, without implementing an autonomous decision loop. |
| Packaging and user operation | Embedded/on-disk skill distribution and manual adapter adoption | Ship all new helper capabilities and their references through supported portable packaging; document a complete, reversible OpenCode candidate. |
| CI and evaluation | Trusted gate code, push evidence and local anti-weakening fixtures | Reuse existing implementation; do not start PR/CI experiments or invent a new evaluation system. Preserve the PR-path and benefit evidence gaps. |

Some rows are integration gaps; others are missing observations, not proven code
defects. Planning must identify the concrete delta before assigning production
changes. A missing native observation does not justify building a new plugin,
proxy or orchestrator. If native behavior already supplies a requirement, record
its supported integration instead of adding a redundant wrapper.

## Scope and actors

**Included:** all remaining R0–R2 implementation necessary for the selected
single-agent lifecycle; the seven helper capabilities below; their portable
packaging, supported OpenCode entry surfaces, maintenance guidance and one final
human-usage handover. Existing baseline functionality remains a requirement.

| Actor | Responsibility and limit |
| --- | --- |
| Human user | Owns consequential choices, combined approval, batch authorization, later final-usage launch and any live operation. |
| Main agent | Drafts/implements only authorized work; investigates discoverable facts before asking the human; does not delegate implicitly. |
| Questionnaire recipient | Supplies attributed knowledge/preferences; cannot approve the feature or authorize implementation by answering. |
| Runtime adapter | Integrates supported native mode/session/status behavior; cannot create consent or certify unknown boundaries. |
| Checks and evidence | Establish only their observed scope; do not mutate the approval or continuation ledger. |

### Explicit exclusions

- R3/R4: independent reviewer/worker agents, concurrency/worktree orchestration,
  advisory AI CI jobs, custom continuity plugins, warming/cache optimization,
  automatic routing/escalation and additional runtime adapters. Portable
  contracts remain extensible; these excluded implementations are not hidden
  prerequisites for the final usage cycle.
- A new agent loop, orchestration engine, dashboard, universal config compiler,
  MCP server, paid evaluator, provider proxy or replacement credential store.
- Full CI/PR-path experiments, local containerized real-AI tests, model smoke
  probes/benchmarks, real-life workflow simulations or per-batch/per-feature human
  usage tests during implementation. Existing CI is not disabled by this spec.
- Real-home apply, account/key changes, shared-service restart, remote policy
  administration, publication/release, or running the final human cycle under
  implementation approval alone.
- Treating the old US$10/month advisory ceiling, later $0.05/run/$0.25/month paid
  planning ceilings, or “free” pricing as inference permission. The spent P1
  allocation and five conditional original S1 attempts are not reused here.

## Requirements and observable acceptance

The labels below define the selected implementation boundary. Later plan/tasks
must assign delivery and verification ownership to each. Examples describe
expected behavior; none is a claim that a model or manual test was run.

### AC-01 — Complete core coverage without rebuilding the baseline

Every R0–R2 area in adoption-plan §2 and §10 has a traceable disposition: adequate
existing implementation reused, concrete remaining implementation delivered, or
specific unresolved blocker. This includes configuration/instructions, profiles,
models/tools/MCP, permissions/isolation, checks, context/recovery, evidence,
headless pending input, safe networking/credentials and operational hygiene.
Language checks use actual project tools; unavailable native diagnostics are not
invented. Unknown support stays explicit. A required gap prevents an
implementation-complete claim; deferred behavioral evidence does not by itself
prevent development of independent authorized components.

### AC-02 — Focused clarification and explicit grilling

Provide both the explicit `grill-me` entry and reusable `grilling` capability,
with identical authority rules. Investigate discoverable facts first; ask no more
than three ready, independent material questions per round. Defer dependent
questions, provide rationale and credible alternatives, preserve uncertainty,
reuse settled answers, and stop when consequential choices are resolved or
recorded as blockers. Ordinary clear requests do not trigger mandatory interviews.
Agreement ends clarification; it does not authorize implementation.

**Example:** “Use my existing setup” is an environment choice, not permission to
replace its config. Do not repeatedly ask the same environment question or
dispatch a research subagent without the user's explicit request.

### AC-03 — Recipient-specific questionnaires and attributed answers

Provide `to-questionnaire` behavior when another person holds otherwise
unavailable knowledge. Reuse known recipient/purpose context; establish only
missing recipient role, knowledge gap and intended use. Cover every material
requested gap with a clear question, answer/uncertainty stub and necessary
context; create a shareable artifact only when requested or justified by the
distinct recipient. Drafting never authorizes sending/contacting anyone.

Returned answers preserve source/date, uncertainty, conflicts and the responsible
decision owner. Facts, preferences and suggestions remain distinct. Propose
affected spec/design changes; the user or established requirement owner confirms
consequential adoption before implementation changes.

### AC-04 — Concrete decisions and late-answer reconciliation

At existing checkpoints, summarize requested decision, exact scope/revisions,
evidence limits, unresolved finding IDs, authorized next work and work still
blocked. Distinguish discovery, recommendation agreement, combined approval,
batch authorization, acceptance and live/spending permissions. Use only the
existing approval and task locations; no new universal decision form/ledger.

A late answer identifies affected requirements/contracts/dependencies and returns
only changed consequential scope for decision. Preserve unaffected approvals.
Human acceptance with technical findings keeps those findings visible.

### AC-05 — Instruction maintenance as an integrated capability

Adapt `writing-for-agents` into portable maintenance guidance reachable when
authoring skills, instruction files or agent-consumed references. Keep mandatory
authority boundaries visible, branch-specific references behind precise
conditional pointers, a single owner for each rule, co-located caveats and
observable completion criteria. Preserve explicit prohibitions for unauthorized
effects. Do not turn this into an automatic repository-wide rewrite or claim
token savings from shorter text alone. Skill invocation/frontmatter assumptions
must be supported by the selected adapter rather than copied across harnesses.

### AC-06 — Canonical-first handoff and resume

Provide explicit `handoff` capability integrated with existing session-recovery
requirements. Update canonical state only within current authority, then produce
sanitized pointers to actual approval, continuation, dirty work, evidence,
findings and next permitted action. Follow the consuming repository's handover
location/format; temporary copies may supplement but not replace its canonical
handover. Suggested skills are advisory, not consent or a required auto-load.

A receiver revalidates actual files/authority/freshness before continuing; a
handoff never grants a new batch. In a non-mutating mode, return the proposed
handover in the response rather than writing project state.

### AC-07 — Explanation without side effects

Provide explicit `wait-what` behavior: a short plain-language explanation of
where the work is, why, what exists versus remains, evidence limits and the next
decision. Use existing glossary terms where available, without requiring new
glossaries or claiming formal simplified-English conformance. Explanation does
not change feature state, verdicts, approval, files or batch selection.

### AC-08 — Teaching separated from feature execution

Provide explicit `teach` behavior. Default to a focused explanation/example and
feedback on the chosen learning question. Reuse existing knowledge/context rather
than forcing a discovery interview or lesson artifacts. Teaching does not advance
the feature, create approval or invoke children.

If the user explicitly requests persistent teaching, support a separately chosen
workspace with a concise mission, trusted resources, lessons/references and
learning records only when warranted. Preserve its existing files, create paths
lazily, distinguish evidence of learning from material merely covered, and
confirm mission changes. Browser launch, community contact, assets and file
creation require the actual requested scope; no automatic conversion of the
feature repository into a teaching workspace. Adapt inspected source formats
with attribution; do not require uninspected upstream assets.

### AC-09 — Portable discovery and safe packaging

All seven capabilities are reachable through documented explicit entry points;
shared bodies are reusable on their appropriate triggers without mandatory
loading of every helper. The plan chooses skill/reference/alias packaging and
names that avoid collisions; no user/global `grilling` or other unrelated skill
is overwritten. The package must work without this machine's external Matt
checkout or absolute source paths. Preserve applicable upstream notices.

Embedded and on-disk source distributions include every selected helper and
required reference, with matching discovery and valid sibling links. Inventories
reflect actual contents rather than a guessed fixed new count. Preserve installer
preview-by-default, explicit apply, backups, identical-content no-op, restore,
unrelated-skill preservation and symlink/special-file refusal. Skill installation
does not silently install runtime/provider/global policy configuration.

### AC-10 — Usable inactive OpenCode candidate and reversible adoption

Provide one coherent user-facing path for draft, build/direct-fix, review,
explanation/teaching, handoff and resume, using supported native facilities and
reusing the current skills. Identify exact intended runtime/config source and
unsupported routes, expose sanitized effective setup and ignored/invalid policy
warnings, and distinguish native policy from OS isolation. A prose promise is
not an execution boundary. An unproven review route remains unavailable.

The complete candidate is distributable without activating it. Preview, backup,
intervening-change detection and rollback preserve unrelated configuration;
machine paths and secrets stay outside Git. Exact model/integration bindings,
effects/destinations/timeouts and unknown capabilities are explicit. Disabled
access, missing accounting or unsupported settings produce a blocked operation,
not inference, automatic provider switching or fallback to weaker guarantees.

### AC-11 — Bounded pending-human behavior and native status mapping

In supported headless entry, a missing consequential answer yields a bounded
pending/blocked result with the exact question, rationale/options, affected scope,
canonical references and next permitted action. Persist only sanitized resumable
context in an authorized location, or emit it in non-mutating modes. Do not wait
indefinitely, guess/default an answer, fabricate an approver or use auto-approval.
Receiving an actual answer reconciles scope before any resumption.

Native process completion, task execution outcome and technical verdict are
separate. Provider errors, timeouts, denied tools, missing usage and partial
results cannot become task success because a wrapper exited zero. The chosen
interface must state unsupported mappings honestly; it must not introduce an
autonomous orchestrator or require a custom plugin.

### AC-12 — Integrated evidence, interruption and recovery

Connect supported observable native session/model/location and deterministic
check results to the existing portable evidence contract. Identify content,
configuration/runtime and evidence freshness; unavailable costs/session fields
carry explicit unknowns. Caller-supplied assertions are not certified trusted
native provenance. Exports omit credentials, private environment dumps and hidden
reasoning; review exports never mutate the reviewed checkout.

For supported user-requested interruption/resume paths, preserve partial work,
track only owned activity, distinguish cancellation confirmation from stream
closure, and reconcile uncertain effects before retry. Compaction/restart/move
must consult actual current authority and changed files, not merely trust a
saved receipt. Reuse native controls when adequate; a new exporter/entry utility
is justified only for an identified missing integration, not for its own sake.

### AC-13 — Implementation checks without restarting the test campaign

Use proportional focused deterministic checks, source/link/schema/packaging
inspection and ordinary code review for changed behavior. Reuse unchanged valid
evidence. For Go changes, repository-required local checks still apply. Such
checks are not per-batch human workflow-usage sessions or proof of model behavior.
Do not initiate full CI/PR experiments, containerized real-AI runs or LLM
evaluations; do not disable existing CI or bypass safety/installer checks.

If a repository-required gate conflicts with the user's stop direction, surface
the conflict before executing it; do not silently run it or remove it. Development
check limits and later manual-usage scope must remain distinct in results.

### AC-14 — One final integrated human usage cycle

Before proposing the final usage launch, deliver the **entire selected
implementation**, its usable inactive candidate, user instructions, rollback,
development evidence and outstanding finding list. No intermediate human
workflow-usage session is required to authorize development of an independent
component. This scheduling rule does not excuse known unsafe defects or grant
future batch/live-operation authority.

The final handover covers one integrated cycle in the user's normal setup, or in
the pinned QA container recorded in §Approval when the user selects it:
feature drafting/combined approval, one implementation batch, meaningful review,
chosen interruption/resume, a bug investigation and a review-only case, plus the
new helper branches. The human chooses the actual safe work and launch scope;
not every scenario must fit in one sitting. No model test is run or simulated by
the agent under this implementation feature. Container results apply only to
that image identity and do not certify the normal setup, native admission,
macOS, billing/provenance or benefit. Further fix/retest cycles, if
needed, are user-directed rather than an automatic validation loop.

### AC-15 — Honest completion and carried findings

Record implementation completion independently from **not-yet-run human
validation**, support certification, adoption readiness and measured benefit.
Preparing a final walkthrough does not claim it passed. A skipped CI PR path,
unproven model/tool route, missing hosted token contract or unsupported native
Darwin behavior remains explicit. Do not label a required missing implementation
“deferred validation” merely to reach the final gate.

004's B3-F1–F4, B5-F1, B6-F1/F2, remaining B7-F1, B8-F1/F2 and S0-F1/F2 keep
their identities and historical evidence. This feature may supply concrete
implementation toward them, but cannot close them through wording changes,
offline fixture counts, human acceptance alone or a reconstructed baseline.
No compliance, reliability, cost-saving or useful-benefit percentage is claimed
before appropriate human-observed evidence exists.

## Edge and failure cases to preserve

- A clear direct-fix request uses no feature interview or forced helper cascade.
- A discoverable fact does not become an unnecessary user question; unresolved
  stakeholder knowledge can justify a questionnaire, but not sending it.
- “Yes” to a design preference or “I understand” does not approve a batch.
- A stakeholder contradicts another answer: retain both and identify the owner.
- A late answer changes one contract: pause that dependency, not unrelated work.
- A helper invoked during review returns an explanation/proposal without edits.
- A handoff suggests implementation or further skills: reconcile actual consent.
- An unrecognized native pending-input event, wrapper exit 0 with provider failure,
  missing cost, or stale result stays unknown/incomplete, not successful.
- A runtime/model/plugin change invalidates affected compatibility evidence; it
  cannot silently trigger network probes, upgrades or a new test campaign.
- The complete implementation is not ready: do not begin a partial human usage
  cycle to bypass the user's final-only testing direction.

## Remaining planning decisions and evidence dependencies

The selected product scope is settled: **core R0–R2 plus all seven helper
capabilities, OpenCode only, one final human usage cycle**. No additional scope
interview or separate spec-approval gate is requested by this draft.

The subsequent plan must determine supported exact runtime/native interfaces,
helper names/packaging and the concrete adapter/evidence delta from 004. Those
design choices must be grounded in current source and supported documentation,
not guessed CLI/API syntax. If a core requirement cannot be delivered without
excluded infrastructure or a new permission, identify the conflict for a scoped
human decision; do not widen implementation silently.

The user's future final-usage workspace/task, any provider limits/secret reference
and actual adoption authorization remain decisions **before that later launch**,
not prerequisites for this drafting request. Unavailable hosted token proof
continues to block affected online work; no new budget/allocation is created.

## Approval

- **Container-based final human cycle accepted (2026-11-02):** after being told
  that AC-14 named the user's normal setup and that B5 did not authorize a live
  install, the user chose **“B — Accepted cycle”**. Narrow deviation: the one
  final human cycle may run in the pinned local QA image `agentic-sdd-qa:005`
  instead of the normal setup. Exact identity at acceptance:
  image `sha256:f2a14af58b029207d952e3bd4845819ec634f7286982e49ddfe8b2836c3dd349`;
  base `codercom/enterprise-base@sha256:191967da651ec0a567de173ef8a029080da8cd4992de192b65df54ef189be769`;
  OpenCode `2.2.5` sha256 `8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050`;
  `agentic-sdd` binary sha256 `4745f48c32a9ee905259a0a97eb081fc9c38726eed3269a8aec42bbc78929ca3`
  from source commit `3fa623176a457dd4b9fe01be10023204fc9b6568`; 18 packaged
  skills preinstalled. The cycle remains **user-operated**: the user performs
  any online login/model call; the agent makes no model call, copies no
  credentials and performs no real-home install/adoption, release or CI
  experiment. Results count only for this image; normal-setup behavior, native
  admission, macOS, billing/provenance and benefit remain explicit gaps.
  All other approval terms are unchanged. (The image was later rebuilt and
  re-pinned; see the next entry.)
- **Accepted container image re-pinned (2026-11-03):** the image was rebuilt
  from the same accepted scope; the prior image `sha256:f2a14af5…` is no longer
  present locally. Current accepted identity: image
  `sha256:35ed49cdc081db1f233d238568e7fd2455ede458377703b9eb05d0a30103258b`;
  `agentic-sdd` binary sha256
  `f3c4b51742274df9e04a235eddac0fd31d388521339d9f8c1444c877e2d48974`
  from source commit `ba90f5737c125e4df9a4935adeca01301173f0ab`; base and
  OpenCode `2.2.5` sha256 unchanged from the entry above. Verified offline
  `--check` **14 passed, 0 failed, 0 skipped** and 18 installed skills. This
  supersedes only the image/binary identity; scope, user-operated rule and all
  other approval terms are unchanged.
- **B5 review acceptance (2026-11-02):** user completed the B5 human review
  ("i accept b5") of the delivered verification, reconciliation and final
  human-cycle plan (commits `c043ae8`, `6cd7afc`, `c04177c`; implementation
  identity `650f60d1…0135d`). This accepts the B5 delivery and the prepared plan
  **as prepared**. It does not execute or accept the human usage cycle, certify
  runtime support, publish the B5 commits, close any carried 004 finding or
  authorize a release/tag. Next decisions are B5 publication and the separate
  final-cycle launch.
- **B4 publication and B5 authorization (2026-11-02):** user directed
  **“submit b4 commits, start B5”**. The five B4 commits (`2e2d461` → `1b8886f`
  → `a6e1809` → `0ec7f45` → `af0036e`) were pushed to `origin/main`
  (`949bcf2..af0036e`), with the disclosed deterministic CI running on push;
  no release tag or credential operation is included. The user then authorized
  **B5** (T11 aggregate development verification and T12 completion
  reconciliation/final human-cycle preparation) as offline/portable work only:
  no model calls, native `opencode acp` processes, provider probes, credential
  reads, human workflow-usage run, live install/adoption, release tag, push
  beyond this publication or account/key change. No separate B4 review verdict
  was stated; B5 is authorized directly by this instruction rather than inferred
  from a review, and any B4 defect found during verification is returned for a
  decision rather than silently waived. This entry records authorization only.
- **B3 review acceptance, B4 authorization and B3 publication (2026-11-02):**
  user completed the B3 human review ("Yes — reviewed, accepted") of the
  delivered source/tests/diff (commits `ce84d16` → `9a9c392` → `ae2d9f1` →
  `edf58db`, B3 source identity `8df49be3…` recorded in the 005c handover).
  The user then explicitly authorized **B4** (production evidence consumer
  T09 and native-observation bridge T10) as offline/portable implementation
  only — no model calls, native `opencode acp` processes, provider probes,
  credential reads or real-home adoption; the C4 admission boundary is
  unchanged. The user also separately authorized pushing the B3 commit chain
  to `origin/main` ("Push now"), with the disclosed deterministic CI running
  on push; no release tag, PR experiment or credential operation is included.
  B5 remains unauthorized. Recording only; this entry adds no other scope.
- **B3 authorization (2026-11-02):** user completed the B1/B2 human review and
  authorized **B3** (native ACP transport and pending-human normalization,
  T07–T08). The user's message added "authorize it using opencode … openrouter
  cheap or free models"; this is recorded verbatim as context but does **not**
  change the C4 admission boundary. B3 implementation is offline (synthetic fake
  peers only); native prompt admission stays denied while the strict pre-send
  token proof, credential isolation, complete accounting, provider caps/allowance
  and auxiliary-retry prevention remain unverified. No exact reviewed model id,
  data scope, bounded request/token/deadline limits, credential provisioning or
  accounting evidence was supplied, so no real model call, native `opencode`
  process, provider probe or paid/free inference is authorized by this wording.
- Decision: **approved** (2026-11-02), user in this conversation. Presented the
  combined spec/plan/tasks r1 and B1 scope, the user replied **“Approve B1 B2”**.
  This approves the combined design and explicitly authorizes B1/T01–T04 and
  B2/T05–T06 sequentially, with B1's ordinary completion review before B2; no B3
  or later batch, human usage session, model/CI experiment or live installation.
- Reviewed pre-approval SHA-256: spec
  `ee3bd1805bbd6fe9d94bf0e5b325a47fab4b9065049bda67acd2ab99516723b2`, plan
  `c6505c475a44568e0b576c6a56e3fdc68c33182193f25497f00b6934dcf8c9bc`, tasks
  `7f47cd06558d02d5a19374b9bb99f31dafd1796752f84bd27ef547ad9aa3796d`.
- Publication decision: user selected **“All stable 004/005 work”**, authorizing
  logical commits/push of reviewed stable 004 fixtures/status/adoption/manual
  documents and 005 drafts plus authorized passing implementation. The question
  disclosed existing deterministic CI would run on push. Exclude private/raw
  evidence, local-only notes and unrelated unreviewed work. No release tag,
  separate PR/CI experiment, credentials or live install authorized.
- Drafting authority: user requested the next spec for remaining vNext and Matt
  integrations, then clarified final-only integrated human usage testing.
- Scope input: user selected **“Core plus Matt helpers”** and **“OpenCode only
  (Recommended)”** when the optional R3/R4/runtime boundary was presented. These
  choices define this draft; they are not implementation or live-use approval.
- The original request was spec-only. The user subsequently clarified **“005
  plan and tasks”**, authorizing those two drafts for the same scope. The later
  **“resume, implement commit push stable changes”** expresses requested execution
  and publication, but these newly written plan/tasks have not yet been presented
  for combined approval; return the concrete set and first-batch/publication scope
  before implementing 005. No model/CI experiment, real-home installation,
  credential operation or walkthrough execution follows from either request.

Current documents: [plan r1](plan.md) and [tasks r1](tasks.md). All selected
implementation (T01–T12) is delivered and the B5 review is accepted. The user
chose the pinned `agentic-sdd-qa:005` container as the accepted final-cycle
environment (identity in §Approval); the user subsequently reported the cycle
complete and accepted its results for that image. Scenario-level outcomes remain
unspecified; see the [continuation](tasks.md#continuation-record). Publication
and any further fix/retest remain separate user decisions. Preserve the
single-batch rule and separate development evidence from human usage validation.
