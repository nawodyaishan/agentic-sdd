# Runtime compatibility record

Recorded: 2026-10-28, B1 inventory plus B3 offline tests. **Support status: blocked / not yet certified.**

| Item | Pinned observation | Evidence |
| --- | --- | --- |
| CLI candidate | `2.2.0` | `opencode --version` → `opencode v2.2.0` |
| Existing service candidate | `2.2.0` | version-only projection of `GET /api/info` |
| Host | Linux amd64 | Go reports `go1.27.1 linux/amd64` |
| Repository Go minimum | 1.23 | `go.mod`; minimum-version execution not tested in B1 |
| Docker | client/server 29.9.1 | Go e2e later passed; B3 native container uses the existing pinned Python image, without pulls/network/credentials |
| Plugin/server revisions | unknown | Discovered IDs/status are not package pins |
| macOS runtime/process cancellation | untested | Darwin cross-compilation is not macOS runtime validation |

Use 2.2.0 as the reproducible candidate for later native fixtures; do not upgrade the running service to satisfy this record. Recheck CLI and service separately before testing. A version change invalidates relevant selection, policy, tool, compaction and cancellation observations.

B3's 9 native fixture tests pass for configuration discovery, permission
evaluation, policy precedence/warnings and synthetic MCP/plugin startup denial.
They do not certify actual model-originated tools, command/session model
selection, canonical paths, private-data handling or cancellation/recovery.
Historical B2 behavioral observations and spend correction are in
[baseline.md](../../harness/baseline.md). No known-good adopted SDD runtime
configuration exists to roll back to; preserve the user's original config
before any later explicitly authorized installation. Details and blockers:
[verification.md](verification.md).

## Offline S0 revalidation — 2026-10-31

The available binary now reports **2.2.5**. After the initial 2.2.0 guard stopped
before prompt submission, the user allowed newer versions for isolated testing.
S0 freezes that exact 2.2.5 binary/hash, exercises fake-response HTTP construction
and repeats the nine credential-free compatibility probes successfully; see
[S0 scope and results](../../tests/harness/S0.md). This is not real endpoint/model
acceptance, a new adopted config, blanket support for every version above 2.2.0,
or a current observation of the user's shared service. The old probe wrapper's
2.2.0 default is preserved; only S0 explicitly selects 2.2.5. Existing tool,
selection, recovery, privacy/budget and macOS blockers remain. Product support
is still **not certified**.

## Feature 005 implementation target — 2026-11-02

The file-only [inactive helper](usage.md) declares target **2.2.5**, using pinned
upstream commit `9bbb8ce18c80f099e0e448775f59d965ead40a21` for source contracts.
Its generated candidate is distinct from the older static B3 example and uses
typed `mcp.use`, `plugin.use` and `skill.use` policies. Source-level/deterministic
tests do not certify actual mode/tool/adoption behavior. No native service or
real model was run for B2; later ACP integration is B3 and the human usage cycle
comes only after all selected 005 implementation. No blanket version support
or live runtime/default-model change follows.

## Native ACP transport — 005/B3 (2026-11-02)

The `opencode session` group implements a narrow ACP protocol-1 client pinned to
the same 2.2.5 source contract, exercised only by synthetic fake peers (this test
binary re-executed as a scripted ACP server over fd 3). It bounds each message to
256 KiB and performs the `initialize` handshake (advertising `elicitation.form`),
session creation with an absolute `cwd` and no additional directories/MCP servers,
and exact model/mode selection before a prompt. Positive and negative outcomes
(happy end_turn, elicitation/permission pending-human, provider error with exit 0,
missing terminal event, child-session metadata, foreign session, duplicate/late
ids, oversized frames, wrong version, deadline) are covered.

These tests do not certify real native behavior: no `opencode acp` process or
model was started, and strict admission stays denied while token/egress/
accounting/credential controls are unverified. Support remains **not certified**.

## Evidence consumer and receipt projection — 005/B4 (2026-11-02)

`internal/harnessevidence` implements the production v1 receipt consumer
(strict typed/semantic validation, scoped `sha256-v1` identity, safe external
export) and `internal/opencodeadapter` projects a runner result plus an optional
owned-transport observation into the same v1 receipt. The `agentic-sdd opencode
evidence snapshot|validate|export|receipt` group exposes them.

Verification is offline and deterministic: Go tests cover valid/invalid receipt
round trips, stale identity and authority, evidence tampering/mode/symlink/
traversal, ready/coverage rules, redaction and overwrite refusal, projection
failure modes and fake-peer elicitation. Golden identity digests were derived
independently by the Python reference consumer
(`tests/harness/go_snapshot_fixture.py`), and the 18 reference receipt tests
still pass. No native process, model, provider or credential store was touched.
This does not certify native runtime behavior, billing or provenance; support
remains **not certified**.

## Final integration — 005/B5 (2026-11-02)

All selected 005 implementation is delivered at source level: the 18-ID helper
package, inactive candidate/config transaction, narrow ACP session and
pending-human normalization, and the production evidence consumer/projection.
`make test`, `make vet`, `go test -race ./...`, `git diff --check`, the required
local check contract and the offline Python reference/schema fixtures pass on
the final source identity recorded in 005 tasks.

Delivery does not certify runtime support. No real `opencode acp` process,
model, provider, credential store or real-home adoption was used; strict
admission remains denied; native tool/permission/selection/recovery behavior,
macOS and billing/provenance stay unobserved. The one integrated human cycle is
prepared in the [final usage plan](../../docs/vnext-human-test-plan.md) and is a
separate launch decision. Support remains **not certified**.
