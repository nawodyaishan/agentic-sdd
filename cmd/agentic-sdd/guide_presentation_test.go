package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"agentic-sdd/internal/skillsync"
)

func presentationLineWithPrefix(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func presentationAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// TestGuidePresentationEmbeddedPackReview covers the terminal review for the
// embedded source with the optional pack and Free-first presets: source-derived
// unique/repeated counts, friendly labels, actual destinations and the
// review-with-back prompt. It declines, so nothing is written.
func TestGuidePresentationEmbeddedPackReview(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home with spaces-日本語's")
	env := &Env{Interactive: true, StdoutTTY: true, Width: 100}
	code, out, errOut := runBrandedGuide(t, "", home, "1\ny\ny\nn\n", env, brandingRuntime("1.2.3", nil))
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	absHome := presentationAbs(t, home)
	for _, want := range []string{
		"1. Choose options",
		"2. Review changes",
		"Target home: " + absHome,
		"A skill is reusable guidance",
		"A profile is a",
		"client-specific agent configuration file, not a running process",
		"sdd-implement-1",
		"sdd-implement-2",
		"sdd-test-implement-1",
		"sdd-test-implement-2",
		"does not authenticate providers",
		"guarantees neither free",
		"future free-model availability",
		"Destination",
		"New",
		"Replace",
		"Unchanged",
		"Shared agents",
		"Codex",
		"Claude Code",
		"Antigravity",
		"OpenCode profiles",
		"19 unique skills across four destinations: 76 entries",
		"18 profile files: 18 entries",
		"94 managed entries: 94 will change, 0 already match.",
		"Install these changes? [y/N/b=change options]:",
		filepath.Join(absHome, ".agents", "skills"),
		filepath.Join(absHome, ".codex", "skills"),
		filepath.Join(absHome, ".claude", "skills"),
		filepath.Join(absHome, ".gemini", "antigravity-cli", "skills"),
		filepath.Join(absHome, ".config", "opencode", "agents"),
		"asdx-core preview --with-opencode --opencode-free-first",
		"asdx-core apply --with-opencode --opencode-free-first",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "agy") {
		t.Errorf("review leaked the internal agy client label:\n%s", out)
	}
	if strings.Contains(out, "~/.agents/skills") || strings.Contains(out, "~/.claude/skills") {
		t.Errorf("review printed misleading ~/ destinations for an explicit home:\n%s", out)
	}
	optionsAt := strings.Index(out, "OpenCode profiles: "+filepath.Join(absHome, ".config", "opencode", "agents"))
	packQuestionAt := strings.Index(out, "Add the OpenCode orchestration pack")
	freeQuestionAt := strings.Index(out, "Prefer Free models first")
	freeExplainAt := strings.Index(out, "does not authenticate providers")
	if optionsAt < 0 || packQuestionAt < 0 || freeQuestionAt < 0 || freeExplainAt < 0 {
		t.Fatalf("missing stage-one explanation or questions:\n%s", out)
	}
	if optionsAt > packQuestionAt {
		t.Errorf("profile destination was not shown before the OpenCode question:\n%s", out)
	}
	if freeExplainAt > freeQuestionAt {
		t.Errorf("Free-first effects were not explained before its question:\n%s", out)
	}
	assertNoWrites(t, home)
}

// TestGuidePresentationCustomSingleSkill checks that a custom one-skill source
// derives four skill entries (not a hardcoded total), reports core-only
// honestly, and quotes commands for a repo/home containing spaces, an
// apostrophe and non-ASCII text.
func TestGuidePresentationCustomSingleSkill(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo with spaces-日本語's")
	home := filepath.Join(root, "home with spaces-日本語's")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	absRepo, absHome := presentationAbs(t, repo), presentationAbs(t, home)

	env := &Env{Interactive: true, StdoutTTY: true, Width: 100}
	code, out, errOut := runBrandedGuide(t, repo, home, "1\nn\nn\n", env, brandingRuntime("1.0.0", nil))
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q\n%s", code, errOut, out)
	}
	for _, want := range []string{
		"1 unique skill across four destinations: 4 entries",
		"Core-only selection: no profiles selected.",
		"4 managed entries: 4 will change, 0 already match.",
		"Shared agents: " + filepath.Join(absHome, ".agents", "skills"),
		"Codex: " + filepath.Join(absHome, ".codex", "skills"),
		"Claude Code: " + filepath.Join(absHome, ".claude", "skills"),
		"Antigravity: " + filepath.Join(absHome, ".gemini", "antigravity-cli", "skills"),
		"asdx-core preview --repo " + shellQuoteIfNeeded(absRepo) + " --home " + shellQuoteIfNeeded(absHome),
		"asdx-core apply --repo " + shellQuoteIfNeeded(absRepo) + " --home " + shellQuoteIfNeeded(absHome),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "asdx-core preview --with-opencode") {
		t.Errorf("core-only preview command carried pack flags:\n%s", out)
	}
	if strings.Contains(out, "~/.claude/skills") {
		t.Errorf("review printed a misleading ~/ destination:\n%s", out)
	}
	assertNoWrites(t, home)
}

