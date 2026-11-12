package tui

// W3-A opt-in local diagnostics tests. Every fixture is a fake seam or a
// t.TempDir() path: no real home, PATH search, network, model, credential or
// file write is performed.

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/fleet"
)

// stubFleetDoctor is the deterministic local-evidence seam. It records exactly
// what the screen asked for and performs no I/O.
type stubFleetDoctor struct {
	view FleetDoctorView
	err  error

	calls      int
	lastModels []string
	lastBin    string
	lastDump   string
}

func (d *stubFleetDoctor) Diagnose(ctx context.Context, models []string, opencodeBin, catalogDump string) (FleetDoctorView, error) {
	d.calls++
	d.lastModels = append([]string(nil), models...)
	d.lastBin = opencodeBin
	d.lastDump = catalogDump
	if d.err != nil {
		return FleetDoctorView{}, d.err
	}
	return d.view, nil
}

// newFleetDoctorModel builds a Fleet model with a loaded three-route status and
// an injected diagnostics seam.
func newFleetDoctorModel(t *testing.T, doctor FleetDoctor, live bool) FleetModel {
	t.Helper()
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	cfg := TUIConfig{
		Width:       110,
		Height:      34,
		Color:       false,
		Home:        t.TempDir(),
		FleetConfig: "/test/fleet.json",
		FleetLive:   live,
	}
	model := NewFleetModel(cfg, reader)
	model.SetDoctor(doctor)
	model.status = testFleetStatusView()
	model.hasLoaded = true
	return model
}

// fleetDoctorType sends one printable rune the way the terminal delivers it.
func fleetDoctorType(t *testing.T, model FleetModel, text string) FleetModel {
	t.Helper()
	for _, r := range text {
		var msg tea.Msg
		model, msg = fleetUpdateKey(t, model, r, string(r))
		if msg != nil {
			t.Fatalf("typing %q produced %T, want no command", string(r), msg)
		}
	}
	return model
}

func TestFleetDoctorDefaultsOfflineWithoutInspectingAnything(t *testing.T) {
	doctor := &stubFleetDoctor{}
	model := newFleetDoctorModel(t, doctor, false)

	model, msg := fleetUpdateKey(t, model, 'd', "d")
	if msg != nil {
		t.Fatalf("opening diagnostics produced %T, want no command", msg)
	}
	if !model.DoctorOpen() {
		t.Fatal("d did not open the diagnostics panel from Overview")
	}
	if doctor.calls != 0 {
		t.Fatalf("the offline default called the doctor seam %d time(s), want 0", doctor.calls)
	}

	view := model.DoctorView()
	if !view.Offline || !view.Available {
		t.Fatalf("default view = %+v, want an available offline report", view)
	}
	if len(view.Entries) != 3 {
		t.Fatalf("default entries = %d, want one per configured route (3)", len(view.Entries))
	}
	for _, entry := range view.Entries {
		if entry.Label != fleet.DoctorNotChecked {
			t.Fatalf("route %q label = %q, want %q", entry.Model, entry.Label, fleet.DoctorNotChecked)
		}
	}
	if len(view.ToolsInspected) != 0 || len(view.Failures) != 0 {
		t.Fatalf("offline default inspected %v / failed %v, want nothing", view.ToolsInspected, view.Failures)
	}

	rendered := strings.Join(model.Lines(110), "\n")
	for _, want := range []string{
		"Fleet Manager · Diagnostics",
		"no PATH search",
		"OpenCode binary",
		"Catalog dump",
		"not checked (offline)",
		"Esc returns to Overview",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("diagnostics panel is missing %q:\n%s", want, rendered)
		}
	}
	// The honesty note says what is *not* claimed; no route may carry the
	// authenticated label here.
	if strings.Contains(rendered, "· authenticated") {
		t.Errorf("offline default invented an authenticated label:\n%s", rendered)
	}
}

