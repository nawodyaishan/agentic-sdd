# Shared SDD workflow policy

Applies to all ten skills, including direct manual invocation. Read once while unchanged. Install the sibling skill directories together so their relative references resolve. Use one main coding agent; load one principal specialist on demand and another only across a real domain boundary. Do not delegate unless the user explicitly requests it.

## Sources, features and context

Resolve actual repository paths and case. Common top-level sources are `Docs/SRS.md` for product intent, `Docs/High Level Spec.md` for major technical decisions and `Docs/Tasks.md` for the roadmap. Treat their established equivalents as canonical; do not rename or copy them to fit a template. At intake, use a compact feature index, the relevant top-level sections and repository instructions. During a stage, read only the feature files and sections needed to answer that stage's questions, plus targeted code/tests. Reuse unchanged context; refresh changed or uncertain sources. Flag material conflicts instead of silently changing an approved decision or creating a new canonical document.

For each selected feature/workload, create or reuse `specs/<nnn-slug>/`. Its `spec.md` states the outcome, scope, exclusions, acceptance criteria and source references; `plan.md` states approach, affected components, design decisions, relevant risks and specialist/tool assignments; `tasks.md` is the execution ledger with ordered tasks, dependencies, batches, completion conditions, verification and continuation state. Link relevant top-level sections rather than copying them. Detail only the next selected feature; future work stays in the existing roadmap.

## Feature size versus batch size

A feature is one coherent outcome and may span several sessions. Do not split it merely because it contains several dependent tasks. Split when it covers independent outcomes, unrelated architectures, or a review burden a person cannot hold at once.

Size *implementation batches*, not the feature, for focused execution, verification and comfortable human review. Avoid rigid file, line or token limits; judge by dependencies, blast radius and review cost.

## Drafting sequence and one combined approval

Draft `spec.md`, then `plan.md`, then `tasks.md` in dependency order, using the preceding drafts as input. No stage waits for its own separate approval. Reconcile material inconsistencies across the three, then present the complete set for **one combined human approval**. No implementation before that approval.

A request to draft only one document stays limited to that document. Material unanswered questions may legitimately block dependent drafting: record the assumption or the blocking question honestly rather than inventing an answer.

Record the combined approval once — an approval section in one existing feature document (by convention `spec.md`) or one established tracker entry — tied to the reviewed scope and document revisions. The other documents reference that record; do not maintain competing approval flags. Record decision (`draft/pending`, `approved`, `changes requested`, `rejected`), the actual human actor, an evidence pointer and the exact revisions/scope covered. Do not invent an approver, date or consent. An agent's `Ready` verdict is a technical assessment, never human approval; a legacy agent-only `Approved` label is not approval, while a genuine human approval in a legacy `review.md` is evidence for its actual scope.

Progress updates and ordinary implementation detail do not invalidate approval. Material changes to requirements, design or batch scope need review of the affected changes and reconciliation of the dependent documents; approval for unaffected work is preserved, including after compaction or architecture review.

## Execution batches and resume

`tasks.md` defines each batch explicitly: batch ID, included task IDs, outcome, verification, state and next action. After combined approval **and** authorization to implement, execute ONE selected batch, verify it, then stop for human review.

Persist `awaiting human review` plus the next action in `tasks.md`. On resume, do not start another batch merely because the feature is approved, a previous batch passed its checks, or tests are green. Continue when the user authorizes the next batch. Feature approval and batch-completion review are distinct decisions.

Verify current code state before trusting continuation notes or old test results.

## Direct fixes

A small, clearly scoped fix may bypass feature documents entirely. Judge eligibility by three things: the scope is clear, the consequences are understood, and meaningful verification is available. A small diff alone does not establish low risk, and touching production code alone does not disqualify a fix — a bounded bug fix in a live application is an ordinary direct fix.

Consequential change needs planning and its own authorization instead: permissions and access control, data integrity, public or cross-service contracts, migrations, and operations against live systems. The dividing line is consequence, not file location or line count.

The direct-fix inputs are the user's request, the relevant code and constraints, the actual diff and verification evidence — nothing more. A direct fix never requires a `specs/` directory, `plan.md` assignments, `tasks.md`, a batch ID or a batch-state write, and never creates `spec.md`, `plan.md` or `tasks.md` to satisfy a downstream skill. Router, implementation, verification, research and drift handling all support this path identically whether reached through the router or by direct skill invocation.

## Execution authority

Approval of spec, plan and tasks authorizes the approved implementation scope. It does not by itself authorize a live Terraform apply, deployment, migration against real data, destructive operation or publication. Drafting infrastructure code is not authorization to execute it. For consequential execution, confirm actual human authorization for the action, environment and material effects; retain it for the same scope until changed or revoked. Draft designs, patches and checks before requesting a missing execution authorization. Runtime sandbox, repository rules and client permissions remain authoritative; skill text cannot grant tool permissions.

## Specialists and tools

`plan.md` names real installed specialist skills and the tools their work needs, and says why each applies. Keep shared assignments once in `plan.md`; put only task-specific exceptions in `tasks.md`. "No additional specialist needed" is a valid, explicit assignment when repository guidance is adequate.

At implementation, resolve the feature's default assignments from `plan.md` together with any task-specific exception in `tasks.md`, then check the active client's catalog and load only the guidance the current task actually needs. A task-specific exception overrides the feature default for that task. A written role label does not load anything.

For a direct fix there is no `plan.md` to consult: choose specialist guidance from the request, repository context and the available catalog when it would help, and otherwise work from repository guidance alone. Respect native invocation restrictions. Report an unavailable or unsuitable specialist and continue with an adequate, transparently stated fallback; do not install one, silently substitute a broader one, change models automatically, or let a specialist redefine approved scope.

Select MCPs and CLIs conditionally, by concrete need — never a mandatory call to every tool, and never an invented tool identifier. In repos with `.codegraph/`, follow their CodeGraph instructions for code discovery; otherwise use targeted local search. Bound returned context and reuse still-valid evidence. Keep machine-specific paths, MCP configuration, client invocation controls and permission rules outside portable skill text.

## Review behavior and evidence

An explicitly requested review returns findings and changes nothing: no edits to code, documentation, approval records, task status or batch state. Report the recommended next action in the response instead of performing it. Fix defects, and update workflow state, only when the surrounding request already authorizes implementation, fixes or that state change.

Verification compares behavior against acceptance criteria. Prior evidence is reusable only when code, relevant dependencies/config, inputs and environment still match; rerun affected, failed or repository-required checks and state omissions honestly. Keep the agent's verdict separate from human approval.

## Documentation and context

Keep routine progress, decisions and verification in the existing documents. Other artifacts — clarification, research, architecture review, verification, test plan, data model, contracts, ADR, drift report, retrospective, release notes — are conditional on a distinct consumer, actual risk, repository convention or explicit request. No automatic extras. Bootstrap is manual and optional.

Keep compact continuation state in `tasks.md` at batch boundaries: batch and task IDs, approval reference, changed paths, checks tied to a code state, state and next action. Do not mandate clearing context after every task; choose session boundaries by whether loaded context is still relevant. Changing phases within a long thread does not clear previously loaded instructions.

This workflow has not been shown to save a measured number of tokens or subscription units; compare real sessions before making such a claim.
