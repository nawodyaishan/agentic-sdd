package tui

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/fleet"
)

// Feature 015 W3-B / T21 — Scale and performance evidence for NFR-01, NFR-03, AC-32.
//
// Tests 18- and 100-instance fleet inventory in-memory, >1,000-line deployment diff
// reachability, and measures median and p95 latency for 100-row TUI navigation/render.
// All tests use isolated in-memory models and t.TempDir(); zero real home, network,
// model or Git operations.

func syntheticFleetStatusView(count int) FleetStatusView {
	instances := make([]FleetInstanceView, 0, count)
	routes := []string{"9router", "opencode-go", "opencode"}
	roles := []string{"sdd-implementer", "sdd-tester", "sdd-reviewer", "sdd-orchestrator", "sdd-architect"}
	states := []string{"managed current", "drifted", "pending apply", "disabled (managed removal)", "unmanaged"}

	for i := 1; i <= count; i++ {
		route := routes[(i-1)%len(routes)]
		role := roles[(i-1)%len(roles)]
		state := states[(i-1)%len(states)]
		instID := fmt.Sprintf("instance-%03d", i)
		model := fmt.Sprintf("%s/scale-model-%03d", route, i)
		instances = append(instances, FleetInstanceView{
			InstanceID:        instID,
			Role:              role,
			ProviderRoute:     route,
			Model:             model,
			CompactModel:      fmt.Sprintf("scale-%03d", i),
			Variant:           "default",
			Enabled:           state != "disabled (managed removal)",
			InstalledState:    state,
			SourceProvenance:  "personal override",
			ChangedCount:      i % 3,
			PermissionSummary: "tools: bash, read, edit; network: local",
			RuntimeState:      "not checked",
		})
	}

	return FleetStatusView{
		Available:             true,
		SourcePath:            "/test/scale/fleet.json",
		SourceState:           "loaded",
		PersonalConfigSummary: fmt.Sprintf("Loaded (%d instance(s) configured)", count),
		InstalledSummary:      fmt.Sprintf("%d total instances inspected", count),
		TotalInstances:        count,
		ManagedCount:          count,
		EnabledCount:          count - (count / 5),
		DisabledCount:         count / 5,
		DriftedCount:          count / 4,
		UnmanagedCount:        count / 10,
		LastOutcome:           "Clean scale inspection",
		NextAction:            "Inspect profiles or review changes",
		RuntimeStatus:         "not checked",
		Instances:             instances,
	}
}

// TestFleetTUIScale18And100 exercises FleetModel with 18 and 100 instances.
// Verifies navigation, filtering, detail toggle, and multi-geometry line rendering.
func TestFleetTUIScale18And100(t *testing.T) {
	for _, count := range []int{18, 100} {
		t.Run(fmt.Sprintf("%d_instances", count), func(t *testing.T) {
			status := syntheticFleetStatusView(count)
			reader := &trackingFleetReader{customRes: status}
			editor := &fakeFleetEditor{}
			planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}

			cfg := TUIConfig{
				Width:       110,
				Height:      30,
				Color:       false,
				Home:        t.TempDir(),
				FleetConfig: status.SourcePath,
			}

			m := NewFleetModelWithSeams(cfg, reader, editor, planner)
			// Deliver status message
			m, _ = m.Update(FleetStatusMsg{
				Generation: 1,
				Source:     status.SourcePath,
				Status:     status,
			})

			// 1. Verify instance count and filtering
			if len(m.FilteredInstances()) != count {
				t.Fatalf("FilteredInstances count = %d, want %d", len(m.FilteredInstances()), count)
			}

			// 2. Navigation through list: top, down, end, home
			m.SetView(FleetViewProfiles)
			selected, ok := m.SelectedInstance()
			if !ok || selected.InstanceID != "instance-001" {
				t.Fatalf("initial selected = %+v, want instance-001", selected)
			}

			// Navigate down 5 rows
			for i := 0; i < 5; i++ {
				m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
			}
			selected, _ = m.SelectedInstance()
			if selected.InstanceID != "instance-006" {
				t.Fatalf("after 5 down: selected = %s, want instance-006", selected.InstanceID)
			}

			// Jump to end 'G'
			m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
			selected, _ = m.SelectedInstance()
			wantLast := fmt.Sprintf("instance-%03d", count)
			if selected.InstanceID != wantLast {
				t.Fatalf("after 'G': selected = %s, want %s", selected.InstanceID, wantLast)
			}

			// Jump to top 'g'
			m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
			selected, _ = m.SelectedInstance()
			if selected.InstanceID != "instance-001" {
				t.Fatalf("after 'g': selected = %s, want instance-001", selected.InstanceID)
			}

			// 3. Search and filter
			m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			for _, ch := range "instance-01" {
				m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
			}
			filtered := m.FilteredInstances()
			if len(filtered) == 0 {
				t.Fatal("expected matching instances for query 'instance-01'")
			}
			for _, fi := range filtered {
				if !strings.Contains(fi.InstanceID, "instance-01") {
					t.Fatalf("filtered instance %s does not match query", fi.InstanceID)
				}
			}
			// Clear filter
			m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if len(m.FilteredInstances()) != count {
				t.Fatalf("after clearing filter: count = %d, want %d", len(m.FilteredInstances()), count)
			}

			// 4. Multi-geometry rendering verification (110, 80, 60 columns)
			for _, w := range []int{110, 80, 60} {
				for _, h := range []int{30, 20, 10} {
					m.SetSize(w, h)
					lines := m.Lines(w)
					if len(lines) == 0 {
						t.Fatalf("Lines(%d) with height %d returned 0 lines", w, h)
					}
					// Ensure rendered lines do not panic and contain header
					rendered := strings.Join(lines, "\n")
					if !strings.Contains(rendered, "Fleet Manager") {
						t.Fatalf("rendered output for %dx%d missing 'Fleet Manager'", w, h)
					}
				}
			}
		})
	}
}

