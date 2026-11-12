package appops

// Feature 015 MCP-A — typed read-only MCP operation seam.
//
// This file mirrors the Feature 013/014/015 request/result pattern for the MCP
// engine: immutable, validated, absolute-path requests; exported read-only
// accessors; and read operations that never write. It is deliberately smaller
// than the fleet seam because MCP-A authorizes no write surface at all: there
// is no apply request, no restore request and no preview token here. A status,
// doctor or preview result therefore authorizes nothing and cannot be replayed
// into a mutation.
//
// Everything in this file is offline by construction. The only process-adjacent
// evidence is skillsync.BuildMCPStatus's bounded exec.LookPath lookup; appops
// adds a local os.Stat for an explicitly supplied binary path and nothing else.
// No binary is executed, no network endpoint is contacted, no model is invoked
// and no package manager is run. Configuration contents, headers, keys and
// credential values are never copied into a doctor detail.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"agentic-sdd/internal/skillsync"
)

// MCP doctor check states. They are the only states a check may report, and
// they describe evidence rather than success: not-checked is not a failure and
// a warn is never rendered as a pass.
const (
	mcpCheckPass       = "pass"
	mcpCheckWarn       = "warn"
	mcpCheckBlocked    = "blocked"
	mcpCheckNotChecked = "not-checked"
)

// MCPStatusRequest is one immutable, validated read of the MCP status
// projection. Construct it with NewMCPStatusRequest and read it through the
// accessors.
type MCPStatusRequest struct {
	home        string
	configPath  string
	projectPath string
}

