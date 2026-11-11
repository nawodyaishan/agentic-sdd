package fleet

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"agentic-sdd/internal/skillsync"
)

func TestFleetRestore(t *testing.T) {
	t.Run("restores managed files, removes created files and preserves user files", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(),
			`"sdd-lightweight":{"source":"agent:sdd-lightweight","model":"9router/ag/gemini-3.8-flash-high"},`+
				`"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"}`)
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		replaced := filepath.Join(targetRoot, "sdd-lightweight.md")
		if err := os.WriteFile(replaced, []byte("original unmanaged content\n"), 0644); err != nil {
			t.Fatal(err)
		}
		unrelated := filepath.Join(targetRoot, "user-agent.md")
		if err := os.WriteFile(unrelated, []byte("keep me\n"), 0644); err != nil {
			t.Fatal(err)
		}

		plan := buildPlanFor(t, configPath, home, true)
		result := applyPlan(t, plan, home)
		if result.Updated != 1 || result.Created != 1 {
			t.Fatalf("unexpected apply result: %+v", result)
		}
		created := filepath.Join(targetRoot, "sdd-implement-1.md")
		if _, err := os.Stat(created); err != nil {
			t.Fatalf("expected the created profile: %v", err)
		}

		var out bytes.Buffer
		restored, err := RestoreBackup(home, result.BackupID, &out)
		if err != nil {
			t.Fatalf("RestoreBackup: %v\noutput: %s", err, out.String())
		}
		if restored.Restored != 1 || restored.Removed != 1 || restored.BackupID == "" {
			t.Fatalf("unexpected restore result: %+v", restored)
		}
		if got := readString(t, replaced); got != "original unmanaged content\n" {
			t.Fatalf("replaced file was not restored: %q", got)
		}
		if _, err := os.Lstat(created); !os.IsNotExist(err) {
			t.Fatalf("created profile was not removed: %v", err)
		}
		if got := readString(t, unrelated); got != "keep me\n" {
			t.Fatalf("unrelated file was modified: %q", got)
		}

		sidecar, err := ReadManifestFile(filepath.Join(restored.BackupPath, ManifestFileName))
		if err != nil || sidecar == nil {
			t.Fatalf("read restore sidecar: %v", err)
		}
		if sidecar.Operation != "restore" || sidecar.RestoredFrom != result.BackupID || sidecar.State != ManifestStateApplied {
			t.Fatalf("unexpected restore sidecar: %+v", sidecar)
		}
		if len(sidecar.Files) != 2 {
			t.Fatalf("expected two restore records, got %+v", sidecar.Files)
		}
		for _, rec := range sidecar.Files {
			switch filepath.Base(rec.Target) {
			case "sdd-lightweight.md":
				if rec.Provenance != ProvenanceUnknown || rec.DesiredDigest == "" {
					t.Fatalf("restored content must not be fabricated as managed/clean: %+v", rec)
				}
			case "sdd-implement-1.md":
				if rec.Provenance != ProvenanceRemoved {
					t.Fatalf("created profile reversal must be a removal: %+v", rec)
				}
			}
		}

		live, err := ReadLiveManifest(home)
		if err != nil || live == nil {
			t.Fatalf("read live manifest: %v", err)
		}
		if rec, ok := live.RecordFor(replaced); ok && rec.IsManaged() {
			t.Fatalf("restored unmanaged content was claimed as managed: %+v", rec)
		}
		if _, ok := live.RecordFor(created); ok {
			t.Fatalf("removed created profile is still live-managed: %+v", live)
		}
	})

	t.Run("refuses a modified target without touching it", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, "sdd-lightweight.md")
		if err := os.WriteFile(target, []byte("before\n"), 0644); err != nil {
			t.Fatal(err)
		}
		result := applyPlan(t, buildPlanFor(t, configPath, home, true), home)
		backupsBefore := len(backupDirs(t, home))

		if err := os.WriteFile(target, []byte("modified after apply\n"), 0644); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if _, err := RestoreBackup(home, result.BackupID, &out); !errors.Is(err, ErrUnsafeTarget) {
			t.Fatalf("expected ErrUnsafeTarget, got %v", err)
		}
		if got := readString(t, target); got != "modified after apply\n" {
			t.Fatalf("modified target was overwritten: %q", got)
		}
		if got := len(backupDirs(t, home)); got != backupsBefore {
			t.Fatalf("refused restore created a backup: %d -> %d", backupsBefore, got)
		}
	})

	t.Run("refuses a legacy backup with no fleet manifest", func(t *testing.T) {
		home := newFleetHome(t)
		root := skillsync.BackupRootFor("", home)
		id := "20260101T000000.000000000Z"
		dir := filepath.Join(root, id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		legacyManifest := `{"created":"` + id + `","source":"test","targets":[],"skills":[],"format":3,"id":"` + id + `"}` + "\n"
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(legacyManifest), 0600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if _, err := RestoreBackup(home, id, &out); !errors.Is(err, ErrLegacyProvenance) {
			t.Fatalf("expected ErrLegacyProvenance, got %v", err)
		}
	})

	t.Run("restores a previous managed removal", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		beforeRemoval := readString(t, target)

		disabledConfig := writeFleetConfig(t, t.TempDir(), `"sdd-lightweight":{"source":"agent:sdd-lightweight","enabled":false}`)
		removal := applyPlan(t, buildPlanFor(t, disabledConfig, home, false), home)
		if removal.Removed != 1 {
			t.Fatalf("expected a removal, got %+v", removal)
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatalf("profile was not removed: %v", err)
		}

		var out bytes.Buffer
		restored, err := RestoreBackup(home, removal.BackupID, &out)
		if err != nil {
			t.Fatalf("RestoreBackup removal: %v\noutput: %s", err, out.String())
		}
		if restored.Restored != 1 {
			t.Fatalf("expected the removed profile to be restored, got %+v", restored)
		}
		if got := readString(t, target); got != beforeRemoval {
			t.Fatalf("removed profile content mismatch: %q", got)
		}
	})

	t.Run("restore rejects malformed backup ids before touching disk", func(t *testing.T) {
		home := newFleetHome(t)
		var out bytes.Buffer
		if _, err := RestoreBackup(home, "../escape", &out); err == nil {
			t.Fatal("expected a malformed backup id to be refused")
		}
		if _, err := os.Lstat(filepath.Join(home, ".agentic-sdd", "backups", "escape")); !os.IsNotExist(err) {
			t.Fatalf("malformed backup id created or touched a path: %v", err)
		}
	})

	t.Run("fleet-backup listing and legacy skillsync coexistence", func(t *testing.T) {
		home := newFleetHome(t)

		// 1. Initially empty
		entries, err := ListFleetBackups(home)
		if err != nil {
			t.Fatalf("ListFleetBackups empty: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected 0 backups, got %d", len(entries))
		}

		// 2. Add an unmanaged profile, apply a fleet plan (creating 1st fleet backup)
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, "sdd-lightweight.md")
		if err := os.WriteFile(target, []byte("initial\n"), 0644); err != nil {
			t.Fatal(err)
		}

		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		result1 := applyPlan(t, buildPlanFor(t, configPath, home, true), home)

		entries, err = ListFleetBackups(home)
		if err != nil {
			t.Fatalf("ListFleetBackups after apply: %v", err)
		}
		if len(entries) != 1 || entries[0].ID != result1.BackupID {
			t.Fatalf("expected 1 fleet backup %s, got %+v", result1.BackupID, entries)
		}
		if entries[0].Manifest.Operation != "apply" || entries[0].Manifest.State != ManifestStateApplied {
			t.Fatalf("unexpected manifest in listed backup: %+v", entries[0].Manifest)
		}

		// 3. Perform a restore (creating 2nd backup: fleet-restore)
		var out bytes.Buffer
		result2, err := RestoreBackup(home, result1.BackupID, &out)
		if err != nil {
			t.Fatalf("RestoreBackup: %v", err)
		}

		entries, err = ListFleetBackups(home)
		if err != nil {
			t.Fatalf("ListFleetBackups after restore: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 fleet backups, got %d", len(entries))
		}
		// Sorted newest first
		if entries[0].ID != result2.BackupID || entries[1].ID != result1.BackupID {
			t.Fatalf("expected backups [%s, %s], got [%s, %s]", result2.BackupID, result1.BackupID, entries[0].ID, entries[1].ID)
		}
		if entries[0].Manifest.Operation != "restore" {
			t.Fatalf("expected newest backup operation=restore, got %s", entries[0].Manifest.Operation)
		}

		// 4. Inject a legacy backup without fleet-manifest.json
		root := skillsync.BackupRootFor("", home)
		legacyID := "20260101T000000.000000000Z"
		legacyDir := filepath.Join(root, legacyID)
		if err := os.MkdirAll(legacyDir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(legacyDir, "manifest.json"), []byte(`{"format":3,"id":"`+legacyID+`"}`), 0600); err != nil {
			t.Fatal(err)
		}

		// ListFleetBackups must ignore legacy non-fleet backups safely
		entries, err = ListFleetBackups(home)
		if err != nil {
			t.Fatalf("ListFleetBackups with legacy: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("expected exactly 2 fleet backups, got %d", len(entries))
		}

		// 5. Test skillsync.ListBackups coexistence: must list fleet backups without Unusable errors
		syncBackups, err := skillsync.ListBackups("", home)
		if err != nil {
			t.Fatalf("skillsync.ListBackups error: %v", err)
		}
		if len(syncBackups) < 2 {
			t.Fatalf("expected at least 2 backups in skillsync.ListBackups, got %d", len(syncBackups))
		}
		for _, b := range syncBackups {
			if b.ID == result1.BackupID || b.ID == result2.BackupID {
				if b.Unusable != "" {
					t.Fatalf("skillsync marked fleet backup %s as unusable: %s", b.ID, b.Unusable)
				}
			}
		}
	})
}
