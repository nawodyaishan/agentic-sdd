# Repository guidance

## Scope

This repository holds the canonical Agentic SDD skill files and a Go CLI that installs them into user skill directories. Treat `skills-files/` as the source of truth. Keep `backups/` local and out of Git; it may contain private previous skill content.

## Go layout

- `cmd/agentic-sdd/main.go`: CLI flags, home-directory default, process exit behavior, and interactive backup selection (`backups`, `restore`).
- `internal/skillsync/sync.go`: planning, backup, installation, and rollback (shared by apply and restore).
- `internal/skillsync/files.go`: path checks, tree comparison, and copying.
- `internal/skillsync/manifest.go`: the backup manifest format (additive: format 1 fields plus format 2's location, tool version, timestamps and per-skill entries) and its tree digest.
- `internal/skillsync/restore.go`: listing and validating backups, and restore's planning/apply.
- `internal/skillsync/sync_test.go`, `manifest_test.go`, `restore_test.go`: isolated-home tests; never write to real user skill directories in tests.
- `internal/skillsync/source.go`: `fs.FS`-based source resolution (embedded `skills-files/` by default, `--repo` on disk otherwise).
- `internal/version/version.go`: build-time version metadata, injected via `-ldflags` at release build time; `dev`/`none`/`unknown` defaults for local builds.
- `skillsfiles.go`: root-level `go:embed all:skills-files`, so an installed binary (e.g. via Homebrew) needs no repository checkout.

Keep client destinations and skill selection together in `internal/skillsync`. Prefer Go standard library code and small, explicit functions. Do not add a package or interface solely to move a few lines of code.

## Safe changes

- Preview must remain the default. `--apply` is the only mode that installs skills.
- Back up every existing skill that will be replaced before modifying destinations.
- Keep unrelated skills untouched and reject symlinks or special files in managed skill trees.
- Preserve the no-op behavior: identical skills should not be replaced or backed up again.
- Update README commands and the Makefile whenever the CLI path or flags change.

## Verification

Run `make test`, `make vet`, and `git diff --check` after Go changes. Use a temporary home in tests. `make preview` is a useful read-only check against the local installation. Install the repository's Lefthook checks with `make hooks-install`.

## Evaluation spending and credentials

- Additional model calls through CLI subprocesses, SDKs, HTTP, smoke probes,
  retries, titles or summaries are external usage, not ordinary deterministic
  tests. Batch approval does not authorize them. Require explicit authorization
  for the exact provider/model, data scope and bounded request/token/deadline
  limits. A general suggestion that models are cheap/free is not a billing cap.
- Never clone, export or seed an evaluation workspace from the primary OpenCode
  database/auth store. Use a fresh credential-free environment for offline
  checks; any separately authorized online fixture needs a dedicated scoped
  evaluation credential, supplied by the user through a secret reference.
- No paid fallback, provider probing, suffix removal from `:free`, model
  escalation, warming or background inference. Use the exact reviewed model ID;
  free price, tool capability and private-data eligibility are independent checks.
  Failure or uncertainty stops the affected run.
- Paid evaluation requires verified current prices, remaining allowance and a
  dedicated provider-side key cap within the run/month allocation. Include all
  requests and in-flight work. Missing/delayed usage is unknown, not zero; stop
  rather than rely on JSON event totals or a mental estimate. Subscription usage
  also requires limits even when it produces no per-token bill.
- Keep provider controls/config adoption, key rotation/logout and removal of old
  credential-bearing sandboxes user-controlled. Do not change the real home or
  shared service while developing this adapter. See
  `integrations/opencode/safety.md` for incident-derived controls and open actions.

## SDD feature conventions

This repository has no top-level product SRS, technical spec, or roadmap document. Each feature lives in `specs/<nnn-slug>/` (`spec.md`, `plan.md`, `tasks.md`); see `specs/001-cli-quality/` for the current one. The combined human approval for a feature's spec/plan/tasks is recorded once, in that feature's `spec.md` under `## Approval`. Batch state and continuation notes live in that feature's `tasks.md`.

`preview` (default) versus `apply`/`--apply` is this repo's dev-versus-live boundary: approval to develop or test installer code never authorizes running `apply` against a real user home; only an explicit, separately authorized real-home apply does.

## Session handovers

Every session that advances a feature writes one handover to `docs/handovers/<YYYY-MM-DD>_<spec-no>.md` (for example `docs/handovers/2026-10-29_004.md`) before it ends or before another agent picks the work up. A second session on the same day for the same feature appends a letter: `<date>_<spec-no>b.md`.

Structure, in this order: batch position and next authorized batch; what the session did; decisions the user made; decisions still open with what each blocks; verified facts the next session must not re-derive; open blockers by ID; the next permitted action; guardrails to preserve; the first commands to run.

- One file per session, not an append-only log. `specs/<nnn-slug>/tasks.md` stays the authoritative batch ledger and `spec.md` §`## Approval` the only approval record; a handover never creates approval, changes batch state or substitutes for a continuation note.
- Link to specs, tasks, evidence and out-of-repo reports by path; do not restate them.
- Never copy credentials, tokens, account records, raw forensic traces or other secret material into a handover.
- Keep it readable in a couple of minutes; delete or fold a handover once its open decisions are resolved.

## Release and publish

`.goreleaser.yml` builds darwin (amd64+arm64, merged into a universal binary) and publishes a Homebrew formula to `nawodyaishan/homebrew-tap`, driven by `.github/workflows/release.yml` on a `v*` tag push. `make tag V=vX.Y.Z MSG="..."` creates the tag; `make release` runs a local snapshot build (no publish, no tag required) to verify the pipeline. Writing a real `v*` tag, pushing it, or generating/rotating `HOMEBREW_TAP_TOKEN` are live-system operations requiring their own authorization — developing or testing this pipeline's code does not authorize them.
