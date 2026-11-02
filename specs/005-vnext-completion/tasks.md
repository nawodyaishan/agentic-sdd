# vNext completion tasks and continuation

Revision: r1 — draft, 2026-11-02. Inputs: [spec r1](spec.md), [plan r1](plan.md).
Single approval location: [spec §Approval](spec.md#approval). Combined r1 approved;
B1 and B2 explicitly authorized sequentially. Specialist/tool assignments live in plan.

## Execution and verification boundary

- Implement only one explicitly authorized batch after combined approval, run
  its permitted focused checks and one completion review, then stop for review.
- Batch review is code/document/evidence review, **not a human workflow-usage
  test**. The user conducts one integrated usage cycle only after all selected
  implementation is delivered; no per-batch/per-feature usage sessions.
- No full CI/PR experiment, local containerized real-AI test, model smoke probe,
  benchmark, implicit child agent, primary credential read/copy, real-home apply,
  account/service change or publication follows from this plan.
- Go-changing batches run repository-required local `make test`, `make vet` and
  `git diff --check`, plus relevant focused tests; no recursive check runner in
  unit tests. Reuse unchanged valid evidence and do not reconstruct 004's costly
  tests. If a required check conflicts with the stop direction, seek a decision
  rather than silently running/removing it. Normal existing commit hooks remain.
- Temporary-home/stdio test programs are synthetic deterministic development
  fixtures, never native models. A fake native response tests the consumer, not
  real OpenCode behavior. Do not promote structural checks to behavioral evidence.
- Later implementation changes may resolve actual code gaps, but 004's finding
  IDs/evidence and human acceptance are retained until actual disposition evidence
  exists. No automatic native/benefit certification or support-pin promotion.

## Ordered tasks

### T01 — Freeze delta, source provenance and core coverage

- **Status:** delivered in B1 (2026-11-02); combined approval/authority recorded in spec.
- **Covers:** AC-01, AC-13, AC-15. **Contracts:** C1–C7, especially C7.
- **Objective:** identify exact baseline files and remaining implementation before
  editing; retain every selected core area/helper, all exclusions and 004 findings.
- **Targets:** this continuation, plan's coverage/source records; scoped source
  identity evidence outside the checkout. No extra completion/approval ledger.
- **Produces:** source/contract/fixture identities, concrete delta ownership and
  provenance/notice list consumed by T02–T12. No behavioral baseline reconstructed.
- **Checks:** actual eleven-ID discovery, reference closure, seven-source capability
  map, pinned source identities and no unwanted active-config path. Inspect current
  dirty work and preserve sentinels/private handovers. Every core area has a
  delivery owner or explicit exclusion; missing observations are not called defects.
- **Complete when:** builders can identify required delta versus reused code,
  source prerequisites and outstanding risks without consequential guessing.

### T02 — Implement all seven portable helper capabilities

- **Status:** delivered in B1. Depends on T01. **Covers:** AC-02/03/05/06/07/08/09.
- **Contracts:** C1/C2. **Targets:** the seven `skills-files/agentic-sdd-*`
  directories selected in C1, their owned references/formats and MIT notice.
- **Objective:** deliver explicit grilling alias/reusable frontier procedure,
  recipient questionnaire and attributed return, instruction maintenance,
  canonical handoff, response-only explanation and scoped teaching.
- **Consumes/produces:** T01 provenance and existing shared authority → portable
  helper bodies/reference graph consumed by T03/T04 and native commands.
- **Checks:** inspect expected outputs/effects for clear request/no interview,
  dependent versus ready questions, discovered facts, repeated answers, conflicting
  recipient inputs, review-only helper, no-write handoff, lightweight teaching and
  explicitly selected persistent teaching. Validate frontmatter/links/notices and
  deliberate mode/effect branches. No phrase check or scripted answer is reported
  as an agent compliance pass; human behavior observation waits until final use.
- **Complete when:** all seven capabilities and every required supporting file
  exist with precise triggers, bounded effects and checkable completion rules;
  no absolute external-source dependency, sending/browser launch or auto-delegation.

### T03 — Integrate helpers into the existing lifecycle

- **Status:** delivered in B1. Depends on T02. **Covers:** AC-01/02/03/04/05/06/09.
- **Contracts:** C1/C2/C7. **Targets:** router/shared policy and new human-input
  reference; spec/plan/tasks/implement/verification-review/drift-retro/bootstrap
  pointers only where their branches need them; supporting task/review references.
- **Objective:** one authority definition, concrete checkpoint summary, late-answer
  impact reconciliation and optional helper entry without mandatory interviews,
  duplicate review or a second ledger. Preserve direct invocation and direct fixes.
- **Checks:** trace action-first paths, spec-only stopping, actual latest batch
  authorization, material late scope change versus unaffected work, acceptance
  with open findings and no-write review/teaching requests. Confirm no repeated
  helper body, unconditional upstream install or implication of new authority.
- **Complete when:** producers/consumers share C2 and clear requests remain direct;
  disclosure pointers reach the right helper without loading all seven every time.

### T04 — Package the expanded suite without installer regression

- **Status:** delivered in B1. Depends on T02/T03. **Covers:** AC-09/13. **Contract:** C1.
- **Targets:** `internal/skillsync/package_test.go`, related discovery/reference
  checks, package README/inventories and embedded-source coverage. Change production
  installer logic only for a demonstrated gap; `discoverSkills` is already generic.
- **Objective:** compare the exact intended eighteen-ID set for embedded/on-disk
  sources, retain references/notices and existing four destination semantics.
- **Checks:** first demonstrate the new expected set fails against the old package;
  then temporary-home install/preview/no-op/restore, original/new-file rollback,
  unrelated skills (including unnamespaced Matt IDs), symlinks/special files and
  backup preservation. No real skill directory or runtime config is written.
- **Complete when:** source modes agree, no helper/reference is omitted, and safe
  installer invariants remain true. All helper distribution/semantics documentation
  distinguishes shipped guidance from unobserved model behavior.

### T05 — Implement sanitized runtime preflight and inactive candidate

- **Status:** pending. Depends on B1 delivery/review and B2 authorization.
- **Covers:** AC-01/10/13/15. **Contracts:** C3/C4.
- **Targets:** `cmd/agentic-sdd` command dispatch/help, `internal/opencodeadapter`,
  embedded inactive resources, existing JSONC/model/lockdown examples and
  supported-version/safety/usage documentation. Update README/Makefile surfaces.
- **Objective:** explicit file-input inventory/default offline preparation, exact
  2.2.5 source contract, safe candidate generation and read-only native inventory
  interface with bounded allowlist. Document every model/tool effect/unknown.
- **Checks:** synthetic paths/version/catalog projections, duplicate/unrecognized
  input, truncated inventory, secret/header/env canaries, location mismatch, inert
  providers/integrations and no active discovery output. Fake subprocess/server
  dispatch tests prove prohibited native/provider operations are not called.
- **Complete when:** preflight/candidate operations produce useful sanitized output
  or a specific blocked result, never active configuration or guessed support.

### T06 — Implement separate config preview/apply/rollback transaction

- **Status:** pending. Depends on T05. **Covers:** AC-10/13. **Contract:** C3.
- **Targets:** adapter CLI/config/files/manifest implementation and tests;
  `integrations/opencode/adoption.md`. Keep skillsync backup manifests unchanged.
- **Objective:** user-curated candidate bytes, private external runtime-config
  backups, no-op and digest-checked rollback, without generic JSONC merge/reformat.
- **Checks:** only temporary reviewed roots: unchanged no-op/no backup, unrelated
  fields/comments retained, missing original, all backups before replacements,
  failure mid-transaction, changed parents/source/destination, symlink/FIFO,
  collision and intervening edit at apply/rollback. Refuse raw sensitive input
  exports and archive/config paths inside active discovery. Errors omit values.
- **Complete when:** planned files alone are changed or safely restored, unknown
  partial effects are retained and neither explicit CLI invocation nor an input
  approval reference is portrayed as proof of user consent. No real-home exercise.

### T07 — Implement the narrow owned native ACP transport

- **Status:** delivered in B3 (2026-11-02). Depends on B2 delivery/review and B3 authorization.
- **Covers:** AC-10/11/12/13. **Contracts:** C4/C5.
- **Targets:** `internal/opencodeadapter` ACP/owned-process code and fake-peer tests;
  `cmd/agentic-sdd` session command. No SDK dependency or replacement agent loop.
- **Objective:** bounded stdio JSON-RPC protocol 1, exact workspace/session and
  supported model/mode selection, no shared server/login/children/arbitrary argv.
- **Checks:** fake peer receives expected handshake/config/session messages once;
  refusal of wrong executable/version/schema/session, foreign/child metadata,
  duplicate/late RPC IDs, oversized/partial data, EOF/deadline and deadlock.
  Synthetic owned-process cancellation preserves unrelated processes. Unproven
  host/process behavior remains unavailable, not certified by cross-build.
- **Complete when:** the controlled transport has positive and negative outcomes,
  retains uncertain cleanup and never launches a real model in tests. Native
  prompt admission is denied while any C4 prerequisite is unverified.

### T08 — Implement pending-human normalization and safe re-entry

- **Status:** delivered in B3 (2026-11-02). Depends on T07/T03. **Covers:** AC-04/06/10/11/12/13.
- **Contracts:** C2/C4/C5. **Targets:** adapter state/outcome/pending-summary code,
  session CLI handling and `integrations/opencode/recovery.md`.
- **Objective:** stop on consequential form/permission before replying, bound
  cancellation/cleanup and return actionable sanitized questions without consent
  fabrication or continuing from the CLI's cancelled-question feedback.
- **Checks:** fake form with choices/custom/unknown rationale, permission request,
  cancellation success/failure/race, unexpected provider error with process exit 0,
  missing terminal event, child/retry and unknown required event. Assert no
  guessed form reply, second prompt, fallback, login or auto-resume. A replayed
  summary or stale/foreign response cannot authorize re-entry; review mode exports
  only to an authorized external file or stdout.
- **Complete when:** distinct execution/stop/review meanings survive persistence
  and reload; primary pending cause remains blocked even after native cancellation.
  Strict online admission still blocks unknown token/egress/accounting controls.

### T09 — Implement the production v1 evidence consumer

- **Status:** delivered in B4 (2026-11-02). Depends on B3 delivery/review and B4 authorization.
- **Covers:** AC-12/13/15. **Contract:** C6.
- **Targets:** `internal/harnessevidence`, existing receipt schema/README, shared
  check-result types only as needed. Preserve the Python reference as development
  material; no new production Python/JSON Schema engine or v1 wire change.
- **Objective:** strict typed/semantic validation, compatible selected-scope
  identity and safe external export with independent input/provenance boundaries.
- **Checks:** existing portable positive/negative fixtures plus independent golden
  expectations: result/contract/log mismatch, unknown fields/duplicates/nulls,
  missing authority/usage, staged/unstaged/untracked/deleted/mode changes, symlink/
  special file, move, stale code/config/runtime, redaction and interrupted writes.
  Verify reference parity without deriving expected values from the new code.
- **Complete when:** valid v1 receipts round-trip and misleading ones fail closed;
  no approval/task state is written, old identity meaning is retained and any
  necessary incompatible change is returned for a separate migration decision.

### T10 — Connect owned native observations and deterministic results

- **Status:** delivered in B4 (2026-11-02). Depends on T09/T08. **Covers:** AC-10/11/12/13/15.
- **Contracts:** C3/C5/C6. **Targets:** adapter receipt projection/CLI, version and
  evidence docs; focused synthetic result-to-native-observation fixtures.
- **Objective:** record source identity before/after checks, normalize only permitted
  native values and emit the existing receipt plus pending sidecar when applicable.
- **Checks:** successful and failed fake turns, missing cost, native-reported versus
  provider-verified billing, altered log/result/observations, interrupted owned
  activity, foreign session, review-only no-write and unknown provenance. No full
  transcript/reasoning/auth DB ingestion. A final zero wrapper exit with provider
  failure cannot yield a successful task or Ready.
- **Complete when:** native/result/freshness producers agree and unknowns remain
  explicit; observations are not misrepresented as authenticated provenance.

### T11 — Verify the complete candidate through permitted development checks

- **Status:** delivered in B5 (2026-11-02). Depends on B4 delivery/review and B5 authorization.
- **Covers:** AC-01–AC-13/15. **Contracts:** C1–C7.
- **Targets:** focused Go/CLI/packaging/reference fixtures and existing documentation;
  this task's acceptance reconciliation. No CI/workflow trust changes.
- **Objective:** exercise component interfaces together with synthetic inputs,
  inspect all selected core/helper coverage and catch regressions before handover.
- **Checks:** source/reference/discovery closure; candidate/preflight transaction;
  fake native pending/error/interrupt/resume-to-new-preparation paths; observed
  deterministic result receipt export/reload/freshness; original installer safety.
  Run applicable local gates once against final source identity, not every edit.
  Never use a model to test skill triggering/compliance or imitate a human usage
  cycle and label it observed acceptance.
- **Complete when:** every required implementation has a concrete artifact and
  development result or a blocking defect; unsupported behavior and missing real
  observations are explicit. No feature Ready merely from all-green components.

### T12 — Reconcile implementation and prepare the one final human cycle

- **Status:** delivered in B5 (2026-11-02). Depends on T11. **Covers:** AC-01/13/14/15. **Contract:** C7.
- **Targets:** this continuation, supported-version/adoption/recovery docs and
  `docs/vnext-human-test-plan.md`; aggregate review and required session handover.
- **Objective:** one integrated candidate and human-readable first-use/rollback
  path, not another implementation-only report or per-feature test checkpoint.
- **Checks:** every AC's owner/result, all helper/core branches, exact identities,
  carried 004 IDs, no prematurely closed evidence-only finding, no false billing/
  support/savings claim. Manual scenario instructions are complete but **not run**.
- **Complete when:** all selected implementation is delivered safely; status says
  `implementation delivered; human usage validation not run` where justified.
  Otherwise retain `incomplete/Blocked` and the concrete missing implementation.
  Present the single final human cycle for a separate actual launch decision.

## Execution batches

| Batch | Tasks | Outcome and focused verification | State / next action |
| --- | --- | --- | --- |
| B1 — Portable helper suite | T01–T04 | Seven helpers, canonical lifecycle pointers, provenance/notices and safe eighteen-ID embedded/on-disk package. Focused source/link/semantic review and temporary-home packaging tests; local gates for Go test changes. | Delivered and self-reviewed Ready for this source/distribution scope. |
| B2 — Inactive runtime delivery | T05–T06 | Sanitized preflight/candidate and separate config transaction. Fake inventory/CLI and temporary-filesystem negative cases; Go local gates. | Delivered in source/tests; A1 artifact produced by `agentic-sdd opencode config preview` not yet run. Self-reviewed Ready for this source/distribution scope. |
| B3 — Native pending/recovery integration | T07–T08 | Thin ACP client, strict admission, owned interruption and pending-human output. Fake-peer/owned-process tests only; Go local gates. | Delivered at source level with fake-peer tests; awaiting human review. No native/model experiment. |
| B4 — Production evidence bridge | T09–T10 | Compatible v1 identity/validation/export connected to owned observations and existing check results. Independent golden/negative/composition cases; Go local gates. | Delivered, published to `origin/main` (`2e2d461`…`af0036e`); B5 authorized directly by the user on 2026-11-02. Offline only; unknown billing/provenance remains unknown. |
| B5 — Complete-candidate handover | T11–T12 | Aggregate development verification, final core/helper coverage and single final human-cycle instructions. Reuse valid checks, inspect final diff and retained findings. | Delivered and human-reviewed (accepted) 2026-11-02; a pinned container environment for the accepted human cycle was subsequently built and recorded in spec §Approval. Awaiting the user-run cycle. |

The final human usage cycle is **not B6 automatically starting**, not an LLM test
job and not authorized by implementation completion. The user owns its later
scope, environment, provider/effect limits and actual execution.

## Acceptance and verification ownership

Development results below are source/deterministic evidence only. "Remaining
observation" is what the one final human cycle (or a later separately authorized
native run) must still establish; none of it is implied by delivery.

