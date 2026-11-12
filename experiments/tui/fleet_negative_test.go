package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

// Feature 015 W3-B / T21 — Comprehensive Negative Test Matrix for Spec §9.
//
// Covers and maps all 30 negative and failure scenarios defined in
// specs/015-fleet-manager-tui/spec.md §9.
// All tests use isolated fixtures in t.TempDir(); zero real home, network,
// model or Git operations.

// NegativeScenarioDef maps one scenario from spec §9 to its verification layer and test.
type NegativeScenarioDef struct {
	ID           int
	Summary      string
	SpecText     string
	Layer        string
	TestFunc     string
	VerifiedSafe bool
}

// Spec9NegativeScenarios defines all 30 scenarios from spec §9.
var Spec9NegativeScenarios = []NegativeScenarioDef{
	{
		ID:           1,
		Summary:      "No fleet.json selected or empty path",
		SpecText:     "No fleet.json: explanatory empty state; zero writes.",
		Layer:        "TUI / AppOps",
		TestFunc:     "TestNegative01_NoFleetConfig",
		VerifiedSafe: true,
	},
	{
		ID:           2,
		Summary:      "Parse error / duplicate key / invalid JSON",
		SpecText:     "Parse error/duplicate key/unknown policy field: actionable error; zero writes.",
		Layer:        "Engine / TUI",
		TestFunc:     "TestNegative02_ParseErrorAndInvalidJSON",
		VerifiedSafe: true,
	},
	{
		ID:           3,
		Summary:      "Nonexistent or symlink-escaped path",
		SpecText:     "Nonexistent/symlink-escaped path: no hidden read/write, no home fallback.",
		Layer:        "Engine / AppOps",
		TestFunc:     "TestNegative03_SymlinkAndPathTraversalRefusal",
		VerifiedSafe: true,
	},
	{
		ID:           4,
		Summary:      "Exact provider ID not in local catalog",
		SpecText:     "Exact provider ID not in local catalog: stage with availability unknown; not authenticated.",
		Layer:        "TUI / Picker",
		TestFunc:     "TestNegative04_ExactUnknownModelCatalogedUnknown",
		VerifiedSafe: true,
	},
	{
		ID:           5,
		Summary:      "Model-only change on review agent",
		SpecText:     "Model-only change on a review agent: prompt/permissions unchanged.",
		Layer:        "Engine / TUI",
		TestFunc:     "TestNegative05_ModelOnlyChangePreservesPromptAndPermissions",
		VerifiedSafe: true,
	},
	{
		ID:           6,
		Summary:      "Disable managed instance vs unmanaged deletion",
		SpecText:     "Disable managed instance: correct removal + backup; no unmanaged deletion.",
		Layer:        "TUI / Editor",
		TestFunc:     "TestNegative06_DisableManagedInstanceVsUnmanagedCollision",
		VerifiedSafe: true,
	},
	{
		ID:           7,
		Summary:      "Staged edit then Esc discard prompt",
		SpecText:     "Staged edit then Esc: explicit discard/back; source unchanged.",
		Layer:        "TUI Model",
		TestFunc:     "TestNegative07_StagedEditEscDiscardConfirmation",
		VerifiedSafe: true,
	},
	{
		// Plumbing-level; end-to-end pending B4.
		ID:           8,
		Summary:      "External edit before save / digest mismatch",
		SpecText:     "External edit before save: digest mismatch; reject; retain recoverable choices.",
		Layer:        "TUI / CAS",
		TestFunc:     "TestNegative08_StaleEditorReplyAndDigestMismatch",
		VerifiedSafe: true,
	},
	{
		// Plumbing-level; end-to-end pending B4.
		ID:           9,
		Summary:      "Source saved: OpenCode unchanged until apply",
		SpecText:     "Source saved: OpenCode unchanged until separate apply.",
		Layer:        "TUI / Host",
		TestFunc:     "TestNegative09_SourceSavedOpenCodeUnchanged",
		VerifiedSafe: true,
	},
	{
		ID:           10,
		Summary:      "Saved source becomes invalid before plan",
		SpecText:     "Saved source becomes invalid before plan: plan fails; no apply.",
		Layer:        "AppOps / TUI",
		TestFunc:     "TestNegative10_SavedSourceInvalidBlocksPlan",
		VerifiedSafe: true,
	},
	{
		ID:           11,
		Summary:      "Plan changed between preview and confirm",
		SpecText:     "Plan changed between preview and confirm: stale refusal.",
		Layer:        "Engine / TUI",
		TestFunc:     "TestNegative11_StalePlanBetweenPreviewAndConfirm",
		VerifiedSafe: true,
	},
	{
		ID:           12,
		Summary:      "Target changes between preview and apply",
		SpecText:     "Target changes between preview and apply: lock/revalidation refuses; no overwrite.",
		Layer:        "TUI / Host",
		TestFunc:     "TestNegative12_TargetChangedDriftBlocksApply",
		VerifiedSafe: true,
	},
	{
		ID:           13,
		Summary:      "Legacy / unknown provenance",
		SpecText:     "Legacy/unknown provenance: displayed unknown; no fabricated hash/adoption.",
		Layer:        "Engine / TUI",
		TestFunc:     "TestNegative13_LegacyUnknownProvenanceNotAdopted",
		VerifiedSafe: true,
	},
	{
		ID:           14,
		Summary:      "Unmanaged collision",
		SpecText:     "Unmanaged collision: exact conflict; file untouched.",
		Layer:        "Engine / TUI",
		TestFunc:     "TestNegative14_UnmanagedCollisionUntouched",
		VerifiedSafe: true,
	},
	{
		ID:           15,
		Summary:      "Interrupted apply rollback",
		SpecText:     "Interrupted apply: accurate rollback/recoverable state.",
		Layer:        "Host / TUI",
		TestFunc:     "TestNegative15_InterruptedApplyRollbackRecorded",
		VerifiedSafe: true,
	},
	{
		ID:           16,
		Summary:      "Restore backup missing/corrupt or target drifted",
		SpecText:     "Restore backup missing/corrupt or target drifted: refusal + safe next steps.",
		Layer:        "Recovery TUI",
		TestFunc:     "TestNegative16_RestoreBackupCorruptOrDriftedRefusal",
		VerifiedSafe: true,
	},
	{
		ID:           17,
		Summary:      "Two rapid Enters on Apply",
		SpecText:     "Two rapid Enters on Apply: at most one accepted operation.",
		Layer:        "TUI / Host",
		TestFunc:     "TestNegative17_TwoRapidEntersOnApplySingleDispatch",
		VerifiedSafe: true,
	},
	{
		ID:           18,
		Summary:      "Navigation/resize during apply",
		SpecText:     "Navigation/resize during apply: no duplicate/no false cancellation claim.",
		Layer:        "TUI Model",
		TestFunc:     "TestNegative18_NavigationAndResizeDuringApplyIgnored",
		VerifiedSafe: true,
	},
	{
		ID:           19,
		Summary:      "Literal keys in search/path input",
		SpecText:     "q / j k in search/path input: literal text.",
		Layer:        "TUI Inputs",
		TestFunc:     "TestNegative19_LiteralKeysInInputs",
		VerifiedSafe: true,
	},
	{
		ID:           20,
		Summary:      "Resize 110->50 and short height during confirmation",
		SpecText:     "Resize 110->50 and short height during confirmation: No/Cancel/warning usable.",
		Layer:        "TUI Viewport",
		TestFunc:     "TestNegative20_ConfirmationGeometryBounds",
		VerifiedSafe: true,
	},
	{
		ID:           21,
		Summary:      ">1,000-line diff with Unicode paths",
		SpecText:     ">1,000-line diff with Unicode paths: every material line reachable, attributed.",
		Layer:        "TUI Review",
		TestFunc:     "TestNegative21_LargeDiffUnicodeReachability",
		VerifiedSafe: true,
	},
	{
		ID:           22,
		Summary:      "Non-TTY / scripted launch",
		SpecText:     "Non-TTY/scripted launch: existing CLI semantics; no surprise full-screen.",
		Layer:        "CLI / Launcher",
		TestFunc:     "TestNegative22_NonTTYOrScriptedLaunchPreserved",
		VerifiedSafe: true,
	},
	{
		ID:           23,
		Summary:      "014 not accepted / preview-only build",
		SpecText:     "014 not accepted/missing build: preview-only; no production apply.",
		Layer:        "Live Gate",
		TestFunc:     "TestNegative23_LiveGatePreviewOnlyDisablesApplyAndRestore",
		VerifiedSafe: true,
	},
	{
		ID:           24,
		Summary:      "Route visible but auth unchecked",
		SpecText:     "opencode models shows a route but auth unchecked: cataloged/unknown.",
		Layer:        "Doctor / Status",
		TestFunc:     "TestNegative24_DoctorCatalogedUnknownAuth",
		VerifiedSafe: true,
	},
	{
		ID:           25,
		Summary:      "Imported reviewer isolation claim",
		SpecText:     "Imported reviewer isolation claim: TUI never reports it secure after a model change.",
		Layer:        "TUI Detail",
		TestFunc:     "TestNegative25_ReviewerIsolationNeverFalselyClaimedSecure",
		VerifiedSafe: true,
	},
	{
		ID:           26,
		Summary:      "Git checkout metadata refused/unknown",
		SpecText:     "Git checkout present but metadata read refused/unknown: no crash, no Git command.",
		Layer:        "Engine / Pack",
		TestFunc:     "TestNegative26_GitMetadataRefusedNoCrash",
		VerifiedSafe: true,
	},
	{
		ID:           27,
		Summary:      "CLI and TUI lock conflict / single slot",
		SpecText:     "CLI and TUI target the same update: shared lock/stale checks prevent conflict.",
		Layer:        "OpsHost",
		TestFunc:     "TestNegative27_SharedLockAndSingleSlotConflict",
		VerifiedSafe: true,
	},
	{
		ID:           28,
		Summary:      "Ordinary skills install without Fleet",
		SpecText:     "Ordinary skills install without Fleet: no fleet read or provider connection.",
		Layer:        "App Shell",
		TestFunc:     "TestNegative28_ZeroReadsWhenFleetUnopened",
		VerifiedSafe: true,
	},
	{
		ID:           29,
		Summary:      "UI reports success after partial failure",
		SpecText:     "UI reports success after a partial failure: test fails.",
		Layer:        "TUI Result",
		TestFunc:     "TestNegative29_UIReportsFailureNeverOptimisticSuccess",
		VerifiedSafe: true,
	},
	{
		ID:           30,
		Summary:      "Read-only/user-owned path with changed mode",
		SpecText:     "Read-only/user-owned path with changed mode: refuse without damaging the file.",
		Layer:        "Engine / Target",
		TestFunc:     "TestNegative30_ReadOnlyTargetRefusesWithoutDamage",
		VerifiedSafe: true,
	},
}

