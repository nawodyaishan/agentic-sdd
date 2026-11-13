# Repository guidance

## Scope

This repository holds the canonical Agentic SDD skill files and the Go CLI and TUI that install them into user skill directories. Treat `skills-files/` as the source of truth. Keep `backups/` local and out of Git; it may contain private previous skill content.

## Command and source names

- Canonical installed commands are `asdx` (TUI) and `asdx-core` (CLI). `agentic-sdd` and `agentic-sdd-tui` remain legacy wrappers for one deprecation window; `agentic-sdd` must never silently become the TUI.
- Source paths stay `cmd/agentic-sdd` (CLI) and `experiments/tui/cmd/tui-companion` (TUI). A source path is not an installed command name.
- The repository, GitHub release endpoint and Homebrew cask keep the `agentic-sdd` name. State and environment names remain `~/.agentic-sdd/` and `AGENTIC_SDD_*`.

## CodeGraph index

CodeGraph is a default code-discovery tool for this repository. Treat it as
session infrastructure, not as product scope or a source of approval.

- **Use it first.** If the CodeGraph MCP server is available, call
  `codegraph_explore` before grep/read for indexed code. It is Read-equivalent
  and returns symbol source, call paths and blast radius in one call. Use the
  CLI fallback (`codegraph explore`, `codegraph node`, `codegraph callers`,
  `codegraph impact`) when the MCP server is unavailable.
- **Bootstrap once.** Check `codegraph status` or `.codegraph/` first. If the
  index is absent, `codegraph` is installed and this is an ordinary project
  checkout (not a home directory, filesystem root or private vault), run
  `codegraph init -y`, then verify with `codegraph status --json`. If the CLI is
  absent or initialization fails, report it and continue with targeted
  Read/Grep; do not install tools silently.
- **Keep it synced.** The MCP server's watcher normally indexes edits within
  about a second. After a bulk edit, a branch switch, or before a review/commit,
  run `codegraph sync` and verify `codegraph status`. If a response carries a
  staleness banner, Read the flagged files before relying on them. `codegraph
  unlock` clears a stale lock; `codegraph index` rebuilds from scratch only
  when the database is genuinely corrupt.
- **Keep it local.** `.codegraph/` is machine-local state and must never be
  committed; it self-ignores, and the root `.gitignore` also excludes it. Do not
  index `backups/`, private homes, credentials or other unmanaged private
  material. Never put secrets or private code in CodeGraph queries.

See [docs/codegraph.md](docs/codegraph.md) for the bootstrap, sync, MCP
continuity and troubleshooting procedure.

## Product vision and engineering principles

[docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md) is the canonical product compass: mission, audiences, non-negotiable engineering principles, and conditional long-term direction.

- Consider it when selecting or scoping a feature, making consequential architectural, security or operational decisions, implementing, and reviewing. Reference the relevant section instead of copying it.
- It guides trade-offs but never overrides an approved feature's `spec.md`/`plan.md`/`tasks.md`, harness evidence, security policy, or an explicit human authorization. Surface a conflict for a human decision.
- Keep its status honest: the vision's horizons are conditional aspirations, not current supported capability. Current support lives in README's release scope, the feature records and `harness/` evidence.
- Changing the vision is a public strategy decision, separate from implementing a feature.

## Go layout

