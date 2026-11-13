// Package execution provides the typed, standard-library service behind the
// public run, task and review CLI families. It observes local state and writes
// only private run artifacts; it never approves work, calls a model, or acts as
// a sandbox.
package execution

import (
	"errors"
	"time"
)

const (
	// SchemaVersion is the version of the run, checkpoint, record and review
	// contracts emitted by this package.
	SchemaVersion = 1

	// MaxTasksFileBytes bounds the canonical tasks.md read.
	MaxTasksFileBytes = 4 << 20
	// MaxSpecFileBytes bounds the canonical spec.md read.
	MaxSpecFileBytes = 8 << 20
	// MaxRecordBytes bounds one caller-supplied run record.
	MaxRecordBytes = 1 << 20
	// DefaultCheckTimeout is the default wall-clock limit for run check.
	DefaultCheckTimeout = 15 * time.Minute
	// DefaultMaxOutputBytes is the default per-stream stdout/stderr cap.
	DefaultMaxOutputBytes = 64 << 20
	// DefaultMaxFileBytes is the default per-file review-package cap.
	DefaultMaxFileBytes = 2 << 20
	// DefaultMaxTotalBytes is the default total review-package cap.
	DefaultMaxTotalBytes = 32 << 20
	// LockStaleAfter is the age after which an unowned lock may be reclaimed.
	LockStaleAfter = 30 * time.Minute
)

var (
	// ErrUsage marks invalid arguments or an invalid request.
	ErrUsage = errors.New("execution: usage")
	// ErrUnsafePath marks a path containment or symlink violation.
	ErrUnsafePath = errors.New("execution: unsafe path")
	// ErrNotFound marks a missing run, task, batch or artifact.
	ErrNotFound = errors.New("execution: not found")
	// ErrConflict marks a lock, identity or concurrent-writer conflict.
	ErrConflict = errors.New("execution: conflict")
	// ErrStale marks a checkpoint or observation that no longer matches.
	ErrStale = errors.New("execution: stale")
	// ErrPartial marks a deliberately partial result.
	ErrPartial = errors.New("execution: partial")
	// ErrUnsupported marks a capability this build does not provide.
	ErrUnsupported = errors.New("execution: unsupported")
)

// ScopeKind identifies a feature-backed or direct-scope run.
type ScopeKind string

const (
	ScopeFeature ScopeKind = "feature"
	ScopeDirect  ScopeKind = "direct"
)

// Scope is the observed source identity of a run. It is a reference, never an
// approval.
type Scope struct {
	Kind             ScopeKind `json:"kind"`
	RepoRoot         string    `json:"repo_root"`
	FeatureDir       string    `json:"feature_dir,omitempty"`
	SpecPath         string    `json:"spec_path,omitempty"`
	SpecDigest       string    `json:"spec_digest,omitempty"`
	TasksPath        string    `json:"tasks_path,omitempty"`
	TasksDigest      string    `json:"tasks_digest,omitempty"`
	BatchID          string    `json:"batch_id,omitempty"`
	BatchScopeDigest string    `json:"batch_scope_digest,omitempty"`
	Label            string    `json:"label,omitempty"`
}

// GitObservation is a point-in-time Git identity and worktree fingerprint.
// Ignored files are excluded. It is an observation, not a security boundary.
type GitObservation struct {
	Head            string    `json:"head"`
	Branch          string    `json:"branch,omitempty"`
	StatusDigest    string    `json:"status_digest,omitempty"`
	DiffDigest      string    `json:"diff_digest,omitempty"`
	UntrackedDigest string    `json:"untracked_digest,omitempty"`
	WorktreeDigest  string    `json:"worktree_digest"`
	GitVersion      string    `json:"git_version,omitempty"`
	ObservedAt      time.Time `json:"observed_at"`
}

// RunIdentity is the immutable run.json record.
type RunIdentity struct {
	SchemaVersion       int            `json:"schemaVersion"`
	Kind                string         `json:"kind"`
	RunID               string         `json:"run_id"`
	CreatedAt           time.Time      `json:"created_at"`
	ToolVersion         string         `json:"tool_version,omitempty"`
	RunRoot             string         `json:"run_root"`
	RunDir              string         `json:"run_dir"`
	Scope               Scope          `json:"scope"`
	Baseline            GitObservation `json:"baseline"`
	ApprovalReference   string         `json:"approval_reference,omitempty"`
	WorkspaceGitVisible bool           `json:"workspace_git_visible,omitempty"`
}

// InitRequest creates one private run workspace. Scope is prepared by the
// caller from FeatureScope or DirectScope; it is a reference, not approval.
type InitRequest struct {
	RunRoot           string `json:"run_root,omitempty"`
	Scope             Scope  `json:"scope"`
	ToolVersion       string `json:"tool_version,omitempty"`
	ApprovalReference string `json:"approval_reference,omitempty"`
}

// CheckpointState is an observed execution state. It never means human
// approval or a completed batch.
type CheckpointState string

