package tui_test

// Feature 020 B6/T11 — navigating the fixed documentation destinations.
//
// Guide is the only entry point. The documentation rows are ordinary selectable
// rows: j/k or arrows highlight one, Enter opens exactly that trusted URL
// through the injected opener, and `g` keeps opening the README. No bare key is
// bound, so nothing here collides with `g` or the 1-5 route jumps. Every test
// injects a recording opener and a temporary home; no browser starts and no
// client directory is touched.

import (
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/appops"
)

// docsNewApp builds the root model with an injected opener and optional host.
// The catalog home is a temporary directory so no read reaches a real home.
func docsNewApp(t *testing.T, opener *tui.Opener, ops *tui.OpsHost) (tui.App, string) {
	t.Helper()
	home := t.TempDir()
	catalog := testCatalog()
	catalog.Home = home
	app := tui.NewAppWithScreens(
		tui.TUIConfig{Width: 160, Height: 100, Home: home, Interactive: true},
		catalog,
		tui.AppOptions{Ops: ops},
		tui.AppScreenOptions{
			UpdateManager: tui.NewUpdateCheckManager(tui.WithDisabled(true)),
			Opener:        opener,
		},
	)
	return app, home
}

func docsUpdate(t *testing.T, app tui.App, msg tea.Msg) (tui.App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(msg)
	updated, ok := next.(tui.App)
	if !ok {
		t.Fatalf("Update returned %T, want tui.App", next)
	}
	return updated, cmd
}

func docsDrain(t *testing.T, app tui.App, cmd tea.Cmd) tui.App {
	t.Helper()
	for _, msg := range executeCmd(cmd) {
		if msg == nil {
			continue
		}
		next, follow := docsUpdate(t, app, msg)
		app = docsDrain(t, next, follow)
	}
	return app
}

// docsEnterGuide activates the Guide route and requires the documentation
// section. A missing section is a contract failure, not a skip.
func docsEnterGuide(t *testing.T, app tui.App) tui.App {
	t.Helper()
	app, _ = docsUpdate(t, app, runes("5"))
	if app.ActiveRoute() != tui.RouteGuide {
		t.Fatalf("5 did not open Guide; route = %v", app.ActiveRoute())
	}
	if view := ansi.Strip(app.View().Content); !strings.Contains(view, "Documentation") {
		t.Fatalf("%s: Guide renders no Documentation section", docsPending)
	}
	return app
}

// docsHighlight moves the Guide cursor downward until documentation destination
// index is the highlighted row. Guide renders the literal URL only for the
// highlighted row, so the change of visible URL is the selection signal.
func docsHighlight(t *testing.T, app tui.App, index int) tui.App {
	t.Helper()
	if index < 0 || index >= len(docsDestinations) {
		t.Fatalf("documentation index %d out of range", index)
	}
	want := docsURLs()[index]
	for step := 0; step < 64; step++ {
		app, _ = docsUpdate(t, app, runes("j"))
		// The status notice also repeats a URL after an open, so selection is
		// read only from the Documentation section's fallback line.
		url, ok := docsVisible(strings.Join(docsSection(app.View().Content), "\n"))
		if !ok {
			continue
		}
		for i, known := range docsURLs() {
			if url != known {
				continue
			}
			if i == index {
				return app
			}
			if i > index {
				t.Fatalf("Guide skipped to documentation destination %d while looking for %d", i, index)
			}
		}
	}
	t.Fatalf("%s: Guide never highlighted documentation destination %d (%s)", docsPending, index, want)
	return app
}

func docsEnter(t *testing.T, app tui.App) tui.App {
	t.Helper()
	app, cmd := docsUpdate(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	return docsDrain(t, app, cmd)
}

// docsSection returns the rendered lines between the Documentation heading and
// the next section.
func docsSection(view string) []string {
	lines := strings.Split(ansi.Strip(view), "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, "Documentation") {
			start = i
			continue
		}
		if start >= 0 && strings.Contains(line, "Execution safety") {
			return lines[start:i]
		}
	}
	if start < 0 {
		return nil
	}
	return lines[start:]
}

