# Plan: Feature 012 — CLI UI/UX Remediation & Hybrid TUI Experience

**Revision:** Draft r2 (Addressing Architecture Review findings)  
**Inputs:** [spec.md](spec.md) draft r2, [ui-ux-review-and-fixes.md](../011-cli-brand-styling/ui-ux-review-and-fixes.md), [tui-migration-methodology.md](../011-cli-brand-styling/tui-migration-methodology.md), Architecture Review verdict (`Needs changes` r1 addressed in r2)  
**Approval Reference:** [spec.md §5](spec.md#5-approval), the single approval record.  

---

## 1. Technical Approach & Architecture

Feature 012 addresses the user experience in two strictly bounded execution batches:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        FEATURE 012 TWO-STAGE PLAN                      │
├────────────────────────────────────────┬───────────────────────────────┤
│    BATCH B1: STDLIB REMEDIATION        │    BATCH B2: TUI SPIKE &      │
│     (Immediate / Production Ready)     │    NESTED EXPERIMENTAL MODULE │
├────────────────────────────────────────┼───────────────────────────────┤
│ • Zero external dependencies           │ • Nested `experiments/tui/`   │
│ • Fixes DEF-01 label-wrapping defect   │ • Root `go.mod` untouched     │
│ • Stable numeric menu IDs (1–6, q)     │ • Read-only catalog spike     │
│ • Groups 18 profiles by purpose        │ • Toolchain migration recipe  │
│ • Eliminates copy stutter              │ • Opt-in `Taskfile.experimental`│
│ • Adds `b` navigation to details       │ • Headless `teatest` harness  │
│ • Preserves byte-identical plain test  │ • Comprehensive digest audit  │
└────────────────────────────────────────┴───────────────────────────────┘
```

### Batch B1 Architecture: Pure Stdlib Presentation Layer
- **Unified Definition Formatter (`cmd/agentic-sdd/guide_presentation.go`):** Implement `printDefinition(out io.Writer, label, text string, width int)`:
  - Consolidates formatting across detail views and list rows (eliminating duplicate wrapping logic from `guide_explorer.go:312-323`).
  - Short labels (≤ 16 runes): Hanging indent where line 0 prints `label + text_line_0`, subsequent lines print `spaces(len(label)) + text_line_N`.
  - Long labels (> 16 runes): Block format where line 0 prints `label`, subsequent lines print `  ` (2 spaces) + `text_line_N`.
  - Shorten the 57-rune label `Declared intent (educational; not an enforced guarantee):` to `Safety:`, ensuring maximum line length never exceeds width at 40, 60, or 80 columns.
- **Grouped Profile List (`cmd/agentic-sdd/guide_explorer.go`):** Render section dividers (`--- Group Name ---`) corresponding to `explorerProfileGroups` while maintaining continuous 1-based indexing (`1` through `18`), avoiding nested submenus and keeping numeric selection intact.
- **Copy Clean-Up & Test Re-alignment (`cmd/agentic-sdd/guide_catalog.go`):**
  - Update `teachingRegistry`: remove `"Optional preset: "` prefixes from purpose strings; simplify example prompts; standardize safety badges.
  - Deliberately update `cmd/agentic-sdd/guide_catalog_test.go:96-108` to remove the outdated assertion requiring purpose to start with `"Optional preset:"`.
  - Reconcile `guide_explorer.go:351` (`Optional preset: yes`) and line 309 suffix.
- **Universal Back Navigation:** Update `explorerDetail` prompter choices to include `"b"` alongside `"q"` and `"enter"`.
- **Whitespace Normalization:** Strip redundant trailing blank lines from sub-screen rendering functions before returning to the main menu loop.
- **Shared Loop / Non-TTY Protection (`cmd/agentic-sdd/guide.go`):**
  - Scope friendlier exit (`Exited.`) and contextual empty state (`No backups found. ...`) strictly to the `terminal` (`env.StdoutTTY`) branches.
  - Non-terminal / piped execution preserves legacy strings (`Cancelled.`, bare `No backups found.`) byte-for-byte, ensuring `guide_plain_compatibility_test.go` passes without fixture modifications.
- **Stable Numeric Menu IDs:** Retain choice numbers `1` through `6` and default `"1"`. Render visual section headers (`GET STARTED`, `MANAGE`, `HELP`) purely in `printGuideMenuTerminal`, preserving all 10 automated test journeys in `tests/guide-e2e-in-container.sh` and existing trace event schemas.

### Batch B2 Architecture: Decoupled TUI & Nested Module Isolation
- **Nested Module Isolation (`experiments/tui/go.mod`):**
  - To prevent breaking `tidy-check`, `mod-verify`, and the pinned `golang:1.25-alpine` container CI, all Charm v2 dependencies (`charm.land/bubbletea/v2`, `charm.land/huh/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`) must reside in an independent nested module: `experiments/tui/go.mod`.
  - Root `go.mod` (Go 1.23, zero dependencies) and root directory (zero `go.sum`) remain completely untouched.
  - Experimental tasks live exclusively in `Taskfile.experimental.yml` (e.g. `task -t Taskfile.experimental.yml tui:test`), preserving default `task test` and `task vet` boundaries on the 7 core packages.
- **Read-Only Scope & Domain Decoupling:**
  - The B2 spike is strictly **read-only browsing** (skills and profiles catalog). Mutation operations (apply, restore) remain exclusively in the CLI.
  - `experiments/tui` consumes domain data solely via `skillsync.InspectSelection(...)` and `skillsync.SummarizePlan(...)`. It does not import `package main`'s `*Env` or `*Tracer`; it defines its own `TUIConfig` (width, color profile, interactive flags).
- **Launch Precedence & Gating:**
  - The TUI spike is gated on an explicit opt-in environment variable: `AGENTIC_SDD_TUI=1`.
  - If `AGENTIC_SDD_FORCE_INTERACTIVE=1` is set (used by automated test journeys), the standard CLI menu always takes precedence over the TUI to prevent alternate-screen hang in scripted test runners.
- **Comprehensive Toolchain Migration Blueprint:**
  - Audit and document all locations pinning Go toolchains, container images, and digests:
    1. `Taskfile.yml:75` (`golang:1.25-alpine`)
    2. `.github/workflows/ci.yml:37` (Docker image digest)
    3. `tests/harness/trusted_ci.py:16` (reviewed trusted image digest)
    4. `internal/harnesscheck/runner.go:319-320` (literal prerequisite table lookup)
    5. `tests/guide-e2e.sh:11` & `tests/docker-e2e.sh:11`
    6. `harness/checks.json:52-56` & `offline-b7-checks.json:5`
    7. `.github/workflows/release.yml:22-26` (`go-version-file: go.mod`)
    8. `README.md:66,95` & `harness/README.md:128` (documented Go 1.23 floor).

---

## 2. Affected Modules and Boundaries

| Module | Nature of Change | Boundary / Constraints |
|---|---|---|
| `cmd/agentic-sdd/guide.go` | Visual menu section headers, terminal-only exit/empty-state copy, whitespace normalization | Stable numeric IDs (1–6); non-TTY plain output byte-identical. |
| `cmd/agentic-sdd/guide_presentation.go` | Unified `printDefinition` formatter | Replaces duplicate wrapping in `guide.go` and `guide_explorer.go`. |
| `cmd/agentic-sdd/guide_explorer.go` | Grouped profile list, `b` detail navigation, status badges | Inspect read-only inventory only; never write files or contact providers. |
| `cmd/agentic-sdd/guide_catalog.go` | Catalog copy cleanup, concise safety badges | Reconcile with updated `guide_catalog_test.go`. |
| `cmd/agentic-sdd/guide_catalog_test.go` | Update test assertion for preset Purpose | Align with deduplicated copy. |
| `experiments/tui/` | Nested module with Charm v2 read-only spike | Separate `experiments/tui/go.mod`; zero impact on root `go.mod`/`go.sum`. |
| `Taskfile.experimental.yml` | Add `tui:test` task | Opt-in only; core `task test`/`task vet` stay on 7 core packages. |

---

## 3. Specialist and Tool Assignments

- **Assigned Specialist:** Main agent coordinator with Go/TDD expertise. No specialized subagent is assigned as a default; implementation and verification follow established repository standards.
- **Tools Assigned:**
  - `edit` / `write`: Targeted Go and test updates.
  - `shell`: Running local test gates (`task test`, `task vet`, `git diff --check`, container tests).
  - `Context7`: Querying Charm v2 and Lip Gloss layout APIs during TUI prototype work.
  - `Exa`: Checking upstream release metadata and CLI UX guidelines.

---

## 4. Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| **Go 1.26 Module Breakage in Container CI** | Importing Charm v2 in root module will break `golang:1.25-alpine` container gates and `tidy-check`. | B2 is isolated in nested module `experiments/tui/go.mod`. Root `go.mod` stays Go 1.23 with 0 dependencies. |
| **Breaking Automated E2E Journeys** | Renumbering menu choices breaks existing container journeys J1–J10. | Keep choice IDs strictly stable (`1`–`6`, `q`); visual grouping is purely presentational in terminal output. |
| **Non-TTY Output Drift** | Changing exit text or empty states in shared code breaks legacy plain fixtures. | Scope `Exited.` and contextual empty states strictly to `terminal` branches; plain branches stay byte-identical. |
| **40-Column Terminal Overflow** | Labels longer than 40 chars overflow the screen width. | Shorten `Declared intent...` to `Safety:`; unit test rendered line length ≤ width at 40, 60, and 80 cols. |

---

## 5. Verification Commands

```sh
# 1. Focused presentation, catalog, and explorer unit tests
go test -count=1 ./cmd/agentic-sdd -run '^(TestGuideUX|TestGuideExplorer|TestGuideCatalog|TestGuidePresentation|TestGuidePlain)'

# 2. Supported-core package tests (verifies zero-dependency root module)
task test

# 3. Code formatting and static analysis
task vet
git diff --check

# 4. Offline end-to-end container journeys (verifies all J1-J10 pass without modification)
sh tests/guide-e2e.sh
sh tests/docker-e2e.sh
```
