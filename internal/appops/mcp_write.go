package appops

// Feature 015 MCP-B — typed MCP write operation seam.
//
// This file extends the Feature 015 MCP seam with transactional, digest-bound
// write operations: plan preview with preview tokens, digest-revalidating
// apply, and entry-level restore preview and apply. It mirrors the Feature
// 013/014/015 request/result pattern: immutable, validated, absolute-path
// requests; exported read-only accessors; read operations that never write; and
// write operations that revalidate a preview token and configuration digest in
// the same call that mutates.
//
// It is stateless by construction: no draft, plan or token is retained between
// calls, and no operation performs network, model, package-manager or OAuth
// operations. The engine (internal/skillsync) remains the only owner of
// validation, digests, JSON/JSONC parsing, file locks, backups and rollback.
// appops does not duplicate engine rules; it verifies token and digest
// integrity before delegating to the engine.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"agentic-sdd/internal/skillsync"
	"agentic-sdd/internal/version"
)

// Preview token versions for MCP plan and restore previews.
const (
	mcpPlanTokenVersion    = "appops-mcp-plan-preview-v1"
	mcpRestoreTokenVersion = "appops-mcp-restore-preview-v1"
)

// MCPPlanRequest is one immutable, validated request to preview an MCP
// configuration edit plan. Construct it with NewMCPPlanRequest and read it
// through its accessors.
type MCPPlanRequest struct {
	home        string
	configPath  string
	selected    []string
	authChoices map[string]string
}

// NewMCPPlanRequest validates and absolutizes one MCP plan preview request.
// home is required; configPath may be empty for the documented default target.
// Server selections and authentication choices are copied at construction. No
// files are read or written here.
func NewMCPPlanRequest(home, configPath string, selected []string, auth map[string]string) (MCPPlanRequest, error) {
	if home == "" {
		return MCPPlanRequest{}, errors.New("mcp plan request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPPlanRequest{}, err
	}
	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return MCPPlanRequest{}, err
		}
	}
	var authCopy map[string]string
	if auth != nil {
		authCopy = make(map[string]string, len(auth))
		for k, v := range auth {
			authCopy[k] = v
		}
	}
	return MCPPlanRequest{
		home:        absHome,
		configPath:  absConfig,
		selected:    mcpCopySelection(selected),
		authChoices: authCopy,
	}, nil
}

// Home returns the absolute home directory the plan is resolved against.
func (r MCPPlanRequest) Home() string { return r.home }

// ConfigPath returns the absolute explicit configuration path, or "" for the
// engine's documented default target.
func (r MCPPlanRequest) ConfigPath() string { return r.configPath }

// Selected returns a copy of the requested server names. Changing the returned
// slice cannot change the request.
func (r MCPPlanRequest) Selected() []string { return mcpCopySelection(r.selected) }

// AuthChoices returns a copy of the explicit authentication mode choices.
func (r MCPPlanRequest) AuthChoices() map[string]string {
	if r.authChoices == nil {
		return nil
	}
	out := make(map[string]string, len(r.authChoices))
	for k, v := range r.authChoices {
		out[k] = v
	}
	return out
}

// MCPPlanResult is the outcome of an MCP plan preview. Plan carries the exact
// target bytes, digests and entry projections; Token authorizes exactly this
// plan and target state for a subsequent apply.
type MCPPlanResult struct {
	Plan  *skillsync.MCPPlan
	Token string
}

// MCPPlanPreview runs skillsync.BuildMCPPlan. It reads the current configuration
// and builds a digest-bound, exact-byte plan. It modifies no files, creates no
// backup, executes no binaries and contacts no remote endpoints.
func MCPPlanPreview(req MCPPlanRequest) (MCPPlanResult, error) {
	plan, err := skillsync.BuildMCPPlan(req.home, req.configPath, req.selected, req.authChoices)
	if err != nil {
		return MCPPlanResult{}, err
	}
	if plan == nil {
		return MCPPlanResult{}, errors.New("mcp plan was nil")
	}
	return MCPPlanResult{
		Plan:  plan,
		Token: mcpPlanPreviewToken(plan),
	}, nil
}