// NewMCPStatusRequest validates and absolutizes one read-only status request.
// home is required; configPath and projectPath may be empty, in which case the
// engine resolves its documented default target and reports the project index
// as not inspected. No file is read and nothing is written here.
func NewMCPStatusRequest(home, configPath, projectPath string) (MCPStatusRequest, error) {
	if home == "" {
		return MCPStatusRequest{}, errors.New("mcp status request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPStatusRequest{}, err
	}
	absConfig := ""
	if configPath != "" {
		if absConfig, err = filepath.Abs(configPath); err != nil {
			return MCPStatusRequest{}, err
		}
	}
	absProject := ""
	if projectPath != "" {
		if absProject, err = filepath.Abs(projectPath); err != nil {
			return MCPStatusRequest{}, err
		}
	}
	return MCPStatusRequest{home: absHome, configPath: absConfig, projectPath: absProject}, nil
}

// Home returns the absolute home directory the status is resolved against.
func (r MCPStatusRequest) Home() string { return r.home }

// ConfigPath returns the absolute explicit configuration path, or "" for the
// engine's documented default target.
func (r MCPStatusRequest) ConfigPath() string { return r.configPath }

// ProjectPath returns the absolute project whose CodeGraph index is inspected,
// or "" when no project was supplied.
func (r MCPStatusRequest) ProjectPath() string { return r.projectPath }

// MCPStatusResult is the frozen read-only MCP projection produced by MCPStatus.
// Status is a fresh projection; callers must treat it as read-only. There is
// deliberately no preview token: a status read authorizes no write.
type MCPStatusResult struct {
	Status *skillsync.MCPStatus
}

// MCPStatus runs skillsync.BuildMCPStatus. It performs no configuration write,
// no backup, no model call, no package installation and no remote contact; the
// engine may perform a bounded local executable lookup for prerequisite
// evidence only.
func MCPStatus(req MCPStatusRequest) (MCPStatusResult, error) {
	status, err := skillsync.BuildMCPStatus(req.home, req.configPath, req.projectPath)
	if err != nil {
		return MCPStatusResult{}, err
	}
	return MCPStatusResult{Status: status}, nil
}

// MCPDoctorRequest is one immutable, validated offline doctor request. Empty
// openCodeBin and codeGraphBin mean "use the engine's local lookup result";
// supplying either path keeps the report explicit without executing it.
type MCPDoctorRequest struct {
	home         string
	configPath   string
	projectPath  string
	openCodeBin  string
	codeGraphBin string
}

// NewMCPDoctorRequest validates and absolutizes one offline doctor request.
// home is required; every non-empty path is made absolute. The request never
// inspects the filesystem while it is built.
func NewMCPDoctorRequest(home, configPath, projectPath, openCodeBin, codeGraphBin string) (MCPDoctorRequest, error) {
	if home == "" {
		return MCPDoctorRequest{}, errors.New("mcp doctor request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPDoctorRequest{}, err
	}
	abs := func(value string) (string, error) {
		if value == "" {
			return "", nil
		}
		return filepath.Abs(value)
	}
	absConfig, err := abs(configPath)
	if err != nil {
		return MCPDoctorRequest{}, err
	}
	absProject, err := abs(projectPath)
	if err != nil {
		return MCPDoctorRequest{}, err
	}
	absOpenCode, err := abs(openCodeBin)
	if err != nil {
		return MCPDoctorRequest{}, err
	}
	absCodeGraph, err := abs(codeGraphBin)
	if err != nil {
		return MCPDoctorRequest{}, err
	}
	return MCPDoctorRequest{
		home:         absHome,
		configPath:   absConfig,
		projectPath:  absProject,
		openCodeBin:  absOpenCode,
		codeGraphBin: absCodeGraph,
	}, nil
}

// Home returns the absolute home directory the doctor is resolved against.
func (r MCPDoctorRequest) Home() string { return r.home }

// ConfigPath returns the absolute explicit configuration path, or "" for the
// engine's documented default target.
func (r MCPDoctorRequest) ConfigPath() string { return r.configPath }

// ProjectPath returns the absolute project whose CodeGraph index is inspected,
// or "" when no project was supplied.
func (r MCPDoctorRequest) ProjectPath() string { return r.projectPath }

// OpenCodeBin returns the explicit local OpenCode binary path, or "". The
// offline doctor never executes it.
func (r MCPDoctorRequest) OpenCodeBin() string { return r.openCodeBin }

// CodeGraphBin returns the explicit local CodeGraph binary path, or "". The
// offline doctor stats it as presence evidence and never executes it.
func (r MCPDoctorRequest) CodeGraphBin() string { return r.codeGraphBin }

// MCPDoctorCheck is one deterministic, redacted diagnostic line. State is one
// of pass, warn, blocked or not-checked. Detail never contains configuration
// text, headers, keys, credential values or a home path the caller did not
// supply.
type MCPDoctorCheck struct {
	Name   string
	State  string
	Detail string
}

// MCPDoctorResult is the frozen offline diagnostic report. Status is the same
// read-only projection MCPStatus returns; Checks are in fixed order; NotChecked
// names every evidence class this offline path did not perform.
type MCPDoctorResult struct {
	Status     *skillsync.MCPStatus
	Checks     []MCPDoctorCheck
	NotChecked []string
}

// MCPDoctor builds the read-only status projection and labels it with offline
// diagnostics. It executes no binary, starts no MCP server, contacts no remote
// endpoint, runs no package manager and invokes no model, so runtime version,
// Context7/Exa authentication and remote connectivity are reported as not
// checked rather than inferred from configuration.
func MCPDoctor(req MCPDoctorRequest) (MCPDoctorResult, error) {
	status, err := skillsync.BuildMCPStatus(req.home, req.configPath, req.projectPath)
	if err != nil {
		return MCPDoctorResult{}, err
	}
	checks := []MCPDoctorCheck{
		mcpConfigTargetCheck(status),
		mcpConfigSchemaCheck(status),
		mcpCodeGraphBinaryCheck(req, status),
		mcpCodeGraphIndexCheck(req, status),
		mcpAuthCheck(status, "context7", "Context7"),
		mcpAuthCheck(status, "exa", "Exa"),
		mcpRuntimeCheck(req, status),
		mcpConnectivityCheck(),
	}
	return MCPDoctorResult{
		Status:     status,
		Checks:     checks,
		NotChecked: mcpNotChecked(status),
	}, nil
}

// MCPPreviewRequest is one immutable, validated read-only recipe preview. The
// selection is copied at construction, so a caller cannot change the request
// after validation. appops retains no draft between calls.
type MCPPreviewRequest struct {
	home       string
	configPath string
	selected   []string
}

// NewMCPPreviewRequest validates and absolutizes one read-only preview request.
// home is required; configPath may be empty for the documented default target.
// Server names are copied and left for the engine to validate: appops does not
// duplicate MCP vocabularies, and an unknown name is an engine refusal rather
// than a silently filtered selection.
func NewMCPPreviewRequest(home, configPath string, servers []string) (MCPPreviewRequest, error) {
	if home == "" {
		return MCPPreviewRequest{}, errors.New("mcp preview request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPPreviewRequest{}, err
	}
	absConfig := ""
	if configPath != "" {
		if absConfig, err = filepath.Abs(configPath); err != nil {
			return MCPPreviewRequest{}, err
		}
	}
	return MCPPreviewRequest{
		home:       absHome,
		configPath: absConfig,
		selected:   mcpCopySelection(servers),
	}, nil
}

// Home returns the absolute home directory the preview is resolved against.
func (r MCPPreviewRequest) Home() string { return r.home }

// ConfigPath returns the absolute explicit configuration path, or "" for the
// engine's documented default target.
func (r MCPPreviewRequest) ConfigPath() string { return r.configPath }

// Selected returns a copy of the requested server names. Changing the returned
// slice cannot change the request. A nil selection stays nil and an empty
// selection stays empty, so the engine can keep its own default semantics.
func (r MCPPreviewRequest) Selected() []string {
	if r.selected == nil {
		return nil
	}
	return append([]string{}, r.selected...)
}

// MCPPreviewResult is the read-only static recipe comparison. It carries no
// apply token, base digest or write identity: in MCP-A a preview authorizes
// nothing and the engine writes nothing.
type MCPPreviewResult struct {
	Preview *skillsync.MCPPreview
}

// MCPPreview runs skillsync.PreviewMCPConfig. It reads the current
// configuration and reports the static recipe comparison; it creates no
// configuration, writes no backup or manifest, executes no binary and contacts
// no remote endpoint.
func MCPPreview(req MCPPreviewRequest) (MCPPreviewResult, error) {
	preview, err := skillsync.PreviewMCPConfig(req.home, req.configPath, req.selected)
	if err != nil {
		return MCPPreviewResult{}, err
	}
	return MCPPreviewResult{Preview: preview}, nil
}

// mcpConfigTargetCheck reports whether a single inspectable target exists.
// Missing configuration is a warning with manual guidance, not a pass and not a
// licence to create a file; an ambiguous .json/.jsonc pair is blocked.
func mcpConfigTargetCheck(status *skillsync.MCPStatus) MCPDoctorCheck {
	const name = "config-target"
	if status == nil {
		return MCPDoctorCheck{Name: name, State: mcpCheckNotChecked, Detail: "MCP status was not produced"}
	}
	if strings.TrimSpace(status.Ambiguity) != "" {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckBlocked,
			Detail: "both opencode.json and opencode.jsonc targets appear present; choose one target explicitly; no file was changed",
		}
	}
	for _, server := range status.Servers {
		if normalizeMCPState(string(server.State)) == "unsafe-target" {
			return MCPDoctorCheck{
				Name:   name,
				State:  mcpCheckBlocked,
				Detail: "the configuration target is unsafe (symlink, non-regular file or out-of-scope path); no file was read through it",
			}
		}
	}
	if mcpConfigAbsent(status) {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: "no global OpenCode configuration was found; a manual snippet is required and no file is created",
		}
	}
	return MCPDoctorCheck{
		Name:   name,
		State:  mcpCheckPass,
		Detail: "one configuration target was detected and inspected",
	}
}

// mcpConfigSchemaCheck reports V1/unknown schema from engine evidence. V1 is
// blocked with migration guidance; it is never converted here.
func mcpConfigSchemaCheck(status *skillsync.MCPStatus) MCPDoctorCheck {
	const name = "config-schema"
	if status == nil {
		return MCPDoctorCheck{Name: name, State: mcpCheckNotChecked, Detail: "MCP status was not produced"}
	}
	if strings.TrimSpace(status.LegacyReason) != "" {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckBlocked,
			Detail: "V1 mcp.<name>/enabled configuration was detected; migration is manual and no entry was converted",
		}
	}
	for _, server := range status.Servers {
		if normalizeMCPState(string(server.State)) == "schema-mismatch" {
			return MCPDoctorCheck{
				Name:   name,
				State:  mcpCheckBlocked,
				Detail: "the mcp.servers schema is unsupported or invalid; no automatic change is available",
			}
		}
	}
	if mcpConfigAbsent(status) {
		return MCPDoctorCheck{Name: name, State: mcpCheckNotChecked, Detail: "no configuration was found to validate"}
	}
	switch normalizeMCPState(status.Schema) {
	case "v2":
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckPass,
			Detail: "the V2 mcp.servers configuration schema was inspected",
		}
	case "invalid", "unknown":
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: "the configuration schema could not be validated; no automatic change is available",
		}
	default:
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: "the configuration schema is not recognized as a supported V2 schema; no automatic change is available",
		}
	}
}

