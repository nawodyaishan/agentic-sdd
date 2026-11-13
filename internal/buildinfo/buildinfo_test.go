package buildinfo_test

import (
	"os"
	"strings"
	"testing"

	"agentic-sdd/internal/buildinfo"
)

func TestDefaultsAreHonestAndUnverifiable(t *testing.T) {
	if buildinfo.Revision != "dev" {
		t.Fatalf("Revision default = %q, want %q", buildinfo.Revision, "dev")
	}
	if buildinfo.Commit != "unknown" {
		t.Fatalf("Commit default = %q, want %q", buildinfo.Commit, "unknown")
	}
	if buildinfo.SourceDigest != "unknown" {
		t.Fatalf("SourceDigest default = %q, want %q", buildinfo.SourceDigest, "unknown")
	}
	if got := buildinfo.CurrentDistribution(); got != buildinfo.DistributionDev {
		t.Fatalf("CurrentDistribution default = %q, want %q", got, buildinfo.DistributionDev)
	}
	if buildinfo.ProtocolRevision != 2 {
		t.Fatalf("ProtocolRevision = %d, want 2", buildinfo.ProtocolRevision)
	}
	if buildinfo.ProductCLI != "asdx-core" {
		t.Fatalf("ProductCLI = %q, want %q", buildinfo.ProductCLI, "asdx-core")
	}
	if buildinfo.ProductCompanion != "asdx" {
		t.Fatalf("ProductCompanion = %q, want %q", buildinfo.ProductCompanion, "asdx")
	}

	identity := buildinfo.CurrentIdentity()
	if err := identity.Validate(); err == nil {
		t.Fatal("Validate accepted the uninjected development defaults")
	} else if !strings.Contains(err.Error(), buildinfo.RebuildPairHint) {
		t.Fatalf("Validate error %q does not carry the rebuild-pair hint", err)
	}
}

func TestIdentityValidate(t *testing.T) {
	valid := buildinfo.Identity{Revision: "v0.3.0", Commit: "0123456789abcdef", SourceDigest: "git:0123456789abcdef"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(%+v) = %v, want nil", valid, err)
	}

	tests := []struct {
		name string
		id   buildinfo.Identity
		want string
	}{
		{
			name: "missing revision",
			id:   buildinfo.Identity{Revision: "", Commit: valid.Commit, SourceDigest: valid.SourceDigest},
			want: "revision",
		},
		{
			name: "dev revision only",
			id:   buildinfo.Identity{Revision: "dev", Commit: valid.Commit, SourceDigest: valid.SourceDigest},
			want: "revision",
		},
		{
			name: "missing commit",
			id:   buildinfo.Identity{Revision: valid.Revision, Commit: "none", SourceDigest: valid.SourceDigest},
			want: "commit",
		},
		{
			name: "unknown digest",
			id:   buildinfo.Identity{Revision: valid.Revision, Commit: valid.Commit, SourceDigest: "unknown"},
			want: "source digest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.id.Validate()
			if err == nil {
				t.Fatalf("Validate(%+v) = nil, want error", tt.id)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate error %q does not name %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), buildinfo.RebuildPairHint) {
				t.Fatalf("Validate error %q does not carry the rebuild-pair hint", err)
			}
		})
	}
}

