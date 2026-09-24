# Spec-Driven Agentic Workflows with Claude Code, Codex, and Zed: What to Keep, What's Outdated, and a Leaner Revised Workflow (September 2026)

Your spec.md → plan.md → tasks.md workflow with human approval gates is still best practice, and it matches a named methodology: Spec-Driven Development, as implemented by GitHub Spec Kit and Kiro. What has aged is the plumbing around the three documents. Loading everything up front, declaring skills and MCP tools in prose, running long single sessions, and adding extra Markdown files for state or handoffs have all been replaced by built-in plan modes, on-demand skills, subagents, checkpoints, and session resume/compaction. Anthropic's own docs name the constraint all of these features exist to manage: "Claude's context window fills up fast, and performance degrades as it fills."

## TL;DR

- **Keep** the three per-feature docs and all three approval gates. Anthropic, OpenAI, and GitHub all document a plan-before-code, spec-reviewed flow. **Change** how the docs are made and used: draft them in plan mode, keep each one short and self-contained, start implementation in a fresh session, and put handoff state inside tasks.md rather than new files.
- **Biggest token and context wins, per the vendors' docs:** `/clear` between tasks (it "costs nothing", while `/compact` is "itself a large request"). Move procedures out of CLAUDE.md/AGENTS.md into on-demand skills. Push exploration, test runs, and logs into subagents on cheaper models. Disable unused MCP servers and prefer CLIs. Use code-intelligence plugins so the agent navigates symbols instead of re-reading files.
- **Model tiering is documented for both tools.** Claude Code: Sonnet for most coding, Opus for architecture and planning (the `opusplan` alias automates this split), `model: haiku` for simple subagent tasks. Codex: `gpt-6-sol` for everyday and complex coding, `gpt-6-luna` for focused, repeatable, high-volume work. Cheap models belong in exploration, summarization, and mechanical tasks, not in drafting spec.md or plan.md.

## Key Findings

### 1. Your workflow is a named, documented methodology

**Documented:** GitHub Spec Kit defines the process as "Constitution once per project; specify → plan → tasks → implement → converge per feature". It produces constitution.md, spec.md, plan.md, and tasks.md, and asks you to "review the result before continuing" at each step. Spec Kit now installs its commands as agent skills (`/speckit-specify`, `/speckit-plan`, `/speckit-tasks`, `/speckit-implement`, `/speckit-converge`, plus the optional `/speckit-clarify`, `/speckit-analyze`, and `/speckit-checklist`). Ry Walker Research reports that Spec Kit "Supports 30+ AI agent integrations including Claude Code, Copilot, Cursor, Gemini CLI, and Codex — with an agent-skills install mode as of v0.10". Spec Kit's integrations reference says Codex skills install into `.agents/skills` and are invoked as `$speckit-<command>`. Kiro uses the same shape with different names: Requirements → Design → Tasks, one Markdown file per step.

**Documented:** Birgitta Böckeler (Thoughtworks, on martinfowler.com, October 2025) separates three levels of SDD:
- **Spec-first:** the spec is used for the task at hand.
- **Spec-anchored:** the spec is kept and evolved with the feature.
- **Spec-as-source:** only the spec is edited by humans.

Your SRS plus high-level spec work as what she calls a "memory bank". Spec Kit calls this the "constitution".

**Recommendation:** Your workflow is spec-first per feature, with project-level documents as a memory bank. Decide explicitly whether feature specs are archived after merge (spec-first) or maintained (spec-anchored). That one decision drives most spec-drift risk.

### 2. What the three vendors document as current best practice

**Claude Code (Anthropic docs, "Best practices for Claude Code"):**

*Documented — workflow and verification:*
- Four phases: Explore → Plan → Implement → Commit, using plan mode (`Shift+Tab` or `claude --permission-mode plan`). `Ctrl+G` opens the plan in your editor.
- Plan mode "adds overhead… If you could describe the diff in one sentence, skip the plan."
- For larger features, have Claude interview you with `AskUserQuestion` and write a spec. Then "start a fresh session to execute it".
- "The most useful specs are self-contained: they name the files and interfaces involved, state what is out of scope, and end with an end-to-end verification step."
- Give Claude a check it can run (tests, build, screenshot). Checks can be gated with `/goal`, a Stop hook, or a verification subagent.

