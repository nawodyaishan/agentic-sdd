# Feature 011 — Branded CLI Styling (Feature 010 follow-up)

**Status:** Approved r6 expansion with review clarifications, 2026-11-09 — B2–B4 implementation authorized; earlier B1 history and outstanding human visual checks preserved.
**Size:** Existing B1 remains awaiting human review; proposed B2–B4 cover the expanded experience within Feature 011. See [plan](plan.md) and [tasks](tasks.md).
**Request:** Style the new Feature 010 CLI, identify UX improvements, recommend stable technology and implementation methodology, include personal branding, colorful ASCII art and animation. Research uses Exa and Context7.
**r2 request:** Show `Agentic SDD` with a one-line vision, browser-openable documentation link and installed version; offer an update check and a copyable update script when an update exists. This adds an explicit read-only network action, not automatic self-update.
**r3 reconciliation:** Added plan/tasks with requested DeepSeek, free and Gemini Flash allocations for parallel execution and verification. No product requirement changed; drafting/allocating agents does not start them or authorize implementation.
**r4 amendment:** Owner reported the 200 ms line reveal was not visibly animated and accepted the proposed one-second color sweep. This replaces only the previous four-chunk/400 ms append-only animation restriction, allowing carriage returns on the logo line before the menu; all suppressors and installer boundaries remain unchanged. Approval evidence is below.

## 1. Outcome and sources

Make `agentic-sdd guide` feel recognizably personal, polished and easy to scan, without turning a safe installer into a full-screen application. Improve existing terminal summaries with the same restrained visual language.

Baseline: [010 spec](../010-guided-cli-ux/spec.md), [010 plan](../010-guided-cli-ux/plan.md), and the current `cmd/agentic-sdd/{env,guide,main,logs}.go`. Feature 010 is completed; its approval does **not** cover this follow-up. This feature adds an explicitly optional branded animation, previously excluded by 010, without reopening its completed batches.

