package fleet

// Feature 014 B5 / T10 & B9 / T14 — scale fixtures for AC-25 / NFR-02.
//
// Two offline fixtures: the golden 18-name personal fleet (spec §2.1
// reconciliation, lane strings are opaque fixtures per R9) and a 100-instance
// synthetic fleet. Timings and allocations are RECORDED for the completion
// report, never asserted as passes: NFR-02 covers offline
// parse/resolve/render/plan design target (< 2s on designated CI runner).
// Both doctor labeling and full end-to-end BuildPlan (parse, resolve, render,
// plan) pipelines are measured over isolated temp-home directories.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// fleet18Golden is the all-18-name personal config shape: five approved 009
// overlaps as agent: sources, two reviewed read-only adapters under
// opencode-pack/adapters/ (B6 Option A), vision mapped to the canonical
// sdd-visual-inspector role, four Free-first preset names, and six machine
// names reusing reviewed preset bodies (R1). It exercises the representation
// AC-02 claims, without 18 duplicated prompts.
const fleet18Golden = `{
  "schemaVersion": 1,
  "fleet": "personal",
  "preset": "opencode-go",
  "instances": {
    "sdd-architecture-review-9r": {"source": "agent:sdd-architecture-review-9r", "model": "9router/ag/gemini-3.8-flash-high", "enabled": true},
    "sdd-code-review-9r": {"source": "agent:sdd-code-review-9r", "model": "9router/ag/gemini-3.8-flash-high", "enabled": true},
    "sdd-review-verification-1": {"source": "agent:sdd-review-verification-1", "model": "opencode-go/deepseek-v4.1-flash", "enabled": true},
    "sdd-review-verification-2": {"source": "agent:sdd-review-verification-2", "model": "opencode-go/glm-5.3-flash", "enabled": true},
    "sdd-lightweight": {"source": "agent:sdd-lightweight", "model": "opencode-go/mimo-v2.6-flash", "enabled": true},
    "sdd-implement-1": {"source": "preset:sdd-implement-1", "model": "opencode-go/deepseek-v4.1-flash#high", "enabled": true},
    "sdd-implement-2": {"source": "preset:sdd-implement-2", "model": "opencode-go/deepseek-v4.1-flash#max", "enabled": true},
    "sdd-test-implement-1": {"source": "preset:sdd-test-implement-1", "model": "opencode-go/deepseek-v4.1-flash", "enabled": true},
    "sdd-test-implement-2": {"source": "preset:sdd-test-implement-2", "model": "opencode-go/qwen3.8-flash", "enabled": true},
    "sdd-implement-9r-1": {"source": "preset:sdd-implement-1", "model": "9router/ag/gemini-3.8-flash-high", "enabled": true},
    "sdd-implement-9r-2": {"source": "preset:sdd-implement-2", "model": "9router/ag/claude-sonnet-5.5-high", "enabled": true},
    "sdd-implement-9r-cheap-1": {"source": "preset:sdd-implement-1", "model": "9router/ag/gemini-3.8-flash-medium", "enabled": true},
    "sdd-implement-9r-cheap-2": {"source": "preset:sdd-implement-2", "model": "9router/ag/gemini-4-argon-high", "enabled": true},
    "sdd-implement-free-1": {"source": "preset:sdd-implement-1", "model": "opencode/glm-5.3-flash-free", "enabled": true},
    "sdd-implement-free-2": {"source": "preset:sdd-implement-2", "model": "opencode/beam-free", "enabled": true},
    "sdd-spec-9r": {"source": "agent:sdd-spec-9r", "model": "9router/ag/claude-opus-5.5-high", "enabled": true},
    "sdd-plan-9r": {"source": "agent:sdd-plan-9r", "model": "9router/ag/claude-opus-5.5-high", "enabled": true},
    "vision": {"source": "agent:sdd-visual-inspector", "model": "opencode-go/deepseek-v4.1-flash", "enabled": false}
  }
}`

