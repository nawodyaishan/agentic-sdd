package skillsync

// MCP-B transactional engine tests. Every fixture lives under t.TempDir(); no
// test reads or writes a real home, contacts the network, installs a package or
// executes a real opencode binary. The only executable is a shell script the
// test writes that prints one version string.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	mcpTestSupportedVersion   = "2.3.4"
	mcpTestUnsupportedVersion = "1.18.2"
)

// mcpTestFilePerm returns the permission bits of one path.
func mcpTestFilePerm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("inspect %s: %v", path, err)
	}
	return info.Mode().Perm()
}

// mcpTestBackupIDs lists the private MCP backup directory names, sorted.
func mcpTestBackupIDs(t *testing.T, home string) []string {
	t.Helper()
	root := filepath.Join(home, ".agentic-sdd", "mcp", "backups")
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		t.Fatalf("read backup root %s: %v", root, err)
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	return ids
}

// mcpTestOpenCodeScript writes a fake opencode executable with a custom body.
func mcpTestOpenCodeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "opencode")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

// mcpTestApplyPlan runs ApplyMCPPlan with the plan's own base digest.
func mcpTestApplyPlan(t *testing.T, home, configPath string, plan *MCPPlan, bin string) (MCPApplyResult, error) {
	t.Helper()
	return ApplyMCPPlan(home, configPath, plan, MCPApplyOptions{
		Home:         home,
		ExpectDigest: plan.BaseDigest,
		OpenCodeBin:  bin,
	})
}

// mcpTestApplyAll writes mcpTestBaseConfig, plans all three managed servers and
// applies them with a fake V2 runtime. It returns the config path, plan and
// result of a real, successful transaction.
func mcpTestApplyAll(t *testing.T, home string) (string, *MCPPlan, MCPApplyResult) {
	t.Helper()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	plan := mcpTestBuildAll(t, home)
	result, err := mcpTestApplyPlan(t, home, "", plan, mcpFakeOpenCode(t, "opencode "+mcpTestSupportedVersion))
	if err != nil {
		t.Fatalf("ApplyMCPPlan: %v", err)
	}
	return path, plan, result
}

