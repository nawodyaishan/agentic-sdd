package tui

// B9 T25–T27 focused coverage: goal-first Home, plain labels with exact
// identities, typed outcome text and generation-bound inventory refresh. Every
// fixture is local and read-only; no test reads a real home, contacts a network
// or writes a destination.

import (
	"errors"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

var b9SupportedClients = []string{"agents", "codex", "claude", "agy"}

// b9CoreCatalog builds one core-skill row per current client status. A client
// missing from the map is reported missing, never silently omitted.
func b9CoreCatalog(statusByClient map[string]skillsync.InventoryStatus) Catalog {
	item := CatalogItem{ID: "sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore}
	for _, id := range b9SupportedClients {
		status, ok := statusByClient[id]
		if !ok {
			status = skillsync.InventoryStatusMissing
		}
		item.Clients = append(item.Clients, ClientStatus{Client: id, Status: status, Path: "/home/test/" + id + "/skills/sdd-router"})
	}
	return Catalog{Home: "/home/test", Items: []CatalogItem{item}}
}

func b9All(status skillsync.InventoryStatus) map[string]skillsync.InventoryStatus {
	out := make(map[string]skillsync.InventoryStatus, len(b9SupportedClients))
	for _, id := range b9SupportedClients {
		out[id] = status
	}
	return out
}

func b9HomeApp(catalog Catalog) App {
	return NewApp(TUIConfig{Width: 100, Height: 40, Color: false}, catalog)
}

// b9Flat normalizes whitespace but keeps word boundaries, so wrapped prose can
// be compared without losing the spaces that make a phrase readable.
func b9Flat(text string) string { return strings.Join(strings.Fields(text), " ") }

func TestDeriveSetupStateClassifiesLocalEvidence(t *testing.T) {
	partial := map[string]skillsync.InventoryStatus{
		"agents": skillsync.InventoryStatusMatchesSource,
		"codex":  skillsync.InventoryStatusMatchesSource,
		"claude": skillsync.InventoryStatusMissing,
		"agy":    skillsync.InventoryStatusMissing,
	}
	modified := map[string]skillsync.InventoryStatus{
		"agents": skillsync.InventoryStatusMatchesSource,
		"codex":  skillsync.InventoryStatusMatchesSource,
		"claude": skillsync.InventoryStatusMatchesSource,
		"agy":    skillsync.InventoryStatusDiffersFromSource,
	}
	cases := []struct {
		name    string
		catalog Catalog
		readErr string
		want    SetupStateKind
	}{
		{"first use", b9CoreCatalog(nil), "", SetupStateFirstUse},
		{"current", b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource)), "", SetupStateCurrent},
		{"partial", b9CoreCatalog(partial), "", SetupStatePartial},
		{"modified", b9CoreCatalog(modified), "", SetupStateModified},
		{"unknown", Catalog{}, "", SetupStateUnknown},
		{"read failure", b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource)), "permission denied", SetupStateReadFailure},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveSetupState(tc.catalog, tc.readErr)
			if got.Kind != tc.want {
				t.Fatalf("state = %q, want %q (evidence %+v)", got.Kind, tc.want, got.Evidence)
			}
			if tc.readErr != "" && got.ReadErr != tc.readErr {
				t.Fatalf("ReadErr = %q, want %q", got.ReadErr, tc.readErr)
			}
		})
	}
}

// TestHomeLeadsWithPurposeThenSetupThenOneAction proves the hierarchy and that
// counts are derived from the snapshot, not fixed product numbers.
func TestHomeLeadsWithPurposeThenSetupThenOneAction(t *testing.T) {
	app := b9HomeApp(b9CoreCatalog(nil))
	text := b9Flat(strings.Join(app.homeLines(90), "\n"))

	purpose := strings.Index(text, b9Flat(HomePurposeLines()[0]))
	setup := strings.Index(text, b9Flat("Recommended next step"))
	inventory := strings.Index(text, b9Flat("Local inventory"))
	if purpose < 0 || setup < 0 || inventory < 0 {
		t.Fatalf("Home is missing purpose, setup or inventory:\n%s", text)
	}
	if !(purpose < setup && setup < inventory) {
		t.Fatalf("Home order is not purpose → setup/action → inventory: %d %d %d\n%s", purpose, setup, inventory, text)
	}
	for _, want := range []string{
		"Nothing installed in this home yet.",
		"Set up / Install Agentic-SDD",
		"Browse Skills (key 3)",
		"Recovery (key b)",
		"0/4 core destinations present",
	} {
		if !strings.Contains(text, b9Flat(want)) {
			t.Fatalf("Home missing %q:\n%s", want, text)
		}
	}
	if got := strings.Count(text, "Set up / Install Agentic-SDD"); got != 1 {
		t.Fatalf("Home has %d primary setup actions, want exactly 1:\n%s", got, text)
	}
	// The old action block is gone: only the composed primary action exists.
	if strings.Contains(text, "Recommended next step Set up / Install Agentic-SDD Read-only local inspection") {
		t.Fatalf("Home kept a second, historical action block:\n%s", text)
	}
}

