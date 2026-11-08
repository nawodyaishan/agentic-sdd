# Tasks: Feature 010 — Guided CLI Experience, Accessible Presentation & Flow Tracing

**Status:** Approved r2 (2026-11-08) — combined approval recorded in `specs/010-guided-cli-ux/spec.md` §11; Batches B1–B5 implemented, verified (`sdd-review-verification-2` Ready), and accepted by owner (`completed`).
**Spec/plan revisions used:** `spec.md` r3, `plan.md` r2.
**Default specialist and tools:** from `plan.md` §7 (`golang-pro` for Go work; no additional specialist for shell and Markdown; repository gates `task test`, `task vet`, `git diff --check`; Docker with local `golang:1.25-alpine` for the container journey).
**Standing rules for every batch:** stdlib only, no new module, no edit to `go.mod`, `harness/checks.json`, `tests/supported-core-go.sh`, or the `test`/`vet` package lists; tests use `t.TempDir()` homes only; never run `apply`, `guide`, or `restore --apply` against a real home; no commit, tag, push or release unless separately authorized; no skip, filter or tolerance added to any gate. Exactly one batch runs per authorization, then stops for human review.

---

## Batch B1 — Terminal detection fix, `Env`, `runEnv` refactor, Checks C1–C2

**Batch ID:** `B1`
**Included tasks:** `T01`, `T02`
**Outcome:** `Env` and the fixed terminal test exist; `run(...)` keeps its exact signature and behavior as a wrapper over `runEnv`; plain output is pinned byte-for-byte; `restore` without an ID and a non-terminal stdin (including `/dev/null`) exits 2 as documented.
**Completion conditions:** C1 and C2 pass; every existing test passes with no assertion changed; `task test`, `task vet`, `git diff --check` pass.
**State:** `completed — accepted (2026-11-08)`
**Next action:** None (Feature 010 complete).

### T01 — Pin plain output first, then add `env.go`

