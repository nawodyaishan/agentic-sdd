package skillsync

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Feature 008 B3 / T05 permission-contract characterization tests.
//
// These tests pin the installer's *existing* permission, no-op, backup and
// restore semantics. They assert current behavior and change no production
// code. The behavior they describe, read from internal/skillsync source:
//
//   - Change detection compares directory structure and file content only
//     (sameSourceTarget in source.go). Permission bits are ignored, so a
//     permission-only difference is a no-op: nothing is rewritten, no new
//     backup is written, and the differing modes are left as they are.
//   - Installed trees are written with fixed requested modes: 0755 for
//     directories and 0644 for files (installedDirMode/installedFileMode in
//     source.go). Creation is masked by the process umask, so the requested
//     modes are upper bounds rather than guarantees.
//   - The root of an installed skill directory is chmod'ed to 0755 after
//     staging (installChanges in sync.go), so that one directory is
//     umask-independent while nested directories keep the umask-masked mode
//     their creation produced.
//   - Backups copy the replaced tree with copyTree (files.go), which
//     preserves the source mode bits, masked by the same umask.
//   - The manifest digest (treeDigest in manifest.go) covers relative paths,
//     entry kinds and file contents, not modes.
//   - Restore reinstalls through the same staged path, so it writes the
//     requested modes again: a restore returns the old content, not the old
//     modes.
//
// The umask-dependent observations run in a test-owned child process whose
// umask is set by the shell that starts it, so no Go test process ever
// changes its own process-global umask. The child is this same test binary,
// filtered to the single probe test.

const (
	// permSyntheticSkill is the on-disk source skill used by the probes.
	// It has one root file and one nested directory with a file, so both
	// "root of the skill directory" and "nested" modes are observable.
	permSyntheticSkill = "agentic-sdd-perms"
	// permEmbeddedSkill is an embedded skill that also has a nested
	// references/ directory, so the same observations apply to the
	// embedded source mode.
	permEmbeddedSkill = "agentic-sdd-router"

	permNewContent = "new content\n"
	permOldContent = "old content\n"

	permScenarioInstall = "install"
	permScenarioRestore = "restore"
)

// permProbeEnv* are the environment variables the parent test uses to
// configure the child probe process.
const (
	permProbeWorkEnv     = "AGENTIC_SDD_PERM_PROBE_WORK"
	permProbeScenarioEnv = "AGENTIC_SDD_PERM_PROBE_SCENARIO"
	permProbeSourceEnv   = "AGENTIC_SDD_PERM_PROBE_SOURCE"
	permProbeSkillEnv    = "AGENTIC_SDD_PERM_PROBE_SKILL"
	permProbeFileModeEnv = "AGENTIC_SDD_PERM_PROBE_FILE_MODE"
	permProbeDirModeEnv  = "AGENTIC_SDD_PERM_PROBE_DIR_MODE"
)

// permOriginalModes returns the modes the probe gives the pre-existing
// destination tree before the apply. They are chosen per umask so that the
// mode a backup preserves and the mode a restore writes are always
// observably different. With umask 022 a 0600 file is restored as 0644.
// With umask 077, 0755 is backed up as 0700 and restored as 0600; the
// removed group/other bits also make backup umask behavior observable.
func permOriginalModes(mask string) (file os.FileMode, dir os.FileMode) {
	if mask == "077" {
		return 0755, 0755
	}
	return 0600, 0700
}

// permParseMask turns a "022"/"077" string into a file mode.
func permParseMask(t *testing.T, mask string) os.FileMode {
	t.Helper()
	value, err := strconv.ParseUint(mask, 8, 32)
	if err != nil {
		t.Fatalf("umask %q: %v", mask, err)
	}
	return os.FileMode(value)
}

