# Harness fixtures

## Supported-core Go selection

`task test` / `task vet` and the `test-core` / `vet-core` aliases select the same
ten supported-core packages: `.`, `./internal/version`, `./internal/skillsync`,
`./internal/harnesscheck`, `./internal/harnessevidence`, `./internal/appops`,
`./internal/buildinfo`, `./internal/fleet`, `./cmd/agentic-sdd`, and `./cmd/sdd-check`. The aliases use
`tests/supported-core-go.sh`, whose required-directory checks fail closed.
Feature 013 D03 authorized adding the shared metadata package, B14/T41 with D07
authorizes adding the build-identity package, and Feature 014 B5/T10 (approved
2026-11-11) adds `internal/fleet`, each without removing earlier coverage.
`internal/opencodeadapter`'s experimental tests and the CLI's
`opencode_test.go` are not selected; no 006 native/profile/dispatch/pilot or 007
historical migration/adoption suite is selected. Shipped-reference/package and
current core check/evidence tests stay covered regardless of their feature origin.

The CLI is selected as a package with its normal build constraints, not an explicit
file list. This profile has Linux evidence only, not a native macOS guarantee.
It clears inherited test filters/foreign workspaces
and disables toolchain/module retrieval; it is not an OS sandbox.

Verify the selection with
`python3 -B -m unittest discover -s tests/harness -p test_supported_core_go.py`.
It uses fake Go in disposable fixtures to check exact argv/environment and failure
propagation, not deprecated-feature or native behavior. Python 3.9+ and Go Task must already be
available for the entry-point checks; absence must not be counted as a pass.

**Not a historical/experimental aggregate gate:** `harness/checks.json` required
test/vet commands now use the same explicit core package set. Updating this
candidate contract does not adopt it into a trusted base: old reviewed definitions
must continue rejecting unadopted changes. Live trusted-base adoption/push needs
separate review and authorization. Report that gate not_run/blocked until its
prerequisites are satisfied; never disable comparison or reinterpret local
package tests as trusted-CI evidence.

## Core workflow text and onboarding checks

Run `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py`
with Python 3's standard library. This explicit selection reads canonical portable
instructions and `core-workflow-scenarios.json`; prompts are data, never executed.
It checks required clauses, source-link integrity, scenario completeness, honest
`agent_observation: not_run` labels and compact onboarding/security-reporting guidance.
In-memory clause removal, authority softening and missing-link/fixture mutations
must be detected. These are **static text/fixture checks**, not model behavior or
runtime enforcement. No client, provider, container, network or subprocess is launched.
Do not substitute blanket discovery or import deprecated integration suites.

## Historical comparison fixtures

`scenarios.json` freezes model-behavior prompts, source revisions, exact one-match mutations and independent expected outcomes. Apply mutations **only in disposable isolated copies**, after verifying the base revision and match count. Never edit the live repository to construct a fixture. Later B2 observations and the spend-incident correction live in `harness/baseline.md`; B3 adds no model requests.

The historical feature fixture starts after the restore core and before its CLI integration. The regression fixture disables the identical-tree fast path. The review fixture reverses the already-absent guard for removing a freshly installed skill. Independent sentinels/backup counts and before/after tree identities are the oracles, not the implementation's own return values.

The legacy restore variant is reserved from tuning. It is not genuinely unseen to this author; an evaluator must supply a hidden concrete case before reporting unseen-case coverage. No new agent is authorized to do that automatically.

`contracts.json` separates schema assertions from future semantic/runtime assertions. Run `python3 tests/harness/validate_contracts.py` with the already available Python `jsonschema` 4.x package. It validates both schemas/examples and rejects malformed versions, paths, timeout, argv, costs, direct-fix state and timestamps. **It does not implement T07/T08's runner or semantic checks**, and passing it does not prove agent compliance, policy denial or runtime cancellation.

Behavioral runs require a frozen model/variant, permissions/tools, isolated execution context, budget and actual authorization. The scenario definitions themselves grant none. Reserve two repetitions for critical boundary/recovery cases when authorized, record failed attempts too, and report sample counts rather than reliability claims.