func TestHomeStatesUseEvidenceAndOneAction(t *testing.T) {
	cases := []struct {
		name       string
		catalog    Catalog
		readErr    string
		wantState  string
		wantAction string
		wantCount  string
	}{
		{"first use", b9CoreCatalog(nil), "", "Nothing installed in this home yet.", "Set up / Install Agentic-SDD", "0/4 core destinations present"},
		{"current", b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource)), "", "Current — every core skill destination matches", "Manage installation", "4/4 core destinations present"},
		{"partial", b9CoreCatalog(map[string]skillsync.InventoryStatus{
			"agents": skillsync.InventoryStatusMatchesSource,
			"codex":  skillsync.InventoryStatusMatchesSource,
		}), "", "Partly installed", "Set up / Install Agentic-SDD", "2/4 core destinations present"},
		{"modified", b9CoreCatalog(map[string]skillsync.InventoryStatus{
			"agents": skillsync.InventoryStatusMatchesSource,
			"codex":  skillsync.InventoryStatusMatchesSource,
			"claude": skillsync.InventoryStatusMatchesSource,
			"agy":    skillsync.InventoryStatusDiffersFromSource,
		}), "", "Installed with local modifications.", "Set up / Install Agentic-SDD", "4/4 core destinations present"},
		{"unknown", Catalog{}, "", "Unknown — the local snapshot cannot support", "Set up / Install Agentic-SDD", "no core client destinations were reported"},
		{"read failure", b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource)), "permission denied", "Unknown — the last local inventory read failed.", "Set up / Install Agentic-SDD", "4/4 core destinations present"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := b9HomeApp(tc.catalog)
			app.inventoryErr = tc.readErr
			text := b9Flat(strings.Join(app.homeLines(90), "\n"))
			for _, want := range []string{tc.wantState, tc.wantAction, tc.wantCount} {
				if !strings.Contains(text, b9Flat(want)) {
					t.Fatalf("Home missing %q:\n%s", want, text)
				}
			}
			if tc.readErr != "" {
				if !strings.Contains(text, b9Flat("permission denied")) {
					t.Fatalf("read failure detail is not shown:\n%s", text)
				}
				if strings.Contains(text, "operation outcome") == false {
					t.Fatalf("read failure is not separated from the operation outcome:\n%s", text)
				}
			}
		})
	}
}

func TestHomeKeyboardPrimaryAndRecovery(t *testing.T) {
	app := b9HomeApp(b9CoreCatalog(nil))
	next, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = next.(App)
	if app.ActiveRoute() != RouteInstall || app.installSection != installSectionHub {
		t.Fatalf("Enter on default Home = route %v section %v, want install hub", app.ActiveRoute(), app.installSection)
	}

	app = b9HomeApp(b9CoreCatalog(nil))
	next, _ = app.Update(tea.KeyPressMsg{Code: 's'})
	app = next.(App)
	if app.ActiveRoute() != RouteInstall || app.installSection != installSectionHub {
		t.Fatalf("s = route %v section %v, want install hub", app.ActiveRoute(), app.installSection)
	}

	app = b9HomeApp(b9CoreCatalog(nil))
	next, _ = app.Update(tea.KeyPressMsg{Code: 'b'})
	app = next.(App)
	if app.ActiveRoute() != RouteInstall || app.installSection != installSectionBackups {
		t.Fatalf("b = route %v section %v, want recovery list", app.ActiveRoute(), app.installSection)
	}
}

