package tui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"agentic-sdd/internal/skillsync"
)

// RouterRecommendationText is the navigation prompt suggested to newcomers.
// It is navigational guidance only; it never executes any example or agent.
const (
	RouterRecommendationText = "New here? Start with SDD Router"
	RouterRecommendationHint = "press s to inspect details"
	RouterSkillID            = "agentic-sdd-router"
)

// FormatCatalogRowTitle returns the readable display name paired with the exact ID.
// Known curated items show "Name (exact-id)" so users get both human and programmatic identity.
// Unknown or uncurated items fall back neutrally to the exact ID.
func FormatCatalogRowTitle(item CatalogItem) string {
	name := strings.TrimSpace(item.Name)
	id := strings.TrimSpace(item.ID)
	if name != "" && name != id {
		return fmt.Sprintf("%s (%s)", name, id)
	}
	return id
}

// FormatCatalogRowDescription returns the purpose-first summary line for a catalog row.
// The purpose leads the description so newcomers see what the item does first, followed
// by install/status counts.
func FormatCatalogRowDescription(item CatalogItem) string {
	purpose := strings.TrimSpace(item.Purpose)
	summary := item.Summary()
	if purpose == "" || strings.EqualFold(purpose, "not documented") {
		return summary
	}
	return fmt.Sprintf("%s · %s", purpose, summary)
}

// EnrichCatalogItemFilterValue augments the search text with task and trigger phrases
// from the item's When and Example fields, enabling task-oriented search (e.g. "debug",
// "diagnose", "interview", "review").
func EnrichCatalogItemFilterValue(item CatalogItem) string {
	parts := []string{item.ID, item.Name, item.Kind, item.Origin, item.Category, item.Group}
	if !strings.Contains(strings.ToLower(item.Purpose), "not documented") {
		parts = append(parts, item.Purpose)
	}
	if !strings.Contains(strings.ToLower(item.When), "not documented") {
		parts = append(parts, item.When)
	}
	if !strings.Contains(strings.ToLower(item.Example), "not documented") {
		parts = append(parts, item.Example)
	}
	for _, c := range item.Clients {
		parts = append(parts, c.Client)
	}
	if item.Kind == skillsync.InventoryKindProfile && item.documentedRouting() {
		parts = append(parts, item.AgentMode, item.AgentProvider, item.AgentModel, item.RoutingNote)
		parts = append(parts, item.AgentFallbacks...)
	}
	var nonNil []string
	for _, p := range parts {
		if p != "" {
			nonNil = append(nonNil, p)
		}
	}
	return strings.ToLower(strings.Join(nonNil, " "))
}

// HelpSection represents one selectable topic within Activity & Help.
type HelpSection int

const (
	HelpSectionAll HelpSection = iota
	HelpSectionActivity
	HelpSectionQuickStart
	HelpSectionShortcuts
	HelpSectionCommands
	HelpSectionTroubleshooting
	HelpSectionDiagnostics
	HelpSectionAbout
	HelpSectionPreferences
	HelpSectionFleet
	HelpSectionMCP
)

// HelpSectionInfo holds display metadata for one help section tab.
type HelpSectionInfo struct {
	Section HelpSection
	Key     string
	Title   string
}

// AllHelpSections lists every selectable topic in Activity & Help.
func AllHelpSections() []HelpSectionInfo {
	return []HelpSectionInfo{
		{Section: HelpSectionAll, Key: "0", Title: "All"},
		{Section: HelpSectionActivity, Key: "1", Title: "Recent Activity"},
		{Section: HelpSectionQuickStart, Key: "2", Title: "Quick start"},
		{Section: HelpSectionShortcuts, Key: "3", Title: "Shortcuts"},
		{Section: HelpSectionCommands, Key: "4", Title: "Commands"},
		{Section: HelpSectionTroubleshooting, Key: "5", Title: "Troubleshooting"},
		{Section: HelpSectionDiagnostics, Key: "6", Title: "Diagnostics"},
		{Section: HelpSectionAbout, Key: "7", Title: "About"},
		{Section: HelpSectionPreferences, Key: "8", Title: "Preferences"},
		{Section: HelpSectionFleet, Key: "9", Title: "Fleet Manager"},
		{Section: HelpSectionMCP, Key: "10", Title: "MCP Tools"},
	}
}

