package skillsync

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// MCPState represents the configuration and readiness state of an MCP server.
type MCPState = string

const (
	MCPStateNotConfigured       MCPState = "not-configured"
	MCPStateConfigured          MCPState = "configured"
	MCPStateDisabled            MCPState = "disabled"
	MCPStateMissingPrerequisite MCPState = "missing-prerequisite"
	MCPStateAuthNotChecked      MCPState = "auth-not-checked"
	MCPStateConflict            MCPState = "conflict"
	MCPStateSchemaMismatch      MCPState = "schema-mismatch"
	MCPStateUnsafeTarget        MCPState = "unsafe-target"
	MCPStateUnknown             MCPState = "unknown"
)

// MCPServerKind distinguishes local stdio servers from remote HTTP servers.
type MCPServerKind = string

const (
	MCPServerLocal  MCPServerKind = "local"
	MCPServerRemote MCPServerKind = "remote"
)

// MCPServerSpec describes one supported default MCP server recipe.
type MCPServerSpec struct {
	Name         string
	Kind         MCPServerKind
	Purpose      string
	Prerequisite string
	Recipe       string
}

// MCPDefaultServers returns the three canonical Agentic-SDD MCP servers in
// fixed order: CodeGraph (local), Context7 (remote), and Exa (remote).
func MCPDefaultServers() []MCPServerSpec {
	return []MCPServerSpec{
		{
			Name:         "codegraph",
			Kind:         MCPServerLocal,
			Purpose:      "repository code intelligence",
			Prerequisite: "codegraph CLI",
			Recipe:       `{"type":"local","command":["codegraph","serve","--mcp"],"disabled":false,"codemode":false}`,
		},
		{
			Name:         "context7",
			Kind:         MCPServerRemote,
			Purpose:      "hosted library documentation",
			Prerequisite: "",
			Recipe:       `{"type":"remote","url":"https://mcp.context7.com/mcp/oauth"}`,
		},
		{
			Name:         "exa",
			Kind:         MCPServerRemote,
			Purpose:      "hosted web research",
			Prerequisite: "",
			Recipe:       `{"type":"remote","url":"https://mcp.exa.ai/mcp"}`,
		},
	}
}

// MCPServerStatus holds the projected state and evidence for one MCP server.
type MCPServerStatus struct {
	Name            string
	Kind            MCPServerKind
	Purpose         string
	State           MCPState
	Evidence        string
	Prerequisite    string
	PrerequisiteMet bool
	AuthMode        string
	Managed         bool
	Disabled        bool
	ConfigPath      string
	Conflict        string
	IndexState      string
}

// MCPStatus is the frozen read-only MCP projection across target configuration
// and default server availability.
type MCPStatus struct {
	Home             string
	ConfigPath       string
	ConfigFormat     string
	Schema           string
	ConfigDigest     string
	Ambiguity        string
	LegacyReason     string
	Servers          []MCPServerStatus
	CodeGraphBinary  string
	CodeGraphVersion string
	OpenCodeVersion  string
	NotChecked       []string
}

// MCPPreviewEntry represents the projected configuration action for one server.
type MCPPreviewEntry struct {
	Name   string
	Op     string // Op: add | leave | disabled | conflict
	Reason string
	Recipe string
}

// MCPPreview represents the read-only static recipe comparison.
type MCPPreview struct {
	Home       string
	TargetPath string
	Format     string
	BaseDigest string
	Selected   []string
	Entries    []MCPPreviewEntry
	Conflicts  []string
	NoOp       bool
	NotChecked []string
}

const maxMCPConfigFileBytes = 1024 * 1024 // 1 MiB bounded read

// ErrMCPUsage marks an MCP request that cannot be interpreted as a valid
// read-only request. The CLI maps it to exit 2 (usage/validation); runtime or
// I/O failures remain exit 1.
var ErrMCPUsage = errors.New("MCP usage error")

