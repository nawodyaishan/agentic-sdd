// Package skillsfiles embeds the canonical skills-files/ tree so a binary
// built via `go build`/GoReleaser (e.g. installed through Homebrew) works
// without a checkout of this repository alongside it. skills-files/ sits at
// the module root, so this file has to live here too: go:embed can only
// reach directories at or below the embedding file's own directory.
package skillsfiles

import "embed"

//go:embed all:skills-files
var Files embed.FS
