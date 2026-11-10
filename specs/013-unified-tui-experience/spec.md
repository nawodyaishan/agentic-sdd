# Feature 013 — Unified TUI Experience and Animated Branding

**Revision:** r3 — review reconciliation, 2026-11-10; approval scope is recorded below.  
**Outcome:** Extend the Feature 012 catalog spike into a coherent, accessible, preview-first management interface.  
**Sources:** Owner's redesign guide and TUI best-practices brief in this session; [012 handover](../../docs/handovers/2026-11-10_012.md); [012 specification](../012-cli-tui-remediation/spec.md); [012 ledger](../012-cli-tui-remediation/tasks.md); [architecture evidence](../../docs/tui-architecture.md). Product compass: [vision §§3, 6–7](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md).

## 1. Context and boundary

Feature 012 delivered formatting remediation and a separately runnable, read-only Bubble Tea v2 spike. Its batches remain awaiting human review. This feature drafts the next interaction surface; it neither accepts those batches nor extends their execution authorization. Inspect current Git state rather than assuming historical uncommitted changes remain present. Preserve any work found.

The inspected catalog already supports `/` filtering and destination badges, but lacks purpose/example metadata, application navigation, integrated write workflows, and responsive detail panes. Actual inventory counts depend on source/options: the current embedded inventory has 19 skills and 14 base profiles, or 18 profiles with Free-first presets. Screens must derive counts, not encode the supplied mockups' numbers.

Primary users are newcomers installing their first skills and experienced engineers managing updates and recovery. Both retain the explicit CLI and human control over writes.

## 2. Scope

### Included

- Persistent application shell with four destinations: **Home**, **Install & Manage**, **Explore**, **Activity & Help**. Backups, restore, and optional OpenCode pack settings live under Install & Manage.
- Locally derived dashboard with installation coverage, meaningful next actions, version, and skippable onboarding; a non-blocking automatic version check on first main-menu entry is the sole automatic network exception. No authentication checks, model connections, agent execution, or telemetry.
- Responsive categorized catalog with search over exact ID, display name, purpose, kind, and category; stable selection/filter/scroll state; rich details and preview navigation.
- Integrated preview, install/update, backup browsing, restore preview/apply, existing log viewing, getting started, help, version, automatic main-menu version checking and explicit retry through shared Go operations; clickable online README guide with a keyboard browser-opening action.
- Client selection for installation, optional OpenCode pack and dependent Free-first option, source/home review, exact preview, default-No confirmation, and honest result/backup summaries.
- Narrow additive engine APIs for structured operation results and unambiguous client selection, preserving existing wrappers/output and backup format; subset installations must remain restorable through the supported restore command.
- Consistent branding: compact ASCII logo with a brief animation, violet/cyan theme, semantic colors, icons and restrained emoji. The owner's explicit branding request supersedes the supplied guide's recommendation to omit logo animation; safety and accessibility still take precedence.
- Personal attribution **nawodyaishan** alongside product branding, and Homebrew interactive bare-command TUI by default, with explicit CLI and non-TTY compatibility retained.

### Excluded

- Per the owner's B2 steering: harness/OpenCode-harness improvements and standalone `tests/harness` validation are experimental/nonshipping and outside this UI feature's execution scope. Do not extend check-runner/evidence/runtime machinery or run OpenCode-harness workflows. The existing generic core Go packages have no OpenCode imports/references and are not being changed; preserve required repository coverage rather than silently filtering it. Harness-related edits/checks already made before the steering remain identified separately for human scope review, not new shipping capability or live trusted-base adoption evidence.
- Rewriting the installer, backup format, rollback, or restore semantics; a new agent runtime, plugin system, provider configuration/authentication, automatic updates, or executing suggested shell commands.
- Root module external dependencies, root toolchain migration, or moving Charm into root `internal/tui` in this feature. Retain nested-module isolation.
- Enabling or validating fenced Feature 005/006/007 native, admission, profile/dispatch/pilot, runtime-certification, or historical migration work. Optional shipped pack installation is file distribution, not runtime activation.
- Installing skills individually: the core skill set remains the supported installation unit. Client choice is not a per-skill picker.
- Real-home apply, publishing, tagging, pushing, or deploying Homebrew formula changes without separate execution authorization. Developing the requested Homebrew default is in scope; draft approval is not release or live-operation consent.

