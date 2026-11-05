# Taskfile migration plan

Draft r1; based on [spec r1](spec.md). Approval is recorded only in spec §Approval.

## Design and boundaries

- Replace the Makefile with a Go Task v3 `Taskfile.yml`; map all existing targets to named tasks and make the default task display the task list. Keep the current target-level effects, ordering, parameter defaults, and error semantics. Explicitly test Task variable precedence and shell quoting rather than assuming Make syntax is equivalent.
- Translate the runner entry points, including owner-private temporary evidence and the `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off` controls. Keep direct trusted check argv independent of candidate task definitions. Preserve `verify-trusted`'s reviewed-base bootstrap and separate candidate requirements. Account for Task's shell differences through executable checks, not a visual translation alone.
- Update `harness/checks.json` local recipes, trusted bootstrap protected paths, targeted tests and current operational docs. Review base-owned trusted gate logic and fixtures before changing any checks; retaining reviewed-base enforcement across the transition takes priority over using the candidate's Taskfile. Historical observations are not rewritten.
- Update `.github/workflows/release.yml` to provision a reviewed pinned Task version before `task verify`. Pin any external action to a reviewed commit and the Task release to an exact version; document any additional network/dependency risk. Decide the exact pin from locally available version and verifiable release/action provenance during implementation; if it cannot be safely verified, report the blocker instead of using a floating reference. Preserve the separate trusted CI workflow's existing base-owned bootstrap and pinned worker image unless a concrete change is required.
- Adjust `README.md`, current harness/integration guidance, `AGENTS.md` and any active scripts to Task commands. Preserve pre-existing unrelated changes in `AGENTS.md`; do not overwrite them. Keep `task apply`, `task tag` and publication actions protected by the same explicit human authorization as before.

## Tools, guidance and verification

Use `cicd-automation-workflow-automate` for workflow changes and `agentic-sdd-writing-for-agents` for `AGENTS.md` (both available in this session); load only when implementing. Use local source inspection, Context7 Task documentation and repository checks. Context7 identifies `version: '3'`, `vars`, `NAME=value` overrides, and official CI setup with a version pin; check precise runtime behavior on the locally installed Task before relying on it. Documentation: https://github.com/go-task/task/blob/main/website/src/latest/docs/guide/variables.md and https://github.com/go-task/task/blob/main/website/src/latest/docs/installation.md.

Test contract parity and negative cases in disposable directories; ensure tests never apply to a real home, create real tags or invoke publish. Validate YAML and release workflow by available offline tools; run a trusted-base isolated gate only if its prerequisites and authority are available. `task test`, `task vet`, `task verify`, and `git diff --check` are expected. Any inability to verify workflow action provenance, trusted CI or Docker must be called out explicitly, not treated as a pass.

## Risks

Deleting Makefile breaks existing developer invocations by design. Variable precedence, templating and quoting may differ; Task prerequisites/version may affect contributors and release CI; a changed trusted-input allowlist can weaken review or fail closed. CI builds may need to fetch the pinned Task distribution; this is distinct from permission to publish. Existing dirty/untracked files, including private handovers, are not part of this feature without explicit identification. A commit and push must contain only authorized migration changes and may happen only after the batch has been reviewed and the appropriate continuation is clear.
