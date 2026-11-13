package skillsync

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// MCP-B configuration planning. This file is strictly read-only: BuildMCPPlan
// never writes, never creates a directory, never executes a binary and never
// contacts the network. It produces the exact bytes a later digest-checked
// apply would write (mcp_transaction.go), plus the evidence needed to review
// that write before it happens.
const (
	mcpConfigDirRel  = ".config/opencode"
	mcpDefaultJSON   = "opencode.json"
	mcpDefaultJSONC  = "opencode.jsonc"
	mcpStateDirRel   = ".agentic-sdd/mcp"
	mcpOwnershipFile = "manifest.json"

	mcpFormatJSON  = "json"
	mcpFormatJSONC = "jsonc"

	mcpSchemaNone = "none"
	mcpSchemaV2   = "v2"

	mcpPlanVersion   = "mcp-plan-v1"
	mcpSnippetIndent = "  "

	mcpAuthOAuth   = "oauth"
	mcpAuthAPIKey  = "api-key"
	mcpAuthKeyless = "keyless"
)

// MCPPlanEntry operation tokens. They are the review vocabulary of a plan, not
// a general-purpose patch language.
const (
	mcpOpAdd      = "add"
	mcpOpChange   = "change"
	mcpOpLeave    = "leave"
	mcpOpDisabled = "disabled"
	mcpOpConflict = "conflict"
	mcpOpRestore  = "restore"
	mcpOpRemove   = "remove"
	mcpOpMissing  = "missing"
)

// mcpManagedNames returns the fixed canonical order of the servers
// Agentic-SDD manages. Selection and plan output follow this order so two
// previews of the same request are byte-identical.
func mcpManagedNames() []string {
	return []string{"codegraph", "context7", "exa"}
}

// mcpPlanTarget is the resolved configuration target for a read-only plan.
type mcpPlanTarget struct {
	Path    string
	Format  string
	Bytes   []byte
	Digest  string
	Mode    os.FileMode
	Missing bool
}

