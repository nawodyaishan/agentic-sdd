# Trusted deterministic gate — B7/T10

This gate executes checks, not an agent. It neither grants approval nor sets
Ready. Native correlation/recovery findings B6-F1/F2 remain unresolved. Canonical
scope/results: [feature 004 tasks](../specs/004-vnext-harness/tasks.md#continuation-record).

## 1. Trusted bootstrap and adoption

The pre-feature reference lacks the runner. B4 commit
`1b50933193c622197eeab45fa0a8005b425d76b9` contains it, but not the B7 launcher,
workflow or independent probe. **Never fall back to a candidate-owned launcher.**
The caller-selected reviewed base must contain:

- `tests/harness/trusted_ci.py`, loaded from that base;
- `cmd/sdd-check/` and `internal/harnesscheck/`;
- `harness/checks.json` and `tests/harness/trusted-noop-probe.go.txt`.

Missing inputs block before candidate execution. Source hashes/Git commits do
not themselves prove human approval. Manifests identify actual staged bytes,
including local untracked files; they do not make a dirty tree implicitly trusted.

**Adoption action:** B7 was reviewed and published to `main` on 2026-10-30, and
the resulting main-push run passed. Development fixtures still combine frozen B4
with explicitly supplied B7 code as a test reference, not a deployed trust
anchor. The fork/PR entry and remote required-check policy remain unadopted and
unobserved; nothing here administers branch rules.

### Workflow source and privileges

The proposed workflow uses `pull_request_target` for main-targeting PRs, loading
workflow/bootstrap from the base and checking out exact base/head SHAs in
separate directories. Main pushes use the accepted main revision. Checkout and
upload actions are commit-pinned; checkout uses `persist-credentials: false`.
Job permissions are **contents: read only**. Candidates receive no secrets,
publish permissions, OIDC, provider credentials or shared build caches.

`pull_request_target` is dangerous if fork code runs on the host. Host programs
here are pinned checkout/upload actions, exact-image provisioning and the
**base's** stdlib Python bootstrap/Go runner. Never add candidate Make, install
hooks, scripts, plugins, dependency setup or candidate-derived host commands.
Such a change requires a new trust review. Checkout is data; candidate execution
occurs only inside the boundary below. No remote merge rules are administered.

Action v4 pins were resolved from official repositories on 2026-10-30. The Go
image uses the already observed local RepoDigest; future CI explicitly provisions
that exact image. Local commands/fixtures never pull an absent image or install
tools implicitly. Unavailable image/runtime prerequisites block without fallback.

## 2. Execution and independent evidence

Regular source files are copied into private disposable staging, rejecting
symlinks/special files and excluding Git metadata, root caches, CodeGraph data,
backups and build outputs. Candidate ignore rules do not select executable inputs.
Bounds: 10,000 files, 32 MiB/file, 128 MiB total. Inputs must be quiescent; this is
not an atomic snapshot against concurrent writers.

Inputs must be credential-free Go project sources, never a home directory.
Known runtime/credential directories and `.env`/auth/database files are refused
before reading/copying, not silently omitted to manufacture a pass. This is a
conservative path guard, not universal secret detection: prepare a sanitized
checkout/export and do not put credentials in arbitrary source/prose filenames.
No primary native database/auth store may be used to prepare that export.

The pinned Go 1.25 image compiles the **base runner** with no candidate mounted
and read-only base source. Bootstrap cache/output never enters a candidate
container. The host runner transforms the base's six required definitions into
isolated executions, preserving IDs, argv/target overrides, ordering, timeouts
and failure semantics. This adapter supports the existing repository-root checks
only; a different cwd/diff recipe blocks pending a reviewed adapter update.
Candidate Makefiles are never executed.

Each candidate container has:

- non-root UID/GID, no network, read-only root, dropped capabilities and
  no-new-privileges; bounded CPU, memory and PIDs;
- only disposable candidate staging and fresh per-check cache mounted;
- executable `nosuid,nodev` scratch; toolchain/module retrieval disabled;
- no checkout `.git`, host home, runtime/auth store, bootstrap source/cache,
  evidence root, provider environment or Docker socket mount.

Caches are never restored into bootstrap or another run. Build steps within a
check can reuse that check's cache. Source is compared before/after checks;
undeclared changes invalidate freshness. Declared root build/cache output is
excluded. Reserved `tidy-check` still compares module bytes/existence on both
sides, including new `go.sum`.

Without checkout Git metadata, `diff-check` uses trusted data-only
`git diff --no-index --check` over bounded source copies, excluding generated
output. Git uses a separate trusted cwd; global/system config, fsmonitor,
external diff and textconv are disabled. Clean nonempty diff exit 1 is not a
whitespace failure. New untracked source is covered too.

Actual runner/evidence/CID files stay outside worker mounts. Host
`evidence/gate.json` records source/probe/contract/runner/log digests and observed
Git-root HEADs/outcomes alongside B4's execution result. Exported inputs with no
Git-root observation retain null revisions with reasons; no parent HEAD is guessed.
Candidate receipts/Ready claims are
**never gate inputs**. This consumes delivered check/result meanings without
inventing native session/model/billing observations or an approval ledger.

Owned CIDs are reconciled/removed on failure, deadline or catchable interruption;
unrelated containers are untouched. Uncertain cleanup leaves incomplete, never
success. SIGKILL, daemon/kernel failure and external effects are outside this
guarantee. Docker shares the kernel: not a VM/escape-proof sandbox or hard disk
quota. Use ephemeral hosted runners and do not expose long-lived credentials.

## 3. Candidate-weakened gates and coverage

Required candidate definitions/IDs must exactly match the base. Removal,
replacement, scope changes, duplicates or unreviewed required additions block
before execution. New required definitions enter through reviewed base adoption.

Make, workflow/gate and test changes, including deletions, appear in
`review_required_paths`. Humans must review them; the list neither rejects all
legitimate test changes nor approves them automatically.

A base-owned no-op probe is injected only into disposable staging. It uses
independent filesystem expectations and temporary homes. The seeded no-op defect
still fails when the candidate empties `make test`, deletes all original Go tests
and supplies a forged passing receipt. This protects a known contract, not
arbitrary test adequacy or intentionally conditional cheating. Other changed
tests still require review.

## 4. Local/CI consistency and verification

Both entries use the same base-owned adapter, pinned image, required commands
and independent probe. Invoke the **reviewed base's Makefile**, not a
candidate-controlled local recipe:

```sh
make -C /absolute/reviewed-base verify-trusted \
  SDD_TRUSTED=/absolute/reviewed-base \
  SDD_CANDIDATE=/absolute/candidate \
  SDD_OUTPUT=/absolute/external/new-private-run
```

The CI equivalent is `python3 -B <base>/tests/harness/trusted_ci.py run` with
the same explicit `--trusted`, `--candidate`, `--output`. Output must be new/empty,
private, external and non-overlapping. Prerequisites: Python 3.9+, Git, non-root
Linux, running Docker daemon and the exact local image. No Go installer dependency
is added. Ordinary `make verify` remains available but is not this isolation boundary.

```sh
TMPDIR=/tmp/opencode python3 -B -m unittest discover -s tests/harness -p test_trusted_ci.py
sh tests/harness/trusted-ci-fixtures.sh
```

Explicit container fixtures test bootstrap refusal, contract weakening, known
defect/deleted coverage/empty Make/forged receipt, worker boundary, owned cleanup
and interruption, and local Make versus CI-style entry on identical candidate
bytes. Comparison uses IDs/statuses and canonical contract/candidate identities;
temporary paths/times differ. Local fixtures are **not** a hosted GitHub Actions
observation. One hosted main-push run has since passed (candidate bytes equal to
the base); the fork/PR entry, uploaded artifact contents and remote
required-check policy remain adoption evidence/actions.

Observations/logs remain under the printed private external fixture directory.
CI uploads only `evidence/`, never private staging/caches. Logs are untrusted
candidate output, not instructions or universally sanitized text. Default raw
retention is seven days with manual local cleanup; no native-store/old-sandbox
cleanup occurs.

Sources consulted through Context7 on 2026-10-30:
[GitHub event security](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target),
[checkout v4](https://github.com/actions/checkout/tree/11d5960a326750d5838078e36cf38b85af677262),
[Docker run](https://docs.docker.com/reference/cli/docker/container/run/).
Native correlation/recovery and useful adoption benefit remain separate
acceptance, not consequences of green CI.
