# Feature 021 — test-case matrix

**Status:** draft companion to [spec.md](spec.md); not approval, not evidence, not a test run.
**Purpose:** close the named public-CLI gaps in Feature 016 (+017), align the orchestration skill surface, and record structural traceability. This does not re-test the 123 typed-service `Test*` functions.

## 1. Evidence base

| Surface | Existing coverage | 021 action |
|---|---|---|
| `internal/execution` typed service | 123 `Test*` (122 tests + `TestCheckHelperProcess`) | Reference; add only if a CLI gap cannot be reached |
| `cmd/agentic-sdd` execution CLI | 7 tests: 3 in `execution_cli_test.go`, 4 in `execution_integrity_test.go` | Reference; add the G-cases |
| CLI shape | `TestDocumentationDispatchInventoryParity`, `TestDocumentationFlagSetInventoryParity`, `TestDocumentationHelpTextParityAndKnownExceptions`, `TestDocumentationSupportedCoreSelectionExcludesExperimental` | Do not re-derive |
| Generated reference | `website/scripts/reference/configuration.mjs` vs `website/src/content/docs/reference/execution.md` | Untouched |
| 11-package gate | `Taskfile.yml`, `harness/checks.json`, `tests/supported-core-go.sh`, two Python parity suites | Untouched |
| Skill surface | `opencode-pack/skills/agentic-sdd-orchestrate/**`, `opencode-pack/routing.md`; ground truth `docs/execution-runs.md` | G14 alignment check |

## 2. Closed gap list

New tests live in `cmd/agentic-sdd/execution_cli_contract_*_test.go`, `execution_skill_alignment_test.go` and `execution_contract_traceability_test.go` (package `main`). Each name is an exact top-level `func Test…`.

| ID | Test | Fixture / technique | Exact assertions |
|---|---|---|---|
| G1 | `TestExecutionContractCheckTimeout` | `sh -c 'sleep 5'`, `--timeout 300ms` | exit 124; `kind=execution.run.check`; `result=timed_out`; status `observed-partial`; stdout/stderr log paths; never a pass |
| G2 | `TestExecutionContractCheckFailure` | `sh -c 'exit 7'` | exit 1; `result=failed`; command exit 7 recorded; wrapper exit is not 7; logs present |
| G3 | `TestExecutionContractCheckOutputLimit` | `--max-output-bytes 1024` + POSIX `sh` builtin loop | exit 1; `result=output_limit`; `output_limit=true`; `stdout_truncated=true`; partial state |
| G4 | `TestExecutionContractCheckInterrupted` | Unix only: `sh -c 'kill -TERM $$'` | exit 1; `result=interrupted`; partial state; lock released; Windows/absent `sh` = `not_run` |
| G5 | `TestExecutionContractWriterLock` | `sh -c 'sleep 3'` holder; poll `.lock` + checkpoint `running` with a 2s deadline; second `run check`/`task brief` while holder sleeps; join | second writer exit 2; no new artifact/checkpoint bytes; lock diagnostic |
| G6 | `TestExecutionContractCleanRefusals` | unowned/symlinked/moved run dir; live lock from a deterministic concurrent check; absent run | unsafe/live-lock exit 2, nothing deleted; absent run exit 0 `already absent`; symlink setup skip = `not_run` |
| G7 | `TestExecutionContractReviewPartialReasons` | `--head` differs; conflicted index | exit 3; `complete=false`; `head_not_current` / `index_conflict`; no complete claim |
| G8 | `TestExecutionContractRecordValidation` | unknown field, >1 MiB, `--result -` via direct `run(...)` stdin | invalid/oversize exit 2, empty stdout, `error:` stderr, no record artifact; stdin success writes raw claim + digest |
| G9 | `TestExecutionContractStatusStale` | pass, then commit or dirty the worktree | exit 0; state `stale`; `head_changed` / `worktree_changed`; `next_permitted_investigation` exact |
| G10 | `TestExecutionContractFlagEdges` | `run check` without `--`; empty vector; `--timeout -1s` with valid run | exit 2; empty stdout; single `error:` on stderr; no long usage block; no artifact change |
| G11 | `TestExecutionContractAdapterWiring` | `--direct` + feature/batch; explicit `--run-root`; feature vs direct `--task`; invalid `--base`; review cap flags | exit 2 for conflicts/invalid base; `workspace_git_visible` surfaced; feature missing `--task` exit 2; direct defaults `direct` |
| G12 | `TestExecutionContractServiceEdges` | missing program; worktree mutated during check; spec/tasks edit vs checkbox-only; record blocker | `not_started` exit 1; worktree change flag + `observed-partial`; `stale` vs `progress_updated`; blocker → `blocked` |
| G13 | `TestExecutionContractTraceabilityMatrix` | `go/parser` over `./*_test.go`, `../../internal/execution/*_test.go`, `../../internal/skillsync/*_test.go`; read `../../specs/021-execution-cli-verification/test-case-matrix.md` | every named top-level test exists; 016 FR-01–FR-12, 016 AC-01–AC-13, 017 AC-01–AC-05 and 019 D-03/D-04/D-10 exactly once; status vocabulary valid; no writes |
| G14 | `TestExecutionContractSkillAlignment` | read `../../opencode-pack/skills/agentic-sdd-orchestrate/**`, `../../opencode-pack/routing.md`, `../../docs/execution-runs.md` | seven canonical command families; `run record` claim; distinct exit tokens `124/3/2` present (no global count); `run status` exit 0 for `stale`/`blocked`; four drift reasons subset; 017 foreign/`--force` recovery; required `not a sandbox` denial; no active legacy execution examples or affirmative sandbox/approval/resume claims; denial wording allowed |

