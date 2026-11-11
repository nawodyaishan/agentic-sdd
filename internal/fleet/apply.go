package fleet

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"agentic-sdd/internal/skillsync"
)

var (
	// ErrPlanConflict reports a plan whose operations require a human
	// decision; apply never resolves one speculatively.
	ErrPlanConflict = errors.New("plan contains conflicts")
	// ErrUnsafeTarget reports a refused unmanaged collision, symlink,
	// special file or traversal target.
	ErrUnsafeTarget = errors.New("unsafe fleet target")
)

// ApplyResult describes the transaction an apply performed.
type ApplyResult struct {
	PlanID     string
	BackupID   string
	BackupPath string
	LivePath   string
	Created    int
	Updated    int
	Removed    int
	NoOp       bool
	Rollback   skillsync.RollbackInfo
}

type applyChange struct {
	file         string
	op           ProposedOp
	target       string
	data         []byte
	old          bool
	remove       bool
	backupDigest string
	provenance   string
}

// ApplyPlan revalidates a saved plan against current config, source and target
// state, then writes exactly the planned managed files through the shared
// skillsync backup/stage/commit/rollback engine. It never writes AGENTS.md,
// never follows a symlink, and never invents a disabled: field.
func ApplyPlan(plan *Plan, home string, out io.Writer) (ApplyResult, error) {
	result := ApplyResult{}
	if plan == nil {
		return result, errors.New("fleet apply: plan is nil")
	}
	result.PlanID = plan.PlanID

	absHome, err := filepath.Abs(home)
	if err != nil {
		return result, err
	}
	homeInfo, err := os.Lstat(absHome)
	if err != nil {
		return result, fmt.Errorf("fleet apply: %w", err)
	}
	if !homeInfo.IsDir() {
		return result, fmt.Errorf("%w: home %s is not a directory", ErrUnsafeTarget, absHome)
	}
	backupRoot := skillsync.BackupRootFor("", absHome)
	lock, err := AcquireLock(backupRoot)
	if err != nil {
		return result, err
	}
	defer func() { _ = lock.Release() }()

	if !plan.VerifyPlanID() {
		return result, fmt.Errorf("%w: plan ID mismatch (corrupted or altered plan)", ErrStalePlan)
	}
	if plan.RendererVersion != RendererVersion {
		return result, fmt.Errorf("%w: renderer version changed (plan %s, current %s)", ErrStalePlan, plan.RendererVersion, RendererVersion)
	}
	targetRoot := TargetRootForHome(absHome)
	if filepath.Clean(plan.TargetRoot) != filepath.Clean(targetRoot) {
		return result, fmt.Errorf("%w: target root changed (plan %s, current %s)", ErrStalePlan, plan.TargetRoot, targetRoot)
	}
	if plan.Scope != "" && plan.Scope != ScopeGlobal {
		return result, fmt.Errorf("%w: unsupported scope %q", ErrStalePlan, plan.Scope)
	}
	if err := checkTargetParents(targetRoot, absHome); err != nil {
		return result, err
	}
	if plan.HasConflicts() {
		return result, fmt.Errorf("%w: %s", ErrPlanConflict, conflictFiles(plan))
	}
	if plan.ConfigPath == "" {
		return result, fmt.Errorf("%w: plan has no config path; regenerate it", ErrStalePlan)
	}

	cfg, _, configDigest, err := LoadConfigFile(plan.ConfigPath)
	if err != nil {
		return result, fmt.Errorf("%w: reload configuration: %v", ErrStalePlan, err)
	}
	if configDigest != plan.ConfigDigest {
		return result, fmt.Errorf("%w: configuration changed (plan %s, current %s)", ErrStalePlan, plan.ConfigDigest, configDigest)
	}
	pack, err := skillsync.OpenCodePackFS(plan.RepoPath)
	if err != nil {
		return result, fmt.Errorf("fleet apply: open canonical OpenCode pack: %w", err)
	}
	resolved, err := Resolve(pack, cfg)
	if err != nil {
		return result, err
	}
	if got := sourceRevision(resolved); got != plan.SourceRev {
		return result, fmt.Errorf("%w: canonical source changed (plan %s, current %s)", ErrStalePlan, plan.SourceRev, got)
	}

	rendered := make(map[string]Rendered, len(resolved))
	for _, r := range resolved {
		if !r.Enabled {
			continue
		}
		rendered[r.InstanceID], err = RenderResolved(r)
		if err != nil {
			return result, err
		}
	}

	opByFile, err := indexPlanOps(plan)
	if err != nil {
		return result, err
	}
	if err := validatePlanSymmetry(plan, resolved, rendered, opByFile); err != nil {
		return result, err
	}

	currentHashes, err := CurrentTargetHashes(plan.TargetRoot, plan.FileList)
	if err != nil {
		return result, err
	}
	for _, f := range plan.FileList {
		expected, ok := plan.LastKnownInstalled[f]
		if !ok || currentHashes[f] != expected {
			return result, fmt.Errorf("%w: target file %q changed since the plan was generated", ErrStalePlan, f)
		}
	}

	live, err := ReadLiveManifest(absHome)
	if err != nil {
		return result, err
	}
	if live == nil {
		// Recover recorded ownership instead of treating every managed file
		// as an unmanaged collision when the live pointer is missing.
		live, err = RebuildLiveManifest(absHome)
		if err != nil {
			return result, err
		}
	}
	if err := validateLiveManifest(live, absHome, targetRoot); err != nil {
		return result, err
	}

	var changes []applyChange
	for _, op := range plan.ProposedOps {
		target, err := targetPathFor(targetRoot, op.File)
		if err != nil {
			return result, err
		}
		rec, hasRec := live.RecordFor(target)
		current := currentHashes[op.File]
		exists := current != ""
		switch op.Op {
		case "noop":
			continue
		case "create", "update":
			out := rendered[op.InstanceID]
			if out.DesiredDigest == "" {
				return result, fmt.Errorf("%w: plan instance %q is not enabled", ErrPlanConflict, op.InstanceID)
			}
			provenance, err := rederiveWrite(op, rec, hasRec, exists, current, out.DesiredDigest, target)
			if err != nil {
				return result, err
			}
			changes = append(changes, applyChange{
				file:         op.File,
				op:           op,
				target:       target,
				data:         out.Bytes,
				old:          exists,
				backupDigest: current,
				provenance:   provenance,
			})
		case "remove":
			if !exists {
				continue
			}
			if !hasRec || !rec.IsManaged() {
				return result, fmt.Errorf("%w: refusing to remove unmanaged file %s", ErrUnsafeTarget, target)
			}
			if current != rec.DesiredDigest && !op.Adopt {
				return result, fmt.Errorf("%w: refusing to remove drifted managed file %s without explicit adoption", ErrUnsafeTarget, target)
			}
			changes = append(changes, applyChange{
				file:         op.File,
				op:           op,
				target:       target,
				old:          true,
				remove:       true,
				backupDigest: current,
				provenance:   ProvenanceRemoved,
			})
		default:
			return result, fmt.Errorf("%w: unknown plan operation %q for %s", ErrPlanConflict, op.Op, op.File)
		}
	}

	if len(changes) == 0 {
		result.NoOp = true
		fmt.Fprintln(out, "No changes: every managed agent profile already matches the plan.")
		return result, nil
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].file < changes[j].file })

	now := time.Now().UTC()
	manifest := &Manifest{
		Format:          ManifestFormat,
		Operation:       "apply",
		State:           ManifestStateApplying,
		CreatedAt:       now.Format(time.RFC3339Nano),
		Home:            absHome,
		BackupRoot:      backupRoot,
		TargetRoot:      targetRoot,
		Scope:           ScopeGlobal,
		SourceRev:       plan.SourceRev,
		ConfigDigest:    plan.ConfigDigest,
		RendererVersion: plan.RendererVersion,
		Files:           transactionRecords(changes, plan),
	}
	hook := func(backupDir string) ([]byte, error) {
		applying := *manifest
		applying.ID = filepath.Base(backupDir)
		applying.BackupPath = backupDir
		if err := verifyBackupCopies(backupDir, applying.Files); err != nil {
			return nil, err
		}
		return marshalManifest(&applying)
	}

	engineChanges := make([]skillsync.AgentFileChange, 0, len(changes))
	for _, c := range changes {
		engineChanges = append(engineChanges, skillsync.AgentFileChange{
			File:   c.file,
			Data:   c.data,
			Old:    c.old,
			Remove: c.remove,
		})
	}
	written, applyErr := skillsync.ApplyAgentFiles(backupRoot, absHome, "fleet", engineChanges, hook, out)
	result.BackupID = written.BackupID
	result.BackupPath = written.BackupDir
	result.Rollback = written.Rollback
	if applyErr != nil {
		if written.BackupDir != "" {
			failed := *manifest
			failed.ID = written.BackupID
			failed.BackupPath = written.BackupDir
			failed.State = ManifestStateFailed
			failed.Rollback = &RollbackState{
				Attempted: written.Rollback.Attempted,
				Complete:  written.Rollback.Complete,
				Error:     written.Rollback.Error,
			}
			_ = writeManifestFile(written.BackupDir, &failed)
			return result, fmt.Errorf("fleet apply: %w (recoverable backup: %s; rollback attempted=%t complete=%t)", applyErr, written.BackupDir, written.Rollback.Attempted, written.Rollback.Complete)
		}
		return result, fmt.Errorf("fleet apply: %w", applyErr)
	}

	applied := *manifest
	applied.ID = written.BackupID
	applied.BackupPath = written.BackupDir
	applied.State = ManifestStateApplied
	for i := range applied.Files {
		if applied.Files[i].Provenance == ProvenanceRemoved {
			continue
		}
		applied.Files[i].InstalledDigest = applied.Files[i].DesiredDigest
	}
	if err := verifyInstalled(applied.Files); err != nil {
		failed := applied
		failed.State = ManifestStateFailed
		_ = writeManifestFile(written.BackupDir, &failed)
		return result, fmt.Errorf("%w (backup is recoverable at %s)", err, written.BackupDir)
	}
	if err := writeManifestFile(written.BackupDir, &applied); err != nil {
		return result, fmt.Errorf("fleet apply: write fleet manifest: %w", err)
	}
	merged := MergeLive(live, &applied)
	if err := WriteLiveManifest(absHome, merged); err != nil {
		return result, fmt.Errorf("fleet apply: write live manifest: %w", err)
	}
	result.LivePath = LivePointerPath(absHome)

	for _, c := range changes {
		switch {
		case c.remove:
			result.Removed++
		case c.old:
			result.Updated++
		default:
			result.Created++
		}
	}
	fmt.Fprintf(out, "Applied %d change(s): %d created, %d updated, %d removed.\n", len(changes), result.Created, result.Updated, result.Removed)
	fmt.Fprintf(out, "Backup: %s\n", written.BackupDir)
	fmt.Fprintf(out, "Live fleet manifest: %s\n", result.LivePath)
	return result, nil
}

