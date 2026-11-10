# Agentic SDD — TUI UX/UI Expert Review & Redesign Brief

**Project:** [nawodyaishan/agentic-sdd](https://github.com/nawodyaishan/agentic-sdd)
**Review date:** 2026-11-10
**Scope:** First-run onboarding, home, skill discovery, installation/update, preview, confirmation, results, backup/recovery, help, visual design, and maintainable Go TUI architecture.
**Theme decision:** **One built-in dark theme only**. No light theme or theme-switching feature.
**Type:** Evidence-based design review and implementation-ready upgrade recommendations; **not a code audit or a claim of tested defects**.
**Evidence:** 12 supplied UI screenshots; publicly accessible repository README; Exa research; Context7 documentation for the Charm TUI ecosystem.
**Intended audience:** New coding-agent users, students and learners, solo engineers, experienced CLI users, and future multi-agent/personal-fleet users.

---

## 1. Executive assessment

**Verdict:** Retain the full-screen TUI and underlying preview-first installer. Redesign its **information hierarchy, first-run journey, change preview, keyboard interaction, and color system**. Do not rewrite the installation engine just to improve the interface.

The current screens suggest a promising, safety-conscious product: explicit selection of client destinations, read-only previews, default-No confirmation, separate OpenCode optional content, backup records, and operation logs. Those are genuine differentiators worth preserving. The major weakness is that **the app reads like an installer diagnostic interface instead of a welcoming developer product**.

A newcomer should be able to answer these questions within the first screen:

1. **What is Agentic SDD?** A repeatable, review-controlled workflow for coding agents.
2. **What will it install?** Agent skills and optional OpenCode profiles, in selected client folders.
3. **What should I do next?** Review detected tools, preview changes, and apply only after confirmation.

### Overall design direction

> **Goal-first, progressive, keyboard-first.** Show a useful choice and its consequence, reveal technical paths and logs on demand, and make every operation's outcome obvious.

### Prioritization summary

| Priority | Theme | Outcome |
|---|---|---|
| **P0** | First-run flow + exact preview + confirmation | Users understand what they are installing and can complete or cancel safely |
| **P0** | Navigation, focus, key binding consistency | Users never wonder what is selected, what Enter will do, or how to go back |
| **P0** | Long-content rendering and result states | The 94-item preview is comprehensible without scrolling through 94 raw paths |
| **P1** | Unified dark design system | A coherent, premium, accessible look across the app |
| **P1** | Explore / Library information architecture | Users discover skills by problem and purpose, not internal names |
| **P1** | Backups / recovery discoverability | Users understand what can be restored and how to preview a restore |
| **P1** | Responsiveness / empty / failure states | Smaller terminals, offline sessions, and no-op installs stay usable |
| **P2** | Guided tour, contextual help, preferences | Lower learning curve without burdening expert users |
| **P2** | Future fleet/profile manager integration | Agent/profile configuration has room to grow without polluting core installation |

**Severity disclaimer:** P0 here denotes **UX delivery priority**, not a confirmed data-loss or security vulnerability.

---

## 2. What the screenshots actually show

The images cover Home, Install & Manage hub, client selection, optional OpenCode pack, configuration review, exact preview, confirmation, no-change result, backup listing, Explore, and Activity & Help.

### Strengths to keep

- **Preview-first safety** and a clear distinction between read-only inspection and writing.
- **Default-No** confirmation: Enter does not unexpectedly apply changes.
- **A complete installation unit** for core skills, avoiding partial skill sets.
- **Optional OpenCode profiles are separate** from the four core skill destinations.
- **Backups and operation logs**, with restore presented as a deliberate action.
- **Keyboard-driven navigation** that supports advanced terminal users.
- **Offline-compatible embedded source** with no download requirement in the shown install workflow.
- A restrained palette and an already recognizable plum/cyan visual signature.

### Principal friction points

| ID | Priority | Evidence from supplied images | User problem | Recommended correction |
|---|---|---|---|---|
| UX-01 | P0 | Home foregrounds inventory counts, paths, and update check | No immediate explanation of why this tool matters or which action to take | Add a 2-line value proposition, setup state, and one primary action |
| UX-02 | P0 | `agents`, `codex`, `claude`, `agy` appear as selectable codes | Newcomers cannot decode `agents` versus a specific tool or `agy` | Use user-facing names and descriptions; keep internal identifiers only in details |
| UX-03 | P0 | Optional pack + `free-first` are shown with internal terminology | Model policy, template pack, and client destination appear conflated | Explain “OpenCode agent profiles” and “Free-model preference”; avoid implying guaranteed free availability |
| UX-04 | P0 | Exact preview expands to a very long list of paths | The decision-critical changes are buried in raw inventory | Lead with action totals and grouped client summaries; make individual paths expandable |
| UX-05 | P0 | Confirmation text appears behind an overlapping modal repeating similar information | Duplicated information, unclear focus, and excessive reading | Use **one authoritative confirmation surface** containing only essential facts |
| UX-06 | P0 | Selected state and keyboard hints vary by page | Inconsistent mental model of Enter, Esc, q, j/k, arrows, and focus | Central keymap, focus model, and context-aware footer |
| UX-07 | P0 | Result “No Changes Needed” appears after confirmation | A no-op is discovered late and can feel like a failed install | Detect no-op during preview; offer “Return Home” without a write confirmation |
| UX-08 | P1 | Large blank area under compact content on many full-screen views | The UI feels unfinished rather than intentionally spacious | Put primary tasks into bounded panels, summaries, and clear visual focal points; **do not fill space with decoration** |
| UX-09 | P1 | “Backups & recovery” lists timestamp and `94 installed` | Backup ID and raw timestamp don't convey recoverability or consequences | Human-readable date, operation, affected clients, and “Preview restore” action |
| UX-10 | P1 | Explore shows 33 items and short jargon-heavy details | Hard to discover an appropriate skill for a task | Search, categories, beginner labels, “Use when…”, invocations, and skill details |
| UX-11 | P1 | Header repeats `upd:unavailable`, `motion:on`, `ascii:off` | Operational settings displace product/status information | Move settings to Help/Preferences; show update information only if actionable |
| UX-12 | P1 | Paths such as `/tmp/agentic-sdd-tui...` are displayed on almost every page | Important content is drowned by environment implementation detail | Collapse “Advanced paths & source”, with copyable complete paths in details |
| UX-13 | P1 | “Version: dev / Commit: none / Date: unknown” displayed in Help | Looks unreliable, even if the development build is legitimate | Show “Development build” succinctly; full build metadata in Diagnostics |
| UX-14 | P1 | Footer includes numerous actions separated by `·` | Shortcuts are not grouped by primary vs secondary purpose | Show 2–4 relevant actions; `?` opens comprehensive shortcut help |
| UX-15 | P1 | No-change result has a yellow notice despite successful exact match | Semantic ambiguity between success, warning, and neutral state | Neutral “Already up to date” state, not danger or warning |
| UX-16 | P2 | No visible first-use explanation or learning path | Users must infer the skill → agent workflow | Optional first-run guide and “Try your first skill” after install |

**Important evidence boundary:** Screens show both a Home view with `0/18` skills present and an installation session reporting `94 unchanged` or a backup with `94 installed`. These may reflect **different moments or test fixtures**. Do **not** classify this as an inventory bug without reproducing the same state on the same home directory. Verify refresh after apply separately.

---

## 3. Revised information architecture

### Recommended navigation

```text
AGENTIC SDD
├── Home                Overview, setup health, next action
├── Setup               Install/update wizard + review current installation
├── Library             Browse/search skills and optional profiles
├── Recovery            Backups, restore previews, operation history
└── Help                Quick start, commands, keyboard shortcuts, diagnostics
```

**Future extension, not part of this review's required implementation:** add a **Fleet** page once the separate configurable-agent-profile/fleet feature is implemented. Do **not** label a placeholder “Fleet” as functional or mingle future fleet editing with the current embedded skill installer.

Maintain direct CLI commands as fully supported first-class workflows. The TUI is an interactive front end, **not a replacement** for `preview`, `apply`, `backups`, `restore`, `logs`, and existing flags.

### Navigation rules

- Top header: `AGENTIC SDD / Setup / Review` + a concise, **actionable** status.
- Left navigation rail only when space permits; selected item has both a visible marker and a high-contrast background.
- Main pane owns a title, one-sentence purpose, primary content, and one obvious next action.
- Footer contains **only currently valid actions**. `?` gives extended help.
- `Tab` / `Shift+Tab` cycles among interactive regions (when multiple regions exist); arrow keys / `j`/`k` move inside the active region.
- `Enter` activates the focused item. `Esc` goes back/dismisses a dialog. `q` quits **only outside text entry and modal contexts** and never aborts a write.
- Global keybind handlers must not swallow keys while a list is filtering or an input box is active.
- Always preserve cursor/focus when returning from details or the wizard.
- A current activity header should replace permanently visible `motion:on`, `ascii:off`, and repeated unavailable update status.

---

## 4. First-run experience: target journey

### Recommended flow

```text
Open TUI
  ↓
Welcome / detect supported tools (read-only)
  ↓
Choose target coding tools (smart suggestions; editable)
  ↓
Optional OpenCode pack (plain-language explanation; opt-in)
  ↓
Review exact impact (grouped summary, zero write)
  ↓
If zero changes → “Already up to date” (no confirmation)
If changes > 0 → explicit default-No confirmation
  ↓
Apply with observable progress
  ↓
Result: installed / updated / unchanged / skipped / failed
  ↓
“Use your first Agentic SDD skill” + links to details/logs
```

**Detection** should mean *observed filesystem/runtime evidence*, not a confident claim that an agent client is authenticated or working. Detecting a directory does not prove an executable, account, provider, or model is configured.

### 4.1 New Home

The screenshot's dense inventory should become **secondary**, not disappear. Proposed wide layout:

```text
┌ AGENTIC SDD ──────────────────────────────────────  Help [?] ┐
│                                                              │
│  Build with AI. Keep control of the engineering.              │
│  Specs → approved tasks → reviewable implementation batches. │
│                                                              │
│  GET STARTED                          YOUR SETUP              │
│  > Set up Agentic SDD                 4 tools selected        │
│    Browse skills                      18 core skills          │
│    Manage an existing install         OpenCode pack optional  │
│                                                              │
│  HOW IT WORKS                                                │
│  1. Pick your coding tool                                    │
│  2. Preview exactly what changes                             │
│  3. Approve install and start a skill                        │
│                                                              │
│  No files change until you confirm.                          │
├──────────────────────────────────────────────────────────────┤
│  ↑↓ move     Enter open     ? help     q quit                 │
└──────────────────────────────────────────────────────────────┘
```

For an existing installation, substitute the **get started** block with `Your setup is up to date` and prominent actions **Browse Skills**, **Manage Setup**, **Restore Backup**. Do not force onboarding on returning users.

### 4.2 Client selection

Change from:

- `agents` → **Shared Agents Skills** — a shared skill directory, not an AI provider.
- `codex` → **Codex CLI** — installs Agentic SDD skills for Codex.
- `claude` → **Claude Code** — installs the Claude Code version.
- `agy` → **Antigravity CLI** — installs its compatible skills.

Show each row as `Selected / Not selected`, tool display name, a short explanation, and optional `Detected` / `Not detected` status. A missing client should not be automatically installed, nor should directory presence be presented as proof of usability. Let the user select explicitly.

**Default behavior:** Preselect detected, supported destinations where reliable; if detection confidence is low, retain current defaults but **explain** them. Allow all selection combinations supported by the engine, including zero selected core clients if the optional-pack-only plan is valid under actual backend rules; otherwise explain the constraint before moving on. The policy must remain enforced in the backend, not only in the TUI.

### 4.3 Optional OpenCode pack

Rename the panel to:

**Optional: OpenCode Agent Profiles**

> Adds reusable Agentic SDD agent definitions for OpenCode. This is separate from installing shared workflow skills.

Within it:

- `Install OpenCode agent profiles` (checkbox)
- `Prefer free models in the provided templates` (checkbox; enabled only if pack selected)
- Helper: **“A preference, not a guarantee. Availability, quotas, credentials, privacy, and model IDs depend on the configured provider.”**
- `Details` shows profile count, destination and conflicts.

**Do not present this as configuring the user's personal fleet subscriptions** unless a separate profile-management feature actually implements that behavior.

### 4.4 Review and preview

Replace first-screen technical dumps with an **impact dashboard**:

```text
SETUP / REVIEW                                      Step 3 of 4

The following files would change:

  0 new     0 replaced     94 unchanged     0 conflicts

  TARGET                    NEW  REPLACE  SAME
  Shared Agents Skills        0      0      18
  Codex CLI                   0      0      18
  Claude Code                 0      0      18
  Antigravity CLI             0      0      18
  OpenCode profiles           0      0      15
  Other selected content      0      0       7

  No modifications are needed for the selected destinations.

  > Return home
    View all 94 item details
    Change options
```

*The per-target counts above are **illustrative layout data**, not a verified partition of the captured 94 entries.* Generate exact counts from the backend plan; do not hardcode them. Show the real expected list of target types and ensure totals reconcile.

When there **are** changes, group by **new**, **replace (backup first)**, **unchanged**, and **conflict/refusal**. Allow opening any group or selecting `View paths`. Keep a short summary on screen during scrolling.

### 4.5 Confirmation

The current screenshot shows a repeated confirmation panel and underlying text. Use one controlled dialog **or** a dedicated confirmation page, not both.

```text
CONFIRM INSTALLATION

  6 new items
  2 existing items will be backed up before replacement
  86 unchanged items will be left alone

  Destinations: Codex CLI, Claude Code, OpenCode profiles

  This operation writes files.
  You may cancel now. Once writing begins, the current
  operation will finish or follow the existing safe-exit policy.

  Proceed with these changes?

  > No, go back     Yes, apply

  Default: No. Enter alone never approves writes.
```

**Safety contract:** Keep the existing explicit affirmative input, default-No semantics, staged writes, rollback behavior, symlink/path validation, and operation logs. The exact wording of interrupt behavior must follow verified backend guarantees. Do not promise “instant cancellation” during a write, or guaranteed rollback after `SIGKILL`.

### 4.6 Result and next steps

Distinguish clearly:

- **Installed successfully** — counts, destinations, backup status, next action.
- **Already up to date** — neutral/successful no-op, no backup created; `Back to Home` and `Browse Skills`.
- **Cancelled** — “Nothing changed” if true.
- **Partially completed / failed** — actual applied state, rollback status, log link, and recovery actions; never claim success from an unverified exit.
- **Preflight refused** — rejected path/source and fix suggestion; no write.

A success screen should offer a concrete, beginner-friendly example, such as opening the SDD Router in a supported coding agent and asking it to outline the next feature. Avoid promising an auto-launch of a client.

---

## 5. Skill library and discovery redesign

**Current issue:** The screenshot's Explore page packs 33 items into a flat list with programmatic IDs and a right-side description. It is efficient for known-item lookup but opaque to newcomers.

**Proposed experience:**

1. Top search input: `Search by task, role or skill name…` (e.g. “debug error”, “plan feature”, “review architecture”).
2. Categories: **Start & Orient**, **Plan & Specify**, **Build & Implement**, **Review & Verify**, **Debug & Recover**, **Agent Coordination**, **Profiles**. Final category membership must come from the actual catalog.
3. Each list item: human title, 1-line purpose, type `Skill`/`Agent profile`, and installed status.
4. Details: **Why use it**, **When to use**, **What it does not do**, **How to invoke**, **Supported clients**, **Installed state**, **Files (advanced)**.
5. Recommended entry: **“New here? Start with SDD Router.”** This is navigation guidance, not automatic execution.
6. Search/filter and list selection must be maintained across detail→back transitions.

**Example list item:**

```text
> SDD Router                     Skill · Workflow
  Start or resume work and choose the next safe step.
  Installed: 3/4 selected clients          Enter: details
```

Expose a distinct informational badge for **“Available template”**, **“Installed and matching”**, **“Modified locally”**, **“Not installed”**, and **“Unreadable / refused”** only when the relevant facts can actually be established. Colors must reinforce labels, not replace them.

---

## 6. Backups, recovery, activity and help

### Backups

The backup list should read as a recovery decision interface:

```text
RECOVERY / BACKUPS

> Nov 10, 2026 · Installation
  94 items recorded · 5 destination groups
  Backup ID: 20261110T134059...

  Enter: inspect     r: refresh

Selected backup
  Saved replacements: 0
  Fresh installs recorded: 94
  This record can support previewing a restore.

  > Preview restore (read-only)
    View technical manifest
    View operation log
```

**Critical:** “94 items recorded” is not the same as “94 file backups exist.” For a fresh install, the manifest may record items to remove on restore and **zero saved replacement snapshots**. Show manifest facts accurately and distinguish *recorded installation entries* from *saved pre-existing content*. A restore action must always run a separate exact preview and confirmation, retaining default-No behavior.

### Activity

Replace giant in-UI log streams with a compact recent-operations list, a selected-operation summary, and an optional scrollable raw-log viewport. Display operation, timestamp, changed/skipped/failed counts, result, log path, and recovery guidance. Preserve the underlying log format and CLI output.

### Help

The current Help page bundles version/build, networking, a long log, URL attribution, and controls. Split into **Quick start**, **Keyboard shortcuts**, **Command reference**, **Troubleshooting**, **Diagnostics**, **About**. Move build metadata and update-check troubleshooting to Diagnostics. If the check is offline/rate-limited, make it a nonblocking help detail—not a persistent alarming header state.

---

## 7. Visual identity: one dark palette, no light mode

### 7.1 Reference palettes considered

| Developer ecosystem | Visual character | Fit for Agentic SDD |
|---|---|---|
| [Catppuccin Mocha](https://catppuccin.com/palette/) | Soft dark navy/plum, pastel accents, cohesive hierarchy | **Best match** for current plum + cyan visual DNA |
| [Tokyo Night / Storm](https://github.com/tokyo-night/tokyo-night-vscode-theme) | Dark navy, electric blue/violet, sharper technical contrast | Good inspiration, slightly more editor-oriented |
| [GitHub Primer dark color system](https://primer.style/product/getting-started/foundations/color-usage/) | Semantic, functional, restrained | Best **token and semantic-role discipline** |

**Final recommendation:** Ship **exactly one dark-only theme**, named internally `agentic-dark`, based on **Catppuccin Mocha** color values, with a strict semantic token layer inspired by GitHub Primer. Do **not** ship a light theme, a theme switcher, or multiple built-in dark skins for this upgrade.

### 7.2 Approved color tokens

The hex values below are sourced from the official Catppuccin Mocha palette; the **assignment to TUI roles is our design recommendation**.

| Token | Hex | Use |
|---|---|---|
| `bg.canvas` | `#1E1E2E` | Overall application background |
| `bg.rail` | `#181825` | Sidebar and header/footer strips |
| `bg.deep` | `#11111B` | Elevated contrast / deepest separators, sparingly |
| `bg.surface` | `#313244` | Panels, muted row backgrounds |
| `bg.selected` | `#45475A` | Selected/focused rows |
| `border.subtle` | `#585B70` | Panel boundaries; avoid full box around every paragraph |
| `text.primary` | `#CDD6F4` | Body, headings, focused row text |
| `text.secondary` | `#BAC2DE` | Secondary information on canvas |
| `text.tertiary` | `#A6ADC8` | Supplementary captions on canvas |
| `accent.brand` | `#CBA6F7` | Product title, primary action cues, active breadcrumb |
| `accent.focus` | `#89B4FA` | Focus marker/border and active keyboard control |
| `accent.info` | `#94E2D5` | Section labels, informational context |
| `status.success` | `#A6E3A1` | Completed install, match confirmed, recoverable success |
| `status.warning` | `#F9E2AF` | Warnings, potential replacement, caution |
| `status.error` | `#F38BA8` | Actual failure, validation refusal, destructive outcomes |

**Do not fill large panels with accent colors** or make all headings magenta. The screen should be ~mostly neutrals, with selectively used brand and semantic highlights.

### 7.3 Contrast checks and constraints

Quick calculated WCAG contrast ratios (using standard sRGB luminance calculations):

| Foreground/background | Ratio | Recommendation |
|---|---:|---|
| Primary text `#CDD6F4` on canvas `#1E1E2E` | **11.34:1** | Strong |
| Secondary-caption `#A6ADC8` on canvas `#1E1E2E` | **7.37:1** | Strong |
| Brand `#CBA6F7` on canvas `#1E1E2E` | **8.07:1** | Strong |
| Primary text `#CDD6F4` on selected `#45475A` | **6.31:1** | Good |
| Caption `#A6ADC8` on selected `#45475A` | **4.10:1** | **Avoid** for normal text |
| Blue `#89B4FA` on selected `#45475A` | **4.33:1** | **Avoid** as normal text; use for border or focus cue |
| Red `#F38BA8` on selected `#45475A` | **3.94:1** | **Avoid** as normal text |

Hence: **when a row receives the selected background, change its text to `text.primary`,** rather than retaining normal caption or status colors. Use blue for a focus border/caret, not as weak-contrast selected text. Recalculate contrast for every new foreground/background combination.

Test with truecolor and low-color terminals, `NO_COLOR=1`, monochrome, color-blind interpretation, and standard default terminal themes. Text must always provide the status. Avoid relying on dim or blinking effects.

### 7.4 Design language rules

- **Brand:** Bold `AGENTIC SDD`; one short tagline, not a giant ASCII logo.
- **Palette ratio:** Neutral surfaces dominate; lilac for brand/primary action; blue for focus; teal for information; green/yellow/red exclusively for applicable status.
- **Typography:** Use only the user's terminal font; establish hierarchy via weight, spacing, and semantic color.
- **Layout:** Consistent margins; 1 blank line between meaningful sections; no duplicate headings.
- **Selection:** A focus indicator plus selected background; never only a colored glyph.
- **Borders:** Use separators intentionally; prefer one pane boundary to nested boxes everywhere.
- **Status:** `Ready`, `No changes`, `Needs review`, `Failed` should be explicit labels.
- **Animation:** None required for the identity. Use only short loading indicators for actual ongoing work; suppress when reduced-motion/low-capability mode is requested.
- **Icons:** Use common Unicode with ASCII alternatives. No Nerd Font dependency.
- **Terminal constraints:** Respect background/capability detection, alternate-screen behavior, SSH/tmux, and resized windows. Do not use background colors in a way that breaks transparency assumptions without a capability fallback.

### 7.5 Suggested screen composition

Use the same structural primitives everywhere:

```text
┌ Header / breadcrumb / meaningful status ─────────────────────┐
│ Sidebar (wide only) │ Page title + purpose                    │
│                     │                                        │
│                     │ Main action or grouped content          │
│                     │ Contextual details, only when useful    │
├──────────────────────────────────────────────────────────────┤
│ Primary shortcut  •  Back  •  Help                           │
└──────────────────────────────────────────────────────────────┘
```

Avoid printing full source paths or verbose safety prose on the Home page. An advanced details disclosure is not a loss of transparency if it is easy to find and exact.

---

## 8. Interaction contract and keyboard standards

| Situation | Expected behavior |
|---|---|
| First run | A highlighted primary action; instructions visible without `?` |
| List | Up/Down or `j/k` moves selection; Enter opens; `/` starts filtering where supported |
| Checkbox | Space toggles; selection state spelled out |
| Tabs / multi-pane | Tab advances focus region; visual focus travels with it |
| Wizard | Enter advances when valid; Esc returns one step; previously chosen values persist |
| Input field | Printable keys type text; global `q`, `/`, `j`, `k` do not activate navigation |
| Help | `?` opens contextual help; Esc closes it |
| Confirmation | Default is No; only explicit affirmative action begins a write |
| Preview | Read-only; line totals reconcile; path details are optional |
| No-op | No pointless confirmation; no backup invented |
| Active write | Navigation and quit cannot corrupt state; progress/operation status visible |
| Error | Explain what, why (if known), where to find evidence, and exact next step |
| Backups | Inspect → preview restore → default-No confirmation → execute |
| Terminal resize | Content and focus remain accessible; footer never overlaps primary content |
| Exit | Restores terminal state and prints a concise persistent result if appropriate |

**Do not rely on mouse input** for essential operations.

---

## 9. Responsive behavior and content density

Use the character grid, not pixel-based breakpoints:

| Terminal size | Recommended behavior |
|---|---|
| `>= 120` columns | Sidebar + list + details only for Explore; sidebar + main pane for other pages |
| `80–119` columns | Narrow sidebar plus main; Explore switches between list and details |
| `60–79` columns | Compact single-pane navigation; breadcrumb shortened |
| `< 60` columns | One-column, minimal safe controls; warn if a critical confirmation cannot fit |
| Height `< 24` rows | Compress header/footer, remove decorative whitespace, maintain scroll/focus |
| Very short/unsupported size | Explain minimum viewport in plain language; never silently clip confirmation |

Treat breakpoints as **starting recommendations**, validated through actual terminal snapshot tests. Content must use measured cell widths, Unicode-aware wrapping, vertical scrolling and safe truncation. Preserve selection during resizing.

---

## 10. Engineering approach for Go (without installer rewrite)

**Do not assume the screenshots prove which TUI framework or version is in use.** Inspect `go.mod`, current TUI packages, key handlers, state models, and install services before deciding on dependency changes.

### If the project already uses Bubble Tea / Lip Gloss

Retain that architecture. Refactor toward **small screen models** and a shared UI design system rather than migrating frameworks. Bubble Tea's Model–Update–View design supports this, and the Bubbles ecosystem provides reusable interactive components.

Recommended conceptual boundaries:

```text
internal/tui/
  app/              # navigation, app state, messages, focus ownership
  screens/
    home/
    setup/
    library/
    recovery/
    help/
  components/       # sidebar, header, status, empty state, modal, diff summary
  theme/            # single dark semantic token map + ANSI fallback
  keymap/           # shared key registry and dynamically generated help
  layout/           # width/height, wrapping, breakpoints
  adapters/         # translate backend plans/results into view models
```

**This is a suggested separation, not a verified current directory structure.** Keep existing packages where a smaller refactor achieves the same boundaries.

### Libraries: conditional fit

| Library | Recommended role | Guardrail |
|---|---|---|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Screen state, event dispatch, async commands, full-screen rendering | Use the version already in the repository unless a justified compatibility change is needed |
| [Bubbles](https://github.com/charmbracelet/bubbles) | List/search, viewport, help, keybindings, text input, table, progress | Prefer documented components over custom scroll logic |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Shared dark styles and responsive composition | Centralize theme tokens; do not scatter hex strings |
| [Huh](https://github.com/charmbracelet/huh) | Optional checkbox/confirm/form primitives | Adopt only if it reduces existing complexity and preserves exact default-No semantics |
| Existing installer engine | Plans, exact preview, validations, backup, apply/restore, logs | **Single source of truth**; UI does not reimplement safety or copy operations |

**Version caution:** Current public Charm documentation includes v2 APIs and older v1 APIs; do not mix their imports or method signatures. Context7 examples are guidance, not proof of compatibility with the project's checked-in dependencies.

### Required state model properties

- Screen state transitions are explicit and testable.
- An immutable `PlanView` is derived from the backend plan; `Preview` and `Apply` refer to the **same validated target plan**, or apply refuses a changed/stale plan and requires a refreshed preview.
- A single owner handles modal focus; the underlying screen must not process confirm keys while modal is open.
- All stateful actions return deterministic results with a clear success/no-op/error distinction.
- View rendering does not mutate files, check network synchronously, or perform surprise expensive I/O.
- Update-check failures never block local setup.
- No destructive action is initiated from selection/hover/navigation alone.
- Read-only browse never creates directories or backups.
- Existing noninteractive CLI output/exit codes remain stable unless intentionally versioned and documented.
- Use capability detection and graceful degradation rather than depending on terminal-specific visual features.

---

## 11. Acceptance criteria and test plan

### User-centered success criteria (proposed targets, not measured benchmarks)

- **5-second test:** A novice can describe what the tool installs, whether it changes files without confirmation, and how to start.
- **60-second orientation test:** A novice can find the install wizard and identify supported clients without reading documentation.
- **Preview comprehension:** A novice correctly distinguishes new, replaced, unchanged, and failed/conflicted items.
- **Safety:** No key sequence that merely navigates or presses default Enter can trigger `apply` or restore.
- **Discoverability:** Users can find SDD Router by task wording or name in Library search.
- **Recovery:** Users can inspect a backup and preview a restore without inadvertently writing.

### Quality gates

- [ ] Reproduce all supplied screenshot flows with a deterministic fixture.
- [ ] Verify same-state inventory refresh after apply and no-op.
- [ ] Test fresh install, partial install, modified destinations, all-up-to-date, refused path, failed write, and restore preview.
- [ ] Test empty, one, and many backups, including backup manifest with zero saved replacements.
- [ ] Test keyboard focus with popup, search field, checkbox, details and list viewports.
- [ ] Test Enter, Esc, q, `/`, Space, `?`, j/k, arrows, Tab, Ctrl+C under each relevant state.
- [ ] Test `NO_COLOR`, ASCII/Unicode fallback, 16-color, 256-color, truecolor, tmux and SSH.
- [ ] Verify all selected-row text pairs against accessibility contrast targets.
- [ ] Snapshot screens at 140×40, 100×30, 80×24, 60×20, and very narrow widths.
- [ ] Test clipping, long paths, Unicode widths, viewport position, search state on resize.
- [ ] Confirm the exact plan totals match the backend and never show illustrative numbers as real values.
- [ ] Verify no-op preview skips write confirmation and creates no backup/log unless explicitly defined as a read-only audit record.
- [ ] Confirm unsafe signals/rollback descriptions match the actual backend contract.
- [ ] Verify repeated setup does not rewrite exact-matching files.
- [ ] Preserve scriptability, argument compatibility, exit status and stdout/stderr behavior.
- [ ] Include an explicit manual novice walkthrough with 3–5 users if feasible.

---

## 12. Delivery sequence (phased UX work; not a new SDD roadmap phase number)

### Slice A — Trust, clarity, and navigation (**do first**)

Fix one confirmation owner, no-op path, action labels, shared focus/keymap, concise footer, error/result clarity, and plan summary grouping. This improves usefulness and safety while minimizing rendering changes.

### Slice B — Newcomer journey

Rebuild Home and Setup wizard around purpose → detected targets → optional integrations → exact review → confirm/result. Rewrite jargon into plain language. Add a post-install “first skill” guide.

### Slice C — Single dark design system

Introduce the **one dark semantic palette** from section 7; unify backgrounds, typography, borders, selected states, focus markers, status roles, modal dimensions, and spacing. No light theme or switcher.

### Slice D — Discovery and recovery

Upgrade Library search/categories/details and Recovery/Activity summaries; preserve raw technical views behind explicit advanced actions.

### Slice E — Robustness and maintainability

Responsive layouts, terminal capability fallback, input/focus regression tests, snapshots, TUI accessibility checks, and documentation updates.

**Keep each slice independently reviewable.** Avoid an all-at-once rewrite. Preserve existing CLI behavior and security checks throughout.

---

## 13. Decisions and explicit non-goals

### Decisions

1. **Only one dark theme.** Default and sole scheme: Mocha-based `agentic-dark`.
2. **Keep preview-first, default-No installation behavior.**
3. **Begin with a goal/action, not inventory or raw paths.**
4. **Treat exact plan and backend as authority**; the TUI displays and confirms them.
5. **Make OpenCode a first-class supported integration**, while keeping its optional agent profile pack separate from portable skills.
6. **Preserve expert workflows** through CLI commands and advanced details, not by forcing every user to read diagnostics.

### Non-goals for this UX review

- Rewriting the Go installer / backup engine.
- Changing subagent model routing or personal subscription configuration.
- Claiming provider credentials or model availability based on a template.
- Creating an agent-profile marketplace (leave an information-architecture slot for later work).
- Adding light theme, multiple color themes, mouse-only controls, or elaborate animation.
- Changing security guarantees based solely on UI wording.
- Claiming source-level defects not reproduced or code-reviewed.

---

## 14. Ready-to-run implementation prompt for the coding agent

> Review the attached `Agentic SDD — TUI UX/UI Expert Review & Redesign Brief` and the current Agentic-SDD repository. Treat the document as a **UX design recommendation**, not verified source truth. First perform a read-only audit of the current Go TUI stack, navigation, state transitions, keymaps, rendering, install plan/apply/restore interfaces, terminal compatibility and test fixtures. Produce a short implementation plan ordered by Slices A–E, calling out any conflict with current security contracts or existing SDD specs. Then implement only the approved next slice. Use the **single dark `agentic-dark` palette** from the report; no light mode or theme switcher. Preserve all existing CLI commands, flags, behavior, file safety checks, backups, no-op semantics and default-No confirmation. Do not reimplement the installer inside UI components. Prefer current dependencies where possible; use Exa and Context7 to verify any new library/API usage. Add focused state-machine/keybinding tests and screen snapshots for the changed flows. Report the exact user-visible differences, limitations and any remaining accessibility risks.

---

## 15. Research sources and limits

### Primary external references

1. [Agentic SDD public repository](https://github.com/nawodyaishan/agentic-sdd) — project intent, workflow, installer safety model, CLI semantics (the README may lag the supplied screenshots).
2. [GitHub TUIKit — Foundations](https://github.com/github/TUIKit/blob/main/docs/foundations.md) — terminal UX, help, status, responsive layout, color, accessibility, keyboard, I/O considerations.
3. [Vercel CLI Design Guidelines](https://vercel.com/design/cli) — progressive disclosure, action-oriented error recovery, noninteractive behavior and consent.
4. [GitHub Primer — Color Usage](https://primer.style/product/getting-started/foundations/color-usage/) — semantic colors and design tokens; separation of raw values and usage roles.
5. [GitHub Primer — Color Considerations](https://primer.style/accessibility/design-guidance/color-considerations/) — contrast and accessible color pairings.
6. [Catppuccin — Official Palette](https://catppuccin.com/palette/) — verified Mocha values used in proposed token map.
7. [Catppuccin — Style Guide](https://github.com/catppuccin/catppuccin/blob/main/docs/style-guide.md) — use-case and legibility guidance.
8. [Tokyo Night VS Code theme](https://github.com/tokyo-night/tokyo-night-vscode-theme) — alternative developer-product palette reference, not the selected scheme.
9. [Microsoft — Command-Line Design Guidance](https://learn.microsoft.com/en-us/dotnet/standard/commandline/design-guidance) — explicit interaction, commands, scriptability.
10. [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Huh](https://github.com/charmbracelet/huh) — official implementation references, cross-checked through Context7.

### Verification notes

- **Screenshots support observational UI critique**, not assertions about the exact internal view model, dependency versions, or filesystem behavior.
- **The public README supports the high-level workflow and safety model** but may not reflect all latest UI changes. Repository checkout/CI review is required before coding.
- **Color tokens are design decisions.** Contrast figures were calculated for the specified sRGB pairs; rendering may differ with terminal capability and font configurations.
- **No user study was performed.** Usability targets are proposed acceptance criteria, not empirically measured outcomes.

---

**Bottom line:** Agentic SDD already has the hard part of a credible tool—a deliberate installer and review-controlled workflow. The highest-leverage UI work is to turn that machinery into a clear, approachable product: explain the purpose, guide the first decision, summarize the impact, maintain strong consent, and hide technical depth until requested. Apply one well-defined dark visual system, not more screens or more decoration.
