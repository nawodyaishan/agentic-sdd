# Feature 015 — Tasks (r1 approved 2026-11-12; MCP amendment pending approval)

**Parents:** [`spec.md`](spec.md) (draft r1; MCP amendment r1 pending) + [`plan.md`](plan.md) (draft r1; §8 MCP amendment pending).
**Approval:** Combined spec/plan/tasks **approved by owner on 2026-11-12**: “3 docs approved / update task batches docs in a way to use maximum parallel implement sub agents using free models deepseek high gemini flash / and start the work”. Wave 1 lanes W1-A, W1-B and W1-C are authorized. B4 design is approved but its source-write implementation remains separately gated. No real-home, private-Git, model/network, package-install, release or 014-gate override is authorized. **MCP-A (T26–T30) was implemented 2026-11-12 and is `awaiting human review`.** **MCP-B–MCP-D are authorized 2026-11-12 for parallel implementation**, including MCP-B's write surface against isolated temporary-home fixtures only. No real-home MCP configuration write, network/OAuth, package installation, real-home change or release is authorized.
**Baseline:** Wave 3 commit `74f568b`; MCP amendment reconciled at `4c2b9bd` (clean tree); Feature 013 accepted/published `v0.8.0`; Feature 014 implemented scope awaiting human/Opus acceptance, batch ledger unchanged; 014 FR-15 adoption deferred; Fleet live capability default `preview`.
**Execution rule:** one explicitly authorized wave at a time; lanes inside a wave run in parallel only with the file ownership below. Each wave ends `awaiting human review` with a report; the next wave starts only after integration and review.
**Boundary:** offline isolated-home fixtures only (`t.TempDir`, `GOPROXY=off`, `GOWORK=off`, `GOTOOLCHAIN=local`). No real `~/.config/opencode`, private Git, model call, network, paid inference, trusted-CI adoption, tag, cask/tap push or release. A skip/blocked check is never `pass`.

## Parallel execution waves

| Wave | Lane | Model lane | Tasks | File ownership | Depends on | State |
|---|---|---|---|---|---|---|
| **W1** | **W1-A — root fleet read engine** | DeepSeek high (`sdd-implement-1`) | T01, T06, T10, T18 (engine halves) | `internal/fleet/{status,diff,restore_preview}.go` + new tests; do not edit existing engine files except a reported tiny helper | approved | `accepted by owner 2026-11-12 — integration green` |
| **W1** | **W1-B — TUI shell + read-only Fleet** | Gemini Flash High (`sdd-implement-9r-1`) | T04, T05 | `experiments/tui/{app,app_screens,config}.go`, new `fleet.go`, `fleet_screens.go`, `fleet_adapter.go`, new tests; no `ops_host.go`/`modal.go` | approved | `accepted by owner 2026-11-12 — integration green` |
| **W1** | **W1-C — OpsHost fleet write plumbing** | Free (`sdd-implement-free-1`) | T12 (host/plumbing half) | `experiments/tui/ops_host.go` + new `ops_host_fleet_test.go` only | approved | `accepted by owner 2026-11-12 — integration green` |
| **W2** | **W2-A — appops full seam + CLI** | DeepSeek high (`sdd-implement-1`) | T02, T03, T07, T11, T12 (appops half), T18 (appops half) | `internal/appops/fleet.go`, `cmd/agentic-sdd/fleet*.go` + tests | W1-A | `accepted by owner 2026-11-12 — integration green` |
| **W2** | **W2-B — TUI staging and source review** | Gemini Flash High (`sdd-implement-9r-1`) | T08, T09 | `experiments/tui/fleet_edit.go`, `fleet_review.go`, related tests; no `ops_host.go` | W1-B, W2-A status/source | `accepted by owner 2026-11-12 — integration green` |
| **W2** | **W2-C — TUI deployment and recovery screens** | DeepSeek high (`sdd-implement-1`) | T12 (UI half), T13, T19 | `experiments/tui/fleet_write.go`, `fleet_recovery.go`, `fleet.go`/`fleet_screens.go`, `ops_host.go` wiring, `app.go`, related tests | W1-C, W2-A, W2-B | `accepted by owner 2026-11-12 — integration green` |
| **W3** | **W3-A — doctor + live capability wiring** | DeepSeek high (`sdd-implement-1`) | T20 + T13 build wiring | `experiments/tui/fleet_doctor.go`, `fleet.go`/`fleet_screens.go` doctor wiring, `internal/buildinfo`, `cmd/tui-companion`/`tuirun`, tests | W2 | `awaiting human review — integration green 2026-11-12` |
| **W3** | **W3-B — scale, performance and negative evidence** | Gemini Flash High (`sdd-implement-9r-1`) | T21 + negative matrix | new test/benchmark files only; no production edits | W2 | `awaiting human review — integration green 2026-11-12` |
| **W3** | **W3-C — PTY, accessibility and paired artifact** | Free lane (partial) + parent completion | T22, T23 | `tests/tui`, `Taskfile.experimental.yml`, evidence files | W2 | `awaiting human review — integration green 2026-11-12` |
| **W3** | **W3-D — docs, help and onboarding** | Free lane (partial) + parent completion | T24 | `library_help.go`, `onboarding.go` copy, README/docs; no fleet screen logic | W2 | `awaiting human review — integration green 2026-11-12` |
| **W3** | **W3-E — completion packet** | Parent + owner | T25 | `specs/015-fleet-manager-tui/w3-acceptance.md`; no product scope | W3-A–D | `complete — owner review pending` |
| **B4** | **B4 — native source save** | separately approved lane | T14–T17 | `internal/fleet/save.go`, appops save, CLI `fleet config save`, TUI save | explicit owner go-ahead | `not authorized` |

**Split-task rule:** T11/T12/T18 span layers. W1-C owns only the `OpsHost` request/plumbing half of T12, W1-A owns only the engine half of T18, and W2-A owns the appops halves plus T11. No lane edits another lane's files; integration is the parent agent's job.

### Wave 2 evidence (2026-11-12)

