package tui

// impact.go derives an immutable presentation projection from the existing
// typed preview plan. It never changes the plan, the preview token or the
// engine's identity: grouping, totals and expansion are presentation only.
//
// Every planned change is classified exactly once. An action kind this screen
// does not know is kept in its own explicit unknown bucket; it is never
// reported as an addition and it disables confirmation until authoritative
// planning resolves it.

import (
	"fmt"
	"path/filepath"
	"strings"

	"agentic-sdd/internal/skillsync"
)

// impactClass is the presentation classification of one planned change.
type impactClass int

const (
	impactNew impactClass = iota
	impactReplace
	impactRestore
	impactRemove
	impactUnchanged
	impactUnknown
	impactClassCount
)

// impactItem references one planned change by its original index. The change
// itself is copied by value only to keep the projection immutable.
type impactItem struct {
	index  int
	change skillsync.ResultChange
}

// impactGroup is one destination (the typed client identity) with its items in
// plan order and their per-action counts. root is the exact common parent of
// the group's paths, or empty when the paths do not share one.
type impactGroup struct {
	client string
	root   string
	items  []impactItem
	counts [impactClassCount]int
}

// count returns the number of items in the group for one action class.
func (g impactGroup) count(class impactClass) int { return g.counts[class] }

// impactProjection is the derived, read-only view of one plan.
type impactProjection struct {
	total  int
	counts [impactClassCount]int
	groups []impactGroup
	// items keeps the original plan order for action-ordered renderers. It
	// references the same items as groups; totals are never derived from it.
	items []impactItem
}

// projectImpact classifies a copy of the plan in one pass. The input slice is
// never modified.
func projectImpact(planned []skillsync.ResultChange) impactProjection {
	projection := impactProjection{total: len(planned)}
	groupIndex := make(map[string]int)
	for i, change := range planned {
		item := impactItem{index: i, change: change}
		class := classifyImpact(change.Action)
		projection.counts[class]++
		projection.items = append(projection.items, item)
		gi, ok := groupIndex[change.Client]
		if !ok {
			gi = len(projection.groups)
			groupIndex[change.Client] = gi
			projection.groups = append(projection.groups, impactGroup{client: change.Client})
		}
		projection.groups[gi].items = append(projection.groups[gi].items, item)
		projection.groups[gi].counts[class]++
	}
	for i := range projection.groups {
		projection.groups[i].root = commonImpactRoot(projection.groups[i].items)
	}
	return projection
}

// classifyImpact maps only the known engine actions to a class. Any other
// action, including an empty one, stays explicitly unknown.
func classifyImpact(action skillsync.ChangeAction) impactClass {
	switch action {
	case skillsync.ChangeInstall:
		return impactNew
	case skillsync.ChangeReplace:
		return impactReplace
	case skillsync.ChangeRestore:
		return impactRestore
	case skillsync.ChangeRemove:
		return impactRemove
	case skillsync.ChangeUpToDate:
		return impactUnchanged
	default:
		return impactUnknown
	}
}

// commonImpactRoot returns the single parent directory shared by every item
// path, or empty when a path is missing or the parents differ.
func commonImpactRoot(items []impactItem) string {
	root := ""
	for i, item := range items {
		if item.change.Path == "" {
			return ""
		}
		dir := filepath.Dir(item.change.Path)
		if i == 0 {
			root = dir
			continue
		}
		if dir != root {
			return ""
		}
	}
	return root
}

func (p impactProjection) count(class impactClass) int { return p.counts[class] }

// itemsOf returns the plan-ordered items for one class.
func (p impactProjection) itemsOf(class impactClass) []impactItem {
	var out []impactItem
	for _, item := range p.items {
		if classifyImpact(item.change.Action) == class {
			out = append(out, item)
		}
	}
	return out
}

func (p impactProjection) hasUnknown() bool { return p.counts[impactUnknown] > 0 }

