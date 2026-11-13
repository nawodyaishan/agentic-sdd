package execution

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupReviewRepo initializes a test Git repo with a baseline commit and an initialized run.
func setupReviewRepo(t *testing.T) (string, string, RunIdentity, string) {
	t.Helper()
	root := repoTestInit(t)
	featureDir := repoTestFeature(t, root)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	baseCommit := repo.Head

	scope, err := FeatureScope(repo, featureDir, "B1")
	if err != nil {
		t.Fatalf("feature scope: %v", err)
	}
	identity, err := InitRun(InitRequest{Scope: scope, ToolVersion: "test"})
	if err != nil {
		t.Fatalf("init run: %v", err)
	}
	return root, identity.RunDir, identity, baseCommit
}

func TestReview_Committed(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, "pkg", "code.go"), "package pkg\n")
	repoTestWrite(t, filepath.Join(root, "README.md"), "updated baseline\n")
	repoTestCommit(t, root, "commit 2")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected manifest to be complete, got partial reasons: %v", manifest.PartialReasons)
	}
	if manifest.Counts.Committed != 2 {
		t.Fatalf("expected 2 committed files, got %d", manifest.Counts.Committed)
	}
	if len(manifest.Files) != 2 {
		t.Fatalf("expected 2 files in manifest, got %d", len(manifest.Files))
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if !strings.Contains(patchContent, "# === committed ===") {
		t.Fatalf("patch missing committed header: %s", patchContent)
	}
	if !strings.Contains(patchContent, "pkg/code.go") {
		t.Fatalf("patch missing pkg/code.go diff: %s", patchContent)
	}
}

func TestReview_Staged(t *testing.T) {
	root, runDir, _, _ := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, "staged.txt"), "staged content\n")
	repoTestGit(t, root, "add", "staged.txt")

	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   repo.Head,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got partial: %v", manifest.PartialReasons)
	}
	if manifest.Counts.Staged != 1 {
		t.Fatalf("expected 1 staged count, got %d", manifest.Counts.Staged)
	}

	var stagedFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "staged.txt" {
			stagedFile = &manifest.Files[i]
			break
		}
	}
	if stagedFile == nil || !stagedFile.Staged {
		t.Fatalf("staged.txt not flagged as staged: %+v", stagedFile)
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if !strings.Contains(patchContent, "# === staged ===") || !strings.Contains(patchContent, "staged content") {
		t.Fatalf("patch missing staged diff: %s", patchContent)
	}
}

func TestReview_Unstaged(t *testing.T) {
	root, runDir, _, _ := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, "README.md"), "unstaged dirty write\n")

	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   repo.Head,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got partial: %v", manifest.PartialReasons)
	}
	if manifest.Counts.Unstaged != 1 {
		t.Fatalf("expected 1 unstaged count, got %d", manifest.Counts.Unstaged)
	}

	var unstagedFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "README.md" {
			unstagedFile = &manifest.Files[i]
			break
		}
	}
	if unstagedFile == nil || !unstagedFile.Unstaged {
		t.Fatalf("README.md not flagged as unstaged: %+v", unstagedFile)
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if !strings.Contains(patchContent, "# === unstaged ===") || !strings.Contains(patchContent, "unstaged dirty write") {
		t.Fatalf("patch missing unstaged diff: %s", patchContent)
	}
}

func TestReview_Untracked(t *testing.T) {
	root, runDir, _, _ := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, "untracked", "hello.txt"), "hello untracked\n")

	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   repo.Head,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got partial: %v", manifest.PartialReasons)
	}
	if manifest.Counts.Untracked != 1 {
		t.Fatalf("expected 1 untracked count, got %d", manifest.Counts.Untracked)
	}

	copied := filepath.Join(manifest.UntrackedDir, "untracked", "hello.txt")
	content := repoTestRead(t, copied)
	if content != "hello untracked\n" {
		t.Fatalf("copied untracked file content mismatch: %q", content)
	}
	info, err := os.Stat(copied)
	if err != nil {
		t.Fatalf("stat copied untracked: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 mode on copied untracked file, got %v", info.Mode().Perm())
	}
}

