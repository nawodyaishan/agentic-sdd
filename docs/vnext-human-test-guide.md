# Feature 005 — human test guide (accepted container)

**Status: accepted environment recorded in spec §Approval; the cycle is NOT
executed.** About 90 minutes, possibly across sittings. The point is to check
whether the workflow behaves sensibly for a human — not to judge model quality,
speed or cost.

**Accepted environment:** the pinned local image `agentic-sdd-qa:005`, not the
normal setup. Identity recorded in spec §Approval:

| Item | Identity |
| --- | --- |
| Image | `sha256:35ed49cdc081db1f233d238568e7fd2455ede458377703b9eb05d0a30103258b` |
| Base | `codercom/enterprise-base@sha256:191967da651ec0a567de173ef8a029080da8cd4992de192b65df54ef189be769` |
| OpenCode | `2.2.5`, sha256 `8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050` |
| agentic-sdd binary | sha256 `f3c4b51742274df9e04a235eddac0fd31d388521339d9f8c1444c877e2d48974` |
| Source commit | `ba90f5737c125e4df9a4935adeca01301173f0ab` |
| Skills | 18 packaged SDD skills preinstalled |

The cycle is **user-operated**: you perform any online login and model call. The
assistant does not run a model call, copy credentials or change a real home.
Results apply only to this image; normal-setup behavior, native admission,
macOS, billing/provenance and benefit remain unproven.

Record each task in the [run sheet](vnext-human-cycle-run-sheet.md). Design
rationale: [final usage plan](vnext-human-test-plan.md).

## What you are checking

1. A direct fix stays direct — no feature paperwork.
2. Spec/plan/tasks are understandable, and nothing is implemented before your
   approval.
3. One approved batch is implemented, reviewed and then stops.
4. Review-only requests produce findings without edits.
5. Interruption/resume preserves your work and authority.
6. Bug investigation, clarification, questionnaire, explanation and teaching
   behave as described without side effects.

## 1. Build the image once (2 minutes)

```sh
cd ~/github/agentic-sdd
make build
tests/build-qa-image.sh
```

Expected final line:

```text
image_id=sha256:35ed49cdc081db1f233d238568e7fd2455ede458377703b9eb05d0a30103258b
```

The build uses the local base and binaries with `--network=none`; nothing is
pulled. If the image id differs from the recorded one, stop and report it — the
accepted environment no longer matches §Approval.

## 2. Verify the environment (offline, 2 minutes)

```sh
mkdir -p /tmp/opencode/qa-work
SDD_QA_WORKSPACE=/tmp/opencode/qa-work tests/run-qa-image.sh --check
```

Expected: `Summary: 14 passed, 0 failed, 0 skipped`. This is a development check
inside the image, not your human acceptance result.

## 3. Start the accepted environment

**Online (for the workflow scenarios and a model):**

```sh
SDD_QA_WORKSPACE=/tmp/opencode/qa-work tests/run-qa-image.sh --online
```

Inside, verify before logging in:

```sh
opencode --version          # opencode v2.2.5
agentic-sdd version
git --version               # git 2.55.x
find "$HOME/.agents/skills" -name SKILL.md | wc -l   # 18
getent hosts opencode.ai
sh /src/tests/vnext-online-check.sh
```

Expected connectivity: `PASS dns-opencode-ai`, `PASS https-opencode-ai`, `PASS
opencode-binary`, `SKIP hi-response`. No model call.

Log in and start the assistant:

```sh
opencode auth login         # open the device-code URL in your host browser
opencode auth list          # optional; no secrets printed
opencode
```

The banner must say `Network: online`. The binary is installed **inside the
image**, not mounted from the host; credentials live in ephemeral `/tmp` and are
lost when the container exits. There is no persisted home, so log in again for a
new sitting.

**Offline (installer/config/evidence only):**

```sh
SDD_QA_WORKSPACE=/tmp/opencode/qa-work tests/run-qa-image.sh
```

## 4. Workflow scenarios (60–75 minutes)

Run these inside `opencode` in the container, using only the disposable
`~/work` project and non-sensitive content. For each: paste the prompt, compare
with “You should see”, record pass/fail. Skip any that do not fit; record
`not run`.

### 1. Direct fix

> Fix this small bug in the safe project: <one or two sentences describing the
> bug and expected behavior>. Do not create feature docs.

You should see: a focused change or a focused question; **no** `specs/` folder,
no spec/plan/tasks, no feature interview.

### 2. Spec-only and approval

> Draft a spec, plan and tasks for this small feature. Stop for my combined
> review. Do not implement anything yet.

You should see: three short, coherent documents; open questions only if genuinely
needed; no code changes; a wait for your approval.

### 3. One batch

> I approve. Implement batch B1 only, verify it, and stop for my review.

You should see: work limited to that batch, a short verification summary, and a
stop with a review request — not an automatic next batch.

### 4. Review-only

> Review the change you just made. Do not edit anything.

You should see: findings and a recommendation only; no code, document, approval
or task-state changes.

### 5. Interruption and resume