- **Status:** Completed (`sdd-implement-1`, 2026-11-08).
- **Objective:** (1) **Before changing any code**, build the unmodified binary and capture its literal output and exit codes for: bare `preview` against the one-skill fixture, `version` (fixed `version.*` values), `version extra`, `restore` with no ID and non-interactive stdin, `restore notanid`, `--opencode-free-first` without `--with-opencode`, `logs` with no log directory, and `unknown command` (asserting the exact `"error: unknown command \"wat\"\n"` prefix + `printUsage` output so that T06's two documented `guide` lines in `printUsage` do not break the pin). Encode them as exact-string expectations in C2 and confirm C2 passes against the **unchanged** code. (2) Add `cmd/agentic-sdd/env.go` per `plan.md` §3.1: `Env`, `statter`, `isTerminal` (character device and not the null device), `resolveEnv` (stdin/stdout/stderr, `NO_COLOR`, `TERM=dumb`, `COLUMNS`, `AGENTIC_SDD_FORCE_INTERACTIVE=1`).
- **Acceptance result:** `isTerminal` is true for a fake character device and false for the real null device, a regular file and a pipe; `resolveEnv` yields the expected `Env` for every row of the C1 table (including non-positive/non-integer `COLUMNS` defaulting to 80).
- **Focused verification:** `go test ./cmd/agentic-sdd -run 'TestIsTerminal|TestResolveEnv|TestPlainOutput'` — PASS.

### T02 — Thread `Env` through `runEnv`; fix the non-terminal restore path

- **Status:** Completed (`sdd-implement-1`, 2026-11-08).
- **Objective:** Introduce `runEnv(args, stdout, stderr, stdin, env *Env)`; make `run` build a non-terminal `Env{Interactive: interactive, Width: 80}` and delegate; make `main()` build the real `Env`. Replace the `interactive bool` parameter of `runRestore`/`selectBackup` and add `env *Env` to `runBackups`, `runLogs`, `runApplyWithLog`, `runRestoreWithLog`, and `runOperationWithLog` as `plan.md` §3.2 describes. Use `env.Interactive` where the code currently reads `interactive`. Do not add any new output.
- **Acceptance result:** all existing tests pass unchanged; the C2 expectations still pass; `restore` with `/dev/null` stdin exits 2 with `restore requires a backup id; run "agentic-sdd backups" to list them` (the real-binary check is part of the B5 container journey; here it is asserted through the resolver and `runEnv`).
- **Focused verification:** `task test` and `task vet` — PASS.

---

## Batch B2 — Optional flow trace (FR-07), Check C9

**Batch ID:** `B2`
**Included tasks:** `T03`, `T04`
**Outcome:** With `AGENTIC_SDD_TRACE=<path>` set, every command emits the documented JSONL events; unset, nothing changes.
**Completion conditions:** C9 passes, including the leak and failure-isolation assertions; B1 checks still pass; `task test`, `task vet`, `git diff --check` pass.
**State:** `completed — accepted (2026-11-08)`
**Next action:** None (Feature 010 complete).

### T03 — `trace.go`

- **Status:** Completed (`sdd-implement-free-1`, 2026-11-08).
- **Objective:** Add `Tracer` (nil-safe `Event(name string, kv ...any)` and `Close() error`), `newTracer(w io.Writer, stable bool)`, `openTrace(path string) (io.WriteCloser, error)` (regular files only, `0600` + `f.Chmod(0600)`, append, no parent creation, `Lstat` guard), `commandName` (skipping flags and flag values), `flagNames` (splitting `=value`), and the random run id, per `plan.md` §3.3. `ReplaceAttr` drops `level`, renames `msg`→`event`, and optionally drops `time`. Do not use `slog.DiscardHandler`.
- **Acceptance result:** a test writer receives one valid JSON object per event with `schema`, `run`, `seq`, `event`; `openTrace` rejects a symlink, a directory and a missing parent without creating anything, and normalizes a pre-existing looser file to `0600`.
- **Focused verification:** `go test ./cmd/agentic-sdd -run 'TestTrace'` and `go vet ./cmd/agentic-sdd` — PASS.

### T04 — Emit run-level and operation events; Check C9

- **Status:** Completed (`sdd-implement-free-1`, 2026-11-08).
- **Objective:** Read `AGENTIC_SDD_TRACE` once in `main()` (treating empty `""` as disabled); on open failure print one `warning: trace disabled: <reason>` line to stderr (without echoing the user-supplied trace path) and continue. Emit `run.start`, `env.resolved`, `run.end` (and `defer env.Trace.Close()`) from `runEnv`, and buffer the operation output inside `runOperationWithLog` to emit `op.result` (`op`: bare `"apply"` or `"restore"`, `exit`, `backup_id`: `filepath.Base` extracted from `backup: <path>` and validated by `skillsync.ValidateBackupID`). Write **C9**: ordered `seq`; documented fields; file mode `0600` (including on pre-existing looser file); a sentinel flag value (`--home=/secret`) and sentinel home path never appear anywhere (including `backup_id` and `op`); unset or `""` → no file and identical stdout/stderr/exit; unwritable and symlink targets → warning without path echo and the same exit code as the untraced run.
- **Acceptance result:** AC-10 conditions for run-level events hold (guide-specific events are added in B4).
- **Focused verification:** `task test`, `task vet` — PASS.

---

## Batch B3 — Terminal presentation: summary, topic help, hints; Checks C3–C5

**Batch ID:** `B3`
**Included tasks:** `T05`, `T06`
**Outcome:** At a terminal, `preview` and `apply` add a counts block and the exact next command; each help topic has its own text; the six listed errors gain a `hint:` line on a terminal. Non-terminal output is unchanged.
**Completion conditions:** C3, C4, C5 pass; C2 still passes unchanged; `task test`, `task vet`, `git diff --check` pass.
**State:** `completed — accepted (2026-11-08)`
**Next action:** None (Feature 010 complete).

### T05 — `SummarizePlan` and the terminal summary blocks

- **Status:** Completed (`sdd-implement-free-1`, 2026-11-08).
- **Objective:** Add `internal/skillsync/summary.go` (`PlanSummary`, `SummarizePlan`) without editing any producer line: skip blank lines, classify `install `, `replace + back up `, `up to date `, `remove + back up `, and `restore `, and return per-client counts in deterministic slice order (`agents`, `codex`, `claude`, `agy`, and `opencode-agents` when present). Print the summary only when `env.StdoutTTY`: appended after the unchanged preview transcript; after a successful `apply` (totals, backup id, `Undo: agentic-sdd restore`). The apply **log file stays pre-feature** (summary printed after `runOperationWithLog` returns). Rebuild the `Next:` command from the raw parsed flags saved **before** `resolveHome(home)` overwrites `home` (so `--home` is included only when explicitly passed), quoting values only when needed. Write the **drift test** (every non-empty real preview/restore line over a seeded home with `--with-opencode --opencode-free-first` and an existing `AGENTS.md` is classified or matches the allowlist `Preview only`, `Note:`, `Legacy backup`, `All skills are up to date`, `Installed `, `Restored `) and **C3** (counts correct; next command carries only explicitly passed flags; no ANSI under `NO_COLOR`; ANSI present when color is on).
- **Acceptance result:** AC-04 holds; changing a producer verb makes the drift test fail.
- **Focused verification:** `go test ./internal/skillsync -run 'TestSummarize'` and `go test ./cmd/agentic-sdd -run 'TestTerminalSummary'` — PASS.

### T06 — Topic help and `hint:` lines

- **Status:** Completed (`sdd-implement-free-1`, 2026-11-08).
- **Objective:** Add `helptext.go` with the seven topics; update the `help` case in `main.go` to print the topic text (when present) followed by `printUsage`; add the `guide` synopsis line and Commands row to `printUsage`; add `guide` and `version` to `isHelpTopic` **without editing the `help` usage-error string**. Add the six `hint:` lines of `plan.md` §3.6 printed to stderr only when `env.StderrTTY`, directly after the unchanged `error:` line (for `logs`, only on `os.IsNotExist` or `no operation logs in …`, never when `checkLogPath` rejects a non-directory/symlinked log path); exit codes unchanged. Write **C4** and **C5**.
- **Acceptance result:** AC-05 and AC-06 hold; the existing help tests pass unmodified.
- **Focused verification:** `task test`, `task vet` — PASS.

---

## Batch B4 — `agentic-sdd guide`, guide trace events; Checks C6–C8

**Batch ID:** `B4`
**Included tasks:** `T07`, `T08`
**Outcome:** `agentic-sdd guide` runs the wizard of spec FR-06 over existing operations, traceable end to end.
**Completion conditions:** C6, C7, C8 pass; C9 extended with guide events passes; earlier batches' checks still pass; `task test`, `task vet`, `git diff --check` pass.
**State:** `completed — accepted (2026-11-08)`
**Next action:** None (Feature 010 complete).

### T07 — Prompter and guide state machine

- **Status:** Completed (`sdd-implement-1`, 2026-11-08).
- **Objective:** Add `guide.go` per `plan.md` §3.4: single shared default-sized `*bufio.Reader`; `ask(id, question string, valid []string, def string) (answer string, kind string)` and `confirm(id, question string) (yes bool, eof bool)` (default No; on `eof == true` at **any** prompt — including the `[y/N]` pack questions — print `Cancelled.` and exit `0` immediately); menu with two re-prompts then exit 2 (options `3` and `4` return to the menu loop; `1` and `2` exit on completion); non-interactive → message, exit 2, no stdin read; install path (explanation, pack question default N, free-first question only after yes, summary from `SummarizePlan`, stop if nothing to change, `confirm`, then call `runApplyWithLog` with an in-memory `bytes.Buffer` as its stdout writer so the log file gets the full transcript while the terminal prints only the concise closing screen with undo and two first prompts); restore path (pre-check `ListBackups`; if none, print `No backups found.` and return to the menu; else delegate to `runRestore`); logs path (`showLatestLog`, friendly message when none, then return to the menu). All guide-owned text lines ≤ 72 columns. Add `guide` dispatch in `runEnv` accepting only `--home` and `--repo`.
- **Acceptance result:** AC-07, AC-08 and AC-09 hold; `opencode-harness` is not reachable from `guide`.
- **Focused verification:** `go test ./cmd/agentic-sdd -run 'TestGuide'` — PASS.

### T08 — Guide trace events and tests C6–C8

- **Status:** Completed (`sdd-implement-1`, 2026-11-08).
- **Objective:** Emit `screen.shown`, `prompt.shown`, `prompt.answered` (`kind` of `default|choice|blank|invalid|eof`; the choice value only for enumerated prompts), `plan.summary`, `confirm` from the guide and from the plain `restore` selection/confirmation (checking the `ok` boolean from `readLine` in `selectBackup` and restore confirmation so EOF is recorded as `kind: "eof"`). Write **C6** (no terminal → exit 2, nothing written, reader untouched; `guide --wat` exits 2), **C7** (guided install equals `apply`: installed trees, backup and log, while stdout has only the concise summary + closing screen; decline writes nothing; repeat is a no-op), **C8** (logs then restore with `"3\n2\n1\ny\n"` in one stream; EOF at menu, pack prompt, and final confirm cancels with `Cancelled.` and exit 0, writing nothing). Add the one-line test that both closing-screen example prompts still appear in `README.md`. Extend C9 to assert guide events and that no free text is recorded.
- **Acceptance result:** AC-07–AC-10 hold together.
- **Focused verification:** `task test`, `task vet` — PASS.

---

## Batch B5 — Container journeys, documentation, walkthrough note

**Batch ID:** `B5`
**Included tasks:** `T09`, `T10`
**Outcome:** Journeys J1–J6 run on the real binary in a network-less container; docs and the agent context file exist; a fresh-user walkthrough record is ready for the owner to run.
**Completion conditions:** the new container script and the existing `docker-e2e` both pass; `task test`, `task vet`, `git diff --check` pass; docs link-check by inspection (no tooling added).
**State:** `completed — accepted (2026-11-08)`
**Next action:** None (Feature 010 complete; AC-12 optional human walkthrough remains available).

### T09 — `tests/guide-e2e.sh` and `tests/guide-e2e-in-container.sh`

- **Status:** Completed (`sdd-implement-1`, 2026-11-08).
- **Objective:** Mirror `tests/docker-e2e.sh`/`tests/e2e-in-container.sh` (`golang:1.25-alpine`, `--network none`, `--pull=never`, read-only source mount, `GOTOOLCHAIN=local`). Build the binary; use a one-skill `--repo` fixture and a temporary `--home`. Run J1 (guided install with `AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1`), J2 (decline), J3 (guided restore), J4 (with `AGENTIC_SDD_FORCE_INTERACTIVE` unset: `guide </dev/null`, `guide` with a pipe, and `restore </dev/null` all exit 2), J5 (with `AGENTIC_SDD_FORCE_INTERACTIVE` unset: plain `preview`/`apply`/`backups`/`restore ID` unchanged), J6 (with `AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=xterm-256color` and `NO_COLOR` unset, output contains an ESC byte matched via `$(printf '\033')`; with `NO_COLOR=1` or `TERM=dumb`, output contains none). Drive J1–J3 with answers on stdin and `AGENTIC_SDD_TRACE=<file>`; assert final files and grep-level trace events. Add the task `docker-guide-e2e` to `Taskfile.yml`. Do **not** add the script to CI.
- **Acceptance result:** AC-11 holds; `sh tests/docker-e2e.sh` still passes.
- **Focused verification:** `sh tests/guide-e2e.sh` and `sh tests/docker-e2e.sh` — PASS.

### T10 — Documentation and walkthrough note

- **Status:** Completed (`sdd-implement-1` + parent reconciliation of `run.start.version`, 2026-11-08).
- **Objective:** Update `README.md` (command table, `guide`), `docs/installer-reference.md` (commands, `help` topics, `AGENTIC_SDD_TRACE`, `AGENTIC_SDD_FORCE_INTERACTIVE`, the non-terminal rule, the corrected restore behavior, and the intentional preservation of the `help` usage-error string), and `docs/getting-started.md` (one pointer under "First five minutes"). Create `docs/guide-flows.md` as the single owner of the journey catalog, screen and prompt ids, trace schema (`schema: 1`) and expected states; other docs and tests link to it. Include the blank AC-12 human walkthrough record section at the bottom of this file (`specs/010-guided-cli-ux/tasks.md`, marked `not_run` until a human completes J1 on a disposable home) so no extra file is added to `docs/`.
- **Acceptance result:** AC-12's record section exists; the docs state the supported scope honestly (macOS/Linux terminals; no Windows claim; the `/dev/zero` limitation).
- **Focused verification:** `git diff --check`; manual read of the changed docs — PASS.

---

## AC-12 Fresh-User Walkthrough Record

- **Status:** `not_run` (requires a human walkthrough of J1 on a disposable `--home` after Batch B5).
- **Command run:** *(pending)*
- **Observed friction or confusion:** *(pending)*

---

## Coverage map (spec → tasks)

| AC | Covered by |
|---|---|
| AC-01, AC-02 | T01, T02 (C1, C2) |
| AC-03 | T01 pin, kept green in every batch (C2) |
| AC-04 | T05 (C3) |
| AC-05, AC-06 | T06 (C4, C5) |
| AC-07, AC-08, AC-09 | T07, T08 (C6–C8) |
| AC-10 | T03, T04, T08 (C9) |
| AC-11 | T09 |
| AC-12 | T10 (template) + owner-run walkthrough, not automated |

## Approval and continuation

- **Approval:** Approved (2026-11-08) in `specs/010-guided-cli-ux/spec.md` §11 (`spec.md` r3, `plan.md` r2, `tasks.md` r2).
- **Continuation notes (2026-11-08):**
  - **Current batch / task IDs:** Batches `B1` (`T01`, `T02`), `B2` (`T03`, `T04`), `B3` (`T05`, `T06`), `B4` (`T07`, `T08`), and `B5` (`T09`, `T10`) completed.
  - **Approval reference:** `specs/010-guided-cli-ux/spec.md` §11.
  - **Changed paths:** `cmd/agentic-sdd/{env.go,env_test.go,trace.go,trace_test.go,helptext.go,ui_test.go,guide.go,guide_test.go,main.go,logs.go,main_test.go}`, `internal/skillsync/{summary.go,summary_test.go}`, `tests/{guide-e2e.sh,guide-e2e-in-container.sh}`, `Taskfile.yml`, `README.md`, `docs/{guide-flows.md,installer-reference.md,getting-started.md}`, `specs/010-guided-cli-ux/{spec.md,plan.md,tasks.md}`.
  - **Checks run against current code state:** `task test` (PASS across all 7 supported-core packages), `task vet` (PASS), `go test -count=1 ./internal/skillsync ./cmd/agentic-sdd` (PASS), `sh tests/guide-e2e.sh` (PASS: `Guide end to end checks passed.`), `sh tests/docker-e2e.sh` (PASS: `Docker end to end checks passed.`), `git diff --check` (PASS).
  - **State:** `completed — accepted (2026-11-08)` (`sdd-review-verification-2` returned `Ready`; AC-12 fresh-user walkthrough remains `not_run`, human-owned).
  - **Next action:** None for Feature 010.
