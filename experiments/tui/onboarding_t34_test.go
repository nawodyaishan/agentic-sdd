package tui

// B12/T34 coverage for the optional in-session Quick start. These tests keep
// every seam fake: no real home, clipboard, opener, browser, client or agent is
// involved, and every asserted command is an explicit user action.

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// t34Setup drives the install journey to a non-default selection so the tests
// can prove the tour never resets or reuses those choices.
func t34Setup(t *testing.T) InstallModel {
	t.Helper()
	cfg := TUIConfig{Home: "/tmp/t34-home", Repo: "/tmp/t34-repo", Width: 90, Height: 28}
	model := NewInstallModel(cfg)
	model, _ = pressKey(model, " ") // deselect the first canonical client
	model, _ = pressKey(model, "enter")
	model, _ = pressKey(model, " ") // OpenCode agent profiles
	model, _ = pressKey(model, "j")
	model, _ = pressKey(model, " ") // free-model template preference
	return model
}

func t34Update(t *testing.T, model OnboardingModel, msg tea.Msg) (OnboardingModel, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(msg)
	next, ok := updated.(OnboardingModel)
	if !ok {
		t.Fatalf("Update returned %T, want OnboardingModel", updated)
	}
	return next, cmd
}

func t34Key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func t34Runes(s string) tea.KeyPressMsg {
	if len(s) == 1 {
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
	return tea.KeyPressMsg{Text: s}
}

func t34Next(t *testing.T, model OnboardingModel) OnboardingModel {
	t.Helper()
	next, _ := t34Update(t, model, t34Key(tea.KeyEnter))
	return next
}

func t34Flat(text string) string { return strings.Join(strings.Fields(text), " ") }

// TestT34SetupSummaryUsesCurrentChoices proves the tour reviews the current
// in-session selection rather than the catalog's default client list, and that
// building the summary costs no writes or commands.
func TestT34SetupSummaryUsesCurrentChoices(t *testing.T) {
	install := t34Setup(t)
	want := install.SelectedClients()
	if len(want) == 0 {
		t.Fatal("test setup left no selected clients")
	}

	summary := NewOnboardingSummaryFromSetup(contrastCatalog(), install)
	if got := strings.Join(summary.Clients, ","); got != strings.Join(want, ",") {
		t.Fatalf("summary clients = %v, want the current selection %v", summary.Clients, want)
	}
	if !summary.WithOpenCode || !summary.OpenCodeFreeFirst {
		t.Fatalf("summary optional content = pack %v free-first %v, want both selected", summary.WithOpenCode, summary.OpenCodeFreeFirst)
	}

	// The summary is a copy: later installer edits cannot mutate the tour.
	install, _ = pressKey(install, " ") // toggle the pack off
	if !summary.WithOpenCode {
		t.Fatal("editing installer options mutated the tour summary")
	}
}

// TestT34SkipReopenBackPreserveChoicesAndConsent proves skip, back and reopen
// keep the same review data and never create consent or revive a stale preview
// after the options changed.
func TestT34SkipReopenBackPreserveChoicesAndConsent(t *testing.T) {
	install := t34Setup(t)

	// Produce a preview, return to options and change them so any token or
	// confirmation from before is stale by construction.
	install.SetPreviewOp(fakePreviewOp(nil, "t34-stale-token", nil))
	install, _ = pressKey(install, "enter")    // review
	install, cmd := pressKey(install, "enter") // preview
	if cmd == nil {
		t.Fatal("review Enter emitted no preview command")
	}
	updated, _ := install.Update(cmd())
	install = updated.(InstallModel)
	if install.PreviewToken() != "t34-stale-token" {
		t.Fatalf("preview token = %q, want the fake token", install.PreviewToken())
	}
	install, _ = pressKey(install, "b") // preview -> review
	install, _ = pressKey(install, "b") // review -> options
	install, _ = pressKey(install, " ") // toggle options: invalidates the preview
	if install.PreviewToken() != "" {
		t.Fatal("changing options kept a stale preview token")
	}

	wantClients := strings.Join(install.SelectedClients(), ",")
	wantOpenCode, wantFreeFirst := install.WithOpenCode(), install.OpenCodeFreeFirst()
	summary := NewOnboardingSummaryFromSetup(contrastCatalog(), install)

	cfg := TUIConfig{Width: 90, Height: 28}
	tour := NewOnboardingModel(cfg, summary)
	if cmd := tour.Init(); cmd != nil {
		t.Fatal("Quick start Init() emitted a command")
	}

	// Forward to the finish step, then back: the review data is unchanged and
	// the completed outcome is cleared rather than left stale.
	tour = t34Next(t, tour)
	tour = t34Next(t, tour)
	tour = t34Next(t, tour)
	if !tour.Finished() {
		t.Fatal("tour did not reach the explicit finish step")
	}
	tour, cmd = t34Update(t, tour, t34Key(tea.KeyLeft))
	if cmd != nil {
		t.Fatalf("back emitted %T, want no command", cmd)
	}
	if tour.Finished() || tour.Outcome() != OnboardingActive {
		t.Fatalf("back from finish left outcome %v / Finished %v, want active", tour.Outcome(), tour.Finished())
	}
	if got := strings.Join(tour.Summary().Clients, ","); got != wantClients {
		t.Fatalf("back changed summary clients to %v, want %q", tour.Summary().Clients, wantClients)
	}

	// Skip is terminal and re-opening starts a fresh optional tour with the
	// same data; neither touches installer choices or consent.
	skipped, cmd := t34Update(t, tour, t34Runes("s"))
	if cmd != nil {
		t.Fatalf("skip emitted %T, want no command", cmd)
	}
	if !skipped.Done() {
		t.Fatal("skip did not reach a terminal outcome")
	}
	reopened := skipped.Reopen()
	if reopened.Step() != OnboardingWelcome || reopened.Outcome() != OnboardingActive || reopened.Done() {
		t.Fatalf("Reopen() = step %v outcome %v done %v, want a fresh active tour", reopened.Step(), reopened.Outcome(), reopened.Done())
	}
	if got := strings.Join(reopened.Summary().Clients, ","); got != wantClients {
		t.Fatalf("reopen changed summary clients to %v, want %q", reopened.Summary().Clients, wantClients)
	}

	// Cancel and re-open once more from the finish step's host-equivalent path.
	cancelled, _ := t34Update(t, reopened, t34Key(tea.KeyEscape))
	if !cancelled.Done() {
		t.Fatal("cancel did not reach a terminal outcome")
	}
	final := cancelled.Reopen()
	if final.Summary().WithOpenCode != wantOpenCode || final.Summary().OpenCodeFreeFirst != wantFreeFirst {
		t.Fatalf("reopen optional content = %v/%v, want %v/%v", final.Summary().WithOpenCode, final.Summary().OpenCodeFreeFirst, wantOpenCode, wantFreeFirst)
	}

	// Nothing in the tour touched the installer: no token, no confirmation and
	// the same selection remains.
	if install.PreviewToken() != "" {
		t.Fatalf("tour revived preview token %q", install.PreviewToken())
	}
	if !install.ConfirmNoSelected() {
		t.Fatal("tour changed the default-No confirmation state")
	}
	if got := strings.Join(install.SelectedClients(), ","); got != wantClients {
		t.Fatalf("tour changed selected clients to %q, want %q", got, wantClients)
	}
}

// TestT34ReopenFromHomePreservesInstallChoices proves the real root shell can
// skip, re-open and cancel the Quick start from Home without resetting the
// install journey's selection or creating consent.
func TestT34ReopenFromHomePreservesInstallChoices(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 30, Interactive: true}, contrastCatalog())

	// Build a non-default selection through the real key path.
	app = contrastPress(t, app, t34Runes("2"))        // Install & Manage
	app = contrastPress(t, app, t34Key(tea.KeyEnter)) // open the install journey
	app = contrastPress(t, app, t34Runes(" "))        // deselect the first client
	app = contrastPress(t, app, t34Key(tea.KeyEnter)) // options
	app = contrastPress(t, app, t34Runes(" "))        // OpenCode agent profiles
	app = contrastPress(t, app, t34Runes("j"))        // free-first row
	app = contrastPress(t, app, t34Runes(" "))        // free-first preference
	wantClients := strings.Join(app.install.SelectedClients(), ",")
	wantOpenCode, wantFreeFirst := app.install.WithOpenCode(), app.install.OpenCodeFreeFirst()

	app = contrastPress(t, app, t34Runes("1")) // Home
	app = contrastPress(t, app, t34Runes("o")) // open the optional Quick start
	if !strings.Contains(t34Flat(app.View().Content), "step 1 of 4") {
		t.Fatalf("o did not open the Quick start:\n%s", app.View().Content)
	}
	app = contrastPress(t, app, t34Runes("s")) // skip
	if !strings.Contains(app.Notice(), "skipped") {
		t.Fatalf("skip notice = %q, want skipped", app.Notice())
	}
	app = contrastPress(t, app, t34Runes("o")) // re-open in the same session
	if !strings.Contains(t34Flat(app.View().Content), "step 1 of 4") {
		t.Fatalf("re-opening the Quick start failed:\n%s", app.View().Content)
	}

	// The review step shows the in-session selection, not a catalog default.
	app = contrastPress(t, app, t34Key(tea.KeyEnter))
	app = contrastPress(t, app, t34Key(tea.KeyEnter))
	review := t34Flat(app.View().Content)
	wantPlainClients := clientPlainNames(strings.Split(wantClients, ","))
	if !strings.Contains(review, "Clients: "+wantPlainClients+".") {
		t.Fatalf("Quick start review does not show the current selection %q:\n%s", wantPlainClients, app.View().Content)
	}

	app = contrastPress(t, app, t34Key(tea.KeyEscape)) // cancel
	if app.install.PreviewToken() != "" || !app.install.ConfirmNoSelected() {
		t.Fatal("Quick start created consent or a preview token")
	}
	if got := strings.Join(app.install.SelectedClients(), ","); got != wantClients {
		t.Fatalf("Quick start changed selected clients to %q, want %q", got, wantClients)
	}
	if app.install.WithOpenCode() != wantOpenCode || app.install.OpenCodeFreeFirst() != wantFreeFirst {
		t.Fatal("Quick start changed optional-content choices")
	}
	if app.install.Step() != InstallStepOptions {
		t.Fatalf("Quick start changed the install step to %v", app.install.Step())
	}
}

