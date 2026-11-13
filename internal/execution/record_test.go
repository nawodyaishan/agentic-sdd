package execution

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func recordTestDocument(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	document := map[string]any{
		"schemaVersion": 1,
		"taskId":        "T01",
		"result":        "passed",
		"summary":       "ok",
	}
	if mutate != nil {
		mutate(document)
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode record: %v", err)
	}
	return data
}

func recordTestFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "records", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestRecordParseValidFixtures(t *testing.T) {
	claim, err := ParseClaim(recordTestFixture(t, "valid.json"), "")
	if err != nil {
		t.Fatalf("parse valid record: %v", err)
	}
	if claim.SchemaVersion != SchemaVersion || claim.TaskID != "T01" || claim.Result != ClaimPassed {
		t.Fatalf("claim = %+v", claim)
	}
	if claim.Summary != "The focused package tests passed." {
		t.Fatalf("summary = %q", claim.Summary)
	}
	if len(claim.ChangedPaths) != 1 || claim.ChangedPaths[0] != "internal/execution/record.go" {
		t.Fatalf("changed paths = %v", claim.ChangedPaths)
	}
	if len(claim.Checks) != 1 || claim.Checks[0].ExitCode != 0 || claim.Checks[0].LogPath == "" {
		t.Fatalf("checks = %+v", claim.Checks)
	}
	if claim.Agent == nil || claim.Agent.Role != "implementer" {
		t.Fatalf("agent = %+v", claim.Agent)
	}
	if len(claim.Unknowns) != 1 {
		t.Fatalf("unknowns = %v", claim.Unknowns)
	}

	blocked, err := ParseClaim(recordTestFixture(t, "blocked.json"), "T01")
	if err != nil {
		t.Fatalf("parse blocked record: %v", err)
	}
	if blocked.Result != ClaimBlocked || len(blocked.Blockers) != 1 || blocked.Blockers[0].Kind != "permission" {
		t.Fatalf("blocked claim = %+v", blocked)
	}
}

func TestRecordParseTaskResolution(t *testing.T) {
	if _, err := ParseClaim(recordTestFixture(t, "valid.json"), "T02"); !errors.Is(err, ErrUsage) {
		t.Fatalf("task mismatch error = %v, want ErrUsage", err)
	}
	noTask := recordTestDocument(t, func(document map[string]any) { delete(document, "taskId") })
	claim, err := ParseClaim(noTask, "direct")
	if err != nil {
		t.Fatalf("default task: %v", err)
	}
	if claim.TaskID != "direct" {
		t.Fatalf("task = %q, want direct", claim.TaskID)
	}
	if _, err := ParseClaim(noTask, ""); !errors.Is(err, ErrUsage) {
		t.Fatalf("missing task error = %v, want ErrUsage", err)
	}
	badTask := recordTestDocument(t, func(document map[string]any) { document["taskId"] = "../escape" })
	if _, err := ParseClaim(badTask, ""); !errors.Is(err, ErrUsage) {
		t.Fatalf("unsafe task error = %v, want ErrUsage", err)
	}
}

