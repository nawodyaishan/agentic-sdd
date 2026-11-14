---
title: Installation
description: Prerequisites, platform-qualified installation methods, write and network effects, and read-only verification.
---

Install `asdx-core` (the CLI) and `asdx` (the terminal UI) for your platform, verify that the binaries work, and understand what each step writes or fetches over the network.

Installing the binaries does **not** install any skills or modify your coding client directories. Skill installation is a separate, preview-first step in [First five minutes](./first-five-minutes.md) or [Install and manage skills](../how-to/skills.md).

## Preview before installing skills

`asdx-core preview` shows exactly which files an `apply` would create or replace, and writes nothing. Run the preview before installing anything, then continue with the worked path in [First five minutes](./first-five-minutes.md).

## Prerequisites

- **macOS Homebrew cask:** [Homebrew](https://brew.sh) on macOS (`darwin`).
- **Source build — CLI only (`asdx-core`):** **Go 1.23 or newer** (`go.mod`). The CLI uses only the Go standard library and requires no external Go module downloads.
- **Source build — paired CLI and TUI (`asdx` + `asdx-core`):** **Go 1.26 or newer** (`experiments/tui/go.mod`), `git`, `task` ([Task](https://taskfile.dev)), and `sha256sum` or `shasum` (`Taskfile.experimental.yml`).
- **Terminal requirements:** `asdx` and `asdx-core guide` require real interactive stdin and stdout terminals; piped or `/dev/null` stdin is rejected so scripted runs stay on `asdx-core`. `asdx-core tui` additionally rejects `AGENTIC_SDD_FORCE_INTERACTIVE=1` with exit `2`; `asdx` itself only checks for real TTYs. The TUI adapts to width (`≥110` sidebar, `80–109` compact navigation, `60–79` single navigation, sidebar again below `60`, resize prompt below `40×8`) and supports `NO_COLOR`, `AGENTIC_SDD_ASCII=1`, and `AGENTIC_SDD_REDUCED_MOTION=1`. See the [TUI reference](../reference/tui.md).
- **Supported coding clients:** See [Supported clients](../introduction/clients.md) for the four skill destinations and optional OpenCode agent profile directory.

## Platform support

| Platform | Distribution status | Qualification |
|---|---|---|
| **macOS (`darwin`)** | Homebrew cask (`nawodyaishan/tap/agentic-sdd`) and source build | Public `v0.11.0` tag, release metadata (`agentic-sdd_0.11.0_darwin_all.tar.gz`, SHA-256 verified in metadata), and tap cask declaration are verified. Archive contents, binary execution, and live Homebrew install/upgrade on macOS have **not** been independently verified in this documentation baseline. |
| **Linux** | Source build only | No prebuilt Linux binary, `.deb`, `.rpm`, or Homebrew Linux bottle is published (`.goreleaser.yml` targets `darwin` only). Build from source using the steps below. |
| **Windows** | Unsupported / unknown | Release builds exclude Windows; filesystem permissions, symlink checks, and terminal behavior are not tested or supported. |

## Install on macOS (Homebrew cask)

```sh
brew install --cask nawodyaishan/tap/agentic-sdd
```

- **Network effect:** Homebrew downloads the cask definition from `nawodyaishan/homebrew-tap` and fetches `agentic-sdd_<version>_darwin_all.tar.gz` from GitHub Releases over HTTPS.
- **Disk writes:** Installs and links four binaries into your Homebrew prefix (`asdx`, `asdx-core`, `agentic-sdd`, `agentic-sdd-tui`), and runs the cask post-install hook (`/usr/bin/xattr -dr com.apple.quarantine` on those four unsigned binaries, defined in `.goreleaser.yml`).
- **Untouched:** Does not touch `~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`, `~/.config/opencode/agents`, or `~/.agentic-sdd/`.

If you previously installed an older Homebrew **formula** named `agentic-sdd`, see [Updates and recovery](../how-to/updates-and-recovery.md) for formula-to-cask upgrade notes.

## Build from source (Linux and macOS)

Clone the repository:

```sh
git clone https://github.com/nawodyaishan/agentic-sdd.git
cd agentic-sdd
```

### Paired CLI and TUI (recommended when using `asdx`)

With **Go 1.26+**, `git`, `task`, and `sha256sum` or `shasum` installed:

```sh
task -t Taskfile.experimental.yml tui:pair
```

- **Network effect:** `git clone` fetches the repository; the first build inside `experiments/tui` fetches its Go module dependencies (`charm.land/*`, `github.com/charmbracelet/*`) unless they are already in your Go module cache (`GOPROXY=off` fails if uncached).
- **Disk writes:** Builds four identity-matched binaries in `bin/tui-pair/`: `asdx`, `asdx-core`, `agentic-sdd`, and `agentic-sdd-tui`, plus Go's build/module cache. All four share the same injected revision, commit, and source digest (`internal/buildinfo/buildinfo.go`).

### CLI only (`asdx-core`)

If you only need the non-interactive CLI or have **Go 1.23–1.25**:

```sh
go build -trimpath -o bin/asdx-core ./cmd/agentic-sdd
```

If `task` is installed, `task build` runs the same command.

- **Network effect:** None after cloning (the root `go.mod` has zero external dependencies).
- **Disk writes:** Writes `bin/asdx-core` (and Go's build cache).

> **Placing binaries on `PATH`:** Both `asdx-core tui` (`cmd/agentic-sdd/tui.go`) and the legacy `agentic-sdd` / `agentic-sdd-tui` wrappers resolve symlinks on the running binary and require the canonical sibling (`asdx` or `asdx-core`) in that **resolved directory** to be a regular executable file, not a symlink or a `PATH` lookup. Keep the four binaries together as regular files in one directory (such as `bin/tui-pair/` or `~/.local/bin/`) rather than symlinking `asdx` separately from `asdx-core`.

## Write and network effects

| Action | Files written | Network calls |
|---|---|---|
| `brew install --cask ...` or `go build` / `tui:pair` | Binaries (and Go caches on source builds) | Package / clone / TUI module fetch |
| `asdx-core version`, `--help`, `preview` | None | None |
| `asdx` (interactive TUI launch) | None until confirmed | One bounded read-only HTTPS GET per session to `https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest` (5s timeout, 64 KiB cap, no redirects, no telemetry). Set `AGENTIC_SDD_NO_UPDATE_CHECK=1` to disable. |
| `asdx-core apply` or confirmed TUI/`guide` install | Backups, selected skills/profiles, and private log under `<home>/.agentic-sdd/logs/` | None |

## Verify the installation

Run these read-only checks (prefix with `./bin/tui-pair/` or `./bin/` if you did not add the build directory to `PATH`):

```sh
asdx-core version
asdx-core --help
```

If you installed or built the paired TUI, you can also check `asdx` non-interactively without opening the alternate screen:

```sh
asdx --version
```

Expected observations:

- `asdx-core version` exits `0` and prints version, commit, build date, and Go version (`dev` / `none` / `unknown` for plain `go build` without ldflags).
- `asdx-core --help` exits `0` and lists command families (`preview`, `apply`, `version`, `backups`, `logs`, `restore`, `guide`, `tui`, `fleet`, `mcp`, `run`, `task`, `review`, `help`).
- `asdx --version` exits `0` and prints revision, commit, source digest, distribution marker, and protocol revision (`2`).

## Next steps

- Try a zero-write run against a temporary directory in [First five minutes](./first-five-minutes.md).
- Decide between CLI-only, TUI, and optional OpenCode orchestration in [Choose your setup](./choose-setup.md).
- Look up exact command flags and TUI routes in the [CLI reference](../reference/cli.md) and [TUI reference](../reference/tui.md).
