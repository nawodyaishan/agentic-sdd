package tui

// B8/T24 regression tests: one authoritative confirmation surface, inert
// backdrop input, reverse Tab, and one shared action vocabulary for the
// contextual footer and help. All operations are fakes; nothing is written.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

func b8ShiftTab() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
}

func b8Count(haystack, needle string) int {
	return strings.Count(haystack, needle)
}

// TestB8ConfirmationHasOneAuthoritativeSurface reproduces the review's
// duplicate-confirmation observation deterministically and requires the
// rendered dialog to carry the essential facts exactly once.
func TestB8ConfirmationHasOneAuthoritativeSurface(t *testing.T) {
	t.Run("install", func(t *testing.T) {
		app := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
		backdrop := ansi.Strip(app.renderModalBackdrop())
		for _, leaked := range []string{"Pre-Execution Policy", "Proceed with installation?", "Target Configuration", "target home"} {
			if strings.Contains(strings.ToLower(backdrop), strings.ToLower(leaked)) {
				t.Fatalf("the inactive backdrop repeats confirmation content %q:\n%s", leaked, backdrop)
			}
		}
		plain := ansi.Strip(app.View().Content)
		for _, fact := range []string{
			"Install & Manage · Confirm Installation",
			"Pre-Execution Policy",
			"Target Configuration",
			"Proceed with installation?",
		} {
			if got := b8Count(plain, fact); got != 1 {
				t.Fatalf("%q appears %d times, want exactly one authoritative copy:\n%s", fact, got, plain)
			}
		}

		// Help over the confirmation must not re-expose a second confirmation
		// copy under the help overlay either.
		help := b8Press(t, app, '?')
		if !help.HelpOpen() {
			t.Fatal("? did not open help over the confirmation")
		}
		helpBackdrop := ansi.Strip(help.renderModalBackdrop())
		if strings.Contains(helpBackdrop, "Pre-Execution Policy") || strings.Contains(helpBackdrop, "Proceed with installation?") {
			t.Fatalf("help backdrop repeats the confirmation:\n%s", helpBackdrop)
		}
	})

	t.Run("restore", func(t *testing.T) {
		app := t19OpenRestoreConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
		backdrop := ansi.Strip(app.renderModalBackdrop())
		for _, leaked := range []string{"Confirm restore", "Default: No", "Restore backup"} {
			if strings.Contains(backdrop, leaked) {
				t.Fatalf("the inactive backdrop repeats restore confirmation content %q:\n%s", leaked, backdrop)
			}
		}
		plain := ansi.Strip(app.View().Content)
		for _, fact := range []string{"Confirm restore", "Default: No", "Will restore a missing destination"} {
			if got := b8Count(plain, fact); got != 1 {
				t.Fatalf("%q appears %d times, want exactly one authoritative copy:\n%s", fact, got, plain)
			}
		}
	})
}

// TestB8ConfirmationBackdropIsInert proves the confirmation surface owns every
// input: route jumps, settings, help-free navigation and Tab cannot reach the
// inactive backdrop or change consent state.
func TestB8ConfirmationBackdropIsInert(t *testing.T) {
	app := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
	token := app.install.PreviewToken()
	route, focus, nav := app.ActiveRoute(), app.Focus(), app.NavIndex()
	motion, ascii := app.Motion(), app.ASCII()

	for _, msg := range []tea.KeyPressMsg{
		b8Key('1'), b8Key('2'), b8Key('3'), b8Key('4'),
		b8Key('m'), b8Key('u'), b8Key('c'), b8Key('['), b8Key(']'),
		b8Code(tea.KeyTab), b8ShiftTab(),
	} {
		next, cmd := b8Update(t, app, msg)
		if cmd != nil {
			t.Fatalf("%q produced a command while a confirmation owns focus", msg.String())
		}
		if next.ActiveRoute() != route || next.Focus() != focus || next.NavIndex() != nav {
			t.Fatalf("%q reached backdrop navigation", msg.String())
		}
		if next.Motion() != motion || next.ASCII() != ascii {
			t.Fatalf("%q reached backdrop session settings", msg.String())
		}
		if next.install.Step() != InstallStepConfirm || next.install.PreviewToken() != token {
			t.Fatalf("%q changed confirmation state", msg.String())
		}
		app = next
	}

	// Enter stays inert; only y starts the write.
	app, cmd := b8Update(t, app, b8Code(tea.KeyEnter))
	if cmd != nil || app.install.Step() != InstallStepConfirm {
		t.Fatal("Enter confirmed or changed the confirmation")
	}
	_, cmd = b8Update(t, app, b8Key('y'))
	if cmd == nil {
		t.Fatal("y did not emit the write request")
	}
}