func indexPlanOps(plan *Plan) (map[string]ProposedOp, error) {
	byFile := make(map[string]ProposedOp, len(plan.ProposedOps))
	for _, op := range plan.ProposedOps {
		if _, err := targetPathFor(plan.TargetRoot, op.File); err != nil {
			return nil, err
		}
		if _, dup := byFile[op.File]; dup {
			return nil, fmt.Errorf("%w: duplicate plan operation for %s", ErrPlanConflict, op.File)
		}
		byFile[op.File] = op
	}
	listed := make(map[string]bool, len(plan.FileList))
	for _, f := range plan.FileList {
		if listed[f] {
			return nil, fmt.Errorf("%w: duplicate plan file list entry %q", ErrPlanConflict, f)
		}
		listed[f] = true
		if _, ok := byFile[f]; !ok {
			return nil, fmt.Errorf("%w: plan file list entry %q has no operation", ErrPlanConflict, f)
		}
	}
	for f := range byFile {
		if !listed[f] {
			return nil, fmt.Errorf("%w: plan operation for %q is missing from the file list", ErrPlanConflict, f)
		}
	}
	return byFile, nil
}

// validatePlanSymmetry confirms the saved plan still describes exactly the
// resolved instances it claims: one file, one operation, one source reference
// and one source digest per instance, and no extra entries in the file list or
// last-known map. A resealed plan can change any field, so matching the plan
// against freshly resolved and rendered state is what makes the plan a
// consistency token rather than an authorization.
func validatePlanSymmetry(plan *Plan, resolved []Resolved, rendered map[string]Rendered, opByFile map[string]ProposedOp) error {
	if len(opByFile) != len(resolved) || len(plan.FileList) != len(resolved) {
		return fmt.Errorf("%w: plan file list no longer matches the configuration", ErrStalePlan)
	}
	if len(plan.LastKnownInstalled) != len(plan.FileList) {
		return fmt.Errorf("%w: plan last-known map does not match its file list", ErrStalePlan)
	}
	for _, r := range resolved {
		file := r.InstanceID + ".md"
		op, ok := opByFile[file]
		if !ok || op.InstanceID != r.InstanceID {
			return fmt.Errorf("%w: plan is missing instance %q", ErrStalePlan, r.InstanceID)
		}
		ref := SourceRef{Kind: r.SourceKind, Name: r.SourceName}.String()
		if op.SourceRef != ref {
			return fmt.Errorf("%w: plan source reference for %s changed (plan %s, current %s)", ErrStalePlan, file, op.SourceRef, ref)
		}
		if op.SourceDigest != r.SourceDigest {
			return fmt.Errorf("%w: plan source digest for %s no longer matches the canonical source", ErrStalePlan, file)
		}
		if op.Model != r.Model {
			return fmt.Errorf("%w: plan model for %s changed since the plan", ErrStalePlan, file)
		}
		if _, ok := plan.LastKnownInstalled[file]; !ok {
			return fmt.Errorf("%w: plan last-known map is missing %s", ErrStalePlan, file)
		}
		if !r.Enabled {
			if op.DesiredDigest != "" {
				return fmt.Errorf("%w: disabled instance %q carries a desired change", ErrStalePlan, r.InstanceID)
			}
			continue
		}
		if op.DesiredDigest != rendered[r.InstanceID].DesiredDigest {
			return fmt.Errorf("%w: rendered bytes for %s changed since the plan", ErrStalePlan, file)
		}
	}
	return nil
}