// mcpTestWriteSnapshot writes one raw snapshot.json fixture under backupID.
func mcpTestWriteSnapshot(t *testing.T, home, backupID string, snapshot mcpSnapshot) string {
	t.Helper()
	dir := filepath.Join(home, ".agentic-sdd", "mcp", "backups", backupID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMCPApplyAddWritesExactBytesBackupAndManifest(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)
	bin := mcpFakeOpenCode(t, "opencode "+mcpTestSupportedVersion)
	plan := mcpTestBuildAll(t, home)
	if plan.NoOp {
		t.Fatal("add plan reported a no-op")
	}
	if plan.BaseDigest != mcpBytesDigest(original) {
		t.Fatal("plan base digest does not describe the current target")
	}

	result, err := mcpTestApplyPlan(t, home, "", plan, bin)
	if err != nil {
		t.Fatalf("ApplyMCPPlan: %v", err)
	}
	if result.NoOp {
		t.Fatal("real write reported a no-op")
	}
	if result.TargetPath != path {
		t.Fatalf("TargetPath = %q, want %q", result.TargetPath, path)
	}
	if got := strings.Join(result.Applied, ","); got != "codegraph,context7,exa" {
		t.Fatalf("Applied = %q, want the three managed names", got)
	}
	if result.RuntimeVersion != mcpTestSupportedVersion {
		t.Fatalf("RuntimeVersion = %q, want %q", result.RuntimeVersion, mcpTestSupportedVersion)
	}
	wantBackupPath := filepath.Join(home, ".agentic-sdd", "mcp", "backups", result.BackupID)
	if result.BackupID == "" || result.BackupPath != wantBackupPath {
		t.Fatalf("backup identity = %q at %q", result.BackupID, result.BackupPath)
	}

	// The target carries exactly the planned bytes and keeps its mode.
	if got := mcpReadTestFile(t, path); !bytes.Equal(got, plan.TargetBytes()) {
		t.Fatalf("target bytes differ from the plan:\ngot:\n%s\nwant:\n%s", got, plan.TargetBytes())
	}
	if got := mcpTestFilePerm(t, path); got != 0o644 {
		t.Fatalf("target mode = %v, want 0644", got)
	}
	if result.InstalledDigest != mcpBytesDigest(plan.TargetBytes()) {
		t.Fatal("InstalledDigest does not describe the bytes that were written")
	}

	// Private backup directory and files.
	if got := mcpTestFilePerm(t, result.BackupPath); got != 0o700 {
		t.Fatalf("backup dir mode = %v, want 0700", got)
	}
	snapshotPath := filepath.Join(result.BackupPath, "snapshot.json")
	intentPath := filepath.Join(result.BackupPath, "intent.json")
	if got := mcpTestFilePerm(t, snapshotPath); got != 0o600 {
		t.Fatalf("snapshot mode = %v, want 0600", got)
	}
	if got := mcpTestFilePerm(t, intentPath); got != 0o600 {
		t.Fatalf("intent mode = %v, want 0600", got)
	}

	snapshot, err := mcpReadSnapshot(result.BackupPath, result.BackupID)
	if err != nil {
		t.Fatalf("mcpReadSnapshot: %v", err)
	}
	if snapshot.Kind != mcpSnapshotKindApply || snapshot.Target != path {
		t.Fatalf("snapshot kind/target = %q/%q", snapshot.Kind, snapshot.Target)
	}
	if snapshot.BaseDigest != plan.BaseDigest || snapshot.InstalledDigest != result.InstalledDigest {
		t.Fatal("snapshot digests do not describe the transaction")
	}
	if snapshot.PlanDigest != plan.PlanDigest {
		t.Fatal("snapshot plan digest does not match the applied plan")
	}
	if snapshot.TargetMode != uint32(0o644) {
		t.Fatalf("snapshot target mode = %v, want 0644", os.FileMode(snapshot.TargetMode))
	}
	if snapshot.RuntimeVersion != mcpTestSupportedVersion {
		t.Fatalf("snapshot runtime version = %q", snapshot.RuntimeVersion)
	}
	if snapshot.RuntimeEvidence != mcpRuntimeEvidenceLocal+":"+bin {
		t.Fatalf("snapshot runtime evidence = %q", snapshot.RuntimeEvidence)
	}
	if snapshot.CreatedMCP || snapshot.CreatedServers {
		t.Fatal("snapshot claims it created containers that already existed")
	}
	if len(snapshot.Entries) != 3 {
		t.Fatalf("snapshot entries = %v, want three adds", snapshot.Entries)
	}
	wantHashes := map[string]string{}
	for _, entry := range plan.Entries {
		wantHashes[entry.Name] = entry.NewHash
	}
	for _, entry := range snapshot.Entries {
		if entry.Op != mcpOpAdd || entry.OldText != "" || entry.NewText == "" {
			t.Fatalf("snapshot entry %q = %+v, want an add with new text only", entry.Name, entry)
		}
		if entry.NewHash != wantHashes[entry.Name] {
			t.Fatalf("snapshot entry %q hash does not match the plan", entry.Name)
		}
	}

	// The backup must not copy unrelated servers or unrelated top-level content.
	snapshotBytes := mcpReadTestFile(t, snapshotPath)
	for _, unwanted := range []string{"example.test", "extra_setting", "$schema"} {
		if bytes.Contains(snapshotBytes, []byte(unwanted)) {
			t.Fatalf("snapshot copied unrelated content %q:\n%s", unwanted, snapshotBytes)
		}
	}

	// Ownership manifest: format 1, private, no credential or entry text.
	manifestPath := filepath.Join(home, ".agentic-sdd", "mcp", "manifest.json")
	if got := mcpTestFilePerm(t, manifestPath); got != 0o600 {
		t.Fatalf("manifest mode = %v, want 0600", got)
	}
	manifest, err := readMCPOwnershipManifest(home)
	if err != nil {
		t.Fatalf("readMCPOwnershipManifest: %v", err)
	}
	if manifest == nil {
		t.Fatal("apply wrote no ownership manifest")
	}
	if manifest.Format != mcpManifestFormat || manifest.ManifestVersion != mcpManifestVersion {
		t.Fatalf("manifest format/version = %d/%d", manifest.Format, manifest.ManifestVersion)
	}
	if manifest.Target != path || manifest.TargetFormat != plan.Format || manifest.Schema != plan.Schema {
		t.Fatalf("manifest target/format/schema = %q/%q/%q", manifest.Target, manifest.TargetFormat, manifest.Schema)
	}
	if manifest.BaseDigest != plan.BaseDigest || manifest.InstalledDigest != result.InstalledDigest {
		t.Fatal("manifest digests do not describe the applied transaction")
	}
	if manifest.BackupID != result.BackupID || manifest.BackupPath != result.BackupPath {
		t.Fatal("manifest backup identity does not match the result")
	}
	if manifest.RuntimeVersion != mcpTestSupportedVersion {
		t.Fatalf("manifest runtime version = %q", manifest.RuntimeVersion)
	}
	if len(manifest.Entries) != 3 {
		t.Fatalf("manifest entries = %v, want three managed names", manifest.Entries)
	}
	for _, entry := range manifest.Entries {
		if entry.InstalledDigest != wantHashes[entry.Name] {
			t.Fatalf("manifest entry %q digest does not match the plan", entry.Name)
		}
	}
	manifestBytes := mcpReadTestFile(t, manifestPath)
	for _, unwanted := range []string{"example.test", "extra_setting", "mcp.context7.com"} {
		if bytes.Contains(manifestBytes, []byte(unwanted)) {
			t.Fatalf("manifest copied configuration content %q:\n%s", unwanted, manifestBytes)
		}
	}

	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentComplete {
		t.Fatalf("intent phase = %q, want complete", intent.Phase)
	}
	if intent.TargetPath != path || intent.BaseDigest != plan.BaseDigest || intent.PlanDigest != plan.PlanDigest {
		t.Fatalf("intent evidence = %+v", intent)
	}
	if intent.PriorManifestDigest != "" {
		t.Fatalf("first apply recorded a prior manifest digest %q", intent.PriorManifestDigest)
	}
}

func TestMCPApplyPreservesTargetMode(t *testing.T) {
	home := t.TempDir()
	dir := mcpTestConfigDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(path, []byte(mcpTestBaseConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := mcpTestBuildAll(t, home)
	result, err := mcpTestApplyPlan(t, home, "", plan, mcpFakeOpenCode(t, mcpTestSupportedVersion))
	if err != nil {
		t.Fatalf("ApplyMCPPlan: %v", err)
	}
	if got := mcpTestFilePerm(t, path); got != 0o600 {
		t.Fatalf("target mode = %v, want the original 0600", got)
	}
	snapshot, err := mcpReadSnapshot(result.BackupPath, result.BackupID)
	if err != nil {
		t.Fatalf("mcpReadSnapshot: %v", err)
	}
	if snapshot.TargetMode != uint32(0o600) {
		t.Fatalf("snapshot target mode = %v, want 0600", os.FileMode(snapshot.TargetMode))
	}
}

func TestMCPApplyChangeManagedEntry(t *testing.T) {
	home := t.TempDir()
	oldEntry := map[string]any{
		"type":     "local",
		"command":  []any{"codegraph", "serve"},
		"disabled": false,
		"codemode": false,
	}
	oldText, oldHash, err := mcpCanonicalEntry(oldEntry)
	if err != nil {
		t.Fatal(err)
	}
	content := "{\n  \"theme\": \"dark\",\n  \"mcp\": {\n    \"servers\": {\n      \"codegraph\": " + oldText +
		",\n      \"unrelated\": {\"type\": \"remote\", \"url\": \"https://example.test/mcp\", \"custom_field\": \"kept\"}\n    }\n  }\n}\n"
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	mcpWriteTestManifest(t, home, mcpTestManifest(path, map[string]string{"codegraph": oldHash}))

	plan, err := BuildMCPPlan(home, "", []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if len(plan.Entries) != 1 || plan.Entries[0].Op != mcpOpChange {
		t.Fatalf("entries = %+v, want one managed change", plan.Entries)
	}
	result, err := mcpTestApplyPlan(t, home, "", plan, mcpFakeOpenCode(t, mcpTestSupportedVersion))
	if err != nil {
		t.Fatalf("ApplyMCPPlan: %v", err)
	}
	if result.InstalledDigest != mcpBytesDigest(plan.TargetBytes()) {
		t.Fatal("change did not record the installed digest")
	}

	codegraph := mcpTestEntry(t, mcpReadTestFile(t, path), "codegraph")
	command, _ := codegraph["command"].([]any)
	if len(command) != 3 || command[2] != "--mcp" {
		t.Fatalf("managed entry was not updated: %v", codegraph)
	}
	servers := mcpTestServers(t, mcpReadTestFile(t, path))
	unrelated, _ := servers["unrelated"].(map[string]any)
	if unrelated["custom_field"] != "kept" {
		t.Fatalf("unrelated server was changed: %v", unrelated)
	}
	root := mcpDecodeJSON(t, mcpReadTestFile(t, path))
	if root["theme"] != "dark" {
		t.Fatalf("unrelated top-level field was changed: %v", root)
	}

	snapshot, err := mcpReadSnapshot(result.BackupPath, result.BackupID)
	if err != nil {
		t.Fatalf("mcpReadSnapshot: %v", err)
	}
	if len(snapshot.Entries) != 1 {
		t.Fatalf("snapshot entries = %v, want one change", snapshot.Entries)
	}
	entry := snapshot.Entries[0]
	if entry.Name != "codegraph" || entry.Op != mcpOpChange {
		t.Fatalf("snapshot entry = %+v", entry)
	}
	if entry.OldText != oldText || entry.OldHash != oldHash {
		t.Fatal("snapshot did not save the original managed-entry text and digest")
	}
	if entry.NewHash != plan.Entries[0].NewHash {
		t.Fatal("snapshot new digest does not match the plan")
	}

	manifest, err := readMCPOwnershipManifest(home)
	if err != nil {
		t.Fatalf("readMCPOwnershipManifest: %v", err)
	}
	if manifest == nil || len(manifest.Entries) != 1 {
		t.Fatalf("manifest = %+v, want one owned entry", manifest)
	}
	if manifest.Entries[0].Name != "codegraph" || manifest.Entries[0].InstalledDigest != entry.NewHash {
		t.Fatalf("manifest entry = %+v", manifest.Entries[0])
	}
}

func TestMCPApplyNoOpCreatesNoSecondBackup(t *testing.T) {
	home := t.TempDir()
	path, plan, first := mcpTestApplyAll(t, home)
	backupsBefore := mcpTestBackupIDs(t, home)
	if len(backupsBefore) != 1 {
		t.Fatalf("backups after one apply = %v, want one", backupsBefore)
	}

	again, err := BuildMCPPlan(home, "", nil, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if !again.NoOp {
		t.Fatalf("identical selection planned a write: %+v", again.Entries)
	}
	if again.TargetBytes() != nil {
		t.Fatal("no-op plan carried target bytes")
	}
	result, err := mcpTestApplyPlan(t, home, "", again, mcpFakeOpenCode(t, mcpTestSupportedVersion))
	if err != nil {
		t.Fatalf("no-op apply: %v", err)
	}
	if !result.NoOp {
		t.Fatal("second identical apply did not report a no-op")
	}
	if result.BackupID != "" || result.BackupPath != "" || result.Applied != nil {
		t.Fatalf("no-op claimed a write: %+v", result)
	}
	if result.InstalledDigest != mcpBytesDigest(plan.TargetBytes()) {
		t.Fatal("no-op installed digest does not describe the unchanged target")
	}
	if got := mcpReadTestFile(t, path); !bytes.Equal(got, plan.TargetBytes()) {
		t.Fatal("no-op changed the target bytes")
	}
	if backups := mcpTestBackupIDs(t, home); strings.Join(backups, ",") != strings.Join(backupsBefore, ",") {
		t.Fatalf("no-op created another backup: %v -> %v", backupsBefore, backups)
	}
	if _, err := os.Stat(filepath.Join(home, ".agentic-sdd", "mcp", ".lock")); !os.IsNotExist(err) {
		t.Fatalf("no-op left the engine lock behind: %v", err)
	}
	if first.BackupID == "" {
		t.Fatal("first apply recorded no backup")
	}
}

func TestMCPApplyConflictRefusesAndAvoidsWrite(t *testing.T) {
	home := t.TempDir()
	path, _, first := mcpTestApplyAll(t, home)
	backupsBefore := mcpTestBackupIDs(t, home)

	// A later edit to a managed entry is untracked user intent.
	current := mcpReadTestFile(t, path)
	edited := bytes.Replace(current, []byte("https://mcp.exa.ai/mcp"), []byte("https://user.example/mcp"), 1)
	if bytes.Equal(edited, current) {
		t.Fatal("fixture edit did not change the target")
	}
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildMCPPlan(home, "", nil, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if len(plan.Conflicts) == 0 {
		t.Fatalf("user-edited managed entry planned %+v instead of a conflict", plan.Entries)
	}
	result, err := ApplyMCPPlan(home, "", plan, MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: mcpFakeOpenCode(t, mcpTestSupportedVersion),
	})
	if !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("apply error = %v, want ErrMCPConflict", err)
	}
	if result.Applied != nil || result.BackupID != "" {
		t.Fatalf("refused apply claimed a write: %+v", result)
	}
	assertFileBytesUnchanged(t, path, edited)
	if backups := mcpTestBackupIDs(t, home); strings.Join(backups, ",") != strings.Join(backupsBefore, ",") {
		t.Fatalf("conflict created a backup: %v -> %v", backupsBefore, backups)
	}
	if first.BackupID == "" {
		t.Fatal("initial apply recorded no backup")
	}
}

func TestMCPApplyStaleDigestRefusesWithoutWrite(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	plan := mcpTestBuildAll(t, home)
	original := mcpReadTestFile(t, path)
	bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)

	// An external edit after preview invalidates the plan.
	edited := bytes.Replace(original, []byte(`"theme": "dark"`), []byte(`"theme": "light"`), 1)
	if bytes.Equal(edited, original) {
		t.Fatal("fixture edit did not change the target")
	}
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := mcpTestApplyPlan(t, home, "", plan, bin)
	if !errors.Is(err, ErrMCPStale) {
		t.Fatalf("apply error = %v, want ErrMCPStale", err)
	}
	if result.Applied != nil || result.BackupID != "" {
		t.Fatalf("stale apply claimed a write: %+v", result)
	}
	assertFileBytesUnchanged(t, path, edited)
	assertNoMCPStateCreated(t, home)

	// A mismatched or missing expected digest refuses before any read.
	fresh := mcpTestBuildAll(t, home)
	if _, err := ApplyMCPPlan(home, "", fresh, MCPApplyOptions{
		Home: home, ExpectDigest: "deadbeef", OpenCodeBin: bin,
	}); !errors.Is(err, ErrMCPStale) {
		t.Fatalf("mismatched expect digest error = %v, want ErrMCPStale", err)
	}
	if _, err := ApplyMCPPlan(home, "", fresh, MCPApplyOptions{
		Home: home, OpenCodeBin: bin,
	}); !errors.Is(err, ErrMCPUsage) {
		t.Fatalf("empty expect digest error = %v, want ErrMCPUsage", err)
	}
	assertFileBytesUnchanged(t, path, edited)
}

func TestMCPApplyRuntimeProbeFailsClosed(t *testing.T) {
	cases := []struct {
		name       string
		bin        func(t *testing.T) string
		wantDetail string
		slow       bool
	}{
		{
			name:       "unsupported V1 runtime",
			bin:        func(t *testing.T) string { return mcpFakeOpenCode(t, mcpTestUnsupportedVersion) },
			wantDetail: "older than the required V2",
		},
		{
			name:       "missing binary",
			bin:        func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent-opencode") },
			wantDetail: "cannot inspect the OpenCode executable",
		},
		{
			name:       "directory instead of binary",
			bin:        func(t *testing.T) string { return t.TempDir() },
			wantDetail: "is a directory",
		},
		{
			name:       "output without a version",
			bin:        func(t *testing.T) string { return mcpTestOpenCodeScript(t, "echo 'no version here'") },
			wantDetail: "could not determine the OpenCode version",
		},
		{
			name:       "probe exits non-zero",
			bin:        func(t *testing.T) string { return mcpTestOpenCodeScript(t, "echo 'boom' >&2; exit 3") },
			wantDetail: "opencode --version failed",
		},
		{
			name:       "probe times out",
			bin:        func(t *testing.T) string { return mcpTestOpenCodeScript(t, "exec sleep 30") },
			wantDetail: "timed out",
			slow:       true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.slow && testing.Short() {
				t.Skip("timeout probe exceeds the short test budget")
			}
			home := t.TempDir()
			path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
			plan := mcpTestBuildAll(t, home)
			result, err := mcpTestApplyPlan(t, home, "", plan, tc.bin(t))
			if !errors.Is(err, ErrMCPUnsupported) {
				t.Fatalf("apply error = %v, want ErrMCPUnsupported", err)
			}
			if !strings.Contains(err.Error(), tc.wantDetail) {
				t.Fatalf("error = %q, want it to mention %q", err.Error(), tc.wantDetail)
			}
			if result.BackupID != "" || result.RuntimeVersion != "" || result.Applied != nil {
				t.Fatalf("failed probe claimed a transaction: %+v", result)
			}
			assertFileBytesUnchanged(t, path, []byte(mcpTestBaseConfig))
			if _, statErr := os.Stat(filepath.Join(home, ".agentic-sdd", "mcp", "backups")); !os.IsNotExist(statErr) {
				t.Fatalf("failed probe created a backup: %v", statErr)
			}
			if _, statErr := os.Stat(filepath.Join(home, ".agentic-sdd", "mcp", ".lock")); !os.IsNotExist(statErr) {
				t.Fatalf("failed probe left the engine lock behind: %v", statErr)
			}
		})
	}

	t.Run("no opencode on PATH", func(t *testing.T) {
		home := t.TempDir()
		path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		plan := mcpTestBuildAll(t, home)
		t.Setenv("PATH", t.TempDir())
		if _, err := ApplyMCPPlan(home, "", plan, MCPApplyOptions{
			Home: home, ExpectDigest: plan.BaseDigest,
		}); !errors.Is(err, ErrMCPUnsupported) || !strings.Contains(err.Error(), "no local opencode executable") {
			t.Fatalf("apply error = %v, want a missing-executable refusal", err)
		}
		assertFileBytesUnchanged(t, path, []byte(mcpTestBaseConfig))
	})
}

func TestMCPApplyFailureBeforeRenamePreservesTarget(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)
	plan := mcpTestBuildAll(t, home)
	sentinel := errors.New("injected pre-rename failure")

	result, err := applyMCPPlan(home, "", plan, MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: mcpFakeOpenCode(t, mcpTestSupportedVersion),
	}, &mcpApplyHook{BeforeRename: func() error { return sentinel }})
	if err == nil {
		t.Fatal("injected failure was reported as success")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want the injected cause", err)
	}
	for _, want := range []string{"nothing changed", mcpIntentPrepared, result.BackupID} {
		if want != "" && !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to mention %q", err.Error(), want)
		}
	}
	if result.Applied != nil || result.InstalledDigest != "" {
		t.Fatalf("failed apply claimed applied changes: %+v", result)
	}
	if result.Rollback.Attempted {
		t.Fatal("pre-rename failure attempted a rollback")
	}
	assertFileBytesUnchanged(t, path, original)
	if _, statErr := os.Stat(filepath.Join(home, ".agentic-sdd", "mcp", "manifest.json")); !os.IsNotExist(statErr) {
		t.Fatalf("pre-rename failure wrote an ownership manifest: %v", statErr)
	}
	matches, globErr := filepath.Glob(filepath.Join(mcpTestConfigDir(home), ".agentic-sdd-mcp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("pre-rename failure left staged files: %v", matches)
	}
	if result.BackupID == "" {
		t.Fatal("pre-rename failure recorded no backup identity")
	}
	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentPrepared {
		t.Fatalf("intent phase = %q, want prepared", intent.Phase)
	}
}

