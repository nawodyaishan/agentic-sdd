# B6 packaging research — unblocking T15

**Status:** implementation evidence for B6/T15. Not an approval record.
**Date:** 2026-11-10
**Question:** Why was the B6 T15 snapshot blocked, and what is the supported GoReleaser contract for paired macOS CLI + companion archives?

## Findings

### 1. GoReleaser creates universal binaries internally; external `lipo` is not required

GoReleaser's universal-binary support is based on the Go `makefat` approach, not a call to macOS `lipo`. The official GoReleaser Dockerfile does not install `lipo`, and the `androidqf` example explicitly notes that universal binaries work on Linux runners. The B6 snapshot task's `lipo`/`llvm-lipo` prerequisite was therefore a false blocker. The task now checks only `git`, `goreleaser` and `go`.

Sources: [GoReleaser universal binaries](https://goreleaser.com/customization/builds/universalbinaries/), [GoReleaser universal-binary PR #2572](https://github.com/goreleaser/goreleaser/pull/2572), [GoReleaser Dockerfile](https://github.com/goreleaser/goreleaser/blob/main/Dockerfile), [androidqf `.goreleaser.yaml`](https://github.com/mvt-project/androidqf/blob/main/.goreleaser.yaml).

### 2. Archive filters must use the universal artifact IDs after `replace: true`

The first snapshot failed with:

```text
error=no linux/macos archives found matching ... ids=[cli-homebrew-pair]
```

`universal_binaries.replace: true` removes the single-arch artifacts. The archive `ids` must select the universal output IDs (`cli-generic-universal`, `cli-homebrew-universal`, `tui-companion-universal`), not the original build IDs. After that correction, both paired archives were created.

### 3. A tar.gz archive can hold multiple builds when their platform matrix matches

GoReleaser's gzip/xz note says a compression-only archive cannot hold multiple files and recommends per-build archives. Our paired archives use `tar.gz`, include two builds with the same darwin `amd64`/`arm64` matrix, and GoReleaser produced both successfully. No `allow_different_binary_count` escape hatch is used. If a future build matrix makes the binary count differ per platform, the archive must be split or the matrix aligned; do not add the escape hatch.

Sources: [GoReleaser archives](https://goreleaser.com/customization/package/archives/), [GoReleaser v2.8 deprecation](https://goreleaser.com/blog/goreleaser-v2.8/).

### 4. Version pinning matters

- GoReleaser `v2.14.0` builds with Go `1.27.1`.
- GoReleaser `v2.20.0` requires Go `>=1.27.3`, so it cannot run on the reviewed `go1.27.1` patch without a toolchain download.
- Pin GoReleaser `v2.14.0` in both the local snapshot task and the release workflow. The snapshot task checks `goreleaser --version` exactly.

Sources: local `go install` attempts and `goreleaser --version`; [GoReleaser releases](https://github.com/goreleaser/goreleaser/releases).

### 5. `brews` is deprecated but functional for this formula

GoReleaser v2.10 deprecates `brews` in favor of `homebrew_casks`, but v2.14 still generates the formula and the snapshot proof succeeds. The formula installs both binaries and has no `post_install` launch. Revisit `homebrew_casks` only in a separately reviewed migration; a CLI formula is still appropriate here.

Source: [GoReleaser Homebrew formulas](https://goreleaser.com/customization/publish/homebrew_formulas/), [Homebrew casks](https://goreleaser.com/customization/publish/homebrew_casks/).

### 6. Release workflow toolchain selection

`actions/setup-go` supports `go-version` and `go-version-file`, including nested `go.mod` files and monorepo cache paths. The release job now selects the reviewed `1.27.1` patch explicitly and caches `experiments/tui/go.sum`; setup-go pins `GOTOOLCHAIN=local`, so the companion build never downloads a toolchain. The existing core verification/toolchain pins remain separate.

Sources: [actions/setup-go advanced usage](https://github.com/actions/setup-go/blob/main/docs/advanced-usage.md), [setup-go GOTOOLCHAIN behavior](https://github.com/actions/setup-go/issues/491), Context7 `/actions/setup-go`.

## Local proof

Tooling: GoReleaser `v2.14.0` installed outside the repository in `/tmp/opencode/b6-tools` (not committed, not vendored); Go `1.27.1`; GoReleaser's internal universal-binary implementation.

Command:

```sh
PATH=/tmp/opencode/b6-tools:$PATH task -t Taskfile.experimental.yml release:snapshot
```

Result: `release succeeded` and `release:snapshot: local nonpublishing snapshot complete for dev+430b0e0ab521-dirty`.

Evidence:

| Check | Result |
|---|---|
| Generic archive | `agentic-sdd_0.6.0-SNAPSHOT-430b0e0_darwin_all_generic.tar.gz`, contains `agentic-sdd-generic` + `agentic-sdd-tui` |
| Homebrew archive | `agentic-sdd_0.6.0-SNAPSHOT-430b0e0_darwin_all_homebrew.tar.gz`, contains `agentic-sdd-homebrew` + `agentic-sdd-tui` |
| Universal slices | `file` reports Mach-O universal with `x86_64` + `arm64` for both binaries |
| Checksums | `dist/checksums.txt` has both archive digests |
| Formula | `dist/homebrew/Formula/agentic-sdd.rb` selects `cli-homebrew-pair`, installs `agentic-sdd-homebrew => agentic-sdd` and `agentic-sdd-tui`, no post-install launch |
| Identity | T15-only snapshot: both binaries contain the same `0.6.0-SNAPSHOT-430b0e0`, commit `430b0e0…`, `local-sha256:a4ca81f…`. Re-run after T14: both contain `local-sha256:70e6e249…`. The companion `--protocol-info` is non-interactive by design and cannot be executed on Linux because the release binary is Mach-O. |
| Marker | Static config asserts `homebrew` only on `cli-homebrew`; generic CLI and companion use `generic` |

## Changes made

- `.goreleaser.yml`: archive `ids` now reference the universal artifact IDs; comments explain the pairing.
- `Taskfile.experimental.yml`: removed the false `lipo` prerequisite, pinned `SDD_RELEASE_GORELEASER_VERSION=v2.14.0`, added `goreleaser --version` and `goreleaser check` validation.
- `.github/workflows/release.yml`: release job selects Go `1.27.1`, caches `experiments/tui/go.sum`, and pins GoReleaser `v2.14.0`.
- `cmd/agentic-sdd/buildinfo_link.go`: keeps the injected `revision`/`commit`/`source_digest` linked into the CLI before T14 wires the companion handshake, without changing CLI output.
- `internal/buildinfo`: existing T15 identity/protocol contract; no new dependencies.

## Remaining after T15

- T14: root CLI explicit `tui`, Homebrew interactive zero-argument default, sibling companion resolution, protocol-info handshake, absent/mismatch errors, additive help lines, parent-child signal forwarding, README/Taskfile commands.
- T16: final scoped regression/walkthrough and architecture support reconciliation.
- `internal/buildinfo` still has focused tests but is not in the 8-package core selector; adding it is a separate gate-selection review.
