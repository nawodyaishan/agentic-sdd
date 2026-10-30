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