// TestDocsEnterOpensHighlightedDestination proves Enter is the explicit action
// for a highlighted documentation row and that it opens exactly the trusted URL
// for that row, in registry order, without leaving Guide.
func TestDocsEnterOpensHighlightedDestination(t *testing.T) {
	rec := &docsLaunchRecorder{}
	app, _ := docsNewApp(t, docsDesktopOpener(rec), nil)
	app = docsEnterGuide(t, app)

	for i, url := range docsURLs() {
		app = docsHighlight(t, app, i)
		app = docsEnter(t, app)

		if rec.calls != i+1 {
			t.Fatalf("after Enter on destination %d launcher calls = %d, want %d", i, rec.calls, i+1)
		}
		if rec.urls[i] != url {
			t.Fatalf("Enter on destination %d opened %q, want %q", i, rec.urls[i], url)
		}
		if app.ActiveRoute() != tui.RouteGuide {
			t.Fatalf("opening documentation destination %d changed route to %v", i, app.ActiveRoute())
		}
		if app.NavIndex() != 4 {
			t.Fatalf("opening documentation destination %d changed NavIndex to %d", i, app.NavIndex())
		}
		notice := app.Notice()
		if !strings.Contains(notice, url) {
			t.Errorf("destination %d notice = %q, want the literal URL", i, notice)
		}
		if !strings.Contains(notice, "Opened") {
			t.Errorf("destination %d notice = %q, want the explicit success wording", i, notice)
		}
		if !strings.Contains(docsCompact(app.View().Content), url) {
			t.Errorf("destination %d fallback no longer shows %q after opening", i, url)
		}
	}
}

// TestDocsGActionStillOpensReadme proves `g` keeps its meaning from Home and
// from Guide, including while a documentation row is highlighted, and never
// opens the highlighted destination instead.
func TestDocsGActionStillOpensReadme(t *testing.T) {
	rec := &docsLaunchRecorder{}
	app, _ := docsNewApp(t, docsDesktopOpener(rec), nil)

	app, cmd := docsUpdate(t, app, runes("g"))
	app = docsDrain(t, app, cmd)
	if rec.calls != 1 || rec.urls[0] != tui.ReadmeURL {
		t.Fatalf("g on Home opened %v, want [%s]", rec.urls, tui.ReadmeURL)
	}
	if app.ActiveRoute() != tui.RouteHome {
		t.Fatalf("g on Home changed route to %v", app.ActiveRoute())
	}

	app = docsEnterGuide(t, app)
	app = docsHighlight(t, app, 3)
	app, cmd = docsUpdate(t, app, runes("g"))
	app = docsDrain(t, app, cmd)
	if rec.calls != 2 || rec.urls[1] != tui.ReadmeURL {
		t.Fatalf("g on Guide opened %v, want the README again", rec.urls)
	}
	if app.ActiveRoute() != tui.RouteGuide {
		t.Fatalf("g on Guide changed route to %v", app.ActiveRoute())
	}
	if strings.Contains(app.Notice(), docsURLs()[3]) {
		t.Fatalf("g opened the highlighted documentation row instead of the README: %q", app.Notice())
	}
}

