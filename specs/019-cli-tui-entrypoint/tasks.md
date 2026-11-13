# Feature 019 — Tasks

**Revision:** DRAFT r2, 2026-11-13 — naming/migration amendment.
**Parents:** [spec.md r2](spec.md), [plan.md r2](plan.md), [naming study](naming-and-migration-study.md).
**Approval reference:** [spec §6](spec.md#6-approval) — **pending**. N-01–N-06 require confirmation. This file is the execution ledger, not an approval record.
**Assignments/tools:** plan §10; no task-specific specialist exceptions.

## Batch ledger

| Batch | Tasks | Outcome | State | Next action |
|---|---|---|---|---|
| B0 | T00 | Verified `asdx`/`asdx-core` names and PATH safety | `awaiting human review — no PATH collision; registry checks partly not_run` | Review the name-verification evidence and residual registry/trademark gaps. |
| B1 | T01–T05 | Canonical `asdx`/`asdx-core`, legacy wrappers, protocol/packaging foundation and core help | `awaiting human review — full gate, race and PTY green` | Review the integrated canonical/legacy behavior and packaging config. |
| B2 | T06–T08 | Guide destination composed from existing TUI screens and updated command reference | `awaiting human review — nested tests green` | Review Guide routing, return behavior and TUI hints. |
| B3 | T09–T11 | Active documentation migration, migration/state evidence, paired build and integrated verification | `awaiting human review — docs/gates green; goreleaser not_run` | Review active docs, state/env evidence and the unrun snapshot boundary. |

**Execution rule:** owner authorized B0–B3 in parallel on 2026-11-13. Parallelism is by disjoint files, not simultaneous writers on one file. B1 shared contracts are frozen first; B2 Guide and B3 docs proceed concurrently where files are disjoint; parent integrates one code state per wave and stops for human review. No live release, tag, tap push, cask publication, real-home write or product model/network call is authorized by these documents.

## Implementation fleet (authorized 2026-11-13)

| Lane | Agent / model | Owns | Does not touch |
|---|---|---|---|
| B0 + packaging | `sdd-implement-free-1`, free Console model | T00 name/PATH verification, `.goreleaser.yml`, `internal/buildinfo/release_config_test.go`, `Taskfile.yml`, `Taskfile.experimental.yml`, packaging/container/guide/vnext scripts | Go entrypoint/buildinfo source, TUI app, docs, `harness/checks.json` |
| B1 core | `sdd-implement-2`, `opencode-go/deepseek-v4.1-flash#max` | T01 inventory, T02 canonical identity/launcher, T03 dual-build legacy mode; `cmd/agentic-sdd/{main,tui}.go`, `internal/buildinfo/{buildinfo,buildinfo_test}.go`, nested `cmd/tui-companion/{main,main_test}.go`, `cmd/internal/tuirun/run.go`, `cmd/agentic-sdd/tui_test.go`, TUI lifecycle scripts | help text, TUI app screens, packaging, docs |
| B1 help | `sdd-implement-9r-1`, `9router/ag/gemini-4-hydrogen-xhigh` | T04 help/TUI-hint migration in root help/task/run/review/log/guide/fleet/MCP/OpenCode files and `main_test.go`/help tests; canonical/legacy examples | `main.go`, `tui.go`, TUI app, packaging, docs |
| B2 Guide | `sdd-implement-9r-2`, `9router/ag/gemini-4-hydrogen-xhigh` | T06/T07 TUI Guide, `routeScroll`, `ResetJourney`, `guideReturn`, all TUI hint files/tests, `activity.go`/`modal.go` Guide additions | nested `cmd/tui-companion`, `tuirun`, root CLI, packaging, docs |
| B3 docs | `sdd-implement-free-2`, free Beam model | T09 README/docs/AGENTS/OpenCode routing/SKILL migration and migration table | code, Taskfiles, historical records |

The parent owns T01 contract freeze, T05/T08/T10 integration, final gates and T11 review. Every agent reports changed paths, commands/exits, deviations and blockers; no commits, pushes, releases, real-home writes or edits outside its lane.

## B0 — Name and migration verification

**Completion:** `asdx`/`asdx-core` availability and local PATH safety are verified against the already-approved N-01–N-06 design; no production changes.

### T00 — Verify canonical names and PATH safety

- [x] **T00** Verify the proposed canonical names.
  - Dependencies: combined approval of spec/plan/tasks and owner confirmation of N-01–N-06.
  - Allowed: read-only research and a decision note in this feature; no binary/source/docs rename.
  - Acceptance: direct GitHub org/repo, npm, PyPI, Homebrew, Repology/AUR, trademark and local PATH checks for `asdx`/`asdx-core` recorded; no `asd`/`asdd` install; name collision or PATH shadowing reported before B1. Wrapper shape, deprecation window and gate impact are already fixed by N-02–N-04 at combined approval, not selected here.
  - Verification: record sources/dates and exact commands; registry checks may use web search; no paid model call or live install.
  - Completion (2026-11-13, free lane): no `asdx`/`asdx-core` on PATH; only the expected incumbent `~/.local/bin/agentic-sdd`; npm/PyPI `asdx` and PyPI `asdx-core` returned 404; crates.io returned 403 (not verified); GitHub/Homebrew/Repology/trademark checks were not run beyond the naming study. No collision for the approved canonical names. Residual: complete registry/trademark checks before publishing.
  - Stop: name collision, PATH shadowing that cannot be resolved, owner rejects Option A, or B0 reveals a scope change requiring amendment.

## B1 — Canonical entrypoint, wrappers, identity and CLI help

**Completion:** one paired archive ships `asdx` + `asdx-core` plus legacy wrappers; protocol identity is truthful; core help uses canonical names; focused/nested/packaging checks pass.

### T01 — Repository-first inventory and contract reconciliation

- [x] **T01** Confirm live CLI/TUI/packaging paths and reconcile the naming amendment.
  - Dependencies: B0 complete and recorded.
  - Allowed: read-only source/config inspection; update plan §1 and this task's completion note.
  - Acceptance: current commit/dirty state recorded; paths for `main.go`, `tui.go`, `helptext.go`, `buildinfo`, GoReleaser, Taskfiles, README/docs and nested TUI entrypoint confirmed; wrapper/gate impact confirmed; any conflict stops B1.
  - Verification: `git status --short`; `git rev-parse HEAD`; CodeGraph/targeted reads; `goreleaser check` availability recorded.
  - Completion (2026-11-13, DeepSeek): baseline `870330eb424b` plus concurrent authorized lane changes confirmed; root/nested paths, buildinfo, GoReleaser, Taskfiles and lifecycle harness confirmed; shared `legacyTarget`/`legacyName`, four executable names and protocol revision 2 frozen before integration.
  - Stop: changed requirements, unconfirmed naming decision or overlapping ownership.

### T02 — Canonical dispatcher, pair identity and launcher

- [x] **T02** Rename installed roles in code/handshake without moving source directories.
  - Dependencies: T01.
  - Files: `cmd/agentic-sdd/{main,tui,helptext,guide_presentation,updates}.go`, `internal/buildinfo/*`, `cmd/agentic-sdd/*_test.go`, `experiments/tui/cmd/tui-companion/{main,main_test}.go`, `experiments/tui/cmd/internal/tuirun/run.go`, `tests/tui/test_launcher_lifecycle.py`, `tests/tui/test_terminal_lifecycle.py`.
  - Acceptance: `ProductCLI=asdx-core`, `ProductCompanion=asdx`, protocol revision bumped; bare core previews; `asdx-core tui` resolves/verifies sibling `asdx`; no Homebrew-only bare TUI branch and `AllowsInteractiveDefault` usage removed; `DistributionHomebrew` removed entirely (dev/generic only) with tests updated; `asdx` refuses CLI-shaped flags and positionals with an `asdx-core` hint; non-TTY copy names `asdx-core preview` while preserving `stdin is not a terminal`; PTY lifecycle expects bare `asdx` TUI, bare `asdx-core`/`agentic-sdd` preview, `asdx-core tui`/`agentic-sdd tui` TUI, `agentic-sdd-tui` TUI.
  - Verification: focused root tests for help/version/tui launch/refusal; nested `go test ./...` for main/tuirun; both lifecycle tests where runnable; `go vet` on both; `tui:pair` four-executable build.
  - Completion (2026-11-13, DeepSeek max): protocol revision 2 and `asdx-core`/`asdx` product labels; Homebrew bare-launch and distribution special case removed; `asdx-core tui` verifies sibling `asdx`; `asdx` refuses CLI flags/positionals with the core hint; non-TTY/forced copy names `asdx-core preview`; nested program/usage/product renamed. Four-name pair built and probed. Full root/nested packages, vet and lifecycle tests passed.
  - Stop: protocol/identity behavior cannot be made truthful or wrapper pair behavior is unspecified.

### T03 — Legacy wrappers and migration behavior

- [x] **T03** Implement dual-build legacy mode for `agentic-sdd` → `asdx-core` and `agentic-sdd-tui` → `asdx`.
  - Dependencies: T02; N-04 dual-build mode confirmed.
  - Files: existing `cmd/agentic-sdd` and nested TUI main/build files; `internal/buildinfo` only if legacy identity needs a shared helper; focused legacy-mode tests; archive/Taskfile integration coordinated with T04. No new root package.
  - Acceptance: `filepath.EvalSymlinks(os.Executable())` same-directory regular target; no PATH/shell; `syscall.Exec` on POSIX forwards streams/signals/exit; `agentic-sdd` never opens TUI; `agentic-sdd-tui` opens TUI; stderr-only deprecation; stdout/JSON/exit unchanged except exact top-level help/version identity; subcommand help/`--` untouched; legacy TUI `--protocol-info` forwards pure JSON with no deprecation on stderr; missing/non-executable target fails closed; no new 008 gate entry.
  - Verification: isolated fixtures for canonical/legacy CLI/TUI, exact top-level help/version, subcommand help, JSON/`--` arguments, deprecation, missing target, signal/exit, symlinked executable (Homebrew), PATH shadowing; root/nested focused tests.
  - Completion (2026-11-13, DeepSeek max): dual-built root/nested legacy modes resolve the canonical sibling through `EvalSymlinks(os.Executable())`, require a regular same-directory executable, `syscall.Exec` on POSIX, stderr-only deprecation, exact top-level help/version, pure-JSON legacy TUI `--protocol-info`, and fail-closed missing/mismatched targets. Windows builds compile and fail closed at runtime. No new 008 package/gate entry. Manual probes and 19 launcher + 10 terminal PTY tests passed.
  - Stop: legacy mode requires a new gate package, shell/PATH fallback, or changes canonical stdout/JSON/exit behavior.

### T04 — Core CLI help, TUI hints and packaging

- [x] **T04** Update canonical help/hints and release artifacts.
  - Dependencies: T02/T03.
  - Files: `cmd/agentic-sdd/{helptext,task,run,review,execution_cli,main,logs,guide,guide_explorer,guide_presentation,updates,fleet,fleet_plan,mcp,mcp_write,opencode}.go`, all TUI hint files `{app_screens,onboarding,install,catalog,library_help,presentation,mcp_screens}.go` and their active tests, `experiments/tui/activity.go` command reference; `.goreleaser.yml`, `internal/buildinfo/release_config_test.go`, `Taskfile.experimental.yml`, `Taskfile.yml`, `tests/build-qa-image.sh`, `tests/qa-container/Dockerfile`, `tests/qa-container/entrypoint.sh`, `tests/e2e-in-container.sh`, `tests/vnext-container.sh`, `tests/vnext-model-free-checks.sh`, `tests/guide-e2e.sh`, `tests/guide-e2e-in-container.sh`, `tests/tui/test_launcher_lifecycle.py`, `tests/tui/test_terminal_lifecycle.py`. Do not edit `harness/checks.json`'s `build-darwin` required contract.
  - Acceptance: `task help` explains `task brief`, run binding, registration/recovery, force/refusal and observation status; core/TUI examples use `asdx-core`; source-string audits cover all non-test core/TUI files and their tests; one archive has four names with unique universal outputs; cask/repo name remains `agentic-sdd`, links all four and quarantines each; build/container/guide/vnext scripts use canonical names; `task build` outputs `bin/asdx-core`, while `build-darwin`/`harness/checks.json` stay unchanged; no root dependency/gate expansion; B1's TUI string audit passes before T05.
  - Verification: focused help/usage/task tests plus audits; static release-config tests; `goreleaser check`; local `tui:pair`; `task -t Taskfile.experimental.yml release:snapshot` when reviewed toolchain is available; otherwise `not_run` with no archive claim.
  - Completion (2026-11-13, Gemini Hydrogen xhigh + free lane + parent): all root help/task/run/review/fleet/MCP/OpenCode strings use `asdx-core` with legacy notes; `task help` documents run binding, registration/recovery, `--force`, direct-scope refusal and observation status; TUI hint files use canonical names; `.goreleaser.yml` has four builds/universals, one archive and one four-name cask; `Taskfile` local `build` is `bin/asdx-core` while `build-darwin`/`harness/checks.json` stay unchanged; container/guide/vnext scripts migrated. `task test`/`vet`, `tui:pair`, `task build`, `go test ./...`, static release tests and `git diff --check` passed. `goreleaser check`/`release:snapshot` are `not_run` because goreleaser is not installed.
  - Stop: release config requires live tap/tag/cask publication or an unapproved gate change.

### T05 — B1 integrated review

- [x] **T05** Integrate and verify B1; stop for human review.
  - Dependencies: T02–T04.
  - Acceptance: AC-01/02/03/05/07 partial or complete as scoped; no source-path/gate churn; exact evidence and residual risks recorded.
  - Verification: `task test`; `task vet`; nested `go test ./...` and `tui:pty-test` where runnable; `git diff --check`; focused race if concurrency changed; `codegraph sync`.
  - Completion (2026-11-13, parent): integrated the four B1 lanes, updated stale root assertions, fixed `fleet_doctor.go`, and verified root/nested packages, race for `internal/execution`, PTY lifecycle, module integrity and `git diff --check`. B1 set to `awaiting human review`; no commit, release or apply.
  - Completion: set B1 `awaiting human review`; no B2 start, commit, release or apply.

## B2 — TUI Guide and command reference

**Completion:** Guide is a visible TUI destination that routes to existing journeys, `5`/`g` behavior is explicit, and no new write/installer logic exists.

### T06 — Guide model and route composition

- [x] **T06** Add `RouteGuide` and a presentation-only Guide hub.
  - Dependencies: B1 reviewed and explicitly authorized.
  - Files: `experiments/tui/{app,app_screens,actions,viewport,activity,onboarding,install,modal}.go`, new `guide.go`, focused Guide tests. Non-Guide TUI hint migrations and their tests belong to B1 T04.
  - Acceptance: AC-04 partial; Guide offers install/update, restore, latest log, quick-start tour, about/update; actions route into existing `InstallModel`/`RestoreModel`/`ActivityModel`/`OnboardingModel`; Guide install calls explicit `ResetJourney` and starts at client selection; `routeScroll` is route-count-agnostic and safe for five routes; no `skillsync`/`appops` call or write request from Guide.
  - Verification: nested `go test ./...` focused Guide/navigation tests; completed-result reset test; plain/ASCII/narrow layouts; no network. B1's TUI source-string audit remains green.
  - Completion (2026-11-13, Gemini Hydrogen xhigh): added fifth `RouteGuide`/key 5 and presentation-only `GuideModel` routing to install/restore/activity/onboarding; `routeScroll` is a `map[Route]int`; Guide entry configures size/theme/render; `ResetJourney` starts install at client selection and refuses active writes; `guideReturn` returns Esc/b to Guide. No new appops/skillsync/write/network path. Nested focused and full tests passed.
  - Stop: guide requires new installer logic or a second confirmation policy.

### T07 — Guide return, keys and command reference

- [x] **T07** Wire safe return behavior and update TUI help/reference.
  - Dependencies: T06.
  - Files: same TUI shell files as T06, `activity.go`, `modal.go`, focused tests.
  - Acceptance: `5`/navigation opens Guide; `g` keeps opening the online README; Guide exposes the README as an explicit action; Esc/b from a guide-opened child returns to Guide via a named origin state reset on ordinary navigation; non-guide behavior unchanged; Guide adds its entry to the existing canonical/legacy Activity reference; keyboard help lists `1–5` and Guide.
  - Verification: nested focused tests for route origin/reset, back behavior, `5`, `g`, online link, help strings, small widths.
  - Completion (2026-11-13, Gemini Hydrogen xhigh + parent): `5` opens Guide, `g` remains online README, Guide exposes README; Esc/b returns from guide-opened install/restore/activity/onboarding; Activity/list hints use asdx/asdx-core and legacy names; keyboard help lists 1–5; catalog standalone title uses asdx. Nested full tests passed; PTY passed against the four-name pair.
  - Stop: return target requires rewriting unrelated screen state or a new global mode.

### T08 — B2 integrated review

- [x] **T08** Integrate and verify B2; stop for human review.
  - Dependencies: T06–T07.
  - Verification: nested `task -t Taskfile.experimental.yml tui:test`; `go vet` nested; local pair PTY smoke where supported; root packages unaffected.
  - Completion (2026-11-13, parent): nested `tui:test`, nested `go vet`, focused Guide/navigation tests, and PTY launcher/terminal tests passed; root packages remained green. B2 set to `awaiting human review`; no commit or release.
  - Completion: set B2 `awaiting human review`; no B3 start, commit or release.

## B3 — Active documentation, migration and integrated verification

**Completion:** active docs describe canonical/legacy names and Guide; state/env compatibility is verified; historical records untouched; paired build and gates have honest evidence.

### T09 — Active documentation migration

- [x] **T09** Update active docs, migration table and active skill references.
  - Dependencies: B2 reviewed and explicitly authorized.
  - Files: `README.md`, `docs/{getting-started,installer-reference,guide-flows,execution-runs,tui-architecture,orchestration-guide,mcp-tools,fleet-manager-tui}.md`, `AGENTS.md` release text, `opencode-pack/README.md`, `opencode-pack/routing.md`, `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`, active orchestration references, relevant Taskfile messages.
  - Acceptance: AC-06; one migration table for canonical/legacy/state/env; no active instruction treats `asdx` as CLI or `agentic-sdd` as TUI; in-TUI hints updated; Homebrew cask/archive conflict checks and rollback/uninstall guidance documented; historical specs/handovers/reports/dated research/accepted vnext records unchanged; links resolve.
  - Verification: link/command audit; `git diff --check`; review active-vs-historical list.
  - Completion (2026-11-13, free Beam + parent): README, getting-started, installer-reference, guide-flows, execution-runs, TUI architecture, orchestration guide, MCP/fleet docs, AGENTS release text, OpenCode README/routing/SKILL/references migrated; README has one migration table covering canonical/legacy/state/env and cask/repo retention. Historical files untouched. Remaining active `go run ./cmd/agentic-sdd` references are source paths, not installed commands. Two pre-existing unresolved router links in the orchestrate SKILL.md remain outside this change.
  - Stop: historical rewrite or unapproved release claim.

### T10 — Migration, state and paired-build verification

- [x] **T10** Verify state/env compatibility, wrappers, rollback and packaging.
  - Dependencies: T09.
  - Files: verification evidence only; no production edits beyond fixes in scope.
  - Acceptance: AC-01/02/07/08/09 evidence; old `~/.agentic-sdd` backups/logs readable; `AGENTIC_SDD_*` honored; wrappers forward correctly; Homebrew cask conflict checks and archive pre-extract checks documented; isolated bin-directory fixture removes/reinstalls the four product-owned executables without touching unowned files; local pair identity matches; static release tests; archive/cask four-name membership proved by `task -t Taskfile.experimental.yml release:snapshot` or recorded `not_run`; root/nested gates; unrun live checks reported.
  - Verification: `task test`, `task vet`, `git diff --check`, nested `tui:test`, local `tui:pair`, `goreleaser check`; `task -t Taskfile.experimental.yml release:snapshot` when reviewed toolchain available; isolated state/env and bin-dir fixtures; `codegraph sync`.
  - Completion (2026-11-13, parent): `task test`, `task vet`, root `go test ./...`, `go test -race ./internal/execution`, nested `tui:test`/`go vet`, `tui:pair`, 19 launcher + 10 terminal PTY tests, `go mod verify` and `git diff --check` passed. Commit `c792190` (`feat(cli): rename to asdx with legacy wrappers and TUI guide`). `task verify` passed all six required checks; `task verify-trusted` passed against base `870330e` (evidence `/tmp/opencode/agentic-sdd-trusted-evidence-019/evidence/gate.json`). Manual four-name probes confirmed identities and legacy behavior. `goreleaser` is absent, so `goreleaser check` and `release:snapshot` are `not_run`; no archive/cask membership claim. Homebrew/tap/macOS quarantine remain unverified.
  - Stop: live release/cask/homebrew action, gate conflict or undocumented platform failure.

### T11 — B3 completion review and handover

- [x] **T11** Record completion evidence, independent review and next action.
  - Dependencies: T10.
  - Acceptance: AC-01–09 mapped to evidence; migration impact, deprecation window and residual risks recorded; no acceptance inferred from tests.
  - Verification: source/diff review, final gate rerun if code changed, independent reviewer evidence.
  - Completion (2026-11-13, parent): recorded the integrated evidence, residual `not_run` snapshot/registry gaps, unexpected unrelated test-skip edits, and the review boundary. B3 set to `awaiting human review`; handover written; no tag/release/push/apply.
  - Completion: set B3 `awaiting human review`; write handover; no tag/release/push/apply.

## Continuation

- Current stage: B0–B3 implementation complete; **awaiting human review**. Combined approval is recorded in [spec §6](spec.md#6-approval); N-01–N-06 are confirmed.
- Baseline: `d781a79`; implementation ran on the integrated dirty tree with exclusive lanes.
- Changed areas: root `cmd/agentic-sdd` identity/help/legacy mode; `internal/buildinfo`; nested TUI program/product/Guide/hints; `.goreleaser.yml`, Taskfiles and packaging/container/guide/vnext scripts; active README/docs/AGENTS/OpenCode skill references; Feature 019 records/handover.
- Checks (integrated state): `task test`, `task vet`, root `go test ./...`, `go test -race ./internal/execution`, nested `tui:test`/`go vet`, `tui:pair`, 19 launcher + 10 terminal PTY tests, `go mod verify`, `git diff --check`, `task verify` (six required checks) and `task verify-trusted` against base `870330e` all passed. Commit `c792190`. `goreleaser check`/`release:snapshot` are `not_run` because goreleaser is absent; no archive/cask claim. B0 direct checks: no PATH collision; npm/PyPI checks mostly 404, crates 403; full registry/trademark audit remains.
- Residual risks: live macOS Homebrew/cask/quarantine/tap behavior and archive membership unverified; Windows legacy mode fails closed at runtime; two pre-existing router links in the orchestrate skill remain unresolved; concurrent unrelated test-skip edits in `cmd/agentic-sdd/execution_cli_test.go` and `internal/execution/repository_test.go` were preserved and are not Feature 019 work.
- Next permitted action: human review of the integrated diff/evidence. No commit, release, tap, real-home apply or additional batch is authorized from this state.
