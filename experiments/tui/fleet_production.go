package tui

// Production fleet adapters. They are the only TUI code that reaches the
// shared appops fleet seam: every method below is read-only, absolutizes
// through appops, and never writes a file, acquires the engine lock, calls a
// model, reads a credential or performs a Git operation. The screens receive
// only immutable view models, so they cannot mutate engine state.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/fleet"
)

// AppOpsFleetReader implements FleetReader through appops.FleetStatus. Home and
// Repo are fixed for the session; every Status call is one read-only projection
// of the explicitly selected source. Construction performs no I/O.
type AppOpsFleetReader struct {
	Home string
	Repo string
}

// Status reads one typed fleet status. A missing or invalid source returns a
// truthful unavailable view with the underlying error, never invented counts.
func (r AppOpsFleetReader) Status(ctx context.Context, source string) (FleetStatusView, error) {
	if err := contextDone(ctx); err != nil {
		return FleetStatusView{Available: false, SourcePath: source, SourceState: "not checked", Message: err.Error()}, err
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return FleetStatusView{
			Available:             false,
			SourceState:           "not selected",
			PersonalConfigSummary: "No fleet.json selected — press s to specify an explicit path",
			InstalledSummary:      "No managed instances inspected",
			RuntimeStatus:         "not checked",
			NextAction:            "Select an explicit local fleet.json source",
			Message:               "No fleet source is selected.",
		}, nil
	}
	if strings.TrimSpace(r.Home) == "" {
		return FleetStatusView{
			Available:   false,
			SourcePath:  source,
			SourceState: "not available",
			Message:     "Fleet status needs a resolved home; the TUI was started without one.",
		}, nil
	}
	req, err := appops.NewFleetStatusRequest(source, r.Repo, r.Home)
	if err != nil {
		return fleetStatusErrorView(source, err), err
	}
	result, err := appops.FleetStatus(req)
	if err != nil {
		return fleetStatusErrorView(source, err), err
	}
	return mapFleetStatus(result.Status), nil
}

// fleetStatusErrorView reports a source read failure without pretending the
// fleet is empty.
func fleetStatusErrorView(source string, err error) FleetStatusView {
	state := "invalid"
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist) {
		state = "missing"
	}
	return FleetStatusView{
		Available:             false,
		SourcePath:            source,
		SourceState:           state,
		PersonalConfigSummary: "Not loaded: " + redactError(err),
		InstalledSummary:      "No reliable installed state was derived",
		RuntimeStatus:         "not checked",
		NextAction:            "Choose a readable local fleet.json and reload (r)",
		Message:               redactError(err),
	}
}

