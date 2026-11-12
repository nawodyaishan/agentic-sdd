package tui

// W2-C recovery tests: managed backup listing, exact preview, default-No
// restore consent, real result/backup and stale/modified/legacy refusals. All
// fixtures are fake or temporary; no real home or network is touched.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

func fleetRecoveryModel(t *testing.T, live bool) (FleetModel, *trackingFleetReader, *fakeFleetPlanner) {
	t.Helper()
	planner := &fakeFleetPlanner{
		backups: []FleetBackupView{{
			ID:        "20260101T000000.000000000Z",
			Operation: "apply",
			State:     "applied",
			CreatedAt: "2026-01-01T00:00:00Z",
			Files:     1,
		}},
		restore: testFleetRestorePreview(),
	}
	model, reader := newFleetWriteModel(t, live, planner)
	return model, reader, planner
}

func loadFleetBackups(t *testing.T, model FleetModel) FleetModel {
	t.Helper()
	model, msg := fleetUpdateKey(t, model, 'e', "e")
	if model.ActiveView() != FleetViewRecovery {
		t.Fatalf("view = %v, want Recovery", model.ActiveView())
	}
	if msg == nil {
		t.Fatal("entering Recovery did not request the managed backup list")
	}
	model, _ = model.Update(msg)
	return model
}

func loadFleetRestorePreview(t *testing.T, model FleetModel) FleetModel {
	t.Helper()
	model, msg := fleetUpdateKey(t, model, tea.KeyEnter, "")
	if msg == nil {
		t.Fatal("Enter did not request a restore preview")
	}
	model, _ = model.Update(msg)
	return model
}

func TestFleetRecoveryListsBackupsAndExactPreview(t *testing.T) {
	model, _, planner := fleetRecoveryModel(t, true)
	model = loadFleetBackups(t, model)
	if planner.backupCalls != 1 {
		t.Fatalf("backup list calls = %d, want 1", planner.backupCalls)
	}
	if len(model.Backups()) != 1 {
		t.Fatalf("backups = %+v, want one managed backup", model.Backups())
	}

	model = loadFleetRestorePreview(t, model)
	if planner.restoreCalls != 1 {
		t.Fatalf("restore preview calls = %d, want 1", planner.restoreCalls)
	}
	preview := model.RestorePreview()
	if !preview.Available || preview.Token == "" {
		t.Fatalf("restore preview = %+v", preview)
	}

	view := strings.Join(model.Lines(110), "\n")
	for _, want := range []string{
		"Managed backups (fleet)",
		"20260101T000000.000000000Z",
		"Selected backup",
		"Backup identity: 20260101T000000.000000000Z",
		"Ownership: only files recorded by this managed transaction are touched. Unrelated files are never deleted.",
		"Restore impact",
		"Files: 1 total · 0 create · 1 update · 0 remove · 0 no-op",
		"Restore files",
		"update · sdd-implement-1",
		"Restore diff (per file)",
		"-old agent",
		"+restored agent",
		"Restore capability: live (reviewed build).",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("recovery view missing %q:\n%s", want, view)
		}
	}
}

func TestFleetRestoreConfirmationDefaultsToNoAndDispatchesOnce(t *testing.T) {
	model, _, _ := fleetRecoveryModel(t, true)
	model = loadFleetBackups(t, model)
	model = loadFleetRestorePreview(t, model)

	model, _ = model.requestRestore()
	if !model.IsConfirmingRestore() {
		t.Fatal("requestRestore did not open the confirmation")
	}

	// Enter keeps the default No.
	model, msg := fleetUpdateKey(t, model, tea.KeyEnter, "")
	if model.IsConfirmingRestore() {
		t.Fatal("Enter confirmed the restore; the default must be No")
	}
	if msg != nil {
		t.Fatalf("Enter produced %T, want no command", msg)
	}

	model, _ = model.requestRestore()
	model, msg = fleetUpdateKey(t, model, 'y', "y")
	start, ok := msg.(StartWriteMsg)
	if !ok {
		t.Fatalf("y produced %T, want StartWriteMsg", msg)
	}
	if start.Request.Kind != WriteFleetRestore {
		t.Fatalf("kind = %v, want WriteFleetRestore", start.Request.Kind)
	}
	fleetReq := start.Request.Fleet
	if fleetReq.BackupID != "20260101T000000.000000000Z" || fleetReq.Token != "restore-token-abc" {
		t.Fatalf("restore identity = %+v", fleetReq)
	}
	if fleetReq.PlanID != "" || fleetReq.Plan != nil {
		t.Fatalf("restore request carried a plan identity: %+v", fleetReq)
	}
	if fleetReq.Source == "" || fleetReq.Source == "embedded" {
		t.Fatalf("restore source = %q, want a non-embedded source", fleetReq.Source)
	}

	model, second := fleetUpdateKey(t, model, 'y', "y")
	if second != nil {
		t.Fatalf("second y produced %T, want no second dispatch", second)
	}
}

