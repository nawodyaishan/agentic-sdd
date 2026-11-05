# Taskfile migration tasks and continuation

Approval: [spec §Approval](spec.md#approval). Shared design, guidance and tools: [plan](plan.md).

## Tasks

- **T01 — Inventory and parity tests:** inventory live Make consumers and protected CI inputs; add tests for Task task names, default, variables, failure paths and effect boundaries, using isolated fixtures. Covers AC 1–3, 5. Pending.
- **T02 — Taskfile and verification:** implement Task equivalents, migrate check local recipes and trusted-input handling and their tests, then remove Makefile. Preserve trusted-base execution and no-download verification settings. Depends on T01. Covers AC 1–3, 5. Pending.
- **T03 — CI and current guidance:** update release CI, README, relevant active docs, scripts and AGENTS instructions; keep historical records accurate and preserve unrelated dirty work. Verify pin/provenance or report an explicit blocker. Depends on T01/T02. Covers AC 1, 4, 5. Pending.
- **T04 — Aggregate verification and review:** run focused and required gates where available, inspect task parity and full authorized diff, report omitted checks and unresolved findings; do not run real-home apply, tag, release or push as a test. Depends on T02/T03. Covers AC 1–5. Pending.

## Execution batches

| Batch | Tasks | Outcome and verification | State / next action |
|---|---|---|---|
| B1 | T01–T04 | Coherent migration of task interface, trusted gate integration and active documentation; isolated parity tests, required local checks, relevant trusted-CI and workflow checks when available. | User approved for the record; technical verdict **Blocked**, with 007-F1 retained. No publication. |
| B2 | T05 | Route CI/CD through the reviewed-base Taskfile while keeping candidate execution isolated; validate the workflow and document remaining transition evidence. | Implementation delivered; **awaiting human review**. Technical verdict **Blocked** for the transition on 007-F1; hosted behavior unobserved. |

### T05 — CI/CD Taskfile continuation

Use a pinned Task setup action and release in CI, invoke `verify-trusted` from the reviewed base only, and retain candidate execution inside the existing pinned isolated gate. Test workflow source boundaries and the existing base-owned entry; do not use the candidate Taskfile on the host or weaken contract equality. Review the current CI/CD diff and record hosted and prior-base transition gaps.

## Readiness reconciliation

Every acceptance criterion is covered above; T01–T03 own implementation and T04 owns aggregate verification. The trusted check producer/consumer boundary remains the reviewed base's executable contract, not the candidate Taskfile. The external Task/action pin remains a research and implementation prerequisite; lack of evidence blocks a safe CI claim. User-requested commit/push must be scoped to the actual migration, never unrelated existing dirty work. No technical verdict or approval has been recorded.

## Continuation record

2026-11-05: Combined r1 approved and B1 authorized; see spec §Approval. Replaced the Makefile in the working tree with `Taskfile.yml`, migrated current documentation, hooks, check local recipes and release CI, and added Taskfile behavior and trusted-path tests. Existing unrelated dirty work was preserved. `AGENTS.md` contains pre-existing changes as well as this migration's three edits; it requires selective staging. The release workflow pins setup-task v2 to commit `a00fbb05ce67b35648be3c78cbc9fd85354c757e` and Task 3.54.0; the action metadata and version input were inspected, but no hosted release run or actionlint validation occurred.

Verification: `task test` and `task vet` passed; `task verify` passed all six required checks, with evidence at `/tmp/agentic-sdd-verify.22R0Mt/`; `task docker-e2e` passed; `python3 -B -m unittest discover -s tests/harness -p test_taskfile.py` passed 7/7; `test_trusted_ci.py` passed 12/12; explicitly opted-in `test_trusted_ci_container.py` passed 5/5 against its frozen reference fixture. `git diff --check` passed. The Task tests exercised failures before implementation and detected and corrected shell `umask`, inherited offline environment and command-line cache precedence differences. Darwin checks are cross-builds, not native runtime validation. The container fixture's matching historical base/candidate contracts do not test the current HEAD-to-candidate transition.

**007-F1 — trusted contract transition blocked:** the currently committed trusted base compares the entire required-check record, including `local_recipe`. Comparing `git show HEAD:harness/checks.json` with the candidate contract through `required_contract` fails with `required contract removed/replaced; reviewed adoption required`. Thus a PR using today's trusted base rejects this change before running checks, even though executable check argv and IDs remain unchanged. A passing candidate-local verify or frozen-fixture gate does not resolve this. The old base also does not identify a newly introduced Taskfile as a review path until its bootstrap is adopted. Do not weaken or bypass the trusted gate, or push the incomplete migration as though it passed. A reviewed transition strategy and any required staged trusted-base adoption need a human decision.

Remaining: resolve 007-F1 and verify the actual transition, examine command parity and release CI on the final candidate, and complete human review. No commit or push has occurred. Existing 006 B4 deferral and findings remain unchanged.

2026-11-05 B2: The user approved B1 for the record and explicitly requested CI/CD be changed to Taskfile; spec §Approval records the scope. Updated `.github/workflows/ci.yml` to install the same commit-pinned Task setup action and exact Task version as release CI, then invoke `verify-trusted` from `gate-trusted` with explicit trusted, candidate and output paths. Candidate checkout remains data on the host; its tests execute only in the existing isolated gate. Added a workflow-boundary regression and updated `harness/trusted-ci.md`. Existing B1 code and unrelated dirty files were preserved.

Consulted Context7 for GitHub Actions and Task, and Exa for official [GitHub pull_request_target security guidance](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target): avoid executing untrusted candidate code with privileged access. The Task setup action is full-SHA pinned; neither external documentation nor syntax validation certifies the running hosted boundary.

B2 checks: Taskfile/workflow unit tests **8/8**, trusted-CI unit tests **12/12**, `task verify` **6/6** (`/tmp/agentic-sdd-verify.WG9ysx/`), YAML parsing for both workflows and Taskfile, and `git diff --check` passed. `actionlint` was unavailable. B1's 5/5 isolated container fixtures previously exercised the same reviewed-base `task --dir ... verify-trusted` entry; they did not execute GitHub Actions or the changed old-base contract transition. **007-F1 remains open**; no hosted CI run, commit, push, release or tag was performed. Completion self-review: **Blocked** for the overall migration; B2 awaits human review. Next action: decide the reviewed trusted-base transition before claiming completion or publishing.

2026-11-05 adoption decision: The user selected direct adoption on the trusted branch, explicitly accepting that the migration does not pass the old-base PR gate, and confirmed inclusion of the pre-existing local `64fbf8f` commit in the push. The technical observation 007-F1 and lack of hosted PR evidence remain; a post-push main run cannot retroactively establish old-base PR validation. No branch rule, tag, release or unrelated dirty file is authorized to change. Stage only feature 007 changes, including only the Taskfile-related hunks of `AGENTS.md`.