func TestHomeNoForcedOnboardingOrNetworkState(t *testing.T) {
	app := b9HomeApp(b9CoreCatalog(nil))
	app.inventoryErr = "permission denied"
	text := strings.ToLower(strings.Join(app.homeLines(90), "\n"))
	for _, forbidden := range []string{"authenticated", "connected", "signed in", "provider", "sandbox"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Home claimed %q:\n%s", forbidden, text)
		}
	}
	if app.onboarding != nil {
		t.Fatal("Home opened onboarding without a user action")
	}
}

// TestPlainLabelsKeepEngineIdentityAndDestinations covers selection, review and
// confirmation: the plain name leads, and the engine ID plus exact path remain
// visible where the decision is made.
func TestPlainLabelsKeepEngineIdentityAndDestinations(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	if err := m.SetClients([]string{"agents", "claude"}); err != nil {
		t.Fatalf("SetClients: %v", err)
	}

	m.step = InstallStepClients
	view := m.Render(100, 0)
	for _, want := range []string{
		"Shared Agents Skills — selected",
		"id agents · destination /tmp/fakehome/.agents/skills",
		"Claude Code — selected",
		"id claude · destination /tmp/fakehome/.claude/skills",
		"Shared Agents Skills is one shared skills directory, not a provider, account or runtime.",
		"At least one client is required",
		"no pack-only installation is offered",
	} {
		if !strings.Contains(b9Flat(view), b9Flat(want)) {
			t.Fatalf("client selection missing %q:\n%s", want, view)
		}
	}

	m.step = InstallStepReview
	view = m.Render(100, 0)
	for _, want := range []string{
		"Shared Agents Skills (agents) → /tmp/fakehome/.agents/skills",
		"Claude Code (claude) → /tmp/fakehome/.claude/skills",
		"Source repository:",
		"Target home directory: /tmp/fakehome",
	} {
		if !strings.Contains(b9Flat(view), b9Flat(want)) {
			t.Fatalf("review missing %q:\n%s", want, view)
		}
	}

	m.step = InstallStepConfirm
	view = m.Render(110, 0)
	for _, want := range []string{
		"Selected clients: Shared Agents Skills (agents), Claude Code (claude)",
		"Shared Agents Skills (agents) → /tmp/fakehome/.agents/skills",
		"Claude Code (claude) → /tmp/fakehome/.claude/skills",
	} {
		if !strings.Contains(b9Flat(view), b9Flat(want)) {
			t.Fatalf("confirmation missing %q:\n%s", want, view)
		}
	}
}

func TestPreviewAndResultsUsePlainClientLabels(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	if err := m.SetClients([]string{"claude"}); err != nil {
		t.Fatalf("SetClients: %v", err)
	}
	m.step = InstallStepPreview
	m.previewResult = &appops.PreviewResult{
		Outcome: skillsync.OutcomeNoChange,
		Planned: []skillsync.ResultChange{{
			Action: skillsync.ChangeInstall,
			ID:     "sdd-router",
			Client: "claude",
			Path:   "/tmp/fakehome/.claude/skills/sdd-router",
		}},
	}
	m.impactExpanded = map[int]bool{0: true}
	view := m.Render(100, 0)
	flat := b9Flat(view)
	if !strings.Contains(flat, b9Flat("Claude Code (claude)")) {
		t.Fatalf("preview did not use the plain client label:\n%s", view)
	}
	if !strings.Contains(flat, "sdd-router") {
		t.Fatalf("expanded preview did not show the exact item ID:\n%s", view)
	}

	m.step = InstallStepResult
	m.outcome = InstallOutcomeApplied
	m.applyResult = &appops.ApplyResult{
		Outcome: skillsync.OutcomeApplied,
		Applied: []skillsync.ResultChange{{Action: skillsync.ChangeInstall, ID: "sdd-router", Client: "claude"}},
	}
	view = m.Render(100, 0)
	if !strings.Contains(b9Flat(view), b9Flat("Selected clients: Claude Code (claude)")) {
		t.Fatalf("result did not use the plain client label:\n%s", view)
	}
}

