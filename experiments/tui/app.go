package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
)

// Route is an application destination in the root shell.
type Route int

const (
	RouteHome Route = iota
	RouteInstall
	RouteExplore
	RouteActivity
)

// Focus indicates whether keyboard input targets the navigation sidebar or
// the active content area.
type Focus int

const (
	FocusNav Focus = iota
	FocusContent
	FocusCategories
)

// RouteInfo holds display metadata for one root navigation destination.
type RouteInfo struct {
	Route       Route
	Key         string
	Title       string
	Icon        string
	ASCII       string
	Description string
}

// BrandTickMsg is emitted periodically to advance the branded logo reveal.
type BrandTickMsg struct{}

type appStartMsg struct{}

// App is the root Bubble Tea model for the Feature 013 unified TUI shell.
// It manages navigation across the four primary destinations, responsive
// cell-aware layout breakpoints, session-level motion and symbol settings,
// the single-write operation host and the exit-after-completion policy, and
// delegates to screen components.
type App struct {
	cfg              TUIConfig
	theme            Theme
	catalog          Catalog
	activeRoute      Route
	navIndex         int
	focus            Focus
	brandState       BrandState
	brandTickPending bool
	motion           bool
	ascii            bool
	helpOpen         bool
	contentScroll    int
	routeScroll      [4]int
	detailScroll     int
	helpScroll       int
	catalogModel     CatalogModel
	width            int
	height           int
	routes           []RouteInfo

	// Operation host and exit policy. ops is shared by pointer so every App
	// copy (Update returns values) sees the same single write slot.
	ops         *OpsHost
	cancelReads func()
	write       *writeState
	lastWrite   WriteResultMsg
	hasResult   bool
	exitPending bool
	exitReason  ShutdownReason
	exitSignal  string
	quitting    bool
	notice      string
	initCmds    []tea.Cmd

	// Write feedback (T20). The spinner is root-owned and belongs to exactly
	// one accepted write generation; its identity makes a late tick from an
	// older write harmless. spinnerActive is false in motion-off, plain, ASCII
	// and non-interactive sessions, which render WriteStaticMarker instead.
	spinner        spinner.Model
	spinnerActive  bool
	spinnerPending bool

	// B4/B5 integrated child screens. installSection selects the hub or one
	// of its children; the child models own their own stage machines.
	installSection  installSection
	installHubIndex int
	install         InstallModel
	restore         RestoreModel
	fleet           FleetModel
	mcp             MCPModel
	fleetReader     FleetReader
	fleetEditor     FleetEditor
	fleetPlanner    FleetPlanner

	// Activity & Help plus the once-per-session update check state machine.
	// The model's read generation binds one bounded log read to the live
	// model, so a completion cannot replace newer section, selection or scroll
	// state.
	activity  ActivityModel
	updateMgr *UpdateCheckManager

	// onboarding is the skippable tour while it is open; opener is the
	// injected explicit-only platform opener.
	onboarding *OnboardingModel
	opener     *Opener

	// Local inventory refresh after a completed write. The generation and
	// scope bind one result to the home/source/options that requested it, so a
	// late read cannot overwrite newer state or a different selection.
	inventoryGeneration uint64
	inventoryScope      catalogScope
	inventoryErr        string
	inventoryPending    bool
	catalogLoader       func(TUIConfig) (Catalog, error)
	activeRequest       WriteRequest
	hasActiveRequest    bool
}

// writeState mirrors the accepted write for rendering. The host remains the
// authority for the slot; this copy only reflects messages already processed.
type writeState struct {
	id      appops.OperationID
	kind    WriteKind
	label   string
	started time.Time
	step    WriteStep
	detail  string
}

// NewApp constructs the root application shell with the production operation
// host and no pre-queued write.
// By default, motion is static unless explicitly requested via cfg.Motion.
// No environment variables are inspected here; presentation flags are resolved
// by the launcher or callers.
func NewApp(cfg TUIConfig, catalog Catalog) App {
	return NewAppWithOptions(cfg, catalog, AppOptions{})
}

// AppScreenOptions injects the B4/B5 screen seams. The zero value uses the
// production read-only operations, the environment-aware update manager and the
// real platform opener. Tests and the PTY harness supply fakes so no run reads
// a real home, opens a browser or contacts GitHub.
type AppScreenOptions struct {
	// UpdateManager replaces the session update state machine.
	UpdateManager *UpdateCheckManager
	// InstallPreview replaces the read-only install preview operation.
	InstallPreview PreviewOp
	// RestoreOps replaces the recovery screen's read-only operations.
	RestoreOps RestoreOps
	// Opener replaces the explicit-only platform opener.
	Opener *Opener
	// CatalogRefresh replaces the generation-bound read-only inventory refresh
	// started after a completed write. Nil uses LoadCatalog.
	CatalogRefresh func(TUIConfig) (Catalog, error)
	// FleetReader replaces the Fleet Manager status and inventory reader.
	FleetReader FleetReader
	// FleetEditor replaces the read-only source preview editor.
	FleetEditor FleetEditor
	// FleetPlanner replaces the read-only deployment and recovery projection.
	FleetPlanner FleetPlanner
	// FleetDoctor replaces the opt-in local diagnostics seam.
	FleetDoctor FleetDoctor
}