| Lane | Delivered | Verification |
|---|---|---|
| **W2-A** | `internal/appops/fleet.go` + tests: status, source preview, plan preview/apply, restore preview/apply, doctor, tokens, no-adopt. CLI `fleet status` refactored onto `fleet.BuildStatus`; golden output tests added. Additive `PriorModel` in `internal/fleet/status.go`. | `go test ./internal/appops ./cmd/agentic-sdd` pass; golden stdout/exit parity for 4 fixtures; 10-package root test pass. |
| **W2-B** | TUI staging/model picker/enablement consequence/source review, dirty protection, four view constants, `FleetEditor` seam, long-diff paging, stale-reply tests. | Nested `go test ./...` and `go vet` pass; 11 new tests. |
| **W2-C** | Production `AppOpsFleetReader`/`Editor`/`Planner`, deployment review + full diff + default-No apply, Recovery backup list/preview/restore, live gate, `FleetWriteRequest.Plan`, production OpsHost seams, catalog/Fleet refresh. | Nested `go test ./...`/`go vet` pass; fixture-home apply through production host; `task test`/`task vet` pass. |
| **Integration** | Root and nested trees compile together; no remaining ownership conflict. | `task test` pass; `task vet` pass; nested `go test ./...`/`go vet` pass; `git diff --check` clean. |

**Not run:** PTY/`tui:pty-test`, paired artifact, owner walkthrough, accessibility review, scale/1,000-line dedicated evidence. **Not authorized:** B4 source save, real home, private Git, network/model calls, release. Wave 2 was accepted by the owner on 2026-11-12; Wave 3 then ran and is now `awaiting human review`.

### Wave 3 evidence (2026-11-12)

Full packet: [`w3-acceptance.md`](w3-acceptance.md).

| Lane | Delivered | Verification |
|---|---|---|
| **W3-A** | Doctor panel (`d`) with offline default and explicit local binary/dump; `FleetCapability` default `preview`, `live` only by build injection; no runtime toggle. | 11 doctor tests pass; buildinfo live/default tests pass; nested suite green. |
| **W3-B** | 18/100-instance scale tests, 30/30 negative matrix, 100-row p95 measurement, >1,000-line diff. | `go test ./internal/fleet` pass; nested scale/negative tests pass; p95 263 µs. |
| **W3-C** | Fleet PTY lifecycle/resize/literal-input harness; paired artifact. | `tui:fleet-pty` 4/4 pass; `tui:pair` builds; protocol/version inspected. |
| **W3-D** | Fleet Help topic (9), Quick Start pointer, optional onboarding hint, user guide, README pointer. | Nested help/onboarding tests pass; copy reviewed for preview-only truthfulness. |
| **Integration** | Root and nested trees green; existing PTY/launcher suites unchanged. | `task test`/`task vet` pass; nested `go test`/`go vet` pass; `tui:pty-test` 10+13 pass; `git diff --check` clean. |

**Not run / not authorized:** owner walkthrough (AC-37), real macOS install, live apply/restore, B4 source save, 014 acceptance, release. Wave 3 is `awaiting human review`; the packet recommends **preview-only / no-go for live release**.

### Post-review fixes (2026-11-12)

- **M-01 (Missing `--fleet-config` flag):** Fixed in code (`cmd/agentic-sdd/tui.go`, companion launcher, `TUIConfig.FleetConfig`) and tests.
- **M-02 (Top-level metadata loss):** Fixed in code/tests; the editor reads the selected source and preserves top-level `fleet`/`preset`; arbitrary formatting/key order remains a B4 concern.
- **L-01 (Wide detail focus):** Fixed in code and tests (110-col detail pane focus and bounds).
- **L-02 (AC overclaims):** Fixed in docs (`w3-acceptance.md` marks AC-15 and AC-20 as partially covered and AC-33 as partial).
- **L-03 (Model-clear limitation):** Documented limitation; clearing an existing model override back to canonical inheritance is a CLI/editor operation in v1 (`docs/fleet-manager-tui.md`).
- **A3-1 (Inline confirmations):** Documented in spec/plan; Fleet apply/restore confirmations are inline subviews with default-No and focus traps; `modal.go` remains global compositor.
- **Post-fix verification:** `task test`/`task vet`, nested `go test`/`go vet`, `tui:fleet-pty` (4/4), `tui:pty-test` (10+13), and `git diff --check` pass.

### Wave 1 evidence (2026-11-12)

| Lane | Delivered | Verification |
|---|---|---|
| **W1-A** | `internal/fleet/status.go`, `diff.go`, `restore_preview.go` + tests: `BuildStatus`, `EncodeConfig`, `PreviewSource`, `PreviewSourceWithRepo`, `PlanFiles`, `LineDiff`, `PreviewRestoreBackup`. | `go test ./internal/fleet` pass; `go vet` pass; `gofmt` clean; read-only fixture assertions. |
| **W1-B** | Third hub child, `len(entries)` navigation, lazy read-only `FleetModel`, Overview/Profiles, `FleetReader` seam, responsive tests. | Nested `go test ./...` and `go vet ./...` pass; zero-write/temp-home tests pass. |
| **W1-C** | `WriteFleetApply`/`WriteFleetRestore`/`WriteSourceSave`, `FleetWriteRequest`, derived command/source/home, nil-seam refusal, single-slot tests. | Nested `go test ./...` pass; existing `TestOpsHost*` unchanged; transcript/identity tests pass. |
| **Integration** | Root and nested trees compile together; no file-ownership conflict. | `task test` pass (10 packages); `task vet` pass; nested `go test ./...` pass; `git diff --check` clean. |

**Not run in Wave 1:** PTY/`tui:pty-test`, paired artifact, 100-instance/1,000-line fixtures, accessibility review, owner walkthrough. These remain W3/B6. **Not authorized:** real home, private Git, network/model calls, B4 source save, release. Wave 1 was accepted by the owner on 2026-11-12; Wave 2 then ran and was accepted on 2026-11-12.

## Batch ledger (scope groups)

| Batch | Scope | Depends on | State |
|---|---|---|---|
| **B1** | Read-only foundation: status projection, CLI status refactor, third hub child, Overview + Profiles | W1-A, W1-B, W2-A | `accepted 2026-11-12` |
| **B2** | Staging and review: source preview, model picker, dirty protection, source diff | W1-B, W2-A | `accepted 2026-11-12` |
| **B3** | Deployment: plan/diff, default-No apply, OpsHost, live gate | W1-A, W1-C, W2-A, W2-C | `accepted 2026-11-12` |
| **B4** | Source save: atomic CAS write, CLI parity, checkpoint | W2-A + **separate approval of the write surface** | `not authorized` |
| **B5** | Recovery: restore preview/apply, drift/legacy/conflict, doctor | W1-A, W2-C, W3-A | `awaiting human review — doctor/recovery green` |
| **B6** | Robustness/docs: scale, PTY, accessibility, artifact, walkthrough | W1–W3 | `awaiting human review — packet at w3-acceptance.md` |

