package tui

// Fleet Manager opt-in local diagnostics (Feature 015 FR-13 / T20).
//
// The diagnostics panel is contextual: it is not a fifth persistent tab, it is
// reached from Overview or Recovery with `d` and returns with Esc. It is offline
// by default: no PATH search, no shell, no network, no model call and no
// credential read happens unless the user types an explicit local path.
//
// Every label comes from the shared engine's local-evidence vocabulary
// (`configured`, `catalog-visible`, `authenticated`, `unknown`, `unavailable`,
// `not checked (offline)`). The screen never invents one, never upgrades a
// route into an entitlement claim, and never treats `openai/*` as subscription
// proof. The seam is a narrow read-only interface; the production adapter lives
// in fleet_production.go and a fake replaces it in tests.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/fleet"
)

// fleetDoctorOfflineReason mirrors the engine's offline default wording so a
// screen that has not run anything reports exactly what fleet.RunDoctor would.
const fleetDoctorOfflineReason = "offline default: no local tool was offered, so nothing was checked"

// fleetDoctorEvidenceNote is the permanent honesty footer of the panel.
const fleetDoctorEvidenceNote = "Labels are local evidence only: installed is not authenticated, not primary and not runtime-verified. openai/* is a provider route, never subscription proof."

// FleetDoctorEntry is one model route with the engine's exact label.
type FleetDoctorEntry struct {
	Model  string
	Label  fleet.DoctorLabel
	Reason string
}

// FleetDoctorView is the immutable diagnostic report the panel renders.
// Offline reports the no-inspection default; Entries carry one label per
// configured route; ToolsInspected and Failures state exactly which local
// tools were consulted and why any of them produced nothing.
type FleetDoctorView struct {
	Available      bool
	Offline        bool
	Entries        []FleetDoctorEntry
	ToolsInspected []string
	Failures       []string
	Message        string
}

// FleetDoctor is the narrow read-only diagnostics seam. Implementations must
// inspect only the paths they are given: no PATH search, no shell, no network,
// no model call and no credential access.
type FleetDoctor interface {
	Diagnose(ctx context.Context, models []string, opencodeBin, catalogDump string) (FleetDoctorView, error)
}

// DefaultFleetDoctor is the safe default when no production adapter is wired.
// It inspects nothing at all and says so.
type DefaultFleetDoctor struct{}

// Diagnose reports the honest not-available state without touching the
// filesystem, a process or the network.
func (DefaultFleetDoctor) Diagnose(ctx context.Context, models []string, opencodeBin, catalogDump string) (FleetDoctorView, error) {
	if err := contextDone(ctx); err != nil {
		return FleetDoctorView{}, err
	}
	return FleetDoctorView{
		Available: false,
		Offline:   true,
		Message:   "Local diagnostics are not available in this build (doctor adapter pending).",
	}, nil
}

// FleetDoctorMsg delivers one asynchronous local diagnostic result.
type FleetDoctorMsg struct {
	Generation uint64
	Bin        string
	Dump       string
	View       FleetDoctorView
	Err        error
}

// SetDoctor updates the read-only diagnostics seam.
func (m *FleetModel) SetDoctor(doctor FleetDoctor) {
	if doctor == nil {
		doctor = DefaultFleetDoctor{}
	}
	m.doctor = doctor
}

// DoctorOpen reports whether the contextual diagnostics panel is showing.
func (m FleetModel) DoctorOpen() bool { return m.doctorOpen }

// DoctorView returns the current diagnostic report.
func (m FleetModel) DoctorView() FleetDoctorView { return m.doctorView }

// DoctorBinInput returns the explicit local OpenCode binary path the user
// typed. Empty means the offline default.
func (m FleetModel) DoctorBinInput() string { return m.doctorBinInput }

// DoctorDumpInput returns the optional explicit local catalog dump path.
func (m FleetModel) DoctorDumpInput() string { return m.doctorDumpInput }

// DoctorError returns the last refusal text, if any.
func (m FleetModel) DoctorError() string { return m.doctorErr }

// fleetsDoctorModels returns the distinct configured routes in status order.
// It derives labels only; the exact model strings stay untouched.
func (m FleetModel) fleetDoctorModels() []string {
	seen := make(map[string]bool, len(m.status.Instances))
	var models []string
	for _, inst := range m.status.Instances {
		model := strings.TrimSpace(inst.ConfiguredModel)
		if model == "" {
			model = strings.TrimSpace(inst.Model)
		}
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		models = append(models, model)
	}
	return models
}