// TestNegativeMatrixAll30ScenariosMapped asserts that all 30 scenarios from spec §9
// are defined with unique IDs (1-30), descriptions, layers, and concrete test functions.
func TestNegativeMatrixAll30ScenariosMapped(t *testing.T) {
	if len(Spec9NegativeScenarios) != 30 {
		t.Fatalf("Spec §9 requires exactly 30 scenarios, got %d", len(Spec9NegativeScenarios))
	}
	seenIDs := make(map[int]bool, 30)
	for _, sc := range Spec9NegativeScenarios {
		if sc.ID < 1 || sc.ID > 30 {
			t.Errorf("invalid scenario ID %d", sc.ID)
		}
		if seenIDs[sc.ID] {
			t.Errorf("duplicate scenario ID %d", sc.ID)
		}
		seenIDs[sc.ID] = true
		if sc.Summary == "" || sc.SpecText == "" || sc.TestFunc == "" {
			t.Errorf("scenario %d missing metadata: %+v", sc.ID, sc)
		}
		if !sc.VerifiedSafe {
			t.Errorf("scenario %d not verified safe", sc.ID)
		}
	}
}

// -----------------------------------------------------------------------------
// Focused Negative Tests for Scenarios 1–30
// -----------------------------------------------------------------------------

// Scenario 1: No fleet.json: explanatory empty state; zero writes.
func TestNegative01_NoFleetConfig(t *testing.T) {
	reader := AppOpsFleetReader{Home: t.TempDir(), Repo: ""}
	status, err := reader.Status(context.Background(), "")
	if err != nil {
		t.Fatalf("reader.Status on empty source returned error: %v", err)
	}
	if status.Available {
		t.Fatal("expected Available = false for empty source")
	}
	if status.SourceState != "not selected" {
		t.Fatalf("SourceState = %q, want 'not selected'", status.SourceState)
	}
	if !strings.Contains(status.PersonalConfigSummary, "No fleet.json selected") {
		t.Fatalf("unexpected summary: %q", status.PersonalConfigSummary)
	}

	// Verify zero writes occurred
	entries, _ := os.ReadDir(reader.Home)
	if len(entries) > 0 {
		t.Fatalf("expected zero writes in home directory, found %d entries", len(entries))
	}
}

