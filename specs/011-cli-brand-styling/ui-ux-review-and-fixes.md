# UI/UX Comprehensive Review and Prioritized Fixes

**Feature:** 011 — Branded CLI Styling & Guided UX  
**Scope:** Terminal presentation, text wrapping, information architecture, navigation, and visual hierarchy  
**Sources:** November 9–10, 2026 CLI interaction logs, codebase inspection (`cmd/agentic-sdd/`), Exa research (clig.dev, Vercel CLI Guidelines, GitHub TUIKit, CLI Style), and Context7 library references (Charm Lipgloss layout/wrapping)  
**Status:** Working Proposal & Implementation Roadmap  

---

## 1. Executive Summary

A comprehensive review of the interactive CLI session (recorded 2026-11-09/10) and the underlying Go implementation identified **12 distinct UI/UX issues**. These range from an acute formatting defect (field labels repeating across wrapped lines) to information overload (18 flat agent listings) and navigation friction (rejecting standard keyboard shortcuts).

While the CLI excels at safety (preview-by-default, explicit confirmations, non-destructive exploration, and strict pipe/automation neutrality), its terminal presentation suffers from formatting defects and structural dissonance between an informative onboarding experience and a basic menu loop.

This document synthesizes the November 9 UI/UX review with concrete code-level findings and industry standards to establish an actionable, prioritized remediation plan.

---

## 2. Master Issues Matrix

| Priority | ID | Issue Title | Nature | Current Friction | Target State | Affected Components |
|---|---|---|---|---|---|---|
| **🔴 P0** | **DEF-01** | **Repeated Field Labels on Wrapped Lines** | Functional Defect | `printWrappedProse` prepends `prefix` to *every* wrapped line; long labels (57 chars) repeat up to 19 times, crushing readable columns. | Label on first line only with hanging indent, or label on own line with indented body for long headers. | `cmd/agentic-sdd/guide.go`<br>`cmd/agentic-sdd/guide_explorer.go` |
| **🔴 P0** | **UX-01** | **Overwhelming Flat Agent Listings** | Information Architecture | All 18 OpenCode profiles are displayed in a single, unsegmented numbered list, confusing beginners. | Group profiles by functional role/purpose group with scannable category headers. | `cmd/agentic-sdd/guide_explorer.go`<br>`cmd/agentic-sdd/guide_catalog.go` |
| **🟠 P1** | **UX-02** | **Label-to-Body Copy Stutter & Redundancy** | Content / Copy | Stuttering text: `Ask your main agent: Ask your main agent to...`, `Declared intent: Declared template intent...`, and `Optional preset: ... (optional preset)`. | Clean, concise labels (`Example:`, `Safety:`); omit redundant leading text; eliminate duplicate preset tags. | `cmd/agentic-sdd/guide_catalog.go`<br>`cmd/agentic-sdd/guide_explorer.go` |
| **🟠 P1** | **UX-03** | **Dominant Safety Disclaimers vs Purpose** | Content / Hierarchy | 4–5 lines of dense legalistic disclaimers overpower the actual purpose and usage instructions on every detail card. | Compact badge-style safety summary on cards with full disclaimer deferred to a concept card or footnote. | `cmd/agentic-sdd/guide_catalog.go`<br>`cmd/agentic-sdd/guide_explorer.go` |
| **🟠 P1** | **UX-04** | **Navigation Friction (`b` Rejected on Details)** | Interaction | Detail card screen rejects `b` (Back) as invalid; only accepts `Enter` or `q`, violating muscle memory from parent lists. | Accept `b`, `B`, and `Enter` to return to list; accept `q` to exit. | `cmd/agentic-sdd/guide_explorer.go` |
| **🟠 P1** | **UX-05** | **Inverted Onboarding Hierarchy** | Information Architecture | First screen prompts for skill installation before explaining what Agentic SDD is or how agents use it. About is menu 4; Explore is menu 6. | Reorganize main menu into logical sections (Get Started, Manage, Help) or offer an optional guided first-use tour. | `cmd/agentic-sdd/guide.go` |
| **🟠 P1** | **UX-06** | **Dead-End Explorations Without Next Steps** | Workflow / Discoverability | Exploring a skill or profile provides no direct route to try, install, or use it; users must back out manually. | Provide contextual next-step hints (e.g. copyable command or shortcut to preview/install). | `cmd/agentic-sdd/guide_explorer.go`<br>`cmd/agentic-sdd/guide_presentation.go` |
| **🟡 P2** | **UX-07** | **Inconsistent Vertical Whitespace & Double Blank Lines** | Visual Rhythm | Menu returns from About (`4`) and Updates (`5`) produce 2 blank lines before dividers, while Backups (`2`) produces 1. | Normalize blank-line emissions across sub-screens to preserve a single consistent separator line. | `cmd/agentic-sdd/guide.go`<br>`cmd/agentic-sdd/updates.go` |
| **🟡 P2** | **UX-08** | **Unclear Installation & Template State Labels** | Clarity | Status strings like `missing; origin pack` require mental translation to determine if an agent is ready. | Use clear status badges: `Template available (Not installed)`, `Installed (Current)`, `Modified`. | `cmd/agentic-sdd/guide_explorer.go` |
| **🟡 P2** | **UX-09** | **Unhelpful Empty States** | Guidance | `No backups found.` prints a bare statement without explaining how backups work or what the user should do next. | Explain that backups are generated automatically upon file replacement and suggest installing first. | `cmd/agentic-sdd/guide.go` |
| **🟡 P2** | **UX-10** | **Abrupt Exit & Development Build Messaging** | Polish | Exiting with `q` outputs `Cancelled.` (feels like an abort); dev build update check provides no local verification hint. | Display `Exited.` or friendly departure; clarify that dev builds track working tree revisions. | `cmd/agentic-sdd/guide.go`<br>`cmd/agentic-sdd/updates.go` |
| **🟡 P2** | **UX-11** | **Excessive Terminal Scrollback in Append-Only TTY** | Ergonomics | Long menus and repetitive screens generate 650+ lines of scrollback across a single brief session. | Compact menu presentation, clear visual anchors, and minimal redundant redisplay. | `cmd/agentic-sdd/guide.go` |

