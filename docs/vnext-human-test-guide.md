# vNext final human test guide — feature 005

**Status: guide prepared; the cycle has NOT been completed.**
This is the step-by-step way to test the delivered 005 candidate in the selected
setup: a Linux container, a disposable project, **preview only**, **no native
model/provider**, 90-minute timebox. Run each numbered step, record the result,
and stop on the first unexpected write or missing prerequisite.

Companions: [final usage plan](vnext-human-test-plan.md) (scenario rationale and
acceptance), [run sheet](vnext-human-cycle-run-sheet.md) (fill-in identity and
result rows), [compact checklist](vnext-model-free-qa-checklist.md).

This guide does not authorize a model call, provider request, live config apply,
real-home install or release. A green run is deterministic evidence, not support
certification.

## 0. Facts verified on 2026-11-02 (read before starting)

- A local `make build` reports `agentic-sdd dev`, `commit: none`, `date:
  unknown`. That is expected for an unversioned local build.
- One canonical skill set is **18 skills / 28 files**. `apply` copies each skill
  into **four client destinations** (`agents`, `codex`, `claude`, `agy`), so a
  full install creates **72 skill directories**, not 18 files. Preview therefore
  prints 72 `install` lines.
- The default container image (`python:3.13-slim`) has **no `git`**. Evidence
  identity steps are optional and are normally run on the host instead (§6).
- The host **OpenCode v2.2.5** binary is mounted read-only at
  `/usr/local/bin/opencode` **by default**; it is not downloaded or installed
  from the network. `opencode --version` works, while sessions remain denied.
- `agentic-sdd apply` prints `Installed N skills; backup: …` even when the
  wording means “replaced and backed up” for changed copies.
- `restore` first backs up the **current** installed tree and then restores the
  selected backup. Restoring the backup taken before an apply therefore brings
  the **pre-apply edited** copy back, not the canonical copy. §4 steps 10–14 test
  this correctly.
- `agentic-sdd opencode config` is a read-only preview and is idempotent; two
  runs produce identical output and no `.opencode` directory.

## Fast path — one copy-paste batch

From the repository root, this single block builds the CLI, creates a scratch
workspace and runs all model-free checks in the sandbox:

```sh
make build
mkdir -p /tmp/opencode/vnext-cycle-work
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work \
  tests/vnext-container.sh --batch
```

Expected tail:

```text
PASS cli-version
PASS opencode-version
PASS preview-no-write
PASS apply-72-zero-links
PASS apply-noop
PASS edit-replaced-with-backup
PASS canonical-after-apply
PASS restore-pre-apply-copy
PASS restore-then-canonical
PASS candidate-preflight
PASS config-preview-no-write
PASS admission-blocked
SKIP evidence-snapshot: git absent in this image; run the host fallback
SKIP evidence-refuses-traversal: git absent in this image

Summary: 12 passed, 0 failed, 2 skipped
```

Exit code 0 means every executed check passed. Any `FAIL` line means stop and
record it; do not rerun until the cause is understood. The batch uses the same
no-network, no-credential container as the manual steps, and it wipes only
`/work/qa-project` and `/tmp/home` inside the container.

Already inside the container? The same batch is one command:

```sh
sh /src/tests/vnext-model-free-checks.sh
```

The numbered sections below are the manual equivalent for inspection,
troubleshooting and the host-side checks. The batch already covers them except
git-based evidence, which is skipped in the default image.

## 1. What this cycle can and cannot prove

**Model-free:** installer preview/apply/no-op/backup/restore, package closure,
candidate/preflight, config preview (no write), strict admission staying blocked,
scoped evidence identity and refusals, host Go gates, static skill-rule review.

**Not here:** real model-originated tools/permissions, native pending-human
handling, native cancellation/recovery, adoption on the real setup, macOS,
billing/provenance and useful benefit. Those stay `not run` or require a separate
explicitly authorized model session.

## 2. Setup

From the repository root:

```sh
make build
bin/agentic-sdd version
mkdir -p /tmp/opencode/vnext-cycle-work
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work tests/vnext-container.sh
```

Expected: `dev`/`none`/unknown version, then an interactive shell whose banner
says no network/credentials/host home and prints `OpenCode: opencode v2.2.5
(mounted read-only; no provider/network)`. The recipe enforces `--pull=never`,
`--network=none`, read-only root and repo, no host home or credential mount,
capability drop, resource limits, host UID, tmpfs home and one read-write scratch
mount at `/work`. It refuses the implementation checkout as the workspace.
`NO_OPENCODE=1` omits the binary; `OPENCODE_BINARY=/path` selects a different
one. No network install occurs.

Inside the container set the home path once and use it exactly:

```sh
H=/tmp/home
```

A typo such as `/tmp/hom` previews/installs a different home; every command below
uses `"$H"`.

If `git --version` fails (the default image), skip §6 in the container and run
its evidence checks on the host (§6.2). Do not add network or credentials to fix
the image.

## 3. Pre-flight

| # | Command | Expected |
| --- | --- | --- |
| 1 | `command -v agentic-sdd` | `/usr/local/bin/agentic-sdd`. |
| 2 | `ls /src` | Readable repository listing (read-only mount). |
| 3 | `agentic-sdd --help` | Lists preview/apply/backups/logs/restore/opencode. |
| 4 | `agentic-sdd opencode --help` | Lists candidate, preflight, config, session (prepare/run/pending) and evidence (snapshot/validate/export/receipt). |
| 5 | `agentic-sdd version` | `agentic-sdd dev`, `commit: none`, `date: unknown`, `go version: unknown`. |
| 6 | `opencode --version` | `opencode v2.2.5` (read-only mount; no provider/network). |

Stop if `/src` is writable, the network is reachable, or a host home/credential
path is mounted.

## 4. Installer safety

Use one shell; `H` must still be set from §2.

**Step 6 — preview (writes nothing)**

```sh
agentic-sdd preview --home "$H"
```

Expected: 72 lines of `install <client>/<skill>` (4 clients × 18 skills), then:

```text
Preview only. Run with apply (or --apply) to install.
```

Check nothing was created:

```sh
test ! -e "$H/.agents/skills" && echo no-install
```

**Step 7 — apply**

```sh
agentic-sdd apply --home "$H"
```

Expected: 72 `install` lines, then `Installed 72 skills; backup:
"$H/.agentic-sdd/backups/<ID>"` and a `Log:` line.

**Step 8 — verify counts and kinds**

```sh
find "$H/.agents/skills" -name SKILL.md | wc -l
find "$H/.codex/skills" -name SKILL.md | wc -l
find "$H/.claude/skills" -name SKILL.md | wc -l
find "$H/.gemini/antigravity-cli/skills" -name SKILL.md | wc -l
find "$H/.agents" "$H/.codex" "$H/.claude" "$H/.gemini" -type l | wc -l
```

Expected: `18`, `18`, `18`, `18`, `0`.

**Step 9 — second apply is a no-op**

```sh
agentic-sdd apply --home "$H"
```

Expected: 72 `up to date` lines and `All skills are up to date.`; no new backup.

**Step 10 — edit one installed copy, then apply**

```sh
printf '\nqa-marker\n' >> "$H/.agents/skills/agentic-sdd-router/SKILL.md"
agentic-sdd apply --home "$H"
```

Expected: one `replace + back up agents/agentic-sdd-router` line, 71 `up to date`
lines, then `Installed 1 skills; backup: …`.

**Step 11 — the canonical copy replaced the edit**

```sh
grep -c qa-marker "$H/.agents/skills/agentic-sdd-router/SKILL.md" || true
diff /src/skills-files/agentic-sdd-router/SKILL.md "$H/.agents/skills/agentic-sdd-router/SKILL.md"
```

Expected: `0` and `diff` exit 0 (canonical bytes restored).

