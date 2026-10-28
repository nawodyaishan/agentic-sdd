package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"agentic-sdd/internal/harnesscheck"
)

func TestUsageAndInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"--scope", "focused"}, {"--deadline", "0s"}, {"--unknown"}, {"extra"}} {
		var out, stderr bytes.Buffer
		if code := run(context.Background(), args, &out, &stderr); code != 2 {
			t.Fatalf("%v: %d", args, code)
		}
	}
	var out, stderr bytes.Buffer
	if code := run(context.Background(), []string{"--help"}, &out, &stderr); code != 0 || out.Len() == 0 || stderr.Len() != 0 {
		t.Fatalf("help: %d", code)
	}
}

func TestMissingPrerequisiteHasStructuredBlockedResult(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	contract := filepath.Join(root, "checks.json")
	data := `{"version":1,"checks":[{"id":"missing","cwd":".","scopes":["required"],"prerequisites":[],"timeout_seconds":1,"effects":{"writes":[],"network":"none"},"steps":[{"argv":["sdd-no-such-command-1739"],"env":{}}],"local_recipe":[],"expected_exit":0,"evidence":"missing.log"}]}`
	if err := os.WriteFile(contract, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := run(context.Background(), []string{"--scope", "required", "--workspace", workspace, "--contract", contract, "--output", filepath.Join(root, "logs")}, &out, &stderr)
	var result harnesscheck.Result
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if code != 2 || result.Execution != "blocked" {
		t.Fatalf("%d %+v %s", code, result, stderr.String())
	}
}
