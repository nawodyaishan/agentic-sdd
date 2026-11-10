package tui

// B8/T22+T23 regression tests: grouped exact impact, explicit unknown and
// refused states, and the read-only no-op branch. Every operation is a fake or
// a temporary home; no real home, write, log or network is touched.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

func b8Key(r rune) tea.KeyPressMsg  { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func b8Code(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func b8Update(t *testing.T, app App, msg tea.Msg) (App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(msg)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want tui.App", next)
	}
	return updated, cmd
}

func b8Press(t *testing.T, app App, r rune) App {
	t.Helper()
	app, _ = b8Update(t, app, b8Key(r))
	return app
}

func b8PressName(t *testing.T, app App, name string) (App, tea.Cmd) {
	t.Helper()
	var msg tea.KeyPressMsg
	switch name {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "space":
		msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	default:
		msg = b8Key(rune(name[0]))
	}
	return b8Update(t, app, msg)
}

func b8Drain(t *testing.T, app App, cmd tea.Cmd) App {
	t.Helper()
	if cmd == nil {
		return app
	}
	msg := cmd()
	if msg == nil {
		return app
	}
	next, follow := b8Update(t, app, msg)
	if follow != nil {
		return b8Drain(t, next, follow)
	}
	return next
}

func b8PreviewModel(t *testing.T, planned []skillsync.ResultChange, token string, previewErr error) InstallModel {
	t.Helper()
	m := NewInstallModel(TUIConfig{Home: t.TempDir(), Width: 100, Height: 24})
	m.SetPreviewOp(fakePreviewOp(planned, token, previewErr))
	m, _ = pressKey(m, "enter") // options
	m, _ = pressKey(m, "enter") // review
	m, cmd := pressKey(m, "enter")
	if cmd == nil {
		t.Fatal("expected a preview command")
	}
	updated, _ := m.Update(cmd())
	return updated.(InstallModel)
}

func b8App(t *testing.T, cfg TUIConfig, host *OpsHost, preview PreviewOp) App {
	t.Helper()
	if cfg.Home == "" {
		cfg.Home = t.TempDir()
	}
	cfg = cfg.Normalize()
	return NewAppWithScreens(cfg, Catalog{Home: cfg.Home}, AppOptions{Ops: host}, AppScreenOptions{
		InstallPreview: preview,
	})
}

// b8OpenInstallPreview walks Home → hub → clients → options → review → preview.
func b8OpenInstallPreview(t *testing.T, app App) App {
	t.Helper()
	app = b8Press(t, app, '2')
	app = b8Press(t, app, 'i')
	app, _ = b8Update(t, app, b8Code(tea.KeyEnter))
	app, _ = b8Update(t, app, b8Code(tea.KeyEnter))
	app, cmd := b8Update(t, app, b8Code(tea.KeyEnter))
	return b8Drain(t, app, cmd)
}

func b8AllUpToDate(token string) []skillsync.ResultChange {
	return []skillsync.ResultChange{
		{Client: "claude", ID: "router", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.claude/skills/router", Action: skillsync.ChangeUpToDate},
		{Client: "agents", ID: "planner", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.agents/skills/planner", Action: skillsync.ChangeUpToDate},
	}
}

// ---------------------------------------------------------------------------
// T22 presentation
// ---------------------------------------------------------------------------

func TestB8InstallPreviewLeadsWithTotalsAndDestinationGroups(t *testing.T) {
	planned := b8MixedPlan()
	m := b8PreviewModel(t, planned, "token-b8-grouped", nil)

	plain := ansi.Strip(m.Render(100, 0))
	for _, want := range []string{
		"Impact totals:",
		"7 planned",
		"2 new",
		"2 replace (backup first)",
		"1 remove",
		"1 unchanged",
		"1 unknown",
		"Claude Code (claude) — /tmp/b8-home/.claude/skills",
		"Shared Agents Skills (agents) — /tmp/b8-home/.agents/skills",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("preview missing %q:\n%s", want, plain)
		}
	}
	// Details start collapsed: paths and item rows wait behind the toggle.
	for _, change := range planned {
		if strings.Contains(plain, change.Path) {
			t.Fatalf("collapsed preview already exposes path %q:\n%s", change.Path, plain)
		}
	}

	m, _ = pressKey(m, "x")
	plain = ansi.Strip(m.Render(100, 0))
	for _, change := range planned {
		if !strings.Contains(plain, change.Path) {
			t.Fatalf("expanded preview missing path %q:\n%s", change.Path, plain)
		}
		if !strings.Contains(plain, change.ID) {
			t.Fatalf("expanded preview missing item %q", change.ID)
		}
	}
	// Each item appears exactly once in the complete expanded render.
	for _, change := range planned {
		if n := strings.Count(plain, change.Path); n != 1 {
			t.Fatalf("path %q appears %d times, want exactly once", change.Path, n)
		}
	}

	m, _ = pressKey(m, "x")
	plain = ansi.Strip(m.Render(100, 0))
	if strings.Contains(plain, planned[0].Path) {
		t.Fatal("second x did not collapse the item paths again")
	}
}

func TestB8PreviewGroupSelectionExpandsOneDestination(t *testing.T) {
	m := b8PreviewModel(t, b8MixedPlan(), "token-b8-groups", nil)

	m, _ = pressKey(m, "]") // select the second destination
	m, _ = pressKey(m, " ")
	plain := ansi.Strip(m.Render(100, 0))
	if !strings.Contains(plain, "/tmp/b8-home/.agents/skills/router") {
		t.Fatalf("selected group did not expand:\n%s", plain)
	}
	if strings.Contains(plain, "/tmp/b8-home/.claude/skills/router") {
		t.Fatalf("space expanded every group instead of the selected one:\n%s", plain)
	}

	// Selection and expansion are clamped/presentational only.
	m, _ = pressKey(m, "[")
	m, _ = pressKey(m, " ")
	m, _ = pressKey(m, "[")
	m, _ = pressKey(m, "[")
	if m.impactCursor != 0 {
		t.Fatalf("impact cursor = %d, want clamped to 0", m.impactCursor)
	}
}

func TestB8UnknownActionIsExplicitAndBlocksConfirmation(t *testing.T) {
	planned := []skillsync.ResultChange{
		{Client: "claude", ID: "router", Path: "/tmp/b8-home/.claude/skills/router", Action: skillsync.ChangeInstall},
		{Client: "claude", ID: "future", Path: "/tmp/b8-home/.claude/skills/future", Action: skillsync.ChangeAction("reticulate")},
	}
	m := b8PreviewModel(t, planned, "token-b8-unknown", nil)

	if m.PreviewNoOp() {
		t.Fatal("a plan with an unknown action was treated as a no-op")
	}
	plain := ansi.Strip(m.Render(100, 0))
	if !strings.Contains(plain, "1 unknown") {
		t.Fatalf("unknown action total is not explicit:\n%s", plain)
	}
	if strings.Contains(plain, "2 new") {
		t.Fatalf("unknown action was silently counted as an addition:\n%s", plain)
	}

	m, _ = pressKey(m, "x")
	plain = ansi.Strip(m.Render(100, 0))
	if !strings.Contains(plain, `unknown action "reticulate"`) {
		t.Fatalf("unknown action kind is not rendered explicitly:\n%s", plain)
	}
	if !strings.Contains(plain, "Confirmation is disabled") {
		t.Fatalf("unknown action did not disable confirmation:\n%s", plain)
	}

	// Enter cannot advance to the confirmation while an unknown action exists.
	m, cmd := pressKey(m, "enter")
	if cmd != nil {
		t.Fatal("Enter produced a command for an unclassifiable plan")
	}
	if m.Step() != InstallStepPreview {
		t.Fatalf("step = %v, want preview while an unknown action is unresolved", m.Step())
	}
	if m.PreviewToken() != "token-b8-unknown" {
		t.Fatal("blocking confirmation changed the authoritative preview token")
	}
}

func TestB8IncompleteOrRefusedPreviewNeverLooksSuccessful(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		m := b8PreviewModel(t, b8AllUpToDate(""), "", nil)
		if m.PreviewNoOp() {
			t.Fatal("a preview without a token was treated as a no-op")
		}
		plain := ansi.Strip(m.Render(100, 0))
		if strings.Contains(plain, "Already up to date") {
			t.Fatalf("an incomplete preview claimed success:\n%s", plain)
		}
		if !strings.Contains(plain, "not confirmable") {
			t.Fatalf("an incomplete preview is not labelled incomplete:\n%s", plain)
		}
		m, cmd := pressKey(m, "enter")
		if cmd != nil || m.Step() != InstallStepPreview {
			t.Fatalf("incomplete preview advanced to %v", m.Step())
		}
	})

	t.Run("empty plan", func(t *testing.T) {
		m := b8PreviewModel(t, nil, "token-b8-empty", nil)
		if m.PreviewNoOp() {
			t.Fatal("an empty plan was treated as a successful no-op")
		}
		if strings.Contains(ansi.Strip(m.Render(100, 0)), "Already up to date") {
			t.Fatal("an empty plan claimed Already up to date")
		}
	})

	t.Run("failed outcome without error", func(t *testing.T) {
		m := b8PreviewModel(t, b8AllUpToDate("token-b8-failed"), "token-b8-failed", nil)
		m.previewResult.Outcome = skillsync.OutcomeFailed
		if m.PreviewNoOp() {
			t.Fatal("a failed preview outcome was treated as a no-op")
		}
		plain := ansi.Strip(m.Render(100, 0))
		if !strings.Contains(plain, "Preview refused") || strings.Contains(plain, "Already up to date") {
			t.Fatalf("failed preview outcome was not refused explicitly:\n%s", plain)
		}
		m, cmd := pressKey(m, "enter")
		if cmd != nil || m.Step() != InstallStepPreview {
			t.Fatalf("failed preview outcome advanced to %v", m.Step())
		}
	})

	t.Run("refused", func(t *testing.T) {
		m := b8PreviewModel(t, b8AllUpToDate("token-b8-refused"), "token-b8-refused", errors.New("destination inspection refused"))
		if m.PreviewNoOp() {
			t.Fatal("a refused preview was treated as a no-op")
		}
		plain := ansi.Strip(m.Render(100, 0))
		if !strings.Contains(plain, "Preview Error") {
			t.Fatalf("a refused preview does not show its error:\n%s", plain)
		}
		if strings.Contains(plain, "Already up to date") {
			t.Fatalf("a refused preview claimed success:\n%s", plain)
		}
		m, cmd := pressKey(m, "enter")
		if cmd != nil || m.Step() != InstallStepPreview {
			t.Fatalf("refused preview advanced to %v", m.Step())
		}
	})
}