**Parallelism:** Wave 1 uses all three requested model lanes (DeepSeek high, Gemini Flash, free). Wave 2 fans out appops/CLI, TUI staging and TUI write screens on disjoint files. Wave 3 runs verification/docs. B4 stays outside the waves until explicitly authorized.

---

## B1 — Read-only foundation

- [x] **T01 — `internal/fleet/status.go`: typed read-only status.**
  - Implement `BuildStatus(configPath, repo, home) (*Status, error)` from `LoadConfigFile`, `Resolve`, `ReadLiveManifest` (fallback `RebuildLiveManifest`, both read-only), target `Lstat`, digests, `TargetRootForHome`, `last-applied` records.
  - Per-instance state: configured model, rendered model, enabled, source ref/digest, desired/current/last-applied digests, `absent | managed-current | managed-drifted | pending-apply | unmanaged | legacy-unknown | unsafe-target | disabled`, conflict reason; aggregate counts.
  - No writes, no plan, no lock acquisition, no `ApplyPlan`, no Git, no model/network.
  - **Accept:** AC-06, AC-07, AC-22 (read side), FR-02; 0/1/18/100 fixtures; no target created/modified; `BuildStatus` does not call `ApplyPlan`/`Sync`.
  - **Verify:** `go test ./internal/fleet -run TestFleetStatus`; read-only fixture assertion.
- [x] **T02 — CLI status refactor.**
  - Replace the inline projection in `cmd/agentic-sdd/fleet.go` with `fleet.BuildStatus`; keep text output and exit codes byte-identical.
  - **Accept:** existing `fleet status` tests pass unchanged; AC-29; no new flags/grammar.
  - **Verify:** `go test ./cmd/agentic-sdd -run TestFleet`.
- [x] **T03 — `internal/appops/fleet.go`: `FleetStatus` request/result.**
  - Immutable, validated, absolute paths; accessors; read-only. No preview token for status.
  - **Accept:** AC-27; screens have a typed read call; no transcript parse.
  - **Verify:** `go test ./internal/appops -run TestFleetStatus`.
- [x] **T04 — TUI hub child and read-only Fleet views.**
  - Add `installSectionFleet` as third child; use `len(entries)` for forward/backward cursor wrap and bounds, not hardcoded 2; Enter dispatches each of the three entries; key `f`; breadcrumb; Home affordance; preserve existing Install/Backups behavior. Update all hub navigation/rendering tests in B1 (L3).
  - `FleetModel` + adapter: Overview (source choice, two-state summary, counts, next action) and Profiles (search/filter/list/detail, role vs instance, permission summary from engine).
  - No write buttons, no direct filesystem, no CLI subprocess.
  - **Accept:** FR-01; AC-01, AC-02, AC-03, AC-07, AC-12, AC-17, AC-27; all hub tests cover three-entry cursor wrap/bounds, Enter and `f`; zero-write assertion.
  - **Verify:** `task -t Taskfile.experimental.yml tui:test`.
- [x] **T05 — Responsive/generation/zero-write foundation tests.**
  - 110/80/60 widths, heights 8/10/16, tiny-terminal guidance; late status reply for another path/generation is discarded; source choice does not scan home; no fleet read when Fleet was never opened.
  - **Accept:** AC-04, AC-05 (read side), AC-31; NFR-01/NFR-02/NFR-09.
  - **Verify:** nested tests + PTY smoke where available; record `not_run` if blocked.

## B2 — Staging and review

- [x] **T06 — `internal/fleet/diff.go` + deterministic encoding.**
  - `EncodeConfig(*Config)` with stable field order; `PreviewSource(path, newBytes)` loads current bytes/digest, validates new bytes with `ParseConfig`, resolves, renders, returns `FilePreview`; `LineDiff` bounded and pure.
  - **Accept:** AC-08, AC-15, AC-19 (source half); long paths/Unicode; 1,000+ lines reachable without unbounded allocation.
  - **Verify:** `go test ./internal/fleet -run 'TestFleet(Encode|PreviewSource|LineDiff)'`.
- [x] **T07 — `appops.FleetSourcePreview` (pure read).**
  - Takes `(configPath, repo, home, draftBytes)`; validates; returns normalized bytes + `FilePreview` + base digest. `appops` retains no draft between calls; draft lives only in `FleetModel`; no adopt; no write.
  - **Accept:** FR-07; AC-14, AC-15, AC-27; M2; independent preview calls cannot retain or leak draft state.
  - **Verify:** `go test ./internal/appops -run TestFleetSourcePreview`.
- [x] **T08 — Model picker, enablement, dirty protection.**
  - Picker groups current/local assignments/cataloged (only from an explicit local source)/exact entry; tags by provenance, no price/auth.
  - Enable/disable consequence dialog; managed removal + backup wording; unmanaged cannot be removed.
  - `N unsaved edits`; Esc/navigation/modal/resize keep/discard/stay; never implicit write.
  - **Accept:** AC-05, AC-08, AC-09, AC-10, AC-11, AC-14.
  - **Verify:** nested tests for picker/draft/dirty paths.
- [x] **T09 — Source review and generation discipline.**
  - Review Changes source view: impact summary first, exact JSON diff, target path/digest, no save yet; stale/external-edit state; late preview cannot clear draft or overwrite a newer plan.
  - **Accept:** AC-15, AC-17, AC-21.
  - **Verify:** nested tests including late-message fixtures.

## B3 — Deployment

- [x] **T10 — `fleet.PlanFiles` read-only diff projection.**
  - Rehydrate the plan from `plan.ConfigPath`/`RepoPath`, `Resolve` + `RenderResolved`, read current targets; return old/new bytes, digests, op, conflict reason; removals/conflicts ordered first.
  - Reuse `ApplyPlan` validation helpers where pure; do not write or lock.
  - **Accept:** AC-19, AC-22 (deployment read side); create/update/remove/noop/conflict fixtures; permission body unchanged.
  - **Verify:** `go test ./internal/fleet -run TestFleetPlanFiles`.
