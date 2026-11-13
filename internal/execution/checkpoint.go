package execution

// Observed checkpoint state, freshness reconciliation and the run status
// result. A checkpoint is disposable observation: it never stores human
// approval, never authorizes a resume, and never claims a next batch.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Freshness reasons. Only progress_updated is informational; every other
// reason blocks trust in earlier evidence.
const (
	ReasonSpecChanged           = "spec_changed"
	ReasonTasksChanged          = "tasks_changed"
	ReasonBatchChanged          = "batch_changed"
	ReasonHeadChanged           = "head_changed"
	ReasonWorktreeChanged       = "worktree_changed"
	ReasonScopeMismatch         = "scope_mismatch"
	ReasonMissingArtifact       = "missing_artifact"
	ReasonInterruptedCheck      = "interrupted_check"
	ReasonStaleLock             = "stale_lock"
	ReasonContradictoryEvidence = "contradictory_evidence"
	ReasonProgressUpdated       = "progress_updated"
)

// Permitted next investigations. These are read-only observations, never an
// automatic resume or a new batch.
const (
	InvestigateNone            = "none"
	InvestigateChangedPaths    = "inspect_changed_paths"
	InvestigateRerunCheck      = "rerun_authorized_check"
	InvestigateReconcileSource = "reconcile_source_change"
	InvestigateResolveBlocker  = "resolve_blocker"
)

// statusKinds are the stable JSON kinds emitted by this package.
const (
	kindRunStatus = "execution.run.status"
	kindRunClean  = "execution.run.clean"
)

var stateOrder = []CheckpointState{
	StateBlocked,
	StateRunning,
	StateStale,
	StateObservedFailed,
	StateObservedPartial,
	StateAwaitingVerification,
	StateVerifiedEvidence,
	StatePrepared,
}

// HighestState applies the documented precedence: blocked, running, stale,
// observed-failed, observed-partial, awaiting-verification, verified-evidence,
// prepared.
func HighestState(states ...CheckpointState) CheckpointState {
	best := StatePrepared
	bestRank := 0
	for _, state := range states {
		if rank := stateRank(state); rank > bestRank {
			best = state
			bestRank = rank
		}
	}
	return best
}

func stateRank(state CheckpointState) int {
	for i, known := range stateOrder {
		if known == state {
			return len(stateOrder) - i
		}
	}
	return 0
}

func isKnownState(state CheckpointState) bool {
	return stateRank(state) > 0
}

// NormalizedBatchScopeDigest normalizes the selected batch section and returns its
// digest. Checkbox and status-only progress changes do not alter the digest;
// requirement, boundary and batch changes do.
func NormalizedBatchScopeDigest(data []byte, batchID string) (string, error) {
	if err := validateToken("batch ID", batchID); err != nil {
		return "", err
	}
	lines := splitNormalizedLines(data)
	start, end, err := locateBatchSection(lines, batchID)
	if err != nil {
		return "", err
	}
	return ScopeSectionDigest([]byte(strings.Join(lines[start:end], "\n"))), nil
}

// ScopeSectionDigest digests a normalized task or batch section. Trailing
// whitespace, checkbox state and status/state/next-action/completion lines are
// ignored.
func ScopeSectionDigest(section []byte) string {
	lines := splitNormalizedLines(section)
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if isProgressOnlyLine(line) {
			continue
		}
		kept = append(kept, strings.TrimRight(normalizeScopeCheckbox(line), " \t"))
	}
	return DigestBytes([]byte(strings.Join(kept, "\n")))
}

func splitNormalizedLines(data []byte) []string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

func isProgressOnlyLine(line string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(line))
	for _, prefix := range []string{"**status:**", "**state:**", "**next action:**", "**completion:**"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func normalizeScopeCheckbox(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	if len(trimmed) >= 5 && strings.HasPrefix(trimmed, "- [") &&
		(trimmed[3] == 'x' || trimmed[3] == 'X') && trimmed[4] == ']' {
		return line[:indent] + "- [ ]" + trimmed[5:]
	}
	return line
}

// locateBatchSection finds exactly one level-2 batch heading matching batchID,
// ignoring headings inside fenced code blocks. The section ends at the next
// level-2 heading or EOF. Missing and duplicate batches fail closed.
func locateBatchSection(lines []string, batchID string) (int, int, error) {
	headings := make([]int, 0, 8)
	fenceChar := byte(0)
	fenceLen := 0
	for index, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)
		if indent <= 3 {
			if char, count, ok := fenceMarker(trimmed); ok {
				switch {
				case fenceChar == 0:
					fenceChar = char
					fenceLen = count
				case char == fenceChar && count >= fenceLen && strings.TrimSpace(trimmed[count:]) == "":
					fenceChar = 0
					fenceLen = 0
				}
				continue
			}
		}
		if fenceChar != 0 || indent > 3 {
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			headings = append(headings, index)
		}
	}
	matches := make([]int, 0, 1)
	for _, index := range headings {
		if matchBatchHeading(lines[index], batchID) {
			matches = append(matches, index)
		}
	}
	switch len(matches) {
	case 0:
		return 0, 0, fmt.Errorf("%w: selected batch is not present in tasks.md", ErrNotFound)
	case 1:
	default:
		return 0, 0, fmt.Errorf("%w: selected batch heading is duplicated in tasks.md", ErrConflict)
	}
	start := matches[0]
	end := len(lines)
	for _, index := range headings {
		if index > start {
			end = index
			break
		}
	}
	return start, end, nil
}

