# Specialist and tool selection

These are candidate skill names, not guaranteed installations. Resolve availability and native invocation restrictions from the active client catalog before selection; do not scan every skill body. Keep installation inventory and machine paths in host configuration, outside this portable reference.

## Specialist candidates

| Work semantics | Candidate installed skill | Selection and focused verification |
|---|---|---|
| Go services/CLI | `golang-pro` | Follow the repo's Go version and patterns; formatting and focused tests, race checks for concurrency risk. |
| TypeScript Node API/service | `nodejs-backend-patterns` | Express/Fastify backend guidance when installed. Typecheck and focused validation/error/contract tests. |
| Existing Express/Prisma/Zod stack with its prescribed conventions | `backend-dev-guidelines` | Use only when its BaseController, Sentry, unifiedConfig and layered architecture requirements fit the repo; do not impose that stack on another app. |
| React components/hooks/interactions | `react-patterns` | Check installed React version; preserve client/server conventions. Component/interaction and accessibility checks. |
| Kubernetes workload manifests | `k8s-manifest-generator` | Render Helm/Kustomize when used, validate schemas/policy for the target cluster version. Arbitrary YAML is not Kubernetes. |
| Kubernetes platform/topology or deployment architecture | `kubernetes-architect` | Review RBAC/network/blast radius, rollout and recovery; do not redesign a cluster for a manifest edit. |
| Terraform/OpenTofu modules/configuration | `terraform-specialist` | Formatting/validation and plan review for the authorized workspace/account; assess replacement, state and recovery. A plan can access remote systems; applying it needs execution authorization. |

Select by behavior and repository context, not extension alone: TypeScript backend and React need different expertise. Load a second specialist only when crossing a real boundary, not every entry in this table. Specialist model hints do not request a new model or agent.

**"No additional specialist needed" is a valid assignment.** When repository guidance and existing patterns adequately cover the work, record that explicitly with a one-line reason instead of attaching a specialist for form's sake. Not every task requires one.

## Tool selection

Assign tools alongside specialists in `plan.md`, as conditional selections — not mandatory calls. Resolve the actual tool identifier from the active client; do not assume a provider's identifier, server name or network availability.

| Need | Conditional selection |
|---|---|
| Local code discovery and cross-module dependencies | CodeGraph where the repo has `.codegraph/`; otherwise targeted local search |
| Uncertain library/framework/API behavior | A versioned documentation tool such as Context7, when the question is concrete |
| External technical research beyond local sources | A web search/fetch tool such as Exa |
| Browser-observable behavior of a running app | Playwright, when the change actually affects it |
| Build, test, lint, schema/policy validation, `terraform plan` | The repository's own commands and required gates |

Use a tool for a concrete unanswered question, bound the returned context, and reuse still-valid evidence.

## When something is unavailable

If a candidate specialist or tool is missing, disabled, manual-only, unreachable or unsuitable for the repository, report that specific gap and continue with an adequate fallback, stating the substitution plainly. Respect native invocation controls; do not bypass a manual-only restriction by reproducing or reading its procedure another way — ask for its manual invocation when required. Do not force the Express conventions onto another backend, redesign a cluster for simple YAML, silently substitute a broader specialist, invent an identifier, or assume another client's installation is available. Never install specialists automatically.

## Where assignments live

At planning, record the actual installed name, client availability, the tools that work needs and why each fits — once, in `plan.md`. In `tasks.md`, record only task-specific exceptions. At implementation, load the selected skill's guidance in the main agent; a written role label alone does not load it. Preserve user and repository constraints if specialist advice conflicts. Do not spawn subagents unless the user requests them.
