package execution

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskBrief_HeadingT1(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "standard_tasks.md")
	specPath := filepath.Join("testdata", "spec", "standard_spec.md")

	req := TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T1",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	}

	brief, err := TaskBrief(req)
	if err != nil {
		t.Fatalf("TaskBrief failed: %v", err)
	}

	if brief.TaskID != "T1" {
		t.Errorf("expected TaskID T1, got %q", brief.TaskID)
	}
	if brief.LineStart <= 0 || brief.LineEnd < brief.LineStart {
		t.Errorf("invalid line range: %d-%d", brief.LineStart, brief.LineEnd)
	}
	// Verify references extracted: AC-01, AC-02, AC-03, FR-01, FR-02
	expectedRefs := []string{"AC-01", "AC-02", "AC-03", "FR-01", "FR-02"}
	if len(brief.References) != len(expectedRefs) {
		t.Errorf("expected %d references, got %v", len(expectedRefs), brief.References)
	} else {
		for i, r := range expectedRefs {
			if brief.References[i] != r {
				t.Errorf("expected reference %q at %d, got %q", r, i, brief.References[i])
			}
		}
	}
}

func TestTaskBrief_HeadingT01(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "zero_padded_tasks.md")
	specPath := filepath.Join("testdata", "spec", "standard_spec.md")

	req := TaskBriefRequest{
		BatchID:   "B01",
		TaskID:    "T01",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	}

	brief, err := TaskBrief(req)
	if err != nil {
		t.Fatalf("TaskBrief failed: %v", err)
	}

	if brief.TaskID != "T01" {
		t.Errorf("expected TaskID T01, got %q", brief.TaskID)
	}
}

func TestTaskBrief_T1_vs_T10(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "standard_tasks.md")
	specPath := filepath.Join("testdata", "spec", "standard_spec.md")

	// 1. Request T1: must not match T10
	brief1, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T1",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	})
	if err != nil {
		t.Fatalf("TaskBrief for T1 failed: %v", err)
	}
	if brief1.TaskID != "T1" {
		t.Errorf("expected TaskID T1, got %q", brief1.TaskID)
	}

	// 2. Request T10: must not match T1
	brief10, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T10",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	})
	if err != nil {
		t.Fatalf("TaskBrief for T10 failed: %v", err)
	}
	if brief10.TaskID != "T10" {
		t.Errorf("expected TaskID T10, got %q", brief10.TaskID)
	}

	// Ensure line ranges do not overlap
	if brief1.LineEnd >= brief10.LineStart {
		t.Errorf("T1 and T10 line ranges overlap: T1=%d-%d, T10=%d-%d",
			brief1.LineStart, brief1.LineEnd, brief10.LineStart, brief10.LineEnd)
	}
}

func TestTaskBrief_CheckboxForm(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "checkbox_only_tasks.md")
	specPath := filepath.Join("testdata", "spec", "standard_spec.md")

	brief, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T01",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	})
	if err != nil {
		t.Fatalf("TaskBrief failed for checkbox task: %v", err)
	}
	if brief.TaskID != "T01" {
		t.Errorf("expected TaskID T01, got %q", brief.TaskID)
	}

	// Also extract T02 which was [x]
	brief2, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T02",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	})
	if err != nil {
		t.Fatalf("TaskBrief failed for checked task T02: %v", err)
	}
	if brief2.TaskID != "T02" {
		t.Errorf("expected TaskID T02, got %q", brief2.TaskID)
	}
}

func TestTaskBrief_NestedHeadingsAndNonTaskBoundary(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "standard_tasks.md")

	// T1 contains #### Step 1 and #### Step 2
	extracted, err := ExtractTask(mustReadFile(t, tasksPath), "B1", "T1")
	if err != nil {
		t.Fatalf("ExtractTask failed: %v", err)
	}
	if !strings.Contains(extracted.RawContent, "#### Step 1") {
		t.Errorf("nested heading #### Step 1 was not included in task section")
	}
	if !strings.Contains(extracted.RawContent, "#### Step 2") {
		t.Errorf("nested heading #### Step 2 was not included in task section")
	}

	// T10 ends before ### Non-task Boundary Heading
	extracted10, err := ExtractTask(mustReadFile(t, tasksPath), "B1", "T10")
	if err != nil {
		t.Fatalf("ExtractTask for T10 failed: %v", err)
	}
	if strings.Contains(extracted10.RawContent, "Non-task Boundary Heading") {
		t.Errorf("non-task boundary heading leaked into T10 task content")
	}
}