// mapFleetStatus projects the engine's typed status into the TUI view model.
// It copies raw source/digest facts so the editor can reconstruct a draft, but
// it derives no ownership or plan identity of its own.
func mapFleetStatus(status *fleet.Status) FleetStatusView {
	if status == nil {
		return FleetStatusView{Available: false, SourceState: "invalid", Message: "The engine returned no fleet status."}
	}
	view := FleetStatusView{
		Available:             true,
		SourcePath:            status.ConfigPath,
		SourceState:           "loaded",
		TotalInstances:        len(status.Instances),
		EnabledCount:          countOf(status.Counts, fleet.StatusCountEnabled),
		DisabledCount:         countOf(status.Counts, fleet.StatusCountDisabled),
		DriftedCount:          countOf(status.Counts, string(fleet.InstalledManagedDrifted)) + countOf(status.Counts, string(fleet.InstalledPendingApply)),
		UnmanagedCount:        countOf(status.Counts, string(fleet.InstalledUnmanaged)),
		LastOutcome:           "None (read-only inspection)",
		RuntimeStatus:         "not checked",
		PersonalConfigSummary: fmt.Sprintf("Loaded (%d instance(s) configured)", len(status.Instances)),
	}
	conflicts := countOf(status.Counts, fleet.StatusCountConflicts)
	view.InstalledSummary = fmt.Sprintf("%d current · %d drifted · %d pending apply · %d absent · %d unmanaged · %d legacy unknown",
		countOf(status.Counts, string(fleet.InstalledManagedCurrent)),
		countOf(status.Counts, string(fleet.InstalledManagedDrifted)),
		countOf(status.Counts, string(fleet.InstalledPendingApply)),
		countOf(status.Counts, string(fleet.InstalledAbsent)),
		countOf(status.Counts, string(fleet.InstalledUnmanaged)),
		countOf(status.Counts, string(fleet.InstalledLegacyUnknown)),
	)
	view.NextAction = fleetNextAction(status, conflicts, view)
	if status.ConfigDigest != "" {
		view.PersonalConfigSummary += " · digest " + shortDigest(status.ConfigDigest)
	}

	view.Instances = make([]FleetInstanceView, 0, len(status.Instances))
	for _, inst := range status.Instances {
		model := inst.RenderedModel
		if model == "" {
			model = inst.ConfiguredModel
		}
		route, variant := splitFleetModel(model)
		changed := 0
		switch inst.Installed {
		case fleet.InstalledManagedDrifted, fleet.InstalledPendingApply:
			changed = 1
		}
		view.Instances = append(view.Instances, FleetInstanceView{
			InstanceID:        inst.InstanceID,
			ProviderRoute:     route,
			Model:             model,
			CompactModel:      CompactModelName(model),
			Variant:           variant,
			Enabled:           inst.Enabled,
			InstalledState:    displayInstalledState(inst.Installed),
			SourceProvenance:  fleetProvenanceLabel(inst),
			ChangedCount:      changed,
			RuntimeState:      "not checked",
			DriftDetail:       sanitizeFleetReason(inst.Conflict),
			SourceRef:         inst.SourceRef,
			ConfiguredModel:   inst.ConfiguredModel,
			SourceDigest:      inst.SourceDigest,
			DesiredDigest:     inst.DesiredDigest,
			CurrentDigest:     inst.CurrentDigest,
			LastAppliedDigest: inst.LastAppliedDigest,
			Provenance:        inst.Provenance,
		})
	}
	sort.SliceStable(view.Instances, func(i, j int) bool {
		return view.Instances[i].InstanceID < view.Instances[j].InstanceID
	})
	return view
}

func fleetNextAction(status *fleet.Status, conflicts int, view FleetStatusView) string {
	switch {
	case len(status.Instances) == 0:
		return "Define at least one instance in the selected personal source"
	case conflicts > 0:
		return "Review conflicts and drift in Recovery before planning a deployment"
	case countOf(status.Counts, string(fleet.InstalledUnmanaged)) > 0:
		return "Review unmanaged files in Recovery; the TUI never takes ownership of or overwrites them"
	case view.DriftedCount > 0:
		return "Inspect drifted instances, then review a fresh deployment plan"
	default:
		return "Inspect profiles or review a fresh deployment plan"
	}
}

func countOf(counts map[string]int, key string) int {
	if counts == nil {
		return 0
	}
	return counts[key]
}

// splitFleetModel splits <provider>/<model>[#variant] for display without
// changing the exact configured identifier.
func splitFleetModel(model string) (route, variant string) {
	if model == "" {
		return "", ""
	}
	base := model
	if i := strings.IndexByte(base, '#'); i >= 0 {
		variant = base[i+1:]
		base = base[:i]
	}
	if i := strings.IndexByte(base, '/'); i > 0 {
		route = base[:i]
	}
	return route, variant
}

// displayInstalledState maps the engine's stable state vocabulary to the
// display vocabulary the rest of the TUI already uses.
func displayInstalledState(state fleet.InstalledState) string {
	switch state {
	case fleet.InstalledManagedCurrent:
		return "managed current"
	case fleet.InstalledManagedDrifted:
		return "drifted"
	case fleet.InstalledPendingApply:
		return "pending apply"
	case fleet.InstalledUnmanaged:
		return "unmanaged"
	case fleet.InstalledLegacyUnknown:
		return "legacy unknown"
	case fleet.InstalledUnsafeTarget:
		return "unsafe target"
	case fleet.InstalledDisabled:
		return "disabled (managed removal)"
	case fleet.InstalledAbsent:
		return "not installed"
	default:
		return string(state)
	}
}

