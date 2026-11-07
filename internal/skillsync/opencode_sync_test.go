package skillsync

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// Feature 009 / Batch B2 / T05 — Checks 2 to 8.

// Check 2: Default core isolation & preview no-write test (AC-04, AC-05).
func TestDefaultIsolationAndPreviewNoWrite(t *testing.T) {
	home := t.TempDir()

	// 1. Default preview and apply: only the 18 core skills, never touches ~/.config/opencode.
	var previewOut bytes.Buffer
	if err := Sync("", home, false, &previewOut); err != nil {
		t.Fatalf("default preview failed: %v", err)
	}
	if strings.Contains(previewOut.String(), "agentic-sdd-orchestrate") {
		t.Fatalf("default preview advertised agentic-sdd-orchestrate:\n%s", previewOut.String())
	}
	if strings.Contains(previewOut.String(), "opencode-agents") {
		t.Fatalf("default preview advertised opencode-agents:\n%s", previewOut.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode")); !os.IsNotExist(err) {
		t.Fatalf("default preview created .config/opencode: %v", err)
	}

	var applyOut bytes.Buffer
	if err := Sync("", home, true, &applyOut); err != nil {
		t.Fatalf("default apply failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode")); !os.IsNotExist(err) {
		t.Fatalf("default apply created .config/opencode: %v", err)
	}
	for _, target := range clientTargets(home) {
		if _, err := os.Stat(filepath.Join(target.path, "agentic-sdd-orchestrate")); !os.IsNotExist(err) {
			t.Fatalf("default apply installed agentic-sdd-orchestrate in %s: %v", target.path, err)
		}
	}

	// 2. --opencode-free-first without --with-opencode returns an error.
	err := SyncWithOptions("", home, false, Options{OpenCodeFreeFirst: true}, &previewOut)
	if err == nil || !strings.Contains(err.Error(), "--opencode-free-first requires --with-opencode") {
		t.Fatalf("expected --opencode-free-first requirement error, got: %v", err)
	}

	// 3. preview --with-opencode and preview --with-opencode --opencode-free-first write nothing.
	disposableHome := t.TempDir()
	var withOpenCodePreview bytes.Buffer
	if err := SyncWithOptions("", disposableHome, false, Options{WithOpenCode: true}, &withOpenCodePreview); err != nil {
		t.Fatalf("preview with-opencode failed: %v", err)
	}
	outStr := withOpenCodePreview.String()
	if !strings.Contains(outStr, "install agents/agentic-sdd-orchestrate") {
		t.Fatalf("preview with-opencode missing orchestrate install:\n%s", outStr)
	}
	if !strings.Contains(outStr, "install opencode-agents/sdd-orchestrator.md") {
		t.Fatalf("preview with-opencode missing sdd-orchestrator.md install:\n%s", outStr)
	}
	if _, err := os.Stat(filepath.Join(disposableHome, ".config", "opencode")); !os.IsNotExist(err) {
		t.Fatalf("preview with-opencode created .config/opencode: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disposableHome, ".agentic-sdd", "backups")); !os.IsNotExist(err) {
		t.Fatalf("preview with-opencode created backups: %v", err)
	}

	var freeFirstPreview bytes.Buffer
	if err := SyncWithOptions("", disposableHome, false, Options{WithOpenCode: true, OpenCodeFreeFirst: true}, &freeFirstPreview); err != nil {
		t.Fatalf("preview with-opencode --opencode-free-first failed: %v", err)
	}
	freeOutStr := freeFirstPreview.String()
	if !strings.Contains(freeOutStr, "install opencode-agents/sdd-implement-1.md") {
		t.Fatalf("preview free-first missing sdd-implement-1.md install:\n%s", freeOutStr)
	}
	if _, err := os.Stat(filepath.Join(disposableHome, ".config", "opencode")); !os.IsNotExist(err) {
		t.Fatalf("preview free-first created .config/opencode: %v", err)
	}
}

// Check 3: Additive --with-opencode placement test (AC-06).
func TestApplyWithOpenCodeAdditive(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if err := SyncWithOptions("", home, true, Options{WithOpenCode: true}, &out); err != nil {
		t.Fatalf("apply with-opencode failed: %v", err)
	}

	// 1. Never install into ~/.config/opencode/skills.
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "skills")); !os.IsNotExist(err) {
		t.Fatalf("agentic-sdd-orchestrate or skills unexpectedly placed in ~/.config/opencode/skills: %v", err)
	}

	// 2. All 19 skills installed across 4 client skill targets.
	targets := clientTargets(home)
	for _, target := range targets {
		entries, err := os.ReadDir(target.path)
		if err != nil {
			t.Fatalf("read %s: %v", target.path, err)
		}
		if len(entries) != 19 {
			t.Fatalf("target %s has %d entries, want 19", target.name, len(entries))
		}
		orchDir := filepath.Join(target.path, "agentic-sdd-orchestrate")
		info, err := os.Stat(orchDir)
		if err != nil || !info.IsDir() {
			t.Fatalf("agentic-sdd-orchestrate missing in %s", target.name)
		}
		if info.Mode().Perm() != 0755 {
			t.Fatalf("agentic-sdd-orchestrate dir mode is %o, want 0755", info.Mode().Perm())
		}
		skillMD := filepath.Join(orchDir, "SKILL.md")
		mdInfo, err := os.Stat(skillMD)
		if err != nil || !mdInfo.Mode().IsRegular() {
			t.Fatalf("SKILL.md missing or non-regular in %s", orchDir)
		}
		if perm := mdInfo.Mode().Perm(); perm != 0644 && perm != 0600 {
			t.Fatalf("SKILL.md mode is %o, want 0644 (or 0600 under umask 077)", perm)
		}
	}

	// 3. Exactly 14 .md profiles in ~/.config/opencode/agents with mode 0644.
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	agentsInfo, err := os.Stat(agentsDir)
	if err != nil || !agentsInfo.IsDir() {
		t.Fatalf("agents directory missing or non-directory: %v", err)
	}
	agentEntries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatalf("read agents dir: %v", err)
	}
	if len(agentEntries) != 14 {
		t.Fatalf("expected 14 agent profiles, got %d", len(agentEntries))
	}
	for _, profile := range openCodeAgentProfiles {
		path := filepath.Join(agentsDir, profile+".md")
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing profile %s: %v", profile, err)
		}
		if !fi.Mode().IsRegular() {
			t.Fatalf("profile %s is not regular file", profile)
		}
		if fi.Mode().Perm() != 0644 {
			t.Fatalf("profile %s mode is %o, want 0644", profile, fi.Mode().Perm())
		}
	}
}