// MCPHelpLines returns the MCP Tools help/quick-start topic as unwrapped prose lines.
// It states only behavior this build implements: discovering MCP tools (key t in Install
// & Manage), status/doctor/preview, default-No apply and restore, A/B capability gates,
// offline operation with no network, and prerequisites for CodeGraph, Context7, and Exa.
func MCPHelpLines() []string {
	return []string{
		"MCP Tools manages configuration for the three canonical Model Context Protocol servers: CodeGraph, Context7, and Exa.",
		"",
		"Discovery & Navigation:",
		"- In Install & Manage (2), press t or navigate to MCP Tools and press Enter.",
		"- Esc returns to the hub from Overview. Navigation preserves draft state across subviews.",
		"",
		"Everyday flow:",
		"1. Overview shows the status of CodeGraph, Context7, and Exa without running network requests.",
		"2. Select servers (s or Enter on Overview) to pick servers and choose authentication modes.",
		"3. Preview configuration (v) displays exact JSON/JSONC diffs, targets, base digests, and conflicts before any change.",
		"4. Confirm changes with default-No. Applying a configuration requires explicit 'y' confirmation; Enter or Esc cancels.",
		"5. Diagnostics (d) inspects offline prerequisites and OpenCode configuration without active network checks.",
		"6. Recovery (r or e) lists managed MCP backups and previews restore before confirmed application.",
		"",
		"Safety & Boundaries:",
		"- Offline only: no model inference, remote MCP network calls, or OAuth flows run from the TUI.",
		"- Prerequisite checks: CodeGraph requires the local codegraph CLI; Context7 requires an auth mode choice; Exa keyless requires rate-limit acknowledgement.",
		"- Default-No confirmation: apply and restore always default to No to prevent accidental writes.",
		"- Gated capabilities: MCP-A provides read-only status, doctor, and static preview; MCP-B adds structured plan preview, transactional apply, and managed restore.",
		"- Single-slot write host: MCP writes are mutually exclusive with install, restore, and fleet writes.",
	}
}

// FleetHelpLines returns the Fleet Manager help/quick-start topic as unwrapped
// prose lines. It states only behavior this build implements: an explicit local
// source, previewed deployment, default-No confirmation, managed-backup restore
// and preview-only capability. It names no machine-specific path and makes no
// subscription, entitlement, authentication or runtime-verification claim.
// Keep it short: the "All" topic renders every section in one bounded view.
func FleetHelpLines() []string {
	return []string{
		"Fleet Manager keeps agent profiles in one personal fleet.json and renders the managed OpenCode agent files from it.",
		"",
		"First use (pick one; nothing is scanned or inferred):",
		"- Existing file: press s on Overview and type its exact path.",
		"- Just looking: read this topic and docs/fleet-manager-tui.md; both are read-only.",
		"- CLI: run agentic-sdd help fleet, then agentic-sdd fleet status --config PATH.",
		"",
		"Everyday flow:",
		"1. Choose the source. Overview shows the explicit path and read state; empty stays \"(none selected)\".",
		"2. Change a model. In Profiles (p) press m to pick a model for one instance; the edit stays staged in memory.",
		"3. Save the personal source. A separate, explicit write, not available in this TUI build; edit fleet.json yourself, then press r.",
		"4. Preview the deployment. Review (c) shows the exact managed-file plan; previewing writes nothing.",
		"5. Apply with default-No. Deployment needs an explicit y; Enter, n or Esc means no. A live apply backs up first.",
		"6. Restore a managed backup. Recovery (e) lists managed backups; a restore also needs an explicit y.",
		"",
		"Labels that are easy to confuse:",
		"- Role vs instance: a role is the job (for example reviewer); an instance is one entry with its own ID, provider, model and enabled state.",
		"- Personal source vs rendered file vs installed agent: fleet.json is yours, the rendered file is generated from it, and the installed agent is what OpenCode loads; they can differ until you apply and reload.",
		"- installed is not selected primary, not authenticated and not runtime-verified: primary choice, credentials and a model reply are unchecked here.",
		"",
		"Limits in this build:",
		"- Preview only: live deployment and restore are disabled, so apply and restore stay previews.",
		"- No subscription, price, entitlement or quota state is shown or promised.",
		"- No automatic adoption or bulk repair; unmanaged files stay untouched; adoption is an explicit CLI decision.",
		"- Nothing is committed, pushed or sent to a Git host.",
	}
}

// LogOperationFacts holds truthful, uninvented structured facts about an operation log.
// If a field cannot be authoritatively established, it stays "unknown" rather than
// being guessed from unstructured log prose.
type LogOperationFacts struct {
	Name         string
	Path         string
	Operation    string
	Timestamp    string
	RawTimestamp string
	Size         int64
	ModTime      time.Time
	Outcome      string
	ChangedCount string
	SkippedCount string
	FailedCount  string
}

