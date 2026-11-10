package tui

// Regression coverage for the OpenAI Sol safety/state review findings:
// confirmations below the usable geometry, stale catalog filter results,
// activity read correlation, the no-op inventory refresh, recommendation
// detail overrides, and the contextual 'm' key. Every operation is a fake or a
// temporary home; nothing writes to a real user directory or the network.

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/skillsync"
)

func reviewKey(r rune) tea.KeyPressMsg {
	if r < 32 || r == 127 {
		return tea.KeyPressMsg{Code: r}
	}
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// shrinkBelowUsableGeometry resizes an app past the documented minimum so the
// viewport can only render resize guidance.
func shrinkBelowUsableGeometry(t *testing.T, app App) App {
	t.Helper()
	next, _ := b8Update(t, app, tea.WindowSizeMsg{Width: 39, Height: 7})
	view := ansi.Strip(next.View().Content)
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("below-geometry view did not show resize guidance:\n%s", view)
	}
	return next
}

// TestConfirmationBelowUsableGeometryCannotSubmit proves an open install or
// restore confirmation accepts no affirmative key while the viewport shows only
// resize guidance. Cancellation and shutdown stay available.
func TestConfirmationBelowUsableGeometryCannotSubmit(t *testing.T) {
	t.Run("install", func(t *testing.T) {
		app := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/review-install/.claude/skills/skill")
		token := app.install.PreviewToken()
		app = shrinkBelowUsableGeometry(t, app)

		for _, msg := range []tea.KeyPressMsg{
			reviewKey('y'), reviewKey('Y'), tea.KeyPressMsg{Code: tea.KeyEnter},
			tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, reviewKey('?'), reviewKey('j'),
		} {
			next, cmd := b8Update(t, app, msg)
			if cmd != nil {
				if emitted := cmd(); emitted != nil {
					if _, write := emitted.(StartWriteMsg); write {
						t.Fatalf("%q submitted a write below usable geometry", msg.String())
					}
				}
			}
			app = next
			if app.install.Step() != InstallStepConfirm {
				t.Fatalf("%q left the confirmation for %v", msg.String(), app.install.Step())
			}
			if app.install.PreviewToken() != token {
				t.Fatalf("%q changed the confirmation token", msg.String())
			}
			if app.Busy() {
				t.Fatalf("%q claimed the root write slot", msg.String())
			}
		}

		// Safe cancellation still works and writes nothing.
		app, cmd := b8Update(t, app, reviewKey('n'))
		if cmd != nil {
			if emitted := cmd(); emitted != nil {
				if _, write := emitted.(StartWriteMsg); write {
					t.Fatal("cancellation submitted a write")
				}
			}
		}
		if app.install.Step() != InstallStepPreview {
			t.Fatalf("n stage = %v, want preview", app.install.Step())
		}
	})

	t.Run("install shutdown", func(t *testing.T) {
		app := t19OpenInstallConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/review-install/.claude/skills/skill")
		app = shrinkBelowUsableGeometry(t, app)
		_, cmd := b8Update(t, app, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Fatal("Ctrl+C was not accepted below usable geometry")
		}
		if _, quit := cmd().(tea.QuitMsg); !quit {
			t.Fatal("Ctrl+C did not request shutdown")
		}
	})

	t.Run("restore", func(t *testing.T) {
		app := t19OpenRestoreConfirm(t, TUIConfig{Width: 100, Height: 32, Color: true}, "/tmp/review-restore/.claude/skills/skill")
		app = shrinkBelowUsableGeometry(t, app)

		for _, msg := range []tea.KeyPressMsg{
			reviewKey('y'), reviewKey('Y'), tea.KeyPressMsg{Code: tea.KeyEnter},
			tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, reviewKey('?'), reviewKey('j'),
		} {
			next, cmd := b8Update(t, app, msg)
			if cmd != nil {
				if emitted := cmd(); emitted != nil {
					if _, write := emitted.(StartWriteMsg); write {
						t.Fatalf("%q submitted a restore below usable geometry", msg.String())
					}
				}
			}
			app = next
			if app.restore.Stage() != RestoreStageConfirm {
				t.Fatalf("%q left the confirmation for %v", msg.String(), app.restore.Stage())
			}
			if app.restore.Busy() {
				t.Fatalf("%q started a restore transaction", msg.String())
			}
			if reported, ok := app.restore.Outcome(); ok {
				t.Fatalf("%q produced a restore outcome: %+v", msg.String(), reported)
			}
		}

		app, _ = b8Update(t, app, reviewKey('n'))
		if app.restore.Stage() != RestoreStagePreview {
			t.Fatalf("n stage = %v, want preview", app.restore.Stage())
		}
	})
}

