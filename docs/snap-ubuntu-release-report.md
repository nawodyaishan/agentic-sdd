# Snap and Ubuntu release report

**Date:** 2026-11-13
**Status:** research report. Not a spec, plan, approval record, batch authorization or release decision.
**Subject:** whether and how to release `agentic-sdd` to Snap and to Ubuntu Linux systems
**Method:** repository inspection (code, release config, existing packaging research) plus current Snapcraft, Snap Store and GoReleaser documentation retrieved through Exa and Context7.

## 1. Executive summary

`agentic-sdd` cannot be released as a strictly confined Snap and still do its job. The tool's purpose is to write skill files into **hidden directories in the user's real home** — `~/.claude/skills`, `~/.codex/skills`, `~/.agents/skills`, `~/.gemini/antigravity-cli/skills`, `~/.config/opencode/agents` — and to keep its own state in `~/.agentic-sdd`. Snapd rewrites `$HOME` to `~/snap/<name>/<revision>` for every non-classic snap, so an unmodified build would report a successful install while writing to a directory no agent ever reads.

The three Snap options are:

| Option | Verdict |
|---|---|
| Strict confinement, no interfaces | **Broken.** Silently installs into `~/snap/...`; the worst possible failure mode |
| Strict confinement + `personal-files` | **Technically possible, practically hostile.** ~80 path entries, a Store-approved snap declaration, a manual `snap connect` per machine, and it still breaks `--home` and `--repo` |
| Classic confinement | **Faithful, gated.** Needs Snap Store review; the policy both supports and rejects this use case depending on how a reviewer reads it. Not installable on Ubuntu Core |

**Recommendation:** ship Ubuntu support as a **`.deb` plus a `linux` tar.gz** produced by the existing GoReleaser pipeline, and treat a Snap as an optional, separately decided classic-confinement channel. The `.deb` has no confinement model to fight, preserves every documented behaviour (`preview`, `apply`, `backups`, `restore`, `--home`, `--repo`), installs the paired CLI + TUI from one artifact, and is the direct analogue of the existing Homebrew cask.

## 2. Scope and method

In scope:

- Current packaging requirements implied by the codebase.
- Snap confinement models against the tool's actual write set.
- Snap Store review policy for classic and `personal-files` interfaces.
- Build and publishing mechanics for a Snap (`snapcraft`, GoReleaser `snapcrafts`, Store credentials, CI shape).
- The Ubuntu-native alternative (`.deb` via GoReleaser `nfpms`), including a PPA as a later step.

Out of scope:

- Flatpak, AppImage, Arch, Fedora or Nix packaging.
- Any change to the CLI's install semantics or the paired-identity protocol.
- Any live operation: Store registration, credential export, tag, upload, release.

## 3. What the codebase requires from a Linux packager

| Requirement | Evidence | Consequence for packaging |
|---|---|---|
| Two binaries in a matched pair | `cmd/agentic-sdd` (root module, `go 1.23`, stdlib only); `experiments/tui/cmd/tui-companion` (nested module, `go 1.26`, Bubble Tea v2) | Any Linux artifact must ship both, and both must be built from the same revision |
| Static binaries | `CGO_ENABLED=0` in every GoReleaser build | No runtime dependencies; nothing to stage or compile at install time |
| Self-contained payload | `go:embed all:skills-files` in `skillsfiles.go` | The binary needs no data directory; `--repo` is optional |
| Exact write set | `clientTargets` in `internal/skillsync/sync.go`; `openCodeAgentsTarget`; `backupRootFor`; `logsDir` | Four hidden client trees, one opt-in hidden tree, one private state tree |
| Real home, by design | `resolveHome` → `os.UserHomeDir()`, overridden by `--home` | The tool must follow the user's actual home, not a substituted one |
| Arbitrary repo | `--repo PATH` reads `skills-files/` from disk | Requires real read access outside the package |
| Verifiable identity | `internal/buildinfo` `-ldflags` injection; `VerifyCompanion` rejects placeholders | A Linux build must inject identical revision/commit/`SDD_SOURCE_DIGEST` into both binaries, or the TUI refuses to pair |

