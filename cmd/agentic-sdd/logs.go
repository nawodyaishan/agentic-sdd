package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agentic-sdd/internal/skillsync"
)

func runLogs(args []string, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("agentic-sdd logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var home string
	fs.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		env.Hint(stderr, "run agentic-sdd help logs for the accepted flags")
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	home, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	home, err = filepath.Abs(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if err := showLatestLog(home, stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		// Only the two "nothing to show yet" forms get a hint. A rejected
		// log path (symlink or non-directory) is a safety error, not a
		// first-run condition, so it stays bare.
		if os.IsNotExist(err) || strings.HasPrefix(err.Error(), "no operation logs in ") {
			env.Hint(stderr, "logs are written by apply and restore; run agentic-sdd preview first to see what apply would do")
		}
		return 1
	}
	return 0
}

func runApplyWithLog(repo, home string, opts skillsync.Options, stdout, stderr io.Writer, env *Env) int {
	cmdName := "apply"
	if opts.WithOpenCode {
		cmdName += " --with-opencode"
		if opts.OpenCodeFreeFirst {
			cmdName += " --opencode-free-first"
		}
	}
	result := runOperation(repo, home, cmdName, stdout, stderr, env, func(out io.Writer) error {
		return skillsync.SyncWithOptions(repo, home, true, opts, out)
	})
	if result.exitCode == 0 {
		printApplySummary(stdout, env, skillsync.SummarizePlan(result.transcript), result.backupID)
	}
	return result.exitCode
}

// printApplySummary appends the terminal-only apply counts block. It writes to
// stdout only, never to the log file, so the log stays the pre-feature
// transcript.
func printApplySummary(out io.Writer, env *Env, sum skillsync.PlanSummary, backupID string) {
	if env == nil || !env.StdoutTTY {
		return
	}
	fmt.Fprintln(out)
	printHeading(out, env, "Apply complete")
	if brandWidth(env) < brandArtWidth {
		fmt.Fprintf(out, "%s Installed %d items\n", env.Green(statusPrefix(env)), sum.TotalChanges())
		fmt.Fprintf(out, "  %d new, %d replaced\n", sum.Install, sum.Replace)
		if backupID != "" {
			fmt.Fprintln(out, "Backup: "+backupID)
		}
		fmt.Fprintln(out, "  Undo: agentic-sdd restore")
		return
	}
	line := fmt.Sprintf("%s Installed %d items (%d new, %d replaced)",
		env.Green(statusPrefix(env)), sum.TotalChanges(), sum.Install, sum.Replace)
	if backupID != "" {
		line += "; backup: " + backupID
	}
	fmt.Fprintln(out, line)
	fmt.Fprintln(out, "  Undo: agentic-sdd restore")
}

// statusPrefix returns the word "ok" with the glyph allowed by the terminal
// capabilities. Status is always readable as a word, never color alone.
func statusPrefix(env *Env) string {
	if env != nil && env.Color && !env.ASCII {
		return "✔ ok"
	}
	return "ok"
}

func runRestoreWithLog(repo, home, id string, stdout, stderr io.Writer, env *Env) int {
	return runOperationWithLog(repo, home, "restore "+id, stdout, stderr, env, func(out io.Writer) error {
		return skillsync.Restore(repo, home, id, true, out)
	})
}

// operationResult reports the outcome of one logged write operation together
// with what the terminal summary needs; the log file itself is unchanged.
type operationResult struct {
	exitCode   int
	transcript string
	backupID   string
}

func runOperationWithLog(repo, home, command string, stdout, stderr io.Writer, env *Env, execute func(io.Writer) error) int {
	return runOperation(repo, home, command, stdout, stderr, env, execute).exitCode
}

func runOperation(repo, home, command string, stdout, stderr io.Writer, env *Env, execute func(io.Writer) error) operationResult {
	home, err := filepath.Abs(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return operationResult{exitCode: 1}
	}
	logFile, err := createRunLog(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return operationResult{exitCode: 1}
	}
	source := repo
	if source == "" {
		source = "embedded"
	}
	fmt.Fprintf(logFile, "Started: %s\nCommand: %s\nSource: %s\nHome: %s\n", time.Now().UTC().Format(time.RFC3339Nano), command, source, home)
	var buf bytes.Buffer
	err = execute(io.MultiWriter(stdout, logFile, &buf))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		fmt.Fprintln(logFile, "error:", err)
	}
	result := "success"
	if err != nil {
		result = "failure"
	}
	fmt.Fprintf(logFile, "Result: %s\nFinished: %s\n", result, time.Now().UTC().Format(time.RFC3339Nano))
	logErr := errors.Join(logFile.Sync(), logFile.Close())
	fmt.Fprintln(stdout, "Log:", logFile.Name())
	exitCode := 0
	if err != nil || logErr != nil {
		if logErr != nil {
			fmt.Fprintln(stderr, "error: writing log:", logErr)
		}
		exitCode = 1
	}
	opVerb := operationVerb(command)
	backupID := traceBackupID(buf.String())
	if backupID != "" {
		env.Trace.Event("op.result", "op", opVerb, "exit", exitCode, "backup_id", backupID)
	} else {
		env.Trace.Event("op.result", "op", opVerb, "exit", exitCode)
	}
	return operationResult{exitCode: exitCode, transcript: buf.String(), backupID: backupID}
}

// operationVerb reduces a log command string ("apply --with-opencode",
// "restore <id>") to the bare verb the trace schema documents.
func operationVerb(command string) string {
	switch {
	case strings.HasPrefix(command, "apply"):
		return "apply"
	case strings.HasPrefix(command, "restore"):
		return "restore"
	}
	return command
}

// traceBackupID extracts the validated backup basename from an operation
// transcript. It returns "" when the transcript has no usable backup line,
// so a path is never recorded.
func traceBackupID(transcript string) string {
	for _, line := range strings.Split(transcript, "\n") {
		idx := strings.Index(line, "; backup: ")
		if idx == -1 {
			continue
		}
		base := filepath.Base(strings.TrimSpace(line[idx+len("; backup: "):]))
		if skillsync.ValidateBackupID(base) == nil {
			return base
		}
	}
	return ""
}

func logsDir(home string) string {
	return filepath.Join(home, ".agentic-sdd", "logs")
}

func checkLogPath(home string) error {
	for _, path := range []string{home, filepath.Join(home, ".agentic-sdd"), logsDir(home)} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("refusing non-directory log path %s", path)
		}
	}
	return nil
}

func createRunLog(home string) (*os.File, error) {
	if err := checkLogPath(home); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(logsDir(home), 0700); err != nil {
		return nil, err
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	return os.CreateTemp(logsDir(home), stamp+"-*.log")
}

func showLatestLog(home string, out io.Writer) error {
	if err := checkLogPath(home); err != nil {
		return err
	}
	entries, err := os.ReadDir(logsDir(home))
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && strings.HasSuffix(entry.Name(), ".log") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return fmt.Errorf("no operation logs in %s", logsDir(home))
	}
	sort.Strings(names)
	path := filepath.Join(logsDir(home), names[len(names)-1])
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := fmt.Fprintln(out, "Log:", path); err != nil {
		return err
	}
	_, err = io.Copy(out, file)
	return err
}