**Step 12 — list backups and capture the newest ID**

```sh
agentic-sdd backups --home "$H"
```

Expected: two entries, newest first. The newest summary is `1 replaced, 0
installed across agents`; the older is `0 replaced, 72 installed across agents,
agy, claude, codex`. Record the newest ID.

**Step 13 — restore the pre-apply backup**

```sh
agentic-sdd restore <NEWEST_ID> --apply --home "$H"
```

Expected: `replace + back up agents/agentic-sdd-router`, then `Restored 1
skills; backup: …`. Restore itself creates a third backup of the canonical copy
it replaced.

**Step 14 — confirm restore returns the edited copy**

```sh
grep -c qa-marker "$H/.agents/skills/agentic-sdd-router/SKILL.md"
diff /src/skills-files/agentic-sdd-router/SKILL.md "$H/.agents/skills/agentic-sdd-router/SKILL.md"
```

Expected: `1` and `diff` exit 1 (they differ). **This is correct**: restore
rolled back the apply, so the pre-apply edited copy is back. To return to
canonical, apply again:

```sh
agentic-sdd apply --home "$H"
grep -c qa-marker "$H/.agents/skills/agentic-sdd-router/SKILL.md" || true
```

Expected: one `replace + back up` line, then `0`.

## 5. Inactive OpenCode candidate and preview

```sh
mkdir -p /work/project
agentic-sdd opencode candidate --feature demo --build-path src > /tmp/candidate.jsonc
agentic-sdd opencode preflight --config /tmp/candidate.jsonc
```

Expected preflight (exact shape observed on this source):

```json
{
  "version": 1,
  "online_ready": false,
  "target_runtime": "2.2.5",
  "candidate_profiles": ["sdd-build", "sdd-draft", "sdd-review"],
  "candidate_commands": [
    "sdd-build", "sdd-draft", "sdd-grill-me", "sdd-grilling",
    "sdd-handoff", "sdd-questionnaire", "sdd-resume", "sdd-review",
    "sdd-teach", "sdd-wait-what", "sdd-writing-for-agents"
  ],
  "candidate_models": ["openrouter/nvidia/nemotron-3-ultra-550b-a55b:free"],
  "observation": "not-observed",
  "blockers": [
    "providers-disabled",
    "native-enforcement-not-certified",
    "strict-token-egress-accounting-unverified",
    "no-live-operation-authority"
  ]
}
```

Config preview:

```sh
agentic-sdd opencode config --workspace /work/project --candidate /tmp/candidate.jsonc
```

Expected: `original: "absent"`, `noop: false`, `destination:
".opencode/opencode.jsonc"`, and a 64-hex `candidate` digest. Run it twice; the
output is identical. Confirm no write:

```sh
test ! -e /work/project/.opencode && echo no-config-write
```

Do **not** run `config apply` in this cycle. It is a scratch write and needs a
separate explicit go-ahead plus rollback.

## 6. Admission and evidence

### 6.1 Admission stays blocked

```sh
agentic-sdd opencode session prepare --workspace /work/project --executable /usr/local/bin/opencode
agentic-sdd opencode session run --workspace /work/project --executable /usr/local/bin/opencode --prompt "hello"
```

Expected: `execution: prepared` with non-empty blockers (13 on this source), then
`execution: blocked` with blockers. **Blocked is correct**; no process or model
may start.

### 6.2 Evidence identity — host fallback (default image has no git)

Run on the host with the built binary:

```sh
repo=$(pwd)
d=$(mktemp -d /tmp/opencode/evidence.XXXXXX)
git -C "$d" init -q
git -C "$d" config user.email qa@example.invalid
git -C "$d" config user.name QA
printf 'hi\n' > "$d/code.txt"
git -C "$d" add .
git -C "$d" commit -qm fixture
"$repo/bin/agentic-sdd" opencode evidence snapshot --workspace "$d" --scope .
```

