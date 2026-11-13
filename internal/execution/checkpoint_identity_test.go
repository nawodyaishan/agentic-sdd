package execution

// Checkpoint ownership regressions for Feature 017. A checkpoint belongs to a
// run directory only when its run ID equals the validated owning run identity;
// a syntactically valid foreign ID is a conflict, never normal status. Fixtures
// are disposable temporary repositories.

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// foreignCheckpointRunID is well-formed but cannot be the owning run ID because
// InitRun always generates a distinct current-time identifier.
const foreignCheckpointRunID = "run-20200101T000000Z-abcdef"

// rewriteCheckpointRunID replaces run_id in checkpoint.json directly, bypassing
// WriteCheckpoint, which already refuses a mismatched identity.
func rewriteCheckpointRunID(t *testing.T, runDir, runID string) []byte {
	t.Helper()
	path := filepath.Join(runDir, checkpointName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	checkpoint.RunID = runID
	encoded, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		t.Fatalf("encode checkpoint: %v", err)
	}
	encoded = append(encoded, '\n')
	if err := writeFileAtomic(path, encoded, 0o600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	return encoded
}

func readCheckpointBytes(t *testing.T, runDir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir, checkpointName))
	if err != nil {
		t.Fatalf("read checkpoint bytes: %v", err)
	}
	return data
}

