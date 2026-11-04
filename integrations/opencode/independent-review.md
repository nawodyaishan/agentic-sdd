# Independent review: opt-in example runbook (006/B3)

**Status: inactive example, not installed or certified.** It exists only in this
repository. Nothing here installs it into a project, changes a real home, starts a
reviewer, calls a model or launches the RHEL pilot. Support remains
[uncertified](supported-version.md); see [verification](verification.md) for the
wider adapter blockers.

## What it is

- [`independent-review.jsonc`](independent-review.jsonc): one read-only coordinator
  (`sdd-review-coordinator`, primary) and one inspection-only reviewer
  (`sdd-independent-reviewer`, subagent). Both are deny-first, bind
  `opencode/big-pickle` with variant `default`, and are bounded to 4 and 6 steps.
  The example adds no commands, providers, MCP or plugins, and no global subagent
  policy.
- The method the profiles follow: [independent review reference](../../skills-files/agentic-sdd-router/references/independent-review.md).
- Use it only for an explicit user request for an independent review. If the route
  is unavailable or blocked, report it as blocked. Do not present a same-session
  review as independent.

## Checks that exist

| Check | Command | Observed on 4 Nov 2026 |
| --- | --- | --- |
| Static profile, permission and collision expectations | `python3 -B -m unittest discover -s tests/harness -p 'test_independent_review*.py'` | 13/13 pass |
| Scratch digest, backup, rollback and refusal walkthrough | `sh tests/harness/independent-review-transaction.sh` | all cases pass; temporary files only |
| Offline native permission probe, pinned OpenCode 2.2.5 | `sh tests/harness/native-independent-review.sh` | 5/5 pass in a network-disabled, read-only, non-root container |
| Packaged reference and pointers | `go test ./internal/skillsync` | pass; the reference ships and both pointers reach it |

The native probe checks permission evaluation and the subagent-launch collision
only. It sends no prompt and starts no model. Passing it does not show that a
child agent actually runs, that its tools are restricted in dispatch, or that a
path or symlink is refused at read time.

## Why the 005 config helper is not used

`agentic-sdd opencode config preview` rejects this candidate with
`candidate must retain the inactive selected defaults`. That helper validates the
005 candidate contract. The procedure below therefore uses ordinary file operations,
which the scratch script walks through. This does not change the 005 helper.

## Manual, digest-checked procedure (not run on any real project)

1. **Check collisions first.** This procedure writes one `.opencode/opencode.jsonc`.
   If the project already has OpenCode configuration (for example the 005 template or
   other settings), hand-merge it into one reviewed whole file; do not concatenate.
   Discovery order across other config locations is not verified here. Check any higher-authority
   global config: if it denies `tool.use subagent:*` (for example
   [policy-lockdown.jsonc](policy-lockdown.jsonc)), the coordinator cannot launch the
   reviewer. That is a blocked route, which the native probe observed. Do not edit
   the lockdown to make it work.
2. **Curate.** Write the whole proposed file outside the project, in a private
   0700 directory. Keep unrelated keys; remove nothing you did not review.
3. **Digest.** Record `sha256sum` of the current file, or `absent`, and of the
   curated file. If they match, stop: this is a no-op and nothing is backed up.
4. **Back up.** Create a private 0700 transaction directory outside the project.
   Copy the original with `cp -p`, and store its digest.
5. **Replace.** Write the curated file to a sibling temporary path and `mv` it over
   the target. Confirm the digest equals the curated digest.
6. **Verify.** Reload and read back only through the user's own runtime. Do not use
   any other provider, credential or service.
7. **Roll back.** Compare the current digest with the applied digest. If they differ
   (for example after an intervening edit), refuse and leave the file untouched. If
   they match, restore the original bytes and mode from the backup, or remove the new
   file when the original was `absent`. Do not remove the `.opencode` directory or
   unrelated files. A repeated rollback after restoration is a no-op.

## Observed versus not observed

| Area | Observed offline | Not observed |
| --- | --- | --- |
| Profiles and mode | Both profiles and modes are exposed by the runtime | `variant` is not exposed by `/api/agent`; the binding is only in the file |
| Permissions | Named reads, the named skill and the coordinator's named launch are allowed; delegation, edits, shell, execution, search, web, MCP, browser and Code Mode are denied | Read-time path or symlink enforcement; tool dispatch; a real child invocation |
| Collision | A global `subagent:*` denial blocks the coordinator's launch | Other global or managed sources in a real install |
| Config transaction | Digest, backup, replace and rollback refusal work on temporary files | Real project or real home; OpenCode reading the curated file |
| Compaction | Not disabled by the example (no verified 2.2.5 key) | Any compaction or maintenance request |
| Model and cost | The runtime reports the model as the provider and ID pair | Eligibility, privacy, price, allowance and any cap |

## Pilot brief — prepared, not launched

Feature 006 B4 remains **blocked**. This brief lists what is verified and what the
user must supply before any live reviewer run.

**Scope when approved:** one disposable project (not the shared service or a real
home), one seeded behavior defect, one insufficient test, and one clean candidate.
Ground truth is kept out of the reviewer's readable paths. Use the bounds below and
do not widen them without a new decision.

| Bound (C4 proposal) | Maximum |
| --- | --- |
| Observations | 3: seeded case, clean case, cancellation case |
| Sessions | 1 parent, at most 1 child, no nesting |
| Model steps | Parent 4, child 6 |
| Requests | 30 across parent, child and auxiliary calls; 0 retries |
| Input | 12,000 per request; 360,000 aggregate |
| Output | 2,000 per request; 60,000 aggregate |
| Time | Child 120 s; observation 180 s; pilot 600 s |
| Fixture | At most 8 reviewed text files, 32 KiB total |
| Paid use | None proposed |

**Verified now:** runtime `opencode v2.2.5` with binary sha256
`8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050`; the example
and its static and native checks; the procedure above.

**Still needed from the user before B4 can be proposed for launch** (minimum set):

1. RHEL target: release, host, user and the disposable workspace path. Do not
   connect to or provision the VM to discover these.
2. Model: current eligibility and data-privacy terms for `opencode/big-pickle`, its
   price and remaining allowance, and either a dedicated provider-side cap or an
   explicit decision that no paid use is allowed.
3. Credential route as a secret reference the user supplies. Do not copy or read a
   primary credential store.
4. Enforcement evidence for request, token, deadline and cancellation limits, and
   for how all parent, child and auxiliary requests are counted (IR-02). Without it,
   the pilot stays blocked.
5. Cleanup owner, the owned session identifiers to stop, and who decides on
   continuation after a stop.

Until items 1–5 are supplied and separately authorized, no reviewer invocation,
inference, VM access or credential read is permitted under this runbook.
