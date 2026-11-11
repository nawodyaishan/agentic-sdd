# Feature 014 — Tasks (r1, 2026-11-11)

**Parents:** `spec.md` (rev 2026-11-11 R1–R11) + `plan.md` (r1).
**Approval:** Combined spec/plan/tasks **Approved 2026-11-11**; B1–B5 authorized parallel. Integration `task test/vet/diff-check` green 2026-11-11 (10 packages). No real-home/model/release authorization.
**Execution rule:** One explicitly authorized batch at a time per repo policy; batches below are *designed* for parallel subagent-fleet execution once authorized. Each task states files, acceptance, and verification.

## Batch ledger

| Batch | Scope | Parallel agent (proposed) | State |
|---|---|---|---|
| B1 | Schema/resolver + strict validator | `sdd-implement-1` (free DeepSeek; routine pure logic) | awaiting human review (integration green) |
| B2 | Renderer + permission gate (+ ≤3 adapters conditional) | `sdd-implement-2` (free DeepSeek; routine pure logic) | awaiting human review (integration green; adapters resolved by B6 Option A) |
| B3 | CLI status/plan + identity/lock | `sdd-implement-9r-1` (9Router Gemini Flash; routine CLI) | awaiting human review |
| B4 | Apply/restore + sidecar/drift/adopt | `sdd-implement-2` DeepSeek max handover (ex `sdd-implement-9r-2` quota) | awaiting human review (integration green) |
| B5 | Doctor + docs + full regression/scale | `sdd-test-implement-1` + verification by `sdd-review-verification-2` | awaiting human review |
| B6 | Adapters + all-18 + ledger consistency | `sdd-implement-2` DeepSeek high | Authorized 2026-11-11, in progress |
| B7 | Apply/restore integrity (drift/lock/legacy) | `sdd-implement-2` DeepSeek high | Authorized 2026-11-11, in progress |
| B8 | Plan/CLI hardening (paths/cap/exits/status) | `sdd-implement-9r-1` Gemini Flash | awaiting human review |
| B9 | Scale evidence + FR-15 + docs | `sdd-implement-9r-1` Gemini Flash | awaiting human review |

All implement agents granted CodeGraph + Context7 read access only. No model/network invocation in code under test. All tests isolated-home (`t.TempDir()`).

## B1 — Schema/resolver + strict validator

- [ ] **T01** `internal/fleet/fleet.go` + `parse.go`: types + strict loader (1 MiB `LimitReader`, `utf8.Valid`, `DisallowUnknownFields`, token-walk duplicate detector, ID/model regexes, traversal + forbidden policy-key rejection, `schemaVersion==1` gate).
  - Accept: §8 tests 1, 2, 8 pass; oversize/duplicate/unknown/traversal fixtures fail exit 2 with sanitized diagnostics (no full config echo, no secrets).
  - Verify: `go test ./internal/fleet -run TestFleetParse`.
- [ ] **T02** `internal/fleet/resolve.go` + fixtures: walk `agents/`+`adapters/`+`presets/` allow-list; one-hop `SourceRef` validation; 18-name golden mapping (5 overlaps→`agent:`, 2 reviewed adapters→`agent:`, `vision`→`agent:sdd-visual-inspector`, 4 free-first→`preset:`, 6 absent→preset reuse).
  - Accept: AC-02/AC-05 (mapping + invalid-source diagnostics); every `SourceRef` resolves to walked file.
  - Verify: `go test ./internal/fleet -run TestFleetResolve`.

## B2 — Renderer + permission gate

- [ ] **T03** `internal/fleet/render.go`: start-from-canonical-bytes (frontmatter state machine: second-`---` split, CRLF, missing/duplicate `model:`, `---` in body, `#variant` preserve); only `model:` scalar changes; body+permissions byte check helper.
  - Accept: AC-03 (identical bytes on re-render; canonical body unchanged); §8 test 11 stale-on-permission-change.
  - Verify: `go test ./internal/fleet -run TestFleetRender`.
- [ ] **T04** Normalized permission-set equality + conditional adapters: reuse frontmatter parser pattern; model-only change preserves normalized set (incl. MCP `execute`/`codegraph_*`/`context7_*`/`exa_*` lane rules); any delta fails closed. Add ≤3 adapters only if inequivalence proven (`sdd-spec-9r`, `sdd-plan-9r`, `vision` vs `sdd-visual-inspector`); each non-writing/non-shell deny-by-default. **B6 resolved this:** two read-only adapters under `opencode-pack/adapters/`; `vision` maps to `agent:sdd-visual-inspector`; no third adapter.
  - Accept: AC-17/AC-18 + §8 tests 11, 12, 17; no parallel prompt tree.
  - Verify: `go test ./internal/fleet -run TestFleetPermissions`.

## B3 — CLI status/plan + identity/lock

- [ ] **T05** `cmd/agentic-sdd/fleet*.go` dispatch: `case "fleet"` + `runFleet` (`status|plan|apply|restore|doctor`), per-subcommand flags, `help fleet`, exit codes (2/1/0), absent-key=ignored vs `enabled:false`=remove, parser regression (non-fleet paths byte-identical).
  - Accept: AC-08/AC-10 + FR-06; `--with-opencode` paths untouched.
  - Verify: `go test ./cmd/agentic-sdd -run TestFleetCLI`.
