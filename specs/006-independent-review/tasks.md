# Independent review tasks and continuation

Revision: r1 · 3 November 2026. Inputs: [spec r2](spec.md), [plan r1](plan.md).
Single approval location: [spec §Approval](spec.md#approval).
Combined approval recorded 4 November 2026. T01 (B1) and T02 (B2) were reviewed
and published. B3 (T03/T04) is delivered for its offline scope; T05/B4 is
deferred by the user, not passed or accepted. `IR-06-01` remains a valid native
access failure, so generic native review of a checkout is unsupported.

## Execution boundary

Execute one selected batch only after combined approval and batch authorization,
verify it, return one completion review, record `awaiting human review` and stop.
Plan contracts C1–C5 own shared details; specialist/tool assignments live there.
No task triggers real inference, VM access, real-home writes or release implicitly.
Existing deterministic commit hooks/CI remain enabled and distinct from a pilot.

## Ordered tasks

### T01 — Integrate the portable independent-review method

- **Status:** reviewed by the user and published (see [spec §Approval](spec.md#approval));
  the original B1 self-review was `Ready` on its portable scope, with no model calls.
  **Covers:** AC-01/02/05/06/07/08/09. **Contracts:** C1/C3.
- **Objective:** own focused context, evidence distinctions, bounded findings and
  reconciliation once, behind explicit independent-review invocation.
- **Targets:** new router reference `independent-review.md`, existing
  `review-resolution.md` and verification-review `SKILL.md` conditional pointers.
- **Consumes/produces:** existing review ownership and spec scope → portable
  parent/child method consumed by T02 profiles and T03 scenarios. Keep 18 skill IDs.
- **Checks:** inspect direct review without specs; actual dirty/deleted inputs;
  missing/truncated evidence; user request versus role/handoff suggestion;
  disputed/needs-evidence findings; review-only no file/state changes; one review
  owner and no automatic fix/review loop. Check local references and frontmatter.
  Source/procedure inspection is not agent-compliance evidence.
- **Complete when:** every selected branch has a bounded outcome and existing
  review rules retain a single owner; no broad instruction rewrite or new authority.

### T02 — Add inactive native profiles and exact-runtime boundary fixtures

- **Status:** reviewed by the user, 4 Nov 2026 (B2 implemented; offline probes only,
  no prompt or model). Depends on T01. **Covers:** AC-01/03/04/06/09.
  **Contracts:** C2/C4.
- **Objective:** distribute a separate coordinator/reviewer example with explicit
  model binding, deny-first permissions and no default candidate regression.
- **Targets:** new `integrations/opencode/independent-review.jsonc`; focused new
  tests under `tests/harness/`, using the existing isolated native-probe pattern.
  Do not modify the production generator/validator, default config or ACP admission.
- **Consumes/produces:** T01 procedure and pinned runtime identity → profile and
  per-route supported/blocked observations consumed by T03/T04.
- **Checks:** static model/variant/mode/permission/step expectations from independent
  constants; own child rules; unknown tools denied; no shell, edit variants,
  browser, MCP/resources, Code Mode or nested children; required reads/skill allowed;
  glob/grep denied; parent grants only named child. Inspect config merge/global
  denial collision and invalid settings. Separate permission evaluation from tool
  dispatch. Verify read manifests and symlink escape in claimed path routes.
- **Native fixture boundary:** exact 2.2.5 digest/source, fresh HOME/XDG state,
  network disabled, non-root process, no host credentials/socket or writable source;
  no prompt/model API, primary database, shared service or provider probing.
  Explicit B2 authorization must cover these offline native probes; missing local
  binaries/images stop them rather than causing downloads or alternate versions.
- **Complete when:** example stays inactive and defaults unchanged; offline
  observations substantiate or specifically block each claimed boundary. An actual
  unsafe permission result is a defect, not a passing negative test. No live support
  certification is claimed. Unavailable prerequisites remain incomplete/blocked.

### T03 — Verify context, findings, freshness and portable distribution

- **Status:** accepted by the user for the record, 5 Nov 2026 (B3, with the fixture
  follow-up; the findings, freshness and scheduling gap below is closed for offline
  scope). Depends on T01/T02.
  **Covers:** AC-01/02/05/06/07/08/09. **Contracts:** C1/C3/C5.
- **Objective:** exercise the component contracts together without a model and
  verify portable distribution, independently derived expectations and preserved state.
- **Targets:** focused `tests/harness/` fixtures; `internal/skillsync/package_test.go`
  for new-reference embedded/on-disk presence and existing install semantics.
- **Consumes/produces:** method/profile, synthetic candidate/evidence snapshots
  and independent oracles → development evidence and T04 pilot inputs.
- **Checks:** staged/unstaged/untracked/deleted candidate coverage; unrelated dirty
  sentinels preserved; missing/stale evidence; changed code/config invalidating
  review; valid/disputed/needs-evidence handling; bounded error/interruption result;
  no inference dispatch, duplicate scheduling or automatic Ready. Scripted child
  outputs test consumers only, not model quality. Preserve symlink/special-file
  refusals, exact 18 IDs and install/preview/no-op/backup/restore behavior.
- **Verification:** focused fixture suite; `make test`, `make vet` and
  `git diff --check` for Go test changes. No phrase assertion is reported as proof
  that a model follows the method. Reuse unchanged valid installer checks.
- **Complete when:** independent oracles catch seeded contract regressions and
  actual distributed content/links match; component failure stays visible.

### T04 — Prepare reversible usage and the final RHEL pilot

- **Status:** awaiting human review (B3, 4 Nov 2026); pilot not launched. Depends on
  T02/T03. **Covers:** AC-03/04/06/09/10. **Contracts:** C2/C4/C5.
- **Objective:** give the user one coherent opt-in path, including blocked
  prerequisites and a final pilot proposal, after development is delivered.
- **Targets:** new `integrations/opencode/independent-review.md`, integration
  README; navigation only where present in the published candidate; offline fixture
  construction/oracle material in `tests/harness/`; this continuation/handover.
- **Consumes/produces:** T02 boundary evidence + T03 integrated checks → exact
  preview/backup/rollback procedure, isolated fixture setup and a bounded pilot brief.
- **Checks:** demonstrate scratch-only digest-checked no-op/backup/restoration,
  intervening edit refusal and preservation of unrelated config. Keep fixture
  answer keys out of all reviewer-readable paths, including directory discovery.
  Inspect source/default config differences and declared unsupported modes.
- **Pilot preparation:** fill only verified fields. Request the minimum unresolved
  RHEL target/user/workspace, current exact provider eligibility, scoped secret
  reference, accounting/bounds and cleanup owner; do not access/provision VM or
  credentials to obtain them. Use the numerical C4 proposal, never silently widen it.
  Missing enforcement means a blocked proposal, not an automatic launcher project.
- **Complete when:** development and per-AC limits are reconciled and the final
  pilot can be accepted for a bounded launch **or** has concrete blocking
  prerequisites. Preparation is not execution or full-feature acceptance.

### T05 — Observe and reconcile the separately authorized RHEL pilot

- **Status:** deferred by the user on 5 November 2026 ("skip b4"), not completed
  or accepted. Full acceptance remains incomplete; the single exploratory
  attempt authorized under the scoped exception in
  [spec](spec.md#pilot-prerequisites-and-retained-findings) has been consumed and
  failed before review. Subsequent VM actual-dispatch verification is authorized
  in [spec §Approval](spec.md#approval); it demonstrated a symlink-boundary failure
  (IR-06-01). No further live review was launched after that failure.
  IR-01–IR-03 remain open for their remaining scope. Depends on T04.
  **Covers:** AC-01–AC-10. **Contracts:** C1–C5.
- **Objective:** demonstrate the selected native route and guided summary, not
  general runtime certification. The user operates/authorizes the pilot environment;
  any agent-operated calls still require the exact separate repository authorization.
- **Consumes/produces:** reviewed pilot inputs and actual authorization → correlated
  sanitized observations, user feedback and retained findings in this continuation.
- **Checks:** actual foreground parent-to-child invocation with fresh context;
  exact effective model/profile/tool identity; seeded and clean cases; supported
  attempted denials plus successful permitted reads; missing evidence and changed
  candidate; owned-child bounded failure/cancellation; index/HEAD/worktree and
  existing sentinels preserved; parent reconciliation and one helpful summary.
  Use offline evidence for deterministic cases rather than model calls everywhere.
- **Stop:** any breach, missed required defect, unexpected model/auxiliary request,
  unknown/delayed accounting, unavailable bounded cancel or exceeded C4 limit.
  Preserve uncertainty; never retry automatically or mark a skipped case pass.
- **Complete when:** all selected observations have pass/fail/blocked/not-run,
  actual scope/evidence and the user has been asked for qualitative usefulness
  feedback. Full acceptance remains incomplete when required claims fail or are
  blocked. Record the result without closing historical 004 findings by assertion.

## Explicit execution batches

These are execution units of one feature, not new roadmap phases.

The dated B1–B3 row observations below retain their original delivery evidence;
later human decisions in spec §Approval and the continuation supersede their old
review/next-action wording. They do not renew B4 authority; B4 remains deferred.

| Batch | Tasks | Outcome / completion and verification | Current state / next action |
| --- | --- | --- | --- |
| B1 — Portable procedure | T01 | Focused method and conditional pointers; source/link/branch inspection, no model. | `awaiting human review` (4 Nov 2026). Changed: new `skills-files/agentic-sdd-router/references/independent-review.md`; pointers in `skills-files/agentic-sdd-verification-review/SKILL.md` and `skills-files/agentic-sdd-router/references/review-resolution.md`. Checks: local links resolve (0 missing), frontmatter unchanged, 18 skill IDs unchanged, `make test`, `make vet`, `git diff --check` passed. Completion review: self-review, no supported defects. Next action: human review; B2 needs its own authorization including offline native probes. |
| B2 — Inactive native integration | T02 | Opt-in profiles and isolated static/native boundary fixtures; denials and positive reads, exact runtime identity, no prompts. | Reviewed by the user (4 Nov 2026). Authorized by the user; see [spec §Approval](spec.md#approval). Changed: `integrations/opencode/independent-review.jsonc`, `tests/harness/test_independent_review.py`, `tests/harness/native_independent_review.py`, `tests/harness/native-independent-review.sh`. Checks: static 9/9; native probe 5/5 on the pinned 2.2.5 binary (sha256 `8d060e1d…`) in the network-none, read-only, non-root container; `test_opencode.py` 7/7; `make test`, `make vet`, `git diff --check` passed. Not observed: `variant` (not exposed by `/api/agent`), actual child invocation, compaction disable, path/symlink enforcement. Next action: human review; B3 needs its own authorization. |
| B3 — Connected development and pilot preparation | T03/T04 | Independent offline scenarios, packaging/gates, scratch rollback and one bounded RHEL pilot brief; distinguish missing live proof. | `awaiting human review` (4 Nov 2026). Authorized by the user (“start b3”); see [spec §Approval](spec.md#approval). Changed: `internal/skillsync/package_test.go` (reference presence and pointers), `tests/harness/test_independent_review_docs.py` (required-rule oracle with seeded deletion/softening), `tests/harness/independent-review-transaction.sh` (scratch-only digest/backup/rollback walkthrough), `integrations/opencode/independent-review.md` (runbook and pilot brief), `integrations/opencode/README.md` (navigation). Checks: `make test`, `make vet`, `git diff --check`, 13/13 Python checks, transaction walkthrough, native probe 5/5, link check 0 missing. Not done: executable findings/freshness consumer (none exists; procedure-only), pilot fixtures and answer keys (deferred to B4), real-home or project config change. Next action: human review; B4 remains blocked. |
| B4 — Final RHEL observation | T05 | Actual bounded native pilot and guided reconciliation; per-case results, evidence and user feedback. | **Deferred by the user (5 Nov 2026: "skip b4").** Not passed; full 006 acceptance remains incomplete. Retain the native symlink failure IR-06-01 and all evidence limitations. No further B4 VM work or pilot is scheduled. No next feature/batch is automatically authorized. |

At each actual batch boundary, record changed paths, current candidate identity,
checks and evidence scope, execution outcome, technical verdict and next permitted
action here. Never infer B2/B3/B4 authorization from a green predecessor.

## Acceptance ownership

| AC | Implementation owner | Verification owner and decisive evidence |
| --- | --- | --- |
| 01 scoped request / one reviewer | T01/T02 | T03 offline routing scenarios; T05 actual parent-child invocation/no duplicate review |
| 02 actual context / identity | T01 | T03 dirty/missing-input cases; T05 reviewed candidate and preserved state |
| 03 fresh child / effective config | T02 | T02 exact-runtime inventory; T05 observed isolation and model/variant |
| 04 inspection-only boundary | T02 | T02 permission/tool-route observations; T05 attempted operations and actual preservation |
| 05 requirements / test adequacy | T01 | T03 independent oracles; T05 seeded defect, insufficient test and clean case |
| 06 bounded findings / interruption | T01/T02 | T03 failure normalization; T05 actual controlled limit/cancel and supported findings |
| 07 reconciliation / guided report | T01 | T03 disputed/unknown finding cases; T05 parent disposition and user feedback |
| 08 freshness / re-review | T01 | T03 changed-candidate/evidence cases; T05 scoped observation without an automatic retry loop |
| 09 portable/reversible assets | T01/T02/T04 | T03 package/gates; T04 scratch preview/restore and no default changes |
| 10 RHEL pilot / truthful result | T04/T05 | T05 exact-environment observations, evidence limits and qualitative feedback |

## Combined-draft readiness reconciliation

- All ten ACs have implementation and verification owners. Producer/consumer order
  is T01 method → T02 profiles/boundary evidence → T03 connected development →
  T04 final pilot preparation → T05 authorized observation. Contracts C1–C5 own
  identity, permissions, output, numerical bounds and environment/evidence once.
- Shared skill reference edits are in B1; packaging checks follow in B3. No new
  CLI/admission/schema change or skill ID is assumed. Existing default safety
  profiles stay denied. Relevant existing check and distribution tests are reused.
- The offline scope can be built without choosing a live credential or operating
  the VM. IR-01–IR-03 are surfaced dependencies, not secretly settled decisions.
  Approving offline delivery does not waive them or make full-feature claims Ready.
- Current V2 docs support the proposed native shape, but 2.2.5 runtime, cancellation,
  bounded accounting and RHEL behavior still need the scoped observations.
  If an excluded engine or broader permission is needed, return the conflict for
  decision instead of treating it as an ordinary implementation detail.
- Specialist/tool assignments live once in plan; no task-specific exceptions.
  Use main-agent self-review for development unless separately requested otherwise;
  feature 006's purpose is not permission to use its unproven reviewer during drafting.

## Combined human decision requested

Review **spec r2 + plan r1 + tasks r1** together, including the optional separate
example, preserved safe defaults, limits and blocked live prerequisites. Proposed
first authorization is **B1/T01 only**. It permits portable instruction/reference
implementation and deterministic review, not another batch or external inference.
No approval is presumed. Acceptance of this design does not provide a price/cap,
credential, VM address, model route proof, install permission or pilot allocation.

## Continuation record

- Latest human decision, 5 November 2026: "skip b4, what next". B4/T05 is
  deferred, not completed, accepted or waived. Preserve IR-06-01 and other open
  findings. Stop B4 execution; review the existing development diff and select the
  next scope from the adoption roadmap. There is no automatically authorized next
  implementation batch or new 007 feature.
- The user subsequently wrote "resume". The clarification (resume B4 versus
  keep it deferred and move to another feature) was not answered; the user then
  requested a session handover. Retain the last unambiguous B4 deferral until the
  intended scope is resolved. Handover:
  `docs/handovers/2026-11-05_006h.md`.

- 2026-11-05: the user authorized one exploratory subscription observation and
  requested removal of the named conflicting launch gates. The scoped exception
  is recorded in [spec](spec.md#pilot-prerequisites-and-retained-findings),
  [plan C4](plan.md#c4--proposed-pilot-bounds-and-stop-contract) and
  [safety](../../integrations/opencode/safety.md#before-any-separately-authorized-online-test).
  Original B4 and feature acceptance remain unchanged. The previous fixture path
  is not present on this host; the inactive profile denies providers and names a
  different model. No new inference or cancellation test was performed.
- The user supplied SSH target `nawo-rhel1@nawo-rhel1`; host key verification
  failed before authentication. A trusted fingerprint is needed before inspection.
- On the user's retry SSH succeeded. The VM contains the existing fixture and
  pinned 2.2.5 binary, but the fixture's existing profile and launcher select
  `opencode-go/mimo-v2.6-flash`, not the authorized subscription model. No pilot
  was launched, and no credentials were inspected. See the session handover.
- The user provided a session ID; sanitized metadata inspection showed agent
  `build`, model `openai/gpt-6-luna#high` and three records. No prompt was sent;
  the prior context is not established as synthetic-only. See the handover.
- The user authorized a fresh synthetic-only parent on the VM. One exploratory
  attempt was made on 5 November 2026 with the authorized model/variant and a
  scratch copy of the existing seeded files and diff. An isolated private-server
  preflight resolved both agent models and evaluated allowed reads and relevant
  denials; its agent inventory did not expose the variant. The actual standalone
  run exited 1 in 1.13 seconds: `Agent not found: "sdd-review-coordinator"`.
  No reviewer launch or model response appeared in the event stream. Sanitized
  session metadata names `openai/gpt-6-luna-fast#high`, with client counters zero
  and only `agent-switched`/`idle` records; this does not prove provider-side zero
  usage. Git status and HEAD matched before/after; the three scratch files matched
  the original source hashes afterward. No remaining members of the launched
  process group were observed. Bounded child/upstream cancellation and index
  preservation were not demonstrated. No retry was made. Owner-only VM evidence:
  `/tmp/sdd006-exploratory-whdbg1q1/`. B4 and full 006 acceptance remain incomplete.
- Offline coordinator investigation: the failed CLI session's actual location was
  `/home/nawo-rhel1`, not the scratch fixture; the fixture-local profile was thus
  outside its discovered configuration. A separate private-server API created a
  session explicitly at the scratch fixture and found the coordinator in the scoped
  agent inventory, without sending a prompt. This is a candidate location fix, not
  proof of successful live dispatch; why the CLI chose home remains unconfirmed.
  The VM has the router and verification-review skills plus their two review
  references; all four checked files match repository SHA-256 digests. Only two
  `agentic-sdd-*` skills were found in `~/.agents/skills`, not the full set of 18.
- Authorized offline development following the user's B4 completion worklist:
  runbook now distinguishes selecting a primary agent from loading a skill,
  documents launching the TUI from a disposable workspace, and stages only the
  verification-review skill plus required references inside that workspace to
  avoid denied external reads while keeping the answer key excluded. Added a
  document-contract test for these instructions. Focused suite:
  `python3 -B -m unittest discover -s tests/harness -p 'test_independent_review*.py'`
  passed 27 tests. This verifies repository prose/tests only, not the VM TUI or
  live permission dispatch. Cancellation and complete enforcement remain unresolved;
  no new model/pilot attempt was made.
- Follow-up offline runtime check on the VM: `cd /tmp/sdd006-exploratory-whdbg1q1/fixture
  && XDG_CONFIG_HOME=/tmp/sdd006-exploratory-whdbg1q1/config opencode --standalone
  debug agents` listed both custom profiles from the fixture config without a
  prompt. This confirms discovery only. Runbook now uses the explicit directory
  argument for TUI startup and documents the listing preflight. No interactive
  selection, tool dispatch or child invocation was performed.
- 2026-11-05 offline staged-guidance check on the VM scratch project: nine
  minimal skill/reference files were staged under `.agents/skills/`, each matching
  the repository SHA-256; the answer-key directory stayed outside the workspace,
  and `.git` was absent. OpenCode 2.2.5's scoped API listed the verification-review
  skill and both custom agents with their intended modes. Exact-file read and skill
  permission evaluations allowed only the review method inputs; answer-key path,
  shell, edit and nested-subagent evaluations denied. The first attempt exposed
  missing explicit staged-file permissions and the `.opencode/` deny collision;
  these were corrected only in the scratch configuration. This remains API
  permission evaluation, not actual tool dispatch/read or TUI behavior. No prompt,
  child launch or model request was made.
- Updated the runbook to use the verified workspace-local `.agents/skills/`
  compatibility layout and exact file-level allow rules. Final focused suite:
  `python3 -B -m unittest discover -s tests/harness -p 'test_independent_review*.py'`
  passed 27 tests; `git diff --check` passed. Cancellation behavior and live
  prohibited-operation enforcement remain unproven; B4 is incomplete.
- Actual-dispatch follow-up, authorized by the user's request to finish B4 VM
  work: added `native_independent_review_dispatch.py` and the rootless network-
  namespace wrapper `native-independent-review-dispatch.sh`. These use scripted
  loopback responses and fresh HOME/XDG state only, not external inference.
  Final exact-runtime run: **3 pass, 1 fail** in 10.106 seconds on the pinned VM
  binary. The CLI test reproduced stale `PWD` failure and verified corrected
  `PWD`/`cwd`; the child loaded the packaged review skill, read permitted files
  and reference, excluded a parent-only marker, and refused eight prohibited
  calls. Both active owned sessions reached durable `interrupted` outcomes within
  five seconds in the local synthetic cancellation test.
- **IR-06-01 — valid, high-impact access failure:** the read tool followed an
  explicitly allowed `src/escape.txt` symlink to an excluded synthetic sentinel
  outside the project. Direct external reads were denied. The native regression
  stays failing; this blocks a generic canonical-path isolation claim. C2 permits
  an explicitly reviewed sanitized read-only export rejecting symlinks and special
  files; no such reviewed live export has been certified by this run. The real
  pilot answer key was not read. No model-backed review was launched after the
  failure. Cancellation evidence is limited to the synthetic local transport.
- Final VM evidence: `/tmp/sdd006-exploratory-whdbg1q1/dispatch-evidence-final2/`,
  `dispatch-final2.stdout`, `dispatch-final2.stderr`, and
  `dispatch-final2-preservation.json` in their parent directory. Original fixture
  HEAD, index digest, Git status and five seeded/clean source-file digests matched
  before/after. The generic boundary's failing result is retained; B4/full 006
  acceptance remain incomplete. Next permitted action: develop/review the C2
  sanitized-export mitigation or keep this mode unsupported, then reassess remaining
  observations rather than repeating the unsafe route.
- Completion self-review for this VM verification increment: **Blocked** for full
  B4 acceptance, `awaiting human review` of IR-06-01 and the retained export
  mitigation. Local focused suite passed **29/29**; the native fixture remained
  **3 pass / 1 fail** on the final code state. Shell syntax and `git diff --check`
  passed. No Go files changed, no shared-service or real-home configuration was
  changed, and no commit/push/publication was performed.

- 2026-11-03: user requested plan/tasks drafting, then explicitly requested commit
  and push. Drafted from spec r2; combined approval remains **pending** in
  [spec §Approval](spec.md#approval). No implementation batch started.
- Discovery: inspected canonical review/router guidance, inactive templates,
  candidate validation, explicit embedding, package tests and offline probe pattern.
  Current official V2/Context7 docs informed design; no runtime or model probe run.
- Checks for this documentation candidate and publication outcome are recorded in
  the session handover. No model compliance, native or RHEL acceptance is claimed.
- 2026-11-04: user approved spec r2, plan r1 and tasks r1 combined and authorized
  B1/T01 only, via the router's approval question. Recorded in
  [spec §Approval](spec.md#approval). B1 implemented as above and published with the 006 B1 commits (see git log).
  Session handover: `docs/handovers/2026-11-04_006b.md`.
- 2026-11-04: user reviewed B1, asked for it to be committed and pushed (`22da737`,
  `a1f9cd4`; `origin/main` updated), then authorized B2 (“authorize the b2”), recorded
  in [spec §Approval](spec.md#approval) with the T02 fixture boundary. B2 implemented
  as above. Offline observations (no prompt, no model, no provider):
  - Static and profile/permission expectations pass; the example stays inactive and
    the default and lockdown templates are unchanged.
  - The runtime exposes `model` as `{id, providerID}` for both profiles; `variant`
    is not exposed and is therefore **not observed**.
  - A higher-authority `subagent:*` denial still blocks the coordinator's launch of the
    reviewer, as intended. The reviewer keeps reads and its skill and denies delegation.
  - Path and symlink escape through the read tool, compaction disabling, actual child
    context and cancellation are **not observed**; permission evaluation does not prove
    tool dispatch. These stay blockers for the affected claims (IR-01/IR-02).
- 2026-11-04: user reviewed B2 and said “start b3”, recorded in
  [spec §Approval](spec.md#approval). B3 done as in the table. Findings from B3 work:
  - The 005 `agentic-sdd opencode config` helper rejects this example (`candidate must
    retain the inactive selected defaults`); the runbook therefore uses a manual
    procedure, walked through on temporary files.
  - The packaged reference is covered by the existing byte-for-byte install and link
    tests and by a new explicit presence test.
  - Snapshot identity already covers staged, unstaged, untracked, deleted and mode
    changes (`TestSnapshotDetectsWorkingTreeChanges`); reused, not duplicated.
  - Follow-up, 5 Nov 2026: `tests/harness/test_independent_review_fixtures.py` with a
    test-only consumer `tests/harness/independent_review_contract.py` covers scripted
    reconciliation (15 outcomes, expectations written independently), rule coverage (each
    rule changes at least one outcome), explicit-only and no-duplicate scheduling, and
    freshness on the real `opencode evidence snapshot` binary in scratch Git repos. The
    consumer is not shipped and is not evidence of model behavior.
  - Still not established: model-produced reports, a runtime duplicate-scheduling
    check, and bounded cancellation. These need the B4 observation.
- 2026-11-05: partial RHEL pilot observations were made after B3 acceptance; the
  [runbook](../../integrations/opencode/independent-review.md) records the
  outcomes and evidence limitations. Parent event streams do not establish complete
  child usage or prove answer-key isolation. Further inference remains blocked.
- Offline recheck: `sh tests/harness/native-independent-review.sh` passed 5/5
  against the pinned 2.2.5 binary in its isolated container. This existing fixture
  does not reproduce the VM external-reference denial and does not establish live
  child behavior, billing, provider request counts or cancellation.
- Follow-up: added an external-reference permission-evaluation case using a
  synthetic file and narrow allow in the network-disabled container. The pinned
  runtime returned allow for the selected reference boundary and read, and deny
  for a separate external boundary, for both profiles (6/6 fixture cases pass).
  This does not reproduce the VM denial or establish actual read-tool dispatch.
- **Next permitted action:** reconcile provider-side usage and dedicated cap without
  exposing credentials, then investigate the reference denial in a credential-free,
  network-disabled exact-runtime fixture. Do not start another live pilot by inference.
- 2026-11-05 supervised-pilot request: prepared a [narrow exception proposal](supervised-pilot-proposal.md)
  for `openai/gpt-6-luna-fast#high`; it is not adopted or executable under the
  applicable higher-priority session controls. Re-ran the offline native fixture:
  6/6 pass, permission evaluation only. No new inference or cancellation test.
- 2026-11-05: investigated the proposed numeric allocation, pinned runtime source,
  historical partial usage and a dedicated API alternative; see [plan research](plan.md#pilot-control-investigation--5-november-2026).
  Source indicates retries and auxiliary title calls can occur outside a simple
  step count. Complete prior provider reconciliation and pre-send bounds remain
  unknown. The API alternative is a proposal only, not an authorized model switch
  or demonstrated workaround. Native fixture passed 6/6; S0 unit suite passed 12
  with nine container-only skips. No model calls, credentials, VM or shared-service
  operations were performed in this research session. B4 remains incomplete/blocked.
