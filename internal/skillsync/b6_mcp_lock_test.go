package skillsync

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// b6LegacyLockFixture writes a legacy lock older than the 30 s window and
// returns its path and the modification time it was given.
func b6LegacyLockFixture(t *testing.T, home string) (string, time.Time) {
	t.Helper()
	legacy := filepath.Join(home, ".agentic-sdd", "mcp", ".lock")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("999\nlegacy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Minute).Truncate(time.Second)
	if err := os.Chtimes(legacy, old, old); err != nil {
		t.Fatal(err)
	}
	return legacy, old
}

// TestB6MCPLegacyLockOldIsRemovedOnlyWhenIdentityMatches: an old legacy lock is
// removed when the object inspected is still the one at the path, and kept
// when it was replaced or re-timed between inspection and removal.
func TestB6MCPLegacyLockOldIsRemovedOnlyWhenIdentityMatches(t *testing.T) {
	t.Run("matching identity is removed", func(t *testing.T) {
		home := t.TempDir()
		path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		legacy, _ := b6LegacyLockFixture(t, home)
		plan := mcpTestBuildAll(t, home)
		bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)
		if _, err := mcpTestApplyPlan(t, home, "", plan, bin); err != nil {
			t.Fatalf("apply over an old legacy lock: %v", err)
		}
		if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
			t.Fatalf("old legacy lock with matching identity was kept: %v", err)
		}
		if got := mcpReadTestFile(t, path); !bytes.Equal(got, plan.TargetBytes()) {
			t.Fatal("apply did not write the planned bytes")
		}
	})

	t.Run("replaced object is kept", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		legacy, old := b6LegacyLockFixture(t, home)
		plan := mcpTestBuildAll(t, home)
		bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)
		b6SetRecheckHook(t, func(path string) {
			_ = os.Remove(path)
			_ = os.WriteFile(path, []byte("888\nsuccessor\n"), 0o600)
			_ = os.Chtimes(path, old, old)
		})
		_, err := mcpTestApplyPlan(t, home, "", plan, bin)
		if !errors.Is(err, ErrMCPConflict) {
			t.Fatalf("apply over a replaced legacy lock error = %v, want ErrMCPConflict", err)
		}
		if got := mcpReadTestFile(t, legacy); string(got) != "888\nsuccessor\n" {
			t.Fatal("replaced legacy lock was removed")
		}
	})

	t.Run("re-timed object is kept", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		legacy, old := b6LegacyLockFixture(t, home)
		plan := mcpTestBuildAll(t, home)
		bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)
		later := old.Add(time.Minute)
		b6SetRecheckHook(t, func(path string) {
			_ = os.Chtimes(path, later, later)
		})
		_, err := mcpTestApplyPlan(t, home, "", plan, bin)
		if !errors.Is(err, ErrMCPConflict) {
			t.Fatalf("apply over a re-timed legacy lock error = %v, want ErrMCPConflict", err)
		}
		if _, statErr := os.Lstat(legacy); statErr != nil {
			t.Fatalf("re-timed legacy lock was removed: %v", statErr)
		}
	})
}

// b6SetRecheckHook installs mcpLegacyLockRecheckHook for one test.
func b6SetRecheckHook(t *testing.T, hook func(path string)) {
	t.Helper()
	prev := mcpLegacyLockRecheckHook
	mcpLegacyLockRecheckHook = hook
	t.Cleanup(func() { mcpLegacyLockRecheckHook = prev })
}

// B6 (T12) MCP lock coverage. The legacy artifact is the per-state-directory
// `.lock` that older binaries created under <home>/.agentic-sdd/mcp. The new
// exclusion is the B1 guard in <home>/.agentic-sdd/locks.

// TestB6MCPYoungLegacyLockIsConflictAndUntouched: a legacy lock younger than
// the 30 s window belongs to a live writer. It is refused and left in place.
func TestB6MCPYoungLegacyLockIsConflictAndUntouched(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)
	plan := mcpTestBuildAll(t, home)
	bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)

	legacy := filepath.Join(home, ".agentic-sdd", "mcp", ".lock")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	content := []byte("999\nlegacy\n")
	if err := os.WriteFile(legacy, content, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := mcpTestApplyPlan(t, home, "", plan, bin)
	if !errors.Is(err, ErrMCPConflict) || !errors.Is(err, ErrMCPLocked) {
		t.Fatalf("apply with a young legacy lock error = %v, want ErrMCPConflict and ErrMCPLocked", err)
	}
	assertFileBytesUnchanged(t, path, original)
	if got := mcpReadTestFile(t, legacy); !bytes.Equal(got, content) {
		t.Fatal("young legacy lock was modified")
	}
	if backups := mcpTestBackupIDs(t, home); len(backups) != 0 {
		t.Fatalf("young-legacy refusal created backups: %v", backups)
	}
}
