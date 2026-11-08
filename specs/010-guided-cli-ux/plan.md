# Plan: Feature 010 — Guided CLI Experience, Accessible Presentation & Flow Tracing

**Spec revision used:** `specs/010-guided-cli-ux/spec.md` r3 (Approved, 2026-11-08).
**Plan revision:** r2 (2026-11-08) — reconciled with `sdd-review-verification-2` and provisional `sdd-architecture-review-9r` (`9router/ag/gemini-4-argon-high`) findings H-1..L-3.
**Approval location:** recorded once, in `specs/010-guided-cli-ux/spec.md` §11. This plan creates no approval.
**Source references:**
- `cmd/agentic-sdd/{main,logs}.go`, `cmd/agentic-sdd/main_test.go` (the `run(args, stdout, stderr, stdin, interactive)` seam and its fixtures `setupBackupCLI`, `read`)
- `internal/skillsync/{sync,restore}.go` (plan-line wording: `install`, `replace + back up`, `up to date`, `remove + back up`; `Preview only…`, `All skills are up to date.`, `Installed N skills; backup: <path>`)
- `tests/docker-e2e.sh`, `tests/e2e-in-container.sh` (container pattern: `golang:1.25-alpine`, `--network none`, `--pull=never`, read-only source mount)
- `Taskfile.yml`, `tests/supported-core-go.sh`, `harness/checks.json` (7-package core selection), `AGENTS.md` (temporary 008 fence, dev-versus-live boundary)
- Go docs: [Structured Logging with slog](https://go.dev/blog/slog), [log/slog](https://pkg.go.dev/log/slog) (`JSONHandler`, `HandlerOptions.ReplaceAttr`), [Go 1.24 notes](https://go.dev/doc/go1.24) (`DiscardHandler` is 1.24+), [`bufio.NewReader`](https://pkg.go.dev/bufio#NewReader) (returns the same reader if it is already a large-enough `*bufio.Reader`)

## 1. Approach

Add presentation, a wizard and a trace **around** the existing operations; do not change what they do. Everything lives in `package main` of `cmd/agentic-sdd` plus one small pure file in `internal/skillsync`. No new package, no new module, no gate edit.

The existing testable seam stays: `run(args, stdout, stderr, stdin, interactive)` keeps its exact signature and meaning and becomes a thin wrapper over a new `runEnv(args, stdout, stderr, stdin, env *Env)`. The wrapper builds an `Env` that is *not* a terminal on stdout/stderr, so every existing test exercises the plain, byte-identical path unchanged. `main()` builds the real `Env` from the process.

Order of work (each step ships safely alone): fix the terminal test and thread `Env` (B1) → trace (B2) → terminal presentation (B3) → `guide` (B4) → container journey and docs (B5).

## 2. Affected components

| Path | Change |
|---|---|
| `cmd/agentic-sdd/env.go` (new) | `Env` struct and `resolveEnv(stdin, stdout, stderr statter, getenv) Env`; `isTerminal(statter)`; color/ASCII/width/force-interactive rules; `Env.Hint`, tiny style helpers |
| `cmd/agentic-sdd/main.go` | `main()` builds `Env` and tracer; `run` becomes wrapper; new `runEnv`; `guide` case; topic help; one usage line; `hint:` calls after the six listed errors; summary block after terminal `preview`; `isHelpTopic` gains `guide`, `version` |
| `cmd/agentic-sdd/trace.go` (new) | `Tracer` (nil-safe) over `slog.JSONHandler`; `openTrace`; `commandName`/`flagNames` helpers |
| `cmd/agentic-sdd/guide.go` (new) | `prompter` (one shared `*bufio.Reader`) and the guide state machine, delegating to existing functions |
| `cmd/agentic-sdd/helptext.go` (new) | per-topic help strings |
| `cmd/agentic-sdd/logs.go` | `runApplyWithLog` tees its transcript to a buffer for the terminal summary; `runOperationWithLog` additionally emits `op.result` and the backup id; log file contents unchanged |
| `internal/skillsync/summary.go` (new) | `SummarizePlan(transcript string) PlanSummary`: pure parser, no I/O, producer lines untouched |
| `*_test.go` | the nine checks of §6 |
| `tests/guide-e2e.sh`, `tests/guide-e2e-in-container.sh` (new) | container launcher and journey script, mirroring the existing `docker-e2e` pair |
| `Taskfile.yml` | one added task `docker-guide-e2e` (no existing task or package list changes) |
| `README.md`, `docs/installer-reference.md`, `docs/getting-started.md`, `docs/guide-flows.md` (new) | FR-08 |

Unchanged: `harness/checks.json`, `tests/supported-core-go.sh`, the `Taskfile.yml` test/vet package lists, `go.mod`, `internal/skillsync` producers (`sync.go`, `restore.go`, `manifest.go`, `files.go`, `source.go`), `opencode*.go`, release config.

## 3. Design decisions

### 3.1 Environment (`env.go`) — FR-01, FR-02

```go
type Env struct {
    Interactive bool // stdin allows prompts
    StdoutTTY   bool
    StderrTTY   bool
    Color       bool // stdout styling on
    ASCII       bool // TERM=dumb
    Width       int
    Trace       *Tracer // nil = disabled
}
```

- `type statter interface{ Stat() (os.FileInfo, error) }` so the resolver is testable with fakes; `*os.File` satisfies it.
- **Terminal test (Q1):** `Mode()&os.ModeCharDevice != 0` **and not the null device** (`os.SameFile(info, nullInfo)` against `os.Stat(os.DevNull)`). Verified on 2026-11-08 with a throwaway Go program: the old test reports `/dev/null` as a terminal; the new one does not; a real pty (via `script`) still reports true. **Residual limitation (documented, accepted):** other non-terminal character devices (for example `/dev/zero`) still pass. `golang.org/x/term.IsTerminal` would be exact, but it is a new dependency and its current release needs Go 1.26.0, so it stays rejected per spec §9 Q1.
- **Force seam (Q2):** `AGENTIC_SDD_FORCE_INTERACTIVE=1` (exactly `1`) sets `Interactive`, `StdoutTTY`, `StderrTTY` true. It never answers or skips a prompt. It is not a privilege boundary: it is equivalent to the user choosing to supply answers on stdin.
- **Color:** `Color = StdoutTTY && getenv("NO_COLOR")=="" && TERM != "dumb"`. `ASCII = TERM == "dumb"`. **Width:** positive integer `COLUMNS`, else 80; used only to size the separator rule (guide text is fixed ≤ 72 columns by construction, so no wrapping code exists).
- **Only stdout is styled.** `hint:` lines on stderr are plain text, avoiding a per-stream color matrix. Status is always a word (`ok`, `error`); a glyph (`✔`) is added only when `Color && !ASCII`.
- **Non-terminal path is the old code path.** All new output is gated on `env.StdoutTTY`/`env.StderrTTY`/`env.Interactive` (guide), so with a non-terminal `Env` the program executes exactly the pre-feature statements.

### 3.2 `runEnv` refactor — keeping tests untouched

`run(...)` = `runEnv(..., &Env{Interactive: interactive, Width: 80})`. `runRestore`, `runBackups`, `runLogs`, `runApplyWithLog`, `runRestoreWithLog`, `runOperationWithLog`, and `selectBackup` gain an `env *Env` parameter in place of `interactive`/nothing; they are not called directly by any existing test (checked: `grep` of the test files found no direct calls). Existing tests keep passing without assertion changes.

### 3.3 Flow trace (`trace.go`) — FR-07

- `AGENTIC_SDD_TRACE` is read once in `main()` (an empty string `""` is treated as disabled, same as unset) and passed in `Env.Trace`, so tests inject writers and never touch process env.
- `Tracer` wraps `*slog.Logger` built with `slog.NewJSONHandler(w, &slog.HandlerOptions{ReplaceAttr: …})` plus an optional `io.Closer`. `ReplaceAttr` (top level only) drops `level`, renames `msg`→`event`, and drops `time` when a test asks for stable output. Verified on 2026-11-08 by running that exact `ReplaceAttr`: output `{"event":"confirm","schema":1,"run":"abc","seq":1,"decision":"yes"}`.
- Common fields via `logger.With("schema",1,"run",runID)`; `seq` is a per-tracer counter attribute. Run id is 8 random bytes (`crypto/rand`, hex).
- **Nil-safe methods** (`func (t *Tracer) Event(name string, kv ...any)` and `func (t *Tracer) Close() error` return immediately when `t == nil`), so a disabled trace is plain no-op code. `runEnv` calls `defer env.Trace.Close()` before returning so the trace file is flushed and closed before `main()` calls `os.Exit`. **`slog.DiscardHandler` is not used** (Go 1.24+; `go vet` reports "requires go1.24 or later" under this module's `go 1.23`, confirmed 2026-11-08, so `task vet` is the guard).
- **Opening the file (`openTrace`):** `Lstat`; refuse if it exists and is not a regular file (symlink, directory, device); `os.OpenFile(path, O_WRONLY|O_APPEND|O_CREATE, 0600)` followed by `f.Chmod(0600)` (so a pre-existing file cannot remain looser than `0600`); never create parent directories. Any failure → one `warning: trace disabled: <reason>` line on stderr (without echoing the user-supplied trace path), tracing off, command unaffected. Residual risk: an `Lstat`-then-open race, the same posture as the existing log-path guard (`checkLogPath`); `O_NOFOLLOW` is avoided to keep the code portable.
- **Never recorded:** flag values, positional arguments, free text, environment, filesystem paths (backup identified by id only). `commandName(args)` skips flag tokens and flag values (`--home /x`, `--home=/x`, `--repo /y`) and returns the known subcommand or `preview`; `flagNames(args)` splits on `=` and returns bare names (`--home=/x` → `home`).
- **Who emits what:** `runEnv` → `run.start`, `env.resolved`, `run.end`; `runOperationWithLog` buffers the operation transcript via `io.MultiWriter(stdout, logFile, &buf)` and emits `op.result` (`op`: bare `"apply"` or `"restore"`, `exit`, `backup_id`: `filepath.Base` extracted from the `backup: <path>` line and validated with `skillsync.ValidateBackupID`); `guide.go` → `screen.shown`, `prompt.shown`, `prompt.answered`, `plan.summary`, `confirm`; `selectBackup` and the restore confirmation check the `ok` boolean from `readLine(reader)` (so EOF is recorded as `kind: "eof"` rather than `"blank"`) and emit `prompt.*`/`confirm`.

### 3.4 Guide (`guide.go`) — FR-06

- `prompter{in *bufio.Reader, out io.Writer, env *Env}` with `ask(id, question string, valid []string, def string) (answer string, kind string)` and `confirm(id, question string) (yes bool, eof bool)` (default No; `y`/`yes` case-insensitive; anything else is No; `eof == true` when `readLine` returns `ok == false` so the caller immediately cancels with `Cancelled.` and exits `0` even on `[y/N]` option prompts where `No` would otherwise continue the flow). `kind` ∈ `default|choice|blank|invalid|eof` for the trace.
- **One reader for the whole run:** `runEnv` wraps stdin once with default-sized `bufio.NewReader(stdin)`; guide passes that same reader on as the `io.Reader` into `runRestore`, whose own `bufio.NewReader(stdin)` returns the same underlying reader by documented stdlib behavior. Check C8 (`"3\n2\n1\ny\n"` in one stdin stream) proves answers are not swallowed.
- **Flow:** non-interactive → message, exit 2, no read. Menu loop (`1` default, `2`, `3`, `4`, `q`; two re-prompts then exit 2; options `3` and `4` return to the menu loop, whereas `1` and `2` exit after completing or cancelling). Install: explanation → pack question (`confirm`, default N; on `eof` cancel immediately) → free-first question only if pack is yes (`confirm`, default N; on `eof` cancel immediately) → `skillsync.SyncWithOptions(…, apply=false)` into a buffer → `SummarizePlan` → if nothing to change, say so and stop; else show per-client counts + resolved home → `confirm` → on yes call `runApplyWithLog` with an in-memory `bytes.Buffer` as its stdout writer (so the `~/.agentic-sdd/logs/` file still gets the full pre-feature transcript and `op.result` is still traced, without dumping 72+ raw lines or a duplicate summary to the terminal) → print the concise closing screen (`ok Installed … Backup: … Log: …`, undo command, and two first prompts). Restore: pre-check `ListBackups`; none → message and back to menu; else `runRestore([]string{"--apply", "--repo", …, "--home", …}, …)`. Logs: `showLatestLog`; missing logs → friendly message, back to menu. After an install or restore the guide exits (0 on success, otherwise the operation's code).
- **Cancel:** end-of-input at any prompt → `Cancelled.`, exit 0, nothing written. No signal handler (spec §5, A5).
- The closing screen's "first prompts" reuse two lines from the README "Everyday prompts" verbatim (single owner stays the README; a one-line test asserts both strings still appear in `README.md` so they cannot drift).

### 3.5 Preview summary and the Q5 resolution — FR-03

- `skillsync.SummarizePlan(transcript string) PlanSummary` skips blank lines and classifies each line by the four sync verbs (`install `, `replace + back up `, `up to date `, `remove + back up `) plus the restore verb (`restore `), extracting `<client>/<name>` (`opencode-agents` counts as a client). Result: per-client counts in deterministic slice order (`agents`, `codex`, `claude`, `agy`, and `opencode-agents` when present) and totals of install / replace / restore / up-to-date / remove. Pure function, no I/O, no change to any producer.
- **Drift pin (this is why option (b) is safe):** a test generates a real preview with `WithOpenCode+OpenCodeFreeFirst` over a seeded home that has an existing `AGENTS.md` (plus a restore preview), and requires **every non-empty output line to be either classified or match a prefix in the short allowlist** (`Preview only`, `Note:`, `Legacy backup`, `All skills are up to date`, `Installed `, `Restored `). A reworded producer line therefore fails the test instead of silently producing wrong counts.
- **Terminal `preview`:** the unchanged transcript (including `Preview only…`) is printed first, then a blank line and a block: per-client counts, totals, "nothing was written", and `Next: agentic-sdd apply [--home …] [--repo …] [--with-opencode] [--opencode-free-first]` rebuilt from the raw parsed flags captured **before** `resolveHome(home)` overwrites `home` (values shell-quoted only when needed). Resolved: the summary is appended, not substituted, so no information is lost.
- **Terminal `apply`:** transcript unchanged; after success, one block with totals, the backup id, and `Undo: agentic-sdd restore`. The apply **log file keeps the pre-feature transcript**: the summary is printed to stdout only, after `runOperationWithLog` returns.
- **Guide:** shows the summary only (not the per-line list) and points to `agentic-sdd preview` for the full list.

### 3.6 Help and hints — FR-04, FR-05

- `helptext.go`: seven topic strings (`preview`, `apply`, `backups`, `logs`, `restore`, `guide`, `version`), each: what it does, whether it writes, one example, one related command. `help <topic>` prints the topic text **then** the unchanged usage block, so the existing tests (which assert `Commands:`/`Usage:` substrings) pass untouched; this is spec A2. The usage block gains a `guide` synopsis line and a `guide` row under Commands (two added lines, nothing reworded). The separate usage-error string for a bad `help` topic (`main.go:49`) is deliberately **not** edited, to keep that error's bytes identical; `help guide` still works. Existing `opencode`/`orchestrate` behavior is untouched.
- `hint:` set (printed on stderr only when `env.StderrTTY`, immediately after the unchanged `error:` line):

| Error | Hint |
|---|---|
| `restore requires a backup id…` | `run agentic-sdd backups to list ids, or agentic-sdd guide to pick one` |
| `unknown command "x"` | `run agentic-sdd help to see the commands` |
| flag parse error | `run agentic-sdd help <command> for the accepted flags` |
| `invalid backup id` | `ids look like 20260101T000000.000000000Z; list them with agentic-sdd backups` |
| `--opencode-free-first requires --with-opencode` | `add --with-opencode, or drop --opencode-free-first` |
| `agentic-sdd logs` fails with a missing log directory (`os.IsNotExist(err)`, printing `error: open …/logs: no such file or directory`) or an empty one (`strings.HasPrefix(err.Error(), "no operation logs in ")`) — never when `checkLogPath` rejects a non-directory or symlinked log path | `logs are written by apply and restore; run agentic-sdd preview first to see what apply would do` |

## 4. Risks and mitigations

| Risk | Level | Mitigation |
|---|---|---|
| Refactoring `run` silently changes plain output | **High impact, low likelihood** | In B1, before touching code, capture literal pre-change output of the pinned commands from the unmodified binary into the C2 test; it must pass before and after the refactor |
| Terminal-only output leaks into non-terminal runs | Medium | All additions gated on `Env` fields; the test wrapper `Env` is non-terminal; C2 pins bytes |
| Swallowed stdin answers when guide delegates to `runRestore` | Medium | Single shared `*bufio.Reader` (documented `bufio.NewReader` identity behavior); C8 uses two answers in one stream |
| Plan-line wording drift breaks counts | Medium | `SummarizePlan` drift test over real producer output (§3.5) |
| Trace becomes an exfiltration or tamper vector | Low–medium | Opt-in; no values, text, env, or paths; `0600`; regular-file-only with `Lstat`; no parent creation; failures never alter exit code; C9 asserts a sentinel home path and flag value never appear |
| `FORCE_INTERACTIVE` mistaken for a safety bypass | Low | Never answers or skips a confirmation; default No; documented as a test seam; C6/C7 show decline and EOF write nothing |
| Go 1.23 floor broken by a newer API | Low | `task vet` reports `requires go1.24 or later` (verified for `slog.DiscardHandler`); CI builds with `go-version-file: go.mod` |
| Residual TTY false positive (`/dev/zero`) | Low | Accepted and documented (§3.1) |
| Scope growth into a TUI | Medium | Spec §2 exclusions; any library adoption is a new feature (spec §10) |

## 5. Security and operational notes

- The guide writes only through the existing `apply`/`restore` code, which already rejects symlinks and special files, stages atomically, backs up, and rolls back. The guide adds no new write path of its own except the opt-in trace file.
- The real-home boundary is unchanged: tests and the container script always pass `--home` to a temporary directory; no step runs `apply` against a real home. A real-home run of `guide` is a separate user decision, as with `apply`.
- No network access anywhere; the container journey runs with `--network none`.
- Dev-versus-live boundary: nothing here authorizes a tag, release, publish, or gate change. Wiring the new container script into CI is **not** part of this plan.

## 6. Verification — nine checks (about eleven test functions) and one container script

All Go checks are isolated-home, in the existing `./cmd/agentic-sdd` and `./internal/skillsync` packages, run by `task test` and `task vet`.

| Check | Where | What it proves |
|---|---|---|
| **C1** | `cmd/agentic-sdd/env_test.go` | `isTerminal` table (fake char device → true; real `/dev/null`, temp file, pipe → false) and `resolveEnv` table (`NO_COLOR`, `TERM=dumb`, `COLUMNS`, `FORCE_INTERACTIVE`, stdin vs stdout vs stderr) |
| **C2** | `main_test.go` | Literal pre-feature output pinned for plain `preview`, `preview` (default), `version`, `version extra`, `restore` without ID (non-interactive), `restore notanid`, `--opencode-free-first` alone, `logs` with none, and `unknown command` (asserting the `error: unknown command "wat"\n` prefix + `printUsage` output so T06's two documented `guide` lines in `printUsage` are accounted for cleanly); `restore` without ID exits 2 with the documented message |
| **C3** | `cmd/agentic-sdd/ui_test.go` + `internal/skillsync/summary_test.go` | Parser drift test; terminal `preview` and `apply` show correct counts and next command (only explicitly passed flags carried), no ANSI under `NO_COLOR`, ANSI present when color is on |
| **C4** | `ui_test.go` | Seven distinct topic outputs exit 0; unknown topic exits 2 unchanged |
| **C5** | `ui_test.go` | `hint:` follows `error:` only when `StderrTTY` (and not on symlinked log path error), exit codes identical |
| **C6** | `guide_test.go` | `guide` without a terminal: exit 2, nothing written, reader never read; `guide --wat` exits 2 |
| **C7** | `guide_test.go` | Guided install produces the same installed trees, backup and log as `apply` while printing only the concise summary + closing screen to stdout; decline writes nothing; repeat is a no-op |
| **C8** | `guide_test.go` | Guided logs then restore (`"3\n2\n1\ny\n"` in one stream) delegates correctly through the shared `*bufio.Reader`; EOF at any prompt (including `[y/N]` pack question) cancels with `Cancelled.` and exit 0, writing nothing |
| **C9** | `trace_test.go` | Valid JSONL, documented fields, ordered `seq`, `0600` (including on pre-existing looser file), no flag value/home path/free text (including in `op.result.backup_id` and `op.result.op`), unset or `""` → no file and same behavior, unwritable/symlink path → warning without path echo and same exit code |

Container script (`tests/guide-e2e-in-container.sh`, launched by `tests/guide-e2e.sh`): builds the real binary and runs J1–J6 against a one-skill `--repo` fixture and a temporary `--home`, driving J1–J3 with `AGENTIC_SDD_FORCE_INTERACTIVE=1` and `NO_COLOR=1`, J4 and J5 with `AGENTIC_SDD_FORCE_INTERACTIVE` unset, and J6 with `AGENTIC_SDD_FORCE_INTERACTIVE=1` (`TERM=xterm-256color` and `NO_COLOR` unset for the color-on check, matching ESC via `$(printf '\033')`; then `NO_COLOR=1` and `TERM=dumb` for the plain checks), answers on stdin, and `AGENTIC_SDD_TRACE=<file>`; asserts final files and grep-level trace events (`"event":"confirm"`, `"decision":"yes"`). It does not parse JSON (alpine has no `jq`); JSON validity is C9's job.

Gate commands: `task test`, `task vet`, `git diff --check`, and the new container script. The existing `docker-e2e` must still pass. Selection stays inside the 7 supported-core packages; no excluded 005/006/007 validation is selected, and no skip or filter is added.

## 7. Specialists and tools

- **`golang-pro`** — installed in the active client; assigned to all Go work in `cmd/agentic-sdd` and `internal/skillsync` (stdlib-only design, `log/slog`, `bufio`, `os` file semantics).
- **No additional specialist needed** for the shell script and Markdown docs: repository guidance and the existing `docker-e2e` pair are the pattern.
- **Conditional tools:** targeted `read`/`grep`/`glob` (this repo has no `.codegraph/` index); Exa for any new external question; Context7 for versioned Go library docs (it supplied the `slog` `ReplaceAttr`/`DiscardHandler` facts used here; available again after a temporary outage during drafting); Docker with the local `golang:1.25-alpine` image for the container journey; repository gates above.

## 8. Next document

`specs/010-guided-cli-ux/tasks.md`.
