package tui_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/skillsync"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// These tests cover the shared, skippable onboarding model. They never write to
// a real home, never start a browser, never copy to the clipboard and never run
// an agent: the model only emits an explicit OpenLinkMsg when a link key is
// pressed, and the host owns any opener.

func onboardingTestSummary() tui.OnboardingSummary {
	return tui.OnboardingSummary{
		Clients:      []string{"claude", "codex"},
		Home:         "/tmp/isolated-home",
		Repo:         "/tmp/isolated-repo",
		SkillCount:   18,
		ProfileCount: 4,
		WithOpenCode: true,
	}
}

func onboardingModel(cfg tui.TUIConfig) tui.OnboardingModel {
	return tui.NewOnboardingModel(cfg, onboardingTestSummary())
}

func flattenSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func onboardingUpdate(t *testing.T, model tui.OnboardingModel, msg tea.Msg) (tui.OnboardingModel, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(msg)
	next, ok := updated.(tui.OnboardingModel)
	if !ok {
		t.Fatalf("Update returned %T, want tui.OnboardingModel", updated)
	}
	return next, cmd
}

func onboardingNext(t *testing.T, model tui.OnboardingModel) tui.OnboardingModel {
	t.Helper()
	next, _ := onboardingUpdate(t, model, key(tea.KeyEnter))
	return next
}

func onboardingBack(t *testing.T, model tui.OnboardingModel) tui.OnboardingModel {
	t.Helper()
	next, _ := onboardingUpdate(t, model, key(tea.KeyLeft))
	return next
}

func onboardingAtFinish(t *testing.T, cfg tui.TUIConfig) tui.OnboardingModel {
	t.Helper()
	model := onboardingModel(cfg)
	model = onboardingNext(t, model)
	model = onboardingNext(t, model)
	model = onboardingNext(t, model)
	return model
}

// TestOnboardingStepOrderAndExplicitFinish proves the shared tour moves
// welcome -> installed -> review -> finish, and that reaching the finish step is
// an explicit completed outcome rather than an implicit side effect.
func TestOnboardingStepOrderAndExplicitFinish(t *testing.T) {
	cfg := tui.TUIConfig{Width: 80, Height: 24}
	model := onboardingModel(cfg)

	if model.Step() != tui.OnboardingWelcome {
		t.Fatalf("initial Step() = %v, want Welcome", model.Step())
	}
	if model.Outcome() != tui.OnboardingActive {
		t.Fatalf("initial Outcome() = %v, want Active", model.Outcome())
	}
	if model.Finished() || model.Done() {
		t.Fatal("a new onboarding model must not be finished or done")
	}
	if model.Prompt() != tui.FirstRouterPrompt {
		t.Fatalf("Prompt() = %q, want the shared first router prompt", model.Prompt())
	}

	model = onboardingNext(t, model)
	if model.Step() != tui.OnboardingInstalled {
		t.Fatalf("after first next Step() = %v, want Installed", model.Step())
	}
	model = onboardingNext(t, model)
	if model.Step() != tui.OnboardingReview {
		t.Fatalf("after second next Step() = %v, want Review", model.Step())
	}
	model = onboardingNext(t, model)
	if model.Step() != tui.OnboardingFinish {
		t.Fatalf("after third next Step() = %v, want Finish", model.Step())
	}
	if model.Outcome() != tui.OnboardingCompleted {
		t.Fatalf("finish Outcome() = %v, want Completed", model.Outcome())
	}
	if !model.Finished() {
		t.Fatal("Finished() = false on the finish step")
	}
	if model.Done() {
		t.Fatal("Done() = true on the finish step; the host must show the prompt before closing")
	}
	if !strings.Contains(model.Render(80), "Onboarding complete") {
		t.Error("finish view does not state that onboarding is complete")
	}
}

