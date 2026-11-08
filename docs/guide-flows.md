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
- The guide adds presentation and orchestration only. Every action it performs
  remains available as a flag-driven command, and it writes through the existing
  `apply`/`restore` code paths (staged replace, backup, rollback, symlink and
  special-file rejection).

## Journeys

The six journeys below are the authored flows, not measured usage. `stdin`
shows the literal answers, one per line, for the numbered prompts.

| ID | Journey | Preconditions | `stdin` answers | Expected outcome |
|---|---|---|---|---|
| **J1** | Guided fresh install, no pack | Empty disposable `--home`; a repository or embedded skill set; an interactive terminal | `1`, `n`, `y` — install, no OpenCode pack, confirm | The same installed skill trees as the equivalent `apply`, a backup (and apply log) created as `apply` does; trace records `confirm` with `decision: yes`, `op.result` with `op: apply` and `exit: 0`, and `run.end` with `exit: 0` |
| **J2** | Guided install declined | Empty disposable `--home` | `1`, `n`, `n` — install, no pack, do not confirm | Nothing is written; trace records `confirm` with `decision: no`; exit `0` |
| **J3** | Guided restore of an earlier backup | A home with one prior `apply`, so exactly one backup exists | `2`, `1`, `y` — restore, select backup 1, confirm | The same effect as `restore apply <ID>`; trace records `op.result` with `op: restore`; the restore's own new backup is allowed |
| **J4** | Guide without a terminal | `AGENTIC_SDD_FORCE_INTERACTIVE` unset; `stdin` is `/dev/null` or an empty pipe | none | Exit `2`; nothing written and nothing read; `guide` points to `preview`, `apply`, and `help guide`; `restore` without an ID prints `restore requires a backup id; …` and exits `2` |
| **J5** | Scripted automation unchanged | `AGENTIC_SDD_FORCE_INTERACTIVE` unset; output not a terminal | none | `preview`, `apply`, `backups`, and `restore <ID>` keep byte-identical pre-feature output and exit codes; `preview` ends with `Preview only. Run with apply (or --apply) to install.` and prints no `Next:` summary block |
| **J6** | Color and plain presentation | `AGENTIC_SDD_FORCE_INTERACTIVE=1`, so stdout is a terminal | none | `TERM=xterm-256color` without `NO_COLOR` contains ANSI escapes; `NO_COLOR=1` or `TERM=dumb` contains none; status is still stated as words (`ok`, `error`) |

`tests/guide-e2e-in-container.sh` drives J1–J3 with
`AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 AGENTIC_SDD_TRACE=<file>` and
asserts the final files and grep-level trace events. J4 and J5 run with
`AGENTIC_SDD_FORCE_INTERACTIVE` unset. J6 checks the color rules above.

## Screens and prompts

`guide` emits the following stable identifiers through the trace. Text is
illustrative and may be reworded; identifiers must not change without a
`schema` bump (see below).

Screens (`screen.shown`):

| Screen | When it is shown |
|---|---|
| `menu` | The top-level choice: install, restore, logs, about, quit |
| `install.explain` | What will be installed and into which four client directories, including the resolved home |
| `install.preview` | The per-client counts summary from the real plan (nothing written yet) |
| `install.done` | The result, backup and log locations, undo command, and first prompts |
| `about` | The "What is this?" explanation, after which the menu returns |

Prompts (`prompt.shown` / `prompt.answered`):

| Prompt | Meaning |
|---|---|
| `menu` | Top-level choice (`1`, `2`, `3`, `4`, or `q`) |
| `install.with_opencode` | Add the OpenCode orchestration pack (default `No`) |
| `install.free_first` | Prefer Free models first (asked only after the pack is accepted; default `No`) |
| `install.confirm` | `Install these N items into <home>? [y/N]` (default `No`) |
| `restore.select` | Existing restore backup selection, reused unchanged |
| `restore.confirm` | Existing `Restore these skills? [y/N]` confirmation, reused unchanged |

Install-path order: `menu` → `install.explain` → `install.with_opencode` →
(`install.free_first` when the pack is accepted) → `install.preview` →
`install.confirm` → `install.done`. Options `3` (logs) and `4` (`about`) return
to the menu; options `1` and `2` exit when their flow completes or cancels.

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
