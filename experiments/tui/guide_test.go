package tui

import (
	"os"
	"strings"
	"testing"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func guideKey(r string) tea.KeyPressMsg {
	if len(r) == 1 {
		return tea.KeyPressMsg{Code: rune(r[0]), Text: r}
	}
	return tea.KeyPressMsg{Text: r}
}

func guideApp(t *testing.T, opened *[]string) App {
	t.Helper()
	home := t.TempDir()
	opener := &Opener{
		GOOS: "linux",
		Getenv: func(k string) string {
			if k == "DISPLAY" {
				return ":0"
			}
			return ""
		},
		LookPath: func(string) (string, error) {
			return "/usr/bin/xdg-open", nil
		},
		Launch: func(program string, args []string) error {
			if opened != nil && len(args) > 0 {
				*opened = append(*opened, args[len(args)-1])
			}
			return nil
		},
	}
	mgr := NewUpdateCheckManager(WithDisabled(true))
	return NewAppWithScreens(
		TUIConfig{Width: 100, Height: 30, Home: home, Interactive: true},
		Catalog{Home: home},
		AppOptions{},
		AppScreenOptions{
			UpdateManager:  mgr,
			InstallPreview: fakePreviewOp(nil, "guide-preview-token", nil),
			RestoreOps: RestoreOps{
				List: func(repo, home string) ([]skillsync.Backup, error) {
					return nil, nil
				},
			},
			Opener: opener,
		},
	)
}

func TestGuidePresentationOnlySourceAudit(t *testing.T) {
	guideSrc, err := os.ReadFile("guide.go")
	if err != nil {
		t.Fatalf("read guide.go: %v", err)
	}
	for _, forbidden := range []string{
		"internal/skillsync",
		"internal/appops",
		"WriteRequest",
		"os/exec",
		"exec.Command",
		"net/http",
		"net.Dial",
		"syscall.Exec",
	} {
		if strings.Contains(string(guideSrc), forbidden) {
			t.Fatalf("guide.go contains forbidden reference %q", forbidden)
		}
	}

	hintFiles := []string{
		"app.go",
		"app_screens.go",
		"actions.go",
		"viewport.go",
		"activity.go",
		"onboarding.go",
		"install.go",
		"modal.go",
		"catalog.go",
		"library_help.go",
		"presentation.go",
		"mcp_screens.go",
		"guide.go",
	}
	staleCLI := []string{
		"agentic-sdd preview",
		"agentic-sdd apply",
		"agentic-sdd backups",
		"agentic-sdd restore",
		"agentic-sdd logs",
		"agentic-sdd version",
		"agentic-sdd guide",
		"agentic-sdd tui",
		"agentic-sdd help",
		"agentic-sdd fleet",
		"agentic-sdd mcp",
		"agentic-sdd opencode",
		"agentic-sdd native",
	}
	for _, file := range hintFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(data)
		for _, stale := range staleCLI {
			if strings.Contains(text, stale) {
				t.Errorf("%s still contains stale CLI example %q", file, stale)
			}
		}
	}
}

func TestGuideKey5AndKeyGAndHelpModal(t *testing.T) {
	var opened []string
	app := guideApp(t, &opened)

	// g on Home opens online README, NOT Guide.
	var cmd tea.Cmd
	app, cmd = b8Update(t, app, guideKey("g"))
	app = b8Drain(t, app, cmd)
	if app.ActiveRoute() != RouteHome {
		t.Fatalf("g on Home switched route to %v, want RouteHome", app.ActiveRoute())
	}
	if len(opened) != 1 || opened[0] != ReadmeURL {
		t.Fatalf("g on Home opened %v, want [%s]", opened, ReadmeURL)
	}

	// 5 opens Guide.
	app = contrastPress(t, app, guideKey("5"))
	if app.ActiveRoute() != RouteGuide {
		t.Fatalf("5 did not open RouteGuide; got %v", app.ActiveRoute())
	}
	if app.NavIndex() != 4 {
		t.Fatalf("NavIndex = %d, want 4", app.NavIndex())
	}

	// Guide exposes online README explicitly.
	view := ansi.Strip(app.View().Content)
	for _, want := range []string{"Guide", "Step-by-step setup & recovery", "Online guide / README", ReadmeURL} {
		if !strings.Contains(view, want) {
			t.Fatalf("Guide view missing %q:\n%s", want, view)
		}
	}

	// g on Guide keeps opening online README.
	app, cmd = b8Update(t, app, guideKey("g"))
	app = b8Drain(t, app, cmd)
	if app.ActiveRoute() != RouteGuide {
		t.Fatalf("g on Guide switched route to %v, want RouteGuide", app.ActiveRoute())
	}
	if len(opened) != 2 || opened[1] != ReadmeURL {
		t.Fatalf("g on Guide opened %v, want [%s, %s]", opened, ReadmeURL, ReadmeURL)
	}

	// Help modal lists 1 - 5 and Guide.
	app = contrastPress(t, app, guideKey("?"))
	helpView := ansi.Strip(app.View().Content)
	if !strings.Contains(helpView, "1 - 5") || !strings.Contains(helpView, "5:Guide") {
		t.Fatalf("help modal missing 1 - 5 / 5:Guide:\n%s", helpView)
	}
}