// TestOnboardingBackIsSafe proves back never wraps past the first step, and
// leaving the finish step returns to review instead of leaving a stale
// completed outcome.
func TestOnboardingBackIsSafe(t *testing.T) {
	cfg := tui.TUIConfig{Width: 80, Height: 24}
	model := onboardingAtFinish(t, cfg)

	model = onboardingBack(t, model)
	if model.Step() != tui.OnboardingReview {
		t.Fatalf("back from finish Step() = %v, want Review", model.Step())
	}
	if model.Outcome() != tui.OnboardingActive || model.Finished() {
		t.Fatalf("back from finish left outcome %v / Finished %v, want Active / false", model.Outcome(), model.Finished())
	}

	model = onboardingBack(t, model)
	if model.Step() != tui.OnboardingInstalled {
		t.Fatalf("back from review Step() = %v, want Installed", model.Step())
	}
	model = onboardingBack(t, model)
	if model.Step() != tui.OnboardingWelcome {
		t.Fatalf("back from installed Step() = %v, want Welcome", model.Step())
	}
	model = onboardingBack(t, model)
	if model.Step() != tui.OnboardingWelcome {
		t.Fatalf("back from welcome Step() = %v, want Welcome (safe no-op)", model.Step())
	}
}

// TestOnboardingSkipAndCancelAreSafe proves skip and cancel are explicit,
// terminal, non-writing outcomes that cannot be advanced afterwards.
func TestOnboardingSkipAndCancelAreSafe(t *testing.T) {
	cfg := tui.TUIConfig{Width: 80, Height: 24}

	skipped, _ := onboardingUpdate(t, onboardingModel(cfg), runes("s"))
	if skipped.Outcome() != tui.OnboardingSkipped {
		t.Fatalf("skip Outcome() = %v, want Skipped", skipped.Outcome())
	}
	if !skipped.Done() || skipped.Finished() {
		t.Fatalf("skip Done() = %v Finished() = %v, want done and not finished", skipped.Done(), skipped.Finished())
	}
	if view := skipped.Render(80); !strings.Contains(view, "skipped") {
		t.Error("skipped view does not state the explicit outcome")
	}
	after, _ := onboardingUpdate(t, skipped, key(tea.KeyEnter))
	if after.Outcome() != tui.OnboardingSkipped || after.Step() != skipped.Step() {
		t.Fatal("a skipped tour changed state after another key")
	}

	cancelled, _ := onboardingUpdate(t, onboardingModel(cfg), key(tea.KeyEscape))
	if cancelled.Outcome() != tui.OnboardingCancelled {
		t.Fatalf("cancel Outcome() = %v, want Cancelled", cancelled.Outcome())
	}
	if !cancelled.Done() || cancelled.Finished() {
		t.Fatalf("cancel Done() = %v Finished() = %v, want done and not finished", cancelled.Done(), cancelled.Finished())
	}
	if view := cancelled.Render(80); !strings.Contains(view, "cancelled") {
		t.Error("cancelled view does not state the explicit outcome")
	}
	after, _ = onboardingUpdate(t, cancelled, key(tea.KeyRight))
	if after.Outcome() != tui.OnboardingCancelled || after.Step() != cancelled.Step() {
		t.Fatal("a cancelled tour changed state after another key")
	}
}

// TestOnboardingPromptAndLinksAreCopyable proves the finish step shows the
// exact first router prompt and both fixed URLs as visible, copyable text.
func TestOnboardingPromptAndLinksAreCopyable(t *testing.T) {
	model := onboardingAtFinish(t, tui.TUIConfig{Width: 80, Height: 24, Color: false})
	view := model.Render(80)

	if !strings.Contains(flattenSpaces(view), flattenSpaces(tui.FirstRouterPrompt)) {
		t.Errorf("finish view does not contain the first prompt %q (wrapped is acceptable)", tui.FirstRouterPrompt)
	}
	if !strings.Contains(tui.FirstRouterPrompt, "agentic-sdd-router") {
		t.Error("the first prompt does not name agentic-sdd-router")
	}
	if !strings.Contains(view, tui.ReadmeURL) {
		t.Errorf("finish view does not contain the README URL %q", tui.ReadmeURL)
	}
	if !strings.Contains(view, tui.AuthorURL) {
		t.Errorf("finish view does not contain the author URL %q", tui.AuthorURL)
	}
	if !strings.Contains(view, tui.BrandAuthor) {
		t.Errorf("finish view does not contain the exact author %q", tui.BrandAuthor)
	}
	if !strings.Contains(view, "Nothing is copied automatically") {
		t.Error("finish view does not explain that copying is manual")
	}
	if !strings.Contains(view, "no agent is started") {
		t.Error("finish view does not explain that no agent runs")
	}
}