// mcpCodeGraphBinaryCheck reports local executable presence only. It stats an
// explicitly supplied path and otherwise trusts the engine's bounded LookPath
// result; it never runs the binary, so no version claim is made.
func mcpCodeGraphBinaryCheck(req MCPDoctorRequest, status *skillsync.MCPStatus) MCPDoctorCheck {
	const name = "codegraph-binary"
	if req.codeGraphBin != "" {
		info, err := os.Stat(req.codeGraphBin)
		if err != nil || info.IsDir() || !info.Mode().IsRegular() {
			return MCPDoctorCheck{
				Name:   name,
				State:  mcpCheckBlocked,
				Detail: "the supplied CodeGraph binary path is not a regular file; installation is manual and no package manager is run",
			}
		}
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckPass,
			Detail: "the supplied CodeGraph binary path is present; its version was not executed offline",
		}
	}
	if status != nil {
		if strings.TrimSpace(status.CodeGraphBinary) != "" {
			return MCPDoctorCheck{
				Name:   name,
				State:  mcpCheckPass,
				Detail: "a CodeGraph executable was found by local lookup; its version was not executed offline",
			}
		}
		if codegraph, ok := mcpServerByName(status, "codegraph"); ok && codegraph.PrerequisiteMet {
			return MCPDoctorCheck{
				Name:   name,
				State:  mcpCheckPass,
				Detail: "the CodeGraph prerequisite is reported met by local lookup; its version was not executed offline",
			}
		}
	}
	return MCPDoctorCheck{
		Name:   name,
		State:  mcpCheckBlocked,
		Detail: "no CodeGraph executable was found by local lookup; install it separately and re-run doctor",
	}
}

