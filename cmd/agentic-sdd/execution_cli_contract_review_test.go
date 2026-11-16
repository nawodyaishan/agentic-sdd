package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"agentic-sdd/internal/execution"
)

var reviewPackageIDPattern = regexp.MustCompile(`^pkg-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{6}$`)

func reviewTestGit(t *testing.T, dir string, allowFailure bool, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"LC_ALL=C",
	)
	out, err := cmd.CombinedOutput()
	if err != nil && !allowFailure {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func assertPartialReviewManifestEnvelope(t *testing.T, raw, wantRunID string) execution.ReviewManifest {
	t.Helper()
	var manifest execution.ReviewManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatalf("decode review manifest: %v\n%s", err, raw)
	}
	if manifest.SchemaVersion != execution.SchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", manifest.SchemaVersion, execution.SchemaVersion)
	}
	if manifest.Kind != "execution.review.package" {
		t.Fatalf("kind = %q, want execution.review.package", manifest.Kind)
	}
	if !reviewPackageIDPattern.MatchString(manifest.PackageID) {
		t.Fatalf("package_id = %q, does not match expected ID grammar", manifest.PackageID)
	}
	if manifest.RunID != wantRunID {
		t.Fatalf("run_id = %q, want %q", manifest.RunID, wantRunID)
	}
	if manifest.CreatedAt.IsZero() {
		t.Fatal("created_at is zero")
	}
	if manifest.RepoRoot == "" || manifest.Base == "" || manifest.Head == "" || manifest.PatchPath == "" {
		t.Fatalf("missing required non-empty manifest fields: %+v", manifest)
	}
	if manifest.Complete {
		t.Fatalf("manifest claimed complete=true for partial package: %+v", manifest)
	}
	if len(manifest.PartialReasons) == 0 {
		t.Fatal("partial_reasons is empty for partial package")
	}
	if len(manifest.Omissions) == 0 {
		t.Fatal("omissions is empty for partial package")
	}
	if manifest.Counts.Omitted != len(manifest.Omissions) {
		t.Fatalf("counts.omitted = %d, want len(omissions) = %d", manifest.Counts.Omitted, len(manifest.Omissions))
	}

	manifestFilePath := filepath.Join(filepath.Dir(manifest.PatchPath), "review.json")
	data, err := os.ReadFile(manifestFilePath)
	if err != nil {
		t.Fatalf("read review.json: %v", err)
	}
	var persisted execution.ReviewManifest
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("decode persisted review.json: %v", err)
	}
	if persisted.Complete {
		t.Fatalf("persisted review.json claimed complete=true: %+v", persisted)
	}
	return manifest
}

func assertCheckpointReviewPackageNotComplete(t *testing.T, runDir, packageID, wantReason string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir, "checkpoint.json"))
	if err != nil {
		t.Fatalf("read checkpoint.json: %v", err)
	}
	var checkpoint execution.Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		t.Fatalf("decode checkpoint.json: %v", err)
	}
	for _, pkg := range checkpoint.ReviewPackages {
		if pkg.PackageID != packageID {
			continue
		}
		if pkg.Complete {
			t.Fatalf("checkpoint review package %s claimed complete=true", packageID)
		}
		if !slices.Contains(pkg.PartialReasons, wantReason) {
			t.Fatalf("checkpoint review package %s partial_reasons = %v, want %q", packageID, pkg.PartialReasons, wantReason)
		}
		return
	}
	t.Fatalf("checkpoint.json missing review package %s", packageID)
}

func assertHasReviewOmission(t *testing.T, omissions []execution.ReviewOmission, wantPath, wantReason string) {
	t.Helper()
	for _, omission := range omissions {
		if omission.Path == wantPath && omission.Reason == wantReason {
			return
		}
	}
	t.Fatalf("omissions %+v missing (%q, %q)", omissions, wantPath, wantReason)
}