## B3 credential-free adapter checks

Run `python3 -B -m unittest discover -s tests/harness -p test_opencode.py`
for template/model-catalog invariants and the synthetic manual adoption exercise.
Run `sh tests/harness/native-opencode.sh` for the pinned 2.2.0 native API probes
inside the unprivileged, network-disabled container. No inference endpoints,
provider credentials or host home/socket mounts are used; no model test launcher
is delivered. The script refuses unavailable image/version prerequisites rather
than pulling/upgrading. See [adapter verification](../../integrations/opencode/verification.md)
for exactly what passes versus remains blocked. Do not reinterpret permission
API results or static model IDs as full agent-tool/selection/privacy guarantees.

## B4 deterministic runner tests

`go test ./internal/harnesscheck ./cmd/sdd-check` executes small synthetic
subprocess fixtures only, never the complete required runner from its own test
suite. Cases cover contract/selection rejection, literal argv/environment lookup,
safe external/private evidence, real exit failure, missing prerequisites,
skipped later work, truncation, missing/deleted evidence, whole-check/run deadlines,
cancellation, new untracked module files and refusal to overwrite old results.
Linux-specific tests exercise ordinary owned descendants, leader exit with live
children, and survival of an unrelated process. `go test -race` is a useful
additional check. macOS runtime cancellation remains unobserved; cross-builds
are not native execution evidence. Full receipt/freshness/redaction acceptance
still belongs to T08, and protected CI bootstrap to T10.

## B5 diagnosis and review evidence

`diagnosis-scenarios.json` records eight acceptance scenarios and the actions an
actual agent run would need to demonstrate. `agent_observation: not_run` means
exactly that: the scenario definition or a static phrase/link check is not an
observed compliance pass. No extra model session, benchmark, trigger optimizer
or delegated reviewer is started by these fixtures.

`sh tests/harness/diagnosis-fixtures.sh` exercises **executable fault oracles**
against the pinned B4 Git tree. It requires the already local pinned Python
container image, host Go 1.27.1/linux/amd64 toolchain, Git and Docker. It never
pulls/installs prerequisites. The archived source and toolchain are read-only
mounts; only the freshly created private evidence directory and container
scratch are writable. There is no host home, provider credential or Docker
socket mount and no container network. `/tmp` is explicitly executable so Go
can run compiled probes, while retaining `nosuid,nodev`, non-root UID, dropped
capabilities and no-new-privileges. Calls have a 120-second maximum and a
600-second total fixture deadline, with no automatic retries.

The probes distinguish a tautological assertion from an independent contract,
setup failure from a real no-op regression, bad reviewer advice from restore
round-trip behavior, a locally passing correction from dependent replacement
failure, and passing core behavior from broken CLI integration. Expected
failures are retained with actual exit/output identities; they are not reported
as passing production checks. All source mutations and simulated applies use
container scratch/test homes, never the real checkout or installed client.
Packaging's actual eleven-skill install/preview/no-op/restore tests live in
`internal/skillsync/package_test.go` for both embedded and on-disk sources.

This fixture driver does **not** evaluate an LLM, intermittent-root-cause claims,
read-only agent mutation behavior, skill triggering or review-invocation counts.
Those behavioral observations and an old-versus-new comparison remain blocked
pending a separately safe controlled run; no reliability/savings claim is made.
Keep raw fixture logs under owner-only access for 7 days by default and clean
up manually. Do not copy primary auth stores or enable a model to complete this
deterministic exercise.

## B6 offline receipts and recovery

The existing optional `jsonschema` development dependency, Python 3.9+ (tested
on host Python 3.14), Git, local Go toolchain and `sh` must already be available.
No tool/package installation, provider access or native session is needed.

```sh
TMPDIR=/tmp/opencode python3 -B -m unittest discover -s tests/harness -p test_receipts.py
TMPDIR=/tmp/opencode python3 -B -m unittest discover -s tests/harness -p test_recovery.py
TMPDIR=/tmp/opencode GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go run ./cmd/sdd-check \
  --scope focused --check receipt-validation --check recovery-fixtures \
  --workspace . --contract tests/harness/offline-b6-checks.json \
  --output /tmp/opencode/a-new-private-b6-run
```

