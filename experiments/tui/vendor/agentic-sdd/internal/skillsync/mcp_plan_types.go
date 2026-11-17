package skillsync

import "errors"

// MCP-B shared contract. These types are intentionally separate from the
// MCP-A read-only MCPPreview: a plan carries exact target bytes, a digest and
// the evidence needed for one digest-checked apply. Only the transaction
// implementation reads targetBytes.
var (
	// ErrMCPStale reports that the target or plan evidence changed after preview.
	ErrMCPStale = errors.New("MCP configuration changed since preview")
	// ErrMCPConflict reports an unmanaged, user-edited or structurally ambiguous entry.
	ErrMCPConflict = errors.New("MCP configuration conflict")
	// ErrMCPUnsafeTarget reports a symlink, traversal, non-regular or out-of-scope target.
	ErrMCPUnsafeTarget = errors.New("unsafe MCP configuration target")
	// ErrMCPUnsupported reports a schema, runtime or JSONC edit this build cannot perform safely.
	ErrMCPUnsupported = errors.New("unsupported MCP configuration or runtime")
)

// MCPPlanEntry is one managed server action in a plan. OldText/NewText are the
// exact entry texts used for review and entry-level rollback; an add has empty
// OldText, a leave/disabled has neither, and a change has both.
type MCPPlanEntry struct {
	Name    string
	Op      string // add | change | leave | disabled | conflict
	Reason  string
	OldText string
	NewText string
	OldHash string
	NewHash string
}

// MCPPlan is the digest-bound, exact-byte plan for one MCP configuration edit.
// targetBytes is package-private so callers cannot write arbitrary bytes.
type MCPPlan struct {
	TargetPath      string
	Format          string // json | jsonc
	Schema          string
	BaseDigest      string
	PlanDigest      string
	Selected        []string
	Entries         []MCPPlanEntry
	Conflicts       []string
	AuthChoices     map[string]string
	RuntimeVersion  string
	RuntimeEvidence string
	NoOp            bool
	NotChecked      []string

	targetBytes []byte
}

// TargetBytes returns a copy of the exact bytes the plan would write. It is
// empty for a no-op or conflict plan.
func (p *MCPPlan) TargetBytes() []byte {
	if p == nil || len(p.targetBytes) == 0 {
		return nil
	}
	out := make([]byte, len(p.targetBytes))
	copy(out, p.targetBytes)
	return out
}

// Clone returns a deep copy suitable for an immutable appops request.
func (p *MCPPlan) Clone() *MCPPlan {
	if p == nil {
		return nil
	}
	out := *p
	out.Selected = append([]string(nil), p.Selected...)
	out.Conflicts = append([]string(nil), p.Conflicts...)
	out.NotChecked = append([]string(nil), p.NotChecked...)
	out.Entries = append([]MCPPlanEntry(nil), p.Entries...)
	if p.AuthChoices != nil {
		out.AuthChoices = make(map[string]string, len(p.AuthChoices))
		for k, v := range p.AuthChoices {
			out.AuthChoices[k] = v
		}
	}
	out.targetBytes = append([]byte(nil), p.targetBytes...)
	return &out
}

// MCPApplyOptions is the immutable apply context. ExpectDigest must equal the
// plan's base digest; OpenCodeBin is optional and is never executed by a
// read-only preview.
type MCPApplyOptions struct {
	Home         string
	ExpectDigest string
	OpenCodeBin  string
}

// MCPApplyResult is the real structured outcome of an apply or restore.
type MCPApplyResult struct {
	TargetPath      string
	BackupID        string
	BackupPath      string
	InstalledDigest string
	RuntimeVersion  string
	NoOp            bool
	Applied         []string
	Rollback        RollbackInfo
}

// MCPRestorePreview is the read-only current-versus-backup comparison. It
// contains only managed entries, never a whole-file replacement.
type MCPRestorePreview struct {
	BackupID   string
	TargetPath string
	BaseDigest string
	Entries    []MCPPlanEntry
	Conflicts  []string
}

// MCPTransactionIntent is durable evidence written before a mutation. Phase is
// prepared, renamed, manifest-written or complete.
type MCPTransactionIntent struct {
	BackupID            string
	TargetPath          string
	BaseDigest          string
	PlanDigest          string
	PriorManifestDigest string
	Phase               string
}
