package skillsync

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

// This is an explicit distribution contract, not an expectation computed from
// discovery itself. It verifies packaging, not whether an agent obeys prose.
var canonicalSkills = []string{
	"agentic-sdd-architecture-review", "agentic-sdd-bootstrap", "agentic-sdd-debugging",
	"agentic-sdd-drift-retro", "agentic-sdd-implement", "agentic-sdd-plan",
	"agentic-sdd-research-spec", "agentic-sdd-router", "agentic-sdd-spec",
	"agentic-sdd-tasks", "agentic-sdd-verification-review",
}

func TestCanonicalDistributionIncludesDebuggingAndSiblingReferences(t *testing.T) {
	for _, repo := range []string{"", filepath.Join("..", "..")} {
		source, err := sourceFS(repo)
		if err != nil {
			t.Fatal(err)
		}
		names, err := discoverSkills(source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(names, canonicalSkills) {
			t.Fatalf("%q discovery = %v, want eleven canonical skills", repo, names)
		}
		checkPackageLinks(t, source)
	}
}

func checkPackageLinks(t *testing.T, source fs.FS) {
	t.Helper()
	links := regexp.MustCompile(`\]\(([^)]+)\)`)
	err := fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		for _, match := range links.FindAllSubmatch(data, -1) {
			target := string(match[1])
			if bytes.Contains(match[1], []byte("://")) || target[0] == '#' {
				continue
			}
			for i, ch := range target {
				if ch == '#' {
					target = target[:i]
					break
				}
			}
			resolved := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(path), target)))
			if _, err := fs.Stat(source, resolved); err != nil {
				t.Errorf("%s: unresolved packaged link %q: %v", path, target, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Use standard filesystem reads as the copy oracle, not the installer's own
// sameSourceTarget comparator (a broken no-op comparator must not bless itself).
func packageSnapshot(t *testing.T, source fs.FS) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			result[path] = "directory"
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("unexpected packaged special entry: %s", path)
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		result[path] = "file:" + string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestCanonicalInstallPreviewNoopRestoreInIsolatedHome(t *testing.T) {
	for _, mode := range []string{"embedded", "on-disk"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			home, repo := filepath.Join(root, "home"), ""
			source, err := sourceFS("")
			if err != nil {
				t.Fatal(err)
			}
			if mode == "on-disk" {
				source, err = sourceFS(filepath.Join("..", ".."))
				if err != nil {
					t.Fatal(err)
				}
				repo = filepath.Join(root, "repo")
				dir := filepath.Join(repo, "skills-files")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := copyFromSource(source, ".", dir); err != nil {
					t.Fatal(err)
				}
			}
			for _, client := range clientTargets(home) {
				for name, body := range map[string]string{"unrelated": "preserve unrelated", "agentic-sdd-debugging": "previous private diagnosis instructions"} {
					dir := filepath.Join(client.path, name)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}
			var out bytes.Buffer
			if err := Sync(repo, home, false, &out); err != nil {
				t.Fatal(err)
			}
			backups, err := ListBackups(repo, home)
			if err != nil || len(backups) != 0 {
				t.Fatalf("preview created backup: %v %v", backups, err)
			}
			for _, client := range clientTargets(home) {
				if read(t, filepath.Join(client.path, "agentic-sdd-debugging/SKILL.md")) != "previous private diagnosis instructions" {
					t.Fatal("preview replaced old skill")
				}
				if _, err := os.Stat(filepath.Join(client.path, "agentic-sdd-spec")); !os.IsNotExist(err) {
					t.Fatal("preview installed a fresh sibling")
				}
			}
			if err := Sync(repo, home, true, &out); err != nil {
				t.Fatal(err)
			}
			backups, err = ListBackups(repo, home)
			if err != nil || len(backups) != 1 || backups[0].Unusable != "" {
				t.Fatalf("apply backup: %v %v", backups, err)
			}
			originalID := backups[0].ID
			for _, client := range clientTargets(home) {
				for _, name := range canonicalSkills {
					expected, err := fs.Sub(source, name)
					if err != nil {
						t.Fatal(err)
					}
					actual := os.DirFS(filepath.Join(client.path, name))
					if !reflect.DeepEqual(packageSnapshot(t, expected), packageSnapshot(t, actual)) {
						t.Fatalf("%s/%s not distributed byte-for-byte", client.name, name)
					}
				}
				checkPackageLinks(t, os.DirFS(client.path))
			}
			path := filepath.Join(clientTargets(home)[0].path, "agentic-sdd-debugging/SKILL.md")
			before, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := Sync(repo, home, true, &out); err != nil {
				t.Fatal(err)
			}
			after, err := os.Stat(path)
			if err != nil || !before.ModTime().Equal(after.ModTime()) || !os.SameFile(before, after) {
				t.Fatal("no-op rewrote debugging skill")
			}
			backups, err = ListBackups(repo, home)
			if err != nil || len(backups) != 1 {
				t.Fatalf("no-op backup: %v %v", backups, err)
			}
			if err := Restore(repo, home, originalID, false, &out); err != nil {
				t.Fatal(err)
			}
			afterPreview, err := os.Stat(path)
			if err != nil || !os.SameFile(after, afterPreview) || !after.ModTime().Equal(afterPreview.ModTime()) {
				t.Fatal("restore preview rewrote debugging skill")
			}
			backups, err = ListBackups(repo, home)
			if err != nil || len(backups) != 1 {
				t.Fatal("restore preview created backup")
			}
			if err := Restore(repo, home, originalID, true, &out); err != nil {
				t.Fatal(err)
			}
			for _, client := range clientTargets(home) {
				if got := read(t, filepath.Join(client.path, "agentic-sdd-debugging/SKILL.md")); got != "previous private diagnosis instructions" {
					t.Fatal("restore lost old debugging content")
				}
				if got := read(t, filepath.Join(client.path, "unrelated/SKILL.md")); got != "preserve unrelated" {
					t.Fatal("unrelated skill changed")
				}
				if _, err := os.Stat(filepath.Join(client.path, "agentic-sdd-spec")); !os.IsNotExist(err) {
					t.Fatal("restore retained freshly installed sibling")
				}
			}
			if err := Restore(repo, home, originalID, true, &out); err != nil {
				t.Fatal(err)
			}
			backups, err = ListBackups(repo, home)
			if err != nil || len(backups) != 2 {
				t.Fatalf("repeat restore created another backup: %v %v", backups, err)
			}
		})
	}
}