// permWriteSource writes the synthetic on-disk source tree (a root file and
// a nested directory holding one file) and returns the repo path.
func permWriteSource(t *testing.T, repo, skill string) string {
	t.Helper()
	src := filepath.Join(repo, "skills-files", skill)
	if err := os.MkdirAll(filepath.Join(src, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	permWriteFile(t, filepath.Join(src, "SKILL.md"), permNewContent)
	permWriteFile(t, filepath.Join(src, "references", "nested.md"), "nested reference\n")
	return repo
}

func permWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// permTreeModes maps every path in the tree at root (relative, "." for the
// root itself) to its permission bits.
func permTreeModes(t *testing.T, root string) map[string]os.FileMode {
	t.Helper()
	result := map[string]os.FileMode{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(rel)] = info.Mode().Perm()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// permChmodTree sets every directory in the tree to dirMode and every file
// to fileMode. chmod is not affected by the umask, so the requested modes are
// set exactly.
func permChmodTree(t *testing.T, root string, dirMode, fileMode os.FileMode) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, dirMode)
		}
		return os.Chmod(path, fileMode)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// permBackupCount reports how many backups exist under repo/home's backup
// root.
func permBackupCount(t *testing.T, repo, home string) int {
	t.Helper()
	backups, err := ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	return len(backups)
}

// permRepoFor returns the repo path for a source mode: the empty string
// selects the embedded skills, anything else the on-disk checkout.
func permRepoFor(source, work string) string {
	if source == "on-disk" {
		return filepath.Join(work, "repo")
	}
	return ""
}

// permSkillFor returns the skill name inspected for a source mode.
func permSkillFor(source string) string {
	if source == "on-disk" {
		return permSyntheticSkill
	}
	return permEmbeddedSkill
}

// permRemovedFileFor names a file of the installed tree used as the negative
// control: removing it must make the next apply report a replacement, which
// proves a permission-only difference is not silently accepted as a no-op.
func permRemovedFileFor(source string) string {
	if source == "on-disk" {
		return "references/nested.md"
	}
	return "references/human-input.md"
}

// TestPermissionOnlyDifferenceIsNoOpWithoutNewBackup characterizes the
// installer's change detection: a difference in permission bits alone is not
// a difference the installer acts on. Nothing is rewritten, no new backup is
// written, and the differing modes are left untouched (there is no
// chmod-on-no-op).
func TestPermissionOnlyDifferenceIsNoOpWithoutNewBackup(t *testing.T) {
	for _, source := range []string{"embedded", "on-disk"} {
		t.Run("apply "+source, func(t *testing.T) {
			work := t.TempDir()
			home := filepath.Join(work, "home")
			repo := permRepoFor(source, work)
			skill := permSkillFor(source)
			if source == "on-disk" {
				permWriteSource(t, repo, skill)
			}

			var out bytes.Buffer
			if err := Sync(repo, home, true, &out); err != nil {
				t.Fatalf("apply: %v\n%s", err, out.String())
			}
			dest := filepath.Join(home, ".claude", "skills", skill)
			installed := permTreeModes(t, dest)
			if permBackupCount(t, repo, home) != 1 {
				t.Fatalf("expected exactly one backup after the first apply")
			}

			// Change only the permission bits of the installed tree.
			permChmodTree(t, dest, 0700, 0600)
			chmodded := permTreeModes(t, dest)
			if reflect.DeepEqual(installed, chmodded) {
				t.Fatal("permission-only change did not take effect")
			}

			// A preview reports the permission-only difference as up to date
			// and writes nothing.
			out.Reset()
			if err := Sync(repo, home, false, &out); err != nil {
				t.Fatalf("preview: %v", err)
			}
			if !strings.Contains(out.String(), "up to date claude/"+skill) {
				t.Fatalf("preview did not report the skill as up to date:\n%s", out.String())
			}
			if got := permBackupCount(t, repo, home); got != 1 {
				t.Fatalf("preview created a backup: %d backups", got)
			}

			// An apply is a no-op: no rewrite, no new backup.
			out.Reset()
			if err := Sync(repo, home, true, &out); err != nil {
				t.Fatalf("second apply: %v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), "All skills are up to date.") {
				t.Fatalf("permission-only difference was not a no-op:\n%s", out.String())
			}
			if got := permBackupCount(t, repo, home); got != 1 {
				t.Fatalf("permission-only difference created a backup: %d backups", got)
			}
			if got := permTreeModes(t, dest); !reflect.DeepEqual(got, chmodded) {
				t.Fatalf("no-op rewrote permission bits: got %v, want %v", got, chmodded)
			}

			// Negative control: a structural difference is still a change,
			// so the no-op result above is not a false positive.
			removed := filepath.Join(dest, filepath.FromSlash(permRemovedFileFor(source)))
			if err := os.Remove(removed); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			if err := Sync(repo, home, true, &out); err != nil {
				t.Fatalf("apply after removal: %v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), "replace + back up claude/"+skill) {
				t.Fatalf("removed file was not reported as a replacement:\n%s", out.String())
			}
			if got := permBackupCount(t, repo, home); got != 2 {
				t.Fatalf("replacement did not create a backup: %d backups", got)
			}
		})

		t.Run("restore "+source, func(t *testing.T) {
			work := t.TempDir()
			home := filepath.Join(work, "home")
			repo := permRepoFor(source, work)
			skill := permSkillFor(source)
			if source == "on-disk" {
				permWriteSource(t, repo, skill)
			}
			// A pre-existing skill under claude: the apply replaces it and
			// saves its tree, which is what a later restore compares against.
			dest := filepath.Join(home, ".claude", "skills", skill)
			if err := os.MkdirAll(dest, 0755); err != nil {
				t.Fatal(err)
			}
			permWriteFile(t, filepath.Join(dest, "SKILL.md"), permOldContent)

			var out bytes.Buffer
			if err := Sync(repo, home, true, &out); err != nil {
				t.Fatalf("apply: %v\n%s", err, out.String())
			}
			backups, err := ListBackups(repo, home)
			if err != nil || len(backups) != 1 {
				t.Fatalf("backups: %v %v", backups, err)
			}
			out.Reset()
			if err := Restore(repo, home, backups[0].ID, true, &out); err != nil {
				t.Fatalf("restore: %v\n%s", err, out.String())
			}
			if got := permBackupCount(t, repo, home); got != 2 {
				t.Fatalf("expected the restore to write its own backup, got %d", got)
			}

			// Change only the permission bits of the restored tree. Its
			// content still equals the backup's saved tree, so the next
			// restore must be a no-op.
			permChmodTree(t, dest, 0700, 0600)
			chmodded := permTreeModes(t, dest)

			out.Reset()
			if err := Restore(repo, home, backups[0].ID, true, &out); err != nil {
				t.Fatalf("second restore: %v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), "All skills are up to date.") {
				t.Fatalf("permission-only difference was not a restore no-op:\n%s", out.String())
			}
			if got := permBackupCount(t, repo, home); got != 2 {
				t.Fatalf("permission-only difference created a backup: %d backups", got)
			}
			if got := permTreeModes(t, dest); !reflect.DeepEqual(got, chmodded) {
				t.Fatalf("restore no-op rewrote permission bits: got %v, want %v", got, chmodded)
			}

			// Negative control: a content difference is still a change.
			permWriteFile(t, filepath.Join(dest, "SKILL.md"), permNewContent)
			out.Reset()
			if err := Restore(repo, home, backups[0].ID, true, &out); err != nil {
				t.Fatalf("restore after content change: %v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), "replace + back up claude/"+skill) {
				t.Fatalf("content difference was not reported as a replacement:\n%s", out.String())
			}
			if got := permBackupCount(t, repo, home); got != 3 {
				t.Fatalf("replacement did not create a backup: %d backups", got)
			}
		})
	}
}

// TestInstalledModesUseRequestedModesSubjectToUmask characterizes the modes
// an install actually writes, for the embedded and the on-disk source, under
// two umasks.
//
// The child probe applies into an isolated home whose pre-existing claude
// tree is replaced, so the backup of that tree is observable at the same
// time. The parent's assertions are the oracle; the child only performs the
// mechanical work under the requested umask.
func TestInstalledModesUseRequestedModesSubjectToUmask(t *testing.T) {
	for _, source := range []string{"embedded", "on-disk"} {
		for _, mask := range []string{"022", "077"} {
			t.Run(source+"/umask-"+mask, func(t *testing.T) {
				work := t.TempDir()
				fileMode, dirMode := permOriginalModes(mask)
				skill := permSkillFor(source)
				if mask == "077" && (fileMode&permParseMask(t, mask) == 0 || dirMode&permParseMask(t, mask) == 0) {
					t.Fatal("backup mode fixture must include bits removed by the restrictive umask")
				}
				runPermissionProbe(t, work, permScenarioInstall, source, skill, mask, fileMode, dirMode)

				umask := permParseMask(t, mask)
				wantFile := os.FileMode(0644) &^ umask
				wantNestedDir := os.FileMode(0755) &^ umask
				home := filepath.Join(work, "home")
				repo := permRepoFor(source, work)
				if source == "on-disk" {
					info, err := os.Stat(filepath.Join(repo, "skills-files", skill, "SKILL.md"))
					if err != nil {
						t.Fatal(err)
					}
					if info.Mode().Perm()&^umask == wantFile {
						t.Fatal("source file mode is not discriminating after umask")
					}
				}

				// The installed tree uses the requested modes, masked by the
				// umask.
				dest := filepath.Join(home, ".claude", "skills", skill)
				modes := permTreeModes(t, dest)
				if got := modes["."]; got != os.FileMode(0755) {
					t.Errorf("installed skill directory mode = %04o, want 0755 (explicit chmod, umask-independent)", got)
				}
				for path, mode := range modes {
					if path == "." {
						continue
					}
					if info, err := os.Stat(filepath.Join(dest, filepath.FromSlash(path))); err != nil {
						t.Fatal(err)
					} else if info.IsDir() {
						if mode != wantNestedDir {
							t.Errorf("installed nested directory %s mode = %04o, want %04o (0755 requested, masked by umask %04o)", path, mode, wantNestedDir, umask)
						}
					} else if mode != wantFile {
						t.Errorf("installed file %s mode = %04o, want %04o (0644 requested, masked by umask %04o)", path, mode, wantFile, umask)
					}
				}

				// The backup keeps the replaced tree's mode bits, masked by
				// the same umask.
				backupRoot, _ := backupRootFor(repo, home)
				id := onlyBackup(t, backupRoot)
				saved := filepath.Join(backupRoot, id, "claude", skill)
				savedModes := permTreeModes(t, saved)
				if got := savedModes["."]; got != dirMode&^umask {
					t.Errorf("backed-up skill directory mode = %04o, want %04o (source mode %04o masked by umask %04o)", got, dirMode&^umask, dirMode, umask)
				}
				if got := savedModes["SKILL.md"]; got != fileMode&^umask {
					t.Errorf("backed-up file mode = %04o, want %04o (source mode %04o masked by umask %04o)", got, fileMode&^umask, fileMode, umask)
				}
				if got := read(t, filepath.Join(saved, "SKILL.md")); got != permOldContent {
					t.Errorf("backed-up content = %q, want %q", got, permOldContent)
				}

				// The manifest digest covers paths, kinds and contents, not
				// modes: a mode-only change in the backup leaves the digest
				// valid and the backup usable.
				m, err := readManifest(filepath.Join(backupRoot, id, "manifest.json"))
				if err != nil {
					t.Fatal(err)
				}
				var replaced *entry
				for i := range m.Entries {
					if m.Entries[i].Client == "claude" && m.Entries[i].Skill == skill {
						replaced = &m.Entries[i]
					}
				}
				if replaced == nil {
					t.Fatalf("no manifest entry for the replaced skill: %+v", m.Entries)
				}
				if replaced.Action != actionReplaced || replaced.SHA256 == "" {
					t.Fatalf("manifest entry = %+v, want a replaced entry with a digest", replaced)
				}
				digest, err := treeDigest(os.DirFS(filepath.Join(backupRoot, id)), replaced.Backup)
				if err != nil {
					t.Fatal(err)
				}
				if digest != replaced.SHA256 {
					t.Fatalf("recorded digest %q != recomputed %q", replaced.SHA256, digest)
				}
				changedMode := savedModes["SKILL.md"] ^ 0100
				if err := os.Chmod(filepath.Join(saved, "SKILL.md"), changedMode); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(filepath.Join(saved, "SKILL.md"))
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() == savedModes["SKILL.md"] {
					t.Fatal("mode-only digest edit did not change actual permission bits")
				}
				after, err := treeDigest(os.DirFS(filepath.Join(backupRoot, id)), replaced.Backup)
				if err != nil {
					t.Fatal(err)
				}
				if after != replaced.SHA256 {
					t.Fatalf("digest changed after a mode-only edit: %q != %q", after, replaced.SHA256)
				}
				if _, _, err := loadBackup(backupRoot, id, home); err != nil {
					t.Fatalf("backup with a mode-only edit was rejected: %v", err)
				}
			})
		}
	}
}

// TestRestoreNormalizesModesInsteadOfRestoringBackedUpModes characterizes
// what a restore actually writes: it reinstalls the backed-up content through
// the same staged path as an apply, so it writes the requested modes again.
// The backup preserved the replaced tree's mode bits; the restore does not
// reproduce them.
func TestRestoreNormalizesModesInsteadOfRestoringBackedUpModes(t *testing.T) {
	for _, source := range []string{"embedded", "on-disk"} {
		for _, mask := range []string{"022", "077"} {
			t.Run(source+"/umask-"+mask, func(t *testing.T) {
				work := t.TempDir()
				fileMode, dirMode := permOriginalModes(mask)
				skill := permSkillFor(source)
				runPermissionProbe(t, work, permScenarioRestore, source, skill, mask, fileMode, dirMode)

				umask := permParseMask(t, mask)
				home := filepath.Join(work, "home")
				repo := permRepoFor(source, work)
				backupRoot, _ := backupRootFor(repo, home)

				// The backup kept the replaced tree's mode bits.
				id := permOldestBackup(t, repo, home)
				saved := filepath.Join(backupRoot, id, "claude", skill)
				savedModes := permTreeModes(t, saved)
				if got := savedModes["."]; got != dirMode&^umask {
					t.Errorf("backed-up skill directory mode = %04o, want %04o", got, dirMode&^umask)
				}
				if got := savedModes["SKILL.md"]; got != fileMode&^umask {
					t.Errorf("backed-up file mode = %04o, want %04o", got, fileMode&^umask)
				}

				// The restored tree holds the old content again, with the
				// requested modes rather than the backed-up ones.
				dest := filepath.Join(home, ".claude", "skills", skill)
				restored := permTreeModes(t, dest)
				if got := restored["."]; got != os.FileMode(0755) {
					t.Errorf("restored skill directory mode = %04o, want 0755", got)
				}
				if got := restored["SKILL.md"]; got != os.FileMode(0644)&^umask {
					t.Errorf("restored file mode = %04o, want %04o (0644 requested, masked by umask %04o)", got, os.FileMode(0644)&^umask, umask)
				}
				if got := read(t, filepath.Join(dest, "SKILL.md")); got != permOldContent {
					t.Errorf("restored content = %q, want %q", got, permOldContent)
				}
				if restored["SKILL.md"] == savedModes["SKILL.md"] {
					t.Errorf("restored file mode %04o equals the backed-up mode; the observation is not discriminating", restored["SKILL.md"])
				}
				if restored["."] == savedModes["."] {
					t.Errorf("restored directory mode %04o equals the backed-up mode; the observation is not discriminating", restored["."])
				}
			})
		}
	}
}

// permOldestBackup returns the id of the oldest backup under repo/home's
// backup root, which is the apply's backup (a restore writes a later one).
func permOldestBackup(t *testing.T, repo, home string) string {
	t.Helper()
	backups, err := ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected the apply and the restore to write one backup each, got %d", len(backups))
	}
	return backups[len(backups)-1].ID
}

// runPermissionProbe runs the child probe process under the given umask and
// fails the test if it does not succeed.
func runPermissionProbe(t *testing.T, work, scenario, source, skill, mask string, fileMode, dirMode os.FileMode) {
	t.Helper()
	env := append(os.Environ(),
		permProbeWorkEnv+"="+work,
		permProbeScenarioEnv+"="+scenario,
		permProbeSourceEnv+"="+source,
		permProbeSkillEnv+"="+skill,
		permProbeFileModeEnv+"=0"+strconv.FormatUint(uint64(fileMode.Perm()), 8),
		permProbeDirModeEnv+"=0"+strconv.FormatUint(uint64(dirMode.Perm()), 8),
	)
	// The umask is set by the shell that starts the child, so neither this
	// process nor the child changes a Go process-global umask in place.
	script := `umask "$1"; shift; exec "$@"`
	cmd := exec.Command("sh", "-c", script, "perm-probe", mask, os.Args[0], "-test.run=^TestPermissionProbeChild$", "-test.v")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("permission probe child failed: %v\n%s", err, out)
	}
}

// TestPermissionProbeChild is the child half of the umask-dependent probes.
// It runs only when the parent test starts this binary with the probe
// environment set; a normal test run skips it.
func TestPermissionProbeChild(t *testing.T) {
	work := os.Getenv(permProbeWorkEnv)
	if work == "" {
		t.Skip("probe helper: runs only as a child process of the permission tests")
	}
	scenario := os.Getenv(permProbeScenarioEnv)
	source := os.Getenv(permProbeSourceEnv)
	skill := os.Getenv(permProbeSkillEnv)
	fileMode := permParseModeEnv(t, permProbeFileModeEnv)
	dirMode := permParseModeEnv(t, permProbeDirModeEnv)

	home := filepath.Join(work, "home")
	repo := ""
	if source == "on-disk" {
		repo = filepath.Join(work, "repo")
		permWriteSource(t, repo, skill)
		// Give the source tree modes that differ from the requested install
		// modes, so the installed modes can only come from the installer's
		// requested modes, not from the source.
		src := filepath.Join(repo, "skills-files", skill)
		if err := os.Chmod(src, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(src, "SKILL.md"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(src, "references"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(src, "references", "nested.md"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// A pre-existing destination tree with its own modes and old content.
	dest := filepath.Join(home, ".claude", "skills", skill)
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	permWriteFile(t, filepath.Join(dest, "SKILL.md"), permOldContent)
	if err := os.Chmod(dest, dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dest, "SKILL.md"), fileMode); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Sync(repo, home, true, &out); err != nil {
		t.Fatalf("apply: %v\n%s", err, out.String())
	}
	if scenario != permScenarioRestore {
		return
	}
	backups, err := ListBackups(repo, home)
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups: %v %v", backups, err)
	}
	out.Reset()
	if err := Restore(repo, home, backups[0].ID, true, &out); err != nil {
		t.Fatalf("restore: %v\n%s", err, out.String())
	}
}

func permParseModeEnv(t *testing.T, key string) os.FileMode {
	t.Helper()
	raw := os.Getenv(key)
	value, err := strconv.ParseUint(strings.TrimPrefix(raw, "0"), 8, 32)
	if err != nil {
		t.Fatalf("%s=%q: %v", key, raw, err)
	}
	return os.FileMode(value)
}
