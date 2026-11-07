//go:build !agentic_sdd_experimental

package main

import (
	"fmt"
	"io"
)

func printOpenCodeUsage(out io.Writer) {
	fmt.Fprintln(out, "The opencode helper is unavailable in this build.")
	fmt.Fprintln(out, "Rebuild with -tags agentic_sdd_experimental to enable experimental opencode commands.")
}

func runOpenCode(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "error: opencode: unavailable in this build (requires -tags agentic_sdd_experimental)")
	return 2
}
