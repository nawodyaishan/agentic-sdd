# Feature 015 — Fleet Manager TUI Integration & Model Configuration UX

**Proposed revision:** r2 (2026-11-11) — focused UX refinement informed by public main TUI source, Exa and Context7; **draft, not approved**. Preserves existing 009/013/014 authority, 21 functional requirements and 38 original acceptance criteria, with supplementary UX checks below. The proposed live feature branch / uncommitted TUI implementation was **not accessible** for this review.

**Status:** DRAFT — for human review; no combined spec/plan/tasks approval or implementation authorization.
**Prepared:** 2026-11-11
**Proposed canonical path:** `specs/015-fleet-manager-tui-experience/spec.md` — verify that `015` is unused in the live checkout before creating this directory. The prior suggestion to use 015 for a local marketplace was not an approved repository feature in the supplied evidence; if the number is now allocated, preserve history and choose the next available number.
**Depends on:** Feature 013 Unified TUI Experience (published TUI surface), Feature 014 Agent Fleet Model Overrides (implemented batches awaiting acceptance), and Feature 009 OpenCode orchestration (approved behavior and permissions).
**Output:** Add an accessible, dark-only, fully integrated **Fleet Manager** to the existing Agentic-SDD Bubble Tea v2 TUI. Reuse the exact fleet engine and shared application operation interfaces. Do not create a new configuration store or agent runtime.
**Target:** The currently supported paired `agentic-sdd` / `agentic-sdd-tui` distribution and tested OpenCode V2 configurations. No implied new platform or OpenCode-version support.

> **Authority boundary:** Drafting this specification is read-only design work. The 2026-11-11 Feature 014 handover explicitly says B1–B9 await human review and Opus final verdict; trusted-base adoption, runtime binary exercise, release and real-home application remain blocked or separately authorized. This spec does not reopen those gates, authorize a new batch, permit paid inference, modify user/private Git files, install to `~/.config/opencode`, or publish a release. Existing SDD combined approval and authorized-batch rules remain in force.

## 0. Executive product decision

**Extend the delivered four-destination TUI; do not add a fifth global navigation destination or build a second TUI framework.** Present the Fleet capability as **four local views** (Overview, Profiles, Review Changes, Recovery) with contextual pickers and dialogs rather than eight competing tabs. The screen must continuously distinguish **unsaved personal source** from **unapplied OpenCode deployment**. Place **Fleet Manager** as a discoverable function under **Install & Manage**, with a contextual Home action and help links. All fleet mutations must flow through **one shared application-service boundary**, using Feature 014's typed status/plan/apply/restore engine and existing Feature 013 `internal/appops` operation host.

The desired experience for a solo developer is:

```text
Home → Install & Manage → Fleet Manager
       ├─ Overview: agent, model, drift, source and availability facts
       ├─ Agent Profiles: browse/search/filter/inspect all instances
       ├─ Edit Model & Enablement: select exact binding; stage in memory
       ├─ Personal Git Config: preview & explicitly save strict JSON
       ├─ Preview Deployment: fresh 014 plan, per-file diff, conflicts
       ├─ Apply: default-No consent, stale-plan guard, real result
       ├─ Restore: scoped managed backups and explicit confirmation
       └─ Diagnostics & Help: provenance, errors, unsupported facts
```

**Two distinct writes; two distinct consent boundaries:**
1. **Save personal source config**: write a new authorized `fleet.json` state to the user's selected local private Git checkout, with its own conflict check and recoverable backup. **No install to OpenCode. No Git commit/push.**
2. **Apply managed fleet**: after refreshing the plan against the saved configuration and current installed state, explicitly apply the selected Feature 014 plan to authorized OpenCode-managed destinations. **No model invocation, no change to AGENTS.md, no entitlement assumption.**

The screens must never write files or bypass the accepted engine merely because a user confirms in the TUI.

## 1. Evidence and boundary

### 1.1 User-provided, authoritative recent Feature 014 handover

`2026-11-11_014b.md` reports:

- Combined Feature 014 spec/plan/tasks **approved 2026-11-11**; B1–B5 and B6–B9 implemented, integration green, **awaiting human review**. No further batch, merge/release, real-home apply, or trusted-base adoption authorized.
- `internal/fleet/` implemented in dependency-free Go 1.23; 18-name resolve/render tests passed; existing 009 `--with-opencode` still installs 14 profiles.
- Plans are integrity-bound, source/location treatment documented, mode `0600`; lock via OS `flock`; exit 2 validation/1 runtime; recorded 18-op plan ~852 microseconds and 100-op plan ~4.08 milliseconds **in those tests**, not general performance guarantees.
- Option-A adapters live under `opencode-pack/adapters/`; `vision` aliases `sdd-visual-inspector`; FR-15 user-driven import/adoption was deferred. **This TUI must not secretly reimplement the deferred FR-15.**
- `authenticated` doctor label based on a CLI dump is still an unresolved scope choice. The TUI must represent authentication as **unknown/not verified** unless an accepted evidence source is available.
- Final architecture/verification verdict (Opus), trusted-base adoption and runtime release exercise remain prerequisites for a supported Feature 014 deployment claim.

### 1.2 Latest inspected public repository material (2026-11-11, via Exa)

- `docs/tui-architecture.md` now reports Feature 013 B1–B13 accepted/published, a `v0.8.0` Homebrew cask with paired CLI/TUI binaries, and a **nested Bubble Tea v2 host** in `experiments/tui/` using `internal/appops` for shared operations. The same doc contains older historical text about the 012 spike; do not treat its past read-only-only rule as superseding 013's delivered controlled-operation seam.
- `specs/013-unified-tui-experience/spec.md` defines the **four global areas**: Home, Install & Manage, Explore, Activity & Help. It also defines goal-first onboarding, preview-first changes, rich diff, backup/restore, compact/dark responsive layout, and keyboard/help contracts. These remain authoritative for the existing shipped TUI.
- The existing TUI is **Go 1.26 in a nested module** using pinned `charm.land/bubbletea/v2`, Bubbles v2 and Lip Gloss v2. The root remains **Go 1.23 with no third-party dependencies**. Preserve this split.
- `specs/014-agent-fleet-model-overrides/spec.md` defines `fleet status|plan|apply|restore|doctor`, strict JSON personal configuration, alias/instance representation, a last-applied drift ledger, no modification to role prompts/permissions, no automatic model calls and no new fallback policy. **Use the actual current CLI handlers and typed engine signatures at implementation time.** This specification does not claim all B1–B9 are accepted or released.

### 1.3 External reference patterns (Exa + Context7)