// rederiveWrite re-derives the three-state drift decision from the current
// disk content and the live sidecar at apply time. The plan's own drift fields
// are informational: a resealed plan cannot turn a manual edit into an
// overwrite. A managed file is written only while it still matches its last
// applied digest or already matches the desired bytes; unmanaged or
// legacy-owned content is overridden only through an explicit adoption, and
// content that already equals the desired bytes is adopted without losing
// anything.
func rederiveWrite(op ProposedOp, rec FileRecord, hasRec, exists bool, current, desired, target string) (string, error) {
	managed := hasRec && rec.IsManaged()
	switch {
	case !exists:
		if managed {
			return "", fmt.Errorf("%w: managed file %s is missing; restore it or resolve its ownership before applying", ErrPlanConflict, target)
		}
		return ProvenanceManaged, nil
	case managed && current == rec.DesiredDigest:
		return rec.Provenance, nil
	case managed && current == desired:
		return rec.Provenance, nil
	case current == desired:
		// The bytes are already exactly the desired content: recording
		// ownership changes no user data.
		return ProvenanceManaged, nil
	case op.Adopt && managed:
		return "", fmt.Errorf("%w: adoption cannot overwrite manually drifted managed content %s", ErrUnsafeTarget, target)
	case op.Adopt:
		return ProvenanceAdopted, nil
	case managed:
		return "", fmt.Errorf("%w: refusing to overwrite manually drifted managed file %s without an explicit decision", ErrUnsafeTarget, target)
	default:
		return "", fmt.Errorf("%w: refusing to overwrite unmanaged file %s without explicit adoption", ErrUnsafeTarget, target)
	}
}