## 3. Existing cases referenced (non-gaps)

- Direct scope/refusal: `TestExecutionCLIDirectScopeAndRefusals`.
- Lifecycle/help/format: `TestExecutionCLIInitBriefStatusRecordClean`, `TestExecutionCLIHelpAndFormat`.
- Foreign checkpoint: `TestExecutionIntegrityForeignCheckpointRefused`, `TestLoadCheckpointRequiresOwningRunID`.
- Deleted/legacy brief: `TestExecutionIntegrityDeletedRegisteredBrief`, `TestExecutionIntegrityLegacyUnregisteredBrief`.
- Repeat/force: `TestExecutionIntegrityRepeatAndForce`.
- Brief extraction: `TestTaskBrief_HeadingT1`, `TestTaskBrief_HeadingT01`, `TestTaskBrief_T1_vs_T10`, `TestTaskBrief_CheckboxForm`, `TestTaskBrief_NestedHeadingsAndNonTaskBoundary`, `TestTaskBrief_CodeFences`, `TestTaskBrief_MalformedFence`, `TestTaskBrief_CRLF`, `TestTaskBrief_UnresolvedReference`, `TestTaskBrief_DuplicateReferenceInSpec`, `TestTaskBrief_MissingTask`, `TestTaskBrief_WrongBatchOnlyTask`, `TestTaskBrief_MissingAndDuplicateBatch`, `TestTaskBrief_DuplicateTaskInBatch`, `TestTaskBrief_UnparseableTaskHeading`, `TestTaskBrief_ScopeDigest_ProgressOnly`, `TestTaskBrief_IdempotenceAndForce`, `TestTaskBrief_FileSizeLimitAndUTF8`, `TestTaskBrief_DirectScopeRefusal`.
- Workspace: `TestRunIDFormat`, `TestValidateRunID`, `TestWorkspaceInitCreatesPrivateIgnoredRun`, `TestWorkspaceInitFailClosedOnVisibleIgnore`, `TestWorkspaceInitRequiresGitRepository`, `TestWorkspaceInitRejectsUnbornHead`, `TestWorkspaceInitRejectsScopeRootSubdirectory`, `TestWorkspaceInitExplicitRootOutsideRepository`, `TestWorkspaceInitExplicitRootInsideRepositoryReportsVisibility`, `TestWorkspaceInitRefusesSymlinkedScratchParent`, `TestWorkspaceInitRefusesSymlinkRunRoot`, `TestResolveRunRoot`, `TestLockConflictAndRelease`, `TestLockOwnershipCheckedRelease`, `TestLockStaleReclaim`, `TestCleanPreviewApplyAndAbsent`, `TestCleanRefusesLiveLock`, `TestCleanRefusesUnownedAndSymlinkedRuns`, `TestCleanRefusesMovedRun`.
- Check service: `TestCheckPassReachesVerifiedEvidence`, `TestCheckArtifactsAndPrivateProvenance`, `TestCheckFailureFarFromTail`, `TestCheckTimeout`, `TestCheckSignalIsInterrupted`, `TestCheckOutputLimitTerminatesAndTruncates`, `TestCheckSummaryTruncation`, `TestCheckSummaryTailFallback`, `TestCheckNoSecretEcho`, `TestCheckMissingProgramIsNotStarted`, `TestCheckRequestValidation`, `TestCheckConcurrentLockAndRunningStatus`, `TestCheckReleasesLockOnEveryPath`, `TestCheckWorktreeChangedDuringCheck`, `TestCheckLaterPassReachesVerifiedEvidence`, `TestCheckKilledWriterLeavesInterruptedPartialState`, `TestCheckEvidenceWriteFailureIsNotAPass`, `TestCheckDirectScopeDefaultsTask`.
- Review service: `TestReview_Committed`, `TestReview_Staged`, `TestReview_Unstaged`, `TestReview_Untracked`, `TestReview_Rename`, `TestReview_Delete`, `TestReview_ModeChange`, `TestReview_Binary`, `TestReview_LargeFile`, `TestReview_MaxTotalBytes`, `TestReview_Sensitive`, `TestReview_Symlink`, `TestReview_SubmoduleGitlink`, `TestReview_ConflictedIndex`, `TestReview_HeadMismatch`, `TestReview_NoMutation`, `TestReview_InvalidBase`, `TestReview_CheckpointIntegration`.
- Checkpoint/status: `TestHighestStatePrecedence`, `TestNormalizedBatchScopeDigest`, `TestCheckpointRoundTrip`, `TestStatusPrepared`, `TestStatusSourceAndWorktreeFreshness`, `TestStatusMissingArtifact`, `TestStatusInterruptedCheck`, `TestStatusBlocked`, `TestStatusContradictoryEvidence`, `TestStatusScopeMismatch`, `TestStatusNeverUsesApprovalLanguage`, `TestStatusStateMachineTable`, `TestLoadCheckpointRejectsUnknownState`, `TestStatusUnknownRun`.
- 017 brief integrity: `TestBriefRegisterContract`, `TestBriefInspectArtifactsDiagnosesDrift`, `TestStatusBriefFreshness`, `TestBriefNextInvestigationReasons`.
- Record: `TestRecordParseValidFixtures`, `TestRecordParseTaskResolution`, `TestRecordRejectsInvalidInput`, `TestRecordIngestPassedRemainsAwaitingVerification`, `TestRecordIngestBlockedYieldsBlocked`, `TestRecordIngestRejectsInvalidWithoutArtifact`, `TestRecordIngestTaskMismatch`, `TestRecordIngestDirectDefault`.