// fleetProvenanceLabel composes the source reference and ownership provenance
// without inventing either one.
func fleetProvenanceLabel(inst fleet.InstanceStatus) string {
	provenance := inst.Provenance
	if provenance == "" {
		provenance = "unknown"
	}
	if inst.SourceRef == "" {
		return provenance
	}
	return inst.SourceRef + " · " + provenance
}

// sanitizeFleetReason keeps the engine's factual conflict wording but never
// displays an adoption/force/fix-all affordance. Adoption remains an explicit
// 014 CLI decision outside 015.
func sanitizeFleetReason(reason string) string {
	if reason == "" {
		return ""
	}
	replacements := []string{
		"use --adopt to remove it with a backup",
		"review it and use an explicit CLI decision outside the TUI",
		"use --adopt after review",
		"review the file and use an explicit CLI decision outside the TUI",
		"--adopt",
		"an explicit CLI decision",
	}
	clean := reason
	for i := 0; i+1 < len(replacements); i += 2 {
		clean = strings.ReplaceAll(clean, replacements[i], replacements[i+1])
	}
	return clean
}

func shortDigest(digest string) string {
	if len(digest) <= 12 {
		return digest
	}
	return digest[:12] + "…"
}

// redactError keeps a typed engine message while stripping any control
// characters. It never adds credentials or full private JSON.
func redactError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < 0x20 {
			return -1
		}
		return r
	}, err.Error())
	return strings.TrimSpace(text)
}