*Documented — context management:*
- `/clear` between unrelated tasks. After two failed corrections, `/clear` and rewrite the prompt.
- `/compact <instructions>`. `/rewind` → "Summarize from here".
- `/btw` for side questions that never enter history.
- "Use subagents to investigate X".
- Checkpoints on every prompt, but they are "not a replacement for git".
- `/rename` plus `claude --continue` / `--resume` to treat sessions "like branches".

*Documented — review:*
- Adversarial review: "Use a subagent to review the rate limiter diff against PLAN.md… Report gaps, not style preferences."
- Anthropic warns that reviewers prompted to find gaps "will usually report some, even when the work is sound".

**Codex (OpenAI docs, "Best practices"):**

*Documented — prompting and planning:*
- Every prompt should state Goal, Context, Constraints, and Done when.
- Pick reasoning effort by task: Low for well-scoped work, Medium/High for complex changes, Extra High for long agentic runs.
- For complex or ambiguous tasks, plan first. Use Plan mode (`/plan` or `Shift+Tab`), ask Codex to interview you, or use a `PLANS.md` template.

*Documented — instructions and skills:*
- Keep AGENTS.md "short, accurate". If it grows, "keep the main file concise and reference task-specific markdown files".
- "When Codex makes the same mistake twice, ask it for a retrospective and update AGENTS.md."
- Skills: "Keep each skill scoped to one job."

*Documented — sessions and subagents:*
- "Keep one thread per coherent unit of work… Fork only when the work truly branches."
- `/resume`, `/fork`, and `/compact` are available, and Codex also compacts automatically.
- Use subagents to "offload bounded work from the main thread" such as exploration, tests, or triage.

**Documented (OpenAI Cookbook, "Using PLANS.md for multi-hour problem solving"):**
- An ExecPlan is a self-contained, living design document with a mandatory Progress section, updated "at every stopping point".
- It tells the agent "do not prompt the user for 'next steps'; simply proceed to the next milestone".
- **Note:** that no-prompting rule conflicts with approval gates during planning. Apply it only after Gate 3.

**Zed (Zed docs):**

