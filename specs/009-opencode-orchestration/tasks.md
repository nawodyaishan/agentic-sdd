# Tasks: Feature 009 — Optional OpenCode Orchestration, Delegation Skill, Model Routing & Context Efficiency

**Status:** Approved (2026-11-07) — combined human approval recorded in `specs/009-opencode-orchestration/spec.md` under `## 11. Approval`; Batches `B1` and `B2` implemented and verified (`awaiting human review`).
**Default specialist and tools:** From `plan.md` (`golang-pro` for Go CLI/installer work and tests; no additional specialist for Markdown templates/docs; repository gates `task test`, `task vet`, `git diff --check`).

---

## Batch B1 — Dedicated `opencode-pack/` assets, MCP agent templates, root embed, and Check 1 static verification

**Batch ID:** `B1`
**Included tasks:** `T01`, `T02`
**Outcome:** The complete `opencode-pack/` directory (`agentic-sdd-orchestrate` skill + 3 references, 12 `--with-opencode` agent `.md` templates, 4 `--opencode-free-first` agent `.md` templates, `routing.md`, `README.md`), root `opencode_pack.go` embed, and **Check 1** static validation test exist and pass `task test`, `task vet`, and `git diff --check` without altering existing CLI behavior.
**State:** `awaiting human review`
**Next action:** Human review of completed Feature 009 (`B1` + `B2`).

### T01 — Create `opencode-pack/skills/agentic-sdd-orchestrate/`, docs, and `opencode_pack.go` embed

- **Status:** Completed (`sdd-implement-free-1`, 2026-11-07).
- **Objective:** Create the opt-in orchestration skill (`opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` and `references/{dispatch-contract,failure-and-reconciliation,context-and-cost}.md`), `opencode-pack/routing.md`, `opencode-pack/README.md`, and root `opencode_pack.go` (`//go:embed all:opencode-pack`).
- **Focused verification:** `go test ./internal/skillsync` and `git diff --check` — PASS.

### T02 — Create the 12 `opencode-pack/agents/*.md` profiles, 4 `presets/free-first/*.md` profiles, and Check 1 static test

- **Status:** Completed (`sdd-implement-free-1`, 2026-11-07).
- **Objective:** Create the 7 new agent profiles (`sdd-orchestrator.md`, `sdd-debug-diagnose.md`, `sdd-test-design.md`, `sdd-research-librarian.md`, `sdd-docs-worker.md`, `sdd-architecture-advisor.md`, `sdd-visual-inspector.md`), the 5 Option B MCP-updated deny-by-default profiles (`sdd-architecture-review-9r.md`, `sdd-code-review-9r.md`, `sdd-review-verification-1.md`, `sdd-review-verification-2.md`, `sdd-lightweight.md`), the 4 Free-first preset profiles (`sdd-implement-1.md`, `sdd-implement-2.md`, `sdd-test-implement-1.md`, `sdd-test-implement-2.md`), and **Check 1** (`TestOpenCodePackAssetsAndPermissions` in `internal/skillsync/opencode_pack_test.go`).
- **Focused verification:** `task test`, `task vet`, and `git diff --check` — PASS.

---

## Batch B2 — Go CLI `--with-opencode` / `--opencode-free-first`, Format 3 single-file sync/restore, and Checks 2–8

**Batch ID:** `B2`
**Included tasks:** `T03`, `T04`, `T05`
**Outcome:** `agentic-sdd [preview|apply] --with-opencode [--opencode-free-first]`, `backups`, and `restore` safely preview, back up, stage, install, and restore the `agentic-sdd-orchestrate` skill and `<home>/.config/opencode/agents/*.md` profiles alongside the 18 core skills, verified by **Checks 2–8** in isolated temporary homes.
**State:** `awaiting human review`
**Next action:** Human review of completed Feature 009 (`B1` + `B2`).

### T03 — Extend `internal/skillsync` for `--with-opencode`, `--opencode-free-first`, `fileDigest`, and Format 3 restore

- **Status:** Completed (`sdd-implement-9r-1`, 2026-11-07).
- **Objective:** Implement `skillsync.Options{WithOpenCode, OpenCodeFreeFirst}`, `SyncWithOptions`, single-file `.md` comparison/staging/backup/rollback, `fileDigest`, Format 3 manifest writing, and Format 3 `ListBackups` / `Restore` in `internal/skillsync/{source,files,manifest,sync,restore}.go`.
- **Focused verification:** `go test ./internal/skillsync` and `go vet ./internal/skillsync` — PASS.