| AC | Implementation owner | Development result (B5, 2026-11-02) | Remaining observation |
| --- | --- | --- | --- |
| 01 core coverage | T01, T05–T12 | 18-ID helper set, C1–C3 adapter path, C4/C5 session/pending, C6 evidence bridge delivered; coverage map in plan §Core coverage reconciliation and this table. | Real workflow behavior in the final human cycle. |
| 02 grilling | T02/T03 | `grill-me`/`grilling` helpers and reference checks delivered; bounded independent questions, no mandatory interview. | Human clarity/effort; no question-count or compliance metric. |
| 03 questionnaire | T02/T03 | `questionnaire` helper with recipient/purpose, attribution and uncertainty guidance; artifact only when justified. | Actual recipient exchange; none sent by tests. |
| 04 decisions/late answers | T03/T08 | Existing approval/task locations, pending normalization and recovery procedure; no new ledger/form. | Human authority and late-answer behavior in real use. |
| 05 instruction maintenance | T02/T03 | `writing-for-agents` helper; provenance, conditional pointers and single-owner rules in the packaged sources. | No automatic rewrite claim; human maintainability review. |
| 06 handoff/resume | T02/T03/T08 | `handoff` helper plus pending/recovery rules and receipt freshness checks; a handoff creates no batch authority. | Native recovery/compaction/move remains unproven. B6-F1. |
| 07 explanation | T02 | `wait-what` helper with explicit no-side-effect behavior. | Human clarity observation; no state change expected. |
| 08 teaching | T02 | `teach` helper with opt-in persistent workspace guidance and attribution rules. | Actual learning evidence; not inferred from material covered. |
| 09 distribution | T04 | 18 skill IDs / 28 files, sibling references, embedded/on-disk tests, preview/no-op/backup/restore and symlink/special-file refusal. | Real-home install not run; support not certified. |
| 10 inactive adapter/adoption | T05–T08/T10 | Candidate/preflight, config preview/apply/rollback transaction and ACP client delivered; admission blocked by default; CLI smoke showed preview writes nothing. | Actual setup/adoption/rollback and effective live skill/tool behavior. B3-F1/F4. |
| 11 pending/native status | T07/T08/T10 | Fake-peer form/permission/cancel/provider-error/unknown-event cases; checks-derived execution and null review in projection. | Real native pending, cancellation and status mapping. B3-F1/F2, B6-F1. |
| 12 evidence/recovery | T07–T10 | v1 consumer/projection/export/reload/freshness and scoped-identity golden parity tests; unknowns explicit; caller-supplied provenance not certified. | Real native session/model/billing correlation. B6-F2. |
| 13 development boundary | Every task; T11 integration | Required gates and focused fixtures run on the final source identity; no new CI/model/container campaign. | Later manual evidence must stay separate from development checks. |
| 14 final-only human cycle | T12 | [Final human usage plan](../../docs/vnext-human-test-plan.md) prepared for the whole candidate; not executed. | User chooses workspace, launch scope and live/model permissions. |
| 15 truthful status/findings | T01/T12 | This disposition keeps 004 finding IDs open and separates missing implementation from missing observation; no support/benefit/savings claim. | Human acceptance and later observations; no wording closes findings. |