---

## 3. Detailed Problem Analysis & Solutions

### 🔴 P0 — Critical Defects & Structural Blockers

#### DEF-01: Repeated Field Labels on Wrapped Lines

* **Observed Output:**
  ```text
  Declared intent (educational; not an enforced guarantee): Declared
  Declared intent (educational; not an enforced guarantee): template
  Declared intent (educational; not an enforced guarantee): intent only,
  Declared intent (educational; not an enforced guarantee): not an
  Declared intent (educational; not an enforced guarantee): operating-system
  Declared intent (educational; not an enforced guarantee): sandbox,
  Declared intent (educational; not an enforced guarantee): permission or
  Declared intent (educational; not an enforced guarantee): runtime
  Declared intent (educational; not an enforced guarantee): guarantee.
  ... (repeated 19 times)
  ```
* **Root Cause (`cmd/agentic-sdd/guide.go:488-496`):**
  ```go
  func printWrappedProse(out io.Writer, prefix, text string, width int) {
      available := width - utf8.RuneCountInString(prefix)
      if available < 1 {
          available = 1
      }
      for _, line := range wrapProse(text, available) {
          fmt.Fprintln(out, prefix+line) // Defect: prefix repeated on EVERY line
      }
  }
  ```
  When `prefix` is long (e.g. 57 characters) and `width` is 72, `available` is only 15 characters. The text wraps into tiny 15-character fragments, each prepended with the full 57-character prefix. On narrow 40-character terminals, `available` drops to 1, causing catastrophic vertical blowout.
* **Standard / Best Practice (clig.dev & Vercel CLI Guidelines):**
  * *Definition List Pattern:* Short labels (≤ 15 characters) use a **hanging indent** where line 1 has the label and subsequent lines are indented with spaces.
  * *Block Pattern:* Long labels (> 15 characters) appear on their own line (`Header:`), followed by the prose body indented by 2 spaces.
* **Remediation:**
  1. Refactor `printWrappedProse` to support hanging indents (`prefix` on line 0, space padding on subsequent lines).
  2. Implement `printDefinition(out, label, text, width)`:
     - If `len(label) > 16`, print `label` on line 1, then wrap `text` with 2-space indentation.
     - If `len(label) <= 16`, wrap with hanging indent equal to `len(label)`.

---

#### UX-01: Overwhelming Flat Agent Listings

* **Observed Output:**
  Menu option `2) OpenCode profiles` displays 18 items in an unsegmented, unbroken list from 1 to 18. Beginners cannot discern differences between `sdd-architecture-advisor`, `sdd-architecture-review-9r`, `sdd-openai-architect`, and verification reviewers.
* **Root Cause (`cmd/agentic-sdd/guide_explorer.go:302-324`):**
  Items are iterated linearly. Although `guide_catalog.go` contains internal `Group` metadata (`Main communicator`, `Read-only investigation`, `Implementation/documentation`, `Test assistance`, `Independent/supplementary review`), the list renderer ignores these groups and renders a flat list.