- [ ] **T06** `internal/fleet/plan.go` + lock: canonical plan JSON + SHA-256 `planID`; full pre-apply revalidation; `${backupRoot}/.fleet.lock` (`O_CREATE|O_EXCL`, PID, 30s stale); single manifest path (`<backup>/<stamp>/fleet-manifest.json` + live pointer); `plan --output` writes only with explicit flag.
  - Accept: AC-09/AC-10 + §8 tests 5, 6, 16; plan ID is token, not authorization.
  - Verify: `go test ./internal/fleet -run TestFleetPlan; go test ./cmd/agentic-sdd -run TestFleetLock`.

## B4 — Apply/restore + sidecar/drift/adopt

- [ ] **T07** `internal/fleet/manifest.go` + apply/remove reuse: sidecar v4 (`desiredDigest` via staged-bytes `fileDigest`, `legacy/unknown` for format-3 history); staged apply + `remove:true` omit path with backup/rollback reuse; three-state drift table; `--adopt` explicit; idempotent no-op; never touch `AGENTS.md`/unmanaged files; symlink/traversal refusal.
  - Accept: AC-11–AC-16 + §8 tests 3, 4, 7, 8, 9, 10, 13, 19.
  - Verify: `go test ./internal/fleet -run TestFleetApply; go test ./cmd/agentic-sdd -run TestFleetApplyCLI`.
- [ ] **T08** Restore: `fleet restore --backup` validates current hashes, restores managed only, never silent user-data loss, never `git revert`=live-restore; failed rollback leaves recoverable state.
  - Accept: AC-12/AC-14/AC-15.
  - Verify: `go test ./internal/fleet -run TestFleetRestore`.

## B5 — Doctor + docs + full regression/scale

- [ ] **T09** `internal/fleet/doctor.go` + `fleet doctor`: local-only inspection; labels `configured|catalog-visible|authenticated|unknown|unavailable`; `openai/*`=route-only; no network/model calls; offline paths label `not checked`.
  - Accept: AC-19–AC-22 + §8 test 14.
  - Verify: `go test ./internal/fleet -run TestFleetDoctor`.
- [ ] **T10** Docs + gates + scale: worked guide (private-Git setup, 1-agent change, preview/apply/drift/restore, multi-machine reproduce, provider-vs-entitlement notes); register `internal/fleet` in `Taskfile.yml`, `harness/checks.json`, `tests/supported-core-go.sh` (+ experimental mirror if present); 18-instance + 100-instance fixtures with NFR-02 timings; 009 regression (§8 test 20); AC-01–AC-29 report.
  - Accept: AC-01/AC-04/AC-06/AC-07/AC-24–AC-29; `task test`, `task vet`, `git diff --check` green; `go.mod` dep-free.
  - Verify: `task test; task vet; git diff --check` (record commit/commands/env).

## Parallel fleet execution (post-approval)

1. Authorize explicitly per batch (one batch = one human authorization; do not auto-chain).
2. Launch B1+B2 in parallel (independent pure logic; shared frozen grammar from plan §2.1); B3 may start against B1/B2 stubs in parallel; B4 after B3 schema freeze; B5 last.
3. Each agent: implement only assigned task IDs, run focused checks named in task, quote commands/results, return changed paths/deviations/uncertainty; parent owns ledger + human handoff.
4. Verification agent (`sdd-review-verification-2`, Beam Free, supplementary) reviews B1–B5 evidence against AC list; consequential verdict still requires Opus review.

## Next action (requires explicit human authorization)

B1–B5 awaiting human review; B6–B9 authorized parallel fix fleet 2026-11-11 (DeepSeek high + Gemini Flash). No real-home/model/release.

## B6 — Adapters + all-18 + ledger consistency (H1/M10)

- [ ] **T11** Option A adapters: `opencode-pack/adapters/` + `resolve.go` dual-walk + 009 14-profile exclusion; golden all-18 resolve+render; `vision` single mapping; fix `plan.md:6`, DRAFT lines, T05/T06 states, handover, trusted-base adopt/defer.
  - **Option A decision:** `sdd-spec-9r.md` and `sdd-plan-9r.md` live under `opencode-pack/adapters/` as reviewed read-only, non-writing/non-shell sources; `resolve.go` walks `agents/` + `adapters/` + `presets/free-first/`; `vision` maps to canonical `agent:sdd-visual-inspector` (equivalent image-transcription contract, so no `vision.md`); `--with-opencode` still installs exactly the 14 `agents/` profiles and a skillsync test pins the adapter exclusion.
  - **Trusted-base:** the 10-package `harness/checks.json` candidate is **not adopted**; running `task verify-trusted` against a live reviewed base is a separately authorized live operation and is **deferred** (not run in B6).
  - Verify: `go test ./internal/fleet -run 'TestFleetResolve|TestFleetRender'; go test ./internal/skillsync -run WithOpenCode`.

## B7 — Apply/restore integrity (M1/M2/M5/L4 + M3)