// BuildMCPStatus evaluates the OpenCode MCP configuration target and projects
// truthful evidence-backed state for CodeGraph, Context7, and Exa. It is
// strictly read-only: no files or directories are created, no configuration is
// written, no binary is executed (only exec.LookPath for presence), and no
// network contact or model invocation occurs.
func BuildMCPStatus(home, configPath, projectPath string) (*MCPStatus, error) {
	if home == "" {
		return nil, fmt.Errorf("%w: home directory is required", ErrMCPUsage)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve home directory: %v", ErrMCPUsage, err)
	}

	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("%w: resolve config path: %v", ErrMCPUsage, err)
		}
	}

	absProject := ""
	if projectPath != "" {
		absProject, err = filepath.Abs(projectPath)
		if err != nil {
			return nil, fmt.Errorf("%w: resolve project path: %v", ErrMCPUsage, err)
		}
	}

	status := &MCPStatus{
		Home:             absHome,
		CodeGraphVersion: "not-checked",
		OpenCodeVersion:  "not-checked",
		NotChecked: []string{
			"OpenCode runtime version",
			"remote connectivity",
			"authentication",
			"CodeGraph version",
		},
	}

	// Probe local CodeGraph binary presence (no execution, LookPath only).
	cgBin, cgLookErr := exec.LookPath("codegraph")
	cgPrereqMet := cgLookErr == nil
	if cgPrereqMet {
		status.CodeGraphBinary = cgBin
	}

	// Probe CodeGraph index presence if projectPath was provided.
	cgIndexState := "not-checked"
	if absProject != "" {
		indexPath := filepath.Join(absProject, ".codegraph")
		if _, statErr := os.Stat(indexPath); statErr == nil {
			cgIndexState = "present"
		} else {
			cgIndexState = "absent"
		}
	}

	// Target resolution
	targetPath, targetFormat, ambiguity, unsafeReason, isMissingExplicit := resolveMCPTarget(absHome, absConfig)
	if ambiguity != "" {
		status.Ambiguity = ambiguity
		status.ConfigPath = ""
		status.ConfigFormat = ""
		status.Schema = ""
		status.Servers = buildAmbiguousOrUnsafeServers(status.Ambiguity, MCPStateConflict, cgPrereqMet, cgIndexState)
		return status, nil
	}

	if unsafeReason != "" {
		status.ConfigPath = targetPath
		status.ConfigFormat = targetFormat
		status.Schema = "unknown"
		status.Servers = buildAmbiguousOrUnsafeServers(unsafeReason, MCPStateUnsafeTarget, cgPrereqMet, cgIndexState)
		return status, nil
	}

	if targetPath == "" {
		// Neither default config file exists
		status.ConfigPath = ""
		status.ConfigFormat = ""
		status.Schema = "none"
		status.ConfigDigest = ""
		status.Servers = buildDefaultNotConfiguredServers(cgPrereqMet, cgIndexState, "")
		return status, nil
	}

	status.ConfigPath = targetPath
	status.ConfigFormat = targetFormat

	if isMissingExplicit {
		// Explicit path was provided but does not exist on disk
		status.ConfigFormat = "none"
		status.Schema = "none"
		status.ConfigDigest = ""
		status.Servers = buildDefaultNotConfiguredServers(cgPrereqMet, cgIndexState, targetPath)
		return status, nil
	}

	// Target exists and is a regular file. Read bounded bytes.
	f, err := os.Open(targetPath)
	if err != nil {
		status.Schema = "unknown"
		reason := fmt.Sprintf("cannot open config file: %v", err)
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateConflict, cgPrereqMet, cgIndexState)
		return status, nil
	}
	defer f.Close()

	raw, err := io.ReadAll(io.LimitReader(f, maxMCPConfigFileBytes+1))
	if err != nil {
		status.Schema = "unknown"
		reason := fmt.Sprintf("cannot read config file: %v", err)
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateConflict, cgPrereqMet, cgIndexState)
		return status, nil
	}

	if len(raw) > maxMCPConfigFileBytes {
		status.Schema = "invalid"
		status.ConfigDigest = fmt.Sprintf("%x", sha256.Sum256(raw))
		reason := fmt.Sprintf("config file exceeds 1 MiB limit (%d bytes)", len(raw))
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateUnknown, cgPrereqMet, cgIndexState)
		return status, nil
	}

	status.ConfigDigest = fmt.Sprintf("%x", sha256.Sum256(raw))

	if !utf8.Valid(raw) {
		status.Schema = "invalid"
		reason := "config file is not valid UTF-8"
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateUnknown, cgPrereqMet, cgIndexState)
		return status, nil
	}

	jsonBytes := raw
	if targetFormat == "jsonc" {
		stripped, stripErr := stripJSONC(raw)
		if stripErr != nil {
			status.Schema = "invalid"
			reason := fmt.Sprintf("malformed JSONC: %v", stripErr)
			status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateUnknown, cgPrereqMet, cgIndexState)
			return status, nil
		}
		jsonBytes = stripped
	}

	if dupErr := detectDuplicateKeys(jsonBytes); dupErr != nil {
		status.Schema = "invalid"
		reason := dupErr.Error()
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateUnknown, cgPrereqMet, cgIndexState)
		return status, nil
	}

	var root map[string]any
	dec := json.NewDecoder(bytes.NewReader(jsonBytes))
	if err := dec.Decode(&root); err != nil {
		status.Schema = "invalid"
		reason := fmt.Sprintf("invalid JSON: %v", err)
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateUnknown, cgPrereqMet, cgIndexState)
		return status, nil
	}

	// Schema evaluation
	mcpVal, hasMCP := root["mcp"]
	if !hasMCP {
		// Valid config without an mcp block yet
		status.Schema = "v2"
		status.Servers = buildDefaultNotConfiguredServers(cgPrereqMet, cgIndexState, targetPath)
		return status, nil
	}

	mcpMap, ok := mcpVal.(map[string]any)
	if !ok {
		status.Schema = "invalid"
		reason := "mcp field is not an object"
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateUnknown, cgPrereqMet, cgIndexState)
		return status, nil
	}

	serversVal, hasServers := mcpMap["servers"]
	if !hasServers {
		// Check for V1 schema: mcp.<name> shape with child keys
		if len(mcpMap) > 0 {
			status.Schema = "v1"
			status.LegacyReason = "legacy V1 mcp.<name> schema detected; OpenCode V2 requires mcp.servers"
			status.Servers = buildAmbiguousOrUnsafeServers(status.LegacyReason, MCPStateSchemaMismatch, cgPrereqMet, cgIndexState)
			return status, nil
		}
		// "mcp": {}
		status.Schema = "v2"
		status.Servers = buildDefaultNotConfiguredServers(cgPrereqMet, cgIndexState, targetPath)
		return status, nil
	}

	serversMap, ok := serversVal.(map[string]any)
	if !ok {
		status.Schema = "invalid"
		reason := "mcp.servers field is not an object"
		status.Servers = buildAmbiguousOrUnsafeServers(reason, MCPStateUnknown, cgPrereqMet, cgIndexState)
		return status, nil
	}

	status.Schema = "v2"

	// Inspect default servers within mcp.servers
	specs := MCPDefaultServers()
	serverStatuses := make([]MCPServerStatus, 0, len(specs))

	for _, spec := range specs {
		srvStatus := MCPServerStatus{
			Name:         spec.Name,
			Kind:         spec.Kind,
			Purpose:      spec.Purpose,
			Prerequisite: spec.Prerequisite,
			ConfigPath:   targetPath,
			Managed:      false,
			Evidence:     "static",
			AuthMode:     "not-checked",
		}

		if spec.Name == "codegraph" {
			srvStatus.Evidence = "local-probe"
			srvStatus.PrerequisiteMet = cgPrereqMet
			srvStatus.IndexState = cgIndexState
		} else {
			srvStatus.PrerequisiteMet = true
		}

		entryVal, present := serversMap[spec.Name]
		if !present {
			srvStatus.State = MCPStateNotConfigured
			serverStatuses = append(serverStatuses, srvStatus)
			continue
		}

		entryMap, ok := entryVal.(map[string]any)
		if !ok {
			srvStatus.State = MCPStateConflict
			srvStatus.Conflict = fmt.Sprintf("server entry %q is not an object", spec.Name)
			serverStatuses = append(serverStatuses, srvStatus)
			continue
		}

		disabled := false
		if dVal, ok := entryMap["disabled"]; ok {
			if dBool, ok := dVal.(bool); ok {
				disabled = dBool
			}
		}
		srvStatus.Disabled = disabled

		if disabled {
			srvStatus.State = MCPStateDisabled
		} else {
			srvStatus.State = MCPStateConfigured
			if spec.Name == "codegraph" && !cgPrereqMet {
				srvStatus.State = MCPStateMissingPrerequisite
			}
		}

		// AuthMode detection
		if spec.Name == "context7" {
			srvStatus.AuthMode = detectContext7AuthMode(entryMap)
		} else if spec.Name == "exa" {
			srvStatus.AuthMode = detectExaAuthMode(entryMap)
		}

		serverStatuses = append(serverStatuses, srvStatus)
	}

	status.Servers = serverStatuses
	return status, nil
}