func TestFleetDoctorExplicitBinaryRunsOnceAndLabelsExactVocabulary(t *testing.T) {
	doctor := &stubFleetDoctor{view: FleetDoctorView{
		Available:      true,
		ToolsInspected: []string{"/tmp/open-opencode (local exec)"},
		Entries: []FleetDoctorEntry{
			{Model: "9router/ag/gemini-3.8-flash-high", Label: fleet.DoctorAuthenticated, Reason: "listed in a local catalog and provider has an explicit local authentication marker"},
			{Model: "opencode-go/deepseek-v4.1-flash", Label: fleet.DoctorCatalogVisible, Reason: "listed in a local catalog; authentication not verified from this evidence"},
			{Model: "9router/claude-opus-5.5", Label: fleet.DoctorUnknown, Reason: "local sources contain no evidence about this provider"},
		},
	}}
	model := newFleetDoctorModel(t, doctor, false)
	model, _ = fleetUpdateKey(t, model, 'd', "d")

	// Printable keys, including q/j/k//, are literal path text while the panel
	// is open; nothing leaks to navigation.
	model = fleetDoctorType(t, model, "/tmp/qjk/opencode")
	if got := model.DoctorBinInput(); got != "/tmp/qjk/opencode" {
		t.Fatalf("binary input = %q, want the literal typed path", got)
	}
	if model.ActiveView() != FleetViewOverview {
		t.Fatalf("typing changed the underlying view to %v", model.ActiveView())
	}
	if !model.IsInputFocused() {
		t.Fatal("the diagnostics input does not own printable keys")
	}
	if doctor.calls != 0 {
		t.Fatalf("typing ran the doctor seam %d time(s), want 0", doctor.calls)
	}

	model, msg := fleetUpdateKey(t, model, tea.KeyEnter, "")
	doctorMsg, ok := msg.(FleetDoctorMsg)
	if !ok {
		t.Fatalf("Enter produced %T, want FleetDoctorMsg", msg)
	}
	model, _ = model.Update(doctorMsg)

	if doctor.calls != 1 {
		t.Fatalf("doctor calls = %d, want exactly one explicit run", doctor.calls)
	}
	if doctor.lastBin != "/tmp/qjk/opencode" || doctor.lastDump != "" {
		t.Fatalf("doctor request = bin %q dump %q, want only the typed binary", doctor.lastBin, doctor.lastDump)
	}
	if len(doctor.lastModels) != 3 {
		t.Fatalf("doctor models = %v, want the three configured routes", doctor.lastModels)
	}

	rendered := strings.Join(model.Lines(110), "\n")
	for _, want := range []string{
		"9router/ag/gemini-3.8-flash-high · authenticated",
		"opencode-go/deepseek-v4.1-flash · catalog-visible",
		"9router/claude-opus-5.5 · unknown",
		"tools inspected: /tmp/open-opencode (local exec)",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("diagnostics report is missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "ChatGPT") || strings.Contains(rendered, "free forever") {
		t.Errorf("diagnostics invented an entitlement claim:\n%s", rendered)
	}
}

func TestFleetDoctorLabelVocabularyIsExact(t *testing.T) {
	want := map[fleet.DoctorLabel]string{
		fleet.DoctorConfigured:     "configured",
		fleet.DoctorCatalogVisible: "catalog-visible",
		fleet.DoctorAuthenticated:  "authenticated",
		fleet.DoctorUnknown:        "unknown",
		fleet.DoctorUnavailable:    "unavailable",
		fleet.DoctorNotChecked:     "not checked (offline)",
	}
	for label, text := range want {
		if string(label) != text {
			t.Errorf("engine label = %q, want the reviewed display text %q", label, text)
		}
		if got := doctorLabel(label); got != label {
			t.Errorf("doctorLabel(%q) = %q, want it unchanged", label, got)
		}
	}
	if got := doctorLabel(fleet.DoctorLabel("authenticated-ish")); got != fleet.DoctorUnknown {
		t.Errorf("doctorLabel(unknown marker) = %q, want %q", got, fleet.DoctorUnknown)
	}
}

func TestFleetDoctorFocusTabBackspaceAndClear(t *testing.T) {
	doctor := &stubFleetDoctor{view: FleetDoctorView{Available: true, Offline: true}}
	model := newFleetDoctorModel(t, doctor, false)
	model, _ = fleetUpdateKey(t, model, 'd', "d")

	model = fleetDoctorType(t, model, "/tmp/bin")
	model, _ = fleetUpdateKey(t, model, tea.KeyTab, "") // focus catalog dump
	model = fleetDoctorType(t, model, "/tmp/catalog.json")
	if got := model.DoctorDumpInput(); got != "/tmp/catalog.json" {
		t.Fatalf("catalog dump input = %q", got)
	}
	model, _ = fleetUpdateKey(t, model, tea.KeyBackspace, "")
	if got := model.DoctorDumpInput(); got != "/tmp/catalog.jso" {
		t.Fatalf("backspace = %q, want the last rune removed", got)
	}
	model, _ = fleetUpdateKey(t, model, tea.KeyTab, "") // back to the binary field
	model, _ = fleetUpdateKey(t, model, 'u', "u")
	if got := model.DoctorBinInput(); got != "/tmp/binu" {
		t.Fatalf("binary input after returning = %q", got)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if got := model.DoctorBinInput(); got != "" {
		t.Fatalf("Ctrl+U left %q, want an empty focused field", got)
	}
	if got := model.DoctorDumpInput(); got != "/tmp/catalog.jso" {
		t.Fatalf("Ctrl+U cleared the wrong field: dump = %q", got)
	}

	// Clearing the remaining field makes Enter return to the offline default
	// without calling the seam.
	model, _ = fleetUpdateKey(t, model, tea.KeyTab, "")
	model, _ = model.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if got := model.DoctorDumpInput(); got != "" {
		t.Fatalf("Ctrl+U left the dump field at %q", got)
	}
	model, msg := fleetUpdateKey(t, model, tea.KeyEnter, "")
	if msg != nil {
		t.Fatalf("offline Enter produced %T, want no command", msg)
	}
	if doctor.calls != 0 {
		t.Fatalf("offline Enter called the seam %d time(s)", doctor.calls)
	}
	if !model.DoctorView().Offline {
		t.Fatal("offline Enter left a non-offline report")
	}
}

func TestFleetDoctorEscReturnsAndKeepsTheUnderlyingView(t *testing.T) {
	model := newFleetDoctorModel(t, &stubFleetDoctor{}, false)
	model.SetView(FleetViewRecovery)
	model, _ = fleetUpdateKey(t, model, 'd', "d")
	if !model.DoctorOpen() {
		t.Fatal("d did not open diagnostics from Recovery")
	}
	if model.WantsClose("esc") {
		t.Fatal("Esc would leave Fleet Manager instead of returning from diagnostics")
	}
	if !strings.Contains(strings.Join(model.Lines(110), "\n"), "Esc returns to Recovery") {
		t.Fatal("the panel does not name its return destination")
	}

	model, msg := fleetUpdateKey(t, model, tea.KeyEscape, "")
	if msg != nil {
		t.Fatalf("Esc produced %T, want no command", msg)
	}
	if model.DoctorOpen() {
		t.Fatal("Esc did not close the diagnostics panel")
	}
	if model.ActiveView() != FleetViewRecovery {
		t.Fatalf("view after Esc = %v, want Recovery", model.ActiveView())
	}
}

func TestFleetDoctorPanelIsBoundsSafeAndScrolls(t *testing.T) {
	doctor := &stubFleetDoctor{view: FleetDoctorView{Available: true, Entries: []FleetDoctorEntry{
		{Model: "9router/aaa", Label: fleet.DoctorConfigured, Reason: "provider appears in local configuration"},
		{Model: "9router/bbb", Label: fleet.DoctorCatalogVisible, Reason: "listed in a local catalog"},
		{Model: "9router/ccc", Label: fleet.DoctorUnknown, Reason: "no local evidence"},
		{Model: "openai/gpt-5", Label: fleet.DoctorUnavailable, Reason: "the requested local inspection tool failed"},
		{Model: "opencode-go/deepseek-v4.1-flash", Label: fleet.DoctorNotChecked, Reason: "offline default"},
	}}}
	model := newFleetDoctorModel(t, doctor, false)
	model, _ = fleetUpdateKey(t, model, 'd', "d")
	model = fleetDoctorType(t, model, "/tmp/bin")
	model, msg := fleetUpdateKey(t, model, tea.KeyEnter, "")
	model, _ = model.Update(msg)

	for _, size := range []struct{ w, h int }{{60, 8}, {60, 10}, {80, 16}, {110, 24}} {
		model.SetSize(size.w, size.h)
		lines := model.Lines(size.w)
		if len(lines) > size.h {
			t.Fatalf("%dx%d: %d rows, want at most %d", size.w, size.h, len(lines), size.h)
		}
		for i, line := range lines {
			if got := len([]rune(line)); got > size.w {
				t.Fatalf("%dx%d: row %d has %d cells > %d: %q", size.w, size.h, i, got, size.w, line)
			}
		}
		footer := lines[len(lines)-1]
		for _, want := range []string{"Enter", "Esc"} {
			if !strings.Contains(footer, want) {
				t.Fatalf("%dx%d: footer %q is missing %q", size.w, size.h, footer, want)
			}
		}
		if strings.Contains(strings.ToLower(footer), "apply") || strings.Contains(strings.ToLower(footer), "restore") {
			t.Fatalf("%dx%d: footer advertises a write: %q", size.w, size.h, footer)
		}
	}

	// Below the usable geometry the shared resize guidance replaces the panel
	// and offers no affirmative key.
	model.SetSize(40, 8)
	small := strings.Join(model.Lines(40), "\n")
	if !strings.Contains(small, "Resize") || !strings.Contains(small, "Press Esc to return") {
		t.Fatalf("40x8 guidance = %q, want the shared resize warning", small)
	}

	// At 60x10 the report windows with a reachability hint, and scrolling
	// reaches the last material row.
	model.SetSize(60, 10)
	first := strings.Join(model.Lines(60), "\n")
	if !strings.Contains(first, "up/down scrolls") {
		t.Fatalf("short-height report lost its scroll hint:\n%s", first)
	}
	for i := 0; i < 12; i++ {
		model, _ = fleetUpdateKey(t, model, tea.KeyDown, "")
	}
	last := strings.Join(model.Lines(60), "\n")
	if first == last {
		t.Fatal("scrolling did not move the diagnostics report")
	}
	if !strings.Contains(last, "not checked (offline)") || !strings.Contains(last, "Labels are local evidence only") {
		t.Fatalf("scrolling never reached the final rows:\n%s", last)
	}
}

func TestFleetDoctorStaleReplyIsDiscarded(t *testing.T) {
	doctor := &stubFleetDoctor{view: FleetDoctorView{Available: true, Entries: []FleetDoctorEntry{
		{Model: "9router/aaa", Label: fleet.DoctorConfigured, Reason: "first"},
	}}}
	model := newFleetDoctorModel(t, doctor, false)
	model, _ = fleetUpdateKey(t, model, 'd', "d")
	model = fleetDoctorType(t, model, "/tmp/bin")

	model, first := fleetUpdateKey(t, model, tea.KeyEnter, "")
	model, _ = fleetUpdateKey(t, model, tea.KeyEnter, "") // second request bumps the generation
	stale, ok := first.(FleetDoctorMsg)
	if !ok {
		t.Fatalf("first run produced %T", first)
	}
	model, _ = model.Update(stale)
	if model.DoctorView().Entries[0].Reason == "first" {
		t.Fatal("a stale diagnostics reply replaced the current report")
	}
}

func TestFleetDoctorDoesNotChangeTheLiveGate(t *testing.T) {
	doctor := &stubFleetDoctor{view: FleetDoctorView{Available: true, Entries: []FleetDoctorEntry{
		{Model: "9router/aaa", Label: fleet.DoctorAuthenticated, Reason: "explicit local marker"},
	}}}
	model := newFleetDoctorModel(t, doctor, false)
	model, _ = fleetUpdateKey(t, model, 'd', "d")
	model = fleetDoctorType(t, model, "/tmp/bin")
	model, msg := fleetUpdateKey(t, model, tea.KeyEnter, "")
	model, _ = model.Update(msg)

	if model.LiveEnabled() {
		t.Fatal("running diagnostics enabled the live capability")
	}
	model = model.closeDoctor()
	model.view = FleetViewReview
	model.planPreview = testFleetPlanPreview()
	before := doctor.calls
	model, cmd := model.requestApply()
	if cmd != nil {
		t.Fatalf("requestApply produced a command in a preview build (%T)", cmd)
	}
	if model.IsConfirmingApply() {
		t.Fatal("the apply confirmation opened in a preview build")
	}
	if model.ApplyOutcome().Headline != "Preview only" {
		t.Fatalf("apply outcome = %q, want Preview only", model.ApplyOutcome().Headline)
	}
	if doctor.calls != before {
		t.Fatal("requestApply consulted the diagnostics seam")
	}
	if !strings.Contains(strings.Join(model.Lines(110), "\n"), "Deployment capability: Preview only") {
		t.Fatal("Review Changes no longer reports the preview-only capability")
	}
}

func TestAppOpsFleetDoctorDefaultConsultsNothing(t *testing.T) {
	models := []string{"9router/ag/gemini-3.8-flash-high", "openai/gpt-5"}
	view, err := AppOpsFleetDoctor{}.Diagnose(context.Background(), models, "", "")
	if err != nil {
		t.Fatalf("offline Diagnose: %v", err)
	}
	if !view.Available || !view.Offline {
		t.Fatalf("view = %+v, want an available offline report", view)
	}
	if len(view.ToolsInspected) != 0 || len(view.Failures) != 0 {
		t.Fatalf("offline adapter inspected %v and failed %v; it must consult nothing", view.ToolsInspected, view.Failures)
	}
	if len(view.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(view.Entries))
	}
	for _, entry := range view.Entries {
		if entry.Label != fleet.DoctorNotChecked {
			t.Fatalf("%q label = %q, want %q", entry.Model, entry.Label, fleet.DoctorNotChecked)
		}
	}
	// openai/* stays a route: no authentication or entitlement is invented.
	for _, entry := range view.Entries {
		if strings.Contains(strings.ToLower(entry.Reason), "authenticated") {
			t.Fatalf("offline reason invented authentication: %q", entry.Reason)
		}
	}
}

// TestFleetDoctorPanelReachableThroughTheShell proves the contextual panel is
// reachable from Overview inside the real shell, owns printable keys, and
// returns to the hub with two Escapes without touching a write path.
func TestFleetDoctorPanelReachableThroughTheShell(t *testing.T) {
	doctor := &stubFleetDoctor{view: FleetDoctorView{Available: true, Entries: []FleetDoctorEntry{
		{Model: "9router/aaa", Label: fleet.DoctorConfigured, Reason: "provider appears in local configuration"},
	}}}
	cfg := TUIConfig{Width: 110, Height: 30, Home: t.TempDir(), FleetConfig: "/test/fleet.json", Color: false}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{
		FleetReader:  &trackingFleetReader{customRes: testFleetStatusView()},
		FleetPlanner: &fakeFleetPlanner{},
		FleetDoctor:  doctor,
	})

	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	if app.installSection != installSectionFleet {
		t.Fatalf("installSection = %v, want installSectionFleet", app.installSection)
	}
	app = fleetKey(t, app, "d")
	if !app.fleet.DoctorOpen() {
		t.Fatal("d did not open diagnostics inside the shell")
	}
	view := app.View().Content
	for _, want := range []string{"Diagnostics", "not checked (offline)", "OpenCode binary"} {
		if !strings.Contains(view, want) {
			t.Fatalf("shell diagnostics view is missing %q:\n%s", want, view)
		}
	}
	if doctor.calls != 0 {
		t.Fatalf("opening the panel called the doctor seam %d time(s)", doctor.calls)
	}

	// q, j, k and / are literal path text, never navigation or quit.
	for _, r := range "/tmp/qjk/opencode" {
		app = fleetPress(t, app, r, string(r))
	}
	if app.quitting {
		t.Fatal("a literal q in the diagnostics path quit the application")
	}
	if got := app.fleet.DoctorBinInput(); got != "/tmp/qjk/opencode" {
		t.Fatalf("binary input = %q, want the literal typed path", got)
	}

	app = fleetKey(t, app, "esc")
	if app.fleet.DoctorOpen() {
		t.Fatal("Esc did not return from diagnostics to the Fleet view")
	}
	if app.installSection != installSectionFleet {
		t.Fatal("Esc from diagnostics left Fleet Manager")
	}
	app = fleetKey(t, app, "esc")
	if app.installSection == installSectionFleet {
		t.Fatal("the second Esc did not return to the Install & Manage hub")
	}
}

