# Agentic SDD

[![Release](https://img.shields.io/github/v/release/nawodyaishan/agentic-sdd?label=release)](https://github.com/nawodyaishan/agentic-sdd/releases/latest)
[![Homebrew](https://img.shields.io/badge/homebrew-nawodyaishan%2Ftap%2Fagentic--sdd-fbb040)](https://github.com/nawodyaishan/homebrew-tap)
[![Go Report](https://img.shields.io/badge/go-1.23%2B-00ADD8)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**A spec-driven development workflow for coding agents — one feature, one human approval, one batch at a time — kept in sync across Codex, Claude Code, and Antigravity CLI by a small Go CLI.**

If you've ever had an agent "helpfully" finish a whole feature you never approved, or drift three files past what you asked for, this is the guardrail: agents draft, you approve once, agents implement in reviewable batches, you stay the one who says go.

```sh
brew install nawodyaishan/tap/agentic-sdd
agentic-sdd apply   # installs the skills into every client you have
```

Two things live here:

- [`skills-files/`](skills-files/) — ten skills that define the workflow. This is the product.
- `cmd/` + `internal/` — a single-purpose installer that copies those skills into your client skill directories, backing up whatever it replaces.

```text
skills-files/  ── preview ── back up ── install
                                    ├── ~/.agents/skills
                                    ├── ~/.codex/skills
                                    ├── ~/.claude/skills
                                    └── ~/.gemini/antigravity-cli/skills
```

---

## Why

Most agent workflows are either no process (freeform prompting, scope creep, silent drift) or too much process (constitutions, templates, a status report after every command). Agentic SDD picks a narrow middle: draft everything up front, get **one** human decision, then implement in small batches that each stop for review. It doesn't replace your judgment — it makes sure the agent actually waits for it.

- **You're using Claude Code, Codex, or Antigravity CLI** and want the same review discipline in all of them, not a different ad-hoc process per client.
- **You've been burned by an agent that kept going** past what you approved — batches exist so a green test run is never mistaken for permission to continue.
- **You want spec-driven development without the ceremony** — `spec.md` → `plan.md` → `tasks.md`, one combined approval, no separate sign-off per document, no constitution to maintain.
- **You maintain multiple repos** and want the same discipline everywhere without copy-pasting instructions — `agentic-sdd apply` keeps one canonical copy in sync.

## The workflow

The design goal is **manageable human review**: you see complete, reconciled work at a small number of decision points instead of approving fragments or waking up to a finished feature you never sanctioned.

**One feature, one approval.** A feature is one coherent outcome. The agent drafts `spec.md`, then `plan.md`, then `tasks.md` — each using the previous draft as input, no waiting in between — reconciles them, and presents all three together for **one combined human approval**. Nothing is implemented before that.

```text
spec.md ──▶ plan.md ──▶ tasks.md ──▶ [ combined human approval ]
                                              │
                                              ▼
                                     batch 1 ──▶ verify ──▶ [ your review ] ──▶ batch 2 ─ ...
```

**One batch at a time.** `tasks.md` groups tasks into explicit execution batches — batch ID, task IDs, outcome, verification, state, next action. After approval *and* your authorization, the agent runs exactly one batch, verifies it, records `awaiting human review`, and stops. A green test run is not permission to continue. On resume, an approved feature and a passing batch do not start the next one; you do.

**A feature may span several sessions.** Batches are sized for focused execution and comfortable review; the feature is not split just because it holds several dependent tasks.

**Direct fixes skip all of it.** A change with clear scope, understood consequences, and meaningful verification goes straight to implement-and-verify — no `specs/` directory, no plan assignment, no batch state. Production code does not disqualify a fix; consequence does. Permissions, data integrity, public contracts, migrations, and live-system operations need planning and their own authorization.

**Approval is not execution authority.** Approving Terraform or migration code never authorizes applying it.

**Review-only means read-only.** Ask for a review and you get findings plus a recommended next action — no edits to code, documents, approval records, task status, or batch state.

**Specialists are loaded, not name-dropped.** `plan.md` assigns real installed skills and the tools the work needs; `tasks.md` records only the exceptions that override those defaults. At implementation the agent loads the guidance the current task actually needs. "No additional specialist needed" is a valid, explicit answer. Unavailable skills and tools are reported, never silently substituted. One main agent does the work; subagents only when you ask.

## The skill set

Ten skills, all manually invocable. Start with the router if you want help choosing a phase — or call any skill directly. Direct invocation and router-driven execution follow identical rules.

| Skill | Role |
| :--- | :--- |
| `agentic-sdd-router` | Route intake, feature work, and direct fixes; resume at the recorded state |
| `agentic-sdd-bootstrap` | Establish project pointers and conventions, on request only |
| `agentic-sdd-spec` | Draft the outcome, scope, exclusions, and acceptance criteria |
| `agentic-sdd-plan` | Draft the approach, design decisions, risks, specialists, and tools |
| `agentic-sdd-tasks` | Draft ordered tasks, dependencies, and execution batches |
| `agentic-sdd-implement` | Implement one approved batch, or one direct fix |
| `agentic-sdd-verification-review` | Verify a batch or fix against acceptance criteria |
| `agentic-sdd-architecture-review` | Review consequential architecture and operational risk |
| `agentic-sdd-research-spec` | Resolve a specific fact blocking a draft or a fix |
| `agentic-sdd-drift-retro` | Handle material drift; capture lessons worth keeping |

Shared rules live once in [`agentic-sdd-router/references/workflow-policy.md`](skills-files/agentic-sdd-router/references/workflow-policy.md); specialist and tool selection lives in [`specialists.md`](skills-files/agentic-sdd-router/references/specialists.md). Every skill links to both, so install the directories together.

The skills reference your repository's own documents — commonly `Docs/SRS.md`, `Docs/High Level Spec.md`, and `Docs/Tasks.md`, or whatever your equivalents are called — and feature documents in `specs/<nnn-slug>/`. They create no constitutions, templates, or routine status reports.

## Install

```sh
brew install nawodyaishan/tap/agentic-sdd
```

The published binary embeds `skills-files/`, so `agentic-sdd preview`/`apply` work from any directory with no repository checkout needed. Run `agentic-sdd version` to check what you have installed.

## Quick start

Requires **Go 1.23+**. `make` is optional.

```sh
make help           # list every command
make preview        # show what would change; writes nothing
make docker-e2e     # exercise preview and apply in an isolated container
make apply          # back up existing skills, then install
make test           # run the safety and behavior tests
make vet            # run go vet
make hooks-install  # install the Lefthook pre-commit hook
```

Preview is always the default, so a bare invocation is safe. Prefer Go directly:

```sh
go run ./cmd/agentic-sdd                    # preview
go run ./cmd/agentic-sdd apply              # install
go run ./cmd/agentic-sdd --help             # full CLI help
```

Before applying to your real home, `make docker-e2e` builds and exercises the CLI inside `golang:1.25-alpine` with the repository mounted read-only, no network, and a throwaway home. It checks preview, backup and install, a repeated apply, and an invalid command. It never touches your home directory. Docker must be running.

## CLI reference

| Invocation | Behavior |
| :--- | :--- |
| `agentic-sdd` / `agentic-sdd preview` | Print one line per skill per client — `install`, `replace + back up`, or `up to date` — then stop without writing. |
| `agentic-sdd apply` | Back up every skill it will replace, stage the new copies, then install. A second apply is a no-op. |
| `agentic-sdd version` | Print version, commit, build date, and Go version. `--version` works too. |
| `agentic-sdd help [preview\|apply]` | Show commands, flags, and defaults. `--help` and `-h` work too. |

| Flag | Meaning |
| :--- | :--- |
| `--repo PATH` | Repository containing `skills-files/` (default: skills embedded in the binary; pass this to source from a checkout instead) |
| `--home PATH` | Home directory holding the client skill directories (default: your home) |
| `--apply` | Legacy form of the `apply` command; cannot be combined with a named command |

Put flags after the command: `agentic-sdd preview --home /tmp/test-home`. Older flag-only scripts still work — no `--apply` previews, `--apply` installs.

Exit codes: **0** success or help, **2** unknown command, bad flag, stray argument, or `--apply` combined with a command, **1** filesystem or sync failure. Usage errors go to stderr; everything else goes to stdout. A preview that lists replacements is a success — it just didn't install them.

## Where the skills go

| Client | Skill directory |
| :--- | :--- |
| Shared agents | `~/.agents/skills` |
| Codex | `~/.codex/skills` |
| Claude Code | `~/.claude/skills` |
| Antigravity CLI (`agy`) | `~/.gemini/antigravity-cli/skills` |

All four are written every run. The installer selects only directories named `agentic-sdd-*` and copies **the complete directory**, so `references/` and any other supporting files travel with `SKILL.md`. Every other skill you have is left untouched.

These destinations follow the [Codex](https://developers.openai.com/codex/skills), [Claude Code](https://code.claude.com/docs/en/skills.md), and [Antigravity](https://antigravity.google/docs/skills/) documentation. Antigravity IDE's legacy `~/.gemini/antigravity/skills` is a different directory and is not used here.

Client-specific invocation settings stay out of the portable skill text; local notes live in `config/`.

## Safety model

Preview and apply plan identically, so a preview tells you exactly what an apply would do.

**Before writing anything**, the installer walks both the source and destination skill trees and refuses symlinks, special files, a non-directory where a skill directory belongs, a non-directory component anywhere between the destination and your home, and a source skill without a regular `SKILL.md`. It stops if no `agentic-sdd-*` skill is found at all.

**Change detection is exact**: directory structure, file permissions, and SHA-256 of every file. Identical trees are reported `up to date`, are not backed up, and are not rewritten. When nothing differs, apply prints `All skills are up to date.` and creates no backup.

**Backups come first.** An apply with changes creates a UTC-stamped directory: under `<repo>/backups` when `--repo` is set, or `<home>/.agentic-sdd/backups` when sourcing from the embedded skills (e.g. a Homebrew install):

```text
backups/20260924T201717.833575000Z/
├── manifest.json        # created, source, target directories, skill names
├── agents/agentic-sdd-plan/...
└── claude/agentic-sdd-plan/...
```

Only skills that already existed are copied there. A repository-local `backups/` directory is git-ignored because your previous skills may contain private content — keep or prune it on your own terms.

**Installation is staged.** Every replacement is built in a temporary directory beside its destination before any installed skill is touched; installs then happen as renames. If one fails partway, the CLI restores what it already moved and removes what it already installed, and the backup remains for manual recovery.

## Working on the skills

Edit the files in `skills-files/`, run `make preview` to confirm the intended clients pick them up, then `make apply`. Because `up to date` skills are skipped, iterating on one skill only rewrites that skill.

Keep shared rules in `workflow-policy.md` rather than repeating them in each `SKILL.md`, keep specialist and tool guidance in `specialists.md`, and keep machine-specific inventory and client configuration in `config/` — not in the portable skill text.

## Repository layout

```text
cmd/agentic-sdd/       CLI parsing, defaults, exit codes
internal/skillsync/    Planning, comparison, backup, staged install, rollback
internal/version/      Build-time version metadata, injected via -ldflags at release
skillsfiles.go         go:embed of skills-files/, so an installed binary needs no checkout
skills-files/          The ten Agentic SDD skills and shared references
.goreleaser.yml        Darwin build, universal binary, Homebrew formula publishing
.github/workflows/     release.yml (tag-triggered publish), ci.yml (build/test/vet)
tests/                 Docker end-to-end scripts
config/                Local inventory and client overlay notes
docs/                  Background research
audits/                Workflow audits
specs/                 Feature documents produced by the workflow itself
backups/               Git-ignored copies from changed installations
```

The CLI stays small; all filesystem behavior lives in `internal/skillsync`. `internal/skillsync/sync_test.go` covers backup and replacement, symlink and conflict refusals, and rollback; `cmd/agentic-sdd/main_test.go` covers help, usage errors, and preview/apply compatibility. Run `make test` and `make vet` after changing the installer, and `make docker-e2e` before trusting an apply.

## Releasing

Releases are built by [GoReleaser](https://goreleaser.com) from `.goreleaser.yml` and published to `nawodyaishan/homebrew-tap` by `.github/workflows/release.yml` on every `v*` tag push:

```sh
make verify              # mod-verify, tidy-check, vet, test, build-darwin
make tag V=vX.Y.Z MSG="release message"
git push origin vX.Y.Z   # triggers .github/workflows/release.yml
```

`make release` runs a local snapshot build (`goreleaser release --snapshot --clean`, no tag or publish required) to check the formula and archive output before tagging.

## Contributing and license

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, checks, and pull request guidance. The pre-commit hook is configured in `lefthook.yml` and installed with `make hooks-install`. Available under the [MIT License](LICENSE).

If this matches how you already want to work with agents, `brew install nawodyaishan/tap/agentic-sdd` and try it on your next feature. Issues, skill improvements, and reports of where the workflow gets in your way are all welcome — [open an issue](https://github.com/nawodyaishan/agentic-sdd/issues) or a PR.
