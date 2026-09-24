# Agentic SDD workflow audit — corrected

24 September 2026. This report supersedes the earlier version of this audit. The earlier recommendation to omit most per-slice `spec.md`, `plan.md`, and `tasks.md` files was based on an incomplete understanding of Nawodya's workflow.

## What the workflow actually does

The top-level `Docs/SRS.md`, `Docs/High Level Spec.md`, and `Docs/Tasks.md` define the product, major technical decisions, and overall roadmap. The coding agent then creates a *bounded, reviewable work packet* for each feature or workload in `specs/<nnn-slug>/`. The user inspects and approves each packet's `spec.md`, then `plan.md`, then `tasks.md`. Implementation follows the approved tasks. Specialist roles and their existing skills are assigned within planning and task breakdown.

The screenshot shows distinct folders including `001-complete-backend`, `002-e2e-testing`, `003-frontend`, and `004-phase-a-acceptance`. The visible folders have different additional documents. Their exact content and whether a large-sounding folder such as “complete-backend” really fits a single execution context cannot be inferred from filenames alone.

```text
Main SRS + technical specification + roadmap
  -> choose one bounded slice and create specs/<nnn-slug>/
  -> draft spec.md -> human reviews and approves
  -> draft plan.md with appropriate specialist(s) -> human reviews and approves
  -> draft tasks.md with execution roles, skills, checks -> human reviews and approves
  -> one main coding agent implements one task/small group
  -> verify against approved packet -> update roadmap and necessary sources
```

The packet is a useful context boundary and approval artifact, not an unnecessary copy of the top-level documents. Its content should link to the source sections and add only slice-specific requirements, design, and execution detail. Distinguish a reviewer's agent verdict from the human's recorded authorization.

Research on the correction: GitHub's current Spec Kit quickstart has a core `specify → plan → tasks → implement → converge` flow and a fuller path with clarification, checklist and analysis gates for production work. Its workflow engine can explicitly pause for human review, showing a review gate between specification and planning. This supports a configured three-stage approval workflow, although it does **not** establish that every team must approve all three phases separately or always create all three documents. [Spec Kit quickstart](https://github.com/github/spec-kit/blob/main/docs/quickstart.md); [Spec Kit workflows](https://github.com/github/spec-kit/blob/main/workflows/README.md)

Anthropic's engineering account of long-running coding work says that attempting too much in one session left half-finished changes and caused later sessions to guess what happened; it favors incremental features with clear continuation state. This supports bounding a work packet and its implementation tasks. It is an engineering case study, **not** a measured optimum for document count or a mandate to use three Markdown files. [Anthropic: Effective harnesses](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)

## The real token and document issue

