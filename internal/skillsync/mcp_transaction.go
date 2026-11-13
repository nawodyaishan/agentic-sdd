package skillsync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"agentic-sdd/internal/version"
)

// MCP-B transactional engine. One digest-checked, entry-level write of the
// managed MCP servers into an existing OpenCode V2 configuration.
//
// The transaction is deliberately small and honest:
//   - a cross-process lock file covers the target and the ownership manifest;
//   - a bounded local `opencode --version` probe fails closed;
//   - a private backup stores only the managed-entry text/hashes and target
//     identity, never unrelated servers or their credentials;
//   - a durable intent record is written before the target is replaced;
//   - a same-directory temp file is fsynced, renamed, then the directory is
//     fsynced;
//   - a failure after the rename restores the managed entries exactly, or
//     reports needs-recovery with the backup identity and digests.
const (
	mcpBackupsDirName = "backups"
	mcpLockFileName   = ".lock"
	mcpSnapshotFile   = "snapshot.json"
	mcpIntentFile     = "intent.json"

	mcpLockStaleAfter      = 30 * time.Second
	mcpRuntimeProbeTimeout = 5 * time.Second

	mcpSnapshotFormat  = 1
	mcpManifestFormat  = 1
	mcpManifestVersion = 1

	mcpIntentPrepared        = "prepared"
	mcpIntentRenamed         = "renamed"
	mcpIntentManifestWritten = "manifest-written"
	mcpIntentComplete        = "complete"
	mcpIntentRolledBack      = "rolled-back"
	mcpIntentNeedsRecovery   = "needs-recovery"

	mcpSnapshotKindApply   = "apply"
	mcpSnapshotKindRestore = "restore"

	mcpRuntimeEvidenceLocal = "local-probe"
)

// ErrMCPLocked reports that another process holds the shared MCP engine lock.
// It is always returned together with ErrMCPConflict so a caller that only
// knows the four shared sentinels still classifies it as a refusal.
var ErrMCPLocked = errors.New("MCP engine lock is held by another writer")

// mcpSnapshot is the private, entry-level backup record. It stores the exact
// managed-entry texts and digests needed to reverse one write. It never stores
// unrelated servers, unrelated top-level fields or credentials.
type mcpSnapshot struct {
	Format          int                `json:"format"`
	BackupID        string             `json:"backup_id"`
	Kind            string             `json:"kind"`
	Created         string             `json:"created"`
	ToolVersion     string             `json:"tool_version"`
	Target          string             `json:"target"`
	TargetFormat    string             `json:"target_format"`
	Schema          string             `json:"schema"`
	TargetMode      uint32             `json:"target_mode"`
	BaseDigest      string             `json:"base_digest"`
	InstalledDigest string             `json:"installed_digest"`
	PlanDigest      string             `json:"plan_digest"`
	RuntimeVersion  string             `json:"runtime_version"`
	RuntimeEvidence string             `json:"runtime_evidence"`
	Actions         []string           `json:"actions"`
	CreatedMCP      bool               `json:"created_mcp"`
	CreatedServers  bool               `json:"created_servers"`
	Entries         []mcpSnapshotEntry `json:"entries"`
}

// mcpSnapshotEntry is one entry-level change. Op is add, change or remove.
type mcpSnapshotEntry struct {
	Name    string `json:"name"`
	Op      string `json:"op"`
	OldText string `json:"old_text,omitempty"`
	OldHash string `json:"old_hash,omitempty"`
	NewText string `json:"new_text,omitempty"`
	NewHash string `json:"new_hash,omitempty"`
}

// mcpManifestEntry records one managed server and the digest of the text this
// tool installed. No entry text, header value or credential is stored.
type mcpManifestEntry struct {
	Name            string `json:"name"`
	InstalledDigest string `json:"installed_digest"`
	BackupID        string `json:"backup_id"`
}

// mcpOwnershipManifest is the MCP ownership manifest (format 1). It is
// separate from the skillsync backup manifest and from Feature 014's fleet
// manifest.
type mcpOwnershipManifest struct {
	Format          int                `json:"format"`
	ManifestVersion int                `json:"manifest_version"`
	ToolVersion     string             `json:"tool_version"`
	Created         string             `json:"created"`
	Target          string             `json:"target"`
	TargetFormat    string             `json:"target_format"`
	Schema          string             `json:"schema"`
	BaseDigest      string             `json:"base_digest"`
	InstalledDigest string             `json:"installed_digest"`
	BackupID        string             `json:"backup_id"`
	BackupPath      string             `json:"backup_path"`
	RuntimeVersion  string             `json:"runtime_version"`
	RuntimeEvidence string             `json:"runtime_evidence"`
	Actions         []string           `json:"actions"`
	Entries         []mcpManifestEntry `json:"entries"`
}

// mcpIntentRecord is the durable transaction-intent evidence.
type mcpIntentRecord struct {
	BackupID            string `json:"backup_id"`
	TargetPath          string `json:"target_path"`
	BaseDigest          string `json:"base_digest"`
	PlanDigest          string `json:"plan_digest"`
	PriorManifestDigest string `json:"prior_manifest_digest"`
	Phase               string `json:"phase"`
	Updated             string `json:"updated"`
}

// mcpApplyHook lets tests inject a failure at a precise transaction phase. A
// nil hook is production behavior; there is no package-level mutable state.
type mcpApplyHook struct {
	BeforeRename  func() error
	AfterRename   func() error
	AfterManifest func() error
}

// mcpStateDir returns <home>/.agentic-sdd/mcp.
func mcpStateDir(home string) string {
	return filepath.Join(home, mcpStateDirRel)
}

// mcpBackupsRoot returns the private MCP backup root.
func mcpBackupsRoot(home string) string {
	return filepath.Join(mcpStateDir(home), mcpBackupsDirName)
}

// mcpOwnershipPath returns the MCP ownership manifest path.
func mcpOwnershipPath(home string) string {
	return filepath.Join(mcpStateDir(home), mcpOwnershipFile)
}

// mcpLockPath returns the shared engine lock path.
func mcpLockPath(home string) string {
	return filepath.Join(mcpStateDir(home), mcpLockFileName)
}

// ApplyMCPPlan applies one digest-bound plan. It acquires the shared engine
// lock, re-reads and re-verifies the target under it, probes the local OpenCode
// runtime, writes a private entry-level backup and intent record, replaces the
// target atomically and records the ownership manifest.
func ApplyMCPPlan(home, configPath string, plan *MCPPlan, opts MCPApplyOptions) (MCPApplyResult, error) {
	return applyMCPPlan(home, configPath, plan, opts, nil)
}

