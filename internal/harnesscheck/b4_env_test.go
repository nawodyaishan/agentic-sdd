package harnesscheck

// B4 T07 (AC-17, environment part): executed checks must not inherit hostile
// Go build/selection controls, while an explicit trusted-contract override is
// preserved.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const b4EnvHelper = "SDD_B4_ENV_HELPER"

// b4EnvStep re-executes this test binary so the child can report the
// environment the runner actually projected.
func b4EnvStep(t *testing.T, action string) Step {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Step{
		Argv: []string{exe, "-test.run=^TestB4EnvHelper$", "--", action},
		Env:  map[string]string{b4EnvHelper: "1", "GORACE": "atexit_sleep_ms=0"},
	}
}

func TestB4EnvHelper(t *testing.T) {
	if os.Getenv(b4EnvHelper) != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "goflags":
		_, _ = os.Stdout.WriteString("GOFLAGS=" + os.Getenv("GOFLAGS") + "\n")
	case "gowork":
		_, _ = os.Stdout.WriteString("GOWORK=" + os.Getenv("GOWORK") + "\n")
	default:
		os.Exit(11)
	}
	os.Exit(0)
}

func TestCheckEnvironmentProjection(t *testing.T) {
	hostile := map[string]string{
		"GOFLAGS":      "-tags=hostile_excluded_work",
		"GOENV":        "/nonexistent/hostile-go-env",
		"GOWORK":       "/nonexistent/hostile/go.work",
		"GO111MODULE":  "off",
		"GOTOOLCHAIN":  "go1.99.0",
		"GOEXPERIMENT": "hostileexperiment",
		"GOOS":         "windows",
		"GOARCH":       "mips",
		"CGO_ENABLED":  "0",
		"GOARM":        "5",
		"GOARM64":      "v8.0",
		"GOAMD64":      "v1",
		"GO386":        "387",
		"GOMIPS":       "softfloat",
		"GOMIPS64":     "softfloat",
		"GOPPC64":      "power7",
		"GORISCV64":    "rva20u64",
		"GOWASM":       "signext",
	}
	for key, value := range hostile {
		t.Setenv(key, value)
	}
	t.Setenv("SDD_B4_ORDINARY", "kept")

	values := map[string]string{}
	for _, entry := range environment(map[string]string{"GOOS": "darwin", "GOARCH": "arm64", "PATH": "/approved"}) {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("malformed environment entry %q", entry)
		}
		values[key] = value
	}

	if values["GOFLAGS"] != "" {
		t.Fatalf("inherited GOFLAGS survived: %q", values["GOFLAGS"])
	}
	if values["GOENV"] != "off" {
		t.Fatalf("GOENV was not pinned: %q", values["GOENV"])
	}
	if values["GOWORK"] != "off" {
		t.Fatalf("GOWORK was not pinned: %q", values["GOWORK"])
	}
	for _, key := range []string{"GO111MODULE", "GOTOOLCHAIN", "GOEXPERIMENT", "CGO_ENABLED", "GOARM", "GOARM64", "GOAMD64", "GO386", "GOMIPS", "GOMIPS64", "GOPPC64", "GORISCV64", "GOWASM"} {
		if value, ok := values[key]; ok {
			t.Fatalf("inherited %s was not removed: %q", key, value)
		}
	}
	if values["GOOS"] != "darwin" || values["GOARCH"] != "arm64" {
		t.Fatalf("approved GOOS/GOARCH override was not preserved: %q/%q", values["GOOS"], values["GOARCH"])
	}
	if values["PATH"] != "/approved" {
		t.Fatalf("approved PATH override was not preserved: %q", values["PATH"])
	}
	if values["SDD_B4_ORDINARY"] != "kept" {
		t.Fatalf("ordinary inherited variable was dropped: %q", values["SDD_B4_ORDINARY"])
	}
}

func TestHostileInheritedGoControlsDoNotReachExecutedChecks(t *testing.T) {
	if !processSupported {
		t.Skip("check execution unsupported on this host")
	}
	t.Setenv("GOFLAGS", "-tags=hostile_excluded_work")
	t.Setenv("GOWORK", "/nonexistent/hostile/go.work")
	c := fixtureContract()
	c.Checks[0].Steps = []Step{b4EnvStep(t, "goflags")}
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "completed" {
		t.Fatalf("%+v %v", r, err)
	}
	data, err := os.ReadFile(filepath.Join(o.Output, "fixture.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hostile_excluded_work") {
		t.Fatalf("inherited GOFLAGS reached the executed check: %q", data)
	}
	if !strings.Contains(string(data), "GOFLAGS=\n") {
		t.Fatalf("executed check did not see a pinned empty GOFLAGS: %q", data)
	}
}

func TestApprovedContractOverridesArePreserved(t *testing.T) {
	if !processSupported {
		t.Skip("check execution unsupported on this host")
	}
	t.Setenv("GOFLAGS", "-tags=hostile_excluded_work")
	c := fixtureContract()
	step := b4EnvStep(t, "goflags")
	step.Env["GOFLAGS"] = "-tags=approved"
	c.Checks[0].Steps = []Step{step}
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "completed" {
		t.Fatalf("%+v %v", r, err)
	}
	data, err := os.ReadFile(filepath.Join(o.Output, "fixture.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "GOFLAGS=-tags=approved") {
		t.Fatalf("approved contract override was not preserved: %q", data)
	}
}
