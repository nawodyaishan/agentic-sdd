# Plan: Homebrew package publishing

Spec revision used: `spec.md` as redrafted 2026-09-25 (draft/pending) — directly mirrors quiz-snap.

## Approach

Port quiz-snap's already-working release pipeline directly rather than redesigning it: `internal/version`, `.goreleaser.yml`, `.github/workflows/release.yml`, `.github/workflows/ci.yml`, and the `Makefile` `verify`/`build-darwin`/`tag`/`release` targets are copied in shape from `/Users/nawodyaishan/Documents/GitHub/quiz-snap`, adjusted only for two real differences: (1) this repo's `AGENTS.md` mandates stdlib `flag`, not cobra, so the `version` command is added to the existing flag-based switch in `cmd/agentic-sdd/main.go`, not a new CLI framework; (2) `agentic-sdd` ships a runtime data directory (`skills-files/`) that quiz-snap has no equivalent of, so embedding it via `go:embed` is the one genuinely new piece of engineering in this feature — everything else is a faithful port. The tap repository is `nawodyaishan/homebrew-tap`, reused as-is from quiz-snap's own `.goreleaser.yml`, not a new tap.

## Affected components

- `cmd/agentic-sdd/main.go` — add a `version` case to the existing command switch (alongside `preview`/`apply`/`help`), printing `agentic-sdd <version>` / `commit:` / `date:` / `go version:`, matching quiz-snap's `internal/cli/version.go` output shape; change the default skills source resolution so an unset `--repo` uses the embedded tree instead of defaulting to `.`.
- `internal/version` (new package) — `Version = "dev"`, `Commit = "none"`, `Date = "unknown"`, `GoVersion = "unknown"`, same doc-comment convention as quiz-snap's `internal/version/version.go`, set via `-ldflags -X` at release build time.
- `internal/skillsync` — embed wiring for `skills-files/` (mechanics below) and a `Sync` signature change to accept an `fs.FS` uniformly: embedded tree by default, `os.DirFS(repo)` when `--repo` is explicitly set. No quiz-snap equivalent to port from; this is new engineering, not a port.
- `.goreleaser.yml` (new, repo root) — ported from quiz-snap's, darwin-only `amd64`+`arm64`, `universal_binaries`, same archive/checksum/changelog shape, `brews:` block pointed at `nawodyaishan/homebrew-tap`, no `caveats` (nothing to port — quiz-snap's caveats are about macOS TCC permissions specific to screen capture, not applicable here).
- `.github/workflows/release.yml` (new) — ported from quiz-snap's: `environment: release`, `permissions: contents: write, id-token: write`, `Verify`/`Get Go version`/`Validate Homebrew tap token`/`Run GoReleaser` steps in the same order.
- `.github/workflows/ci.yml` (new, optional per spec) — ported from quiz-snap's quality-gate shape, minus the `golangci-lint` step (no lint config in this repo — not adding one as a side effect).
- `Makefile` — add `mod-verify`, `tidy-check`, `build`, `build-darwin`, `verify`, `tag`, `release` targets alongside the existing `preview`/`apply`/`test`/`vet`/`docker-e2e`/`hooks-install`, matching quiz-snap's target names/behavior.
- `README.md`, `AGENTS.md` — install/release documentation pointers, matching quiz-snap's documented flow (`make tag V=... MSG=...` then `git push origin vX.Y.Z`).
- `cmd/agentic-sdd/main_test.go` — coverage for `version` and embedded-source `preview` from a directory with no `skills-files/`.

## Design decisions

