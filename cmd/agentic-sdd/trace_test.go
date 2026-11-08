package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
	"agentic-sdd/internal/version"
)

// traceFixture builds the same one-skill repository fixture the other CLI
// tests use, with sentinel directory names that must never appear in a trace.
func traceFixture(t *testing.T) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "sentinel-secret-repo")
	home = filepath.Join(root, "sentinel-secret-home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	return repo, home
}

// runTraced executes one command through the real runEnv seam with stdin
// that never supplies an answer.
func runTraced(t *testing.T, tracer *Tracer, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runEnv(args, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80, Trace: tracer})
	return code, stdout.String(), stderr.String()
}

// decodeTrace parses every trace line as JSON; a non-JSON line fails the test.
func decodeTrace(t *testing.T, raw string) []map[string]any {
	t.Helper()
	trimmed := strings.TrimRight(raw, "\n")
	if trimmed == "" {
		t.Fatal("trace is empty")
	}
	var events []map[string]any
	for i, line := range strings.Split(trimmed, "\n") {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("trace line %d is not valid JSON: %q: %v", i+1, line, err)
		}
		events = append(events, event)
	}
	return events
}

// checkTraceRun asserts the documented common fields and the exact ordered
// event names of one run.
func checkTraceRun(t *testing.T, raw string, want []string) []map[string]any {
	t.Helper()
	events := decodeTrace(t, raw)
	if len(events) != len(want) {
		t.Fatalf("got %d events %v, want %d %v", len(events), raw, len(want), want)
	}
	runID := ""
	for i, event := range events {
		if _, ok := event["msg"]; ok {
			t.Fatalf("event %d kept the raw msg key: %v", i+1, event)
		}
		if _, ok := event["level"]; ok {
			t.Fatalf("event %d kept the level field: %v", i+1, event)
		}
		if event["schema"] != float64(1) {
			t.Fatalf("event %d schema = %v, want 1", i+1, event["schema"])
		}
		id, ok := event["run"].(string)
		if !ok || len(id) != 16 {
			t.Fatalf("event %d run = %v, want 16 hex chars", i+1, event["run"])
		}
		if _, err := hex.DecodeString(id); err != nil {
			t.Fatalf("event %d run id %q is not hex: %v", i+1, id, err)
		}
		if runID == "" {
			runID = id
		} else if id != runID {
			t.Fatalf("event %d run id %q differs from %q", i+1, id, runID)
		}
		if event["seq"] != float64(i+1) {
			t.Fatalf("event %d seq = %v, want %d", i+1, event["seq"], i+1)
		}
		if event["event"] != want[i] {
			t.Fatalf("event %d = %v, want %q", i+1, event["event"], want[i])
		}
	}
	return events
}

func assertNoPathLeak(t *testing.T, raw, repo, home string) {
	t.Helper()
	for _, secret := range []string{repo, home, filepath.Dir(repo), "sentinel-secret-home", "sentinel-secret-repo"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("trace leaked %q:\n%s", secret, raw)
		}
	}
	if strings.Contains(raw, "\"flags\":[]") {
		// Flags were passed, so the array must not be empty.
		t.Fatalf("trace recorded no flag names:\n%s", raw)
	}
}

// TestTraceCommandAndFlagNames pins the privacy rule: only the subcommand and
// bare flag names are recorded, never flag values or positional arguments.
func TestTraceCommandAndFlagNames(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		cmd   string
		flags []string
	}{
		{"empty", nil, "preview", nil},
		{"bare preview", []string{"preview"}, "preview", nil},
		{"version flag", []string{"--version"}, "version", []string{"version"}},
		{"short help", []string{"-h"}, "help", []string{"h"}},
		{"legacy apply", []string{"--apply"}, "apply", []string{"apply"}},
		{"equals value", []string{"--home=/secret/path", "apply"}, "apply", []string{"home"}},
		{"separate value", []string{"--home", "/secret", "apply"}, "apply", []string{"home"}},
		{"restore id", []string{"restore", "apply", "20260101T000000.000000000Z", "--repo", "/secret"}, "restore", []string{"repo"}},
		{"unknown", []string{"wat"}, "unknown", nil},
		{"repo equals", []string{"--repo=/secret", "preview"}, "preview", []string{"repo"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := commandName(tc.args); got != tc.cmd {
				t.Fatalf("commandName(%v) = %q, want %q", tc.args, got, tc.cmd)
			}
			got := flagNames(tc.args)
			if len(got) != len(tc.flags) {
				t.Fatalf("flagNames(%v) = %v, want %v", tc.args, got, tc.flags)
			}
			for i := range got {
				if got[i] != tc.flags[i] {
					t.Fatalf("flagNames(%v) = %v, want %v", tc.args, got, tc.flags)
				}
				if strings.Contains(got[i], "/") || strings.Contains(got[i], "=") {
					t.Fatalf("flagNames(%v) recorded a value: %q", tc.args, got[i])
				}
			}
		})
	}
}