- `cmd/agentic-sdd/main.go`: canonical `asdx-core` CLI flags, home-directory default, process exit behavior, and interactive backup selection (`backups`, `restore`).
- `cmd/agentic-sdd/tui.go` and `cmd/internal/tuirun/`: explicit `asdx-core tui` companion resolution, build-identity handshake, and signal/exit forwarding. The canonical TUI source is the nested `experiments/tui` module.
- `internal/skillsync/sync.go`: planning, backup, installation, and rollback (shared by apply and restore).
- `internal/skillsync/files.go`: path checks, tree comparison, and copying.
- `internal/skillsync/manifest.go`: the backup manifest format (additive: format 1 fields plus format 2's location, tool version, timestamps and per-skill entries) and its tree digest.
- `internal/skillsync/restore.go`: listing and validating backups, and restore's planning/apply.
- `internal/skillsync/sync_test.go`, `manifest_test.go`, `restore_test.go`: isolated-home tests; never write to real user skill directories in tests.
- `internal/skillsync/source.go`: `fs.FS`-based source resolution (embedded `skills-files/` by default, `--repo` on disk otherwise).
- `internal/fleet/`: Feature 014 offline fleet core (stdlib only) — `fleet.go`/`parse.go` strict personal config, `resolve.go` source resolution, `render.go` deterministic rendering, `plan.go` plan identity and lock, `manifest.go` fleet provenance sidecar, and `doctor.go` opt-in local-only model-route diagnostics.
- `internal/execution/`: Feature 016 typed execution service (stdlib only) — repository identity, private workspace and locks, checkpoint state/freshness, task-brief extraction, strict record ingestion, check evidence, and review-package assembly; rendering and flag parsing stay in `cmd/agentic-sdd` and it must not import CLI or UI packages.
- `internal/buildinfo/`: product names, protocol revision, build-identity checks and distribution markers shared by the CLI and TUI pair.
- `internal/version/version.go`: build-time version metadata, injected via `-ldflags` at release build time; `dev`/`none`/`unknown` defaults for local builds.
- `skillsfiles.go`: root-level `go:embed all:skills-files`, so an installed binary (e.g. via Homebrew) needs no repository checkout.

Keep client destinations and skill selection together in `internal/skillsync`. Prefer Go standard library code and small, explicit functions. Do not add a package or interface solely to move a few lines of code.

## Safe changes

- Preview must remain the default. `--apply` is the only mode that installs skills.
- Back up every existing skill that will be replaced before modifying destinations.
- Keep unrelated skills untouched and reject symlinks or special files in managed skill trees.
- Preserve the no-op behavior: identical skills should not be replaced or backed up again.
- Update README commands and the Taskfile whenever the CLI path or flags change; keep canonical and legacy command names consistent.

## Execution run artifacts

`.agentic-sdd/runs/` is private execution scratch written by the `asdx-core run`, `asdx-core task` and `asdx-core review` commands (Feature 016): a self-ignoring tree that keeps `git status` clean without editing the root `.gitignore`. Keep it out of Git, backups and any upload; remove a run with `asdx-core run clean`. A run, checkpoint, check log or review package is an observation, never approval, batch state, verification, reviewer isolation or a sandbox. `spec.md`'s approval section and `tasks.md` stay authoritative. See [docs/execution-runs.md](docs/execution-runs.md).

## Verification

Run `task test`, `task vet`, and `git diff --check` after Go changes. Use a temporary home in tests. `task preview` is a useful read-only check against the local installation. Install the repository's Lefthook checks with `task hooks-install`.

For skill or documentation changes, also run the explicitly selected static suites:

```sh
python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py
python3 -B -m unittest discover -s tests/harness -p test_research_workflow.py
```

They check instruction and fixture coverage, not runtime model, MCP, retrieval or security behavior. CI skips the trusted candidate gate for `docs/**` and `specs/**`-only changes; product content (`skills-files/`, `opencode-pack/`, `README.md`), tests, workflows, `Taskfile.yml` and Go code still trigger it.

## SDD feature conventions

This repository has no top-level product SRS or technical specification; long-lived product strategy lives in the [vision](docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md). Each feature lives in `specs/<nnn-slug>/` (`spec.md`, `plan.md`, `tasks.md`); see `specs/018-research-evidence-handoff/` for the latest accepted example and `specs/019-cli-tui-entrypoint/` for the current CLI/TUI naming records. A feature `spec.md` may link the vision by section where that explains intent or audience, but does not copy it or treat it as acceptance scope. The combined human approval for a feature's spec/plan/tasks is recorded once, in that feature's `spec.md` under its approval section (`## Approval` or a numbered equivalent such as `## 9. Approval`). Batch state and continuation notes live in that feature's `tasks.md`. Status summaries in README, docs, handovers and feature task rows are pointers; they do not create approval or batch authorization.

`preview` (default) versus `apply`/`--apply` is this repo's dev-versus-live boundary: approval to develop or test installer code never authorizes running `apply` against a real user home; only an explicit, separately authorized real-home apply does.

## Session handovers

Every session that advances a feature writes one handover to `docs/handovers/<YYYY-MM-DD>_<spec-no>.md` (for example `docs/handovers/2026-10-29_004.md`) before it ends or before another agent picks the work up. A second session on the same day for the same feature appends a letter: `<date>_<spec-no>b.md`.

Structure, in this order: batch position and next authorized batch; what the session did; decisions the user made; decisions still open with what each blocks; verified facts the next session must not re-derive; open blockers by ID; the next permitted action; guardrails to preserve; the first commands to run.

- One file per session, not an append-only log. `specs/<nnn-slug>/tasks.md` stays the authoritative batch ledger and `spec.md`'s approval section the only approval record; a handover never creates approval, changes batch state or substitutes for a continuation note.
- Link to specs, tasks, evidence and out-of-repo reports by path; do not restate them.
- Never copy credentials, tokens, account records, raw forensic traces or other secret material into a handover.
- Keep it readable in a couple of minutes; delete or fold a handover once its open decisions are resolved.

## Release and publish

`.goreleaser.yml` builds darwin (amd64+arm64, merged into four universal binaries in one paired archive) and publishes a Homebrew cask to `nawodyaishan/homebrew-tap`, driven by `.github/workflows/release.yml` on a `v*` tag push. The cask links the four shipped names from the paired archive (`asdx`, `asdx-core`, `agentic-sdd`, `agentic-sdd-tui`); the binaries are unsigned, so a post-install hook removes macOS quarantine. `task tag V=vX.Y.Z MSG="..."` creates the tag; `task release` runs a local snapshot build (no publish, no tag required) to verify the pipeline. Writing a real `v*` tag, pushing it, or generating/rotating `HOMEBREW_TAP_TOKEN` are live-system operations requiring their own authorization — developing or testing this pipeline's code does not authorize them. Moving existing formula users to the cask also requires a reviewed tap commit (`Casks/agentic-sdd.rb` present, then `tap_migrations.json` plus removal of `Formula/agentic-sdd.rb` in one push); see `specs/013-unified-tui-experience/b6-packaging-research.md`. Canonical installed commands are `asdx` (TUI) and `asdx-core` (CLI); `agentic-sdd` and `agentic-sdd-tui` remain legacy wrappers for one deprecation window, and the Homebrew cask and GitHub repository/release endpoint keep the `agentic-sdd` name. State and environment names (`~/.agentic-sdd/`, `AGENTIC_SDD_*`) are unchanged.

## Temporary 008 validation scope

Temporary fence for Feature 008 test and validation selection. It applies until the owner explicitly changes it, and it grants no approval, batch authorization, ledger or orchestration rights.

- **Do not select** excluded or blocked feature parts in any test or validation run: Feature 005 blocked native ACP, admission, online and runtime-certification experiments; Feature 006 native, profile, dispatch, pilot and C2-mitigation work; Feature 007 historical migration, adoption and old-base transition exercises. Their records stay historical evidence, not work queues.
- **Do keep selecting** the maintained supported core: shipped portable skill assets, package/installer distribution and integrity checks, check-runner and evidence/receipt checks, the current `Taskfile.yml` commands, and the trusted-CI checks.
- Default `task test`, `task vet`, Lefthook pre-commit hooks, and `harness/checks.json` now select the 11 supported-core packages (`.`, `./internal/version`, `./internal/skillsync`, `./internal/harnesscheck`, `./internal/harnessevidence`, `./internal/appops`, `./internal/buildinfo`, `./internal/fleet`, `./internal/execution`, `./cmd/agentic-sdd`, `./cmd/sdd-check`) with sanitized environment settings, while `Taskfile.experimental.yml` holds separated opt-in experimental tasks. Never add a skip, filter, tolerance or exclusion to make an aggregate suite pass. When an aggregate suite would include excluded validations, run the explicitly named core selection for scoped evidence and report the required aggregate as `not_run` or `blocked` with its reason; a gate conflict stops for a human decision rather than weakening the gate.
- `excluded` and `not_run` are not `pass`. Claim no model, native, online or external-evaluation behavior, and never write to a real user home.
- Select Python checks explicitly by file pattern (for example `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py`); broad discovery across `tests/harness/` is not an approved selection.
- A preserved or committed 005/006/007 work-in-progress change stays out of scope: being committed is not evidence and does not re-enable that work.

Canonical sources: [spec 008](specs/008-core-release-quality/spec.md) §Approval, [plan 008](specs/008-core-release-quality/plan.md) §1 deprecated-feature work fence, and [tasks 008](specs/008-core-release-quality/tasks.md) for batch state. They own scope, approval and batch state; this section only restates the selection fence for test and validation runs.
