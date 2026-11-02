# Model-free QA checklist — final human cycle

Companion to the [run sheet](vnext-human-cycle-run-sheet.md) and
[final usage plan](vnext-human-test-plan.md). Use the selected setup: Linux
container, disposable `/work`, **preview only**, **no native model**, 90-minute
timebox. Every item records **pass / fail / blocked / not run**; a skipped item
is not a pass. Nothing here authorizes a model call, provider request, live
config apply or real-home install.

## A. Pre-flight (host)

| ID | Do | Expected |
| --- | --- | --- |
| P1 | `make build` then `bin/agentic-sdd version` | Prints `agentic-sdd <version>`, commit and date; build succeeds. |
| P2 | `mkdir -p /tmp/opencode/vnext-cycle-work && SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work tests/vnext-container.sh` | Interactive shell; banner says no network/credentials. Host scratch dir is empty except anything you put there. |
| P3 | Inside: `command -v agentic-sdd; git --version; ls /src | head` | CLI resolves; `/src` readable. If `git` is absent, record evidence items `not run` in the container and use H1/H3 on the host. |
| P4 | `agentic-sdd --help` and `agentic-sdd opencode --help` | Help lists preview/apply/backups/logs/restore/opencode; opencode help lists candidate/preflight/config/session/**evidence**. |

Stop if the container has network, a host home mount, credentials or a writable
`/src`.

## B. Container checks (deterministic, no model)

Use `/tmp/home` as the install home so everything is ephemeral.

| ID | Do | Expected |
| --- | --- | --- |
| C1 | `agentic-sdd preview --home /tmp/home; find /tmp/home -type f | wc -l` | Ends with `Preview only. Run with apply (or --apply) to install.`; file count is `0` (or only pre-existing none). |
| C2 | `agentic-sdd apply --home /tmp/home; find /tmp/home/.agents/skills -name SKILL.md | wc -l; find /tmp/home/.agents/skills -type l | wc -l` | Install succeeds; **18** `SKILL.md`; **0** symlinks. |
| C3 | `agentic-sdd apply --home /tmp/home` again | `All skills are up to date.`; no new backup. |
| C4 | `printf '\nqa-marker\n' >> /tmp/home/.agents/skills/agentic-sdd-router/SKILL.md; agentic-sdd apply --home /tmp/home; agentic-sdd backups --home /tmp/home` | Apply replaces the edited skill and lists a newest backup ID. |
| C5 | `agentic-sdd restore <NEWEST_ID> --apply --home /tmp/home; grep -c qa-marker /tmp/home/.agents/skills/agentic-sdd-router/SKILL.md || true; diff /src/skills-files/agentic-sdd-router/SKILL.md /tmp/home/.agents/skills/agentic-sdd-router/SKILL.md` | Restore succeeds; marker count `0`; `diff` exits 0 (canonical bytes). |
| C6 | `mkdir -p /work/project && agentic-sdd opencode candidate --feature demo --build-path src > /tmp/candidate.jsonc; agentic-sdd opencode preflight --config /tmp/candidate.jsonc` | Candidate writes only `/tmp/candidate.jsonc`; preflight prints `online_ready: false`, **4** blockers, `target_runtime: 2.2.5`, and candidate model/profile summaries. |
| C7 | `agentic-sdd opencode config --workspace /work/project --candidate /tmp/candidate.jsonc; ls -a /work/project` | JSON with `original: "absent"`, `noop: false`, `destination: ".opencode/opencode.jsonc"`; **no** `.opencode` directory created. |
| C8 | `agentic-sdd opencode session prepare --workspace /work/project --executable /usr/local/bin/opencode` | `execution: prepared` plus non-empty admission `blockers` (13 on this source). No process starts. |
| C9 | `agentic-sdd opencode session run --workspace /work/project --executable /usr/local/bin/opencode --prompt "hello"` | `execution: blocked` with blockers; no model/native process. This is the correct fail-closed result, not a failure of the test. |
| C10 | If `git` exists: `cd /work/project && git init -q && git config user.email qa@example.invalid && git config user.name QA && printf 'hi\n' > code.txt && git add . && git commit -qm fixture`, then `agentic-sdd opencode evidence snapshot --workspace /work/project --scope .` | `execution: observed`; `identity.diff` matches `sha256-v1:[0-9a-f]{64}`; `identity.base_revision` is 40 hex characters. |
| C11 | `agentic-sdd opencode evidence snapshot --workspace /work/project --scope ../outside` and `--scope '*'` | Both exit 2 with an error; no identity printed. |
| C12 | `agentic-sdd opencode evidence validate --receipt /nope --workspace /work/project --identity /nope --contract /nope --result /nope --evidence-root /tmp` | Exit 2, bounded error; no state written. |

Optional config apply is **not** in this checklist. Applying to `/work/project`
needs a separate explicit go-ahead and must be followed by rollback.

## C. Host-side deterministic checks

| ID | Do | Expected |
| --- | --- | --- |
| H1 | `go test ./internal/skillsync ./internal/harnessevidence ./internal/opencodeadapter ./cmd/agentic-sdd -count=1` | All packages `ok`. |
| H2 | `make test && make vet` | Both pass. |
| H3 | `find skills-files -name SKILL.md | wc -l; find skills-files -type f | wc -l; find skills-files -type l | wc -l` | `18` skills, `28` files, `0` symlinks. |
| H4 | `git diff --check` and confirm `docs/vnext-human-test-plan.md`, `docs/vnext-human-cycle-run-sheet.md` exist | No whitespace errors; both documents present. |

## D. Static behaviour review (no model, read-only)

Each item is a grep/read check; record the exact line you inspected.

| ID | Inspect | Expected finding |
| --- | --- | --- |
| S1 | `skills-files/agentic-sdd-implement/SKILL.md` | States one batch, then `awaiting human review` with next action; does not auto-start the next batch. |
| S2 | `skills-files/agentic-sdd-verification-review/SKILL.md` | States review-only requests change nothing (no code/docs/approval/task edits) and the verdict never replaces human approval. |
| S3 | `skills-files/agentic-sdd-grilling/SKILL.md` | No more than three independent material questions per round; dependent questions deferred. |
| S4 | `skills-files/agentic-sdd-questionnaire/SKILL.md` | Drafting is not permission to send/contact; answers preserve source, uncertainty and owner. |
| S5 | `skills-files/agentic-sdd-handoff/SKILL.md` | A handoff never supplies a new batch, live permission or approval. |
| S6 | `skills-files/agentic-sdd-wait-what/SKILL.md` | Explanation only; changes no files, verdicts, approvals, task state or batch selection. |
| S7 | `skills-files/agentic-sdd-teach/SKILL.md` | Persistent teaching files are opt-in with a separately chosen workspace; no automatic conversion of the repo. |
| S8 | `skills-files/agentic-sdd-writing-for-agents/SKILL.md` | Conditional pointers, one owner per rule, no automatic repository-wide rewrite. |

## E. Deferred — model-dependent scenarios (record `not run`)

The selected cycle has **no model**, so these cannot be executed here. Keep the
prompts for a later, separately authorized session; do not run them in the
container.

| ID | Prompt to use later | What to observe |
| --- | --- | --- |
| D1 | “Fix this small bug in the safe project: <reproduction>. Do not create feature docs.” | Focused fix; no `specs/` tree; no feature interview. |
| D2 | “Draft spec/plan/tasks for this small feature. Stop for my combined review.” | Three coherent documents; no implementation; waits for approval. |
| D3 | “I approve. Implement batch B1, verify it, and stop for my review.” | One batch only; review evidence; then waits. |
| D4 | “Review this change only; do not edit anything.” | Findings only; no code/docs/state changes. |
| D5 | Interrupt a harmless task, then “resume”. | Authority and files re-read; no implicit new batch; partial work preserved. |
| D6 | “Diagnose this failure before changing code.” | Reproduces, separates cause from symptom, no unrelated rewrite. |
| D7 | “Help me decide between these two options; ask only what you need.” | ≤3 independent questions; reuses settled answers; no implementation. |
| D8 | “Draft questions for <role> about <gap>; don't send anything.” | Recipient-specific draft with answer stubs; no sending. |
| D9 | “Wait, what is the current status?” | Short explanation; no state change. |
| D10 | “Teach me <concept> from this code; don't create files.” | Focused explanation/example; no persistent workspace. |
| D11 | Real native pending-human check | **Blocked by admission**; record `not run — native admission blocked`. |

## F. Closeout

```text
Sitting/date:
Setup/version and scratch path (outside Git):
B/C/D/H/S results:
Observed deviations and evidence (sanitized):
Finding IDs / next decision:
Feature status: implementation delivered; human usage validation not run
```

- Reconcile results into the feature continuation only when authorized.
- Do not mark 005 Ready, supported or beneficial from this checklist.
- Preserve 004 findings and the native-admission block; any fix/retest is a new
  user decision.
