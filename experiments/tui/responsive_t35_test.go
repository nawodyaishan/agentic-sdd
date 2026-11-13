package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/skillsync"
)

// Helper catalogs for T35 state variations

func makeT35CatalogBase() Catalog {
	return Catalog{
		Home:       "/test/home",
		Repo:       "/test/repo",
		BackupRoot: "/test/home/.agentic-sdd/backups",
	}
}

func makeT35FreshCatalog() Catalog {
	cat := makeT35CatalogBase()
	cat.Items = []CatalogItem{
		{
			ID:       "agentic-sdd-router",
			Name:     "SDD Router",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Route a request through combined spec, plan, tasks approval.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMissing, Path: "/test/home/.claude/skills/router"},
				{Client: "codex", Status: skillsync.InventoryStatusMissing, Path: "/test/home/.codex/skills/router"},
			},
		},
		{
			ID:       "agentic-sdd-plan",
			Name:     "SDD Plan",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Draft a feature plan from the draft spec.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMissing, Path: "/test/home/.claude/skills/plan"},
			},
		},
	}
	return cat
}

func makeT35PartialCatalog() Catalog {
	cat := makeT35CatalogBase()
	cat.Items = []CatalogItem{
		{
			ID:       "agentic-sdd-router",
			Name:     "SDD Router",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Route a request through combined spec, plan, tasks approval.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: "/test/home/.claude/skills/router"},
				{Client: "codex", Status: skillsync.InventoryStatusMissing, Path: "/test/home/.codex/skills/router"},
			},
		},
		{
			ID:       "agentic-sdd-plan",
			Name:     "SDD Plan",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Draft a feature plan from the draft spec.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMissing, Path: "/test/home/.claude/skills/plan"},
			},
		},
	}
	return cat
}

func makeT35ModifiedCatalog() Catalog {
	cat := makeT35CatalogBase()
	cat.Items = []CatalogItem{
		{
			ID:       "agentic-sdd-router",
			Name:     "SDD Router",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Route a request through combined spec, plan, tasks approval.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusDiffersFromSource, Path: "/test/home/.claude/skills/router"},
			},
		},
	}
	return cat
}

func makeT35CurrentCatalog() Catalog {
	cat := makeT35CatalogBase()
	cat.Items = []CatalogItem{
		{
			ID:       "agentic-sdd-router",
			Name:     "SDD Router",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Route a request through combined spec, plan, tasks approval.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: "/test/home/.claude/skills/router"},
				{Client: "codex", Status: skillsync.InventoryStatusMatchesSource, Path: "/test/home/.codex/skills/router"},
			},
		},
	}
	return cat
}

func makeT35RefusedCatalog() Catalog {
	cat := makeT35CatalogBase()
	cat.Items = []CatalogItem{
		{
			ID:       "agentic-sdd-router",
			Name:     "SDD Router",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Route a request through combined spec, plan, tasks approval.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusUnavailable, Path: "/root/.claude/skills/router"},
			},
		},
	}
	return cat
}

func makeT35FailureCatalog() Catalog {
	cat := makeT35CatalogBase()
	cat.Notices = []string{
		"Destination check failed: permission denied for /protected/dir",
		"Notice: local environment cannot be verified automatically.",
	}
	cat.Items = []CatalogItem{
		{
			ID:       "agentic-sdd-router",
			Name:     "SDD Router",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "Route a request through combined spec, plan, tasks approval.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusUnavailable, Path: "/protected/dir/router"},
			},
		},
	}
	return cat
}

func makeT35EmptyCatalog() Catalog {
	cat := makeT35CatalogBase()
	cat.Items = nil
	return cat
}