Destination map:

| Destination | Writer | Hidden | Third-party owned |
|---|---|---|---|
| `~/.agents/skills/agentic-sdd-*` | `skillsync` | yes | yes |
| `~/.codex/skills/agentic-sdd-*` | `skillsync` | yes | yes |
| `~/.claude/skills/agentic-sdd-*` | `skillsync` | yes | yes |
| `~/.gemini/antigravity-cli/skills/agentic-sdd-*` | `skillsync` | yes | yes |
| `~/.config/opencode/agents/*.md` | `--with-opencode` | yes | yes |
| `~/.agentic-sdd/{backups,logs}` | apply/restore | yes | no (this tool) |

## 4. Findings

### F1 — Strict confinement rewrites `$HOME`, and the tool trusts `$HOME`

The official environment-variable reference states that for non-classic snaps `HOME` is re-written to `SNAP_USER_DATA`, and that classic snaps keep the original value. `resolveHome` returns `os.UserHomeDir()`, which on Linux reads `HOME`.

Net effect of an unmodified strictly confined build:

```text
$ agentic-sdd apply
install claude/agentic-sdd-plan  →  ~/snap/agentic-sdd/12/.claude/skills/agentic-sdd-plan
```

Exit code 0, files written, nothing installed where Claude Code looks. Even `preview` becomes misleading, because the destination summary is the snap's private directory.

### F2 — The `home` interface cannot help

`home` grants access to *non-hidden* files owned by the user in `$HOME`. Every destination in the map above is hidden. `home` is therefore not part of any viable design here, beyond incidental reads.

### F3 — `personal-files` is the "correct" strict answer, but the ownership rule does not fit

`personal-files` accepts `read` and `write` path lists and requires an approved snap declaration for Store distribution. Its documented purpose is access to top-level hidden directories *"where the snap is the clear owner of the target directory"*.

This tool is not the owner of `.claude`, `.codex`, `.agents`, `.gemini/antigravity-cli` or `.config/opencode`. The narrow defensible shape would be per-skill paths:

```yaml
plugs:
  dot-claude-skill-plan:
    interface: personal-files
    write: [$HOME/.claude/skills/agentic-sdd-plan]
  # ... × 19 skills × 4 clients, plus ~/.agentic-sdd and the opt-in agents dir
```

That is roughly 80 declaration entries, a Store review, and a manual or granted auto-connection per machine. It also permanently breaks `--home PATH` and `--repo PATH`, which are documented features. Store reviewers have refused directory-level write access to shared dot-directories before — for example `$HOME/.config` for `doctl` — on exactly the grounds that a snap would then be able to modify other applications' configuration.

### F4 — Classic confinement is faithful but review-gated, with an ambiguous fit

`confinement: classic` preserves the real `$HOME`, so the tool behaves exactly as the tarball and Homebrew builds do. Costs: installs require `--classic`; the snap is not installable on Ubuntu Core; and it runs in the global mount namespace, so the publisher must ship all dependencies (trivially satisfied by a static Go binary).

The Store review policy is genuinely two-sided for this project:

**Supported category that plausibly applies**

> "AI agent/assistant that runs arbitrary user-directed code and needs to reach files and programs that aren't known at build time"

**Unsupported entries that plainly apply**

> "access to dot files in `$HOME` (use `$HOME` instead of `getent*`, personal-files)"
> "3rd party installer snaps (eg, for native packages, AppImages, Flatpaks, Snaps, etc)"
> "management snaps"

The repository cannot assume approval. A classic request would have to argue the first line and explain why the second does not control — a reasonable argument, but a reviewer judgement, not an engineering fact.

### F5 — The distribution-marker contract constrains the build

`buildinfo.Distribution.Valid()` accepts exactly `dev`, `generic` and `homebrew`. `AllowsInteractiveDefault()` is true only for `homebrew`, so a Snap build has two honest choices:

1. **Reuse `generic`.** Zero code change. Bare `agentic-sdd` keeps the historical preview behaviour. `VerifyCompanion` still passes, because it compares identity, not marker equality.
2. **Add a `snap` marker.** Touches `internal/buildinfo`, its tests, and the marker-count assertions in `internal/buildinfo/release_config_test.go`.

