package skillsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agentic-sdd/internal/version"
)

type target struct {
	name string
	path string
}

// change describes one destination skill directory to install, replace or
// remove. src is the corresponding path within the operation's source tree;
// for a plain apply it always equals skill, but a restore's source is
// rooted at the backup directory, where the path is "<client>/<skill>".
// remove is set only by a restore undoing a fresh install: there is no
// source content to stage, so the destination is simply backed up and
// deleted rather than replaced.
type change struct {
	target target
	skill  string
	src    string
	old    bool
	remove bool
	stage  string
	undo   string
}

// operation identifies which command is writing a backup, so installChanges
// can record it in the manifest and pick matching wording. restoredFrom is
// set only when kind is "restore".
type operation struct {
	kind         string
	restoredFrom string
}

// Sync previews or installs agentic-sdd skills into user skill directories.
// When repo is empty, skills are read from the tree embedded in the binary
// (skillsfiles.Files), so an installed binary needs no repository checkout;
// an explicit repo overrides this with an on-disk "<repo>/skills-files"
// checkout, and backups are then kept alongside it at "<repo>/backups" as
// before. With no explicit repo, backups go under "<home>/.agentic-sdd/backups".
func Sync(repo, home string, apply bool, out io.Writer) error {
	var err error
	if repo != "" {
		repo, err = filepath.Abs(repo)
		if err != nil {
			return err
		}
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return err
	}
	source, err := sourceFS(repo)
	if err != nil {
		return err
	}
	names, err := discoverSkills(source)
	if err != nil {
		return err
	}
	targets := clientTargets(home)
	changes, err := planChanges(source, home, names, targets, out)
	if err != nil {
		return err
	}
	if !apply {
		fmt.Fprintln(out, "Preview only. Run with apply (or --apply) to install.")
		return nil
	}
	if len(changes) == 0 {
		fmt.Fprintln(out, "All skills are up to date.")
		return nil
	}
	backupRoot, sourceLabel := backupRootFor(repo, home)
	return installChanges(backupRoot, home, sourceLabel, source, names, targets, changes, operation{kind: "apply"}, out)
}

// clientTargets returns the four client skill directories under home, in
// the fixed order Sync and Restore both plan against and record in a
// manifest's targets/entries.
func clientTargets(home string) []target {
	return []target{
		{"agents", filepath.Join(home, ".agents", "skills")},
		{"codex", filepath.Join(home, ".codex", "skills")},
		{"claude", filepath.Join(home, ".claude", "skills")},
		{"agy", filepath.Join(home, ".gemini", "antigravity-cli", "skills")},
	}
}

// backupRootFor returns the backup root and source label for repo/home,
// matching Sync's existing rule: an explicit repo keeps backups alongside
// it at "<repo>/backups"; otherwise they live under home's own
// ".agentic-sdd/backups" (e.g. when sourcing from the embedded skills).
func backupRootFor(repo, home string) (root, sourceLabel string) {
	if repo != "" {
		return filepath.Join(repo, "backups"), filepath.Join(repo, "skills-files")
	}
	return filepath.Join(home, ".agentic-sdd", "backups"), "embedded"
}

func discoverSkills(source fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "agentic-sdd-") {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf("source skill %s is not a directory", name)
		}
		if err := checkTreeFS(source, name); err != nil {
			return nil, err
		}
		info, err := fs.Stat(source, name+"/SKILL.md")
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s requires a regular SKILL.md", name)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, errors.New("no agentic-sdd skills found")
	}
	sort.Strings(names)
	return names, nil
}

func planChanges(source fs.FS, home string, names []string, targets []target, out io.Writer) ([]change, error) {
	var changes []change
	for _, t := range targets {
		if err := checkParents(t.path, home); err != nil {
			return nil, err
		}
		for _, name := range names {
			path := filepath.Join(t.path, name)
			info, err := os.Lstat(path)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			old := err == nil
			if old && !info.IsDir() {
				return nil, fmt.Errorf("refusing non-directory target %s", path)
			}
			if old {
				if err := checkTree(path); err != nil {
					return nil, err
				}
				same, err := sameSourceTarget(source, name, path)
				if err != nil {
					return nil, err
				}
				if same {
					fmt.Fprintf(out, "up to date %s/%s\n", t.name, name)
					continue
				}
			}
			changes = append(changes, change{target: t, skill: name, src: name, old: old})
			verb := "install"
			if old {
				verb = "replace + back up"
			}
			fmt.Fprintf(out, "%s %s/%s\n", verb, t.name, name)
		}
	}
	return changes, nil
}