## Combined-draft readiness reconciliation

- All fifteen ACs have implementation and verification ownership. C1–C7 own shared
  boundaries; consumers reference them rather than copying types/units/authority.
- Producer/consumer order is T01 → portable package → inactive adapter → ACP
  outcomes → v1 evidence bridge → final integration/human handover. No human usage
  gate is placed between those development batches.
- Exact 2.2.5 ACP source was identified; the CLI cancelled-question assumption
  was rejected. Fake peers are development evidence, not runtime certification.
- Packaging is eighteen named IDs with source closure, not blind installation of
  external Matt skills. Configuration backups do not reuse skill manifests.
- No material product-scope question remains: R0–R2 plus seven helpers, OpenCode
  only. Online admission evidence, final real setup/task and any live permissions
  remain unmet prerequisites for **later execution**, not consent created by drafts.
- The drafts are ready to present together, not approved. No implementation,
  commit/push, provider call or installation follows from this reconciliation.

## Continuation record

- Current execution (2026-11-02): user approved the combined r1 set and **B1 B2**,
  then completed B1/B2 human review and authorized **B3**, then completed the
  **B3 human review (accepted)** and authorized **B4**, then directed B4
  publication and authorized **B5** (all recorded in spec §Approval). B1–B4 are
  delivered; **B5 (T11–T12) is delivered and its human review was accepted on
  2026-11-02**. All selected implementation is now delivered. The B3 and B4
  commit chains were pushed to `origin/main` per separate publication decisions;
  the B5 commits remain local pending that decision. The one final human usage
  cycle is prepared but not executed — its launch is a separate later decision.