func makeT35UnicodeLongPathCatalog() Catalog {
	cat := makeT35CatalogBase()
	deepNestedPath := "/test/home/" + strings.Repeat("nested_subfolder_level/", 8) + "日本語・résumé・tool/PATH-SENTINEL-END"
	cat.Items = []CatalogItem{
		{
			ID:       "agentic-sdd-unicode-cjk-テスト",
			Name:     "日本語・Skill・テスト",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  "テスト用: Wide characters (CJK) like 漢字, ひらがな, and combining diacritics e\u0301 (résumé).",
			Caveat:   "End-of-detail — REACHABILITY-SENTINEL.",
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: deepNestedPath},
			},
		},
		{
			ID:       "agentic-sdd-super-long-path-item",
			Name:     "Long Path Item",
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "custom",
			Purpose:  "A skill with a very long path designed to verify hard wrapping and page reachability.",
			Clients: []ClientStatus{
				{Client: "codex", Status: skillsync.InventoryStatusMissing, Path: "/var/folders/zz/zyxvpxvq6csfxvn_n0000000000000/T/agentic-sdd-deep-nesting-boundary-check-sample-path-test/skills/long"},
			},
		},
	}
	return cat
}

func makeT35ManyItemsCatalog(count int) Catalog {
	cat := makeT35CatalogBase()
	cat.Items = make([]CatalogItem, count)
	for i := 0; i < count; i++ {
		cat.Items[i] = CatalogItem{
			ID:       fmt.Sprintf("item-%03d", i+1),
			Name:     fmt.Sprintf("Catalog Item %03d", i+1),
			Kind:     skillsync.InventoryKindSkill,
			Origin:   skillsync.InventoryOriginCore,
			Group:    GroupCoreSkills,
			Category: "core-workflow",
			Purpose:  fmt.Sprintf("Purpose for test catalog item %03d.", i+1),
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: fmt.Sprintf("/test/home/.claude/skills/item-%03d", i+1)},
			},
		}
	}
	return cat
}

func assertT35Bounds(t *testing.T, label, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Fatalf("%s: lines = %d, want %d", label, len(lines), height)
	}
	for i, line := range lines {
		w := ansi.StringWidth(line)
		if w > width {
			t.Fatalf("%s: line %d width %d exceeds max %d:\n%q", label, i, w, width, line)
		}
	}
}

