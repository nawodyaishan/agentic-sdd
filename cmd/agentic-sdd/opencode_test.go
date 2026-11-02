package main

import (
	"agentic-sdd/internal/opencodeadapter"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeCommandsStayOfflineAndDefaultToPreview(t *testing.T) {
	var out, errout bytes.Buffer
	if code := runNonInteractive([]string{"opencode", "candidate", "--feature", "demo", "--build-path", "src"}, &out, &errout); code != 0 || !strings.Contains(out.String(), "openrouter/nvidia/nemotron") {
		t.Fatalf("candidate: %d %s", code, errout.String())
	}
	root := t.TempDir()
	staged := filepath.Join(t.TempDir(), "candidate.jsonc")
	if err := os.WriteFile(staged, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "config", "--workspace", root, "--candidate", staged}, &out, &errout); code != 0 || !strings.Contains(out.String(), `"original":"absent"`) {
		t.Fatalf("default preview: %d %s", code, errout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode")); !os.IsNotExist(err) {
		t.Fatal("preview installed")
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "session", "--auto"}, &out, &errout); code != 2 {
		t.Fatal("accepted unfinished/unsafe native operation")
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "apply", "--help"}, &out, &errout); code != 2 {
		t.Fatal("accepted unnamespaced config operation")
	}
}

func TestOpenCodeApplyFailureRetainsActionableBackupObservation(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-denial fixture requires unprivileged user")
	}
	root := t.TempDir()
	dir := filepath.Join(root, ".opencode")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(`{"unrelated":"preserve"}`), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := opencodeadapter.Candidate("demo", "src")
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(t.TempDir(), "candidate.jsonc")
	if err := os.WriteFile(staged, data, 0600); err != nil {
		t.Fatal(err)
	}
	p, err := opencodeadapter.Preview(root, staged)
	if err != nil {
		t.Fatal(err)
	}
	backup := t.TempDir()
	if err := os.Chmod(backup, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)
	var out, errs bytes.Buffer
	code := runNonInteractive([]string{"opencode", "config", "apply", "--workspace", root, "--candidate", staged, "--backup-root", backup, "--expect-original", p.Original, "--expect-candidate", p.Candidate}, &out, &errs)
	var observation struct {
		Execution   string                      `json:"execution"`
		Transaction opencodeadapter.Transaction `json:"transaction"`
	}
	if code != 2 || json.Unmarshal(out.Bytes(), &observation) != nil || observation.Execution != "incomplete" || observation.Transaction.Backup == "" {
		t.Fatalf("lost partial observation: %d %s %s", code, out.String(), errs.String())
	}
}
