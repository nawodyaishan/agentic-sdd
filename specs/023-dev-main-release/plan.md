# 023 implementation plan — r1

For [spec r2](spec.md). **Confirmed release decision:** the owner pushes the tag after merging the release PR. Only after hosted proof and separately authorized cutover does that push replace the manual environment confirmation in [022 §12.9](../022-architecture-remediation/plan.md).

## Design

1. **Protect refs before relying on actors (AC1, AC6).** Keep `main` as default. Require a promotion PR and a trusted check even for the owner; block force pushes and deletion on both branches. Permit only the owner to update `dev` directly or merge to either branch. Contributors use forks and receive an explicit owner review covering the current head before merge. Require evidence of either the owner's authenticated authorship and last update, or that current-head review; a bypass actor identifies the merger, not the contributor. Pin the repository and owner numeric IDs from GitHub, not commit metadata. Inspect effective rules, including apps and other bypasses. If GitHub cannot enforce the distinction on this account, stop rather than granting a wider exception.
2. **Run quality where work happens (AC2–3).** Adapt [CI](../../.github/workflows/ci.yml) and the validation-only jobs in [Docs](../../.github/workflows/docs.yml). Keep [the existing check selection](../../harness/checks.json), including the 008 fence. Use reviewed `main` workflow/launcher code, including for a direct `dev` push; merely changing the checkout in a workflow loaded from `dev` is insufficient. Candidate content runs only in the isolated gate. Publish the required result for the actual candidate SHA, not the base SHA of `pull_request_target`. For docs/spec-only changes explicitly record the path decision and applicable docs checks; never skip the entire required workflow. Development runs may supersede older runs on the same ref. No dev deployment, release environment, or tap token.
3. **Make a release PR's intent immutable (AC3–4).** Proposed minimal convention: a single tracked `release-version` line on `dev` contains `vX.Y.Z`; the promotion PR changes it relative to `main` and summarizes the release. Use a merge commit for promotion, retaining the tested dev head and prior main as parents. Take the version from the resulting tree, not the title/body or current branch tip. One merged promotion corresponds to one new version. Compare numeric version tuples against prior stable releases; do not use lexical sorting. Reject ambiguous PR matches, changed heads/bases, or version conflicts.
4. **Require evidence on the merge SHA (AC3–5).** Before merge, validate the proposed result for the same-repository `dev` → `main` PR. After merge, trusted `main` independently resolves the merged PR, authenticated merger, expected parents, version, and quality evidence; it emits release eligibility on that exact merge SHA. Never infer success from a similarly named check. Initially rerun quality for promotion if exact reuse cannot be proved; correctness is worth one run per release. Keep the existing publisher's wait and checks; require the specific eligibility result as well, rather than replacing existing checks with a green tag build.
5. **Publish through the existing pair (AC4–5).** [Release build](../../.github/workflows/release.yml) remains unprivileged. [Release publish](../../.github/workflows/release-publish.yml) verifies the original tag-push event (not the rerunner), true tag ref, owner ID, tag target, promotion evidence and version before public writes. Restrict `v*` creation to the owner and prohibit update/deletion without bypass. Preserve tooling pin, exact artifact checks, own-suite exclusion, serialization and tap downgrade/no-clobber behavior. Never automatically repair a partial publication.

### Shared interface between parallel workers

| Producer → consumer | Minimum evidence |
|---|---|
| Trusted quality → promotion | Repository ID, tested commit and tree, trusted gate revision, scope and input identities, authoritative run ID and conclusion. Reuse only for identical tree, gate, scope and inputs. |
| Trusted promotion → publisher | Repository ID, PR number, base/head refs and parent SHAs, merged SHA, owner merger ID, immutable version, trusted quality run link, successful eligibility on the merged SHA. |
| Tag build → publisher | Original event/run identity, owner sender ID, full tag ref and peeled SHA plus the existing run-bound manifest. A rerun never substitutes its initiator for the original sender. |

These are logical fields, not three new services or a database. Prefer existing GitHub run/check records and git objects. Treat build artifacts and candidate-supplied JSON as data, never as identity proof. Resolve authoritative run/workflow, source ref and SHA through GitHub. Missing, ambiguous, stale, timed-out or conflicting evidence denies promotion/publication. The first batch must prove the actual GitHub event and result bindings; if a field cannot be authenticated, report the gap rather than inventing it.

## Parallel delivery

| Lane | Assignment after authorization |
|---|---|
| OpenCode Console `sdd-implement-deepseek-free-1` | Development CI, docs validation routing and their focused tests. |
| OpenCode Console `sdd-implement-deepseek-free-2` | Promotion/tag eligibility, publisher wiring and their focused tests. |
| GPT Terra — `general` with `openai/gpt-6.1-terra` | In parallel with the two writers, develop black-box contract fixtures and release dry-run assertions against the agreed interface. Afterwards assess the integrated result, distinguishing its own fixtures from independently verified behavior. |
| Parent | Own shared contracts, integration, full selection, reviewer handoff and batch ledger. |

Use distinct worktrees rooted at the same reviewed revision, one writer per worktree. Workers return a diff, file list, commands/results and blockers. Integrate CI first, then release, then Terra's tests; only the parent resolves shared paths. The Console lanes must stay on free Console fallbacks. Terra is a verified model ID, **not verified free**; confirm allowance before dispatch if zero spend is required. Do not silently replace an unavailable lane with paid inference. Give workers only repository inputs appropriate for the model, no credentials or private logs.

Use installed `cicd-automation-workflow-automate` for workflow routing and `agentic-sdd-verification-review` for batch closeout. Use repository-native Task/Python checks, temporary git repos, and fake API/tap endpoints. Use Context7/GitHub docs for event-field questions, and GitHub rule/run inspection for hosted proof. No new CI platform or permanent test repository is required.

## Rollout and recovery

Keep the old release confirmation during development. Rehearse with an explicitly authorized disposable repository and inert publisher, then compare its effective rules with production before enabling anything. Seed `dev` from green `main`, enable its observed quality check, then require promotion evidence, protect tags, and switch the environment confirmation **last**. If evidence or rule enforcement differs, halt publication, retain or restore the manual confirmation, and correct forward by PR. Never move a tag to roll back. A production rehearsal, settings change, tag or public release requires its own live authorization.
