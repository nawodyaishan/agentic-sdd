package tui

// T19 tests for the bounded modal presentation. They live in the package so
// they can assert the compositor's geometry decisions directly, and they use
// only fake read-only operations: no real home, write, network or browser.

import (
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

func t19Key(r rune) tea.KeyPressMsg  { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func t19Code(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func t19Update(t *testing.T, app App, msg tea.Msg) (App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(msg)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want tui.App", next)
	}
	return updated, cmd
}

func t19Press(t *testing.T, app App, r rune) App {
	t.Helper()
	app, _ = t19Update(t, app, t19Key(r))
	return app
}

func t19PressCode(t *testing.T, app App, c rune) App {
	t.Helper()
	app, _ = t19Update(t, app, t19Code(c))
	return app
}

func t19Run(t *testing.T, app App, cmd tea.Cmd) App {
	t.Helper()
	if cmd == nil {
		return app
	}
	msg := cmd()
	if msg == nil {
		return app
	}
	app, _ = t19Update(t, app, msg)
	return app
}

func t19HelpApp(t *testing.T, cfg TUIConfig) App {
	t.Helper()
	cfg = cfg.Normalize()
	app := NewApp(cfg, Catalog{Home: cfg.Home})
	app = t19Press(t, app, '?')
	if !app.HelpOpen() {
		t.Fatal("? did not open help")
	}
	return app
}

func t19InstallPreview(path string) PreviewOp {
	return func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error) {
		return appops.PreviewResult{
			PreviewToken: "t19-install-token",
			Planned: []skillsync.ResultChange{{
				ID:     "t19-long-skill",
				Client: "claude",
				Path:   path,
				Action: skillsync.ChangeInstall,
			}},
		}, nil
	}
}

func t19OpenInstallConfirm(t *testing.T, cfg TUIConfig, path string) App {
	t.Helper()
	if cfg.Home == "" {
		cfg.Home = t.TempDir()
	}
	cfg = cfg.Normalize()
	app := NewAppWithScreens(cfg, Catalog{Home: cfg.Home}, AppOptions{}, AppScreenOptions{
		InstallPreview: t19InstallPreview(path),
	})
	app = t19Press(t, app, '2')                          // Install & Manage hub
	app = t19Press(t, app, 'i')                          // install journey
	app = t19PressCode(t, app, tea.KeyEnter)             // clients -> options
	app = t19PressCode(t, app, tea.KeyEnter)             // options -> review
	app, cmd := t19Update(t, app, t19Code(tea.KeyEnter)) // review -> preview
	app = t19Run(t, app, cmd)
	app = t19PressCode(t, app, tea.KeyEnter) // preview -> confirm
	if app.install.Step() != InstallStepConfirm {
		t.Fatalf("install step = %v, want confirm", app.install.Step())
	}
	return app
}

func t19RestoreOps(path string) RestoreOps {
	return RestoreOps{
		List: func(repo, home string) ([]skillsync.Backup, error) {
			return []skillsync.Backup{{
				ID:      "20260101T000000.000000000Z",
				Summary: "seed install",
			}}, nil
		},
		Preview: func(req appops.RestoreRequest) (appops.PreviewResult, error) {
			return appops.PreviewResult{
				Operation:    skillsync.OperationRestore,
				Outcome:      skillsync.OutcomePreview,
				PreviewToken: "t19-restore-token",
				Planned: []skillsync.ResultChange{{
					ID:     "t19-long-skill",
					Client: "claude",
					Path:   path,
					Action: skillsync.ChangeRestore,
				}},
			}, nil
		},
	}
}

func t19OpenRestoreConfirm(t *testing.T, cfg TUIConfig, path string) App {
	t.Helper()
	if cfg.Home == "" {
		cfg.Home = t.TempDir()
	}
	cfg = cfg.Normalize()
	app := NewAppWithScreens(cfg, Catalog{Home: cfg.Home}, AppOptions{}, AppScreenOptions{
		RestoreOps: t19RestoreOps(path),
	})
	app = t19Press(t, app, '2') // hub
	app, cmd := t19Update(t, app, t19Key('b'))
	app = t19Run(t, app, cmd)                // backup list
	app = t19PressCode(t, app, tea.KeyEnter) // details
	app, cmd = t19Update(t, app, t19Code(tea.KeyEnter))
	app = t19Run(t, app, cmd)                // preview
	app = t19PressCode(t, app, tea.KeyEnter) // confirm
	if app.restore.Stage() != RestoreStageConfirm {
		t.Fatalf("restore stage = %v, want confirm", app.restore.Stage())
	}
	return app
}

func t19Lines(view string) []string { return strings.Split(view, "\n") }

func t19AssertBounds(t *testing.T, label, view string, width, height int) {
	t.Helper()
	lines := t19Lines(view)
	if len(lines) != height {
		t.Fatalf("%s: rows = %d, want %d", label, len(lines), height)
	}
	for row, line := range lines {
		if got := ansi.StringWidth(line); got > width {
			t.Fatalf("%s: row %d width %d > %d: %q", label, row, got, width, ansi.Strip(line))
		}
	}
}

func t19ScrollAll(t *testing.T, app App, key rune, limit int) []string {
	t.Helper()
	var seen []string
	for i := 0; i < limit; i++ {
		seen = append(seen, app.View().Content)
		next, _ := t19Update(t, app, t19Code(key))
		if next.View().Content == app.View().Content {
			break
		}
		app = next
	}
	seen = append(seen, app.View().Content)
	return seen
}

// TestT19GeometryAndModes exercises every approved breakpoint for help,
// install confirmation and restore confirmation. It asserts exact bounds and
// that a centered dialog is only used when its pinned rows genuinely fit.
func TestT19GeometryAndModes(t *testing.T) {
	widths := []int{40, 60, 79, 80, 109, 110}
	heights := []int{8, 12, 16, 24}

	for _, w := range widths {
		for _, h := range heights {
			cfg := TUIConfig{Width: w, Height: h, Color: true, Home: "/tmp/t19-home"}
			app := t19HelpApp(t, cfg)
			_, mode, inner, maxBox := app.modalPresentation()
			if mode == modalNone {
				t.Fatalf("help %dx%d: no modal presentation", w, h)
			}
			if w < 60 || h < 16 {
				if mode != modalFullscreen {
					t.Fatalf("help %dx%d: mode %v, want full-screen fallback", w, h, mode)
				}
			}
			if mode == modalCentered {
				outer := inner + 4
				if outer > min(modalMaxOuterWidth, w-4) {
					t.Fatalf("help %dx%d: outer width %d exceeds cap", w, h, outer)
				}
				if maxBox > h-4 {
					t.Fatalf("help %dx%d: box height %d exceeds terminal height - 4", w, h, maxBox)
				}
			}
			t19AssertBounds(t, "help", app.View().Content, w, h)
			plain := ansi.Strip(app.View().Content)
			if !strings.Contains(plain, "Keyboard & Navigation Reference") {
				t.Fatalf("help %dx%d: title hidden", w, h)
			}
			if mode == modalCentered && strings.Contains(plain, "Full-screen dialog") {
				t.Fatalf("help %dx%d: centered dialog rendered fallback guidance", w, h)
			}
			if mode == modalFullscreen && !strings.Contains(plain, "Full-screen dialog") {
				t.Fatalf("help %dx%d: fallback missing guidance", w, h)
			}
		}
	}

	// Help fits centered at a normal 80x24 terminal and falls back below 16 rows.
	for _, w := range []int{60, 79, 80, 109, 110} {
		if _, mode, _, _ := t19HelpApp(t, TUIConfig{Width: w, Height: 24, Color: true}).modalPresentation(); mode != modalCentered {
			t.Fatalf("help %dx24 mode = %v, want centered", w, mode)
		}
	}
	if _, mode, _, _ := t19HelpApp(t, TUIConfig{Width: 80, Height: 12, Color: true}).modalPresentation(); mode != modalFullscreen {
		t.Fatalf("help 80x12 mode = %v, want full-screen", mode)
	}

	// Restore's shorter pinned decision fits centered at 80x24 and wider.
	for _, w := range []int{80, 109, 110} {
		r := t19OpenRestoreConfirm(t, TUIConfig{Width: w, Height: 24, Color: true}, "/tmp/t19-home/.claude/skills/t19")
		if _, mode, _, _ := r.modalPresentation(); mode != modalCentered {
			t.Fatalf("restore confirm %dx24 mode = %v, want centered", w, mode)
		}
		t19AssertBounds(t, "restore confirm", r.View().Content, w, 24)
	}

	// The install confirmation pins the full write-interruption policy; when
	// that cannot fit it must fall back rather than truncate.
	install := t19OpenInstallConfirm(t, TUIConfig{Width: 80, Height: 24, Color: true}, "/tmp/t19-home/.claude/skills/t19")
	if _, mode, _, _ := install.modalPresentation(); mode != modalFullscreen {
		t.Fatalf("install confirm 80x24 mode = %v, want full-screen fallback", mode)
	}
	if !strings.Contains(ansi.Strip(install.View().Content), "SIGKILL") {
		t.Fatal("install fallback hides the SIGKILL warning")
	}
	t19AssertBounds(t, "install confirm", install.View().Content, 80, 24)

	// A taller terminal fits the framed install confirmation.
	tall := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/t19-home/.claude/skills/t19")
	if _, mode, _, _ := tall.modalPresentation(); mode != modalCentered {
		t.Fatalf("install confirm 100x32 mode = %v, want centered", mode)
	}
	t19AssertBounds(t, "install confirm tall", tall.View().Content, 100, 32)
}

// TestT19CenteredBoxGeometry checks the exact frame, one-cell padding and
// horizontal centering of the composited dialog.
func TestT19CenteredBoxGeometry(t *testing.T) {
	const w, h = 80, 24
	app := t19HelpApp(t, TUIConfig{Width: w, Height: h, Color: true})
	lines := t19Lines(ansi.Strip(app.View().Content))

	top, bottom := -1, -1
	topBox, bottomBox := "", ""
	for i, line := range lines {
		if start := strings.Index(line, "┌"); start >= 0 {
			if end := strings.LastIndex(line, "┐"); end > start {
				top, topBox = i, line[start:end+len("┐")]
			}
		}
		if start := strings.Index(line, "└"); start >= 0 {
			if end := strings.LastIndex(line, "┘"); end > start {
				bottom, bottomBox = i, line[start:end+len("┘")]
			}
		}
	}
	if top < 0 || bottom <= top {
		t.Fatalf("centered frame not found: top=%d bottom=%d\n%s", top, bottom, strings.Join(lines, "\n"))
	}
	boxHeight := bottom - top + 1
	if boxHeight > h-4 {
		t.Fatalf("box height %d > terminal height - 4 (%d)", boxHeight, h-4)
	}
	outer := ansi.StringWidth(topBox)
	if ansi.StringWidth(bottomBox) != outer {
		t.Fatalf("frame top/bottom widths differ: %d vs %d", outer, ansi.StringWidth(bottomBox))
	}
	if outer > min(modalMaxOuterWidth, w-4) {
		t.Fatalf("box width %d exceeds cap", outer)
	}
	startByte := strings.Index(lines[top], "┌")
	left := ansi.StringWidth(lines[top][:startByte])
	right := w - left - outer
	if left < 1 || right < 1 || left-right > 1 || right-left > 1 {
		t.Fatalf("box not centered within %d columns: left=%d right=%d width=%d", w, left, right, outer)
	}
	// One-cell internal padding: "│ " before content and " │" after it.
	contentRow := ansi.Cut(lines[top+1], left, left+outer)
	if !strings.HasPrefix(contentRow, "│ ") {
		t.Fatalf("frame lacks one-cell left padding: %q", contentRow)
	}
	if !strings.HasSuffix(contentRow, " │") {
		t.Fatalf("frame lacks one-cell right padding: %q", contentRow)
	}
	if !strings.Contains(contentRow, "Keyboard & Navigation Reference") {
		t.Fatalf("title is not the first framed row: %q", contentRow)
	}
}

// TestT19FullscreenFallbackReachesAllContent pages the fallback and proves
// warnings, decisions, sections and long paths are never silently truncated.
func TestT19FullscreenFallbackReachesAllContent(t *testing.T) {
	help := t19HelpApp(t, TUIConfig{Width: 60, Height: 16, Color: true})
	seen := strings.Join(t19ScrollAll(t, help, tea.KeyPgDown, 60), "\n")
	plain := ansi.Strip(seen)
	for _, want := range []string{
		"Write Execution Safety",
		"SIGKILL",
		"Navigation",
		"Catalog & Search",
		"Session Preferences",
		"Press ? or Esc to return",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("help fallback never reached %q", want)
		}
	}

	longPath := "/tmp/t19-home/.claude/skills/very-long-t19-skill-directory/that/needs/wrapping"
	install := t19OpenInstallConfirm(t, TUIConfig{Width: 80, Height: 24, Color: true}, longPath)
	seen = strings.Join(t19ScrollAll(t, install, tea.KeyPgDown, 60), "\n")
	plain = ansi.Strip(seen)
	for _, want := range []string{
		"Pre-Execution Policy",
		"Default is No",
		"Target Configuration",
		"Target home: " + install.install.home,
		"Selected clients:",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("install fallback never reached %q:\n%s", want, plain)
		}
	}

	restore := t19OpenRestoreConfirm(t, TUIConfig{Width: 80, Height: 24, Color: true}, longPath)
	seen = strings.Join(t19ScrollAll(t, restore, tea.KeyPgDown, 60), "\n")
	plain = ansi.Strip(seen)
	for _, want := range []string{"Confirm restore", "Default: No", "Will restore a missing destination", "very-long-t19-skill-directory"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("restore confirm never reached %q:\n%s", want, plain)
		}
	}

	// Direct content check: wrapping never drops the long path from the parts.
	parts := install.install.modalParts(80)
	joined := ansi.Strip(strings.Join(parts.Body, ""))
	if !strings.Contains(joined, "Target home: ") {
		t.Fatalf("wrapped body lost the target home: %q", joined)
	}
}

