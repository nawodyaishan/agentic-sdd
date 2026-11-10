package skillsync

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestNormalizeClientSelection(t *testing.T) {
	if got, err := NormalizeClientSelection(nil); err != nil || got != nil {
		t.Fatalf("nil selection = %v, %v; want nil, nil", got, err)
	}
	if _, err := NormalizeClientSelection([]string{}); err == nil || !strings.Contains(err.Error(), "client selection is empty") {
		t.Fatalf("explicit empty selection error = %v", err)
	}
	if _, err := NormalizeClientSelection([]string{"claude", "nope"}); err == nil || !strings.Contains(err.Error(), `unknown client "nope"`) {
		t.Fatalf("unknown client error = %v", err)
	}
	if _, err := NormalizeClientSelection([]string{""}); err == nil || !strings.Contains(err.Error(), `unknown client ""`) {
		t.Fatalf("empty client ID error = %v", err)
	}

	input := []string{"agy", "claude", "agy"}
	got, err := NormalizeClientSelection(input)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"claude", "agy"}; !slices.Equal(got, want) {
		t.Fatalf("normalized selection = %v, want %v", got, want)
	}
	if !slices.Equal(input, []string{"agy", "claude", "agy"}) {
		t.Fatalf("normalization mutated the caller's slice: %v", input)
	}
	got[0] = "agents"
	if again, _ := NormalizeClientSelection(input); again[0] != "claude" {
		t.Fatalf("normalized selection aliases caller memory: %v", again)
	}

	if want := []string{"agents", "codex", "claude", "agy"}; !slices.Equal(SupportedClientIDs(), want) {
		t.Fatalf("supported client IDs = %v, want %v", SupportedClientIDs(), want)
	}
	var canonical []string
	for _, target := range clientTargets("") {
		canonical = append(canonical, target.name)
	}
	if !slices.Equal(SupportedClientIDs(), canonical) {
		t.Fatalf("supported IDs %v drifted from clientTargets %v", SupportedClientIDs(), canonical)
	}
	ids := SupportedClientIDs()
	ids[0] = "tampered"
	if SupportedClientIDs()[0] != "agents" {
		t.Fatal("SupportedClientIDs returned the canonical slice itself")
	}
}

func TestClientSelectionRejectedBeforeWrites(t *testing.T) {
	repo, home := newSkillFixture(t)
	cases := []struct {
		name string
		ids  []string
		want string
	}{
		{"explicit empty", []string{}, "client selection is empty"},
		{"unknown", []string{"nope"}, `unknown client "nope"`},
		{"mixed known and unknown", []string{"claude", "nope"}, `unknown client "nope"`},
		{"empty ID", []string{""}, `unknown client ""`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := Options{ClientIDs: tc.ids}
			var out bytes.Buffer
			if err := SyncWithOptions(repo, home, false, opts, &out); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("SyncWithOptions preview error = %v", err)
			}
			if err := SyncWithOptions(repo, home, true, opts, &out); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("SyncWithOptions apply error = %v", err)
			}
			result, err := SyncWithOptionsResult(repo, home, true, opts, &out)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("SyncWithOptionsResult error = %v", err)
			}
			if result.Operation != OperationApply || result.Outcome != OutcomeFailed || result.BackupID != "" {
				t.Fatalf("invalid selection result = %+v", result)
			}
			if _, err := InspectSelection(repo, home, opts); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("InspectSelection error = %v", err)
			}
			if out.Len() != 0 {
				t.Fatalf("invalid selection wrote a transcript: %q", out.String())
			}
			for _, target := range clientTargets(home) {
				if _, err := os.Stat(target.path); !os.IsNotExist(err) {
					t.Fatalf("invalid selection created %s: %v", target.path, err)
				}
			}
			if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
				t.Fatalf("invalid selection created a backup: %v", err)
			}
		})
	}

	// A valid caller slice is not rewritten by the engine.
	input := []string{"claude"}
	var out bytes.Buffer
	if _, err := SyncWithOptionsResult(repo, home, false, Options{ClientIDs: input}, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(input, []string{"claude"}) {
		t.Fatalf("engine mutated the caller's client selection: %v", input)
	}
}