func TestMCPApplyFailureAfterRenameRollsBackEntries(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)
	plan := mcpTestBuildAll(t, home)
	sentinel := errors.New("injected post-rename failure")

	result, err := applyMCPPlan(home, "", plan, MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: mcpFakeOpenCode(t, mcpTestSupportedVersion),
	}, &mcpApplyHook{AfterRename: func() error { return sentinel }})
	if err == nil {
		t.Fatal("injected failure was reported as success")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want the injected cause", err)
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("error = %q, want an explicit rolled-back report", err.Error())
	}
	if result.Applied != nil {
		t.Fatalf("failed apply claimed applied changes: %v", result.Applied)
	}
	if !result.Rollback.Attempted || !result.Rollback.Complete {
		t.Fatalf("rollback = %+v, want a complete rollback", result.Rollback)
	}
	if result.Rollback.Error != "" {
		t.Fatalf("complete rollback carried an error: %q", result.Rollback.Error)
	}
	if result.InstalledDigest != plan.BaseDigest {
		t.Fatal("rolled-back result does not report the original digest")
	}

	// The original managed entries are absent again and unrelated content is
	// preserved exactly.
	afterBytes := mcpReadTestFile(t, path)
	servers := mcpTestServers(t, afterBytes)
	if len(servers) != 1 {
		t.Fatalf("servers after rollback = %v, want only the unrelated server", servers)
	}
	unrelated, _ := servers["unrelated"].(map[string]any)
	if unrelated["custom_field"] != "kept" || unrelated["url"] != "https://example.test/mcp" {
		t.Fatalf("rollback changed the unrelated server: %v", unrelated)
	}
	root := mcpDecodeJSON(t, afterBytes)
	if root["theme"] != "dark" || root["$schema"] != "https://opencode.ai/schema.json" {
		t.Fatalf("rollback changed unrelated top-level fields: %v", root)
	}
	for _, name := range mcpManagedNames() {
		if _, present := servers[name]; present {
			t.Fatalf("rollback left managed entry %q behind", name)
		}
	}
	if _, statErr := os.Stat(filepath.Join(home, ".agentic-sdd", "mcp", "manifest.json")); !os.IsNotExist(statErr) {
		t.Fatalf("post-rename rollback left an ownership manifest: %v", statErr)
	}

	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentRolledBack {
		t.Fatalf("intent phase = %q, want rolled-back", intent.Phase)
	}
	// The original is evidence, not a filename convention: the snapshot must
	// still describe it.
	snapshot, err := mcpReadSnapshot(result.BackupPath, result.BackupID)
	if err != nil {
		t.Fatalf("mcpReadSnapshot: %v", err)
	}
	if snapshot.BaseDigest != mcpBytesDigest(original) || snapshot.InstalledDigest != mcpBytesDigest(plan.TargetBytes()) {
		t.Fatal("snapshot does not record both sides of the rolled-back transaction")
	}
}