func TestB8RestoreConfirmationOwnsInputAndQDoesNotQuit(t *testing.T) {
	app := t19OpenRestoreConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
	route, focus := app.ActiveRoute(), app.Focus()
	token := app.restore.preview.PreviewToken

	for _, msg := range []tea.KeyPressMsg{b8Key('2'), b8Key('m'), b8Code(tea.KeyTab), b8ShiftTab()} {
		next, cmd := b8Update(t, app, msg)
		if cmd != nil {
			t.Fatalf("%q produced a command in restore confirmation", msg.String())
		}
		if next.ActiveRoute() != route || next.Focus() != focus || next.restore.Stage() != RestoreStageConfirm {
			t.Fatalf("%q escaped the restore confirmation", msg.String())
		}
		app = next
	}

	// q is a safe cancel, never an application quit through the modal.
	next, cmd := b8Update(t, app, b8Key('q'))
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("q quit the application through the restore confirmation")
		}
	}
	if next.restore.Stage() != RestoreStagePreview {
		t.Fatalf("q stage = %v, want preview", next.restore.Stage())
	}
	if next.restore.consentToken != "" {
		t.Fatal("q retained restore consent")
	}
	if next.restore.preview.PreviewToken != token {
		t.Fatal("q changed the read-only preview token")
	}

	// Uppercase Y is the same explicit affirmative alias.
	app = t19OpenRestoreConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
	app, cmd = b8Update(t, app, b8Key('Y'))
	if cmd == nil {
		t.Fatal("Y did not start the restore revalidation")
	}
	if app.restore.Stage() != RestoreStageRevalidating {
		t.Fatalf("Y stage = %v, want revalidating", app.restore.Stage())
	}
}

func TestB8InstallConfirmationQSafeCancelsWithoutQuitting(t *testing.T) {
	app := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
	next, cmd := b8Update(t, app, b8Key('q'))
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("q quit the application through the install confirmation")
		}
	}
	if next.install.Step() != InstallStepResult || next.install.Outcome() != InstallOutcomeCancelled {
		t.Fatalf("q step=%v outcome=%v, want a safe cancellation", next.install.Step(), next.install.Outcome())
	}
}

func TestB8ShiftTabCyclesFocusInReverse(t *testing.T) {
	app := NewApp(TUIConfig{Width: 120, Height: 30, Interactive: true}, t19ActivityCatalog())
	if app.Focus() != FocusNav {
		t.Fatalf("initial focus = %v, want nav", app.Focus())
	}
	app, _ = b8Update(t, app, b8ShiftTab())
	if app.Focus() != FocusContent {
		t.Fatalf("reverse Tab from nav = %v, want content", app.Focus())
	}
	app, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyTab})
	if app.Focus() != FocusNav {
		t.Fatalf("forward Tab from content = %v, want nav", app.Focus())
	}

	// Explore has three regions; reverse order is nav → content → categories.
	app = b8Press(t, app, '3')
	if app.Focus() != FocusContent {
		t.Fatalf("Explore focus = %v, want content", app.Focus())
	}
	app, _ = b8Update(t, app, b8ShiftTab())
	if app.Focus() != FocusCategories {
		t.Fatalf("Explore reverse Tab from content = %v, want categories", app.Focus())
	}
	app, _ = b8Update(t, app, b8ShiftTab())
	if app.Focus() != FocusNav {
		t.Fatalf("Explore reverse Tab from categories = %v, want nav", app.Focus())
	}
	app, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyTab})
	if app.Focus() != FocusCategories {
		t.Fatalf("Explore forward Tab from nav = %v, want categories", app.Focus())
	}
}

