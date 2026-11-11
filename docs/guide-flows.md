# Guided CLI journeys and flow tracing

This page is the single owner of the `agentic-sdd guide` journey catalog, the
screen and prompt identifiers, and the `AGENTIC_SDD_TRACE` JSON Lines schema.
Tests (`tests/guide-e2e-in-container.sh`) and agent context link here instead of
restating these contracts. Command behavior, flags, and file semantics live in
the [installer reference](installer-reference.md).

## Scope and limits

- Supported terminals are macOS and Linux terminals. Windows behavior is
  untested and not claimed.
- Terminal detection is stdlib-only: a character device that is not the null
  device counts as a terminal. `/dev/null` is therefore correctly non-terminal,
  while other character devices such as `/dev/zero` remain a known residual
  false positive.
- The guide adds presentation and orchestration only. Every installer action it
  performs (preview, apply, restore, show logs) remains available as a
  flag-driven command, and it writes through the existing `apply`/`restore`
  code paths (staged replace, backup, rollback, symlink and special-file
  rejection). The CLI update check (menu choice 5) is offered as a
  terminal-guide convenience; there is no top-level update command or
  self-updater, and the fixed GitHub releases URL provides the manual
  alternative. Upgrading the CLI binary does not update installed skill files:
  run `agentic-sdd preview` afterwards, and apply only after reviewing the
  preview.

## Visual presentation, branding, and animation

On terminal stdout, `agentic-sdd guide` adds restrained visual styling:

- **Branded introduction:** At startup on terminal stdout, the guide prints:
  - A compact cyan ASCII frame with the legible name `AGENTIC SDD` and a static `UNDERSTAND -> PLAN -> BUILD -> VERIFY` motif.
  - One-line vision: `Build software with AI. Stay the engineer.`
  - Creator credit: `by nawodyaishan`
  - Installed version: `Version: <version>` (e.g. `dev` or release version from build metadata).
  - Documentation link: `Docs: https://github.com/nawodyaishan/agentic-sdd#readme` (complete copyable URL; terminals with URL auto-linking can open this in a browser; no browser process is launched).
  - Development/custom versions are explicitly labeled as non-release builds; the raw installed version stays visible.
  - One safety reminder: `Skill and profile changes require confirmation.` Write confirmation defaults to No. This does not claim operation logs or opt-in traces never write.
  - The intro renders once per guide session. Returning to the menu after inspecting logs, About, or update checks does not replay the intro.
  - When stdout is not a terminal (e.g. piped stdout), the unbranded, unanimated transcript and original 4-choice menu are preserved.
- **Narrow terminals (`COLUMNS < 60`):**
  - The ASCII art is omitted; the title is printed as plain text `Agentic SDD`.
  - Guide-owned prose wraps to `min(72, Env.Width)`.
  - Install-review counts use stacked labeled records below 80 columns; the separate art/animation threshold is 60 columns.
  - Indivisible commands, paths, and update instructions are never truncated.
- **Color gating (`NO_COLOR` / `TERM=dumb`):**
  - Any non-empty `NO_COLOR` or `TERM=dumb` disables all ANSI escape sequences.
  - Status remains stated as words (`ok`, `error`), and static ASCII branding remains legible.
- **Optional animation (`AGENTIC_SDD_ANIMATE=1`):**
  - Static display is default with zero intentional delay.
  - Setting `AGENTIC_SDD_ANIMATE=1` exactly opts in to a two-pass sweep across the ASCII name: twenty highlights on cyan, with 75 ms per frame (1.5 seconds planned delay), before reading input. Ten frames travel forward and ten return; this is branding, not file-operation progress.
  - Animation requires interactive stdin, terminal stdout, enabled color, non-dumb terminal, and terminal width >= 60.
  - Any other value, `NO_COLOR`, `TERM=dumb`, narrow terminal, or non-terminal stdout suppresses animation.
  - Only the fixed-width name line is redrawn with carriage returns. It settles to the static cyan logo before showing details/menu, so the final displayed text is the same as static mode, not the raw output bytes. It never hides/repositions the cursor with escape codes, clears/erases the screen, uses an alternate screen buffer, loops or redraws prompts. Every frame resets styling before waiting.