| Reference | Pattern adopted | Pattern deliberately not copied |
|---|---|---|
| [Bubble Tea v2](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md), [Bubbles v2](https://github.com/charmbracelet/bubbles) | Model/Update/View separation; stateful navigation; keyboard and viewport components; declarative terminal view | A second TUI runtime or rewriting shipped Feature 013 screens |
| [Huh v2](https://github.com/charmbracelet/huh) | Conditional form validation and screen-reader-compatible prompts if justified | Mandatory form library or a dependency migration just for fleet models |
| [chezmoi](https://www.chezmoi.io/user-guide/setup/) | Declarative Git-backed source → inspect diff → deliberate apply | Automatic Git commits/pushes, bulk dotfile ownership |
| [OCX](https://github.com/kdcokenny/ocx), [docs](https://ocx.kdco.dev/docs/) | Editable user profile, ownership records, update hashes, local-edit protection | A second OpenCode profile package manager, registry, or launcher |
| [Vercel CLI design guidelines](https://vercel.com/design/cli) | Dry-run/no-op clarity; explicit diff; immutable plan identity; stale operation refusal | UI-specific safety rules divergent from the CLI |
| [OpenCode agents/config/models](https://opencode.ai/docs/agents/) | Distinguish agent installed, enabled, selected primary, model available and actually authenticated | Treating a catalog model or session agent as proof of provider entitlements |

**Research qualification:** Examples and documentation inform design. No external source validates this repository's installed package, all runtime permissions, private Git state or final 014 review verdict. Bubble Tea v1 and v2 API snippets differ; follow the versions pinned in the nested TUI module rather than mixing types.

## 2. Scope, audiences and success

### 2.1 Primary users

- **New solo developer**: sees a plain-language explanation of “Agent profiles” and can inspect configured models without understanding OpenCode frontmatter.
- **Advanced multi-provider user**: updates one agent instance from a Free/Go/9Router/OpenAI route to another exact model ID with source provenance, guarded preview and reliable restore.
- **Learner/intern**: can understand what an agent does, what model powers it and what remains unverified, without being forced through a new SDD approval workflow.
- **Maintainer**: can diagnose drift, compare the generated/installed state, and reproduce test cases with a temporary home.

### 2.2 User goals

G1. Locate Fleet Manager quickly from the shipped TUI.
G2. Understand actual managed agents, model bindings, config source and installation status.
G3. Stage **one or several model/enablement edits** without changing role instructions/permissions.
G4. Update Git-backed personal JSON **safely and separately** from updating installed OpenCode agents.
G5. Inspect a fresh full plan/diff, then intentionally apply it.
G6. Detect drifts, stale plans, missing models, conflicts and restore options without guessing what happened.
G7. Accomplish the same actions through the noninteractive CLI and retain the prior four-destination UX.

### 2.3 Product-level definition of done

A user on a supported and accepted Feature 014 build can open the delivered TUI, choose a local personal `fleet.json`, change a permitted model or `enabled` assignment, review a private-source diff, save that source intentionally, review the new fleet install plan and all affected agent-file differences, apply with a separate default-No confirmation, inspect actual result/drift, and restore a managed backup where supported. If any precondition is not met, the TUI must **explain and stop** rather than silently upgrade the user's authority or pretend success.

All paths and test environments are explicit; **no real-home application is authorized by this specification**.

## 3. Architecture and ownership

```text
Existing Feature 013 TUI (nested module: experiments/tui, Go 1.26)
  Home -> Install & Manage -> Fleet Manager screens
               |        (navigation, focus, rendering, staged UI state)
               v
  FleetTUIAdapter (typed requests/results; no filesystem access)
               |
               v
  internal/appops (root Go 1.23, shared CLI/application boundary)
    |                 |
    | read/status     | source edit, plan/apply/restore/doctor
    v                 v
  internal/fleet  [EXISTING 014 ENGINE + NARROW ADDITIVE SOURCE-EDIT USE CASE]
    strict JSON, canonical role/preset resolver, renderer, digest/plan,
    drift, locking, transaction, managed backup, restore, doctor
               |
        +------+-------+
        |              |
        v              v
  Private local     Managed OpenCode agent files
  fleet.json        (global/project only if actually supported by 014)
  (user Git)        ~/.config/opencode/agents/*.md

Root CLI uses identical appops / fleet operations for every mutation.
OpenCode executes models; Agentic-SDD manages source files, not sessions.
```

### 3.1 Strictly enforce boundaries

- `experiments/tui/` owns **UI view state only**, typed immutable request/response adapters, keyboard help, accessible layout, and feedback. Screens **never** directly use `os.WriteFile`, `os.Rename`, `filepath.Walk`, arbitrary shell/Git calls, manifest/lock writes, `skillsync.Sync` or `fleet.Apply`; they **never** parse CLI stdout to decide safety.
- `internal/appops` owns the typed seam shared between the CLI and TUI. It calls Feature 014's authoritative `internal/fleet` functions; do **not** duplicate validation, path canonicalization, permission comparisons, hash calculation or backup logic in screens.
- `internal/fleet` remains root **Go 1.23, stdlib only**. New source-edit capability (if 014 lacks it) must be a small typed, separately authorized **atomic save** operation with strict JSON schema validation, compare-against-last-read digest and recoverable backup. Do not redesign the 014 renderer/manifest.
- The personal JSON owns **model identifier, variant and enabled state only**. The canonical `opencode-pack` owns prompt, agent role, permissions, delegation/fallback policy. The TUI has **no free-form prompt/permission editor**.
- No automatic Git clone/checkout/commit/push, no credential reads, no provider authentication call, no paid model testing, no silent default-agent activation, no hidden update-install transaction.
- Root CLI methods/flags that predate 015 remain backward-compatible; 009 `--with-opencode` retains its 14-profile behavior. The new TUI respects 014 and 013 accepted versions and source ownership.

### 3.2 Source edit/write decision

The public product must **not** introduce a TUI-only mutation implementation. One of these two designs must be selected after source inspection:

- **Preferred:** Add a small root `appops` + `internal/fleet` `SavePersonalConfig` typed operation, exposed to both CLI and TUI, with in-memory staging and optimistic-concurrency hash checks. `fleet.json` write is a separate explicit operation from `fleet apply` and **does not alter any `~/.config/opencode` files**.
- **Acceptable constrained fallback:** For this release, provide a verified, documented “Open `fleet.json` in your editor” action with explicit user consent plus re-read/re-plan upon return. The Fleet TUI then remains preview/apply/restore capable but **must not advertise an integrated JSON editor**. No unsafe shell-string interpolation or auto-apply after editor exit.

Do not ship a screen that appears to save changes but only alters in-memory labels.

### 3.3 Read/inspect provenance

The UI must derive meaningful labels from engine results. It cannot claim that a model is authenticated simply because OpenCode lists the model or that a profile is running simply because an agent file exists. The handover's `authenticated` doctor scope is pending; until accepted evidence exists, use `Not verified` and offer safe local diagnostics or manual instructions.

## 4. Navigation and UX contract

### 4.1 Information architecture — r2 refinement

**Preserve existing four global destinations**: Home; Install & Manage; Explore; Activity & Help. Fleet Manager is a third **Install & Manage child**, beside Install and Backups, and a contextual Home action. It is not a fifth root Route.

Within Fleet Manager, offer **four local views**:

1. **Overview:** selected source, the two independent configuration/deployment states, next appropriate action and compact diagnostics.
2. **Profiles:** browse/search/filter all instances by role/provider/changed/status; select instance; contextual Model Picker and Enablement controls; detail drill-down.
3. **Review Changes:** source JSON diff and save confirmation, followed by a separately generated deployment plan/file diff and apply confirmation. These remain two explicit operations, not one wizard transaction.
4. **Recovery:** conflict/drift/legacy diagnostic explanation, managed backup preview and restore, with truthful outcome and CLI recovery hints.

The Model Picker, Source selector, Profile Details, Diagnostics, Help, confirmation, and Result are **contextual overlays/drill-downs**, not independent persistent Fleet tabs. Keep current tab, selection, filters, scroll position and in-memory draft when navigating among Fleet views; existing parent routes preserve their own state. All users retain a route back to Install & Manage.

**Navigation priority:** The first screen answers `What is configured?`, `What differs?`, `What next?`. A new user can inspect a read-only example; no private config path is guessed and no Git remote is contacted. A seasoned user can change one model in Profiles, filter by `Changed`, and move directly to Review Changes without visiting eight screens.

**Non-goals remain:** no hosted marketplace; no provider auth management, inference, subscription/price ranking, automatic Git commits or TUI-only policy edits.

### 4.2 Local view and contextual detail contract — r2 refinement

| Surface | Show by default | Primary action | Contextual detail |
|---|---|---|---|
| **Overview** | Source selected/not selected, `Personal config` draft/save status, `OpenCode files` managed/current/pending/drift status, one next action, honest runtime `not checked` | Inspect profiles or choose source | Source selector, last-operation detail, diagnostics |
| **Profiles** | Searchable agent instance ID, compact role, provider route, model summary, enabled, managed/drift status; changed count | Inspect instance / stage model choice | Role/source detail, Model Picker, Enable/Disable confirmation of consequences |
| **Review Changes** | Sticky summary of `N personal source edits`, creates/updates/**removals**/no-ops/conflicts, scope and last observed source/plan identity | Review source diff → confirm source save; then **fresh** plan → confirm OpenCode apply | Complete scrollable per-file diff and consequences; no silent apply |
| **Recovery** | Only relevant conflicts, manual drift, legacy provenance, managed backups, truthful failed/partial result | Replan, inspect, preview safe restore, seek CLI guidance | Error detail `What happened / What stayed unchanged / Next step`; restore modal |
| **Model Picker** *(contextual)* | Current choice; groups `Current`, `Known local assignments`, `Locally cataloged`, `Enter exact provider/model#variant`; source of suggestion, unknown availability | Stage only; no API call | Exact ID/variant; route vs model family; no unverified pricing/entitlement |
| **Profile detail** *(contextual)* | Canonical role/source, instance ID, full model ID, enablement, installed/runtime state, engine-derived permission summary | Change model, toggle, return | External symlink/reviewer limitation when relevant |
| **Source & diagnostics** *(contextual)* | Explicit local private `fleet.json` path (redacted where necessary), strict validation, digest, Git state only if safely read | Select/reload source; inspect failure | No automatic scan, clone, credential read or Git write |

The overview's two-state indication is persistent across Fleet views when geometry permits. It must not introduce a sticky status bar that hides material warning, Back/Cancel, or confirmation on short terminals. `Saved locally` does **not** mean `Git committed`; `Applied to OpenCode files` does **not** mean `agent ran` or `model authenticated`.

### 4.3 Goal-driven flows

**Flow A: First inspection (no writes):** Open Fleet Manager → select known local `fleet.json` or use clearly labeled sample → view agents and models → check status → exit. No model or network calls.

**Flow B: Change one model:** Select profile → choose/enter model → stage edit → see exact personal JSON diff → **Save personal source? [No]** → on consent save via shared operation → recompute 014 plan → full OpenCode diff → **Apply to chosen target? [No]** → on consent apply → show measured result and backup. The two confirmations are not merged.

**Flow C: Toggle an agent:** Disable a managed instance → show that 014 may **remove a managed installed file** rather than merely hiding a checkbox → exact affected path and backup → confirm save source → separately confirm install plan. An unmanaged instance cannot be removed automatically.

**Flow D: Private config edited externally:** On return or refresh, compare source digest → show `Source changed outside TUI` → discard/reload or review conflict; never overwrite unseen edits.

**Flow E: Installed agent drift:** Show `Installed file modified` with the precise affected managed instance → default to no overwrite → offer view diff / replan / documented recovery. Do not auto-merge frontmatter, prompts or permissions.

**Flow F: Stale plan:** Any source/target/renderer/installed-state change since preview causes apply to refuse → retain the user's staged choices in memory if safe → require a fresh plan and new confirmation.

**Flow G: Failure/restore:** Simulated interrupted update produces an accurate failed/rolled-back/needs-recovery state → show backup ID and next safe action → restore only after separate preview and confirmation.

**Flow H: Missing 014 release/acceptance:** Fleet Manager can show supported **read-only exploration**, with a clear `Fleet engine not yet approved for installed-file changes` state. No hidden production apply path may circumvent the pending 014 review, trusted-base or release boundaries.

## 5. Functional requirements

### FR-01 — Existing shell integration

Add a first-class Fleet Manager route **inside** the existing Feature 013 shell. Preserve Home, Install & Manage, Explore, Activity & Help, onboarding, activity logs, update behavior, help, client installer and backup UX. The Fleet route must use the currently delivered navigation and key-binding conventions, not a nested application that steals global shortcuts. TUI users must be able to return to Install & Manage without losing unrelated screen state.

### FR-02 — Read-only overview and truthful states

Derive counts, labels, agent roles, instances and available actions from typed Feature 014 inventory/status responses. Do not embed constants such as 18 profiles, 14 pack members or 100 managed files in the UI. Distinguish at minimum: `source configured`, `source missing/invalid`, `rendered`, `managed installed`, `unmanaged collision`, `disabled`, `drifted`, `legacy/unknown provenance`, `model configured`, `provider/catalog availability not checked`, `selected primary not checked` and `blocked`. Use meaningful text and icons together; no vague green `Ready` for an agent merely being installed.

### FR-03 — Profile browser and detail

Provide a filterable, keyboard-driven list of agent **instances** with model IDs, logical role, personal/config origin, enabled state and effective installed state. The detail pane displays the canonical source and a read-only permission summary derived from Feature 014, including whether any effective runtime permissions remain unverified. Do not construct permissions from display strings or agent names. For a role alias, show the canonical role and the alias separately.

### FR-04 — Model picker and safe candidate selection

Permit model/variant changes for one selected instance by choosing from (a) the model IDs already present in the user's local desired configuration, (b) existing supported/locally observable enabled catalog entries if provided by Feature 014, and (c) an explicitly entered exact provider/model identifier. Clearly distinguish **known syntax**, **locally cataloged**, **configured**, **authenticated (if actually checked and accepted)**, **usable** and **unverified**. Never promise subscription access, price, rate limits, zero-retention or automatic route success. Validity is determined by shared root validation, not screen-specific regexes. No model inference occurs while browsing or selecting.

### FR-05 — Enable/disable semantics

Provide explicit enable/disable controls only for an instance that Feature 014 can manage safely. Show the resulting rendered/installed-file operation, including managed removal and potential backup creation. Unknown or unmanaged files cannot be removed by toggling a checkbox. Do not conflate `enabled` with `installed`, `default primary`, `live model availability` or a working subagent. Reviewer isolation limits stay visible even for enabled instances.

### FR-06 — Personal strict-JSON source selection

Allow a user to enter or select **an explicit local file path** to strict-JSON `fleet.json` in their private Git checkout. The UI may display Git provenance only if a specifically permitted **read-only metadata source** is available; it must not traverse the user's home searching for credentials, use network Git operations or assume a private repo exists. Reject symlink/traversal/unsafe paths according to Feature 014 source handling. Saving a new path preference is not permission to mutate that file, other Git files or installed agents. If config-source preference persistence is desired, use the existing accepted TUI preferences mechanism only after separate scope review; default to session-only selection.

### FR-07 — Staged editing and safe personal-source save

Maintain staged edits in memory until a separate, visible **Save personal configuration** action. Show the exact proposed strict-JSON diff, including unchanged fields when relevant for comprehension; show current source digest and any external-modification conflict. Saving must use a **typed shared application operation** and root stdlib implementation with:

- `schemaVersion`/field allowlist, unknown/duplicate-key rejection and size/UTF-8 checks from 014;
- file identity, canonical path, mode, ownership/symlink and last-observed digest precondition;
- deterministic serialization with stable field order and no hidden unrelated model changes; any full-file formatting changes are previewed;
- temporary file with appropriate private mode (`0600` on supported POSIX targets), atomic replacement where supported, recoverable backup and accurate failure reporting;
- no writes to OpenCode agent files, global `AGENTS.md`, provider credentials or Git history; no automatic Git stage, commit, push, pull or clone;
- a clearly separate source-save confirmation (default No) and state refresh on success.

If the implementation chooses the documented external-editor fallback instead, remove in-TUI source-save claims and keep source re-validation mandatory after editor return. Do not silently invoke `$EDITOR` with shell expansion.

### FR-08 — Exact deployment preview

Generate a fresh Feature 014 `fleet plan` through `internal/appops`, preserving the plan's integrity/identity semantics. Display exact current/desired file paths, create/update/remove/no-op/conflict/blocked status, source and target digests where safe, model changes, permission implications and stable plan ID. Provide **full diff access**, not only counts. Long paths, long model IDs, Unicode and many affected files must be scrollable and reachable. Plan freshness is checked when entering confirmation and again by the authoritative engine immediately before apply. A saved plan path/ID is not user approval.

### FR-09 — Explicit apply and outcome

The TUI may dispatch 014 `fleet apply` only after:

1. Feature 014's exact deployed engine and intended target scope are eligible and accepted for real-home operation;
2. a successfully validated, current plan exists;
3. every conflict or drift is resolved by an accepted operation, never an unreviewed merge;
4. a separate explicit **Apply to OpenCode files?** confirmation is given, with No preselected;
5. `appops` revalidates plan/source/installed state and acquires the engine lock.

Use typed progress/results. Block navigation or confirm double submission while an operation owns the lock, but provide a safe way to inspect status. Cancellation is honest about what was committed and what rollback succeeded. A cancelled view is **not** evidence that an already-started filesystem transaction was cancelled. Show the final actual outcome and recorded backup; never infer success from an optimistic UI state.

### FR-10 — Restore and managed backup experience

Use existing 014 restore plus Feature 013 backup UX where possible. Expose only recoverable managed 014 backups in a scoped Fleet Manager context. Preview the exact current-versus-restore delta, modified-target conflicts, ownership limits and backup identity. Restore requires a **separate default-No confirmation**; no restore if the target changed unexpectedly since preview. Never imply a `git revert` changes installed OpenCode files. Other application backups must remain accessible through the existing Install & Manage workflow.

### FR-11 — Drift, legacy and conflict recovery

Display the precise three-state status from 014 (`last applied`, `current`, `new desired`) using the current manifest/ledger. Present remedial choices as actions with consequences: reload source, replan, preserve installed change, review manual edit, restore a managed backup or stop. Format-1–3 / format-4 ambiguity remains **unknown** until explicitly adopted through a supported 014 operation. Since 014 **deferred FR-15 import/adoption**, the TUI must **not** offer a functional auto-adopt button unless that operation is separately approved and implemented. Instructions may point to the existing supported CLI/manual process.

### FR-12 — Diagnostics and actionable errors

Translate typed engine errors into concise, non-patronizing, user-actionable messages without masking the original technical reason. At minimum handle invalid strict JSON, bad schema, duplicate keys, missing canonical source, invalid model, unsupported OpenCode runtime, config missing, source changed, target drift, unmanaged collision, symlink/refusal, stale plan, lock contention, backup missing/corrupt, interrupted apply, restore refusal, unverified runtime permissions, missing authentication evidence and failure to locate the 014 engine. Never disclose personal tokens, credential contents, full private JSON or untrusted instruction text in debug output. Support a safe redacted details view if the existing TUI pattern permits.

### FR-13 — Orchestrator selection remains separate

Show the `sdd-orchestrator` as a normal managed profile and distinguish installed/enabled from the active primary session. Do not set `default_agent` automatically. If a later feature offers explicit activation, that is a separate approved source/config operation with preview and restore; 015 must not sneak it into a model change. Display an appropriate help hint for manual selection using OpenCode's supported behavior.

### FR-14 — Scriptable CLI parity

For every mutating action made possible by the TUI, expose or reuse an equivalent supported **root CLI application operation**; no screen-only filesystem write. Preserve `fleet status|plan|apply|restore|doctor` and normal skill install behaviors. If 015 adds an in-app personal source editor, add the smallest corresponding CLI/application-service operation with the same validation, identity, dry-run, conflict, rollback and exit-code semantics. The TUI must never spawn `agentic-sdd fleet ...` and parse text as its security API.

### FR-15 — Async work without duplicate execution

Perform potentially slow read/plan/doctor/save/apply operations via the existing Feature 013 command/result channel. Provide busy states and prevent duplicate submissions or race-induced plan overwrites. Use cancellation only when the application service supports it; never force-stop a transaction in a way that leaves the UI reporting false rollback. If a result arrives after the user navigated away, record its outcome and refresh the relevant screen without triggering new writes. A refresh or resize must never resubmit an operation.

### FR-16 — Responsive, accessible and dark-only UI

Reuse the shipped Feature 013 `agentic-dark` palette and its typography hierarchy, focus tokens, reduced-motion/static and ASCII/no-color preferences. No light mode and no redesign of the global shell. Supported responsive targets from Feature 013 continue to apply: **≥110 columns** allows list/details, **80–109** compact root navigation with an optional Fleet list-to-detail drill-down when long exact IDs would be clipped, **60–79** single pane with detail navigation, and **<60** minimal content or explicit resize guidance. Short terminal heights must preserve accessible Back/Cancel, main warning, current selection and confirmation. Long exact paths and diffs must be reachable by scrolling. Never rely on color alone; label warning, success, unavailable and conflict states in text.

### FR-17 — Keyboard and form interactions

Honor existing 013 keys: arrows or j/k move lists when unfocused, Enter opens, Esc backs/cancels, `/` searches, Tab cycles focus, `?` help, Home/End bound selection and `q` quits when not editing text. **When an input field is focused, printable keys—including j/k/q and `/`—must be treated as text.** A global key must not activate a destructive action while typing. Show a contextual footer with only relevant keys, and a full help view. Escape during a staged edit asks about discarding unsaved changes rather than silently losing them.

### FR-18 — Configured model taxonomy, not a marketplace

Group locally displayed provider routes for clarity: `opencode` (Console/free where verified), `opencode-go`, `9router` (label known Claude/Gemini/Argon/other upstreams accurately when evidenced), `openai` (including optional ChatGPT sign-in when verified), and custom providers. Sorting, filtering and model search use locally known names; **do not implement remote package browsing, model shopping, 9Router account APIs, synthetic provider names, subscription billing, or a cloud marketplace**. Previously proposed local catalog/remote marketplace work is deferred and remains a separate product decision.

### FR-19 — Feature availability and release dependency guard

If Feature 014 has not passed its required human review, runtime-binary exercise, trusted-base boundary decision and publication/installation gate, 015 may be developed **against isolated fixtures and an opt-in preview mode only**. Production apply/restore controls must be disabled or absent until the actual dependency contract is accepted and bundled. The TUI may show `Fleet operations unavailable in this build` or `Preview only` with accurate next steps; do not make a `Ready` badge from green isolated tests. Once 014 is accepted, a separate reviewed build/release decision may enable the supported operations.

### FR-20 — Developer maintenance and compatibility

Keep root Go 1.23 and zero third-party `require` entries, leave Feature 013's nested Go 1.26 / pinned Charm v2 dependency strategy unchanged, and avoid importing the nested module into root packages. Place new view models and tests alongside the existing TUI screen code, and typed shared operations alongside existing `internal/appops`; prefer minimal pure helper functions over another framework or daemon. Preserve existing binary discovery (`agentic-sdd tui`, paired `agentic-sdd-tui` Homebrew cask), CLI/non-TTY behavior, the 013 four-route shell and 009 `--with-opencode` installer. Determine exact files and signatures from the working tree, not this proposal's pseudocode.

### FR-21 — Onboarding and documentation

Add one concise guided path to the existing TUI Help/Quick Start: **“Configure a personal OpenCode fleet”**. Explain the difference among role, instance, model, personal Git source, rendered file and installed state. Provide a safe temporary-home demo with one model change, one drift/conflict and one restore. Document that saving private JSON does not apply to OpenCode, that changing a model does not grant subscription access, and that restore does not alter Git history. Link to the existing canonical Feature 014 CLI docs and explain preview-only states without exposing private paths.

## 6. Visual design and state semantics

### 6.1 Dark-only r2 wireframe (illustrative, not a fifth root destination)

```text
AGENTIC-SDD / Install & Manage / Fleet Manager
────────────────────────────────────────────────────────────────────────────
Personal fleet: private/fleet.json   | Config: 2 UNSAVED | OpenCode: unchanged
[ Overview ]  [ Profiles ]  [ Review Changes 2 ]  [ Recovery ]

Agents / search [/]                 │ Selected instance
Role: All   Route: All   [Changed] │ sdd-implement-1  / Implementation
> sdd-implement-1       Go        │ Model: opencode-go/deepseek-...
  sdd-implement-9r-1    9Router   │ Source: preset:sdd-implement-1
  sdd-lightweight       Console   │ Enabled in source: Yes
  sdd-openai-architect  OpenAI     │ Installed: Managed / current
                                   │ Runtime: NOT VERIFIED
                                   │ [Change model] [Disable]

Draft: 2 edits. Personal JSON not saved; no OpenCode files changed.
Enter details   / search   Tab focus   ? help   Esc back
```

Counts, paths, models and identities are examples only. Real values come from typed Feature 014 results. List rows may abbreviate long model IDs, but exact IDs must be accessible in detail and diff. At width under 110 columns prefer a focused list → detail drill-down rather than forcing narrow permanent split panes; preserve Feature 013 root responsive shell, lower-width safety and viewport contracts.

### 6.2 Meaningful statuses

| UI text | Permitted evidence | Not implied |
|---|---|---|
| `Configured` | Valid personal JSON binding | Model authenticated or usable |
| `Enabled in source` | Personal config state | Agent installed or running |
| `Installed / matches managed source` | Actual installed hash matches last applied | OpenCode discovered profile or has entitlement |
| `Modified on disk` | Current digest differs from last applied | File malicious or must be overwritten |
| `Unmanaged` | Target exists without managed provenance | Permission to adopt or delete |
| `Legacy / provenance unknown` | Missing desired digest in historical manifest | Proven safe restore or fresh desired state |
| `Model found in local catalog` | Optional read-only model listing check | Paid subscription or actual provider route works |
| `Authentication not verified` | No accepted relevant auth evidence | Authentication denied |
| `Runtime validation not run` | No version/dispatch exercise | Runtime compatibility |
| `Blocked / no release authorization` | Feature acceptance/release dependency missing | A request to override the gate |

### 6.3 Interaction states (state machine)

```text
NoSource ──select──> LoadedClean ──edit──> StagedDirty
                       |                      |
                  inspect/plan          preview source diff
                       |                      |
                  ReadOnlyPlan      save? (No default)
                       |                      |
                       |           shared source-save operation
                       |                      |
                       +-------> SavedNewSource
                                      |
                            fresh fleet plan / inspect
                                      |
                             ReadyPlan or Conflict
                                      |
                        confirm? (No default; only eligible)
                                      |
                           apply via appops/014
                                      |
                    Applied / NoOp / Failed / RolledBack
                                      |
                         refresh + optional restore
```

Navigation and external file events may invalidate a staged digest or plan. They never silently apply. A stale/unknown state has no direct edge to an enabled Apply action.

### 6.4 r2 UX behavior refinements (normative; reuse existing FR-01–FR-21)

**A. Two explicit state tracks.** Project **personal JSON** (`missing`, `invalid`, `loaded`, `draft dirty`, `saved locally`, `externally modified`) and **managed OpenCode files** (`not installed`, `current`, `pending apply`, `drifted`, `unmanaged`, `legacy unknown`, `disabled`). Runtime model/catalog/auth evidence is an independent **third** track (`not checked`, `catalog visible`, etc., only if observed). Do not collapse these into a single `Ready` state. The UI may compute its in-memory draft dirty count, but may not infer authoritative installed state or permissions.

**B. Save-deploy checkpoint.** After a valid confirmed personal-source save, explicitly state **“Saved locally. OpenCode files unchanged.”** Offer **“Preview OpenCode changes”** as the next step. A fresh 014 plan must be created only from saved current source. After deployment show the actual engine outcome, backup reference and remaining unknowns. A user can exit between save and apply without losing the source state. A `git revert` alone never deploys anything.

**C. Impact-first review without hiding details.** Begin with source or destination, exact scope, create/update/managed-remove/no-op/conflict counts and reason for any block. Show risky operations/removals/conflicts before routine updates. Provide a scrollable file list and complete exact diff (including 1,000+ lines) on demand, with stable selected file and scroll. No `Quick apply` bypasses full preview or default-No consent; no need to force the user to scroll every line before a safe no-op.

**D. Focus model.** One focus owner at a time: global shell / Fleet view / list / detail / text field / modal. Contextual footer changes with focus. `q`, `/`, `j`, `k` are literal text while input is focused; Enter cannot submit two writes from rapid presses. On Esc with dirty draft, offer **Keep editing / Discard draft / Stay** without overwriting the disk. If the terminal becomes too small during a confirmation, disallow affirmative action until the choice and warning can be shown.

**E. Async identity.** Every read/plan/doctor response is tagged with the selected source path, source digest, target scope and generation. A stale reply cannot replace current source, clear a staged draft, overwrite a newer plan or prompt an unexpected apply. Writes use the existing single-write OpsHost slot; status refreshes and resize never resubmit transactions. Cancellation messaging follows actual root host behavior rather than suggesting an in-progress write may be interrupted safely.

**F. Model discovery without fake confidence.** Present provider **route** separately from optional known model **family/upstream**. Local bindings/presets/catalog entries are suggestions only; do not rank by cost or imply subscription. Explicit exact custom ID remains possible under accepted 014 syntax. Free-tier and 9Router labels reflect their actual observed catalog only, never claims of guaranteed availability or retention.

**G. Copy and recovery.** Every failure presents **What happened**, **What stayed unchanged**, and **Next safe action**. A stale plan offers Replan; a stale JSON source offers Reload/review; a modified managed file offers Inspect/CLI recovery. Do not show enabled `Adopt`, `Force overwrite`, `Fix all`, or automated private Git reconciliation while 014 FR-15 remains deferred.

**H. First-use path.** Without a config file show `Choose existing local fleet.json`, `View read-only example`, and `Read CLI setup guide`. Never auto-search private Git repositories, initiate a Git clone or generate a file without explicit approved source-edit operation. Onboarding should be skippable and fit on one screen rather than another mandatory wizard.

**I. Theme and accessibility.** Reuse the one `agentic-dark` scheme, existing truecolor/ANSI-256 and ANSI-16 limits, ASCII/no-color and reduced-motion. Semantic text/markers—not provider colors—convey risk. Tab and Shift+Tab visit visible controls; modal keeps focus. Accessible CLI is an explicit supported alternative; do not equate Huh documentation with completed accessibility testing.

**J. Implementation touchpoints verified on public main.** Root `App` currently defines four routes; `app_screens.go` defines Hub/Install/Backups only; `ops_host.go` currently dispatches only `WriteInstall` and `WriteRestore`; `viewport.go` and `modal.go` provide the bounded viewport and modal compositor. Extend those existing seams and do not add a fifth global route or a second file-mutation host. Exact target checkout may differ and must be rechecked before edits.

## 7. Non-functional requirements (measurable)

| ID | Quality attribute | Requirement and measurement |
|---|---|---|
| **NFR-01** | Responsiveness | On reference hardware/test fixtures, view-state navigation, focus changes and filter updates across 18 and 100 instances must complete without blocking on disk/model calls; record median and p95 update/render timings. Target p95 ≤100 ms for a 100-row in-memory navigation fixture, with failures investigated rather than hidden. |
| **NFR-02** | Startup | If Fleet Manager is not opened, the existing TUI start path must not load personal JSON, invoke `opencode`, make network/model calls or run fleet plan. Measure startup delta against Feature 013 baseline and document regressions. |
| **NFR-03** | Scale | Inventory, preview and navigation must remain usable with 18 and 100 profiles, 100 planned file ops, long model IDs, long paths and >1,000 diff lines. Use viewport/scroll/pagination; record memory and latency, no unbounded rendering stalls. |
| **NFR-04** | Integrity | No source or target write can succeed after stale-source/plan/target detection; prove with deterministic race/conflict fixtures. |
| **NFR-05** | Recoverability | Simulated failure after staging and mid-apply produces accurate result/backup/rollback state with no loss of unrelated files. |
| **NFR-06** | No surprise costs | Overview, browsing, selection, plan, source save, apply and restore do not invoke LLMs or remote model APIs. No automatic online model catalog refresh. |
| **NFR-07** | Privacy | No credentials, API keys, full private JSON, unredacted home paths or unexpected Git metadata in logs, crash output, example fixtures or telemetry. No opt-out tracking added. |
| **NFR-08** | Accessibility | Keyboard-only completion; visible focus; no color-only distinction; reduced-motion/plain/ASCII respected; alternative CLI path for screen-reader users. Screen-reader-compatible form fallback should be evaluated against existing UI, not promised merely because Huh offers it. |
| **NFR-09** | Layout | Essential content, safety warnings, Back, Cancel and default-No confirmation reachable at 110/80/60 and narrow/short-height cases, including very long file paths. No horizontal clipping of material differences. |
| **NFR-10** | Compatibility | No root Go dependency/toolchain change; no 009/013 or 014 CLI contract regression; maintain nested companion build. Test only the platforms and OpenCode versions actually supported. |
| **NFR-11** | Evidence | Tests and logs distinguish UI fixture success from genuine 014 engine runtime verification; actual commit, config digest and environment are recorded for release claims. |
| **NFR-12** | Local-only UX | Offline read-only workflows function without Git remotes, OpenCode authentication, providers, or Internet. Unavailable services show actionable, nonblocking diagnostics. |

## 8. Acceptance criteria (product and safety)

All criteria receive an explicit **Pass / Fail / Blocked / Not run** entry with revision and verification class. Do not convert static UI assertions into an assertion that live OpenCode accepted a model or that 014 has been released.

### A. Navigation and discoverability

- **AC-01 — Four routes preserved:** The global Home, Install & Manage, Explore and Activity & Help routes and their existing keyboard/resize behavior pass without changes in default installation behavior.
- **AC-02 — Fleet entry:** From Home and Install & Manage, the user can reach Fleet Manager when the capability exists; missing/unsupported engine shows honest read-only/unavailable state, not a dead action.
- **AC-03 — Back/context continuity:** Selecting a profile, entering details, previewing a plan and returning retains selection, filters, scroll and source without losing another Feature 013 screen's state.
- **AC-04 — Responsive layout:** All supported width/height fixtures keep material status, actions, Back and Cancel reachable; no confirmation or security warning is clipped away.
- **AC-05 — Keyboard/focus:** `j/k/q` and `/` are text when an input is focused; Esc/back and Tab/help behave according to the existing 013 contract; no destructive shortcut fires during editing.

### B. Truthful inventory and model selection

- **AC-06 — Dynamic inventory:** List counts and state derive from typed 014 results for zero, one, 18 and 100-profile fixture fleets; no hardcoded machine-specific profile total.
- **AC-07 — Role/instance distinction:** One canonical role with multiple 9Router, Console Free and Go instances displays different agent IDs/models but one shared role policy; source provenance is visible.
- **AC-08 — Exact model syntax:** Model selector can search existing bindings, select locally known IDs, and accept valid custom provider/model syntax including `#variant`; invalid or disallowed fields surface 014 errors without mutation.
- **AC-09 — No entitlement invention:** The TUI never labels an `openai/*` model `ChatGPT Plus authenticated`, a 9Router upstream as guaranteed available, or a Console model `free forever` on the strength of the configured identifier alone.
- **AC-10 — Model-only preservation:** A staged model change does not alter the canonical prompt body, role, permissions, 009 fallback rules or SDD approvals; a permission difference blocks model-only apply unless separately accepted under its own feature contract.
- **AC-11 — Enablement with consequences:** Toggling `enabled:false` shows the exact managed-file removal or omission from the accepted 014 plan; unmanaged/profile-conflict cases cannot be silently removed.
- **AC-12 — Orchestrator state:** Installed/enabled/selected-primary/runtime-observed are reported as separate facts. The TUI does not set the OpenCode default or launch a subagent.

### C. Personal Git source editing

- **AC-13 — Explicit local source:** Selecting `fleet.json` requires a known path and rejects unsupported, unsafe, invalid, inaccessible or symlink-escaped inputs per 014 rules; does not scan private home or contact Git hosts.
- **AC-14 — Staged editing:** Changing a selection updates only in-memory view state until an explicit source-save action; quitting with dirty edits offers discard/return instead of silently writing.
- **AC-15 — Private-source preview:** The user can inspect the exact JSON changes and resulting bindings before a separate default-No save confirmation; no installed OpenCode files change during this step.
- **AC-16 — Atomic source save:** A supported shared service handles strict JSON validation, digest match, atomic write/backup and recoverable error state on temporary private fixture paths; no UI direct writes and no automatic Git operations.
- **AC-17 — External edits preserved:** If another process changes the private JSON after it was read, a stale source-save request is rejected without data loss; reloading/reshowing changes is required.
- **AC-18 — CLI parity:** The same source-save operation and constraints can be invoked via documented root CLI/application services (or, if external-editor fallback is chosen, the TUI makes no in-app save promise).

### D. Deployment, drift and recovery

- **AC-19 — Full plan:** Plan screen shows every meaningful create/update/remove/no-op/conflict, exact destination/source binding and complete diff, including large (>1000 line) plans; scrolling and selection retain identity.
- **AC-20 — Two confirmations:** Saving personal JSON and applying generated OpenCode files each require their own confirmation; No is selected by default; declining either causes no corresponding mutation.
- **AC-21 — Plan integrity:** A plan is regenerated from current saved JSON; source, target, renderer, scope or installed-state changes after preview make apply refuse and require replan. The TUI never applies based solely on summary text.
- **AC-22 — Drift refusal:** Existing unmanaged files, changed managed files and legacy unknown provenance cannot be silently overwritten; the TUI displays the reason and the next permitted action.
- **AC-23 — Accurate apply results:** A successful apply, no-op, conflict, aborted operation and partial/failed rollback display distinct outcomes from the real typed service result; no optimistic green success.
- **AC-24 — Scoped restore:** Restore only affects a selected 014 managed backup in a supported scope; stale targets/modified files refuse unless the accepted engine explicitly authorizes a safe resolution, and the UI never deletes unrelated files.
- **AC-25 — Error transparency:** A `flock` contention, plan mismatch, permission failure, corrupt backup, unsupported runtime or other engine error yields an actionable redacted explanation; no generic success status masks failure.
- **AC-26 — Deferred adoption respected:** No active auto-import/adopt action is exposed as part of the 015 TUI while 014 FR-15 remains deferred, even if a design mockup shows such a button.

### E. Existing product, quality and deployment

- **AC-27 — Shared app boundary:** Source inspection and tests show Fleet screens call only typed adapter/appops methods; they do not invoke CLI subprocess output as authority, mutate files, or calculate ownership/plan digests locally.
- **AC-28 — Feature 013 regression:** Existing Install & Manage preview/install/restore, Explore, Home, animation preferences, Help, activity views, keyboard focus and Homebrew paired-binary launch remain usable and do not unexpectedly load fleet source.
- **AC-29 — Feature 009/014 regression:** `--with-opencode` still follows approved 009 Option B; 014 `status|plan|apply|restore|doctor`, 18-name renderer and legacy manifest tests pass unchanged or with reviewed additions.
- **AC-30 — Toolchain isolation:** Root `go.mod` stays Go 1.23 dependency-free; nested TUI build stays under `experiments/tui` with pinned Charm v2; no root migration to Go 1.26 or import of nested UI code into root.
- **AC-31 — Local-only safe mode:** A temporary-home walkthrough can inspect/stage/plan/status with no Internet, model calls, provider credentials, real-home writes or Git changes.
- **AC-32 — Responsive and performance evidence:** Tests record p95 in-memory UI navigation/render (100 agents), large-diff reachability, terminal resize behavior and startup delta; known benchmark limits/host context are reported, not promoted to broad guarantees.
- **AC-33 — Keyboard/accessibility:** Keyboard-only full safe journey, plain/ASCII/no-color rendering, static/reduced-motion behavior, readable color contrast and narrow-terminal confirmation recovery are exercised using headless/PTY or supported equivalent tests.
- **AC-34 — Shipped artifact parity:** A **selected** release candidate's companion `agentic-sdd-tui` and root `agentic-sdd` use compatible shared-operation contracts; the actual paired package/launcher is exercised in isolated mode before claiming support.
- **AC-35 — Security and no surprise charges:** No view/plan/save/apply/restore or error handling invokes a hosted model, reads provider secrets, silently writes `AGENTS.md`, changes default primary, runs `git push`, broadens agent permissions, or edits real home during tests.
- **AC-36 — Dependency gates:** Feature 014's required human review/Opus verdict, trusted-base release posture, runtime exercise and release authorization are reported honestly; absent gates cause safe feature unavailability/preview-only behavior, **never implicit permission to publish**.
- **AC-37 — Owner walkthrough:** One ordinary user path (choose private config → model change → save JSON → plan → apply → restore), a modified-file conflict and a first-time no-config path receive structured owner review in **temporary fixtures**. Distinguish owner inspection from an external novice usability study.
- **AC-38 — Completion packet:** Provide exact changes, current state, supported platform, tests run/not run, screenshot or PTY evidence, safety guarantees actually tested, known gaps, proposed release disposition and next human action. Green tests are not release approval.

### F. Supplemental r2 interaction acceptance (quality refinements, no extra product layer)

- **AC-39 — Four local Fleet views:** Within the existing four global routes, Fleet provides Overview, Profiles, Review Changes and Recovery as the only persistent local navigation views. Picker, source selection, details and confirmations are contextual. Full original Feature 015 functionality remains reachable.
- **AC-40 — Source/deployment checkpoint:** After source save and before apply, `Saved locally / OpenCode unchanged` is visible and `Preview OpenCode changes` is offered. After apply, actual engine result is shown without claiming runtime availability or Git commit.
- **AC-41 — Draft continuity:** Search/filter/focus/modal/resize/background read/return from another route preserves a valid staged draft and its accurate change count; leaving intentionally dirty state offers explicit discard.
- **AC-42 — Impact-first full review:** Both JSON source and OpenCode deployment have separate complete scrollable diffs, with removals/conflicts visible before routine changes; 100 ops and 1,000+ diff lines reach the final line.
- **AC-43 — Model route taxonomy:** Role, instance, provider route, exact model ID/variant, canonical source and optional family are distinct. A local catalog entry does not produce a subscription/auth/cost badge.
- **AC-44 — Focus and double-submit:** Text editing consumes printable shortcut characters; modals own focus; Enter repetition and terminal resize cannot result in duplicate or unseen affirmative write.
- **AC-45 — Scope and async correctness:** Late source/status/doctor/plan messages for another config, scope or generation do not clear drafts, move focus incorrectly, or replace an approved current plan.
- **AC-46 — One active write:** Fleet source-save, Fleet apply, Fleet restore and existing skill install/restore use one exclusive root transaction slot; an active write blocks other write submissions and accurately reports completion/exit behavior.
- **AC-47 — First-use no-config:** A new user sees three safe options (select existing local file, inspect example, use CLI guide), with zero reads of unselected private paths and zero source/agent writes.
- **AC-48 — Three-part recovery guidance:** Stale JSON, stale plan, unmanaged collision, drift and failed restore views state cause, unchanged effects and next safe action, without unsupported auto-adopt/force operations.
- **AC-49 — Layout and no-color:** Widths 110/80/60 and height 8/10/16 preserve Back/Cancel and warning; exact model/path and full diff are reachable; color-disabled and ASCII rendering preserve text/status meaning.
- **AC-50 — No hidden release activation:** If 014 final acceptance/trusted-base/runtime gate is missing, Fleet reads and fixtures may work but managed apply/restore remain explicitly unavailable; no new TUI shortcut overrides the gate.

## 9. Required negative and failure-path scenarios

The implementation plan must allocate fixtures/tests for at least:

1. First launch with no personal `fleet.json`: explanatory, nonblocking empty state; zero writes.
2. Personal JSON parse error/duplicate key/unknown policy field: actionable error; zero source/agent writes.
3. Nonexistent or symlink-escaped private path: no hidden read/write, no fallback to home search.
4. Exact provider ID entered but model not in local catalog: valid syntax may be staged with explicit **availability unknown**; do not claim authenticated.
5. Changing only model on a review agent: prompt and permissions unchanged; unresolved IR-06-01 remains visible.
6. Toggling disabled for a managed agent: correct proposed safe removal and backup; no unmanaged deletion.
7. Partial-source edit staged then Esc: explicit discard/back; source unchanged.
8. Private `fleet.json` externally modified before save: digest mismatch; reject write and retain recoverable choices.
9. Private source saved successfully: OpenCode install directory unchanged until separately confirmed apply.
10. Saved private source becomes invalid/changes between source save and fleet plan: plan fails; no apply.
11. Plan changed between preview and confirmation: stale-plan refusal, new inspection required.
12. Target file changes between preview and engine apply: lock/revalidation refuses without overwriting user edits.
13. Format-3/legacy provenance unknown: display unknown; no fabricated desired hash, no one-click adoption.
14. Install has unmanaged colliding agent: show exact conflict; no change to unmanaged file.
15. Apply interrupted after some staged writes: report rollback or recoverable failure accurately.
16. Restore backup missing/corrupt or current file drifted: refusal and safe next steps, no silent deletion.
17. Two rapid Enter presses on Apply: at most one accepted operation; lock and state control validated.
18. Cancellation/navigation/resize during running apply: no duplicate apply and no false cancellation claim.
19. `q`, `/`, `j`, `k` typed in model search or path field: interpreted as text, not shortcuts.
20. Resize while confirmation is open from 110 to 50 columns and short height: No/Cancel/warning remain usable.
21. A >1000-line diff with long Unicode paths: all material lines reachable and correctly attributed to a plan.
22. TUI launched in non-TTY/scripted context: existing CLI semantics preserved; no surprise full-screen mode.
23. Fleet build missing/014 not accepted: explanatory preview-only state, no production apply path.
24. `opencode models` shows a route, but authentication is not checked: label `cataloged/unknown`, not `authenticated`.
25. Reviewer path-isolation claim in an imported profile: TUI never reports it secure because model changed.
26. Git checkout is present but private metadata read is refused/unknown: no crash, no automatic network or Git commands.
27. CLI and TUI attempt same target update: shared lock and stale-plan checks prevent conflicting commits.
28. Ordinary skills installation without Fleet Manager ever opened: no reading a private fleet file or provider connection.
29. TUI reports success after the shared service returns a partial failure: **test fails**; UI must show actual failure.
30. Source-save operation attempted against a read-only or user-owned path with changed mode: refuse without damaging the existing file.

## 10. Verification strategy

| Layer | Tests / evidence | What it establishes | What it does not establish |
|---|---|---|---|
| **Pure UI-model tests** | Navigation, focus, state transitions, search, staged diff, default-No confirmation, error mapping | TUI contract under controlled messages | Filesystem transaction or model behavior |
| **Typed adapter tests** | Fake appops for status/plan/source save/apply/restore/doctor | Screens call proper requests exactly once; no shell/prose parser | Actual user file safety |
| **Root engine regression** | 014 fleet tests, source-edit extension tests, 009/skillsync/appops, root Taskfile gate | Domain/transaction behavior on fixture homes | Live OpenCode permissions on other versions |
| **Temporary-home integration** | 18 instances, 100 instance fixture, private source save vs install, drift, lock, rollback, restore | End-to-end intended filesystem effects in an isolated home | Real-home permission to install |
| **Headless Bubble Tea v2** | Update/View messages; layout size fixtures; expected semantic focus + viewport behavior | Interactive model correctness in tests | All terminals/screen readers |
| **PTY smoke tests** | Launch, narrow resize, keyboard search, preview/confirm, error recovery, quit | Real terminal signal/routing in controlled environment | Cross-platform compatibility not tested |
| **Packaging and paired binaries** | Nested TUI and root CLI build, release-candidate invocation via both modes; installed help/status | Artifact/binary contract on actually tested OS | Future releases, other OS |
| **Human visual/usability review** | Owner reviews end-to-end flow and accessibility fallback | Human acceptance of documented experience | A statistical study of novice users |

**Verification restrictions:** No real `~/.config/opencode`, private Git checkout, network, OpenCode model calls or paid provider use. Use isolated filesystem roots, local fixture JSON and fake runtime evidence. Preserve trusted-base ownership of any changed CI contract; no candidate-controlled privileged execution. Go root tests use `GOPROXY=off`/`GOWORK=off` and approved package gates where appropriate; nested Charm dependencies use the existing separately pinned module and Taskfile path. Never report missing Linux/macOS/manual checks as passed.

## 11. Delivery grouping for the eventual implementation plan

These are **possible batch subjects**, not authorized implementation batches and **not** a replacement for `plan.md` or `tasks.md`:

| Workstream | Main outcome | Dependencies |
|---|---|---|
| D1 — Read-only Fleet entry | Route, overview, profiles/search/detail, truthful status, tests | Accepted Feature 013 shell; inspect actual 014 typed APIs |
| D2 — Model-edit staging | Model picker, enable/disable staging, local validation, source diff | D1; verified 014 schema/renderer |
| D3 — Personal-source write seam | Small root `appops`/fleet source-save operation + equivalent CLI path, tested against private fixtures | D2; explicit approval of new source-write surface |
| D4 — Deployment & recovery | Full 014 plan/diff, default-No apply, drift, restore/doctor, state refresh | D1–D3; accepted/eligible 014 engine |
| D5 — Robustness & docs | UX refinements, PTY/resize/accessibility/large-fleet tests, paired artifact test and owner walkthrough | D1–D4 |

If Feature 014 release gates remain blocked, D4 should be implemented in **fixture/preview mode** only; do not bring a production apply path live merely to complete the TUI.

## 12. Deferred features and explicit exclusions

- **Not a new main TUI**: no new global destination, alternate theme, architecture migration or replacement of Feature 013 work.
- **Not a model execution manager**: no inference, provider account switching, auth setup, cost spending, quota monitoring or automatic retries.
- **Not a new fallback policy**: 009 continues to own delegation/fallback; 014 personal JSON owns model and enablement only.
- **Not a profile marketplace**: no remote package import, registry, signing claims, popularity/review ratings, remote JSON fetch or external marketplace installation. A future curated catalog can reuse the same read-only picker.
- **Not a canonical policy editor**: no editing prompt bodies, roles, permission blocks, nested delegation, security exception rules or `AGENTS.md` through the model editor.
- **Not profile migration/import**: 014 FR-15 deferred; no automatic adoption of unmanaged machine agents.
- **Not an OpenCode launcher/primary switcher**: install/configure only; changing default agent/session is a distinct future decision.
- **No broad new platform promises**: maintain actual shipped/tested macOS/other target matrix and OpenCode version boundaries; do not claim Linux packaging based on a successful root Go build.
- **No Git automation**: no private repo clone/commit/push/pull, no Git credential collection or remote synchronization.
- **No unreviewed dependency expansion**: optional Huh accessibility or other components may be evaluated within existing nested-module constraints, but should not be added without user-value and version/maintenance assessment.
- **No new permanent approval/task ledger**: the TUI does not create its own SDD authority; Feature 014 and repo SDD records remain canonical.

## 13. Risks, evidence, and release gates

| Risk | Design response / release gate |
|---|---|
| 014 still awaiting Opus/human review and trusted-base adoption | No production Fleet apply/restore until dependency accepted and release posture established; mock-only preview is permitted for development with authorization. |
| Public docs/handovers are different snapshots | At planning time identify exact checkout SHA, uncommitted files, accepted 013 continuation and latest 014 state. Don't rewrite historic records. |
| TUI source editor becomes second configuration manager | Shared typed root application service; CLI parity and engine tests; no local filesystem side effects in screens. |
| Model-only changes silently widen tools | Renderer preserves canonical permissions; surface engine permission conflict, not a locally computed 'safe' badge. |
| False auth/provider availability status | `configured/cataloged/unknown` labels; authentication only from an explicitly accepted evidence source. |
| Double-confirm or double-dispatch bug | Explicit two write boundaries and one dispatch per operation; keyboard/focus and rapid Enter negative tests. |
| Source JSON drift after TUI opened | Compare-against-read digest and show full diff; stop on stale. |
| Long preview clipped | Reuse/extend Feature 013 scrollable viewport with reachable content and warnings. |
| Existing 013 UX regresses | Reuse shell/theme/preferences/contract tests; no new 5th global tab. |
| New root deps or migration break CI | Root Go 1.23 stdlib stays; nested Go 1.26 Charm v2 remains isolated. |
| Real-home path accidentally hit during tests | Explicit t.TempDir/isolated HOME; no real home/network/model calls. |
| Release artifact mismatches source | Paired CLI/TUI candidate-specific test, checksums/packaging check and separate publication approval. |

**Release readiness:** Successful source implementation and tests are insufficient if the 014 dependency gate remains unresolved or if the actual paired candidate binary was not exercised. The final report must say *no-go* or *preview-only* where appropriate. No separate design work grants tag, cask/tap update, CI trusted-base adoption or production installation authority.

## 14. Required completion and owner review packet

1. Exact checkout SHA and pre/post dirty/untracked changes.
2. Feature 013 shell/tokens and Feature 014 engine versions actually integrated, with evidence of acceptable dependencies.
3. Final UX navigation/screens, keyboard map, accessibility modes and screenshots/PTY recordings **only if produced** (do not invent evidence).
4. Exact typed operation/API changes, with proof that screens do not mutate files and CLI parity is preserved.
5. Model/enablement edit flows and two-stage source-save versus OpenCode-apply consent, including stale conflicts.
6. Effective status provenance; blocked/unknown auth/provider/permission states.
7. Source and installed backup/restore scenarios including rollbacks and interrupted/error cases.
8. Root, nested TUI, relevant regression, PTY and packaging checks, with pass/fail/blocked/not-run and selected candidate identity.
9. Remaining Opus/trusted-base/runtime release and IR-06-01 limits; known platform exclusions.
10. Recommended go/no-go and the **next action requiring explicit human authorization**; stop for review.

## 15. Approval

**Specification state:** DRAFT only.
**Combined spec/plan/tasks approval:** Not recorded.
**Implementation authorization:** None.
**Real-home apply/private-source modification:** Not authorized by this document.
**Hosted model/API/paid usage:** Not authorized.
**Trusted CI/release/tag/tap publication:** Not authorized.
**Existing 014 approval/batch history:** Preserved, not superseded or reopened.
**Next SDD action:** Inspect current repo and exact available 015 slot; review this spec against accepted 013/014/009 contracts; draft a small `plan.md` and `tasks.md`; reconcile and seek one combined human approval before selecting a bounded batch.

## 16. Source notes

**Owner-provided:** `2026-11-11_014b.md` (Feature 014 handover, 2026-11-11); previous 014 draft and 013 Fleet Manager concept (historical design source).
**Public current repository via Exa (inspected 2026-11-11):** [`docs/tui-architecture.md`](https://github.com/nawodyaishan/agentic-sdd/blob/main/docs/tui-architecture.md), [`specs/013-unified-tui-experience/spec.md`](https://github.com/nawodyaishan/agentic-sdd/blob/main/specs/013-unified-tui-experience/spec.md), [`specs/014-agent-fleet-model-overrides/spec.md`](https://github.com/nawodyaishan/agentic-sdd/blob/main/specs/014-agent-fleet-model-overrides/spec.md), [`experiments/tui/go.mod`](https://github.com/nawodyaishan/agentic-sdd/blob/main/experiments/tui/go.mod). Public HEAD can differ from the owner's active unmerged local tree; reinspection is required before implementation.
**Research via Exa and Context7:** [Bubble Tea v2 upgrade](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md); [Bubbles v2](https://github.com/charmbracelet/bubbles); [Huh v2](https://github.com/charmbracelet/huh); [Bubble Tea testing](https://charm.land/blog/teatest/); [chezmoi](https://www.chezmoi.io/user-guide/setup/); [OCX](https://github.com/kdcokenny/ocx); [Vercel CLI guidelines](https://vercel.com/design/cli); [OpenCode agents](https://opencode.ai/docs/agents/), [models](https://opencode.ai/docs/models/) and [configuration](https://opencode.ai/docs/config/). External examples are architecture patterns, not proof of safety or performance in this repository.

### 16.1 r2 review/provenance addendum

r2 is based on `015-fleet-manager-tui-ux-review-2026-11-11.md` (provided separately as an advisory artifact), with public source inspection of `experiments/tui/{app,app_screens,ops_host,viewport,theme}.go` and external Exa/Context7 patterns: [Lazygit official guide](https://lazygit.dev/docs/guide/), [K9s commands](https://k9scli.io/topics/commands/), [Bubble Tea v2 upgrade guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md), [Bubbles v2](https://github.com/charmbracelet/bubbles/blob/v2.0.0/README.md), and [Huh v2 accessible forms](https://github.com/charmbracelet/huh/blob/v2.0.0/README.md).

**Boundary:** These are source and specification observations; no unpushed/local Fleet TUI implementation, screenshots or PTY runtime were available, and no validation or release action was performed. This document is a proposed revision and not an authorization to amend a canonical approved document without the project's SDD process.