func TestMCPApplyFailureAfterManifestLeavesRecoverableEvidence(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	plan := mcpTestBuildAll(t, home)
	sentinel := errors.New("injected manifest-phase failure")

	result, err := applyMCPPlan(home, "", plan, MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: mcpFakeOpenCode(t, mcpTestSupportedVersion),
	}, &mcpApplyHook{AfterManifest: func() error { return sentinel }})
	if err == nil {
		t.Fatal("injected failure was reported as success")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want the injected cause", err)
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("error = %q, want an explicit rolled-back report", err.Error())
	}
	if result.Applied != nil || !result.Rollback.Attempted || !result.Rollback.Complete {
		t.Fatalf("manifest-phase failure result = %+v", result)
	}

	// The managed entries are back to their original (absent) state while
	// unrelated content stayed intact.
	afterBytes := mcpReadTestFile(t, path)
	servers := mcpTestServers(t, afterBytes)
	if len(servers) != 1 {
		t.Fatalf("servers after rollback = %v, want only the unrelated server", servers)
	}
	root := mcpDecodeJSON(t, afterBytes)
	if root["theme"] != "dark" || fmt.Sprint(root["extra_setting"]) != "42" {
		t.Fatalf("rollback changed unrelated content: %v", root)
	}

	// The manifest written before the failure is durable evidence of the
	// attempted transaction, and a later plan never silently rewrites the
	// rolled-back entries: an absent managed name is a deliberate re-add.
	manifest, err := readMCPOwnershipManifest(home)
	if err != nil {
		t.Fatalf("readMCPOwnershipManifest: %v", err)
	}
	if manifest == nil {
		t.Fatal("manifest-phase failure left no ownership evidence")
	}
	if manifest.BackupID != result.BackupID {
		t.Fatalf("manifest backup = %q, want %q", manifest.BackupID, result.BackupID)
	}
	if manifest.InstalledDigest != mcpBytesDigest(plan.TargetBytes()) {
		t.Fatal("manifest does not record the attempted installed digest")
	}
	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentRolledBack {
		t.Fatalf("intent phase = %q, want rolled-back", intent.Phase)
	}

	next, err := BuildMCPPlan(home, "", nil, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if next.NoOp {
		t.Fatal("plan after a rolled-back manifest write reported no work")
	}
	for _, entry := range next.Entries {
		if entry.Op != mcpOpAdd {
			t.Fatalf("post-failure plan entry %q op = %q, want a deliberate re-add", entry.Name, entry.Op)
		}
		if !strings.Contains(entry.Reason, "previously managed") {
			t.Fatalf("re-add reason = %q, want the prior-ownership explanation", entry.Reason)
		}
	}
}