1. **Direct port, not redesign, for everything with a quiz-snap equivalent.** The user explicitly asked to refer directly to quiz-snap's finely-working version; no alternative structures were evaluated for `.goreleaser.yml`, the release workflow, or the Makefile targets — they're copied and adapted only where this repo's code genuinely differs (flag vs. cobra, no TCC caveats, no lint config).
2. **Reuse `nawodyaishan/homebrew-tap` as-is.** Confirmed from quiz-snap's own `.goreleaser.yml` (`repository.name: homebrew-tap`, not a per-project tap name) — GoReleaser's `brews:` supports multiple formulas in one tap via distinct `Formula/*.rb` files. This removes the earlier "invent a tap name" placeholder; the only remaining manual step is confirming the repo and token are actually usable (spec Acceptance 5, Outside scope).
3. **Embed with `--repo` override, not embed-only** (unchanged from the prior draft, no quiz-snap equivalent to defer to). Preserves the existing `make preview`/`make apply` checkout-based workflow while making the distributed binary self-contained.
4. **darwin-only, no hedge.** The prior draft treated this as an open question ("confirm at plan time"); mirroring quiz-snap directly resolves it — quiz-snap targets darwin only and this port keeps that scope rather than introducing Linux support quiz-snap doesn't have. If Linux support is wanted later, it's new scope, not part of this port.
5. **`ci.yml` is in scope but optional-if-time**, since the spec marks it as such — it isn't part of "Homebrew publishing" per se, but it's what quiz-snap runs before every release and is a small, low-risk direct port; include it in B2 rather than deferring to a separate feature, since skipping it would leave this repo without the quality gate the release workflow assumes exists.

## Risks

- **Embed mechanics risk (real, no quiz-snap equivalent):** Go's `//go:embed` requires the embedded path to be within (or below) the directory of the package that embeds it; `skills-files/` sits at repo root alongside `cmd/` and `internal/`, not inside a Go package directory. Resolve in T2 (e.g., an embed file in a root-level Go package, or restructuring so `internal/skillsync` can embed a relative path satisfying the "same subtree" rule) — this is implementation work, not a design decision made here.
- **Tap repo/token validity is unverified (accepted, matches spec's Outside scope and Acceptance 5):** the agent cannot confirm `nawodyaishan/homebrew-tap` is reachable or that a valid `HOMEBREW_TAP_TOKEN` exists for this repo — quiz-snap's own workflow already depends on both existing, but this repo's secret must be provisioned separately even if the tap is shared. Mitigated by porting the same `Validate Homebrew tap token` fail-fast step quiz-snap uses, so a misconfiguration fails loudly on the first real tag push instead of silently.
- **`make verify`'s `tidy-check` is currently vacuous (low):** this repo has no `go.sum` (no dependencies yet), so `git diff --exit-code -- go.mod go.sum` is trivially satisfied. Not a problem — it's forward-compatible with quiz-snap's target and will do real work once dependencies are added.
- **Silent default-source regression (low, testable):** changing `--repo`'s default behavior could break a script relying on `--repo .` implicitly working from inside a checkout. Mitigated: `os.DirFS(".")` still contains `skills-files/` when run from a checkout root, so explicit `--repo .` keeps working; only the *unset* default changes.

## Specialists and tools

- **`golang-pro`** — all Go changes (`internal/version`, `go:embed` wiring, `Sync` refactor, `version` command, ldflags). Ordinary Go CLI/library work matching this repo's existing Go patterns.
- **No additional specialist** for `.goreleaser.yml`, the GitHub Actions YAML, and the Makefile/docs — this is a direct, already-verified-working port from quiz-snap, not novel configuration design; repository guidance plus the ported reference is adequate.
- **CodeGraph** — this repo has `.codegraph/`; use it during implementation to trace `Sync`'s callers/callees before refactoring its source-resolution signature.
- **Repository gates** — `make test`, `make vet`, `git diff --check`, plus the new `make verify` (`mod-verify tidy-check vet test build-darwin`), `goreleaser check`, and `make release` (snapshot) for B2.
- Context7 (`/goreleaser/goreleaser`) and Exa were already consulted for the `brews:`/Actions-token reference material behind quiz-snap's own working config; re-consult only if a concrete GoReleaser question comes up that quiz-snap's config doesn't already answer.

## Next document

`tasks.md`.
