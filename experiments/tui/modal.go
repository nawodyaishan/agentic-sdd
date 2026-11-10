package tui

// modal.go owns the single cell-aware modal compositor for the root viewport.
//
// Help and the install/restore confirmation screens keep their existing
// renderers and handlers. They only supply content through modalParts; this
// file frames that content, centers it over a dimmed backdrop when the pinned
// title/warning/policy/footer rows fit, and falls back to a full-screen
// scrollable presentation on narrow or short terminals. It deliberately owns
// no program loop, dialog controller, consent state or mouse semantics.

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type modalMode int

const (
	modalNone modalMode = iota
	modalCentered
	modalFullscreen
)

type modalKind int

const (
	modalKindHelp modalKind = iota
	modalKindInstallConfirm
	modalKindRestoreConfirm
)

// modalParts is the content contract between an existing renderer and the
// compositor. Warning and Policy are pinned rows; Body is the only paged
// region. Every slice is already wrapped for the width it was built at.
type modalParts struct {
	Title    string
	Subtitle string
	Warning  []string
	Policy   []string
	Body     []string
	Footer   string
}

// modalMaxOuterWidth caps the dialog so it stays readable on wide terminals.
const modalMaxOuterWidth = 88

// activeModalKind reports which presentation, if any, owns the root viewport.
// A started write always owns the shell, so no modal may cover its policy.
func (a App) activeModalKind() (modalKind, bool) {
	if a.write != nil || a.childWritePending() {
		return 0, false
	}
	if a.helpOpen {
		return modalKindHelp, true
	}
	if a.activeRoute == RouteInstall {
		switch a.installSection {
		case installSectionInstall:
			if a.install.Step() == InstallStepConfirm {
				return modalKindInstallConfirm, true
			}
		case installSectionBackups:
			if a.restore.Stage() == RestoreStageConfirm {
				return modalKindRestoreConfirm, true
			}
		}
	}
	return 0, false
}

// modalPresentation resolves the active modal's content and geometry. It
// returns the mode and the box inner width; centered uses the capped dialog
// width, full-screen uses one cell of side margin.
func (a App) modalPresentation() (modalParts, modalMode, int, int) {
	kind, ok := a.activeModalKind()
	if !ok {
		return modalParts{}, modalNone, 0, 0
	}

	bodyStart := a.headerRowCount()
	footerRows := a.footerRowCount()
	available := a.height - bodyStart - footerRows
	if available < 1 {
		available = 1
	}
	maxBoxH := available
	if limit := a.height - 4; limit < maxBoxH {
		maxBoxH = limit
	}

	if a.width >= 60 && a.height >= 16 && maxBoxH >= 3 {
		outer := min(modalMaxOuterWidth, a.width-4)
		inner := outer - 4 // one border and one padding cell on each side
		if inner >= 8 {
			parts := a.modalPartsFor(kind, inner)
			if a.modalFits(parts, inner, maxBoxH) {
				return parts, modalCentered, inner, maxBoxH
			}
		}
	}

	inner := max(1, a.width-2)
	return a.modalPartsFor(kind, inner), modalFullscreen, inner, available
}

// modalPartsFor builds and normalizes the content for one presentation.
func (a App) modalPartsFor(kind modalKind, width int) modalParts {
	var parts modalParts
	switch kind {
	case modalKindHelp:
		parts = a.helpModalParts(width)
	case modalKindInstallConfirm:
		parts = a.install.modalParts(width)
	case modalKindRestoreConfirm:
		parts = a.restore.modalParts(width)
	}
	return a.normalizeModalParts(parts, width)
}

// modalLines converts one content block to 7-bit data when requested and
// re-wraps it cell-aware. Wrapping (never truncation) keeps long paths,
// warnings and wide glyphs reachable at any width.
func (a App) modalLines(lines []string, width int) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		if a.ascii {
			line = toASCII(line)
		}
		if width <= 0 {
			out = append(out, line)
			continue
		}
		out = append(out, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	return out
}

func (a App) normalizeModalParts(p modalParts, width int) modalParts {
	p.Warning = a.modalLines(p.Warning, width)
	p.Policy = a.modalLines(p.Policy, width)
	p.Body = a.modalLines(p.Body, width)
	return p
}

// modalTextRows converts and wraps a title/subtitle/footer and applies one
// theme role per produced row.
func (a App) modalTextRows(role func(string) string, text string, width int) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if a.ascii {
		text = toASCII(text)
	}
	if width <= 0 {
		return []string{role(text)}
	}
	rows := Paragraph(text, width, 0)
	for i := range rows {
		rows[i] = role(rows[i])
	}
	return rows
}

