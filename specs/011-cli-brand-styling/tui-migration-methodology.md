# TUI Migration Plan & Methodology: Charm v2 Evaluation

**Feature Reference:** 011 — Branded CLI Styling & Guided UX  
**Document Type:** Technical Feasibility, Architecture Blueprint & Migration Methodology  
**Target Ecosystem:** Charm v2 (`charm.land/bubbletea/v2`, `charm.land/huh/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`)  
**Status:** Evaluated Proposal & Strategic Recommendation  

---

## 1. Executive Conclusion on Current Issues

An objective analysis of the November 9–10 CLI logs and the Go codebase confirms that the primary UX friction stems from **two distinct categories of problems**:

```
+-----------------------------------------------------------------------------------+
|                            CURRENT PROBLEM DECOMPOSITION                          |
+-------------------------------------------------+---------------------------------+
|          CATEGORY A: PRESENTATION BUGS          |  CATEGORY B: ARCHITECTURAL FLOW |
|        (Not solved by a TUI framework)          |    (Directly solved by a TUI)   |
+-------------------------------------------------+---------------------------------+
| • DEF-01: Repeated labels on wrapped lines      | • UX-11: 650+ lines scrollback  |
| • UX-02: Copy stutter & redundant prefix text   | • UX-01: Flat 18-profile list   |
| • UX-03: Dominant legal disclaimers vs purpose  | • UX-04: Rigid numbered inputs  |
| • UX-07: Inconsistent blank-line emissions      | • UX-05: Inverted onboarding    |
+-------------------------------------------------+---------------------------------+
```

### Key Takeaway on Framework Selection
Migrating to Bubble Tea or Huh **will not fix Category A**. If a string formatter blindly prepends a 57-character label to every line of text, or if copy is repetitive and disclaimer-heavy, rendering it inside a Bubble Tea viewport will still display broken, verbose text. 

However, for **Category B**, a full TUI provides a leap forward:
- Persistent in-place screen updates replace continuous terminal scrolling.
- Filterable, searchable list components (`bubbles/list`) replace static 18-item menus.
- Keyboard-driven navigation (`j`/`k`, arrows, `/` search, `Esc` back) replaces rigid numeric prompts.
- Guided multi-step forms (`huh/v2`) replace linear yes/no command-line prompts.

---

## 2. Tech Solution: Modern Charm v2 Ecosystem

Research conducted across Charm's current releases (Q1/Q2 2026) reveals that the Charm toolchain underwent a major architectural overhaul with **Charm v2**, moving to the new vanity import path `charm.land/*`.

### Component Matrix

| Module | Vanity Import Path | Purpose in Agentic SDD | Role |
|---|---|---|---|
| **Bubble Tea v2** | `charm.land/bubbletea/v2` | Application core; Elm Model-Update-View runtime | Manages active screen state, keyboard routing, window resize events, and clean terminal exits. |
| **Huh v2** | `charm.land/huh/v2` | Interactive forms & wizard sequences | Powers the guided onboarding and installation flow: client selection, path inspection, and preview confirmation. |
| **Lip Gloss v2** | `charm.land/lipgloss/v2` | Declarative terminal CSS styling & layout | Renders borders, colors, status badges, padding, and definition lists with hanging indents. |
| **Bubbles v2** | `charm.land/bubbles/v2` | Reusable UI widgets | Provides `list.Model` (searchable agent catalog), `viewport.Model` (scrollable detail panes), and `spinner.Model` (updates/github checks). |

### Why Charm v2 Over Alternatives?

1. **Charm v2 vs. v1:** Charm v1 (`github.com/charmbracelet/...`) is in maintenance mode. Charm v2 introduces a completely rewritten ANSI rendering engine (`ultraviolet`), declarative layout primitives, true-color fallback handling, and native accessibility modes. Targeting v1 in late 2026 would accumulate technical debt immediately.
2. **Charm v2 vs. tview / cview:** `tview` uses an imperative, stateful widget tree that tightly couples rendering with logic, making unit testing difficult. Bubble Tea's functional Elm architecture (`Model -> (Model, Cmd)`) allows testing state transitions entirely in memory without a real terminal.
3. **Charm v2 vs. Pure Stdlib:** The Go standard library has no concept of raw terminal manipulation, window resize signals (`SIGWINCH`), or keypress-by-keypress capture without `golang.org/x/term`. Building an interactive browser in pure stdlib requires reinventing terminal state machines from scratch.

---

## 3. Recommended Architecture: The Decoupled Hybrid Pattern

Under no circumstances should the CLI commands be replaced or rewritten with a TUI. Coding agents, CI pipelines, shell scripts, and headless environments depend on deterministic, unstyled, exit-code-based CLI execution.

The solution is a **Strict Hybrid Architecture**:

```
                              agentic-sdd binary
                                      │
                     [Terminal & Invocation Dispatcher]
                                      │
           ┌──────────────────────────┴──────────────────────────┐
           ▼                                                     ▼
   Headless / CLI Mode                                   Interactive TUI Mode
 (Non-TTY, pipes, flags, CI)                       (TTY stdout, interactive terminal)
           │                                                     │
 ┌─────────────────────┐                               ┌─────────────────────┐
 │  cmd/agentic-sdd/   │                               │    internal/tui/    │
 │  • preview / apply  │                               │  • Bubble Tea Root  │
 │  • restore / logs   │                               │  • Huh Wizards      │
 │  • scriptable flows │                               │  • Bubbles Catalog  │
 └──────────┬──────────┘                               └──────────┬──────────┘
            │                                                     │
            └──────────────────────────┬──────────────────────────┘
                                       ▼
                       Shared Core Domain Operations
                        (internal/skillsync package)
                                       │
            ┌──────────────────────────┼──────────────────────────┐
            ▼                          ▼                          ▼
     Plan / Preview            Filesystem Sync           Backup / Restore
    (Pure calculation)       (Atomic file writes)       (Manifest rollback)
```

### Architectural Rules
1. **Zero Domain Logic in TUI:** The `internal/tui` package must contain **no filesystem mutation code, no path resolution logic, and no backup algorithms**. It communicates with `internal/skillsync` solely through strongly typed command structs and read-only queries.
2. **Non-TTY Immunity:** If `!env.StdoutTTY || !env.Interactive`, Bubble Tea is never initialized. The program executes the classic fast-path CLI, guaranteeing 100% byte-for-byte backward compatibility with existing tests and container harnesses.
3. **Accessible Mode (Huh):** Huh v2 includes a built-in accessible mode (`huh.WithAccessible(true)`) that automatically switches interactive forms to plain, screen-reader-friendly prompts when `ACCESSIBLE=1` or `TERM=dumb`.

---

## 4. TUI Structural Design

### Package Layout
```text
internal/
├── skillsync/              # Unchanged: authoritative installer/restore engine
└── tui/
    ├── app.go              # Root tea.Model & screen router
    ├── theme.go            # Lip Gloss styles, brand colors (cyan/magenta), borders
    ├── keys.go             # Keymaps (help, quit, back, search, enter)
    ├── screens/
    │   ├── home.go         # 4-zone dashboard (Get Started, Manage, Help)
    │   ├── wizard.go       # Huh-based 4-step installation & preview flow
    │   ├── catalog.go      # Bubbles list with category filtering & live search
    │   ├── detail.go       # Viewport-backed detail pane with definition layout
    │   ├── backups.go      # Backup selection & manifest diff viewer
    │   └── logs.go         # Viewport log viewer
    └── components/
        ├── header.go       # Compact ASCII branding & version indicator
        ├── statusbar.go    # Breadcrumbs & active client indicators
        └── footer.go       # Dynamic keybinding hints
```

### Screen Flow & State Machine

```
              ┌───────────────┐
              │   Home Menu   │◄─────────────────────────────┐
              └───┬───┬───┬───┘                              │
                  │   │   │                                  │
    ┌─────────────┘   │   └─────────────┐                    │
    ▼                 ▼                 ▼                    │
┌──────────────┐┌──────────────┐┌──────────────┐             │
│ Installation ││ Agent & Skill││   Backups &  │             │
│ Wizard (Huh) ││Catalog (List)││ Restore View │             │
└───────┬──────┘└──────┬───────┘└───────┬──────┘             │
        │              │                │                    │
        │              ▼                │                    │
        │       ┌──────────────┐        │                    │
        │       │ Detail Pane  │        │                    │
        │       │  (Viewport)  │        │                    │
        │       └──────┬───────┘        │                    │
        │              │                │                    │
        └──────────────┴────────────────┴────────────────────┘
                     (Esc / 'b' / Back key)
```

---

## 5. Critical Limitations, Prerequisites & Risks

While Bubble Tea v2 provides an outstanding developer experience, our live Exa/Context7 investigation revealed **major technical hurdles** that must be resolved before adoption.

### ⚠️ Prerequisite 1: Go Toolchain Incompatibility
This is the most critical blocker:
- **Current Repository:** `go.mod` specifies `go 1.23`.
- **Active Container Image:** `Taskfile.yml` and `.github/workflows/ci.yml` strictly pin `golang:1.25-alpine`.
- **Live Charm v2 Requirements:**
  - `charm.land/bubbletea/v2` requires **`go 1.26.0`** minimum (`go.mod` in main).
  - `charm.land/huh/v2` requires **`go 1.25.8`** minimum.
- **Impact:** Charm v2 **cannot compile** under the project's current container testing toolchain (`golang:1.25-alpine`). Adopting Charm v2 requires:
  1. Updating `go.mod` from `1.23` to `1.26`.
  2. Upgrading Docker/CI base images to `golang:1.26-alpine`.
  3. Verifying Homebrew and package build pipelines support Go 1.26.

