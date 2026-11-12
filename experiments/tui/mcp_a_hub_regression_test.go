package tui

import (
	"strings"
	"testing"
)

// Feature 015 / MCP-A / T30 — AC-54.
//
// The MCP amendment is staged: MCP-C (the TUI hub child) is deferred, so the
// Install & Manage hub still has exactly three children — Install, Backups and
// Fleet Manager. This file is the MCP-A regression guard: the entry list and
// its keys stay stable, and Fleet Manager stays reachable from the hub by
// cursor plus Enter and by the `f` shortcut. When MCP-C lands, it adds the
// fourth child and updates this test deliberately.
//
// The tests are offline and deterministic: they use the existing fake fleet
// reader seam, temporary-free in-memory catalogs and no filesystem, network or
// model calls.

// TestMCPAHubRegressionEntries asserts the MCP-A hub contract: three entries
// with the keys i, b, f in order, bound to the install, backups and fleet
// sections, and no MCP entry of any kind.
func TestMCPAHubRegressionEntries(t *testing.T) {
	if len(installHubEntries) != 3 {
		t.Fatalf("len(installHubEntries) = %d, want 3 (MCP-C adds the fourth child later)", len(installHubEntries))
	}

	wantKeys := []string{"i", "b", "f"}
	wantSections := []installSection{installSectionInstall, installSectionBackups, installSectionFleet}
	for i, entry := range installHubEntries {
		if entry.key != wantKeys[i] {
			t.Errorf("installHubEntries[%d].key = %q, want %q", i, entry.key, wantKeys[i])
		}
		if entry.section != wantSections[i] {
			t.Errorf("installHubEntries[%d].section = %v, want %v", i, entry.section, wantSections[i])
		}
		if strings.TrimSpace(entry.title) == "" {
			t.Errorf("installHubEntries[%d] has an empty title", i)
		}
	}

	// No MCP hub entry exists in MCP-A; MCP-C owns the future fourth child.
	for i, entry := range installHubEntries {
		blob := strings.ToLower(entry.key + " " + entry.title + " " + entry.desc)
		if strings.Contains(blob, "mcp") {
			t.Errorf("installHubEntries[%d] (%q) mentions MCP; MCP-C is deferred out of MCP-A", i, entry.title)
		}
	}
}

// newMCPAHubApp builds the root shell with the deterministic fake fleet reader
// and returns it positioned on the Install & Manage hub.
func newMCPAHubApp(t *testing.T) App {
	t.Helper()
	cfg := TUIConfig{Width: 100, Height: 30, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})
	app = fleetKey(t, app, "2")
	if app.ActiveRoute() != RouteInstall {
		t.Fatalf("route = %v, want RouteInstall", app.ActiveRoute())
	}
	if app.installSection != installSectionHub {
		t.Fatalf("installSection = %v, want installSectionHub", app.installSection)
	}
	if app.installHubIndex != 0 {
		t.Fatalf("initial installHubIndex = %d, want 0", app.installHubIndex)
	}
	return app
}

// TestMCPAHubRegressionFleetCursorAndKeys asserts that MCP-A hub navigation
// still reaches Fleet Manager at index 2 with j/k (including wrap bounds) and
// that Enter and the `f` shortcut open installSectionFleet.
func TestMCPAHubRegressionFleetCursorAndKeys(t *testing.T) {
	t.Run("j reaches fleet and wraps at both bounds", func(t *testing.T) {
		app := newMCPAHubApp(t)

		app = fleetKey(t, app, "j")
		if app.installHubIndex != 1 {
			t.Fatalf("after j, installHubIndex = %d, want 1", app.installHubIndex)
		}
		app = fleetKey(t, app, "j")
		if app.installHubIndex != 2 {
			t.Fatalf("after j, installHubIndex = %d, want 2 (Fleet)", app.installHubIndex)
		}
		// Bounds: forward past the last entry wraps to the first.
		app = fleetKey(t, app, "j")
		if app.installHubIndex != 0 {
			t.Fatalf("after wrap j, installHubIndex = %d, want 0", app.installHubIndex)
		}
		// Bounds: backward past the first entry wraps to Fleet at index 2.
		app = fleetKey(t, app, "k")
		if app.installHubIndex != 2 {
			t.Fatalf("after wrap k, installHubIndex = %d, want 2 (Fleet)", app.installHubIndex)
		}
		// Enter on the Fleet row opens the Fleet section.
		app = fleetKey(t, app, "enter")
		if app.installSection != installSectionFleet {
			t.Fatalf("enter on Fleet, installSection = %v, want installSectionFleet", app.installSection)
		}
	})

	t.Run("k from the top reaches fleet then enter opens it", func(t *testing.T) {
		app := newMCPAHubApp(t)
		app = fleetKey(t, app, "k")
		if app.installHubIndex != 2 {
			t.Fatalf("after k, installHubIndex = %d, want 2 (Fleet)", app.installHubIndex)
		}
		app = fleetKey(t, app, "enter")
		if app.installSection != installSectionFleet {
			t.Fatalf("enter on Fleet, installSection = %v, want installSectionFleet", app.installSection)
		}
		if bc := app.fleet.Breadcrumb(); bc != "Install & Manage / Fleet Manager" {
			t.Fatalf("Fleet breadcrumb = %q, want 'Install & Manage / Fleet Manager'", bc)
		}
	})

	t.Run("f shortcut opens fleet from the hub", func(t *testing.T) {
		app := newMCPAHubApp(t)
		app = fleetKey(t, app, "f")
		if app.installSection != installSectionFleet {
			t.Fatalf("pressing f from hub, installSection = %v, want installSectionFleet", app.installSection)
		}
	})

	t.Run("esc returns from fleet to the hub", func(t *testing.T) {
		app := newMCPAHubApp(t)
		app = fleetKey(t, app, "f")
		app = fleetKey(t, app, "esc")
		if app.installSection != installSectionHub {
			t.Fatalf("esc from fleet, installSection = %v, want installSectionHub", app.installSection)
		}
	})
}
