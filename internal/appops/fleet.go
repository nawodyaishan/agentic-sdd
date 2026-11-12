package appops

// Feature 015 — typed fleet operation seam.
//
// This file mirrors the Feature 013 request/result pattern for the fleet
// engine: immutable, validated, absolute-path requests; exported read-only
// accessors; read operations that never write; and write operations that
// revalidate a preview token in the same call that writes. It is stateless by
// construction: no draft, plan or token is retained between calls, and no
// operation parses CLI prose or performs Git, network or model work.
//
// The engine remains the only owner of validation, digests, plan identity,
// locking, backup and rollback. appops does not duplicate fleet rules; it
// re-runs the engine's pure read projections to detect drift before a write,
// exactly as ApplyInstall/ApplyRestore do for skillsync.

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

// Fleet preview token versions make an encoding change explicit instead of
// silently accepting tokens produced by an older scheme.
const (
	fleetPlanTokenVersion    = "appops-fleet-plan-preview-v1"
	fleetRestoreTokenVersion = "appops-fleet-restore-preview-v1"
)

// FleetStatusRequest is one immutable, validated read of the fleet inventory.
// Construct it with NewFleetStatusRequest and read it through the accessors.
type FleetStatusRequest struct {
	configPath string
	repo       string
	home       string
}

// NewFleetStatusRequest validates and absolutizes one read-only status
// request. The config path and home are required; repo may be empty for the
// embedded canonical pack. No file is read and nothing is written here.
func NewFleetStatusRequest(configPath, repo, home string) (FleetStatusRequest, error) {
	if configPath == "" {
		return FleetStatusRequest{}, errors.New("fleet status request config path is required")
	}
	if home == "" {
		return FleetStatusRequest{}, errors.New("fleet status request home is required")
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return FleetStatusRequest{}, err
	}
	absRepo := ""
	if repo != "" {
		if absRepo, err = filepath.Abs(repo); err != nil {
			return FleetStatusRequest{}, err
		}
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return FleetStatusRequest{}, err
	}
	return FleetStatusRequest{configPath: absConfig, repo: absRepo, home: absHome}, nil
}

// ConfigPath returns the absolute personal fleet source path.
func (r FleetStatusRequest) ConfigPath() string { return r.configPath }

// Repo returns the absolute canonical-pack root, or "" for the embedded pack.
func (r FleetStatusRequest) Repo() string { return r.repo }

// Home returns the absolute home directory the status was resolved against.
func (r FleetStatusRequest) Home() string { return r.home }

// FleetStatusResult is the frozen read-only inventory produced by FleetStatus.
// Status is a fresh projection; callers must treat it as read-only. There is
// deliberately no preview token: a status read authorizes no write.
type FleetStatusResult struct {
	Status *fleet.Status
}

// FleetStatus runs fleet.BuildStatus. It performs no plan, lock, apply,
// backup, Git, network or model operation and creates or modifies no file.
func FleetStatus(req FleetStatusRequest) (FleetStatusResult, error) {
	status, err := fleet.BuildStatus(req.configPath, req.repo, req.home)
	if err != nil {
		return FleetStatusResult{}, err
	}
	return FleetStatusResult{Status: status}, nil
}

// FleetSourcePreviewRequest is one immutable, validated read-only personal
// source preview. draftBytes is copied at construction so a caller cannot
// change the request after validation, and appops retains no draft between
// calls.
type FleetSourcePreviewRequest struct {
	configPath string
	repo       string
	home       string
	draft      []byte
}

