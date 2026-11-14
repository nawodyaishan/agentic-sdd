---
title: TUI reference
description: Interactive asdx routes, child screens, keys, focus, and update or browser behavior.
---

Reference for the interactive terminal UI binary (`asdx`, legacy compatibility name `agentic-sdd-tui`) and the `asdx-core tui` launcher. Source of truth: `experiments/tui/` and `cmd/agentic-sdd/tui.go`.

## Launch and entrypoint gates

- **Direct and launcher invocations**: run `asdx` directly in an interactive terminal, or run `asdx-core tui`. `asdx-core tui` resolves the sibling `asdx` binary in the same installation (no `PATH` search or shell), verifies build identity via `asdx --protocol-info` (protocol revision `2`, matching revision, commit, and source digest), and exits `1` if `asdx` is missing or mismatched. Bare `asdx-core` always runs a read-only CLI `preview` and never opens the TUI.
- **TTY and scripted-run gates**: both `asdx` and `asdx-core tui` require real stdin and stdout terminals. `asdx-core tui` additionally refuses `AGENTIC_SDD_FORCE_INTERACTIVE=1` with exit `2`; bare `asdx` does not check that variable.
- **Non-interactive queries on `asdx`**: `--version`, `--protocol-info` (mutually exclusive with `--version`, exit `2`), and `-h` / `--help` answer without a terminal and inspect no network or filesystem state.
- **Options (`asdx` and `asdx-core tui`)**:
  - `--home PATH`: user home whose client directories are inspected.
  - `--repo PATH`: repository checkout to read skills from (default: embedded assets).
  - `--fleet-config PATH`: explicit personal `fleet.json` path (default: empty/none selected; never scanned or inferred).
  - `--with-opencode`: include the optional OpenCode pack and profiles (**defaults to `true` on `asdx` / `asdx-core tui`**, unlike `asdx-core` CLI commands; pass `--with-opencode=false` for core skills only).
  - `--opencode-free-first`: preset OpenCode profiles to Free-first models (requires `--with-opencode`, else exit `2`).
- **Presentation environment**: `NO_COLOR` disables ANSI styling, `AGENTIC_SDD_ASCII=1` selects 7-bit ASCII borders and symbols (reversibly escaping non-ASCII data text), `AGENTIC_SDD_REDUCED_MOTION=1` disables the Home logo reveal animation, and `AGENTIC_SDD_NO_UPDATE_CHECK=1` disables automatic release checks.

## Primary routes (keys `1`–`5`)

Keys `1`–`5` jump directly to the five primary destinations from anywhere outside a blocking modal, write transaction, or active text input:

| Key | Route | Purpose |
|---|---|---|
| `1` | **Home** | Purpose summary, evidence-based setup state, primary setup/manage action (`Enter` or `s`), inventory overview, once-per-session update check status (`r` retry, `d` toggle), direct recovery (`b`/`R`), skippable 4-step **Quick start** tour (`o`), and explicit external links (`g` online README, `a` author profile). |
| `2` | **Install & Manage** | Destination hub for **Install or update** (`i`), **Backups & restore** (`b`), **Fleet Manager** (`f`), and **MCP Tools** (`t`), plus copyable read-only CLI preview hints and the active backup directory. |
| `3` | **Explore** | Read-only catalog of core skills, optional OpenCode pack/profiles, and Free-first presets (`/` search filter, `c` or `[`/`]` category filter, `Enter` detail card, `s` curated `agentic-sdd-router` details). Never writes files. |
| `4` | **Activity & Help** | Read-only inspection across 11 sections (`t`/`T` cycles `All`, `Recent Activity`, `Quick start`, `Shortcuts`, `Commands`, `Troubleshooting`, `Diagnostics`, `About`, `Preferences`, `Fleet Manager`, `MCP Tools`). Scans up to `10` `.log` files newest first from `<home>/.agentic-sdd/logs` (`[`/`]` or `p`/`n` selects log; `J`/`K` scrolls raw log content up to `64 KiB` / `200` lines; `r` refreshes logs and retries update check; `d` toggles update checks; `g`/`a` open external links). Unknown structured log counts stay `unknown`. |
| `5` | **Guide** | Presentation-only action menu (`i` Install or update, `b` Restore a backup, `l` View latest log in Activity, `o` Quick start tour, `a` About & update check in Activity, `g` Online guide / README). Opening a screen from Guide sets a return target so `Esc` or `b` at that destination's entry step returns to Guide; pressing `1`–`5` clears it. |

## Install & Manage child screens

All writes in the TUI go through the single-slot `OpsHost` (`app.go`). Only one write transaction (`Install`, `Restore`, `Fleet`, or `MCP`) can run at a time, and any write requires an explicit preview first.