// TestGuidePresentationNarrowRecords checks the below-80-column fallback:
// labeled records instead of an alignment-dependent table, with no path loss.
func TestGuidePresentationNarrowRecords(t *testing.T) {
	repo, home := guideFixture(t)
	absHome := presentationAbs(t, home)
	env := &Env{Interactive: true, StdoutTTY: true, Width: 79}
	code, out, errOut := runBrandedGuide(t, repo, home, "1\nn\nn\n", env, brandingRuntime("1.0.0", nil))
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	if strings.Contains(out, "New  Replace  Unchanged") {
		t.Errorf("narrow review used the wide table:\n%s", out)
	}
	for _, want := range []string{
		"Shared agents\n  New: 1\n  Replace: 0\n  Unchanged: 0\n",
		"Shared agents: " + filepath.Join(absHome, ".agents", "skills"),
		"1 unique skill across four destinations: 4 entries",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("narrow review missing %q in:\n%s", want, out)
		}
	}
	assertNoWrites(t, home)
}

// TestGuidePresentationReviewBackThenDecline proves b loops to options for the
// same home/repo, re-renders a fresh preview, writes nothing, and traces the
// review-back without recording a write approval.
func TestGuidePresentationReviewBackThenDecline(t *testing.T) {
	repo, home := guideFixture(t)
	var traceBuf bytes.Buffer
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80, Trace: newTracer(&traceBuf, nil, true)}
	code, out, errOut := runBrandedGuide(t, repo, home, "1\nn\nb\nn\nn\n", env, brandingRuntime("1.0.0", nil))
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	if got := strings.Count(out, "1. Choose options"); got != 2 {
		t.Errorf("options shown %d times, want 2:\n%s", got, out)
	}
	if got := strings.Count(out, "2. Review changes"); got != 2 {
		t.Errorf("review shown %d times, want 2:\n%s", got, out)
	}
	if !strings.Contains(out, "Install these changes? [y/N/b=change options]:") {
		t.Fatalf("review prompt missing:\n%s", out)
	}
	if !strings.Contains(out, "Cancelled.") {
		t.Fatalf("decline did not cancel:\n%s", out)
	}
	assertNoWrites(t, home)

	sawBack, sawConfirmNo := false, false
	for _, event := range decodeTrace(t, traceBuf.String()) {
		if event["event"] == "op.result" {
			t.Fatalf("declined review recorded an operation: %v", event)
		}
		if event["event"] == "prompt.answered" && event["prompt"] == "install.confirm" &&
			event["kind"] == "choice" && event["choice"] == "b" {
			sawBack = true
		}
		if event["event"] == "confirm" && event["prompt"] == "install.confirm" && event["decision"] == "no" {
			sawConfirmNo = true
		}
	}
	if !sawBack || !sawConfirmNo {
		t.Fatalf("trace did not record review-back and confirmation no:\n%s", traceBuf.String())
	}
}

