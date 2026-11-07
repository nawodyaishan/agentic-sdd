# Feature 009 — Optional OpenCode Orchestration, Delegation Skill, Model Routing & Context Efficiency

**Status:** DRAFT v6 — updated with Claude Opus 5.5 architecture review & owner decisions (1–10); awaiting combined spec/plan/tasks approval  
**Revision date:** 2026-11-07  
**Canonical path:** `specs/009-opencode-orchestration/spec.md`  
**Primary update:** Incorporates `specs/009-opencode-orchestration/refrences/opencode-fleet-consolidated-report.md` (2026-11-07), owner decisions D1–D6 (2026-11-06), `sdd-architecture-review-9r` findings (H1–H5, M1–M9, L1–L5) with owner approvals 1–10 (2026-11-07), Exa-verified OpenCode v2 permission/Code Mode rules (`execute` + `codegraph_*` + `context7_*` + `exa_web_search_exa`/`exa_web_fetch_exa`), dedicated `opencode-pack/` directory separated from legacy 005/006/007 `integrations/opencode/`, user-friendly Go CLI flags (`--with-opencode`, `--opencode-free-first`), and strictly lean Go-CLI-only verification (8 isolated-home Go CLI tests; no skills runtime checks or broad QA suites).

> **Authority notice:** Owner decisions authorize this *specification design*, not changes to the real global `~/.config/opencode` installation; model inference; credentials or spending; native reviewer activation; release publication; or any implementation batch. Existing Agentic-SDD spec → plan → tasks, combined human approval, one authorized implementation batch, and stop-for-review rules govern Feature 009.

## 1. Product outcome

**One capable main agent communicates with the human, decides when delegation is worthwhile, and supervises specialized subagents—but never hands them human authority.** Feature 009 improves main-agent context efficiency and completed-task economics through concise static instructions and OpenCode v2 agent profiles while preserving accuracy, reviewability, and engineering learning.

Feature 009 delivers a dedicated, supported opt-in pack (`opencode-pack/`) through the existing Go CLI (`agentic-sdd`):