func TestAppOpsFleetDoctorRefusesMissingOrUnusableBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file mode expectations are not portable to Windows")
	}
	home := t.TempDir()
	models := []string{"9router/ag/gemini-3.8-flash-high"}

	missing := filepath.Join(home, "absent-opencode")
	if _, err := (AppOpsFleetDoctor{}).Diagnose(context.Background(), models, missing, ""); err == nil {
		t.Fatal("a missing explicit binary was not refused")
	} else if !strings.Contains(err.Error(), "not readable") {
		t.Fatalf("missing-binary error = %q, want an explicit refusal", err)
	}

	// A bare command name is never resolved through PATH.
	if _, err := (AppOpsFleetDoctor{}).Diagnose(context.Background(), models, "opencode", ""); err == nil {
		t.Fatal("a non-absolute binary name was accepted")
	} else if !strings.Contains(err.Error(), "never searches PATH") {
		t.Fatalf("relative-binary error = %q, want a PATH refusal", err)
	}

	if _, err := (AppOpsFleetDoctor{}).Diagnose(context.Background(), models, home, ""); err == nil {
		t.Fatal("a directory path was accepted as a binary")
	} else if !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory-binary error = %q, want a regular-file refusal", err)
	}

	// A regular but non-executable file exists: the engine reports the failed
	// local probe as `unavailable` plus a sanitized failure note. Nothing is
	// executed through a shell and nothing is written.
	notExecutable := filepath.Join(home, "opencode-fixture")
	if err := os.WriteFile(notExecutable, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	view, err := (AppOpsFleetDoctor{}).Diagnose(context.Background(), models, notExecutable, "")
	if err != nil {
		t.Fatalf("unusable binary Diagnose returned an error instead of a labelled report: %v", err)
	}
	if view.Offline {
		t.Fatal("an explicit local binary was reported as the offline default")
	}
	if len(view.Entries) != 1 || view.Entries[0].Label != fleet.DoctorUnavailable {
		t.Fatalf("entries = %+v, want one unavailable label", view.Entries)
	}
	if len(view.Failures) == 0 {
		t.Fatal("an unusable local tool produced no failure note")
	}
	for _, failure := range view.Failures {
		if strings.ContainsAny(failure, "\n\r\t") {
			t.Fatalf("failure text carries unfiltered control characters: %q", failure)
		}
	}

	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "opencode-fixture" {
		t.Fatalf("diagnostics wrote to the fixture home: %v", doctorDirNames(entries))
	}
}

func doctorDirNames(entries []os.DirEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name())
	}
	return out
}
