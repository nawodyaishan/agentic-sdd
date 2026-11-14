---
title: Orchestrate optional specialist agents
description: Enable the optional OpenCode pack, issue bounded briefs, inspect child claims and apply manual fallback.
---

Use this guide when you explicitly want one main OpenCode agent to delegate bounded tasks to specialist subagents. The default Agentic-SDD setup installs only the 18 core portable skills in [the skill catalog](../reference/skills.md) for single-agent work.

Nothing in this workflow creates automatic model failover, an operating-system sandbox, or a substitute for human approval (`opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`).

## Enable the optional OpenCode pack

Check what will change before writing any files. Bare `asdx-core` previews the 18 core skills only, while `--with-opencode` selects the optional pack in `opencode-pack/` (`cmd/agentic-sdd/main.go`, `internal/skillsync/sync.go`):

```sh
# Read-only preview: 18 core skills + agentic-sdd-orchestrate + 14 agent profiles
asdx-core preview --with-opencode

# Isolated scratch preview (writes nothing; safe for checking paths)
asdx-core preview --with-opencode --home /tmp/example-opencode-home
```

To write into your real home directory (a live operation requiring explicit authorization), run `apply`:

```sh
# Install core skills, agentic-sdd-orchestrate, and 14 profiles into ~/.config/opencode/agents/
asdx-core apply --with-opencode

# Also back up and install the 4 Free-first writer/test templates
asdx-core apply --with-opencode --opencode-free-first
```

Key CLI and TUI behavior differences:

- **CLI vs TUI default:** `asdx-core` keeps `--with-opencode` off by default. By contrast, `asdx` (and `asdx-core tui`) defaults `--with-opencode=true`; pass `--with-opencode=false` for core-only mode (`cmd/agentic-sdd/tui.go`). Legacy `agentic-sdd` wraps `asdx-core` and `agentic-sdd-tui` wraps `asdx` (`internal/buildinfo/buildinfo.go`).
- **Flag dependency:** Passing `--opencode-free-first` without `--with-opencode` exits `2` (`cmd/agentic-sdd/main.go`).
- **Destinations:** `agentic-sdd-orchestrate` is installed alongside sibling skills in the four client directories (`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`) so relative links to `../agentic-sdd-router/references/` resolve; it is not duplicated into `~/.config/opencode/skills/` (`opencode-pack/README.md`, `internal/skillsync/sync.go`). The 14 profiles (and 4 optional Free-first presets) go to `<home>/.config/opencode/agents/`.
- **Backups and recovery:** Existing skills and replaced profiles are backed up before replacement under `<home>/.agentic-sdd/backups/` (or `<repo>/backups/` with `--repo`). See [the files and backups reference](../reference/files-and-backups.md) and [the CLI reference](../reference/cli.md) for `asdx-core backups` and `asdx-core restore`.

## Distinguish shipped profiles, presets, adapters and ambient agents

Before dispatching a child, separate what `--with-opencode` installs from what already exists locally on a machine (`opencode-pack/README.md`, `internal/skillsync/opencode_adapters_test.go`):