func matchBatchHeading(line, batchID string) bool {
	trimmed := strings.TrimLeft(line, " ")
	if !strings.HasPrefix(trimmed, "## ") {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(trimmed[3:]))
	if len(fields) == 0 {
		return false
	}
	if strings.TrimRight(fields[0], ":") == batchID {
		return true
	}
	return len(fields) >= 2 && fields[0] == "Batch" && strings.TrimRight(fields[1], ":") == batchID
}

func fenceMarker(trimmed string) (byte, int, bool) {
	if len(trimmed) < 3 {
		return 0, 0, false
	}
	char := trimmed[0]
	if char != '`' && char != '~' {
		return 0, 0, false
	}
	count := 0
	for count < len(trimmed) && trimmed[count] == char {
		count++
	}
	if count < 3 {
		return 0, 0, false
	}
	return char, count, true
}

// LoadCheckpoint reads and validates checkpoint.json. A missing file returns an
// error wrapping fs.ErrNotExist so callers can synthesize a prepared state.
func LoadCheckpoint(runDir string) (*Checkpoint, error) {
	data, err := readBoundedRegular(filepath.Join(runDir, checkpointName), maxCheckpointBytes)
	if err != nil {
		return nil, err
	}
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("%w: checkpoint is not valid JSON", ErrUsage)
	}
	if checkpoint.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%w: unsupported checkpoint schema", ErrUnsupported)
	}
	if err := ValidateRunID(checkpoint.RunID); err != nil {
		return nil, err
	}
	if !isKnownState(checkpoint.State) {
		return nil, fmt.Errorf("%w: checkpoint state is not recognized", ErrUsage)
	}
	for taskID, task := range checkpoint.Tasks {
		if strings.TrimSpace(taskID) == "" || !isKnownState(task.State) {
			return nil, fmt.Errorf("%w: checkpoint task state is not recognized", ErrUsage)
		}
	}
	return &checkpoint, nil
}

// WriteCheckpoint atomically persists the checkpoint. The caller must hold the
// run lock; run.json must already exist and carry the same run ID.
func WriteCheckpoint(runDir string, checkpoint *Checkpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("%w: checkpoint is required", ErrUsage)
	}
	identity, err := LoadRunIdentity(runDir)
	if err != nil {
		return err
	}
	if checkpoint.RunID != identity.RunID {
		return fmt.Errorf("%w: checkpoint run ID does not match run identity", ErrConflict)
	}
	if checkpoint.SchemaVersion == 0 {
		checkpoint.SchemaVersion = SchemaVersion
	}
	if checkpoint.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: unsupported checkpoint schema", ErrUnsupported)
	}
	if checkpoint.State == "" {
		checkpoint.State = StatePrepared
	}
	if !isKnownState(checkpoint.State) {
		return fmt.Errorf("%w: checkpoint state is not recognized", ErrUsage)
	}
	for taskID, task := range checkpoint.Tasks {
		if strings.TrimSpace(taskID) == "" || !isKnownState(task.State) {
			return fmt.Errorf("%w: checkpoint task state is not recognized", ErrUsage)
		}
	}
	checkpoint.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("checkpoint encoding unavailable")
	}
	return writeFileAtomic(filepath.Join(runDir, checkpointName), append(data, '\n'), 0o600)
}