// TestT34PostSetupOffersCopyablePrompt proves a successful setup result shows
// the exact first Router prompt and next steps as visible text, with no
// automatic copy or execution.
func TestT34PostSetupOffersCopyablePrompt(t *testing.T) {
	model := NewInstallModel(TUIConfig{Home: "/tmp/t34-home", Repo: "/tmp/t34-repo", Width: 90})
	model.step = InstallStepResult
	model.outcome = InstallOutcomeApplied

	view := model.Render(90, 0)
	flat := t34Flat(view)
	if !strings.Contains(flat, t34Flat(FirstRouterPrompt)) {
		t.Fatalf("successful setup result does not show the copyable prompt %q:\n%s", FirstRouterPrompt, view)
	}
	for _, want := range []string{PostSetupSectionTitle, "Next steps:", "Nothing is copied automatically"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("successful setup result is missing %q:\n%s", want, view)
		}
	}
	if !strings.Contains(flat, "no agent, client, clipboard or browser is started") {
		t.Fatal("successful setup result does not state that nothing is launched")
	}
}

// TestT34PlainNarrowAndNoFuturePlaceholder proves the offered guidance stays
// readable in plain narrow sessions and never introduces the deferred Fleet or
// profile-editor surface.
func TestT34PlainNarrowAndNoFuturePlaceholder(t *testing.T) {
	finish := NewOnboardingModel(TUIConfig{Width: 40, Height: 20}, NewOnboardingSummaryFromSetup(contrastCatalog(), t34Setup(t)))
	for i := 0; i < 3; i++ {
		finish = t34Next(t, finish)
	}
	for _, width := range []int{40, 60, 79, 80} {
		tourText := finish.Render(width)
		if strings.Contains(tourText, "\x1b") {
			t.Fatalf("plain finish view at width %d contains an escape sequence", width)
		}
		for _, line := range strings.Split(tourText, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("finish line width %d exceeds %d: %q", ansi.StringWidth(line), width, line)
			}
		}
		if strings.Contains(tourText, "Fleet") {
			t.Fatalf("Quick start finish view exposes the deferred Fleet surface")
		}
	}

	// The new guidance itself is wrapped at every reviewed narrow width. The
	// result screen's pre-existing heading wrapping is owned by B13, not T34.
	for _, width := range []int{40, 60, 79, 80} {
		for _, line := range Section(PostSetupSectionTitle, PostSetupBody(), width) {
			if ansi.StringWidth(line) > width {
				t.Fatalf("post-setup guidance line width %d exceeds %d: %q", ansi.StringWidth(line), width, line)
			}
		}
	}

	result := NewInstallModel(TUIConfig{Home: "/tmp/t34-home", Repo: "/tmp/t34-repo", Width: 40})
	result.step = InstallStepResult
	result.outcome = InstallOutcomeApplied
	for _, width := range []int{40, 60, 79, 80} {
		text := result.Render(width, 0)
		if strings.Contains(text, "\x1b") {
			t.Fatalf("plain result view at width %d contains an escape sequence", width)
		}
		if !strings.Contains(t34Flat(text), t34Flat(FirstRouterPrompt)) {
			t.Fatalf("result at width %d dropped the copyable prompt", width)
		}
		if strings.Contains(text, "Fleet") || strings.Contains(text, "profile editor") {
			t.Fatalf("successful setup result at width %d exposes a deferred surface", width)
		}
	}
}

