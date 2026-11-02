# Feature 005 — human test guide

**Status: ready for you to run; not executed.** About 90 minutes. The point is
to check whether the workflow behaves sensibly for a human — not to judge model
quality, speed or cost.

Use a **safe, disposable project** (not production data). Do not change your
normal OpenCode configuration in this cycle; the config section is preview-only.
The default cycle needs no credentials or payments. The optional online
OpenCode mode uses your own login and one disposable prompt; it is separate from
the delivered adapter, whose native admission stays blocked.

Record each task in the [run sheet](vnext-human-cycle-run-sheet.md). For the
design rationale, see the [final usage plan](vnext-human-test-plan.md).

## What you are checking

1. A direct fix stays direct — no feature paperwork.
2. Spec/plan/tasks are understandable, and nothing is implemented before your
   approval.
3. One approved batch is implemented, reviewed and then stops.
4. Review-only requests produce findings without edits.
5. Interruption/resume preserves your work and authority.
6. Bug investigation, clarification, questionnaire, explanation and teaching
   behave as described without side effects.

## Before you start

- Pick a small, real task you can throw away.
- Use your normal assistant/chat for the workflow scenarios below. The delivered
  adapter’s native prompt admission is still blocked; do not treat an optional
  online OpenCode session as adapter evidence.
- Decide a timebox (90 minutes suggested) and stop on anything unexpected.
- Have the repository path available for the build check.

## 1. Confirm the build (2 minutes)

From the repository root:

```sh
make build
sh tests/vnext-model-free-checks.sh
```

Expected last line:

```text
Summary: 14 passed, 0 failed, 0 skipped
```

If anything fails, stop here and share the output. This check is deterministic
and model-free; it covers the installer, candidate/preflight, config preview,
admission and evidence basics so you do not have to repeat them by hand.

Prefer a fuller Linux environment? Use the larger local Ubuntu base, which also
has `git` inside so the evidence checks run there too:

```sh
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work \
  tests/vnext-container.sh --batch --full
```

Expected the same `14 passed, 0 failed, 0 skipped`. No image is pulled; the
recipe fails if it is not already present locally.

## 2. Workflow scenarios (60–75 minutes)

For each scenario: open your safe project in a chat, paste the prompt, compare
with “You should see”, and record pass/fail. Scenarios are independent; skip one
if it does not fit your project, and record `not run`.

### 1. Direct fix

> Fix this small bug in the safe project: <one or two sentences describing the
> bug and expected behavior>. Do not create feature docs.

You should see: a focused change or a focused question; **no** `specs/` folder,
no spec/plan/tasks, no feature interview.

Record: did it stay direct? Any unnecessary documents?

### 2. Spec-only and approval

> Draft a spec, plan and tasks for this small feature. Stop for my combined
> review. Do not implement anything yet.

You should see: three short, coherent documents; open questions if genuinely
needed; no code changes. Approve or reject them yourself.

Record: were they understandable? Did implementation wait for you?

### 3. One batch

> I approve. Implement batch B1 only, verify it, and stop for my review.

You should see: work limited to that batch, a short verification summary, and a
stop with a review request — not an automatic next batch.

Record: did it stop? Was the evidence understandable?

### 4. Review-only

> Review the change you just made. Do not edit anything.

You should see: findings and a recommendation only; no code, document, approval
or task-state changes.

Record: any edits? Were findings concrete?

### 5. Interruption and resume

> Start a harmless multi-step task, then pause and tell me where you are.

After it pauses:

> Resume. Re-check the current state and continue only what is authorized.

You should see: your files preserved, current state re-read, no new batch
started implicitly, and a clear next action.

Record: was anything lost or silently restarted?

### 6. Bug investigation

> This fails: <paste the failure or reproduction>. Diagnose it before changing
> code.

You should see: reproduction, a cause distinguished from the symptom, and a
proposed fix — not an unrelated rewrite or an immediate blind patch.

Record: was the diagnosis believable?

### 7. Clarification / grilling

> Help me decide between <option A> and <option B>. Ask only what you need.

You should see: at most three independent questions in a round, reuse of what
you already answered, and no implementation.

Record: did it ask too much or start work?

### 8. Questionnaire

> Draft questions for <role/person> about <missing information>. Do not send
> anything.

You should see: a short, recipient-specific draft with answer stubs; no sending,
contacting or claims about answers.

Record: was it appropriately scoped?

### 9. Explanation

> Wait, what is the current status?

You should see: a short plain-language explanation of where the work is, what
exists versus remains, and the next decision — with no state change.

Record: was it clear?

### 10. Teaching

> Teach me <one concept> using this code as the example. Do not create files.

