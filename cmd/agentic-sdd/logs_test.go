package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// logTimestampRE matches only the RFC3339Nano header/footer times so the rest
// of a log file can be compared byte for byte.
var logTimestampRE = regexp.MustCompile(`[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z`)

// logsFixture builds an isolated repository with one portable skill and a
// home. seedOld gives claude a different copy so apply replaces and backs it
// up instead of installing all four destinations fresh.
func logsFixture(t *testing.T, seedOld bool) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo, home = filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if seedOld {
		old := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
		if err := os.MkdirAll(old, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return repo, home
}

// logsLineValue returns the value after prefix in a single stdout line.
func logsLineValue(t *testing.T, text, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	t.Fatalf("output has no line starting with %q:\n%s", prefix, text)
	return ""
}

// logsOnlyLogFile returns the operation log path, requiring exactly one.
func logsOnlyLogFile(t *testing.T, home string) string {
	t.Helper()
	entries, err := os.ReadDir(logsDir(home))
	if err != nil || len(entries) != 1 {
		t.Fatalf("operation logs = %v, %v", entries, err)
	}
	return filepath.Join(logsDir(home), entries[0].Name())
}

// logsBackupCount counts backup directories under an explicit repo.
func logsBackupCount(t *testing.T, repo string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	return len(entries)
}

// logsOpResultLine returns the raw op.result trace line, preserving field
// order for order assertions.
func logsOpResultLine(t *testing.T, raw string) string {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSuffix(raw, "\n"), "\n") {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("trace line is not JSON: %q: %v", line, err)
		}
		if event["event"] == "op.result" {
			return line
		}
	}
	t.Fatalf("trace has no op.result event:\n%s", raw)
	return ""
}

// TestApplyLogAndStdoutExact pins the complete apply stdout and log file for a
// deterministic fixture, including log permissions. The backup and log names
// are generated, so they are read from stdout and then required to appear
// byte-exactly.
func TestApplyLogAndStdoutExact(t *testing.T) {
	repo, home := logsFixture(t, false)
	var stdout, stderr bytes.Buffer
	code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80})
	if code != 0 {
		t.Fatalf("apply exit %d: %s", code, stderr.String())
	}
	backupPath := logsLineValue(t, stdout.String(), "Installed 4 skills; backup: ")
	logPath := logsLineValue(t, stdout.String(), "Log: ")
	if !strings.HasPrefix(backupPath, filepath.Join(repo, "backups")+string(filepath.Separator)) {
		t.Fatalf("backup path = %q", backupPath)
	}
	wantStdout := "install agents/agentic-sdd-plan\n" +
		"install codex/agentic-sdd-plan\n" +
		"install claude/agentic-sdd-plan\n" +
		"install agy/agentic-sdd-plan\n" +
		"Installed 4 skills; backup: " + backupPath + "\n" +
		"Log: " + logPath + "\n"
	if stdout.String() != wantStdout {
		t.Fatalf("stdout = %q, want %q", stdout.String(), wantStdout)
	}
	if stderr.Len() != 0 {
		t.Fatalf("apply stderr = %q", stderr.String())
	}
	if logPath != logsOnlyLogFile(t, home) {
		t.Fatalf("stdout log path %q is not the created log", logPath)
	}
	info, err := os.Stat(logPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("log permissions = %v, %v", info, err)
	}
	transcript := strings.TrimSuffix(stdout.String(), "Log: "+logPath+"\n")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	wantLog := "Started: <time>\nCommand: apply\nSource: " + repo + "\nHome: " + home + "\n" +
		transcript + "Result: success\nFinished: <time>\n"
	if got := logTimestampRE.ReplaceAllString(string(data), "<time>"); got != wantLog {
		t.Fatalf("log = %q, want %q", got, wantLog)
	}
}