func TestMCPApplyNeedsRecoveryWhenRollbackCannotVerify(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	plan := mcpTestBuildAll(t, home)
	sentinel := errors.New("injected post-rename failure")
	var mutated []byte

	result, err := applyMCPPlan(home, "", plan, MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: mcpFakeOpenCode(t, mcpTestSupportedVersion),
	}, &mcpApplyHook{AfterRename: func() error {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		mutated = append(append([]byte(nil), data...), '\n')
		if writeErr := os.WriteFile(path, mutated, 0o644); writeErr != nil {
			return writeErr
		}
		return sentinel
	}})
	if !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("error = %v, want ErrMCPConflict", err)
	}
	for _, want := range []string{"needs recovery", result.BackupID, result.BackupPath, "current digest",
		"original error: " + sentinel.Error()} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to mention %q", err.Error(), want)
		}
	}
	if result.Applied != nil {
		t.Fatal("needs-recovery result claimed applied changes")
	}
	if !result.Rollback.Attempted || result.Rollback.Complete || result.Rollback.Error == "" {
		t.Fatalf("rollback = %+v, want an incomplete rollback with its error", result.Rollback)
	}
	if got := mcpReadTestFile(t, path); !bytes.Equal(got, mutated) {
		t.Fatal("needs-recovery path rewrote the target after a failed verification")
	}
	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentNeedsRecovery {
		t.Fatalf("intent phase = %q, want needs-recovery", intent.Phase)
	}
}

