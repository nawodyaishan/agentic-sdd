# Fleet profiles worked guide — Feature 014 (draft, B5)

**Status of this guide:** written with Feature 014 B5 (2026-11-11). It documents
the approved design and the commands each batch wires. `fleet doctor` is the
opt-in diagnostics landed by B5; `fleet status|plan|apply|restore` follow the
frozen grammar of `plan.md` §2.3 and are delivered by B3/B4 — treat their
sections here as the contract to verify at batch review, not as tested release
behavior. Nothing in this feature runs a model, touches a provider over the
network, or writes your real `~/.config/opencode` without an explicit apply.

Target reader: an OpenCode V2 user who keeps Agentic-SDD installed and wants
personal model bindings for the approved agent roles, versioned in their own
private Git repository.

## 1. Set up the private Git configuration (once, per user)

1. Create a **private** repository (e.g. `agentic-sdd-fleet`) on your Git host.
   Your personal model choices are configuration metadata — keep the repo
   private if you don't want your provider lanes public.
2. Add one file, `fleet.json`:

   ```json
   {
     "schemaVersion": 1,
     "fleet": "personal",
     "preset": "opencode-go",
     "instances": {
       "sdd-implement-1": {
         "source": "preset:sdd-implement-1",
         "model": "opencode-go/deepseek-v4.1-flash#high",
         "enabled": true
       },
       "sdd-implement-9r-1": {
         "source": "preset:sdd-implement-1",
         "model": "9router/ag/gemini-3.8-flash-high",
         "enabled": true
       }
     }
   }
   ```

   Rules that the strict parser enforces (spec §4, R8–R9):
   * Standard JSON only — no comments, no YAML.
   * `source` is one hop: `agent:<name>` (from `opencode-pack/agents/`) or
     `preset:<name>` (from `opencode-pack/presets/free-first/`). Instances
     never reference instances.
   * `model` is `<provider>/<model-id>[#variant]`.
   * Unknown keys are rejected, including anything policy-shaped —
     `permissions`, `shell`, `execute`, `subagent`, `fallback*`,
     `automaticDelegation`, spending flags. You choose **models and
     enablement only**; the repository roles keep the prompts and permissions.
   * Duplicate JSON keys, oversize files (>1 MiB), traversal in names and
     invalid UTF-8 fail closed.
3. Never commit credentials. The CLI reads no keychain and stores no tokens;
   account state stays where OpenCode keeps it.
4. You are responsible for commits, branches and remote ACLs. The CLI never
   clones, pushes or rewrites your Git state (spec 3A, FR-03).

Install or refresh the canonical roles as usual — unchanged 009 behavior:

```sh
agentic-sdd apply --with-opencode        # exactly as before; fleet is opt-in
```

`fleet.json` is only ever consulted by `agentic-sdd fleet …` subcommands.
Non-fleet commands never read it (NFR-03/AC-06).

## 2. Change one agent's model

Goal: move `sdd-implement-1` from its current model to another lane, without
touching its prompt or permissions.

1. Edit `fleet.json` — change only the one `model` string.
2. Preview (read-only, offline):

   ```sh
   agentic-sdd fleet plan --config ~/private/agentic-sdd-fleet/fleet.json
   ```

   The plan lists every create/update/removal/no-op, the exact target files
   under `~/.config/opencode/agents/`, and any permission-relevant difference.
   The renderer starts from the canonical pack bytes and replaces only the
   `model:` scalar; body bytes and the permission set must be unchanged or the
   plan fails (FR-04/FR-11, AC-03/AC-17).
3. Apply with a fresh, explicit action:

   ```sh
   agentic-sdd fleet plan --config ... --output /tmp/fleet.plan.json
   agentic-sdd fleet apply --plan /tmp/fleet.plan.json
   ```

   Apply revalidates source, config and installed hashes against the plan,
   refuses stale plans, takes an operation lock, backs up replaced managed
   files, and rolls back on partial failure (FR-07/FR-08, AC-10/AC-12).
   A plan ID is a consistency token, not an authorization.
4. Only that agent's file changes. Its role prompt, `permissions:` block and
   your global/project `AGENTS.md` are untouched (AC-11). Disabling an
   instance is `enabled: false` → a previewed managed removal with a
   recoverable backup (R7, AC-16); an unrelated same-name file is never
   deleted (conflict instead).

## 3. Diagnose drift and restore

If someone edits an installed managed file by hand:

* `fleet plan` again: current ≠ last-applied with a differing desired state is
  a **conflict**; the CLI stops for a human decision and never merges text
  speculatively (FR-10, AC-13).
* Manual drift where the new desired equals last-applied is reported as
  drift without overwrite.

To undo a fleet apply:

```sh
agentic-sdd fleet restore --backup <fleet-backup-id>
```

Restore validates the current hashes first: if the managed files changed
after that apply, it refuses without an explicit review, and unrelated user
files are preserved (AC-14). `git revert` in your fleet repo only changes
your JSON source; installed files still require preview → apply.

## 4. Reproduce on a second machine

```sh
git clone git@github.com:you/agentic-sdd-fleet.git ~/private/agentic-sdd-fleet
agentic-sdd apply --with-opencode                       # canonical pack
agentic-sdd fleet plan --config ~/private/agentic-sdd-fleet/fleet.json
agentic-sdd fleet apply --plan ...
```

Same canonical pack source + same `fleet.json` + same renderer version gives
**byte-identical** managed files (NFR-01, AC-03). Differences that can still
matter are reported by the plan: a different OpenCode version or a locally
differing `~/.config/opencode/opencode.json` can change what OpenCode itself
treats as effective — the CLI renders correctly but qualifies runtime claims
(spec §3.1, FR-14). Format-3 install histories have no fleet provenance;
those targets are reported as `unmanaged/legacy provenance unknown` and
adopted only through an explicit `--adopt` preview + apply (R4, AC-15).

## 5. Providers, models and entitlement — read this before picking a lane

Provider prefixes are **routes**, not subscriptions (spec §2.2/§4.2):

| Route | Meaning | Not implied |
|---|---|---|
| `opencode-go/…` | OpenCode Go subscription lane | per-model entitlement |
| `opencode/…` | OpenCode Console catalog (free or metered IDs) | price or availability |
| `9router/…` | one configured local proxy route; IDs may name Gemini/Claude/Argon families | any OpenCode-hosted lane |
| `openai/…` | a provider namespace that may be ChatGPT OAuth **or** API-key | ChatGPT Plus/Pro usage |

A model string is never proof of authentication or entitlement. Use the
opt-in diagnostics to see what local evidence says:

```sh
agentic-sdd fleet doctor --config ~/private/agentic-sdd-fleet/fleet.json
```

Default mode is fully offline: every route is labeled `not checked (offline)`
and nothing is consulted. Point the doctor at your local OpenCode executable
to get evidence labels:

```sh
agentic-sdd fleet doctor --config ... --opencode-bin ~/.opencode/bin/opencode
```

Labels (R6): `configured` (provider present in local evidence) ·
`catalog-visible` (model appears in a local catalog listing) · `authenticated`
(explicit local authentication marker — never inferred from the route or
catalog presence) · `unknown` (inspected, undetermined) · `unavailable` (the
tool you asked it to inspect failed). The report names exactly which local
tools it consulted. `fleet doctor` performs no network request, runs no model,
and writes nothing; a `catalog-visible` or even `authenticated` label still
does not authorize paid use — selecting a lane that costs money remains your
account's decision, outside this CLI (FR-12, NFR-09).

## 6. Limitations of this draft state

* Only `fleet doctor` is wired and tested by B5. `status/plan/apply/restore`
  behavior above is the approved contract; each lands with its own batch and
  human review (B3/B4), so do not treat their output here as verified.
* No release is authorized from this guide; no real-home apply was performed
  by any B5 evidence — all tests use disposable temp homes.
* Runtime effectiveness was **not** proven against a pinned OpenCode binary
  in this batch; runtime claims stay `not checked (offline)` until B5-level
  native validation actually exercises a version (FR-14, NFR-11).
* IR-06-01 (profile path isolation is an OS concern) is unchanged: fleet
  rendering is not a sandbox and claims no isolation guarantee.
* 18-name machine-fleet reproduction (AC-02) is represented by the 18-name
  scale fixture and the B1/B2 golden mapping. `sdd-spec-9r` and `sdd-plan-9r`
  use reviewed read-only, non-writing/non-shell adapter sources under
  `opencode-pack/adapters/`; `vision` maps to the canonical
  `sdd-visual-inspector` role (single mapping, no divergent `vision.md`).
  None of these adapters is installed by `--with-opencode`, which stays 14
  profiles.
* No TUI, no marketplace, no remote fetching, no daemon, no automatic Git —
  deliberate non-goals of Feature 014 (spec §9).