// NewFleetSourcePreviewRequest validates and absolutizes one personal-source
// preview request. configPath and home are required; repo may be empty. The
// draft bytes are copied. No file is read or written here.
func NewFleetSourcePreviewRequest(configPath, repo, home string, draftBytes []byte) (FleetSourcePreviewRequest, error) {
	if configPath == "" {
		return FleetSourcePreviewRequest{}, errors.New("fleet source preview request config path is required")
	}
	if home == "" {
		return FleetSourcePreviewRequest{}, errors.New("fleet source preview request home is required")
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return FleetSourcePreviewRequest{}, err
	}
	absRepo := ""
	if repo != "" {
		if absRepo, err = filepath.Abs(repo); err != nil {
			return FleetSourcePreviewRequest{}, err
		}
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return FleetSourcePreviewRequest{}, err
	}
	return FleetSourcePreviewRequest{
		configPath: absConfig,
		repo:       absRepo,
		home:       absHome,
		draft:      append([]byte(nil), draftBytes...),
	}, nil
}

// ConfigPath returns the absolute personal fleet source path.
func (r FleetSourcePreviewRequest) ConfigPath() string { return r.configPath }

// Repo returns the absolute canonical-pack root, or "" for the embedded pack.
func (r FleetSourcePreviewRequest) Repo() string { return r.repo }

// Home returns the absolute home directory the preview was resolved against.
func (r FleetSourcePreviewRequest) Home() string { return r.home }

// DraftBytes returns a copy of the staged candidate bytes. Changing the
// returned slice cannot change the request.
func (r FleetSourcePreviewRequest) DraftBytes() []byte {
	return append([]byte(nil), r.draft...)
}

// FleetSourcePreviewResult is the read-only outcome of one source preview.
// Normalized holds the deterministic strict-JSON encoding the engine validated
// and rendered; BaseDigest is SHA-256 of the bytes currently on disk and is the
// compare-and-swap base a later save must match. Preview repeats the engine's
// FilePreview so a caller can render the exact old-versus-new diff.
type FleetSourcePreviewResult struct {
	Preview    fleet.FilePreview
	Normalized []byte
	BaseDigest string
}

// FleetSourcePreview validates and renders a candidate personal source without
// writing anything. It returns the normalized bytes, the engine's file preview
// and the current base digest. The request's draft is never retained.
func FleetSourcePreview(req FleetSourcePreviewRequest) (FleetSourcePreviewResult, error) {
	preview, err := fleet.PreviewSourceWithRepo(req.configPath, req.draft, req.repo)
	if err != nil {
		return FleetSourcePreviewResult{}, err
	}
	return FleetSourcePreviewResult{
		Preview:    preview,
		Normalized: append([]byte(nil), preview.NewBytes...),
		BaseDigest: preview.OldDigest,
	}, nil
}

// FleetPreviewRequest is one immutable, validated read-only deployment plan
// request. It has no adopt field: adoption is an explicit 014 CLI decision
// outside 015 and every plan is built with adopt=false.
type FleetPreviewRequest struct {
	configPath string
	repo       string
	home       string
}

// NewFleetPreviewRequest validates and absolutizes one deployment-preview
// request. configPath and home are required; repo may be empty.
func NewFleetPreviewRequest(configPath, repo, home string) (FleetPreviewRequest, error) {
	if configPath == "" {
		return FleetPreviewRequest{}, errors.New("fleet preview request config path is required")
	}
	if home == "" {
		return FleetPreviewRequest{}, errors.New("fleet preview request home is required")
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return FleetPreviewRequest{}, err
	}
	absRepo := ""
	if repo != "" {
		if absRepo, err = filepath.Abs(repo); err != nil {
			return FleetPreviewRequest{}, err
		}
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return FleetPreviewRequest{}, err
	}
	return FleetPreviewRequest{configPath: absConfig, repo: absRepo, home: absHome}, nil
}

// ConfigPath returns the absolute personal fleet source path.
func (r FleetPreviewRequest) ConfigPath() string { return r.configPath }

// Repo returns the absolute canonical-pack root, or "" for the embedded pack.
func (r FleetPreviewRequest) Repo() string { return r.repo }

// Home returns the absolute home directory the plan was resolved against.
func (r FleetPreviewRequest) Home() string { return r.home }