func ansiWidth(line string) int {
	return ansi.StringWidth(line)
}

// TestT34QuickStartHasNoAutomationPath proves the tour and the post-setup
// guidance contain no clipboard, process, network or client-execution seam.
func TestT34QuickStartHasNoAutomationPath(t *testing.T) {
	source, err := os.ReadFile("onboarding.go")
	if err != nil {
		t.Fatalf("read onboarding.go: %v", err)
	}
	for _, forbidden := range []string{
		"atotto/clipboard", "os/exec", "exec.Command", "exec.Start",
		"os.StartProcess", "net/http", "net.Dial", "syscall.Exec",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Errorf("onboarding.go contains %q; Quick start must not automate anything", forbidden)
		}
	}

	// Only explicit link keys may emit a command; navigation, skip, cancel,
	// back and reopen never do.
	tour := NewOnboardingModel(TUIConfig{Width: 90, Height: 28}, NewOnboardingSummaryFromSetup(contrastCatalog(), t34Setup(t)))
	keys := []tea.KeyPressMsg{
		t34Key(tea.KeyEnter), t34Key(tea.KeyEnter), t34Key(tea.KeyEnter),
		t34Key(tea.KeyLeft), t34Key(tea.KeyRight), t34Runes("j"), t34Runes("k"),
		t34Runes("s"),
	}
	for _, msg := range keys {
		_, cmd := t34Update(t, tour, msg)
		if cmd != nil {
			t.Fatalf("key %q emitted %T; only explicit link keys may act", msg.String(), cmd)
		}
	}
}