func TestB8ContextualFooterAndHelpShareEligibility(t *testing.T) {
	install := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: false}, "/tmp/b8-home/.claude/skills/t19")
	view := install.View().Content
	last := ansi.Strip(t19Lines(view)[len(t19Lines(view))-1])
	if !strings.Contains(last, "y confirm") || !strings.Contains(last, "cancel") {
		t.Fatalf("confirmation footer lacks its decision actions: %q", last)
	}
	if strings.Contains(last, "q quit") {
		t.Fatalf("confirmation footer falsely advertises q quit: %q", last)
	}
	if strings.Contains(last, "Enter confirm") || strings.Contains(last, "Enter preview") {
		t.Fatalf("confirmation footer advertises an inert key: %q", last)
	}

	help := b8Press(t, install, '?')
	plain := ansi.Strip(help.View().Content)
	if !strings.Contains(plain, "This Screen") {
		t.Fatalf("help lacks the contextual action section:\n%s", plain)
	}
	for _, want := range []string{"only y or Y confirms", "Enter is inert", "safe cancel"} {
		if !strings.Contains(ansi.Strip(strings.Join(help.helpModalParts(80).Body, "\n")), want) {
			t.Fatalf("contextual help missing %q:\n%s", want, plain)
		}
	}
	help = b8Press(t, help, '?')
	if help.install.Step() != InstallStepConfirm || help.install.PreviewToken() == "" {
		t.Fatal("closing contextual help lost the confirmation state")
	}

	// The no-op preview footer advertises only its read-only actions.
	noop := b8App(t, TUIConfig{Width: 100, Height: 30}, b8CountingHost(&b8Counters{}), fakePreviewOp(b8AllUpToDate("token-footer"), "token-footer", nil))
	noop = b8OpenInstallPreview(t, noop)
	last = ansi.Strip(t19Lines(noop.View().Content)[len(t19Lines(noop.View().Content))-1])
	for _, want := range []string{"h Home", "Esc", "q quit"} {
		if !strings.Contains(last, want) {
			t.Fatalf("no-op footer missing %q: %q", want, last)
		}
	}
	if strings.Contains(last, "Enter confirm") || strings.Contains(last, "y confirm") {
		t.Fatalf("no-op footer offers a write action: %q", last)
	}
}

func TestB8HelpModalQDoesNotQuit(t *testing.T) {
	app := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
	app = b8Press(t, app, '?')
	if !app.HelpOpen() {
		t.Fatal("? did not open help")
	}
	next, cmd := b8Update(t, app, b8Key('q'))
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("q quit the application through the help modal")
		}
	}
	if !next.HelpOpen() {
		t.Fatal("q closed help unexpectedly; Esc/?/Enter are the documented close keys")
	}
	next, _ = b8Update(t, next, b8Code(tea.KeyEscape))
	if next.HelpOpen() {
		t.Fatal("Esc did not close help")
	}
	if next.install.Step() != InstallStepConfirm || next.install.PreviewToken() == "" {
		t.Fatal("closing help lost the confirmation state")
	}
}

func TestB8RestoreUnknownActionBlocksConfirmation(t *testing.T) {
	ops := RestoreOps{
		List: func(repo, home string) ([]skillsync.Backup, error) {
			return []skillsync.Backup{{ID: "20260101T000000.000000000Z"}}, nil
		},
		Preview: func(req appops.RestoreRequest) (appops.PreviewResult, error) {
			return appops.PreviewResult{
				Operation:    skillsync.OperationRestore,
				Outcome:      skillsync.OutcomePreview,
				PreviewToken: "token-restore-unknown",
				Planned: []skillsync.ResultChange{{
					Client: "claude",
					ID:     "future",
					Path:   "/tmp/b8-home/.claude/skills/future",
					Action: skillsync.ChangeAction("reticulate"),
				}},
			}, nil
		},
	}
	cfg := TUIConfig{Width: 100, Height: 30, Color: true, Home: t.TempDir()}
	app := NewAppWithScreens(cfg.Normalize(), Catalog{Home: cfg.Home}, AppOptions{}, AppScreenOptions{RestoreOps: ops})
	app = b8Press(t, app, '2')
	app, cmd := b8Update(t, app, b8Key('b'))
	app = b8Drain(t, app, cmd)
	app, _ = b8Update(t, app, b8Code(tea.KeyEnter)) // details
	app, cmd = b8Update(t, app, b8Code(tea.KeyEnter))
	app = b8Drain(t, app, cmd) // preview
	if app.restore.Stage() != RestoreStagePreview {
		t.Fatalf("stage = %v, want preview", app.restore.Stage())
	}
	plain := ansi.Strip(app.View().Content)
	for _, want := range []string{"Impact totals:", "Other planned changes (1)", "Confirmation is disabled"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("restore preview missing %q:\n%s", want, plain)
		}
	}
	next, cmd := b8Update(t, app, b8Code(tea.KeyEnter))
	if cmd != nil || next.restore.Stage() != RestoreStagePreview {
		t.Fatalf("unknown restore action advanced to %v", next.restore.Stage())
	}
}