func TestMCPApplyLockRefusesCompetingWriter(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)
	plan := mcpTestBuildAll(t, home)
	bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)

	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := applyMCPPlan(home, "", plan, MCPApplyOptions{
			Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: bin,
		}, &mcpApplyHook{BeforeRename: func() error {
			close(entered)
			<-release
			return nil
		}})
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the first writer never reached the locked mutation")
	}

	// A competing writer must refuse without touching the target or adding a
	// backup, because the shared lock covers the target and manifest.
	result, err := ApplyMCPPlan(home, "", plan.Clone(), MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: bin,
	})
	if !errors.Is(err, ErrMCPLocked) || !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("competing apply error = %v, want ErrMCPLocked and ErrMCPConflict", err)
	}
	if !strings.Contains(err.Error(), "wait for the competing writer") {
		t.Fatalf("competing apply error = %q, want the retry instruction", err.Error())
	}
	if result.Applied != nil || result.BackupID != "" {
		t.Fatalf("competing apply claimed a write: %+v", result)
	}
	assertFileBytesUnchanged(t, path, original)
	if backups := mcpTestBackupIDs(t, home); len(backups) != 1 {
		t.Fatalf("competing refusal created backups: %v", backups)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if got := mcpReadTestFile(t, path); !bytes.Equal(got, plan.TargetBytes()) {
		t.Fatal("winning apply did not write the planned bytes")
	}
	if backups := mcpTestBackupIDs(t, home); len(backups) != 1 {
		t.Fatalf("backups after the winning apply = %v, want one", backups)
	}

	// The competing request only succeeds as a no-op, after a fresh preview.
	next := mcpTestBuildAll(t, home)
	if !next.NoOp {
		t.Fatalf("plan after the winning apply = %+v, want a no-op", next.Entries)
	}
	noop, err := mcpTestApplyPlan(t, home, "", next, bin)
	if err != nil {
		t.Fatalf("retried apply: %v", err)
	}
	if !noop.NoOp || noop.BackupID != "" {
		t.Fatalf("retried apply = %+v, want a no-op", noop)
	}
}

func TestMCPLockStaleTakeoverAndFreshRefusal(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)
	plan := mcpTestBuildAll(t, home)
	bin := mcpFakeOpenCode(t, mcpTestSupportedVersion)

	stateDir := filepath.Join(home, ".agentic-sdd", "mcp")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(stateDir, ".lock")
	if err := os.WriteFile(lockPath, []byte("999\nheld\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// A fresh lock belongs to a live writer.
	if _, err := mcpTestApplyPlan(t, home, "", plan, bin); !errors.Is(err, ErrMCPLocked) {
		t.Fatalf("apply with a fresh lock error = %v, want ErrMCPLocked", err)
	}
	assertFileBytesUnchanged(t, path, original)
	if backups := mcpTestBackupIDs(t, home); len(backups) != 0 {
		t.Fatalf("fresh-lock refusal created backups: %v", backups)
	}

	// An abandoned lock older than the stale window is reclaimed once.
	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := mcpTestApplyPlan(t, home, "", plan, bin); err != nil {
		t.Fatalf("apply after a stale lock: %v", err)
	}
	if got := mcpReadTestFile(t, path); !bytes.Equal(got, plan.TargetBytes()) {
		t.Fatal("apply after a stale lock did not write the planned bytes")
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("released lock still exists: %v", err)
	}
}

func TestMCPRestorePreviewAndApplyPreserveUnrelated(t *testing.T) {
	home := t.TempDir()
	path, plan, applied := mcpTestApplyAll(t, home)
	current := mcpReadTestFile(t, path)

	preview, err := PreviewMCPRestore(home, applied.BackupID, "")
	if err != nil {
		t.Fatalf("PreviewMCPRestore: %v", err)
	}
	if preview.BackupID != applied.BackupID || preview.TargetPath != path {
		t.Fatalf("preview identity = %q/%q", preview.BackupID, preview.TargetPath)
	}
	if preview.BaseDigest != mcpBytesDigest(current) {
		t.Fatal("preview base digest does not describe the current target")
	}
	if len(preview.Conflicts) != 0 {
		t.Fatalf("preview conflicts = %v", preview.Conflicts)
	}
	if len(preview.Entries) != 3 {
		t.Fatalf("preview entries = %+v, want the three managed removals", preview.Entries)
	}
	for _, entry := range preview.Entries {
		if entry.Op != mcpOpRestore || entry.NewText != "" || entry.OldText == "" {
			t.Fatalf("preview entry %q = %+v, want a restore-to-absent entry", entry.Name, entry)
		}
	}

	// Later edits to unrelated content must survive entry-level restore: there
	// is no whole-file restore in v1.
	edited := bytes.Replace(current, []byte(`"custom_field": "kept"`),
		[]byte(`"custom_field": "kept", "later_edit": "preserved"`), 1)
	if bytes.Equal(edited, current) {
		t.Fatal("fixture edit did not touch the unrelated server")
	}
	edited = bytes.Replace(edited, []byte(`"theme": "dark"`),
		[]byte(`"theme": "dark", "post_apply_field": true`), 1)
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	preview, err = PreviewMCPRestore(home, applied.BackupID, "")
	if err != nil {
		t.Fatalf("PreviewMCPRestore after an unrelated edit: %v", err)
	}
	if len(preview.Conflicts) != 0 {
		t.Fatalf("unrelated edits were treated as conflicts: %v", preview.Conflicts)
	}
	if preview.BaseDigest != mcpBytesDigest(edited) {
		t.Fatal("edited preview base digest is stale")
	}

	result, err := RestoreMCPConfig(home, applied.BackupID, "", preview.BaseDigest)
	if err != nil {
		t.Fatalf("RestoreMCPConfig: %v", err)
	}
	if got := strings.Join(result.Applied, ","); got != "codegraph,context7,exa" {
		t.Fatalf("restored entries = %q", got)
	}
	if result.BackupID == "" || result.BackupID == applied.BackupID {
		t.Fatalf("restore backup identity = %q, want a new pre-restore backup", result.BackupID)
	}
	afterBytes := mcpReadTestFile(t, path)
	if result.InstalledDigest != mcpBytesDigest(afterBytes) {
		t.Fatal("restore installed digest does not describe the target")
	}
	servers := mcpTestServers(t, afterBytes)
	if len(servers) != 1 {
		t.Fatalf("servers after restore = %v, want only the unrelated server", servers)
	}
	unrelated, _ := servers["unrelated"].(map[string]any)
	if unrelated["later_edit"] != "preserved" || unrelated["custom_field"] != "kept" {
		t.Fatalf("restore erased a later edit to an unrelated server: %v", unrelated)
	}
	root := mcpDecodeJSON(t, afterBytes)
	if root["theme"] != "dark" || root["post_apply_field"] != true {
		t.Fatalf("restore erased a later top-level field: %v", root)
	}
	for _, name := range mcpManagedNames() {
		if _, present := servers[name]; present {
			t.Fatalf("restore left managed entry %q behind", name)
		}
	}

	snapshot, err := mcpReadSnapshot(result.BackupPath, result.BackupID)
	if err != nil {
		t.Fatalf("mcpReadSnapshot: %v", err)
	}
	if snapshot.Kind != mcpSnapshotKindRestore || snapshot.BaseDigest != mcpBytesDigest(edited) {
		t.Fatalf("restore snapshot = %+v", snapshot)
	}
	if len(snapshot.Entries) != 3 {
		t.Fatalf("restore snapshot entries = %+v", snapshot.Entries)
	}
	for _, entry := range snapshot.Entries {
		if entry.Op != mcpOpRemove || entry.OldText == "" || entry.NewText != "" {
			t.Fatalf("restore snapshot entry %q = %+v, want the removed text", entry.Name, entry)
		}
	}
	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentComplete {
		t.Fatalf("restore intent phase = %q, want complete", intent.Phase)
	}
	manifest, err := readMCPOwnershipManifest(home)
	if err != nil {
		t.Fatalf("readMCPOwnershipManifest: %v", err)
	}
	if manifest == nil || len(manifest.Entries) != 0 {
		t.Fatalf("manifest after restoring every managed entry = %+v, want none", manifest)
	}

	// Repeating the same restore is an honest no-op with no third backup.
	preview, err = PreviewMCPRestore(home, applied.BackupID, "")
	if err != nil {
		t.Fatalf("PreviewMCPRestore: %v", err)
	}
	for _, entry := range preview.Entries {
		if entry.Op != mcpOpLeave {
			t.Fatalf("second restore preview entry %q op = %q, want leave", entry.Name, entry.Op)
		}
	}
	backupsBefore := mcpTestBackupIDs(t, home)
	noop, err := RestoreMCPConfig(home, applied.BackupID, "", preview.BaseDigest)
	if err != nil {
		t.Fatalf("second RestoreMCPConfig: %v", err)
	}
	if !noop.NoOp || noop.BackupID != "" || noop.Applied != nil {
		t.Fatalf("second restore = %+v, want a no-op", noop)
	}
	if backups := mcpTestBackupIDs(t, home); strings.Join(backups, ",") != strings.Join(backupsBefore, ",") {
		t.Fatalf("no-op restore created another backup: %v -> %v", backupsBefore, backups)
	}
	if got := mcpReadTestFile(t, path); !bytes.Equal(got, afterBytes) {
		t.Fatal("no-op restore changed the target")
	}
	if plan.BaseDigest == result.InstalledDigest {
		t.Fatal("restore left the target at the applied digest")
	}
}

func TestMCPRestoreRefusesUserEditedManagedEntry(t *testing.T) {
	home := t.TempDir()
	path, _, applied := mcpTestApplyAll(t, home)
	backupsBefore := mcpTestBackupIDs(t, home)

	current := mcpReadTestFile(t, path)
	edited := bytes.Replace(current, []byte("https://mcp.context7.com/mcp/oauth"),
		[]byte("https://user.example/mcp"), 1)
	if bytes.Equal(edited, current) {
		t.Fatal("fixture edit did not change the managed entry")
	}
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	preview, err := PreviewMCPRestore(home, applied.BackupID, "")
	if err != nil {
		t.Fatalf("PreviewMCPRestore: %v", err)
	}
	if len(preview.Conflicts) != 1 || !strings.Contains(preview.Conflicts[0], "context7") {
		t.Fatalf("preview conflicts = %v, want the user-edited context7 entry", preview.Conflicts)
	}
	var conflictEntry *MCPPlanEntry
	for i := range preview.Entries {
		if preview.Entries[i].Name == "context7" {
			conflictEntry = &preview.Entries[i]
		}
	}
	if conflictEntry == nil || conflictEntry.Op != mcpOpConflict {
		t.Fatalf("context7 preview entry = %+v, want a conflict", conflictEntry)
	}
	if !strings.Contains(conflictEntry.Reason, "later user edit") {
		t.Fatalf("conflict reason = %q", conflictEntry.Reason)
	}

	result, err := RestoreMCPConfig(home, applied.BackupID, "", preview.BaseDigest)
	if !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("restore error = %v, want ErrMCPConflict", err)
	}
	if result.Applied != nil || result.BackupID != "" {
		t.Fatalf("conflict restore claimed a write: %+v", result)
	}
	assertFileBytesUnchanged(t, path, edited)
	if backups := mcpTestBackupIDs(t, home); strings.Join(backups, ",") != strings.Join(backupsBefore, ",") {
		t.Fatalf("conflict restore created a backup: %v -> %v", backupsBefore, backups)
	}
}