- **CLI update check and instructions (menu choice 5):**
  - Available on terminal stdout only.
  - Performs one bounded read-only HTTPS GET to `https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest`.
  - 5-second total timeout, 64 KiB response body limit, and redirect rejection.
  - No background polling, telemetry, token lookup, or binary download.
  - Compares stable numeric `major.minor.patch` SemVer. Non-stable, prerelease, or `dev` builds still make the explicitly requested check; after valid metadata is received they explain why comparison is unavailable and show the validated latest published version.
  - Offline, timeout, or error responses output `Could not check for updates.` with the releases URL and return to the menu without changing installer state.
  - If a newer release exists, displays copyable package update commands:
    ```sh
    brew update && brew upgrade nawodyaishan/tap/agentic-sdd
    agentic-sdd version
    ```
    with a release/package-neutral note that package metadata can lag a GitHub release, plus manual installation links to the GitHub releases URL. No clipboard access (OSC 52), shell execution, or automatic upgrades are performed.

## Journeys

### Expanded terminal review and learning

Terminal installation shows numbered Options, Review changes and Installation
result stages. The final review accepts `b` to change options for the same
selected home/repo, recomputes the preview and requires a new affirmative
confirmation. Blank/No/EOF still cancel. This back action is not available at
the optional-pack questions or ordinary restore confirmation.

Destinations are actual paths under the selected home. Suggested preview and
recovery commands preserve that home and any supplied repository; skill-preview
commands preserve selected OpenCode options. Recovery commands preview restoring
the actual snapshot rather than applying it immediately. Unique skills copied
to multiple destinations are distinguished from managed directory/file entries.

Terminal menu choice `6` opens an optional local explorer: Core skills,
OpenCode profiles and Key concepts. Choose an item number for a detail card,
`b` to go back or `q` to exit the guide. At a detail card Enter returns to the
list; blank at a list returns to categories. EOF exits. Nothing in this explorer
installs files, executes agents or calls a model/provider. File status means
matches source, differs from source, missing or unavailable/refused—not running,
authenticated or verified working. Profiles are available in the selected
source; their declared roles are template intent, not an OS sandbox guarantee.
Unknown custom-source IDs have generic descriptions. Unrelated installed
profiles are not scanned.

The existing schema-1 `screen.shown` event adds `learn.menu`, `learn.skills`,
`learn.profiles`, `learn.concepts` and `learn.detail`. Existing `prompt.shown`
and `prompt.answered` use prompt IDs `learn.category`, `learn.item` and
`learn.detail`; choices are bounded numbers or navigation tokens, never profile
IDs, paths, content or user free text. `install.confirm` can additionally record
choice `b` with `confirm` decision `no`, followed by a fresh options/review flow.

Terminal menus are separated by `Main menu` headings and short ASCII rules;
results stay in scrollback. Enter selects the preview-and-install journey,
which still requires confirmation before skill changes. Option 3 shows the
latest install **or restore** log. About explains first use, restore impact,
and the distinction between upgrading the CLI and updating installed skills.
EOF exits the guide; Ctrl+C interrupts the process, not an automatic undo.
The existing non-terminal transcript is unchanged.

The journeys below are the authored flows, not measured usage. `stdin`
shows the literal answers, one per line, for the numbered prompts.