// StatusRun re-observes a run and returns its truthful state. The reported
// state is data, not approval, and the next permitted investigation is never an
// automatic resume.
func StatusRun(runRoot, runID string) (Status, error) {
	if err := ValidateRunID(runID); err != nil {
		return Status{}, err
	}
	root, err := canonicalRunRoot(runRoot)
	if err != nil {
		return Status{}, err
	}
	runDir := filepath.Join(root, runID)
	identity, err := LoadRunIdentity(runDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Status{}, fmt.Errorf("%w: run is not present under the selected run root", ErrNotFound)
		}
		return Status{}, err
	}
	if filepath.Clean(identity.RunRoot) != root {
		return Status{}, fmt.Errorf("%w: run identity does not match the selected run root", ErrConflict)
	}
	checkpoint, err := LoadCheckpoint(runDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return Status{}, err
		}
		baseline := identity.Baseline
		checkpoint = &Checkpoint{
			SchemaVersion:   SchemaVersion,
			RunID:           runID,
			UpdatedAt:       identity.CreatedAt,
			State:           StatePrepared,
			LastObservation: &baseline,
		}
	}
	status := Status{
		SchemaVersion:   SchemaVersion,
		Kind:            kindRunStatus,
		RunID:           runID,
		Scope:           identity.Scope,
		LastObservation: checkpoint.LastObservation,
		Checks:          checkpoint.Checks,
		Records:         checkpoint.Records,
		Blockers:        checkpoint.Blockers,
		RunDir:          runDir,
		UpdatedAt:       checkpoint.UpdatedAt,
	}
	if status.LastObservation == nil {
		baseline := identity.Baseline
		status.LastObservation = &baseline
	}
	freshness := evaluateFreshness(identity, checkpoint, runDir)
	status.CurrentObservation = freshness.current
	status.StaleReasons = freshness.reasons
	status.ProgressUpdated = freshness.progress
	status.State = effectiveState(checkpoint.State, freshness.reasons, freshness.lock)
	status.NextPermittedInvestigation = nextInvestigation(status.State, freshness.reasons)
	return status, nil
}

// freshness is the reconciliation result for one status observation.
type freshness struct {
	reasons  []string
	progress bool
	current  *GitObservation
	lock     lockObservation
}

// evaluateFreshness compares recorded identity and observations with the
// current source, Git identity, worktree and artifacts. It never mutates the
// checkpoint.
func evaluateFreshness(identity RunIdentity, checkpoint *Checkpoint, runDir string) freshness {
	result := freshness{}
	addReason := func(reason string) {
		for _, existing := range result.reasons {
			if existing == reason {
				return
			}
		}
		result.reasons = append(result.reasons, reason)
	}
	result.lock = inspectLock(runDir)
	if checkpoint.State == StateRunning {
		if !result.lock.exists {
			addReason(ReasonInterruptedCheck)
		} else if result.lock.stale {
			addReason(ReasonInterruptedCheck)
			addReason(ReasonStaleLock)
		}
	}
	for _, task := range checkpoint.Tasks {
		if task.BriefPath != "" && !regularFileExists(task.BriefPath) {
			addReason(ReasonMissingArtifact)
		}
	}
	for _, check := range checkpoint.Checks {
		for _, artifact := range []string{check.ArgvPath, check.StdoutPath, check.StderrPath} {
			if artifact != "" && !regularFileExists(artifact) {
				addReason(ReasonMissingArtifact)
			}
		}
	}
	for _, record := range checkpoint.Records {
		if record.RawPath != "" && !regularFileExists(record.RawPath) {
			addReason(ReasonMissingArtifact)
		}
	}
	repo, err := OpenRepository(identity.Scope.RepoRoot)
	if err != nil {
		addReason(ReasonScopeMismatch)
		return result
	}
	current, err := ObserveRun(repo, identity.Scope, runDir)
	if err != nil {
		addReason(ReasonContradictoryEvidence)
		return result
	}
	result.current = &current
	last := checkpoint.LastObservation
	if last == nil {
		last = &identity.Baseline
	}
	if last.Head != current.Head {
		addReason(ReasonHeadChanged)
	}
	if last.StatusDigest != current.StatusDigest ||
		last.DiffDigest != current.DiffDigest ||
		last.UntrackedDigest != current.UntrackedDigest {
		addReason(ReasonWorktreeChanged)
	}
	if identity.Scope.Kind == ScopeFeature {
		specPath := scopeSourcePath(identity.Scope.RepoRoot, identity.Scope.SpecPath)
		specData, specErr := readBoundedRegular(specPath, MaxSpecFileBytes)
		switch {
		case specErr != nil:
			addReason(ReasonMissingArtifact)
		case DigestBytes(specData) != identity.Scope.SpecDigest:
			addReason(ReasonSpecChanged)
		}
		tasksPath := scopeSourcePath(identity.Scope.RepoRoot, identity.Scope.TasksPath)
		tasksData, tasksErr := readBoundedRegular(tasksPath, MaxTasksFileBytes)
		switch {
		case tasksErr != nil:
			addReason(ReasonMissingArtifact)
		case DigestBytes(tasksData) != identity.Scope.TasksDigest:
			scopeDigest, digestErr := NormalizedBatchScopeDigest(tasksData, identity.Scope.BatchID)
			if digestErr != nil || scopeDigest != identity.Scope.BatchScopeDigest {
				addReason(ReasonTasksChanged)
				addReason(ReasonBatchChanged)
			} else {
				result.progress = true
			}
		}
	}
	if checkpoint.State == StateVerifiedEvidence && contradictoryEvidence(checkpoint) {
		addReason(ReasonContradictoryEvidence)
	}
	if result.progress {
		addReason(ReasonProgressUpdated)
	}
	return result
}

