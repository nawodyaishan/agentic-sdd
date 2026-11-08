# Feature 010 — Guided CLI Experience, Accessible Presentation & Flow Tracing

**Status:** Approved r3 (2026-11-08) — Batches B1–B5 implemented, verified (`sdd-review-verification-2` Ready), and accepted by owner; commit and push authorized
**Canonical path:** `specs/010-guided-cli-ux/spec.md`
**Source assessment:** the read-only UX assessment delivered in this session (2026-11-07/08), accepted by the owner as the approach ("i accept your approach"). The owner then accepted the recommended defaults for Q1–Q5 ("i accept your decisions", 2026-11-08); §9 records them.
**r2 changes from r1 (found while drafting the plan; each is a clarification, not new scope):** (A1) the interactive test seam also treats stdout/stderr as terminals so terminal-only presentation is testable, while color still follows `NO_COLOR`/`TERM`; (A2) FR-02's byte-identity now explicitly exempts the additive help text of FR-04; (A3) the Q1–Q5 decisions are recorded in §9, with Q5 resolved to a parser owned by `skillsync` and pinned by a drift test; (A4) "No backups found." is removed from the stderr hint set because it is stdout output, not an error line; (A5) Ctrl+C is handled by the default process behavior, not a handler.
**r3 changes from r2 (reconciled from `sdd-review-verification-2` and `sdd-architecture-review-9r` findings H-1..L-3, approved by owner 2026-11-08):** (R1) FR-02/FR-04 clarify that the two added `guide` lines in `printUsage` also appear when `printUsage(stderr)` is invoked on top-level usage errors; (R2) FR-06 specifies that `guide` install suppresses the raw per-skill line dump to stdout while preserving the full transcript in the apply log file, and that EOF cancels immediately at every prompt (including `[y/N]` pack questions); (R3) FR-07 specifies empty `AGENTIC_SDD_TRACE=""` is disabled, `op.result.backup_id` is `filepath.Base` validated by `ValidateBackupID`, `op.result.op` is the bare verb (`"apply"`/`"restore"`), existing trace files are `Chmod(0600)`, and open-failure warnings do not echo the raw path; (R4) FR-08 keeps the AC-12 walkthrough template inside `specs/010-guided-cli-ux/tasks.md` (no fifth file in `docs/`).
**Vision links:** [§3 audiences](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#3-who-agentic-sdd-serves) (individual developers, learners, experienced engineers), [§6 principles 1, 4, 11, 12](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#6-non-negotiable-engineering-principles), [§10 decision gate](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#decision-gate-for-new-investments), and the H0 gate "usable beginner-to-professional walkthroughs".

> **Authority notice:** This draft grants no approval, batch authorization or live-system permission. Per `AGENTS.md`, developing or testing installer code never authorizes `apply` against a real user home, publishing, tagging, or changing a test gate. Approval is recorded once, in §11, after human review.

## 1. Outcome

A person at a terminal can run **one command, `agentic-sdd guide`**, and be led through installing, restoring, or inspecting their Agentic-SDD skills in plain language with a previewed, confirmed, reversible result. Everyone else — scripts, CI, existing users, experienced engineers — sees **no change** unless they are at a terminal, and then only richer *presentation* of the same operations.

Separately, every CLI run can **optionally record a structured, privacy-safe flow trace** so user journeys can be replayed and asserted in test containers and read by AI agents as context, without parsing human-oriented output.

**Product principle:** the CLI stays a correct, reversible file installer ([vision §7](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md#7-product-architecture-and-ownership-boundaries)). This feature improves *how it is operated and understood*, not *what it does*.

### Observed problems this addresses (decision-gate item 1)

Verified against the current binary and source on 2026-11-08:

| # | Observation | Evidence |
|---|---|---|
| P1 | Preview prints one bare line per skill per client (dozens of lines for a fresh home), with no summary or grouping, and ends with only `Preview only. Run with apply (or --apply) to install.` | `internal/skillsync/sync.go` `planChanges`; a fresh-home `preview` run |
| P2 | `help <topic>` for `preview`, `apply`, `backups`, `logs`, `restore` all reprint the same global usage block. | `cmd/agentic-sdd/main.go:47-61` |
| P3 | Errors are one `error: <message>` line with no remediation. | `main.go` throughout |
| P4 | Nothing explains what the four client directories are, or what to do after installing. | README / `docs/getting-started.md` only |
| P5 | **TTY detection treats `/dev/null` as a terminal.** `agentic-sdd restore --home H </dev/null` prints the backup picker and exits 0 "Cancelled." — contradicting the documented rule "Without a terminal, an ID is required." Only stdin is checked; stdout/stderr are never checked. | `isTerminal` uses `os.ModeCharDevice` (`main.go:22-28`); Go test run confirmed `/dev/null` reports `ModeCharDevice` true |
| P6 | There is no machine-readable record of an interactive flow. The existing `~/.agentic-sdd/logs/` apply/restore log is a human transcript of writing operations only. | `cmd/agentic-sdd/logs.go` |

P1–P4 are plausible beginner friction, not measured usage; the project has no usage data. A short fresh-user walkthrough (AC-12) is the evidence this feature must produce.

## 2. Scope

### In scope

1. **Environment resolution (one place):** decide, once per run, whether prompts are allowed, whether color/Unicode glyphs are used, and the width. Fixes P5.
2. **Presentation polish (terminal only):** grouped preview with a summary and next-step line; real per-topic help; `hint:` lines on selected errors.
3. **`agentic-sdd guide`:** a numbered-prompt wizard that orchestrates *existing* operations (preview, apply, restore, logs).
4. **Optional flow trace:** opt-in structured events (JSON Lines) for every command, with a stable documented schema.
5. **Simple tests:** a small, fixed set of isolated-home Go tests plus one container journey script (§8).
6. **Docs:** README command table, `docs/installer-reference.md`, and a flow catalog for humans and agents (§7 FR-08).

### Explicitly excluded

- A full-screen TUI/dashboard, alt-screen, mouse, arrow-key menus, spinners or animation, and **any new third-party dependency** (Bubble Tea, Huh, Lip Gloss, `x/term`, cobra, etc.). See §10 for the evaluated trade-offs and conditions that would reopen this.
- Changing what bare `agentic-sdd` does (it stays a preview), or any existing command, flag, exit code, preview/apply semantics, backup/manifest format, or restore behavior — except the single documented non-TTY fix in FR-01.
- A configuration file, new top-level commands other than `guide`, shell completion, telemetry, or any network access.
- The experimental `opencode-harness` helper (`-tags agentic_sdd_experimental`) and Feature 005/006/007 material.
- Windows support claims (the project's releases and CI are macOS/Linux; behavior elsewhere is untested and not promised).
- Real-home execution, release/tag publication, and gate changes (see §9 Q4).

## 3. Actors and journeys

| Actor | Journey | What they see |
|---|---|---|
| **Beginner / undergraduate** (vision audience 2) | `agentic-sdd guide` → choose Install → read plain-language explanation → see grouped preview → confirm → get first prompts to try with their agent | The guided flow (§6 example), numbered prompts only |
| **Solo developer** (audience 1) | `agentic-sdd preview` / `apply` as today, at a terminal | Same operations, grouped output with a summary and the exact next command |
| **Experienced engineer / CI** (audiences 3–5) | Existing commands and flags, piped or scripted | **Byte-identical** output, exit codes, and prompts-never-required behavior |
| **Screen-reader or `NO_COLOR` user** | Any of the above | No color, no redraws, no cursor control; status always as words, not color alone |
| **Test author / AI agent** | Runs a journey in a container with tracing on | A JSONL trace of screens, answers, plan counts, results and exit codes |

## 4. Functional requirements

### FR-01 — Single environment resolution, including the non-TTY fix

- A single function resolves, per run: *interactive* (stdin is a terminal), *stdout terminal*, *stderr terminal*, *color enabled*, *ASCII-only glyphs*, and *width*.
- **Terminal test must not treat `/dev/null` (or any non-terminal character device) as a terminal.** Implementation must remain stdlib-only (see §9 Q1); the plan chooses the mechanism.
- **Color is off** when the target stream is not a terminal, when `NO_COLOR` is set and non-empty, or when `TERM=dumb` ([NO_COLOR](https://no-color.org/), [clig.dev](https://clig.dev/#output)). **ASCII glyphs** are used when `TERM=dumb`. Turning color off removes only color — text, structure, and status words are unchanged.
- **Width** is `COLUMNS` when it is a positive integer, else 80. All fixed guide text lines are ≤ 72 columns so no wrapping logic is required.
- **Behavior change (bug fix, documented):** `restore` with non-terminal stdin and no ID now behaves as documented — error `restore requires a backup id; …`, exit 2 — including when stdin is `/dev/null`.
- **Test seam (decided, §9 Q2; clarified in r2):** `AGENTIC_SDD_FORCE_INTERACTIVE=1` (exactly `1`) makes the run treat stdin, stdout and stderr as terminals, so the terminal-only behavior (prompts, grouped preview, `hint:` lines, `guide`) can be exercised in a container where output is captured to a file. It never skips or pre-answers a confirmation (default stays No; end-of-input cancels). Color is still decided by the normal rules: a forced terminal is colored unless `NO_COLOR` is non-empty or `TERM=dumb`, so container tests set `NO_COLOR=1` for stable output and one journey (J6) omits it to assert styling is present. It is documented as a test seam, not a user feature.

### FR-02 — Plain output is preserved

When the relevant stream is not a terminal (or color/glyphs are disabled as above), all existing command output on that stream is **byte-identical** to the pre-feature output, for `preview`, `apply`, `backups`, `restore`, `logs`, `version`, `help`, and every error path. New presentation may only be added when that stream is a terminal. **The only exception is the additive help text of FR-04** (the two added `guide` lines in `printUsage` — synopsis line + Commands row — and a topic section printed *before* the unchanged usage block for `help <topic>`), which appears regardless of terminal status (including when `printUsage(stderr)` is called after `unknown command`, top-level flag parse error, or `unexpected argument`); the existing help tests use substring checks and must pass unchanged.

### FR-03 — Grouped preview and next step (terminal stdout only)

When stdout is a terminal, `preview` (and the preview section of `apply`, `--with-opencode`, `--opencode-free-first`) additionally shows, after the unchanged per-line plan **or in place of it, as the plan chooses**:

- counts per client target and totals: *to install*, *to replace (backed up first)*, *up to date*;
- an explicit statement that nothing was written (preview) or where the backup was stored (apply);
- the exact next command, carrying over `--home`, `--repo`, `--with-opencode`, and `--opencode-free-first` when they were supplied.

The summary must not alter the order or content of the data that `apply` acts on. How the counts are derived is a plan decision (§9 Q5); the constraint is that business logic is not duplicated and `skillsync` behavior is unchanged.

### FR-04 — Per-topic help

`agentic-sdd help <topic>` prints topic-specific text for each of `preview`, `apply`, `backups`, `logs`, `restore`, `guide`, `version` (existing `opencode`/`opencode-harness`/`orchestrate` behavior unchanged). Each topic states what the command does, whether it writes, an example, and a related command. `help` with an unknown topic still exits 2 with the existing usage error. The global usage block gains a `guide` synopsis line and a `guide` Commands row (two added lines, nothing reworded); the usage-error string for a bad `help` topic is left byte-identical. Help is never styled.

### FR-05 — Helpful errors, additive only

For a fixed small set of common errors, an extra line `hint: <next step>` follows the **unchanged** `error:` line on stderr, **only when stderr is a terminal** (§9 Q3). Initial set: restore without an ID, unknown command, unknown/invalid flag, backup ID with bad syntax, `--opencode-free-first` without `--with-opencode`, and `no operation logs` — covering only its two missing/empty failure forms: a missing log directory (`os.IsNotExist`, `open …/logs: no such file or directory`) and an empty one (`no operation logs in …`), never when `checkLogPath` rejects a non-directory or symlinked log path. ("No backups found." is normal stdout output, not an error line; it is not given a stderr hint, and the guide handles it in its own flow — r2 clarification A4.) Exit codes never change.

### FR-06 — `agentic-sdd guide`

- **Entry:** only the explicit `guide` command. It accepts `--home PATH` and `--repo PATH` with the same meaning as elsewhere, and no other flags. `agentic-sdd` with no arguments is unchanged.
- **Requires a terminal.** Without an interactive stdin it exits **2** with a message pointing to `preview`, `apply`, and `help guide`, writes nothing, and reads nothing from stdin. This is the same fail-fast stance as `gh auth login --with-token` on a terminal stdin ([cli/cli #14679](https://github.com/cli/cli/pull/14679)).
- **Presentation:** numbered choices and typed answers only; one clearly worded prompt at a time; every prompt shows its default and how to cancel; **no cursor control, redraws, spinners, or alt-screen**; end-of-input cancels and writes nothing (Ctrl+C: see §5). A prompt is never the only way to do something (every action remains a flag-driven command).
- **Menu** (v1): `1) Install or update skills` (default), `2) Restore an earlier backup`, `3) Show the last install log`, `4) What is this?`, `q) Quit`. An invalid answer re-prompts up to two more times, then exits 2. Options `3` and `4` return to the menu after displaying their output; options `1` and `2` (when backups exist and the flow completes or cancels) exit.
- **Install path:** explain in two or three sentences what will be installed and into which four client directories; show the resolved home path; ask whether to add the OpenCode orchestration pack (default **No**), and if yes whether to prefer Free models first (default **No**); show the FR-03 preview summary (without dumping the 72+ raw per-skill lines, while pointing to `agentic-sdd preview` for the full list); ask `Install these N items into <home>? [y/N]`; on **y**, perform the exact `apply` operation (writing the full per-line transcript to the `~/.agentic-sdd/logs/` file as `apply` does, while buffering the raw stdout dump so the terminal shows only the concise closing screen); finish with the install result, backup and log locations, how to undo (`agentic-sdd restore`), and two copy-pasteable first prompts for the user's agent drawn from the existing README "Everyday prompts".
- **Restore path:** pre-check `ListBackups` (if none, print `No backups found.` and return to the menu); otherwise reuse the existing restore selection and its preview-then-confirm behavior.
- **Logs path:** show the same content as `agentic-sdd logs` (or a friendly one-line message if no logs exist yet, then return to the menu).
- **Safety:** the default answer for every confirmation is **No**; **end-of-input (`EOF`) at any prompt — including `[y/N]` option prompts where `No` would otherwise continue the flow — immediately prints `Cancelled.` and exits `0` with nothing written**; nothing is written before an explicit **y**; the guide never bypasses the existing symlink/special-file rejection, staged replace, rollback, or backup behavior; it calls the same code paths as the commands. Input is read through one shared default-sized `*bufio.Reader` so scripted multi-answer stdin is not lost between steps.
- **Experimental flags:** `opencode-harness` and its build tag are untouched and not reachable from `guide`.

### FR-07 — Optional flow trace (logging for test containers and AI agents)

- **Opt-in only.** Enabled by the environment variable `AGENTIC_SDD_TRACE=<path>` when set to a non-empty string (additive; no new flag). When unset or empty (`""`), behavior, output and files are exactly as without this feature.
- **Format:** JSON Lines via the standard library `log/slog` `JSONHandler` ([go.dev/blog/slog](https://go.dev/blog/slog)), one object per event. `log/slog` has been in the standard library since Go 1.21; the repo's `go 1.23` floor is sufficient. **`slog.DiscardHandler` is Go 1.24+ ([go1.24 notes](https://go.dev/doc/go1.24)) and must not be used**; a disabled trace uses a no-op the code owns.
- **Common fields on every event:** `schema` (integer, starts at `1`), `run` (random run id), `seq` (monotonic integer), `event` (stable name), `time` (suppressible in tests via `HandlerOptions.ReplaceAttr`).
- **Events (stable names):** `run.start` (command name, flag *names* only, tool version), `env.resolved` (interactive, stdout_tty, stderr_tty, color, ascii, width), `screen.shown` (`screen` id), `prompt.shown` (`prompt` id, `default`), `prompt.answered` (`prompt` id, `kind`: `default|choice|blank|invalid|eof`, `choice` only for enumerated prompts), `plan.summary` (`install`, `replace`, `up_to_date`), `confirm` (`prompt` id, `decision`: `yes|no`), `op.result` (`op`: bare `"apply"` or `"restore"`, `exit`, `backup_id`: `filepath.Base` validated by `skillsync.ValidateBackupID` when present), `run.end` (`exit`).
- **Privacy and safety:** no free-text user input, no environment dump, no credentials or file contents. Flag *values* are never recorded, and **no filesystem paths are recorded at all** (a backup is identified only by its validated basename id, which is not a path) — a deliberate tightening of the earlier "relative paths" idea. The trace file is opened with `0600` and normalized with `Chmod(0600)` (so a pre-existing file cannot remain looser), appended to (so a multi-command journey yields one file), closed before `runEnv` returns, and refused if the path is a symlink or non-regular file, consistent with how the existing log path is guarded. The trace never creates parent directories.
- **Failure isolation:** a trace that cannot be opened or written never changes the command's result or exit code; one warning line goes to stderr (without echoing the user-supplied trace path) and the run continues.
- **Not evidence:** the trace is not the apply log, not a harness receipt, not approval, and not proof of model or client behavior. `~/.agentic-sdd/logs/` and `agentic-sdd logs` are unchanged.
- **Schema is a public contract** for tests and agents: documented once (FR-08); additive changes keep `schema: 1`; renaming or removing an event or field bumps `schema`.

### FR-08 — Documentation and agent context

- **README** command table and `docs/installer-reference.md` gain `guide`, `help` topics, `AGENTIC_SDD_TRACE`, and the non-TTY rule.
- **One new reference, `docs/guide-flows.md`** (the canonical agent/test context): lists each journey (below) with its screen and prompt ids, the answers that drive it, the expected files/exit code, and the trace events it must emit. It is the single place that owns the schema and journey catalog; tests and the container script reference it instead of restating it.
- `docs/getting-started.md` "First five minutes" gains a short `guide` pointer. No other doc is restructured.

## 5. Edge cases

| Case | Required behavior |
|---|---|
| stdin piped or `/dev/null`, any command | No prompt is ever shown or read; `restore` without ID exits 2 (FR-01); `guide` exits 2 |
| stdout piped but stdin a terminal | Prompts allowed; stdout unstyled; trace/`hint:` rules follow each stream's own terminal status |
| `NO_COLOR=1`, `TERM=dumb`, narrow `COLUMNS` | No ANSI sequences; ASCII glyphs on `dumb`; text unchanged otherwise; no line exceeds 72 columns in guide-owned text |
| End-of-input mid-guide | "Cancelled." and exit 0 with nothing written, matching existing restore cancel behavior; trace records `eof` |
| Ctrl+C at a prompt | No signal handler is installed; the process ends with the terminal's usual interrupt status. Nothing has been written, because writes happen only after an explicit **y**; the trace simply has no `run.end` (r2 clarification A5) |
| No backups when choosing Restore | "No backups found." and return to the menu |
| Preview shows 0 changes | Summary says everything is up to date; the guide does not offer to install |
| `--home` on a read-only or invalid path | Existing error unchanged; hint added on a terminal |
| `AGENTIC_SDD_TRACE` unwritable, a symlink, or a directory | One stderr warning; the command proceeds and exits as it would without tracing |
| Repeating the guided install | Equivalent to repeating `apply`: clean no-op, no new backup |
| Real home | The guide acts on the resolved home exactly like `apply`; it shows the path before confirmation. Tests never use a real home |

## 6. Example guided flow (illustrative, terminal)

```
Agentic SDD — engineering skills for your coding agents
Nothing is written until you confirm.

What would you like to do?
  1) Install or update skills   (default)
  2) Restore an earlier backup
  3) Show the last install log
  4) What is this?
  q) Quit
Choose [1]: 1

This installs 18 skills into four places your agents read from:
  ~/.agents/skills   ~/.codex/skills   ~/.claude/skills   ~/.gemini/antigravity-cli/skills
Home: /home/ana
Add the OpenCode orchestration pack (extra agent profiles)? [y/N]: n

Preview — nothing written yet
  agents   18 to install
  codex    18 to install
  claude   18 to install
  agy      18 to install
  Existing skills are backed up before they are replaced.

Install these 72 items into /home/ana? [y/N]: y
ok  Installed 72 skills. Backup: ~/.agentic-sdd/backups/…  Log: ~/.agentic-sdd/logs/…

Next, in a project, tell your agent:
  "Use agentic-sdd-router. Draft spec.md, plan.md, tasks.md for my approval. Stop before implementing."
Undo any time: agentic-sdd restore
```

The wording is illustrative; counts come from the real plan. Status is always conveyed by words (`ok`, `error`), with an optional glyph and color on capable terminals.

## 7. Journey catalog (the flows that tests and the flow doc cover)

| ID | Journey | Preconditions | Expected outcome |
|---|---|---|---|
| J1 | Guided fresh install, no pack | empty temp home | 72 skill dirs match `apply`; backup/log created per `apply`; trace has `confirm=yes`, `op.result exit=0` |
| J2 | Guided install declined | empty temp home | nothing written; `confirm=no`; exit 0 |
| J3 | Guided restore of an earlier backup | home with one prior apply | restore preview then confirm; same effect as `restore apply ID` |
| J4 | Guide without a terminal | stdin `/dev/null` or pipe | exit 2, hint text, nothing written, nothing read |
| J5 | Scripted automation unchanged | any | `preview`/`apply`/`backups`/`restore ID` output byte-identical, exit codes unchanged |
| J6 | Plain presentation | `NO_COLOR=1` or `TERM=dumb` at a forced terminal | no ANSI sequences, status words present |

## 8. Acceptance criteria and lean verification

Accepted when AC-01–AC-12 hold. Verification is deliberately small: **nine isolated-home Go checks** and **one container journey script**. No snapshot framework, pty libraries, broad matrices, live model tests, or skills-runtime checks. Per the temporary 008 fence in `AGENTS.md`, selection stays within the existing supported-core packages; excluded 005/006/007 work is not selected.

| ID | Acceptance condition | Check |
|---|---|---|
| AC-01 | Environment resolution is correct for the table of inputs (stdin/stdout/stderr terminal or not, `/dev/null`, `NO_COLOR`, `TERM=dumb`, `COLUMNS`) | **C1** unit test of the resolver with injected files/env |
| AC-02 | `restore` without an ID and non-terminal stdin (including `/dev/null`) exits 2 with the documented message | **C1** (resolver) + existing `TestRunRestoreNonInteractiveMissingID` continues to pass |
| AC-03 | Non-terminal output of `preview`, `apply`, `backups`, `restore ID`, `logs`, `version`, `help`, and the listed error paths is byte-identical to pre-feature output; all existing tests pass without assertion changes | **C2** exact-string tests built on the existing one-skill fixture, plus the existing suite |
| AC-04 | Terminal preview shows correct per-client and total counts and the correct next command (flags carried over); no ANSI when color is off | **C3** |
| AC-05 | Each help topic is distinct and exits 0; unknown topic exits 2 unchanged | **C4** |
| AC-06 | `hint:` follows the unchanged `error:` line only when stderr is a terminal, exit codes unchanged | **C5** |
| AC-07 | `guide` without a terminal exits 2, writes and reads nothing | **C6** |
| AC-08 | Guided install produces the same installed files, backup and log as the equivalent `apply`; decline writes nothing; a repeat is a no-op | **C7** (in-process, scripted stdin, temp home) |
| AC-09 | Guided restore and logs delegate to the existing behavior; EOF mid-flow cancels with nothing written | **C8** |
| AC-10 | With `AGENTIC_SDD_TRACE` set, valid JSONL with the documented fields and ordered events; `0600`; no flag values, free text or home path leak; unset → no file and identical behavior; unwritable path → warning only, same exit code | **C9** |
| AC-11 | Journeys J1–J6 run end to end on the real binary in a network-less container with a read-only source mount, final filesystem state and trace events asserted | one script `tests/guide-e2e-in-container.sh`, following `tests/docker-e2e.sh` (`golang:1.25-alpine`, `--network none`, `--pull=never`) |
| AC-12 | A fresh-user walkthrough record exists: one beginner-level user completes J1 on a disposable home, and any point of confusion is listed | written note, human-run; not an automated pass |

Always run: `task test`, `task vet`, `git diff --check`. Real-home commands are never used.

## 9. Decisions (owner accepted the recommended defaults for Q1–Q5 on 2026-11-08)

Q5 was refined by the plan (see `plan.md` §3.5): the counts parser lives in `internal/skillsync` next to the code that prints the lines, and a drift test generates a real plan and asserts the parsed counts. This is the "additive, behavior-neutral change inside `skillsync`" option; no producer line is edited.

| # | Question | Decision (recommended default, accepted) | What it affects |
|---|---|---|---|
| **Q1** | TTY detection mechanism and dependency policy | **Stdlib only:** terminal = character device **and** not the null device. No new module. `golang.org/x/term.IsTerminal` is the conventional alternative (it performs a real terminal ioctl), but its current release requires Go 1.26.0 ([pkg.go.dev](https://pkg.go.dev/golang.org/x/term)), the local cache has only v0.39.0, and the gates run with `GOPROXY=off`; adopting it would need a dependency, a `go.sum`/vendoring decision and possibly a toolchain bump | FR-01, gates, `go.mod` |
| **Q2** | Test seam to force *interactive* on for container journeys (alpine has no pty tool) | Add documented `AGENTIC_SDD_FORCE_INTERACTIVE=1` (FR-01); it never skips confirmation. Alternative considered: test `guide` only in-process | FR-01, AC-11 coverage of J1–J3/J6 |
| **Q3** | `hint:` lines: terminal-only, or always on stderr | **Terminal-only** (keeps non-terminal bytes identical, FR-02). AI agents and CI get the same remediation through `help <topic>`, the trace, and `docs/guide-flows.md`. Alternative: always emit, trading byte-identity for agent visibility | FR-05, AC-03 |
| **Q4** | Where new code lives | **Inside `cmd/agentic-sdd`** (package `main`: environment, presentation, guide and trace files). The earlier assessment sketched `internal/ui`; a new package would have to be added to the three 7-package lists (`Taskfile.yml`, `tests/supported-core-go.sh`, `harness/checks.json`; Lefthook inherits them through `task test`/`task vet`) — a gate change needing separate approval, so it is deferred unless the plan shows `cmd/agentic-sdd` becoming unwieldy | Gates, plan layout |
| **Q5** | Source of preview counts | **Resolved by the plan:** option (b) refined — a small parser owned by `internal/skillsync` (beside the code that prints the lines) turns a preview transcript into counts; the producer lines are not edited; a drift test pins it to real output. Constraint in FR-03 stands | FR-03 |

The owner may also decide whether the `restore </dev/null` change (P5) should ship in the same release as a separately noted bug fix.

## 10. Library trade-offs and why the answer is stdlib (verified 2026-11-08)

| Option | Go floor (from each module's `go.mod`) | Decision |
|---|---|---|
| Stdlib only | repo `go 1.23`, unchanged | **Chosen.** Matches the existing numbered `restore` prompt, is accessible by construction, works with the offline gates, adds zero modules |
| Huh v1 line (v0.8.0) | go 1.23.0 | Rejected for now: adds ≈20 transitive modules, a `go.sum`, and offline-gate work; its accessible mode ([README](https://github.com/charmbracelet/huh/blob/main/README.md)) would give plain prompts equivalent to what we write anyway. Could return as an optional adapter if feedback shows numbered prompts are insufficient |
| Huh v2 `charm.land/huh/v2` | go 1.25.8 | Rejected: forces a toolchain bump above the stated Go 1.23 floor |
| Bubble Tea v2 / Lip Gloss v2 (v2 released Feb 2026, [announcement](https://charm.land/blog/v2/)) | go 1.26.0 / go 1.26.7 | Rejected: a persistent dashboard has no demonstrated need for a four-command installer and would add a large new test surface |

**What would reopen this decision:** the AC-12 walkthrough or issue feedback showing numbered prompts block users; or the project independently raising its Go floor to ≥ 1.25 for other reasons. Any reopening is a new feature, not an amendment.

## 11. Approval

- **Combined `spec.md`/`plan.md`/`tasks.md` approval:** **Approved (2026-11-08)** for `spec.md` (r3), `plan.md` (r2), and `tasks.md` (r2).
- **Approver:** The repository owner in this conversation.
- **Decision evidence (2026-11-08):** After `spec.md` r3, `plan.md` r2, and `tasks.md` r2 were presented for combined approval, the owner instructed: *"i approve docs, and i advice ti begin implementation of all batches using deepseek implemet, free subagent and gemini flash"*.
- **Authorized implementation batches:** All batches `B1` (`T01`, `T02`), `B2` (`T03`, `T04`), `B3` (`T05`, `T06`), `B4` (`T07`, `T08`), and `B5` (`T09`, `T10`), delegated across `sdd-implement-1` (DeepSeek), `sdd-implement-free-1` / `sdd-implement-free-2` (Console Free), and `sdd-implement-9r-1` (9Router Gemini Flash), with parent supervision and focused verification at each batch.
- **Verification review & owner acceptance (2026-11-08):** After Batches `B1`–`B5` completed and passed all checks, the owner instructed *"1 then 2"* (run independent verification review via `sdd-review-verification-2`, then accept the implementation and commit/push). `sdd-review-verification-2` returned `Ready` with zero blocking findings.
- **Commit and push:** Authorized (2026-11-08). Real-home apply, release tag, or gate change remain not authorized.

## 12. Source references

- Repo: `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/logs.go`, `internal/skillsync/{sync,restore}.go`, `tests/docker-e2e.sh`, `tests/e2e-in-container.sh`, `AGENTS.md`, `README.md`, `docs/getting-started.md`, `docs/installer-reference.md`.
- Go standard library: [Structured Logging with slog](https://go.dev/blog/slog); [log/slog](https://pkg.go.dev/log/slog) (`JSONHandler`, `HandlerOptions.Level`/`ReplaceAttr`); [Go 1.24 release notes](https://go.dev/doc/go1.24) (`DiscardHandler`).
- CLI practice: [clig.dev](https://clig.dev/) (prompt only on a TTY, `NO_COLOR`/`TERM=dumb`, never require a prompt); [NO_COLOR](https://no-color.org/); [Vercel CLI guidelines](https://vercel.com/design/cli) (resolve interaction/presentation independently; accessible mode = numbered choices, no redraws, review before consequential work); [gh `GH_FORCE_TTY`](https://cli.github.com/manual/gh_help_environment) (precedent for a forced-terminal seam); [cli/cli #9789](https://github.com/cli/cli/issues/9789) and [PR #14679](https://github.com/cli/cli/pull/14679) (fail fast instead of hanging when stdin is a terminal); [rustup #3429](https://github.com/rust-lang/rustup/issues/3429) (label what Enter will do — hence explicit defaults in every prompt).
- TTY detection pitfall: `os.ModeCharDevice` is true for `/dev/null` (verified locally); real-world fixes use an actual terminal ioctl, e.g. [pilat/devbox #45](https://github.com/pilat/devbox/pull/45).
- Charm: [Huh](https://github.com/charmbracelet/huh/blob/main/README.md) (`WithAccessible`, app must wire the env var itself; accessible mode drops the TUI for standard prompts), [Bubble Tea/Lip Gloss/Bubbles v2](https://charm.land/blog/v2/), and each module's `go.mod` for Go floors.
- Evidence limits: the `go.mod` floors and `x/term` version were read on 2026-11-08 and can change; the trade-off table is a snapshot, not a guarantee.