func TestMCPRestoreRequiresExplicitDigest(t *testing.T) {
	home := t.TempDir()
	path, _, applied := mcpTestApplyAll(t, home)
	current := mcpReadTestFile(t, path)
	preview, err := PreviewMCPRestore(home, applied.BackupID, "")
	if err != nil {
		t.Fatalf("PreviewMCPRestore: %v", err)
	}
	if _, err := RestoreMCPConfig(home, applied.BackupID, "", ""); !errors.Is(err, ErrMCPUsage) {
		t.Fatalf("empty digest restore error = %v, want ErrMCPUsage", err)
	}
	if _, err := RestoreMCPConfig(home, applied.BackupID, "", "deadbeef"); !errors.Is(err, ErrMCPStale) {
		t.Fatalf("mismatched digest restore error = %v, want ErrMCPStale", err)
	}
	if preview.BaseDigest != mcpBytesDigest(current) {
		t.Fatal("preview base digest does not describe the current target")
	}
	assertFileBytesUnchanged(t, path, current)
}

func TestMCPRestoreRefusesMissingCorruptOrAbsentBackup(t *testing.T) {
	t.Run("missing backup", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		missing := "20260101T000000.000000000Z"
		if _, err := PreviewMCPRestore(home, missing, ""); !errors.Is(err, ErrMCPConflict) ||
			!strings.Contains(err.Error(), "no MCP backup") {
			t.Fatalf("preview error = %v, want a missing-backup refusal", err)
		}
		if _, err := RestoreMCPConfig(home, missing, "", "deadbeef"); !errors.Is(err, ErrMCPConflict) {
			t.Fatalf("restore error = %v, want ErrMCPConflict", err)
		}
		assertFileBytesUnchanged(t, mcpTestConfigPath(home), []byte(mcpTestBaseConfig))
	})

	t.Run("traversal backup id", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		if _, err := PreviewMCPRestore(home, "../escape", ""); !errors.Is(err, ErrMCPUnsafeTarget) {
			t.Fatalf("preview error = %v, want ErrMCPUnsafeTarget", err)
		}
	})

	t.Run("corrupt snapshot", func(t *testing.T) {
		home := t.TempDir()
		path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		id := "20260101T000001.000000000Z"
		dir := filepath.Join(home, ".agentic-sdd", "mcp", "backups", id)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := PreviewMCPRestore(home, id, ""); !errors.Is(err, ErrMCPConflict) {
			t.Fatalf("preview error = %v, want ErrMCPConflict", err)
		}
		assertFileBytesUnchanged(t, path, []byte(mcpTestBaseConfig))
	})

	t.Run("snapshot records another backup", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		id := "20260101T000002.000000000Z"
		mcpTestWriteSnapshot(t, home, id, mcpSnapshot{Format: mcpSnapshotFormat, BackupID: "other"})
		if _, err := PreviewMCPRestore(home, id, ""); !errors.Is(err, ErrMCPConflict) ||
			!strings.Contains(err.Error(), "records backup ID") {
			t.Fatalf("preview error = %v, want a backup identity refusal", err)
		}
	})

	t.Run("unsupported snapshot format", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		id := "20260101T000003.000000000Z"
		mcpTestWriteSnapshot(t, home, id, mcpSnapshot{Format: 99, BackupID: id})
		if _, err := PreviewMCPRestore(home, id, ""); !errors.Is(err, ErrMCPUnsupported) {
			t.Fatalf("preview error = %v, want ErrMCPUnsupported", err)
		}
	})

	t.Run("target removed after apply", func(t *testing.T) {
		home := t.TempDir()
		path, _, applied := mcpTestApplyAll(t, home)
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewMCPRestore(home, applied.BackupID, "")
		if err != nil {
			t.Fatalf("PreviewMCPRestore: %v", err)
		}
		if len(preview.Conflicts) == 0 || !strings.Contains(strings.Join(preview.Conflicts, " "), "no creation in v1") {
			t.Fatalf("preview conflicts = %v, want a missing-target refusal", preview.Conflicts)
		}
		if _, err := RestoreMCPConfig(home, applied.BackupID, "", "deadbeef"); !errors.Is(err, ErrMCPConflict) {
			t.Fatalf("restore error = %v, want ErrMCPConflict", err)
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("restore recreated the missing target: %v", statErr)
		}
	})
}