func TestTraceJSONLines(t *testing.T) {
	repo, home := traceFixture(t)

	t.Run("preview in-memory", func(t *testing.T) {
		var traceBuf bytes.Buffer
		tracer := newTracer(&traceBuf, nil, true)
		code, stdout, stderr := runTraced(t, tracer, "preview", "--home", home, "--repo", repo)
		if code != 0 {
			t.Fatalf("preview exit %d: stdout %q stderr %q", code, stdout, stderr)
		}
		events := checkTraceRun(t, traceBuf.String(), []string{"run.start", "env.resolved", "run.end"})

		start := events[0]
		if start["command"] != "preview" {
			t.Fatalf("run.start command = %v", start["command"])
		}
		if start["version"] != version.Version {
			t.Fatalf("run.start version = %v, want %q", start["version"], version.Version)
		}
		flags, ok := start["flags"].([]any)
		if !ok || len(flags) != 2 || flags[0] != "home" || flags[1] != "repo" {
			t.Fatalf("run.start flags = %v, want [home repo]", start["flags"])
		}

		resolved := events[1]
		for _, field := range []string{"interactive", "stdout_tty", "stderr_tty", "color", "ascii"} {
			if resolved[field] != false {
				t.Fatalf("env.resolved %s = %v, want false", field, resolved[field])
			}
		}
		if resolved["width"] != float64(80) {
			t.Fatalf("env.resolved width = %v, want 80", resolved["width"])
		}
		if events[2]["exit"] != float64(0) {
			t.Fatalf("run.end exit = %v, want 0", events[2]["exit"])
		}
		if _, ok := events[0]["time"]; ok {
			t.Fatalf("in-memory tracer kept time despite dropTime: %v", events[0])
		}
		assertNoPathLeak(t, traceBuf.String(), repo, home)
	})

	t.Run("apply via openTrace", func(t *testing.T) {
		tracePath := filepath.Join(t.TempDir(), "journey.jsonl")
		tracer, err := openTrace(tracePath)
		if err != nil || tracer == nil {
			t.Fatalf("openTrace: %v, %v", tracer, err)
		}
		code, stdout, stderr := runTraced(t, tracer, "apply", "--home", home, "--repo", repo)
		if code != 0 {
			t.Fatalf("apply exit %d: stdout %q stderr %q", code, stdout, stderr)
		}
		raw, err := os.ReadFile(tracePath)
		if err != nil {
			t.Fatal(err)
		}
		events := checkTraceRun(t, string(raw), []string{"run.start", "env.resolved", "op.result", "run.end"})

		if events[0]["command"] != "apply" {
			t.Fatalf("run.start command = %v", events[0]["command"])
		}
		if _, ok := events[0]["time"]; !ok {
			t.Fatalf("production tracer dropped time: %v", events[0])
		}

		result := events[2]
		if result["op"] != "apply" {
			t.Fatalf("op.result op = %v, want apply", result["op"])
		}
		if result["exit"] != float64(0) {
			t.Fatalf("op.result exit = %v, want 0", result["exit"])
		}
		id, ok := result["backup_id"].(string)
		if !ok || id == "" {
			t.Fatalf("op.result backup_id = %v", result["backup_id"])
		}
		if strings.Contains(id, "/") {
			t.Fatalf("backup_id is a path, not a basename: %q", id)
		}
		if err := skillsync.ValidateBackupID(id); err != nil {
			t.Fatalf("backup_id %q fails ValidateBackupID: %v", id, err)
		}
		if events[3]["exit"] != float64(0) {
			t.Fatalf("run.end exit = %v, want 0", events[3]["exit"])
		}
		assertNoPathLeak(t, string(raw), repo, home)

		flags := events[0]["flags"].([]any)
		for _, flag := range flags {
			name, _ := flag.(string)
			if strings.ContainsAny(name, "/=") {
				t.Fatalf("flag name %q carries a value or path", name)
			}
		}
		info, err := os.Stat(tracePath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("trace file mode = %v, want 0600", info.Mode().Perm())
		}
	})
}