// TestT19UnicodeBoundsAndASCII ensures wide/combining data and ASCII mode keep
// every row inside the terminal.
func TestT19UnicodeBoundsAndASCII(t *testing.T) {
	home := "/home/用户/résumé-e\u0301/very-long-路径-name"
	path := home + "/.claude/skills/t19-日本語-skill"

	for _, w := range []int{40, 60, 80, 110} {
		for _, h := range []int{8, 12, 16, 24} {
			app := t19OpenInstallConfirm(t, TUIConfig{Width: w, Height: h, Color: true, Home: home}, path)
			t19AssertBounds(t, "unicode install", app.View().Content, w, h)
			// Wrapping must preserve the whole home cell-by-cell.
			var joined strings.Builder
			for _, line := range app.install.modalParts(80).Body {
				joined.WriteString(strings.TrimSpace(ansi.Strip(line)))
			}
			if !strings.Contains(joined.String(), home) {
				t.Fatalf("%dx%d: wrapped body lost the Unicode home path: %q", w, h, joined.String())
			}
		}
	}

	// 7-bit session: the whole composed view (including the frame and footer)
	// must stay ASCII, and decoding must not drop content.
	asciiApp := t19OpenInstallConfirm(t, TUIConfig{Width: 60, Height: 24, Color: true, ASCII: true}, path)
	view := asciiApp.View().Content
	for _, r := range view {
		if r > 127 {
			t.Fatalf("ASCII confirm view contains non-7-bit rune %q", r)
		}
	}
	t19AssertBounds(t, "ascii install", view, 60, 24)
	if !strings.Contains(ansi.Strip(view), "Confirm Installation") {
		t.Fatal("ASCII confirm lost its title")
	}

	// Plain color-off output is unstyled and bounded.
	plainApp := t19OpenInstallConfirm(t, TUIConfig{Width: 60, Height: 24, Color: false}, path)
	plainView := plainApp.View().Content
	if strings.Contains(plainView, "\x1b") {
		t.Fatal("color-off modal output contains ANSI escapes")
	}
	t19AssertBounds(t, "plain install", plainView, 60, 24)
}

