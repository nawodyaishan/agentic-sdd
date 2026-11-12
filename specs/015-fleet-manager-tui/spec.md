# Feature 015 — Fleet Manager TUI

**Status:** APPROVED r1 (2026-11-12) — combined spec/plan/tasks approval in `## Approval`; Wave 1 authorized; B4 source-write remains separately gated.
**Location:** `specs/015-fleet-manager-tui/spec.md` (this directory; input drafts retained beside it).
**Prepared:** 2026-11-11.
**Depends on:** Feature 013 Unified TUI Experience (accepted, published `v0.8.0`), Feature 014 Agent Fleet Model Overrides (B1–B9 implemented, **awaiting human review + Opus verdict**), Feature 009 OpenCode orchestration (accepted behavior and permissions), `internal/appops` (Feature 013 shared operation seam).
**Target:** Paired `agentic-sdd` / `agentic-sdd-tui` distribution, root Go 1.23 + nested TUI Go 1.26/Charm v2, supported OpenCode V2 configurations only.
**Inputs:** `015-fleet-manager-tui-experience-r2-ux-optimized.md` and `015-fleet-manager-tui-ux-review-2026-11-11.md` in this directory. Both are advisory; this document is the canonical proposal.

> **Approval boundary.** Drafting and review are read-only. This document does not authorize code changes, a new batch, real-home application, private Git mutation, model/network calls, paid inference, trusted-CI adoption, tag/release/tap publication, or overriding the Feature 014 gate. The only approval that can activate work is the single combined `spec/plan/tasks` record under `## Approval`.

## 1. Executive decision

Add **Fleet Manager** to the already shipped Feature 013 TUI as a **third child of Install & Manage**, not as a fifth global route and not as a second TUI framework. Fleet Manager exposes **four local views** — **Overview**, **Profiles**, **Review Changes**, **Recovery** — with model picker, source selection, details and confirmations as contextual overlays/drill-downs.

Two facts stay visually and operationally separate at all times:

1. **Personal source** — the user's strict-JSON `fleet.json` in their private checkout. Owned by the user; changed only by an explicit save (native save or their external editor).
2. **Installed OpenCode agents** — managed files rendered by the accepted Feature 014 engine. Changed only by a fresh plan plus an explicit default-No apply.

Native source save and deployment apply are **two writes with two consents**, both default No. Saves never install; applies never commit or push Git; neither invokes a model or claims provider entitlement.

The TUI owns **view state and draft state only**. Every authoritative fact — paths, digests, plan identity, drift, ownership, locks, permissions — comes from `internal/fleet` through a typed seam in `internal/appops`. No screen reads or writes fleet files directly and no screen parses CLI text.

## 2. Verified source baseline (checkout `d5eaa7a`, inspected 2026-11-11)

### 2.1 Existing capabilities this feature reuses

| Capability | Verified state |
|---|---|
| Shell | `experiments/tui/app.go` defines exactly four root routes: Home, Install, Explore, Activity. Root owns focus, layout, modal compositor and the single-write host. |
| Install hub | `app_screens.go` `installSection` has Hub, Install, Backups; the hub has **two** entries and a two-entry index modulo. Adding Fleet touches entry count, keys, cursor bounds and hub tests. |
| Write host | `ops_host.go` has one exclusive write slot, `WriteInstall` and `WriteRestore`, typed `WriteRequest`, flush-before-result, deferred exit, progress phases and `appops.OperationID` generations. |
| Operation seam | `internal/appops` owns immutable `InstallRequest`/`RestoreRequest`, preview tokens, revalidation (`ErrNeedsPreview`) and structured results. It has **no fleet request/result type**. |
| Fleet engine | `internal/fleet` has `LoadConfigFile`, `ParseConfig`, `Resolve`, `Render`, `BuildPlan`, `SavePlan`/`LoadPlan`, `ApplyPlan`, `ListFleetBackups`, `RestoreBackup`, `RunDoctor`, manifest/lock and `TargetRootForHome`. Stdlib-only. |
| CLI | `cmd/agentic-sdd/fleet.go` implements `fleet status\|plan\|apply\|restore\|doctor` with inline status projection and text output. `fleet plan --output` writes a plan file; `fleet apply --plan` consumes one. |
| Layout/theme | `viewport.go` enforces 40×8 usable geometry; `modal.go` is the compositor for global install/restore/help confirmations; Fleet apply/restore confirmations use inline subviews with matching default-No, focus-trap and tiny-geometry guards; `theme.go`/`preferences` provide the one dark-only semantic palette plus ASCII/plain/reduced-motion modes. |

### 2.2 Verified gaps this feature must close (do not paper over in the UI)

| Gap | Consequence |
|---|---|
| **No typed status/inventory API.** `fleet status` computes labels inline in `cmd/agentic-sdd` (package `main`), which the TUI cannot import. | A TUI must not reimplement status logic or parse CLI prose. A shared read-only projection is required, and the CLI should consume the same projection while keeping its text output byte-compatible. |
| **Plans carry no rendered content.** `Plan` exposes ops, digests, source refs and model names; `RenderResolved` exists but `BuildPlan` does not return per-file desired/current bytes. | "Full diff" cannot be built from the plan JSON alone. A read-only plan-detail/diff projection is required. |
| **No restore preview.** `ListFleetBackups` lists entries and `RestoreBackup` writes; there is no read-only current-versus-restore projection. | FR-11/AC-24 cannot show exact effects without a new preview. The UI must not guess. |
| **No personal source save.** `internal/fleet` parses and plans but never writes `fleet.json`; `appops` has no save request. | Either add a small, separately approved atomic save operation with CLI parity, or keep the external-editor path and make no in-app save promise. |
| **`ApplyPlan` takes a `*Plan` in memory, while the CLI path uses a saved plan file.** | The TUI can hold the plan in memory, but the shared apply seam must revalidate config/source/targets exactly as the file path does, and must not become a second dispatcher. |
| **No adoption in 015.** `fleet plan --adopt` exists; 014 FR-15 import/adoption was deferred. | The TUI must not expose an Adopt/Force/Fix-all affordance. Adoption remains an explicit CLI decision. |

### 2.3 Explicit non-claims

- Feature 014 is **not accepted** merely because its tests are green; the handover records `awaiting human review` and an outstanding Opus verdict.
- The existing TUI is Feature 013's accepted shell; this spec does not reopen its theme, routes, safety or write policy.
- No claim is made that an installed agent is authenticated, selected as primary, runtime-verified, entitled, or cheaper because a model route is configured.

