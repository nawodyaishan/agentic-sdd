package tui

import (
	"strings"
	"testing"
)

// Feature 015 / MCP-C / T38 — AC-54.
//
// The Install & Manage hub has four children — Install, Backups, Fleet Manager
// and MCP Tools. The first three keep their keys and indices; MCP Tools is the
// deliberate fourth child reached by `t` and Enter. This test also keeps Fleet
// Manager reachable by cursor plus Enter and by the `f` shortcut.
//
// The tests are offline and deterministic: they use the existing fake fleet
// reader seam, temporary-free in-memory catalogs and no filesystem, network or
// model calls.

// TestMCPAHubRegressionEntries asserts the four-entry hub contract.
func TestMCPAHubRegressionEntries(t *testing.T) {
	if len(installHubEntries) != 4 {
		t.Fatalf("len(installHubEntries) = %d, want 4", len(installHubEntries))
	}

	wantKeys := []string{"i", "b", "f", "t"}
	wantSections := []installSection{installSectionInstall, installSectionBackups, installSectionFleet, installSectionMCP}
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

	if got := strings.ToLower(installHubEntries[3].title); !strings.Contains(got, "mcp") {
		t.Errorf("fourth hub entry title = %q, want MCP Tools", installHubEntries[3].title)
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
		app = fleetKey(t, app, "j")
		if app.installHubIndex != 3 {
			t.Fatalf("after j, installHubIndex = %d, want 3 (MCP Tools)", app.installHubIndex)
		}
		// Bounds: forward past the last entry wraps to the first.
		app = fleetKey(t, app, "j")
		if app.installHubIndex != 0 {
			t.Fatalf("after wrap j, installHubIndex = %d, want 0", app.installHubIndex)
		}
		// Bounds: backward past the first entry wraps to MCP Tools at index 3.
		app = fleetKey(t, app, "k")
		if app.installHubIndex != 3 {
			t.Fatalf("after wrap k, installHubIndex = %d, want 3 (MCP Tools)", app.installHubIndex)
		}
		app = fleetKey(t, app, "k")
		if app.installHubIndex != 2 {
			t.Fatalf("after k, installHubIndex = %d, want 2 (Fleet)", app.installHubIndex)
		}
		// Enter on the Fleet row opens the Fleet section.
		app = fleetKey(t, app, "enter")
		if app.installSection != installSectionFleet {
			t.Fatalf("enter on Fleet, installSection = %v, want installSectionFleet", app.installSection)
		}
	})

	t.Run("k wraps to MCP then Fleet, then enter opens Fleet", func(t *testing.T) {
		app := newMCPAHubApp(t)
		app = fleetKey(t, app, "k")
		if app.installHubIndex != 3 {
			t.Fatalf("after k, installHubIndex = %d, want 3 (MCP Tools)", app.installHubIndex)
		}
		app = fleetKey(t, app, "k")
		if app.installHubIndex != 2 {
			t.Fatalf("after second k, installHubIndex = %d, want 2 (Fleet)", app.installHubIndex)
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
