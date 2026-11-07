# Plan: Feature 009 — Optional OpenCode Orchestration, Delegation Skill, Model Routing & Context Efficiency

**Spec revision used:** `specs/009-opencode-orchestration/spec.md` v6 (Approved, 2026-11-07).
**Approval location:** Combined approval is recorded only in `specs/009-opencode-orchestration/spec.md` under `## 11. Approval`.
**Source references:**
- `specs/009-opencode-orchestration/spec.md` (DRAFT v6)
- `specs/009-opencode-orchestration/refrences/opencode-fleet-consolidated-report.md` (2026-11-07)
- `specs/009-opencode-orchestration/refrences/SKILL(1).md`
- `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/opencode_disabled.go`, `cmd/agentic-sdd/opencode_disabled_test.go`
- `internal/skillsync/{sync,restore,manifest,source,files}.go`
- `skillsfiles.go`, `adapterfiles.go`, `harness/checks.json`

## 1. Approach

Feature 009 is delivered in two focused, independently verifiable steps that preserve the existing 7-package core layout and require only simple Go CLI and static asset tests:

1. **Package the opt-in skill and OpenCode v2 agent templates in a dedicated `opencode-pack/` directory (`opencode_pack.go`).**
   - Keep `skills-files/` (the 18 default core skills) and `integrations/opencode/` (legacy Feature 005/006/007 experimental harness material) untouched.
   - Place the new `agentic-sdd-orchestrate` skill (`SKILL.md` + 3 reference files), the 12 `--with-opencode` agent profiles (7 new + 5 Option B MCP-updated deny-by-default profiles), the 4 `--opencode-free-first` agent profiles, `routing.md`, and `README.md` under top-level `opencode-pack/`, embedded via `//go:embed all:opencode-pack` in root `opencode_pack.go`.
   - Validate all embedded and on-disk Markdown frontmatter and OpenCode v2 `permissions:` ordering (`*` deny first → `execute` + `codegraph_*` + `context7_*` + role-specific `exa_web_search_exa`/`exa_web_fetch_exa` allows → `.env*` and `subagent`/`shell`/`edit` denies last) via **Check 1**.
2. **Extend `internal/skillsync` and `cmd/agentic-sdd` for `--with-opencode` and `--opencode-free-first` alongside existing skills (Checks 2–8).**
   - Add `skillsync.Options{WithOpenCode, OpenCodeFreeFirst}` and `SyncWithOptions`, keeping `Sync(repo, home, apply, out)` unchanged for default runs.
   - Generalize `change`, backup copying, staging, and `manifest.json` (Format 3 when single-file `opencode-agents` entries are present, with dedicated `fileDigest` SHA-256 hashing) so `apply`, `backups`, and `restore` work through a single atomic transaction for both skill directories and `<home>/.config/opencode/agents/*.md` files.
   - Isolate the legacy Feature 005/006 experimental subcommand under `opencode-harness` (`-tags agentic_sdd_experimental`) so `--with-opencode` and `--opencode-free-first` in `cmd/agentic-sdd` never collide with Feature 008's experimental-helper boundary check.

## 2. Affected components

- `opencode_pack.go` (new, module root):
  - `//go:embed all:opencode-pack` into `var OpenCodePackFiles embed.FS`.
- `opencode-pack/` (new directory):
  - `skills/agentic-sdd-orchestrate/SKILL.md` and `references/{dispatch-contract,failure-and-reconciliation,context-and-cost}.md`.
  - `agents/*.md` (14 files): `sdd-orchestrator.md`, `sdd-debug-diagnose.md`, `sdd-test-design.md`, `sdd-research-librarian.md`, `sdd-docs-worker.md`, `sdd-architecture-advisor.md`, `sdd-visual-inspector.md`, `sdd-openai-architect.md`, `sdd-openai-implement.md`, `sdd-architecture-review-9r.md`, `sdd-code-review-9r.md`, `sdd-review-verification-1.md`, `sdd-review-verification-2.md`, `sdd-lightweight.md`.
  - `presets/free-first/*.md` (4 files): `sdd-implement-1.md`, `sdd-implement-2.md`, `sdd-test-implement-1.md`, `sdd-test-implement-2.md`.
  - `routing.md` and `README.md`.
- `internal/skillsync/source.go` & `files.go`:
  - `openCodePackFS(repo string) (fs.FS, error)` resolving `opencode-pack` from `OpenCodePackFiles` (default) or `<repo>/opencode-pack` (`--repo`).
  - `sameSourceTargetFile(source fs.FS, srcPath, dstPath string) (bool, error)` and `copyFileFromSource(source fs.FS, srcPath, dstPath string) error` for single `.md` agent files (mode `0644`, rejecting symlinks/special files).
- `internal/skillsync/manifest.go`:
  - Add optional Format 3 fields to `manifest`: `OpenCodeTargets []string` (`json:"opencode_targets,omitempty"`) and `Kind string` (`json:"kind,omitempty"`, `"skill"` or `"opencode-agent"`) on `entry`.
  - Add `fileDigest(source fs.FS, name string) (string, error)` computing the SHA-256 hex digest of a single regular file's contents.
  - Keep `Format: 2` when a backup contains only skill entries; write `Format: 3` when any `opencode-agent` entry is recorded.
