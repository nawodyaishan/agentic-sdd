package tui

// Read-only production adapter tests. Every fixture uses t.TempDir(): no real
// home, network, model, Git or credential is read or written.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/fleet"
)

func writeFleetFixture(t *testing.T, dir, instances string) string {
	t.Helper()
	path := filepath.Join(dir, "fleet.json")
	body := `{"schemaVersion":1,"fleet":"test","instances":{` + instances + `}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func singleFleetInstance(model string) string {
	return `"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"` + model + `"}`
}

func TestAppOpsFleetReaderMapsStatusWithoutWrites(t *testing.T) {
	home := t.TempDir()
	configPath := writeFleetFixture(t, home, singleFleetInstance("9router/ag/gemini-3.8-flash-high"))

	reader := AppOpsFleetReader{Home: home}
	view, err := reader.Status(context.Background(), configPath)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !view.Available || view.SourceState != "loaded" {
		t.Fatalf("view = %+v, want a loaded available status", view)
	}
	if view.TotalInstances != 1 || view.EnabledCount != 1 || view.DisabledCount != 0 {
		t.Fatalf("counts = total %d enabled %d disabled %d, want 1/1/0", view.TotalInstances, view.EnabledCount, view.DisabledCount)
	}
	if len(view.Instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(view.Instances))
	}
	inst := view.Instances[0]
	if inst.InstanceID != "sdd-implement-1" || inst.SourceRef != "preset:sdd-implement-1" {
		t.Fatalf("instance identity = %+v", inst)
	}
	if inst.ConfiguredModel != "9router/ag/gemini-3.8-flash-high" || inst.ProviderRoute != "9router" {
		t.Fatalf("model mapping = configured %q route %q", inst.ConfiguredModel, inst.ProviderRoute)
	}
	if !inst.Enabled || inst.InstalledState != "not installed" {
		t.Fatalf("instance state = enabled %t state %q", inst.Enabled, inst.InstalledState)
	}

	targetRoot := filepath.Join(home, ".config", "opencode", "agents")
	if _, err := os.Stat(targetRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("status read created or touched the target root: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil || !strings.Contains(string(data), `"schemaVersion":1`) {
		t.Fatalf("config was modified by a read: %v\n%s", err, data)
	}
}

func TestAppOpsFleetReaderReportsMissingSourceTruthfully(t *testing.T) {
	home := t.TempDir()
	reader := AppOpsFleetReader{Home: home}
	missing := filepath.Join(home, "absent.json")
	view, err := reader.Status(context.Background(), missing)
	if err == nil {
		t.Fatal("Status accepted a missing source")
	}
	if view.Available || view.SourceState != "missing" {
		t.Fatalf("view = %+v, want an unavailable missing status", view)
	}
}

func TestAppOpsFleetEditorPreviewNoWrites(t *testing.T) {
	home := t.TempDir()
	configPath := writeFleetFixture(t, home, singleFleetInstance("9router/ag/gemini-3.8-flash-high"))
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	editor := AppOpsFleetEditor{Home: home}
	preview, err := editor.PreviewSource(context.Background(), configPath, []FleetModelEdit{{
		InstanceID: "sdd-implement-1",
		Model:      "opencode-go/deepseek-v4.1-flash",
		Enabled:    true,
	}})
	if err != nil {
		t.Fatalf("PreviewSource: %v", err)
	}
	if !preview.Available || preview.Op != "update" {
		t.Fatalf("preview = %+v, want an available update", preview)
	}
	if preview.BaseDigest == "" || preview.NewDigest == "" || preview.BaseDigest == preview.NewDigest {
		t.Fatalf("digests = base %q new %q, want a real change", preview.BaseDigest, preview.NewDigest)
	}
	joined := strings.Join(preview.DiffLines, "\n")
	if !strings.Contains(joined, "-") || !strings.Contains(joined, "+") || !strings.Contains(joined, "deepseek-v4.1-flash") {
		t.Fatalf("diff does not show the model change:\n%s", joined)
	}

	cfg, err := fleet.ParseConfig(bytes.NewReader(preview.NewBytes))
	if err != nil {
		t.Fatalf("normalized bytes do not parse: %v\n%s", err, preview.NewBytes)
	}
	inst := cfg.Instances["sdd-implement-1"]
	if inst.Model != "opencode-go/deepseek-v4.1-flash" || !inst.IsEnabled() {
		t.Fatalf("normalized instance = %+v", inst)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("source preview wrote the selected file")
	}
}