// offlineFleetDoctorView labels every configured route `not checked (offline)`.
// It performs no I/O and calls no seam, so opening the panel can never search
// PATH or contact anything.
func offlineFleetDoctorView(models []string) FleetDoctorView {
	view := FleetDoctorView{Available: true, Offline: true}
	for _, model := range models {
		view.Entries = append(view.Entries, FleetDoctorEntry{
			Model:  model,
			Label:  fleet.DoctorNotChecked,
			Reason: fleetDoctorOfflineReason,
		})
	}
	return view
}

// openDoctor shows the panel on its offline default. A previous explicit run is
// kept, because silently re-labelling real local evidence would be misleading.
func (m FleetModel) openDoctor() FleetModel {
	m.doctorOpen = true
	m.doctorField = 0
	m.doctorScroll = 0
	if !m.doctorRan {
		m.doctorView = offlineFleetDoctorView(m.fleetDoctorModels())
		m.doctorErr = ""
		m.doctorError = nil
	}
	return m
}

// closeDoctor returns to the view the panel was opened from. It changes no
// report and performs no read.
func (m FleetModel) closeDoctor() FleetModel {
	m.doctorOpen = false
	m.doctorScroll = 0
	return m
}

// handleDoctorKey owns every key while the panel is open, so printable keys are
// literal path text (j, k, q and / included) and nothing leaks to navigation.
func (m FleetModel) handleDoctorKey(msg tea.KeyPressMsg) (FleetModel, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		return m.closeDoctor(), nil
	case "tab", "shift+tab":
		m.doctorField = (m.doctorField + 1) % 2
		return m, nil
	case "enter":
		return m.runDoctor()
	case "backspace":
		m.setDoctorField(trimLastRune(m.doctorFieldValue()))
		return m, nil
	case "ctrl+u":
		m.setDoctorField("")
		return m, nil
	case "up", "pgup":
		m.doctorScroll -= 1
		if m.doctorScroll < 0 {
			m.doctorScroll = 0
		}
		return m, nil
	case "down", "pgdown":
		m.doctorScroll += 1
		return m, nil
	default:
		if len(msg.Text) > 0 {
			m.setDoctorField(m.doctorFieldValue() + msg.Text)
		} else if len(key) == 1 {
			m.setDoctorField(m.doctorFieldValue() + key)
		}
		return m, nil
	}
}

// doctorFieldValue returns the focused field's current text.
func (m FleetModel) doctorFieldValue() string {
	if m.doctorField == 0 {
		return m.doctorBinInput
	}
	return m.doctorDumpInput
}

// setDoctorField replaces the focused field's text.
func (m *FleetModel) setDoctorField(value string) {
	if m.doctorField == 0 {
		m.doctorBinInput = value
		return
	}
	m.doctorDumpInput = value
}