// PreviewMCPConfig produces the read-only static recipe comparison for the
// selected servers. It performs no write, creates no files or directories,
// generates no token, and makes no network contact or process execution.
func PreviewMCPConfig(home, configPath string, selected []string) (*MCPPreview, error) {
	if home == "" {
		return nil, fmt.Errorf("%w: home directory is required", ErrMCPUsage)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve home directory: %v", ErrMCPUsage, err)
	}

	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("%w: resolve config path: %v", ErrMCPUsage, err)
		}
	}

	validNames := map[string]bool{
		"codegraph": true,
		"context7":  true,
		"exa":       true,
	}

	var selectedList []string
	if len(selected) == 0 {
		selectedList = []string{"codegraph", "context7", "exa"}
	} else {
		for _, name := range selected {
			trimmed := strings.TrimSpace(name)
			if !validNames[trimmed] {
				return nil, fmt.Errorf("%w: unknown server name %q (expected one of codegraph, context7, exa)", ErrMCPUsage, name)
			}
			selectedList = append(selectedList, trimmed)
		}
	}

	status, err := BuildMCPStatus(absHome, absConfig, "")
	if err != nil {
		return nil, err
	}

	targetPath := status.ConfigPath
	format := status.ConfigFormat
	if status.Ambiguity != "" {
		// Do not propose either file when the target is ambiguous. The conflict
		// below carries the exact reason; choosing a side is a user decision.
		targetPath = ""
		format = ""
	} else if targetPath == "" {
		if absConfig != "" {
			targetPath = absConfig
			if strings.HasSuffix(absConfig, ".jsonc") {
				format = "jsonc"
			} else {
				format = "json"
			}
		} else {
			targetPath = filepath.Join(absHome, ".config", "opencode", "opencode.json")
			format = "json"
		}
	}

	preview := &MCPPreview{
		Home:       absHome,
		TargetPath: targetPath,
		Format:     format,
		BaseDigest: status.ConfigDigest,
		Selected:   selectedList,
		NotChecked: append([]string{}, status.NotChecked...),
	}

	if status.Ambiguity != "" {
		preview.Conflicts = append(preview.Conflicts, status.Ambiguity)
	}
	if status.LegacyReason != "" {
		preview.Conflicts = append(preview.Conflicts, status.LegacyReason)
	}

	selectedSet := make(map[string]bool, len(selectedList))
	for _, name := range selectedList {
		selectedSet[name] = true
	}

	statusMap := make(map[string]MCPServerStatus, len(status.Servers))
	for _, srv := range status.Servers {
		statusMap[srv.Name] = srv
	}

	specs := MCPDefaultServers()
	for _, spec := range specs {
		if !selectedSet[spec.Name] {
			continue
		}

		srv, exists := statusMap[spec.Name]
		if !exists {
			srv = MCPServerStatus{State: MCPStateNotConfigured}
		}

		entry := MCPPreviewEntry{
			Name: spec.Name,
		}

		switch srv.State {
		case MCPStateNotConfigured:
			entry.Op = "add"
			entry.Reason = "not configured"
			entry.Recipe = spec.Recipe
		case MCPStateConfigured, MCPStateMissingPrerequisite:
			entry.Op = "leave"
			entry.Reason = "already configured; exact merge preview is MCP-B"
			entry.Recipe = ""
		case MCPStateDisabled:
			entry.Op = "disabled"
			entry.Reason = "server is disabled; user disabled state is preserved"
			entry.Recipe = ""
		case MCPStateConflict, MCPStateSchemaMismatch, MCPStateUnsafeTarget, MCPStateUnknown:
			entry.Op = "conflict"
			entry.Reason = srv.Conflict
			if entry.Reason == "" {
				entry.Reason = string(srv.State)
			}
			entry.Recipe = ""
			if !stringSliceContains(preview.Conflicts, entry.Reason) {
				preview.Conflicts = append(preview.Conflicts, entry.Reason)
			}
		default:
			entry.Op = "conflict"
			entry.Reason = string(srv.State)
			entry.Recipe = ""
			if !stringSliceContains(preview.Conflicts, entry.Reason) {
				preview.Conflicts = append(preview.Conflicts, entry.Reason)
			}
		}

		preview.Entries = append(preview.Entries, entry)
	}

	// NoOp is true only if all selected entries are leave/disabled and there are no conflicts.
	if len(preview.Conflicts) == 0 && len(preview.Entries) > 0 {
		allLeaveOrDisabled := true
		for _, e := range preview.Entries {
			if e.Op != "leave" && e.Op != "disabled" {
				allLeaveOrDisabled = false
				break
			}
		}
		preview.NoOp = allLeaveOrDisabled
	}

	return preview, nil
}