func TestAppOpsFleetEditorPreviewPreservesTopLevelMetadata(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, "fleet.json")
	// Write the canonical encoding so unchanged top-level lines align in the
	// line diff; a compact fixture would hide the field-level regression.
	fixture := &fleet.Config{
		SchemaVersion: fleet.SchemaVersion,
		Fleet:         "team",
		Preset:        "free-first",
		Instances: map[string]fleet.Instance{
			"sdd-implement-1": {Source: "preset:sdd-implement-1", Model: "9router/ag/gemini-3.8-flash-high"},
		},
	}
	body, err := fleet.EncodeConfig(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	editor := AppOpsFleetEditor{Home: home}
	preview, err := editor.PreviewSource(context.Background(), configPath, []FleetModelEdit{{
		InstanceID: "sdd-implement-1",
		Model:      "opencode-go/deepseek-v4.1-flash",
		Enabled:    true,
	}})
	if err != nil {
		t.Fatalf("PreviewSource: %v", err)
	}
	if !preview.Available || preview.Op != "update" {
		t.Fatalf("preview = %+v, want an available update", preview)
	}

	cfg, err := fleet.ParseConfig(bytes.NewReader(preview.NewBytes))
	if err != nil {
		t.Fatalf("normalized bytes do not parse: %v\n%s", err, preview.NewBytes)
	}
	if cfg.Fleet != "team" || cfg.Preset != "free-first" {
		t.Fatalf("normalized labels = %q/%q, want team/free-first", cfg.Fleet, cfg.Preset)
	}
	if inst := cfg.Instances["sdd-implement-1"]; inst.Model != "opencode-go/deepseek-v4.1-flash" || !inst.IsEnabled() {
		t.Fatalf("normalized instance = %+v", inst)
	}

	joined := strings.Join(preview.DiffLines, "\n")
	for _, line := range preview.DiffLines {
		if strings.HasPrefix(line, "-") && (strings.Contains(line, `"fleet"`) || strings.Contains(line, `"preset"`)) {
			t.Fatalf("preview proposes deleting untouched top-level metadata:\n%s", joined)
		}
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("source preview wrote the selected file")
	}
}

func TestAppOpsFleetPlannerPreviewAndNoWrites(t *testing.T) {
	home := t.TempDir()
	configPath := writeFleetFixture(t, home, singleFleetInstance("9router/ag/gemini-3.8-flash-high"))

	planner := AppOpsFleetPlanner{Home: home}
	preview, err := planner.PreviewPlan(context.Background(), configPath)
	if err != nil {
		t.Fatalf("PreviewPlan: %v", err)
	}
	if !preview.Available || preview.PlanID == "" || preview.Token == "" || preview.Plan == nil {
		t.Fatalf("preview identity = %+v", preview)
	}
	if preview.Counts[fleet.FileOpCreate] != 1 || len(preview.Files) != 1 {
		t.Fatalf("counts/files = %+v / %d, want 1 create", preview.Counts, len(preview.Files))
	}
	if preview.Files[0].Op != fleet.FileOpCreate || len(preview.Files[0].DiffLines) == 0 {
		t.Fatalf("file preview = %+v", preview.Files[0])
	}
	if len(preview.DiffLines) == 0 || !strings.Contains(strings.Join(preview.DiffLines, "\n"), "+") {
		t.Fatalf("deployment diff is empty:\n%s", strings.Join(preview.DiffLines, "\n"))
	}
	target := filepath.Join(home, ".config", "opencode", "agents", "sdd-implement-1.md")
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only plan created the target: %v", err)
	}
}

func TestAppOpsFleetPlannerBackupsAndRestoreRefusal(t *testing.T) {
	home := t.TempDir()
	planner := AppOpsFleetPlanner{Home: home}

	backups, err := planner.ListBackups(context.Background())
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("backups = %+v, want none", backups)
	}
	if _, err := os.Stat(filepath.Join(home, ".agentic-sdd", "backups")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup listing created a directory: %v", err)
	}

	_, err = planner.PreviewRestore(context.Background(), "20260101T000000.000000000Z")
	if err == nil {
		t.Fatal("PreviewRestore accepted a nonexistent backup")
	}
}
