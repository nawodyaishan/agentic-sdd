package fleet

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"agentic-sdd/internal/skillsync"
)

func mustParseFleetConfig(t *testing.T, body string) *Config {
	t.Helper()
	cfg, err := ParseConfig(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parse fixture config: %v", err)
	}
	return cfg
}

func TestFleetEncodeConfigDeterministic(t *testing.T) {
	cfg := mustParseFleetConfig(t, fleet18Golden)
	first, err := EncodeConfig(cfg)
	if err != nil {
		t.Fatalf("EncodeConfig: %v", err)
	}
	second, err := EncodeConfig(cfg)
	if err != nil {
		t.Fatalf("EncodeConfig repeat: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("EncodeConfig is not deterministic")
	}
	if len(first) == 0 || first[len(first)-1] != '\n' {
		t.Fatalf("expected exactly one trailing newline, got %q", first)
	}
	if bytes.Count(first, []byte("\n")) < 2 {
		t.Fatalf("expected indented output, got %q", first)
	}

	// Round-trip through the strict parser must be lossless.
	reparsed, err := ParseConfig(bytes.NewReader(first))
	if err != nil {
		t.Fatalf("reparse encoded config: %v", err)
	}
	if !reflect.DeepEqual(cfg, reparsed) {
		t.Fatalf("encoded config did not round-trip\nwant: %+v\ngot:  %+v", cfg, reparsed)
	}

	// Field order is the struct order; instance keys are sorted.
	text := string(first)
	last := -1
	for _, field := range []string{`"schemaVersion"`, `"fleet"`, `"preset"`, `"instances"`} {
		idx := strings.Index(text, field)
		if idx <= last {
			t.Fatalf("field %s is missing or out of order in:\n%s", field, text)
		}
		last = idx
	}
	if strings.Index(text, `"sdd-architecture-review-9r"`) > strings.Index(text, `"vision"`) {
		t.Fatalf("instance keys are not sorted:\n%s", text)
	}

	if _, err := EncodeConfig(nil); err == nil {
		t.Fatal("expected EncodeConfig(nil) to fail")
	}
}

func TestFleetPreviewSource(t *testing.T) {
	t.Run("update, noop, create and refusals", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "fleet.json")
		before := mustParseFleetConfig(t, singleConfigBody("9router/ag/gemini-3.8-flash-high"))
		after := mustParseFleetConfig(t, singleConfigBody("9router/ag/claude-sonnet-5.5-high"))
		oldBytes, err := EncodeConfig(before)
		if err != nil {
			t.Fatal(err)
		}
		newBytes, err := EncodeConfig(after)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, oldBytes, 0600); err != nil {
			t.Fatal(err)
		}
		snapshot := snapshotTree(t, dir)

		preview, err := PreviewSource(path, newBytes)
		if err != nil {
			t.Fatalf("PreviewSource update: %v", err)
		}
		if preview.Op != FileOpUpdate || preview.Removed || preview.Conflict {
			t.Fatalf("unexpected preview op: %+v", preview)
		}
		if !bytes.Equal(preview.OldBytes, oldBytes) || preview.OldDigest != digestOf(oldBytes) {
			t.Fatalf("old side mismatch: %+v", preview)
		}
		if preview.NewDigest != digestOf(newBytes) || !bytes.Equal(preview.NewBytes, newBytes) {
			t.Fatalf("new side mismatch: %+v", preview)
		}
		if changes := diffKindCounts(LineDiff(preview.OldBytes, preview.NewBytes)); changes[DiffAdd] != 1 || changes[DiffDel] != 1 {
			t.Fatalf("expected a one-line source change, got %+v", changes)
		}

		noop, err := PreviewSource(path, oldBytes)
		if err != nil {
			t.Fatalf("PreviewSource noop: %v", err)
		}
		if noop.Op != FileOpNoOp || noop.OldDigest != noop.NewDigest {
			t.Fatalf("expected noop, got %+v", noop)
		}

		created, err := PreviewSource(filepath.Join(dir, "new.json"), newBytes)
		if err != nil {
			t.Fatalf("PreviewSource create: %v", err)
		}
		if created.Op != FileOpCreate || created.OldDigest != "" || created.OldBytes != nil {
			t.Fatalf("expected create, got %+v", created)
		}

		// Refusals: directory, oversize, invalid JSON, unknown source.
		if _, err := PreviewSource(dir, newBytes); !errors.Is(err, ErrUnsafeTarget) {
			t.Fatalf("expected ErrUnsafeTarget for a directory, got %v", err)
		}
		if _, err := PreviewSource(path, make([]byte, MaxConfigBytes+1)); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("expected ErrInvalidConfig for oversize input, got %v", err)
		}
		if _, err := PreviewSource(path, []byte(`{"schemaVersion":1,"instances":`)); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("expected ErrInvalidConfig for invalid JSON, got %v", err)
		}
		unknownSource := []byte(`{"schemaVersion":1,"instances":{"x":{"source":"agent:nope"}}}`)
		if _, err := PreviewSource(path, unknownSource); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("expected ErrInvalidConfig for an unknown source, got %v", err)
		}

		assertSnapshotUnchanged(t, snapshot, snapshotTree(t, dir), "PreviewSource")
	})

	t.Run("symlinked source is refused", func(t *testing.T) {
		dir := t.TempDir()
		real := filepath.Join(dir, "real.json")
		cfg := mustParseFleetConfig(t, singleConfigBody("9router/ag/gemini-3.8-flash-high"))
		data, err := EncodeConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(real, data, 0600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link.json")
		if err := os.Symlink(real, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := PreviewSource(link, data); err == nil {
			t.Fatal("expected a symlinked source to be refused")
		}
	})
}

func TestFleetLineDiff(t *testing.T) {
	t.Run("CRLF and LF compare equal", func(t *testing.T) {
		diff := LineDiff([]byte("alpha\r\nbeta\r\n"), []byte("alpha\nbeta\n"))
		if len(diff) != 2 {
			t.Fatalf("unexpected diff: %+v", diff)
		}
		for _, line := range diff {
			if line.Kind != DiffEqual {
				t.Fatalf("expected equal lines, got %+v", diff)
			}
		}
	})

	t.Run("Unicode text is preserved", func(t *testing.T) {
		diff := LineDiff([]byte("café\n"), []byte("café\nnaïve ✓\n"))
		if len(diff) != 2 || diff[1].Kind != DiffAdd || diff[1].Text != "naïve ✓" || diff[1].NewLine != 2 {
			t.Fatalf("unexpected Unicode diff: %+v", diff)
		}
	})

	t.Run("long lines are clipped at a rune boundary", func(t *testing.T) {
		long := strings.Repeat("é", MaxDiffLineBytes)
		diff := LineDiff([]byte("short\n"), []byte(long+"\n"))
		if len(diff) != 2 {
			t.Fatalf("unexpected diff: %+v", diff)
		}
		if !utf8.ValidString(diff[1].Text) {
			t.Fatalf("clipped line is not valid UTF-8")
		}
		if len(diff[1].Text) > MaxDiffLineBytes+len("…") {
			t.Fatalf("clipped line is %d bytes", len(diff[1].Text))
		}
		if !strings.HasSuffix(diff[1].Text, "…") {
			t.Fatalf("expected a truncation marker, got %q", diff[1].Text)
		}
	})

	t.Run("more than one thousand lines are fully represented", func(t *testing.T) {
		oldLines := make([]string, 1500)
		newLines := make([]string, 1500)
		for i := range oldLines {
			oldLines[i] = fmt.Sprintf("line %04d", i)
			newLines[i] = oldLines[i]
		}
		newLines[750] = "line 0750 changed"
		diff := LineDiff([]byte(strings.Join(oldLines, "\n")+"\n"), []byte(strings.Join(newLines, "\n")+"\n"))
		if len(diff) != 1501 {
			t.Fatalf("expected all 1501 rows, got %d", len(diff))
		}
		counts := diffKindCounts(diff)
		if counts[DiffEqual] != 1499 || counts[DiffDel] != 1 || counts[DiffAdd] != 1 {
			t.Fatalf("unexpected kind counts: %+v", counts)
		}
		var added DiffLine
		for _, line := range diff {
			if line.Kind == DiffAdd {
				added = line
			}
		}
		if added.Text != "line 0750 changed" || added.NewLine != 751 || added.OldLine != 0 {
			t.Fatalf("unexpected added row: %+v", added)
		}
	})

	t.Run("output is bounded for large unrelated inputs", func(t *testing.T) {
		oldLines := make([]string, 30000)
		newLines := make([]string, 30000)
		for i := range oldLines {
			oldLines[i] = fmt.Sprintf("old-%05d", i)
			newLines[i] = fmt.Sprintf("new-%05d", i)
		}
		diff := LineDiff([]byte(strings.Join(oldLines, "\n")), []byte(strings.Join(newLines, "\n")))
		if len(diff) != MaxDiffLines+1 {
			t.Fatalf("expected a bounded diff with one marker, got %d rows", len(diff))
		}
		last := diff[len(diff)-1]
		if last.Kind != DiffOmitted || !strings.Contains(last.Text, "omitted") {
			t.Fatalf("expected an omitted marker, got %+v", last)
		}
	})

	t.Run("empty and one-sided inputs", func(t *testing.T) {
		if diff := LineDiff(nil, nil); len(diff) != 0 {
			t.Fatalf("expected no rows, got %+v", diff)
		}
		diff := LineDiff(nil, []byte("only\nnew\n"))
		if len(diff) != 2 || diff[0].Kind != DiffAdd || diff[1].NewLine != 2 {
			t.Fatalf("unexpected add-only diff: %+v", diff)
		}
		diff = LineDiff([]byte("only\nold\n"), nil)
		if len(diff) != 2 || diff[0].Kind != DiffDel || diff[1].OldLine != 2 {
			t.Fatalf("unexpected del-only diff: %+v", diff)
		}
	})
}

func TestFleetPlanFiles(t *testing.T) {
	home := newFleetHome(t)
	configDir := t.TempDir()
	configPath := writeFleetConfig(t, configDir,
		`"keep":{"source":"agent:sdd-lightweight","model":"9router/ag/gemini-3.8-flash-high"},`+
			`"keep2":{"source":"agent:sdd-orchestrator"},`+
			`"gone":{"source":"preset:sdd-implement-1","model":"opencode-go/deepseek-v4.1-flash"},`+
			`"drift":{"source":"agent:sdd-code-review-9r","model":"9router/ag/gemini-3.8-flash-high"}`)
	applyPlan(t, buildPlanFor(t, configPath, home, false), home)

	targetRoot := TargetRootForHome(home)
	driftTarget := filepath.Join(targetRoot, "drift.md")
	driftData, err := os.ReadFile(driftTarget)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(driftTarget, append(driftData, []byte("\nmanual edit\n")...), 0600); err != nil {
		t.Fatal(err)
	}

	writeFleetConfig(t, configDir,
		`"keep":{"source":"agent:sdd-lightweight","model":"9router/ag/claude-sonnet-5.5-high"},`+
			`"keep2":{"source":"agent:sdd-orchestrator"},`+
			`"gone":{"source":"preset:sdd-implement-1","enabled":false},`+
			`"drift":{"source":"agent:sdd-code-review-9r","model":"9router/ag/claude-sonnet-5.5-high"},`+
			`"new":{"source":"preset:sdd-implement-2","model":"opencode-go/qwen3.8-flash"}`)
	plan := buildPlanFor(t, configPath, home, false)
	planID := plan.PlanID

	snapshot := snapshotTree(t, home)
	previews, err := PlanFiles(plan)
	if err != nil {
		t.Fatalf("PlanFiles: %v", err)
	}
	assertSnapshotUnchanged(t, snapshot, snapshotTree(t, home), "PlanFiles")
	if plan.PlanID != planID || !plan.VerifyPlanID() {
		t.Fatal("PlanFiles changed the plan identity")
	}

	var order []string
	for _, preview := range previews {
		order = append(order, preview.InstanceID+":"+preview.Op)
	}
	want := []string{"gone:remove", "drift:conflict", "keep:update", "new:create", "keep2:noop"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected preview order\nwant: %v\ngot:  %v", want, order)
	}

	byID := map[string]FilePreview{}
	for _, preview := range previews {
		byID[preview.InstanceID] = preview
	}

	gone := byID["gone"]
	if !gone.Removed || gone.NewBytes != nil || gone.OldDigest == "" || gone.Op != FileOpRemove {
		t.Fatalf("unexpected removal preview: %+v", gone)
	}
	drift := byID["drift"]
	if !drift.Conflict || drift.Reason == "" || drift.NewDigest == "" || drift.Op != FileOpConflict {
		t.Fatalf("unexpected conflict preview: %+v", drift)
	}
	if got := string(drift.OldBytes); !strings.Contains(got, "manual edit") {
		t.Fatalf("conflict preview must show the current modified bytes: %q", got)
	}

	keep := byID["keep"]
	if keep.Op != FileOpUpdate || keep.Path != filepath.Join(targetRoot, "keep.md") {
		t.Fatalf("unexpected update preview: %+v", keep)
	}
	if !strings.Contains(string(keep.OldBytes), "gemini-3.8-flash-high") {
		t.Fatalf("old bytes do not show the installed model: %q", keep.OldBytes)
	}
	if !strings.Contains(string(keep.NewBytes), "claude-sonnet-5.5-high") {
		t.Fatalf("new bytes do not show the desired model: %q", keep.NewBytes)
	}
	if keep.NewDigest != digestOf(keep.NewBytes) || keep.OldDigest != digestOf(keep.OldBytes) {
		t.Fatalf("preview digests do not match their bytes: %+v", keep)
	}
	// Model-only preservation: the rendered body and permissions are unchanged.
	pack, err := skillsync.OpenCodePackFS("")
	if err != nil {
		t.Fatal(err)
	}
	source, err := fs.ReadFile(pack, "agents/sdd-lightweight.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyModelOnly(source, keep.NewBytes); err != nil {
		t.Fatalf("desired bytes are not a model-only render: %v", err)
	}
	if changes := diffKindCounts(LineDiff(keep.OldBytes, keep.NewBytes)); changes[DiffAdd] != 1 || changes[DiffDel] != 1 {
		t.Fatalf("expected only the model line to change, got %+v", changes)
	}

	created := byID["new"]
	if created.Op != FileOpCreate || created.OldBytes != nil || created.OldDigest != "" || created.NewDigest == "" {
		t.Fatalf("unexpected create preview: %+v", created)
	}
	noop := byID["keep2"]
	if noop.Op != FileOpNoOp || noop.OldDigest != noop.NewDigest || noop.OldDigest == "" {
		t.Fatalf("unexpected noop preview: %+v", noop)
	}

	// A stale config is refused rather than drawn against different state.
	writeFleetConfig(t, configDir, singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	if _, err := PlanFiles(plan); !errors.Is(err, ErrStalePlan) {
		t.Fatalf("expected ErrStalePlan for a changed config, got %v", err)
	}

	if _, err := PlanFiles(nil); err == nil {
		t.Fatal("expected PlanFiles(nil) to fail")
	}
	emptyConfig := writeFleetConfig(t, t.TempDir(), "")
	emptyPlan := buildPlanFor(t, emptyConfig, newFleetHome(t), false)
	emptyPreviews, err := PlanFiles(emptyPlan)
	if err != nil {
		t.Fatalf("PlanFiles empty plan: %v", err)
	}
	if len(emptyPreviews) != 0 {
		t.Fatalf("expected no previews for an empty plan, got %+v", emptyPreviews)
	}
}

func singleConfigBody(model string) string {
	return `{"schemaVersion":1,"fleet":"test","instances":{"sdd-lightweight":{"source":"agent:sdd-lightweight","model":"` + model + `"}}}`
}

func TestFleetPlanFilesLongUnicodePath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "hôme-"+strings.Repeat("naïve-", 24)+"user")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	plan := buildPlanFor(t, configPath, home, false)
	previews, err := PlanFiles(plan)
	if err != nil {
		t.Fatalf("PlanFiles long Unicode path: %v", err)
	}
	if len(previews) != 1 {
		t.Fatalf("expected one preview, got %+v", previews)
	}
	wantPath := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
	if previews[0].Path != wantPath || previews[0].Op != FileOpCreate || previews[0].NewDigest == "" {
		t.Fatalf("unexpected long-path preview: %+v", previews[0])
	}
	if counts := diffKindCounts(LineDiff(previews[0].OldBytes, previews[0].NewBytes)); counts[DiffAdd] == 0 {
		t.Fatalf("expected the full create diff, got %+v", counts)
	}
}

func TestFleetPlanFilesSymlinkRefusal(t *testing.T) {
	home := newFleetHome(t)
	targetRoot := TargetRootForHome(home)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
	plan := buildPlanFor(t, configPath, home, false)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(targetRoot, "sdd-lightweight.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := PlanFiles(plan); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("expected ErrUnsafeTarget for a symlinked target, got %v", err)
	}
	if got := readString(t, outside); got != "outside content\n" {
		t.Fatalf("PlanFiles followed the symlink: %q", got)
	}
}

func diffKindCounts(lines []DiffLine) map[DiffKind]int {
	counts := map[DiffKind]int{}
	for _, line := range lines {
		counts[line.Kind]++
	}
	return counts
}
