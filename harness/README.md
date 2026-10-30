# Portable harness contract — version 1

Status: B4 adds a deterministic Go check runner; B6 adds optional offline
receipt/freshness/redaction validators and recovery fixtures. B3's native adapter
templates remain inactive and acceptance-blocked. B7 adds a [trusted CI adapter](trusted-ci.md)
and anti-weakening fixtures; a hosted main-push run has passed, while the fork/PR
entry and remote required-check policy remain unobserved.
Native correlation, agent recovery and connected behavioral validation remain unproven.
Approval and execution state live in [feature 004](../specs/004-vnext-harness/spec.md#approval)
and its [tasks](../specs/004-vnext-harness/tasks.md), not in results/receipts.

## Authority and modes

Run entry records mode (`draft`, `build`, `direct-fix`, `review`, `resume`), workspace, request/authorization references, feature/batch when applicable, allowed effects, actual model/profile/tools, selected checks and effort limits. Resume reconciles current authorization and current files before resolving an action. Missing authorization blocks effects; a summary or receipt never supplies consent.

Direct fixes have null feature/batch and no task IDs. Review never changes project state; receipts/logs go outside the target repository. Draft permits only agreed documents. Build permits one authorized batch. Process/tool permission is distinct from workflow authorization. Required runtime boundaries not demonstrated by the adapter remain blocked.

## Check contract

[`checks.json`](checks.json) describes current repository commands; [`checks.schema.json`](checks.schema.json) defines their shape. The contract is executable data: choose it from a trusted revision, not an untrusted candidate. B4 implements local execution; [B7's adapter](trusted-ci.md) adds independent-base execution, with CI adoption still explicit.

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

Check results use `passed`, `failed`, `blocked`, `incomplete`, `cancelled`, `not_run`. An executed nonzero check fails; a missing executable blocks; a timeout is incomplete; interruption is cancelled; later skipped checks are not_run. Zero process exit alone does not establish acceptance. Runner exit codes are 0 (all selected checks pass), 1 (failure/incomplete/cancelled), 2 (invalid input/blocked prerequisites).

Technical review is null until performed, otherwise `Ready`, `Needs changes` or `Blocked`. Ready requires fresh required evidence plus actual requirements/engineering review. A stopped batch can be `awaiting human review` with a Blocked verdict. Implementation owns one completion review and the authorized state write. Review-only changes nothing in the project.

## Receipts and semantic validation

[`evidence/receipt.schema.json`](evidence/receipt.schema.json) and synthetic [fixtures](../tests/harness/contracts.json) define the portable observation shape. Schemas check syntax and bounded enums, not truth or authorization. The [offline reference consumer](evidence/README.md) now exercises the following consistency rules; native correlation and trusted provenance remain unverified:

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

Historical B1/B2 observations and their spending-incident correction are in the
baseline record. Do not reconstruct a before-result after tuning or repeat the
credential-copy/provider-switch procedure. Future comparisons require their own
safely bounded authorization and unchanged model/fixture conditions.

## B4 runner usage and limitations

```sh
go run ./cmd/sdd-check --scope required --workspace . \
  --contract harness/checks.json --output /absolute/external/new-run
go run ./cmd/sdd-check --scope focused --check vet --check test \
  --workspace . --contract harness/checks.json --output /absolute/external/another-run
make verify SDD_OUTPUT=/absolute/external/required-run
make tidy-check SDD_OUTPUT=/absolute/external/tidy-run
```

`--scope` and `--output` are required. `--workspace` defaults to `.`;
`--contract` defaults to `harness/checks.json`, resolved from the invocation
directory, not from a different target workspace. Choose a trusted contract
explicitly when checking an untrusted candidate. `--check` repeats only for
focused scope; baseline/required select every eligible check in contract order.
`--deadline` is a positive Go duration (default `15m`) covering the run; each
check also has its own positive whole-check timeout, not a fresh timeout per
step. Deadline stops new work; bounded termination/evidence cleanup may follow.
The runner stops scheduling after any selected non-pass and marks later
checks `not_run`. No automatic retry, install, model call or provider fallback
is implemented. Arbitrary trusted argv can still execute arbitrary programs:
review the contract and isolate candidate code rather than treating this CLI
as a billing/permission firewall.

Output must be new or empty and owner-only, must not overlap the checkout,
and must have no symlink components (on macOS use canonical physical paths).
Log paths are validated relative paths; output is reserved without overwriting
existing evidence. Full command output is drained but only the first bounded
10 MiB/check is retained, with a truncation marker and `truncated: true`.
Truncation or an output-pipe/evidence failure makes an otherwise passing check
`incomplete`. Log names may be nested but cannot collide with another log or
the reserved `result.json` path. The contract must be regular JSON, at most
1 MiB, with known fields/version, unique IDs, non-null fields and valid ordered
steps/environment overrides. Relative PATH entries are not used for executable
lookup. No implicit shell expansion occurs; a deliberately configured `sh -c`
remains executable code, not a safe data value.

`result.json` and stdout contain the same structured execution result: version,
canonical workspace, SHA-256 of the exact loaded contract, selection scope/IDs,
run/check timestamps, execution/status/exit, evidence paths, truncation and
reasons. It is **not** T08's full observation receipt or a freshness proof, and
contains no review verdict or authority/task-state write. Flag/input errors can
occur before any safe output directory exists and are reported to the caller
without creating evidence. Compiled CLI exits: 0 for all selected passes, 1 for
failure/incomplete/cancelled execution or result-write failure, 2 for invalid
input/blocked prerequisites. `go run`/Make may wrap a nonzero executable exit;
inspect the structured result for the authoritative execution distinction.

Executables are looked up without installing anything. The two current named
capabilities are `running Docker daemon` and `preloaded golang:1.25-alpine image`;
they use bounded metadata probes. Unknown capability phrases block. The Docker
recipe also uses `--pull=never`, closing the race after image preflight. The
reserved `tidy-check` ID snapshots regular `go.mod`/`go.sum` files before and
after all steps, including absence/new files, without undoing any change. This
replaces its old staged-index-dependent Git diff; using a different ID would
lose the invariant and is a contract change requiring review. CI protection
against candidate contract/runner weakening is still T10, not this local gate.

Linux and Darwin have process-group implementations; only Linux execution is
enabled. Darwin is cross-compiled but deliberately returns blocked until native
cancellation tests are observed; other hosts also block before execution.
Each enabled command owns a new process group;
on cancellation/deadline it receives TERM then bounded KILL, and direct wait /
output pipes are bounded. A leader exiting with remaining group activity cannot
pass. Remaining membership (including possible zombies) is reported as uncertain.
Children deliberately escaping the group, external effects and a stalled
filesystem are outside this mechanism; use an isolated execution boundary for
untrusted/daemonizing commands. It never kills unrelated process groups. No
concurrent untrusted filesystem writer is permitted: Go 1.23 path validation
is not a race-proof sandbox. Native OpenCode cancellation is separate T09 work.

Make creates private temporary output when `SDD_OUTPUT` is omitted; set
`TMPDIR` to an approved external location to choose its parent. Keep raw logs
7 days by default and clean up manually, retaining explicitly accepted evidence
with an owner/expiry. Do not commit logs or copy native credential databases.
Go downloads are disabled at the Make runner entry points, but candidate code
can still use the host network/credentials unless OS-isolated. Pending account
caps/key rotation do not prevent these provider-free deterministic checks and
remain separate from authorization to run any LLM. The user's B6 remediation
report and its verification limits are in [spec §Approval](../specs/004-vnext-harness/spec.md#approval).