func TestExecutionContractReviewPartialReasons(t *testing.T) {
	t.Run("head_not_current", func(t *testing.T) {
		dir := executionTestRepo(t)
		base := executionGitOutput(t, dir, "rev-parse", "HEAD")

		code, out, errOut := runExecutionCLI(t, dir, "run", "init", "--feature", "feature", "--batch", "B1", "--format", "json")
		if code != 0 {
			t.Fatalf("run init exit %d: %s", code, errOut)
		}
		identity := decodeRunIdentity(t, out)

		if err := os.WriteFile(filepath.Join(dir, "first.txt"), []byte("first commit\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		reviewTestGit(t, dir, false, "add", "first.txt")
		reviewTestGit(t, dir, false, "commit", "-qm", "first")
		olderHead := executionGitOutput(t, dir, "rev-parse", "HEAD")

		if err := os.WriteFile(filepath.Join(dir, "second.txt"), []byte("second commit\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		reviewTestGit(t, dir, false, "add", "second.txt")
		reviewTestGit(t, dir, false, "commit", "-qm", "second")

		if err := os.WriteFile(filepath.Join(dir, "dirty-untracked.txt"), []byte("untracked worktree file\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		code, out, errOut = runExecutionCLI(t, dir, "review", "package", "--run", identity.RunID, "--base", base, "--head", olderHead, "--format", "json")
		if code != 3 {
			t.Fatalf("review package exit %d, want 3; stdout=%q stderr=%q", code, out, errOut)
		}
		if errOut != "" {
			t.Fatalf("unexpected stderr on partial JSON review package: %q", errOut)
		}

		manifest := assertPartialReviewManifestEnvelope(t, out, identity.RunID)
		if manifest.Base != base || manifest.Head != olderHead || manifest.HeadRequested != olderHead {
			t.Fatalf("manifest commits = (base=%q head=%q requested=%q), want (%q %q %q)", manifest.Base, manifest.Head, manifest.HeadRequested, base, olderHead, olderHead)
		}
		if !slices.Equal(manifest.PartialReasons, []string{"head_not_current"}) {
			t.Fatalf("partial_reasons = %v, want [head_not_current]", manifest.PartialReasons)
		}
		if manifest.Counts.Committed != 1 || manifest.Counts.Staged != 0 || manifest.Counts.Unstaged != 0 || manifest.Counts.Untracked != 0 {
			t.Fatalf("unexpected counts for head_not_current: %+v", manifest.Counts)
		}
		if manifest.UntrackedDir != "" {
			t.Fatalf("untracked_dir = %q, want empty when head_not_current", manifest.UntrackedDir)
		}

		assertHasReviewOmission(t, manifest.Omissions, "(staged)", "head_not_current")
		assertHasReviewOmission(t, manifest.Omissions, "(unstaged)", "head_not_current")
		assertHasReviewOmission(t, manifest.Omissions, "(untracked)", "head_not_current")

		var sawFirst bool
		for _, f := range manifest.Files {
			switch f.Path {
			case "first.txt":
				sawFirst = f.Included
			case "second.txt", "dirty-untracked.txt":
				t.Fatalf("unexpected file %q in head_not_current manifest: %+v", f.Path, manifest.Files)
			}
		}
		if !sawFirst {
			t.Fatalf("expected committed first.txt included in manifest: %+v", manifest.Files)
		}

		patchBytes, err := os.ReadFile(manifest.PatchPath)
		if err != nil {
			t.Fatalf("read review.patch: %v", err)
		}
		patch := string(patchBytes)
		if !strings.Contains(patch, "# omitted: head_not_current") {
			t.Fatalf("review.patch missing head_not_current omission marker:\n%s", patch)
		}
		if strings.Contains(patch, "dirty-untracked.txt") || strings.Contains(patch, "second.txt") {
			t.Fatalf("review.patch leaked out-of-scope worktree/newer-head paths:\n%s", patch)
		}

		assertCheckpointReviewPackageNotComplete(t, identity.RunDir, manifest.PackageID, "head_not_current")
	})

	t.Run("index_conflict", func(t *testing.T) {
		dir := executionTestRepo(t)
		baseBranch := executionGitOutput(t, dir, "symbolic-ref", "--short", "HEAD")

		if err := os.WriteFile(filepath.Join(dir, "conflict.txt"), []byte("base\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		reviewTestGit(t, dir, false, "add", "conflict.txt")
		reviewTestGit(t, dir, false, "commit", "-qm", "base conflict")
		base := executionGitOutput(t, dir, "rev-parse", "HEAD")

		reviewTestGit(t, dir, false, "branch", "feat")
		if err := os.WriteFile(filepath.Join(dir, "conflict.txt"), []byte("base edit\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		reviewTestGit(t, dir, false, "commit", "-qam", "base edit")

		reviewTestGit(t, dir, false, "checkout", "-q", "feat")
		if err := os.WriteFile(filepath.Join(dir, "conflict.txt"), []byte("feat edit\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		reviewTestGit(t, dir, false, "commit", "-qam", "feat edit")

		reviewTestGit(t, dir, true, "merge", baseBranch)
		if unmerged := executionGitOutput(t, dir, "ls-files", "-u"); unmerged == "" {
			t.Fatal("expected unmerged index entries for conflict fixture")
		}

		code, out, errOut := runExecutionCLI(t, dir, "run", "init", "--feature", "feature", "--batch", "B1", "--format", "json")
		if code != 0 {
			t.Fatalf("run init exit %d: %s", code, errOut)
		}
		identity := decodeRunIdentity(t, out)

		code, out, errOut = runExecutionCLI(t, dir, "review", "package", "--run", identity.RunID, "--base", base, "--format", "json")
		if code != 3 {
			t.Fatalf("review package exit %d, want 3; stdout=%q stderr=%q", code, out, errOut)
		}
		if errOut != "" {
			t.Fatalf("unexpected stderr on partial JSON review package: %q", errOut)
		}

		manifest := assertPartialReviewManifestEnvelope(t, out, identity.RunID)
		if manifest.Base != base || manifest.Head != executionGitOutput(t, dir, "rev-parse", "HEAD") {
			t.Fatalf("manifest commits = (base=%q head=%q)", manifest.Base, manifest.Head)
		}
		if !slices.Equal(manifest.PartialReasons, []string{"index_conflict"}) {
			t.Fatalf("partial_reasons = %v, want [index_conflict]", manifest.PartialReasons)
		}
		if manifest.Counts.Staged != 0 {
			t.Fatalf("counts.staged = %d, want 0 when index_conflict omits cached diff", manifest.Counts.Staged)
		}

		assertHasReviewOmission(t, manifest.Omissions, "(index)", "index_conflict")
		assertHasReviewOmission(t, manifest.Omissions, "conflict.txt", "index_conflict")

		var sawConflictFile bool
		for _, f := range manifest.Files {
			if f.Staged {
				t.Fatalf("unexpected staged file %+v when index is conflicted", f)
			}
			if f.Path == "conflict.txt" {
				sawConflictFile = true
				if f.Status != "U" || f.Included || f.Reason != "index_conflict" {
					t.Fatalf("conflict.txt entry = %+v, want status=U included=false reason=index_conflict", f)
				}
			}
		}
		if !sawConflictFile {
			t.Fatalf("manifest.Files missing conflict.txt: %+v", manifest.Files)
		}

		patchBytes, err := os.ReadFile(manifest.PatchPath)
		if err != nil {
			t.Fatalf("read review.patch: %v", err)
		}
		patch := string(patchBytes)
		_, afterStaged, ok := strings.Cut(patch, "# === staged ===\n")
		if !ok {
			t.Fatalf("review.patch missing staged header:\n%s", patch)
		}
		stagedSection, _, ok := strings.Cut(afterStaged, "\n# === unstaged ===")
		if !ok {
			t.Fatalf("review.patch missing unstaged header:\n%s", patch)
		}
		if !strings.Contains(stagedSection, "# omitted: index_conflict") {
			t.Fatalf("staged section missing index_conflict omission:\n%s", stagedSection)
		}
		if strings.Contains(stagedSection, "diff --git") {
			t.Fatalf("staged section unexpectedly included cached diff:\n%s", stagedSection)
		}

		assertCheckpointReviewPackageNotComplete(t, identity.RunDir, manifest.PackageID, "index_conflict")
	})
}