// TestDocsExplicitOnlyAffordance proves the documentation rows have no bare
// key: neutral keys and the 1-5 route jumps retain their existing behavior and
// open nothing, Enter on a guided action opens nothing, and no documentation
// row advertises a key hint.
func TestDocsExplicitOnlyAffordance(t *testing.T) {
	rec := &docsLaunchRecorder{}
	app, _ := docsNewApp(t, docsDesktopOpener(rec), nil)
	app = docsEnterGuide(t, app)

	for _, k := range []string{
		"d", "h", "n", "p", "r", "s", "t", "v", "w", "x", "y", "z",
		"0", "6", "7", "8", "9", "[", "]", ".", ",", "/", "-", "+", "=",
	} {
		var cmd tea.Cmd
		app, cmd = docsUpdate(t, app, runes(k))
		app = docsDrain(t, app, cmd)
		if rec.calls != 0 {
			t.Fatalf("key %q opened %v; the documentation action must be explicit-only", k, rec.urls)
		}
	}

	routes := map[string]tui.Route{
		"1": tui.RouteHome,
		"2": tui.RouteInstall,
		"3": tui.RouteExplore,
		"4": tui.RouteActivity,
		"5": tui.RouteGuide,
	}
	for _, k := range []string{"1", "2", "3", "4", "5"} {
		var cmd tea.Cmd
		app, cmd = docsUpdate(t, app, runes(k))
		app = docsDrain(t, app, cmd)
		if rec.calls != 0 {
			t.Fatalf("route key %q opened %v", k, rec.urls)
		}
		if app.ActiveRoute() != routes[k] {
			t.Fatalf("route key %q reached %v, want %v", k, app.ActiveRoute(), routes[k])
		}
		if k != "5" {
			app, _ = docsUpdate(t, app, runes("5"))
		}
	}

	// No documentation row renders a [key] hint, so no bare key is advertised.
	section := docsSection(app.View().Content)
	if len(section) == 0 {
		t.Fatalf("%s: Guide lost the Documentation section", docsPending)
	}
	for _, line := range section {
		for _, dest := range docsDestinations {
			if strings.Contains(line, dest.Label) && strings.Contains(line, "[") {
				t.Fatalf("documentation row %q advertises a key hint: %q", dest.Label, line)
			}
		}
	}

	// Enter on the default guided action (Install or update) is not a link and
	// keeps its existing behavior.
	app, cmd := docsUpdate(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	app = docsDrain(t, app, cmd)
	if rec.calls != 0 {
		t.Fatalf("Enter on the default guided action opened %v", rec.urls)
	}
	if app.ActiveRoute() != tui.RouteInstall {
		t.Fatalf("Enter on the default guided action reached %v, want the Install route", app.ActiveRoute())
	}
}

// TestDocsFallbackNoticeKeepsURLWithoutSuccess proves the app-level notice for
// headless, SSH, unsupported-platform and failed opener environments keeps the
// literal URL, never claims success, and leaves Guide and its fallback intact.
func TestDocsFallbackNoticeKeepsURLWithoutSuccess(t *testing.T) {
	environments := []struct {
		name      string
		build     func(*docsLaunchRecorder) *tui.Opener
		wantCalls int
	}{
		{"headless", docsHeadlessOpener, 0},
		{"ssh", docsSSHOpener, 0},
		{"unsupported platform", docsUnsupportedOpener, 0},
		{"launcher failure", docsFailingOpener, 1},
	}

	for _, env := range environments {
		rec := &docsLaunchRecorder{}
		app, _ := docsNewApp(t, env.build(rec), nil)
		app = docsEnterGuide(t, app)
		app = docsHighlight(t, app, 0)
		app = docsEnter(t, app)

		url := docsURLs()[0]
		if rec.calls != env.wantCalls {
			t.Errorf("%s: launcher calls = %d, want %d", env.name, rec.calls, env.wantCalls)
		}
		notice := app.Notice()
		if !strings.Contains(notice, url) {
			t.Errorf("%s: notice = %q, want the literal URL", env.name, notice)
		}
		if strings.HasPrefix(notice, "Opened ") || strings.Contains(notice, "Opened "+url) {
			t.Errorf("%s: notice = %q, must not claim success", env.name, notice)
		}
		if !strings.Contains(strings.ToLower(notice), "manually") {
			t.Errorf("%s: notice = %q, want manual-opening guidance", env.name, notice)
		}
		if app.ActiveRoute() != tui.RouteGuide {
			t.Errorf("%s: opening changed route to %v", env.name, app.ActiveRoute())
		}
		if !strings.Contains(docsCompact(app.View().Content), url) {
			t.Errorf("%s: Guide no longer shows %q", env.name, url)
		}
	}
}

// docsDiscardLog is an in-memory write log seam. It never touches disk.
type docsDiscardLog struct{}

func (docsDiscardLog) Write(p []byte) (int, error) { return len(p), nil }
func (docsDiscardLog) Flush() error                { return nil }
func (docsDiscardLog) Close() error                { return nil }

// docsWriteCounters records whether any write seam ran.
type docsWriteCounters struct {
	install atomic.Int32
	restore atomic.Int32
	logs    atomic.Int32
}

func docsCountingOps(counters *docsWriteCounters) *tui.OpsHost {
	return tui.NewOpsHost(tui.OpsHostOptions{
		ApplyInstall: func(req appops.InstallRequest, token string, out io.Writer) (appops.ApplyResult, error) {
			counters.install.Add(1)
			return appops.ApplyResult{}, nil
		},
		ApplyRestore: func(req appops.RestoreRequest, token string, out io.Writer) (appops.ApplyResult, error) {
			counters.restore.Add(1)
			return appops.ApplyResult{}, nil
		},
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			counters.logs.Add(1)
			return docsDiscardLog{}, "/tmp/docs-navigation-test.log", nil
		},
	})
}