*Documented — instructions and skills:*
- AGENTS.md is "the primary instruction file". Personal instructions live in `~/.config/zed/AGENTS.md`.
- For project instructions, Zed "uses the first matching file" from a list: `.rules`, `.cursorrules`, `.windsurfrules`, `.clinerules`, `.github/copilot-instructions.md`, `AGENT.md`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`.
- "Rules have been replaced by Skills and Instructions."
- Skills live in `~/.agents/skills/` or `.agents/skills/`. They use progressive disclosure, support `disable-model-invocation: true`, and should keep SKILL.md "under 500 lines".

*Documented — threads and checkpoints:*
- The Agent Panel has "Restore Checkpoint" on every edit.
- "New From Summary" starts a fresh thread "seeded with a summary of the current conversation". You can @-mention a past thread.
- Thread History and Git worktree isolation are available.
- Agent Profiles choose which built-in and MCP tools a thread may use.

*Documented — External Agents:*
- Claude Agent and Codex run in Zed via ACP as External Agents. They own their own config: "Zed Skills — Do not apply as Zed Skills". Zed MCP servers "may be forwarded over ACP".
- Checkpoints, history restore, and token display "depend on the agent integration".

### 3. Criticisms and failure modes of SDD (credible sources)

**Documented:**
- **Böckeler:** Spec Kit produced many files per spec. She "would rather review code than all these markdown files", saw a "false sense of control" because agents ignored or over-applied instructions, and doubts one fixed workflow fits many problem sizes.
- **Marmelab (François Zaninotto, "Spec-Driven Development: The Waterfall Strikes Back", November 2025):** a Spec Kit run to display the current date produced 8 files and 1,300 lines of text. The post names "Markdown Madness", context blindness (missed existing functions), and double review (spec, then code).
- **Spec drift:** Molisha Shah of Augment Code ("6 Best Spec-Driven Development Tools for AI Coding in 2026", March 2026, updated June 2026) writes that "Static spec tools produce documents that drift from implementation within hours". Treat this as a vendor claim: Augment sells a competing "living spec" product.

**Recommendation:** These critiques hit your workflow's weak spots: doc volume, approval fatigue, and drift. They do not argue for dropping the gates. Size budgets, one file per stage, and a spec-vs-diff review subagent answer them directly.

## Details: KEEP vs OUTDATED vs SIMPLIFY

| Your current practice | Verdict | Why |
|---|---|---|
| Project SRS + high-level spec + tasks doc | **KEEP, but stop auto-loading them** | Documented: CLAUDE.md/AGENTS.md "is loaded every session, so only include things that apply broadly", and `@imports` still load at launch. Recommendation: reference sections by path/heading from spec.md instead of importing whole documents. |
| spec.md → plan.md → tasks.md per feature | **KEEP** | Matches Spec Kit and Kiro, and Anthropic's "write a spec… then start a fresh session". |
| Human approval of each stage | **KEEP** | Every vendor puts plan review before code. Spec Kit asks for review after each command. |
| Agent drafts docs in normal (write-enabled) mode | **OUTDATED** | Use plan mode (Claude `--permission-mode plan`; Codex `/plan`) for read-only exploration while drafting. Edit plans directly with `Ctrl+G`. |
| Drafting, approving, and implementing in one long session | **OUTDATED** | Documented: start a fresh session to execute a spec. `/clear` between tasks. Long sessions "with accumulated corrections" underperform. |
| Skills/MCP declared in plan.md/tasks.md as prose instructions | **SIMPLIFY** | Skills already advertise themselves through name and description, and bodies load on demand. Declare them by name only; never paste skill content into docs. Put side-effecting skills behind `disable-model-invocation: true` so they only run when a task names them. |
| All MCP servers enabled all the time | **OUTDATED** | Documented: "Disable unused servers"; CLIs "are still more context-efficient than MCP servers". Zed Agent Profiles and Codex per-agent `mcp_servers` scope tools per role. |
| Extra Markdown for research, handoffs, notes, checklists | **SIMPLIFY** | Böckeler and Marmelab both cite file sprawl. Put handoff state in a `## Progress` section of tasks.md (the ExecPlan pattern) and let git hold history. |
| Manual re-reading of docs and code at session start | **OUTDATED** | Use `claude --continue/--resume`, Codex `/resume`, and Zed Thread History / New From Summary. Add code-intelligence plugins so the agent navigates symbols instead of reading files. |
| Relying on instructions for must-happen steps (tests, lint) | **OUTDATED** | Documented: hooks "are deterministic and guarantee the action happens", while CLAUDE.md is "advisory". |

## Details: Reducing Usage, Context Bloat, Re-reads, and Markdown Sprawl

### (a) Subscription and usage-limit efficiency

**Documented (Claude Code "Manage costs"):**
- "Sonnet handles most coding tasks well and costs less than Opus. Reserve Opus for complex architectural decisions or multi-step reasoning… For simple subagent tasks, specify `model: haiku`." A switch to Opus also applies to subagents that inherit the session model.
- Thinking tokens bill as output tokens. The default budget "can be tens of thousands of tokens per request", so lower `/effort` for simple tasks.
- Agent teams use "approximately 7x more tokens than standard sessions" when teammates run in plan mode.
- `/usage` on Pro/Max/Team/Enterprise shows usage by skill, subagent, plugin, and individual MCP server, and flags behaviors such as cache misses at 10% or more of usage.
- Team/Enterprise seat allowances reset "on a rolling five-hour window and a weekly window".
- Prompt-cache stats may name the "likely cause" of a miss, for example "tool definitions changed".

**Documented (Codex):**
- Subagents run only "when you explicitly ask", and "subagent workflows consume more tokens than comparable single-agent runs".
- `agents.max_threads` defaults to 6 and `agents.max_depth` to 1. Raising depth "can turn broad delegation instructions into repeated fan-out".