// ParseLogOperationFacts derives structured facts from log metadata and standard filename conventions.
// Unknown facts stay strictly "unknown".
func ParseLogOperationFacts(log LogSummary) LogOperationFacts {
	facts := LogOperationFacts{
		Name:         log.Name,
		Path:         log.Path,
		Size:         log.Size,
		ModTime:      log.ModTime,
		Operation:    "unknown",
		Timestamp:    "unknown",
		RawTimestamp: "unknown",
		Outcome:      "unknown (not recorded in structured log index)",
		ChangedCount: "unknown",
		SkippedCount: "unknown",
		FailedCount:  "unknown",
	}

	base := strings.TrimSuffix(log.Name, ".log")
	parts := strings.Split(base, "-")
	if len(parts) >= 1 && strings.Contains(parts[0], "T") && strings.HasSuffix(parts[0], "Z") {
		facts.RawTimestamp = parts[0]
		if t, err := time.Parse("20060102T150405.000000000Z", parts[0]); err == nil {
			facts.Timestamp = t.UTC().Format("Jan 02, 2006 15:04:05 UTC")
		} else if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
			facts.Timestamp = t.UTC().Format("Jan 02, 2006 15:04:05 UTC")
		} else {
			facts.Timestamp = parts[0]
		}
	}
	if len(parts) >= 2 {
		op := strings.ToLower(parts[1])
		switch op {
		case "apply", "restore", "preview":
			facts.Operation = op
		default:
			facts.Operation = op
		}
	}

	return facts
}

// ExcludedCommands lists experimental operations that are explicitly non-executable
// in this release and displayed for explanatory purpose only.
func ExcludedCommands() []CommandHelp {
	return []CommandHelp{
		{
			Command:     "agentic-sdd opencode ...",
			Description: "OpenCode runtime pack activation (experimental · non-executable under release boundaries)",
		},
		{
			Command:     "agentic-sdd native ...",
			Description: "Experimental native engine execution (non-executable · historical research only)",
		},
	}
}

// TroubleshootingTopic describes one common operational issue and safe recovery advice.
type TroubleshootingTopic struct {
	Title    string
	Advice   string
	Recovery string
}

// TroubleshootingTopics lists canonical recovery tips for common TUI scenarios.
func TroubleshootingTopics() []TroubleshootingTopic {
	return []TroubleshootingTopic{
		{
			Title:    "Update check offline, timed out or rate-limited",
			Advice:   "Automatic update checks run once per session asynchronously on first Home entry. Offline or rate-limited states are non-fatal and never block navigation, installation or restore.",
			Recovery: "Press 'r' in Activity & Help to retry the check, or press 'd' to toggle offline mode (AGENTIC_SDD_NO_UPDATE_CHECK=1).",
		},
		{
			Title:    "Backup unreadable, foreign home or corrupt",
			Advice:   "A backup made for another home directory, or one with invalid JSON manifests, is refused by the restore engine for safety to prevent corrupting local configuration.",
			Recovery: "Inspect existing backups with 'agentic-sdd backups' via explicit CLI. A fresh install or update creates a new valid backup.",
		},
		{
			Title:    "Refused destination (symlink or special file)",
			Advice:   "If a destination directory contains a symlink or unexpected non-regular file, the installer refuses to write to prevent following unverified links.",
			Recovery: "Remove or restore the affected path manually, then run 'agentic-sdd preview' to verify clean destination state.",
		},
		{
			Title:    "No operation logs found",
			Advice:   "Operation logs are created automatically when an install or restore write executes. An empty log list is normal for a fresh installation before any changes are written.",
			Recovery: "Run 'agentic-sdd preview' then confirm an apply to create your first operation log.",
		},
	}
}

// DiagnosticsMetadata aggregates environment, build, runtime, and update state for debugging.
type DiagnosticsMetadata struct {
	Version      string
	Commit       string
	Date         string
	Author       string
	GoVersion    string
	Platform     string
	LogDir       string
	UpdateStatus string
	OfflineMode  bool
}

// CollectDiagnostics extracts current system and session diagnostics.
func CollectDiagnostics(meta AboutMetadata, logDir string, mgr *UpdateCheckManager, theme Theme) DiagnosticsMetadata {
	updateStatus := "Update check disabled or not initialized"
	offline := false
	if mgr != nil {
		updateStatus = mgr.RenderStatus(theme)
		offline = mgr.IsDisabled()
	}
	return DiagnosticsMetadata{
		Version:      meta.Version,
		Commit:       meta.Commit,
		Date:         meta.Date,
		Author:       meta.Author,
		GoVersion:    runtime.Version(),
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		LogDir:       logDir,
		UpdateStatus: updateStatus,
		OfflineMode:  offline,
	}
}

// IsDevBuild reports whether a version identifier signifies a non-release development build.
func IsDevBuild(v string) bool {
	clean := strings.ToLower(strings.TrimSpace(v))
	return clean == "" || clean == "dev" || strings.HasPrefix(clean, "dev-") || strings.Contains(clean, "dev")
}
