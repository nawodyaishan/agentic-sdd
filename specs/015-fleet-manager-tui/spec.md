# Feature 015 — Fleet Manager TUI

**Status:** APPROVED r1 (2026-11-12) — combined spec/plan/tasks approval in `## Approval`; Wave 1 authorized; B4 source-write remains separately gated.
**Amendment status:** **MCP Integration Amendment r1 (2026-11-12) — MCP-A–MCP-D IMPLEMENTED; AWAITING HUMAN REVIEW.** MCP-A (T26–T30) was completed first; the owner then authorized **MCP-B, MCP-C and MCP-D**, which are now implemented and integration-green. MCP-B's configuration write surface was exercised only against temporary-home fixtures. T46 (owner walkthrough/macOS) remains pending. No package installation, network/OAuth call, real-home change, real-home apply or release is authorized.
**Location:** `specs/015-fleet-manager-tui/spec.md` (this directory; input drafts retained beside it).
**Prepared:** 2026-11-11; MCP amendment prepared 2026-11-12.
**Depends on:** Feature 013 Unified TUI Experience (accepted, published `v0.8.0`), Feature 014 Agent Fleet Model Overrides (implemented scope **awaiting human/Opus acceptance**; batch ledger unchanged), Feature 009 OpenCode orchestration (accepted behavior and permissions), `internal/appops` (Feature 013 shared operation seam).
**Target:** Paired `agentic-sdd` / `agentic-sdd-tui` distribution, root Go 1.23 + nested TUI Go 1.26/Charm v2, supported OpenCode V2 configurations only.
**Inputs:** `015-fleet-manager-tui-experience-r2-ux-optimized.md` and `015-fleet-manager-tui-ux-review-2026-11-11.md` in this directory. Both are advisory; this document is the canonical proposal. The MCP amendment also uses `docs/mcp-default-integration-study.md` and `docs/codegraph.md`; vendor documentation was checked on 2026-11-12 and is advisory.

> **Approval boundary.** Drafting and review are read-only. This document does not authorize code changes, a new batch, real-home application, private Git mutation, model/network calls, paid inference, trusted-CI adoption, tag/release/tap publication, package installation, OAuth login, or overriding the Feature 014 gate. The only approval that can activate work is the single combined `spec/plan/tasks` record under `## Approval`, including the separate MCP amendment approval. The MCP amendment text in this file is a proposal, not authority.

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

**FR-23 — MCP discovery and truthful status (P0).** Add a read-only MCP surface for the default set — CodeGraph, Context7 and Exa — inside the existing TUI and CLI. Ordinary `status`, `preview` and offline `doctor` operations perform no write, no model call and no remote contact. They may perform a bounded local executable lookup (`exec.LookPath`) but do not execute `opencode`, `codegraph`, a package manager or a vendor installer. Version evidence for a write is collected only inside the explicit apply path; connection/handshake execution is deferred to a separately approved capability. Report only evidence-backed states: `not configured`, `configured`, `disabled`, `missing prerequisite`, `authentication not checked` / `needs authentication` (only from local runtime evidence), `conflict`, `schema mismatch`, `unsafe target`, `unknown / not checked` and `verified connected` (only from an explicit connection check). A configured remote server is never labeled connected, authenticated or working without verification. The status projection is shared by CLI and TUI; screens never parse CLI prose.

**FR-24 — MCP configuration ownership and scope (P1).** The initial supported target is the global OpenCode V2 configuration (`~/.config/opencode/opencode.json(c)`). v1 requires an existing regular target file; when neither `.json` nor `.jsonc` exists it reports `not configured` and shows a manual snippet instead of creating one. Validate `mcp.servers` and preserve every unrelated top-level field, unknown field, unmanaged server and user-disabled server. Detect both `.json` and `.jsonc`; when both exist, block rather than guess. Never silently migrate V1 `mcp.<name>`/`enabled` configuration. Reject unmanaged name collisions. Do not add MCP keys to `fleet.json`, reuse Feature 014's personal model manifest, rewrite global `AGENTS.md`, or create a second fallback/delegation or permission registry. Project-local scope is deferred until precedence, conflict and restoration behavior are fully specified.

**FR-25 — MCP preview, apply and restore transaction (P1).** Provide a shared preview that shows the exact target file, selected servers, entries to add/change/leave unchanged, conflicts, authentication implications, local-binary requirements, backup and restoration effects. A separate default-No confirmation is required before apply. Apply requires an exact configuration digest, performs a bounded local OpenCode version/schema probe using `--opencode-bin` when supplied or one resolved local executable (no shell, no network, no model), takes a shared engine lock covering both the target and the ownership manifest, revalidates the file under that lock immediately before writing, creates a private recoverable backup of the managed-entry ranges plus metadata and a separate MCP ownership manifest, writes only the selected managed entries, and reports the real result, no-op, conflict or rollback. The `OpsHost` single slot is process-local; it is not a substitute for the cross-process engine lock. A missing or unsupported runtime fails closed. Restore (and any future uninstall) removes or reverts only entries and files Agentic-SDD actually manages and must never erase later user edits. The CLI and TUI share one validation, planning, application and restoration implementation.

**FR-26 — JSONC and unsafe-edit handling.** Handle JSONC conservatively. Preserve comments, key order and unrelated formatting where a lossless byte-range edit is supported; "in-place" means a byte-range transformation followed by atomic replacement, not an overwrite of the live file. If preservation cannot be proven, refuse and show an exact manual snippet rather than reserializing the file. Reject malformed JSON/JSONC, invalid server entries, wrong schema, duplicate keys, symlinks, special files, traversal, ambiguous targets and changed digests. Do not add a JSONC dependency merely to reformat a user file. Never write a plaintext API key or OAuth token into the configuration; use `{env:NAME}` placeholders and delegate OAuth to OpenCode. A managed entry that contains a literal credential is refused until the user moves it to a placeholder; this check runs before any diff or error text is rendered, and the matched value is never echoed. The private backup stores only the managed-entry text and metadata, not unrelated servers or their credentials; it is `0700`/`0600`, never repo-local, never printed, and redacted in reports.

**FR-27 — CodeGraph lifecycle.** Detect the installed `codegraph` CLI, the configured MCP entry, the project `.codegraph/` index and index freshness only to the extent actually checked. Distinguish all five facts: CLI installed, MCP entry configured, project index initialized, index current/stale, and server connection tested/not tested. `codegraph status` is not proof that the MCP server is connected. Do not call `codegraph install` as the default config writer because it can also modify agent instructions and other configuration. Do not automatically run `codegraph init -y` during global MCP setup. Project index initialization is a separate, explicitly scoped action and must never target the home directory, filesystem root, `backups/`, private vaults or unrelated repositories. Respect the repository's CodeGraph discovery and synchronization rules.