func TestGuideRoutesAndEscBReturnOrigin(t *testing.T) {
	// 1. Install or update -> InstallModel at client selection -> Esc/b returns to Guide.
	for _, backKey := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, guideKey("b")} {
		app := guideApp(t, nil)
		app = contrastPress(t, app, guideKey("5"))
		app = contrastPress(t, app, guideKey("i"))
		if app.ActiveRoute() != RouteInstall || app.installSection != installSectionInstall {
			t.Fatalf("Guide i did not open InstallModel: route=%v section=%v", app.ActiveRoute(), app.installSection)
		}
		if app.install.Step() != InstallStepClients {
			t.Fatalf("Guide install step = %v, want InstallStepClients", app.install.Step())
		}
		if !app.guideReturn {
			t.Fatal("guideReturn should be true after opening install from Guide")
		}

		app = contrastPress(t, app, backKey)
		if app.ActiveRoute() != RouteGuide {
			t.Fatalf("back (%s) from Guide-opened install went to %v, want RouteGuide", backKey.String(), app.ActiveRoute())
		}
		if app.guideReturn {
			t.Fatal("guideReturn was not cleared on return to Guide")
		}
		if app.installSection != installSectionHub {
			t.Fatalf("installSection = %v, want installSectionHub after returning to Guide", app.installSection)
		}
	}

	// 2. Restore a backup -> RestoreModel -> Esc/b at list returns to Guide.
	for _, backKey := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, guideKey("b")} {
		app := guideApp(t, nil)
		app = contrastPress(t, app, guideKey("5"))
		app = contrastPress(t, app, guideKey("b"))
		if app.ActiveRoute() != RouteInstall || app.installSection != installSectionBackups {
			t.Fatalf("Guide b did not open RestoreModel: route=%v section=%v", app.ActiveRoute(), app.installSection)
		}
		app = contrastPress(t, app, backKey)
		if app.ActiveRoute() != RouteGuide {
			t.Fatalf("back (%s) from Guide-opened restore went to %v, want RouteGuide", backKey.String(), app.ActiveRoute())
		}
	}

	// 3. View latest log -> ActivityModel (HelpSectionActivity) -> Esc/b returns to Guide.
	for _, backKey := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, guideKey("b")} {
		app := guideApp(t, nil)
		app = contrastPress(t, app, guideKey("5"))
		app = contrastPress(t, app, guideKey("l"))
		if app.ActiveRoute() != RouteActivity {
			t.Fatalf("Guide l did not open RouteActivity: got %v", app.ActiveRoute())
		}
		if app.activity.ActiveSection() != HelpSectionActivity {
			t.Fatalf("Guide l section = %v, want HelpSectionActivity", app.activity.ActiveSection())
		}
		app = contrastPress(t, app, backKey)
		if app.ActiveRoute() != RouteGuide {
			t.Fatalf("back (%s) from Guide-opened logs went to %v, want RouteGuide", backKey.String(), app.ActiveRoute())
		}
	}

	// 4. About / update check -> ActivityModel (HelpSectionAll) -> Esc/b returns to Guide.
	for _, backKey := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, guideKey("b")} {
		app := guideApp(t, nil)
		app = contrastPress(t, app, guideKey("5"))
		app = contrastPress(t, app, guideKey("a"))
		if app.ActiveRoute() != RouteActivity {
			t.Fatalf("Guide a did not open RouteActivity: got %v", app.ActiveRoute())
		}
		if app.activity.ActiveSection() != HelpSectionAll {
			t.Fatalf("Guide a section = %v, want HelpSectionAll", app.activity.ActiveSection())
		}
		app = contrastPress(t, app, backKey)
		if app.ActiveRoute() != RouteGuide {
			t.Fatalf("back (%s) from Guide-opened about went to %v, want RouteGuide", backKey.String(), app.ActiveRoute())
		}
	}

	// 5. Quick start tour -> OnboardingModel -> Esc anywhere and b on step 1 return to Guide.
	app := guideApp(t, nil)
	app = contrastPress(t, app, guideKey("5"))
	app = contrastPress(t, app, guideKey("o"))
	if app.onboarding == nil {
		t.Fatal("Guide o did not open OnboardingModel")
	}
	// Step forward then back with b stays in onboarding until step 1, then b returns to Guide.
	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if app.onboarding == nil || app.onboarding.Step() != OnboardingInstalled {
		t.Fatal("Enter did not advance to OnboardingInstalled")
	}
	app = contrastPress(t, app, guideKey("b"))
	if app.onboarding == nil || app.onboarding.Step() != OnboardingWelcome {
		t.Fatal("b on step 2 did not step back to OnboardingWelcome")
	}
	app = contrastPress(t, app, guideKey("b"))
	if app.onboarding != nil || app.ActiveRoute() != RouteGuide || app.guideReturn {
		t.Fatalf("b on step 1 did not return to Guide cleanly: onboarding=%v route=%v guideReturn=%v",
			app.onboarding != nil, app.ActiveRoute(), app.guideReturn)
	}

	// Esc closes onboarding back to Guide from any step.
	app = contrastPress(t, app, guideKey("o"))
	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyEscape})
	if app.onboarding != nil || app.ActiveRoute() != RouteGuide || app.guideReturn {
		t.Fatalf("Esc in onboarding did not return to Guide: onboarding=%v route=%v", app.onboarding != nil, app.ActiveRoute())
	}
}

