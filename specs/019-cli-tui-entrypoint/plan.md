# Feature 019 — Plan

**Revision:** DRAFT r2, 2026-11-13 — naming/migration amendment. Input: [spec.md r2](spec.md), [naming study](naming-and-migration-study.md).
**Approval:** [spec §6](spec.md#6-approval) pending; N-01–N-06 require confirmation. No implementation authorized.

## 1. Approach

This is a coordinated naming, migration and TUI-composition change:

- Canonical installed commands become `asdx` (TUI) and `asdx-core` (CLI).
- Legacy `agentic-sdd` and `agentic-sdd-tui` remain deterministic wrappers for one deprecation window. `agentic-sdd` must never silently become the TUI.
- One paired archive contains all four names; the cask links all four.
- Existing state (`~/.agentic-sdd/`) and `AGENTIC_SDD_*` remain unchanged.
- The TUI gains a Guide destination that routes into existing screens; no installer/write logic is copied.
- Active help/docs migrate once, with a single migration table.

**Technical verdict (main agent): Needs changes in the current naming/packaging/entrypoint; the revised r2 design is ready for compatibility review and combined approval.** This is not an independent Opus verdict or human approval.

### 1.1 Source-path decision

Keep root CLI source at `cmd/agentic-sdd` and nested TUI source at `experiments/tui/cmd/tui-companion`; change built/installed names only. Renaming source directories would force a supported-core gate-list change (008 fence), package-path churn and extra review risk without improving the installed contract. Source builds use explicit output names (`go build -o bin/asdx-core ./cmd/agentic-sdd`; local pair task).

### 1.2 Artifact contract

| Archive | Installed name | Source/role | Distribution marker |
|---|---|---|---|
| Paired | `asdx` | nested TUI; zero-argument human entrypoint | generic |
| Paired | `asdx-core` | root CLI; canonical scriptable CLI | generic |
| Paired | `agentic-sdd` | legacy CLI wrapper → `asdx-core` | generic |
| Paired | `agentic-sdd-tui` | legacy TUI wrapper → `asdx` | generic |
| — | `agentic-sdd-core` | never shipped; not a compatibility name | — |

One archive, unique universal names, one cask. The former Homebrew marker no longer changes bare-launch behavior; no duplicate CLI build is retained solely for provenance. Cask quarantine covers every executable. `release:snapshot` is the packaging proof.

## 2. Canonical CLI and build identity

- Introduce one package-level canonical CLI name (`asdx-core`) and use it in usage, topic help, hints and copyable commands.
- Bare `asdx-core` preserves the historical preview default. Remove the Homebrew-only zero-argument TUI branch from the core dispatcher; the TUI is a separate executable.
- Retain `asdx-core tui` as the explicit identity-verified launcher. It resolves sibling `asdx`, checks regular file/executable/identity, forwards signals and exit codes, and never searches PATH or falls back to preview.
- `asdx` accepts only TUI options, `--help`, `--version`, `--protocol-info`. CLI-shaped arguments fail with a hint naming `asdx-core`.
- `buildinfo`: `ProductCLI = "asdx-core"`, `ProductCompanion = "asdx"`; bump `ProtocolRevision` to `2`; update `RebuildPairHint`, protocol tests and `--version` output.
- Remove `Distribution.AllowsInteractiveDefault` and `DistributionHomebrew` entirely; keep only dev/generic markers. `internal/skillsync` manifests do not store distribution, so no compatibility reason remains. Update `buildinfo_test.go`, `tui_test.go`, `release_config_test.go` and the `homebrew-default` trace expectation. `asdx-core` remains preview-default for every distribution.
- Nested TUI entrypoint: program/usage name, product label, `--help`/`--version`/`--protocol-info`, unknown-positional refusal with the core hint, and non-TTY/forced-interactive copy that names `asdx-core preview` while preserving `stdin is not a terminal`.

## 3. Legacy wrapper architecture (N-04 resolved)

Use **dual-build legacy mode**, not a new wrapper package:

- Build `asdx-core` and `agentic-sdd` from the existing root `cmd/agentic-sdd` source; inject a `legacyTarget=asdx-core` build value into the `agentic-sdd` build.
- Build `asdx` and `agentic-sdd-tui` from the existing nested TUI source; inject a `legacyTarget=asdx` build value into the `agentic-sdd-tui` build.
- The 11-package supported-core gate and trusted-CI contract stay unchanged; no new root package or 008 fence update.
- Legacy mode resolves the running path with `filepath.EvalSymlinks(os.Executable())` (Homebrew links are symlinks), verifies the target is a same-directory regular executable, writes a concise deprecation/compatibility notice to stderr, and replaces the process with `syscall.Exec` on POSIX. No shell, PATH search, or silent fallback.
- `agentic-sdd` targets `asdx-core`; `agentic-sdd-tui` targets `asdx`. The legacy CLI never opens the TUI; the legacy TUI never becomes the CLI.
- Exact top-level `--help`/`-h`/`--version` on a legacy name identify the compatibility name and canonical target; subcommand help, `--` arguments and JSON commands are untouched. `agentic-sdd-tui --protocol-info` forwards pure JSON to `asdx --protocol-info` with no deprecation notice on stderr; the legacy CLI does not perform a protocol handshake because `asdx-core` has no `--protocol-info`.
- Missing/non-executable/mismatched targets fail closed with rebuild guidance. Deprecation is stderr-only; stdout/JSON/exit behavior is unchanged.
- Wrappers are removed only after the announced deprecation window; state/env names remain stable. The cost is duplicate legacy binaries; the benefit is no gate/CI expansion.

## 4. TUI Guide composition

Add `RouteGuide` as a fifth navigation destination (`5`, “Guide”, “Step-by-step setup & recovery”). `GuideModel` is presentation-only:

| Guide action | Existing destination |
|---|---|
| Install or update | `RouteInstall` → `installSectionInstall` (`InstallModel`) |
| Restore a backup | `RouteInstall` → `installSectionBackups` (`RestoreModel`) |
| View latest log | `RouteActivity` (`ActivityModel`) |
| Quick start tour | existing `OnboardingModel` overlay |
| About / update check | `RouteActivity`; no new network path |

Rules:

- No new `skillsync`/`appops` call, write request, preview token or confirmation policy.
- Add explicit `guideReturn`/origin state; `Esc`/`b` from a guide-opened child returns to Guide. Reset on ordinary navigation.
- `5` opens Guide; `g` keeps online-README behavior. Guide may expose the README explicitly.
- `routeScroll` must become route-count-agnostic (`map[Route]int` or enum-tied structure) before `RouteGuide`; configure Guide dimensions in `switchRoute`/layout and add render/fallback cases.
- Guide install calls `InstallModel.ResetJourney` (or equivalent) and opens at client selection, never a completed result/stale preview.
- Reuse `Theme`, `Section`, `Paragraph`, `actionBinding`, viewport sizing and ASCII/plain behavior. No new dependency.

## 5. Help and documentation

### 5.1 Help surfaces

- `cmd/agentic-sdd/helptext.go`: `task`, `run`, `review` topics, canonical names, migration note.
- `cmd/agentic-sdd/{task,run,review,execution_cli}.go`: usage lines, flag-set names, examples, exit codes.
- `main.go` global usage/hints; `logs.go`, `guide.go`, `guide_explorer.go`, `guide_presentation.go`, `updates.go`, `fleet.go`, `fleet_plan.go`, `mcp.go`, `mcp_write.go`, `opencode.go` user-facing commands.
- TUI `activity.go` `SupportedCommands`: canonical names plus legacy table; `NavigationShortcuts` gains Guide (`5`) and keeps `g` for online README.
- TUI copyable hints in `app_screens.go`, `onboarding.go`, `install.go`, `catalog.go`, `library_help.go`, `presentation.go`, `mcp_screens.go`: canonical core commands. Audit all non-test TUI source.
- `Taskfile.yml`, `Taskfile.experimental.yml` messages, `tests/build-qa-image.sh`, `tests/e2e-in-container.sh`, `tests/vnext-container.sh`: build/output names and command strings. Do not change `harness/checks.json`'s trusted `build-darwin` required contract.
- One CLI program-name constant; source-string audits cover core and TUI non-test files. Tests that assert command strings update atomically.

`task help` must explain `task brief --run ID --task T01 [--force]`, owning-run requirement, registration/digest recovery, direct-scope refusal, and observation status. `run`/`review` help preserves accepted Feature 017 semantics.

### 5.2 Active documentation

Update:

- `README.md` (install, quick start, TUI, command table, source build, migration).
- `docs/getting-started.md`, `docs/installer-reference.md`, `docs/guide-flows.md`.
- `docs/execution-runs.md` (canonical examples; names note).
- `docs/tui-architecture.md` (active entrypoint/package prose; historical tables remain historical).
- `docs/orchestration-guide.md`, `docs/mcp-tools.md`, `docs/fleet-manager-tui.md`.
- `opencode-pack/README.md`, `opencode-pack/routing.md`, `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`; active orchestration references.
- `AGENTS.md` release/install text; Taskfile user-facing text.
- In-TUI copyable hints.

Classify `docs/vnext-human-test-guide.md`, `docs/vnext-human-test-plan.md`, dated research records and release reports as historical; do not rewrite accepted test instructions.

Migration table:

| Before | After |
|---|---|
| `agentic-sdd preview|apply|guide|run|task|review` | `asdx-core …` (legacy name still works via wrapper) |
| `agentic-sdd tui` | `asdx` (legacy `agentic-sdd-tui` wrapper works) |
| `agentic-sdd-tui` | `asdx` |
| `agentic-sdd-core` | never shipped; use `asdx-core` |
| `~/.agentic-sdd/`, `AGENTIC_SDD_*` | unchanged |

## 6. Packaging and release configuration

- `.goreleaser.yml`: four builds (root core `asdx-core`, root legacy `agentic-sdd`, nested TUI `asdx`, nested legacy `agentic-sdd-tui`), four universal names, one paired archive; legacy builds inject `legacyTarget`. No new package/gate entry.
- `homebrew_casks`: one archive; `binaries: [asdx, asdx-core, agentic-sdd, agentic-sdd-tui]`; quarantine every executable; keep cask name `agentic-sdd` and GitHub repo/release endpoint `nawodyaishan/agentic-sdd` because `updates.go`/appops depend on them; no live tap action.
- `internal/buildinfo/release_config_test.go` and `buildinfo_test.go`: canonical names, product labels, protocol revision, 4 builds/universals, 1 archive, 4 cask binaries/hooks; remove `AllowsInteractiveDefault`/Homebrew special-case test expectations; keep only dev/generic markers.
- `Taskfile.experimental.yml`: `tui:pair` builds four executables with generic markers and no Homebrew default; update PTY tasks/messages. `Taskfile.yml` `build` outputs `bin/asdx-core`; `build-darwin` and `harness/checks.json` stay unchanged (`bin/agentic-sdd_darwin_*`) during the deprecation window. Source-run commands can still use `./cmd/agentic-sdd`.
- `tests/build-qa-image.sh`, `tests/qa-container/Dockerfile`, `tests/qa-container/entrypoint.sh`, `tests/e2e-in-container.sh`, `tests/vnext-container.sh`, `tests/vnext-model-free-checks.sh`, `tests/guide-e2e.sh`, `tests/guide-e2e-in-container.sh`, `tests/tui/test_launcher_lifecycle.py`, `tests/tui/test_terminal_lifecycle.py`, package scripts: canonical/wrapper names and four-name fixtures.
- README source-build/local-pair instructions use explicit output names.
- `task -t Taskfile.experimental.yml release:snapshot` is the required nonpublishing archive/cask proof when the reviewed Go/GoReleaser toolchain is available; update its completion message for the four-name pair. Otherwise `not_run` with no archive claim. Release/tag/tap/cask publication remains out of scope.

## 7. Compatibility and failure behavior

- **Additive migration:** old `agentic-sdd` CLI and `agentic-sdd-tui` TUI keep working through wrappers; canonical names are `asdx`/`asdx-core`.
- **No hijack:** `agentic-sdd` never becomes the TUI; `agentic-sdd-tui` never becomes the CLI.
- **TUI refusal:** `asdx preview` prints an error and the `asdx-core` hint, exits `2`, writes nothing.
- **Missing/mismatched sibling:** `asdx-core tui` and wrappers fail with rebuild-pair guidance, never preview.
- **Non-TTY:** bare `asdx` fails with guidance; never opens an alternate screen in a script.
- **Protocol:** old/new pairs mismatch because product labels/protocol revision changed; both binaries must be replaced together.
- **State/env:** no forced migration; old backups/logs and `AGENTIC_SDD_*` remain usable.
- **Uninstall/rollback:** Homebrew cask conflict checks and documented archive pre-extract checks protect unowned paths; `brew uninstall --cask agentic-sdd` leaves `~/.agentic-sdd/` untouched; previous release can be reinstalled. Wrappers refuse PATH/symlink targets at runtime.

## 8. Verification strategy

| Area | Checks |
|---|---|
| Name availability | GitHub/npm/PyPI/Homebrew/trademark checks; local PATH shadowing; no `asd`/`asdd` install. |
| Canonical CLI/help | `asdx-core` usage/topics/task help; old command/exit behavior; no TUI default branch. |
| Legacy wrappers | Target resolution, stderr-only deprecation, stdout/JSON/exit unchanged, help/version identity, no shell/PATH, missing target failure. |
| Pair launcher | `asdx-core tui` sibling `asdx`, product labels, protocol revision, missing/mismatch/non-executable, signals/exit. |
| TUI entrypoint | Nested `main_test` program/usage/version/protocol; `tuirun` gating; CLI-shaped argument refusal; terminal-lifecycle substring. |
| TUI Guide | Route navigation, install/restore/activity/onboarding reuse, back-to-Guide, `5`/`g`, ResetJourney, no writes/network, ASCII/plain/narrow. |
| Packaging | Static tests; `goreleaser check`; `task -t Taskfile.experimental.yml release:snapshot` when toolchain available; archive/cask four-name membership; wrappers regular files; checksums. |
| State/migration | Old backups/logs readable; `AGENTIC_SDD_*` honored; no forced rename; uninstall/rollback. |
| Docs | Link and command-string audit of active docs/TUI hints; one migration table; historical files untouched. |
| Gates | `task test`, `task vet`, `git diff --check`, nested `tui:test`; focused race where warranted. No live release/tap/Homebrew check. |

## 9. Risks

| Risk | Mitigation |
|---|---|
| Double rename if names change after B1 | N-01–N-06 and direct name verification before implementation; no B1 until confirmed. |
| Legacy script breakage | Option A wrappers; `agentic-sdd` stays CLI; stderr-only deprecation; rollback. |
| Wrapper hijack/PATH search | Same-directory regular executable; no shell/PATH; identity checks; unowned-path refusal. |
| Four names confuse packaging/cask | One archive, unique universal names; static and snapshot proof; one migration table. |
| Protocol drift | Product labels/protocol revision bump; paired tests; rebuild guidance. |
| Guide duplicates installer logic | Route-only Guide; explicit reset; existing screens retain write/preview ownership. |
| Fifth route outgrows fixed shell state | Replace/widen `routeScroll`; render/fallback and route-count tests before Guide. |
| State migration scope creep | Preserve `~/.agentic-sdd`/`AGENTIC_SDD_*`; no forced rename; separate approval for any later change. |
| Wrapper adds a root package/gate churn | N-04 uses dual-build legacy mode from existing sources; no new 008 package or trusted-CI contract change. Cost is duplicate legacy binaries. |
| Live release unauthorized | Local snapshot/static checks only; tag/tap/cask publish separately authorized. |

## 10. Specialists and tools

| Work | Specialist | Tools / why |
|---|---|---|
| Root Go CLI, wrappers, launcher, buildinfo, packaging tests | `golang-pro` (installed) | Standard-library Go, process/path safety, focused tests. |
| TUI Guide composition/navigation | `ui-ux-designer` (installed) + existing `experiments/tui` patterns | Reuse screens, responsive/accessibility checks; no new design system. |
| Documentation migration | `writing-for-agents` (installed) only if active skill/reference wording changes; otherwise repository docs patterns | Precise agent instructions; historical records untouched. |
| Independent draft/compatibility review | Gemini Hydrogen XHigh reviewers, then normal SDD review roles | Read-only migration/packaging compatibility checks; no edits/approval. |

Tools: CodeGraph, local `go test`/`go vet`, nested `tui:test`, Taskfile, `goreleaser check`/`release:snapshot`, link/command audit. No browser or external research beyond the existing naming study unless name verification needs a registry check.

## 11. Delivery outline (not authorization)

| Batch | Scope | Gate |
|---|---|---|
| **B0 — Name verification** | Direct registry/GitHub/Homebrew/trademark/PATH checks for `asdx`/`asdx-core`; no production edits. N-01–N-06 are confirmed at combined approval; B0 only verifies the chosen name. | Human decision; no implementation |
| **B1 — Canonical entrypoint, wrappers, identity and help** | `asdx`/`asdx-core`, legacy wrappers, protocol/product labels, core help, packaging config, PTY harness, focused/nested tests | Human review; no live release |
| **B2 — TUI Guide and command reference** | `RouteGuide`, Guide composition, `routeScroll`, ResetJourney, `5`/`g`, Activity reference, nested tests | Human review after B1 |
| **B3 — Active docs and integrated verification** | README/docs/skill references, migration table, state/rollback checks, paired build and gates | Human review; acceptance separate |

B0 must complete before B1. B1 must land before B2 because naming/protocol and wrapper contracts are shared. B3 documents and verifies the integrated result. No batch starts from this draft.