You should see: a focused explanation/example; no persistent teaching workspace
unless you explicitly asked for one.

Record: did it stay focused?

## 3. OpenCode inside the container (optional, 5–15 minutes)

Start the interactive sandbox (not `--batch`):

```sh
cd ~/github/agentic-sdd
make build
mkdir -p /tmp/opencode/vnext-cycle-work
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work \
  tests/vnext-container.sh
```

Add `--full` to use the larger Ubuntu base (git/bash/coreutils). Inside you
should see a banner with `OpenCode: opencode v2.2.5`. Then:

```sh
opencode --version    # opencode v2.2.5
pwd                   # /work
ls /src               # repository, read-only
opencode              # opens the TUI
```

Expected: the TUI opens. It has **no network and no credentials**, so it cannot
answer a prompt or reach a provider; do not try to send one. Use it only to see
the interface and check that the binary runs. Quit with the TUI's quit key or
`Ctrl-C`, then `exit` to leave the container.

The binary is **mounted read-only from the host**, not downloaded into the
image. If the TUI reports a render-library error, you are in an old container:
exit and start a fresh one with the current script.

### SDD skills and config inside the container

On startup the interactive container installs the **18 Agentic SDD skills** from
the embedded CLI into `~/.agents/skills` (and the corresponding codex/claude/agy
directories). Verify:

```sh
find "$HOME/.agents/skills" -name SKILL.md | wc -l    # 18
ls "$HOME/.agents/skills" | head
```

OpenCode discovers these skills without any host configuration. If the assistant
answers with only the builtin `opencode` skill and no `agentic-sdd-*` skill, you
are in an old container: exit and restart with the current script.

The container deliberately **does not activate** the SDD OpenCode candidate,
because that candidate denies providers. It writes an inactive copy to
`/work/sdd-candidate.jsonc`; the examples under `/src/integrations/opencode` are
read-only. Preview them, do not apply them in this cycle.

### Online mode — exact sequence (you drive)

This is the only part that uses network and your login. Follow it in order.

**Step 1 — start the online container**

```sh
cd ~/github/agentic-sdd
make build
mkdir -p /tmp/opencode/vnext-cycle-work
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work \
  tests/vnext-container.sh --online --full
```

Expected banner:

```text
Container mode: online network, no host home or mounted credentials.
OpenCode: opencode v2.2.5 (mounted read-only)
```

If it says `offline`, you are in the wrong container. Exit and rerun the command
above.

**Step 2 — check connectivity (no model call)**

```sh
getent hosts opencode.ai
sh /src/tests/vnext-online-check.sh
```

Expected: addresses from `getent`, then:

```text
PASS dns-opencode-ai
PASS https-opencode-ai
PASS opencode-binary
SKIP hi-response: connectivity only; use --hi with SDD_HI_MODEL for one prompt

Summary: 3 passed, 0 failed, 1 skipped
```

If DNS/HTTPS fails, stop. `cat /etc/resolv.conf` must contain `1.1.1.1` and
`8.8.8.8`; if it contains only `127.0.0.53`, you are in a stale container.

**Step 3 — log in**

```sh
opencode auth login
```

Open the printed device-code URL in your **host** browser and complete the code.
Optional confirmation (no secrets printed):

```sh
opencode auth list
```

**Step 4 — one “hi” round trip (one model call)**

```sh
SDD_HI_MODEL=<exact provider/model id> \
  sh /src/tests/vnext-online-check.sh --hi
```

Example from the usage catalog: `opencode/deepseek-v4.1-flash-free` (reported
free; access/privacy are not verified here). Expected: `PASS hi-response` plus a
short reply (truncated to 300 characters). Use only disposable, non-sensitive
content. Without `SDD_HI_MODEL` the check stays skipped.

**Step 5 — TUI check**

```sh
opencode
```

Type `hi` and confirm a reply. Quit with the TUI’s quit key or `Ctrl-C`, then
`exit` to leave the container.

**Step 6 — record**

Record login result, exact model id, whether `hi` replied, and any error. The
login lives only in the container’s ephemeral `/tmp` and disappears on exit.

### SDD router inside the container (optional, online)

After login, confirm OpenCode uses the packaged SDD skills rather than a generic
answer. In the TUI, paste:

> Use the agentic-sdd-router skill. Draft a spec, plan and tasks for a minimal
> React + Go + SQLite post app: a textbox with Submit and a feed showing posts
> from the database. Stop for my approval; do not implement anything yet.

You should see: the router/feature-drafting behavior (three documents), no
implementation, and a wait for your combined approval. It may ask a small
number of clarifying questions.