// ---------------------------------------------------------------------------
// T23 read-only no-op
// ---------------------------------------------------------------------------

func TestB8NoOpPreviewIsReadOnlyAndRequiresNoConfirmation(t *testing.T) {
	m := b8PreviewModel(t, b8AllUpToDate("token-b8-noop"), "token-b8-noop", nil)
	if !m.PreviewNoOp() {
		t.Fatal("complete all-up-to-date preview was not classified as a no-op")
	}

	plain := ansi.Strip(m.Render(100, 0))
	for _, want := range []string{"Already up to date", "no confirmation is offered", "no backup", "no operation log", "Source:", "Target home:"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("no-op preview missing %q:\n%s", want, plain)
		}
	}
	for _, forbidden := range []string{"Proceed with installation?", "y confirm", "Enter: proceed to confirmation"} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("no-op preview offered confirmation %q:\n%s", forbidden, plain)
		}
	}

	// Read-only details keep every path reachable.
	m, _ = pressKey(m, "x")
	plain = ansi.Strip(m.Render(100, 0))
	for _, change := range b8AllUpToDate("token-b8-noop") {
		if !strings.Contains(plain, change.Path) {
			t.Fatalf("no-op details lost path %q", change.Path)
		}
	}

	// Every input stays read-only: no confirmation and no write host command.
	for _, press := range []string{"enter", "y", "Y", "n", " ", "x", "j", "pgdown"} {
		next, cmd := pressKey(m, press)
		if cmd != nil {
			t.Fatalf("%q produced a command in the no-op preview", press)
		}
		m = next
		if m.Step() != InstallStepPreview {
			t.Fatalf("%q changed the step to %v", press, m.Step())
		}
	}
	if m.PreviewToken() != "token-b8-noop" {
		t.Fatal("read-only input changed the preview token")
	}
}