**FR-28 — Context7 and Exa authentication honesty.** Prefer the documented hosted Context7 endpoint. Recommend OAuth as the default Context7 recipe only when the target OpenCode version supports it; the user must explicitly accept that recipe in the configuration plan. Document the API-key route as a separate alternative using an environment placeholder. Do not perform OAuth login or persist a preference. Prefer the hosted Exa endpoint and make keyless the default only after the user explicitly acknowledges the visible rate-limit and privacy limitations; make OAuth or API-key authentication optional and user-controlled. Do not automatically upgrade Exa to an authenticated/paid route, and do not treat keyless availability as unlimited free usage. Do not invoke `ctx7 setup`, because it can install additional rules and skills. Configuration is not authentication and authentication is not actual tool availability. Availability checks must not transmit private source code, credentials or unrelated repository contents.

**FR-29 — MCP TUI experience.** Extend the existing four-route application; do not add a fifth global route. Place **MCP Tools** inside **Install & Manage** alongside Install, Backups and Fleet Manager, appended after Fleet Manager so existing hub indices remain stable. Provide an MCP Overview for the three default servers, a server-selection step, an exact configuration preview, a separate default-No confirmation, and diagnostics/recovery. Choosing a server, reviewing configuration and applying a change are separate actions. Connection testing is not a TUI action in v1: the Diagnostics view shows `connection not tested` and CLI guidance. A future TUI connection test requires its own explicit capability and confirmation. Reuse the `agentic-dark` theme, existing key bindings, modal/focus behavior, responsive breakpoints, plain/ASCII/reduced-motion modes and Help/Activity experience. No mandatory animation, onboarding wizard or new design system. Large diffs are scrollable, searchable where practical and readable on narrow terminals.

**FR-30 — MCP CLI parity, compatibility and shared host.** Add an isolated `agentic-sdd mcp` command family with read-only `status`, offline-default `doctor`, `preview`, `apply` and `restore`/`restore preview`. A network or connection-enabled doctor is a distinct, explicitly requested CLI operation and is **deferred** out of MCP-A; the TUI exposes no active network/connection test in v1. Preserve every existing `preview`, `apply`, `backups`, `restore`, `guide`, `tui`, `fleet`, `opencode` and `help` contract. `--with-opencode` never writes MCP configuration merely because an agent pack was installed. A future `--with-mcp` convenience flag is optional and must not be introduced unless it preserves every existing CLI compatibility contract; it is not part of this amendment's first implementation. The TUI never directly parses or rewrites OpenCode configuration, executes shell commands, writes backups or manifests, performs model inference, contacts Context7/Exa, runs CodeGraph indexing or installs packages. Any future explicitly confirmed connection test would be a typed `appops` request, not a TUI network call; it is not part of v1. Domain operations live outside `Update()`/`View()`, use typed requests/results, and route through `internal/appops` and the existing single-write `OpsHost` slot.

**FR-31 — Deferred installation and launch (P2).** Defer all missing-software installation and vendor setup from this amendment. The MCP screens may display missing prerequisites and supported installation guidance, but must not execute package managers, run vendor setup scripts, download or execute arbitrary scripts, install Node or npm packages silently, rewrite `AGENTS.md`, automatically initialize CodeGraph in a repository, or perform OAuth/provider login on the user's behalf. `agentic-sdd mcp install-missing` and `agentic-sdd opencode launch --free` are future separately approved capabilities, not part of MCP-A–MCP-D.

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
| NFR-13 | MCP `status`, `preview` and ordinary `doctor` are zero-write and offline. They invoke no model, package manager, vendor installer or remote MCP endpoint. A network or connection-enabled doctor is explicit, bounded by a timeout and separate from the default path. |
| NFR-14 | MCP configuration uses `{env:NAME}` placeholders; Agentic-SDD never writes a plaintext API key or OAuth token into it, and OAuth is delegated to OpenCode. A managed entry with a literal credential is refused. Backups store only managed-entry text/metadata, are `0700`/`0600`, never repo-local, and credentials are never printed in UI/logs/manifests/fixtures. |
| NFR-15 | Static configuration checks, local probes and runtime connectivity are labeled separately. Green static parsing never becomes a `connected`, `authenticated` or `tool available` claim. |
| NFR-16 | MCP Overview/Preview/Diagnostics meet the existing 110/80/60-column and 8/10/16-height breakpoints, preserve focus/scroll/state across drill-downs, and keep large diffs bounded and reachable. A resize cannot confirm a write. |
| NFR-17 | No new root Go dependency and no new root package are introduced for MCP. `internal/skillsync`, `internal/appops`, `cmd/agentic-sdd` and the nested TUI remain the integration points; the 10-package supported-core gate stays intact unless a separately approved package decision changes it. |

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

### F. MCP integration amendment (pending approval)

These criteria are appended without renumbering AC-01–AC-38. They apply only if the separate MCP amendment approval is recorded. Network, OAuth, package installation and real-home checks remain `not_run` unless separately authorized.

