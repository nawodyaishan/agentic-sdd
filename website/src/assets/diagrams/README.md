# Diagrams D01–D05 (Feature 020, B3-W1-K)

Purpose: five small, legible Mermaid sources backing the architecture/security
explanations. The `.mmd` source is authoritative; the PNG is a generated
projection, never a second editable diagram.

## Rendering

Projections are produced with a **digest-pinned** Mermaid CLI container. No
runtime Mermaid integration and no headless-browser dependency in the site
build or CI.

```sh
IMG='ghcr.io/mermaid-js/mermaid-cli/mermaid-cli@sha256:18938e53497376d524e04b1fc0bf7b6d61a491ce9ff7032bb2855d57155acb10'
for pair in D01-context:d01-context D02-components:d02-components \
            D03-install-transaction:d03-install-transaction \
            D04-delegation:d04-delegation D05-evidence:d05-evidence; do
  src=${pair%%:*}; out=${pair##*:}
  docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$PWD/website/src/assets/diagrams:/src" -v "$PWD/website/public/diagrams:/out" "$IMG" \
    -i "/src/$src.mmd" -o "/out/$out.png" -b white
done
```

Pinned image digest: `sha256:18938e53497376d524e04b1fc0bf7b6d61a491ce9ff7032bb2855d57155acb10`
(observed 2026-11-14; the same digest is published as `minlag/mermaid-cli`).
Rendered 2026-11-16 from source revision `449245a`.

Note for editors: Mermaid treats `;` as a statement separator inside sequence
notes and messages. Write `Report, then await approval`, not `Report; await
approval`, or the parser splits the line and fails.

## Projections

| Source | Size (px) |
|---|---|
| `D01-context.mmd` → `public/diagrams/d01-context.png` | 784 × 296 |
| `D02-components.mmd` → `public/diagrams/d02-components.png` | 784 × 689 |
| `D03-install-transaction.mmd` → `public/diagrams/d03-install-transaction.png` | 784 × 342 |
| `D04-delegation.mmd` → `public/diagrams/d04-delegation.png` | 784 × 458 |
| `D05-evidence.mmd` → `public/diagrams/d05-evidence.png` | 542 × 1173 |

Alt text (one line each):

- D01-context: human uses asdx-core CLI or asdx TUI to sync embedded/repo skill sources into four client skill dirs; models/MCP are external.
- D02-components: CLI dispatches directly to skillsync/fleet/MCP/execution engines; TUI pairs with a same-install companion after a buildinfo handshake; legacy names forward.
- D03-install-transaction: preview stops before writes; apply goes plan, backup, write, verify with rollback; restore reloads a backup; fleet/MCP/execution stores are separate.
- D04-delegation: human authorizes one bounded brief, child returns evidence, main reconciles, human approves; no daemon or automatic routing.
- D05-evidence: MCP applies config under a lock with snapshot verify/rollback; execution records briefs/checkpoints and evaluates freshness for review; neither implies sandbox or live auth.