// NewAppWithScreens constructs the root shell with both the host seams and the
// integrated screen seams. Production callers use NewApp; tests use this to
// keep every screen deterministic.
func NewAppWithScreens(cfg TUIConfig, catalog Catalog, opts AppOptions, screens AppScreenOptions) App {
	app := NewAppWithOptions(cfg, catalog, opts)
	if screens.UpdateManager != nil {
		app.updateMgr = screens.UpdateManager
		app.activity.updateManager = screens.UpdateManager
	}
	if screens.InstallPreview != nil {
		app.install.SetPreviewOp(screens.InstallPreview)
	}
	if screens.RestoreOps.List != nil || screens.RestoreOps.Preview != nil {
		app.restore = NewRestoreModel(app.cfg, screens.RestoreOps)
		app.restore.SetSize(app.childContentWidth(), app.childBodyHeight())
		app.restore.SetTheme(app.theme)
	}
	if screens.Opener != nil {
		app.opener = screens.Opener
	}
	if screens.CatalogRefresh != nil {
		app.catalogLoader = screens.CatalogRefresh
	}
	if screens.FleetReader != nil || screens.FleetEditor != nil || screens.FleetPlanner != nil {
		reader := screens.FleetReader
		if reader == nil {
			reader = app.fleetReader
		}
		editor := screens.FleetEditor
		if editor == nil {
			editor = app.fleetEditor
		}
		planner := screens.FleetPlanner
		if planner == nil {
			planner = app.fleetPlanner
		}
		app.fleetReader = reader
		app.fleetEditor = editor
		app.fleetPlanner = planner
		app.fleet = NewFleetModelWithSeams(app.cfg, reader, editor, planner)
		app.fleet.SetSize(app.childContentWidth(), app.childBodyHeight())
		app.fleet.SetTheme(app.theme)
	}
	if screens.FleetDoctor != nil {
		app.fleet.SetDoctor(screens.FleetDoctor)
	}
	app.mcp = NewMCPModel(app.cfg, nil)
	app.mcp.SetSize(app.childContentWidth(), app.childBodyHeight())
	app.mcp.SetTheme(app.theme)
	return app
}

// newApp constructs the shell without operation-host seams.
func newApp(cfg TUIConfig, catalog Catalog) App {
	cfg = cfg.Normalize()
	theme := newThemeForConfig(cfg)
	motion := cfg.Motion && cfg.Interactive
	ascii := cfg.ASCII

	brandState := NewBrandState(motion)
	catModel := NewCatalogModel(cfg, catalog)
	catModel.SetShowHelp(false) // Single contextual footer is owned by App

	routes := []RouteInfo{
		{
			Route:       RouteHome,
			Key:         "1",
			Title:       "Home",
			Icon:        "1",
			ASCII:       "1",
			Description: "Dashboard & welcome",
		},
		{
			Route:       RouteInstall,
			Key:         "2",
			Title:       "Install & Manage",
			Icon:        "2",
			ASCII:       "2",
			Description: "Configure & preview",
		},
		{
			Route:       RouteExplore,
			Key:         "3",
			Title:       "Explore",
			Icon:        "3",
			ASCII:       "3",
			Description: "Browse skills & profiles",
		},
		{
			Route:       RouteActivity,
			Key:         "4",
			Title:       "Activity & Help",
			Icon:        "4",
			ASCII:       "4",
			Description: "Guides, logs & reference",
		},
	}

	updateMgr := NewUpdateCheckManager()
	logHome := cfg.Home
	if logHome == "" {
		logHome = catalog.Home
	}

	// Production fleet seams are read-only adapters over appops. They perform
	// no read at construction: the engine is reached only when Fleet is opened
	// and an explicit source is selected. A session without a resolved home
	// keeps the truthful not-available defaults.
	fleetReader := FleetReader(DefaultFleetReader{})
	fleetEditor := FleetEditor(DefaultFleetEditor{})
	fleetPlanner := FleetPlanner(DefaultFleetPlanner{})
	if strings.TrimSpace(cfg.Home) != "" {
		fleetReader = AppOpsFleetReader{Home: cfg.Home, Repo: cfg.Repo}
		fleetEditor = AppOpsFleetEditor{Home: cfg.Home, Repo: cfg.Repo}
		fleetPlanner = AppOpsFleetPlanner{Home: cfg.Home, Repo: cfg.Repo}
	}

	app := App{
		cfg:          cfg,
		theme:        theme,
		catalog:      catalog,
		activeRoute:  RouteHome,
		navIndex:     0,
		focus:        FocusNav,
		brandState:   brandState,
		motion:       motion,
		ascii:        ascii,
		catalogModel: catModel,
		width:        cfg.Width,
		height:       cfg.Height,
		routes:       routes,
		install:      NewInstallModel(cfg),
		restore:      NewRestoreModel(cfg, RestoreOps{}),
		fleetReader:  fleetReader,
		fleetEditor:  fleetEditor,
		fleetPlanner: fleetPlanner,
		fleet:        NewFleetModelWithSeams(cfg, fleetReader, fleetEditor, fleetPlanner),
		activity: NewActivityModel(ActivityConfig{
			Theme:         theme,
			Width:         cfg.Width,
			Height:        cfg.Height,
			LogDir:        homeLogDir(logHome),
			UpdateManager: updateMgr,
		}),
		updateMgr:     updateMgr,
		opener:        NewOpener(),
		catalogLoader: LoadCatalog,
	}
	app.inventoryScope = catalogScope{
		Home:              catalog.Home,
		Repo:              catalog.Repo,
		WithOpenCode:      cfg.WithOpenCode,
		OpenCodeFreeFirst: cfg.OpenCodeFreeFirst,
	}
	if app.inventoryScope.Home == "" {
		app.inventoryScope.Home = cfg.Home
	}

	// Child screens and the local Activity view share the same body budget as
	// the root viewport, so wrapping and paging agree on every row.
	app.install.SetSize(app.childContentWidth(), app.childBodyHeight())
	app.restore.SetSize(app.childContentWidth(), app.childBodyHeight())
	app.fleet.SetSize(app.childContentWidth(), app.childBodyHeight())
	app.mcp = NewMCPModel(cfg, nil)
	app.mcp.SetSize(app.childContentWidth(), app.childBodyHeight())
	// The diagnostics adapter inspects only a path the user types; it needs no
	// home and performs no read until the panel runs an explicit request.
	app.fleet.SetDoctor(AppOpsFleetDoctor{})
	app.activity.SetSize(app.childContentWidth(), app.childBodyHeight())
	app.applyTheme(theme)
	return app
}

// applyTheme updates the root theme and propagates it to all child screens
// without reconstructing models or resetting state.
func (a *App) applyTheme(theme Theme) {
	a.theme = theme
	a.install.SetTheme(theme)
	a.restore.SetTheme(theme)
	a.fleet.SetTheme(theme)
	a.mcp.SetTheme(theme)
	a.activity.SetTheme(theme)
	a.catalogModel.SetTheme(theme)
	if a.onboarding != nil {
		a.onboarding.SetTheme(theme)
	}
}

// ActiveRoute returns the currently displayed destination.
func (a App) ActiveRoute() Route { return a.activeRoute }