// TestT19BackdropIsANSIAndOSCSafe proves the inactive backdrop is stripped
// before composition, so links/OSC 8/cursor controls cannot leak, and that the
// dialog itself is not dimmed.
func TestT19BackdropIsANSIAndOSCSafe(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 32, Color: true}, t19ActivityCatalog())
	app = t19Press(t, app, '4') // Activity & Help, which contains OSC 8 links
	for i := 0; i < 20; i++ {
		app = t19PressCode(t, app, tea.KeyPgDown)
	}
	app = t19Press(t, app, '?')
	view := app.View().Content
	if strings.Contains(view, "\x1b]8;") || strings.Contains(view, "\x1b]") {
		t.Fatalf("underlying OSC/hyperlink leaked into the modal frame:\n%q", view)
	}
	for _, cursor := range []string{"\x1b[?25", "\x1b[?1049", "\x1b7", "\x1b8"} {
		if strings.Contains(view, cursor) {
			t.Fatalf("underlying cursor control %q leaked into the modal", cursor)
		}
	}
	title := app.theme.Title.Render("Keyboard & Navigation Reference")
	if !strings.Contains(view, title) {
		t.Fatal("modal title is not rendered with its own (undimmed) title role")
	}
	last := ansi.Strip(t19Lines(view)[len(t19Lines(view))-1])
	if strings.Contains(last, "\x1b]8") {
		t.Fatalf("modal footer leaked a hyperlink: %q", last)
	}
}