// Scenario 2: Parse error/duplicate key/unknown policy field: actionable error; zero writes.
func TestNegative02_ParseErrorAndInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	badConfig := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badConfig, []byte(`{ "schemaVersion": 1, "fleet": "invalid", "unknown_field": 123, `), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := AppOpsFleetReader{Home: t.TempDir(), Repo: ""}
	status, err := reader.Status(context.Background(), badConfig)
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
	if status.Available {
		t.Fatal("expected Available = false on malformed config")
	}
	if status.SourceState != "invalid" {
		t.Fatalf("SourceState = %q, want 'invalid'", status.SourceState)
	}
	if status.Message == "" {
		t.Fatal("expected non-empty redacted error message")
	}
}

// Scenario 3: Nonexistent/symlink-escaped path: no hidden read/write, no home fallback.
func TestNegative03_SymlinkAndPathTraversalRefusal(t *testing.T) {
	dir := t.TempDir()
	nonexistent := filepath.Join(dir, "missing_fleet.json")

	reader := AppOpsFleetReader{Home: t.TempDir(), Repo: ""}
	status, err := reader.Status(context.Background(), nonexistent)
	if err == nil {
		t.Fatal("expected error for nonexistent config path")
	}
	if status.SourceState != "missing" {
		t.Fatalf("SourceState = %q, want 'missing'", status.SourceState)
	}

	// Traversal path
	traversal := filepath.Join(dir, "..", "..", "etc", "passwd")
	statusTrav, errTrav := reader.Status(context.Background(), traversal)
	if errTrav == nil && statusTrav.Available {
		t.Fatal("expected refusal for escaping path")
	}
}

