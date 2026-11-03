# Independent review tasks and continuation

Revision: r1 · 3 November 2026. Inputs: [spec r2](spec.md), [plan r1](plan.md).
Single approval location: [spec §Approval](spec.md#approval).
All tasks are pending; drafting/publication does not authorize implementation.

## Execution boundary

Execute one selected batch only after combined approval and batch authorization,
verify it, return one completion review, record `awaiting human review` and stop.
Plan contracts C1–C5 own shared details; specialist/tool assignments live there.
No task triggers real inference, VM access, real-home writes or release implicitly.
Existing deterministic commit hooks/CI remain enabled and distinct from a pilot.

## Ordered tasks

### T01 — Integrate the portable independent-review method

- **Status:** pending. **Covers:** AC-01/02/05/06/07/08/09. **Contracts:** C1/C3.
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

- **Status:** pending. Depends on T01. **Covers:** AC-01/03/04/06/09.
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

- **Status:** pending. Depends on T01/T02. **Covers:** AC-01/02/05/06/07/08/09.
  **Contracts:** C1/C3/C5.
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

- **Status:** pending. Depends on T02/T03. **Covers:** AC-03/04/06/09/10.
  **Contracts:** C2/C4/C5.
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

- **Status:** pending; live execution blocked on IR-01–IR-03 and explicit launch
  authorization. Depends on T04. **Covers:** AC-01–AC-10. **Contracts:** C1–C5.
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

| Batch | Tasks | Outcome / completion and verification | Current state / next action |
| --- | --- | --- | --- |
| B1 — Portable procedure | T01 | Focused method and conditional pointers; source/link/branch inspection, no model. | Not started; first independently ready batch after combined approval and explicit B1 authorization. Stop after one completion review. |
| B2 — Inactive native integration | T02 | Opt-in profiles and isolated static/native boundary fixtures; denials and positive reads, exact runtime identity, no prompts. | Not started; requires B1 review and explicit B2 authority including offline probes. Stop for human review. |
| B3 — Connected development and pilot preparation | T03/T04 | Independent offline scenarios, packaging/gates, scratch rollback and one bounded RHEL pilot brief; distinguish missing live proof. | Not started; requires B2 disposition and explicit B3 authorization. End `awaiting human review`; do not launch pilot. |
| B4 — Final RHEL observation | T05 | Actual bounded native pilot and guided reconciliation; per-case results, evidence and user feedback. | Not started / live gate blocked. Requires B3 review, resolved prerequisites and separate exact VM/effect/model launch authorization. Stop for human acceptance. |

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

- 2026-11-03: user requested plan/tasks drafting, then explicitly requested commit
  and push. Drafted from spec r2; combined approval remains **pending** in
  [spec §Approval](spec.md#approval). No implementation batch started.
- Discovery: inspected canonical review/router guidance, inactive templates,
  candidate validation, explicit embedding, package tests and offline probe pattern.
  Current official V2/Context7 docs informed design; no runtime or model probe run.
- Checks for this documentation candidate and publication outcome are recorded in
  the session handover. No model compliance, native or RHEL acceptance is claimed.
- **Next permitted action:** present combined documents for one human decision.
  On approval plus explicit B1 authority, implement only T01; otherwise stay at
  drafting/review. B4 remains separately blocked; no code or live effect follows
  from publication.
