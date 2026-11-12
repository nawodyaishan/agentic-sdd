package fleet

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// snapshotTree captures every path under root as a stable digest string, so a
// read-only operation can be proven not to have created, removed, chmod-ed or
// rewritten anything. A missing root is an empty snapshot.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			snapshot[rel] = "dir"
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			snapshot[rel] = "special:" + info.Mode().Type().String()
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[rel] = info.Mode().Perm().String() + ":" + digestOf(data)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot
		}
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return snapshot
}

func assertSnapshotUnchanged(t *testing.T, before, after map[string]string, label string) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("%s modified the tree\nbefore: %v\nafter:  %v", label, before, after)
	}
}

// writeManualLiveManifest installs a hand-built live manifest, used only to
// construct legacy/unknown provenance fixtures that a normal apply cannot make.
func writeManualLiveManifest(t *testing.T, home, targetRoot string, files []FileRecord) {
	t.Helper()
	manifest := &Manifest{
		Format:     ManifestFormat,
		ID:         "manual-fixture",
		Operation:  "apply",
		State:      ManifestStateApplied,
		CreatedAt:  "2026-01-01T00:00:00Z",
		Home:       home,
		BackupRoot: skillsync.BackupRootFor("", home),
		TargetRoot: targetRoot,
		Scope:      ScopeGlobal,
		Files:      files,
	}
	if err := WriteLiveManifest(home, manifest); err != nil {
		t.Fatalf("write manual live manifest: %v", err)
	}
}

func TestFleetStatusZeroOneAndReadOnly(t *testing.T) {
	home := newFleetHome(t)
	dir := t.TempDir()

	emptyConfig := writeFleetConfig(t, dir, "")
	before := snapshotTree(t, home)
	empty, err := BuildStatus(emptyConfig, "", home)
	if err != nil {
		t.Fatalf("BuildStatus zero instances: %v", err)
	}
	if len(empty.Instances) != 0 || empty.Counts[StatusCountInstances] != 0 {
		t.Fatalf("expected zero instances, got %+v", empty)
	}
	if empty.TargetRoot != TargetRootForHome(home) || empty.ConfigDigest == "" {
		t.Fatalf("unexpected status identity: %+v", empty)
	}
	assertSnapshotUnchanged(t, before, snapshotTree(t, home), "BuildStatus zero instances")

	oneConfig := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	one, err := BuildStatus(oneConfig, "", home)
	if err != nil {
		t.Fatalf("BuildStatus one instance: %v", err)
	}
	if len(one.Instances) != 1 {
		t.Fatalf("expected one instance, got %+v", one.Instances)
	}
	inst := one.Instances[0]
	if inst.InstanceID != "sdd-lightweight" || inst.SourceRef != "agent:sdd-lightweight" {
		t.Fatalf("unexpected instance identity: %+v", inst)
	}
	if inst.SourceDigest == "" || inst.DesiredDigest == "" || inst.CurrentDigest != "" {
		t.Fatalf("unexpected digests: %+v", inst)
	}
	if inst.ConfiguredModel != "9router/ag/gemini-3.8-flash-high" || inst.RenderedModel != inst.ConfiguredModel {
		t.Fatalf("unexpected models: %+v", inst)
	}
	if !inst.Enabled || inst.Installed != InstalledAbsent || inst.Provenance != ProvenanceUnknown || inst.Conflict != "" {
		t.Fatalf("unexpected absent projection: %+v", inst)
	}
	if one.Counts[StatusCountEnabled] != 1 || one.Counts[StatusCountDisabled] != 0 ||
		one.Counts[string(InstalledAbsent)] != 1 || one.Counts[StatusCountConflicts] != 0 {
		t.Fatalf("unexpected counts: %+v", one.Counts)
	}
}

