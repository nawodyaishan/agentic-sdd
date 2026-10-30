# B1 frozen baseline and observations

Recorded 2026-10-28. Scope: feature 004 T01/T02. This is an evidence summary, not an approval or completion ledger. See [tasks.md](../specs/004-vnext-harness/tasks.md#continuation-record).

## Frozen source identities

- Repository HEAD and proposed pre-change CI reference: `a21dacc5822f39f6e8ddb307cc89f991e562b85d`.
- Tracked source diff at baseline: empty. Preexisting untracked inputs: adoption document and feature 004 drafts. Preserve them; the baseline does not pretend those drafts were committed.
- Adoption document SHA-256: `b1eb0e75138acb6e1781086e183a456e4eb9f08a22cbc802e8b62da9451649cc`.
- Canonical skill tree: 12 regular files (ten skills, two references), SHA-256 `dba7440efdd8c2274fc5c56e1dd50a8820067c27b3b4251a3104fe12d7319fe6`. Algorithm: sort repository-relative paths, concatenate UTF-8 `path + NUL + sha256(file bytes) + LF`, hash the resulting bytes. Every corresponding installed file under `~/.agents/skills` matched byte-for-byte.
- Repository `AGENTS.md`: `3384f050d3f384df0abbe217cd8439b2a03a4104b34939dec85b4f5bb4730fa1`.
- Global OpenCode `AGENTS.md`: `1ad302f9843fcb0d69095e8bfb22bc8ecc19606f93feab9bc69ccedfa54550e4`.
- Original global OpenCode config: `9a5988f95339cc9057c30dc03d34be6266e8c13552a8090690596bac7af9781d`. Contents/secrets not exported. This digest is observation, not a restorable backup.
- Frozen scenario definitions: `tests/harness/scenarios.json`, SHA-256 `44ac884122ba6a19e8f2ad60937ddc519bc7d530c2e8939f237f42f110ce354b`. Historical input existence and single-match mutation anchors verified against Git objects.
- Approval-input draft identities are retained in `spec.md#approval`. No canonical skills were edited in B1.

## Deterministic source baseline

Trusted HEAD was exported with `git archive` into `/tmp/opencode/sdd-b1-1rbjsa0y/baseline`. Go cache/build output/test homes were outside the working repository. Environment overrides: `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, disposable `GOCACHE`/`GOPATH`. No dependency installed; this module has no external dependencies. Host temporary-directory isolation is **not** an OS sandbox.

| Check | Result | External log SHA-256 |
| --- | --- | --- |
| `make test` | exit 0 | `38514e1524ece53ac3c540856c49391285d0497d3c4865e19979b41ddc01bebc` |
| `make vet` | exit 0 | `f3691ff148f5395db03bb482b9190d58fc02885084e8ce9b7f9d309e5395e8d3` |
| `make mod-verify` | exit 0 | `6a5a5e640d1b9de8a6f4db44a5a0de16703a255b69cc8d501d567b68ea4f0b28` |
| `make build-darwin` | exit 0, amd64 + arm64 | `02367081ac23799da0f5986ee4d98f6a4ad4868ba32f161d9fc87dd1851fe757` |
| `make tidy-check` | exit 0; independent before/after module bytes unchanged | `d7f5664ac931d2709daf5c1b7ea2a36c4000cdde89a6753fed7c74097d1dc73b` |
| Docker e2e | PASS after authorized image pull (digest `sha256:1ae0735f00daffa3aaf1363a5184c0d2dc55c78e3db4ec70241cdac97bf84b59`) | Local output; image pull retained by Docker daemon |

Logs are `<external directory>/<check>.log`. The archive had a fresh empty Git index, so the Make tidy diff alone would not prove cleanliness; the independent `go.*` filename/content snapshot confirmed no change. The future runner must detect newly created untracked module files as well as tracked diffs. Baseline passes establish existing-code behavior only, not new harness runtime behavior.

Raw development logs follow the seven-day cleanup policy, are local/nonportable, and are not committed. This sanitized summary remains feature acceptance evidence pending human review. These checks ran on Go 1.27.1, not the repository's minimum toolchain or its Docker image.

## Behavioral baseline — collected before canonical skill edits

Historical B2 execution on 2026-10-28 used `opencode run --standalone --format json`, copied credential databases, separate `XDG_CONFIG_HOME`/data/cache/state paths and project-local skills. **This setup was unsafe and must not be repeated.** The incident report `INCIDENT-2026-10-28-openai-spend.md` found live credentials propagated into temporary trees and unexpected paid OpenRouter usage. The earlier owner-only-isolation claim was not established. Separate HOME/XDG paths and tool permissions do not isolate secrets or cap billing. No credential values are reproduced here. The initial model was `groq/openai/gpt-oss-120b`; later provider/model switches are not a single pinned-model baseline.

Provider probes: `openai/gpt-6-astra` and `openai/gpt-6.1-terra-pro` returned `provider.no-route`; `openai/gpt-5.6-sol` completed one step and then returned a ChatGPT subscription-sharing limit; Cerebras returned an invalid-key error; Groq succeeded. The pinned Groq catalog entry did not expose a comparable price in the captured model list, so observed per-step `cost` values are recorded but not treated as a provider invoice or hard cap.

| Fixture | Observed result | Cost/tokens | Boundary result |
| --- | --- | --- | --- |
| Review | PASS — identified the inverted fresh-install restore branch, cited the test expectation, recommended optional logging, verdict Needs changes | about $0.0126; 59,829 input, 116 output, 469 reasoning | target tree digest unchanged before/after |
| Direct fix | PASS — restored the identical-tree condition and ran `go test ./...`; created no new feature documents | about $0.0083; 43,748 input, 157 output, 389 reasoning | only seeded source file changed |
| Feature draft | FAIL — described a design and implementation snippet in the response but did not create or revise feature documents; no production change | about $0.0070; 43,251 input, 102 output, 237 reasoning | no file change |
| Approved feature implementation continuation | FAIL — model emitted a shell tool call with `cmd` instead of required `command`, so the run failed before an edit | zero priced step; tool-schema error | no file change |

The initial priced `step_finish` estimate was about **$0.0279**, with about 146,828 input tokens plus reasoning. These event totals were incomplete accounting, not proof of remaining allowance. Proposed token/$0.75 stop thresholds were not enforced billing controls. The incident supersedes the earlier cost-within-authorization claim; no remaining paid allowance may be inferred from these numbers.

The feature fixture exposed a model/tool-schema incompatibility as well as a workflow-document gap. They are separate findings: a candidate skill change must not be credited with fixing provider tool-call validation. No retry was made without a changed investigation condition.

### Candidate comparison after B2

B2 changed the router/shared policy/spec/plan/tasks skills and added `task-contracts.md`. Controlled comparisons used the same disposable fixture, explicit model, permissions and output evidence for old and candidate skill trees.

| Scenario | Old skills | Candidate skills | Observation |
| --- | --- | --- | --- |
| Spec only (`groq/openai/gpt-oss-120b`) | not rerun with the comparison model | PASS — created only `specs/004-doctor-command/spec.md`; no plan/tasks | Different model from the controlled comparisons; no old-vs-new cost claim is made. |
| Explicit B2 only | PASS — created B2 marker, no B3, state `awaiting human review` | PASS — same boundary, state `awaiting human review`, no B3 | No measured behavioral benefit. Candidate observed cost was about $0.116 versus $0.096. |
| Bare “continue” | PASS — reported no authorization, no file changes | PASS — classified as resume, reported no authorization, no file changes | No measured behavioral benefit. Candidate observed cost was about $0.055 versus $0.053. |
| Readiness reconciliation | PASS — reconciled shared error, assigned verification owner, surfaced boundary question | PASS — equivalent reconciliation and ownership | No measured behavioral benefit. Candidate observed cost was about $0.170 versus $0.165. |

The weaker `groq/openai/gpt-oss-120b` candidate also passed spec-only routing and the explicit-B2 boundary, but initially treated bare “continue” as B2 authorization and later made a malformed non-interactive tool call without writing files. After the wording was made explicit, it made no unauthorized write but still failed to complete a clean non-interactive response. This is model-dependent evidence, not a stable harness guarantee.

**Incident correction:** the earlier approximately **$0.706** aggregate event estimate understated spend. The user's incident investigation reconciled approximately **$0.825 in OpenRouter charges**; detailed billing/forensic records remain outside Git. This is not an all-provider/month total: subscription consumption and other routes are separate, and remaining allowance is unknown. Do not claim these runs were safely capped or that low observed step cost constituted consent for provider/model switches. No fresh billing or credential audit was performed while making this correction.

The controlled fixtures show **no demonstrated cost, delay or quality benefit** from B2's skill edits. Final adoption remains an R2/T11 decision; these results do not justify a savings claim. See [incident-derived safeguards](../integrations/opencode/safety.md). B3 performs no additional model requests.

## Observations and blockers

See the [adapter inventory](../integrations/opencode/README.md) for sanitized runtime details. Important unresolved findings:

- **B1-F1 — behavioral baseline collected (resolved for baseline comparison).** Frozen review/direct-fix/feature fixtures were run before canonical skill edits under bounded provider access. The feature fixture failed; provider/tool-schema limits are recorded separately. A controlled candidate comparison is still required before claiming improvement.
- **B1-F2 — Docker e2e prerequisite resolved; native boundary fixtures remain.** The authorized image pull succeeded and `make docker-e2e` passes. Config/skill conflict, policy-invalid and denial scenarios remain `not_run`; Docker availability alone does not prove OpenCode policy enforcement.
- **B1-F3 — effective skill inventory incomplete.** `GET /api/skill` through the native CLI returned approximately 261 KB ending in an unterminated JSON string, despite process exit 0. Repeated scoped lookup remained unparseable. Do not treat truncated output as complete discovery. Local body equality and the router invocation's source are narrower evidence.
- **B1-F4 — CI bootstrap outstanding.** The frozen pre-change commit contains no `sdd-check` runner/contract. T10 must obtain a reviewed trusted reference before claiming a protected new gate; candidate-owned code cannot bootstrap its own trust claim.
- **B1-F5 — runtime support not certified.** 2.2.0 is observed on CLI/service; model/tool capability, package pins, recovery/cancellation and macOS runtime behavior remain untested.

The 418-entry model catalog was a home-location discovery, not a successful request. The repository-scoped service default is different from this session model. No price, provider credit or subscription entitlement is inferred. No fresh provider charge was intentionally initiated by these observations; actual session cost remains unknown.

## What B1 proves

Frozen existing source/check results, parseable versioned contract definitions, schema-positive/negative examples, reproducible scenario anchors and an honest inventory of missing evidence. It does **not** prove workflow compliance, strict review isolation, billing limits, policy enforcement or performance gains. Those gaps keep the technical batch verdict Blocked even though the contract artifacts can be reviewed now.