// TestTraceGuideFlow extends Check C9 to the guide: the documented screens,
// prompts, plan summary, confirmation and operation result are recorded in
// order, with no free text and no filesystem path.
func TestTraceGuideFlow(t *testing.T) {
	repo, home := traceFixture(t)
	var traceBuf bytes.Buffer
	tracer := newTracer(&traceBuf, nil, true)
	var stdout, stderr bytes.Buffer
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80, Trace: tracer}
	code := runEnv([]string{"guide", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader("1\nn\ny\n"), env)
	if code != 0 {
		t.Fatalf("guide exit %d: stdout %q stderr %q", code, stdout.String(), stderr.String())
	}

	events := checkTraceRun(t, traceBuf.String(), []string{
		"run.start", "env.resolved",
		"screen.shown", "prompt.shown", "prompt.answered",
		"screen.shown", "prompt.shown", "prompt.answered", "confirm",
		"screen.shown", "plan.summary",
		"prompt.shown", "prompt.answered", "confirm",
		"op.result",
		"screen.shown",
		"run.end",
	})

	screens := []struct {
		index  int
		screen string
	}{
		{2, "menu"},
		{5, "install.explain"},
		{9, "install.preview"},
		{15, "install.done"},
	}
	for _, want := range screens {
		if events[want.index]["screen"] != want.screen {
			t.Fatalf("event %d screen = %v, want %q", want.index+1, events[want.index]["screen"], want.screen)
		}
	}

	if events[3]["prompt"] != "menu" || events[3]["default"] != "1" {
		t.Fatalf("menu prompt.shown = %v", events[3])
	}
	if events[4]["kind"] != "choice" || events[4]["choice"] != "1" {
		t.Fatalf("menu prompt.answered = %v", events[4])
	}

	if events[6]["prompt"] != "install.with_opencode" || events[6]["default"] != "n" {
		t.Fatalf("pack prompt.shown = %v", events[6])
	}
	if events[7]["kind"] != "choice" || events[7]["choice"] != "n" {
		t.Fatalf("pack prompt.answered = %v", events[7])
	}
	if events[8]["prompt"] != "install.with_opencode" || events[8]["decision"] != "no" {
		t.Fatalf("pack confirm = %v", events[8])
	}

	summary := events[10]
	if summary["install"] != float64(4) || summary["replace"] != float64(0) || summary["up_to_date"] != float64(0) {
		t.Fatalf("plan.summary = %v, want 4 installs", summary)
	}

	if events[11]["prompt"] != "install.confirm" || events[11]["default"] != "n" {
		t.Fatalf("install confirm prompt.shown = %v", events[11])
	}
	if events[12]["kind"] != "choice" || events[12]["choice"] != "y" {
		t.Fatalf("install confirm prompt.answered = %v", events[12])
	}
	if events[13]["prompt"] != "install.confirm" || events[13]["decision"] != "yes" {
		t.Fatalf("install confirm = %v", events[13])
	}

	result := events[14]
	if result["op"] != "apply" || result["exit"] != float64(0) {
		t.Fatalf("op.result = %v, want apply exit 0", result)
	}
	id, ok := result["backup_id"].(string)
	if !ok || id == "" || strings.Contains(id, "/") {
		t.Fatalf("op.result backup_id = %v", result["backup_id"])
	}
	if err := skillsync.ValidateBackupID(id); err != nil {
		t.Fatalf("backup_id %q fails ValidateBackupID: %v", id, err)
	}
	if events[16]["exit"] != float64(0) {
		t.Fatalf("run.end exit = %v, want 0", events[16]["exit"])
	}

	raw := traceBuf.String()
	for _, freeText := range []string{"Add the OpenCode", "Prefer Free", "Install these", "agentic-sdd-router", "sentinel-secret"} {
		if strings.Contains(raw, freeText) {
			t.Fatalf("trace recorded free text %q:\n%s", freeText, raw)
		}
	}
	assertNoPathLeak(t, raw, repo, home)
}