**Recommendation:**
1. Run `/usage` weekly to find which skill or MCP server is eating allowance.
2. Don't toggle MCP servers or skills mid-session. Changing tool definitions invalidates the prompt cache (inferred from the documented miss cause).
3. Use subagents for read-heavy side work only, never for parallel writes in a single-main-agent workflow.

### (b) Context compaction and window bloat

**Documented:**
- `/compact` reads the whole conversation, so compacting a large context "is itself a large request". "When you want a fresh start instead of continuity, `/clear` costs nothing."
- You can steer compaction with CLAUDE.md text such as "When compacting, always preserve the full list of modified files and any test commands".
- After compaction, Claude Code re-attaches the most recent invocation of each skill, "keeping the first 5,000 tokens of each", within a combined 25,000-token budget.
- Project-root CLAUDE.md is re-read after compaction. Path-scoped rules and nested CLAUDE.md files return only when a matching file is read again.
- Codex auto-compacts, and the threshold is configurable with `model_auto_compact_token_limit`.
- In Zed, "New From Summary" is the explicit compaction path.

**Recommendation:** Size each task to finish comfortably inside one context window. Then `/clear`, rather than compacting, at every task boundary. With your state in tasks.md and git, a cleared session loses nothing.

### (c) Repeated and redundant file reading

**Documented:**
- Code-intelligence plugins mean "a single 'go to definition' call replaces what might otherwise be a grep followed by reading multiple candidate files".
- Hooks can pre-filter output, for example grepping a 10,000-line log for `ERROR` to cut "tens of thousands of tokens to hundreds".
- If Claude re-invokes a skill whose content is identical, Claude Code adds "a short note that the skill is already loaded rather than a second copy".
- The built-in Explore and Plan subagents "skip your CLAUDE.md files and the git status snapshot to keep research fast and inexpensive".
- Specific prompts ("add input validation to the login function in auth.ts") avoid "broad scanning".

**Recommendation:** Make plan.md name exact files and interfaces per task, as Anthropic's "self-contained spec" guidance says. The implementing session then reads only those files. Add a "Files already analyzed" line to the tasks.md Progress section so a resumed session doesn't redo exploration.

### (d) Markdown file sprawl

**Documented:**
- Spec Kit's topology is "many files per spec" (Böckeler).
- Claude Code skill listings are capped at 1% of the context window. Descriptions get truncated when you have too many skills, and `/skill-doctor` finds ones to disable.

**Recommendation — hard cap of six agent-facing Markdown artifact types per repo:**
1. AGENTS.md, the canonical instruction file
2. A CLAUDE.md shim
3. The project SRS
4. The project high-level spec
5. The three per-feature docs
6. SKILL.md files

Ban research.md, notes.md, handoff.md, and similar. Their content goes into a section of an existing document: decisions go in the plan.md `## Decision log`, and progress and handoff go in the tasks.md `## Progress`.

## Details: Specialist Skills, Subagents, and MCP Assignment

**Documented (Claude Code Skills):**
- "Unlike CLAUDE.md content, a skill's body loads only when it's used."
- Custom commands have been merged into skills: `.claude/commands/x.md` and `.claude/skills/x/SKILL.md` both create `/x`.
- `!`cmd`` performs dynamic context injection: Claude Code runs the command before the model sees the skill.
- Nested `.claude/skills/` directories load when Claude works in that subdirectory.

**Documented (Codex Skills):**
- Skills live in `$HOME/.agents/skills` and `.agents/skills`. Invoke them explicitly with `$skill-name`.
- `agents/openai.yaml` supports `policy: allow_implicit_invocation: false` and can declare MCP tool dependencies such as `dependencies: tools: - type: "mcp"`.
- Codex's advice for skill descriptions: say "what the skill does and when to use it" and include trigger phrases.

**Documented (Zed):** Zed also reads `.agents/skills/`. When the agent invokes a user skill, Zed "prompts you to allow or deny it".

