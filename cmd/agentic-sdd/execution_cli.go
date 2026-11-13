package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"agentic-sdd/internal/execution"
)

// executionCommonFlags are shared by the new run/task/review command families.
// They are deliberately plain: no model, network, prompt or credential input.
type executionCommonFlags struct {
	repo    string
	runRoot string
	format  string
}

func newExecutionFlagSet(name string, common *executionCommonFlags) *flag.FlagSet {
	fs := flag.NewFlagSet("agentic-sdd "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&common.repo, "repo", "", "repository path (default: current working tree)")
	fs.StringVar(&common.runRoot, "run-root", "", "private execution run root")
	fs.StringVar(&common.format, "format", "text", "output format: text or json")
	return fs
}

// parseExecutionFlags keeps the existing CLI convention: --help prints usage
// and succeeds, flag/positional errors exit 2.
func parseExecutionFlags(fs *flag.FlagSet, args []string, usage func(io.Writer), stdout, stderr io.Writer) (bool, int) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(stdout)
			return false, 0
		}
		fmt.Fprintln(stderr, "error:", err)
		fmt.Fprintln(stderr, "note: run \"agentic-sdd help\" for the accepted flags")
		return false, 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return false, 2
	}
	return true, 0
}

func parseExecutionFormat(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "text":
		return "text", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("unsupported format %q (use text or json)", value)
	}
}

// executionErrorCode maps the typed service errors to the documented public
// exit codes. A missing run for status is a runtime observation failure (1);
// a missing task/batch is invalid input (2).
func executionErrorCode(err error, status bool) int {
	switch {
	case errors.Is(err, execution.ErrUsage),
		errors.Is(err, execution.ErrUnsafePath),
		errors.Is(err, execution.ErrConflict):
		return 2
	case errors.Is(err, execution.ErrNotFound):
		if status {
			return 1
		}
		return 2
	case errors.Is(err, execution.ErrPartial):
		return 3
	default:
		return 1
	}
}

func writeExecutionJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

// resolveExecutionRepo opens the selected repository. No write occurs.
func resolveExecutionRepo(common *executionCommonFlags, stderr io.Writer) (*execution.Repository, int) {
	repo, err := execution.OpenRepository(common.repo)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return nil, executionErrorCode(err, false)
	}
	return repo, 0
}

// resolveExecutionRunDir resolves an existing run identity path without
// creating anything. Service calls still validate run.json and ownership.
func resolveExecutionRunDir(repo *execution.Repository, runRoot, runID string) (string, error) {
	if strings.TrimSpace(runID) == "" {
		return "", fmt.Errorf("%w: --run is required", execution.ErrUsage)
	}
	if err := execution.ValidateRunID(runID); err != nil {
		return "", err
	}
	root, err := execution.ResolveRunRoot(repo.Root, runRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, runID), nil
}

// readExecutionRecord reads a caller-supplied record with the service's size
// bound. The "FILE" form is explicit; "-" reads stdin for pipelines.
func readExecutionRecord(path string, stdin io.Reader) ([]byte, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(io.LimitReader(stdin, execution.MaxRecordBytes+1))
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: read record: %v", execution.ErrUsage, err)
	}
	if len(data) > execution.MaxRecordBytes {
		return nil, fmt.Errorf("%w: record exceeds %d bytes", execution.ErrUsage, execution.MaxRecordBytes)
	}
	return data, nil
}
