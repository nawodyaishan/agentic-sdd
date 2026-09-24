# Tasks: Homebrew package publishing

Status: implemented — both batches complete and locally verified; see the combined approval record in `spec.md`. Neither batch has been human-reviewed yet; treat both as `awaiting human review`.

## Batch B1 — self-contained CLI (version + embedded skills)

Outcome: `agentic-sdd`, built as a plain binary with no `skills-files/` on disk beside it, reports its version (in the format quiz-snap's `version` command uses) and correctly previews/applies skills from any working directory. This is the code change the release pipeline in B2 depends on and reviews independently of CI/YAML.

### T1 — `internal/version` package + `version` command

Port `internal/version` from `/Users/nawodyaishan/Documents/GitHub/quiz-snap/internal/version/version.go` verbatim in shape: `Version = "dev"`, `Commit = "none"`, `Date = "unknown"`, `GoVersion = "unknown"`, same doc comment convention. Add a `version` case to `cmd/agentic-sdd/main.go`'s existing command switch (alongside `preview`/`apply`/`help`) printing the same four lines quiz-snap's `internal/cli/version.go` prints, adapted to this repo's stdlib-`flag` dispatch (no cobra — matches `AGENTS.md`). Extend `main_test.go` for the command and its no-ldflags default output. Done when `go build ./cmd/agentic-sdd && ./agentic-sdd version` prints the default (`dev`/`none`/`unknown`) values, and building with `-ldflags "-X agentic-sdd/internal/version.Version=v9.9.9 ..."` reflects injected values.

### T2 — embed `skills-files/` with `--repo` override

Resolve the embed-mechanics risk from `plan.md` (Go's `//go:embed` requires the embedded tree to sit under the embedding package's own directory — no quiz-snap equivalent to port, this is new engineering) — CodeGraph-trace `Sync`'s current callers/signature first (`internal/skillsync/sync.go:36`, `internal/skillsync/sync.go:46`) before changing it. Refactor `Sync`'s source resolution to accept an `fs.FS`: default to the embedded `skills-files/` tree when `--repo` is unset, and continue honoring an explicit `--repo` (via `os.DirFS`) exactly as today. Done when running the built binary's `preview`/`apply` from an empty temp directory (no `skills-files/` anywhere nearby) lists/installs the embedded skills correctly, and `go run ./cmd/agentic-sdd preview` / `--repo .` from a checkout still behaves as before.

### T3 — regression coverage

Add/extend `internal/skillsync` and `cmd/agentic-sdd` tests for: embedded-source preview from a directory without `skills-files/`, explicit `--repo` override still working, `version` command output, and the existing backup/rollback/no-op contract unaffected. Done when `make test`, `make vet`, and `git diff --check` pass.

## Batch B2 — port quiz-snap's release pipeline (GoReleaser, GitHub Actions, Makefile, docs)

Depends on: B1 complete (release artifacts need a real version command and a self-contained binary to be meaningful). Outcome: `.goreleaser.yml`, `.github/workflows/release.yml`, `.github/workflows/ci.yml`, and `Makefile` targets ported directly from quiz-snap and locally verifiable, ready for the manual tap-token prerequisite in `spec.md`'s Outside scope.

### T4 — `Makefile` targets

Add `mod-verify`, `tidy-check`, `build`, `build-darwin`, `verify` (`mod-verify tidy-check vet test build-darwin`), `tag` (`V=`/`MSG=` annotated tag), and `release` (`GOVERSION=$(shell go version | awk '{print $$3}') goreleaser release --snapshot --clean`) targets, matching quiz-snap's `Makefile` names and behavior; leave existing `preview`/`apply`/`test`/`vet`/`docker-e2e`/`hooks-install` untouched. Done when `make verify` runs cleanly end to end (all sub-targets pass) on the current code state.

### T5 — `.goreleaser.yml`

Port quiz-snap's `.goreleaser.yml` directly: `before.hooks: [go mod tidy]`; `builds` for `cmd/agentic-sdd` (`darwin` only, `amd64`+`arm64`, `CGO_ENABLED=0`, `-trimpath`, `mod_timestamp`, ldflags injecting the four `internal/version` fields + `GOVERSION`); `universal_binaries` (`replace: true`); `archives` (tar.gz, same name template); `checksum`; `changelog` (same exclude filters); `brews:` (`repository.owner: nawodyaishan`, `repository.name: homebrew-tap`, `repository.token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"`, `directory: Formula`, `homepage`, `description` for this project, `license: MIT`, `install`, `test` running `--help` and `version`) — no `caveats` block. Done when `goreleaser check` passes and `make release` produces `dist/homebrew/Formula/agentic-sdd.rb` with correct `url`/`sha256`/`install`/`test`.

### T6 — `.github/workflows/release.yml`

Port quiz-snap's release workflow directly: trigger on `push: tags: ["v*"]`; `permissions: contents: write, id-token: write`; job `environment: release`; steps in the same order — checkout (`fetch-depth: 0`), `actions/setup-go` (`go-version-file: go.mod`), `Verify` (`make verify`), `Get Go version` (export `GOVERSION`), `Validate Homebrew tap token` (`gh api repos/nawodyaishan/homebrew-tap --jq .default_branch` using `HOMEBREW_TAP_TOKEN`), `Run GoReleaser` (`goreleaser/goreleaser-action`, `args: release --clean`, `GITHUB_TOKEN`/`HOMEBREW_TAP_TOKEN`/`GOVERSION` env). Done when the workflow YAML is valid and every step up to the actual GoReleaser publish has been exercised locally (`make verify`, the `gh api` command run manually once a token exists per the manual prerequisite, `make release` standing in for the GoReleaser step).

### T7 — `.github/workflows/ci.yml` (optional per spec, include if time allows)

Port quiz-snap's quality-gate workflow shape (`push`/`pull_request`, `ubuntu-latest`, `make mod-verify`, `make vet`, `make test`, `make build-darwin`) minus the `golangci-lint` step (no lint config in this repo — not adding one here). Done when the workflow YAML is valid and its steps pass locally via the equivalent `make` targets. Task-specific exception to `plan.md`'s "no additional specialist" default: none — still no additional specialist needed, same reasoning as T5/T6.

### T8 — documentation

Update `README.md` with the `brew install nawodyaishan/tap/agentic-sdd` path and the release flow (`make tag V=vX.Y.Z MSG="..."` then `git push origin vX.Y.Z`), matching quiz-snap's documented flow. Add a pointer in `AGENTS.md`'s "SDD feature conventions" section to `.goreleaser.yml`/`release.yml` as the release/publish mechanism. Done when both docs are internally consistent with the actual T5/T6 content.

## Verification and handoff

Specialist for T1–T3: `golang-pro`, per `plan.md`. T4–T8 use no additional specialist (direct port of already-working config/docs; plan's explicit assignment). Tools: CodeGraph for T2's pre-refactor trace; repository gates (`make test`, `make vet`, `git diff --check`, new `make verify`) for every batch; `goreleaser check` and `make release` (snapshot) as this feature's release-specific verification for B2.

Record changed paths and check results here after each batch, then set that batch to `awaiting human review`. Do not push a real `v*` tag, run `gh api` against the real tap with production credentials, or generate/rotate `HOMEBREW_TAP_TOKEN` from this workflow — those remain the manual prerequisite documented in `spec.md`'s Outside scope.

## Batch B1 result — state: awaiting human review

Implemented per the user's "go ahead till complete 002 implementation" authorization, continuing straight into B2 without an intermediate stop.

Changed paths:
- `internal/version/version.go` (new) — `Version`/`Commit`/`Date`/`GoVersion`, `dev`/`none`/`unknown` defaults.
- `skillsfiles.go` (new, package `skillsfiles`) — root-level `//go:embed all:skills-files`; required because Go's embed directive must sit at or above the embedded tree, and `skills-files/` stays at repo root per `AGENTS.md`.
- `internal/skillsync/source.go` (new) — `sourceFS`, `installedDirMode`/`installedFileMode`, `checkTreeFS`, `sameSourceTarget`, `copyFromSource`.
- `internal/skillsync/sync.go` — `Sync` resolves source via `sourceFS(repo)`; `backupRoot` defaults to `<home>/.agentic-sdd/backups` when `repo == ""` (embedded), else `<repo>/backups` (unchanged); `discoverSkills`/`planChanges`/`installChanges` now `fs.FS`-based.
- `internal/skillsync/files.go` — removed superseded `sameTree` and its now-unused `bytes`/`crypto/sha256` imports (logic moved to `source.go`'s `sameSourceTarget`).
- `cmd/agentic-sdd/main.go` — added `version` command + `--version` flag (`printVersion`), `--repo` default changed from `"."` to `""` (embedded-by-default), usage text updated.
- `cmd/agentic-sdd/main_test.go` — added `TestRunVersion`, `TestRunPreviewAndApplyFromEmbeddedSource`.

Checks (all against this code state):
- `go build ./...`, `go vet ./...` — clean.
- `go test ./...` — all pass, including the two new tests; existing `TestRunPreviewAndApplyCompatibility` (explicit `--repo`) and all of `internal/skillsync/sync_test.go` unchanged and passing, confirming the `--repo`/backup-path/no-op/rollback contract is unaffected.
- `git diff --check` — clean.
- Manual smoke test from `/tmp` (no `skills-files/` nearby): `agentic-sdd version` prints `dev`/`none`/`unknown`; `preview`/`apply --home <tmp>` install all embedded skills and create the first backup under `<home>/.agentic-sdd/backups`.

Next action: human review of the embed/backup-path refactor and the version command before authorizing any live tag/publish.

## Batch B2 result — state: awaiting human review

### T4 — Makefile

Added `mod-verify`, `tidy-check`, `build`, `build-darwin`, `verify`, `tag`, `release` targets (ported from quiz-snap's `Makefile`, same names/behavior), plus `BINARY_NAME`/`CMD_DIR` vars; existing `preview`/`apply`/`test`/`vet`/`docker-e2e`/`hooks-install` untouched. `make verify` (`mod-verify tidy-check vet test build-darwin`) passes cleanly end to end.

### T5 — `.goreleaser.yml`

Ported directly from quiz-snap: darwin-only amd64+arm64 build of `./cmd/agentic-sdd`, `universal_binaries` (replace: true), ldflags injecting `agentic-sdd/internal/version.{Version,Commit,Date,GoVersion}`, `brews:` pointed at `nawodyaishan/homebrew-tap` with this project's own homepage/description, no `caveats` block (no macOS TCC permissions to document, unlike quiz-snap). `goreleaser check` reports the same pre-existing `brews` deprecation warning quiz-snap's own working config reports with the same locally installed goreleaser version (config still valid) — not a defect in the port. `GOVERSION=$(go version | awk '{print $3}') goreleaser release --snapshot --clean` succeeded, producing `dist/homebrew/Formula/agentic-sdd.rb` with correct `url`/`sha256`/`install`/`test`; the formula's `test` block commands (`agentic-sdd --help`, `agentic-sdd version`) were run directly against the built binary and both exit 0.

### T6 — `.github/workflows/release.yml`

Ported directly from quiz-snap: `push: tags: ["v*"]` trigger, `permissions: contents: write, id-token: write`, `environment: release`, steps in the same order (checkout `fetch-depth: 0`, setup-go, `make verify`, Go-version export, `gh api repos/nawodyaishan/homebrew-tap` tap-token validation, `goreleaser/goreleaser-action` `release --clean`). YAML validated by inspection; every step up through GoReleaser was exercised locally via `make verify` and the snapshot release above (the `gh api` tap-token-validation and real publish steps were not run — they require the not-yet-created `HOMEBREW_TAP_TOKEN`, out of this session's scope per `spec.md`).

### T7 — `.github/workflows/ci.yml`

Ported quiz-snap's quality-gate shape minus `golangci-lint` (no lint config in this repo, matching `tasks.md`'s stated exception) and minus the tidy-check-disabled comment (agentic-sdd is not darwin-only source, so `tidy-check` runs on the `ubuntu-latest` runner instead of being skipped). `push`/`pull_request` triggers, `make mod-verify`, `make tidy-check`, `make vet`, `make test`, `make build-darwin` — all four `make` targets pass locally on this code state.

### T8 — documentation

`README.md`: added an "Install" section (`brew install nawodyaishan/tap/agentic-sdd`), a "Releasing" section (`make verify`, `make tag`, `git push origin vX.Y.Z`, `make release` for local snapshot verification), a `version` row in the CLI reference table, corrected the `--repo` flag description and the backup-location paragraph (embedded-default vs. `--repo`-relative), and added the new files to the repository layout listing. `AGENTS.md`: documented the three new/changed Go files under "Go layout" and added a "Release and publish" section naming `.goreleaser.yml`/`release.yml`, the `make tag`/`make release` targets, and reiterating that developing/testing this pipeline does not authorize a live tag push or token generation.

Checks: `make verify` passes end to end on the final code state (mod-verify, tidy-check, vet, test, build-darwin); `git diff --check` clean; `goreleaser check` + snapshot `make release` verified above under T5.

Not run, and explicitly out of scope for this session per `spec.md`'s Approval/Outside-scope sections: pushing a real `v*` tag, a real `gh api` call against `nawodyaishan/homebrew-tap` with production credentials, generating/rotating `HOMEBREW_TAP_TOKEN`, or a live GoReleaser publish.

Next action: human review of the full B1+B2 diff; once satisfied, the remaining manual prerequisite (creating `nawodyaishan/homebrew-tap` and `HOMEBREW_TAP_TOKEN`) and a real tag push are separate, explicitly-authorized follow-up actions, not part of this batch.
