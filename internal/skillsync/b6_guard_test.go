//go:build linux || darwin

package skillsync

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// B6 (T12) guard coverage. The holder is a real child process that holds one
// kernel-held guard (see startB1MutationHolder), so the exclusion is observed
// across processes, not only across descriptors in this test process.

// b6AssertNoInstallerWrites fails if the installer left any agent profile,
// skill or backup behind in a fresh home.
func b6AssertNoInstallerWrites(t *testing.T, home string) {
	t.Helper()
	for _, rel := range []string{
		filepath.Join(".config", "opencode", "agents"),
		filepath.Join(".agents", "skills"),
		filepath.Join(".agentic-sdd", "backups"),
	} {
		if _, err := os.Lstat(filepath.Join(home, rel)); !os.IsNotExist(err) {
			t.Fatalf("refused apply still wrote %s: %v", rel, err)
		}
	}
}

// TestB6TwoProcessInstallerExcludedByAgentTargetHolder: while another process
// holds the agent-target guard, the installer refuses before any write.
func TestB6TwoProcessInstallerExcludedByAgentTargetHolder(t *testing.T) {
	home := b1CanonicalHome(t)
	holder := startB1MutationHolder(t, home, DomainAgentTarget)
	t.Cleanup(func() { holder.stop(t) })

	_, err := SyncWithOptionsResult("", home, true, Options{WithOpenCode: true}, io.Discard)
	if !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("installer apply error = %v, want ErrMutationConflict while another process holds the agent target", err)
	}
	b6AssertNoInstallerWrites(t, home)
}

// TestB6HolderBlocksFleetStyleAgentFileApply: the same agent-target domain
// excludes the fleet's ApplyAgentFiles writer from another process.
func TestB6HolderBlocksFleetStyleAgentFileApply(t *testing.T) {
	home := b1CanonicalHome(t)
	holder := startB1MutationHolder(t, home, DomainAgentTarget)
	t.Cleanup(func() { holder.stop(t) })

	changes := []AgentFileChange{{File: "sdd-b6.md", Data: []byte("profile\n")}}
	_, err := ApplyAgentFiles(filepath.Join(home, ".agentic-sdd", "backups"), home, "fleet", changes, nil, io.Discard)
	if !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("ApplyAgentFiles error = %v, want ErrMutationConflict", err)
	}
	if _, statErr := os.Lstat(filepath.Join(home, ".config", "opencode", "agents", "sdd-b6.md")); !os.IsNotExist(statErr) {
		t.Fatalf("refused fleet apply wrote the profile: %v", statErr)
	}
}

// TestB6MCPDomainIsIndependentOfAgentTarget: MCP configuration writes keep
// their own domain, so an agent-profile holder does not block them.
func TestB6MCPDomainIsIndependentOfAgentTarget(t *testing.T) {
	home := b1CanonicalHome(t)
	holder := startB1MutationHolder(t, home, DomainAgentTarget)
	t.Cleanup(func() { holder.stop(t) })

	mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	plan := mcpTestBuildAll(t, home)
	bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)
	if _, err := mcpTestApplyPlan(t, home, "", plan, bin); err != nil {
		t.Fatalf("MCP apply blocked by an agent-profile holder: %v", err)
	}
}

// TestB6MCPHolderBlocksApplyAndAgeDoesNotEvict: a live MCP-domain holder blocks
// the MCP writer, and back-dating the leaf does not evict it.
func TestB6MCPHolderBlocksApplyAndAgeDoesNotEvict(t *testing.T) {
	home := b1CanonicalHome(t)
	holder := startB1MutationHolder(t, home, DomainMCPTarget)
	t.Cleanup(func() { holder.stop(t) })

	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)
	plan := mcpTestBuildAll(t, home)
	bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)

	if _, err := mcpTestApplyPlan(t, home, "", plan, bin); !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("apply with a live MCP holder error = %v, want ErrMCPConflict", err)
	}
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(b1LockLeaf(home, DomainMCPTarget), old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := mcpTestApplyPlan(t, home, "", plan, bin); !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("apply after back-dating the live holder error = %v, want ErrMCPConflict", err)
	}
	assertFileBytesUnchanged(t, path, original)
	if backups := mcpTestBackupIDs(t, home); len(backups) != 0 {
		t.Fatalf("refused MCP apply created backups: %v", backups)
	}
}

// TestB6LocksDirectorySurvivesInstallAndRestore: install, replace and restore
// never delete <home>/.agentic-sdd/locks or the guard leaf inside it.
func TestB6LocksDirectorySurvivesInstallAndRestore(t *testing.T) {
	home := t.TempDir()
	names := b6Names(t)
	id, _ := b6ReplacedBackup(t, home, names[0])
	if _, err := RestoreResult("", home, id, true, io.Discard); err != nil {
		t.Fatalf("restore: %v", err)
	}
	leaf := b1LockLeaf(home, DomainAgentTarget)
	info, err := os.Lstat(leaf)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("agent-target guard leaf after install and restore = %v, %v; want a retained regular file", info, err)
	}
}
