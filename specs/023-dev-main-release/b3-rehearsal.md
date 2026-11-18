# B3/T7 hosted rehearsal — 2026-11-18

Disposable private repo `nawodyaishan/asdx-release-rehearsal` only. It was seeded with the B2 workflows and scripts at `release-version` v0.14.0, with no secrets, no `RELEASE_TOOLS_SHA` and no tap token. The `release` environment exists but is unconfigured, so `publish` cannot pass its pin step. **Zero GitHub releases were written.** The real `agentic-sdd` repository, its settings and its tap were not touched. Rulesets mirror [the runbook](runbook.md): `main` PR-only with required `promotion-admission` and no bypass; `dev` owner bypass; `v*` creation by the admin role; `v*` update/deletion with no bypass.

## Defects the hosted runs found (all fixed and test-locked in `agentic-sdd`)

| # | Defect | Why offline tests missed it | Fix commit subject |
|---|---|---|---|
| 1 | `dev-gate` split the workflow name “Dev wake” into two arguments | Fixtures ran helper functions, not the shell step | `fix(dev): pass helper arguments as a bash array` |
| 2 | `promotion-admission` raced a still-running `dev-gate`, recorded a failure and never re-evaluated | No concurrency in fixtures | `fix(dev): wait for the head gate before promotion admission` |
| 3 | Preflight compared the run's `name` to “Release build”, but GitHub reports the rendered `run-name` there, so a genuine owner tag was refused | Hand-written run records | `fix(release): stop checking the run name that run-name replaces` |
| 4 | The commit→PRs endpoint returns the short PR form (no `merged`, no `merged_by`), so a genuine merged promotion was refused | Hand-written PR records | `fix(release): fetch each associated PR in full before the promotion check` |

New tests use hosted record shapes (`tests/harness/test_release_eligibility.py`) and structural locks (`test_dev_gate.py`).

## Observed on GitHub

| Scenario | Result |
|---|---|
| Owner direct push to `dev` | Accepted (owner bypass of the PR rule); `Dev wake` → `Dev gate` from `main`; real containerized gate passed; one `dev-gate` check from `github-actions` on the exact head. |
| Release PR with increased version | `promotion-admission` passed; PR `CLEAN`. |
| Release PR with a non-increasing version | `promotion-admission` failed (“release version did not increase”); PR `BLOCKED`. |
| New commit on the PR head | Old green result did not carry over; gate and admission re-ran on the new head. |
| Owner tag on the merge commit (v0.14.3) | `Release build` succeeded; preflight verified the tag event (run-name `refs/tags/<tag>|true|<owner id>|<sha>`, workflow id and path, workflow blob equals reviewed `main`), the merged PR and the version; the fresh isolated gate completed on the tagged merge; `publish` then failed closed at the unset `RELEASE_TOOLS_SHA`. No release, no tap write. |
| Delete a pushed `v*` tag, move it | Both rejected by the tag ruleset. |
| Branch named like a version | No release workflow started. |
| Tag on an unpromoted `dev` tip (v0.14.4) | `Release build` succeeded (the tag is accepted: creation is the owner's right); preflight refused (“not a two-parent promotion merge”); `publish` skipped; no release or tap write. |

## Not verified (named, not passed)

- **Contributor behavior**: no second GitHub identity. Fork denial, “contributor PR needs your review”, and “new commits dismiss review” are unexercised hosted settings.
- Publish writes (release assets, tap) and the happy `publish` path with a pinned `RELEASE_TOOLS_SHA`; the tap flow is covered only by the offline fake-sink tests.
- Annotated tags on a hosted run (lightweight tags were used; annotated is fixture-tested).
- A different-user workflow rerun.
- The real `agentic-sdd` repository settings, the `github-pages` environment restriction and the `release` environment: not inspected. **Cutover is not authorized and the manual release confirmation stays in place.**
- The `main` ruleset was briefly set to disabled in the rehearsal repo only, to seed fixture content; no equivalent is needed in production.
