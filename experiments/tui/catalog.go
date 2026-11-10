package tui

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// Presentation groups. They mirror the purpose groups the CLI shows for the
// same inventory, but they are derived here from kind and origin so the UI
// never has to invent domain meaning.
const (
	GroupCoreSkills       = "Core skills"
	GroupOpenCodePack     = "OpenCode pack"
	GroupOpenCodeProfiles = "OpenCode profiles"
	GroupFreeFirstPresets = "Free-first presets"
)

// Category IDs for filtered browsing.
const (
	CategoryAll      = "all"
	CategoryCore     = "core"
	CategoryPack     = "pack"
	CategoryProfiles = "profiles"
	CategoryPresets  = "presets"
)

// CatalogCategory describes one category filter option with its visible item count.
type CatalogCategory struct {
	ID    string
	Label string
	Count int
}

// catalogKeySeparator joins kind and ID into one map key. NUL cannot appear in
// a managed file name, so distinct pairs cannot collide.
const catalogKeySeparator = "\x00"

// catalogGroupOrder fixes group ordering so the list is stable across runs.
var catalogGroupOrder = []string{
	GroupCoreSkills,
	GroupOpenCodePack,
	GroupOpenCodeProfiles,
	GroupFreeFirstPresets,
}

// ClientStatus is one destination's read-only state for a catalog item.
type ClientStatus struct {
	Client string
	Status skillsync.InventoryStatus
	Path   string
}

// maxSummaryBadgeWidth bounds the provider badge appended to a profile list
// row so the badge still fits the narrowest 32-cell list pane. Longer
// documented names are truncated with a cell-aware ellipsis, never rewritten.
const maxSummaryBadgeWidth = 24

// routingUndocumented is the honest provider label for a profile that Agentic
// SDD does not document. Nothing is inferred from its ID or model text.
const routingUndocumented = "not documented by Agentic SDD"

// routingUndocumentedNote keeps an unlisted profile truthful instead of
// filling the gap with a guessed lane.
const routingUndocumentedNote = "No curated routing metadata is documented for this profile in Agentic SDD. Its provider, model and fallbacks are not inferred from the profile ID, and no credential, quota, availability or runtime state is claimed."

// CatalogItem is one read-only catalog row: a managed skill directory or a
// managed profile file, with the per-client statuses InspectSelection reported.
//
// The Agent* and RoutingNote fields are additive, display-only configuration
// facts about a profile template: the mode, provider lane, pinned model and
// documented manual alternatives it declares. They never mean a credential
// exists, a model is reachable, or an agent is running.
type CatalogItem struct {
	ID       string
	Kind     string
	Origin   string
	Group    string
	Clients  []ClientStatus
	Name     string
	Purpose  string
	When     string
	Example  string
	Caveat   string
	Category string

	AgentMode      string
	AgentProvider  string
	AgentModel     string
	AgentFallbacks []string
	RoutingNote    string
}

// enrichItem fills additive metadata fields from appops.TeachingFor and
// appops.AgentRoutingFor if not already set. Non-empty fields are preserved so
// existing or caller-supplied values are never overwritten.
func enrichItem(item *CatalogItem) {
	if item.Name == "" || item.Purpose == "" || item.When == "" || item.Example == "" || item.Caveat == "" || item.Category == "" {
		card := appops.TeachingFor(item.ID, item.Kind)
		if item.Name == "" {
			if card.DisplayName != "" {
				item.Name = card.DisplayName
			} else {
				item.Name = item.ID
			}
		}
		if item.Purpose == "" {
			item.Purpose = card.Purpose
		}
		if item.When == "" {
			item.When = card.When
		}
		if item.Example == "" {
			item.Example = card.Example
		}
		if item.Caveat == "" {
			item.Caveat = card.Caveat
		}
		if item.Category == "" {
			item.Category = card.Group
		}
	}
	enrichRouting(item)
	item.Group = catalogGroup(item.Kind, item.Origin)
}

// enrichRouting fills the documented model policy of an agent profile. Skills
// and concepts have no routing policy, so they stay empty. A profile Agentic
// SDD does not document keeps whatever display values it already carries and
// gains only the honest absence wording; no provider or model is guessed from
// an arbitrary ID or string.
func enrichRouting(item *CatalogItem) {
	if item.Kind != skillsync.InventoryKindProfile {
		return
	}
	if routing, ok := appops.AgentRoutingFor(item.ID); ok {
		if item.AgentMode == "" {
			item.AgentMode = routing.Mode
		}
		if item.AgentProvider == "" {
			item.AgentProvider = routing.Provider
		}
		if item.AgentModel == "" {
			item.AgentModel = routing.Model
		}
		if len(item.AgentFallbacks) == 0 {
			item.AgentFallbacks = append([]string(nil), routing.Fallbacks...)
		}
		if item.RoutingNote == "" {
			item.RoutingNote = routing.Note
		}
		return
	}
	if item.AgentMode == "" {
		item.AgentMode = routingUndocumented
	}
	if item.AgentProvider == "" {
		item.AgentProvider = routingUndocumented
	}
	if item.AgentModel == "" {
		item.AgentModel = routingUndocumented
	}
	if item.RoutingNote == "" {
		item.RoutingNote = routingUndocumentedNote
	}
}