func TestExplicitEmptySelectionIsRejected(t *testing.T) {
	m := NewInstallModel(TUIConfig{Home: "/tmp/fakehome"})
	if err := m.SetClients(nil); err == nil {
		t.Fatal("SetClients(nil) accepted an explicit empty selection")
	}

	// Deselect every core client through the UI and try to continue.
	m.step = InstallStepClients
	for i := range m.supportedClients {
		if i > 0 {
			m, _ = pressKey(m, "down")
		}
		m, _ = pressKey(m, " ")
	}
	if got := m.SelectedClients(); len(got) != 0 {
		t.Fatalf("selected clients = %v, want none", got)
	}
	m, _ = pressKey(m, "enter")
	if m.Step() != InstallStepClients {
		t.Fatalf("empty selection advanced to %v", m.Step())
	}
	if !strings.Contains(strings.ToLower(m.ValidationError()), "at least one client must be selected") {
		t.Fatalf("validation = %q", m.ValidationError())
	}
	if _, err := m.installWriteRequest(); !errors.Is(err, ErrWriteInvalid) {
		t.Fatalf("empty selection write request error = %v, want ErrWriteInvalid", err)
	}
}

func TestInstallResultOutcomeText(t *testing.T) {
	cases := []struct {
		name      string
		outcome   InstallOutcome
		result    *appops.ApplyResult
		want      []string
		forbidden []string
	}{
		{
			name:    "installed and updated",
			outcome: InstallOutcomeApplied,
			result: &appops.ApplyResult{
				Outcome: skillsync.OutcomeApplied,
				Applied: []skillsync.ResultChange{
					{Action: skillsync.ChangeInstall, ID: "a"},
					{Action: skillsync.ChangeInstall, ID: "b"},
					{Action: skillsync.ChangeReplace, ID: "c"},
				},
				BackupID: "backup-1", BackupPath: "/tmp/backups/backup-1",
			},
			want: []string{"Installation Applied Successfully", "3 items (2 new, 1 updated)", "New destinations installed: 2", "Existing content updated (backed up first): 1", "Backup created: backup-1", "agentic-sdd-router"},
		},
		{
			name:    "already current",
			outcome: InstallOutcomeNoChange,
			result:  &appops.ApplyResult{Outcome: skillsync.OutcomeNoChange},
			want:    []string{"Already Current", "No files were modified, and no backup was created."},
		},
		{
			name:      "cancelled before write",
			outcome:   InstallOutcomeCancelled,
			want:      []string{"Installation Cancelled", "Cancelled before any write", "No files were modified on disk."},
			forbidden: []string{"Successfully"},
		},
		{
			name:      "refused",
			outcome:   InstallOutcomeRefused,
			want:      []string{"Installation Refused", "fresh preview and confirmation are required"},
			forbidden: []string{"Successfully"},
		},
		{
			name:      "failed rolled back",
			outcome:   InstallOutcomeFailed,
			result:    &appops.ApplyResult{Rollback: skillsync.RollbackInfo{Attempted: true, Complete: true}},
			want:      []string{"Installation Failed (rolled back)", "Rollback status: Succeeded."},
			forbidden: []string{"partial state"},
		},
		{
			name:      "failed without rollback",
			outcome:   InstallOutcomeFailed,
			result:    &appops.ApplyResult{},
			want:      []string{"Installation Failed", "Rollback status: not attempted."},
			forbidden: []string{"Succeeded"},
		},
		{
			name:      "unknown",
			outcome:   InstallOutcomeUnknown,
			result:    &appops.ApplyResult{Outcome: skillsync.OutcomePreview, BackupID: "backup-x"},
			want:      []string{"Installation Result Unknown", "does not claim success or failure", "Backup reference: backup-x"},
			forbidden: []string{"Applied Successfully", "Installation Failed", "rolled back", "No files were modified"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewInstallModel(TUIConfig{Home: "/tmp/fakehome"})
			m.step = InstallStepResult
			m.outcome = tc.outcome
			m.applyResult = tc.result
			m.logPath = "/tmp/logs/write.log"
			text := b9Flat(strings.Join(m.resultLines(100), "\n"))
			for _, want := range tc.want {
				if !strings.Contains(text, b9Flat(want)) {
					t.Fatalf("result missing %q:\n%s", want, text)
				}
			}
			for _, forbidden := range tc.forbidden {
				if strings.Contains(text, b9Flat(forbidden)) {
					t.Fatalf("result claimed %q:\n%s", forbidden, text)
				}
			}
		})
	}
}

