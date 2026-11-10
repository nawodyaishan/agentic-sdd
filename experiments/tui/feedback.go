package tui

// feedback.go owns the root shell's truthful write feedback (Feature 013
// B7/T20, FR-10.4).
//
// The spinner is root-owned and exists for exactly one accepted write
// generation:
//
//   - it starts from WriteStartedMsg, never from a key, a render or a timer;
//   - it advances only on a spinner.TickMsg that carries this generation's
//     identity, so a late tick from an earlier write cannot restart it;
//   - it stops on WriteResultMsg, on a refusal that left no write running, and
//     on quit; a stopped or hidden spinner schedules nothing further.
//
// A tick never dispatches an operation, advances a phase or invents a
// percentage. The phase text (`starting`, `writing`, `flushing`) comes only
// from WriteStartedMsg/WriteProgressMsg, which the operation host emits for
// work it really performs. Motion-off, plain, ASCII, color-off, headless and
// non-interactive sessions render a static ASCII marker beside the same phase
// text, so the fallback is never a second, different status.

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// WriteStaticMarker is the non-animated phase marker. It is ASCII and
// single-cell, so it renders identically in Unicode, ASCII and NO_COLOR
// sessions and never implies progress.
const WriteStaticMarker = "*"

// motionEnabled reports whether an indeterminate animation is truthful for this
// session. Motion must be requested, the session interactive, color enabled and
// the glyph set Unicode. Anything else (plain, ASCII, NO_COLOR, headless or an
// unknown suitability) uses the static marker instead.
func (a App) motionEnabled() bool {
	return a.motion && a.cfg.Interactive && a.cfg.Color && !a.ascii
}

// writePhaseText returns the phase text for the active write. It only
// normalizes the zero value to the host's first phase; it never advances one.
func (a App) writePhaseText() string {
	if a.write == nil {
		return ""
	}
	step := a.write.step
	if step == "" {
		step = WriteStepStarting
	}
	return string(step)
}

// writePhaseMarker returns the marker shown beside the phase text: the current
// spinner frame while the animation is truthful, otherwise the static marker.
func (a App) writePhaseMarker() string {
	if a.spinnerActive && a.write != nil {
		if frame := a.spinner.View(); frame != "" {
			return frame
		}
	}
	return WriteStaticMarker
}

// startWriteFeedback begins the spinner for one accepted write generation and
// returns its first tick command, or nil when this session must stay static.
func (a *App) startWriteFeedback() tea.Cmd {
	a.spinnerActive = false
	a.spinnerPending = false
	if !a.motionEnabled() {
		return nil
	}
	model := spinner.New(spinner.WithSpinner(spinner.Line))
	if a.theme.Color {
		model.Style = a.theme.Accent
	}
	a.spinner = model
	a.spinnerActive = true
	a.spinnerPending = true
	return a.spinner.Tick
}

// stopWriteFeedback ends the current generation. A tick already in flight is
// ignored by identity, so stopping never has to drain a timer.
func (a *App) stopWriteFeedback() {
	a.spinnerActive = false
	a.spinnerPending = false
}

// handleWriteTick advances the spinner only for the active write generation.
// A stale identity, a completed write, a hidden busy footer and a quitting
// program drop the tick without scheduling another one.
func (a *App) handleWriteTick(msg spinner.TickMsg) tea.Cmd {
	if !a.spinnerActive || a.write == nil || a.quitting {
		a.stopWriteFeedback()
		return nil
	}
	if msg.ID != a.spinner.ID() {
		// A previous generation's tick must not advance or restart this one.
		return nil
	}
	model, cmd := a.spinner.Update(msg)
	a.spinner = model
	a.spinnerPending = cmd != nil
	return cmd
}
