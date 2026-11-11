# OpenCode orchestration pack (optional)

This directory is the optional, opt-in OpenCode pack delivered by the `agentic-sdd` CLI. It is separate from `skills-files/` (the 18 default core skills) and from `integrations/opencode/` (the legacy Feature 005/006/007 experimental harness resources). Nothing here is embedded in the core install and nothing is written until you pass `--with-opencode`.

## What it contains

```text
opencode-pack/
  skills/agentic-sdd-orchestrate/
    SKILL.md                            # main-agent delegation instructions
    references/
      dispatch-contract.md              # brief fields and child return schema
      failure-and-reconciliation.md     # manual fallback, interrupted writes, unverified claims
      context-and-cost.md               # D0-D4 / blocked classes, minimal context, honest cost
  agents/                               # 14 profiles installed by --with-opencode
    sdd-orchestrator.md                 # primary; uses the session model
    sdd-debug-diagnose.md
    sdd-test-design.md
    sdd-research-librarian.md
    sdd-docs-worker.md
    sdd-architecture-advisor.md
    sdd-visual-inspector.md
    sdd-openai-architect.md             # optional OpenAI GPT-6.1 Sol read-only advisor
    sdd-openai-implement.md             # optional OpenAI GPT-6.1 Terra authorized writer
    sdd-architecture-review-9r.md       # updated in place after backup
    sdd-code-review-9r.md               # updated in place after backup
    sdd-review-verification-1.md        # updated in place after backup
    sdd-review-verification-2.md        # updated in place after backup
    sdd-lightweight.md                  # updated in place after backup
  presets/free-first/                   # 4 profiles installed by --opencode-free-first
    sdd-implement-1.md
    sdd-implement-2.md
    sdd-test-implement-1.md
    sdd-test-implement-2.md
  adapters/                             # fleet-only render sources (Feature 014); never installed by --with-opencode
    sdd-spec-9r.md
    sdd-plan-9r.md
  routing.md                            # models, lanes, fallbacks, MCP access
  README.md
```

`opencode_pack.go` at the repository root embeds the whole tree with `//go:embed all:opencode-pack` so an installed binary needs no checkout.

## Install

```bash
agentic-sdd preview --with-opencode                       # default: show the plan, write nothing
agentic-sdd apply   --with-opencode                       # 18 core skills + orchestrate + 14 profiles
agentic-sdd preview --with-opencode --opencode-free-first  # also plan the 4 Free-first profiles
agentic-sdd apply   --with-opencode --opencode-free-first  # back up and replace them too
```

- The `agentic-sdd-orchestrate` skill is installed into the four standard client skill directories (`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`) so sibling links such as `../agentic-sdd-router/references/workflow-policy.md` resolve. It is never duplicated into `~/.config/opencode/skills/`, which would create duplicate skill IDs.
- The 14 agent profiles go to `<home>/.config/opencode/agents/`, derived from the resolved `--home` (not `XDG_CONFIG_HOME`).
- Existing copies of the five deny-by-default profiles (`sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1`, `sdd-review-verification-2`, `sdd-lightweight`) are backed up before replacement.
- `--opencode-free-first` additionally backs up and replaces `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1` and `sdd-test-implement-2`.
- `sdd-spec-9r`, `sdd-plan-9r`, `sdd-implement-9r-*`, `sdd-implement-free-*`, `vision`, `opencode.json` and `AGENTS.md` are not modified in any mode. The `adapters/` sources are fleet-only: `--with-opencode` never installs them, so the installed set stays 14 profiles.

## Preview, backup, restore

```bash
agentic-sdd                                               # preview of the core install
agentic-sdd backups                                       # list backups, including opencode-agents files
agentic-sdd restore                                       # preview the most recent restore
agentic-sdd restore <BACKUP_ID> --apply                   # restore skills and agent profiles
```

`apply` is a no-op when installed files already match, so a re-run writes nothing and creates no backup. A later default `agentic-sdd apply` (without `--with-opencode`) leaves installed opt-in files in place. Restoring a backup removes profiles that were freshly installed and restores profiles that were replaced; a tampered backup is refused.

The CLI does not edit `~/.config/opencode/AGENTS.md`. When `--opencode-free-first` is used and that file exists, `preview` and `apply` print a one-line notice saying so and pointing at `agentic-sdd restore <BACKUP_ID>`.

## Routing and models

See [routing.md](routing.md) for the pinned specialist models, the Free-first ladder, the preserved Claude/9Router fallback exception, the two Beam cross-lane rules, and the MCP access rules. Model identity, provider availability, quota and pricing are configuration facts, not guarantees: the parent must report the model that actually ran.

## Safety notes

- This pack changes instructions and permissions, not the operating-system sandbox. A prompt constraint is not filesystem isolation, and permission rules do not prevent symlink or path escapes on their own.
- `preview` is the default; only `apply` writes. Running `apply` against a real user home is a live operation with its own authorization requirement, separate from approving or developing this pack.
- Subagents are read-only by default here. Only `sdd-docs-worker` and the Free-first writer/test presets allow `edit`, and none of them may delegate further.
- A technical review verdict never replaces human approval.

## Verification

The pack is covered by the static test `TestOpenCodePackAssetsAndPermissions` in `internal/skillsync`, which checks both the embedded tree and the on-disk directory for required files, valid frontmatter, resolvable relative links, OpenCode v2 permission syntax, and the deny-by-default ordering rules. The fleet-only `adapters/` sources are covered by the Feature 014 resolver/render tests and by `TestWithOpenCodeExcludesFleetAdapters`, which pins their exclusion from the 009 install set.
