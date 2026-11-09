# Tasks: Feature 012 — CLI UI/UX Remediation & Hybrid TUI Experience

**Revision:** Draft r2 (Addressing Architecture Review findings)  
**Inputs:** [spec.md](spec.md) draft r2, [plan.md](plan.md) draft r2  
**Approval Pointer:** [spec.md §5](spec.md#5-approval), the single approval record.  

---

## Batch Overview

| Batch | Objective | Tasks | Target Outcome | State |
|---|---|---|---|---|
| **B1** | **Stdlib UI/UX Remediation & Defect Fixes** | T01–T05 | Correct text wrapping, copy stutter, list grouping, universal `b` navigation, visual menu headers, and terminal-only messages in zero-dependency stdlib Go. | `awaiting human review — delivered & verified` |
| **B2** | **Hybrid TUI Architecture & Nested Module Spike** | T06–T08 | Establish decoupled `internal/tui` boundary in nested module `experiments/tui/`, comprehensive toolchain migration roadmap, and isolated Bubble Tea v2 prototype. | `awaiting human review — delivered & verified` |

---

## Batch B1 — Stdlib UI/UX Remediation & Defect Fixes

- **Outcome:** All 12 acute UI/UX issues resolved within the existing zero-dependency Go stdlib architecture. Numeric menu IDs (`1`–`6`, `q`) and defaults remain stable, preserving all automated container journeys J1–J10. Non-TTY plain output remains byte-identical.
- **Dependencies:** Combined human approval of spec/plan/tasks and explicit B1 execution authorization.
- **Completion Conditions:** AC-01 through AC-09 pass cleanly; `task test`, `task vet`, `git diff --check`, `guide_plain_compatibility_test.go`, and both offline container test suites pass without gate or journey script modifications.
- **State:** `awaiting human review — delivered & verified`.
- **Next Action:** Owner reviews the remediated terminal UX and B2 prototype evidence.

### Tasks: Batch B1

| ID | Task Description | Dependencies | Verification & Target Files | Status |
|---|---|---|---|---|
| **T01** | **Red Stage Tests for Layout & Navigation:** Add test cases to existing `guide_presentation_test.go` and `guide_ux_test.go` asserting definition list wrapping, hanging indents at widths 40/60/80 (max line length ≤ width), catalog copy deduplication, and detail `b` key navigation. | B1 authorization | `cmd/agentic-sdd/guide_ux_test.go`<br>`cmd/agentic-sdd/guide_presentation_test.go`<br>`go test ./cmd/agentic-sdd -run '^(TestGuideUX|TestGuidePresentation)'` (assert red stage) | completed |
| **T02** | **Unified Definition Formatter (DEF-01):** Implement `printDefinition` in `guide_presentation.go`. Short labels (≤ 16 runes) format with hanging indent; long labels format as header + 2-space indented block. Shorten 57-rune label to `Safety:`. Replace duplicate wrapping logic in `guide.go` and `guide_explorer.go`. | T01 | `cmd/agentic-sdd/guide_presentation.go`<br>`cmd/agentic-sdd/guide.go`<br>`cmd/agentic-sdd/guide_explorer.go`<br>`go test ./cmd/agentic-sdd -run '^TestGuidePresentation'` | completed |
| **T03** | **Catalog Copy & Safety Badging (UX-02, UX-03):** Clean `teachingRegistry` entries in `guide_catalog.go`. Strip redundant prompt lead-ins, remove `"Optional preset: "` prefix from Purpose, and condense disclaimers to concise badges. Update existing assertion in `guide_catalog_test.go:96-108`. | T01 | `cmd/agentic-sdd/guide_catalog.go`<br>`cmd/agentic-sdd/guide_catalog_test.go`<br>`go test ./cmd/agentic-sdd -run '^TestGuideCatalog'` | completed |
| **T04** | **Grouped Catalog, Universal `b`, Status Badges & Next Steps (UX-01, UX-04, UX-06, UX-08):** Segment 18 OpenCode profiles by functional role group while preserving continuous 1–18 numeric selection; accept `b`, `B`, and `Enter` on detail screens; standardize destination status labels; add copyable next-step hints. | T02, T03 | `cmd/agentic-sdd/guide_explorer.go`<br>`cmd/agentic-sdd/guide_explorer_test.go`<br>`go test ./cmd/agentic-sdd -run '^TestGuideExplorer'` | completed |
| **T05** | **Visual Menu Grouping, Whitespace & Terminal Messages (UX-05, UX-07, UX-09, UX-10, UX-11):** Add visual section headers (`GET STARTED`, `MANAGE`, `HELP`) to terminal menu while keeping choice IDs `1`–`6` strictly stable; normalize blank lines; scope `Exited.` and contextual empty states strictly to terminal stdout while preserving non-TTY plain output byte-identical. Run full B1 test suite. | T04 | `cmd/agentic-sdd/guide.go`<br>`cmd/agentic-sdd/updates.go`<br>`cmd/agentic-sdd/guide_plain_compatibility_test.go`<br>`task test`<br>`task vet`<br>`git diff --check`<br>`sh tests/guide-e2e.sh` | completed |

---

## Batch B2 — Hybrid TUI Architecture & Nested Module Spike

- **Outcome:** Clean decoupled TUI architecture, isolated Charm v2 prototype spike in nested module `experiments/tui/go.mod` (root `go.mod` and root `go.sum` completely untouched), and documented comprehensive toolchain migration blueprint.
- **Dependencies:** B1 review completion and explicit B2 authorization.
- **Completion Conditions:** AC-10 and AC-11 pass; root module retains Go 1.23 and zero dependencies; `experiments/tui` interfaces cleanly with `internal/skillsync` solely through read-only queries.
- **State:** `awaiting human review — delivered & verified`.
- **Next Action:** Owner reviews the remediated terminal UX and B2 prototype evidence.

### Tasks: Batch B2

| ID | Task Description | Dependencies | Verification & Target Files | Status |
|---|---|---|---|---|
| **T06** | **TUI Architecture Interface & Domain Isolation:** Define `TUIConfig` and boundary types in `experiments/tui/`. Ensure zero filesystem or backup operations occur within UI components; interface strictly with `skillsync.InspectSelection(...)` and `skillsync.SummarizePlan(...)`. No changes to `internal/skillsync/`. | B2 authorization | `experiments/tui/config.go`<br>`experiments/tui/theme.go`<br>`internal/skillsync/` (read-only verification) | completed |
| **T07** | **Nested Charm v2 Spike & Opt-in Experimental Task:** Build an isolated Bubble Tea v2 / Bubbles list prototype demonstrating interactive catalog browsing, live `/` search, and `teatest` headless tests in `experiments/tui/go.mod`. Add opt-in task in `Taskfile.experimental.yml`. Root `go.mod` stays untouched. | T06 | `experiments/tui/screens/catalog.go`<br>`experiments/tui/catalog_test.go`<br>`Taskfile.experimental.yml`<br>`task -t Taskfile.experimental.yml tui:test` | completed |
| **T08** | **Comprehensive Toolchain Migration Blueprint & Full Gate Verification:** Document the complete toolchain upgrade roadmap across all 8 pinned image/digest/toolchain sites (`Taskfile.yml`, `ci.yml`, `trusted_ci.py`, `runner.go`, `release.yml`, etc.); verify standard core CLI gates remain 100% passing. | T07 | `docs/tui-architecture.md`<br>`task test`<br>`task vet`<br>`git diff --check`<br>`sh tests/docker-e2e.sh` | completed |

---

## Verification Commands & Acceptance Map

```sh
# Step 1: Unit verification for presentation and catalog
go test -count=1 ./cmd/agentic-sdd -run '^(TestGuideUX|TestGuideExplorer|TestGuideCatalog|TestGuidePresentation|TestGuidePlain)'

# Step 2: Core test suite (7 supported packages, 0 dependencies in root module)
task test
task vet
git diff --check

# Step 3: Offline container verification (all 10 journeys pass without modification)
sh tests/guide-e2e.sh
sh tests/docker-e2e.sh
```

### Acceptance Coverage
- **AC-01 (Hanging Indent / DEF-01):** Delivered by T02; verified by unit tests at widths 40, 60, and 80.
- **AC-02 (Grouped Profiles / UX-01):** Delivered by T04; verified by explorer tests.
- **AC-03 (Copy Stutter / UX-02):** Delivered by T03; verified by updated catalog tests.
- **AC-04 (Safety Badges / UX-03):** Delivered by T03; verified by card tests.
- **AC-05 (Detail `b` Key / UX-04):** Delivered by T04; verified by prompter tests.
- **AC-06 (Stable IDs & Visual Menu / UX-05):** Delivered by T05; verified by J1–J10 automated journeys in `guide-e2e.sh`.
- **AC-07 (Blank Lines / UX-07):** Delivered by T05; verified by output buffer tests.
- **AC-08 (Non-TTY Compatibility):** Verified across T02–T05 by `guide_plain_compatibility_test.go` without fixture changes.
- **AC-09 (Next Steps & Badges / UX-06, UX-08):** Delivered by T04; verified by unit tests.
- **AC-10 (Nested Module TUI Spike):** Delivered by T06–T07 in `experiments/tui/`.
- **AC-11 (Core Gates & Docker E2E):** Verified by T05 and T08.

---

## Continuation & Ledger State

- **Current State:** `awaiting human review — Batches B1 & B2 delivered and verified (2026-11-10)`
- **Approval Reference:** [spec.md §5](spec.md#5-approval), approved by owner on 2026-11-10.
- **Changed Paths:**
  - `cmd/agentic-sdd/guide_presentation.go`: unified `printDefinition` formatter with hanging indent (≤ 16 runes) and 2-space block indent (> 16 runes).
  - `cmd/agentic-sdd/guide_presentation_test.go`: added 5 tests asserting no line exceeds width at 40/60/80 cols.
  - `cmd/agentic-sdd/guide_catalog.go`: clean copy, deduplicated presets, concise safety badges.
  - `cmd/agentic-sdd/guide_catalog_test.go`: updated preset purpose and caveat assertions, added `TestTeachingExamplesDoNotRepeatLabels`.
  - `cmd/agentic-sdd/guide_explorer.go`: grouped profile listing with section headers, `b` key on detail screen, next-step copyable hints.
  - `cmd/agentic-sdd/guide_explorer_test.go`: added `TestGuideExplorerDetailBackNavigationAndGroupedHeaders`.
  - `cmd/agentic-sdd/guide.go`: visual menu section headers (`GET STARTED`, `MANAGE & LOGS`, `HELP`), normalized blank lines, terminal-only empty state context.
  - `cmd/agentic-sdd/updates.go`: normalized trailing whitespace before menu boundary.
  - `experiments/tui/`: nested Go 1.26 module (`go.mod`, `go.sum`, `config.go`, `catalog.go`, `screens/catalog.go`, `cmd/tui-spike/main.go`, `catalog_test.go`).
  - `docs/tui-architecture.md`: comprehensive 8-point toolchain migration blueprint.
  - `Taskfile.experimental.yml`: added `tui:test` task.
- **Verification Evidence:**
  - `go test -count=1 ./cmd/agentic-sdd`: PASS (all presentation, explorer, catalog, plain compatibility tests pass).
  - `task test`: PASS across all 7 supported core packages.
  - `task vet`: PASS across all 7 supported core packages.
  - `git diff --check`: PASS.
  - `sh tests/guide-e2e.sh`: PASS (all 10 automated journeys J1–J10 pass).
  - `sh tests/docker-e2e.sh`: PASS.
  - `task -t Taskfile.experimental.yml tui:test`: PASS (nested module tests pass).
- **Next Permitted Action:** Owner reviews the remediated terminal UX and B2 prototype evidence. No further batch or release operation is authorized.