- B1 evidence: new eighteen-ID distribution test first failed against the original
  eleven-skill tree. After implementation, `TMPDIR=/tmp/opencode GOTOOLCHAIN=local
  GOPROXY=off GOSUMDB=off make test` and `make vet` passed; `git diff --check` and
  frontmatter/packaged local links/whitespace passed for 18 skills, 28 files.
  Source identity `7e66f50530c38c5f3b3ab2fb6cb43eba2c0d5ca09e3d66b426b8a49ee5982d90`
  (sorted repository path + NUL + content SHA-256 + LF). Embedded/on-disk tests
  exercise actual copy/preview/no-op/backup/restore and unnamespaced helper
  preservation in temporary homes. No installed skill was changed.
- B2 evidence: `internal/opencodeadapter` (candidate emission, preflight inspection,
  config preview/apply/rollback) and `cmd/agentic-sdd` (`opencode` command group)
  implemented and tested with fake peers/temporary roots. Required checks `make test`,
  `make vet`, `sdd-check required` (6/6: mod-verify/tidy-check/vet/test/build-darwin/diff-check)
  and `git diff --check` all pass. No model call, native native service, PR experiment
  or real-home adoption was performed. Evidence root `/tmp/opencode/sdd-b2-verify-20261102/`
  result SHA-256 `5d5d918dff3a553f0ca48ac45d9d39390f916be4d45d147e92c86413bda83ece`.
