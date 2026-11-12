package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

func TestMCPErrorCodeUsageVersusRuntime(t *testing.T) {
	if got := mcpErrorCode(fmt.Errorf("%w: unknown server", skillsync.ErrMCPUsage)); got != 2 {
		t.Fatalf("usage error exit code = %d, want 2", got)
	}
	if got := mcpErrorCode(errors.New("unexpected I/O failure")); got != 1 {
		t.Fatalf("runtime error exit code = %d, want 1", got)
	}
}

func TestMCPTargetRendersAmbiguity(t *testing.T) {
	var out bytes.Buffer
	printMCPTarget(&out, skillsync.MCPStatus{
		Ambiguity: "both opencode.json and opencode.jsonc exist",
	})
	text := out.String()
	if !strings.Contains(text, "ambiguous") || !strings.Contains(text, "Ambiguity") {
		t.Fatalf("ambiguous target output missing ambiguity detail: %q", text)
	}
}

func TestMCPTargetRendersMissingExplicitPath(t *testing.T) {
	var out bytes.Buffer
	printMCPTarget(&out, skillsync.MCPStatus{
		ConfigPath:   "/tmp/does-not-exist.json",
		ConfigFormat: "none",
		Schema:       "none",
	})
	text := out.String()
	if !strings.Contains(text, "not configured") || !strings.Contains(text, "does not exist") {
		t.Fatalf("missing explicit target output is not truthful: %q", text)
	}
}
