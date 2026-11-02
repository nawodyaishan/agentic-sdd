# Inactive configuration helper — 005/B2

The `agentic-sdd opencode` group prepares and checks **file inputs only**. It
does not invoke `opencode`, connect to a native/shared service, probe a model,
read a credential store or certify runtime enforcement. Source declarations in
its output are labeled `target_runtime` and `candidate_*`, not actual native
observations. Support remains [uncertified](supported-version.md).

The intended candidate is exact OpenCode 2.2.5 with the selected synthetic-only
Nemotron Ultra `:free` planning reference. All providers remain denied. The
candidate adds current-session helper commands and uses supported typed skill,
plugin and MCP denials; it is a new inactive proposal, not a re-certification of
004's older template. No alternative/fallback or online operation is enabled.

## File-only operations

From a checkout, `go run ./cmd/agentic-sdd opencode --help` prints the same help
as a newly built binary. Existing installed release binaries may lack this group.
The examples below describe user operations; they were not run on the real home.

```sh
agentic-sdd opencode candidate --feature 005-vnext-completion --build-path skills-files
agentic-sdd opencode preflight --config /absolute/private-staging/candidate.jsonc
agentic-sdd opencode config preview --workspace /absolute/project --candidate /absolute/private-staging/candidate.jsonc
```

`candidate` emits inactive JSON to stdout, not to an automatically discovered
path. Review it and, when desired, save it outside the project and active skill/
configuration discovery. Do not redirect it into a live config as an unreviewed
shortcut. Empty `--feature` omits feature-document edit grants for direct fixes.
`--build-path` is an explicit narrow literal project-relative directory.

`preflight` validates the candidate's fixed inactive model/policies/profiles and
projects only metadata. It always reports `online_ready: false`, including after
an optional `--observation FILE`. That file accepts only bounded fields: `version`,
profile/command/skill ID arrays, enum warnings (`invalid-policy`, `ignored-setting`,
`unknown-source`) and `truncated`. Its origin is **caller-supplied-not-verified**;
unknown/truncated/mismatched native state remains a blocker. B3 will supply owned
native lifecycle integration; B2 does not silently start a process to inventory it.
Do not supply full API/config/transcript exports, account data or native databases.

Configuration inputs support UTF-8 JSON/JSONC comments and trailing commas for
validation. Staged/original bytes are preserved, never round-tripped/merged.
Duplicate keys, unresolved placeholders, missing protected denials, review write
grants, child/shell-interpolating commands and known secret-bearing fields are
rejected without echoing rejected values. This is not universal secret detection:
the user must curate inputs and keep all credentials out of them.

## Explicit config transaction — separate live authorization

This capability is implemented/tested on temporary files, **not permission to
install into the user's setup or start the final human usage cycle**. Keep all
selected implementation complete before that later launch. The root skill
installer's `apply`/`restore` and manifests are unchanged and separate.

`opencode config` defaults to read-only preview. It emits the fixed relative
destination `.opencode/opencode.jsonc`, original digest (or `absent`), candidate
digest and no-op flag. The candidate must be external to the selected project and
active discovery. Preserve unrelated settings/comments by manually curating a
whole proposed file; no automatic JSONC merge, ancestor/global config rewrite,
service reload/restart or credential migration is performed.

Only after actual separately scoped authorization, explicit `config apply`
requires that preview's `--expect-original` and `--expect-candidate`, plus an
existing owner-private `--backup-root` separate from project/candidate/discovery.
Re-check the complete candidate diff; hash pins identify bytes, not user consent
or policy effectiveness. Existing config containing credentials must be reconciled
by the user rather than copied through this helper.

Before replacement, the helper reserves a private transaction directory, backs
up the original bytes/mode and persists `manifest.json`. It refuses changed
inputs/parents, symlinks/special files and intervening edits. Identical bytes
produce no rewrite or backup. Transactions assume a quiescent trusted filesystem;
they are not a race-proof sandbox against a concurrent hostile writer.

`config rollback --workspace DIR --backup TRANSACTION_DIR` validates the distinct
runtime-config manifest and backup/current digests and mode. It restores original
bytes/mode or removes only a verified newly added config file; it does not remove
the `.opencode` directory or unrelated state. A later edit blocks rollback. A
repeated already-restored rollback is a no-op, not a new backup.

If apply/rollback fails after backup reservation, the CLI exits nonzero and emits
an `execution: incomplete` transaction observation when possible. Preserve the
backup and reconcile phase/current files before any retry; `prepared` is not
proof that replacement occurred or that nothing changed. Do not delete backup
directories to retry, treat wrapper exit as model success or put private backups
in Git. Native prompt execution/receipts are later B3/B4, not delivered by B2.

## Native ACP session — 005/B3 (fake-peer transport, strict admission)

The `opencode session` group is a narrow ACP protocol-1 client over one owned
private `opencode acp` child. It is exercised only by synthetic fake peers and
**never starts a real native process or model in development**. Three operations:

```sh
agentic-sdd opencode session prepare --workspace /absolute/project --executable /absolute/opencode [--model M] [--mode M]
agentic-sdd opencode session run --workspace /absolute/project --executable /absolute/opencode --prompt TEXT [--model M] [--mode M] [--deadline SECS] [--launch FILE]
agentic-sdd opencode session pending --file /absolute/private/pending-….json
```

`prepare` (the default) validates the fixed executable identity, absolute
workspace and model/mode identifiers and emits `execution: prepared` with the
admission blockers. It starts nothing.

`run` is gated by strict admission (contract C4). A launch declaration
(`--launch FILE`) must carry authorization, intended effects, exact model/data
identity, and deadline/attempt/token/spend bounds — and even a fully declared
launch is still denied because the independent controls (strict pre-send token
proof, credential isolation, complete accounting, provider caps/allowance,
auxiliary-retry prevention) cannot be proven by a declaration. While any control
is unverified, `run` returns `execution: blocked` with those blockers and starts
no process. No `--force`, free-price exception or soft estimate exists.

The transport itself bounds each ACP message to 256 KiB, aggregate observations
to 10 MiB, control requests to 10 s and the session to an explicit deadline. It
performs the `initialize` handshake (advertising `elicitation.form`), creates one
session with an absolute `cwd` and no additional directories/MCP servers, selects
the exact offered model/mode before prompting, and refuses wrong versions,
foreign sessions, child-session metadata, unoffered models/modes, duplicate/late
RPC ids, oversized frames and unknown required events. A provider error with
process exit 0 is never task success.

On a consequential `elicitation/create` or permission request, the client keeps
it unanswered, cancels the owned prompt, and returns `execution: blocked` with
`stop_reason: pending_human` plus a sanitized resumable summary (exact question,
bounded options, `rationale: unknown` when absent, affected scope, evidence
pointers and next permitted action). It never fabricates an answer, guesses,
auto-approves or resumes.

`pending --file` re-emits a persisted summary for review only; it validates the
schema and never re-enters, resumes or replays the session. The summary is an
observation, not a second approval ledger. See [recovery](recovery.md) for the
reconciliation procedure.
