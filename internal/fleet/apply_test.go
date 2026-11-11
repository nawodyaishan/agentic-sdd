package fleet

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

func newFleetHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeFleetConfig(t *testing.T, dir, instances string) string {
	t.Helper()
	path := filepath.Join(dir, "fleet.json")
	body := `{"schemaVersion":1,"fleet":"test","instances":{` + instances + `}}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func singleInstance(source, model string) string {
	return `"sdd-lightweight":{"source":"` + source + `","model":"` + model + `"}`
}

func buildPlanFor(t *testing.T, configPath, home string, adopt bool) *Plan {
	t.Helper()
	plan, err := BuildPlan(configPath, "", home, adopt)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	return plan
}

func applyPlan(t *testing.T, plan *Plan, home string) ApplyResult {
	t.Helper()
	var out bytes.Buffer
	result, err := ApplyPlan(plan, home, &out)
	if err != nil {
		t.Fatalf("ApplyPlan: %v\noutput: %s", err, out.String())
	}
	return result
}

func readString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func backupDirs(t *testing.T, home string) []string {
	t.Helper()
	root := skillsync.BackupRootFor("", home)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "20") {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs
}

func latestBackup(t *testing.T, home string) string {
	t.Helper()
	dirs := backupDirs(t, home)
	if len(dirs) == 0 {
		t.Fatal("expected a backup directory")
	}
	return dirs[len(dirs)-1]
}

func TestFleetApply(t *testing.T) {
	t.Run("create writes rendered bytes and format-4 provenance", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		agentsMD := filepath.Join(targetRoot, "AGENTS.md")
		if err := os.WriteFile(agentsMD, []byte("user-owned instructions\n"), 0644); err != nil {
			t.Fatal(err)
		}
		unrelated := filepath.Join(targetRoot, "user-agent.md")
		if err := os.WriteFile(unrelated, []byte("user file\n"), 0644); err != nil {
			t.Fatal(err)
		}

		plan := buildPlanFor(t, configPath, home, false)
		if len(plan.ProposedOps) != 1 || plan.ProposedOps[0].Op != "create" {
			t.Fatalf("expected one create op, got %+v", plan.ProposedOps)
		}
		op := plan.ProposedOps[0]
		if op.DesiredDigest == "" || op.SourceDigest == "" {
			t.Fatalf("plan op is missing digests: %+v", op)
		}
		if !filepath.IsAbs(plan.ConfigPath) || plan.Scope != ScopeGlobal || plan.RendererVersion != RendererVersion {
			t.Fatalf("unexpected plan identity: %+v", plan)
		}

		result := applyPlan(t, plan, home)
		if result.NoOp || result.Created != 1 || result.BackupID == "" {
			t.Fatalf("unexpected apply result: %+v", result)
		}

		target := filepath.Join(targetRoot, "sdd-lightweight.md")
		installed := readString(t, target)
		if !strings.Contains(installed, "model: 9router/ag/gemini-3.8-flash-high\n") {
			t.Fatalf("model override missing:\n%s", installed)
		}
		if strings.Contains(installed, "\ndisabled:") || strings.Contains(installed, "\ndisabled :") {
			t.Fatalf("apply invented a disabled field:\n%s", installed)
		}
		if got := readString(t, agentsMD); got != "user-owned instructions\n" {
			t.Fatalf("AGENTS.md was modified: %q", got)
		}
		if got := readString(t, unrelated); got != "user file\n" {
			t.Fatalf("unrelated file was modified: %q", got)
		}

		pack, err := skillsync.OpenCodePackFS("")
		if err != nil {
			t.Fatal(err)
		}
		source, err := fs.ReadFile(pack, "agents/sdd-lightweight.md")
		if err != nil {
			t.Fatal(err)
		}
		if err := VerifyModelOnly(source, []byte(installed)); err != nil {
			t.Fatalf("installed profile is not a model-only render: %v", err)
		}

		backup := result.BackupPath
		sidecar, err := ReadManifestFile(filepath.Join(backup, ManifestFileName))
		if err != nil || sidecar == nil {
			t.Fatalf("read sidecar: %v", err)
		}
		if sidecar.Format != ManifestFormat || sidecar.Operation != "apply" || sidecar.State != ManifestStateApplied {
			t.Fatalf("unexpected sidecar identity: %+v", sidecar)
		}
		if sidecar.ID != result.BackupID || filepath.Clean(sidecar.BackupPath) != filepath.Clean(backup) {
			t.Fatalf("sidecar backup identity mismatch: %+v", sidecar)
		}
		if len(sidecar.Files) != 1 {
			t.Fatalf("expected one provenance record, got %+v", sidecar.Files)
		}
		rec := sidecar.Files[0]
		if rec.InstanceID != "sdd-lightweight" || rec.SourceRef != "agent:sdd-lightweight" {
			t.Fatalf("unexpected provenance identity: %+v", rec)
		}
		if rec.Scope != ScopeGlobal || filepath.Clean(rec.Target) != filepath.Clean(target) || rec.Provenance != ProvenanceManaged {
			t.Fatalf("unexpected provenance record: %+v", rec)
		}
		if rec.DesiredDigest == "" || rec.DesiredDigest != rec.InstalledDigest || rec.DesiredDigest != digestOf([]byte(installed)) {
			t.Fatalf("unexpected provenance digests: %+v", rec)
		}
		if rec.ConfigDigest != plan.ConfigDigest || rec.RendererVersion != RendererVersion || rec.SourceDigest == "" {
			t.Fatalf("provenance is missing plan identity: %+v", rec)
		}

		live, err := ReadLiveManifest(home)
		if err != nil || live == nil {
			t.Fatalf("read live manifest: %v", err)
		}
		liveRec, ok := live.RecordFor(target)
		if !ok || !liveRec.IsManaged() || liveRec.DesiredDigest != rec.DesiredDigest {
			t.Fatalf("live manifest does not record ownership: %+v", live)
		}

		// The generic skillsync manifest remains format 3 and untouched.
		generic, err := os.ReadFile(filepath.Join(backup, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(generic), `"format": 3`) {
			t.Fatalf("expected the shared manifest to stay format 3: %s", generic)
		}
	})

	t.Run("reapply without drift is a no-op", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		before := len(backupDirs(t, home))

		plan := buildPlanFor(t, configPath, home, false)
		if len(plan.ProposedOps) != 1 || plan.ProposedOps[0].Op != "noop" {
			t.Fatalf("expected a noop plan, got %+v", plan.ProposedOps)
		}
		result := applyPlan(t, plan, home)
		if !result.NoOp || result.BackupID != "" {
			t.Fatalf("expected a no-op apply, got %+v", result)
		}
		if got := len(backupDirs(t, home)); got != before {
			t.Fatalf("no-op created a backup: %d -> %d", before, got)
		}
	})

	t.Run("changed config or renderer is a stale plan", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		plan := buildPlanFor(t, configPath, home, false)

		changedConfigBytes := []byte(`{"schemaVersion":1,"instances":{"sdd-lightweight":{"source":"agent:sdd-lightweight","model":"9router/ag/claude-sonnet-5.5-high"}}}`)
		if err := os.WriteFile(configPath, changedConfigBytes, 0644); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if _, err := ApplyPlan(plan, home, &out); !errors.Is(err, ErrStalePlan) {
			t.Fatalf("expected ErrStalePlan after config change, got %v", err)
		}
		if _, err := os.Lstat(filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")); !os.IsNotExist(err) {
			t.Fatalf("stale apply wrote a file: %v", err)
		}

		rendererPlan := buildPlanFor(t, writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high")), home, false)
		rendererPlan.RendererVersion = "999"
		if err := rendererPlan.Seal(); err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyPlan(rendererPlan, home, &out); !errors.Is(err, ErrStalePlan) {
			t.Fatalf("expected ErrStalePlan after renderer change, got %v", err)
		}
	})

	t.Run("manual drift is a conflict and is not overwritten", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		if err := os.WriteFile(target, []byte("manually edited\n"), 0644); err != nil {
			t.Fatal(err)
		}

		plan := buildPlanFor(t, configPath, home, false)
		if !plan.HasConflicts() || plan.ProposedOps[0].Op != "conflict" {
			t.Fatalf("expected a manual-drift conflict, got %+v", plan.ProposedOps)
		}
		var out bytes.Buffer
		if _, err := ApplyPlan(plan, home, &out); !errors.Is(err, ErrPlanConflict) {
			t.Fatalf("expected ErrPlanConflict, got %v", err)
		}
		if got := readString(t, target); got != "manually edited\n" {
			t.Fatalf("conflicting apply overwrote the file: %q", got)
		}
	})

	t.Run("unmanaged collision conflicts and --adopt overwrites with a backup", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, "sdd-lightweight.md")
		if err := os.WriteFile(target, []byte("unmanaged content\n"), 0644); err != nil {
			t.Fatal(err)
		}

		plan := buildPlanFor(t, configPath, home, false)
		if plan.ProposedOps[0].Op != "conflict" {
			t.Fatalf("expected an unmanaged collision, got %+v", plan.ProposedOps[0])
		}
		var out bytes.Buffer
		if _, err := ApplyPlan(plan, home, &out); !errors.Is(err, ErrPlanConflict) {
			t.Fatalf("expected ErrPlanConflict, got %v", err)
		}
		if got := readString(t, target); got != "unmanaged content\n" {
			t.Fatalf("conflict overwrote the unmanaged file: %q", got)
		}

		adopt := buildPlanFor(t, configPath, home, true)
		if adopt.ProposedOps[0].Op != "update" || !adopt.ProposedOps[0].Adopt {
			t.Fatalf("expected an explicit adoption op, got %+v", adopt.ProposedOps[0])
		}
		result := applyPlan(t, adopt, home)
		sidecar, err := ReadManifestFile(filepath.Join(result.BackupPath, ManifestFileName))
		if err != nil || sidecar == nil {
			t.Fatalf("read sidecar: %v", err)
		}
		rec := sidecar.Files[0]
		if rec.Provenance != ProvenanceAdopted || rec.BackupRef == "" {
			t.Fatalf("adoption was not recorded with a backup: %+v", rec)
		}
		if rec.BackupDigest != digestOf([]byte("unmanaged content\n")) {
			t.Fatalf("adoption backup digest mismatch: %+v", rec)
		}
		if got := readString(t, filepath.Join(result.BackupPath, "opencode-agents", "sdd-lightweight.md")); got != "unmanaged content\n" {
			t.Fatalf("adoption did not back up the previous bytes: %q", got)
		}
		if got := readString(t, target); !strings.Contains(got, "model: 9router/ag/gemini-3.8-flash-high") {
			t.Fatalf("adoption did not install the render: %q", got)
		}
	})

	t.Run("disabled managed profile is removed with a backup while unmanaged files are preserved", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		targetRoot := TargetRootForHome(home)
		target := filepath.Join(targetRoot, "sdd-lightweight.md")
		installed := readString(t, target)
		unrelated := filepath.Join(targetRoot, "user-agent.md")
		if err := os.WriteFile(unrelated, []byte("keep\n"), 0644); err != nil {
			t.Fatal(err)
		}

		disabledConfig := writeFleetConfig(t, t.TempDir(), `"sdd-lightweight":{"source":"agent:sdd-lightweight","enabled":false}`)
		plan := buildPlanFor(t, disabledConfig, home, false)
		if plan.ProposedOps[0].Op != "remove" {
			t.Fatalf("expected a managed removal, got %+v", plan.ProposedOps[0])
		}
		result := applyPlan(t, plan, home)
		if result.Removed != 1 {
			t.Fatalf("expected one removal, got %+v", result)
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatalf("disabled profile was not removed: %v", err)
		}
		sidecar, err := ReadManifestFile(filepath.Join(result.BackupPath, ManifestFileName))
		if err != nil || sidecar == nil {
			t.Fatalf("read sidecar: %v", err)
		}
		rec := sidecar.Files[0]
		if rec.Provenance != ProvenanceRemoved || rec.BackupRef == "" || rec.BackupDigest != digestOf([]byte(installed)) {
			t.Fatalf("removal provenance is incomplete: %+v", rec)
		}
		if got := readString(t, filepath.Join(result.BackupPath, rec.BackupRef)); got != installed {
			t.Fatal("removal did not save the previous profile")
		}
		if got := readString(t, unrelated); got != "keep\n" {
			t.Fatalf("unrelated file was modified: %q", got)
		}
		if live, _ := ReadLiveManifest(home); live != nil {
			if _, ok := live.RecordFor(target); ok {
				t.Fatalf("removed target is still live-managed: %+v", live)
			}
		}

		// An unmanaged same-name file is preserved when the instance is
		// disabled, and a key absent from the config is ignored entirely.
		unmanagedHome := newFleetHome(t)
		unmanagedRoot := TargetRootForHome(unmanagedHome)
		if err := os.MkdirAll(unmanagedRoot, 0755); err != nil {
			t.Fatal(err)
		}
		unmanagedTarget := filepath.Join(unmanagedRoot, "sdd-lightweight.md")
		if err := os.WriteFile(unmanagedTarget, []byte("user same-name\n"), 0644); err != nil {
			t.Fatal(err)
		}
		preservePlan := buildPlanFor(t, disabledConfig, unmanagedHome, false)
		if preservePlan.ProposedOps[0].Op != "noop" {
			t.Fatalf("unmanaged disabled file must be preserved, got %+v", preservePlan.ProposedOps[0])
		}
		if got := readString(t, unmanagedTarget); got != "user same-name\n" {
			t.Fatalf("unmanaged disabled file was modified: %q", got)
		}

		emptyConfig := writeFleetConfig(t, t.TempDir(), "")
		ignored := buildPlanFor(t, emptyConfig, home, false)
		if len(ignored.ProposedOps) != 0 {
			t.Fatalf("absent config key must be ignored, got %+v", ignored.ProposedOps)
		}
		if got := readString(t, unrelated); got != "keep\n" {
			t.Fatalf("absent-key plan changed files: %q", got)
		}
	})

	t.Run("legacy provenance is never treated as clean", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")

		live, err := ReadLiveManifest(home)
		if err != nil || live == nil {
			t.Fatalf("read live manifest: %v", err)
		}
		for i := range live.Files {
			live.Files[i].Provenance = ProvenanceLegacy
			live.Files[i].DesiredDigest = ""
			live.Files[i].InstalledDigest = ""
		}
		if err := WriteLiveManifest(home, live); err != nil {
			t.Fatal(err)
		}

		plan := buildPlanFor(t, configPath, home, false)
		if plan.ProposedOps[0].Op != "conflict" {
			t.Fatalf("legacy provenance must conflict, got %+v", plan.ProposedOps[0])
		}
		var out bytes.Buffer
		if _, err := ApplyPlan(plan, home, &out); !errors.Is(err, ErrPlanConflict) {
			t.Fatalf("expected ErrPlanConflict, got %v", err)
		}
		adopt := buildPlanFor(t, configPath, home, true)
		if adopt.ProposedOps[0].Op != "update" || !adopt.ProposedOps[0].Adopt {
			t.Fatalf("expected explicit adoption, got %+v", adopt.ProposedOps[0])
		}
		applyPlan(t, adopt, home)
		if got := readString(t, target); !strings.Contains(got, "model: 9router/ag/gemini-3.8-flash-high") {
			t.Fatalf("adoption did not install the render: %q", got)
		}
	})

	t.Run("symlinked targets and target roots are refused", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(t.TempDir(), "outside.md")
		if err := os.WriteFile(outside, []byte("protected\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(targetRoot, "sdd-lightweight.md")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := BuildPlan(configPath, "", home, false); err == nil {
			t.Fatal("expected a symlinked target to be refused")
		}
		if got := readString(t, outside); got != "protected\n" {
			t.Fatalf("external file was modified: %q", got)
		}

		symlinkHome := newFleetHome(t)
		externalDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(symlinkHome, ".config", "opencode"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(externalDir, TargetRootForHome(symlinkHome)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := BuildPlan(configPath, "", symlinkHome, false); err == nil {
			t.Fatal("expected a symlinked target root to be refused")
		}
	})

	t.Run("traversal and AGENTS.md plan targets are refused", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		plan := buildPlanFor(t, configPath, home, false)
		for _, file := range []string{"../evil.md", "AGENTS.md", "sub/agent.md"} {
			mutated := *plan
			mutated.ProposedOps = append([]ProposedOp(nil), plan.ProposedOps...)
			mutated.ProposedOps[0].File = file
			mutated.FileList = append([]string(nil), plan.FileList...)
			mutated.FileList[0] = file
			if err := mutated.Seal(); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if _, err := ApplyPlan(&mutated, home, &out); err == nil {
				t.Fatalf("expected %q to be refused", file)
			}
		}
	})

	t.Run("engine rollback restores committed files and keeps the backup", func(t *testing.T) {
		home := newFleetHome(t)
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		first := filepath.Join(targetRoot, "a.md")
		if err := os.WriteFile(first, []byte("old-a"), 0644); err != nil {
			t.Fatal(err)
		}
		// A non-empty directory at the second destination makes its rename
		// fail after the first change committed, exercising rollback.
		blocked := filepath.Join(targetRoot, "b.md")
		if err := os.MkdirAll(blocked, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(blocked, "marker"), []byte("dir"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := skillsync.ApplyAgentFiles(
			skillsync.BackupRootFor("", home), home, "test",
			[]skillsync.AgentFileChange{
				{File: "a.md", Data: []byte("new-a"), Old: true},
				{File: "b.md", Data: []byte("new-b")},
			}, nil, io.Discard)
		if err == nil {
			t.Fatal("expected the second rename to fail")
		}
		if got := readString(t, first); got != "old-a" {
			t.Fatalf("rollback did not restore the first file: %q", got)
		}
		if got := readString(t, filepath.Join(blocked, "marker")); got != "dir" {
			t.Fatalf("blocked target was modified: %q", got)
		}
		dirs := backupDirs(t, home)
		if len(dirs) != 1 {
			t.Fatalf("expected one recoverable backup, got %v", dirs)
		}
		if _, err := os.Stat(filepath.Join(dirs[0], "manifest.json")); err != nil {
			t.Fatalf("backup manifest missing: %v", err)
		}
	})
}