Either way, `release_config_test.go` already asserts `goos = [darwin]` for all three builds and exact marker counts (one `homebrew`, two `generic`). It will fail the moment a Linux build is added — by design — so it must be updated in the same change as any `linux` matrix addition.

### F6 — Linux build support does not exist yet

`.goreleaser.yml` builds only `darwin` (`amd64`, `arm64`, then universal binaries). There is no `linux` build, no `.deb`, no snap. The `Taskfile.yml` `build-darwin` and `release` tasks reflect that. Any Ubuntu channel requires adding `linux/amd64` and `linux/arm64` to all three builds first; that is the shared prerequisite and the highest-risk unknown.

### F7 — Snap build mechanics are workable but have concrete frictions

- `core26` (Ubuntu 26.04) is stable from 2026-04-29; `core24` is the conservative alternative.
- The nested TUI module needs `go 1.26`. The `go` snap publishes `1.26/stable` (1.26.11) and `1.27/stable` (1.27.4), so `build-snaps: [go/1.26/stable]` is a valid provision. The Go plugin does not bundle a compiler.
- GoReleaser has a `snapcrafts` pipe that can set `confinement`, `base`, `grade`, `plugs`, per-app commands, channels and `publish`. It shells out to `snapcraft` and documents that snaps **cannot be built inside a Docker container**.
- GoReleaser's documented `base` values are stale (still showing `core18`), so `base: core26` must be proven on the pinned `v2.14.0` before it is trusted; otherwise hand-author `snap/snapcraft.yaml`.
- On this host: `snapcraft` is not installed, `snapd` is 2.78, Ubuntu is 26.04, LXD is present with no instances, and `snap info agentic-sdd` reports no published snap. The name appears unclaimed, but registration is first-come and is a live operation.

### F8 — `nfpms` covers the Ubuntu need with no new constraints

GoReleaser's `nfpms` pipe builds `.deb` from the same prebuilt Linux binaries, with `ids`, `bindir`, `contents`, `dependencies`, `section`, `priority` and per-packager lintian overrides (for example `statically-linked-binary`). It needs no confinement declaration, no Store review, no daemon, no install-time script, and no rename of the paired binaries.

## 5. Options

| ID | Option | Meets tool's semantics | Distribution cost | Risk | Verdict |
|---|---|---|---|---|---|
| O1 | Snap, strict, no interfaces | No | Low | Silent wrong success | Reject |
| O2 | Snap, strict + `personal-files` | No (`--home`/`--repo` broken) | High (declaration + manual connect) | Review refusal, ~80 entries | Reject |
| O3 | Snap, classic | Yes | Medium–High (review, `--classic`, no Ubuntu Core) | Ambiguous policy fit | Optional, separate decision |
| O4 | `.deb` via `nfpms` + `linux` tar.gz | Yes | Low (same release run) | Low | **Recommend** |
| O5 | Launchpad PPA on top of O4 | Yes | Medium (PPA ownership, signing) | Low | Later, if discoverability matters |

## 6. Recommendation

1. **Adopt O4 as the Ubuntu support statement.** Add `linux/amd64` and `linux/arm64` to the three GoReleaser builds, add an `nfpms` entry producing `agentic-sdd_<version>_<arch>.deb` with both binaries in `/usr/bin`, and keep the `linux` tar.gz archives as the no-package fallback.
2. **Keep `DistributionMarker=generic`** for the Linux artifacts. Bare `agentic-sdd` on Linux keeps the historical preview behaviour, which is also the safer default for a package manager install.
3. **Do not ship a strictly confined snap.** If a Snap is wanted later, ship it as `confinement: classic` and accept the review gate, or distribute the `.snap` from GitHub Releases with `snap install --classic --dangerous` and document that it is unlisted.
4. **Document the Linux install path in README** next to the Homebrew cask, and state plainly that on Linux the tool writes to the real home — the same contract as the tarball.

## 7. Proposed implementation outline

Phased, cheapest and most informative first. Only phase 0 is a prerequisite for everything else.