- **Install or update (`i`)**: `Clients` (`Space` toggle, `Enter`/`->` next; requires at least one client) → `Options` (`--with-opencode`, `--opencode-free-first`) → `Review` → `Preview` (`x` toggle all paths, `[`/`]` select group, `Space` expand/collapse; zero-change no-op previews offer `h` Home, `l` Explore, `o` Options, and leaving discards the preview) → `Confirm` (default-No modal: `y`/`Y` only confirms, `Enter` is inert, `n`/`Esc`/`b`/`q` cancels) → `Running` → `Result`.
- **Backups & restore (`b`)**: `List` (`j`/`k` select, `Enter` open details, `r` refresh, `Esc` back) → `Details` (`m`/`M` toggles bounded raw `manifest.json`, `Enter`/`p` preview restore) → `Preview` (`Enter` confirm, `p` re-preview) → `Confirm` (default-No modal: `y`/`Y` only) → `Revalidating` (re-checks preview token before writing; drift returns to preview) → `Running` → `Result`.
- **Fleet Manager (`f`)**:
  - Views (`1`/`o` Overview, `2`/`p` Profiles, `3`/`c` Review Changes, `4`/`e` Recovery; `Tab` switches Overview/Profiles or toggles list/detail focus in wide Profiles).
  - **Overview**: `s` edits an explicit `fleet.json` path (no scanning), `d` opens the contextual offline diagnostics panel (`Esc` returns), `r` reloads, `Enter` opens Profiles, `Esc` returns to the hub (if staged edits exist, prompts `k` keep, `s`/`Esc` stay, or `d` discard). `H` jumps to Home.
  - **Profiles**: `/` filters instances, `m` opens the Model Picker (stages `provider/model[#variant]` in memory), `t` toggles enabled (disabling a managed instance shows a consequence prompt; unmanaged files cannot be removed by toggle), `Enter` opens detail card (compact) or Model Picker (wide), `l` toggles detail focus (wide). Staged edits stay in memory only; personal `fleet.json` source-saving is not available in this TUI build.
  - **Review Changes & Recovery**: `v` previews deployment or selected backup restore; `a` (apply) and `t` (restore) require `FleetLive` enabled at build time (`FleetCapabilityMarker=live`). Default shipped builds use `FleetCapabilityMarker=preview`, so deployment apply and restore remain preview-only and no runtime flag, environment variable, or config changes that marker. See [Fleet reference](./fleet.md).
- **MCP Tools (`t`)**:
  - Sub-views: **Overview** (fixed 3 rows: `codegraph`, `context7`, `exa`; `s`/`Enter` Select Servers, `v` Preview, `d` Diagnostics, `e`/`r` Recovery, `h` Home, `Esc` hub) → **Select Servers** (`j`/`k` move, `Space` toggle server or cycle auth mode `context7=oauth|api-key`, `exa=keyless|oauth|api-key` plus keyless rate-limit/privacy acknowledgement, `v`/`Enter` Preview) → **Preview** (`/` filter diff, `j`/`k` scroll, `Enter`/`y` open confirmation when conflict-free) → **Confirm Changes** (inline default-No confirmation with `No, cancel` selected initially; `Tab`/arrows toggle, `y` applies, `Enter` activates selected button, `Esc` cancels) → **Diagnostics** (`r` refresh offline checks, `Esc` return) → **Recovery** (`~/.agentic-sdd/mcp/backups/`: `j`/`k` select, `v`/`Enter` preview restore, `r`/`Enter` default-No confirm restore).
  - All MCP inspections and diagnostics are offline; no network handshake or OAuth flow runs from the TUI. See [MCP reference](./mcp.md).

## Contextual keys and modal precedence

Input precedence is strictly ordered (`handleKey` in `experiments/tui/app.go` and `actions.go`):

1. **Active write transaction**: `q`, `Esc`, navigation, help, and settings are ignored. `Ctrl+C` (or OS signal) schedules exit *after* the active write and log flush finish.
2. **Active text input**: Explore search (`/`), Fleet filter/source/model-picker/doctor fields, and MCP filter capture printable keys (including `q` and `1`–`5`); `Ctrl+C` still requests shutdown.
3. **Help overlay (`?`)**: shows global keys plus actions for the underlying screen; `m` toggles motion, `u` toggles Unicode/ASCII, `d` toggles update checks; `j`/`k`/`PgUp`/`PgDn`/`Ctrl+D`/`Ctrl+U` scroll; `?`/`Esc`/`Enter` close (`q` never quits through a modal).
4. **Install/Restore confirmation modal**: only `y`/`Y` accepts, `Enter` is inert, `n`/`Esc`/`b`/`q` cancels safely without quitting, `j`/`k`/arrows/`PgUp`/`PgDn`/`Home`/`g` scroll, `?` opens help, and `Ctrl+C` schedules shutdown.
5. **Quick start onboarding overlay (`o` on Home/Guide)**: 4-step read-only tour (`Enter`/`Right`/`l`/`Tab` next, `Left`/`h`/`Backspace`/`b` back, `s` skip, `Esc` cancel, `o`/`a` explicit external links on Finish).
6. **Context-specific key overrides**: `c`/`C` in Fleet Manager opens `Review Changes` instead of Explore categories; `m`/`M` toggles raw `manifest.json` on Restore `Details` and opens the Model Picker on Fleet `Profiles` instead of toggling global motion. Outside those contexts, `m` toggles motion, `u` toggles Unicode/ASCII, and `q`/`Ctrl+C` quit when idle.