func TestB8NoOpChangeOptionsInvalidatesPreview(t *testing.T) {
	m := b8PreviewModel(t, b8AllUpToDate("token-b8-options"), "token-b8-options", nil)
	m, cmd := pressKey(m, "o")
	if cmd != nil || m.Step() != InstallStepOptions {
		t.Fatalf("o: step=%v cmd nil=%t, want options", m.Step(), cmd == nil)
	}
	if m.PreviewToken() != "" || m.PreviewResult() != nil {
		t.Fatal("changing options kept the no-op preview authorization")
	}
}

func TestB8ChangedLaterPreviewRequiresNormalConfirmation(t *testing.T) {
	m := NewInstallModel(TUIConfig{Home: t.TempDir(), Width: 100, Height: 24})
	calls := 0
	m.SetPreviewOp(func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error) {
		calls++
		if calls == 1 {
			return appops.PreviewResult{Outcome: skillsync.OutcomePreview, PreviewToken: "token-first-noop", Planned: b8AllUpToDate("token-first-noop")}, nil
		}
		return appops.PreviewResult{Outcome: skillsync.OutcomePreview, PreviewToken: "token-second-mutation", Planned: []skillsync.ResultChange{
			{Client: "claude", ID: "router", Path: "/tmp/b8-home/.claude/skills/router", Action: skillsync.ChangeInstall},
		}}, nil
	})
	m, _ = pressKey(m, "enter") // options
	m, _ = pressKey(m, "enter") // review
	m, cmd := pressKey(m, "enter")
	updated, _ := m.Update(cmd())
	m = updated.(InstallModel)
	if !m.PreviewNoOp() {
		t.Fatal("first preview was not a no-op")
	}

	// Leaving the no-op preview forces a fresh preview before any consent.
	m, _ = pressKey(m, "esc")
	if m.Step() != InstallStepReview || m.PreviewToken() != "" {
		t.Fatalf("leaving no-op kept step=%v token=%q, want review with no consent", m.Step(), m.PreviewToken())
	}
	m, cmd = pressKey(m, "enter")
	updated, _ = m.Update(cmd())
	m = updated.(InstallModel)
	if m.PreviewNoOp() {
		t.Fatal("the changed second preview was still treated as a no-op")
	}
	if m.PreviewToken() != "token-second-mutation" {
		t.Fatalf("token = %q, want the fresh plan token", m.PreviewToken())
	}
	// The changed plan must use the normal default-No confirmation.
	m, _ = pressKey(m, "enter")
	if m.Step() != InstallStepConfirm {
		t.Fatalf("changed preview step = %v, want confirm", m.Step())
	}
	m, cmd = pressKey(m, "enter")
	if cmd != nil {
		t.Fatal("Enter confirmed the changed plan")
	}
	m, cmd = pressKey(m, "y")
	if cmd == nil {
		t.Fatal("y did not produce the confirmed write request")
	}
	if _, ok := cmd().(StartWriteMsg); !ok {
		t.Fatal("y did not emit a StartWriteMsg")
	}
}