- [x] **T11 — `appops.FleetPreview` with token.**
  - Fresh `BuildPlan(configPath, repo, home, false)`; return plan + files + source digest + token; token covers plan ID/config digest/source revision/target hashes/scope.
  - No adopt field; no file write.
  - **Accept:** FR-12; AC-19, AC-21, AC-26; H2; no adopt field/parameter or adopting plan; explicit 014 CLI adoption stays outside 015.
  - **Verify:** `go test ./internal/appops -run TestFleetPreview`.
- [x] **T12 — `OpsHost` fleet apply + Review deployment view.**
  - Add `WriteFleetApply`, request validation, resolved request home via `home()` to `openFileWriteLog`, non-`embedded` fleet `source()` and `Command` header `fleet apply --plan <planID>` (in-memory plan identity, not a plan file); assert transcript contents (M3/M4).
  - Extend `startInventoryRefresh()` for fleet apply: refresh catalog and Fleet status; assert generation-bound fresh results and rejection of stale replies (L1).
  - All five write kinds (existing install/restore plus fleet source-save/apply/restore) use one `OpsHost` slot. Test active apply versus existing install/restore in both directions, explicit busy refusal and no queued second write; T17/T19 complete the cross-kind matrix (L2 / r2 AC-46).
  - Review view: scope, counts, plan ID, full scrollable diff, default-No apply, real result/no-op/conflict/rollback; no optimistic success.
  - **Accept:** FR-16, FR-21; AC-19, AC-20, AC-21, AC-23, AC-25, AC-27, AC-28; double-Enter/resize/navigation negatives; transcript/refresh/slot assertions above.
  - **Verify:** nested tests + fixture-home apply; `fleet.ApplyPlan` only inside `appops`/host.
- [x] **T13 — Live-capability gate and preview-only build.**
  - One reviewed release-metadata build capability (for example `internal/buildinfo`, default `preview`) → `TUIConfig.FleetLive`; no env/config/key/CLI override. When false: apply/restore absent or disabled with `Preview only`; no key sequence dispatches.
  - Tests inject `FleetLive=true` for apply and restore UI and assert the shipped default disables/hides both. Do not change `.goreleaser.yml`/release workflow to `live`; that requires 014 acceptance and a separately authorized release decision, outside 015 implementation.
  - **Accept:** AC-36; FR-10, FR-11, FR-20; H1; no hidden release activation; B5/T19 exercises the restore half.
  - **Verify:** build/variant tests + nested tests.

## B4 — Source save (separately gated — `not authorized`)

- [ ] **T14 — `internal/fleet/save.go`: atomic CAS save (`not authorized`).**
  - Existing regular file only; `SaveOptions.Home` is the resolved home; required non-empty `ExpectDigest`; empty is usage/refusal, mismatch is `ErrStaleSource`, neither writes. Strict parse; `0600` temp in same dir; `fsync`; atomic replace where supported (refuse otherwise); mandatory pre-save backup for a changed file under `<home>/.agentic-sdd/fleet-source-backups/`, outside checkout and skillsync backup namespace; no Git; no new-file creation.
  - **Accept:** FR-06, FR-08, FR-21; AC-16, AC-17; M1/M3; empty/mismatched digest and nonexistent-file refusals; negative rows 7, 8, 30; permissions/mode tests; no installed-file change.
  - **Verify:** `go test ./internal/fleet -run TestFleetSave`.
- [ ] **T15 — `appops.FleetSourceSave` (`not authorized`).**
  - Immutable request with resolved home + config path + required non-empty expected digest + new bytes; pass home to `SaveOptions.Home`; token revalidation never substitutes for CAS or masks `ErrStaleSource` on a source mismatch; result path/digest/backup/NOOP.
  - **Accept:** FR-08, FR-21; AC-16, AC-17, AC-27; M1/M3; no CLI subprocess; no installed change.
  - **Verify:** `go test ./internal/appops -run TestFleetSourceSave`.
- [ ] **T16 — CLI `fleet config save` (`not authorized`).**
  - Grammar in plan §2.3; required non-empty `--expect-digest`, `--dry-run`, `--home`; backup is mandatory for a changed file, not an opt-out flag; help/usage; exit codes `2/1/0`; byte-identical existing fleet commands.
  - **Accept:** FR-15; AC-16, AC-17, AC-18; M1; CLI/TUI parity; parser tests for empty digest (usage), mismatch (`ErrStaleSource`) and nonexistent-file refusal.
  - **Verify:** `go test ./cmd/agentic-sdd -run TestFleetConfigSave`; help fixture.
- [ ] **T17 — TUI save confirmation and checkpoint (`not authorized`).**
  - Exact diff/target; default-No save; "Saved locally. OpenCode files unchanged."; `Preview OpenCode changes`; refresh Fleet status; failure shows what happened/what stayed unchanged/next action.
  - Wire `WriteSourceSave` through the one `OpsHost` slot; `home()` returns the save request's resolved home to `openFileWriteLog`; `source()` is not `embedded`; assert `fleet config save` transcript header even with config outside home (M3/M4).
  - Assert fresh Fleet status after save, stale replies discarded and installed state unchanged (L1). Test source-save against every other write kind in both directions with explicit busy refusal and no queued second write (L2 / r2 AC-46).
  - **Accept:** FR-16, FR-21; AC-15, AC-20, AC-23, AC-27, AC-28.
  - **Verify:** nested tests for save success/stale/refusal.

## B5 — Recovery

- [x] **T18 — `PreviewRestoreBackup` + appops restore preview/apply.**
  - Read-only current-vs-restore delta, conflict/ownership checks, shared validator with `RestoreBackup`; token covers backup ID/config/target hashes; no `dryRun` flag added to the writer.
  - **Accept:** AC-24, AC-25; preview/apply parity; legacy/corrupt refusal.
  - **Verify:** `go test ./internal/fleet -run TestFleetRestorePreview ./internal/appops -run TestFleetRestore`.