// BuildMCPPlan computes the structured, digest-bound MCP configuration plan for
// the selected servers. It is read-only: it reads the current target and the
// MCP ownership manifest, validates the V2 schema, and produces the exact bytes
// a separate apply would write.
//
// Failure classes are typed for the caller:
//   - ErrMCPUsage: empty home, unknown server name, unknown auth choice.
//   - ErrMCPConflict: ambiguous target, malformed/duplicate/invalid JSON,
//     unmanaged name collision, user-modified managed entry, literal
//     credential, unreadable or oversized target.
//   - ErrMCPUnsupported: legacy V1 schema (never migrated), unsupported target
//     extension, or a JSONC edit whose comment/format preservation cannot be
//     proven.
//   - ErrMCPUnsafeTarget: symlink, non-regular file, traversal path or an
//     explicit path outside the supported global OpenCode scope.
//
// A missing target is not an error: the plan reports a conflict and carries no
// target bytes, because v1 never creates a configuration file. A JSONC refusal
// returns the populated plan together with the typed error so a caller may
// still render the target, digest and manual snippet.
func BuildMCPPlan(home, configPath string, selected []string, auth map[string]string) (*MCPPlan, error) {
	if home == "" {
		return nil, fmt.Errorf("%w: home directory is required", ErrMCPUsage)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve home directory: %v", ErrMCPUsage, err)
	}

	names, authModes, err := mcpPlanSelection(selected, auth)
	if err != nil {
		return nil, err
	}

	target, err := mcpResolvePlanTarget(absHome, configPath)
	if err != nil {
		return nil, err
	}

	plan := &MCPPlan{
		TargetPath:      target.Path,
		Format:          target.Format,
		Schema:          mcpSchemaNone,
		BaseDigest:      target.Digest,
		Selected:        names,
		AuthChoices:     authModes,
		RuntimeVersion:  "not-checked",
		RuntimeEvidence: "not-checked",
		NotChecked: []string{
			"OpenCode runtime version (apply-time local probe only)",
			"remote connectivity",
			"authentication",
			"CodeGraph index freshness",
		},
	}

	if target.Missing {
		plan.Conflicts = append(plan.Conflicts,
			fmt.Sprintf("target missing; no creation in v1: %s", target.Path))
		plan.NotChecked = append(plan.NotChecked, "manual snippet in MCPManualSnippetForPlan")
		for _, name := range names {
			desiredEntry, err := mcpDesiredEntry(name, authModes[name])
			if err != nil {
				return nil, err
			}
			text, hash, err := mcpCanonicalEntry(desiredEntry)
			if err != nil {
				return nil, fmt.Errorf("%w: render manual snippet for %q: %v", ErrMCPConflict, name, err)
			}
			plan.Entries = append(plan.Entries, MCPPlanEntry{
				Name:    name,
				Op:      mcpOpAdd,
				Reason:  "not configured; manual snippet required (no creation in v1)",
				NewText: text,
				NewHash: hash,
			})
		}
		plan.PlanDigest = mcpPlanDigest(plan)
		return plan, nil
	}

	root, err := mcpParseDocument(target.Path, target.Format, target.Bytes)
	if err != nil {
		return nil, err
	}

	servers, hasMCP, hasServers, err := mcpServersOf(root)
	if err != nil {
		return nil, err
	}
	plan.Schema = mcpSchemaV2

	manifest, err := readMCPOwnershipManifest(absHome)
	if err != nil {
		return nil, err
	}
	installed := map[string]string{}
	if manifest != nil && manifest.Target == target.Path {
		for _, entry := range manifest.Entries {
			installed[entry.Name] = entry.InstalledDigest
		}
	}

	desired := make(map[string]map[string]any, len(names))
	for _, name := range names {
		entry, err := mcpDesiredEntry(name, authModes[name])
		if err != nil {
			return nil, err
		}
		desired[name] = entry
	}

	// A literal credential in a selected managed entry is refused before any
	// plan entry or diff text is rendered, and the value is never echoed.
	for _, name := range names {
		current, ok := servers[name].(map[string]any)
		if !ok {
			continue
		}
		if reason := mcpLiteralCredentialReason(name, current); reason != "" {
			return nil, fmt.Errorf("%w: %s", ErrMCPConflict, reason)
		}
	}

	for _, name := range names {
		entry := mcpPlanEntryFor(name, servers[name], desired[name], installed[name])
		plan.Entries = append(plan.Entries, entry)
		if entry.Op == mcpOpConflict {
			reason := entry.Reason
			if reason == "" {
				reason = fmt.Sprintf("conflict on managed server %q", name)
			}
			plan.Conflicts = append(plan.Conflicts, reason)
		}
	}

	if len(plan.Conflicts) == 0 {
		plan.NoOp = mcpEntriesAreNoOp(plan.Entries)
	}
	plan.PlanDigest = mcpPlanDigest(plan)

	if plan.NoOp || len(plan.Conflicts) > 0 {
		return plan, nil
	}

	// A write is planned. Refresh the document with exactly the managed
	// add/change entries; unrelated top-level keys, unrelated servers, unknown
	// fields and user-disabled servers are carried through untouched.
	newRoot := mcpCloneDocument(root)
	newServers := mcpEnsureServers(newRoot, hasMCP, hasServers)
	for _, entry := range plan.Entries {
		if entry.Op != mcpOpAdd && entry.Op != mcpOpChange {
			continue
		}
		newServers[entry.Name] = mcpDeepCopy(desired[entry.Name])
	}

	newBytes, err := mcpMarshalDocument(newRoot, target.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: render MCP configuration: %v", ErrMCPConflict, err)
	}

	if target.Format == mcpFormatJSONC {
		// AC-62: preserve comments and formatting losslessly, or refuse. This
		// build cannot prove ownership of every touched comment/delimiter, so
		// it refuses and prints a copyable snippet instead of reserializing.
		snippet := mcpManualSnippetFromEntries(plan.Entries)
		plan.Conflicts = append(plan.Conflicts, "JSONC edit refused; manual snippet required")
		plan.PlanDigest = mcpPlanDigest(plan)
		return plan, fmt.Errorf(
			"%w: cannot prove lossless comment/format preservation for %s; nothing was written. Merge this into mcp.servers manually:\n%s",
			ErrMCPUnsupported, target.Path, snippet)
	}

	plan.targetBytes = newBytes
	return plan, nil
}

// MCPManualSnippetForPlan returns the copyable configuration snippet for a plan
// that refused or could not write. It is empty when the plan writes normally.
func MCPManualSnippetForPlan(plan *MCPPlan) string {
	if plan == nil {
		return ""
	}
	return mcpManualSnippetFromEntries(plan.Entries)
}

