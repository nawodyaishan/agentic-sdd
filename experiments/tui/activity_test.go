package tui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
)

func TestActivity_ImmediateLocalRender(t *testing.T) {
	model := NewActivityModel(ActivityConfig{
		Theme: NewTheme(false),
		Width: 80,
		Metadata: AboutMetadata{
			Version: "v1.2.3",
			Commit:  "abcdef1",
			Date:    "2026-11-10",
			Author:  BrandAuthor,
			URL:     "https://github.com/nawodyaishan/agentic-sdd#readme",
		},
		LogReader: func(dir string) ([]LogSummary, error) {
			return nil, nil
		},
	})

	lines := model.Lines(80)
	if len(lines) == 0 {
		t.Fatalf("Lines(80) returned empty")
	}

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Activity & Help") {
		t.Errorf("Missing header in rendered lines")
	}
	if !strings.Contains(joined, "v1.2.3") {
		t.Errorf("Missing version in rendered lines")
	}
	if !strings.Contains(joined, "No install or restore logs found yet.") {
		t.Errorf("Missing empty log state in rendered lines")
	}
}

func TestActivity_EmptyState(t *testing.T) {
	cases := []struct {
		name      string
		logReader LogReaderFunc
		wantErr   bool
	}{
		{
			name: "nil slice from reader",
			logReader: func(dir string) ([]LogSummary, error) {
				return nil, nil
			},
		},
		{
			name: "empty slice from reader",
			logReader: func(dir string) ([]LogSummary, error) {
				return []LogSummary{}, nil
			},
		},
		{
			name: "directory read error",
			logReader: func(dir string) ([]LogSummary, error) {
				return nil, errors.New("permission denied")
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := NewActivityModel(ActivityConfig{
				Theme:     NewTheme(false),
				Width:     80,
				LogReader: tc.logReader,
			})

			lines := model.Lines(80)
			joined := strings.Join(lines, "\n")

			if tc.wantErr {
				if !strings.Contains(joined, "permission denied") {
					t.Fatalf("Expected error message in lines, got: %s", joined)
				}
			} else {
				if !strings.Contains(joined, "No install or restore logs found yet.") {
					t.Fatalf("Expected empty state notice, got: %s", joined)
				}
				if !strings.Contains(joined, "Logs are created automatically when you run apply or restore.") {
					t.Fatalf("Expected explanation notice, got: %s", joined)
				}
			}
		})
	}
}

func TestActivity_BoundedLogLoading(t *testing.T) {
	// Create an oversized fake log content with 500 lines and > 100 KB
	var longContent strings.Builder
	for i := 1; i <= 500; i++ {
		longContent.WriteString(fmt.Sprintf("Line %d: %s\n", i, strings.Repeat("A", 250)))
	}
	rawContent := longContent.String()

	fakeSummaries := []LogSummary{
		{
			Name:    "20261110T120000.000000000Z-apply.log",
			Path:    "/injected/logs/20261110T120000.000000000Z-apply.log",
			ModTime: time.Now(),
			Size:    int64(len(rawContent)),
		},
	}

	fakeContentReader := func(path string, maxBytes int64, maxLines int) (string, error) {
		// Verify max limits passed
		if maxBytes != DefaultMaxLogBytes {
			t.Errorf("maxBytes = %d, want %d", maxBytes, DefaultMaxLogBytes)
		}
		if maxLines != DefaultMaxLogLines {
			t.Errorf("maxLines = %d, want %d", maxLines, DefaultMaxLogLines)
		}

		// Truncate to maxBytes and maxLines
		data := rawContent
		if int64(len(data)) > maxBytes {
			data = data[:maxBytes]
		}
		lines := strings.Split(data, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		}
		return strings.Join(lines, "\n") + "\n... [truncated: log exceeds display limit]", nil
	}

	model := NewActivityModel(ActivityConfig{
		Theme: NewTheme(false),
		Width: 80,
		LogReader: func(dir string) ([]LogSummary, error) {
			return fakeSummaries, nil
		},
		ContentReader: fakeContentReader,
	})

	content := model.LogContent()
	if !strings.Contains(content, "[truncated: log exceeds display limit]") {
		t.Fatalf("Content should contain truncation notice, got: %s", content)
	}

	lines := strings.Split(content, "\n")
	if len(lines) > DefaultMaxLogLines+2 { // allow for truncation line
		t.Fatalf("Loaded lines count %d exceeds limit %d", len(lines), DefaultMaxLogLines)
	}

	rendered := strings.Join(model.Lines(80), "\n")
	if !strings.Contains(rendered, "20261110T120000.000000000Z-apply.log") {
		t.Errorf("Rendered view missing log name")
	}
	if !strings.Contains(rendered, "[truncated: log exceeds display limit]") {
		t.Errorf("Rendered view missing truncation notice")
	}
}