**Recommendation — how to assign skills and MCP tools in plan.md/tasks.md:**
- Declare per task: `skill: <name>` and `mcp: <server>.<tool>` or `cli: gh`. Never inline instructions from the skill.
- Keep specialist skills narrow ("one job") with `disable-model-invocation: true` (Claude/Zed) or `allow_implicit_invocation: false` (Codex). Then the task list, not the model's guesswork, decides when a skill loads. This makes your existing explicit-assignment habit deterministic.
- Author skills once in `.agents/skills/`, which Codex and Zed both read. For Claude Code, mirror or symlink into `.claude/skills/`. Symlink support is not explicitly documented, so verify with `/context`.
- Use subagents only for three jobs: exploration, verbose operations (tests and logs), and independent review. A "zoo" of overlapping subagent descriptions degrades routing (practitioner report).

## Details: Task Sizing

**Documented:**
- METR's blog post "Measuring AI Ability to Complete Long Software Tasks" (Kwa et al., March 19, 2025) reports that "current models have almost 100% success rate on tasks taking humans less than 4 minutes, but succeed <10% of the time on tasks taking more than around 4 hours". It also says the 50% horizon "has been doubling approximately every 7 months for the last 6 years". METR now flags some claims, including the doubling time, as outdated and points readers to Time Horizon 1.1 (January 2026).
- Anthropic: "Test incrementally: Write one file, test it, then continue." Skip planning when the diff fits in one sentence.
- OpenAI's own `openai/codex` repo AGENTS.md: "Unless the change is mechanical the total number of changed lines should not exceed 800 lines. For complex logic changes the size should be under 500 lines."
- Justin Young's Anthropic Engineering post "Effective harnesses for long-running agents" (November 26, 2025) had the coding agent "work on only one feature at a time", committing after each.

