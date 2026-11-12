package tui_test

// Deterministic tests for the fleet write kinds on the nested TUI's single
// write host. Every seam is a fake: no appops fleet call runs, no destination
// is touched and the transcript lives in memory. The only filesystem entry
// created is the temporary home the requests are resolved against.

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// fleetLog is an in-memory WriteLog that records the host's lifecycle.
type fleetLog struct {
	mu      sync.Mutex
	content strings.Builder
	flushed bool
	closed  bool
}

func (l *fleetLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.content.Write(p)
}

func (l *fleetLog) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushed = true
	return nil
}

func (l *fleetLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}

func (l *fleetLog) transcript() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.content.String()
}

func (l *fleetLog) lifecycle() (flushed, closed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flushed, l.closed
}

// fleetSeamResult is the structured result one fake fleet seam returns.
func fleetSeamResult(operation skillsync.OperationKind) appops.ApplyResult {
	return appops.ApplyResult{
		Operation: operation,
		Outcome:   skillsync.OutcomeApplied,
		Applied:   []skillsync.ResultChange{{Client: "opencode", ID: "sdd-plan", Action: skillsync.ChangeReplace}},
		BackupID:  "20260101T000000.000000000Z",
	}
}

// testFleetRequest builds one valid request for the given fleet kind with a
// resolved home, config path, identity and preview token.
func testFleetRequest(t *testing.T, kind tui.WriteKind) tui.WriteRequest {
	t.Helper()
	home := t.TempDir()
	fleet := tui.FleetWriteRequest{
		Home:       home,
		ConfigPath: filepath.Join(home, "private-checkout", "fleet.json"),
		Token:      "fleet-preview-token",
		Label:      "Fleet change",
		Source:     "personal fleet.json",
	}
	switch kind {
	case tui.WriteFleetApply:
		fleet.PlanID = "sha256:plan-abc123"
	case tui.WriteFleetRestore:
		fleet.BackupID = "20260101T000000.000000000Z"
	}
	return tui.WriteRequest{Kind: kind, Fleet: fleet}
}

// testInstallWriteRequest builds one valid install request in its own home.
func testInstallWriteRequest(t *testing.T) tui.WriteRequest {
	t.Helper()
	request, err := appops.NewInstallRequest("", t.TempDir(), true, skillsync.Options{})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	return tui.WriteRequest{
		Kind:    tui.WriteInstall,
		Install: request,
		Token:   "install-preview-token",
		Command: "apply --with-opencode",
	}
}

func fleetKinds() []tui.WriteKind {
	return []tui.WriteKind{tui.WriteFleetApply, tui.WriteFleetRestore, tui.WriteSourceSave}
}

func wantFleetVerb(kind tui.WriteKind) string {
	switch kind {
	case tui.WriteFleetApply:
		return "fleet-apply"
	case tui.WriteFleetRestore:
		return "fleet-restore"
	case tui.WriteSourceSave:
		return "save-config"
	default:
		return "unknown"
	}
}

// wantFleetCommand is the exact transcript header FR-21 requires.
func wantFleetCommand(req tui.WriteRequest) string {
	switch req.Kind {
	case tui.WriteFleetApply:
		return "fleet apply --plan " + req.Fleet.PlanID
	case tui.WriteFleetRestore:
		return "fleet restore " + req.Fleet.BackupID
	case tui.WriteSourceSave:
		return "fleet config save"
	default:
		return ""
	}
}

// fleetNoopSeams wires one no-op fake for every fleet kind.
func fleetNoopSeams() tui.OpsHostOptions {
	noop := func(tui.FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
		return appops.ApplyResult{}, nil
	}
	return tui.OpsHostOptions{
		ApplyFleet:        noop,
		ApplyFleetRestore: noop,
		SaveFleetSource:   noop,
	}
}

