# Plan: Feature 011 — Branded CLI Styling

**Revision:** r4 with review clarifications, 2026-11-09. **Input:** [spec.md](spec.md) approved r6, especially §9. B2–B4 authorized at spec §Approval; existing B1/r3–r5 history remains below.
**Authority:** [Spec §Approval](spec.md#approval) is the sole approval record. Earlier implementation/delegation permissions do not authorize this expanded draft. §7 below owns the proposed r6 architecture/assignments where it supersedes the original one-batch design.

## 1. Approach and one-batch boundary

One batch B1 delivers the complete entry/summary polish, bounded opt-in ASCII reveal, installed version/docs link, explicit update check and manual update instructions. Two independent Go leaf components and documentation can be prepared in parallel; shared command wiring and final verification follow dependencies, not forced parallelism. The existing stdlib-only command package and 010 seams make this bounded enough for one integrated review. No new package, dependency, Go floor, installer behavior, gate or release change.

Sources: spec §§1, 4–7; completed [010 plan](../010-guided-cli-ux/plan.md); `cmd/agentic-sdd/{env,guide,main,logs}.go`, their tests, `internal/version/version.go`, `docs/guide-flows.md`, and `tests/guide-e2e-in-container.sh`. There is no top-level SRS/roadmap; repository conventions and the linked vision are the product sources. Do not reopen 005/006/007 or completed 010 batches.

## 2. Components and contracts

| Component | Planned responsibility |
|---|---|
| New `cmd/agentic-sdd/branding.go`, `branding_test.go` | Semantic accent/heading helpers, compact ASCII intro, plain/narrow fallback, one-second name-line sweep with injected waits, small prose/layout helpers |
| New `cmd/agentic-sdd/updates.go`, `updates_test.go` | Stable-version parser/comparator, bounded HTTP client, validated result classification and plain manual-update text |
| Existing `env.go`, `env_test.go` | Resolve exact `AGENTIC_SDD_ANIMATE=1` into an additive bool; retain existing color/TTY rules |
| Existing `guide.go`; new `guide_branding_test.go`, `guide_updates_test.go` | Intro once before the menu; terminal-only menu `5`; narrow guide layouts; dependency injection; unchanged legacy/plain branch |
| Existing `main.go`, `logs.go`; new `branding_summary_test.go` | Shared styles and narrow stacked counts in existing terminal-only summaries; preserve raw transcripts, next-command strings and file logs |
| `docs/installer-reference.md`, `docs/guide-flows.md` | Opt-in animation, docs/version entry, terminal-only update action, no self-updater, IDs and network/fallback limits |
| `tests/guide-e2e-in-container.sh` | Add targeted static/animated/narrow/no-color guide assertions; preserve all existing network-less journeys |

Freeze these small internal interfaces before parallel writers start (names may change only by coordinator agreement):

- Branding entry accepts `io.Writer`, `*Env`, installed-version string, explicit animate bool and `func(time.Duration)` wait. It does not read input, environment, build globals or network. Static/suppressed/nil-wait rendering preserves its bytes; animated frames use name-line carriage returns and settle to the same displayed text.
- Shared presentation helpers accept the existing `*Env` and plain content. Width decisions happen before applying SGR; wrap prose by Unicode code points rather than UTF-8 bytes, preserve indivisible paths/commands, and let the terminal soft-wrap wide glyphs. No general display-width engine.
- Update leaf accepts an injected `*http.Client` and installed-version string, returning a small typed result (`newer`, `equal`, `ahead`, `unavailable`, `failed`) with validated version text only. It never writes output, logs, files or invokes shell tools. A separate plain renderer uses fixed URLs and fixed script text.
- Keep `runGuide`'s signature. A small internal wrapper/runtime value carries installed version, wait function and update-check function into the guide for tests; production defaults use `version.Version`, `time.Sleep` and the bounded client. No mutable package-global test hooks, no shell-based detection. Constructing defaults must not contact the network.
- New test files may use existing fixtures without changing their meaning. Leaf files must compile without shared guide wiring; cross-component tests are added after serial integration.

## 3. Behavior decisions

### Current r5 UX correction (supersedes affected menu copy above/below)

The owner authorized adding and implementing the UI-assessment corrections; spec §Approval owns that permission. Main coordinator handles this small follow-up directly with Go/TDD guidance, preserving other uncommitted 011 work. No new delegation or architecture phase.

Keep static branding and the approved optional sweep. Add a development/non-release explanation beside the raw version (using the existing stable-version parser). The once-only intro gains the precise skill/profile confirmation notice and No-default reminder. Remove the broad no-write promise from terminal menus and About; do not alter legacy plain helpers.

Every terminal menu gets a blank line, at most 24 ASCII dashes and a `Main menu` heading. Change only terminal labels to spec r5's preview-first/logs/restore/About/network wording; keep option numbers, Enter default, prompt IDs and input handling. About adds first-use guidance, restore impact and CLI-versus-skills explanation. Display the same restore-impact warning before delegating to `runRestore`, only on the terminal guide path.

Before option 5's checker call, print the current action and five-second limit. Do not change HTTP execution, timeout, transport, response validation or states. `updateUnavailable` means valid metadata was fetched but the installed version cannot be compared: say so, and revalidate the latest version before printing it. Failures state that no release information was confirmed and suggest retry/connection checking without guessing a specific HTTP cause. Existing successful comparison and manual command text remains unchanged.

Test first with `guide_ux_test.go`: script `5\n4\nq\n`, check one notice and three menu boundaries; check latest metadata/build explanation, no writes, and restore warning before confirmation. Change only replaced terminal-copy assertions in existing tests; preserve exact legacy-output tests. Run the supported-core gates and both offline container suites, plus new focused tests. Prior animation snapshots/review are not evidence for this amended tree; human review stays pending.

### Presentation and animation

Use cyan/basic ANSI SGR for the product accent, bold magenta for the transient highlight, and ordinary foreground for essential detail. Static is default. Resolve animate once, then additionally require interactive stdin, terminal stdout, enabled color, non-dumb terminal and width ≥60. For the owner-approved r2 sweep, print the top border, render ten full-width ASCII name frames with a three-character highlight advancing left to right, wait 100 ms after each frame, then redraw that same line in static cyan and finish the border/details. Use carriage returns only between these equal-width name frames; no cursor positioning/hiding, erasure, signal handler, background task or prompt-time frame. Reset SGR before waits. A nil wait produces static bytes with no redraws. The initial two-wait/200 ms append-only design is superseded, not silently retained as current behavior.

For terminal stdout, print the intro once outside the menu loop; keep the menu's defaults and cancel behavior. For non-terminal stdout, keep the current print helpers/transcript unchanged, including when stdin is interactive. NO_COLOR/dumb terminal stdout still gets static branding and menu `5`, without ANSI or motion. Narrow mode omits art, wraps prose and stacks labeled counts. Do not wrap or style the update script, paths or copyable commands themselves. Existing test strings describing core outcomes/counts must still pass; add narrow-mode expectations rather than weakening existing assertions.

### Update check and manual instructions

Only accepted terminal menu choice `5` calls the checker once. Use a five-second `http.Client.Timeout`, explicit redirect rejection, fixed HTTPS endpoint, fixed non-sensitive headers and a body reader bounded to 64 KiB + one byte to detect overflow. Reject oversized or invalid JSON, missing/invalid tags, drafts and prereleases; close response bodies. Treat all request/parse/status errors as generic failure, never raw response output. Do not introduce endpoint override flags or environment variables. Use a fake RoundTripper for tests, not public networking.

Parse optional lowercase `v` plus three decimal version components; reject leading-zero multi-digit components, suffixes, whitespace and controls. Compare normalized decimal components by digit count and lexical order, avoiding machine-integer overflow. A `dev`/custom/prerelease current build still permits the explicit request, but the displayed result is `unavailable` if the installed version cannot be compared; no upgrade claim or script follows. Failed transport/body validation takes precedence over comparison. Display only validated current/latest text, never upstream body/assets/URLs.

On `newer`, show exactly spec FR-08's Homebrew-only command block, fixed release link, manual-install alternative, tap-lag/dependency caveat, and separate skills-preview guidance. Equal/ahead/unavailable/failed return to the same menu without writing or failing the guide. No browser launch, clipboard/OSC, authentication, binary download, automatic retry or command execution.

Keep existing trace IDs/fields; choosing `5` uses the existing menu prompt answer and adds `screen.shown` with screen ID `updates.result` for the result. Document the additive screen and menu choice, with no response/version/URL/error fields or `op.result` event. No schema bump; old journeys/events remain unchanged. Revise the docs' statement that *every* guide action has a flag-driven equivalent: installer actions retain their command equivalents, while checking updates is a terminal-guide convenience and the fixed releases URL is its manual alternative.

## 4. Parallel ownership and integration

All agents get the same spec/plan/tasks, bounded assignment, allowed paths, acceptance IDs, prohibition on live actions and required evidence format. No worker changes approval/tasks/handover state; the main coordinator owns those records. No nested delegation or unsolicited commit/push. No worker may stage, revert or overwrite another's work.

| Phase | Work and sole writer | Ownership / dependency |
|---|---|---|
| Contract/baseline | Main coordinator, T01 | Freeze interfaces above and baseline output/check state before any writer starts |
| Parallel lane A | Free implementation agent, T02 | Only `branding.go`, `branding_test.go`; no edits to `Env` or existing print functions |
| Parallel lane B | DeepSeek implementation agent, T03 | Only `updates.go`, `updates_test.go`; no menu or environment edits |
| Parallel lane C | Gemini Flash implementation agent, T04 | Only the two docs above and `tests/guide-e2e-in-container.sh`; no production Go or ledger edits |
| Join/integration | DeepSeek implementation agent, T05 | After A/B/C finish: owns all shared Go wiring and new integration tests; reconciles leaf contracts with consent, not concurrent edits |
| Parallel verification | Free validation worker T06; separate Gemini read-only reviewer T07 | After integration freezes: free worker runs gates; Gemini reviews production Go and its tests. Gemini did not author these code/test files. Main coordinator independently inspects Gemini-authored docs/shell assertions |
| Human checkpoint | Main coordinator, T08 | Reconcile results, arrange AC-07/manual URL check, persist evidence, stop for human review |

Workers may run their focused tests concurrently, but aggregate gates run only after all writers stop; record the exact tree/diff fingerprint and commands. Shared working-tree execution is safe only with these disjoint ownership sets; if an overlap emerges, pause that writer and serialize the edit. A worker cannot claim the whole package green while unrelated parallel work is incomplete. During read-only verification, no worker edits any file. Fixes return to the responsible author, then repeat affected checks and review on the changed snapshot.

## 5. Agents, specialist guidance and tools

Configured primary assignments verified from the active agent definitions on 2026-11-08; service availability/credentials have **not** been tested by drafting.

| Assignment | Agent / configured model | Reason |
|---|---|---|
| T02 branding leaf | `sdd-implement-free-1` / `opencode/deepseek-v4.1-flash-free` | Zero-cost bounded rendering/test work; explicitly requested free lane |
| T03 checker and T05 integration (serial assignments) | `sdd-implement-2` / `opencode-go/deepseek-v4.1-flash#max` | Explicitly requested DeepSeek; bounded network parsing/security and shared wiring need stronger reasoning; may incur Go charges |
| T04 docs/container assertions | `sdd-implement-9r-1` / `9router/ag/gemini-3.8-flash-high` | Explicitly requested Gemini Flash, disjoint routine lane |
| T06 gate execution, no edits | `sdd-implement-free-2` / `opencode/deepseek-v4.1-flash-free` | Explicit bounded validation-only assignment; deterministic results, not an independent sole review verdict |
| T07 independent code review, no edits | `general` with explicit `9router/ag/gemini-3.8-flash-high` model | User requested Gemini Flash; model ID verified in the model catalog. Fresh review session, no code/test authorship; load `agentic-sdd-verification-review` and follow its read-only contract |
| T01/T08 coordination and docs/shell review | Main agent | Integrates evidence and owns the human checkpoint; never treats an agent verdict as human approval |

The generic Gemini reviewer is deliberately not an implementation-agent role or a Claude/Sonnet review assignment. It must not implement fixes. Use a fresh session distinct from the docs worker; coordinator checks the docs worker's own artifacts rather than asking it to approve them. This is a scoped Gemini technical review, not an Opus verdict or a model-runtime certification.

Specialist skills available in this client: `golang-pro` for Go authors and code review; `test-driven-development` before implementation; `agentic-sdd-verification-review` for T07/coordinator verification. Load the applicable guidance at execution, not merely the role label. No additional domain specialist for Markdown/shell; repository conventions and existing container script suffice.

Use configured primaries first. Standing host fallback policy applies to the named SDD agents, preserving provider/free-cost limits and inspecting partial edits before retry. The generic Gemini reviewer has no automatic fallback chain: if unavailable, report blocked and request a different reviewer rather than inventing provider routing. Do not substitute Opus/Sonnet, route Go through 9Router, spend on paid models for a free lane, or install tools. Report actual models and fallback reasons; no model was launched during drafting.

Tools: targeted Read/Grep/Glob (CodeGraph failed to resolve this repo's index earlier); shell for isolated tests/gates; Docker only for the existing local-image network-less journeys; Context7/Exa only for newly unresolved facts (spec §7 evidence already covers the chosen APIs). No browser automation is necessary: the human URL-following check is in their terminal/browser. No actual public update request is needed for verification.

## 6. Risks and verification

| Risk | Control and evidence |
|---|---|
| Network action leaks into startup or plain mode | Injected checker spy: zero calls for all other choices/non-TTY; one call per explicit `5`; no public API in tests |
| HTTP/JSON resource or terminal-injection issue | Time/body/redirect limits; validate numeric tags before displaying; generic failure text; malicious/oversized payload tests |
| Styling corrupts command copyability or accessibility | Unstyled command lines, complete URLs/paths, no OSC/cursor controls, ASCII/default text and 40/60/80-column tests |
| Parallel writers break shared files/evidence | Disjoint ownership, serial integration, writer-free review snapshot, rerun after changes |
| Free-service quota or 9Router unavailable | Report blocked; only permitted host fallback, never silently change cost/provider or weaken verification |
| Scope becomes a self-updater/TUI | Stop and revise documents; no clipboard, binary replacement, shell execution or new framework |

Required automated checks: task-defined supported-core `task test`, `task vet`, `git diff --check`, `sh tests/guide-e2e.sh`, `sh tests/docker-e2e.sh`, plus focused new tests. All tests use isolated homes. Container checks stay `--network none`; update success/error fixtures are in-process, not an online container requirement. Assert the container's explicit update action fails gracefully offline without writes, in addition to existing journeys. No skips, tolerances, aggregate experimental discovery or gate changes.

Acceptance map and continuation live in tasks.md. AC-07 and URL-following are human checks, recorded honestly as `not_run` until performed. Review may be ready for human inspection while these remain outstanding; do not mark the batch accepted or all acceptance criteria passed. Automated failure or a blocking review finding prevents readiness.

## 7. Proposed r6 architecture — bounded presentation and learning layers

### 7.1 Baseline and component boundaries

Preserve the uncommitted B1/r5 tree and its historical approvals/checks. New work uses existing packages only: `cmd/agentic-sdd` for terminal views and navigation; a small additive read-only helper in `internal/skillsync` for source/target inventory. This explicitly revises the original prohibition on skillsync edits. No new dependencies, package lists, manifest formats, CLI commands or public flags; no profile/skill content edits.

| Component | r6 responsibility |
|---|---|
| `cmd/agentic-sdd/guide.go` | Context carried throughout options/review/receipt; terminal-only `b` navigation and menu 6; preserve legacy branch exactly |
| `cmd/agentic-sdd/guide_presentation.go` (proposed) | Actual destinations, friendly labels, stage headings, width-aware counts, receipt/recovery/next steps; small functions over data, no business logic |
| `cmd/agentic-sdd/guide_catalog.go` (proposed) | Curated purpose/when/example/intent records keyed by managed IDs, generic custom-ID fallback, concepts; no live model IDs or dynamic permission evaluator |
| `cmd/agentic-sdd/guide_explorer.go` (proposed) | Plain numbered categories/lists/details over read-only inventory and catalog; no writes, external calls or raw-mode terminal handling |
| `internal/skillsync/inventory.go` (proposed) | Resolve selected source, managed IDs, target paths, backup root and file-comparison statuses using existing helpers; no separate install planner |
| `cmd/agentic-sdd/branding.go` | Legible framed title, static workflow motif, deterministic twenty-frame/two-pass animation with existing opt-in/capability gates |
| `cmd/agentic-sdd/logs.go` | Reuse `operationResult`; pass confirmed transcript/backup identity to receipt; distinguish missing logs from refused/read failures in guide handling |
| `Taskfile.yml` demo tasks only | Print explicit demo/cleanup disclosure; do not modify test/vet/CI/installer commands |
| `docs/guide-flows.md`, `docs/installer-reference.md` | New trace IDs, explorer terms/navigation, counts/paths, copyable recovery examples and motion contract |
| `tests/guide-e2e-in-container.sh` and focused Go tests | Add only the amended journey assertions; original core/network-less checks remain selected |

Names above are seams, not an oversized new framework: no presenter interfaces, page router, theme engine or persistent catalog database.

### 7.2 Context and inventory contract

Carry resolved absolute home and optional absolute repo in a small guide context, plus the chosen `skillsync.Options`. Reuse existing `shellQuoteIfNeeded` and command builder logic rather than writing a second quoting rule. Preview/apply suggestions retain options; guide/restore suggestions accept only their existing flags (never put `--with-opencode` on `guide` or `restore`). Existing commands are printed unstyled as standalone lines. Assert shell token round-trip against fixture values containing spaces, apostrophes and non-ASCII text; executing suggested fixture commands is limited to read-only preview.

Proposed read-only seam: `skillsync.InspectSelection(repo, home string, opts Options) (SelectionInventory, error)`. Resolve through existing `sourceFS`, `openCodePackFS`, `discoverSkills`, `clientTargets`, `openCodeAgentsTarget`, `backupRootFor`, parent/tree validation and source comparison functions. Result carries source-selected skill IDs/profile IDs (with core/base/preset origin), managed target entries (`id`, `kind`, `client`, `absolute path`, `status`), and backup root. Status enum: `matches source`, `differs from source`, `missing`, `unavailable/refused`. It never opens or enumerates unrelated installed profile files, follows rejected links, or creates directories.

This is an inventory/comparison helper, not a planner or mutation API. Keep original `SyncWithOptions`/`Restore` output and behavior unchanged. Core inspection must not require an optional pack. The explorer's profile category explicitly requests a full pack+preset inspection; absent/invalid optional source is displayed as unavailable and offers Back, not an empty-success claim or permission to install. Report target validation failures distinctly; do not turn permission errors into `missing`. Any small extraction of shared validation/enumeration must retain producer behavior and existing tests; never duplicate target path policy in the view.

Review clarification: return core inspection independently; expose a typed `ErrPackMissing` for absent pack source and a distinct non-missing error for invalid/unreadable pack sources when a pack inspection is requested. The explorer catches those category errors and offers Back; installation propagates both. Keep explicit resolved `--home` in guide suggestions even for the default home, as FR-09 already specifies; do not adopt the review's optional omission suggestion.

Use source inventory for unique desired IDs/destinations; use the existing real preview summary for requested change counts. After a successful `runOperation`, compute actual receipt change counts from `operationResult.transcript`, not the preview. Recovery location is inventory's canonical backup root joined to the validated actual `backupID`, displayed only when a snapshot was actually created. The existing `Log:` extraction supplies the real operation log path. No further write, manifest edit or real-home check is needed.

### 7.3 Options, review and error state machine

Terminal action 1: Options → Preview → final question; input `b` loops to Options, retaining home/repo but asking options again, then recomputes inventory/preview. `y` alone (and existing accepted `yes`) enters the shared write path. Blank/No/EOF cancel as before. Keep legacy non-terminal guide code untouched. Use one default-sized `bufio.Reader`; add no post-success read. Only classify `b` as a valid choice at `install.confirm`; emit `prompt.answered(kind=choice, choice=b)` plus `confirm(decision=no)`, then restart. This records no write approval. Source/read errors stop with the original failure rather than rendering a success receipt.

Use a specialized `confirmWithBack` returning an explicit yes/no/back/EOF choice only for this final terminal prompt. Do not change shared `prompter.confirm`'s `(yes, eof)` contract or option/restore/legacy semantics.

For logs in the terminal guide, only a missing directory/empty list is a friendly empty state. Propagate permission/refused path/read errors to stderr with exit 1, matching ordinary log-command failure semantics. Do not infer “nothing changed” on write failure: log and error evidence are shown when known, with conservative retry guidance; do not add a transaction-status API just for a green badge.

At ≥80 columns use the small table; <80 use stacked labeled records. The art gate stays ≥60 separately. Show actual paths on dedicated lines, not table cells. Width estimates can count runes for fixed labels; fall back to records if numeric columns cannot fit. No custom Unicode-width engine. Status colors use existing semantic helpers; paths/commands and essential values never dim or truncate.

### 7.4 Catalog and optional learning

`guide_catalog.go` is a small curated teaching map, not a second source inventory. Each known ID has a category, one-line purpose, “when”, safe example request and declared-intent caveat. Enumerate actual items through inventory; absent known IDs are not shown, unknown custom IDs use a generic description. A set-equality fixture checks that every shipped skill and all base/preset profile IDs have a curated record; any asset addition/removal forces an explicit teaching-map review. Validate intent examples against actual profile templates; do not infer filesystem isolation from a `read-only` label. Avoid a general YAML/frontmatter parser or dependency: exact models/provider health/pricing are omitted; templates and routing docs remain canonical.

Menu 6 enters categories → item list → detail. Lists retain deterministic source ID order grouped by purpose; they are append-only, without OS pager or automatic clearing. Existing prompt reader handles all choices. Details show per-target file status, complete destination and an example **request to a main agent**, never a shell dispatch. For core-only selections, receipt says no profiles selected; the optional explorer may still show available pack profiles, distinctly labeled as available rather than installed. Do not represent all files in a user's OpenCode directory as this install's managed set.

New fixed trace IDs: screens `learn.menu`, `learn.skills`, `learn.profiles`, `learn.concepts`, `learn.detail`; prompts `learn.category`, `learn.item`, `learn.detail`. Use existing `screen.shown`/`prompt.*` events; trace only bounded category/item **numbers** and known navigation tokens, never selected profile IDs, paths, source content or arbitrary input. Existing `install.*` events are reused for review-back. Document these additive enum values under schema 1; no new telemetry sink.

No optional extra prompt after success: print one context-preserving `guide` command and “choose 6”. Keep installation quick for experts; teaching is explicit, local and skippable. Profile intent wording is educational, not an invitation to bypass the user's batch/permission/provider approvals.

### 7.5 Branding choreography

Proposed static frame is ≤52 ASCII columns with unspaced `AGENTIC SDD`; the motif `UNDERSTAND -> PLAN -> BUILD -> VERIFY` follows once. Frame generator returns equal-visible-width name-line frames only: ten forward and ten return highlights, 75 ms per frame. Render using existing SGR palette and injected wait; settle the exact static name before printing remaining text. Total intentional delay = 1.5 seconds, not fake file-operation progress. No whole-screen redraw, rapid on/off flash or extra animation mode/configuration. All current suppressors and nil-wait fallback are tested.

Do not reuse old exact ten-frame tests as evidence for this amendment: replace only superseded animation-count expectations and add new direction/settled-display checks. Human inspection in light/dark terminals, static NO_COLOR and width 40 is still required before an aesthetic/accessibility claim. Do not interpret successful frame timing as proof users find it attractive.

### 7.6 Specialist/tool assignments and execution constraints

For B2–B4, owner-authorized execution uses `sdd-implement-2` (DeepSeek) for inventory and shared guide integration and `sdd-implement-free-1` (Console Free) for isolated branding/catalog leaves. Independent leaves may execute before earlier batch acceptance under the explicit all-batches instruction; this does not grant acceptance or permit concurrent edits of shared guide files. Final docs/scripts integrate after the writers stop. A fresh `sdd-review-verification-1` agent provides the explicitly requested independent verification after integration; its configured primary is used, with the standing fallback policy only on qualifying provider failure. No worker edits approval/ledger records, commits or delegates further. The original B1 lanes in §5 are historical, not fresh authorship claims.

Installed principal specialist: `golang-pro` for Go inventory/rendering/reader integration, with `test-driven-development` before implementation. `agentic-sdd-verification-review` for the batch completion review. No additional specialist is needed for Markdown and the small existing-pattern shell changes. Available conditional tools: local Read/Grep/Glob (no indexed project), shell for sanitized supported-core gates, existing offline Docker images; Exa evidence in spec §9 already covers UX. Context7 only for a new concrete stdlib/library API question, not ritual calls. No provider/model/browser API needs testing here.

### 7.7 Lean tests and risks

Six focused table/journey test groups, rather than a combinatorial UI matrix:

1. Context/commands + inventory: embedded and one custom source; spaces/apostrophes/non-ASCII paths; no mutation; missing/changed/matching/refused target statuses; custom-source counts, no optional-pack dependency for core.
2. Review/receipt: pack/preset combinations and mixed changes; review `b` then decline/confirm; actual result counts; snapshot/no-op behavior; refusing logs isn't an empty state. Assert suggestions run only read-only fixture previews.
3. Explorer: known registry completeness, custom fallback, available vs installed, core-only and missing pack; `6 → list → detail → back → quit`, EOF and invalid input; all original flows preserved.
4. Choreography: pure 20×75 ms frames, direction change, exact settled display, no post-prompt redraw, full suppression table. Optional single PTY timing smoke; not a fleet of visual snapshots.
5. Trace/privacy: exercise review-back and explorer; validate JSON, original fields/types, additive IDs, no IDs/content/paths/free text from learner choices. Assert zero checker/model/shell calls on learner paths.
6. Small container extensions: one context-preserving fixture install/recovery-preview journey, one explorer and one back-before-confirm journey plus revised animation expectations, on existing network-less local image. Never reselect excluded 005/006/007 validation.

Risks: source inventory drift (reuse helpers + asset-set test); stale receipts (actual successful result, not predicted counts); misleading profile security/cost (declared intent + no provider validation claim); teaching overwhelms (optional category view, grouped receipt); animation overload (opt-in/cap ≤1.5 s/suppressors); loss of user WIP (pin current diff/untracked hashes, never reset). Existing trace Lstat race and TTY character-device limitation are unchanged, not silently claimed solved.

Each proposed batch runs `task test`, `task vet`, `git diff --check`, and affected offline container suites; final integration runs both `sh tests/guide-e2e.sh` and `sh tests/docker-e2e.sh`. No gate list edits, skips or dependency/toolchain downloads. Future tests/check results stay `not_run` until actually executed on that batch's tree.