// trimLastRune removes the last UTF-8 rune so backspace never splits a path.
func trimLastRune(value string) string {
	if len(value) == 0 {
		return value
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}

// runDoctor dispatches one read-only diagnosis. With no explicit local path it
// stays offline and calls no seam at all.
func (m FleetModel) runDoctor() (FleetModel, tea.Cmd) {
	bin := strings.TrimSpace(m.doctorBinInput)
	dump := strings.TrimSpace(m.doctorDumpInput)
	if bin == "" && dump == "" {
		m.doctorRan = false
		m.doctorView = offlineFleetDoctorView(m.fleetDoctorModels())
		m.doctorErr = ""
		m.doctorError = nil
		m.doctorScroll = 0
		return m, nil
	}
	if m.doctor == nil {
		m.doctorErr = "Local diagnostics are not available in this build (doctor adapter pending)."
		m.doctorError = errors.New(m.doctorErr)
		m.doctorView = offlineFleetDoctorView(m.fleetDoctorModels())
		return m, nil
	}
	m.doctorGen++
	m.doctorLoading = true
	m.doctorErr = ""
	m.doctorError = nil
	m.doctorScroll = 0
	generation := m.doctorGen
	doctor := m.doctor
	models := m.fleetDoctorModels()
	return m, func() tea.Msg {
		view, err := doctor.Diagnose(context.Background(), models, bin, dump)
		return FleetDoctorMsg{Generation: generation, Bin: bin, Dump: dump, View: view, Err: err}
	}
}

// doctorSummary is the one-line result state under the path fields.
func (m FleetModel) doctorSummary() string {
	switch {
	case m.doctorLoading:
		return "reading the explicit local evidence..."
	case m.doctorErr != "":
		return "refused — see the error detail below"
	case m.doctorView.Offline:
		return fmt.Sprintf("not checked (offline) · %d configured route(s) · no local tool inspected", len(m.doctorView.Entries))
	case len(m.doctorView.ToolsInspected) == 0:
		return fmt.Sprintf("%d configured route(s) · no local tool reported evidence", len(m.doctorView.Entries))
	default:
		return fmt.Sprintf("%d configured route(s) · tools inspected: %s", len(m.doctorView.Entries), strings.Join(m.doctorView.ToolsInspected, ", "))
	}
}

// doctorFieldRow renders one editable path field with its focus marker.
func (m FleetModel) doctorFieldRow(field int, label string, width int) string {
	value := m.doctorBinInput
	if field == 1 {
		value = m.doctorDumpInput
	}
	placeholder := "(not set — offline default)"
	if field == 1 {
		placeholder = "(not set — optional)"
	}
	if strings.TrimSpace(value) == "" {
		value = placeholder
	}
	prefix := "  "
	if m.doctorField == field {
		prefix = "> "
		value += "_"
	}
	line := fmt.Sprintf("%s%s: %s", prefix, label, value)
	if m.doctorField == field && m.theme.Color {
		line = m.theme.Selected.Render(line)
	}
	return truncateCells(line, width)
}

// doctorRows is the complete unpaged report body: labels first, then the
// sanitized failures and the honesty note.
func (m FleetModel) doctorRows(width int) []string {
	var rows []string
	if m.doctorLoading {
		return []string{truncateCells("  Reading the named local tool (read-only, no network)...", width)}
	}
	if m.doctorErr != "" {
		what, unchanged, next := fleetErrorGuidance(m.doctorError)
		rows = append(rows,
			truncateCells("  Error detail: "+what, width),
			truncateCells("  What stayed unchanged: "+unchanged, width),
			truncateCells("  Next safe action: "+next, width),
		)
	}
	if !m.doctorView.Available && m.doctorView.Message != "" {
		rows = append(rows, truncateCells("  "+m.doctorView.Message, width))
	}
	for _, entry := range m.doctorView.Entries {
		line := fmt.Sprintf("  %s · %s", entry.Model, entry.Label)
		if entry.Reason != "" {
			line += " · " + entry.Reason
		}
		rows = append(rows, truncateCells(line, width))
	}
	if len(m.doctorView.Entries) == 0 && m.doctorView.Available && m.doctorErr == "" {
		rows = append(rows, truncateCells("  No configured model route is available to label yet.", width))
	}
	for _, failure := range m.doctorView.Failures {
		rows = append(rows, truncateCells("  problem: "+failure, width))
	}
	rows = append(rows, truncateCells("  "+fleetDoctorEvidenceNote, width))
	return rows
}

// doctorBodyRows windows the report body into the rows the panel can show, with
// a reachability hint instead of silently dropping material lines.
func (m FleetModel) doctorBodyRows(width, budget int) []string {
	rows := m.doctorRows(width)
	if budget < 1 {
		budget = 1
	}
	if len(rows) <= budget {
		return rows
	}
	if budget == 1 {
		return rows[:1]
	}
	visible := budget - 1
	maxScroll := len(rows) - visible
	scroll := m.doctorScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + visible
	out := append([]string(nil), rows[scroll:end]...)
	out = append(out, truncateCells(fmt.Sprintf("  … rows %d-%d of %d · up/down scrolls", scroll+1, end, len(rows)), width))
	return out
}

// doctorLines renders the contextual Diagnostics panel. The panel is not a
// persistent tab: it replaces the current view until Esc returns.
func (m FleetModel) doctorLines(width int) []string {
	var lines []string

	header := "Install & Manage / Fleet Manager · Diagnostics"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, truncateCells(header, width))
	lines = append(lines, truncateCells("Local, opt-in diagnostics · no PATH search · no network or model call · Esc returns to "+m.view.String(), width))
	lines = append(lines, m.doctorFieldRow(0, "OpenCode binary", width))
	lines = append(lines, m.doctorFieldRow(1, "Catalog dump   ", width))
	lines = append(lines, truncateCells("Result: "+m.doctorSummary(), width))

	budget := m.height - 6
	if budget < 1 {
		budget = 1
	}
	lines = append(lines, m.doctorBodyRows(width, budget)...)

	footer := "Esc: back · Tab: next field · Enter: run doctor (offline unless a path is typed) · up/down: scroll · Ctrl+U: clear field"
	if m.theme.Color {
		footer = m.theme.Hint.Render(footer)
	}
	lines = append(lines, truncateCells(footer, width))
	return lines
}

// doctorLabel keeps only the reviewed label vocabulary. An unrecognized engine
// label degrades to `unknown` instead of displaying a new claim.
func doctorLabel(label fleet.DoctorLabel) fleet.DoctorLabel {
	switch label {
	case fleet.DoctorConfigured, fleet.DoctorCatalogVisible, fleet.DoctorAuthenticated,
		fleet.DoctorUnknown, fleet.DoctorUnavailable, fleet.DoctorNotChecked:
		return label
	default:
		return fleet.DoctorUnknown
	}
}
