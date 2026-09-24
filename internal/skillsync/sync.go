package skillsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type target struct {
	name string
	path string
}

type change struct {
	target target
	skill  string
	old    bool
	stage  string
	undo   string
}

type manifest struct {
	Created string   `json:"created"`
	Source  string   `json:"source"`
	Targets []string `json:"targets"`
	Skills  []string `json:"skills"`
}

// Sync previews or installs agentic-sdd skills from repo into user skill directories.
func Sync(repo, home string, apply bool, out io.Writer) error {
	var err error
	repo, err = filepath.Abs(repo)
	if err != nil {
		return err
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return err
	}
	source := filepath.Join(repo, "skills-files")
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "agentic-sdd-") {
			continue
		}
		if !entry.IsDir() {
			return fmt.Errorf("source skill %s is not a directory", name)
		}
		if err := checkTree(filepath.Join(source, name)); err != nil {
			return err
		}
		if info, err := os.Lstat(filepath.Join(source, name, "SKILL.md")); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%s requires a regular SKILL.md", name)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return errors.New("no agentic-sdd skills found")
	}
	sort.Strings(names)
	targets := []target{
		{"agents", filepath.Join(home, ".agents", "skills")},
		{"codex", filepath.Join(home, ".codex", "skills")},
		{"claude", filepath.Join(home, ".claude", "skills")},
		{"agy", filepath.Join(home, ".gemini", "antigravity-cli", "skills")},
	}
	var changes []change
	for _, t := range targets {
		if err := checkParents(t.path, home); err != nil {
			return err
		}
		for _, name := range names {
			path := filepath.Join(t.path, name)
			info, err := os.Lstat(path)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			old := err == nil
			if old && !info.IsDir() {
				return fmt.Errorf("refusing non-directory target %s", path)
			}
			if old {
				if err := checkTree(path); err != nil {
					return err
				}
				same, err := sameTree(filepath.Join(source, name), path)
				if err != nil {
					return err
				}
				if same {
					fmt.Fprintf(out, "up to date %s/%s\n", t.name, name)
					continue
				}
			}
			changes = append(changes, change{target: t, skill: name, old: old})
			verb := "install"
			if old {
				verb = "replace + back up"
			}
			fmt.Fprintf(out, "%s %s/%s\n", verb, t.name, name)
		}
	}
	if !apply {
		fmt.Fprintln(out, "Preview only. Run with --apply to install.")
		return nil
	}
	if len(changes) == 0 {
		fmt.Fprintln(out, "All skills are up to date.")
		return nil
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	backupRoot := filepath.Join(repo, "backups")
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
	m := manifest{Created: stamp, Source: source, Skills: names}
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
	// Build every replacement before touching an installed skill.
	for i := range changes {
		c := &changes[i]
		if err := os.MkdirAll(c.target.path, 0755); err != nil {
			cleanupStages(changes)
			return err
		}
		c.stage, err = os.MkdirTemp(c.target.path, ".agentic-sdd-stage-")
		if err != nil {
			cleanupStages(changes)
			return err
		}
		if err := copyContents(filepath.Join(source, c.skill), c.stage); err != nil {
			cleanupStages(changes)
			return err
		}
		sourceInfo, err := os.Stat(filepath.Join(source, c.skill))
		if err != nil {
			cleanupStages(changes)
			return err
		}
		if err := os.Chmod(c.stage, sourceInfo.Mode().Perm()); err != nil {
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
		if err = os.Rename(c.stage, path); err != nil {
			if c.old {
				err = errors.Join(err, os.Rename(c.undo, path))
			}
			err = errors.Join(err, rollback(changes, done))
			cleanupStages(changes)
			return err
		}
		c.stage = ""
		done = append(done, i)
	}
	for _, i := range done {
		if changes[i].old {
			_ = os.RemoveAll(changes[i].undo)
		}
	}
	fmt.Fprintln(out, "Installed", len(changes), "skills; backup:", backup)
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