const (
	StatePrepared             CheckpointState = "prepared"
	StateRunning              CheckpointState = "running"
	StateObservedPartial      CheckpointState = "observed-partial"
	StateObservedFailed       CheckpointState = "observed-failed"
	StateAwaitingVerification CheckpointState = "awaiting-verification"
	StateVerifiedEvidence     CheckpointState = "verified-evidence"
	StateStale                CheckpointState = "stale"
	StateBlocked              CheckpointState = "blocked"
)

// TaskState is the observed state for one task within a batch-level run.
type TaskState struct {
	TaskID       string          `json:"task_id"`
	State        CheckpointState `json:"state"`
	BriefPath    string          `json:"brief_path,omitempty"`
	BriefDigest  string          `json:"brief_digest,omitempty"`
	LastCheckID  string          `json:"last_check_id,omitempty"`
	LastRecordID string          `json:"last_record_id,omitempty"`
	UpdatedAt    time.Time       `json:"updated_at,omitempty"`
}

// Checkpoint is the mutable checkpoint.json record. It is disposable
// observation, not an approval or batch ledger.
type Checkpoint struct {
	SchemaVersion   int                   `json:"schemaVersion"`
	RunID           string                `json:"run_id"`
	UpdatedAt       time.Time             `json:"updated_at"`
	State           CheckpointState       `json:"state"`
	LastObservation *GitObservation       `json:"last_observation,omitempty"`
	Tasks           map[string]TaskState  `json:"tasks,omitempty"`
	Checks          []CheckRecord         `json:"checks,omitempty"`
	Records         []ClaimRecord         `json:"records,omitempty"`
	Blockers        []Blocker             `json:"blockers,omitempty"`
	ReviewPackages  []ReviewPackageRecord `json:"review_packages,omitempty"`
}

// CheckResult is the observed result of one local command.
type CheckResult string

const (
	CheckPassed      CheckResult = "passed"
	CheckFailed      CheckResult = "failed"
	CheckTimedOut    CheckResult = "timed_out"
	CheckInterrupted CheckResult = "interrupted"
	CheckOutputLimit CheckResult = "output_limit"
	CheckNotStarted  CheckResult = "not_started"
)

// CheckRecord is the metadata for one run check. The full argv is stored
// separately in the private run directory; summaries use ArgvDigest.
type CheckRecord struct {
	CheckID                    string      `json:"check_id"`
	TaskID                     string      `json:"task_id,omitempty"`
	Result                     CheckResult `json:"result"`
	CommandProgram             string      `json:"command_program"`
	CommandArgCount            int         `json:"command_arg_count"`
	ArgvDigest                 string      `json:"argv_digest"`
	ArgvPath                   string      `json:"argv_path"`
	WorkingDir                 string      `json:"working_dir"`
	StartedAt                  time.Time   `json:"started_at"`
	EndedAt                    *time.Time  `json:"ended_at,omitempty"`
	ExitCode                   *int        `json:"exit_code,omitempty"`
	Signal                     string      `json:"signal,omitempty"`
	TimedOut                   bool        `json:"timed_out"`
	OutputLimit                bool        `json:"output_limit"`
	StdoutPath                 string      `json:"stdout_path"`
	StderrPath                 string      `json:"stderr_path"`
	StdoutBytes                int64       `json:"stdout_bytes"`
	StderrBytes                int64       `json:"stderr_bytes"`
	StdoutTruncated            bool        `json:"stdout_truncated"`
	StderrTruncated            bool        `json:"stderr_truncated"`
	Summary                    []string    `json:"summary,omitempty"`
	SummaryTruncated           bool        `json:"summary_truncated"`
	WorktreeChangedDuringCheck bool        `json:"worktree_changed_during_check,omitempty"`
}

// ClaimResult is the caller-reported result. It remains a claim.
type ClaimResult string

const (
	ClaimPassed     ClaimResult = "passed"
	ClaimFailed     ClaimResult = "failed"
	ClaimPartial    ClaimResult = "partial"
	ClaimBlocked    ClaimResult = "blocked"
	ClaimNeedsInput ClaimResult = "needs_input"
)

// Blocker is an explicit reason execution cannot continue without a decision.
type Blocker struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// AgentIdentity is optional reported provenance; it is not verified.
type AgentIdentity struct {
	Role  string `json:"role,omitempty"`
	Model string `json:"model,omitempty"`
}

// ClaimCheck is a check reported by a caller. It is not observed evidence.
type ClaimCheck struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	LogPath  string `json:"log_path,omitempty"`
}

// ClaimRecord is one untrusted caller report persisted under records/.
type ClaimRecord struct {
	SchemaVersion int            `json:"schemaVersion"`
	RecordID      string         `json:"record_id"`
	TaskID        string         `json:"task_id"`
	Result        ClaimResult    `json:"result"`
	Summary       string         `json:"summary"`
	ChangedPaths  []string       `json:"changed_paths,omitempty"`
	Checks        []ClaimCheck   `json:"checks,omitempty"`
	Blockers      []Blocker      `json:"blockers,omitempty"`
	Agent         *AgentIdentity `json:"agent,omitempty"`
	Unknowns      []string       `json:"unknowns,omitempty"`
	ReceivedAt    time.Time      `json:"received_at"`
	RawPath       string         `json:"raw_path"`
	RawDigest     string         `json:"raw_digest"`
}