## 3. Scope, users and success

### 3.1 Users and jobs

| User | Job |
|---|---|
| New solo developer | Understand what an agent instance is, which model it uses, and what is still unverified; reach a safe first plan without a private path being guessed. |
| Multi-provider developer | Change one instance's exact `provider/model#variant`, save the personal source, preview the file changes and apply once. |
| Learner/intern | Inspect profiles, diffs and outcomes without editing prompts, permissions or provider accounts. |
| Maintainer | Reproduce drift, stale-plan, lock and restore behavior on temporary fixtures and diagnose without touching a real home. |

### 3.2 Product-level definition of done

On a supported build with the 014 dependency accepted, a user can: select an explicit local `fleet.json`; browse and search instances; stage a model or enablement change; review the exact personal-source diff; save it intentionally; generate a fresh 014 plan from the saved source; review every file change with a complete diff; apply with a separate default-No confirmation; see the real result and backup; and restore a managed backup preview-first. If any precondition is missing, the TUI explains and stops.

No real-home application is authorized by this specification.

## 4. Architecture and ownership

```text
Feature 013 shell (nested module experiments/tui, Go 1.26, Charm v2)
  Home ─ Install & Manage ─┬─ Install
                           ├─ Backups & restore
                           └─ Fleet Manager (new)
                                Overview | Profiles | Review Changes | Recovery
                                           │  view + draft state only
                                           v
                     FleetTUIAdapter (typed requests/results, no filesystem)
                                           │
                                           v
                     internal/appops (root Go 1.23 shared seam)
                       FleetStatus / FleetSourcePreview / FleetPreview / FleetDoctor (read-only)
                       FleetSourceSave / FleetApply / FleetRestore (writes)
                                           │
                                           v
                     internal/fleet (accepted 014 engine + additive read/save ops)
                       parse · resolve · render · plan · diff · status
                       apply · restore · manifest · lock · doctor
                          │                         │
                          v                         v
                 personal fleet.json        managed ~/.config/opencode/agents
                 (never Git operations)     (never prompts/permissions changes)
```

### 4.1 Ownership rules

- **TUI:** navigation, focus, presentation, in-memory draft, selection/filter/scroll, generation-bound async state, confirmations. It never calls `os.WriteFile`, `os.Rename`, `filepath.Walk`, `exec`, Git, `skillsync.Sync` or `fleet.ApplyPlan` directly.
- **`internal/appops`:** stateless immutable validated requests, preview tokens, revalidation before every write, structured results. `FleetSourcePreview` takes the config path and draft bytes (plus resolved repo/home context) and returns validated normalized bytes, `FilePreview` and the base digest without storing a draft. It must expose fleet operations to CLI and TUI identically, and must not contain fleet validation logic duplicated from `internal/fleet`.
- **`internal/fleet`:** the only owner of strict JSON parsing, source resolution, rendering, digests, plan identity, drift, ownership, lock, transaction, backup and restore. Additive read projections and the source-save operation live here, not in screens.
- **Canonical pack:** `opencode-pack/agents`, `adapters`, `presets/free-first` remain the only prompt/permission source. The personal file owns model, variant and enablement only. No prompt/permission editor in 015.
- **OpenCode:** executes models. Agentic-SDD manages source files only. Installed ≠ enabled ≠ selected primary ≠ authenticated ≠ runtime-verified.

### 4.2 Proposed additive shared API (review target)

These are design proposals, not frozen signatures. Exact names are reconciled at implementation time.

| Layer | Additive surface | Purpose |
|---|---|---|
| `internal/fleet/status.go` | `BuildStatus(configPath, repo, home) (*Status, error)`; per-instance configured/enabled/source/model/desired/current/last-applied/provenance/drift labels | One typed read projection. `fleet status` refactors onto it and keeps its current text output. |
| `internal/fleet/diff.go` | `PreviewSource(path, newBytes)` and `PlanFiles(plan *Plan)` returning `FilePreview` (old/new bytes + digests, op, conflict reason) plus a pure bounded line diff | Exact scrollable diffs without a second renderer. Reuses `Resolve` + `RenderResolved`. |
| `internal/fleet/restore_preview.go` | `PreviewRestoreBackup(home, backupID) (*RestorePreview, error)` | Read-only current-versus-restore effects and ownership/conflict checks. `RestoreBackup` remains the only writer. |
| `internal/fleet/save.go` | `SaveConfig(path string, newBytes []byte, opts SaveOptions) (SaveResult, error)` with strict parse, 1 MiB cap, exact-mode checks, **mandatory digest CAS for an existing file**, `0600` temp + atomic replace where supported, pre-save backup under `<home>/.agentic-sdd/fleet-source-backups/` | The only personal-source write path; existing selected file only in v1; no Git, no installed-file effect. |
| `internal/appops/fleet.go` | `FleetStatusRequest/Result`, `FleetSourcePreviewRequest/Result`, `FleetPreviewRequest/Result` (plan + diffs + token), `FleetDoctorRequest/Result`, `FleetSourceSaveRequest/Result`, `FleetApplyRequest/Result`, `FleetRestoreRequest/Result` | Typed CLI/TUI seam with preview-token revalidation and no transcript parsing; no stateful draft and no adopt field. |
| `cmd/agentic-sdd/fleet_config.go` | `agentic-sdd fleet config save --config PATH --from PATH --expect-digest SHA256 [--dry-run] [--home PATH]` | CLI parity for the only new mutation. Extends the frozen 014 grammar only under this feature's approval. |

### 4.3 Write boundaries

| Write | Confirmation | Engine | Never does |
|---|---|---|---|
| Save personal source | Separate default-No; exact diff + target path; existing selected file only | `SaveConfig` via `appops` | Create a new file in v1, install, apply, commit/push, touch `AGENTS.md` |
| Apply fleet | Separate default-No after a **fresh** plan from saved source; only when the reviewed live capability is enabled | `ApplyPlan` via `appops` and the single `OpsHost` slot | Commit/push, invoke models, change prompts/permissions, adopt unmanaged files |
| Restore managed backup | Separate default-No after restore preview; only when the reviewed live capability is enabled | `RestoreBackup` via `appops` | Delete unrelated files, treat `git revert` as a deployment |

All three use the existing **single** `OpsHost` write slot. Install/restore and fleet operations cannot run concurrently. A plan ID or preview token is a consistency token, never user approval.

