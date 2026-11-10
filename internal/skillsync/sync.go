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

// openCodeAgentsTarget returns the target descriptor for OpenCode agent profiles.
func openCodeAgentsTarget(home string) target {
	return target{name: "opencode-agents", path: filepath.Join(home, ".config", "opencode", "agents")}
}

// change describes one destination skill directory or agent file to install,
// replace or remove.
type change struct {
	target target
	skill  string
	src    string
	source fs.FS
	isFile bool
	old    bool
	remove bool
	// action is the planned structured action for the result API. Direct
	// internal test callers may leave it empty; changeAction derives one.
	action ChangeAction
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

// Options configures optional skill and agent synchronization.
type Options struct {
	WithOpenCode      bool
	OpenCodeFreeFirst bool

	// ClientIDs selects the client skill destinations to plan and install.
	// A nil selection keeps the historical all-clients behavior. A non-nil
	// selection must contain at least one supported ID; it is validated and
	// normalized to canonical client order and copied, and an explicit empty
	// or unknown selection fails before any write. The optional OpenCode pack
	// and profile destinations controlled by WithOpenCode/OpenCodeFreeFirst
	// are independent of this selection.
	ClientIDs []string
}

// Sync previews or installs agentic-sdd skills into user skill directories.
// It is a convenience wrapper for SyncWithOptions with default Options.
func Sync(repo, home string, apply bool, out io.Writer) error {
	return SyncWithOptions(repo, home, apply, Options{}, out)
}

// SyncWithOptions previews or installs skills with the specified options.
// It is a wrapper for SyncWithOptionsResult that keeps the historical error
// and transcript contract and discards the structured result.
func SyncWithOptions(repo, home string, apply bool, opts Options, out io.Writer) error {
	_, err := SyncWithOptionsResult(repo, home, apply, opts, out)
	return err
}

// SyncWithOptionsResult previews or installs skills with the specified
// options and returns the structured outcome together with the historical
// error. The transcript written to out is byte-for-byte the same as
// SyncWithOptions would write.
func SyncWithOptionsResult(repo, home string, apply bool, opts Options, out io.Writer) (OperationResult, error) {
	result := OperationResult{Operation: OperationApply, Outcome: OutcomeFailed}
	if opts.OpenCodeFreeFirst && !opts.WithOpenCode {
		return result, errors.New("--opencode-free-first requires --with-opencode")
	}
	var err error
	if repo != "" {
		repo, err = filepath.Abs(repo)
		if err != nil {
			return result, err
		}
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return result, err
	}
	targets, err := clientTargetsFor(home, opts.ClientIDs)
	if err != nil {
		return result, err
	}
	source, err := sourceFS(repo)
	if err != nil {
		return result, err
	}
	names, err := discoverSkills(source)
	if err != nil {
		return result, err
	}
	coreNames := append([]string(nil), names...)
	changes, planned, err := planChanges(source, home, names, targets, out)
	if err != nil {
		return result, err
	}
	result.Planned = planned

	if opts.WithOpenCode {
		packFS, err := openCodePackFS(repo)
		if err != nil {
			return result, err
		}
		packSkillsFS, err := fs.Sub(packFS, "skills")
		if err != nil {
			return result, err
		}

		for _, t := range targets {
			path := filepath.Join(t.path, "agentic-sdd-orchestrate")
			info, err := os.Lstat(path)
			if err != nil && !os.IsNotExist(err) {
				return result, err
			}
			old := err == nil
			if old && !info.IsDir() {
				return result, fmt.Errorf("refusing non-directory target %s", path)
			}
			if old {
				if err := checkTree(path); err != nil {
					return result, err
				}
				same, err := sameSourceTarget(packSkillsFS, "agentic-sdd-orchestrate", path)
				if err != nil {
					return result, err
				}
				if same {
					fmt.Fprintf(out, "up to date %s/agentic-sdd-orchestrate\n", t.name)
					result.Planned = append(result.Planned, plannedChange(t, "agentic-sdd-orchestrate", false, ChangeUpToDate))
					continue
				}
			}
			action := planAction(old)
			changes = append(changes, change{target: t, skill: "agentic-sdd-orchestrate", src: "agentic-sdd-orchestrate", source: packSkillsFS, old: old, action: action})
			result.Planned = append(result.Planned, plannedChange(t, "agentic-sdd-orchestrate", false, action))
			verb := "install"
			if old {
				verb = "replace + back up"
			}
			fmt.Fprintf(out, "%s %s/agentic-sdd-orchestrate\n", verb, t.name)
		}
		names = append(names, "agentic-sdd-orchestrate")
		sort.Strings(names)

		agentTarget := openCodeAgentsTarget(home)
		if err := checkParents(agentTarget.path, home); err != nil {
			return result, err
		}

		agentEntries, err := fs.ReadDir(packFS, "agents")
		if err != nil {
			return result, err
		}
		for _, entry := range agentEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			filename := entry.Name()
			srcRel := "agents/" + filename
			dstPath := filepath.Join(agentTarget.path, filename)
			info, err := os.Lstat(dstPath)
			if err != nil && !os.IsNotExist(err) {
				return result, err
			}
			old := err == nil
			if old && !info.Mode().IsRegular() {
				return result, fmt.Errorf("refusing special file or symlink %s", dstPath)
			}
			if old {
				same, err := sameSourceTargetFile(packFS, srcRel, dstPath)
				if err != nil {
					return result, err
				}
				if same {
					fmt.Fprintf(out, "up to date opencode-agents/%s\n", filename)
					result.Planned = append(result.Planned, plannedChange(agentTarget, filename, true, ChangeUpToDate))
					continue
				}
			}
			action := planAction(old)
			changes = append(changes, change{target: agentTarget, skill: filename, src: srcRel, source: packFS, isFile: true, old: old, action: action})
			result.Planned = append(result.Planned, plannedChange(agentTarget, filename, true, action))
			verb := "install"
			if old {
				verb = "replace + back up"
			}
			fmt.Fprintf(out, "%s opencode-agents/%s\n", verb, filename)
		}

		if opts.OpenCodeFreeFirst {
			freeEntries, err := fs.ReadDir(packFS, "presets/free-first")
			if err != nil {
				return result, err
			}
			for _, entry := range freeEntries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}
				filename := entry.Name()
				srcRel := "presets/free-first/" + filename
				dstPath := filepath.Join(agentTarget.path, filename)
				info, err := os.Lstat(dstPath)
				if err != nil && !os.IsNotExist(err) {
					return result, err
				}
				old := err == nil
				if old && !info.Mode().IsRegular() {
					return result, fmt.Errorf("refusing special file or symlink %s", dstPath)
				}
				if old {
					same, err := sameSourceTargetFile(packFS, srcRel, dstPath)
					if err != nil {
						return result, err
					}
					if same {
						fmt.Fprintf(out, "up to date opencode-agents/%s\n", filename)
						result.Planned = append(result.Planned, plannedChange(agentTarget, filename, true, ChangeUpToDate))
						continue
					}
				}
				action := planAction(old)
				changes = append(changes, change{target: agentTarget, skill: filename, src: srcRel, source: packFS, isFile: true, old: old, action: action})
				result.Planned = append(result.Planned, plannedChange(agentTarget, filename, true, action))
				verb := "install"
				if old {
					verb = "replace + back up"
				}
				fmt.Fprintf(out, "%s opencode-agents/%s\n", verb, filename)
			}

			if _, err := os.Lstat(filepath.Join(home, ".config", "opencode", "AGENTS.md")); err == nil {
				fmt.Fprintln(out, `Note: ~/.config/opencode/AGENTS.md was not modified; use "agentic-sdd restore <BACKUP_ID>" to revert Free-first agent profiles.`)
			}
		}
	}

	// The digest covers exactly what the plan read from source, once per
	// source tree, and never the destinations the plan compared against.
	digest, err := syncSourceDigest(source, coreNames, repo, opts)
	if err != nil {
		return result, err
	}
	result.SourceDigest = digest

	if !apply {
		fmt.Fprintln(out, "Preview only. Run with apply (or --apply) to install.")
		result.Outcome = OutcomePreview
		return result, nil
	}
	if len(changes) == 0 {
		fmt.Fprintln(out, "All skills are up to date.")
		result.Outcome = OutcomeNoChange
		return result, nil
	}
	backupRoot, sourceLabel := backupRootFor(repo, home)
	written, err := installChanges(backupRoot, home, sourceLabel, source, names, changes, operation{kind: "apply"}, out)
	result.BackupID = written.backupID
	result.BackupPath = written.backupPath
	result.Rollback = written.rollback
	if err != nil {
		return result, err
	}
	result.Outcome = OutcomeApplied
	result.Applied = written.applied
	return result, nil
}

