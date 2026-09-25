package skillsync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// backupIDPattern matches exactly the stamp format installChanges writes
// ("20060102T150405.000000000Z"). Because the character class excludes
// path separators and ".", a string that fails this match can never
// traverse out of a backup root; there is no separate traversal check.
var backupIDPattern = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{9}Z$`)

// ValidateBackupID reports whether id has the exact syntax of a backup
// directory name. Callers that need a distinct usage-error exit status for
// a malformed ID (as opposed to a well-formed ID that turns out not to
// exist, or a backup that fails validation) call this first.
func ValidateBackupID(id string) error {
	if !backupIDPattern.MatchString(id) {
		return fmt.Errorf("invalid backup id %q", id)
	}
	return nil
}

// restoreRecord is one skill a backup can restore, normalized from either
// manifest format and cross-checked against what loadBackup actually found
// on disk in the backup.
type restoreRecord struct {
	client    string
	skill     string
	action    string // actionReplaced or actionInstalled
	backupRel string // relative path within the backup dir; set for actionReplaced only
}

// Backup summarizes one entry under a backup root for listing and
// selection. A malformed or otherwise unusable entry is still listed, with
// Unusable set to why, rather than silently dropped.
type Backup struct {
	ID       string
	Legacy   bool
	Manifest manifest
	Unusable string
	Summary  string
}

// ListBackups returns every backup under repo/home's backup root, newest
// first. A missing root is not an error: it reports zero backups.
func ListBackups(repo, home string) ([]Backup, error) {
	home, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	root, _ := backupRootFor(repo, home)
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var backups []Backup
	for _, de := range dirEntries {
		if !backupIDPattern.MatchString(de.Name()) {
			continue
		}
		info, err := os.Lstat(filepath.Join(root, de.Name()))
		if err != nil || !info.IsDir() {
			continue // not a backup this tool produced
		}
		b := Backup{ID: de.Name()}
		m, records, err := loadBackup(root, de.Name(), home)
		if err != nil {
			b.Unusable = err.Error()
		} else {
			b.Manifest = m
			b.Legacy = m.isLegacy()
			b.Summary = summarizeRecords(records)
		}
		backups = append(backups, b)
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].ID > backups[j].ID })
	return backups, nil
}

func summarizeRecords(records []restoreRecord) string {
	var replaced, installed int
	clients := map[string]bool{}
	for _, r := range records {
		clients[r.client] = true
		if r.action == actionReplaced {
			replaced++
		} else {
			installed++
		}
	}
	names := make([]string, 0, len(clients))
	for c := range clients {
		names = append(names, c)
	}
	sort.Strings(names)
	return fmt.Sprintf("%d replaced, %d installed across %s", replaced, installed, strings.Join(names, ", "))
}

// loadBackup validates the backup at <root>/<id> and returns its parsed
// manifest plus the records it can restore. It never modifies anything
// under root. Destinations are computed fresh from home's current client
// targets throughout; the manifest's own recorded paths are read only to
// be checked against those, never to decide where anything is written.
func loadBackup(root, id, home string) (manifest, []restoreRecord, error) {
	if err := ValidateBackupID(id); err != nil {
		return manifest{}, nil, err
	}
	backupDir := filepath.Join(root, id)
	dirInfo, err := os.Lstat(backupDir)
	if err != nil {
		return manifest{}, nil, err
	}
	if !dirInfo.IsDir() {
		return manifest{}, nil, fmt.Errorf("refusing non-directory backup %s", backupDir)
	}

	m, err := readManifest(filepath.Join(backupDir, "manifest.json"))
	if err != nil {
		return manifest{}, nil, err
	}
	if !m.isLegacy() && m.ID != id {
		return manifest{}, nil, fmt.Errorf("manifest id %q does not match backup directory %s", m.ID, id)
	}

	targets := clientTargets(home)
	clientPath := make(map[string]string, len(targets))
	wantTargets := make([]string, len(targets))
	for i, t := range targets {
		clientPath[t.name] = t.path
		wantTargets[i] = t.path
	}
	if !slices.Equal(m.Targets, wantTargets) {
		return manifest{}, nil, fmt.Errorf("backup %s was made for a different home (targets %v, current %v)", id, m.Targets, wantTargets)
	}
	if !m.isLegacy() {
		wantHome, err := filepath.Abs(home)
		if err != nil {
			return manifest{}, nil, err
		}
		if m.Home != wantHome {
			return manifest{}, nil, fmt.Errorf("backup %s was made for a different home (%s, current %s)", id, m.Home, wantHome)
		}
	}

	// Allowlist top-level entries (manifest.json plus known client dirs),
	// then confirm every saved tree really is an agentic-sdd-* directory
	// with a regular SKILL.md, refusing symlinks and special files. Every
	// Lstat below happens before checkTreeFS walks anything, because
	// fs.WalkDir's own root lookup on an os.DirFS follows a symlink.
	topEntries, err := os.ReadDir(backupDir)
	if err != nil {
		return manifest{}, nil, err
	}
	diskTrees := map[string]bool{}
	for _, te := range topEntries {
		if te.Name() == "manifest.json" {
			info, err := os.Lstat(filepath.Join(backupDir, te.Name()))
			if err != nil {
				return manifest{}, nil, err
			}
			if !info.Mode().IsRegular() {
				return manifest{}, nil, fmt.Errorf("refusing special manifest.json in backup %s", id)
			}
			continue
		}
		if _, known := clientPath[te.Name()]; !known {
			return manifest{}, nil, fmt.Errorf("unexpected entry %s in backup %s", te.Name(), id)
		}
		clientDir := filepath.Join(backupDir, te.Name())
		clientInfo, err := os.Lstat(clientDir)
		if err != nil {
			return manifest{}, nil, err
		}
		if !clientInfo.IsDir() {
			return manifest{}, nil, fmt.Errorf("refusing non-directory client entry %s in backup %s", clientDir, id)
		}
		skillEntries, err := os.ReadDir(clientDir)
		if err != nil {
			return manifest{}, nil, err
		}
		for _, se := range skillEntries {
			if !strings.HasPrefix(se.Name(), "agentic-sdd-") {
				return manifest{}, nil, fmt.Errorf("refusing non-skill entry %s/%s in backup %s", te.Name(), se.Name(), id)
			}
			skillDir := filepath.Join(clientDir, se.Name())
			skillInfo, err := os.Lstat(skillDir)
			if err != nil {
				return manifest{}, nil, err
			}
			if !skillInfo.IsDir() {
				return manifest{}, nil, fmt.Errorf("refusing non-directory skill entry %s in backup %s", skillDir, id)
			}
			rel := filepath.Join(te.Name(), se.Name())
			if err := checkTreeFS(os.DirFS(backupDir), rel); err != nil {
				return manifest{}, nil, err
			}
			skillMD, err := os.Lstat(filepath.Join(skillDir, "SKILL.md"))
			if err != nil || !skillMD.Mode().IsRegular() {
				return manifest{}, nil, fmt.Errorf("%s requires a regular SKILL.md in backup %s", rel, id)
			}
			diskTrees[rel] = true
		}
	}

	if m.isLegacy() {
		records := make([]restoreRecord, 0, len(diskTrees))
		for rel := range diskTrees {
			client, skill := filepath.Split(rel)
			records = append(records, restoreRecord{client: strings.TrimSuffix(client, string(filepath.Separator)), skill: skill, action: actionReplaced, backupRel: rel})
		}
		sortRecords(records)
		return m, records, nil
	}

	records := make([]restoreRecord, 0, len(m.Entries))
	for _, e := range m.Entries {
		path, known := clientPath[e.Client]
		if !known {
			return manifest{}, nil, fmt.Errorf("backup %s entry references unknown client %s", id, e.Client)
		}
		if !strings.HasPrefix(e.Skill, "agentic-sdd-") {
			return manifest{}, nil, fmt.Errorf("backup %s entry has invalid skill name %s", id, e.Skill)
		}
		wantPath := filepath.Join(path, e.Skill)
		if e.Path != wantPath {
			return manifest{}, nil, fmt.Errorf("backup %s entry %s/%s path %q does not match current home (want %q)", id, e.Client, e.Skill, e.Path, wantPath)
		}
		rel := filepath.Join(e.Client, e.Skill)
		switch e.Action {
		case actionReplaced:
			if !diskTrees[rel] {
				return manifest{}, nil, fmt.Errorf("backup %s entry %s has no saved tree", id, rel)
			}
			delete(diskTrees, rel)
			digest, err := treeDigest(os.DirFS(backupDir), rel)
			if err != nil {
				return manifest{}, nil, err
			}
			if e.SHA256 != "" && digest != e.SHA256 {
				return manifest{}, nil, fmt.Errorf("backup %s entry %s failed digest verification", id, rel)
			}
			records = append(records, restoreRecord{client: e.Client, skill: e.Skill, action: actionReplaced, backupRel: rel})
		case actionInstalled:
			if diskTrees[rel] {
				return manifest{}, nil, fmt.Errorf("backup %s entry %s unexpectedly has a saved tree", id, rel)
			}
			records = append(records, restoreRecord{client: e.Client, skill: e.Skill, action: actionInstalled})
		default:
			return manifest{}, nil, fmt.Errorf("backup %s entry %s has unknown action %q", id, rel, e.Action)
		}
	}
	for rel := range diskTrees {
		return manifest{}, nil, fmt.Errorf("backup %s has saved tree %s with no manifest entry", id, rel)
	}
	return m, records, nil
}

func sortRecords(records []restoreRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].client != records[j].client {
			return records[i].client < records[j].client
		}
		return records[i].skill < records[j].skill
	})
}

// Restore previews or applies the restoration of the backup identified by
// id, found under repo/home's backup root. Preview (apply == false) prints
// the plan and writes nothing. Apply backs up every current skill it will
// change into a new backup (recording operation "restore" and this id as
// restored_from) before installing or removing anything, through the same
// staged, rollback-guarded path Sync uses.
func Restore(repo, home, id string, apply bool, out io.Writer) error {
	home, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	root, _ := backupRootFor(repo, home)
	m, records, err := loadBackup(root, id, home)
	if err != nil {
		return err
	}
	backupDir := filepath.Join(root, id)
	targets := clientTargets(home)
	targetByName := make(map[string]target, len(targets))
	for _, t := range targets {
		targetByName[t.name] = t
	}

	if m.isLegacy() {
		fmt.Fprintln(out, "Legacy backup: only the saved skills are recorded; any skill installed fresh by that apply is left as is.")
	}

	source := os.DirFS(backupDir)
	var changes []change
	checked := map[string]bool{}
	for _, r := range records {
		t, ok := targetByName[r.client]
		if !ok {
			return fmt.Errorf("backup %s references unknown client %s", id, r.client)
		}
		if !checked[t.name] {
			if err := checkParents(t.path, home); err != nil {
				return err
			}
			checked[t.name] = true
		}
		path := filepath.Join(t.path, r.skill)
		info, statErr := os.Lstat(path)
		if statErr != nil && !os.IsNotExist(statErr) {
			return statErr
		}
		exists := statErr == nil
		if exists && !info.IsDir() {
			return fmt.Errorf("refusing non-directory target %s", path)
		}
		if exists {
			if err := checkTree(path); err != nil {
				return err
			}
		}
		switch r.action {
		case actionReplaced:
			if exists {
				same, err := sameSourceTarget(source, r.backupRel, path)
				if err != nil {
					return err
				}
				if same {
					fmt.Fprintf(out, "up to date %s/%s\n", r.client, r.skill)
					continue
				}
			}
			verb := "restore"
			if exists {
				verb = "replace + back up"
			}
			fmt.Fprintf(out, "%s %s/%s\n", verb, r.client, r.skill)
			changes = append(changes, change{target: t, skill: r.skill, src: r.backupRel, old: exists})
		case actionInstalled:
			if !exists {
				fmt.Fprintf(out, "up to date %s/%s\n", r.client, r.skill)
				continue
			}
			fmt.Fprintf(out, "remove + back up %s/%s\n", r.client, r.skill)
			changes = append(changes, change{target: t, skill: r.skill, old: true, remove: true})
		}
	}

	if !apply {
		fmt.Fprintln(out, "Preview only. Run with --apply (or restore apply) to restore.")
		return nil
	}
	if len(changes) == 0 {
		fmt.Fprintln(out, "All skills are up to date.")
		return nil
	}
	skillSet := map[string]bool{}
	for _, c := range changes {
		skillSet[c.skill] = true
	}
	names := make([]string, 0, len(skillSet))
	for s := range skillSet {
		names = append(names, s)
	}
	sort.Strings(names)
	return installChanges(root, home, backupDir, source, names, targets, changes, operation{kind: "restore", restoredFrom: id}, out)
}
