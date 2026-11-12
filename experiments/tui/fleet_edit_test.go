package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fakeFleetEditor struct {
	previewCalls []FleetSourcePreview
	customDiff   []string
	customErr    error
	calls        int
	lastSource   string
	lastEdits    []FleetModelEdit
}

func (f *fakeFleetEditor) PreviewSource(ctx context.Context, source string, edits []FleetModelEdit) (FleetSourcePreview, error) {
	f.calls++
	f.lastSource = source
	f.lastEdits = edits
	if f.customErr != nil {
		return FleetSourcePreview{}, f.customErr
	}
	diff := f.customDiff
	if len(diff) == 0 && len(edits) > 0 {
		diff = GenerateFallbackDiffLines(edits, source)
	}
	return FleetSourcePreview{
		Available:  true,
		SourcePath: source,
		BaseDigest: "sha256:abc123base",
		OldDigest:  "sha256:abc123base",
		NewDigest:  "sha256:def456new",
		Op:         "update",
		DiffLines:  diff,
	}, nil
}

func newTestFleetModel(source string, reader FleetReader, editor FleetEditor) FleetModel {
	cfg := TUIConfig{
		Width:       110,
		Height:      30,
		Color:       false,
		FleetConfig: source,
	}
	m := NewFleetModelWithEditor(cfg, reader, editor)
	// Populate status directly for testing
	status := testFleetStatusView()
	status.SourcePath = source
	m.status = status
	m.hasLoaded = true
	return m
}

func TestFleetModelPickerExactValidAndInvalid(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)

	// Switch to Profiles view
	m.view = FleetViewProfiles
	selected, ok := m.SelectedInstance()
	if !ok || selected.InstanceID != "sdd-implement-9r-1" {
		t.Fatalf("selected = %v, want sdd-implement-9r-1", selected.InstanceID)
	}

	// 1. Open model picker with 'm'
	m, _ = m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if !m.IsEditingModel() {
		t.Fatal("expected IsEditingModel() to be true after pressing 'm'")
	}
	if !m.IsInputFocused() {
		t.Fatal("expected IsInputFocused() to be true while editing model")
	}

	// 2. Type invalid model syntax: "invalid model" (contains space)
	for _, ch := range "invalid model" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Must remain in editingModel mode with error displayed; dirty count must be 0
	if !m.IsEditingModel() {
		t.Fatal("expected model picker to stay open on invalid syntax")
	}
	if m.pickerErr == "" {
		t.Fatal("expected pickerErr to be set for invalid model syntax")
	}
	if m.DirtyCount() != 0 {
		t.Fatalf("dirtyCount = %d, want 0 after invalid entry", m.DirtyCount())
	}

	// 3. Clear input with ctrl+u
	m, _ = m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if m.modelInput != "" {
		t.Fatalf("modelInput = %q, want empty after ctrl+u", m.modelInput)
	}

	// 4. Type invalid model syntax: missing provider "deepseek-v4.1-flash"
	for _, ch := range "deepseek-v4.1-flash" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.pickerErr == "" {
		t.Fatal("expected pickerErr for model lacking provider prefix")
	}

	// 5. Clear and enter valid exact model syntax: "opencode-go/deepseek-v4.1-flash#high"
	m, _ = m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	for _, ch := range "opencode-go/deepseek-v4.1-flash#high" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.IsEditingModel() {
		t.Fatal("expected model picker to close after valid model entered")
	}
	if m.pickerErr != "" {
		t.Fatalf("unexpected pickerErr: %v", m.pickerErr)
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("dirtyCount = %d, want 1", m.DirtyCount())
	}

	// Verify draft carries the staged edit
	draft := m.Draft()
	edit, has := draft["sdd-implement-9r-1"]
	if !has {
		t.Fatal("missing staged edit for sdd-implement-9r-1")
	}
	if edit.Model != "opencode-go/deepseek-v4.1-flash#high" {
		t.Fatalf("staged model = %q, want opencode-go/deepseek-v4.1-flash#high", edit.Model)
	}
	if !edit.Enabled {
		t.Fatal("expected edit.Enabled to remain true")
	}

	// Process preview command
	if cmd == nil {
		t.Fatal("expected command from valid stage")
	}
	previewMsg := cmd()
	m, _ = m.Update(previewMsg)
	if m.SourcePreview().BaseDigest != "sha256:abc123base" {
		t.Fatalf("BaseDigest = %q, want sha256:abc123base", m.SourcePreview().BaseDigest)
	}

	// 6. Verify old->new display in list and detail
	lines := strings.Join(m.Lines(110), "\n")
	if !strings.Contains(lines, "1 unsaved edit") {
		t.Errorf("Profiles missing unsaved edit indicator:\n%s", lines)
	}
	if !strings.Contains(lines, "->") || !strings.Contains(lines, "deepseek-v4.1-flash#high") {
		t.Errorf("Profiles missing old->new model transition:\n%s", lines)
	}
}