// FilterValue is the text the Bubbles list searches. It includes the ID, name,
// purpose (for curated items), kind, origin, category, group and client names so `/` can match any of them.
// A profile additionally contributes its documented mode, provider, primary
// model, fallbacks and routing note so a search can reach a lane or model name.
func (i CatalogItem) FilterValue() string {
	parts := []string{i.ID, i.Name, i.Kind, i.Origin, i.Category, i.Group}
	if !strings.Contains(strings.ToLower(i.Purpose), "not documented") {
		parts = append(parts, i.Purpose)
	}
	for _, c := range i.Clients {
		parts = append(parts, c.Client)
	}
	if i.Kind == skillsync.InventoryKindProfile && i.documentedRouting() {
		parts = append(parts, i.AgentMode, i.AgentProvider, i.AgentModel, i.RoutingNote)
		parts = append(parts, i.AgentFallbacks...)
	}
	var nonNil []string
	for _, p := range parts {
		if p != "" {
			nonNil = append(nonNil, p)
		}
	}
	return strings.ToLower(strings.Join(nonNil, " "))
}

// documentedRouting reports whether the item carries curated routing metadata.
// The honest absence placeholder is uniform boilerplate, so it is left out of
// search text, exactly as an undocumented purpose already is: adding it would
// only let unrelated fuzzy queries match every unlisted profile.
func (i CatalogItem) documentedRouting() bool {
	return i.AgentProvider != "" && i.AgentProvider != routingUndocumented
}

// Summary is the one-line list description: kind, origin and install coverage.
// An agent profile uses a compact coverage count plus its documented provider
// lane as the final badge, so the row stays readable in a narrow list pane
// without claiming the model ran. Skills and concepts keep the original string.
func (i CatalogItem) Summary() string {
	if i.Kind == skillsync.InventoryKindProfile {
		return fmt.Sprintf("%s · %d/%d", i.routingBadge(), i.InstalledClients(), len(i.Clients))
	}
	return fmt.Sprintf("%s · %s · %d/%d clients installed", i.Kind, i.Origin, i.InstalledClients(), len(i.Clients))
}

// routingBadge is the compact provider lane of a profile row. Empty routing is
// labelled honestly rather than left blank, and an over-long name is truncated
// cell-aware to maxSummaryBadgeWidth so the badge cannot widen a list row.
func (i CatalogItem) routingBadge() string {
	badge := strings.TrimSpace(i.AgentProvider)
	if badge == "" {
		badge = routingUndocumented
	}
	return ansi.Truncate(badge, maxSummaryBadgeWidth, "…")
}

// InstalledClients counts destinations that hold content (matching or
// modified). It never counts a missing or refused destination.
func (i CatalogItem) InstalledClients() int {
	n := 0
	for _, c := range i.Clients {
		if c.Status == skillsync.InventoryStatusMatchesSource || c.Status == skillsync.InventoryStatusDiffersFromSource {
			n++
		}
	}
	return n
}

// InstallHint is the copyable next action for this item. Preview is always
// suggested first: the TUI never performs a change.
func (i CatalogItem) InstallHint() string {
	switch {
	case i.Origin == skillsync.InventoryOriginPreset:
		return "agentic-sdd preview --with-opencode --opencode-free-first"
	case i.Origin == skillsync.InventoryOriginPack:
		return "agentic-sdd preview --with-opencode"
	default:
		return "agentic-sdd preview"
	}
}

// ClientStatusFor returns one destination's status by client name.
func (i CatalogItem) ClientStatusFor(client string) (ClientStatus, bool) {
	for _, c := range i.Clients {
		if c.Client == client {
			return c, true
		}
	}
	return ClientStatus{}, false
}

// Catalog is the read-only snapshot the browser renders.
type Catalog struct {
	Home       string
	Repo       string
	BackupRoot string
	Items      []CatalogItem
	// Notices are non-fatal conditions worth showing, for example an absent
	// optional OpenCode pack.
	Notices []string
}

// SkillCount and ProfileCount are the header counts.
func (c Catalog) SkillCount() int { return c.countKind(skillsync.InventoryKindSkill) }

// ProfileCount returns how many profile rows the snapshot holds.
func (c Catalog) ProfileCount() int { return c.countKind(skillsync.InventoryKindProfile) }

func (c Catalog) countKind(kind string) int {
	n := 0
	for _, item := range c.Items {
		if item.Kind == kind {
			n++
		}
	}
	return n
}