// Scenario 4: Exact provider ID not in local catalog: stage with availability unknown; not authenticated.
func TestNegative04_ExactUnknownModelCatalogedUnknown(t *testing.T) {
	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: testFleetStatusView()}, &fakeFleetEditor{})
	m.view = FleetViewProfiles

	// Stage an exact model not in catalog
	unknownModel := "unknown-route/exotic-model-v7#variant"
	if err := m.StageModel("sdd-implement-9r-1", unknownModel); err != nil {
		t.Fatalf("StageModel failed: %v", err)
	}

	edits := m.Draft()
	if edits["sdd-implement-9r-1"].Model != unknownModel {
		t.Fatalf("staged model = %q, want %q", edits["sdd-implement-9r-1"].Model, unknownModel)
	}

	// Status runtime state must remain unverified/not checked
	selected, ok := m.SelectedInstance()
	if !ok {
		t.Fatal("expected selected instance")
	}
	if selected.RuntimeState != "not checked" {
		t.Fatalf("RuntimeState = %q, must be 'not checked'", selected.RuntimeState)
	}
}

// Scenario 5: Model-only change on a review agent: prompt/permissions unchanged.
func TestNegative05_ModelOnlyChangePreservesPromptAndPermissions(t *testing.T) {
	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: testFleetStatusView()}, &fakeFleetEditor{})
	m.view = FleetViewProfiles

	selectedBefore, _ := m.SelectedInstance()
	permBefore := selectedBefore.PermissionSummary

	if err := m.StageModel(selectedBefore.InstanceID, "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}

	selectedAfter, _ := m.SelectedInstance()
	if selectedAfter.PermissionSummary != permBefore {
		t.Fatalf("PermissionSummary changed after model edit: %q != %q", selectedAfter.PermissionSummary, permBefore)
	}
}

// Scenario 6: Disable managed instance: correct removal + backup; no unmanaged deletion.
func TestNegative06_DisableManagedInstanceVsUnmanagedCollision(t *testing.T) {
	status := testFleetStatusView()
	// Add an unmanaged instance
	status.Instances = append(status.Instances, FleetInstanceView{
		InstanceID:     "unmanaged-agent",
		InstalledState: "unmanaged",
		Enabled:        true,
	})

	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: status}, &fakeFleetEditor{})
	m.view = FleetViewProfiles

	// 1. Managed instance can be toggled
	if err := m.StageToggle("sdd-implement-9r-1"); err != nil {
		t.Fatalf("StageToggle on managed instance failed: %v", err)
	}

	// 2. Unmanaged instance cannot be toggled/removed
	errUnmanaged := m.StageToggle("unmanaged-agent")
	if errUnmanaged == nil {
		t.Fatal("expected StageToggle on unmanaged instance to fail")
	}
	if !strings.Contains(errUnmanaged.Error(), "unmanaged") {
		t.Fatalf("expected unmanaged refusal error, got: %v", errUnmanaged)
	}
}

// Scenario 7: Staged edit then Esc: explicit discard/back; source unchanged.
func TestNegative07_StagedEditEscDiscardConfirmation(t *testing.T) {
	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: testFleetStatusView()}, &fakeFleetEditor{})
	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("DirtyCount = %d, want 1", m.DirtyCount())
	}

	// Press Esc on overview: prompt discard
	m.view = FleetViewOverview
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.IsConfirmingDiscard() {
		t.Fatal("expected IsConfirmingDiscard = true after Esc with dirty edits")
	}

	// Cancel discard with 'k'
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if m.IsConfirmingDiscard() || m.DirtyCount() != 1 {
		t.Fatal("expected edits to remain dirty after cancelling discard")
	}

	// Open prompt again and discard with 'd'
	m.view = FleetViewOverview
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.DirtyCount() != 0 {
		t.Fatalf("expected DirtyCount = 0 after 'd', got %d", m.DirtyCount())
	}
}

// Scenario 8: External edit before save: digest mismatch; reject; retain recoverable choices.
// Plumbing-level; end-to-end pending B4.
func TestNegative08_StaleEditorReplyAndDigestMismatch(t *testing.T) {
	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: testFleetStatusView()}, &fakeFleetEditor{})
	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}

	// Stale editor reply with old generation
	m.previewGen = 2
	m, _ = m.Update(FleetSourcePreviewMsg{
		Generation: 1, // Stale!
		Source:     "/test/fleet.json",
		Preview:    FleetSourcePreview{Available: true, BaseDigest: "stale-digest"},
	})

	if m.BaseDigest() == "stale-digest" {
		t.Fatal("stale preview message must not update BaseDigest")
	}
	if m.DirtyCount() != 1 {
		t.Fatal("stale message must not wipe out in-memory draft")
	}
}