func TestReview_Rename(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestGit(t, root, "mv", "README.md", "README_NEW.md")
	repoTestCommit(t, root, "rename file")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got %v", manifest.PartialReasons)
	}

	var renFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "README_NEW.md" {
			renFile = &manifest.Files[i]
			break
		}
	}
	if renFile == nil {
		t.Fatalf("README_NEW.md not found in files: %+v", manifest.Files)
	}
	if renFile.Status != "R" {
		t.Fatalf("expected status 'R', got %q", renFile.Status)
	}
	if renFile.OldPath != "README.md" {
		t.Fatalf("expected old path 'README.md', got %q", renFile.OldPath)
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if !strings.Contains(patchContent, "rename from README.md") || !strings.Contains(patchContent, "rename to README_NEW.md") {
		t.Fatalf("patch missing rename header: %s", patchContent)
	}
}

func TestReview_Delete(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestGit(t, root, "rm", "README.md")
	repoTestCommit(t, root, "delete readme")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got %v", manifest.PartialReasons)
	}

	var delFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "README.md" {
			delFile = &manifest.Files[i]
			break
		}
	}
	if delFile == nil || delFile.Status != "D" {
		t.Fatalf("expected deleted file entry: %+v", delFile)
	}
}

func TestReview_ModeChange(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	script := filepath.Join(root, "run.sh")
	repoTestWrite(t, script, "#!/bin/sh\necho hello\n")
	repoTestGit(t, root, "add", "run.sh")
	repoTestCommit(t, root, "add script")

	baseCommit2 := repoTestGit(t, root, "rev-parse", "HEAD")

	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatalf("chmod script: %v", err)
	}
	repoTestGit(t, root, "add", "run.sh")
	repoTestCommit(t, root, "chmod script")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got %v", manifest.PartialReasons)
	}

	var modFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "run.sh" {
			modFile = &manifest.Files[i]
			break
		}
	}
	if modFile == nil || modFile.Mode != "100755" {
		t.Fatalf("expected mode 100755, got %+v", modFile)
	}
	_ = baseCommit
}

func TestReview_Binary(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	binPath := filepath.Join(root, "data.bin")
	if err := os.WriteFile(binPath, []byte{0x00, 0x01, 0x02, 0xff}, 0o600); err != nil {
		t.Fatalf("write bin: %v", err)
	}
	repoTestGit(t, root, "add", "data.bin")
	repoTestCommit(t, root, "add binary")

	base2 := repoTestGit(t, root, "rev-parse", "HEAD")

	if err := os.WriteFile(binPath, []byte{0x00, 0x01, 0x02, 0xfe, 0xaa}, 0o600); err != nil {
		t.Fatalf("write bin: %v", err)
	}
	repoTestCommit(t, root, "update binary")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   base2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got %v", manifest.PartialReasons)
	}

	var binFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "data.bin" {
			binFile = &manifest.Files[i]
			break
		}
	}
	if binFile == nil || !binFile.Binary {
		t.Fatalf("expected binary file flagged binary: %+v", binFile)
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if !strings.Contains(patchContent, "GIT binary patch") {
		t.Fatalf("expected GIT binary patch in diff: %s", patchContent)
	}
	_ = baseCommit
}

func TestReview_LargeFile(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	largeData := strings.Repeat("A", 10000)
	repoTestWrite(t, filepath.Join(root, "large.txt"), largeData)
	repoTestCommit(t, root, "add large file")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir:       runDir,
		Base:         baseCommit,
		MaxFileBytes: 1000,
	})
	if !errors.Is(err, ErrPartial) {
		t.Fatalf("expected ErrPartial, got: %v", err)
	}
	if manifest.Complete {
		t.Fatalf("expected incomplete manifest")
	}

	foundReason := false
	for _, r := range manifest.PartialReasons {
		if r == "max_file_bytes_exceeded" {
			foundReason = true
			break
		}
	}
	if !foundReason {
		t.Fatalf("expected max_file_bytes_exceeded in partial reasons: %v", manifest.PartialReasons)
	}

	var largeFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "large.txt" {
			largeFile = &manifest.Files[i]
			break
		}
	}
	if largeFile == nil || largeFile.Included {
		t.Fatalf("expected large.txt to be omitted: %+v", largeFile)
	}
}

