# 023 — Develop on `dev`, release from `main`

Draft r2 · 2026-11-18 · Solo maintainer: `nawodyaishan`

## Outcome

Use two long-lived branches. `dev` is where work lands and quality feedback happens. `main` stays the default branch and contains stable promotions only. One release PR describes one public version. The owner can ship without a second reviewer; nobody, including the owner, can turn a failed check into a release.

## Proposed day-to-day flow

| Event | Expected result |
|---|---|
| Owner pushes commits to `dev` | Accept the push without review. Run quality CI. A red tip is visible and cannot be promoted. No release or deployment. |
| Contributor proposes a change | Open a PR against `dev`. Run isolated quality checks. Require the owner's explicit review of the current changes before merge. No direct contributor push to either protected branch. |
| Owner is ready to ship | Open one same-repository `dev` → `main` PR naming the next stable version (e.g. `v0.14.0`) and summarizing the changes. No release branch or version bot. |
| Owner merges the green release PR | `main` advances to the stable result. Record the PR, version and exact merge commit. Closing without merging has no effect. No direct `main` pushes, even by the owner. |
| Owner pushes the matching tag at that commit | The existing tag build and trusted publisher run automatically. No additional human sign-off on this path. Merging alone does not publish; the owner tag is the explicit ship action. |

Owner decision (2026-11-18): a PR determines what ships; pushing the matching owner tag determines when. That push is the final “ship it” action, with no second environment-approval click after cutover.

## Acceptance criteria

**AC1 — Owner-only review exception.** Use the authenticated GitHub account, including its stable account ID where checked programmatically. Git author/committer fields, tagger text, PR titles, labels, actor display names and a fork named `dev` confer no authority. Owner PRs require no second-person review; do not post synthetic approvals. Someone else's commits added after review invalidate that review. An owner-authored PR updated by someone else does not carry the exemption. Bots, collaborators and rerun initiators cannot inherit owner authority. Only the owner may promote. Branch deletion and force pushes are blocked.

**AC2 — One meaningful development gate.** For product changes, reuse the isolated trusted gate and the supported selection in [the checks manifest](../../harness/checks.json): module integrity/tidiness, vet, supported-core tests, Darwin build and diff hygiene. Retain the existing applicable skill, documentation and nested-module checks and the Feature 008 exclusions. PR code runs as candidate data under reviewed tooling, with no release credentials. In particular, a direct `dev` push must not make a changed gate its own trusted base. Cancel obsolete development checks where useful. Never cancel an in-progress publication to save CI time.

**AC3 — Optimize duplicate work, not evidence.** Require a consistently named result from the trusted workflow for the candidate under consideration. Docs/spec-only PRs get an explicit scope decision and applicable docs checks, not a permanently pending required check or invented product-test pass. Mixed changes run product CI. A newer `dev` head or changed `main` base invalidates stale promotion evidence. Reuse green `dev` evidence only when the promoted tree, reviewed gate and inputs match. Otherwise run the gate on the actual promotion result. The publisher must see a successful release-eligibility check on the **tagged `main` SHA** linking the relevant quality evidence; an unrelated green job, base-branch check or green badge on another SHA is insufficient. Missing, cancelled or failed evidence blocks publication.

**AC4 — One version, one immutable target.** Only strictly increasing stable `vMAJOR.MINOR.PATCH` versions, without leading zeroes, qualify. Bind the version at merge time; editing the PR description later cannot change it. Verify owner identity for tag creation and release promotion, the merged PR's base and head repository/branches, version, and exact merge SHA before any public write. A valid-looking tag on `dev`, an older arbitrary ancestor of `main`, a different PR's merge, a branch named like a tag, or a version already used for another commit cannot ship. Protect `v*` creation separately from update/deletion: even the owner cannot move a published tag under the policy. Ordinary tags never publish.

**AC5 — Keep the existing publisher.** Preserve the unprivileged tag build, default-branch publisher, pinned `RELEASE_TOOLS_SHA`, successful check-run requirement, build-run artifact/digest verification, and tap downgrade/no-overwrite protections. Source `main` is the stable branch; the separate Homebrew tap retains its existing publishing contract. Retry a failed release using the same tag and commit; report partial release/tap state and follow the existing recovery procedure, never retag. An ordinary push, unmerged PR or unauthorized tag causes zero release or tap writes.

**AC6 — Prove both sides before cutover.** Demonstrate an owner push, an owner PR without a second review, and a successful PR→tag dry run; also demonstrate a contributor PR pending owner approval, rejected direct push, stale quality result, unauthorized tag and mismatched PR/tag rejection. Inspect effective branch/tag rules and release-environment permissions. If the account type, plan or effective bypass list cannot distinguish the sole owner, keep the existing manual release gate. Local fixtures can prove policy decisions, not hosted enforcement.

## Research and trade-offs

- GitHub [does not allow self-approval](https://docs.github.com/en/pull-requests/how-tos/review-pull-requests/reviewing-proposed-changes-in-a-pull-request). Treat “auto-approved” as **no redundant review**, not automatic merging or bypass of CI. [Rulesets layer](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets#about-rule-layering): keep non-bypassable release PR/check requirements separate from the narrowly scoped review exception. A [bypass actor](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository) is the actor performing the action, not proof of PR authorship. On a [personal repository](https://docs.github.com/en/account-and-profile/reference/permission-levels-for-a-personal-account-repository) there is one owner; do not assume “admin” still means this individual after an ownership or access change.
- GitHub [tag rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) distinguish creation from update/deletion. They do not attest which PR or commit a version represents. The publisher must establish that link.
- [Workflow-level path filters leave required checks pending](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/skip-workflow-runs). Scope expensive jobs inside the development gate. Keep the decision visible even when the expensive work is unnecessary.
- GitHub distinguishes [merged from merely closed PRs](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#running-your-pull_request-workflow-when-a-pull-request-merges). A tag pushed by the repository's [`GITHUB_TOKEN` does not trigger another push workflow](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow). An authenticated owner tag preserves the existing trigger without a PAT, GitHub App or new release service.

## Transition and limits

Today [CI](../../.github/workflows/ci.yml) targets `main`; [Release build](../../.github/workflows/release.yml) responds to `v*`; [Release publish](../../.github/workflows/release-publish.yml) checks ancestry and artifacts but ancestry alone does not prove a release PR. Seed `dev` from a known green `main`; establish the dev gate before requiring it. Exercise promotion and rejection cases before switching tag permissions or removing a reviewer. Preserve the existing [trusted-main docs publication](../../.github/workflows/docs.yml); `dev` validates only. Bring `main` history back into `dev` after promotion as needed; no permanent hotfix lane. Urgent fixes use the same release path.

The existing [022 decision §12.9 item 12](../022-architecture-remediation/plan.md) deliberately requires a manual release-environment confirmation. The owner selected its replacement by the matching tag push. Retain the current confirmation until AC6 shows the owner-only path works and cutover is separately authorized. No change to package formats, public commands, tap names or the [vision §6 engineering principles](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md). Branch creation, hosted settings, credentials, tags and publication are live operations outside this draft.

## Approval

Approved together by owner `nawodyaishan`, 2026-11-18. Evidence: “approve spec r2, plan r1, and tasks r1 together and authorize B1 only”. Execution scope is B1 (T1–T2); no later batch or live cutover authorized.