// TestGuidePresentationReviewBackThenConfirm installs only after the second
// review and renders a truthful receipt from the operation transcript with a
// context-correct recovery preview and no post-success question.
func TestGuidePresentationReviewBackThenConfirm(t *testing.T) {
	repo, home := guideFixture(t)
	absRepo, absHome := presentationAbs(t, repo), presentationAbs(t, home)
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
	code, out, errOut := runBrandedGuide(t, repo, home, "1\nn\nb\nn\ny\n", env, brandingRuntime("1.0.0", nil))
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	if got := strings.Count(out, "1. Choose options"); got != 2 {
		t.Errorf("options shown %d times, want 2:\n%s", got, out)
	}
	for _, want := range []string{
		"Success:",
		"Installed / updated",
		"Installed 4 managed entries",
		"Desired managed entries: 4",
		"Skills by purpose",
		"Profiles",
		"none selected (core-only install)",
		"Recovery",
		"Preview restoring this snapshot",
		"Log: ",
		"Next steps",
		"agentic-sdd-router",
		"Plan a feature:",
		"Fix an existing bug:",
		"then choose 6",
		"asdx-core guide --home " + shellQuoteIfNeeded(absHome) + " --repo " + shellQuoteIfNeeded(absRepo),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("receipt missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "nil-pointer") {
		t.Errorf("terminal receipt still directs users to a fictional nil-pointer bug:\n%s", out)
	}
	if strings.Contains(out, "✔ ok") {
		t.Errorf("receipt reused the operation status prefix:\n%s", out)
	}

	backupLine := presentationLineWithPrefix(out, "Snapshot: ")
	if backupLine == "" {
		t.Fatalf("receipt lost the snapshot line:\n%s", out)
	}
	id := strings.TrimPrefix(backupLine, "Snapshot: ")
	if err := skillsync.ValidateBackupID(id); err != nil {
		t.Fatalf("snapshot id %q is not whole: %v", id, err)
	}
	locationLine := presentationLineWithPrefix(out, "Location: ")
	if locationLine != "Location: "+filepath.Join(absRepo, "backups", id) {
		t.Fatalf("snapshot location = %q, want %q", locationLine, "Location: "+filepath.Join(absRepo, "backups", id))
	}
	recovery := "asdx-core restore " + id + " --home " + shellQuoteIfNeeded(absHome) + " --repo " + shellQuoteIfNeeded(absRepo)
	if !strings.Contains(out, recovery) {
		t.Errorf("recovery preview missing %q in:\n%s", recovery, out)
	}
	if strings.Contains(out, "restore "+id+" --apply") || strings.Contains(out, "restore --apply") {
		t.Errorf("recovery preview applied instead of previewing:\n%s", out)
	}

	logLine := presentationLineWithPrefix(out, "Log: ")
	if logLine == "" {
		t.Fatalf("receipt lost the log line:\n%s", out)
	}
	if _, err := os.Stat(strings.TrimPrefix(logLine, "Log: ")); err != nil {
		t.Fatalf("log path is not complete: %v", err)
	}
	backups, err := skillsync.ListBackups(repo, home)
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, %v; want one", backups, err)
	}
	logs, err := os.ReadDir(logsDir(home))
	if err != nil || len(logs) != 1 {
		t.Fatalf("logs = %v, %v; want one", logs, err)
	}
}

// TestGuidePresentationReviewBackChangesOptions proves b re-asks the options
// and recomputes a different preview before any write. The first review selects
// the pack; the second deliberately drops it, and only that second selection is
// installed after a fresh confirmation.
func TestGuidePresentationReviewBackChangesOptions(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	coreInventory, err := skillsync.InspectSelection("", home, skillsync.Options{})
	if err != nil {
		t.Fatal(err)
	}
	coreEntries := len(coreInventory.SkillIDs) * 4
	env := &Env{Interactive: true, StdoutTTY: true, Width: 100}
	code, out, errOut := runBrandedGuide(t, "", home, "1\ny\ny\nb\nn\ny\n", env, brandingRuntime("1.0.0", nil))
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	for _, want := range []string{
		"94 managed entries: 94 will change, 0 already match.",
		fmt.Sprintf("%d managed entries: %d will change, 0 already match.", coreEntries, coreEntries),
		"Core-only selection: no profiles selected.",
		fmt.Sprintf("Installed %d managed entries.", coreEntries),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	backups, err := skillsync.ListBackups("", home)
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, %v; want exactly one install", backups, err)
	}
}