// b8WriteLog is an in-memory log seam; it records that a log was opened.
type b8WriteLog struct{}

func (b8WriteLog) Write(p []byte) (int, error) { return len(p), nil }
func (b8WriteLog) Flush() error                { return nil }
func (b8WriteLog) Close() error                { return nil }

type b8Counters struct {
	applies  int32
	restores int32
	logs     int32
}

func b8CountingHost(counters *b8Counters) *OpsHost {
	return NewOpsHost(OpsHostOptions{
		ApplyInstall: func(req appops.InstallRequest, token string, out io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&counters.applies, 1)
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		ApplyRestore: func(req appops.RestoreRequest, token string, out io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&counters.restores, 1)
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		OpenLog: func(kind WriteKind, home string) (WriteLog, string, error) {
			atomic.AddInt32(&counters.logs, 1)
			return b8WriteLog{}, "/tmp/b8-fake.log", nil
		},
	})
}

func TestB8NoOpAppNeverCallsTheWriteHost(t *testing.T) {
	counters := &b8Counters{}
	app := b8App(t, TUIConfig{Width: 100, Height: 30}, b8CountingHost(counters), fakePreviewOp(b8AllUpToDate("token-app-noop"), "token-app-noop", nil))
	app = b8OpenInstallPreview(t, app)
	token := app.install.PreviewToken()
	if !app.install.PreviewNoOp() {
		t.Fatal("app preview was not classified as a no-op")
	}
	view := ansi.Strip(app.View().Content)
	if !strings.Contains(view, "Already up to date") {
		t.Fatalf("app no-op view missing the neutral status:\n%s", view)
	}

	for _, press := range []string{"enter", "y", "Y", "n", "space", "x"} {
		next, cmd := b8PressName(t, app, press)
		app = next
		if cmd != nil {
			if msg := cmd(); msg != nil {
				if _, write := msg.(StartWriteMsg); write {
					t.Fatalf("%q submitted a write from the no-op preview", press)
				}
			}
		}
		if app.install.Step() != InstallStepPreview {
			t.Fatalf("%q left the no-op preview for %v", press, app.install.Step())
		}
	}
	if app.Busy() {
		t.Fatal("no-op preview entered a busy write state")
	}
	if got := atomic.LoadInt32(&counters.applies); got != 0 {
		t.Fatalf("apply seam called %d times from a no-op preview", got)
	}
	if got := atomic.LoadInt32(&counters.logs); got != 0 {
		t.Fatalf("operation log opened %d times from a no-op preview", got)
	}
	if app.install.PreviewToken() != token {
		t.Fatal("no-op input changed the preview token")
	}
}

