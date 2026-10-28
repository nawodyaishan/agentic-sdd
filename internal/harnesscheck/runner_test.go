package harnesscheck

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func optionsFor(t *testing.T, c Contract) Options {
	t.Helper()
	if !processSupported {
		t.Skip("check execution unsupported on this host")
	}
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(c)
	contract := filepath.Join(root, "checks.json")
	if err := os.WriteFile(contract, data, 0600); err != nil {
		t.Fatal(err)
	}
	return Options{Workspace: workspace, ContractPath: contract, Output: filepath.Join(root, "evidence"), Scope: "required", Deadline: 10 * time.Second}
}

func helperStep(t *testing.T, action string) Step {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Step{Argv: []string{exe, "-test.run=^TestCheckHelper$", "--", action}, Env: map[string]string{"SDD_CHECK_HELPER": "1", "GORACE": "atexit_sleep_ms=0"}}
}

func TestCheckHelper(t *testing.T) {
	if os.Getenv("SDD_CHECK_HELPER") != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "pass":
		_, _ = os.Stdout.WriteString("checked\n")
	case "fail":
		os.Exit(7)
	case "write-sum":
		_ = os.WriteFile("go.sum", []byte("new untracked module file\n"), 0600)
	case "write-sentinel":
		_ = os.WriteFile("sentinel", []byte("unexpected"), 0600)
	case "flood":
		_, _ = os.Stdout.WriteString(strings.Repeat("x", 8192))
	case "sleep":
		time.Sleep(30 * time.Second)
	case "sleep-short":
		time.Sleep(600 * time.Millisecond)
	case "literal":
		_, _ = os.Stdout.WriteString(os.Args[len(os.Args)-2])
	case "unlink-log":
		_ = os.Remove("../evidence/fixture.log")
	case "unlink-result":
		_ = os.Remove("../evidence/result.json")
	default:
		os.Exit(11)
	}
	os.Exit(0)
}