If you instead get a generic one-off plan or a builtin `opencode` skill answer,
check the skill count above and restart the container. Record the outcome and
the model used.

**What changes and what does not**

- outbound network is enabled; credentials stay in ephemeral `/tmp`;
- no host home, credential store, SSH keys or Docker socket is mounted;
- root and `/src` stay read-only; `/work` is the only writable project mount;
- use only a disposable project and non-sensitive data. “Free” does not prove
  privacy/ZDR or tool capability;
- this is **outside** the model-free QA batch and is not evidence for native
  admission, billing or support.

### Common errors and the exact fix

| You see | Meaning | Do |
| --- | --- | --- |
| `Transport error (POST https://opencode.ai/console/auth/device/code)` | Offline/stale container or DNS failure | Check the banner says `online network`; rerun Step 1, then Step 2. |
| `EAI_AGAIN opencode.ai` or `getaddrinfo` | DNS cannot resolve | `cat /etc/resolv.conf`; if no `1.1.1.1`, restart with the current script and `--online`. |
| `Failed to initialize OpenTUI render library` | Old container without `exec` on `/tmp` | Exit and start a fresh container with the current script. |
| `cannot open /src/tests/...` | You are on the host, not inside | Inside the container use `/src/...`; on the host use `tests/...`. |
| `No rule to make target 'build'` | Wrong directory | `cd ~/github/agentic-sdd` first. |
| Batch shows old counts (e.g. 40 skills, 8 failures) | An older `agentic-sdd` from `PATH` | `make build`; the scripts prefer the checkout’s `bin/agentic-sdd`. |

## 4. Optional: inactive OpenCode configuration (10 minutes)

Only if you want to see the OpenCode path. This is preview-only; it writes
nothing:

```sh
mkdir -p /tmp/opencode/human-preview
bin/agentic-sdd opencode candidate --feature demo --build-path src > /tmp/opencode/candidate.jsonc
bin/agentic-sdd opencode preflight --config /tmp/opencode/candidate.jsonc
bin/agentic-sdd opencode config --workspace /tmp/opencode/human-preview --candidate /tmp/opencode/candidate.jsonc
```

You should see: `online_ready: false`, four blockers, `target_runtime: 2.2.5`,
and `original: "absent"` with `noop: false`. Confirm no config appeared:

```sh
test ! -e /tmp/opencode/human-preview/.opencode && echo no-write
```

Do **not** run `config apply`; adoption is out of scope for this cycle.

## 5. Optional: evidence (5 minutes)

The build check already exercised evidence on the host when git is available. To
see it yourself:

```sh
d=$(mktemp -d /tmp/opencode/evidence.XXXXXX)
git -C "$d" init -q
git -C "$d" config user.email qa@example.invalid
git -C "$d" config user.name QA
printf 'hi\n' > "$d/code.txt"
git -C "$d" add .
git -C "$d" commit -qm fixture
bin/agentic-sdd opencode evidence snapshot --workspace "$d" --scope .
```

You should see `execution: observed`, a `sha256-v1:` diff and a 40-character
base revision. A reuse check must fail after you change `code.txt`; do not treat
a receipt as approval.

## What counts as a pass

A pass means the behavior matched the “You should see” list for the setup you
actually used. It does **not** mean the feature is supported, safe for private
data or beneficial; those need separate evidence and decisions.

Record `fail` or `blocked` honestly. Do not rerun a failing scenario until you
understand why. Keep your workspace and any notes; no credentials or raw logs
in Git.

## If something looks wrong

Stop the scenario, record what you saw, and share the smallest reproduction and
the relevant output. Do not try to “make it pass” by changing scope, disabling
checks or enabling a model. Further fixes/retests are a separate decision.

## Technical appendix

The one-command container batch (isolated, no network/credentials) is:

```sh
mkdir -p /tmp/opencode/vnext-cycle-work
SDD_CONTAINER_WORKSPACE=/tmp/opencode/vnext-cycle-work \
  tests/vnext-container.sh --batch
```

Expected: `Summary: 12 passed, 0 failed, 2 skipped` (the two skips are git-based
evidence). With `--full` (the larger Ubuntu base) the same batch reports
**14 passed, 0 failed, 0 skipped** because `git` is present and the evidence
checks run inside the container. For the interactive shell with `opencode` and
the OpenCode TUI, run the same command **without** `--batch`; add `--full` for
the larger base. See §3. Inside the interactive container only, the same checks
run as `sh /src/tests/vnext-model-free-checks.sh`.

Not covered here: real native model tools/permissions, native pending-human
handling, native cancellation/compaction/recovery, real-home adoption, macOS,
billing/provenance and useful benefit. Those remain `not run` or require a
separate authorization.