### T04 — Wire `--with-opencode` and `--opencode-free-first` in `cmd/agentic-sdd` and isolate `opencode-harness`

- **Status:** Completed (`sdd-implement-9r-1` + parent reconciliation, 2026-11-07).
- **Objective:** Add `--with-opencode` and `--opencode-free-first` to `cmd/agentic-sdd/main.go` (`preview`/`apply` and `help orchestrate`), enforce that `--opencode-free-first` requires `--with-opencode` (exit `2`), and route `opencode-harness` (alongside `opencode` for 008 boundary compatibility) to the disabled experimental stub.
- **Focused verification:** `go test ./cmd/agentic-sdd` and `go vet ./cmd/agentic-sdd` — PASS.

### T05 — Implement Checks 2–8 Go CLI tests and update CLI documentation

- **Status:** Completed (`sdd-implement-9r-1`, 2026-11-07).
- **Objective:** Add the 7 isolated-home Go CLI tests (**Checks 2–8** from `spec.md` §7) across `internal/skillsync/opencode_sync_test.go` and `cmd/agentic-sdd/main_test.go`, and update `README.md` and `docs/installer-reference.md` with `--with-opencode` and `--opencode-free-first`.
- **Focused verification:** `task test`, `task vet`, and `git diff --check` — PASS.

---

## Batch B3 — Optional OpenAI specialists (`sdd-openai-architect`, `sdd-openai-implement`), full usage guide, repo metadata & release

**Batch ID:** `B3`
**Included tasks:** `T06`, `T07`
**Outcome:** `sdd-openai-architect.md` (`openai/gpt-6.1-sol`) and `sdd-openai-implement.md` (`openai/gpt-6.1-terra`) are added to `opencode-pack/agents/`, verified by Check 1 and Checks 3/6; `README.md`, `docs/orchestration-guide.md`, and `docs/installer-reference.md` provide a complete usage guide; GitHub repo About/topics are updated; and changes are committed, pushed, and tagged for release (`v0.6.0`).
**State:** `completed`

### T06 — Add `sdd-openai-architect.md` and `sdd-openai-implement.md` and update Go pack/sync tests

- **Status:** Completed (2026-11-07).
- **Focused verification:** `go test -count=1 ./internal/skillsync ./cmd/agentic-sdd` and `task vet` — PASS.

### T07 — Complete `README.md` usage section, `docs/orchestration-guide.md`, GitHub repo metadata, and release tag

- **Status:** Completed (2026-11-07).
- **Focused verification:** `task test`, `task vet`, `git diff --check` — PASS.

---

## Continuation state

- **Feature:** `009-opencode-orchestration`
- **Batch & task IDs completed:** `B1` (`T01`, `T02`) — accepted by owner; `B2` (`T03`, `T04`, `T05`) — accepted by owner; `B3` (`T06`, `T07`) — authorized and completed.
- **Approval reference:** `specs/009-opencode-orchestration/spec.md` §11 (`## 11. Approval`, 2026-11-07)
- **Changed paths:**
  - `opencode_pack.go`
  - `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` and `references/{dispatch-contract,failure-and-reconciliation,context-and-cost}.md`
  - `opencode-pack/agents/*.md` (14 files, including `sdd-openai-architect.md` and `sdd-openai-implement.md`) and `opencode-pack/presets/free-first/*.md` (4 files)
  - `opencode-pack/routing.md` and `opencode-pack/README.md`
  - `internal/skillsync/{source,files,manifest,sync,restore,opencode_pack_test,opencode_sync_test}.go`
  - `cmd/agentic-sdd/{main,logs,main_test}.go`
  - `README.md`, `docs/README.md`, `docs/installer-reference.md`, and `docs/orchestration-guide.md`
- **Verification checks run against current code state:**
  - `go test -count=1 ./internal/skillsync ./cmd/agentic-sdd`: PASS (all 8 checks in `spec.md` §7 pass, including the 14 profiles with `sdd-openai-architect` and `sdd-openai-implement`)
  - `task test`: PASS (all 7 supported-core packages)
  - `task vet`: PASS
  - `git diff --check`: PASS
- **Current state:** `completed`
- **Next permitted action:** None pending; `B1`–`B3` delivered, committed, pushed, and tagged per owner authorization.
