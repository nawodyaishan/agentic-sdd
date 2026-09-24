# CLI quality and usability

## Outcome and sources

Make the Go skill installer easier to understand and maintain without changing its installation safety contract. Sources: `README.md` (Get started, Backups and safe replacement, Use another home or repository), `AGENTS.md` (Go layout, Safe changes, Verification), and the existing CLI and sync tests. This repository has no top-level product SRS or roadmap.

## Scope

- Add explicit `preview` and `apply` commands and useful top-level and command help. Keep an empty command line as preview, and retain `--apply`, `--repo`, and `--home` for existing scripts.
- Report invalid commands, extra arguments, and invalid flags clearly with a nonzero exit status. Help exits successfully; operational failures remain distinct from usage failures.
- Simplify the Go code so argument handling and sync planning are independently readable and testable, using the standard library and existing package layout.
- Update README and Makefile commands to match the CLI.
- Add a Docker container based end to end test that exercises preview, apply, repeat apply, backup, and invalid command behavior against an isolated temporary repository and home. Make it runnable before anyone uses apply on their real home.

## Safety and compatibility

- Preview remains read-only and the default. Only the explicit apply command or legacy `--apply` installs skills.
- An apply backs up every existing skill that it replaces before touching destinations. Identical skills create no new backup. Unrelated skills stay untouched.
- Managed trees continue to reject symlinks and special files, and failures preserve the existing rollback behavior.
- Existing `--repo` and `--home` isolated-home workflows remain supported.

## Acceptance

1. `agentic-sdd`, `agentic-sdd preview`, and legacy flag-only invocations show the same planned changes without modifying target skill directories or creating backups.
2. `agentic-sdd apply` and `agentic-sdd --apply` perform the same safe installation. Repeating either after a successful apply is a no-op.
3. `--help`, `help`, and command help explain commands, flags, safety, and defaults. Invalid input gives actionable stderr output and a usage error status.
4. Tests cover command parsing, exit behavior, and isolated-home sync behavior. A documented Docker end to end command checks a built CLI in a container with a temporary home. `make test`, `make vet`, and `git diff --check` pass.

## Outside scope

Skill content changes, destination selection, new third-party CLI frameworks, live installation to the user's home, and changes to backup format.

## Approval

Decision: **approved for implementation**, including the Docker end to end addition. Evidence: the user's "approve" and subsequent "before apply add docker container based e2e tests also" / "to spec" messages in this conversation. Scope: this spec and its linked plan/tasks, with Docker testing before any real-home apply. No real-home apply is authorized.