func installChanges(backupRoot, home, sourceLabel string, source fs.FS, names []string, targets []target, changes []change, op operation, out io.Writer) error {
	now := time.Now().UTC()
	stamp := now.Format("20060102T150405.000000000Z")
	if info, err := os.Lstat(backupRoot); err == nil && !info.IsDir() {
		return fmt.Errorf("refusing non-directory backup path %s", backupRoot)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(backupRoot, 0700); err != nil {
		return err
	}
	backup := filepath.Join(backupRoot, stamp)
	if err := os.Mkdir(backup, 0700); err != nil {
		return err
	}
	for _, c := range changes {
		if !c.old {
			continue
		}
		if err := copyTree(filepath.Join(c.target.path, c.skill), filepath.Join(backup, c.target.name, c.skill)); err != nil {
			return fmt.Errorf("backup %s/%s: %w", c.target.name, c.skill, err)
		}
	}
	entries := make([]entry, 0, len(changes))
	for _, c := range changes {
		e := entry{
			Client: c.target.name,
			Skill:  c.skill,
			Path:   filepath.Join(c.target.path, c.skill),
		}
		if c.old {
			e.Action = actionReplaced
			e.Backup = filepath.Join(c.target.name, c.skill)
			digest, err := treeDigest(os.DirFS(backup), e.Backup)
			if err != nil {
				return fmt.Errorf("digest %s: %w", e.Backup, err)
			}
			e.SHA256 = digest
		} else {
			e.Action = actionInstalled
		}
		entries = append(entries, e)
	}
	m := manifest{
		Created: stamp, Source: sourceLabel, Skills: names,
		Format: 2, ID: stamp, CreatedAt: now.Format(time.RFC3339Nano),
		Operation: op.kind, RestoredFrom: op.restoredFrom,
		Tool: &toolInfo{
			Version:   version.Version,
			Commit:    version.Commit,
			Date:      version.Date,
			GoVersion: version.GoVersion,
		},
		Home:       home,
		BackupRoot: backupRoot,
		Entries:    entries,
	}
	for _, t := range targets {
		m.Targets = append(m.Targets, t.path)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(backup, "manifest.json"), append(data, '\n'), 0600); err != nil {
		return err
	}
	// Build every replacement before touching an installed skill. A remove
	// change has no source content to stage: it is only backed up above and
	// deleted below.
	for i := range changes {
		c := &changes[i]
		if err := os.MkdirAll(c.target.path, 0755); err != nil {
			cleanupStages(changes)
			return err
		}
		if c.remove {
			continue
		}
		c.stage, err = os.MkdirTemp(c.target.path, ".agentic-sdd-stage-")
		if err != nil {
			cleanupStages(changes)
			return err
		}
		if err := copyFromSource(source, c.src, c.stage); err != nil {
			cleanupStages(changes)
			return err
		}
		if err := os.Chmod(c.stage, installedDirMode); err != nil {
			cleanupStages(changes)
			return err
		}
	}
	var done []int
	for i := range changes {
		c := &changes[i]
		path := filepath.Join(c.target.path, c.skill)
		if c.old {
			c.undo, err = os.MkdirTemp(c.target.path, ".agentic-sdd-undo-")
			if err != nil {
				err = errors.Join(err, rollback(changes, done))
				cleanupStages(changes)
				return err
			}
			if err = os.Remove(c.undo); err != nil {
				err = errors.Join(err, rollback(changes, done))
				cleanupStages(changes)
				return err
			}
			if err = os.Rename(path, c.undo); err != nil {
				err = errors.Join(err, rollback(changes, done))
				cleanupStages(changes)
				return err
			}
		}
		if !c.remove {
			if err = os.Rename(c.stage, path); err != nil {
				if c.old {
					err = errors.Join(err, os.Rename(c.undo, path))
				}
				err = errors.Join(err, rollback(changes, done))
				cleanupStages(changes)
				return err
			}
			c.stage = ""
		}
		done = append(done, i)
	}
	for _, i := range done {
		if changes[i].old {
			_ = os.RemoveAll(changes[i].undo)
		}
	}
	verb := "Installed"
	if op.kind == "restore" {
		verb = "Restored"
	}
	fmt.Fprintln(out, verb, len(changes), "skills; backup:", backup)
	return nil
}

func rollback(changes []change, done []int) error {
	var errs []error
	for i := len(done) - 1; i >= 0; i-- {
		c := changes[done[i]]
		path := filepath.Join(c.target.path, c.skill)
		if err := os.RemoveAll(path); err != nil {
			errs = append(errs, fmt.Errorf("remove installed %s: %w", path, err))
			continue
		}
		if c.old {
			if err := os.Rename(c.undo, path); err != nil {
				errs = append(errs, fmt.Errorf("restore %s: %w", path, err))
			}
		}
	}
	return errors.Join(errs...)
}

func cleanupStages(changes []change) {
	for _, c := range changes {
		if c.stage != "" {
			_ = os.RemoveAll(c.stage)
		}
	}
}