// TestCatalogStaleFilterResultsCannotRestoreOlderRows covers both halves of the
// finding: an old result arriving after a newer query is checked against its
// generation, and an untagged result is never applied at all.
func TestCatalogStaleFilterResultsCannotRestoreOlderRows(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		{ID: "alpha-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "beta-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "echo-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
	}}
	model := NewCatalogModel(TUIConfig{Width: 120, Height: 24}, cat)

	next, _ := model.Update(reviewKey('/'))
	model = next.(CatalogModel)
	if !model.IsFiltering() {
		t.Fatal("/ did not open the filter")
	}

	next, oldCmd := model.Update(reviewKey('b'))
	model = next.(CatalogModel)
	next, newCmd := model.Update(reviewKey('e'))
	model = next.(CatalogModel)

	oldMsgs := runB9Cmd(oldCmd)
	newMsgs := runB9Cmd(newCmd)
	if len(oldMsgs) == 0 || len(newMsgs) == 0 {
		t.Fatalf("filter commands produced %d/%d messages", len(oldMsgs), len(newMsgs))
	}

	// The old query's command must emit only the checked envelope, never the
	// untagged result that used to be accepted unconditionally.
	var staleEnvelope FilterEnvelope
	for _, msg := range oldMsgs {
		if _, untagged := msg.(list.FilterMatchesMsg); untagged {
			t.Fatal("a filter command emitted an untagged result")
		}
		envelope, ok := msg.(FilterEnvelope)
		if !ok {
			continue
		}
		if _, ok := envelope.Msg.(list.FilterMatchesMsg); ok {
			staleEnvelope = envelope
		}
	}
	if staleEnvelope.Generation == 0 {
		t.Fatalf("old filter command emitted no filter envelope: %#v", oldMsgs)
	}

	for _, msg := range newMsgs {
		next, _ := model.Update(msg)
		model = next.(CatalogModel)
	}
	wantRows := model.VisibleItems()
	if len(wantRows) != 1 || wantRows[0].ID != "beta-thing" {
		t.Fatalf("newest filter rows = %v, want [beta-thing]", visibleIDs(wantRows))
	}
	category := model.ActiveCategory()

	// The older generation arrives late and is discarded.
	for _, msg := range oldMsgs {
		next, _ := model.Update(msg)
		model = next.(CatalogModel)
	}
	rows := model.VisibleItems()
	if len(rows) != 1 || rows[0].ID != "beta-thing" {
		t.Fatalf("stale generation restored older rows: %v", visibleIDs(rows))
	}
	if model.ActiveCategory() != category {
		t.Fatalf("stale generation changed the category to %q", model.ActiveCategory())
	}

	// An untagged result, as the old emitter produced, is ignored outright.
	stale, ok := staleEnvelope.Msg.(list.FilterMatchesMsg)
	if !ok {
		t.Fatalf("stale envelope carried %T", staleEnvelope.Msg)
	}
	next, _ = model.Update(stale)
	model = next.(CatalogModel)
	rows = model.VisibleItems()
	if len(rows) != 1 || rows[0].ID != "beta-thing" {
		t.Fatalf("untagged filter result restored older rows: %v", visibleIDs(rows))
	}
}