- **Skill (`agentic-sdd-orchestrate`):** Portable instructions for the *main agent* defining when and how to delegate, create a scoped task brief, supervise, reconcile evidence, and report to a human. Installed into the 4 standard client skill directories alongside the 18 core skills when `--with-opencode` is passed.
- **Primary agent profile (`sdd-orchestrator.md`):** OpenCode runtime identity (`mode: primary`, no pinned model) that uses the **human-selected effective session model** and invokes eligible subagents through OpenCode's `subagent` tool.
- **Six additive specialist profiles (`sdd-*.md`):** `sdd-debug-diagnose`, `sdd-test-design`, `sdd-research-librarian`, `sdd-docs-worker`, `sdd-architecture-advisor`, and `sdd-visual-inspector` with pinned models, OpenCode v2 `permissions:` lists, and MCP access.
- **Five MCP-updated existing deny-by-default subagent profiles (Option B):** `sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1`, `sdd-review-verification-2`, and `sdd-lightweight`, updated (with automatic backup before replacement) to include `execute`, `codegraph_*`, and `context7_*` (plus `exa_web_search_exa` and `exa_web_fetch_exa` on `sdd-architecture-review-9r` and `sdd-code-review-9r`).
- **Four optional Free-first writer/test profiles (`--opencode-free-first`):** `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, and `sdd-test-implement-2` (backed up and replaced only when `--opencode-free-first` is explicitly passed alongside `--with-opencode`).

## 2. Audience and interaction modes

Support solo coding-agent users first, interns/undergraduates/self-taught learners second, and experienced backend/platform/SRE engineers and maintainers third. Teaching remains **opt-in**.

| Mode | Human decision | Primary agent | Subagents |
|---|---|---|---|
| `direct` (default) | Ordinary request without delegated-agent intent | Uses direct tools, existing SDD skills and reviews | None required |
| `assist` (opt-in orchestration pack) | Human selects the orchestration experience | May assign **one** genuinely useful bounded read-only investigation within policy | Read-only discovery/diagnosis/test design when eligible |
| `delegate` (explicit task/role request) | Human requests specialist work for an already authorized scope | Creates dispatch brief, supervises work, integrates and verifies | Only scoped specialist(s); writing needs explicit authorization |
| `learn` (optional overlay) | Human asks for learning or wants to implement personally | Gives hints/questions/review; avoids stealing the learning task | Only appropriate read-only assistance unless explicitly changed |

## 3. Fleet inventory, disposition, and scope boundaries

### 3.1 Disposition of the 21 existing subagents (Option B approved)

Based on `specs/009-opencode-orchestration/refrences/opencode-fleet-consolidated-report.md` (OpenCode v2.2.5):

| Existing role group | Existing IDs | Feature 009 CLI & spec disposition |
|---|---|---|
| Primary model session | `build`/`plan`, user-selected model | Inspect effective selected model via instructions; CLI never alters global default model; adds optional `sdd-orchestrator.md` |
| Claude/9Router spec, plan & writer | `sdd-spec-9r`, `sdd-plan-9r`, `sdd-implement-9r-2`, `sdd-implement-9r-1`, `sdd-implement-9r-cheap-1`, `sdd-implement-9r-cheap-2` | **Untouched by CLI** in all modes; preserve existing models, prompts, and fallback exception (already inherit MCP tools because they do not use `action: "*"` deny) |
| Claude/9Router deny-by-default reviewers | `sdd-architecture-review-9r`, `sdd-code-review-9r` | **Managed by `--with-opencode` (Option B):** backed up before replacement to add `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, and `exa_web_fetch_exa` while preserving `9router/ag/claude-opus-5-5-high` and existing review prompts |
| Other deny-by-default review & discovery | `sdd-review-verification-1`, `sdd-review-verification-2`, `sdd-lightweight` | **Managed by `--with-opencode` (Option B):** backed up before replacement to add `execute`, `codegraph_*`, and `context7_*` while preserving their existing models (`opencode-go/glm-5.3-flash`, `opencode/beam-free`) and prompts |
| Go implementation/test writers | `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, `sdd-test-implement-2` | **Untouched by `--with-opencode` alone;** backed up and replaced (or installed if absent) only when `--opencode-free-first` is also passed; includes explicit `exa_*` deny |
| Console Free writers, Vision & JSON/built-in | `sdd-implement-free-1`, `sdd-implement-free-2`, `vision`, `explorer` (`opencode.json`), `explore`, `general`, `AGENTS.md` | **Untouched by CLI** in all modes |

### 3.2 Out-of-scope runtime items (managed by user)

| # | Unresolved runtime issue | Feature 009 disposition |
| --- | --- | --- |
| 1 | Project overrides and session model differ from global configuration | Static pre-dispatch instruction in `agentic-sdd-orchestrate/SKILL.md` |
| 2 | Configured models may be unavailable at runtime | Static manual fallback ladder in `opencode-pack/routing.md` and `SKILL.md` |
| 3 | 9Router upstream privacy, billing, and retention | Excluded from scope; managed by user |
| 4 | Permissions do not guarantee OS filesystem isolation (`IR-06-01`) | Excluded from scope; managed by user |
| 5 | MCP server connection health and local `codemode` settings | Excluded from runtime testing; static templates allow both `execute` and `<server>_<tool>` so both Code Mode and direct MCP modes work |

## 4. Exa-verified OpenCode v2 mechanics & CLI separation from 005/006/007

1. **OpenCode v2 permission syntax and Code Mode (`execute`) (`opencode.ai/v2/docs/{permissions,tools,agents}`):**
   - V2 uses `permissions` (ordered list of `{ action, resource, effect }`), `shell`, `subagent`, and `edit` (covering `edit`, `write`, `patch`). V1 names (`permission`, `bash`, `task`, `write`) are rejected.
   - The last matching rule wins.
   - MCP tools use action `<server>_<tool>` with resource `*` (hyphens normalized to `_`).
   - By default, OpenCode exposes MCP servers through Code Mode (`execute`) unless a server sets `codemode: false`. Denying `execute` removes Code Mode and every MCP tool reached through it, while nested tools called inside `execute` still enforce their own `<server>_<tool>` permissions.
   - Therefore, every deny-by-default subagent template places `{ action: "*", resource: "*", effect: "deny" }` first, allows `{ action: "execute", resource: "*", effect: "allow" }` plus `{ action: "codegraph_*", ... }` and `{ action: "context7_*", ... }` (and `exa_web_search_exa` / `exa_web_fetch_exa` on high-intensity roles), and places `.env` and explicit `subagent` / `shell` / `edit` denies at the end.
2. **Strict separation from Feature 005/006/007 experimental harness (Owner decision #3):**
   - `integrations/opencode/` and `adapterfiles.go` remain untouched and reserved for the legacy Feature 005/006/007 experimental ACP/harness material.
   - The legacy experimental CLI subcommand (`cmd/agentic-sdd/opencode.go` and `opencode_disabled.go`) is renamed from `opencode` to `opencode-harness` (gated behind `-tags agentic_sdd_experimental`), freeing `--with-opencode` and `--opencode-free-first` on `agentic-sdd [preview|apply]` to deliver **only** Feature 009's supported opt-in pack from `opencode-pack/`.

## 5. System & CLI delivery architecture

```text
agentic-sdd [preview|apply]                       -> Installs/updates 18 core skills only (default)
agentic-sdd [preview|apply] --with-opencode       -> 18 core skills + agentic-sdd-orchestrate skill (in the 4 client skill dirs)
                                                     + 12 OpenCode agent profiles in <home>/.config/opencode/agents/:
                                                       * 7 new profiles (sdd-orchestrator + 6 specialists)
                                                       * 5 MCP-updated deny-by-default profiles (backed up before replacement)
                                                     (sdd-spec-9r, sdd-plan-9r, sdd-implement-*, vision, opencode.json, AGENTS.md untouched)
