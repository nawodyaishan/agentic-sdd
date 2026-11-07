# Full Usage & Orchestration Guide

This guide walks through every way to use **Agentic SDD**—from the default 18-skill single-agent workflow to the opt-in **OpenCode Orchestration Pack (`--with-opencode`)**, model routing (`--opencode-free-first`), OpenAI / Claude / Gemini / DeepSeek specialists, MCP tools (`codegraph`, `context7`, `exa`), and safe backups/restores.

---

## 1. Installation & CLI Commands

`agentic-sdd` **previews by default** and never writes to disk unless you run `apply` (or `--apply`).

### Standard install (18 core portable skills)

```sh
# 1. Preview what will be installed or replaced across all 4 client directories
agentic-sdd preview

# 2. Back up any changed existing skills and install the 18 core skills
agentic-sdd apply
```

### Opt-in OpenCode Orchestration Pack (`--with-opencode`)

Pass `--with-opencode` to install the 18 core skills **plus** the `agentic-sdd-orchestrate` skill (into `~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, and `~/.gemini/antigravity-cli/skills`) and **14 OpenCode v2 agent profiles** into `~/.config/opencode/agents/`:

```sh
# Preview the core skills + orchestration skill + 14 OpenCode agent profiles
agentic-sdd preview --with-opencode

# Back up any replaced files and apply
agentic-sdd apply --with-opencode
```

### Opt-in Free-First Writer/Test Preset (`--opencode-free-first`)

Add `--opencode-free-first` alongside `--with-opencode` to also back up and configure the 4 routine implementation/test profiles (`sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, `sdd-test-implement-2`) to use the zero-cost OpenCode Console Free ladder (`opencode/deepseek-v4.1-flash-free` → `glm-5.3-flash-free` → `mimo-v2.6-flash-free`) before asking for paid Go approval:

```sh
agentic-sdd preview --with-opencode --opencode-free-first
agentic-sdd apply --with-opencode --opencode-free-first
```

### Backups, Logs & Restore

Every `apply` that replaces or installs files creates a timestamped, SHA-256-verified backup in `~/.agentic-sdd/backups/` (or `<repo>/backups/` when `--repo` is used). Restore works across both skill directories and `~/.config/opencode/agents/*.md` files without needing extra flags:

```sh
# List backups newest first
agentic-sdd backups

# Preview restoring a specific backup ID (or run interactively on a terminal)
agentic-sdd restore BACKUP_ID

# Apply the restore (backs up current state first so even a restore is reversible)
agentic-sdd restore apply BACKUP_ID

# Inspect the latest apply or restore operation log
agentic-sdd logs

# Show OpenCode orchestration flag help
agentic-sdd help orchestrate
```

---

## 2. How the Core Workflow Works

Whether you work with one main agent or delegate to specialists, Agentic SDD follows the same engineering lifecycle:

1. **Direct Fix (small, well-understood changes):**
   - Skip `specs/` paperwork when scope and consequences are clear.
   - Example prompt:
     > Use `agentic-sdd-implement` to fix the off-by-one check in `internal/skillsync/files.go`. Run `task test` and report the diff.
2. **Feature Work (`spec.md` → `plan.md` → `tasks.md`):**
   - Draft all three documents in `specs/<nnn-slug>/` and review them together for **one combined human approval** recorded in `spec.md` (`## Approval`).
   - Example prompt:
     > Use `agentic-sdd-router` to draft `spec.md`, `plan.md`, and `tasks.md` for Feature 010. Present all three together for my combined approval and stop before implementing.
3. **Batched Implementation & Verification:**
   - Authorize **one batch** at a time from `tasks.md`.
   - The agent implements that batch, runs focused tests (`task test`, `task vet`, `git diff --check`), runs completion review (`agentic-sdd-verification-review`), sets `tasks.md` to `awaiting human review`, and **stops**.
4. **Learning & Challenging Ideas (optional):**
   - `agentic-sdd-teach`: Socratic explanations, guided debugging, or trade-off exercises without turning your repo into a course.
   - `agentic-sdd-grilling` / `agentic-sdd-wait-what`: Stress-test a design or restate an unclear plan in plain language before committing to code.

---

## 3. Using the OpenCode Orchestration Pack (`agentic-sdd-orchestrate`)

When `--with-opencode` is installed, one capable **main agent** remains your sole communicator and decision partner. It classifies every task before considering a subagent:

| Class | Meaning | Action |
|---|---|---|
| **`D0` Direct** | Single-file read, narrow `grep`, `git status`, small understood fix | Handled directly by the main agent (no subagent overhead) |
| **`D1` Cheap Read-Only** | Multi-file discovery, log/failure triage, test-case matrix design | Delegated to one bounded read-only specialist (`sdd-lightweight`, `sdd-debug-diagnose`, `sdd-test-design`) |
| **`D2` Authorized Writer** | Scoped implementation, test writing, or documentation batch | Delegated to one active writer (`sdd-implement-*`, `sdd-openai-implement`, `sdd-test-implement-*`, `sdd-docs-worker`) after human authorization |
| **`D3` Advisor / Research / Vision** | Architecture critique, external docs lookup, screenshot inspection | Delegated on request (`sdd-openai-architect`, `sdd-architecture-advisor`, `sdd-research-librarian`, `sdd-visual-inspector`) |
| **`D4` Critical Main** | Security, auth, migrations, destructive/live operations | Owned directly by the main session model with human supervision |
| **`Blocked`** | Missing approval, unverified partial writes, unauthorized spend | Stops and asks the human |

