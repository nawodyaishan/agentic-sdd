# Fleet Manager in the terminal UI

> Migration stub (Feature 020, T08, lane B4-W1-B). This file is no longer the Fleet Manager manual. Canonical ownership moves to the pages below; this stub preserves old fragment anchors and records the preview-vs-apply limits a maintainer still needs. Review status: unreviewed. Applies to: unreleased.

## Canonical destinations

- How-to fleet: `how-to/fleet` (integration-pending — authored in a parallel lane).
- Fleet reference: `reference/fleet` (integration-pending — authored in a parallel lane).
- TUI route and key detail lives with the TUI canonical owner, not here.

## Retained operational notes

- Deployment and restore in the current build are preview only; the review and recovery screens show the exact plan while the live write stays disabled unless a reviewed build enables it.
- The in-app source editor does not save the personal `fleet.json` source; saving the source is a separate, explicitly approved capability this build does not ship, and the next session starts with no source selected.
- Clearing an existing model override back to canonical inheritance is a direct file operation outside the in-app picker; the picker accepts replacement values but not override removal.
- The surface reads no credentials, invokes no model and makes no provider or network call; installed is not selected primary, not authenticated and not runtime-verified, and no subscription, price, entitlement or quota state is shown or promised.
- Source choice, staged edits and view state are session-only; nothing here authorizes a live apply, a real-home write, a release or a publication.
- Canonical workflow and acceptance detail remain in the historical sources linked below; this stub changes no normative behavior.

## Retained anchors

Old `docs/fleet-manager-tui.md` fragments below resolve on this page and forward to the canonical destinations above. No procedures or configuration values are duplicated here.

<a id="fleet-manager-in-the-terminal-ui"></a>
<a id="where-it-lives"></a>
<a id="first-use"></a>
<a id="five-things-to-keep-apart"></a>
<a id="the-journey"></a>
<a id="conflicts-and-limits"></a>
<a id="cli-equivalents-and-further-reading"></a>

Historical sources (unchanged, not canonical manuals): [Fleet profiles worked guide](../specs/014-agent-fleet-model-overrides/fleet-worked-guide.md), [Feature 015 spec](../specs/015-fleet-manager-tui/spec.md), [TUI architecture](tui-architecture.md) (integration-pending — owned by a parallel lane).
