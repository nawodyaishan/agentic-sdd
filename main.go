package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
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

func main() {
	var repo, home string
	var apply bool
	flag.StringVar(&repo, "repo", ".", "repository containing skills-files")
	flag.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	flag.BoolVar(&apply, "apply", false, "back up and install skills; default is preview")
	flag.Parse()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			fatal(err)
		}
	}
	if err := syncSkills(repo, home, apply, os.Stdout); err != nil {
		fatal(err)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }

func syncSkills(repo, home string, apply bool, out io.Writer) error {
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
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	backup := filepath.Join(repo, "backups", stamp)
	if err := os.MkdirAll(backup, 0700); err != nil {
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
	}
	var done []int
	for i := range changes {
		c := &changes[i]
		path := filepath.Join(c.target.path, c.skill)
		if c.old {
			c.undo, err = os.MkdirTemp(c.target.path, ".agentic-sdd-undo-")
			if err != nil {
				rollback(changes, done)
				cleanupStages(changes)
				return err
			}
			if err = os.Remove(c.undo); err != nil {
				rollback(changes, done)
				cleanupStages(changes)
				return err
			}
			if err = os.Rename(path, c.undo); err != nil {
				rollback(changes, done)
				cleanupStages(changes)
				return err
			}
		}
		if err = os.Rename(c.stage, path); err != nil {
			if c.old {
				_ = os.Rename(c.undo, path)
			}
			rollback(changes, done)
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

func rollback(changes []change, done []int) {
	for i := len(done) - 1; i >= 0; i-- {
		c := changes[done[i]]
		path := filepath.Join(c.target.path, c.skill)
		_ = os.RemoveAll(path)
		if c.old {
			_ = os.Rename(c.undo, path)
		}
	}
}

func cleanupStages(changes []change) {
	for _, c := range changes {
		if c.stage != "" {
			_ = os.RemoveAll(c.stage)
		}
	}
}

func checkParents(path, home string) error {
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && !info.IsDir() {
			return fmt.Errorf("refusing non-directory path component %s", p)
		}
		if p == home {
			break
		}
	}
	return nil
}

func checkTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file or symlink %s", path)
		}
		return nil
	})
}

func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	return copyContents(src, dst)
}

func copyContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from, to := filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := copyTree(from, to); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file %s", from)
		}
		in, err := os.Open(from)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(file, in)
		closeErr := file.Close()
		_ = in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