// ReviewFile is one changed path in a review package manifest.
type ReviewFile struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Staged    bool   `json:"staged,omitempty"`
	Unstaged  bool   `json:"unstaged,omitempty"`
	Untracked bool   `json:"untracked,omitempty"`
	Binary    bool   `json:"binary,omitempty"`
	Submodule bool   `json:"submodule,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Digest    string `json:"digest,omitempty"`
	Included  bool   `json:"included"`
	Reason    string `json:"reason,omitempty"`
}

// ReviewOmission explains content left out of a review package.
type ReviewOmission struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// ReviewManifest is review.json.
type ReviewManifest struct {
	SchemaVersion  int              `json:"schemaVersion"`
	Kind           string           `json:"kind"`
	PackageID      string           `json:"package_id"`
	RunID          string           `json:"run_id"`
	CreatedAt      time.Time        `json:"created_at"`
	RepoRoot       string           `json:"repo_root"`
	Base           string           `json:"base"`
	Head           string           `json:"head"`
	HeadRequested  string           `json:"head_requested,omitempty"`
	Branch         string           `json:"branch,omitempty"`
	GitVersion     string           `json:"git_version,omitempty"`
	Scope          Scope            `json:"scope"`
	Complete       bool             `json:"complete"`
	PartialReasons []string         `json:"partial_reasons,omitempty"`
	Files          []ReviewFile     `json:"files,omitempty"`
	Omissions      []ReviewOmission `json:"omissions,omitempty"`
	Counts         ReviewCounts     `json:"counts"`
	PatchPath      string           `json:"patch_path"`
	UntrackedDir   string           `json:"untracked_dir,omitempty"`
	RunState       CheckpointState  `json:"run_state,omitempty"`
	StaleReasons   []string         `json:"stale_reasons,omitempty"`
}

// ReviewCounts summarizes a review package.
type ReviewCounts struct {
	Committed int `json:"committed"`
	Staged    int `json:"staged"`
	Unstaged  int `json:"unstaged"`
	Untracked int `json:"untracked"`
	Omitted   int `json:"omitted"`
}

// ReviewPackageRecord is the checkpoint reference to a package.
type ReviewPackageRecord struct {
	PackageID      string    `json:"package_id"`
	ManifestPath   string    `json:"manifest_path"`
	PatchPath      string    `json:"patch_path"`
	Complete       bool      `json:"complete"`
	PartialReasons []string  `json:"partial_reasons,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Status is the run status result. It is an observation, never approval.
type Status struct {
	SchemaVersion              int             `json:"schemaVersion"`
	Kind                       string          `json:"kind"`
	RunID                      string          `json:"run_id"`
	Scope                      Scope           `json:"scope"`
	State                      CheckpointState `json:"state"`
	StaleReasons               []string        `json:"stale_reasons,omitempty"`
	ProgressUpdated            bool            `json:"progress_updated,omitempty"`
	LastObservation            *GitObservation `json:"last_observation,omitempty"`
	CurrentObservation         *GitObservation `json:"current_observation,omitempty"`
	Checks                     []CheckRecord   `json:"checks,omitempty"`
	Records                    []ClaimRecord   `json:"records,omitempty"`
	Blockers                   []Blocker       `json:"blockers,omitempty"`
	NextPermittedInvestigation string          `json:"next_permitted_investigation"`
	RunDir                     string          `json:"run_dir"`
	UpdatedAt                  time.Time       `json:"updated_at"`
}

// CleanResult previews or applies removal of exactly one owned run directory.
type CleanResult struct {
	SchemaVersion int      `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	RunID         string   `json:"run_id"`
	RunRoot       string   `json:"run_root"`
	RunDir        string   `json:"run_dir"`
	Applied       bool     `json:"applied"`
	Removed       bool     `json:"removed"`
	AlreadyAbsent bool     `json:"already_absent"`
	FileCount     int      `json:"file_count"`
	TotalBytes    int64    `json:"total_bytes"`
	Entries       []string `json:"entries,omitempty"`
	LockState     string   `json:"lock_state,omitempty"`
}

// Brief is the deterministic task brief result.
type Brief struct {
	SchemaVersion int       `json:"schemaVersion"`
	Kind          string    `json:"kind"`
	RunID         string    `json:"run_id"`
	TaskID        string    `json:"task_id"`
	BriefPath     string    `json:"brief_path"`
	BriefDigest   string    `json:"brief_digest"`
	SourcePath    string    `json:"source_path"`
	LineStart     int       `json:"line_start"`
	LineEnd       int       `json:"line_end"`
	References    []string  `json:"references,omitempty"`
	GeneratedAt   time.Time `json:"generated_at"`
}