func TestLoadCheckpointRequiresOwningRunID(t *testing.T) {
	t.Run("foreign but valid run ID is a conflict", func(t *testing.T) {
		if err := ValidateRunID(foreignCheckpointRunID); err != nil {
			t.Fatalf("fixture run ID is not valid: %v", err)
		}
		_, _, identity := workspaceTestRun(t)
		foreign := rewriteCheckpointRunID(t, identity.RunDir, foreignCheckpointRunID)
		if _, err := LoadCheckpoint(identity.RunDir); !errors.Is(err, ErrConflict) {
			t.Fatalf("LoadCheckpoint(foreign) error = %v, want ErrConflict", err)
		}
		if after := readCheckpointBytes(t, identity.RunDir); !bytes.Equal(after, foreign) {
			t.Fatal("LoadCheckpoint changed the refused checkpoint")
		}
	})

	t.Run("status refuses a foreign checkpoint without a normal status", func(t *testing.T) {
		_, runRoot, identity := workspaceTestRun(t)
		foreign := rewriteCheckpointRunID(t, identity.RunDir, foreignCheckpointRunID)
		status, err := StatusRun(runRoot, identity.RunID)
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("StatusRun(foreign) error = %v, want ErrConflict", err)
		}
		if status.State != "" || status.NextPermittedInvestigation != "" || len(status.StaleReasons) != 0 {
			t.Fatalf("foreign checkpoint produced a usable status: %+v", status)
		}
		if after := readCheckpointBytes(t, identity.RunDir); !bytes.Equal(after, foreign) {
			t.Fatal("status changed the foreign checkpoint")
		}
	})

	t.Run("checkpoint-consuming mutation refuses before writing", func(t *testing.T) {
		_, _, identity := workspaceTestRun(t)
		foreign := rewriteCheckpointRunID(t, identity.RunDir, foreignCheckpointRunID)
		_, err := RunCheck(CheckRequest{
			RunDir:  identity.RunDir,
			RunID:   identity.RunID,
			TaskID:  "T01",
			Argv:    []string{os.Args[0], "-test.run=^$"},
			Timeout: 30 * time.Second,
		})
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("RunCheck(foreign) error = %v, want ErrConflict", err)
		}
		if after := readCheckpointBytes(t, identity.RunDir); !bytes.Equal(after, foreign) {
			t.Fatal("refused check changed the foreign checkpoint")
		}
		entries, readErr := os.ReadDir(filepath.Join(identity.RunDir, "checks"))
		if readErr != nil {
			t.Fatalf("read checks directory: %v", readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("refused check left %d check artifact(s)", len(entries))
		}
		if _, statErr := os.Stat(filepath.Join(identity.RunDir, lockFileName)); !errors.Is(statErr, fs.ErrNotExist) {
			t.Fatalf("refused check left the writer lock: %v", statErr)
		}
	})

	t.Run("record and review packaging refuse a foreign checkpoint", func(t *testing.T) {
		_, _, identity := workspaceTestRun(t)
		foreign := rewriteCheckpointRunID(t, identity.RunDir, foreignCheckpointRunID)
		claim := []byte(`{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"fixture","changedPaths":[],"checks":[],"blockers":[],"unknowns":[]}`)
		if _, err := IngestRecord(identity.RunDir, "T01", claim); !errors.Is(err, ErrConflict) {
			t.Fatalf("IngestRecord(foreign) error = %v, want ErrConflict", err)
		}
		if entries, err := os.ReadDir(filepath.Join(identity.RunDir, "records")); err == nil && len(entries) != 0 {
			t.Fatalf("refused record left %d artifact(s)", len(entries))
		}
		if _, err := BuildReviewPackage(ReviewPackageRequest{
			RunDir: identity.RunDir,
			RunID:  identity.RunID,
			Base:   "HEAD",
		}); !errors.Is(err, ErrConflict) {
			t.Fatalf("BuildReviewPackage(foreign) error = %v, want ErrConflict", err)
		}
		if after := readCheckpointBytes(t, identity.RunDir); !bytes.Equal(after, foreign) {
			t.Fatal("refused record or review packaging changed the foreign checkpoint")
		}
		if _, err := os.Stat(filepath.Join(identity.RunDir, lockFileName)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("refused record or review packaging left the writer lock: %v", err)
		}
	})

	t.Run("write still refuses a foreign checkpoint identity", func(t *testing.T) {
		_, _, identity := workspaceTestRun(t)
		before := readCheckpointBytes(t, identity.RunDir)
		checkpoint, err := LoadCheckpoint(identity.RunDir)
		if err != nil {
			t.Fatalf("load checkpoint: %v", err)
		}
		checkpoint.RunID = foreignCheckpointRunID
		if err := WriteCheckpoint(identity.RunDir, checkpoint); !errors.Is(err, ErrConflict) {
			t.Fatalf("WriteCheckpoint(foreign) error = %v, want ErrConflict", err)
		}
		if after := readCheckpointBytes(t, identity.RunDir); !bytes.Equal(after, before) {
			t.Fatal("refused write changed the matching checkpoint")
		}
	})

	t.Run("matching schema-v1 checkpoint still loads and reports prepared", func(t *testing.T) {
		_, runRoot, identity := workspaceTestRun(t)
		checkpoint, err := LoadCheckpoint(identity.RunDir)
		if err != nil {
			t.Fatalf("load checkpoint: %v", err)
		}
		if checkpoint.SchemaVersion != SchemaVersion || checkpoint.RunID != identity.RunID {
			t.Fatalf("checkpoint envelope = schema %d run %q", checkpoint.SchemaVersion, checkpoint.RunID)
		}
		status := statusForTest(t, runRoot, identity.RunID)
		if status.State != StatePrepared || len(status.StaleReasons) != 0 {
			t.Fatalf("matching run = %q with reasons %v, want prepared with none", status.State, status.StaleReasons)
		}
	})

	t.Run("missing checkpoint keeps the synthesized prepared status", func(t *testing.T) {
		_, runRoot, identity := workspaceTestRun(t)
		if err := os.Remove(filepath.Join(identity.RunDir, checkpointName)); err != nil {
			t.Fatalf("remove checkpoint: %v", err)
		}
		if _, err := LoadCheckpoint(identity.RunDir); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("LoadCheckpoint(missing) error = %v, want fs.ErrNotExist", err)
		}
		status := statusForTest(t, runRoot, identity.RunID)
		if status.State != StatePrepared || len(status.StaleReasons) != 0 {
			t.Fatalf("missing checkpoint = %q with reasons %v, want prepared with none", status.State, status.StaleReasons)
		}
		if status.NextPermittedInvestigation != InvestigateNone {
			t.Fatalf("next investigation = %q, want none", status.NextPermittedInvestigation)
		}
	})
}
