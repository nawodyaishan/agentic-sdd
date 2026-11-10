package tui

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
)

// EnvNoUpdateCheck is the environment variable that suppresses automatic update checks.
// Only the exact value "1" disables checks; any other value allows them.
const EnvNoUpdateCheck = "AGENTIC_SDD_NO_UPDATE_CHECK"

// UpdateCheckState classifies the lifecycle of the TUI release check.
type UpdateCheckState int

const (
	// UpdateStatusNotStarted means no check has been requested yet this session.
	UpdateStatusNotStarted UpdateCheckState = iota
	// UpdateStatusChecking means an asynchronous check is currently in flight.
	UpdateStatusChecking
	// UpdateStatusCurrent means the installed version is up to date with the latest release.
	UpdateStatusCurrent
	// UpdateStatusNewer means a newer published release is available.
	UpdateStatusNewer
	// UpdateStatusUnavailable means no release comparison could be established (network failure,
	// timeout, rate limit, unparseable dev build, etc.). This is non-fatal.
	UpdateStatusUnavailable
	// UpdateStatusDisabled means checks are disabled by configuration, offline mode, or environment.
	UpdateStatusDisabled
)

// String returns the canonical name for the update check state.
func (s UpdateCheckState) String() string {
	switch s {
	case UpdateStatusNotStarted:
		return "not started"
	case UpdateStatusChecking:
		return "checking"
	case UpdateStatusCurrent:
		return "current"
	case UpdateStatusNewer:
		return "newer"
	case UpdateStatusUnavailable:
		return "unavailable"
	case UpdateStatusDisabled:
		return "disabled"
	default:
		return "unavailable"
	}
}

// CheckReleaseFunc matches the signature of appops.CheckLatestRelease.
type CheckReleaseFunc func(ctx context.Context, client *http.Client, current string) appops.UpdateResult

// UpdateCheckResultMsg is dispatched by the async Bubble Tea command when an update check completes.
// It carries the generation ID to ensure out-of-order or stale results are discarded.
type UpdateCheckResultMsg struct {
	Generation uint64
	Result     appops.UpdateResult
}

// UpdateCheckOption configures an UpdateCheckManager.
type UpdateCheckOption func(*UpdateCheckManager)

// WithDisabled forces update checks to be disabled (e.g. offline mode).
func WithDisabled(disabled bool) UpdateCheckOption {
	return func(m *UpdateCheckManager) {
		m.disabled = disabled
		if disabled {
			m.state = UpdateStatusDisabled
		}
	}
}

// WithCheckFunc injects a custom release check function for testing.
func WithCheckFunc(fn CheckReleaseFunc) UpdateCheckOption {
	return func(m *UpdateCheckManager) {
		m.checkFunc = fn
	}
}

// WithHTTPClient injects a custom HTTP client for testing transport policies and errors.
func WithHTTPClient(client *http.Client) UpdateCheckOption {
	return func(m *UpdateCheckManager) {
		m.client = client
	}
}

// WithTimeout sets a custom timeout for the bounded update check.
func WithTimeout(d time.Duration) UpdateCheckOption {
	return func(m *UpdateCheckManager) {
		if d > 0 {
			m.timeout = d
		}
	}
}

// WithEnvLookup injects an environment lookup function for checking AGENTIC_SDD_NO_UPDATE_CHECK.
func WithEnvLookup(lookup func(string) string) UpdateCheckOption {
	return func(m *UpdateCheckManager) {
		if lookup != nil && lookup(EnvNoUpdateCheck) == "1" {
			m.disabled = true
			m.state = UpdateStatusDisabled
		}
	}
}

// UpdateCheckManager manages the asynchronous, bounded update-check state machine.
// It guarantees exactly one automatic attempt per session across Home revisits and resizes,
// generation-based stale result suppression, cooperative cancellation when disabled or aborted,
// and non-blocking local rendering.
type UpdateCheckManager struct {
	mu         sync.Mutex
	state      UpdateCheckState
	attempted  bool
	disabled   bool
	generation uint64
	cancel     context.CancelFunc
	result     appops.UpdateResult
	current    string
	latest     string
	timeout    time.Duration
	client     *http.Client
	checkFunc  CheckReleaseFunc
}

// NewUpdateCheckManager constructs an UpdateCheckManager with default settings.
// By default, it checks the environment variable AGENTIC_SDD_NO_UPDATE_CHECK=1.
func NewUpdateCheckManager(opts ...UpdateCheckOption) *UpdateCheckManager {
	m := &UpdateCheckManager{
		state:     UpdateStatusNotStarted,
		timeout:   appops.UpdateRequestTimeout,
		checkFunc: appops.CheckLatestRelease,
		client:    &http.Client{Timeout: appops.UpdateRequestTimeout},
	}

	if os.Getenv(EnvNoUpdateCheck) == "1" {
		m.disabled = true
		m.state = UpdateStatusDisabled
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m
}

// State returns the current update check state.
func (m *UpdateCheckManager) State() UpdateCheckState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Result returns the latest received appops.UpdateResult.
func (m *UpdateCheckManager) Result() appops.UpdateResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.result
}

// IsDisabled reports whether update checking is currently disabled.
func (m *UpdateCheckManager) IsDisabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.disabled
}

// Attempted reports whether the automatic once-per-session check has already run.
func (m *UpdateCheckManager) Attempted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attempted
}

// Generation returns the current generation counter.
func (m *UpdateCheckManager) Generation() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generation
}

// Current returns the validated installed version string, if resolved.
func (m *UpdateCheckManager) Current() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// Latest returns the validated published release version string, if resolved.
func (m *UpdateCheckManager) Latest() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.latest
}

