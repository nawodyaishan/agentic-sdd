package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFleetReviewChangesImpactSummary(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)

	// 1. Review Changes with 0 edits staged
	m.SetView(FleetViewReview)
	viewZero := strings.Join(m.Lines(100), "\n")

	for _, want := range []string{
		"Install & Manage / Fleet Manager · Review Changes",
		"Impact summary",
		"Staged in-memory edits: 0 edit(s)",
		"Personal source file: /test/fleet.json",
		"Installed agent files: Unchanged (0 OpenCode files modified)",
		"Save status: NOT SAVED (in-memory draft only; B4 source save not authorized)",
		"No changes staged. Select an instance in Profiles",
	} {
		if !strings.Contains(viewZero, want) {
			t.Errorf("Review view with 0 edits missing %q:\n%s", want, viewZero)
		}
	}

	// Must NOT contain Save button or claim edits are saved
	for _, forbidden := range []string{"[save]", "save changes", "saved to disk", "apply plan"} {
		if strings.Contains(strings.ToLower(viewZero), forbidden) {
			t.Errorf("Review view contains forbidden action %q:\n%s", forbidden, viewZero)
		}
	}

	// 2. Stage 2 edits
	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}
	if err := m.StageToggle("sdd-spec-9r"); err != nil {
		t.Fatal(err)
	}

	viewStaged := strings.Join(m.Lines(100), "\n")
	for _, want := range []string{
		"Staged in-memory edits: 2 edit(s)",
		"Personal source file: /test/fleet.json",
		"Installed agent files: Unchanged (0 OpenCode files modified)",
		"Save status: NOT SAVED",
		"instance sdd-implement-9r-1:",
		"+   model: 9router/ag/gemini-4-argon-medium",
		"instance sdd-spec-9r:",
		"+   enabled: true",
		"Note: Staged edits remain in memory only. No save button is offered (B4 not authorized).",
	} {
		if !strings.Contains(viewStaged, want) {
			t.Errorf("Review view with staged edits missing %q:\n%s", want, viewStaged)
		}
	}
}

func TestFleetReviewSourceDiffRenderingAndColor(t *testing.T) {
	sampleDiff := []string{
		"--- a/test/fleet.json",
		"+++ b/test/fleet.json",
		"@@ -10,4 +10,4 @@",
		"   \"sdd-implement-9r-1\": {",
		"-    \"model\": \"ag/gemini-3.8-flash-high\"",
		"+    \"model\": \"9router/ag/gemini-4-argon-medium\"",
		"   }",
	}
	editor := &fakeFleetEditor{customDiff: sampleDiff}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)

	// Stage an edit to activate preview
	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}
	m, cmd := m.requestSourcePreview()
	if cmd == nil {
		t.Fatal("expected preview command")
	}
	m, _ = m.Update(cmd())

	m.SetView(FleetViewReview)

	// Test non-color rendering
	m.cfg.Color = false
	m.theme = newThemeForConfig(m.cfg)
	viewNoColor := strings.Join(m.Lines(100), "\n")
	for _, line := range sampleDiff {
		if !strings.Contains(viewNoColor, line) {
			t.Errorf("Review view missing diff line %q:\n%s", line, viewNoColor)
		}
	}

	// Test color rendering
	m.cfg.Color = true
	m.theme = newThemeForConfig(m.cfg)
	viewColor := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(viewColor, "gemini-4-argon-medium") {
		t.Errorf("Color review view missing staged model:\n%s", viewColor)
	}

	// Navigation back to Profiles via 'p'
	m, _ = m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if m.ActiveView() != FleetViewProfiles {
		t.Fatalf("after pressing 'p': view = %v, want FleetViewProfiles", m.ActiveView())
	}

	// Return to Review, then press 'esc' -> returns to Profiles
	m.SetView(FleetViewReview)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.ActiveView() != FleetViewProfiles {
		t.Fatalf("after pressing Esc: view = %v, want FleetViewProfiles", m.ActiveView())
	}
}

func TestFleetReviewLongDiffPaging(t *testing.T) {
	// Generate a 1,200 line diff
	const totalLines = 1200
	longDiff := make([]string, 0, totalLines)
	longDiff = append(longDiff, "--- a/test/fleet.json", "+++ b/test/fleet.json", "@@ -1,1200 +1,1200 @@")
	for i := 4; i <= totalLines; i++ {
		longDiff = append(longDiff, fmt.Sprintf("+   line_%04d: config_value_for_testing_long_diff", i))
	}

	editor := &fakeFleetEditor{customDiff: longDiff}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)

	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}
	m, cmd := m.requestSourcePreview()
	if cmd != nil {
		m, _ = m.Update(cmd())
	}

	m.SetView(FleetViewReview)
	m.height = 30 // viewport height

	// 1. Initial page at top
	view0 := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(view0, "line_0004") {
		t.Fatalf("initial page should contain line_0004:\n%s", view0)
	}
	if !strings.Contains(view0, "Showing diff lines 1-") {
		t.Fatalf("initial page should display page range indicator:\n%s", view0)
	}

	// 2. Scroll down with 'j'
	for i := 0; i < 5; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}
	if m.reviewScroll != 5 {
		t.Fatalf("reviewScroll = %d, want 5 after 5 j presses", m.reviewScroll)
	}

	// 3. Jump to end with 'G' (end key)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.reviewScroll < totalLines-20 {
		t.Fatalf("reviewScroll = %d, expected near end of 1200 lines", m.reviewScroll)
	}
	viewEnd := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(viewEnd, "line_1200") {
		t.Fatalf("end view should contain last line (line_1200):\n%s", viewEnd)
	}

	// 4. Return to top with 'g' (home key)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.reviewScroll != 0 {
		t.Fatalf("reviewScroll = %d, want 0 after 'g'", m.reviewScroll)
	}
	viewTop := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(viewTop, "line_0004") {
		t.Fatalf("top view should contain line_0004:\n%s", viewTop)
	}
}

