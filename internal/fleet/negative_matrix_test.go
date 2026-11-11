package fleet

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// TestFleetSidecarApplyingAndFailedState verifies that the sidecar lifecycle
// records "applying" before commit, and upon engine failure records "failed"
// with rollback information while keeping the backup recoverable (AC-12 / FR-08).
func TestFleetSidecarApplyingAndFailedState(t *testing.T) {
	home := newFleetHome(t)
	targetRoot := TargetRootForHome(home)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	plan := buildPlanFor(t, configPath, home, false)

	// Make targetRoot read-only so rename fails after backup is created, exercising failed state
	if err := os.Chmod(targetRoot, 0555); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(targetRoot, 0755)
	}()

	var out bytes.Buffer
	result, err := ApplyPlan(plan, home, &out)
	if err == nil {
		t.Fatal("expected ApplyPlan to fail when targetRoot is read-only")
	}
	if result.BackupPath == "" {
		t.Fatal("expected recoverable backup path on failed apply")
	}

	// Verify the sidecar in the backup directory has state "failed"
	sidecar, readErr := ReadManifestFile(filepath.Join(result.BackupPath, ManifestFileName))
	if readErr != nil || sidecar == nil {
		t.Fatalf("failed to read sidecar from backup: %v", readErr)
	}
	if sidecar.State != ManifestStateFailed {
		t.Fatalf("expected sidecar state %q, got %q", ManifestStateFailed, sidecar.State)
	}
	if sidecar.Rollback == nil {
		t.Fatalf("expected rollback state to be recorded in sidecar")
	}

	// Attempting to restore this failed backup must be refused (restore.go line 84)
	var restoreOut bytes.Buffer
	_, restoreErr := RestoreBackup(home, result.BackupID, &restoreOut)
	if !errors.Is(restoreErr, ErrLegacyProvenance) {
		t.Fatalf("expected ErrLegacyProvenance when restoring a failed backup, got %v", restoreErr)
	}
}

// TestFleetNegativeMatrixRow3DuplicateInstalledNameOrAliasToAlias verifies
// spec §8 row 3: two instances mapping to the same installed name or an alias
// referencing another alias must be rejected.
func TestFleetNegativeMatrixRow3DuplicateInstalledNameOrAliasToAlias(t *testing.T) {
	home := newFleetHome(t)

	// 1. Alias to alias is rejected by Resolve
	pack, err := skillsync.OpenCodePackFS("")
	if err != nil {
		t.Fatal(err)
	}
	cfgAlias := &Config{
		SchemaVersion: SchemaVersion,
		Fleet:         "test",
		Instances: map[string]Instance{
			"alias1": {Source: "agent:alias2", Model: "opencode-go/deepseek-v4.1-flash"},
		},
	}
	if _, err := Resolve(pack, cfgAlias); err == nil {
		t.Fatal("expected alias-to-alias to be rejected by Resolve")
	}

	// 2. Duplicate installed names in plan operations rejected by ApplyPlan
	configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	plan := buildPlanFor(t, configPath, home, false)
	plan.ProposedOps = append(plan.ProposedOps, plan.ProposedOps[0])
	plan.FileList = append(plan.FileList, plan.FileList[0])
	_ = plan.Seal()
	var out bytes.Buffer
	if _, err := ApplyPlan(plan, home, &out); err == nil {
		t.Fatal("expected duplicate agent file in plan to be rejected by ApplyPlan")
	}
}

// TestFleetNegativeMatrixRow5OpenCodeOverlapVsFleet verifies spec §8 row 5:
// simple --with-opencode follows 009 Option B replacement behavior without invoking
// fleet mode, and fleet mode does not silently override or touch 009 sync.
func TestFleetNegativeMatrixRow5OpenCodeOverlapVsFleet(t *testing.T) {
	home := newFleetHome(t)
	targetRoot := TargetRootForHome(home)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}

	lwTarget := filepath.Join(targetRoot, "sdd-lightweight.md")
	originalContent := []byte("original unmanaged content before sync\n")
	if err := os.WriteFile(lwTarget, originalContent, 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Run SyncWithOptionsResult with WithOpenCode: true (009 Option B)
	opts := skillsync.Options{
		WithOpenCode: true,
	}
	var out bytes.Buffer
	res, err := skillsync.SyncWithOptionsResult("", home, true, opts, &out)
	if err != nil {
		t.Fatalf("SyncWithOptionsResult 009 failed: %v", err)
	}
	if len(res.Applied) == 0 {
		t.Fatal("expected 009 to replace overlap profiles")
	}

	// 2. 009 sync must NOT create fleet-manifest.json or touch live fleet pointer
	if live, _ := ReadLiveManifest(home); live != nil {
		t.Fatal("009 sync must not create a live fleet manifest")
	}

	// 3. Fleet plan over this 009-installed file detects it as unmanaged/legacy provenance
	configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	fleetPlan, err := BuildPlan(configPath, "", home, false)
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if fleetPlan.ProposedOps[0].Op != "conflict" {
		t.Fatalf("fleet plan must report conflict against 009 unadopted file, got %s", fleetPlan.ProposedOps[0].Op)
	}
}