// Check 4: Fleet preservation & Option B MCP update test (AC-06).
func TestApplyWithOpenCodeFleetPreservationAndOptionB(t *testing.T) {
	home := t.TempDir()
	opencodeDir := filepath.Join(home, ".config", "opencode")
	agentsDir := filepath.Join(opencodeDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Seed files that MUST remain untouched:
	preserved := map[string]string{
		filepath.Join(opencodeDir, "opencode.json"):               `{"preserved":"json"}`,
		filepath.Join(opencodeDir, "AGENTS.md"):                   `# Preserved AGENTS`,
		filepath.Join(agentsDir, "sdd-spec-9r.md"):                `# Preserved Spec`,
		filepath.Join(agentsDir, "sdd-plan-9r.md"):                `# Preserved Plan`,
		filepath.Join(agentsDir, "sdd-implement-1.md"):            `# Preserved Impl 1`,
		filepath.Join(agentsDir, "sdd-implement-free-1.md"):       `# Preserved Impl Free 1`,
		filepath.Join(agentsDir, "vision.md"):                     `# Preserved Vision`,
	}
	for path, content := range preserved {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Seed the 5 Option B profiles with old content to be replaced:
	optionBProfiles := []string{
		"sdd-architecture-review-9r",
		"sdd-code-review-9r",
		"sdd-review-verification-1",
		"sdd-review-verification-2",
		"sdd-lightweight",
	}
	for _, name := range optionBProfiles {
		if err := os.WriteFile(filepath.Join(agentsDir, name+".md"), []byte("old content for "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	if err := SyncWithOptions("", home, true, Options{WithOpenCode: true}, &out); err != nil {
		t.Fatalf("apply with-opencode failed: %v", err)
	}

	// Verify preserved files are byte-for-byte untouched:
	for path, want := range preserved {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(data) != want {
			t.Fatalf("file %s was modified! got %q, want %q", path, string(data), want)
		}
	}

	// Verify Option B profiles were updated:
	for _, name := range optionBProfiles {
		data, err := os.ReadFile(filepath.Join(agentsDir, name+".md"))
		if err != nil {
			t.Fatalf("read updated %s: %v", name, err)
		}
		if string(data) == "old content for "+name {
			t.Fatalf("profile %s was not updated", name)
		}
	}

	// Verify backup contains the old Option B profile content:
	backups, err := ListBackups("", home)
	if err != nil || len(backups) == 0 {
		t.Fatalf("expected backup to be created, got %v, %v", backups, err)
	}
	backupDir := filepath.Join(home, ".agentic-sdd", "backups", backups[0].ID)
	for _, name := range optionBProfiles {
		saved := filepath.Join(backupDir, "opencode-agents", name+".md")
		data, err := os.ReadFile(saved)
		if err != nil {
			t.Fatalf("expected saved backup for %s: %v", name, err)
		}
		if string(data) != "old content for "+name {
			t.Fatalf("backup content mismatch for %s: %q", name, string(data))
		}
	}
}

// Check 5: Opt-in --opencode-free-first backup & fileDigest tamper test (AC-07, AC-10).
func TestApplyWithOpenCodeFreeFirstAndFileDigestTamper(t *testing.T) {
	home := t.TempDir()
	opencodeDir := filepath.Join(home, ".config", "opencode")
	agentsDir := filepath.Join(opencodeDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Seed the 4 Free-first profiles with original content:
	freeFirstProfiles := []string{
		"sdd-implement-1",
		"sdd-implement-2",
		"sdd-test-implement-1",
		"sdd-test-implement-2",
	}
	for _, name := range freeFirstProfiles {
		if err := os.WriteFile(filepath.Join(agentsDir, name+".md"), []byte("original "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	agentsMDPath := filepath.Join(opencodeDir, "AGENTS.md")
	if err := os.WriteFile(agentsMDPath, []byte("# Existing AGENTS"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := SyncWithOptions("", home, true, Options{WithOpenCode: true, OpenCodeFreeFirst: true}, &out); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify AGENTS.md notice is printed:
	if !strings.Contains(out.String(), `Note: ~/.config/opencode/AGENTS.md was not modified; use "agentic-sdd restore <BACKUP_ID>" to revert Free-first agent profiles.`) {
		t.Fatalf("missing AGENTS.md notice in output:\n%s", out.String())
	}

	// Verify AGENTS.md was untouched:
	agentsMDData, err := os.ReadFile(agentsMDPath)
	if err != nil || string(agentsMDData) != "# Existing AGENTS" {
		t.Fatalf("AGENTS.md was modified or missing: %v", err)
	}

	// Verify Format 3 manifest and fileDigest values:
	backups, err := ListBackups("", home)
	if err != nil || len(backups) == 0 {
		t.Fatalf("expected backup: %v", err)
	}
	b := backups[0]
	if b.Unusable != "" {
		t.Fatalf("backup marked unusable: %s", b.Unusable)
	}
	if b.Manifest.Format != 3 {
		t.Fatalf("expected manifest format 3, got %d", b.Manifest.Format)
	}
	if len(b.Manifest.OpenCodeTargets) != 1 || b.Manifest.OpenCodeTargets[0] != agentsDir {
		t.Fatalf("expected opencode_targets [%s], got %v", agentsDir, b.Manifest.OpenCodeTargets)
	}
	for _, name := range freeFirstProfiles {
		found := false
		for _, e := range b.Manifest.Entries {
			if e.Client == "opencode-agents" && e.Skill == name+".md" {
				found = true
				if e.Kind != "opencode-agent" || e.Action != actionReplaced || e.SHA256 == "" {
					t.Fatalf("invalid entry for %s: %+v", name, e)
				}
				// Verify fileDigest matches:
				backupFile := filepath.Join(home, ".agentic-sdd", "backups", b.ID, e.Backup)
				d, err := fileDigest(os.DirFS(filepath.Join(home, ".agentic-sdd", "backups", b.ID)), e.Backup)
				if err != nil || d != e.SHA256 {
					t.Fatalf("digest mismatch on %s: %s vs %s", backupFile, d, e.SHA256)
				}
			}
		}
		if !found {
			t.Fatalf("entry %s.md not found in backup", name)
		}
	}

	// Tamper test: corrupt one backed-up .md file:
	tamperFile := filepath.Join(home, ".agentic-sdd", "backups", b.ID, "opencode-agents", "sdd-implement-1.md")
	if err := os.WriteFile(tamperFile, []byte("tampered content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Verify ListBackups flags the backup as unusable:
	backupsAfterTamper, err := ListBackups("", home)
	if err != nil || len(backupsAfterTamper) == 0 {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if !strings.Contains(backupsAfterTamper[0].Unusable, "failed digest verification") {
		t.Fatalf("expected unusable reason to contain 'failed digest verification', got %q", backupsAfterTamper[0].Unusable)
	}

	// Verify Restore fails closed:
	var restoreOut bytes.Buffer
	err = Restore("", home, b.ID, false, &restoreOut)
	if err == nil || !strings.Contains(err.Error(), "failed digest verification") {
		t.Fatalf("expected Restore to fail digest verification, got %v", err)
	}
}

// Check 6: Idempotent no-op & subsequent default apply test (AC-08).
func TestApplyWithOpenCodeIdempotentAndSubsequentDefaultApply(t *testing.T) {
	home := t.TempDir()
	var out1 bytes.Buffer
	if err := SyncWithOptions("", home, true, Options{WithOpenCode: true, OpenCodeFreeFirst: true}, &out1); err != nil {
		t.Fatalf("initial apply failed: %v", err)
	}
	backups1, err := ListBackups("", home)
	if err != nil {
		t.Fatal(err)
	}
	count1 := len(backups1)

	// Re-run apply with the same options:
	var out2 bytes.Buffer
	if err := SyncWithOptions("", home, true, Options{WithOpenCode: true, OpenCodeFreeFirst: true}, &out2); err != nil {
		t.Fatalf("second apply failed: %v", err)
	}
	if !strings.Contains(out2.String(), "All skills are up to date.") {
		t.Fatalf("second apply not idempotent:\n%s", out2.String())
	}
	backups2, err := ListBackups("", home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups2) != count1 {
		t.Fatalf("new backup created on no-op apply: count %d vs %d", len(backups2), count1)
	}

	// Subsequent default apply without --with-opencode:
	var defaultOut bytes.Buffer
	if err := Sync("", home, true, &defaultOut); err != nil {
		t.Fatalf("subsequent default apply failed: %v", err)
	}
	if !strings.Contains(defaultOut.String(), "All skills are up to date.") {
		t.Fatalf("default apply reported unexpected changes:\n%s", defaultOut.String())
	}

	// Verify agentic-sdd-orchestrate and opencode agents were NOT deleted:
	for _, target := range clientTargets(home) {
		if _, err := os.Stat(filepath.Join(target.path, "agentic-sdd-orchestrate")); err != nil {
			t.Fatalf("agentic-sdd-orchestrate deleted by default apply in %s: %v", target.name, err)
		}
	}
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil || len(entries) != 18 { // 14 + 4 = 18
		t.Fatalf("opencode agents deleted by default apply: count %d", len(entries))
	}
}

// Check 7: Symlink rejection, non-.md temp staging & rollback test (AC-09).
func TestApplyWithOpenCodeSymlinkRejectionAndRollback(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Case A: symlink at target .md file
	symlinkTarget := filepath.Join(home, "external.txt")
	if err := os.WriteFile(symlinkTarget, []byte("target"), 0644); err != nil {
		t.Fatal(err)
	}
	orchestratorPath := filepath.Join(agentsDir, "sdd-orchestrator.md")
	if err := os.Symlink(symlinkTarget, orchestratorPath); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := SyncWithOptions("", home, true, Options{WithOpenCode: true}, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing special file or symlink") {
		t.Fatalf("expected symlink refusal, got %v", err)
	}
	_ = os.Remove(orchestratorPath)

	// Case B: directory at target .md file
	if err := os.MkdirAll(orchestratorPath, 0755); err != nil {
		t.Fatal(err)
	}
	err = SyncWithOptions("", home, true, Options{WithOpenCode: true}, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing special file or symlink") {
		t.Fatalf("expected directory collision refusal, got %v", err)
	}
	_ = os.RemoveAll(orchestratorPath)

	// Case C: agents directory itself is a symlink
	_ = os.RemoveAll(agentsDir)
	realAgents := filepath.Join(home, "real-agents")
	if err := os.MkdirAll(realAgents, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realAgents, agentsDir); err != nil {
		t.Fatal(err)
	}
	err = SyncWithOptions("", home, true, Options{WithOpenCode: true}, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("expected symlink agents dir refusal, got %v", err)
	}
	_ = os.Remove(agentsDir)
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Mid-apply failure rollback test with mixed changes:
	// 1. Existing agent file to be replaced and restored:
	keepAgent := filepath.Join(agentsDir, "sdd-debug-diagnose.md")
	if err := os.WriteFile(keepAgent, []byte("original debug diagnose"), 0644); err != nil {
		t.Fatal(err)
	}
	// 2. An unwritable non-empty directory collision at a later target file in changes:
	failAgentName := "sdd-visual-inspector.md"
	failPath := filepath.Join(agentsDir, failAgentName)
	if err := os.MkdirAll(failPath, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(failPath, "marker.txt")
	if err := os.WriteFile(marker, []byte("pre-existing dir"), 0644); err != nil {
		t.Fatal(err)
	}

	// Prepare changes: first an agent file to replace, then the one that fails on rename:
	agentTarget := openCodeAgentsTarget(home)
	sourceFS := fstest.MapFS{
		"debug.md": &fstest.MapFile{Data: []byte("new debug diagnose"), Mode: 0644},
		"fail.md":  &fstest.MapFile{Data: []byte("new visual"), Mode: 0644},
	}
	changes := []change{
		{target: agentTarget, skill: "sdd-debug-diagnose.md", src: "debug.md", source: sourceFS, isFile: true, old: true},
		{target: agentTarget, skill: failAgentName, src: "fail.md", source: sourceFS, isFile: true, old: false},
	}
	backupRoot := filepath.Join(home, ".agentic-sdd", "backups")
	err = installChanges(backupRoot, home, "test-source", sourceFS, []string{"sdd-debug-diagnose.md"}, []target{agentTarget}, changes, operation{kind: "apply"}, &out)
	if err == nil {
		t.Fatal("expected failure on renaming file over directory")
	}

	// Verify rollback restored the replaced file:
	content, readErr := os.ReadFile(keepAgent)
	if readErr != nil || string(content) != "original debug diagnose" {
		t.Fatalf("original agent file not restored by rollback: %q, %v", string(content), readErr)
	}

	// Verify no temporary files (.agentic-sdd-stage-* or .agentic-sdd-undo-*) remain:
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".agentic-sdd-stage-") || strings.HasPrefix(e.Name(), ".agentic-sdd-undo-") {
			t.Fatalf("leftover temporary staging/undo file: %s", e.Name())
		}
	}
}

// Check 8: Unified backups and restore round-trip test (AC-10).
func TestRestoreFormat3OpenCodeRoundTrip(t *testing.T) {
	home := t.TempDir()
	opencodeDir := filepath.Join(home, ".config", "opencode")
	agentsDir := filepath.Join(opencodeDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Seed original contents for 2 Option B profiles and 2 Free-first profiles:
	seeded := map[string]string{
		"sdd-architecture-review-9r.md": "seeded original arch review",
		"sdd-lightweight.md":            "seeded original lightweight",
		"sdd-implement-1.md":            "seeded original impl 1",
		"sdd-test-implement-1.md":       "seeded original test 1",
	}
	for name, content := range seeded {
		if err := os.WriteFile(filepath.Join(agentsDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Run apply --with-opencode --opencode-free-first
	var applyOut bytes.Buffer
	if err := SyncWithOptions("", home, true, Options{WithOpenCode: true, OpenCodeFreeFirst: true}, &applyOut); err != nil {
		t.Fatalf("initial apply failed: %v", err)
	}
	backups, err := ListBackups("", home)
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d (%v)", len(backups), err)
	}
	b1ID := backups[0].ID

	// Verify that newly installed files exist (e.g. sdd-orchestrator.md) and seeded files were replaced:
	if _, err := os.Stat(filepath.Join(agentsDir, "sdd-orchestrator.md")); err != nil {
		t.Fatalf("sdd-orchestrator.md missing after apply: %v", err)
	}
	for name, original := range seeded {
		cur, err := os.ReadFile(filepath.Join(agentsDir, name))
		if err != nil || string(cur) == original {
			t.Fatalf("file %s was not replaced", name)
		}
	}

	// 2. Run Restore (without --with-opencode!)
	var restoreOut bytes.Buffer
	if err := Restore("", home, b1ID, true, &restoreOut); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify seeded files are restored to their originals:
	for name, original := range seeded {
		cur, err := os.ReadFile(filepath.Join(agentsDir, name))
		if err != nil || string(cur) != original {
			t.Fatalf("file %s not restored to original: got %q, want %q", name, string(cur), original)
		}
	}

	// Verify freshly installed opt-in files and skills are REMOVED:
	if _, err := os.Stat(filepath.Join(agentsDir, "sdd-orchestrator.md")); !os.IsNotExist(err) {
		t.Fatalf("sdd-orchestrator.md was not removed by restore: %v", err)
	}
	for _, target := range clientTargets(home) {
		if _, err := os.Stat(filepath.Join(target.path, "agentic-sdd-orchestrate")); !os.IsNotExist(err) {
			t.Fatalf("agentic-sdd-orchestrate was not removed by restore in %s: %v", target.name, err)
		}
	}

	// 3. Round-trip: Restore itself created backup B2. Restoring B2 brings back the opt-in state!
	backupsAfterRestore, err := ListBackups("", home)
	if err != nil || len(backupsAfterRestore) != 2 {
		t.Fatalf("expected 2 backups after restore, got %d (%v)", len(backupsAfterRestore), err)
	}
	b2ID := backupsAfterRestore[0].ID // newest first

	var restore2Out bytes.Buffer
	if err := Restore("", home, b2ID, true, &restore2Out); err != nil {
		t.Fatalf("second restore failed: %v", err)
	}

	// Verify sdd-orchestrator.md and agentic-sdd-orchestrate are present again:
	if _, err := os.Stat(filepath.Join(agentsDir, "sdd-orchestrator.md")); err != nil {
		t.Fatalf("sdd-orchestrator.md missing after second restore: %v", err)
	}
	for _, target := range clientTargets(home) {
		if _, err := os.Stat(filepath.Join(target.path, "agentic-sdd-orchestrate")); err != nil {
			t.Fatalf("agentic-sdd-orchestrate missing after second restore in %s: %v", target.name, err)
		}
	}
}