> Start a harmless multi-step task, then pause and tell me where you are.

After it pauses:

> Resume. Re-check the current state and continue only what is authorized.

You should see: files preserved, current state re-read, no new batch started
implicitly, and a clear next action.

### 6. Bug investigation

> This fails: <paste the failure or reproduction>. Diagnose it before changing
> code.

You should see: reproduction, a cause distinguished from the symptom, and a
proposed fix — not an unrelated rewrite or a blind patch.

### 7. Clarification / grilling

> Help me decide between <option A> and <option B>. Ask only what you need.

You should see: at most three independent questions per round, reuse of settled
answers, and no implementation.

### 8. Questionnaire

> Draft questions for <role/person> about <missing information>. Do not send
> anything.

You should see: a short, recipient-specific draft with answer stubs; no sending,
contacting or claims about answers.

### 9. Explanation

> Wait, what is the current status?

You should see: a short plain-language explanation of where the work is, what
exists versus remains, and the next decision — with no state change.

### 10. Teaching

> Teach me <one concept> using this code as the example. Do not create files.

You should see: a focused explanation/example; no persistent teaching workspace
unless you explicitly asked for one.

### 11. SDD router inside OpenCode (the original failure case)

After login, this is the check that the packaged skills are actually loaded:

> Use the agentic-sdd-router skill. Draft a spec, plan and tasks for a minimal
> React + Go + SQLite post app: a textbox with Submit and a feed showing posts
> from the database. Stop for my approval; do not implement anything yet.

You should see: SDD router/feature-drafting behavior (three documents), no
implementation, and a wait for your approval. If you see only the builtin
`opencode` skill or one generic plan, check the skill count (18) and restart the
current image.

## 5. Installer, config and evidence inside the image

**Skills** are already installed at build time (18 under `~/.agents/skills`, plus
the other client dirs). To exercise installer behavior manually:

```sh
agentic-sdd preview --home "$HOME"    # no writes
agentic-sdd apply --home "$HOME"      # 72 copies, backup under ~/.agentic-sdd
agentic-sdd backups --home "$HOME"
```

Restore semantics: the newest backup is the pre-apply state, so restoring it
returns the edited copy, not canonical. Re-apply to canonicalize.

**Config is preview-only.** An inactive candidate is written to
`~/work/sdd-candidate.jsonc`; examples are read-only under
`/src/integrations/opencode`. Do **not** apply the candidate — it denies
providers by design.

```sh
agentic-sdd opencode preflight --config "$HOME/work/sdd-candidate.jsonc"
agentic-sdd opencode config --workspace "$HOME/work" --candidate "$HOME/work/sdd-candidate.jsonc"
```

Expected: `online_ready: false`, four blockers, `original: "absent"`, and no
`.opencode` directory.

**Evidence** (git is present in the image):

```sh
d=$(mktemp -d /tmp/evidence.XXXXXX)
git -C "$d" init -q
git -C "$d" config user.email qa@example.invalid
git -C "$d" config user.name QA
printf 'hi\n' > "$d/code.txt"
git -C "$d" add .
git -C "$d" commit -qm fixture
agentic-sdd opencode evidence snapshot --workspace "$d" --scope .
```

Expected: `execution: observed`, a `sha256-v1:` diff and a 40-character base
revision. A receipt is not approval.

## 6. Record, stop, and report

Use the [run sheet](vnext-human-cycle-run-sheet.md). Record the image id, exact
model/provider used, each scenario result, and any error. Allowed results:
**pass / fail / blocked / not run**.

### Common errors

| You see | Meaning | Do |
| --- | --- | --- |
| `EAI_AGAIN` or `Transport error (POST …/device/code)` | Container is offline or DNS failed | Check the banner says `Network: online`; rerun `getent hosts opencode.ai` and `vnext-online-check.sh`. |
| `Failed to initialize OpenTUI render library` | Old image/container | Rebuild with `tests/build-qa-image.sh`; the image uses `exec` on `/tmp`. |
| `cannot open /src/tests/...` | Host shell, not container | Enter the image; `/src` exists only there. |
| No SDD skills / builtin `opencode` only | Old image or missing install | `find "$HOME/.agents/skills" -name SKILL.md | wc -l` must be 18. |
| Batch passes but features feel generic | You are looking at development checks, not the model cycle | Use Step 3 online + login; the accepted cycle is the OpenCode session. |

### What “pass” means

A pass means the observed behavior matched “You should see” **in this image**
with the recorded model. It does not certify the normal setup, native admission,
macOS, billing/provenance, adoption or benefit. Final status remains
**implementation delivered; human usage validation not run** until your results
are recorded and accepted.

## Technical appendix

- Build: `tests/build-qa-image.sh` (local base/binaries, `--network=none`).
- Run: `tests/run-qa-image.sh` (`--online` for login/model; `--check` for the
  model-free batch).
- Old ad-hoc `tests/vnext-container.sh` remains a development aid; it is **not**
  the accepted environment.
- Not covered: real native adapter admission, macOS, adoption, billing,
  provenance and useful benefit.