// mcpCodeGraphIndexCheck reports project index presence without running
// codegraph. Absence and unverified freshness are warnings, never a pass.
func mcpCodeGraphIndexCheck(req MCPDoctorRequest, status *skillsync.MCPStatus) MCPDoctorCheck {
	const name = "codegraph-index"
	if req.projectPath == "" {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckNotChecked,
			Detail: "no project path was supplied; CodeGraph index state was not inspected",
		}
	}
	if status == nil {
		return MCPDoctorCheck{Name: name, State: mcpCheckNotChecked, Detail: "MCP status was not produced"}
	}
	codegraph, ok := mcpServerByName(status, "codegraph")
	if !ok {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckNotChecked,
			Detail: "the CodeGraph server was not inspected; index state is unknown",
		}
	}
	index := normalizeMCPState(codegraph.IndexState)
	switch {
	case index == "", index == "not-checked":
		return MCPDoctorCheck{Name: name, State: mcpCheckNotChecked, Detail: "no project index evidence was reported"}
	case strings.Contains(index, "absent"), strings.Contains(index, "not-initialized"), strings.Contains(index, "missing"):
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: "the project has no CodeGraph index; initialize it explicitly with codegraph init -y when you intend to index this checkout",
		}
	case strings.Contains(index, "stale"):
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: "the project index may be stale; freshness is only reported by an explicit local check and was not verified offline",
		}
	case strings.Contains(index, "present"), strings.Contains(index, "freshness"):
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: "a project index is present; freshness was not checked offline",
		}
	default:
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckNotChecked,
			Detail: "the reported index state is not recognized; freshness was not checked offline",
		}
	}
}