func TestUnknownInstallOutcomeIsClassifiedHonestly(t *testing.T) {
	if got := classifyInstallOutcome(appops.ApplyResult{Outcome: skillsync.OutcomePreview}, nil); got != InstallOutcomeUnknown {
		t.Fatalf("outcome = %v, want unknown", got)
	}
	if got := classifyInstallOutcome(appops.ApplyResult{}, errors.New("boom")); got != InstallOutcomeFailed {
		t.Fatalf("error outcome = %v, want failed", got)
	}
}

func TestRestoreUnknownResultText(t *testing.T) {
	m := NewRestoreModel(TUIConfig{Home: "/tmp/fakehome"}, RestoreOps{})
	m.stage = RestoreStageResult
	m.result = &RestoreOutcome{Kind: RestoreOutcomeUnknown, BackupID: "backup-1", LogPath: "/tmp/logs/write.log"}
	text := b9Flat(strings.Join(m.Lines(100), "\n"))
	for _, want := range []string{"Restore result unknown", "does not claim success or failure", "Operation log: /tmp/logs/write.log"} {
		if !strings.Contains(text, b9Flat(want)) {
			t.Fatalf("restore result missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Restore failed.") {
		t.Fatalf("unknown restore was reported as failure:\n%s", text)
	}

	m.result = &RestoreOutcome{Kind: RestoreOutcomeFailed}
	text = b9Flat(strings.Join(m.Lines(100), "\n"))
	if !strings.Contains(text, "Restore failed.") {
		t.Fatalf("failed restore did not report failure:\n%s", text)
	}
	if strings.Contains(text, "Restore result unknown") {
		t.Fatalf("failed restore was reported as unknown:\n%s", text)
	}
}

// b9Log is an in-memory write transcript; the refresh tests never touch disk.
type b9Log struct{}

func (b9Log) Write(p []byte) (int, error) { return len(p), nil }
func (b9Log) Flush() error                { return nil }
func (b9Log) Close() error                { return nil }

func b9RefreshHost(t *testing.T) *OpsHost {
	t.Helper()
	return NewOpsHost(OpsHostOptions{
		OpenLog: func(WriteKind, string) (WriteLog, string, error) { return b9Log{}, "/tmp/logs/write.log", nil },
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{
				Outcome: skillsync.OutcomeApplied,
				Applied: []skillsync.ResultChange{{Action: skillsync.ChangeInstall, ID: "sdd-router", Client: "claude"}},
			}, nil
		},
	})
}

// startB9Write accepts one install request and returns the result message the
// host would emit, without running a second BeginWrite through Update.
func startB9Write(t *testing.T, app *App, host *OpsHost, req appops.InstallRequest) WriteResultMsg {
	t.Helper()
	started, work, err := host.BeginWrite(WriteRequest{
		Kind:    WriteInstall,
		Install: req,
		Token:   "token",
		Command: "apply --clients claude",
		Label:   "install",
	})
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	app.activeRequest = WriteRequest{Kind: WriteInstall, Install: req, Token: "token"}
	app.hasActiveRequest = true
	app.acceptStarted(started)
	msg, ok := work().(WriteResultMsg)
	if !ok {
		t.Fatalf("work returned %T", work())
	}
	return msg
}

func runB9Cmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, nested := range batch {
			out = append(out, runB9Cmd(nested)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// TestInstallRefreshUsesSameHomeSourceAndOptions proves the read after an
// install is bound to the request's home, source and optional-content choices.
func TestInstallRefreshUsesSameHomeSourceAndOptions(t *testing.T) {
	home := t.TempDir()
	var scopes []catalogScope
	refreshed := b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource))
	app := NewAppWithScreens(TUIConfig{Home: home, Width: 100, Height: 40}, b9CoreCatalog(nil), AppOptions{Ops: b9RefreshHost(t)}, AppScreenOptions{
		CatalogRefresh: func(cfg TUIConfig) (Catalog, error) {
			scopes = append(scopes, catalogScope{Home: cfg.Home, Repo: cfg.Repo, WithOpenCode: cfg.WithOpenCode, OpenCodeFreeFirst: cfg.OpenCodeFreeFirst})
			return refreshed, nil
		},
	})
	req, err := appops.NewInstallRequest("", home, true, skillsync.Options{ClientIDs: []string{"claude"}, WithOpenCode: true})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}

	msg := startB9Write(t, &app, app.Ops(), req)
	next, cmd := app.Update(msg)
	app = next.(App)
	if !app.inventoryPending {
		t.Fatal("completed write did not queue an inventory refresh")
	}
	for _, m := range runB9Cmd(cmd) {
		next, _ := app.Update(m)
		app = next.(App)
	}
	if len(scopes) != 1 {
		t.Fatalf("refresh reads = %d, want 1", len(scopes))
	}
	if scopes[0].Home != home || scopes[0].WithOpenCode != true || scopes[0].OpenCodeFreeFirst != false {
		t.Fatalf("refresh scope = %+v, want same home and OpenCode option", scopes[0])
	}
	if app.inventoryErr != "" {
		t.Fatalf("refresh error = %q", app.inventoryErr)
	}
	if got := DeriveSetupState(app.catalog, app.inventoryErr).Kind; got != SetupStateCurrent {
		t.Fatalf("refreshed setup state = %q, want current", got)
	}
}

