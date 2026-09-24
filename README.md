# Agentic SDD · skill sync

**One source for the Agentic SDD skills you use across Codex, Claude Code, and Antigravity CLI.**

Edit a skill in [`skills-files/`](skills-files/), preview the changes, then install it everywhere with one command. Existing copies are saved inside this repository before anything is replaced.

```text
skills-files/  ── preview ── back up ── install
                                    ├── ~/.agents/skills
                                    ├── ~/.codex/skills
                                    ├── ~/.claude/skills
                                    └── ~/.gemini/antigravity-cli/skills
```

## Get started

Requires **Go 1.23+**. `make` is optional.

```sh
cd agentic-sdd
make help       # see every command
make preview    # inspect installs and replacements; changes nothing
make apply      # save existing skills, then install the new copies
make test       # run the safety and behavior tests
```

Prefer Go directly? Run `go run ./cmd/agentic-sdd` to preview and `go run ./cmd/agentic-sdd --apply` to install. The default is always a preview.

## Where the skills go

| Client | User skill directory | Purpose |
| :--- | :--- | :--- |
| Shared agents | `~/.agents/skills` | Codex user skills and shared agent discovery |
| Codex local | `~/.codex/skills` | Existing local Codex catalog |
| Claude Code | `~/.claude/skills` | Personal Claude skills |
| Antigravity CLI (`agy`) | `~/.gemini/antigravity-cli/skills` | Global CLI skills |

The installer copies **the complete skill directory**, so a skill's `references/` and other supporting files travel with its `SKILL.md`. It selects directories named `agentic-sdd-*` and leaves all other skills alone.

These destinations follow the [Codex](https://developers.openai.com/codex/skills), [Claude Code](https://code.claude.com/docs/en/skills.md), and [Antigravity](https://antigravity.google/docs/skills/) skill documentation. Antigravity IDE's legacy `~/.gemini/antigravity/skills` directory is separate from the CLI directory used here.

## The skill set

The ten skills cover a bounded SDD workflow. Start with the router when you want help choosing the right phase.

| Skill | Role |
| :--- | :--- |
| `agentic-sdd-router` | Route intake and work packets through the workflow |
| `agentic-sdd-bootstrap` | Establish the initial project pointers and conventions |
| `agentic-sdd-research-spec` | Resolve a specific fact blocking a spec or plan |
| `agentic-sdd-spec` | Draft and approve a bounded packet spec |
| `agentic-sdd-architecture-review` | Review consequential architecture and operational risks |
| `agentic-sdd-plan` | Plan a packet from an approved spec |
| `agentic-sdd-tasks` | Turn an approved plan into execution tasks |
| `agentic-sdd-implement` | Implement one approved task |
| `agentic-sdd-verification-review` | Check implementation against the approved packet |
| `agentic-sdd-drift-retro` | Handle material drift and capture useful lessons |

## Backups and safe replacement

An apply with changes creates a UTC timestamped directory such as:

```text
backups/20260924T201717.833575000Z/
├── manifest.json
├── agents/agentic-sdd-plan/...
└── claude/agentic-sdd-plan/...
```

Only skills that already existed appear in the backup. The `manifest.json` records the source, target directories, and skill names. `backups/` is in `.gitignore` because previous skills may contain personal content; keep or archive it according to your own retention needs.

If every destination already matches the source, `make apply` reports that the skills are up to date and creates no backup. Only changed or missing skills are installed.

Before installation, the program checks source and destination skill trees, rejects symlinks and special files, and stages every new copy. It then replaces matching skill directories and attempts to restore earlier copies if a later replacement fails. A backup remains available for manual recovery.

## Use another home or repository

The program accepts `--home` and `--repo` for an isolated installation or a different source checkout:

```sh
go run ./cmd/agentic-sdd --repo /path/to/agentic-sdd --home /path/to/test-home
go run ./cmd/agentic-sdd --repo /path/to/agentic-sdd --home /path/to/test-home --apply
```

## Repository layout

```text
cmd/agentic-sdd/       CLI flags, defaults, and error reporting
internal/skillsync/    Sync workflow, filesystem helpers, and tests
skills-files/          Canonical Agentic SDD skill directories
config/                Local inventory and client overlay notes
audits/                Workflow audits
backups/               Local, Git-ignored copies from changed installations
```

The CLI stays small; filesystem behavior lives in `internal/skillsync`. Run `make test` after changing the installer.
