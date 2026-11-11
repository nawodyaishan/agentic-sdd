package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFleetPlan(t *testing.T) {
	tempDir := t.TempDir()
	targetRoot := filepath.Join(tempDir, "agents")
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial target agent file
	agentFile := filepath.Join(targetRoot, "sdd-implement-1.md")
	agentContent := []byte("original agent content")
	if err := os.WriteFile(agentFile, agentContent, 0644); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(agentContent)
	agentHash := hex.EncodeToString(h[:])

	configBytes := []byte(`{"schemaVersion":1,"instances":{"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"}}}`)
	cfgHash := sha256.Sum256(configBytes)
	configDigest := hex.EncodeToString(cfgHash[:])

	plan := &Plan{
		SourceRev:          "rev-001",
		ConfigDigest:       configDigest,
		RendererVersion:    "1.0.0",
		TargetRoot:         targetRoot,
		FileList:           []string{"sdd-implement-1.md"},
		LastKnownInstalled: map[string]string{"sdd-implement-1.md": agentHash},
		ProposedOps: []ProposedOp{
			{
				InstanceID:    "sdd-implement-1",
				File:          "sdd-implement-1.md",
				Op:            "update",
				SourceRef:     "preset:sdd-implement-1",
				Model:         "9router/ag/gemini-3.8-flash-high",
				CurrentDigest: agentHash,
			},
		},
		PermNotes: []string{"deny-shell", "deny-subagent"},
	}

	// 1. Verify CanonicalBytes field order
	canonical, err := CanonicalBytes(plan)
	if err != nil {
		t.Fatalf("CanonicalBytes failed: %v", err)
	}
	canonicalStr := string(canonical)

	// Field order: sourceRev, configDigest, rendererVersion, targetRoot, fileList, lastKnownInstalled, proposedOps, permNotes
	expectedOrder := []string{
		`"sourceRev"`,
		`"configDigest"`,
		`"rendererVersion"`,
		`"targetRoot"`,
		`"fileList"`,
		`"lastKnownInstalled"`,
		`"proposedOps"`,
		`"permNotes"`,
	}
	lastIdx := -1
	for _, field := range expectedOrder {
		idx := strings.Index(canonicalStr, field)
		if idx == -1 {
			t.Fatalf("expected field %s in canonical JSON: %s", field, canonicalStr)
		}
		if idx <= lastIdx {
			t.Fatalf("field %s out of order: idx %d <= lastIdx %d in %s", field, idx, lastIdx, canonicalStr)
		}
		lastIdx = idx
	}

	// Verify paths are excluded from CanonicalBytes
	if strings.Contains(canonicalStr, `"configPath"`) || strings.Contains(canonicalStr, `"repoPath"`) {
		t.Fatalf("canonical bytes must not contain configPath or repoPath: %s", canonicalStr)
	}

	// 2. Deterministic PlanID computation and Seal
	if err := plan.Seal(); err != nil {
		t.Fatalf("Seal failed: %v", err)
	}
	if len(plan.PlanID) != 64 {
		t.Fatalf("expected 64-character planID, got %q", plan.PlanID)
	}
	if !plan.VerifyPlanID() {
		t.Fatalf("VerifyPlanID returned false after Seal")
	}

	// Verify planID is invariant to ConfigPath/RepoPath
	p1 := *plan
	p1.ConfigPath = "/path/user1/fleet.json"
	p1.RepoPath = "/repo/user1"
	p2 := *plan
	p2.ConfigPath = "/path/user2/fleet.json"
	p2.RepoPath = "/repo/user2"
	id1, _ := ComputePlanID(&p1)
	id2, _ := ComputePlanID(&p2)
	idPlan, _ := ComputePlanID(plan)
	if id1 != id2 || id1 != idPlan {
		t.Fatalf("planID changed with config/repo paths: %s vs %s", id1, id2)
	}

	// 3. Tampering detection
	tampered := *plan
	tampered.SourceRev = "rev-002"
	if tampered.VerifyPlanID() {
		t.Fatalf("expected VerifyPlanID to fail on tampered plan")
	}

	// 4. Save and Load
	planFilePath := filepath.Join(tempDir, "plan.json")
	if err := SavePlan(plan, planFilePath); err != nil {
		t.Fatalf("SavePlan failed: %v", err)
	}
	info, err := os.Stat(planFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("SavePlan file mode = %o, want 0600", info.Mode().Perm())
	}
	loaded, err := LoadPlan(planFilePath)
	if err != nil {
		t.Fatalf("LoadPlan failed: %v", err)
	}
	if loaded.PlanID != plan.PlanID {
		t.Fatalf("loaded plan ID %q != original %q", loaded.PlanID, plan.PlanID)
	}

	// Tampered saved file
	if err := os.WriteFile(planFilePath, []byte(strings.Replace(string(canonical), "rev-001", "rev-tampered", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPlan(planFilePath); err == nil || !errors.Is(err, ErrInvalidPlanStructure) {
		t.Fatalf("expected LoadPlan to fail with ErrInvalidPlanStructure on tampered file, got %v", err)
	}

	// LoadPlan rejects non-regular file (e.g. directory)
	if _, err := LoadPlan(tempDir); err == nil || !errors.Is(err, ErrInvalidPlanStructure) {
		t.Fatalf("expected LoadPlan to fail on directory with ErrInvalidPlanStructure, got %v", err)
	}

	// LoadConfigFile rejects non-regular file (e.g. directory)
	if _, _, _, err := LoadConfigFile(tempDir); err == nil || !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected LoadConfigFile to fail on directory with ErrInvalidConfig, got %v", err)
	}

	// Revalidate require-all: fails if any field is missing
	if err := plan.Revalidate(RevalidationSet{}); err == nil || !errors.Is(err, ErrStalePlan) {
		t.Fatalf("expected Revalidate to fail on empty RevalidationSet with ErrStalePlan, got %v", err)
	}

	// 5. Revalidate: fresh state
	if err := RevalidateAgainstDisk(plan, configBytes, "rev-001", "1.0.0"); err != nil {
		t.Fatalf("expected RevalidateAgainstDisk to succeed, got: %v", err)
	}

	// 6. Stale Plan: target file modified on disk
	if err := os.WriteFile(agentFile, []byte("mid-run modification"), 0644); err != nil {
		t.Fatal(err)
	}
	err = RevalidateAgainstDisk(plan, configBytes, "rev-001", "1.0.0")
	if err == nil || !errors.Is(err, ErrStalePlan) {
		t.Fatalf("expected ErrStalePlan on modified target file, got: %v", err)
	}

	// Restore target file
	if err := os.WriteFile(agentFile, agentContent, 0644); err != nil {
		t.Fatal(err)
	}

	// 7. Stale Plan: config changed
	changedConfig := []byte(`{"schemaVersion":1,"instances":{"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"9router/ag/claude-sonnet-5.5-high"}}}`)
	err = RevalidateAgainstDisk(plan, changedConfig, "rev-001", "1.0.0")
	if err == nil || !errors.Is(err, ErrStalePlan) {
		t.Fatalf("expected ErrStalePlan on changed config, got: %v", err)
	}

	// 8. Stale Plan: source rev changed
	err = RevalidateAgainstDisk(plan, configBytes, "rev-different", "1.0.0")
	if err == nil || !errors.Is(err, ErrStalePlan) {
		t.Fatalf("expected ErrStalePlan on changed source rev, got: %v", err)
	}

	// 9. Path helpers
	stamp := "20261111-120000"
	expectedManifest := filepath.Join(tempDir, "backups", stamp, "fleet-manifest.json")
	if got := ManifestPath(filepath.Join(tempDir, "backups"), stamp); got != expectedManifest {
		t.Fatalf("ManifestPath got %q, want %q", got, expectedManifest)
	}
	expectedPointer := filepath.Join(tempDir, "home", ".agentic-sdd", "fleet", "current.json")
	if got := LivePointerPath(filepath.Join(tempDir, "home")); got != expectedPointer {
		t.Fatalf("LivePointerPath got %q, want %q", got, expectedPointer)
	}
}

func TestFleetLockUnit(t *testing.T) {
	tempDir := t.TempDir()
	backupRoot := filepath.Join(tempDir, "backups")

	// 1. Acquire lock
	lock1, err := AcquireLock(backupRoot)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	// 2. Concurrent acquire fails
	lock2, err := AcquireLock(backupRoot)
	if err == nil {
		t.Fatalf("expected concurrent AcquireLock to fail")
	}
	if !errors.Is(err, ErrLockHeld) {
		t.Fatalf("expected ErrLockHeld, got: %v", err)
	}
	if lock2 != nil {
		_ = lock2.Release()
	}

	// 3. Release and re-acquire
	if err := lock1.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	lock3, err := AcquireLock(backupRoot)
	if err != nil {
		t.Fatalf("AcquireLock after release failed: %v", err)
	}
	if err := lock3.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// 4. Stale lock recovery (>30s)
	lockFile := filepath.Join(backupRoot, LockFileName)
	staleTime := time.Now().Add(-45 * time.Second).UTC()
	staleContent := fmt.Sprintf("999999\n%s\n", staleTime.Format(time.RFC3339Nano))
	if err := os.WriteFile(lockFile, []byte(staleContent), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(lockFile, staleTime, staleTime)

	staleLock, err := AcquireLock(backupRoot)
	if err != nil {
		t.Fatalf("AcquireLock on stale lock failed: %v", err)
	}
	if err := staleLock.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}
}

func TestFleetPlanReservedAndRouteWarnings(t *testing.T) {
	tempDir := t.TempDir()
	home := filepath.Join(tempDir, "home")
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(agentsDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create an installed agent file with an older model
	existingAgent := `---
mode: subagent
model: 9router/ag/gemini-3.8-flash-medium
---

agent body
`
	if err := os.WriteFile(filepath.Join(agentsDir, "sdd-implement-1.md"), []byte(existingAgent), 0600); err != nil {
		t.Fatal(err)
	}

	// Config with a reserved name ("vision") and a route change for sdd-implement-1
	configJSON := `{
  "schemaVersion": 1,
  "instances": {
    "vision": {
      "source": "agent:sdd-visual-inspector",
      "model": "9router/ag/gemini-3.8-flash-high"
    },
    "sdd-implement-1": {
      "source": "preset:sdd-implement-1",
      "model": "9router/ag/gemini-3.8-flash-high"
    }
  }
}`
	configPath := filepath.Join(tempDir, "fleet.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(configPath, "", home, false)
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}

	hasReservedWarning := false
	hasRouteChangeWarning := false
	for _, note := range plan.PermNotes {
		if strings.Contains(note, "warning: instance \"vision\" uses reserved SDD name") {
			hasReservedWarning = true
		}
		if strings.Contains(note, "warning: instance sdd-implement-1 model route changed") {
			hasRouteChangeWarning = true
		}
	}
	if !hasReservedWarning {
		t.Fatalf("expected reserved name warning in PermNotes, got: %v", plan.PermNotes)
	}
	if !hasRouteChangeWarning {
		t.Fatalf("expected route change warning in PermNotes, got: %v", plan.PermNotes)
	}
}