func (a App) modalTitleRows(parts modalParts, width int) []string {
	return a.modalTextRows(func(s string) string { return a.theme.Title.Render(s) }, parts.Title, width)
}

func (a App) modalSubtitleRows(parts modalParts, width int) []string {
	return a.modalTextRows(func(s string) string { return a.theme.Subtitle.Render(s) }, parts.Subtitle, width)
}

func (a App) modalFooterRows(parts modalParts, width int) []string {
	return a.modalTextRows(func(s string) string { return a.theme.Hint.Render(s) }, parts.Footer, width)
}

// modalFixedRows returns the pinned rows before the body and the pinned
// footer block. Callers insert the paged body between them.
func (a App) modalFixedRows(parts modalParts, width int) (pre, footer []string) {
	pre = append(pre, a.modalTitleRows(parts, width)...)
	pre = append(pre, a.modalSubtitleRows(parts, width)...)
	addBlock(&pre, parts.Warning)
	addBlock(&pre, parts.Policy)
	return pre, a.modalFooterRows(parts, width)
}

func addBlock(rows *[]string, block []string) {
	if len(block) == 0 {
		return
	}
	if len(*rows) > 0 {
		*rows = append(*rows, "")
	}
	*rows = append(*rows, block...)
}

// modalFits reports whether the pinned title, warning, decision policy, key
// footer and at least one body row fit inside the box.
func (a App) modalFits(parts modalParts, width, maxBoxH int) bool {
	if ansi.StringWidth(modalPlainText(a.ascii, parts.Title)) > width {
		return false
	}
	pre, footer := a.modalFixedRows(parts, width)
	maxInner := maxBoxH - 2
	if maxInner < 1 {
		return false
	}
	used := len(pre) + len(footer)
	if len(parts.Body) == 0 {
		if len(pre) > 0 && len(footer) > 0 {
			used++
		}
		return used <= maxInner
	}
	if len(pre) > 0 {
		used++ // blank between pinned rows and body
	}
	used++ // at least one body row
	if len(footer) > 0 {
		used++ // blank between body and footer
	}
	return used <= maxInner
}

// modalBodyLimit is the paged body height inside a box of maxInner rows.
func (a App) modalBodyLimit(parts modalParts, width, maxInner int) int {
	pre, footer := a.modalFixedRows(parts, width)
	used := len(pre) + len(footer)
	if len(pre) > 0 {
		used++ // blank before body
	}
	if len(footer) > 0 {
		used++ // blank before footer
	}
	return max(0, maxInner-used)
}

// modalInnerRows assembles the box content around a paged body slice.
func (a App) modalInnerRows(parts modalParts, width int, body []string) []string {
	pre, footer := a.modalFixedRows(parts, width)
	rows := pre
	if len(body) > 0 {
		if len(rows) > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, body...)
	}
	if len(footer) > 0 {
		if len(rows) > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, footer...)
	}
	return rows
}

func (a App) modalOffset() int {
	if a.helpOpen {
		return a.helpScroll
	}
	return a.contentScroll
}

func (a App) visibleModalBody(parts modalParts, limit, offset int) []string {
	if limit <= 0 || len(parts.Body) == 0 {
		return nil
	}
	maxOffset := max(0, len(parts.Body)-limit)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	end := min(len(parts.Body), offset+limit)
	return parts.Body[offset:end]
}

// renderModal composes the active modal over the normal backdrop. Only the
// body area is replaced/framed; the root footer stays the active key footer.
func (a App) renderModal() string {
	parts, mode, width, maxBoxH := a.modalPresentation()
	if mode == modalNone {
		return a.renderBackdrop()
	}
	backdrop := strings.Split(a.renderBackdrop(), "\n")
	bodyStart := a.headerRowCount()
	footerRows := a.footerRowCount()
	bodyEnd := a.height - footerRows
	if bodyEnd > len(backdrop) {
		bodyEnd = len(backdrop)
	}
	if bodyStart > bodyEnd {
		bodyStart = bodyEnd
	}
	if mode == modalCentered {
		a.overlayCentered(backdrop, parts, width, maxBoxH, bodyStart, bodyEnd)
	} else {
		a.replaceFullscreen(backdrop, parts, width, bodyStart, bodyEnd)
	}
	return strings.Join(backdrop, "\n")
}