## Focus, responsive layout, and scroll behavior

- **Focus cycling**: `Tab` / `Shift+Tab` toggles `Nav` ↔ `Content` (on Explore, it cycles `Nav` → `Categories` → `Content`). Switching routes via `1`–`5` moves focus to `Content`.
- **Scroll preservation and clamping**: each primary route keeps its own `routeScroll` offset across `1`–`5` switches, while stage/step/view transitions inside Install, Restore, Fleet, and Activity reset scroll to `0`. Scroll offsets are clamped before handling input so repeated `PgDn` at the bottom does not accumulate hidden offset. On Install `Preview` (`height >= 16`), the summary stays pinned while destination details scroll below it.
- **Responsive breakpoints**:
  - Below `40x8`: shows only resize guidance and `q quit`; affirmative confirmation keys are ignored.
  - Persistent left sidebar (`18` columns): active when `width >= 110` or `60 <= width < 80`. Compact top navigation pills are used at `80 <= width <= 109` and `width < 60`.
  - Explore split pane: side-by-side list + detail card at `width >= 80`; single-pane list/drill-down below `80`.
  - Fleet Profiles split pane: side-by-side at `width >= 110` (or child content `width >= 90` when `cfg.Width >= 110`).
  - Modals (`?` help and Install/Restore confirmations): centered framed box (`maxOuterWidth = 88`) when `width >= 60 && height >= 16` and content fits; otherwise full-screen scrollable presentation.
  - Status notices (`Notice:`): rendered in a reserved wrapped footer block above the key hint (`height > 8`), never truncated.

## Update checks and browser links

- **Release update check (`experiments/tui/update_state.go`, `internal/appops/update.go`)**:
  - Runs asynchronously at most once automatically on first `Home` entry unless `AGENTIC_SDD_NO_UPDATE_CHECK=1` is set or checks were disabled with `d`.
  - Sends a single HTTP `GET` to `https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest` (5s timeout, `64 KiB` body limit, redirects refused). Sends no credentials or local inventory, and never downloads or installs a release.
  - States: `not started`, `checking`, `current`, `newer`, `unavailable` (offline, rate-limited, or dev build), `disabled`. Explicit `r` cancels any in-flight check and uses a generation counter to discard stale replies.
- **Browser links (`experiments/tui/opener.go`)**:
  - Only explicit user actions on trusted fixed destinations invoke the opener (`g` for the online README, `a` for the author profile, `o` in Onboarding Finish, and `Enter` on a Guide documentation row).
  - No shell is used: macOS executes `/usr/bin/open` and Linux uses cached `xdg-open` from `PATH` (relative paths are refused).
  - When `SSH_CONNECTION`, `SSH_CLIENT`, or `SSH_TTY` is set, on headless Linux (`DISPLAY` and `WAYLAND_DISPLAY` both unset), on unsupported OSes, or if launching fails, no browser runs and the literal HTTPS URL remains visible in a `Notice:` so you can copy it manually. OSC 8 terminal hyperlinks are emitted only for `Color && !ASCII` on non-wrapped URLs.

## Documentation in the TUI

The Guide route (key `5`) shows a Documentation section listing seven fixed destinations: Getting started, CLI reference, TUI reference, Skills, Updating and recovery, Troubleshooting, and Contributing / docs changes. Move with `j`/`k`, press `Enter` to open the selected destination explicitly, and `g` still opens the online README. Each destination is a fixed GitHub blob URL of the canonical page source on `main`. On a headless or SSH session, on an unsupported platform, or when the opener fails, no browser is launched: the literal URL stays visible in the `Notice:` so it can be copied manually, and no success is claimed.

## Related reference pages

- [CLI reference](./cli.md) — `asdx-core` command families, flags, and exit codes.
- [Skill catalog](./skills.md) — core skills and optional orchestration skill.
- [Files and backups](./files-and-backups.md) — managed destinations, backup roots, and store separation.
- [Fleet reference](./fleet.md) — `fleet.json` schema and build-time capability gate.
- [MCP reference](./mcp.md) — MCP target resolution, auth modes, and offline diagnostics.
