//go:build windows

package main

import "errors"

// execLegacyTarget fails closed on platforms without process replacement. The
// compatibility build never falls back to a child process it cannot fully
// replace, because that would change stream and signal behavior.
func execLegacyTarget(string, []string) error {
	return errors.New("legacy compatibility mode needs a POSIX process replacement; run the canonical binary directly")
}
