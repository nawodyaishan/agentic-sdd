package tui_test

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/experiments/tui/screens"
	"agentic-sdd/internal/skillsync"
	tea "charm.land/bubbletea/v2"
)

func TestLaunchAllowed(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		stdoutTTY bool
		enabled   bool
		reason    string
	}{
		{
			name:      "opt-in missing",
			env:       map[string]string{},
			stdoutTTY: true,
			reason:    "is not set",
		},
		{
			name:      "opt-in value other than 1",
			env:       map[string]string{tui.EnvTUI: "true"},
			stdoutTTY: true,
			reason:    "is not set",
		},
		{
			name:      "stdout is not a terminal",
			env:       map[string]string{tui.EnvTUI: "1"},
			stdoutTTY: false,
			reason:    "not a terminal",
		},
		{
			name:      "scripted forced interactive run",
			env:       map[string]string{tui.EnvTUI: "1", tui.EnvForceInteractive: "1"},
			stdoutTTY: true,
			reason:    tui.EnvForceInteractive,
		},
		{
			name:      "explicit opt-in on a terminal",
			env:       map[string]string{tui.EnvTUI: "1"},
			stdoutTTY: true,
			enabled:   true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tui.LaunchAllowed(func(key string) string { return tc.env[key] }, tc.stdoutTTY)
			if got.Enabled != tc.enabled {
				t.Fatalf("Enabled = %v, want %v (reason %q)", got.Enabled, tc.enabled, got.Reason)
			}
			if tc.enabled {
				if got.Reason != "" {
					t.Errorf("Reason = %q, want empty for an enabled decision", got.Reason)
				}
				return
			}
			if !strings.Contains(got.Reason, tc.reason) {
				t.Errorf("Reason = %q, want it to contain %q", got.Reason, tc.reason)
			}
		})
	}
}

func TestTUIConfigNormalize(t *testing.T) {
	got := tui.TUIConfig{Width: -4, Height: 0}.Normalize()
	if got.Width != tui.DefaultWidth || got.Height != tui.DefaultHeight {
		t.Fatalf("Normalize() = %dx%d, want %dx%d", got.Width, got.Height, tui.DefaultWidth, tui.DefaultHeight)
	}
	if kept := (tui.TUIConfig{Width: 40, Height: 12}).Normalize(); kept.Width != 40 || kept.Height != 12 {
		t.Fatalf("Normalize() changed explicit dimensions to %dx%d", kept.Width, kept.Height)
	}
}

