package skillsync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// manifest is the on-disk record written to <backup>/manifest.json by every
// apply or restore that changes anything. The format 1 fields (Created,
// Source, Targets, Skills) are unchanged from the original backup format and
// keep their JSON names and meaning, so an older reader (or a human) can
// still make sense of them. Format 2 adds the remaining fields additively;
// a format 1 manifest simply omits them (they are all `omitempty`).
type manifest struct {
	// Format 1 fields.
	Created string   `json:"created"`
	Source  string   `json:"source"`
	Targets []string `json:"targets"`
	Skills  []string `json:"skills"`

	// Format 2 fields.
	Format       int       `json:"format,omitempty"`
	ID           string    `json:"id,omitempty"`
	CreatedAt    string    `json:"created_at,omitempty"`
	Operation    string    `json:"operation,omitempty"`
	RestoredFrom string    `json:"restored_from,omitempty"`
	Tool         *toolInfo `json:"tool,omitempty"`
	Home         string    `json:"home,omitempty"`
	BackupRoot   string    `json:"backup_root,omitempty"`
	Entries      []entry   `json:"entries,omitempty"`
}

// toolInfo records the agentic-sdd build that wrote a format 2 manifest.
type toolInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
}

// entry records what happened to one destination skill directory during
// the operation that wrote this manifest. Action is "replaced" when the
// skill existed before the operation (its previous tree is saved under
// Backup, with SHA256 over that saved tree), or "installed" when it did
// not (nothing to save).
type entry struct {
	Client string `json:"client"`
	Skill  string `json:"skill"`
	Path   string `json:"path"`
	Action string `json:"action"`
	Backup string `json:"backup,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

const (
	actionReplaced  = "replaced"
	actionInstalled = "installed"
)

// isLegacy reports whether m was written before format 2 (no per-entry
// records, no recorded location or tool version).
func (m manifest) isLegacy() bool {
	return m.Format < 2
}

// readManifest loads and parses the manifest.json at path. It returns
// whatever format the file was written in: a legacy caller distinguishes
// the two with isLegacy.
func readManifest(path string) (manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return manifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// treeDigest returns a hex-encoded SHA-256 digest over the subtree at root
// in source: its relative paths, entry kinds (file or directory) and file
// contents, walked in the stable lexical order fs.WalkDir already provides.
// It detects a changed or truncated backup copy; it is not a security
// signature.
func treeDigest(source fs.FS, root string) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(source, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, root+"/")
		if rel == root {
			return nil
		}
		if d.IsDir() {
			fmt.Fprintf(h, "d %s\n", rel)
			return nil
		}
		fmt.Fprintf(h, "f %s\n", rel)
		f, openErr := source.Open(p)
		if openErr != nil {
			return openErr
		}
		defer f.Close()
		_, copyErr := io.Copy(h, f)
		return copyErr
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
