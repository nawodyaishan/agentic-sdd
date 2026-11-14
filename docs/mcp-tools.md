# MCP Tools: CodeGraph, Context7 and Exa

> Migration stub (Feature 020, T08, lane B4-W1-B). This file is no longer the MCP manual. Canonical ownership moves to the pages below; this stub preserves old fragment anchors and records the offline-vs-connected distinction a maintainer still needs. Review status: unreviewed. Applies to: unreleased.

## Canonical destinations

- How-to MCP: `how-to/mcp` (integration-pending — authored in a parallel lane).
- MCP reference: `reference/mcp` (integration-pending — authored in a parallel lane).
- Security and trust boundaries for credentials and tool availability live with their canonical owners, not here.

## Retained operational notes

- Read-only status, offline doctor and preview write nothing, execute nothing and contact no model, installer or remote endpoint.
- A configured server is not a connected, authenticated or working server. Without an explicit check the state reads authentication not checked and connection not tested; availability reporting never transmits private source, credentials or unrelated repository contents.
- Connection and network checks are a separately approved capability and are refused here; the TUI surface carries no connection test.
- Apply remains opt-in and digest-gated with a shared write slot and managed-entry-only backups; unmanaged entries, unknown fields, disabled servers and unsafe targets are preserved or refused, never silently merged or forced.
- Global configuration scope only; nothing is created and no general uninstaller, forced replace or hidden install path exists here.
- Governing requirements remain in the historical Feature 015 spec linked below; this stub changes no normative behavior.

## Retained anchors

Old `docs/mcp-tools.md` fragments below resolve on this page and forward to the canonical destinations above. No procedures or configuration values are duplicated here.

<a id="mcp-tools-codegraph-context7-and-exa"></a>
<a id="the-beginner-journey"></a>
<a id="cli-equivalents"></a>
<a id="read-only-first-mcp-a"></a>
<a id="applying-and-restoring-mcp-b"></a>
<a id="global-configuration-only-nothing-is-created"></a>
<a id="jsonc-conflicts-and-unsafe-targets"></a>
<a id="codegraph-prerequisite-and-index"></a>
<a id="context7-and-exa-honesty"></a>
<a id="tui-keys"></a>
<a id="what-this-guide-does-not-promise"></a>

Historical requirements source (unchanged, not a canonical manual): [Feature 015 spec](../specs/015-fleet-manager-tui/spec.md).
