package skillsync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InventoryStatus describes how one managed target entry compares with the
// selected source. It never claims runtime state such as enabled, running,
// authenticated or verified working.
type InventoryStatus string

const (
	// InventoryStatusMatchesSource means the managed target exists, is safe to
	// inspect, and its content equals the selected source.
	InventoryStatusMatchesSource InventoryStatus = "matches source"
	// InventoryStatusDiffersFromSource means the managed target exists and is
	// safe to inspect, but its content differs from the selected source.
	InventoryStatusDiffersFromSource InventoryStatus = "differs from source"
	// InventoryStatusMissing means the managed target does not exist yet.
	InventoryStatusMissing InventoryStatus = "missing"
	// InventoryStatusUnavailable means the target or one of its parents is a
	// symlink, special file, unsafe path, or otherwise cannot be inspected
	// safely. Such targets are never followed.
	InventoryStatusUnavailable InventoryStatus = "unavailable/refused"
)

const (
	// InventoryKindSkill marks a managed skill directory.
	InventoryKindSkill = "skill"
	// InventoryKindProfile marks a managed OpenCode agent profile file.
	InventoryKindProfile = "profile"

	// InventoryOriginCore marks a skill from the portable core source.
	InventoryOriginCore = "core"
	// InventoryOriginPack marks the optional OpenCode orchestration skill or a
	// base agent profile from the optional OpenCode pack.
	InventoryOriginPack = "pack"
	// InventoryOriginPreset marks a Free-first profile template.
	InventoryOriginPreset = "preset"
)

// ErrPackMissing reports that the optional OpenCode pack source is absent.
// Callers can use errors.Is to distinguish an absent optional pack from an
// invalid, unreadable or refused pack source.
var ErrPackMissing = errors.New("opencode pack source is missing")

const inventoryOrchestrateSkill = "agentic-sdd-orchestrate"

// InventoryEntry is one managed skill directory or profile file for one
// client target. ID has no filename suffix: skill IDs are directory names and
// profile IDs omit ".md".
type InventoryEntry struct {
	ID     string
	Kind   string
	Client string
	Path   string
	Origin string
	Status InventoryStatus
}

// SelectionInventory is the read-only inventory for one source and home
// selection. Home and Repo are absolute (Repo is empty for the embedded
// source); SkillIDs and ProfileIDs are distinct and sorted.
type SelectionInventory struct {
	Home       string
	Repo       string
	BackupRoot string
	Entries    []InventoryEntry
	SkillIDs   []string
	ProfileIDs []string
}

