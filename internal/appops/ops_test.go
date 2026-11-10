package appops

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// newOpsFixture creates an isolated repository with one portable skill and a
// distinct home. No test reads the real user home or the network.
func newOpsFixture(t *testing.T) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	home = filepath.Join(root, "home")
	writeOpsSkill(t, repo, "agentic-sdd-plan", "new")
	return repo, home
}

func writeOpsSkill(t *testing.T, repo, skill, content string) {
	t.Helper()
	dir := filepath.Join(repo, "skills-files", skill)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeOpsPack(t *testing.T, repo string) {
	t.Helper()
	files := map[string]string{
		"opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md": "orchestrate",
		"opencode-pack/agents/sdd-base.md":                      "base profile",
		"opencode-pack/presets/free-first/sdd-free.md":          "free profile",
	}
	for rel, content := range files {
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func opsClientPath(home, client, skill string) string {
	sub := map[string]string{
		"agents": filepath.Join(".agents", "skills"),
		"codex":  filepath.Join(".codex", "skills"),
		"claude": filepath.Join(".claude", "skills"),
		"agy":    filepath.Join(".gemini", "antigravity-cli", "skills"),
	}[client]
	return filepath.Join(home, sub, skill)
}

func writeOpsDestination(t *testing.T, home, client, skill, content string) string {
	t.Helper()
	dir := opsClientPath(home, client, skill)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readOpsFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func opsBackupCount(t *testing.T, repo string) int {
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

// assertNoInstallWrites proves a refused apply wrote nothing: no client
// destination tree and no backup directory appeared.
func assertNoInstallWrites(t *testing.T, repo, home string) {
	t.Helper()
	for _, sub := range []string{
		filepath.Join(".agents", "skills"),
		filepath.Join(".codex", "skills"),
		filepath.Join(".claude", "skills"),
		filepath.Join(".gemini", "antigravity-cli", "skills"),
	} {
		if _, err := os.Stat(filepath.Join(home, sub)); !os.IsNotExist(err) {
			t.Fatalf("refused apply touched %s: %v", filepath.Join(home, sub), err)
		}
	}
	if got := opsBackupCount(t, repo); got != 0 {
		t.Fatalf("refused apply created %d backups", got)
	}
}

func TestNewInstallRequestValidatesAndCopies(t *testing.T) {
	repo, home := newOpsFixture(t)

	if _, err := NewInstallRequest(repo, "", false, skillsync.Options{}); err == nil || !strings.Contains(err.Error(), "home is required") {
		t.Fatalf("empty home error = %v", err)
	}
	if _, err := NewInstallRequest(repo, home, false, skillsync.Options{ClientIDs: []string{}}); err == nil || !strings.Contains(err.Error(), "client selection is empty") {
		t.Fatalf("explicit empty selection error = %v", err)
	}
	if _, err := NewInstallRequest(repo, home, false, skillsync.Options{ClientIDs: []string{"nope"}}); err == nil || !strings.Contains(err.Error(), `unknown client "nope"`) {
		t.Fatalf("unknown selection error = %v", err)
	}
	if _, err := NewInstallRequest(repo, home, false, skillsync.Options{OpenCodeFreeFirst: true}); err == nil || !strings.Contains(err.Error(), "--opencode-free-first requires --with-opencode") {
		t.Fatalf("free-first without pack error = %v", err)
	}

	relativeRepo, relativeHome := filepath.Join("relative", "repo"), filepath.Join("relative", "home")
	req, err := NewInstallRequest(relativeRepo, relativeHome, true, skillsync.Options{WithOpenCode: true})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(req.Repo()) || !filepath.IsAbs(req.Home()) {
		t.Fatalf("request paths are not absolute: repo=%q home=%q", req.Repo(), req.Home())
	}
	if !req.Apply() {
		t.Fatal("request lost the apply flag")
	}
	if req.Options().ClientIDs != nil {
		t.Fatalf("nil selection did not stay nil: %v", req.Options().ClientIDs)
	}

	input := []string{"agy", "claude", "agy"}
	req, err = NewInstallRequest(repo, home, false, skillsync.Options{ClientIDs: input})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := req.Options().ClientIDs, []string{"claude", "agy"}; !slices.Equal(got, want) {
		t.Fatalf("normalized selection = %v, want %v", got, want)
	}
	input[0] = "agents"
	if got, want := req.Options().ClientIDs, []string{"claude", "agy"}; !slices.Equal(got, want) {
		t.Fatalf("request aliases the caller's selection: %v, want %v", got, want)
	}
	accessorCopy := req.Options()
	accessorCopy.ClientIDs[0] = "agents"
	if got, want := req.Options().ClientIDs, []string{"claude", "agy"}; !slices.Equal(got, want) {
		t.Fatalf("Options accessor exposed the request's slice: %v, want %v", got, want)
	}
}

func TestNewRestoreRequestValidates(t *testing.T) {
	if _, err := NewRestoreRequest("", "", "20060102T150405.000000000Z", false); err == nil || !strings.Contains(err.Error(), "home is required") {
		t.Fatalf("empty home error = %v", err)
	}
	if _, err := NewRestoreRequest("", t.TempDir(), "not-an-id", false); err == nil || !strings.Contains(err.Error(), "invalid backup id") {
		t.Fatalf("invalid backup id error = %v", err)
	}
	req, err := NewRestoreRequest(filepath.Join("relative", "repo"), filepath.Join("relative", "home"), "20060102T150405.000000000Z", true)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(req.Repo()) || !filepath.IsAbs(req.Home()) || req.BackupID() != "20060102T150405.000000000Z" || !req.Apply() {
		t.Fatalf("restore request = %+v", req)
	}
}

func TestOperationIDStaleness(t *testing.T) {
	current := OperationID{Request: 4, Generation: 7}
	for _, tc := range []struct {
		name  string
		id    OperationID
		stale bool
	}{
		{"same identity", current, false},
		{"newer generation", OperationID{Request: 4, Generation: 8}, false},
		{"older generation", OperationID{Request: 4, Generation: 6}, true},
		{"different request", OperationID{Request: 5, Generation: 7}, true},
		{"other request and older", OperationID{Request: 5, Generation: 1}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.id.StaleFor(current); got != tc.stale {
				t.Fatalf("StaleFor = %v, want %v", got, tc.stale)
			}
		})
	}
}

func TestPreviewTokenStableAndApplyParity(t *testing.T) {
	repo, home := newOpsFixture(t)
	req, err := NewInstallRequest(repo, home, true, skillsync.Options{})
	if err != nil {
		t.Fatal(err)
	}

	first, err := PreviewInstall(req, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome != skillsync.OutcomePreview || len(first.PreviewToken) != 64 || len(first.SourceDigest) != 64 {
		t.Fatalf("preview result = %+v", first)
	}
	if len(first.Applied) != 0 {
		t.Fatalf("preview claims applied changes: %+v", first.Applied)
	}
	second, err := PreviewInstall(req, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if second.PreviewToken != first.PreviewToken || second.SourceDigest != first.SourceDigest {
		t.Fatalf("identical previews disagree: %+v vs %+v", first, second)
	}

	applied, err := ApplyInstall(req, first.PreviewToken, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != skillsync.OutcomeApplied || applied.PreviewToken != first.PreviewToken {
		t.Fatalf("apply result = %+v", applied)
	}
	if applied.SourceDigest != first.SourceDigest {
		t.Fatalf("apply digest %q != preview digest %q", applied.SourceDigest, first.SourceDigest)
	}
	if !reflect.DeepEqual(applied.Planned, first.Planned) || len(applied.Applied) != 4 {
		t.Fatalf("apply plan/applied = %+v / %+v", applied.Planned, applied.Applied)
	}
	for _, client := range []string{"agents", "codex", "claude", "agy"} {
		if got := readOpsFile(t, filepath.Join(opsClientPath(home, client, "agentic-sdd-plan"), "SKILL.md")); got != "new" {
			t.Fatalf("%s skill = %q, want new", client, got)
		}
	}
}

func TestApplyInstallRequiresMatchingPreview(t *testing.T) {
	t.Run("changed home", func(t *testing.T) {
		repo, homeA := newOpsFixture(t)
		homeB := filepath.Join(filepath.Dir(homeA), "home-b")
		reqA, err := NewInstallRequest(repo, homeA, true, skillsync.Options{})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewInstall(reqA, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		reqB, err := NewInstallRequest(repo, homeB, true, skillsync.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyInstall(reqB, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		assertNoInstallWrites(t, repo, homeB)
	})

	t.Run("changed repo", func(t *testing.T) {
		repoA, home := newOpsFixture(t)
		repoB := filepath.Join(filepath.Dir(repoA), "repo-b")
		writeOpsSkill(t, repoB, "agentic-sdd-plan", "new")
		reqA, err := NewInstallRequest(repoA, home, true, skillsync.Options{})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewInstall(reqA, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		reqB, err := NewInstallRequest(repoB, home, true, skillsync.Options{})
		if err != nil {
			t.Fatal(err)
		}
		// Identical source payload in a different checkout keeps the same
		// source identity; only the pinned repo path differs.
		previewB, err := PreviewInstall(reqB, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		if previewB.SourceDigest != preview.SourceDigest {
			t.Fatalf("identical source trees produced different digests: %q != %q", previewB.SourceDigest, preview.SourceDigest)
		}
		if previewB.PreviewToken == preview.PreviewToken {
			t.Fatal("different repo paths produced the same preview token")
		}
		if _, err := ApplyInstall(reqB, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		assertNoInstallWrites(t, repoB, home)
	})

	t.Run("changed selection", func(t *testing.T) {
		repo, home := newOpsFixture(t)
		reqAll, err := NewInstallRequest(repo, home, true, skillsync.Options{})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewInstall(reqAll, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		reqSubset, err := NewInstallRequest(repo, home, true, skillsync.Options{ClientIDs: []string{"claude"}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyInstall(reqSubset, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		assertNoInstallWrites(t, repo, home)
	})

	t.Run("changed options", func(t *testing.T) {
		root := t.TempDir()
		repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
		writeOpsSkill(t, repo, "agentic-sdd-plan", "new")
		writeOpsPack(t, repo)
		reqPack, err := NewInstallRequest(repo, home, true, skillsync.Options{WithOpenCode: true})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewInstall(reqPack, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		reqPreset, err := NewInstallRequest(repo, home, true, skillsync.Options{WithOpenCode: true, OpenCodeFreeFirst: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyInstall(reqPreset, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		assertNoInstallWrites(t, repo, home)
	})

	t.Run("changed source content", func(t *testing.T) {
		repo, home := newOpsFixture(t)
		req, err := NewInstallRequest(repo, home, true, skillsync.Options{})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewInstall(req, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		writeOpsSkill(t, repo, "agentic-sdd-plan", "changed after preview")
		if _, err := ApplyInstall(req, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		assertNoInstallWrites(t, repo, home)
	})

	t.Run("changed destination state", func(t *testing.T) {
		repo, home := newOpsFixture(t)
		req, err := NewInstallRequest(repo, home, true, skillsync.Options{ClientIDs: []string{"claude"}})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewInstall(req, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		conflict := writeOpsDestination(t, home, "claude", "agentic-sdd-plan", "appeared after preview")
		if _, err := ApplyInstall(req, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		if got := readOpsFile(t, conflict); got != "appeared after preview" {
			t.Fatalf("refused apply changed the destination: %q", got)
		}
		if got := opsBackupCount(t, repo); got != 0 {
			t.Fatalf("refused apply created %d backups", got)
		}
	})

	t.Run("destination became up to date", func(t *testing.T) {
		repo, home := newOpsFixture(t)
		req, err := NewInstallRequest(repo, home, true, skillsync.Options{ClientIDs: []string{"claude"}})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewInstall(req, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		// Creating the exact source content turns the previewed install into
		// an up-to-date entry: a new plan must be reviewed.
		seedDestinationFromSource(t, repo, home)
		if _, err := ApplyInstall(req, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		if got := opsBackupCount(t, repo); got != 0 {
			t.Fatalf("refused apply created %d backups", got)
		}
	})
}

// seedDestinationFromSource installs the fixture source with plain
// filesystem writes, avoiding the seam under test for the "destination became
// up to date" setup.
func seedDestinationFromSource(t *testing.T, repo, home string) {
	t.Helper()
	src := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	data, err := os.ReadFile(filepath.Join(src, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(opsClientPath(home, "claude", "agentic-sdd-plan"), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestUnselectedDestinationEditDoesNotInvalidatePreview(t *testing.T) {
	repo, home := newOpsFixture(t)
	agentsPlan := writeOpsDestination(t, home, "agents", "agentic-sdd-plan", "unselected")
	req, err := NewInstallRequest(repo, home, true, skillsync.Options{ClientIDs: []string{"claude"}})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewInstall(req, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Planned) != 1 || preview.Planned[0].Client != "claude" {
		t.Fatalf("subset preview = %+v", preview.Planned)
	}

	// An unrelated unselected tree changes after the preview. It is not part
	// of the plan, so the reviewed change set is still exact.
	if err := os.WriteFile(agentsPlan, []byte("edited while reviewing"), 0644); err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyInstall(req, preview.PreviewToken, io.Discard)
	if err != nil {
		t.Fatalf("selected preview was invalidated by an unselected edit: %v", err)
	}
	if applied.Outcome != skillsync.OutcomeApplied || len(applied.Applied) != 1 {
		t.Fatalf("subset apply = %+v", applied)
	}
	if got := readOpsFile(t, agentsPlan); got != "edited while reviewing" {
		t.Fatalf("unselected destination was changed: %q", got)
	}
	if got := opsBackupCount(t, repo); got != 1 {
		t.Fatalf("subset apply backups = %d, want 1", got)
	}
}

func TestApplyRefusesEmptyStaleAndDoubleTokens(t *testing.T) {
	repo, home := newOpsFixture(t)
	req, err := NewInstallRequest(repo, home, true, skillsync.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyInstall(req, "", io.Discard); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("empty-token error = %v, want ErrNeedsPreview", err)
	}
	assertNoInstallWrites(t, repo, home)

	preview, err := PreviewInstall(req, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyInstall(req, strings.Repeat("0", 64), io.Discard); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("stale-token error = %v, want ErrNeedsPreview", err)
	}
	assertNoInstallWrites(t, repo, home)

	applied, err := ApplyInstall(req, preview.PreviewToken, io.Discard)
	if err != nil || applied.Outcome != skillsync.OutcomeApplied {
		t.Fatalf("first apply = %+v, %v", applied, err)
	}
	if got := opsBackupCount(t, repo); got != 1 {
		t.Fatalf("first apply backups = %d, want 1", got)
	}
	if _, err := ApplyInstall(req, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("double-apply error = %v, want ErrNeedsPreview", err)
	}
	if got := opsBackupCount(t, repo); got != 1 {
		t.Fatalf("double apply created another backup: %d", got)
	}
	if got := readOpsFile(t, filepath.Join(opsClientPath(home, "claude", "agentic-sdd-plan"), "SKILL.md")); got != "new" {
		t.Fatalf("double apply changed the destination: %q", got)
	}
}

func TestSubsetRoundTripThroughSeam(t *testing.T) {
	repo, home := newOpsFixture(t)
	claudePlan := writeOpsDestination(t, home, "claude", "agentic-sdd-plan", "claude old")
	agentsPlan := writeOpsDestination(t, home, "agents", "agentic-sdd-plan", "agents old")

	install, err := NewInstallRequest(repo, home, true, skillsync.Options{ClientIDs: []string{"claude"}})
	if err != nil {
		t.Fatal(err)
	}
	installPreview, err := PreviewInstall(install, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyInstall(install, installPreview.PreviewToken, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != skillsync.OutcomeApplied || len(applied.Applied) != 1 || applied.BackupID == "" {
		t.Fatalf("subset install = %+v", applied)
	}
	if got := readOpsFile(t, claudePlan); got != "new" {
		t.Fatalf("selected client = %q, want new", got)
	}
	if got := readOpsFile(t, agentsPlan); got != "agents old" {
		t.Fatalf("unselected client changed to %q", got)
	}
	// The manifest keeps the full canonical home identity even for a subset.
	manifest := readOpsManifest(t, applied.BackupPath)
	var targets []string
	for _, client := range []string{"agents", "codex", "claude", "agy"} {
		targets = append(targets, opsClientPath(home, client, ""))
	}
	gotTargets, _ := manifest["targets"].([]any)
	if len(gotTargets) != len(targets) {
		t.Fatalf("manifest targets = %v, want %v", gotTargets, targets)
	}
	for i, target := range targets {
		if gotTargets[i] != target {
			t.Fatalf("manifest target %d = %v, want %q", i, gotTargets[i], target)
		}
	}

	restore, err := NewRestoreRequest(repo, home, applied.BackupID, true)
	if err != nil {
		t.Fatal(err)
	}
	restorePreview, err := PreviewRestore(restore, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(restorePreview.Planned) != 1 || restorePreview.Planned[0].Client != "claude" {
		t.Fatalf("subset restore preview = %+v", restorePreview.Planned)
	}
	restored, err := ApplyRestore(restore, restorePreview.PreviewToken, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Outcome != skillsync.OutcomeApplied || len(restored.Applied) != 1 {
		t.Fatalf("subset restore = %+v", restored)
	}
	if got := readOpsFile(t, claudePlan); got != "claude old" {
		t.Fatalf("restore did not recover claude: %q", got)
	}
	if got := readOpsFile(t, agentsPlan); got != "agents old" {
		t.Fatalf("restore changed the unselected client: %q", got)
	}
}

func TestApplyRestoreRequiresMatchingPreview(t *testing.T) {
	t.Run("changed destination", func(t *testing.T) {
		repo, home, id := opsBackupFixture(t)
		req, err := NewRestoreRequest(repo, home, id, true)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewRestore(req, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(opsClientPath(home, "claude", "agentic-sdd-plan"), "SKILL.md")
		wantReplace := false
		for _, change := range preview.Planned {
			if change.Client == "claude" && change.Action == skillsync.ChangeReplace {
				wantReplace = true
			}
		}
		if !wantReplace {
			t.Fatalf("restore preview has no claude replace: %+v", preview.Planned)
		}
		// Removing the reviewed destination turns the same entry from a
		// replace into a restore: the proposed write changed.
		if err := os.RemoveAll(filepath.Dir(destination)); err != nil {
			t.Fatal(err)
		}
		before := opsBackupCount(t, repo)
		if _, err := ApplyRestore(req, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("restore error = %v, want ErrNeedsPreview", err)
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatalf("refused restore wrote the removed destination: %v", err)
		}
		if after := opsBackupCount(t, repo); after != before {
			t.Fatalf("refused restore created %d backups", after-before)
		}
	})

	t.Run("changed backup payload", func(t *testing.T) {
		repo, home, id := opsBackupFixture(t)
		req, err := NewRestoreRequest(repo, home, id, true)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewRestore(req, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		backupDir := filepath.Join(repo, "backups", id)
		rel := filepath.Join("claude", "agentic-sdd-plan")
		if err := os.WriteFile(filepath.Join(backupDir, rel, "SKILL.md"), []byte("consistent tamper"), 0644); err != nil {
			t.Fatal(err)
		}
		// Keep the recorded digest consistent so the backup still validates;
		// only its identity changed.
		manifest := readOpsManifest(t, backupDir)
		entries, _ := manifest["entries"].([]any)
		for _, raw := range entries {
			entry, _ := raw.(map[string]any)
			if entry["client"] == "claude" && entry["skill"] == "agentic-sdd-plan" {
				entry["sha256"] = opsTreeDigest(t, backupDir, rel)
			}
		}
		writeOpsManifest(t, backupDir, manifest)
		destination := filepath.Join(opsClientPath(home, "claude", "agentic-sdd-plan"), "SKILL.md")
		before := opsBackupCount(t, repo)
		if _, err := ApplyRestore(req, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("restore error = %v, want ErrNeedsPreview", err)
		}
		if got := readOpsFile(t, destination); got != "new" {
			t.Fatalf("refused restore changed the destination: %q", got)
		}
		if after := opsBackupCount(t, repo); after != before {
			t.Fatalf("refused restore created %d backups", after-before)
		}
	})

	t.Run("invalid backup payload", func(t *testing.T) {
		repo, home, id := opsBackupFixture(t)
		req, err := NewRestoreRequest(repo, home, id, true)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := PreviewRestore(req, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		payload := filepath.Join(repo, "backups", id, "claude", "agentic-sdd-plan", "SKILL.md")
		if err := os.WriteFile(payload, []byte("digest-breaking tamper"), 0644); err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(opsClientPath(home, "claude", "agentic-sdd-plan"), "SKILL.md")
		if _, err := ApplyRestore(req, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("restore error = %v, want ErrNeedsPreview", err)
		}
		if got := readOpsFile(t, destination); got != "new" {
			t.Fatalf("refused restore changed the destination: %q", got)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		repo, home, id := opsBackupFixture(t)
		req, err := NewRestoreRequest(repo, home, id, true)
		if err != nil {
			t.Fatal(err)
		}
		before := opsBackupCount(t, repo)
		if _, err := ApplyRestore(req, "", io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("empty-token error = %v, want ErrNeedsPreview", err)
		}
		if after := opsBackupCount(t, repo); after != before {
			t.Fatalf("empty-token restore created %d backups", after-before)
		}
	})
}

// opsBackupFixture installs the fixture with the seam, then re-seeds the
// replaced claude destination with "new" so a restore has a visible change.
func opsBackupFixture(t *testing.T) (repo, home, id string) {
	t.Helper()
	repo, home = newOpsFixture(t)
	writeOpsDestination(t, home, "claude", "agentic-sdd-plan", "old")
	req, err := NewInstallRequest(repo, home, true, skillsync.Options{})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewInstall(req, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyInstall(req, preview.PreviewToken, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if applied.BackupID == "" {
		t.Fatal("apply created no backup")
	}
	return repo, home, applied.BackupID
}

func readOpsManifest(t *testing.T, backupDir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(backupDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeOpsManifest(t *testing.T, backupDir string, manifest map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "manifest.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

// opsTreeDigest mirrors the engine's tree digest so a test can update a
// manifest entry consistently after changing its saved payload.
func opsTreeDigest(t *testing.T, root, rel string) string {
	t.Helper()
	h := sha256.New()
	err := fs.WalkDir(os.DirFS(root), rel, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		trimmed := strings.TrimPrefix(p, rel+"/")
		if trimmed == rel {
			return nil
		}
		if entry.IsDir() {
			h.Write([]byte("d " + trimmed + "\n"))
			return nil
		}
		h.Write([]byte("f " + trimmed + "\n"))
		data, err := fs.ReadFile(os.DirFS(root), p)
		if err != nil {
			return err
		}
		h.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