Expected: `execution: observed`; `identity.diff` is `sha256-v1:` plus 64 hex;
`identity.base_revision` is 40 hex. Then the refusals:

```sh
"$repo/bin/agentic-sdd" opencode evidence snapshot --workspace "$d" --scope ../outside
"$repo/bin/agentic-sdd" opencode evidence snapshot --workspace "$d" --scope '*'
"$repo/bin/agentic-sdd" opencode evidence validate --receipt /nope --workspace "$d" --identity /nope --contract /nope --result /nope --evidence-root /tmp
```

Expected: each exits 2 with a bounded error and no identity/state written.

If you want the evidence steps inside a container, a locally present Ubuntu base
image with git is available on this host:

```sh
SDD_CONTAINER_IMAGE=sha256:191967da651ec0a567de173ef8a029080da8cd4992de192b65df54ef189be769 \
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work tests/vnext-container.sh
```

The image must already exist locally; the recipe never pulls. Inside it, repeat
the §4 and §6.2 commands with `/work/project`.

## 7. Host-side deterministic checks

```sh
go test ./internal/skillsync ./internal/harnessevidence ./internal/opencodeadapter ./cmd/agentic-sdd -count=1
make test
make vet
find skills-files -name SKILL.md | wc -l
find skills-files -type f | wc -l
find skills-files -type l | wc -l
git diff --check
```

Expected: Go packages `ok`; `make` targets pass; `18`, `28`, `0`; no whitespace
errors.

## 8. Static rule review (no model)

Read each file and record the exact line that supports the rule.

| # | File | Confirm |
| --- | --- | --- |
| 15 | `skills-files/agentic-sdd-implement/SKILL.md` | One batch, then `awaiting human review`; does not auto-start the next batch. |
| 16 | `skills-files/agentic-sdd-verification-review/SKILL.md` | Review-only changes nothing; verdict never replaces human approval. |
| 17 | `skills-files/agentic-sdd-grilling/SKILL.md` | At most three independent material questions per round. |
| 18 | `skills-files/agentic-sdd-questionnaire/SKILL.md` | Drafting is not permission to send/contact; answers preserve source/uncertainty/owner. |
| 19 | `skills-files/agentic-sdd-handoff/SKILL.md` | A handoff never grants a new batch or live permission. |
| 20 | `skills-files/agentic-sdd-wait-what/SKILL.md` | Explanation only; no file/verdict/approval/task/batch change. |
| 21 | `skills-files/agentic-sdd-teach/SKILL.md` | Persistent teaching files are opt-in on a separately chosen workspace. |
| 22 | `skills-files/agentic-sdd-writing-for-agents/SKILL.md` | Conditional pointers, one owner per rule, no repository-wide rewrite. |

## 9. Deferred — model-dependent scenarios

Record each as **not run** in this cycle; do not paste them into the model-free
container.

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

Record deviations immediately; do not rerun a failed step blindly. Reconcile
notes into the feature continuation only when authorized.

## 11. Stop and rollback

- **Stop** on: unexpected write, permission prompt, network access, missing
  credential isolation, unexpected cost, or a result contradicting the expected
  column.
- **Skills:** the container home is ephemeral; discard it. On a real setup, use
  `agentic-sdd backups` and `agentic-sdd restore <ID> --apply`. Restore backs up
  the current tree first, so it is safe to retry a restore, but it does not
  overwrite unrelated skills.
- **Config:** preview only here; use the digest-checked
  [adoption/rollback procedure](../integrations/opencode/adoption.md) only under
  separate authorization.
- **Evidence:** keep sanitized external evidence only under the documented
  retention policy; never copy auth stores or raw logs into Git.

## 12. What “pass” means

A pass means the recorded step produced the expected observation in this
container/setup. It does **not** mean the feature is ready, supported, safe for
private data or beneficial, and it does not close carried 004 findings or the
native-admission block. Final status remains **implementation delivered; human
usage validation not run** until actual human evidence is recorded.
