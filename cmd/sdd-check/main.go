// sdd-check executes a trusted deterministic check contract, never an agent loop.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agentic-sdd/internal/harnesscheck"
)

type checkIDs []string

func (ids *checkIDs) String() string         { return fmt.Sprint([]string(*ids)) }
func (ids *checkIDs) Set(value string) error { *ids = append(*ids, value); return nil }

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sdd-check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var options harnesscheck.Options
	var ids checkIDs
	fs.StringVar(&options.Workspace, "workspace", ".", "workspace whose checks will execute")
	fs.StringVar(&options.ContractPath, "contract", "harness/checks.json", "trusted check contract (relative to invocation directory)")
	fs.StringVar(&options.Output, "output", "", "new or empty private evidence directory outside workspace")
	fs.StringVar(&options.Scope, "scope", "", "baseline, focused or required (explicit)")
	fs.Var(&ids, "check", "check ID; repeat for focused scope only")
	fs.DurationVar(&options.Deadline, "deadline", 15*time.Minute, "positive whole-run deadline")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: sdd-check --scope baseline|focused|required --output <external-directory> [options]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(stdout)
			fs.Usage()
			return 0
		}
		fmt.Fprintln(stderr, "sdd-check:", err)
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "unexpected positional arguments")
		return 2
	}
	options.CheckIDs = ids
	result, err := harnesscheck.Run(ctx, options)
	if err != nil {
		fmt.Fprintln(stderr, "sdd-check:", err)
	}
	if encodeErr := json.NewEncoder(stdout).Encode(result); encodeErr != nil {
		fmt.Fprintln(stderr, "write result:", encodeErr)
		return 1
	}
	return harnesscheck.ExitCode(result, err)
}