**Confirmation pattern:** Fleet apply/restore confirmations are inline subview confirmations with the same default-No, focus-trap and tiny-geometry guards; `modal.go` remains the compositor for global install/restore/help. Do not claim modal reuse that does not exist.

### 4.4 Source-edit decision

**Chosen:** native save, because a TUI that stages edits and cannot persist them is misleading. It is bounded: one operation, one file, strict validation, CAS, backup, no Git.

**Fallback if review rejects the new write surface:** ship the read/stage/plan/apply/restore experience and replace Save with a vetted **"Open `fleet.json` in your editor"** action plus mandatory reload/revalidate on return. The UI must then make no in-app save claim and must not auto-apply. This fallback is not a silent degradation; it is recorded in the approval.

## 5. UX contract

### 5.1 Navigation and local views

Global navigation stays **Home / Install & Manage / Explore / Activity & Help**. Fleet Manager is the third child of Install & Manage (key `f`) and has a contextual Home affordance. No fifth route.

| View | Default content | Primary action | Contextual surfaces |
|---|---|---|---|
| **Overview** | Selected source, two-state checkpoint (personal config / OpenCode files), managed counts, last outcome, one next action, runtime `not checked` | Select source or inspect profiles | Source dialog, diagnostics, last-operation detail |
| **Profiles** | Searchable instance list: role, instance ID, provider route, compact model, enabled, installed/drift state, changed count | Inspect or stage a model/enablement change | Detail drill-down, Model Picker, enable/disable consequence dialog |
| **Review Changes** | Impact summary first: creates/updates/**removals**/no-ops/conflicts, scope, plan identity; source diff then deployment diff | Review source diff → save; then fresh plan → apply | Full scrollable diff, both confirmations, result/backup |
| **Recovery** | Only real conflicts, drift, legacy provenance, managed backups and failed/partial results | Replan, inspect, preview restore, CLI guidance | Error detail (`what happened / what stayed unchanged / next safe action`), restore modal |

Selections, filters, scroll, current view and valid draft survive overlay open/close and parent-route round trips. No tab is created for a picker, dialog or one-off error.

### 5.2 Three independent state tracks

| Track | Display vocabulary | Never inferred |
|---|---|---|
| Personal source | `not selected`, `missing`, `invalid`, `loaded`, `draft dirty`, `saved locally`, `changed externally` | Git committed/pushed |
| Installed agent | `not installed`, `managed current`, `pending apply`, `drifted`, `unmanaged`, `legacy unknown`, `disabled (managed removal)` | Authenticated, primary, safe to overwrite |
| Runtime | `not checked`, `catalog visible`, `locally verified` only from accepted evidence | Entitlement, cost, model success |

Operation state is a fourth, transient track: `idle`, `previewing`, `ready`, `saving`, `applying`, `restoring`, `blocked`, `failed`, `rollback incomplete`. A green marker never replaces text.

### 5.3 Primary flows

- **A — Inspect (no writes):** open Fleet → choose explicit source or labeled read-only example → browse/search/detail → status/doctor → exit.
- **B — Change one model:** Profiles → select instance → pick/enter exact model → staged old→new → Review Changes source diff → **Save personal source? [No]** → "Saved locally. OpenCode files unchanged." → **Preview OpenCode changes** → fresh plan + full diff → **Apply to OpenCode files? [No]** → actual result + backup.
- **C — Disable:** consequence dialog states managed removal + backup; unmanaged files cannot be removed by the toggle.
- **D — External source edit:** digest mismatch → `Source changed outside the TUI` → reload/review; never overwrite unseen edits; draft preserved where safe.
- **E — Installed drift:** `Installed file modified` → no default overwrite → inspect diff, replan, CLI recovery.
- **F — Stale plan:** any source/target/renderer/scope change → apply refuses → fresh plan + fresh consent; staged choices retained if safe.
- **G — Failure/restore:** real failed/rolled-back/needs-recovery state, backup ID and next safe action; restore only after its own preview and confirmation.

### 5.4 Keyboard, layout, theme

Reuse 013 keys: arrows/`j`/`k`, Enter, Esc, `/`, Tab/Shift+Tab, `?`, Home/End, `q` outside inputs. Printable keys are literal text whenever an input is focused. One focus owner. Footer shows only current-view keys.

Layout: ≥110 columns list+detail; 80–109 compact/drill-down; 60–79 single pane; <60 or height <8 resize guidance and **no affirmative key**. Warnings, Back/Cancel, selection and the full confirmation survive every supported size. Long paths/models elide only in list rows; exact values and full diffs are reachable by scrolling. One dark palette, text+shape cues, ASCII/plain/no-color modes, reduced motion.

## 6. Functional requirements

**FR-01 — Shell integration.** Add Fleet Manager inside the existing Install & Manage child set and a contextual Home entry. Use `len(entries)` for hub cursor wrap/bounds, dispatch Enter to each of the three entries and key `f` to Fleet; replace the two-entry assumptions and update all hub tests in B1. Preserve the four global routes, 013 onboarding, activity, update behavior, help, installer, backups, focus and resize contracts. Do not create a nested application that steals global keys.

**FR-02 — Typed status and truthful inventory.** Derive all counts, labels, roles, models and availability from a typed shared projection. Distinguish the three state tracks in §5.2. No hardcoded 18/14/100 counts, no single `Ready` label, no UI-derived security decision.

**FR-03 — Profile browser and detail.** Provide a filterable, keyboard-driven list of instances with role, source provenance, model, enablement and installed state. Detail shows canonical role vs alias, full model, engine-derived permission summary and unverified runtime facts. Never construct permissions from display strings.

**FR-04 — Model picker.** Allow selecting an exact `provider/model[#variant]` from current bindings, locally known assignments, locally cataloged routes (only if a supported local source exists) or explicit entry. Suggestions are labeled by source, never ranked by price or entitlement. Validation comes from shared root validation. No inference or network call occurs.

