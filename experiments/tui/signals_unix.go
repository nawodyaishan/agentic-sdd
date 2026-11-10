//go:build unix

package tui

// This file owns the supported-Unix OS shutdown signal set. The nested TUI
// replaces Bubble Tea's default handler explicitly with
// tea.WithoutSignalHandler() and registers these signals before Program.Run,
// so SIGINT, SIGTERM and SIGHUP all follow the same busy-state policy as
// Ctrl+C. SIGHUP is deliberately included: closing a terminal or losing an SSH
// session must not abandon an in-flight write.

import (
	"os"
	"syscall"
)

// unixShutdownSignals records that this build registers the full Unix set.
const unixShutdownSignals = true

// shutdownSignals returns the OS signals this build forwards as typed shutdown
// requests, in a stable order.
func shutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
}

// shutdownSignalName labels one received signal for display and tests.
func shutdownSignalName(sig os.Signal) string {
	switch sig {
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGHUP:
		return "SIGHUP"
	default:
		return sig.String()
	}
}
