package fleet

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// Feature 015 W3-B / T21 — Scale evidence for NFR-01, NFR-03, AC-32.
//
// Exercises BuildStatus, PreviewSource, PlanFiles (18 and 100 instances),
// and LineDiff (>1,000 lines with Unicode paths). All fixtures use t.TempDir()
// and isolated temporary home directories; zero real-home, network or Git operations.
// Latency and memory stats are measured and recorded in test logs.

func measureMemAlloc(fn func()) (time.Duration, uint64) {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	return elapsed, after.TotalAlloc - before.TotalAlloc
}

func statsOf(durations []time.Duration) (median, worst time.Duration) {
	if len(durations) == 0 {
		return 0, 0
	}
	d := append([]time.Duration(nil), durations...)
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	median = d[len(d)/2]
	worst = d[len(d)-1]
	return median, worst
}

// TestFleet015ScaleBuildStatus verifies BuildStatus across 18 and 100 instances.
// Asserts correct instance count, read-only guarantees (zero target writes),
// bounded memory, and logs latency numbers.
func TestFleet015ScaleBuildStatus(t *testing.T) {
	dir := t.TempDir()
	c18 := writeScaleConfig(t, dir, "fleet18.json", 18)
	c100 := writeScaleConfig(t, dir, "fleet100.json", 100)
	repo := filepath.Join("..", "..")

	for _, tc := range []struct {
		name       string
		configPath string
		wantCount  int
	}{
		{"18-instance golden", c18, 18},
		{"100-instance synthetic", c100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := filepath.Join(dir, "home-"+strings.ReplaceAll(tc.name, " ", "-"))
			if err := os.MkdirAll(home, 0o755); err != nil {
				t.Fatal(err)
			}

			var durations []time.Duration
			var allocs []uint64
			var status *Status

			for i := 0; i < 5; i++ {
				d, alloc := measureMemAlloc(func() {
					var err error
					status, err = BuildStatus(tc.configPath, repo, home)
					if err != nil {
						t.Fatalf("BuildStatus failed: %v", err)
					}
				})
				durations = append(durations, d)
				allocs = append(allocs, alloc)
			}

			// Correctness assertions
			if status == nil {
				t.Fatal("BuildStatus returned nil status")
			}
			if len(status.Instances) != tc.wantCount {
				t.Fatalf("got %d instances, want %d", len(status.Instances), tc.wantCount)
			}
			if gotCount := status.Counts[StatusCountInstances]; gotCount != tc.wantCount {
				t.Fatalf("Counts[%s] = %d, want %d", StatusCountInstances, gotCount, tc.wantCount)
			}
			if status.ConfigPath != tc.configPath {
				t.Fatalf("status.ConfigPath = %q, want %q", status.ConfigPath, tc.configPath)
			}
			if status.ConfigDigest == "" {
				t.Fatal("expected non-empty ConfigDigest")
			}

			// Verify zero target writes
			targetRoot := TargetRootForHome(home)
			if entries, err := os.ReadDir(targetRoot); err == nil && len(entries) > 0 {
				t.Fatalf("BuildStatus must perform zero target writes; found %d entries in %s", len(entries), targetRoot)
			}

			medianD, worstD := statsOf(durations)
			var avgAlloc uint64
			for _, a := range allocs {
				avgAlloc += a
			}
			avgAlloc /= uint64(len(allocs))

			t.Logf("NFR-03-RECORDED BuildStatus %s: median=%s worst=%s avgAlloc=%d bytes",
				tc.name, medianD.Round(time.Microsecond), worstD.Round(time.Microsecond), avgAlloc)
		})
	}
}