**Recommendation — task-sizing rule for tasks.md:**
- One behavior per task.
- Expect roughly 1–5 files touched and a diff under about 300 lines (well inside OpenAI's 500-line ceiling for complex logic).
- Name one verification command.
- Aim for roughly 15–60 minutes of human-equivalent work.
- If a task needs more than one skill, or touches more than one subsystem, split it.

These numbers are the researcher's synthesis, not vendor rules.

## Details: Session Handoffs and Continuity

**Documented:**
- **Claude Code:** CLAUDE.md is loaded every session. Auto memory loads its first 200 lines or 25 KB. `claude --continue` resumes the latest session and `--resume` opens a picker. `/rename` names sessions. Checkpoints survive closing the terminal.
- **Codex:** AGENTS.md is read "once per run". Instruction files combined are capped at 32 KiB (`project_doc_max_bytes`) and truncation is silent. `/resume`, `/fork`, and `/compact` are available.
- **Zed:** Thread History, New From Summary, @-mention of past threads, Restore Checkpoint, and worktree-per-thread.
- **Anthropic harness:** Justin Young's post "Effective harnesses for long-running agents" (November 26, 2025) is summarized secondhand by ZenML and rajrajhans.com. They say an initializer writes init.sh, `claude-progress.txt`, and a `feature_list.json` of "200+ features (all marked 'failing')". Each coding session then reads git logs and the progress file and picks one feature. Anthropic found JSON "harder for model to corrupt than Markdown".

**Recommendation — handoff protocol with no new files:**
1. At the end of each task, the agent updates `tasks.md ## Progress` in five lines or fewer: done, next, open questions, files touched, verification result. Then it commits (`feat(<feature>): T3 …`).
2. The next session begins with "Read tasks.md ## Progress and `git log -5 --oneline`; do task N". Use `claude --continue` only if the conversation itself is valuable. Otherwise start fresh.
3. Keep task status as checkboxes in tasks.md. If the agent keeps corrupting status, switch that list to a JSON block, following Anthropic's harness finding.

## Details: When Cheaper Models Help

**Documented (Claude Code):**
- Sonnet for most coding, Opus for architecture and multi-step reasoning, `model: haiku` for simple subagent tasks.
- The `opusplan` alias "uses opus during plan mode, then switches to sonnet for execution".
- Since v2.1.198 the built-in Explore subagent inherits the main model (capped at Opus). To keep exploration cheap, define your own `Explore` subagent with `model: haiku`.
- `CLAUDE_CODE_SUBAGENT_MODEL` forces a model onto every subagent.

**Documented (Codex):**
- The Models page says: "Use Sol for complex coding and agentic workflows, and Luna for focused, repeatable tasks." Luna is "Our most efficient model for focused, high-volume tasks, including summarization, extraction, and focused coding."
- Custom agents set `model` and `model_reasoning_effort` per TOML file.
- **Conflict:** OpenAI's subagent pages still recommend `gpt-5.4-mini` and `gpt-5.5`. The Models page says `gpt-5.4`/`gpt-5.4-mini` "retired from Codex with ChatGPT sign-in on August 31, 2026" (replaced by `gpt-6-sol`/`gpt-6-luna`), and that GPT-5.5 retires October 14, 2026. API-key users are unaffected by the 5.4 retirement.

**Recommendation — tiering by workflow stage:**

| Stage | Model tier | Rationale |
|---|---|---|
| spec.md and plan.md drafting | Frontier (Opus / Sol at medium-high, or `opusplan`) | Plan errors compound across every task. It is also the stage you review. |
| tasks.md decomposition | Mid (Sonnet / Sol) | Structured transformation of an approved plan |
| Implementation | Mid (Sonnet / Sol); escalate to Opus for tricky tasks | The vendors' defaults for coding |
| Codebase exploration, grep, "where is X" | Cheap (Haiku / Luna) | Read-only work that returns summaries |
| Test/log summarization, Progress notes, commit messages | Cheap | Summarization and extraction |
| Spec-vs-diff review | Mid or frontier in a fresh subagent | Independence matters more than cost |

## Details: Instruction-File Conventions Across the Three Tools

**Documented:**
- **Codex:** reads AGENTS.md hierarchically (global `~/.codex`, then repo root, then down to the working directory). `AGENTS.override.md` wins at each level. `project_doc_fallback_filenames` adds other filenames.
- **Zed:** loads only the first match in its list, with AGENTS.md ahead of CLAUDE.md. External Agents read their own native files.
- **Claude Code:** its docs say "Claude can also read a repository's AGENTS.md files, on their own or alongside CLAUDE.md". It supports `@path` imports (up to four hops) and path-scoped `.claude/rules/*.md` with `paths:` frontmatter. `/import` copies another agent's configuration. CLAUDE.md should stay under 200 lines.
- **Source conflict:** some third-party guides still say Claude Code does not read AGENTS.md.

**Recommendation:**
- Make AGENTS.md canonical and under about 150 lines.
- Use `CLAUDE.md` = `@AGENTS.md` + Claude-only lines. This works regardless of the conflict above.
- In both files, point to (never import) the SRS, the high-level spec, and `specs/<feature>/`.
- Add one line: "Active feature docs live in specs/<feature>/; read only the doc named in the prompt."

## Recommendations: The Revised Workflow (three docs and three gates preserved)

**Phase 0 — one-time setup.**
- AGENTS.md (+ CLAUDE.md shim), with pointers only.
- Four skills in `.agents/skills/` (mirrored to `.claude/skills/`): `feature-spec`, `feature-plan`, `feature-tasks`, `implement-task`. All side-effecting ones use `disable-model-invocation: true`.
- A cheap `Explore` subagent.
- A Stop or PostToolUse hook that runs lint and tests.
- Unused MCP servers disabled.

**Gate 1 — spec.md.** Session in plan mode on a frontier model. The agent interviews you and writes spec.md: problem, acceptance criteria (Given/When/Then), out of scope, SRS section references, and the end-to-end verification step. Budget: about 150 lines or fewer. You approve.

**Gate 2 — plan.md.** Same session, or a fresh one that reads only spec.md. Contents:
- Files and interfaces to change
- Decisions with rationale
- A skill/MCP assignment table
- Risks
- `## Decision log`

Budget: about 200 lines or fewer. You approve, optionally after a fresh-subagent critique.

**Gate 3 — tasks.md.** Checkbox tasks. Each carries files, skill, MCP/CLI, done-when, and a verify command, sized per the rule above. Add an empty `## Progress` section at the bottom. You approve.

**Implement.** `/clear` or a new session per task (or per two or three tiny tasks). Once Gate 3 passes, the agent proceeds task-to-task without asking, ExecPlan-style, but stops on any deviation from plan.md. After the last task, a review subagent checks the diff against spec.md and plan.md: "report gaps affecting correctness or stated requirements only". Merge, then archive (spec-first) or keep (spec-anchored) the feature folder.

### Example artifacts and commands

**tasks.md task format:**
```markdown
- [ ] T3 Add token refresh to session middleware
  files: src/auth/session.ts, src/auth/session.test.ts
  skill: api-conventions   mcp: none (cli: gh)
  done-when: expired token triggers one refresh; test passes
  verify: pnpm test src/auth/session.test.ts

## Progress
- T1 ✅ T2 ✅ | next: T3 | files analyzed: src/auth/* | open: none
```

**Claude Code skill (`.claude/skills/implement-task/SKILL.md`):**
```markdown
---
name: implement-task
description: Implement exactly one approved task from specs/<feature>/tasks.md. Use only when the user names a task ID.
disable-model-invocation: true
---
Feature: $ARGUMENTS
Current progress:
!`sed -n '/## Progress/,$p' specs/$0/tasks.md`
1. Read only the task block and the files it lists. Use the declared skill/MCP only.
2. Implement, run the task's verify command, iterate until green.
3. Update ## Progress (≤5 lines), commit "feat($0): <task id> <summary>". Stop.
```
Invoke: `/implement-task oauth-login T3`. (The `$0` positional argument and `sed` injection are illustrative. Check argument syntax against your Claude Code version.)

**Cheap exploration subagent (`.claude/agents/Explore.md`):**
```markdown
---
name: Explore
description: Read-only codebase search; returns file paths and a ≤15-line summary.
tools: Read, Grep, Glob
model: haiku
---
Return only paths, symbols, and a short summary. Never paste whole files.
```

**CLI invocations:**
```bash
claude --permission-mode plan --model opusplan      # Gates 1–2 drafting
claude --continue                                   # resume last session
claude -p "Summarize failing tests in the last run" --model haiku
codex                                               # then: /plan, $feature-spec, /resume, /fork
```

**Codex cheap explorer (`.codex/agents/explorer-lite.toml`):**
```toml
name = "explorer_lite"
description = "Read-only explorer that returns distilled findings."
developer_instructions = "Stay read-only. Return paths and a short summary."
model = "gpt-6-luna"
model_reasoning_effort = "medium"
sandbox_mode = "read-only"
```

**Codex skill metadata restricting invocation and declaring MCP (`.agents/skills/feature-plan/agents/openai.yaml`):**
```yaml
policy:
  allow_implicit_invocation: false
dependencies:
  tools:
    - type: "mcp"
      value: "linear"
      description: "Issue tracker for linking tasks"
```

**CLAUDE.md compaction guard (add one line):** `When compacting, preserve: active feature path, current task ID, files modified, verify commands.`

**Zed:**
- Keep AGENTS.md canonical, since Zed loads it before CLAUDE.md.
- Create Agent Profiles such as "Spec" (read-only tools, no MCP) and "Implement" (edit + terminal + the one MCP needed).
- Use New From Summary at task boundaries.
- If you run Claude Agent or Codex inside Zed, configure skills and instructions in those tools' native files, not in Zed Skills.

## Caveats

- **Version churn is extreme.** Model names (Opus/Sonnet versions, gpt-5.x → gpt-6-sol/luna), Explore's default model, and command names changed within months. OpenAI's own subagent and models pages disagree. Verify with `/model`, `/context`, and `/status` before relying on any name here.
- **Secondary sourcing.** The details of Justin Young's harness post (progress file, JSON feature list) come from secondary summaries (ZenML, rajrajhans.com), not a direct fetch.
- **METR's figures** come from a specific benchmark suite and measure autonomous success, not interactive gated workflows. METR itself marks some of the March 2025 claims as outdated.
- **Researcher numbers.** All size budgets (doc line counts, diff sizes, task durations) are the researcher's recommendations, calibrated against the documented numbers above but not vendor-prescribed.
- **Zed "New From Summary" bugs.** Open GitHub issues report the feature failing ("Failed to generate summary") or appearing only near the context limit. Keep tasks.md `## Progress` as the source of truth rather than depending on it.