// TestDocsActionCannotTriggerWrite proves opening documentation destinations
// never enters a write path: no apply, restore or log seam runs, the temporary
// home is byte-for-byte unchanged, and the notice never reports a write.
func TestDocsActionCannotTriggerWrite(t *testing.T) {
	counters := &docsWriteCounters{}
	rec := &docsLaunchRecorder{}
	app, home := docsNewApp(t, docsDesktopOpener(rec), docsCountingOps(counters))
	before := snapshotTree(t, home)

	app = docsEnterGuide(t, app)
	for i := range docsDestinations {
		app = docsHighlight(t, app, i)
		app = docsEnter(t, app)
	}
	app = docsHighlight(t, app, len(docsDestinations)-1)
	app, cmd := docsUpdate(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	app = docsDrain(t, app, cmd)

	if got := counters.install.Load(); got != 0 {
		t.Errorf("install seam ran %d times", got)
	}
	if got := counters.restore.Load(); got != 0 {
		t.Errorf("restore seam ran %d times", got)
	}
	if got := counters.logs.Load(); got != 0 {
		t.Errorf("write log seam ran %d times", got)
	}
	if after := snapshotTree(t, home); !reflect.DeepEqual(before, after) {
		t.Errorf("opening documentation changed the home tree: before %v after %v", before, after)
	}
	if rec.calls != len(docsDestinations)+1 {
		t.Errorf("launcher calls = %d, want %d", rec.calls, len(docsDestinations)+1)
	}
	notice := strings.ToLower(app.Notice())
	for _, forbidden := range []string{"applied", "installed", "restored", "backup created"} {
		if strings.Contains(notice, forbidden) {
			t.Errorf("notice = %q, want no write result (%q)", app.Notice(), forbidden)
		}
	}
}

// TestDocsNarrowPlainRetainsSelectableURL proves a narrow, plain (ASCII,
// no-color) terminal still shows every fixed destination's literal URL with its
// label, so a terminal without OSC 8 hyperlinks keeps a copyable address.
func TestDocsNarrowPlainRetainsSelectableURL(t *testing.T) {
	for _, width := range []int{40, 60} {
		model := tui.NewGuideModel(tui.TUIConfig{Width: width, Height: 48, ASCII: true, Color: false, Interactive: false})
		var seen []string
		for step := 0; step < 64 && len(seen) < len(docsDestinations); step++ {
			view := model.View().Content
			if strings.Contains(view, "\x1b") {
				t.Fatalf("plain Guide emitted an ANSI sequence at width %d", width)
			}
			if url, ok := docsVisible(view); ok {
				if len(seen) == 0 || seen[len(seen)-1] != url {
					seen = append(seen, url)
					label := model.SelectedEntry().Title
					if strings.TrimSpace(label) == "" {
						t.Fatalf("width %d: highlighted documentation row has no label", width)
					}
					if !strings.Contains(docsCompact(view), strings.ReplaceAll(label, " ", "")) {
						t.Fatalf("width %d: highlighted row label %q is not visible", width, label)
					}
				}
			}
			next, _ := model.Update(runes("j"))
			model = next
		}
		want := docsURLs()
		if len(seen) != len(want) {
			t.Fatalf("%s: narrow/plain Guide exposed %d destinations at width %d, want %d", docsPending, len(seen), width, len(want))
		}
		for i := range want {
			if seen[i] != want[i] {
				t.Errorf("width %d: destination %d = %q, want %q", width, i, seen[i], want[i])
			}
		}
		if !strings.Contains(ansi.Strip(model.View().Content), "Documentation") {
			t.Errorf("width %d: plain Guide lost the Documentation heading", width)
		}
	}
}