- [x] **T19 — Recovery view, `WriteFleetRestore`, drift/legacy/conflict.**
  - Managed backups only; exact effects; default-No restore; three-state drift; `what happened / what stayed unchanged / next safe action`; stale/modified refusal; post-restore refresh.
  - Gate restore on the same reviewed build capability as apply: default `preview` disables/hides it with `Preview only` and no key dispatch; tests inject live=true (H1). No adopt/force/fix-all; adoption stays explicit 014 CLI only (H2).
  - For `WriteFleetRestore`, `WriteRequest.home()` returns the resolved restore-request home to `openFileWriteLog`; `source()` is not `embedded`; assert `fleet restore <backupID>` header (M3/M4).
  - `startInventoryRefresh()` handles fleet restore; assert catalog plus Fleet status refresh and stale-reply rejection (L1). Complete all five kinds' pairwise busy-refusal matrix in both directions, including restore versus existing install/restore and fleet apply/source-save (L2 / r2 AC-46).
  - **Accept:** FR-10, FR-11, FR-12, FR-16, FR-20, FR-21; AC-22, AC-23, AC-24, AC-25, AC-26, AC-27, AC-28, AC-36.
  - **Verify:** nested tests + fixture-home restore/rollback.
- [x] **T20 — Opt-in doctor.**
  - Explicit local `--opencode-bin` selection; offline default `not checked`; labels `configured/catalog-visible/authenticated/unknown/unavailable` only from local evidence; no PATH/network/model.
  - **Accept:** AC-09, AC-12 (runtime facts); AC-24; negative row 24.
  - **Verify:** `go test ./internal/fleet -run TestFleetDoctor`; nested doctor tests.

## B6 — Robustness, docs and acceptance

- [x] **T21 — Scale, diff reachability and performance.**
  - 18/100 instances, 100 ops, >1,000 diff lines, Unicode/long paths; p95 ≤100 ms in-memory target; startup delta vs 013; bounded memory.
  - **Accept:** AC-32; NFR-01/NFR-03/NFR-09.
  - **Verify:** nested benchmarks/fixtures; record host and results.
- [x] **T22 — PTY, resize, keyboard and accessibility.**
  - 110/80/60 and 8/10/16; confirmation at 50 columns; type-through `j/k/q//`; modal focus; plain/ASCII/no-color/static; Back/Cancel/warning reachable.
  - **Accept:** AC-04, AC-05, AC-33.
  - **Verify:** `task -t Taskfile.experimental.yml tui:pty-test`; record blocked/not_run honestly.
- [x] **T23 — Paired artifact and regressions.**
  - Root/nested builds, paired candidate invocation, `--fleet-config` handshake, 013/014 regressions, 10-package gate, no new root `require`, 009 14-profile behavior.
  - **Accept:** AC-28, AC-29, AC-30, AC-34, AC-35.
  - **Verify:** `task test`, `task vet`, `git diff --check`, `task -t Taskfile.experimental.yml tui:pair`.
- [x] **T24 — Documentation and onboarding.**
  - Help topic/Quick Start, first-use options (existing file, read-only example, CLI guide), two-write explanation, `installed ≠ authenticated ≠ primary`, preview-only wording, 014 CLI links, no private path exposure.
  - **Accept:** AC-02, AC-13, AC-37 (docs half); FR-22.
  - **Verify:** help/text fixtures; review of copy.
- [ ] **T25 — Owner walkthrough and completion packet** (packet complete; owner walkthrough pending).
  - Temporary-fixture journey (source → model change → save → plan → apply → restore), modified-file conflict, no-config path; collect SHA, commands, Pass/Fail/Blocked/Not run, screenshots/PTY only if produced, known gaps, no-go/preview-only disposition.
  - **Accept:** AC-37, AC-38; whole-feature evidence.
  - **Verify:** manual owner review; no release/tag/real-home operation.

---

## MCP integration amendment workstreams (pending approval — not authorized)

> **Status:** the independent reviewer recommends **MCP-A only** now; MCP-B–MCP-D are deferred and no task below is authorized. They do not extend Waves 1–3, do not repurpose B4, and do not authorize code, config writes, package installation, network, OAuth, model calls, real-home changes, commit, tag or release. MCP-A may be developed/tested separately after amendment approval; MCP-B requires a **separate pre-start MCP write-surface authorization**; MCP-C depends on A (and B for write screens); MCP-D depends on A–C. Feature 014 acceptance, Wave 3 review, B4, Fleet live and release remain separate gates.

### MCP workstream ledger

| Workstream | Scope | Tasks | Depends on | Owner checkpoint | State |
|---|---|---|---|---|---|
| **MCP-A — Read-only foundation** | Metadata, target/config detection, offline status/doctor, read-only static preview, state vocabulary, CLI `status`/`doctor`/`preview` | T26–T30 | **Authorized 2026-11-12**; no write authorization | CP-A after T30 | `awaiting human review — integration green` |
| **MCP-B — Safe configuration management** | Structured preview/digest, JSONC byte-range merge, ownership manifest, backup, transactional apply/restore, runtime evidence, cross-process lock, failure tests | T31–T37 | **Authorized 2026-11-12**; explicit MCP write-surface authorization recorded | CP-B after T37 | `authorized — parallel implementation` |
| **MCP-C — TUI experience** | Hub child, Overview, selection, preview, confirmation, diagnostics/recovery, single-write host | T38–T42 | **Authorized 2026-11-12**; MCP-B interfaces | CP-C after T42 | `authorized — parallel implementation` |
| **MCP-D — Verification and closeout** | Root/nested gates, PTY/headless, offline/temp-home evidence, docs/onboarding, acceptance packet | T43–T46 | **Authorized 2026-11-12**; MCP-A–C | CP-D after T46 | `authorized — parallel implementation` |

**Prerequisites and boundaries:** P0 is the default entry point; P1 is opt-in and never implied by `--with-opencode`, an agent pack or opening Fleet Manager; P2 installation is deferred. Global OpenCode configuration is the only v1 target. No `fleet.json` extension, no second permission/fallback registry, no `AGENTS.md` rewrite, no vendor installer and no plaintext credentials. Tests use temporary homes, fake `codegraph`/`opencode` executables and no network/model/package calls.

### MCP-A — Read-only foundation