// NavIndex returns the highlighted index in the navigation sidebar.
func (a App) NavIndex() int { return a.navIndex }

// Focus returns the current input focus (Nav or Content).
func (a App) Focus() Focus { return a.focus }

// Routes returns the list of supported root destinations.
func (a App) Routes() []RouteInfo { return a.routes }

// Motion reports whether animation is enabled.
func (a App) Motion() bool { return a.motion }

// ASCII reports whether ASCII-only border and symbol rendering is enabled.
func (a App) ASCII() bool { return a.ascii }

// HelpOpen reports whether the contextual help overlay is active.
func (a App) HelpOpen() bool { return a.helpOpen }

// BrandDone reports whether the branding animation has completed or is static.
func (a App) BrandDone() bool { return a.brandState.Done() }

// Ops returns the shared single-write operation host.
func (a App) Ops() *OpsHost { return a.ops }

// Busy reports whether a write transaction is in flight. While Busy is true,
// q, Esc and navigation are ignored and only a shutdown request is accepted.
func (a App) Busy() bool { return a.write != nil }

// WriteStatus returns a display snapshot of the active write, if any.
func (a App) WriteStatus() (WriteStatusView, bool) {
	if a.write == nil {
		return WriteStatusView{}, false
	}
	return WriteStatusView{
		ID:        a.write.id,
		Kind:      a.write.kind,
		Label:     a.write.label,
		StartedAt: a.write.started,
		Step:      a.write.step,
		Detail:    a.write.detail,
	}, true
}

// ExitPending reports whether an exit has been requested and is waiting for
// the active write and its log flush to finish.
func (a App) ExitPending() bool { return a.exitPending }

// ExitRequest returns the reason and optional signal name of the pending exit.
func (a App) ExitRequest() (ShutdownReason, string) { return a.exitReason, a.exitSignal }

// LastWriteResult returns the last result accepted for the current generation.
func (a App) LastWriteResult() (WriteResultMsg, bool) { return a.lastWrite, a.hasResult }

// Notice returns the latest execution notice, such as a refused duplicate
// write. Screens may display it; it never authorizes a write.
func (a App) Notice() string { return a.notice }

// CatalogSelected returns the item currently selected in Explore.
func (a App) CatalogSelected() (CatalogItem, bool) { return a.catalogModel.Selected() }

// VisibleCatalogItems returns the list rows currently visible.
func (a App) VisibleCatalogItems() []CatalogItem { return a.catalogModel.VisibleItems() }

// IsFiltering reports whether Explore is currently in filter text-input mode.
func (a App) IsFiltering() bool { return a.catalogModel.IsFiltering() }

// Init requests the terminal window size, emits an initial startup message,
// requests terminal background color in interactive colored sessions,
// starts the once-per-session Home update check and runs any write already
// queued before the program started. Local content renders before the check
// result arrives.
func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.RequestWindowSize, func() tea.Msg { return appStartMsg{} }}
	if a.cfg.Interactive && a.cfg.Color {
		cmds = append(cmds, tea.RequestBackgroundColor)
	}
	if cmd := a.automaticUpdateCheckCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	cmds = append(cmds, a.initCmds...)
	return tea.Batch(cmds...)
}

func tickBrand() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return BrandTickMsg{}
	})
}

