package execution

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// repoTestEnv returns a deterministic, isolated environment for disposable Git
// repositories. No global or system Git configuration, home directory or
// network is consulted.
func repoTestEnv(t *testing.T) []string {
	t.Helper()
	overrides := map[string]string{
		"HOME":                   t.TempDir(),
		"GIT_CONFIG_GLOBAL":      "/dev/null",
		"GIT_CONFIG_SYSTEM":      "/dev/null",
		"GIT_CONFIG_NOSYSTEM":    "1",
		"GIT_AUTHOR_NAME":        "Execution Test",
		"GIT_AUTHOR_EMAIL":       "execution@example.invalid",
		"GIT_COMMITTER_NAME":     "Execution Test",
		"GIT_COMMITTER_EMAIL":    "execution@example.invalid",
		"GIT_OPTIONAL_LOCKS":     "0",
		"GIT_TERMINAL_PROMPT":    "0",
		"GIT_NO_REPLACE_OBJECTS": "1",
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		name := entry
		if index := strings.IndexByte(entry, '='); index >= 0 {
			name = entry[:index]
		}
		if _, replaced := overrides[name]; replaced {
			continue
		}
		env = append(env, entry)
	}
	for _, key := range keys {
		env = append(env, key+"="+overrides[key])
	}
	return env
}

func repoTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, code := repoTestGitCode(t, dir, args...)
	if code != 0 {
		t.Fatalf("git %s exited %d\n%s", strings.Join(args, " "), code, out)
	}
	return strings.TrimSpace(out)
}

func repoTestGitCode(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = repoTestEnv(t)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	err := cmd.Run()
	if err == nil {
		return combined.String(), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return combined.String(), exitErr.ExitCode()
	}
	t.Fatalf("git %s could not run: %v", strings.Join(args, " "), err)
	return "", -1
}

func repoTestInitEmpty(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve temp dir: %v", err)
	}
	repoTestGit(t, root, "init", "-q")
	return root
}

func repoTestInit(t *testing.T) string {
	t.Helper()
	root := repoTestInitEmpty(t)
	repoTestWrite(t, filepath.Join(root, ".gitignore"), "ignored.txt\n")
	repoTestWrite(t, filepath.Join(root, "README.md"), "baseline\n")
	repoTestCommit(t, root, "baseline")
	return root
}

func repoTestWrite(t *testing.T, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		t.Fatalf("create parent for %s: %v", name, err)
	}
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
}

func repoTestRead(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func repoTestCommit(t *testing.T, root, message string) {
	t.Helper()
	repoTestGit(t, root, "add", "-A")
	repoTestGit(t, root, "commit", "-q", "-m", message)
}

const repoTestSpec = `# Test Feature

## 7. Requirements

| **FR-01** | Do the thing. |

## 8. Acceptance

| **AC-01** | The thing works. |

## 14. Approval

Approved for the fixture only.
`

const repoTestTasks = `# Test Feature — Tasks

## Batch ledger

| Batch | Scope |
|---|---|
| **B1** | Test batch |

## B1 — Test batch

**Batch ID:** B1
**State:** ` + "`in progress`" + `

### T01 — First task

- [ ] **T01** Do a thing.
  - **Objective:** Exercise the fixture.
  - **Verification:** ` + "`go test ./...`" + `

### T02 — Second task

- [ ] **T02** Do another thing.

## B2 — Other batch

**Batch ID:** B2

### T01 — Look-alike task

- [ ] **T01** Appears only in another batch.
`

// repoTestFeature writes and commits a small feature fixture under
// specs/001-test and returns its directory.
func repoTestFeature(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "specs", "001-test")
	repoTestWrite(t, filepath.Join(dir, "spec.md"), repoTestSpec)
	repoTestWrite(t, filepath.Join(dir, "tasks.md"), repoTestTasks)
	repoTestCommit(t, root, "feature fixture")
	return dir
}