// FleetPreviewResult is the read-only deployment preview. Plan is the fresh,
// sealed plan; Files is the complete per-file old-versus-desired projection
// (removals and conflicts first); PreviewToken authorizes exactly this plan and
// target state and nothing else. A plan, file or token is never user approval.
type FleetPreviewResult struct {
	Plan            *fleet.Plan
	Files           []fleet.FilePreview
	ConfigDigest    string
	SourceRev       string
	RendererVersion string
	Scope           string
	PreviewToken    string
}

// FleetPreview builds a fresh non-adopting plan from the current saved source,
// projects every file change, and returns a preview token. It writes nothing
// and acquires no engine lock. A stale config/source/renderer/scope or a
// changed target during projection returns the engine's typed refusal
// (errors.Is fleet.ErrStalePlan where applicable) instead of a partial plan.
func FleetPreview(req FleetPreviewRequest) (FleetPreviewResult, error) {
	plan, err := fleet.BuildPlan(req.configPath, req.repo, req.home, false)
	if err != nil {
		return FleetPreviewResult{}, err
	}
	files, err := fleet.PlanFiles(plan)
	if err != nil {
		return FleetPreviewResult{}, err
	}
	return FleetPreviewResult{
		Plan:            plan,
		Files:           copyFleetFilePreviews(files),
		ConfigDigest:    plan.ConfigDigest,
		SourceRev:       plan.SourceRev,
		RendererVersion: plan.RendererVersion,
		Scope:           plan.Scope,
		PreviewToken:    fleetPlanPreviewToken(plan, files),
	}, nil
}

// FleetApplyRequest is one immutable, validated deployment of a previously
// previewed plan. The constructor takes a deep copy of the plan, so later
// caller mutation cannot change the request; Plan returns another copy.
type FleetApplyRequest struct {
	plan *fleet.Plan
	home string
}

// NewFleetApplyRequest validates and copies one fleet apply request. The plan
// must be non-nil and carry a verifiable identity; home is required and made
// absolute. It does not re-seal, re-render or otherwise mutate the plan.
func NewFleetApplyRequest(plan *fleet.Plan, home string) (FleetApplyRequest, error) {
	if plan == nil {
		return FleetApplyRequest{}, errors.New("fleet apply request plan is required")
	}
	if home == "" {
		return FleetApplyRequest{}, errors.New("fleet apply request home is required")
	}
	if plan.PlanID == "" || !plan.VerifyPlanID() {
		return FleetApplyRequest{}, errors.New("fleet apply request plan identity is invalid; generate a fresh plan")
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return FleetApplyRequest{}, err
	}
	return FleetApplyRequest{plan: cloneFleetPlan(plan), home: absHome}, nil
}

// Plan returns a deep copy of the request's plan. Changing it cannot change the
// request.
func (r FleetApplyRequest) Plan() *fleet.Plan { return cloneFleetPlan(r.plan) }

// Home returns the absolute home directory the apply is resolved against.
func (r FleetApplyRequest) Home() string { return r.home }

// PlanID returns the immutable plan identity the apply was confirmed against.
func (r FleetApplyRequest) PlanID() string {
	if r.plan == nil {
		return ""
	}
	return r.plan.PlanID
}

// FleetRestorePreviewRequest is one immutable, validated read-only restore
// preview for a managed fleet backup.
type FleetRestorePreviewRequest struct {
	home     string
	backupID string
}

// NewFleetRestorePreviewRequest validates and absolutizes one restore preview
// request. home is required and the backup ID must have the exact
// backup-directory syntax.
func NewFleetRestorePreviewRequest(home, backupID string) (FleetRestorePreviewRequest, error) {
	if home == "" {
		return FleetRestorePreviewRequest{}, errors.New("fleet restore preview request home is required")
	}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		return FleetRestorePreviewRequest{}, err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return FleetRestorePreviewRequest{}, err
	}
	return FleetRestorePreviewRequest{home: absHome, backupID: backupID}, nil
}