func TestReview_MaxTotalBytes(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, "file1.txt"), strings.Repeat("B", 300))
	repoTestWrite(t, filepath.Join(root, "file2.txt"), strings.Repeat("C", 300))
	repoTestCommit(t, root, "add two files")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir:        runDir,
		Base:          baseCommit,
		MaxFileBytes:  2000,
		MaxTotalBytes: 500, // file1 will fit, file2 will exceed total
	})
	if !errors.Is(err, ErrPartial) {
		t.Fatalf("expected ErrPartial, got: %v", err)
	}
	if manifest.Complete {
		t.Fatalf("expected incomplete manifest")
	}

	foundTotalReason := false
	for _, r := range manifest.PartialReasons {
		if r == "max_total_bytes_exceeded" {
			foundTotalReason = true
			break
		}
	}
	if !foundTotalReason {
		t.Fatalf("expected max_total_bytes_exceeded in partial reasons: %v", manifest.PartialReasons)
	}
}

func TestReview_Sensitive(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, ".env"), "SECRET_KEY=12345\n")
	repoTestWrite(t, filepath.Join(root, "keys", "server.key"), "PRIVATE KEY\n")
	repoTestWrite(t, filepath.Join(root, "id_rsa"), "OPENSSH PRIVATE KEY\n")
	repoTestWrite(t, filepath.Join(root, "credentials.json"), "{\"token\": \"xyz\"}\n")
	repoTestCommit(t, root, "add sensitive files")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
	})
	if !errors.Is(err, ErrPartial) {
		t.Fatalf("expected ErrPartial, got: %v", err)
	}
	if manifest.Complete {
		t.Fatalf("expected incomplete manifest")
	}

	foundSensitive := false
	for _, r := range manifest.PartialReasons {
		if r == "sensitive_path" {
			foundSensitive = true
			break
		}
	}
	if !foundSensitive {
		t.Fatalf("expected sensitive_path in partial reasons: %v", manifest.PartialReasons)
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if strings.Contains(patchContent, "SECRET_KEY") || strings.Contains(patchContent, "OPENSSH") {
		t.Fatalf("patch leaked sensitive content: %s", patchContent)
	}

	for _, f := range manifest.Files {
		if strings.HasSuffix(f.Path, ".env") || strings.HasSuffix(f.Path, "server.key") || strings.HasSuffix(f.Path, "id_rsa") {
			if f.Included {
				t.Fatalf("sensitive file %s was included in review package", f.Path)
			}
			if f.Reason != "sensitive_path" {
				t.Fatalf("expected reason sensitive_path for %s, got %s", f.Path, f.Reason)
			}
		}
	}
}

func TestReview_Symlink(t *testing.T) {
	root, runDir, _, _ := setupReviewRepo(t)

	target := filepath.Join(root, "README.md")
	symlinkPath := filepath.Join(root, "link_to_readme")
	if err := os.Symlink(target, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   repo.Head,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got %v", manifest.PartialReasons)
	}

	copied := filepath.Join(manifest.UntrackedDir, "link_to_readme")
	info, err := os.Lstat(copied)
	if err != nil {
		t.Fatalf("lstat copied link: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("copied link should not be a symlink itself, but a metadata file")
	}
	data := repoTestRead(t, copied)
	if !strings.Contains(data, "README.md") {
		t.Fatalf("expected symlink metadata containing target path, got %q", data)
	}
}

func TestReview_SubmoduleGitlink(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestGit(t, root, "update-index", "--add", "--cacheinfo", "160000", baseCommit, "submodule_dir")
	repoTestGit(t, root, "commit", "-m", "add submodule gitlink")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !manifest.Complete {
		t.Fatalf("expected complete, got %v", manifest.PartialReasons)
	}

	var subFile *ReviewFile
	for i := range manifest.Files {
		if manifest.Files[i].Path == "submodule_dir" {
			subFile = &manifest.Files[i]
			break
		}
	}
	if subFile == nil || !subFile.Submodule {
		t.Fatalf("expected submodule file flagged submodule: %+v", subFile)
	}
}

func TestReview_ConflictedIndex(t *testing.T) {
	root, runDir, _, _ := setupReviewRepo(t)

	baseBranch := repoTestGit(t, root, "symbolic-ref", "--short", "HEAD")

	repoTestWrite(t, filepath.Join(root, "conflict.txt"), "base\n")
	repoTestGit(t, root, "add", "conflict.txt")
	repoTestGit(t, root, "commit", "-m", "base conflict")

	repoTestGit(t, root, "branch", "feat")
	repoTestWrite(t, filepath.Join(root, "conflict.txt"), "base edit\n")
	repoTestGit(t, root, "commit", "-am", "base edit")

	repoTestGit(t, root, "checkout", "feat")
	repoTestWrite(t, filepath.Join(root, "conflict.txt"), "feat edit\n")
	repoTestGit(t, root, "commit", "-am", "feat edit")

	// Trigger merge conflict with the base branch
	cmd := exec.Command("git", "merge", baseBranch)
	cmd.Dir = root
	cmd.Env = repoTestEnv(t)
	_ = cmd.Run()

	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   repo.Head,
	})
	if !errors.Is(err, ErrPartial) {
		t.Fatalf("expected ErrPartial, got: %v", err)
	}
	if manifest.Complete {
		t.Fatalf("expected incomplete manifest")
	}

	foundConflict := false
	for _, r := range manifest.PartialReasons {
		if r == "index_conflict" {
			foundConflict = true
			break
		}
	}
	if !foundConflict {
		t.Fatalf("expected index_conflict in partial reasons: %v", manifest.PartialReasons)
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if !strings.Contains(patchContent, "omitted: index_conflict") {
		t.Fatalf("expected patch to note index_conflict omission: %s", patchContent)
	}
}