func TestTaskBrief_CodeFences(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "fenced_tasks.md")

	extracted, err := ExtractTask(mustReadFile(t, tasksPath), "B1", "T01")
	if err != nil {
		t.Fatalf("ExtractTask failed: %v", err)
	}

	// Headings inside fences must be present as content
	if !strings.Contains(extracted.RawContent, "### T02 — This is content inside fence") {
		t.Errorf("fenced content missing from T01")
	}
	if !strings.Contains(extracted.RawContent, "## B2 — Batch inside fence") {
		t.Errorf("fenced batch heading missing from T01")
	}
	if !strings.Contains(extracted.RawContent, "Post-fence content for T01.") {
		t.Errorf("post-fence content missing from T01")
	}

	// Real T02 must be extractable as a separate task
	extracted2, err := ExtractTask(mustReadFile(t, tasksPath), "B1", "T02")
	if err != nil {
		t.Fatalf("ExtractTask for real T02 failed: %v", err)
	}
	if !strings.Contains(extracted2.RawContent, "Real task following fence.") {
		t.Errorf("real T02 content mismatch")
	}
}

func TestTaskBrief_MalformedFence(t *testing.T) {
	unclosedFenceTasks := `## B1
### T01 — Broken fence task
` + "```markdown" + `
content without closing fence
`
	_, err := ExtractTask([]byte(unclosedFenceTasks), "B1", "T01")
	if err == nil {
		t.Fatalf("expected error on unclosed fence, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
}

func TestTaskBrief_CRLF(t *testing.T) {
	crlfContent := "# CRLF doc\r\n\r\n## B1 — Batch\r\n\r\n### T01 — CRLF task\r\n\r\n- [ ] **T01** CRLF line\r\n  - **Acceptance:** AC-01\r\n\r\n### T02 — Second\r\n- [ ] **T02** Next\r\n"
	specContent := "# Spec\r\n\r\n| ID | Condition |\r\n|---|---|\r\n| **AC-01** | First |\r\n"

	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	specFile := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(tasksFile, []byte(crlfContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specFile, []byte(specContent), 0600); err != nil {
		t.Fatal(err)
	}

	brief, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T01",
		TasksPath: tasksFile,
		SpecPath:  specFile,
	})
	if err != nil {
		t.Fatalf("TaskBrief failed on CRLF content: %v", err)
	}
	if brief.TaskID != "T01" {
		t.Errorf("expected T01, got %q", brief.TaskID)
	}
	if brief.LineStart != 5 {
		t.Errorf("expected LineStart 5, got %d", brief.LineStart)
	}
}

func TestTaskBrief_UnresolvedReference(t *testing.T) {
	tasksContent := `## B1
### T01 — Task with missing ref
- [ ] **T01** Task
  - **Acceptance:** AC-99
`
	specContent := `## Acceptance Criteria
| ID | Condition |
|---|---|
| **AC-01** | Basic |
`
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	specFile := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(tasksFile, []byte(tasksContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specFile, []byte(specContent), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T01",
		TasksPath: tasksFile,
		SpecPath:  specFile,
	})
	if err == nil {
		t.Fatalf("expected error on unresolved reference, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
}

func TestTaskBrief_DuplicateReferenceInSpec(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "zero_padded_tasks.md")
	specPath := filepath.Join("testdata", "spec", "duplicate_ref_spec.md")

	_, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B01",
		TaskID:    "T01",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	})
	if err == nil {
		t.Fatalf("expected error on duplicate reference definition in spec, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
}

func TestTaskBrief_MissingTask(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "standard_tasks.md")
	specPath := filepath.Join("testdata", "spec", "standard_spec.md")

	_, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T99",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	})
	if err == nil {
		t.Fatalf("expected error for non-existent task, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskBrief_WrongBatchOnlyTask(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "standard_tasks.md")
	specPath := filepath.Join("testdata", "spec", "standard_spec.md")

	// T20 belongs to B2, but we ask for it in B1
	_, err := TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T20",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	})
	if err == nil {
		t.Fatalf("expected error for wrong-batch-only task, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
	if !strings.Contains(err.Error(), "appears in batch \"B2\" only") {
		t.Errorf("expected error to name batch B2, got %v", err)
	}
}

func TestTaskBrief_MissingAndDuplicateBatch(t *testing.T) {
	tasksContent := `## B1
### T01 — Task 1

## B1
### T02 — Duplicate batch task
`
	_, err := ExtractTask([]byte(tasksContent), "B1", "T01")
	if err == nil {
		t.Fatalf("expected error on duplicate batch, got nil")
	}
	if !errors.Is(err, ErrUsage) && !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrUsage or ErrConflict, got %v", err)
	}

	_, err = ExtractTask([]byte(tasksContent), "B99", "T01")
	if err == nil {
		t.Fatalf("expected error on missing batch, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskBrief_DuplicateTaskInBatch(t *testing.T) {
	tasksContent := `## B1
### T01 — Task 1
- [ ] **T01** First
### T01 — Duplicate Task 1
- [ ] **T01** Duplicate
`
	_, err := ExtractTask([]byte(tasksContent), "B1", "T01")
	if err == nil {
		t.Fatalf("expected error on duplicate task in batch, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
}

func TestTaskBrief_UnparseableTaskHeading(t *testing.T) {
	cases := []string{
		"## B1\n### T\n- [ ] Content",
		"## B1\n### T01?\n- [ ] Content",
		"## B1\n### Task\n- [ ] Content",
		"## B1\n### Task:\n- [ ] Content",
	}

	for _, c := range cases {
		_, err := ExtractTask([]byte(c), "B1", "T01")
		if err == nil {
			t.Errorf("expected error for unparseable heading %q, got nil", c)
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("expected ErrUsage for %q, got %v", c, err)
		}
	}
}

func TestTaskBrief_ScopeDigest_ProgressOnly(t *testing.T) {
	initialTasks := `## B1 — Batch 1

**Batch ID:** B1
**Outcome:** Verification of core functionality
**State:** authorized — wave 1 in progress
**Next action:** DeepSeek max implements T02

### T01 — First task

- [ ] **T01** Implementation.
  - **Objective:** Verify basic task extraction.
**Status:** in progress
  - **Acceptance:** AC-01

### T02 — Second task

- [ ] **T02** Next task.
**Completion:** 0%
`

	digest1, err := NormalizedBatchScopeDigest([]byte(initialTasks), "B1")
	if err != nil {
		t.Fatalf("NormalizedBatchScopeDigest failed: %v", err)
	}

	// Simulate progress: checkbox checked, state updated, next action changed, status updated, completion 100%
	progressTasks := `## B1 — Batch 1

**Batch ID:** B1
**Outcome:** Verification of core functionality
**State:** awaiting human review
**Next action:** Review B1 completion

### T01 — First task

- [x] **T01** Implementation.
  - **Objective:** Verify basic task extraction.
**Status:** completed
  - **Acceptance:** AC-01

### T02 — Second task

- [X] **T02** Next task.
**Completion:** 100%
`

	digest2, err := NormalizedBatchScopeDigest([]byte(progressTasks), "B1")
	if err != nil {
		t.Fatalf("NormalizedBatchScopeDigest failed on progress: %v", err)
	}

	if digest1 != digest2 {
		t.Fatalf("progress-only change altered scope digest:\nbefore: %s\nafter:  %s", digest1, digest2)
	}

	// Now make a real scope change: edit acceptance requirement
	editedTasks := strings.Replace(initialTasks, "AC-01", "AC-02", 1)
	digest3, err := NormalizedBatchScopeDigest([]byte(editedTasks), "B1")
	if err != nil {
		t.Fatalf("NormalizedBatchScopeDigest failed on edit: %v", err)
	}

	if digest1 == digest3 {
		t.Fatalf("substantive task edit did NOT change scope digest: %s", digest1)
	}
}

func TestTaskBrief_IdempotenceAndForce(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	req := TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
		Force:  false,
	}

	// 1. Initial generation: creates briefs/T01.md
	brief1, err := TaskBrief(req)
	if err != nil {
		t.Fatalf("initial TaskBrief failed: %v", err)
	}

	briefFile := filepath.Join(runDir, "briefs", "T01.md")
	info, err := os.Stat(briefFile)
	if err != nil {
		t.Fatalf("expected brief file to exist: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 file permissions, got %v", info.Mode().Perm())
	}

	// 2. Second generation (same content): must be no-op success
	brief2, err := TaskBrief(req)
	if err != nil {
		t.Fatalf("second TaskBrief failed: %v", err)
	}
	if brief1.BriefDigest != brief2.BriefDigest {
		t.Errorf("brief digest changed between identical runs: %s vs %s", brief1.BriefDigest, brief2.BriefDigest)
	}

	// 3. Overwrite with differing content without Force: must fail with ErrConflict
	// Tamper with briefFile
	if err := os.WriteFile(briefFile, []byte("tampered content"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err = TaskBrief(req)
	if err == nil {
		t.Fatalf("expected conflict error when brief file differs without force, got nil")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// 4. Overwrite with differing content with Force = true: must succeed
	req.Force = true
	brief4, err := TaskBrief(req)
	if err != nil {
		t.Fatalf("TaskBrief with Force failed: %v", err)
	}
	if brief4.BriefDigest != brief1.BriefDigest {
		t.Errorf("expected restored brief digest %s, got %s", brief1.BriefDigest, brief4.BriefDigest)
	}
}

func TestTaskBrief_FileSizeLimitAndUTF8(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Oversized tasks.md
	largeTasks := filepath.Join(tmpDir, "large_tasks.md")
	f, err := os.Create(largeTasks)
	if err != nil {
		t.Fatal(err)
	}
	// Write slightly over 4 MiB
	chunk := make([]byte, 1024*1024)
	for i := 0; i < 5; i++ {
		if _, err := f.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()

	specPath := filepath.Join("testdata", "spec", "standard_spec.md")
	_, err = TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T1",
		TasksPath: largeTasks,
		SpecPath:  specPath,
	})
	if err == nil {
		t.Fatalf("expected error for >4 MiB tasks file, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}

	// 2. Invalid UTF-8 tasks.md
	invalidUTF8Tasks := filepath.Join(tmpDir, "invalid_utf8.md")
	if err := os.WriteFile(invalidUTF8Tasks, []byte{0xff, 0xfe, 0xfd}, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = TaskBrief(TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T1",
		TasksPath: invalidUTF8Tasks,
		SpecPath:  specPath,
	})
	if err == nil {
		t.Fatalf("expected error for invalid UTF-8 tasks file, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
}

func TestTaskBrief_DirectScopeRefusal(t *testing.T) {
	tmpDir := t.TempDir()
	runDir := filepath.Join(tmpDir, "run-direct")
	if err := os.MkdirAll(runDir, 0700); err != nil {
		t.Fatal(err)
	}

	identity := RunIdentity{
		SchemaVersion: 1,
		RunID:         "run-20261113T120000Z-a1b2c3",
		CreatedAt:     time.Now().UTC(),
		RunRoot:       tmpDir,
		RunDir:        runDir,
		Scope: Scope{
			Kind:     ScopeDirect,
			RepoRoot: tmpDir,
			Label:    "direct-fix",
		},
	}
	data, _ := json.Marshal(identity)
	if err := os.WriteFile(filepath.Join(runDir, "run.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	})
	if err == nil {
		t.Fatalf("expected refusal for direct-scope run, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
	if !strings.Contains(err.Error(), "not supported for direct-scope runs") {
		t.Errorf("expected message to mention direct-scope runs, got %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return data
}
