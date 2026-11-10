package tui_test

import (
	"fmt"
	"strings"
	"testing"

	"agentic-sdd/experiments/tui"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestIntegratedASCIIUnicodePathBounds(t *testing.T) {
	cat := testCatalog()
	cat.Items[0].Clients[0].Path = "/" + strings.Repeat("résumé🚀/", 20) + "RECOVERY-END"
	for _, width := range []int{40, 60, 80, 110} {
		app := tui.NewApp(tui.TUIConfig{Width: width, Height: 12, ASCII: true}, cat)
		next, _ := app.Update(runes("3"))
		app = next.(tui.App)
		next, _ = app.Update(key(tea.KeyEnter))
		app = next.(tui.App)
		var observed strings.Builder
		for page := 0; page < 100; page++ {
			view := app.View().Content
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > width {
					t.Fatalf("width %d page %d overflow %d: %q", width, page, ansi.StringWidth(line), line)
				}
			}
			observed.WriteString(view)
			next, _ = app.Update(key(tea.KeyPgDown))
			app = next.(tui.App)
		}
		if !strings.Contains(observed.String(), "END") {
			t.Fatalf("width %d: path end is unreachable by scrolling", width)
		}
	}
}

func TestIntegratedFooterAlwaysOffersQuit(t *testing.T) {
	for _, width := range []int{40, 60, 80, 110} {
		app := tui.NewApp(tui.TUIConfig{Width: width, Height: 8}, testCatalog())
		lines := strings.Split(app.View().Content, "\n")
		if !strings.Contains(lines[len(lines)-1], "q quit") {
			t.Fatalf("width %d: quit missing from footer %q", width, lines[len(lines)-1])
		}
	}
}

func TestIntegratedContextualFooterAndScrollPersistence(t *testing.T) {
	app := tui.NewApp(tui.TUIConfig{Width: 80, Height: 12}, testCatalog())
	next, _ := app.Update(runes("4"))
	app = next.(tui.App)
	for i := 0; i < 3; i++ {
		next, _ = app.Update(key(tea.KeyPgDown))
		app = next.(tui.App)
	}
	before := app.View().Content
	next, _ = app.Update(runes("1"))
	app = next.(tui.App)
	next, _ = app.Update(runes("4"))
	app = next.(tui.App)
	if app.View().Content != before {
		t.Fatal("route switch lost the Activity scroll position")
	}
	next, _ = app.Update(runes("3"))
	app = next.(tui.App)
	lines := strings.Split(app.View().Content, "\n")
	if !strings.Contains(lines[len(lines)-1], "/ filter") {
		t.Fatalf("catalog footer lacks search action: %q", lines[len(lines)-1])
	}
}

func TestIntegratedInstallCommandHintsUseActualFlags(t *testing.T) {
	app := tui.NewApp(tui.TUIConfig{Width: 160, Height: 80, Home: "/tmp/a b", Repo: "/tmp/r's"}, testCatalog())
	next, _ := app.Update(runes("2"))
	view := next.(tui.App).View().Content
	if strings.Contains(view, "--backup") {
		t.Fatal("restore hint invents an unsupported --backup flag")
	}
	if !strings.Contains(view, "--home '/tmp/a b'") || !strings.Contains(view, "--repo '/tmp/r'\"'\"'s'") {
		t.Fatalf("custom source/home must remain shell-quoted in hints:\n%s", view)
	}
}

func TestIntegratedScrollUpImmediatelyAfterEnd(t *testing.T) {
	app := tui.NewApp(tui.TUIConfig{Width: 80, Height: 12}, testCatalog())
	next, _ := app.Update(runes("4"))
	app = next.(tui.App)
	for i := 0; i < 100; i++ {
		next, _ = app.Update(key(tea.KeyPgDown))
		app = next.(tui.App)
	}
	atEnd := app.View().Content
	next, _ = app.Update(key(tea.KeyPgUp))
	if next.(tui.App).View().Content == atEnd {
		t.Fatal("overscrolling created hidden offset: PageUp did not move from the bottom")
	}
}

func TestIntegratedLastCatalogSelectionIsVisible(t *testing.T) {
	cat := testCatalog()
	cat.Items = nil
	for i := 0; i < 30; i++ {
		cat.Items = append(cat.Items, tui.CatalogItem{ID: fmt.Sprintf("item-%02d", i), Kind: "skill"})
	}
	for _, width := range []int{60, 80, 110} {
		app := tui.NewApp(tui.TUIConfig{Width: width, Height: 12}, cat)
		next, _ := app.Update(runes("3"))
		app = next.(tui.App)
		next, _ = app.Update(key(tea.KeyEnd))
		app = next.(tui.App)
		if !strings.Contains(app.View().Content, "item-29") {
			t.Fatalf("width %d: selected last entry is hidden:\n%s", width, app.View().Content)
		}
	}
}

func TestIntegratedHomePreservesInventoryNotice(t *testing.T) {
	cat := testCatalog()
	cat.Notices = []string{"Optional pack unavailable; showing core skills only"}
	app := tui.NewApp(tui.TUIConfig{Width: 100, Height: 40}, cat)
	observed := app.View().Content
	for page := 0; page < 50 && !strings.Contains(observed, cat.Notices[0]); page++ {
		next, _ := app.Update(key(tea.KeyPgDown))
		app = next.(tui.App)
		observed += app.View().Content
	}
	if !strings.Contains(observed, cat.Notices[0]) {
		t.Fatal("Home discarded an inventory warning")
	}
}
