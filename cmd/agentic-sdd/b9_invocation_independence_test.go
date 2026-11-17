package main

// Feature 022 Batch B9 (T20): AC-19 invocation independence.
//
// The touched MCP and launcher paths must let two differently configured
// invocations run without swapping a package global or sharing draft or
// confirmation state. Each call carries its dependencies in an invocation
// value; these tests construct two of them side by side, concurrently where
// that is the strongest evidence, and assert that neither sees the other's
// engine, prompt answer or launcher.

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// b9StatusEngine answers status reports with its own label, so a leaked engine
// is visible in the output.
func b9StatusEngine(label string) mcpEngine {
	engine := productionMCPEngine(strings.NewReader(""))
	engine.newStatusRequest = func(home, config, project string) (appops.MCPStatusRequest, error) {
		return appops.MCPStatusRequest{}, nil
	}
	engine.statusReport = func(appops.MCPStatusRequest) (appops.MCPStatusResult, error) {
		status := *mcpFixtureStatus()
		status.Home = "/tmp/" + label
		status.ConfigPath = "/tmp/" + label + "/opencode.json"
		return appops.MCPStatusResult{Status: &status}, nil
	}
	return engine
}

func TestB9ConcurrentMCPInvocationsKeepTheirOwnEngine(t *testing.T) {
	labels := []string{"engine-a", "engine-b"}
	type result struct {
		code   int
		stdout string
		stderr string
	}
	results := make([]result, len(labels))
	var wg sync.WaitGroup
	for i, label := range labels {
		wg.Add(1)
		go func(i int, label string) {
			defer wg.Done()
			var stdout, stderr bytes.Buffer
			inv := productionInvocation(strings.NewReader(""))
			inv.mcp = b9StatusEngine(label)
			code := runEnvWith([]string{"mcp", "status", "--home", "/tmp/" + label}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80}, inv)
			results[i] = result{code: code, stdout: stdout.String(), stderr: stderr.String()}
		}(i, label)
	}
	wg.Wait()

	for i, label := range labels {
		if results[i].code != 0 {
			t.Fatalf("%s: exit %d; stderr %q", label, results[i].code, results[i].stderr)
		}
		if !strings.Contains(results[i].stdout, "/tmp/"+label) {
			t.Fatalf("%s: status did not use its own engine:\n%s", label, results[i].stdout)
		}
		for _, other := range labels {
			if other == label {
				continue
			}
			if strings.Contains(results[i].stdout, "/tmp/"+other) {
				t.Fatalf("%s: status leaked the %s engine:\n%s", label, other, results[i].stdout)
			}
		}
	}
}

// b9RestoreEngine confirms one restore against a per-invocation prompt answer
// and records whether the writer was reached.
func b9RestoreEngine(answer string, applied *int) mcpEngine {
	prompt := strings.NewReader(answer)
	engine := productionMCPEngine(prompt)
	engine.restorePreview = func(home, configPath, backupID string) (*skillsync.MCPRestorePreview, string, error) {
		return mcpWriteRestorePreviewFixture(), mcpWriteToken, nil
	}
	engine.restoreApply = func(home, configPath, backupID, token, expectDigest string) (*skillsync.MCPApplyResult, error) {
		*applied++
		return mcpWriteRestoreResultFixture(), nil
	}
	return engine
}

func TestB9MCPConfirmationsArePerInvocation(t *testing.T) {
	type result struct {
		code   int
		stdout string
	}
	run := func(label, answer string, applied *int) result {
		var stdout, stderr bytes.Buffer
		inv := productionInvocation(strings.NewReader(""))
		inv.mcp = b9RestoreEngine(answer, applied)
		code := runEnvWith([]string{
			"mcp", "restore", "apply", mcpWriteBackupID,
			"--home", "/tmp/" + label,
			"--config", mcpWriteConfigPath,
			"--expect-config", mcpWriteBaseDigest,
		}, &stdout, &stderr, strings.NewReader(""), &Env{Interactive: true, Width: 80}, inv)
		if code != 0 {
			t.Fatalf("%s: exit %d; stderr %q", label, code, stderr.String())
		}
		return result{code: code, stdout: stdout.String()}
	}

	// Run both invocations concurrently: a shared prompt or confirmation state
	// would let one answer decide the other restore.
	var declined, applied int
	var wg sync.WaitGroup
	var declinedOut, appliedOut string
	wg.Add(2)
	go func() {
		defer wg.Done()
		declinedOut = run("declined-home", "n\n", &declined).stdout
	}()
	go func() {
		defer wg.Done()
		appliedOut = run("applied-home", "y\n", &applied).stdout
	}()
	wg.Wait()

	if declined != 0 {
		t.Fatalf("the declined invocation wrote %d times", declined)
	}
	if applied != 1 {
		t.Fatalf("the confirmed invocation wrote %d times, want 1", applied)
	}
	if !strings.Contains(declinedOut, "Cancelled.") || strings.Contains(declinedOut, "Result:      applied") {
		t.Fatalf("the declined invocation did not stay cancelled:\n%s", declinedOut)
	}
	if !strings.Contains(appliedOut, "Result:      applied") {
		t.Fatalf("the confirmed invocation did not apply:\n%s", appliedOut)
	}
}

func TestB9TUIInvocationsKeepTheirOwnLauncher(t *testing.T) {
	launcher := func(code int, seen *[]string) tuiLauncher {
		return func(req tuiRequest, _ io.Reader, _ io.Writer, _ io.Writer, _ *Env) int {
			*seen = append(*seen, req.Home)
			return code
		}
	}

	var firstSeen, secondSeen []string
	first := productionInvocation(strings.NewReader(""))
	first.launchTUI = launcher(0, &firstSeen)
	second := productionInvocation(strings.NewReader(""))
	second.launchTUI = launcher(3, &secondSeen)

	env := &Env{RealStdinTTY: true, RealStdoutTTY: true, Width: 80}
	for _, tc := range []struct {
		name string
		inv  invocation
		home string
		code int
		seen *[]string
	}{
		{name: "first", inv: first, home: "/tmp/tui-first", code: 0, seen: &firstSeen},
		{name: "second", inv: second, home: "/tmp/tui-second", code: 3, seen: &secondSeen},
	} {
		var stdout, stderr bytes.Buffer
		code := runEnvWith([]string{"tui", "--home", tc.home}, &stdout, &stderr, strings.NewReader(""), env, tc.inv)
		if code != tc.code {
			t.Fatalf("%s: exit %d, want %d; stderr %q", tc.name, code, tc.code, stderr.String())
		}
	}
	if len(firstSeen) != 1 || firstSeen[0] != "/tmp/tui-first" {
		t.Fatalf("first launcher saw %v", firstSeen)
	}
	if len(secondSeen) != 1 || secondSeen[0] != "/tmp/tui-second" {
		t.Fatalf("second launcher saw %v", secondSeen)
	}
}
