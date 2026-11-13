package execution

// Local command execution with truthful evidence: explicit argv (no shell),
// repository-root working directory, bounded process-group cancellation, capped
// stdout/stderr logs, observed exit/signal and a compact failure excerpt. A check
// is an observation of one command in the current worktree. It is never
// approval, never a sandbox, and never a claim about the work it tests.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	checkIDPrefix = "chk"

	minCheckMaxOutputBytes = 1 << 10 // 1 KiB
	maxCheckMaxOutputBytes = 1 << 30 // 1 GiB
	maxCheckArgvBytes      = 1 << 20 // 1 MiB of private command provenance
	maxCheckArgvArgs       = 4096    // one vector, no shell expansion
	checkSummaryMatchLimit = 40      // failure-marker lines kept
	checkSummaryTailLines  = 20      // fallback tail lines when no marker
	checkSummaryLineBytes  = 400     // per-excerpt line bound
	checkErrorBytes        = 2000    // bounded metadata/error text
	checkWaitDelay         = 2 * time.Second

	stdoutLogName = "stdout.log"
	stderrLogName = "stderr.log"
	checkMetaName = "meta.json"
	checkArgvName = "argv.json"

	kindCheckMeta = "execution.run.check.meta"
	kindCheckArgv = "execution.run.check.argv"
)

// CheckRunning is the in-progress observation written when a check starts. It
// is never a final outcome: the observed result replaces it when the command
// ends, and it survives only when the writer was interrupted before a final
// observation. A running record is evidence that no result was observed.
const CheckRunning CheckResult = "running"

// checkFailureMarkers are case-insensitive excerpts of a failing command's
// output. They are only a summary cue; the capped logs remain authoritative.
var checkFailureMarkers = []string{
	"fail",
	"error",
	"panic",
	"not ok",
	"fatal",
	"cannot",
	"undefined",
}

// CheckRequest executes one already-authorized local command inside a run. RunDir
// is the run directory whose run.json carries the repository identity. Argv is
// the exact argument vector, with no shell, glob or variable interpolation.
// Timeout <= 0 disables the wall-clock limit; MaxOutputBytes <= 0 selects
// DefaultMaxOutputBytes and is clamped to 1 KiB..1 GiB per stream.
type CheckRequest struct {
	RunDir         string
	RunID          string
	TaskID         string
	Argv           []string
	Timeout        time.Duration
	MaxOutputBytes int64
}

