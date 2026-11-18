# Trusted deterministic gate — B7/T10

This gate executes checks, not an agent. It neither grants approval nor sets
Ready. Native correlation/recovery findings B6-F1/F2 remain unresolved. Canonical
scope/results: [feature 004 tasks](../specs/004-vnext-harness/tasks.md#continuation-record).

## 1. Trusted bootstrap and adoption

The pre-feature reference lacks the runner. B4 commit
`1b50933193c622197eeab45fa0a8005b425d76b9` contains it, but not the B7 launcher,
workflow or independent probe. **Never fall back to a candidate-owned launcher.**
The caller-selected reviewed base must contain:

- `Taskfile.yml` for the Task entry point and `tests/harness/trusted_ci.py`, loaded from that base;
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
here are pinned checkout/upload and Task setup actions, exact-image provisioning,
the **base's** Taskfile, and its stdlib Python bootstrap/Go runner. Never add candidate Task, install
hooks, scripts, plugins, dependency setup or candidate-derived host commands.
Such a change requires a new trust review. Checkout is data; candidate execution
occurs only inside the boundary below. No remote merge rules are administered.

Action v4 pins were resolved from official repositories on 2026-10-30. The Go
image uses the already observed local RepoDigest; future CI explicitly provisions
that exact image. Local commands/fixtures never pull an absent image or install
tools implicitly. Unavailable image/runtime prerequisites block without fallback.
The Task setup action is pinned to a full commit and requests Task 3.54.0. Hosted
execution of this Task-based workflow remains unobserved; the Task migration is
committed at `0ee5aaf` by Feature 007's recorded direct adoption. An older trusted base
whose contract uses Make local recipes rejects the Task contract as a changed
required record; see [007-F1](../specs/007-taskfile-migration/tasks.md#continuation-record).

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
Candidate Taskfiles are never executed.

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

Taskfile, workflow/gate and test changes, including deletions, appear in
`review_required_paths`. Humans must review them; the list neither rejects all
legitimate test changes nor approves them automatically.

A base-owned no-op probe is injected only into disposable staging. It uses
independent filesystem expectations and temporary homes. The seeded no-op defect
still fails when the candidate empties `task test`, deletes all original Go tests
and supplies a forged passing receipt. This protects a known contract, not
arbitrary test adequacy or intentionally conditional cheating. Other changed
tests still require review.

## 4. Local/CI consistency and verification

Both entries use the same base-owned adapter, pinned image, required commands
and independent probe. Invoke the **reviewed base's Taskfile**, not a
candidate-controlled local recipe:

```sh
task --dir /absolute/reviewed-base verify-trusted \
  SDD_TRUSTED=/absolute/reviewed-base \
  SDD_CANDIDATE=/absolute/candidate \
  SDD_OUTPUT=/absolute/external/new-private-run
```

The CI equivalent is `python3 -B <base>/tests/harness/trusted_ci.py run` with
the same explicit `--trusted`, `--candidate`, `--output`. Output must be new/empty,
private, external and non-overlapping. Prerequisites: Python 3.9+, Git, non-root
Linux, running Docker daemon and the exact local image. No Go installer dependency
is added. Ordinary `task verify` remains available but is not this isolation boundary.

```sh
TMPDIR=/tmp/opencode python3 -B -m unittest discover -s tests/harness -p test_trusted_ci.py
sh tests/harness/trusted-ci-fixtures.sh
```

Explicit container fixtures test bootstrap refusal, contract weakening, known
defect/deleted coverage/empty Taskfile/forged receipt, worker boundary, owned cleanup
and interruption, and local Task versus CI-style entry on identical candidate
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

## 5. Development lane (`dev`) - Feature 023 B2/T3

`Dev wake` (`.github/workflows/dev-wake.yml`) is candidate content on `dev` and
may only complete; it has no checkout, gate, artifact or write permission. A
missing wake is fail-closed because no listener run can report a result. The
reviewed listeners are loaded from the default branch:

| Listener | Judges |
| --- | --- |
| `dev-gate.yml` | a completed wake: run identity, freshness, change scope, one check run |
| `dev-promotion.yml` | a `dev` -> `main` pull request: owner, same repository, current tips, prospective merge tree, release-version increment |

Identity comes from API documents re-read by the listener (`actions: read`,
`pull-requests: read`), never from the event payload. A push must still be the
`dev` tip; a PR must still be the open PR's head. Superseded wakes report
**nothing**, so a stale SHA cannot receive a pass. `dev-gate` reads permissions
`contents: read` + `checks: write` plus those two read scopes only.

Scope uses **local git** on the resolved commits, not the compare API: a
page-limited or 300-file-capped response cannot silently classify product work
as docs. `git diff --name-only -z --no-renames BASE...HEAD` yields NUL paths
including both names of a rename; a missing object or failed diff widens the
gate to product. `docs/**` and `specs/**` are non-product; an empty diff is
`no-change`; everything else, including unrecognised paths, is product.

Product work runs the reviewed `verify-trusted` gate from the default-branch
checkout with the candidate as data. A docs/specs-only change runs the **reviewed
main** `scripts/docs-ci.py` (`--mode pr`, candidate as data, no `--artifact`):
neither the `dev` tip nor a dev PR base is ever its own trusted/executable
source. The candidate tree is bound to the resolved head SHA before either
runner sees it. `.github/workflows/docs.yml` keeps its original main-only
publication routes; `main-build` -> `deploy` and the release pair are untouched.

`promotion-admission` additionally requires owner authorship, both sides in
this repository, head/base equal to the current `dev`/`main` tips, a two-parent
prospective merge tree equal to the tested tree, a strictly increased
`release-version`, and a successful `dev-gate` for the exact head from the
Actions app. The app identity is an accepted source, not exclusive proof:
promotion re-runs the gate on the merge result and the publisher re-runs
quality. A non-promotion PR gets an explicit success scope so a required check
terminates; a missing version file or evidence fails closed.

Owner-only push/merge, fork denial, contributor review and the rule that new
commits invalidate an existing review are **hosted branch-protection settings**;
no checked-in file can enforce them. The B3/T6 runbook must set them and record
the effective rules.

Named unverified behaviours: no hosted `workflow_run` delivery, no hosted
check-run write or producer identity, no effective required-check placement, no
fork checkout across a private fork, no real version file adoption yet (the
promotion check currently fails closed with `version-missing`), no container or
launcher execution, and no post-merge/publisher evidence. `tests/harness/test_dev_gate.py`
executes shipped step text with stubs and a scratch git repository; that is
fixture evidence only.