func TestMCPRestoreFailureBeforeRenamePreservesTarget(t *testing.T) {
	home := t.TempDir()
	path, _, applied := mcpTestApplyAll(t, home)
	appliedBytes := mcpReadTestFile(t, path)
	preview, err := PreviewMCPRestore(home, applied.BackupID, "")
	if err != nil {
		t.Fatalf("PreviewMCPRestore: %v", err)
	}
	sentinel := errors.New("injected restore failure")

	result, err := restoreMCPConfig(home, applied.BackupID, "", preview.BaseDigest, &mcpApplyHook{
		BeforeRename: func() error { return sentinel },
	})
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("restore error = %v, want the injected cause", err)
	}
	if !strings.Contains(err.Error(), "nothing changed") {
		t.Fatalf("restore error = %q, want an explicit nothing-changed report", err.Error())
	}
	if result.Applied != nil || result.InstalledDigest != "" {
		t.Fatalf("failed restore claimed a write: %+v", result)
	}
	if result.Rollback.Attempted {
		t.Fatal("pre-rename restore failure attempted a rollback")
	}
	assertFileBytesUnchanged(t, path, appliedBytes)
	if result.BackupID == "" {
		t.Fatal("pre-rename restore failure recorded no backup identity")
	}
	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentPrepared {
		t.Fatalf("restore intent phase = %q, want prepared", intent.Phase)
	}
	matches, globErr := filepath.Glob(filepath.Join(mcpTestConfigDir(home), ".agentic-sdd-mcp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("pre-rename restore failure left staged files: %v", matches)
	}
}

func TestMCPRestoreFailureAfterRenameRollsBack(t *testing.T) {
	home := t.TempDir()
	path, _, applied := mcpTestApplyAll(t, home)
	appliedBytes := mcpReadTestFile(t, path)
	preview, err := PreviewMCPRestore(home, applied.BackupID, "")
	if err != nil {
		t.Fatalf("PreviewMCPRestore: %v", err)
	}
	sentinel := errors.New("injected restore failure")

	result, err := restoreMCPConfig(home, applied.BackupID, "", preview.BaseDigest, &mcpApplyHook{
		AfterRename: func() error { return sentinel },
	})
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("restore error = %v, want the injected cause", err)
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("restore error = %q, want an explicit rolled-back report", err.Error())
	}
	if result.Applied != nil || !result.Rollback.Attempted || !result.Rollback.Complete {
		t.Fatalf("restore failure result = %+v", result)
	}
	if got := mcpReadTestFile(t, path); !bytes.Equal(got, appliedBytes) {
		t.Fatalf("restore rollback did not reproduce the applied bytes:\ngot:\n%s\nwant:\n%s", got, appliedBytes)
	}
	servers := mcpTestServers(t, mcpReadTestFile(t, path))
	if len(servers) != 4 {
		t.Fatalf("servers after restore rollback = %v, want the applied entries plus the unrelated server", servers)
	}
	intent, err := ReadMCPTransactionIntent(home, result.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.Phase != mcpIntentRolledBack {
		t.Fatalf("restore intent phase = %q, want rolled-back", intent.Phase)
	}
}

func TestMCPTransactionIntentEvidence(t *testing.T) {
	home := t.TempDir()
	_, plan, applied := mcpTestApplyAll(t, home)

	intent, err := ReadMCPTransactionIntent(home, applied.BackupID)
	if err != nil {
		t.Fatalf("ReadMCPTransactionIntent: %v", err)
	}
	if intent.BackupID != applied.BackupID || intent.TargetPath != plan.TargetPath {
		t.Fatalf("intent identity = %+v", intent)
	}
	if intent.BaseDigest != plan.BaseDigest || intent.PlanDigest != plan.PlanDigest {
		t.Fatalf("intent digests = %+v", intent)
	}

	if _, err := ReadMCPTransactionIntent(home, "20260101T000000.000000000Z"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing intent error = %v, want os.ErrNotExist", err)
	}
	if _, err := ReadMCPTransactionIntent(home, "../escape"); !errors.Is(err, ErrMCPUnsafeTarget) {
		t.Fatalf("traversal intent error = %v, want ErrMCPUnsafeTarget", err)
	}
	if _, err := ReadMCPTransactionIntent("", applied.BackupID); !errors.Is(err, ErrMCPUsage) {
		t.Fatalf("empty home error = %v, want ErrMCPUsage", err)
	}
}
