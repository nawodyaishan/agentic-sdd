package fleet

import (
	"fmt"
	"os"
	"path/filepath"

	"agentic-sdd/internal/skillsync"
)

// InstalledState is the read-only installed-state label for one fleet
// instance. It is derived from the freshly rendered desired bytes, the target
// bytes observed on disk and the live fleet manifest. It is a projection, not
// an approval, a security verdict or a runtime claim.
type InstalledState string

const (
	// InstalledAbsent means no managed file and nothing to remove.
	InstalledAbsent InstalledState = "absent"
	// InstalledManagedCurrent means the installed bytes equal both the last
	// applied digest and the freshly rendered desired digest.
	InstalledManagedCurrent InstalledState = "managed-current"
	// InstalledManagedDrifted means a managed file is missing or no longer
	// matches the last applied digest. Overwriting it requires a human
	// decision; status never resolves that decision.
	InstalledManagedDrifted InstalledState = "managed-drifted"
	// InstalledPendingApply means the managed file still matches the last
	// applied digest while the desired render differs.
	InstalledPendingApply InstalledState = "pending-apply"
	// InstalledUnmanaged means a file exists that fleet does not own.
	InstalledUnmanaged InstalledState = "unmanaged"
	// InstalledLegacyUnknown means a live record exists whose provenance is
	// legacy or unknown, so no clean managed state can be fabricated.
	InstalledLegacyUnknown InstalledState = "legacy-unknown"
	// InstalledUnsafeTarget means the installed path is a symlink or another
	// special file; it is never followed.
	InstalledUnsafeTarget InstalledState = "unsafe-target"
	// InstalledDisabled means the instance is disabled while its managed file
	// is still installed, i.e. a managed removal is pending.
	InstalledDisabled InstalledState = "disabled"
)

// Status count keys that are always present in Status.Counts, including zero.
// Each InstalledState value is also a key, so a reader can iterate states
// without inventing labels. The aggregate keys are prefixed because the
// installed-state vocabulary already uses the bare word "disabled".
const (
	StatusCountInstances = "count:instances"
	StatusCountEnabled   = "count:enabled"
	StatusCountDisabled  = "count:disabled"
	StatusCountConflicts = "count:conflicts"
)

// allInstalledStates is the stable state vocabulary used to seed counts.
var allInstalledStates = []InstalledState{
	InstalledAbsent,
	InstalledManagedCurrent,
	InstalledManagedDrifted,
	InstalledPendingApply,
	InstalledUnmanaged,
	InstalledLegacyUnknown,
	InstalledUnsafeTarget,
	InstalledDisabled,
}

// InstanceStatus is one instance's read-only projection.
//
// ConfiguredModel is the personal override (empty means "inherit the canonical
// model"). RenderedModel is the model actually present in the freshly rendered
// bytes, so a canonical inherited model is visible without inventing an
// override. PriorModel is the model recorded by the last applied transaction
// or, when that record carries none, the model found in the current installed
// bytes; it is display-only and never an ownership claim. DesiredDigest is
// empty for a disabled instance because a disabled instance has no desired
// install.
type InstanceStatus struct {
	InstanceID        string
	SourceRef         string
	SourceDigest      string
	ConfiguredModel   string
	RenderedModel     string
	PriorModel        string
	Enabled           bool
	DesiredDigest     string
	CurrentDigest     string
	LastAppliedDigest string
	Installed         InstalledState
	Provenance        string
	Conflict          string
}

// Status is the typed, read-only fleet inventory. ConfigPath is the
// caller-supplied path unchanged so CLI text output stays stable; every other
// path and digest is resolved by this package.
type Status struct {
	ConfigPath   string
	ConfigDigest string
	TargetRoot   string
	Instances    []InstanceStatus
	Counts       map[string]int
}