The ten original uploaded skills totalled 3,691 words and 26,163 characters; this is not a token measurement, and installation alone does not necessarily load all ten bodies at once. The waste risk was *extraneous artifacts* and *repeated reads*, not the per-slice three-file structure itself. Current OpenAI guidance favors short skill entry points, progressive disclosure and contextual file reads rather than mandatory review of the entire repository before each edit. [OpenAI: Rethinking skills and prompts](https://developers.openai.com/blog/rethinking-skills-and-prompts-for-gpt-6-astra)

Keep `spec.md`, `plan.md`, and `tasks.md` **for each chosen packet**, because that is the user's approval interface. Make `clarify.md`, `research.md`, `review.md`, `verification.md`, `test-plan.md`, `data-model.md`, contracts, checklists, ADRs, retrospectives and release notes conditional on actual need, project conventions, or explicit requests. Verification results can usually be summarized in the current task/response; when an independent audit record is necessary, retain it.

Bootstrap should discover top-level documents and existing governance without materializing twelve generic templates. Work packets should be sized by actual dependencies, verification, and review cost; do not set a target of filling an entire model context window. If a packet includes several dependent architectures or requires many unrelated file reads, split its work into smaller packets or tasks.

For each session, consult a compact packet index and the relevant portions of the three top-level documents, then read `spec.md`, `plan.md`, and the current part of `tasks.md` as required by that stage. After final approval, implement one ready task and run focused checks. Start a new task session if previously loaded phases/skills obscure useful context. Claude's context documentation notes that skill bodies can be reinjected after compaction, so merely changing phases within one long session does not erase prior context. [Claude context guidance](https://code.claude.com/docs/en/context-window)

## Corrective changes needed in the revised ten skills

The ten revised skills supplied on 24 September 2026 were read in full. They now overcorrect toward optional/spec-less work. The following changes reconcile them with the intended work-packet workflow:

| Revised skill or shared reference | Required correction |
|---|---|
| `agentic-sdd-router` | Distinguish **project intake** from **selected packet workflow**. Choosing a new feature/workload creates or reuses `specs/<nnn-slug>/`. Within that packet route through `spec.md`, `plan.md`, `tasks.md` and the user's separate approval checkpoint after each. The absence of a packet file matters once a packet is selected; the presence of the main SRS does not replace it. Preserve a direct-fix path when the user explicitly chooses to bypass the packet for a trivial change. |
| `agentic-sdd-bootstrap` | Stay minimal; do not revive twelve templates. Recognize `Docs/`, existing `specs/`, and a canonical place for recording packet decisions/approvals. |
| `agentic-sdd-spec` | By default, for a selected new packet, write `specs/<nnn-slug>/spec.md`. Derive its outcome, scope and acceptance from the relevant SRS/main-task sections. Mark it draft and stop for the user's review before moving to plan unless approval for that exact spec was already given. |
| `agentic-sdd-plan` | For an approved packet spec, write `plan.md`. This contains the concrete design, boundaries and specialist assignment using real installed skill names. Mark it draft and stop for user review before task breakdown; high-risk design may receive a separate architecture review as well. |
| `agentic-sdd-tasks` | For an approved packet plan, write `tasks.md`: small verifiable implementation tasks, sequence/dependencies, per-task specialist skill(s), and required checks. Mark it draft and stop for user review before implementation. Preserve the *global* `Docs/Tasks.md` as the roadmap, not the task-level execution ledger. |
| `agentic-sdd-architecture-review` | Continue to check consequential architecture/operational risks. It does not substitute for the user's ordinary spec/plan/task signoffs. Technical `Ready` is not a human signature; keep authorization once at the correct packet scope. |
| `agentic-sdd-implement` | Select one approved packet task or a small approved group, load its relevant specialist into the main agent, read the current packet plus targeted code and source links, implement and verify. Do not demand every optional packet document. |
| `agentic-sdd-verification-review` | Compare the actual diff and test evidence to the packet's approved spec, plan and task acceptance criteria. Reuse applicable checks and flag deviations; only add `verification.md` if the repo needs a separate record. |
| `agentic-sdd-research-spec` | Research only a named gap in the current packet and save relevant evidence in `spec.md` or `plan.md`; create `research.md` only for a distinct consumer. |
| `agentic-sdd-drift-retro` | Update impacted canonical sources and roadmap where behaviour/scope materially changed. Avoid routine `no drift` files or per-task retrospectives. |
| `workflow-policy.md` and `specialists.md` references | These two referenced files were not attached; inspect them in the actual skills repository. They must encode the per-packet three-stage human approvals and true installed specialist names, without duplicating full policy in all ten skills. |

**Approval evidence:** Store the review decision once per packet stage in the existing packet or an established tracker (e.g., a small metadata section), with the actor, exact document revision/scope and decision. A user may approve a defined set of stages in advance; do not demand a redundant ceremony. If source/plan changes materially after approval, re-review the affected stage. Human approval of a plan is different from authorizing a live Terraform apply, deployment, or database migration.

**Specialist assignment:** Keep one main coding agent. Plan should identify the appropriate architect, Go/TypeScript, React, Kubernetes/Terraform skill by its actual installed name and say *why*; task breakdown should map relevant tasks to those specialists and targeted checks. A task label does not load a skill automatically; the implementing agent must discover and invoke the skill. Do not assume a skill available only in Codex is available in Claude Code. Zed's native agent profiles likewise do not automatically control external Claude/Codex agents. [Zed external agents](https://zed.dev/docs/ai/external-agents)

## Practical prompt sequence for packet 001

1. **Select bounded packet:** “Use `Docs/SRS.md`, `Docs/High Level Spec.md` and `Docs/Tasks.md` to identify the next bounded, independently verifiable workload. Propose packet 001's objective, source references, boundaries and candidate role. Show the proposal before making files if workload size is unclear.”
2. **Spec draft:** “`/agentic-sdd-spec` Draft `specs/001-<slug>/spec.md` for that workload. Link only relevant main-document sections, add slice-specific outcomes and acceptance checks, mark draft, and stop for my review. Do not write the plan or tasks yet.”
3. **Plan draft (after approval):** “I approve packet 001's spec at the current revision. `/agentic-sdd-plan` Draft `specs/001-<slug>/plan.md`, choosing actual installed specialist skills, design boundaries and tests. Stop for my review before task breakdown.”
4. **Task draft (after approval):** “I approve packet 001's plan at the current revision. `/agentic-sdd-tasks` Draft `specs/001-<slug>/tasks.md` with small executable steps, relevant specialist per task and verification. Stop for my review before coding.”
5. **Implementation (after approval):** “I approve packet 001's tasks at the current revision. `/agentic-sdd-implement` Take the first ready task, load its specialist into the main agent, implement, verify and summarize. Do not start another task until I review the result.”

Repeat with packet 002, 003 and so on. The *top-level* documents remain the source and roadmap; the numbered packet trio is the implementation contract for one manageable slice.

## Validation before using the repaired workflow

Run a dry exercise against one actual top-level task and one existing `specs/<nnn-slug>/` folder. Verify that (a) a new packet generates **exactly the three expected drafts unless justified extras are requested**, (b) each phase stops for actual human review, (c) roles name installed skills and their client availability, (d) approval is not inferred from an agent review, and (e) implementation handles one bounded task and targeted tests. Compare subscription usage, context compactions and human review minutes with an old packet. No savings percentage can be predicted from the uploaded skill text alone.

## Sources

- Local user-provided evidence: screenshot of the `kubelab/specs/` tree and the ten updated skills uploaded 24 September 2026.
- [Claude Code skills](https://code.claude.com/docs/en/skills) and [context-window explanation](https://code.claude.com/docs/en/context-window).
- [Codex agent skills](https://developers.openai.com/codex/skills).
- [GitHub Spec Kit existing-project guide](https://github.com/github/spec-kit/blob/main/docs/guides/existing-projects.md) (a precedent for bounded changes, not a requirement to remove the per-slice packet).
- [GitHub Spec Kit quickstart](https://github.com/github/spec-kit/blob/main/docs/quickstart.md) and [workflow review gates](https://github.com/github/spec-kit/blob/main/workflows/README.md).
- [Anthropic: Effective harnesses for long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents).
- [OpenAI: Rethinking skills and prompts](https://developers.openai.com/blog/rethinking-skills-and-prompts-for-gpt-6-astra).
- [Zed external agents](https://zed.dev/docs/ai/external-agents).