// overlayCentered dims the inactive backdrop and centers the framed dialog in
// the body area. Backdrop text is stripped before it is cut or restyled, so an
// underlying hyperlink, OSC 8 sequence or cursor control can never leak into
// the dialog or its footer, and escape bytes are never spliced by index.
func (a App) overlayCentered(backdrop []string, parts modalParts, width, maxBoxH, bodyStart, bodyEnd int) {
	maxInner := maxBoxH - 2
	limit := a.modalBodyLimit(parts, width, maxInner)
	body := a.visibleModalBody(parts, limit, a.modalOffset())
	rows := a.modalInnerRows(parts, width, body)
	box := a.frameModalRows(rows, width)
	if len(box) == 0 {
		return
	}

	x := (a.width - (width + 4)) / 2
	if x < 0 {
		x = 0
	}
	y := bodyStart + (bodyEnd-bodyStart-len(box))/2
	if y < bodyStart {
		y = bodyStart
	}
	if y+len(box) > bodyEnd {
		y = bodyEnd - len(box)
	}

	for i := range backdrop {
		plain := padCells(ansi.Strip(backdrop[i]), a.width)
		row := i - y
		if i >= bodyEnd {
			// The root footer stays live rather than dimmed.
			continue
		}
		if row < 0 || row >= len(box) {
			backdrop[i] = a.dimCells(plain)
			continue
		}
		left := padCells(ansi.Truncate(plain, x, ""), x)
		rightWidth := a.width - (x + width + 4)
		right := ""
		if rightWidth > 0 {
			right = ansi.Cut(plain, x+width+4, a.width)
			right = padCells(ansi.Truncate(right, rightWidth, ""), rightWidth)
		}
		backdrop[i] = a.dimCells(left) + box[row] + a.dimCells(right)
	}
}

func (a App) dimCells(s string) string {
	if s == "" {
		return ""
	}
	if a.theme.Color {
		return a.theme.Dim.Render(s)
	}
	return s
}

// frameModalRows surrounds already wrapped content rows with a one-cell
// padding and a bordered frame. Every row is exactly width+4 cells.
func (a App) frameModalRows(rows []string, width int) []string {
	if width <= 0 {
		return nil
	}
	glyphs := GlyphSet(a.ascii)
	rule := strings.Repeat(glyphs.DividerH, width+2)
	top := a.theme.BorderFocus.Render(glyphs.CornerTL + rule + glyphs.CornerTR)
	bottom := a.theme.BorderFocus.Render(glyphs.CornerBL + rule + glyphs.CornerBR)
	side := a.theme.BorderFocus.Render(glyphs.DividerV)

	out := make([]string, 0, len(rows)+2)
	out = append(out, top)
	for _, row := range rows {
		// Content is pre-wrapped; pad (never truncate) so the border is stable.
		out = append(out, side+" "+padCells(row, width)+" "+side)
	}
	out = append(out, bottom)
	return out
}

// replaceFullscreen fills the body area with the full-screen fallback: a
// pinned title and a scrollable body that still contains every warning and
// impact path. The normal footer remains the pinned key row.
func (a App) replaceFullscreen(backdrop []string, parts modalParts, width, bodyStart, bodyEnd int) {
	title := a.modalTitleRows(parts, width)
	content := a.fullscreenModalContent(parts, width)
	rows := max(0, bodyEnd-bodyStart-len(title))
	if rows < 1 && len(title) > 0 {
		title = title[:1]
		rows = max(0, bodyEnd-bodyStart-len(title))
	}
	offset := a.modalOffset()
	maxOffset := max(0, len(content)-rows)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	visible := content[offset:min(len(content), offset+rows)]

	// Dim the chrome above the content area without touching the live footer.
	for i := 0; i < bodyStart && i < len(backdrop); i++ {
		backdrop[i] = a.dimCells(padCells(ansi.Strip(backdrop[i]), a.width))
	}
	for i := bodyStart; i < bodyEnd && i < len(backdrop); i++ {
		backdrop[i] = ""
	}
	line := 0
	for _, row := range title {
		if bodyStart+line >= bodyEnd {
			break
		}
		backdrop[bodyStart+line] = " " + padCells(row, width)
		line++
	}
	for _, row := range visible {
		if bodyStart+line >= bodyEnd {
			break
		}
		backdrop[bodyStart+line] = " " + padCells(row, width)
		line++
	}
	for bodyStart+line < bodyEnd {
		backdrop[bodyStart+line] = ""
		line++
	}
}