| ID | Journey | Preconditions | `stdin` answers | Expected outcome |
|---|---|---|---|---|
| **J1** | Guided fresh install, no pack | Empty disposable `--home`; a repository or embedded skill set; an interactive terminal | `1`, `n`, `y` — install, no OpenCode pack, confirm | The same installed skill trees as the equivalent `apply`, a backup (and apply log) created as `apply` does; trace records `confirm` with `decision: yes`, `op.result` with `op: apply` and `exit: 0`, and `run.end` with `exit: 0` |
| **J2** | Guided install declined | Empty disposable `--home` | `1`, `n`, `n` — install, no pack, do not confirm | Nothing is written; trace records `confirm` with `decision: no`; exit `0` |
| **J3** | Guided restore of an earlier backup | A home with one prior `apply`, so exactly one backup exists | `2`, `1`, `y` — restore, select backup 1, confirm | The same effect as `restore apply <ID>`; trace records `op.result` with `op: restore`; the restore's own new backup is allowed |
| **J4** | Guide without a terminal | `AGENTIC_SDD_FORCE_INTERACTIVE` unset; `stdin` is `/dev/null` or an empty pipe | none | Exit `2`; nothing written and nothing read; `guide` points to `preview`, `apply`, and `help guide`; `restore` without an ID prints `restore requires a backup id; …` and exits `2` |
| **J5** | Scripted automation unchanged | `AGENTIC_SDD_FORCE_INTERACTIVE` unset; output not a terminal | none | `preview`, `apply`, `backups`, and `restore <ID>` keep byte-identical pre-feature output and exit codes; `preview` ends with `Preview only. Run with apply (or --apply) to install.` and prints no `Next:` summary block |
| **J6** | Color and plain presentation | `AGENTIC_SDD_FORCE_INTERACTIVE=1`, so stdout is a terminal | none | `TERM=xterm-256color` without `NO_COLOR` contains ANSI escapes; `NO_COLOR=1` or `TERM=dumb` contains none; status is still stated as words (`ok`, `error`) |
| **J7** | Guided update check offline | Terminal stdout; network-less environment (`--network none`) | `5`, `q` — check updates, return to menu, quit | Attempts GitHub release check, fails gracefully with `Could not check for updates.` and fixed releases URL; returns to menu; no files written, no backup; trace records `prompt.answered` for menu with choice `5`, `screen.shown` with `screen: "updates.result"`, and subsequent menu prompt; exit `0` |
| **J8** | Optional learning | One-skill fixture source; disposable empty home | `6`, `1`, `1`, Enter, `b`, `b`, `q` | Shows a skill detail card and returns through lists/categories/menu; home unchanged; trace has `learn.detail` but no asset IDs, paths or operation result |
| **J9** | Change options before installing | Disposable empty home | `1`, `n`, `b`, `n`, `n` | Reopens options after first review, then cancels; no installer writes or operation result |
| **J10** | Context-correct install and recovery preview | Fixture home with spaces | `1`, `n`, `y`, then read-only restore preview | Receipt preserves selected home/repo in recovery/reentry commands; previewing its actual snapshot leaves installed files unchanged |

`tests/guide-e2e-in-container.sh` drives J1–J3 with
`AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 AGENTIC_SDD_TRACE=<file>` and
asserts the final files and grep-level trace events. J4 and J5 run with
`AGENTIC_SDD_FORCE_INTERACTIVE` unset. J6 checks the color rules above.
Additional container checks assert static branding (color vs NO_COLOR),
narrow terminals (width 40), the opt-in animated reveal (`AGENTIC_SDD_ANIMATE=1`),
and J7's offline update check without writes.

## Screens and prompts

`guide` emits the following stable identifiers through the trace. Text is
illustrative and may be reworded; identifiers must not change without a
`schema` bump (see below).

Screens (`screen.shown`):

| Screen | When it is shown |
|---|---|
| `menu` | The top-level choice: install, restore, logs, about, updates and learning explorer (on terminal stdout), quit |
| `install.explain` | What will be installed and into which four client directories, including the resolved home |
| `install.preview` | The per-client counts summary from the real plan (nothing written yet) |
| `install.done` | The result, backup and log locations, undo command, and first prompts |
| `about` | The "What is this?" explanation, after which the menu returns |
| `updates.result` | The outcome of the explicit update check (menu choice 5: newer, up to date, ahead, unavailable, or failed) with manual upgrade instructions or fallback releases URL, after which the menu returns |

Prompts (`prompt.shown` / `prompt.answered`):

| Prompt | Meaning |
|---|---|
| `menu` | Top-level choice (`1`, `2`, `3`, `4`, `5`, `6`, or `q`) on terminal stdout; choices `1`–`4` or `q` when stdout is not a terminal |
| `install.with_opencode` | Add the OpenCode orchestration pack (default `No`) |
| `install.free_first` | Prefer Free models first (asked only after the pack is accepted; default `No`) |
| `install.confirm` | Terminal review: `Install these changes? [y/N/b=change options]` (default `No`); legacy non-terminal transcript retains the original target-specific `[y/N]` question |
| `restore.select` | Existing restore backup selection, reused unchanged |
| `restore.confirm` | Existing `Restore these skills? [y/N]` confirmation, reused unchanged |

Install-path order: `menu` → `install.explain` → `install.with_opencode` →
(`install.free_first` when the pack is accepted) → `install.preview` →
`install.confirm` → `install.done` on success; terminal `b` returns to options
and computes a fresh preview. Options `3` (logs), `4` (`about`), `5` (`updates`)
and `6` (explorer, when Back is chosen) return to the menu; options `1` and `2`
exit when their flow completes or cancels. Refused/read-failed log paths are
errors rather than an empty log result.

