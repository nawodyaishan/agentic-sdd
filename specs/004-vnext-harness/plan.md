# vNext harness implementation plan

Revision: r1 (draft, 2026-10-28). Input: [spec.md r1](spec.md), AC-01–AC-12, and its linked adoption plan revision 5. Approval is recorded only in [spec.md §Approval](spec.md#approval).

## Approach

Deliver the R0–R2 outcome incrementally: establish a reproducible baseline and contracts, strengthen the portable workflow, add an opt-in native OpenCode adapter, connect deterministic verification and evidence, then prove interruption/recovery and connected feature behavior. Keep OpenCode's agent loop. No plugin, worker, autonomous job or general orchestration layer is needed.

Repository discovery established:

- `internal/skillsync/sync.go` discovers all `agentic-sdd-*` directories with a regular `SKILL.md`; `skillsfiles.go` embeds the entire canonical tree. Adding debugging needs inventory and compatibility checks, not a new skill-selection mechanism.
- Skill backup manifests and restore validation describe known client skill trees, not arbitrary runtime configuration. Do not put OpenCode config files into those manifests or change their meaning.
- `config/client-overlays.json` concerns existing clients; `config/local-skill-catalog.json` explicitly describes a historical local inventory. Neither is an OpenCode runtime configuration or proof of today's access.
- `Makefile` and `.github/workflows/ci.yml` already run module verification, tidy checking, vet, tests and Darwin builds. Reuse these commands rather than replacing their implementations.
- `tests/docker-e2e.sh` already uses a network-disabled container, a read-only source mount and a temporary home. Extend that isolation pattern for deterministic fixtures, not the real user environment.

## Ownership and affected components

Paths marked new are intended delivery locations, not existing functionality.

| Component | Change and owner |
| --- | --- |
| `skills-files/agentic-sdd-router/` | Own shared authority/lifecycle rules, action-first routing and shared references. |
| Existing spec/plan/tasks skills | Own acceptance examples, material assumptions, shared design contracts and readiness reconciliation respectively. |
| Implement, verification-review, research-spec, drift-retro and bootstrap skills | Align lifecycle, diagnosis/review handoffs and context pointers without duplicating runtime settings. |
| `skills-files/agentic-sdd-debugging/SKILL.md` (new) | Reusable diagnosis process; not a domain specialist. |
| `harness/README.md`, `harness/checks.json`, `harness/evidence/` (new) | Portable contract, executable check descriptions, receipt schema/examples and compatibility evidence pointers. |
| `integrations/opencode/` (new) | Version record, credential-free native config templates, adoption/rollback procedure and native-mode/recovery instructions. No active root configuration is installed by development. |
| `cmd/sdd-check/`, `internal/harnesscheck/` (new) | Thin standard-library Go check runner; independent of model runtime and of the skill installer. |
| `tests/harness/` (new) | Deterministic contract/boundary fixtures and recorded behavioral scenario definitions. Not a new benchmark service. |
| `Makefile`, `.github/workflows/ci.yml` | Expose and reuse the check contract; protect the required check set from candidate weakening. |
| `README.md`, `AGENTS.md`, existing installer tests | Update product/skill inventory and canonical pointers; verify embedded and on-disk distribution compatibility. |

Shared reference additions are limited to task/readiness, testing and review-resolution guidance when necessary to keep entry skills concise. Each rule has one owner; other skills link it.

## C1 — Mode, authorization and result contract

Inputs: mode (`draft`, `build`, `direct-fix`, `review`, `resume`), workspace, request reference, feature/batch when applicable, actual authorization reference, permitted effects, selected model/profile, available tools, checks and limits. `resume` resolves to the original permitted action after reconciliation; it grants no effects by itself.

Unknown observations are nullable with a reason. Missing required authorization, unsafe workspace identity or unsupported mode guarantees block the affected action. Direct fixes use the request as their scope source and omit feature/batch IDs.

Output dimensions:

- Execution: `completed`, `incomplete`, `blocked`, `cancelled`, `failed`.
- Technical review, when performed: `Ready`, `Needs changes`, `Blocked`.
- Workflow state: current batch/task position and next permitted action; an authorized batch stops at `awaiting human review`, with unresolved issues preserved.

`completed` means the requested execution ended, not that review passed. `Ready` requires fresh relevant deterministic evidence plus requirements/engineering review. Neither creates approval. A blocked prerequisite is distinguished from an executed failing check. Partial results survive cancellation.

Implementation owns the completion-review invocation and authorized task-state write. Verification-review returns findings and verdict; standalone review writes no project state. The router consumes implementation's result. Correction changes may trigger scoped re-review; unchanged results do not receive duplicate completion reviews.

The combined approval remains `spec.md#approval`; `tasks.md` owns continuation state. Receipts reference both. Explicit new batch authorization is reconciled with stored state, while vague continuation and stale summaries never imply all-batch permission.

## C2 — Native adapter, adoption and enforcement

### Configuration delivery

Use `.opencode/opencode.jsonc` as the one project-config convention, with namespaced `sdd-*` primary profiles and commands defined there. Ship the template under `integrations/opencode/`, not as an automatically active config in this repository. Commands contain prompts only: **no shell interpolation blocks**, because current V2 docs say those execute outside agent tool permissions. Set commands to the current session; no child execution.

Initial delivery is a documented, tested manual adoption procedure rather than a config-merging CLI:

1. Inventory direct and `.opencode` config and discovered commands/skills/plugins, including ancestors. Explain conflicts before choosing the effective location.
2. Stage the proposed native config outside active discovery; show the diff and validate it against the pinned runtime in an isolated fixture.
3. Back up each existing file to user-owned storage outside Git with original path, digest and existence state. Preserve unrelated keys and definitions; ambiguous JSONC edits stop for manual resolution.
4. After explicit installation authorization, apply only the reviewed change. Identical content is a no-op. Refuse symlink/special-file destinations.
5. Verify the effective result. Rollback restores the backup, or removes a freshly added file, only when the current content still matches the installed digest; otherwise reconcile intervening edits rather than overwrite them.

Exercise the procedure in temporary directories, including preexisting unrelated keys, duplicate definitions, no-op, rollback and post-install edits. This satisfies reversible opt-in adoption without extending skill-only backup formats. Skills continue using existing installer backup/restore. A generalized config installer would be a separate design change.

### Profiles and hard boundaries

All profiles begin deny-by-default and explicitly allow only tested actions:

| Profile | Initial allowed behavior | Restrictions |
| --- | --- | --- |
| `sdd-draft` | Scoped source reads, required skills/questions, edits to the selected feature's agreed documents | No general shell, browser, MCP writes, delegation or production-code edits. Replace the selected feature path explicitly; no broad `specs/**` write grant by default. |
| `sdd-build` | Scoped source reads/edits and explicitly authorized check commands | Shell is ask-by-default with tested narrow exceptions; no push/deploy/credential access or delegation inferred from batch approval. |
| `sdd-review` | Scoped native file observation, approved skill reads and questions | Deny editing, shell, browser, MCP, resource reads, Code Mode and child creation initially. Consume supplied current-diff/evidence exports rather than run tests. |

Read rules must account for symlink/canonical paths, secrets and external boundaries. Grep permissions match the query rather than the search path in current docs; do not claim they provide filesystem isolation. For strict review, use a sanitized read-only fixture/export with no private host mounts, and disable search routes that cannot be scoped adequately.

Higher-authority policy examples remain separate from project defaults and are installed only by the authorized user. Deny unapproved providers/integrations before connections load. Review uses a separate clean runtime environment where inherited executable plugins and MCP connections cannot run implicitly. Tests verify effective behavior, including invalid statements and lower-authority re-enablement attempts; a plausible deny string is not evidence.

Runtime policy covers managed paths, not arbitrary plugin code or shell process capabilities. Use an unprivileged isolated container for executable negative fixtures: read-only inputs, dedicated writable scratch, no host home/credentials/socket mounts and no network for deterministic cases. A model-driven fixture needs separately authorized provider egress/credentials; keep the test program's environment separate from that credential-bearing service. If the host cannot provide the required boundary, mark the runtime scenario blocked.

### Versions, models and integrations

Before adopting settings, B1 records the installed CLI and service versions separately, configuration sources and a sanitized capability inventory. Pin the tested exact release; documentation pages consulted here are current product documentation, not proof of installed behavior.

Economy, Standard and Deep are logical labels. Initially all may map to the same verified model; at most one alternative is permitted. Select actual IDs/variants only from available authorized access. Use explicit command-level selection where required and verify the observed model; agent selection alone is insufficient. No guessed model names, implicit provider fallback or automatic escalation.

Initial run limits proposed for fixtures: one active session, zero children, 15-minute deadline, at most 20 model steps, at most two transient retries per operation; non-idempotent actions require reconciliation instead of automatic retry. Limits are recorded per run and may be tightened. A required increase returns an incomplete result for a decision. Native step limits alone are not elapsed-time or billing controls.

The additional monthly ceiling is US$10; paid evaluation remains disabled until explicitly authorized. If authorized later, cap the first comparison allocation at US$1 and include all failed/retried work. Unknown price, remaining allowance or provider billing limits block additional paid execution until resolved. Use provider-side controls where available; otherwise report the absence of a hard cap and obtain an explicit bounded-run decision rather than claim one.

B3 incident follow-up (user-requested, no acceptance waiver): the incident
reconciles approximately $0.825 OpenRouter spend, not the earlier $0.706 event
estimate. Do not clone primary credential databases for evaluations. Offline
tests use no provider egress/credentials; future paid execution requires a
dedicated user-provisioned capped key, verified pricing/allowance and bounded
requests including auxiliary/in-flight work. An environment opt-in alone is
insufficient. The user's selected DeepSeek/GLM free roles and dormant MiMo/
Nemotron candidates live in `integrations/opencode/model-usage.json`; model,
free-price, tool capability and privacy eligibility are distinct observations.
All shipped provider routes remain denied; native permission API probes do not
substitute for real tool/selection acceptance. These constraints implement the
existing fail-closed C2 boundary; they do not expand deployment/model scope.

Do not enable an MCP merely because it exists in the user's setup. An enabled integration's record specifies actual tool/resource IDs, mode, destination, credentials by reference, bounded output, startup/catalog/execution timeout and cancellation/retry semantics. Review keeps these disabled until an independently tested read-only route is warranted. Native Code Mode is preferred for permitted build/draft integrations, with nested permissions verified.

## C3 — Portable deterministic check contract

`harness/checks.json` is a versioned list of checks. Each entry defines a stable ID, argument vector (no implicit shell), repository-relative working directory, scope tags, prerequisites, positive timeout in seconds, write/network effects, expected exit and evidence output. Paths must stay within the selected workspace; IDs are unique; unknown schema versions or invalid entries fail before execution. The contract is executable code in data form and must come from a trusted revision.

The thin Go runner invokes existing commands without interpreting arbitrary shell text. It selects explicit scopes, records start/end and exit status, preserves full bounded-on-disk logs with truncation metadata, and emits a structured result. It never invokes an LLM, changes task approval or sets Ready. An unavailable prerequisite is blocked, a nonzero executed check fails, interruption is cancelled, and a deadline without completed evidence is incomplete. The runner stops scheduling new checks after a required failure and marks remaining checks not run.

Proposed CLI: `go run ./cmd/sdd-check --scope <scope> --workspace <path> --contract <path> --output <external-directory>`. Require an explicit output directory outside the target checkout for review-derived evidence. `--scope` selects `baseline`, `focused` or `required`; focused runs also require explicit check IDs so a partial run cannot impersonate required coverage. Process exit: 0 only for all selected checks passing, 1 for failed/incomplete/cancelled execution, 2 for invalid input or blocked prerequisites. Structured statuses remain authoritative for diagnosis, not for human authorization.

Use standard-library Go and small explicit functions. `internal/harnesscheck` exists because check execution, receipts and their fixture tests share a distinct contract, not to generalize the installer. The first supported execution hosts are Linux and macOS; unsupported process-tree cancellation behavior blocks the relevant check instead of claiming it stopped. Track owned process groups, request graceful cancellation, then bounded termination; inspect remaining activity and uncertain effects.

Initial required project checks reuse `make mod-verify`, `make tidy-check`, `make vet`, `make test`, `make build-darwin` and `git diff --check`; add harness contract/boundary tests to ordinary Go tests. Avoid recursive verification: the runner's tests use small subprocess fixtures, never call the full required runner from `make test`. Container end-to-end checks remain an explicit prerequisite-dependent scope and must run before claiming installer/isolation acceptance.

`make verify` becomes the local entry to the contract; its component targets remain available. Preserve existing behavior and document side effects: tidy may write module files, builds/caches write files and dependency retrieval may need network. Baseline and final runs record this rather than treating them as read-only.

### Trusted CI gate

For pull requests, run the check runner and required command definitions from the trusted base revision in a separate directory against a disposable candidate checkout. Reject removal/replacement of required IDs and compare candidate changes to the gate/tests as review inputs. Do not rely solely on candidate `make` targets: trusted contract commands for required Go gates resolve to the actual `go mod verify`, `go mod tidy` plus diff check, `go vet`, `go test` and Darwin build argument vectors, or a trusted copy of equivalent recipes. This preserves behavior even if a candidate empties a Makefile target.

Candidate tests still execute code: use no publish tokens, persist no checkout credentials, set minimal read permissions, bound execution and isolate dependencies/test execution. The gate is not proof that all tests are good; T10 supplies a known weakened-check fixture and human/agent review covers legitimate test changes. New required checks enter the trusted contract only through reviewed adoption. Remote branch-protection setup is outside this feature; do not claim local workflow files enforce merge rules. B1 must identify a trusted pre-change reference; if none contains the new runner yet, report bootstrap trust as a release prerequisite rather than trusting a PR's runner by default.

B7 implementation uses a base-owned stdlib Python bootstrap and the existing Go
runner, a pinned Go container per check, and an independent base-owned no-op
probe in disposable candidate staging. Local trusted-Make and CI use the same
adapter. Proposed `pull_request_target` loads the main-base workflow; read-only
checkout tokens are not persisted and candidate code never runs on the host.
No host candidate Make/install hook, credential mount, shared trusted cache or
remote branch administration is allowed. B4 supplies the committed runner;
the B7 launcher/workflow/probe still require reviewed trusted-branch adoption.
Architecture self-review: consistent with C3 within this boundary; shared-kernel,
disk-DoS, uncatchable cancellation and actual hosted activation remain limits,
not stronger guarantees. Details and official-source references are in
`harness/trusted-ci.md`; native correlation stays unresolved.

## C4 — Evidence, freshness and retention

Use one versioned JSON receipt schema in `harness/evidence/`. Fields include native session/run ID when available, task IDs, C1 input/result, runtime/config/skill revision identities, actual model/variant, base commit, staged/unstaged/untracked relevant-content identity, check IDs/results/times/log references, findings and unknowns. Direct-fix feature fields are null, not synthetic tasks. Schema validation checks shape and internal consistency, not truth or approval.

Diff identity covers both file paths and content, deletions and untracked relevant files; do not hash only HEAD or only `git diff`. Freshness is evaluated against each check's code, configuration, dependency, input and environment scope. Unknown scope identity prevents evidence reuse. Post-format or post-merge changes refresh affected checks and review verdicts.

Use existing native exports when adequate, otherwise fill the portable receipt from observed events/check output; no plugin or automatic completion ledger. Persist run evidence under user-owned external storage (proposed default `$XDG_STATE_HOME/agentic-sdd/runs`, or `~/.local/state/agentic-sdd/runs`). Use owner-only access. Commit only sanitized synthetic fixtures and deliberate sanitized baseline/comparison summaries, not raw user traces.

Default retention proposal: raw logs/exports 7 days, sanitized receipts 30 days, manual cleanup documented; never prune native OpenCode databases/snapshots automatically. Explicitly retained acceptance evidence is exempt and has a recorded owner/expiry. Allowlist exported fields, redact before writing portable artifacts, and use secret-canary fixtures to test redaction. Sensitive full native logs remain under their host's access/retention policy, not copied blindly.

B6 offline implementation uses the existing optional Python/jsonschema fixture
tooling for a reference consumer (`tests/harness/receipt_tools.py`), not a new
Go/runtime dependency. Schema v1 stays unchanged. Scoped Git identities include
HEAD/index and current/untracked/deleted file observations; callers separately
supply current metadata and previously captured result/log identities. This
checks consistency, not authenticity of a candidate-provided observation or
human authority. Native export correlation remains blocked. See
`harness/evidence/README.md` for scope, bounds, trust assumptions and safe export.

## C5 — Recovery and review behavior

Resume reopens the current spec approval, tasks and diff, then resolves workspace identity, authorized action, model/tools, limits, open findings and freshness. The old summary is a pointer, not authority. Changed location, uncommitted files or unresolved side effects are reconciled before new work.

Keep native compaction and native cancellation initially. Maintain manual recovery steps and prove them through observed sessions. Do not add a headless runner to automate agent continuation. The deterministic check runner tracks only processes it starts; native session cancellation must be independently observed. Closing a stream, cancelling a check or deleting a workspace is not a substitute for stopping the agent.

Review findings have stable IDs, affected location, impact, evidence and disposition. Implement only valid in-scope corrections, reject incorrect recommendations with evidence, and return scope-expanding findings for decision. Propose a maximum of two corrective review passes per fixture run; exhaustion leaves Needs changes/Blocked. This is an effort ceiling, not an instruction to ignore a correctness problem. Final feature review evaluates aggregate acceptance and cross-batch behavior.

## C6 — Evaluation and baseline

Before modifying canonical skills, freeze the baseline revision and relevant uncommitted source identities. Define three bounded fixtures grounded in this repository:

- Feature: a disposable replay of an existing installer contract crossing CLI, sync/restore and filesystem behavior; choose a frozen historical input with expected acceptance, without reopening its real approval.
- Regression: inject a known no-op/restore regression only into a disposable copy, with an independent expected filesystem outcome.
- Review: a seeded diff with a known contract defect and a tempting logging edit; require findings without mutation.

B1 selects exact revisions/inputs and reserves one unseen variant before any tuning. A historical replay measures execution behavior, not novel design quality. Also use this real feature's later batches as prospective integration evidence; do not claim historical fixtures alone prove open-ended design performance.

Scripted cases cover malformed contracts, cancellation, policy precedence, invalid policy, config collisions, evidence staleness, secret redaction and weakened gates. Behavioral cases cover Appendix A8 and AC-01/05/07/08/10; inspect diffs and traces, not Markdown phrase presence. Keep two critical-case repetitions where authorized and report counts, not reliability percentages.

The baseline and candidate must use the same model, fixture inputs, tools, permissions and check environment. If baseline execution is unavailable or paid use is not authorized, retain frozen fixtures and mark observations blocked. Do not tune and retrospectively invent a baseline. Paid comparisons require the separate decision in C2. Adoption stays open until useful benefit and unchanged quality are evidenced.

## T11 follow-up proposal — isolated free-model acceptance (2026-10-30)

**Preparation only; no inference is authorized by this proposal.** The user
selected full acceptance, DeepSeek Flash Free and an isolated Linux pilot,
and said “use any free opencode / openrouter model”. Propose the already chosen
exact `opencode/deepseek-v4.1-flash-free`, with no variant override and no fallback.
Changing the endpoint/model requires another bounded decision, not an implicit
substitution. All existing acceptance criteria and findings remain in force.

#### Public-source facts and limits

Official V2 [Console models](https://opencode.ai/v2/docs/console/models),
[providers](https://opencode.ai/v2/docs/providers),
[agents](https://opencode.ai/v2/docs/agents) and
[compaction](https://opencode.ai/v2/docs/compaction) were read on 2026-10-30,
with Context7 `/websites/opencode_ai/v2` corroboration. The models page lists
`deepseek-v4.1-flash-free` separately from the paid unsuffixed model, prices its
input/output/cache reads as Free, and states no training/zero retention for
its provider. Free availability is time-limited and free chat endpoints require
OpenCode. These are published endpoint facts, not an observed inference,
account entitlement, tool-compatibility result or universal privacy guarantee.
No catalog/inference/account endpoint or primary credential store was queried.

The same docs warn that session titles may use other low-cost models; hidden
title/summary/compaction agents perform maintenance, step allowances reset with
new input, and provider timeouts may be retried. Therefore setting a free main
model plus `steps` and a wall deadline does **not** demonstrate a hard cumulative
request/token/model boundary. Do not enable the existing provider-denied template
and assume it is safe. Current docs also do not prove pinned 2.2.0 behavior.

#### Proposed execution slices (each stops for review)

| Slice | Scope and evidence | Execution authority / limits |
| --- | --- | --- |
| S0, offline safety preparation | In a fresh credential-free, network-disabled native fixture, observe request construction against a local fake endpoint; exercise wrong-model, retry, auxiliary-request, accounting/deadline denial and owned cleanup. Establish effective config and binary identity. No real endpoint or inference. | Authorized 2026-10-31 and delivered for review. Byte-budget/loopback controls are observed; verified real tokens and online isolation/allocation remain S0-F1/F2. No product proxy/plugin or implicit fixture relaxation. |
| S1, first native capability preflight | One fresh synthetic project/session on exact 2.2.5; exact model/command/session observation, read/edit positive control and actual check-result/native-session correlation. Keep all other providers/integrations/children unavailable. No private repository, incident, user config or original credential content. | **Conditionally authorized 2026-10-31**, see spec §Approval: exact `opencode/deepseek-v4.1-flash-free`, 6 upstream inference attempts total including auxiliary activity; 32,768 input / 2,048 output tokens per request, 196,608 input / 12,288 output aggregate; 10-minute total / 90-second per-request deadlines; one active primary session, zero children/forwarded retries, $0 paid usage. Any unsupported/unverifiable accounting/egress control blocks forwarding. No S2/S3 allocation or paid fallback. |
| S2, native acceptance | Actual mode/tool/canonical-path denials, all eight B5 behavioral cases, connected lifecycle, native cancellation/compaction/restart/location resume, no duplicate review and trusted native receipt correlation; two repetitions of critical native cases. Test reversible adoption only in a dedicated disposable pilot. | Separate exact scenario/session/request/token/deadline allocation after S1 review; **no open-ended allocation now**. Direct real-home apply, shared-service changes and macOS support are not implied. |
| S3, comparison and final review | Hold model, frozen fixture inputs, tool/permission/runtime environment and evaluation oracles fixed across old and candidate instructions; include failures/interventions/latency/full usage and a user-supplied undisclosed concrete holdout. No model judge or tuning to the holdout. | Separate exact allocation after S2 review. Retain original pre-tuning B1/B2 observations unchanged. A matched replay of frozen old instructions on DeepSeek would be a new retrospective study, not a newly invented pre-tuning baseline or a directly comparable B2 cost result. If provenance/benefit is insufficient, remain Blocked. |

#### Isolation and authorization prerequisites

- Use only synthetic pilot inputs, explicitly allowlisted public skill text when
  needed, fresh native HOME/XDG/state and dedicated owner-only external evidence.
  No host home/auth database, Git credentials, primary service or Docker socket
  is exposed to model-selected test programs. State/restart changes affect only
  the pilot. No existing credential-bearing sandbox is opened or copied.
- Before S1, demonstrate a single controlled outgoing path that rejects every
  unapproved host/path/model and counts **all** generation attempts before send,
  including concurrent and auxiliary work. A blind HTTPS host allowlist is not
  model/request enforcement. The test-program execution environment must be
  separate from provider egress/credential-bearing execution. Failure or an
  unobservable bypass stops the affected slice.
- No runtime-supported enforcement has yet been proven. An isolated lab-only
  forwarding gate would be a consequential design addition and touches the
  exclusion of proxies/custom infrastructure; **do not implement it without a
  separately scoped user decision**. A user-provided existing equivalent may
  be used only after its actual controls are reviewed. Do not weaken limits to
  bypass this unresolved prerequisite.
- No paid credential, recharge, subscription-sharing token, provider probing or
  fallback is permitted. Prefer a credential-free free route if the pinned
  runtime supports it. If authentication is necessary, first review a dedicated
  scope with the user, who supplies a secret reference outside chat; never ask
  for a key value or copy the primary auth store. The previously reported
  OpenRouter cap is not an OpenCode credential or verified current allowance.
- Reconfirm the exact endpoint's published price/policy before execution.
  Runtime claims must match actual selected/sent IDs and destinations. Unknown
  price, uncontrolled auxiliary routes, delayed/unknown billable usage or
  changing terms stops the run. Do not fetch paid alternatives to diagnose it.
- An isolated Linux pilot does not supply native Darwin evidence or silently
  revise the approved supported-host plan. Any remaining host-support scope
  decision returns for review; remote required-check administration and actual
  user installation remain separate optional live operations, not hidden tasks.

Main-agent architecture assessment: **Blocked for online execution**. The S0
lab-only exception is now approved and offline byte/HTTP/native-fake controls
are observed; real-token accounting and online egress/credential/allocation
provenance remain unverified (S0-F1/F2). Public price/privacy documentation narrows
B3-F3 but does not close B3-F1/F2/F3/F4, B5-F1, B6-F1/F2 or B8-F1/F2. Next is
S0 review and resolution of those prerequisites, not inference permission or a
promise that the feature will become Ready.

#### S0 implementation contract (authorized 2026-10-31)

Keep this optional stdlib fixture under `tests/harness/`, never in the product,
active config or an installed skill. Its HTTP upstream is fixed to numeric IPv4
loopback in the same network-disabled container; no DNS, external URLs, redirect,
CONNECT, WebSocket, inherited proxy or runtime endpoint switch. Only one exact
chat-completions path/model and a bounded JSON field allowlist are admitted.
No user headers/auth material are forwarded. Error/log records are metadata-only.

The run-level budget is shared across sessions/maintenance requests, locked
before reservation, and counts every accepted forwarding attempt including a
failed upstream request. Reserve full requested output capacity and observed
input units before dispatch; never refund on failure/cancellation or trust usage
events to release allowance. Persist reservations with fsync before send; an
existing allocation journal cannot be reopened/reset after a restart. Unknown
accounting, malformed input, cancellation, expiry, read limit, persistence error
or uncertainty stops forwarding. Per-operation timeout and response-size bounds
are separate from the absolute run deadline. Streaming may be buffered under
those bounds for this offline test; it is not certified online streaming.

S0 uses a deliberately conservative **UTF-8 byte quota** for synthetic HTTP
request/response payloads, explicitly not a verified DeepSeek tokenizer. Unknown
real-model input/output token accounting remains blocked for S1; do not rename
byte counters “tokens”. Native request construction against a fake provider
does not prove the real OpenCode free endpoint accepts routing/authentication or
model limits. No real endpoint/credential path exists in this implementation.

Test the state machine first, then loopback HTTP behavior in the unprivileged
network-none container and the explicitly selected native request shape with fake responses only.
Execute synthetic test programs in a separate network-none worker with no
runtime/gate state, headers or control socket mounted. The trusted host test
driver owns both containers and cleanup; no Docker socket enters either.
This demonstrates an offline separation, not verified online egress/credential
isolation or a general native-shell sandbox. Load `opencode` guidance in the
main agent and `backend-security-coder` for the transport/validation boundary;
no subagents, external tokenizer installs or native-template changes.

S0 follow-up observation (2026-10-31): the actual binary was 2.2.5, not 2.2.0;
the initial guard stopped before prompt submission. User allowed later versions,
so only the observed 2.2.5 binary is frozen/revalidated. The S0-only config must
explicitly allow its fake `integration.use` entry under 2.2.3+ semantics, with
separate MCP/skill/plugin denials. `max_completion_tokens` is observed on the
main request; exactly one of it and `max_tokens` is supported, never both.
Cap-less requests are rejected. Byte accounting/loopback-only egress and unknown
online allocation provenance remain S0-F1/F2, not a completed real-token boundary.
See `tests/harness/S0.md` and current tasks for actual results and readiness.

#### S1 accounting research and dependency decision (2026-10-31)

Named blocker: establish real input/output accounting and the selected hosted
endpoint's request projection before forwarding; a raw-byte quota is not renamed
a token quota. S0 is now accepted and the exact S1 envelope conditionally
authorized; do not re-request unchanged model/budget authorization or waive its
conditions to obtain a result.

Official [DeepSeek token usage](https://api-docs.deepseek.com/quick_start/token_usage)
(Context7 `/websites/api-docs_deepseek` plus direct page) says character ratios
are approximate and response usage is authoritative. It links the offline
[V4 tokenizer archive](https://cdn.deepseek.com/api-docs/deepseek_v4_tokenizer.zip).
Public source inspection only: SHA-256
`e7310d1dafe0a86d8a5629fe78a7c763760f651db9b8682718a1781dcd6fe495`,
retained at `/tmp/opencode/sdd-s1-research.f9hbu53h/public-tokenizer.zip`.
No archive code was run or installed. It contains tokenizer JSON/config and a
277-byte Python demo recommending `transformers` with `trust_remote_code=True`.
**Do not execute that demo.** JSON describes byte-level BPE, no normalizer,
a chat template and legacy `model_max_length=16384`; neither that stale-length
field nor the archive alone proves the OpenCode-hosted V4.1 chat/tool projection.
Keep server-inserted framing/tools/reasoning and gateway translation as explicit
verification prerequisites, not locally invented constants.

The existing host lacks `tokenizers` and `transformers`; no install was attempted.
Hugging Face `/huggingface/tokenizers` docs support local
`Tokenizer.from_file(...)` / `encode(..., add_special_tokens=False)` without
running model-provided Python or calling the Hub. Candidate **for review only**:
official PyPI `tokenizers==0.22.1`, Linux cp39-abi3 manylinux x86_64 wheel,
3,250,221 bytes, SHA-256
`e2ef6063d7a84994129732b47e7915e8710f27f99f3a3260b8a38fc7ccd083f4`.
PyPI metadata declares `huggingface-hub<2,>=0.16.4`; do not install it or other
transitives automatically. A scoped exception is needed for optional isolated
lab dependency acquisition/import, with archive/member audit, pinned hashes,
read-only mounts, no network while testing and no system/user/image/Go dependency
changes. If that wheel cannot work without more reviewed dependencies, stop and
report it rather than resolve packages or download model code implicitly.

Official OpenCode Console models page was re-read on 2026-10-31: the exact free
endpoint remains listed free with the stated no-training/zero-retention policy.
No inference/catalog/account endpoint was probed. This public policy evidence
does not establish availability/auth requirements or every actual request path.
S1 still needs the separately reviewed network/credential/allocation boundary;
do not turn the offline loopback reference into an internet forwarder by simply
relaxing its destination guard. Tokenizer acquisition alone does not clear it.

Planned online topology (not implemented/verified): the native runtime has only
an internal pilot network to a trusted gate and a dedicated synthetic workspace;
it receives no host home, provider credential, gate journal or Docker socket.
Only the gate has provider egress, with a fixed TLS-verified host/path/model,
no redirects/proxy inheritance, durable allocation-bound reservations and a
single external deadline. Synthetic test execution remains a separate
network-none worker driven by trusted checks, not a model-selected host shell.
Unknown token projection, cap-less calls, unapproved destinations/headers,
loss/restart of allocation state or any bypass stop forwarding. Prove the
topology with fake upstream/canary negative controls before enabling exactly
the approved real route. No paid account or primary auth store is needed or
permitted merely to discover whether the credential-free free route works.

#### Local source audit result and remaining decision (2026-10-31)

The approved exact wheel was acquired and its members inspected, then only its
package files were unpacked into disposable `wheel-library/` under the private
research root. No pip/system/user/image install or automatic transitive package
resolution. With readonly wheel/data mounts in the existing non-root network-none
Python container, **four source-audit tests pass** without importing Transformers
or Hugging Face Hub. They check fixed public-data hashes, independent special-token
IDs, known text, disabled truncation/padding and a 40,000-token input that must
not silently truncate. A literal plain system/user chat example counts 9 local
tokens. Reproduce using `tests/harness/test_tokenizer_source.py` with explicitly
prepared `/wheel` / `/token-source` mounts; it is not part of production or a live
governor. The exact data identities and result paths live in tasks.

**S0-F1 is narrowed, not closed.** A working local tokenizer does not establish
the hosted `opencode/deepseek-v4.1-flash-free` chat/tool/reasoning projection.
The inspected public template does not by itself prove the gateway's added
framing/tool-schema treatment. The current strict pre-send input-token condition
therefore remains unverified. Per the dependency decision, stop here rather than
install more packages, label byte/character estimates exact or submit a real
request to discover whether the original condition holds. The separate online
egress/allocation proof also remains outstanding. Original S1 authorization is
retained but no generation allowance is spent.

Scope-review alternative **P1 (authorized by corrected choice, 2026-11-01)**: a distinct one-request
observational measurement, not an acceptance pass or a relaxation silently
applied to S1. If the user explicitly selects it, it consumes **one of S1's six
total allowed upstream attempts**, not an extra allocation. Exact model and
runtime stay unchanged, zero variant/fallback/paid usage, no provider credential,
synthetic literal text only and no tool execution. Hard outgoing body limit
**4,096 UTF-8 bytes**, requested output cap **256 tokens**, one primary session,
zero children/retries, **90 seconds** for the request and the existing ten-minute
overall limit. Input tokens would be measured/validated from the response, **not
claimed pre-proven**. No receipt or result can label original S1's hard input
condition satisfied on that basis. Stop after this attempt even on success;
remaining original S1 attempts stay conditional. If usage/zero pricing is
unavailable or uncertain, record unknown and do not continue. Online egress/
request/allocation controls still must pass fake negative tests before any P1
forwarding; this alternative is not permission to bypass those controls. The
other choices are supply matching hosted-projection evidence or keep S1 blocked.

P1 implementation boundary: keep S0's loopback/byte fixture unchanged. A separate
lab-only controller accepts a short authenticated request from an internal-only
native container, validates exact model/path/cap and the literal P1 user prompt,
then emits a **fixed text-only canonical provider payload**, omitting tool schemas,
native system/environment details and caller headers. Record both input/payload
identities; do not claim the transformed payload is original S1 tool execution.
Only the gate can reach a fixed TLS-verified `opencode.ai` chat-completions path;
no redirect, automatic retry, runtime-selected URL, inherited proxy or provider
credential. Forward only bounded genuine OpenCode client-identification headers
needed by its signed-out free tier, never authorization/cookies/local gate tokens.
If the endpoint requires an account or unsupported routing, stop rather than
fetch credentials or substitute a paid route.

Use a newly created private **stable P1 allocation directory** outside the repo,
exclusive/fsynced reservation before network dispatch and no restart/reset/refund.
The real allocation path is fixed for this authorization, not a fresh temp path
on each invocation. A prior/uncertain marker prevents another attempt. Offline
tests use separate fake allocations. Gate and runtime have separate containers;
runtime is only on a dedicated internal network, gate alone also has outbound
connectivity. No published ports, host home/state or Docker socket in either.
Only trusted host code owns network/container creation and scoped cleanup.

Before real execution test wrong/uncapped models, framing, body size, bad local
capability, concurrency/replay/restart, redirect/error, deadlines and runtime
direct-egress denial against fake upstreams. For P1, output-token requests and
response-byte limits are separate units; never reuse S0's byte reservation as a
token count. Buffer a bounded JSON response, retain only final text and allowlisted
usage/model/ID/status metadata, discard reasoning/private headers, then adapt
to the native client's SSE envelope if needed. Unknown or exceeded usage remains
unknown/failure, not a reason to send a second request. Report published free
pricing separately from observed billing, which may be unknown.

P1 outcome (2026-11-01): the single approved attempt returned HTTP 429 and stopped.
See tasks for exact source-bound offline proof and actual evidence. Native 2.2.5
JSON-quotes the CLI literal; the gate accepts only that exact encoding or the
literal/text part and still forwards the fixed canonical body above. Omit the
synthetic `steps=1` setting, which inserts an assistant warning on the initial
request; the gate reservation, denied tools and process deadline bound this lab.
No general assistant-turn allowance was added. Driver completion is not model
success; 1/6 attempts consumed, five remain conditional and may not run
automatically. Do not retry 429, probe routes or reset the allocation.

## Safety prerequisite research and public CI observation (2026-11-01)

Scope: documentary evidence/public unauthenticated metadata only, per spec
§Approval. Question: does OpenRouter establish a matching pre-send accounting
contract and safe exact route for the requested candidate families, and can
existing public CI evidence narrow B7-F1 without a new PR or credentials?
Evidence threshold is an endpoint-specific contract, not local token counts,
catalog flags or a successful upload step. Runtime remains frozen 2.2.5; no
adapter/provider configuration was changed or executed.

Planning decisions after this research: the user selected exact
`openrouter/nvidia/nemotron-3-ultra-550b-a55b:free` first, proposed upstream NVIDIA,
synthetic-only, no paid fallback. Ling is an investigated alternative, not the
selected default. Later paid proposals must fit **$0.05/run and $0.25/month**;
these are planning ceilings, not spend or credential/cap-management authority.
The user retained **strict pre-send token guarantees** instead of a distinct
post-measured exception. Therefore **online work stays blocked** on matching
evidence; do not draft an exception as if selected, implement an adapter or
transfer the original S1 attempts. Sole decision record: spec §Approval.

### OpenRouter candidate metadata (not execution approval)

Public `GET https://openrouter.ai/api/v1/models` and model-specific `/endpoints`
metadata were inspected. Advertised USD **per million tokens**, not a run cap:

| Exact OpenRouter model ID | Advertised input / output | Relevant limitation |
| --- | --- | --- |
| `inclusionai/ling-3.0-flash` | $0.021 / $0.063 at `novita`; $0.06 / $0.18 at `deepinfra/bf16` | Both exact routes appear in the public ZDR list with `safety_retention: false`. Catalog says reasoning optional/default-on; tools/max_tokens advertised, not behaviorally tested. Unselected alternative, not enabled. |
| `nvidia/nemotron-3-ultra-550b-a55b:free` | $0 / $0 | Selected planning candidate, exact endpoint `nvidia`; absent from the inspected ZDR list. Provider policy says retention unknown/may train. Synthetic-only; no suffix removal or paid fallback. Endpoint metadata reported `status: -2`, not interpreted as a successful/available inference observation. |
| `nvidia/nemotron-3.5-lightning:free` | $0 / $0 (model catalog) | Another possible interpretation of the user's Nemotron family; absent from the ZDR list. Exact endpoint/capability validation not performed. Not automatically selected. |
| `z-ai/glm-5.3-flash` | From $0.0285 / $0.0923 (catalog/`relace/fp4` route) | Many providers/quantizations/prices; 34 exact-model ZDR entries observed, not permission to route to all of them. Catalog reasoning is mandatory/default max; a tiny visible-text cap is not a safe claim about total reasoning. Exact provider still requires a decision. |
| `qwen/qwen3.7-flash` | From $0.03 / $0.13 (model catalog) | Additional cheap candidate only; endpoint/privacy checks outstanding. No rolling `latest`, auto router or broad cheap-model allowlist is proposed. |

Catalog prices are time-sensitive routing offers, not verified paid eligibility,
key/account allowance, native tool correctness or a comparative-quality result.
In particular a private/ZDR endpoint cannot inherit the cheapest unrelated
route's price. Match exact model **and endpoint**, pricing and retention before
any later allocation; pin a provider rather than accepting default routing.

### Accounting and isolation verdict

Official [usage accounting](https://openrouter.ai/docs/cookbook/administration/usage-accounting)
documents native token/cost/cache/reasoning counts **after** completion. Context7
plus targeted official-document search found no authoritative pre-send hosted
chat/tool projection contract for these routes; this is an unresolved evidence
search, not proof that no such API could exist. Metadata says tokenizer `Other`
and instruct type null. **S0-F1 remains open**; an OpenRouter switch does not
satisfy original exact-model S1 or transfer its five remaining attempts.

[Routing](https://openrouter.ai/docs/guides/routing/provider-selection) defaults
`allow_fallbacks` true and `require_parameters` false; future reviewed payloads
would need exact provider restriction, disabled fallbacks, required parameter
support and appropriate data-policy/ZDR enforcement. `max_price` limits unit
prices, not total requests or total spend.
[Reasoning guidance](https://openrouter.ai/docs/guides/best-practices/reasoning-tokens)
states reasoning is billed output and counts against the output limit on **most**
providers; excluding returned reasoning does not make it free or disable it.
That wording is not an endpoint-specific universal hard-output guarantee.
[ZDR](https://openrouter.ai/docs/guides/features/zdr) is endpoint-specific and
does not cover plugins/server-tool backends or eliminate disclosed legal/safety
exceptions. [Provider logging](https://openrouter.ai/docs/guides/privacy/provider-logging)
lists NVIDIA as unknown retention/may train; do not relax privacy to obtain a
free result.

[Key-limit documentation](https://openrouter.ai/docs/api_reference/limits)
describes key limits/remaining allowance and effective budget layers. No
authenticated account/key endpoint was called, key opened or cap changed. Paid
planning still needs an explicit run/month ceiling and independently verified
dedicated provider-side cap/remaining budget, counting all in-flight work.
No key value belongs in conversation or Git.

**S0-F2 remains open beyond P1's narrow boundary.** The frozen P1 courier only
permits signed-out `opencode.ai` and transforms to fixed text; it cannot be
relabeled an authenticated OpenRouter/native-tool boundary. A future adapter
needs separate reviewed offline isolation, scoped secret delivery outside the
runtime, exact upstream payload/routing enforcement and independent reconciliation
of native sessions/usage. Current research authorizes neither its implementation
nor an online experiment. Freeze model/provider/quantization and all other
conditions within each future baseline/candidate comparison; mixed-model results
cannot establish instruction benefit.

### Existing public CI observation

Unauthenticated GitHub API reconfirmed push run
[`41083183856`](https://github.com/nawodyaishan/agentic-sdd/actions/runs/41083183856)
on `ee8bd0dde118edf05ef4f41410e24f87a108046e`: quality job/gate/upload steps
success. Artifact `12990182846`, `trusted-deterministic-checks`, 3,225 bytes,
not expired at observation; API-advertised digest
`sha256:c62d84829103251b457ccd9892c25383c69ea01c15ecb0158043fd75fa921463`.
This is listing metadata, **not a downloaded-content digest or content review**.
One unauthenticated archive GET returned **HTTP 401**; stopped without using
credentials. `event=pull_request_target` run listing returned total_count **0**.
Older `pull_request` evidence predates this workflow and is not a substitute.
**B7-F1 remains open** for PR-path/content evidence; artifact existence/upload
success are now independently observed. No PR/push/branch rule or service changed.

Subsequently the user supplied the actual ZIP at `/home/nawo-ubuntu1/github/scp`.
Its SHA-256 matches the GitHub upload digest; all nine members were inspected,
the embedded/separate check results agree, six exit-zero check statuses and six
log digests match. The two revisions are the same push commit, so this closes the
**artifact-content observation**, not the separate PR/fork-path or remote-policy
gaps. See tasks for exact evidence and scope. No user file was changed.

Raw public API observations are in owner-private
`/tmp/opencode/sdd-safety-research.owgkkz1x/`; tasks links their identities.
Official prose above was inspected through Context7/Exa extraction; do not claim
raw HTML hashes for it. No inference/account probe, dependency/model download,
model child, fixture execution or code change occurred. Subsequent planning
answers above retain strict blocking; no automatic next run. Missing inputs are
matching authoritative token evidence and separately supplied CI content/PR-path
evidence, not another unchanged model/budget confirmation.

## Specialist and tool assignments

Use one main agent; no delegation is authorized. These skills are present in the current catalog; recheck availability when implementing. Load only the relevant guidance, not all entries together.

| Work | Principal guidance | Reason |
| --- | --- | --- |
| Native adapter/preflight/recovery | `opencode` | Version-sensitive V2 configuration and native lifecycle. Loaded during this design. |
| Canonical workflow skill changes | `skill-creator` | Skill structure, concise references and behavioral evaluation. |
| Go check runner and compatibility tests | `golang-pro` | Standard-library subprocess/error/testing patterns; repository Go constraints control. |
| Deterministic CI trust/isolation | `cicd-automation-workflow-automate` | Trusted gate, permissions and candidate execution separation. |

`agentic-sdd-architecture-review` supplies consequential-boundary scrutiny in the main agent, not independent review. For plain inventory/pointer updates, no additional specialist is needed. Runtime execution isolation may load `docker-expert` only when actually designing the container boundary.

Conditional tools: local Read/Grep/Glob and shell for discovery/checks (CodeGraph reported this project unavailable despite a local directory; use local discovery until an index is available); Context7 plus official V2 pages for concrete runtime questions; Exa only for gaps beyond official docs. Browser automation is not needed for the initial product, but a browser write route, if ever enabled, needs its own boundary test. Resolve actual tool IDs at execution. Do not install tools, switch models or enable integrations from assignment labels.

## Design evidence and architecture self-review

On 2026-10-28, Context7 `/websites/opencode_ai/v2` policy guidance and official V2 [config](https://opencode.ai/v2/docs/config), [permissions](https://opencode.ai/v2/docs/permissions), [policies](https://opencode.ai/v2/docs/policies), [agents](https://opencode.ai/v2/docs/agents), and [commands](https://opencode.ai/v2/docs/commands) were consulted. Current policy docs control over an indexed Console-precedence snippet that differed from the fetched page. Revalidate against B1's pinned release.

Architecture self-review: **Ready for combined design review**, not runtime readiness or human approval. Findings addressed in this draft:

- AR-01: candidate-controlled verification can erase its own gate → C3 uses a trusted base contract, detects weakening and surfaces bootstrap trust.
- AR-02: command shell blocks bypass agent permissions → C2 prohibits them and tests config/command collisions.
- AR-03: skill-only backup manifests cannot safely represent config merges → C2 uses a separate opt-in, digest-checked manual config adoption procedure.
- AR-04: host permissions and read/search tools do not constitute isolation → C2 requires clean isolated negative fixtures and disables unproven review routes.
- AR-05: a baseline collected after skill tuning is invalid → C6 freezes inputs and requires observations or an explicit blocker before tuning.

Remaining execution dependencies: exact runtime/access, provider authorization, available isolation, trusted CI bootstrap and baseline observations. B1 resolves or records them. A discovery that changes contracts, permissions, requirements or batch scope returns the affected documents for human review; it does not silently expand this design.

## Rollout and verification

Develop only repository artifacts and disposable fixtures after approval and batch authorization. Test configuration in isolated locations before optional adoption. Keep the original skills/config revisions for comparison/rollback. Do not apply to the real home, publish, spend on a benchmark or change remote enforcement as part of development authorization.

For Go changes run `make test`, `make vet` and `git diff --check`; for the runner/CI transition also prove required-contract execution and old target compatibility. For canonical skill changes check discovery, sibling links and observed behavior; structural tests are not compliance evidence. Run container e2e where required; absence of Docker is an explicit blocker for its claimed acceptance, not a pass.

Next: [tasks.md](tasks.md) assigns the contracts and ACs to ordered batches and verification owners. No implementation is authorized by this plan.