func TestFleetModelPickerSuggestionSelection(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)
	m.view = FleetViewProfiles

	// Open model picker
	m, _ = m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if !m.IsEditingModel() {
		t.Fatal("expected model picker open")
	}

	// Navigate down to 2nd suggestion
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.pickerIndex != 1 {
		t.Fatalf("pickerIndex = %d, want 1", m.pickerIndex)
	}

	suggestions := m.pickerSuggestions()
	if len(suggestions) < 2 {
		t.Fatalf("expected at least 2 suggestions, got %d", len(suggestions))
	}
	targetSuggestion := suggestions[1]

	// Press Tab to complete suggestion into input
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.modelInput != targetSuggestion {
		t.Fatalf("modelInput = %q, want %q after Tab", m.modelInput, targetSuggestion)
	}

	// Press Enter to stage
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.IsEditingModel() {
		t.Fatal("expected model picker to close on Enter")
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("dirtyCount = %d, want 1", m.DirtyCount())
	}
	draft := m.Draft()
	if draft["sdd-implement-9r-1"].Model != targetSuggestion {
		t.Fatalf("staged model = %q, want %q", draft["sdd-implement-9r-1"].Model, targetSuggestion)
	}
}

func TestFleetStageOneAndManyEdits(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)

	// Stage edit 1
	err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium")
	if err != nil {
		t.Fatalf("StageModel 1 err = %v", err)
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("dirtyCount = %d, want 1", m.DirtyCount())
	}

	// Stage edit 2
	err = m.StageModel("sdd-orchestrator", "opencode-go/qwen3.8-flash")
	if err != nil {
		t.Fatalf("StageModel 2 err = %v", err)
	}
	if m.DirtyCount() != 2 {
		t.Fatalf("dirtyCount = %d, want 2", m.DirtyCount())
	}

	// Stage edit 3 (toggle)
	err = m.StageToggle("sdd-spec-9r")
	if err != nil {
		t.Fatalf("StageToggle 3 err = %v", err)
	}
	if m.DirtyCount() != 3 {
		t.Fatalf("dirtyCount = %d, want 3", m.DirtyCount())
	}

	edits := m.Edits()
	if len(edits) != 3 {
		t.Fatalf("len(Edits) = %d, want 3", len(edits))
	}
	if edits[0].InstanceID != "sdd-implement-9r-1" ||
		edits[1].InstanceID != "sdd-orchestrator" ||
		edits[2].InstanceID != "sdd-spec-9r" {
		t.Fatalf("unexpected sorted edits order: %+v", edits)
	}
}

func TestFleetEnableDisableConsequenceManagedRemoval(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)
	m.view = FleetViewProfiles

	// Select first instance (sdd-implement-9r-1, managed current, enabled)
	selected, _ := m.SelectedInstance()
	if !selected.Enabled {
		t.Fatal("expected sdd-implement-9r-1 to be enabled")
	}

	// 1. Press 't' to toggle disable
	m, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	if !m.IsConfirmingDisable() {
		t.Fatal("expected confirmDisable to be active")
	}
	if !m.IsInputFocused() {
		t.Fatal("expected IsInputFocused() to be true during confirmDisable")
	}

	// Verify consequence dialog content
	dialog := strings.Join(m.Lines(110), "\n")
	for _, want := range []string{
		"Confirm Disable · sdd-implement-9r-1",
		"CONSEQUENCE NOTICE",
		"managed",
		"removal",
		"OpenCode agent file upon apply",
		"backup",
		"Unmanaged files cannot be deleted",
		"Stage disable in-memory draft? [y/N]",
	} {
		if !strings.Contains(dialog, want) {
			t.Errorf("Disable dialog missing %q:\n%s", want, dialog)
		}
	}

	// 2. Cancel with 'n'
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if m.IsConfirmingDisable() {
		t.Fatal("expected confirmDisable to close after 'n'")
	}
	if m.DirtyCount() != 0 {
		t.Fatalf("dirtyCount = %d, want 0 after cancel", m.DirtyCount())
	}

	// 3. Press 't' again and confirm with 'y'
	m, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	if !m.IsConfirmingDisable() {
		t.Fatal("expected confirmDisable active")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if m.IsConfirmingDisable() {
		t.Fatal("expected confirmDisable closed after 'y'")
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("dirtyCount = %d, want 1 after disable confirmed", m.DirtyCount())
	}
	draft := m.Draft()
	if draft["sdd-implement-9r-1"].Enabled {
		t.Fatal("expected draft enabled = false")
	}

	// 4. Test unmanaged instance cannot be disabled
	unmanagedStatus := testFleetStatusView()
	unmanagedStatus.Instances = append(unmanagedStatus.Instances, FleetInstanceView{
		InstanceID:       "unmanaged-agent",
		Role:             "custom",
		InstalledState:   "unmanaged",
		Enabled:          true,
		SourceProvenance: "external",
	})
	m.status = unmanagedStatus
	m.selectedIndex = len(m.status.Instances) - 1 // point to unmanaged-agent

	m, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	if m.IsConfirmingDisable() {
		t.Fatal("unmanaged instance must NOT trigger confirmDisable")
	}
	if m.disableWarning == "" || !strings.Contains(m.disableWarning, "unmanaged") {
		t.Fatalf("expected unmanaged disableWarning, got: %q", m.disableWarning)
	}
}

