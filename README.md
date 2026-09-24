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
make help           # see every command
make preview        # inspect installs and replacements; changes nothing
make docker-e2e     # test preview and apply in an isolated Docker container
make apply          # save existing skills, then install the new copies
make test           # run the safety and behavior tests
make vet            # run Go static analysis
make hooks-install  # install the pre-commit hook
```

Prefer Go directly? Run `go run ./cmd/agentic-sdd preview` to preview and `go run ./cmd/agentic-sdd apply` to install. The default is always a preview, so an invocation without a command still previews. Existing `--apply` scripts remain supported. Run `go run ./cmd/agentic-sdd --help` for the full CLI help.

Before applying to your real home, run `make docker-e2e`. This builds and tests the CLI inside a Go container using a temporary repository and home. It checks preview, backup and install, repeated apply, and invalid input; it never mounts your home. Docker must be running and the `golang:1.25-alpine` image available.

## CLI commands

| Command | Behavior |
| :--- | :--- |
| `agentic-sdd` or `agentic-sdd preview` | Show each install, replacement, or up-to-date skill without writing files. |
| `agentic-sdd apply` | Back up changed existing skills and install only the changes. A second apply is a no-op. |
| `agentic-sdd help` or `agentic-sdd --help` | Show commands, flags, defaults, and safety behavior. `preview --help` and `apply --help` also work. |

Use `--repo PATH` to choose the repository and `--home PATH` to choose the destination home. Put flags after a named command, as in `agentic-sdd preview --home /tmp/test-home`. For existing scripts, flag-only invocations still work: no `--apply` means preview, and `--apply` means install. Do not combine `--apply` with a named command.

The CLI exits with status `0` on success or help, `2` for invalid commands or flags, and `1` for filesystem or sync failures. Usage errors go to stderr. A preview that lists replacements is still successful; it does not install them.

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

A feature is drafted as `spec.md`, `plan.md` and `tasks.md` in that order, presented together for **one combined human approval**. Implementation then runs **one batch at a time**, stopping for human review after each. Small, clearly scoped fixes skip the feature documents entirely.

| Skill | Role |
| :--- | :--- |
| `agentic-sdd-router` | Route intake, feature work and direct fixes through the workflow |
| `agentic-sdd-bootstrap` | Establish the initial project pointers and conventions |
| `agentic-sdd-research-spec` | Resolve a specific fact blocking a draft or a fix |
| `agentic-sdd-spec` | Draft the feature outcome, scope and acceptance criteria |
| `agentic-sdd-architecture-review` | Review consequential architecture and operational risks |
| `agentic-sdd-plan` | Draft the approach, specialists and tools from the spec draft |
| `agentic-sdd-tasks` | Draft ordered tasks and explicit execution batches |
| `agentic-sdd-implement` | Implement one approved batch, or one direct fix |
| `agentic-sdd-verification-review` | Verify a batch or fix against acceptance criteria |
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
go run ./cmd/agentic-sdd apply --repo /path/to/agentic-sdd --home /path/to/test-home
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

## Contributing and license

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, checks, and pull request guidance. The pre-commit hook is configured in `lefthook.yml` and installed locally with `make hooks-install`. This project is available under the [MIT License](LICENSE).