// Home returns the absolute home directory the preview is resolved against.
func (r FleetRestorePreviewRequest) Home() string { return r.home }

// BackupID returns the validated backup directory name.
func (r FleetRestorePreviewRequest) BackupID() string { return r.backupID }

// FleetRestorePreviewResult is the read-only restore projection. Preview holds
// the exact current-versus-restore files; PreviewToken covers the backup
// identity and the current target hashes observed here.
type FleetRestorePreviewResult struct {
	BackupID     string
	Preview      *fleet.RestorePreview
	PreviewToken string
}

// FleetRestorePreview reports exactly what RestoreBackup would do. It writes
// nothing, creates no backup and acquires no engine lock. Legacy/unknown,
// modified-target and corrupt-backup refusals are returned as typed engine
// errors rather than a misleading preview.
func FleetRestorePreview(req FleetRestorePreviewRequest) (FleetRestorePreviewResult, error) {
	preview, err := fleet.PreviewRestoreBackup(req.home, req.backupID)
	if err != nil {
		return FleetRestorePreviewResult{}, err
	}
	return FleetRestorePreviewResult{
		BackupID:     req.backupID,
		Preview:      preview,
		PreviewToken: fleetRestorePreviewToken(preview),
	}, nil
}

// FleetRestoreApplyRequest is one immutable, validated restore of a previously
// previewed managed backup.
type FleetRestoreApplyRequest struct {
	home     string
	backupID string
}

// NewFleetRestoreApplyRequest validates and absolutizes one restore apply
// request. home is required and the backup ID must have the exact
// backup-directory syntax.
func NewFleetRestoreApplyRequest(home, backupID string) (FleetRestoreApplyRequest, error) {
	if home == "" {
		return FleetRestoreApplyRequest{}, errors.New("fleet restore request home is required")
	}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		return FleetRestoreApplyRequest{}, err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return FleetRestoreApplyRequest{}, err
	}
	return FleetRestoreApplyRequest{home: absHome, backupID: backupID}, nil
}

// Home returns the absolute home directory the restore is resolved against.
func (r FleetRestoreApplyRequest) Home() string { return r.home }

// BackupID returns the validated backup directory name.
func (r FleetRestoreApplyRequest) BackupID() string { return r.backupID }

// FleetDoctorRequest is one immutable, validated local-only doctor request.
// Empty OpenCodeBin and CatalogDump mean no local tool is consulted; doctor
// never searches PATH and never contacts the network or a model.
type FleetDoctorRequest struct {
	models      []string
	opencodeBin string
	catalogDump string
}

// NewFleetDoctorRequest validates and absolutizes one doctor request. Models
// are copied in report order; optional local tool paths are absolutized.
func NewFleetDoctorRequest(models []string, opencodeBin, catalogDump string) (FleetDoctorRequest, error) {
	absBin := ""
	if opencodeBin != "" {
		var err error
		if absBin, err = filepath.Abs(opencodeBin); err != nil {
			return FleetDoctorRequest{}, err
		}
	}
	absDump := ""
	if catalogDump != "" {
		var err error
		if absDump, err = filepath.Abs(catalogDump); err != nil {
			return FleetDoctorRequest{}, err
		}
	}
	return FleetDoctorRequest{
		models:      append([]string(nil), models...),
		opencodeBin: absBin,
		catalogDump: absDump,
	}, nil
}

// Models returns a copy of the routes to label.
func (r FleetDoctorRequest) Models() []string { return append([]string(nil), r.models...) }

// OpenCodeBin returns the explicit local OpenCode binary path, or "".
func (r FleetDoctorRequest) OpenCodeBin() string { return r.opencodeBin }

// CatalogDump returns the explicit local catalog-dump path, or "".
func (r FleetDoctorRequest) CatalogDump() string { return r.catalogDump }