- **AC-51 — MCP Overview coverage.** The MCP Overview displays exactly the three default servers — CodeGraph, Context7 and Exa — with name, purpose, local/remote type, configuration status, prerequisite, verification state and next recommended action. It remains reachable inside Install & Manage and does not create a fifth global route.
- **AC-52 — Read-only, offline status and preview.** Ordinary `mcp status`, `mcp preview` and offline `mcp doctor` create, modify and delete no file; invoke no model, package manager, vendor installer or remote MCP endpoint; and report `not checked` where evidence was not collected.
- **AC-53 — Existing CLI compatibility.** Every pre-existing `preview`, `apply`, `backups`, `restore`, `guide`, `tui`, `fleet`, `opencode` and `help` invocation keeps its existing grammar, text output and exit behavior except additive `mcp` help/usage text. Golden and parser tests cover the unchanged paths.
- **AC-54 — Fleet Manager regression.** Fleet Manager remains reachable from the hub by cursor/Enter and `f`; its Overview, Profiles, Review Changes, Recovery, doctor, live gate and single-write behavior remain unchanged apart from updating hub entry-count/navigation tests for the fourth child.
- **AC-55 — `--with-opencode` does not write MCP configuration.** A `preview` or `apply` with `--with-opencode` leaves `opencode.json`/`opencode.jsonc` byte-identical and creates no MCP backup, manifest or entry, even when an agent pack is installed.
- **AC-56 — Missing CodeGraph is reported, not installed.** With no `codegraph` binary available, status reports `missing prerequisite` and supported guidance; no package manager, download, vendor installer or network request runs, and no binary is installed.
- **AC-57 — Missing CodeGraph index is not silently initialized.** With an existing repository checkout but no `.codegraph/`, status/doctor reports `index not initialized`; no `codegraph init` or `codegraph init -y` runs. Project initialization remains a separate, explicitly scoped action with home/root/backups/vaults guards.
- **AC-58 — No invented MCP authentication.** Context7 and Exa are never shown as authenticated, connected or tool-available from configuration alone. `needs authentication` appears only from local OpenCode/connection evidence; otherwise the label is `authentication not checked` or `unknown`.
- **AC-59 — Unmanaged MCP server preservation.** An unrelated `mcp.servers` entry, unknown top-level field, unknown field inside an unrelated entry and an existing `disabled: true` server survive preview and apply unchanged; a disabled server is never re-enabled. The diff and manifest identify only Agentic-SDD-managed names.
- **AC-60 — Conflict refusal.** An existing unmanaged server with a managed name but different content is a conflict. Preview reports it; apply refuses without overwriting; the file digest is unchanged.
- **AC-61 — JSON/JSONC ambiguity and schema mismatch.** When both `.json` and `.jsonc` targets exist for the selected scope, status/preview report an ambiguity and apply refuses. The tool never guesses, merges or creates the missing alternative. A V1 `mcp.<name>`/`enabled` shape, unknown schema or invalid `mcp.servers` entry is `schema mismatch`/`invalid`, never silently migrated.
- **AC-62 — JSONC comments preserved or refused.** A JSONC file with comments is either edited losslessly with comments/formatting preserved, or refused with an exact manual snippet. Comments are never silently discarded by JSON reserialization.
- **AC-63 — Stale digest, lock and runtime evidence block apply.** Apply requires the exact preview digest and a successful bounded local OpenCode version/schema probe bound into the token. Any external edit, schema change, target change, selected-server change or evidence mismatch after preview makes apply refuse with no write and a replan/reload instruction. A shared engine lock covers the target and ownership manifest; a competing CLI/TUI writer is refused. A missing, older or unsupported OpenCode runtime/schema is reported `unsupported`/`unknown` and blocks apply rather than being rewritten.
- **AC-64 — Unsafe or missing target rejection.** Symlinked configuration, non-regular files, traversal paths, paths outside the selected global scope, changed modes and unsupported special files are rejected without reading or writing outside the intended target. A missing global target is reported `not configured` with a manual snippet; v1 does not create it.
- **AC-65 — Accurate recoverable backup.** A successful apply creates a private backup containing the original managed-entry text/range, digest, mode and target identity plus a separate MCP ownership manifest; it does not copy unrelated servers or their credentials. Restore preview can identify the exact recoverable entries without inferring success from a filename.
- **AC-66 — Failure and rollback honesty.** A simulated failure before or during the multi-step write either restores the original managed-entry bytes exactly or reports what changed, what stayed unchanged, the transaction-intent phase, the backup identity and the exact recovery action. Structural ambiguity (created containers, comment/delimiter ownership, stale ranges) is `needs recovery`, never a best-effort rewrite. No partial write is labeled successful.
- **AC-67 — Idempotent no-op.** Applying an identical selection and configuration twice reports `up to date` / no-op on the second run, creates no second backup and leaves the target bytes unchanged.
- **AC-68 — Restore preserves later user edits.** Restore and any supported uninstall path change only managed entries whose current content still matches the recorded installed digest. Later edits to unrelated top-level fields, unrelated MCP servers or a managed entry are preserved or refused with a conflict, never erased.
- **AC-69 — Single-write host.** MCP apply/restore and the existing install/restore and fleet apply/restore write kinds (plus the source-save seam when enabled) share the one `OpsHost` slot. Tests cover both directions, explicit busy refusal and no queued second write. Because `OpsHost` is process-local, a separate shared engine lock covers cross-process CLI/TUI writers.
- **AC-70 — Navigation continuity.** Server selection, preview, confirmation, diagnostics and recovery retain focus, scroll, selection and valid draft state across overlay open/close, Back/Esc and parent-route round trips.
- **AC-71 — Resize cannot confirm.** Resizing from a wide layout to a narrow or short layout during MCP confirmation keeps `No` selected, keeps Cancel/Back reachable and cannot dispatch a write.
- **AC-72 — Script compatibility.** Non-interactive `mcp status`, `mcp preview`, `mcp apply` and `mcp restore` do not enter the alternate screen, do not prompt, emit stable text or `--json`, use documented exit codes and require no terminal.
- **AC-73 — Offline test safety.** Offline tests invoke no default model, no remote MCP endpoint, no OAuth flow and no package installation. Fake `codegraph`/`opencode` executables and temporary homes are the only probes; `doctor --network`/`--connect` are recorded as `not_run`.
- **AC-74 — Permission and role preservation.** MCP configuration does not change canonical agent roles, Feature 009 permissions, fallback/delegation policy or Feature 014 model/fleet state. A hash/snapshot of `opencode-pack` and installed agent profiles is unchanged by MCP apply/restore.
- **AC-75 — Static versus runtime evidence.** Test reports and UI labels distinguish static config parsing, local executable/config probes and actual runtime MCP connectivity. A static fixture never supports a `verified connected` claim, and a runtime check that was not run is `not_run`, not `pass`.

### 8.1 Disposition of the r2 supplemental criteria

The advisory r2 draft added AC-39–AC-50. They are not dropped; each is enforced by a canonical criterion above. Those identifiers remain historical advisory IDs and are not reused by the MCP amendment, whose criteria begin at AC-51:

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

### 9.1 MCP amendment negative and failure fixtures (pending approval)

The plan must allocate deterministic fixtures for at least:

31. Malformed JSON/JSONC and duplicate keys: parse error with redacted location; zero writes.
32. Invalid `mcp.servers` entry, unknown `type`, wrong schema or V1 `mcp.<name>` shape: blocked with migration guidance; no silent conversion.
33. Managed and unmanaged servers share a name with different content: conflict; file untouched.
34. Both `opencode.json` and `opencode.jsonc` exist: ambiguity; no target selected, no write.
35. `codegraph` binary absent: `missing prerequisite`; no install, download or network.
36. Existing `disabled: true` server: preserved as disabled; never re-enabled by a merge.
37. Injected failure after temp write or rename: original restored or exact rollback/recovery evidence; no false success.
38. Source/config change or stale preview token between preview and apply: `ErrStaleConfig`/`ErrNeedsPreview`; no write.
39. Missing, older or unsupported OpenCode runtime version: static `unsupported`/`unknown` report; no schema rewrite and no runtime claim.
40. JSONC comments/formatting in the edit region: lossless edit or manual-snippet refusal; no comment loss.
41. Symlinked config, traversal path or special file: rejection before read/write; no path escape.
42. Restore after a later user edit to an unrelated field or managed entry, or after deletion of a managed entry: preserve/refuse; no erasure and no silent re-add.
43. CodeGraph index absent or stale: `index not initialized` or `freshness not checked`; no automatic init, index or sync.
44. Context7 OAuth pending or Exa keyless rate limit: `authentication not checked`/`rate-limited (not verified)` from evidence only; no invented success and no credential handling.
45. Managed entry contains a literal API key/OAuth token instead of `{env:NAME}`: `blocked: literal credential`; no write and no credential copied into backup/manifest.
46. No global `opencode.json(c)` exists: `not configured` plus a manual snippet; apply refuses and creates no file or directory.

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
| MCP static config | golden JSON/JSONC fixtures, UI status/preview, diff and token tests | detection/merge/ownership logic and offline behavior | remote server connection, OAuth or tool availability |
| MCP local probe | fake `codegraph`/`opencode` executables, explicit binary paths, temporary homes | local prerequisite and version reporting | package installation or a real user installation |
| MCP runtime (optional) | explicit `mcp doctor --network`/`--connect` against an authorized endpoint | one observed connection result | general availability, authentication entitlement or release approval |
| Human review | owner end-to-end walkthrough + accessibility fallback | acceptance of the documented experience | statistical novice study |

