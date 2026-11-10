# Plan: Feature 013 — Unified TUI Experience and Animated Branding

**Revision:** r3 — review reconciliation, 2026-11-10  
**Input:** [spec.md](spec.md) r3. **Approval:** [spec.md §Approval](spec.md#approval) only; the owner's r3 confirmation is recorded there.

## 1. Architecture and verified baseline

Keep the existing Bubble Tea v2 foundation and `experiments/tui` nested module. Root `go.mod` stays dependency-free at Go 1.23, with no root `go.sum` or `go.work`. The nested module currently declares Go 1.26; that is its selected target, not evidence that all Charm v2 packages require Go 1.26. [Architecture §4](../../docs/tui-architecture.md#4-toolchain-migration-blueprint) distinguishes measured dependency floors from platform choices.

Current source anchors: `experiments/tui/{config,catalog,theme}.go`, `screens/catalog.go`, `cmd/tui-spike/main.go`; root `cmd/agentic-sdd/{main,guide,guide_catalog,logs,updates}.go`; `internal/skillsync/{sync,inventory,restore}.go`. Inspect current Git state before editing: the r3 session finds only the untracked 013 documents/handover, not the earlier 011/012 working changes. CodeGraph could not resolve this checkout's index; use targeted reads/search unless available later.

The following is a **new 013 design**, not a claim about 012's delivered boundary:

```text
existing explicit CLI ──┐
                       ├─ shared stdlib operation adapters ─ skillsync/services
nested TUI host ────────┘          (validation/logging/results)
       │
 root model → screens → typed requests/results (no direct file mutation)
```

013 extends 012's read-only browser restriction only for approved 013 workflows: screens request operations through a controlled adapter; they never copy files, manipulate manifests, or implement backup/rollback. Do not retroactively loosen 012's plan or describe the spike as write-capable. Update `docs/tui-architecture.md` in the relevant implementation batch to identify the new boundary and preserve historical evidence.

### Shared operations, not command duplication

Create `internal/appops` in B2/T04 for pure teaching metadata first; B3/T06 extends that existing package with reusable preview/apply/log/backup/restore/update orchestration. No duplicated registry and no forward dependency on B3. Keep flag parsing, legacy output formatting (subject only to FR-09's additive help exceptions), exit mapping, and CLI tracing at the CLI boundary. Package creation is contingent on D03: explicitly authorize additive coverage in `Taskfile.yml`, `harness/checks.json`, `tests/supported-core-go.sh` (both required-directory and test/vet lists), and audit Lefthook/other maintained selectors and package-count guidance for consistency. Land coverage with the package in T04, then audit it again in T06. Preserve all existing core packages and sanitation; unresolved seven-package policy conflicts stop B2 rather than leave new code untested.

Use concrete request/result types and narrow injectable function seams, not an interface hierarchy. Requests pin resolved source/home, supported client IDs, integration options and backup identity. Results include existing plan data/transcript, actual outcomes, backup/log references, and errors; do not infer rollback success or backup identity from decorative text. Preserve operation semantics and private-log redaction/permissions. Do not use `runEnv` recursively, import `package main`, invoke a shell, or scrape CLI prose as the sole safety authority. Existing `SummarizePlan` remains usable for presentation counts.

Add an optional client-selection value to `skillsync.Options`: nil means historical all clients; a non-nil selection contains a validated nonempty allowlisted ID slice. Explicit empty/unknown values fail at the engine and adapter seams, not only in the UI. Copy/normalize the selection into immutable requests; use it consistently for preview/apply/inventory. Selection narrows only planned changes/entries; the backup manifest `Targets` remains the full canonical `clientTargets(home)` home identity expected by `loadBackup`, with optional profile identity preserved separately. Test a subset install replacing existing files followed by supported restore, verifying selected recovery and untouched unselected destinations. No manifest-format migration, arbitrary destinations or per-skill selection.

Add narrowly scoped result-returning engine variants for sync and restore, retaining current exported functions as wrappers that discard results and preserve exact transcripts/errors. Return actual outcome, backup identity/path, and available partial-failure/rollback information directly from the engine; propagate through appops into existing logging and TUI results. Define no-change/preview/failed-before-backup semantics explicitly. Stop relying on `logs.go`'s `"; backup: "` transcript parser as the authority. Test old/new API output equivalence and structured results for successful replacement, no-op, failure before/after backup and rollback. This is an explicit r3 engine API amendment, not a wholesale installer rewrite.

Before write execution, compare a token covering resolved home/repo, normalized selection/options, selected source identities and all reviewed planned destination states (including absent and unchanged states whose drift could introduce new writes). Restore additionally pins backup ID, manifest digest and payload digests used by the restore plan. Recompute and compare the complete proposed change set; any new/changed write requires fresh preview/consent. Exclude unselected/unplanned client trees so unrelated edits cannot invalidate selection. Changed inputs return `needs preview` with no writes. Reuse existing digest/path validation, not a new manifest. This is pre-execution drift detection, not isolation from hostile concurrent writers; preserve mutation-time checks and state residual race guarantees honestly.

## 2. Application model and screens

- Root model owns route/back stack, focus, session settings, operation identity/busy state, and shared snapshots. Screens own list selection, filter, scroll, and form state. Keep `screens` below the root model without import cycles; shared display data lives in the existing `tui` package or a small neutral file.
- Bubble Tea `tea.Cmd` performs filesystem/network operations outside Update/View and emits typed start/progress/result messages. Attach request IDs/generations; ignore stale read results. One write transaction at a time; no unbounded queues or fictitious percentage progress.
- Prefer existing Bubbles list/input/viewport components. Huh may be introduced **only in the nested module** for embedded selections/default-No confirmations if its pinned v2 integration meets focus/accessibility needs; otherwise use existing components rather than running a competing terminal loop. Resolve current version/docs before adding it; Huh is not currently pinned.
- Keep Explore usable after the shell lands. Categories are focusable controls; Tab cycles focus, arrows operate within a focused region. Disable child list help in favor of the root contextual footer.
- Wide/medium/narrow layout follows FR-02 with height-aware degradation. Measure terminal cells, not byte/rune counts; account for ANSI, emoji variation, combining marks and wide glyphs. Essential information is scrollable; small terminal mode never hides an active default-No decision.
- Reuse curated `teachingRegistry` metadata by moving it behind a shared data seam while preserving existing CLI tests. Inventory remains authoritative for membership/status; metadata enriches purpose/display name/example only. Preserve unknown-source fallback and exact kind+ID identity.
- For agent profiles, add a curated routing table in the same stdlib seam: mode, provider lane, primary model, documented fallbacks and a manual-failover policy note. Drift tests compare it with `opencode-pack/routing.md` and preset frontmatter. Unlisted profiles stay explicitly "not documented"; no provider is inferred from a model string at render time. This is configuration display, not credential, quota, availability or runtime proof.

### Operation-to-screen coverage

| Existing surface | TUI route | Shared behavior / restriction |
|---|---|---|
| `preview`, legacy default preview | Install & Manage → Preview | Same source/destination planning; no writes |
| `apply`, legacy `--apply` | Install & Manage → Confirm install | Shared engine/log orchestration; default No and fresh preview |
| `--with-opencode`, `--opencode-free-first` | Install & Manage → Optional integration | File templates only; preset depends on pack |
| `backups` | Install & Manage → Backups | Existing discovery/validation; deterministic display ordering |
| `restore preview`, `restore apply` | Install & Manage → Restore | Distinct inspect/preview/confirm/run states |
| `logs` | Activity & Help → Activity | Existing latest/selected supported log behavior, bounded viewport |
| `guide` and read-only explorer | Home onboarding / Explore / Help | Skippable guidance; preserve legacy CLI menu |
| `version`, `--version`, `help` | Activity & Help → About / Reference | Existing metadata and accepted commands |
| Guide's update action (not a standalone `update` command) | Activity & Help → Check updates | Existing fixed GitHub endpoint, bounds, no auto-upgrade |
| New automatic version-check trigger | Home / main menu | Once per session, async, opt-out; reuses the same bounded update service |
| Online README / personal attribution | Home / Help / About | Fixed HTTPS links, terminal hyperlinks and explicit keyboard opener with copyable fallback |
| `opencode` / `opencode-harness` | Install & Manage → Integration explanation | Inspect build availability; no executable experimental adapter/session/config/evidence controls |

`cmd/agentic-sdd/opencode.go` is build-tagged `agentic_sdd_experimental`; do not mistake source presence for shipped support. Audit default-build stubs/help before finalizing labels. No orchestrate/runtime command is invented from help strings. All excluded/blocked capabilities stay non-executable and are not selected for validation.

### Transaction state machine

```text
configure → inspecting → preview → confirm(No) → revalidate → running → result
     ↑            │         │          │             │
     └── edit/cancel ────────┘          └─ drift ─→ fresh preview
```

Only affirmative confirmation of the current token enters revalidation. q/Esc cannot leave running writes. Ctrl+C records exit-after-completion; let the existing uncancellable transaction finish/rollback and flush its result/log before terminal teardown. During read tasks, cancellation invalidates result generation and cancels context where supported. Display cancelled vs failed vs incomplete accurately; restarting the TUI never automatically resumes a transaction.

The TUI host owns OS signals explicitly: use Bubble Tea v2 `WithoutSignalHandler()` and register `os/signal.Notify` for SIGINT/SIGTERM and SIGHUP on supported Unix builds before starting the program. Route signals as typed shutdown requests through the same busy-state policy as Ctrl+C; repeated signals do not hard-kill writes. Idle/read-only shutdown cancels work and quits; writes defer shutdown until their final result and log flush. Normal `Run` cleanup plus an idempotent deferred terminal-release path (`ReleaseTerminal` as appropriate to pinned v2 lifecycle) covers handled errors/signals, including host launch failure; stop signal delivery and avoid double restoration. Test launcher-to-companion forwarding and avoid parent default-signal exit abandoning the child. No cleanup promise for SIGKILL/power loss or a wedged uncancellable write. Document the last-resort kill limitation before execution.

## 3. Branding and accessibility design

Build reusable header/navigation/status/footer/preview styles in the existing nested theme/components area. Use terminal background rather than painting a dark canvas; violet focus, cyan secondary, neutral readable text, labelled semantic colors. A compact ASCII mark appears in the header/welcome; a bounded reveal animates only at first welcome, with no sleep, delayed navigation, ongoing idle timer, or simulated work.

The reviewed shell uses the Exa-researched Midnight Lagoon (OKLab-balanced) high-contrast palette: canvas `#0B1C22`, header surface `#123039`, foreground `#FAF5E9`, muted `#B6CACB`, violet `#F39FEC`, cyan `#35DADA`, success `#84D987`, warning `#EAB946`, error `#FBA7A0`, border `#8FA9AD`. Every text role clears WCAG AAA (>=7:1) on its actual background; borders clear 3:1. Deterministic contrast tests guard the tokens. The top bar paints a surface band with two-cell edge padding, a blank padding row at normal heights and a colored divider; it collapses to one compact row on short terminals. The full view sets the terminal background/foreground through Bubble Tea v2's declarative View fields only when color is on. Status meaning stays text-labelled, and plain sessions emit no styles.

Add secondary **by nawodyaishan** attribution in the full shell/welcome/About, collapsing it into About when space is limited. Centralize the exact author identity, author profile URL and README URL as trusted presentation constants rather than duplicating strings or deriving URLs from catalog content. Terminal hyperlink support is best-effort (OSC 8 where appropriate), always paired with visible/copyable URLs and keyboard actions. Plain/ASCII output emits no hyperlink control sequences. An injected platform-opener adapter launches only the fixed HTTPS README/profile URL through an argument vector (`open`/`xdg-open` or the supported platform equivalent), never a shell or arbitrary source-provided URL; tests never launch a real browser. Handle absence/failure without exiting the TUI and explain remote SSH limitations. Opening links is explicitly requested, not a startup side effect.

Opener trust differs from companion trust: use fixed `/usr/bin/open` on macOS; on Linux resolve the user-installed `xdg-open` once with `exec.LookPath` to an absolute path and explicitly treat the user's PATH/program as trusted desktop configuration, not a security boundary. Never use a source-configured executable/URL or a working-directory-relative fallback. Missing opener retains the URL. All opener tests inject resolution/launch.

The root app tracks automatic update state (`not started`, `checking`, `current`, `newer`, `unavailable`, `disabled`) separately from local inventory. On first main-menu/Home entry, dispatch one asynchronous shared update query unless `AGENTIC_SDD_NO_UPDATE_CHECK=1` or the session offline control disables it. Mark the attempt before dispatch; navigation/resize cannot enqueue duplicates. Results use request IDs; disabling cancels/ignores pending responses, and explicit Retry is separate. Reuse existing endpoint/client timeout/body cap/redirect/version policies without changing the explicit CLI guide action. Treat dev/unparseable versions and rate limits as non-fatal unavailable states; do not install anything, persist cache, send inventory, or delay local rendering. Test with injected transport/clock, not live GitHub calls. Until B5 delivers this service-backed behavior, earlier shell batches remain explicitly local-only intermediate increments, not complete AC-07 evidence.

Extract a context-aware checker using `http.NewRequestWithContext`; keep an existing-behavior CLI wrapper with a background context and existing timeout. Disabling checks/shutting down cancels the request context and invalidates its generation; test actual transport cancellation as well as stale-result suppression.

Define motion=`auto|off` and symbols=`unicode|ascii` session choices with discoverable controls/help; reuse existing ASCII/color environment semantics instead of silently changing them. Plain/accessibility mode forces static ASCII; NO_COLOR strips decorative color but does not equate to ASCII detection. Default to static if terminal suitability is unknown. Unicode mode may use a small set such as “Explore,” “Install,” and “Help” icons/emoji with adjacent words; never replace warning/result text with a symbol. Test width and fallback explicitly. No fonts or external image assets required.

At B1 show owner the actual animated/static frames and narrow/light/dark/plain variants. Branding is not accepted merely because automated tests pass. Keep motion disabled on confirmation/execution and stop ticks when offscreen.

## 4. Launch and distribution proposal

Retain the experimental launcher for development; add a named companion binary built from the nested module for explicit `agentic-sdd tui` and the Homebrew interactive zero-argument default. Resolve dispatch in this order: explicit command/flags preserve their existing path; forced-interactive test mode/non-TTY retains preview; otherwise zero arguments plus an explicit Homebrew distribution marker selects the TUI; all other bare invocations preview. Check real stdin and stdout TTYs, not forced terminal flags. Set a narrow build-time distribution value for the actual Homebrew artifact/formula build; generic archives/source builds retain the default CLI policy. Do not guess from `/opt/homebrew`/`/usr/local`, user shell configuration, or a broad wrapper. The formula installs paired binaries, but installation/post-install hooks never open the UI.

Root CLI launches the trusted same-install `agentic-sdd-tui` sibling with an argument vector, never a shell or PATH search. Define the companion's direct public interface as `--help`, `--version`, and the same supported home/repo/pack/preset and presentation options exposed by `agentic-sdd tui`; direct interactive invocation is supported and obeys identical safety/TTY gates. A private non-interactive protocol-info query returns protocol revision and build identity without network/filesystem inspection. The parent verifies protocol plus matching release/commit identity before starting it; this is compatibility checking, not authentication or write authorization. Both binaries receive the same injected build ID (revision plus source-content/dirty digest for local paired builds), never merely `dev`; missing/unverifiable development identity fails with rebuild-pair guidance. Preserve accepted CLI arguments/exit behavior, forward handled signals safely, and never silently fall back. No downloads or credentials/environment enumeration.

Choose prebuilt paired archives, not formula source builds. Define separate build IDs `cli-generic` (default preview), `cli-homebrew` (link-time Homebrew distribution marker), and `tui-companion` (nested module). Each macOS application/build gets its own explicit universal-binary entry, input build IDs, `replace: true` and non-colliding output names. Never merge different applications into one fat binary. Define filtered, uniquely named archives: generic CLI+companion and Homebrew-marked CLI+companion. The formula consumes only the Homebrew archive via explicit archive selection/URL and installs the marked CLI as `agentic-sdd` plus companion as `agentic-sdd-tui`; generic archive install naming is likewise deterministic. No runtime toolchain or formula compilation. Test actual archive contents/names/checksums, both Mach-O slices, pair identities and marker behavior; do not assume GoReleaser defaults or mixed-platform filtering are correct. Use schema fields supported by the pinned GoReleaser version (archive filtering may be `ids` or legacy `builds`); no `allow_different_binary_count` escape hatch.

B6/T15 establishes and locally validates that build/archive/marker contract **before** T14 implements its consumer. Configure an explicit reviewed Go patch version satisfying nested `go 1.26` for the companion release-build stage, with `GOTOOLCHAIN=local`; select/record the concrete patch in T15 before running the build rather than relying on implicit downloads or root `go-version-file`. Keep the existing sanitized core verification stage/toolchain and trusted-image pins separate and unchanged; use staged jobs or reselect the pinned build toolchain after core verification. A missing toolchain/lipo/GoReleaser prerequisite blocks package proof, not a passing skip. A local nonpublishing snapshot validates generated formula and both paired archives before launcher integration; no tags, tap pushes, token checks or live release steps. Update README and Taskfiles with actual commands only after this evidence.

**D01 resolved in r2:** Owner requests Homebrew-default TUI; `guide` remains unchanged. Owner subsequently confirmed r3 at the existing approval record. B1's completion does not authorize later batches or release/push/tag/formula publication. Build artifacts are not proof of an installed Homebrew journey; report unrun platform checks honestly.

## 5. Specialist and tool assignments

Main agent coordinates. Owner authorized B1 free/DeepSeek/Gemini lanes and subsequently B2 free/DeepSeek/MiMo/MiniMax/Gemini Flash lanes. B2 separates additive gate coverage, shared metadata, pure spacing/Home helpers, acceptance tests and catalog UX; main agent integrates root focus/layout. Dependencies still govern integration and batch review stops remain. Available guidance: `ui-ux-designer`, `golang-pro`, `test-driven-development`, `agentic-sdd-verification-review`, and `writing-for-agents` for D03's narrow AGENTS.md count/list amendment, loaded as needed. The completed owner-requested DeepSeek review (`opencode-go/deepseek-v4.1-flash`, child session `ses_e3db3502effe5rkXV1pHs0osbG`) returned Needs changes; r3 reconciles those findings, not a claim of architecture re-review or an Opus verdict.

During the owner B2 review, the free-model quota was exhausted: both free DeepSeek and free GLM lanes returned the shared free-usage limit, and the owner-directed `muse-spark-1.3-contributor` replacement was blocked by the account's paid request-data-training privacy setting. The owner then explicitly authorized paid DeepSeek Go implementation agents for the remaining review work. Two `opencode-go/deepseek-v4.1-flash` lanes delivered routing metadata/display and the high-contrast shell; the coordinator integrated summary/list-density and ASCII escaping. Color selection was researched through Exa rather than guessed.

Use `read`/`glob`/`grep` for targeted discovery (CodeGraph when resolvable), `patch` for changes, `shell` for scoped gates/PTY/build checks. Context7 for pinned Charm/Huh/library facts; direct official versioned sources if retrieved examples are stale. During drafting Context7 returned v1 template APIs, so the [official v2.0.2 migration guide](https://github.com/charmbracelet/bubbletea/blob/v2.0.2/UPGRADE_GUIDE_V2.md) was checked: root `View() tea.View`, `view.AltScreen`, and `tea.KeyPressMsg` are the correct v2 contracts. Do not implement v1 `WithAltScreen`/`EnterAltScreen` examples. External UX references in the owner's brief are design inputs, not proof they were researched in this session; no new skill installation is planned.

## 6. Risks and validation

| Risk | Mitigation / evidence |
|---|---|
| Extracting CLI orchestration changes bytes/log privacy | Characterization tests before extraction; retain exact CLI formatting/exit/trace seams |
| Client selection expands write scope unexpectedly | Allowlisted IDs, default-all compatibility, subset tests including optional profile destinations |
| Stale preview or double-submit writes unintended files | Identity-bound confirmation, pre-write revalidation, one active write and fake-call-count tests |
| Interrupting a write strands state | Exit-after-safe-completion policy, engine rollback evidence, handled-interrupt PTY tests |
| Branded emoji/animation break layout/accessibility | Static ASCII mode, cell-width fixtures, bounded fake-clock animation and human review |
| Companion packaging couples different versions/toolchains | Same-release pairing/handshake, separate build, absence/mismatch tests; no automatic fallback |
| Experimental handlers appear supported | Build-specific inventory and explanation-only routes; preserve repository validation fence |
| Routing display drifts from the documented model policy | Curated table plus drift tests against `routing.md`/preset frontmatter; unlisted profiles marked not documented |
| High-contrast palette regresses or terminal ignores background | AAA/3:1 contrast tests, plain-mode byte checks, standard ANSI fallback; real-terminal visual review still required |
| Automatic check surprises offline users or blocks Home | Visible GitHub notice, one bounded async request/session, opt-out, injected timeout/rate-limit tests; local state renders first |
| Browser link escape sequences or opener abuse | Fixed HTTPS destinations, no shell, no automatic opening, plain fallback and injected opener tests |
| Homebrew default leaks into scripts/generic binaries | Explicit distribution metadata plus zero-argument/real-TTY dispatch matrix; preserve explicit and piped contracts |

Test state transitions and narrow layouts with deterministic fixtures rather than exhaustive full-screen snapshots. All filesystem integration tests use temporary homes; network tests inject transport and never contact GitHub. PTY checks verify terminal cleanup after quit, error, and deferred interrupt. Human checks cover truecolor/reduced color, light/dark, SSH/tmux where available; missing environments are `not_run`, not pass.

Use an explicitly selected Python standard-library PTY harness, proposed `tests/tui/test_terminal_lifecycle.py`, to allocate a PTY, launch the built nested binary, send keys/OS signals, and compare terminal attributes/cleanup and child exit. Run only `python3 -B -m unittest discover -s tests/tui -p test_terminal_lifecycle.py`; do not add Python dependencies, root Go PTY dependencies or assume Alpine includes util-linux `script`. Add a separate opt-in experimental task with explicit Unix/Python/toolchain prerequisites. Unavailable prerequisites yield `not_run`/`blocked` evidence, never pass. Cover idle and delayed-write signal cases, repeated signals and parent-child forwarding; unit tests supply deterministic in-flight writes without touching a real home.

When revising `docs/tui-architecture.md` in T08/T15, explicitly reconcile its current-support heading, read-only/mutation table, proposed `guide` wiring and Phase 2/3 root migration prerequisites. Preserve them as 012 history while naming 013's controlled operation host, unchanged guide, nested companion distribution and no required root migration. Screens remain mutation-free.

For Go-changing batches run `task test`, `task vet`, `git diff --check` plus the relevant focused tests. Run opt-in `task -t Taskfile.experimental.yml tui:test` separately and confirm tests actually executed: its current old-toolchain skip is not pass evidence. For shared operation/launch changes also run `sh tests/guide-e2e.sh` and `sh tests/docker-e2e.sh` with unchanged scripted choice inputs, subject to supported-core selection. Any aggregate selecting excluded work is `not_run`/`blocked`; stop on gate conflicts, do not weaken selectors. No broad Python discovery or 005/006/007 validation.

## 7. Delivery

Six reviewable batches in [tasks.md](tasks.md) refine the supplied three milestones: shell/branding; categorized browsing; shared operation seam; installation; recovery/activity/onboarding; launch/package readiness. Each ends with verification and human review. Earlier batches retain the read-only launcher until write workflows are explicitly delivered. Final support claims require completed acceptance evidence, not draft status.
