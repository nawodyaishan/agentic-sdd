package tui

import (
	"context"
	"fmt"
	"strings"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// MCPReader is the read-only projection seam for MCP status and diagnostics.
// Implementations must perform only read operations; no file mutation, CLI
// subprocess, or network calls are permitted.
type MCPReader interface {
	Status(ctx context.Context, home, configPath, projectPath string) (MCPStatusView, error)
	Doctor(ctx context.Context, home, configPath, projectPath, openCodeBin, codeGraphBin string) (MCPDoctorView, error)
}

// MCPPlanner is the read-only configuration planning and recovery seam. Every
// method performs a read-only projection: PreviewPlan, ListBackups, and
// PreviewRestore never write, lock, or modify files.
type MCPPlanner interface {
	PreviewPlan(ctx context.Context, home, configPath string, selected []string, auth map[string]string) (MCPPlanPreviewView, error)
	ListBackups(ctx context.Context, home string) ([]MCPBackupView, error)
	PreviewRestore(ctx context.Context, home, backupID string) (MCPRestorePreviewView, error)
}

// MCPServerView represents one of the three default MCP servers in the TUI.
type MCPServerView struct {
	Name              string // "codegraph", "context7", "exa"
	DisplayName       string // "CodeGraph", "Context7", "Exa"
	Kind              string // "local" or "remote"
	Purpose           string // repository code intelligence, hosted documentation, web research
	ConfigState       string // "configured", "not configured", "disabled", "conflict", "schema mismatch", "unsafe target", "missing prerequisite", "unknown"
	Prerequisite      string // e.g. "codegraph CLI", "none"
	PrerequisiteMet   bool
	AuthMode          string // "oauth", "api-key", "keyless", "none"
	VerificationState string // "connection not tested", "not checked"
	NextAction        string
	Managed           bool
	Disabled          bool
	Conflict          string
	IndexState        string
}

// MCPStatusView is the immutable typed view-model for MCP Tools status.
type MCPStatusView struct {
	Available        bool
	Home             string
	ConfigPath       string
	ConfigFormat     string // "json", "jsonc"
	Schema           string
	ConfigDigest     string
	Ambiguity        string
	LegacyReason     string
	Servers          []MCPServerView
	CodeGraphBinary  string
	CodeGraphVersion string
	OpenCodeVersion  string
	NotChecked       []string
	RawStatus        *skillsync.MCPStatus
}

// MCPEntryDiffView represents one planned server entry with its diff projection.
type MCPEntryDiffView struct {
	Name      string
	Op        string // "add", "change", "leave", "disabled", "conflict"
	Reason    string
	OldText   string
	NewText   string
	DiffLines []string
}

// MCPPlanPreviewView is the read-only configuration preview projection.
type MCPPlanPreviewView struct {
	Available       bool
	TargetPath      string
	Format          string // "json", "jsonc"
	Schema          string
	BaseDigest      string
	PlanDigest      string
	Selected        []string
	Entries         []MCPEntryDiffView
	Conflicts       []string
	AuthChoices     map[string]string
	RuntimeVersion  string
	RuntimeEvidence string
	NoOp            bool
	NotChecked      []string
	Token           string
	RawPlan         *skillsync.MCPPlan
	DiffLines       []string
	Message         string
}

// MCPBackupView represents one recorded MCP backup listed for recovery.
type MCPBackupView struct {
	ID        string
	Path      string
	CreatedAt string
	Servers   []string
}

// MCPRestorePreviewView is the read-only current-versus-backup restore projection.
type MCPRestorePreviewView struct {
	Available  bool
	BackupID   string
	TargetPath string
	BaseDigest string
	Token      string
	Entries    []MCPEntryDiffView
	Conflicts  []string
	RawPreview *skillsync.MCPRestorePreview
	DiffLines  []string
}

// MCPDoctorCheckView is one diagnostic check item.
type MCPDoctorCheckView struct {
	Name   string
	State  string // "pass", "warn", "blocked", "not-checked"
	Detail string
}

// MCPDoctorView is the structured diagnostic report view.
type MCPDoctorView struct {
	Available  bool
	Checks     []MCPDoctorCheckView
	NotChecked []string
}

// DefaultMCPReader is the production reader seam using internal/appops.
type DefaultMCPReader struct{}

// Status builds the read-only MCP status projection.
func (DefaultMCPReader) Status(ctx context.Context, home, configPath, projectPath string) (MCPStatusView, error) {
	if home == "" {
		return MCPStatusView{
			Available: false,
			Servers:   defaultServerViews(),
		}, nil
	}
	req, err := appops.NewMCPStatusRequest(home, configPath, projectPath)
	if err != nil {
		return MCPStatusView{
			Available: false,
			Servers:   defaultServerViews(),
		}, err
	}
	res, err := appops.MCPStatus(req)
	if err != nil {
		return MCPStatusView{
			Available: false,
			Home:      home,
			Servers:   defaultServerViews(),
		}, err
	}
	return mapStatusView(res.Status), nil
}

// Doctor builds the read-only offline doctor projection.
func (DefaultMCPReader) Doctor(ctx context.Context, home, configPath, projectPath, openCodeBin, codeGraphBin string) (MCPDoctorView, error) {
	if home == "" {
		return MCPDoctorView{Available: false}, nil
	}
	req, err := appops.NewMCPDoctorRequest(home, configPath, projectPath, openCodeBin, codeGraphBin)
	if err != nil {
		return MCPDoctorView{Available: false}, err
	}
	res, err := appops.MCPDoctor(req)
	if err != nil {
		return MCPDoctorView{Available: false}, err
	}
	checks := make([]MCPDoctorCheckView, 0, len(res.Checks))
	for _, c := range res.Checks {
		checks = append(checks, MCPDoctorCheckView{
			Name:   c.Name,
			State:  c.State,
			Detail: c.Detail,
		})
	}
	return MCPDoctorView{
		Available:  true,
		Checks:     checks,
		NotChecked: res.NotChecked,
	}, nil
}

// DefaultMCPPlanner provides safe offline default planning and restore previews.
type DefaultMCPPlanner struct{}

// PreviewPlan provides the production, read-only configuration plan preview.
func (DefaultMCPPlanner) PreviewPlan(ctx context.Context, home, configPath string, selected []string, auth map[string]string) (MCPPlanPreviewView, error) {
	req, err := appops.NewMCPPlanRequest(home, configPath, selected, auth)
	if err != nil {
		return MCPPlanPreviewView{}, err
	}
	res, err := appops.MCPPlanPreview(req)
	if err != nil {
		return MCPPlanPreviewView{}, err
	}
	return mapMCPPlanView(res.Plan, res.Token), nil
}

// ListBackups lists managed MCP backups from the private MCP backup root.
func (DefaultMCPPlanner) ListBackups(ctx context.Context, home string) ([]MCPBackupView, error) {
	req, err := appops.NewMCPRestoreListRequest(home)
	if err != nil {
		return nil, err
	}
	backups, err := appops.MCPRestoreList(req)
	if err != nil {
		return nil, err
	}
	out := make([]MCPBackupView, 0, len(backups))
	for _, backup := range backups {
		out = append(out, MCPBackupView{
			ID:        backup.ID,
			Path:      backup.Path,
			CreatedAt: backup.CreatedAt,
			Servers:   append([]string(nil), backup.Servers...),
		})
	}
	return out, nil
}

// PreviewRestore previews restoring one managed MCP backup.
func (DefaultMCPPlanner) PreviewRestore(ctx context.Context, home, backupID string) (MCPRestorePreviewView, error) {
	req, err := appops.NewMCPRestorePreviewRequest(home, "", backupID)
	if err != nil {
		return MCPRestorePreviewView{}, err
	}
	res, err := appops.MCPRestorePreview(req)
	if err != nil {
		return MCPRestorePreviewView{}, err
	}
	return mapMCPRestorePreviewView(res.Preview, res.Token), nil
}

func mapMCPPlanView(plan *skillsync.MCPPlan, token string) MCPPlanPreviewView {
	if plan == nil {
		return MCPPlanPreviewView{}
	}
	entries := make([]MCPEntryDiffView, 0, len(plan.Entries))
	diffLines := []string{
		fmt.Sprintf("--- %s (current)", plan.TargetPath),
		fmt.Sprintf("+++ %s (plan)", plan.TargetPath),
	}
	for _, entry := range plan.Entries {
		entryDiff := []string{}
		if entry.OldText != "" {
			entryDiff = append(entryDiff, "-"+entry.OldText)
		}
		if entry.NewText != "" {
			entryDiff = append(entryDiff, "+"+entry.NewText)
		}
		entries = append(entries, MCPEntryDiffView{
			Name:      entry.Name,
			Op:        entry.Op,
			Reason:    entry.Reason,
			OldText:   entry.OldText,
			NewText:   entry.NewText,
			DiffLines: entryDiff,
		})
		diffLines = append(diffLines, entryDiff...)
	}
	return MCPPlanPreviewView{
		Available:       true,
		TargetPath:      plan.TargetPath,
		Format:          plan.Format,
		Schema:          plan.Schema,
		BaseDigest:      plan.BaseDigest,
		PlanDigest:      plan.PlanDigest,
		Selected:        append([]string(nil), plan.Selected...),
		Entries:         entries,
		Conflicts:       append([]string(nil), plan.Conflicts...),
		AuthChoices:     cloneStringMap(plan.AuthChoices),
		RuntimeVersion:  plan.RuntimeVersion,
		RuntimeEvidence: plan.RuntimeEvidence,
		NoOp:            plan.NoOp,
		NotChecked:      append([]string(nil), plan.NotChecked...),
		Token:           token,
		RawPlan:         plan,
		DiffLines:       diffLines,
	}
}

func mapMCPRestorePreviewView(preview *skillsync.MCPRestorePreview, token string) MCPRestorePreviewView {
	if preview == nil {
		return MCPRestorePreviewView{}
	}
	entries := make([]MCPEntryDiffView, 0, len(preview.Entries))
	diffLines := []string{
		fmt.Sprintf("--- %s (current)", preview.TargetPath),
		fmt.Sprintf("+++ backup %s", preview.BackupID),
	}
	for _, entry := range preview.Entries {
		entryDiff := []string{}
		if entry.OldText != "" {
			entryDiff = append(entryDiff, "-"+entry.OldText)
		}
		if entry.NewText != "" {
			entryDiff = append(entryDiff, "+"+entry.NewText)
		}
		entries = append(entries, MCPEntryDiffView{
			Name:      entry.Name,
			Op:        entry.Op,
			Reason:    entry.Reason,
			OldText:   entry.OldText,
			NewText:   entry.NewText,
			DiffLines: entryDiff,
		})
		diffLines = append(diffLines, entryDiff...)
	}
	return MCPRestorePreviewView{
		Available:  true,
		BackupID:   preview.BackupID,
		TargetPath: preview.TargetPath,
		BaseDigest: preview.BaseDigest,
		Token:      token,
		Entries:    entries,
		Conflicts:  append([]string(nil), preview.Conflicts...),
		RawPreview: preview,
		DiffLines:  diffLines,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// defaultServerViews returns the three canonical rows when no status is available.
func defaultServerViews() []MCPServerView {
	return []MCPServerView{
		{
			Name:              "codegraph",
			DisplayName:       "CodeGraph",
			Kind:              "local",
			Purpose:           "repository code intelligence",
			ConfigState:       "not configured",
			Prerequisite:      "codegraph CLI",
			PrerequisiteMet:   false,
			VerificationState: "connection not tested",
			NextAction:        "Select and preview configuration",
		},
		{
			Name:              "context7",
			DisplayName:       "Context7",
			Kind:              "remote",
			Purpose:           "hosted library documentation",
			ConfigState:       "not configured",
			Prerequisite:      "none",
			PrerequisiteMet:   true,
			AuthMode:          "oauth",
			VerificationState: "connection not tested",
			NextAction:        "Select and choose OAuth or API-key",
		},
		{
			Name:              "exa",
			DisplayName:       "Exa",
			Kind:              "remote",
			Purpose:           "hosted web research",
			ConfigState:       "not configured",
			Prerequisite:      "none",
			PrerequisiteMet:   true,
			AuthMode:          "keyless",
			VerificationState: "connection not tested",
			NextAction:        "Select and acknowledge keyless limits",
		},
	}
}

// mapStatusView converts an internal skillsync.MCPStatus into our MCPStatusView.
func mapStatusView(s *skillsync.MCPStatus) MCPStatusView {
	if s == nil {
		return MCPStatusView{
			Available: false,
			Servers:   defaultServerViews(),
		}
	}

	views := make([]MCPServerView, 0, 3)
	nameOrder := []string{"codegraph", "context7", "exa"}
	serverMap := make(map[string]skillsync.MCPServerStatus, len(s.Servers))
	for _, srv := range s.Servers {
		serverMap[srv.Name] = srv
	}

	displayNames := map[string]string{
		"codegraph": "CodeGraph",
		"context7":  "Context7",
		"exa":       "Exa",
	}

	for _, name := range nameOrder {
		srv, exists := serverMap[name]
		if !exists {
			for _, def := range defaultServerViews() {
				if def.Name == name {
					views = append(views, def)
					break
				}
			}
			continue
		}

		stateStr := normalizeMCPState(srv.State)
		prereq := srv.Prerequisite
		if prereq == "" {
			prereq = "none"
		}

		nextAction := "Preview configuration"
		switch stateStr {
		case "configured":
			nextAction = "Up to date (leave or edit in Select)"
		case "missing-prerequisite":
			nextAction = "Install codegraph CLI before configuring"
		case "conflict":
			nextAction = "Resolve unmanaged name collision"
		case "schema-mismatch":
			nextAction = "Inspect schema mismatch in Diagnostics"
		case "disabled":
			nextAction = "Managed disabled (leave or reconfigure)"
		case "not-configured":
			nextAction = "Select and preview configuration"
		}

		views = append(views, MCPServerView{
			Name:              srv.Name,
			DisplayName:       displayNames[srv.Name],
			Kind:              srv.Kind,
			Purpose:           srv.Purpose,
			ConfigState:       stateStr,
			Prerequisite:      prereq,
			PrerequisiteMet:   srv.PrerequisiteMet,
			AuthMode:          srv.AuthMode,
			VerificationState: "connection not tested",
			NextAction:        nextAction,
			Managed:           srv.Managed,
			Disabled:          srv.Disabled,
			Conflict:          srv.Conflict,
			IndexState:        srv.IndexState,
		})
	}

	return MCPStatusView{
		Available:        true,
		Home:             s.Home,
		ConfigPath:       s.ConfigPath,
		ConfigFormat:     s.ConfigFormat,
		Schema:           s.Schema,
		ConfigDigest:     s.ConfigDigest,
		Ambiguity:        s.Ambiguity,
		LegacyReason:     s.LegacyReason,
		Servers:          views,
		CodeGraphBinary:  s.CodeGraphBinary,
		CodeGraphVersion: s.CodeGraphVersion,
		OpenCodeVersion:  s.OpenCodeVersion,
		NotChecked:       s.NotChecked,
		RawStatus:        s,
	}
}

// normalizeMCPState standardizes the state string for rendering.
func normalizeMCPState(s string) string {
	state := strings.ToLower(strings.TrimSpace(s))
	switch state {
	case "configured", "disabled", "not-configured", "missing-prerequisite", "auth-not-checked", "conflict", "schema-mismatch", "unsafe-target":
		return state
	case "":
		return "not-configured"
	default:
		return state
	}
}