// LoadCatalog inspects one home through skillsync and maps the result into
// renderable rows. It performs no writes: InspectSelection only reads sources
// and lstat-compares destinations.
//
// An absent optional OpenCode pack is not an error: the core rows are kept and
// a notice is recorded, matching InspectSelection's documented behavior.
func LoadCatalog(cfg TUIConfig) (Catalog, error) {
	cfg = cfg.Normalize()
	if strings.TrimSpace(cfg.Home) == "" {
		return Catalog{}, errors.New("tui: TUIConfig.Home is required")
	}
	if cfg.OpenCodeFreeFirst && !cfg.WithOpenCode {
		return Catalog{}, errors.New("tui: OpenCodeFreeFirst requires WithOpenCode")
	}

	inv, err := skillsync.InspectSelection(cfg.Repo, cfg.Home, skillsync.Options{
		WithOpenCode:      cfg.WithOpenCode,
		OpenCodeFreeFirst: cfg.OpenCodeFreeFirst,
	})
	cat := Catalog{Home: inv.Home, Repo: inv.Repo, BackupRoot: inv.BackupRoot}
	switch {
	case err == nil:
	case errors.Is(err, skillsync.ErrPackMissing):
		cat.Notices = append(cat.Notices, "OpenCode pack source not found; showing core skills only")
	default:
		return Catalog{}, fmt.Errorf("tui: inspect selection: %w", err)
	}

	// One row per managed ID, not per (ID, client) pair, so the browser shows a
	// skill once with every destination's state.
	index := make(map[string]*CatalogItem, len(inv.Entries))
	for _, entry := range inv.Entries {
		key := entry.Kind + catalogKeySeparator + entry.ID
		item, seen := index[key]
		if !seen {
			item = &CatalogItem{
				ID:     entry.ID,
				Kind:   entry.Kind,
				Origin: entry.Origin,
				Group:  catalogGroup(entry.Kind, entry.Origin),
			}
			enrichItem(item)
			index[key] = item
		}
		item.Clients = append(item.Clients, ClientStatus{
			Client: entry.Client,
			Status: entry.Status,
			Path:   entry.Path,
		})
	}
	cat.Items = make([]CatalogItem, 0, len(index))
	for _, item := range index {
		sort.Slice(item.Clients, func(a, b int) bool {
			return item.Clients[a].Client < item.Clients[b].Client
		})
		cat.Items = append(cat.Items, *item)
	}
	sort.Slice(cat.Items, func(a, b int) bool {
		ga, gb := catalogGroupRank(cat.Items[a].Group), catalogGroupRank(cat.Items[b].Group)
		if ga != gb {
			return ga < gb
		}
		if cat.Items[a].Kind != cat.Items[b].Kind {
			return cat.Items[a].Kind < cat.Items[b].Kind
		}
		return cat.Items[a].ID < cat.Items[b].ID
	})
	return cat, nil
}

// SummarizePreview classifies a preview or apply transcript. It is the only
// other domain entry point the TUI is allowed to use, and it is pure: the
// transcript is produced by the CLI layer, never by this package.
func SummarizePreview(transcript string) skillsync.PlanSummary {
	return skillsync.SummarizePlan(transcript)
}

// StatusBadge is the shared destination vocabulary the CLI and the TUI both
// show. Keeping one mapping avoids a second, drifting status language.
func StatusBadge(status skillsync.InventoryStatus) string {
	switch status {
	case skillsync.InventoryStatusMatchesSource:
		return "Installed (matches source)"
	case skillsync.InventoryStatusDiffersFromSource:
		return "Installed (modified)"
	case skillsync.InventoryStatusMissing:
		return "Template available (not installed)"
	case skillsync.InventoryStatusUnavailable:
		return "Unavailable / refused"
	default:
		return string(status)
	}
}

func catalogGroup(kind, origin string) string {
	switch {
	case kind == skillsync.InventoryKindProfile && origin == skillsync.InventoryOriginPreset:
		return GroupFreeFirstPresets
	case kind == skillsync.InventoryKindProfile:
		return GroupOpenCodeProfiles
	case origin == skillsync.InventoryOriginPack:
		return GroupOpenCodePack
	default:
		return GroupCoreSkills
	}
}

func catalogGroupRank(group string) int {
	for i, g := range catalogGroupOrder {
		if g == group {
			return i
		}
	}
	return len(catalogGroupOrder)
}

// classifyItem returns the base category ID for an item derived authoritatively from its Kind and Origin.
func classifyItem(item CatalogItem) string {
	switch {
	case item.Kind == skillsync.InventoryKindProfile && item.Origin == skillsync.InventoryOriginPreset:
		return CategoryPresets
	case item.Kind == skillsync.InventoryKindProfile:
		return CategoryProfiles
	case item.Origin == skillsync.InventoryOriginPack:
		return CategoryPack
	default:
		return CategoryCore
	}
}

// itemMatchesCategory reports whether an item belongs to the requested category ID.
func itemMatchesCategory(item CatalogItem, categoryID string) bool {
	catLower := strings.ToLower(categoryID)
	switch catLower {
	case CategoryAll, "":
		return true
	case CategoryCore, CategoryPack, CategoryProfiles, CategoryPresets:
		return classifyItem(item) == catLower
	default:
		if strings.HasPrefix(catLower, "purpose:") {
			group := categoryID[len("purpose:"):]
			return strings.EqualFold(item.Category, group)
		}
		return false
	}
}
