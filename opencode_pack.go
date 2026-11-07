// Package skillsfiles embeds the opt-in OpenCode orchestration pack so a
// binary built via `go build`/GoReleaser can install it with
// `--with-opencode` without a repository checkout alongside it.
//
// opencode-pack/ is deliberately separate from skills-files/ (the 18 default
// core skills) and from integrations/opencode/ (the legacy Feature 005/006/007
// experimental harness resources). Embedding these files installs nothing on
// its own: only an explicit `agentic-sdd apply --with-opencode` writes them.
package skillsfiles

import "embed"

// OpenCodePackFiles holds the opt-in orchestration skill, the agent profiles
// and documentation served by `agentic-sdd --with-opencode`.
//
//go:embed all:opencode-pack
var OpenCodePackFiles embed.FS