- **Batch state: B5 delivered and human review accepted (2026-11-02); all
  selected implementation is delivered.** B4 was published on user direction and
  B5 was authorized directly ("submit b4 commits, start B5"). B3 evidence:
  `internal/opencodeadapter` ACP transport/owned-process
  lifecycle, strict admission (`admission.go`), pending-human normalization
  (`pending.go`) and `cmd/agentic-sdd` (`opencode session prepare|run|pending`)
  implemented and exercised only by synthetic fake peers (this test binary
  re-executed as a scripted ACP server on fd 3). Outcomes covered: happy end_turn
  with usage, elicitation/permission pending-human (with bounded option capture and
  cancellation confirmation), provider error with process exit 0, missing terminal
  event, child-session metadata (invalidates usage), foreign session, duplicate/late
  ids, oversized frames, wrong version, unknown request, deadline, and owned-process
  termination that preserves unrelated processes. `make test`, `make vet`,
  `go test -race ./...`, `git diff --check` and `sdd-check required` (6/6:
  mod-verify/tidy-check/vet/test/build-darwin/diff-check) all pass. Source
  identity for the B3 Go files + tests (sorted path + NUL + content SHA-256 + LF)
  `8df49be3a33ca0bc36f5be3487648d967571fa7b7c5528932f06ba8ca994f458`. Verification
  result root `/tmp/opencode/sdd-b3-final.AU3dlx/` (execution completed, 6/6).
  No real
  `opencode acp` process, model call, provider probe, credential read or real-home
  adoption occurred; `session run` stays `blocked` because strict admission controls
  remain unverified. B4–B5 do not start; stopping here per batch rules.
