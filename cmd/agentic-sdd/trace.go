package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
)

// Tracer writes the opt-in flow trace described in Feature 010 FR-07: one
// JSON object per event, on stdout-independent writer, with the common
// schema/run/seq/event fields. A nil *Tracer is a no-op, so disabled tracing
// needs no special code at call sites.
type Tracer struct {
	logger *slog.Logger
	closer io.Closer
	seq    int64
}

// newTracer builds a Tracer over w. dropTime removes the "time" field so a
// test can compare deterministic output; production tracing keeps it.
func newTracer(w io.Writer, closer io.Closer, dropTime bool) *Tracer {
	if w == nil {
		return nil
	}
	runID := "0000000000000000"
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		runID = hex.EncodeToString(b[:])
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) != 0 {
				return a
			}
			switch a.Key {
			case slog.LevelKey:
				return slog.Attr{}
			case slog.TimeKey:
				if dropTime {
					return slog.Attr{}
				}
			case slog.MessageKey:
				a.Key = "event"
			}
			return a
		},
	})
	return &Tracer{
		logger: slog.New(handler).With("schema", 1, "run", runID),
		closer: closer,
	}
}

// Event appends one event to the trace. kv is passed to slog unchanged; the
// documented events never include flag values, free text or filesystem paths.
func (t *Tracer) Event(name string, kv ...any) {
	if t == nil || t.logger == nil {
		return
	}
	seq := atomic.AddInt64(&t.seq, 1)
	t.logger.Info(name, append([]any{"seq", seq}, kv...)...)
}

// Close closes the underlying file once. A nil Tracer or a Tracer built
// without a closer is a no-op.
func (t *Tracer) Close() error {
	if t == nil || t.closer == nil {
		return nil
	}
	closer := t.closer
	t.closer = nil
	return closer.Close()
}

// openTrace opens path for appending as a 0600 regular file. An empty or
// whitespace-only path disables tracing. Every error deliberately omits the
// user-supplied path so the stderr warning can never echo it.
func openTrace(path string) (*Tracer, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	info, err := os.Lstat(path)
	if err == nil && !info.Mode().IsRegular() {
		return nil, errors.New("refusing non-regular trace target")
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.New("cannot inspect trace target")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return nil, errors.New("cannot open trace target")
	}
	// Normalize a pre-existing looser file; permissions passed to OpenFile
	// only apply when the file is created.
	_ = f.Chmod(0600)
	return newTracer(f, f, false), nil
}

// commandName returns the subcommand a run will execute, without recording
// any positional argument or flag value. It understands both "--flag value"
// and "--flag=value" forms.
func commandName(args []string) string {
	sawApply := false
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			name := strings.TrimLeft(arg, "-")
			if bare, _, found := strings.Cut(name, "="); found {
				switch bare {
				case "version":
					return "version"
				case "h", "help":
					return "help"
				case "apply":
					sawApply = true
				}
				continue
			}
			switch name {
			case "version":
				return "version"
			case "h", "help":
				return "help"
			case "apply":
				sawApply = true
			case "home", "repo":
				skipNext = true
			}
			continue
		}
		switch arg {
		case "preview", "apply", "version", "backups", "logs", "restore", "help", "guide", "opencode", "opencode-harness":
			return arg
		}
		return "unknown"
	}
	if sawApply {
		return "apply"
	}
	return "preview"
}

// flagNames returns the bare names of the flags in args. Values are never
// recorded: "--home=/x" contributes "home" and "--home /x" skips "/x".
func flagNames(args []string) []string {
	names := []string{}
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		bare, _, hasValue := strings.Cut(name, "=")
		if bare == "" {
			continue
		}
		names = append(names, bare)
		if !hasValue && (bare == "home" || bare == "repo") {
			skipNext = true
		}
	}
	return names
}