func TestRecordRejectsInvalidInput(t *testing.T) {
	rawNUL := []byte("{\"schemaVersion\":1,\"taskId\":\"T01\",\"result\":\"passed\",\"summary\":\"a\x00b\"}")
	oversize := []byte(`{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"` + strings.Repeat("x", (1<<20)+1) + `"}`)
	duplicate := recordTestFixture(t, "duplicate-key.json")
	valid := recordTestDocument(t, nil)

	cases := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"array", []byte("[]")},
		{"null", []byte("null")},
		{"invalid utf8", []byte{0xff, 0xfe}},
		{"raw NUL", rawNUL},
		{"oversize", oversize},
		{"trailing data", append(append([]byte{}, valid...), []byte(" {}")...)},
		{"duplicate key", duplicate},
		{"unknown field", recordTestFixture(t, "unknown-field.json")},
		{"case variant field", recordTestDocument(t, func(document map[string]any) {
			document["Summary"] = "case variant"
		})},
		{"unsupported schema", recordTestDocument(t, func(document map[string]any) { document["schemaVersion"] = 2 })},
		{"missing schema", recordTestDocument(t, func(document map[string]any) { delete(document, "schemaVersion") })},
		{"missing result", recordTestDocument(t, func(document map[string]any) { delete(document, "result") })},
		{"unknown result", recordTestDocument(t, func(document map[string]any) { document["result"] = "completed" })},
		{"long summary", recordTestDocument(t, func(document map[string]any) {
			document["summary"] = strings.Repeat("x", 8193)
		})},
		{"too many changed paths", recordTestDocument(t, func(document map[string]any) {
			paths := make([]string, 1001)
			for index := range paths {
				paths[index] = "a/b"
			}
			document["changedPaths"] = paths
		})},
		{"absolute changed path", recordTestDocument(t, func(document map[string]any) {
			document["changedPaths"] = []string{"/etc/passwd"}
		})},
		{"traversal changed path", recordTestDocument(t, func(document map[string]any) {
			document["changedPaths"] = []string{"../escape"}
		})},
		{"changed paths not an array", recordTestDocument(t, func(document map[string]any) {
			document["changedPaths"] = "a/b"
		})},
		{"decoded NUL summary", recordTestDocument(t, func(document map[string]any) {
			document["summary"] = "a\u0000b"
		})},
		{"unknown blocker kind", recordTestDocument(t, func(document map[string]any) {
			document["blockers"] = []map[string]any{{"kind": "surprise", "detail": "x"}}
		})},
		{"blocker without detail", recordTestDocument(t, func(document map[string]any) {
			document["blockers"] = []map[string]any{{"kind": "test"}}
		})},
		{"too many checks", recordTestDocument(t, func(document map[string]any) {
			checks := make([]map[string]any, 101)
			for index := range checks {
				checks[index] = map[string]any{"command": "go test ./...", "exitCode": 0}
			}
			document["checks"] = checks
		})},
		{"nested unknown check field", recordTestDocument(t, func(document map[string]any) {
			document["checks"] = []map[string]any{{"command": "go test ./...", "exitCode": 0, "shell": true}}
		})},
		{"check without exit code", recordTestDocument(t, func(document map[string]any) {
			document["checks"] = []map[string]any{{"command": "go test ./..."}}
		})},
		{"absolute log path", recordTestDocument(t, func(document map[string]any) {
			document["checks"] = []map[string]any{{"command": "go test ./...", "exitCode": 0, "logPath": "/tmp/log"}}
		})},
		{"too many unknowns", recordTestDocument(t, func(document map[string]any) {
			unknowns := make([]string, 101)
			for index := range unknowns {
				unknowns[index] = "unknown"
			}
			document["unknowns"] = unknowns
		})},
		{"agent not an object", recordTestDocument(t, func(document map[string]any) {
			document["agent"] = "implementer"
		})},
		{"agent unknown field", recordTestDocument(t, func(document map[string]any) {
			document["agent"] = map[string]any{"role": "child", "tokens": 10}
		})},
		{"nested duplicate key", []byte(`{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"ok","agent":{"role":"a","role":"b"}}`)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ParseClaim(testCase.data, "")
			if !errors.Is(err, ErrUsage) && !errors.Is(err, ErrUnsafePath) {
				t.Fatalf("ParseClaim() error = %v, want a validation error", err)
			}
		})
	}
}