* **Standard / Best Practice (GitHub TUIKit & Vercel CLI):**
  Group commands and items by user task and role. Present clear category headers (`--- Main Orchestrator ---`, `--- Architecture & Review ---`) within the list, or provide category filtering.
* **Remediation:**
  In `printExplorerProfilesList`:
  - Group items under section headings corresponding to `explorerProfileGroups`.
  - Maintain continuous 1-based indexing for direct selection while breaking the wall of text into scannable logical clusters.

---

### 🟠 P1 — High-Impact UX & Navigation Improvements

#### UX-02: Label-to-Body Copy Stutter & Redundancy

* **Observed Output:**
  ```text
  Ask your main agent: Ask your main agent to request a read-only architecture opinion...
  Declared intent (educational; not an enforced guarantee): Declared template intent only...
  7) sdd-implement-1 — Optional preset: routine implementation... (optional preset)
  ```
* **Root Cause (`cmd/agentic-sdd/guide_catalog.go` & `guide_explorer.go`):**
  1. Card examples duplicate the prompt prefix in their text bodies.
  2. Caveats repeat `"Declared template intent only"` right after the label `"Declared intent"`.
  3. Catalog `Purpose` begins with `"Optional preset:"` while `printExplorerListRow` appends `" (optional preset)"`.
* **Remediation:**
  - Simplify card labels: use `Example:`, `When to use:`, `Safety:`.
  - Trim redundant lead-ins from catalog strings (e.g. `"Request a read-only architecture opinion..."`).
  - Strip `"Optional preset: "` prefixes from catalog `Purpose` values.

---

#### UX-03: Dominant Safety Disclaimers vs Functional Purpose

* **Observed Output:**
  Every single detail card renders a massive 4-line legal disclaimer explaining that loading a profile does not start an agent, grant sandbox access, or contact a provider. This disclaimer is larger than the description of what the agent actually does.
* **Standard / Best Practice:**
  *Progressive Disclosure:* Keep essential safety points scannable in the main card; defer exhaustive legal mechanisms to dedicated documentation.
* **Remediation:**
  - On detail cards, condense the caveat to a compact summary:
    ```text
    Safety:
      Advisory template · No automated execution · Requires human authorization
    ```
  - Reference `Key concepts -> human-approval` or `preview-apply` for complete governance details.

---

#### UX-04: Navigation Friction (`b` Rejected on Details)

* **Observed Output:**
  ```text
  Press Enter to return to the list, or q to exit guide: b
  Press Enter to return to the list, or q to exit guide.
  ```
* **Root Cause (`cmd/agentic-sdd/guide_explorer.go:257-263`):**
  `p.ask` only whitelists `[]string{"q"}` with default `"enter"`. Entering `b` triggers the invalid input path.
* **Standard / Best Practice (clig.dev):**
  Hierarchical menus must support consistent navigation keys. If `b` means "back" on screens 1, 2, and 3, it must mean "back" on screen 4.
* **Remediation:**
  Update `explorerDetail` to accept `[]string{"b", "q"}` with default `"enter"`. Both `enter` and `b` cleanly return `false, 0`.

---

#### UX-05: Inverted Onboarding Hierarchy

* **Observed Output:**
  The top-level menu places installation first:
  ```text
  1) Preview and install skills   (default)
  2) Restore from a backup
  3) View latest install or restore log
  4) About and getting started
  5) Check for CLI updates (contacts GitHub)
  6) Explore skills and agent profiles
  q) Quit
  ```
  New users are asked to install files before knowing what the toolkit does, what files will be created, or how their agent will use them.
* **Remediation:**
  Organize the menu into clear functional sections:
  ```text
  GET STARTED
    1) Explore skills and agent profiles
    2) Preview and install skills   (default)
    3) About and getting started

  MAINTENANCE & LOGS
    4) Restore from a backup
    5) View latest install or restore log
    6) Check for CLI updates (contacts GitHub)

    q) Quit
  ```

---

#### UX-06: Dead-End Explorations Without Next Steps

* **Observed Output:**
  After viewing a skill or profile detail, the only option is to return to the list or quit. There is no guidance on how to install just that profile, run a preview, or invoke it in a coding agent.
* **Remediation:**
  Add a footer to detail cards with context-preserving next steps:
  ```text
  Next step:
    To install this profile, run: agentic-sdd apply --with-opencode
    To use in session: ask your agent "Use sdd-openai-architect to review..."
  ```

---

### 🟡 P2 — Visual Polish, State Clarity & Guidance

#### UX-07: Inconsistent Vertical Whitespace & Double Blank Lines

* **Observed Output:**
  Returning from option `4` (About) or option `5` (Updates) prints two consecutive blank lines before `------------------------ Main menu`, whereas option `2` prints only one.