// TestLoadCatalogReadOnly proves the adapter reads the embedded inventory and
// leaves the inspected home untouched.
func TestLoadCatalogReadOnly(t *testing.T) {
	home := t.TempDir()
	before := snapshotTree(t, home)

	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	if cat.SkillCount() != 19 {
		t.Errorf("SkillCount = %d, want 19 (18 core skills plus the pack skill)", cat.SkillCount())
	}
	if cat.ProfileCount() != 18 {
		t.Errorf("ProfileCount = %d, want 18 (14 pack profiles plus 4 presets)", cat.ProfileCount())
	}
	if len(cat.Items) != 37 {
		t.Errorf("len(Items) = %d, want 37", len(cat.Items))
	}
	if !filepath.IsAbs(cat.Home) {
		t.Errorf("Catalog.Home = %q, want an absolute path", cat.Home)
	}
	if len(cat.Notices) != 0 {
		t.Errorf("Notices = %v, want none with the embedded pack", cat.Notices)
	}

	// The embedded source is core-only for skills and pack-only for profiles, so
	// these known IDs must be present with the expected groups.
	byID := make(map[string]tui.CatalogItem, len(cat.Items))
	for _, item := range cat.Items {
		byID[item.Kind+"/"+item.ID] = item
	}
	for _, want := range []struct {
		kind, id, group string
	}{
		{skillsync.InventoryKindSkill, "agentic-sdd-plan", tui.GroupCoreSkills},
		{skillsync.InventoryKindSkill, "agentic-sdd-orchestrate", tui.GroupOpenCodePack},
		{skillsync.InventoryKindProfile, "sdd-lightweight", tui.GroupOpenCodeProfiles},
		{skillsync.InventoryKindProfile, "sdd-implement-1", tui.GroupFreeFirstPresets},
	} {
		item, ok := byID[want.kind+"/"+want.id]
		if !ok {
			t.Errorf("missing %s %q", want.kind, want.id)
			continue
		}
		if item.Group != want.group {
			t.Errorf("%s group = %q, want %q", want.id, item.Group, want.group)
		}
		if len(item.Clients) == 0 {
			t.Errorf("%s has no client statuses", want.id)
		}
		for _, client := range item.Clients {
			if client.Path == "" {
				t.Errorf("%s/%s has an empty destination path", want.id, client.Client)
			}
		}
	}

	// A fresh home has no installed skills, so every core destination is a
	// missing target: the badge must not claim an install.
	if plan := byID[skillsync.InventoryKindSkill+"/agentic-sdd-plan"]; plan.InstalledClients() != 0 {
		t.Errorf("agentic-sdd-plan InstalledClients = %d in a fresh home, want 0", plan.InstalledClients())
	}

	if after := snapshotTree(t, home); !reflect.DeepEqual(before, after) {
		t.Fatalf("LoadCatalog changed the home tree:\nbefore: %v\nafter:  %v", before, after)
	}
}

func TestLoadCatalogCoreOnly(t *testing.T) {
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: t.TempDir()})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if cat.SkillCount() != 18 || cat.ProfileCount() != 0 {
		t.Fatalf("counts = %d skills, %d profiles; want 18 and 0", cat.SkillCount(), cat.ProfileCount())
	}
}

func TestLoadCatalogRequiresHome(t *testing.T) {
	if _, err := tui.LoadCatalog(tui.TUIConfig{Home: "  "}); err == nil {
		t.Fatal("LoadCatalog accepted an empty home")
	}
	if _, err := tui.LoadCatalog(tui.TUIConfig{Home: t.TempDir(), OpenCodeFreeFirst: true}); err == nil {
		t.Fatal("LoadCatalog accepted OpenCodeFreeFirst without WithOpenCode")
	}
}

// TestLoadCatalogMissingPack documents the non-fatal path: core rows survive an
// absent optional pack and the notice explains the reduced catalog.
func TestLoadCatalogMissingPack(t *testing.T) {
	repo := t.TempDir()
	skill := filepath.Join(repo, "skills-files", "agentic-sdd-spike")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# spike\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat, err := tui.LoadCatalog(tui.TUIConfig{Repo: repo, Home: t.TempDir(), WithOpenCode: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if cat.SkillCount() != 1 || cat.ProfileCount() != 0 {
		t.Fatalf("counts = %d skills, %d profiles; want 1 and 0", cat.SkillCount(), cat.ProfileCount())
	}
	if len(cat.Notices) != 1 || !strings.Contains(cat.Notices[0], "pack source not found") {
		t.Fatalf("Notices = %v, want one pack-missing notice", cat.Notices)
	}
}

func TestStatusBadgeAndHints(t *testing.T) {
	for status, want := range map[skillsync.InventoryStatus]string{
		skillsync.InventoryStatusMatchesSource:     "Installed (matches source)",
		skillsync.InventoryStatusDiffersFromSource: "Installed (modified)",
		skillsync.InventoryStatusMissing:           "Template available (not installed)",
		skillsync.InventoryStatusUnavailable:       "Unavailable / refused",
	} {
		if got := tui.StatusBadge(status); got != want {
			t.Errorf("StatusBadge(%q) = %q, want %q", status, got, want)
		}
	}

	for origin, want := range map[string]string{
		skillsync.InventoryOriginCore:   "asdx-core preview",
		skillsync.InventoryOriginPack:   "asdx-core preview --with-opencode",
		skillsync.InventoryOriginPreset: "asdx-core preview --with-opencode --opencode-free-first",
	} {
		if got := (tui.CatalogItem{Origin: origin}).InstallHint(); got != want {
			t.Errorf("InstallHint(%q) = %q, want %q", origin, got, want)
		}
	}
}