func TestFleetDirtyProtectionDiscardAndKeep(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)
	m.view = FleetViewProfiles

	// Stage an edit
	if err := m.StageModel("sdd-implement-9r-1", "opencode-go/deepseek-v4.1-flash"); err != nil {
		t.Fatal(err)
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("dirtyCount = %d, want 1", m.DirtyCount())
	}

	// 1. Press Esc with dirty edits -> opens confirmDiscard prompt
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.IsConfirmingDiscard() {
		t.Fatal("expected confirmDiscard active on Esc with dirty edits")
	}

	promptView := strings.Join(m.Lines(100), "\n")
	for _, want := range []string{
		"Unsaved Edits",
		"1 unsaved edit(s)",
		"[k] Keep editing",
		"[d] Discard edits",
		"[s / Esc] Stay",
	} {
		if !strings.Contains(promptView, want) {
			t.Errorf("Discard prompt missing %q:\n%s", want, promptView)
		}
	}

	// 2. Press 'k' (keep editing) -> prompt closes, edits preserved
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if m.IsConfirmingDiscard() {
		t.Fatal("expected confirmDiscard closed after 'k'")
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("dirtyCount = %d, want 1", m.DirtyCount())
	}

	// 3. Press Esc, then 's' (stay) -> prompt closes, edits preserved
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.IsConfirmingDiscard() {
		t.Fatal("expected confirmDiscard active")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if m.IsConfirmingDiscard() {
		t.Fatal("expected confirmDiscard closed after 's'")
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("dirtyCount = %d, want 1", m.DirtyCount())
	}

	// 4. Press Esc, then 'd' (discard) -> edits discarded, draft cleared
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.IsConfirmingDiscard() {
		t.Fatal("expected confirmDiscard active")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.IsConfirmingDiscard() {
		t.Fatal("expected confirmDiscard closed after 'd'")
	}
	if m.DirtyCount() != 0 {
		t.Fatalf("dirtyCount = %d, want 0 after discard", m.DirtyCount())
	}
	if len(m.Draft()) != 0 {
		t.Fatalf("draft = %+v, want empty after discard", m.Draft())
	}

	// 5. In Overview with dirtyCount == 0, WantsClose is true on Esc
	m.view = FleetViewOverview
	if !m.WantsClose("esc") {
		t.Fatal("expected WantsClose('esc') = true when dirtyCount == 0 in Overview")
	}
}

func TestFleetStagingZeroWritesAndNoRealHome(t *testing.T) {
	tempRoot := t.TempDir()
	fleetPath := filepath.Join(tempRoot, "fleet.json")
	initialContent := []byte(`{"version": 1}`)
	if err := os.WriteFile(fleetPath, initialContent, 0o644); err != nil {
		t.Fatal(err)
	}
	infoBefore, err := os.Stat(fleetPath)
	if err != nil {
		t.Fatal(err)
	}

	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel(fleetPath, reader, editor)

	// Stage multiple edits and toggle
	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}
	if err := m.StageToggle("sdd-spec-9r"); err != nil {
		t.Fatal(err)
	}

	// Render all views
	for _, v := range []FleetView{FleetViewOverview, FleetViewProfiles, FleetViewReview, FleetViewRecovery} {
		m.SetView(v)
		lines := m.Lines(100)
		if len(lines) == 0 {
			t.Errorf("view %v returned empty lines", v)
		}
	}

	// Verify fleet.json on disk was NEVER modified
	infoAfter, err := os.Stat(fleetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !infoAfter.ModTime().Equal(infoBefore.ModTime()) {
		t.Fatalf("fleet.json modtime changed: %v -> %v", infoBefore.ModTime(), infoAfter.ModTime())
	}
	contentAfter, err := os.ReadFile(fleetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contentAfter) != string(initialContent) {
		t.Fatalf("fleet.json was mutated on disk: %s", string(contentAfter))
	}
}