## 3. Requirements

### FR-01 — Shell and interaction

Home must visibly offer “Set up / Install Agentic-SDD.” Navigation persists across screens, with one contextual footer and an unambiguous focus indicator. Use arrows/j/k for selection, Enter to open, Esc to back/close, `/` to search, Tab to cycle visible focus targets, `?` for help, Home/End for list bounds, and q to quit navigation screens. Text-input focus consumes printable keys, including q/j/k. Contextual help explains Tab's current targets; categories are ordinary focusable controls, not a conflicting second Tab meaning.

### FR-02 — Responsive, accessible presentation

Initial layout targets: ≥110 columns sidebar/list/details; 80–109 compact navigation/list/details; 60–79 single list with separate details; <60 minimal usable view or explicit resize guidance. Height matters too: collapse decoration before essential controls, then offer safe resize/back/quit guidance if the viewport cannot show a confirmation. Resizing preserves state and never confirms an operation. Long paths, warnings, and exact commands remain fully available through wrapping/scrolling, never silently truncated or rewritten.

Keyboard-only operation, terminal-native light/dark backgrounds, reduced colors, `NO_COLOR`, ASCII and static/reduced-motion modes are required. Icons/color are supplementary to text. Preserve a plain explicit CLI alternative for screen readers, scripts, pipes, and non-TTY use; do not claim screen-reader certification without evidence.

### FR-03 — Catalog and truthful status

Offer kind/origin filters (all, core skills, pack skills, profiles, optional presets) plus purpose-oriented categories derived from maintained teaching metadata. Do not fabricate categories for absent content. Show readable display name and exact ID, purpose, best use, example prompt, origin, destinations, per-client file status, caveat, and contextual next action. Missing metadata has a neutral fallback, not invented explanations.

For agent profiles, additionally show the declared agent mode, provider, primary model, and every documented fallback, plus a policy note that the written ladder is manual guidance rather than automatic failover. These are configuration facts about templates, not proof of credentials, quota, availability, authentication or runtime use; the display must never claim a model actually ran. Skills/concepts that have no routing must omit routing fields or state that they are not agent profiles.

Separate “installed (matches source),” “installed (modified),” “template available — not installed,” and “unavailable/refused.” Modified content is not automatically a conflict. Unsafe/refused destinations explain why execution is unavailable. Profile presence never means running, authenticated, enabled, sandboxed, or approved. Search/empty-result handling preserves or safely resets selection by stable kind+ID, not row number.

### FR-04 — Installation journey

Choose supported coding clients → optional integrations → review source/home/destinations → preview exact impact → default-No confirmation → apply via existing engine → result, backup/log reference, next prompt or Home. Zero clients is invalid. An explicit empty selection must never mean “all”; omission in historical CLI calls still means the complete set. Unknown IDs fail before writes. Free-first requires the OpenCode pack and is clearly a template preset, not an availability/cost guarantee. OpenCode profile destinations must be stated separately from core client selection.

Preview is read-only and reports actual additions/replacements/unchanged/refusals with inspectable paths. Changing any source, home, client, pack, or preset option invalidates confirmation. Revalidate before apply; changed destinations/source require a fresh preview and confirmation. Prevent duplicate submissions. Existing replacement backups, symlink refusal, no-op behavior, rollback, unrelated-file protection, and log privacy remain intact.

### FR-05 — Backups and recovery

List backups with deterministic ordering and visible timestamps/identity; opening one is never a restore. Validate selection, display exact overwrite/removal impact, and require a separate default-No confirmation before restore apply. Stale preview or changed backup/destination invalidates consent. Missing, empty, corrupt, refused, and unavailable backup states explain recovery without offering unsafe execution. Results distinguish successful, failed, incomplete, rolled-back, and cancelled outcomes only when the operation supplies that evidence.

Backups from client-subset installations must pass existing home/target identity validation and restore only their recorded changes. Keep the manifest's full canonical core target list as home identity, independent of selected write targets; entries describe actual changes. Preserve optional profile identity, legacy compatibility and unrelated/unselected files. Return backup identity and actual outcome through structured engine results, including partial failure information, rather than parsing display text. Existing public wrappers and transcripts remain compatible.