* **Root Cause:**
  `printGuideAboutTerminal` and `printUpdateResult` emit trailing blank lines (`fmt.Fprintln(out)`), and `printGuideMenuTerminal` emits its own leading blank line.
* **Remediation:**
  Normalize whitespace: sub-screen printers must not emit trailing blank lines if the main menu loop emits a leading divider newline.

---

#### UX-08: Unclear Installation & Template State Labels

* **Observed Output:**
  ```text
  Destination file status at the selected home (not runtime state):
    /home/.../.config/opencode/agents/sdd-openai-architect.md — missing; origin pack
  ```
  The phrase `"missing; origin pack"` sounds like an error or corrupt asset rather than an uninstalled template.
* **Remediation:**
  Adopt standard status badges:
  - `Not installed (template available in pack)`
  - `Installed (matches source)`
  - `Installed (locally modified)`

---

#### UX-09: Unhelpful Empty States

* **Observed Output:**
  ```text
  Choose [1]: 2
  No backups found.
  ```
* **Remediation:**
  Enrich empty states with helpful operational context:
  ```text
  No backups found.
  Backups are created automatically whenever an installation replaces
  existing skills or profiles.
  ```

---

#### UX-10: Abrupt Exit & Development Build Messaging

* **Observed Output:**
  Entering `q` displays `Cancelled.`, which implies an aborted operation or failure rather than an intentional exit.
* **Remediation:**
  - Change message to `Exited.` or `Thank you for using Agentic SDD.`
  - When running a development build, explain that updates are tracked via Git rather than published release tags.

---

#### UX-11: Excessive Terminal Scrollback in Append-Only TTY

* **Observed Output:**
  A typical exploration session generates 600+ lines of terminal output due to complete menu re-prints after every single action.
* **Remediation:**
  While preserving standard-library-only append-only architecture (avoiding full-screen TUI dependencies), make redisplayed menus compact:
  - When returning from a sub-screen, print a compact one-line prompt instead of the full 15-line main menu banner, or allow re-printing the full menu on demand (`m=menu`).

---

## 4. Prioritized Implementation Roadmap

### Phase 1: Defect Remediation & Copy Polish (Immediate / Non-Breaking)
1. **Fix `printWrappedProse` hanging indent and add `printDefinition`:**
   - Eliminate label repetition on wrapped lines (DEF-01).
   - Clean up card definition wrapping on narrow and wide terminals.
2. **Clean catalog copy and eliminate stutter (UX-02):**
   - Refactor `cmd/agentic-sdd/guide_catalog.go` example prompts and purpose strings.
   - Remove duplicate `"Optional preset:"` prefixes.
3. **Add `b` navigation to detail view (UX-04):**
   - Allow users to press `b` or `Enter` to return from card details.
4. **Normalize blank lines (UX-07):**
   - Remove redundant trailing newlines before main menu boundaries.

### Phase 2: Information Architecture & Detail View Redesign
1. **Redesign Detail Cards (UX-03, UX-06, UX-08):**
   - Format: Title/Type -> Actionable Purpose -> Example -> Compact Safety -> Status Badge -> Contextual Next Action.
2. **Group OpenCode Profiles (UX-01):**
   - Insert functional section dividers into the 18-profile list.
3. **Improve Empty States & Messaging (UX-09, UX-10):**
   - Add explanatory text for `No backups found.` and improve exit messaging.

### Phase 3: Onboarding & Menu Flow (Future Polish)
1. **Reorganize Main Menu Hierarchy (UX-05):**
   - Structure menu choices into Get Started, Maintenance, and Help sections.
2. **Explore Interactive TUI / Compact Redraws (UX-11):**
   - Investigate lightweight TTY control for redraw-in-place without adding heavy third-party runtime frameworks.

---

## 5. Architectural Guardrails & Compatibility

All recommended fixes must strictly adhere to Agentic SDD repository constraints:
1. **Stdlib-Only Runtime:** No external TUI dependencies (e.g. Bubble Tea) in the core installer path. Formatting improvements use Go standard library primitives (`strings`, `unicode/utf8`, `fmt`, `bufio`).
2. **Piped Output Neutrality:** All visual enhancements (ANSI colors, hanging indents, section dividers) apply exclusively to `env.StdoutTTY`. Non-TTY pipes must remain byte-identical and unstyled.
3. **Safe Sandboxing:** Exploring skills and viewing status remains strictly read-only; no network calls, agent execution, or filesystem mutations occur without explicit user confirmation.
4. **Test Verification:** Every copy change and layout helper must be verified against `task test`, `task vet`, and existing offline container test suites (`tests/guide-e2e.sh`).