func TestFleetRecoveryRefusalsExplainWhatHappened(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "legacy", err: fleet.ErrLegacyProvenance, want: "legacy/unknown provenance"},
		{name: "modified target", err: fleet.ErrUnsafeTarget, want: "modified, recreated or is not a safe regular file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model, _, planner := fleetRecoveryModel(t, true)
			planner.restoreErr = tc.err
			model = loadFleetBackups(t, model)
			model = loadFleetRestorePreview(t, model)
			if model.RestorePreview().Available {
				t.Fatal("a refused preview was rendered as available")
			}
			view := strings.Join(model.Lines(110), "\n")
			for _, want := range []string{
				"What happened:",
				"What stayed unchanged:",
				"Next safe action:",
				tc.want,
			} {
				if !strings.Contains(view, want) {
					t.Errorf("refusal view missing %q:\n%s", want, view)
				}
			}
			lower := strings.ToLower(view)
			for _, forbidden := range []string{"adopt", "force", "fix-all", "fix all"} {
				if strings.Contains(lower, forbidden) {
					t.Errorf("refusal view exposes %q:\n%s", forbidden, view)
				}
			}

			// A refused preview cannot open a restore confirmation.
			model, _ = model.requestRestore()
			if model.IsConfirmingRestore() {
				t.Fatal("restore confirmation opened without an available preview")
			}
		})
	}
}

func TestFleetRestoreResultShowsRealOutcomeAndRefreshes(t *testing.T) {
	model, reader, planner := fleetRecoveryModel(t, true)
	model = loadFleetBackups(t, model)
	model = loadFleetRestorePreview(t, model)
	model, _ = model.requestRestore()
	model, msg := fleetUpdateKey(t, model, 'y', "y")
	start := msg.(StartWriteMsg)

	model, _ = model.Update(WriteStartedMsg{ID: opIDForTest(2), Kind: WriteFleetRestore})
	beforeStatus := reader.calls
	beforeBackups := planner.backupCalls
	model, cmd := model.Update(WriteResultMsg{
		ID:   opIDForTest(2),
		Kind: WriteFleetRestore,
		Result: appops.ApplyResult{
			Outcome:      skillsync.OutcomeApplied,
			FleetRestore: &fleet.RestoreResult{Restored: 1, Removed: 0, BackupID: "backup-restore-1"},
		},
	})
	if start.Request.Kind != WriteFleetRestore {
		t.Fatalf("dispatch kind = %v", start.Request.Kind)
	}
	if model.WritePending() {
		t.Fatal("a real restore result did not clear the pending-write guard")
	}
	outcome := model.RestoreOutcome()
	if outcome.Headline != "completed" || !strings.Contains(outcome.Detail, "Restored 1") {
		t.Fatalf("outcome = %+v, want the real restored count", outcome)
	}
	if outcome.BackupID != "backup-restore-1" {
		t.Fatalf("backup = %q, want backup-restore-1", outcome.BackupID)
	}
	if cmd == nil {
		t.Fatal("the restore result did not queue a status/backup refresh")
	}
	model = drainFleetCommands(t, model, cmd)
	if reader.calls != beforeStatus+1 {
		t.Fatalf("status reads = %d, want %d", reader.calls, beforeStatus+1)
	}
	if planner.backupCalls != beforeBackups+1 {
		t.Fatalf("backup list calls = %d, want %d", planner.backupCalls, beforeBackups+1)
	}
}

// drainFleetCommands feeds model messages from one command, including nested
// batches.
func drainFleetCommands(t *testing.T, model FleetModel, cmd tea.Cmd) FleetModel {
	t.Helper()
	for _, msg := range fleetCmdMessages(cmd) {
		if msg == nil {
			continue
		}
		next, follow := model.Update(msg)
		model = next
		if follow != nil {
			model = drainFleetCommands(t, model, follow)
		}
	}
	return model
}