func TestFleetWriteKindVerbs(t *testing.T) {
	cases := []struct {
		kind tui.WriteKind
		want string
	}{
		{kind: tui.WriteInstall, want: "apply"},
		{kind: tui.WriteRestore, want: "restore"},
		{kind: tui.WriteFleetApply, want: "fleet-apply"},
		{kind: tui.WriteFleetRestore, want: "fleet-restore"},
		{kind: tui.WriteSourceSave, want: "save-config"},
		{kind: tui.WriteKind(99), want: "unknown"},
	}
	for _, tc := range cases {
		if got := tc.kind.String(); got != tc.want {
			t.Fatalf("%d.String() = %q, want %q", int(tc.kind), got, tc.want)
		}
	}
	// The fleet kinds are real, distinct kinds with their own stable verbs.
	seen := map[tui.WriteKind]bool{tui.WriteInstall: true, tui.WriteRestore: true}
	for _, kind := range fleetKinds() {
		if seen[kind] {
			t.Fatalf("fleet kind %v reuses an install/restore value", kind)
		}
		seen[kind] = true
		if got := kind.String(); got != wantFleetVerb(kind) {
			t.Fatalf("%v.String() = %q, want %q", kind, got, wantFleetVerb(kind))
		}
	}
}

func TestFleetWriteRequestValidation(t *testing.T) {
	apply := testFleetRequest(t, tui.WriteFleetApply)
	restore := testFleetRequest(t, tui.WriteFleetRestore)
	save := testFleetRequest(t, tui.WriteSourceSave)
	install := testInstallWriteRequest(t)

	invalidate := func(req tui.WriteRequest, mutate func(*tui.WriteRequest)) tui.WriteRequest {
		mutate(&req)
		return req
	}

	cases := []struct {
		name string
		req  tui.WriteRequest
	}{
		{name: "unknown kind", req: tui.WriteRequest{Kind: tui.WriteKind(99), Token: "t"}},
		{
			name: "install carrying a fleet payload",
			req:  invalidate(install, func(r *tui.WriteRequest) { r.Fleet = apply.Fleet }),
		},
		{
			name: "fleet apply without its token",
			req:  invalidate(apply, func(r *tui.WriteRequest) { r.Fleet.Token = "" }),
		},
		{
			name: "fleet apply with a mismatched outer token",
			req:  invalidate(apply, func(r *tui.WriteRequest) { r.Token = "another-token" }),
		},
		{
			name: "fleet apply without a resolved home",
			req:  invalidate(apply, func(r *tui.WriteRequest) { r.Fleet.Home = "" }),
		},
		{
			name: "fleet apply without a config path",
			req:  invalidate(apply, func(r *tui.WriteRequest) { r.Fleet.ConfigPath = "" }),
		},
		{
			name: "fleet apply without a plan identity",
			req:  invalidate(apply, func(r *tui.WriteRequest) { r.Fleet.PlanID = "" }),
		},
		{
			name: "fleet apply carrying a backup identity",
			req:  invalidate(apply, func(r *tui.WriteRequest) { r.Fleet.BackupID = "backup" }),
		},
		{
			name: "fleet restore without a backup identity",
			req:  invalidate(restore, func(r *tui.WriteRequest) { r.Fleet.BackupID = "" }),
		},
		{
			name: "fleet restore carrying a plan identity",
			req:  invalidate(restore, func(r *tui.WriteRequest) { r.Fleet.PlanID = "plan" }),
		},
		{
			name: "source save with a plan identity",
			req:  invalidate(save, func(r *tui.WriteRequest) { r.Fleet.PlanID = "plan" }),
		},
		{
			name: "source save with a backup identity",
			req:  invalidate(save, func(r *tui.WriteRequest) { r.Fleet.BackupID = "backup" }),
		},
		{
			name: "fleet command header cannot drift",
			req:  invalidate(apply, func(r *tui.WriteRequest) { r.Command = "fleet apply --from-file /tmp/plan.json" }),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := tui.NewOpsHost(fleetNoopSeams())
			_, cmd, err := host.BeginWrite(tc.req)
			if !errors.Is(err, tui.ErrWriteInvalid) {
				t.Fatalf("BeginWrite error = %v, want ErrWriteInvalid", err)
			}
			if cmd != nil {
				t.Fatal("a rejected fleet request returned a command")
			}
			if _, ok := host.ActiveWrite(); ok {
				t.Fatal("a rejected fleet request claimed the write slot")
			}
		})
	}

	// The valid baseline for every fleet kind is accepted, and an exactly
	// matching command header is accepted too.
	for _, tc := range []struct {
		req tui.WriteRequest
	}{
		{req: apply},
		{req: restore},
		{req: save},
		{req: invalidate(apply, func(r *tui.WriteRequest) { r.Command = wantFleetCommand(apply) })},
	} {
		host := tui.NewOpsHost(fleetNoopSeams())
		if _, cmd, err := host.BeginWrite(tc.req); err != nil || cmd == nil {
			t.Fatalf("%v BeginWrite = %v / cmd nil=%t; want an accepted write", tc.req.Kind, err, cmd == nil)
		}
	}
}