// mcpPlanEntryFor classifies one selected server. The order of the checks is
// part of the contract: a disabled server is never re-enabled, an identical
// entry needs no write, and an entry may change only when the ownership
// manifest proves this tool installed exactly those bytes.
func mcpPlanEntryFor(name string, current any, desired map[string]any, installedDigest string) MCPPlanEntry {
	entry := MCPPlanEntry{Name: name}

	if current == nil {
		newText, newHash, err := mcpCanonicalEntry(desired)
		if err != nil {
			entry.Op = mcpOpConflict
			entry.Reason = fmt.Sprintf("cannot render managed recipe for %q: %v", name, err)
			return entry
		}
		entry.Op = mcpOpAdd
		entry.NewText = newText
		entry.NewHash = newHash
		if installedDigest != "" {
			entry.Reason = "not configured; previously managed entry is absent and will be re-added"
		} else {
			entry.Reason = "not configured"
		}
		return entry
	}

	currentMap, ok := current.(map[string]any)
	if !ok {
		entry.Op = mcpOpConflict
		entry.Reason = fmt.Sprintf("server entry %q is not a JSON object", name)
		return entry
	}
	oldText, oldHash, err := mcpCanonicalEntry(currentMap)
	if err != nil {
		entry.Op = mcpOpConflict
		entry.Reason = fmt.Sprintf("cannot canonicalize server entry %q: %v", name, err)
		return entry
	}
	entry.OldText = oldText
	entry.OldHash = oldHash

	if reason := mcpLiteralCredentialReason(name, currentMap); reason != "" {
		// The literal-credential check runs before any diff text exists, and
		// the matched value is never echoed.
		entry.Op = mcpOpConflict
		entry.OldText = ""
		entry.Reason = reason
		return entry
	}

	if mcpEntryDisabled(currentMap) {
		entry.Op = mcpOpDisabled
		entry.NewHash = oldHash
		entry.Reason = "server is disabled; the user-disabled state is preserved and never re-enabled"
		return entry
	}

	// A manifest entry whose current bytes no longer match the recorded
	// installed digest is a user edit, even if it happens to match the current
	// recipe semantically. Refuse before the leave/change classification.
	if installedDigest != "" && installedDigest != oldHash {
		entry.Op = mcpOpConflict
		entry.Reason = fmt.Sprintf(
			"server %q was modified after the last apply and no longer matches the recorded installed digest; refusing to overwrite", name)
		return entry
	}

	if mcpEntriesEquivalent(currentMap, desired) {
		entry.Op = mcpOpLeave
		entry.NewHash = oldHash
		entry.Reason = "already matches the managed recipe"
		return entry
	}

	newText, newHash, err := mcpCanonicalEntry(desired)
	if err != nil {
		entry.Op = mcpOpConflict
		entry.Reason = fmt.Sprintf("cannot render managed recipe for %q: %v", name, err)
		return entry
	}

	if installedDigest != "" && installedDigest == oldHash {
		entry.Op = mcpOpChange
		entry.NewText = newText
		entry.NewHash = newHash
		entry.Reason = "managed entry installed by Agentic-SDD; updating it to the current recipe"
		return entry
	}

	entry.Op = mcpOpConflict
	entry.Reason = fmt.Sprintf(
		"existing server %q differs from the managed recipe and is not owned by Agentic-SDD; refusing to replace (no replace flag in v1)", name)
	if installedDigest != "" {
		entry.Reason = fmt.Sprintf(
			"server %q was modified after the last apply and no longer matches the recorded installed digest; refusing to overwrite", name)
	}
	return entry
}

// mcpEntriesAreNoOp reports whether a plan changes nothing on disk.
func mcpEntriesAreNoOp(entries []MCPPlanEntry) bool {
	if len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		if entry.Op != mcpOpLeave && entry.Op != mcpOpDisabled {
			return false
		}
	}
	return true
}

// mcpPlanSelection validates and normalizes the selected names and auth modes.
func mcpPlanSelection(selected []string, auth map[string]string) ([]string, map[string]string, error) {
	canonical := mcpManagedNames()
	valid := make(map[string]bool, len(canonical))
	for _, name := range canonical {
		valid[name] = true
	}

	var names []string
	if len(selected) == 0 {
		names = append(names, canonical...)
	} else {
		wanted := make(map[string]bool, len(selected))
		for _, raw := range selected {
			name := strings.TrimSpace(raw)
			if !valid[name] {
				return nil, nil, fmt.Errorf(
					"%w: unknown server name %q (expected one of %s)",
					ErrMCPUsage, raw, strings.Join(canonical, ", "))
			}
			wanted[name] = true
		}
		for _, name := range canonical {
			if wanted[name] {
				names = append(names, name)
			}
		}
	}

	// Validate every supplied auth value, including one for a server that is
	// not selected: a mistyped value must fail before any diff is reviewed.
	keys := make([]string, 0, len(auth))
	for key := range auth {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !valid[key] {
			return nil, nil, fmt.Errorf("%w: unknown auth target %q (expected one of %s)",
				ErrMCPUsage, key, strings.Join(canonical, ", "))
		}
		if _, err := mcpAuthMode(key, auth[key]); err != nil {
			return nil, nil, err
		}
	}

	modes := make(map[string]string, len(names))
	for _, name := range names {
		mode, err := mcpAuthMode(name, auth[name])
		if err != nil {
			return nil, nil, err
		}
		if mode != "" {
			modes[name] = mode
		}
	}
	return names, modes, nil
}

