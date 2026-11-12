package main

// Feature 015 W2-A — `fleet status` byte-compatibility golden tests.
//
// The refactor onto fleet.BuildStatus must not change the text a person or a
// script already reads. These tests compare the complete stdout and the exit
// code, not selected substrings, for the read-only and post-apply fixtures that
// the pre-refactor CLI produced.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"agentic-sdd/internal/fleet"
)

type fleetStatusGoldenInstance struct {
	id               string
	model            string
	source           string
	state            string
	installed        string
	provenanceDigest string
	beforeModel      string
	reserved         bool
}

func fleetStatusGoldenDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func fleetStatusGoldenShort(digest string) string {
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

func fleetStatusGoldenOutput(configPath, home, configDigest string, instances []fleetStatusGoldenInstance) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Fleet Status:\n")
	fmt.Fprintf(&b, "  Configuration: %s (digest: %s)\n", configPath, configDigest)
	fmt.Fprintf(&b, "  Target root:   %s\n", filepath.Join(home, ".config", "opencode", "agents"))
	fmt.Fprintf(&b, "  Instances (%d):\n", len(instances))
	for _, inst := range instances {
		fmt.Fprintf(&b, "    - %s:\n", inst.id)
		fmt.Fprintf(&b, "        Model:            %s\n", inst.model)
		fmt.Fprintf(&b, "        Source:           %s\n", inst.source)
		fmt.Fprintf(&b, "        State:            %s\n", inst.state)
		fmt.Fprintf(&b, "        Installed:        %s\n", inst.installed)
		fmt.Fprintf(&b, "        Selected primary: not selected (offline default)\n")
		fmt.Fprintf(&b, "        Runtime:          not checked (offline)\n")
		if inst.provenanceDigest != "" {
			fmt.Fprintf(&b, "        Provenance:       managed (last applied %s)\n", fleetStatusGoldenShort(inst.provenanceDigest))
		}
		if inst.beforeModel != "" && inst.model != "" && inst.beforeModel != inst.model {
			fmt.Fprintf(&b, "        Warning:          model route changed from %q to %q\n", inst.beforeModel, inst.model)
		}
		if inst.reserved {
			fmt.Fprintf(&b, "        Warning:          instance uses reserved SDD name (%s)\n", inst.id)
		}
	}
	return b.String()
}

