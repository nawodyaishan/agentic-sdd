package execution

// Task-brief provenance and freshness. A persisted brief is private run scratch,
// not evidence or approval. Registration records the artifact that the service
// actually wrote; inspection reports drift without repairing it.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Freshness reasons specific to persisted task briefs. They block trust in an
// earlier observation exactly like the other non-progress reasons.
const (
	ReasonBriefDigestMismatch   = "brief_digest_mismatch"
	ReasonUnregisteredBrief     = "unregistered_brief"
	ReasonBriefInspectionFailed = "brief_inspection_failed"
)

const (
	briefArtifactDirName = "briefs"
	// maxBriefArtifacts bounds directory inspection so a damaged or hostile run
	// cannot turn status into an unbounded walk.
	maxBriefArtifacts = 4096
	briefReadBatch    = 256
)

// registerTaskBrief records one successfully persisted brief in the checkpoint.
// The caller holds the run lock and persists the checkpoint. The helper verifies
// that the artifact is the task's designated regular file with the reported
// digest; it never changes task/run state or writes to disk.
func registerTaskBrief(checkpoint *Checkpoint, runDir string, brief Brief) error {
	if checkpoint == nil {
		return fmt.Errorf("%w: checkpoint is required", ErrUsage)
	}
	if err := ValidateRunID(checkpoint.RunID); err != nil {
		return err
	}
	if strings.TrimSpace(brief.RunID) == "" || brief.RunID != checkpoint.RunID {
		return fmt.Errorf("%w: brief run ID does not match the checkpoint", ErrConflict)
	}
	taskID := strings.TrimSpace(brief.TaskID)
	if err := validateToken("task ID", taskID); err != nil {
		return err
	}
	if strings.TrimSpace(brief.BriefPath) == "" || strings.TrimSpace(brief.BriefDigest) == "" {
		return fmt.Errorf("%w: brief path and digest are required", ErrUsage)
	}
	root, err := filepath.Abs(runDir)
	if err != nil {
		return fmt.Errorf("%w: run directory is invalid", ErrUsage)
	}
	root = filepath.Clean(root)
	expected := filepath.Join(root, briefArtifactDirName, taskID+".md")
	path := filepath.Clean(brief.BriefPath)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	if filepath.Clean(path) != expected {
		return fmt.Errorf("%w: brief path is not the task's designated artifact", ErrUnsafePath)
	}
	data, err := readBoundedRegular(expected, MaxSpecFileBytes)
	if err != nil {
		return fmt.Errorf("%w: read brief artifact: %v", ErrConflict, err)
	}
	digest := DigestBytes(data)
	if digest != brief.BriefDigest {
		return fmt.Errorf("%w: brief digest does not match the artifact", ErrConflict)
	}
	if checkpoint.Tasks == nil {
		checkpoint.Tasks = map[string]TaskState{}
	}
	task := checkpoint.Tasks[taskID]
	if task.TaskID != "" && task.TaskID != taskID {
		return fmt.Errorf("%w: checkpoint task identity is inconsistent", ErrConflict)
	}
	if task.TaskID == "" {
		task.TaskID = taskID
	}
	if task.State == "" {
		task.State = StatePrepared
	}
	task.BriefPath = expected
	task.BriefDigest = digest
	if brief.GeneratedAt.IsZero() {
		task.UpdatedAt = time.Now().UTC()
	} else {
		task.UpdatedAt = brief.GeneratedAt
	}
	checkpoint.Tasks[taskID] = task
	return nil
}

// inspectBriefArtifacts returns deterministic freshness reasons for registered
// brief artifacts and for regular files under briefs/ that have no registered
// provenance. The caller treats any error as a reconciliation failure, never as
// a clean observation.
func inspectBriefArtifacts(runDir string, checkpoint *Checkpoint) ([]string, error) {
	root, err := filepath.Abs(runDir)
	if err != nil {
		return nil, fmt.Errorf("%w: run directory is invalid", ErrUsage)
	}
	root = filepath.Clean(root)
	briefsDir := filepath.Join(root, briefArtifactDirName)
	reasons := map[string]bool{}
	add := func(reason string) { reasons[reason] = true }

	registered := map[string]bool{}
	taskIDs := make([]string, 0)
	if checkpoint != nil {
		for taskID := range checkpoint.Tasks {
			taskIDs = append(taskIDs, taskID)
		}
	}
	sort.Strings(taskIDs)
	for _, taskID := range taskIDs {
		task := checkpoint.Tasks[taskID]
		if task.BriefPath == "" {
			if task.BriefDigest != "" {
				add(ReasonBriefInspectionFailed)
			}
			continue
		}
		path := filepath.Clean(task.BriefPath)
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		path = filepath.Clean(path)
		if path == briefsDir || !withinPath(briefsDir, path) {
			return nil, fmt.Errorf("%w: registered brief path is outside the run's brief directory", ErrUnsafePath)
		}
		registered[path] = true
		info, statErr := os.Lstat(path)
		switch {
		case errors.Is(statErr, fs.ErrNotExist):
			add(ReasonMissingArtifact)
			continue
		case statErr != nil:
			add(ReasonBriefInspectionFailed)
			continue
		case info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular():
			add(ReasonBriefInspectionFailed)
			continue
		}
		if task.BriefDigest == "" {
			add(ReasonBriefInspectionFailed)
			continue
		}
		data, readErr := readBoundedRegular(path, MaxSpecFileBytes)
		if readErr != nil {
			add(ReasonBriefInspectionFailed)
			continue
		}
		if DigestBytes(data) != task.BriefDigest {
			add(ReasonBriefDigestMismatch)
		}
	}

	info, err := os.Lstat(briefsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return sortedBriefReasons(reasons), nil
	}
	if err != nil {
		return nil, fmt.Errorf("brief directory is unreadable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("%w: brief directory is a symlink or not a directory", ErrUnsafePath)
	}
	dir, err := os.Open(briefsDir)
	if err != nil {
		return nil, fmt.Errorf("brief directory is unreadable: %w", err)
	}
	defer dir.Close()

	total := 0
	for {
		entries, readErr := dir.ReadDir(briefReadBatch)
		for _, entry := range entries {
			total++
			if total > maxBriefArtifacts {
				return nil, fmt.Errorf("%w: brief directory exceeds %d entries", ErrUnsafePath, maxBriefArtifacts)
			}
			full := filepath.Clean(filepath.Join(briefsDir, entry.Name()))
			if registered[full] {
				continue
			}
			entryInfo, statErr := os.Lstat(full)
			if statErr != nil || entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() {
				add(ReasonBriefInspectionFailed)
				continue
			}
			add(ReasonUnregisteredBrief)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("brief directory is unreadable: %w", readErr)
		}
		if len(entries) == 0 {
			break
		}
	}
	return sortedBriefReasons(reasons), nil
}

func sortedBriefReasons(reasons map[string]bool) []string {
	if len(reasons) == 0 {
		return nil
	}
	result := make([]string, 0, len(reasons))
	for reason := range reasons {
		result = append(result, reason)
	}
	sort.Strings(result)
	return result
}