func applyMCPPlan(home, configPath string, plan *MCPPlan, opts MCPApplyOptions, hook *mcpApplyHook) (MCPApplyResult, error) {
	result := MCPApplyResult{}

	if home == "" {
		return result, fmt.Errorf("%w: home directory is required", ErrMCPUsage)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return result, fmt.Errorf("%w: resolve home directory: %v", ErrMCPUsage, err)
	}
	if opts.Home != "" {
		absOptHome, absErr := filepath.Abs(opts.Home)
		if absErr != nil {
			return result, fmt.Errorf("%w: resolve apply home: %v", ErrMCPUsage, absErr)
		}
		if absOptHome != absHome {
			return result, fmt.Errorf("%w: apply home %s does not match request home %s",
				ErrMCPUsage, absOptHome, absHome)
		}
	}
	if plan == nil {
		return result, fmt.Errorf("%w: an MCP plan is required", ErrMCPUsage)
	}

	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return result, fmt.Errorf("%w: resolve config path: %v", ErrMCPUsage, err)
		}
	}
	expectedPath, expectedFormat, err := mcpExpectedTarget(absHome, absConfig)
	if err != nil {
		return result, err
	}
	result.TargetPath = expectedPath

	if plan.TargetPath != expectedPath {
		return result, fmt.Errorf("%w: plan target %q does not match the current target %q; re-run preview",
			ErrMCPConflict, plan.TargetPath, expectedPath)
	}
	if plan.Format != expectedFormat {
		return result, fmt.Errorf("%w: plan format %q does not match the target format %q; re-run preview",
			ErrMCPConflict, plan.Format, expectedFormat)
	}
	if len(plan.Conflicts) > 0 {
		return result, fmt.Errorf("%w: the plan has conflicts: %s",
			ErrMCPConflict, strings.Join(plan.Conflicts, "; "))
	}
	if plan.Format != mcpFormatJSON {
		return result, fmt.Errorf(
			"%w: %s configuration edits are not supported in this build; merge the manual snippet instead",
			ErrMCPUnsupported, plan.Format)
	}
	if opts.ExpectDigest == "" {
		return result, fmt.Errorf("%w: an explicit --expect-config digest is required before an MCP apply", ErrMCPUsage)
	}
	if plan.BaseDigest == "" {
		return result, fmt.Errorf("%w: the plan has no base digest; only a plan built from an existing target can be applied",
			ErrMCPConflict)
	}
	if opts.ExpectDigest != plan.BaseDigest {
		return result, fmt.Errorf("%w: expected digest %s does not match the plan base digest %s; re-run preview",
			ErrMCPStale, opts.ExpectDigest, plan.BaseDigest)
	}

	current, _, err := mcpReadConfigBytes(expectedPath)
	if err != nil {
		return result, err
	}
	if got := mcpBytesDigest(current); got != plan.BaseDigest {
		return result, fmt.Errorf("%w: %s changed since preview (base %s, current %s); re-run preview",
			ErrMCPStale, expectedPath, plan.BaseDigest, got)
	}

	if plan.NoOp {
		result.NoOp = true
		result.InstalledDigest = mcpBytesDigest(current)
		result.RuntimeVersion = plan.RuntimeVersion
		return result, nil
	}
	if len(plan.targetBytes) == 0 {
		return result, fmt.Errorf("%w: the plan carries no target bytes to write", ErrMCPConflict)
	}

	lock, err := acquireMCPLock(absHome)
	if err != nil {
		return result, err
	}
	defer func() { _ = lock.Release() }()

	// Re-read under the lock: the CAS digest must still describe the file that
	// is about to be replaced.
	lockedBytes, lockedMode, err := mcpReadConfigBytes(expectedPath)
	if err != nil {
		return result, err
	}
	if got := mcpBytesDigest(lockedBytes); got != plan.BaseDigest {
		return result, fmt.Errorf("%w: %s changed before the lock was acquired (base %s, current %s); re-run preview",
			ErrMCPStale, expectedPath, plan.BaseDigest, got)
	}

	runtimePath, runtimeVersion, err := probeMCPRuntime(opts.OpenCodeBin)
	if err != nil {
		return result, err
	}
	result.RuntimeVersion = runtimeVersion
	plan.RuntimeVersion = runtimeVersion
	plan.RuntimeEvidence = mcpRuntimeEvidenceLocal + ":" + runtimePath

	baseRoot, err := mcpParseDocument(expectedPath, mcpFormatJSON, lockedBytes)
	if err != nil {
		return result, err
	}
	_, baseHasMCP, baseHasServers, err := mcpServersOf(baseRoot)
	if err != nil {
		return result, err
	}

	installedDigest := mcpBytesDigest(plan.targetBytes)
	result.InstalledDigest = installedDigest

	priorManifestDigest := ""
	if data, readErr := os.ReadFile(mcpOwnershipPath(absHome)); readErr == nil {
		priorManifestDigest = mcpBytesDigest(data)
	}

	backupID := time.Now().UTC().Format("20060102T150405.000000000Z")
	backupPath := filepath.Join(mcpBackupsRoot(absHome), backupID)
	if err := mcpEnsurePrivateDir(mcpBackupsRoot(absHome)); err != nil {
		return result, err
	}
	if err := os.Mkdir(backupPath, 0700); err != nil {
		return result, fmt.Errorf("create MCP backup %s: %w", backupPath, err)
	}
	result.BackupID = backupID
	result.BackupPath = backupPath

	snapshot := mcpSnapshot{
		Format:          mcpSnapshotFormat,
		BackupID:        backupID,
		Kind:            mcpSnapshotKindApply,
		Created:         time.Now().UTC().Format(time.RFC3339Nano),
		ToolVersion:     version.Version,
		Target:          expectedPath,
		TargetFormat:    plan.Format,
		Schema:          plan.Schema,
		TargetMode:      uint32(lockedMode.Perm()),
		BaseDigest:      plan.BaseDigest,
		InstalledDigest: installedDigest,
		PlanDigest:      plan.PlanDigest,
		RuntimeVersion:  runtimeVersion,
		RuntimeEvidence: plan.RuntimeEvidence,
		CreatedMCP:      !baseHasMCP,
		CreatedServers:  !baseHasServers,
	}
	for _, entry := range plan.Entries {
		snapshot.Actions = append(snapshot.Actions, entry.Name+":"+entry.Op)
		switch entry.Op {
		case mcpOpAdd:
			snapshot.Entries = append(snapshot.Entries, mcpSnapshotEntry{
				Name: entry.Name, Op: mcpOpAdd,
				NewText: entry.NewText, NewHash: entry.NewHash,
			})
		case mcpOpChange:
			snapshot.Entries = append(snapshot.Entries, mcpSnapshotEntry{
				Name: entry.Name, Op: mcpOpChange,
				OldText: entry.OldText, OldHash: entry.OldHash,
				NewText: entry.NewText, NewHash: entry.NewHash,
			})
		}
	}
	if err := mcpWriteSnapshot(backupPath, snapshot); err != nil {
		return mcpApplyBeforeWrite(result, mcpIntentPrepared, backupID, err)
	}

	intent := mcpIntentRecord{
		BackupID:            backupID,
		TargetPath:          expectedPath,
		BaseDigest:          plan.BaseDigest,
		PlanDigest:          plan.PlanDigest,
		PriorManifestDigest: priorManifestDigest,
		Phase:               mcpIntentPrepared,
		Updated:             time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := mcpWriteIntent(backupPath, intent); err != nil {
		return mcpApplyBeforeWrite(result, mcpIntentPrepared, backupID, err)
	}

	staged, err := mcpStageTarget(expectedPath, plan.targetBytes, lockedMode.Perm())
	if err != nil {
		return mcpApplyBeforeWrite(result, mcpIntentPrepared, backupID, err)
	}
	if hook != nil && hook.BeforeRename != nil {
		if err := hook.BeforeRename(); err != nil {
			_ = os.Remove(staged)
			return mcpApplyBeforeWrite(result, mcpIntentPrepared, backupID, err)
		}
	}
	if err := os.Rename(staged, expectedPath); err != nil {
		_ = os.Remove(staged)
		return mcpApplyBeforeWrite(result, mcpIntentPrepared, backupID, err)
	}
	_ = mcpSyncDir(filepath.Dir(expectedPath))

	intent.Phase = mcpIntentRenamed
	intent.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	_ = mcpWriteIntent(backupPath, intent)

	if hook != nil && hook.AfterRename != nil {
		if hookErr := hook.AfterRename(); hookErr != nil {
			return mcpApplyRollback(absHome, result, snapshot, backupPath, mcpIntentRenamed, hookErr)
		}
	}

	postRoot, err := mcpParseDocument(expectedPath, mcpFormatJSON, plan.targetBytes)
	if err != nil {
		return mcpApplyRollback(absHome, result, snapshot, backupPath, mcpIntentRenamed, err)
	}
	postServers, _, _, err := mcpServersOf(postRoot)
	if err != nil {
		return mcpApplyRollback(absHome, result, snapshot, backupPath, mcpIntentRenamed, err)
	}

	manifest, err := mcpNextOwnershipManifest(absHome, expectedPath, plan, snapshot, runtimePath, postServers)
	if err != nil {
		return mcpApplyRollback(absHome, result, snapshot, backupPath, mcpIntentRenamed, err)
	}
	if err := writeMCPOwnershipManifest(absHome, manifest); err != nil {
		return mcpApplyRollback(absHome, result, snapshot, backupPath, mcpIntentRenamed, err)
	}

	intent.Phase = mcpIntentManifestWritten
	intent.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	_ = mcpWriteIntent(backupPath, intent)

	if hook != nil && hook.AfterManifest != nil {
		if hookErr := hook.AfterManifest(); hookErr != nil {
			return mcpApplyRollback(absHome, result, snapshot, backupPath, mcpIntentManifestWritten, hookErr)
		}
	}

	intent.Phase = mcpIntentComplete
	intent.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	_ = mcpWriteIntent(backupPath, intent)

	result.Applied = mcpPlannedNames(plan.Entries)
	return result, nil
}