func visibleIDs(items []CatalogItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// TestCatalogStaleFilterResultCannotRestoreAfterRefresh covers the same
// generation boundary against a newer catalog refresh instead of a newer query.
func TestCatalogStaleFilterResultCannotRestoreAfterRefresh(t *testing.T) {
	initial := Catalog{Items: []CatalogItem{
		{ID: "alpha-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "beta-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "gamma-profile", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPack},
	}}
	model := NewCatalogModel(TUIConfig{Width: 120, Height: 24}, initial)
	model.SetFilter("thi")

	staleCmd := model.SetCategory(CategoryCore)
	if staleCmd == nil {
		t.Fatal("category change produced no filter command")
	}

	refreshed := Catalog{Items: []CatalogItem{
		{ID: "delta-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "epsilon-profile", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPack},
	}}
	refreshCmd := model.ReplaceCatalog(refreshed)
	if refreshCmd == nil {
		t.Fatal("ReplaceCatalog produced no command")
	}

	for _, msg := range runB9Cmd(refreshCmd) {
		next, _ := model.Update(msg)
		model = next.(CatalogModel)
	}
	rows := model.VisibleItems()
	if len(rows) != 1 || rows[0].ID != "delta-thing" {
		t.Fatalf("refreshed rows = %v, want [delta-thing]", visibleIDs(rows))
	}

	for _, msg := range runB9Cmd(staleCmd) {
		next, _ := model.Update(msg)
		model = next.(CatalogModel)
	}
	rows = model.VisibleItems()
	if len(rows) != 1 || rows[0].ID != "delta-thing" {
		t.Fatalf("stale filter result restored pre-refresh rows: %v", visibleIDs(rows))
	}
}

// activityReviewApp builds an app whose activity screen reads only fake,
// in-memory logs.
func activityReviewApp(t *testing.T) App {
	t.Helper()
	cfg := TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}
	app := NewApp(cfg, Catalog{Home: cfg.Home})
	now := time.Now()
	app.activity.logReader = func(dir string) ([]LogSummary, error) {
		return []LogSummary{
			{Name: "20260101T000000.000000000Z-new.log", Path: "/fake/new.log", ModTime: now, Size: 10},
			{Name: "20251201T000000.000000000Z-old.log", Path: "/fake/old.log", ModTime: now.Add(-time.Hour), Size: 20},
		}, nil
	}
	app.activity.contentReader = func(path string, maxBytes int64, maxLines int) (string, error) {
		return "content:" + path, nil
	}
	app.activity.logDir = "/fake/logs"
	app.activity.reloadLogs()
	return app
}

// TestActivityReadPreservesNewerSectionAndSelection proves a completed log read
// applies only read data to the live model: a Help section, log scroll and
// selection changed while the read ran survive it.
func TestActivityReadPreservesNewerSectionAndSelection(t *testing.T) {
	app := activityReviewApp(t)
	next, readCmd := b8Update(t, app, reviewKey('4'))
	app = next
	if readCmd == nil {
		t.Fatal("entering Activity did not start a bounded read")
	}

	next, _ = b8Update(t, app, reviewKey('t'))
	app = next
	section := app.activity.ActiveSection()
	next, _ = b8Update(t, app, reviewKey('J'))
	app = next
	scroll := app.activity.logScroll

	app = b8Drain(t, app, readCmd)
	if app.activity.ActiveSection() != section {
		t.Fatalf("a completed read reset the section to %v", app.activity.ActiveSection())
	}
	if app.activity.logScroll != scroll {
		t.Fatalf("a completed read reset the log scroll to %d, want %d", app.activity.logScroll, scroll)
	}
	if len(app.activity.Logs()) != 2 {
		t.Fatalf("accepted read did not apply logs: %d", len(app.activity.Logs()))
	}
}

// TestActivityStaleReadCannotOverwriteNewerRead proves generation ordering:
// the newer selection read wins even when the older read finishes last.
func TestActivityStaleReadCannotOverwriteNewerRead(t *testing.T) {
	app := activityReviewApp(t)
	next, staleCmd := b8Update(t, app, reviewKey('4'))
	app = next

	next, freshCmd := b8Update(t, app, reviewKey(']'))
	app = next
	if staleCmd == nil || freshCmd == nil {
		t.Fatal("selection did not schedule both reads")
	}
	if app.activity.LogContent() != "" {
		t.Fatalf("selection kept the previous log content while its read was pending: %q", app.activity.LogContent())
	}

	// Apply the newer read, then the older one.
	app = b8Drain(t, app, freshCmd)
	if app.activity.selectedLog != 1 {
		t.Fatalf("newer read selection = %d, want 1", app.activity.selectedLog)
	}
	if !strings.Contains(app.activity.LogContent(), "old.log") {
		t.Fatalf("newer read content = %q", app.activity.LogContent())
	}
	app = b8Drain(t, app, staleCmd)
	if app.activity.selectedLog != 1 {
		t.Fatalf("stale read moved the selection to %d", app.activity.selectedLog)
	}
	if !strings.Contains(app.activity.LogContent(), "old.log") {
		t.Fatalf("stale read replaced newer content: %q", app.activity.LogContent())
	}
}

// TestNoOpPreviewRefreshesInventoryWithoutWriting covers the read-only no-op
// path: it clears a stale inventory error through one scoped read and never
// touches the apply or log seams.
func TestNoOpPreviewRefreshesInventoryWithoutWriting(t *testing.T) {
	home := t.TempDir()
	counters := &b8Counters{}
	var refreshes int
	var scope catalogScope
	refreshed := b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource))
	app := NewAppWithScreens(
		TUIConfig{Home: home, Width: 100, Height: 30, Color: false},
		b9CoreCatalog(b9All(skillsync.InventoryStatusMissing)),
		AppOptions{Ops: b8CountingHost(counters)},
		AppScreenOptions{
			InstallPreview: fakePreviewOp(b8AllUpToDate("token-review-noop"), "token-review-noop", nil),
			CatalogRefresh: func(cfg TUIConfig) (Catalog, error) {
				refreshes++
				scope = catalogScope{Home: cfg.Home, Repo: cfg.Repo, WithOpenCode: cfg.WithOpenCode, OpenCodeFreeFirst: cfg.OpenCodeFreeFirst}
				return refreshed, nil
			},
		},
	)
	app.inventoryErr = "stale inventory read failed"
	if got := DeriveSetupState(app.catalog, app.inventoryErr).Kind; got != SetupStateReadFailure {
		t.Fatalf("precondition state = %q, want read failure", got)
	}

	app = b8OpenInstallPreview(t, app)
	if !app.install.PreviewNoOp() {
		t.Fatalf("preview step = %v, not the no-op panel", app.install.Step())
	}
	if refreshes != 1 {
		t.Fatalf("no-op refreshes = %d, want exactly 1", refreshes)
	}
	if scope.Home != home {
		t.Fatalf("refresh scope home = %q, want %q", scope.Home, home)
	}
	if app.inventoryPending {
		t.Fatal("no-op refresh stayed pending")
	}
	if app.inventoryErr != "" {
		t.Fatalf("a successful no-op preview left the stale inventory error: %q", app.inventoryErr)
	}
	if got := DeriveSetupState(app.catalog, app.inventoryErr).Kind; got != SetupStateCurrent {
		t.Fatalf("setup state after no-op preview = %q, want current", got)
	}
	for _, change := range app.catalog.Items[0].Clients {
		if change.Status != skillsync.InventoryStatusMatchesSource {
			t.Fatalf("refreshed status = %q", change.Status)
		}
	}
	if counters.applies != 0 || counters.restores != 0 || counters.logs != 0 {
		t.Fatalf("no-op preview touched write seams: applies=%d restores=%d logs=%d",
			counters.applies, counters.restores, counters.logs)
	}
}