func TestActivity_SelectedLogNavigation(t *testing.T) {
	summaries := []LogSummary{
		{Name: "log-1-newest.log", Path: "/logs/log-1.log", Size: 100},
		{Name: "log-2-older.log", Path: "/logs/log-2.log", Size: 200},
		{Name: "log-3-oldest.log", Path: "/logs/log-3.log", Size: 300},
	}

	contentMap := map[string]string{
		"/logs/log-1.log": "Log 1 content: install completed",
		"/logs/log-2.log": "Log 2 content: restore completed",
		"/logs/log-3.log": "Log 3 content: setup preview",
	}

	model := NewActivityModel(ActivityConfig{
		Theme: NewTheme(false),
		Width: 80,
		LogReader: func(dir string) ([]LogSummary, error) {
			return summaries, nil
		},
		ContentReader: func(path string, maxBytes int64, maxLines int) (string, error) {
			return contentMap[path], nil
		},
	})

	// Initial selection is index 0 (newest)
	sel, ok := model.SelectedLog()
	if !ok || sel.Name != "log-1-newest.log" {
		t.Fatalf("Initial SelectedLog() = (%v, %v), want log-1-newest.log", sel, ok)
	}
	if model.LogContent() != "Log 1 content: install completed" {
		t.Fatalf("Initial LogContent() = %q", model.LogContent())
	}

	// Navigate to next log (index 1)
	model.SelectNextLog()
	sel, _ = model.SelectedLog()
	if sel.Name != "log-2-older.log" {
		t.Fatalf("After SelectNextLog(), SelectedLog() = %s, want log-2-older.log", sel.Name)
	}
	if model.LogContent() != "Log 2 content: restore completed" {
		t.Fatalf("After SelectNextLog(), LogContent() = %q", model.LogContent())
	}

	// Navigate to next log (index 2)
	model.SelectNextLog()
	sel, _ = model.SelectedLog()
	if sel.Name != "log-3-oldest.log" {
		t.Fatalf("After SelectNextLog(), SelectedLog() = %s, want log-3-oldest.log", sel.Name)
	}

	// Wraparound
	model.SelectNextLog()
	sel, _ = model.SelectedLog()
	if sel.Name != "log-1-newest.log" {
		t.Fatalf("After wraparound, SelectedLog() = %s, want log-1-newest.log", sel.Name)
	}

	// Previous log
	model.SelectPrevLog()
	sel, _ = model.SelectedLog()
	if sel.Name != "log-3-oldest.log" {
		t.Fatalf("After SelectPrevLog(), SelectedLog() = %s, want log-3-oldest.log", sel.Name)
	}
}

func TestActivity_VersionAndHelpMetadata(t *testing.T) {
	model := NewActivityModel(ActivityConfig{
		Theme: NewTheme(false),
		Width: 80,
		Metadata: AboutMetadata{
			Version: "v2.5.0",
			Commit:  "9876543",
			Date:    "2026-11-10T15:00:00Z",
			Author:  BrandAuthor,
			URL:     "https://github.com/nawodyaishan/agentic-sdd#readme",
		},
		LogReader: func(dir string) ([]LogSummary, error) {
			return nil, nil
		},
	})

	joined := strings.Join(model.Lines(80), "\n")

	// Version & Build
	for _, expected := range []string{"v2.5.0", "9876543", "2026-11-10T15:00:00Z"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("Missing expected build metadata %q in lines", expected)
		}
	}

	// Author and README
	if !strings.Contains(joined, BrandAuthor) {
		t.Errorf("Missing author %q", BrandAuthor)
	}
	if !strings.Contains(joined, "https://github.com/nawodyaishan/agentic-sdd#readme") {
		t.Errorf("Missing README link")
	}

	// Commands
	for _, cmd := range []string{"preview", "apply", "backups", "restore", "logs", "version", "guide", "tui"} {
		if !strings.Contains(joined, "agentic-sdd "+cmd) {
			t.Errorf("Missing CLI command reference for %s", cmd)
		}
	}

	// Shortcuts
	for _, key := range []string{"Tab", "1 - 4", "Enter", "b / Esc", "/", "r", "q / Ctrl+C"} {
		if !strings.Contains(joined, key) {
			t.Errorf("Missing navigation shortcut %s", key)
		}
	}
}

