package appops

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strconv"

	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

// ErrNeedsPreview reports that an apply was refused because the request
// inputs, the selected source identity or the proposed change set no longer
// match the preview that produced its token. Nothing was written; the caller
// must run a fresh preview and obtain fresh consent. errors.Is can match it
// even when the cause is wrapped.
var ErrNeedsPreview = errors.New("inputs or planned changes changed since preview; a new preview is required")

// previewTokenVersion makes a token encoding change explicit instead of
// silently accepting tokens produced by an older scheme.
const previewTokenVersion = "appops-preview-v1"

// PreviewResult is the concrete read-only outcome of one preview through the
// shared operation seam. Planned holds the complete plan, including absent and
// up-to-date entries; Applied and Rollback are always empty for a preview.
// PreviewToken authorizes exactly the inputs and plan captured here.
type PreviewResult struct {
	Operation    skillsync.OperationKind
	Outcome      skillsync.Outcome
	BackupID     string
	BackupPath   string
	Planned      []skillsync.ResultChange
	Applied      []skillsync.ResultChange
	Rollback     skillsync.RollbackInfo
	SourceDigest string
	PreviewToken string
}

// ApplyResult is the concrete outcome of one apply through the shared
// operation seam. Planned is the plan the engine used, including entries it
// left untouched; Applied lists only committed changes and stays empty on
// failure. BackupID and BackupPath identify the backup the operation wrote
// (for a restore, the pre-restore backup). PreviewToken is the token this
// apply consumed; a refused apply carries none. FleetApply and FleetRestore
// are additive fleet-engine details populated only by ApplyFleet and
// ApplyFleetRestore; they stay nil for install and skillsync restore, whose
// existing fields and semantics are unchanged.
type ApplyResult struct {
	Operation    skillsync.OperationKind
	Outcome      skillsync.Outcome
	BackupID     string
	BackupPath   string
	Planned      []skillsync.ResultChange
	Applied      []skillsync.ResultChange
	Rollback     skillsync.RollbackInfo
	SourceDigest string
	PreviewToken string
	FleetApply   *fleet.ApplyResult
	FleetRestore *fleet.RestoreResult
}

// PreviewInstall runs the read-only install engine variant and returns its
// exact transcript, structured plan, source identity and a preview token.
// Nothing is written.
func PreviewInstall(req InstallRequest, out io.Writer) (PreviewResult, error) {
	if out == nil {
		out = io.Discard
	}
	result, err := skillsync.SyncWithOptionsResult(req.repo, req.home, false, req.opts, out)
	preview := previewFromEngine(result)
	if err != nil {
		return preview, err
	}
	preview.PreviewToken = installPreviewToken(req, result)
	return preview, nil
}

// PreviewRestore runs the read-only restore engine variant and returns its
// exact transcript, structured plan, backup/source identity and a preview
// token. Nothing is written.
func PreviewRestore(req RestoreRequest, out io.Writer) (PreviewResult, error) {
	if out == nil {
		out = io.Discard
	}
	result, err := skillsync.RestoreResult(req.repo, req.home, req.id, false, out)
	preview := previewFromEngine(result)
	if err != nil {
		return preview, err
	}
	preview.PreviewToken = restorePreviewToken(req, result)
	return preview, nil
}

// ApplyInstall revalidates the install before writing. It requires a
// non-empty token, re-runs the read-only preview, recomputes its token and
// only then runs the apply engine variant when the token still matches. An
// empty token, a changed request/source/plan or a preview that can no longer
// be reproduced returns ErrNeedsPreview with no writes. The revalidation is
// pre-execution drift detection; it is not isolation from hostile concurrent
// writers, so the engine's own mutation-time checks stay authoritative.
func ApplyInstall(req InstallRequest, token string, out io.Writer) (ApplyResult, error) {
	if out == nil {
		out = io.Discard
	}
	if token == "" {
		return ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeFailed}, ErrNeedsPreview
	}
	current, err := skillsync.SyncWithOptionsResult(req.repo, req.home, false, req.opts, io.Discard)
	if err != nil {
		return refusedApply(current), fmt.Errorf("%w: revalidating preview: %v", ErrNeedsPreview, err)
	}
	if token != installPreviewToken(req, current) {
		return refusedApply(current), ErrNeedsPreview
	}
	result, err := skillsync.SyncWithOptionsResult(req.repo, req.home, true, req.opts, out)
	applied := applyFromEngine(result)
	applied.PreviewToken = token
	return applied, err
}