**Restrictions:** no real `~/.config/opencode`, private Git checkout, network, model calls or paid provider use. Isolated roots, fixture JSON and fake runtime evidence only. Root tests use `GOPROXY=off`/`GOWORK=off` and the approved package gate; nested Charm deps stay in their module. Missing Linux/macOS/manual checks are `not_run`, never `pass`. MCP tests never install CodeGraph, Context7, Exa, Node or npm packages and never invoke OAuth.

## 11. Delivery grouping (not batch authorization)

| Group | Outcome | Depends on |
|---|---|---|
| D1 — Read-only Fleet entry | Third hub child, Overview, Profiles/search/detail, typed status, responsive tests | Accepted 013 shell; additive status projection |
| D2 — Staging and review | Model picker, enablement staging, dirty protection, source diff, impact-first review | D1; verified schema/renderer |
| D3 — Deployment | Fresh plan, full diff, default-No apply through OpsHost, result/backup | D2; 014 eligible; plan/diff projection |
| D4 — Source save | Native save operation + CLI parity + CAS/backup tests (separately gated) | D2; approval of the new write surface |
| D5 — Recovery | Restore preview/restore, drift/legacy/conflict guidance, doctor | D1, D3; restore preview |
| D6 — Robustness and docs | PTY/resize/accessibility/large-fleet, paired artifact, owner walkthrough, docs | D1–D5 |
| D7 — MCP read-only foundation (MCP-A) | Metadata, target/config detection, offline status/doctor, shared state vocabulary, CLI parity | Amendment approval; no write authorization |
| D8 — MCP safe configuration (MCP-B) | Preview/digest, ownership manifest, backup, transactional apply/restore, JSONC/conflict handling | D7; separate MCP write-surface authorization |
| D9 — MCP TUI experience (MCP-C) | Hub child, Overview, selection, preview, confirmation, diagnostics/recovery, single-write host | D7; D8 for write screens |
| D10 — MCP verification and closeout (MCP-D) | Root/nested tests, PTY/headless evidence, no-network/temp-home safety, docs, acceptance packet | D7–D9; owner review |

If the 014 gate remains blocked, D3/D5 land in fixture/preview mode only; no production apply path is enabled to complete the TUI. D7–D10 are independent of Fleet live enablement but remain `pending approval`; D8 requires its own write-surface decision and is not B4.

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
- **MCP P2 installation is deferred:** no package manager, vendor setup script, `codegraph install`, `ctx7 setup`, Node/npm install, automatic `codegraph init -y`, OAuth/provider login or `AGENTS.md` rewrite.
- **MCP project-local scope is deferred** until its precedence, conflict, ownership and restoration behavior is fully specified; v1 manages the global configuration only.
- **MCP uninstall and `agentic-sdd opencode launch --free` are deferred** to separately approved capabilities. The amendment's restore/rollback work does not authorize a general uninstaller or an agent launcher.
- **`--with-mcp` is not introduced** by this amendment. If proposed later, it must preserve every existing CLI compatibility contract and must never install missing software.
- **MCP permissions are not configured here.** Feature 009 remains the sole permission/delegation authority; MCP configuration only makes a server available to OpenCode.

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
| MCP edit becomes a second config manager | One owned target; semantic merge; separate manifest; CLI/TUI share `appops`/engine; no vendor installer; no `fleet.json` extension. |
| JSONC reserialization destroys comments | Conservative lossless edit or refusal with an exact manual snippet; no reformat dependency; golden JSONC fixtures. |
| False MCP connectivity/authentication | Static config, local probe and runtime evidence labeled separately; `connected` only from an explicit check; `not_run` stays `not_run`. |
| Unmanaged server overwritten | Name-collision conflict; exact digest and ownership manifest; restore only matching managed entries. |
| CodeGraph init/indexes an unsafe path | Project initialization is separate and explicit; home/root/backups/vaults rejection; no automatic `init -y` in global setup. |
| MCP write bypasses the single-write slot | `WriteMCPApply`/`WriteMCPRestore` are the only TUI path; one `OpsHost` slot; pairwise busy-refusal tests. |
| MCP feature accidentally installs software | P2 excluded; status/doctor only report; no package manager/vendor installer; fake binaries in tests. |
| Amendment is read as Fleet/014 approval | Separate pending MCP approval; B4 and Fleet live gates untouched; release authorization remains separate. |

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
**Next SDD action:** Implement and verify **MCP-A only** (T26–T30), then return it for human review. MCP-B–MCP-D remain deferred. Wave 3 acceptance, the B4 decision and Feature 014 acceptance remain separate; preview-only / no-go for live release until 014 acceptance. MCP-A does not resolve any of those gates.

### Review reconciliation (2026-11-12)

Gemini provisional code/architecture reviews and the verification review identified the following items:
- **M-01 (Missing `--fleet-config` flag):** Fixed in code and docs (`cmd/agentic-sdd/tui.go`, companion launcher, `TUIConfig.FleetConfig`).
- **M-02 (Top-level metadata loss):** Fixed in code and docs; the editor now reads the selected source and preserves top-level `fleet`/`preset`. Arbitrary key order/formatting and full raw-text editing remain B4 concerns.
- **L-01 (Wide detail focus):** Fixed in code and docs (110-col list+detail pane focus and bounds).
- **L-02 (Acceptance criteria overclaims):** Fixed in docs (`w3-acceptance.md` marks AC-15 and AC-20 as partially covered and AC-33 as partial).
- **L-03 (Model-clear limitation):** Documented limitation; clearing an existing model override back to canonical inheritance is a CLI/editor operation in v1 (`docs/fleet-manager-tui.md`).
- **A3-1 (Inline confirmations):** Fixed in docs; Fleet apply/restore confirmations are inline subview confirmations with the same default-No, focus-trap and tiny-geometry guards, while `modal.go` remains the compositor for global install/restore/help. Do not claim modal reuse that does not exist.

### MCP Integration Amendment — APPROVED THROUGH MCP-D (2026-11-12)

**State:** **APPROVED — MCP-A–MCP-D implemented; implementation accepted; `v0.9.0` release authorized.** This is the amendment approval record for §16, FR-23–FR-31, NFR-13–NFR-17 and AC-51–AC-75.

**Implementation acceptance and release authorization (2026-11-13):** Owner message: “spec implementation approved , commit and push , and add new tag and release and check for ci releases to pass and clean handover dir”. This accepts the integrated MCP-A–D implementation, authorizes committing/pushing the current tree, creating the next annotated tag (`v0.9.0`), publishing the matching GitHub release and letting the release workflow publish the Homebrew cask, monitoring CI/release results, and folding/removing obsolete handovers. It does **not** accept Feature 014, accept Wave 3, authorize B4, enable `FleetCapability=live`, authorize a real-home apply, install packages, or authorize any unrelated live operation.

