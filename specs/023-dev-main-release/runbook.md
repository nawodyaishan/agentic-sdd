# 023 owner runbook — ship a release from `dev`

Solo owner `nawodyaishan` (GitHub account ID `50957846`). Describes the policy approved in [spec r3](spec.md). Settings below are **hosted**; nothing in the repository enforces them. Evidence from the disposable rehearsal is in [b3-rehearsal.md](b3-rehearsal.md).

## Ship a version

1. Work on `dev`. Push directly; each push runs `Dev wake` then `Dev gate`. Red is visible, never promoted.
2. Raise `release-version` on `dev` to the next stable `vX.Y.Z` (numeric, no leading zeros, above every existing tag).
3. Open one `dev` → `main` PR. `Promotion admission` must pass: owner-authored, same repo, current tips, release-version increased, green `dev-gate` on the head.
4. Merge with a **merge commit** (two parents). Never squash or rebase: the publisher reads the two parents.
5. Push the matching tag on that merge commit: `git tag vX.Y.Z <merge-sha> && git push origin vX.Y.Z`. This is the ship action.
6. `Release build` → `Release publish`: `preflight` re-verifies the tag event, the PR and the version, and runs a fresh isolated gate; only then `publish` writes the release and the tap.

## Hosted settings (apply in this order; confirmation last)

| Order | Setting | Value |
|---|---|---|
| 1 | `dev` branch ruleset | Block deletion and force pushes. Require a PR with 1 approval, dismiss stale reviews, last-push approval; bypass **always** for the Repository admin role (owner direct push). |
| 2 | `main` ruleset | Block deletion and force pushes. Require a PR (0 approvals). Required status `promotion-admission` from GitHub Actions, up to date. **No bypass actors.** |
| 3 | `v*` tag ruleset A | Restrict creation; bypass for the Repository admin role only. |
| 4 | `v*` tag ruleset B | Restrict update and deletion; **no bypass**. |
| 5 | Files on both branches | `release-version`, `dev-*.yml`, release workflows and scripts must exist on `main` before the rulesets rely on them. |
| 6 | `github-pages` environment | Limit deployment branches to `main` (`dev` validates only). |
| 7 | `RELEASE_TOOLS_SHA` | Re-pin the `release` environment variable to the reviewed commit that contains the current `scripts/release_*.py`. |
| 8 | `release` environment reviewers | Remove the manual confirmation **last**, only after the hosted exercise passed and you authorize cutover. Until then keep it. |

The Repository admin role is not “only nawodyaishan”: on a personal repository you are the only writer, so the effective guarantee is “only the owner can write”. Re-check collaborators whenever access changes. A contributor PR is reviewed by you; you cannot approve your own PR, which is why your PRs need no second reviewer.

## Accepted and refused (observed or fixture-tested)

| Action | Result |
|---|---|
| Owner pushes `dev` | Accepted (owner bypass); wake and gate run. |
| Promotion PR with an increased version and green head | `promotion-admission` passes. |
| Promotion PR whose version did not increase | Check fails, PR `BLOCKED`. |
| Tag on an ordinary `main` ancestor, a branch named `v…`, or a tag without a merged release PR | Preflight refuses; no release or tap write. |
| Moving or deleting a pushed `v*` tag | Refused by ruleset B. |
| A PR description naming another version | Ignored; the tracked `release-version` decides. |

## Recovery

- Failed `publish`: rerun **all jobs** of the same publish run (preflight evidence is attempt-scoped). Never retag.
- Partial assets: the upload step refuses existing names; delete only that tag's partial assets after review, then rerun.
- Tap already advanced: the downgrade guard refuses on purpose; reconcile the tap by hand.
- Changed `main` after merge: if `release.yml` or the required checks changed before you tag, preflight fails closed; ship promptly or promote a new version.
- Wrong version shipped: correct forward with a new PR and a higher version. Never move a tag.
- Evidence missing or ambiguous: stop; the manual confirmation (if still in place) is the backstop.
