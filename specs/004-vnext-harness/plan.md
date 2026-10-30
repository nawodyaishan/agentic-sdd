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