func TestPassAndPrivateEvidence(t *testing.T) {
	c := fixtureContract()
	c.Checks[0].Steps = []Step{helperStep(t, "pass")}
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "completed" || ExitCode(r, err) != 0 {
		t.Fatalf("%+v %v", r, err)
	}
	check := r.Checks[0]
	if check.Status != "passed" || check.ExitCode == nil || *check.ExitCode != 0 || check.StartedAt == nil || check.EndedAt == nil {
		t.Fatalf("%+v", check)
	}
	data, err := os.ReadFile(filepath.Join(o.Output, "fixture.log"))
	if err != nil || !strings.Contains(string(data), "checked") {
		t.Fatalf("log: %q %v", data, err)
	}
	encoded, err := os.ReadFile(filepath.Join(o.Output, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted Result
	if err := json.Unmarshal(encoded, &persisted); err != nil || !persisted.EndedAt.Equal(r.EndedAt) {
		t.Fatalf("persisted/returned result mismatch: %v", err)
	}
	for _, path := range []string{o.Output, filepath.Join(o.Output, "fixture.log"), filepath.Join(o.Output, "result.json")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatalf("not private: %s %v", path, err)
		}
	}
	if _, err := Run(context.Background(), o); err == nil {
		t.Fatal("overwrote existing evidence")
	}
}

func TestFailureStopsLaterStepsAndChecks(t *testing.T) {
	c := fixtureContract()
	c.Checks[0].Steps = []Step{helperStep(t, "fail"), helperStep(t, "write-sentinel")}
	later := c.Checks[0]
	later.ID, later.Evidence = "later", "later.log"
	later.Steps = []Step{helperStep(t, "write-sentinel")}
	c.Checks = append(c.Checks, later)
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "failed" || ExitCode(r, err) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	if *r.Checks[0].ExitCode != 7 || r.Checks[1].Status != "not_run" || r.Checks[1].StartedAt != nil {
		t.Fatalf("%+v", r.Checks)
	}
	if _, err := os.Stat(filepath.Join(o.Workspace, "sentinel")); !os.IsNotExist(err) {
		t.Fatal("ran later work")
	}
}

func TestMissingExecutableAndPrerequisiteBlock(t *testing.T) {
	for _, prereq := range []bool{false, true} {
		c := fixtureContract()
		c.Checks[0].Steps = []Step{helperStep(t, "write-sentinel")}
		if prereq {
			c.Checks[0].Prerequisites = []string{"sdd-no-such-executable-6921"}
		} else {
			c.Checks[0].Steps[0].Argv[0] = "sdd-no-such-executable-6921"
		}
		o := optionsFor(t, c)
		r, err := Run(context.Background(), o)
		if err != nil || r.Execution != "blocked" || ExitCode(r, err) != 2 || r.Checks[0].ExitCode != nil {
			t.Fatalf("%+v %v", r, err)
		}
		if _, err := os.Stat(filepath.Join(o.Workspace, "sentinel")); !os.IsNotExist(err) {
			t.Fatal("ran blocked work")
		}
	}
}

func TestBoundedOutput(t *testing.T) {
	c := fixtureContract()
	c.Checks[0].Steps = []Step{helperStep(t, "flood")}
	o := optionsFor(t, c)
	o.MaxLogBytes = 256
	r, err := Run(context.Background(), o)
	if err != nil || !r.Checks[0].Truncated || r.Execution != "incomplete" {
		t.Fatalf("%+v %v", r, err)
	}
	data, _ := os.ReadFile(filepath.Join(o.Output, "fixture.log"))
	if len(data) > 256 || !strings.Contains(string(data), "TRUNCATED") {
		t.Fatalf("bounded log: %q", data)
	}
}

func TestTidyDetectsNewUntrackedSum(t *testing.T) {
	c := fixtureContract()
	c.Checks[0].ID = "tidy-check"
	c.Checks[0].Steps = []Step{helperStep(t, "write-sum")}
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "failed" || !strings.Contains(r.Checks[0].Reason, "module") {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestPathsFailBeforeExecution(t *testing.T) {
	for _, which := range []string{"cwd-symlink", "output-in-workspace", "output-symlink", "nested-cwd"} {
		t.Run(which, func(t *testing.T) {
			c := fixtureContract()
			c.Checks[0].Steps = []Step{helperStep(t, "write-sentinel")}
			o := optionsFor(t, c)
			switch which {
			case "cwd-symlink":
				if err := os.Symlink(filepath.Dir(o.Workspace), filepath.Join(o.Workspace, "escape")); err != nil {
					t.Fatal(err)
				}
				c.Checks[0].CWD = "escape"
			case "output-in-workspace":
				o.Output = filepath.Join(o.Workspace, "evidence")
			case "output-symlink":
				if err := os.Symlink(o.Workspace, o.Output); err != nil {
					t.Fatal(err)
				}
			case "nested-cwd":
				c.Checks[0].CWD = "missing"
			}
			data, _ := json.Marshal(c)
			_ = os.WriteFile(o.ContractPath, data, 0600)
			if _, err := Run(context.Background(), o); err == nil {
				t.Fatal("unsafe input accepted")
			}
			if _, err := os.Stat(filepath.Join(o.Workspace, "sentinel")); !os.IsNotExist(err) {
				t.Fatal("executed invalid contract")
			}
		})
	}
}

func TestFocusedDoesNotClaimRequired(t *testing.T) {
	c := fixtureContract()
	c.Checks[0].Steps = []Step{helperStep(t, "pass")}
	o := optionsFor(t, c)
	o.Scope, o.CheckIDs = "focused", []string{"fixture"}
	r, err := Run(context.Background(), o)
	if err != nil || r.Selection.Scope != "focused" || len(r.Selection.CheckIDs) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestTimeoutAndCancellation(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		c := fixtureContract()
		c.Checks[0].Steps = []Step{helperStep(t, "sleep")}
		o := optionsFor(t, c)
		ctx := context.Background()
		want := "incomplete"
		if cancel {
			var stop context.CancelFunc
			ctx, stop = context.WithCancel(ctx)
			timer := time.AfterFunc(150*time.Millisecond, stop)
			defer timer.Stop()
			defer stop()
			want = "cancelled"
		} else {
			o.Deadline = 150 * time.Millisecond
		}
		start := time.Now()
		r, err := Run(ctx, o)
		if err != nil || r.Execution != want || time.Since(start) > 4*time.Second {
			t.Fatalf("%+v %v", r, err)
		}
	}
}

func TestTimeoutCoversAllStepsNotEachStep(t *testing.T) {
	c := fixtureContract()
	c.Checks[0].TimeoutSeconds = 1
	c.Checks[0].Steps = []Step{helperStep(t, "sleep-short"), helperStep(t, "sleep-short")}
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "incomplete" {
		t.Fatalf("%+v %v", r, err)
	}
	data, _ := os.ReadFile(filepath.Join(o.Output, "fixture.log"))
	if !strings.Contains(string(data), "step 2/2") {
		t.Fatal("fixture did not reach second step; cannot prove whole-check timeout")
	}
}

func TestArgvIsLiteralAndEnvOverridesControlLookup(t *testing.T) {
	c := fixtureContract()
	step := helperStep(t, "literal")
	payload := "$(touch sentinel); $HOME"
	step.Argv = append(step.Argv[:len(step.Argv)-1], payload, "literal")
	c.Checks[0].Steps = []Step{step}
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "completed" {
		t.Fatalf("%+v %v", r, err)
	}
	data, _ := os.ReadFile(filepath.Join(o.Output, "fixture.log"))
	if !strings.Contains(string(data), payload) {
		t.Fatal("argument was expanded or altered")
	}
	if _, err := os.Stat(filepath.Join(o.Workspace, "sentinel")); !os.IsNotExist(err) {
		t.Fatal("implicit shell executed")
	}
	c.Checks[0].Steps = []Step{{Argv: []string{"go", "version"}, Env: map[string]string{"PATH": t.TempDir()}}}
	o = optionsFor(t, c)
	r, err = Run(context.Background(), o)
	if err != nil || r.Execution != "blocked" {
		t.Fatalf("ignored overridden PATH: %+v %v", r, err)
	}
}

func TestMissingEvidenceCannotPass(t *testing.T) {
	for _, action := range []string{"unlink-log", "unlink-result"} {
		t.Run(action, func(t *testing.T) {
			c := fixtureContract()
			c.Checks[0].Steps = []Step{helperStep(t, action)}
			o := optionsFor(t, c)
			r, err := Run(context.Background(), o)
			if r.Execution != "incomplete" || ExitCode(r, err) != 1 {
				t.Fatalf("missing evidence passed: %+v %v", r, err)
			}
		})
	}
}

func TestLogRetainsAnEarlierWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "read-only-handle")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	log := &boundedLog{file: file, remaining: 100}
	if _, err := log.Write([]byte("step marker")); err == nil {
		t.Fatal("fixture handle unexpectedly writable")
	}
	if err := log.close(); err == nil {
		t.Fatal("lost earlier write failure after successful sync/close")
	}
}

func TestClosedLogCanDrainLateOutputWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "closed.log")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	log := &boundedLog{file: file, remaining: 100}
	if err := log.close(); err != nil {
		t.Fatal(err)
	}
	if n, err := log.Write([]byte("late output")); err != nil || n != len("late output") {
		t.Fatalf("late output was not drained: %d %v", n, err)
	}
	data, _ := os.ReadFile(path)
	if len(data) != 0 {
		t.Fatal("late output changed closed evidence")
	}
}