### FR-06 — Activity, help, onboarding, and operation coverage

Read existing installation/restore logs with bounded loading and scroll; show a contextual empty state when none exist. About exposes version/build information; help links current commands and local getting-started content. Onboarding reuses the installation journey, is skippable, and ends with a copyable first `agentic-sdd-router` prompt, not automatic clipboard or agent execution. Show unsupported per-skill choices as unavailable rather than simulating support.

On first TUI main-menu/Home entry, automatically check GitHub for a newer CLI version asynchronously, once per process session. Render useful local content immediately and show a concise “Checking for updates (GitHub)” indicator followed by current/newer/unavailable status. Re-entering Home, resizing, or rerendering must not repeat the request. Reuse the existing fixed endpoint, timeout, response-size limit, redirect refusal and version comparison; no credentials or local installation inventory are sent. An offline/session control and `AGENTIC_SDD_NO_UPDATE_CHECK=1` suppress automatic checks; explicit retry remains user-triggered. Offline, timeout, rate-limit and development-version states are non-fatal and never block navigation/install/restore. No automatic upgrade/download, persistent update cache, or browser launch.

Home and Activity & Help expose **Online guide / README** pointing to `https://github.com/nawodyaishan/agentic-sdd#readme`. Make it a terminal hyperlink where supported and display the copyable URL with an accessible keyboard “Open in browser” action. Open only on explicit selection through a platform opener, without a shell; unsupported/headless/SSH environments or opener failure retain the URL and explain how to open it locally. Do not claim a remote SSH opener launches the user's laptop browser. No browser/network action is triggered merely by rendering the link. Other CLI operations are inventoried against the actual build: unavailable/experimental OpenCode handlers are explanatory help only, never executable TUI controls under this feature's fence.

### FR-07 — Asynchrony and safe interruption

Render/update handlers do not perform blocking filesystem/network work. Typed operation results carry operation identity; stale results cannot overwrite newer screen state or authorize writes. During apply/restore, q/Esc/navigation cannot abandon or repeat the transaction. Ctrl+C requests exit-after-safe-completion rather than forcibly terminating an uncancellable engine transaction; show that policy before execution. Read-only tasks may cancel cooperatively. Cleanup restores terminal state on normal exit, errors, and handled interruption; abrupt OS kill/power loss is not claimed to be safely cancellable.

Handle OS SIGINT/SIGTERM and platform-supported SIGHUP as well as Ctrl+C key events. During writes defer exit until the engine finishes/rolls back and logs flush; while idle exit with terminal cleanup. Repeated signals do not force-abort a transaction. A permanently wedged engine can require SIGKILL, which cannot guarantee cleanup; state this limitation rather than claiming unconditional safe cancellation.

### FR-08 — Branding, animation, and symbols

Use **AGENTIC SDD** and “Build software with AI. Stay the engineer.” consistently in header, welcome, confirmation, and results. Include **by nawodyaishan** in welcome/About and in the full-size branded shell; narrow layouts may move attribution into About, never crowding safety controls. Keep attribution secondary to the product name, preserve its exact spelling, and link the author name to `https://github.com/nawodyaishan` using the same explicit-opening/fallback policy as the online guide. Define shared tokens/components for focus, headings, metadata, warnings, diffs, badges, and footer. Violet is the primary focus accent, cyan secondary, with text-labelled green/amber/red outcomes.

Use a high-contrast dark shell palette with a distinct background and a colored, padded top bar that gives the title more visual space than a bare one-line heading. Every text role must meet WCAG AAA (>=7:1) against its actual background, and non-text borders/icons at least 3:1; status meaning stays text-labelled. The top bar collapses to a compact single-row form on short terminals and must never consume warning, confirmation, footer or paging space. Color-off/plain output remains unstyled and bounds-safe.

Deliver a compact ASCII logo with a short, non-looping reveal (target ≤1 second, ≤10 frames/second), never a blocking splash or fake progress. Inputs work from frame one and can skip it; no animation during confirmation or writes. Logo space collapses on narrow/short terminals. Provide a visible session setting for motion and symbols; plain/accessibility modes are static and ASCII-only. Honor existing ASCII/color controls; `NO_COLOR` disables color, not implicitly every Unicode character. Emoji/icons are optional, restrained, measured by terminal cell width, and replaceable with textual/ASCII equivalents without losing meaning. Animation schedules stop when hidden, disabled, complete, or quitting.