| Group | Source directory | Installed by | Purpose and boundary |
|---|---|---|---|
| 14 shipped pack profiles | `opencode-pack/agents/` | `asdx-core apply --with-opencode` | `sdd-orchestrator` (`primary`, inherits session model) plus 13 `subagent` profiles (`opencode-pack/agents/`) |
| 4 Free-first preset templates | `opencode-pack/presets/free-first/` | `asdx-core apply --with-opencode --opencode-free-first` | `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, `sdd-test-implement-2` |
| 2 fleet adapter profiles | `opencode-pack/adapters/` | Not installed by `--with-opencode` | `sdd-spec-9r` and `sdd-plan-9r`; resolver-only sources for `asdx-core fleet` (`internal/fleet/resolve.go`) |
| Ambient / local OpenCode files | User `~/.config/opencode/` | Untouched by installer | Local `sdd-implement-9r-*`, `sdd-implement-free-*`, `vision`, `opencode.json`, and `AGENTS.md` |

When `--opencode-free-first` runs and `<home>/.config/opencode/AGENTS.md` already exists, `internal/skillsync/sync.go` prints a notice confirming `AGENTS.md` was left unmodified and naming `asdx-core restore <BACKUP_ID>`.

For the full per-profile table, mode, pinned models, and permission rules, see [the agent profile catalog](../reference/profiles.md).

## Classify work and enforce consent boundaries

The main human-facing agent loads `opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md` and classifies each task before spawning any child (`opencode-pack/skills/agentic-sdd-orchestrate/references/context-and-cost.md`):

- **`D0` direct:** Single reads, short searches, known answers, or small understood fixes stay in the main session with no child.
- **`D1` cheap read-only:** Multi-file discovery, failure triage, or test-case design using one bounded read-only profile (`sdd-lightweight`, `sdd-debug-diagnose`, `sdd-test-design`).
- **`D2` authorized writer:** Approved implementation, test writing, or bounded documentation edits (`sdd-docs-worker`, `sdd-openai-implement`, or Free-first `sdd-implement-*` / `sdd-test-implement-*`). One writer at a time per checkout.
- **`D3` advisor, research, or vision:** Architecture critique (`sdd-architecture-advisor`, `sdd-openai-architect`), external docs lookup (`sdd-research-librarian`), or screenshot inspection (`sdd-visual-inspector`).
- **`D4` critical-main:** Approvals, permissions, credentials, releases, or destructive operations stay in the main session.
- **`blocked`:** Missing approval, unclear path isolation, missing network/cost authority, or ambiguous working-tree state stops before dispatch.

Enforce these consent and delegation limits (`opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`):

- **Explicit authorization:** Writers (`D2`), side-effecting tests, paid or cross-lane model switches, external web queries (`exa_*`, `webfetch`, `websearch`), image disclosure, and independent verification reviews require explicit authorization in the governing approval or user request.
- **No nested delegation:** Every shipped subagent profile denies `subagent` in its OpenCode v2 `permissions:` rules (`internal/skillsync/opencode_pack_test.go`).
- **No unmeasured savings claims:** Smaller briefs and bounded read-only children are mechanisms to limit redundant calls, not proof of measured token or cost savings (`opencode-pack/skills/agentic-sdd-orchestrate/references/context-and-cost.md`).

## Issue bounded delegation briefs

Pass a compact self-contained packet to the child instead of pasting a conversation transcript or full specification (`opencode-pack/skills/agentic-sdd-orchestrate/references/dispatch-contract.md`). Include these five fields:

| Field | What to specify |
|---|---|
| `task_id` | Stable unit ID such as `T02` or `fix-<slug>` |
| `authorized_scope` | Governing feature approval and batch ID (or direct-fix request), plus explicit exclusions (no commits, pushes, deploys, or next batches) |
| `allowed_paths` | Exact files, directories, or packages the child may read or modify |
| `expected_verification` | Exact commands to run and evidence to quote back, or checks that remain unrun |
| `stop_conditions` | Pre-existing failures outside scope, edits needed outside `allowed_paths`, unavailable models that alter privacy/cost, or unclear working-tree state |

When working inside an initialized execution run (`.agentic-sdd/runs/<run-id>/`), `asdx-core task brief --run ID --task T01` extracts one exact task and records its digest in `checkpoint.json` (`cmd/agentic-sdd/helptext.go`). A generated brief, run checkpoint, or review package is local observation only — it calls no model, creates no approval, and is not an OS sandbox. See [the execution reference](../reference/execution.md).

A synthetic fixture brief is available in `website/tests/examples/orchestration/bounded-brief.md` and walked through in [the specialists tutorial](../tutorials/specialists.md).

## Inspect child results instead of trusting claims

Every child returns the structured YAML schema in `opencode-pack/skills/agentic-sdd-orchestrate/references/dispatch-contract.md` (`status`, `scope_observed`, `changed_paths`, `findings`, `source_references`, `checks_run`, `unverified_claims`, `actual_model`, `questions_for_parent`).

Treat the return as a **claim** to verify before reporting (`opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`):

1. **Check the actual diff and scope:** Run `git status --short` and `git diff` to confirm `changed_paths` stays inside `allowed_paths` and matches `scope_observed`.
2. **Verify cited references:** Confirm every file path and line number in `findings` and `source_references` actually states what the child claimed.
3. **Require concrete check evidence:** Reject vague claims like "tests pass"; verify the exact command, package selection, exit code, and log output (or capture them via `asdx-core run check --run ID -- <command>`).
4. **Preserve `unverified_claims` and `actual_model`:** Never upgrade an unverified claim to verified, and report the model that actually ran rather than the configured model string.
5. **Stop at the human gate:** Synthesize one concise result for the human reviewer. A child verdict or single completion review (`Ready`) is a technical assessment, never combined approval or authorization for the next batch (`skills-files/agentic-sdd-router/references/workflow-policy.md`).

## Apply manual fallback and safe continuation

Written routing ladders in `opencode-pack/routing.md` are **policy, not an automatic failover engine**. Neither `asdx-core` nor `opencode-pack` automatically retries across models or providers (`opencode-pack/routing.md`, `opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`).

### Classify the failure before acting

| Failure category | How to respond |
|---|---|
| **Provider or transport error** | Read the actual error text, then follow the manual lane-bounded fallback rules below |
| **Wrong implementation or quality drift** | Compare against the brief and diff; stop and re-scope instead of switching models |
| **Test or command failure** | Record command, exit code, and full log; fix only if inside `allowed_paths` and `authorized_scope` |
| **Partial write or unknown tree state** | Stop writes immediately and inspect the working tree before resuming or reporting `blocked` |
| **Permission, billing, or data-policy denial** | Stop; treat as an authorization boundary, never a transport error to retry elsewhere |

### Manual fallback rules and lane limits

When a child fails from model unavailability, quota exhaustion, or repeated transient provider errors:

1. **Try each listed substitute at most once, in order, inside its lane** (`opencode-pack/routing.md`):
   - **Free-first ladder (`opencode/`):** `opencode/deepseek-v4.1-flash-free` -> `opencode/glm-5.3-flash-free` -> `opencode/mimo-v2.6-flash-free`. Moving onward to `opencode-go/` is `blocked` without separate explicit human authorization because Go models may draw on allowance or incur charges.
   - **Preserved 5-agent Claude/9Router exception:** applies only to `sdd-spec-9r`, `sdd-plan-9r`, `sdd-architecture-review-9r`, `sdd-code-review-9r`, and `sdd-implement-9r-2` (`9router/ag/claude-opus-5-5-high` or `claude-sonnet-5-5-high` -> `9router/ag/gemini-3.8-flash-high` -> `opencode-go/deepseek-v4.1-flash`). Specs, plans, and reviews produced by a fallback stay provisional.
   - **Beam cross-lane rules (`sdd-lightweight`, `sdd-review-verification-2`):** a `opencode/beam-free` failure never triggers an automatic `opencode-go/` switch; human authorization is required with explicit cost disclosure, and `sdd-review-verification-2` remains supplementary rather than the sole consequential reviewer.
   - **Optional OpenAI specialists (`sdd-openai-architect`, `sdd-openai-implement`):** stop as `blocked` on missing access or unauthorized spend; no silent cross-provider fallback.
2. **Verify compatibility before switching:** Check context window, output ceiling (for example, the four free Flash models are documented with 200,000 context / 32,000 output vs larger ceilings on Claude/Gemini/Go DeepSeek in `opencode-pack/routing.md`), and input modality (do not send images to a text-only model).
3. **Never retry task/permission errors as model failures:** Missing tools, invalid requests, and denied permissions must not be rerouted.

### Recover interrupted writes and compacted sessions

Before resuming any write task or continuing after session compaction (`opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`):

1. If an execution run exists, inspect `asdx-core run status --run <run-id>` first. A `stale` or `blocked` checkpoint stops execution until source changes, task brief digests, or the worktree are reconciled.
2. Inspect recorded tool outputs and check `git status --short` and `git diff` to see what was already written.
3. Preserve completed edits and patch only missing parts — never replay an entire write blindly. If state is ambiguous, stop and report `blocked`.

## Related pages

- [Specialists and routing tutorial](../tutorials/specialists.md) — guided walkthrough with inert brief, return, and `fleet.json` fixtures
- [Agent profile catalog](../reference/profiles.md) — source-derived profile, preset, adapter, and permission reference
- [Fleet reference](../reference/fleet.md) — schema-v1 `fleet.json` overrides, plans, and offline `fleet doctor` labels
- [CLI reference](../reference/cli.md) — exact flags and exit codes for `preview`, `apply`, `restore`, `fleet`, `run`, and `task`
- Related sitemap guides (integration-pending cross-links): `introduction/concepts`, `how-to/fleet`, `how-to/execution`, `how-to/updates-and-recovery`, `reference/tui`