## 4. Traceability — Feature 016 FR

Origin: `existing-service`, `existing-cli`, `new-cli`, `static`, `static-check`. Runtime status: `covered` (executed behavioral assertion), `covered-static` (executed content/structure check), or `not_run` (no assertion).

| Clause | Evidence | Origin | Runtime status |
|---|---|---|---|
| FR-01 source/identity | `TestRepositoryOpenAndObserve`, `TestFeatureScope`, `TestDirectScope`, `TestExecutionCLIInitBriefStatusRecordClean` | existing-service / existing-cli | covered |
| FR-02 workspace privacy | `TestWorkspaceInitCreatesPrivateIgnoredRun`, `TestWorkspaceInitFailClosedOnVisibleIgnore`, `TestWorkspaceInitExplicitRootOutsideRepository`, `TestWorkspaceInitExplicitRootInsideRepositoryReportsVisibility`, `TestWorkspaceInitRefusesSymlinkRunRoot`; G6, G11 | existing-service / new-cli | covered |
| FR-03 brief extraction | `TestTaskBrief_HeadingT01`, `TestTaskBrief_T1_vs_T10`, `TestTaskBrief_CheckboxForm`, `TestTaskBrief_CodeFences`, `TestTaskBrief_MissingTask`, `TestTaskBrief_IdempotenceAndForce` | existing-service | covered |
| FR-04 check evidence | `TestCheckPassReachesVerifiedEvidence`, `TestCheckFailureFarFromTail`, `TestCheckNoSecretEcho`; G1–G4, G12 | existing-service / new-cli | covered |
| FR-05 review package | `TestReview_Committed`, `TestReview_Staged`, `TestReview_Unstaged`, `TestReview_Untracked`, `TestReview_HeadMismatch`, `TestReview_ConflictedIndex`; G7, G11 | existing-service / new-cli | covered |
| FR-06 checkpoint/stale | `TestStatusSourceAndWorktreeFreshness`, `TestStatusInterruptedCheck`, `TestStatusContradictoryEvidence`, `TestBriefNextInvestigationReasons`; G8, G9, G12 | existing-service / new-cli | covered |
| FR-07 failure classification | `opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md` | static | not_run |
| FR-08 severity/evidence | `skills-files/agentic-sdd-router/references/review-resolution.md` | static | not_run |
| FR-09 request-only reviewer | orchestration/reviewer skill prose; no runtime dispatch in 016 | static | not_run |
| FR-10 orchestration integration | `opencode-pack/skills/agentic-sdd-orchestrate/` prose; G14 | static-check | covered-static |
| FR-11 CLI compatibility | `TestExecutionCLIHelpAndFormat`, `TestDocumentationDispatchInventoryParity`, `TestDocumentationFlagSetInventoryParity`; G10 | existing-cli / new-cli | covered |
| FR-12 honest accounting | `TestCheckNoSecretEcho`, `TestStatusNeverUsesApprovalLanguage`; G9 | existing-service / new-cli | covered |

