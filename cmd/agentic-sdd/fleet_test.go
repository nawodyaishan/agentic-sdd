package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFleetCLI(t *testing.T) {
	// 1. Help invocations
	helpCases := []struct {
		name string
		args []string
		want string
	}{
		{"help fleet", []string{"help", "fleet"}, "Usage: asdx-core fleet"},
		{"fleet help", []string{"fleet", "help"}, "Usage: asdx-core fleet"},
		{"fleet --help", []string{"fleet", "--help"}, "Usage: asdx-core fleet"},
		{"fleet status --help", []string{"fleet", "status", "--help"}, "Usage: asdx-core fleet status"},
		{"fleet plan --help", []string{"fleet", "plan", "--help"}, "Usage: asdx-core fleet plan"},
		{"fleet apply --help", []string{"fleet", "apply", "--help"}, "Usage: asdx-core fleet apply"},
		{"fleet restore --help", []string{"fleet", "restore", "--help"}, "Usage: asdx-core fleet restore"},
		{"fleet doctor --help", []string{"fleet", "doctor", "--help"}, "Usage: asdx-core fleet doctor"},
	}

	for _, tc := range helpCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit code %d, want 0; stderr: %s", code, stderr.String())
			}
			got := stdout.String() + stderr.String()
			if !strings.Contains(got, tc.want) {
				t.Fatalf("output %q does not contain %q", got, tc.want)
			}
		})
	}

	// 2. Unknown subcommand and missing subcommand (Exit 2)
	unknownCases := []struct {
		name string
		args []string
		want string
	}{
		{"bare fleet", []string{"fleet"}, "Usage: asdx-core fleet"},
		{"unknown sub", []string{"fleet", "bogus"}, "unknown fleet command"},
		{"status missing config", []string{"fleet", "status"}, "--config is required"},
		{"plan missing config", []string{"fleet", "plan"}, "--config is required"},
		{"apply missing plan", []string{"fleet", "apply"}, "--plan is required"},
		{"restore missing backup", []string{"fleet", "restore"}, "--backup is required"},
		{"restore positional ID rejected", []string{"fleet", "restore", "20261111-120000"}, "usage: asdx-core fleet restore --backup ID"},
	}

	for _, tc := range unknownCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(tc.args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("exit code %d, want 2; stdout: %s, stderr: %s", code, stdout.String(), stderr.String())
			}
			got := stdout.String() + stderr.String()
			if !strings.Contains(got, tc.want) {
				t.Fatalf("output %q does not contain %q", got, tc.want)
			}
		})
	}

	// 3. Valid status and plan execution
	tempDir := t.TempDir()
	home := filepath.Join(tempDir, "home")
	configPath := filepath.Join(tempDir, "fleet.json")
	configJSON := `{
  "schemaVersion": 1,
  "instances": {
    "sdd-implement-1": {
      "source": "preset:sdd-implement-1",
      "model": "9router/ag/gemini-3.8-flash-high"
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	var statusOut, statusErr bytes.Buffer
	code := runNonInteractive([]string{"fleet", "status", "--config", configPath, "--home", home}, &statusOut, &statusErr)
	if code != 0 {
		t.Fatalf("fleet status failed code %d: %s", code, statusErr.String())
	}
	if !strings.Contains(statusOut.String(), "Fleet Status:") || !strings.Contains(statusOut.String(), "sdd-implement-1") {
		t.Fatalf("unexpected fleet status output: %s", statusOut.String())
	}

	// Plan without --output writes nothing to disk
	planPath := filepath.Join(tempDir, "plan.json")
	var planOut, planErr bytes.Buffer
	code = runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--home", home}, &planOut, &planErr)
	if code != 0 {
		t.Fatalf("fleet plan failed code %d: %s", code, planErr.String())
	}
	if !strings.Contains(planOut.String(), "Fleet Plan (Plan ID:") {
		t.Fatalf("unexpected fleet plan output: %s", planOut.String())
	}
	if _, err := os.Stat(planPath); !os.IsNotExist(err) {
		t.Fatalf("fleet plan without --output unexpectedly created file: %v", err)
	}

	// Plan with --output writes plan.json
	var planOut2, planErr2 bytes.Buffer
	code = runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--home", home}, &planOut2, &planErr2)
	if code != 0 {
		t.Fatalf("fleet plan --output failed code %d: %s", code, planErr2.String())
	}
	if _, err := os.Stat(planPath); err != nil {
		t.Fatalf("expected plan.json to exist: %v", err)
	}

	// Doctor execution
	var docOut, docErr bytes.Buffer
	code = runNonInteractive([]string{"fleet", "doctor"}, &docOut, &docErr)
	if code != 0 {
		t.Fatalf("fleet doctor failed code %d: %s", code, docErr.String())
	}
	if !strings.Contains(docOut.String(), "fleet doctor") || !strings.Contains(docOut.String(), "offline") {
		t.Fatalf("unexpected fleet doctor output: %s", docOut.String())
	}

	// 4. Non-fleet regression: existing paths are byte-identical
	var regOut, regErr bytes.Buffer
	code = runNonInteractive([]string{"version"}, &regOut, &regErr)
	if code != 0 {
		t.Fatalf("version failed code %d: %s", code, regErr.String())
	}
	if !strings.Contains(regOut.String(), "asdx-core") {
		t.Fatalf("version output: %s", regOut.String())
	}

	var prevOut, prevErr bytes.Buffer
	code = runNonInteractive([]string{"preview", "--home", home}, &prevOut, &prevErr)
	if code != 0 {
		t.Fatalf("preview failed code %d: %s", code, prevErr.String())
	}

	var watOut, watErr bytes.Buffer
	code = runNonInteractive([]string{"wat"}, &watOut, &watErr)
	if code != 2 || !strings.Contains(watErr.String(), "unknown command") {
		t.Fatalf("expected unknown command exit 2, got %d: %s", code, watErr.String())
	}
}

func TestFleetHelpLegend(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runNonInteractive([]string{"fleet", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fleet --help failed: %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Exit Codes:") {
		t.Fatalf("help output missing Exit Codes legend:\n%s", out)
	}
	for _, codeStr := range []string{"0  Success", "1  Runtime error", "2  Usage error"} {
		if !strings.Contains(out, codeStr) {
			t.Errorf("help legend missing %q in:\n%s", codeStr, out)
		}
	}
}

func TestFleetExitCodesTable(t *testing.T) {
	tempDir := t.TempDir()
	home := filepath.Join(tempDir, "home")
	validCfg := filepath.Join(tempDir, "valid.json")
	if err := os.WriteFile(validCfg, []byte(`{"schemaVersion":1,"instances":{"sdd-implement-1":{"source":"preset:sdd-implement-1"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	invalidJSONCfg := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(invalidJSONCfg, []byte(`{"schemaVersion": 1, bad json`), 0600); err != nil {
		t.Fatal(err)
	}
	unknownSourceCfg := filepath.Join(tempDir, "unknown_source.json")
	if err := os.WriteFile(unknownSourceCfg, []byte(`{"schemaVersion":1,"instances":{"a":{"source":"agent:nonexistent"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	dupKeyCfg := filepath.Join(tempDir, "dup.json")
	if err := os.WriteFile(dupKeyCfg, []byte(`{"schemaVersion":1,"instances":{"a":{"source":"preset:sdd-implement-1"},"a":{"source":"preset:sdd-implement-2"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	malformedPlan := filepath.Join(tempDir, "bad_plan.json")
	if err := os.WriteFile(malformedPlan, []byte(`not json`), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantSub  string
	}{
		{"status success", []string{"fleet", "status", "--config", validCfg, "--home", home}, 0, "Fleet Status:"},
		{"plan success", []string{"fleet", "plan", "--config", validCfg, "--home", home}, 0, "Fleet Plan"},
		{"help success", []string{"fleet", "help"}, 0, "Usage: asdx-core fleet"},
		{"unknown sub exit 2", []string{"fleet", "unknowncmd"}, 2, "unknown fleet command"},
		{"missing flag exit 2", []string{"fleet", "status"}, 2, "--config is required"},
		{"invalid json config exit 2", []string{"fleet", "status", "--config", invalidJSONCfg, "--home", home}, 2, "error:"},
		{"unknown source exit 2", []string{"fleet", "status", "--config", unknownSourceCfg, "--home", home}, 2, "error:"},
		{"duplicate key config exit 2", []string{"fleet", "status", "--config", dupKeyCfg, "--home", home}, 2, "duplicate key"},
		{"non-regular config exit 2", []string{"fleet", "status", "--config", tempDir, "--home", home}, 2, "refusing non-regular"},
		{"malformed plan apply exit 2", []string{"fleet", "apply", "--plan", malformedPlan, "--home", home}, 2, "error:"},
		{"non-regular plan apply exit 2", []string{"fleet", "apply", "--plan", tempDir, "--home", home}, 2, "refusing non-regular"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr: %s, stdout: %s", code, tt.wantCode, stderr.String(), stdout.String())
			}
			out := stdout.String() + stderr.String()
			if !strings.Contains(out, tt.wantSub) {
				t.Fatalf("output %q does not contain %q", out, tt.wantSub)
			}
		})
	}
}

func TestFleetStatusDriftLabelsAndWarnings(t *testing.T) {
	tempDir := t.TempDir()
	home := filepath.Join(tempDir, "home")
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(agentsDir, 0700); err != nil {
		t.Fatal(err)
	}

	// 1. Create a managed file via apply
	cfgPath := filepath.Join(tempDir, "fleet.json")
	if err := os.WriteFile(cfgPath, []byte(`{
  "schemaVersion": 1,
  "instances": {
    "sdd-implement-1": {
      "source": "preset:sdd-implement-1",
      "model": "9router/ag/gemini-3.8-flash-high"
    },
    "vision": {
      "source": "agent:sdd-visual-inspector"
    }
  }
}`), 0600); err != nil {
		t.Fatal(err)
	}

	planPath := filepath.Join(tempDir, "plan.json")
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"fleet", "plan", "--config", cfgPath, "--output", planPath, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("plan failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("apply failed: %s", stderr.String())
	}

	// Initial status: managed, clean
	stdout.Reset()
	stderr.Reset()
	if code := runNonInteractive([]string{"fleet", "status", "--config", cfgPath, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("status failed: %s", stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Installed:        installed (managed)") {
		t.Fatalf("expected installed (managed) in output:\n%s", out)
	}
	if !strings.Contains(out, "Selected primary: not selected (offline default)") {
		t.Fatalf("expected selected primary label in output:\n%s", out)
	}
	if !strings.Contains(out, "Runtime:          not checked (offline)") {
		t.Fatalf("expected runtime label in output:\n%s", out)
	}
	if !strings.Contains(out, "Warning:          instance uses reserved SDD name (vision)") {
		t.Fatalf("expected reserved name warning in output:\n%s", out)
	}

	// 2. Introduce drift to managed file
	targetFile := filepath.Join(agentsDir, "sdd-implement-1.md")
	if err := os.WriteFile(targetFile, []byte("manually altered content"), 0600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runNonInteractive([]string{"fleet", "status", "--config", cfgPath, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("status failed: %s", stderr.String())
	}
	out = stdout.String()
	if !strings.Contains(out, "Installed:        installed (managed, drifted)") {
		t.Fatalf("expected installed (managed, drifted) in output:\n%s", out)
	}

	// 3. Make target a symlink (unsafe target)
	_ = os.Remove(targetFile)
	otherFile := filepath.Join(tempDir, "other.txt")
	_ = os.WriteFile(otherFile, []byte("target"), 0600)
	if err := os.Symlink(otherFile, targetFile); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runNonInteractive([]string{"fleet", "status", "--config", cfgPath, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("status failed: %s", stderr.String())
	}
	out = stdout.String()
	if !strings.Contains(out, "Installed:        installed (unsafe target; refused)") {
		t.Fatalf("expected unsafe target refused in output:\n%s", out)
	}
}
