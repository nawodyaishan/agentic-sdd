package tui

import (
	"context"

	"agentic-sdd/internal/fleet"
)

// FleetReader is the read-only projection interface for Fleet Manager status and inventory.
// Implementations must perform only read operations; no file mutation, CLI subprocess,
// or model calls are permitted.
type FleetReader interface {
	Status(ctx context.Context, source string) (FleetStatusView, error)
}

// FleetPlanner is the read-only deployment and recovery projection seam. Every
// method performs a read-only engine call: PlanFiles and PreviewRestoreBackup
// never write, lock or acquire a transaction. Implementations must never call
// ApplyPlan/RestoreBackup, spawn a CLI or parse CLI prose.
type FleetPlanner interface {
	// PreviewPlan builds a fresh non-adopting plan from the saved source and
	// returns the exact per-file old-versus-desired projection plus its token.
	PreviewPlan(ctx context.Context, source string) (FleetPlanPreview, error)
	// ListBackups lists managed fleet backups only.
	ListBackups(ctx context.Context) ([]FleetBackupView, error)
	// PreviewRestore reports exactly what restoring one managed backup would do.
	PreviewRestore(ctx context.Context, backupID string) (FleetRestorePreview, error)
}

// FleetStatusView is the immutable typed view-model for Fleet Manager.
type FleetStatusView struct {
	Available             bool
	SourcePath            string
	SourceState           string // "not selected", "missing", "invalid", "loaded", "not available"
	PersonalConfigSummary string // e.g. "Loaded (18 instances configured)"
	InstalledSummary      string // e.g. "14 current, 2 drifted, 2 missing"
	TotalInstances        int
	ManagedCount          int
	EnabledCount          int
	DisabledCount         int
	DriftedCount          int
	UnmanagedCount        int
	LastOutcome           string // e.g. "None"
	NextAction            string // one recommended safe next action
	RuntimeStatus         string // "not checked" (truthful offline default)
	Message               string
	Instances             []FleetInstanceView
}

// FleetInstanceView represents one agent instance in the fleet.
type FleetInstanceView struct {
	InstanceID        string
	Role              string
	ProviderRoute     string
	Model             string
	CompactModel      string
	Variant           string
	Enabled           bool
	InstalledState    string // "managed current", "drifted", "not installed", "pending apply", "unmanaged", "legacy unknown", "disabled (managed removal)"
	SourceProvenance  string // e.g. "opencode-pack/agents", "personal override", "preset: free-first"
	ChangedCount      int
	PermissionSummary string // engine-derived summary, e.g. "tools: bash, read, edit; network: local"
	RuntimeState      string // "not checked" / "unverified"
	DriftDetail       string

	// Raw engine facts retained for the editor and details drill-down. They
	// are display-only: no screen may derive ownership, digest or plan identity
	// from them.
	SourceRef         string
	ConfiguredModel   string
	SourceDigest      string
	DesiredDigest     string
	CurrentDigest     string
	LastAppliedDigest string
	Provenance        string
}

// FleetFileDiff is one read-only file projection with bounded line diff rows.
type FleetFileDiff struct {
	InstanceID string
	Path       string
	Op         string
	Reason     string
	Removed    bool
	Conflict   bool
	OldDigest  string
	NewDigest  string
	DiffLines  []string
}

// FleetPlanPreview is the read-only deployment review. It carries the fresh
// plan identity, the preview token and the complete per-file projection
// (removals/conflicts first). It is never user approval.
type FleetPlanPreview struct {
	Available       bool
	SourcePath      string
	PlanID          string
	Scope           string
	ConfigDigest    string
	SourceRev       string
	RendererVersion string
	Token           string
	Counts          map[string]int
	Files           []FleetFileDiff
	DiffLines       []string
	Message         string
	Plan            *fleet.Plan
}

// FleetBackupView is one managed fleet backup listed by the recovery view.
type FleetBackupView struct {
	ID        string
	Path      string
	Operation string
	State     string
	CreatedAt string
	Files     int
}

// FleetRestorePreview is the read-only current-versus-restore projection for
// one managed backup.
type FleetRestorePreview struct {
	Available  bool
	BackupID   string
	BackupPath string
	Counts     map[string]int
	Files      []FleetFileDiff
	DiffLines  []string
	Token      string
	Message    string
}

// DefaultFleetReader is the safe default reader when no production appops adapter is provided.
// It returns a truthful not-available state without accessing the filesystem or spawning processes.
type DefaultFleetReader struct{}

// Status returns a clear not-available status view.
func (d DefaultFleetReader) Status(ctx context.Context, source string) (FleetStatusView, error) {
	state := "not selected"
	summary := "No fleet.json selected — press s to specify an explicit path"
	if source != "" {
		state = "not available"
		summary = "Fleet engine adapter not connected in this build"
	}
	return FleetStatusView{
		Available:             false,
		SourcePath:            source,
		SourceState:           state,
		PersonalConfigSummary: summary,
		InstalledSummary:      "No managed instances inspected",
		RuntimeStatus:         "not checked",
		NextAction:            "Inspect local profiles or wait for engine integration",
		Message:               "Fleet operations are not available in this build (adapter pending).",
	}, nil
}

// DefaultFleetPlanner is the safe default deployment/recovery projection. It
// returns truthful not-available results without reading any file.
type DefaultFleetPlanner struct{}

// PreviewPlan returns a not-available deployment preview.
func (d DefaultFleetPlanner) PreviewPlan(ctx context.Context, source string) (FleetPlanPreview, error) {
	return FleetPlanPreview{
		Available:  false,
		SourcePath: source,
		Message:    "Deployment planning is not available in this build (adapter pending).",
	}, nil
}

// ListBackups returns no backups without inspecting the filesystem.
func (d DefaultFleetPlanner) ListBackups(ctx context.Context) ([]FleetBackupView, error) {
	return nil, nil
}

// PreviewRestore returns a not-available restore preview.
func (d DefaultFleetPlanner) PreviewRestore(ctx context.Context, backupID string) (FleetRestorePreview, error) {
	return FleetRestorePreview{
		Available: false,
		BackupID:  backupID,
		Message:   "Restore preview is not available in this build (adapter pending).",
	}, nil
}