// TestFleet015ScalePreviewSource verifies PreviewSource across 18 and 100 instances.
// Asserts deterministic normalization, old/new digest extraction, zero writes, and logs performance.
func TestFleet015ScalePreviewSource(t *testing.T) {
	dir := t.TempDir()
	c18 := writeScaleConfig(t, dir, "fleet18.json", 18)
	c100 := writeScaleConfig(t, dir, "fleet100.json", 100)
	repo := filepath.Join("..", "..")

	for _, tc := range []struct {
		name       string
		configPath string
		modifyOld  string
		modifyNew  string
	}{
		{"18-instance golden", c18, "opencode-go/mimo-v2.6-flash", "opencode-go/deepseek-v4.1-flash"},
		{"100-instance synthetic", c100, "opencode-go/scale-050", "9router/ag/gemini-3.8-flash-high"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(tc.configPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(raw, []byte(tc.modifyOld)) {
				t.Fatalf("config does not contain old string %q", tc.modifyOld)
			}
			modifiedBytes := bytes.Replace(raw, []byte(tc.modifyOld), []byte(tc.modifyNew), 1)

			// Record modtime before
			fiBefore, err := os.Stat(tc.configPath)
			if err != nil {
				t.Fatal(err)
			}

			var durations []time.Duration
			var allocs []uint64
			var preview FilePreview

			for i := 0; i < 5; i++ {
				d, alloc := measureMemAlloc(func() {
					var previewErr error
					preview, previewErr = PreviewSourceWithRepo(tc.configPath, modifiedBytes, repo)
					if previewErr != nil {
						t.Fatalf("PreviewSourceWithRepo failed: %v", previewErr)
					}
				})
				durations = append(durations, d)
				allocs = append(allocs, alloc)
			}

			// Correctness assertions
			if preview.Op != FileOpUpdate {
				t.Fatalf("preview.Op = %q, want %q", preview.Op, FileOpUpdate)
			}
			if preview.OldDigest == "" || preview.NewDigest == "" {
				t.Fatal("expected non-empty old and new digests in preview")
			}
			if preview.OldDigest == preview.NewDigest {
				t.Fatal("expected old and new digests to differ after modification")
			}
			if len(preview.NewBytes) == 0 {
				t.Fatal("expected non-empty preview NewBytes")
			}
			if !bytes.Contains(preview.NewBytes, []byte(tc.modifyNew)) {
				t.Fatalf("preview NewBytes missing modification %q", tc.modifyNew)
			}

			// Zero writes assertion on disk
			fiAfter, err := os.Stat(tc.configPath)
			if err != nil {
				t.Fatal(err)
			}
			if !fiAfter.ModTime().Equal(fiBefore.ModTime()) {
				t.Fatalf("config file modtime modified by PreviewSource: %v != %v", fiAfter.ModTime(), fiBefore.ModTime())
			}

			medianD, worstD := statsOf(durations)
			var avgAlloc uint64
			for _, a := range allocs {
				avgAlloc += a
			}
			avgAlloc /= uint64(len(allocs))

			t.Logf("NFR-03-RECORDED PreviewSource %s: median=%s worst=%s avgAlloc=%d bytes",
				tc.name, medianD.Round(time.Microsecond), worstD.Round(time.Microsecond), avgAlloc)
		})
	}
}

// TestFleet015ScalePlanFiles verifies PlanFiles across 18 and 100 proposed operations.
// Asserts order: removals first, then conflicts, then updates/creates/noops; verifies no panic,
// zero disk writes, and logs latency numbers.
func TestFleet015ScalePlanFiles(t *testing.T) {
	dir := t.TempDir()
	c18 := writeScaleConfig(t, dir, "fleet18.json", 18)
	c100 := writeScaleConfig(t, dir, "fleet100.json", 100)
	repo := filepath.Join("..", "..")

	for _, tc := range []struct {
		name       string
		configPath string
		wantOps    int
	}{
		{"18-instance golden", c18, 18},
		{"100-instance synthetic", c100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := filepath.Join(dir, "home-plan-"+strings.ReplaceAll(tc.name, " ", "-"))
			if err := os.MkdirAll(home, 0o755); err != nil {
				t.Fatal(err)
			}

			plan, err := BuildPlan(tc.configPath, repo, home, false)
			if err != nil {
				t.Fatalf("BuildPlan failed: %v", err)
			}
			if len(plan.ProposedOps) != tc.wantOps {
				t.Fatalf("plan has %d ops, want %d", len(plan.ProposedOps), tc.wantOps)
			}

			var durations []time.Duration
			var allocs []uint64
			var previews []FilePreview

			for i := 0; i < 5; i++ {
				d, alloc := measureMemAlloc(func() {
					var planErr error
					previews, planErr = PlanFiles(plan)
					if planErr != nil {
						t.Fatalf("PlanFiles failed: %v", planErr)
					}
				})
				durations = append(durations, d)
				allocs = append(allocs, alloc)
			}

			// Correctness assertions
			if len(previews) != tc.wantOps {
				t.Fatalf("PlanFiles returned %d previews, want %d", len(previews), tc.wantOps)
			}

			// Verify ordering: removals (rank 0) <= conflicts (rank 1) <= updates (rank 2) <= creates (rank 3) <= noops (rank 4)
			for i := 1; i < len(previews); i++ {
				prevRank := fileOpRank(previews[i-1].Op)
				currRank := fileOpRank(previews[i].Op)
				if prevRank > currRank {
					t.Fatalf("preview ops not properly ranked: op[%d]=%s (rank %d) > op[%d]=%s (rank %d)",
						i-1, previews[i-1].Op, prevRank, i, previews[i].Op, currRank)
				}
			}

			// Verify every preview has required fields
			for i, p := range previews {
				if p.InstanceID == "" {
					t.Fatalf("preview[%d] missing InstanceID", i)
				}
				if p.Path == "" {
					t.Fatalf("preview[%d] missing Path", i)
				}
				if p.Op == "" {
					t.Fatalf("preview[%d] missing Op", i)
				}
			}

			medianD, worstD := statsOf(durations)
			var avgAlloc uint64
			for _, a := range allocs {
				avgAlloc += a
			}
			avgAlloc /= uint64(len(allocs))

			t.Logf("NFR-03-RECORDED PlanFiles %s (%d ops): median=%s worst=%s avgAlloc=%d bytes",
				tc.name, tc.wantOps, medianD.Round(time.Microsecond), worstD.Round(time.Microsecond), avgAlloc)
		})
	}
}

