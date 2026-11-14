---
title: First five minutes
description: Run version, help and an isolated read-only preview with a disposable home directory and no real-home writes or provider keys.
---

Verify that `asdx-core` works and see exactly what a core skill installation would do—without installing anything into your real home directory, configuring any AI provider keys, or spawning any subagents.

## Before you start

- **Prerequisites:** `asdx-core` on your `PATH` (from [Installation](./installation.md)), or a trusted source checkout with **Go 1.23 or newer**.
- **What stays untouched:** Your real `$HOME`, existing client directories, provider credentials, and network. None of the commands on this page make network calls or invoke an AI model.
- **Boundary note:** `--home` and temporary directories isolate where the installer reads and writes; they are not an OS security sandbox. Run only binaries or checkouts you trust.

## Step 1: Create a disposable home

Create a temporary directory to stand in for `$HOME`:

```sh
demo_home=$(mktemp -d "${TMPDIR:-/tmp}/asdx-demo.XXXXXX")
```

**Effect:** Creates one empty directory under your temporary folder. Nothing in your real home directory is modified.

## Step 2: Run version and help

Check the binary identity and the `preview` help block:

```sh
asdx-core version
asdx-core help preview
```

If you are running directly from a source checkout without installing a binary first, use Go's offline flags instead (note that `go run` writes to your Go build cache):

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go run ./cmd/agentic-sdd version
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go run ./cmd/agentic-sdd help preview
```

**Expected observations:**

- `asdx-core version` exits `0` and prints the CLI version, commit, build date, and Go version. Note that `version` accepts no flags or extra arguments (`asdx-core version --help` exits `2`; use `asdx-core help version`).
- `asdx-core help preview` exits `0`, prints `Writes files: no (read-only)`, and then prints the top-level usage block, whose `Flags:` list is `--repo`, `--home`, `--apply`, `--version`, and `-h/--help`.

## Step 3: Preview without installing anything

Run `preview` targeting the disposable home:

```sh
asdx-core preview --home "$demo_home"
```

Bare `asdx-core --home "$demo_home"` behaves identically because `asdx-core` defaults to `preview` and never opens the terminal UI.

**Effect:** Read-only. Reads the 18 core skills embedded in the binary (`skills-files/`) and inspects the four client directories under `"$demo_home"` (`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`). Creates **no** directories, skills, backups, or logs.

**Illustrative output (not a live capture from your machine):**

```text
install agents/agentic-sdd-architecture-review
install agents/agentic-sdd-bootstrap
...
install agy/agentic-sdd-writing-for-agents
Preview only. Run with apply (or --apply) to install.

Summary (preview — nothing was written)
  agents           18 to install, 0 to replace, 0 up to date
  codex            18 to install, 0 to replace, 0 up to date
  claude           18 to install, 0 to replace, 0 up to date
  agy              18 to install, 0 to replace, 0 up to date
  Total: 72 to install, 0 to replace (backed up first), 0 up to date — nothing was written.
  Next: asdx-core apply --home /tmp/asdx-demo.XXXXXX
```

When stdout is piped or redirected (not a TTY), the `Summary (...)` block is omitted so scripts see only the line-by-line plan and the final `Preview only.` line.

> **What an empty home does not predict:** Previewing an empty directory shows a fresh install (72 `install` lines). It does **not** predict whether skills already exist or would be backed up and replaced in your real `$HOME`.

## Step 4: Verify no writes and clean up

Remove the temporary directory with `rmdir`:

```sh
rmdir "$demo_home"
```

**Expected observation:** `rmdir` exits `0` only if `"$demo_home"` is completely empty, confirming that `preview` wrote nothing inside it.

## Stop or continue

You can **stop here**—nothing was installed on your machine, and no cleanup beyond `rmdir` is needed.

When you are ready to go further:

- Choose between CLI core-only, the TUI, and optional OpenCode orchestration in [Choose your setup](./choose-setup.md).
- Install skills to your real home or pick specific clients in [Install and manage skills](../how-to/skills.md).
- Walk through a small code change without feature paperwork in [Direct fix tutorial](../tutorials/direct-fix.md), or the spec/plan/tasks approval loop in [First feature tutorial](../tutorials/first-feature.md).
- Look up every subcommand and flag in the [CLI reference](../reference/cli.md).