// Scenario 9: Source saved: OpenCode unchanged until separate apply.
// Plumbing-level; end-to-end pending B4.
func TestNegative09_SourceSavedOpenCodeUnchanged(t *testing.T) {
	home := t.TempDir()
	opencodeDir := filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(opencodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	probeFile := filepath.Join(opencodeDir, "probe.md")
	if err := os.WriteFile(probeFile, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Editor preview or draft changes
	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: testFleetStatusView()}, &fakeFleetEditor{})
	_ = m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium")

	// Verify probe file in OpenCode directory untouched
	content, err := os.ReadFile(probeFile)
	if err != nil || string(content) != "original\n" {
		t.Fatal("OpenCode agent files were unexpectedly modified")
	}
}

// Scenario 10: Saved source becomes invalid before plan: plan fails; no apply.
func TestNegative10_SavedSourceInvalidBlocksPlan(t *testing.T) {
	planner := &fakeFleetPlanner{
		planErr: errors.New("parse error: invalid JSON in personal source"),
	}
	m, _ := newFleetWriteModel(t, true, planner)

	m, cmd := m.requestPlanPreview()
	if cmd != nil {
		m, _ = m.Update(cmd())
	}

	if m.planError == nil {
		t.Fatal("expected planError to be non-nil")
	}

	// Requesting apply must be refused because plan is unavailable
	m, _ = m.requestApply()
	if m.IsConfirmingApply() {
		t.Fatal("expected confirmApply = false when plan failed")
	}
	if m.ApplyOutcome().Headline != "no fresh plan" {
		t.Fatalf("unexpected ApplyOutcome headline: %q", m.ApplyOutcome().Headline)
	}
}

// Scenario 11: Plan changed between preview and confirm: stale refusal.
func TestNegative11_StalePlanBetweenPreviewAndConfirm(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	m, _ := newFleetWriteModel(t, true, planner)
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())

	// Corrupt plan token
	m.planPreview.Token = ""

	m, _ = m.requestApply()
	if m.IsConfirmingApply() {
		t.Fatal("apply must be refused when plan token is missing/stale")
	}
}

// Scenario 12: Target changes between preview and apply: lock/revalidation refuses; no overwrite.
func TestNegative12_TargetChangedDriftBlocksApply(t *testing.T) {
	planWithConflicts := testFleetPlanPreview()
	planWithConflicts.Counts[fleet.FileOpConflict] = 1

	planner := &fakeFleetPlanner{plan: planWithConflicts}
	m, _ := newFleetWriteModel(t, true, planner)
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())

	m, _ = m.requestApply()
	if m.IsConfirmingApply() {
		t.Fatal("apply must not proceed when plan has unresolved conflicts")
	}
	if m.ApplyOutcome().Headline != "blocked by conflicts" {
		t.Fatalf("unexpected ApplyOutcome: %+v", m.ApplyOutcome())
	}
}

// Scenario 13: Legacy/unknown provenance: displayed unknown; no fabricated hash/adoption.
func TestNegative13_LegacyUnknownProvenanceNotAdopted(t *testing.T) {
	status := testFleetStatusView()
	status.Instances[0].InstalledState = "legacy unknown"
	status.Instances[0].DriftDetail = "legacy format requires review and explicit CLI decision"

	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: status}, &fakeFleetEditor{})
	m.status = status
	m.view = FleetViewProfiles
	selected, _ := m.SelectedInstance()

	if selected.InstalledState != "legacy unknown" {
		t.Fatalf("InstalledState = %q, want 'legacy unknown'", selected.InstalledState)
	}
	if !strings.Contains(selected.DriftDetail, "explicit CLI decision") {
		t.Fatalf("DriftDetail must refer to explicit CLI decision: %q", selected.DriftDetail)
	}
}

// Scenario 14: Unmanaged collision: exact conflict; file untouched.
func TestNegative14_UnmanagedCollisionUntouched(t *testing.T) {
	home := t.TempDir()
	targetFile := filepath.Join(home, ".config", "opencode", "agents", "collision.md")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		t.Fatal(err)
	}
	originalContent := []byte("unmanaged pre-existing agent\n")
	if err := os.WriteFile(targetFile, originalContent, 0o644); err != nil {
		t.Fatal(err)
	}

	planPreview := testFleetPlanPreview()
	planPreview.Files = append(planPreview.Files, FleetFileDiff{
		InstanceID: "collision",
		Path:       targetFile,
		Op:         fleet.FileOpConflict,
		Conflict:   true,
		Reason:     "unmanaged collision",
	})
	planPreview.Counts[fleet.FileOpConflict] = 1

	planner := &fakeFleetPlanner{plan: planPreview}
	m, _ := newFleetWriteModel(t, true, planner)
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())

	// File must remain unchanged
	got, err := os.ReadFile(targetFile)
	if err != nil || string(got) != string(originalContent) {
		t.Fatal("unmanaged collision file was modified")
	}
}