**Release published (2026-11-13):** main CI run `43375024192` **passed**; Release workflow `43375046768` for tag `v0.9.0` **passed**. GitHub Release `v0.9.0` is published with `agentic-sdd_0.9.0_darwin_all_generic.tar.gz`, `agentic-sdd_0.9.0_darwin_all_homebrew.tar.gz` and `checksums.txt`; the Homebrew tap cask is version `0.9.0` with SHA256 `7cd56a8e146e79a53ec77c0325f90ea594250813c3d87caf55416cb724f3c270`. T46 owner walkthrough/macOS remains `not_run`.

**Approval evidence:**
1. Owner message on 2026-11-12: “accpted all” followed by “Start implementation parally using sub agents of 9r gemini flash , deepseek free , mimo free , deepseek high”. This authorized the reviewer-recommended **MCP-A (T26–T30)** scope.
2. Owner message on 2026-11-12: “commit current changes and start mcp b c d in parallelly if possible using implement agents utilizing maximum num , also use deepseek high, free models and gemini flash high as models”. This explicitly authorizes **MCP-B, MCP-C and MCP-D**, including MCP-B's MCP configuration write surface for development, fixture tests and CLI/TUI implementation, and authorizes committing the current changes.

**Authorized scope:**
1. **MCP-A (T26–T30):** implemented; `awaiting human review` at CP-A.
2. **MCP-B (T31–T37):** authorized. The write surface may be implemented and exercised only against isolated temporary-home fixtures. No real-home OpenCode configuration may be modified.
3. **MCP-C (T38–T42):** authorized, including the TUI fourth child, preview/confirmation/diagnostics/recovery and `OpsHost` MCP write kinds.
4. **MCP-D (T43–T46):** authorized for verification/closeout.
5. **Commit:** authorized for the current MCP-A code/tests and Feature 015 documentation.

**What this approval still does not do:** It does not accept Feature 014, accept Wave 3, authorize B4, enable `FleetCapability=live`, authorize a real-home write, install CodeGraph/Context7/Exa/Node/npm, start OAuth/login, invoke a model, or authorize a tag/cask/tap/release.

**Next action:** monitor the `v0.9.0` tag-triggered release workflow and record its outcome. T46 owner walkthrough/macOS remains `not_run`; no real-home apply. Feature 014, Wave 3, B4 and Fleet live remain separate gates.

## 15. Provenance

- **Owner-provided:** `docs/handovers/2026-11-11_014b.md`; the two advisory documents in this directory.
- **Repository evidence:** checkout `d5eaa7a`; `experiments/tui/{app,app_screens,ops_host,viewport,modal,theme,config}.go`; `internal/appops/{ops,requests,routing}.go`; `internal/fleet/*.go`; `cmd/agentic-sdd/fleet*.go`; `specs/013-unified-tui-experience/{spec,plan,tasks}.md`; `specs/014-agent-fleet-model-overrides/{spec,plan,tasks}.md`; root/nested `go.mod`.
- **External patterns (advisory only):** Bubble Tea v2 / Bubbles v2 / Huh v2 docs, Lazygit, K9s, chezmoi, Vercel CLI, OpenCode agent/model docs. Design patterns are not evidence of this repository's safety, performance or acceptance.
- **Boundary:** this is a proposed canonical document. It does not validate runtime behavior, approve 014, or authorize implementation.

## 16. MCP integration amendment (proposed 2026-11-12 — pending owner approval)

> **Amendment boundary.** This section is a proposal for the owner's review. It authorizes no code, configuration write, package installation, network call, OAuth flow, model invocation, real-home change, commit, tag or release. It does not alter accepted Feature 015 Waves 1–2, the implemented-but-unreviewed Wave 3, the Feature 014 acceptance gate, the B4 source-save gate or the Fleet live-capability gate. The amendment approval is recorded separately under `## Approval` only after the owner explicitly accepts it.

### 16.1 Objective and staged decision

Make the default Agentic-SDD MCP tool set — **CodeGraph** (local code intelligence), **Context7** (hosted library documentation) and **Exa** (hosted web research) — discoverable, configurable, diagnosable and recoverable by beginners without weakening the repository's preview-first, digest-checked, backup-first, single-writer guarantees.

The binding staging is:

| Phase | Scope | Authorization |
|---|---|---|
| **P0 — Read-only foundation** | Detect and report the three servers; shared state vocabulary; offline `status`/`doctor`; a read-only `preview` that shows the current recipe/static comparison and writes nothing, with **no apply token**. This is the default entry point. | MCP-A, authorized and implemented. No write authorization. |
| **P1 — Opt-in configuration and restoration** | Explicit server selection, exact preview, separate default-No confirmation, digest-checked apply, private backup/ownership manifest, managed restore. Global OpenCode configuration only. | MCP-B/MCP-C, authorized and implemented; write surface exercised on fixtures only. |
| **P2 — Missing-software installation** | Package managers, vendor installers, Node/npm, OAuth login, automatic CodeGraph init. | **Deferred.** Not part of this amendment. A future feature must provide version pinning, explicit argv review, consent, timeout and supply-chain controls. |

`agentic-sdd opencode launch --free` is likewise deferred to a separate optional capability. It is not bundled into MCP configuration work.

### 16.2 Baseline reconciliation (2026-11-12)

The pre-amendment baseline was clean on `main` at `4c2b9bd` (`docs: Document CodeGraph workflows`), whose parent is `74f568b` (`feat(fleet): add fleet manager TUI`). The latest handover's statement that the CodeGraph default-rule changes were uncommitted is **stale**: `AGENTS.md`, `docs/codegraph.md`, `docs/orchestration-guide.md`, the portable router policy and the docs index were committed in `4c2b9bd`. No code changed between `74f568b` and the pre-amendment baseline `4c2b9bd`.

Reconciled state:

- Feature 015 Waves 1–2 are owner-accepted; Wave 3 is implemented and pushed as `74f568b`, **awaiting human review**. `w3-acceptance.md` retains its historical "uncommitted tree" baseline; this amendment does not rewrite that history.
- Feature 014 remains **awaiting its human/Opus acceptance**; its batch ledger is unchanged. This amendment does not accept, reopen or apply Feature 014.
- B4 Fleet native source save remains **not authorized** (`T14`–`T17` unchecked). MCP work does not repurpose B4 or its reserved IDs.
- Fleet live apply/restore remains gated by the reviewed `FleetCapability` build value, default `preview`. MCP work does not change it.
- MCP-A–MCP-D are **implemented and awaiting human review**; no further MCP batch is authorized without a new decision.
- `docs/mcp-default-integration-study.md` recommends a broader P2 installation and an OpenCode launcher. This amendment **narrows** that study: P0 first, P1 opt-in, P2 and the launcher deferred.

### 16.3 Ownership and configuration precedence

| Component | Authority |
|---|---|
| Existing SDD skills and approvals | Engineering methodology, skill content and approval records |
| Feature 009 | Agent permissions, delegation and fallback policy |
| Feature 014 | Personal models, fleet instances and the `fleet.json` schema |
| MCP integration | MCP server configuration and its managed lifecycle |
| OpenCode | Runtime connection, authentication, MCP tool execution and effective configuration |

Rules:

- Do **not** extend `fleet.json` with MCP server definitions. Do not reuse Feature 014's personal model manifest as the MCP ownership record.
- Do **not** create another fallback/delegation policy or permission registry. Feature 009 remains authoritative.
- Do **not** overwrite global `AGENTS.md`. The MCP integration writes only the selected OpenCode configuration file and its own private backup/manifest area.
- Configuration ownership is separate from runtime ownership: Agentic-SDD owns the managed entry text and lifecycle; OpenCode owns OAuth credentials, connection state, tool catalogs and execution. The private MCP ownership state lives under `<home>/.agentic-sdd/mcp/` (`manifest.json` and `backups/<stamp>/`), outside the selected checkout and the skillsync backup namespace.
- MCP configuration does not grant permissions. Configuring a server makes it available to OpenCode; Feature 009 profiles and OpenCode V2 `permissions`/`policies` still decide whether an agent may use it.
- OpenCode V2 keeps these permission/policy facts separate from MCP configuration: MCP tools use permission action `_` (resource `*`); Code Mode uses `execute`; remote/local server entries can set `codemode: false` to expose tools directly; and policy action `mcp.use` can deny a server by name. Policy-action support is version-dependent; an older runtime is reported `unknown`/`unsupported` and is never edited. The MCP amendment never edits those rules. A `configured` server may still be denied or unused by an agent, and the UI must not conflate configuration with availability.

OpenCode configuration precedence, verified against current V2 vendor documentation:

| Layer | Typical path | MCP consequence |
|---|---|---|
| Global | `~/.config/opencode/opencode.json(c)` | The only target managed in this amendment. |
| Project direct | `<project>/opencode.json(c)` | Higher precedence; a same-name server replaces the entire global server object. v1 does not manage it. |
| Project `.opencode` | `<project>/.opencode/opencode.json(c)` | Higher precedence than direct project config. v1 does not manage it. |
| System / managed | `/etc/opencode/opencode.json`, macOS managed preferences | Highest precedence; can override or deny. Never managed by this amendment. |
| Runtime/OAuth | OpenCode credential store outside the config | Never read or written by Agentic-SDD. |

A global entry is reported as `configured (global)`, not `effective`, because a project or system layer may override it. Both `.json` and `.jsonc` at the selected scope are an ambiguity, not a preference. V2 uses `mcp.servers.<name>` and `disabled`; V1 uses `mcp.<name>` and `enabled`. V1 is detected and refused with migration guidance, never silently converted. The default global path follows OpenCode's documented platform location; macOS variants must be verified at implementation, and `--config` is the explicit override.

### 16.4 Supported default servers

These are the proposed default recipes. Exact endpoint/header/field details are verified at implementation against the pinned software versions; the manifest records the version used.

| Server | Type | Purpose | Proposed entry | Prerequisite | Authentication |
|---|---|---|---|---|---|
| **CodeGraph** | local stdio | Repository code intelligence (`codegraph_explore`, CLI fallback) | `command: ["codegraph","serve","--mcp"]`, `codemode: false` | `codegraph` CLI; a project `.codegraph/` index for actual tools | None |
| **Context7** | remote HTTP | Hosted library documentation (`resolve-library-id`, `query-docs`) | OAuth route `https://mcp.context7.com/mcp/oauth` (recommended default); API-key alternative `https://mcp.context7.com/mcp` with `Authorization: Bearer {env:CONTEXT7_API_KEY}` and `oauth: false` | None | OAuth recommended and explicitly accepted, or API-key placeholder; no automatic login |
| **Exa** | remote HTTP | Hosted web research (`web_search_exa`, `web_fetch_exa`; `agent_run` requires auth) | Keyless `https://mcp.exa.ai/mcp` (default after acknowledgement); optional OAuth `?login` or `x-api-key: {env:EXA_API_KEY}` | None | Keyless (rate-limited) with explicit limits/privacy acknowledgement; optional OAuth or API key; no automatic upgrade |

Proposed CodeGraph entry. Local CLI 1.6.2 was verified to support `codegraph serve --mcp` and `codegraph init -y`; the OpenCode entry shape is the vendor's documented `--print-config opencode` output recorded in the study and must be re-verified before any write:

```json
{
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["codegraph", "serve", "--mcp"],
        "disabled": false,
        "codemode": false
      }
    }
  }
}
```

Context7 and Exa are written as remote entries with only documented fields. The implementation must not copy V1 `mcp.<name>`/`enabled` examples, and must not invent headers. The Context7 OAuth entry is selected only when the user explicitly chooses OAuth; the API-key entry is a separately labeled choice with an environment placeholder. The Exa keyless entry is the default only if the user accepts the visible rate-limit and privacy note; OAuth/API key are optional and user-controlled.

CodeGraph facts that the status surface must keep separate:

1. CLI installed (local lookup).
2. MCP entry configured (config inspection).
3. Project index initialized (`.codegraph/` presence).
4. Index current/stale (only from a separate explicit local probe; ordinary status/offline doctor execute nothing and show `freshness not checked`).
5. MCP server connection tested/not tested (only from an explicit connection check).

`codegraph status` is an index check, not MCP connectivity. `codegraph install` is not a config writer for this feature because it can also write agent instructions/`AGENTS.md`. `codegraph init -y` is not run during global MCP setup; project initialization is a separate, explicitly scoped action that refuses the home directory, filesystem root, `backups/`, private vaults and unrelated repositories. The repository's CodeGraph discovery/synchronization rules in `AGENTS.md` and `docs/codegraph.md` remain binding.

### 16.5 User journeys

**A — Discover (P0, no writes).** Install & Manage → MCP Tools → Overview. The user sees three server rows, the local/remote type, configuration status, prerequisite, verification state and one next action. Opening the screen does not read the network, run a model or write configuration. A beginner can tell “not configured” from “configured but not checked” and from “missing CodeGraph CLI.”

**B — Configure (P1, opt-in).** MCP Tools → Select servers → choose CodeGraph/Context7/Exa. Context7 presents OAuth as the recommended default requiring explicit acceptance; Exa presents keyless as the default requiring acknowledgement of the rate/privacy limits. Each has an explicitly selected API-key alternative where applicable. Then Preview → review the exact target file, entries to add/change/leave, conflicts, auth implications, local-binary requirement and backup/restore effect → default-No Confirm → apply through the shared operation host → see the real result and backup identity. If no global config exists, the journey stops at a manual snippet; v1 does not create one. Selecting `--with-opencode`, installing an agent pack or merely opening Fleet Manager never enables MCP configuration.

**C — Diagnose.** MCP Tools → Diagnostics (`d` or equivalent). The panel explains: CodeGraph missing; CodeGraph index missing/stale; Context7 authentication pending; Exa rate limits; OpenCode schema mismatch; unmanaged MCP server conflict; JSONC formatting conflict; stale plan; failed apply or incomplete rollback. Recovery uses actual transaction metadata. Connection state is shown as `not tested` with CLI guidance; the TUI has no active network/connection test in v1, and any future CLI `--network`/`--connect` remains a separate explicitly approved capability.

