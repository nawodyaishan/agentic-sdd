package main

import "agentic-sdd/internal/buildinfo"

// buildIdentity keeps the release-injected pair identity linked into the CLI
// before T14 wires the TUI companion handshake. It has no runtime behavior and
// does not change any CLI output. The reference is what makes the -X values in
// .goreleaser.yml part of the released CLI binary, so the parent and companion
// can prove they were built from the same source.
var buildIdentity = buildinfo.Identity{
	Revision:     buildinfo.Revision,
	Commit:       buildinfo.Commit,
	SourceDigest: buildinfo.SourceDigest,
}

func init() { _ = buildIdentity }