// resolveMCPTarget identifies the target configuration file path and format.
func resolveMCPTarget(home, explicitConfig string) (targetPath, format, ambiguity, unsafeReason string, isMissingExplicit bool) {
	if explicitConfig != "" {
		ext := filepath.Ext(explicitConfig)
		if ext != ".json" && ext != ".jsonc" {
			return explicitConfig, "unknown", "", fmt.Sprintf("unsupported configuration extension %q; expected .json or .jsonc", ext), false
		}
		fi, err := os.Lstat(explicitConfig)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if ext == ".jsonc" {
					return explicitConfig, "jsonc", "", "", true
				}
				return explicitConfig, "json", "", "", true
			}
			return explicitConfig, "unknown", "", fmt.Sprintf("cannot inspect configuration path: %v", err), false
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return explicitConfig, "unknown", "", fmt.Sprintf("target is a symlink: %s", explicitConfig), false
		}
		if !fi.Mode().IsRegular() {
			return explicitConfig, "unknown", "", fmt.Sprintf("target is not a regular file: %s", explicitConfig), false
		}
		if ext == ".jsonc" {
			return explicitConfig, "jsonc", "", "", false
		}
		return explicitConfig, "json", "", "", false
	}

	jsonPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	jsoncPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")

	fiJSON, errJSON := os.Lstat(jsonPath)
	hasJSON := errJSON == nil
	fiJSONC, errJSONC := os.Lstat(jsoncPath)
	hasJSONC := errJSONC == nil

	if hasJSON && hasJSONC {
		ambiguity = fmt.Sprintf("both opencode.json and opencode.jsonc exist in %s", filepath.Join(home, ".config", "opencode"))
		return "", "", ambiguity, "", false
	}

	if !hasJSON && !hasJSONC {
		return "", "none", "", "", false
	}

	target := jsonPath
	targetFi := fiJSON
	format = "json"
	if hasJSONC {
		target = jsoncPath
		targetFi = fiJSONC
		format = "jsonc"
	}

	if targetFi.Mode()&os.ModeSymlink != 0 {
		return target, format, "", fmt.Sprintf("target is a symlink: %s", target), false
	}
	if !targetFi.Mode().IsRegular() {
		return target, format, "", fmt.Sprintf("target is not a regular file: %s", target), false
	}

	return target, format, "", "", false
}

