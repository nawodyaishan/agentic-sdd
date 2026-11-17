# Contributing

Thanks for improving Agentic SDD. Changes to skills and the installer are both welcome.

For the safe first-use path and three worked workflow examples, start with
[Getting started](docs/getting-started.md). Teaching is optional, not a contribution prerequisite.

## Product vision and design choices

Read the [project vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md)
before proposing a new command, agent, document set, directory or workflow step.
It records the observed problems, engineering principles and evidence gates used
to judge additions.

Contributions should explain the user value, the smallest useful change, how
success will be measured, and any migration, security or authority consequence.
The vision informs those decisions; it does not grant approval or override an
approved feature scope, a security requirement, or maintainer authorization.

## Set up

1. Install Go 1.23 or newer and [Lefthook](https://lefthook.dev/install/).
2. Install Go Task, clone the repository and run `task hooks-install` once.
3. Run `task help` to see the available commands.

The pre-commit hook checks Go formatting, runs `task test` and `task vet` on the 11 supported-core Go packages, and checks staged changes for whitespace errors. Run the same checks yourself before opening a pull request:

```sh
task test
task vet
git diff --check
```

### Supported-core verification and experimental task separation

Default `task test` and `task vet` run the 11 supported-core Go packages (`.`, `./internal/version`, `./internal/skillsync`, `./internal/harnesscheck`, `./internal/harnessevidence`, `./internal/appops`, `./internal/buildinfo`, `./internal/fleet`, `./internal/execution`, `./cmd/agentic-sdd`, `./cmd/sdd-check`). `task test-core` and `task vet-core` are retained as parity-enforced aliases (`sh tests/supported-core-go.sh test|vet`).

Experimental tools (such as the `opencode` CLI helper) are compiled out of default release and core builds via build tags (`!agentic_sdd_experimental`) on CLI entry points (`cmd/agentic-sdd/opencode_disabled.go`), severing the import edge to `internal/opencodeadapter` (`go list -deps ./cmd/agentic-sdd`) and returning exit 2 (`unavailable in this build`). The `internal/opencodeadapter` package itself is excluded from default verification by the explicit 11-package core selection. Their tasks live in the separate, non-default `Taskfile.experimental.yml` (`task --taskfile Taskfile.experimental.yml opencode-help`) and are never invoked by default tasks, pre-commit hooks, or release pipelines.

Portable OpenCode skill installation (`~/.agents/skills`) remains part of the supported core installer.

While blocked 005/006/007 feature validations remain excluded under the [temporary fence](AGENTS.md#temporary-008-validation-scope), the supported-core suite and trusted checks verify the active product. If an aggregate verification conflicts with an excluded validation, report it as blocked/not_run rather than weakening tests, hiding a failure, or bypassing pre-commit hooks.

## Change a skill

Edit the canonical copy in `skills-files/agentic-sdd-*/`. Keep its `SKILL.md` frontmatter and any relative references valid. Use `task preview` to see how the change would affect installed skills. `task apply` writes to your user skill directories and creates a local backup, so run it only when you intend to update your own installation.

Optional development-only text/fixture checks use Python's standard library:

```sh
python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py
```

These guard core instruction wording, scenario coverage and onboarding links;
they do not call a model or prove model compliance. Go package tests cover embedded
skill/reference distribution. See [fixture ownership](tests/harness/README.md#core-workflow-text-and-onboarding-checks).

## Change the Go installer

Keep CLI flags and exit behavior in `cmd/agentic-sdd/`. Put sync behavior in `internal/skillsync/`, with filesystem helpers in `files.go`. Add focused tests using temporary repositories and home directories; tests must never write into real user skill locations. Preserve preview as the default and back up every changed existing skill before replacement.

## Send a change

Keep pull requests focused. Explain the behavior you changed, how you checked it, and any effect on installed skills or backups. Update the README and Taskfile when commands or paths change. Do not commit `backups/`, local Lefthook overrides, or personal skill content.

## Public website docs

Contributor, compatibility, and migration guidance for the public site lives in
the website source and is owned by the docs lanes:

- `website/src/content/docs/contributing/docs-and-changes.md` — docs style, impact review, and page/skill/profile ownership.
- `website/src/content/docs/releases/compatibility.md` — supported checkout, release, platform, and client combinations.
- `website/src/content/docs/releases/migration.md` — the sole canonical/legacy/state migration table.

Those pages are the source of truth for website content; this file remains the
repository entry point.

## Reporting bugs and security concerns

Use [repository issues](https://github.com/nawodyaishan/agentic-sdd/issues) for
non-sensitive bug reports with reproduction steps and sanitized expected/actual results.
Do not post credentials, private backups, raw session traces or exploit details.

For sensitive reports, use the repository's private vulnerability reporting option
if available. Otherwise open a public issue requesting a private reporting channel,
without disclosing the vulnerability. No dedicated security email or response-time guarantee
is documented here; do not assume a private channel exists or send secrets publicly.