// mcpAuthCheck labels authentication from local evidence only. Configuration
// presence never proves authentication, so this check never reports pass.
func mcpAuthCheck(status *skillsync.MCPStatus, serverName, label string) MCPDoctorCheck {
	name := strings.ToLower(label) + "-auth"
	if status == nil {
		return MCPDoctorCheck{Name: name, State: mcpCheckNotChecked, Detail: label + " authentication was not checked"}
	}
	server, ok := mcpServerByName(status, serverName)
	if !ok || strings.Contains(normalizeMCPState(string(server.State)), "not-configured") {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckNotChecked,
			Detail: label + " is not configured; authentication was not checked",
		}
	}
	if state := normalizeMCPState(string(server.State)); strings.Contains(state, "needs-auth") {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: label + " reports needing authentication from local runtime evidence; this offline doctor started no login",
		}
	}
	if serverName == "exa" && strings.EqualFold(strings.TrimSpace(server.AuthMode), "keyless") {
		return MCPDoctorCheck{
			Name:   name,
			State:  mcpCheckWarn,
			Detail: "Exa is configured for keyless access, which is rate-limited; authentication was not checked and no upgrade was attempted",
		}
	}
	return MCPDoctorCheck{
		Name:   name,
		State:  mcpCheckWarn,
		Detail: label + " authentication was not checked; configuration does not prove authentication",
	}
}

// mcpRuntimeCheck states plainly that no runtime probe ran. Supplying an
// explicit binary does not change that: the offline doctor never executes it.
func mcpRuntimeCheck(req MCPDoctorRequest, status *skillsync.MCPStatus) MCPDoctorCheck {
	const name = "opencode-runtime"
	detail := "the OpenCode runtime version was not checked; the offline doctor executes no binary"
	if req.openCodeBin != "" {
		detail = "an explicit OpenCode binary path was supplied but not executed; the runtime version was not checked"
	} else if status != nil {
		if version := strings.TrimSpace(status.OpenCodeVersion); version != "" && !strings.EqualFold(version, "not-checked") {
			detail = "a locally reported OpenCode version was observed; connection and runtime behavior were not verified"
		}
	}
	return MCPDoctorCheck{Name: name, State: mcpCheckNotChecked, Detail: detail}
}

// mcpConnectivityCheck is always not-checked in MCP-A: --network/--connect are
// deferred out of this workstream.
func mcpConnectivityCheck() MCPDoctorCheck {
	return MCPDoctorCheck{
		Name:   "remote-connectivity",
		State:  mcpCheckNotChecked,
		Detail: "remote MCP connectivity was not checked; network and connection probes are a separate capability and were not requested",
	}
}

// mcpNotChecked lists every evidence class this offline path did not perform.
func mcpNotChecked(status *skillsync.MCPStatus) []string {
	var out []string
	seen := make(map[string]bool)
	add := func(value string) {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, value)
	}
	if status != nil {
		for _, value := range status.NotChecked {
			add(value)
		}
	}
	add("OpenCode runtime connection")
	add("CodeGraph index freshness")
	return out
}

// mcpConfigAbsent reports whether the engine found no inspectable target. The
// engine uses "none" for an absent default target and an empty format for an
// unprojected status.
func mcpConfigAbsent(status *skillsync.MCPStatus) bool {
	format := normalizeMCPState(status.ConfigFormat)
	return format == "" || format == "none"
}

// mcpServerByName finds one server in report order. Names are matched
// case-insensitively because the recipe name is the stable identity, not the
// display label.
func mcpServerByName(status *skillsync.MCPStatus, name string) (skillsync.MCPServerStatus, bool) {
	if status == nil {
		return skillsync.MCPServerStatus{}, false
	}
	for _, server := range status.Servers {
		if strings.EqualFold(strings.TrimSpace(server.Name), name) {
			return server, true
		}
	}
	return skillsync.MCPServerStatus{}, false
}

// mcpCopySelection preserves nil-versus-empty while making the slice
// independent of the caller's backing array.
func mcpCopySelection(servers []string) []string {
	if servers == nil {
		return nil
	}
	return append([]string{}, servers...)
}

// normalizeMCPState makes state comparison tolerant of the separator and case
// a display layer may use, without accepting a different word.
func normalizeMCPState(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer(" ", "-", "_", "-").Replace(normalized)
	return normalized
}