// TestOnboardingBoundsAtBreakpoints proves every rendered line fits the named
// breakpoints in terminal cells, including the wrapped URL and prompt.
func TestOnboardingBoundsAtBreakpoints(t *testing.T) {
	for _, width := range []int{40, 60, 79, 80, 109, 110} {
		model := onboardingModel(tui.TUIConfig{Width: width, Height: 24, Color: true})
		for step := 0; step < 4; step++ {
			for _, line := range model.Lines(width) {
				if got := ansi.StringWidth(line); got > width {
					t.Fatalf("width %d step %d: line width %d exceeds %d: %q", width, step, got, width, line)
				}
			}
			model = onboardingNext(t, model)
		}
	}
}

// TestOnboardingTinyWidthsDoNotPanic proves a degenerate viewport returns an
// empty or bounded result instead of overflowing or panicking.
func TestOnboardingTinyWidthsDoNotPanic(t *testing.T) {
	for _, width := range []int{0, 1, 2, 5, 20} {
		model := onboardingModel(tui.TUIConfig{Width: width, Height: 1})
		for step := 0; step < 4; step++ {
			for _, line := range model.Lines(width) {
				if width > 0 {
					if got := ansi.StringWidth(line); got > width {
						t.Fatalf("width %d: line width %d exceeds the viewport: %q", width, got, line)
					}
				}
			}
			_ = model.Render(width)
			_ = model.View()
			model = onboardingNext(t, model)
		}
	}
}

// TestOnboardingPlainAndASCIIHaveNoHyperlinks proves plain output carries no
// escape sequences and ASCII output never emits OSC 8, while the visible URL
// remains present in both.
func TestOnboardingPlainAndASCIIHaveNoHyperlinks(t *testing.T) {
	plain := onboardingAtFinish(t, tui.TUIConfig{Width: 80, Height: 24, Color: false})
	plainView := plain.Render(80)
	if strings.Contains(plainView, "\x1b") {
		t.Errorf("plain onboarding contains an escape sequence: %q", plainView)
	}
	if !strings.Contains(plainView, tui.ReadmeURL) {
		t.Error("plain onboarding dropped the visible README URL")
	}

	ascii := onboardingAtFinish(t, tui.TUIConfig{Width: 80, Height: 24, Color: true, ASCII: true})
	asciiView := ascii.Render(80)
	if strings.Contains(asciiView, "\x1b]8;;") {
		t.Errorf("ASCII onboarding emitted an OSC 8 hyperlink: %q", asciiView)
	}
	if !strings.Contains(asciiView, tui.ReadmeURL) {
		t.Error("ASCII onboarding dropped the visible README URL")
	}
}

// TestOnboardingOSC8OnlyWhenColorAndUnicode proves the terminal hyperlink is
// best-effort and only enabled for a colored, non-ASCII session.
func TestOnboardingOSC8OnlyWhenColorAndUnicode(t *testing.T) {
	model := onboardingAtFinish(t, tui.TUIConfig{Width: 80, Height: 24, Color: true})
	view := model.Render(80)
	if !strings.Contains(view, "\x1b]8;;"+tui.ReadmeURL) {
		t.Errorf("colored onboarding does not hyperlink the README URL: %q", view)
	}
	if !strings.Contains(view, tui.ReadmeURL) {
		t.Error("hyperlinked onboarding dropped the visible README URL")
	}
}

