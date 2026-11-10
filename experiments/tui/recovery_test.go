package tui

// Recovery presentation tests. They cover the pure backup analysis, the
// bounded asynchronous manifest read and the honest presentation of legacy,
// empty and unusable entries. Every fixture is a temporary home/repo; nothing
// reads or writes a real user directory.

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

func createSyncBackup(t *testing.T, withOldSkill bool) (skillsync.Backup, string) {
	t.Helper()
	_, home, backup := syncBackupFixture(t, withOldSkill)
	return backup, home
}

// syncBackupFixture applies one install and returns the temporary repo, home
// and the single backup it wrote, so a test can rewrite the manifest in place.
func syncBackupFixture(t *testing.T, withOldSkill bool) (string, string, skillsync.Backup) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")

	source := filepath.Join(repo, "skills-files", "agentic-sdd-router")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("router skill content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if withOldSkill {
		old := filepath.Join(home, ".claude", "skills", "agentic-sdd-router")
		if err := os.MkdirAll(old, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old router"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := skillsync.Sync(repo, home, true, io.Discard); err != nil {
		t.Fatalf("sync error: %v", err)
	}

	backups, err := skillsync.ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	return repo, home, backups[0]
}

// legacyFixture creates one applied backup that saved old content for two
// clients, then rewrites its manifest into the format-1 shape, which recorded
// only a skill-name set and destination paths.
func legacyFixture(t *testing.T) (skillsync.Backup, string, string) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")

	source := filepath.Join(repo, "skills-files", "agentic-sdd-router")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("router skill content"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{
		filepath.Join(home, ".claude", "skills", "agentic-sdd-router"),
		filepath.Join(home, ".agents", "skills", "agentic-sdd-router"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("old router"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := skillsync.Sync(repo, home, true, io.Discard); err != nil {
		t.Fatalf("sync error: %v", err)
	}
	backups, err := skillsync.ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	id := backups[0].ID

	manifestPath := filepath.Join(repo, "backups", id, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"format", "id", "created_at", "operation", "restored_from", "tool",
		"home", "backup_root", "opencode_targets", "entries",
	} {
		delete(raw, key)
	}
	// Legacy recorded the skill-name set once, not one snapshot per client.
	raw["skills"] = []string{"agentic-sdd-router"}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}

	reloaded, err := skillsync.ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded) != 1 {
		t.Fatalf("expected 1 legacy backup, got %d", len(reloaded))
	}
	if !reloaded[0].Legacy {
		t.Fatal("fixture is not a legacy backup")
	}
	if reloaded[0].Unusable != "" {
		t.Fatalf("legacy fixture is unusable: %s", reloaded[0].Unusable)
	}
	if len(reloaded[0].Manifest.Skills) != 1 {
		t.Fatalf("legacy skill names = %v, want the one rewritten name", reloaded[0].Manifest.Skills)
	}
	return reloaded[0], repo, home
}

func TestRecovery_BreakdownFreshInstallVsReplacements(t *testing.T) {
	// A fresh install records fresh installations across all 4 client destinations and 0 saved replacement snapshots
	freshBackup, _ := createSyncBackup(t, false)

	breakdown := AnalyzeBackup(freshBackup)
	if breakdown.RecordedFreshInstalls != 4 {
		t.Errorf("RecordedFreshInstalls = %d, want 4", breakdown.RecordedFreshInstalls)
	}
	if breakdown.SavedReplacements != 0 {
		t.Errorf("SavedReplacements = %d, want 0 snapshots for fresh install", breakdown.SavedReplacements)
	}
	if len(breakdown.AffectedClients) != 4 {
		t.Errorf("AffectedClients length = %d, want 4", len(breakdown.AffectedClients))
	}
	if !breakdown.ClientsKnown {
		t.Error("ClientsKnown = false for a format-2 backup with entries")
	}

	// Mixed update backup with 1 replacement and 3 fresh installs
	mixedBackup, _ := createSyncBackup(t, true)

	mixedBreakdown := AnalyzeBackup(mixedBackup)
	if mixedBreakdown.SavedReplacements != 1 {
		t.Errorf("SavedReplacements = %d, want 1", mixedBreakdown.SavedReplacements)
	}
	if mixedBreakdown.RecordedFreshInstalls != 3 {
		t.Errorf("RecordedFreshInstalls = %d, want 3", mixedBreakdown.RecordedFreshInstalls)
	}
}

func TestRecovery_LegacyBackupHandling(t *testing.T) {
	legacyBackup, _, _ := legacyFixture(t)

	breakdown := AnalyzeBackup(legacyBackup)
	if !breakdown.IsLegacy {
		t.Fatal("expected IsLegacy to be true")
	}
	if breakdown.RecordedFreshInstalls != unknownCount {
		t.Errorf("RecordedFreshInstalls must be unknown, got %d", breakdown.RecordedFreshInstalls)
	}
	if breakdown.SavedReplacements != unknownCount {
		t.Errorf("SavedReplacements must be unknown, got %d (legacy records a skill set, not snapshots)", breakdown.SavedReplacements)
	}
	if breakdown.ClientsKnown || len(breakdown.AffectedClients) != 0 {
		t.Errorf("legacy client groups must be unknown, got %v", breakdown.AffectedClients)
	}
	if breakdown.IsEmpty {
		t.Error("legacy emptiness cannot be proven from the manifest")
	}

	// The disk-derived list summary remains authoritative and is displayed.
	text := detailsTextFor(t, legacyBackup, "")
	flat := strings.Join(strings.Fields(text), " ")
	for _, want := range []string{
		"Saved replacement snapshots: unknown (legacy format records the skill set, not saved per-client snapshots)",
		"Recorded fresh-install entries: unknown (not recorded in legacy format)",
		"Known affected client groups: unknown (legacy format records destination paths only)",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("legacy details missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "fresh install; no pre-existing files replaced") {
		t.Fatalf("legacy details invented a fresh-install fact:\n%s", text)
	}
	if strings.Contains(text, "Saved replacement snapshots: 1") {
		t.Fatalf("legacy skill-name count was reported as saved snapshots:\n%s", text)
	}
	if !strings.Contains(text, "2 replaced, 0 installed") {
		t.Fatalf("legacy details lost the authoritative recorded summary:\n%s", text)
	}
}

func TestRecovery_EmptyFormatTwoBackupIsNotEmptyWording(t *testing.T) {
	repo, home, backup := syncBackupFixture(t, false)
	// Strip every per-destination record so the backup provably covers
	// nothing. No saved trees exist for a fresh install.
	manifestPath := filepath.Join(repo, "backups", backup.ID, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["entries"] = []any{}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}

	reloaded, err := skillsync.ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded) != 1 {
		t.Fatalf("backups = %d, want 1", len(reloaded))
	}
	if reloaded[0].Unusable != "" {
		t.Fatalf("rewritten backup is unusable: %s", reloaded[0].Unusable)
	}
	backup = reloaded[0]

	breakdown := AnalyzeBackup(backup)
	if !breakdown.IsEmpty {
		t.Fatal("format-2 backup with zero entries must be empty")
	}
	if breakdown.SavedReplacements != 0 || breakdown.RecordedFreshInstalls != 0 {
		t.Fatalf("empty backup counts = %d/%d, want 0/0", breakdown.SavedReplacements, breakdown.RecordedFreshInstalls)
	}
	text := detailsTextFor(t, backup, home)
	if !strings.Contains(text, "empty backup records no destinations") {
		t.Fatalf("empty backup wording missing:\n%s", text)
	}
	if strings.Contains(text, "fresh install; no pre-existing files replaced") {
		t.Fatalf("empty backup was described as a fresh install:\n%s", text)
	}
}

func TestRecovery_EmptyLegacyEntryStaysUnknown(t *testing.T) {
	// A legacy manifest with no recorded skills or targets still cannot prove
	// how many saved trees the validated disk scan found, so nothing is zero.
	empty := skillsync.Backup{ID: "20250101T000000.000000000Z", Legacy: true}
	breakdown := AnalyzeBackup(empty)
	if breakdown.IsEmpty {
		t.Fatal("legacy emptiness cannot be proven from the manifest")
	}
	if breakdown.SavedReplacements != unknownCount || breakdown.RecordedFreshInstalls != unknownCount {
		t.Fatalf("empty legacy counts = %d/%d, want unknown/unknown", breakdown.SavedReplacements, breakdown.RecordedFreshInstalls)
	}
	text := detailsTextFor(t, empty, "")
	flat := strings.Join(strings.Fields(text), " ")
	if !strings.Contains(flat, "Saved replacement snapshots: unknown") {
		t.Fatalf("empty legacy details invented a count:\n%s", text)
	}
	if strings.Contains(text, "Empty backup") || strings.Contains(text, "fresh install; no pre-existing files replaced") {
		t.Fatalf("empty legacy details invented a state:\n%s", text)
	}
}

func TestRecovery_UnusableBackupRendersUnknownNotZero(t *testing.T) {
	unusable := skillsync.Backup{
		ID:       "20260101T000000.000000000Z",
		Unusable: "manifest.json is unreadable",
	}
	breakdown := AnalyzeBackup(unusable)
	if !breakdown.IsUnusable {
		t.Fatal("expected IsUnusable")
	}
	if breakdown.SavedReplacements != unknownCount || breakdown.RecordedFreshInstalls != unknownCount {
		t.Fatalf("unusable counts = %d/%d, want unknown/unknown", breakdown.SavedReplacements, breakdown.RecordedFreshInstalls)
	}
	if breakdown.ClientsKnown {
		t.Error("unusable entry claimed known client groups")
	}
	if breakdown.Operation == "apply" {
		t.Error("unusable entry invented an apply operation")
	}

	text := detailsTextFor(t, unusable, "")
	for _, want := range []string{
		"Recorded changed destinations: unavailable",
		"Saved replacement snapshots: unknown",
		"Known affected client groups: unknown",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("unusable details missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "fresh install; no pre-existing files replaced") {
		t.Fatalf("unusable details invented a fresh-install fact:\n%s", text)
	}
}

// TestRecovery_LogAssociationIsNeverGuessedFromFilenames reproduces the
// review's fabrication: a log whose name starts with the backup ID is not
// evidence that the log belongs to that backup.
func TestRecovery_LogAssociationIsNeverGuessedFromFilenames(t *testing.T) {
	backup, home := createSyncBackup(t, false)
	logDir := filepath.Join(home, ".agentic-sdd", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A prefix-looking name must not create an association.
	if err := os.WriteFile(filepath.Join(logDir, backup.ID+"-apply.log"), []byte("looks related\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	breakdown := AnalyzeBackup(backup)
	text := detailsTextFor(t, backup, "")
	if !strings.Contains(text, "Associated operation log: unavailable (no verified association is recorded)") {
		t.Fatalf("details claimed a log association:\n%s", text)
	}
	if strings.Contains(text, "-apply.log") {
		t.Fatalf("details printed an unverified log path:\n%s", text)
	}
	_ = breakdown
}

func TestRecovery_ManifestAndLogKeysInDetails(t *testing.T) {
	repo, home, backup := syncBackupFixture(t, false)

	cfg := TUIConfig{Width: 80, Height: 24, Repo: repo, Home: home}
	m := NewRestoreModel(cfg, RestoreOps{
		List: func(r, h string) ([]skillsync.Backup, error) {
			return []skillsync.Backup{backup}, nil
		},
		Preview: func(req appops.RestoreRequest) (appops.PreviewResult, error) {
			return appops.PreviewResult{PreviewToken: "token-1"}, nil
		},
	})

	m = restoreStep(t, m, m.Init())
	m, _ = m.openSelected()
	if m.Stage() != RestoreStageDetails {
		t.Fatalf("stage = %v, want RestoreStageDetails", m.Stage())
	}

	// 'm' must not read inside Update: it schedules a bounded read.
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd == nil {
		t.Fatal("m did not schedule the manifest read")
	}
	if !m.manifestLoading || m.manifestContent != "" {
		t.Fatalf("m read inside Update: loading=%t content=%q", m.manifestLoading, m.manifestContent)
	}
	lines := strings.Join(m.detailsLines(80), "\n")
	if !strings.Contains(lines, "Technical manifest") || !strings.Contains(lines, "Reading manifest.json") {
		t.Fatalf("expected a loading manifest section:\n%s", lines)
	}
	if strings.Contains(lines, `"format"`) {
		t.Fatalf("render showed manifest content before the read completed:\n%s", lines)
	}

	msg := cmd()
	manifestMsg, ok := msg.(RestoreManifestMsg)
	if !ok {
		t.Fatalf("manifest command returned %T", msg)
	}
	if manifestMsg.Err != nil {
		t.Fatalf("manifest read error: %v", manifestMsg.Err)
	}
	m, _ = m.Update(manifestMsg)
	lines = strings.Join(m.detailsLines(80), "\n")
	if m.manifestLoading {
		t.Fatal("manifest stayed in the loading state")
	}
	if !strings.Contains(lines, `"format"`) {
		t.Fatalf("manifest content missing after the read:\n%s", lines)
	}

	// The related-log key was removed: it is never advertised or handled.
	lines = strings.Join(m.detailsLines(80), "\n")
	if strings.Contains(lines, "l view log") {
		t.Fatalf("details still advertise an unverified log view:\n%s", lines)
	}
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"}); cmd != nil {
		t.Fatal("l still triggers a read")
	}
}

func TestRecovery_StaleManifestReadIsDiscarded(t *testing.T) {
	repo, home, backup := syncBackupFixture(t, false)
	cfg := TUIConfig{Width: 80, Height: 24, Repo: repo, Home: home}
	m := NewRestoreModel(cfg, RestoreOps{
		List: func(r, h string) ([]skillsync.Backup, error) {
			return []skillsync.Backup{backup}, nil
		},
	})
	m = restoreStep(t, m, m.Init())
	m, _ = m.openSelected()
	m, staleCmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})

	// Leaving details invalidates the request and clears the raw view.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.Stage() != RestoreStageList {
		t.Fatalf("stage = %v, want list", m.Stage())
	}
	m, _ = m.openSelected()
	m, _ = m.Update(staleCmd())
	if m.manifestContent != "" {
		t.Fatalf("stale manifest result replaced newer state: %q", m.manifestContent)
	}

	// A result for another backup identity is ignored as well.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	other := RestoreManifestMsg{Generation: m.manifestReadGen, BackupID: "20200101T000000.000000000Z", Content: "{}"}
	m, _ = m.Update(other)
	if m.manifestContent != "" {
		t.Fatalf("manifest result for another backup was accepted: %q", m.manifestContent)
	}
}

func TestReadBackupManifestContentRefusesUnsafeFiles(t *testing.T) {
	root := t.TempDir()
	id := "20260101T000000.000000000Z"
	backupDir := filepath.Join(root, id)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		root    string
		id      string
		wantErr string
	}{
		{name: "invalid id", root: root, id: "../escape", wantErr: "invalid backup id"},
		{name: "missing root", root: "", id: id, wantErr: "missing backup root"},
		{name: "missing backup dir", root: t.TempDir(), id: id, wantErr: "unable to inspect backup directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReadBackupManifestContent(tc.root, tc.id, DefaultMaxLogBytes); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}

	// A symlinked manifest is refused instead of followed.
	target := filepath.Join(root, "outside.json")
	if err := os.WriteFile(target, []byte(`{"secret":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(backupDir, "manifest.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBackupManifestContent(root, id, DefaultMaxLogBytes); err == nil || !strings.Contains(err.Error(), "non-regular manifest") {
		t.Fatalf("symlink err = %v, want a refusal", err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}

	// A directory named manifest.json is refused too.
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBackupManifestContent(root, id, DefaultMaxLogBytes); err == nil || !strings.Contains(err.Error(), "non-regular manifest") {
		t.Fatalf("directory err = %v, want a refusal", err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}

	// A regular manifest is read with the bounded limit.
	content := strings.Repeat("x", 4096)
	if err := os.WriteFile(link, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadBackupManifestContent(root, id, 128)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, strings.Repeat("x", 128)) || !strings.Contains(got, "truncated") {
		t.Fatalf("bounded read = %q", got)
	}
}

// TestReadBackupManifestContentRejectsSwapAfterValidation covers the
// Lstat/Open race the static checks alone cannot close: a symlink or
// replacement that appears between validation and open is refused, never read.
func TestReadBackupManifestContentRejectsSwapAfterValidation(t *testing.T) {
	root := t.TempDir()
	id := "20260101T000000.000000000Z"
	backupDir := filepath.Join(root, id)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(backupDir, "manifest.json")
	writeManifest := func() {
		t.Helper()
		if err := os.Remove(manifest); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if err := os.WriteFile(manifest, []byte(`{"format":3}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest()

	outside := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(outside, []byte(`{"secret":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// A symlink escaping the backup root swapped in after validation cannot be
	// followed, because the open is scoped to the validated root.
	escapeSwap := func() {
		if err := os.Remove(manifest); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, manifest); err != nil {
			t.Fatal(err)
		}
	}
	if content, err := readBackupManifestContent(root, id, DefaultMaxLogBytes, escapeSwap); err == nil {
		t.Fatalf("an escaping symlink swap was followed and returned %q", content)
	}

	// A replacement that stays inside the root passes the scope check but is
	// still rejected by file identity.
	writeManifest()
	other := filepath.Join(backupDir, "other.json")
	if err := os.WriteFile(other, []byte(`{"other":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	inRootSwap := func() {
		if err := os.Remove(manifest); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("other.json", manifest); err != nil {
			t.Fatal(err)
		}
	}
	if content, err := readBackupManifestContent(root, id, DefaultMaxLogBytes, inRootSwap); err == nil || strings.Contains(content, "other") {
		t.Fatalf("an in-root replacement was accepted: content=%q err=%v", content, err)
	}

	// A stable regular manifest still reads: the guard refuses swaps, not the
	// normal path.
	writeManifest()
	if _, err := readBackupManifestContent(root, id, DefaultMaxLogBytes, nil); err != nil {
		t.Fatalf("a stable manifest should still read: %v", err)
	}
}

func TestRecovery_DirectHomeNavigation(t *testing.T) {
	cat := Catalog{
		Home: t.TempDir(),
	}
	cfg := TUIConfig{Width: 80, Height: 24, Home: cat.Home}
	app := NewApp(cfg, cat)

	if app.ActiveRoute() != RouteHome {
		t.Fatalf("initial route = %v, want RouteHome", app.ActiveRoute())
	}

	// Pressing 'b' or 'R' from Home triggers OpenRecovery
	updated, _ := app.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	app = updated.(App)

	if app.ActiveRoute() != RouteInstall {
		t.Fatalf("route after 'b' = %v, want RouteInstall", app.ActiveRoute())
	}
	if app.installSection != installSectionBackups {
		t.Fatalf("installSection = %v, want installSectionBackups", app.installSection)
	}
}

// detailsTextFor renders one backup's details without any operation.
func detailsTextFor(t *testing.T, backup skillsync.Backup, home string) string {
	t.Helper()
	if home == "" {
		home = t.TempDir()
	}
	cfg := TUIConfig{Width: 100, Height: 40, Home: home}
	m := NewRestoreModel(cfg, RestoreOps{
		List: func(r, h string) ([]skillsync.Backup, error) {
			return []skillsync.Backup{backup}, nil
		},
	})
	m = restoreStep(t, m, m.Init())
	m, _ = m.openSelected()
	return strings.Join(m.detailsLines(100), "\n")
}