// mcpApplyBeforeWrite reports a failure that happened before the target was
// replaced: nothing changed, and the intent phase names what was prepared.
func mcpApplyBeforeWrite(result MCPApplyResult, phase, backupID string, cause error) (MCPApplyResult, error) {
	result.Applied = nil
	result.InstalledDigest = ""
	return result, fmt.Errorf(
		"mcp apply failed before writing the configuration; nothing changed (transaction phase %s, backup %s): %w",
		phase, backupID, cause)
}

// mcpApplyRollback restores the managed entries after a post-rename failure
// and reports either "rolled back" or "needs recovery" with exact evidence.
func mcpApplyRollback(home string, result MCPApplyResult, snapshot mcpSnapshot, backupPath, phase string, cause error) (MCPApplyResult, error) {
	rollbackErr := mcpRestoreSnapshotEntries(result.TargetPath, snapshot, os.FileMode(snapshot.TargetMode))
	if rollbackErr == nil {
		result.Applied = nil
		result.InstalledDigest = snapshot.BaseDigest
		result.Rollback = RollbackInfo{Attempted: true, Complete: true}
		mcpUpdateIntentPhase(backupPath, mcpIntentRolledBack)
		return result, fmt.Errorf(
			"mcp apply failed after replacing %s; managed entries were rolled back and unrelated content is unchanged (transaction phase %s, backup %s): %w",
			snapshot.Target, phase, snapshot.BackupID, cause)
	}

	result.Applied = nil
	result.Rollback = RollbackInfo{Attempted: true, Complete: false, Error: rollbackErr.Error()}
	mcpUpdateIntentPhase(backupPath, mcpIntentNeedsRecovery)
	currentDigest := ""
	if data, err := os.ReadFile(snapshot.Target); err == nil {
		currentDigest = mcpBytesDigest(data)
	}
	return result, fmt.Errorf(
		"%w: mcp apply needs recovery (transaction phase %s, backup %s at %s, target %s, expected installed digest %s, current digest %s): rollback failed: %v; original error: %v",
		ErrMCPConflict, phase, snapshot.BackupID, backupPath, snapshot.Target,
		snapshot.InstalledDigest, currentDigest, rollbackErr, cause)
}

// mcpRestoreSnapshotEntries reverses one snapshot's entry-level changes.
func mcpRestoreSnapshotEntries(path string, snapshot mcpSnapshot, mode os.FileMode) error {
	current, _, err := mcpReadConfigBytes(path)
	if err != nil {
		return err
	}
	if got := mcpBytesDigest(current); got != snapshot.InstalledDigest {
		return fmt.Errorf("target no longer matches the installed digest %s (current %s)", snapshot.InstalledDigest, got)
	}
	root, err := mcpParseDocument(path, mcpFormatJSON, current)
	if err != nil {
		return err
	}
	servers, hasMCP, hasServers, err := mcpServersOf(root)
	if err != nil {
		return err
	}
	if servers == nil {
		servers = map[string]any{}
	}
	for _, entry := range snapshot.Entries {
		switch entry.Op {
		case mcpOpAdd:
			delete(servers, entry.Name)
		case mcpOpChange, mcpOpRemove:
			if entry.OldText == "" {
				delete(servers, entry.Name)
				continue
			}
			var value any
			dec := json.NewDecoder(strings.NewReader(entry.OldText))
			dec.UseNumber()
			if err := dec.Decode(&value); err != nil {
				return fmt.Errorf("decode snapshot entry %q: %w", entry.Name, err)
			}
			servers[entry.Name] = value
		default:
			return fmt.Errorf("snapshot entry %q has unsupported operation %q", entry.Name, entry.Op)
		}
	}
	if hasServers && snapshot.CreatedServers && len(servers) == 0 {
		if mcpMap, ok := root["mcp"].(map[string]any); ok {
			delete(mcpMap, "servers")
		}
	}
	if hasMCP && snapshot.CreatedMCP {
		if mcpMap, ok := root["mcp"].(map[string]any); ok && len(mcpMap) == 0 {
			delete(root, "mcp")
		}
	}
	newBytes, err := mcpMarshalDocument(root, current)
	if err != nil {
		return err
	}
	if err := mcpWriteFileAtomic(path, newBytes, mode); err != nil {
		return err
	}
	return mcpVerifySnapshotEntries(path, snapshot)
}

// mcpVerifySnapshotEntries proves a rollback landed the original entry bytes.
func mcpVerifySnapshotEntries(path string, snapshot mcpSnapshot) error {
	data, _, err := mcpReadConfigBytes(path)
	if err != nil {
		return err
	}
	root, err := mcpParseDocument(path, mcpFormatJSON, data)
	if err != nil {
		return err
	}
	servers, _, _, err := mcpServersOf(root)
	if err != nil {
		return err
	}
	for _, entry := range snapshot.Entries {
		current, present := servers[entry.Name]
		switch entry.Op {
		case mcpOpAdd:
			if present {
				return fmt.Errorf("entry %q is still present after rollback", entry.Name)
			}
		case mcpOpChange, mcpOpRemove:
			if entry.OldText == "" {
				if present {
					return fmt.Errorf("entry %q is still present after rollback", entry.Name)
				}
				continue
			}
			entryMap, ok := current.(map[string]any)
			if !ok {
				return fmt.Errorf("entry %q is not present after rollback", entry.Name)
			}
			_, hash, err := mcpCanonicalEntry(entryMap)
			if err != nil {
				return err
			}
			if entry.OldHash != "" && hash != entry.OldHash {
				return fmt.Errorf("entry %q does not match the original digest after rollback", entry.Name)
			}
		}
	}
	return nil
}