func TestFleetWriteTranscriptIdentity(t *testing.T) {
	for _, kind := range fleetKinds() {
		t.Run(kind.String(), func(t *testing.T) {
			req := testFleetRequest(t, kind)
			log := &fleetLog{}
			var gotKind tui.WriteKind
			var gotHome string
			var gotRequest tui.FleetWriteRequest
			var gotToken string
			var calls int32

			host := tui.NewOpsHost(tui.OpsHostOptions{
				OpenLog: func(writeKind tui.WriteKind, home string) (tui.WriteLog, string, error) {
					gotKind = writeKind
					gotHome = home
					return log, "/tmp/fleet-operation.log", nil
				},
				ApplyFleet: func(r tui.FleetWriteRequest, token string, _ io.Writer) (appops.ApplyResult, error) {
					calls++
					gotRequest, gotToken = r, token
					return fleetSeamResult(skillsync.OperationApply), nil
				},
				ApplyFleetRestore: func(r tui.FleetWriteRequest, token string, _ io.Writer) (appops.ApplyResult, error) {
					calls++
					gotRequest, gotToken = r, token
					return fleetSeamResult(skillsync.OperationRestore), nil
				},
				SaveFleetSource: func(r tui.FleetWriteRequest, token string, _ io.Writer) (appops.ApplyResult, error) {
					calls++
					gotRequest, gotToken = r, token
					return appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeApplied}, nil
				},
			})
			var progress []tui.WriteProgressMsg
			host.SetProgressSink(func(msg tea.Msg) {
				if update, ok := msg.(tui.WriteProgressMsg); ok {
					progress = append(progress, update)
				}
			})

			started, cmd, err := host.BeginWrite(req)
			if err != nil {
				t.Fatalf("BeginWrite: %v", err)
			}
			if cmd == nil {
				t.Fatal("BeginWrite returned no command")
			}
			if started.Kind != kind {
				t.Fatalf("started kind = %v, want %v", started.Kind, kind)
			}

			msg := cmd().(tui.WriteResultMsg)
			if msg.ID != started.ID {
				t.Fatalf("result ID = %+v, want %+v", msg.ID, started.ID)
			}
			if msg.Kind != kind {
				t.Fatalf("result kind = %v, want %v", msg.Kind, kind)
			}
			if msg.Err != nil || msg.LogErr != nil {
				t.Fatalf("result errors = %v / %v", msg.Err, msg.LogErr)
			}
			if calls != 1 {
				t.Fatalf("fleet seam ran %d times, want exactly one", calls)
			}
			if gotKind != kind {
				t.Fatalf("log factory kind = %v, want %v", gotKind, kind)
			}
			// The resolved request home reaches the private log, even though the
			// personal source lives outside it.
			if gotHome != req.Fleet.Home || gotHome == "" {
				t.Fatalf("log factory home = %q, want the resolved request home %q", gotHome, req.Fleet.Home)
			}
			if gotRequest != req.Fleet {
				t.Fatalf("seam request = %+v, want %+v", gotRequest, req.Fleet)
			}
			if gotToken != req.Fleet.Token {
				t.Fatalf("seam token = %q, want %q", gotToken, req.Fleet.Token)
			}
			if msg.Result.Operation == "" || msg.Result.Outcome != skillsync.OutcomeApplied {
				t.Fatalf("result = %+v, want the structured seam result", msg.Result)
			}

			transcript := log.transcript()
			for _, want := range []string{
				"Command: " + wantFleetCommand(req),
				"Source: " + req.Fleet.Source,
				"Home: " + req.Fleet.Home,
				"Result: success",
			} {
				if !strings.Contains(transcript, want) {
					t.Fatalf("transcript missing %q:\n%s", want, transcript)
				}
			}
			if strings.Contains(transcript, "Source: embedded") {
				t.Fatalf("fleet source was labelled embedded:\n%s", transcript)
			}
			if flushed, closed := log.lifecycle(); !flushed || !closed {
				t.Fatalf("log lifecycle flushed=%t closed=%t, want both", flushed, closed)
			}

			wantSteps := []tui.WriteStep{tui.WriteStepStarting, tui.WriteStepWriting, tui.WriteStepFlushing}
			if len(progress) != len(wantSteps) {
				t.Fatalf("progress steps = %v, want %v", progress, wantSteps)
			}
			for i, step := range wantSteps {
				if progress[i].Step != step || progress[i].ID != started.ID || progress[i].Kind != kind {
					t.Fatalf("progress[%d] = %+v, want %s for %+v", i, progress[i], step, started.ID)
				}
			}

			if !host.Complete(started.ID) {
				t.Fatal("Complete refused the matching result identity")
			}
			if _, ok := host.ActiveWrite(); ok {
				t.Fatal("the write slot is still held after a matching Complete")
			}
		})
	}
}