func TestB8NoOpHomeAndLibraryUseExistingRoutes(t *testing.T) {
	preview := fakePreviewOp(b8AllUpToDate("token-routes"), "token-routes", nil)
	app := b8App(t, TUIConfig{Width: 100, Height: 30}, b8CountingHost(&b8Counters{}), preview)
	app = b8OpenInstallPreview(t, app)

	app = b8Press(t, app, 'h')
	if app.ActiveRoute() != RouteHome {
		t.Fatalf("h route = %v, want Home", app.ActiveRoute())
	}
	if app.install.Step() != InstallStepReview || app.install.PreviewToken() != "" {
		t.Fatalf("leaving the no-op preview kept authorization: step=%v token=%q", app.install.Step(), app.install.PreviewToken())
	}

	app = b8Press(t, app, '2') // back to Install & Manage
	app, cmd := b8Update(t, app, b8Code(tea.KeyEnter))
	app = b8Drain(t, app, cmd) // fresh preview from Review
	if !app.install.PreviewNoOp() {
		t.Fatalf("fresh preview step=%v is not the no-op panel", app.install.Step())
	}
	app = b8Press(t, app, 'l')
	if app.ActiveRoute() != RouteExplore {
		t.Fatalf("l route = %v, want Explore", app.ActiveRoute())
	}
}

// snapshotTree captures every path, mode and file size below root so a no-op
// can be proven byte-for-byte unchanged without reading real client data.
func b8SnapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		entry := path + "|" + info.Mode().String()
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry += "|" + string(data)
		}
		out = append(out, entry)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}

func TestB8RealNoOpLeavesTemporaryHomeUnchanged(t *testing.T) {
	home := t.TempDir()
	if _, err := skillsync.SyncWithOptionsResult("", home, true, skillsync.Options{}, io.Discard); err != nil {
		t.Fatalf("seed install: %v", err)
	}
	before := b8SnapshotTree(t, home)

	counters := &b8Counters{}
	app := b8App(t, TUIConfig{Width: 100, Height: 30, Home: home}, b8CountingHost(counters), appops.PreviewInstall)
	app = b8OpenInstallPreview(t, app)
	if !app.install.PreviewNoOp() {
		preview := app.install.PreviewResult()
		var planned int
		if preview != nil {
			planned = len(preview.Planned)
		}
		t.Fatalf("real second preview on the same home was not a no-op (planned=%d)", planned)
	}
	for _, press := range []string{"enter", "y", "Y", "space", "j"} {
		next, cmd := b8PressName(t, app, press)
		app = next
		if cmd != nil {
			if msg := cmd(); msg != nil {
				if _, write := msg.(StartWriteMsg); write {
					t.Fatalf("%q wrote from the no-op preview", press)
				}
			}
		}
	}
	after := b8SnapshotTree(t, home)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatal("the no-op preview changed the temporary home tree")
	}
	if got := atomic.LoadInt32(&counters.applies); got != 0 {
		t.Fatalf("apply seam called %d times", got)
	}
	if got := atomic.LoadInt32(&counters.logs); got != 0 {
		t.Fatalf("operation log opened %d times", got)
	}
}