func buildDefaultNotConfiguredServers(cgPrereqMet bool, cgIndexState, configPath string) []MCPServerStatus {
	specs := MCPDefaultServers()
	out := make([]MCPServerStatus, 0, len(specs))
	for _, spec := range specs {
		srv := MCPServerStatus{
			Name:         spec.Name,
			Kind:         spec.Kind,
			Purpose:      spec.Purpose,
			State:        MCPStateNotConfigured,
			Prerequisite: spec.Prerequisite,
			ConfigPath:   configPath,
			Evidence:     "static",
			AuthMode:     "not-checked",
		}
		if spec.Name == "codegraph" {
			srv.Evidence = "local-probe"
			srv.PrerequisiteMet = cgPrereqMet
			srv.IndexState = cgIndexState
		} else {
			srv.PrerequisiteMet = true
		}
		out = append(out, srv)
	}
	return out
}

func buildAmbiguousOrUnsafeServers(reason string, state MCPState, cgPrereqMet bool, cgIndexState string) []MCPServerStatus {
	specs := MCPDefaultServers()
	out := make([]MCPServerStatus, 0, len(specs))
	for _, spec := range specs {
		srv := MCPServerStatus{
			Name:         spec.Name,
			Kind:         spec.Kind,
			Purpose:      spec.Purpose,
			State:        state,
			Evidence:     "static",
			Prerequisite: spec.Prerequisite,
			Conflict:     reason,
			AuthMode:     "not-checked",
		}
		if spec.Name == "codegraph" {
			srv.Evidence = "local-probe"
			srv.PrerequisiteMet = cgPrereqMet
			srv.IndexState = cgIndexState
		} else {
			srv.PrerequisiteMet = true
		}
		out = append(out, srv)
	}
	return out
}