func TestNoChangeAndRestoreRefreshKeepTheKnownScope(t *testing.T) {
	home := t.TempDir()
	otherHome := t.TempDir()
	refreshed := b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource))
	var scopes []catalogScope
	app := NewAppWithScreens(TUIConfig{Home: home, Width: 100, Height: 40}, b9CoreCatalog(nil), AppOptions{Ops: b9RefreshHost(t)}, AppScreenOptions{
		CatalogRefresh: func(cfg TUIConfig) (Catalog, error) {
			scopes = append(scopes, catalogScope{Home: cfg.Home, Repo: cfg.Repo, WithOpenCode: cfg.WithOpenCode, OpenCodeFreeFirst: cfg.OpenCodeFreeFirst})
			return refreshed, nil
		},
	})
	req, err := appops.NewInstallRequest("", home, true, skillsync.Options{ClientIDs: []string{"claude"}, WithOpenCode: true})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}

	// A no-op result is still a completed operation and refreshes once.
	msg := startB9Write(t, &app, app.Ops(), req)
	msg.Result = appops.ApplyResult{Outcome: skillsync.OutcomeNoChange}
	next, cmd := app.Update(msg)
	app = next.(App)
	for _, m := range runB9Cmd(cmd) {
		next, _ := app.Update(m)
		app = next.(App)
	}
	if len(scopes) != 1 || scopes[0].Home != home || !scopes[0].WithOpenCode {
		t.Fatalf("no-op refresh scopes = %+v", scopes)
	}

	// A restore request carries no optional-content fields, so the read keeps
	// the scope that is currently on screen instead of guessing.
	restoreReq, err := appops.NewRestoreRequest("", otherHome, "20260101T000000.000000000Z", true)
	if err != nil {
		t.Fatalf("NewRestoreRequest: %v", err)
	}
	app.activeRequest = WriteRequest{Kind: WriteRestore, Restore: restoreReq, Token: "token"}
	app.hasActiveRequest = true
	runB9Cmd(app.startInventoryRefresh())
	if len(scopes) != 2 {
		t.Fatalf("restore refresh reads = %d, want 2 total", len(scopes))
	}
	if got := scopes[1]; got.Home != otherHome || !got.WithOpenCode || got.OpenCodeFreeFirst {
		t.Fatalf("restore refresh scope = %+v, want restore home with the retained options", got)
	}
}