- [x] **T26 — MCP metadata and target/config detection.**
  - Add `internal/skillsync/mcp.go` with the three default server descriptors (CodeGraph local; Context7/Exa remote), target resolution for `<home>/.config/opencode/opencode.json(c)`, `.json`/`.jsonc` ambiguity, V2 `mcp.servers`/V1 `mcp.<name>` detection, read-only JSON/JSONC parsing and duplicate-key/server-entry detection sufficient for the static P0 preview, symlink/non-regular/traversal rejection, CodeGraph CLI/index presence and the state vocabulary from spec §16.6.
  - Read-only: no write, no process execution of `codegraph`/`opencode`, no network, no package manager. `exec.LookPath` is allowed for local prerequisite reporting only; version execution is apply-only in MCP-B, not part of status/preview/offline doctor.
  - **Accept:** AC-51 data foundation (the TUI Overview is MCP-C), AC-52, AC-56, AC-57, AC-61, AC-64, AC-73, AC-75; zero-write assertions.
  - **Verify:** `go test ./internal/skillsync -run 'TestMCP(Status|Detect|Target)'`.
- [x] **T27 — MCP status projection and evidence labels.**
  - Project per-server state/evidence (`static`, `local-probe`, `runtime`, `not-checked`), configured/effective/disabled distinctions, CodeGraph index state (`absent`, `present`, `freshness-not-checked`) and conflict/schema/unsafe states. No inferred `connected`/`authenticated`.
  - **Accept:** AC-51 data foundation, AC-52, AC-56, AC-58, AC-75; fixture states for all three servers.
  - **Verify:** `go test ./internal/skillsync -run 'TestMCPStatus'`.
- [x] **T28 — Offline `mcp doctor`.**
  - Add `internal/appops/mcp.go` doctor request/result with explicit `OpenCodeBin`/`CodeGraphBin`/`ProjectPath`; offline default may use `exec.LookPath` but executes no binary, no network and no model. `--network`/`--connect` are deferred out of MCP-A and `not_run` in default tests; ordinary doctor has no execution path.
  - **Accept:** AC-52, AC-56, AC-57, AC-58, AC-73, AC-75; consistent lookup/no-execution behavior.
  - **Verify:** `go test ./internal/appops -run 'TestMCPDoctor'`; no-process/no-network assertion.
- [x] **T29 — CLI `mcp status`, `mcp doctor` and read-only `mcp preview`.**
  - Add `cmd/agentic-sdd/mcp.go`, isolated `case "mcp"`, `help mcp`, `--home`/`--config`/`--project`/`--json` (status/doctor; `--servers` is preview-only), stable text/exit. `mcp` with no subcommand = status. The P0 preview is a read-only static recipe/comparison with no apply token and no runtime execution; existing commands/flags/output unchanged.
  - **Accept:** AC-51 data foundation, AC-52, AC-53, AC-72; no-TTY/golden behavior; preview writes nothing.
  - **Verify:** `go test ./cmd/agentic-sdd -run 'TestMCP(Status|Preview|CLI|Help)'`; existing CLI goldens.
- [x] **T30 — Read-only compatibility and negative evidence.**
  - Golden configs for not-configured/configured/disabled/V1/both `.json`+`.jsonc`/malformed/unknown server entry/symlink/traversal/missing CodeGraph/index absent; assert no writes and no network. Add `--with-opencode` byte-identity test and Fleet hub regression test.
  - **Accept:** AC-52, AC-53, AC-54, AC-55, AC-56, AC-57, AC-59/AC-60 detection-only (apply preservation/refusal is MCP-B), AC-61, AC-72, AC-73; negative rows 31–41, 43–44, 46.
  - **Verify:** `go test ./internal/skillsync ./internal/appops ./cmd/agentic-sdd -run 'TestMCP'`; `git diff --check`.
  - **Checkpoint CP-A:** owner reviews read-only status/doctor/CLI and confirms no write path exists.

### MCP-A evidence (2026-11-12)

| Lane | Delivered | Verification |
|---|---|---|
| **9Router Gemini 3.8 Flash High** | `internal/skillsync/mcp.go`: default server specs, target/JSON/JSONC/schema/duplicate/UTF-8/oversize detection, V1 refusal, ambiguity/unsafe states, CodeGraph LookPath/index presence, auth-mode detection, read-only static preview; `mcp_test.go` (13 tests). | `go test ./internal/skillsync -run 'TestMCP'` pass; full `internal/skillsync` pass; `go vet` clean. |
| **DeepSeek V4.1 Flash high** | `internal/appops/mcp.go`: immutable status/doctor/preview requests, offline doctor checks, no token/no writes; `mcp_test.go` (21 tests). | `go test ./internal/appops -run 'TestMCP'` pass; `go vet` clean. |
| **DeepSeek V4.1 Flash free** | `cmd/agentic-sdd/mcp.go` + `main.go` dispatch/help: `mcp status|doctor|preview|help`, JSON/text, A-only guards for `apply`/`restore`/`--network`/`--connect`; `mcp_test.go` plus parent `mcp_usage_test.go`. | `go test ./cmd/agentic-sdd -run 'TestMCP'` pass; full `cmd/agentic-sdd` pass; manual temp-home smoke test. |
| **MiMo V2.6 Flash free** | `internal/skillsync/mcp_compat_test.go` (AC-55 `--with-opencode` byte identity/no MCP state) and `experiments/tui/mcp_a_hub_regression_test.go` (AC-54 hub/Fleet reachability). | Focused tests pass; nested `tui:test` pass. |
| **Parent integration** | `skillsync.ErrMCPUsage` + CLI exit-2 mapping; status `--project`/`--servers` grammar reconciliation; ambiguity text rendering; Exa keyless label; docs/approval/ledger. | `task test` pass (10 packages); `task vet` pass; nested `tui:test` pass; `git diff --check` clean. |

**MCP-A files:** `internal/skillsync/mcp.go`, `internal/skillsync/mcp_test.go`, `internal/skillsync/mcp_compat_test.go`, `internal/appops/mcp.go`, `internal/appops/mcp_test.go`, `cmd/agentic-sdd/mcp.go`, `cmd/agentic-sdd/mcp_test.go`, `cmd/agentic-sdd/mcp_usage_test.go`, `cmd/agentic-sdd/main.go`, `experiments/tui/mcp_a_hub_regression_test.go`.

**Partial coverage:** AC-51 is covered only as CLI/status data; the TUI MCP Overview is MCP-C. AC-59/AC-60 are covered only as detection/conflict reporting; apply-time preservation and refusal are MCP-B. Do not claim those criteria complete from MCP-A.

**Not run / not authorized:** `mcp apply`/`restore` (MCP-B), any MCP configuration write, `--network`/`--connect`, OAuth/login, package installation, real-home walkthrough, live remote connectivity, macOS-specific checks. These are `not_run`, not pass. No commit/tag/release was performed.