// ApplyRestore revalidates the restore before writing, using the same rules as
// ApplyInstall: the token, the request, the backup identity/payloads and the
// complete proposed change set must still match. A mismatch returns
// ErrNeedsPreview with no writes.
func ApplyRestore(req RestoreRequest, token string, out io.Writer) (ApplyResult, error) {
	if out == nil {
		out = io.Discard
	}
	if token == "" {
		return ApplyResult{Operation: skillsync.OperationRestore, Outcome: skillsync.OutcomeFailed}, ErrNeedsPreview
	}
	current, err := skillsync.RestoreResult(req.repo, req.home, req.id, false, io.Discard)
	if err != nil {
		return refusedApply(current), fmt.Errorf("%w: revalidating preview: %v", ErrNeedsPreview, err)
	}
	if token != restorePreviewToken(req, current) {
		return refusedApply(current), ErrNeedsPreview
	}
	result, err := skillsync.RestoreResult(req.repo, req.home, req.id, true, out)
	applied := applyFromEngine(result)
	applied.PreviewToken = token
	return applied, err
}

// previewFromEngine copies one engine result into the preview shape so a
// caller cannot mutate the engine's slices through the result.
func previewFromEngine(result skillsync.OperationResult) PreviewResult {
	return PreviewResult{
		Operation:    result.Operation,
		Outcome:      result.Outcome,
		BackupID:     result.BackupID,
		BackupPath:   result.BackupPath,
		Planned:      append([]skillsync.ResultChange(nil), result.Planned...),
		Applied:      append([]skillsync.ResultChange(nil), result.Applied...),
		Rollback:     result.Rollback,
		SourceDigest: result.SourceDigest,
	}
}

// applyFromEngine copies one engine result into the apply shape.
func applyFromEngine(result skillsync.OperationResult) ApplyResult {
	return ApplyResult{
		Operation:    result.Operation,
		Outcome:      result.Outcome,
		BackupID:     result.BackupID,
		BackupPath:   result.BackupPath,
		Planned:      append([]skillsync.ResultChange(nil), result.Planned...),
		Applied:      append([]skillsync.ResultChange(nil), result.Applied...),
		Rollback:     result.Rollback,
		SourceDigest: result.SourceDigest,
	}
}

// refusedApply reports an apply that did not run. The fresh plan and source
// identity are informational only: without a token they authorize nothing.
func refusedApply(fresh skillsync.OperationResult) ApplyResult {
	refused := applyFromEngine(fresh)
	refused.Outcome = skillsync.OutcomeFailed
	return refused
}

// installPreviewToken covers the install request identity and its complete
// proposed change set. The apply flag is deliberately absent: the token
// authorizes a plan, and the apply must reproduce that same plan before it
// may write.
func installPreviewToken(req InstallRequest, result skillsync.OperationResult) string {
	return previewToken(result.Operation, req.home, req.repo, "", req.opts, result)
}

// restorePreviewToken covers the restore request identity, the backup
// identity carried by SourceDigest, and the complete proposed change set.
func restorePreviewToken(req RestoreRequest, result skillsync.OperationResult) string {
	return previewToken(result.Operation, req.home, req.repo, req.id, skillsync.Options{}, result)
}

// previewToken builds the canonical SHA-256 token described by plan §1: the
// operation kind, resolved home, resolved repo or embedded marker, normalized
// options/selection, the engine's source identity and every planned change in
// order, including up-to-date entries.
func previewToken(operation skillsync.OperationKind, home, repo, backupID string, opts skillsync.Options, result skillsync.OperationResult) string {
	enc := newTokenEncoding()
	enc.field(previewTokenVersion)
	enc.field(string(operation))
	enc.field(home)
	if repo == "" {
		enc.field("embedded")
	} else {
		enc.field("repo")
		enc.field(repo)
	}
	if opts.ClientIDs == nil {
		enc.field("all-clients")
	} else {
		enc.field("selected-clients")
		enc.field(strconv.Itoa(len(opts.ClientIDs)))
		for _, id := range opts.ClientIDs {
			enc.field(id)
		}
	}
	enc.boolean(opts.WithOpenCode)
	enc.boolean(opts.OpenCodeFreeFirst)
	enc.field(backupID)
	enc.field(result.SourceDigest)
	enc.field(strconv.Itoa(len(result.Planned)))
	for _, change := range result.Planned {
		enc.field(change.Client)
		enc.field(change.Kind)
		enc.field(change.ID)
		enc.field(change.Path)
		enc.field(string(change.Action))
	}
	return enc.sum()
}

// tokenEncoding is a canonical, length-prefixed field sequence. Lengths make
// the sequence unambiguous even when a path or ID contains a separator.
type tokenEncoding struct {
	h hash.Hash
}

func newTokenEncoding() *tokenEncoding {
	return &tokenEncoding{h: sha256.New()}
}

func (t *tokenEncoding) field(value string) {
	fmt.Fprintf(t.h, "%d:%s\n", len(value), value)
}

func (t *tokenEncoding) boolean(value bool) {
	if value {
		t.field("1")
		return
	}
	t.field("0")
}

func (t *tokenEncoding) sum() string {
	return hex.EncodeToString(t.h.Sum(nil))
}
