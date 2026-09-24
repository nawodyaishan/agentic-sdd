// Package version holds build-time metadata injected via -ldflags at release
// build time (see .goreleaser.yml / Makefile). Defaults below are used for
// local `go build`/`go run` where no ldflags are supplied.
package version

var (
	// Version is the semantic version of this build (e.g. "v0.1.0").
	Version = "dev"
	// Commit is the git commit SHA this build was produced from.
	Commit = "none"
	// Date is the build timestamp (RFC3339).
	Date = "unknown"
	// GoVersion is the Go toolchain version used to produce this build.
	GoVersion = "unknown"
)