The focused fixture contract produces external private logs/results; it does
not change the main required-check set or trusted CI. Test homes, Git config,
indexes and recovery-child environments are synthetic. Git fixtures disable
inherited hooks/templates/global configuration. The helper does not import a
primary native database, call a model or write approval/task state. These tests
execute trusted local fixture code, not untrusted candidate programs; OS network
isolation is not claimed for a host Python process.

`receipt_tools.py` supplies the optional reference consumer described in
[the receipt guide](../../harness/evidence/README.md). Tests include valid
direct-fix and Ready observations as positive controls, not blanket rejection;
missing null reasons, unknown/changed identity, incomplete/duplicate coverage,
invented billing/model/session/paid authority, invalid status/times, missing or
changed/private/symlink/traversal evidence, secret-canary export and preservation
of unrelated state. Snapshots exercise independently seeded staged, unstaged,
untracked, deleted and mode-changed files. No actual secret is used.

Saved-receipt fixtures test rejection after dirty code/task changes, a workspace
move or changed current authorization. `test_recovery.py` executes the existing
runner and observes owned interruption/deadline behavior, preserved partial
files, an unrelated process surviving, no later-check replay and unchanged
cancelled/incomplete results after persistence/reload. This is **not native
agent cancellation, compaction, restart, scope obedience or duplicate-review
evidence**. Those T09 cases and B5-F1 remain unobserved. See the
[manual recovery procedure](../../integrations/opencode/recovery.md).

## B7 trusted CI bootstrap and anti-weakening

Run `TMPDIR=/tmp/opencode python3 -B -m unittest discover -s tests/harness -p test_trusted_ci.py`
for stdlib policy tests. Run `sh tests/harness/trusted-ci-fixtures.sh` for the
explicit container suite; it prints an owner-only evidence root and retains
per-fixture observations/logs there. No image/tool install, model inference,
native database/auth copy or real-home skill/config change occurs.

The container suite deliberately combines frozen B4 source with the current B7
launcher/probe as a **development reference**, not an already approved/deployed
branch. It tests missing bootstrap, required-contract removal, empty candidate
Make, deleted original Go coverage, forged Ready receipt and a real seeded no-op
defect. The independent probe must fail that defect; a schema-valid receipt or
empty Make target cannot replace actual results. Positive fixtures compare
local trusted-Make and CI-style entry on identical candidate bytes and exercise
non-root/no-network/read-only-root/no-host-file/env/socket access. CID cleanup
and catchable interruption fixtures affect only test-owned containers and keep
partial candidate files. These are deterministic checks, not native agent recovery.

`offline-b7-checks.json` exposes that same explicit suite as a focused runner
check. Unit/static checks are not execution-boundary evidence; container fixtures
are not a hosted GitHub Actions observation. There is no real branch-rule change
or candidate-owned trust bootstrap. See [the trust/adoption guide](../../harness/trusted-ci.md)
and current feature continuation for actual evidence and unresolved adoption.

## B8 connected offline components and feature acceptance

```sh
TMPDIR=/tmp/opencode python3 -B -m unittest discover -s tests/harness -p test_connected.py
```

`test_connected.py` compiles the actual check runner and feeds its observed
results/logs through the delivered receipt consumer and exporter. Three tests
exercise successful execution, missing/changed authority, stale untracked input,
location reconciliation with an actual rerun, a real failed content assertion
and correction, and two repetitions each of signal/deadline interruption.
Pending checks must stay `not_run`; exported/reloaded partial results cannot
be promoted to completion/Ready. Corrected checks cannot clear an unresolved
finding, and source/task sentinels must survive receipt consumption unchanged.

All fixture Git repositories, commits, authority references, project files and
findings are **synthetic**. Only independently specified stdlib check programs
execute. No native model tool, draft generation, human approval, agent review,
compaction or autonomous resume is simulated and counted as acceptance. These
tests add no production runtime, check/receipt schema or state machine.