func TestRecordIngestPassedRemainsAwaitingVerification(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	data := recordTestFixture(t, "valid.json")
	record, err := IngestRecord(identity.RunDir, "", data)
	if err != nil {
		t.Fatalf("ingest record: %v", err)
	}
	if !strings.HasPrefix(record.RecordID, "rec-") {
		t.Fatalf("record ID = %q", record.RecordID)
	}
	if record.RawDigest != DigestBytes(data) {
		t.Fatalf("raw digest = %q", record.RawDigest)
	}
	if record.ReceivedAt.IsZero() {
		t.Fatal("received timestamp is zero")
	}
	rawPath := filepath.Join(identity.RunDir, "records", record.RecordID+".json")
	if record.RawPath != rawPath {
		t.Fatalf("raw path = %q, want %q", record.RawPath, rawPath)
	}
	stored, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("read raw record: %v", err)
	}
	if string(stored) != string(data) {
		t.Fatal("stored record does not match the supplied bytes")
	}
	info, err := os.Stat(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("record mode = %o, want 600", info.Mode().Perm())
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateAwaitingVerification {
		t.Fatalf("checkpoint state = %q, want awaiting-verification", checkpoint.State)
	}
	if checkpoint.State == StateVerifiedEvidence {
		t.Fatal("a passed claim became verified evidence")
	}
	if checkpoint.Tasks["T01"].State != StateAwaitingVerification {
		t.Fatalf("task state = %q", checkpoint.Tasks["T01"].State)
	}
	if len(checkpoint.Records) != 1 || checkpoint.Records[0].RecordID != record.RecordID {
		t.Fatalf("checkpoint records = %+v, want the ingested claim", checkpoint.Records)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateAwaitingVerification {
		t.Fatalf("status state = %q, want awaiting-verification", status.State)
	}
	if len(status.Records) != 1 || status.Records[0].RecordID != record.RecordID {
		t.Fatalf("status records = %+v, want the ingested claim", status.Records)
	}
	if status.NextPermittedInvestigation != InvestigateChangedPaths {
		t.Fatalf("next investigation = %q", status.NextPermittedInvestigation)
	}
}

func TestRecordIngestBlockedYieldsBlocked(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	record, err := IngestRecord(identity.RunDir, "", recordTestFixture(t, "blocked.json"))
	if err != nil {
		t.Fatalf("ingest blocked record: %v", err)
	}
	if record.Result != ClaimBlocked {
		t.Fatalf("record result = %q", record.Result)
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.State != StateBlocked {
		t.Fatalf("checkpoint state = %q, want blocked", checkpoint.State)
	}
	if len(checkpoint.Blockers) != 1 || checkpoint.Blockers[0].Kind != "permission" {
		t.Fatalf("checkpoint blockers = %+v", checkpoint.Blockers)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != StateBlocked || status.NextPermittedInvestigation != InvestigateResolveBlocker {
		t.Fatalf("status = %q/%q", status.State, status.NextPermittedInvestigation)
	}
}

func TestRecordIngestRejectsInvalidWithoutArtifact(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	recordsDir := filepath.Join(identity.RunDir, "records")
	before, err := os.ReadDir(recordsDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := IngestRecord(identity.RunDir, "", recordTestFixture(t, "unknown-field.json")); !errors.Is(err, ErrUsage) {
		t.Fatalf("ingest invalid record error = %v, want ErrUsage", err)
	}
	after, err := os.ReadDir(recordsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("invalid record left an artifact: %d files before, %d after", len(before), len(after))
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.State != StatePrepared {
		t.Fatalf("invalid record changed checkpoint state to %q", checkpoint.State)
	}
}

func TestRecordIngestTaskMismatch(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	if _, err := IngestRecord(identity.RunDir, "T02", recordTestFixture(t, "valid.json")); !errors.Is(err, ErrUsage) {
		t.Fatalf("task mismatch error = %v, want ErrUsage", err)
	}
}

func TestRecordIngestDirectDefault(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	data := recordTestDocument(t, func(document map[string]any) { delete(document, "taskId") })
	record, err := IngestRecord(identity.RunDir, "direct", data)
	if err != nil {
		t.Fatalf("ingest direct record: %v", err)
	}
	if record.TaskID != "direct" {
		t.Fatalf("task = %q, want direct", record.TaskID)
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Tasks["direct"].State != StateAwaitingVerification {
		t.Fatalf("direct task state = %q", checkpoint.Tasks["direct"].State)
	}
}
