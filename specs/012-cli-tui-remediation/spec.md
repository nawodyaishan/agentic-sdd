# Feature 012 — CLI UI/UX Remediation & Hybrid TUI Experience

**Status:** Draft r2 — Awaiting Combined Human Approval  
**Inputs:** [ui-ux-review-and-fixes.md](../011-cli-brand-styling/ui-ux-review-and-fixes.md), [tui-migration-methodology.md](../011-cli-brand-styling/tui-migration-methodology.md), [011 spec](../011-cli-brand-styling/spec.md), [010 spec](../010-guided-cli-ux/spec.md), Architecture Review verdict (`Needs changes` r1 addressed in r2)  
**Product Compass:** [vision §3](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#3-who-agentic-sdd-serves), [§6](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#6-non-negotiable-engineering-principles), [§7](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#7-product-architecture-and-ownership-boundaries)  

---

## 1. Outcome and Context

Feature 011 introduced branded terminal styling, ASCII wordmarks, and an initial skills/profiles explorer. Subsequent human testing (2026-11-09/10) identified acute UI/UX issues, including a critical text-wrapping defect where field labels repeat on every wrapped line, flat 18-profile lists, copy stutter, rigid numeric navigation, and excessive terminal scrollback.

An architectural evaluation of modern Go TUI frameworks (Charm v2: `bubbletea/v2`, `huh/v2`, `lipgloss/v2`, `bubbles/v2`) confirmed that while an interactive TUI provides an excellent long-term experience for interactive browsing and forms, it requires Go 1.26+ (incompatible with the repository's pinned `golang:1.25-alpine` container gates) and introduces 25+ transitive dependencies.

Feature 012 delivers a **safe, staged two-batch solution**:
1. **Batch B1 (Stdlib UI/UX Remediation):** Correct all acute formatting defects, copy stutter, navigation friction, list segmentation, visual menu grouping, and empty states within the zero-dependency Go standard library codebase, strictly preserving Go 1.23+ compatibility, stable numeric menu IDs, and byte-identical non-TTY plain output.
2. **Batch B2 (Decoupled TUI Architecture & Experimental Nested Spike):** Establish an isolated architectural boundary in a nested experimental module (`experiments/tui/`) that leaves the root `go.mod` (zero dependencies, Go 1.23) untouched, document the comprehensive toolchain migration blueprint, and prototype a read-only Charm v2 catalog browser.

---

## 2. In-Scope and Explicitly Excluded Work

### In-Scope
- **DEF-01 (Text Wrapping Defect):** Refactor prose wrapping with a dedicated definition-list formatter supporting hanging indents for short labels and block headers for long labels, eliminating repetitive labels across wrapped lines. Shorten the longest 57-character label to prevent label overflow at width 40.
- **UX-01 (Catalog Grouping):** Visually segment the 18 OpenCode profiles by functional role group in list views while preserving continuous 1–18 numeric selection.
- **UX-02 & UX-03 (Copy De-duplication & Concise Safety):** Clean catalog example strings, remove duplicate `"Optional preset:"` prefixes, and condense verbose disclaimers into scannable badges. Deliberately update existing test assertions that enforced the duplicate prefix.
- **UX-04 (Navigation Consistency):** Accept `b`, `B`, and `Enter` on the explorer detail screen to return to the list, matching muscle memory.
- **UX-05 (Stable Numeric IDs with Visual Grouping):** Maintain stable menu choice IDs (`1`=Install/Preview default, `2`=Restore, `3`=Logs, `4`=About, `5`=Updates, `6`=Explore, `q`=Quit) to preserve all automated journeys, scripts, and trace contracts, while adding visual section dividers (`GET STARTED`, `MANAGE`, `HELP`).
- **UX-06 (Actionable Next Steps):** Provide copyable next-action commands and usage hints on detail cards.
- **UX-07 (Vertical Whitespace):** Normalize blank-line emissions across sub-screens to prevent double blank lines before menu boundaries.
- **UX-08 (Clear State Terminology):** Standardize destination file status badges (`Template available`, `Installed (matches source)`, `Installed (modified)`).
- **UX-09 & UX-10 (Empty State & Exit Messaging):** Add operational context to `No backups found.` and provide friendlier exit text on terminal stdout while strictly preserving legacy plain-text fixtures for non-TTY stdout.
- **UX-11 (Scrollback Reduction):** Reduce unnecessary vertical sprawl via compact menu presentation and definition blocks.
- **Decoupled TUI Architecture & Nested Module Spike:** Establish `experiments/tui/` with its own nested `go.mod`, testing Charm v2 read-only catalog browsing without adding dependencies or modifying root `go.mod`/`go.sum`.
- **Toolchain Migration Blueprint:** Comprehensive catalog of all hardcoded toolchain/image/digest locations across the repository.

### Explicitly Excluded
- **No Root Module Dependency Changes in B1 or B2:** The root `go.mod` remains `go 1.23` with zero external dependencies; no `go.sum` is created in the repository root.
- **No Installer Core Mutation:** `internal/skillsync` logic (backup manifest, file copying, tree comparison, rollback, and preview calculation) remains untouched.
- **No Breaking Non-TTY Changes:** Piped stdout, stderr, CI runs, and automated flags remain byte-identical and unstyled. Plain fixtures (`guide_plain_compatibility_test.go`) are untouched.
- **No Automatic Network Access or Agent Runtime Claims:** Startup remains completely offline; templates never claim active process or sandbox enforcement.

---

## 3. Functional Requirements

### FR-01: Definition Formatting & Narrow Terminal Bounds
- Short labels (≤ 16 runes, e.g. `Purpose:`, `When to use:`, `Example:`) must render with the label on line 1 and subsequent wrapped lines padded with spaces equal to the label width.
- Long labels (> 16 runes, e.g. `Safety:`) must print the label on its own line, with subsequent text wrapped and indented by 2 spaces.
- The former 57-rune label `Declared intent (educational; not an enforced guarantee):` must be shortened to `Safety:`, ensuring that at `COLUMNS=40`, no line exceeds the 40-column boundary.
- Indivisible path and command tokens must never be broken across lines.

### FR-02: Structured Profile & Skill Catalog Presentation
- The OpenCode profiles list must display items grouped under their authoritative purpose groups (`Main communicator`, `Read-only investigation`, `Implementation/documentation`, `Test assistance`, `Independent/supplementary review`).
- Numbered selection must remain 1-based and continuous (1 through 18) so direct numeric entry continues to work without nested submenus.

### FR-03: Copy Deduplication & Progressive Safety Disclosure
- Catalog entries in `guide_catalog.go` must eliminate redundant text prefixes (e.g. `Example:` must not be followed by `Ask your main agent to...`).
- Preset profiles must display `(optional preset)` as a single distinct badge; the `"Optional preset: "` prefix must be removed from `Purpose`. Existing tests in `guide_catalog_test.go` asserting the old prefix must be updated deliberately.
- Safety disclaimers on detail cards must display as a concise badge (`Safety: Advisory template · No automated execution · Requires human authorization`), pointing to concepts for extended legal context.

### FR-04: Universal Navigation & Frictionless Prompts
- The detail view prompt `Press Enter to return to the list, or q to exit guide:` must accept `b`, `B`, and `Enter` to return to the parent category list.
- Empty states (e.g. `No backups found.`) on terminal stdout must provide operational context explaining when backups are created.
- On terminal stdout, exiting via `q` must emit `Exited.` rather than `Cancelled.`. On non-TTY stdout, legacy plain output (`Cancelled.`) is preserved byte-for-byte.

### FR-05: Stable Numeric IDs with Visual Grouping
- Menu numeric choices and defaults remain strictly stable:
  - `1) Preview and install skills   (default)`
  - `2) Restore from a backup`
  - `3) View latest install or restore log`
  - `4) About and getting started`
  - `5) Check for CLI updates (contacts GitHub)`
  - `6) Explore skills and agent profiles`
  - `q) Quit`
- Visual visual hierarchy is achieved via section subheadings in the terminal view (`GET STARTED`, `MANAGE`, `HELP`) without altering numeric values, trace events, or automated test journey inputs.

### FR-06: Actionable Next Steps & Clear State Badges
- Detail cards must render a standardized status badge (`Template available (Not installed)`, `Installed (matches source)`, `Installed (modified)`).
- Detail cards must include a copyable next-action hint indicating how to preview or install the item.

### FR-07: Nested Module TUI Architecture & Isolation
- The experimental TUI spike must be placed in a nested module (`experiments/tui/go.mod`) with an opt-in task in `Taskfile.experimental.yml`.
- The root `go.mod` and root `go.sum` must remain completely untouched.
- The TUI must interface with `internal/skillsync` strictly via read-only queries (`skillsync.InspectSelection(...)`), with zero mutation or backup logic in UI components.
- The TUI spike must be accessible only via an explicit environment toggle (`AGENTIC_SDD_TUI=1`), and only when `env.StdoutTTY` is true and `AGENTIC_SDD_FORCE_INTERACTIVE` is not set to an automated script mode.

---

## 4. Observable Acceptance Criteria

| ID | Requirement | Verification Method |
|---|---|---|
| **AC-01** | Field labels never repeat on wrapped lines; rendered line length never exceeds terminal width at widths 40, 60, and 80. | Unit tests in `guide_presentation_test.go` checking rendered lines against width bounds. |
| **AC-02** | OpenCode profiles list renders 18 profiles segmented by 5 clear section headers while preserving continuous 1–18 numeric selection. | Terminal output test asserting headers appear and selecting `1` through `18` resolves to correct IDs. |
| **AC-03** | No copy stutter in detail views; preset profiles do not duplicate `"Optional preset:"`; `guide_catalog_test.go` updated and passing. | `go test ./cmd/agentic-sdd -run '^TestGuideCatalog'` PASS. |
| **AC-04** | Detail cards display concise safety badges rather than 5-line disclaimers. | Text matching test asserting concise badge is emitted. |
| **AC-05** | Entering `b` or `B` at the detail card prompt returns to the list view without an invalid choice error. | Prompter fixture test passing `"b\n"` and asserting clean return. |
| **AC-06** | Main menu renders visual section headers while keeping numeric choices `1`–`6` and default `1` unchanged. All automated journeys J1–J10 pass. | `sh tests/guide-e2e.sh` PASS without modifying script inputs. |
| **AC-07** | Sub-screens returning to the main menu emit exactly one blank separator line, eliminating double blank lines. | Buffer output test checking line endings between About/Updates and menu dividers. |
| **AC-08** | Non-TTY stdout and stderr remain byte-identical to Feature 010/011 baselines. | `guide_plain_compatibility_test.go` PASS without edits. |
| **AC-09** | Detail cards display copyable next-action hints and clear status badges. | Unit tests asserting next-action copy and status strings. |
| **AC-10** | Nested `experiments/tui` module compiles independently in an environment with Go 1.26 without affecting root `go.mod` or root `go.sum`. | `task -t Taskfile.experimental.yml tui:test` (opt-in only). |
| **AC-11** | Supported-core test suite (`task test`, `task vet`, `git diff --check`, `tests/docker-e2e.sh`) passes cleanly. | Standard CI gate execution. |

---

## 5. Approval

*This section records the single combined human approval for spec.md, plan.md, and tasks.md.*

- **Human Approval Status:** `Approved by owner (spec r2, plan r2, tasks r2)` on 2026-11-10.
- **Reviewed Scope:** Feature 012 specification r2, plan r2, and tasks r2.
- **Approved Batch:** Batches B1 and B2 authorized for parallel execution (parallel free and DeepSeek implementation subagents).
- **Authorized Actions:** Execution of Batch B1 tasks (T01–T05) within the zero-dependency Go stdlib boundary, and Batch B2 tasks (T06–T08) within the isolated nested module `experiments/tui/` without modifying root `go.mod` or root `go.sum`.