**D — Restore.** MCP Tools → Recovery → managed MCP backups → preview exact current-versus-restore effects → default-No restore. Only entries Agentic-SDD manages are considered. Later user edits to unrelated fields, unrelated servers or a managed entry are preserved or reported as a conflict, never erased.

**E — Refuse safely.** JSONC comments, `.json`/`.jsonc` ambiguity, V1 schema, unmanaged collision, symlink/traversal, stale digest, missing auth choice or unsupported runtime produces a specific explanation, the exact safe next action and, where applicable, a copyable manual snippet. No best-effort rewrite and no optimistic success.

### 16.6 Status and evidence vocabulary

| Display state | Evidence | Never implies |
|---|---|---|
| `not configured` | No managed name in the inspected file | Server absent from project/system layers |
| `configured (global)` | Entry exists, not `disabled: true` | Connected, authenticated, effective or tool-available |
| `disabled` | Entry exists with `disabled: true` | Removed; the user's disabled state is preserved |
| `missing prerequisite` | Local binary lookup failed for a selected local server | Permission to install |
| `authentication not checked` | Offline/default path; no local runtime evidence | Authentication failure or success |
| `needs authentication` | Explicit local OpenCode or connection evidence reports it | That every server/tool is unusable |
| `conflict` | Unmanaged name collision, differing managed entry, JSONC ambiguity or unsafe edit | Permission to overwrite |
| `schema mismatch` | V1 `mcp.<name>`, unknown schema or invalid `mcp.servers` | Permission to migrate |
| `unsafe target` | Symlink, traversal, non-regular file, changed mode, out-of-scope path | Permission to follow or repair silently |
| `unknown / not checked` | Offline path did not perform the check | Failure |
| `verified connected` | Explicit bounded connection check produced evidence | Authenticated, entitled, model success or tool correctness |
| `index not initialized` | CodeGraph project has no `.codegraph/` | Permission to run `init` |
| `index freshness not checked` | No explicit `codegraph status`/`sync` run | Index current |
| `stale plan` | Config digest/target/selection changed after preview | Permission to reuse the preview |
| `previously managed, now absent` | Ownership manifest records the name but the current config has no entry | Permission to re-add; explicit confirmation is required, never silent restoration |

Configured entries also carry an ownership axis: `managed` (recorded in the MCP ownership manifest with a matching installed digest) or `unmanaged` (present but not owned by Agentic-SDD). Only managed entries can be restored or removed; unmanaged entries are preserved and a same-name content difference is a conflict. Static config, local probe and runtime connection are separate evidence classes. A report must say which class supports each state.

### 16.7 TUI experience and shared architecture

MCP Tools is a **fourth child of Install & Manage**, reached by key `t` (Tools) and Enter. `m` is not used because it is the existing global motion toggle. The hub's cursor wrap/bounds continue to use `len(entries)`. There is no fifth global destination.

Local MCP views:

| View | Content | Primary action |
|---|---|---|
| **Overview** | Three server rows: name, purpose, type, configuration, prerequisite, verification, next action; last operation and backup pointer | Open selection, diagnostics or recovery |
| **Select servers** | All three defaults, explicit selection, Context7 OAuth/API-key choice, Exa keyless acknowledgement/optional auth, prerequisite warnings | Preview configuration |
| **Preview** | Selected servers, existing configuration, exact target file and base digest, entries add/change/leave, conflicts, auth implications, local-binary requirement, backup/restore effect, scrollable diff | Confirm or Back |
| **Confirm** | Default-No inline confirmation; exact target and digest; separate from selection and review | Apply through `OpsHost` |
| **Diagnostics** | Contextual explanations and next safe actions; connection state `not tested`; CLI guidance only | Reload/replan/CLI guidance |
| **Recovery** | Managed backups, restore preview, default-No restore, conflict/rollback evidence | Restore or Back |

Interaction constraints:

- Reuse `agentic-dark`, existing keys (`j/k`, Enter, Esc, `/`, Tab/Shift+Tab, `?`, Home/End), focus traps, viewport and reduced-motion/plain/ASCII modes. Status text is primary; color/shape cues are secondary, never a decorative badge that replaces the explanation.
- Navigation preserves selection, filter, scroll, view and valid state. A picker/dialog/error is a local subview, not a tab.
- Large diffs are scrollable; `/` filters lines where practical; long paths/URLs are reachable and elide only in list rows.
- At <60 columns or height <8, show resize guidance and no affirmative key. Resize cannot confirm.
- No mandatory animation, onboarding wizard, new design system, fifth route or duplicate Help/Activity experience.
- The MCP panel feels like a natural extension of Fleet Manager, not a separate application.

Dependency direction:

```text
Existing Bubble Tea TUI (experiments/tui)
                 │  typed view/draft state only
                 ▼
          internal/appops
                 │  immutable requests/results, preview tokens
                 ▼
  Existing supported Go core (internal/skillsync MCP config + transaction)
                 │  parse · merge · digest · backup · manifest · restore
                 ▼
        global OpenCode configuration
```

The TUI must not directly parse/rewrite OpenCode configuration, execute shell commands, write backups/manifests, perform model inference, contact Context7/Exa, run CodeGraph indexing or install packages. Domain work stays outside `Update()`/`View()`. Apply/restore route through the existing single `OpsHost` write slot; MCP and install/restore/Fleet writes are mutually exclusive.

### 16.8 CLI surface and compatibility

Proposed grammar (command-name availability verified: `cmd/agentic-sdd/main.go` at `4c2b9bd` has no `mcp` dispatch or help topic; `fleet` is the additive-dispatch precedent):

```text
agentic-sdd mcp status  [--home PATH] [--config PATH] [--project PATH] [--json]
agentic-sdd mcp doctor  [--home PATH] [--config PATH] [--opencode-bin PATH]
                        [--codegraph-bin PATH] [--project PATH] [--json]
                        # --network/--connect: deferred separate capability, not MCP-A
agentic-sdd mcp preview [--servers NAME,...] [--auth context7=oauth|api-key]
                        [--auth exa=keyless|oauth|api-key] [--home PATH] [--config PATH] [--json]
agentic-sdd mcp apply   --expect-config SHA256 [--opencode-bin PATH] [--servers NAME,...]
                        [--auth ...] [--home PATH] [--config PATH] [--json]
agentic-sdd mcp restore list|preview|apply [ID] [--home PATH] [--config PATH] [--json]
agentic-sdd mcp help
```

