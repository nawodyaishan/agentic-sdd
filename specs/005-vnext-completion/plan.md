# vNext completion implementation plan

Revision: r1 — draft, 2026-11-02. Input: [spec r1](spec.md).
Approval is recorded only in [spec §Approval](spec.md#approval); none yet.
This plan covers **R0–R2 core plus seven Matt-inspired capabilities, OpenCode
only**. R3/R4, autonomous evaluation and live installation remain excluded.

## Delivery approach

1. Preserve 004's source, results, unresolved findings and installer behavior.
   Freeze current source identities before changing the candidate; do not
   reconstruct a behavioral baseline or repeat its test campaign.
2. Complete and package the portable human-input suite first, then the inactive
   runtime/adoption interface, native lifecycle adapter and evidence integration.
3. Use focused deterministic development checks and one scoped completion review
   per authorized batch. **No batch/feature-level human workflow-usage sessions.**
4. Deliver the entire selected implementation and one final human-usage handover.
   The later human launch is a separate decision; preparation is not execution,
   native certification or a useful-benefit claim.

## Source-grounded decisions

- `internal/skillsync/sync.go:discoverSkills` already discovers regular
  `agentic-sdd-*` directories from `fs.FS`; `skillsfiles.go` embeds their contents.
  Extend the existing package, not a second installer or a new client destination.
  `internal/skillsync/package_test.go` currently expects eleven canonical IDs;
  new behavior must update the actual ID inventory and its compatibility tests.
- `integrations/opencode/` contains inactive templates and manual adoption/
  recovery guidance, **not a production adapter entry/export implementation**.
  `tests/harness/receipt_tools.py` is a development reference, not production code.
  These are concrete integration deltas rather than reasons to rebuild checks.
- Keep the existing Go/standard-library CLI architecture. Add a narrow
  `agentic-sdd opencode` command group, not another installed binary or a general
  workflow orchestrator. Existing preview/apply/backups/logs/restore behavior is
  unchanged. This is proposed syntax, not currently available commands.
- Runtime target: exact **OpenCode 2.2.5**, previously inspected binary SHA-256
  `8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050`.
  Do not download/upgrade it or certify all versions above 2.2.0. Different builds
  require their own recorded source/compatibility decision before use.
- The public tag `v2.2.5` resolves to upstream commit
  `9bbb8ce18c80f099e0e448775f59d965ead40a21`. Its
  `packages/cli/src/run/noninteractive.ts`, inspected SHA-256
  `dd60c7aea0fbcfc9ee3b36edb048926ae498e9dce947fbe222377c12a52d4946`,
  cancels unavailable questions with feedback to **continue without an answer**.
  Therefore `opencode run --format json` is not the pending-human control path.
- Select native **ACP protocol 1** for the narrow headless session interface.
  The same commit's `services/www/src/docs/content/cli/acp.mdx`, SHA-256
  `8fbbebf1fb509e802280b8a9e6502069d80f897568b0b76eaac9f9993019dc99`,
  documents a private stdio server, `elicitation.form`/`elicitation/create`,
  `session/cancel`, config options, resume/location rules and usage limitations.
  Build a small client of that contract; OpenCode still owns its agent loop.
  Do not transplant ACP cancellation meanings onto HTTP or CLI transport.
- Current official V2 skills/commands/CLI/API/ACP pages and Context7 were consulted
  for planning on 2026-11-02. Moving documentation and `dev` snippets are not
  observations of the installed binary; pinned source controls conflicting
  details. No native service, provider/account endpoint or model was invoked.

Sources: [pinned ACP document](https://github.com/anomalyco/opencode/blob/9bbb8ce18c80f099e0e448775f59d965ead40a21/services/www/src/docs/content/cli/acp.mdx),
[pinned non-interactive runner](https://github.com/anomalyco/opencode/blob/9bbb8ce18c80f099e0e448775f59d965ead40a21/packages/cli/src/run/noninteractive.ts),
[V2 skills](https://opencode.ai/v2/docs/skills),
[commands](https://opencode.ai/v2/docs/commands),
[CLI](https://opencode.ai/v2/docs/cli/commands),
[ACP](https://opencode.ai/v2/docs/cli/acp),
[API](https://opencode.ai/v2/docs/api).

## Components and ownership

| Component | Decision / proposed location |
| --- | --- |
| Portable helpers | Seven new namespaced directories in `skills-files/`, listed in C1; references stay with their procedural owner. |
| Shared authority and human input | Existing router policy owns authorization; `references/human-input.md` owns common decision/question/late-answer rules. Existing entry skills carry conditional pointers, not copied helper bodies. |
| Adapter CLI | Small dispatch additions in `cmd/agentic-sdd/main.go`; separate `opencode.go` command handlers/tests if needed. Reuse `flag`, standard library, existing build/release path. Update help/README/Makefile where command surfaces change. |
| Adapter implementation | `internal/opencodeadapter/`: source/template resolution, bounded native projection, staged config operations, ACP transport and outcome normalization. Small explicit functions; no general plugin framework. |
| Evidence implementation | `internal/harnessevidence/`: typed v1 receipt validation, scoped identity and safe export, needed by adapter independently of the development Python reference. Reuse `internal/harnesscheck` result types/semantics instead of duplicating the runner. |
| Embedded adapter resources | Root embed file for only shipped inactive JSON templates/catalog and required schema resources. Resolve embedded versus explicit repository sources with `fs.FS`; do not require the checkout or external Matt installation at runtime. |
| Adapter examples/procedures | Existing `integrations/opencode/` remains the home for profiles, supported-version, safety, adoption, recovery and new usage/contracts documentation. No automatically discovered `.opencode` file is installed while developing. |
| Development cases | Existing Go package tests and focused `tests/harness/` source/link/schema consumers. Fake stdio peers and curated synthetic observations only; no native generation or new container/CI campaign. |
| Human final handover | Update `docs/vnext-human-test-plan.md` at final integration to the selected scope and final-only schedule; evidence/finding state remains in feature tasks and 004's original records. |

## Contracts

### C1 — Helper packaging and invocation

| Source capability | Canonical package ID | Trigger / procedural owner |
| --- | --- | --- |
| `grill-me` | `agentic-sdd-grill-me` | Explicit user entry; delegates instructions to the exact grilling ID, not a child agent. |
| `grilling` | `agentic-sdd-grilling` | Explicit challenge or consequential clarification branch; ready-question frontier, at most three questions. |
| `to-questionnaire` | `agentic-sdd-questionnaire` | Explicit request or justified knowledge gap held by a distinct recipient; owns questionnaire structure/answer ingestion. |
| `writing-for-agents` | `agentic-sdd-writing-for-agents` | Authoring/maintaining agent-consumed instructions; owns maintenance reference and completion checklist. |
| `handoff` | `agentic-sdd-handoff` | Explicit transfer or repository-required handover; owns canonical-first transfer steps. |
| `wait-what` | `agentic-sdd-wait-what` | Explicit request to re-explain scope/status/rationale; response only. |
| `teach` | `agentic-sdd-teach` | Explicit learning request; lightweight default, separately requested persistent-workspace branch. |

This choice produces eighteen canonical skill IDs if the current eleven remain
unchanged. Tests compare the full ID set and reference closure, not the number
alone. Short upstream names are documentation labels, not installed overrides.
Use descriptions for reusable branches; explicit-only entries use supported
visibility metadata. OpenCode's `disable-model-invocation` hides advertisement
but does **not** prevent explicit ID loading: instruction checks still enforce
the user's actual request. Do not describe that metadata as an access-control
barrier or copy stronger semantics from another harness.

Every helper loads the shared workflow policy on demand and preserves the
invoking mode. Ordinary requests load no helper cascade. Shared references can
be read directly when a supporting explicit-only entry is not auto-advertised.
All required references/notices ship in the canonical package, including the
Matt MIT notice and adaptation provenance under the router's references.
Reuse the inspected source/formats from spec provenance; inspect/hash any newly
needed upstream material before copying it. No external absolute-path dependency.

### C2 — Decision, questionnaire, handoff and teaching effects

Common input: requested action, known facts, current mode/effects, consequential
unknowns, actual user decisions and canonical references when they exist. Direct
fix/explanation/teaching do not invent feature documents. Output: resolved inputs,
attributed proposed changes or explicit blockers; no new approval field.

- Questions investigate facts first, reuse settled answers, present independent
  ready choices neutrally and stop at resolved material scope or explicit block.
- Questionnaires cover every requested gap, establish recipient/purpose only
  where unknown, preserve conflicting/uncertain answers and require confirmation
  of consequential adoption. Writing a questionnaire never sends it.
- Late answers name affected spec/contract/task edges and preserve unaffected
  approval. Existing tasks own continuation; spec owns approval.
- Handoff follows the consuming repository's path/structure. Without permitted
  writes or a justified canonical location, return the handover in the response;
  do not create a universal new ledger. Suggested skills grant no authority.
- Explanation changes no project state. Teaching defaults to response content;
  persistent missions/resources/learning records/HTML are a separately requested
  branch in a selected workspace, lazily created without overwriting unrelated
  content. Learning evidence is not inferred from mere topic coverage.

### C3 — Preflight, inactive candidate and configuration transaction

Planned CLI operations: `opencode preflight`, `opencode config preview`,
`opencode config apply`, `opencode config rollback`. They are separate from
root skill apply/restore. Preview is the default for configuration work; no
effect is inferred from an approval reference string or a CLI switch alone.

Preflight accepts explicit workspace/runtime/config inputs, emits allowlisted
IDs/versions/source digests, denied/unknown routes and actionable blockers.
Default operation examines supplied files only. Native read-only inventory is
an explicit operation with bounded metadata-only endpoints and location; it must
not start/restart a shared service, scan the primary DB or contact a provider.
Output excludes raw config, credential/header/environment values and transcript
content. Incomplete/truncated catalogs stay unknown, not complete inventory.
Native inventory may load executable extensions even without a prompt: only a
separately selected clean, credential-free state/config root is eligible. Do not
start an unknown normal setup merely to inspect it; accept an operator-curated
sanitized projection with its trust limits instead.

Candidates use the existing project `.opencode/opencode.jsonc` convention but
are emitted **outside active discovery**. Preserve provider/integration denial,
one default/at most one reviewed alternative, exact model/variant references,
current-session prompt-only commands and disabled unproven review routes.
Economy/Standard/Deep bindings are explicit proposals, not enabled model access.
New helper commands have `subagent: false` and no shell interpolation blocks.
Questionnaire/teaching file effects do not inherit broad build grants.
The initial inactive proposal uses the previously selected planning candidate
`openrouter/nvidia/nemotron-3-ultra-550b-a55b:free` consistently across default,
profiles and commands, with providers still denied and alternative unset. This
does not enable it, certify its tools/privacy or transfer an existing allocation.

No generic lossless JSONC merge is introduced. With an existing file, the user
curates a proposed whole file retaining comments/unrelated content; the tool
compares/stages bytes and checks known unsafe/ambiguous settings without
silently rewriting it. Reject duplicate keys, symlinks/special files, ambiguous
parse/unknown compatibility and sensitive credential-bearing inputs for export.
Private source data is never printed on parser error. Do not claim universal
secret detection or a concurrent-writer sandbox.

Runtime-config backup manifest is distinct from skill manifests: format version,
kind, canonical destination, original absence/digest/mode, proposed installed
digest, per-file backup identity and transaction result. It contains no consent
or credentials. Backup every existing destination before the first replacement;
identical bytes cause no rewrite or backup. Recheck parent/file identity and
expected digest before replacement; reject intervening edits. Use private external
storage and same-filesystem atomic replacement. Rollback verifies both backup and
installed digests; remove only a verified originally absent file, never a tree.
Require quiescent trusted storage. Tests use temporary homes, never the real home.

### C4 — Native entry and strict admission

Planned `opencode session` is a narrow ACP client with two distinct paths:
default offline preparation/observation, and explicitly requested native
operation after independently reviewed prerequisites. No generic shell/argv,
arbitrary server URL, custom MCP injection or implicit latest-session selection.
Use a fixed executable identity, exact workspace/session and one owned private
ACP process; no shared service, `--auto`, `--login`, credential export/import,
fork, child session, added directory grant or plugin installation. Native models
and profiles come from the explicitly selected inactive candidate once separately
adopted. Select and compare actual native model/mode config before any prompt.
Do not inherit the caller's HOME/XDG/provider/proxy environment by default.
Native preparation requires an explicit dedicated state root and an allowlisted
environment; never clone a primary store. Any later credentialed native use needs
the user's separately scoped state/secret provisioning decision, handled by the
native runtime without exporting credential values. No such use is performed
while implementing this feature or silently substituted for the normal-setup
human launch.

Launch declarations include actual authorization reference, intended effects,
model/data identity, deadline/attempt/token/spend bounds and evidence for the
controls enforcing them. **Declarations do not prove consent or controls.**
Missing strict pre-send token proof, credential isolation, complete accounting,
provider caps/allowance when required, or prevention of auxiliary retries blocks
prompt admission. ACP usage/events are observations, not an egress/billing gate.
No `--force`, free-price exception or claimed hard cap from step limits. This
feature adds no broker/proxy to bypass those unknowns; current online work stays
blocked. Normal human setup use still requires its separate actual launch scope.

Development never invokes a real prompt, private auth store or native model
process. Exercise the implementation through fake ACP peers and caller-supplied
synthetic observations. Native metadata/startup/cancellation behavior is not
certified by those tests; the final human cycle owns real-usage observations.

### C5 — Pending-human state and transport lifecycle

Bound ACP JSON-RPC 1 messages to 256 KiB each, aggregate retained observations
to 10 MiB, control requests to 10 seconds, and a session operation to an explicit
deadline (default 15 minutes for non-inference preparation; no inferred inference
allocation). Unexpected IDs, malformed/oversized data, unsupported handshake,
unknown required event shape or foreign session/location cause incomplete/blocked
output. Unknown optional native fields are discarded, not exported verbatim.

Advertise `elicitation.form`. On a consequential `elicitation/create` or pending
permission, keep the elicitation unanswered, capture the allowlisted question and
options, and cancel the owned active prompt; never send a fabricated answer or
decline that instructs the model to guess. Primary outcome is `blocked` with
`pending_human`, separate from the observed cancellation reason. Wait at most
5 seconds for native cancellation, then close owned transport/terminate only its
owned process group with bounded cleanup. If stop cannot be confirmed, record
incomplete/uncertain effects and never auto-resume or claim cancellation succeeded.
Handle concurrent late replies idempotently; no second prompt is dispatched.

Persist only a sanitized resumable question summary in a new private external
file, or stdout for non-mutating mode. Required fields: version, run/session and
workspace identity, request/authority references, exact question/options/rationale,
affected scope, evidence/finding pointers and next permitted action. Missing
rationale from native data stays explicitly unknown; do not invent one. This is
an observation, not a second approval ledger. Re-entry requires actual human
response and fresh state/authority reconciliation; no automatic replay.

ACP `stopReason: end_turn` establishes only that the native turn ended. A provider
failure, interrupted request, missing final event or cancelled tool cannot become
task success from process exit 0. Unexpected child-session metadata stops the
operation and invalidates aggregate usage claims. Retry/compaction events are
bounded observations; if required request accounting cannot be enforced, C4 blocks
admission rather than claiming that observing a retry prevented it.

### C6 — Evidence bridge and compatibility

Use the existing receipt **v1** without enlarging its mode enum. Helpers map to
the actual invoking workflow mode; standalone explanation/teaching need no fake
feature receipt. A pending-question summary is an adapter-specific sidecar and
v1 `execution: blocked`, `review: null`, `next_action`/`unknowns` refer to it.
Do not invent checks/tasks just to populate a receipt.

The Go evidence module implements strict typed/semantic v1 validation and scoped
identity/export using the current schema/reference contract. It does not attempt
a general JSON Schema engine or make Python a production dependency. Preserve
existing `sha256-v1` identity semantics; use independent golden fixture expectations
for parity, including staged/unstaged/untracked/deleted/mode-changed inputs. If
exact compatibility cannot be retained, request a versioned migration decision
before changing the wire format; do not silently change old digests.

Read only allowlisted native observations from the owned transport or explicit
user-curated sanitized input; no DB scan/full session export and no reasoning
capture. Record native-reported usage/cost as such, not verified provider billing.
ACP prompt usage excludes child sessions and optional fields may be absent:
unknown remains null with reasons. Refuse incompatible result/contract/check-log
digests, mismatched session/workspace/config or stale code. Snapshot before and
after relevant checks; do not associate an old result with a new snapshot.

Export new owner-private files outside the target checkout; allowlist fields,
bound inputs, redact known canaries before writes and require curation for unknown
secrets. No raw traces or complete config/environment exports. Default manual
retention stays seven days raw/thirty days sanitized; do not prune old sandboxes
or native stores. Receipts never write canonical state, resolve findings or set
technical Ready from a status alone.

### C7 — Completion and final human milestone

All selected requirements must have delivered artifacts and applicable focused
development evidence before `implementation delivered / human validation not run`.
Known unsafe defects and absent required implementation block that milestone;
unperformed native/benefit observations remain explicit rather than being
misclassified as code defects. Ordinary batch review/authorization is retained,
but no batch requires a human workflow-usage session.

The final handover lists exact source/runtime/config identities, inactive candidate,
safe opt-in/rollback, scenario order, existing evidence limits and carried IDs.
It covers all core/helper branches in one integrated human cycle, potentially
across sittings, only after all batches are delivered. The user separately chooses
task/workspace/live/model scope. The user may select the pinned QA container
recorded in spec §Approval instead of the normal setup; its image/binary
identities and limits then define the acceptance environment, while
normal-setup behavior remains unproven. No CI experiment, benchmark, paid request
or real-life test is executed to satisfy this development plan.

## Core coverage reconciliation

| Adoption-plan area | Implementation disposition / owner contract |
| --- | --- |
| Configuration, instructions, skill discovery | Reuse 004 foundation; complete C1–C3 namespaced package and metadata/candidate path. |
| Models/providers and task profiles | C3/C4 explicit inactive bindings/admission; capabilities/prices/accounting remain unverified until actual evidence. |
| Permissions/policies and isolation | C3 explicit mode grants/denials; C4 private owned session. Unproven routes remain unavailable; no OS sandbox claim from ACP or a worktree. |
| Verification gates/language checks | Reuse `sdd-check`/Make/compiler/vet/tests; C6 result integration. No new LSP or CI gate. |
| Context lifecycle and recovery | C2 handoff plus C5/C6 native state reconciliation, no reconstructed authorization. |
| Observability/evaluation | C6/C7 source-bound observations and final human scenarios; no model evaluator or savings claim. |
| MCP/tool contracts | C3 inventory of IDs/effects/destinations/timeouts and denied defaults; no new connections, custom tools or server. |
| Commands/headless/service | C3 prompt-only current-session commands and C4/C5 private ACP; no shared-service writes or advisory AI CI. |
| Networking/credentials | C3/C4 explicit host/operator ownership; no insecure TLS, proxy inheritance fixups, auth-store copying or login automation. |
| Hygiene/output/snapshots/TUI | C5/C6 bounded private output/manual retention, preserve existing disabled snapshot/warming choices; no cleanup automation/cosmetic extension. |
| Parallel work/children, plugins, warming/cache optimization, other adapters | Explicit R3/R4 exclusions in spec; not missing implementation in this selected feature. |

## Development verification, risks and rollout

Focused Go/temp-home tests, source/reference closure and schema/fixture validation
are permitted development evidence. Use test-first for new Go behavior; invoke
fake processes only with a minimal credential-free environment and no native
executable fallback. No routine full suite after every edit. Go-changing batches
run `make test`, `make vet`, `git diff --check` and applicable required local gates;
no new full CI, containerized real-AI or PR experiment. Resolve a check-scope
conflict before executing a forbidden gate; never bypass hooks or delete tests.

Key risks and responses:

- Helper sprawl/authority drift: C1/C2 shared ownership, trigger matrix and small
  independent semantic-case expectations; static checks are not model compliance.
- CLI silently continuing after a question: pinned-source evidence rejects that
  design; C5 handles native forms before answering them and reports uncertain stop.
- Native version drift: exact target/source identity; unsupported protocol fails
  before prompt, with no automatic upgrades or older-protocol fallback.
- Unknown model budgets/retries: C4 denies prompt admission; no alternate endpoint
  or soft mental estimate. Token proof does not block independent portable work.
- Config corruption/secret copying: C3 user-curated bytes, separate private backup
  manifests, explicit paths and digest-checked rollback; no generic JSONC merger.
- Forged/stale evidence: C6 independent scoped identity/log/result comparison,
  provenance limits and no canonical state writes.
- Existing dirty 004/user work: never reset or overwrite it. Keep publication
  changes separate; keep private handovers/raw evidence out of commits.

No live rollout occurs during development. Skills use the existing opt-in
installer; runtime config uses its separately reviewed transaction. The final
human launch cannot be obtained by marking templates/providers enabled on behalf
of the user. No release tag, credential mutation, push or PR follows from approval
of these drafts alone.

## Specialist and tool assignments

Use one main agent; no delegation. Available catalog guidance, loaded only when
needed: `skill-creator` and `writing-for-agents` for helper structure/provenance;
`opencode` for exact native contracts; `golang-pro` for CLI/adapter/evidence code;
`backend-security-coder` for projections/admission/config/export boundaries;
`agentic-sdd-architecture-review` for consequential boundary self-review; and
`agentic-sdd-verification-review` once per implemented batch. No additional
specialist is needed for inventory/continuation prose. These assignments grant
no child agents, model changes or extra external usage.

Conditional tools: local source discovery (CodeGraph already reported this repo
unindexed), Read/Grep/Glob, shell for focused deterministic checks, Context7 and
official pinned source for native syntax uncertainty. Browser/CI/provider/model
tools and live native execution are not needed for implementation validation.

Architecture self-review of this draft: **Ready for combined design review**,
not execution approval or runtime certification. C4/C5 explicitly retain the
online admission block; C6 is a real implementation task rather than treating
the old Python fixture as a shipped exporter. Review corrections included the
CLI question-continuation mismatch, native inventory loading executable plugins,
accidental HOME/XDG/credential inheritance, explicit-only visibility versus
permission semantics, and runtime-config versus skill backup ownership. These
are same-agent design checks, not an independent review or tested guarantee.
Next: [tasks.md](tasks.md).