func TestB8PinnedSummaryWhileStressDetailsPage(t *testing.T) {
	clients := []string{"agents", "codex", "claude", "agy"}
	var planned []skillsync.ResultChange
	for i := 0; i < 94; i++ {
		planned = append(planned, skillsync.ResultChange{
			Client: clients[i%len(clients)],
			ID:     fmt.Sprintf("skill-%02d", i),
			Kind:   skillsync.InventoryKindSkill,
			Path:   fmt.Sprintf("/tmp/b8-home/.%s/skills/very-long-directory-name-%02d-日本語", clients[i%len(clients)], i),
			Action: skillsync.ChangeInstall,
		})
	}
	app := b8App(t, TUIConfig{Width: 100, Height: 24}, b8CountingHost(&b8Counters{}), fakePreviewOp(planned, "token-stress", nil))
	app = b8OpenInstallPreview(t, app)
	app = b8Press(t, app, 'x') // expand every destination

	view := ansi.Strip(app.View().Content)
	if !strings.Contains(view, "Impact totals") || !strings.Contains(view, "94 planned") {
		t.Fatalf("summary missing before paging:\n%s", view)
	}
	projection := projectImpact(planned)
	lastGroup := projection.groups[len(projection.groups)-1]
	lastPath := lastGroup.items[len(lastGroup.items)-1].change.Path
	if strings.Contains(view, lastPath) {
		t.Fatal("the last path was already visible before paging")
	}

	for i := 0; i < 80; i++ {
		app, _ = b8Update(t, app, b8Code(tea.KeyPgDown))
	}
	view = ansi.Strip(app.View().Content)
	if !strings.Contains(view, "Impact totals") || !strings.Contains(view, "94 planned") {
		t.Fatalf("pinned totals scrolled out of view:\n%s", view)
	}
	if !strings.Contains(view, lastPath) {
		t.Fatalf("paging never reached the last planned path %q:\n%s", lastPath, view)
	}
}

// TestB8PreviewBoundsAcrossBreakpoints keeps the grouped preview and its
// expanded long Unicode paths inside every approved width/height boundary.
func TestB8PreviewBoundsAcrossBreakpoints(t *testing.T) {
	clients := []string{"agents", "codex", "claude", "agy"}
	var planned []skillsync.ResultChange
	for i := 0; i < 94; i++ {
		planned = append(planned, skillsync.ResultChange{
			Client: clients[i%len(clients)],
			ID:     fmt.Sprintf("skill-%02d", i),
			Path:   fmt.Sprintf("/tmp/b8-用户-home/résumé/.%s/skills/very-long-路径-%02d", clients[i%len(clients)], i),
			Action: skillsync.ChangeInstall,
		})
	}
	for _, width := range []int{40, 60, 80, 110} {
		for _, height := range []int{8, 16, 24} {
			for _, cfg := range []TUIConfig{
				{Width: width, Height: height, Color: true, Home: t.TempDir()},
				{Width: width, Height: height, Color: true, ASCII: true, Home: t.TempDir()},
			} {
				app := b8App(t, cfg, b8CountingHost(&b8Counters{}), fakePreviewOp(planned, "token-bounds", nil))
				app = b8OpenInstallPreview(t, app)
				app = b8Press(t, app, 'x')
				for i := 0; i < 2; i++ {
					app, _ = b8Update(t, app, b8Code(tea.KeyPgDown))
				}
				view := app.View().Content
				lines := strings.Split(view, "\n")
				if len(lines) != height {
					t.Fatalf("w=%d h=%d ascii=%t rows = %d, want %d", width, height, cfg.ASCII, len(lines), height)
				}
				for row, line := range lines {
					if got := ansi.StringWidth(line); got > width {
						t.Fatalf("w=%d h=%d ascii=%t row %d width %d > %d: %q", width, height, cfg.ASCII, row, got, width, ansi.Strip(line))
					}
					if cfg.ASCII {
						for _, r := range line {
							if r > 127 {
								t.Fatalf("w=%d h=%d ascii row %d is not 7-bit: %q", width, height, row, line)
							}
						}
					}
				}
			}
		}
	}
}

func TestB8PreviewIdentityIsPresentationOnly(t *testing.T) {
	planned := b8MixedPlan()
	m := b8PreviewModel(t, planned, "token-identity", nil)
	token := m.PreviewToken()
	before := append([]skillsync.ResultChange(nil), m.PreviewResult().Planned...)

	m, _ = pressKey(m, "x")
	m, _ = pressKey(m, "]")
	m, _ = pressKey(m, " ")
	m.SetSize(60, 16)
	_ = m.Render(60, 16)
	m.SetSize(120, 30)
	_ = m.Render(120, 30)

	if m.PreviewToken() != token {
		t.Fatal("presentation changed the preview token")
	}
	after := m.PreviewResult().Planned
	if len(after) != len(before) {
		t.Fatalf("planned length changed from %d to %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("planned[%d] changed: %+v != %+v", i, before[i], after[i])
		}
	}
}
