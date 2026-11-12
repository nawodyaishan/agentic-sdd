# Feature 015 — Fleet Manager TUI: Product UX and Implementation Review

**Date:** 2026-11-11
**Status:** Read-only expert assessment and proposed refinements; **not** approval or implementation authorization.
**Scope inspected:** User-provided proposed `015-fleet-manager-tui-experience-spec.md` (21 FR, 38 AC), the 2026-11-11 Feature 014 handover and specification, publicly retrievable main-branch TUI/architecture sources, current external Go TUI reference documentation.
**Not inspected:** Any unpushed/local `fleet-manager-tui-experience` branch, screenshots of a new Fleet screen, user private Git files, working OpenCode home, runtime model execution, or real interactive tests. A public URL for a `fleet-manager-tui-experience` branch was not retrievable. Findings below are **spec/design review plus source-informed integration analysis**, *not* a completed code audit of an unseen branch.

## 1. Executive verdict

**Retain and refine Feature 015 rather than redesign it.** The existing spec has strong safety boundaries and correctly relies on Feature 014. The main opportunity is **reducing the number of concepts/screens a newcomer must juggle** while improving the expert model-switching workflow. It currently describes eight local Fleet routes, multiple state vocabularies, and two distinct writes. Those capabilities are necessary, but should appear as **four main Fleet views, progressive detail, and a persistent draft/deployment status indicator** rather than eight competing tabs or a long wizard.

**Highest-value improvement:** make the product answer three questions immediately:

1. **What am I using?** Agent instance → canonical role → selected model/provider → enabled/managed/runtime-known facts.
2. **What will change?** Unsaved source changes versus saved-but-unapplied OpenCode changes; full and risk-focused diff.
3. **What should I do next?** Save config, preview deployment, apply eligible plan, resolve a conflict, or stop.

**Non-negotiable:** keep personal `fleet.json` saving separate from OpenCode apply; both remain default-No, separately authorized; no new fallback/permission/model-execution system.

## 2. Verified source-informed integration findings

| Source | Observed pattern | Concrete implication |
|---|---|---|
| `experiments/tui/app.go` (public main) | Four root `Route` values: Home, Install, Explore, Activity; App owns shared focus/navigation and child models | Do **not** add `RouteFleet` as fifth top-level area; add `FleetModel` as Install child and one Home affordance. |
| `experiments/tui/app_screens.go` (public main) | `installSection` currently enumerates Hub, Install, Backups; root builds immutable `WriteRequest` after validated preview | Add an Install Fleet child; retain preview-token and root-dispatch pattern. No direct screen mutations. |
| `experiments/tui/ops_host.go` (public main) | `WriteKind` currently distinguishes Install and Restore; OpsHost owns the single-write slot, progress and result | Extend **one existing** transaction host for source-save/fleet apply/fleet restore; do not create a parallel Fleet writer or button handler. |
| `experiments/tui/viewport.go` (public main) | Bounded viewport, reserved footer/status notice, single modal compositor, 40×8 minimum usable geometry, wrapping/scrolling | Fleet warnings/confirmations should use the same compositor. Avoid adding a permanent status strip that consumes the last usable row in short terminals. |
| `experiments/tui/theme.go` (public main) | Existing dark-only Catppuccin-Mocha-inspired tokens, truecolor/ANSI variants, contrast documentation | **Keep** palette and semantics. Improve hierarchy, whitespace and labels; do not re-theme or color-code provider identity as if verified health. |
| `internal/fleet/` (public main) | Fleet parse/resolve/render/plan/apply/restore/doctor package exists | Reuse exact typed results and recorded outcome semantics; no second parser, source of truth or plan hash in TUI. |
| `015` proposal | Separate source-save then OpenCode deployment, extensive negative cases and CLI parity | Good safety architecture; improve copy/flow/feedback so users understand why there are two approvals. |

**Version caveat:** public main and the owner's unmerged work may differ. Reinspect exact checkout + Feature 014 approval/Opus gate before implementation. No live UI screenshots for the new Fleet feature were provided, so visual findings below are design requirements, not claims of observed visual defects.

## 3. Prioritized UX defects and improvements