func reviewCatalog() Catalog {
	return Catalog{Items: []CatalogItem{
		{ID: "alpha-skill", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: RouterSkillID, Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
	}}
}

// TestRouterRecommendationOverrideDoesNotSurviveEsc opens the recommendation,
// escapes, then opens another entry: the other entry's details must show.
func TestRouterRecommendationOverrideDoesNotSurviveEsc(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		name := "unicode"
		if ascii {
			name = "ascii"
		}
		t.Run(name, func(t *testing.T) {
			cfg := TUIConfig{Width: 70, Height: 24, Color: false, ASCII: ascii}
			app := NewApp(cfg, reviewCatalog())
			next, _ := b8Update(t, app, reviewKey('3'))
			app = next

			next, _ = b8Update(t, app, reviewKey('s'))
			app = next
			if !app.catalogModel.InDetail() {
				t.Fatal("s did not open the recommendation detail")
			}
			if app.catalogModel.detailOverride == nil {
				t.Fatal("s did not set a recommendation override")
			}
			view := ansi.Strip(app.View().Content)
			if !strings.Contains(view, RouterSkillID) || !strings.Contains(view, "SDD Router") {
				t.Fatalf("recommendation detail missing Router facts:\n%s", view)
			}
			if strings.Contains(view, "alpha-skill") {
				t.Fatalf("the list selection replaced the recommendation detail:\n%s", view)
			}

			next, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyEscape})
			app = next
			if app.catalogModel.InDetail() {
				t.Fatal("Esc did not close the detail card")
			}
			if app.catalogModel.detailOverride != nil {
				t.Fatal("Esc kept the recommendation override")
			}

			next, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
			app = next
			if !app.catalogModel.InDetail() {
				t.Fatal("Enter did not open the selected item")
			}
			view = ansi.Strip(app.View().Content)
			if !strings.Contains(view, "alpha-skill") {
				t.Fatalf("the next item's details are missing:\n%s", view)
			}
			if strings.Contains(view, "agentic-sdd-router") {
				t.Fatalf("the recommendation override replaced the next item's details:\n%s", view)
			}
		})
	}
}

