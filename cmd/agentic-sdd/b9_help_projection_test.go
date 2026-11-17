package main

// Feature 022 Batch B9 (T20): truthful root help and preserved help-order
// exceptions. FR-13/AC-13 require the root and topic help to disclose the
// actual MCP write surface and the opt-in pack flags, and the accepted help
// topics. O-07 requires the existing help ordering and grammar to stay put:
// only the disclosed facts change, never the command names, the flag grammar or
// the order of the blocks.

import (
	"bytes"
	"strings"
	"testing"
)

func b9RootHelp(t *testing.T) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit %d: %s", code, stderr.String())
	}
	return stdout.String()
}

// TestB9RootHelpDisclosesMCPWritesAndPackFlags pins the corrected projection:
// the write surface and the opt-in pack options are named, and the old
// "read-only" description of the mcp command is gone.
func TestB9RootHelpDisclosesMCPWritesAndPackFlags(t *testing.T) {
	help := b9RootHelp(t)

	for _, want := range []string{
		"asdx-core mcp apply",
		"asdx-core mcp restore",
		"--with-opencode",
		"--opencode-free-first",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("root help does not disclose %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "read-only (status, doctor, preview)") {
		t.Fatalf("root help still describes mcp as read-only over three commands:\n%s", help)
	}
	for _, forbidden := range []string{"asdx-core opencode ", "asdx-core opencode-harness"} {
		if strings.Contains(help, forbidden) {
			t.Fatalf("root help advertises the experimental command %q:\n%s", forbidden, help)
		}
	}
}

// TestB9RootHelpDisclosesAcceptedTopics pins the topic synopsis: every topic
// isHelpTopic accepts is listed, so the root help cannot hide an accepted
// query.
func TestB9RootHelpDisclosesAcceptedTopics(t *testing.T) {
	help := b9RootHelp(t)
	for _, topic := range []string{"preview", "apply", "backups", "logs", "restore", "guide", "version", "tui", "orchestrate", "fleet", "mcp", "run", "task", "review"} {
		if !strings.Contains(help, topic) {
			t.Fatalf("root help does not disclose accepted topic %q:\n%s", topic, help)
		}
	}
	// Experimental commands are accepted only as the bare dispatch word, never
	// advertised here: they stay behind the build tag and are not topics.
	if strings.Contains(help, "opencode-harness|") || strings.Contains(help, "|opencode|") {
		t.Fatalf("root help advertises the experimental opencode topic:\n%s", help)
	}
}

// TestB9RootHelpOrderPreserved pins the current block order (O-07): the usage
// synopsis first, then commands, then restore forms, then flags. Only the
// disclosed facts change; the ordering exception stays as documented.
func TestB9RootHelpOrderPreserved(t *testing.T) {
	help := b9RootHelp(t)
	order := []string{"Usage: asdx-core", "\nCommands:\n", "\nrestore forms:\n", "\nFlags:\n"}
	previous := -1
	for _, marker := range order {
		index := strings.Index(help, marker)
		if index < 0 {
			t.Fatalf("root help lost the %q block:\n%s", marker, help)
		}
		if index <= previous {
			t.Fatalf("root help reordered the %q block:\n%s", marker, help)
		}
		previous = index
	}
}

// TestB9MCPTopicHelpDisclosesWrites pins the MCP topic help: it already names
// the write commands, the opt-in digest and the forbidden operations, and that
// must not regress while the root help is corrected.
func TestB9MCPTopicHelpDisclosesWrites(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"help", "mcp"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help mcp exit %d: %s", code, stderr.String())
	}
	help := stdout.String()
	for _, want := range []string{
		"asdx-core mcp apply --expect-config",
		"asdx-core mcp restore apply",
		"MCP-B adds apply and",
		"no replace or force option",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("mcp help does not disclose %q:\n%s", want, help)
		}
	}
}

// TestB9HelpGrammarExceptionsPreserved documents and pins the deliberate
// grammar exceptions rather than silently changing them: the execution
// families accept the "help" word only, while fleet and mcp also accept
// --help/-h. The wrapper answers only the exact top-level queries.
func TestB9HelpGrammarExceptionsPreserved(t *testing.T) {
	for _, family := range []string{"run", "task", "review"} {
		t.Run(family+" help word", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runNonInteractive([]string{family, "help"}, &stdout, &stderr); code != 0 {
				t.Fatalf("%s help exit %d: %s", family, code, stderr.String())
			}
			if !strings.Contains(stdout.String(), "Usage: asdx-core "+family) {
				t.Fatalf("%s help output = %q", family, stdout.String())
			}
		})
		t.Run(family+" rejects flag help", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runNonInteractive([]string{family, "--help"}, &stdout, &stderr); code != 2 {
				t.Fatalf("%s --help exit %d, want the documented 2", family, code)
			}
		})
	}
	for _, family := range []string{"fleet", "mcp"} {
		for _, flag := range []string{"--help", "-h"} {
			var stdout, stderr bytes.Buffer
			if code := runNonInteractive([]string{family, flag}, &stdout, &stderr); code != 0 {
				t.Fatalf("%s %s exit %d: %s", family, flag, code, stderr.String())
			}
			if !strings.Contains(stdout.String(), "Usage: asdx-core "+family) {
				t.Fatalf("%s %s output = %q", family, flag, stdout.String())
			}
		}
	}
}
