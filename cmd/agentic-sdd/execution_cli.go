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
	fs := flag.NewFlagSet(canonicalCLIName+" "+name, flag.ContinueOnError)
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
		fmt.Fprintf(stderr, "note: run %q for the accepted flags\n", canonicalCLIName+" help")
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
// bound. The "FILE" form is explicit and must be a regular file: the byte
// bound is applied to the opened file's own metadata before any allocation
// proportional to its size, and a symlink, directory, device or FIFO is
// refused from its type alone so it can never block or be read. "-" keeps the
// bounded stdin pipeline form.
func readExecutionRecord(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		data, err := io.ReadAll(io.LimitReader(stdin, execution.MaxRecordBytes+1))
		if err != nil {
			return nil, recordReadFailure(err)
		}
		return boundedRecordBytes(data)
	}
	return readExecutionRecordFile(path)
}

// readExecutionRecordFile opens one regular record file under the service
// limit. The path is inspected before opening, and the opened descriptor is
// compared with the inspected file so a replacement cannot slip a different
// object past the type and size checks.
func readExecutionRecordFile(path string) ([]byte, error) {
	inspected, err := os.Lstat(path)
	if err != nil {
		return nil, recordReadFailure(err)
	}
	if inspected.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: read record %s: refusing a symlink; pass the regular file itself or - for stdin", execution.ErrUsage, path)
	}
	if !inspected.Mode().IsRegular() {
		return nil, recordNotRegular(path)
	}
	if inspected.Size() > execution.MaxRecordBytes {
		return nil, recordLimitFailure()
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, recordReadFailure(err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, recordReadFailure(err)
	}
	if !opened.Mode().IsRegular() {
		return nil, recordNotRegular(path)
	}
	if !os.SameFile(inspected, opened) {
		return nil, fmt.Errorf("%w: read record %s: the file changed while it was opened; retry with a stable regular file", execution.ErrUsage, path)
	}
	if opened.Size() > execution.MaxRecordBytes {
		return nil, recordLimitFailure()
	}
	data, err := io.ReadAll(io.LimitReader(file, execution.MaxRecordBytes+1))
	if err != nil {
		return nil, recordReadFailure(err)
	}
	return boundedRecordBytes(data)
}

// boundedRecordBytes applies the limit+1 rule to an already bounded read.
func boundedRecordBytes(data []byte) ([]byte, error) {
	if len(data) > execution.MaxRecordBytes {
		return nil, recordLimitFailure()
	}
	return data, nil
}

// recordReadFailure keeps the diagnostic content-free: it carries only the
// operating-system error, which names the path and the failure, never bytes.
func recordReadFailure(err error) error {
	return fmt.Errorf("%w: read record: %v", execution.ErrUsage, err)
}

func recordNotRegular(path string) error {
	return fmt.Errorf("%w: read record %s: not a regular file; pass a regular file or - for stdin", execution.ErrUsage, path)
}

func recordLimitFailure() error {
	return fmt.Errorf("%w: record exceeds %d bytes", execution.ErrUsage, execution.MaxRecordBytes)
}