// Update handles incoming messages: keys, window size, and brand animation ticks.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		// Feature 013 B10 (T29): single agentic-dark identity; background-color
		// messages must not switch the scheme, change consent/route, or reset selection/filter/scroll.
		return a, nil

	case tea.ColorProfileMsg:
		// Bubble Tea reports the detected terminal profile at startup. Resolve
		// the explicit reference mapping for the reported profile instead of
		// leaving every session on the TrueColor token values. A non-color or
		// unknown profile keeps the existing theme, and no route, consent,
		// selection, filter or scroll state is touched.
		if !a.cfg.Color {
			return a, nil
		}
		switch msg.Profile {
		case colorprofile.ANSI, colorprofile.ANSI256, colorprofile.TrueColor:
		default:
			// Unknown, no-TTY or ASCII report does not override the explicit
			// identity; color-off is controlled by cfg.Color.
			return a, nil
		}
		profile := referenceProfile(msg.Profile)
		if profile == a.theme.Profile {
			return a, nil
		}
		a.cfg.Profile = profile
		a.applyTheme(NewThemeWithProfile(true, true, profile))
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.catalogModel.SetSize(a.contentWidth(), a.contentHeight())
		a.install.SetSize(a.childContentWidth(), a.childBodyHeight())
		a.restore.SetSize(a.childContentWidth(), a.childBodyHeight())
		a.fleet.SetSize(a.childContentWidth(), a.childBodyHeight())
		a.mcp.SetSize(a.childContentWidth(), a.childBodyHeight())
		a.activity.SetSize(a.childContentWidth(), a.childBodyHeight())
		if a.onboarding != nil {
			a.onboarding.SetSize(a.childContentWidth(), a.childBodyHeight())
		}
		return a, nil

	case appStartMsg:
		if a.activeRoute == RouteHome && a.motion && !a.brandState.Done() && !a.helpOpen && a.height >= 14 && !a.brandTickPending {
			a.brandTickPending = true
			return a, tickBrand()
		}
		return a, nil

	case BrandTickMsg:
		a.brandTickPending = false
		if a.activeRoute == RouteHome && a.motion && !a.brandState.Done() && !a.helpOpen && a.height >= 14 {
			a.brandState = a.brandState.Advance()
			if !a.brandState.Done() {
				a.brandTickPending = true
				return a, tickBrand()
			}
		}
		return a, nil

	case ShutdownRequestMsg:
		return a, a.requestShutdown(msg)

	case StartWriteMsg:
		if a.write != nil {
			return a, refusedWriteCmd(msg.Request.Kind, fmt.Errorf("%w: %s is still running", ErrWriteInProgress, a.write.kind))
		}
		if a.ops == nil {
			return a, refusedWriteCmd(msg.Request.Kind, fmt.Errorf("%w: the operation host is unavailable", ErrWriteInvalid))
		}
		started, cmd, err := a.ops.BeginWrite(msg.Request)
		if err != nil {
			return a, refusedWriteCmd(msg.Request.Kind, err)
		}
		a.activeRequest = msg.Request
		a.hasActiveRequest = true
		a.acceptStarted(started)
		return a, tea.Batch(func() tea.Msg { return started }, cmd)

	case WriteStartedMsg:
		// A late start for an already-completed transaction must not resurrect
		// a busy state, and a start for a newer identity cannot claim this slot.
		if a.write == nil || a.write.id != msg.ID {
			return a, nil
		}
		a.write.kind = msg.Kind
		a.write.label = msg.Label
		a.write.started = msg.StartedAt
		// The phase spinner starts with the accepted write, not with a key or a
		// render, and its first tick is returned beside the screen's command.
		return a, tea.Batch(a.routeWriteMsg(msg.Kind, msg), a.startWriteFeedback())

	case spinner.TickMsg:
		return a, a.handleWriteTick(msg)

	case WriteProgressMsg:
		if a.write == nil || a.write.id != msg.ID {
			return a, nil
		}
		a.write.step = msg.Step
		a.write.detail = msg.Detail
		return a, a.routeWriteMsg(msg.Kind, msg)

	case WriteResultMsg:
		if a.write == nil || a.write.id != msg.ID {
			// Stale results never clear a newer transaction and never authorize
			// another write.
			return a, nil
		}
		kind := a.write.kind
		a.lastWrite = msg
		a.hasResult = true
		a.write = nil
		a.stopWriteFeedback()
		if a.ops != nil {
			a.ops.Complete(msg.ID)
		}
		summary := SummarizeWriteResult(msg)
		a.notice = summary.Headline + ": " + summary.Detail
		cmd := a.routeWriteMsg(kind, msg)
		if a.exitPending {
			// The result is emitted only after the operation log is flushed and
			// closed, so the deferred exit can run now without losing it. No
			// inventory read is queued for a session that is already exiting.
			return a, tea.Batch(cmd, a.beginQuit())
		}
		if refresh := a.startInventoryRefresh(); refresh != nil {
			cmd = tea.Batch(cmd, refresh)
		}
		return a, cmd

	case WriteRefusedMsg:
		a.notice = msg.Reason
		if a.write != nil {
			// A refusal for a different screen must not reset the screen that
			// owns the running transaction.
			return a, nil
		}
		// No write was accepted, so nothing may animate.
		a.stopWriteFeedback()
		cmd := a.routeWriteMsg(msg.Kind, msg)
		if a.exitPending && !a.childWritePending() {
			// A deferred Ctrl+C found only a refused request: nothing is left to
			// wait for, so the idle exit can run now.
			return a, tea.Batch(cmd, a.beginQuit())
		}
		return a, cmd

	case CatalogRefreshMsg:
		return a, a.handleCatalogRefresh(msg)

	case UpdateCheckResultMsg:
		// One shared manager owns the generation check, so Home and Activity
		// always agree and stale results are discarded there.
		if a.updateMgr != nil {
			a.updateMgr.HandleMsg(msg)
		}
		return a, nil

	case OpenLinkMsg:
		// Opening is explicit only: the message arrives from a key press, never
		// from rendering or startup.
		return a, a.openLinkCmd(msg.Link)

	case OpenLinkResultMsg:
		a.notice = msg.Result.Explanation()
		return a, nil

	case ActivityLogReadMsg:
		return a, a.handleActivityRead(msg)

	case InstallPreviewMsg:
		previous := a.install.Step()
		updated, cmd := a.install.Update(msg)
		if model, ok := updated.(InstallModel); ok {
			a.install = model
		}
		if a.install.Step() != previous {
			a.contentScroll = 0
		}
		// A successful zero-change preview never reaches the write host, so it
		// must clear a stale inventory read error the same way a completed
		// write does: one read-only, scope-bound refresh with no write.
		if msg.Err == nil && !msg.OpID.StaleFor(a.install.opID) && a.install.PreviewNoOp() {
			if refresh := a.startNoOpPreviewRefresh(); refresh != nil {
				cmd = tea.Batch(cmd, refresh)
			}
		}
		return a, cmd

	case InstallInspectMsg:
		previous := a.install.Step()
		updated, cmd := a.install.Update(msg)
		if model, ok := updated.(InstallModel); ok {
			a.install = model
		}
		if a.install.Step() != previous {
			a.contentScroll = 0
		}
		return a, cmd

	case RestoreBackupsMsg, RestorePreviewMsg, RestoreRevalidatedMsg, RestoreManifestMsg:
		previous := a.restore.Stage()
		model, cmd := a.restore.Update(msg)
		a.restore = model
		if a.restore.Stage() != previous {
			a.contentScroll = 0
		}
		return a, cmd

	case FleetStatusMsg, FleetSourcePreviewMsg, FleetPlanPreviewMsg, FleetBackupsMsg, FleetRestorePreviewMsg, FleetDoctorMsg:
		updated, cmd := a.fleet.Update(msg)
		a.fleet = updated
		return a, cmd

	case MCPStatusMsg, MCPPlanPreviewMsg, MCPDoctorMsg, MCPBackupListMsg, MCPRestorePreviewMsg:
		updated, cmd := a.mcp.Update(msg)
		a.mcp = updated
		return a, cmd

	case tea.KeyPressMsg:
		a.clampScrollOffsets()
		return a.handleKey(msg)

	default:
		// Forward background commands (e.g. a list filter match) to catalogModel.
		// Spinner ticks have their own case above, so a tick can never fall
		// through to a screen model.
		var cmd tea.Cmd
		updated, cmd := a.catalogModel.Update(msg)
		if cat, ok := updated.(CatalogModel); ok {
			a.catalogModel = cat
		}
		return a, cmd
	}
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// A write transaction owns the shell until its result arrives. q, Esc,
	// navigation, help, settings and any confirmation key cannot abandon or
	// repeat it; Ctrl+C (and an OS signal elsewhere) only records that the
	// process should exit after the write and its log are complete.
	if a.write != nil || a.childWritePending() {
		if msg.String() == "ctrl+c" {
			return *a, a.requestShutdown(ShutdownRequestMsg{Reason: ShutdownReasonInterruptKey})
		}
		a.notice = "write in progress: q, Esc and navigation are ignored; Ctrl+C exits after completion"
		return *a, nil
	}

	// Filter text input precedence: Explore's search input must consume all
	// printable characters (including q, j, k, m, u, 1-4).
	if a.activeRoute == RouteExplore && a.catalogModel.IsFiltering() {
		var cmd tea.Cmd
		updated, cmd := a.catalogModel.Update(msg)
		if cat, ok := updated.(CatalogModel); ok {
			a.catalogModel = cat
		}
		return *a, cmd
	}
	if a.activeRoute == RouteInstall && a.installSection == installSectionFleet && a.fleet.IsInputFocused() {
		return a.handleInstallRouteKey(msg)
	}
	if a.activeRoute == RouteInstall && a.installSection == installSectionMCP && a.mcp.IsInputFocused() {
		return a.handleInstallRouteKey(msg)
	}

	key := msg.String()

	// Any user input immediately settles brand reveal animation.
	if !a.brandState.Done() {
		a.brandState = NewBrandState(false)
		a.brandTickPending = false
	}

	if a.helpOpen {
		switch key {
		case "?", "esc", "enter":
			a.helpOpen = false
			a.helpScroll = 0
			return *a, nil
		case "ctrl+c":
			return *a, a.requestShutdown(ShutdownRequestMsg{Reason: ShutdownReasonInterruptKey})
		case "m":
			a.motion = !a.motion
			if !a.motion {
				a.brandState = NewBrandState(false)
				a.brandTickPending = false
			}
			return *a, nil
		case "u":
			a.ascii = !a.ascii
			return *a, nil
		case "d":
			a.toggleUpdateChecks()
			return *a, nil
		case "pgdown", "pagedown", "ctrl+d", "down", "j":
			a.helpScroll += 4
			return *a, nil
		case "pgup", "pageup", "ctrl+u", "up", "k":
			a.helpScroll -= 4
			if a.helpScroll < 0 {
				a.helpScroll = 0
			}
			return *a, nil
		}
		// q never quits through a modal; Esc, ? or Enter close help.
		return *a, nil
	}

	// An active install/restore confirmation is the only authoritative surface
	// for its decision. Route navigation, settings and Tab cannot reach the
	// inactive backdrop; only the shared confirmation vocabulary is forwarded.
	if kind, ok := a.activeModalKind(); ok && kind != modalKindHelp {
		return a.handleConfirmationModalKey(msg)
	}

	// The skippable onboarding overlay owns navigation keys; only help and the
	// global quit keys stay above it.
	if a.onboarding != nil {
		return a.handleOnboardingKey(msg)
	}

	// Fleet Manager owns a few printable keys that collide with global shell
	// toggles. Route them to the Fleet model before the global switch so the
	// deployment review, model picker and view tabs are reachable. Route jumps
	// (1-4), quit, help and Tab keep their global meaning.
	if a.activeRoute == RouteInstall && a.installSection == installSectionFleet && !a.fleet.IsInputFocused() {
		switch key {
		case "c", "C":
			return a.handleInstallRouteKey(msg)
		case "m", "M":
			if a.fleet.ActiveView() == FleetViewProfiles {
				return a.handleInstallRouteKey(msg)
			}
		}
	}

	// A backup-details action owns 'm' in its own context, so the raw manifest
	// view is reachable exactly where it is advertised. Every other screen
	// keeps the global motion toggle below.
	if (key == "m" || key == "M") && a.backupDetailsActive() {
		previous := a.restore.Stage()
		model, cmd := a.restore.Update(msg)
		a.restore = model
		if a.restore.Stage() != previous {
			a.contentScroll = 0
		}
		return *a, cmd
	}

	switch key {
	case "ctrl+c":
		return *a, a.requestShutdown(ShutdownRequestMsg{Reason: ShutdownReasonInterruptKey})
	case "q":
		return *a, a.requestShutdown(ShutdownRequestMsg{Reason: ShutdownReasonQuitKey})
	case "?":
		a.helpOpen = true
		a.helpScroll = 0
		return *a, nil
	case "tab", "shift+tab":
		a.cycleFocus(key == "shift+tab")
		return *a, nil
	case "m":
		a.motion = !a.motion
		if !a.motion {
			a.brandState = NewBrandState(false)
			a.brandTickPending = false
		} else if a.activeRoute == RouteHome && !a.brandState.Done() && !a.brandTickPending && a.height >= 14 {
			a.brandTickPending = true
			return *a, tickBrand()
		}
		return *a, nil
	case "u":
		a.ascii = !a.ascii
		return *a, nil
	case "c":
		if a.activeRoute == RouteExplore {
			a.focus = FocusCategories
		}
		return *a, nil
	case "[", "]":
		if a.activeRoute == RouteExplore {
			direction := 1
			if key == "[" {
				direction = -1
			}
			cmd := a.cycleCategory(direction)
			return *a, cmd
		}
	case "1":
		return a.switchRoute(0)
	case "2":
		return a.switchRoute(1)
	case "3":
		return a.switchRoute(2)
	case "4":
		return a.switchRoute(3)
	case "pgdown", "pagedown", "ctrl+d":
		if a.activeRoute == RouteExplore {
			a.detailScroll += 4
		} else {
			a.contentScroll += 4
		}
		return *a, nil
	case "pgup", "pageup", "ctrl+u":
		if a.activeRoute == RouteExplore {
			a.detailScroll -= 4
			if a.detailScroll < 0 {
				a.detailScroll = 0
			}
		} else {
			a.contentScroll -= 4
			if a.contentScroll < 0 {
				a.contentScroll = 0
			}
		}
		return *a, nil
	}

	// Home's discovery controls work from either focus pane, so they are not
	// hidden behind Tab. They never collide with navigation or route jumps.
	if a.activeRoute == RouteHome {
		switch key {
		case "enter":
			// The default Home selection is Home itself; Enter is the primary
			// setup/manage action there instead of a no-op re-entry. With
			// another navigation row selected, Enter still opens that route.
			if a.focus == FocusNav && a.navIndex == 0 {
				return a.switchRoute(1)
			}
		case "s":
			return a.switchRoute(1)
		case "o":
			cmd := a.openOnboardingCmd()
			if a.onboarding != nil {
				a.onboarding.SetTheme(a.theme)
			}
			return *a, cmd
		case "b", "R":
			return *a, a.OpenRecovery()
		case "g":
			return *a, a.openLinkCmd(LinkReadme)
		case "a":
			return *a, a.openLinkCmd(LinkAuthor)
		case "r":
			return *a, a.retryUpdateCheckCmd()
		case "d":
			a.toggleUpdateChecks()
			return *a, nil
		}
	}

	// Activity's read-only controls are available from either focus pane, so
	// the update/log/link actions are not hidden behind Tab.
	if a.activeRoute == RouteActivity {
		switch key {
		case "r":
			cmds := make([]tea.Cmd, 0, 2)
			if cmd := a.retryUpdateCheckCmd(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			cmds = append(cmds, a.startActivityRead(a.activity.selectedLog))
			return *a, tea.Batch(cmds...)
		case "d":
			a.toggleUpdateChecks()
			return *a, nil
		case "g":
			return *a, a.openLinkCmd(LinkReadme)
		case "a":
			return *a, a.openLinkCmd(LinkAuthor)
		}
	}

	// The hub's explicit shortcuts work from navigation focus too.
	if a.activeRoute == RouteInstall && a.installSection == installSectionHub {
		switch key {
		case "i":
			return *a, a.openInstallScreen()
		case "b":
			return *a, a.openBackupsScreen()
		case "f":
			return *a, a.openFleetScreen()
		}
	}

	if a.focus == FocusCategories {
		switch key {
		case "up", "k":
			cmd := a.cycleCategory(-1)
			return *a, cmd
		case "down", "j":
			cmd := a.cycleCategory(1)
			return *a, cmd
		case "enter":
			a.focus = FocusContent
		case "esc":
			a.focus = FocusNav
		}
		return *a, nil
	}
	if a.focus == FocusNav {
		switch key {
		case "up", "k":
			a.navIndex = (a.navIndex - 1 + len(a.routes)) % len(a.routes)
			return *a, nil
		case "down", "j":
			a.navIndex = (a.navIndex + 1) % len(a.routes)
			return *a, nil
		case "home":
			a.navIndex = 0
			return *a, nil
		case "end":
			a.navIndex = len(a.routes) - 1
			return *a, nil
		case "enter":
			return a.switchRoute(a.navIndex)
		}
		return *a, nil
	}

	// Focus is in Content
	switch a.activeRoute {
	case RouteExplore:
		if key == "esc" {
			if a.catalogModel.InDetail() {
				a.catalogModel.SetDetail(false)
				a.detailScroll = 0
				return *a, nil
			}
			a.focus = FocusNav
			return *a, nil
		}
		if a.catalogModel.InDetail() && (key == "j" || key == "down") {
			a.detailScroll++
			return *a, nil
		}
		if a.catalogModel.InDetail() && (key == "k" || key == "up") {
			if a.detailScroll > 0 {
				a.detailScroll--
			}
			return *a, nil
		}
		// Reset detail scroll on list movement
		if key == "j" || key == "k" || key == "up" || key == "down" || key == "home" || key == "end" {
			a.detailScroll = 0
		}
		var cmd tea.Cmd
		updated, cmd := a.catalogModel.Update(msg)
		if cat, ok := updated.(CatalogModel); ok {
			a.catalogModel = cat
		}
		return *a, cmd

	case RouteHome:
		switch key {
		case "enter":
			// Primary action on Home: open the Install & Manage hub.
			return a.switchRoute(1)
		case "r", "R":
			// Direct recovery navigation from Home
			return *a, a.OpenRecovery()
		case "j", "down":
			a.contentScroll++
			return *a, nil
		case "k", "up":
			if a.contentScroll > 0 {
				a.contentScroll--
			}
			return *a, nil
		case "esc":
			a.focus = FocusNav
			return *a, nil
		}

	case RouteInstall:
		return a.handleInstallRouteKey(msg)

	case RouteActivity:
		return a.handleActivityRouteKey(msg)
	}

	return *a, nil
}

