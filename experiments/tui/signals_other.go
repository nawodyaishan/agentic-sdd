//go:build !unix

package tui

// Non-Unix fallback. Windows and other non-Unix targets do not provide
// SIGHUP/SIGTERM through os/signal reliably, so the host registers only the
// portable interrupt signal and documents the narrower guarantee: an idle
// build exits with terminal cleanup, while a write still follows the same
// exit-after-completion policy for that signal. Ctrl+C key handling is
// unchanged on every platform. This file exists so all builds compile; the
// repository's PTY lifecycle evidence is Unix-only and is never claimed here.

import "os"

// unixShutdownSignals records that this build lacks the full Unix set.
const unixShutdownSignals = false

// shutdownSignals returns the portable signal this build forwards.
func shutdownSignals() []os.Signal { return []os.Signal{os.Interrupt} }

// shutdownSignalName labels one received signal for display and tests.
func shutdownSignalName(sig os.Signal) string {
	return sig.String()
}