func TestFleetStaleEditorReplyDiscarded(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)

	// Stage edit 1 -> increments previewGen to 1
	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}
	m, _ = m.requestSourcePreview()
	gen1 := m.previewGen

	// Stage edit 2 -> increments previewGen to 2
	if err := m.StageModel("sdd-orchestrator", "opencode-go/qwen3.8-flash"); err != nil {
		t.Fatal(err)
	}
	m, _ = m.requestSourcePreview()
	gen2 := m.previewGen

	if gen1 == gen2 {
		t.Fatal("expected incremented generation for second edit")
	}

	// 1. Simulate arrival of STALE preview msg (gen1)
	staleMsg := FleetSourcePreviewMsg{
		Generation: gen1,
		Source:     "/test/fleet.json",
		Preview: FleetSourcePreview{
			Available:  true,
			SourcePath: "/test/fleet.json",
			BaseDigest: "stale_digest_gen1",
			DiffLines:  []string{"+ stale line from gen 1"},
		},
	}
	m, _ = m.Update(staleMsg)

	// Verify stale reply was completely ignored
	if m.BaseDigest() == "stale_digest_gen1" {
		t.Fatal("stale preview generation 1 was accepted; must be discarded")
	}
	if len(m.SourcePreview().DiffLines) > 0 && m.SourcePreview().DiffLines[0] == "+ stale line from gen 1" {
		t.Fatal("stale preview diff was accepted; must be discarded")
	}
	// Verify draft was NOT cleared by stale reply
	if m.DirtyCount() != 2 {
		t.Fatalf("dirtyCount = %d, want 2 preserved", m.DirtyCount())
	}

	// 2. Simulate arrival of msg with WRONG source path
	wrongPathMsg := FleetSourcePreviewMsg{
		Generation: gen2,
		Source:     "/wrong/source.json",
		Preview: FleetSourcePreview{
			Available:  true,
			SourcePath: "/wrong/source.json",
			BaseDigest: "wrong_source_digest",
		},
	}
	m, _ = m.Update(wrongPathMsg)
	if m.BaseDigest() == "wrong_source_digest" {
		t.Fatal("mismatched source path preview was accepted; must be discarded")
	}

	// 3. Deliver valid gen2 reply
	validMsg := FleetSourcePreviewMsg{
		Generation: gen2,
		Source:     "/test/fleet.json",
		Preview: FleetSourcePreview{
			Available:  true,
			SourcePath: "/test/fleet.json",
			BaseDigest: "sha256:valid_gen2_base",
			DiffLines:  []string{"+ valid line from gen 2"},
		},
	}
	m, _ = m.Update(validMsg)
	if m.BaseDigest() != "sha256:valid_gen2_base" {
		t.Fatalf("BaseDigest = %q, want sha256:valid_gen2_base", m.BaseDigest())
	}
	if len(m.SourcePreview().DiffLines) != 1 || m.SourcePreview().DiffLines[0] != "+ valid line from gen 2" {
		t.Fatalf("unexpected diff lines: %+v", m.SourcePreview().DiffLines)
	}
}

func TestFleetViewFilterScrollContinuity(t *testing.T) {
	editor := &fakeFleetEditor{}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := newTestFleetModel("/test/fleet.json", reader, editor)

	// Stage an edit
	if err := m.StageModel("sdd-implement-9r-1", "9router/ag/gemini-4-argon-medium"); err != nil {
		t.Fatal(err)
	}

	// 1. Cycle views: Profiles -> Overview -> Review -> Recovery -> Profiles
	views := []FleetView{
		FleetViewOverview,
		FleetViewProfiles,
		FleetViewReview,
		FleetViewRecovery,
		FleetViewProfiles,
	}
	for _, v := range views {
		m.SetView(v)
		if m.DirtyCount() != 1 {
			t.Fatalf("switching to view %v dropped dirtyCount (got %d, want 1)", v, m.DirtyCount())
		}
		if _, has := m.Draft()["sdd-implement-9r-1"]; !has {
			t.Fatalf("switching to view %v lost draft edit", v)
		}
	}

	// 2. Set filter in Profiles
	m.view = FleetViewProfiles
	m.filter = "implement"
	if m.DirtyCount() != 1 {
		t.Fatalf("filtering dropped dirtyCount (got %d, want 1)", m.DirtyCount())
	}

	// 3. Toggle detailOpen
	m.detailOpen = true
	if m.DirtyCount() != 1 {
		t.Fatal("toggling detailOpen dropped dirtyCount")
	}
	m.detailOpen = false

	// 4. Resize viewport
	widths := []int{120, 110, 80, 60, 50, 120}
	for _, w := range widths {
		m.SetSize(w, 24)
		_ = m.Lines(w)
		if m.DirtyCount() != 1 {
			t.Fatalf("resizing to width %d dropped dirtyCount", w)
		}
	}

	// 5. Open discard prompt and cancel
	m.confirmDiscard = true
	_ = m.Lines(100)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"}) // keep editing
	if m.DirtyCount() != 1 {
		t.Fatal("opening and cancelling discard prompt dropped dirtyCount")
	}
}