func conflictFiles(plan *Plan) string {
	var files []string
	for _, op := range plan.ProposedOps {
		if op.Op == "conflict" {
			files = append(files, op.File)
		}
	}
	sort.Strings(files)
	return fmt.Sprintf("resolve %v before applying (run fleet plan again after review)", files)
}

func transactionRecords(changes []applyChange, plan *Plan) []FileRecord {
	records := make([]FileRecord, 0, len(changes))
	for _, c := range changes {
		record := FileRecord{
			InstanceID:      c.op.InstanceID,
			SourceRef:       c.op.SourceRef,
			SourceDigest:    c.op.SourceDigest,
			ConfigDigest:    plan.ConfigDigest,
			RendererVersion: plan.RendererVersion,
			Model:           c.op.Model,
			Scope:           ScopeGlobal,
			Target:          c.target,
			Provenance:      c.provenance,
		}
		if c.remove {
			record.Provenance = ProvenanceRemoved
		} else {
			record.DesiredDigest = c.op.DesiredDigest
		}
		if c.old {
			record.BackupRef = filepath.Join("opencode-agents", c.file)
			record.BackupDigest = c.backupDigest
		}
		records = append(records, record)
	}
	return records
}

// verifyInstalled confirms the bytes actually written match the sidecar's
// claim before the live pointer is updated. A mismatch leaves the applied
// sidecar marked failed and the backup recoverable instead of claiming a
// clean install.
func verifyInstalled(files []FileRecord) error {
	for _, r := range files {
		if r.Provenance == ProvenanceRemoved {
			if _, exists, err := onDiskDigest(r.Target); err != nil {
				return err
			} else if exists {
				return fmt.Errorf("managed removal %s is still present", r.Target)
			}
			continue
		}
		digest, exists, err := onDiskDigest(r.Target)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("installed file %s is missing", r.Target)
		}
		if digest != r.InstalledDigest {
			return fmt.Errorf("installed file %s does not match its recorded digest", r.Target)
		}
	}
	return nil
}
