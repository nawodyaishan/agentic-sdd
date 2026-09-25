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

AI can generate changes faster than you can understand and review them. Agentic SDD organizes that work around your review capacity: agree on the feature, implement a manageable batch, verify it, then decide what happens next.

- **One main agent, manageable review.** Follow one implementation and one set of decisions at a time. The agent loads specialist guidance as needed; separate agents are used only when you request them.
- **Review the complete approach before coding.** Read the feature's requirements, design, and tasks together, then give one combined approval.
- **Control the pace.** Each implementation batch ends with verification and a review summary. You choose when the next batch begins.
- **Reuse your existing work.** Start from your SRS, technical specification, roadmap, and repository conventions. Read relevant sections and create supporting documents only when they serve a purpose.
- **Carry the same workflow across clients.** Keep a shared set of skills for Claude Code, Codex, and Antigravity CLI, with client-specific settings kept separate.

The goal is useful progress you can explain, verify, and maintain.

## The workflow

### 1. Select the next feature

Choose one coherent outcome from your existing project documents or request. Draft this feature in detail; leave future work in the roadmap.

A feature may span several sessions. Its implementation batches should be small enough for focused work and comfortable review.

### 2. Prepare three connected documents

The agent drafts these in order inside `specs/<number>-<feature>/`:

| Document | What you review |
| :--- | :--- |
| `spec.md` | Outcome, scope, exclusions, and acceptance criteria |
| `plan.md` | Technical approach, design decisions, relevant risks, specialist skills, and tools |
| `tasks.md` | Ordered tasks, dependencies, implementation batches, and verification |

The agent reconciles the three drafts and presents them together. **One combined human approval is required before implementation.**

### 3. Implement one batch

Authorize a selected batch. The main agent loads the specialist guidance needed for its tasks, implements the changes, and runs relevant checks.

Specialization does not require another agent: the same agent can use Go, React, Kubernetes, or Terraform guidance as the work requires. Tools are selected for concrete needs, and simple tasks may need no additional specialist.

### 4. Review the result and continue

The agent summarizes the changes, verification, deviations, and remaining issues, records `awaiting human review`, and stops.

You can request corrections or authorize the next batch. On resume, the agent checks the recorded state and current code before continuing. Passing tests alone do not start another batch.

**For smaller work:** clearly scoped fixes with understood consequences can go directly to implementation and verification, without feature documents.

**For reviews:** a review-only request produces findings without changing code, documents, or workflow status.

**For live operations:** approving infrastructure or migration code does not authorize applying it to real systems.

## The skill set

Start with **`agentic-sdd-router`** when beginning or resuming work. Use **`agentic-sdd-implement`** when you already know which approved batch or direct fix to execute.

All ten skills are manually invocable. You do not need to call every skill for every change.

| Skill | When to use it |
| :--- | :--- |
| `agentic-sdd-router` | Start, resume, or choose the appropriate next step |
| `agentic-sdd-bootstrap` | Add missing project pointers and conventions when requested |
| `agentic-sdd-spec` | Define a feature's outcome and acceptance criteria |
| `agentic-sdd-plan` | Design the approach and assign specialist guidance and tools |
| `agentic-sdd-tasks` | Turn the approach into tasks and reviewable batches |
| `agentic-sdd-implement` | Execute one authorized batch or direct fix |
| `agentic-sdd-verification-review` | Check the actual changes against requirements and evidence |
| `agentic-sdd-architecture-review` | Examine consequential design or operational decisions |
| `agentic-sdd-research-spec` | Resolve a specific question blocking progress |
| `agentic-sdd-drift-retro` | Address meaningful drift or capture a useful lesson |

Invoke the relevant skill through your client, then give it a concrete request:

- **Start:** "Use the relevant documents in `Docs/` to draft the next feature's spec, plan, and tasks. Stop for my combined review."
- **Implement:** "I approve this feature's documents. Implement batch B1, verify it, and stop for my review."
- **Resume:** "Check the current feature and batch state. Tell me the next action; preserve any pending review."
- **Direct fix:** "Fix this validation bug and run focused checks."
- **Review:** "Review this diff against the approved feature. Report findings only."

Requesting only a spec, plan, or review keeps the work limited to that request.

Shared workflow rules live in [`workflow-policy.md`](skills-files/agentic-sdd-router/references/workflow-policy.md), and specialist/tool selection lives in [`specialists.md`](skills-files/agentic-sdd-router/references/specialists.md). Install the skill directories together so these references remain available.

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

Before applying to your real home, `make docker-e2e` builds and exercises the CLI inside `golang:1.25-alpine` with the repository mounted read-only, no network, and a throwaway home. It checks preview, backup and install, a repeated apply, listing and restoring a backup, and an invalid command. It never touches your home directory. Docker must be running.

## CLI reference

| Invocation | Behavior |
| :--- | :--- |
| `agentic-sdd` / `agentic-sdd preview` | Print one line per skill per client — `install`, `replace + back up`, or `up to date` — then stop without writing. |
| `agentic-sdd apply` | Back up every skill it will replace, stage the new copies, then install. A second apply is a no-op. |
| `agentic-sdd version` | Print version, commit, build date, and Go version. `--version` works too. |
| `agentic-sdd backups` | List backups newest first: id, time, operation, tool version, source, and what each holds. Read-only. |
| `agentic-sdd restore [ID]` / `restore preview [ID]` | Show what restoring backup `ID` would change, without writing. With no `ID` on a terminal, list backups and prompt for a number. |
| `agentic-sdd restore ID --apply` / `restore apply ID` | Restore backup `ID`: back up whatever it's about to change into a new backup, then install the saved skills and remove any it freshly installed. Interactive selection asks `[y/N]` before writing. |
| `agentic-sdd help [preview\|apply\|backups\|restore]` | Show commands, flags, and defaults. `--help` and `-h` work too. |