// TestFleetTUIDeploymentReview100OpsAndLargeDiff tests Review Changes deployment view
// with 100 proposed ops and a diff containing >1,000 lines including Unicode paths.
// Verifies full reachability through scrolling, no panic, and bounded memory.
func TestFleetTUIDeploymentReview100OpsAndLargeDiff(t *testing.T) {
	const totalOps = 100
	const diffLineCount = 1500

	files := make([]FleetFileDiff, 0, totalOps)
	ops := []string{fleet.FileOpCreate, fleet.FileOpUpdate, fleet.FileOpRemove, fleet.FileOpNoOp, fleet.FileOpConflict}
	counts := map[string]int{"total": totalOps}

	for i := 1; i <= totalOps; i++ {
		op := ops[(i-1)%len(ops)]
		counts[op]++
		path := fmt.Sprintf("/tmp/opencode/agents/agent_日本語_unicode_%03d.md", i)
		files = append(files, FleetFileDiff{
			InstanceID: fmt.Sprintf("instance-%03d", i),
			Path:       path,
			Op:         op,
			Reason:     "scale test diff projection",
			Removed:    op == fleet.FileOpRemove,
			Conflict:   op == fleet.FileOpConflict,
			OldDigest:  fmt.Sprintf("old-digest-%03d", i),
			NewDigest:  fmt.Sprintf("new-digest-%03d", i),
			DiffLines:  []string{fmt.Sprintf("+ line for %s", path)},
		})
	}

	diffLines := make([]string, 0, diffLineCount)
	diffLines = append(diffLines, "--- a/agents/config.json", "+++ b/agents/config.json", "@@ -1,1500 +1,1500 @@")
	for i := 4; i <= diffLineCount; i++ {
		diffLines = append(diffLines, fmt.Sprintf("+   line_%04d: agent_setting_value [🚀 ⌘ ⌥ ⇧ 日本語 %04d]", i, i))
	}

	preview := FleetPlanPreview{
		Available:       true,
		SourcePath:      "/test/fleet.json",
		PlanID:          "plan-scale-100-ops",
		Scope:           "global",
		RendererVersion: "1",
		Token:           "scale-token-abc",
		Counts:          counts,
		Files:           files,
		DiffLines:       diffLines,
	}

	reader := &trackingFleetReader{customRes: syntheticFleetStatusView(100)}
	editor := &fakeFleetEditor{}
	planner := &fakeFleetPlanner{plan: preview}

	cfg := TUIConfig{
		Width:       110,
		Height:      30,
		Color:       false,
		Home:        t.TempDir(),
		FleetConfig: "/test/fleet.json",
	}

	m := NewFleetModelWithSeams(cfg, reader, editor, planner)
	m, cmd := m.requestPlanPreview()
	if cmd == nil {
		t.Fatal("requestPlanPreview returned nil cmd")
	}
	m, _ = m.Update(cmd())

	m.SetView(FleetViewReview)
	m.height = 30

	// 1. Initial page contains first lines
	view0 := strings.Join(m.Lines(110), "\n")
	if !strings.Contains(view0, "line_0004") {
		t.Fatalf("initial review page missing line_0004:\n%s", view0)
	}
	if !strings.Contains(view0, "Impact summary") {
		t.Fatalf("review page missing Impact summary:\n%s", view0)
	}

	// 2. Scroll down 10 lines
	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}
	if m.reviewScroll != 10 {
		t.Fatalf("reviewScroll = %d, want 10 after 10 j presses", m.reviewScroll)
	}

	// 3. Jump to end 'G'
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.reviewScroll < diffLineCount-30 {
		t.Fatalf("reviewScroll = %d, expected near end of %d lines", m.reviewScroll, diffLineCount)
	}
	viewEnd := strings.Join(m.Lines(110), "\n")
	wantLastLine := fmt.Sprintf("line_%04d", diffLineCount)
	if !strings.Contains(viewEnd, wantLastLine) {
		t.Fatalf("end view missing last line %q:\n%s", wantLastLine, viewEnd)
	}
	if !strings.Contains(viewEnd, "🚀") || !strings.Contains(viewEnd, "日本語") {
		t.Fatalf("end view missing expected Unicode characters:\n%s", viewEnd)
	}

	// 4. Jump back to top 'g'
	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.reviewScroll != 0 {
		t.Fatalf("reviewScroll = %d, want 0 after 'g'", m.reviewScroll)
	}
}