func TestB8InstallEligibilityMatrixOnlyYStartsAWrite(t *testing.T) {
	keys := []tea.KeyPressMsg{
		b8Code(tea.KeyEnter), b8Code(tea.KeyEsc), b8Key('q'), b8Key(' '), b8Key('?'), b8Key('/'),
		b8Key('j'), b8Key('k'), b8Code(tea.KeyUp), b8Code(tea.KeyDown),
		b8Code(tea.KeyLeft), b8Code(tea.KeyRight), b8Code(tea.KeyTab), b8ShiftTab(),
		b8Key('n'), b8Key('y'), b8Key('Y'),
	}
	steps := []InstallStep{InstallStepClients, InstallStepOptions, InstallStepReview, InstallStepPreview, InstallStepConfirm, InstallStepResult}
	for _, step := range steps {
		for _, key := range keys {
			m := b8PreviewModel(t, b8AllUpToDate("token-matrix"), "token-matrix", nil)
			m.step = step
			if step == InstallStepConfirm {
				m.confirmNoSelected = true
			}
			next, cmd := m.Update(key)
			model, ok := next.(InstallModel)
			if !ok {
				t.Fatalf("Update returned %T", next)
			}
			if cmd == nil {
				continue
			}
			if _, write := cmd().(StartWriteMsg); write {
				affirmative := key.String() == "y" || key.String() == "Y"
				if step != InstallStepConfirm || !affirmative {
					t.Fatalf("step %v key %q started a write", step, key.String())
				}
			}
			if step == InstallStepConfirm && model.Step() == InstallStepRunning && !(key.String() == "y" || key.String() == "Y") {
				t.Fatalf("step %v key %q entered running without an affirmative key", step, key.String())
			}
		}
	}
}

func TestB8RestoreConfirmationEligibilityMatrix(t *testing.T) {
	keys := []tea.KeyPressMsg{
		b8Code(tea.KeyEnter), b8Code(tea.KeyEsc), b8Key('q'), b8Key(' '), b8Key('?'), b8Key('/'),
		b8Key('j'), b8Key('k'), b8Code(tea.KeyTab), b8ShiftTab(), b8Key('n'), b8Key('y'), b8Key('Y'),
	}
	for _, key := range keys {
		app := t19OpenRestoreConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
		next, cmd := b8Update(t, app, key)
		if cmd != nil {
			if _, write := cmd().(StartWriteMsg); write {
				t.Fatalf("restore confirmation key %q started a write before revalidation", key.String())
			}
			if msg := cmd(); msg != nil {
				if _, quit := msg.(tea.QuitMsg); quit {
					t.Fatalf("restore confirmation key %q quit the application", key.String())
				}
			}
		}
		if next.write != nil {
			t.Fatalf("restore confirmation key %q claimed the root write slot", key.String())
		}
	}
}

func TestB8RestoreImpactTotalsAndPathsRemainInspectable(t *testing.T) {
	app := t19OpenRestoreConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/b8-home/.claude/skills/t19")
	plain := ansi.Strip(app.View().Content)
	if !strings.Contains(plain, "Impact totals:") {
		t.Fatalf("restore preview/confirm lacks exact action totals:\n%s", plain)
	}
	if !strings.Contains(plain, "/tmp/b8-home/.claude/skills/t19") {
		t.Fatalf("restore confirmation lost its complete path:\n%s", plain)
	}
}