// mcpNextOwnershipManifest builds the manifest for a successful apply,
// retaining ownership of names this plan did not touch while their installed
// digest still describes the file that was just written.
func mcpNextOwnershipManifest(home, target string, plan *MCPPlan, snapshot mcpSnapshot, runtimePath string, postServers map[string]any) (mcpOwnershipManifest, error) {
	previous, err := readMCPOwnershipManifest(home)
	if err != nil {
		return mcpOwnershipManifest{}, err
	}
	manifest := mcpOwnershipManifest{
		Format:          mcpManifestFormat,
		ManifestVersion: mcpManifestVersion,
		ToolVersion:     version.Version,
		Created:         time.Now().UTC().Format(time.RFC3339Nano),
		Target:          target,
		TargetFormat:    plan.Format,
		Schema:          plan.Schema,
		BaseDigest:      snapshot.BaseDigest,
		InstalledDigest: snapshot.InstalledDigest,
		BackupID:        snapshot.BackupID,
		BackupPath:      filepath.Join(mcpBackupsRoot(home), snapshot.BackupID),
		RuntimeVersion:  snapshot.RuntimeVersion,
		RuntimeEvidence: snapshot.RuntimeEvidence,
		Actions:         append([]string(nil), snapshot.Actions...),
	}
	if runtimePath == "" {
		manifest.RuntimeEvidence = "not-checked"
	}

	handled := map[string]bool{}
	for _, entry := range plan.Entries {
		switch entry.Op {
		case mcpOpAdd, mcpOpChange:
			manifest.Entries = append(manifest.Entries, mcpManifestEntry{
				Name:            entry.Name,
				InstalledDigest: entry.NewHash,
				BackupID:        snapshot.BackupID,
			})
			handled[entry.Name] = true
		case mcpOpLeave, mcpOpDisabled:
			// A still-owned entry keeps its recorded digest below; an identical
			// but unmanaged entry is not adopted by a no-write plan.
			handled[entry.Name] = true
			if previousEntry, ok := mcpRecordedEntry(previous, target, entry.Name); ok &&
				previousEntry.InstalledDigest == entry.OldHash {
				manifest.Entries = append(manifest.Entries, previousEntry)
			}
		}
	}
	if previous != nil && previous.Target == target {
		for _, previousEntry := range previous.Entries {
			if handled[previousEntry.Name] {
				continue
			}
			// Retain ownership only while the written file still carries the
			// exact entry text this tool installed.
			current, ok := postServers[previousEntry.Name].(map[string]any)
			if !ok {
				continue
			}
			_, hash, err := mcpCanonicalEntry(current)
			if err != nil || hash != previousEntry.InstalledDigest {
				continue
			}
			manifest.Entries = append(manifest.Entries, previousEntry)
		}
	}
	manifest.Entries = mcpSortManifestEntries(manifest.Entries)
	return manifest, nil
}

// mcpRecordedEntry returns a recorded ownership entry for target and name.
func mcpRecordedEntry(manifest *mcpOwnershipManifest, target, name string) (mcpManifestEntry, bool) {
	if manifest == nil || manifest.Target != target {
		return mcpManifestEntry{}, false
	}
	for _, entry := range manifest.Entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return mcpManifestEntry{}, false
}

