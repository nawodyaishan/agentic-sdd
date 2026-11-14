package tui

// Feature 020 — App-level nil-opener documentation fallback.
//
// TestDocsNoOpenerAppRetainsURL drives the real App with a nil opener: it
// enters Guide, highlights the first fixed documentation destination, presses
// Enter and feeds the resulting command back through Update. The notice must
// retain the literal registry URL and must not claim the page opened. This
// covers the one branch the opener-level tests cannot reach — openLinkCmd with
// no opener at all — without starting any process. Guide renders the fallback
// line through DocsFallbackLine, so the same literal URL is also on screen.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// docsNoOpenerApp builds the root model and then removes the opener. The
// private field is reachable because this test lives in the implementation
// package; external callers can only supply a non-nil opener, which is why the
// external suite hand-builds OpenResult instead of driving this path.
func docsNoOpenerApp(t *testing.T) App {
	t.Helper()
	home := t.TempDir()
	mgr := NewUpdateCheckManager(WithDisabled(true))
	app := NewAppWithScreens(
		TUIConfig{Width: 160, Height: 100, Home: home, Interactive: true},
		Catalog{Home: home},
		AppOptions{},
		AppScreenOptions{UpdateManager: mgr},
	)
	app.opener = nil
	if app.opener != nil {
		t.Fatal("test setup did not produce a nil opener")
	}
	return app
}

// docsNoOpenerNav sends one navigation key and discards any command. Route
// switches may queue background work the test never needs; only the explicit
// documentation command below is executed.
func docsNoOpenerNav(t *testing.T, app App, msg tea.Msg) App {
	t.Helper()
	next, _ := app.Update(msg)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", next)
	}
	return updated
}

// docsNoOpenerCmd runs one command off the render path with a timeout so a
// blocking background command cannot hang the test.
func docsNoOpenerCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("documentation action returned no command")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatal("documentation command did not return")
		return nil
	}
}

func TestDocsNoOpenerAppRetainsURL(t *testing.T) {
	app := docsNoOpenerApp(t)

	app = docsNoOpenerNav(t, app, guideKey("5"))
	if app.ActiveRoute() != RouteGuide {
		t.Fatalf("5 did not open Guide; route = %v", app.ActiveRoute())
	}

	want := DocsLinks()[0]
	if !want.Trusted() {
		t.Fatalf("DocsLinks()[0] = %q is not a trusted registry destination", string(want))
	}

	highlighted := false
	for step := 0; step < 64; step++ {
		if app.guide.SelectedEntry().Link == want {
			highlighted = true
			break
		}
		app = docsNoOpenerNav(t, app, guideKey("j"))
	}
	if !highlighted {
		t.Fatalf("Guide never highlighted the first documentation destination %q", want.URL())
	}

	next, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", next)
	}
	app = updated
	msg := docsNoOpenerCmd(t, cmd)
	result, ok := msg.(OpenLinkResultMsg)
	if !ok {
		t.Fatalf("documentation command returned %T, want OpenLinkResultMsg", msg)
	}
	if result.Result.Opened {
		t.Fatalf("nil opener reported success opening %q", result.Result.Plan.URL)
	}
	if result.Result.Plan.URL != want.URL() {
		t.Fatalf("Plan.URL = %q, want %q", result.Result.Plan.URL, want.URL())
	}
	app = docsNoOpenerNav(t, app, result)

	if app.ActiveRoute() != RouteGuide {
		t.Fatalf("opening documentation with a nil opener left Guide; route = %v", app.ActiveRoute())
	}
	notice := app.Notice()
	if !strings.Contains(notice, want.URL()) {
		t.Fatalf("notice = %q, want the literal URL %q", notice, want.URL())
	}
	if strings.HasPrefix(notice, "Opened ") || strings.Contains(notice, "Opened "+want.URL()) {
		t.Fatalf("notice = %q, must not claim the page opened", notice)
	}
}