// handleConfirmationModalKey dispatches one key while an install or restore
// confirmation owns focus. Only the shared confirmation vocabulary is
// forwarded; everything else is ignored and can never reach the backdrop.
//
// Below the usable geometry the viewport renders only resize guidance, so the
// confirmation accepts safe cancellation and shutdown and nothing else: an
// affirmative key can never submit a write the user cannot see or review.
func (a *App) handleConfirmationModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	action := classifyConfirmKey(key)
	if action == confirmKeyShutdown {
		return *a, a.requestShutdown(ShutdownRequestMsg{Reason: ShutdownReasonInterruptKey})
	}
	if !a.usableGeometry() && action != confirmKeyCancel {
		return *a, nil
	}
	switch action {
	case confirmKeyHelp:
		a.helpOpen = true
		a.helpScroll = 0
		return *a, nil
	case confirmKeyScroll:
		a.scrollChildContent(key, a.childContentWidth(), a.childBodyHeight())
		return *a, nil
	case confirmKeyIgnore:
		return *a, nil
	}

	var cmd tea.Cmd
	switch {
	case a.installSection == installSectionInstall && a.install.Step() == InstallStepConfirm:
		previous := a.install.Step()
		updated, c := a.install.Update(msg)
		if model, ok := updated.(InstallModel); ok {
			a.install = model
		}
		if a.install.Step() != previous {
			a.contentScroll = 0
		}
		cmd = c
	case a.installSection == installSectionBackups && a.restore.Stage() == RestoreStageConfirm:
		previous := a.restore.Stage()
		model, c := a.restore.Update(msg)
		a.restore = model
		if a.restore.Stage() != previous {
			a.contentScroll = 0
		}
		cmd = c
	}
	return *a, cmd
}

