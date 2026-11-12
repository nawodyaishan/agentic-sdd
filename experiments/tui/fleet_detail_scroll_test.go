package tui

import (
	"strings"
	"testing"
)

func TestFleetProfilesWideDetailScroll(t *testing.T) {
	// Full App wrapper at width 120 (wide layout >= 110)
	cfg := TUIConfig{Width: 120, Height: 30, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	// 1. Enter Fleet -> Profiles
	app = fleetKey(t, app, "2") // Install & Manage hub
	app = fleetKey(t, app, "f") // Fleet Manager
	app = fleetKey(t, app, "p") // Profiles view

	if app.fleet.ActiveView() != FleetViewProfiles {
		t.Fatalf("active view = %v, want %v", app.fleet.ActiveView(), FleetViewProfiles)
	}

	// Initial state: list focused, selection at index 0, detailScroll at 0
	if app.fleet.focusDetail {
		t.Fatal("expected focusDetail false initially")
	}
	if app.fleet.selectedIndex != 0 {
		t.Fatalf("selectedIndex = %d, want 0", app.fleet.selectedIndex)
	}
	if app.fleet.detailScroll != 0 {
		t.Fatalf("detailScroll = %d, want 0", app.fleet.detailScroll)
	}

	// Verify full App View shows side-by-side detail card title at scroll 0
	fullView0 := app.View().Content
	if !strings.Contains(fullView0, "Profile details: sdd-implement-9r-1") {
		t.Fatalf("expected full app view to contain 'Profile details: sdd-implement-9r-1':\n%s", fullView0)
	}

	// Verify contextual footer shows 'l: focus detail'
	footerInitial := app.fleet.renderProfilesFooter()
	if !strings.Contains(footerInitial, "l: focus detail") {
		t.Fatalf("footer missing 'l: focus detail': %s", footerInitial)
	}
	if !strings.Contains(footerInitial, "j/k: navigate") {
		t.Fatalf("footer missing 'j/k: navigate': %s", footerInitial)
	}

	// Verify global Tab still cycles root App focus without touching focusDetail
	if app.focus != FocusContent {
		t.Fatalf("expected initial focus = FocusContent, got %v", app.focus)
	}
	app = fleetKey(t, app, "tab")
	if app.focus != FocusNav {
		t.Fatalf("expected Tab to cycle focus to FocusNav, got %v", app.focus)
	}
	if app.fleet.focusDetail {
		t.Fatal("global Tab must not toggle Fleet focusDetail")
	}
	app = fleetKey(t, app, "tab")
	if app.focus != FocusContent {
		t.Fatalf("expected second Tab to cycle focus back to FocusContent, got %v", app.focus)
	}

	// 2. Press 'l' to toggle focus to detail pane
	app = fleetKey(t, app, "l")
	if !app.fleet.focusDetail {
		t.Fatal("expected focusDetail true after pressing 'l'")
	}

	// Contextual footer should now show 'j/k: scroll detail' and 'l: focus list'
	footerDetail := app.fleet.renderProfilesFooter()
	if !strings.Contains(footerDetail, "j/k: scroll detail") {
		t.Fatalf("footer missing 'j/k: scroll detail': %s", footerDetail)
	}
	if !strings.Contains(footerDetail, "l: focus list") {
		t.Fatalf("footer missing 'l: focus list': %s", footerDetail)
	}

	// 3. Press 'j' -> detail pane scrolls, list selection DOES NOT move
	app = fleetKey(t, app, "j")
	if app.fleet.selectedIndex != 0 {
		t.Fatalf("selectedIndex = %d, want 0 (selection must not move while focusDetail is active)", app.fleet.selectedIndex)
	}
	if app.fleet.detailScroll != 1 {
		t.Fatalf("detailScroll = %d, want 1 after pressing 'j'", app.fleet.detailScroll)
	}

	// At scroll 1, the first line (title) has scrolled out of the detail card in full App View
	fullView1 := app.View().Content
	if strings.Contains(fullView1, "Profile details: sdd-implement-9r-1") {
		t.Fatalf("expected title to have scrolled out of full app view at detailScroll=1:\n%s", fullView1)
	}
	// List item remains visible in the left pane
	if !strings.Contains(fullView1, "sdd-implement-9r-1") {
		t.Fatalf("expected list item to remain visible at detailScroll=1:\n%s", fullView1)
	}

	// Press 'j' again -> detailScroll becomes 2
	app = fleetKey(t, app, "j")
	if app.fleet.selectedIndex != 0 {
		t.Fatalf("selectedIndex = %d, want 0", app.fleet.selectedIndex)
	}
	if app.fleet.detailScroll != 2 {
		t.Fatalf("detailScroll = %d, want 2 after second 'j'", app.fleet.detailScroll)
	}

	// 4. Press 'k' -> scrolls detail pane back up
	app = fleetKey(t, app, "k")
	if app.fleet.selectedIndex != 0 {
		t.Fatalf("selectedIndex = %d, want 0", app.fleet.selectedIndex)
	}
	if app.fleet.detailScroll != 1 {
		t.Fatalf("detailScroll = %d, want 1 after pressing 'k'", app.fleet.detailScroll)
	}

	app = fleetKey(t, app, "k")
	if app.fleet.detailScroll != 0 {
		t.Fatalf("detailScroll = %d, want 0 after second 'k'", app.fleet.detailScroll)
	}
	fullViewBack := app.View().Content
	if !strings.Contains(fullViewBack, "Profile details: sdd-implement-9r-1") {
		t.Fatalf("expected title to reappear after scrolling back to 0:\n%s", fullViewBack)
	}

	// 5. Press 'l' again to toggle focus back to list
	app = fleetKey(t, app, "l")
	if app.fleet.focusDetail {
		t.Fatal("expected focusDetail false after toggling 'l' again")
	}

	// Contextual footer back to list navigation
	footerList := app.fleet.renderProfilesFooter()
	if !strings.Contains(footerList, "l: focus detail") {
		t.Fatalf("footer missing 'l: focus detail' after toggle back: %s", footerList)
	}

	// Now 'j' navigates the list
	app = fleetKey(t, app, "j")
	if app.fleet.selectedIndex != 1 {
		t.Fatalf("selectedIndex = %d, want 1 after 'j' with list focused", app.fleet.selectedIndex)
	}
	if app.fleet.detailScroll != 0 {
		t.Fatalf("detailScroll = %d, want 0 after moving list selection", app.fleet.detailScroll)
	}

	// 6. Test Esc key exits detail focus when focusDetail is active
	app = fleetKey(t, app, "l")
	if !app.fleet.focusDetail {
		t.Fatal("expected focusDetail true after 'l'")
	}
	app = fleetKey(t, app, "esc")
	if app.fleet.focusDetail {
		t.Fatal("expected focusDetail false after 'esc'")
	}
	if app.fleet.ActiveView() != FleetViewProfiles {
		t.Fatalf("esc from detail focus should remain in Profiles, got %v", app.fleet.ActiveView())
	}
}

func TestFleetProfilesNarrowLayoutUnchanged(t *testing.T) {
	for _, width := range []int{80, 100} {
		cfg := TUIConfig{Width: width, Height: 24, Color: false}
		reader := &trackingFleetReader{customRes: testFleetStatusView()}
		app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

		app = fleetKey(t, app, "2")
		app = fleetKey(t, app, "f")
		app = fleetKey(t, app, "p")

		// In narrow/compact mode (< 110), footer retains 'Enter: details' and does not show 'l: focus detail'
		footer := app.fleet.renderProfilesFooter()
		if !strings.Contains(footer, "Enter: details") {
			t.Fatalf("width %d footer missing 'Enter: details': %s", width, footer)
		}
		if strings.Contains(footer, "l: focus detail") {
			t.Fatalf("width %d footer must not contain 'l: focus detail': %s", width, footer)
		}

		// Pressing 'l' in narrow mode does not enable focusDetail
		app = fleetKey(t, app, "l")
		if app.fleet.focusDetail {
			t.Fatalf("width %d: focusDetail must not be enabled at narrow width", width)
		}

		// 'j' navigates list normally
		if app.fleet.selectedIndex != 0 {
			t.Fatalf("width %d: selectedIndex = %d, want 0", width, app.fleet.selectedIndex)
		}
		app = fleetKey(t, app, "j")
		if app.fleet.selectedIndex != 1 {
			t.Fatalf("width %d: selectedIndex = %d, want 1", width, app.fleet.selectedIndex)
		}
	}
}