## 5. Traceability — Feature 016 AC

| Clause | Evidence | Origin | Runtime status |
|---|---|---|---|
| AC-01 commands/help | `TestExecutionCLIHelpAndFormat`, `TestDocumentationHelpTextParityAndKnownExceptions` | existing-cli | covered |
| AC-02 reusable services | package import graph; no CLI/TUI business logic | static | not_run |
| AC-03 workspace safety | `TestWorkspaceInitCreatesPrivateIgnoredRun`, `TestWorkspaceInitRefusesSymlinkRunRoot`, `TestCleanRefusesUnownedAndSymlinkedRuns`; G6 | existing-service / new-cli | covered |
| AC-04 task extraction | `TestTaskBrief_HeadingT01`, `TestTaskBrief_CheckboxForm`, `TestTaskBrief_T1_vs_T10` | existing-service | covered |
| AC-05 check evidence | `TestCheckPassReachesVerifiedEvidence`, `TestCheckFailureFarFromTail`; G1–G4 | existing-service / new-cli | covered |
| AC-06 review package | `TestReview_Committed`, `TestReview_Sensitive`, `TestReview_NoMutation`; G7 | existing-service / new-cli | covered |
| AC-07 checkpoint/stale | `TestStatusSourceAndWorktreeFreshness`, `TestStatusStateMachineTable`; G9 | existing-service / new-cli | covered |
| AC-08 failure boundary | orchestration failure guidance | static | not_run |
| AC-09 severity/validity | `review-resolution.md` | static | not_run |
| AC-10 request-only reviewer | skill/reference prose | static | not_run |
| AC-11 orchestration refs | orchestration skill prose; G14 | static-check | covered-static |
| AC-12 CLI compatibility | `TestExecutionCLIHelpAndFormat`, `TestDocumentationDispatchInventoryParity`; G10 | existing-cli / new-cli | covered |
| AC-13 direct scope | `TestExecutionCLIDirectScopeAndRefusals`; G11 | existing-cli / new-cli | covered |