// mcpPlanPreviewToken computes a canonical SHA-256 token over the plan's
// target path, format, schema, digests, selected servers, authentication
// choices, tool version and entries. It contains no secret credentials.
func mcpPlanPreviewToken(plan *skillsync.MCPPlan) string {
	if plan == nil {
		return ""
	}
	enc := newTokenEncoding()
	enc.field(mcpPlanTokenVersion)
	enc.field(version.Version)
	enc.field(plan.TargetPath)
	enc.field(plan.Format)
	enc.field(plan.Schema)
	enc.field(plan.BaseDigest)
	enc.field(plan.PlanDigest)
	enc.field(plan.RuntimeVersion)
	enc.field(plan.RuntimeEvidence)
	enc.boolean(plan.NoOp)
	enc.field(strconv.Itoa(len(plan.Selected)))
	for _, s := range plan.Selected {
		enc.field(s)
	}
	if len(plan.AuthChoices) == 0 {
		enc.field("0")
	} else {
		keys := make([]string, 0, len(plan.AuthChoices))
		for k := range plan.AuthChoices {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		enc.field(strconv.Itoa(len(keys)))
		for _, k := range keys {
			enc.field(k)
			enc.field(plan.AuthChoices[k])
		}
	}
	enc.field(strconv.Itoa(len(plan.Entries)))
	for _, e := range plan.Entries {
		enc.field(e.Name)
		enc.field(e.Op)
		enc.field(e.Reason)
		enc.field(e.OldHash)
		enc.field(e.NewHash)
	}
	return enc.sum()
}

// MCPApplyRequest is one immutable, validated request to apply a previously
// previewed MCP plan. Construct it with NewMCPApplyRequest and read it through
// its accessors.
type MCPApplyRequest struct {
	home         string
	configPath   string
	plan         *skillsync.MCPPlan
	token        string
	expectDigest string
	openCodeBin  string
}

// NewMCPApplyRequest validates and absolutizes one MCP apply request.
// home, plan, token and expectDigest are required. The constructor stores a
// deep copy of the plan so caller mutation cannot affect the request.
func NewMCPApplyRequest(home, configPath string, plan *skillsync.MCPPlan, token, expectDigest, openCodeBin string) (MCPApplyRequest, error) {
	if home == "" {
		return MCPApplyRequest{}, errors.New("mcp apply request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPApplyRequest{}, err
	}
	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return MCPApplyRequest{}, err
		}
	}
	if plan == nil {
		return MCPApplyRequest{}, errors.New("mcp apply request plan is required")
	}
	if token == "" {
		return MCPApplyRequest{}, errors.New("mcp apply request token is required")
	}
	if expectDigest == "" {
		return MCPApplyRequest{}, errors.New("mcp apply request expect digest is required")
	}
	if plan.BaseDigest != "" && expectDigest != plan.BaseDigest {
		return MCPApplyRequest{}, errors.New("mcp apply request expect digest does not match plan base digest")
	}
	absOpenCode := ""
	if openCodeBin != "" {
		absOpenCode, err = filepath.Abs(openCodeBin)
		if err != nil {
			return MCPApplyRequest{}, err
		}
	}
	return MCPApplyRequest{
		home:         absHome,
		configPath:   absConfig,
		plan:         plan.Clone(),
		token:        token,
		expectDigest: expectDigest,
		openCodeBin:  absOpenCode,
	}, nil
}

// Home returns the absolute home directory the apply is resolved against.
func (r MCPApplyRequest) Home() string { return r.home }

// ConfigPath returns the absolute explicit configuration path, or "".
func (r MCPApplyRequest) ConfigPath() string { return r.configPath }

// Plan returns a deep copy of the request's plan.
func (r MCPApplyRequest) Plan() *skillsync.MCPPlan { return r.plan.Clone() }

// Token returns the preview token authorizing this apply.
func (r MCPApplyRequest) Token() string { return r.token }

// ExpectDigest returns the expected base digest of the target configuration.
func (r MCPApplyRequest) ExpectDigest() string { return r.expectDigest }

// OpenCodeBin returns the explicit OpenCode binary path, or "".
func (r MCPApplyRequest) OpenCodeBin() string { return r.openCodeBin }

// MCPApplyResult is the structured outcome of an MCP apply or restore.
type MCPApplyResult struct {
	Result *skillsync.MCPApplyResult
}

// MCPApply revalidates the preview token and expect digest in-call and then
// applies the plan via skillsync.ApplyMCPPlan. An empty or mismatched token,
// or an empty or mismatched digest, returns ErrNeedsPreview with no write.
func MCPApply(req MCPApplyRequest) (MCPApplyResult, error) {
	if req.home == "" || req.plan == nil {
		return MCPApplyResult{}, errors.New("invalid mcp apply request")
	}
	if req.token == "" {
		return MCPApplyResult{}, ErrNeedsPreview
	}
	if req.token != mcpPlanPreviewToken(req.plan) {
		return MCPApplyResult{}, ErrNeedsPreview
	}
	if req.expectDigest == "" {
		return MCPApplyResult{}, ErrNeedsPreview
	}
	if req.plan.BaseDigest != "" && req.expectDigest != req.plan.BaseDigest {
		return MCPApplyResult{}, ErrNeedsPreview
	}
	opts := skillsync.MCPApplyOptions{
		Home:         req.home,
		ExpectDigest: req.expectDigest,
		OpenCodeBin:  req.openCodeBin,
	}
	res, err := skillsync.ApplyMCPPlan(req.home, req.configPath, req.plan.Clone(), opts)
	if err != nil {
		return MCPApplyResult{}, err
	}
	return MCPApplyResult{Result: &res}, nil
}