func TestRepositoryOpenAndObserve(t *testing.T) {
	root := repoTestInit(t)
	repoTestWrite(t, filepath.Join(root, "sub", "file.txt"), "nested\n")
	repoTestCommit(t, root, "nested")

	repo, err := OpenRepository(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	if repo.Root != root {
		t.Fatalf("repository root = %q, want %q", repo.Root, root)
	}
	if len(repo.Head) != 40 {
		t.Fatalf("HEAD = %q, want a full commit ID", repo.Head)
	}
	if repo.Branch == "" {
		t.Fatal("branch is empty on an attached HEAD")
	}
	if !strings.HasPrefix(repo.GitVersion, "git version") {
		t.Fatalf("Git version = %q", repo.GitVersion)
	}
	observation, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if observation.Head != repo.Head {
		t.Fatalf("observation HEAD = %q, want %q", observation.Head, repo.Head)
	}
	if !strings.HasPrefix(observation.WorktreeDigest, "sha256:") {
		t.Fatalf("worktree digest = %q", observation.WorktreeDigest)
	}
	if observation.ObservedAt.IsZero() {
		t.Fatal("observation timestamp is zero")
	}
}

func TestRepositoryRejectsNonRepository(t *testing.T) {
	dir := t.TempDir()
	if _, err := OpenRepository(dir); !errors.Is(err, ErrUsage) {
		t.Fatalf("OpenRepository(non-repo) error = %v, want ErrUsage", err)
	}
}

func TestRepositoryRejectsUnbornHead(t *testing.T) {
	root := repoTestInitEmpty(t)
	if _, err := OpenRepository(root); !errors.Is(err, ErrUsage) {
		t.Fatalf("OpenRepository(unborn HEAD) error = %v, want ErrUsage", err)
	}
}

func TestRepositoryObservationTracksWorktree(t *testing.T) {
	root := repoTestInit(t)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	first, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	second, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe again: %v", err)
	}
	if first.WorktreeDigest != second.WorktreeDigest {
		t.Fatal("worktree digest is not stable for an unchanged worktree")
	}

	repoTestWrite(t, filepath.Join(root, "README.md"), "baseline\nchanged\n")
	modified, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe modified: %v", err)
	}
	if modified.WorktreeDigest == first.WorktreeDigest {
		t.Fatal("worktree digest did not change after a tracked edit")
	}
	if modified.Head != first.Head {
		t.Fatal("HEAD changed without a commit")
	}

	repoTestWrite(t, filepath.Join(root, "README.md"), "baseline\n")
	reverted, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe reverted: %v", err)
	}
	if reverted.WorktreeDigest != first.WorktreeDigest {
		t.Fatal("worktree digest did not return after reverting a tracked edit")
	}

	repoTestWrite(t, filepath.Join(root, "new.txt"), "new\n")
	untracked, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe untracked: %v", err)
	}
	if untracked.WorktreeDigest == first.WorktreeDigest {
		t.Fatal("worktree digest did not change after adding an untracked file")
	}

	repoTestWrite(t, filepath.Join(root, "ignored.txt"), "secret\n")
	ignored, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe ignored: %v", err)
	}
	if ignored.WorktreeDigest != untracked.WorktreeDigest {
		t.Fatal("ignored files must not affect the worktree digest")
	}

	repoTestCommit(t, root, "second")
	committed, err := repo.Observe()
	if err != nil {
		t.Fatalf("observe committed: %v", err)
	}
	if committed.Head == first.Head {
		t.Fatal("HEAD did not change after a commit")
	}
}

func TestRepositoryObserveExcludingSources(t *testing.T) {
	root := repoTestInit(t)
	featureDir := repoTestFeature(t, root)
	relative := filepath.ToSlash(filepath.Join("specs", "001-test", "tasks.md"))
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	before, err := repo.ObserveExcluding([]string{relative})
	if err != nil {
		t.Fatalf("observe excluding: %v", err)
	}
	repoTestWrite(t, filepath.Join(featureDir, "tasks.md"), repoTestTasks+"\nAdditional note.\n")
	afterSourceEdit, err := repo.ObserveExcluding([]string{relative})
	if err != nil {
		t.Fatalf("observe excluding after edit: %v", err)
	}
	if afterSourceEdit.WorktreeDigest != before.WorktreeDigest {
		t.Fatal("excluding the source file must hide a source-only edit")
	}
	repoTestWrite(t, filepath.Join(root, "README.md"), "baseline\nother\n")
	afterOtherEdit, err := repo.ObserveExcluding([]string{relative})
	if err != nil {
		t.Fatalf("observe excluding after other edit: %v", err)
	}
	if afterOtherEdit.WorktreeDigest == before.WorktreeDigest {
		t.Fatal("an unrelated edit must still change the worktree digest")
	}
	if _, err := repo.ObserveExcluding([]string{"../escape"}); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("ObserveExcluding traversal error = %v, want ErrUnsafePath", err)
	}
}