| Priority | Gap in current Feature 015 draft | Recommended design change | Why it matters |
|---|---|---|---|
| P0 | Eight local routes read like eight navigation destinations | Limit Fleet top-level local navigation to **Overview, Profiles, Review Changes, Recovery**; use contextual drill-down and overlays for Model Picker, Source, Help, Backups, Diagnostics | Reduces navigation burden while keeping all required capabilities. |
| P0 | Source JSON state and deployed agent state can be confused | Persistent two-state mini-summary: `Personal config: Unsaved / Saved locally / Changed externally` and `OpenCode: Current / Needs apply / Drifted / Not checked` | Makes the two-write contract understandable. |
| P0 | Status vocabulary has many orthogonal terms | Use three separate dimensions: **Source**, **Installed**, **Runtime**. Never use a single ambiguous green `Ready` | Avoids false assurances about models and installed agents. |
| P0 | Saving configuration and applying deployment may feel like one long process | After source-save show **“Saved locally. OpenCode unchanged.”** and one clear next step `Preview OpenCode changes`; after apply show **actual** result + backup | Keeps user confident about what actually changed. |
| P0 | Full diff requirement risks a dense default screen | **Impact-first review**: number of creates/updates/removals/conflicts and source scope first, then file list and full diff pane; always make the complete diff accessible | Fast understanding with no hidden risk. |
| P0 | Existing OpsHost supports only install/restore write kinds | Extend existing typed single-write lifecycle and read generation tags; source-save and deployment cannot run together | Prevents race-induced double writes and conflicting busy states. |
| P0 | UI can show obsolete results after switching source/profile | Bind every read to `{config path, source digest, scope, generation}`; discard stale responses; never overwrite staged edits | Necessary for safe TUI async behavior. |
| P1 | Model picking can overwhelm 18 instances and multiple providers | One searchable model picker with groups `Current`, `Known local assignments`, `Locally cataloged`, `Enter exact ID`; tags describe source, **not** rank or entitlement | Improves speed while avoiding invented price/availability claims. |
| P1 | `9router` route vs Claude/Gemini family can look like multiple providers | List **provider route** separately from **model family** (only when known); `openai/*` not necessarily ChatGPT-subscribed | Avoids wrong purchase/auth inference. |
| P1 | Agent list may overuse long IDs, model routes and status markers | Two-level list `Role / Instance`, with an expandable details panel and compact one-line summary; filter by Role, Provider, Changed, Problem | Scales to 18 and 100 instances with lower visual noise. |
| P1 | First run requires a private JSON path, with uncertain empty state | Show three clear options: `Use an existing fleet.json`, `View example (read-only)`, `Open CLI setup guide`; no inferred private path or automatic clone | Helps newcomers without violating source security. |
| P1 | Dirty edits could be lost through Esc, navigation or refresh | Persistent `N unsaved edits`, visible `Review edits`/`Discard` actions, confirmation when leaving dirty context, no implicit disk write | Prevents user surprise. |
| P1 | Conflict/error list may be technically correct but hard to act on | Each error should have **What happened / What remained unchanged / Next safe action**; for `stale plan` offer `Replan`, not `Retry apply` | Actionable recovery without unauthorized automatic resolution. |
| P1 | Very long model IDs and paths can dominate layout | Middle-elide in list only; full exact ID in scrollable details/preview; preserve copyable redacted text if supported | Works at 80/60 columns. |
| P1 | Focus and mode interactions risk `q`/`/` or double Enter misfires | One focus owner, explicit editing mode, mode-specific help/footer; disable affirmative action on resize too-small | Complements existing 013 tests. |
| P2 | Repetitive updates across many agents may become slow | Add a **session-level draft summary** (`3 edited`, `1 disabled`), clear filter `Changed`, and support multi-instance staging without bulk apply surprises | Power-user speed without new provider presets. |
| P2 | Onboarding may duplicate full app tour | One 3-step skippable inline hint for Fleet only: Choose source → Change model → Save & apply separately | Progressive disclosure. |
| P2 | Additional color palette request could cause theme drift | Reuse existing single dark theme; make warning and focus text/shape dominant, provider names neutral | Matches Feature 013 and its contrast work. |

## 4. Proposed information architecture

Keep **four global destinations** (`Home`, `Install & Manage`, `Explore`, `Activity & Help`). Under Install & Manage, add **Fleet Manager** alongside existing Install and Backups.

Within Fleet Manager, expose **four local views** only:

| View | Purpose | Contextual views/actions |
|---|---|---|
| **Overview** | Status of selected source, config save state, managed agents, last relevant outcome, next action | Change local source, inspect diagnostics, open profile |
| **Profiles** | Search, filter, inspect/change model and enablement | Model Picker overlay, Role Detail, Config Source dialog, `Changed` filter |
| **Review Changes** | View pending source diff or current deployment plan; act on one safe write at a time | Source Save confirmation; plan file/diff; Apply confirmation; results |
| **Recovery** | Drift, conflicts, legacy provenance, managed backups, safe restore | Diagnostics details, restore preview and confirmation; CLI help |

Do not create tabs for a page that is only used as a confirmation, an error overlay or a model picker. Preserve selections/scroll/filter state when returning from those contexts.

### 4.1 Optimized desktop/wide layout (illustrative only)

```text
AGENTIC-SDD / Install & Manage / Fleet Manager
─────────────────────────────────────────────────────────────────────────────────
Source: private/fleet.json · Config: 2 UNSAVED · OpenCode: not updated
[ Overview ]   [ Profiles ]   [ Review Changes 2 ]   [ Recovery ]
─────────────────────────────────────────────────────────────────────────────────
Agents / search [/]                │ Selected: sdd-implement-1
[Role: All] [Route: All] [Changed] │ Role       Implementation
> sdd-implement-1    Go           │ Model      opencode-go/deepseek-...
  sdd-implement-9r-1  9Router     │ Source     preset:sdd-implement-1
  sdd-lightweight     Console     │ Config     enabled; changed in draft
  sdd-openai-architect OpenAI     │ Installed  current / not checked
                                  │ Runtime    NOT VERIFIED
                                  │ [Change model]  [Disable]
─────────────────────────────────────────────────────────────────────────────────
Draft: 2 model changes · source not saved · no OpenCode files changed
Enter details  / search  Tab focus  ? help  Esc back
```

**Copy semantics:** “Config saved locally” never means “Git committed”; “OpenCode applied” never means “agent ran” or “authenticated”. Avoid status colors without adjacent text.

### 4.2 Narrow layout

At width under 110 columns, prefer **list → details drill-down** instead of squeezing long model identifiers into a permanently split pane; Feature 013's root navigation may remain compact. At 60–79 show one pane at a time; below 60 preserve safe Back/Cancel and resize guidance. A tiny terminal never accepts an unseen `Yes` action. The full exact ID/diff remains available via scrolling/detail, not by clipping.

## 5. Primary flow: economical model switch

1. **Overview:** local source is known, no hidden home scanning; show current managed and staged status.
2. **Profiles:** select `sdd-implement-1` and see role/source/current model.
3. **Change model:** search locally known routes (or enter exact ID). Selecting a model **stages in memory only**. Show old → new route and `Runtime not verified`.
4. **Review Changes:** stage shows `1 model change`, `no permission change`, `source not saved`, zero installed changes. Full JSON diff is accessible.
5. **Confirm Save** (default No): shared typed operation writes the private `fleet.json` only, with last-observed digest CAS and backup; no Git actions.
6. **Clear checkpoint:** `Personal config saved locally. OpenCode unchanged.` Provide `Preview deployment` and `Back`.
7. **Deployment Review:** generate a **new** 014 plan from saved JSON; display exact file changes, removes first, conflicts, paths and complete diff. No automatic apply.
8. **Confirm Apply** (default No): root OpsHost dispatches once; engine revalidates plan/scope/target lock.
9. **Results:** state actual changed/unchanged/failed/rollback info + backup ID; `Installed config updated` but **not** `model authenticated/running`.
10. **Recovery:** safe managed restore previews exact effects; Git revert remains a separate action that does not change installed OpenCode files.

The user may leave after step 6. The next time Fleet opens, `Config saved locally / OpenCode changes pending` should be visible based on authoritative refreshed state, not optimistic in-memory markers.

## 6. UX contracts to enforce in Feature 015

### 6.1 State projection (no UI-derived security decisions)

| Dimension | Display values / source | Do **not** infer |
|---|---|---|
| **Personal JSON** | `missing`, `invalid`, `loaded`, `dirty draft`, `saved locally`, `changed externally` (compare actual source + session draft) | Git committed/pushed |
| **Installed agent** | `not installed`, `managed current`, `pending apply`, `drifted`, `unmanaged`, `legacy unknown`, `disabled (not managed-installed)` (014 typed engine) | Agent authenticated, primary active or safe sandbox |
| **OpenCode runtime** | `not checked`, `catalog visible`, `verified by supported local diagnostic` if accepted source | Subscription entitlement or guaranteed model success |
| **Operation** | `not started`, `previewed`, `saving`, `saved`, `applying`, `applied`, `no-op`, `blocked`, `failed`, `rollback incomplete` | That a busy spinner means work succeeded |