### FR-09 — Compatibility and launch

Existing preview/apply/backups/logs/restore/guide/version behavior, flags, exit codes, trace contracts, legacy numeric guide choices/defaults, and operation stdout/stderr bytes remain unchanged. The permitted additive compatibility exceptions are the `tui` dispatch/help topic and its usage/reference lines (including usage displayed after invalid input); review those exact fixture diffs, not a blanket output rebaseline. **For Homebrew installations, running bare `agentic-sdd` in a real interactive terminal launches the TUI by default.** This applies only to zero-argument invocation, not flags or explicit subcommands. Non-Homebrew bare invocation and all non-TTY bare invocation retain historical preview behavior; `agentic-sdd preview` remains the universal explicit preview escape hatch. The Homebrew install command itself must not launch the TUI from a package-manager hook. Add discoverable explicit `agentic-sdd tui` with real input/output TTY checks; forced-interactive test mode never launches the alternate screen. No silent fallback to a different write workflow if the TUI is missing or fails.

Retain the standalone experimental entry during development. Companion-binary packaging is the proposed isolation approach; release artifacts must pair matching CLI/TUI versions and require no Go toolchain at runtime. Existing explicit CLI remains usable without the companion. Identify the Homebrew distribution through explicit packaging/build metadata, not a guessed path, shell alias, or broad formula redirect. An absent/mismatched companion on a Homebrew default launch gives a clear error and suggests explicit preview without executing it automatically.

## 4. Acceptance criteria

| ID | Observable result | Required evidence |
|---|---|---|
| AC-01 | Four destinations and consistent focus/key/footer behavior; no duplicate child help | Model navigation tests and owner walkthrough |
| AC-02 | Correct 40/60/79/80/109/110-column and short-height behavior; resize preserves selection and consent safety | Cell-width/layout tests, long-path/warning fixtures |
| AC-03 | Dynamic counts, kind/purpose filters, live purpose/ID search, complete metadata, truthful file statuses and profile routing (mode, provider, primary model, documented fallbacks, manual-failover note) | Catalog fixtures including presets, unknown metadata, routing/provider/fallback and zero-match cases |
| AC-04 | Client/pack selection matches exact preview and apply; explicit empty selection cannot expand to all clients | Isolated-home subset/omitted/explicit-empty/invalid-client tests |
| AC-05 | No writes before affirmative confirmation; option/input drift requires re-preview; double Enter cannot double apply | Fake operation call assertions plus isolated-home integration tests |
| AC-06 | Backup Enter only opens details; preview/default-No confirmation protect recovery, including backups from subset installs | Subset replacement→backup→restore round trip, stale/corrupt backup tests; unselected files untouched |
| AC-07 | Local Home renders immediately; exactly one bounded automatic main-menu update check per session, opt-out and explicit retry work without blocking; help/logs/version/onboarding work | Injected filesystem/network tests for success/offline/timeout/rate-limit/dev version, repeated Home entry and bounded log fixtures |
| AC-08 | Slow operations do not block rendering where permitted; writes cannot be abandoned; key and OS-signal exits restore terminal after safe completion | Delayed-result tests and PTY Ctrl+C/SIGINT/SIGTERM/SIGHUP cleanup evidence where supported |
| AC-09 | Compact animated ASCII identity, exact nawodyaishan attribution, colors, icons/emoji, AA/AAA high-contrast shell and static/ASCII variants meet FR-08 without obscuring controls | Deterministic contrast-ratio, frame/timer and attribution/link/plain/no-color/cell-width fixtures; human visual review |
| AC-10 | Existing operation bytes/contracts unchanged except enumerated additive TUI help/usage; no scripted/non-TTY TUI auto-launch | Characterization tests, reviewed additive help fixture diffs, unchanged J1–J10 inputs |
| AC-11 | Isolated nested module remains buildable, root dependency/toolchain contracts preserved; explicit launch handles absent/mismatched companion safely | Root/nested builds, package-isolation and launcher/package tests |
| AC-12 | Every supported major operation is discoverable, with excluded experimental operations labelled unavailable and non-executable | Operation-to-screen matrix, tests, and owner walkthrough |
| AC-13 | README/author links are hyperlinked where supported and keyboard-openable only on explicit action, with visible URLs and failure/SSH/plain fallbacks | Injected opener/URL tests, terminal hyperlink fixtures and owner browser-opening walkthrough |
| AC-14 | Homebrew interactive zero-argument invocation opens TUI; explicit commands/flags, non-Homebrew and piped invocation retain their contracts; install hook never auto-launches | Distribution-marker/TTY/argument dispatch matrix and paired Homebrew package validation |