func TestRefreshResultsAreGenerationBound(t *testing.T) {
	home := t.TempDir()
	refreshed := b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource))
	var scopes []catalogScope
	app := NewAppWithScreens(TUIConfig{Home: home, Width: 100, Height: 40}, b9CoreCatalog(nil), AppOptions{Ops: b9RefreshHost(t)}, AppScreenOptions{
		CatalogRefresh: func(cfg TUIConfig) (Catalog, error) {
			scopes = append(scopes, catalogScope{Home: cfg.Home, Repo: cfg.Repo, WithOpenCode: cfg.WithOpenCode, OpenCodeFreeFirst: cfg.OpenCodeFreeFirst})
			return refreshed, nil
		},
	})
	req, err := appops.NewInstallRequest("", home, true, skillsync.Options{ClientIDs: []string{"claude"}, WithOpenCode: true})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	app.activeRequest = WriteRequest{Kind: WriteInstall, Install: req, Token: "token"}
	app.hasActiveRequest = true

	first := app.startInventoryRefresh()
	second := app.startInventoryRefresh()
	if len(scopes) != 0 {
		t.Fatalf("read ran inside Update: %+v", scopes)
	}

	// The older generation arrives last and must not win.
	older := runB9Cmd(first)
	newer := runB9Cmd(second)
	next, _ := app.Update(older[0])
	app = next.(App)
	if got := app.catalog.Items[0].Clients[0].Status; got != skillsync.InventoryStatusMissing {
		t.Fatalf("stale refresh changed the catalog status to %q", got)
	}
	if !app.inventoryPending {
		t.Fatal("a newer refresh was still pending but the app stopped reporting it")
	}
	next, _ = app.Update(newer[0])
	app = next.(App)
	if got := app.catalog.Items[0].Clients[0].Status; got != skillsync.InventoryStatusMatchesSource {
		t.Fatalf("newest refresh was not applied, status = %q", got)
	}
	if app.inventoryPending {
		t.Fatal("accepted refresh stayed pending")
	}
	if got := DeriveSetupState(app.catalog, app.inventoryErr).Kind; got != SetupStateCurrent {
		t.Fatalf("state = %q, want current", got)
	}
}

func TestRefreshFailureKeepsOperationOutcomeAndShowsUnknown(t *testing.T) {
	home := t.TempDir()
	app := NewAppWithScreens(TUIConfig{Home: home, Width: 100, Height: 40}, b9CoreCatalog(b9All(skillsync.InventoryStatusMatchesSource)), AppOptions{Ops: b9RefreshHost(t)}, AppScreenOptions{
		CatalogRefresh: func(TUIConfig) (Catalog, error) {
			return Catalog{}, errors.New("inventory read failed: permission denied")
		},
	})
	req, err := appops.NewInstallRequest("", home, true, skillsync.Options{ClientIDs: []string{"claude"}})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	msg := startB9Write(t, &app, app.Ops(), req)
	next, cmd := app.Update(msg)
	app = next.(App)
	for _, m := range runB9Cmd(cmd) {
		next, _ := app.Update(m)
		app = next.(App)
	}

	last, ok := app.LastWriteResult()
	if !ok || last.Result.Outcome != skillsync.OutcomeApplied {
		t.Fatalf("refresh failure changed the accepted operation result: %+v, %t", last, ok)
	}
	if app.inventoryPending {
		t.Fatal("failed refresh stayed pending")
	}
	if !strings.Contains(app.inventoryErr, "permission denied") {
		t.Fatalf("inventoryErr = %q", app.inventoryErr)
	}
	if got := DeriveSetupState(app.catalog, app.inventoryErr).Kind; got != SetupStateReadFailure {
		t.Fatalf("setup state = %q, want read-failure", got)
	}
	text := b9Flat(strings.Join(app.homeLines(90), "\n"))
	for _, want := range []string{"last local inventory read failed", "permission denied", "not an operation outcome"} {
		if !strings.Contains(text, b9Flat(want)) {
			t.Fatalf("Home missing refresh failure %q:\n%s", want, text)
		}
	}
}

func TestReplaceCatalogPreservesSelection(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30}
	initial := Catalog{Items: []CatalogItem{
		{ID: "alpha", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "beta", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
	}}
	model := NewCatalogModel(cfg, initial)
	if cmd := model.ReplaceCatalog(initial); cmd != nil {
		runB9Cmd(cmd)
	}
	model.list.Select(1)
	selected, ok := model.Selected()
	if !ok || selected.ID != "beta" {
		t.Fatalf("setup selection = %+v, %t", selected, ok)
	}

	refreshed := Catalog{Items: []CatalogItem{
		{ID: "alpha", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "beta", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		{ID: "gamma", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
	}}
	runB9Cmd(model.ReplaceCatalog(refreshed))
	selected, ok = model.Selected()
	if !ok || selected.ID != "beta" {
		t.Fatalf("refreshed selection = %+v, %t, want beta preserved", selected, ok)
	}
	if len(model.VisibleItems()) != 3 {
		t.Fatalf("visible items = %d, want 3", len(model.VisibleItems()))
	}
}