### Specialist Fleet Installed by `--with-opencode`

| Agent Profile | Role & Mode | Default Model | MCP Access |
|---|---|---|---|
| `sdd-orchestrator` | Primary human-facing profile (`mode: primary`) | Uses active session model | Inherits session tools |
| `sdd-openai-architect` | Read-only architecture/spec review & complex reasoning | `openai/gpt-6.1-sol` | `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-openai-implement` | Complex feature implementation, debugging & refactoring | `openai/gpt-6.1-terra` | `execute`, `codegraph_*`, `context7_*` (`exa_*` denied) |
| `sdd-architecture-advisor` | Read-only architecture/risk alternatives & critique | `9router/ag/claude-sonnet-5-5-high` | `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-debug-diagnose` | Read-only root-cause hypotheses & reproduction pointers | `opencode/deepseek-v4.1-flash-free` | `execute`, `codegraph_*`, `context7_*` |
| `sdd-test-design` | Read-only test cases, edge cases & coverage gap analysis | `opencode/deepseek-v4.1-flash-free` | `execute`, `codegraph_*`, `context7_*` |
| `sdd-research-librarian` | Read-only upstream library/API docs & version checks | `opencode/deepseek-v4.1-flash-free` | `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-docs-worker` | Bounded documentation edits within approved scope | `opencode/deepseek-v4.1-flash-free` | `execute`, `codegraph_*`, `context7_*` |
| `sdd-visual-inspector` | Read-only inspection of screenshots, diagrams & UI logs | `opencode/glm-5.3-flash-free` | `execute`, `codegraph_*`, `context7_*` |
| `sdd-architecture-review-9r` | Read-only Opus architecture & operational risk verdict | `9router/ag/claude-opus-5-5-high` | `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-code-review-9r` | Read-only Opus code & verification review verdict | `9router/ag/claude-opus-5-5-high` | `execute`, `codegraph_*`, `context7_*`, `exa_web_search_exa`, `exa_web_fetch_exa` |
| `sdd-review-verification-1` | Read-only GLM verification review | `opencode-go/glm-5.3-flash` | `execute`, `codegraph_*`, `context7_*` |
| `sdd-review-verification-2` | Read-only Beam Free supplementary verification review | `opencode/beam-free` | `execute`, `codegraph_*`, `context7_*` |
| `sdd-lightweight` | Fast bounded read-only discovery (`steps: 10`) | `opencode/beam-free` | `execute`, `codegraph_*`, `context7_*` |

> **What stays untouched:** Your `~/.config/opencode/opencode.json`, `~/.config/opencode/AGENTS.md`, `sdd-spec-9r.md`, `sdd-plan-9r.md`, `sdd-implement-9r-*.md`, `sdd-implement-free-*.md`, and `vision.md` are never overwritten by `--with-opencode`.

---

## 4. Example Orchestration Prompts

### Delegate an architecture review to OpenAI GPT-6.1 Sol or Claude Opus
> Use `agentic-sdd-orchestrate`. Dispatch `sdd-openai-architect` (or `sdd-architecture-review-9r`) to review `specs/009-opencode-orchestration/spec.md` and `internal/skillsync/` for rollback and manifest compatibility risks. Reconcile the findings against the actual source files and present a ranked summary.

### Delegate complex implementation to OpenAI GPT-6.1 Terra
> I approve `spec.md`, `plan.md`, and `tasks.md` and authorize Batch B1. Use `agentic-sdd-orchestrate` to delegate Batch B1 to `sdd-openai-implement` with `codegraph` and `context7` enabled, then verify the diff and run `task test` and `task vet`.

### Delegate routine work to Free-First / Gemini Flash agents and verify with a Free reviewer
> Start Batch B1 using `sdd-implement-free-1` and Batch B2 using `sdd-implement-9r-1`, then run a read-only verification review with `sdd-review-verification-2` and stop for my review.

---

## 5. MCP Integration (`codegraph`, `context7`, `exa`)

All managed subagent profiles use OpenCode v2 `permissions:` rules so MCP tools work whether a server is configured for Code Mode (`execute`) or direct tool calls (`codemode: false`):

- **CodeGraph (`codegraph_*`):** Allowed across all discovery, debugging, test-design, implementation, and review subagents for one-call symbol and call-graph exploration in `.codegraph/`-indexed repositories.
- **Context7 (`context7_*`):** Allowed across all technical subagents for up-to-date library and framework documentation lookups.
- **Exa (`exa_web_search_exa`, `exa_web_fetch_exa`):** Allowed only on high-intensity architecture, review, and research roles (`sdd-openai-architect`, `sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-architecture-advisor`, `sdd-research-librarian`) with an explicit prompt rule prohibiting private repository code or secrets in external queries.
