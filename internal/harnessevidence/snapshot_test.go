package harnessevidence

// Snapshot tests. The golden digests below were derived independently by the
// Python reference consumer (tests/harness/go_snapshot_fixture.py, run on
// 2026-11-02), never from this package:
//
//	full    (scope ".")        sha256-v1:8af96defe04ea392a3adf2bcc453b51c842b85f1f4a65e277332bfb07a67f5a7
//	focused (scope "code.txt") sha256-v1:884d3bbea15ec8d24993247de25445821627a19782384e814469784967a148cc
//
// The fixture intentionally covers committed, staged, unstaged, untracked,
// ignored, deleted and mode-changed files. All development fixtures are
// synthetic; no real workspace, credential store or provider is touched.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	goldenFullDiff    = "sha256-v1:8af96defe04ea392a3adf2bcc453b51c842b85f1f4a65e277332bfb07a67f5a7"
	goldenFocusedDiff = "sha256-v1:884d3bbea15ec8d24993247de25445821627a19782384e814469784967a148cc"
)

var fixtureContents = map[string]string{
	".gitignore":  "ignored.txt\n",
	"code.txt":    "original\n",
	"tasks.md":    "B1 awaiting human review\n",
	"sub/dep.txt": "dependency\n",
	"gone.txt":    "gone\n",
}

