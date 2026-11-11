// Package fleet implements the offline, standard-library-only core of the
// Feature 014 agent fleet configuration: strict parsing of a personal
// fleet.json, source-reference resolution against the canonical OpenCode pack,
// deterministic rendering, plan identity, managed apply/restore and the shared
// schema constants.
//
// The package performs no network, model or process operations. Parsing,
// resolution, rendering, planning and status are read-only; apply and restore
// write only when called explicitly and only through the shared skillsync
// backup/stage/commit/rollback engine.
package fleet

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	// SchemaVersion is the only fleet.json schema version accepted in v1.
	SchemaVersion = 1

	// MaxConfigBytes bounds a fleet.json document at 1 MiB.
	MaxConfigBytes = 1 << 20

	// KindAgent and KindPreset name the two allowed SourceRef namespaces.
	KindAgent  = "agent"
	KindPreset = "preset"
)

// ErrInvalidConfig marks every rejected config or resolution input so callers
// can map it to a usage/validation exit status without inspecting messages.
var ErrInvalidConfig = errors.New("invalid fleet config")

var (
	instanceIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	sourceNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	sourceRefRe  = regexp.MustCompile(`^(agent|preset):([a-z0-9][a-z0-9-]*)$`)
	// modelRe follows spec R9: <provider>/<model>[#variant]. The model part
	// may contain slashes for nested provider routes; empty segments are
	// rejected separately by validateModel.
	modelRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$`)
)

// Config is the strict-JSON personal fleet document (schema version 1).
type Config struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Fleet         string              `json:"fleet,omitempty"`
	Preset        string              `json:"preset,omitempty"`
	Instances     map[string]Instance `json:"instances"`
}

// Instance binds one OpenCode-visible instance ID to a canonical source and an
// optional model override.
//
// Model semantics:
// An empty string (""), omitted field, or null value means "leave the canonical
// model untouched", inheriting the source role's canonical model.
//
// Enabled semantics:
// A nil pointer (omitted in JSON or explicit null) defaults to enabled (true).
// An explicit false disables the managed instance (resulting in removal if
// previously installed). An explicit true keeps the instance enabled.
type Instance struct {
	Source  string `json:"source"`
	Model   string `json:"model,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

// IsEnabled reports the effective enablement, defaulting to true when the
// personal config omits the key or sets it to null.
func (i Instance) IsEnabled() bool {
	if i.Enabled == nil {
		return true
	}
	return *i.Enabled
}

// ReservedNames lists reserved Agentic SDD instance names (spec, plan, vision).
var ReservedNames = map[string]bool{
	"spec":   true,
	"plan":   true,
	"vision": true,
}

// IsReservedName reports whether an instance ID is a reserved SDD name.
func IsReservedName(name string) bool {
	return ReservedNames[name]
}

// SourceRef is a one-hop reference to a canonical source file.
type SourceRef struct {
	Kind string
	Name string
}

// String renders the canonical agent:<name> or preset:<name> form.
func (r SourceRef) String() string {
	return r.Kind + ":" + r.Name
}

// ParseSourceRef validates the one-hop source grammar and returns its parts.
func ParseSourceRef(raw string) (SourceRef, error) {
	ref, err := parseSourceRef(raw)
	if err != nil {
		return SourceRef{}, configError("%s", err)
	}
	return ref, nil
}

func parseSourceRef(raw string) (SourceRef, error) {
	if raw == "" {
		return SourceRef{}, errors.New("source is required")
	}
	if strings.ContainsAny(raw, `/\`) || strings.Contains(raw, "..") {
		return SourceRef{}, fmt.Errorf("invalid source %q: path separators and traversal are not allowed", raw)
	}
	m := sourceRefRe.FindStringSubmatch(raw)
	if m == nil {
		return SourceRef{}, fmt.Errorf("invalid source %q: want agent:<name> or preset:<name>", raw)
	}
	return SourceRef{Kind: m[1], Name: m[2]}, nil
}

// Validate applies the v1 schema rules: exact schema version, safe labels,
// instance IDs, one-hop source references and the R9 model grammar.
func (c *Config) Validate() error {
	if c == nil {
		return configError("config is nil")
	}
	if c.SchemaVersion != SchemaVersion {
		return configError("unsupported schemaVersion %d (supported: %d)", c.SchemaVersion, SchemaVersion)
	}
	if err := validateLabel("fleet", c.Fleet); err != nil {
		return configError("%s", err)
	}
	if err := validateLabel("preset", c.Preset); err != nil {
		return configError("%s", err)
	}
	for _, id := range sortedInstanceIDs(c.Instances) {
		inst := c.Instances[id]
		if err := validateInstanceID(id); err != nil {
			return configError("%s", err)
		}
		if _, err := parseSourceRef(inst.Source); err != nil {
			return configError("instance %q: %s", id, err)
		}
		if err := validateModel(inst.Model); err != nil {
			return configError("instance %q: %s", id, err)
		}
	}
	return nil
}

func validateInstanceID(id string) error {
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return fmt.Errorf("invalid instance ID %q: path separators and traversal are not allowed", id)
	}
	if !instanceIDRe.MatchString(id) {
		return fmt.Errorf("invalid instance ID %q: want lowercase letters, digits and hyphens (max 64)", id)
	}
	return nil
}

func validateLabel(field, value string) error {
	if value == "" {
		return nil
	}
	if strings.ContainsAny(value, `/\`) || strings.Contains(value, "..") {
		return fmt.Errorf("%s %q must not contain path separators or traversal", field, value)
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%s contains a control character", field)
		}
	}
	return nil
}

func validateModel(model string) error {
	if model == "" {
		return nil
	}
	if !modelRe.MatchString(model) {
		return fmt.Errorf("invalid model %q: want <provider>/<model>[#variant]", model)
	}
	base := model
	if i := strings.IndexByte(base, '#'); i >= 0 {
		base = base[:i]
	}
	for _, part := range strings.Split(base, "/") {
		if part == "" {
			return fmt.Errorf("invalid model %q: empty path segment", model)
		}
	}
	return nil
}

func sortedInstanceIDs(instances map[string]Instance) []string {
	ids := make([]string, 0, len(instances))
	for id := range instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func configError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, fmt.Sprintf(format, args...))
}