func mcpSortManifestEntries(entries []mcpManifestEntry) []mcpManifestEntry {
	order := map[string]int{}
	for i, name := range mcpManagedNames() {
		order[name] = i
	}
	sorted := append([]mcpManifestEntry(nil), entries...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left, leftOK := order[sorted[i].Name]
		right, rightOK := order[sorted[j].Name]
		if leftOK && rightOK {
			return left < right
		}
		if leftOK != rightOK {
			return leftOK
		}
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

// mcpPlannedNames lists the names a plan or restore actually writes, in order.
func mcpPlannedNames(entries []MCPPlanEntry) []string {
	var names []string
	for _, entry := range entries {
		if entry.Op == mcpOpAdd || entry.Op == mcpOpChange || entry.Op == mcpOpRestore {
			names = append(names, entry.Name)
		}
	}
	return names
}

// mcpExpectedTarget re-resolves the apply/restore target and format. A missing
// target is refused: v1 never creates a configuration file.
func mcpExpectedTarget(home, absConfig string) (string, string, error) {
	if absConfig != "" {
		format := mcpFormatForPath(absConfig)
		if format == "" {
			return "", "", fmt.Errorf("%w: unsupported configuration extension %q; expected .json or .jsonc",
				ErrMCPUnsupported, filepath.Ext(absConfig))
		}
		scopeDir := filepath.Join(home, mcpConfigDirRel)
		if !mcpPathWithin(scopeDir, absConfig) {
			return "", "", fmt.Errorf(
				"%w: %s is outside the supported global scope %s; project-local MCP scope is deferred in v1",
				ErrMCPUnsafeTarget, absConfig, scopeDir)
		}
		if err := mcpCheckTargetSafety(absConfig, absConfig); err != nil {
			return "", "", err
		}
		return absConfig, format, nil
	}

	jsonPath := filepath.Join(home, mcpConfigDirRel, mcpDefaultJSON)
	jsoncPath := filepath.Join(home, mcpConfigDirRel, mcpDefaultJSONC)
	_, jsonErr := os.Lstat(jsonPath)
	_, jsoncErr := os.Lstat(jsoncPath)
	hasJSON := jsonErr == nil
	hasJSONC := jsoncErr == nil
	if hasJSON && hasJSONC {
		return "", "", fmt.Errorf("%w: both %s and %s exist; refusing to guess which target to manage",
			ErrMCPConflict, jsonPath, jsoncPath)
	}
	if !hasJSON && !hasJSONC {
		return "", "", fmt.Errorf("%w: target missing; no creation in v1 (%s)", ErrMCPConflict, jsonPath)
	}
	path := jsonPath
	format := mcpFormatJSON
	if hasJSONC {
		path = jsoncPath
		format = mcpFormatJSONC
	}
	if err := mcpCheckTargetSafety("", path); err != nil {
		return "", "", err
	}
	return path, format, nil
}

// PreviewMCPRestore compares the current target with one private backup,
// reporting only managed entries. It is read-only and creates nothing.
func PreviewMCPRestore(home, backupID, configPath string) (*MCPRestorePreview, error) {
	preview, _, err := mcpRestorePlan(home, backupID, configPath)
	return preview, err
}

// RestoreMCPConfig restores only the managed entries recorded by backupID. It
// refuses when a managed entry changed after the apply, and it never performs a
// whole-file restore.
func RestoreMCPConfig(home, backupID, configPath, expectDigest string) (MCPApplyResult, error) {
	return restoreMCPConfig(home, backupID, configPath, expectDigest, nil)
}

func restoreMCPConfig(home, backupID, configPath, expectDigest string, hook *mcpApplyHook) (MCPApplyResult, error) {
	result := MCPApplyResult{}
	preview, state, err := mcpRestorePlan(home, backupID, configPath)
	if err != nil {
		return result, err
	}
	result.TargetPath = preview.TargetPath
	if len(preview.Conflicts) > 0 {
		return result, fmt.Errorf("%w: restore has conflicts: %s",
			ErrMCPConflict, strings.Join(preview.Conflicts, "; "))
	}
	if expectDigest == "" {
		return result, fmt.Errorf("%w: an explicit configuration digest is required before an MCP restore", ErrMCPUsage)
	}
	if expectDigest != preview.BaseDigest {
		return result, fmt.Errorf("%w: expected digest %s does not match the current target digest %s; re-run restore preview",
			ErrMCPStale, expectDigest, preview.BaseDigest)
	}

	var toRestore []MCPPlanEntry
	for _, entry := range preview.Entries {
		if entry.Op == mcpOpRestore {
			toRestore = append(toRestore, entry)
		}
	}
	if len(toRestore) == 0 {
		result.NoOp = true
		result.InstalledDigest = preview.BaseDigest
		result.RuntimeVersion = "not-checked"
		return result, nil
	}

	absHome, err := filepath.Abs(state.Home)
	if err != nil {
		return result, fmt.Errorf("%w: resolve home directory: %v", ErrMCPUsage, err)
	}
	lock, err := acquireMCPLock(absHome)
	if err != nil {
		return result, err
	}
	defer func() { _ = lock.Release() }()

	lockedBytes, lockedMode, err := mcpReadConfigBytes(state.TargetPath)
	if err != nil {
		return result, err
	}
	if got := mcpBytesDigest(lockedBytes); got != preview.BaseDigest {
		return result, fmt.Errorf("%w: %s changed before the lock was acquired (base %s, current %s)",
			ErrMCPStale, state.TargetPath, preview.BaseDigest, got)
	}
	root, err := mcpParseDocument(state.TargetPath, mcpFormatJSON, lockedBytes)
	if err != nil {
		return result, err
	}
	servers, hasMCP, hasServers, err := mcpServersOf(root)
	if err != nil {
		return result, err
	}
	if servers == nil {
		root = mcpCloneDocument(root)
		servers = mcpEnsureServers(root, hasMCP, hasServers)
	}
	for _, entry := range toRestore {
		if entry.NewText == "" {
			delete(servers, entry.Name)
			continue
		}
		var value any
		dec := json.NewDecoder(strings.NewReader(entry.NewText))
		dec.UseNumber()
		if err := dec.Decode(&value); err != nil {
			return result, fmt.Errorf("decode restore entry %q: %w", entry.Name, err)
		}
		servers[entry.Name] = value
	}
	newBytes, err := mcpMarshalDocument(root, lockedBytes)
	if err != nil {
		return result, fmt.Errorf("%w: render restored MCP configuration: %v", ErrMCPConflict, err)
	}
	installedDigest := mcpBytesDigest(newBytes)
	result.InstalledDigest = installedDigest
	result.RuntimeVersion = "not-checked"

	priorManifestDigest := ""
	if data, readErr := os.ReadFile(mcpOwnershipPath(absHome)); readErr == nil {
		priorManifestDigest = mcpBytesDigest(data)
	}

	newBackupID := time.Now().UTC().Format("20060102T150405.000000000Z")
	newBackupPath := filepath.Join(mcpBackupsRoot(absHome), newBackupID)
	if err := mcpEnsurePrivateDir(mcpBackupsRoot(absHome)); err != nil {
		return result, err
	}
	if err := os.Mkdir(newBackupPath, 0700); err != nil {
		return result, fmt.Errorf("create MCP restore backup %s: %w", newBackupPath, err)
	}
	result.BackupID = newBackupID
	result.BackupPath = newBackupPath

	snapshot := mcpSnapshot{
		Format:          mcpSnapshotFormat,
		BackupID:        newBackupID,
		Kind:            mcpSnapshotKindRestore,
		Created:         time.Now().UTC().Format(time.RFC3339Nano),
		ToolVersion:     version.Version,
		Target:          state.TargetPath,
		TargetFormat:    mcpFormatJSON,
		Schema:          mcpSchemaV2,
		TargetMode:      uint32(lockedMode.Perm()),
		BaseDigest:      preview.BaseDigest,
		InstalledDigest: installedDigest,
		PlanDigest:      "",
		RuntimeVersion:  "not-checked",
		RuntimeEvidence: "not-checked",
	}
	for _, entry := range toRestore {
		snapshot.Actions = append(snapshot.Actions, entry.Name+":"+mcpOpRestore)
		op := mcpOpChange
		if entry.NewText == "" {
			op = mcpOpRemove
		}
		snapshot.Entries = append(snapshot.Entries, mcpSnapshotEntry{
			Name: entry.Name, Op: op,
			OldText: entry.OldText, OldHash: entry.OldHash,
			NewText: entry.NewText, NewHash: entry.NewHash,
		})
	}
	if err := mcpWriteSnapshot(newBackupPath, snapshot); err != nil {
		return mcpRestoreBeforeWrite(result, newBackupID, err)
	}
	intent := mcpIntentRecord{
		BackupID:            newBackupID,
		TargetPath:          state.TargetPath,
		BaseDigest:          preview.BaseDigest,
		PlanDigest:          "",
		PriorManifestDigest: priorManifestDigest,
		Phase:               mcpIntentPrepared,
		Updated:             time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := mcpWriteIntent(newBackupPath, intent); err != nil {
		return mcpRestoreBeforeWrite(result, newBackupID, err)
	}

	staged, err := mcpStageTarget(state.TargetPath, newBytes, lockedMode.Perm())
	if err != nil {
		return mcpRestoreBeforeWrite(result, newBackupID, err)
	}
	if hook != nil && hook.BeforeRename != nil {
		if err := hook.BeforeRename(); err != nil {
			_ = os.Remove(staged)
			return mcpRestoreBeforeWrite(result, newBackupID, err)
		}
	}
	if err := os.Rename(staged, state.TargetPath); err != nil {
		_ = os.Remove(staged)
		return mcpRestoreBeforeWrite(result, newBackupID, err)
	}
	_ = mcpSyncDir(filepath.Dir(state.TargetPath))

	intent.Phase = mcpIntentRenamed
	intent.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	_ = mcpWriteIntent(newBackupPath, intent)
	if hook != nil && hook.AfterRename != nil {
		if hookErr := hook.AfterRename(); hookErr != nil {
			return mcpApplyRollback(absHome, result, snapshot, newBackupPath, mcpIntentRenamed, hookErr)
		}
	}

	manifest, err := mcpRestoreOwnershipManifest(absHome, state.TargetPath, snapshot, toRestore)
	if err != nil {
		return mcpApplyRollback(absHome, result, snapshot, newBackupPath, mcpIntentRenamed, err)
	}
	if err := writeMCPOwnershipManifest(absHome, manifest); err != nil {
		return mcpApplyRollback(absHome, result, snapshot, newBackupPath, mcpIntentRenamed, err)
	}
	intent.Phase = mcpIntentManifestWritten
	intent.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	_ = mcpWriteIntent(newBackupPath, intent)
	if hook != nil && hook.AfterManifest != nil {
		if hookErr := hook.AfterManifest(); hookErr != nil {
			return mcpApplyRollback(absHome, result, snapshot, newBackupPath, mcpIntentManifestWritten, hookErr)
		}
	}
	intent.Phase = mcpIntentComplete
	intent.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	_ = mcpWriteIntent(newBackupPath, intent)

	result.Applied = mcpPlannedNames(toRestore)
	return result, nil
}

func mcpRestoreBeforeWrite(result MCPApplyResult, backupID string, cause error) (MCPApplyResult, error) {
	result.Applied = nil
	result.InstalledDigest = ""
	return result, fmt.Errorf(
		"mcp restore failed before writing the configuration; nothing changed (transaction phase %s, backup %s): %w",
		mcpIntentPrepared, backupID, cause)
}

// mcpRestoreOwnershipManifest updates ownership after a restore: restored names
// carry the restored digest, removed names lose ownership, and untouched names
// keep their recorded identity.
func mcpRestoreOwnershipManifest(home, target string, snapshot mcpSnapshot, restored []MCPPlanEntry) (mcpOwnershipManifest, error) {
	previous, err := readMCPOwnershipManifest(home)
	if err != nil {
		return mcpOwnershipManifest{}, err
	}
	manifest := mcpOwnershipManifest{
		Format:          mcpManifestFormat,
		ManifestVersion: mcpManifestVersion,
		ToolVersion:     version.Version,
		Created:         time.Now().UTC().Format(time.RFC3339Nano),
		Target:          target,
		TargetFormat:    mcpFormatJSON,
		Schema:          mcpSchemaV2,
		BaseDigest:      snapshot.BaseDigest,
		InstalledDigest: snapshot.InstalledDigest,
		BackupID:        snapshot.BackupID,
		BackupPath:      filepath.Join(mcpBackupsRoot(home), snapshot.BackupID),
		RuntimeVersion:  "not-checked",
		RuntimeEvidence: "not-checked",
		Actions:         append([]string(nil), snapshot.Actions...),
	}
	restoredDigest := map[string]string{}
	for _, entry := range restored {
		if entry.NewText == "" {
			restoredDigest[entry.Name] = ""
			continue
		}
		restoredDigest[entry.Name] = entry.NewHash
	}
	seen := map[string]bool{}
	if previous != nil && previous.Target == target {
		for _, previousEntry := range previous.Entries {
			if seen[previousEntry.Name] {
				continue
			}
			seen[previousEntry.Name] = true
			if digest, ok := restoredDigest[previousEntry.Name]; ok {
				if digest == "" {
					continue
				}
				previousEntry.InstalledDigest = digest
				previousEntry.BackupID = snapshot.BackupID
			}
			manifest.Entries = append(manifest.Entries, previousEntry)
		}
	}
	for _, entry := range restored {
		if seen[entry.Name] || entry.NewText == "" {
			continue
		}
		seen[entry.Name] = true
		manifest.Entries = append(manifest.Entries, mcpManifestEntry{
			Name:            entry.Name,
			InstalledDigest: entry.NewHash,
			BackupID:        snapshot.BackupID,
		})
	}
	manifest.Entries = mcpSortManifestEntries(manifest.Entries)
	return manifest, nil
}

// mcpRestorePlan reads the backup and current target and compares managed
// entries only. It returns the read-only preview and the state an apply needs.
func mcpRestorePlan(home, backupID, configPath string) (*MCPRestorePreview, *mcpRestoreState, error) {
	if home == "" {
		return nil, nil, fmt.Errorf("%w: home directory is required", ErrMCPUsage)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: resolve home directory: %v", ErrMCPUsage, err)
	}
	if err := mcpValidateBackupID(backupID); err != nil {
		return nil, nil, err
	}
	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: resolve config path: %v", ErrMCPUsage, err)
		}
	}

	backupPath := filepath.Join(mcpBackupsRoot(absHome), backupID)
	snapshot, err := mcpReadSnapshot(backupPath, backupID)
	if err != nil {
		return nil, nil, err
	}
	if snapshot.TargetFormat != mcpFormatJSON {
		return nil, nil, fmt.Errorf("%w: backups of %s targets cannot be restored by this build",
			ErrMCPUnsupported, snapshot.TargetFormat)
	}

	preview := &MCPRestorePreview{BackupID: backupID, TargetPath: snapshot.Target}
	state := &mcpRestoreState{
		Home:       absHome,
		BackupID:   backupID,
		BackupPath: backupPath,
		TargetPath: snapshot.Target,
		Format:     snapshot.TargetFormat,
		Snapshot:   snapshot,
	}

	expectedPath, expectedFormat, err := mcpExpectedTarget(absHome, absConfig)
	if err != nil {
		// An ambiguous or missing target is a restore conflict, not a crash:
		// report it on the preview so the caller can render it.
		if errors.Is(err, ErrMCPConflict) {
			preview.Conflicts = append(preview.Conflicts, err.Error())
			return preview, state, nil
		}
		return nil, nil, err
	}
	if expectedPath != snapshot.Target {
		preview.Conflicts = append(preview.Conflicts,
			fmt.Sprintf("backup target %s does not match the current target %s", snapshot.Target, expectedPath))
		return preview, state, nil
	}
	if expectedFormat != mcpFormatJSON {
		return nil, nil, fmt.Errorf("%w: %s targets cannot be restored by this build",
			ErrMCPUnsupported, expectedFormat)
	}
	state.TargetPath = expectedPath
	state.Format = expectedFormat

	data, mode, err := mcpReadConfigBytes(expectedPath)
	if err != nil {
		return nil, nil, err
	}
	state.Bytes = data
	state.Mode = mode
	state.BaseDigest = mcpBytesDigest(data)
	preview.BaseDigest = state.BaseDigest

	root, err := mcpParseDocument(expectedPath, mcpFormatJSON, data)
	if err != nil {
		return nil, nil, err
	}
	servers, hasMCP, hasServers, err := mcpServersOf(root)
	if err != nil {
		return nil, nil, err
	}
	state.Root = root
	state.HasMCP = hasMCP
	state.HasServers = hasServers

	for _, snapEntry := range snapshot.Entries {
		planEntry, conflict := mcpRestoreEntryFor(snapEntry, servers)
		if conflict != "" {
			preview.Conflicts = append(preview.Conflicts, conflict)
		}
		preview.Entries = append(preview.Entries, planEntry)
	}
	return preview, state, nil
}

// mcpRestoreState is the read-only state a restore apply re-reads and verifies.
type mcpRestoreState struct {
	Home       string
	BackupID   string
	BackupPath string
	TargetPath string
	Format     string
	Bytes      []byte
	Mode       os.FileMode
	BaseDigest string
	Root       map[string]any
	HasMCP     bool
	HasServers bool
	Snapshot   mcpSnapshot
}

// mcpRestoreEntryFor compares one recorded entry with the current document.
func mcpRestoreEntryFor(snapEntry mcpSnapshotEntry, servers map[string]any) (MCPPlanEntry, string) {
	entry := MCPPlanEntry{Name: snapEntry.Name}
	current, present := servers[snapEntry.Name]
	if !present {
		if snapEntry.Op == mcpOpAdd {
			entry.Op = mcpOpLeave
			entry.Reason = "entry is already absent; nothing to restore"
			return entry, ""
		}
		if snapEntry.Op == mcpOpRemove && snapEntry.OldText != "" {
			entry.Op = mcpOpRestore
			entry.NewText = snapEntry.OldText
			entry.NewHash = snapEntry.OldHash
			entry.Reason = "re-add the entry an earlier restore removed"
			return entry, ""
		}
		entry.Op = mcpOpConflict
		entry.Reason = fmt.Sprintf("managed entry %q is missing; restore refuses to recreate a server the user removed", snapEntry.Name)
		return entry, entry.Reason
	}
	currentMap, ok := current.(map[string]any)
	if !ok {
		entry.Op = mcpOpConflict
		entry.Reason = fmt.Sprintf("current entry %q is not a JSON object", snapEntry.Name)
		return entry, entry.Reason
	}
	currentText, currentHash, err := mcpCanonicalEntry(currentMap)
	if err != nil {
		entry.Op = mcpOpConflict
		entry.Reason = fmt.Sprintf("cannot read the current entry %q", snapEntry.Name)
		return entry, entry.Reason
	}
	entry.OldText = currentText
	entry.OldHash = currentHash

	if snapEntry.NewHash != "" && currentHash == snapEntry.NewHash {
		entry.Op = mcpOpRestore
		entry.NewText = snapEntry.OldText
		entry.NewHash = snapEntry.OldHash
		if snapEntry.OldText == "" {
			entry.Reason = "remove the entry Agentic-SDD added"
		} else {
			entry.Reason = "restore the entry text recorded before apply"
		}
		return entry, ""
	}
	if snapEntry.OldHash != "" && currentHash == snapEntry.OldHash {
		entry.Op = mcpOpLeave
		entry.NewText = snapEntry.OldText
		entry.NewHash = snapEntry.OldHash
		entry.Reason = "already restored"
		return entry, ""
	}
	entry.Op = mcpOpConflict
	entry.Reason = fmt.Sprintf(
		"managed entry %q changed after apply; restore refuses to erase a later user edit", snapEntry.Name)
	return entry, entry.Reason
}

// ReadMCPTransactionIntent returns the durable intent record for one backup.
func ReadMCPTransactionIntent(home, backupID string) (*MCPTransactionIntent, error) {
	if home == "" {
		return nil, fmt.Errorf("%w: home directory is required", ErrMCPUsage)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve home directory: %v", ErrMCPUsage, err)
	}
	if err := mcpValidateBackupID(backupID); err != nil {
		return nil, err
	}
	backupPath := filepath.Join(mcpBackupsRoot(absHome), backupID)
	path := filepath.Join(backupPath, mcpIntentFile)
	data, err := mcpReadPrivateJSON(path)
	if err != nil {
		return nil, err
	}
	var record mcpIntentRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("%w: invalid transaction intent at %s: %v", ErrMCPConflict, path, err)
	}
	if record.BackupID != backupID {
		return nil, fmt.Errorf("%w: transaction intent at %s records backup %q",
			ErrMCPConflict, path, record.BackupID)
	}
	return &MCPTransactionIntent{
		BackupID:            record.BackupID,
		TargetPath:          record.TargetPath,
		BaseDigest:          record.BaseDigest,
		PlanDigest:          record.PlanDigest,
		PriorManifestDigest: record.PriorManifestDigest,
		Phase:               record.Phase,
	}, nil
}