func TestFeatureScope(t *testing.T) {
	root := repoTestInit(t)
	featureDir := repoTestFeature(t, root)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	scope, err := FeatureScope(repo, featureDir, "B1")
	if err != nil {
		t.Fatalf("feature scope: %v", err)
	}
	if scope.Kind != ScopeFeature {
		t.Fatalf("scope kind = %q, want feature", scope.Kind)
	}
	if scope.RepoRoot != root {
		t.Fatalf("scope repository root = %q, want %q", scope.RepoRoot, root)
	}
	if scope.FeatureDir != "specs/001-test" {
		t.Fatalf("feature dir = %q", scope.FeatureDir)
	}
	if scope.SpecPath != "specs/001-test/spec.md" || scope.TasksPath != "specs/001-test/tasks.md" {
		t.Fatalf("source paths = %q, %q", scope.SpecPath, scope.TasksPath)
	}
	specData := []byte(repoTestRead(t, filepath.Join(featureDir, "spec.md")))
	tasksData := []byte(repoTestRead(t, filepath.Join(featureDir, "tasks.md")))
	if scope.SpecDigest != DigestBytes(specData) {
		t.Fatalf("spec digest = %q", scope.SpecDigest)
	}
	if scope.TasksDigest != DigestBytes(tasksData) {
		t.Fatalf("tasks digest = %q", scope.TasksDigest)
	}
	wantBatch, err := NormalizedBatchScopeDigest(tasksData, "B1")
	if err != nil {
		t.Fatalf("batch scope digest: %v", err)
	}
	if scope.BatchID != "B1" || scope.BatchScopeDigest != wantBatch {
		t.Fatalf("batch identity = %q/%q", scope.BatchID, scope.BatchScopeDigest)
	}
}