| Flag | Meaning |
| :--- | :--- |
| `--repo PATH` | Repository containing `skills-files/` (default: skills embedded in the binary; pass this to source from a checkout instead) |
| `--home PATH` | Home directory holding the client skill directories (default: your home) |
| `--apply` | Legacy form of the `apply` command, or the writing flag for `restore`; cannot be combined with a named command |

Put flags after the command: `agentic-sdd preview --home /tmp/test-home`. Older flag-only scripts still work — no `--apply` previews, `--apply` installs.

Exit codes: **0** success, a completed preview or cancel, **2** unknown command, bad flag, stray argument, a malformed backup id, or `--apply` combined with a command, **1** filesystem/sync failure, a backup that fails validation (home mismatch, digest mismatch, missing), or a not-found id. Usage errors go to stderr; everything else goes to stdout. A preview that lists replacements is a success — it just didn't install them.

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

**Backups come first.** An apply (or restore) with changes creates a UTC-stamped directory: under `<repo>/backups` when `--repo` is set, or `<home>/.agentic-sdd/backups` when sourcing from the embedded skills (e.g. a Homebrew install):

```text
backups/20260924T201717.833575000Z/
├── manifest.json        # what this backup holds and where it came from
├── agents/agentic-sdd-plan/...
└── claude/agentic-sdd-plan/...
```

Only skills that already existed are copied there. A repository-local `backups/` directory is git-ignored because your previous skills may contain private content — keep or prune it on your own terms.

**The manifest records what happened, not just what's saved.** Alongside the original `created`/`source`/`targets`/`skills` fields, every backup written by this version records:

| Field | Meaning |
| :--- | :--- |
| `id` / `created_at` | The backup's directory name, and the same moment as an RFC 3339 timestamp |
| `operation` | `apply`, or `restore` (with `restored_from` naming the backup that was restored) |
| `tool` | The version, commit, build date, and Go version of the binary that wrote it |
| `home` / `backup_root` | The absolute home and backup root the operation ran against |
| `entries` | One record per changed skill: client, skill, destination path, and `replaced` (with a saved-copy path and a SHA-256 digest) or `installed` (nothing to save — it didn't exist before) |

A backup written by an older version of the CLI has none of these extra fields; `agentic-sdd backups` marks it `[legacy]` and restores it on a best-effort basis (see below). The digest is checked before every restore, so a truncated or hand-edited backup is refused rather than silently applied.

**Installation is staged.** Every replacement is built in a temporary directory beside its destination before any installed skill is touched; installs then happen as renames. If one fails partway, the CLI restores what it already moved and removes what it already installed, and the backup remains for manual recovery.

**Restoring a backup** follows the same contract. `agentic-sdd restore <ID>` previews; `--apply` (or `restore apply <ID>`) backs up whatever it's about to change — into a new backup you can restore to undo the restore itself — then puts back every skill that backup recorded as `replaced`, and removes every skill it recorded as `installed` (a fresh install that apply made, which restoring should undo too). Destinations always come from your current `--home`, never from paths recorded in the manifest, so a restore cannot write outside your four client skill directories, and it refuses a backup made for a different home. A `[legacy]` backup has no `installed` records, so restoring it only puts back the skills it actually saved — anything a later apply installed fresh is left as is. Either way, restoring is a no-op if nothing differs, and a failure partway through rolls back exactly like apply's does.

## Working on the skills

Edit the files in `skills-files/`, run `make preview` to confirm the intended clients pick them up, then `make apply`. Because `up to date` skills are skipped, iterating on one skill only rewrites that skill.

Keep shared rules in `workflow-policy.md` rather than repeating them in each `SKILL.md`, keep specialist and tool guidance in `specialists.md`, and keep machine-specific inventory and client configuration in `config/` — not in the portable skill text.

## Repository layout

```text
cmd/agentic-sdd/       CLI parsing, defaults, exit codes, interactive backup selection
internal/skillsync/    Planning, comparison, backup, staged install, rollback, restore
internal/version/      Build-time version metadata, injected via -ldflags at release
skillsfiles.go         go:embed of skills-files/, so an installed binary needs no checkout
skills-files/          The ten Agentic SDD skills and shared references
.goreleaser.yml        Darwin build, universal binary, Homebrew formula publishing
.github/workflows/     release.yml (tag-triggered publish), ci.yml (build/test/vet)
tests/                 Docker end-to-end scripts
config/                Local inventory and client overlay notes
ROADMAP.md             Product direction: Now / Next / Later, what we won't do, kill criteria
docs/                  Background research
audits/                Workflow audits
specs/                 Feature documents produced by the workflow itself
backups/               Git-ignored copies from changed installations
```

The CLI stays small; all filesystem behavior lives in `internal/skillsync`. `sync_test.go` covers backup and replacement, symlink and conflict refusals, and rollback; `manifest_test.go` covers the format 2 manifest and its digest; `restore_test.go` covers listing, backup validation (including every kind of malformed or tampered backup), and restore's preview/apply/rollback behavior. `cmd/agentic-sdd/main_test.go` covers help, usage errors, preview/apply compatibility, and every `backups`/`restore` form, including the interactive prompt. Run `make test` and `make vet` after changing the installer, and `make docker-e2e` before trusting an apply or a restore.

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