| Phase | Work | Live operation |
|---|---|---|
| 0 | Add `linux/amd64` + `linux/arm64` to the three builds; update `release_config_test.go` (`goos`, marker counts, archive expectations); prove with `task release` snapshot that both binaries carry an identical injected identity | No |
| 1 | Add the `nfpms` block for `deb`; snapshot again; inspect with `dpkg-deb -c`; install into a disposable container and run `preview`, `apply --home <tmp>`, `backups`, `restore`, and `agentic-sdd-tui --version` | No |
| 2 | Document the `.deb` and tar.gz install paths in README; note the Linux write-set contract | No |
| 3 | *Only if a Snap is approved:* hand-author `snap/snapcraft.yaml` (`base: core26`, `confinement: classic`, two apps), `snapcraft pack`, install with `--dangerous --classic`, and verify the destination paths are the real home | Local only |
| 4 | *Only if a Store listing is approved:* `snapcraft register`, `export-login` to CI credentials, upload to `edge`, promote | **Yes** |

A strict-confinement variant is worth building once as evidence, to demonstrate the `~/snap/...` divergence rather than assert it. It should not be published.

## 8. Decision register

| ID | Decision | Blocks | Owner |
|---|---|---|---|
| D1 | Is Snap a supported channel, an unlisted sideload, or out of scope? | Any `snap/` tree, `snapcrafts`, Store work | Owner |
| D2 | If Snap: `classic` or `strict` + `personal-files`? | `confinement`, interface declaration, README wording | Owner |
| D3 | Accept `.deb` as the primary Ubuntu artifact? | `nfpms` block, README, release workflow | Owner |
| D4 | Add `linux/amd64` + `linux/arm64` to the release matrix? | Both channels, `release_config_test.go` | Owner |
| D5 | New `snap` distribution marker, or reuse `generic`? | `internal/buildinfo`, handshake tests, marker counts | Owner |
| D6 | Authorize a real `snapcraft register`, tag, or Store upload? | Phases 3–4 | Owner |

## 9. Open questions and unverified items

- Whether GoReleaser `v2.14.0`'s `snapcrafts` pipe emits a current-enough `snapcraft.yaml` for `base: core26` and multi-arch. Unverified; the hand-authored path avoids the question.
- Whether `snapcraft` in GitHub Actions can build without `--destructive-mode` on a hosted runner, given the documented Docker restriction and the LXD requirement.
- Whether Store reviewers would grant classic to this specific tool. Policy supports and rejects it; only a request settles it.
- Whether the snap name `agentic-sdd` is registrable. Nothing is published under it, but reservation status is only visible at registration.
- Whether a `linux` build changes the `SDD_SOURCE_DIGEST` convention in `release.yml` (`SDD_SOURCE_DIGEST` falls back to `git:<commit>`), and therefore whether companion identity stays byte-identical across the macOS and Linux matrices.

## 10. Evidence and sources

Repository evidence:

- `.goreleaser.yml` — darwin-only builds, three build IDs, two paired archives, cask, marker injection.
- `.github/workflows/release.yml` — `task verify`, Go `1.27.1`, GoReleaser `v2.14.0`, tap token check.
- `internal/buildinfo/buildinfo.go` — markers, `Valid()`, `AllowsInteractiveDefault()`, `VerifyCompanion`.
- `internal/buildinfo/release_config_test.go` — `goos = [darwin]`, marker counts, archive and cask assertions.
- `internal/skillsync/sync.go` — `clientTargets`, `backupRootFor`.
- `cmd/agentic-sdd/main.go` — `resolveHome`, `printUsage`, `--home`/`--repo`.
- `specs/013-unified-tui-experience/b6-packaging-research.md` — prior packaging evidence and cask migration.

External sources:

- Snap environment variables (`HOME` rewrite, `SNAP_REAL_HOME`): <https://snapcraft.io/docs/reference/development/environment-variables/>
- `home` interface, non-hidden files only: <https://snapcraft.io/docs/reference/interfaces/home-interface/>
- `personal-files` interface, declaration approval and ownership expectation: <https://snapcraft.io/docs/reference/interfaces/personal-files-interface/>
- Classic confinement review policy, supported and unsupported categories: <https://snapcraft.io/docs/reference/administration/reviewing-classic-confinement-snaps/>
- Classic confinement behaviour and maintenance cost: <https://snapcraft.io/docs/explanation/security/classic-confinement/>
- Snap bases and `core26`: <https://snapcraft.io/core26>
- Snapcraft base selection: <https://ubuntu.com/docs/snapcraft/9/how-to/crafting/specify-a-base/>
- Go plugin and compiler provisioning: <https://ubuntu.com/docs/snapcraft/9.2/common/craft-parts/reference/plugins/go_plugin/>
- GoReleaser snapcraft pipe: <https://goreleaser.com/customization/package/snapcraft/>
- GoReleaser nFPM pipe: <https://goreleaser.com/customization/package/nfpm/>
- Store precedent on dot-directory write access (`doctl`): <https://forum.snapcraft.io/t/personal-files-request-for-doctl-was-classic-confinement-request-doctl/11785>

## Appendix A — configuration sketches (not validated, not applied)

`.deb` via GoReleaser:

```yaml
nfpms:
  - id: linux-deb
    ids: [cli-generic, tui-companion]
    package_name: agentic-sdd
    formats: [deb]
    bindir: /usr/bin
    maintainer: Nawodya Ishan <nawodyaishan@users.noreply.github.com>
    homepage: https://github.com/nawodyaishan/agentic-sdd
    description: |
      Spec-driven development workflow skills, kept in sync across Codex,
      Claude Code and the Antigravity CLI.
    license: MIT
    section: devel
    priority: optional
    deb:
      lintian_overrides:
        - statically-linked-binary
```

Classic snap via GoReleaser:

```yaml
snapcrafts:
  - id: cli-snap
    ids: [cli-generic, tui-companion]
    name: agentic-sdd
    base: core26          # must be proven on the pinned GoReleaser
    confinement: classic
    grade: stable
    license: MIT
    summary: Spec-driven development workflow skills for agent CLIs
    publish: false        # publishing is a separate authorized step
    apps:
      agentic-sdd:
        command: agentic-sdd
      agentic-sdd-tui:
        command: agentic-sdd-tui
```

Hand-authored classic snap, from source:

```yaml
name: agentic-sdd
base: core26
version: "0.7.0"
summary: Spec-driven development workflow skills for agent CLIs
description: |
  Preview-first installer that keeps Agentic SDD skills in sync across
  supported agent clients, with backups, restore and a terminal UI.
grade: stable
confinement: classic
platforms:
  amd64:
  arm64:

parts:
  cli:
    plugin: go
    source: .
    # Root module declares go 1.23; the 1.26 toolchain builds it unchanged.
    build-snaps: [go/1.26/stable]
  tui:
    plugin: go
    source: experiments/tui
    build-snaps: [go/1.26/stable]

apps:
  agentic-sdd:
    command: bin/agentic-sdd
  agentic-sdd-tui:
    command: bin/agentic-sdd-tui
```

## Appendix B — verification commands for the proposed phases

```sh
# Phase 0: snapshot release, then confirm identity injection on Linux
task release
file dist/agentic-sdd_*_linux_amd64_generic/agentic-sdd-generic
dist/agentic-sdd_*_linux_amd64_generic/agentic-sdd-generic version
dist/agentic-sdd_*_linux_amd64_generic/agentic-sdd-tui --protocol-info

# Phase 1: inspect and smoke-test the .deb in an isolated container
dpkg-deb -c dist/agentic-sdd_*_linux_amd64.deb
docker run --rm -v "$PWD/dist:/dist:ro" debian:stable-slim \
  sh -c 'dpkg -i /dist/agentic-sdd_*_linux_amd64.deb && agentic-sdd version'

# Phase 3 (Snap, local only)
snapcraft pack
sudo snap install --dangerous --classic ./agentic-sdd_*.snap
agentic-sdd preview            # destinations must be the real home
agentic-sdd apply --home "$(mktemp -d)"
```