// AutomaticCheck triggers the once-per-session automatic update check.
// If checks are disabled or an automatic check has already been attempted this session,
// it returns (nil, false) without dispatching any command.
// Otherwise, it marks the check as attempted, transitions state to checking,
// and returns an asynchronous tea.Cmd.
func (m *UpdateCheckManager) AutomaticCheck(currentVersion string) (tea.Cmd, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.disabled || m.attempted {
		return nil, false
	}
	m.attempted = true
	return m.startCheckLocked(currentVersion), true
}

// RetryCheck triggers an explicit, user-initiated update check retry.
// Unlike AutomaticCheck, it is not blocked by the once-per-session flag.
// If checks are disabled, it returns (nil, false).
// If a check is already in flight, it cancels the previous context, increments
// the generation ID to invalidate pending responses, and dispatches a fresh check.
func (m *UpdateCheckManager) RetryCheck(currentVersion string) (tea.Cmd, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.disabled {
		return nil, false
	}
	return m.startCheckLocked(currentVersion), true
}

func (m *UpdateCheckManager) startCheckLocked(currentVersion string) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	m.generation++
	gen := m.generation
	m.state = UpdateStatusChecking

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	m.cancel = cancel

	client := m.client
	check := m.checkFunc
	if check == nil {
		check = appops.CheckLatestRelease
	}

	return func() tea.Msg {
		res := check(ctx, client, currentVersion)
		return UpdateCheckResultMsg{
			Generation: gen,
			Result:     res,
		}
	}
}

// HandleMsg inspects incoming tea.Msg values. If the message is an UpdateCheckResultMsg,
// it validates the generation ID and updates state. Returns true if handled.
func (m *UpdateCheckManager) HandleMsg(msg tea.Msg) bool {
	resMsg, ok := msg.(UpdateCheckResultMsg)
	if !ok {
		return false
	}
	return m.handleResult(resMsg)
}

func (m *UpdateCheckManager) handleResult(msg UpdateCheckResultMsg) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Discard stale generation results, results received after disabling, or results
	// received when not in the checking state.
	if msg.Generation != m.generation || m.disabled || m.state != UpdateStatusChecking {
		return false
	}

	m.cancel = nil
	m.result = msg.Result
	m.current = msg.Result.Current
	m.latest = msg.Result.Latest

	switch msg.Result.State {
	case appops.UpdateEqual, appops.UpdateAhead:
		m.state = UpdateStatusCurrent
	case appops.UpdateNewer:
		m.state = UpdateStatusNewer
	case appops.UpdateUnavailable, appops.UpdateFailed:
		m.state = UpdateStatusUnavailable
	default:
		m.state = UpdateStatusUnavailable
	}

	return true
}

// Disable disables update checking, cancels any active in-flight request context,
// and increments the generation ID so any pending response is discarded.
func (m *UpdateCheckManager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.generation++
	m.disabled = true
	m.state = UpdateStatusDisabled
}

// Enable re-enables update checking. If the state was disabled, it resets to not started.
func (m *UpdateCheckManager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.disabled = false
	if m.state == UpdateStatusDisabled {
		m.state = UpdateStatusNotStarted
	}
}

// Cancel cooperatively cancels any in-flight check without marking the manager as disabled.
func (m *UpdateCheckManager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.generation++
	if m.state == UpdateStatusChecking {
		m.state = UpdateStatusUnavailable
	}
}

// StatusText returns a human-readable description of the current update status.
func (m *UpdateCheckManager) StatusText() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.state {
	case UpdateStatusNotStarted:
		return "Update check: not started"
	case UpdateStatusChecking:
		return "Checking for updates (GitHub)..."
	case UpdateStatusCurrent:
		if m.current != "" {
			return fmt.Sprintf("Agentic-SDD is up to date (%s)", m.current)
		}
		return "Agentic-SDD is up to date"
	case UpdateStatusNewer:
		if m.latest != "" && m.current != "" {
			return fmt.Sprintf("Update available: %s (installed: %s)", m.latest, m.current)
		} else if m.latest != "" {
			return fmt.Sprintf("Update available: %s", m.latest)
		}
		return "Update available"
	case UpdateStatusUnavailable:
		if m.latest != "" && m.current != "" {
			return fmt.Sprintf("Development version (%s); latest release is %s", m.current, m.latest)
		} else if m.latest != "" {
			return fmt.Sprintf("Development version; latest release is %s", m.latest)
		}
		return "Update check unavailable (offline or rate-limited)"
	case UpdateStatusDisabled:
		return "Update checks disabled"
	default:
		return "Update check unavailable"
	}
}

// BadgeText returns a concise label for headers or summaries.
func (m *UpdateCheckManager) BadgeText() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.state {
	case UpdateStatusNotStarted:
		return "not checked"
	case UpdateStatusChecking:
		return "checking"
	case UpdateStatusCurrent:
		return "up to date"
	case UpdateStatusNewer:
		if m.latest != "" {
			return "update: " + m.latest
		}
		return "update available"
	case UpdateStatusUnavailable:
		return "unavailable"
	case UpdateStatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// RenderStatus renders the status line styled according to the active theme.
// In plain mode (theme.Color == false), it returns unstyled plain text.
func (m *UpdateCheckManager) RenderStatus(theme Theme) string {
	text := m.StatusText()
	if !theme.Color {
		return text
	}

	state := m.State()
	switch state {
	case UpdateStatusChecking:
		return theme.Accent.Render(text)
	case UpdateStatusCurrent:
		return theme.OK.Render(text)
	case UpdateStatusNewer:
		return theme.Warn.Render(text)
	case UpdateStatusUnavailable:
		return theme.Hint.Render(text)
	case UpdateStatusDisabled:
		return theme.Hint.Render(text)
	default:
		return text
	}
}