// InspectSelection resolves the selected source, managed IDs and client target
// paths without writing anything. Core inspection never requires the optional
// OpenCode pack. When a pack inspection is requested and the pack source is
// absent, the returned inventory contains the core results and the error
// matches ErrPackMissing; an invalid or unreadable pack source returns a
// different error.
func InspectSelection(repo, home string, opts Options) (SelectionInventory, error) {
	var inv SelectionInventory
	if opts.OpenCodeFreeFirst && !opts.WithOpenCode {
		return inv, errors.New("--opencode-free-first requires --with-opencode")
	}
	var err error
	if repo != "" {
		if repo, err = filepath.Abs(repo); err != nil {
			return inv, err
		}
	}
	if home, err = filepath.Abs(home); err != nil {
		return inv, err
	}
	inv.Home = home
	inv.Repo = repo
	inv.BackupRoot, _ = backupRootFor(repo, home)

	targets, err := clientTargetsFor(home, opts.ClientIDs)
	if err != nil {
		return inv, err
	}
	source, err := sourceFS(repo)
	if err != nil {
		return inv, fmt.Errorf("resolve skills source: %w", err)
	}
	names, err := discoverSkills(source)
	if err != nil {
		return inv, fmt.Errorf("read skills source: %w", err)
	}
	inv.SkillIDs = append([]string(nil), names...)

	parentErrs := make(map[string]error, len(targets))
	for _, tgt := range targets {
		parentErrs[tgt.name] = checkParents(tgt.path, home)
	}
	for _, tgt := range targets {
		for _, id := range names {
			path := filepath.Join(tgt.path, id)
			inv.Entries = append(inv.Entries, InventoryEntry{
				ID:     id,
				Kind:   InventoryKindSkill,
				Client: tgt.name,
				Path:   path,
				Origin: InventoryOriginCore,
				Status: inventoryEntryStatus(parentErrs[tgt.name], source, id, path, false),
			})
		}
	}
	if !opts.WithOpenCode {
		return inv, nil
	}

	packFS, err := inventoryPackFS(repo)
	if err != nil {
		return inv, err
	}
	packSkillsFS, err := fs.Sub(packFS, "skills")
	if err != nil {
		return inv, fmt.Errorf("resolve opencode pack skills source: %w", err)
	}
	if err := checkTreeFS(packSkillsFS, inventoryOrchestrateSkill); err != nil {
		return inv, fmt.Errorf("validate opencode pack skill source: %w", err)
	}
	if err := inventoryCheckTreeReadable(packSkillsFS, inventoryOrchestrateSkill); err != nil {
		return inv, fmt.Errorf("read opencode pack skill source: %w", err)
	}
	orchestrateInfo, err := fs.Stat(packSkillsFS, inventoryOrchestrateSkill+"/SKILL.md")
	if err != nil {
		return inv, fmt.Errorf("validate opencode pack skill source: %w", err)
	}
	if !orchestrateInfo.Mode().IsRegular() {
		return inv, fmt.Errorf("validate opencode pack skill source: %s/SKILL.md is not a regular file", inventoryOrchestrateSkill)
	}

	profiles, err := inventoryProfileSources(packFS, "agents", InventoryOriginPack)
	if err != nil {
		return inv, err
	}
	if opts.OpenCodeFreeFirst {
		presets, err := inventoryProfileSources(packFS, "presets/free-first", InventoryOriginPreset)
		if err != nil {
			return inv, err
		}
		profiles = append(profiles, presets...)
	}

	inv.SkillIDs = inventoryUniqueSorted(append(inv.SkillIDs, inventoryOrchestrateSkill))
	for _, tgt := range targets {
		path := filepath.Join(tgt.path, inventoryOrchestrateSkill)
		inv.Entries = append(inv.Entries, InventoryEntry{
			ID:     inventoryOrchestrateSkill,
			Kind:   InventoryKindSkill,
			Client: tgt.name,
			Path:   path,
			Origin: InventoryOriginPack,
			Status: inventoryEntryStatus(parentErrs[tgt.name], packSkillsFS, inventoryOrchestrateSkill, path, false),
		})
	}

	profileTarget := openCodeAgentsTarget(home)
	profileParentErr := checkParents(profileTarget.path, home)
	profileIDs := make([]string, 0, len(profiles))
	seenProfileIDs := make(map[string]bool, len(profiles))
	for _, profile := range profiles {
		path := filepath.Join(profileTarget.path, profile.id+".md")
		inv.Entries = append(inv.Entries, InventoryEntry{
			ID:     profile.id,
			Kind:   InventoryKindProfile,
			Client: profileTarget.name,
			Path:   path,
			Origin: profile.origin,
			Status: inventoryEntryStatus(profileParentErr, packFS, profile.rel, path, true),
		})
		if !seenProfileIDs[profile.id] {
			seenProfileIDs[profile.id] = true
			profileIDs = append(profileIDs, profile.id)
		}
	}
	sort.Strings(profileIDs)
	inv.ProfileIDs = profileIDs
	return inv, nil
}

// inventoryProfileSource is one managed profile file in the optional pack.
type inventoryProfileSource struct {
	id     string
	rel    string
	origin string
}

// inventoryPackFS resolves the optional pack source and distinguishes an
// absent pack (ErrPackMissing) from an invalid or unreadable one. The final
// on-disk component must be a real directory so an optional-pack symlink is
// refused rather than followed.
func inventoryPackFS(repo string) (fs.FS, error) {
	if repo != "" {
		root := filepath.Join(repo, "opencode-pack")
		info, err := os.Lstat(root)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("%w: %s", ErrPackMissing, root)
			}
			return nil, fmt.Errorf("inspect opencode pack source %s: %w", root, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("opencode pack source %s is not a directory", root)
		}
	}
	packFS, err := openCodePackFS(repo)
	if err != nil {
		return nil, fmt.Errorf("resolve opencode pack source: %w", err)
	}
	if repo == "" {
		info, err := fs.Stat(packFS, ".")
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, ErrPackMissing
			}
			return nil, fmt.Errorf("inspect embedded opencode pack source: %w", err)
		}
		if !info.IsDir() {
			return nil, errors.New("embedded opencode pack source is not a directory")
		}
	}
	return packFS, nil
}