- `internal/skillsync/sync.go`:
  - Add `Options struct { WithOpenCode bool; OpenCodeFreeFirst bool }` and `SyncWithOptions(repo, home string, apply bool, opts Options, out io.Writer) error`.
  - When `opts.WithOpenCode` is true:
    - Include `agentic-sdd-orchestrate` in the skill set planned across the 4 client skill targets (`agents`, `codex`, `claude`, `agy`).
    - Plan the 12 `.md` files (plus the 4 Free-first `.md` files when `opts.OpenCodeFreeFirst` is true) against target `target{name: "opencode-agents", path: filepath.Join(home, ".config", "opencode", "agents")}`.
    - Stage single-file changes via `os.CreateTemp(c.target.path, ".agentic-sdd-stage-*")` (no `.md` suffix), chmod to `0644`, and `os.Rename` into `<name>.md` with `.agentic-sdd-undo-*` rollback on error.
    - When `opts.OpenCodeFreeFirst` is true and `<home>/.config/opencode/AGENTS.md` exists, print the one-line notice that `AGENTS.md` is not modified by the CLI.
- `internal/skillsync/restore.go`:
  - Extend `loadBackup` and `Restore` to accept Format 3 manifests (`opencode-agents` backup subdirectory, `.md` filename validation, `fileDigest` check, and single-file restore/remove) without requiring any extra flags on `backups` or `restore`.
- `cmd/agentic-sdd/main.go`, `opencode.go`, `opencode_disabled.go`, `opencode_disabled_test.go`, `opencode_test.go`:
  - Add `--with-opencode` and `--opencode-free-first` to `preview`/`apply` in `main.go` (exiting `2` if `--opencode-free-first` is used without `--with-opencode`).
  - Isolate the legacy 005/006 experimental subcommand under `opencode-harness` (`help opencode-harness`, `agentic-sdd opencode-harness ...`) so default `--help` can document `--with-opencode` and `--opencode-free-first` cleanly while keeping the experimental harness disabled and unreachable in default builds.
- `internal/skillsync/*_test.go` and `cmd/agentic-sdd/main_test.go`:
  - Implement the 8 Go CLI verification checks.

## 3. Key design decisions

1. **Dedicated `opencode-pack/` root directory (Owner decision #3 / H2):** Keeps Feature 009 assets completely separate from `skills-files/` (the 18 default skills) and `integrations/opencode/` (legacy 005/006/007 experimental harness files).
2. **Option B profile set in `--with-opencode` (Owner decision #2 / H1):** `--with-opencode` installs the 7 new profiles and backs up/updates the 5 existing deny-by-default profiles (`sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1/-2`, `sdd-lightweight`) so subagents have working `execute` + `codegraph_*` + `context7_*` (+ `exa_web_search_exa`/`exa_web_fetch_exa` on architecture/code review) out of the box.
3. **Format 2 preserved for default runs; Format 3 for OpenCode agent runs (Owner decision #4 / H3):** Default `agentic-sdd apply` writes byte-for-byte identical Format 2 manifests. Only backups containing `opencode-agents/*.md` entries write `format: 3` with `opencode_targets` and `kind: "opencode-agent"`, and `fileDigest` hashes single files accurately (Owner decision #5 / H4).
4. **Non-`.md` temporary staging files (Owner decision #10 / M9):** Temp files in `<home>/.config/opencode/agents/` use `.agentic-sdd-stage-*` and `.agentic-sdd-undo-*` (never ending in `.md`), preventing OpenCode from indexing an incomplete or leftover file.
5. **Strict `--home` derivation (Owner decision #10 / M8):** `<home>/.config/opencode/agents` is always computed from `filepath.Join(home, ".config", "opencode", "agents")`, ignoring `XDG_CONFIG_HOME` when `--home` is provided.
6. **Trusted check contract (`harness/checks.json`) unchanged:** All Go code and the 8 tests live in existing core packages (`./internal/skillsync` and `./cmd/agentic-sdd`).

## 4. Risks and mitigations

- **Single-file vs directory rollback in `installChanges` (medium, testable):** Mixed skill-directory and single-file `.md` changes share one rollback loop. Mitigated by keeping the same stage → rename-old-to-undo → rename-stage-to-target state machine for both directories and files, verified by Check 7.
- **Digest verification on single files (medium, testable):** Using `treeDigest` on a file would return the empty-input hash. Mitigated by dedicated `fileDigest` and Check 5's tamper test.
- **Duplicate skill discovery in OpenCode (low, resolved):** Installing `agentic-sdd-orchestrate` only in the 4 standard client skill directories (and not `~/.config/opencode/skills/`) avoids duplicate skill IDs and preserves relative sibling links (`../agentic-sdd-router/references/...`).

## 5. Specialists and tools

- **`golang-pro`** (installed in the active client): assigned for all Go implementation and test work in `internal/skillsync`, `cmd/agentic-sdd`, and `opencode_pack.go`.
- **No additional specialist needed** for the Markdown skill/agent templates and documentation (`opencode-pack/`): repository guidance, `specs/009-opencode-orchestration/spec.md` §6, and Exa-verified OpenCode v2 documentation are sufficient.
- **Conditional tools:**
  - Local code discovery: targeted `read` / `grep` / `glob` across `internal/skillsync` and `cmd/agentic-sdd` (this repo has no `.codegraph/` index).
  - External docs research: Exa (already used during spec/architecture review to verify OpenCode v2 permission and Code Mode semantics; available if a concrete question arises).
  - Repository verification gates: `task test`, `task vet`, and `git diff --check`.

## 6. Next document

`specs/009-opencode-orchestration/tasks.md`.