## 6. Traceability — Feature 017 AC and 019 clauses

| Clause | Evidence | Origin | Runtime status |
|---|---|---|---|
| 017 AC-01 foreign checkpoint | `TestExecutionIntegrityForeignCheckpointRefused`, `TestLoadCheckpointRequiresOwningRunID` | existing-cli / existing-service | covered |
| 017 AC-02 locked brief | `TestTaskBrief_Transaction_LiveWriterLock`, `TestTaskBrief_Transaction_ContradictoryRequestAndDirectScope`, `TestTaskBrief_Transaction_LockReleasedOnError`, `TestTaskBrief_IdempotenceAndForce`, `TestExecutionIntegrityRepeatAndForce`; G5 | existing-service / new-cli | covered |
| 017 AC-03 registration/digest | `TestBriefRegisterContract`, `TestBriefInspectArtifactsDiagnosesDrift`, `TestTaskBrief_Transaction_RegistrationAndStatus`, `TestTaskBrief_Transaction_PreservesPriorTaskState`, `TestTaskBrief_Transaction_TamperedBriefForce`, `TestTaskBrief_Transaction_LegacyRepairAndCheckpointFailure` | existing-service | covered |
| 017 AC-04 legacy reuse | `TestExecutionIntegrityLegacyUnregisteredBrief`, `TestStatusBriefFreshness`, `TestBriefInspectArtifactsDiagnosesDrift`, `TestBriefNextInvestigationReasons` | existing-cli / existing-service | covered |
| 017 AC-05 authority/interfaces | `TestExecutionIntegrityForeignCheckpointRefused`, `TestStatusNeverUsesApprovalLanguage` | existing-cli / existing-service | covered |
| 019 D-03 legacy wrappers | `TestLegacyForwardsEveryOtherInvocation`, `TestLegacyExactTopLevelQueriesIdentifyBothNames` (stub/injected; real link-time wrapper not exercised) | existing-cli (stub) | not_run |
| 019 D-04 `asdx` refusal | 019 launcher tests (nested TUI binary); not this execution surface | static | not_run |
| 019 D-10 wrapper determinism | `TestLegacyForwardsEveryOtherInvocation`, `TestLegacyExactTopLevelQueriesIdentifyBothNames` (stub/injected; real built wrapper not exercised) | existing-cli (stub) | not_run |

## 7. Origin/status vocabulary and G13 contract

The traceability tables use two separate columns:

- **Origin:** one or more slash-separated tokens from `existing-service`, `existing-cli`, `existing-cli (stub)`, `new-cli`, `static`, `static-check`.
- **Runtime status:** `covered` (an executed behavioral test asserts it), `covered-static` (an executed content/structure check asserts it), or `not_run` (no assertion in 021; reason and static/asset reference recorded).

No row may use `pass`, `accepted` or `certified` as an origin or status.

`G13` checks only:

1. Every exact backticked `Test<name>` token (not a placeholder such as `Test*` or `Test…`) exists as a top-level `func Test…` in `./*_test.go`, `../../internal/execution/*_test.go`, or `../../internal/skillsync/*_test.go`.
2. Every 016 FR-01–FR-12, 016 AC-01–AC-13, 017 AC-01–AC-05, and 019 D-03/D-04/D-10 clause appears exactly once as a row.
3. Every row's origin (split on `/`, allowing the `(stub)` qualifier) and runtime status use the vocabulary above.
4. `G13` itself does not appear as a dependency of a covered row.

