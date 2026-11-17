package skillsync

import (
	"errors"
	"fmt"
	"strings"
)

// supportedClientIDs is the canonical client order clientTargets, manifests
// and plans all use.
var supportedClientIDs = []string{"agents", "codex", "claude", "agy"}

// SupportedClientIDs returns the canonical, allowlisted client IDs in the
// order the engine plans and records them. The returned slice is a copy;
// callers cannot change the engine's canonical order.
func SupportedClientIDs() []string {
	return append([]string(nil), supportedClientIDs...)
}

// NormalizeClientSelection validates an explicit client selection and returns
// it in canonical client order. A nil selection is returned as nil, which
// means the historical all-clients behavior. A non-nil selection must contain
// at least one supported ID; an explicit empty or unknown selection is an
// error. Duplicate IDs are accepted and collapsed. The returned slice is a
// new copy, never the caller's backing array.
func NormalizeClientSelection(ids []string) ([]string, error) {
	if ids == nil {
		return nil, nil
	}
	if len(ids) == 0 {
		return nil, errors.New("client selection is empty; omit the selection to use every supported client")
	}
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !isSupportedClientID(id) {
			return nil, fmt.Errorf("unknown client %q; supported clients are %s", id, strings.Join(supportedClientIDs, ", "))
		}
		wanted[id] = true
	}
	normalized := make([]string, 0, len(wanted))
	for _, id := range supportedClientIDs {
		if wanted[id] {
			normalized = append(normalized, id)
		}
	}
	return normalized, nil
}

// clientTargetsFor resolves the client skill destinations a plan may touch.
// A nil selection keeps every canonical client; a non-nil selection is
// validated and narrowed in canonical order. It never changes the manifest's
// full canonical home identity.
func clientTargetsFor(home string, ids []string) ([]target, error) {
	all := clientTargets(home)
	if ids == nil {
		return all, nil
	}
	normalized, err := NormalizeClientSelection(ids)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]target, len(all))
	for _, t := range all {
		byName[t.name] = t
	}
	selected := make([]target, 0, len(normalized))
	for _, id := range normalized {
		selected = append(selected, byName[id])
	}
	return selected, nil
}

func isSupportedClientID(id string) bool {
	for _, supported := range supportedClientIDs {
		if id == supported {
			return true
		}
	}
	return false
}
