---
title: Practice specialist selection and synthetic routing
description: Walk through opt-in pack preview, bounded brief drafting and offline fleet.json routing using inert fixtures.
---

This tutorial walks through opt-in delegation, choosing a specialist profile, inspecting a synthetic child return, and testing a model-routing configuration offline with `asdx-core fleet`.

Before starting, review the core feature workflow (`tutorials/first-feature` in the sitemap) and the boundaries in [the orchestration how-to](../how-to/orchestration.md). Every step below uses read-only previews or the inert fixtures in `website/tests/examples/orchestration/`: **no real home directory is modified, no credentials are read, and no live provider or model calls are made**.

## Step 1: Preview the opt-in pack in an isolated home

Check what `--with-opencode` and `--opencode-free-first` select without touching your real configuration by passing `--home` pointing at an isolated temporary directory (`cmd/agentic-sdd/main.go`, `internal/skillsync/sync.go`):

```sh
# 1. Core-only baseline (18 core skills, no agent profiles)
asdx-core preview --home /tmp/example-opencode-home

# 2. Add the optional OpenCode pack (18 core skills + agentic-sdd-orchestrate + 14 profiles)
asdx-core preview --with-opencode --home /tmp/example-opencode-home

# 3. Include the 4 Free-first writer/test templates
asdx-core preview --with-opencode --opencode-free-first --home /tmp/example-opencode-home
```

**Expected observation (Illustrative — not a captured live run):**

```text
# With --with-opencode:
install agents/agentic-sdd-orchestrate
install codex/agentic-sdd-orchestrate
install claude/agentic-sdd-orchestrate
install agy/agentic-sdd-orchestrate
install opencode-agents/sdd-orchestrator.md
... (14 profile files total under opencode-agents/) ...
```

Notice that `preview` writes nothing and creates no backup. Notice also that the two fleet-only adapter profiles (`sdd-spec-9r`, `sdd-plan-9r` in `opencode-pack/adapters/`) are absent from the `--with-opencode` preview (`internal/skillsync/opencode_adapters_test.go`). For real-home `apply` and `restore` procedures, see [the orchestration how-to](../how-to/orchestration.md) and [the CLI reference](../reference/cli.md).

## Step 2: Select a profile and inspect a bounded brief

When a task genuinely needs delegation beyond `D0` direct work, choose one profile matching the delegation class (`D1` read-only, `D2` authorized writer, or `D3` advisor/research/vision) from [the agent profile catalog](../reference/profiles.md) rather than duplicating its full model table here:

- **Bounded read-only discovery (`D1`):** `sdd-lightweight` (`steps: 10`, `edit` and `shell` denied) or `sdd-debug-diagnose` (root-cause diagnosis, `edit` and `shell` denied).
- **Authorized bounded writer (`D2`):** `sdd-docs-worker` (documentation edits only; `shell` denied), `sdd-openai-implement` (optional OpenAI writer; requires explicit spend/provider access), or one of the four `--opencode-free-first` templates (`sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, `sdd-test-implement-2`).
- **Advisor, review, or research (`D3`):** `sdd-architecture-advisor`, `sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-research-librarian`, `sdd-openai-architect`, or `sdd-visual-inspector`.

Inspect the inert brief fixture at `website/tests/examples/orchestration/bounded-brief.md`:

```yaml
task_id: T02
authorized_scope: >
  Feature 000 (synthetic example), Batch B1 only; combined approval recorded in
  example-app/specs/000-example/spec.md#approval. Not authorized: other batches,
  git commit, git push, publishing, or live environment changes.
allowed_paths:
  - example-app/internal/greeting/greeting.go
  - example-app/internal/greeting/greeting_test.go
expected_verification:
  - "go test ./internal/greeting/... (quote exact command and exit/output)"
stop_conditions:
  - "Pre-existing checks fail outside allowed_paths"
  - "Required edit falls outside allowed_paths"
  - "Pinned model is unavailable and switching would change privacy or cost"
  - "Interrupted write leaves unclear working-tree state"
```

Why this shape works (`opencode-pack/skills/agentic-sdd-orchestrate/references/dispatch-contract.md`):

- `authorized_scope` names the one batch and explicitly rules out commits, pushes, and next batches.
- `allowed_paths` prevents silent edits to callers outside the task.
- `stop_conditions` forces the child to stop instead of guessing or switching to an unauthorized model.

## Step 3: Reconcile a synthetic child return

Inspect the inert return fixture at `website/tests/examples/orchestration/child-return.yaml`:

```yaml
# Synthetic Child Return Fixture (Illustrative — not a captured live run)
task_id: T02
status: partial
scope_observed: >
  Read example-app/internal/greeting/greeting.go and edited
  example-app/internal/greeting/greeting_test.go only; left greeting.go
  unmodified after test failure showed an existing caller contract question.
changed_paths:
  - example-app/internal/greeting/greeting_test.go
findings:
  - "example-app/internal/greeting/greeting.go:18 currently returns an empty string on empty input; callers in example-app/cmd/cli/main.go:42 are outside allowed_paths."
source_references:
  - example-app/specs/000-example/spec.md#approval
  - example-app/internal/greeting/greeting.go:15-24
checks_run:
  - "go test ./internal/greeting/... -> FAIL (exit 1): TestFormatGreetingEmpty expected error, got nil"
unverified_claims:
  - "Whether example-app/cmd/cli/main.go handles a non-nil error from FormatGreeting was not checked because it is outside allowed_paths."
actual_model: opencode/deepseek-v4.1-flash-free
questions_for_parent:
  - "Should Batch B1 expand allowed_paths to inspect or update example-app/cmd/cli/main.go, or keep empty-name behavior unchanged?"
```

In a real session, the parent agent would verify `changed_paths` with `git status --short` and `git diff`, confirm the cited lines in `greeting.go`, and present `questions_for_parent` plus the observed `actual_model` to the human instead of bypassing `allowed_paths` (`opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`).

## Step 4: Exercise synthetic model-routing configuration offline

The pack's `opencode-pack/routing.md` documents manual fallback policy, while `asdx-core fleet` (`internal/fleet/`) provides deterministic, model-only overrides over canonical pack profiles. Look at the synthetic valid config in `website/tests/examples/orchestration/fleet-routing.valid.json`:

```json
{
  "schemaVersion": 1,
  "fleet": "example-orchestration",
  "preset": "free-first",
  "instances": {
    "orchestrator": {
      "source": "agent:sdd-orchestrator"
    },
    "docs-worker": {
      "source": "agent:sdd-docs-worker",
      "model": "opencode/glm-5.3-flash-free"
    },
    "spec-adapter": {
      "source": "agent:sdd-spec-9r",
      "model": "9router/ag/gemini-3.8-flash-high",
      "enabled": true
    },
    "disabled-writer": {
      "source": "preset:sdd-implement-1",
      "enabled": false
    }
  }
}
```

Four parser and renderer contracts illustrated here (`internal/fleet/fleet.go`, `internal/fleet/resolve.go`, `internal/fleet/render.go`):

1. **One-hop source resolution:** `source` accepts only `agent:<name>` (walking `opencode-pack/agents/` and `opencode-pack/adapters/`) or `preset:<name>` (walking `opencode-pack/presets/free-first/`). Notice `spec-adapter` resolves `agent:sdd-spec-9r` from `adapters/` even though `--with-opencode` excludes it.
2. **Model-only rendering:** `internal/fleet/render.go` (`VerifyModelOnly`) rewrites only the top-level `model:` scalar and fails closed if any permission rule or prompt body changes. Omitting `model` on `orchestrator` keeps `sdd-orchestrator` inheriting the session model.
3. **Model-pin grammar:** `model` must match `<provider>/<model>[#variant]`.
4. **Explicit disablement:** omitting `enabled` defaults to `true`; `enabled: false` plans removal if that instance was previously managed.

Now compare `website/tests/examples/orchestration/fleet-routing.invalid.json`, which triggers strict schema-v1 validation failures (`internal/fleet/parse.go`, `internal/fleet/fleet.go`):

```json
{
  "schemaVersion": 1,
  "fleet": "example-invalid",
  "instances": {
    "bad-model": {
      "source": "agent:sdd-docs-worker",
      "model": "deepseek-v4.1-flash-free"
    },
    "bad-source": {
      "source": "agent:../escape"
    }
  }
}
```

- `bad-model` fails `validateModel` because `deepseek-v4.1-flash-free` omits the required `<provider>/` prefix.
- `bad-source` fails `parseSourceRef` because path separators and `..` traversal are rejected.

You can run read-only offline checks against these fixtures with an isolated `--home` (no provider calls, no writes to destinations):

```sh
# Inspect valid fixture against an empty temporary home (read-only)
asdx-core fleet status \
  --config website/tests/examples/orchestration/fleet-routing.valid.json \
  --home /tmp/example-opencode-home

# Preview plan without --output or apply (read-only)
asdx-core fleet plan \
  --config website/tests/examples/orchestration/fleet-routing.valid.json \
  --home /tmp/example-opencode-home

# Run offline route diagnostic (never calls a network or provider endpoint)
asdx-core fleet doctor \
  --config website/tests/examples/orchestration/fleet-routing.valid.json

# Verify invalid fixture fails closed with exit code 2
asdx-core fleet status \
  --config website/tests/examples/orchestration/fleet-routing.invalid.json \
  --home /tmp/example-opencode-home
```

**Expected observations (Illustrative — not a captured live run):**

- `fleet status` and `fleet plan` on `fleet-routing.valid.json` succeed read-only (`exit 0`) against the empty scratch home; no agent files are written unless you explicitly save a plan (`--output`) and run `fleet apply --plan ...`.
- `fleet doctor` without `--opencode-bin` reports each pinned route as `not checked (offline)` (`internal/fleet/doctor.go`) and exits `0`. Even with a local `--opencode-bin`, `fleet doctor` only reads local evidence (`configured`, `catalog-visible`, `authenticated`, `unknown`, `unavailable`, `not checked (offline)`) and never confirms live quota, billing, or runtime availability.
- `fleet status` on `fleet-routing.invalid.json` exits `2` (`ErrInvalidConfig`) without leaking the config body.

## Next steps

- Review full profile permissions and model-pin rules in [the agent profile catalog](../reference/profiles.md).
- Review full `fleet.json`, lock, and `fleet-manifest.json` rules in [the fleet reference](../reference/fleet.md).
- Review manual fallback and interrupted-write recovery in [the orchestration how-to](../how-to/orchestration.md).
- Related sitemap guides (integration-pending cross-links): `tutorials/first-feature`, `how-to/fleet`, `how-to/updates-and-recovery`