// t19ActivityCatalog provides enough content for the Activity screen,
// including its explicit hyperlinks.
func t19ActivityCatalog() Catalog {
	return Catalog{
		Home: "/tmp/t19-activity-home",
		Items: []CatalogItem{{
			ID:     "t19-activity-item",
			Name:   "T19 Activity Item",
			Kind:   skillsync.InventoryKindSkill,
			Origin: skillsync.InventoryOriginCore,
			Group:  "Core skills",
		}},
	}
}

// TestT19FocusAndScrollRestoration proves opening/closing help restores the
// underlying route scroll and always reopens at the top.
func TestT19FocusAndScrollRestoration(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 24, Color: true}, t19ActivityCatalog())
	app = t19Press(t, app, '4')
	for i := 0; i < 3; i++ {
		app = t19PressCode(t, app, tea.KeyPgDown)
	}
	before := app.contentScroll
	if before == 0 {
		t.Fatal("Activity did not scroll")
	}
	app = t19Press(t, app, '?')
	for i := 0; i < 5; i++ {
		app = t19PressCode(t, app, tea.KeyPgDown)
	}
	if app.helpScroll == 0 {
		t.Fatal("help modal did not scroll")
	}
	app = t19PressCode(t, app, tea.KeyEscape)
	if app.HelpOpen() {
		t.Fatal("Esc did not close help")
	}
	if app.helpScroll != 0 {
		t.Fatalf("helpScroll = %d after close, want 0", app.helpScroll)
	}
	if app.contentScroll != before {
		t.Fatalf("underlying contentScroll = %d after close, want %d", app.contentScroll, before)
	}
	app = t19Press(t, app, '?')
	if app.helpScroll != 0 {
		t.Fatalf("help reopened at scroll %d, want 0", app.helpScroll)
	}
}