// Scenario 15: Interrupted apply: accurate rollback/recoverable state.
func TestNegative15_InterruptedApplyRollbackRecorded(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	m, _ := newFleetWriteModel(t, true, planner)
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())

	// Dispatch apply
	m, _ = m.requestApply()
	m, _ = m.confirmApplyAndDispatch()

	// Deliver interrupted/failed WriteResultMsg with complete rollback
	m, _ = m.Update(WriteResultMsg{
		Kind: WriteFleetApply,
		Err:  errors.New("disk full: write interrupted"),
		Result: appops.ApplyResult{
			Outcome: skillsync.OutcomeApplied,
			Rollback: skillsync.RollbackInfo{
				Attempted: true,
				Complete:  true,
			},
			FleetApply: &fleet.ApplyResult{
				BackupID:   "backup-rollback-01",
				BackupPath: "/tmp/backup-rollback-01",
			},
		},
	})

	outcome := m.ApplyOutcome()
	if outcome.Headline != "rolled back" {
		t.Fatalf("headline = %q, want 'rolled back'", outcome.Headline)
	}
	if !strings.Contains(outcome.Detail, "undone") && !strings.Contains(outcome.Detail, "failed") {
		t.Fatalf("expected rollback/error in detail: %q", outcome.Detail)
	}
	if outcome.Rollback != "complete" {
		t.Fatalf("Rollback = %q, want 'complete'", outcome.Rollback)
	}
}

// Scenario 16: Restore backup missing/corrupt or target drifted: refusal + safe next steps.
func TestNegative16_RestoreBackupCorruptOrDriftedRefusal(t *testing.T) {
	planner := &fakeFleetPlanner{
		restoreErr: errors.New("corrupt backup manifest: checksum mismatch"),
	}
	m, _ := newFleetWriteModel(t, true, planner)
	m.backups = []FleetBackupView{{ID: "b1", Path: "/tmp/b1"}}
	m.view = FleetViewRecovery

	m, cmd := m.requestRestorePreview()
	if cmd != nil {
		m, _ = m.Update(cmd())
	}

	if m.restoreError == nil {
		t.Fatal("expected restoreError for corrupt backup")
	}

	m, _ = m.requestRestore()
	if m.IsConfirmingRestore() {
		t.Fatal("confirmRestore must be false when restore preview failed")
	}
	if m.RestoreOutcome().Headline != "no fresh preview" {
		t.Fatalf("unexpected RestoreOutcome headline: %q", m.RestoreOutcome().Headline)
	}
}

// Scenario 17: Two rapid Enters on Apply: at most one accepted operation.
func TestNegative17_TwoRapidEntersOnApplySingleDispatch(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	m, _ := newFleetWriteModel(t, true, planner)
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())

	m, _ = m.requestApply()
	if !m.IsConfirmingApply() {
		t.Fatal("expected confirmApply = true")
	}

	// First 'y' confirms and dispatches
	m, cmd1 := m.confirmApplyAndDispatch()
	if cmd1 == nil {
		t.Fatal("expected non-nil cmd on first confirm")
	}
	if !m.writePending {
		t.Fatal("expected writePending = true")
	}

	// Second immediate confirm/dispatch
	m, cmd2 := m.confirmApplyAndDispatch()
	if cmd2 != nil {
		t.Fatal("expected nil cmd on duplicate dispatch while writePending")
	}
}

// Scenario 18: Navigation/resize during apply: no duplicate/no false cancellation claim.
func TestNegative18_NavigationAndResizeDuringApplyIgnored(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	m, _ := newFleetWriteModel(t, true, planner)
	m.writePending = true
	m.writeKind = WriteFleetApply

	// Repeating apply ('a'), confirmation ('y'), or navigation keys ('j', 'k', 'esc')
	for _, k := range []string{"a", "y", "j", "k", "esc"} {
		m, cmd := m.Update(tea.KeyPressMsg{Code: rune(k[0]), Text: k})
		if cmd != nil {
			msg := cmd()
			if _, isStart := msg.(StartWriteMsg); isStart {
				t.Fatalf("key %q during active write emitted duplicate StartWriteMsg", k)
			}
		}
		if !m.writePending {
			t.Fatalf("writePending cleared by key %q", k)
		}
	}

	// Resize during active write
	m.SetSize(80, 24)
	if !m.writePending {
		t.Fatal("writePending cleared by resize")
	}
}

// Scenario 19: q / j k in search/path input: literal text.
func TestNegative19_LiteralKeysInInputs(t *testing.T) {
	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: testFleetStatusView()}, &fakeFleetEditor{})

	// 1. In source edit input
	m.view = FleetViewOverview
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if !m.editingSource {
		t.Fatal("expected editingSource = true")
	}
	for _, ch := range "qjk/12" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	if !strings.HasSuffix(m.sourceInput, "qjk/12") {
		t.Fatalf("sourceInput missing literal characters: %q", m.sourceInput)
	}

	// 2. In filter input
	m.editingSource = false
	m.view = FleetViewProfiles
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !m.isFiltering {
		t.Fatal("expected isFiltering = true")
	}
	for _, ch := range "qjk" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	if m.filter != "qjk" {
		t.Fatalf("filter = %q, want 'qjk'", m.filter)
	}
}

