package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// statter is the small part of *os.File that terminal detection needs, so the
// resolver can be exercised with fakes in tests.
type statter interface {
	Stat() (os.FileInfo, error)
}

// Env is the per-run environment decided once at startup. It is nil-safe for
// presentation helpers: a nil *Env behaves like a plain, non-terminal run.
type Env struct {
	Interactive bool    // stdin allows prompts
	StdoutTTY   bool    // stdout is a terminal
	StderrTTY   bool    // stderr is a terminal
	Color       bool    // stdout styling is on
	ASCII       bool    // TERM=dumb: ASCII-only glyphs
	Width       int     // terminal width, COLUMNS or 80
	Trace       *Tracer // nil = tracing disabled
}

// Hint writes an additive "hint: <msg>" line to stderr, but only when stderr
// is a terminal. It never changes the preceding error line or the exit code.
func (e *Env) Hint(stderr io.Writer, msg string) {
	if e == nil || !e.StderrTTY {
		return
	}
	fmt.Fprintf(stderr, "hint: %s\n", msg)
}

// style wraps s in an ANSI SGR sequence only when color is enabled.
func (e *Env) style(code, s string) string {
	if e == nil || !e.Color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// Bold returns s in bold when color is enabled, s unchanged otherwise.
func (e *Env) Bold(s string) string { return e.style("1", s) }

// Dim returns s dimmed when color is enabled, s unchanged otherwise.
func (e *Env) Dim(s string) string { return e.style("2", s) }

// Green returns s in green when color is enabled, s unchanged otherwise.
func (e *Env) Green(s string) string { return e.style("32", s) }

// Red returns s in red when color is enabled, s unchanged otherwise.
func (e *Env) Red(s string) string { return e.style("31", s) }

// isTerminal reports whether f is an actual terminal. It treats a character
// device as a terminal except for the null device, which also reports
// os.ModeCharDevice but is not interactive (the pre-feature bug behind
// "restore </dev/null" showing a prompt).
func isTerminal(f statter) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil || info == nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if nullInfo, err := os.Stat(os.DevNull); err == nil && os.SameFile(info, nullInfo) {
		return false
	}
	return true
}

// resolveEnv decides interaction, terminal, color, ASCII and width for one
// run. getenv is injected so tests never mutate the process environment.
func resolveEnv(stdin, stdout, stderr statter, getenv func(string) string) Env {
	inTTY := isTerminal(stdin)
	outTTY := isTerminal(stdout)
	errTTY := isTerminal(stderr)
	if getenv("AGENTIC_SDD_FORCE_INTERACTIVE") == "1" {
		inTTY, outTTY, errTTY = true, true, true
	}
	term := getenv("TERM")
	ascii := term == "dumb"
	color := outTTY && getenv("NO_COLOR") == "" && !ascii
	width := 80
	if cols, err := strconv.Atoi(strings.TrimSpace(getenv("COLUMNS"))); err == nil && cols > 0 {
		width = cols
	}
	return Env{
		Interactive: inTTY,
		StdoutTTY:   outTTY,
		StderrTTY:   errTTY,
		Color:       color,
		ASCII:       ascii,
		Width:       width,
	}
}
