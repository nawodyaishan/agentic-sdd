package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFleetDefaultReaderSafeNotAvailable(t *testing.T) {
	reader := DefaultFleetReader{}

	// 1. With empty source
	res, err := reader.Status(context.Background(), "")
	if err != nil {
		t.Fatalf("DefaultFleetReader.Status error = %v, want nil", err)
	}
	if res.Available {
		t.Fatal("DefaultFleetReader returned Available = true, want false")
	}
	if res.SourceState != "not selected" {
		t.Errorf("SourceState = %q, want 'not selected'", res.SourceState)
	}
	if res.RuntimeStatus != "not checked" {
		t.Errorf("RuntimeStatus = %q, want 'not checked'", res.RuntimeStatus)
	}
	if !strings.Contains(res.Message, "not available") {
		t.Errorf("Message = %q, want 'not available'", res.Message)
	}

	// 2. With explicit source
	resExplicit, err := reader.Status(context.Background(), "/nonexistent/fleet.json")
	if err != nil {
		t.Fatalf("DefaultFleetReader.Status error = %v, want nil", err)
	}
	if resExplicit.Available {
		t.Fatal("DefaultFleetReader returned Available = true, want false")
	}
	if resExplicit.SourceState != "not available" {
		t.Errorf("SourceState = %q, want 'not available'", resExplicit.SourceState)
	}
	if resExplicit.RuntimeStatus != "not checked" {
		t.Errorf("RuntimeStatus = %q, want 'not checked'", resExplicit.RuntimeStatus)
	}
}

func TestFleetZeroWritesAndNoRealHome(t *testing.T) {
	tempRoot := t.TempDir()
	fakeHome := filepath.Join(tempRoot, "home")
	opencodeDir := filepath.Join(fakeHome, ".config", "opencode", "agents")
	if err := os.MkdirAll(opencodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(opencodeDir, "sdd-implementer.md")
	originalContent := []byte("# original managed agent\n")
	if err := os.WriteFile(targetFile, originalContent, 0o644); err != nil {
		t.Fatal(err)
	}
	fleetFile := filepath.Join(tempRoot, "fleet.json")
	fleetContent := []byte(`{"version": 1}`)
	if err := os.WriteFile(fleetFile, fleetContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Snapshot timestamps/contents
	targetInfoBefore, err := os.Stat(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	fleetInfoBefore, err := os.Stat(fleetFile)
	if err != nil {
		t.Fatal(err)
	}

	// Set up App with DefaultFleetReader (no fake reader) and temporary home
	cfg := TUIConfig{
		Width:       110,
		Height:      30,
		Home:        fakeHome,
		Repo:        tempRoot,
		FleetConfig: fleetFile,
		Color:       false,
	}
	cat := Catalog{Home: fakeHome, Repo: tempRoot}
	app := NewApp(cfg, cat)

	// Run through full Fleet journey
	app = fleetKey(t, app, "2") // Install hub
	app = fleetKey(t, app, "f") // Fleet Manager
	if app.installSection != installSectionFleet {
		t.Fatalf("installSection = %v, want installSectionFleet", app.installSection)
	}

	// Edit source path input with literal characters including q, j, k, /
	app = fleetKey(t, app, "s")
	if !app.fleet.IsInputFocused() {
		t.Fatal("expected IsInputFocused() true when editing source")
	}
	app = fleetKey(t, app, "ctrl+u")
	for _, ch := range "/tmp/other-qjk12.json" {
		app = fleetPress(t, app, ch, string(ch))
	}
	if app.quitting {
		t.Fatal("literal 'q' in source input must not quit")
	}
	app = fleetKey(t, app, "enter") // submits source path and calls Reload()
	if app.fleet.SourcePath() != "/tmp/other-qjk12.json" {
		t.Fatalf("SourcePath = %q, want '/tmp/other-qjk12.json'", app.fleet.SourcePath())
	}

	// Reload explicitly
	app = fleetKey(t, app, "r")

	// Switch to Profiles, search, toggle detail, switch back
	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "/")
	app = fleetPress(t, app, 'x', "x")
	app = fleetKey(t, app, "esc")
	app = fleetKey(t, app, "esc") // clear filter
	app = fleetKey(t, app, "1")   // overview

	// Verify zero filesystem writes in tempRoot
	entriesAfter, err := os.ReadDir(opencodeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entriesAfter) != 1 || entriesAfter[0].Name() != "sdd-implementer.md" {
		t.Fatalf("opencodeDir entries changed: %+v", entriesAfter)
	}
	targetInfoAfter, err := os.Stat(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if !targetInfoAfter.ModTime().Equal(targetInfoBefore.ModTime()) {
		t.Fatalf("targetFile modtime changed: %v -> %v", targetInfoBefore.ModTime(), targetInfoAfter.ModTime())
	}
	gotTarget, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotTarget) != string(originalContent) {
		t.Fatalf("targetFile content mutated")
	}

	fleetInfoAfter, err := os.Stat(fleetFile)
	if err != nil {
		t.Fatal(err)
	}
	if !fleetInfoAfter.ModTime().Equal(fleetInfoBefore.ModTime()) {
		t.Fatalf("fleetFile modtime changed: %v -> %v", fleetInfoBefore.ModTime(), fleetInfoAfter.ModTime())
	}

	// Ensure no backup directories were created
	if _, err := os.Stat(filepath.Join(fakeHome, ".agentic-sdd")); !os.IsNotExist(err) {
		t.Fatalf("unexpected .agentic-sdd directory created in fakeHome")
	}

	// Ensure no write transaction was ever started on OpsHost
	if app.Busy() {
		t.Fatal("app.Busy() = true, expected false for read-only Fleet")
	}
	if _, ok := app.LastWriteResult(); ok {
		t.Fatal("expected no LastWriteResult in read-only Fleet")
	}

	// Give any potential goroutine a tick and re-verify
	time.Sleep(5 * time.Millisecond)
	if app.Busy() {
		t.Fatal("app.Busy() became true asynchronously")
	}
}
