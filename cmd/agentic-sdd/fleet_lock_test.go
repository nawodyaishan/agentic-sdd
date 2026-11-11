package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

func TestFleetLock(t *testing.T) {
	tempDir := t.TempDir()
	home := filepath.Join(tempDir, "home")
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial agent file
	agentFile := filepath.Join(agentsDir, "sdd-implement-1.md")
	agentBytes := []byte("original content")
	if err := os.WriteFile(agentFile, agentBytes, 0644); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tempDir, "fleet.json")
	configJSON := `{
  "schemaVersion": 1,
  "instances": {
    "sdd-implement-1": {
      "source": "preset:sdd-implement-1",
      "model": "9router/ag/gemini-3.8-flash-high"
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	planPath := filepath.Join(tempDir, "plan.json")
	var planOut, planErr bytes.Buffer
	code := runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--adopt", "--home", home}, &planOut, &planErr)
	if code != 0 {
		t.Fatalf("fleet plan failed (%d): %s", code, planErr.String())
	}

	backupRoot := skillsync.BackupRootFor("", home)
	if err := os.MkdirAll(backupRoot, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Concurrent lock test
	lock1, err := fleet.AcquireLock(backupRoot)
	if err != nil {
		t.Fatalf("failed to acquire initial lock: %v", err)
	}

	var applyOut, applyErr bytes.Buffer
	code = runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &applyOut, &applyErr)
	if code != 1 {
		t.Fatalf("expected exit code 1 for lock held, got %d; stdout: %s, stderr: %s", code, applyOut.String(), applyErr.String())
	}
	if !strings.Contains(applyErr.String(), "fleet lock is held") && !strings.Contains(applyErr.String(), "lock held") {
		t.Fatalf("expected lock held error message, got: %s", applyErr.String())
	}

	// Release lock and verify apply succeeds
	if err := lock1.Release(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	var applyOut2, applyErr2 bytes.Buffer
	code = runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &applyOut2, &applyErr2)
	if code != 0 {
		t.Fatalf("expected apply to succeed after lock release, got %d; stderr: %s", code, applyErr2.String())
	}

	// 2. Stale lock recovery test (>30s). Regenerate the plan first: the
	// successful apply above changed the target, so the original plan is
	// correctly stale and this step must exercise lock recovery only.
	var regenOut, regenErr bytes.Buffer
	code = runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--adopt", "--home", home}, &regenOut, &regenErr)
	if code != 0 {
		t.Fatalf("fleet plan regeneration failed (%d): %s", code, regenErr.String())
	}
	lockFile := filepath.Join(backupRoot, fleet.LockFileName)
	staleTime := time.Now().Add(-45 * time.Second).UTC()
	staleContent := fmt.Sprintf("999999\n%s\n", staleTime.Format(time.RFC3339Nano))
	if err := os.WriteFile(lockFile, []byte(staleContent), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(lockFile, staleTime, staleTime)

	var applyOut3, applyErr3 bytes.Buffer
	code = runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &applyOut3, &applyErr3)
	if code != 0 {
		t.Fatalf("expected apply to reclaim stale lock and succeed, got %d; stderr: %s", code, applyErr3.String())
	}

	// 3. Stale plan detection: target file modified after plan
	if err := os.WriteFile(agentFile, []byte("externally modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	var applyOut4, applyErr4 bytes.Buffer
	code = runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &applyOut4, &applyErr4)
	if code != 1 {
		t.Fatalf("expected exit code 1 for stale plan, got %d; stdout: %s, stderr: %s", code, applyOut4.String(), applyErr4.String())
	}
	if !strings.Contains(applyErr4.String(), "stale plan") {
		t.Fatalf("expected stale plan error in stderr, got: %s", applyErr4.String())
	}
}
