package tui

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// FleetModelEdit represents one in-memory staged modification to an agent instance.
// It carries the instance ID, source provenance, staged model override, and enablement state.
type FleetModelEdit struct {
	InstanceID string
	Source     string
	Model      string
	Enabled    bool
}

// FleetEditor is the TUI-local interface for previewing personal source modifications.
// Implementations perform read-only diff projections; no filesystem mutation, CLI subprocess,
// or model execution is permitted.
type FleetEditor interface {
	PreviewSource(ctx context.Context, source string, edits []FleetModelEdit) (FleetSourcePreview, error)
}

// DefaultFleetEditor is the safe default editor when no production appops adapter is connected.
// It returns a truthful not-available result without filesystem access or external imports.
type DefaultFleetEditor struct{}

// PreviewSource returns a truthful not-available source preview without touching disk.
func (d DefaultFleetEditor) PreviewSource(ctx context.Context, source string, edits []FleetModelEdit) (FleetSourcePreview, error) {
	return FleetSourcePreview{
		Available:  false,
		SourcePath: source,
		Op:         "unavailable",
		Message:    "Source preview editor not available in this build (adapter pending).",
		DiffLines:  []string{"(source diff unavailable in this build)"},
	}, nil
}

// modelRe matches the Feature 014 model syntax: <provider>/<model>[#variant].
var modelRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$`)

// ValidateModelSyntax validates an exact provider/model[#variant] string per Feature 014 spec.
// Whitespace, empty values, control characters, or missing provider prefixes return descriptive errors.
func ValidateModelSyntax(model string) error {
	m := strings.TrimSpace(model)
	if m == "" {
		return errors.New("model cannot be empty")
	}
	if strings.ContainsAny(m, " \t\r\n") {
		return fmt.Errorf("invalid model %q: whitespace is not allowed", model)
	}
	if !modelRe.MatchString(m) {
		return fmt.Errorf("invalid model %q: want <provider>/<model>[#variant]", model)
	}
	base := m
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

// DefaultKnownModels returns standard locally cataloged route suggestions.
func DefaultKnownModels() []string {
	return []string{
		"9router/ag/gemini-3.8-flash-high",
		"9router/ag/gemini-3.8-flash-medium",
		"9router/ag/gemini-4-argon-medium",
		"opencode-go/deepseek-v4.1-flash",
		"opencode-go/qwen3.8-flash",
		"opencode-go/mimo-v2.6-flash",
		"opencode/claude-sonnet-5.5",
		"opencode/claude-opus-5.5",
	}
}

// KnownModelSuggestions builds a deduplicated list of model suggestions from current instances
// and standard routes, optionally filtered by a search query.
func KnownModelSuggestions(instances []FleetInstanceView, currentModel, query string) []string {
	seen := make(map[string]bool)
	var list []string

	add := func(m string) {
		m = strings.TrimSpace(m)
		if m != "" && !seen[m] && ValidateModelSyntax(m) == nil {
			seen[m] = true
			list = append(list, m)
		}
	}

	resolveInstModel := func(inst FleetInstanceView) string {
		m := strings.TrimSpace(inst.Model)
		if m == "" {
			return ""
		}
		if !strings.Contains(m, "/") && inst.ProviderRoute != "" && inst.ProviderRoute != "unspecified" {
			return strings.TrimSpace(inst.ProviderRoute) + "/" + m
		}
		return m
	}

	if currentModel != "" {
		add(currentModel)
	}
	for _, inst := range instances {
		add(resolveInstModel(inst))
	}
	for _, k := range DefaultKnownModels() {
		add(k)
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return list
	}

	var filtered []string
	for _, m := range list {
		if strings.Contains(strings.ToLower(m), q) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// CompactModelName returns a short display representation of a full model route.
func CompactModelName(full string) string {
	if full == "" {
		return "(default)"
	}
	parts := strings.Split(full, "/")
	last := parts[len(parts)-1]
	return last
}