// TestGuidePresentationFailureGuidance proves a failed apply never renders a
// success receipt or claims rollback, and still names the operation log.
func TestGuidePresentationFailureGuidance(t *testing.T) {
	repo, home := guideFixture(t)
	if err := os.WriteFile(filepath.Join(repo, "backups"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
	code, out, errOut := runBrandedGuide(t, repo, home, "1\nn\ny\n", env, brandingRuntime("1.0.0", nil))
	if code != 1 {
		t.Fatalf("failed apply exit = %d, want 1; stdout %q stderr %q", code, out, errOut)
	}
	for _, want := range []string{
		"Installation failed",
		"Installation failed; inspect the operation log before retrying.",
		"No success was recorded.",
		"Log: ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("failure guidance missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Success:") {
		t.Errorf("failed apply claimed success:\n%s", out)
	}
	if strings.Contains(out, "rolled back successfully") || strings.Contains(out, "all changes were reversed") {
		t.Errorf("failed apply claimed rollback:\n%s", out)
	}
	logLine := presentationLineWithPrefix(out, "Log: ")
	if logLine == "" {
		t.Fatalf("failure guidance lost the log path:\n%s", out)
	}
	if _, err := os.Stat(strings.TrimPrefix(logLine, "Log: ")); err != nil {
		t.Fatalf("failure log path is not complete: %v", err)
	}
}

// TestGuidePresentationNoOpCreatesNothing proves a guided no-op prints an
// up-to-date result without a confirmation, operation log or snapshot.
func TestGuidePresentationNoOpCreatesNothing(t *testing.T) {
	repo, home, _ := setupBackupCLI(t)
	backupsBefore, err := skillsync.ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	logsBefore, err := os.ReadDir(logsDir(home))
	if err != nil {
		t.Fatal(err)
	}

	env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
	code, out, errOut := runBrandedGuide(t, repo, home, "1\nn\n", env, brandingRuntime("1.0.0", nil))
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	if !strings.Contains(out, "Already up to date") {
		t.Fatalf("no-op did not report already up to date:\n%s", out)
	}
	if strings.Contains(out, "Install these changes?") {
		t.Fatalf("no-op asked for write confirmation:\n%s", out)
	}
	backupsAfter, err := skillsync.ListBackups(repo, home)
	if err != nil || len(backupsAfter) != len(backupsBefore) {
		t.Fatalf("no-op changed backups: before %v after %v err %v", backupsBefore, backupsAfter, err)
	}
	logsAfter, err := os.ReadDir(logsDir(home))
	if err != nil || len(logsAfter) != len(logsBefore) {
		t.Fatalf("no-op changed logs: before %v after %v err %v", logsBefore, logsAfter, err)
	}
}

// TestGuidePresentationLogsSafetyError proves the terminal log action only
// treats genuinely missing/empty logs as friendly; a refused path is an error
// with exit 1 and is never swallowed as "No logs found".
func TestGuidePresentationLogsSafetyError(t *testing.T) {
	repo, _ := guideFixture(t)
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".agentic-sdd"), 0755); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(home, ".agentic-sdd", "logs")); err != nil {
		t.Fatal(err)
	}
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
	code, out, errOut := runBrandedGuide(t, repo, home, "3\n", env, brandingRuntime("1.0.0", nil))
	if code != 1 {
		t.Fatalf("refused log path exit = %d, want 1; stdout %q stderr %q", code, out, errOut)
	}
	if !strings.Contains(errOut, "error:") || !strings.Contains(errOut, "refusing non-directory log path") {
		t.Fatalf("safety error was not reported on stderr: %q", errOut)
	}
	if strings.Contains(out, "No install or restore logs found yet.") {
		t.Fatalf("safety error was swallowed as an empty state:\n%s", out)
	}
}

