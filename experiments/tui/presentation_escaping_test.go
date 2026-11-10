package tui

// Regression coverage for the ASCII data-escaping finding: install/restore
// dynamic paths, IDs, notices, errors and log references must be escaped before
// decorative 7-bit mapping, while the authoritative request keeps the exact
// original value. Every operation here is a fake or an in-memory model; nothing
// is written.

import (
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

const escapeEmDash = "\u2014"

func assertSevenBit(t *testing.T, label, text string) {
	t.Helper()
	for i, r := range text {
		if r > 127 {
			t.Fatalf("%s contains non-ASCII rune %q (U+%04X) at %d:\n%s", label, r, r, i, text)
		}
	}
}

func assertEscapedOnce(t *testing.T, label, text, data string) {
	t.Helper()
	if !strings.Contains(data, escapeEmDash) {
		t.Fatalf("%s fixture does not contain an em dash: %q", label, data)
	}
	escaped := escapeData(data)
	if !strings.Contains(text, escaped) {
		t.Fatalf("%s did not render the reversible escaped value %q:\n%s", label, escaped, text)
	}
	decorated := strings.ReplaceAll(data, escapeEmDash, "-")
	if strings.Contains(text, data) || strings.Contains(text, decorated) {
		t.Fatalf("%s rendered the raw or decoratively rewritten path %q instead of %q:\n%s", label, data, escaped, text)
	}
}

// TestInstallPathEscapingIsReversibleAndRequestExact proves preview,
// confirmation and result rendering use an escaped display value for dynamic
// paths/IDs, while the preview and write requests keep the original bytes.
func TestInstallPathEscapingIsReversibleAndRequestExact(t *testing.T) {
	home := "/tmp/a" + escapeEmDash + "b"
	repo := "/tmp/r" + escapeEmDash + "p"
	plannedPath := home + "/.claude/skills/skill" + escapeEmDash + "id"
	logPath := home + "/logs/op" + escapeEmDash + "log.log"
	backupPath := home + "/backups/b" + escapeEmDash + "1"

	var previewReq appops.InstallRequest
	preview := func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error) {
		previewReq = req
		return appops.PreviewResult{
			Operation:    skillsync.OperationApply,
			Outcome:      skillsync.OutcomePreview,
			PreviewToken: "token-escape",
			Planned: []skillsync.ResultChange{{
				ID:     "skill" + escapeEmDash + "id",
				Client: "claude",
				Path:   plannedPath,
				Action: skillsync.ChangeInstall,
			}},
		}, nil
	}

	app := NewAppWithScreens(
		TUIConfig{Repo: repo, Home: home, Width: 100, Height: 40, Color: false, ASCII: true},
		Catalog{Home: home, Repo: repo},
		AppOptions{},
		AppScreenOptions{InstallPreview: preview},
	)
	app = b8OpenInstallPreview(t, app)
	if app.install.Step() != InstallStepPreview {
		t.Fatalf("preview step = %v", app.install.Step())
	}

	// The read-only preview request is authoritative and unchanged.
	if previewReq.Home() != home || previewReq.Repo() != repo {
		t.Fatalf("preview request home/repo = %q/%q, want %q/%q", previewReq.Home(), previewReq.Repo(), home, repo)
	}

	// Paths start collapsed; expand them and check the escaped data.
	app = b8Press(t, app, 'x')
	previewText := ansi.Strip(app.View().Content)
	assertSevenBit(t, "install preview", previewText)
	assertEscapedOnce(t, "install preview home", previewText, home)
	assertEscapedOnce(t, "install preview path", previewText, plannedPath)
	assertEscapedOnce(t, "install preview ID", previewText, "skill"+escapeEmDash+"id")

	// The confirmation keeps the exact home and target directories visible.
	token := app.install.PreviewToken()
	app = b8Press(t, app, 'x') // collapse paths, then move to confirmation
	app, _ = b8Update(t, app, b8Code(tea.KeyEnter))
	if app.install.Step() != InstallStepConfirm {
		t.Fatalf("confirmation step = %v", app.install.Step())
	}
	confirmText := ansi.Strip(app.View().Content)
	assertSevenBit(t, "install confirmation", confirmText)
	assertEscapedOnce(t, "install confirmation home", confirmText, home)
	assertEscapedOnce(t, "install confirmation destination", confirmText, ClientTargetDir(home, "claude"))
	if app.install.PreviewToken() != token {
		t.Fatal("ASCII rendering changed the preview token")
	}

	// The confirmed write request keeps the original home/repo even though the
	// rendered confirmation used escaped display strings.
	next, cmd := b8Update(t, app, b8Key('y'))
	app = next
	if cmd == nil {
		t.Fatal("y did not emit a write command")
	}
	msg := cmd()
	start, ok := msg.(StartWriteMsg)
	if !ok {
		t.Fatalf("y emitted %T, want StartWriteMsg", msg)
	}
	if start.Request.Install.Home() != home || start.Request.Install.Repo() != repo {
		t.Fatalf("write request home/repo = %q/%q, want %q/%q",
			start.Request.Install.Home(), start.Request.Install.Repo(), home, repo)
	}

	// Result rendering escapes the home, backup references and operation log.
	m := NewInstallModel(TUIConfig{Repo: repo, Home: home, Width: 100, Height: 40, ASCII: true})
	m.step = InstallStepResult
	m.outcome = InstallOutcomeApplied
	m.home = home
	m.repo = repo
	m.logPath = logPath
	m.applyResult = &appops.ApplyResult{
		Outcome:    skillsync.OutcomeApplied,
		BackupID:   "backup" + escapeEmDash + "id",
		BackupPath: backupPath,
	}
	resultText := strings.Join(m.Lines(100), "\n")
	assertSevenBit(t, "install result", resultText)
	assertEscapedOnce(t, "install result home", resultText, home)
	assertEscapedOnce(t, "install result log", resultText, logPath)
	assertEscapedOnce(t, "install result backup", resultText, backupPath)
	if m.home != home || m.repo != repo || m.logPath != logPath {
		t.Fatal("rendering mutated authoritative install paths")
	}
}