`G13` does **not** prove a test body is non-empty, unskipped, selected by a run, or passing. It is structural traceability only. It reads the matrix with `go/parser`-derived symbol sets and never writes. It excludes 018's `test_research_workflow.py` and its scenario fixtures; 018 uses a separate suite and manifest.

## 8. Technique and compatibility constraints

- New files `cmd/agentic-sdd/execution_cli_contract_*_test.go`, `execution_skill_alignment_test.go`, `execution_contract_traceability_test.go`; all in package `main`.
- G14 reads the orchestration skill and references plus `docs/execution-runs.md`; it asserts canonical commands and boundary phrases and rejects active legacy execution examples. It asserts the required denial `not a sandbox` but only the absence of **affirmative** forms (`is a sandbox`, `acts as a sandbox`) and affirmative approval/resume phrases; denial wording (`never resume`, `not approval`) is allowed. It asserts distinct exit tokens `2`/`3`/`124` by presence (not a global occurrence count) and the four drift reasons as a subset; it does not assert full sentences, section numbering, line numbers, flag defaults, link depth, the eight-state list or the full freshness vocabulary.
- G14 treats `dispatch-contract.md` handoff fields as `covered-existing`; new assertions focus on the missing `run record`/`run clean`, exit `2/3/124`, `run status` exit 0 for stale/blocked, and 017 foreign/drift recovery.
- `routing.md` gets a prose-only cross-reference; G14 must not require a table row or a backticked model name there (`internal/appops/routing_test.go` parses the table).
- Reuse `executionTestRepo`, `runExecutionCLI`, `executionGitOutput`, `decodeRunIdentity`, `initExecutionIntegrityRun`; no second repo builder.
- No new package/dependency/Taskfile entry; 11-package gate unchanged.
- No PTY, no golden JSON, no built binary, no real `os.Exit`.
- Shell lookup: every shell-invoking case (`G1`–`G6`, `G12` where applicable) resolves `sh` via `exec.LookPath("sh")` and skips as `not_run` if absent; do not hard-code `/bin/sh`.
- Timeout: 250–300ms with `sleep 5`; output cap: POSIX `sh` builtin loop; signal: Unix + `LookPath("sh")`.
- Lock: holder runs `sh -c 'sleep 3'` (or a FIFO gate); poll `.lock` + checkpoint `running` with a 2s observation deadline; issue the competing call while the holder sleeps; then join. `RunCheck` writes the final state before releasing the lock, so polling `running` alone is not enough. No `t.Fatal` in workers; `t.Setenv` only on the test goroutine.
- Stdin: direct `run(...)` with `strings.NewReader`.
- Added runtime about 6s; the lock holder sleeps 3s with a 2s observation deadline; `harness/checks.json` test budget 180s.
- Only test files, feature docs and the named orchestration skill files change; docs/reference parity and the generated reference stay untouched.

## 9. Explicit exclusions / not_run accounting

- Real legacy `agentic-sdd` execution commands (link-time wrapper; in-process tests never enter it).
- Pre-019 `agentic-sdd run|task|review` examples inside frozen 016/017 records; historical, not active guidance.
- Real `os.Exit`, built `asdx-core` process behavior, `EPIPE`/closed-pipe semantics.
- Real ENOSPC; only permission/write-failure simulation exists.
- Runtime AC-08..AC-10 compliance (prose/skill behavior); AC-11 is `covered-static` via G14.
- Model compliance with the skill prose; G14 is a content check, not activation or behavior proof.
- Non-Linux runtime; G4 on Windows or hosts without `sh`.
- Real home `apply`, network/model/MCP, sandbox/isolation, `IR-06-01`, release/packaging.
- Performance, fuzz and token-savings claims.