func TestIdentityEqualAndString(t *testing.T) {
	a := buildinfo.Identity{Revision: "v0.3.0", Commit: "abc", SourceDigest: "local-sha256:def"}
	b := a
	if !a.Equal(b) {
		t.Fatal("Equal returned false for identical identities")
	}
	b.Commit = "different"
	if a.Equal(b) {
		t.Fatal("Equal returned true for different commits")
	}
	want := "revision=v0.3.0 commit=abc source_digest=local-sha256:def"
	if got := a.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestDistributionMatrix(t *testing.T) {
	tests := []struct {
		marker buildinfo.Distribution
		valid  bool
	}{
		{buildinfo.DistributionDev, true},
		{buildinfo.DistributionGeneric, true},
		{buildinfo.Distribution("homebrew"), false},
		{buildinfo.Distribution("unknown"), false},
		{buildinfo.Distribution(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.marker), func(t *testing.T) {
			if got := tt.marker.Valid(); got != tt.valid {
				t.Fatalf("%q.Valid() = %v, want %v", tt.marker, got, tt.valid)
			}
		})
	}
}

func TestFleetCapabilityDefaultsToPreviewAndFailsSafe(t *testing.T) {
	original := buildinfo.FleetCapabilityMarker
	t.Cleanup(func() { buildinfo.FleetCapabilityMarker = original })

	if got := buildinfo.CurrentFleetCapability(); got != buildinfo.FleetCapabilityPreview {
		t.Fatalf("CurrentFleetCapability default = %q, want %q", got, buildinfo.FleetCapabilityPreview)
	}
	if buildinfo.FleetLiveEnabled() {
		t.Fatal("FleetLiveEnabled default = true, want false so an uninjected build stays preview-only")
	}

	buildinfo.FleetCapabilityMarker = string(buildinfo.FleetCapabilityLive)
	if !buildinfo.FleetLiveEnabled() {
		t.Fatal("FleetLiveEnabled = false for an explicit live marker")
	}

	for _, invalid := range []string{"", "LIVE", "Live", "true", "1", "yes", "preview-only", " live "} {
		buildinfo.FleetCapabilityMarker = invalid
		if got := buildinfo.CurrentFleetCapability(); got != buildinfo.FleetCapabilityPreview {
			t.Fatalf("CurrentFleetCapability(%q) = %q, want the preview fallback", invalid, got)
		}
		if buildinfo.FleetLiveEnabled() {
			t.Fatalf("FleetLiveEnabled(%q) = true; an invalid marker must never enable live writes", invalid)
		}
	}
}

func TestFleetCapabilityMatrix(t *testing.T) {
	tests := []struct {
		marker buildinfo.FleetCapability
		valid  bool
		live   bool
	}{
		{buildinfo.FleetCapabilityPreview, true, false},
		{buildinfo.FleetCapabilityLive, true, true},
		{buildinfo.FleetCapability(""), false, false},
		{buildinfo.FleetCapability("preview-only"), false, false},
		{buildinfo.FleetCapability("live "), false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.marker), func(t *testing.T) {
			if got := tt.marker.Valid(); got != tt.valid {
				t.Fatalf("%q.Valid() = %v, want %v", tt.marker, got, tt.valid)
			}
			if got := tt.marker.AllowsLiveFleet(); got != tt.live {
				t.Fatalf("%q.AllowsLiveFleet() = %v, want %v", tt.marker, got, tt.live)
			}
		})
	}
}

// TestReleaseConfigKeepsFleetPreviewOnly guards FR-20/AC-36: the released
// capability is a link-time value that no release configuration may flip to
// live. Enabling it is a separate release decision after Feature 014
// acceptance.
func TestReleaseConfigKeepsFleetPreviewOnly(t *testing.T) {
	for _, path := range []string{"../../.goreleaser.yml", "../../.github/workflows/release.yml"} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		config := string(source)
		if strings.Contains(config, "FleetCapability") {
			t.Fatalf("%s references the fleet capability; the released default must stay preview", path)
		}
	}
}

func TestProtocolInfoJSONIsStable(t *testing.T) {
	info := buildinfo.NewProtocolInfo(
		buildinfo.ProductCompanion,
		buildinfo.DistributionDev,
		buildinfo.Identity{Revision: "v0.3.0", Commit: "abc123", SourceDigest: "local-sha256:deadbeef"},
	)
	data, err := info.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}
	want := `{"product":"asdx","protocol_revision":2,"distribution":"dev","revision":"v0.3.0","commit":"abc123","source_digest":"local-sha256:deadbeef"}` + "\n"
	if string(data) != want {
		t.Fatalf("JSON() = %q, want %q", data, want)
	}

	parsed, err := buildinfo.ParseProtocolInfo(data)
	if err != nil {
		t.Fatalf("ParseProtocolInfo() error: %v", err)
	}
	if parsed != info {
		t.Fatalf("ParseProtocolInfo() = %+v, want %+v", parsed, info)
	}
}