**Rule:** The UI can own view/draft state, but authoritative installed/permission/plan state comes from `internal/fleet` through `appops`. No UI guesses about file ownership, hashes, locks or provider billing.

### 6.2 Async/result rules

- Every request carries scope and generation; late results for previous paths/homes/instances cannot replace new views.
- Do not let a delayed `status` or `doctor` response reset staged model edits.
- A file-change event while the user edits marks source as stale but does not destroy their draft; next save must fail/review.
- No plans are re-used after source-save, selected-scope change, external edit, renderer/source digest change or apply result.
- A write uses the **existing single global OpsHost slot**. Moving between screens, rapid Enter, resize and stale result cannot start another write.
- While a write is active, show progress/outcome without promising interruption or cancellation if the engine cannot abort safely. Use the established Feature 013 exit-after-write contract.

### 6.3 Source editing decision

The draft spec proposes native strict-JSON source save through a new shared use case. This is acceptable **only with separate explicit approval for the root CLI extension**, atomicity/backups, compare-and-swap and config validation. If that additional capability is deferred, offer `Open in external editor` through the existing vetted opener mechanism and **do not fake a native Save action**. On returning from an external editor, reload/revalidate; never auto-apply. Note `opencode` profiles are not the user's private JSON file.

### 6.4 Safety and confirmations

- Default `No` for **both** personal JSON save and fleet apply, and separately for restore.
- Confirmation shows **scope + exact target root + relevant create/update/remove counts**; disabling an installed agent highlights managed-file removal.
- Conflicts and missing/unknown provenance block apply; no magic `Fix all`, `Adopt`, `Override` or `Force` button (014 FR-15 deferred).
- A `No-op` is neutral, not success-colored and not an error.
- Source-save success ≠ deployed success; deployed success ≠ runtime model verification.

### 6.5 Accessibility / theme

- Reuse existing dark-only `agentic-dark` palette (no light mode). Preserve ANSI-16 best-effort limits, plain/ASCII modes and reduced-motion.
- Focusable controls show a text label, visible selection marker and focus indication; no color-only interpretation.
- Tab/Shift+Tab cycle **visible** actionable targets; focus does not escape a modal until dismissal; `q`, `/`, `j`, `k` type literally inside inputs.
- Show only relevant contextual shortcuts in footer; `?` opens full current-view help.
- Never add another permanent badge/status cluster that consumes scarce screen height on short terminals.
- Include a documented CLI/noninteractive fallback for screen-reader users; an accessibility-supporting library is not proof that the finished TUI is screen-reader accessible.

## 7. Existing code integration checklist

The coding agent should confirm these exact seams in the target checkout before editing:

- `experiments/tui/app.go`: root `App`, four `Route`s, focus and lifecycle; add optional Fleet child state without changing top-level routes.
- `experiments/tui/app_screens.go`: add Install-section Fleet entry; preserve existing Install/Backups behavior, `StartWriteMsg` dispatch pattern and breadcrumb consistency.
- `experiments/tui/ops_host.go`: extend `WriteKind`/request dispatch/result types for *authorized* source save, fleet apply, fleet restore; retain one global active write, flush and exit contracts.
- `experiments/tui/viewport.go`/`modal.go`: reuse existing bounded layout, modal confirmation, safe no-Yes under too-small dimensions.
- `experiments/tui/theme.go`: reuse existing semantic tokens and dark-only contrast tests.
- `internal/appops/`: add the smallest typed fleet source/status/plan/write adapter, shared with CLI; never pass arbitrary command strings or parse CLI prose.
- `internal/fleet/`: reuse accepted 014 engine; no duplicated validation/hashing/installer code. `fleet.json` source-save needs its own accepted, testable operation if implemented.
- `experiments/tui/*_test.go`: headless `tea.KeyPressMsg`/resize tests matching the **pinned Bubble Tea v2 version**, rather than mixing v1 API types.

## 8. Test-first prioritized acceptance additions

These are suggested **amendments to existing Feature 015 acceptance**, not a new feature-numbered phase:

