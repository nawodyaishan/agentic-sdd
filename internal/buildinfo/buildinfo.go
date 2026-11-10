// Package buildinfo defines the narrow build-time identity contract shared by
// the agentic-sdd CLI and its TUI companion, plus the private parent/companion
// protocol-info handshake built on it.
//
// The values below are injected with -ldflags at build time (see
// .goreleaser.yml and Taskfile.experimental.yml). A source build that supplies
// no ldflags keeps honest development defaults; the handshake in
// VerifyCompanion deliberately rejects those defaults instead of silently
// pairing two unidentifiable binaries.
package buildinfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// ProtocolRevision is the parent/companion protocol-info contract revision.
	// Bump it whenever a field's meaning or the handshake rules change
	// incompatibly.
	ProtocolRevision = 1

	// ProductCLI and ProductCompanion identify the two processes that take part
	// in the handshake. They are protocol labels, not filesystem paths.
	ProductCLI       = "agentic-sdd"
	ProductCompanion = "agentic-sdd-tui"

	// RebuildPairHint is appended to every identity failure. There is no safe
	// fallback: a mismatched or unidentifiable pair must be rebuilt together.
	RebuildPairHint = "rebuild both binaries as a matched pair so they receive the same injected revision, commit and source digest"
)

// Distribution is the explicit packaging marker of one artifact. It is
// provenance, not part of the parent/companion identity match: the Homebrew
// CLI and the shared companion deliberately carry different markers.
type Distribution string

const (
	// DistributionDev is the honest default for an uninjected source build.
	DistributionDev Distribution = "dev"
	// DistributionGeneric marks prebuilt archives and source-built releases
	// that keep the historical CLI/default-preview policy.
	DistributionGeneric Distribution = "generic"
	// DistributionHomebrew marks only the Homebrew-based CLI artifact, which
	// may open the TUI for a zero-argument interactive launch.
	DistributionHomebrew Distribution = "homebrew"
)

// Valid reports whether d is one of the three reviewed markers.
func (d Distribution) Valid() bool {
	switch d {
	case DistributionDev, DistributionGeneric, DistributionHomebrew:
		return true
	}
	return false
}

// AllowsInteractiveDefault reports whether this distribution may replace a
// zero-argument interactive invocation with the TUI. Only the explicitly
// Homebrew-marked CLI opts in; generic archives, source builds and unknown
// markers keep the historical preview behavior.
func (d Distribution) AllowsInteractiveDefault() bool {
	return d == DistributionHomebrew
}

// These variables are the link-time injection points. Keep their names stable:
// .goreleaser.yml and the local paired-build task address them with
// -X agentic-sdd/internal/buildinfo.<Name>=<value>.
var (
	// Revision is the release version or a development marker such as
	// "dev+1a2b3c4d5e6f". The "dev" default means no ldflags were injected.
	Revision = "dev"
	// Commit is the full source commit SHA. "unknown" means uninjected.
	Commit = "unknown"
	// SourceDigest identifies the exact source content, including a dirty
	// tree for local paired builds, for example "local-sha256:<hex>". "unknown"
	// means uninjected.
	SourceDigest = "unknown"
	// DistributionMarker is the packaging provenance; see Distribution.
	DistributionMarker = string(DistributionDev)
)

// Identity is the part of the build identity that must match across a parent
// CLI and its companion. Distribution is intentionally excluded.
type Identity struct {
	Revision     string
	Commit       string
	SourceDigest string
}

// CurrentIdentity returns the identity injected into this binary.
func CurrentIdentity() Identity {
	return Identity{
		Revision:     Revision,
		Commit:       Commit,
		SourceDigest: SourceDigest,
	}
}

// CurrentDistribution returns the injected packaging marker.
func CurrentDistribution() Distribution {
	return Distribution(DistributionMarker)
}

// Validate reports whether this identity is complete and was actually
// injected. It never treats a development placeholder as verifiable.
func (i Identity) Validate() error {
	var missing []string
	if placeholder(i.Revision) {
		missing = append(missing, "revision")
	}
	if placeholder(i.Commit) {
		missing = append(missing, "commit")
	}
	if placeholder(i.SourceDigest) {
		missing = append(missing, "source digest")
	}
	if len(missing) > 0 {
		return fmt.Errorf("build identity is not verifiable (%s missing or still a development placeholder); %s", strings.Join(missing, ", "), RebuildPairHint)
	}
	return nil
}

