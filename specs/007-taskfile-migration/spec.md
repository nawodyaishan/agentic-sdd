# Replace Makefile with Taskfile

Revision: r1 draft, 2026-11-05.

## Outcome and sources

Replace the repository's Make-based developer interface with a Go Task Taskfile, preserving the intended behavior of its commands and the trusted verification and release boundaries. Sources: `Makefile`, `README.md`, `AGENTS.md`, `harness/checks.json`, `harness/trusted-ci.md`, `.github/workflows/{ci,release}.yml` and `tests/harness/trusted_ci.py`. This repository has no top-level SRS or roadmap. The existing feature 006 pilot remains deferred; this feature does not resolve its findings.

## Scope and acceptance

1. Provide documented Task equivalents for every current Make target, including help, preview, apply, backups, logs, test, vet, Docker e2e, hooks, module verification, tidy check, builds, verification, trusted verification, tag and release snapshot. Remove `Makefile` after all required consumers are migrated. Explicitly document the breaking change from `make` to `task`.
2. Preserve command effects, failure handling and caller-selectable variables, including `GOCACHE`, `SDD_OUTPUT`, `SDD_TRUSTED`, `SDD_CANDIDATE`, `V` and `MSG`. Default invocation must not install skills, tag, publish or otherwise perform a live operation. No real-home apply or real tag is part of development verification.
3. Preserve required-check semantics: the six required checks, offline Go execution settings, external private fresh evidence, trusted-base bootstrap and candidate isolation must not be weakened. Migrate contract local recipes, protected-input identification and associated fixtures without treating the candidate Taskfile as trusted bootstrap. Preserve check IDs, executable argv and status semantics unless explicitly reconciled in review.
4. Update current user/maintainer documentation, repository instructions, relevant scripts and release CI to use Task and provision an explicitly pinned, reviewed Task release for the release workflow. Preserve historical specs, handovers and prior evidence as historical records, rather than rewriting what was actually run. Do not add Task as a runtime dependency of the distributed CLI.
5. Add meaningful automated checks for command/argument and safety parity, including default behavior, required input errors, evidence directory handling and protected-input coverage. Run focused tests plus `task test`, `task vet`, `task verify` and `git diff --check` where prerequisites permit; identify unavailable Docker, trusted CI, or release prerequisites rather than claiming success.

## Exclusions and live operations

No provider/model evaluation, real-home install, real tag, release, credential change, branch-protection change or unrelated feature work. The requested eventual commit and push are separate from implementation verification and must exclude unrelated existing changes and private local material. A push is a live operation; the user's explicit request is relevant authorization only for the intended changes and destination, not for tags or release publishing.

## Approval

Decision: **approved**. The user said “apporved docs” on 2026-11-05 after the r1 spec, plan and tasks were presented, and previously requested “implement all batches”; B1 is authorized. This does not authorize real-home apply, a tag or release.

On 2026-11-05 the user said “b1 approved finis b2 also change ci cd to taskfile”. This accepts B1 for the record and authorizes the scoped CI/CD Taskfile continuation in B2; it does not waive 007-F1 or establish that its missing transition evidence passed. The user also requested Exa and Context7 for this work.

On 2026-11-05 the user selected direct adoption of the migration on the trusted branch rather than a staged PR transition, and confirmed that the existing local commit ahead of origin may be included in the requested push. This accepts the old-base PR-gate evidence gap for that adoption; it does not make the gate pass, change branch protection, authorize a tag or release, or resolve other findings.