// TestRestoreLogAndStdoutExact pins restore apply's stdout and log after an
// apply-produced backup; one installed destination is removed so the
// transcript covers the up-to-date branch.
func TestRestoreLogAndStdoutExact(t *testing.T) {
	repo, home := logsFixture(t, true)
	var setupOut, setupErr bytes.Buffer
	if code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &setupOut, &setupErr, strings.NewReader(""), &Env{Width: 80}); code != 0 {
		t.Fatalf("setup apply exit %d: %s", code, setupErr.String())
	}
	id := logsOnlyBackupID(t, repo)
	if err := os.RemoveAll(filepath.Join(home, ".codex", "skills", "agentic-sdd-plan")); err != nil {
		t.Fatal(err)
	}
	// The setup log would confuse logsOnlyLogFile; keep only the restore log.
	logsRemoveAll(t, logsDir(home))

	var stdout, stderr bytes.Buffer
	code := runEnv([]string{"restore", "apply", id, "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80})
	if code != 0 {
		t.Fatalf("restore exit %d: %s", code, stderr.String())
	}
	backupPath := logsLineValue(t, stdout.String(), "Restored 3 skills; backup: ")
	logPath := logsLineValue(t, stdout.String(), "Log: ")
	wantStdout := "remove + back up agents/agentic-sdd-plan\n" +
		"up to date codex/agentic-sdd-plan\n" +
		"replace + back up claude/agentic-sdd-plan\n" +
		"remove + back up agy/agentic-sdd-plan\n" +
		"Restored 3 skills; backup: " + backupPath + "\n" +
		"Log: " + logPath + "\n"
	if stdout.String() != wantStdout {
		t.Fatalf("stdout = %q, want %q", stdout.String(), wantStdout)
	}
	if logPath != logsOnlyLogFile(t, home) {
		t.Fatalf("stdout log path %q is not the created log", logPath)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	transcript := strings.TrimSuffix(stdout.String(), "Log: "+logPath+"\n")
	wantLog := "Started: <time>\nCommand: restore " + id + "\nSource: " + repo + "\nHome: " + home + "\n" +
		transcript + "Result: success\nFinished: <time>\n"
	if got := logTimestampRE.ReplaceAllString(string(data), "<time>"); got != wantLog {
		t.Fatalf("log = %q, want %q", got, wantLog)
	}
}

func logsOnlyBackupID(t *testing.T, repo string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("backups = %v, %v", entries, err)
	}
	return entries[0].Name()
}

func logsRemoveAll(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

// TestApplyPackFailurePreservesPartialTranscript proves the seam's read-only
// preflight does not lose the plan lines an apply that fails while planning
// historically wrote. A repo without opencode-pack fails after the core and
// orchestrate plan lines; stdout and the log must still carry them.
func TestApplyPackFailurePreservesPartialTranscript(t *testing.T) {
	repo, home := logsFixture(t, false)
	var stdout, stderr bytes.Buffer
	code := runEnv([]string{"apply", "--with-opencode", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80})
	if code != 1 {
		t.Fatalf("apply exit %d, want 1; stderr %q", code, stderr.String())
	}
	logPath := logsLineValue(t, stdout.String(), "Log: ")
	transcript := "install agents/agentic-sdd-plan\n" +
		"install codex/agentic-sdd-plan\n" +
		"install claude/agentic-sdd-plan\n" +
		"install agy/agentic-sdd-plan\n" +
		"install agents/agentic-sdd-orchestrate\n" +
		"install codex/agentic-sdd-orchestrate\n" +
		"install claude/agentic-sdd-orchestrate\n" +
		"install agy/agentic-sdd-orchestrate\n"
	if want := transcript + "Log: " + logPath + "\n"; stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if want := "error: open agents: no such file or directory\n"; stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	wantLog := "Started: <time>\nCommand: apply --with-opencode\nSource: " + repo + "\nHome: " + home + "\n" +
		transcript + "error: open agents: no such file or directory\nResult: failure\nFinished: <time>\n"
	if got := logTimestampRE.ReplaceAllString(string(data), "<time>"); got != wantLog {
		t.Fatalf("log = %q, want %q", got, wantLog)
	}
	if got := logsBackupCount(t, repo); got != 0 {
		t.Fatalf("pack failure created %d backups", got)
	}
}

// TestApplyAndRestoreTraceCarryStructuredBackupID proves op.result's
// backup_id comes from the structured result: it is the exact backup
// basename and appears after op/exit in the documented field order.
func TestApplyAndRestoreTraceCarryStructuredBackupID(t *testing.T) {
	t.Run("apply", func(t *testing.T) {
		repo, home := logsFixture(t, true)
		var traceBuf, stdout, stderr bytes.Buffer
		tracer := newTracer(&traceBuf, nil, true)
		code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80, Trace: tracer})
		if code != 0 {
			t.Fatalf("apply exit %d: %s", code, stderr.String())
		}
		backupPath := logsLineValue(t, stdout.String(), "Installed 4 skills; backup: ")
		backupID := filepath.Base(backupPath)
		line := logsOpResultLine(t, traceBuf.String())
		for _, want := range []string{`"op":"apply"`, `"exit":0`, `"backup_id":"` + backupID + `"`} {
			if !strings.Contains(line, want) {
				t.Fatalf("op.result %q missing %q", line, want)
			}
		}
		if !(strings.Index(line, `"op":"apply"`) < strings.Index(line, `"exit":0`) &&
			strings.Index(line, `"exit":0`) < strings.Index(line, `"backup_id"`)) {
			t.Fatalf("op.result fields are out of order: %q", line)
		}
		if err := skillsync.ValidateBackupID(backupID); err != nil {
			t.Fatalf("backup_id %q: %v", backupID, err)
		}
	})

	t.Run("restore", func(t *testing.T) {
		repo, home := logsFixture(t, true)
		var setupOut, setupErr bytes.Buffer
		if code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &setupOut, &setupErr, strings.NewReader(""), &Env{Width: 80}); code != 0 {
			t.Fatalf("setup apply exit %d: %s", code, setupErr.String())
		}
		id := logsOnlyBackupID(t, repo)
		var traceBuf, stdout, stderr bytes.Buffer
		tracer := newTracer(&traceBuf, nil, true)
		code := runEnv([]string{"restore", "apply", id, "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80, Trace: tracer})
		if code != 0 {
			t.Fatalf("restore exit %d: %s", code, stderr.String())
		}
		backupPath := logsLineValue(t, stdout.String(), "Restored 4 skills; backup: ")
		backupID := filepath.Base(backupPath)
		line := logsOpResultLine(t, traceBuf.String())
		for _, want := range []string{`"op":"restore"`, `"exit":0`, `"backup_id":"` + backupID + `"`} {
			if !strings.Contains(line, want) {
				t.Fatalf("op.result %q missing %q", line, want)
			}
		}
		if backupID == id {
			t.Fatalf("restore trace reported the restored backup %q instead of the pre-restore backup", id)
		}
	})
}

// TestApplyFailureBeforeBackupTraceOmitsBackupID pins the failure branch: the
// exact error/stdout bytes are kept and op.result carries no backup_id
// because no backup exists.
func TestApplyFailureBeforeBackupTraceOmitsBackupID(t *testing.T) {
	repo, home := logsFixture(t, false)
	target := filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	var traceBuf, stdout, stderr bytes.Buffer
	tracer := newTracer(&traceBuf, nil, true)
	code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80, Trace: tracer})
	if code != 1 {
		t.Fatalf("apply exit %d, want 1; stderr %q", code, stderr.String())
	}
	logPath := logsLineValue(t, stdout.String(), "Log: ")
	if want := "Log: " + logPath + "\n"; stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if want := "error: refusing non-directory target " + target + "\n"; stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
	line := logsOpResultLine(t, traceBuf.String())
	if strings.Contains(line, "backup_id") {
		t.Fatalf("failure before backup reported backup_id: %q", line)
	}
	if !strings.Contains(line, `"exit":1`) {
		t.Fatalf("op.result = %q, want exit 1", line)
	}
	if got := logsBackupCount(t, repo); got != 0 {
		t.Fatalf("failure before backup created %d backups", got)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	wantLog := "Started: <time>\nCommand: apply\nSource: " + repo + "\nHome: " + home + "\n" +
		"error: refusing non-directory target " + target + "\nResult: failure\nFinished: <time>\n"
	if got := logTimestampRE.ReplaceAllString(string(data), "<time>"); got != wantLog {
		t.Fatalf("log = %q, want %q", got, wantLog)
	}
}

