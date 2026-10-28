# Runtime compatibility record

Recorded: 2026-10-28, B1. **Support status: blocked / not yet certified.**

| Item | Pinned observation | Evidence |
| --- | --- | --- |
| CLI candidate | `2.2.0` | `opencode --version` → `opencode v2.2.0` |
| Existing service candidate | `2.2.0` | version-only projection of `GET /api/info` |
| Host | Linux amd64 | Go reports `go1.27.1 linux/amd64` |
| Repository Go minimum | 1.23 | `go.mod`; minimum-version execution not tested in B1 |
| Docker | client/server 29.9.1 | version-only query; required e2e image missing |
| Plugin/server revisions | unknown | Discovered IDs/status are not package pins |
| macOS runtime/process cancellation | untested | Darwin cross-compilation is not macOS runtime validation |

Use 2.2.0 as the reproducible candidate for later native fixtures; do not upgrade the running service to satisfy this record. Recheck CLI and service separately before testing. A version change invalidates relevant selection, policy, tool, compaction and cancellation observations.

No model capability smoke test, permission denial, compaction/recovery test or behavioral comparison has passed yet. No known-good SDD runtime configuration exists to roll back to; preserve the user's original configuration before any later opt-in adoption. Current original-config digest is retained in [baseline.md](../../harness/baseline.md), without its contents.