### ⚠️ Prerequisite 2: Supply Chain & Dependency Bloat
- **Current State:** The repository has **0 external dependencies** (pure Go standard library). It compiles in ~1.2 seconds, produces a 3.8 MB binary, and has zero supply-chain risk.
- **Charm v2 Footprint:** Importing `charm.land/bubbletea/v2` + `charm.land/huh/v2` pulls in:
  - **10 direct dependencies**
  - **18–22 transitive dependencies** (`golang.org/x/sys`, `golang.org/x/sync`, `catppuccin/go`, `mitchellh/hashstructure`, `lucasb-eyer/go-colorful`, `muesli/cancelreader`, `rivo/uniseg`, etc.).
  - Binary size increases from **~3.8 MB to ~16–19 MB**.
  - Releases and license scanning must now audit 25+ third-party modules.

### ⚠️ Prerequisite 3: PTY & Test Automation Constraints
- The project currently verifies interactive behavior via lightweight PTY scripts (`tests/guide-e2e.sh`).
- Bubble Tea applications require terminal alternate-screen buffers (`altscreen`). Scripting alternate-screen applications via plain bash `read` loops requires PTY harnesses (like `expect` or `teatest`) rather than simple newline-delimited stdin fixtures.

---

## 6. Migration Methodology & Phased Plan

To manage risk and prevent blocking active release cycles, migration should follow a **Three-Phase Strategy**:

### Phase 1: Stdlib Polish & Presentation Fixes (Immediate / In Feature 011)
*Do not wait for a framework migration to fix active user-facing defects.*
1. **Fix `printWrappedProse`:** Implement hanging indentation and sectioned definition formatting (resolves DEF-01).
2. **Clean Copy Stutter:** Remove redundant prefixes from `guide_catalog.go` and `guide_explorer.go` (resolves UX-02, UX-03).
3. **Accept `b` on Details:** Enable seamless Back navigation across all menus (resolves UX-04).
4. **Group Catalog Output:** Visually segment the 18 profiles by purpose in the existing menu (resolves UX-01).
*Deliverable:* Production-ready, zero-dependency release with polished CLI UX.

### Phase 2: Toolchain Preparation & TUI Feasibility Spike (Experimental Branch)
1. **Toolchain Upgrade:** Bump repository `go.mod` to `1.26` and update `Taskfile.yml` container images from `golang:1.25-alpine` to `golang:1.26-alpine`.
2. **Isolated Spike (`cmd/sdd-tui-spike`):**
   - Build a standalone prototype proving `charm.land/bubbletea/v2` and `charm.land/bubbles/v2/list` against the existing `internal/skillsync.InspectSelection(...)` inventory.
   - Verify memory usage, binary size, and SSH/narrow-terminal behavior.
   - Prove the `teatest` headless unit-test workflow.

### Phase 3: Hybrid Production Integration
1. Implement `internal/tui` with the decoupled architecture described in §3.
2. Wire `agentic-sdd guide` to launch Bubble Tea when running on an interactive TTY, falling back to Phase 1's stdlib menu when piped or non-interactive.
3. Migrate the installation confirmation to a Huh v2 multi-step form.
4. Validate CI gates, GoReleaser packaging, and Homebrew universal binary generation.

---

## 7. Final Verdict & Best Option

| Option | Description | Pros | Cons | Verdict |
|---|---|---|---|---|
| **Option A: Pure Stdlib TUI** | Stay with Go stdlib; enhance current menu loop with hanging indents, grouped lists, and clean redraws. | Zero dependencies, instant builds, 3.8MB binary, works on Go 1.23+, zero supply-chain risk. | Cannot do live search-as-you-type or split-pane viewports without significant custom code. | **Best Immediate Option (Phase 1)** |
| **Option B: Charm v1 (`charmbracelet/bubbletea`)** | Adopt older v1 modules (`github.com/charmbracelet/bubbletea`). | Compatible with Go 1.23. | Dead-end API; misses v2 performance, declarative layout, and accessibility features. | **Rejected** |
| **Option C: Charm v2 Hybrid Migration** | Upgrade toolchain to Go 1.26; adopt `charm.land/bubbletea/v2` + `huh/v2` for interactive TTYs, keeping stdlib CLI for pipes/headless. | World-class user experience, searchable catalog, guided forms, elegant responsive layout. | Requires bumping Go toolchain to 1.26+, breaks current Go 1.25 CI container, adds 25+ dependencies, increases binary size. | **Best Long-Term Option (Phase 2 & 3)** |

### Strategic Recommendation
**Adopt the Staged Hybrid Approach (Option A immediately, transitioning to Option C):**
1. **Execute Phase 1 immediately in Feature 011:** Fix the text-wrapping defect (DEF-01), catalog grouping (UX-01), copy stutter (UX-02), and navigation friction (UX-04) within the existing zero-dependency Go codebase. This delivers 80% of the UX value immediately without destabilizing toolchains or dependencies.
2. **Authorize Phase 2 as a dedicated follow-up feature:** Once Go 1.26 is formally adopted across project CI and container infrastructure, implement the Charm v2 hybrid TUI using the architecture and isolation boundaries specified in this document.