func TestFleetStatusManagedPendingAndDrifted(t *testing.T) {
	home := newFleetHome(t)
	configDir := t.TempDir()
	configPath := writeFleetConfig(t, configDir,
		`"sdd-lightweight":{"source":"agent:sdd-lightweight","model":"9router/ag/gemini-3.8-flash-high"},`+
			`"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"opencode-go/deepseek-v4.1-flash"}`)
	applyPlan(t, buildPlanFor(t, configPath, home, false), home)

	status, err := BuildStatus(configPath, "", home)
	if err != nil {
		t.Fatalf("BuildStatus after apply: %v", err)
	}
	if len(status.Instances) != 2 {
		t.Fatalf("expected two instances, got %+v", status.Instances)
	}
	for _, inst := range status.Instances {
		if inst.Installed != InstalledManagedCurrent || inst.Provenance != ProvenanceManaged {
			t.Fatalf("expected managed-current, got %+v", inst)
		}
		if inst.CurrentDigest == "" || inst.CurrentDigest != inst.DesiredDigest || inst.CurrentDigest != inst.LastAppliedDigest {
			t.Fatalf("expected converged digests, got %+v", inst)
		}
		if inst.Conflict != "" {
			t.Fatalf("unexpected conflict: %+v", inst)
		}
	}
	if status.Counts[string(InstalledManagedCurrent)] != 2 || status.Counts[StatusCountConflicts] != 0 {
		t.Fatalf("unexpected counts: %+v", status.Counts)
	}

	// A saved model change that has not been applied is a pending apply, not
	// drift and not a conflict.
	writeFleetConfig(t, configDir,
		`"sdd-lightweight":{"source":"agent:sdd-lightweight","model":"9router/ag/claude-sonnet-5.5-high"},`+
			`"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"opencode-go/deepseek-v4.1-flash"}`)
	before := snapshotTree(t, home)
	status, err = BuildStatus(configPath, "", home)
	if err != nil {
		t.Fatalf("BuildStatus pending apply: %v", err)
	}
	assertSnapshotUnchanged(t, before, snapshotTree(t, home), "BuildStatus pending apply")
	if got := instanceByID(t, status, "sdd-lightweight"); got.Installed != InstalledPendingApply {
		t.Fatalf("expected pending-apply, got %+v", got)
	}
	if got := instanceByID(t, status, "sdd-implement-1"); got.Installed != InstalledManagedCurrent {
		t.Fatalf("expected the untouched instance to stay managed-current, got %+v", got)
	}

	// A manual edit to a managed file is drift with an explicit conflict.
	target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, append(data, []byte("\nmanual edit\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	status, err = BuildStatus(configPath, "", home)
	if err != nil {
		t.Fatalf("BuildStatus drift: %v", err)
	}
	drifted := instanceByID(t, status, "sdd-lightweight")
	if drifted.Installed != InstalledManagedDrifted || !strings.Contains(drifted.Conflict, "manual drift") {
		t.Fatalf("expected managed-drifted with a conflict, got %+v", drifted)
	}
	if status.Counts[StatusCountConflicts] != 1 {
		t.Fatalf("expected one conflict, got %+v", status.Counts)
	}

	// A deleted managed file is drift too, and the reason says it is missing.
	if err := os.Remove(filepath.Join(TargetRootForHome(home), "sdd-implement-1.md")); err != nil {
		t.Fatal(err)
	}
	status, err = BuildStatus(configPath, "", home)
	if err != nil {
		t.Fatalf("BuildStatus missing target: %v", err)
	}
	missing := instanceByID(t, status, "sdd-implement-1")
	if missing.Installed != InstalledManagedDrifted || !strings.Contains(missing.Conflict, "missing") {
		t.Fatalf("expected managed-drifted missing, got %+v", missing)
	}
}

func TestFleetStatusUnmanagedLegacyAndUnsafe(t *testing.T) {
	home := newFleetHome(t)
	targetRoot := TargetRootForHome(home)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := writeFleetConfig(t, t.TempDir(),
		`"myself":{"source":"agent:sdd-lightweight"},`+
			`"legacy-one":{"source":"preset:sdd-implement-1"},`+
			`"linked":{"source":"agent:sdd-orchestrator"}`)

	// Unmanaged file: present, never recorded.
	if err := os.WriteFile(filepath.Join(targetRoot, "myself.md"), []byte("user-owned\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Legacy provenance record: recorded ownership that fleet must not treat
	// as clean or managed.
	legacyTarget := filepath.Join(targetRoot, "legacy-one.md")
	if err := os.WriteFile(legacyTarget, []byte("legacy content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	writeManualLiveManifest(t, home, targetRoot, []FileRecord{{
		InstanceID: "legacy-one",
		SourceRef:  "preset:sdd-implement-1",
		Scope:      ScopeGlobal,
		Target:     legacyTarget,
		Provenance: ProvenanceLegacy,
	}})

	// Unsafe target: a symlink is reported, never followed.
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(targetRoot, "linked.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	before := snapshotTree(t, home)
	status, err := BuildStatus(configPath, "", home)
	if err != nil {
		t.Fatalf("BuildStatus: %v", err)
	}
	assertSnapshotUnchanged(t, before, snapshotTree(t, home), "BuildStatus unmanaged/legacy/unsafe")

	if got := instanceByID(t, status, "myself"); got.Installed != InstalledUnmanaged || got.Conflict == "" {
		t.Fatalf("expected unmanaged with a conflict, got %+v", got)
	}
	if got := instanceByID(t, status, "legacy-one"); got.Installed != InstalledLegacyUnknown ||
		got.Provenance != ProvenanceLegacy || got.Conflict == "" {
		t.Fatalf("expected legacy-unknown with a conflict, got %+v", got)
	}
	if got := instanceByID(t, status, "linked"); got.Installed != InstalledUnsafeTarget || got.Conflict == "" {
		t.Fatalf("expected unsafe-target with a conflict, got %+v", got)
	}
	if got := status.Counts[string(InstalledUnmanaged)]; got != 1 {
		t.Fatalf("expected one unmanaged count, got %+v", status.Counts)
	}
}

func TestFleetStatusDisabled(t *testing.T) {
	home := newFleetHome(t)
	configDir := t.TempDir()
	configPath := writeFleetConfig(t, configDir,
		`"sdd-lightweight":{"source":"agent:sdd-lightweight","model":"9router/ag/gemini-3.8-flash-high"}`)
	applyPlan(t, buildPlanFor(t, configPath, home, false), home)

	// Disabled while still installed: a managed removal is pending.
	writeFleetConfig(t, configDir, `"sdd-lightweight":{"source":"agent:sdd-lightweight","enabled":false}`)
	status, err := BuildStatus(configPath, "", home)
	if err != nil {
		t.Fatalf("BuildStatus disabled: %v", err)
	}
	disabled := instanceByID(t, status, "sdd-lightweight")
	if disabled.Enabled || disabled.Installed != InstalledDisabled || disabled.DesiredDigest != "" {
		t.Fatalf("expected disabled with a pending managed removal, got %+v", disabled)
	}
	if status.Counts[StatusCountDisabled] != 1 {
		t.Fatalf("expected one disabled instance, got %+v", status.Counts)
	}

	// Disabled with no managed file is simply absent.
	emptyHome := newFleetHome(t)
	status, err = BuildStatus(configPath, "", emptyHome)
	if err != nil {
		t.Fatalf("BuildStatus disabled absent: %v", err)
	}
	if got := instanceByID(t, status, "sdd-lightweight"); got.Installed != InstalledAbsent {
		t.Fatalf("expected absent, got %+v", got)
	}
	if status.Counts[string(InstalledAbsent)] != 1 {
		t.Fatalf("unexpected counts: %+v", status.Counts)
	}
}

func TestFleetStatusScale(t *testing.T) {
	dir := t.TempDir()
	c0 := writeFleetConfig(t, dir, "")
	c1 := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	c18 := writeScaleConfig(t, dir, "fleet18.json", 18)
	c100 := writeScaleConfig(t, dir, "fleet100.json", 100)

	for _, tc := range []struct {
		name      string
		config    string
		instances int
		enabled   int
		disabled  int
		conflicts int
	}{
		{name: "zero", config: c0, instances: 0, enabled: 0, disabled: 0, conflicts: 0},
		{name: "one", config: c1, instances: 1, enabled: 1, disabled: 0, conflicts: 0},
		{name: "eighteen", config: c18, instances: 18, enabled: 17, disabled: 1, conflicts: 0},
		{name: "hundred", config: c100, instances: 100, enabled: 100, disabled: 0, conflicts: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := newFleetHome(t)
			before := snapshotTree(t, home)
			status, err := BuildStatus(tc.config, "", home)
			if err != nil {
				t.Fatalf("BuildStatus: %v", err)
			}
			assertSnapshotUnchanged(t, before, snapshotTree(t, home), "BuildStatus "+tc.name)
			if len(status.Instances) != tc.instances {
				t.Fatalf("instances = %d, want %d", len(status.Instances), tc.instances)
			}
			if status.Counts[StatusCountInstances] != tc.instances ||
				status.Counts[StatusCountEnabled] != tc.enabled ||
				status.Counts[StatusCountDisabled] != tc.disabled ||
				status.Counts[StatusCountConflicts] != tc.conflicts {
				t.Fatalf("unexpected counts: %+v", status.Counts)
			}
			if status.Counts[string(InstalledAbsent)] != tc.instances {
				t.Fatalf("expected every fresh instance to be absent: %+v", status.Counts)
			}
			// Deterministic over identical inputs.
			again, err := BuildStatus(tc.config, "", home)
			if err != nil {
				t.Fatalf("BuildStatus repeat: %v", err)
			}
			if !reflect.DeepEqual(status, again) {
				t.Fatalf("BuildStatus is not deterministic")
			}
		})
	}
}

func TestFleetStatusRefusesInvalidConfig(t *testing.T) {
	home := newFleetHome(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "fleet.json")
	if err := os.WriteFile(configPath, []byte(`{"schemaVersion":1,"instances":{"bad":{"source":"agent:does-not-exist"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildStatus(configPath, "", home); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for an unknown source, got %v", err)
	}
}

func instanceByID(t *testing.T, status *Status, id string) InstanceStatus {
	t.Helper()
	for _, inst := range status.Instances {
		if inst.InstanceID == id {
			return inst
		}
	}
	t.Fatalf("instance %q not found in %+v", id, status.Instances)
	return InstanceStatus{}
}