// TestInstallRuntimeASCIIToggleEscapesData covers the session path where ASCII
// is enabled after the screen was constructed: the root display copy must
// activate the child's data escaping for that render.
func TestInstallRuntimeASCIIToggleEscapesData(t *testing.T) {
	home := "/tmp/c" + escapeEmDash + "d"
	plannedPath := home + "/.claude/skills/skill" + escapeEmDash + "id"
	preview := fakePreviewOp([]skillsync.ResultChange{{
		ID:     "skill" + escapeEmDash + "id",
		Client: "claude",
		Path:   plannedPath,
		Action: skillsync.ChangeInstall,
	}}, "token-runtime-ascii", nil)

	app := b8App(t, TUIConfig{Home: home, Width: 100, Height: 40, Color: false}, nil, preview)
	app = b8OpenInstallPreview(t, app)
	app = b8Press(t, app, 'u')
	if !app.ASCII() {
		t.Fatal("u did not enable ASCII")
	}
	app = b8Press(t, app, 'x')
	text := ansi.Strip(app.View().Content)
	assertSevenBit(t, "runtime ASCII preview", text)
	assertEscapedOnce(t, "runtime ASCII home", text, home)
	assertEscapedOnce(t, "runtime ASCII path", text, plannedPath)
	if app.install.home != home {
		t.Fatalf("ASCII display copy mutated the install home to %q", app.install.home)
	}
}