func TestFeatureScopeFailsClosed(t *testing.T) {
	t.Run("missing spec", func(t *testing.T) {
		root := repoTestInit(t)
		featureDir := repoTestFeature(t, root)
		if err := os.Remove(filepath.Join(featureDir, "spec.md")); err != nil {
			t.Fatal(err)
		}
		repo, err := OpenRepository(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FeatureScope(repo, featureDir, "B1"); !errors.Is(err, ErrUsage) {
			t.Fatalf("error = %v, want ErrUsage", err)
		}
	})
	t.Run("missing tasks", func(t *testing.T) {
		root := repoTestInit(t)
		featureDir := repoTestFeature(t, root)
		if err := os.Remove(filepath.Join(featureDir, "tasks.md")); err != nil {
			t.Fatal(err)
		}
		repo, err := OpenRepository(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FeatureScope(repo, featureDir, "B1"); !errors.Is(err, ErrUsage) {
			t.Fatalf("error = %v, want ErrUsage", err)
		}
	})
	t.Run("outside repository", func(t *testing.T) {
		root := repoTestInit(t)
		repo, err := OpenRepository(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FeatureScope(repo, t.TempDir(), "B1"); !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("error = %v, want ErrUnsafePath", err)
		}
	})
	t.Run("symlinked feature directory", func(t *testing.T) {
		root := repoTestInit(t)
		featureDir := repoTestFeature(t, root)
		link := filepath.Join(root, "specs", "linked")
		if err := os.Symlink(featureDir, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		repo, err := OpenRepository(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FeatureScope(repo, link, "B1"); !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("error = %v, want ErrUnsafePath", err)
		}
	})
	t.Run("missing batch", func(t *testing.T) {
		root := repoTestInit(t)
		featureDir := repoTestFeature(t, root)
		repo, err := OpenRepository(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FeatureScope(repo, featureDir, "B9"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("error = %v, want ErrNotFound", err)
		}
	})
	t.Run("duplicate batch", func(t *testing.T) {
		root := repoTestInit(t)
		featureDir := repoTestFeature(t, root)
		repoTestWrite(t, filepath.Join(featureDir, "tasks.md"), repoTestTasks+"\n## B1 — Duplicate\n")
		repo, err := OpenRepository(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FeatureScope(repo, featureDir, "B1"); !errors.Is(err, ErrConflict) {
			t.Fatalf("error = %v, want ErrConflict", err)
		}
	})
	t.Run("unsafe batch token", func(t *testing.T) {
		root := repoTestInit(t)
		featureDir := repoTestFeature(t, root)
		repo, err := OpenRepository(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FeatureScope(repo, featureDir, "../B1"); !errors.Is(err, ErrUsage) {
			t.Fatalf("error = %v, want ErrUsage", err)
		}
	})
}

func TestDirectScope(t *testing.T) {
	root := repoTestInit(t)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	scope, err := DirectScope(repo, "fix flaky test")
	if err != nil {
		t.Fatalf("direct scope: %v", err)
	}
	if scope.Kind != ScopeDirect || scope.Label != "fix flaky test" || scope.BatchID != "" {
		t.Fatalf("direct scope = %+v", scope)
	}
	if _, err := DirectScope(repo, "bad\x00label"); !errors.Is(err, ErrUsage) {
		t.Fatalf("control-character label error = %v, want ErrUsage", err)
	}
	if _, err := DirectScope(repo, strings.Repeat("x", 201)); !errors.Is(err, ErrUsage) {
		t.Fatalf("long label error = %v, want ErrUsage", err)
	}
}

func TestValidateRelativePath(t *testing.T) {
	cases := []struct {
		path string
		ok   bool
	}{
		{"a/b.txt", true},
		{"a", true},
		{"a/..b/c", true},
		{"", false},
		{"/etc/passwd", false},
		{"../escape", false},
		{"a/../../escape", false},
		{"./a", false},
		{"a//b", false},
		{"a/", false},
		{".", false},
		{"..", false},
		{"a\\b", false},
		{"a\x00b", false},
		{"a\nb", false},
		{strings.Repeat("a", 4097), false},
	}
	for _, testCase := range cases {
		err := ValidateRelativePath(testCase.path)
		if testCase.ok && err != nil {
			t.Errorf("ValidateRelativePath(%q) = %v, want nil", testCase.path, err)
		}
		if !testCase.ok && !errors.Is(err, ErrUnsafePath) {
			t.Errorf("ValidateRelativePath(%q) = %v, want ErrUnsafePath", testCase.path, err)
		}
	}
}

func TestPathIdentity(t *testing.T) {
	root := t.TempDir()
	if !withinPath(root, root) {
		t.Fatal("root must contain itself")
	}
	if !withinPath(root, filepath.Join(root, "a", "b")) {
		t.Fatal("nested path must be inside root")
	}
	if withinPath(root, filepath.Join(root, "..")) {
		t.Fatal("parent path must not be inside root")
	}
	if withinPath(root, filepath.Dir(root)) {
		t.Fatal("sibling path must not be inside root")
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ensureNoSymlink(root, "sub"); err != nil {
		t.Fatalf("regular component was refused: %v", err)
	}
	if err := ensureNoSymlink(root, "sub/missing/child"); err != nil {
		t.Fatalf("missing components were refused: %v", err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(sub, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := ensureNoSymlink(root, "sub/link/child"); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("symlinked component error = %v, want ErrUnsafePath", err)
	}
	resolved, relative, err := resolveInside(root, filepath.Join("sub", "file.txt"))
	if err != nil {
		t.Fatalf("resolve inside: %v", err)
	}
	if resolved != filepath.Join(root, "sub", "file.txt") || relative != "sub/file.txt" {
		t.Fatalf("resolved = %q / %q", resolved, relative)
	}
	if _, _, err := resolveInside(root, ".."); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("resolve parent error = %v, want ErrUnsafePath", err)
	}
}

func TestReadBoundedRegularRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := repoTestWrite(t, filepath.Join(dir, "target.txt"), "content")
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readBoundedRegular(link, 1024); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("readBoundedRegular(symlink) error = %v, want ErrUnsafePath", err)
	}
	if _, err := readBoundedRegular(target, 1); !errors.Is(err, ErrUsage) {
		t.Fatalf("readBoundedRegular(oversize) error = %v, want ErrUsage", err)
	}
}

func TestDigestBytesIsStable(t *testing.T) {
	first := DigestBytes([]byte("example"))
	second := DigestBytes([]byte("example"))
	if first != second || !strings.HasPrefix(first, "sha256:") {
		t.Fatalf("digest = %q / %q", first, second)
	}
	if DigestBytes([]byte("example ")) == first {
		t.Fatal("digest did not change for different input")
	}
}