// mcpAuthMode validates one authentication choice and returns its default.
func mcpAuthMode(name, value string) (string, error) {
	mode := strings.TrimSpace(strings.ToLower(value))
	switch name {
	case "codegraph":
		if mode != "" {
			return "", fmt.Errorf("%w: codegraph is a local server and has no authentication choice", ErrMCPUsage)
		}
		return "", nil
	case "context7":
		if mode == "" {
			return mcpAuthOAuth, nil
		}
		switch mode {
		case mcpAuthOAuth, mcpAuthAPIKey:
			return mode, nil
		}
		return "", fmt.Errorf("%w: unsupported context7 auth %q (expected oauth or api-key)", ErrMCPUsage, value)
	case "exa":
		if mode == "" {
			return mcpAuthKeyless, nil
		}
		switch mode {
		case mcpAuthKeyless, mcpAuthOAuth, mcpAuthAPIKey:
			return mode, nil
		}
		return "", fmt.Errorf("%w: unsupported exa auth %q (expected keyless, oauth or api-key)", ErrMCPUsage, value)
	}
	return "", fmt.Errorf("%w: unknown server %q", ErrMCPUsage, name)
}

// mcpDesiredEntry renders the canonical recipe for one managed server. Only
// documented OpenCode V2 fields are written, and credentials are always
// {env:NAME} placeholders.
func mcpDesiredEntry(name, authMode string) (map[string]any, error) {
	switch name {
	case "codegraph":
		return map[string]any{
			"type":     "local",
			"command":  []any{"codegraph", "serve", "--mcp"},
			"disabled": false,
			"codemode": false,
		}, nil
	case "context7":
		switch authMode {
		case mcpAuthOAuth, "":
			return map[string]any{
				"type": "remote",
				"url":  "https://mcp.context7.com/mcp/oauth",
			}, nil
		case mcpAuthAPIKey:
			return map[string]any{
				"type":  "remote",
				"url":   "https://mcp.context7.com/mcp",
				"oauth": false,
				"headers": map[string]any{
					"Authorization": "Bearer {env:CONTEXT7_API_KEY}",
				},
			}, nil
		}
		return nil, fmt.Errorf("%w: unsupported context7 auth %q", ErrMCPUsage, authMode)
	case "exa":
		switch authMode {
		case mcpAuthKeyless, "":
			return map[string]any{
				"type": "remote",
				"url":  "https://mcp.exa.ai/mcp",
			}, nil
		case mcpAuthOAuth:
			return map[string]any{
				"type": "remote",
				"url":  "https://mcp.exa.ai/mcp?login",
			}, nil
		case mcpAuthAPIKey:
			return map[string]any{
				"type": "remote",
				"url":  "https://mcp.exa.ai/mcp",
				"headers": map[string]any{
					"x-api-key": "{env:EXA_API_KEY}",
				},
			}, nil
		}
		return nil, fmt.Errorf("%w: unsupported exa auth %q", ErrMCPUsage, authMode)
	}
	return nil, fmt.Errorf("%w: unknown server %q", ErrMCPUsage, name)
}

