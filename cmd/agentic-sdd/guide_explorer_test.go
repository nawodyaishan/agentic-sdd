package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// newExplorerFixture builds a temporary custom source and empty selected home.
// Core skills default to one known and one unknown ID; callers may pass an
// explicit set for deterministic single-item navigation.
func newExplorerFixture(t *testing.T, coreIDs ...string) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	home = filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	if len(coreIDs) == 0 {
		coreIDs = []string{"agentic-sdd-spec", "agentic-sdd-custom"}
	}
	for _, id := range coreIDs {
		writeExplorerSkill(t, repo, id)
	}
	copyExplorerPackFile(t, repo, "agents/sdd-orchestrator.md")
	copyExplorerPackFile(t, repo, "agents/sdd-debug-diagnose.md")
	copyExplorerPackFile(t, repo, "presets/free-first/sdd-implement-1.md")
	copyExplorerPackFile(t, repo, "skills/agentic-sdd-orchestrate/SKILL.md")
	return repo, home
}

func writeExplorerSkill(t *testing.T, repo, id string) {
	t.Helper()
	dir := filepath.Join(repo, "skills-files", id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + id + "\ndescription: explorer fixture\n---\nfixture body\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// copyExplorerPackFile copies a real pack asset so inventory validation sees
// the same frontmatter it would in a repository checkout.
func copyExplorerPackFile(t *testing.T, repo, rel string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "opencode-pack", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read real pack asset %s: %v", rel, err)
	}
	dst := filepath.Join(repo, "opencode-pack", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func explorerClientRoot(home, client string) string {
	switch client {
	case "agents":
		return filepath.Join(home, ".agents", "skills")
	case "codex":
		return filepath.Join(home, ".codex", "skills")
	case "claude":
		return filepath.Join(home, ".claude", "skills")
	default:
		return filepath.Join(home, ".gemini", "antigravity-cli", "skills")
	}
}

// writeExplorerSkillTarget writes one destination skill directory. changed
// appends a byte so the inventory reports a source difference.
func writeExplorerSkillTarget(t *testing.T, repo, home, client, id string, changed bool) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(repo, "skills-files", id, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		source = append(source, []byte("changed\n")...)
	}
	dir := filepath.Join(explorerClientRoot(home, client), id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), source, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeExplorerProfileTarget(t *testing.T, repo, home, profile, preset string, changed bool) {
	t.Helper()
	rel := "agents/" + profile + ".md"
	if preset != "" {
		rel = "presets/free-first/" + profile + ".md"
	}
	source, err := os.ReadFile(filepath.Join(repo, "opencode-pack", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		source = append(source, []byte("changed\n")...)
	}
	dir := filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, profile+".md"), source, 0644); err != nil {
		t.Fatal(err)
	}
}

func runExplorer(t *testing.T, repo, home, input string, env *Env) (quit bool, code int, stdout, stderr string) {
	t.Helper()
	if env == nil {
		env = &Env{Interactive: true, StdoutTTY: true, StderrTTY: true, Width: 80}
	}
	var out, errBuf bytes.Buffer
	p := &prompter{in: bufio.NewReader(strings.NewReader(input)), out: &out, env: env}
	quit, code = runGuideExplorer(repo, home, p, &errBuf)
	return quit, code, out.String(), errBuf.String()
}

// numberedExplorerItems maps displayed IDs to their numeric list index.
func numberedExplorerItems(t *testing.T, out string) map[string]int {
	t.Helper()
	re := regexp.MustCompile(`(?m)^  ([0-9]+)\) ([^ ]+)`)
	items := map[string]int{}
	for _, m := range re.FindAllStringSubmatch(out, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("bad item number in %q: %v", m[0], err)
		}
		items[m[2]] = n
	}
	if len(items) == 0 {
		t.Fatalf("no numbered explorer items in output:\n%s", out)
	}
	return items
}

// treeState captures every path, mode, content and symlink target below root
// so a read-only explorer run can be proven to change nothing.
func treeState(t *testing.T, root string) map[string]string {
	t.Helper()
	state := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			state[rel] = "link:" + target
		case entry.IsDir():
			state[rel] = "dir"
		default:
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			state[rel] = "file:" + string(data)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return state
}

func TestGuideExplorerCoreDetailStatusesAndNavigation(t *testing.T) {
	repo, home := newExplorerFixture(t, "agentic-sdd-spec")
	writeExplorerSkillTarget(t, repo, home, "agents", "agentic-sdd-spec", false)
	writeExplorerSkillTarget(t, repo, home, "codex", "agentic-sdd-spec", true)
	refused := filepath.Join(home, ".claude", "skills", "agentic-sdd-spec")
	if err := os.MkdirAll(filepath.Dir(refused), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "not-a-skill"), refused); err != nil {
		t.Fatal(err)
	}
	beforeRepo, beforeHome := treeState(t, repo), treeState(t, home)

	quit, code, out, stderr := runExplorer(t, repo, home, "1\n1\n\nb\nq\n", nil)
	if !quit || code != 0 {
		t.Fatalf("quit=%v code=%d stderr=%q", quit, code, stderr)
	}
	for _, want := range []string{
		"agentic-sdd-spec",
		"Available in selected source",
		filepath.Join(home, ".agents", "skills", "agentic-sdd-spec"),
		"matches source",
		"differs from source",
		"missing",
		"refused",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("core detail missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"enabled:", "running:", "authenticated:", "verified working"} {
		if strings.Contains(strings.ToLower(out), forbidden) {
			t.Fatalf("core detail asserted runtime/health state %q:\n%s", forbidden, out)
		}
	}
	if got := treeState(t, repo); !equalTreeState(beforeRepo, got) {
		t.Fatal("explorer modified the repository source")
	}
	if got := treeState(t, home); !equalTreeState(beforeHome, got) {
		t.Fatal("explorer modified the selected home")
	}
}

func TestGuideExplorerProfilesIncludeBasesAndOptionalPresets(t *testing.T) {
	repo, home := newExplorerFixture(t)
	writeExplorerProfileTarget(t, repo, home, "sdd-orchestrator", "", false)
	writeExplorerProfileTarget(t, repo, home, "sdd-implement-1", "free-first", false)

	inv, err := skillsync.InspectSelection(repo, home, skillsync.Options{WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		t.Fatalf("profile inventory: %v", err)
	}
	if len(inv.ProfileIDs) < 2 {
		t.Fatalf("fixture profile inventory has %d IDs: %v", len(inv.ProfileIDs), inv.ProfileIDs)
	}

	_, code, listOut, stderr := runExplorer(t, repo, home, "2\nb\nq\n", nil)
	if code != 0 {
		t.Fatalf("list exit=%d stderr=%q", code, stderr)
	}
	items := numberedExplorerItems(t, listOut)
	for _, id := range inv.ProfileIDs {
		if _, ok := items[id]; !ok {
			t.Fatalf("profile %q not listed:\n%s", id, listOut)
		}
	}
	if !regexp.MustCompile(`Available in the selected source:\s+` + strconv.Itoa(len(inv.ProfileIDs)) + `\s+profile\s+templates\.`).MatchString(listOut) {
		t.Fatalf("profile source count not derived from inventory (%d):\n%s", len(inv.ProfileIDs), listOut)
	}
	if !strings.Contains(listOut, "optional preset") {
		t.Fatalf("optional preset profile not marked:\n%s", listOut)
	}

	idx, ok := items["sdd-implement-1"]
	if !ok {
		t.Fatalf("optional preset sdd-implement-1 missing:\n%s", listOut)
	}
	_, code, detailOut, stderr := runExplorer(t, repo, home,
		"2\n"+strconv.Itoa(idx)+"\n\nb\nq\n", nil)
	if code != 0 {
		t.Fatalf("detail exit=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{
		"sdd-implement-1",
		filepath.Join(home, ".config", "opencode", "agents", "sdd-implement-1.md"),
		"matches source",
		"optional preset",
	} {
		if !strings.Contains(detailOut, want) {
			t.Fatalf("preset detail missing %q:\n%s", want, detailOut)
		}
	}
}

func TestGuideExplorerMissingOptionalPackKeepsCoreAvailable(t *testing.T) {
	repo, home := newExplorerFixture(t, "agentic-sdd-spec")
	if err := os.RemoveAll(filepath.Join(repo, "opencode-pack")); err != nil {
		t.Fatal(err)
	}
	before := treeState(t, home)

	quit, code, out, stderr := runExplorer(t, repo, home, "2\n1\nb\nq\n", nil)
	if !quit || code != 0 {
		t.Fatalf("quit=%v code=%d stderr=%q", quit, code, stderr)
	}
	for _, want := range []string{"OpenCode profiles: missing", "OpenCode pack"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing pack notice missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "agentic-sdd-spec") {
		t.Fatalf("core category unavailable after optional pack failure:\n%s", out)
	}
	if !equalTreeState(before, treeState(t, home)) {
		t.Fatal("explorer modified the selected home")
	}
}

func TestGuideExplorerUnreadableOptionalPackIsUnavailable(t *testing.T) {
	repo, home := newExplorerFixture(t, "agentic-sdd-spec")
	if err := os.RemoveAll(filepath.Join(repo, "opencode-pack")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "opencode-pack"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	before := treeState(t, home)

	quit, code, out, stderr := runExplorer(t, repo, home, "2\n1\nb\nq\n", nil)
	if !quit || code != 0 {
		t.Fatalf("quit=%v code=%d stderr=%q", quit, code, stderr)
	}
	for _, want := range []string{"OpenCode profiles: unavailable", "could not be read safely"} {
		if !strings.Contains(out, want) {
			t.Fatalf("unreadable pack notice missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "agentic-sdd-spec") {
		t.Fatalf("core category unavailable after unreadable pack:\n%s", out)
	}
	if !equalTreeState(before, treeState(t, home)) {
		t.Fatal("explorer modified the selected home")
	}
}

func TestGuideExplorerUnknownCustomIDSafeFallback(t *testing.T) {
	repo, home := newExplorerFixture(t)
	_, code, listOut, stderr := runExplorer(t, repo, home, "1\nb\nq\n", nil)
	if code != 0 {
		t.Fatalf("list exit=%d stderr=%q", code, stderr)
	}
	idx, ok := numberedExplorerItems(t, listOut)["agentic-sdd-custom"]
	if !ok {
		t.Fatalf("custom skill not listed:\n%s", listOut)
	}
	_, code, detailOut, stderr := runExplorer(t, repo, home,
		"1\n"+strconv.Itoa(idx)+"\n\nq\n", nil)
	if code != 0 {
		t.Fatalf("detail exit=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(strings.ToLower(detailOut), "custom") {
		t.Fatalf("custom skill lacks generic fallback text:\n%s", detailOut)
	}
	if !strings.Contains(detailOut, "agentic-sdd-custom") {
		t.Fatalf("custom detail omits exact ID:\n%s", detailOut)
	}
	if strings.Contains(strings.ToLower(detailOut), "enabled:") ||
		strings.Contains(strings.ToLower(detailOut), "running:") {
		t.Fatalf("custom detail invented runtime state:\n%s", detailOut)
	}
}

func TestGuideExplorerConceptsStable(t *testing.T) {
	repo, home := newExplorerFixture(t, "agentic-sdd-spec")
	concepts := teachingConcepts()
	if len(concepts) == 0 {
		t.Fatal("teachingConcepts returned no concepts")
	}
	before := treeState(t, home)
	quit, code, out, stderr := runExplorer(t, repo, home, "3\n1\n\nb\nq\n", nil)
	if !quit || code != 0 {
		t.Fatalf("quit=%v code=%d stderr=%q", quit, code, stderr)
	}
	for _, want := range []string{
		concepts[0].ID,
		"No destination",
		"teaching",
	} {
		if !strings.Contains(strings.ToLower(out), strings.ToLower(want)) {
			t.Fatalf("concept screen missing %q:\n%s", want, out)
		}
	}
	if !equalTreeState(before, treeState(t, home)) {
		t.Fatal("concept screen modified the selected home")
	}
}

func TestGuideExplorerMainMenuReturnAndListEOF(t *testing.T) {
	repo, home := newExplorerFixture(t, "agentic-sdd-spec")
	quit, code, _, stderr := runExplorer(t, repo, home, "b\n", nil)
	if quit || code != 0 {
		t.Fatalf("b: quit=%v code=%d stderr=%q, want main-menu return", quit, code, stderr)
	}
	quit, code, _, stderr = runExplorer(t, repo, home, "1\n", nil)
	if !quit || code != 0 {
		t.Fatalf("list EOF: quit=%v code=%d stderr=%q, want exit-guide 0", quit, code, stderr)
	}
}

func TestGuideExplorerEOFAndInvalidInput(t *testing.T) {
	t.Run("eof at categories exits guide", func(t *testing.T) {
		repo, home := newExplorerFixture(t, "agentic-sdd-spec")
		quit, code, _, _ := runExplorer(t, repo, home, "", nil)
		if !quit || code != 0 {
			t.Fatalf("quit=%v code=%d, want exit-guide 0", quit, code)
		}
	})

	t.Run("eof at detail exits safely", func(t *testing.T) {
		repo, home := newExplorerFixture(t, "agentic-sdd-spec")
		quit, code, _, _ := runExplorer(t, repo, home, "1\n1\n", nil)
		if !quit || code != 0 {
			t.Fatalf("quit=%v code=%d, want exit-guide 0", quit, code)
		}
	})

	t.Run("invalid input reprompts then bounded exit", func(t *testing.T) {
		repo, home := newExplorerFixture(t, "agentic-sdd-spec")
		var trace bytes.Buffer
		env := &Env{Interactive: true, StdoutTTY: true, StderrTTY: true, Width: 80, Trace: newTracer(&trace, nil, true)}
		secret := "SECRET/raw/path"
		quit, code, out, stderr := runExplorer(t, repo, home, secret+"\n"+secret+"\n"+secret+"\n", env)
		if !quit || code != 2 {
			t.Fatalf("quit=%v code=%d, want bounded exit 2; stderr=%q", quit, code, stderr)
		}
		if !strings.Contains(stderr, "error:") {
			t.Fatalf("bounded invalid exit lacks safe error: %q", stderr)
		}
		if strings.Contains(out, secret) || strings.Contains(stderr, secret) || strings.Contains(trace.String(), secret) {
			t.Fatalf("invalid raw input was emitted")
		}
	})

	t.Run("invalid then valid recovers", func(t *testing.T) {
		repo, home := newExplorerFixture(t, "agentic-sdd-spec")
		quit, code, out, _ := runExplorer(t, repo, home, "zz\n1\nb\nq\n", nil)
		if !quit || code != 0 {
			t.Fatalf("quit=%v code=%d, want recovered exit 0", quit, code)
		}
		if !strings.Contains(out, "agentic-sdd-spec") {
			t.Fatalf("recovery did not reach the core list:\n%s", out)
		}
	})

	t.Run("blank list returns to categories", func(t *testing.T) {
		repo, home := newExplorerFixture(t, "agentic-sdd-spec")
		quit, code, out, _ := runExplorer(t, repo, home, "1\n\nq\n", nil)
		if !quit || code != 0 {
			t.Fatalf("quit=%v code=%d, want exit-guide 0", quit, code)
		}
		if strings.Count(out, "Explore skills and agent profiles") < 2 {
			t.Fatalf("blank list did not return to categories:\n%s", out)
		}
	})
}

func equalTreeState(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestGuideExplorerTraceIsBoundedAndPrivate(t *testing.T) {
	repo, home := newExplorerFixture(t)
	writeExplorerProfileTarget(t, repo, home, "sdd-orchestrator", "", false)
	var trace bytes.Buffer
	env := &Env{Interactive: true, StdoutTTY: true, StderrTTY: true, Width: 80, Trace: newTracer(&trace, nil, true)}

	// categories -> core list -> detail -> back -> categories -> profiles ->
	// back -> concepts -> detail -> back -> invalid -> quit.
	input := "1\n1\n\nb\n2\nb\n3\n1\n\nb\nSECRET/raw/path\nq\n"
	quit, code, _, stderr := runExplorer(t, repo, home, input, env)
	if !quit || code != 0 {
		t.Fatalf("quit=%v code=%d stderr=%q", quit, code, stderr)
	}

	allowedScreens := map[string]bool{
		"learn.menu": true, "learn.skills": true, "learn.profiles": true,
		"learn.concepts": true, "learn.detail": true,
	}
	allowedPrompts := map[string]bool{
		"learn.category": true, "learn.item": true, "learn.detail": true,
	}
	choicePattern := regexp.MustCompile(`^(?:[1-9][0-9]*|b|q|enter)$`)
	wantScreens := map[string]bool{"learn.menu": false, "learn.skills": false, "learn.profiles": false, "learn.concepts": false, "learn.detail": false}

	lines := strings.Split(strings.TrimSpace(trace.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("explorer emitted no trace")
	}
	var lastSeq float64
	for _, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("trace line is not JSON: %v\n%s", err, line)
		}
		if m["schema"] != float64(1) {
			t.Fatalf("trace schema = %v, want 1", m["schema"])
		}
		seq, ok := m["seq"].(float64)
		if !ok || seq <= lastSeq {
			t.Fatalf("trace seq not increasing: %v", m["seq"])
		}
		lastSeq = seq
		event, _ := m["event"].(string)
		switch event {
		case "screen.shown":
			screen, _ := m["screen"].(string)
			if !allowedScreens[screen] {
				t.Fatalf("unexpected explorer screen %q", screen)
			}
			wantScreens[screen] = true
			if screen == "learn.detail" && len(m) != 5 {
				t.Fatalf("detail screen event carries metadata: %v", m)
			}
		case "prompt.shown":
			prompt, _ := m["prompt"].(string)
			if !allowedPrompts[prompt] {
				t.Fatalf("unexpected explorer prompt %q", prompt)
			}
			if _, ok := m["default"]; !ok {
				t.Fatalf("prompt.shown without default: %v", m)
			}
		case "prompt.answered":
			prompt, _ := m["prompt"].(string)
			if !allowedPrompts[prompt] {
				t.Fatalf("unexpected explorer prompt %q", prompt)
			}
			if choice, ok := m["choice"]; ok {
				text, _ := choice.(string)
				if !choicePattern.MatchString(text) {
					t.Fatalf("unbounded choice %q in trace", text)
				}
			}
		default:
			t.Fatalf("unexpected explorer trace event %q", event)
		}
	}
	for screen, seen := range wantScreens {
		if !seen {
			t.Fatalf("explorer did not trace screen %q", screen)
		}
	}
	raw := trace.String()
	for _, forbidden := range []string{
		home, repo, "SECRET", "agentic-sdd-spec", "agentic-sdd-custom",
		"sdd-implement-1", "sdd-orchestrator", "matches source", "Purpose",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("trace leaked %q:\n%s", forbidden, raw)
		}
	}
}

func TestGuideExplorerDetailBackNavigationAndGroupedHeaders(t *testing.T) {
	repo, home := newExplorerFixture(t)
	writeExplorerProfileTarget(t, repo, home, "sdd-orchestrator", "", false)

	// Test 1: Typing 'b' on detail screen returns to the list view (AC-05).
	quit, code, out, stderr := runExplorer(t, repo, home, "2\n1\nb\nb\nq\n", nil)
	if !quit || code != 0 {
		t.Fatalf("b at detail: quit=%v code=%d stderr=%q", quit, code, stderr)
	}
	if !strings.Contains(out, "OpenCode profile templates") {
		t.Fatalf("b at detail did not return to profile list:\n%s", out)
	}

	// Test 2: Category list renders section group headers (AC-02).
	if !strings.Contains(out, "--- Main communicator ---") {
		t.Fatalf("profiles list missing group header:\n%s", out)
	}

	// Test 3: Detail card renders next steps and concise safety badge (AC-04, AC-09).
	if !strings.Contains(out, "Safety: ") {
		t.Fatalf("detail card missing Safety badge:\n%s", out)
	}
	if !strings.Contains(out, "Next steps:") {
		t.Fatalf("detail card missing Next steps:\n%s", out)
	}
}