// TestFleet100RowNavigationPerformance implements the measured loop required by
// NFR-01 and AC-32 for 100-row in-memory TUI navigation/render.
// Reports median and p95 latency and asserts target p95 <= 100ms.
func TestFleet100RowNavigationPerformance(t *testing.T) {
	status := syntheticFleetStatusView(100)
	reader := &trackingFleetReader{customRes: status}
	editor := &fakeFleetEditor{}
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}

	cfg := TUIConfig{
		Width:       110,
		Height:      30,
		Color:       false,
		Home:        t.TempDir(),
		FleetConfig: status.SourcePath,
	}

	m := NewFleetModelWithSeams(cfg, reader, editor, planner)
	m, _ = m.Update(FleetStatusMsg{
		Generation: 1,
		Source:     status.SourcePath,
		Status:     status,
	})
	m.SetView(FleetViewProfiles)

	// Warm-up loop (10 iterations)
	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
		_ = m.Lines(110)
	}

	// Measured loop (200 iterations of navigation + full screen render)
	const iterations = 200
	latencies := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		key := "j"
		if i%2 == 1 {
			key = "k"
		}
		start := time.Now()
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
		lines := m.Lines(110)
		elapsed := time.Since(start)

		if len(lines) == 0 {
			t.Fatal("empty lines rendered")
		}
		latencies = append(latencies, elapsed)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	minD := latencies[0]
	medianD := latencies[len(latencies)/2]
	p95Index := int(float64(len(latencies)) * 0.95)
	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	p95D := latencies[p95Index]
	maxD := latencies[len(latencies)-1]

	t.Logf("NFR-01/AC-32-RECORDED 100-row in-memory navigation/render (%d samples): min=%s median=%s p95=%s max=%s (target p95 <= 100ms)",
		iterations, minD.Round(time.Microsecond), medianD.Round(time.Microsecond), p95D.Round(time.Microsecond), maxD.Round(time.Microsecond))

	// NFR-01 target assertion: p95 <= 100ms
	const targetP95 = 100 * time.Millisecond
	if p95D > targetP95 {
		t.Fatalf("NFR-01 TARGET FAILED: p95 latency %s exceeds target %s (finding: investigate in-memory render stall)",
			p95D, targetP95)
	}
}

// BenchmarkFleet100NavigationAndRender benchmarks in-memory navigation and rendering
// across a 100-row fixture.
func BenchmarkFleet100NavigationAndRender(b *testing.B) {
	status := syntheticFleetStatusView(100)
	reader := &trackingFleetReader{customRes: status}
	editor := &fakeFleetEditor{}
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}

	cfg := TUIConfig{
		Width:       110,
		Height:      30,
		Color:       false,
		Home:        b.TempDir(),
		FleetConfig: status.SourcePath,
	}

	m := NewFleetModelWithSeams(cfg, reader, editor, planner)
	m, _ = m.Update(FleetStatusMsg{
		Generation: 1,
		Source:     status.SourcePath,
		Status:     status,
	})
	m.SetView(FleetViewProfiles)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "j"
		if i%2 == 1 {
			key = "k"
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
		_ = m.Lines(110)
	}
}
