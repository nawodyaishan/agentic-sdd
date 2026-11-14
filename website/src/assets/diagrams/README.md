# Diagrams D01–D05 (Feature 020, B3-W1-K)

Purpose: five small, legible Mermaid sources backing the architecture/security
explanations. The `.mmd` source is authoritative; any PNG is a generated
projection, never a second editable diagram.

Authoring tool expectation: Mermaid source authored by hand; projection
rendered by the maintainer with a pinned `mermaid-cli` (`mmdc`) invocation.
No runtime Mermaid integration and no headless-browser dependency in CI.

Exact render command (maintainer, network install if needed):

```sh
for f in D01-context D02-components D03-install-transaction D04-delegation D05-evidence; do
  mmdc -i website/src/assets/diagrams/$f.mmd -o website/src/assets/diagrams/$f.png -b white
done
```

Source revision: `449245a`.

Provenance: PNGs are `blocked` in this lane — no `mermaid-cli` exists locally
and no network install is authorized. `.mmd` files only; maintainer renders
and records tool version + source/image hashes at integration.

Alt text (one line each):

- D01-context: human uses asdx-core CLI or asdx TUI to sync embedded/repo skill sources into four client skill dirs; models/MCP are external.
- D02-components: CLI dispatches directly to skillsync/fleet/MCP/execution engines; TUI pairs with a same-install companion after a buildinfo handshake; legacy names forward.
- D03-install-transaction: preview stops before writes; apply goes plan, backup, write, verify with rollback; restore reloads a backup; fleet/MCP/execution stores are separate.
- D04-delegation: human authorizes one bounded brief, child returns evidence, main reconciles, human approves; no daemon or automatic routing.
- D05-evidence: MCP applies config under a lock with snapshot verify/rollback; execution records briefs/checkpoints and evaluates freshness for review; neither implies sandbox or live auth.
