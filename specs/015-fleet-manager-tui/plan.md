# Feature 015 — Plan (draft r1, 2026-11-11)

**Parent spec:** [`spec.md`](spec.md) (draft r1, 2026-11-11).
**Status:** APPROVED r1 (2026-11-12) — combined spec/plan/tasks approval recorded in [spec §Approval](spec.md#approval). Wave 1 (W1-A/W1-B/W1-C) authorized; B4 source-write separately gated. **No real-home, private-Git, model/network or release operation is authorized.**
**Target:** Root Go 1.23 stdlib for shared operations; nested `experiments/tui` Go 1.26 / pinned Charm v2 for UI. No real-home, private-Git, model, network or release operation.
**Baseline:** checkout `d5eaa7a`; Feature 013 accepted/published (`v0.8.0`); Feature 014 B1–B9 implemented, **awaiting human review + Opus verdict**.
**Inputs:** the two advisory documents in this directory and the verification in spec §2.

## 1. Repository-first verification (must precede implementation)

Re-confirm in the implementation checkout; do not trust this plan's line numbers:

1. `experiments/tui/app.go`: four routes, `App.Update`, focus, `routeScroll`, modal/help hooks, `NewApp`, `TUIConfig` assembly.
2. `experiments/tui/app_screens.go`: `installSection` (Hub/Install/Backups), two-entry `installHubIndex` modulo, `installHubLines`, `handleInstallRouteKey`, `openInstallScreen`, `closeInstallHub`, `childContentWidth/BodyHeight`.
3. `experiments/tui/ops_host.go`: `WriteKind`, `WriteRequest`, `BeginWrite`, single-slot refusal, `WriteStartedMsg`/progress/result, flush/exit policy, `OpsHostOptions` seams.
4. `experiments/tui/{viewport,modal,theme,glyphs,presentation}.go` and `*_test.go`: 40×8 geometry, modal kinds/parts, palette, ASCII/plain modes, responsive fixtures.
5. `experiments/tui/cmd/tui-companion` and `cmd/internal/tuirun`: flag parsing, `TUIConfig`, startup path; `cmd/agentic-sdd/tui.go`: launcher request/arg forwarding and handshake.
6. `internal/appops/{ops,requests}.go`: immutable request pattern, preview-token revalidation, `OperationID` staleness, `ApplyInstall`/`ApplyRestore`.
7. `internal/fleet`: `LoadConfigFile`, `ParseConfig`, `Resolve`, `RenderResolved`, `BuildPlan`, `Plan` fields, `ApplyPlan` revalidation, `ListFleetBackups`, `RestoreBackup` validation, `RunDoctor`, manifest/lock, `TargetRootForHome`.
8. `cmd/agentic-sdd/fleet*.go`: dispatch, inline status projection, exit codes, plan file round-trip.
9. `Taskfile.yml`, `Taskfile.experimental.yml` (`tui:test`, `tui:pty-test`, `tui:pair`), `harness/checks.json`, `AGENTS.md` 008 fence and the 10-package core selection.
10. Do **not** inspect a real `~/.config/opencode` or private fleet repository; use sanitized `t.TempDir()` fixtures.

## 2. Design freeze proposal

### 2.1 Package and file map

| File | Change | Owner |
|---|---|---|
| `internal/fleet/status.go` | **New** read-only `Status` projection (config, instances, desired/current/last-applied, provenance, drift, counts). | root |
| `internal/fleet/diff.go` | **New** pure line diff + `FilePreview` (old/new bytes, digests, op, conflict) used by source and deployment review. | root |
| `internal/fleet/restore_preview.go` | **New** read-only restore plan; shares validation with `RestoreBackup`. | root |
| `internal/fleet/save.go` | **New** atomic personal-source save with mandatory CAS for an existing file, backup outside the checkout; strict parse reuses `parse.go`. | root |
| `internal/fleet/{fleet,plan,render,resolve,apply,restore,doctor,manifest}.go` | No semantic rewrite. Small exported helpers only if a projection needs them. | root |
| `internal/appops/fleet.go` | **New** immutable fleet requests/results, preview tokens, revalidation. `FleetSourcePreview` is a pure read; no adopt parameter; no stateful draft. | root |
| `cmd/agentic-sdd/fleet_config.go` | **New** `fleet config save` CLI parity; refactor `fleet status` onto `fleet.BuildStatus` with byte-identical text. | root |
| `experiments/tui/fleet.go` | **New** `FleetModel`: four views, draft, selection/filter/scroll, generations. | TUI |
| `experiments/tui/fleet_screens.go` | **New** view rendering: Overview, Profiles, Review Changes, Recovery, overlays. | TUI |
| `experiments/tui/fleet_adapter.go` | **New** typed adapter around `appops`; no filesystem. | TUI |
| `experiments/tui/fleet_test.go`, `fleet_*_test.go` | **New** headless model, responsive, async, negative tests. | TUI |
| `experiments/tui/app.go`, `app_screens.go`, `ops_host.go`, `config.go` | **Edit** third hub child, Fleet key, WriteKinds, inline fleet confirmations (no modal.go edit), `FleetConfig`/`FleetLive` fields. | TUI |
| `experiments/tui/cmd/internal/tuirun/run.go`, `cmd/tui-companion/main.go`, `cmd/agentic-sdd/tui.go` | **Edit** explicit `--fleet-config` forwarding and reviewed live-capability injection (no default path). | TUI/CLI |
| `specs/015-fleet-manager-tui/*` | Spec/plan/tasks and evidence. | docs |

### 2.2 Typed shared API (proposed; exact signatures reconciled at implementation)

```go
// internal/fleet/status.go
type InstalledState string // absent | managed-current | managed-drifted | pending-apply | unmanaged | legacy-unknown | unsafe-target | disabled
type InstanceStatus struct {
    InstanceID, SourceRef, SourceDigest string
    ConfiguredModel, RenderedModel      string
    Enabled                            bool
    DesiredDigest, CurrentDigest       string
    LastAppliedDigest                  string
    Installed                          InstalledState
    Provenance                         string
    Conflict                           string
}
type Status struct {
    ConfigPath, ConfigDigest, TargetRoot string
    Instances                            []InstanceStatus
    Counts                               map[string]int
}
func BuildStatus(configPath, repo, home string) (*Status, error) // read-only

// internal/fleet/diff.go
type FilePreview struct {
    InstanceID, Path, Op, Reason string
    OldBytes, NewBytes           []byte
    OldDigest, NewDigest         string
    Removed, Conflict            bool
}
func PreviewSource(path string, newBytes []byte) (FilePreview, error)       // read-only
func PlanFiles(p *Plan) ([]FilePreview, error)                             // read-only
func LineDiff(old, new []byte) []DiffLine                                  // pure, bounded

// internal/fleet/restore_preview.go
type RestorePreview struct { BackupID, BackupPath string; Files []FilePreview; Counts map[string]int }
func PreviewRestoreBackup(home, backupID string) (*RestorePreview, error)  // read-only

// internal/fleet/save.go
type SaveOptions struct {
    Home         string // resolved home for the private backup root and log
    ExpectDigest string // required, non-empty; must match sha256(current)
}
type SaveResult struct { Path, BackupPath, Digest string; NoOp bool }
var ErrStaleSource = errors.New("source changed since it was read")
func EncodeConfig(c *Config) ([]byte, error)                               // deterministic
func SaveConfig(path string, newBytes []byte, opts SaveOptions) (SaveResult, error)
```

`SaveConfig` accepts only an existing regular file in v1. An empty `ExpectDigest` is a usage/refusal error; a mismatch returns `ErrStaleSource`; neither writes. It always makes the pre-save backup for a changed file under `<home>/.agentic-sdd/fleet-source-backups/` (outside the selected checkout and outside the skillsync backup namespace), and never creates a new `fleet.json`. Creation is a separate future operation.

`internal/appops/fleet.go` mirrors the 013 pattern: unexported immutable fields, constructors that validate and absolutize, accessors, per-operation preview tokens, and `ErrNeedsPreview` on mismatch. Read requests (`FleetStatus`, `FleetPreview`, `FleetRestorePreview`, `FleetDoctor`, `FleetSourcePreview`) perform no writes. Write requests (`FleetSourceSave`, `FleetApply`, `FleetRestore`) revalidate in the same call that writes. `FleetPreview` has **no adopt field** and calls `BuildPlan(..., false)`.

`appops` remains stateless: `FleetSourcePreview(configPath, repo, home, draftBytes)` returns validated normalized bytes, `FilePreview` and the base digest, retaining no draft between calls. Draft state lives only in `FleetModel`. `FleetSourceSaveRequest` carries the resolved home through to `SaveOptions.Home`; token checks do not replace the mandatory digest CAS or mask `ErrStaleSource` for a source mismatch.

### 2.3 CLI grammar (only new mutation)

```text
agentic-sdd fleet config save --config PATH --from PATH --expect-digest SHA256 [--dry-run] [--home PATH]
```

- `--from` is a local file containing the complete new strict JSON; `--dry-run` validates and prints the diff/NOOP without writing.
- `--expect-digest` is required; it must match the current file. An empty value is a usage error, never an implicit overwrite.
- The selected file must already exist and be a regular, non-symlink file. Creation is not part of v1.
- `--home` (default `os.UserHomeDir`) locates the private source-backup directory and is not a write target.
- Exit codes follow 014: `2` usage/validation, `1` runtime/stale/conflict, `0` ok.
- Existing `status|plan|apply|restore|doctor` grammar and text output remain compatible. `fleet status` may be refactored internally but its output is byte-identical in existing tests.
- The TUI never invokes this parser; both paths call `appops`.

### 2.4 TUI seams

- `installSectionFleet` added after Backups; hub entries become three and all cursor wrap/bounds use `len(entries)` (including backward wrap), not a hardcoded 2. Enter dispatches the selected entry across all three; key `f` opens Fleet. Update all hub tests in B1; breadcrumb `Install & Manage / Fleet Manager`.
- `FleetModel` owns `view` (Overview/Profiles/Review/Recovery), `draft map[string]Instance`, `baseDigest`, `sourcePath`, `generation`, `selected`, `filter`, `scroll`, `plan`, `planToken`, `previews`, `operation`. It never owns a writer. Root `Update` routes keys/messages and `WindowSizeMsg` to it; it is lazily loaded only when Fleet is opened.
- `OpsHost` gains `WriteFleetApply`, `WriteFleetRestore`, `WriteSourceSave` and a `Fleet` request field. `WriteRequest.validate()` rejects a mismatched kind/payload. For **each** of these kinds, `WriteRequest.home()` returns the request's resolved home to `openFileWriteLog` (required even when the config lives outside home); `source()` reports the fleet config path or a stable fleet source label, never `embedded`. `Command` headers identify `fleet config save`, `fleet apply --plan <planID>` (identity label for the in-memory plan, not a temporary file), and `fleet restore <backupID>`. Transcript assertions cover all three. `startInventoryRefresh()` handles fleet apply/restore and refreshes catalog plus Fleet status; source save refreshes Fleet status without claiming installed changes. Generation-bound refresh tests cover every fleet write kind.
- Fleet apply/restore confirmations are inline subview confirmations with the same default-No, focus-trap and tiny-geometry guards; `modal.go` remains the compositor for global install/restore/help (do not claim modal reuse that does not exist).
- `--fleet-config` is now wired through launcher (`cmd/agentic-sdd/tui.go`), companion (`tuirun`/`tui-companion`), and `TUIConfig.FleetConfig` as an explicit optional path; empty means the Overview first-use choice. No scanning, no default file. `TUIConfig.FleetLive` is injected from reviewed build metadata and defaults false; it is not a user preference.
- `generation` increments on source change, scope change, manual refresh and successful write. Every async result carries `{path, digest, scope, generation}` and is dropped if stale.
- Draft edits are applied to a copy of the last loaded `Config`; save serializes via `fleet.EncodeConfig`, then calls the save request. The TUI never serializes free-form JSON. Top-level metadata is preserved; arbitrary formatting/key order and a full raw-text edit remain B4 concerns.

### 2.5 Integrity and security rules

1. All paths are absolutized and checked by `internal/fleet`; screens never join or clean paths for a write.
2. Source save: existing regular file only, no symlink, 1 MiB cap, strict parse, **mandatory** non-empty digest CAS, `0600` temp in the same directory, `fsync` where supported, atomic rename, pre-save backup under `<home>/.agentic-sdd/fleet-source-backups/`, no Git, no new-file creation.
3. Apply: fresh `BuildPlan(..., false)` from the saved source; `appops` token covers plan ID + config digest + source revision + target hashes; `ApplyPlan` revalidates under the engine lock. A plan is never reused after a save, source change, scope change, external edit or apply result. The shared preview request has no adopt field; adopt is hardcoded false. Fleet source-save/apply/restore and existing install/restore use one `OpsHost` slot and explicitly refuse each other in both directions (AC-27, including r2 AC-46).
4. Restore: preview and apply share one validator; apply refuses if the target changed since preview, or the backup is incomplete/legacy.
5. No `--adopt` exposure in the TUI. No force/fix-all/merge.
6. Doctor is opt-in: the user must select an explicit local `--opencode-bin` path; default is offline `not checked`. No PATH search, no network.
7. Redaction: no tokens, credentials, full private JSON or unfiltered home paths in UI/logs/fixtures. Agent file contents are shown in the diff because they are the deployment target; private config diff is shown only for the selected file.
8. Root tests use `GOPROXY=off`, `GOWORK=off`, `GOTOOLCHAIN=local`; TUI tests use the nested module and `Taskfile.experimental.yml`. No test writes a real home.
9. Live capability: a single reviewed build value (for example `buildinfo.FleetCapability=preview|live`, default `preview`) is injected at build time and read once by the companion. `TUIConfig.FleetLive` exposes it to the model. There is no env var, config key, hidden key or CLI flag that flips it at runtime. Tests inject `FleetLive=true`; shipped preview builds hide/disable apply and restore and label the screen `Preview only`.
10. Log/refresh parity: all three fleet writes carry a resolved home for `openFileWriteLog` and the exact `Command`/`source()` identities in §2.4. Apply/restore refresh catalog and Fleet status; source-save refreshes Fleet status only, with installed state unchanged. `startInventoryRefresh()` must not silently return nil for fleet apply/restore kinds.

### 2.6 Toolchain and gates

- Root `go.mod` stays `go 1.23` with zero `require`. No root import of `experiments/tui`.
- Nested TUI stays `go 1.26` and pinned `charm.land/*` v2.
- New root packages: none. `internal/fleet`/`internal/appops`/`cmd/agentic-sdd` are already in the 10-package core gate; no gate change expected. If a package is added, update both `Taskfile.yml` and `harness/checks.json` and respect the 008 fence.
- The live-capability field (for example in `internal/buildinfo`) is code in this feature; setting it to `live` in `.goreleaser.yml` or the release workflow is a **release operation**, not part of 015 implementation, and requires 014 acceptance plus separate authorization. No batch may flip the released default.
- Nested verification uses `task -t Taskfile.experimental.yml tui:test` and, for the final batch, `tui:pty-test`/`tui:pair`; a skip is reported as `not_run`, never `pass`.

## 3. Batch decomposition and approved waves

The B1–B6 table groups the work by capability. **Approved execution is organized into the parallel waves in [`tasks.md`](tasks.md#parallel-execution-waves);** the groups are not a serial gate. Wave 1 (W1-A root engine, W1-B TUI shell/read-only, W1-C OpsHost plumbing) is authorized. B4 remains outside the waves until its source-write surface is explicitly authorized.

| Batch | Scope | Depends on | Proposed lane |
|---|---|---|---|
| **B1 — Read-only foundation** | `fleet.BuildStatus`; CLI status refactor with byte-identical output; `appops.FleetStatus`; third hub child with `len(entries)` navigation; Fleet Overview + Profiles list/detail read-only; search/filter; responsive shell updates; zero-write tests. | Combined approval; 013 accepted. | Gemini Flash / DeepSeek high |
| **B2 — Staging and review** | `fleet.EncodeConfig`/`PreviewSource`/`LineDiff`; `appops.FleetSourcePreview` (not a stateful draft); model picker; enable/disable consequences; dirty protection; source diff; impact-first review; async generations. | B1 | Gemini Flash / DeepSeek high |
| **B3 — Deployment** | `fleet.PlanFiles`; `appops.FleetPreview` (plan + files + token, adopt=false); Review Changes deployment view; full scrollable diff; default-No apply through `WriteFleetApply`; real result/backup; stale/double-Enter/resize tests; post-apply catalog/Fleet refresh. Apply is fixture/preview-only while the live capability is disabled. | B2 | DeepSeek high / Gemini Flash |
| **B4 — Source save** | `fleet.SaveConfig` (existing file, mandatory CAS, private backup); `appops.FleetSourceSave`; CLI `fleet config save`; Save confirmation + checkpoint; CAS/atomic/backup/read-only-mode/CLI-parity tests; no installed-file change; Fleet status refresh after save. **Separate approval of the new write surface.** | B2 | DeepSeek max / Gemini Flash |
| **B5 — Recovery** | `fleet.PreviewRestoreBackup`; `appops` restore preview/apply; `WriteFleetRestore`; Recovery view; drift/legacy/conflict guidance; opt-in doctor; lock/corrupt/legacy tests; post-restore refresh. | B1, B3 | Gemini Flash / DeepSeek high |
| **B6 — Robustness and docs** | 100-instance/100-op/1,000-line fixtures; 110/80/60 and 8/10/16 PTY tests; ASCII/plain/no-color; paired artifact; owner walkthrough; Help/Quick Start; completion packet. | B1–B5 | verification lane + owner |

**Execution:** Wave 1 runs W1-A, W1-B and W1-C in parallel on disjoint files. Wave 2 fans out appops/CLI, TUI staging/review and TUI write screens. Wave 3 runs verification/docs. Each wave ends `awaiting human review` with evidence; the next wave starts only after integration. B4 is not part of Waves 1–3 until explicitly authorized.

## 4. Verification matrix

| Area | Commands / evidence | Pass condition |
|---|---|---|
| Root engine | `go test ./internal/fleet ./internal/appops ./cmd/agentic-sdd` with the sanitized env | status/diff/save/restore-preview tests pass; 014 regressions unchanged |
| Core gate | `task test`, `task vet`, `git diff --check` | 10-package selection green; no new `require` |
| TUI model | `task -t Taskfile.experimental.yml tui:test` | all nested tests pass; new fleet tests included |
| PTY | `task -t Taskfile.experimental.yml tui:pty-test` | launch/resize/confirm/quit pass, or `blocked` with reason |
| Paired artifact | `task -t Taskfile.experimental.yml tui:pair` + isolated invocation | paired candidate uses compatible contracts |
| Isolation | `grep`/test assertions for `t.TempDir`, `GOPROXY=off`, no real-home paths | no test touches real config |
| Truthfulness | fixture apply/no-op/conflict/rollback/restore results | UI shows the typed outcome, never optimistic success |
| Accessibility | headless + PTY at breakpoints, plain/ASCII/no-color | Back/Cancel/warning reachable; no unseen Yes |
| Evidence | per-batch report: SHA, files, commands, Pass/Fail/Blocked/Not run | no unrun check claimed as pass |

## 5. Decisions after architecture review

The 2026-11-11 architecture-compatibility review (Gemini 3.8 Flash High, provisional per the fallback rule) returned **Approve with conditions**: H1–H2 blocking, M1–M4 pre-implementation, L1–L3 low. All are folded into this plan and the spec. The eight design questions were answered as follows; the owner may still reject any of them at combined approval.

| # | Decision | Rationale |
|---|---|---|
| 1 | **Typed status lives in `internal/fleet/status.go`**, with `cmd/agentic-sdd/fleet.go` refactored onto it and byte-identical output tests. | Fleet owns digests, manifest and ownership; `appops` stays a thin validated seam. |
| 2 | **Shared `FilePreview` + bounded `LineDiff` in `internal/fleet/diff.go`.** | Desired bytes come from `Resolve` + `RenderResolved`; the TUI never renders a second way; the CLI can reuse it. |
| 3 | **Native `SaveConfig` + `fleet config save` proposed, B4 separately gated.** External-editor fallback stays documented. | A staging UI that cannot persist is misleading; the write surface still requires explicit approval. |
| 4 | **Add optional `--fleet-config PATH`** to the launcher and companion. No default, no scan, no handshake change. | Testability and CLI parity without new discovery policy. |
| 5 | **Apply the in-memory `*Plan` through `appops`**, not a temporary plan file. | `ApplyPlan` already revalidates plan ID, config digest, source revision, target hashes and the lock. |
| 6 | **Separate read-only `PreviewRestoreBackup`**, not a `dryRun` flag on `RestoreBackup`. | Keeps the writer's single control flow and lock boundary untouched; shares the validator. |
| 7 | **Hub key `f`**; no collision with `i`, `b`, `j/k`, Enter or Esc. | Verified in `handleInstallRouteKey`. |
| 8 | **Gate only live apply/restore**, not read/stage/save. Default build is `Preview only`; the live capability is a reviewed build value. | Read/stage/save against an existing local file is safe and useful; deployment is the consequential step that depends on 014 acceptance. |

**Residual owner decision:** whether to authorize B4's new source-write surface at combined approval, or defer it and ship the external-editor fallback. The plan supports either; B4 is not implicitly approved by approving B1–B3/B5.

### 5.1 Patch record

Status records design coverage only, not implementation, acceptance or authorization. "Applied (docs)" means the chosen option is written into spec/plan/tasks; "code pending" means the implementation and verification task has not run. The architecture verdict remains provisional pending Opus review.

| Finding | Chosen option | Spec anchor | Plan anchor | Tasks | Status |
|---|---|---|---|---|---|
| **H1** apply/restore gated | One release-metadata build capability, default `preview`; no env/config/CLI/key runtime toggle; hide/disable both with `Preview only`; tests inject live=true. Release enablement is separate from 015. | FR-10, FR-11, FR-20; AC-36 | §2.5 rule 9; §2.6 | B3/T13; B5/T19 | Applied (docs); code pending |
| **H2** no adoption seam | `FleetPreview` has no adopt field; `BuildPlan(..., false)`; no Adopt/Force/Fix-all; adoption remains explicit 014 CLI only. | FR-12; AC-26 | §2.2; §2.5 rules 3, 5 | B3/T11; B5/T19 | Retained (docs); code pending |
| **M1** mandatory CAS | Existing regular file only; non-empty `ExpectDigest`; empty = usage/refusal, mismatch = `ErrStaleSource`, no write. Mandatory private backup; no creation. | FR-06, FR-08, FR-15; AC-16–AC-18 | §2.2; §2.3; §2.5 rule 2 | B4/T14–T16 | Applied (docs); code pending |
| **M2** stateless appops | Pure `FleetSourcePreview`: path + draft bytes → validated bytes + `FilePreview` + base digest; draft only in `FleetModel`. Historical `FleetDraft` operation renamed, not an API. | FR-07; AC-14, AC-27 | §2.2; §2.4 | B2/T07–T09 | Applied (docs); code pending |
| **M3** resolved home | `SaveOptions.Home` and appops save request carry resolved home; `home()` handles all three fleet write kinds for `openFileWriteLog`. | FR-21; AC-23 | §2.2; §2.4; §2.5 rule 10 | B3/T12; B4/T14, T15, T17; B5/T19 | Applied (docs); code pending |
| **M4** transcript identity | Extend `source()`/`Command`; assert non-`embedded` source and `fleet config save`, `fleet apply --plan <planID>`, `fleet restore <backupID>` headers. | FR-21; AC-23 | §2.4; §2.5 rule 10 | B3/T12; B4/T17; B5/T19 | Applied (docs); code pending |
| **L1** post-write refresh | `startInventoryRefresh()` handles fleet apply/restore, refreshing catalog/Fleet; source-save refreshes Fleet status; generation-bound assertions. | FR-21; AC-28 | §2.4; §2.5 rule 10 | B3/T12; B4/T17; B5/T19 | Applied (docs); code pending |
| **L2** one-slot traceability | r2 AC-46 → AC-27; source-save/apply/restore and existing install/restore mutually refuse through one `OpsHost` slot. | FR-16; AC-27; §8.1 | §2.5 rule 3 | B3/T12; B4/T17; B5/T19 | Retained (docs); task coverage explicit |
| **L3** third hub entry | `len(entries)` cursor wrap/bounds; Enter for all three; key `f`; update all hub tests in B1. | FR-01; AC-02 | §2.4; §6 | B1/T04 | Applied (docs); code pending |

**Independent caveats added by the parent review** (not in the subagent report): backup must live outside the checkout and outside the skillsync backup namespace; new-file creation is deferred; Windows/non-atomic replacement must refuse rather than overwrite; `fleet config save` must be added to help/usage and parser tests; status projection must stay read-only and must not call `ApplyPlan`; diff size is bounded and paged.

## 6. Risks and controls

| Risk | Control |
|---|---|
| Hub 2→3 breaks cursor/tests | Use `len(entries)` modulo/bounds; update keys and all hub tests in B1; responsive fixtures. |
| Status projection drifts from CLI text | Refactor CLI to the projection and keep byte-comparison tests. |
| Diff engine becomes a second renderer | `PlanFiles` reuses `Resolve` + `RenderResolved`; no screen-side rendering. |
| Source save becomes a second config manager | One operation, one existing file, strict parse, mandatory CAS, `0600`, private backup, CLI parity, no Git, no creation. |
| Backup pollutes the private checkout or backup namespace | Write backups to `<home>/.agentic-sdd/fleet-source-backups/`, never beside the config or under the skillsync backup root. |
| Apply bypasses OpsHost | `WriteFleetApply` is the only path; `BeginWrite` single-slot refusal tested; no direct `fleet.ApplyPlan` in TUI. |
| Restore preview diverges from restore | Shared manifest validator; preview/apply parity tests. |
| 014 gate bypass | FR-20/AC-36; reviewed build capability default `preview`; apply/restore hidden/disabled; no env/key override. |
| Deferred adoption sneaks in | No adopt field in `appops`; `BuildPlan(..., false)`; AC-26. |
| Fleet log fails or misattributes | `WriteRequest.home()`/`source()`/`Command` extended; transcript assertions in B3/B4/B5. |
| Stale catalog after fleet write | `startInventoryRefresh()` handles fleet kinds; source save refreshes Fleet status. |
| Real-home/model calls in tests | Isolated `t.TempDir`, `GOPROXY=off`, fake doctor probe, no network. |
| Long diffs freeze the UI | Bounded line diff, viewport paging, 100-op/1,000-line fixtures, p95 evidence. |
| New CLI grammar conflicts with 014 | Only `fleet config save` added; existing grammar/output byte-identical; help/parser tests; separate approval in B4. |
| Windows/non-atomic replace | Refuse or document a safe replace strategy; no best-effort overwrite. |

## 7. Release disposition

B1–B3 and B5 can be developed and verified against fixtures while 014 awaits its verdict, but a supported build must not enable production apply/restore until 014's human review/Opus/trusted-base/runtime/release gates are satisfied. B4 introduces a new write surface and is not covered by 014's approval. The final report states **preview-only** or **no-go** where appropriate; no tag, cask, tap or real-home operation is authorized by this plan.
