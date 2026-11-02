# vNext final human test guide — feature 005

**Status: guide prepared; the cycle has NOT been run.**
This single guide is the step-by-step way to test the delivered 005 candidate in
the selected setup: a Linux container, a disposable project, **preview only**,
**no native model/provider**, 90-minute timebox. Run each step, record the
result, and stop on the first unexpected write or missing prerequisite.

Companions:

- [Final human usage plan](vnext-human-test-plan.md) — scenario rationale,
  authority and acceptance criteria.
- [Run sheet](vnext-human-cycle-run-sheet.md) — fill-in session identity and
  result rows.
- [QA checklist](vnext-model-free-qa-checklist.md) — the same checks as compact
  IDs if you prefer ticking boxes.

This guide does not authorize a model call, provider request, live config apply,
real-home install or release. A green run is deterministic evidence, not support
certification.

## 1. What this cycle can and cannot prove

**Can be tested model-free:** installer preview/apply/no-op/backup/restore,
package closure, candidate/preflight, config preview (no write), strict
admission staying blocked, scoped evidence identity and refusal cases, host Go
gates, and static skill-rule review.

**Cannot be tested here:** real model-originated tool/permission behavior,
native pending-human handling, native cancellation/recovery, adoption on the
real setup, macOS, billing/provenance, and useful benefit. Those stay `not run`
or require a separate explicitly authorized model session.

## 2. Setup

From the repository root:

```sh
make build
bin/agentic-sdd version
mkdir -p /tmp/opencode/vnext-cycle-work
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work tests/vnext-container.sh
```

What the recipe enforces: local image only (`--pull=never`), `--network=none`,
read-only root and repo, no host home or credentials, dropped capabilities,
PID/memory/CPU limits, host UID, tmpfs home, and one scratch directory mounted
read-write at `/work`. It refuses the implementation checkout as the workspace.

Inside the container, `agentic-sdd` is at `/usr/local/bin/agentic-sdd` and the
repo is read-only at `/src`. Use `/tmp/home` for all install state so nothing
persists.

If `git --version` fails inside the image, skip the evidence identity steps and
run the host checks in §7 instead. Do not add network or credentials to fix it.

## 3. Pre-flight

| # | Command | Expected |
| --- | --- | --- |
| 1 | `command -v agentic-sdd; ls /src | head` | CLI resolves; repo readable. |
| 2 | `agentic-sdd --help` | Lists preview/apply/backups/logs/restore/opencode. |
| 3 | `agentic-sdd opencode --help` | Lists candidate, preflight, config, session (**prepare/run/pending**) and **evidence** (snapshot/validate/export/receipt). |
| 4 | `agentic-sdd version` | Prints version, commit and date. |

Stop if `/src` is writable, the network is reachable, or a host home/credential
path is mounted.

## 4. Installer safety (the main model-free value)

| # | Command | Expected |
| --- | --- | --- |
| 5 | `agentic-sdd preview --home /tmp/home` | Text plan ending with `Preview only. Run with apply (or --apply) to install.` No files under `/tmp/home/.agents`. |
| 6 | `agentic-sdd apply --home /tmp/home` | Install succeeds; backup directory created under `/tmp/home/.agentic-sdd/backups`. |
| 7 | `find /tmp/home/.agents/skills -name SKILL.md | wc -l` | **18** |
| 8 | `find /tmp/home/.agents/skills -type l | wc -l` | **0** |
| 9 | `agentic-sdd apply --home /tmp/home` again | `All skills are up to date.` No new backup. |
| 10 | `printf '\nqa-marker\n' >> /tmp/home/.agents/skills/agentic-sdd-router/SKILL.md` then `agentic-sdd apply --home /tmp/home` | Edited skill is replaced; a new backup is created. |
| 11 | `agentic-sdd backups --home /tmp/home` | Lists the newest backup first; note its ID. |
| 12 | `agentic-sdd restore <NEWEST_ID> --apply --home /tmp/home` | Restore succeeds. |
| 13 | `grep -c qa-marker /tmp/home/.agents/skills/agentic-sdd-router/SKILL.md` then `diff /src/skills-files/agentic-sdd-router/SKILL.md /tmp/home/.agents/skills/agentic-sdd-router/SKILL.md` | Marker count `0` (or grep exits 1); `diff` exits 0. |

If step 6 fails, stop: do not retry with a different home or a real home.

## 5. Inactive OpenCode candidate and preview

| # | Command | Expected |
| --- | --- | --- |
| 14 | `mkdir -p /work/project` | Empty scratch project. |
| 15 | `agentic-sdd opencode candidate --feature demo --build-path src > /tmp/candidate.jsonc` | Candidate JSON only on stdout; no other write. |
| 16 | `agentic-sdd opencode preflight --config /tmp/candidate.jsonc` | `online_ready: false`, **4** blockers, `target_runtime: 2.2.5`, one candidate model/profile summary. |
| 17 | `agentic-sdd opencode config --workspace /work/project --candidate /tmp/candidate.jsonc` | `original: "absent"`, `noop: false`, `destination: ".opencode/opencode.jsonc"`. |
| 18 | `ls -a /work/project` | **No** `.opencode` directory — preview wrote nothing. |

Do **not** run `config apply` in this cycle. It is a scratch write and needs a
separate explicit go-ahead plus rollback.

## 6. Admission and evidence

