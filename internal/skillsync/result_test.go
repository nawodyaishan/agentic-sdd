package skillsync

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// newSkillFixture creates an isolated repo with one portable skill source and
// an untouched home. Fixtures stay deterministic: no test reads the real home
// or the network.
func newSkillFixture(t *testing.T) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	home = filepath.Join(root, "home")
	writeTestSkillSource(t, repo, "agentic-sdd-plan", "new")
	return repo, home
}

func writeTestSkillSource(t *testing.T, repo, skill, content string) {
	t.Helper()
	dir := filepath.Join(repo, "skills-files", skill)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// writeOldSkill installs a pre-existing skill tree under one client so a
// plan must replace and back it up.
func writeOldSkill(t *testing.T, home, client, skill, content string) string {
	t.Helper()
	dir := filepath.Join(home, clientSubpath(t, client), skill)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func manifestTargets(home string) []string {
	var targets []string
	for _, t := range clientTargets(home) {
		targets = append(targets, t.path)
	}
	return targets
}

func actionsByPath(changes []ResultChange) map[string]ChangeAction {
	actions := make(map[string]ChangeAction, len(changes))
	for _, c := range changes {
		actions[c.Path] = c.Action
	}
	return actions
}

func wantAllActions(t *testing.T, home string, actions map[string]ChangeAction) map[string]ChangeAction {
	t.Helper()
	want := make(map[string]ChangeAction, len(actions))
	for client, action := range actions {
		want[filepath.Join(home, clientSubpath(t, client), "agentic-sdd-plan")] = action
	}
	return want
}

// comparePlanToSummary proves the structured plan reports the same items the
// transcript summarizes, without parsing the transcript for the result.
func comparePlanToSummary(t *testing.T, transcript string, planned []ResultChange) {
	t.Helper()
	want := map[ChangeAction]int{}
	for _, c := range planned {
		want[c.Action]++
	}
	got := SummarizePlan(transcript)
	if got.Install != want[ChangeInstall] ||
		got.Replace != want[ChangeReplace] ||
		got.Restore != want[ChangeRestore] ||
		got.Remove != want[ChangeRemove] ||
		got.UpToDate != want[ChangeUpToDate] {
		t.Fatalf("plan summary %+v does not match structured plan %v", got, want)
	}
	if len(got.Unclassified) != 0 {
		t.Fatalf("transcript has unclassified lines: %v", got.Unclassified)
	}
}

func TestOperationResultZeroValueIsNotSuccess(t *testing.T) {
	var result OperationResult
	for _, success := range []Outcome{OutcomePreview, OutcomeNoChange, OutcomeApplied} {
		if result.Outcome == success {
			t.Fatalf("zero OperationResult reads as %q", success)
		}
	}
	if result.BackupID != "" || result.BackupPath != "" || len(result.Applied) != 0 {
		t.Fatalf("zero OperationResult claims completed work: %+v", result)
	}
}

func TestSyncResultReplacementAndManifestIdentity(t *testing.T) {
	repo, home := newSkillFixture(t)
	claudePlan := writeOldSkill(t, home, "claude", "agentic-sdd-plan", "old")

	var out bytes.Buffer
	result, err := SyncWithOptionsResult(repo, home, true, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation != OperationApply || result.Outcome != OutcomeApplied {
		t.Fatalf("operation/outcome = %q/%q, want apply/applied", result.Operation, result.Outcome)
	}
	if err := ValidateBackupID(result.BackupID); err != nil {
		t.Fatalf("result backup id %q: %v", result.BackupID, err)
	}
	if want := filepath.Join(repo, "backups", result.BackupID); result.BackupPath != want {
		t.Fatalf("result backup path = %q, want %q", result.BackupPath, want)
	}
	if info, err := os.Stat(result.BackupPath); err != nil || !info.IsDir() {
		t.Fatalf("reported backup missing: %v", err)
	}
	if len(result.Planned) != 4 {
		t.Fatalf("planned changes = %d, want 4: %+v", len(result.Planned), result.Planned)
	}
	wantActions := wantAllActions(t, home, map[string]ChangeAction{
		"agents": ChangeInstall,
		"codex":  ChangeInstall,
		"claude": ChangeReplace,
		"agy":    ChangeInstall,
	})
	if got := actionsByPath(result.Planned); !reflect.DeepEqual(got, wantActions) {
		t.Fatalf("planned actions = %v, want %v", got, wantActions)
	}
	if len(result.Applied) != 4 {
		t.Fatalf("applied changes = %d, want 4: %+v", len(result.Applied), result.Applied)
	}
	for _, c := range result.Applied {
		if c.Client == "claude" {
			if c.Action != ChangeReplace || c.Backup == "" || len(c.SHA256) != 64 {
				t.Fatalf("claude applied change lacks saved previous content: %+v", c)
			}
			continue
		}
		if c.Action != ChangeInstall || c.Backup != "" || c.SHA256 != "" {
			t.Fatalf("%s applied change = %+v, want an unsaved install", c.Client, c)
		}
	}
	if got := read(t, filepath.Join(claudePlan, "SKILL.md")); got != "new" {
		t.Fatalf("claude plan = %q, want new", got)
	}

	m, err := readManifest(filepath.Join(result.BackupPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := manifestTargets(home); !slices.Equal(m.Targets, want) {
		t.Fatalf("manifest targets = %v, want full canonical identity %v", m.Targets, want)
	}
	if len(m.Entries) != 4 {
		t.Fatalf("manifest entries = %d, want 4", len(m.Entries))
	}
	if m.Operation != "apply" || !slices.Equal(m.Skills, []string{"agentic-sdd-plan"}) {
		t.Fatalf("manifest operation/skills = %q/%v", m.Operation, m.Skills)
	}
	comparePlanToSummary(t, out.String(), result.Planned)

	// A repeated apply is a definite no-change outcome and creates no backup.
	out.Reset()
	again, err := SyncWithOptionsResult(repo, home, true, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if again.Outcome != OutcomeNoChange || again.BackupID != "" || again.BackupPath != "" || len(again.Applied) != 0 {
		t.Fatalf("no-change result = %+v", again)
	}
	for _, c := range again.Planned {
		if c.Action != ChangeUpToDate {
			t.Fatalf("no-change planned %s/%s action = %q", c.Client, c.ID, c.Action)
		}
	}
	comparePlanToSummary(t, out.String(), again.Planned)
	backups, err := ListBackups(repo, home)
	if err != nil || len(backups) != 1 {
		t.Fatalf("no-change backups = %v, %v", backups, err)
	}
}

func TestSyncResultPreviewIsReadOnly(t *testing.T) {
	repo, home := newSkillFixture(t)
	claudePlan := writeOldSkill(t, home, "claude", "agentic-sdd-plan", "old")

	var out bytes.Buffer
	result, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation != OperationApply || result.Outcome != OutcomePreview {
		t.Fatalf("operation/outcome = %q/%q, want apply/preview", result.Operation, result.Outcome)
	}
	if result.BackupID != "" || result.BackupPath != "" || len(result.Applied) != 0 {
		t.Fatalf("preview claims completed work: %+v", result)
	}
	wantActions := wantAllActions(t, home, map[string]ChangeAction{
		"agents": ChangeInstall,
		"codex":  ChangeInstall,
		"claude": ChangeReplace,
		"agy":    ChangeInstall,
	})
	if got := actionsByPath(result.Planned); !reflect.DeepEqual(got, wantActions) {
		t.Fatalf("planned actions = %v, want %v", got, wantActions)
	}
	comparePlanToSummary(t, out.String(), result.Planned)
	if got := read(t, filepath.Join(claudePlan, "SKILL.md")); got != "old" {
		t.Fatalf("preview changed claude skill: %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, clientSubpath(t, "agents"))); !os.IsNotExist(err) {
		t.Fatalf("preview created an unselected install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("preview created a backup: %v", err)
	}
}

func TestSyncResultFailureBeforeBackup(t *testing.T) {
	repo, home := newSkillFixture(t)
	outside := t.TempDir()
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "backups")); err != nil {
		t.Fatal(err)
	}

	var wrapperOut, resultOut bytes.Buffer
	wrapperErr := SyncWithOptions(repo, home, true, Options{}, &wrapperOut)
	result, resultErr := SyncWithOptionsResult(repo, home, true, Options{}, &resultOut)
	if wrapperErr == nil || resultErr == nil {
		t.Fatalf("expected backup path refusal, got wrapper=%v result=%v", wrapperErr, resultErr)
	}
	if wrapperErr.Error() != resultErr.Error() {
		t.Fatalf("wrapper error %q != result error %q", wrapperErr, resultErr)
	}
	if wrapperOut.String() != resultOut.String() {
		t.Fatalf("wrapper transcript %q != result transcript %q", wrapperOut.String(), resultOut.String())
	}
	if !strings.Contains(resultErr.Error(), "refusing non-directory backup path") {
		t.Fatalf("unexpected error: %v", resultErr)
	}
	if result.Operation != OperationApply || result.Outcome != OutcomeFailed {
		t.Fatalf("operation/outcome = %q/%q, want apply/failed", result.Operation, result.Outcome)
	}
	if result.BackupID != "" || result.BackupPath != "" {
		t.Fatalf("failed-before-backup result claims a backup: %+v", result)
	}
	if result.Rollback.Attempted {
		t.Fatalf("failed-before-backup result claims rollback: %+v", result.Rollback)
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed apply wrote outside the repository: %v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(home, clientSubpath(t, "agents"), "agentic-sdd-plan")); !os.IsNotExist(err) {
		t.Fatalf("failed apply installed a target: %v", err)
	}
}

func TestSyncResultFailureAfterBackup(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based staging failure cannot be provoked as root")
	}
	repo, home := newSkillFixture(t)
	// A readable SKILL.md keeps discovery and planning healthy; an unreadable
	// nested file makes the staging copy fail only after the backup exists.
	sourceDir := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	sourceFile := filepath.Join(sourceDir, "locked.md")
	if err := os.WriteFile(sourceFile, []byte("locked"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sourceFile, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sourceFile, 0644) })

	var out bytes.Buffer
	result, err := SyncWithOptionsResult(repo, home, true, Options{}, &out)
	if err == nil {
		t.Fatal("expected staging to fail on an unreadable source")
	}
	if result.Operation != OperationApply || result.Outcome != OutcomeFailed {
		t.Fatalf("operation/outcome = %q/%q, want apply/failed", result.Operation, result.Outcome)
	}
	if result.BackupID == "" || result.BackupPath == "" {
		t.Fatalf("failure after a complete backup did not report it: %+v (error: %v)", result, err)
	}
	if err := ValidateBackupID(result.BackupID); err != nil {
		t.Fatalf("backup id %q: %v", result.BackupID, err)
	}
	if _, err := readManifest(filepath.Join(result.BackupPath, "manifest.json")); err != nil {
		t.Fatalf("reported backup is not complete: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Fatalf("failed result claims applied changes: %+v", result.Applied)
	}
	if result.Rollback.Attempted {
		t.Fatalf("staging failure needs no rollback: %+v", result.Rollback)
	}
	if _, err := os.Stat(filepath.Join(home, clientSubpath(t, "agents"), "agentic-sdd-plan")); !os.IsNotExist(err) {
		t.Fatalf("staging failure installed a target: %v", err)
	}
	err = filepath.WalkDir(home, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.HasPrefix(entry.Name(), ".agentic-sdd-stage-") {
			t.Fatalf("leftover staging entry %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInstallChangesReportsRollbackAfterBackup(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	targetPath := filepath.Join(home, clientSubpath(t, "claude"))
	removeName, failName := "agentic-sdd-remove-me", "agentic-sdd-fail"
	removeDir := filepath.Join(targetPath, removeName)
	if err := os.MkdirAll(removeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(removeDir, "SKILL.md"), []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	// A non-empty pre-existing directory makes the final install rename fail
	// after the remove change above has already committed.
	failDir := filepath.Join(targetPath, failName)
	if err := os.MkdirAll(failDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(failDir, "marker.md"), []byte("pre-existing"), 0644); err != nil {
		t.Fatal(err)
	}
	source := fstest.MapFS{failName + "/SKILL.md": &fstest.MapFile{Data: []byte("new"), Mode: 0644}}
	tgt := target{name: "claude", path: targetPath}
	changes := []change{
		{target: tgt, skill: removeName, old: true, remove: true, action: ChangeRemove},
		{target: tgt, skill: failName, src: failName, action: ChangeInstall},
	}
	backupRoot := filepath.Join(root, "backups")
	var out bytes.Buffer
	written, err := installChanges(backupRoot, home, "test-source", source, []string{failName}, changes, operation{kind: "apply"}, &out)
	if err == nil {
		t.Fatal("expected the forced rename failure")
	}
	if err := ValidateBackupID(written.backupID); err != nil {
		t.Fatalf("rollback outcome backup id %q: %v", written.backupID, err)
	}
	if want := filepath.Join(backupRoot, written.backupID); written.backupPath != want {
		t.Fatalf("rollback outcome backup path = %q, want %q", written.backupPath, want)
	}
	if !written.rollback.Attempted || !written.rollback.Complete || written.rollback.Error != "" {
		t.Fatalf("rollback info = %+v, want a complete attempt", written.rollback)
	}
	if len(written.applied) != 0 {
		t.Fatalf("failed write reports applied changes: %+v", written.applied)
	}
	if got := read(t, filepath.Join(removeDir, "SKILL.md")); got != "keep me" {
		t.Fatalf("rollback did not restore the removed skill: %q", got)
	}
	m, err := readManifest(filepath.Join(written.backupPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := manifestTargets(home); !slices.Equal(m.Targets, want) {
		t.Fatalf("partial-failure backup targets = %v, want %v", m.Targets, want)
	}
}

func TestRestoreResultStructuredOutcomes(t *testing.T) {
	repo, home, _, id := setupBaseBackup(t)

	var out bytes.Buffer
	preview, err := RestoreResult(repo, home, id, false, &out)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Operation != OperationRestore || preview.Outcome != OutcomePreview {
		t.Fatalf("preview operation/outcome = %q/%q", preview.Operation, preview.Outcome)
	}
	if preview.BackupID != "" || len(preview.Applied) != 0 {
		t.Fatalf("restore preview claims completed work: %+v", preview)
	}
	wantPreview := wantAllActions(t, home, map[string]ChangeAction{
		"claude": ChangeReplace,
		"agents": ChangeRemove,
		"codex":  ChangeRemove,
		"agy":    ChangeRemove,
	})
	if got := actionsByPath(preview.Planned); !reflect.DeepEqual(got, wantPreview) {
		t.Fatalf("restore preview actions = %v, want %v", got, wantPreview)
	}
	comparePlanToSummary(t, out.String(), preview.Planned)
	if got := read(t, filepath.Join(home, clientSubpath(t, "claude"), "agentic-sdd-plan", "SKILL.md")); got != "new" {
		t.Fatalf("restore preview changed claude skill: %q", got)
	}

	out.Reset()
	applied, err := RestoreResult(repo, home, id, true, &out)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied {
		t.Fatalf("restore apply outcome = %q, want applied", applied.Outcome)
	}
	if err := ValidateBackupID(applied.BackupID); err != nil {
		t.Fatalf("restore backup id %q: %v", applied.BackupID, err)
	}
	if len(applied.Applied) != 4 {
		t.Fatalf("restore applied changes = %d, want 4", len(applied.Applied))
	}
	for _, c := range applied.Applied {
		if c.Backup == "" || len(c.SHA256) != 64 {
			t.Fatalf("restore applied change lacks saved current content: %+v", c)
		}
	}
	m, err := readManifest(filepath.Join(applied.BackupPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Operation != "restore" || m.RestoredFrom != id {
		t.Fatalf("pre-restore backup operation/restored_from = %q/%q", m.Operation, m.RestoredFrom)
	}
	if got := read(t, filepath.Join(home, clientSubpath(t, "claude"), "agentic-sdd-plan", "SKILL.md")); got != "old" {
		t.Fatalf("restore did not put back the old skill: %q", got)
	}

	out.Reset()
	noChange, err := RestoreResult(repo, home, id, true, &out)
	if err != nil {
		t.Fatal(err)
	}
	if noChange.Outcome != OutcomeNoChange || noChange.BackupID != "" || len(noChange.Applied) != 0 {
		t.Fatalf("repeated restore result = %+v", noChange)
	}
	for _, c := range noChange.Planned {
		if c.Action != ChangeUpToDate {
			t.Fatalf("repeated restore planned %s/%s action = %q", c.Client, c.ID, c.Action)
		}
	}
	comparePlanToSummary(t, out.String(), noChange.Planned)
}

var backupStampPattern = regexp.MustCompile(`[0-9]{8}T[0-9]{6}\.[0-9]{9}Z`)

func normalizeBackupStamps(transcript string) string {
	return backupStampPattern.ReplaceAllString(transcript, "<backup-id>")
}

func TestSyncWrapperResultPreviewEquivalence(t *testing.T) {
	repo, home := newSkillFixture(t)
	writeOldSkill(t, home, "claude", "agentic-sdd-plan", "old")

	var wrapperOut, resultOut bytes.Buffer
	wrapperErr := SyncWithOptions(repo, home, false, Options{}, &wrapperOut)
	result, resultErr := SyncWithOptionsResult(repo, home, false, Options{}, &resultOut)
	if wrapperErr != nil || resultErr != nil {
		t.Fatalf("preview errors: wrapper=%v result=%v", wrapperErr, resultErr)
	}
	if wrapperOut.String() != resultOut.String() {
		t.Fatalf("preview bytes differ:\nwrapper: %q\nresult:  %q", wrapperOut.String(), resultOut.String())
	}
	if result.Outcome != OutcomePreview {
		t.Fatalf("preview outcome = %q", result.Outcome)
	}
}

func TestSyncWrapperResultApplyEquivalence(t *testing.T) {
	repo, homeA := newSkillFixture(t)
	homeB := filepath.Join(filepath.Dir(homeA), "home-b")
	writeOldSkill(t, homeA, "claude", "agentic-sdd-plan", "old")
	writeOldSkill(t, homeB, "claude", "agentic-sdd-plan", "old")

	var wrapperOut, resultOut bytes.Buffer
	wrapperErr := SyncWithOptions(repo, homeA, true, Options{}, &wrapperOut)
	result, resultErr := SyncWithOptionsResult(repo, homeB, true, Options{}, &resultOut)
	if wrapperErr != nil || resultErr != nil {
		t.Fatalf("apply errors: wrapper=%v result=%v", wrapperErr, resultErr)
	}
	if normalizeBackupStamps(wrapperOut.String()) != normalizeBackupStamps(resultOut.String()) {
		t.Fatalf("apply bytes differ:\nwrapper: %q\nresult:  %q", wrapperOut.String(), resultOut.String())
	}
	if result.Outcome != OutcomeApplied || result.BackupID == "" {
		t.Fatalf("apply result = %+v", result)
	}
	for _, home := range []string{homeA, homeB} {
		if got := read(t, filepath.Join(home, clientSubpath(t, "claude"), "agentic-sdd-plan", "SKILL.md")); got != "new" {
			t.Fatalf("%s claude plan = %q, want new", home, got)
		}
	}
}

func TestSyncWrapperResultNoChangeEquivalence(t *testing.T) {
	repo, home := newSkillFixture(t)
	var applyOut bytes.Buffer
	if err := SyncWithOptions(repo, home, true, Options{}, &applyOut); err != nil {
		t.Fatal(err)
	}

	var wrapperOut, resultOut bytes.Buffer
	wrapperErr := SyncWithOptions(repo, home, true, Options{}, &wrapperOut)
	result, resultErr := SyncWithOptionsResult(repo, home, true, Options{}, &resultOut)
	if wrapperErr != nil || resultErr != nil {
		t.Fatalf("no-change errors: wrapper=%v result=%v", wrapperErr, resultErr)
	}
	if wrapperOut.String() != resultOut.String() {
		t.Fatalf("no-change bytes differ:\nwrapper: %q\nresult:  %q", wrapperOut.String(), resultOut.String())
	}
	if result.Outcome != OutcomeNoChange {
		t.Fatalf("no-change outcome = %q", result.Outcome)
	}
}

func TestRestoreWrapperResultPreviewEquivalence(t *testing.T) {
	repo, home, _, id := setupBaseBackup(t)
	var wrapperOut, resultOut bytes.Buffer
	wrapperErr := Restore(repo, home, id, false, &wrapperOut)
	result, resultErr := RestoreResult(repo, home, id, false, &resultOut)
	if wrapperErr != nil || resultErr != nil {
		t.Fatalf("restore preview errors: wrapper=%v result=%v", wrapperErr, resultErr)
	}
	if wrapperOut.String() != resultOut.String() {
		t.Fatalf("restore preview bytes differ:\nwrapper: %q\nresult:  %q", wrapperOut.String(), resultOut.String())
	}
	if result.Outcome != OutcomePreview || result.Operation != OperationRestore {
		t.Fatalf("restore preview result = %+v", result)
	}
}

func TestRestoreWrapperResultApplyEquivalence(t *testing.T) {
	repo, home, _, id := setupBaseBackup(t)
	snapshot := filepath.Join(t.TempDir(), "home-snapshot")
	if err := copyTree(home, snapshot); err != nil {
		t.Fatal(err)
	}

	var wrapperOut bytes.Buffer
	if err := Restore(repo, home, id, true, &wrapperOut); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(home); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(snapshot, home); err != nil {
		t.Fatal(err)
	}
	var resultOut bytes.Buffer
	result, err := RestoreResult(repo, home, id, true, &resultOut)
	if err != nil {
		t.Fatal(err)
	}
	if normalizeBackupStamps(wrapperOut.String()) != normalizeBackupStamps(resultOut.String()) {
		t.Fatalf("restore apply bytes differ:\nwrapper: %q\nresult:  %q", wrapperOut.String(), resultOut.String())
	}
	if result.Outcome != OutcomeApplied {
		t.Fatalf("restore result outcome = %q", result.Outcome)
	}
}

func TestRestoreErrorEquivalence(t *testing.T) {
	repo, home, _, _ := setupBaseBackup(t)
	var wrapperOut, resultOut bytes.Buffer
	wrapperErr := Restore(repo, home, "not-an-id", false, &wrapperOut)
	result, resultErr := RestoreResult(repo, home, "not-an-id", false, &resultOut)
	if wrapperErr == nil || resultErr == nil {
		t.Fatalf("expected invalid id errors, got wrapper=%v result=%v", wrapperErr, resultErr)
	}
	if wrapperErr.Error() != resultErr.Error() {
		t.Fatalf("restore errors differ: %q vs %q", wrapperErr, resultErr)
	}
	if wrapperOut.String() != resultOut.String() {
		t.Fatalf("restore error bytes differ: %q vs %q", wrapperOut.String(), resultOut.String())
	}
	if result.Operation != OperationRestore || result.Outcome != OutcomeFailed || result.BackupID != "" {
		t.Fatalf("restore error result = %+v", result)
	}
}