func TestSubsetApplyBackupRestoreRoundTrip(t *testing.T) {
	repo, home := newSkillFixture(t)
	claudePlan := writeOldSkill(t, home, "claude", "agentic-sdd-plan", "claude old")
	agentsPlan := writeOldSkill(t, home, "agents", "agentic-sdd-plan", "agents old")
	opts := Options{ClientIDs: []string{"claude"}}

	var out bytes.Buffer
	preview, err := SyncWithOptionsResult(repo, home, false, opts, &out)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Outcome != OutcomePreview || len(preview.Planned) != 1 {
		t.Fatalf("subset preview result = %+v", preview)
	}
	if preview.Planned[0].Client != "claude" || preview.Planned[0].Action != ChangeReplace {
		t.Fatalf("subset preview planned change = %+v", preview.Planned[0])
	}
	if strings.Contains(out.String(), "agents/") {
		t.Fatalf("subset preview advertised an unselected client:\n%s", out.String())
	}

	out.Reset()
	applied, err := SyncWithOptionsResult(repo, home, true, opts, &out)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied || len(applied.Applied) != 1 {
		t.Fatalf("subset apply result = %+v", applied)
	}
	if got := read(t, filepath.Join(claudePlan, "SKILL.md")); got != "new" {
		t.Fatalf("selected client skill = %q, want new", got)
	}
	if got := read(t, filepath.Join(agentsPlan, "SKILL.md")); got != "agents old" {
		t.Fatalf("unselected client skill changed to %q", got)
	}
	for _, client := range []string{"codex", "agy"} {
		if _, err := os.Stat(filepath.Join(home, clientSubpath(t, client))); !os.IsNotExist(err) {
			t.Fatalf("subset apply touched unselected client %s: %v", client, err)
		}
	}

	m, err := readManifest(filepath.Join(applied.BackupPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := manifestTargets(home); !slices.Equal(m.Targets, want) {
		t.Fatalf("subset manifest targets = %v, want full canonical identity %v", m.Targets, want)
	}
	if len(m.Entries) != 1 || m.Entries[0].Client != "claude" || m.Entries[0].Action != actionReplaced {
		t.Fatalf("subset manifest entries = %+v, want one replaced claude entry", m.Entries)
	}

	// The supported restore path accepts the subset backup and recovers only
	// its recorded change.
	var restoreOut bytes.Buffer
	restorePreview, err := RestoreResult(repo, home, applied.BackupID, false, &restoreOut)
	if err != nil {
		t.Fatalf("subset backup rejected by restore: %v", err)
	}
	if restorePreview.Outcome != OutcomePreview || len(restorePreview.Planned) != 1 || restorePreview.Planned[0].Client != "claude" {
		t.Fatalf("subset restore preview = %+v", restorePreview)
	}
	if strings.Contains(restoreOut.String(), "agents/") {
		t.Fatalf("subset restore preview touched an unselected client:\n%s", restoreOut.String())
	}

	restoreOut.Reset()
	restored, err := RestoreResult(repo, home, applied.BackupID, true, &restoreOut)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Outcome != OutcomeApplied || len(restored.Applied) != 1 {
		t.Fatalf("subset restore result = %+v", restored)
	}
	if got := read(t, filepath.Join(claudePlan, "SKILL.md")); got != "claude old" {
		t.Fatalf("subset restore did not recover the selected client: %q", got)
	}
	if got := read(t, filepath.Join(agentsPlan, "SKILL.md")); got != "agents old" {
		t.Fatalf("subset restore changed the unselected client: %q", got)
	}
	for _, client := range []string{"codex", "agy"} {
		if _, err := os.Stat(filepath.Join(home, clientSubpath(t, client))); !os.IsNotExist(err) {
			t.Fatalf("subset restore created unselected client %s: %v", client, err)
		}
	}
	backups, err := ListBackups(repo, home)
	if err != nil || len(backups) != 2 {
		t.Fatalf("subset round-trip backups = %v, %v", backups, err)
	}
	for _, backup := range backups {
		if backup.Unusable != "" {
			t.Fatalf("subset round-trip backup %s unusable: %s", backup.ID, backup.Unusable)
		}
	}
}

func TestSubsetPreviewApplyInventoryParity(t *testing.T) {
	repo, home := newSkillFixture(t)
	opts := Options{ClientIDs: []string{"claude", "codex"}}

	inventory, err := InspectSelection(repo, home, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Entries) != 2 {
		t.Fatalf("subset inventory entries = %d, want 2: %+v", len(inventory.Entries), inventory.Entries)
	}
	var inventoryClients []string
	for _, entry := range inventory.Entries {
		inventoryClients = append(inventoryClients, entry.Client)
		if entry.Status != InventoryStatusMissing {
			t.Fatalf("fresh subset inventory entry status = %q", entry.Status)
		}
	}
	if want := []string{"codex", "claude"}; !slices.Equal(inventoryClients, want) {
		t.Fatalf("inventory clients = %v, want canonical subset order %v", inventoryClients, want)
	}

	var out bytes.Buffer
	preview, err := SyncWithOptionsResult(repo, home, false, opts, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Planned) != len(inventory.Entries) {
		t.Fatalf("preview planned %d changes, inventory has %d entries", len(preview.Planned), len(inventory.Entries))
	}
	planned := actionsByPath(preview.Planned)
	for _, entry := range inventory.Entries {
		if planned[entry.Path] != ChangeInstall {
			t.Fatalf("inventory entry %s has no matching preview install: %v", entry.Path, planned)
		}
	}

	out.Reset()
	applied, err := SyncWithOptionsResult(repo, home, true, opts, &out)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied || len(applied.Applied) != len(inventory.Entries) {
		t.Fatalf("subset apply result = %+v", applied)
	}
	appliedByPath := actionsByPath(applied.Applied)
	post, err := InspectSelection(repo, home, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range post.Entries {
		if entry.Status != InventoryStatusMatchesSource {
			t.Fatalf("post-apply inventory entry %s status = %q", entry.Path, entry.Status)
		}
		if appliedByPath[entry.Path] != ChangeInstall {
			t.Fatalf("post-apply entry %s has no matching applied change", entry.Path)
		}
	}
	for _, client := range []string{"agents", "agy"} {
		if _, err := os.Stat(filepath.Join(home, clientSubpath(t, client))); !os.IsNotExist(err) {
			t.Fatalf("subset apply touched client outside the selection %s: %v", client, err)
		}
	}
}

func TestSubsetSelectionWithOpenCodePack(t *testing.T) {
	home := t.TempDir()
	opts := Options{WithOpenCode: true, ClientIDs: []string{"agents"}}

	inventory, err := InspectSelection("", home, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range inventory.Entries {
		if entry.Client != "agents" && entry.Client != "opencode-agents" {
			t.Fatalf("subset pack inventory included unselected client %+v", entry)
		}
	}
	if len(inventory.ProfileIDs) != 14 {
		t.Fatalf("subset pack profile IDs = %d, want 14", len(inventory.ProfileIDs))
	}
	var skillEntries, profileEntries int
	for _, entry := range inventory.Entries {
		switch entry.Kind {
		case InventoryKindSkill:
			skillEntries++
		case InventoryKindProfile:
			profileEntries++
		}
	}
	if skillEntries != 19 || profileEntries != 14 {
		t.Fatalf("subset pack inventory = %d skills, %d profiles; want 19 and 14", skillEntries, profileEntries)
	}

	var out bytes.Buffer
	applied, err := SyncWithOptionsResult("", home, true, opts, &out)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied {
		t.Fatalf("subset pack apply outcome = %q", applied.Outcome)
	}
	agentsSkills := filepath.Join(home, clientSubpath(t, "agents"))
	entries, err := os.ReadDir(agentsSkills)
	if err != nil || len(entries) != 19 {
		t.Fatalf("selected client skill entries = %d, %v; want 19", len(entries), err)
	}
	for _, client := range []string{"codex", "claude", "agy"} {
		if _, err := os.Stat(filepath.Join(home, clientSubpath(t, client))); !os.IsNotExist(err) {
			t.Fatalf("subset pack apply created unselected client %s: %v", client, err)
		}
	}
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	profiles, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatal(err)
	}
	var profileFiles int
	for _, entry := range profiles {
		if strings.HasSuffix(entry.Name(), ".md") {
			profileFiles++
		}
	}
	if profileFiles != 14 {
		t.Fatalf("opencode profile files = %d, want 14 (independent of client selection)", profileFiles)
	}

	m, err := readManifest(filepath.Join(applied.BackupPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := manifestTargets(home); !slices.Equal(m.Targets, want) {
		t.Fatalf("subset pack manifest targets = %v, want %v", m.Targets, want)
	}
	if !slices.Equal(m.OpenCodeTargets, []string{agentsDir}) {
		t.Fatalf("subset pack opencode targets = %v", m.OpenCodeTargets)
	}
	if len(m.Entries) != 33 {
		t.Fatalf("subset pack manifest entries = %d, want 33", len(m.Entries))
	}

	// The subset pack backup also restores through the supported path.
	var restoreOut bytes.Buffer
	if err := Restore("", home, applied.BackupID, true, &restoreOut); err != nil {
		t.Fatalf("restore rejected a subset pack backup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentsSkills, "agentic-sdd-plan")); !os.IsNotExist(err) {
		t.Fatalf("restore left a selected skill installed: %v", err)
	}
	if remaining, err := os.ReadDir(agentsDir); err != nil {
		t.Fatal(err)
	} else if len(remaining) != 0 {
		t.Fatalf("restore left opencode profiles installed: %v", remaining)
	}
}