func TestReview_HeadMismatch(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, "f1.txt"), "1\n")
	repoTestCommit(t, root, "commit 1")
	head1 := repoTestGit(t, root, "rev-parse", "HEAD")

	repoTestWrite(t, filepath.Join(root, "f2.txt"), "2\n")
	repoTestCommit(t, root, "commit 2")

	// Create unstaged/untracked dirty worktree
	repoTestWrite(t, filepath.Join(root, "untracked.txt"), "untracked\n")

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
		Head:   head1, // Head differs from current HEAD
	})
	if !errors.Is(err, ErrPartial) {
		t.Fatalf("expected ErrPartial, got: %v", err)
	}
	if manifest.Complete {
		t.Fatalf("expected incomplete manifest")
	}

	foundHeadMismatch := false
	for _, r := range manifest.PartialReasons {
		if r == "head_not_current" {
			foundHeadMismatch = true
			break
		}
	}
	if !foundHeadMismatch {
		t.Fatalf("expected head_not_current in partial reasons: %v", manifest.PartialReasons)
	}

	patchContent := repoTestRead(t, manifest.PatchPath)
	if !strings.Contains(patchContent, "omitted: head_not_current") {
		t.Fatalf("expected patch to note head_not_current: %s", patchContent)
	}
}

func TestReview_NoMutation(t *testing.T) {
	root, runDir, _, baseCommit := setupReviewRepo(t)

	repoTestWrite(t, filepath.Join(root, "staged.txt"), "staged\n")
	repoTestGit(t, root, "add", "staged.txt")
	repoTestWrite(t, filepath.Join(root, "README.md"), "unstaged modification\n")
	repoTestWrite(t, filepath.Join(root, "untracked.txt"), "untracked new\n")

	statusBefore := repoTestGit(t, root, "status", "--porcelain=v1", "-z")

	_, _ = BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
	})

	statusAfter := repoTestGit(t, root, "status", "--porcelain=v1", "-z")
	if statusBefore != statusAfter {
		t.Fatalf("worktree/index was mutated!\nBefore: %q\nAfter:  %q", statusBefore, statusAfter)
	}
}

func TestReview_InvalidBase(t *testing.T) {
	_, runDir, _, _ := setupReviewRepo(t)

	_, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   "invalid-base-commit-hash-that-does-not-exist",
	})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage for invalid base, got: %v", err)
	}

	_, err = BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   "",
	})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage for empty base, got: %v", err)
	}
}

func TestReview_CheckpointIntegration(t *testing.T) {
	_, runDir, identity, baseCommit := setupReviewRepo(t)

	manifest, err := BuildReviewPackage(ReviewPackageRequest{
		RunDir: runDir,
		Base:   baseCommit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkpoint, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if len(checkpoint.ReviewPackages) != 1 {
		t.Fatalf("expected 1 review package in checkpoint, got %d", len(checkpoint.ReviewPackages))
	}
	rec := checkpoint.ReviewPackages[0]
	if rec.PackageID != manifest.PackageID {
		t.Fatalf("package ID mismatch: %q vs %q", rec.PackageID, manifest.PackageID)
	}
	if rec.ManifestPath != filepath.Join(runDir, "review", manifest.PackageID, "review.json") {
		t.Fatalf("manifest path mismatch: %q", rec.ManifestPath)
	}
	if checkpoint.LastObservation == nil {
		t.Fatalf("expected last observation to be set in checkpoint")
	}
	_ = identity
}