// TestT19ResizeHelpAndBackgroundNeverConfirm proves none of the presentation
// events can produce consent, consume the token or advance the stage.
func TestT19ResizeHelpAndBackgroundNeverConfirm(t *testing.T) {
	app := t19OpenInstallConfirm(t, TUIConfig{Width: 80, Height: 24, Color: true}, "/tmp/t19-home/.claude/skills/t19")
	token := app.install.PreviewToken()
	if token == "" {
		t.Fatal("confirmation has no preview token")
	}

	// Resize.
	next, cmd := t19Update(t, app, tea.WindowSizeMsg{Width: 60, Height: 16})
	app = next
	if cmd != nil || app.install.Step() != InstallStepConfirm || app.install.PreviewToken() != token {
		t.Fatal("resize changed confirmation state/token or produced a command")
	}
	// Background reply.
	app, cmd = t19Update(t, app, tea.BackgroundColorMsg{})
	if cmd != nil || app.install.Step() != InstallStepConfirm || app.install.PreviewToken() != token {
		t.Fatal("background reply changed confirmation state/token")
	}
	// Help refresh open/close.
	app = t19Press(t, app, '?')
	if app.install.PreviewToken() != token {
		t.Fatal("opening help changed the preview token")
	}
	app = t19PressCode(t, app, tea.KeyEscape)
	if app.install.PreviewToken() != token || app.install.Step() != InstallStepConfirm {
		t.Fatal("closing help changed confirmation state/token")
	}
	// Enter stays inert.
	app, cmd = t19Update(t, app, t19Code(tea.KeyEnter))
	if cmd != nil || app.install.Step() != InstallStepConfirm || app.install.PreviewToken() != token {
		t.Fatal("Enter confirmed or changed the confirmation")
	}
	// Only the existing affirmative key starts a write.
	app, cmd = t19Update(t, app, t19Key('y'))
	if cmd == nil {
		t.Fatal("y did not produce a write command")
	}
	if app.install.Step() != InstallStepRunning {
		t.Fatalf("step after y = %v, want running", app.install.Step())
	}
	if _, ok := cmd().(StartWriteMsg); !ok {
		t.Fatal("y command did not produce a StartWriteMsg")
	}
	// A duplicate submit while running dispatches nothing.
	if _, dup := t19Update(t, app, t19Key('y')); dup != nil {
		t.Fatal("duplicate y produced another write command")
	}
}