// FleetDoctorResult is the labeled local-evidence report. It is a diagnostic,
// never an entitlement or runtime-verification claim.
type FleetDoctorResult struct {
	Report fleet.DoctorReport
}

// FleetDoctor runs fleet.RunDoctor from local evidence only. The default
// request inspects nothing and reports every route as "not checked (offline)".
func FleetDoctor(req FleetDoctorRequest) FleetDoctorResult {
	return FleetDoctorResult{Report: fleet.RunDoctor(fleet.DoctorOptions{
		Models:      req.Models(),
		OpenCodeBin: req.opencodeBin,
		CatalogDump: req.catalogDump,
	})}
}

// FleetApplyResult is the shared ApplyResult returned by ApplyFleet, with the
// FleetApply detail populated. It is an alias so the single OpsHost fleet-write
// contract stays identical across save/apply/restore kinds.
type FleetApplyResult = ApplyResult

// FleetRestoreResult is the shared ApplyResult returned by ApplyFleetRestore,
// with the FleetRestore detail populated.
type FleetRestoreResult = ApplyResult

// ApplyFleet revalidates a fleet preview token and then applies its plan. It
// requires a non-empty token, re-runs the read-only PlanFiles projection over
// the immutable plan, recomputes the token and only then calls
// fleet.ApplyPlan. An empty token, a stale config/source/renderer/scope, a
// changed target hash or any other token mismatch returns ErrNeedsPreview with
// no write, no backup and no lock acquisition. The engine's own mutation-time
// checks remain authoritative.
//
// The result populates the additive FleetApply field; install/restore fields
// keep their existing meaning and stay empty.
func ApplyFleet(req FleetApplyRequest, token string, out io.Writer) (ApplyResult, error) {
	if out == nil {
		out = io.Discard
	}
	plan := cloneFleetPlan(req.plan)
	result := ApplyResult{
		Operation:  skillsync.OperationApply,
		FleetApply: &fleet.ApplyResult{PlanID: req.PlanID()},
	}
	if token == "" {
		result.Outcome = skillsync.OutcomeFailed
		return result, ErrNeedsPreview
	}
	files, err := fleet.PlanFiles(plan)
	if err != nil {
		result.Outcome = skillsync.OutcomeFailed
		return result, fmt.Errorf("%w: revalidating fleet preview: %v", ErrNeedsPreview, err)
	}
	if token != fleetPlanPreviewToken(plan, files) {
		result.Outcome = skillsync.OutcomeFailed
		return result, ErrNeedsPreview
	}
	applied, err := fleet.ApplyPlan(plan, req.home, out)
	result.FleetApply = &applied
	result.BackupID = applied.BackupID
	result.BackupPath = applied.BackupPath
	result.SourceDigest = plan.SourceRev
	result.PreviewToken = token
	if err != nil {
		result.Outcome = skillsync.OutcomeFailed
		return result, err
	}
	if applied.NoOp {
		result.Outcome = skillsync.OutcomeNoChange
	} else {
		result.Outcome = skillsync.OutcomeApplied
	}
	return result, nil
}