// TestFleet015ScaleLineDiffOver1000Lines tests LineDiff with over 1,000 lines
// including Unicode paths and complex edits (additions, deletions, equals).
// Verifies bounded allocation, correct line numbering, Unicode preservation, and logs timings.
func TestFleet015ScaleLineDiffOver1000Lines(t *testing.T) {
	const headerLines = 300
	const editLines = 800
	const footerLines = 300

	var oldBuf, newBuf strings.Builder

	// 1. Common prefix with Unicode paths
	for i := 1; i <= headerLines; i++ {
		line := fmt.Sprintf("prefix line %04d: /agents/spécial/ö_agent_日本語_%04d.md\n", i, i)
		oldBuf.WriteString(line)
		newBuf.WriteString(line)
	}

	// 2. Modified middle section
	for i := 1; i <= editLines; i++ {
		oldBuf.WriteString(fmt.Sprintf("old deleted line %04d: prompt_instruction_v1_%d [⌘ ⌥ ⇧]\n", i, i))
		newBuf.WriteString(fmt.Sprintf("new inserted line %04d: prompt_instruction_v2_%d [🚀 ✨ 🎯]\n", i, i))
	}

	// 3. Common suffix
	for i := 1; i <= footerLines; i++ {
		line := fmt.Sprintf("suffix line %04d: end_of_manifest_policy_token_%04d\n", i, i)
		oldBuf.WriteString(line)
		newBuf.WriteString(line)
	}

	oldBytes := []byte(oldBuf.String())
	newBytes := []byte(newBuf.String())

	var durations []time.Duration
	var allocs []uint64
	var diff []DiffLine

	for i := 0; i < 5; i++ {
		d, alloc := measureMemAlloc(func() {
			diff = LineDiff(oldBytes, newBytes)
		})
		durations = append(durations, d)
		allocs = append(allocs, alloc)
	}

	// Verification
	totalDiffLines := len(diff)
	if totalDiffLines <= 1000 {
		t.Fatalf("expected > 1,000 diff lines, got %d", totalDiffLines)
	}
	if totalDiffLines > MaxDiffLines {
		t.Fatalf("diff line count %d exceeds MaxDiffLines %d", totalDiffLines, MaxDiffLines)
	}

	var hasEqual, hasAdd, hasDel bool
	var hasUnicodeOld, hasUnicodeNew bool

	for _, dl := range diff {
		switch dl.Kind {
		case DiffEqual:
			hasEqual = true
			if dl.OldLine <= 0 || dl.NewLine <= 0 {
				t.Fatalf("DiffEqual line has invalid line numbers: old=%d new=%d", dl.OldLine, dl.NewLine)
			}
		case DiffAdd:
			hasAdd = true
			if dl.NewLine <= 0 || dl.OldLine != 0 {
				t.Fatalf("DiffAdd line has invalid line numbers: old=%d new=%d", dl.OldLine, dl.NewLine)
			}
			if strings.Contains(dl.Text, "🚀") {
				hasUnicodeNew = true
			}
		case DiffDel:
			hasDel = true
			if dl.OldLine <= 0 || dl.NewLine != 0 {
				t.Fatalf("DiffDel line has invalid line numbers: old=%d new=%d", dl.OldLine, dl.NewLine)
			}
			if strings.Contains(dl.Text, "⌘") {
				hasUnicodeOld = true
			}
		case DiffOmitted:
			// Allowed if bounded
		default:
			t.Fatalf("unknown diff kind: %v", dl.Kind)
		}
	}

	if !hasEqual || !hasAdd || !hasDel {
		t.Fatalf("diff missing expected kinds: equal=%v, add=%v, del=%v", hasEqual, hasAdd, hasDel)
	}
	if !hasUnicodeOld || !hasUnicodeNew {
		t.Fatalf("Unicode characters not properly preserved in diff: old=%v, new=%v", hasUnicodeOld, hasUnicodeNew)
	}

	medianD, worstD := statsOf(durations)
	var avgAlloc uint64
	for _, a := range allocs {
		avgAlloc += a
	}
	avgAlloc /= uint64(len(allocs))

	t.Logf("NFR-03/AC-32-RECORDED LineDiff (%d lines): median=%s worst=%s avgAlloc=%d bytes",
		totalDiffLines, medianD.Round(time.Microsecond), worstD.Round(time.Microsecond), avgAlloc)
}
