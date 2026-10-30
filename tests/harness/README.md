# Harness fixtures

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