// TestFleetNegativeMatrixRow12SpecPlanAdapterNonwritingPolicy verifies spec §8 row 12:
// sdd-spec-9r and sdd-plan-9r adapter roles enforce non-writing and non-shell policy;
// any attempt to smuggle edit or shell access fails closed.
func TestFleetNegativeMatrixRow12SpecPlanAdapterNonwritingPolicy(t *testing.T) {
	pack, err := skillsync.OpenCodePackFS(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	specBytes, err := fs.ReadFile(pack, "adapters/sdd-spec-9r.md")
	if err != nil {
		t.Fatalf("read adapters/sdd-spec-9r.md: %v", err)
	}
	planBytes, err := fs.ReadFile(pack, "adapters/sdd-plan-9r.md")
	if err != nil {
		t.Fatalf("read adapters/sdd-plan-9r.md: %v", err)
	}

	for name, data := range map[string][]byte{"sdd-spec-9r": specBytes, "sdd-plan-9r": planBytes} {
		fields, rules := parseAdapterPolicy(t, data)
		assertAdapterPolicy(t, name, fields, rules)

		// Tampering with permissions fails VerifyModelOnly
		tampered := bytes.Replace(data, []byte("mode: subagent"), []byte("mode: subagent\npermissions:\n  - action: edit\n    resource: \"*\"\n    effect: allow"), 1)
		if err := VerifyModelOnly(data, tampered); err == nil {
			t.Fatalf("%s tampered with edit:allow was not rejected by VerifyModelOnly", name)
		}
	}
}

// TestFleetNegativeMatrixRow18OrchestratorInstallDoesNotChangeDefault verifies spec §8 row 18:
// installing the orchestrator profile does not alter OpenCode default agent configuration.
func TestFleetNegativeMatrixRow18OrchestratorInstallDoesNotChangeDefault(t *testing.T) {
	home := newFleetHome(t)
	targetRoot := TargetRootForHome(home)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := writeFleetConfig(t, t.TempDir(),
		`"sdd-orchestrator":{"source":"agent:sdd-orchestrator","enabled":true}`)
	plan := buildPlanFor(t, configPath, home, false)
	res := applyPlan(t, plan, home)
	if res.Created != 1 {
		t.Fatalf("expected 1 created profile, got %+v", res)
	}

	target := filepath.Join(targetRoot, "sdd-orchestrator.md")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("orchestrator profile was not installed: %v", err)
	}

	// Verify no OpenCode default-agent or configuration was created or touched
	opencodeConfig := filepath.Join(home, ".config", "opencode", "opencode.json")
	if _, err := os.Stat(opencodeConfig); !os.IsNotExist(err) {
		t.Fatalf("orchestrator install created unexpected config: %s", opencodeConfig)
	}
}

// TestFleetNegativeMatrixRow19GitRevertRequiresExplicitApply verifies spec §8 row 19:
// reverting fleet.json (simulating git revert) does not automatically modify installed files;
// changes take effect only after explicit fleet plan and apply.
func TestFleetNegativeMatrixRow19GitRevertRequiresExplicitApply(t *testing.T) {
	home := newFleetHome(t)
	targetRoot := TargetRootForHome(home)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}

	cfgDir := t.TempDir()
	configPath := writeFleetConfig(t, cfgDir, singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	plan1 := buildPlanFor(t, configPath, home, false)
	applyPlan(t, plan1, home)

	target := filepath.Join(targetRoot, "sdd-lightweight.md")
	installedV1 := readString(t, target)
	if !strings.Contains(installedV1, "9router/ag/gemini-3.8-flash-high") {
		t.Fatal("v1 not installed")
	}

	// Update to v2
	v2Content := `"sdd-lightweight":{"source":"agent:sdd-lightweight","model":"9router/ag/claude-sonnet-5.5-high"}`
	_ = writeFleetConfig(t, cfgDir, v2Content)
	plan2 := buildPlanFor(t, configPath, home, false)
	applyPlan(t, plan2, home)
	installedV2 := readString(t, target)
	if !strings.Contains(installedV2, "9router/ag/claude-sonnet-5.5-high") {
		t.Fatal("v2 not installed")
	}

	// Git user reverts fleet.json back to v1
	_ = writeFleetConfig(t, cfgDir, singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))

	// Files on disk MUST NOT have changed automatically
	if current := readString(t, target); current != installedV2 {
		t.Fatalf("installed agent file mutated without apply: %s", current)
	}

	// BuildPlan detects update needed
	revertPlan := buildPlanFor(t, configPath, home, false)
	if revertPlan.ProposedOps[0].Op != "update" {
		t.Fatalf("expected update op, got %s", revertPlan.ProposedOps[0].Op)
	}

	// Explicit apply reverts file to v1
	applyPlan(t, revertPlan, home)
	if current := readString(t, target); current != installedV1 {
		t.Fatalf("reverted content mismatch after apply: %s", current)
	}
}