**Additional verification:** `go test -race` passed for `internal/skillsync`; `internal/appops`/`cmd/agentic-sdd` race builds were blocked by the `/tmp` disk quota (`not_run`). `tui:pty-test` passed (10 terminal + 13 launcher).

### MCP-B — Safe configuration management

- [ ] **T31 — Structured `mcp preview` plan/digest/token (`not_started — pending approval`).**
  - Add `internal/skillsync/mcp_config.go` JSON-only structured preview: target/base/plan digest, add/change/leave/conflict/no-op, selected names, auth implications, prereqs, backup/restore effect. Missing target is `not configured` plus a manual snippet, never creation. No write; the token/apply path is added after T33.
  - **Accept:** AC-59, AC-60, AC-63, AC-67; exact target/digest in output.
  - **Verify:** `go test ./internal/skillsync -run 'TestMCPPreview'`.
- [ ] **T32 — JSONC conservative editor and manual-snippet refusal (`not_started — pending approval`).**
  - Add a stdlib scanner/editor that preserves comments/key order/unrelated formatting where ownership of every touched comment/delimiter is proven; byte-range transformation plus atomic replacement only. Refuse otherwise with an exact copyable snippet. Detect duplicate keys, wrong schema, invalid entries, unsupported structural edits, and comment/delimiter ambiguity.
  - **Accept:** AC-61, AC-62, AC-64; no JSON reserialization of `.jsonc`; missing target is not created; structural/comment fixtures pass.
  - **Verify:** `go test ./internal/skillsync -run 'TestMCPJSONC'`; golden comment/format fixtures.
- [ ] **T33 — Appops MCP preview/apply/restore contract (`not_started — pending approval`).**
  - Add `internal/appops/mcp.go` immutable requests/results, `appops-mcp-preview-v1` token, `ErrNeedsPreview`/`ErrStaleConfig`/`ErrMCPConflict`/`ErrMCPUnsafeTarget`/`ErrMCPUnsupported`, stateless reads and write revalidation. No adopt/replace field.
  - **Accept:** AC-52, AC-63, AC-69; token covers target/format/base/plan/selected/entry/auth/runtime evidence/tool version and binds the shared-lock revalidation.
  - **Verify:** `go test ./internal/appops -run 'TestMCP(Preview|Apply|Restore|Token)'`.
- [ ] **T34 — Transactional `mcp apply` (`not_started — pending approval`).**
  - Add `internal/skillsync/mcp_transaction.go`: shared engine lock covering target + ownership manifest, bounded `opencode --version` evidence (`--opencode-bin` or one resolved executable), durable transaction intent, private backup `<home>/.agentic-sdd/mcp/backups/<stamp>/` (`0700`/`0600`) containing only managed-entry text/range plus metadata, separate ownership manifest, same-directory temp/fsync/rename/fsync-dir, digest CAS, no-op, structural rollback. Refuse literal credentials in managed entries, non-atomic filesystems/unsafe targets.
  - **Accept:** AC-59, AC-60, AC-63, AC-65, AC-66, AC-67; backup/mode/digest/lock/runtime assertions; missing target refused, no creation.
  - **Verify:** `go test ./internal/skillsync -run 'TestMCPApply'`; injected failure tests.
- [ ] **T35 — Managed restore and ownership (`not_started — pending approval`).**
  - Add restore list/preview/apply: only managed names, entry-level comparison from managed-entry snapshots, preserve later user edits/unrelated fields, conflict/refusal, exact recovery evidence. Whole-file restore is not part of v1; structural ambiguity refuses with a manual snippet/diff.
  - **Accept:** AC-65, AC-66, AC-68; scenario 42; container/comment/delimiter structural fixtures.
  - **Verify:** `go test ./internal/skillsync -run 'TestMCPRestore'`.
- [ ] **T36 — Failure-path and negative matrix (`not_started — pending approval`).**
  - Cover malformed JSON, invalid entries, overlapping names, wrong schema, missing binary, disabled server, partial write, stale plan, unsupported runtime, JSONC, symlink/traversal, user-edit restore, cross-process competing writers, and structural recovery at each intent phase. Assert exact bytes/rollback evidence and no false success.
  - **Accept:** AC-59–AC-68, AC-73; negative rows 31–46 plus lock/structural fixtures.
  - **Verify:** `go test ./internal/skillsync ./internal/appops -run 'TestMCP'`.
- [ ] **T37 — CLI `mcp preview|apply|restore` (`not_started — pending approval`).**
  - Add parser/renderers, `--expect-config` empty=2/mismatch=1, explicit-path runtime probe, `--json`, `restore list|preview|apply`, no-TTY script behavior. Existing CLI output unchanged; no `--with-mcp`.
  - **Accept:** AC-53, AC-63, AC-65, AC-68, AC-72; exit/golden tests.
  - **Verify:** `go test ./cmd/agentic-sdd -run 'TestMCP(Preview|Apply|Restore)'`.
  - **Checkpoint CP-B:** the pre-start MCP write-surface authorization must already be recorded; owner then reviews root preview/apply/restore, lock/runtime evidence and backup/rollback. Evidence acceptance is separate from the pre-start authorization.

### MCP-C — TUI experience

- [ ] **T38 — Install & Manage hub fourth child and MCP Overview (`not_started — pending approval`).**
  - Add `installSectionMCP`, `installHubEntries` fourth entry (`t`; `m` is the global motion toggle and is not used), all `len(entries)` wrap/bounds, Enter/`t` dispatch, breadcrumb/Home/Esc, lazy status load. Overview renders three rows with required fields.
  - **Accept:** AC-51, AC-54, AC-70; hub/Fleet regression tests.
  - **Verify:** `task -t Taskfile.experimental.yml tui:test`.
- [ ] **T39 — Server selection and preview (`not_started — pending approval`).**
  - Add `experiments/tui/mcp.go`/`mcp_screens.go`/`mcp_adapter.go`: checkbox selection, Context7 OAuth/API-key choice, Exa keyless acknowledgement/OAuth/API-key choice, prerequisite warnings, exact target/digest, diff, conflicts, backup/restore effect, scroll/filter, narrow layout.
  - **Accept:** AC-51, AC-59, AC-60, AC-62, AC-70; no filesystem/network in adapter.
  - **Verify:** nested MCP headless tests; responsive 110/80/60 and 8/10/16.