// mcpResolvePlanTarget resolves the configuration target without writing.
func mcpResolvePlanTarget(home, configPath string) (mcpPlanTarget, error) {
	if configPath != "" {
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return mcpPlanTarget{}, fmt.Errorf("%w: resolve config path: %v", ErrMCPUsage, err)
		}
		format := mcpFormatForPath(absPath)
		if format == "" {
			return mcpPlanTarget{}, fmt.Errorf(
				"%w: unsupported configuration extension %q; expected .json or .jsonc",
				ErrMCPUnsupported, filepath.Ext(absPath))
		}
		scopeDir := filepath.Join(home, mcpConfigDirRel)
		if !mcpPathWithin(scopeDir, absPath) {
			return mcpPlanTarget{}, fmt.Errorf(
				"%w: %s is outside the supported global scope %s; project-local MCP scope is deferred in v1",
				ErrMCPUnsafeTarget, absPath, scopeDir)
		}
		if err := mcpCheckTargetSafety(configPath, absPath); err != nil {
			return mcpPlanTarget{}, err
		}
		info, err := os.Lstat(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return mcpPlanTarget{Path: absPath, Format: format, Missing: true}, nil
			}
			return mcpPlanTarget{}, fmt.Errorf("%w: inspect %s: %v", ErrMCPConflict, absPath, err)
		}
		if !info.Mode().IsRegular() {
			return mcpPlanTarget{}, fmt.Errorf("%w: target is not a regular file: %s", ErrMCPUnsafeTarget, absPath)
		}
		data, mode, err := mcpReadConfigBytes(absPath)
		if err != nil {
			return mcpPlanTarget{}, err
		}
		return mcpPlanTarget{
			Path:   absPath,
			Format: format,
			Bytes:  data,
			Digest: mcpBytesDigest(data),
			Mode:   mode,
		}, nil
	}

	jsonPath := filepath.Join(home, mcpConfigDirRel, mcpDefaultJSON)
	jsoncPath := filepath.Join(home, mcpConfigDirRel, mcpDefaultJSONC)
	jsonInfo, jsonErr := os.Lstat(jsonPath)
	jsoncInfo, jsoncErr := os.Lstat(jsoncPath)
	hasJSON := jsonErr == nil
	hasJSONC := jsoncErr == nil
	if jsonErr != nil && !errors.Is(jsonErr, os.ErrNotExist) {
		return mcpPlanTarget{}, fmt.Errorf("%w: inspect %s: %v", ErrMCPConflict, jsonPath, jsonErr)
	}
	if jsoncErr != nil && !errors.Is(jsoncErr, os.ErrNotExist) {
		return mcpPlanTarget{}, fmt.Errorf("%w: inspect %s: %v", ErrMCPConflict, jsoncPath, jsoncErr)
	}

	if hasJSON && hasJSONC {
		return mcpPlanTarget{}, fmt.Errorf(
			"%w: both %s and %s exist; refusing to guess which target to manage",
			ErrMCPConflict, jsonPath, jsoncPath)
	}
	if !hasJSON && !hasJSONC {
		return mcpPlanTarget{Path: jsonPath, Format: mcpFormatJSON, Missing: true}, nil
	}

	path := jsonPath
	format := mcpFormatJSON
	if hasJSONC {
		path = jsoncPath
		format = mcpFormatJSONC
	}
	if err := mcpCheckTargetSafety("", path); err != nil {
		return mcpPlanTarget{}, err
	}
	info := jsonInfo
	if hasJSONC {
		info = jsoncInfo
	}
	if !info.Mode().IsRegular() {
		return mcpPlanTarget{}, fmt.Errorf("%w: target is not a regular file: %s", ErrMCPUnsafeTarget, path)
	}
	data, mode, err := mcpReadConfigBytes(path)
	if err != nil {
		return mcpPlanTarget{}, err
	}
	return mcpPlanTarget{
		Path:   path,
		Format: format,
		Bytes:  data,
		Digest: mcpBytesDigest(data),
		Mode:   mode,
	}, nil
}

// mcpFormatForPath maps a target extension to the supported format token.
func mcpFormatForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return mcpFormatJSON
	case ".jsonc":
		return mcpFormatJSONC
	}
	return ""
}

// mcpPathWithin reports whether path is dir or inside it, after cleaning.
func mcpPathWithin(dir, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(path))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// mcpCheckTargetSafety rejects a traversal path, a symlinked target and any
// symlinked or non-directory parent component. It never follows the link it
// refuses.
func mcpCheckTargetSafety(rawInput, absPath string) error {
	if rawInput != "" {
		for _, part := range strings.Split(filepath.ToSlash(rawInput), "/") {
			if part == ".." {
				return fmt.Errorf("%w: configuration path contains a parent traversal component: %s",
					ErrMCPUnsafeTarget, rawInput)
			}
		}
	}
	if absPath == "" || filepath.Base(absPath) == string(filepath.Separator) {
		return fmt.Errorf("%w: empty or root configuration path", ErrMCPUnsafeTarget)
	}

	dir := filepath.Dir(absPath)
	var ancestors []string
	for d := dir; ; d = filepath.Dir(d) {
		ancestors = append(ancestors, d)
		if parent := filepath.Dir(d); parent == d {
			break
		}
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		info, err := os.Lstat(ancestors[i])
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("%w: inspect %s: %v", ErrMCPConflict, ancestors[i], err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: configuration parent is a symlink: %s", ErrMCPUnsafeTarget, ancestors[i])
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: configuration parent is not a directory: %s", ErrMCPUnsafeTarget, ancestors[i])
		}
	}

	info, err := os.Lstat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%w: inspect %s: %v", ErrMCPConflict, absPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: configuration target is a symlink: %s", ErrMCPUnsafeTarget, absPath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: configuration target is not a regular file: %s", ErrMCPUnsafeTarget, absPath)
	}
	return nil
}