// BuildStatus derives the read-only fleet inventory from configPath, the
// canonical pack at repo (empty means the embedded pack) and home. It uses
// LoadConfigFile, Resolve, RenderResolved, the live manifest (with the
// recorded-transaction rebuild fallback) and per-target Lstat only.
//
// It acquires no lock, builds no plan, applies nothing and writes nothing: no
// file, directory, backup, live pointer, Git, network or model operation is
// performed. A rendered-bytes failure is returned instead of being hidden, so
// an inventory is never invented from an unrenderable source.
func BuildStatus(configPath, repo, home string) (*Status, error) {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	cfg, _, configDigest, err := LoadConfigFile(configPath)
	if err != nil {
		return nil, err
	}
	absRepo := ""
	if repo != "" {
		absRepo, err = filepath.Abs(repo)
		if err != nil {
			return nil, err
		}
	}
	pack, err := skillsync.OpenCodePackFS(absRepo)
	if err != nil {
		return nil, fmt.Errorf("open canonical OpenCode pack: %w", err)
	}
	resolved, err := Resolve(pack, cfg)
	if err != nil {
		return nil, err
	}

	targetRoot := TargetRootForHome(absHome)
	if err := checkTargetParents(targetRoot, absHome); err != nil {
		return nil, err
	}
	live, err := ReadLiveManifest(absHome)
	if err != nil {
		return nil, err
	}
	if live == nil {
		// A lost live pointer must not turn managed files into unmanaged
		// collisions. Replay is read-only.
		live, err = RebuildLiveManifest(absHome)
		if err != nil {
			return nil, err
		}
	}
	if live != nil {
		if err := validateLiveManifest(live, absHome, targetRoot); err != nil {
			return nil, err
		}
		if err := live.Validate(live.ID, absHome, live.BackupRoot, targetRoot); err != nil {
			return nil, fmt.Errorf("live fleet manifest: %w", err)
		}
	}

	status := &Status{
		ConfigPath:   configPath,
		ConfigDigest: configDigest,
		TargetRoot:   targetRoot,
		Instances:    make([]InstanceStatus, 0, len(resolved)),
		Counts:       make(map[string]int, len(allInstalledStates)+4),
	}

	for _, r := range resolved {
		rendered, err := RenderResolved(r)
		if err != nil {
			return nil, err
		}
		target, err := targetPathFor(targetRoot, r.InstanceID+".md")
		if err != nil {
			return nil, err
		}
		currentDigest, exists, unsafe, err := inspectTargetFile(target)
		if err != nil {
			return nil, err
		}
		rec, hasRec := live.RecordFor(target)
		inst := InstanceStatus{
			InstanceID:      r.InstanceID,
			SourceRef:       SourceRef{Kind: r.SourceKind, Name: r.SourceName}.String(),
			SourceDigest:    r.SourceDigest,
			ConfiguredModel: r.Model,
			RenderedModel:   ExtractModelFromContent(rendered.Bytes),
			PriorModel:      priorInstalledModel(hasRec, rec, exists, target),
			Enabled:         r.Enabled,
			CurrentDigest:   currentDigest,
			Provenance:      ProvenanceUnknown,
		}
		if r.Enabled {
			inst.DesiredDigest = rendered.DesiredDigest
		}
		if hasRec {
			inst.Provenance = rec.Provenance
			if rec.IsManaged() {
				inst.LastAppliedDigest = rec.DesiredDigest
			}
		}
		inst.Installed, inst.Conflict = classifyInstalledState(inst, hasRec, rec, exists, unsafe)
		status.Instances = append(status.Instances, inst)
	}

	status.Counts[StatusCountInstances] = len(status.Instances)
	status.Counts[StatusCountEnabled] = 0
	status.Counts[StatusCountDisabled] = 0
	status.Counts[StatusCountConflicts] = 0
	for _, state := range allInstalledStates {
		status.Counts[string(state)] = 0
	}
	for _, inst := range status.Instances {
		status.Counts[string(inst.Installed)]++
		if inst.Enabled {
			status.Counts[StatusCountEnabled]++
		} else {
			status.Counts[StatusCountDisabled]++
		}
		if inst.Conflict != "" {
			status.Counts[StatusCountConflicts]++
		}
	}
	return status, nil
}

// priorInstalledModel mirrors the plan's prior-model projection: a recorded
// model wins, otherwise the currently installed bytes are inspected. It is a
// display label only and returns "" when neither source is readable.
func priorInstalledModel(hasRec bool, rec FileRecord, exists bool, target string) string {
	if hasRec && rec.Model != "" {
		return rec.Model
	}
	if exists {
		if content, err := os.ReadFile(target); err == nil {
			return ExtractModelFromContent(content)
		}
	}
	return ""
}

// inspectTargetFile returns the digest of a regular target file. A missing
// file reports ("", false, false, nil). A symlink or special file reports
// unsafe instead of being followed, which is what makes the unsafe-target
// state reportable rather than fatal.
func inspectTargetFile(path string) (digest string, exists, unsafe bool, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, false, nil
		}
		return "", false, false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, true, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, false, err
	}
	return digestBytes(data), true, false, nil
}

// classifyInstalledState mirrors the plan classifier's three-state table as a
// display-only projection. It deliberately reuses the plan's conflict wording
// so the inventory and a fresh plan do not tell different stories, and it
// never drops a conflict to make the summary look clean.
func classifyInstalledState(inst InstanceStatus, hasRec bool, rec FileRecord, exists, unsafe bool) (InstalledState, string) {
	if unsafe {
		return InstalledUnsafeTarget, "refusing special file or symlink at the installed path"
	}
	managed := hasRec && rec.IsManaged()
	if !inst.Enabled {
		switch {
		case managed && exists && inst.CurrentDigest == rec.DesiredDigest:
			return InstalledDisabled, ""
		case managed && exists:
			return InstalledDisabled, "managed file drifted from the last applied digest; use --adopt to remove it with a backup"
		case exists:
			return InstalledUnmanaged, "unmanaged file preserved (not removed)"
		default:
			return InstalledAbsent, ""
		}
	}
	switch {
	case managed && !exists:
		return InstalledManagedDrifted, "managed file is missing; restore it or remove its recorded ownership before applying"
	case managed && inst.CurrentDigest == rec.DesiredDigest && inst.DesiredDigest == inst.CurrentDigest:
		return InstalledManagedCurrent, ""
	case managed && inst.CurrentDigest == rec.DesiredDigest:
		return InstalledPendingApply, ""
	case managed:
		return InstalledManagedDrifted, "manual drift: installed file differs from the last applied digest"
	case hasRec:
		return InstalledLegacyUnknown, "legacy/unknown provenance; use --adopt after review"
	case exists:
		return InstalledUnmanaged, "unmanaged file already exists; use --adopt after review"
	default:
		return InstalledAbsent, ""
	}
}