func TestSummarizePreview(t *testing.T) {
	summary := tui.SummarizePreview(strings.Join([]string{
		"install claude/agentic-sdd-plan",
		"replace + back up codex/agentic-sdd-plan",
		"up to date agents/agentic-sdd-guide",
		"Preview only. No files were changed.",
	}, "\n"))
	if summary.Install != 1 || summary.Replace != 1 || summary.UpToDate != 1 {
		t.Fatalf("summary = %+v, want 1 install, 1 replace and 1 up-to-date", summary)
	}
	if summary.TotalChanges() != 2 {
		t.Fatalf("TotalChanges = %d, want 2", summary.TotalChanges())
	}
}

func TestCatalogModelNavigation(t *testing.T) {
	model := screens.NewCatalog(tui.TUIConfig{Width: 80, Height: 24}, testCatalog())

	if view := viewContent(t, model); strings.Contains(view, "\x1b") {
		t.Fatalf("a color-free list view must not contain ANSI escapes:\n%q", view)
	}

	model = update(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})
	if w, h := model.Size(); w != 80 || h != 24 {
		t.Fatalf("Size = %dx%d, want 80x24", w, h)
	}

	first, ok := model.Selected()
	if !ok || first.ID != "alpha-skill" {
		t.Fatalf("initial selection = %q (ok=%v), want alpha-skill", first.ID, ok)
	}

	model = update(t, model, runes("j"))
	if got, _ := model.Selected(); got.ID != "beta-skill" {
		t.Fatalf("after j selection = %q, want beta-skill", got.ID)
	}

	model = update(t, model, key(tea.KeyEnter))
	if !model.InDetail() {
		t.Fatal("enter did not open the detail card")
	}
	detail := viewContent(t, model)
	if strings.Contains(detail, "\x1b") {
		t.Fatalf("a color-free view must not contain ANSI escapes:\n%q", detail)
	}
	for _, want := range []string{
		"beta-skill",
		"Destination status",
		"claude",
		"Template available (not installed)",
		"Next action",
		"asdx-core preview",
		"b / esc back",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail view missing %q:\n%s", want, detail)
		}
	}

	model = update(t, model, runes("b"))
	if model.InDetail() {
		t.Fatal("b did not return to the list")
	}

	if _, cmd := model.Update(runes("q")); cmd == nil {
		t.Fatal("q returned no command; want quit")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not produce a quit message")
	}
}

func TestCatalogModelFilter(t *testing.T) {
	model := screens.NewCatalog(tui.TUIConfig{Width: 80, Height: 24}, testCatalog())

	model.SetFilter("beta")
	if visible := model.VisibleItems(); len(visible) != 1 || visible[0].ID != "beta-skill" {
		t.Fatalf("filtered items = %v, want only beta-skill", ids(visible))
	}

	// Filtering by kind must find the profile row it shares no name with.
	model.SetFilter("profile")
	if visible := model.VisibleItems(); len(visible) != 1 || visible[0].ID != "zulu-profile" {
		t.Fatalf("kind filter = %v, want only zulu-profile", ids(visible))
	}

	model.SetFilter("")
	if visible := model.VisibleItems(); len(visible) != 3 {
		t.Fatalf("cleared filter = %v, want all three rows", ids(visible))
	}
}