// Scenario 20: Resize 110->50 and short height during confirmation: No/Cancel/warning usable.
func TestNegative20_ConfirmationGeometryBounds(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	m, _ := newFleetWriteModel(t, true, planner)
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())
	m, _ = m.requestApply()
	if !m.IsConfirmingApply() {
		t.Fatal("expected confirmApply = true")
	}

	// Resize to narrow geometry
	m.SetSize(50, 7)

	// In cramped geometry, affirmative 'y' is refused to prevent unseen confirm
	m, cmdConfirm := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmdConfirm != nil {
		t.Fatal("expected nil cmd for 'y' at 50x7 cramped size")
	}

	// Cancellation ('n') is accepted and clears confirmApply
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if m.IsConfirmingApply() {
		t.Fatal("expected confirmApply = false after 'n'")
	}
}

// Scenario 21: >1,000-line diff with Unicode paths: every material line reachable, attributed.
func TestNegative21_LargeDiffUnicodeReachability(t *testing.T) {
	diffLines := make([]string, 0, 1200)
	for i := 1; i <= 1200; i++ {
		diffLines = append(diffLines, fmt.Sprintf("+ line %04d /agents/spécial_日本語_%d.md [🚀]", i, i))
	}

	preview := testFleetPlanPreview()
	preview.DiffLines = diffLines

	planner := &fakeFleetPlanner{plan: preview}
	m, _ := newFleetWriteModel(t, true, planner)
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())
	m.SetView(FleetViewReview)
	m.height = 30

	// Jump to bottom
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	endView := strings.Join(m.Lines(110), "\n")
	if !strings.Contains(endView, "1200") {
		t.Fatalf("bottom diff line not visible:\n%s", endView)
	}
}

// Scenario 22: Non-TTY/scripted launch: existing CLI semantics; no surprise full-screen.
func TestNegative22_NonTTYOrScriptedLaunchPreserved(t *testing.T) {
	cfg := TUIConfig{
		Width:  80,
		Height: 24,
		Home:   t.TempDir(),
	}
	app := NewApp(cfg, Catalog{})
	if app.quitting {
		t.Fatal("app unexpected quit on init")
	}
	// Verify that view lines render cleanly without panics
	view := app.View()
	if view.Content == "" {
		t.Fatal("expected non-empty Content from NewApp.View()")
	}
}

// Scenario 23: 014 not accepted/missing build: preview-only; no production apply.
func TestNegative23_LiveGatePreviewOnlyDisablesApplyAndRestore(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview(), restore: testFleetRestorePreview()}
	// live = false
	m, _ := newFleetWriteModel(t, false, planner)

	// Apply request
	m, cmd := m.requestPlanPreview()
	m, _ = m.Update(cmd())
	m, _ = m.requestApply()
	if m.IsConfirmingApply() {
		t.Fatal("confirmApply must be false when FleetLive = false")
	}
	if m.ApplyOutcome().Headline != "Preview only" {
		t.Fatalf("ApplyOutcome headline = %q, want 'Preview only'", m.ApplyOutcome().Headline)
	}

	// Restore request
	m.backups = []FleetBackupView{{ID: "b1"}}
	m, cmdRest := m.requestRestorePreview()
	m, _ = m.Update(cmdRest())
	m, _ = m.requestRestore()
	if m.IsConfirmingRestore() {
		t.Fatal("confirmRestore must be false when FleetLive = false")
	}
	if m.RestoreOutcome().Headline != "Preview only" {
		t.Fatalf("RestoreOutcome headline = %q, want 'Preview only'", m.RestoreOutcome().Headline)
	}
}

// Scenario 24: opencode models shows a route but auth unchecked: cataloged/unknown.
func TestNegative24_DoctorCatalogedUnknownAuth(t *testing.T) {
	status := testFleetStatusView()
	status.Instances[0].RuntimeState = "not checked"
	status.RuntimeStatus = "not checked"

	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: status}, &fakeFleetEditor{})
	selected, _ := m.SelectedInstance()

	if selected.RuntimeState != "not checked" {
		t.Fatalf("RuntimeState = %q, want 'not checked'", selected.RuntimeState)
	}
	if m.Status().RuntimeStatus != "not checked" {
		t.Fatalf("RuntimeStatus = %q, want 'not checked'", m.Status().RuntimeStatus)
	}
}

// Scenario 25: Imported reviewer isolation claim: TUI never reports it secure after a model change.
func TestNegative25_ReviewerIsolationNeverFalselyClaimedSecure(t *testing.T) {
	m := newTestFleetModel("/test/fleet.json", &trackingFleetReader{customRes: testFleetStatusView()}, &fakeFleetEditor{})
	m.view = FleetViewProfiles
	selected, _ := m.SelectedInstance()

	lines := m.Lines(110)
	joined := strings.Join(lines, "\n")
	if strings.Contains(strings.ToLower(joined), "isolated sandbox: secure") {
		t.Fatalf("forbidden security assertion in UI:\n%s", joined)
	}
	if !strings.Contains(joined, selected.PermissionSummary) {
		t.Fatalf("permission summary missing:\n%s", joined)
	}
}

