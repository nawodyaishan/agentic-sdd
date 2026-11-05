# Contributing

Thanks for improving Agentic SDD. Changes to skills and the installer are both welcome.

## Set up

1. Install Go 1.23 or newer and [Lefthook](https://lefthook.dev/install/).
2. Install Go Task, clone the repository and run `task hooks-install` once.
3. Run `task help` to see the available commands.

The pre-commit hook checks Go formatting, runs `task test` and `task vet`, and checks staged changes for whitespace errors. Run the same checks yourself before opening a pull request:

```sh
task test
task vet
git diff --check
```

## Change a skill

Edit the canonical copy in `skills-files/agentic-sdd-*/`. Keep its `SKILL.md` frontmatter and any relative references valid. Use `task preview` to see how the change would affect installed skills. `task apply` writes to your user skill directories and creates a local backup, so run it only when you intend to update your own installation.

## Change the Go installer

Keep CLI flags and exit behavior in `cmd/agentic-sdd/`. Put sync behavior in `internal/skillsync/`, with filesystem helpers in `files.go`. Add focused tests using temporary repositories and home directories; tests must never write into real user skill locations. Preserve preview as the default and back up every changed existing skill before replacement.

## Send a change

Keep pull requests focused. Explain the behavior you changed, how you checked it, and any effect on installed skills or backups. Update the README and Taskfile when commands or paths change. Do not commit `backups/`, local Lefthook overrides, or personal skill content.