func runFleetStatusGolden(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runNonInteractive(append([]string{"fleet", "status"}, args...), &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func TestFleetStatusGoldenWithoutInstall(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configPath := filepath.Join(root, "fleet.json")
	config := `{
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
}
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runFleetStatusGolden(t, "--config", configPath, "--home", home)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := fleetStatusGoldenOutput(configPath, home, fleetStatusGoldenDigest([]byte(config)), []fleetStatusGoldenInstance{
		{id: "sdd-implement-1", model: "9router/ag/gemini-3.8-flash-high", source: "preset:sdd-implement-1", state: "enabled", installed: "absent"},
		{id: "vision", source: "agent:sdd-visual-inspector", state: "enabled", installed: "absent", reserved: true},
	})
	if stdout != want {
		t.Fatalf("status output changed:\n--- got ---\n%s\n--- want ---\n%s", stdout, want)
	}
	if _, err := os.Lstat(filepath.Join(home, ".config", "opencode", "agents")); !os.IsNotExist(err) {
		t.Fatalf("read-only status touched the target root: %v", err)
	}
}

func TestFleetStatusGoldenManagedDriftAndUnsafe(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configPath := filepath.Join(root, "fleet.json")
	config := `{
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
}
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"fleet", "plan", "--config", configPath, "--output", planPath, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("plan failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runNonInteractive([]string{"fleet", "apply", "--plan", planPath, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("apply failed: %s", stderr.String())
	}
	appliedImplement, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "agents", "sdd-implement-1.md"))
	if err != nil {
		t.Fatal(err)
	}
	appliedVision, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "agents", "vision.md"))
	if err != nil {
		t.Fatal(err)
	}
	implementApplied := fleetStatusGoldenDigest(appliedImplement)
	visionApplied := fleetStatusGoldenDigest(appliedVision)
	instances := []fleetStatusGoldenInstance{
		{id: "sdd-implement-1", model: "9router/ag/gemini-3.8-flash-high", source: "preset:sdd-implement-1", state: "enabled", installed: "installed (managed)", provenanceDigest: implementApplied},
		{id: "vision", source: "agent:sdd-visual-inspector", state: "enabled", installed: "installed (managed)", provenanceDigest: visionApplied, reserved: true},
	}
	digest := fleetStatusGoldenDigest([]byte(config))

	got, gotErr, gotCode := runFleetStatusGolden(t, "--config", configPath, "--home", home)
	if gotCode != 0 {
		t.Fatalf("exit code = %d; stderr: %s", gotCode, gotErr)
	}
	if want := fleetStatusGoldenOutput(configPath, home, digest, instances); got != want {
		t.Fatalf("managed status output changed:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	// Model change with an externally modified managed file: drift label,
	// recorded prior model warning, provenance still the last applied digest.
	changedConfig := `{
  "schemaVersion": 1,
  "instances": {
    "sdd-implement-1": {
      "source": "preset:sdd-implement-1",
      "model": "9router/ag/gemini-3.8-flash-medium"
    },
    "vision": {
      "source": "agent:sdd-visual-inspector"
    }
  }
}
`
	if err := os.WriteFile(configPath, []byte(changedConfig), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "opencode", "agents", "sdd-implement-1.md"), []byte("manually altered content\n"), 0600); err != nil {
		t.Fatal(err)
	}
	instances[0] = fleetStatusGoldenInstance{
		id: "sdd-implement-1", model: "9router/ag/gemini-3.8-flash-medium", source: "preset:sdd-implement-1",
		state: "enabled", installed: "installed (managed, drifted)", provenanceDigest: implementApplied,
		beforeModel: "9router/ag/gemini-3.8-flash-high",
	}
	digest = fleetStatusGoldenDigest([]byte(changedConfig))
	got, gotErr, gotCode = runFleetStatusGolden(t, "--config", configPath, "--home", home)
	if gotCode != 0 {
		t.Fatalf("exit code = %d; stderr: %s", gotCode, gotErr)
	}
	if want := fleetStatusGoldenOutput(configPath, home, digest, instances); got != want {
		t.Fatalf("drifted status output changed:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	// A symlink at the target is an unsafe target, never followed.
	other := filepath.Join(root, "other.txt")
	if err := os.WriteFile(other, []byte("target\n"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".config", "opencode", "agents", "sdd-implement-1.md")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, target); err != nil {
		t.Fatal(err)
	}
	instances[0].installed = "installed (unsafe target; refused)"
	got, gotErr, gotCode = runFleetStatusGolden(t, "--config", configPath, "--home", home)
	if gotCode != 0 {
		t.Fatalf("exit code = %d; stderr: %s", gotCode, gotErr)
	}
	if want := fleetStatusGoldenOutput(configPath, home, digest, instances); got != want {
		t.Fatalf("unsafe status output changed:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if got, err := os.ReadFile(other); err != nil || string(got) != "target\n" {
		t.Fatalf("status followed or changed the symlink target: %q, %v", got, err)
	}
}

func TestFleetStatusGoldenInvalidConfigExit(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "invalid.json")
	if err := os.WriteFile(configPath, []byte(`{"schemaVersion": 1, bad json`), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runFleetStatusGolden(t, "--config", configPath, "--home", filepath.Join(root, "home"))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Fatalf("invalid config wrote stdout: %q", stdout)
	}
	if !bytes.Contains([]byte(stderr), []byte("error:")) {
		t.Fatalf("invalid config stderr = %q", stderr)
	}
}

// TestFleetStatusGoldenMatchesBuildStatus pins the display mapping to the
// typed projection so a future state label cannot silently change the CLI
// without this test failing.
func TestFleetStatusGoldenMatchesBuildStatus(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configPath := filepath.Join(root, "fleet.json")
	config := `{"schemaVersion":1,"instances":{"sdd-implement-1":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"}}}`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := fleet.BuildStatus(configPath, "", home)
	if err != nil {
		t.Fatal(err)
	}
	inst := status.Instances[0]
	if inst.Installed != fleet.InstalledAbsent {
		t.Fatalf("installed = %q, want absent", inst.Installed)
	}
	stdout, stderr, code := runFleetStatusGolden(t, "--config", configPath, "--home", home)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr: %s", code, stderr)
	}
	want := fleetStatusGoldenOutput(configPath, home, status.ConfigDigest, []fleetStatusGoldenInstance{
		{id: "sdd-implement-1", model: "9router/ag/gemini-3.8-flash-high", source: "preset:sdd-implement-1", state: "enabled", installed: "absent"},
	})
	if stdout != want {
		t.Fatalf("status output drifted from BuildStatus:\n--- got ---\n%s\n--- want ---\n%s", stdout, want)
	}
}
