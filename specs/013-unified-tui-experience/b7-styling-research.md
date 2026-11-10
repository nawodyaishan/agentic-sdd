# B7 styling research — terminal UI refinement

**Status:** research input for the B7 draft. Not approved and not an implementation authorization.
**Date:** 2026-11-10
**Question:** How should the Feature 013 TUI be styled further without weakening its read-only, preview-first safety model or its terminal compatibility?

## 1. Inputs

- Owner styling brief supplied 2026-11-10 (two-tone palette, semantic status colors, borders/spacing, header/footer/breadcrumbs, focusable lists, overlay modals, inputs, spinners/progress/toasts, glyph hygiene, adaptive contrast).
- Current shell: `experiments/tui/theme.go`, `app.go`, `app_screens.go`, `viewport.go`, `catalog_screen.go`, `install.go`, `restore.go`, `activity.go`, `onboarding.go`.
- Pinned libraries: `charm.land/bubbletea/v2 v2.0.2`, `charm.land/bubbles/v2 v2.1.0`, `charm.land/lipgloss/v2 v2.0.3`, `github.com/charmbracelet/x/ansi v0.11.7`.

## 2. Research findings

### 2.1 Hierarchy is spacing and alignment, not decoration

- Mature terminal design guidance converges on content-first, quiet interfaces: color, borders and backgrounds communicate state or hierarchy, never decoration; keep one obvious focus point and stable geometry. See the [hepi-mono DESIGN.md](https://github.com/hheei/hepi-mono/blob/main/DESIGN.md) and [Comet's terminal UX notes](https://www.comet.com/site/blog/agent-first-terminal-ux/).
- Use at most one blank line between semantic blocks; prefer whitespace, indentation and alignment over extra containers. See [GitHub TUIKit foundations](https://github.com/github/TUIKit/blob/main/docs/foundations.md).
- Define semantic tokens (`text`, `muted`, `dim`, `accent`, `selectedBg`, `border`, `borderFocus`, `success`, `warning`, `error`) instead of feature-specific colors; the [TUI design skill](https://github.com/diegomarino/tui-design/blob/main/skills/tui-design/SKILL.md) rejects raw hex in mockups and requires glyph+text+color for status.
- The minimal TUI design reference recommends one empty line between components, one-cell internal panel padding, two-space indentation, and left alignment except timestamps/numbers; see [awesome-tui-design minimal](https://github.com/cola-runner/awesome-tui-design/blob/master/designs/minimal/DESIGN.md).

### 2.2 Color must survive the user's terminal

- Design monochrome-first; named ANSI colors adapt to the user's theme, while hard-coded hex may not; color is enhancement. See [Comet](https://www.comet.com/site/blog/agent-first-terminal-ux/).
- Lip Gloss v2 supports adaptive rendering through `lipgloss.HasDarkBackground`, `lipgloss.LightDark`, `compat.AdaptiveColor`, and `lipgloss.Complete` for TrueColor/ANSI256/ANSI fallbacks. With Bubble Tea, request `tea.RequestBackgroundColor` and handle `tea.BackgroundColorMsg`; downsampling is built into Bubble Tea v2. Source: [Lip Gloss v2 upgrade guide](https://github.com/charmbracelet/lipgloss/blob/main/UPGRADE_GUIDE_V2.md), [Lip Gloss README](https://github.com/charmbracelet/lipgloss), Context7 `/charmbracelet/lipgloss`.
- Contrast guidance: WCAG AA body text ≥4.5:1 and ≥3:1 for large/bold text is a common floor; the current Midnight Lagoon palette already targets AAA (≥7:1) text and 3:1 borders. See [Monospace Design TUI §9](https://coreyt.github.io/monospace-design-tui/standard/accessibility/).
- `NO_COLOR` must remain respected; reduced-color terminals need a semantic ANSI-16 fallback rather than a broken TrueColor-only shell.
- Status must never be color-only: pair a glyph and a text label with color; see [TUI design skill](https://github.com/diegomarino/tui-design/blob/main/skills/tui-design/SKILL.md) and [Monospace Design TUI §9](https://coreyt.github.io/monospace-design-tui/standard/accessibility/).

### 2.3 Borders, spacing and responsive panes

- Lip Gloss v2 offers `NormalBorder`, `RoundedBorder`, custom `Border` structs, `JoinHorizontal`/`JoinVertical`, and `Place`/`PlaceHorizontal`/`PlaceVertical` for centering and overlay boxes. Source: Context7 `/charmbracelet/lipgloss`; [Lip Gloss layout API](https://github.com/charmbracelet/lipgloss/blob/main/_autodocs/api-reference/layout.md).
- Prefer one border style per app, 1–2 cells of horizontal padding inside framed surfaces, and whitespace/section dividers over nested boxes; see [awesome-tui-design minimal](https://github.com/cola-runner/awesome-tui-design/blob/master/designs/minimal/DESIGN.md) and [hepi-mono](https://github.com/hheei/hepi-mono/blob/main/DESIGN.md).
- Use flexible master-detail layouts with a stable list column and a wide detail pane; collapse gracefully below a minimum width. The current shell already has 40/60/79/80/109/110 breakpoints and short-height collapse; B7 should refine, not replace, that behavior.

### 2.4 Navigation and chrome

- A single top bar should show product identity, current context/breadcrumb, and global status. A pinned footer should carry context-sensitive key hints; global hints can sit on the right. See [TUI design skill](https://github.com/diegomarino/tui-design/blob/main/skills/tui-design/SKILL.md) and [hepi-mono](https://github.com/hheei/hepi-mono/blob/main/DESIGN.md).
- Nested views need a compact breadcrumb (`Install & Manage > Backups > Details`) so the depth is visible without opening help.
- Active tabs can be solid/blocks or bracketed labels; inactive tabs stay text-only. The current shell has a colored top bar but no breadcrumb/tab/status cluster.
- Footer hints should be contextual and stable; disabled/unsafe keys should not appear as actionable hints. The current footer is plain text and already changes by route.

### 2.5 Lists, focus and overlays

- Bubbles `list.DefaultDelegate` exposes `Styles` (`NormalTitle`, `NormalDesc`, `SelectedTitle`, `SelectedDesc`, `DimmedTitle`, `DimmedDesc`, `FilterMatch`) and `DefaultStyles(isDark)`; the selected row can carry a marker, bold/tint or reverse video. Source: Context7 `/charmbracelet/bubbles`; [Bubbles list README](https://github.com/charmbracelet/bubbles/blob/main/list/README.md).
- A stable cursor marker (`▸`, `>`, `▶`) plus bold/tint is more robust than color alone; keep the marker width reserved on unselected rows so geometry does not shift. See [hepi-mono](https://github.com/hheei/hepi-mono/blob/main/DESIGN.md) and [awesome-tui-design minimal](https://github.com/cola-runner/awesome-tui-design/blob/master/designs/minimal/DESIGN.md).
- Overlay modals should be centered floating boxes that dim the background; Lip Gloss `Place` can center a bounded box. The current help is a full-screen replacement and onboarding is an overlay-style full body, not a floating modal.
- Inputs/filters need a distinct focus border, placeholder text and a visible cursor. The current Explore filter uses the Bubbles list's filter input.

### 2.6 Feedback and live indicators

- Use real phase spinners (Bubbles `spinner`) next to a status verb for indeterminate work; use `progress.ViewAs(percent)` only when the operation genuinely reports progress. The current write host already emits `starting`/`writing`/`flushing` phases, which can drive a spinner but not a fake percentage.
- Toast-style transient notices belong inline in the footer or a corner, auto-dismissed; blocking modals are for decisions, not status. See [Comet](https://www.comet.com/site/blog/agent-first-terminal-ux/) and the owner brief.
- The current app has `notice` strings rendered in the page body; B7 can style them as inline status pills/toasts while preserving the full text.
- Bubble Tea v2 `tea.View` exposes `ProgressBar` and `ProgramStatus` (OSC 7501) for terminals that support them; these are progressive enhancements, not replacements for visible text. Source: [Bubble Tea v2 upgrade guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md).

### 2.7 Glyphs, emoji and width hazards

- Prefer safe single-width Unicode (`●`, `◆`, `▪`, `✓`, `✗`, `▸`, `→`) over emoji. Emoji and East-Asian-wide glyphs can render as one or two cells depending on terminal/font; Nerd Font glyphs likewise have width ambiguity. See [Nerd Fonts FAQ](https://github.com/ryanoasis/nerd-fonts/wiki/FAQ-and-Troubleshooting), [WezTerm cell_widths](https://wezterm.org/config/lua/config/cell_widths.html), and [wcwidth-icons](https://github.com/powerman/wcwidth-icons).
- The current route icons use emoji (`🏠`, `📦`, `🔍`, `💡`) with an ASCII alternative; B7 should make the default safe-Unicode set the primary visual and keep emoji optional/opt-in, or prove the width in a cell-width fixture before keeping them.
- Nerd Font icons are not a safe default for a general-purpose CLI; keep them out unless the owner explicitly opts in. OSC 8 hyperlinks are already best-effort and paired with visible URLs.

### 2.8 Technical seams confirmed for B7

- `tea.View` fields: `Content`, `AltScreen`, `Cursor`, `ForegroundColor`, `BackgroundColor`, `ProgressBar`, `ProgramStatus`, `WindowTitle`, `MouseMode`, `ReportFocus`, `KeyboardEnhancements`, `OnMouse`.
- `tea.RequestBackgroundColor` + `tea.BackgroundColorMsg` for adaptive styles; `lipgloss.LightDark` for light/dark variants.
- `lipgloss.Place`/`PlaceHorizontal`/`PlaceVertical` for centered overlays; `JoinHorizontal`/`JoinVertical` for split layouts.
- Bubbles `list.DefaultDelegate.Styles`, `spinner.Model`, and `progress.Model.ViewAs` are available in the already-pinned versions. No new dependency is required for B7.

## 3. Gap analysis of the current shell

| Area | Current state | B7 opportunity |
|---|---|---|
| Palette | TrueColor Midnight Lagoon only; `NO_COLOR` strips styles | Add semantic token layer with LightDark and ANSI-16/ANSI-256 fallback; preserve AAA on dark and validate light |
| Chrome | Colored top bar, focus text, footer hints | Add breadcrumb, tab/route pills, compact status cluster, styled footer keys |
| Borders | Mostly whitespace; some section dividers | One border style for framed surfaces/modals; 1-cell padding; no box nesting |
| Lists | Bubbles default delegate | Stable cursor marker + selected tint/bold; fixed headers; filter focus styling |
| Overlays | Help is full-screen; onboarding is full-body | Centered modal for help/confirmations with dimmed background; keep full-screen fallback on tiny terminals |
| Feedback | Busy footer text and page notices | Phase spinner, inline status pill/toast, native `ProgressBar` only where truthful |
| Glyphs | Emoji route icons + ASCII toggle | Safe Unicode default; emoji only if width-proven; keep ASCII fallback |
| Adaptive | No background query; no light-mode palette | `tea.RequestBackgroundColor` + `LightDark`; preserve explicit color-off tests |
| Inputs | List filter input | Focus border/placeholder/cursor styling; no behavior change |
| Accessibility | AAA dark tests, plain/ASCII tests, screen-reader unclaimed | Add light-mode and reduced-color contrast tests; keep text labels for every status |

## 4. Proposed B7 scope for the architect

B7 should be a styling-only batch:

- Introduce a semantic style/token layer (surface, text, muted, dim, accent, selected, border, focus, success, warning, error, info) and keep `theme.go` as the single palette source.
- Add adaptive light/dark rendering using `tea.RequestBackgroundColor`/`lipgloss.LightDark`, plus ANSI-16/256 fallback and `NO_COLOR` behavior.
- Refine chrome: breadcrumb/trail, route/tab pills, status cluster, padded header, contextual footer with key-pill styling.
- Refine lists/detail: stable cursor marker, selected row tint/bold, fixed list column, filter focus, no geometry shift.
- Add a bounded centered help/confirmation modal pattern with dimmed backdrop and a tiny-terminal fallback.
- Add truthful feedback: phase spinner during writes, inline status/toast styling, optional native progress/status fields; no fake percentage.
- Normalize glyph vocabulary: safe single-width Unicode default, ASCII fallback, emoji/Nerd Font opt-in or removal; cell-width fixtures.
- Preserve all behavior, write flows, confirmation semantics, byte contracts, read-only boundaries, root `go.mod` isolation and existing tests.

## 5. Guardrails

- B7 is **not approved**; this note only informs the draft.
- No new dependency without explicit justification; no Huh unless it is pinned and proven necessary.
- No root `go.mod` changes, no root dependency, no move out of `experiments/tui`.
- No live apply, release, tag, formula publication or real-home operation.
- Every visual change needs a deterministic test plus an owner visual walkthrough; platform-specific checks stay `not_run` until evidenced.

## 6. Sources

- [hepi-mono DESIGN.md](https://github.com/hheei/hepi-mono/blob/main/DESIGN.md)
- [Comet — UX Without a UI](https://www.comet.com/site/blog/agent-first-terminal-ux/)
- [diegomarino/tui-design skill](https://github.com/diegomarino/tui-design/blob/main/skills/tui-design/SKILL.md)
- [Monospace Design TUI — Accessibility](https://coreyt.github.io/monospace-design-tui/standard/accessibility/)
- [GitHub TUIKit foundations](https://github.com/github/TUIKit/blob/main/docs/foundations.md)
- [awesome-tui-design minimal](https://github.com/cola-runner/awesome-tui-design/blob/master/designs/minimal/DESIGN.md)
- [Lip Gloss v2 upgrade guide](https://github.com/charmbracelet/lipgloss/blob/main/UPGRADE_GUIDE_V2.md)
- [Lip Gloss layout API](https://github.com/charmbracelet/lipgloss/blob/main/_autodocs/api-reference/layout.md)
- [Bubbles list README](https://github.com/charmbracelet/bubbles/blob/main/list/README.md)
- [Bubble Tea v2 upgrade guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md)
- [Nerd Fonts FAQ](https://github.com/ryanoasis/nerd-fonts/wiki/FAQ-and-Troubleshooting)
- [WezTerm cell_widths](https://wezterm.org/config/lua/config/cell_widths.html)
- [wcwidth-icons](https://github.com/powerman/wcwidth-icons)
- Context7: `/charmbracelet/lipgloss`, `/charmbracelet/bubbles`, `/charmbracelet/bubbletea`
- Owner styling brief, current session, 2026-11-10.
