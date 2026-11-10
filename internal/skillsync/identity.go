package skillsync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
)

// sourceIdentity builds a canonical record of exactly the source payload one
// engine plan uses and hashes it. Every record is encoded as
//
//	<len(kind)>:<kind> <len(name)>:<name> <len(value)>:<value>\n
//
// so names cannot be confused with separators and the sequence is unambiguous.
// It deliberately contains no destination content: a preview and the apply it
// authorizes must agree even though apply changes destinations.
type sourceIdentity struct {
	h hash.Hash
}

func newSourceIdentity() *sourceIdentity {
	return &sourceIdentity{h: sha256.New()}
}

func (s *sourceIdentity) add(kind, name, value string) {
	fmt.Fprintf(s.h, "%d:%s %d:%s %d:%s\n", len(kind), kind, len(name), name, len(value), value)
}

func (s *sourceIdentity) sum() string {
	return hex.EncodeToString(s.h.Sum(nil))
}

// addSourceTree records every directory and file under root in the stable
// lexical order fs.WalkDir provides. A file that cannot be read, or is not a
// regular file, is recorded by name and size rather than failing the digest:
// planning may legitimately reach that source only when it stages the write,
// and the identity must match between preview and apply instead of reporting
// an earlier, different failure. The write path still refuses it.
func (s *sourceIdentity) addSourceTree(source fs.FS, root, label string) error {
	return fs.WalkDir(source, root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, root+"/")
		if rel == root {
			return nil
		}
		name := label + "/" + rel
		if entry.IsDir() {
			s.add("dir", name, "")
			return nil
		}
		info, statErr := entry.Info()
		size := int64(-1)
		if statErr == nil {
			size = info.Size()
		}
		if statErr == nil && !info.Mode().IsRegular() {
			s.add("special", name, strconv.FormatInt(size, 10))
			return nil
		}
		file, openErr := source.Open(p)
		if openErr != nil {
			s.add("unreadable", name, strconv.FormatInt(size, 10))
			return nil
		}
		digest := sha256.New()
		_, copyErr := io.Copy(digest, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			s.add("unreadable", name, strconv.FormatInt(size, 10))
			return nil
		}
		s.add("file", name, hex.EncodeToString(digest.Sum(nil)))
		return nil
	})
}

// addSourceFile records one regular source file under a canonical label with
// the same tolerance as addSourceTree.
func (s *sourceIdentity) addSourceFile(source fs.FS, rel, label string) {
	info, err := fs.Stat(source, rel)
	if err != nil {
		s.add("unreadable", label, "-1")
		return
	}
	if !info.Mode().IsRegular() {
		s.add("special", label, strconv.FormatInt(info.Size(), 10))
		return
	}
	file, err := source.Open(rel)
	if err != nil {
		s.add("unreadable", label, strconv.FormatInt(info.Size(), 10))
		return
	}
	digest := sha256.New()
	_, copyErr := io.Copy(digest, file)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		s.add("unreadable", label, strconv.FormatInt(info.Size(), 10))
		return
	}
	s.add("file", label, hex.EncodeToString(digest.Sum(nil)))
}

// openCodeProfileRels lists the regular .md profile sources one pack
// directory would offer, in canonical order. It mirrors the plan's own
// directory scan so the identity covers exactly those sources.
func openCodeProfileRels(packFS fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(packFS, dir)
	if err != nil {
		return nil, err
	}
	rels := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		rels = append(rels, dir+"/"+name)
	}
	sort.Strings(rels)
	return rels, nil
}

// syncSourceDigest identifies the source payload a sync plan uses: each
// selected portable skill tree and, when the OpenCode pack is requested, the
// pack skill and profile sources the plan would install. Each source tree is
// hashed once, independent of how many client destinations the selection
// plans for. Destination content and the client selection itself are not part
// of the identity.
func syncSourceDigest(source fs.FS, coreNames []string, repo string, opts Options) (string, error) {
	identity := newSourceIdentity()
	for _, name := range coreNames {
		if err := identity.addSourceTree(source, name, "skill/"+name); err != nil {
			return "", err
		}
	}
	if !opts.WithOpenCode {
		return identity.sum(), nil
	}
	packFS, err := openCodePackFS(repo)
	if err != nil {
		return "", err
	}
	packSkillsFS, err := fs.Sub(packFS, "skills")
	if err != nil {
		return "", err
	}
	if err := identity.addSourceTree(packSkillsFS, "agentic-sdd-orchestrate", "pack-skill/agentic-sdd-orchestrate"); err != nil {
		return "", err
	}
	dirs := []string{"agents"}
	if opts.OpenCodeFreeFirst {
		dirs = append(dirs, "presets/free-first")
	}
	for _, dir := range dirs {
		rels, err := openCodeProfileRels(packFS, dir)
		if err != nil {
			return "", err
		}
		for _, rel := range rels {
			identity.addSourceFile(packFS, rel, "pack-profile/"+rel)
		}
	}
	return identity.sum(), nil
}

// restoreSourceDigest identifies the backup a restore plan uses: the backup
// ID, the validated manifest bytes and the digest of every payload the plan
// would read. It uses the same tree and file digests backup validation
// already relies on; no new manifest format is introduced.
func restoreSourceDigest(backupDir, id string, records []restoreRecord) (string, error) {
	source := os.DirFS(backupDir)
	identity := newSourceIdentity()
	identity.add("backup-id", id, "")
	manifestDigest, err := fileDigest(source, "manifest.json")
	if err != nil {
		return "", err
	}
	identity.add("manifest", "manifest.json", manifestDigest)
	sorted := append([]restoreRecord(nil), records...)
	sortRecords(sorted)
	for _, record := range sorted {
		if record.action != actionReplaced {
			continue
		}
		name := record.client + "/" + record.skill
		if record.isFile {
			digest, err := fileDigest(source, record.backupRel)
			if err != nil {
				return "", err
			}
			identity.add("payload-file", name, digest)
			continue
		}
		digest, err := treeDigest(source, record.backupRel)
		if err != nil {
			return "", err
		}
		identity.add("payload-tree", name, digest)
	}
	return identity.sum(), nil
}