// buildFixtureWorkspace reproduces the reference fixture exactly. File modes
// are set explicitly so the fixture is umask independent.
func buildFixtureWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"--no-optional-locks", "-C", repo}, args...)...)
		cmd.Env = fixtureGitEnv(root)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("fixture git %v: %v: %s", args, err, stderr.String())
		}
	}
	git("init", "-q")
	git("config", "user.name", "Synthetic Fixture")
	git("config", "user.email", "fixture@example.invalid")
	for name, text := range fixtureContents {
		target := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(target, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("commit", "-qm", "synthetic fixture")
	write := func(name, text string, mode os.FileMode) {
		target := filepath.Join(repo, name)
		if err := os.WriteFile(target, []byte(text), mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(target, mode); err != nil {
			t.Fatal(err)
		}
	}
	write("staged.txt", "staged-new\n", 0o644)
	git("add", "staged.txt")
	write("code.txt", "unstaged\n", 0o644)
	write("new.txt", "untracked\n", 0o644)
	write("ignored.txt", "ignored-content\n", 0o644)
	if err := os.Remove(filepath.Join(repo, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(repo, "tasks.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

func fixtureGitEnv(root string) []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + root,
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_COUNT=3",
		"GIT_CONFIG_KEY_0=init.templateDir", "GIT_CONFIG_VALUE_0=",
		"GIT_CONFIG_KEY_1=core.hooksPath", "GIT_CONFIG_VALUE_1=" + os.DevNull,
		"GIT_CONFIG_KEY_2=commit.gpgsign", "GIT_CONFIG_VALUE_2=false",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1",
	}
}

func TestSnapshotMatchesReferenceGolden(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := buildFixtureWorkspace(t)
	full, err := Snapshot(repo, []string{"."})
	if err != nil {
		t.Fatalf("full snapshot: %v", err)
	}
	if full.Diff != goldenFullDiff {
		t.Fatalf("full diff %q does not match the reference golden %q", full.Diff, goldenFullDiff)
	}
	if len(full.BaseRevision) != 40 {
		t.Fatalf("base revision is not a commit identity: %q", full.BaseRevision)
	}
	focused, err := Snapshot(repo, []string{"code.txt"})
	if err != nil {
		t.Fatalf("focused snapshot: %v", err)
	}
	if focused.Diff != goldenFocusedDiff {
		t.Fatalf("focused diff %q does not match the reference golden %q", focused.Diff, goldenFocusedDiff)
	}
	if focused == full {
		t.Fatalf("focused and full identities must differ")
	}
}

func TestSnapshotDetectsWorkingTreeChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := buildFixtureWorkspace(t)
	base, err := Snapshot(repo, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(filepath.Join(repo, "sub"), []string{"."}); err == nil {
		t.Fatalf("workspace must be the repository root")
	}
	steps := []struct {
		name string
		do   func()
		undo func()
	}{
		{"staged", func() {
			writeFixtureFile(t, repo, "later.txt", "later\n", 0o644)
			fixtureGit(t, repo, "add", "later.txt")
		}, func() {
			fixtureGit(t, repo, "reset", "-q", "--", "later.txt")
			_ = os.Remove(filepath.Join(repo, "later.txt"))
		}},
		{"unstaged", func() {
			writeFixtureFile(t, repo, "code.txt", "changed\n", 0o644)
		}, func() {
			writeFixtureFile(t, repo, "code.txt", "unstaged\n", 0o644)
		}},
		{"untracked", func() {
			writeFixtureFile(t, repo, "extra.txt", "extra\n", 0o644)
		}, func() {
			_ = os.Remove(filepath.Join(repo, "extra.txt"))
		}},
		{"deleted", func() {
			_ = os.Remove(filepath.Join(repo, "tasks.md"))
		}, func() {
			writeFixtureFile(t, repo, "tasks.md", "B1 awaiting human review\n", 0o755)
		}},
		{"mode", func() {
			_ = os.Chmod(filepath.Join(repo, "code.txt"), 0o755)
		}, func() {
			_ = os.Chmod(filepath.Join(repo, "code.txt"), 0o644)
		}},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			step.do()
			defer step.undo()
			after, err := Snapshot(repo, []string{"."})
			if err != nil {
				t.Fatalf("snapshot after %s: %v", step.name, err)
			}
			if after.Diff == base.Diff {
				t.Fatalf("%s change did not change the scoped identity", step.name)
			}
		})
	}
	final, err := Snapshot(repo, []string{"."})
	if err != nil || final.Diff != base.Diff {
		t.Fatalf("fixture not restored: %v %+v", err, final)
	}
}

func TestSnapshotRefusesUnsafeScopesAndFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := buildFixtureWorkspace(t)
	for _, paths := range [][]string{
		{},
		{"code.txt", "code.txt"},
		{"../outside"},
		{"/absolute"},
		{"*.txt"},
		{"a//b"},
		{"./code.txt"},
		{"sub/../code.txt"},
		{".git"},
		{"a\nb"},
	} {
		if _, err := Snapshot(repo, paths); err == nil {
			t.Fatalf("unsafe scope %q accepted", paths)
		}
	}
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "subdir", "not-authorized.txt"), []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(repo, []string{"."}); err == nil {
		t.Fatalf("symlink in scope was not refused")
	}
	if _, err := Snapshot(repo, []string{"alias/subdir"}); err == nil {
		t.Fatalf("symlink traversal was not refused")
	}
	if err := os.Remove(filepath.Join(repo, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := mkfifo(filepath.Join(repo, "fifo")); err != nil {
		t.Skipf("cannot create fixture fifo: %v", err)
	}
	defer os.Remove(filepath.Join(repo, "fifo"))
	if _, err := Snapshot(repo, []string{"."}); err == nil {
		t.Fatalf("special file was not refused")
	}
	if _, err := Snapshot(repo, []string{"fifo"}); err == nil {
		t.Fatalf("explicit special file was not refused")
	}
}

func TestSnapshotExplicitScopeIgnoresUnrelatedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := buildFixtureWorkspace(t)
	focused, err := Snapshot(repo, []string{"tasks.md"})
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, repo, "unrelated.txt", "not a dependency\n", 0o644)
	defer os.Remove(filepath.Join(repo, "unrelated.txt"))
	again, err := Snapshot(repo, []string{"tasks.md"})
	if err != nil {
		t.Fatal(err)
	}
	if focused.Diff != again.Diff {
		t.Fatalf("unrelated file changed a focused identity")
	}
	whole, err := Snapshot(repo, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if whole.Diff == again.Diff {
		t.Fatalf("focused scope must differ from the whole repository")
	}
}

func TestSnapshotExplicitNamedIgnoredFileIsIncluded(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := buildFixtureWorkspace(t)
	without, err := Snapshot(repo, []string{"new.txt"})
	if err != nil {
		t.Fatal(err)
	}
	with, err := Snapshot(repo, []string{"new.txt", "ignored.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if with.Diff == without.Diff {
		t.Fatalf("explicitly named ignored file must be part of the identity")
	}
}

func writeFixtureFile(t *testing.T, repo, name, text string, mode os.FileMode) {
	t.Helper()
	target := filepath.Join(repo, name)
	if err := os.WriteFile(target, []byte(text), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, mode); err != nil {
		t.Fatal(err)
	}
}

func fixtureGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"--no-optional-locks", "-C", repo}, args...)...)
	cmd.Env = fixtureGitEnv(filepath.Dir(repo))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("fixture git %v: %v: %s", args, err, stderr.String())
	}
}

func mkfifo(path string) error {
	cmd := exec.Command("mkfifo", path)
	return cmd.Run()
}