// TestCatalogModelFilterKeys proves `/` focuses the filter input, so a
// following q is search text rather than a quit command.
func TestCatalogModelFilterKeys(t *testing.T) {
	model := screens.NewCatalog(tui.TUIConfig{Width: 80, Height: 24}, testCatalog())

	// The filter-focus command only starts the cursor blink; this test drives no
	// renderer, so it is not executed.
	next, _ := model.Update(runes("/"))
	model = next.(screens.Model)
	for _, r := range "betaq" {
		next, _ := model.Update(runes(string(r)))
		model = next.(screens.Model)
	}

	if view := viewContent(t, model); !strings.Contains(view, "betaq") {
		t.Fatalf("the filter input did not receive the typed keys:\n%s", view)
	}
	if model.InDetail() {
		t.Fatal("typing in the filter opened the detail card")
	}
}

// TestCatalogProgramHeadless runs the real Bubble Tea program without a
// terminal or renderer. It proves the model quits cleanly on q, which is the
// only behavior a scripted harness can rely on.
func TestCatalogProgramHeadless(t *testing.T) {
	program := tea.NewProgram(
		screens.NewCatalog(tui.TUIConfig{Width: 80, Height: 24}, testCatalog()),
		tea.WithInput(strings.NewReader("q")),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
		tea.WithWindowSize(80, 24),
	)
	if _, err := program.Run(); err != nil {
		t.Fatalf("program.Run: %v", err)
	}
}

func TestCatalogModelEmpty(t *testing.T) {
	model := screens.NewCatalog(tui.TUIConfig{}, tui.Catalog{})
	if _, ok := model.Selected(); ok {
		t.Fatal("an empty catalog reported a selection")
	}
	model = update(t, model, runes("j"))
	model = update(t, model, key(tea.KeyEnter))
	if model.InDetail() {
		t.Fatal("enter opened a detail card with no rows")
	}
	if view := viewContent(t, model); !strings.Contains(view, "0 skills · 0 profiles") {
		t.Fatalf("empty view missing zero counts:\n%s", view)
	}
}

func testCatalog() tui.Catalog {
	return tui.Catalog{
		Items: []tui.CatalogItem{
			{
				ID:     "alpha-skill",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginCore,
				Group:  tui.GroupCoreSkills,
				Clients: []tui.ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMissing, Path: "/tmp/home/.claude/skills/alpha-skill"},
				},
			},
			{
				ID:     "beta-skill",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginPack,
				Group:  tui.GroupOpenCodePack,
				Clients: []tui.ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMissing, Path: "/tmp/home/.claude/skills/beta-skill"},
				},
			},
			{
				ID:     "zulu-profile",
				Kind:   skillsync.InventoryKindProfile,
				Origin: skillsync.InventoryOriginPack,
				Group:  tui.GroupOpenCodeProfiles,
				Clients: []tui.ClientStatus{
					{Client: "opencode", Status: skillsync.InventoryStatusMatchesSource, Path: "/tmp/home/.config/opencode/agent/zulu-profile.md"},
				},
			},
		},
	}
}

// update sends one message and returns the next model. Navigation keys in this
// screen return no command the test needs to run.
func update(t *testing.T, model screens.Model, msg tea.Msg) screens.Model {
	t.Helper()
	next, _ := model.Update(msg)
	updated, ok := next.(screens.Model)
	if !ok {
		t.Fatalf("Update returned %T, want screens.Model", next)
	}
	return updated
}

func viewContent(t *testing.T, model screens.Model) string {
	t.Helper()
	return model.View().Content
}

func runes(s string) tea.KeyPressMsg {
	r := []rune(s)
	msg := tea.KeyPressMsg{}
	if len(r) > 0 {
		msg.Code = r[0]
		msg.Text = s
	}
	return msg
}

func key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

// snapshotTree records every path under root with its size so a test can prove
// a read-only operation created, removed or rewrote nothing.
func snapshotTree(t *testing.T, root string) map[string]int64 {
	t.Helper()
	tree := map[string]int64{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		tree[rel] = info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return tree
}

func ids(items []tui.CatalogItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	sort.Strings(out)
	return out
}