// checkArgvDocument is the private provenance file for one check. It keeps the
// exact vector because CheckRecord stores only the digest and program basename.
type checkArgvDocument struct {
	SchemaVersion int      `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	Digest        string   `json:"digest"`
	Argv          []string `json:"argv"`
}

// checkMeta is the private meta.json for one check. It carries the observed
// record plus the start/end worktree observations, the configured limits and the
// exact execution error, which never belongs in the run-level record.
type checkMeta struct {
	SchemaVersion    int             `json:"schemaVersion"`
	Kind             string          `json:"kind"`
	Check            CheckRecord     `json:"check"`
	Error            string          `json:"error,omitempty"`
	Finished         bool            `json:"finished"`
	DurationMillis   int64           `json:"duration_ms"`
	TimeoutMillis    int64           `json:"timeout_ms"`
	MaxOutputBytes   int64           `json:"max_output_bytes"`
	StartObservation *GitObservation `json:"start_observation,omitempty"`
	EndObservation   *GitObservation `json:"end_observation,omitempty"`
}

// checkRun carries one in-flight check within a locked run.
type checkRun struct {
	runDir     string
	repo       *Repository
	scope      Scope
	checkpoint *Checkpoint
	argv       []string
	record     CheckRecord
	meta       checkMeta
	metaPath   string
	startObs   GitObservation
	timeout    time.Duration
	maxOutput  int64
}

// RunCheck executes one explicit local command within a run, persists truthful
// evidence under checks/<check-id>/, and updates the observed checkpoint. It
// requires a non-empty argv and never invokes a shell, expands globs or
// interpolates variables. The command runs in the repository root recorded by the
// run identity, with the invoker's permissions.
//
// A returned error means no trustworthy evidence record could be produced or
// persisted; the caller must treat it as a runtime/usage failure. An observed
// command outcome (passed, failed, timed_out, interrupted, output_limit,
// not_started) is returned with a nil error, and the only success claim is
// CheckPassed. A failed, timed-out or interrupted check never yields
// verified-evidence.
func RunCheck(req CheckRequest) (CheckRecord, error) {
	if err := validateCheckArgv(req.Argv); err != nil {
		return CheckRecord{}, err
	}
	if req.Timeout < 0 {
		return CheckRecord{}, fmt.Errorf("%w: timeout must not be negative", ErrUsage)
	}
	if strings.TrimSpace(req.RunDir) == "" {
		return CheckRecord{}, fmt.Errorf("%w: run directory is required", ErrUsage)
	}
	runDir, err := filepath.Abs(req.RunDir)
	if err != nil {
		return CheckRecord{}, fmt.Errorf("%w: run directory is invalid", ErrUsage)
	}
	runDir = filepath.Clean(runDir)
	identity, err := LoadRunIdentity(runDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return CheckRecord{}, fmt.Errorf("%w: run is not present at the selected directory", ErrNotFound)
		}
		return CheckRecord{}, err
	}
	if req.RunID != "" {
		if err := ValidateRunID(req.RunID); err != nil {
			return CheckRecord{}, err
		}
		if req.RunID != identity.RunID {
			return CheckRecord{}, fmt.Errorf("%w: run identity does not match the requested run", ErrConflict)
		}
	}
	repo, err := OpenRepository(identity.Scope.RepoRoot)
	if err != nil {
		return CheckRecord{}, err
	}
	if err := sameRoot(repo.Root, identity.Scope.RepoRoot); err != nil {
		return CheckRecord{}, err
	}
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" && identity.Scope.Kind == ScopeDirect {
		taskID = "direct"
	}
	if taskID != "" {
		if err := validateToken("task ID", taskID); err != nil {
			return CheckRecord{}, err
		}
	}
	lock, err := acquireRunLock(runDir, "check")
	if err != nil {
		return CheckRecord{}, err
	}
	defer func() { _ = lock.release() }()

	checkpoint, err := LoadCheckpoint(runDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return CheckRecord{}, err
		}
		baseline := identity.Baseline
		checkpoint = &Checkpoint{
			SchemaVersion:   SchemaVersion,
			RunID:           identity.RunID,
			UpdatedAt:       time.Now().UTC(),
			State:           StatePrepared,
			LastObservation: &baseline,
		}
	}
	if err := ensureArtifactDir(runDir, "checks"); err != nil {
		return CheckRecord{}, err
	}
	startedAt := time.Now().UTC()
	checkID, checkDir, err := createCheckDir(runDir, startedAt)
	if err != nil {
		return CheckRecord{}, err
	}
	run := &checkRun{
		runDir:     runDir,
		repo:       repo,
		scope:      identity.Scope,
		checkpoint: checkpoint,
		argv:       req.Argv,
		metaPath:   filepath.Join(checkDir, checkMetaName),
		timeout:    req.Timeout,
		maxOutput:  normalizeCheckMaxOutputBytes(req.MaxOutputBytes),
	}
	run.record = CheckRecord{
		CheckID:         checkID,
		TaskID:          taskID,
		Result:          CheckRunning,
		CommandProgram:  filepath.Base(req.Argv[0]),
		CommandArgCount: len(req.Argv) - 1,
		ArgvDigest:      checkArgvDigest(req.Argv),
		ArgvPath:        filepath.Join(checkDir, checkArgvName),
		WorkingDir:      repo.Root,
		StartedAt:       startedAt,
		StdoutPath:      filepath.Join(checkDir, stdoutLogName),
		StderrPath:      filepath.Join(checkDir, stderrLogName),
	}
	argvDocument, err := json.MarshalIndent(checkArgvDocument{
		SchemaVersion: SchemaVersion,
		Kind:          kindCheckArgv,
		Digest:        run.record.ArgvDigest,
		Argv:          append([]string(nil), req.Argv...),
	}, "", "  ")
	if err != nil {
		return CheckRecord{}, fmt.Errorf("check argv encoding unavailable")
	}
	if err := writeFileAtomic(run.record.ArgvPath, append(argvDocument, '\n'), 0o600); err != nil {
		return CheckRecord{}, err
	}
	startObservation, err := ObserveRun(repo, identity.Scope, runDir)
	if err != nil {
		return CheckRecord{}, err
	}
	run.startObs = startObservation
	run.meta = checkMeta{
		SchemaVersion:    SchemaVersion,
		Kind:             kindCheckMeta,
		Check:            run.record,
		TimeoutMillis:    req.Timeout.Milliseconds(),
		MaxOutputBytes:   run.maxOutput,
		StartObservation: &startObservation,
	}
	if err := writeCheckMeta(run.metaPath, run.meta); err != nil {
		return CheckRecord{}, err
	}
	applyCheckStart(checkpoint, run.record)
	checkpoint.LastObservation = &startObservation
	if err := WriteCheckpoint(runDir, checkpoint); err != nil {
		return CheckRecord{}, err
	}
	record, err := run.execute()
	if err != nil {
		return record, err
	}
	if err := lock.release(); err != nil {
		return record, err
	}
	return record, nil
}

// execute runs the prepared command and finalizes its evidence. The run lock is
// already held and the running checkpoint is persisted.
func (c *checkRun) execute() (CheckRecord, error) {
	stdoutFile, stdoutErr := createCheckLog(c.record.StdoutPath)
	stderrFile, stderrErr := createCheckLog(c.record.StderrPath)
	causes := make([]string, 0, 2)
	if stdoutErr != nil {
		c.record.StdoutPath = ""
		causes = append(causes, "cannot create the stdout log: "+stdoutErr.Error())
	}
	if stderrErr != nil {
		c.record.StderrPath = ""
		causes = append(causes, "cannot create the stderr log: "+stderrErr.Error())
	}
	if len(causes) > 0 {
		closeCheckLog(stdoutFile)
		closeCheckLog(stderrFile)
		return c.complete(CheckNotStarted, strings.Join(causes, "; "), false, false, nil, "", checkCapture{}, checkCapture{})
	}
	ctx, cancel := checkContext(c.timeout)
	defer cancel()
	stdout := newCheckStream(stdoutFile, c.maxOutput, cancel)
	stderr := newCheckStream(stderrFile, c.maxOutput, cancel)
	cmd := exec.CommandContext(ctx, c.argv[0], c.argv[1:]...)
	cmd.Dir = c.repo.Root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = checkWaitDelay
	configureCheckProcess(cmd)
	if startErr := cmd.Start(); startErr != nil {
		return c.complete(CheckNotStarted, "cannot start the command: "+startErr.Error(), false, false, nil, "",
			stdout.finish(), stderr.finish())
	}
	waitErr := cmd.Wait()
	stdoutCapture := stdout.finish()
	stderrCapture := stderr.finish()

	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
	outputLimit := stdoutCapture.Truncated || stderrCapture.Truncated
	state := cmd.ProcessState
	var exit *int
	if state != nil && state.Exited() {
		code := state.ExitCode()
		exit = &code
	}
	signal := checkProcessSignal(state)
	cause := ""
	switch {
	case stdoutCapture.WriteErr != nil:
		cause = "cannot persist stdout: " + stdoutCapture.WriteErr.Error()
	case stderrCapture.WriteErr != nil:
		cause = "cannot persist stderr: " + stderrCapture.WriteErr.Error()
	case waitErr != nil && !isCheckExitError(waitErr):
		cause = waitErr.Error()
	case state == nil:
		cause = "process exit state is unavailable"
	}
	result := CheckPassed
	switch {
	case outputLimit:
		result = CheckOutputLimit
	case timedOut:
		result = CheckTimedOut
	case cause != "":
		result = CheckFailed
	case signal != "":
		result = CheckInterrupted
	case exit == nil:
		result = CheckFailed
	case *exit != 0:
		result = CheckFailed
	}
	return c.complete(result, cause, timedOut, outputLimit, exit, signal, stdoutCapture, stderrCapture)
}

// complete records the observed outcome, reconciles the worktree observation and
// persists meta.json plus the checkpoint. A worktree change during the check
// keeps the evidence observable but not verified.
func (c *checkRun) complete(result CheckResult, cause string, timedOut, outputLimit bool, exit *int, signal string, stdout, stderr checkCapture) (CheckRecord, error) {
	end := time.Now().UTC()
	c.record.Result = result
	c.record.EndedAt = &end
	c.record.ExitCode = exit
	c.record.Signal = signal
	c.record.TimedOut = timedOut
	c.record.OutputLimit = outputLimit
	c.record.StdoutBytes = stdout.Bytes
	c.record.StdoutTruncated = stdout.Truncated
	c.record.StderrBytes = stderr.Bytes
	c.record.StderrTruncated = stderr.Truncated
	observation, observationErr := ObserveRun(c.repo, c.scope, c.runDir)
	if observationErr == nil {
		c.record.WorktreeChangedDuringCheck = observation.WorktreeDigest != c.startObs.WorktreeDigest
		c.meta.EndObservation = &observation
	}
	summary, summaryTruncated := checkSummary(c.record.StdoutPath, c.record.StderrPath, c.maxOutput)
	if cause != "" {
		if line := sanitizeCheckText([]byte(cause), checkSummaryLineBytes); line != "" {
			summary = append([]string{line}, summary...)
			if len(summary) > checkSummaryMatchLimit {
				summary = summary[:checkSummaryMatchLimit]
				summaryTruncated = true
			}
		}
	}
	c.record.Summary = summary
	c.record.SummaryTruncated = summaryTruncated
	c.meta.Check = c.record
	c.meta.Error = sanitizeCheckText([]byte(cause), checkErrorBytes)
	c.meta.Finished = true
	duration := end.Sub(c.record.StartedAt)
	if duration < 0 {
		duration = 0
	}
	c.meta.DurationMillis = duration.Milliseconds()
	state := checkState(result, c.record.WorktreeChangedDuringCheck)
	if observationErr != nil && state == StateVerifiedEvidence {
		// Without an end observation the worktree cannot be compared, so the
		// check cannot support verified evidence.
		state = StateObservedPartial
	}
	applyCheckEnd(c.checkpoint, c.record, state)
	if observationErr == nil {
		c.checkpoint.LastObservation = &observation
	}
	if err := writeCheckMeta(c.metaPath, c.meta); err != nil {
		return c.record, err
	}
	if err := WriteCheckpoint(c.runDir, c.checkpoint); err != nil {
		return c.record, err
	}
	return c.record, nil
}

// validateCheckArgv requires an explicit, bounded, NUL-free command vector. No
// shell parsing, glob expansion or variable interpolation happens anywhere.
func validateCheckArgv(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("%w: a command vector is required after --", ErrUsage)
	}
	if len(argv) > maxCheckArgvArgs {
		return fmt.Errorf("%w: the command vector exceeds %d arguments", ErrUsage, maxCheckArgvArgs)
	}
	total := 0
	for index, arg := range argv {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("%w: command arguments must not contain NUL bytes", ErrUsage)
		}
		if index == 0 && strings.TrimSpace(arg) == "" {
			return fmt.Errorf("%w: the command program is required", ErrUsage)
		}
		total += len(arg)
	}
	if total > maxCheckArgvBytes {
		return fmt.Errorf("%w: the command vector exceeds %d bytes", ErrUsage, maxCheckArgvBytes)
	}
	return nil
}

// normalizeCheckMaxOutputBytes applies the default and the supported range.
func normalizeCheckMaxOutputBytes(limit int64) int64 {
	switch {
	case limit <= 0:
		return DefaultMaxOutputBytes
	case limit < minCheckMaxOutputBytes:
		return minCheckMaxOutputBytes
	case limit > maxCheckMaxOutputBytes:
		return maxCheckMaxOutputBytes
	default:
		return limit
	}
}

// checkContext applies the optional wall-clock limit. Zero disables it.
func checkContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(context.Background())
	}
	return context.WithTimeout(context.Background(), timeout)
}

// createCheckDir allocates one private checks/<check-id> directory.
func createCheckDir(runDir string, now time.Time) (string, string, error) {
	for attempt := 0; attempt < maxArtifactAttempts; attempt++ {
		checkID, err := newArtifactID(checkIDPrefix, now)
		if err != nil {
			return "", "", err
		}
		dir := filepath.Join(runDir, "checks", checkID)
		if err := os.Mkdir(dir, 0o700); err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return "", "", fmt.Errorf("create check directory: %w", err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			os.RemoveAll(dir)
			return "", "", fmt.Errorf("restrict check directory permissions: %w", err)
		}
		return checkID, dir, nil
	}
	return "", "", fmt.Errorf("%w: could not allocate a unique check ID", ErrConflict)
}

// checkArgvDigest is the stable digest of the exact argument vector. The vector
// itself stays in the private argv.json.
func checkArgvDigest(argv []string) string {
	h := sha256.New()
	hashField(h, "argv-v1")
	hashField(h, strconv.Itoa(len(argv)))
	for _, arg := range argv {
		hashField(h, arg)
	}
	return digestPrefix(h)
}

// createCheckLog creates one private, empty stream log.
func createCheckLog(name string) (*os.File, error) {
	file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func closeCheckLog(file *os.File) error {
	if file == nil {
		return nil
	}
	return file.Close()
}

// checkStream streams one child stream into a capped private log. When the cap
// is reached the process is cancelled; the overflow is counted as truncated and
// swallowed so a chatty child cannot block on a full pipe.
type checkStream struct {
	mu        sync.Mutex
	file      *os.File
	limit     int64
	written   int64
	truncated bool
	writeErr  error
	cancel    context.CancelFunc
}

// checkCapture is the final observation of one stream log.
type checkCapture struct {
	Bytes     int64
	Truncated bool
	WriteErr  error
}

func newCheckStream(file *os.File, limit int64, cancel context.CancelFunc) *checkStream {
	return &checkStream{file: file, limit: limit, cancel: cancel}
}

func (s *checkStream) Write(p []byte) (int, error) {
	s.mu.Lock()
	if s.writeErr != nil {
		err := s.writeErr
		s.mu.Unlock()
		return 0, err
	}
	remaining := s.limit - s.written
	if remaining <= 0 {
		s.truncated = true
		s.mu.Unlock()
		s.requestCancel()
		return len(p), nil
	}
	chunk := p
	overflow := false
	if int64(len(chunk)) > remaining {
		chunk = chunk[:remaining]
		overflow = true
	}
	n, err := s.file.Write(chunk)
	s.written += int64(n)
	if err != nil {
		s.writeErr = err
		s.mu.Unlock()
		return n, err
	}
	if overflow {
		s.truncated = true
		s.mu.Unlock()
		s.requestCancel()
		return len(p), nil
	}
	s.mu.Unlock()
	return len(p), nil
}

func (s *checkStream) requestCancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

// finish snapshots the stream and closes its log file.
func (s *checkStream) finish() checkCapture {
	s.mu.Lock()
	capture := checkCapture{Bytes: s.written, Truncated: s.truncated, WriteErr: s.writeErr}
	s.mu.Unlock()
	if s.file != nil {
		if err := s.file.Close(); err != nil && capture.WriteErr == nil {
			capture.WriteErr = err
		}
	}
	return capture
}

// checkSummary extracts a bounded excerpt: case-insensitive failure-marker lines
// first, deduplicated and capped at 40, or the last 20 lines when no marker is
// present. The capped logs are authoritative; the excerpt never claims
// completeness.
func checkSummary(stdoutPath, stderrPath string, limit int64) ([]string, bool) {
	streams := [][]byte{
		readCheckLog(stdoutPath, limit),
		readCheckLog(stderrPath, limit),
	}
	seen := make(map[string]bool)
	matches := make([]string, 0, checkSummaryMatchLimit)
	truncated := false
	for _, data := range streams {
		forEachCheckLine(data, func(line []byte) {
			if !hasCheckFailureMarker(line) {
				return
			}
			sanitized := sanitizeCheckText(line, checkSummaryLineBytes)
			if sanitized == "" || seen[sanitized] {
				return
			}
			seen[sanitized] = true
			if len(matches) >= checkSummaryMatchLimit {
				truncated = true
				return
			}
			matches = append(matches, sanitized)
		})
	}
	if len(matches) > 0 {
		return matches, truncated
	}
	tail := make([]string, 0, checkSummaryTailLines)
	for _, data := range streams {
		forEachCheckLine(data, func(line []byte) {
			sanitized := sanitizeCheckText(line, checkSummaryLineBytes)
			if sanitized == "" {
				return
			}
			if len(tail) == checkSummaryTailLines {
				copy(tail, tail[1:])
				tail[checkSummaryTailLines-1] = sanitized
				return
			}
			tail = append(tail, sanitized)
		})
	}
	return tail, false
}

// readCheckLog reads one capped log. A missing or oversized log yields no
// excerpt; the check record still carries the exact log path.
func readCheckLog(name string, limit int64) []byte {
	if name == "" {
		return nil
	}
	data, err := readBoundedRegular(name, limit)
	if err != nil {
		return nil
	}
	return data
}

// forEachCheckLine visits line content without splitting the whole log at once.
func forEachCheckLine(data []byte, visit func([]byte)) {
	for len(data) > 0 {
		index := bytes.IndexByte(data, '\n')
		if index < 0 {
			visit(data)
			return
		}
		visit(data[:index])
		data = data[index+1:]
	}
}

// hasCheckFailureMarker reports a case-insensitive ASCII marker match.
func hasCheckFailureMarker(line []byte) bool {
	for _, marker := range checkFailureMarkers {
		if containsCheckMarker(line, marker) {
			return true
		}
	}
	return false
}

func containsCheckMarker(line []byte, marker string) bool {
	if marker == "" || len(line) < len(marker) {
		return false
	}
	for start := 0; start+len(marker) <= len(line); start++ {
		matched := true
		for offset := 0; offset < len(marker); offset++ {
			if lowerCheckASCII(line[start+offset]) != marker[offset] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func lowerCheckASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}

// sanitizeCheckText makes one excerpt line safe for text/JSON output: valid
// UTF-8, control characters replaced, and a hard byte bound. This keeps terminal
// escapes and binary noise out of the summary.
func sanitizeCheckText(raw []byte, limit int) string {
	text := strings.ToValidUTF8(strings.TrimRight(string(raw), "\r"), "?")
	var builder strings.Builder
	cut := false
	for _, r := range text {
		if builder.Len() >= limit {
			cut = true
			break
		}
		if r == 0 || r < 0x20 || r == 0x7f {
			builder.WriteByte(' ')
			continue
		}
		builder.WriteRune(r)
	}
	result := strings.TrimSpace(builder.String())
	if result == "" {
		return ""
	}
	if cut {
		result += " ..."
	}
	return result
}

// isCheckExitError reports an ordinary process exit (including a signal) as
// opposed to a start, pipe or wait failure.
func isCheckExitError(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

// applyCheckStart appends the in-progress record and reports the run and task as
// running. The record stays observable until the final observation replaces it.
func applyCheckStart(checkpoint *Checkpoint, record CheckRecord) {
	checkpoint.Checks = append(checkpoint.Checks, record)
	if record.TaskID != "" {
		if checkpoint.Tasks == nil {
			checkpoint.Tasks = map[string]TaskState{}
		}
		task := checkpoint.Tasks[record.TaskID]
		task.TaskID = record.TaskID
		task.State = StateRunning
		task.LastCheckID = record.CheckID
		task.UpdatedAt = record.StartedAt
		checkpoint.Tasks[record.TaskID] = task
	}
	RecomputeRunState(checkpoint)
	checkpoint.State = HighestState(checkpoint.State, StateRunning)
}

// applyCheckEnd replaces the in-progress record with the observed result and
// updates the task and run state. Blockers still take precedence.
func applyCheckEnd(checkpoint *Checkpoint, record CheckRecord, state CheckpointState) {
	replaced := false
	for index := range checkpoint.Checks {
		if checkpoint.Checks[index].CheckID == record.CheckID {
			checkpoint.Checks[index] = record
			replaced = true
			break
		}
	}
	if !replaced {
		checkpoint.Checks = append(checkpoint.Checks, record)
	}
	if record.TaskID != "" {
		if checkpoint.Tasks == nil {
			checkpoint.Tasks = map[string]TaskState{}
		}
		task := checkpoint.Tasks[record.TaskID]
		task.TaskID = record.TaskID
		task.State = state
		task.LastCheckID = record.CheckID
		task.UpdatedAt = record.StartedAt
		checkpoint.Tasks[record.TaskID] = task
	}
	RecomputeRunState(checkpoint)
	checkpoint.State = HighestState(checkpoint.State, state)
}

// checkState maps an observed result to the checkpoint state. Only a passing
// check on an unchanged worktree can support verified evidence.
func checkState(result CheckResult, worktreeChanged bool) CheckpointState {
	switch result {
	case CheckPassed:
		if worktreeChanged {
			return StateObservedPartial
		}
		return StateVerifiedEvidence
	case CheckFailed, CheckNotStarted:
		return StateObservedFailed
	case CheckTimedOut, CheckInterrupted, CheckOutputLimit:
		return StateObservedPartial
	default:
		return StateObservedPartial
	}
}

// writeCheckMeta atomically persists the private check metadata.
func writeCheckMeta(name string, meta checkMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("check metadata encoding unavailable")
	}
	return writeFileAtomic(name, append(data, '\n'), 0o600)
}