// cycleFocus moves through the visible interactive regions. Explore has the
// navigation, category control and content regions; every other route has
// navigation and content. shift+tab reverses the same order.
func (a *App) cycleFocus(reverse bool) {
	if a.activeRoute == RouteExplore {
		if reverse {
			switch a.focus {
			case FocusNav:
				a.focus = FocusContent
			case FocusContent:
				a.focus = FocusCategories
			default:
				a.focus = FocusNav
			}
			return
		}
		switch a.focus {
		case FocusNav:
			a.focus = FocusCategories
		case FocusCategories:
			a.focus = FocusContent
		default:
			a.focus = FocusNav
		}
		return
	}
	if a.focus == FocusNav {
		a.focus = FocusContent
		return
	}
	a.focus = FocusNav
}

// routeWriteMsg delivers one typed write message to the screen that owns the
// transaction. The screen identity comes from the host-assigned WriteKind, so
// a result can only reach its originating journey.
func (a *App) routeWriteMsg(kind WriteKind, msg tea.Msg) tea.Cmd {
	switch kind {
	case WriteInstall:
		previous := a.install.Step()
		updated, cmd := a.install.Update(msg)
		if model, ok := updated.(InstallModel); ok {
			a.install = model
		}
		if a.install.Step() != previous {
			a.contentScroll = 0
		}
		return cmd
	case WriteRestore:
		previous := a.restore.Stage()
		model, cmd := a.restore.Update(msg)
		a.restore = model
		if a.restore.Stage() != previous {
			a.contentScroll = 0
		}
		return cmd
	case WriteFleetApply, WriteFleetRestore, WriteSourceSave:
		updated, cmd := a.fleet.Update(msg)
		a.fleet = updated
		return cmd
	case WriteMCPApply, WriteMCPRestore:
		updated, cmd := a.mcp.Update(msg)
		a.mcp = updated
		return cmd
	default:
		return nil
	}
}

