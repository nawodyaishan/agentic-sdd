package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestT30HelpSessionPreferencesReachable(t *testing.T) {
	cat := contrastCatalog()
	cfg := TUIConfig{Width: 80, Height: 24, Color: true, Motion: true, ASCII: false, Interactive: true}
	app := NewApp(cfg, cat)

	// Open help modal by pressing ?
	updated, _ := app.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	app = updated.(App)
	if !app.HelpOpen() {
		t.Fatal("expected Help to be open after ?")
	}

	// Verify Help modal content contains Session Preferences
	parts, mode, _, _ := app.modalPresentation()
	if mode == modalNone {
		t.Fatal("expected modalPresentation mode to be active")
	}
	joined := strings.Join(parts.Body, "\n")
	if !strings.Contains(joined, "Session Preferences") {
		t.Fatalf("Help modal missing Session Preferences section:\n%s", joined)
	}
	if !strings.Contains(joined, "Toggle motion") || !strings.Contains(joined, "Toggle symbol") || !strings.Contains(joined, "Toggle update check") {
		t.Fatalf("Help modal missing motion/symbol/offline preference lines:\n%s", joined)
	}

	// Press m inside Help modal to toggle motion
	updated, _ = app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	app = updated.(App)
	if app.Motion() {
		t.Fatal("expected motion to be false after pressing m in Help")
	}

	// Press u inside Help modal to toggle ASCII/Unicode
	updated, _ = app.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	app = updated.(App)
	if !app.ASCII() {
		t.Fatal("expected ASCII to be true after pressing u in Help")
	}

	// Press d inside Help modal to toggle update checks (offline mode)
	if app.updateMgr == nil {
		t.Fatal("expected updateMgr to be non-nil")
	}
	wasDisabled := app.updateMgr.IsDisabled()
	updated, _ = app.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	app = updated.(App)
	if app.updateMgr.IsDisabled() == wasDisabled {
		t.Fatal("expected updateMgr disabled state to toggle after pressing d in Help")
	}

	// Close Help with Esc
	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)
	if app.HelpOpen() {
		t.Fatal("expected Help to be closed after Esc")
	}
	// Settings persist after closing Help
	if app.Motion() {
		t.Fatal("motion setting should persist after closing Help")
	}
	if !app.ASCII() {
		t.Fatal("ASCII setting should persist after closing Help")
	}
}

func TestT30RestrainedHeaderStatusCluster(t *testing.T) {
	cat := contrastCatalog()
	cfg := TUIConfig{Width: 100, Height: 24, Color: true}
	app := NewApp(cfg, cat)

	// Routine idle/current states must NOT produce permanent clutter in the top bar header.
	cluster := app.StatusCluster()
	if cluster != "" {
		t.Fatalf("idle/routine status cluster should be empty, got %q", cluster)
	}

	// Simulate checking state: concise notice
	app.updateMgr.state = UpdateStatusChecking
	cluster = app.StatusCluster()
	if !strings.Contains(strings.ToLower(cluster), "checking") {
		t.Fatalf("expected concise checking notice, got %q", cluster)
	}

	// Simulate newer version available: actionable notice
	app.updateMgr.state = UpdateStatusNewer
	cluster = app.StatusCluster()
	if !strings.Contains(strings.ToLower(cluster), "update") {
		t.Fatalf("expected actionable newer version status, got %q", cluster)
	}

	// Routine unavailable or disabled state must not clutter header
	app.updateMgr.state = UpdateStatusUnavailable
	if c := app.StatusCluster(); c != "" {
		t.Fatalf("unavailable state should not clutter header, got %q", c)
	}

	app.updateMgr.state = UpdateStatusDisabled
	if c := app.StatusCluster(); c != "" {
		t.Fatalf("disabled state should not clutter header, got %q", c)
	}
}

func TestT30NeutralNoticeRoleContrast(t *testing.T) {
	theme := NewTheme(true)
	canvas := contrastRGB(t, theme.CanvasColor())
	noticeRGB := contrastRGB(t, theme.Notice.GetForeground())

	ratio := contrastRatio(noticeRGB, canvas)
	if ratio < 7.0 {
		t.Fatalf("Notice role contrast %.2f:1, want >= 7.0:1 AAA", ratio)
	}

	// Notice must be neutral (Text color), not Warning (yellow)
	textRGB := contrastRGB(t, theme.Text.GetForeground())
	warnRGB := contrastRGB(t, theme.Warn.GetForeground())
	if noticeRGB == warnRGB {
		t.Fatal("Notice role should not use Warning color (must be neutral)")
	}
	if noticeRGB != textRGB {
		t.Fatalf("Notice role = %v, want Text role %v", noticeRGB, textRGB)
	}
}

// TestActivityPreferencesReportActualSessionState pins both states of the
// motion/ASCII/checks controls instead of the previous hardcoded "off /
// Unicode / online" text.
func TestActivityPreferencesReportActualSessionState(t *testing.T) {
	cases := []struct {
		name             string
		motion, ascii    bool
		disableChecks    bool
		wantMotion       string
		wantSymbol       string
		wantChecks       string
		notWantSubstring string
	}{
		{"defaults", false, false, false, "motion off", "Unicode", "checks enabled", "checks disabled"},
		{"toggled", true, true, true, "motion on", "ASCII", "checks disabled", "checks enabled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewActivityModel(ActivityConfig{
				Theme:    NewTheme(false),
				Width:    100,
				Metadata: AboutMetadata{Version: "dev-feature-013", Commit: "1234567", Date: "2026-11-11", Author: BrandAuthor},
			})
			m.SelectSection(HelpSectionPreferences)
			m.SetSessionPreferences(tc.motion, tc.ascii)
			if tc.disableChecks {
				m.updateManager.Disable()
			}
			text := strings.Join(m.Lines(100), "\n")
			for _, want := range []string{tc.wantMotion, tc.wantSymbol, tc.wantChecks} {
				if !strings.Contains(text, want) {
					t.Errorf("preferences missing %q:\n%s", want, text)
				}
			}
			if strings.Contains(text, tc.notWantSubstring) {
				t.Errorf("preferences still show %q:\n%s", tc.notWantSubstring, text)
			}
			if strings.Contains(strings.ToLower(text), "online") {
				t.Errorf("preferences still infer connectivity as online:\n%s", text)
			}
		})
	}
}

// TestAppActivityPreferencesFollowRootToggles proves the Activity route reads
// the root-owned settings rather than a stale local copy.
func TestAppActivityPreferencesFollowRootToggles(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 30, Color: false}, contrastCatalog())
	app.activity.SelectSection(HelpSectionPreferences)

	text := strings.Join(app.activityRouteLines(100), "\n")
	if !strings.Contains(text, "motion off") || !strings.Contains(text, "Unicode") || !strings.Contains(text, "checks enabled") {
		t.Fatalf("initial preferences do not reflect root state:\n%s", text)
	}

	app = contrastPress(t, app, tea.KeyPressMsg{Code: 'm', Text: "m"})
	app = contrastPress(t, app, tea.KeyPressMsg{Code: 'u', Text: "u"})
	app.toggleUpdateChecks()

	text = strings.Join(app.activityRouteLines(100), "\n")
	if !strings.Contains(text, "motion on") || !strings.Contains(text, "ASCII") || !strings.Contains(text, "checks disabled") {
		t.Fatalf("toggled preferences do not reflect root state:\n%s", text)
	}
	if strings.Contains(text, "checks enabled") {
		t.Fatalf("toggled preferences still claim checks enabled:\n%s", text)
	}
}
