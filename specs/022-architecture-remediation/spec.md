# Feature 022 — Architecture remediation and release assurance

**Revision:** r3, 2026-11-16 (r2 plus plan/tasks reconciliation; see §9). **Status:** draft; not approved.
**Requested output:** spec, plan and tasks drafts; no implementation.
**Origin:** [architecture and feature review, 2026-11-16](../../docs/agentic-sdd-architecture-review-2026-11-16.md).
**Numbering:** the owner selected a new Feature 022, preserving the approved
[Feature 021](../021-execution-cli-verification/spec.md) and its delivery history.

## 1. Outcome and source authority

Make the supported product safe at its mutation, consent and evidence boundaries,
and make release qualification cover what is actually shipped. Resolve the
review's 55 findings in priority order, with reproducible evidence and truthful
record reconciliation rather than a general rewrite.

The [product vision §6–8](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#6-non-negotiable-engineering-principles)
guides explicit authority, reviewable scope and evidence-first closure. It adds
no acceptance scope. The report identifies candidate defects and recommendations;
it is advisory, not approval or a final consequential architecture verdict.
Current code, governing feature contracts and actual owner decisions control.

This is one cross-feature corrective outcome with separately reviewable priority
levels. It is not authorization to execute every priority at once. A later
plan/tasks set must bound implementation by dependency and blast radius; this
spec does not define batches or delegate work.
The requested execution design allows explicitly authorized independent batches
to run concurrently in dependency waves. It does not make all batches ready at
once or remove integration/human checkpoints.

### 1.1 Baseline reconciliation

| Source | What is known at drafting | Consequence for 022 |
|---|---|---|
| Review baseline `c542b4a` | Findings A-01–A-09, B-01–B-10, C-01–C-10, D-01–D-14, E-01–E-12; review-era passes/probes are attributed evidence | Preserve IDs, re-establish applicable findings on the implementation candidate; report line numbers are historical anchors |
| Current HEAD `e4b0d14` | Includes `08c1abd` execution corrections, `4e06b84` execution docs/fixtures, and `e4b0d14` B9 records | The earlier report's “B9 active/uncommitted” observation is no longer the current code-state description |
| [020 §12](../020-documentation-ecosystem/spec.md#12-execution-documentation-and-016-corrective-amendment-2026-11-16) and [B9 evidence](../020-documentation-ecosystem/tasks.md#b9-completion-evidence-and-review-2026-11-16) | B9 delivered; awaiting human review. Adds heartbeat, cleanup locking and interrupt handling; records residual refresh TOCTOU and crash limits | Do not reimplement, accept or relabel B9. A-04 execution portion/A-05 need post-B9 race/ownership verification; A-01's successful-leader-exit case is distinct from interrupt handling |
| [008 approval](../008-core-release-quality/spec.md#approval) and repository guidance | Supported-core selection fence and base-owned gate adoption remain authoritative | Proposed new required lanes need explicit approval/adoption; never bypass the old trusted contract to make the new candidate pass. The owner's 2026-11-07 amendment dropped native macOS runtime evidence as a blocking requirement (Linux CI sufficient; Darwin cross-builds still required); 022 adds no native claim |
| [021 approval](../021-execution-cli-verification/spec.md#11-approval) | Execution contract tests delivered; awaiting human review; release is not acceptance | Preserve command/exit contracts and 021's approval; do not turn this spec into a replacement verification feature |

At drafting, the working tree also had uncommitted changes under `opencode-pack/` and
`website/` that overlap FR-14 and FR-16. The plan must re-baseline those files
before editing them.

CodeGraph was used first for current code discovery: `RunCheck`, `CleanRun`,
`acquireRunLock`, fleet `AcquireLock`, `treeDigest`, MCP transaction locking and
the composed TUI confirmation routes. This establishes scope and overlap, not a
fresh defect verdict. No reproduction or product test was run while drafting.

## 2. Actors, scope and non-goals

### Actors

- **User:** previews or authorizes an install/restore/check through CLI or TUI.
- **Maintainer:** develops fixes, validates isolated candidates and reviews evidence.
- **Trusted verifier:** evaluates candidates using separately reviewed tooling
  and check selections, without candidate-supplied authority expansion.
- **Publisher:** performs separately authorized release operations with scoped
  credentials; does not lend those credentials to verification.
- **Human owner:** approves contracts, adoption, acceptance and live actions.
  No test or reconciliation process substitutes for that decision.

### In scope

- Residual execution, fleet, installer and MCP lock/path/concurrency defects.
- Process-completion truthfulness, backup identity, bounded input and provenance recovery.
- Composed TUI consent/focus/results and related CLI boundary contracts.
- Shipped-surface verification, release hardening and non-publishing reproducibility.
- Consistent portable/pack policy, current documentation and source-backed
  reconciliation of the cited feature/evidence records.
- Small dependency/interface changes necessary to enforce those contracts;
  performance/context changes only with the evidence conditions in §4.3.

### Excluded

- Native ACP/admission/online/runtime-certification work in 005; native/profile/
  dispatch/pilot/C2 work in 006; historical migration/adoption/old-base exercises
  in 007. Their records may be clarified, but their experiments stay excluded.
- New product commands, automatic model failover, provider authentication,
  fleet live-capability expansion, or selecting unstarted Feature 015 B4.
- Real-home apply/restore, deployment, tag/release/push, tap migration, secret
  rotation or live trusted-reference adoption merely because fixes are approved.
- Retrospective consent, inferred acceptance, invented verification transcripts,
  deleted historical failures or forced closure of G-008-01.
- A framework rewrite, cosmetic package splitting, merging distinct evidence
  schemas, global-host configuration edits or copying a personal fleet upstream.
- Claims of sandboxing, escaped-process containment, signing/notarization,
  native macOS/Windows runtime certification, accessibility certification,
  provider availability or measured savings not demonstrated by this work.

## 3. Priority and completion rules

Priority is operational order, not a replacement for the report's severity.

| Priority | Required outcome | Ordering rule |
|---|---|---|
| **P0 — safety and qualification** | Stop destructive/concurrent writes and false completion, ensure visible consent, cover shipped surfaces, harden release privilege and expose unresolved assurance decisions | Must be satisfied or explicitly risk-dispositioned by the owner before a **022-qualified** release or affected live expansion is claimed |
| **P1 — identity, durability and authority** | Make backup/input/provenance interfaces reliable; reconcile routing, evidence, docs and verification environment | Follow relevant P0 ownership contracts; approval-sensitive decisions may block only their affected scope |
| **P2 — bounded maintainability and usability** | Remove demonstrated coupling, improve discoverability, reduce avoidable context/startup work, stabilize verification evidence | Must not delay an independently reviewable P0 correction; do not introduce speculative abstractions |

A-05 is promoted to P0 because cleanup participates in the same exclusion
contract as A-03/A-04. E-03 and B-03 are P1 despite appearing later in the report's
sequence: inconsistent verification scope and false outcome presentation are
correctness issues. C-01/C-02 remain P1 in execution order, but their unresolved
authority decisions must be settled before the affected pack work proceeds.

E-06/E-07 (FR-07) and E-09/E-10 (FR-08) are P1, not P0, after architect review:
reproducibility and observation reliability are qualification properties, not
safety or consent defects. FR-07 remains in scope for any release that claims
reproducibility, and the owner may promote it. Dependencies the plan must respect:
FR-03's lock owner precedes FR-12 (finalization inside the lock); the FR-10 format
decision precedes FR-12 recovery and FR-11 manifest reader limits; FR-04's additive
lanes precede gate-dependent FR-06/FR-07/FR-23 changes; FR-06's permission design
precedes FR-08's trigger wiring; FR-05 precedes FR-19 and FR-21 in the same
composed-app code; FR-16 follows FR-09's owner decisions.

**Finding dispositions:** `open`, `fixed`, `satisfied-by-existing-change`,
`not-reproduced`, `record-reconciled`, `owner-decision-pending`, or
`deferred-by-owner`. Record the exact candidate, applicable requirement/AC,
evidence and residual limits. A fresh reproduction plus regression is required
for `fixed`; existing corrections need equivalent current evidence.
`not-reproduced` is a qualified conclusion, not an automatic fix.
An owner deferral remains unresolved engineering debt, never a pass.

For P0 qualification, no P0 finding may remain unaccounted for. Any remaining
owner-dependent gate is explicitly open or carries the actual owner decision.
For **whole-feature completion**, all included ACs require evidence; an unmet AC
needs an approved scope revision/deferral, not a green label.
Technical completion, human acceptance, trusted-base adoption and publication
remain separate outcomes.

## 4. Requirements

Each FR has its own same-numbered AC in §5. §6 maps every report ID exactly once
to a primary FR/AC; a fix may also support other criteria.

### 4.1 P0 — safety and qualification

**FR-01 — Truthful process completion (A-01).** A successful command leader is
not sufficient to claim completed check evidence while ordinary children remain
active in its owned process group. Bounded termination and completion checks
cover normal leader exit as well as timeout/output-limit/interruption. Surviving
or unobservable owned-group activity cannot produce `verified-evidence` or exit
0. Preserve existing exit-code classes; do not add a new result/schema token
without an explicitly approved compatibility decision. Liveness is determined from
the owned process group id, never from recorded PIDs, which can be reused after the
leader is reaped. The plan states the post-exit grace bound and maps the
surviving-group outcome to an existing check outcome and exit code that is
distinguishable from timeout in the check record. Document limitations of
detached/escaped children and uncatchable crashes; this is not a sandbox.

**FR-02 — Safe lock leaves (A-02).** A symlink, FIFO, directory, device or replaced
lock leaf must not cause acquisition/release to modify a referent or unrelated
file. Validate the actual opened object and ownership, not just ancestors or an
earlier path lookup. A handle must not clear or release another owner's lock.
Open the leaf without following links and without blocking on a FIFO (for example
`O_NONBLOCK` before the regular-file check). Windows is either supported with an
equivalent leaf check or fails closed; the plan states which. Unsupported platforms
fail closed rather than pretending to provide exclusion.

**FR-03 — Shared mutation ownership (A-03, A-04, A-05).** The plan names one owning
package for the cross-writer agent-target exclusion primitive and inventories
writers by their actual destination and supported status. Installer and fleet
writes to global agent profiles share exclusion. MCP configuration writes keep a
separate target domain. The default-disabled `opencodeadapter` project-config
writer is inventoried statically, not enabled, modified or validated through
excluded experiments. Each supported overlapping writer acquires its exclusion before
revalidation and holds it through backup, mutation, rollback and provenance
finalization. Execution run locks remain a separate per-run domain; FR-03 does not
merge them into the target lock. The exclusion key is the canonical resolved
agent-target directory (the home), not the backup root, so two homes under one
backup root never contend. No lock is evicted by elapsed time alone while its
holder is live. Reclaim is conditional on the observed lock identity (token or
inode) still being the one inspected; a kernel-released advisory lock satisfies
this without a heartbeat. The plan chooses the mechanism per lock domain: fleet's
`flock`, MCP's file lock and execution's heartbeat lock have different remedies.
Cleanup and run writers cannot race through deletion or directory recreation.
Preserve dead-holder recovery and nonblocking conflict semantics. Preview stays
non-mutating. Do not deadlock nested installer/fleet operations.

**FR-04 — Required coverage of shipped surfaces (E-01, E-02).** Define a
reviewed qualification contract covering:

- Existing eleven-package root tests/vet/module integrity.
- Nested TUI tests/vet/module integrity on its supported toolchain.
- The two explicitly permitted skill-workflow Python suites and individually
  reviewed supported-core trusted-gate regressions; no broad discovery.
- All four shipped names, both Darwin architectures, shared build identity,
  paired-archive members and checksum integrity. Retain existing `sdd-check`
  coverage; do not count it as one of the four shipped applications.
- Native-host subprocess smoke for the launcher/identity contract, distinct
  from Darwin compile/archive inspection and unperformed native macOS execution.

Missing prerequisites block the affected qualification rather than silently
skipping it. Required failures block release publication; PR required-status
enforcement is claimed only if separately observed. Preserve the base-owned
anti-weakening boundary; new interpreter/tool permissions require reviewed
adoption, not arbitrary candidate-controlled command execution.

**FR-05 — Visible, focus-owned consent (B-01, B-02).** MCP and Fleet apply/restore
confirmations obey the same shell-level consent contract as install/restore:
action, target, consequences and decision controls are visible at usable sizes;
otherwise affirmative input cannot dispatch. Resize/scroll must not expose a
hidden authorization path. Confirmation owns relevant keys before global
navigation; advertised Tab/selection/cancellation behavior is truthful. Default
No, explicit confirmation and existing write tokens remain required. Preserve
Fleet's current capability gate; test gated behavior through isolated doubles.

**FR-06 — Least-privilege release preparation (E-04, E-05).** Verification has
no publish/tap credentials or unused OIDC/write permission; checkout does not
leave reusable credentials. Third-party release actions are pinned to reviewed
full commit SHAs. Publication is serialized, bounded and consumes evidence bound
to the same source/artifacts; a failed verifier cannot publish. Retain diagnostic
evidence without secrets. Distinguish token-read checks from write authorization;
document partial-release/tap recovery and manual reconciliation without requiring
a test push or broader credential. Publication itself remains separately authorized.

**FR-09 — Honest assurance and acceptance disposition (D-01, D-02, D-03,
D-07).** Reconcile G-008-01, 014 integrated AC/finding evidence, owning publication
consent and 016/021 release-versus-`not_run` records. Retain historical limitations
with dates. Obtain actual missing owner decisions or state that they remain
pending. Publication does not prove acceptance; tag ancestry does not establish
consent; a pre-integration report is not integrated evidence. No closure or
Fleet-live expansion is inferred from 022 approval. Current engineering evidence
and approval-sensitive decisions are reported separately.

### 4.2 P1 — identity, durability and authority

**FR-07 — Reproducible build and snapshot identity (E-06, E-07).** With identical
source, pinned toolchain/dependencies and build inputs, repeat non-publishing
builds produce identical application bytes and deterministic archive/checksum
content. Runtime wall-clock timestamps must not vary identity. Public and
experimental snapshot entrypoints have one documented policy for prerequisites,
external output, offline limits and source identity. Dirty snapshots are refused
or carry an actual content identity, never a misleading commit-only identity.
Do not revive excluded experimental validations through snapshot consolidation.
The `version.Date` ldflag fed by `{{ .Date }}` is the wall-clock source to replace
with a commit-bound value; the plan cites the GoReleaser template-field page for
the exact definition.

**FR-08 — Reliable release/site observations (E-09, E-10).** Post-release
observation is explicitly reachable when publication uses `GITHUB_TOKEN`; it does
not rely solely on a suppressed `release: published` event or a new broad PAT.
Its trigger design is chosen inside FR-06's permission budget and shares one
release-workflow review with it. Asset resolution handles root-relative and
relative URLs without duplicating the repository base path. Read-only observations
remain clearly distinguished from publication prerequisites; transient CDN delay
is not invented deployment failure, and tolerated failures remain visible rather
than silently successful.

**FR-10 — Unambiguous backup identity (A-06).** Different supported tree
structures/contents cannot share a digest because of ambiguous serialization.
Identity includes entry kind, path and content boundaries, in deterministic
order. New backups have an explicit digest-version discriminator; existing
backups remain verifiable/restorable using their original declared semantics.
Unsupported versions are rejected, never silently reinterpreted. Legacy identity
limitations remain documented, not relabeled cryptographic security. The
discriminator is an additive per-entry field (for example `digest_version`); an
absent field means legacy. Legacy and new digest functions coexist, `TreeDigest`
stays exported for fleet, and fleet's provenance sidecar is versioned the same way.
Replacing the legacy function in place is not acceptable.

**FR-11 — Bounded regular-file ingestion (A-08, B-05).** Backup/fleet manifests
and `run record --result FILE` reject unsafe file types and bound reads before
unbounded allocation or blocking on special files. Validate object identity
across opening/replacement where supported; refuse an unsafe fallback. Keep
bounded stdin support, strict parsing and redacted diagnostics. The later plan
must name byte limits and compatibility behavior for all affected readers.

**FR-12 — Recoverable provenance and owned mutation interfaces (A-07, A-09).**
Failure between file installation and metadata finalization yields an explicit,
recoverable outcome, not false success or silent complete rollback. Sidecar/live
pointer recovery detects interrupted or inconsistent finalization without
overwriting newer target bytes. Public mutation entrypoints establish/check their
own exclusion and before-state; callers cannot silently omit a private lock or
supply stale `Old` bytes as authoritative backup content. Preserve typed results,
compatibility boundaries and the distinction between process-crash recovery and
unverified power-loss guarantees.

**FR-13 — Truthful result and help projection (B-03, B-06).** MCP root/child
views retain structured no-op, backup, rollback, partial and log-error facts,
including results accompanying errors. A no-op does not claim a new backup and a
log failure is visible. Root/topic help and generated website references disclose
actual MCP writes, opt-in pack flags and accepted topics. Remove unjustified
parity exceptions, not safety warnings or script output compatibility.

**FR-14 — Portable routing and reviewer authority (C-01, C-02, C-06, C-09).**
Name a supported upstream owner for model/profile defaults and distinguish
personal overrides. Routing tables, shipped profile identifiers, adapters,
fallback text and writer inventory agree with it; availability remains an
unverified runtime fact. C-09 (writer inventory) is not blocked by O-02/O-03 and may proceed independently.
Do not import host renames/pins merely because they are
installed. Separate ordinary advisory review from the strict independent method;
each advertises and permits only compatible tools. Ordinary scoped correction
review follows core policy within existing authority; a new independent/model
dispatch remains a separate authorized action. No online certification of 006
or relaxation of its constraints is included.

**FR-15 — Read-only discovery and bounded research (C-03, C-08).** CodeGraph
remains first choice for an existing suitable index, with staleness handling.
Index creation/sync that mutates a checkout requires matching write authority;
review-only work falls back to targeted reads without silently bootstrapping.
Portable and pack guidance agree. Research trigger/body text agrees on whether
an explicit broad request must name an SDD decision; this clarification does not
authorize new files, background research, spending or external disclosure.

**FR-16 — Canonical record and documentation reconciliation (C-04, C-05,
C-10, D-04, D-05, D-06, D-08, D-09, D-10, D-11, D-12, D-13, D-14, B-10).** Current
navigation links to the website/manual source and marks dated research/retired
guides as historical. Preserve usable historical anchors and evidence paths.
Reconcile the cited approval headers, accepted task leaves, continuation notes,
B8 delivery statements, AR dispositions, dead links and release qualifiers with
current sources and actual decisions. Separate parent-run tests from
inspection-only reviewer evidence; state absent early approval provenance
without inferring misconduct. Update summaries by reference, not a second
approval ledger. Resolved handovers may be folded only when authoritative
pointers survive; never erase negative evidence or manufacture acceptance.

**FR-17 — Consistent verification scope and environment (E-03, E-08).**
Local verification bootstrap, invoked checks and automated gates use a defined
environment/selection policy: inherited `GOFLAGS`, `GOENV`, `GOWORK` and related
controls cannot silently broaden the approved suite or select excluded work.
Trusted staging remains bounded, rejects unsafe candidate entries, and handles
reviewed generated-output directories without trusting candidate ignore rules.
Enumerate each `go` entrypoint and show it pins the environment; the Taskfile already
does for `test` and `vet`, so the gaps are elsewhere, and the criterion names them.
FR-16 owns the 019 aggregate-test attribution, preserving admissible scoped
results and explicitly labeling inadmissible/unverified claims.

**FR-18 — Strict receipt termination (E-12).** The receipt-result reader accepts
exactly one schema-valid JSON document plus permitted trailing whitespace.
Trailing malformed data, extra documents and duplicate keys are rejected with
bounded diagnostics. Preserve existing valid receipts and the distinction
between consistency validation and approval.

### 4.3 P2 — bounded maintainability and usability

**FR-19 — Explicit dependency seams (B-04).** Command inputs/services and touched
journey state become explicit enough to test independent invocations without
process-global substitution. Define mutually exclusive confirmation/operation
states for the touched MCP path and retain one write host. Extract only seams
needed for these contracts; do not require package count, a framework or a
wholesale rewrite. A-09's lock/before-state safety remains FR-12, not optional
cleanup. FR-05 and FR-13 change only the minimum seam they need; FR-19 completes the
remaining touched paths. A new process-global lock or registry is not an acceptable
seam.

**FR-20 — Stable help and real-wrapper contracts (B-07, B-08).** Specify and test
the existing help/version ordering exceptions or an explicitly approved narrow
compatibility correction. Real link-time legacy wrappers preserve canonical
stdout bytes, put only the appropriate deprecation notice on stderr, forward
nonzero exits and `--` arguments, and retain lifecycle coverage. Separate-pipe
subprocess evidence complements, not replaces, PTY evidence.

**FR-21 — Nonblocking startup observation (B-09).** Initial inventory/activity
reads must not prevent a useful loading/empty frame. Delayed/error/stale results
do not overwrite a newer route/home generation. Measure a reproducible
slow-source fixture before/after; claim no unmeasured global performance gain.

**FR-22 — Proportionate skill context (C-07).** Lightweight explanation/teaching/
handoff branches load only the authority and mode constraints they need plus
conditional lifecycle references. Preserve no-write/approval/privacy boundaries,
sibling link closure and explicit completion outputs. Demonstrate reduced
mandatory loaded words on representative branches; this is not a claim of saved
subscription tokens or better model behavior.

**FR-23 — Measured verification reliability (E-11).** Establish cold-run timing
and explicit supported-core race coverage, including the nested TUI where the
selected lane supports it. Separate timeout, failure and unavailable prerequisite
outcomes. Synchronization-sensitive regressions use observable handshakes rather
than tiny sleeps. Budget changes need measured justification; do not add retries,
skips or tolerances to turn defects green.

## 5. Observable acceptance criteria

Commands/tool versions, exact fixtures, byte limits and selected suite inventory
belong in the subsequent plan/tasks. These ACs define outcomes, not a choice of
locking library, serialization algorithm or package layout.

| AC | Acceptance example / evidence required |
|---|---|
| AC-01 | A fixture leader exits 0 after launching a redirected same-group child. Check evidence cannot pass while that child runs; bounded cleanup/uncertainty is observed and recorded. Include interruption/timeout/output-limit/no-child regressions and documented escaped-group limits. |
| AC-02 | Pre-existing symlink/FIFO/directory lock leaves and a controlled replacement cause safe refusal without changing sentinel referent bytes. Releasing an old handle does not clear a successor; normal lock/unlock still works on each claimed platform. |
| AC-03 | Barrier-controlled installer-versus-fleet apply/restore, live-old-owner, stale-reclaimer-versus-new-owner, cleanup-versus-writer and directory-recreation cases allow only the rightful writer or a truthful conflict. A paused holder cannot lose ownership by age. Dead-holder recovery and independent-home progress pass. Lease-based domains expose a test clock/window; kernel-held domains instead test that age does not affect ownership. No fixture waits out a production stale period. |
| AC-04 | A reviewed gate inventory names every required surface; deliberately failing nested-TUI/Python fixtures, missing archive names, wrong slices and mismatched pair identity each fail the appropriate gate. One immutable candidate passes the qualification set if it is adopted; unobserved PR protection/native macOS remains unclaimed. |
| AC-05 | Composed-app tests at 100×16 MCP and 100×20 Fleet restore, plus smaller/resized/scrolled layouts, show no affirmative dispatch with hidden essential content. Tab, global shortcuts, Enter, y, No and Esc obey focus/default-No rules. No real write is dispatched. |
| AC-06 | Offline workflow/permission tests show SHA pins, no retained checkout credential/unused OIDC, read-only verification, publisher-only secrets and failure-gated publication. Concurrency/timeouts/evidence retention and a partial-release recovery walkthrough are documented. No live credential probe/publish is needed for local acceptance. |
| AC-07 | Two clean temporary builds with identical pinned inputs produce identical four-application bytes and archive/checksums. Dirty-source snapshot tests refuse or expose changed-content identity. Both snapshot entrypoints report the same prerequisites and never publish. |
| AC-08 | Synthetic workflow-event tests show the GITHUB_TOKEN publication route reaches observation; root/relative CSS/JS URL fixtures resolve the base path once. Delayed/failed observations remain recorded without silently becoming successful or a new blocking CDN policy. |
| AC-09 | Source-backed amendments distinguish actual release decisions, pending acceptance and dated not-run facts. 014 has current integrated AC/finding dispositions, or an explicit blocker. G-008-01 stays visibly open until a genuine owner designation/adoption decision with evidence is recorded. |
| AC-10 | The reported ambiguous two-tree fixture has different new-format digests. Empty/directory/nested/non-ASCII/path-boundary cases are deterministic; original-format backups still validate/restore under original semantics; unknown versions fail safely. |
| AC-11 | Each affected reader rejects oversized input at limit+1 and FIFO/symlink/device/directory/replacement fixtures without unbounded blocking/allocation; supported regular files and bounded stdin work. Limit and diagnostic tests match the documented compatibility policy. |
| AC-12 | Faults before/after file commit, sidecar write and live-pointer finalization produce accurate outcomes; recovery is repeatable, detects newer targets and does not overwrite them. Calls with absent lock ownership/stale before-state are safely handled without caller access to private helpers. |
| AC-13 | MCP root/child results agree for no-op, applied, rollback, incomplete backup and log-failure cases. Root/topic/generated help lists actual write surfaces/flags; piped structured output remains contract-compatible. |
| AC-14 | A source-backed pack inventory agrees on profile/adapter/model references, allowed fallback lanes and writers. Advisory versus strict-review scenarios have no tool-contract contradiction. Correction re-review and separately authorized dispatch have distinct, consistent rules. Host config is untouched. |
| AC-15 | In an unindexed disposable checkout, a read-only-review scenario leaves no index/cache artifacts. Existing-index discovery and stale fallback remain usable. Research description/body examples select one unambiguous authorized route and do not grant additional effects. |
| AC-16 | Every cited current-state contradiction/link/AR claim has a fix or source-backed pending disposition; current navigation reaches real pages, historical evidence stays dated, and reviewer-executed versus parent-executed tests are accurately labeled. Missing original consent evidence remains explicitly missing. 019's broad-suite claim is reconciled without selecting excluded experiments. |
| AC-17 | Hostile inherited Go controls cannot change selected scope. Fresh-export and generated-output staging fixtures behave consistently; unsafe or arbitrary ignored candidate material cannot bypass checks. |
| AC-18 | Valid receipt JSON with trailing whitespace passes; malformed tail, second object and duplicate keys fail. Existing valid fixtures remain accepted and do not become approval evidence. |
| AC-19 | Two independently configured invocations/test instances can exercise the touched command/MCP paths without swapping process globals or sharing draft/confirmation state; public command and one-host write contracts remain unchanged. |
| AC-20 | Real canonical/legacy binary pairs exercised with separate pipes preserve JSON/stdout, deprecation-only stderr, `--` argument vectors and nonzero exits. Chosen help/version exceptions have exact expected code/stream tests and documentation. |
| AC-21 | A deliberately stalled inventory/log source still permits a useful initial frame; a later result for an old generation is ignored and an error is visible. Before/after fixture timings and environment are recorded, without an arbitrary performance threshold. |
| AC-22 | Representative lightweight skill branches load fewer mandatory words than baseline while authority/no-write/privacy assertions, completion outputs and all installed sibling links remain valid. Static evidence is not labeled runtime compliance. |
| AC-23 | Cold-run measurements and race results are tied to candidate/environment/selection; justified budgets are recorded, no new silent retries/skips exist, and concurrency fixtures synchronize deterministically. Unavailable race/platform tooling is not counted as pass. |

## 6. Complete finding-to-priority traceability

IDs and original severity stay in the [source report](../../docs/agentic-sdd-architecture-review-2026-11-16.md#3-findings-by-lane).
This table is the scope map, not a batch ledger or a claim that any finding is fixed.

| Finding | Priority | Primary FR / AC | Boundary / disposition required |
|---|---|---|---|
| A-01 | P0 | 01 | Successful-leader descendants; not covered merely by B9 signal handling |
| A-02 | P0 | 02 | Fleet lock-leaf object safety |
| A-03 | P0 | 03 | Installer/fleet shared target exclusion |
| A-04 | P0 | 03 | Execution post-B9 residual and MCP lifetime/reclamation |
| A-05 | P0 | 03 | Verify B9 cleanup correction and remaining deletion/recreation race |
| A-06 | P1 | 10 | Versioned tree identity and original backup compatibility |
| A-07 | P1 | 12 | Crash-visible provenance reconciliation |
| A-08 | P1 | 11 | Manifest object/type/size safety |
| A-09 | P1 | 12 | Lock/before-state obligations owned by mutation entrypoints |
| B-01 | P0 | 05 | Visible confirmation across journeys; retain Fleet capability gate |
| B-02 | P0 | 05 | Composed focus/navigation ownership |
| B-03 | P1 | 13 | No-op/backup/rollback/log distinctions |
| B-04 | P2 | 19 | Narrow invocation/MCP seams, not a flat-package rewrite |
| B-05 | P1 | 11 | Bounded file record input |
| B-06 | P1 | 13 | Root/topic/generated help truthfulness |
| B-07 | P2 | 20 | Explicit help/version compatibility policy |
| B-08 | P2 | 20 | Actual wrappers, separate stdout/stderr |
| B-09 | P2 | 21 | Initial frame and generation-safe observations |
| B-10 | P1 | 16 | Historical TUI/release text subordinate to current guidance |
| C-01 | P1 | 14 | Upstream policy decision, not wholesale host copying |
| C-02 | P1 | 14 | Distinct advisory/strict independent tool contracts |
| C-03 | P1 | 15 | Read-only CodeGraph discovery without bootstrap writes |
| C-04 | P1 | 16 | Site migration/index pointers |
| C-05 | P1 | 16 | Dated research framed as historical |
| C-06 | P1 | 14 | Scoped re-review versus new dispatch authority |
| C-07 | P2 | 22 | Measure mandatory context without weakened constraints |
| C-08 | P1 | 15 | Unambiguous research trigger/body |
| C-09 | P1 | 14 | Accurate shipped writer inventory |
| C-10 | P1 | 16 | Release qualifiers remain evidence-specific |
| D-01 | P0 | 09 | G-008-01 owner decision remains explicit |
| D-02 | P0 | 09 | Actual later publication consent in owning records |
| D-03 | P0 | 09 | Time-scope contradictory release/not-run statements |
| D-04 | P1 | 16 | Recheck B8 delivery statements after B9 commits |
| D-05 | P1 | 16 | Approved records have current approval pointers |
| D-06 | P1 | 16 | Accepted leaves reconciled only to actual acceptance |
| D-07 | P0 | 09 | Integrated 014 evidence, no automatic acceptance/live gate |
| D-08 | P1 | 16 | AR-1–AR-10 explicit fix/deferral/pending dispositions |
| D-09 | P1 | 16 | Fence-compliant attribution of historical validation |
| D-10 | P1 | 16 | Resolve stale current-state handover/index text |
| D-11 | P1 | 16 | Preserve authoritative/history links |
| D-12 | P1 | 16 | Do not attribute parent tests to shell-denied reviewers |
| D-13 | P1 | 16 | Explicit missing provenance, not invented consent |
| D-14 | P1 | 16 | Evidence-bound release-facing pointers |
| E-01 | P0 | 04 | Required nested module and shipped-pair/archive surfaces |
| E-02 | P0 | 04 | Individually approved Python suites, no broad discovery |
| E-03 | P1 | 17 | Shared explicit verification environment |
| E-04 | P0 | 06 | Release SHA pins and least privilege; `go-task/setup-task` and `goreleaser-action` are already SHA-pinned, so checkout and setup-go remain (partly satisfied by existing state) |
| E-05 | P0 | 06 | Serialization, budgets, evidence and partial-publish recovery |
| E-06 | P1 | 07 | Fixed-input build/archive reproducibility |
| E-07 | P1 | 07 | One nonpublishing snapshot identity contract |
| E-08 | P1 | 17 | Trusted generated-output staging policy |
| E-09 | P1 | 08 | Reachable observation without suppressed-event dependency |
| E-10 | P1 | 08 | Base-path-safe assets and truthful observation |
| E-11 | P2 | 23 | Measured budgets and selected race coverage |
| E-12 | P1 | 18 | Strict JSON EOF/duplicate-key validation |

## 7. Compatibility, security and verification constraints

1. Preserve canonical `asdx`/`asdx-core`, legacy wrapper behavior, repository/
   cask naming and state/environment names. Root stays stdlib-only; the TUI
   stays a separate supported module.
2. Preview remains the default; no-op does not create backups; unrelated skills
   remain untouched. Restores and rollback cannot overwrite another transaction.
3. Preserve existing CLI result/exit contracts unless a narrow change is called
   out for combined approval. Backup digest compatibility must be explicit;
   fixing serialization must not silently invalidate historical backups.
4. No automatic provider/network/model calls in tests. Use disposable homes,
   repositories, process fixtures and local action/release doubles. Keep raw
   reports/scratch outside backups and uploads; never include credentials.
5. Keep the exact existing core gate until a reviewed replacement/additive
   contract is explicitly approved. Trusted-base adoption uses independent
   authority; a candidate cannot designate itself trusted. Document required
   but unavailable aggregates as `blocked`/`not_run`.
6. Go changes require `task test`, `task vet`, `git diff --check`, affected
   scoped race/negative/concurrency checks and the new selected lanes after
   adoption. Content changes retain the two explicitly selected workflow suites;
   website/CI changes also retain their applicable existing source/built/reference
   and trusted-policy checks. No broad harness discovery or root `go test ./...`.
7. Verification records commands, candidate/tree identity, environment, timing,
   outcome and limitations. Review-era counts/coverage are not current targets.
   A static policy test is not a runtime permission test; cross-build is not a
   native execution claim; configuration is not observed branch protection.
8. Consequential lock/identity/gate/routing designs need the repository-required
   Opus architecture verdict before implementation is treated as ready. That
   review is not performed or delegated by this drafting request.
9. Gate adoption is sequenced, not circular. New required lanes (FR-04) land
   additively behind the existing gate, which passes the candidate that adds them.
   A lane becomes required only after separate base adoption (O-01). Changes to
   release, build and budget gates (FR-06, FR-07, FR-23) are verified by the old
   gate plus their own offline fixtures until that adoption.

## 8. Decisions to resolve before affected planning/approval

These are proposals or existing constraints, **not owner decisions made here**.
They do not block delivery of this draft, but the dependent plan must not guess.

| ID | Decision | Proposed default / effect if unresolved |
|---|---|---|
| O-01 | Exact additive qualification suites, toolchains, prerequisites and trusted adoption path | Preserve the 008 exclusion fence; add reviewed TUI/archive/named-Python lanes, not experimental core expansion. Blocks FR-04 implementation/adoption. |
| O-02 | Authoritative upstream model/provider/profile defaults | Keep upstream separate from personal overrides; inspect available sources without live provider calls. Blocks FR-14 routing edits; host policy is not automatically product policy. |
| O-03 | Ordinary versus strict independent reviewer profiles | Keep ordinary advisory tooling and strict independent constraints explicitly separate; no claim of runtime-certified isolation. Blocks FR-14 profile/permission edits. |
| O-04 | G-008-01 designation, 014 closeout and pending feature acceptance | Keep pending until actual decisions/evidence exist. Blocks the affected assurance claim, not unrelated safety fixes. |
| O-05 | Backup digest discriminator and original-format restoration boundary | New writes use unambiguous versioned identity; retain old-format reads with disclosed limits. Plan must settle format/API compatibility before FR-10 implementation. |
| O-06 | Lock-domain owner and mechanism per domain; process liveness and outcome mapping; supported OS guarantees | The plan names the cross-writer target-exclusion owner and writer inventory; fleet `flock`, MCP file lock and execution heartbeat lock are chosen per domain; liveness uses the process-group id; Windows is supported or fails closed; no new exit/schema token by assumption. Blocks FR-01–03 and FR-12 design. |
| O-07 | Reader byte limits, special-file policy and help-order exceptions | Bound regular file inputs; preserve bounded stdin. Keep current help ordering unless a narrow change is explicitly approved. Blocks FR-11/FR-20 compatibility decisions. |
| O-08 | Timing/word-load baselines and acceptance scope | Use deterministic slow-source/context fixtures, not claimed savings. A safe measured no-change decision is a finding disposition, not proof that FR-21/FR-22 was implemented; revise scope if those outcomes are deferred; deferring them is acceptable without penalty. |

## 9. Approval

- **Combined approval (2026-11-16):** the repository owner said “accepted” after
  reviewing spec r3, plan r1 and tasks r1, then “start implementation using sub
  agents”. Recorded for exactly those revisions. The plan §2 proposals are accepted
  as working defaults **except** O-01 (real trusted-base adoption), O-06
  group-identity/legacy-lock migration and O-05 format details, which remain gated
  by the required Opus verdict.
- **Authorized now:** B0 parent preflight; W1 members **B3** (Claude Agent Haiku)
  and **B5** (9Router Gemini 4 Hydrogen) only. B1, B2 and B4 wait for the Opus
  verdict. Later waves wait for W1 human review.
- **Not authorized:** commit, push, tag, release, real-home apply, live trusted
  adoption, paid Go escalation, or acceptance of any batch.
- **Review pass (r2, 2026-11-16):** read-only architect review and spec research
  review, both on `claude-agent/sonnet#medium`, reports under
  `/tmp/opencode/022-spec-review/`. Applied: a lock-domain owner and writer
  inventory (FR-03, O-06); identity-based reclaim and home-keyed exclusion; an
  additive `digest_version` discriminator (FR-10); process-group liveness and
  outcome mapping (FR-01); gate-adoption sequencing (§7.9); FR-07 and FR-08 moved to
  P1; C-09 unblocked from O-02/O-03; FR-17 and FR-19 narrowed; source qualifiers
  for E-04 (partly satisfied) and GitHub/GoReleaser claims. Counts are now P0 15,
  P1 34, P2 6. Nothing here is owner approval.
- **Drafting authority:** the owner requested a priority-ordered spec addressing
  the report, then selected **“New Feature 022 (Recommended)”** when told 021
  already existed. This authorizes this spec draft, not implementation.
- **Plan/tasks request (2026-11-16):** the owner requested plan/tasks designed
  for parallel implementation with Claude Agent Haiku, DeepSeek Flash High and
  9Router Gemini 4 Hydrogen implement subagents. This authorizes document
  drafting and those proposed assignments, not worker launches or implementation.
- **r3 reconciliation:** writer domains respect the 008 fence; D-09 has one
  primary owner (FR/AC-16); kernel-held locks need no artificial stale clock.
  All 55 findings remain included with unchanged priority counts.
- **Next checkpoint:** review [plan r1](plan.md) and [tasks r1](tasks.md) with
  this spec for one combined approval. Review sensitive designs under §7.8.
- **Implementation/batches:** see the authorized-now list above; nothing else.
- **Live actions:** no real-home writes, trusted-base adoption, branch protection
  changes, credentials, commit/push, tag/release, tap publication or deployment
  authorized. Existing feature approvals/acceptances remain unchanged.
- **Progress and owner-acceptance pointer (2026-11-17):** the B0 Opus verdict is
  recorded in [plan §12](plan.md#12-opus-verdict-and-conditions-frozen-at-b0-r2-2026-11-16);
  2026-11-17 owner decisions and batch acceptances are recorded in
  [plan §12.7–§12.9](plan.md#127-owner-decisions-recorded-2026-11-17) and the
  [tasks batch ledger](tasks.md#2-batch-ledger-and-dependency-waves) (B2, B3, B4
  T07/T09, B5, B7 T15, B9 accepted; B1, B4 container overlay/TUI vendoring/test
  fixes, B6 T12, B7 T14 awaiting human review; B6 T13 in progress; D-01/D-02/D-07,
  O-01 real adoption, help/probe conflict, container overlay decision and
  `RELEASE_TOOLS_SHA` remain open). Feature-level approval status above is unchanged.

## 10. Sources and fact-check boundaries

- [Architecture review](../../docs/agentic-sdd-architecture-review-2026-11-16.md):
  finding IDs, priorities, source anchors and evidence limitations.
- [Repository guidance](../../AGENTS.md): source ownership, safety, verification,
  acceptance and the temporary 008 validation fence.
- [020 §12](../020-documentation-ecosystem/spec.md#12-execution-documentation-and-016-corrective-amendment-2026-11-16),
  [B9 evidence](../020-documentation-ecosystem/tasks.md#b9-completion-evidence-and-review-2026-11-16),
  commits `08c1abd`, `4e06b84`, `e4b0d14`: current correction boundary, not acceptance.
- [021 approval/release](../021-execution-cli-verification/spec.md#11-approval):
  existing contract and why this work uses a new feature number.
- Official GitHub documentation, retrieved through Context7 on 2026-11-16:
  [GITHUB_TOKEN event recursion limits](https://docs.github.com/en/actions/concepts/security/github_token),
  [secure use / least privilege](https://docs.github.com/en/actions/reference/security/secure-use),
  [workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax).
  Supports explicit post-release wiring, scoped permissions, and full-SHA pinning as
  strong guidance (tags remain allowed for trusted creators). `GITHUB_TOKEN`
  exceptions apply only to events that token triggers, not to a human or PAT release;
  does not establish effective repository settings.
- Official GoReleaser documentation, retrieved through Context7 on 2026-11-16:
  [Go build reproducibility](https://goreleaser.com/customization/builds/go/),
  [reproducible builds](https://goreleaser.com/blog/reproducible-builds/).
  Explains commit-bound timestamps and fixed build inputs; byte equality still
  needs candidate-specific evidence. No new external-library fact required Exa.
