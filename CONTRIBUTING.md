# Contributing

Thanks for improving Agentic SDD. Changes to skills and the installer are both welcome.

## Set up

1. Install Go 1.23 or newer and [Lefthook](https://lefthook.dev/install/).
2. Clone the repository and run `make hooks-install` once.
3. Run `make help` to see the available commands.

The pre-commit hook checks Go formatting, runs `make test` and `make vet`, and checks staged changes for whitespace errors. Run the same checks yourself before opening a pull request:

```sh
make test
make vet
git diff --check
```

## Change a skill

Edit the canonical copy in `skills-files/agentic-sdd-*/`. Keep its `SKILL.md` frontmatter and any relative references valid. Use `make preview` to see how the change would affect installed skills. `make apply` writes to your user skill directories and creates a local backup, so run it only when you intend to update your own installation.

## Change the Go installer

Keep CLI flags and exit behavior in `cmd/agentic-sdd/`. Put sync behavior in `internal/skillsync/`, with filesystem helpers in `files.go`. Add focused tests using temporary repositories and home directories; tests must never write into real user skill locations. Preserve preview as the default and back up every changed existing skill before replacement.

## Send a change

Keep pull requests focused. Explain the behavior you changed, how you checked it, and any effect on installed skills or backups. Update the README and Makefile when commands or paths change. Do not commit `backups/`, local Lefthook overrides, or personal skill content.