## Flow trace (`AGENTIC_SDD_TRACE`)

Opt in by setting `AGENTIC_SDD_TRACE=<path>` to a non-empty path. Unset, or set
to the empty string, tracing is off and behavior, output, and files are
identical to a build without this feature.

The trace is JSON Lines (one object per event) written with the standard
library `log/slog` `JSONHandler`. It is a public contract for tests and agents:
additive changes keep `schema: 1`; renaming or removing an event or field bumps
`schema`.

Common fields on every event:

| Field | Type | Meaning |
|---|---|---|
| `schema` | integer | Trace schema version, currently `1` |
| `run` | string | Random identifier for this CLI run |
| `seq` | integer | Monotonic event counter within the run |
| `event` | string | Stable event name |
| `time` | string | Event time (suppressed only in in-process tests) |

Events and their attributes:

| Event | Attributes | Notes |
|---|---|---|
| `run.start` | `command`, `flags`, `version` | `flags` are flag *names* only, never values |
| `env.resolved` | `interactive`, `stdout_tty`, `stderr_tty`, `color`, `ascii`, `width` | The decision `resolveEnv` made for the run |
| `screen.shown` | `screen` | One of the screen identifiers above |
| `prompt.shown` | `prompt`, `default` | Prompt identifier and its default answer |
| `prompt.answered` | `prompt`, `kind`, `choice` | `kind` is `default`, `choice`, `blank`, `invalid`, or `eof`; `choice` only for enumerated prompts |
| `plan.summary` | `install`, `replace`, `up_to_date` | Counts from the real plan |
| `confirm` | `prompt`, `decision` | `decision` is `yes` or `no` |
| `op.result` | `op`, `exit`, `backup_id` | `op` is the bare verb `apply` or `restore`; `backup_id` is present only when a backup was created |
| `run.end` | `exit` | Absent when the process is interrupted before a normal return |

Example line (time omitted as tests do):

```json
{"event":"confirm","schema":1,"run":"9f2c1a7d4b6e0c13","seq":7,"prompt":"install.confirm","decision":"yes"}
```

### Privacy and safety

- The trace records no flag values, no positional arguments, no free-text
  answers, no environment dump, no credentials, and no file contents.
- **No filesystem paths are recorded at all.** A backup is identified only by
  its validated basename id, which is not a path.
- **No network payloads are recorded.** In addition to the existing run/menu events, menu choice 5 records
  `prompt.answered` with `choice: "5"` and `screen.shown` with `screen: "updates.result"`.
  Update-check events record no HTTP response bodies, release tags/versions, or URLs,
  and emit no `op.result` event. The existing `run.start` tool-version metadata remains unchanged.
- The file is opened with mode `0600`, and an existing file is normalized to
  `0600` so it cannot remain looser. It is appended to, so a multi-command
  journey can share one file. Parent directories are never created, and a
  symlink or non-regular target is refused.
- The trace is not the apply log, not a harness receipt, not approval, and not
  evidence of model or client behavior. `~/.agentic-sdd/logs/` and
  `agentic-sdd logs` are unchanged.

### Failure isolation

A trace that cannot be opened or written never changes the command's result or
exit code. One `warning: trace disabled: <reason>` line goes to stderr, without
echoing the user-supplied trace path, and the run continues untraced.

## Test seam: `AGENTIC_SDD_FORCE_INTERACTIVE=1`

Container images have no pty tool, and captured output is not a terminal. For
testing only, `AGENTIC_SDD_FORCE_INTERACTIVE=1` (exactly the string `1`) makes
the run treat stdin, stdout, and stderr as terminals, so terminal-only behavior
— prompts, the grouped preview, `hint:` lines, and `guide` — can be exercised
while output is redirected to a file.

It is a test seam, not a user feature:

- It never answers, skips, or pre-approves a prompt. Every default stays `No`,
  and end-of-input still cancels and writes nothing.
- Color still follows the normal rules: a forced terminal is colored unless
  `NO_COLOR` is non-empty or `TERM=dumb`, which is why the stable journeys set
  `NO_COLOR=1` and J6 omits it to assert styling is present.
- It is not a privilege boundary; it is equivalent to a user supplying answers
  on stdin.
