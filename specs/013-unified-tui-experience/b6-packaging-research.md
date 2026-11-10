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

### 5. `brews` is deprecated; the reviewed migration is a cask

GoReleaser v2.10 deprecated `brews` in favor of `homebrew_casks`, and v2.16 says the formula pipeline is fully deprecated. The pinned v2.14 still generates a working formula, but GoReleaser's supported path for prebuilt binaries is a cask whose `binary` stanzas link executables into `$(brew --prefix)/bin`. Homebrew's general packaging guidance prefers formulae for source-buildable open-source CLIs; this project does not build from source at install time, so the prebuilt-artifact cask is the honest package type. The migration decision is recorded in this document and the Feature 013 decision log; no live tap change was made.

Sources: [GoReleaser v2.10 release notes](https://goreleaser.com/blog/goreleaser-v2.10), [GoReleaser v2.16 release notes](https://goreleaser.com/blog/goreleaser-v2.16), [Homebrew casks](https://goreleaser.com/customization/publish/homebrew_casks/), [Homebrew: Adding Software](https://docs.brew.sh/Adding-Software-to-Homebrew), [Cask Cookbook `binary`](https://docs.brew.sh/Cask-Cookbook#stanza-binary).

### 6. Release workflow toolchain selection

`actions/setup-go` supports `go-version` and `go-version-file`, including nested `go.mod` files and monorepo cache paths. The release job now selects the reviewed `1.27.1` patch explicitly and caches `experiments/tui/go.sum`; setup-go pins `GOTOOLCHAIN=local`, so the companion build never downloads a toolchain. The existing core verification/toolchain pins remain separate.

Sources: [actions/setup-go advanced usage](https://github.com/actions/setup-go/blob/main/docs/advanced-usage.md), [setup-go GOTOOLCHAIN behavior](https://github.com/actions/setup-go/issues/491), Context7 `/actions/setup-go`.

## Local proof (T15-era formula)

This section records the pre-migration formula proof for history; the current package contract is the cask proof below.

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
| Homebrew archive (pre-migration) | `agentic-sdd_0.6.0-SNAPSHOT-430b0e0_darwin_all_homebrew.tar.gz`, contains `agentic-sdd-homebrew` + `agentic-sdd-tui` |
| Universal slices | `file` reports Mach-O universal with `x86_64` + `arm64` for both binaries |
| Checksums | `dist/checksums.txt` has both archive digests |
| Formula (pre-migration) | `dist/homebrew/Formula/agentic-sdd.rb` selects `cli-homebrew-pair`, installs `agentic-sdd-homebrew => agentic-sdd` and `agentic-sdd-tui`, no post-install launch |
| Identity | T15-only snapshot: both binaries contain the same `0.6.0-SNAPSHOT-430b0e0`, commit `430b0e0…`, `local-sha256:a4ca81f…`. Re-run after T14: both contain `local-sha256:70e6e249…`. The companion `--protocol-info` is non-interactive by design and cannot be executed on Linux because the release binary is Mach-O. |
| Marker | Static config asserts `homebrew` only on `cli-homebrew`; generic CLI and companion use `generic` |

## Changes made

- `.goreleaser.yml`: archive `ids` now reference the universal artifact IDs; comments explain the pairing.
- `.goreleaser.yml`: the Homebrew CLI build and universal output are named `agentic-sdd`; the cask links that file directly instead of renaming it inside a formula.
- `.goreleaser.yml`: `brews` is replaced by `homebrew_casks` (`directory: Casks`, `binaries: [agentic-sdd, agentic-sdd-tui]`) with a reviewed post-install hook that removes the macOS quarantine bit from both unsigned binaries.
- `Taskfile.experimental.yml`: removed the false `lipo` prerequisite, pinned `SDD_RELEASE_GORELEASER_VERSION=v2.14.0`, added `goreleaser --version` and `goreleaser check` validation, and updated the completion text from formula to cask.
- `.github/workflows/release.yml`: release job selects Go `1.27.1`, caches `experiments/tui/go.sum`, and pins GoReleaser `v2.14.0`.
- `cmd/agentic-sdd/buildinfo_link.go`: keeps the injected `revision`/`commit`/`source_digest` linked into the CLI before T14 wires the companion handshake, without changing CLI output.
- `internal/buildinfo`: existing T15 identity/protocol contract; the release-config test now asserts the cask shape, Casks directory, binary names and quarantine hook, and rejects the deprecated `brews`/`Formula` layout.
- `README.md`, `AGENTS.md`, `docs/tui-architecture.md`: install command is `brew install --cask`, with the earlier-formula upgrade path and the prebuilt-binary/Host-GO consequence stated.

## Remaining

- T16: final scoped regression/walkthrough and architecture support reconciliation, including the cask journey on a real macOS Homebrew host.
- The tap still contains `Formula/agentic-sdd.rb` and no `Casks/agentic-sdd.rb`; publishing the cask and the tap migration are separate live operations.
- `internal/buildinfo` still has focused tests but is not in the 8-package core selector; adding it is a separate gate-selection review.

## Cask migration (implemented locally 2026-11-10)

The owner directed the cask migration in this session (“start cask migration”), using Exa/Context7 research. Implementation and local nonpublishing proof are authorized; no tag, release, tap push or live Homebrew install is.

### Why the artifact name had to change

GoReleaser `v2.14.0` accepts only a string list for `homebrew_casks.binaries`; a map with `name`/`target` is rejected by `goreleaser check` (`cannot unmarshal !!map into string`). A Homebrew cask can rename a `binary` with `target:`, but GoReleaser does not expose that field at this version. The archive member is therefore named `agentic-sdd`, and `binaries` lists `agentic-sdd` and `agentic-sdd-tui` directly.

### Generated cask (snapshot, not published)

`dist/homebrew/Casks/agentic-sdd.rb`:

```ruby
cask "agentic-sdd" do
  binary "agentic-sdd"
  binary "agentic-sdd-tui"

  on_macos do
    url "https://github.com/nawodyaishan/agentic-sdd/releases/download/v0.6.0/agentic-sdd_#{version}_darwin_all_homebrew.tar.gz"
    sha256 "…"
  end

  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/agentic-sdd"]
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/agentic-sdd-tui"]
    end
  end
end
```

### Quarantine and signing

Casks download into a quarantined Caskroom. The binaries are not signed or notarized, so macOS Gatekeeper can block them. GoReleaser's documented mitigation is a post-install `xattr -dr com.apple.quarantine` hook; signing/notarization is the alternative but needs Apple credentials. The hook is deliberately limited to those two `xattr` calls, and the static config test rejects install-time compilation or binary execution. A real macOS walkthrough is still required, because that behavior cannot be observed on the Linux build host.

### Cask snapshot proof

Command: `PATH=/tmp/opencode/b6-tools:$PATH task -t Taskfile.experimental.yml release:snapshot` (GoReleaser `v2.14.0`, Go `1.27.1`).

| Check | Result |
|---|---|
| Cask | `dist/homebrew/Casks/agentic-sdd.rb`; `binary "agentic-sdd"` + `binary "agentic-sdd-tui"`; no tap push in snapshot |
| Homebrew archive | `agentic-sdd_0.6.0-SNAPSHOT-974eeca_darwin_all_homebrew.tar.gz`, contains `LICENSE`, `README.md`, `agentic-sdd`, `agentic-sdd-tui` |
| Generic archive | `agentic-sdd_0.6.0-SNAPSHOT-974eeca_darwin_all_generic.tar.gz`, contains `LICENSE`, `README.md`, `agentic-sdd-generic`, `agentic-sdd-tui` |
| Universal slices | `file` reports Mach-O universal with `x86_64` + `arm64` for the CLI and companion |
| Checksums | `dist/checksums.txt` has both archive digests; the cask `sha256` matches the Homebrew archive |
| Cask test stanza | GoReleaser v2.14 emits none; `--help`, `version` and `agentic-sdd-tui --version` must be run in the T16 owner walkthrough |
| macOS-only | The cask URL is inside `on_macos`; GoReleaser v2.14 does not emit `depends_on :macos`, so Linux `brew install --cask` is unsupported in practice but is not statically declared |

### Tap migration sequence (not performed)

Confirmed contract from Homebrew changes: formula→cask migration within a third-party tap requires Homebrew `5.0.6` or newer, a cask that already exists in the tap, and `tap_migrations.json` plus deletion of the formula in the **same pushed tap commit**. Migration runs during a `brew update` that moves the tap to the new commit; it installs the cask and unlinks (but does not uninstall) the old formula, then asks the user to run `brew uninstall --formula agentic-sdd`.

1. Publish a cask-bearing release (GoReleaser pushes `Casks/agentic-sdd.rb`). The formula stops receiving updates from this release onward, so formula users stay on the last formula version until the tap migration lands.
2. In the tap, verify `Casks/agentic-sdd.rb` is present and installable.
3. In one tap push, add `tap_migrations.json` with `{"agentic-sdd": "agentic-sdd"}` and remove `Formula/agentic-sdd.rb`.
4. Verify on macOS with Homebrew `>=5.0.6`: `brew update` migrates, then `brew uninstall --formula agentic-sdd`, `agentic-sdd version`, `agentic-sdd --help`, `agentic-sdd-tui --version`.

Sources: [Homebrew PR #20800](https://github.com/Homebrew/brew/pull/20800), [PR #20932](https://github.com/Homebrew/brew/pull/20932), [PR #21224](https://github.com/Homebrew/brew/pull/21224), [issue #21207](https://github.com/Homebrew/brew/issues/21207), [GoReleaser Homebrew casks](https://goreleaser.com/customization/publish/homebrew_casks/), [Cask Cookbook](https://docs.brew.sh/Cask-Cookbook).

### Limits

- No live release, tag, tap push, token check, `brew install` or macOS execution was performed. The cask and archive were generated on Linux.
- The tap remains formula-only, so the current published install path is unchanged until a cask-bearing release and the tap migration are separately authorized.