- B4 evidence (2026-11-02): `internal/harnessevidence` (strict typed v1 parse,
  semantic validation against independently supplied identity/contract/result/
  log digests/authorization/observations, scoped `sha256-v1` identity, safe
  external export) and `internal/opencodeadapter/receipt.go` (checks-derived
  projection plus private pending sidecar) implemented, with
  `agentic-sdd opencode evidence snapshot|validate|export|receipt`. Tests cover
  valid/invalid round trips, schema/duplicates/nulls, stale identity and
  authority, evidence tamper/mode/missing/symlink/traversal/root, workspace move,
  Ready/coverage, cancelled/blocked promotion, redaction/overwrite/checkout
  refusal, identity drift, contract mismatch, missing cost/session/model,
  provider-error and interrupted turns over completed results, and fake-peer
  elicitation. Golden identity digests were derived independently through the
  Python reference (`tests/harness/go_snapshot_fixture.py`), and all 18 reference
  receipt tests still pass. `make test`, `make vet`, `go test -race ./...`,
  `git diff --check` and `sdd-check required` (6/6: mod-verify/tidy-check/vet/
  test/build-darwin/diff-check) pass. Source identity for the B4 Go/test/reference
  files (sorted repository path + NUL + content SHA-256 + LF)
  `5da0c69dbfa3640b6ab1265ff208dc65a7bd8337f713dea3a1925f33bc909706`. Verification
  result root `/tmp/opencode/sdd-b4-final.hiHwvb/` (execution completed, 6/6;
  result.json SHA-256 `13f2d4b734c9148f241e119c439ce71d8518a0f4579672d7d90a5d2640b4c830`)
  and post-commit root `/tmp/opencode/sdd-b4-post-commit.llckep/` (execution
  completed, 6/6).
  No model call, native `opencode acp` process, provider probe, credential read,
  real-home adoption, approval/review write or native-service change occurred.
  One latent reference defect was fixed: `git check-ignore` rejects
  `--literal-pathspecs`, so the reference and Go consumer now omit that flag for
  check-ignore only and fail closed on pathspec-magic names.
