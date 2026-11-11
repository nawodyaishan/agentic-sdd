package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/fleet"
)

func TestFleetApplyCLI(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "fleet.json")
	configJSON := `{
  "schemaVersion": 1,
  "instances": {
    "sdd-lightweight": {
      "source": "agent:sdd-lightweight",
      "model": "9router/ag/gemini-3.8-flash-high"
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")

	var out, errOut bytes.Buffer
	if code := runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--home", home}, &out, &errOut); code != 0 {
		t.Fatalf("fleet plan failed (%d): %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &out, &errOut); code != 0 {
		t.Fatalf("fleet apply failed (%d): %s%s", code, out.String(), errOut.String())
	}
	target := filepath.Join(fleet.TargetRootForHome(home), "sdd-lightweight.md")
	installed, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("installed profile missing: %v", err)
	}
	if !strings.Contains(string(installed), "model: 9router/ag/gemini-3.8-flash-high") {
		t.Fatalf("model override missing: %s", installed)
	}
	live, err := fleet.ReadLiveManifest(home)
	if err != nil || live == nil || live.ID == "" {
		t.Fatalf("live fleet manifest missing: %v", err)
	}

	// The same create-plan is stale once the target exists.
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &out, &errOut); code != 1 {
		t.Fatalf("expected stale plan exit 1, got %d: %s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "stale plan") {
		t.Fatalf("expected stale plan diagnostic, got: %s", errOut.String())
	}

	// A regenerated plan is an idempotent no-op.
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--home", home}, &out, &errOut); code != 0 {
		t.Fatalf("fleet plan regeneration failed (%d): %s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &out, &errOut); code != 0 {
		t.Fatalf("no-op apply failed (%d): %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "No changes") {
		t.Fatalf("expected a no-op message, got: %s", out.String())
	}

	// Restore the first transaction through the CLI. The target still matches
	// the recorded installed digest, so the reverse proceeds; the profile the
	// transaction created is removed.
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "restore", "--backup", live.ID, "--home", home}, &out, &errOut); code != 0 {
		t.Fatalf("fleet restore failed (%d): %s%s", code, out.String(), errOut.String())
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("restore did not remove the created profile: %v", err)
	}

	// Reinstall, then confirm a manual edit is a conflict and is never
	// overwritten.
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--home", home}, &out, &errOut); code != 0 {
		t.Fatalf("fleet plan after restore failed (%d): %s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &out, &errOut); code != 0 {
		t.Fatalf("fleet apply after restore failed (%d): %s", code, errOut.String())
	}
	if err := os.WriteFile(target, []byte("manual edit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--home", home}, &out, &errOut); code != 1 {
		t.Fatalf("expected conflict plan exit 1, got %d: %s", code, out.String())
	}
	if !strings.Contains(out.String(), "[conflict]") {
		t.Fatalf("expected conflict output, got: %s", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &out, &errOut); code != 1 {
		t.Fatalf("expected conflict apply exit 1, got %d: %s", code, errOut.String())
	}
	if got, _ := os.ReadFile(target); string(got) != "manual edit\n" {
		t.Fatalf("conflict overwrote manual edit: %q", got)
	}

	out.Reset()
	errOut.Reset()
	if code := runNonInteractive([]string{"fleet", "restore", "--backup", "../escape", "--home", home}, &out, &errOut); code != 2 {
		t.Fatalf("expected malformed backup id exit 2, got %d: %s", code, errOut.String())
	}
}
