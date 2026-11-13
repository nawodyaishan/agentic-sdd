# Feature 019 — Short commands, safe legacy migration, and TUI Guide

**Revision:** APPROVED r2, 2026-11-13 — combined approval recorded in §6; B0–B3 implementation complete and **awaiting human review**. A green suite is not acceptance.
**Location:** `specs/019-cli-tui-entrypoint/spec.md`.
**Study:** [naming-and-migration-study.md](naming-and-migration-study.md) (advisory; adopted into this draft, not authority by itself).
**Amends:** accepted Feature 013 binary/launcher and packaging behavior, and accepted Feature 017 help/documentation presentation. It preserves their accepted authority and execution semantics.
**Numbering note:** `specs/018/` contains unrelated untracked research material; this feature uses the next free number.

> **Authority notice.** This is a proposed amendment, not authority to rename binaries, install wrappers, publish a release, change the Homebrew tap, or start implementation. One combined human approval of `spec.md`, `plan.md` and `tasks.md` is required, followed by explicit batch authorization and human review. The naming study's recommendation is adopted here as a proposal, not a completed owner decision.

## 1. Outcome

A shell user types **`asdx`** and gets the terminal UI. Automation uses **`asdx-core`** for the existing commands, flags and exit codes. During one deprecation window, existing names remain safe wrappers:

| Legacy name | Behavior |
|---|---|
| `agentic-sdd` | CLI wrapper to `asdx-core`; never silently becomes the TUI |
| `agentic-sdd-tui` | TUI wrapper to `asdx` |

The TUI gains a **Guide** destination that reuses the existing install, restore, activity/log and onboarding components. `task`, `run`, `review` help and active documentation describe the canonical commands and the legacy migration honestly.

**One-sentence success:** a human can open `asdx` and complete the guided journey, existing scripts using `agentic-sdd` keep working as CLI during the deprecation window, and automation uses `asdx-core` with truthful help.

## 2. Binding decisions and compatibility

| ID | Decision | Rule |
|---|---|---|
| **D-01** | `asdx` is the canonical TUI | A zero-argument real-terminal invocation opens the TUI. No Homebrew-only default and no `asdx tui` requirement. |
| **D-02** | `asdx-core` is the canonical CLI | All existing CLI commands and flags move here. Bare `asdx-core` retains the existing preview default. One paired archive and the cask expose the canonical names. |
| **D-03** | Legacy names are wrappers (Option A) | `agentic-sdd` is a CLI wrapper to `asdx-core`; `agentic-sdd-tui` is a TUI wrapper to `asdx` for one deprecation window. No silent role change. |
| **D-04** | Explicit migration, no silent delegation | CLI-shaped arguments passed to `asdx` fail closed with an `asdx-core …` hint. The legacy `agentic-sdd` wrapper continues to run the CLI, not the TUI. |
| **D-05** | Core `tui` compatibility | `asdx-core tui` may remain as the explicit, identity-verified launcher of sibling `asdx`; documented as compatibility. |
| **D-06** | Guide is a TUI destination | A fifth destination, Guide, is opened by key `5` and composes existing screens/onboarding. It adds no new write path, installer logic, network call or approval authority. `g` keeps its existing online-README meaning. |
| **D-07** | Documentation follows behavior | Active docs and in-TUI copyable hints use canonical names and the legacy table. Historical specs, handovers, release reports and dated/accepted test records stay historical. |
| **D-08** | One paired archive, four names | The archive ships canonical `asdx` + `asdx-core` and legacy wrappers `agentic-sdd` + `agentic-sdd-tui`; the cask links all four. The Homebrew cask and GitHub repository/release endpoint keep the `agentic-sdd` name. Unique universal names and release:snapshot proof are required. |
| **D-09** | State and environment compatibility | Keep `~/.agentic-sdd/` (backups, logs, manifests) and `AGENTIC_SDD_*` unchanged. Do not force a data-directory or environment-variable migration for a rename. |
| **D-10** | Wrapper behavior is deterministic | Legacy names are dual-built from the canonical sources, resolve the target in the same installation, never use a shell or PATH search, forward streams/signals/exit codes, preserve stdout/JSON, and emit deprecation only on stderr. Exact top-level `--help`/`--version` identify the compatibility name; `--protocol-info` on the legacy TUI forwards pure JSON with no deprecation notice; subcommand help and JSON commands are untouched. |
| **D-11** | Verify the name before implementation | Recheck registry/GitHub/Homebrew/trademark and local PATH shadowing for `asdx`/`asdx-core`; never install occupied `asd`/`asdd`. |
| **D-12** | One announced removal window | Wrappers remain for at least one minor release with release notes, upgrade/uninstall/rollback checks and an announced removal release; state remains stable after removal. |
| **D-13** | No new 008 gate package | Keep the legacy behavior in the existing root/TUI `main` packages via build-time legacy mode. Do not add a wrapper package to the 008 supported-core gate or change `harness/checks.json`'s trusted required contract. The cost is duplicate legacy binaries; the benefit is no gate/CI expansion. |

