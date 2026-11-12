# Feature 015 — Plan (draft r1, 2026-11-11)

**Parent spec:** [`spec.md`](spec.md) (draft r1, 2026-11-11; MCP amendment r1, 2026-11-12, **pending approval**).
**Status:** APPROVED r1 (2026-11-12) — combined spec/plan/tasks approval recorded in [spec §Approval](spec.md#approval). Wave 1 (W1-A/W1-B/W1-C) authorized; B4 source-write separately gated. **MCP-A (T26–T30) is implemented and awaiting human review; MCP-B–MCP-D are authorized for parallel implementation.** §8 is the design reference for B–D; real-home apply, network/OAuth, package install and release remain unauthorized. **No real-home, private-Git, model/network, package-install or release operation is authorized.**
**Target:** Root Go 1.23 stdlib for shared operations; nested `experiments/tui` Go 1.26 / pinned Charm v2 for UI. No real-home, private-Git, model, network, package-install or release operation.
**Baseline:** MCP amendment reconciled at `4c2b9bd` (clean tree; parent `74f568b` Wave 3); Feature 013 accepted/published (`v0.8.0`); Feature 014 implemented scope **awaiting human/Opus acceptance**; batch ledger unchanged; B4 not authorized; Fleet live capability default `preview`.
**Inputs:** the two advisory documents in this directory, `docs/mcp-default-integration-study.md`, `docs/codegraph.md`, and the verification in spec §2/§16.

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
11. **MCP amendment only:** re-confirm `cmd/agentic-sdd/main.go` dispatch/`isHelpTopic`/usage, `experiments/tui/app_screens.go` hub entries, `experiments/tui/ops_host.go` write kinds/slot, `internal/skillsync` backup/atomic/digest helpers, and the current OpenCode V2 `mcp.servers` schema before editing. Do not inspect a real user config; use golden fixtures.

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

B1–B3 and B5 can be developed and verified against fixtures while 014 awaits its verdict, but a supported build must not enable production apply/restore until 014's human review/Opus/trusted-base/runtime/release gates are satisfied. B4 introduces a new write surface and is not covered by 014's approval. MCP-B (§8) introduces another separate write surface and is not covered by B4, 014 or the r1 approval. The final report states **preview-only** or **no-go** where appropriate; no tag, cask, tap or real-home operation is authorized by this plan.

## 8. MCP integration amendment (proposed 2026-11-12 — pending approval)

> **Status:** proposal only. This section does not authorize implementation, a new package, a new dependency, a configuration write, a network call, package installation, OAuth login, a real-home change, a commit, a tag or a release. MCP-A–MCP-D in `tasks.md` are `not_started — pending approval`. MCP-B additionally requires a separate MCP write-surface decision; it is not B4.

### 8.1 Reconciliation and scope freeze

Reconciled at checkout `4c2b9bd` (clean `main`; parent `74f568b`). The CodeGraph documentation changes the handover listed as uncommitted are committed in `4c2b9bd`; no product code changed between `74f568b` and that pre-amendment baseline.

| Claim | Reconciled state |
|---|---|
| Feature 015 Waves 1–2 accepted | Yes. |
| Wave 3 implemented/pushed as `74f568b` | Yes; still **awaiting human review**. |
| Feature 014 awaiting final acceptance | Yes; its human/Opus acceptance and batch ledger are unchanged. |
| B4 Fleet source save not authorized | Yes; `T14`–`T17` remain unchecked. |
| Fleet live application gated | Yes; `buildinfo.FleetCapabilityMarker` defaults to `preview`; `live` is build-injected only. |
| CodeGraph default-rule changes uncommitted | **Stale.** Committed in `4c2b9bd` (`AGENTS.md`, `docs/codegraph.md`, `docs/orchestration-guide.md`, router policy, docs index). |
| No new implementation batch authorized | Yes. MCP work is proposed, not authorized. |

**Scope freeze:** P0 read-only discovery/diagnostics; P1 opt-in global configuration and managed restoration; **P2 installation deferred**. The advisory study's `mcp install-missing` and `opencode launch --free` are explicitly out of scope. The aesthetic and behavioral constraints of Feature 015 r1 remain in force.

### 8.2 Existing-code reuse map

| Need | Existing code to reuse | Reuse rule | New code |
|---|---|---|---|
| Home/path resolution and CLI flags | `cmd/agentic-sdd/main.go` (`resolveHome`, flat dispatch), `cmd/agentic-sdd/fleet.go` (isolated subcommand dispatch/exit codes), `cmd/agentic-sdd/tui.go` (`--fleet-config` forwarding precedent) | Add `case "mcp"` before `default`; keep all existing flags/output byte-identical; add `mcp` to `isHelpTopic`/usage only. | `cmd/agentic-sdd/mcp.go` |
| Backup root/private directories | `internal/skillsync/backupRootFor`, `copyFile`, `fileDigest`, `RollbackInfo`, manifest conventions | Reuse the private `0700` root/digest/rollback **primitives**. Do not call `installChangesWithHook` directly: it is a skill/agent-tree writer with its own manifest and target semantics, not a proven MCP config writer. | `internal/skillsync/mcp_transaction.go` |
| Single-file atomic write | `internal/skillsync/sync.go` staging/rename/rollback pattern | Add a dedicated package-internal single-file writer in `mcp_transaction.go`; no second installer. `internal/fleet/save.go` is proposed B4 work, **not implemented**, so it is a design reference only. | same |
| Strict JSON parsing and duplicate detection | `internal/fleet/parse.go` duplicate/token-walk pattern; `internal/skillsync` digest helpers | Reuse the pattern, not the fleet schema; MCP config is a different document with its own allowlist. | `internal/skillsync/mcp_config.go` |
| Typed immutable requests, preview tokens | `internal/appops/fleet.go`, `internal/appops/requests.go` (`OperationID`, `ErrNeedsPreview`, preview-token constructors) | Same constructor/accessor/revalidation style; no stateful draft; no transcript parsing. | `internal/appops/mcp.go` |
| Single-write host | `experiments/tui/ops_host.go` (`WriteKind`, `WriteRequest`, one active slot, `BeginWrite`, `Complete`, `home()`, `source()`, `Command`) | Add `WriteMCPApply`/`WriteMCPRestore` and an `MCP` payload; keep install/restore/fleet behavior unchanged; pairwise busy-refusal tests. | `experiments/tui/ops_host.go` edits |
| Hub child and navigation | `experiments/tui/app_screens.go` (`installHubEntries`, `len(entries)` wrap/bounds, `openFleetScreen`, `closeInstallHub`) | Add a fourth entry and `installSectionMCP`; key `t` (not `m`, which is the global motion toggle); update all hub tests. | `experiments/tui/app_screens.go` edits |
| Read-only status/doctor vocabulary | `internal/fleet/status.go`, `internal/fleet/doctor.go`, `experiments/tui/fleet_doctor.go` | Reuse the honest-label and explicit-binary-path patterns; do not infer connectivity from config. | `internal/skillsync/mcp.go` |
| Responsive shell/theme/modal/focus | `experiments/tui/{viewport,theme,presentation,modal}.go`; 013 tests | No new theme/design system/fifth route; inline subview confirmations with default-No and tiny-geometry guards. | `experiments/tui/mcp_screens.go` |
| Temporary-home/offline tests | `internal/skillsync`/`internal/fleet` test patterns, `t.TempDir`, sanitized `GOPROXY=off`/`GOWORK=off`/`GOTOOLCHAIN=local` | Fake `codegraph`/`opencode` executables only; no real home, network, model or package install. | MCP tests |

**No new root Go package.** `internal/skillsync` is the existing supported-core transaction/backup engine and already owns OpenCode-managed file writes; adding `mcp_*.go` there reuses its private backup, digest and atomic-write machinery and keeps the 10-package gate unchanged. A separate `internal/mcp` package would duplicate those primitives and force a gate/package decision without a demonstrated need. A JSONC dependency is also rejected; a conservative stdlib scanner plus manual-snippet refusal is sufficient.

### 8.3 Proposed file touch list

| File | Change | Layer |
|---|---|---|
| `internal/skillsync/mcp.go` | **New** MCP server metadata, status states, target/config detection, read-only status projection. | root engine |
| `internal/skillsync/mcp_config.go` | **New** JSON/JSONC read/merge/preview; V2/V1/schema/ambiguity detection; conflict model; manual snippet. | root engine |
| `internal/skillsync/mcp_transaction.go` | **New** single-file backup/apply/rollback, MCP ownership manifest, restore preview/apply. | root engine |
| `internal/skillsync/mcp-servers.json` | **New** embedded versioned default-server manifest (`//go:embed` in this package). | root engine |
| `internal/skillsync/mcp_*_test.go` | **New** status/config/merge/transaction/negative/golden tests. | root tests |
| `internal/appops/mcp.go` | **New** immutable MCP status/preview/apply/restore/doctor requests/results, preview tokens, revalidation. | shared seam |
| `internal/appops/mcp_test.go` | **New** token/staleness/validation tests. | root tests |
| `cmd/agentic-sdd/mcp.go` | **New** `mcp status\|doctor\|preview\|apply\|restore\|help` dispatch and renderers. | CLI |
| `cmd/agentic-sdd/mcp_test.go`, `mcp_status_golden_test.go` | **New** parser/golden/exit/JSON/no-TTY tests. | CLI tests |
| `cmd/agentic-sdd/main.go` | **Edit** `case "mcp"`, `isHelpTopic("mcp")`, usage block; no existing command changes. | CLI |
| `experiments/tui/mcp.go` | **New** `MCPModel`: views, selection, preview, token, restore, generation, focus/scroll. | TUI |
| `experiments/tui/mcp_screens.go` | **New** Overview, Select, Preview, Confirm, Diagnostics, Recovery rendering. | TUI |
| `experiments/tui/mcp_adapter.go` | **New** typed adapter over `appops`; no filesystem/exec/network. | TUI |
| `experiments/tui/mcp_*_test.go` | **New** headless/responsive/focus/resize/no-write/negative tests. | TUI tests |
| `experiments/tui/app_screens.go` | **Edit** fourth hub entry, key `t`, `installSectionMCP`, dispatch, breadcrumb, `len(entries)` tests. | TUI |
| `experiments/tui/app.go` | **Edit** route/message dispatch and MCP refresh wiring; no second writer. | TUI |
| `experiments/tui/ops_host.go` | **Edit** `WriteMCPApply`/`WriteMCPRestore`, `MCP` payload, validation, `home()`/`source()`/`Command`, single-slot tests. | TUI |
| `tests/tui/*mcp*`, `Taskfile.experimental.yml` | **New/Edit** explicit MCP PTY scenario and task wiring; no core gate or default-task change. | evidence |
| `Taskfile.yml`, `harness/checks.json` | **No change expected** while the 10-package gate is unchanged. If a package is genuinely added, update both under a separately approved decision and respect the 008 fence. | gates |
| `README.md`, `docs/mcp-tools.md`, Help/onboarding copy | **MCP-D only** after implementation evidence; no product claim before then. | docs |

Not touched: `fleet.json`, `opencode-pack/agents/*`, canonical prompt/permission files, global `AGENTS.md`, `.goreleaser.yml`, release workflow, Homebrew tap, `internal/fleet` semantics, Feature 014 manifest/schema, `internal/opencodeadapter` core path.

### 8.4 Typed root/TUI contract

Proposed types (names/signatures reconciled at implementation; no frozen API yet):

```go
// internal/skillsync/mcp.go
type MCPState string // not-configured | configured | disabled | missing-prerequisite | auth-not-checked | needs-auth | conflict | schema-mismatch | unsafe-target | unknown | verified-connected | previously-managed-absent
type MCPServerStatus struct {
    Name, Type, Purpose string
    ConfigState         MCPState
    Evidence            string // static | local-probe | runtime | not-checked
    Prerequisite        string
    AuthState           string
    Disabled            bool
    Managed             bool
    Conflict            string
    IndexState          string // codegraph only: absent | present | freshness-not-checked | stale
    ConnectionState     string // not-tested | connected | needs-auth | unreachable
}
type MCPStatus struct {
    ConfigPath, ConfigFormat, Schema string
    ConfigDigest                    string
    Servers                         []MCPServerStatus
    Ambiguity, LegacyReason         string
}
func BuildMCPStatus(home, configPath, projectPath string) (*MCPStatus, error) // read-only

// internal/skillsync/mcp_config.go
type MCPEntryPreview struct { Name, Op, Reason string; Old, New []byte; OldDigest, NewDigest string }
type MCPPlan struct {
    TargetPath, Format, Schema string
    BaseDigest, PlanDigest     string
    RuntimeVersion, RuntimeEvidence string
    Selected                   []string
    Entries                    []MCPEntryPreview
    Conflicts                  []string
    AuthChoices                map[string]string
    Prerequisites              map[string]string
    NoOp                       bool
}
func PreviewMCPConfig(home, configPath string, selected []string, auth map[string]string) (*MCPPlan, error)

// internal/skillsync/mcp_transaction.go
type MCPApplyOptions struct { Home, ExpectDigest, OpenCodeBin string }
type MCPApplyResult struct { TargetPath, BackupID, BackupPath, InstalledDigest, RuntimeVersion string; NoOp bool; Applied []string; Rollback RollbackInfo }
func ApplyMCPConfig(home, configPath string, plan *MCPPlan, opts MCPApplyOptions) (MCPApplyResult, error)

type MCPTransactionIntent struct { BackupID, TargetPath, BaseDigest, PlanDigest, PriorManifestDigest, Phase string }
func ReadMCPTransactionIntent(home, backupID string) (*MCPTransactionIntent, error) // recovery evidence

type MCPRestorePreview struct { BackupID, TargetPath, BaseDigest string; Entries []MCPEntryPreview; Conflicts []string }
func PreviewMCPRestore(home, backupID, configPath string) (*MCPRestorePreview, error)
func RestoreMCPConfig(home, backupID, configPath, expectDigest string) (MCPApplyResult, error)
```

`MCPPlan.RuntimeVersion`/`RuntimeEvidence` are empty in a read-only preview and are populated by the apply-time local probe; the token binds the observed evidence. `internal/appops/mcp.go` mirrors the 013/015 pattern: immutable structs with unexported fields, constructors that validate/absolutize, exported accessors, read-only operations that never write, and write operations that revalidate a token in the same call. Proposed operations:

- `MCPStatusRequest/Result` (no token; read-only).
- `MCPDoctorRequest/Result` (`Offline`, `Network`, `Connect`, explicit `OpenCodeBin`/`CodeGraphBin`/`ProjectPath`; no implicit PATH execution in offline mode).
- `MCPPreviewRequest/Result` (`Plan`, `Files`, `Token`, `BaseDigest`).
- `MCPApplyRequest/Result` (requires token + `ExpectDigest`; no adopt/replace field).
- `MCPRestorePreviewRequest/Result`.
- `MCPRestoreRequest/Result` (requires token + digest; managed entries only).

The preview token (`appops-mcp-preview-v1`) covers: target path, config format/schema, base digest, plan digest, selected names, canonical entry digests, auth choices, tool/manifest version and scope. `ErrNeedsPreview`/`ErrStaleConfig`/`ErrMCPConflict`/`ErrMCPUnsafeTarget`/`ErrMCPUnsupported` are typed and mapped to UI copy without exposing private content.

The TUI adapter interface is exactly the typed seam:

```go
type MCPReader interface {
    Status(ctx context.Context, req appops.MCPStatusRequest) (appops.MCPStatusResult, error)
    Preview(ctx context.Context, req appops.MCPPreviewRequest) (appops.MCPPreviewResult, error)
    Doctor(ctx context.Context, req appops.MCPDoctorRequest) (appops.MCPDoctorResult, error)
    RestorePreview(ctx context.Context, req appops.MCPRestorePreviewRequest) (appops.MCPRestorePreviewResult, error)
}
type MCPWriter interface {
    Apply(ctx context.Context, req appops.MCPApplyRequest) (appops.MCPApplyResult, error)
    Restore(ctx context.Context, req appops.MCPRestoreRequest) (appops.MCPRestoreResult, error)
}
```

Writes are dispatched only by the root shell's `OpsHost`, never by a screen. The adapter has no `os`, `exec`, `net/http` or `skillsync` write access.

### 8.5 CLI contract and compatibility

`cmd/agentic-sdd/mcp.go` adds an isolated dispatch branch before `default`, following `fleet`:

```text
agentic-sdd mcp [status] [--home PATH] [--config PATH] [--project PATH] [--json]
agentic-sdd mcp doctor [--home PATH] [--config PATH] [--opencode-bin PATH] [--codegraph-bin PATH]
                       [--project PATH] [--json]
                       # --network/--connect: deferred, separate approval; not MCP-A
agentic-sdd mcp preview [--servers NAME,...] [--auth context7=oauth|api-key]
                        [--auth exa=keyless|oauth|api-key] [--home PATH] [--config PATH] [--json]
agentic-sdd mcp apply --expect-config SHA256 [--opencode-bin PATH] [--servers NAME,...]
                      [--auth ...] [--home PATH] [--config PATH] [--json]
agentic-sdd mcp restore list|preview|apply [ID] [--home PATH] [--config PATH] [--json]
agentic-sdd mcp help
```

- **MCP-A scope:** implement only `status`, offline `doctor`, read-only `preview` and `help`; `apply`/`restore` are MCP-B and must not be reachable in an A-only build.
- Default target: `<home>/.config/opencode/opencode.json`, or `.jsonc` if it is the only target; both present is an ambiguity. If neither exists, report `not configured` and print the exact manual snippet; v1 does not create a new global config (a future `--create` is a separate decision). `--config` selects an explicit file; it must be a regular non-symlink file under an allowed global scope for apply (read-only status may inspect an explicit file with a clear `explicit` label).
- `mcp` with no subcommand = `status`; `mcp help` = usage.
- `status`, `preview`, offline `doctor` write nothing, execute no binary and invoke no model/package installer/remote endpoint. They may use a bounded local `exec.LookPath` for prerequisite presence. `doctor --network`/`--connect` are **deferred out of MCP-A** to a separately approved capability; they are never default and never a TUI action.
- In **MCP-A**, `preview` prints the current recipe/static comparison and writes nothing; it carries no apply token. In **MCP-B**, it is extended to print target, base/plan digest, selected names, add/change/leave entries, conflicts, auth implications, prerequisite, backup/restore effect and a token. `--json` is stable and redacted.
- `apply` requires `--expect-config`; empty is usage (2), mismatch is stale/conflict (1), success/no-op (0). It performs a bounded local `opencode --version` probe using `--opencode-bin` when supplied, otherwise one resolved local executable; it binds the observed path/version to the token and fails closed on missing/unsupported/unknown. It prints the actual backup ID/path and applied/no-op result. `OpsHost` is process-local; a shared engine lock covers cross-process writers.
- `restore list|preview|apply` follows the existing restore style; restore preview is read-only and apply requires an explicit ID/digest. It restores managed entries only; whole-file restore is not part of v1 and cannot erase later edits.
- Existing commands/flags/exit codes/output are unchanged. `--with-opencode` does not write MCP config. `--with-mcp` is **not introduced**. The TUI never invokes this parser.

### 8.6 TUI integration

- `installHubEntries` gains a fourth entry `{key: "t", title: "MCP Tools", section: installSectionMCP}` **appended after Fleet Manager**, so existing Install/Backups/Fleet indices remain stable. All cursor wrap/bounds use `len(entries)`; `t` and Enter dispatch; `f` remains Fleet; `i`/`b` remain Install/Backups. `m` is not used because it is the existing global motion toggle. Update every hub navigation/render test.
- `MCPModel` owns `view` (Overview, Select, Preview, Confirm, Diagnostics, Recovery), `selected map[string]bool`, `auth map[string]string`, `status`, `preview`, `token`, `baseDigest`, `restorePreview`, `generation`, `focus`, `scroll`, `filter`, `confirm`, `writePending`, `outcome`. It owns no writer and never imports `os`/`exec`/`net/http`.
- Overview renders exactly three rows with name, purpose, local/remote, config state, prerequisite, verification state, next action. Status is lazy: opening the hub does not read MCP config until MCP Tools is opened.
- Select is a checkbox list with explicit auth choices for Context7/Exa; no default selection that writes. Preview is read-only and shows the exact target, digest, entries, conflicts, auth/prerequisite/backup implications, and a scrollable diff. Confirm is an inline default-No subview with focus trap and tiny-geometry guard. Diagnostics is contextual; Recovery lists only managed MCP backups and previews restore.
- `WriteMCPApply`/`WriteMCPRestore` carry an `MCPWriteRequest{Home, ConfigPath, PlanID/BackupID, Token, Label, Source}`. `WriteRequest.validate()` rejects kind/payload mismatches. `home()` returns the resolved request home; `source()` is `mcp:<configPath>` (never `embedded`); `Command` is `mcp apply --config <path>` or `mcp restore <backupID>`. `startInventoryRefresh()` handles MCP kinds by refreshing MCP status only; it does not claim installed-agent changes. Pairwise busy-refusal tests cover MCP vs install/restore/fleet apply/restore and the source-save seam (when enabled) in both directions. Existing install/restore/Fleet flush-before-result, cancellation, deferred-exit and shutdown behavior is unchanged.
- Diagnostics shows `connection not tested` and CLI guidance; there is **no active TUI connection test in v1**. Any future TUI test requires its own explicit capability and confirmation, and never runs during preview/apply. CLI `--network`/`--connect` remain deferred separate capabilities.
- Responsive/focus rules: 110/80/60 and 8/10/16; `/` filters preview lines where practical; long paths/URLs elide only in rows; exact values and full diffs are scrollable; resize cannot confirm; Back/Cancel/warning survive all sizes.

### 8.7 Configuration merge algorithm

1. **Resolve scope.** Global only in v1. Compute `<home>/.config/opencode/opencode.json(c)` or use explicit `--config`. Reject symlinks, non-regular files, traversal, paths outside the selected global scope and ambiguous `.json`/`.jsonc` pairs. If neither file exists, report `not configured` and emit a manual snippet; v1 does not create a new config.
2. **Read bounded.** `io.LimitReader` 1 MiB, UTF-8 validation, strict JSON for `.json`; for `.jsonc`, a conservative scanner that records comments, key order, duplicate keys and byte ranges. Never use a generic JSON marshal for a `.jsonc` rewrite. Record whether a comment/delimiter belongs to the managed entry, a container, or an unrelated region; ambiguous ownership blocks the edit.
3. **Detect schema.** Require V2 `mcp.servers` for management. Detect V1 `mcp.<name>`/`enabled` and refuse with migration guidance. Unknown `type`, malformed command/url, wrong required field types or duplicate server names block. Unknown top-level keys and unknown fields in unrelated/unmanaged entries are preserved, not treated as invalid.
4. **Load ownership.** Read `<home>/.agentic-sdd/mcp/manifest.json` if present. A managed name is owned only if its recorded installed digest/target matches the current entry; otherwise treat it as user-modified/unmanaged and conflict.
5. **Plan entries.** For each selected server:
   - absent and no unmanaged collision → `add` (or `restore-removed` with explicit confirmation when the manifest says it was previously managed);
   - present, semantically identical → `leave` (no-op);
   - present with `disabled: true` → `leave disabled` (never re-enable);
   - present and matches the managed installed digest but canonical fields changed → `change` (managed update);
   - present with different content and no matching managed provenance → `conflict` (refuse; no replace flag in v1);
   - Context7 with no explicit OAuth acceptance or API-key choice → `blocked: Context7 auth choice required`;
   - Exa keyless without the explicit limits/privacy acknowledgement, or no choice when OAuth/API-key was selected → `blocked: Exa auth acknowledgement required`;
   - a managed header/environment value looks like a literal secret rather than `{env:NAME}` → `blocked: literal credential`.
   The literal-credential check runs **before** any diff or error text is rendered and never echoes the value. Preserve every unmanaged server and unknown field; never add unknown fields to managed entries.
6. **Compute digests.** `baseDigest = sha256(exact target bytes)`, `planDigest = sha256(canonical plan projection)`; the appops token binds both plus selected names, entry digests, auth choices, the observed OpenCode version/runtime evidence, target and tool/manifest version. The plan is semantic, but apply writes exact bytes.
7. **Preview.** Show `add/change/leave/disable/conflict/no-op`, exact target, backup root, file mode, JSON vs JSONC, auth/prerequisite implications and restore effect. No write before confirmation. In MCP-A the preview is a static recipe/comparison without a token; the structured preview/token is MCP-B.
8. **Confirm.** Default No. The confirmation is bound to the preview token and digest; any change invalidates it.
9. **Apply.** Acquire a shared engine lock covering the target **and** ownership manifest (same lock pattern as the existing fleet engine, stale-lock handling included); the `OpsHost` slot alone is not cross-process. Under the lock, perform a bounded local `opencode --version` probe with an explicit binary path (no shell/network/model), re-read the target and verify `ExpectDigest == baseDigest`, token/plan/target identity and runtime evidence. Reject on any mismatch. Refuse a managed entry that contains a literal credential. Write a transaction-intent record (`prepared` with base/plan digests, target, backup ID, previous manifest digest) before mutation. Create `<home>/.agentic-sdd/mcp/backups/<stamp>/` (`0700`), write the exact original managed-entry text/range plus target identity/mode/digest (`0600`), then stage a same-directory temp file (`0600`), `fsync`, rename, `fsync` the directory where supported. Do not copy unrelated servers or their credentials into the backup. Refuse on filesystems without a documented atomic strategy; never best-effort overwrite.
10. **Manifest.** Write `<home>/.agentic-sdd/mcp/manifest.json` (`0600`, format 1) with managed names, target path, config format/schema, original/installed digests, per-entry installed digest, backup ID/path, tool version, manifest version, observed runtime version/evidence, timestamp and actions. No credentials or full private config. This is separate from the skillsync manifest and Feature 014's fleet manifest. Mark the intent `complete` only after the manifest and target are durable.
11. **Idempotency.** A second identical selection/digest is `up to date`; no second backup, no write, exit 0. A changed selection is a new plan/token.
12. **JSONC.** Edit only the managed entry byte ranges when the scanner proves ownership of every touched comment/delimiter and the rest of the file is unchanged; otherwise refuse and print a copyable snippet showing the exact `mcp.servers` addition/change. Apply the transformed bytes through the atomic replacement, not by rewriting the live file in place. Never silently reorder keys, strip comments or reformat unrelated regions. A future `--allow-reformat` is not part of v1.

### 8.8 Rollback and restore algorithm

- **Transaction intent.** Before mutation, write a durable intent record under the backup ID: target path, base/plan digests, previous manifest digest, backup ID, selected names, phase (`prepared`, `renamed`, `manifest-written`, `complete`). A crash leaves an inspectable intent, not an inferred success.
- **Apply failure before rename:** remove temp, leave target untouched, report `nothing changed`.
- **Apply failure after rename / during manifest write:** under the shared lock, restore the managed-entry ranges from the snapshot (preserving unrelated content) and verify the resulting managed-entry digests. If exact, report `rolled back`; if not, report `needs recovery` with backup ID, current/target digests and the exact next command. Never report success.
- **Structural recovery.** If the transaction created `mcp`/`servers` containers, recovery may remove only containers it created, only when they are empty and unchanged; otherwise refuse. If the managed entry was absent before and is present now, or vice versa, recovery locates it by semantic server name, not a stale numeric offset. Comment/delimiter ownership is re-checked. Any ambiguous structure, changed surrounding bytes, or unverifiable ownership is `needs recovery`, never a best-effort rewrite.
- **Restore preview:** read the managed backup and current target; compare only managed names. Report each entry as `restore`, `leave` (already matches), `conflict` (later user edit), `missing` or `structural ambiguity`.
- **Restore apply:** require explicit backup ID and current digest; restore only entries whose current content still matches the recorded installed digest or whose absence is expected. Preserve unrelated top-level keys, unrelated servers and later edits to a managed entry (conflict). **No whole-file restore in v1.** If entry-level recovery is ambiguous, refuse and show the manual snippet/diff.
- **Uninstall (future, not v1):** remove only matching managed entries; a user-edited entry is refused, not deleted. The manifest enables this later without reusing Fleet's manifest.
- **Recovery evidence:** every failure reports backup ID/path, target path/digest, what changed, what stayed unchanged, the intent phase and the next safe action. A file named like a backup is not evidence; only the intent/manifest and verified digests are.

### 8.9 Version and compatibility strategy

| Contract | Supported in v1 | Detection/refusal |
|---|---|---|
| OpenCode V2 `mcp.servers` | Yes | Parse/validate; V1 `mcp.<name>`/`enabled` is refused, never migrated. |
| OpenCode V2 `disabled`, `codemode`, `headers`, `oauth`, `environment`, `timeout` | Yes, documented fields only | Unknown top-level keys and unknown fields in unrelated/unmanaged entries are preserved; an unknown `type`, missing required local/remote field, wrong field type or duplicate server name blocks. Never invent V1 fields. |
| OpenCode version | V2 only for managed writes | Status/preview/offline doctor do not execute and say `runtime not checked`. Apply performs a bounded `opencode --version` probe (`--opencode-bin` or one resolved local executable) and binds the path/version evidence into the token; missing/unsupported/unknown fails closed. |
| OAuth fields | `client_id`, `client_secret`, `scope`, `callback_port`, `redirect_uri`, `auth_server_metadata_url` (snake_case) | Context7 OAuth is the recommended default after explicit plan acceptance; Agentic-SDD does not initiate login or store tokens. |
| CodeGraph | `codegraph serve --mcp`; local CLI 1.6.2 verified | Manifest records a minimum version; missing/incompatible binary → `missing prerequisite`/`unsupported`; no install. |
| CodeGraph index | `.codegraph/` presence; freshness only in a separate explicit local probe (deferred from MCP-A) | Offline status/doctor execute nothing and say `freshness not checked`; never run `init`/`index`/`sync` during global MCP setup; project action is separate and guarded. |
| Context7 | Hosted `https://mcp.context7.com/mcp`; OAuth `/mcp/oauth`; API key `Authorization: Bearer` | OAuth recommended default with explicit acceptance; API key is a separately selected alternative. Vendor docs checked 2026-11-12; exact header/endpoint re-verified at implementation. No `ctx7 setup` or automatic login. |
| Exa | Hosted `https://mcp.exa.ai/mcp`; keyless search/fetch; OAuth `?login`; API key `x-api-key` | Keyless default only after explicit rate/privacy acknowledgement; rate-limited and not unlimited; no automatic authenticated/paid upgrade. Exact tool list/auth re-verified at implementation. No private code/credentials in probes. |
| Bubble Tea/Bubbles | Existing nested Charm v2 pins | No new dependency or major-version migration; reuse list/viewport/help patterns. |

**Evidence classes:** static config parsing, local executable/config probe, apply-time runtime version evidence, and runtime MCP connection are reported separately. A `doctor --network`/`--connect` result is one observation, not a general availability or entitlement claim. Tests label static fixtures separately from runtime checks. The cross-process engine lock covers the target and ownership manifest; `OpsHost` is only a TUI-process slot.

### 8.10 Offline verification approach

| Area | Command/evidence | Pass condition |
|---|---|---|
| Root MCP engine | `go test ./internal/skillsync -run 'TestMCP'` | status/config/merge/transaction/negative tests pass; no real home/network. |
| Appops seam | `go test ./internal/appops -run 'TestMCP'` | immutable requests, token/stale/conflict behavior; no write on preview. |
| CLI | `go test ./cmd/agentic-sdd -run 'TestMCP'` plus golden/JSON/no-TTY | stable output/exit; existing CLI goldens unchanged. |
| Core gate | `task test`, `task vet`, `git diff --check` | 10 packages green; no new `require`; no gate skip. |
| TUI model | `task -t Taskfile.experimental.yml tui:test` | Overview/select/preview/confirm/diagnostics/recovery, focus/resize/state, no-write tests. |
| PTY | `task -t Taskfile.experimental.yml tui:pty-test` plus the new MCP PTY task | launch/navigate/preview/cancel/quit; no write on cancel; `not_run` if blocked. |
| Isolation | fake `codegraph`/`opencode`, `t.TempDir`, `GOPROXY=off`/`GOWORK=off`/`GOTOOLCHAIN=local` | no real home, network, model, OAuth or package install. |
| Runtime version | injected fake `opencode --version`; explicit-path probe tests | supported/unsupported/missing cases; apply refuses unknown; no network/model. |
| Concurrency | CLI/TUI competing apply fixtures; one engine lock; stale-lock tests | one writer wins; the other refuses with no write. |
| Structural recovery | inserted/removed `mcp`/`servers` containers, comment/delimiter ownership, termination at each intent phase | exact rollback or explicit `needs recovery`; never best-effort rewrite. |
| Network/handshake | `mcp doctor --network`/`--connect` | **deferred/not_run**; never claimed as pass. |

Verification must show zero-write reads, digest revalidation, exact backup/rollback evidence, single-slot pairwise refusal, cross-process lock behavior, unchanged `--with-opencode`, unchanged Fleet/013/014/009 behavior, and no new dependency. A skip/blocked check is `not_run`, not `pass`.

**Acceptance evidence map:**

| Criteria | Primary evidence |
|---|---|
| AC-51–AC-58 | `internal/skillsync` status/detection tests; `cmd/agentic-sdd` status/doctor/help goldens; hub/Overview model tests; fake-binary/no-network assertions. |
| AC-59–AC-62 | `internal/skillsync` JSON/JSONC merge/preview goldens; unmanaged/conflict/ambiguity/comment fixtures; manual-snippet assertions. |
| AC-63–AC-68 | `internal/appops` token/stale tests; `internal/skillsync` transaction/backup/rollback/restore tests; injected-failure and user-edit fixtures. |
| AC-69–AC-72 | `experiments/tui` OpsHost pairwise/slot tests; MCP confirmation/resize/focus tests; CLI no-TTY/JSON goldens; PTY scenario. |
| AC-73–AC-75 | Sanitized root/nested suites, fake-binary/PATH and temporary-home evidence; `opencode-pack`/profile hashes; explicit static/local/runtime labels; network checks `not_run`. |

### 8.11 Risks and dependencies

| Risk | Control |
|---|---|
| MCP integration becomes a second config manager | One target, one manifest, one transaction; reuse skillsync/appops; no vendor installer; no `fleet.json`. |
| JSONC rewrite destroys user content | Conservative scanner; lossless edit or manual snippet; no generic marshal; golden fixtures. |
| False connectivity/auth claims | Evidence classes; `verified connected` only from explicit check; runtime `not_run` stays `not_run`. |
| Unmanaged collision/overwrite | Conflict state; ownership manifest; exact digest; restore only matching managed entries. |
| CodeGraph init/index unsafe path | Separate explicit project action; home/root/backups/vaults guards; no auto-init. |
| OAuth/key handling leaks secrets | `{env:NAME}` placeholders; OpenCode owns OAuth; no token storage; redacted status/backups. |
| MCP write bypasses single slot | `WriteMCPApply`/`WriteMCPRestore` only; one `OpsHost` slot; pairwise CLI/TUI tests. |
| CAS is mistaken for cross-process locking | Shared engine lock covers target + ownership manifest; revalidate under lock; competing-process tests. |
| Crash leaves ambiguous structure | Durable transaction intent; semantic entry lookup; structural ownership checks; refuse ambiguous recovery; no whole-file fallback. |
| Runtime compatibility is assumed from static schema | Apply-time bounded `opencode --version` probe; evidence bound to token; unknown/unsupported fails closed. |
| Primitive reuse is overclaimed | Use skillsync primitives, not `installChangesWithHook` as a config writer; `internal/fleet/save.go` is proposed B4 work, not implemented. |
| Network/package install leaks into tests | Offline default; fake binaries; explicit network/handshake test `not_run`; no installer path. |
| New package/dependency creep | No new root package/dependency; 10-package gate; reject JSONC dependency. |
| Amendment confused with Fleet/014 approval | Separate pending amendment approval; B4/Fleet live/release gates untouched. |

**Dependencies:** amendment approval; MCP-A needs none of Fleet live/014; MCP-B needs separate MCP write-surface approval; MCP-C needs A (and B for write screens); MCP-D needs A–C. Feature 014 acceptance, Wave 3 acceptance, B4, Fleet live and release remain separate gates.

### 8.12 Workstream sequencing

| Workstream | Tasks | Depends on | Owner checkpoint | State |
|---|---|---|---|---|
| **MCP-A** | T26–T30 | **Implemented 2026-11-12**; no write authorization | CP-A: review read-only status/doctor/read-only preview/CLI evidence; confirm no writes/network | `awaiting human review — integration green` |
| **MCP-B** | T31–T37 | MCP-A + explicit MCP write-surface authorization | CP-B: review preview/digest/backup/rollback/restore/lock/runtime evidence | `authorized — parallel implementation` |
| **MCP-C** | T38–T42 | MCP-A; MCP-B interfaces | CP-C: review TUI navigation/preview/confirm/diagnostics/single-slot evidence | `authorized — parallel implementation` |
| **MCP-D** | T43–T46 | MCP-A–C | CP-D: owner walkthrough and final acceptance packet; release disposition separate | `authorized — parallel implementation` |

MCP-A can be developed and tested independently of the optional write surface. Deterministic UI/config tests do not require live remote connections, OAuth, package installation or a real OpenCode home. MCP-B must not start from the r1 or MCP-A approval alone; the owner must record a **pre-start** write-surface authorization, and CP-B evidence acceptance is a later, separate step. MCP-C's read-only screens may start after MCP-A, but its apply/restore controls remain disabled until MCP-B is integrated. MCP-D reports network/package/real-home checks as `not_run` unless separately authorized.
