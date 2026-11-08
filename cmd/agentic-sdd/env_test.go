package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeFileInfo is a minimal os.FileInfo for terminal tests. Sys returns nil so
// os.SameFile never mistakes a fake character device for the null device.
type fakeFileInfo struct {
	mode os.FileMode
}

func (fi fakeFileInfo) Name() string       { return "fake" }
func (fi fakeFileInfo) Size() int64        { return 0 }
func (fi fakeFileInfo) Mode() os.FileMode  { return fi.mode }
func (fi fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fi fakeFileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi fakeFileInfo) Sys() any           { return nil }

type fakeStatter struct {
	info os.FileInfo
	err  error
}

func (f fakeStatter) Stat() (os.FileInfo, error) { return f.info, f.err }

func fakeCharDevice() statter  { return fakeStatter{info: fakeFileInfo{mode: os.ModeCharDevice}} }
func fakeRegularFile() statter { return fakeStatter{info: fakeFileInfo{mode: 0}} }

func mapGetenv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestIsTerminal(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()

	temp, err := os.CreateTemp(t.TempDir(), "regular")
	if err != nil {
		t.Fatal(err)
	}
	defer temp.Close()

	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pipeR.Close()
	defer pipeW.Close()

	tests := []struct {
		name string
		f    statter
		want bool
	}{
		{"fake character device", fakeCharDevice(), true},
		{"nil statter", nil, false},
		{"null device", devNull, false},
		{"regular temp file", temp, false},
		{"pipe reader", pipeR, false},
		{"pipe writer", pipeW, false},
		{"stat error", fakeStatter{err: errors.New("stat failed")}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTerminal(tc.f); got != tc.want {
				t.Fatalf("isTerminal(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestResolveEnv(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()

	term := fakeCharDevice()
	notTerm := fakeRegularFile()
	allTerminal := Env{Interactive: true, StdoutTTY: true, StderrTTY: true, Color: true, Width: 80}

	tests := []struct {
		name   string
		stdin  statter
		stdout statter
		stderr statter
		env    map[string]string
		want   Env
	}{
		{
			name:   "non-terminal defaults",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			want:   Env{Width: 80},
		},
		{
			name:   "all terminals",
			stdin:  term,
			stdout: term,
			stderr: term,
			want:   allTerminal,
		},
		{
			name:   "null device stdin is not interactive",
			stdin:  devNull,
			stdout: term,
			stderr: term,
			want:   Env{StdoutTTY: true, StderrTTY: true, Color: true, Width: 80},
		},
		{
			name:   "stdout not a terminal disables color",
			stdin:  term,
			stdout: notTerm,
			stderr: term,
			want:   Env{Interactive: true, StderrTTY: true, Width: 80},
		},
		{
			name:   "stderr not a terminal keeps stdout color",
			stdin:  term,
			stdout: term,
			stderr: notTerm,
			want:   Env{Interactive: true, StdoutTTY: true, Color: true, Width: 80},
		},
		{
			name:   "NO_COLOR disables color",
			stdin:  term,
			stdout: term,
			stderr: term,
			env:    map[string]string{"NO_COLOR": "1"},
			want:   Env{Interactive: true, StdoutTTY: true, StderrTTY: true, Width: 80},
		},
		{
			name:   "TERM=dumb sets ASCII and disables color",
			stdin:  term,
			stdout: term,
			stderr: term,
			env:    map[string]string{"TERM": "dumb"},
			want:   Env{Interactive: true, StdoutTTY: true, StderrTTY: true, ASCII: true, Width: 80},
		},
		{
			name:   "COLUMNS sets width",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"COLUMNS": "120"},
			want:   Env{Width: 120},
		},
		{
			name:   "COLUMNS with surrounding spaces",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"COLUMNS": " 132 "},
			want:   Env{Width: 132},
		},
		{
			name:   "non-integer COLUMNS defaults to 80",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"COLUMNS": "wide"},
			want:   Env{Width: 80},
		},
		{
			name:   "negative COLUMNS defaults to 80",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"COLUMNS": "-5"},
			want:   Env{Width: 80},
		},
		{
			name:   "zero COLUMNS defaults to 80",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"COLUMNS": "0"},
			want:   Env{Width: 80},
		},
		{
			name:   "force interactive treats non-terminals as terminals",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"AGENTIC_SDD_FORCE_INTERACTIVE": "1"},
			want:   allTerminal,
		},
		{
			name:   "force interactive still honors NO_COLOR",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"AGENTIC_SDD_FORCE_INTERACTIVE": "1", "NO_COLOR": "1"},
			want:   Env{Interactive: true, StdoutTTY: true, StderrTTY: true, Width: 80},
		},
		{
			name:   "force interactive still honors TERM=dumb",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"AGENTIC_SDD_FORCE_INTERACTIVE": "1", "TERM": "dumb"},
			want:   Env{Interactive: true, StdoutTTY: true, StderrTTY: true, ASCII: true, Width: 80},
		},
		{
			name:   "force interactive requires exactly 1",
			stdin:  notTerm,
			stdout: notTerm,
			stderr: notTerm,
			env:    map[string]string{"AGENTIC_SDD_FORCE_INTERACTIVE": "true"},
			want:   Env{Width: 80},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveEnv(tc.stdin, tc.stdout, tc.stderr, mapGetenv(tc.env))
			if got != tc.want {
				t.Fatalf("resolveEnv() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestRunEnvRestoreWithDevNullStdin proves the fixed resolver feeds the
// documented non-interactive restore path: /dev/null stdin is not interactive,
// so no picker is shown and the run exits 2 with the required message.
func TestRunEnvRestoreWithDevNullStdin(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()

	env := resolveEnv(devNull, devNull, devNull, mapGetenv(nil))
	var out, errOut bytes.Buffer
	code := runEnv([]string{"restore", "--home", t.TempDir()}, &out, &errOut, devNull, &env)
	if code != 2 {
		t.Fatalf("exit %d, want 2; stdout %q; stderr %q", code, out.String(), errOut.String())
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty (no picker)", out.String())
	}
	if !strings.Contains(errOut.String(), "restore requires a backup id") {
		t.Fatalf("stderr = %q, want missing-id message", errOut.String())
	}
}
