//go:build !agentic_sdd_experimental

package main

import (
	"bytes"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// These tests describe the default (core) build profile: the experimental
// OpenCode helper is inactive and unreachable. They use synthetic temporary
// homes only; they never compile or run the experimental variant, a native
// executable, a provider or a credential store.

func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		entries = append(entries, rel+"|"+info.Mode().String())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	return entries
}

// isolate points the process-visible configuration roots and working
// directory at synthetic empty directories, returning them for inspection.
func isolate(t *testing.T) (home, cwd string) {
	t.Helper()
	root := t.TempDir()
	home, cwd = filepath.Join(root, "home"), filepath.Join(root, "cwd")
	for _, dir := range []string{home, cwd} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "OPENCODE_CONFIG", "OPENCODE_API_KEY", "OPENAI_API_KEY"} {
		t.Setenv(key, filepath.Join(home, "canary-"+key))
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return home, cwd
}

func TestDefaultHelpAdvertisesOnlySupportedCommands(t *testing.T) {
	for _, args := range []string{"--help", "help", "help apply", "preview --help", "wat"} {
		var stdout, stderr bytes.Buffer
		runNonInteractive(strings.Fields(args), &stdout, &stderr)
		got := strings.ToLower(stdout.String() + stderr.String())
		if !strings.Contains(got, "usage: agentic-sdd") {
			t.Fatalf("%q: no usage text in %q", args, got)
		}
		if strings.Contains(got, "opencode") {
			t.Fatalf("%q: default help advertises the experimental helper:\n%s", args, got)
		}
		for _, want := range []string{"preview", "apply", "version", "backups", "logs", "restore"} {
			if !strings.Contains(got, want) {
				t.Fatalf("%q: default help lost supported command %q", args, want)
			}
		}
	}
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"help", "bogus"}, &stdout, &stderr); code != 2 || strings.Contains(strings.ToLower(stderr.String()), "opencode") {
		t.Fatalf("help usage error: code %d, stderr %q", code, stderr.String())
	}
}

func TestDefaultOpenCodeDenialHasNoSideEffects(t *testing.T) {
	cases := [][]string{
		{"opencode"},
		{"opencode", "--help"},
		{"opencode", "help"},
		{"opencode", "candidate", "--feature", "demo", "--build-path", "src"},
		{"opencode", "preflight", "--config", "missing.jsonc"},
		{"opencode", "config", "apply", "--workspace", "w", "--candidate", "c", "--backup-root", "b", "--expect-original", "absent", "--expect-candidate", "x"},
		{"opencode", "session", "run", "--workspace", "w", "--executable", "/nonexistent/native", "--prompt", "hi"},
		{"opencode", "evidence", "snapshot", "--workspace", ".", "--scope", "."},
		{"opencode", "unknown-operation"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			home, cwd := isolate(t)
			beforeHome, beforeCwd := snapshotTree(t, home), snapshotTree(t, cwd)
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("exit %d, want 2; stdout %q stderr %q", code, stdout.String(), stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("denial wrote to stdout: %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), "unavailable in this build") {
				t.Fatalf("diagnostic %q does not say unavailable in this build", stderr.String())
			}
			if strings.Contains(stderr.String(), "Usage: agentic-sdd opencode") {
				t.Fatalf("denial claimed the helper's usage: %q", stderr.String())
			}
			if got := snapshotTree(t, home); strings.Join(got, "\n") != strings.Join(beforeHome, "\n") {
				t.Fatalf("home changed: before %v after %v", beforeHome, got)
			}
			if got := snapshotTree(t, cwd); strings.Join(got, "\n") != strings.Join(beforeCwd, "\n") {
				t.Fatalf("cwd changed: before %v after %v", beforeCwd, got)
			}
		})
	}
}