## 3. Actors and journeys

| Actor | Journey | Expected outcome |
|---|---|---|
| New human user | Runs `asdx` | TUI opens; Guide leads to install, restore, logs and quick start. |
| Existing human user | Runs `agentic-sdd-tui` | Legacy wrapper opens `asdx`, prints a stderr deprecation notice, preserves streams/exit. |
| Automation / CI | Runs `asdx-core preview|apply|run|task|review` | Existing command and exit behavior; no TUI. |
| Existing automation | Runs `agentic-sdd preview|apply|…` | Legacy CLI wrapper runs `asdx-core`; stdout/JSON/exit unchanged; stderr deprecation only. |
| Future consumer | Uses `asdx-core tui` or typed services | Identity-verified pair; no CLI-output parsing from screens. |

### 3.1 Non-goals

- No merge of the nested TUI module into the root Go 1.23 module.
- No `asd`/`asdd`/`asdf` installed default; no name that collides with an active CLI.
- No forced state-directory or `AGENTIC_SDD_*` rename.
- No new installer, backup, restore, fleet, MCP, execution or review logic.
- No new model/provider routing, background daemon, telemetry or self-update.
- No live release, tag, tap push, cask publication or Homebrew install in this feature.

## 4. Functional requirements

### FR-01 — Canonical command contract

Ship `asdx` (TUI) and `asdx-core` (CLI) from one paired archive, plus legacy wrappers `agentic-sdd` and `agentic-sdd-tui`. The TUI opens on a real interactive terminal with no arguments and supports its direct options (`--home`, `--repo`, `--fleet-config`, `--with-opencode`, `--opencode-free-first`, `--version`, `--help`, `--protocol-info`). `asdx-core` retains the existing CLI grammar, defaults, non-TTY behavior and exit codes.

**AC-01:** Archive/cask tests prove all four names and the paired identity; no `asd`/`asdd` artifact is installed.

### FR-02 — Pair identity and protocol

`asdx-core` launches only a same-directory, regular-file, executable `asdx` after the build-identity handshake. Product labels and protocol revision describe the new pair. Missing, non-executable or mismatched siblings fail closed with rebuild guidance. The legacy CLI wrapper resolves a same-directory regular `asdx-core` and `syscall.Exec`s it; it does not perform a protocol handshake because the CLI has no `--protocol-info`. The legacy TUI wrapper forwards `--protocol-info` unchanged to `asdx` so an older parent can verify or reject the pair cleanly.

**AC-02:** Focused tests cover matching/missing/mismatched pairs, protocol revision, product labels, wrapper targeting, exact top-level help/version behavior, legacy TUI `--protocol-info` forwarding, and no PATH/shell resolution. Non-TTY TUI copy names `asdx-core preview` while preserving the `stdin is not a terminal` test substring.

### FR-03 — Legacy migration behavior

`agentic-sdd <cli-command>` continues to run `asdx-core`; `agentic-sdd-tui` continues to open `asdx`. Deprecation notices go only to stderr and never change stdout, JSON or exit codes. `asdx` with CLI-shaped args refuses with an `asdx-core` hint. `asdx-core tui` remains explicit compatibility.

**AC-03:** Tests distinguish canonical and legacy behavior, wrapper identity/help/version, deprecation stderr, unchanged stdout/JSON/exit, and refusal of CLI-shaped args on `asdx`. No silent delegation or role reversal.

### FR-04 — TUI Guide reuses existing components

Add Guide as a fifth navigation destination opened by key `5`/navigation. It composes existing TUI models and offers install/update, restore, latest-log/activity, quick-start onboarding, and about/update actions by routing to existing journeys. No duplicate `skillsync`/`appops` logic, shell call, direct write or second confirmation policy. Guide's install action calls an explicit `InstallModel` reset (`ResetJourney`) and starts at client selection. `routeScroll` must be route-count-agnostic before Guide is registered. `g` keeps the online-README meaning; Guide may expose it explicitly.

**AC-04:** Guide routes to existing install/restore/activity/onboarding; writes require existing preview/confirmation; completed install cannot be resumed; back/Esc returns to Guide; five-route state is safe; plain/ASCII/narrow layouts bounded; `5` opens Guide and `g` opens the online README.

### FR-05 — Help, command reference and in-TUI hints

Update `task`, `run`, `review` help, TUI Activity/Guide/catalog/library/MCP references and every copyable hint to canonical names and the migration table. `task help` explains `task brief`, run binding, registration/recovery, force/refusal and observation status. Unknown flags exit `2`; `--help` exits `0`. One CLI program-name constant plus source-string audits cover core and TUI non-test files.

**AC-05:** `asdx-core task help`, `task brief --help`, `run help`, `review help`, TUI references and hints name canonical/legacy behavior correctly; audits find no active CLI example through `asdx`; old CLI output/exit tests update atomically.

### FR-06 — Active documentation migration

