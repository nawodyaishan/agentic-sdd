package tui

import (
	"errors"
	"fmt"
	"sort"
	"strings"

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

// CatalogItem is one read-only catalog row: a managed skill directory or a
// managed profile file, with the per-client statuses InspectSelection reported.
type CatalogItem struct {
	ID      string
	Kind    string
	Origin  string
	Group   string
	Clients []ClientStatus
}

// FilterValue is the text the Bubbles list searches. It includes the ID, kind,
// origin, group and client names so `/` can match any of them.
func (i CatalogItem) FilterValue() string {
	parts := []string{i.ID, i.Kind, i.Origin, i.Group}
	for _, c := range i.Clients {
		parts = append(parts, c.Client)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// Summary is the one-line list description: kind, origin and install coverage.
func (i CatalogItem) Summary() string {
	return fmt.Sprintf("%s · %s · %d/%d clients installed", i.Kind, i.Origin, i.InstalledClients(), len(i.Clients))
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