// TestCatalogNoticesAndRootNoticeEscaping proves Home notices and the root
// execution notice use the same reversible data escaping as child paths.
func TestCatalogNoticesAndRootNoticeEscaping(t *testing.T) {
	home := "/tmp/m" + escapeEmDash + "n"
	packPath := home + "/pack" + escapeEmDash + "dir"
	cat := Catalog{Home: home, Notices: []string{"Optional pack missing at " + packPath}}

	app := NewApp(TUIConfig{Home: home, Width: 100, Height: 30, Color: false, ASCII: true}, cat)
	// displayCopy is the render seam: Home's notice lines must be escaped even
	// though the raw catalog still holds the original path.
	copy := app.displayCopy()
	noticeText := strings.Join(copy.catalog.Notices, "\n")
	assertEscapedOnce(t, "Home notice path", noticeText, packPath)
	if copy.catalog.Notices[0] == cat.Notices[0] {
		t.Fatal("display copy kept the raw catalog notice")
	}

	app.notice = "Inventory read failed for " + packPath
	view := ansi.Strip(app.View().Content)
	assertSevenBit(t, "root notice", view)
	assertEscapedOnce(t, "root notice path", view, packPath)
}

// TestRestorePathEscapingIsReversible proves the recovery preview,
// confirmation and result escape dynamic IDs and paths without changing the
// configured request paths.
func TestRestorePathEscapingIsReversible(t *testing.T) {
	home := "/tmp/e" + escapeEmDash + "f"
	repo := "/tmp/g" + escapeEmDash + "h"
	plannedPath := home + "/.claude/skills/skill" + escapeEmDash + "id"
	safetyPath := home + "/backups/safe" + escapeEmDash + "1"
	logPath := home + "/logs/restore" + escapeEmDash + "log.log"

	m := NewRestoreModel(TUIConfig{Repo: repo, Home: home, Width: 100, Height: 40, ASCII: true}, RestoreOps{})

	m.stage = RestoreStagePreview
	m.previewID = "20260101T000000.000000000Z"
	m.preview = appops.PreviewResult{
		Operation:    skillsync.OperationRestore,
		Outcome:      skillsync.OutcomePreview,
		PreviewToken: "token-restore-escape",
		Planned: []skillsync.ResultChange{{
			ID:     "skill" + escapeEmDash + "id",
			Client: "claude",
			Path:   plannedPath,
			Action: skillsync.ChangeRestore,
		}},
	}
	previewText := strings.Join(m.Lines(100), "\n")
	assertSevenBit(t, "restore preview", previewText)
	assertEscapedOnce(t, "restore preview path", previewText, plannedPath)

	m.stage = RestoreStageConfirm
	confirmText := strings.Join(m.Lines(100), "\n")
	assertSevenBit(t, "restore confirmation", confirmText)
	assertEscapedOnce(t, "restore confirmation path", confirmText, plannedPath)

	m.stage = RestoreStageResult
	m.result = &RestoreOutcome{
		Kind:             RestoreOutcomeApplied,
		BackupID:         "20260101T000000.000000000Z",
		SafetyBackupID:   "20260102T000000.000000000Z",
		SafetyBackupPath: safetyPath,
		LogPath:          logPath,
		Applied: []skillsync.ResultChange{{
			ID:     "skill" + escapeEmDash + "id",
			Client: "claude",
			Path:   plannedPath,
			Action: skillsync.ChangeRestore,
		}},
	}
	resultText := strings.Join(m.Lines(100), "\n")
	assertSevenBit(t, "restore result", resultText)
	assertEscapedOnce(t, "restore result safety path", resultText, safetyPath)
	assertEscapedOnce(t, "restore result log", resultText, logPath)
	assertEscapedOnce(t, "restore result path", resultText, plannedPath)

	if m.cfg.Home != home || m.cfg.Repo != repo {
		t.Fatalf("rendering mutated restore config home/repo = %q/%q", m.cfg.Home, m.cfg.Repo)
	}
	req, err := m.restoreWriteRequest("20260101T000000.000000000Z", "token-restore-escape")
	if err != nil {
		t.Fatalf("restore write request: %v", err)
	}
	if req.Restore.Home() != home || req.Restore.Repo() != repo {
		t.Fatalf("restore request home/repo = %q/%q, want %q/%q", req.Restore.Home(), req.Restore.Repo(), home, repo)
	}
}