The same command is exposed by `offline-b8-checks.json` as the explicit focused
check `connected-offline`. Its Python/jsonschema, Git and Go prerequisites must
already be installed. Child processes use a fresh credential-free HOME/Git
configuration, no global hooks or signing, and disabled toolchain/module
retrieval. Temporary directories are not an OS sandbox; do not replace the
synthetic programs with untrusted commands. Real installer/isolated-gate
compatibility is verified separately, not inferred from this fixture.

For optional retained fixture observations, set `SDD_B8_EVIDENCE_ROOT` to an
existing owner-only absolute directory outside the repository. Only generated
results/receipts/logs and sanitized observations are copied there, never native
stores or Git authentication. Fixture workspaces are removed after tests, so
the retained receipts are observations, not reusable current-workspace evidence.
Use seven-day manual raw-log / thirty-day receipt retention by default; no
automatic old-sandbox/native-database pruning. The all-AC evidence and comparison
assessment is in [feature acceptance](../../harness/acceptance.md); current batch
state and actual command/result paths remain in the feature task continuation.

## S0 offline-only follow-up

The separately authorized S0 lab boundary and reproduction commands are in
[S0.md](S0.md). It adds byte-budget/loopback HTTP tests, fake-response request
construction and compatibility probes for the actually observed OpenCode 2.2.5,
and a separate unprivileged worker. No real model is called and no auth store is
copied. The original B8 checks above remain valid within their recorded scopes;
S0 is additional evidence, not a rerun counted as native acceptance. Byte quotas
are not verified DeepSeek tokens. A later exact S1 allocation is conditionally
authorized in spec §Approval, but its original accounting/egress conditions remain
unproven. The later P1 observation below is a distinct, approved exception.

The separately approved `test_tokenizer_source.py` consumes the pinned tokenizers
0.22.1 wheel and official JSON as readonly lab mounts (`/wheel`, `/token-source`)
inside the existing non-root network-none Python image, with
`PYTHONPATH=/wheel SDD_TOKENIZER_SOURCE_FIXTURE=1 HF_HUB_OFFLINE=1`. It performs
four local source/encoding checks without Transformers, Hub imports, model
downloads or system installation. With no approved mounts/opt-in, its tests are
skipped, not passed. Exact data/wheel hashes and private test results are in the
plan/tasks; no package or model data is vendored into this repository.
The local chat example is not proof of hosted gateway chat/tool/reasoning
framing. Original strict S1 remains conditional, not established by local counts.

## P1 bounded text observation

The corrected one-request approval and outcome live in feature 004's
[spec](../../specs/004-vnext-harness/spec.md#approval) and
[tasks](../../specs/004-vnext-harness/tasks.md#continuation-record).
P1 has **run once and stopped on HTTP 429**, consuming one of six original
attempts. Usage/billing remain unknown; no retry or remaining attempt is implied.

`python3 -B tests/harness/p1-preflight.py` defaults to fake mode: 17 isolated
policy/courier/driver tests followed by a native fake-provider topology. It needs
the already local exact binary, pinned Python image, Docker internal isolated
network support and `/tmp/opencode`; it never pulls or installs prerequisites.
The native client is internal-only, with a separate dual-homed gate and no host
home/auth/control/socket mount. A fixed canonical text body discards native
system/tool framing; this is not a model-originated tool test.

The allocation-bound live option is **not a reusable smoke test**. Its stable
control directory and spent dispatch ticket must never be deleted/reset to run
again. Offline proof is source-bound; preserve old proof/evidence if sources
change rather than treating it as current. Public free pricing is distinct from
observed billing. `observation.json`'s `completed` denotes the driver finishing,
not a successful generation: inspect `gate_observation.result.status`, HTTP
status and the separate native exit. Here the wrapper exited 0, native exited 1,
and the gate recorded `incomplete` after 429. No token or feature-acceptance
claim follows. Keep local evidence owner-only with manual retention as above.