func TestParseProtocolInfoRejectsUnknownAndTrailingData(t *testing.T) {
	valid := `{"product":"asdx","protocol_revision":2,"distribution":"generic","revision":"v1","commit":"c","source_digest":"d"}`
	if _, err := buildinfo.ParseProtocolInfo([]byte(valid)); err != nil {
		t.Fatalf("ParseProtocolInfo(valid) error: %v", err)
	}
	unknown := `{"product":"asdx","protocol_revision":2,"distribution":"generic","revision":"v1","commit":"c","source_digest":"d","token":"secret"}`
	if _, err := buildinfo.ParseProtocolInfo([]byte(unknown)); err == nil {
		t.Fatal("ParseProtocolInfo accepted an unknown field")
	}
	trailing := valid + "{}"
	if _, err := buildinfo.ParseProtocolInfo([]byte(trailing)); err == nil {
		t.Fatal("ParseProtocolInfo accepted trailing data")
	}
}

func TestVerifyCompanion(t *testing.T) {
	identity := buildinfo.Identity{Revision: "v0.3.0", Commit: "0123456789abcdef", SourceDigest: "local-sha256:feedface"}
	parent := buildinfo.NewProtocolInfo(buildinfo.ProductCLI, buildinfo.DistributionGeneric, identity)
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, identity)

	if err := buildinfo.VerifyCompanion(parent, peer); err != nil {
		t.Fatalf("VerifyCompanion(matching pair) = %v, want nil", err)
	}

	tests := []struct {
		name string
		mut  func(*buildinfo.ProtocolInfo)
		want string
	}{
		{
			name: "wrong product",
			mut:  func(p *buildinfo.ProtocolInfo) { p.Product = buildinfo.ProductCLI },
			want: "product",
		},
		{
			name: "protocol revision mismatch",
			mut:  func(p *buildinfo.ProtocolInfo) { p.ProtocolRevision = buildinfo.ProtocolRevision + 1 },
			want: "protocol revision",
		},
		{
			name: "unknown peer distribution",
			mut:  func(p *buildinfo.ProtocolInfo) { p.Distribution = "tap" },
			want: "distribution",
		},
		{
			name: "peer missing revision",
			mut:  func(p *buildinfo.ProtocolInfo) { p.Revision = "dev" },
			want: "revision",
		},
		{
			name: "peer missing commit",
			mut:  func(p *buildinfo.ProtocolInfo) { p.Commit = "unknown" },
			want: "commit",
		},
		{
			name: "peer missing digest",
			mut:  func(p *buildinfo.ProtocolInfo) { p.SourceDigest = "" },
			want: "source digest",
		},
		{
			name: "different commit",
			mut:  func(p *buildinfo.ProtocolInfo) { p.Commit = "ffffffffffffffff" },
			want: "does not match",
		},
		{
			name: "different digest",
			mut:  func(p *buildinfo.ProtocolInfo) { p.SourceDigest = "local-sha256:cafebabe" },
			want: "does not match",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broken := peer
			tt.mut(&broken)
			err := buildinfo.VerifyCompanion(parent, broken)
			if err == nil {
				t.Fatalf("VerifyCompanion(%+v) = nil, want error", broken)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("VerifyCompanion error %q does not contain %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), buildinfo.RebuildPairHint) {
				t.Fatalf("VerifyCompanion error %q does not carry the rebuild-pair hint", err)
			}
		})
	}
}

func TestVerifyCompanionRejectsUninjectedParent(t *testing.T) {
	identity := buildinfo.Identity{Revision: "v0.3.0", Commit: "0123456789abcdef", SourceDigest: "local-sha256:feedface"}
	parent := buildinfo.NewProtocolInfo(buildinfo.ProductCLI, buildinfo.DistributionDev, buildinfo.Identity{Revision: "dev", Commit: "unknown", SourceDigest: "unknown"})
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionDev, identity)

	err := buildinfo.VerifyCompanion(parent, peer)
	if err == nil {
		t.Fatal("VerifyCompanion accepted an uninjected parent identity")
	}
	if !strings.Contains(err.Error(), "this CLI cannot verify the companion") {
		t.Fatalf("VerifyCompanion error %q does not identify the parent side", err)
	}
}