// mcpReadConfigBytes reads a bounded, validated configuration file.
func mcpReadConfigBytes(path string) ([]byte, os.FileMode, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: inspect %s: %v", ErrMCPConflict, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("%w: target is not a regular file: %s", ErrMCPUnsafeTarget, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: open %s: %v", ErrMCPConflict, path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxMCPConfigFileBytes+1))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: read %s: %v", ErrMCPConflict, path, err)
	}
	if len(data) > maxMCPConfigFileBytes {
		return nil, 0, fmt.Errorf("%w: %s exceeds the %d byte MCP configuration limit",
			ErrMCPConflict, path, maxMCPConfigFileBytes)
	}
	if !utf8.Valid(data) {
		return nil, 0, fmt.Errorf("%w: %s is not valid UTF-8", ErrMCPConflict, path)
	}
	return data, info.Mode().Perm(), nil
}

// mcpParseDocument parses one target file into a generic document with strict
// duplicate detection and json.Number preservation.
func mcpParseDocument(path, format string, data []byte) (map[string]any, error) {
	jsonBytes := data
	if format == mcpFormatJSONC {
		stripped, err := stripJSONC(data)
		if err != nil {
			return nil, fmt.Errorf("%w: malformed JSONC in %s: %v", ErrMCPConflict, path, err)
		}
		jsonBytes = stripped
	}
	if err := detectDuplicateKeys(jsonBytes); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrMCPConflict, path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(jsonBytes))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON in %s: %v", ErrMCPConflict, path, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: %s has trailing data after the top-level value", ErrMCPConflict, path)
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s top-level value is not a JSON object", ErrMCPConflict, path)
	}
	return rootMap, nil
}

// mcpServersOf extracts mcp.servers and detects the schema honestly. V1
// mcp.<name>/enabled documents are refused, never migrated.
func mcpServersOf(root map[string]any) (map[string]any, bool, bool, error) {
	mcpValue, hasMCP := root["mcp"]
	if !hasMCP {
		return nil, false, false, nil
	}
	mcpMap, ok := mcpValue.(map[string]any)
	if !ok {
		return nil, true, false, fmt.Errorf("%w: mcp field is not a JSON object", ErrMCPConflict)
	}
	serversValue, hasServers := mcpMap["servers"]
	if !hasServers {
		if len(mcpMap) > 0 {
			return nil, true, false, fmt.Errorf(
				"%w: legacy V1 mcp.<name> schema detected; OpenCode V2 requires mcp.servers and this build never migrates automatically",
				ErrMCPUnsupported)
		}
		return nil, true, false, nil
	}
	serversMap, ok := serversValue.(map[string]any)
	if !ok {
		return nil, true, false, fmt.Errorf("%w: mcp.servers is not a JSON object", ErrMCPConflict)
	}
	for name, entry := range serversMap {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			return nil, true, true, fmt.Errorf("%w: mcp.servers entry %q is not a JSON object", ErrMCPConflict, name)
		}
		if mcpManagedName(name) {
			if err := mcpValidateManagedEntry(name, entryMap); err != nil {
				return nil, true, true, err
			}
		}
	}
	return serversMap, true, true, nil
}

// mcpManagedName reports whether name is one of the managed default servers.
func mcpManagedName(name string) bool {
	for _, managed := range mcpManagedNames() {
		if managed == name {
			return true
		}
	}
	return false
}

// mcpValidateManagedEntry rejects a managed entry whose documented shape cannot
// be diffed safely. Unrelated entries keep their unknown fields untouched.
func mcpValidateManagedEntry(name string, entry map[string]any) error {
	typeValue, hasType := entry["type"]
	if hasType {
		typeString, ok := typeValue.(string)
		if !ok {
			return fmt.Errorf("%w: mcp.servers.%s.type must be a string", ErrMCPConflict, name)
		}
		switch typeString {
		case "local":
			if !mcpIsStringArray(entry["command"]) {
				return fmt.Errorf("%w: mcp.servers.%s.command must be an array of strings", ErrMCPConflict, name)
			}
		case "remote":
			if !mcpIsNonEmptyString(entry["url"]) {
				return fmt.Errorf("%w: mcp.servers.%s.url must be a non-empty string", ErrMCPConflict, name)
			}
		default:
			return fmt.Errorf("%w: mcp.servers.%s.type %q is not supported", ErrMCPConflict, name, typeString)
		}
	}
	if disabled, ok := entry["disabled"]; ok {
		if _, isBool := disabled.(bool); !isBool {
			return fmt.Errorf("%w: mcp.servers.%s.disabled must be a boolean", ErrMCPConflict, name)
		}
	}
	if headers, ok := entry["headers"]; ok {
		headerMap, isMap := headers.(map[string]any)
		if !isMap {
			return fmt.Errorf("%w: mcp.servers.%s.headers must be an object", ErrMCPConflict, name)
		}
		for key, value := range headerMap {
			if _, isString := value.(string); !isString {
				return fmt.Errorf("%w: mcp.servers.%s.headers.%s must be a string", ErrMCPConflict, name, key)
			}
		}
	}
	return nil
}