// placeholder reports the honest defaults and other non-identifying values.
func placeholder(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "dev", "none", "unknown":
		return true
	}
	return false
}

// String renders the identity deterministically for diagnostics.
func (i Identity) String() string {
	return fmt.Sprintf("revision=%s commit=%s source_digest=%s", i.Revision, i.Commit, i.SourceDigest)
}

// Equal compares every pair-identity field.
func (i Identity) Equal(other Identity) bool {
	return i.Revision == other.Revision && i.Commit == other.Commit && i.SourceDigest == other.SourceDigest
}

// ProtocolInfo is the stable, non-interactive response of the companion's
// --protocol-info query. Field order is part of the wire format; add fields
// only with a ProtocolRevision bump.
type ProtocolInfo struct {
	Product          string `json:"product"`
	ProtocolRevision int    `json:"protocol_revision"`
	Distribution     string `json:"distribution"`
	Revision         string `json:"revision"`
	Commit           string `json:"commit"`
	SourceDigest     string `json:"source_digest"`
}

// NewProtocolInfo builds one protocol-info value from explicit inputs. Tests
// and callers that must not depend on link-time state use this constructor.
func NewProtocolInfo(product string, distribution Distribution, identity Identity) ProtocolInfo {
	return ProtocolInfo{
		Product:          product,
		ProtocolRevision: ProtocolRevision,
		Distribution:     string(distribution),
		Revision:         identity.Revision,
		Commit:           identity.Commit,
		SourceDigest:     identity.SourceDigest,
	}
}

// CurrentProtocolInfo builds this binary's protocol-info response.
func CurrentProtocolInfo(product string) ProtocolInfo {
	return NewProtocolInfo(product, CurrentDistribution(), CurrentIdentity())
}

// Identity returns the pair-identity fields carried by the response.
func (p ProtocolInfo) Identity() Identity {
	return Identity{Revision: p.Revision, Commit: p.Commit, SourceDigest: p.SourceDigest}
}

// DistributionValue returns the packaging marker as a typed value.
func (p ProtocolInfo) DistributionValue() Distribution {
	return Distribution(p.Distribution)
}

// JSON renders the response as one stable JSON line. It performs no network or
// filesystem access.
func (p ProtocolInfo) JSON() ([]byte, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("render protocol info: %w", err)
	}
	return append(data, '\n'), nil
}

// ParseProtocolInfo strictly decodes one --protocol-info response. Unknown
// fields, trailing data and multiple documents fail closed.
func ParseProtocolInfo(data []byte) (ProtocolInfo, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var info ProtocolInfo
	if err := decoder.Decode(&info); err != nil {
		return ProtocolInfo{}, fmt.Errorf("parse companion protocol info: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ProtocolInfo{}, fmt.Errorf("parse companion protocol info: trailing data")
	}
	return info, nil
}

// VerifyCompanion applies the parent-side handshake. It is a compatibility
// check, not authentication or write authorization. Every failure is explicit:
// an absent, unknown, stale or mismatched identity never falls back silently.
func VerifyCompanion(parent, peer ProtocolInfo) error {
	if peer.Product != ProductCompanion {
		return fmt.Errorf("companion reported product %q, want %q; %s", peer.Product, ProductCompanion, RebuildPairHint)
	}
	if peer.ProtocolRevision != ProtocolRevision {
		return fmt.Errorf("companion protocol revision %d does not match this CLI's revision %d; %s", peer.ProtocolRevision, ProtocolRevision, RebuildPairHint)
	}
	if !parent.DistributionValue().Valid() {
		return fmt.Errorf("this CLI has an unknown distribution marker %q; %s", parent.Distribution, RebuildPairHint)
	}
	if !peer.DistributionValue().Valid() {
		return fmt.Errorf("companion reported an unknown distribution marker %q; %s", peer.Distribution, RebuildPairHint)
	}
	if err := parent.Identity().Validate(); err != nil {
		return fmt.Errorf("this CLI cannot verify the companion: %w", err)
	}
	if err := peer.Identity().Validate(); err != nil {
		return fmt.Errorf("companion identity is not verifiable: %w", err)
	}
	if !parent.Identity().Equal(peer.Identity()) {
		return fmt.Errorf("companion identity (%s) does not match this CLI (%s); %s", peer.Identity(), parent.Identity(), RebuildPairHint)
	}
	return nil
}