func contextDone(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// AppOpsFleetEditor implements FleetEditor. It reconstructs the current
// personal document from the typed status (source reference, configured model
// and enablement per instance), applies the in-memory edits, encodes the
// deterministic strict JSON and asks appops for a read-only preview. Nothing is
// written and no draft is retained.
type AppOpsFleetEditor struct {
	Home string
	Repo string
}

// PreviewSource returns the validated normalized bytes, file diff and CAS base
// digest for the current source plus staged edits.
func (e AppOpsFleetEditor) PreviewSource(ctx context.Context, source string, edits []FleetModelEdit) (FleetSourcePreview, error) {
	if err := contextDone(ctx); err != nil {
		return FleetSourcePreview{}, err
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return FleetSourcePreview{
			Available: false,
			Message:   "Select an explicit fleet.json source before staging changes.",
			DiffLines: []string{"(no source selected)"},
		}, nil
	}
	if strings.TrimSpace(e.Home) == "" {
		return FleetSourcePreview{
			Available:  false,
			SourcePath: source,
			Op:         "unavailable",
			Message:    "The TUI was started without a resolved home; source preview is unavailable.",
			DiffLines:  []string{"(source preview requires a resolved home)"},
		}, nil
	}
	statusReq, err := appops.NewFleetStatusRequest(source, e.Repo, e.Home)
	if err != nil {
		return FleetSourcePreview{}, err
	}
	status, err := appops.FleetStatus(statusReq)
	if err != nil {
		return FleetSourcePreview{}, err
	}
	// Status projects instances only. Re-read the typed document for the
	// top-level fleet/preset metadata so a staged edit cannot look like it
	// deletes fields it never touched. This is the same read-only path the
	// status used; a failure is reported instead of inventing metadata.
	raw, _, _, err := fleet.LoadConfigFile(source)
	if err != nil {
		return FleetSourcePreview{}, err
	}
	cfg, err := fleetConfigFromStatus(status.Status, raw.Fleet, raw.Preset)
	if err != nil {
		return FleetSourcePreview{}, err
	}
	for _, edit := range edits {
		inst, ok := cfg.Instances[edit.InstanceID]
		if !ok {
			return FleetSourcePreview{}, fmt.Errorf("instance %q is not in the selected source; reload and retry", edit.InstanceID)
		}
		if trimmed := strings.TrimSpace(edit.Model); trimmed != "" {
			inst.Model = trimmed
		}
		inst.Enabled = optionalEnabled(edit.Enabled)
		cfg.Instances[edit.InstanceID] = inst
	}
	normalized, err := fleet.EncodeConfig(cfg)
	if err != nil {
		return FleetSourcePreview{}, err
	}
	previewReq, err := appops.NewFleetSourcePreviewRequest(source, e.Repo, e.Home, normalized)
	if err != nil {
		return FleetSourcePreview{}, err
	}
	result, err := appops.FleetSourcePreview(previewReq)
	if err != nil {
		return FleetSourcePreview{}, err
	}
	preview := result.Preview
	return FleetSourcePreview{
		Available:  true,
		SourcePath: source,
		OldBytes:   append([]byte(nil), preview.OldBytes...),
		NewBytes:   append([]byte(nil), preview.NewBytes...),
		OldDigest:  preview.OldDigest,
		NewDigest:  preview.NewDigest,
		BaseDigest: result.BaseDigest,
		Op:         preview.Op,
		DiffLines:  fleetLineDiffLines(preview.OldBytes, preview.NewBytes),
	}, nil
}

// fleetConfigFromStatus rebuilds the personal document the status projection
// describes. The status view exposes source/ref/model/enabled per instance, so
// the editor never re-reads free-form JSON or invents a source reference. The
// caller passes the document's own top-level fleet/preset values so those
// fields survive an edit unchanged.
func fleetConfigFromStatus(status *fleet.Status, fleetName, preset string) (*fleet.Config, error) {
	if status == nil {
		return nil, errors.New("fleet source preview: the engine returned no status")
	}
	cfg := &fleet.Config{
		SchemaVersion: fleet.SchemaVersion,
		Fleet:         fleetName,
		Preset:        preset,
		Instances:     make(map[string]fleet.Instance, len(status.Instances)),
	}
	for _, inst := range status.Instances {
		ref, err := fleet.ParseSourceRef(inst.SourceRef)
		if err != nil {
			return nil, fmt.Errorf("instance %q: %w", inst.InstanceID, err)
		}
		cfg.Instances[inst.InstanceID] = fleet.Instance{
			Source:  ref.String(),
			Model:   inst.ConfiguredModel,
			Enabled: optionalEnabled(inst.Enabled),
		}
	}
	return cfg, nil
}

// optionalEnabled omits the explicit key when an instance is enabled, matching
// the schema's "absent means enabled" default. A disabled instance keeps the
// explicit false that produces a managed removal.
func optionalEnabled(enabled bool) *bool {
	if enabled {
		return nil
	}
	disabled := false
	return &disabled
}

// AppOpsFleetDoctor implements FleetDoctor through appops.NewFleetDoctorRequest
// and appops.FleetDoctor. Only a path the user typed is inspected: it never
// searches PATH, spawns a shell, reads a credential or contacts a network or
// model. The engine's labels are copied verbatim; the screen invents no
// entitlement.
type AppOpsFleetDoctor struct{}

// Diagnose runs one local-only diagnosis. An explicit binary path that is
// missing, unreadable or not a regular file is refused here instead of being
// silently downgraded, and every inspection failure stays in the report.
func (d AppOpsFleetDoctor) Diagnose(ctx context.Context, models []string, opencodeBin, catalogDump string) (FleetDoctorView, error) {
	if err := contextDone(ctx); err != nil {
		return FleetDoctorView{}, err
	}
	bin := strings.TrimSpace(opencodeBin)
	dump := strings.TrimSpace(catalogDump)
	if bin != "" {
		if !filepath.IsAbs(bin) {
			return FleetDoctorView{}, fmt.Errorf("opencode binary %q is not an absolute path; enter the exact local path because this build never searches PATH", bin)
		}
		info, err := os.Stat(bin)
		if err != nil {
			return FleetDoctorView{}, fmt.Errorf("opencode binary is not readable at %s: %v", bin, err)
		}
		if !info.Mode().IsRegular() {
			return FleetDoctorView{}, fmt.Errorf("opencode binary at %s is not a regular file", bin)
		}
	}
	req, err := appops.NewFleetDoctorRequest(models, bin, dump)
	if err != nil {
		return FleetDoctorView{}, err
	}
	return mapFleetDoctorReport(appops.FleetDoctor(req).Report), nil
}

// mapFleetDoctorReport copies the engine's local-evidence report into the
// immutable view. Reasons and failures are sanitized: no control characters, no
// adoption affordance and no credential or payload echo.
func mapFleetDoctorReport(report fleet.DoctorReport) FleetDoctorView {
	view := FleetDoctorView{Available: true, Offline: report.Offline}
	view.ToolsInspected = append([]string(nil), report.ToolsInspected...)
	for _, failure := range report.Failures {
		view.Failures = append(view.Failures, sanitizeFleetReason(sanitizeDoctorText(failure)))
	}
	for _, entry := range report.Entries {
		view.Entries = append(view.Entries, FleetDoctorEntry{
			Model:  entry.Model,
			Label:  doctorLabel(entry.Label),
			Reason: sanitizeFleetReason(sanitizeDoctorText(entry.Reason)),
		})
	}
	return view
}

// sanitizeDoctorText strips control characters from local tool text without
// rewriting the technical cause.
func sanitizeDoctorText(value string) string {
	return redactError(errors.New(value))
}

// AppOpsFleetPlanner implements the read-only deployment and recovery
// projections through appops.FleetPreview, fleet.ListFleetBackups and
// appops.FleetRestorePreview.
type AppOpsFleetPlanner struct {
	Home string
	Repo string
}

// PreviewPlan builds a fresh non-adopting plan and maps every file projection
// into a bounded diff. No target is created or modified.
func (p AppOpsFleetPlanner) PreviewPlan(ctx context.Context, source string) (FleetPlanPreview, error) {
	if err := contextDone(ctx); err != nil {
		return FleetPlanPreview{}, err
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return FleetPlanPreview{Available: false, Message: "Select an explicit fleet.json source before planning."}, nil
	}
	if strings.TrimSpace(p.Home) == "" {
		return FleetPlanPreview{Available: false, SourcePath: source, Message: "Deployment planning needs a resolved home."}, nil
	}
	req, err := appops.NewFleetPreviewRequest(source, p.Repo, p.Home)
	if err != nil {
		return FleetPlanPreview{}, err
	}
	result, err := appops.FleetPreview(req)
	if err != nil {
		return FleetPlanPreview{}, err
	}
	files, lines, counts := mapFleetFileDiffs(result.Files)
	return FleetPlanPreview{
		Available:       true,
		SourcePath:      source,
		PlanID:          result.Plan.PlanID,
		Scope:           result.Scope,
		ConfigDigest:    result.ConfigDigest,
		SourceRev:       result.SourceRev,
		RendererVersion: result.RendererVersion,
		Token:           result.PreviewToken,
		Counts:          counts,
		Files:           files,
		DiffLines:       lines,
		Plan:            result.Plan,
	}, nil
}

// ListBackups lists only backups that carry a format-4 fleet manifest.
func (p AppOpsFleetPlanner) ListBackups(ctx context.Context) ([]FleetBackupView, error) {
	if err := contextDone(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Home) == "" {
		return nil, nil
	}
	entries, err := fleet.ListFleetBackups(p.Home)
	if err != nil {
		return nil, err
	}
	views := make([]FleetBackupView, 0, len(entries))
	for _, entry := range entries {
		view := FleetBackupView{ID: entry.ID, Path: entry.Path}
		if entry.Manifest != nil {
			view.Operation = entry.Manifest.Operation
			view.State = entry.Manifest.State
			view.CreatedAt = entry.Manifest.CreatedAt
			view.Files = len(entry.Manifest.Files)
		}
		views = append(views, view)
	}
	return views, nil
}

// PreviewRestore reports exactly what RestoreBackup would do. Legacy/corrupt
// backups and modified targets are typed refusals, not a misleading preview.
func (p AppOpsFleetPlanner) PreviewRestore(ctx context.Context, backupID string) (FleetRestorePreview, error) {
	if err := contextDone(ctx); err != nil {
		return FleetRestorePreview{}, err
	}
	if strings.TrimSpace(p.Home) == "" {
		return FleetRestorePreview{Available: false, BackupID: backupID, Message: "Restore preview needs a resolved home."}, nil
	}
	req, err := appops.NewFleetRestorePreviewRequest(p.Home, backupID)
	if err != nil {
		return FleetRestorePreview{}, err
	}
	result, err := appops.FleetRestorePreview(req)
	if err != nil {
		return FleetRestorePreview{}, err
	}
	if result.Preview == nil {
		return FleetRestorePreview{Available: false, BackupID: backupID, Message: "The engine returned no restore preview."}, nil
	}
	files, lines, counts := mapFleetFileDiffs(result.Preview.Files)
	if len(result.Preview.Counts) > 0 {
		counts = make(map[string]int, len(result.Preview.Counts))
		for op, n := range result.Preview.Counts {
			counts[op] = n
		}
	}
	return FleetRestorePreview{
		Available:  true,
		BackupID:   result.BackupID,
		BackupPath: result.Preview.BackupPath,
		Counts:     counts,
		Files:      files,
		DiffLines:  lines,
		Token:      result.PreviewToken,
	}, nil
}

// mapFleetFileDiffs copies every engine file projection into a view model with
// bounded line-diff rows. Removals and conflicts stay first because the engine
// already sorts them that way.
func mapFleetFileDiffs(previews []fleet.FilePreview) ([]FleetFileDiff, []string, map[string]int) {
	files := make([]FleetFileDiff, 0, len(previews))
	lines := make([]string, 0, len(previews)*4)
	counts := map[string]int{"total": 0}
	for _, preview := range previews {
		diff := fleetLineDiffLines(preview.OldBytes, preview.NewBytes)
		files = append(files, FleetFileDiff{
			InstanceID: preview.InstanceID,
			Path:       preview.Path,
			Op:         preview.Op,
			Reason:     sanitizeFleetReason(preview.Reason),
			Removed:    preview.Removed,
			Conflict:   preview.Conflict,
			OldDigest:  preview.OldDigest,
			NewDigest:  preview.NewDigest,
			DiffLines:  diff,
		})
		counts["total"]++
		counts[preview.Op]++
		header := fmt.Sprintf("=== %s · %s · %s", preview.Op, preview.InstanceID, preview.Path)
		if preview.Reason != "" {
			header += " · " + sanitizeFleetReason(preview.Reason)
		}
		lines = append(lines, header)
		if len(diff) == 0 {
			lines = append(lines, "  (no line changes)")
		} else {
			lines = append(lines, diff...)
		}
	}
	return files, lines, counts
}

// fleetLineDiffLines renders the engine's bounded line diff with stable ASCII
// prefixes. It preserves Unicode text and marks omitted rows honestly.
func fleetLineDiffLines(oldBytes, newBytes []byte) []string {
	diffs := fleet.LineDiff(oldBytes, newBytes)
	if len(diffs) == 0 {
		return nil
	}
	lines := make([]string, 0, len(diffs))
	for _, line := range diffs {
		switch line.Kind {
		case fleet.DiffAdd:
			lines = append(lines, "+"+line.Text)
		case fleet.DiffDel:
			lines = append(lines, "-"+line.Text)
		case fleet.DiffOmitted:
			lines = append(lines, "…"+line.Text)
		default:
			lines = append(lines, " "+line.Text)
		}
	}
	return lines
}
