# Independent review implementation plan

Revision: r1 · 3 November 2026. Input: [spec r2](spec.md).
Draft for one combined human review with [tasks r1](tasks.md).
Approval is recorded only in [spec §Approval](spec.md#approval).

## Approach and inspected baseline

Implement Appendix A6 of the [adoption plan](../../docs/Agentic-SDD-vNext-Adoption-Plan.md)
as one optional interactive review capability. Keep the existing main-agent
completion-review owner, result vocabulary and finding dispositions. Do not add
an orchestrator, new skill ID, CLI command, receipt schema or automatic delegation.

The following current sources establish the concrete delta:

- `skills-files/agentic-sdd-verification-review/SKILL.md` and router reference
  `review-resolution.md` already own review scopes, actual dirty changes,
  self-review labelling and finding resolution. Add one conditional independent
  review reference and concise entry pointers rather than duplicating them.
- `integrations/opencode/opencode.jsonc` defines three **primary** profiles and
  denies `subagent:*`. `policy-lockdown.jsonc` independently denies children.
  These defaults remain unchanged.
- `internal/opencodeadapter/adapter.go` generates a provider-denying 2.2.5
  candidate, requires the child denial and rejects non-primary profiles.
  Therefore the independent-review example is **not** passed through that
  generator, preflight or config transaction. Do not loosen their validators.
- `internal/skillsync/package_test.go` checks the exact 18-ID inventory,
  embedded/on-disk parity, sibling links and isolated-home install/no-op/restore.
  A new reference is included by `skillsfiles.go` without adding another skill.
- `tests/harness/native_opencode.py` and `native-opencode.sh` provide separate
  credential-free, network-disabled API probes. Reuse their isolation pattern,
  not their historical result as proof of this new route. Their safe endpoint
  allowlist must not be extended to send prompts.
- `adapterfiles.go` embeds an explicit list of inactive resources. Keep that list
  and the binary's current candidate interface unchanged; distribute the optional
  example in the repository alongside its runbook. This release makes no promise
  that an installed binary alone can emit the new example.

## Decisions and alternatives

1. **Portable method:** add
   `skills-files/agentic-sdd-router/references/independent-review.md`; link it
   conditionally from the verification-review skill and `review-resolution.md`.
   The router keeps routing review to its existing owner. Do not auto-load the
   method or invoke children on ordinary review/implementation requests.
2. **Native route:** foreground `subagent` invocation of one named profile,
   `sdd-independent-reviewer`. Use a separate inactive JSONC example,
   `integrations/opencode/independent-review.jsonc`, with an opt-in read-only
   primary `sdd-review-coordinator`. No direct child slash command: the parent
   prepares context and reconciles the report. No background or reused child
   context for a new review; a scoped re-review is a newly authorized invocation.
3. **Runtime target:** OpenCode **2.2.5**, matching the supplied baseline. Current
   V2 docs describe the route, but compatibility is not certified. Pin binary
   digest and source in B2. If required features are absent, stop the affected
   route; a version change requires reconciled documents and a scoped decision.
4. **Model proposal, not permission:** parent and child explicitly bind to
   `opencode/big-pickle`, variant `default`, because that exact ID appears in the
   user-provided 005 cycle. This is not current price, availability, tool or
   privacy evidence. Keep providers denied in the shipped example. Actual route
   verification and bounded-use authorization are mandatory before inference.
   No substitute model or paid alternative is selected.
5. **Read boundary:** prefer explicit approved regular-file reads. Deny `glob`
   and `grep`: their permission resources are patterns/queries, not filesystem
   boundaries. The parent supplies the file manifest and diff context. This
   satisfies focused inspection without unrestricted discovery or child shell.
   External evidence must be a reviewed sanitized input with narrowly tested
   path access; never grant all external directories.
6. **Adoption:** a user-curated, whole-file optional configuration example and
   explicit manual preview/backup/digest-checked restoration instructions. This
   example must not silently merge into or replace the default candidate or
   real-home settings. The pilot uses a disposable project and fresh runtime
   state, not the primary service or auth database.

Why not the existing ACP client? Its blocked admission and child-metadata handling
were not designed to prove this interactive parent-child route. Reusing it would
expand scope and risk weakening controls. Why not a new skill or plugin? Existing
review ownership and native configuration cover the intended design; no measured
gap justifies another execution engine or installation surface.

## Shared contracts

### C1 — Review input and identity

The parent supplies: explicit request and allowed effects; selected scope/base;
candidate revision and scoped dirty identity; actual staged, unstaged, untracked,
deleted and mode-changed content where relevant; requirement/contract pointers;
exact readable-file manifest; supplied check result identities, commands and
freshness limits; retained finding IDs; runtime/profile/model/variant; effort
limits and next permitted action. Direct reviews without feature records use
the actual request, not invented spec/task IDs.

Use the existing `harnessevidence` scoped identity when adequate. Content or
metadata not available is labelled missing, truncated or caller-supplied, not
filled with plausible values. Unsupported/symlink/special-file inputs are refused
or explicitly excluded with the affected conclusion blocked. Hashing is not
permission to expose private content. No full implementation conversation,
credential store, answer key or hidden reasoning is supplied.

In review-only mode the input is passed through invocation and existing permitted
readable material, not written into the reviewed checkout. An external export
needs actual export authority; otherwise return the limitation. Do not modify
the source to manufacture a complete review. Parent observation before/after
establishes review freshness; concurrent relevant edits invalidate affected
conclusions. This is a procedure contract, not a new serialized evidence schema.

### C2 — Effective profile and invocation boundary

The optional example has a complete deny-first permissions list for **each**
profile. The coordinator may use approved local reads and the required existing
SDD review skill, and launch only `sdd-independent-reviewer`. The child may use
approved local reads and `agentic-sdd-verification-review` only; other skills and
capabilities default to deny. Neither profile may edit, shell, browse, access
MCP/resources, Code Mode, remote tools or initiate other effects. The child cannot
launch any child; unknown/custom tools are denied. Disable snapshots, formatting,
warming and automatic/background compaction in the example where supported;
record unavoidable native maintenance requests rather than assuming zero.

Policy configuration must be consistent with the one coordinator-to-reviewer
edge, while the child's own permissions still deny all delegation. Existing
global/managed child denials remain authoritative: report a blocked route, do not
change higher-authority controls. Preserve the original default templates and
existing primary profiles. Explicit model/variant binding is required on both
profiles, not inheritance from a parent session. Inspect effective sources and
tool inventory, including added instructions; stop on collisions or unexpected
tools. Reject unresolved template markers before adoption.

The permission list is not an OS sandbox or a dynamic one-review lock. One active
reviewer and explicit-request scheduling remain workflow rules, evaluated in the
pilot. RHEL pilot isolation supplies a separate disposable filesystem/process/
network boundary. Read access to selected files must work, while forbidden reads,
symlink escape, edits through every advertised route and nested delegation must
fail in the claimed configuration. If precise native path enforcement cannot be
demonstrated, use an explicitly reviewed sanitized read-only export or leave the
affected mode unsupported; do not claim enforcement from path-pattern inspection.

### C3 — Findings and reconciliation

Child output contains review identity, inspected scope, evidence availability,
requirements and engineering conclusions, stable finding IDs, severity/impact,
locations, rationale, evidence and remaining uncertainty. Existing
`Ready`/`Needs changes`/`Blocked` meanings apply. Execution completion is separate
from verdict; missing terminal output, limits, cancellation or provider failure
cannot yield Ready. No schema parser or automated Ready setter is introduced.

The parent retains IDs and assigns `valid`, `disputed` or `needs evidence`, with
rationale and proposed resolution. Reuse `review-resolution.md`; do not duplicate
its full procedure. Report one guided summary with meaningful code/test pointers,
limits and next decision. During authorized implementation only the parent may
fix an in-scope defect and update canonical state. Review-only remains non-mutating.
Scoped re-review consumes new authorization and remaining bounds; no automatic
repair loop or second unchanged completion review. Changed relevant inputs make
affected findings/verdict stale, not silently current.

### C4 — Proposed pilot bounds and stop contract

For the single user-authorized exploratory subscription observation described in
[spec §Pilot prerequisites](spec.md#pilot-prerequisites-and-retained-findings),
the pre-send accounting, enforcement and bounded-cancellation launch gates below
are excepted, not demonstrated. The 20-minute deadline and other explicit user
constraints apply. Do not apply this exception to other runs or to acceptance.

These numerical ceilings are a **proposal**, not an inference allocation or
claim that the native route enforces them:

| Bound | Proposed maximum |
| --- | --- |
| Top-level observations | 3: seeded case, clean case, cancellation case |
| Active main/child sessions | 1 parent, at most 1 child; nesting zero |
| Model steps per observation | Parent 4, child 6; fresh authorization required for more |
| Aggregate model requests | 30 including all parent/child/auxiliary attempts; retries zero |
| Input per request / aggregate | 12,000 / 360,000 tokens, including instructions and tools |
| Output per request / aggregate | 2,000 / 60,000 tokens, including reasoning where applicable |
| Time | Child 120 seconds; observation 180 seconds; pilot 600 seconds total |
| Cleanup | Stop new work immediately; confirm owned activity stopped within a further 30 seconds or mark uncertain/blocked |
| Fixture size | At most 8 reviewed text files, 32 KiB total; no private data |
| Additional paid use | No paid allocation proposed; paid route requires a new bounded decision and verified provider-side cap |

`steps` is a documented interaction bound, not a request/token/billing/deadline
guarantee. New user input can reset it; forbid implicit continuation. Before B4,
demonstrate how all requests and in-flight work are counted and bounded, including
titles, summaries and compaction. Missing pre-send bounds, accounting, eligibility
or enforceable deadline/cancellation means **blocked**, not an exception, silent
estimate or permission to relax the requirement. Do not build a provider proxy or
custom supervisor to force a pilot through this feature.

The operator must identify the owned parent/child IDs and supported cancel path
for this native route. Cancel only owned work, confirm termination and inspect
uncertain side effects; stream closure is not confirmation. If cancellation cannot
be bounded without an excluded change, leave B4 blocked and return the design
conflict. No provider probing, retries, fallback, nested review or historical
attempt reuse. No current cap, allowance, privacy or free-price claim is made.

### C5 — RHEL pilot and evidence

The target is the user's extra RHEL VM, not the 005 container. RHEL version,
architecture, dedicated user, safe workspace, runtime digest, secret reference,
OS isolation and cleanup owner are unconfirmed; do not connect/provision to find
them without permission. A later bounded proposal must name them exactly.

Prepare offline fixture construction, not a large demo: Go/check-relevant seeded
behavior defect plus insufficient test, and a clean counterpart. Keep independent
oracles and answer keys outside every child-readable path; reviewers receive
requirements and actual candidate, not expected finding text. Pilot setup/checks
and before/after worktree/index/HEAD observation belong to authorized observers,
never the reviewer. Preserve existing dirty sentinels. Confirm allowed reads in
addition to attempted forbidden operations; distinguish permission evaluation,
actual tool dispatch, synthetic transport and real-model behavior in the report.

Missing-evidence, changed-candidate, forged instruction, stale result, disputed
advice and no-duplicate scheduling cases use offline fixtures where adequate.
Live attempts must follow C4 and separate authorization. Save sanitized evidence
outside the target checkout, with owner-only storage; use the existing evidence
retention policy and canonical task pointers, no new ledger. Each observation is
pass/fail/blocked/not run. Missed required defects or a breach prevent the
corresponding acceptance claim. Ask for qualitative usefulness feedback without
claiming savings or closing historical 004/005 findings.

## Components and verification assignments

| Component | Planned change / owner |
| --- | --- |
| Portable review reference and existing entry pointers | T01; C1/C3, no new skill ID |
| `integrations/opencode/independent-review.jsonc` (new inactive example) | T02; C2/C4, no active discovery path |
| `tests/harness/` focused new independent-review fixtures | T02/T03; static fixture tests and separate credential-free exact-runtime probe; no prompts in offline tests |
| `internal/skillsync/package_test.go` | T03; retain exact 18 IDs and verify new reference is distributed with existing parity/backup/no-op behavior |
| `integrations/opencode/independent-review.md` (new runbook), integration README | T04; preview/backup/rollback, limitations and pilot procedure |
| `docs/README.md` or existing root navigation | T04 only if the file exists in the published candidate; avoid unrelated doc reorganization |
| `specs/006-independent-review/tasks.md` and required handovers | Lifecycle owner; evidence/continuation pointers, no duplicate approvals |
| Existing ACP admission, candidate CLI, receipts, default templates, CI trust rules | No production changes selected |

Run focused offline fixture checks first. Go test changes require `make test`,
`make vet`, `git diff --check`. Reuse unchanged evidence only for matching inputs;
do not present prose checks as model compliance. Native offline checks run only
in explicitly authorized credential-free/network-disabled isolation. No packages,
images, credentials or services are installed/altered implicitly. The final RHEL
pilot is after development completion and human review, not a test after every
batch. No LLM runs in CI, unit tests or commit hooks.

## Specialists and conditional tools

All work uses one main agent. Available guidance:

- `opencode` is the principal runtime specialist for T02/T04/T05 and was loaded
  during drafting; Context7 and official V2 pages resolve concrete API/config
  uncertainties. No runtime calls are implied by documentation research.
- `agentic-sdd-writing-for-agents` is available and must be loaded for T01 portable
  guidance changes. It complements, not replaces, existing review ownership.
- `golang-pro` is available for T03's Go packaging tests; load only for that work.
- `agentic-sdd-architecture-review` was loaded in the main drafting agent because
  child permissions and external inference are consequential. No reviewer was
  spawned. Its risk assessment informs combined approval, never replaces it.

Local tools: Read/Glob/Grep for targeted repository discovery (CodeGraph reported
no usable index), patch for edits, shell for permitted deterministic checks and
Git. Context7 and official web fetch for version-sensitive facts; Exa only for
additional public primary-source gaps. Browser tools are not needed. Real child
invocation is the feature under test, not permission to delegate this drafting.

## Risks, dependencies and architecture assessment

| ID | Risk / unresolved dependency | Consequence and disposition |
| --- | --- | --- |
| IR-01 | 2.2.5 effective subagent config, context separation and path enforcement unobserved | B2 must record source/digest and offline observations; do not claim support before exact-route evidence. |
| IR-02 | Native step limit does not establish full token/request/accounting/deadline/cancellation bounds | Blocks B4 until C4 can be satisfied; not permission to modify the blocked ACP path. |
| IR-03 | RHEL target, exact model availability/privacy and dedicated credential route unknown | Blocks VM operations/live pilot, not independent offline drafting or B1. |
| IR-04 | Parent report can over-trust child advice or omit dirty code | C1/C3 fixtures, independent oracles, freshness checks and guided user summary. |
| IR-05 | Global denials or config merges defeat the optional route | Detect/reject collision; keep user controls intact and preserve safe defaults. |
| IR-06-01 | Actual 2.2.5 read dispatch follows an allowed symlink outside the project, even though a direct external read is denied | Demonstrated by the rootless, network-isolated scripted VM fixture. Generic checkout canonical-path isolation is unsupported. Keep the native regression failing; use C2's explicitly reviewed sanitized read-only export with symlink/special-file refusal, or leave the dependent activity blocked. |

### B4 exact-runtime development observations — 5 November 2026

The headless location bug is resolved for the scripted route: pinned CLI source
prefers `PWD` over `process.cwd()`. Passing Python `cwd=` alone retained the SSH
home `PWD`. The native regression reproduced `Agent not found` with stale `PWD`
and passed with `PWD` and `cwd` set to the same fixture location.

The new offline dispatch fixture uses one parent and one fresh child, a fresh
HOME/XDG environment, a rootless user/network namespace with loopback only, and
scripted local protocol responses. Actual skill/reference/file reads completed;
direct external reads and prohibited write/edit/patch, shell, execute, web and
nested-agent calls failed. The child context excluded the parent-only marker.
Interrupting active owned work produced durable `interrupted` outcomes for both
parent and child within five seconds in this synthetic-transport test. This is
not upstream provider cancellation, general recovery, live review quality or
complete request-limit certification. The symlink case remains a real failing
assertion (IR-06-01), not a passing negative test. See the
[runbook](../../integrations/opencode/independent-review.md#actual-tool-dispatch-observations--5-november-2026)
and [task continuation](tasks.md#continuation-record) for evidence locations.

**Draft technical assessment:** offline design is ready for combined human review;
native support and live pilot remain **Blocked** on IR-01–IR-03. This is a main-agent
architecture assessment, not independent review or approval. No implementation
gap is relabelled optional; B4 cannot be marked complete merely because B1–B3 pass.
Retain 004 findings by their existing IDs; new observations apply only to this
review route, not general recovery, billing or normal-home certification.

## Pilot control investigation — 5 November 2026

The user's proposed subscription allocation for `openai/gpt-6-luna-fast#high` is
60 total requests, 720,000 input and 120,000 output tokens, per-request ceilings of
24,000 input and 4,000 output, a 1,200-second total deadline, and zero retries.
These are proposed allocations, not demonstrated controls or a B4 launch decision.
They differ materially from C4 and require reconciliation and scoped human review;
the [supervised proposal](supervised-pilot-proposal.md) predates these numbers and
its statement that no numbers were supplied is historical.

### Findings from public sources and existing observations

- **Pinned source:** The reviewed 2.2.5 source identity used by this project is
  [`9bbb8ce18c80f099e0e448775f59d965ead40a21`](https://github.com/anomalyco/opencode/tree/9bbb8ce18c80f099e0e448775f59d965ead40a21).
  Its [runner](https://github.com/anomalyco/opencode/blob/9bbb8ce18c80f099e0e448775f59d965ead40a21/packages/core/src/session/runner/llm.ts)
  loops within a logical step for retry/continuation, can reset the step on new
  input, and can schedule title generation. Its
  [retry policy](https://github.com/anomalyco/opencode/blob/9bbb8ce18c80f099e0e448775f59d965ead40a21/packages/core/src/model-retry.ts)
  has retry scheduling; [title generation](https://github.com/anomalyco/opencode/blob/9bbb8ce18c80f099e0e448775f59d965ead40a21/packages/core/src/session/title.ts)
  can make a separate model call. The source identity is a reviewed source contract,
  not independent proof that the local binary was built from that commit. No
  validated zero-retry or complete pre-send aggregate token/request control was
  identified for the selected native parent/child route.
- **Accounting:** Current [ACP documentation](https://opencode.ai/v2/docs/cli/acp)
  says end-turn usage excludes child sessions. The prior parent streams similarly
  omit child usage. Current [API documentation](https://opencode.ai/v2/docs/api)
  lists session logs, statistics, usage-limit and interrupt endpoints, but neither
  current docs nor the offline permission fixture verifies their 2.2.5 behavior,
  pre-send enforcement or provider reconciliation. Unknown, failed, in-flight,
  title, compaction and child requests cannot be counted as zero.
- **Cancellation:** An earlier timeout reported a child still running at parent
  shutdown; a later process check found it gone. That is not proof of a supported,
  bounded interrupt of both owned sessions or upstream cancellation. No isolated
  active parent/child interruption test was performed in this research session.
- **Prior paid attempts:** The [runbook](../../integrations/opencode/independent-review.md)
  records two completed client probe events, an interrupted probe, failed/no-route
  and subsequent pilot attempts. Parent event totals and reported zero cost are
  partial; child, auxiliary and provider-side totals remain unknown. The earlier
  [004 acceptance](../../harness/acceptance.md) also records a distinct historical
  spend incident and a P1 HTTP 429. These records cannot reconcile invoices,
  remaining allowance or actual in-flight charges. No credential store or provider
  account was accessed to fill the gap.
- **Offline checks:** `sh tests/harness/native-independent-review.sh` passed 6/6;
  this checks permission evaluation in a network-disabled container, not tool
  dispatch, accounting or cancellation. `TMPDIR=/tmp/opencode python3 -B -m unittest
  discover -s tests/harness -p test_offline_transport.py` ran 21 cases: 12 pass,
  9 container-only cases skipped. The [S0 gate](../../tests/harness/S0.md) is a
  loopback-only synthetic transport, counts UTF-8 bytes rather than verified model
  tokens, and is not an online route.

### Smallest candidate API route — proposal, not operational

An independently reviewed candidate is OpenRouter model
`openrouter/xiaomi/mimo-v2.6-flash`, synthetic inputs only, with one upstream
provider pinned and fallback disabled. This is **not** the subscription model and
must not be substituted without an explicit new model decision. The user—not the
agent—would create a dedicated limited evaluation key, set a non-resetting $1
key cap, verify remaining account allowance and provide it via a separately scoped
secret reference. Do not include keys, account records or secret values here.

[OpenRouter's key-limit documentation](https://openrouter.ai/docs/api/reference/limits)
describes per-key limits and an in-flight budget, but the latter applies only to a
subset of accounts; documented checks do not establish an absolute $1 cap under
all concurrent/in-flight conditions. Its [usage documentation](https://openrouter.ai/docs/guides/guides/usage-accounting)
describes response and generation-ID usage; absent terminal responses and delayed
provider accounting remain unknown. A dollar cap is not a request count or token
reservation. [Provider routing](https://openrouter.ai/docs/guides/routing/provider-selection)
supports restricting providers and disabling fallback in API requests, but the
pinned OpenCode adapter's ability to apply these options to every parent, child
and auxiliary request is unverified. Current [model pricing](https://openrouter.ai/xiaomi/mimo-v2.6-flash)
lists multiple upstream prices: for example Xiaomi at $0.14/M input and $0.28/M
output. At those exact rates, the proposed aggregate maxima produce an illustrative
$0.1344 token-only estimate, not a quote, expected bill or hard cap; charges,
cache/reasoning treatment, provider selection and prices need verification at launch.

The minimum missing engineering evidence is an enforceable pre-send reservation
covering all model requests/tokens, retries and auxiliary calls, verified isolated
credential delivery and egress, plus owned parent/child cancellation. The existing
ACP admission is deliberately fail-closed, and C4 expressly excludes building a
proxy or supervisor in 006. If native controls cannot prove these properties, a
separately reviewed feature/design decision is required; this candidate is not yet
a workable authorized route. Do not create a key or send inference during preparation.

## Documentation research

Read on 3 November 2026: official V2 [agents](https://opencode.ai/v2/docs/agents/),
[permissions](https://opencode.ai/v2/docs/permissions/) and
[tools](https://opencode.ai/v2/docs/tools/), plus Context7
`/websites/opencode_ai/v2`. They document fresh child context, own permissions,
explicit model/variant binding, foreground invocation and step-limit semantics.
They also show why glob/grep queries are not path allowlists. Current pages are
design evidence, not a certificate for binary 2.2.5; cancellation and exact
request bounds are unresolved for the selected interactive route. No native
process, provider/account endpoint or inference was used in planning.