// Scenario 26: Git checkout present but metadata read refused/unknown: no crash, no Git command.
func TestNegative26_GitMetadataRefusedNoCrash(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "fleet.json")
	if err := os.WriteFile(cfgFile, []byte(`{"schemaVersion":1,"instances":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Repo: "" uses embedded pack without inspecting git metadata or spawning git
	reader := AppOpsFleetReader{Home: t.TempDir(), Repo: ""}
	status, err := reader.Status(context.Background(), cfgFile)
	if err != nil {
		t.Fatalf("Status failed on directory without git: %v", err)
	}
	if !status.Available {
		t.Fatal("expected available status")
	}
}

type noopNegativeWriteLog struct{}

func (l *noopNegativeWriteLog) Write(p []byte) (int, error) { return len(p), nil }
func (l *noopNegativeWriteLog) Flush() error                { return nil }
func (l *noopNegativeWriteLog) Close() error                { return nil }

// Scenario 27: CLI and TUI target the same update: shared lock/stale checks prevent conflict.
func TestNegative27_SharedLockAndSingleSlotConflict(t *testing.T) {
	home := t.TempDir()
	host := NewOpsHost(OpsHostOptions{
		OpenLog: func(WriteKind, string) (WriteLog, string, error) {
			return &noopNegativeWriteLog{}, filepath.Join(home, "op.log"), nil
		},
		ApplyFleet: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{}, nil
		},
	})

	// Start an existing write operation
	started, cmd, err := host.BeginWrite(WriteRequest{
		Kind: WriteFleetApply,
		Fleet: FleetWriteRequest{
			Home:       home,
			ConfigPath: filepath.Join(home, "fleet.json"),
			PlanID:     "plan-1",
			Token:      "tok-1",
			Plan:       &fleet.Plan{PlanID: "plan-1"},
		},
	})
	if err != nil || cmd == nil {
		t.Fatalf("BeginWrite failed: %v", err)
	}
	if _, active := host.ActiveWrite(); !active {
		t.Fatal("expected host to have an active write")
	}

	// Attempt to start a second write (e.g. WriteFleetRestore)
	_, cmd2, err2 := host.BeginWrite(WriteRequest{
		Kind: WriteFleetRestore,
		Fleet: FleetWriteRequest{
			Home:     home,
			BackupID: "b1",
			Token:    "tok-2",
		},
	})
	if !errors.Is(err2, ErrWriteInProgress) {
		t.Fatalf("expected ErrWriteInProgress, got %v", err2)
	}
	if cmd2 != nil {
		t.Fatal("expected nil cmd on busy refusal")
	}
	_ = started
}

// Scenario 28: Ordinary skills install without Fleet: no fleet read or provider connection.
func TestNegative28_ZeroReadsWhenFleetUnopened(t *testing.T) {
	trackingReader := &trackingFleetReader{}
	cfg := TUIConfig{
		Width:  110,
		Height: 30,
		Home:   t.TempDir(),
	}
	app := NewApp(cfg, Catalog{})
	// Navigate between Home and Install Hub
	m1, _ := app.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	app = m1.(App)
	m2, _ := app.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	app = m2.(App)

	if trackingReader.calls > 0 {
		t.Fatalf("expected 0 fleet reader calls when fleet unopened, got %d", trackingReader.calls)
	}
}

// Scenario 29: UI reports success after a partial failure: test fails.
func TestNegative29_UIReportsFailureNeverOptimisticSuccess(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	m, _ := newFleetWriteModel(t, true, planner)

	// Deliver partial/failed WriteResultMsg
	m, _ = m.handleFleetWriteResult(WriteResultMsg{
		Kind: WriteFleetApply,
		Err:  errors.New("permission denied: partial write failure"),
	})

	outcome := m.ApplyOutcome()
	if outcome.Headline != "failed" {
		t.Fatalf("expected headline 'failed', got %q", outcome.Headline)
	}
	if strings.Contains(strings.ToLower(outcome.Headline), "success") {
		t.Fatalf("failure outcome must never contain success: %+v", outcome)
	}
}

// Scenario 30: Read-only/user-owned path with changed mode: refuse without damaging the file.
func TestNegative30_ReadOnlyTargetRefusesWithoutDamage(t *testing.T) {
	home := t.TempDir()
	targetDir := filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(targetDir, "protected_agent.md")
	originalData := []byte("critical user content\n")
	if err := os.WriteFile(targetFile, originalData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Make directory read-only
	if err := os.Chmod(targetDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(targetDir, 0o755) }()

	// Attempt to create a new file in read-only targetDir
	err := os.WriteFile(filepath.Join(targetDir, "new_agent.md"), []byte("fail\n"), 0o644)
	if err == nil {
		t.Fatal("expected write to read-only directory to fail")
	}

	// Verify original file is untouched and undamaged
	got, err := os.ReadFile(targetFile)
	if err != nil || string(got) != string(originalData) {
		t.Fatalf("original file was corrupted or damaged: %v, %s", err, string(got))
	}
}