// writeScaleConfig writes one strict fleet.json fixture and returns its path.
func writeScaleConfig(t *testing.T, dir, name string, instances int) string {
	t.Helper()
	var body string
	if name == "fleet18.json" {
		body = fleet18Golden
	} else {
		var b strings.Builder
		b.WriteString("{\n  \"schemaVersion\": 1,\n  \"fleet\": \"scale\",\n  \"instances\": {\n")
		for i := 1; i <= instances; i++ {
			sep := ","
			if i == instances {
				sep = ""
			}
			fmt.Fprintf(&b, "    \"instance-%03d\": {\"source\": \"preset:sdd-implement-1\", \"model\": \"opencode-go/scale-%03d\", \"enabled\": true}%s\n", i, i, sep)
		}
		b.WriteString("  }\n}")
		body = b.String()
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// runScaleOnce performs the offline parse pre-read + doctor labeling and
// returns the wall duration plus allocation delta.
func runScaleOnce(t *testing.T, configPath string, inspectDump string) (time.Duration, uint64) {
	t.Helper()
	models, err := DoctorModelsFromConfig(configPath)
	if err != nil {
		t.Fatalf("fixture config rejected: %v", err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	opts := DoctorOptions{Models: models}
	if inspectDump != "" {
		opts.CatalogDump = inspectDump
	}
	rep := RunDoctor(opts)
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	if len(rep.Entries) != len(models) {
		t.Fatalf("entry count %d != model count %d", len(rep.Entries), len(models))
	}
	return elapsed, after.TotalAlloc - before.TotalAlloc
}

func medianOf(d []time.Duration) time.Duration {
	x := append([]time.Duration(nil), d...)
	sort.Slice(x, func(i, j int) bool { return x[i] < x[j] })
	if len(x) == 0 {
		return 0
	}
	return x[len(x)/2]
}

// TestFleetDoctorScaleRecordsTimings builds the 18-name and 100-instance
// fixtures and RECORDS NFR-02 style numbers (median of five warm runs per
// size, default offline and dump-inspected). It asserts correctness only; the
// 2s design target is a plan decision, not something this test claims.
func TestFleetDoctorScaleRecordsTimings(t *testing.T) {
	dir := t.TempDir()
	c18 := writeScaleConfig(t, dir, "fleet18.json", 18)
	c100 := writeScaleConfig(t, dir, "fleet100.json", 100)

	// A 100-model local catalog dump for the inspected path.
	var models []string
	for i := 1; i <= 100; i++ {
		models = append(models, fmt.Sprintf("\"opencode-go/scale-%03d\"", i))
	}
	dump := filepath.Join(dir, "catalog100.json")
	if err := os.WriteFile(dump, []byte(`{"providers":[{"id":"opencode-go","auth":"api-key"}],"models":[`+strings.Join(models, ",")+`]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	record := func(label, cfg, insp string, wantModels int) {
		var runs []time.Duration
		var lastAlloc uint64
		for i := 0; i < 5; i++ {
			d, alloc := runScaleOnce(t, cfg, insp)
			runs = append(runs, d)
			lastAlloc = alloc
		}
		worst := runs[0]
		for _, r := range runs[1:] {
			if r > worst {
				worst = r
			}
		}
		t.Logf("NFR-02-RECORDED %s: median-of-5=%s worst=%s alloc=%d bytes (doctor+pre-read scope only)",
			label, medianOf(runs).Round(time.Microsecond), worst.Round(time.Microsecond), lastAlloc)
	}

	// Correctness assertions (not timings): sizes and full labeling. The
	// 18-instance golden fleet shares routes across instances (13 unique);
	// doctor deduplicates routes while RunDoctor labels every entry.
	for _, tc := range []struct {
		name, cfg string
		want      int
	}{
		{"18-name golden", c18, 13},
		{"100-instance synthetic", c100, 100},
	} {
		models, err := DoctorModelsFromConfig(tc.cfg)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(models) != tc.want {
			t.Fatalf("%s: got %d unique routes, want %d", tc.name, len(models), tc.want)
		}
		rep := RunDoctor(DoctorOptions{Models: models})
		if !rep.Offline || len(rep.Entries) != tc.want {
			t.Fatalf("%s: offline default mislabeled: %+v", tc.name, rep)
		}
		for _, e := range rep.Entries {
			if e.Label != DoctorNotChecked {
				t.Fatalf("%s: offline entry %+v", tc.name, e)
			}
		}
		record(tc.name+" offline", tc.cfg, "", tc.want)
		record(tc.name+" dump-inspected", tc.cfg, dump, tc.want)
	}
}

// runBuildPlanScaleOnce runs BuildPlan end-to-end and returns elapsed time, allocation delta, and plan.
func runBuildPlanScaleOnce(t *testing.T, configPath, repoPath, homePath string) (time.Duration, uint64, *Plan) {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	p, err := BuildPlan(configPath, repoPath, homePath, false)
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	return elapsed, after.TotalAlloc - before.TotalAlloc, p
}

// TestFleetBuildPlanScaleRecordsTimings builds the 18-name golden and 100-instance
// synthetic fixtures and executes real BuildPlan (parse, resolve, render, plan)
// recording median-of-5, worst duration, and memory allocation delta for NFR-02/AC-25.
func TestFleetBuildPlanScaleRecordsTimings(t *testing.T) {
	dir := t.TempDir()
	c18 := writeScaleConfig(t, dir, "fleet18.json", 18)
	c100 := writeScaleConfig(t, dir, "fleet100.json", 100)
	repo := filepath.Join("..", "..")

	runnerInfo := fmt.Sprintf("runner=%s/%s go=%s numCPU=%d",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU())

	recordPipeline := func(label, cfg string, wantOps int) {
		tempHome := filepath.Join(dir, "home-"+label)
		if err := os.MkdirAll(tempHome, 0755); err != nil {
			t.Fatal(err)
		}

		// Warm-up run
		_, _, warmPlan := runBuildPlanScaleOnce(t, cfg, repo, tempHome)
		if len(warmPlan.ProposedOps) != wantOps {
			t.Fatalf("%s: got %d proposed ops, want %d", label, len(warmPlan.ProposedOps), wantOps)
		}
		if !warmPlan.VerifyPlanID() {
			t.Fatalf("%s: plan ID integrity check failed", label)
		}
		if warmPlan.HasConflicts() {
			t.Fatalf("%s: unexpected conflicts in fresh plan", label)
		}

		var runs []time.Duration
		var allocs []uint64
		for i := 0; i < 5; i++ {
			d, alloc, p := runBuildPlanScaleOnce(t, cfg, repo, tempHome)
			runs = append(runs, d)
			allocs = append(allocs, alloc)
			if len(p.ProposedOps) != wantOps {
				t.Fatalf("%s run %d: got %d ops, want %d", label, i, len(p.ProposedOps), wantOps)
			}
		}

		worst := runs[0]
		for _, r := range runs[1:] {
			if r > worst {
				worst = r
			}
		}
		medianD := medianOf(runs)
		var avgAlloc uint64
		for _, a := range allocs {
			avgAlloc += a
		}
		avgAlloc /= uint64(len(allocs))

		t.Logf("NFR-02-RECORDED BuildPlan %s: median-of-5=%s worst=%s alloc=%d bytes (%s)",
			label, medianD.Round(time.Microsecond), worst.Round(time.Microsecond), avgAlloc, runnerInfo)
	}

	recordPipeline("18-name golden", c18, 18)
	recordPipeline("100-instance synthetic", c100, 100)
}