| # | Command | Expected |
| --- | --- | --- |
| 19 | `agentic-sdd opencode session prepare --workspace /work/project --executable /usr/local/bin/opencode` | `execution: prepared`; non-empty admission blockers (13 on this source). No process starts. |
| 20 | `agentic-sdd opencode session run --workspace /work/project --executable /usr/local/bin/opencode --prompt "hello"` | `execution: blocked` with blockers. **Blocked is correct** — strict admission is deliberately denied. |
| 21* | If `git` exists: `cd /work/project && git init -q && git config user.email qa@example.invalid && git config user.name QA && printf 'hi\n' > code.txt && git add . && git commit -qm fixture` | Repository created. |
| 22* | `agentic-sdd opencode evidence snapshot --workspace /work/project --scope .` | `execution: observed`; `identity.diff` = `sha256-v1:` + 64 hex; `identity.base_revision` = 40 hex. |
| 23* | `agentic-sdd opencode evidence snapshot --workspace /work/project --scope ../outside` | Exit 2, error, no identity. |
| 24* | `agentic-sdd opencode evidence snapshot --workspace /work/project --scope '*'` | Exit 2, error, no identity. |
| 25* | `agentic-sdd opencode evidence validate --receipt /nope --workspace /work/project --identity /nope --contract /nope --result /nope --evidence-root /tmp` | Exit 2, bounded error, no state written. |

`*` Skip only if git is absent; record `not run` and use §7.

## 7. Host-side deterministic checks

Run on the host in the repository checkout:

| # | Command | Expected |
| --- | --- | --- |
| 26 | `go test ./internal/skillsync ./internal/harnessevidence ./internal/opencodeadapter ./cmd/agentic-sdd -count=1` | All packages `ok`. |
| 27 | `make test && make vet` | Both pass. |
| 28 | `find skills-files -name SKILL.md | wc -l; find skills-files -type f | wc -l; find skills-files -type l | wc -l` | `18`, `28`, `0`. |
| 29 | `git diff --check` | No whitespace errors. |

## 8. Static rule review (no model)

Read each file and confirm the stated rule exists. Record the exact line.

| # | File | Confirm |
| --- | --- | --- |
| 30 | `skills-files/agentic-sdd-implement/SKILL.md` | One batch, then `awaiting human review`; does not auto-start the next batch. |
| 31 | `skills-files/agentic-sdd-verification-review/SKILL.md` | Review-only changes nothing; verdict never replaces human approval. |
| 32 | `skills-files/agentic-sdd-grilling/SKILL.md` | At most three independent material questions per round. |
| 33 | `skills-files/agentic-sdd-questionnaire/SKILL.md` | Drafting is not permission to send/contact; answers preserve source/uncertainty/owner. |
| 34 | `skills-files/agentic-sdd-handoff/SKILL.md` | A handoff never grants a new batch or live permission. |
| 35 | `skills-files/agentic-sdd-wait-what/SKILL.md` | Explanation only; no file/verdict/approval/task/batch change. |
| 36 | `skills-files/agentic-sdd-teach/SKILL.md` | Persistent teaching files are opt-in on a separately chosen workspace. |
| 37 | `skills-files/agentic-sdd-writing-for-agents/SKILL.md` | Conditional pointers, one owner per rule, no repository-wide rewrite. |

## 9. Deferred — model-dependent scenarios

These are listed for a later, separately authorized model session. In this cycle
record each as **not run**. Do not paste them into the model-free container.

| # | Prompt for later | Observe |
| --- | --- | --- |
| D1 | “Fix this small bug in the safe project: <reproduction>. Do not create feature docs.” | Focused fix; no feature docs or interview. |
| D2 | “Draft spec/plan/tasks for this small feature. Stop for my combined review.” | Three coherent documents; no implementation. |
| D3 | “I approve. Implement batch B1, verify it, and stop for my review.” | One batch, evidence, then waits. |
| D4 | “Review this change only; do not edit anything.” | Findings only; no state change. |
| D5 | Interrupt a harmless task, then “resume”. | Authority/files re-read; partial work preserved; no implicit batch. |
| D6 | “Diagnose this failure before changing code.” | Reproduces, separates cause from symptom. |
| D7 | “Help me decide between these options; ask only what you need.” | ≤3 independent questions; no implementation. |
| D8 | “Draft questions for <role> about <gap>; don't send anything.” | Recipient-specific draft; no sending. |
| D9 | “Wait, what is the current status?” | Short explanation; no state change. |
| D10 | “Teach me <concept> from this code; don't create files.” | Focused teaching; no persistent workspace. |
| D11 | Real native pending-human check. | `not run — native admission blocked`. |

## 10. Recording

```text
Sitting/date:
Setup/version and scratch path (outside Git):
Steps run / skipped:
Results (pass | fail | blocked | not run) and evidence (sanitized):
Finding IDs / next decision:
Feature status: implementation delivered; human usage validation not run
```

Record deviations immediately. Do not rerun a failed step blindly; reconcile the
cause first. Reconcile notes into the feature continuation only when authorized.

## 11. Stop and rollback

- **Stop** on: unexpected write, permission prompt, network access, missing
  credential isolation, unexpected cost, or a result that contradicts the
  expected column.
- **Skills:** the container home is ephemeral; discard it. On a real setup,
  `agentic-sdd backups` / `agentic-sdd restore <ID> --apply` is the only
  rollback.
- **Config:** preview only here; use the digest-checked
  [adoption/rollback procedure](../integrations/opencode/adoption.md) only under
  separate authorization.
- **Evidence:** keep sanitized external evidence only under the documented
  retention policy; never copy auth stores or raw logs into Git.

## 12. What “pass” means

A pass means the recorded step produced the expected observation in this
container/setup. It does **not** mean the feature is ready, supported, safe for
private data, or beneficial, and it does not close the carried 004 findings or
the native-admission block. Final status remains **implementation delivered;
human usage validation not run** until actual human evidence is recorded.
