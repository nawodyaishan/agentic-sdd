# Request: consequential architecture verdict for Feature 022 (DRAFT, not sent)

**Status:** draft prepared 2026-11-18. Sending it to a model is a separate owner go.
Role: `sdd-architecture-review-opus` (Claude Agent lane, read-only). A fallback verdict
would be provisional per repository guidance. This request creates no approval.

## Question

Is the delivered 022 design safe to treat as ready, or does it need changes? Return
`Ready`, `Needs changes` or `Blocked`, with findings by ID, severity and the file and
line each rests on. Separate evidence from inference.

## Scope to inspect (candidate = current `origin/main`, record the SHA)

1. **K1 mutation ownership:** `internal/skillsync` target guard and lock order (legacy
   `.fleet.lock` first, then the agent guard; plan 12.10), MCP legacy lock
   `.agentic-sdd/mcp/.lock`, fleet journal order (live pointer before sidecar).
2. **K2 digest and provenance:** `digest_version` codec, legacy backup readers,
   fail-closed sidecar reads, `ConflictRecoveryHint` and the deliberate absence of a
   conflict resolver (plan 13.5).
3. **K3 execution:** process-group liveness, outcome mapping, Darwin classification-only
   residual, Windows fail-closed.
4. **K4 TUI:** composed confirmation gate, single MCP outcome projection, deferred
   generation- and scope-bound startup reads. Known gap: `internal/appops` drops engine
   rollback facts on MCP apply failure.
5. **K5 release:** `Release build` and `Release publish` split, `workflow_run` tag trust,
   `RELEASE_TOOLS_SHA` pin, check-suite own-run exclusion, ancestry and manifest checks,
   self-review risk accepted by the owner (plan 12.9 item 12).
6. **Observation route (T16):** `docs.yml` `workflow_call` entrypoint. Open design
   question: a caller job granting only `contents: read` may fail at startup because
   `docs.yml` also contains a `deploy` job requesting `pages: write` and `id-token: write`.
   Please assess the alternatives in the owner note below.
7. **Gate and qualification:** `harness/qualification.json`, container lane, TUI vendor
   drift check, trusted-base boundary. Note plan 8 differs from `qualification.json` on the
   nested-TUI selection (`-mod=vendor`).

## Evidence available (observations, not approvals)

- Ledger: `tasks.md` section 2 and 8; handover `docs/handovers/2026-11-17c_022.md`.
- Tests at `4424b16`: Go race selection, `task test-core`, `task vet-core`, nested TUI
  vet and tests, vendor check, Python static suites all pass; two clean exports are
  byte-identical on 8 pairs.
- Snapshot (2026-11-18, goreleaser v2.14.0, non-publishing) at `de3bcb2`: one archive,
  four universal Mach-O binaries, checksum verified.
- Container lane: 6 of 6 at `7313566`; not re-run at the final HEAD.
- Not available: live GitHub runs, native macOS execution, a designated trusted base.

## Constraints on the reviewer

Read-only. No edits, no tags, releases, publishes, real-home apply or model calls from
the product. Do not treat tests as proof of runtime or security behaviour. Do not record
approval or acceptance; the owner records those.

## Owner note: T16 callsite alternatives (decision pending)

- **A.** Call `docs.yml` from `release-publish.yml` and grant `pages: write` and
  `id-token: write` to that job. Rejected for now: it widens the workflow that holds the
  tap token.
- **B.** A small separate workflow triggered by `workflow_run` on `Release publish`,
  with `contents: read` only, runs the observation from the default branch. Preferred: no
  token-event suppression and no privilege change to the publisher.
- **C.** Defer; E-09 stays open at integration level.