// TestT19CancelAndStalePreviewKeepSafety proves cancel returns to the existing
// destination and a stale preview token requires a fresh preview.
func TestT19CancelAndStalePreviewKeepSafety(t *testing.T) {
	install := t19OpenInstallConfirm(t, TUIConfig{Width: 80, Height: 24, Color: true}, "/tmp/t19-home/.claude/skills/t19")
	cancelled, cmd := t19Update(t, install, t19Key('n'))
	if cmd != nil || cancelled.install.Step() != InstallStepPreview {
		t.Fatalf("install n: step=%v cmd nil=%t, want preview", cancelled.install.Step(), cmd == nil)
	}

	// A refused apply (changed inputs) returns to preview without a token.
	stale, _ := t19Update(t, install, t19Key('y'))
	stale.routeWriteMsg(WriteInstall, WriteResultMsg{Kind: WriteInstall, Err: appops.ErrNeedsPreview})
	if stale.install.Step() != InstallStepPreview {
		t.Fatalf("stale preview step = %v, want preview", stale.install.Step())
	}
	if stale.install.PreviewToken() != "" {
		t.Fatal("stale preview retained its token")
	}
	if _, cmd := t19Update(t, stale, t19Key('y')); cmd != nil {
		t.Fatal("y without a fresh preview produced a write")
	}

	restore := t19OpenRestoreConfirm(t, TUIConfig{Width: 80, Height: 24, Color: true}, "/tmp/t19-home/.claude/skills/t19")
	cancelledR, cmd := t19Update(t, restore, t19Key('n'))
	if cmd != nil || cancelledR.restore.Stage() != RestoreStagePreview {
		t.Fatalf("restore n: stage=%v cmd nil=%t, want preview", cancelledR.restore.Stage(), cmd == nil)
	}
	if cancelledR.restore.consentToken != "" {
		t.Fatal("restore cancel retained consent")
	}
}

// TestT19BusyPolicyIsNeverCovered proves an active write suppresses every
// modal composition, even if a help flag were somehow set.
func TestT19BusyPolicyIsNeverCovered(t *testing.T) {
	app := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 30, Color: true}, "/tmp/t19-home/.claude/skills/t19")
	app.write = &writeState{kind: WriteInstall, step: WriteStepStarting}
	app.helpOpen = true

	if app.modalActive() || func() bool { _, ok := app.activeModalKind(); return ok }() {
		t.Fatal("modal composed while a write owns the shell")
	}
	view := app.View().Content
	if strings.Contains(view, "Keyboard & Navigation Reference") {
		t.Fatal("help modal covered the active write policy")
	}
	last := t19Lines(view)[len(t19Lines(view))-1]
	if !strings.Contains(last, WriteBusyMarker) {
		t.Fatalf("busy policy missing from the last row: %q", ansi.Strip(last))
	}
	if strings.Contains(last, "q quit") {
		t.Fatalf("busy footer still offered q quit: %q", ansi.Strip(last))
	}
	// Input during the write cannot even open help.
	app.helpOpen = false
	app = t19Press(t, app, '?')
	if app.HelpOpen() {
		t.Fatal("? opened help during an active write")
	}
}