// acceptStarted records the typed start message in the display mirror. The
// host already owns the slot, so this cannot fail and cannot create a second
// transaction.
func (a *App) acceptStarted(msg WriteStartedMsg) {
	a.write = &writeState{
		id:      msg.ID,
		kind:    msg.Kind,
		label:   msg.Label,
		started: msg.StartedAt,
		step:    WriteStepStarting,
	}
}

// refusedWriteCmd delivers a refusal as a typed message instead of starting
// work. Update stays free of blocking I/O.
func refusedWriteCmd(kind WriteKind, err error) tea.Cmd {
	return func() tea.Msg {
		reason := "write refused"
		if err != nil {
			reason = err.Error()
		}
		return WriteRefusedMsg{Kind: kind, Err: err, Reason: reason}
	}
}

// requestShutdown routes one key or OS signal through the single policy. While
// a write is active the request is recorded and the process keeps running;
// while idle, cooperative read work is cancelled and the program quits.
func (a *App) requestShutdown(msg ShutdownRequestMsg) tea.Cmd {
	if a.quitting {
		return nil
	}
	a.exitReason = msg.Reason
	if msg.Signal != "" {
		a.exitSignal = msg.Signal
	}
	if a.write != nil || a.childWritePending() {
		// Repeated signals and Ctrl+C are idempotent: they never hard-kill the
		// uncancellable engine transaction, and a confirmed child that has not
		// yet reached the host still gets to start and finish.
		a.exitPending = true
		return nil
	}
	return a.beginQuit()
}

// beginQuit performs the idle shutdown once: cancel cooperative read work and
// ask Bubble Tea to restore the terminal through its normal Run cleanup.
func (a *App) beginQuit() tea.Cmd {
	if a.quitting {
		return nil
	}
	a.quitting = true
	a.exitPending = true
	a.stopWriteFeedback()
	if a.cancelReads != nil {
		a.cancelReads()
		a.cancelReads = nil
	}
	return tea.Quit
}

func (a *App) cycleCategory(direction int) tea.Cmd {
	categories := a.catalogModel.Categories()
	if len(categories) == 0 {
		return nil
	}
	index := 0
	for i, category := range categories {
		if category.ID == a.catalogModel.ActiveCategory() {
			index = i
			break
		}
	}
	index = (index + direction + len(categories)) % len(categories)
	a.detailScroll = 0
	return a.catalogModel.SetCategory(categories[index].ID)
}

func (a *App) switchRoute(index int) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(a.routes) {
		return *a, nil
	}
	// A zero-change preview is read-only and never grants consent. Leaving the
	// install preview discards it, so returning later needs a fresh preview.
	if a.activeRoute == RouteInstall && a.routes[index].Route != RouteInstall &&
		a.installSection == installSectionInstall && a.install.PreviewNoOp() {
		a.install.invalidateConfirmation()
	}
	a.navIndex = index
	a.routeScroll[a.activeRoute] = a.contentScroll
	a.activeRoute = a.routes[index].Route
	a.focus = FocusContent
	a.contentScroll = a.routeScroll[a.activeRoute]
	a.notice = ""

	var cmds []tea.Cmd
	if a.activeRoute == RouteHome {
		if a.motion && !a.brandState.Done() && !a.brandTickPending && a.height >= 14 && !a.helpOpen {
			a.brandTickPending = true
			cmds = append(cmds, tickBrand())
		}
		// Re-entering Home never repeats the request: the manager's
		// attempted flag makes this a no-op after the first session entry.
		if cmd := a.automaticUpdateCheckCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if a.activeRoute == RouteActivity {
		a.activity.SetSize(a.childContentWidth(), a.childBodyHeight())
		cmds = append(cmds, a.startActivityRead(a.activity.selectedLog))
	}
	if a.activeRoute == RouteInstall {
		a.install.SetSize(a.childContentWidth(), a.childBodyHeight())
		a.restore.SetSize(a.childContentWidth(), a.childBodyHeight())
		a.fleet.SetSize(a.childContentWidth(), a.childBodyHeight())
		a.mcp.SetSize(a.childContentWidth(), a.childBodyHeight())
	}
	if len(cmds) == 0 {
		return *a, nil
	}
	return *a, tea.Batch(cmds...)
}

// OpenRecovery navigates directly to the recovery (backups) screen without creating a fifth route.
func (a *App) OpenRecovery() tea.Cmd {
	a.navIndex = 1
	a.routeScroll[a.activeRoute] = a.contentScroll
	a.activeRoute = RouteInstall
	a.focus = FocusContent
	a.notice = ""
	return a.openBackupsScreen()
}

func (a App) sidebarWidth() int {
	switch {
	case a.width >= 110:
		return 22
	case a.width >= 80:
		return 18
	case a.width >= 60:
		return 14
	default:
		return a.width
	}
}

