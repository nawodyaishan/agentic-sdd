package fleet

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"agentic-sdd/internal/skillsync"
)

func TestFleetRestorePreview(t *testing.T) {
	home := newFleetHome(t)
	targetRoot := TargetRootForHome(home)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}
	replaced := filepath.Join(targetRoot, "sdd-lightweight.md")
	if err := os.WriteFile(replaced, []byte("original unmanaged content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high")+","+
		`"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"opencode-go/deepseek-v4.1-flash"}`)
	result := applyPlan(t, buildPlanFor(t, configPath, home, true), home)
	if result.Updated != 1 || result.Created != 1 {
		t.Fatalf("unexpected apply result: %+v", result)
	}
	installedLight := readString(t, replaced)
	created := filepath.Join(targetRoot, "sdd-implement-1.md")
	installedCreated := readString(t, created)

	before := snapshotTree(t, home)
	preview, err := PreviewRestoreBackup(home, result.BackupID)
	if err != nil {
		t.Fatalf("PreviewRestoreBackup: %v", err)
	}
	assertSnapshotUnchanged(t, before, snapshotTree(t, home), "PreviewRestoreBackup")

	if preview.BackupID != result.BackupID || filepath.Clean(preview.BackupPath) != filepath.Clean(result.BackupPath) {
		t.Fatalf("unexpected backup identity: %+v", preview)
	}
	if len(preview.Files) != 2 {
		t.Fatalf("expected two preview files, got %+v", preview.Files)
	}
	if preview.Counts["total"] != 2 || preview.Counts[FileOpUpdate] != 1 || preview.Counts[FileOpRemove] != 1 ||
		preview.Counts[FileOpCreate] != 0 || preview.Counts[FileOpNoOp] != 0 {
		t.Fatalf("unexpected counts: %+v", preview.Counts)
	}

	byName := map[string]FilePreview{}
	for _, file := range preview.Files {
		byName[filepath.Base(file.Path)] = file
	}
	restored := byName["sdd-lightweight.md"]
	if restored.Op != FileOpUpdate || restored.Removed || restored.Conflict {
		t.Fatalf("unexpected restored preview: %+v", restored)
	}
	if string(restored.OldBytes) != installedLight || restored.OldDigest != digestOf([]byte(installedLight)) {
		t.Fatalf("restored old side mismatch: %+v", restored)
	}
	if string(restored.NewBytes) != "original unmanaged content\n" || restored.NewDigest != digestOf(restored.NewBytes) {
		t.Fatalf("restored new side mismatch: %+v", restored)
	}
	removal := byName["sdd-implement-1.md"]
	if removal.Op != FileOpRemove || !removal.Removed || removal.NewBytes != nil {
		t.Fatalf("unexpected removal preview: %+v", removal)
	}
	if string(removal.OldBytes) != installedCreated || removal.OldDigest == "" {
		t.Fatalf("removal old side mismatch: %+v", removal)
	}

	// Parity with the writer: the preview's new bytes are exactly what
	// RestoreBackup writes, and its removal is exactly what is removed.
	var out bytes.Buffer
	if _, err := RestoreBackup(home, result.BackupID, &out); err != nil {
		t.Fatalf("RestoreBackup after preview: %v\noutput: %s", err, out.String())
	}
	if got := readString(t, replaced); got != string(restored.NewBytes) {
		t.Fatalf("restored bytes differ from the preview\nwant: %q\ngot:  %q", restored.NewBytes, got)
	}
	if _, err := os.Lstat(created); !os.IsNotExist(err) {
		t.Fatalf("previewed removal was not removed: %v", err)
	}
}