Update README, getting-started, installer reference, execution runs, guide flows, TUI architecture, orchestration guide, MCP tools, fleet-manager TUI, OpenCode pack README/routing, the active orchestration skill/references, AGENTS release text and in-TUI hints. Provide one migration table for canonical/legacy/state/environment names. Leave historical specs, handovers, release reports, dated research and accepted vnext test records unchanged.

**AC-06:** Active docs and hints contain no instruction to run a CLI command through `asdx` or to treat `agentic-sdd` as the TUI; one migration table; links resolve.

### FR-07 — Packaging, source builds and gates

Use one paired archive with four names and unique universal outputs. Update GoReleaser, cask, Taskfile pair/PTY/snapshot, container/guide/vnext scripts and source-build instructions. Keep root Go 1.23 and nested TUI module shape; no new root dependency and no new 008 gate package. Keep `harness/checks.json`'s `build-darwin` required contract unchanged; `task build` may output `bin/asdx-core`, while the trusted `build-darwin` outputs remain `bin/agentic-sdd_darwin_*`. The legacy names are dual-built from existing sources with build-time legacy mode, so the 11-package supported-core selection and trusted-CI contract stay unchanged.

**AC-07:** Static tests, `goreleaser check` and `task -t Taskfile.experimental.yml release:snapshot` prove archive/cask membership when the reviewed toolchain is available; otherwise report `not_run` without archive claims. Live release/tap/Homebrew checks remain `not_run` unless separately authorized.

### FR-08 — State, environment and rollback compatibility

Existing `~/.agentic-sdd/` backups/logs/manifests remain readable; `AGENTIC_SDD_*` switches keep working. Install-time unowned-path protection is enforced by Homebrew cask conflict checks and documented archive pre-extract checks, not by a custom binary installer. Runtime wrappers refuse PATH/symlink resolution. A documented rollback can reinstall the previous release and remove only product-owned executables; it leaves `~/.agentic-sdd/` untouched.

**AC-08:** Isolated fixtures prove state/env compatibility, wrapper runtime path refusal, cask/archive conflict-check documentation, wrapper removal/reinstall rollback, and no forced data-directory change.

### FR-09 — Honest completion boundary

No implementation claims acceptance from a green suite. Report actual models, changed names, wrapper behavior, checks run, platform limits and unresolved risks. Human review remains required after each batch.

**AC-09:** Batch evidence names the exact code state, commands/exits, artifacts built locally, migration results and unrun checks.

## 5. Decisions requiring owner confirmation

**Prior owner decision:** O-01–O-09 were confirmed on 2026-11-13 for the earlier `agentic-sdd`/`agentic-sdd-core` draft. The later naming study is a material amendment; it supersedes conflicting O-01/O-02/O-07 and requires the following decisions before combined approval.

| ID | Question | Recommendation |
|---|---|---|
| N-01 | Adopt `asdx` + `asdx-core` as canonical names? | Yes, subject to B0 direct registry/trademark/PATH verification; do not ship `asd`/`asdd`. |
| N-02 | Adopt Option A legacy wrappers? | Yes: `agentic-sdd` → `asdx-core`; `agentic-sdd-tui` → `asdx`; no silent TUI change. |
| N-03 | Deprecation window and removal release? | At least one minor release; confirm the removal release before combined approval. |
| N-04 | Wrapper implementation shape? | Dual-build legacy mode from existing root/TUI sources; no new 008 gate package; `syscall.Exec` on POSIX; exact top-level help/version; stderr deprecation. |
| N-05 | One archive with four names and cask links? | Yes; unique universal names, quarantine all executables, `release:snapshot` proof. |
| N-06 | Preserve state/env names? | Yes: `~/.agentic-sdd` and `AGENTIC_SDD_*` remain unchanged; no forced migration. |

**Owner confirmation (2026-11-13):** N-01–N-06 confirmed as recommended, including `asdx`/`asdx-core`, Option A wrappers, dual-build legacy mode, one four-name archive and state/env preservation. This confirms the design decisions; it is not the combined spec/plan/tasks approval in §6 and does not authorize implementation.

## 6. Approval

**Combined decision:** **Approved, 2026-11-13.** The owner approved the reviewed `spec.md` r2, `plan.md` r2 and `tasks.md` r2, including N-01–N-06.

- **Approver:** The repository owner in this conversation (no additional identity asserted).
- **Approved revisions/scope:** `spec.md` r2, `plan.md` r2 and `tasks.md` r2 as presented; `asdx`/`asdx-core`, Option A wrappers, dual-build legacy mode, one four-name archive, state/env preservation, Guide, help/docs migration.
- **Decision evidence:** the owner said “I accept the 3 docs” on 2026-11-13, after confirming N-01–N-06.
- **Implementation authorization:** The owner said “start implementation parallelly all” on 2026-11-13. B0–B3 were executed under the dependency-safe, disjoint-file fleet schedule in `tasks.md`; this was parallel disjoint work, not simultaneous writers on one file. Implementation is complete; human review remains required.
- **Still not authorized:** live release/tag/tap/cask publication, real-home apply, paid model calls outside the selected implementation fleet, or deployment.
