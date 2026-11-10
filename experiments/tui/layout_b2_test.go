package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestB2ReadableProseWordBoundaries(t *testing.T) {
	app := NewApp(TUIConfig{Width: 80, Height: 24}, Catalog{})
	got := app.viewportLines("A readable description with useful guidance.", 22)
	want := []string{"A readable description", "with useful guidance."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prose should wrap between words: got %q want %q", got, want)
	}
}

func TestB2CategoryFocusAndVisibleControl(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		{ID: "agentic-sdd-router", Kind: "skill", Origin: "core"},
		{ID: "agentic-sdd-orchestrate", Kind: "skill", Origin: "pack"},
	}}
	app := NewApp(TUIConfig{Width: 100, Height: 24}, cat)
	press := func(k tea.KeyPressMsg) {
		next, _ := app.Update(k)
		app = next.(App)
	}
	press(tea.KeyPressMsg{Code: '3', Text: "3"})
	if !strings.Contains(app.View().Content, "Category:") {
		t.Fatal("Explore does not expose a category control")
	}
	press(tea.KeyPressMsg{Code: tea.KeyTab}) // content -> navigation
	press(tea.KeyPressMsg{Code: tea.KeyTab}) // navigation -> categories
	if app.focus != FocusCategories {
		t.Fatalf("Tab must reach categories, got focus %v", app.focus)
	}
	press(tea.KeyPressMsg{Code: tea.KeyDown})
	if app.catalogModel.ActiveCategory() == "all" {
		t.Fatal("category selection did not change")
	}
	press(tea.KeyPressMsg{Code: tea.KeyEnter})
	if app.focus != FocusContent {
		t.Fatal("Enter on category control should focus the catalog")
	}
}

func TestB2ContentGuttersSeparatePanels(t *testing.T) {
	for _, width := range []int{60, 80, 110} {
		app := NewApp(TUIConfig{Width: width, Height: 24}, Catalog{})
		view := app.View().Content
		for _, line := range strings.Split(view, "\n") {
			if i := strings.Index(line, "|"); i >= 0 && i+1 < len(line) && strings.TrimSpace(line[i+1:]) != "" {
				if line[i+1] != ' ' {
					t.Fatalf("width %d: panel divider touches text: %q", width, line)
				}
			}
		}
	}
}