func TestGuideReturnResetOnOrdinaryNavigationAndNonGuideUnchanged(t *testing.T) {
	app := guideApp(t, nil)
	// Open logs from Guide, then use ordinary route key 4 (or 1 then 4) -> guideReturn must clear.
	app = contrastPress(t, app, guideKey("5"))
	app = contrastPress(t, app, guideKey("l"))
	if !app.guideReturn {
		t.Fatal("expected guideReturn=true")
	}
	app = contrastPress(t, app, guideKey("4"))
	if app.guideReturn {
		t.Fatal("ordinary route jump 4 did not reset guideReturn")
	}
	// Esc on ordinary Activity focuses Nav, does not jump to Guide.
	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyEscape})
	if app.ActiveRoute() != RouteActivity || app.Focus() != FocusNav {
		t.Fatalf("ordinary Activity Esc got route=%v focus=%v, want RouteActivity+FocusNav", app.ActiveRoute(), app.Focus())
	}

	// Ordinary Backups via 2 -> b -> Esc returns to Install hub, not Guide.
	app = contrastPress(t, app, guideKey("2"))
	app = contrastPress(t, app, guideKey("b"))
	if app.installSection != installSectionBackups || app.guideReturn {
		t.Fatalf("ordinary Backups state: section=%v guideReturn=%v", app.installSection, app.guideReturn)
	}
	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyEscape})
	if app.ActiveRoute() != RouteInstall || app.installSection != installSectionHub {
		t.Fatalf("ordinary Backups Esc got route=%v section=%v, want RouteInstall+hub", app.ActiveRoute(), app.installSection)
	}
}