- **MCP-A implements only `status`, offline `doctor`, read-only `preview` and `help`.** `apply`, `restore` and any write path remain MCP-B and are not present in an A-only build. `agentic-sdd mcp` with no subcommand behaves as read-only `status`; `mcp help` prints usage.
- `status`/`preview`/ordinary `doctor` are offline and zero-write. They may use a local executable lookup, but execute no binary. `doctor --network` (remote reachability) and `doctor --connect` (MCP handshake, which may start a local server) are deferred out of MCP-A to a separately approved capability and are never default.
- In **MCP-A**, `preview` is a read-only current-recipe/static comparison and returns no apply token; it may print a manual snippet. In **MCP-B**, the same command is extended to return the semantic diff, target identity, base/plan digest and token. `apply` requires `--expect-config`; empty is a usage error, mismatch is a stale/conflict error, and neither writes. `apply` also requires a successful bounded local OpenCode version/schema probe bound into the token.
- `restore list|preview|apply` follows the existing restore style: preview first, explicit ID, default-No interactive confirmation, entry-level restore. Whole-file restore is not part of v1 and cannot erase later edits.
- Exit codes follow 014/015: `2` usage/validation/schema/unsafe target, `1` runtime/stale/conflict/rollback, `0` success or no-op. `--json` output is stable and script-compatible.
- Existing CLI grammar/output is unchanged. `--with-opencode` never writes MCP configuration. A future `--with-mcp` is **not introduced** unless separately approved and compatibility-tested.
- The TUI never spawns `agentic-sdd mcp ...` and parses text. Both CLI and TUI call the same `appops` requests.

### 16.9 Exclusions and deferrals

- No P2 installation: package managers, vendor setup scripts, `codegraph install`, `ctx7 setup`, Node/npm installs, downloads, arbitrary scripts, OAuth/provider login.
- No automatic `codegraph init -y` in global setup; no indexing of home, root, `backups/`, private vaults or unrelated repositories.
- No project-local MCP configuration management in v1; no `.json`/`.jsonc` guessing; no silent V1 migration.
- No creation of a missing global config in v1 and no `--create` implementation; a future `--create` requires a separate design with exclusive creation, absent-file identity, parent-directory handling and recovery of an unchanged tool-created file.
- No `--replace`/force flag in v1. Conflicts are always refused; a future replace capability requires its own adoption/credential/restore design.
- No CodeGraph project initialization implementation (`codegraph init`) in this amendment; a future explicit project action must carry path guards, consent and its own approval.
- No `fleet.json` MCP fields; no second permission/fallback registry; no `AGENTS.md` rewrite; no agent profile changes.
- No MCP marketplace, remote catalog, server discovery, ratings or automatic server recommendations.
- No plaintext API keys or OAuth tokens written by Agentic-SDD into configuration, logs, manifests, backups or fixtures. A managed entry with a literal credential is refused; backups store only managed-entry text/metadata. OAuth credentials stay in OpenCode's store.
- No general MCP uninstaller, `mcp install-missing` or `opencode launch --free` in this amendment.
- No live network/MCP handshake, OAuth flow, package install, real-home write or model call in default tests or CI.
- No code, commit, tag, cask/tap, release or real-home operation is authorized by this amendment text.

### 16.10 Dependencies, release gates and workstreams

| Workstream | Scope | Depends on | State |
|---|---|---|---|
| **MCP-A — Read-only foundation** | Metadata; target detection; offline status/doctor; read-only preview (no token); state vocabulary; CLI `status`/`doctor`/`preview`; compatibility tests | Amendment approval; no write authorization | `implemented — awaiting human review` |
| **MCP-B — Safe configuration management** | Structured preview/digest; JSONC refusal/manual snippet; ownership manifest; backup; transactional apply/restore; runtime evidence; cross-process lock; failure tests | MCP-A + explicit MCP write-surface authorization | `implemented — awaiting human review` |
| **MCP-C — TUI experience** | Hub child; Overview; selection; preview; confirmation; diagnostics; recovery; single-write host | MCP-A; MCP-B interfaces | `implemented — awaiting human review` |
| **MCP-D — Verification and closeout** | Root/nested gates; PTY/headless; no-network/temp-home evidence; docs/onboarding; acceptance packet | MCP-A–C; owner review | `awaiting human review — owner walkthrough pending` |

Prerequisites and release gates:

- The amendment approval is separate from Feature 015 r1 approval. It approves the amended text; it does not by itself authorize MCP-B's write surface. The owner must explicitly name that surface, exactly as B4 requires a separate decision.
- MCP-A may be developed and tested independently of the write surface. Deterministic UI/config tests must not require live remote connections, OAuth, package installation or a real OpenCode home.
- Feature 014 acceptance, the Feature 015 Wave 3 review, B4, Fleet live capability and release authorization remain separate. No MCP test result, green build or reviewer verdict resolves them.
- No test result authorizes a real-home modification, package installation, network call, model invocation, OAuth flow, commit, tag or publication.

### 16.11 Recommended decisions (proposed; owner approval still required)

**Independent review (OpenAI GPT-6.1 Sol, 2026-11-12): verdict “Needs changes.”** Its blocking findings and decision recommendations were reconciled into this amendment: offline-doctor consistency, a distinct P0 preview without a token, apply-time runtime evidence, cross-process locking, structural recovery, conflict/create deferral, and corrected reuse claims. The review is advisory and is not human approval.

The table records the reviewer’s **original** recommendation (A-only). The owner subsequently authorized MCP-B–MCP-D, which are now implemented; the later authorization is recorded under `## Approval`. The recommendations below remain the design baseline for the implemented behavior.

| # | Issue | Recommended decision | Guardrail |
|---|---|---|---|
| 1 | Approval scope | **Approve MCP-A only now.** Keep MCP-B–D deferred as proposed direction. | A has no writer, no usable apply path and no connection execution. |
| 2 | MCP-B write surface | **Require a separate pre-start write-surface authorization**, not B4 and not implied by A. | Authorization before T34; evidence acceptance at CP-B is a separate step. |
| 3 | Context7 auth | **OAuth recommended default, explicitly accepted in the plan.** API key is a separately selected alternative. | No automatic login, credential-store access or silent API-key fallback. |
| 4 | Exa auth | **Keyless default after explicit acknowledgement of rate/privacy limits.** OAuth/API key optional. | No automatic authenticated/paid upgrade; no keyless-unlimited claim. |
| 5 | JSONC | **Lossless byte-range editing only.** Refuse when preservation cannot be proven; atomic replacement, not live overwrite. | Preserve unrelated bytes/comments; ambiguous comment ownership or delimiter edits produce a manual snippet. |
| 6 | Conflicts | **Always refuse in v1; no replace flag.** | Updates to unchanged, proven-managed entries remain allowed; unmanaged/user-modified entries do not. |
| 7 | Connection test | **CLI-only when later approved; no TUI connection action in v1.** `--network`/`--connect` are deferred out of MCP-A. | TUI shows `connection not tested` and CLI guidance; no process start or auth side effect from the TUI. |
| 8 | Missing config | **No creation in v1; no `--create` authorization.** | A future create design needs exclusive creation, absent-file identity, parent-directory handling and recovery of an unchanged tool-created file. |
| 9 | Future scope | **Defer uninstall, `--with-mcp`, project-local management and CodeGraph project-init.** | Each needs a separate precedence/ownership/recovery/consent design. |

**Implementation confirmations (not owner decisions):** CodeGraph `codemode: false` per the vendor print-config recipe; Context7/Exa default `codemode` remains to be verified at implementation; CodeGraph 1.6.2 is locally verified but the supported minimum version must be recorded in the manifest; exact Context7/Exa headers/endpoints are re-verified before any write.
