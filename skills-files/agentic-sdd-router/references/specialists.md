# Specialist selection

These are candidate skill names, not guaranteed installations. Resolve availability and native invocation restrictions from the active client catalog before selection; do not scan every skill body. Keep installation inventory and machine paths in host configuration, outside this portable reference.

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

If a candidate is missing, disabled, manual-only or unsuitable for the repository, report that specific gap. Respect native invocation controls; do not bypass a manual-only restriction by reproducing or reading its procedure another way. Ask for its manual invocation when required, or continue with adequate repository guidance. Do not force the Express conventions onto another backend, redesign a cluster for simple YAML, silently substitute a broader specialist, or assume another client’s installation is available. Never install specialists automatically.

At packet planning, record the actual installed name, client availability and why it fits. In packet tasks, assign that skill and focused checks to relevant work. At implementation, invoke the selected skill in the main agent; a written role label alone does not load it. Preserve user and repository constraints if specialist advice conflicts.