// mutable reports how many planned items would change a destination. Unknown
// actions are excluded because this screen cannot claim what they do.
func (p impactProjection) mutable() int {
	return p.counts[impactNew] + p.counts[impactReplace] + p.counts[impactRestore] + p.counts[impactRemove]
}

// noOp reports whether every item is an explicit up-to-date match with at
// least one item planned. An empty or unknown plan is never a successful
// no-op.
func (p impactProjection) noOp() bool {
	return p.total > 0 && p.mutable() == 0 && !p.hasUnknown()
}

// impactClassLabel is the short, honest label used in totals and group rows.
func impactClassLabel(class impactClass) string {
	switch class {
	case impactNew:
		return "new"
	case impactReplace:
		return "replace (backup first)"
	case impactRestore:
		return "restore"
	case impactRemove:
		return "remove"
	case impactUnchanged:
		return "unchanged"
	default:
		return "unknown"
	}
}

// impactSummary is the concise totals line. Every class is stated so a zero
// count is visible instead of hidden.
func impactSummary(p impactProjection) string {
	parts := []string{
		fmt.Sprintf("%d planned", p.total),
		fmt.Sprintf("%d new", p.count(impactNew)),
		fmt.Sprintf("%d %s", p.count(impactReplace), impactClassLabel(impactReplace)),
		fmt.Sprintf("%d %s", p.count(impactRestore), impactClassLabel(impactRestore)),
		fmt.Sprintf("%d %s", p.count(impactRemove), impactClassLabel(impactRemove)),
		fmt.Sprintf("%d %s", p.count(impactUnchanged), impactClassLabel(impactUnchanged)),
		fmt.Sprintf("%d %s", p.count(impactUnknown), impactClassLabel(impactUnknown)),
	}
	return "Impact totals: " + strings.Join(parts, " · ")
}

// impactGroupSummary is the per-destination line: the typed destination
// identity, its exact root when one is known, the planned count and the
// non-zero action counts. data escapes dynamic values at the presentation
// boundary without changing the projection.
func impactGroupSummary(g impactGroup, data func(string) string) string {
	if data == nil {
		data = func(s string) string { return s }
	}
	var b strings.Builder
	client := g.client
	if strings.TrimSpace(client) == "" {
		client = "unreported destination"
	} else {
		client = ClientSelectionLabel(client)
	}
	b.WriteString(data(client))
	if g.root != "" {
		b.WriteString(" — " + data(g.root))
	}
	b.WriteString(fmt.Sprintf(" · %d planned", len(g.items)))
	for _, class := range []impactClass{impactNew, impactReplace, impactRestore, impactRemove, impactUnchanged, impactUnknown} {
		if n := g.count(class); n > 0 {
			b.WriteString(fmt.Sprintf(" · %d %s", n, impactClassLabel(class)))
		}
	}
	return b.String()
}

// impactItemLabel is the one-item action label. Unknown actions carry their
// raw token so nothing is silently reclassified.
func impactItemLabel(item impactItem) string {
	class := classifyImpact(item.change.Action)
	if class != impactUnknown {
		return impactClassLabel(class)
	}
	action := string(item.change.Action)
	if strings.TrimSpace(action) == "" {
		action = "unreported"
	}
	return fmt.Sprintf("unknown action %q (not treated as new)", action)
}

// impactItemIdentity is the stable ID + client reference for one item. data
// escapes those dynamic values for the active presentation profile.
func impactItemIdentity(item impactItem, data func(string) string) string {
	if data == nil {
		data = func(s string) string { return s }
	}
	id := item.change.ID
	if strings.TrimSpace(id) == "" {
		id = "(no ID reported)"
	}
	client := item.change.Client
	if strings.TrimSpace(client) == "" {
		client = "unreported destination"
	} else {
		client = ClientSelectionLabel(client)
	}
	return fmt.Sprintf("%s (%s)", data(id), data(client))
}