func mcpIsStringArray(value any) bool {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	for _, item := range items {
		if _, ok := item.(string); !ok {
			return false
		}
	}
	return true
}

func mcpIsNonEmptyString(value any) bool {
	s, ok := value.(string)
	return ok && strings.TrimSpace(s) != ""
}

// mcpEntryDisabled reports an explicit disabled state. It never re-enables.
func mcpEntryDisabled(entry map[string]any) bool {
	disabled, ok := entry["disabled"].(bool)
	return ok && disabled
}

// mcpNormalizeEntry projects an entry onto the fields Agentic-SDD manages,
// applying documented defaults, so an omitted "disabled": false compares equal
// to an explicit one.
func mcpNormalizeEntry(entry map[string]any) map[string]any {
	out := make(map[string]any, 8)
	if value, ok := entry["type"]; ok {
		out["type"] = value
	}
	if value, ok := entry["command"]; ok {
		out["command"] = value
	}
	if value, ok := entry["url"]; ok {
		out["url"] = value
	}
	if value, ok := entry["headers"]; ok {
		out["headers"] = value
	}
	if value, ok := entry["oauth"]; ok {
		out["oauth"] = value
	}
	disabled, ok := entry["disabled"].(bool)
	if !ok {
		disabled = false
	}
	out["disabled"] = disabled
	codemode, ok := entry["codemode"].(bool)
	if !ok {
		codemode = false
	}
	out["codemode"] = codemode
	return out
}

// mcpEntriesEquivalent compares only managed fields, so an unrelated unknown
// field in an existing entry never forces a write.
func mcpEntriesEquivalent(current, desired map[string]any) bool {
	return mcpDeepEqual(mcpNormalizeEntry(current), mcpNormalizeEntry(desired))
}

func mcpDeepEqual(a, b any) bool {
	left, errLeft := json.Marshal(a)
	right, errRight := json.Marshal(b)
	if errLeft != nil || errRight != nil {
		return false
	}
	return bytes.Equal(left, right)
}

// mcpLiteralCredentialReason reports a managed entry that carries a literal
// credential instead of an {env:NAME} placeholder. The matched value is never
// included in the reason.
func mcpLiteralCredentialReason(name string, entry map[string]any) string {
	for _, section := range []string{"headers", "environment"} {
		values, ok := entry[section].(map[string]any)
		if !ok {
			continue
		}
		for key, raw := range values {
			value, ok := raw.(string)
			if !ok || strings.TrimSpace(value) == "" {
				continue
			}
			if strings.Contains(value, "{env:") {
				continue
			}
			if mcpCredentialKey(key) || mcpCredentialValue(value) {
				return fmt.Sprintf(
					"managed entry %q contains a literal credential in %s.%s; move it to an {env:NAME} placeholder before applying",
					name, section, key)
			}
		}
	}
	return ""
}