// MCPRestorePreviewRequest is one immutable, validated request to preview an
// MCP configuration restore.
type MCPRestorePreviewRequest struct {
	home       string
	configPath string
	backupID   string
}

// NewMCPRestorePreviewRequest validates and absolutizes one MCP restore preview
// request. home and backupID are required; backupID is syntax-checked.
func NewMCPRestorePreviewRequest(home, configPath, backupID string) (MCPRestorePreviewRequest, error) {
	if home == "" {
		return MCPRestorePreviewRequest{}, errors.New("mcp restore preview request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPRestorePreviewRequest{}, err
	}
	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return MCPRestorePreviewRequest{}, err
		}
	}
	if backupID == "" {
		return MCPRestorePreviewRequest{}, errors.New("mcp restore preview request backup id is required")
	}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		return MCPRestorePreviewRequest{}, err
	}
	return MCPRestorePreviewRequest{
		home:       absHome,
		configPath: absConfig,
		backupID:   backupID,
	}, nil
}

// Home returns the absolute home directory the preview is resolved against.
func (r MCPRestorePreviewRequest) Home() string { return r.home }

// ConfigPath returns the absolute explicit configuration path, or "".
func (r MCPRestorePreviewRequest) ConfigPath() string { return r.configPath }

// BackupID returns the validated backup directory identifier.
func (r MCPRestorePreviewRequest) BackupID() string { return r.backupID }

// MCPBackupInfo is one managed MCP backup entry listed by MCPRestoreList.
type MCPBackupInfo struct {
	ID        string
	Path      string
	CreatedAt string
	Servers   []string
}

// MCPRestoreListRequest is one immutable, validated request to list managed MCP
// backups.
type MCPRestoreListRequest struct {
	home string
}

// NewMCPRestoreListRequest validates and absolutizes one MCP restore list request.
func NewMCPRestoreListRequest(home string) (MCPRestoreListRequest, error) {
	if home == "" {
		return MCPRestoreListRequest{}, errors.New("mcp restore list request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPRestoreListRequest{}, err
	}
	return MCPRestoreListRequest{home: absHome}, nil
}

// Home returns the absolute home directory the listing is resolved against.
func (r MCPRestoreListRequest) Home() string { return r.home }

// MCPRestoreList lists managed MCP backups found under
// <home>/.agentic-sdd/mcp/backups. Newest backups appear first.
func MCPRestoreList(req MCPRestoreListRequest) ([]MCPBackupInfo, error) {
	if req.home == "" {
		return nil, errors.New("mcp restore list request home is required")
	}
	backupDir := filepath.Join(req.home, ".agentic-sdd", "mcp", "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []MCPBackupInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if err := skillsync.ValidateBackupID(id); err != nil {
			continue
		}
		out = append(out, MCPBackupInfo{
			ID:        id,
			Path:      filepath.Join(backupDir, id),
			CreatedAt: id,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID > out[j].ID
	})
	return out, nil
}

// MCPRestorePreviewResult is the read-only restore comparison. Preview carries
// only managed entries; Token covers the backup ID, target, current digest and entries.
type MCPRestorePreviewResult struct {
	Preview *skillsync.MCPRestorePreview
	Token   string
}

// MCPRestorePreview runs skillsync.PreviewMCPRestore. It reads the managed backup
// and current target and reports entry-level restore actions. It writes nothing.
func MCPRestorePreview(req MCPRestorePreviewRequest) (MCPRestorePreviewResult, error) {
	preview, err := skillsync.PreviewMCPRestore(req.home, req.backupID, req.configPath)
	if err != nil {
		return MCPRestorePreviewResult{}, err
	}
	if preview == nil {
		return MCPRestorePreviewResult{}, errors.New("mcp restore preview was nil")
	}
	return MCPRestorePreviewResult{
		Preview: preview,
		Token:   mcpRestorePreviewToken(preview),
	}, nil
}

// mcpRestorePreviewToken computes a canonical SHA-256 token over the backup ID,
// target path, base digest and entries.
func mcpRestorePreviewToken(preview *skillsync.MCPRestorePreview) string {
	if preview == nil {
		return ""
	}
	enc := newTokenEncoding()
	enc.field(mcpRestoreTokenVersion)
	enc.field(version.Version)
	enc.field(preview.BackupID)
	enc.field(preview.TargetPath)
	enc.field(preview.BaseDigest)
	enc.field(strconv.Itoa(len(preview.Entries)))
	for _, e := range preview.Entries {
		enc.field(e.Name)
		enc.field(e.Op)
		enc.field(e.Reason)
		enc.field(e.OldHash)
		enc.field(e.NewHash)
	}
	enc.field(strconv.Itoa(len(preview.Conflicts)))
	for _, c := range preview.Conflicts {
		enc.field(c)
	}
	return enc.sum()
}

// MCPRestoreRequest is one immutable, validated request to restore managed MCP
// configuration entries from a backup.
type MCPRestoreRequest struct {
	home         string
	configPath   string
	backupID     string
	token        string
	expectDigest string
}

// NewMCPRestoreRequest validates and absolutizes one MCP restore request.
// home, backupID, token and expectDigest are required.
func NewMCPRestoreRequest(home, configPath, backupID, token, expectDigest string) (MCPRestoreRequest, error) {
	if home == "" {
		return MCPRestoreRequest{}, errors.New("mcp restore request home is required")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return MCPRestoreRequest{}, err
	}
	absConfig := ""
	if configPath != "" {
		absConfig, err = filepath.Abs(configPath)
		if err != nil {
			return MCPRestoreRequest{}, err
		}
	}
	if backupID == "" {
		return MCPRestoreRequest{}, errors.New("mcp restore request backup id is required")
	}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		return MCPRestoreRequest{}, err
	}
	if token == "" {
		return MCPRestoreRequest{}, errors.New("mcp restore request token is required")
	}
	if expectDigest == "" {
		return MCPRestoreRequest{}, errors.New("mcp restore request expect digest is required")
	}
	return MCPRestoreRequest{
		home:         absHome,
		configPath:   absConfig,
		backupID:     backupID,
		token:        token,
		expectDigest: expectDigest,
	}, nil
}