agentic-sdd [preview|apply] --with-opencode \
                            --opencode-free-first -> Above + backs up and installs/replaces the 4 designated writer/test profiles
                                                     (sdd-implement-1/-2, sdd-test-implement-1/-2) with Free-first routing
agentic-sdd backups / restore [ID] [--apply]      -> Lists and restores both skill directories and opencode-agents/*.md files
                                                     from the unified backup manifest (no extra flags needed)
```

## 6. Functional requirements

### FR-01 — Primary communicator & `agentic-sdd-orchestrate` skill

- `opencode-pack/agents/sdd-orchestrator.md` defines `mode: primary` with no pinned `model:` so it uses the active session model.
- `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` and its 3 references (`dispatch-contract.md`, `failure-and-reconciliation.md`, `context-and-cost.md`) define main-agent delegation suitability (`D0` direct, `D1` cheap read-only, `D2` authorized writer, `D3` advisor/research/vision, `D4` critical-main, `blocked`), compact task briefs, single-writer supervision, parent verification against real `git diff`/tests, and preserving `agentic-sdd-router` (`references/workflow-policy.md`, `references/human-input.md`), `agentic-sdd-implement`, and `agentic-sdd-verification-review` as single owners.
- `agentic-sdd-orchestrate` is installed into the 4 standard client skill directories (`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`) when `--with-opencode` is passed—never duplicated into `~/.config/opencode/skills/`, so sibling relative links (`../agentic-sdd-router/...`) always resolve and OpenCode sees no duplicate skill IDs.

### FR-02 — Six new specialists, pinned models, and subagent MCP access (Owner decisions #1, #2, #6, #7, #9)

| Agent profile (`.md`) | Disposition in `--with-opencode` | Default pinned model & documented alternatives | V2 `permissions` & MCP access |
|---|---|---|---|
| `sdd-orchestrator.md` | Additive (new) | None (uses session model) | Primary agent; allows `subagent` |
| `sdd-debug-diagnose.md` | Additive (new) | `opencode/deepseek-v4.1-flash-free` (alt: `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high`) | Deny-by-default; allows `read` (except `.env*`), `glob`, `grep`, `skill`, `execute`, `codegraph_*`, `context7_*`; denies `subagent`, `shell`, `edit`, `exa_*` |
| `sdd-test-design.md` | Additive (new) | `opencode/deepseek-v4.1-flash-free` (alt: `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high`) | Same as `sdd-debug-diagnose.md` |
| `sdd-research-librarian.md` | Additive (new) | `opencode/deepseek-v4.1-flash-free` (alt: `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high`) | Deny-by-default; allows `read` (except `.env*`), `glob`, `grep`, `skill`, `webfetch`, `websearch`, `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa`; denies `subagent`, `shell`, `edit` |
| `sdd-docs-worker.md` | Additive (new) | `opencode/deepseek-v4.1-flash-free` (alt: `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high`) | Writer; allows `read` (except `.env*`), `edit`, `glob`, `grep`, `skill`, `execute`, `codegraph_*`, `context7_*`; explicitly denies `subagent`, `shell`, `exa_*` |
| `sdd-architecture-advisor.md` | Additive (new) | `9router/ag/claude-sonnet-5-5-high` (fallback: `9router/ag/gemini-4-argon-high` → `opencode-go/deepseek-v4.1-flash#max`) | Deny-by-default; allows `read` (except `.env*`), `glob`, `grep`, `skill`, `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa`; denies `subagent`, `shell`, `edit` |
| `sdd-visual-inspector.md` | Additive (new) | `opencode/glm-5.3-flash-free` (fallback: `opencode-go/glm-5.3-flash`) | Deny-by-default; allows `read` (except `.env*`), `glob`, `grep`, `execute`, `codegraph_*`, `context7_*`; denies `subagent`, `shell`, `edit`, `exa_*` |
| `sdd-architecture-review-9r.md` | Updated (Option B, backed up) | `9router/ag/claude-opus-5-5-high` (preserved) | Deny-by-default; preserves `read`/`glob`/`grep`/`skill`/`external_directory`/`git diff`/`git status --short` + adds `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa`; denies `subagent`, `edit` |
| `sdd-code-review-9r.md` | Updated (Option B, backed up) | `9router/ag/claude-opus-5-5-high` (preserved) | Same permissions as `sdd-architecture-review-9r.md` |
| `sdd-review-verification-1.md` | Updated (Option B, backed up) | `opencode-go/glm-5.3-flash` (preserved) | Same as above but without `exa_*` (explicitly denies `exa_*`, `subagent`, `edit`) |
| `sdd-review-verification-2.md` | Updated (Option B, backed up) | `opencode/beam-free` (preserved) | Same as `sdd-review-verification-1.md` |
| `sdd-lightweight.md` | Updated (Option B, backed up) | `opencode/beam-free`, `steps: 10` (preserved) | Preserves `.env*`/`*secret*`/`*credential*` denies + adds `execute`, `codegraph_*`, `context7_*`; denies `subagent`, `shell`, `edit`, `exa_*` |

All Exa-enabled profiles include a prompt rule prohibiting private repository code, secrets, or internal identifiers in external search queries.

### FR-03 — Opt-in Free-first writer/test preset (`--opencode-free-first`)

When `--with-opencode --opencode-free-first` is passed, the CLI also installs or replaces (after backup) `sdd-implement-1.md`, `sdd-implement-2.md`, `sdd-test-implement-1.md`, and `sdd-test-implement-2.md` in `<home>/.config/opencode/agents/` using `opencode-pack/presets/free-first/`:
- Primary model set to `opencode/deepseek-v4.1-flash-free` (with documented Free ladder `deepseek-v4.1-flash-free` → `glm-5.3-flash-free` → `mimo-v2.6-flash-free`, then scoped-approval Go ladder `deepseek-v4.1-flash` → `glm-5.3-flash` → `minimax-m3.1-flash` → `mimo-v2.6-flash`).
- Permissions explicitly deny `subagent` and `exa_*` while inheriting `codegraph_*` and `context7_*`.
- If `<home>/.config/opencode/AGENTS.md` exists, `preview` and `apply` print a one-line notice that `AGENTS.md` is not modified by the CLI and `agentic-sdd restore <BACKUP_ID>` reverts the preset.

### FR-04 — Safe Go CLI placement, Format 3 manifest, and single-file digest (Owner decisions #4, #5, #8, #10)

Extend `internal/skillsync` and `cmd/agentic-sdd`:
1. **Target path & label:** Derive `<home>/.config/opencode/agents` strictly from the resolved `--home` directory (never reading `XDG_CONFIG_HOME` when `--home` is set). Use target label `opencode-agents` in preview/apply output (e.g., `install opencode-agents/sdd-orchestrator.md`, `replace + back up opencode-agents/sdd-architecture-review-9r.md`) to avoid colliding with `agents/<skill>`.
2. **Single-file staging & atomic rename:** Stage `.md` files in `<home>/.config/opencode/agents/` using temporary files named `.agentic-sdd-stage-*` (never ending in `.md`, so OpenCode never discovers a half-written or leftover file), chmod to `0644`, and `os.Rename` into `<name>.md` with `.agentic-sdd-undo-*` rollback on error.
3. **Single-file SHA-256 digest (`fileDigest`):** Hash single-file `.md` backups with a dedicated `fileDigest` helper (not `treeDigest`) and verify the SHA-256 digest on `ListBackups` and `Restore`.
4. **Manifest Format 2 / Format 3 compatibility:**
   - Default runs (`agentic-sdd apply` without `--with-opencode`) write `format: 2` with `targets` containing only the 4 skill client paths—100% identical to existing behavior.
   - Runs with `--with-opencode` keep `targets` as the 4 skill client paths, add `opencode_targets: ["<home>/.config/opencode/agents"]`, set `kind: "opencode-agent"` (or `"skill"`) on `entries`, and set `format: 3`. `Restore` and `ListBackups` accept Formats 1, 2, and 3 without needing `--with-opencode` on `restore`. A later default `agentic-sdd apply` updates only the 18 core skills and leaves previously installed opt-in files intact.

## 7. Lean acceptance criteria

Feature 009 is accepted when static inspection (AC-01, AC-03) and the **8 simple Go CLI tests** (AC-02, AC-04 to AC-10) pass. **No skills runtime behavioral checks, no live model smoke tests, and no broad QA suites are used.**

| ID | Acceptance condition | Verification method |
|---|---|---|
| **AC-01** | `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` and its 3 references exist as regular files with valid frontmatter, working relative links (`../agentic-sdd-router/references/...`), and main-agent-only authority rules. | Check 1 (Go static test) + human review |
| **AC-02** | All 12 `opencode-pack/agents/*.md` profiles (7 new + 5 MCP-updated) and 4 `opencode-pack/presets/free-first/*.md` profiles use strictly OpenCode v2 `permissions:` syntax (`shell`, `subagent`, `edit`, `execute`), include `execute` + `codegraph_*` + `context7_*` on deny-by-default profiles, include `exa_web_search_exa`/`exa_web_fetch_exa` on high-intensity roles, and deny `exa_*` on routine/free writers. | Check 1 (Go static test) |
| **AC-03** | `opencode-pack/routing.md` and `opencode-pack/README.md` document the pinned specialist models, the Free-first ladder, the preserved 5 Claude/9Router fallback exception, the 2 Beam cross-lane rules, and CLI restore usage. | Check 1 (Go static test) + human review |
| **AC-04** | Legacy experimental 005/006/007 CLI helper is isolated under `opencode-harness` (`-tags agentic_sdd_experimental`), while default `agentic-sdd` (`preview`/`apply` without `--with-opencode`) manages only the 18 core skills and never touches `<home>/.config/opencode/`. | Check 2 (Go CLI test) |
| **AC-05** | `agentic-sdd preview --with-opencode [--opencode-free-first]` prints planned `agents/...` and `opencode-agents/*.md` actions without modifying the filesystem or creating backups/logs; passing `--opencode-free-first` without `--with-opencode` exits `2`. | Check 2 (Go CLI test) |
| **AC-06** | `agentic-sdd apply --with-opencode` installs the 18 core skills + `agentic-sdd-orchestrate` in the 4 client skill dirs (never `~/.config/opencode/skills`), installs the 7 new `.md` profiles, backs up and updates the 5 deny-by-default `.md` profiles if present, and leaves `sdd-spec-9r.md`, `sdd-plan-9r.md`, `sdd-implement-*.md`, `vision.md`, `opencode.json`, and `AGENTS.md` byte-for-byte untouched. | Check 3 & Check 4 (Go CLI tests) |
| **AC-07** | `agentic-sdd apply --with-opencode --opencode-free-first` backs up existing copies of the 4 designated implement/test `.md` profiles with verified single-file SHA-256 digests (`fileDigest`) in a Format 3 `manifest.json` before replacing them (and prints the `AGENTS.md` notice when `AGENTS.md` exists). | Check 5 (Go CLI test) |
| **AC-08** | Re-running `agentic-sdd apply --with-opencode [--opencode-free-first]` when files are up to date is a no-op that writes nothing and creates no new backup; running default `agentic-sdd apply` afterward does not delete installed opt-in files. | Check 6 (Go CLI test) |
| **AC-09** | The Go CLI rejects symlinks, special files, and non-directory collisions in `<home>/.config/opencode/agents/` and skill targets, stages `.md` files without a `.md` suffix, and rolls back cleanly on simulated mid-apply failure. | Check 7 (Go CLI test) |
| **AC-10** | `agentic-sdd backups` and `agentic-sdd restore [ID] [--apply]` (with no extra flags) restore replaced `.md` profiles, remove freshly installed opt-in `.md` profiles and `agentic-sdd-orchestrate`, and refuse a backup whose saved `.md` file was tampered with. | Check 5 & Check 8 (Go CLI tests) |

### Minimal critical verification plan — eight checks, not a large regression campaign

All 8 checks are deterministic, isolated-home Go tests in `./internal/skillsync` and `./cmd/agentic-sdd` executed by `task test` and `task vet`:

1. **Static pack, V2 frontmatter & MCP permission test:** Verify `opencode-pack/` (embedded and `--repo`) contains `agentic-sdd-orchestrate` (`SKILL.md` + 3 references), 12 `agents/*.md`, and 4 `presets/free-first/*.md`; reject V1 keys (`permission:`, `tools:`, `bash`, `task`, `write`); verify deny-by-default profiles start with `*` deny followed by `execute`, `codegraph_*`, `context7_*` (plus `exa_web_search_exa`/`exa_web_fetch_exa` on high-intensity roles) and end with `subagent`/`shell`/`edit` denies.
2. **Default core isolation & `preview` no-write test:** Verify default `preview`/`apply` installs only the 18 core skills, `--opencode-free-first` alone exits `2`, and `preview --with-opencode [--opencode-free-first]` writes nothing to `<tmp>`.
3. **Additive `--with-opencode` placement test:** On an empty `<tmp>`, verify `apply --with-opencode` installs the 19 skills across the 4 client skill dirs (and not `~/.config/opencode/skills`) plus the 12 `.md` profiles in `<tmp>/.config/opencode/agents/` with modes `0755`/`0644`.
4. **Fleet preservation & Option B MCP update test:** Seed `<tmp>/.config/opencode/` with `opencode.json`, `AGENTS.md`, `sdd-spec-9r.md`, `sdd-plan-9r.md`, `sdd-implement-1.md`, `sdd-implement-free-1.md`, `vision.md`, and the 5 deny-by-default profiles (`sdd-architecture-review-9r.md`, etc.); verify `apply --with-opencode` backs up and updates only the 5 deny-by-default profiles while leaving all other seeded files byte-for-byte untouched.
5. **Opt-in `--opencode-free-first` backup & `fileDigest` tamper test:** Seed the 4 implement/test `.md` files and `AGENTS.md`; verify `apply --with-opencode --opencode-free-first` backs them up with non-empty single-file SHA-256 digests in a Format 3 manifest, replaces the 4 files, prints the `AGENTS.md` notice, and refuses `restore` if a backed-up `.md` file is modified on disk.
6. **Idempotent no-op & subsequent default `apply` test:** Verify re-running `apply --with-opencode --opencode-free-first` is a no-op with no new backup, and a subsequent default `agentic-sdd apply` leaves the opt-in skill and `.md` profiles intact.
7. **Symlink rejection, non-`.md` temp staging & rollback test:** Verify symlinks/non-directory collisions in `<tmp>/.config/opencode/agents/` fail closed before writes, staging temp files never end in `.md`, and a simulated mid-apply failure rolls back both skill and `.md` changes.
8. **Unified `backups` and `restore` round-trip test:** Verify `agentic-sdd restore apply <ID> --home <tmp>` restores the 5 MCP-updated and 4 Free-first `.md` files to their seeded originals and removes freshly installed opt-in `.md` files and `agentic-sdd-orchestrate`.

## 8. Dedicated repository structure (`opencode-pack/`)

```text
opencode-pack.go                 # root //go:embed all:opencode-pack (separate from skillsfiles.go & adapterfiles.go)
opencode-pack/
  skills/
    agentic-sdd-orchestrate/
      SKILL.md
      references/
        dispatch-contract.md
        failure-and-reconciliation.md
        context-and-cost.md
  agents/                        # 12 profiles managed by --with-opencode
    sdd-orchestrator.md
    sdd-debug-diagnose.md
    sdd-test-design.md
    sdd-research-librarian.md
    sdd-docs-worker.md
    sdd-architecture-advisor.md
    sdd-visual-inspector.md
    sdd-architecture-review-9r.md
    sdd-code-review-9r.md
    sdd-review-verification-1.md
    sdd-review-verification-2.md
    sdd-lightweight.md
  presets/
    free-first/                  # 4 profiles managed by --with-opencode --opencode-free-first
      sdd-implement-1.md
      sdd-implement-2.md
      sdd-test-implement-1.md
      sdd-test-implement-2.md
  routing.md
  README.md
specs/009-opencode-orchestration/
  spec.md
  plan.md
  tasks.md
```

## 9. Verification and delivery policy

- **Simple Go CLI & static asset verification only:** Feature 009 is verified exclusively by `task test`, `task vet`, and `git diff --check` across the existing 7 supported-core Go packages (`harness/checks.json` package list remains unchanged).
- **No skills runtime checks or broad QA campaigns:** No live LLM/subagent smoke tests, ACP session runners, or Python harness suites.
- **Dev-versus-live boundary preserved:** All automated tests use `t.TempDir()`. Running `agentic-sdd apply --with-opencode` against the real `~/.config/opencode` or publishing a release requires separate human authorization.

## 10. Settled design decisions for `plan.md` & `tasks.md`

All architectural review items (1–10) are settled by owner decision:
1. Mandatory `execute` + `codegraph_*` + `context7_*` (+ `exa_web_search_exa`/`exa_web_fetch_exa` on high-intensity roles) on deny-by-default subagent templates.
2. Option B selected: `--with-opencode` installs the 7 new profiles and backs up/updates the 5 existing deny-by-default profiles (`sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1/-2`, `sdd-lightweight`).
3. Legacy 005/006/007 CLI helper isolated under `opencode-harness`; Feature 009 assets live in dedicated top-level `opencode-pack/` embedded via `opencode-pack.go`.
4. Format 3 manifest when `opencode-agents` entries are recorded; Format 2 preserved for default runs.
5. Dedicated `fileDigest` for single-file `.md` backups + tamper test in Check 5.
6. Strictly OpenCode v2 `permissions:` syntax and action names (`shell`, `subagent`, `edit`, `execute`).
7. Exact `exa_web_search_exa` / `exa_web_fetch_exa` allows on high-intensity roles and `exa_*` denies on routine/free writers.
8. `agentic-sdd-orchestrate` installed in the 4 standard client skill dirs only (not duplicated in `~/.config/opencode/skills/`).
9. Pinned specialist models per §6 FR-02.
10. Target label `opencode-agents/<name>.md`, non-`.md` staging temp files, `AGENTS.md` notice on `--opencode-free-first`, and strict `--home` path derivation.

## 11. Approval

- **Combined `spec.md`/`plan.md`/`tasks.md` approval:** Pending (drafting `plan.md` and `tasks.md` next for combined approval).
- **Owner spec & architecture decisions (2026-11-07):** Owner approved items 1–10 from the `sdd-architecture-review-9r` review, including Option B for MCP profile updates, dedicated `opencode-pack/` directory separated from 005/006/007 harness, pinned specialist models, and the 8 Go-CLI-only verification checks.
- **Implementation batch authorization:** None yet.
- **Global OpenCode configuration changes (`~/.config/opencode`):** None authorized.
- **Release publication:** None authorized.