func mcpCredentialKey(key string) bool {
	lower := strings.ToLower(key)
	for _, marker := range []string{"authorization", "api-key", "apikey", "token", "secret", "password", "credential"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func mcpCredentialValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		return true
	}
	for _, prefix := range []string{"sk-", "sk_", "ghp_", "gho_", "xoxb-", "aiza"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// mcpCanonicalEntry renders one server entry as compact, key-sorted JSON and
// returns its sha256 digest. That canonical text is the entry-level identity a
// backup, manifest and restore comparison all use.
func mcpCanonicalEntry(entry map[string]any) (string, string, error) {
	data, err := json.Marshal(entry)
	if err != nil {
		return "", "", err
	}
	return string(data), mcpBytesDigest(data), nil
}

// mcpBytesDigest returns the lowercase hex sha256 of data.
func mcpBytesDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// mcpPlanDigest hashes the canonical plan projection: everything that decides
// what apply would write, and nothing the caller merely displays.
func mcpPlanDigest(plan *MCPPlan) string {
	if plan == nil {
		return ""
	}
	type projectionEntry struct {
		Name    string `json:"name"`
		Op      string `json:"op"`
		OldHash string `json:"old_hash,omitempty"`
		NewHash string `json:"new_hash,omitempty"`
	}
	type projection struct {
		Version    string            `json:"version"`
		Target     string            `json:"target"`
		Format     string            `json:"format"`
		Schema     string            `json:"schema"`
		BaseDigest string            `json:"base_digest"`
		Selected   []string          `json:"selected"`
		Auth       map[string]string `json:"auth,omitempty"`
		Entries    []projectionEntry `json:"entries"`
		Conflicts  []string          `json:"conflicts,omitempty"`
		NoOp       bool              `json:"no_op"`
	}
	proj := projection{
		Version:    mcpPlanVersion,
		Target:     plan.TargetPath,
		Format:     plan.Format,
		Schema:     plan.Schema,
		BaseDigest: plan.BaseDigest,
		Selected:   plan.Selected,
		Auth:       plan.AuthChoices,
		Conflicts:  plan.Conflicts,
		NoOp:       plan.NoOp,
	}
	for _, entry := range plan.Entries {
		proj.Entries = append(proj.Entries, projectionEntry{
			Name:    entry.Name,
			Op:      entry.Op,
			OldHash: entry.OldHash,
			NewHash: entry.NewHash,
		})
	}
	if proj.Entries == nil {
		proj.Entries = []projectionEntry{}
	}
	data, err := json.Marshal(proj)
	if err != nil {
		return ""
	}
	return mcpBytesDigest(data)
}

// mcpCloneDocument deep-copies a parsed document so planning never mutates the
// structure it compared against.
func mcpCloneDocument(root map[string]any) map[string]any {
	return mcpDeepCopy(root).(map[string]any)
}

func mcpDeepCopy(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, inner := range typed {
			out[key] = mcpDeepCopy(inner)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, inner := range typed {
			out[i] = mcpDeepCopy(inner)
		}
		return out
	default:
		return value
	}
}

// mcpEnsureServers returns the servers object of a cloned document, creating
// only the containers the document is missing.
func mcpEnsureServers(root map[string]any, hasMCP, hasServers bool) map[string]any {
	if !hasMCP {
		mcpMap := map[string]any{}
		root["mcp"] = mcpMap
		servers := map[string]any{}
		mcpMap["servers"] = servers
		return servers
	}
	mcpMap, _ := root["mcp"].(map[string]any)
	if !hasServers {
		servers := map[string]any{}
		mcpMap["servers"] = servers
		return servers
	}
	servers, _ := mcpMap["servers"].(map[string]any)
	return servers
}

// mcpDetectIndent returns the indentation unit of an existing document so a
// rewrite keeps its visual shape. It defaults to two spaces.
func mcpDetectIndent(data []byte) string {
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := bytes.TrimLeft(line, " \t")
		if len(trimmed) == 0 || len(trimmed) == len(line) {
			continue
		}
		unit := string(line[:len(line)-len(trimmed)])
		if len(unit) > 8 {
			continue
		}
		if strings.Contains(unit, " ") && strings.Contains(unit, "\t") {
			continue
		}
		return unit
	}
	return mcpSnippetIndent
}

// mcpMarshalDocument renders a document deterministically, keeping the
// original indentation unit and trailing-newline convention.
func mcpMarshalDocument(root map[string]any, original []byte) ([]byte, error) {
	indent := mcpDetectIndent(original)
	data, err := json.MarshalIndent(root, "", indent)
	if err != nil {
		return nil, err
	}
	if len(original) == 0 || bytes.HasSuffix(original, []byte("\n")) {
		data = append(data, '\n')
	}
	return data, nil
}

// mcpManualSnippetFromEntries renders the exact mcp.servers text a user can
// paste for a plan that refused to write.
func mcpManualSnippetFromEntries(entries []MCPPlanEntry) string {
	servers := map[string]any{}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Op != mcpOpAdd && entry.Op != mcpOpChange {
			continue
		}
		if entry.NewText == "" {
			continue
		}
		var value any
		dec := json.NewDecoder(strings.NewReader(entry.NewText))
		dec.UseNumber()
		if err := dec.Decode(&value); err != nil {
			continue
		}
		servers[entry.Name] = value
		names = append(names, entry.Name)
	}
	if len(names) == 0 {
		for _, entry := range entries {
			names = append(names, entry.Name)
		}
	}
	sort.Strings(names)
	doc := map[string]any{"mcp": map[string]any{"servers": servers}}
	data, err := json.MarshalIndent(doc, "", mcpSnippetIndent)
	if err != nil {
		return ""
	}
	return string(data)
}