func (a App) contentWidth() int {
	if a.width < 60 {
		return a.width
	}
	if a.width >= 80 && a.width <= 109 {
		// Medium layout: compact top navigation gives full horizontal width to content
		return a.width
	}
	w := a.width - a.sidebarWidth() - 1
	if w <= 0 {
		return 40
	}
	return w
}

func (a App) contentHeight() int {
	h := a.height - 3
	if a.width >= 80 && a.width <= 109 {
		h = a.height - 4
	}
	if h <= 4 {
		return 4
	}
	return h
}

// busyStatusLine is the single truthful footer while a write runs. It names
// the policy instead of offering q as a quit path, and it carries the phase
// marker plus the phase text the host actually reported. No percentage,
// throughput or completion claim is ever added here.
func (a App) busyStatusLine() string {
	line := fmt.Sprintf("%s | %s %s %s | q/Esc disabled | Ctrl+C exits after completion",
		WriteBusyMarker, a.writePhaseMarker(), a.write.kind, a.writePhaseText())
	if a.exitPending {
		line += " (exit pending"
		if a.exitSignal != "" {
			line += " by " + a.exitSignal
		}
		line += ")"
	}
	if a.cfg.Color {
		return a.theme.Warn.Render(line)
	}
	return line
}

// replaceLastLine swaps the footer row without changing the view's line count.
func replaceLastLine(content, line string, width int) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}
	if width > 0 {
		line = ansi.Truncate(line, width, "")
	}
	lines[len(lines)-1] = line
	return strings.Join(lines, "\n")
}

// View renders the application view, enforcing exact terminal width and height bounds
// without silently dropping headers or footers.
func (a App) View() tea.View {
	content := a.renderViewport()

	if !a.cfg.Color {
		content = ansi.Strip(content)
	}
	if a.write != nil && content != "" {
		// The busy policy owns the footer while a transaction runs, so a
		// rendered quit hint can never invite abandoning the write.
		content = replaceLastLine(content, a.busyStatusLine(), a.width)
	} else if content != "" {
		if line := a.footerOverride(); line != "" {
			if a.ascii {
				line = toASCII(line)
			}
			if a.cfg.Color {
				line = a.theme.Hint.Render(line)
			}
			content = replaceLastLine(content, line, a.width)
		}
	}

	view := tea.NewView(content)
	view.AltScreen = a.cfg.Interactive
	view.WindowTitle = BrandName
	// Bubble Tea v2 paints the full view: the canvas and default prose color
	// come from the theme only for colored sessions. Headless/plain runs leave
	// both nil, so their output and terminal state stay untouched.
	if a.cfg.Color {
		view.BackgroundColor = a.theme.CanvasColor()
		view.ForegroundColor = a.theme.DefaultForeground()
	}
	return view
}

func (a App) getContentLines(width int) []string {
	var lines []string
	if a.onboarding != nil {
		lines = a.onboarding.Lines(width)
	} else {
		switch a.activeRoute {
		case RouteHome:
			lines = a.homeLines(width)
		case RouteInstall:
			lines = a.installRouteLines(width)
		case RouteActivity:
			lines = a.activityRouteLines(width)
		}
	}

	// A notice normally owns the reserved footer status block, so scrolling or
	// resizing cannot hide it. Only when that block cannot be reserved (a tiny
	// terminal, or an active write whose busy footer carries the policy) does
	// the notice keep its historical scrollable body placement; the text is
	// never truncated or dropped.
	if a.notice != "" && len(a.statusNoticeRows(a.width)) == 0 {
		notice := Section(a.theme.Section.Render("Notice"), []string{a.notice}, width)
		lines = append(append(notice, ""), lines...)
	}
	return lines
}

// assembleView constructs the final viewport string with guaranteed exact height lines
// and width bounds, keeping the header at the top, scrolling the body in the middle,
// and preserving the footer at the very bottom.
func assembleView(header, body, footer []string, width, height, scroll int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	footerCount := len(footer)
	if height <= 8 && footerCount > 1 {
		footerCount = 1
		footer = footer[len(footer)-1:]
	}

	headerCount := len(header)
	if headerCount+footerCount >= height {
		headerCount = height - footerCount - 1
		if headerCount < 0 {
			headerCount = 0
		}
		header = header[:headerCount]
	}

	bodyHeight := height - headerCount - footerCount
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	maxScroll := len(body) - bodyHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	visibleBody := make([]string, 0, bodyHeight)
	if len(body) > 0 && scroll < len(body) {
		end := scroll + bodyHeight
		if end > len(body) {
			end = len(body)
		}
		visibleBody = append(visibleBody, body[scroll:end]...)
	}
	for len(visibleBody) < bodyHeight {
		visibleBody = append(visibleBody, "")
	}

	allLines := make([]string, 0, height)
	allLines = append(allLines, header...)
	allLines = append(allLines, visibleBody...)
	allLines = append(allLines, footer...)

	for i, line := range allLines {
		if ansi.StringWidth(line) > width {
			allLines[i] = ansi.Truncate(line, width, "")
		}
	}

	if len(allLines) > height {
		allLines = allLines[:height]
	}
	return strings.Join(allLines, "\n")
}

// toASCII maps Unicode box-drawing, arrows, bullets and symbols to 7-bit ASCII equivalents,
// and reversibly escapes unknown Unicode code points as \uXXXX or \UXXXXXXXX without deleting glyphs.
func toASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '•':
			b.WriteRune('*')
		case '·':
			b.WriteRune('-')
		case '►', '▸', '›':
			b.WriteRune('>')
		case '◄', '◂', '‹':
			b.WriteRune('<')
		case '▲', '▴':
			b.WriteRune('^')
		case '▼', '▾':
			b.WriteRune('v')
		case '─':
			b.WriteRune('-')
		case '│':
			b.WriteRune('|')
		case '╭', '╮', '╯', '╰', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼':
			b.WriteRune('+')
		case '…':
			b.WriteString("...")
		case '‘', '’':
			b.WriteRune('\'')
		case '“', '”':
			b.WriteRune('"')
		case '–', '—':
			b.WriteRune('-')
		default:
			if r <= 127 {
				b.WriteRune(r)
			} else if r <= 0xFFFF {
				fmt.Fprintf(&b, "\\u%04x", r)
			} else {
				fmt.Fprintf(&b, "\\U%08x", r)
			}
		}
	}
	return b.String()
}