func TestFleetWriteSourceFallsBackToConfigPath(t *testing.T) {
	req := testFleetRequest(t, tui.WriteFleetApply)
	req.Fleet.Source = ""
	log := &fleetLog{}
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return log, "/tmp/fleet-operation.log", nil
		},
		ApplyFleet: func(req tui.FleetWriteRequest, _ string, _ io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
	})
	_, cmd, err := host.BeginWrite(req)
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	if msg := cmd().(tui.WriteResultMsg); msg.Err != nil {
		t.Fatalf("fleet apply error = %v", msg.Err)
	}
	transcript := log.transcript()
	if !strings.Contains(transcript, "Source: "+req.Fleet.ConfigPath) {
		t.Fatalf("transcript missing the config-path source:\n%s", transcript)
	}
	if strings.Contains(transcript, "embedded") {
		t.Fatalf("fleet source fell back to embedded:\n%s", transcript)
	}
}

func TestFleetWriteNilSeamRefuses(t *testing.T) {
	// Fleet apply and fleet restore now have production appops seams in
	// NewOpsHost. Source save remains deliberately unwired until B4's
	// separately approved write surface lands, so its nil seam must still
	// refuse rather than fake a successful save.
	for _, kind := range []tui.WriteKind{tui.WriteSourceSave} {
		t.Run(kind.String(), func(t *testing.T) {
			req := testFleetRequest(t, kind)
			var openLogCalls int
			host := tui.NewOpsHost(tui.OpsHostOptions{
				OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
					openLogCalls++
					return &fleetLog{}, "/tmp/fleet-operation.log", nil
				},
			})
			started, cmd, err := host.BeginWrite(req)
			if err != nil {
				t.Fatalf("BeginWrite: %v", err)
			}
			msg := cmd().(tui.WriteResultMsg)
			if !errors.Is(msg.Err, tui.ErrFleetUnavailable) {
				t.Fatalf("result error = %v, want ErrFleetUnavailable", msg.Err)
			}
			if !strings.Contains(msg.Err.Error(), kind.String()) {
				t.Fatalf("error %q does not name the refused kind %q", msg.Err, kind)
			}
			if msg.Result.Outcome != "" || len(msg.Result.Applied) != 0 || msg.Result.BackupID != "" {
				t.Fatalf("refused fleet write reported a structured result: %+v", msg.Result)
			}
			if openLogCalls != 0 {
				t.Fatalf("the transcript was opened %d times for an unwired fleet write", openLogCalls)
			}
			if msg.LogPath != "" {
				t.Fatalf("refused fleet write reported a log path %q", msg.LogPath)
			}
			if msg.ID != started.ID {
				t.Fatalf("result ID = %+v, want %+v so the slot can be released", msg.ID, started.ID)
			}
			if !host.Complete(started.ID) {
				t.Fatal("the refused write could not release its slot")
			}
		})
	}
}