func TestActivity_UpdateCheckIntegration(t *testing.T) {
	cases := []struct {
		name       string
		state      UpdateCheckState
		updateRes  appops.UpdateResult
		wantSubstr string
	}{
		{
			name:       "not started",
			state:      UpdateStatusNotStarted,
			wantSubstr: "Update check: not started",
		},
		{
			name:       "checking in flight",
			state:      UpdateStatusChecking,
			wantSubstr: "Checking for updates (GitHub)...",
		},
		{
			name:  "current version",
			state: UpdateStatusCurrent,
			updateRes: appops.UpdateResult{
				State:   appops.UpdateEqual,
				Current: "v1.0.0",
				Latest:  "v1.0.0",
			},
			wantSubstr: "Agentic-SDD is up to date (v1.0.0)",
		},
		{
			name:  "newer version available",
			state: UpdateStatusNewer,
			updateRes: appops.UpdateResult{
				State:   appops.UpdateNewer,
				Current: "v1.0.0",
				Latest:  "v2.0.0",
			},
			wantSubstr: "Update available: v2.0.0 (installed: v1.0.0)",
		},
		{
			name:       "disabled",
			state:      UpdateStatusDisabled,
			wantSubstr: "Update checks disabled",
		},
		{
			name:       "unavailable",
			state:      UpdateStatusUnavailable,
			wantSubstr: "Update check unavailable (offline or rate-limited)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := NewUpdateCheckManager(
				WithDisabled(tc.state == UpdateStatusDisabled),
				WithCheckFunc(func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
					return tc.updateRes
				}),
				WithEnvLookup(func(string) string { return "" }),
			)

			if tc.state != UpdateStatusNotStarted && tc.state != UpdateStatusDisabled {
				cmd, _ := mgr.AutomaticCheck("v1.0.0")
				if tc.state != UpdateStatusChecking {
					mgr.HandleMsg(cmd())
				}
			}

			model := NewActivityModel(ActivityConfig{
				Theme:         NewTheme(false),
				Width:         80,
				UpdateManager: mgr,
				LogReader: func(string) ([]LogSummary, error) {
					return nil, nil
				},
			})

			lines := model.Lines(80)
			joined := strings.Join(lines, "\n")
			if !strings.Contains(joined, tc.wantSubstr) {
				t.Fatalf("Rendered lines missing expected update status %q:\n%s", tc.wantSubstr, joined)
			}
		})
	}
}

func TestActivity_PlainModeAndBoundsSafety(t *testing.T) {
	summaries := []LogSummary{
		{Name: "20261110T120000.000000000Z-apply.log", Path: "/logs/test.log", Size: 1024},
	}

	widths := []int{40, 60, 79, 80, 109, 110}

	for _, w := range widths {
		t.Run(fmt.Sprintf("width_%d", w), func(t *testing.T) {
			model := NewActivityModel(ActivityConfig{
				Theme: NewTheme(false), // Plain mode: no color, no ANSI escape sequences
				Width: w,
				LogReader: func(string) ([]LogSummary, error) {
					return summaries, nil
				},
				ContentReader: func(string, int64, int) (string, error) {
					return "Installed 4 skills; backup: /home/test/.agentic-sdd/backups/test\nLog: /home/test/.agentic-sdd/logs/test.log", nil
				},
			})

			lines := model.Lines(w)
			for idx, line := range lines {
				// Assert plain mode: no ANSI escapes
				stripped := ansi.Strip(line)
				if stripped != line {
					t.Fatalf("Line %d contains ANSI escapes in plain mode: %q", idx, line)
				}

				// Assert bounds safety: cell width must not exceed requested width
				lineWidth := ansi.StringWidth(line)
				if lineWidth > w {
					t.Fatalf("Line %d cell width (%d) exceeds available width (%d): %q", idx, lineWidth, w, line)
				}
			}
		})
	}
}

func TestActivity_BubbleTeaLifecycle(t *testing.T) {
	model := NewActivityModel(ActivityConfig{
		Theme: NewTheme(false),
		Width: 80,
		LogReader: func(string) ([]LogSummary, error) {
			return []LogSummary{
					{Name: "log1.log", Path: "/log1.log", Size: 10},
					{Name: "log2.log", Path: "/log2.log", Size: 20},
				},
				nil
		},
		ContentReader: func(path string, maxBytes int64, maxLines int) (string, error) {
			return "content for " + path, nil
		},
	})

	// Init contract
	_ = model.Init()

	// Window resize
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	actModel := updated.(ActivityModel)
	if actModel.width != 100 || actModel.height != 30 {
		t.Fatalf("WindowSizeMsg did not update dimensions: w=%d h=%d", actModel.width, actModel.height)
	}

	// Key press: "]" moves to next log
	updated, _ = actModel.Update(tea.KeyPressMsg{Code: ']'})
	actModel = updated.(ActivityModel)
	sel, _ := actModel.SelectedLog()
	if sel.Name != "log2.log" {
		t.Fatalf("Key press ']' did not advance selected log: got %s", sel.Name)
	}

	// Key press: "[" moves to prev log
	updated, _ = actModel.Update(tea.KeyPressMsg{Code: '['})
	actModel = updated.(ActivityModel)
	sel, _ = actModel.SelectedLog()
	if sel.Name != "log1.log" {
		t.Fatalf("Key press '[' did not return to prev log: got %s", sel.Name)
	}

	// View contract
	view := actModel.View()
	if view.AltScreen {
		t.Errorf("View.AltScreen should be false for embedded screen")
	}
	if view.Content == "" {
		t.Errorf("View.Content should not be empty")
	}
}