// TestApplyPostBackupFailureReportsBackupID pins a staging failure that
// happens after a complete backup: the exit code and log stay honest, and
// op.result still reports the real backup identity so recovery is possible.
func TestApplyPostBackupFailureReportsBackupID(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based staging failure cannot be provoked as root")
	}
	repo, home := logsFixture(t, false)
	locked := filepath.Join(repo, "skills-files", "agentic-sdd-plan", "locked.md")
	if err := os.WriteFile(locked, []byte("locked"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0644) })

	var traceBuf, stdout, stderr bytes.Buffer
	tracer := newTracer(&traceBuf, nil, true)
	code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader(""), &Env{Width: 80, Trace: tracer})
	if code != 1 {
		t.Fatalf("apply exit %d, want 1; stderr %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	line := logsOpResultLine(t, traceBuf.String())
	backupID := ""
	if match := regexp.MustCompile(`"backup_id":"([^"]+)"`).FindStringSubmatch(line); match != nil {
		backupID = match[1]
	}
	if backupID == "" {
		t.Fatalf("post-backup failure did not report backup_id: %q", line)
	}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		t.Fatalf("backup_id %q: %v", backupID, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups", backupID, "manifest.json")); err != nil {
		t.Fatalf("reported backup is not complete: %v", err)
	}
	logPath := logsLineValue(t, stdout.String(), "Log: ")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Result: failure") {
		t.Fatalf("failure log = %q", data)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")); !os.IsNotExist(err) {
		t.Fatalf("failed staging installed a target: %v", err)
	}
}