// fullscreenModalContent is the scrollable fallback content: resize guidance,
// pinned warnings/policy and the full impact body. It is deterministic so the
// scroll clamp and the renderer agree.
func (a App) fullscreenModalContent(parts modalParts, width int) []string {
	var content []string
	for _, line := range wrapPath(a.modalGuidance(), width) {
		content = append(content, a.theme.Hint.Render(line))
	}
	addBlock(&content, a.modalSubtitleRows(parts, width))
	addBlock(&content, parts.Warning)
	addBlock(&content, parts.Policy)
	addBlock(&content, parts.Body)
	for len(content) > 0 && content[len(content)-1] == "" {
		content = content[:len(content)-1]
	}
	return content
}

func (a App) modalGuidance() string {
	text := "Full-screen dialog: the dialog does not fit the framed view; every line stays scrollable."
	switch {
	case a.width < 60:
		text = "Full-screen dialog: widen to at least 60 columns for the framed view; every line stays scrollable."
	case a.height < 16:
		text = "Full-screen dialog: resize to at least 16 rows for the framed view; every line stays scrollable."
	}
	if a.ascii {
		text = toASCII(text)
	}
	return text
}

// modalPlainText applies the session's 7-bit conversion for fit checks.
func modalPlainText(ascii bool, text string) string {
	if ascii {
		return toASCII(text)
	}
	return text
}

// modalActive reports whether renderViewport will compose a modal.
func (a App) modalActive() bool {
	_, ok := a.activeModalKind()
	return ok
}

// helpModalParts supplies the existing help reference to the compositor. The
// content is unchanged; only its framing and scrolling are new.
func (a App) helpModalParts(width int) modalParts {
	section := func(title string, rows []string) []string {
		out := []string{a.theme.Section.Render(title), ""}
		for _, row := range rows {
			out = append(out, wrapPath(row, width)...)
		}
		return out
	}

	var body []string
	add := func(block []string) {
		if len(block) == 0 {
			return
		}
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, block...)
	}

	add(section("Navigation", []string{
		"  Tab          Cycle navigation, categories (Explore), and content",
		"  ↑/↓/j/k      Move selection in navigation or catalog list",
		"  1 - 4        Direct jump to destination (1:Home, 2:Install, 3:Explore, 4:Help)",
		"  Enter        Open item or activate highlighted route",
		"  b / Esc      Back to list or return focus to navigation",
		"  Home / End   Jump to top or bottom of navigation / catalog",
		"  pgup / pgdn  Scroll content up / down",
	}))
	add(section("Catalog & Search", []string{
		"  /            Filter catalog items by ID, kind, origin, or client",
		"  c            Focus the category control",
		"  [ / ]        Previous / next kind or purpose category",
		"  Esc          Clear filter or exit filter mode",
	}))

	var safety []string
	for _, line := range wrapPath("  "+WritePolicyText, width) {
		safety = append(safety, a.theme.Warn.Render(line))
	}

	add(section("Session Settings", []string{
		"  m            Toggle logo reveal motion animation (on/off)",
		"  u            Toggle symbol set (Unicode icons vs 7-bit ASCII)",
		"  ?            Toggle this help screen",
		"  q / Ctrl+C   Quit when idle; Ctrl+C during a write exits after it",
	}))
	add([]string{a.theme.Hint.Render("Press ? or Esc to return to the previous screen.")})

	return modalParts{
		Title:    "Keyboard & Navigation Reference",
		Subtitle: "Contextual help · press ? or Esc to close",
		Warning:  section("Write Execution Safety", safety),
		Body:     body,
		Footer:   "? or Esc close · PgUp/PgDn scroll · q quit",
	}
}

// clampModalScroll clamps the active modal's scroll offset to the rows its
// current presentation actually exposes. It reports false when no modal owns
// the viewport.
func (a *App) clampModalScroll() bool {
	parts, mode, width, maxBoxH := a.modalPresentation()
	if mode == modalNone {
		return false
	}
	maxOffset := 0
	if mode == modalCentered {
		limit := a.modalBodyLimit(parts, width, maxBoxH-2)
		maxOffset = max(0, len(parts.Body)-limit)
	} else {
		content := a.fullscreenModalContent(parts, width)
		rows := max(0, a.height-a.headerRowCount()-a.footerRowCount()-len(a.modalTitleRows(parts, width)))
		maxOffset = max(0, len(content)-rows)
	}
	if a.helpOpen {
		a.helpScroll = max(0, min(a.helpScroll, maxOffset))
	} else {
		a.contentScroll = max(0, min(a.contentScroll, maxOffset))
	}
	return true
}