// TestBackupDetailsOwnsLowercaseMThroughApp proves the contextual priority:
// in backup details 'm' opens the bounded manifest view without changing the
// global motion setting, and outside details it still toggles motion.
func TestBackupDetailsOwnsLowercaseMThroughApp(t *testing.T) {
	repo, home, backup := syncBackupFixture(t, false)
	app := NewAppWithScreens(
		TUIConfig{Repo: repo, Home: home, Width: 100, Height: 30, Color: false},
		Catalog{Home: home},
		AppOptions{},
		AppScreenOptions{RestoreOps: RestoreOps{
			List: func(r, h string) ([]skillsync.Backup, error) {
				return []skillsync.Backup{backup}, nil
			},
		}},
	)

	next, cmd := b8Update(t, app, reviewKey('b'))
	app = b8Drain(t, next, cmd)
	if app.restore.Stage() != RestoreStageList {
		t.Fatalf("stage = %v, want list", app.restore.Stage())
	}
	next, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	app = next
	if app.restore.Stage() != RestoreStageDetails {
		t.Fatalf("stage = %v, want details", app.restore.Stage())
	}
	// The details action wins from either focus pane.
	next, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyTab})
	app = next

	motion := app.Motion()
	next, manifestCmd := b8Update(t, app, reviewKey('m'))
	app = next
	if app.Motion() != motion {
		t.Fatal("m in backup details toggled the global motion setting")
	}
	if manifestCmd == nil {
		t.Fatal("m in backup details did not open the manifest read")
	}
	app = b8Drain(t, app, manifestCmd)
	text := strings.Join(app.restore.Lines(90), "\n")
	if !strings.Contains(text, "Technical manifest") || !strings.Contains(text, `"format"`) {
		t.Fatalf("manifest view missing after m:\n%s", text)
	}

	// Leaving details restores the global motion meaning of m.
	next, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyTab}) // content focus
	app = next
	next, _ = b8Update(t, app, tea.KeyPressMsg{Code: tea.KeyEscape})
	app = next
	if app.restore.Stage() != RestoreStageList {
		t.Fatalf("stage = %v, want list", app.restore.Stage())
	}
	next, _ = b8Update(t, app, reviewKey('m'))
	app = next
	if app.Motion() == motion {
		t.Fatal("m outside backup details did not toggle motion")
	}
}

// TestCatalogSetFilterRejectsStaleQueryEnvelope covers the programmatic-filter
// finding: SetFilter bumps the generation, and an envelope whose query no
// longer matches the live filter text is discarded even when its generation and
// category still look current.
func TestCatalogSetFilterRejectsStaleQueryEnvelope(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		{ID: "alpha-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "beta-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
	}}
	model := NewCatalogModel(TUIConfig{Width: 120, Height: 24}, cat)
	before := model.filterGeneration

	model.SetFilter("beta")
	if model.filterGeneration == before {
		t.Fatal("SetFilter did not bump the filter generation")
	}
	rows := model.VisibleItems()
	if len(rows) != 1 || rows[0].ID != "beta-thing" {
		t.Fatalf("SetFilter rows = %v, want [beta-thing]", visibleIDs(rows))
	}

	// Same generation and category, stale query: an empty result computed for
	// "alpha" must not replace the programmatic "beta" selection.
	stale := FilterEnvelope{
		Generation: model.filterGeneration,
		Category:   model.ActiveCategory(),
		Query:      "alpha",
		Msg:        list.FilterMatchesMsg{},
	}
	next, _ := model.Update(stale)
	model = next.(CatalogModel)
	rows = model.VisibleItems()
	if len(rows) != 1 || rows[0].ID != "beta-thing" {
		t.Fatalf("stale-query envelope replaced rows: %v", visibleIDs(rows))
	}

	// A matching-query envelope is still applied.
	current := FilterEnvelope{
		Generation: model.filterGeneration,
		Category:   model.ActiveCategory(),
		Query:      "beta",
		Msg:        list.FilterMatchesMsg{},
	}
	next, _ = model.Update(current)
	model = next.(CatalogModel)
	if rows := model.VisibleItems(); len(rows) != 0 {
		t.Fatalf("current envelope rows = %v, want none from the empty result", visibleIDs(rows))
	}
}