func scopeSourcePath(repoRoot, relative string) string {
	return filepath.Join(repoRoot, filepath.FromSlash(relative))
}

// scopeExclusions lists the canonical feature sources whose edits are judged by
// their own digests instead of as generic worktree drift.
func scopeExclusions(scope Scope) []string {
	if scope.Kind != ScopeFeature {
		return nil
	}
	exclusions := make([]string, 0, 2)
	if scope.SpecPath != "" {
		exclusions = append(exclusions, scope.SpecPath)
	}
	if scope.TasksPath != "" {
		exclusions = append(exclusions, scope.TasksPath)
	}
	return exclusions
}

// observationExclusions adds the run's own scratch root when it lives inside
// the observed worktree, so scratch never reports itself as project drift.
func observationExclusions(scope Scope, runDir string) []string {
	exclusions := scopeExclusions(scope)
	if scope.RepoRoot == "" || runDir == "" {
		return exclusions
	}
	scratchRoot := filepath.Dir(filepath.Clean(runDir))
	if !withinPath(scope.RepoRoot, scratchRoot) {
		return exclusions
	}
	relative, err := filepath.Rel(scope.RepoRoot, scratchRoot)
	if err != nil {
		return exclusions
	}
	relative = filepath.ToSlash(relative)
	if err := ValidateRelativePath(relative); err != nil {
		return exclusions
	}
	return append(exclusions, relative)
}

func regularFileExists(name string) bool {
	info, err := os.Lstat(name)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular()
}

func contradictoryEvidence(checkpoint *Checkpoint) bool {
	if len(checkpoint.Blockers) > 0 {
		return true
	}
	lastCheck := CheckResult("")
	for _, check := range checkpoint.Checks {
		lastCheck = check.Result
	}
	if lastCheck != CheckPassed {
		return true
	}
	for _, record := range checkpoint.Records {
		if record.Result != ClaimPassed {
			return true
		}
	}
	return false
}

// effectiveState applies the precedence to the recorded state, the freshness
// reasons and the observed lock. An interrupted check becomes observed-partial;
// source, identity or artifact drift makes the run stale; blocked always wins.
func effectiveState(base CheckpointState, reasons []string, lock lockObservation) CheckpointState {
	state := base
	if state == "" {
		state = StatePrepared
	}
	if state == StateRunning && (!lock.exists || lock.stale) {
		state = StateObservedPartial
	}
	blocking := false
	for _, reason := range reasons {
		switch reason {
		case ReasonProgressUpdated, ReasonInterruptedCheck, ReasonStaleLock:
			continue
		default:
			blocking = true
		}
	}
	if blocking {
		state = HighestState(state, StateStale)
	}
	return state
}

// nextInvestigation names the only permitted next investigation for a state.
func nextInvestigation(state CheckpointState, reasons []string) string {
	switch state {
	case StateBlocked:
		return InvestigateResolveBlocker
	case StateStale:
		for _, reason := range reasons {
			switch reason {
			case ReasonSpecChanged, ReasonTasksChanged, ReasonBatchChanged,
				ReasonScopeMismatch, ReasonMissingArtifact, ReasonContradictoryEvidence:
				return InvestigateReconcileSource
			}
		}
		for _, reason := range reasons {
			if reason == ReasonHeadChanged || reason == ReasonWorktreeChanged {
				return InvestigateChangedPaths
			}
		}
		return InvestigateReconcileSource
	case StateObservedFailed, StateObservedPartial:
		return InvestigateRerunCheck
	case StateAwaitingVerification:
		return InvestigateChangedPaths
	default:
		return InvestigateNone
	}
}