## 5. Decisions and dependencies

- **D01 — Resolved scope direction:** The owner's follow-up explicitly requests Homebrew-default TUI. r2 specifies Homebrew interactive zero-argument launch, preserves explicit/non-TTY behavior, and leaves `guide` on its historical CLI path. This removes the r1 design hold; it does not record combined approval or authorize publication.
- **D02 — B1 accepted with refinement:** Owner reviewed B1 and said “im impressed, only worry is improveing spacing in text and lines”, then requested parallel next work. This accepts B1 as the B2 baseline and authorizes spacing refinement alongside T04/T05; final B2 visual review remains required.
- **DEP-012 — accepted:** Owner accepted Feature 012 B1/B2 as the implementation baseline on 2026-11-10, current session message “all 1 2 3 approved accepted”. Historical 012 delivery evidence remains in its own ledger.
- r3 proposes two narrow engine additions: validated explicit client selection with restorable subset backups, and structured operation-result APIs with compatibility wrappers. Neither authorizes a second installer, arbitrary destinations, or a manifest-format change.
- **D03 — authorized:** Owner's “all 1 2 3 approved accepted” explicitly permits adding `internal/appops` to every maintained core selector/required-directory list and reconciling package-count guidance in B2, without removing existing coverage or enabling excluded experiments. B1 does not create that package.

## Approval

Single combined approval record for this feature's three documents.

- **Decision:** `approved` by owner on 2026-11-10.
- **Human actor / evidence:** Repository owner, current session message: “approved 3 docs” followed by “now run architecture review using subagent via deepseek flash model”.
- **Scope approved:** `spec.md` r3, `plan.md` r3, `tasks.md` r3, including all architecture-review amendments. Original r2 approval is preserved in the preceding evidence; r3 confirmation is the owner's current-session “all 1 2 3 approved accepted” responding to the explicit three-point approval/acceptance request.
- **r3 amendment status:** `approved`, including D03 additive gate coverage and DEP-012 baseline acceptance. Technical review remains distinct from this human approval.
- **B2 acceptance:** `accepted by owner` on 2026-11-10, current-session message “approved the changes, commit and push”, after the owner reviewed the routing, high-contrast background and colored top bar changes. This accepts B2 and the B2-level visual result. Platform-specific checks (macOS PTY, SSH/tmux, reduced-color/truecolor, screen readers) remain `not_run`; acceptance is not evidence for them.
- **B3 authorization:** `authorized by owner` on 2026-11-10. Current-session message: “start from docs/handovers/2026-11-10_013b.md / then start b3 parallelly multiple sub agents by using sdd free and sdd implement deepseek go according to task's effort”, followed by “also use muse spark contributor 1.3”. This authorized the B3 tasks T06–T08 with parallel delegated implementation lanes, including paid `opencode-go/deepseek-v4.1-flash` and `opencode-go/muse-spark-1.3-contributor`.
- **B3 acceptance:** `accepted by owner` on 2026-11-10, current-session message “b3 accepted , start b4 if b4 b5 parallelly using implement agent fleet of free, deepseek, gemini flash”, followed by “first commit and push current changes”. This accepts the implemented B3 working tree and authorizes committing and pushing it.
- **B4/B5 authorization:** `authorized by owner` on 2026-11-10 in the same message. The owner requested B4 and B5 in parallel using free, DeepSeek and Gemini Flash implementation lanes. Dependencies still govern integration: B5 screens build on the B4 operation host, so the batches may start together only with disjoint file ownership and a shared contract.
- **Execution authorization:** B1, B2 and B3 accepted; B4 and B5 authorized and starting; **B6 remains unauthorized.** No live apply, release, tag, formula publication or real-home operation is authorized.