// shTokens is a deliberately independent POSIX-ish tokenizer used only to
// prove shellQuoteIfNeeded round-trips a value. It accepts the quoting and
// backslash escaping the tool emits, so a second quoting bug cannot hide
// behind comparing the renderer with its own helper.
func shTokens(t *testing.T, command string) []string {
	t.Helper()
	var (
		tokens  []string
		current strings.Builder
		have    bool
		quoted  bool
	)
	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == '\'':
			quoted = !quoted
			have = true
		case r == '\\' && !quoted && i+1 < len(runes):
			i++
			current.WriteRune(runes[i])
			have = true
		case (r == ' ' || r == '\t') && !quoted:
			if have {
				tokens = append(tokens, current.String())
				current.Reset()
				have = false
			}
		default:
			current.WriteRune(r)
			have = true
		}
	}
	if quoted {
		t.Fatalf("unbalanced quotes in %q", command)
	}
	if have {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// TestGuideSelectionCommandQuotingRoundTrip proves the emitted commands parse
// back to the exact selected values, including spaces, apostrophes and
// non-ASCII characters, without executing them.
func TestGuideSelectionCommandQuotingRoundTrip(t *testing.T) {
	cases := []struct{ home, repo string }{
		{"/tmp/plain/home", ""},
		{"/tmp/home with spaces", "/tmp/repo with spaces"},
		{"/tmp/it's here", "/tmp/o'brien repo"},
		{"/tmp/日本語 home", "/tmp/日本語-repo"},
	}
	for _, tc := range cases {
		opts := skillsync.Options{WithOpenCode: true, OpenCodeFreeFirst: true}
		for _, command := range []string{
			guidePreviewCommand(tc.home, tc.repo, opts),
			guideApplyCommand(tc.home, tc.repo, opts),
		} {
			tokens := shTokens(t, command)
			values := map[string]string{}
			for i := 0; i+1 < len(tokens); i++ {
				if strings.HasPrefix(tokens[i], "--") {
					values[strings.TrimPrefix(tokens[i], "--")] = tokens[i+1]
				}
			}
			if values["home"] != tc.home {
				t.Errorf("%q parsed home %q, want %q", command, values["home"], tc.home)
			}
			if tc.repo != "" && values["repo"] != tc.repo {
				t.Errorf("%q parsed repo %q, want %q", command, values["repo"], tc.repo)
			}
			if _, ok := values["with-opencode"]; !ok {
				t.Errorf("%q lost --with-opencode", command)
			}
			if _, ok := values["opencode-free-first"]; !ok {
				t.Errorf("%q lost --opencode-free-first", command)
			}
		}
	}
}

// definitionOutput renders one definition field and returns its lines. It fails
// on empty or unterminated output so a renderer that drops the trailing newline
// cannot pass by accident.
func definitionOutput(t *testing.T, label, text string, width int) []string {
	t.Helper()
	var buf bytes.Buffer
	printDefinition(&buf, label, text, width)
	rendered := buf.String()
	if !strings.HasSuffix(rendered, "\n") {
		t.Fatalf("printDefinition(%q, width=%d) rendered %q without a trailing newline", label, width, rendered)
	}
	return strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")
}

// assertDefinitionFits proves every rendered line stays inside the width the
// caller budgeted for it.
func assertDefinitionFits(t *testing.T, label string, lines []string, width int) {
	t.Helper()
	for i, line := range lines {
		if got := utf8.RuneCountInString(line); got > width {
			t.Errorf("printDefinition(%q, width=%d) line %d is %d runes, want at most %d: %q", label, width, i, got, width, line)
		}
	}
}

// assertDefinitionLabelOnce proves the definition term names the field once and
// is not repeated as a continuation prefix.
func assertDefinitionLabelOnce(t *testing.T, label string, lines []string) {
	t.Helper()
	term := strings.TrimRight(label, " ")
	if term == "" {
		t.Fatalf("test label %q has no definition term", label)
	}
	if got := strings.Count(strings.Join(lines, "\n"), term); got != 1 {
		t.Errorf("definition term %q appears %d times, want 1:\n%s", term, got, strings.Join(lines, "\n"))
	}
}

// definitionBody rejoins a rendered field into its text, dropping the label and
// each line's indentation, so a lost, duplicated or reordered word is visible.
func definitionBody(label, indent string, lines []string) string {
	parts := make([]string, 0, len(lines))
	for i, line := range lines {
		switch {
		case i == 0:
			parts = append(parts, strings.TrimPrefix(line, label))
		default:
			parts = append(parts, strings.TrimPrefix(line, indent))
		}
	}
	return strings.Join(parts, " ")
}

// TestGuidePresentationDefinitionHangingIndent proves a short label keeps the
// first text line beside it and indents every later line by exactly the label
// width, at the widths the guide budgets for, without repeating the label.
func TestGuidePresentationDefinitionHangingIndent(t *testing.T) {
	text := "Fix the failing parser test. Keep the diff minimal, name the evidence before changing code, and report what remains uncertain."
	for _, width := range []int{40, 60, 80} {
		for _, label := range []string{"Purpose: ", "Use when: ", "Safety: ", "Example: "} {
			t.Run(fmt.Sprintf("%s/%d", strings.TrimSpace(label), width), func(t *testing.T) {
				if utf8.RuneCountInString(label) > guideDefinitionHangingMax {
					t.Fatalf("label %q is not a short label", label)
				}
				lines := definitionOutput(t, label, text, width)
				if len(lines) < 2 {
					t.Fatalf("label %q at width %d produced %d lines, want a wrapped field:\n%s", label, width, len(lines), strings.Join(lines, "\n"))
				}
				indent := strings.Repeat(" ", utf8.RuneCountInString(label))
				if !strings.HasPrefix(lines[0], label) {
					t.Errorf("first line %q does not start with label %q", lines[0], label)
				}
				if strings.TrimPrefix(lines[0], label) == "" {
					t.Errorf("first line %q carried no text", lines[0])
				}
				for i, line := range lines[1:] {
					if !strings.HasPrefix(line, indent) {
						t.Errorf("continuation line %d %q is not indented by %d spaces", i+1, line, len(indent))
					}
					if strings.TrimSpace(strings.TrimPrefix(line, indent)) == "" {
						t.Errorf("continuation line %d %q carried no text", i+1, line)
					}
				}
				if got := definitionBody(label, indent, lines); got != text {
					t.Errorf("label %q at width %d rejoined to\n%q\nwant\n%q", label, width, got, text)
				}
				assertDefinitionLabelOnce(t, label, lines)
				assertDefinitionFits(t, label, lines, width)
			})
		}
	}
}

// TestGuidePresentationDefinitionBlockFormat proves a long label becomes its own
// header line and the text follows as a two-space indented block, so even a
// label wider than the hanging-indent budget cannot squeeze the text.
func TestGuidePresentationDefinitionBlockFormat(t *testing.T) {
	text := "Declared intent is educational. It is not an enforced guarantee, and the text is never used to imply that a model, provider or runtime was contacted."
	for _, width := range []int{40, 60, 80} {
		for _, label := range []string{
			"Ask your main agent: ",
			"Destination file status: ",
		} {
			t.Run(fmt.Sprintf("%s/%d", strings.TrimSpace(label), width), func(t *testing.T) {
				if utf8.RuneCountInString(label) <= guideDefinitionHangingMax {
					t.Fatalf("label %q is not a long label", label)
				}
				lines := definitionOutput(t, label, text, width)
				if len(lines) < 2 {
					t.Fatalf("label %q at width %d produced %d lines, want a header and a block:\n%s", label, width, len(lines), strings.Join(lines, "\n"))
				}
				if got := strings.TrimRight(lines[0], " "); got != strings.TrimRight(label, " ") {
					t.Errorf("header line %q does not name label %q alone", lines[0], label)
				}
				if strings.HasPrefix(lines[0], " ") {
					t.Errorf("header line %q is indented", lines[0])
				}
				for i, line := range lines[1:] {
					if !strings.HasPrefix(line, "  ") {
						t.Errorf("block line %d %q lacks the two-space indent", i+1, line)
					}
					if strings.HasPrefix(strings.TrimPrefix(line, "  "), " ") {
						t.Errorf("block line %d %q is indented beyond two spaces", i+1, line)
					}
				}
				parts := make([]string, 0, len(lines)-1)
				for _, line := range lines[1:] {
					parts = append(parts, strings.TrimPrefix(line, "  "))
				}
				if got := strings.Join(parts, " "); got != text {
					t.Errorf("label %q at width %d rejoined to\n%q\nwant\n%q", label, width, got, text)
				}
				assertDefinitionLabelOnce(t, label, lines)
				assertDefinitionFits(t, label, lines, width)
			})
		}
	}
}

// TestGuidePresentationDefinitionLabelWiderThanWidth pins the one width
// exception: a definition term is never split, so a label wider than the column
// budget is printed whole on its own line while the body still fits. DEF-01 was
// exactly this defect — a 57-rune label overflowed a 40-column terminal — and the
// remedy is to shorten such labels at the call site (to "Safety:"), not to break
// a term mid-word.
func TestGuidePresentationDefinitionLabelWiderThanWidth(t *testing.T) {
	text := "Declared intent is educational and never implies a runtime guarantee."
	const width = 40
	label := "Declared intent (educational; not an enforced guarantee): "
	if utf8.RuneCountInString(label) <= width {
		t.Fatalf("label %q is only %d runes, so this case no longer covers an over-wide label", label, utf8.RuneCountInString(label))
	}
	lines := definitionOutput(t, label, text, width)
	if len(lines) < 2 {
		t.Fatalf("label %q rendered %d lines, want a header and a block:\n%s", label, len(lines), strings.Join(lines, "\n"))
	}
	if lines[0] != label {
		t.Errorf("header line %q was altered, want the whole label %q", lines[0], label)
	}
	parts := make([]string, 0, len(lines)-1)
	for i, line := range lines[1:] {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("block line %d %q lacks the two-space indent", i+1, line)
		}
		if got := utf8.RuneCountInString(line); got > width {
			t.Errorf("block line %d is %d runes, want at most %d: %q", i+1, got, width, line)
		}
		parts = append(parts, strings.TrimPrefix(line, "  "))
	}
	if got := strings.Join(parts, " "); got != text {
		t.Errorf("block body rejoined to\n%q\nwant\n%q", got, text)
	}
	assertDefinitionLabelOnce(t, label, lines)
}