func TestFleetWriteSharesSingleSlotBothWays(t *testing.T) {
	t.Run("active fleet write refuses an install", func(t *testing.T) {
		var installCalls int
		host := tui.NewOpsHost(tui.OpsHostOptions{
			OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
				return &fleetLog{}, "/tmp/operation.log", nil
			},
			ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
				installCalls++
				return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
			},
			SaveFleetSource: func(tui.FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
				return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
			},
		})
		fleetReq := testFleetRequest(t, tui.WriteSourceSave)
		fleetStarted, fleetCmd, err := host.BeginWrite(fleetReq)
		if err != nil {
			t.Fatalf("fleet BeginWrite: %v", err)
		}
		installReq := testInstallWriteRequest(t)
		_, installCmd, err := host.BeginWrite(installReq)
		if !errors.Is(err, tui.ErrWriteInProgress) {
			t.Fatalf("install BeginWrite error = %v, want ErrWriteInProgress", err)
		}
		if installCmd != nil {
			t.Fatal("the refused install queued a command")
		}
		active, ok := host.ActiveWrite()
		if !ok || active.ID != fleetStarted.ID || active.Kind != tui.WriteSourceSave {
			t.Fatalf("active write = %+v ok=%t, want the fleet write %+v", active, ok, fleetStarted.ID)
		}
		// Only the fleet transaction produces a result.
		if _, ok := fleetCmd().(tui.WriteResultMsg); !ok {
			t.Fatal("the fleet command did not return a write result")
		}
		if installCalls != 0 {
			t.Fatalf("install seam ran %d times for a refused request", installCalls)
		}
		if !host.Complete(fleetStarted.ID) {
			t.Fatal("Complete refused the fleet identity")
		}
		// Once released, the install request is accepted on the same slot.
		installStarted, installRun, err := host.BeginWrite(installReq)
		if err != nil || installRun == nil {
			t.Fatalf("install after release = %v / cmd nil=%t", err, installRun == nil)
		}
		if _, ok := installRun().(tui.WriteResultMsg); !ok {
			t.Fatal("the install command did not return a write result")
		}
		if installCalls != 1 {
			t.Fatalf("install seam ran %d times, want one", installCalls)
		}
		host.Complete(installStarted.ID)
	})

	t.Run("active install refuses a fleet write", func(t *testing.T) {
		var fleetCalls int
		host := tui.NewOpsHost(tui.OpsHostOptions{
			OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
				return &fleetLog{}, "/tmp/operation.log", nil
			},
			ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
				return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
			},
			ApplyFleet: func(tui.FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
				fleetCalls++
				return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
			},
		})
		installReq := testInstallWriteRequest(t)
		installStarted, installRun, err := host.BeginWrite(installReq)
		if err != nil {
			t.Fatalf("install BeginWrite: %v", err)
		}
		fleetReq := testFleetRequest(t, tui.WriteFleetApply)
		_, fleetCmd, err := host.BeginWrite(fleetReq)
		if !errors.Is(err, tui.ErrWriteInProgress) {
			t.Fatalf("fleet BeginWrite error = %v, want ErrWriteInProgress", err)
		}
		if fleetCmd != nil {
			t.Fatal("the refused fleet write queued a command")
		}
		if fleetCalls != 0 {
			t.Fatalf("fleet seam ran %d times for a refused request", fleetCalls)
		}
		if _, ok := installRun().(tui.WriteResultMsg); !ok {
			t.Fatal("the install command did not return a write result")
		}
		if !host.Complete(installStarted.ID) {
			t.Fatal("Complete refused the install identity")
		}
		fleetStarted, fleetRun, err := host.BeginWrite(fleetReq)
		if err != nil || fleetRun == nil {
			t.Fatalf("fleet after release = %v / cmd nil=%t", err, fleetRun == nil)
		}
		fleetRun()
		if fleetCalls != 1 {
			t.Fatalf("fleet seam ran %d times, want one", fleetCalls)
		}
		host.Complete(fleetStarted.ID)
	})
}

func TestFleetWriteCompleteRequiresMatchingIdentity(t *testing.T) {
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return &fleetLog{}, "/tmp/operation.log", nil
		},
		ApplyFleetRestore: func(tui.FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return fleetSeamResult(skillsync.OperationRestore), nil
		},
	})
	req := testFleetRequest(t, tui.WriteFleetRestore)
	started, cmd, err := host.BeginWrite(req)
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	if msg := cmd().(tui.WriteResultMsg); msg.ID != started.ID {
		t.Fatalf("result ID = %+v, want %+v", msg.ID, started.ID)
	}
	stale := appops.OperationID{Request: started.ID.Request + 7, Generation: started.ID.Generation}
	if host.Complete(stale) {
		t.Fatal("Complete accepted a mismatched identity")
	}
	if active, ok := host.ActiveWrite(); !ok || active.ID != started.ID {
		t.Fatalf("a mismatched Complete changed the active write: %+v ok=%t", active, ok)
	}
	if !host.Complete(started.ID) {
		t.Fatal("Complete refused the matching identity")
	}
	if host.Complete(started.ID) {
		t.Fatal("Complete accepted the same identity twice")
	}
	if completed, ok := host.Completed(); !ok || completed != started.ID {
		t.Fatalf("Completed = %+v ok=%t, want %+v", completed, ok, started.ID)
	}
}