func TestDefaultHelpOpenCodeOnlyExplainsUnavailability(t *testing.T) {
	home, cwd := isolate(t)
	beforeHome, beforeCwd := snapshotTree(t, home), snapshotTree(t, cwd)
	var stdout, stderr bytes.Buffer
	code := runNonInteractive([]string{"help", "opencode"}, &stdout, &stderr)
	out := stdout.String() + stderr.String()
	if code != 0 {
		t.Fatalf("help opencode exit %d: %s", code, out)
	}
	if !strings.Contains(out, "unavailable in this build") || !strings.Contains(out, "agentic_sdd_experimental") {
		t.Fatalf("help opencode does not explain unavailability and the opt-in build: %q", out)
	}
	for _, claim := range []string{"candidate --feature", "session run", "evidence snapshot", "config apply", "Usage: agentic-sdd opencode"} {
		if strings.Contains(out, claim) {
			t.Fatalf("help opencode advertises capability %q: %s", claim, out)
		}
	}
	if strings.Join(snapshotTree(t, home), "\n") != strings.Join(beforeHome, "\n") || strings.Join(snapshotTree(t, cwd), "\n") != strings.Join(beforeCwd, "\n") {
		t.Fatal("help opencode had a filesystem side effect")
	}
}

func TestDefaultOpenCodeDenialIgnoresPathAndNativeLookup(t *testing.T) {
	_, _ = isolate(t)
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"opencode", "session", "run", "--executable", "opencode"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
}

func goEnvForDefaultProfile() []string {
	env := os.Environ()[:0:0]
	for _, kv := range os.Environ() {
		switch {
		case strings.HasPrefix(kv, "GOFLAGS="), strings.HasPrefix(kv, "GOENV="), strings.HasPrefix(kv, "GOWORK="):
		default:
			env = append(env, kv)
		}
	}
	// Sanitized: a hostile inherited GOFLAGS/GOENV must not change which files
	// are compiled when this test inspects the default profile.
	return append(env, "GOFLAGS=", "GOENV=off", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
}

func TestDefaultBuildExcludesExperimentalFilesAndAdapterImport(t *testing.T) {
	ctx := build.Default
	ctx.BuildTags = nil // the default profile: no opt-in tag
	pkg, err := ctx.ImportDir(".", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range append(append([]string{}, pkg.GoFiles...), pkg.TestGoFiles...) {
		if file == "opencode.go" || file == "opencode_test.go" {
			t.Fatalf("default compiled file set includes experimental file %s", file)
		}
	}
	for _, want := range []string{"logs.go", "main.go", "opencode_disabled.go"} {
		found := false
		for _, file := range pkg.GoFiles {
			found = found || file == want
		}
		if !found {
			t.Fatalf("default compiled file set lacks %s: %v", want, pkg.GoFiles)
		}
	}
	if len(pkg.IgnoredGoFiles) < 2 {
		t.Fatalf("expected opencode.go and opencode_test.go ignored by constraints, got %v", pkg.IgnoredGoFiles)
	}
	for _, imp := range append(append([]string{}, pkg.Imports...), pkg.TestImports...) {
		if strings.Contains(imp, "opencodeadapter") {
			t.Fatalf("default package imports the experimental backend: %s", imp)
		}
	}
}

func TestDefaultTransitiveDependenciesExcludeExperimentalBackend(t *testing.T) {
	gobin := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(gobin, "list", "-deps", "-test", "-f", "{{.ImportPath}}", ".")
	cmd.Env = goEnvForDefaultProfile()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, stderr.String())
	}
	deps := stdout.String()
	if !strings.Contains(deps, "agentic-sdd/internal/skillsync") {
		t.Fatalf("dependency listing lacks the core installer package; listing unusable: %q", deps)
	}
	if strings.Contains(deps, "opencodeadapter") {
		t.Fatalf("default build transitively depends on the experimental backend:\n%s", deps)
	}
}

func TestDefaultGoListExcludesExperimentalFiles(t *testing.T) {
	gobin := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(gobin, "list", "-f", "{{.GoFiles}} {{.TestGoFiles}}", ".")
	cmd.Env = goEnvForDefaultProfile()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list files: %v\n%s", err, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"opencode_disabled.go", "opencode_disabled_test.go"} {
		if !strings.Contains(out, want) {
			t.Fatalf("go list missing expected core file %s:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"opencode.go", "opencode_test.go"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("go list includes experimental file %s:\n%s", unwanted, out)
		}
	}
}