// TestGuidePresentationDefinitionFieldsFitNarrowWidth renders the fields the
// detail screens actually use, including the shortened Safety: term, at 40, 60
// and 80 columns and asserts no rendered line overflows. It is the AC-01 guard:
// every definition label must fit the narrowest supported width, so a long
// label is shortened rather than hung or overflowed.
func TestGuidePresentationDefinitionFieldsFitNarrowWidth(t *testing.T) {
	const narrowest = 40
	fields := []struct{ label, text string }{
		{"Purpose: ", "Drive routine implementation with a Free-first model order."},
		{"Use when: ", "You want bounded, read-only discovery across several files before the main agent commits context."},
		{"Ask your main agent: ", "Use `agentic-sdd-implement` to fix this bug and report the focused test results."},
		{"Safety: ", "Declared intent is educational; it is not an enforced guarantee, and no provider is contacted."},
	}
	for _, field := range fields {
		if got := utf8.RuneCountInString(field.label); got >= narrowest {
			t.Errorf("label %q is %d runes and cannot fit a %d-column line; shorten it instead", field.label, got, narrowest)
		}
	}
	for _, width := range []int{40, 60, 80} {
		for _, field := range fields {
			lines := definitionOutput(t, field.label, field.text, width)
			assertDefinitionFits(t, field.label, lines, width)
			assertDefinitionLabelOnce(t, field.label, lines)
		}
	}
}

// TestGuidePresentationDefinitionEmptyText proves a field with no text still
// names itself once instead of vanishing or printing a blank value line.
func TestGuidePresentationDefinitionEmptyText(t *testing.T) {
	for _, width := range []int{40, 60, 80} {
		for _, label := range []string{"Safety: ", "Destination file status: "} {
			lines := definitionOutput(t, label, "", width)
			if len(lines) != 1 {
				t.Fatalf("label %q at width %d rendered %d lines, want the label alone:\n%s", label, width, len(lines), strings.Join(lines, "\n"))
			}
			if got := strings.TrimRight(lines[0], " "); got != strings.TrimRight(label, " ") {
				t.Errorf("label %q at width %d rendered %q, want the label alone", label, width, lines[0])
			}
			assertDefinitionFits(t, label, lines, width)
		}
	}
}