1. **Journey**: 18-instance fixture can complete one model edit → source-save → fresh plan → apply → inspect result in ≤4 Fleet local navigation views, and neither write occurs before its distinct confirmation.
2. **Truthful checkpoint**: after source-save and before deployment, the UI unambiguously says `Saved locally / OpenCode unchanged` and still offers `Preview deployment`.
3. **Staged persistence**: search, filter, modal, resize, route-back and late async results preserve dirty draft and its accurate edit count, or explicitly request discard.
4. **Prompt/permissions**: model-only variant makes no role/prompt/permission changes, and the preview shows this fact from engine results rather than a UI-only guess.
5. **Two-step review**: source diff and deployment diff are distinct and independently scrollable to their last lines; no more than one confirmation is active at a time.
6. **Diff reachability**: 1,000+ lines, 100 ops, very long paths/models can be browsed in the actual viewport; risk/removal rows appear before ordinary updates.
7. **Model picker**: exact `provider/model#variant` always available, local-catalog match separately labeled; no subscription/auth/price guarantees.
8. **Responsive**: 110/80/60 columns and height 8/10/16 tests preserve Back/Cancel/focus and disallow hidden Yes.
9. **Async discipline**: stale generation from prior config cannot change selected source or clear draft; double Enter cannot submit a second write.
10. **Conflict action**: stale JSON and modified installed-file tests show *what happened / what stayed unchanged / next step* without a write or auto-adopt.
11. **No config yet**: first-use example/tutorial works read-only without auto-discovering private Git sources.
12. **OpsHost parity**: Fleet writes and existing Install/Backups share the single write slot; an Install transaction in progress prevents Fleet apply.
13. **CLI parity**: any new source-save operation is available through a documented root CLI typed route, with same stale digest/backups validation; no screen-exclusive file mutation.
14. **Release truthfulness**: if 014 is still gated, only read-only/fixture-preview is available and the TUI never claims a production deploy capability.
15. **Regression**: root CLI Go1.23 stdlib still dependency-free; nested Go1.26 Charm v2 pinned; B8–B13 old TUI UI/PTY tests remain green.

## 9. Suggested implementation order (bounded, not batch authorization)

| Priority | Scope | Rationale |
|---|---|---|
| 1 | Read-only Fleet entry, source/status, profile browser and responsive layout | Immediate user value, no write risk |
| 2 | Draft model picker, dirty-state indicators, source diff | Reliable user mental model |
| 3 | Approved typed source-save shared use case, CLI parity and backup | Safest way to enable editing; separate approval needed |
| 4 | Fresh deployment plan, impact-first diff, apply and restore through OpsHost | Reuse trusted engine and avoid double dispatcher |
| 5 | Error recovery, negative PTY/generation tests, accessibility, packaging/owner visual review | Release quality and no regression |

No step above is permission to write real `~/.config/opencode`, private Git or release a binary.

## 10. External design sources (Exa + Context7)

- [Lazygit official user guide](https://lazygit.dev/docs/guide/): focusable panels, contextual key help, diff-oriented workflows. **Borrow** discoverability and focused panels, not Git actions.
- [K9s commands](https://k9scli.io/topics/commands/): search/filter, current-view keyboard guidance. **Borrow** predictable navigation and concise help.
- [Charm Bubble Tea v2 upgrade](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md): declarative view output, async command/message handling; match pinned version.
- [Charm Bubbles v2](https://github.com/charmbracelet/bubbles/blob/v2.0.0/README.md): list, key binding/help, viewport for large diffs and focus-state composition.
- [Charm Huh v2](https://github.com/charmbracelet/huh/blob/v2.0.0/README.md): validated inputs and optional accessible mode; use only if justified and inside existing nested module.
- [chezmoi](https://www.chezmoi.io/user-guide/setup/): source/diff/apply mental model, not automatic Git sync.
- [Vercel CLI design](https://vercel.com/design/cli): explicit no-op/change/error and stale plan safeguards.
- [OpenCode agent docs](https://opencode.ai/docs/agents/): agent file/state semantics; installed ≠ selected primary/entitled.

**External-design qualification:** These are product patterns, not claims that the Agentic-SDD Fleet feature is implemented or verified. Context7 Bubble Tea query results can mix v1/v2 snippets; use pinned `charm.land/bubbletea/v2` and `tea.KeyPressMsg` for the nested codebase, not old `github.com/charmbracelet/bubbletea` APIs.

## 11. Approval/next action

This review is **read-only advisory work**. Before editing Feature 015's canonical spec, compare it with the current checkout, actual approval/continuation state, and the owner's latest decisions. Prepare a focused UX revision to that **same** feature and request reconciliation/approval per current SDD policy. **No new Feature 016 is warranted merely to optimize these views.**