func TestTraceOpenTraceNormalizesModeAndAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")

	tracer, err := openTrace(path)
	if err != nil || tracer == nil {
		t.Fatalf("openTrace new file: %v, %v", tracer, err)
	}
	tracer.Event("run.start", "command", "preview")
	if err := tracer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("new trace file mode = %v, want 0600", got)
	}

	// A pre-existing looser file must be normalized on open, not left loose.
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	tracer, err = openTrace(path)
	if err != nil || tracer == nil {
		t.Fatalf("openTrace existing file: %v, %v", tracer, err)
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("reopened trace file mode = %v, want 0600", got)
	}
	tracer.Event("run.start", "command", "apply")
	if err := tracer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	events := decodeTrace(t, string(raw))
	if len(events) != 2 {
		t.Fatalf("expected two appended events, got %d: %s", len(events), raw)
	}
	for i, event := range events {
		if event["seq"] != float64(1) {
			t.Fatalf("event %d seq = %v, want 1 per tracer", i+1, event["seq"])
		}
	}
}

func TestTraceDisabledIsNoOp(t *testing.T) {
	for _, path := range []string{"", "   ", "\n\t"} {
		tracer, err := openTrace(path)
		if err != nil {
			t.Fatalf("openTrace(%q) error: %v", path, err)
		}
		if tracer != nil {
			t.Fatalf("openTrace(%q) = %v, want nil", path, tracer)
		}
	}

	var nilTracer *Tracer
	nilTracer.Event("run.start", "command", "preview")
	if err := nilTracer.Close(); err != nil {
		t.Fatalf("nil Close: %v", err)
	}

	repo, home := traceFixture(t)
	var plainOut, plainErr, envOut, envErr bytes.Buffer
	plainCode := run([]string{"preview", "--repo", repo, "--home", home}, &plainOut, &plainErr, strings.NewReader(""), false)
	envCode := runEnv([]string{"preview", "--repo", repo, "--home", home}, &envOut, &envErr, strings.NewReader(""), &Env{Width: 80})
	if plainCode != envCode || plainOut.String() != envOut.String() || plainErr.String() != envErr.String() {
		t.Fatalf("disabled trace changed behavior: codes %d/%d, stdout %q/%q, stderr %q/%q",
			plainCode, envCode, plainOut.String(), envOut.String(), plainErr.String(), envErr.String())
	}
}

func TestTraceOpenFailuresIsolateCommand(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.jsonl")
	if err := os.WriteFile(realFile, []byte("keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.jsonl")
	if err := os.Symlink(realFile, link); err != nil {
		t.Fatal(err)
	}
	dirTarget := filepath.Join(dir, "dir-target.jsonl")
	if err := os.Mkdir(dirTarget, 0700); err != nil {
		t.Fatal(err)
	}
	missingParent := filepath.Join(dir, "missing", "trace.jsonl")

	cases := []struct {
		name string
		path string
	}{
		{"symlink", link},
		{"directory", dirTarget},
		{"missing parent", missingParent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tracer, err := openTrace(tc.path)
			if err == nil {
				t.Fatalf("openTrace(%q) succeeded with tracer %v", tc.name, tracer)
			}
			if tracer != nil {
				t.Fatalf("openTrace(%q) returned a tracer on failure", tc.name)
			}
			if strings.Contains(err.Error(), tc.path) {
				t.Fatalf("error %q echoes the trace path", err)
			}
		})
	}

	if data, err := os.ReadFile(realFile); err != nil || string(data) != "keep\n" {
		t.Fatalf("symlink target was modified: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Dir(missingParent)); !os.IsNotExist(err) {
		t.Fatalf("openTrace created a parent directory: %v", err)
	}

	// Simulate main() after a failed open: tracing stays off, and the command
	// result and stdout are identical to the untraced run.
	repo, home := traceFixture(t)
	var baseOut, baseErr bytes.Buffer
	baseCode := run([]string{"preview", "--repo", repo, "--home", home}, &baseOut, &baseErr, strings.NewReader(""), false)

	env := &Env{Width: 80}
	if _, err := openTrace(link); err == nil {
		t.Fatalf("expected the symlink target to be refused")
	} else {
		var stdout, stderr bytes.Buffer
		fmt.Fprintf(&stderr, "warning: trace disabled: %v\n", err)
		code := runEnv([]string{"preview", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), env)
		if code != baseCode {
			t.Fatalf("exit code changed with a failed trace: %d vs %d", code, baseCode)
		}
		if stdout.String() != baseOut.String() {
			t.Fatalf("stdout changed with a failed trace:\n%q\nvs\n%q", stdout.String(), baseOut.String())
		}
		if !strings.HasPrefix(stderr.String(), "warning: trace disabled: ") {
			t.Fatalf("missing warning line: %q", stderr.String())
		}
		if strings.Contains(stderr.String(), link) {
			t.Fatalf("warning echoes the trace path: %q", stderr.String())
		}
	}
}
