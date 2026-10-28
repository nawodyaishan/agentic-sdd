# Portable harness contract — version 1

Status: B1 contract and baseline artifacts. The runner, native profiles, behavioral validation and CI integration are not implemented here. Approval and execution state live in [feature 004](../specs/004-vnext-harness/spec.md#approval) and its [tasks](../specs/004-vnext-harness/tasks.md), not in receipts.

## Authority and modes

Run entry records mode (`draft`, `build`, `direct-fix`, `review`, `resume`), workspace, request/authorization references, feature/batch when applicable, allowed effects, actual model/profile/tools, selected checks and effort limits. Resume reconciles current authorization and current files before resolving an action. Missing authorization blocks effects; a summary or receipt never supplies consent.

Direct fixes have null feature/batch and no task IDs. Review never changes project state; receipts/logs go outside the target repository. Draft permits only agreed documents. Build permits one authorized batch. Process/tool permission is distinct from workflow authorization. Required runtime boundaries not demonstrated by the adapter remain blocked.

## Check contract

[`checks.json`](checks.json) describes current repository commands; [`checks.schema.json`](checks.schema.json) defines their shape. The contract is executable data: choose it from a trusted revision, not an untrusted candidate. B1 does not add an executor or change Make/CI.

- `version` is exactly 1. Reject unknown fields/versions and duplicate check IDs.
- `cwd` is `.` or a repository-relative directory. Resolve symlinks and reject paths escaping the selected workspace before execution.
- Each check runs `steps` sequentially. `argv` is passed directly to the executable, never through an implicit shell. `env` contains explicit nonsecret overrides only; never interpolate input into shell text.
- `timeout_seconds` covers the whole check, including all steps. The run deadline may shorten it. Stop scheduling after a required failure; mark remaining checks `not_run` with a reason.
- A `baseline`/`required` run selects every entry tagged with that scope. A `focused` run requires explicit IDs. A focused result cannot establish required completion. Container checks are focused, prerequisite-dependent acceptance evidence, not silently skipped required checks.
- `prerequisites` names actual executables or capabilities. Missing prerequisites produce blocked, not failed or passed. Never install/download a missing prerequisite automatically.
- `effects` names filesystem writes and network possibilities; it is a declaration, not a sandbox. Go toolchain/dependency downloads must be disabled or separately permitted in the execution context. Build/test code can have effects beyond cache paths: execute untrusted code in isolation.
- `evidence` is a relative log filename beneath the run's external output directory. Reject traversal, absolute paths and symlink escapes. Keep logs owner-only; retain at most 10 MiB per check and record truncation. Truncated output cannot prove an assertion whose evidence is missing.
- `expected_exit` is zero. A later step is not run if an earlier step fails; recording only the last step's success is invalid.

Local recipes preserve the existing Make targets. The required entries also include direct equivalent commands so trusted CI need not run a candidate-weakened Makefile. Neither a trusted runner nor a schema proves candidate test adequacy; changed tests and gate definitions require review. The pre-change reference is recorded in the baseline, but contains no new runner: trusted-gate bootstrap remains a later prerequisite.

## Result contract

Execution and review are separate:

| Execution | Meaning |
| --- | --- |
| `completed` | Requested execution ended; not a claim of technical Ready. |
| `incomplete` | Deadline/effort/evidence gap prevented completion. |
| `blocked` | Required authority, capability or prerequisite unavailable. |
| `cancelled` | Cancellation observed; uncertain remaining effects stay explicit. |
| `failed` | Executed operation failed. |

Check results use `passed`, `failed`, `blocked`, `incomplete`, `cancelled`, `not_run`. An executed nonzero check fails; a missing executable blocks; a timeout is incomplete; interruption is cancelled; later skipped checks are not_run. Zero process exit alone does not establish acceptance. Proposed runner exit codes remain 0 (all selected checks pass), 1 (failure/incomplete/cancelled), 2 (invalid input/blocked prerequisites).

Technical review is null until performed, otherwise `Ready`, `Needs changes` or `Blocked`. Ready requires fresh required evidence plus actual requirements/engineering review. A stopped batch can be `awaiting human review` with a Blocked verdict. Implementation owns one completion review and the authorized state write. Review-only changes nothing in the project.

## Receipts and semantic validation

[`evidence/receipt.schema.json`](evidence/receipt.schema.json) and synthetic [fixtures](../tests/harness/contracts.json) define the portable observation shape. Schemas check syntax and bounded enums, not truth or authorization. The future consumer must additionally enforce:

1. Unique check/task/finding IDs; check IDs belong to the selected trusted contract, and required coverage is complete before a required pass claim.
2. Null observations have nonempty explanations in `unknowns` keyed by JSON Pointer. Direct-fix null feature/batch/workflow state, absent review verdict, and not-executed check exit code/start time are structural absence, not unknown measurements. `workflow_state` observes canonical tasks; it never owns that state.
3. Passed checks have exit 0, start/end timestamps, and evidence; nonexecuted results have null exit. End time cannot precede start. Failed, timeout, cancelled or missing evidence cannot set Ready.
4. Evidence identity covers base commit, staged changes, unstaged changes, relevant untracked files and deletions. Hash sorted records of path, kind and content digest; record scope and algorithm. Never treat HEAD alone as the diff identity. Unknown identity disallows reuse.
5. Freshness compares relevant code, config, dependency, input and environment identities, including the harness and trusted contract revisions. Any affected change invalidates reuse/Ready; post-format identity is the checked identity. Tidy checks must additionally compare pre/post module filenames/content so newly created untracked `go.sum` cannot escape a tracked-only diff.
6. Authorization references point to actual user decisions and the one canonical approval. They are observations, not an approval field. Missing authority blocks the relevant operation even if a schema-valid string is present.
7. Logs and receipts reside outside the target checkout for review. Canonicalized paths must remain in the authorized evidence root; no secret, hidden reasoning, credential or complete environment dump is exported.

Use native IDs/exports where adequate; absent values stay null with reasons. Model/catalog availability is not proof of billing access, compatibility or selected session model. Costs include failed/retried work; absent cost is never zero. Default limits: one session, no children, 900 seconds, 20 steps, two transient retries; no blind retry for non-idempotent actions. Additional paid work is disabled without separate authorization and known allowance within US$10/month.

Retention: external owner-only raw logs 7 days, sanitized receipts 30 days, manual cleanup. Explicitly retained acceptance evidence records an owner/expiry. Do not prune native service databases or snapshots. Committed examples are synthetic; baseline summaries are intentionally sanitized.

## Baseline and verification

- [Frozen baseline](baseline.md): source identities, deterministic results and open blockers.
- [OpenCode observed version](../integrations/opencode/supported-version.md): inspected release, not a support certification.
- [Scenario definitions](../tests/harness/scenarios.json): independently expected behavior, with no simulated pass claims.
- Schema checks on a host with Python 3 and `jsonschema` 4.x already available: `python3 tests/harness/validate_contracts.py`. This is development validation only; no production Python runtime or package installation is required. Runtime/semantic checks belong to T07/T08 and native boundary checks to T05/T09.

Before B2, collect the missing frozen-baseline behavioral observations under explicitly permitted provider access. Do not modify the canonical skills and then reconstruct a supposed before-result.