// ApplyFleetRestore revalidates a restore preview token and then restores the
// managed backup. It requires a non-empty token, re-runs
// PreviewRestoreBackup, recomputes the token and only then calls
// RestoreBackup. A changed target, missing/corrupt backup or any other token
// mismatch returns ErrNeedsPreview with no write. The result populates the
// additive FleetRestore field.
func ApplyFleetRestore(req FleetRestoreApplyRequest, token string, out io.Writer) (ApplyResult, error) {
	if out == nil {
		out = io.Discard
	}
	result := ApplyResult{
		Operation:    skillsync.OperationRestore,
		FleetRestore: &fleet.RestoreResult{RestoredFrom: req.backupID},
	}
	if token == "" {
		result.Outcome = skillsync.OutcomeFailed
		return result, ErrNeedsPreview
	}
	preview, err := fleet.PreviewRestoreBackup(req.home, req.backupID)
	if err != nil {
		result.Outcome = skillsync.OutcomeFailed
		return result, fmt.Errorf("%w: revalidating fleet restore preview: %v", ErrNeedsPreview, err)
	}
	if token != fleetRestorePreviewToken(preview) {
		result.Outcome = skillsync.OutcomeFailed
		return result, ErrNeedsPreview
	}
	restored, err := fleet.RestoreBackup(req.home, req.backupID, out)
	result.FleetRestore = &restored
	result.BackupID = restored.BackupID
	result.BackupPath = restored.BackupPath
	result.PreviewToken = token
	if err != nil {
		result.Outcome = skillsync.OutcomeFailed
		return result, err
	}
	if restored.NoOp {
		result.Outcome = skillsync.OutcomeNoChange
	} else {
		result.Outcome = skillsync.OutcomeApplied
	}
	return result, nil
}

// fleetPlanPreviewToken covers the plan identity, the config/renderer/scope
// identity and every current target hash observed by PlanFiles. It authorizes
// exactly that state; it is not user approval.
func fleetPlanPreviewToken(plan *fleet.Plan, files []fleet.FilePreview) string {
	enc := newTokenEncoding()
	enc.field(fleetPlanTokenVersion)
	enc.field(plan.PlanID)
	enc.field(plan.ConfigDigest)
	enc.field(plan.SourceRev)
	enc.field(plan.RendererVersion)
	enc.field(plan.Scope)
	enc.field(plan.TargetRoot)
	enc.field(plan.ConfigPath)
	enc.field(plan.RepoPath)
	enc.field(strconv.Itoa(len(files)))
	for _, f := range files {
		enc.field(f.Path)
		enc.field(f.Op)
		enc.field(f.OldDigest)
		enc.field(f.NewDigest)
	}
	return enc.sum()
}

// fleetRestorePreviewToken covers the backup identity and every current target
// hash observed by the restore preview.
func fleetRestorePreviewToken(preview *fleet.RestorePreview) string {
	enc := newTokenEncoding()
	enc.field(fleetRestoreTokenVersion)
	enc.field(preview.BackupID)
	enc.field(preview.BackupPath)
	enc.field(strconv.Itoa(len(preview.Files)))
	for _, f := range preview.Files {
		enc.field(f.Path)
		enc.field(f.Op)
		enc.field(f.OldDigest)
		enc.field(f.NewDigest)
	}
	return enc.sum()
}

// cloneFleetPlan returns a deep copy so no request or result can be mutated
// through another holder. ProposedOp carries only value fields.
func cloneFleetPlan(plan *fleet.Plan) *fleet.Plan {
	if plan == nil {
		return nil
	}
	clone := *plan
	clone.FileList = append([]string(nil), plan.FileList...)
	clone.PermNotes = append([]string(nil), plan.PermNotes...)
	clone.ProposedOps = append([]fleet.ProposedOp(nil), plan.ProposedOps...)
	if plan.LastKnownInstalled != nil {
		clone.LastKnownInstalled = make(map[string]string, len(plan.LastKnownInstalled))
		for target, digest := range plan.LastKnownInstalled {
			clone.LastKnownInstalled[target] = digest
		}
	}
	return &clone
}

// copyFleetFilePreviews copies the engine's byte slices so a result holder
// cannot mutate engine state through a shared backing array.
func copyFleetFilePreviews(in []fleet.FilePreview) []fleet.FilePreview {
	if in == nil {
		return nil
	}
	out := make([]fleet.FilePreview, len(in))
	for i, preview := range in {
		out[i] = preview
		out[i].OldBytes = append([]byte(nil), preview.OldBytes...)
		out[i].NewBytes = append([]byte(nil), preview.NewBytes...)
	}
	return out
}