// mcpValidateBackupID rejects traversal and unknown identifiers.
func mcpValidateBackupID(backupID string) error {
	if backupID == "" {
		return fmt.Errorf("%w: a backup ID is required", ErrMCPUsage)
	}
	if backupID != filepath.Base(backupID) || backupID == "." || backupID == ".." ||
		strings.ContainsAny(backupID, `/\`) {
		return fmt.Errorf("%w: unusable backup ID %q", ErrMCPUnsafeTarget, backupID)
	}
	for _, r := range backupID {
		valid := r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == '.' || r == '-' || r == '_'
		if !valid {
			return fmt.Errorf("%w: unusable backup ID %q", ErrMCPUnsafeTarget, backupID)
		}
	}
	return nil
}

// acquireMCPLock takes the shared, cross-process engine lock covering the
// target and ownership manifest. A lock older than 30 seconds is treated as
// abandoned by a crashed writer and reclaimed once.
func acquireMCPLock(home string) (*mcpLock, error) {
	dir := mcpStateDir(home)
	if err := mcpEnsurePrivateDir(dir); err != nil {
		return nil, err
	}
	lockPath := mcpLockPath(home)
	for attempt := 0; attempt < 2; attempt++ {
		token := fmt.Sprintf("%d\n%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			if _, writeErr := f.WriteString(token); writeErr != nil {
				_ = f.Close()
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("write MCP lock %s: %w", lockPath, writeErr)
			}
			if syncErr := f.Sync(); syncErr != nil {
				_ = f.Close()
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("sync MCP lock %s: %w", lockPath, syncErr)
			}
			if closeErr := f.Close(); closeErr != nil {
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("close MCP lock %s: %w", lockPath, closeErr)
			}
			return &mcpLock{path: lockPath, token: token}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("acquire MCP lock %s: %w", lockPath, err)
		}
		info, statErr := os.Stat(lockPath)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("inspect MCP lock %s: %w", lockPath, statErr)
		}
		age := time.Since(info.ModTime())
		if age < mcpLockStaleAfter {
			return nil, fmt.Errorf(
				"%w: %w: lock held at %s for %s; wait for the competing writer or retry after it exits",
				ErrMCPConflict, ErrMCPLocked, lockPath, age.Round(time.Second))
		}
		if removeErr := os.Remove(lockPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %w: remove stale lock %s: %v",
				ErrMCPConflict, ErrMCPLocked, lockPath, removeErr)
		}
	}
	return nil, fmt.Errorf("%w: %w: could not acquire %s", ErrMCPConflict, ErrMCPLocked, lockPath)
}

// mcpLock is one acquired lock handle.
type mcpLock struct {
	path  string
	token string
}

// Release removes the lock file only when it still carries this acquisition's
// token, so a lock re-created by another process is never disturbed.
func (l *mcpLock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	data, err := os.ReadFile(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if string(data) != l.token {
		return nil
	}
	if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// mcpEnsurePrivateDir creates a 0700 directory that is a real directory, never
// a symlink.
func mcpEnsurePrivateDir(dir string) error {
	info, err := os.Lstat(dir)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s is a symlink", ErrMCPUnsafeTarget, dir)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: %s is not a directory", ErrMCPUnsafeTarget, dir)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	info, err = os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", dir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: %s is not a usable private directory", ErrMCPUnsafeTarget, dir)
	}
	return nil
}

// probeMCPRuntime runs one bounded local version probe and requires OpenCode
// V2. It never uses a shell, the network or a model.
func probeMCPRuntime(openCodeBin string) (path, versionText string, err error) {
	bin := strings.TrimSpace(openCodeBin)
	if bin == "" {
		resolved, lookErr := exec.LookPath("opencode")
		if lookErr != nil {
			return "", "", fmt.Errorf(
				"%w: no local opencode executable was found; pass --opencode-bin: %v", ErrMCPUnsupported, lookErr)
		}
		bin = resolved
	} else {
		info, statErr := os.Stat(bin)
		if statErr != nil {
			return "", "", fmt.Errorf(
				"%w: cannot inspect the OpenCode executable %s: %v", ErrMCPUnsupported, bin, statErr)
		}
		if info.IsDir() {
			return "", "", fmt.Errorf("%w: the OpenCode executable path is a directory: %s", ErrMCPUnsupported, bin)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), mcpRuntimeProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--version")
	stdout := &mcpLimitedBuffer{limit: 4096}
	stderr := &mcpLimitedBuffer{limit: 4096}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = nil
	if runErr := cmd.Run(); runErr != nil {
		if ctx.Err() != nil {
			return "", "", fmt.Errorf("%w: opencode --version timed out after %s",
				ErrMCPUnsupported, mcpRuntimeProbeTimeout)
		}
		return "", "", fmt.Errorf("%w: opencode --version failed (%v): %s",
			ErrMCPUnsupported, runErr, mcpTrim(stderr.String(), 200))
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = strings.TrimSpace(stderr.String())
	}
	parsed, major, ok := mcpParseVersion(output)
	if !ok {
		return "", "", fmt.Errorf("%w: could not determine the OpenCode version from %q",
			ErrMCPUnsupported, mcpTrim(output, 120))
	}
	if major < 2 {
		return "", "", fmt.Errorf("%w: OpenCode %s is older than the required V2 runtime",
			ErrMCPUnsupported, parsed)
	}
	return bin, parsed, nil
}

var mcpVersionPattern = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)

// mcpParseVersion extracts the first dotted version and its major number.
func mcpParseVersion(output string) (string, int, bool) {
	match := mcpVersionPattern.FindString(output)
	if match == "" {
		return "", 0, false
	}
	parts := strings.SplitN(match, ".", 2)
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", 0, false
	}
	return match, major, true
}

// mcpLimitedBuffer bounds captured process output.
type mcpLimitedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (b *mcpLimitedBuffer) Write(p []byte) (int, error) {
	if remaining := b.limit - b.buf.Len(); remaining > 0 {
		if len(p) <= remaining {
			b.buf.Write(p)
		} else {
			b.buf.Write(p[:remaining])
		}
	}
	return len(p), nil
}

func (b *mcpLimitedBuffer) String() string { return b.buf.String() }

func mcpTrim(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= limit {
		return trimmed
	}
	return trimmed[:limit] + "..."
}

// mcpStageTarget writes the new bytes to a same-directory 0600 temp file,
// fsyncs it and applies the original permission bits before the caller renames.
func mcpStageTarget(path string, data []byte, mode os.FileMode) (string, error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".agentic-sdd-mcp-*")
	if err != nil {
		return "", fmt.Errorf("stage MCP configuration in %s: %w", dir, err)
	}
	name := f.Name()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", fmt.Errorf("write staged MCP configuration: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", fmt.Errorf("sync staged MCP configuration: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close staged MCP configuration: %w", err)
	}
	if mode != 0 {
		if err := os.Chmod(name, mode); err != nil {
			_ = os.Remove(name)
			return "", fmt.Errorf("preserve configuration mode: %w", err)
		}
	}
	return name, nil
}

// mcpWriteFileAtomic writes a private state file or a restored target through a
// same-directory temp file, fsync and rename.
func mcpWriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".agentic-sdd-mcp-state-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	_ = mcpSyncDir(dir)
	return nil
}

// mcpSyncDir fsyncs a directory where the platform supports it.
func mcpSyncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	syncErr := f.Sync()
	closeErr := f.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

// mcpWriteSnapshot writes snapshot.json with private permissions.
func mcpWriteSnapshot(backupPath string, snapshot mcpSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return mcpWriteFileAtomic(filepath.Join(backupPath, mcpSnapshotFile), append(data, '\n'), 0600)
}

// mcpReadSnapshot reads and validates one backup snapshot.
func mcpReadSnapshot(backupPath, backupID string) (mcpSnapshot, error) {
	path := filepath.Join(backupPath, mcpSnapshotFile)
	data, err := mcpReadPrivateJSON(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return mcpSnapshot{}, fmt.Errorf("%w: no MCP backup %q at %s",
				ErrMCPConflict, backupID, backupPath)
		}
		return mcpSnapshot{}, err
	}
	var snapshot mcpSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return mcpSnapshot{}, fmt.Errorf("%w: invalid MCP backup %s: %v", ErrMCPConflict, path, err)
	}
	if snapshot.Format != mcpSnapshotFormat {
		return mcpSnapshot{}, fmt.Errorf("%w: unsupported MCP backup format %d at %s",
			ErrMCPUnsupported, snapshot.Format, path)
	}
	if snapshot.BackupID != backupID {
		return mcpSnapshot{}, fmt.Errorf("%w: MCP backup %s records backup ID %q",
			ErrMCPConflict, path, snapshot.BackupID)
	}
	return snapshot, nil
}

// mcpWriteIntent records the durable transaction phase.
func mcpWriteIntent(backupPath string, intent mcpIntentRecord) error {
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return err
	}
	return mcpWriteFileAtomic(filepath.Join(backupPath, mcpIntentFile), append(data, '\n'), 0600)
}

// mcpUpdateIntentPhase rewrites the phase of the existing intent record,
// preserving the digests it already carries.
func mcpUpdateIntentPhase(backupPath, phase string) {
	path := filepath.Join(backupPath, mcpIntentFile)
	record := mcpIntentRecord{Phase: phase}
	if data, err := os.ReadFile(path); err == nil {
		var existing mcpIntentRecord
		if json.Unmarshal(data, &existing) == nil {
			record = existing
		}
	}
	record.Phase = phase
	record.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	_ = mcpWriteIntent(backupPath, record)
}

// mcpReadPrivateJSON reads a bounded private JSON state file.
func mcpReadPrivateJSON(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file", ErrMCPUnsafeTarget, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := readAllBounded(f, maxMCPConfigFileBytes)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// readAllBounded reads at most limit bytes and rejects larger input.
func readAllBounded(f *os.File, limit int) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(io.LimitReader(f, int64(limit)+1)); err != nil {
		return nil, err
	}
	if buf.Len() > limit {
		return nil, fmt.Errorf("%w: private state file exceeds %d bytes", ErrMCPConflict, limit)
	}
	return buf.Bytes(), nil
}

// readMCPOwnershipManifest reads <home>/.agentic-sdd/mcp/manifest.json when it
// exists. A missing manifest is not an error; an unsupported format is.
func readMCPOwnershipManifest(home string) (*mcpOwnershipManifest, error) {
	path := mcpOwnershipPath(home)
	data, err := mcpReadPrivateJSON(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var manifest mcpOwnershipManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("%w: invalid MCP ownership manifest %s: %v", ErrMCPConflict, path, err)
	}
	if manifest.Format != mcpManifestFormat {
		return nil, fmt.Errorf("%w: unsupported MCP ownership manifest format %d at %s",
			ErrMCPUnsupported, manifest.Format, path)
	}
	return &manifest, nil
}

// writeMCPOwnershipManifest writes the ownership manifest atomically at 0600.
func writeMCPOwnershipManifest(home string, manifest mcpOwnershipManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := mcpEnsurePrivateDir(mcpStateDir(home)); err != nil {
		return err
	}
	return mcpWriteFileAtomic(mcpOwnershipPath(home), append(data, '\n'), 0600)
}