func TestFleetRestorePreviewRefusals(t *testing.T) {
	t.Run("malformed and missing backup ids write nothing", func(t *testing.T) {
		home := newFleetHome(t)
		if _, err := PreviewRestoreBackup(home, "../escape"); err == nil {
			t.Fatal("expected a malformed backup id to be refused")
		}
		if _, err := os.Lstat(filepath.Join(home, ".agentic-sdd", "backups", "escape")); !os.IsNotExist(err) {
			t.Fatalf("malformed id touched a path: %v", err)
		}
		missing := "20260101T000000.000000000Z"
		if _, err := PreviewRestoreBackup(home, missing); err == nil {
			t.Fatal("expected a missing backup to be refused")
		}
		if _, err := os.Lstat(skillsync.BackupRootFor("", home)); !os.IsNotExist(err) {
			t.Fatalf("preview created the backup root: %v", err)
		}
	})

	t.Run("legacy backup without a fleet manifest", func(t *testing.T) {
		home := newFleetHome(t)
		root := skillsync.BackupRootFor("", home)
		id := "20260101T000000.000000000Z"
		dir := filepath.Join(root, id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"format":3,"id":"`+id+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := PreviewRestoreBackup(home, id); !errors.Is(err, ErrLegacyProvenance) {
			t.Fatalf("expected ErrLegacyProvenance, got %v", err)
		}
	})

	t.Run("non-applied transaction state", func(t *testing.T) {
		home := newFleetHome(t)
		root := skillsync.BackupRootFor("", home)
		id := "20260101T000000.000000000Z"
		dir := filepath.Join(root, id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		manifest := &Manifest{
			Format:     ManifestFormat,
			ID:         id,
			Operation:  "apply",
			State:      ManifestStateFailed,
			CreatedAt:  "2026-01-01T00:00:00Z",
			Home:       home,
			BackupRoot: root,
			TargetRoot: TargetRootForHome(home),
			Scope:      ScopeGlobal,
		}
		if err := writeManifestFile(dir, manifest); err != nil {
			t.Fatal(err)
		}
		if _, err := PreviewRestoreBackup(home, id); !errors.Is(err, ErrLegacyProvenance) {
			t.Fatalf("expected ErrLegacyProvenance, got %v", err)
		}
	})

	t.Run("corrupt fleet manifest", func(t *testing.T) {
		home := newFleetHome(t)
		root := skillsync.BackupRootFor("", home)
		id := "20260101T000000.000000000Z"
		dir := filepath.Join(root, id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(`{"format":`), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := PreviewRestoreBackup(home, id); err == nil {
			t.Fatal("expected a corrupt fleet manifest to be refused")
		}
	})

	t.Run("modified target refuses exactly like restore", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		result := applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		if err := os.WriteFile(target, []byte("modified after apply\n"), 0600); err != nil {
			t.Fatal(err)
		}
		before := snapshotTree(t, home)
		if _, err := PreviewRestoreBackup(home, result.BackupID); !errors.Is(err, ErrUnsafeTarget) {
			t.Fatalf("expected ErrUnsafeTarget, got %v", err)
		}
		assertSnapshotUnchanged(t, before, snapshotTree(t, home), "refused restore preview")
		var out bytes.Buffer
		if _, err := RestoreBackup(home, result.BackupID, &out); !errors.Is(err, ErrUnsafeTarget) {
			t.Fatalf("expected RestoreBackup to refuse identically, got %v", err)
		}
		if got := readString(t, target); got != "modified after apply\n" {
			t.Fatalf("modified target was overwritten: %q", got)
		}
	})

	t.Run("recreated removal refuses exactly like restore", func(t *testing.T) {
		home := newFleetHome(t)
		configDir := t.TempDir()
		configPath := writeFleetConfig(t, configDir, singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		disabledConfig := writeFleetConfig(t, configDir, `"sdd-lightweight":{"source":"agent:sdd-lightweight","enabled":false}`)
		removal := applyPlan(t, buildPlanFor(t, disabledConfig, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		if err := os.WriteFile(target, []byte("recreated by the user\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := PreviewRestoreBackup(home, removal.BackupID); !errors.Is(err, ErrUnsafeTarget) {
			t.Fatalf("expected ErrUnsafeTarget, got %v", err)
		}
		var out bytes.Buffer
		if _, err := RestoreBackup(home, removal.BackupID, &out); !errors.Is(err, ErrUnsafeTarget) {
			t.Fatalf("expected RestoreBackup to refuse identically, got %v", err)
		}
		if got := readString(t, target); got != "recreated by the user\n" {
			t.Fatalf("recreated file was overwritten: %q", got)
		}
	})

	t.Run("symlinked target is never followed", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		result := applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		outside := filepath.Join(t.TempDir(), "outside.md")
		if err := os.WriteFile(outside, []byte("outside content\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(target); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, target); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := PreviewRestoreBackup(home, result.BackupID); err == nil {
			t.Fatal("expected a symlinked target to be refused")
		}
		var out bytes.Buffer
		if _, err := RestoreBackup(home, result.BackupID, &out); err == nil {
			t.Fatal("expected RestoreBackup to refuse the symlink")
		}
		if got := readString(t, outside); got != "outside content\n" {
			t.Fatalf("preview or restore followed the symlink: %q", got)
		}
		if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("symlink was replaced: %v %v", info, err)
		}
	})

	t.Run("incomplete backup copy is refused", func(t *testing.T) {
		home := newFleetHome(t)
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, "sdd-lightweight.md")
		if err := os.WriteFile(target, []byte("before apply\n"), 0644); err != nil {
			t.Fatal(err)
		}
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		result := applyPlan(t, buildPlanFor(t, configPath, home, true), home)
		if err := os.Remove(filepath.Join(result.BackupPath, "opencode-agents", "sdd-lightweight.md")); err != nil {
			t.Fatal(err)
		}
		if _, err := PreviewRestoreBackup(home, result.BackupID); err == nil {
			t.Fatal("expected an incomplete backup copy to be refused")
		}
		var out bytes.Buffer
		if _, err := RestoreBackup(home, result.BackupID, &out); err == nil {
			t.Fatal("expected RestoreBackup to refuse the incomplete backup")
		}
		if got := readString(t, target); got == "before apply\n" {
			t.Fatalf("refused restore changed the target: %q", got)
		}
	})
}