// inventoryProfileSources lists the managed regular .md files in one pack
// directory. Directory entries and non-.md files are not managed profiles;
// symlinks and special files with a .md name are refused as invalid source.
func inventoryProfileSources(packFS fs.FS, dir, origin string) ([]inventoryProfileSource, error) {
	entries, err := fs.ReadDir(packFS, dir)
	if err != nil {
		return nil, fmt.Errorf("read opencode pack profile source %s: %w", dir, err)
	}
	sources := make([]inventoryProfileSource, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect opencode pack profile source %s/%s: %w", dir, name, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("refusing special file or symlink in opencode pack profile source %s/%s", dir, name)
		}
		rel := dir + "/" + name
		file, err := packFS.Open(rel)
		if err != nil {
			return nil, fmt.Errorf("read opencode pack profile source %s: %w", rel, err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close opencode pack profile source %s: %w", rel, err)
		}
		id := strings.TrimSuffix(name, ".md")
		if id == "" {
			return nil, fmt.Errorf("opencode pack profile source %s/%s has an empty ID", dir, name)
		}
		sources = append(sources, inventoryProfileSource{id: id, rel: rel, origin: origin})
	}
	return sources, nil
}

// inventoryCheckTreeReadable opens every regular file in a validated source
// tree so permission failures surface as an invalid source rather than as a
// target that merely looks missing.
func inventoryCheckTreeReadable(source fs.FS, root string) error {
	return fs.WalkDir(source, root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file or symlink %s", p)
		}
		file, err := source.Open(p)
		if err != nil {
			return err
		}
		return file.Close()
	})
}

// inventoryEntryStatus classifies one target without following unsafe links.
// A target parent failure makes every entry under that parent unavailable.
func inventoryEntryStatus(parentErr error, source fs.FS, srcPath, dstPath string, isFile bool) InventoryStatus {
	if parentErr != nil {
		return InventoryStatusUnavailable
	}
	if isFile {
		return inventoryFileStatus(source, srcPath, dstPath)
	}
	return inventoryTreeStatus(source, srcPath, dstPath)
}

// inventoryTreeStatus compares a skill directory after validating that the
// whole existing target tree is made only of regular files and directories.
func inventoryTreeStatus(source fs.FS, srcPath, dstPath string) InventoryStatus {
	info, err := os.Lstat(dstPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return InventoryStatusMissing
		}
		return InventoryStatusUnavailable
	}
	if !info.IsDir() {
		return InventoryStatusUnavailable
	}
	if err := checkTree(dstPath); err != nil {
		return InventoryStatusUnavailable
	}
	same, err := sameSourceTarget(source, srcPath, dstPath)
	if err != nil {
		return InventoryStatusUnavailable
	}
	if same {
		return InventoryStatusMatchesSource
	}
	return InventoryStatusDiffersFromSource
}

// inventoryFileStatus compares a profile file after validating that the
// existing target entry is a regular file, never a symlink or special file.
func inventoryFileStatus(source fs.FS, srcPath, dstPath string) InventoryStatus {
	info, err := os.Lstat(dstPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return InventoryStatusMissing
		}
		return InventoryStatusUnavailable
	}
	if !info.Mode().IsRegular() {
		return InventoryStatusUnavailable
	}
	same, err := sameSourceTargetFile(source, srcPath, dstPath)
	if err != nil {
		return InventoryStatusUnavailable
	}
	if same {
		return InventoryStatusMatchesSource
	}
	return InventoryStatusDiffersFromSource
}

// inventoryUniqueSorted sorts values in place and removes duplicates.
func inventoryUniqueSorted(values []string) []string {
	sort.Strings(values)
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}
