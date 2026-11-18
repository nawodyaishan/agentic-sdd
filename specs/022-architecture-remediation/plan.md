# Feature 022 — Implementation plan

**Revision:** r2, 2026-11-16 (adds §12 Opus conditions). **Parent:** [spec r3](spec.md).
**State:** draft; approval lives only in [spec §9](spec.md#9-approval).
**Execution ledger:** [tasks r1](tasks.md). No implementation is authorized.

## 1. Approach and current baseline

Correct the demonstrated safety contracts first, then identity/durability and
policy, then bounded maintainability. Preserve the typed library facades,
preview-first behavior, legacy backup readers and the trusted-base boundary.
Parallelism is by independent ownership, not by editing shared files together.

Draft-time HEAD is `e4b0d14`; 020/B9 corrections are committed, not automatically
accepted. Existing dirty work includes `opencode-pack/**`, appops routing,
fleet/skillsync/TUI routing tests and two website pages. Treat it as owner work:
record its identity, inspect it and obtain a scoped adoption/handoff before a
worker edits those files. No reset, overwrite, implicit commit or model-pin
migration. A worktree based on HEAD does **not** contain that overlay.

CodeGraph supplied current symbols/callers for the execution, installer, fleet,
MCP and receipt boundaries. Targeted config reads established the six required
checks, the eleven-package selection, actual release workflow and test inventory.
Two Sonnet r1 spec reviews are advisory; their incompletely examined surfaces
are not a consequential architecture verdict. Required Opus review is pending.

### 1.1 Concrete scope correction

`cmd/agentic-sdd/opencode_disabled.go` rejects normal-build adapter commands.
The experimental adapter writes `.opencode/opencode.jsonc`, not global profiles.
It is inventoried without implementation or excluded tests. This correction is
in spec r3; it does not re-enable Feature 005/006/007 work.

## 2. Decisions proposed for combined review

These are explicit **design proposals**, not adopted owner choices. B0 freezes
the accepted values before consumers start. Changes after freeze return affected
documents/contracts for reconciliation; workers may not choose alternatives alone.

| Spec decision | Proposal / remaining checkpoint |
|---|---|
| O-01 | Retain current required contract unchanged while adding a separate, reviewed qualification lane; named suite inventory in §8. Candidate-owned evidence is supplementary. Separate trusted-base adoption remains a real owner action; its SHA cannot be chosen now. |
| O-02 | `opencode-pack/routing.md` owns upstream route policy, profile frontmatter implements it, tests/docs project it. Preserve the owner's pending pack edits. No host-derived provider migration; exact changed pins need owner adoption of that overlay before B8 routing edits. |
| O-03 | Existing tool-capable review profiles are ordinary advisory reviewers, explicitly not the strict independent method. Keep strict independent inspection rules in their canonical portable reference; do not claim an installed strict runtime or 006 certification. |
| O-04 | Prepare factual dispositions and integrated 014 evidence; keep G-008-01, acceptance and missing consent pending until actual owner decisions. Evidence reconciliation can finish without falsifying gate closure. |
| O-05 | Add per-entry integer `digest_version`: absent/0 = existing algorithm; 2 = framed tree-v2. Raw file SHA-256 remains raw, with explicit algorithm metadata only where needed; no raw digest is reinterpreted as a tree digest. §4 owns details. |
| O-06 | `internal/skillsync` owns supported home-target locking; execution retains a separate per-run owner. Linux/Darwin use stable kernel-held descriptors, no age eviction; changed mutation paths on unsupported OSes fail closed. Process outcome proposal: existing `failed`, exit 1, `timed_out:false`, preserved actual leader exit and explicit completion-uncertainty summary. Group-identity safety needs Opus resolution before B2. |
| O-07 | 4 MiB each for skillsync/fleet manifest reads, existing `execution.MaxRecordBytes` for records; limit+1 detection, regular files only, bounded `-` stdin retained. Preserve current help/version ordering; document/test exceptions, no grammar change. |
| O-08 | Retain startup/context outcomes using deterministic stalled-reader and mandatory-word fixtures. Timings are observations, not fragile speed thresholds; retain P2 scope unless the owner explicitly defers it. |

No extra deployment, authentication, model-spending, live adoption or publishing
decision is bundled into document approval. Minimum OS/Go compatibility and old
backup fixtures must be checked before agreeing to the platform/size proposals.

## 3. Shared contract K1 — mutation ownership

### 3.1 Owner and domain inventory

| Writer / observed object | Domain and owner | Integration |
|---|---|---|
| `SyncWithOptionsResult` installer with OpenCode profiles; skill restore touching profiles | Canonical `<home>/.config/opencode/agents`; owner `internal/skillsync` | Acquire before authoritative destination revalidation; hold through backup/write/rollback |
| `fleet.ApplyPlan`, `fleet.RestoreBackup`, provenance reconciliation | Same agent-target domain | Acquire through skillsync before plan CAS; pass a checked handle to the nested engine and hold through sidecar/live-pointer finalization |
| `ApplyMCPPlan` and MCP restore | Canonical config target under the supported home; owner `internal/skillsync` | Separate MCP domain, not gratuitous contention with agent profiles; replace age-based reclaim |
| Init/brief/record/check/package/clean/checkpoint writes | `(canonical run root, run ID)`; owner `internal/execution` | Guard outside the directory cleanup removes; ownership and run identity revalidated after acquisition |
| Experimental adapter Apply/Rollback | Project config; default-disabled | Inventory only; no new integration or excluded validation |

### 3.2 Proposed API and invariants

Add the mechanism inside skillsync, not a new root package solely for a few
helpers. Proposed names are `AcquireTargetMutation(home, domain)` and an opaque
`*MutationGuard` with `Close` and checked home/domain ownership. Inputs are
canonical absolute paths and a closed set of supported domains, never arbitrary
user-supplied lock filenames. Output is a held descriptor or a typed conflict/
unsafe-path/unsupported error. No source bytes or secrets appear in diagnostics.

The guard is non-forgeable outside its package (private fields), usable only
while held, and validated on each nested entry. Existing exported convenience
operations acquire their own guard. A guarded variant accepts an actual matching
handle, not a bool or caller claim; internal locked helpers do not reacquire.
Lock acquisition is nonblocking. Map to existing fleet/MCP/execution conflict
classes and exit 2 at CLI boundaries; do not introduce command/JSON exit classes.

Stable lock files are private regular files in owned state outside deletion
targets. On Linux/Darwin open without following symlinks and without blocking
on FIFOs; validate descriptor type, path identity and parent policy. Lock, unlock
and diagnostics never truncate an unrelated referent. Do not unlink a held
kernel-lock inode on release or use elapsed age as abandonment evidence.
Refuse a replaced/mismatched object; advisory locks are not security isolation.
Windows/other changed mutation paths return a clear unsupported result unless
an equivalent reviewed implementation is approved; read-only paths remain usable.

Execution's stable guard lives under `<runRoot>/.locks/`, keyed by validated run
ID, not inside `<runRoot>/<runID>`. All writers acquire it before run identity
validation/mutation; clean holds it through deletion and final absence check.
Readers report state without inventing ownership. Init/clean/recreate cannot
cross one another. Existing `.lock`/heartbeat observations remain readable;
legacy live/unknown ownership blocks migration. Migration assumes quiescent
older binaries: cross-version exclusion with binaries that ignore the new guard
is not claimed. The exact migration protocol is part of O-06/Opus review.

Fleet's existing `AcquireLock(backupRoot)` needs safe-leaf hardening even if
product callers migrate to the target guard. Preserve its exported internal
compatibility where possible; never mistake a backup-root lock for the new
home-target contract. Test two homes sharing one backup root using the new
domain guard, not a guarantee that this legacy wrapper becomes target-keyed.

### 3.3 Cross-task boundary

B1 produces the guard/error/checked-handle contract. B6 consumes it without
changing its semantics. B2 owns execution's separate guard; it must satisfy the
same safety properties without importing skillsync's installer surface.
If a worker needs a different signature, it returns a proposed contract delta;
no concurrent caller repair or hidden fallback is allowed.

## 4. Shared contract K2 — backup identity, input and provenance

### 4.1 Digest compatibility

Keep `treeDigest`/exported `TreeDigest` as legacy semantics. Add an explicitly
versioned tree-digest entrypoint for new backups and dispatch verification by
the recorded entry version. Unknown version fails safely, never guessed.

Proposed v2 stream: fixed domain header `agentic-sdd/tree/v2\0`, then lexical
relative slash paths; each entry is a kind byte, uint64 big-endian path-byte
length, path bytes, uint64 content-byte length and content bytes (directory
length zero). Root itself is excluded consistently; reject links/special files,
and detect incomplete/changed reads rather than hashing different lengths.
This is identity, not a signature. Preserve raw file SHA-256 unchanged.

Skillsync's existing format 1/2/3 data stays readable. New tree entries carry
`digest_version:2`; legacy absent/0 dispatches v1. Fleet format-4 per-file
digests are raw bytes today (`verifyBackupCopies`); add compatible per-record
algorithm metadata only for the digest that actually changes. Do not label
those raw SHA-256 fields “tree-v2”. Any tree identity referenced in a fleet
sidecar carries the same tree-version discriminator. Fixtures cover old fleet
sidecars, mixed old/new backup entries and original restore-source identity.

### 4.2 Bounded input contract

Manifest limits are 4 MiB **per file**; record limit is the existing
`MaxRecordBytes`, not a new unrelated constant. Reader opens a regular no-follow
object nonblocking, checks pre/open identity and reads at most limit+1 bytes.
Reject oversize/unknown-version/unsafe input with bounded, content-free errors.
Stdin records remain explicitly supported and bounded. No indefinite FIFO open,
whole-file allocation before checking, or silent special-file fallback.
Use package-local primitives where policy differs; share only mechanics justified
by actual callers. Limits are not a guarantee that all legacy oversized files
remain accepted: O-07 explicitly adopts that compatibility boundary.

### 4.3 Durable finalization and before-state

Held K1 exclusion covers destination comparison, backup digest, commit and
metadata. The engine re-reads before-state under the guard and checks the
caller's expected digest; `AgentFileChange.Old` is not a backup authority.
Checkpoint public mutators similarly own acquisition or require an opaque
checked handle; internal locked variants prevent recursive acquisition.

Proposed journal ordering: prepared backup/intent → files committed → sidecar
finalized → live pointer finalized. Each record is atomically replaced, file
synced and containing directory synced on supported OSes. Record truthful
failure/rollback details even if the final metadata write fails. Reconcile on
the next authorized mutation, not as a side effect of status/preview; read-only
status may report pending reconciliation. Never replay over newer target bytes.
Recovery is idempotent under K1. Preserve existing sidecar states; if their
vocabulary cannot express an intermediate fact, obtain an explicit additive
metadata compatibility revision rather than inventing a “success” state.
Do not claim power-loss proof from process-failure injection.

## 5. Shared contract K3 — execution completion

B2 owns `RunCheck`, process handling, run ownership and checkpoint entrypoints.
Normal leader exit, timeout, limit, interruption and capture failure converge on
one finalization path. A same-group survivor or unobservable completion yields
existing `CheckFailed`, CLI exit 1, `TimedOut=false`, leader exit preserved if
known, and a bounded cause in existing evidence/summary. Real timeout retains
124. A run with this failure cannot be `verified-evidence`; no new result token.

Proposed ordinary post-exit grace: 2 seconds; bounded escalation/confirmation:
1 further second. Tests inject clock/probes or handshake with real fixture
processes; they do not wait out production stale periods. Preserve existing
interrupt and output-limit classifications. Do not use a saved PID as liveness
proof. A negative-PGID probe alone does not prove identity after group-ID reuse:
Opus must approve how group ownership remains demonstrable through wait/cleanup.
If uncertain, record failure without signaling a potentially reused unrelated
group. This is a **blocking O-06 design point**, not permission to improvise a
supervisor, shell wrapper or new product command. Detached children/host crashes
remain outside the containment claim.

## 6. Shared contract K4 — TUI, CLI and policy

### TUI

B3 owns the root confirmation contract; B10 consumes it later. Explicit
decision state owns focus before root shortcuts. A confirmation projection
contains action, target, consequences and decision controls plus a computed
`canConfirm`; render and input routing use the same geometry. Resize invalidates
affirmative dispatch until that projection is visible. Keep details scrollable,
decision controls pinned, default No and capability/write tokens intact.
Extend existing modal/message types; no new UI framework or global registry.

B10 preserves structured adapter results even on error and projects outcomes
through `SummarizeWriteResult` (or one equivalent owner), so child/root notices
cannot disagree. Then make touched MCP inputs/services instance-local and move
initial reads behind generation-tagged commands. Event tests observe a useful
first frame while an injected source is blocked; elapsed timings are recorded.

### CLI and reference ownership

B9 owns record input, root/topic help, per-invocation streams/services and real
wrapper contracts. Preserve current help-order exceptions rather than silently
changing grammar. Build test binaries with `-o` outside the checkout, separate
stdout/stderr pipes, injected fixture companions and link-time legacy identity.
No inherited global test substitutions in the touched paths.

Only B11 regenerates website reference blocks/manifests after interfaces freeze;
workers return documentation-impact notes instead of editing generated pages.

### Policy/content

B8 owns portable/pack policy together, after the existing pack overlay is
explicitly reconciled. Current routing text is the upstream owner; provider
strings in a personal host are not product decisions. C-09 inventory can be
corrected before model choices. Advisory profile descriptions must not claim
the strict independent method. Core owns scoped correction re-review; new
model/independent dispatch still needs authority.

Read-only cases cannot create/sync index files without write authority.
Choose named-SDD-blocker research as the `research-spec` automatic trigger;
explicit broad research is a separately authorized method with effects/privacy
retained. Conditional authority references may reduce mandatory lightweight
branch word loads, never remove prohibitions. Static checks are instruction
coverage, not runtime model/permission compliance.

## 7. Shared contract K5 — release and gate handoff

B4 adds a **separate qualification contract/entrypoint**, not an unreviewed edit
to `harness/checks.json`. Fixed selections are base-owned and independent of
candidate `.gitignore`, model text or workflow defaults. Audit the supported
Python files before selecting them; no broad discovery or excluded experiment.
Until adoption, the old gate plus focused new-lane fixtures provide explicitly
limited development evidence. A contract-changing candidate cannot supply its
own independent trusted verdict. A real base SHA/designation/adoption needs the
owner; `blocked` remains visible if that decision is absent.

B7 consumes B4's frozen command/result contract. Proposed release layout:
read-only verify/package job → artifact+digest+candidate manifest → protected
publisher job → explicit read-only observation. Verification has no release/tap
secrets or OIDC; checkout does not persist credentials. Publisher validates same
commit/artifact digests and does no candidate test execution with secrets.
If GoReleaser cannot publish the preverified artifacts using the pinned version
without rebuilding in the privileged job, stop for the release architecture
decision; do not substitute a privileged rebuild of unverified bytes.

Serialize tap-affecting publishing under one stable group, `cancel-in-progress:
false`, with bounded jobs and explicit recovery steps. This prevents simultaneous
publishers, not a durable FIFO queue: GitHub can replace a pending run. Document
that limitation and safe rerun/reconciliation; do not claim every tag is queued.
Persist sanitized verification records even on failure, bound to the candidate.
Use reviewed full action SHAs and minimal per-job permissions. No token-write
probe is needed for development; readability does not prove tap write scope.

Use a same-workflow post-publish observation or reviewed reusable read-only
workflow, not a PAT workaround or sole `release: published` trigger. Resolve
asset paths with an origin/base URL resolver; keep CDN observation nonblocking
and visibly failed when applicable.

GoReleaser `.Date` is current UTC date; `.CommitDate` is UTC commit RFC3339;
`.CommitTimestamp` is Unix commit time. Replace application wall-clock injection
with commit time, keep `-trimpath` and fixed archive/universal timestamps.
Consolidate snapshot policy, refuse dirty trees for reproducible snapshots,
and keep output/cache outside source. Two clean exports must yield identical
four-application/archive/checksum bytes with pinned inputs. Never execute the
publishing recipe during tests; snapshot execution needs the authorized
nonpublishing task and already-available tools.

## 8. Verification selection and evidence

### Exact development selections

Use repository-sanitized Go environment: `GOTOOLCHAIN=local GOPROXY=off
GOSUMDB=off GOENV=off GOWORK=off GOFLAGS=`, disposable HOME/TMPDIR and external
cache/output. Do not dump inherited environment or secrets into evidence.

| Selection | Recipe / bound | What it proves |
|---|---|---|
| Existing core | `task test`, `task vet`, `go mod verify`, `git diff --check` | Existing eleven supported root packages; no native/online claim |
| Affected core race | `go test -race ./internal/skillsync ./internal/fleet ./internal/execution ./internal/harnessevidence ./cmd/agentic-sdd` | Scoped in-process race checks plus explicit subprocess exclusion tests |
| Nested TUI | In `experiments/tui`: `go test -count=1 -timeout=15m ./...`, `go vet ./...`, and the offline `go list -mod=vendor ./...` plus `tests/tui-vendor-check.sh` (owner decision 2026-11-17 replaced `go mod verify`, which cannot check the path-replaced root module; `harness/qualification.json` is authoritative); race separately with a measured job budget | Nested module only; no excluded root packages |
| Portable content | `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py`; same for `test_research_workflow.py` | The explicitly permitted static suites |
| Proposed gate regressions | Individually select `test_supported_core_go.py`, `test_appops_core_selection.py`, `test_trusted_ci.py`, `test_trusted_ci_container.py` after B0 source audit | Fixed selection, launch/staging/anti-weakening; a missing Docker prerequisite blocks actual container evidence |
| Docs policy | `test_docs_ci.py` individually and website's applicable source/reference/built/export suites | Docs-lane evidence only, not a new root broad discovery |
| Distribution | Both Darwin architectures for all four build IDs; inspect Mach-O using bounded tooling, archive members and checksums; native-host pair smoke | Cross-build/package integrity, not native macOS execution |
| Reliability | Two-CPU cold cache timings with complete outcomes; deterministic barrier/clock fixtures; final scoped race lane | Timing evidence, not unmeasured flake immunity |

The nested `./...` is restricted to the TUI module. Root `go test ./...` and
broad Python discovery remain forbidden. Do not install tools, pull images or
fetch modules silently to satisfy a lane. The 15-minute TUI test ceiling is a
proposed *local measurement allowance*, not a change to the trusted 120/180-second
contract. New job budgets require B12 measurements and approval.

Go 1.27.1 (existing release pin), Task 3.54.0 and GoReleaser 2.14.0 are initial
qualification proposals; retain root `go 1.23` and nested `go 1.26` module
declarations. Minimum-toolchain checks are additional evidence only when locally
available. Retain the docs lane's own reviewed Node/container requirements.
No missing minimum/native toolchain is reported as pass.

### Evidence record

Each task returns changed paths; parent/base identity and overlay digest; actual
model/variant; exact checks, exits and code-state identity; fixture/probe names;
remaining findings/decisions and documentation impact. Report `not_run`/
`blocked` explicitly. Do not manufacture review independence from another
worker's checks. Run artifacts/logs stay private; only sanitized summaries and
durable regression fixtures belong in the repository.

## 9. Agents, specialists and conditional tools

The following are actual installed profiles/catalog IDs checked during drafting,
not authorization to launch them now.

| Alias | Implement profile / explicit model | Assignment and reason |
|---|---|---|
| **H** | `sdd-implement-haiku`, `claude-agent/haiku#high` | Higher-risk concurrency, transaction, release privilege and composed TUI changes; its current profile permits complex work |
| **D** | `sdd-implement-deepseek-1`, override `opencode/deepseek-v4.1-flash-free#high` | Bounded Go/JSON/CLI changes and regression fixtures; free profile has 200k context/32k output limits, so split task briefs and preserve partial writes |
| **G** | `sdd-implement-hydrogen-high`, `9router/ag/gemini-4-hydrogen-high` | Routine fixed-scope Python/config/documentation/policy transformations after contracts freeze |

“DeepSeek Flash High” here explicitly means the installed **Free Console**
model at high effort, not paid `opencode-go/`, native DeepSeek or a 9Router
substitute. If paid Go was intended, ask before changing this assignment.
Hydrogen has no extra `#high` suffix: effort is in its model ID.
Availability is catalog/config evidence, not a successful provider call.

No silent fallback outside these three assignments. On failure preserve partial
state, inspect the child/paths and report blocked; ask before a different model/
provider or extra charge. No child spawns more agents or updates the canonical
ledger/approval. Parent performs integration and permission reconciliation.

| Work semantics | Installed guidance | Selection |
|---|---|---|
| Go/CLI/TUI libraries | `golang-pro` | Principal for code lanes; repository module/toolchain constraints override generic recommendations |
| Regression development | `test-driven-development` | Load before implementation; reproduce red, then repair and retain meaningful green tests |
| Workflow release security | `cicd-automation-workflow-automate`, then `secrets-management` only for the privilege boundary | No credential inspection or live actions; not a request to redesign unrelated pipelines |
| Portable/pack instructions | `agentic-sdd-writing-for-agents` | Keep triggers, authority, progressive disclosure and completion observable |
| Records/navigation only | No additional specialist needed | Existing ownership/approval/link rules suffice; source attribution matters more than an extra role |

Resolve/load only the guidance needed by the active task, not all rows at once.
CodeGraph is default for indexed code, including refresh after changes.
Context7 is conditional for versioned APIs; Exa only for named public facts not
answered locally. Browser/vision is optional for actual rendered docs/diagram
acceptance, not for Go TUI model tests. Repository gates and local fixtures own
verification, not a tool's prose. No new MCP install/authentication is implied.

## 10. Parallel execution design

### What “parallel all batches” means

All engineering work is assigned to the three requested model families and can
be scheduled by this DAG; it is **not** safe to launch every writer at once.
B0 contract/decision freeze and B13 final integration are barriers, not optional
overhead. Tasks sharing a file have an explicit owner transfer.

Default workflow remains one explicitly authorized batch. If the owner approves
an explicit wave set, its independent named batches may run together, with each
reporting its own completion and the parent stopping at the wave's human review
checkpoint. A dependency being green does not authorize another wave. The user's
request to *design* parallelism supplies no implementation authority.

| Wave | Concurrent batches | Main ownership / output |
|---|---|---|
| W0 | **B0** parent preflight/contract freeze | Rebase dirty overlays, O-01–O-08 disposition, Opus review and approval/authorization references |
| W1 — P0 foundation | **B1 H**, **B2 H**, **B3 H**, **B4 D+G**, **B5 G** | Guard primitive; execution; consent; additive gate; assurance records, on disjoint paths |
| W2 — consumers/P1 | **B6 H+D**, **B7 H+G**, **B8 G**, **B9 D** | Integrate guards/digest/provenance; release pipeline; policy; CLI/contracts |
| W3 — presentation/records | **B10 H**, **B11 G** | TUI consumers/startup and canonical docs/record summaries |
| W4 — qualification closeout | **B12 D+G**, then **B13 parent** | Measured reliability/fixed qualification inputs, then serial whole-candidate verification and finding reconciliation |

Proposed cap: six active write workers, at most three H, two D and two G
(the overall cap still applies). Lower it on provider quota, memory or test
contention; do not fill capacity with lower-priority work while a ready P0 is
waiting. Multiple sessions may use one profile with distinct briefs/paths.

Use disjoint-file ownership in the current tree only after the owner-work
overlay is frozen. For a same-package lane needing an unstable producer, use
separate disposable workspaces seeded with the **same explicitly adopted**
base/overlay, or wait for the producer; worktrees are not isolation/security.
No shared `.git`, staging, cherry-pick or commit operations by workers.
Parent applies returned diffs and re-runs integrated checks; applying another
worker's patch is integration, never permission to overwrite unrelated work.

### Shared-path transfers

- `internal/skillsync`: B1 new guard/reader mechanics → B6 transaction/digest
  consumers. B1 does not concurrently edit B6's existing transaction files.
- `internal/fleet/plan.go` and lock files: B1 safe lock leaf → B6 consumers.
- `internal/execution` and checkpoint interfaces: B2 only; B9 edits CLI consumers
  later, never service files.
- `experiments/tui/{app,modal,mcp,fleet*}`: B3 consent → B10 results/seams/startup.
  B8 never edits TUI code; B11 never regenerates during active interface edits.
- `Taskfile.yml`, `tests/harness/trusted_ci.py`: B4 additive selection →
  B7 snapshot wiring (Taskfile only) → B12 measured budget/selection proposal.
- `.github/workflows/release.yml`, `.goreleaser.yml`, docs observation wiring:
  B7 only. B4 proposes consumer commands and CI lane; it does not edit release YAML.
- `specs/**` historical authority: B5 targeted factual findings → B11 summaries.
  Spec 022 approval/ledger and root `AGENTS.md` are parent-only.
- Root README/docs index, website generated blocks and freshness manifest: B11
  only. B7 returns release-recovery prose as a patch proposal, not edits there.

## 11. Readiness, risks and next checkpoint

| Risk / blocker | Control / what remains unresolved |
|---|---|
| Unsafe group-ID reuse or migration from older file locks | O-06 and required Opus verdict; B1/B2/B6 cannot choose a shortcut |
| Arbitrary interpreter expansion in a trusted runner | Separate reviewed lane, fixed argv inventory and real adoption barrier |
| Candidate cannot pass exact trusted contract after changing it | Keep old contract; qualify additive work with old-base evidence plus supplementary lane; never call candidate itself trusted |
| Legacy digest/sidecar ambiguity | Per-entry versions, raw-vs-tree distinction and old fixtures; O-05 approval |
| Privileged rebuild differs from verified artifact | Publisher validates the artifact handoff; unresolved GoReleaser reuse capability blocks release design, not permission to rebuild secretly |
| Owner routing changes in flight | Freeze/adopt overlay or block affected B8/related tests; do not restore HEAD over it |
| P0 appears complete but acceptance/designation absent | Keep engineering evidence and human disposition separate; no fleet-live expansion |
| All-batches parallel request interpreted as unrestricted execution | Explicit named wave authorization plus checkpoints; no worker self-advancement |

The [tasks readiness reconciliation](tasks.md#7-readiness-and-continuation)
names every AC owner and unresolved decision. The package is a concrete draft
for **combined review**, not a claim that every dependent batch is implementation
ready. Next action: resolve/adopt proposed O choices, obtain the required Opus
architecture review, then approve exact revisions and explicitly authorize B0/
W1 or a narrower scope. Do not start implementations from this draft.

### Primary-source checks reused/updated during planning

- [GoReleaser template fields](https://goreleaser.com/customization/templates/):
  `.Date` is current UTC RFC3339, `.CommitDate`/`.CommitTimestamp` are commit time.
  Context7 returned the literal common-fields definitions on 2026-11-16.
- [Reproducible builds](https://goreleaser.com/blog/reproducible-builds/) and
  [universal binaries](https://goreleaser.com/customization/universalbinaries/):
  fixed timestamps and independent application IDs; still require byte checks.
- [GitHub workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax):
  per-job least privilege and concurrency; pending-run replacement is not FIFO.
  [Reusable workflows](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations)
  cannot elevate the caller's token permissions.

## 12. Opus verdict and conditions frozen at B0 (r2, 2026-11-16)

The required consequential architecture verdict came from `claude-opus-5-5`
(read-only). Overall: *needs changes*; each of B1, B2, B4, B6 and B7 is *ready
with conditions*. The conditions below are the frozen B0 contract for those
batches. They tighten mechanisms only; no spec requirement changes. The verdict
is not a repository record and is not approval. Working copy:
`/tmp/opencode/022-impl/opus-verdict.md`.

### 12.1 K1 guard — B1 and B6

- Use kernel `flock` on a private regular lock file. Open with `O_CLOEXEC`,
  `O_NOFOLLOW|O_NONBLOCK|O_CREAT`, never `O_TRUNC`. After open, require a regular
  file owned by the current user with mode 0600.
- After locking, compare device and inode of the descriptor with the path; retry
  once, then report a conflict. Check each parent with `lstat` inside the home.
- Write content only through the locked descriptor, after validation.
- Agent and MCP guard files are never unlinked. An execution `.locks/` file may
  be unlinked only with an identity recheck after locking; otherwise accumulation
  is documented.
- Lock order is fixed: agent target → MCP → legacy. Nested work uses only the
  checked handle.
- `ENOTSUP`/`ENOLCK` returns a typed refusal, never success.
- Fleet additionally acquires the legacy `<backupRoot>/.fleet.lock`, with the same
  hardening, so old fleet binaries are excluded.
- MCP uses a **new** lock filename; `mcp.lock` is never reused. While holding the
  new guard, a legacy `mcp.lock` younger than 30 s is a conflict. Otherwise it is
  removed only if device, inode, size and mtime match the inspected file.
- Execution: after acquiring `.locks/<runID>`, a live or unknown legacy
  `<runDir>/.lock` is a conflict; a stale one is removed conditionally as above.
  Keep a legacy-format `.lock` with heartbeat while holding the new guard, so old
  binaries refuse to write. `ValidateRunID` rejects `.locks`; enumeration and
  clean skip it.
- Windows mutation paths fail closed (B1 T02).
- Documented limits: old binaries that ignore the guard; downgrade is fail-closed
  for `digest_version:2`.

### 12.2 O-06 group identity — B2

- Every group signal passes one gate that closes once reaping begins. This covers
  interrupt, timeout, output limit and post-exit cleanup.
- Linux: observe exit with `waitid(P_PID, …, WEXITED|WNOWAIT)` (does not reap).
  While pinned, inspect the group through `/proc`. Survivors: up to 2 s grace,
  then SIGTERM, 1 s, then SIGKILL, all while pinned. Result `failed`.
- Pass only when, after reaping, `kill(-pgid, 0)` returns `ESRCH` within the grace
  period and the exit is 0 with no other cause.
- Any uncertainty → `CheckFailed`, exit 1, `timed_out:false`, leader exit
  preserved, cause in the summary. No further signal.
- Children never inherit the guard descriptor (`O_CLOEXEC`).
- Darwin: classification only, no new signals after leader exit; residual
  documented; cross-compile evidence only.
- Required real-process fixtures: redirected survivor → failed; no child → pass;
  survivor exiting within grace → pass; injected uncertain group → no signal.
- Existing exit contracts stay: timeout 124, interrupted 1, conflict 2.
- Existing age-policy tests encode the superseded rule; update them to the
  identity rule rather than keep them.

### 12.3 K2 digest — B6

- Freeze at B0: kind bytes (0x01 directory, 0x02 file); relative paths sorted
  byte-wise over the full path; root must be a directory; empty directory is the
  header only; no Unicode normalization; permissions and timestamps excluded.
- Content length comes from `fstat`; read size+1 bytes and fail on mismatch.
- v2 walks with `lstat`, rejects links and special files, and does not use
  `os.DirFS` `Open`.
- Dispatch: skill entries absent/0 → v1, 2 → v2, otherwise reject. Agent entries
  use the raw digest only; any nonzero version is rejected. New entries require a
  non-empty `sha256`. `identity.go:204` dispatches by version too.
- Fleet `ManifestFormat` is not bumped; the sidecar keeps raw file digests.

### 12.4 Release — B7 (T14 blocked; T15/T16 may proceed after W2)

- **Owner decision required (blocks T14):** verified GoReleaser documentation
  states the `prebuilt` builder and split/merge are **GoReleaser Pro only**
  (`goreleaser.com/customization/builds/builders/prebuilt/`,
  `…/general/partial/`, checked 2026-11-16). Choose either (a) GoReleaser Pro for
  the artifact handoff, or (b) upload the verified artifacts with `gh` through a
  reviewed script that renders the cask. A privileged rebuild of unverified bytes
  stays forbidden.
- Build job runs no candidate tests. Only the publisher job uses
  `environment: release` and tap secrets, with per-job timeouts.
- Concurrency group is a fixed string, `cancel-in-progress: false`, set on the
  publisher job.
- Workflow-level permissions become empty or read-only. Remove `id-token: write`
  unless used. Current `release.yml` sets `id-token: write` at workflow level
  (line 10) and `environment: release` on the job (line 15).
- Pin checkout, setup-go and upload/download-artifact to full SHAs.
- Publisher re-hashes every artifact against the verifier manifest and checks
  `github.sha` equals the tag commit.
- Downgrade guard: refuse a cask update when the tap version is already at or
  above the tag.
- Document that GitHub can replace a pending run and that a cancelled run needs a
  manual rerun on the original commit.

### 12.5 Gate — B4

- T07 and T09 proceed. The Python and archive lanes of T08 proceed.
- **Owner decision required (blocks the nested-TUI lane only):** the isolated gate
  runs with `--network=none` and `GOPROXY=off` (`tests/harness/trusted_ci.py`
  lines 154–161), and `experiments/tui/go.mod` has 23 requirements with no vendor
  directory. Choose vendoring, or a network-enabled module-download step with
  checksum verification, read-only mounted, containing no candidate code.
- Push evidence to `main` is not independent: there the trusted base is the
  candidate itself.

### 12.6 Owner decisions still open

1. T14 publish route (12.4): GoReleaser Pro, or `gh` upload plus a reviewed cask
   renderer. Blocks B7 T14 only.
2. TUI-lane module supply (12.5). Blocks the nested-TUI lane only.
3. Darwin O-06 posture: classification only, with the residual documented. The
   owner may override it; B2 proceeds under the default.

### 12.7 Owner decisions recorded 2026-11-17

1. **B1 lock API:** approved as built. B6 must confirm restore and cleanup never delete `<home>/.agentic-sdd/locks/`.
2. **Container suite:** diagnose first (read-only findings); no adoption of the container lane until the diagnosis is owned.
3. **Publish route (T14):** `gh` upload of verified artifacts plus a reviewed script that renders the Homebrew cask. No GoReleaser Pro dependency.
4. **TUI module supply:** vendor the nested-module dependencies so the gate runs offline.
5. **Darwin group identity:** accepted as classification-only, with the residual documented.
6. **Behaviour changes accepted:** `run check` fails closed on Windows; `q`, `?` and route keys are inert during TUI confirmations.

### 12.8 Owner decisions recorded 2026-11-17 (release path, T14)

1. **Tag without CI:** the publisher checks that the tagged commit has passing check runs (fails closed when none exist) before publishing.
2. **Tap-update scripts:** run from a reviewed tooling commit pinned by the repository variable `RELEASE_TOOLS_SHA`, not from the tag. The owner sets that variable after reviewing and committing this batch; until then the release fails closed.

### 12.9 Owner decisions recorded 2026-11-17 (release gate, approvals, TUI)

1. **Residual run after the final CI snapshot:** accepted and documented.
2. **Own-run exclusion:** by check-suite ID of this workflow run, not by details_url. Implementation applies the deepseek-max review findings before the release batch is committed.
3. **Second release queues behind the first (up to 50 min):** accepted.
4. **RELEASE_TOOLS_SHA:** environment-scoped variable on the `release` environment, with required reviewers and self-review prevented. The owner sets it in GitHub settings.
5. **Release batch review:** deepseek-max read-only review, findings applied by an implement lane, then commit and push of the release files only.
6. **T11 digest codec:** approved.
7. **TUI lane vendoring:** approved. The vendored copy must be regenerated after T12 and T13 land, because it contains a copy of the root packages.
8. **Container overlay fix:** not yet decided; pending its report.
9. **TUI failures:** fix the restore-path test to be wrap-aware. Diagnose and fix T19 in its own batch.

10. **Tag trust (2026-11-17):** publishing moves to its own workflow (`Release publish`) triggered by `workflow_run` from the build workflow (`Release build`, tag-triggered, no tokens). The publish workflow file is taken from the default branch, so a tag cannot change its code. Applied in the release batch before commit.
11. **Manifest and ancestry (review fixes, applied):** verify requires the exact set {checksums.txt, one archive}; the tagged commit must be reachable from the default branch; own-run exclusion uses the check-suite id.

12. **Release approvals (2026-11-17, owner):** the release environment keeps required reviewers with the owner as the approver. Prevent-self-review is OFF and admin bypass is ON, so the owner approves their own releases. This is a manual confirmation gate, not an independent second-person control. Recorded as an accepted risk.

13. **Amendments to the frozen plan (owner, 2026-11-17):**
    - Lock order (amends §12.1): the fleet legacy `.fleet.lock` is taken first, then the agent-target guard. Only fleet takes the legacy lock, so no cycle exists today.
    - Journal order (amends §4.3): the live pointer is written before the provenance sidecar, so a failed live write after an `applied` sidecar cannot go unreconciled.
    - Not yet ruled on: fail-closed reading of every sidecar under the backup root, and the absence of a conflict-resolution command. Both stay open.

## 13. Amendment r4: open-item resolutions (2026-11-17)

Owner accepted the recommendations in the read-only options report (see
[spec §9](spec.md#9-approval), amendment r4). This section records the design for
each; it creates no batch authorization. Items are named R1–R7.

### 13.1 R1 — G-008-01 stays open (D-01)
No change to 008 records beyond a dated pointer to this plan. Closure path, in order:
(1) owner designates an exact clean SHA after R2 passes; (2) a fresh base-owned gate
run against a later, identified candidate with no `trusted == candidate` shortcut;
(3) a recorded Opus verdict. A push to `main` is never evidence of closure.
Preparing the Opus review request is allowed; sending it needs authorization.

### 13.2 R2 — trusted base and container lane (O-01)
Step 1: re-run the container suite on current HEAD with the pinned base `1b50933`
and the reviewed overlay, and record all six test results with the HEAD SHA. Use
`/dev/shm` for temp and Go caches. Docker is local; it writes no real home.
Step 2 (separate authorization, only after six of six pass): owner picks a clean
reviewed base SHA, advances the base and drops the overlay. A failing run is
reported, not weakened: no skip, filter or tolerance.

### 13.3 R3 — consent provenance (D-02)
For 014, 018, 019 and the 016 release quote: the owning record gets either the
owner-supplied quote with its date, or the line "consent not recoverable
(recorded 2026-11-17)". Never reconstructed from runs or tags. A template is
drafted for the owner; missing items stay missing.

**Owner disposition 2026-11-17:** record as "not recoverable" today. Applied as a
dated note to specs 014, 018 and 019. Spec 016 has no B5 consent annotation, and the
verbatim 016 release quote was not located; it stays "not recoverable" here and the
owner may replace any note by supplying the original words and date.

### 13.4 R4 — 014 closeout table (D-07)
A read-only per-AC and per-finding table with evidence pointers and explicit
`not_run`/`blocked` entries, built from the B5 disposition report and the
current code. The owner then accepts, accepts with exceptions, or records a blocker.
No automatic acceptance and no Fleet-live gate change by the table itself.

### 13.5 R5 — T13 conflict command deferred
Do not add a conflict resolver. A fleet transaction left conflicted must make the
CLI refuse with a stable message naming the manual recovery steps (inspect the
journal and sidecar, restore from the backup, re-plan). Current code already
refuses conflicted plans (`ErrPlanConflict`, `internal/fleet/apply.go`); the work
is the conflicted-transaction message and its test. A future resolver needs its own
spec and race tests (AC-03, AC-12). Fail-closed sidecar reads stay as decided.

### 13.6 R6 — B8 routing and reviewer policy (O-02, O-03)
O-02: routing policy lives in `opencode-pack/routing.md`; profile frontmatter
implements it, tests and docs project it. The owner adopts or discards the pending
pack overlay first; no restore of HEAD over it, no host-derived pins.
O-03: tool-capable reviewers are labelled ordinary and advisory; strict independent
inspection rules stay in the portable reference; no installed strict runtime or 006
certification claim. C-09 stays unblocked.

### 13.7 R7 — nested TUI vendor drift
Add a `task tui-vendor-check` (read-only): re-vendor into a temporary directory
using the same offline settings as the existing TUI lane, diff against
`experiments/tui/vendor`, and resolve the companion's dependencies from `vendor/` (`go mod verify` cannot check the path-replaced root module). Add a stamp file recording the
source SHA of `internal/skillsync` and `internal/fleet` at vendor time; the check
fails early when those trees changed since. Wire it into the trusted-CI TUI lane
only after review. `replace` to the root module already exists in `go.mod`, so the
vendored copy is the offline mechanism, not a replacement for it.

### 13.8 Owner decisions recorded 2026-11-17 (after amendment r4)

1. **Push:** the four r4 commits were pushed (`7313566`).
2. **A1:** re-run on a clean commit via an isolated detached worktree (`/dev/shm/wt-a1`), not the live tree.
3. **014 closeout:** require more evidence first; an argon evidence pass writes `/dev/shm/022-A5/evidence-014.md`. The draft stays unaccepted.
4. **D-02:** record "not recoverable" (13.3).
5. **Trusted base:** owner chose option 1: wait for the 022 batches (B8, B10, T16 and the rest to B13), then designate one clean SHA. No SHA is chosen now, and A2/A3 stay blocked on B13 plus a clean-commit A1 re-run. Live adoption still needs its own explicit go. The Opus review request is drafted after B13.
6. **O-02 pack overlay:** adopted (already committed at HEAD). A7/B8 unblocked.
7. **R7 CI wiring:** best-approach decision in 13.9.
8. **Parallel batches:** T16, B8 and B10 launched together on DeepSeek implement agents with disjoint paths; B11 waits for B8 and final B10 notes, B12/B13 wait for B1-B11.

### 13.9 R7 wiring decision (parent, 2026-11-17)

The drift check is a gate-contract change if made a required CI check, and the
trusted base rejects unadopted definitions, so it is **not** added to `harness/checks.json`
now. It runs locally as `task tui-vendor-check` and as a lefthook pre-commit job
limited to the root packages and nested module. The diff ignores root-level docs and
config (`*.md`, `LICENSE`, `Taskfile*.yml`, `lefthook.yml`) so documentation edits no
longer force a re-vendor; Go sources, embedded skills and pack files stay compared.
Promoting it to a required status check is a later adoption step with the trusted base.

14. **Release environment branch policy (owner, 2026-11-18, amends 12.9 item 12 and 12.7):**
    the first live publish for `v0.14.0` (run `44165821810`) was refused in about two
    seconds with no steps run. `Release publish` is a `workflow_run` workflow, so its
    deployment ref is the default branch `main`, and the `release` environment allowed
    only the tag policy `v*`. Nothing was published (no release, tap still 0.13.0). The
    owner approved adding the branch policy `main` to the `release` environment (policies
    are now `branch:main` and `tag:v*`; the owner stays the required reviewer,
    prevent-self-review off, admin bypass on). The tag-trust design is unchanged: the
    workflow file comes from the default branch by design. This could not be found
    offline; the offline tests did not model environment deployment rules. The failed run
    was re-run (attempt 2) per the workflow's recovery notes, without retagging.

15. **CI gate skip semantics (owner, 2026-11-18):** the second publish attempt for
    `v0.14.0` refused because event-conditional Docs workflow jobs report conclusion
    `skipped` for a tag push, and `release_manifest.py` treated every non-success
    conclusion as a failure. The gate now requires exactly the hosted release checks
    (`quality`, `verify`, `Trusted main validate and build`) to be present and to
    succeed, tolerates `skipped` only for the named non-applicable lanes (PR validate,
    scheduled and documentation observations, Pages deploy/rollback), and still refuses
    failures, cancellations, unknown conclusions, missing required checks and any
    skipped name it does not know. Fixing the gate scripts changes the tooling commit,
    so `RELEASE_TOOLS_SHA` must be re-set to the commit that carries this change
    before the next publish attempt; the publisher workflow file itself is unchanged.

16. **v0.14.0 published and auto-approval scope (owner, 2026-11-18):** the owner
    approved the waiting deployment, and `v0.14.0` published from `effab6e` (release
    assets plus tap cask 0.14.0, tap commit `2026-11-18T07:09:27Z`). The first live
    `Docs observe` run (`44256694768`) then recorded the route evidence and matched the
    binding to the published revision, so the F-02 window fix is confirmed on a real
    release and AC-08 has live evidence for the first time. The owner asked that their
    own pushes publish without a further click. GitHub exposes no REST/GraphQL endpoint
    for a classic environment required-reviewer rule (DELETE returned 404), so that step
    is UI-only: Settings -> Environments -> release -> Required reviewers -> remove the
    reviewer. Until the owner does this, each publish keeps one approval click. The
    branch policies (`branch:main`, `tag:v*`) stay regardless.