Product compass: [vision §3](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#3-who-agentic-sdd-serves), [§6](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#6-non-negotiable-engineering-principles), and [§7](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#7-product-architecture-and-ownership-boundaries): accessible entry, proportional scope, simple reversible architecture, installer-only responsibilities.

## 2. Styling needed

| Surface | Current friction, observed in source | Required improvement |
|---|---|---|
| Guide entry | Product name is plain prose; no creator identity | Compact colorful ASCII wordmark, existing tagline and personal credit |
| Menu and questions | Similar visual weight throughout | Bold section labels, consistent two-space indentation, explicit textual defaults and cancel guidance |
| Install explanation and preview | Text blocks blend together | Separate explanation, target home, counts and confirmation with short headings and whitespace |
| Completion | Success, backup, log, undo and first prompts compete visually | Result first, then labeled recovery details, then next steps; never dim essential information |
| Existing terminal summaries | Already have minimal bold/green styling | Reuse the same accent and heading conventions; retain raw per-skill transcript and existing counts |
| Optional delight | No animated introduction | Brief, opt-in, colorful ASCII reveal before the first guide menu only |
| Orientation and maintenance | Guide entry lacks version/docs; no update check | Show vision, installed version and documentation URL; add an explicit update check with safe copyable instructions |

These are source-based design observations, not measured usability results. No claim of improved completion time or accessibility certification is made.

## 3. Personal brand and visual direction

Proposed identity, subject to owner approval:

- **Product:** `Agentic SDD`.
- **One-line vision:** `Build software with AI. Stay the engineer.` — reuse README's existing wording; one logical line, soft-wrapped only when needed on narrow terminals.
- **Installed version:** show `Version: <version.Version>` from the existing build metadata; preserve `dev` honestly and point to `agentic-sdd version` for full metadata. Do not hardcode a release number.
- **Documentation:** show `Docs: https://github.com/nawodyaishan/agentic-sdd#readme` as a complete, browser-openable URL.
- **Creator:** `by nawodyaishan` — grounded in the repository owner links and LICENSE; no inferred legal/display name, personal website or social account.
- **Palette:** cyan for the product accent, bold/default foreground for headings, green for successful results, yellow only for meaningful caution. Keep existing stderr diagnostics plain. No background fills, gradients, blinking, or dependence on true color.
- Use the terminal's basic ANSI palette, not fixed RGB brand colors. Cyan is a proposal, not an existing documented personal brand color. Color is supplementary; status and action remain words.
- Use compact ASCII art, not emoji, Nerd Fonts or a large FIGlet banner. Illustrative entry treatment:

```text
+-------------------------+
| A G E N T I C   S D D    |
+-------------------------+
Build software with AI. Stay the engineer.
by nawodyaishan
Version: <installed version>
Docs: https://github.com/nawodyaishan/agentic-sdd#readme

Nothing is written until you confirm.

What would you like to do?
  1) Install or update skills   (default)
  2) Restore an earlier backup
  3) Show the last install log
  4) What is this?
  5) Check for CLI updates (uses GitHub)
  q) Quit
Choose [1]:
```

The wordmark is cyan on capable terminals; the tagline and credit use normal foreground for readability. Render the branded entry once per guide invocation, not on every menu return. Do not add branding to ordinary `preview`, `apply`, help, version, error output, logs or traces. The guide About text may repeat the creator credit without repeating the art or animation.

## 4. Behavior and compatibility requirements

**FR-01 — Safe enhancement boundary.** All new layout/art belongs to terminal stdout only. Preserve byte-identical non-terminal stdout/stderr, existing help/version output, operation logs and existing JSONL trace contracts. Terminal stderr stays plain. With terminal stdin and piped stdout, retain the current unbranded, unanimated guide transcript and original menu; the new update option is terminal-stdout-only. Preserve all commands, flags, counts, ordering of raw plan lines, next-command arguments, exit codes and business behavior. The only menu extension is FR-07's explicit update action; no new top-level command or flag is introduced. Any additive screen/prompt IDs for that action must be documented in `docs/guide-flows.md`; existing IDs/events and schema stay unchanged, with no network payload or URL added to traces.

**FR-02 — Legible static styling.** Guide-owned labels and summaries use consistent spacing and headings; menu numbers, `[1]`, `[y/N]`, target home, backup/log information and undo remain readable as plain text. Do not style user input. Preserve whole paths and copy-pasteable commands/prompts; no truncation or insertion of decorative characters into command strings. Keep existing prompt IDs, defaults, input consumption, EOF and Ctrl+C behavior unchanged, except accepting the new menu choice `5` on terminal stdout. Cancellation guidance states that `q` quits the menu, EOF/Ctrl+C cancel a pending prompt, and write confirmation defaults to No; do not imply `q` is a global command at every prompt.

**FR-03 — Color and narrow terminals.** Reuse 010's `Env` policy: no ANSI at all with non-empty `NO_COLOR`, `TERM=dumb`, or non-terminal stdout. This is intentionally stricter than NO_COLOR's color-only convention. Static ASCII branding remains understandable without color. At widths below 60 columns, use a compact text title and credit instead of the art, and stacked labeled counts instead of alignment-dependent rows. Wrap guide-owned prose to at most `min(72, Env.Width)` where words fit; leave long indivisible values intact for terminal soft wrapping. Essential paths/commands must never be shortened. No automatic dark/light palette queries or new terminal probing.

**FR-04 — Optional colorful ASCII animation.** Static presentation is the default. Only exact `AGENTIC_SDD_ANIMATE=1` requests animation; every other value is off. It runs only for `guide`, with interactive stdin, terminal stdout, enabled color, non-dumb terminal and width at least 60. Existing forced-interactive test semantics may exercise it; they never answer confirmation. `NO_COLOR`, `TERM=dumb`, narrow width or redirected stdout always wins over the opt-in.

The animation performs one visible left-to-right sweep across the name: ten 100 ms frames (one second intentional delay), with a bold magenta highlight over the cyan wordmark. Only the fixed-width ASCII name line may be redrawn with carriage returns; borders, details and all subsequent output remain append-only. Every frame has the full static line's width, resets SGR before waiting, and uses no cursor-positioning/hiding, erasure, screen clear, alternate screen or background goroutine. After the sweep, print the static cyan logo, finish the intro, then show the menu. The settled display matches static branding, although animated raw bytes necessarily differ. No replay on menu return, animation during operations or prompts, looping or whole-screen flashing. Nil wait injection falls back to static rendering. Ctrl+C retains default process behavior; no terminal mode or cursor visibility was changed to restore. All capability/NO_COLOR gates above still apply.

**FR-05 — Safety unchanged.** Preview stays the default. Only explicit existing confirmation authorizes writes. Preserve backup-before-replace, no-op, rejection of symlinks/special files, rollback, log contents and trace privacy. No dependency, Go-floor, package-selection, CI-gate or release changes. Tests and walkthroughs use disposable homes only; this spec authorizes no real-home action.

**FR-06 — Documentation link and version.** The terminal guide entry prints the product, one-line vision, creator, installed version and full documentation URL once. The URL must be selectable and clickable in terminals with URL auto-linking; following it opens the user's browser through the terminal's normal interaction. Do not launch a browser automatically, execute `open`/`xdg-open`, or add OSC 8 controls. Terminals without auto-linking retain a complete copyable URL. The existing `agentic-sdd version` command and its output remain unchanged. Neither printing the version nor showing docs performs a network request.

**FR-07 — Explicit, bounded update check.** On terminal stdout, menu choice `5) Check for CLI updates (uses GitHub)` performs one read-only request to `https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest`, then returns to the menu. Choosing this disclosed action is the network opt-in; startup, About, version, preview, apply and all other existing paths never check automatically. No background polling, telemetry, credentials, token lookup, disk cache or binary download. Send no home/repository paths or input; use a fixed product User-Agent and GitHub JSON Accept header. Enforce a five-second total request timeout, reject redirects, limit response bodies to 64 KiB and close them. Ignore release body/assets; validate fields used for display and never print raw server errors or control characters.

Compare only valid stable `major.minor.patch` versions with an optional leading `v`, numerically rather than lexically. For `dev`, prerelease, custom or malformed installed versions, say comparison is unavailable and provide the fixed releases URL; do not assert that a newer release exists. Validate the upstream tag in the same way and reject drafts/prereleases. Report `Update available: <current> -> <latest>`, `You are up to date.`, or `Installed version is newer than the latest published release.` as applicable. Offline, timeout, rate-limit, 404/no release, non-200, oversized or malformed responses yield `Could not check for updates.` with the releases URL, then return to the menu without changing installer state or causing a failing guide exit. Returning to the menu does not automatically retry.

**FR-08 — Copyable update instructions, never self-execution.** When FR-07 confirms a newer release, show `https://github.com/nawodyaishan/agentic-sdd/releases/latest` and label the following as **For Homebrew installations — copy and run manually**:

```sh
brew update && brew upgrade nawodyaishan/tap/agentic-sdd
agentic-sdd version
```

Render command text on separate unstyled lines, without shell prompt prefixes, art or line-number characters; do not interpolate any server-supplied value. Warn that Homebrew refreshes package metadata, may update dependencies, and its formula can lag a GitHub release. Never claim success until the user actually runs and verifies the commands. For source/manual installations, point to the README/release instructions; do not assume Homebrew ownership, invent an installation path, or supply a destructive universal updater. Explain that upgrading the CLI binary does **not** update installed skill files: run `agentic-sdd preview` afterwards, and apply only after separately reviewing the preview.

A standard terminal has no portable clipboard button. The one-batch deliverable is this clearly labeled copyable script block, not a fake `[Copy]` control. Do not write to the clipboard, emit OSC 52, download-and-execute scripts, run a shell, invoke Homebrew, or replace the running binary. A literal mouse-click copy button/browser UI requires a separate scope decision; the request does not authorize those side effects.

## 5. Recommended stable technology and methodology

This section summarizes delivery guidance; [plan](plan.md) owns design/agent assignments and [tasks](tasks.md) owns the proposed single-batch ledger. None authorizes implementation before combined approval.

| Option | Assessment for this small follow-up |
|---|---|
| **Existing Go standard library and `Env` helpers — recommended** | Repo already uses `fmt`, `io.Writer`, `strings`, ANSI SGR helpers and Go 1.23. Use small semantic render helpers, a bounded synchronous `time`-based reveal, and `net/http`/`encoding/json` for the explicit update check with injected transport for tests. No module downloads, `go.sum`, event loop or toolchain migration. Best stability/maintenance fit here, not a universal library ranking. |
| **Lip Gloss — reserve for a later richer layout need** | A credible standalone styling library; Bubble Tea is not required. Context7 documents a v1/v2 rendering distinction: v2 `Render` emits full-fidelity ANSI and writer functions handle downsampling. It is not a drop-in guarantee of this CLI's existing injected-writer and plain-output contracts. No dependency adoption or current minimum-Go-version claim is made here. |
| **Bubble Tea / Huh** | Unnecessary event-loop/form surface for an ASCII intro and numbered prompts. Do not rewrite Feature 010's interaction model. |

**One-batch feasibility envelope:** presentation helpers, a small bounded read-only update checker and call sites in existing `cmd/agentic-sdd`, focused tests in the same package, small documentation updates to `docs/installer-reference.md` and `docs/guide-flows.md`, and only necessary extensions to the existing guide container checks. No changes to `internal/skillsync` or portable skill assets. The plan bounds the r2 network addition to a leaf checker plus one menu action, with parallel disjoint work followed by serial integration. If it needs a new UI package, dependencies, self-updater, browser/clipboard integration or gate edits, stop and re-scope rather than stretching this batch.

**Implementation methodology:** test first against current output → introduce a few shared semantic helpers in the existing command package → apply branding/layout only at presentation boundaries → add bounded opt-in reveal using an injected wait function → add a pure stable-version comparator and injected HTTP checker with fixture responses → run compatibility and isolated-home journey checks → inspect static/animated views on light and dark terminals → stop for human review. Keep tests deterministic and network-free; assert no transport invocation before explicit choice `5`. Keep frame generation separate from waiting; do not build a theme engine, animation framework or background goroutines. Specialist and agent allocations are in plan §5. The user requested these allocations; actual implementation/delegation awaits combined approval and B1 authorization.

## 6. Acceptance and verification

| ID | Observable acceptance | Focused evidence |
|---|---|---|
| AC-01 | Terminal guide shows ASCII branding, tagline and creator once; returning from About/logs does not replay the intro | In-process output tests with injected `Env` |
| AC-02 | Static/suppressed modes incur zero waits/redraws; opt-in shows ten advancing highlight frames with 100 ms waits, resets before waits, redraws only the fixed-width name line and settles before the menu | Deterministic frame/wait tests plus real-PTY timing smoke; animated output allows SGR and name-line CR only; settled display equals static |
| AC-03 | Animation opt-in is suppressed by each disabled condition; plain status/defaults remain understandable | Table tests: absent/invalid opt-in, NO_COLOR, dumb, non-TTY stdout, noninteractive stdin and width 40 |
| AC-04 | At widths 40/60/80, art fits or is omitted; labels/counts remain readable and full long paths/commands survive | Small table tests, including spaces and non-ASCII characters in paths; no full Unicode-width engine required |
| AC-05 | Existing non-terminal output, error lines/exit codes, logs and trace contracts remain unchanged; no branding/ANSI leaks into them | Existing plain-output, log and trace assertions plus focused new branding checks; do not relax old expectations to hide regressions |
| AC-06 | Install/decline/EOF/repeat-no-op and restore still behave as in 010, using temporary homes; no frame appears after the first prompt | Existing guide tests and container journeys; add only targeted assertions |
| AC-07 | Owner can inspect the static color, static NO_COLOR and opt-in animation variants on a disposable home | Short human walkthrough, light/dark themes and narrow width; record observations, not an automated accessibility claim |
| AC-08 | Guide entry shows one-line vision, injected installed version (including `dev`) and complete docs URL; no browser launch, URL control sequence or network call | Renderer tests; manual URL-following check in a terminal with auto-link support |
| AC-09 | Only terminal menu choice `5` checks once; numeric version comparisons, unavailable comparisons and failures follow FR-07 and return to menu | Fake-transport tests: newer/equal/older, `v0.9.0` vs `v0.10.0`, dev/prerelease/invalid versions, timeout, 403/429, 404, malformed/oversized body and redirect rejection; no public network |
| AC-10 | A confirmed newer release shows the exact copyable Homebrew script, manual-install link and separate skills-preview guidance; no shell/clipboard/download/write effects | Exact unstyled command assertions and no side-effect dependencies; no update script offered as a universal installer |
| AC-11 (r5) | Terminal menu returns preserve results with clear boundaries; exactly one precise skill/profile safety notice; descriptive actions/default/exit text; development explanation and honest fetched-metadata results; restore impact precedes confirmation; legacy plain bytes remain unchanged | `TestGuideUX*`, existing update/branding/legacy tests and offline container suites; all network success cases injected |

Implementation gates: `task test`, `task vet`, `git diff --check`, `sh tests/guide-e2e.sh`, and `sh tests/docker-e2e.sh`. Respect the 008 supported-core fence; no broad experimental discovery, skips or gate weakening. Unavailable container/human checks are `not_run` or `blocked`, never inferred passes. This drafting session runs no runtime acceptance checks.

## 7. Research evidence (2026-11-08)

- **Exa search and fetch:** [CLI Guidelines](https://clig.dev/#output): intentional color, independent stream handling, NO_COLOR/dumb compatibility. [GitHub Primer CLI accessibility](https://primer.style/accessibility/foundations/cli-accessibility/): text carries meaning, readable narrow layouts, persistent results, restrained decoration and plain fallbacks. These support keeping the wordmark compact and animation optional; they do not establish that ASCII animation improves usability.
- **Exa fetch:** [NO_COLOR](https://no-color.org/): a non-empty value disables color; the convention itself does not require disabling bold. This repo deliberately retains 010's stronger no-ANSI behavior.
- **Context7:** `/charmbracelet/lipgloss`, README and `UPGRADE_GUIDE_V2.md`: standalone styling, v2 render-versus-writer downsampling behavior. Library adoption would require a separately reviewed dependency/toolchain assessment; do not copy historical 010 version tables as freshly verified facts.
- **Context7:** `/golang/go/go1_23_10`, `api/go1.12.txt`, `api/go1.16.txt` and `doc/godebug.md`: writer primitives and Go 1.23 timer behavior. Local `go.mod` independently confirms the 1.23 floor; current `env.go` already owns ANSI/reset and color gating. No latest-version or benchmark superiority claim is needed for the stdlib recommendation.
- **Brand sources:** [README](../../README.md) tagline and `nawodyaishan` repository links; [LICENSE](../../LICENSE) copyright identity. Palette and ASCII treatment are new proposals.
- **r2 Context7:** `/websites/github_en_rest`, [latest-release endpoint](https://docs.github.com/en/rest/releases/releases#get-the-latest-release): published full release metadata; validate stable tags rather than assuming the endpoint yields the numerically greatest SemVer. `/golang/go/go1_23_10`: HTTP client total timeout support. Timeout, redirect/body bounds and sanitization above are this feature's chosen safety constraints.
- **r2 Exa:** [Homebrew FAQ](https://docs.brew.sh/FAQ) and [manpage](https://docs.brew.sh/Manpage): `brew update` refreshes definitions; upgrading a named formula uses `brew upgrade <formula>` and may affect dependencies. README already documents the fully qualified formula. GitHub release availability does not establish tap synchronization.

## 8. Decisions and next review

- Combined approval accepted creator `by nawodyaishan`, cyan accent, existing README one-line vision and static-by-default/explicit opt-in. The r4 owner-approved amendment replaces the initial append-only reveal with the one-second color sweep.
- Documentation opens through terminal URL clicking, not auto-launch; update checks happen only on disclosed menu choice `5`; the portable copy affordance is an unstyled script block. A literal copy button still requires a revised UI/platform scope, not an implementation assumption.
- Initial B1 was implemented and received passing gates/Gemini review; that evidence predates r4. See [tasks](tasks.md) for amendment verification rather than treating the original snapshot as proof of changed animation.
- Next action: human batch review, including the pending visual walkthrough and actual docs-link following. No human batch acceptance is recorded; initial combined approval does not substitute for that review.

## Approval

**r6 combined approval and execution (2026-11-09):** Following the Gemini Flash provisional review, the owner instructed: “apply suggestion fixes to the docs from the review then start implementation using free and deepseek implement agents parally all batches, after finish dispatch a verification agent also”. This approves spec r6 / plan r4 / tasks r4 with the bounded review clarifications: distinguish missing optional pack from corrupt/unreadable source, use a specialized terminal review-with-back prompt, and retain explicit resolved `--home` in guide-generated suggestions (already required, not an open decision). B2–B4 are authorized across Free and DeepSeek agents; independent leaf work may proceed in parallel, shared-file integration remains single-writer and verification follows integration. This is not human acceptance of delivered batches or permission for real-home operations, public API checks, commit/push/tag or release. Earlier approvals below remain valid for their scope. §9 is the current expanded contract.

**Scoped UX amendment (r5, 2026-11-09):** The owner requested “add proposed fixes to current spec and start implementation of fixes”, then requested Exa research. This authorizes the following bounded terminal-guide corrections in the existing B1, not a new development phase:

This amendment supersedes the affected entry/menu/safety/update-copy requirements and illustrative transcript in §§2–4; the unrelated branding, animation and network safety constraints remain in force.

- Show the safety notice once at entry: `Skill and profile changes require confirmation.` Do not claim logs/traces never write. Keep confirmation defaults No and all installer semantics unchanged.
- Separate each menu with whitespace, a short ASCII rule and `Main menu`; preserve action results in scrollback, without screen clearing. Label choices `Preview and install skills`, `Restore from a backup`, `View latest install or restore log`, `About and getting started`, and `Check for CLI updates (contacts GitHub)`. Enter still selects 1 and starts the preview-first journey, never a write without confirmation.
- Say `Enter selects 1. Type q to exit.` and `EOF exits; Ctrl+C interrupts the process.` Do not suggest interruption reverses completed writes. Show the No-default reminder once at entry, not on every menu return.
- Explain development/non-release builds next to the existing installed version; retain actual version metadata without inventing a release number. Preserve the previously requested logo/animation, credit and full entry docs URL; moving/removing this approved identity treatment is deferred rather than silently undoing it.
- About explains the first agent step, restore impact (replacement and removal of freshly installed items), and that upgrading the CLI binary does not update installed skills. Show the impact warning before entering the terminal guide's existing restore flow as well. No change to ordinary restore output or behavior.
- Update results distinguish successfully fetched metadata from failed checks: for unsupported installed versions explain why comparison is unavailable, show the validated latest release if available, and state that GitHub metadata was received. Keep the existing one-request behavior for `dev`, five-second timeout and other HTTP bounds; do not silently claim a request was skipped. Print a checking message before the bounded request and a recovery step on failure. No raw upstream errors, automatic retries or new network paths.
- Preserve legacy non-terminal guide bytes, commands, trace schema, log contents, animation suppressors, safety controls and all gate selections. New presentation checks cover `5 → 4 → q`, one safety notice, menu boundaries, development-build explanation, latest-release reporting, and restore impact before confirmation. Existing presentation assertions may change only for the labels replaced here; legacy compatibility assertions remain unchanged.

**r5 research:** Exa search (2026-11-09) returned [GitHub Primer CLI accessibility](https://primer.style/accessibility/foundations/cli-accessibility/) (persistent textual results, descriptive labels, explicit defaults, target/impact before confirmation, honest cancellation) and [Command Line Interface Guidelines](https://clig.dev/) (human-oriented errors and recovery, concise scannable output). These inform this amendment; they do not certify accessibility or justify a TUI rewrite.

**Decision:** Approved (2026-11-08), by the repository owner in this conversation, for `spec.md` r3, `plan.md` r1 and `tasks.md` r1 as presented together. Evidence: **“approved, start implementation using parallel lanes”**. B1 (T01–T08) is authorized with the plan's DeepSeek/free/Gemini parallel allocations and verification. This does not authorize real-home apply, running update instructions, public API validation, commit/push, tag or publication. Feature 010's completed approval is unaffected.

**Scoped amendment approval (2026-11-09):** After the owner reported “agentic sdd is not a animation”, the assistant proposed “a clearly visible one-second color sweep across ‘Agentic SDD’, then settling before the menu appears”, retaining NO_COLOR motion suppression. The owner answered **“yes”**. This approves and authorizes that bounded animation replacement (spec r4 / plan r2 / tasks r2 affected portions), not unrelated scope or human acceptance of the resulting implementation. Initial approval/evidence above remains preserved.

## 9. Proposed r6 — confident installation and optional learning

### Outcome and scope precedence

A beginner can identify the exact target, understand what each optional pack changes, review changes, recover using a context-correct command, and learn what the selected skills/profiles do. An experienced developer can skip teaching and complete the existing numbered flow. Stronger branding is an enhancement, not a prerequisite for using the installer.

This section supersedes the affected r3–r5 terminal presentation/animation constraints only **if approved**. The new `6` explorer and `b` review navigation are explicit extensions to FR-01/02. The 1.5-second animation below replaces FR-04's exact ten-frame/one-second contract and AC-02's exact frame/timing values; AC-16 supplies the new values, while AC-02's suppression/reset/settling guarantees remain. FR-11 replaces the terminal table breakpoint with 80 columns (the separate art/animation gate stays 60). No expansion to a general agent runtime, full-screen TUI or new product phase.

### FR-09 — Context-correct targets, commands and recovery

- Show actual absolute destinations under the resolved selected home, including the OpenCode profile destination when selected. Never print `~/…` as the target when `--home` points elsewhere. Use friendly names: Shared agents, Codex, Claude Code, Antigravity, OpenCode profiles; preserve canonical IDs internally and in traces.
- Commands suggested anywhere in the terminal guide carry the selected absolute `--home` and absolute `--repo` when supplied, correctly shell-quoted. Suggested skill preview/apply commands also carry `--with-opencode` / `--opencode-free-first` when selected. Do not execute suggestions or emit styling, bullets or prompt characters inside command lines.
- After success, show actual changed/unchanged counts, recovery snapshot ID **and location**, and log location. A snapshot of a fresh install records removal for recovery; do not imply it contains earlier file content. Print a read-only recovery preview using the snapshot ID and same home/repo. Label it `Preview restoring this snapshot`; never call it an immediate undo. No new snapshot means no invented ID/location/restore instruction.
- Explicit demo tasks (`task run`, `task docker-guide`) disclose before launching the guide that the selected home/container is temporary and recovery/logs disappear on task exit. The CLI itself does not infer temporary status from `/tmp` or a directory name. No new public flag or demo-detection environment variable.

### FR-10 — Informed options and reversible review navigation

- Stage headings: `1. Choose options`, `2. Review changes`, `3. Installation result`. Explain `skill` (reusable guidance loaded by a coding agent) versus `profile` (a client-specific agent configuration, not a running process).
- Before the OpenCode question, explain its additional orchestration skill and specialist profile files, destination, and backup-before-replacement. Before Free-first, explain which routine writer/test profiles it affects and that it changes routing defaults, does not authenticate providers, and guarantees neither free execution nor future free-model availability.
- Keep existing final affirmative/negative semantics and ordinary scripted answer sequences. On the **terminal guide's final installation review only**, extend the prompt to `Install these changes? [y/N/b=change options]` with the target and full change counts immediately above. `b` restarts the options for the same home/repo; it writes no skills/logs/backups, recomputes the preview, and requires a fresh affirmative confirmation. Blank/No/EOF still cancel; `q` remains a menu action, not silently added to all prompts. Existing non-terminal guide input consumption and bytes stay unchanged.
- Selecting a pack is configuration intent, not consent to execute an agent/model. Display errors as errors: a refused/unreadable log path is not `No logs found`. Preserve the originating exit status for safety/read failures; only genuinely missing/empty logs get the friendly empty state. Do not infer rollback success from silence or generic error strings.

### FR-11 — Readable preview and truthful completion

- At widths ≥80, use a small aligned table: Destination / New / Replace / Unchanged. Below 80, use labeled stacked records. Never align by bytes for non-ASCII labels, truncate a path/ID, or use color as the only status indicator. Values exceeding the nominal table width fall back to records rather than losing information.
- Explain repeated installations separately from unique source assets. Baseline example (not a fixed count): 19 unique skills copied to four destinations = 76 skill-directory entries; 18 profiles = 18 profile-file entries; 94 managed entries total. Derive counts for custom `--repo`, core-only, pack-only and Free-first selections. Mixed unchanged/replaced installs distinguish desired inventory from entries actually changed.
- Completion starts `Success:` with an optional checkmark (not `✔ ok`), then sections `Installed / updated`, `Recovery`, `Next steps`. List core skills by purpose group and profile role-group counts; show exact IDs only in the optional explorer to avoid dumping eighteen cards after every installation. Core-only runs say no profiles were selected; no-op runs say `Already up to date` and create no new operation log or snapshot through the guide.
- Use successful operation results for changed counts, not a stale pre-confirmation total. On failure preserve errors/log locations when known and state `Installation failed; inspect the operation log before retrying`; do not display `Success` or assume all changes were reversed. Progress is a truthful static activity line, not a fabricated percentage or cosmetic wait during file operations.
- One actionable starting prompt first: open a coding agent in a project and ask the router to scope work. Separate optional `Plan a feature` and `Fix an existing bug` examples. Do not imply users must fix a fictional nil-pointer bug.

### FR-12 — Optional skills and agent-profile explorer

- Add terminal-only menu `6) Explore skills and agent profiles`. No new top-level command, persistent preference, onboarding quiz or automatic teaching. About links to this option; success prints a context-preserving guide command and tells users to select 6, without adding a mandatory post-install prompt.
- Explorer categories: `1) Core skills`, `2) OpenCode profiles`, `3) Key concepts`, `b) Main menu`, `q) Exit guide`. Category lists accept a numbered item for details and `b` to return; details wait for Enter to return to the list or `q` to exit. Blank at a list returns to the explorer categories; EOF exits safely; invalid input re-prompts without writes. Show instructions with each prompt. No arrow-key/raw-mode/pager dependency.
- Distinguish `Available in selected source` from per-destination file status at the selected home: `matches source`, `differs from source`, `missing`, `unavailable/refused`. Mere file presence is never `enabled`, `running`, `authenticated`, or `verified working`. Unknown/unrelated user profiles are not scanned, counted as managed or displayed. Source inventory is authoritative; profile teaching metadata is explanatory only.
- Core categories: Scope/plan, Implement/debug, Verify/review, Learn/reflect. Profiles grouped as Main communicator, Read-only investigation, Implementation/documentation, Test assistance, Independent/supplementary review. Detail cards show exact ID, plain purpose, when to use, a safe example request, its scope/authority caveat and actual destination(s). Label the declared role of curated profiles as a template intent, **not security enforcement**. Omit exact model IDs, live cost claims and inferred permission guarantees in this version; point to pack routing docs for configuration details.
- Cover **all managed profiles offered by the selected source**, including 14 base and four optional Free-first profile templates in today's embedded assets, without hardcoding these counts. Mark optional preset profiles separately. Unknown custom-source IDs receive a generic `Custom profile — purpose not documented` card, not an invented role. Missing optional pack source yields an honest unavailable category, not a broken core-only install.
- Distinguish an absent optional pack (`missing`) from malformed/unreadable/refused source (`unavailable` with a safe diagnostic). Either category can return Back without inventing installed status; core inventory remains independent. Selection of a missing/invalid pack for installation still fails rather than silently installing less than requested.
- Concepts explain spec/plan/tasks, skill vs profile, main agent vs subagent, preview vs apply, Free-first limits and human approval. Teaching does not create approval, start an agent, contact a model/provider, read private user config, or install anything. Safe example requests remain plain copyable text; no executable dispatch buttons.

### FR-13 — Eye-catching but bounded brand choreography

- Keep `Agentic SDD`, the tagline and creator identity. Replace widely spaced title letters with a legible ASCII name inside a compact frame; a static plain-text product name must remain readable. Add the static motif `UNDERSTAND -> PLAN -> BUILD -> VERIFY` below the logo, not as an animated task-progress indicator.
- Proposed opt-in animation: twenty 75 ms frames (1.5 s planned delay total), two highlight passes over **one fixed-width name line**: forward cyan/magenta scan, then a return highlight that settles to the static cyan name. No full-line on/off flashes, gradients, repeated loops, animation after first prompt, or movement of information. Borders/tagline/motif/credit/version/docs are append-only and printed once. All frames have equal visible width and reset SGR before waiting. Static and animated settled displays match.
- `AGENTIC_SDD_ANIMATE=1` remains the only opt-in; default/no-color/dumb/narrow/non-TTY/noninteractive/nil-wait modes have zero waits/redraws. Existing width≥60 gate remains; art fits within 52 columns. No new screen clears, cursor hiding/positioning, alternate-screen, OSC, sound, network, third-party dependency or background animation thread. Ctrl+C keeps normal process behavior; no special “skip animation” input stealing.
- Eye-catching and accessible are separate claims: deterministic tests prove frames/timing/suppression, while the owner reviews appearance on light/dark/narrow terminals. Reduced-motion users keep static mode; essential instructions/results never depend on motion.

### FR-14 — Evidence, compatibility and privacy

Preserve all existing CLI commands/flags, default preview, install/restore transactions, no-op rules, manifest format, unrelated assets and seven-package gates. Read-only source/inventory helpers may be added to the existing `internal/skillsync` package; this is an explicit expansion of the original no-skillsync-edit boundary, not a new installation algorithm. Source content or role drift must be detected in tests.

New explorer screen/prompt IDs and the review `b` answer are additive to trace schema 1, documented in `docs/guide-flows.md`; no filesystem path, profile content, role text, model identity or user free text is added to traces. Existing journey events retain names/types. Source inventory and learning metadata never become alternate approval, execution or backup ledgers.

### Illustrative experience (not hardcoded content/counts)

```text
2. Review changes
Target home: /selected/home

Destination          New  Replace  Unchanged
Shared agents         19        0          0
Codex                 19        0          0
Claude Code           19        0          0
Antigravity           19        0          0
OpenCode profiles     18        0          0

19 unique skills across four destinations: 76 entries
18 profile files: 18 entries
94 entries will change. Replacements are backed up first.
Install these changes? [y/N/b=change options]:

Success: Installed 94 managed entries.
Recovery
Snapshot: <actual ID and complete location>
Preview restoring this snapshot:
agentic-sdd restore <actual-ID> --home /selected/home

Next steps
Open your coding agent in a project; ask agentic-sdd-router to scope work.
Explore the available profiles: run the command below, then choose 6.
agentic-sdd guide --home /selected/home
```

Example optional card:

```text
sdd-debug-diagnose — investigate a failure
Use when: you need a root-cause hypothesis before changing code.
Declared intent: read-only investigation; not an OS sandbox guarantee.
Ask your main agent: Investigate this failing test with
sdd-debug-diagnose. Return evidence and hypotheses; do not edit files.
File status: matches selected source (not proof it runs).
Destination: /selected/home/.config/opencode/agents/sdd-debug-diagnose.md
```

### Additional acceptance criteria (draft; no new checks have run)

| ID | Acceptance condition | Lean evidence |
|---|---|---|
| AC-12 | Every displayed target and preview/recovery/explorer command preserves actual home, repo and selected options; shell quoting handles spaces/apostrophes; recovery remains preview-only | Fixture homes/repos with spaces; compare commands to parsed selection; execute only read-only suggested previews against fixtures |
| AC-13 | Pack/Free-first effects explained before questions; terminal `b` review safely recomputes choices; decline/EOF create no installer writes; unsafe log failures remain errors | One scripted review-back journey, option/EOF table, symlink/read-error tests; unchanged legacy transcript tests |
| AC-14 | Core/pack/preset/custom and mixed no-op cases have accurate distinct inventory vs change totals, friendly labels, responsive preview and truthful completion/recovery | Small inventory/receipt table tests, including 19×4+18 fixture and one custom-source fixture; assert no snapshot/log on guided no-op |
| AC-15 | Optional explorer covers the source's managed skills/base/preset profiles, safe teaching cards and per-target file status; absent/custom/refused sources never invent runtime capability | Temporary fixtures: present/missing/different/symlinked files; known registry/source ID drift test; scripted menu 6/back/EOF; zero HTTP/model/shell invocation |
| AC-16 | Twenty 75 ms highlight frames, same settled static display, full suppression matrix, no redraw after first prompt; readable name/motif | Pure frame/injected-wait tests; short real-PTY smoke when available; human visual acceptance remains pending |
| AC-17 | New navigation is traceable without content/path leaks, demos disclose cleanup, all original compatibility and offline suites pass | Decode new trace events, command/environment matrices, existing core gates and targeted container journeys; no public network/model tests |

### Research and source ownership for r6

Exa fetched [Primer CLI accessibility](https://primer.style/accessibility/foundations/cli-accessibility/) and [clig.dev](https://clig.dev/) on 2026-11-09: persistent text, meaningful labels, impact before confirmation, responsive column layouts and restrained decoration. These support optional teaching and plain fallbacks; neither source establishes a superior animation or measured learning benefit. The 1.5-second choreography is a product proposal requiring human review, not a research conclusion.

Inventory sources: `skills-files/`, `opencode-pack/{agents,presets/free-first}`, `internal/skillsync/{source,sync,summary,restore}.go`, `opencode-pack/routing.md`. A local inspection confirmed 14 base profile files plus four preset templates; `opencode-pack/README.md` still has historical `12 profiles` wording in an example, so it must not supply hardcoded counts or be treated as runtime evidence. Preserve portable asset contents and profile permissions; the explorer explains them, not rewrites them.

### r6 decisions and execution boundary

Static-by-default + opt-in 1.5-second two-pass animation; menu 6 explorer; explicit `b` only at final terminal review; grouped receipt plus optional details; source-derived counts with a small curated teaching registry and generic custom-ID fallback. These design choices and the review clarifications are covered by the owner's r6 combined approval/all-batches execution instruction in §Approval. Delivered work is not accepted merely because implementation is authorized; final verification and human visual/learning review remain separate.