**FR-05 — Enablement consequences.** Enable/disable only for instances 014 manages. Show the resulting create/update/**managed removal** and backup; disable cannot delete an unmanaged file. Do not conflate enabled, installed, default primary or working subagent.

**FR-06 — Explicit local source.** Require an explicit path (typed in the UI or passed by an explicit `--fleet-config` flag). Reject symlinks, traversal, non-regular files and unsupported paths per 014. Never scan the home directory, infer a private Git path, clone, read credentials, or contact a Git remote. Source preference is session-only; no new persistence without separate scope. Source save in v1 targets an **existing** selected file; creating a new `fleet.json` is deferred to a separately approved operation.

**FR-07 — Staged draft.** Keep edits in memory only in `FleetModel` until an explicit save; `appops` stays stateless. `FleetSourcePreview` is the pure-read contract in §4.1, not a draft store. Show `N unsaved edits`; protect against Esc/navigation/modal/resize loss with keep-editing/discard/stay. Never write implicitly. Do not let late reads or resize clear the draft.

**FR-08 — Personal source diff and save.** Show the exact strict-JSON diff and target path before a separate default-No save. The save operation must: validate schema/allowlist/UTF-8/size with 014 rules; require an existing regular file whose **last-observed digest is non-empty and matches** (a new file is not created in v1); check path identity, mode, ownership and symlink state; serialize deterministically; use a `0600` temp file, atomic replacement where supported, and a recoverable pre-save backup outside the user's Git checkout under `<home>/.agentic-sdd/fleet-source-backups/`; write only that file; perform no Git action; report stale-source conflicts without data loss. Windows or filesystems without atomic replace must refuse or use a documented safe strategy, not a best-effort overwrite. If the native surface is not approved, use the vetted external-editor fallback and claim no in-app save.

For FR-08, `SaveConfig` requires `SaveOptions.ExpectDigest`: empty is a usage/refusal error, mismatch is `ErrStaleSource`, and neither permits a write. Source backups are also outside the skillsync backup namespace; they are mandatory for a changed file.

**FR-09 — Fresh plan and full diff.** Generate a new 014 plan from saved current source via the shared seam. Show scope, exact create/update/remove/no-op/conflict counts, conflict reasons, plan ID, source/target digests where safe, model changes and permission implications. Removals/conflicts appear before routine updates. Provide the complete scrollable per-file diff, including >1,000 lines and long Unicode paths. A saved plan is not approval.

**FR-10 — Explicit apply and outcome.** Dispatch apply only when: the reviewed **live capability** is enabled for this build (not a per-screen toggle) and 014 is accepted/eligible for the intended scope; a current validated plan exists; conflicts/drift are resolved by an accepted operation; a separate default-No confirmation is given; `appops` revalidates plan/source/target and the engine lock is held. When the live capability is disabled, apply is absent or disabled with `Preview only`, and no key sequence can dispatch it. Show real progress phases, actual created/updated/removed/no-op/rollback result and backup. Do not offer unsafe cancellation or claim an in-flight transaction was cancelled.

**FR-11 — Restore preview and outcome.** Use `ListFleetBackups` plus a read-only restore preview. Show exact current-versus-restore delta, conflicts, ownership limits and backup identity. Restore requires its own default-No confirmation, the reviewed live capability, and refuses if the target changed since preview. Never imply `git revert` restores installed files. Existing non-fleet backups remain in the 013 flow.

**FR-12 — Drift, legacy and conflict recovery.** Display the 014 three-state facts (last applied, current, desired). Offer only supported actions: reload, replan, inspect, preserve installed change, restore a managed backup, CLI guidance, stop. Format-1–3/4 ambiguity stays `unknown` until explicitly adopted. No auto-adopt, force, fix-all or automatic private-Git reconciliation while 014 FR-15 is deferred. The shared fleet preview request has **no adopt parameter** and hardcodes `adopt=false`; the TUI cannot construct an adopting plan. Adoption remains an explicit CLI decision outside 015.

**FR-13 — Diagnostics and actionable errors.** Translate typed errors without masking the technical cause. Cover invalid/duplicate/oversized JSON, bad schema, unknown source, invalid model, missing config, source changed, target drift, unmanaged collision, symlink refusal, stale plan, lock contention, corrupt/missing backup, interrupted apply, restore refusal, missing auth evidence and engine unavailable. Redact credentials, tokens and private payloads. Every failure states what happened, what stayed unchanged and the next safe action.

**FR-14 — Orchestrator visibility.** Show `sdd-orchestrator` as a normal managed profile; installed/enabled/selected-primary/runtime-observed are separate facts. Never set the OpenCode default or launch a subagent. Any future activation is a separate approved operation.

**FR-15 — CLI parity.** Every TUI mutation has an equivalent documented root CLI/application operation. The only new mutation is personal source save (`fleet config save`); it uses the same validation, mandatory digest CAS, backup, dry-run and exit codes. The TUI never spawns `agentic-sdd fleet ...` and parses text as its security API. Preserve `status|plan|apply|restore|doctor` and non-fleet behavior.

**FR-16 — Async and single-write discipline.** Run status/plan/doctor/save/apply/restore through the existing command channel. Tag every read with `{config path, source digest, scope, generation}`; discard stale replies. Writes use the one `OpsHost` slot; rapid Enter, resize, navigation and duplicate messages cannot start or submit a second write. Fleet source-save, fleet apply, fleet restore and existing skill install/restore are mutually exclusive; a busy host must refuse the new request explicitly. A late result updates its own outcome without new writes or draft loss.

**FR-17 — Responsive, accessible, dark-only.** Reuse the 013 `agentic-dark` palette, typography, focus tokens and reduced-motion/plain/ASCII preferences. No light mode or global redesign. Meet the §5.4 breakpoints. Never rely on color alone. Narrow/short geometry must not allow an unseen affirmative action. Long exact values and diffs remain reachable.

**FR-18 — Keyboard and forms.** Honor 013 navigation and help. While a text field is focused, `j`/`k`/`q`/`/` are literal. Esc with a dirty draft asks before discarding. Tab cycles visible actionable controls; modals own focus. Footer shows only relevant keys; `?` opens current-view help.

**FR-19 — Provider taxonomy, not a marketplace.** Distinguish provider **route** from known model **family/upstream**. Group `opencode`, `opencode-go`, `9router`, `openai`, custom routes by locally known facts only. No remote catalog browsing, shopping, pricing, subscription claims, account APIs, synthetic providers or marketplace installs.

**FR-20 — Dependency and release guard.** If 014 is not accepted for the intended deployment scope (human review, Opus verdict, trusted-base posture, runtime exercise, release authorization), 015 may ship read/stage/source-save (if B4 is approved) and fixture-preview capability, but no live apply/restore, and must say `Preview only` or `Fleet operations unavailable in this build`. The live gate is a **single reviewed build capability** injected from release metadata (for example `internal/buildinfo`, default `preview`); no env, config, CLI flag or key can toggle it at runtime. Tests inject live=true to exercise apply/restore; the shipped default hides/disables both and cannot dispatch them. Setting `live` in `.goreleaser.yml` or the release workflow is a separate release decision after 014 acceptance, outside 015 implementation. Green isolated tests are not a `Ready` badge.

**FR-21 — Maintenance and compatibility.** Root `go.mod` stays Go 1.23 with zero `require`; nested TUI stays Go 1.26 with pinned Charm v2; no root import of nested code. New view/tests live with `experiments/tui`; shared operations with `internal/appops`/`internal/fleet`. Preserve the 013 four-route shell, 009 `--with-opencode` 14-profile behavior, paired cask launch, CLI/non-TTY behavior and existing backup semantics. `SaveOptions.Home` and the appops save request carry the resolved home; `WriteRequest.home()` returns the resolved request home for `WriteFleetApply`, `WriteFleetRestore` and `WriteSourceSave` so `openFileWriteLog` can open the private log. Extend `WriteRequest.source()` and `Command` for all three kinds: never label fleet source as `embedded`; headers identify `fleet config save`, `fleet apply --plan <planID>` (an identity label, not a temporary plan-file claim), and `fleet restore <backupID>`. `startInventoryRefresh()` handles fleet apply/restore, refreshing catalog and Fleet status; source save refreshes Fleet status without implying installed-file change. Update `Taskfile.yml`/`harness/checks.json` only if the package gate actually changes; respect the 008 supported-core fence.

**FR-22 — Onboarding and docs.** Add one skippable inline Fleet hint and one Help topic: choose source → change model → save source → preview/apply separately. Explain role vs instance, source vs rendered vs installed, and that installed ≠ authenticated ≠ primary. Provide a temporary-home demo with one model change, one conflict and one restore. Link the canonical 014 CLI docs. No private path exposure.

## 7. Non-functional requirements

| ID | Requirement |
|---|---|
| NFR-01 | In-memory navigation/focus/filter across 18 and 100 instances must not block on disk; record median/p95, target p95 ≤100 ms for a 100-row fixture; investigate failures. |
| NFR-02 | Opening the TUI without Fleet must not read fleet JSON, run plan/doctor, invoke `opencode`, or make network/model calls. Record startup delta vs the 013 baseline. |
| NFR-03 | Usable with 18/100 instances, 100 plan ops, >1,000 diff lines, long paths/models; bounded viewport, no unbounded render stall. |
| NFR-04 | No stale source/plan/target write can succeed; prove with deterministic race/conflict fixtures. |
| NFR-05 | Simulated mid-apply failure yields accurate failure/rollback/backup state with no unrelated file loss. |
| NFR-06 | No view, plan, save, apply, restore or error path invokes an LLM, provider API, catalog refresh or paid call. |
| NFR-07 | No credentials, tokens, full private JSON, unexpected home paths or Git metadata in logs, UI, fixtures or telemetry. |
| NFR-08 | Keyboard-only completion; visible focus; text+shape status; plain/ASCII/no-color/reduced-motion; documented CLI alternative for screen-reader users. |
| NFR-09 | Essential content, warnings, Back/Cancel and default-No reachable at 110/80/60 columns and heights 8/10/16, including long paths and 100 ops. |
| NFR-10 | No root dependency/toolchain change; no 009/013/014 CLI contract regression; nested companion build preserved. |
| NFR-11 | Evidence distinguishes UI-fixture success from a genuine engine run; record commit, config digest and environment for any release claim. |
| NFR-12 | Offline read-only workflows function without Git remotes, provider auth or Internet; unavailable services show nonblocking diagnostics. |

## 8. Acceptance criteria

Every criterion records **Pass / Fail / Blocked / Not run** with revision and verification class. Static UI assertions never become claims about live OpenCode.

### A. Navigation and discoverability

- **AC-01 — Four routes preserved.** Home, Install & Manage, Explore, Activity & Help keep their existing keyboard/resize behavior and default installation behavior.
- **AC-02 — Fleet entry.** Home and Install & Manage reach Fleet Manager; hub cursor wrap/bounds use `len(entries)`, Enter reaches all three entries and `f` reaches Fleet, with all hub tests updated in B1. A missing/unsupported engine shows honest preview/unavailable state, not a dead action.
- **AC-03 — Context continuity.** Profile selection, detail, plan preview and back retain selection, filters, scroll and source without losing other 013 screen state.
- **AC-04 — Responsive layout.** At 110/80/60 columns and heights 8/10/16, material status, actions, Back/Cancel and warnings remain reachable; no confirmation is clipped.
- **AC-05 — Keyboard/focus.** `j/k/q` and `/` are text in inputs; Esc/Tab/help follow 013; no destructive shortcut fires while editing; modals keep focus.

### B. Truthful inventory and model selection

- **AC-06 — Dynamic inventory.** Counts/states derive from typed results for zero, one, 18 and 100-instance fixture fleets; no machine-specific constant.
- **AC-07 — Role/instance distinction.** One canonical role with several 9Router/Console/Go instances shows distinct IDs/models and one shared role policy with visible source provenance.
- **AC-08 — Exact model syntax.** Picker searches bindings, selects locally known IDs and accepts valid custom `provider/model#variant`; invalid input surfaces 014 errors without mutation.
- **AC-09 — No entitlement invention.** No `ChatGPT authenticated`, guaranteed 9Router upstream, or `free forever` label from a configured identifier alone.
- **AC-10 — Model-only preservation.** A staged model change does not alter prompt body, role, permissions, 009 fallback rules or approvals; a permission delta blocks model-only apply.
- **AC-11 — Enablement consequences.** Disabling shows the exact managed removal/omission in the accepted plan; unmanaged/profile conflicts cannot be silently removed.
- **AC-12 — Orchestrator state.** Installed/enabled/selected-primary/runtime facts are separate; the TUI sets no default and launches no subagent.

### C. Personal source

- **AC-13 — Explicit source.** Selecting `fleet.json` requires a known path and rejects unsafe/invalid/inaccessible/symlink-escaped inputs per 014; no home scan, clone or Git-host contact.
- **AC-14 — Staged editing.** Changes update only `FleetModel` in-memory draft state until an explicit save; stateless `appops.FleetSourcePreview` takes config path + draft bytes and returns validated normalized bytes + `FilePreview` + base digest without retaining draft state. Quitting dirty offers discard/return rather than writing.
- **AC-15 — Source preview and consent.** Exact JSON changes and resulting bindings are visible before a separate default-No save; installed files do not change in this step.
- **AC-16 — Atomic save.** A shared service performs strict validation, mandatory non-empty digest match for an existing regular file, `0600` atomic replace where supported, and a recoverable backup under `<home>/.agentic-sdd/fleet-source-backups/`, outside the checkout and the skillsync backup namespace; screens do not write; no Git operation; a new-file save is refused in v1.
- **AC-17 — External edits preserved.** A digest mismatch returns `ErrStaleSource` with no write or data loss; an empty `ExpectDigest` returns a usage/refusal error, never implicit overwrite; reload/review is required.
- **AC-18 — CLI parity.** The same save operation and constraints are available through the documented root CLI, or the external-editor fallback makes no in-app save claim.

### D. Deployment, drift and recovery

- **AC-19 — Full plan and diff.** Every create/update/remove/no-op/conflict, exact bindings, scope and complete diff (including >1,000 lines) are shown; removals/conflicts precede routine updates; scroll/selection retain identity.
- **AC-20 — Two confirmations.** Source save and OpenCode apply each require their own confirmation with No preselected; declining either causes no corresponding mutation.
- **AC-21 — Plan integrity.** A plan is generated from current saved JSON; any source/target/renderer/scope/installed change after preview makes apply refuse and require replan.
- **AC-22 — Drift refusal.** Unmanaged files, changed managed files and legacy/unknown provenance cannot be silently overwritten; the reason and next permitted action are shown.
- **AC-23 — Accurate results.** Apply, no-op, conflict, abort and partial/failed rollback display distinct outcomes from the real typed result; no optimistic green success. Transcript tests for source-save/apply/restore assert the resolved home reaches `openFileWriteLog` and the fleet source is not `embedded`; headers identify `fleet config save`, `fleet apply --plan <planID>` and `fleet restore <backupID>` per FR-21.
- **AC-24 — Scoped restore.** Restore affects only a selected managed 014 backup in a supported scope; stale/modified targets refuse; unrelated files are never deleted.
- **AC-25 — Error transparency.** Lock contention, plan mismatch, permission failure, corrupt backup, unsupported runtime and other engine errors yield actionable redacted explanations; no generic success masks a failure.
- **AC-26 — Deferred adoption respected.** No active auto-import/adopt/force/fix-all action is exposed while 014 FR-15 remains deferred, even if a mockup shows one; the shared preview request has no adopt parameter and always plans with `adopt=false`.

### E. Product, quality and release

- **AC-27 — Shared boundary and one write slot.** Source inspection/tests show Fleet screens call only typed adapter/appops methods; they do not spawn CLI, parse prose, mutate files or compute ownership/plan digests locally. Fleet source-save, fleet apply, fleet restore and existing install/restore share the one `OpsHost` slot; an active operation refuses every other write in both directions.
- **AC-28 — 013 regression.** Install & Manage preview/install/restore, Explore, Home, preferences, Help, activity, focus and paired-binary launch still work and do not load fleet source unexpectedly. Fleet apply/restore completion refreshes catalog and Fleet status through `startInventoryRefresh()`; source-save refreshes Fleet status without changing installed state. Generation-bound tests prevent stale catalog/Fleet results.
- **AC-29 — 009/014 regression.** `--with-opencode` keeps 009 Option B 14 profiles; 014 `status|plan|apply|restore|doctor`, 18-name renderer and legacy manifest tests pass unchanged or with reviewed additions.
- **AC-30 — Toolchain isolation.** Root `go.mod` stays Go 1.23 dependency-free; nested TUI stays Go 1.26 with pinned Charm v2; no root import of nested UI code.
- **AC-31 — Local-only safe mode.** A temporary-home walkthrough inspects/stages/plans/status with no Internet, model calls, credentials, real-home writes or Git changes.
- **AC-32 — Responsive/performance evidence.** Tests record p95 in-memory navigation (100 agents), large-diff reachability, resize behavior and startup delta with host context; no broad guarantee.
- **AC-33 — Keyboard/accessibility.** Keyboard-only safe journey, plain/ASCII/no-color, static/reduced motion, contrast and narrow-terminal confirmation recovery are exercised headlessly/PTY.
- **AC-34 — Shipped artifact parity.** A selected release candidate's paired CLI/TUI use compatible shared contracts and are exercised in isolated mode before support is claimed.
- **AC-35 — Security/no surprise charges.** No path invokes hosted models, reads provider secrets, writes `AGENTS.md`, changes the default primary, runs `git push`, broadens permissions or edits real home in tests.
- **AC-36 — Dependency-gate honesty.** 014 review/Opus/trusted-base/runtime/release gates are reported honestly. One release-metadata build capability defaults to `preview`: both apply and restore are hidden/disabled, labeled `Preview only`, and undispatchable by any key. No env/config/CLI/key runtime override exists; tests inject live=true for both flows. Enabling `live` in release configuration requires a separate release decision after 014 acceptance, never 015 implementation or implicit publication permission.
- **AC-37 — Owner walkthrough.** One ordinary journey (choose source → model change → save → plan → apply → restore), one modified-file conflict and one no-config path are reviewed in temporary fixtures.
- **AC-38 — Completion packet.** Exact changes, state, supported platform, tests run/not run, PTY/screenshot evidence if produced, tested guarantees, gaps, release disposition and next human action. Green tests are not release approval.

### 8.1 Disposition of the r2 supplemental criteria

The advisory r2 draft added AC-39–AC-50. They are not dropped; each is enforced by a canonical criterion above:

| r2 ID | Canonical mapping |
|---|---|
| AC-39 four local views | AC-01, AC-02, AC-03 |
| AC-40 save/deploy checkpoint | AC-15, AC-20, AC-23 |
| AC-41 draft continuity | AC-03, AC-14, AC-17 |
| AC-42 impact-first full review | AC-19 |
| AC-43 model route taxonomy | AC-07, AC-08, AC-09 |
| AC-44 focus/double-submit | AC-05, AC-20 |
| AC-45 scope/async correctness | AC-15, AC-17, AC-21 |
| AC-46 one active write | AC-27 |
| AC-47 first-use no-config | AC-02, AC-13 |
| AC-48 recovery guidance | AC-22, AC-25 |
| AC-49 layout/no-color | AC-04, AC-33 |
| AC-50 no hidden release activation | AC-36 |

## 9. Required negative and failure scenarios

The plan must allocate fixtures/tests for at least:

1. No `fleet.json`: explanatory empty state; zero writes.
2. Parse error/duplicate key/unknown policy field: actionable error; zero writes.
3. Nonexistent/symlink-escaped path: no hidden read/write, no home fallback.
4. Exact provider ID not in local catalog: stage with `availability unknown`; not authenticated.
5. Model-only change on a review agent: prompt/permissions unchanged.
6. Disable managed instance: correct removal + backup; no unmanaged deletion.
7. Staged edit then Esc: explicit discard/back; source unchanged.
8. External edit before save: digest mismatch; reject; retain recoverable choices.
9. Source saved: OpenCode unchanged until separate apply.
10. Saved source becomes invalid before plan: plan fails; no apply.
11. Plan changed between preview and confirm: stale refusal.
12. Target changes between preview and apply: lock/revalidation refuses; no overwrite.
13. Legacy/unknown provenance: displayed unknown; no fabricated hash/adoption.
14. Unmanaged collision: exact conflict; file untouched.
15. Interrupted apply: accurate rollback/recoverable state.
16. Restore backup missing/corrupt or target drifted: refusal + safe next steps.
17. Two rapid Enters on Apply: at most one accepted operation.
18. Navigation/resize during apply: no duplicate/no false cancellation claim.
19. `q / j k` in search/path input: literal text.
20. Resize 110→50 and short height during confirmation: No/Cancel/warning usable.
21. >1,000-line diff with Unicode paths: every material line reachable, attributed.
22. Non-TTY/scripted launch: existing CLI semantics; no surprise full-screen.
23. 014 not accepted/missing build: preview-only; no production apply.
24. `opencode models` shows a route but auth unchecked: `cataloged/unknown`.
25. Imported reviewer isolation claim: TUI never reports it secure after a model change.
26. Git checkout present but metadata read refused/unknown: no crash, no Git command.
27. CLI and TUI target the same update: shared lock/stale checks prevent conflict.
28. Ordinary skills install without Fleet: no fleet read or provider connection.
29. UI reports success after a partial failure: test **fails**.
30. Read-only/user-owned path with changed mode: refuse without damaging the file.

## 10. Verification strategy

| Layer | Evidence | Establishes | Does not establish |
|---|---|---|---|
| Pure UI-model | navigation, focus, state transitions, search, staged diff, default-No, error mapping | TUI contract under controlled messages | Filesystem transaction or model behavior |
| Typed adapter | fake appops for status/preview/save/apply/restore/doctor | correct request once; no shell/prose parser | real file safety |
| Root engine | 014 fleet tests + additive status/diff/save/restore-preview tests; 009/skillsync/appops regressions | domain/transaction behavior on fixtures | live OpenCode permissions |
| Temporary-home integration | 18/100 instances, save vs install, drift, lock, rollback, restore | intended isolated-home effects | permission to install to a real home |
| Headless Bubble Tea v2 | Update/View messages, size fixtures, semantic focus/viewport | interactive model correctness | all terminals/screen readers |
| PTY smoke | launch, narrow resize, search, preview/confirm, recovery, quit | real terminal routing in a controlled environment | untested platforms |
| Packaging | nested/root builds, paired candidate invocation, installed help/status | artifact contract on tested OS | future releases/other OS |
| Human review | owner end-to-end walkthrough + accessibility fallback | acceptance of the documented experience | statistical novice study |

**Restrictions:** no real `~/.config/opencode`, private Git checkout, network, model calls or paid provider use. Isolated roots, fixture JSON and fake runtime evidence only. Root tests use `GOPROXY=off`/`GOWORK=off` and the approved package gate; nested Charm deps stay in their module. Missing Linux/macOS/manual checks are `not_run`, never `pass`.

## 11. Delivery grouping (not batch authorization)

| Group | Outcome | Depends on |
|---|---|---|
| D1 — Read-only Fleet entry | Third hub child, Overview, Profiles/search/detail, typed status, responsive tests | Accepted 013 shell; additive status projection |
| D2 — Staging and review | Model picker, enablement staging, dirty protection, source diff, impact-first review | D1; verified schema/renderer |
| D3 — Deployment | Fresh plan, full diff, default-No apply through OpsHost, result/backup | D2; 014 eligible; plan/diff projection |
| D4 — Source save | Native save operation + CLI parity + CAS/backup tests (separately gated) | D2; approval of the new write surface |
| D5 — Recovery | Restore preview/restore, drift/legacy/conflict guidance, doctor | D1, D3; restore preview |
| D6 — Robustness and docs | PTY/resize/accessibility/large-fleet, paired artifact, owner walkthrough, docs | D1–D5 |

If the 014 gate remains blocked, D3/D5 land in fixture/preview mode only; no production apply path is enabled to complete the TUI.

## 12. Deferred and excluded

- No fifth global destination, alternate theme, architecture migration or replacement of 013.
- No inference, provider login, account switching, quota/cost monitoring, retries or model execution manager.
- No new fallback/delegation policy; 009 remains authoritative.
- No marketplace, remote catalog, registry, ratings, signing claims or remote JSON fetch.
- No prompt/role/permission/AGENTS.md editor; no policy changes through model editing.
- No adoption/import of unmanaged machine profiles while 014 FR-15 is deferred.
- No automatic Git clone/commit/push/pull or credential handling.
- No unreviewed dependency expansion; no Huh migration without a separate value/maintenance assessment.
- No second approval ledger; `specs/015-fleet-manager-tui/` and the repo SDD records are canonical.

## 13. Risks and release gates

| Risk | Response |
|---|---|
| 014 still awaiting human/Opus review | No production apply/restore until accepted; fixture/preview development only. |
| Public/working-tree snapshots differ | Plan freezes the exact checkout SHA and re-inspects the seams before edits; historic records are not rewritten. |
| TUI becomes a second config manager | Typed shared seam; CLI parity; additive projections only; no screen-side validation/hashing. |
| Model-only change widens tools | Renderer/permission gate from 014; surface engine conflict, never a local "safe" badge. |
| False auth/availability | configured/cataloged/unknown vocabulary; auth only from accepted local evidence. |
| Double confirm/dispatch | Two boundaries, one write slot, generation/identity tests, rapid-Enter/resize negatives. |
| Source drift after open | Digest CAS; full diff; stop on stale. |
| Long diff clipped | Reuse/extend 013 viewport; reachability tests. |
| 013 UX regression | Reuse shell/theme/preferences; full 013 regression suite. |
| New root deps/migration | Root Go 1.23 stdlib; nested Charm v2 isolated. |
| Real-home hit in tests | `t.TempDir`/isolated HOME; no network/model calls. |
| Artifact mismatch | Paired candidate test, checksums/packaging check, separate publication approval. |

**Release readiness:** implementation and green tests are insufficient while 014 is unresolved or the paired candidate is unexercised. The final report says **no-go** or **preview-only** where appropriate. No design work grants tag, cask/tap, CI trusted-base or production-install authority.

## 14. Completion and owner-review packet

1. Checkout SHA and pre/post dirty/untracked state.
2. 013 shell/tokens and 014 engine versions actually integrated.
3. Final navigation/screens, keyboard map, accessibility modes and screenshots/PTY recordings only if produced.
4. Exact typed operation/API changes, proof screens do not mutate files, CLI parity.
5. Two-stage save/apply consent, stale conflicts, model/enablement flows.
6. Status provenance and blocked/unknown auth/provider/permission states.
7. Source and installed backup/restore scenarios, rollbacks and interrupted/error cases.
8. Root, nested TUI, regression, PTY and packaging checks with Pass/Fail/Blocked/Not run and candidate identity.
9. Remaining Opus/trusted-base/runtime and IR-06-01 limits; platform exclusions.
10. Recommended go/no-go and the next action requiring explicit human authorization.

## Approval

**Specification state:** APPROVED r1 (2026-11-12) — combined spec/plan/tasks approval recorded once here.
**Combined spec/plan/tasks approval:** `approved by owner` on 2026-11-12. Exact current-session message: “3 docs approved / update task batches docs in a way to use maximum parallel implement sub agents using free models deepseek high gemini flash / and start the work”. This approves `spec.md` r1, `plan.md` r1 and `tasks.md` r1, including the parallel execution-wave amendment.
**Authorized implementation:** Wave 1 lanes W1-A (root fleet engine), W1-B (TUI shell/read-only Fleet) and W1-C (appops preview/apply/doctor seam), as defined in `tasks.md` §Parallel execution waves. Later waves start only after Wave 1 integration and review.
**Wave 1 acceptance:** `accepted by owner` on 2026-11-12, current-session message “wave 1 approved start wave 2”. This accepts the W1-A/W1-B/W1-C integration-green evidence and authorizes Wave 2 (W2-A appops/CLI, W2-B TUI staging/review, W2-C TUI deployment/recovery) as defined in `tasks.md`.
**Wave 2 implementation:** complete and integration-green on 2026-11-12; accepted by owner (see Wave 2 acceptance below). B4 source save remains separately gated.
**Wave 2 acceptance:** `accepted by owner` on 2026-11-12, current-session message “wave 2 approved .. start wave 3”. This accepts the W2-A/W2-B/W2-C integration-green evidence and authorizes Wave 3 (W3-A doctor/live capability, W3-B scale/negative evidence, W3-C PTY/artifact, W3-D docs/onboarding) as defined in `tasks.md`.
**Wave 3 implementation:** complete and integration-green on 2026-11-12, `awaiting human review`. Completion packet: [`w3-acceptance.md`](w3-acceptance.md). No release, live capability enablement or B4 authorization is implied.
**B4 source-save surface:** design approved; implementation remains separately gated. Wave 1 and Wave 2 do not start `SaveConfig`/`fleet config save` until the owner explicitly authorizes the source-write surface.
**Architecture-compatibility review:** 2026-11-11, Gemini 3.8 Flash High (provisional under the fallback rule) — *Approve with conditions* H1–H2, M1–M4, L1–L3; all reconciled in this spec and `plan.md` §5.1. A final Opus review is still required before treating the architecture verdict as final.
**Real-home apply / private-source modification:** Not authorized.
**Hosted model/API/paid usage:** Not authorized.
**Trusted CI / release / tag / tap publication:** Not authorized.
**014 approval/batch history:** Preserved; not superseded or reopened. Live apply/restore remains behind the 014 gate and the reviewed build capability.
**Next SDD action:** Owner review of the Wave 3 completion packet (`w3-acceptance.md`) and B4 decision; preview-only / no-go for live release until 014 acceptance.

### Review reconciliation (2026-11-12)

Gemini provisional code/architecture reviews and the verification review identified the following items:
- **M-01 (Missing `--fleet-config` flag):** Fixed in code and docs (`cmd/agentic-sdd/tui.go`, companion launcher, `TUIConfig.FleetConfig`).
- **M-02 (Top-level metadata loss):** Fixed in code and docs; the editor now reads the selected source and preserves top-level `fleet`/`preset`. Arbitrary key order/formatting and full raw-text editing remain B4 concerns.
- **L-01 (Wide detail focus):** Fixed in code and docs (110-col list+detail pane focus and bounds).
- **L-02 (Acceptance criteria overclaims):** Fixed in docs (`w3-acceptance.md` marks AC-15 and AC-20 as partially covered and AC-33 as partial).
- **L-03 (Model-clear limitation):** Documented limitation; clearing an existing model override back to canonical inheritance is a CLI/editor operation in v1 (`docs/fleet-manager-tui.md`).
- **A3-1 (Inline confirmations):** Fixed in docs; Fleet apply/restore confirmations are inline subview confirmations with the same default-No, focus-trap and tiny-geometry guards, while `modal.go` remains the compositor for global install/restore/help. Do not claim modal reuse that does not exist.

## 15. Provenance

- **Owner-provided:** `docs/handovers/2026-11-11_014b.md`; the two advisory documents in this directory.
- **Repository evidence:** checkout `d5eaa7a`; `experiments/tui/{app,app_screens,ops_host,viewport,modal,theme,config}.go`; `internal/appops/{ops,requests,routing}.go`; `internal/fleet/*.go`; `cmd/agentic-sdd/fleet*.go`; `specs/013-unified-tui-experience/{spec,plan,tasks}.md`; `specs/014-agent-fleet-model-overrides/{spec,plan,tasks}.md`; root/nested `go.mod`.
- **External patterns (advisory only):** Bubble Tea v2 / Bubbles v2 / Huh v2 docs, Lazygit, K9s, chezmoi, Vercel CLI, OpenCode agent/model docs. Design patterns are not evidence of this repository's safety, performance or acceptance.
- **Boundary:** this is a proposed canonical document. It does not validate runtime behavior, approve 014, or authorize implementation.