- [ ] **T40 — Confirmation and single-write host (`not_started — pending approval`).**
  - Add `WriteMCPApply`/`WriteMCPRestore`, `MCPWriteRequest`, validation, `home()`/`source()`/`Command`, one slot, pairwise busy refusal vs install/restore/fleet apply/restore (and the source-save seam when enabled). `OpsHost` is process-local; the engine lock covers CLI/TUI concurrency. Confirmation is inline/default-No/focus-trapped; result is real only.
  - **Accept:** AC-65, AC-67, AC-69, AC-71; no optimistic success.
  - **Verify:** `cd experiments/tui && go test ./... -run 'TestOpsHostMCP|TestMCPConfirm'`.
- [ ] **T41 — Diagnostics and recovery (`not_started — pending approval`).**
  - Add contextual diagnostics (CodeGraph missing/index; Context7 auth pending; Exa rate limit; schema mismatch; unmanaged conflict; JSONC; stale plan; failed/rollback) and Recovery backup list/preview/restore. Connection state is `not tested` with CLI guidance; **no active TUI connection test in v1**.
  - **Accept:** AC-56, AC-57, AC-58, AC-66, AC-68, AC-75.
  - **Verify:** nested diagnostics/recovery tests; no-network assertion.
- [ ] **T42 — Responsive, focus, resize and noninteractive TUI evidence (`not_started — pending approval`).**
  - Preserve focus/scroll/selection/view across overlays/back; resize cannot confirm; plain/ASCII/reduced-motion; noninteractive launch keeps CLI semantics; no fifth route. Add the MCP PTY scenario under `tests/tui` and its explicit `Taskfile.experimental.yml` wiring (no default gate change).
  - **Accept:** AC-54, AC-70, AC-71, AC-72; PTY evidence where available.
  - **Verify:** `task -t Taskfile.experimental.yml tui:test`; the new MCP PTY task; `git diff --check`.
  - **Checkpoint CP-C:** owner reviews TUI navigation/preview/confirm/diagnostics and single-slot evidence.

### MCP-D — Verification and closeout

- [ ] **T43 — Root, nested and gate verification (`not_started — pending approval`).**
  - Run 10-package core tests/vet, nested tests/vet, golden/CLI/TUI suites, PTY MCP scenarios, `git diff --check`; prove no new root `require`, no new package, no gate skip; 013/014/009 regressions.
  - **Accept:** AC-52, AC-53, AC-54, AC-69, AC-73, AC-74.
  - **Verify:** `task test`, `task vet`, nested `go test`/`go vet`, `tui:test`, `tui:pty-test`, the MCP PTY scenario added by T42, `git diff --check`.
- [ ] **T44 — Offline/no-network/temp-home evidence and negative matrix (`not_started — pending approval`).**
  - Record fixture identities, fake binaries, no real home/private Git, no network/model/OAuth/package install; map negative rows 31–46; label static/local/runtime evidence. `--network`/`--connect` remain deferred and `not_run`.
  - **Accept:** AC-52, AC-56–AC-68, AC-73, AC-75.
  - **Verify:** test report with Pass/Fail/Blocked/Not run; `mcp doctor --network`/`--connect` explicitly `not_run`.
- [ ] **T45 — Documentation and onboarding (`not_started — pending approval`).**
  - Help topic, MCP Overview copy, beginner journey (discover → select → preview → confirm → restore), manual JSONC snippet, CodeGraph prerequisite/index guidance, Context7/Exa auth/privacy/rate-limit notes, no private paths/secrets, explicit P2 deferral. README/docs only after implementation evidence.
  - **Accept:** AC-51, AC-56, AC-58, AC-75; copy review.
  - **Verify:** help/text fixtures; docs review; no product claim beyond evidence.
- [ ] **T46 — Paired artifact, macOS where supported, owner walkthrough and packet (`not_started — pending approval`).**
  - Paired candidate build/identity, temporary-home owner journey, diagnostics/conflict/restore, platform limitations, completion packet with go/no-go and next human action. No release/tag/cask/tap/real-home.
  - **Accept:** AC-54, AC-70, AC-71, AC-74; whole-amendment evidence.
  - **Verify:** `task -t Taskfile.experimental.yml tui:pair`; manual owner review; `not_run` where unsupported.
  - **Checkpoint CP-D:** owner acceptance and separate release disposition.

### Deferred MCP capabilities (not tasks in this amendment)

- **P2 `mcp install-missing`:** package-manager/vendor-installer execution, version pinning, explicit argv/consent, timeouts and supply-chain controls. Future separately approved feature.
- **`agentic-sdd opencode launch --free`:** free-model detection, suggested prompt, dry-run and launch. Future separately approved capability; not bundled with MCP configuration.
- **Project-local MCP scope:** precedence/conflict/restoration design first; v1 remains global-only.
- **`--create` for a missing global config:** separate design for exclusive creation, absent-file identity, parent-directory handling and recovery of an unchanged tool-created file.
- **`--replace`/force conflict handling:** separate adoption, credential and restore design; v1 always refuses.
- **CodeGraph project initialization (`codegraph init`):** separate explicit project action with path guards, consent and its own approval.
- **General `mcp uninstall` and `--with-mcp`:** future scope; no implementation or approval here.

---

## Task acceptance evidence (per batch)

Each batch report must include:

1. Checkout SHA and pre/post `git status --short`.
2. Files changed and the exact typed operations added.
3. Commands run, with `Pass / Fail / Blocked / Not run` for each.
4. Fixture identities (`t.TempDir`, config digest) and whether any real home/private Git was touched (must be "no").
5. ACs satisfied and any not covered, with reason.
6. Safety assertions actually tested: zero-write reads, single write slot, stale plan/source, no adopt, no model/network.
7. Remaining risks and the next action requiring explicit human authorization.

## Fence reminders

- Do not select excluded 005/006/007 work or reinterpret their historical records.
- Keep the 10-package supported-core gate; do not add a skip/filter/tolerance to make a suite pass.
- `task -t Taskfile.experimental.yml tui:test`/`tui:pty-test` are explicit selections; a toolchain skip is `not_run`, not a pass.
- No real `~/.config/opencode`, private fleet repo, provider credential, model call or release action.