// planAction is the planned action for a destination the source already
// needs to write: replace when it exists, install when it does not.
func planAction(old bool) ChangeAction {
	if old {
		return ChangeReplace
	}
	return ChangeInstall
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

func planChanges(source fs.FS, home string, names []string, targets []target, out io.Writer) ([]change, []ResultChange, error) {
	var changes []change
	var planned []ResultChange
	for _, t := range targets {
		if err := checkParents(t.path, home); err != nil {
			return nil, nil, err
		}
		for _, name := range names {
			path := filepath.Join(t.path, name)
			info, err := os.Lstat(path)
			if err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			old := err == nil
			if old && !info.IsDir() {
				return nil, nil, fmt.Errorf("refusing non-directory target %s", path)
			}
			if old {
				if err := checkTree(path); err != nil {
					return nil, nil, err
				}
				same, err := sameSourceTarget(source, name, path)
				if err != nil {
					return nil, nil, err
				}
				if same {
					fmt.Fprintf(out, "up to date %s/%s\n", t.name, name)
					planned = append(planned, plannedChange(t, name, false, ChangeUpToDate))
					continue
				}
			}
			action := planAction(old)
			changes = append(changes, change{target: t, skill: name, src: name, old: old, action: action})
			planned = append(planned, plannedChange(t, name, false, action))
			verb := "install"
			if old {
				verb = "replace + back up"
			}
			fmt.Fprintf(out, "%s %s/%s\n", verb, t.name, name)
		}
	}
	return changes, planned, nil
}

// writeOutcome carries the structured facts installChanges learned while
// writing. The engine copies them into an OperationResult; a failure keeps
// whatever backup identity and rollback detail exist at that point.
type writeOutcome struct {
	backupID   string
	backupPath string
	applied    []ResultChange
	rollback   RollbackInfo
}

// installChanges backs up every destination it will replace or remove, writes
// a manifest whose Targets stay the full canonical client identity for home,
// then stages and commits the changes with rollback on failure.
func installChanges(backupRoot, home, sourceLabel string, source fs.FS, names []string, changes []change, op operation, out io.Writer) (writeOutcome, error) {
	var written writeOutcome
	now := time.Now().UTC()
	stamp := now.Format("20060102T150405.000000000Z")
	if info, err := os.Lstat(backupRoot); err == nil && !info.IsDir() {
		return written, fmt.Errorf("refusing non-directory backup path %s", backupRoot)
	} else if err != nil && !os.IsNotExist(err) {
		return written, err
	}
	if err := os.MkdirAll(backupRoot, 0700); err != nil {
		return written, err
	}
	backup := filepath.Join(backupRoot, stamp)
	if err := os.Mkdir(backup, 0700); err != nil {
		return written, err
	}
	for _, c := range changes {
		if !c.old {
			continue
		}
		if c.isFile {
			if err := copyFile(filepath.Join(c.target.path, c.skill), filepath.Join(backup, c.target.name, c.skill)); err != nil {
				return written, fmt.Errorf("backup %s/%s: %w", c.target.name, c.skill, err)
			}
		} else {
			if err := copyTree(filepath.Join(c.target.path, c.skill), filepath.Join(backup, c.target.name, c.skill)); err != nil {
				return written, fmt.Errorf("backup %s/%s: %w", c.target.name, c.skill, err)
			}
		}
	}
	hasFileChanges := false
	for _, c := range changes {
		if c.isFile {
			hasFileChanges = true
			break
		}
	}
	entries := make([]entry, 0, len(changes))
	applied := make([]ResultChange, 0, len(changes))
	for _, c := range changes {
		e := entry{
			Client: c.target.name,
			Skill:  c.skill,
			Path:   filepath.Join(c.target.path, c.skill),
		}
		if c.isFile {
			e.Kind = "opencode-agent"
		} else if hasFileChanges {
			e.Kind = "skill"
		}
		if c.old {
			e.Action = actionReplaced
			e.Backup = filepath.Join(c.target.name, c.skill)
			if c.isFile {
				digest, err := fileDigest(os.DirFS(backup), e.Backup)
				if err != nil {
					return written, fmt.Errorf("digest %s: %w", e.Backup, err)
				}
				e.SHA256 = digest
			} else {
				digest, err := treeDigest(os.DirFS(backup), e.Backup)
				if err != nil {
					return written, fmt.Errorf("digest %s: %w", e.Backup, err)
				}
				e.SHA256 = digest
			}
		} else {
			e.Action = actionInstalled
		}
		entries = append(entries, e)
		applied = append(applied, appliedChange(c, e))
	}

	format := 2
	var openCodeTargets []string
	if hasFileChanges {
		format = 3
		openCodeTargets = []string{openCodeAgentsTarget(home).path}
	}

	m := manifest{
		Created: stamp, Source: sourceLabel, Skills: names,
		Format: format, ID: stamp, CreatedAt: now.Format(time.RFC3339Nano),
		Operation: op.kind, RestoredFrom: op.restoredFrom,
		Tool: &toolInfo{
			Version:   version.Version,
			Commit:    version.Commit,
			Date:      version.Date,
			GoVersion: version.GoVersion,
		},
		Home:            home,
		BackupRoot:      backupRoot,
		OpenCodeTargets: openCodeTargets,
		Entries:         entries,
	}
	// Targets always records the complete canonical client identity for home,
	// independent of a narrower client selection: that is the home identity
	// loadBackup validates. Entries describe only what this operation changed.
	for _, t := range clientTargets(home) {
		m.Targets = append(m.Targets, t.path)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return written, err
	}
	if err := os.WriteFile(filepath.Join(backup, "manifest.json"), append(data, '\n'), 0600); err != nil {
		return written, err
	}
	// From here on the backup is complete and restorable; a failure below
	// no longer means "nothing happened".
	written.backupID = stamp
	written.backupPath = backup
	// Build every replacement before touching an installed skill or agent profile.
	for i := range changes {
		c := &changes[i]
		if err := os.MkdirAll(c.target.path, 0755); err != nil {
			cleanupStages(changes)
			return written, err
		}
		if c.remove {
			continue
		}
		srcFS := c.source
		if srcFS == nil {
			srcFS = source
		}
		if c.isFile {
			stageF, err := os.CreateTemp(c.target.path, ".agentic-sdd-stage-*")
			if err != nil {
				cleanupStages(changes)
				return written, err
			}
			c.stage = stageF.Name()
			in, err := srcFS.Open(c.src)
			if err != nil {
				_ = stageF.Close()
				cleanupStages(changes)
				return written, err
			}
			_, copyErr := io.Copy(stageF, in)
			closeInErr := in.Close()
			closeFErr := stageF.Close()
			if copyErr != nil {
				cleanupStages(changes)
				return written, copyErr
			}
			if closeInErr != nil {
				cleanupStages(changes)
				return written, closeInErr
			}
			if closeFErr != nil {
				cleanupStages(changes)
				return written, closeFErr
			}
			if err := os.Chmod(c.stage, installedFileMode); err != nil {
				cleanupStages(changes)
				return written, err
			}
		} else {
			c.stage, err = os.MkdirTemp(c.target.path, ".agentic-sdd-stage-")
			if err != nil {
				cleanupStages(changes)
				return written, err
			}
			if err := copyFromSource(srcFS, c.src, c.stage); err != nil {
				cleanupStages(changes)
				return written, err
			}
			if err := os.Chmod(c.stage, installedDirMode); err != nil {
				cleanupStages(changes)
				return written, err
			}
		}
	}
	var done []int
	fail := func(cause error) (writeOutcome, error) {
		rbErr := rollback(changes, done)
		written.rollback = rollbackInfo(rbErr)
		cleanupStages(changes)
		return written, errors.Join(cause, rbErr)
	}
	for i := range changes {
		c := &changes[i]
		path := filepath.Join(c.target.path, c.skill)
		if c.old {
			if c.isFile {
				undoF, err := os.CreateTemp(c.target.path, ".agentic-sdd-undo-*")
				if err != nil {
					return fail(err)
				}
				c.undo = undoF.Name()
				_ = undoF.Close()
				if err = os.Remove(c.undo); err != nil {
					return fail(err)
				}
				if err = os.Rename(path, c.undo); err != nil {
					return fail(err)
				}
			} else {
				c.undo, err = os.MkdirTemp(c.target.path, ".agentic-sdd-undo-")
				if err != nil {
					return fail(err)
				}
				if err = os.Remove(c.undo); err != nil {
					return fail(err)
				}
				if err = os.Rename(path, c.undo); err != nil {
					return fail(err)
				}
			}
		}
		if !c.remove {
			if err = os.Rename(c.stage, path); err != nil {
				var undoErr error
				if c.old {
					undoErr = os.Rename(c.undo, path)
				}
				rbErr := rollback(changes, done)
				rb := rollbackInfo(rbErr)
				rb.Complete = rb.Complete && undoErr == nil
				rb.Error = errorText(undoErr, rbErr)
				written.rollback = rb
				cleanupStages(changes)
				return written, errors.Join(err, undoErr, rbErr)
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
	written.applied = applied
	verb := "Installed"
	if op.kind == "restore" {
		verb = "Restored"
	}
	fmt.Fprintln(out, verb, len(changes), "skills; backup:", backup)
	return written, nil
}

// rollbackInfo describes the engine's own rollback attempt. A nil error means
// the attempt completed cleanly.
func rollbackInfo(err error) RollbackInfo {
	info := RollbackInfo{Attempted: true, Complete: err == nil}
	if err != nil {
		info.Error = err.Error()
	}
	return info
}

// errorText joins the non-nil errors' messages exactly as errors.Join would
// print them, for RollbackInfo.Error.
func errorText(errs ...error) string {
	var messages []string
	for _, err := range errs {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	return strings.Join(messages, "\n")
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