- B5 evidence (2026-11-02): T11/T12 delivered. Final aggregate verification on
  the delivered tree: `make test`, `make vet`, `go test -race -count=1 ./...`,
  `git diff --check`, `sdd-check required` (6/6: mod-verify/tidy-check/vet/test/
  build-darwin/diff-check, execution completed) and the offline Python reference
  (`test_receipts` 18/18; `validate_contracts.py` both schemas + 12 cases) pass.
  CLI smoke exercised the real candidate → preflight → config preview path with
  no native/provider access; preview wrote no workspace config. Package closure
  is 18 skill IDs / 28 files with no symlinks; installer preview/no-op/backup/
  restore and symlink/special-file refusal remain covered by `internal/skillsync`
  tests. Implementation source identity (134 tracked implementation files:
  `skills-files/`, `cmd/`, `internal/`, `harness/`, `tests/`, `.github/`,
  `go.mod`, `go.sum`, `Makefile`, `.goreleaser.yml`, `adapterfiles.go`,
  `skillsfiles.go`; sorted path + NUL + content SHA-256 + LF, Markdown docs under
  `docs/`/`specs/` excluded)
  `650f60d1752262244ac673b230d2c4305afe760440df1340ff6246a18620135d`.
  Verification root `/tmp/opencode/sdd-b5-final.e3Xw5I/`, execution completed
  6/6, `result.json` SHA-256
  `0579d60e28fa1fa189032f2a3049afb003567318d598315f23e8914b0b5ec496`;
  post-commit root `/tmp/opencode/sdd-b5-post-commit.qIdA4c/` also execution
  completed 6/6 (`result.json` SHA-256
  `76b4e2d517cbc421a1ab0520cddf7352cb58e47526fca126165f4e2b1140105a`).
  T12 updated the [final human usage plan](../../docs/vnext-human-test-plan.md)
  (prepared, not executed), the acceptance/verification disposition table, and
  the OpenCode README/usage/adoption/recovery/verification/supported-version
  records. No model call, native `opencode acp` process, provider probe,
  credential read, human usage run, live install/adoption, approval/review write
  or release tag occurred.
- Container acceptance (2026-11-02): after the AC-14/B5 scope check, the user
  chose the pinned container as the accepted human-cycle environment (spec
  §Approval). `tests/qa-container/` + `tests/build-qa-image.sh` build
  `agentic-sdd-qa:005` with the local Ubuntu base, OpenCode 2.2.5, the
  `agentic-sdd` binary, and the 18 packaged skills preinstalled; the build runs
  with `--network=none`. Verified offline: `opencode v2.2.5`, `git 2.55.0`, 18
  skills, candidate file; online connectivity check reported 3 passed/0 failed/
  1 skipped. Image identity and binary hashes are recorded in spec §Approval.
  No login or model call was performed by the agent.
- Next permitted action: run the accepted human cycle in the pinned image
  (`tests/build-qa-image.sh`, then `SDD_QA_WORKSPACE=... tests/run-qa-image.sh`,
  add `--online` for user-driven login). Record results in the
  [run sheet](../../docs/vnext-human-cycle-run-sheet.md). No further batch,
  real-home install/adoption or model call is authorized by this delivery.
- Verification: Go tests/vet, repository-required gates and documentation
  whitespace/local-link/coverage reconciliation passed. No Go/fixture/native/CI
  tests run merely as human-usage substitutes.