- [ ] **T12** Re-derive drift at apply; `restored` provenance; PID lock + ownership; backup-digest verify + live rebuild; legacy tolerance or segregated root + non-`sdd-*` test.
  - Verify: `go test ./internal/fleet -run TestFleetApply|TestFleetRestore|TestFleetPlan; go test ./cmd/agentic-sdd -run TestFleetLock`.

## B8 — Plan/CLI hardening (M6/M7/M8/M4 + M9/L1-L3/L5-L8)

- [x] **T13** Drop paths from planID (or 0600 + docs); LimitReader boundary; exits 2/1 + legend; status drift/resolve/labels/PermNotes/prior-model warn; reserved-name allowlist; CRLF/block-scalar/symlink/0700/doctor-cap fixes.
  - **Plan identity & security:** Dropped `ConfigPath` and `RepoPath` from `canonicalPayload` in `internal/fleet/plan.go`; plan ID is now invariant to config/repo paths across different checkout environments while maintaining exact canonical ordering. `SavePlan` writes 0600 file permissions under 0700 directories.
  - **LimitReader & non-regular rejection:** `LoadConfigFile` and `LoadPlan` reject non-regular files (directories, symlinks) and enforce 1 MiB / 4 MiB limits via `io.LimitReader`, returning errors wrapping `ErrInvalidConfig` and `ErrInvalidPlanStructure`.
  - **Exit codes & CLI legend:** `printFleetUsage` displays standardized Exit Codes legend (0 success, 1 runtime/stale/conflict/lock, 2 usage/flag/invalid-config/malformed-plan). CLI maps `ErrInvalidConfig`, flag errors, plan structure errors, and unknown source to exit 2; stale plan, lock held, and plan conflicts map to exit 1. Comprehensive table test `TestFleetExitCodesTable` passes.
  - **Status enhancements:** `fleet status` resolves sources via pack resolver (`--repo` supported, unknown source exits 2), detects target drift (`installed (managed, drifted)`), reports unsafe targets (`installed (unsafe target; refused)`), prints offline/selected-primary labels (`Selected primary: not selected (offline default)`, `Runtime: not checked (offline)`), warns on route changes from prior models, and warns on reserved names (`spec`, `plan`, `vision`).
  - **Hardening fixes:** Fixed CRLF insertion in `render.go` where `splitFrontmatter` failed to detect CRLF when model was missing; implemented block-scalar fail-closed in `replaceModel` and `parseScalar`; strictly enforced all fields in `Plan.Revalidate` (require-all); refused symlink state directories in `ReadLiveManifest`, `WriteLiveManifest`, and `AcquireLock`; enforced absolute home directory throughout; capped doctor listing stdout to `MaxConfigBytes` (1 MiB); documented empty-model and null-enabled semantics in `fleet.go`.
  - Verify: `go test ./internal/fleet -run TestFleetPlan|TestFleetParse; go test ./cmd/agentic-sdd -run TestFleetCLI|TestFleetDoctor`.

## B9 — Scale evidence + FR-15 + docs (H2/H3/M10/L9)

- [x] **T14** Real BuildPlan 18+100 scale timings; FR-15 import or deferral note; real-pack render 14; §8 gaps; consolidated AC report + handover.
  - **Scale measurements:** `TestFleetBuildPlanScaleRecordsTimings` records median-of-5, worst duration, memory allocation delta and runner metadata for 18-instance golden fleet (median 852µs, worst 933µs, ~776KB) and 100-instance synthetic fleet (median 4.08ms, worst 4.58ms, ~4.2MB), proving NFR-02 design target (< 2s) is comfortably met.
  - **FR-15 Deferral:** recorded human-approved deferral in `spec.md` §9 and §13; adoption of existing machine profiles is handled via explicit `fleet plan --adopt`.
  - **Real-pack render:** `TestFleetRender/real_pack_render_all_14` verifies round-trip model overrides, body byte identity, and permission set preservation for all 14 canonical agents.
  - **§8 negative matrix & lifecycle:** `TestFleetSidecarApplyingAndFailedState`, `TestFleetNegativeMatrixRow3DuplicateInstalledNameOrAliasToAlias`, `TestFleetNegativeMatrixRow5OpenCodeOverlapVsFleet`, `TestFleetNegativeMatrixRow12SpecPlanAdapterNonwritingPolicy`, `TestFleetNegativeMatrixRow18OrchestratorInstallDoesNotChangeDefault`, `TestFleetNegativeMatrixRow19GitRevertRequiresExplicitApply`, `TestFleetRestoreIntegrity/managed_update_restores_and_replans_without_adoption`, and `TestFleetRestore/fleet-backup_listing_and_legacy_skillsync_coexistence` pass.
  - **Guide placement:** retained under `specs/014-agent-fleet-model-overrides/fleet-worked-guide.md` during development; moves to `docs/guides/` at release readiness.
  - **Consolidated report & handover:** AC-01–AC-29 status report delivered; handover recorded in `docs/handovers/2026-11-11_014.md`.
  - Verify: `go test ./internal/fleet -run 'TestFleet.*Scale'; task test; task vet; git diff --check`.