func TestInstallResetJourneyAndGuideCompletedResultReset(t *testing.T) {
	// Unit test on InstallModel.ResetJourney().
	m := NewInstallModel(TUIConfig{Home: "/tmp/home", Repo: "/tmp/repo", Width: 90, Height: 28})
	m.step = InstallStepResult
	m.outcome = InstallOutcomeApplied
	m.previewToken = "stale-token"
	m.previewResult = &appops.PreviewResult{PreviewToken: "stale-token"}
	m.applyResult = &appops.ApplyResult{Outcome: skillsync.OutcomeApplied, BackupID: "bkp-1"}
	m.withOpenCode = true
	m.freeFirst = true
	m.confirmNoSelected = false
	m.notice = "previous notice"
	m.logPath = "/tmp/home/.agentic-sdd/logs/last.log"
	oldOpID := m.opID

	m.ResetJourney()
	if m.Step() != InstallStepClients {
		t.Fatalf("Step after ResetJourney = %v, want InstallStepClients", m.Step())
	}
	if m.Outcome() != InstallOutcomeNone {
		t.Fatalf("Outcome after ResetJourney = %v, want InstallOutcomeNone", m.Outcome())
	}
	if m.PreviewToken() != "" || m.PreviewResult() != nil || m.ApplyResult() != nil {
		t.Fatal("ResetJourney left stale preview/apply state")
	}
	if m.WithOpenCode() || m.OpenCodeFreeFirst() || !m.ConfirmNoSelected() {
		t.Fatal("ResetJourney did not restore default options and safe default-No")
	}
	if m.Notice() != "" || m.logPath != "" {
		t.Fatal("ResetJourney left stale notice or logPath")
	}
	if m.opID.Generation <= oldOpID.Generation {
		t.Fatal("ResetJourney did not advance opID generation")
	}

	// Late preview from old generation is discarded.
	updated, _ := m.Update(InstallPreviewMsg{
		OpID:   oldOpID,
		Result: appops.PreviewResult{PreviewToken: "late-token"},
	})
	m = updated.(InstallModel)
	if m.PreviewToken() != "" {
		t.Fatal("stale InstallPreviewMsg resurrected preview token after ResetJourney")
	}

	// Root App test: completed install cannot be resumed when opened from Guide.
	app := guideApp(t, nil)
	app.install.step = InstallStepResult
	app.install.outcome = InstallOutcomeApplied
	app.install.previewToken = "completed-token"
	app.install.applyResult = &appops.ApplyResult{Outcome: skillsync.OutcomeApplied}

	app = contrastPress(t, app, guideKey("5"))
	app = contrastPress(t, app, guideKey("i"))
	if app.ActiveRoute() != RouteInstall || app.installSection != installSectionInstall {
		t.Fatalf("Guide i did not open InstallModel: route=%v section=%v", app.ActiveRoute(), app.installSection)
	}
	if app.install.Step() != InstallStepClients || app.install.Outcome() != InstallOutcomeNone {
		t.Fatalf("Guide install resumed completed result: step=%v outcome=%v", app.install.Step(), app.install.Outcome())
	}
	if app.install.PreviewToken() != "" || app.install.ApplyResult() != nil {
		t.Fatal("Guide install retained completed install preview/apply state")
	}
	view := ansi.Strip(app.View().Content)
	if !strings.Contains(view, "Client Target Selection") {
		t.Fatalf("Guide install did not render client selection:\n%s", view)
	}
}

func TestGuidePlainASCIIAndNarrowLayoutBounds(t *testing.T) {
	for _, width := range []int{40, 60, 79, 80, 110} {
		for _, ascii := range []bool{false, true} {
			m := NewGuideModel(TUIConfig{Width: width, Height: 24, Color: false, ASCII: ascii})
			lines := m.Lines(width)
			if len(lines) == 0 {
				t.Fatalf("width %d ascii %t: empty lines", width, ascii)
			}
			for i, line := range lines {
				if ansi.Strip(line) != line {
					t.Fatalf("width %d line %d contains ANSI in plain mode: %q", width, i, line)
				}
				if w := ansi.StringWidth(line); w > width {
					t.Fatalf("width %d ascii %t line %d exceeds width (%d > %d): %q", width, ascii, i, w, width, line)
				}
				if ascii {
					for _, r := range line {
						if r > 127 {
							t.Fatalf("width %d line %d non-7-bit rune %q: %q", width, i, r, line)
						}
					}
				}
			}
		}
	}
}

func TestRouteScrollFiveRoutesIndependent(t *testing.T) {
	app := guideApp(t, nil)
	app.height = 16
	app = contrastPress(t, app, guideKey("5"))
	initialView := app.View().Content
	for i := 0; i < 3; i++ {
		app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyPgDown})
	}
	scrolledView := app.View().Content
	if scrolledView == initialView {
		t.Fatal("PgDown on Guide at height 16 did not scroll content")
	}
	app = contrastPress(t, app, guideKey("4"))
	if app.contentScroll != 0 {
		t.Fatalf("Activity started with non-zero scroll %d", app.contentScroll)
	}
	app = contrastPress(t, app, guideKey("5"))
	if app.View().Content != scrolledView {
		t.Fatal("returning to Guide lost its scroll position")
	}
}