// TestOnboardingLinksOpenOnlyOnExplicitKey proves the model never launches a
// browser while rendering or initializing; only o/a produce an OpenLinkMsg.
func TestOnboardingLinksOpenOnlyOnExplicitKey(t *testing.T) {
	model := onboardingAtFinish(t, tui.TUIConfig{Width: 80, Height: 24})

	if cmd := model.Init(); cmd != nil {
		t.Fatal("Init() returned a command; onboarding must not act at startup")
	}

	if _, cmd := onboardingUpdate(t, model, key(tea.KeyEnter)); cmd != nil {
		t.Fatal("Enter returned a command; navigation must not open a link")
	}
	if _, cmd := onboardingUpdate(t, model, key(tea.KeyRight)); cmd != nil {
		t.Fatal("Right returned a command; navigation must not open a link")
	}

	_, cmd := onboardingUpdate(t, model, runes("o"))
	if cmd == nil {
		t.Fatal("pressing o returned no command; the explicit README action is missing")
	}
	msg := cmd()
	open, ok := msg.(tui.OpenLinkMsg)
	if !ok {
		t.Fatalf("o command produced %T, want tui.OpenLinkMsg", msg)
	}
	if open.Link != tui.LinkReadme {
		t.Errorf("o opened %q, want the README link", open.Link)
	}

	_, cmd = onboardingUpdate(t, model, runes("a"))
	if cmd == nil {
		t.Fatal("pressing a returned no command; the explicit author action is missing")
	}
	msg = cmd()
	open, ok = msg.(tui.OpenLinkMsg)
	if !ok {
		t.Fatalf("a command produced %T, want tui.OpenLinkMsg", msg)
	}
	if open.Link != tui.LinkAuthor {
		t.Errorf("a opened %q, want the author link", open.Link)
	}
}

// TestOnboardingSummaryFromCatalog proves the review summary is derived from
// the read-only catalog: unique sorted clients, honest counts and optional
// content presence.
func TestOnboardingSummaryFromCatalog(t *testing.T) {
	catalog := tui.Catalog{
		Home: "/tmp/isolated-home",
		Repo: "/tmp/isolated-repo",
		Items: []tui.CatalogItem{
			{ID: "one", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Clients: []tui.ClientStatus{{Client: "codex"}, {Client: "claude"}}},
			{ID: "two", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Clients: []tui.ClientStatus{{Client: "claude"}}},
			{ID: "pack", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginPack},
			{ID: "preset", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPreset},
		},
	}

	summary := tui.NewOnboardingSummary(catalog)
	if want := []string{"claude", "codex"}; !reflect.DeepEqual(summary.Clients, want) {
		t.Errorf("Clients = %v, want unique sorted %v", summary.Clients, want)
	}
	if summary.SkillCount != 3 {
		t.Errorf("SkillCount = %d, want 3", summary.SkillCount)
	}
	if summary.ProfileCount != 1 {
		t.Errorf("ProfileCount = %d, want 1", summary.ProfileCount)
	}
	if !summary.WithOpenCode || !summary.OpenCodeFreeFirst {
		t.Errorf("optional content = pack %v presets %v, want both present", summary.WithOpenCode, summary.OpenCodeFreeFirst)
	}
	if summary.Home != catalog.Home || summary.Repo != catalog.Repo {
		t.Errorf("source summary = %q/%q, want %q/%q", summary.Home, summary.Repo, catalog.Home, catalog.Repo)
	}
}

// TestOnboardingDoesNotWriteOrAutomate proves the tour has no filesystem side
// effect and its source does not import a clipboard or process launcher.
func TestOnboardingDoesNotWriteOrAutomate(t *testing.T) {
	home := t.TempDir()
	before, err := os.ReadDir(home)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", home, err)
	}

	model := tui.NewOnboardingModel(
		tui.TUIConfig{Width: 80, Height: 24, Color: true},
		tui.OnboardingSummary{Home: home, Repo: "/tmp/isolated-repo", Clients: []string{"claude"}},
	)
	for step := 0; step < 4; step++ {
		_ = model.Render(80)
		_ = model.View()
		model = onboardingNext(t, model)
	}
	_ = model.Render(80)

	after, err := os.ReadDir(home)
	if err != nil {
		t.Fatalf("ReadDir(%q) after tour: %v", home, err)
	}
	if len(before) != len(after) {
		t.Fatalf("onboarding changed %q: before %d entries, after %d", home, len(before), len(after))
	}

	source, err := os.ReadFile("onboarding.go")
	if err != nil {
		t.Fatalf("read onboarding.go: %v", err)
	}
	for _, forbidden := range []string{"atotto/clipboard", "os/exec", "exec.Command", "exec.Start"} {
		if strings.Contains(string(source), forbidden) {
			t.Errorf("onboarding.go contains %q; the tour must not copy or launch anything", forbidden)
		}
	}
}