// Home returns the absolute home directory the restore is resolved against.
func (r MCPRestoreRequest) Home() string { return r.home }

// ConfigPath returns the absolute explicit configuration path, or "".
func (r MCPRestoreRequest) ConfigPath() string { return r.configPath }

// BackupID returns the validated backup directory identifier.
func (r MCPRestoreRequest) BackupID() string { return r.backupID }

// Token returns the preview token authorizing this restore.
func (r MCPRestoreRequest) Token() string { return r.token }

// ExpectDigest returns the expected base digest of the target configuration.
func (r MCPRestoreRequest) ExpectDigest() string { return r.expectDigest }

// MCPRestoreResult is an alias for MCPApplyResult returned by MCPRestore.
type MCPRestoreResult = MCPApplyResult

// MCPRestore revalidates the restore preview token by running a fresh
// PreviewMCPRestore and verifying the token and base digest before calling
// skillsync.RestoreMCPConfig. A token mismatch, digest mismatch or missing token
// returns ErrNeedsPreview with no write.
func MCPRestore(req MCPRestoreRequest) (MCPRestoreResult, error) {
	if req.home == "" || req.backupID == "" {
		return MCPRestoreResult{}, errors.New("invalid mcp restore request")
	}
	if req.token == "" {
		return MCPRestoreResult{}, ErrNeedsPreview
	}
	if req.expectDigest == "" {
		return MCPRestoreResult{}, ErrNeedsPreview
	}
	preview, err := skillsync.PreviewMCPRestore(req.home, req.backupID, req.configPath)
	if err != nil {
		return MCPRestoreResult{}, fmt.Errorf("%w: revalidating mcp restore preview: %v", ErrNeedsPreview, err)
	}
	if preview == nil {
		return MCPRestoreResult{}, fmt.Errorf("%w: revalidating mcp restore preview: preview was nil", ErrNeedsPreview)
	}
	if req.token != mcpRestorePreviewToken(preview) {
		return MCPRestoreResult{}, ErrNeedsPreview
	}
	if preview.BaseDigest != "" && req.expectDigest != preview.BaseDigest {
		return MCPRestoreResult{}, ErrNeedsPreview
	}
	res, err := skillsync.RestoreMCPConfig(req.home, req.backupID, req.configPath, req.expectDigest)
	if err != nil {
		return MCPRestoreResult{}, err
	}
	return MCPRestoreResult{Result: &res}, nil
}