// 1. Layout breakpoints 40/60/79/80/109/110, tier edges 100/130, heights 8/12/16/24,
// and review samples 140x40, 100x30, 80x24, 60x20.
func TestT35BreakpointsAndReviewGeometries(t *testing.T) {
	cat := makeT35PartialCatalog()

	type geom struct {
		w, h int
		name string
	}

	widths := []int{40, 60, 79, 80, 100, 109, 110, 130}
	heights := []int{8, 12, 16, 24}

	var geoms []geom
	for _, w := range widths {
		for _, h := range heights {
			geoms = append(geoms, geom{w, h, fmt.Sprintf("geom-%dx%d", w, h)})
		}
	}

	// Add review samples explicitly
	reviewSamples := []geom{
		{140, 40, "review-140x40"},
		{100, 30, "review-100x30"},
		{80, 24, "review-80x24"},
		{60, 20, "review-60x20"},
	}
	for _, rs := range reviewSamples {
		alreadyPresent := false
		for _, g := range geoms {
			if g.w == rs.w && g.h == rs.h {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			geoms = append(geoms, rs)
		}
	}

	routes := []struct {
		key  rune
		name string
	}{
		{'1', "Home"},
		{'2', "Install"},
		{'3', "Explore"},
		{'4', "Activity"},
		{'5', "Guide"},
	}

	for _, g := range geoms {
		for _, color := range []bool{true, false} {
			for _, ascii := range []bool{false, true} {
				cfg := TUIConfig{
					Width:  g.w,
					Height: g.h,
					Color:  color,
					ASCII:  ascii,
					Home:   "/test/home",
					Repo:   "/test/repo",
				}
				app := NewApp(cfg, cat)

				for _, r := range routes {
					next := t19Press(t, app, r.key)
					view := next.View().Content
					label := fmt.Sprintf("%s-%s-color%t-ascii%t", g.name, r.name, color, ascii)
					assertT35Bounds(t, label, view, g.w, g.h)

					// Verify footer is non-empty and has quit indication
					lines := strings.Split(view, "\n")
					footer := ansi.Strip(lines[len(lines)-1])
					if !strings.Contains(footer, "q") && !strings.Contains(strings.ToLower(footer), "quit") {
						t.Fatalf("%s: footer missing quit hint: %q", label, footer)
					}
				}
			}
		}
	}
}

// 2. State coverage: fresh, partial, modified, current, offline, refused, failure, empty
func TestT35StateCoverage(t *testing.T) {
	testSizes := []struct {
		w, h int
		name string
	}{
		{140, 40, "desktop-large"},
		{100, 30, "desktop-medium"},
		{80, 24, "standard-terminal"},
		{60, 20, "split-pane"},
		{40, 8, "tiny-terminal"},
	}

	states := []struct {
		name    string
		catalog Catalog
		offline bool
		verify  func(t *testing.T, app App, view string)
	}{
		{
			name:    "fresh",
			catalog: makeT35FreshCatalog(),
			verify: func(t *testing.T, app App, view string) {
				// B9 leads with setup; inventory remains in the full Home content.
				_ = ansi.Strip(view)
				content := strings.Join(app.getContentLines(100), "\n")
				if !strings.Contains(content, "Nothing installed") && app.activeRoute == RouteHome {
					t.Errorf("fresh home view missing first-use setup state")
				}
				if !strings.Contains(content, "Local inventory") && app.activeRoute == RouteHome {
					t.Errorf("fresh home content missing Local inventory heading")
				}
			},
		},
		{
			name:    "partial",
			catalog: makeT35PartialCatalog(),
			verify: func(t *testing.T, app App, view string) {
				_ = ansi.Strip(view)
				content := strings.Join(app.getContentLines(100), "\n")
				if !strings.Contains(content, "Partly installed") && app.activeRoute == RouteHome {
					t.Errorf("partial home view missing partial setup state")
				}
				if !strings.Contains(content, "Local inventory") && app.activeRoute == RouteHome {
					t.Errorf("partial home content missing Local inventory heading")
				}
			},
		},
		{
			name:    "modified",
			catalog: makeT35ModifiedCatalog(),
			verify: func(t *testing.T, app App, view string) {
				stripped := ansi.Strip(view)
				if !strings.Contains(stripped, "modified") && !strings.Contains(stripped, "differs") && app.activeRoute == RouteHome {
					// Verify modified state is mentioned in home client status
					lines := app.getContentLines(100)
					combined := strings.Join(lines, "\n")
					if !strings.Contains(combined, "modified") {
						t.Errorf("modified state not reflected in Home lines: %s", combined)
					}
				}
			},
		},
		{
			name:    "current",
			catalog: makeT35CurrentCatalog(),
			verify: func(t *testing.T, app App, view string) {
				lines := app.getContentLines(100)
				combined := strings.Join(lines, "\n")
				if !strings.Contains(combined, "matching") {
					t.Errorf("current state not reflected in Home lines: %s", combined)
				}
			},
		},
		{
			name:    "offline",
			catalog: makeT35CurrentCatalog(),
			offline: true,
			verify: func(t *testing.T, app App, view string) {
				prefs := app.SessionPreferences()
				if !strings.Contains(prefs, "upd:disabled") {
					t.Errorf("offline state SessionPreferences missing upd:disabled: %q", prefs)
				}
			},
		},
		{
			name:    "refused",
			catalog: makeT35RefusedCatalog(),
			verify: func(t *testing.T, app App, view string) {
				lines := app.getContentLines(100)
				combined := strings.Join(lines, "\n")
				if !strings.Contains(combined, "refused") {
					t.Errorf("refused state not reflected in Home lines: %s", combined)
				}
			},
		},
		{
			name:    "failure",
			catalog: makeT35FailureCatalog(),
			verify: func(t *testing.T, app App, view string) {
				// Notices must be present
				lines := app.getContentLines(100)
				combined := strings.Join(lines, "\n")
				if !strings.Contains(combined, "permission denied") {
					t.Errorf("failure notice not shown in Home lines: %s", combined)
				}
			},
		},
		{
			name:    "empty",
			catalog: makeT35EmptyCatalog(),
			verify: func(t *testing.T, app App, view string) {
				lines := app.getContentLines(100)
				combined := strings.Join(lines, "\n")
				if !strings.Contains(combined, "No managed skills or profiles") {
					t.Errorf("empty catalog recovery explanation missing: %s", combined)
				}
			},
		},
	}

	for _, s := range states {
		for _, sz := range testSizes {
			cfg := TUIConfig{
				Width:  sz.w,
				Height: sz.h,
				Color:  true,
				Home:   "/test/home",
				Repo:   "/test/repo",
			}
			app := NewApp(cfg, s.catalog)
			if s.offline && app.updateMgr != nil {
				app.updateMgr.Disable()
			}

			// Check Home route
			view := app.View().Content
			label := fmt.Sprintf("%s-%s", s.name, sz.name)
			assertT35Bounds(t, label, view, sz.w, sz.h)
			s.verify(t, app, view)

			// Check Explore route for every state
			appExplore := t19Press(t, app, '3')
			viewExplore := appExplore.View().Content
			assertT35Bounds(t, label+"-explore", viewExplore, sz.w, sz.h)

			// Check Activity route for every state
			appAct := t19Press(t, app, '4')
			viewAct := appAct.View().Content
			assertT35Bounds(t, label+"-activity", viewAct, sz.w, sz.h)
		}
	}
}

// 3. Unicode and long paths
func TestT35UnicodeAndLongPaths(t *testing.T) {
	cat := makeT35UnicodeLongPathCatalog()

	sizes := []struct {
		w, h int
	}{
		{140, 40},
		{100, 30},
		{80, 24},
		{60, 20},
		{40, 12},
	}

	for _, sz := range sizes {
		for _, ascii := range []bool{false, true} {
			cfg := TUIConfig{
				Width:  sz.w,
				Height: sz.h,
				Color:  true,
				ASCII:  ascii,
				Home:   "/test/home",
				Repo:   "/test/repo",
			}
			app := NewApp(cfg, cat)

			// 1. Check Explore list view
			app = t19Press(t, app, '3')
			view := app.View().Content
			label := fmt.Sprintf("unicode-w%d-h%d-ascii%t", sz.w, sz.h, ascii)
			assertT35Bounds(t, label+"-list", view, sz.w, sz.h)

			// 2. Open detail view
			app = t19PressCode(t, app, tea.KeyEnter)
			viewDetail := app.View().Content
			assertT35Bounds(t, label+"-detail", viewDetail, sz.w, sz.h)

			// 3. Page down through the detail card and collect every visible
			// row. Reachability is asserted with content sentinels, not only
			// geometry: the caveat marker and the escaped/raw CJK data must
			// actually be rendered at some point.
			var seen strings.Builder
			seen.WriteString(ansi.Strip(viewDetail))
			for p := 0; p < 12; p++ {
				app = t19PressCode(t, app, tea.KeyPgDown)
				paged := app.View().Content
				assertT35Bounds(t, fmt.Sprintf("%s-detail-p%d", label, p), paged, sz.w, sz.h)
				seen.WriteString("\n")
				seen.WriteString(ansi.Strip(paged))
			}
			reached := seen.String()
			if !strings.Contains(reached, "REACHABILITY-SENTINEL") {
				t.Fatalf("%s: paging never reached the content sentinel:\n%s", label, reached)
			}
			if !strings.Contains(reached, "Advanced files:") {
				t.Fatalf("%s: paging never reached the Advanced files section:\n%s", label, reached)
			}
			if ascii {
				for _, want := range []string{`\u65e5`, `\u2014`} {
					if !strings.Contains(reached, want) {
						t.Fatalf("%s: ASCII paging lost the reversible escape %q:\n%s", label, want, reached)
					}
				}
				if strings.ContainsRune(reached, '日') || strings.ContainsRune(reached, '—') {
					t.Fatalf("%s: ASCII paging rendered raw Unicode data:\n%s", label, reached)
				}
			} else {
				if !strings.Contains(reached, "日本語") {
					t.Fatalf("%s: Unicode paging lost the raw CJK identity:\n%s", label, reached)
				}
				if !strings.Contains(reached, "—") {
					t.Fatalf("%s: Unicode paging lost the raw em dash caveat:\n%s", label, reached)
				}
			}
		}
	}
}

// 4. Footer, last-item reachability and focus cycling
func TestT35FooterLastItemAndFocus(t *testing.T) {
	// Catalog with 30 items
	cat := makeT35ManyItemsCatalog(30)
	cfg := TUIConfig{Width: 80, Height: 24, Color: true}
	app := NewApp(cfg, cat)

	// Route Explore
	app = t19Press(t, app, '3')

	// Focus cycling with Tab
	if app.focus != FocusContent {
		t.Fatalf("initial explore focus = %v, want FocusContent", app.focus)
	}
	app = t19PressCode(t, app, tea.KeyTab)
	if app.focus != FocusNav {
		t.Fatalf("after Tab focus = %v, want FocusNav", app.focus)
	}
	app = t19PressCode(t, app, tea.KeyTab)
	if app.focus != FocusCategories {
		t.Fatalf("after Tab focus = %v, want FocusCategories", app.focus)
	}
	app = t19PressCode(t, app, tea.KeyTab)
	if app.focus != FocusContent {
		t.Fatalf("after Tab cycle focus = %v, want FocusContent", app.focus)
	}

	// Navigate to the last item in the list
	for i := 0; i < 35; i++ {
		app = t19Press(t, app, 'j')
	}

	selected, ok := app.catalogModel.Selected()
	if !ok {
		t.Fatal("no item selected after navigating down")
	}
	if selected.ID != "item-030" {
		t.Fatalf("selected item = %q, want item-030 (last item)", selected.ID)
	}

	// Verify bounds and footer integrity with last item selected
	view := app.View().Content
	assertT35Bounds(t, "explore-last-item-selected", view, 80, 24)
	lines := strings.Split(view, "\n")
	footer := ansi.Strip(lines[len(lines)-1])
	if !strings.Contains(footer, "q") {
		t.Fatalf("footer missing quit hint on last item: %q", footer)
	}
}

// 5. Confirmation fit: install and restore confirm modals at all breakpoints
func TestT35ConfirmationFitAndFallback(t *testing.T) {
	sizes := []struct {
		w, h int
		name string
	}{
		// Centered candidates
		{140, 40, "centered-140x40"},
		{100, 30, "centered-100x30"},
		{80, 24, "centered-80x24"},
		{60, 20, "centered-60x20"},
		// Breakpoints
		{130, 24, "tier-130x24"},
		{130, 16, "tier-130x16"},
		{110, 24, "bp-110x24"},
		{110, 16, "bp-110x16"},
		{110, 12, "bp-110x12"},
		{109, 24, "bp-109x24"},
		{109, 16, "bp-109x16"},
		{100, 24, "tier-100x24"},
		{80, 16, "bp-80x16"},
		{79, 24, "bp-79x24"},
		{79, 16, "bp-79x16"},
		{60, 16, "bp-60x16"},
		// Fullscreen fallbacks (short or narrow)
		{40, 24, "fallback-40x24"},
		{60, 12, "fallback-60x12"},
		{80, 12, "fallback-80x12"},
		{100, 8, "fallback-100x8"},
		{40, 8, "fallback-40x8"},
	}

	samplePath := "/test/home/.claude/skills/sample-skill"

	for _, sz := range sizes {
		for _, ascii := range []bool{false, true} {
			cfg := TUIConfig{
				Width:  sz.w,
				Height: sz.h,
				Color:  true,
				ASCII:  ascii,
				Home:   t.TempDir(),
			}

			// Test Install Confirmation Modal
			appInstall := t19OpenInstallConfirm(t, cfg, samplePath)
			viewInstall := appInstall.View().Content
			labelInstall := fmt.Sprintf("install-confirm-%s-ascii%t", sz.name, ascii)
			assertT35Bounds(t, labelInstall, viewInstall, sz.w, sz.h)

			// Paged body scrolling in confirmation modal
			appScrolled, _ := t19Update(t, appInstall, t19Code(tea.KeyPgDown))
			assertT35Bounds(t, labelInstall+"-scrolled", appScrolled.View().Content, sz.w, sz.h)

			// Verify safe confirmation key hints are present
			footerInstall := ansi.Strip(strings.Split(viewInstall, "\n")[sz.h-1])
			if !strings.Contains(footerInstall, "y confirm") {
				t.Fatalf("%s: footer missing 'y confirm': %q", labelInstall, footerInstall)
			}

			// Test Restore Confirmation Modal
			appRestore := t19OpenRestoreConfirm(t, cfg, samplePath)
			viewRestore := appRestore.View().Content
			labelRestore := fmt.Sprintf("restore-confirm-%s-ascii%t", sz.name, ascii)
			assertT35Bounds(t, labelRestore, viewRestore, sz.w, sz.h)

			footerRestore := ansi.Strip(strings.Split(viewRestore, "\n")[sz.h-1])
			if !strings.Contains(footerRestore, "y restore") && !strings.Contains(footerRestore, "y confirm") {
				t.Fatalf("%s: footer missing restore confirmation: %q", labelRestore, footerRestore)
			}
		}
	}
}

// TestT35ConfirmationResizeKeepsSafeNo resizes an already-open install and
// restore confirmation through every approved geometry. The active
// confirmation must stay default-No, keep its token, and never emit a write
// from Enter while resizing; n still cancels safely afterwards. Below the
// usable minimum, TestConfirmationBelowUsableGeometryCannotSubmit covers the
// stricter no-input case.
func TestT35ConfirmationResizeKeepsSafeNo(t *testing.T) {
	sizes := []struct {
		w, h int
	}{
		{140, 40},
		{100, 30},
		{80, 24},
		{60, 20},
		{40, 8},
	}
	for _, ascii := range []bool{false, true} {
		t.Run(fmt.Sprintf("ascii%t", ascii), func(t *testing.T) {
			home := t.TempDir()
			cfg := TUIConfig{Width: 140, Height: 40, Color: true, ASCII: ascii, Home: home}
			path := home + "/.claude/skills/sample"

			install := t19OpenInstallConfirm(t, cfg, path)
			installToken := install.install.PreviewToken()
			for _, sz := range sizes {
				install, _ = t19Update(t, install, tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
				label := fmt.Sprintf("install-%dx%d-ascii%t", sz.w, sz.h, ascii)
				assertT35Bounds(t, label, install.View().Content, sz.w, sz.h)
				if install.install.Step() != InstallStepConfirm || install.install.PreviewToken() != installToken {
					t.Fatalf("%s: resize changed confirmation state/token", label)
				}
				next, cmd := t19Update(t, install, t19Code(tea.KeyEnter))
				if submitted(cmd) {
					t.Fatalf("%s: Enter submitted a write after resize", label)
				}
				install = next
			}
			next, cmd := t19Update(t, install, t19Key('n'))
			if submitted(cmd) {
				t.Fatal("install cancellation submitted a write after resize")
			}
			install = next
			if install.install.Step() != InstallStepPreview {
				t.Fatalf("install stage after n = %v, want preview", install.install.Step())
			}

			restore := t19OpenRestoreConfirm(t, cfg, path)
			restoreToken := restore.restore.preview.PreviewToken
			for _, sz := range sizes {
				restore, _ = t19Update(t, restore, tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
				label := fmt.Sprintf("restore-%dx%d-ascii%t", sz.w, sz.h, ascii)
				assertT35Bounds(t, label, restore.View().Content, sz.w, sz.h)
				if restore.restore.Stage() != RestoreStageConfirm || restore.restore.preview.PreviewToken != restoreToken {
					t.Fatalf("%s: resize changed restore confirmation state/token", label)
				}
				next, cmd := t19Update(t, restore, t19Code(tea.KeyEnter))
				if submitted(cmd) {
					t.Fatalf("%s: Enter submitted a restore after resize", label)
				}
				restore = next
			}
			next, cmd = t19Update(t, restore, t19Key('n'))
			if submitted(cmd) {
				t.Fatal("restore cancellation submitted a write after resize")
			}
			restore = next
			if restore.restore.Stage() != RestoreStagePreview {
				t.Fatalf("restore stage after n = %v, want preview", restore.restore.Stage())
			}
		})
	}
}

// submitted reports whether a command emitted a write request.
func submitted(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if msg == nil {
		return false
	}
	_, ok := msg.(StartWriteMsg)
	return ok
}

// 7. Representative deterministic snapshots for visual review inspection
func TestT35RepresentativeScreenSnapshots(t *testing.T) {
	cat := makeT35PartialCatalog()

	// Review sample: 80x24 plain ASCII Home
	cfg80x24 := TUIConfig{Width: 80, Height: 24, Color: false, ASCII: true, Home: "/test/home", Repo: "/test/repo"}
	app80 := NewApp(cfg80x24, cat)
	viewHome80 := ansi.Strip(app80.View().Content)

	// Ensure structural elements of Home are present and deterministic
	expectedHomeSnippets := []string{
		"AGENTIC SDD / Home [nav] by nawodyaishan",
		"Install shared Agentic SDD skills",
		"Every change is previewed first",
		"Setup",
		"Partly installed",
		"Recommended next step",
		"j/k select - Enter open - Tab pane - ? help - q quit",
	}
	for _, snippet := range expectedHomeSnippets {
		if !strings.Contains(viewHome80, snippet) {
			t.Errorf("80x24 Home view missing expected snippet %q:\n%s", snippet, viewHome80)
		}
	}

	// Switch focus to content to test content-focused footer and scrolling
	app80Content := t19PressCode(t, app80, tea.KeyTab)
	viewHome80Content := ansi.Strip(app80Content.View().Content)
	if !strings.Contains(viewHome80Content, "AGENTIC SDD / Home [content] by nawodyaishan") {
		t.Errorf("80x24 Home content focus missing expected header:\n%s", viewHome80Content)
	}
	if !strings.Contains(viewHome80Content, "Enter setup - PgDn scroll - ? help - q quit") {
		t.Errorf("80x24 Home content focus missing expected footer:\n%s", viewHome80Content)
	}

	// Page through Home to prove the secondary inventory and client evidence
	// remain reachable after B9 moved them below the first screen.
	viewHome80Paged := t35PageDownText(t, app80Content, 12)
	for _, snippet := range []string{
		"Local inventory",
		"Skills: 2",
		"Client status (core skills)",
		"claude: present 1/2 (1 matching + 0 modified) - 1 not present - 0 refused",
		"codex: present 0/1 (0 matching + 0 modified) - 1 not present - 0 refused",
	} {
		if !strings.Contains(viewHome80Paged, snippet) {
			t.Errorf("80x24 Home paged views never reached %q:\n%s", snippet, viewHome80Paged)
		}
	}

	// Review sample: 60x20 plain ASCII Home
	cfg60x20 := TUIConfig{Width: 60, Height: 20, Color: false, ASCII: true, Home: "/test/home", Repo: "/test/repo"}
	app60 := NewApp(cfg60x20, cat)
	viewHome60 := ansi.Strip(app60.View().Content)
	expected60Snippets := []string{
		"AGENTIC SDD / Home [nav]",
		"Install shared Agentic SDD skills",
		"Every change is previewed first",
		"Setup",
		"Partly installed",
		"j/k select - Enter open - Tab pane - ? help - q quit",
	}
	for _, snippet := range expected60Snippets {
		if !strings.Contains(viewHome60, snippet) {
			t.Errorf("60x20 Home view missing expected snippet %q:\n%s", snippet, viewHome60)
		}
	}
	// B9 makes setup the first screen; inventory remains reachable by scrolling.
	app60Content := t19PressCode(t, app60, tea.KeyTab)
	viewHome60Paged := t35PageDownText(t, app60Content, 18)
	if !strings.Contains(viewHome60Paged, "Local inventory") || !strings.Contains(viewHome60Paged, "Skills: 2") {
		t.Errorf("60x20 Home paged views never reached the secondary inventory:\n%s", viewHome60Paged)
	}

	// Review sample: 80x24 Explore (catalog)
	appExplore := t19Press(t, app80, '3')
	viewExplore := ansi.Strip(appExplore.View().Content)
	expectedExploreSnippets := []string{
		"AGENTIC SDD / Explore [content] by nawodyaishan",
		"SDD Router",
		"SDD Plan",
		"/ filter - c category - Enter details - q quit",
	}
	for _, snippet := range expectedExploreSnippets {
		if !strings.Contains(viewExplore, snippet) {
			t.Errorf("80x24 Explore view missing expected snippet %q:\n%s", snippet, viewExplore)
		}
	}

	// Review sample: 80x24 Install confirm modal
	appConfirm := t19OpenInstallConfirm(t, cfg80x24, "/test/home/.claude/skills/router")
	viewConfirm := ansi.Strip(appConfirm.View().Content)
	expectedConfirmSnippets := []string{
		"Confirm Installation",
		"Pre-Execution Policy",
		"Proceed with installation?",
		"[ No ] Yes",
		"y confirm - n/Esc cancel - j/k scroll - ? help",
	}
	for _, snippet := range expectedConfirmSnippets {
		if !strings.Contains(viewConfirm, snippet) {
			t.Errorf("80x24 Confirm modal missing expected snippet %q:\n%s", snippet, viewConfirm)
		}
	}
}

// t35PageDownText collects every visible page while paging down. Reachability
// is the contract; a single page must not be expected to hold long Home content.
func t35PageDownText(t *testing.T, app App, pages int) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(ansi.Strip(app.View().Content))
	for i := 0; i < pages; i++ {
		app = t19PressCode(t, app, tea.KeyPgDown)
		b.WriteString("\n")
		b.WriteString(ansi.Strip(app.View().Content))
	}
	return b.String()
}

// 8. Tiny terminal safety (<40x8)
func TestT35TinyTerminalSafety(t *testing.T) {
	cat := makeT35PartialCatalog()
	tinySizes := []struct {
		w, h int
	}{
		{39, 24},
		{80, 7},
		{35, 6},
		{20, 5},
	}

	for _, sz := range tinySizes {
		cfg := TUIConfig{Width: sz.w, Height: sz.h, Color: true}
		app := NewApp(cfg, cat)

		view := app.View().Content
		label := fmt.Sprintf("tiny-%dx%d", sz.w, sz.h)
		assertT35Bounds(t, label, view, sz.w, sz.h)

		lines := strings.Split(view, "\n")
		footer := ansi.Strip(lines[len(lines)-1])
		if !strings.Contains(footer, "q") {
			t.Fatalf("%s: footer missing quit hint: %q", label, footer)
		}
		if !strings.Contains(view, "Terminal too small") {
			t.Fatalf("%s: view missing resize message: %q", label, view)
		}
	}
}