func detectContext7AuthMode(entryMap map[string]any) string {
	urlVal, _ := entryMap["url"].(string)
	if strings.Contains(urlVal, "/mcp/oauth") {
		return "oauth"
	}
	if headersMap, ok := entryMap["headers"].(map[string]any); ok {
		for k, v := range headersMap {
			if strings.EqualFold(k, "Authorization") {
				if sVal, ok := v.(string); ok && strings.Contains(sVal, "{env:") {
					return "api-key"
				}
			}
		}
	}
	return "not-checked"
}

func detectExaAuthMode(entryMap map[string]any) string {
	urlVal, _ := entryMap["url"].(string)
	if strings.Contains(urlVal, "?login") {
		return "oauth"
	}
	if headersMap, ok := entryMap["headers"].(map[string]any); ok {
		for k, v := range headersMap {
			if strings.EqualFold(k, "x-api-key") || strings.EqualFold(k, "Authorization") {
				if sVal, ok := v.(string); ok && strings.Contains(sVal, "{env:") {
					return "api-key"
				}
			}
		}
	}
	return "keyless"
}

// stripJSONC replaces comments and trailing commas with spaces, preserving
// line offsets and exact string literal contents. It does not modify live files.
func stripJSONC(data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	n := len(out)
	i := 0
	for i < n {
		b := out[i]
		if b == '"' {
			i++
			for i < n {
				if out[i] == '\\' {
					i += 2
					continue
				}
				if out[i] == '"' {
					i++
					break
				}
				i++
			}
		} else if b == '/' && i+1 < n && out[i+1] == '/' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			for i < n && out[i] != '\n' {
				out[i] = ' '
				i++
			}
		} else if b == '/' && i+1 < n && out[i+1] == '*' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			closed := false
			for i+1 < n {
				if out[i] == '*' && out[i+1] == '/' {
					out[i] = ' '
					out[i+1] = ' '
					i += 2
					closed = true
					break
				}
				if out[i] != '\n' && out[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			if !closed {
				return nil, errors.New("unclosed block comment in JSONC")
			}
		} else if b == ',' {
			j := i + 1
			for j < n {
				if out[j] == ' ' || out[j] == '\t' || out[j] == '\n' || out[j] == '\r' {
					j++
					continue
				}
				if out[j] == '/' && j+1 < n && out[j+1] == '/' {
					j += 2
					for j < n && out[j] != '\n' {
						j++
					}
					continue
				}
				if out[j] == '/' && j+1 < n && out[j+1] == '*' {
					j += 2
					for j+1 < n && !(out[j] == '*' && out[j+1] == '/') {
						j++
					}
					j += 2
					continue
				}
				break
			}
			if j < n && (out[j] == '}' || out[j] == ']') {
				out[i] = ' '
			}
			i++
		} else {
			i++
		}
	}
	return out, nil
}

// detectDuplicateKeys walks the JSON token stream and detects duplicate keys
// at each object level.
func detectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := walkMCPJSONValue(dec, tok, 0); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected data after top-level value")
		}
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func walkMCPJSONValue(dec *json.Decoder, tok json.Token, depth int) error {
	if depth > 32 {
		return errors.New("JSON nesting limit exceeded")
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			key, ok := keyTok.(string)
			if !ok {
				return errors.New("JSON object key must be a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate key %q", key)
			}
			seen[key] = struct{}{}
			valueTok, err := dec.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			if err := walkMCPJSONValue(dec, valueTok, depth+1); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return nil
	case '[':
		for dec.More() {
			valueTok, err := dec.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			if err := walkMCPJSONValue(dec, valueTok, depth+1); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return nil
	}
	return nil
}

func stringSliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
