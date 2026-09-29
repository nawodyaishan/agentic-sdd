package main

import (
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

func runLogs(args []string, stdout, stderr io.Writer) int {
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
		return 1
	}
	return 0
}

func runApplyWithLog(repo, home string, stdout, stderr io.Writer) int {
	return runOperationWithLog(repo, home, "apply", stdout, stderr, func(out io.Writer) error {
		return skillsync.Sync(repo, home, true, out)
	})
}

func runRestoreWithLog(repo, home, id string, stdout, stderr io.Writer) int {
	return runOperationWithLog(repo, home, "restore "+id, stdout, stderr, func(out io.Writer) error {
		return skillsync.Restore(repo, home, id, true, out)
	})
}

func runOperationWithLog(repo, home, command string, stdout, stderr io.Writer, execute func(io.Writer) error) int {
	home, err := filepath.Abs(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	logFile, err := createRunLog(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	source := repo
	if source == "" {
		source = "embedded"
	}
	fmt.Fprintf(logFile, "Started: %s\nCommand: %s\nSource: %s\nHome: %s\n", time.Now().UTC().Format(time.RFC3339Nano), command, source, home)
	err = execute(io.MultiWriter(stdout, logFile))
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
	if err != nil || logErr != nil {
		if logErr != nil {
			fmt.Fprintln(stderr, "error: writing log:", logErr)
		}
		return 1
	}
	return 0
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
