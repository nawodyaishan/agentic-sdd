package fleet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agentic-sdd/internal/skillsync"
)

const (
	// ManifestFormat is the fleet sidecar format written next to a backup's
	// generic manifest.json. It is separate from, and must not reinterpret,
	// the existing skillsync formats 1-3.
	ManifestFormat = 4

	// ManifestFileName is the sidecar file name inside a backup directory.
	ManifestFileName = "fleet-manifest.json"

	// ScopeGlobal is the only installation scope in v1: the user's global
	// OpenCode agent directory.
	ScopeGlobal = "global"
)

// Per-file provenance. A record with ProvenanceRemoved documents a managed
// removal in that transaction and is not live-managed afterwards. A record
// with ProvenanceRestored means the bytes came from a verified backup in a
// restore transaction: fleet owns them, and their desired digest is the
// restored content itself, so a later plan compares against them exactly like
// any other managed file.
const (
	ProvenanceManaged  = "managed"
	ProvenanceAdopted  = "adopted"
	ProvenanceRestored = "restored"
	ProvenanceLegacy   = "legacy"
	ProvenanceUnknown  = "unknown"
	ProvenanceRemoved  = "removed"
)

// Transaction states recorded in the sidecar. "applying" means the backup is
// complete but the transaction did not report success; it is explicit,
// recoverable state, never a claim that files were installed.
const (
	ManifestStateApplying = "applying"
	ManifestStateApplied  = "applied"
	ManifestStateFailed   = "failed"
)

// FileRecord is the per-file provenance the spec requires: what was wanted,
// what was installed, where the previous content was saved, and who owns it.
type FileRecord struct {
	InstanceID      string `json:"instanceID"`
	SourceRef       string `json:"sourceRef"`
	SourceDigest    string `json:"sourceDigest,omitempty"`
	ConfigDigest    string `json:"configDigest,omitempty"`
	RendererVersion string `json:"rendererVersion,omitempty"`
	Model           string `json:"model,omitempty"`
	DesiredDigest   string `json:"desiredDigest,omitempty"`
	InstalledDigest string `json:"installedDigest,omitempty"`
	BackupRef       string `json:"backupRef,omitempty"`
	BackupDigest    string `json:"backupDigest,omitempty"`
	Scope           string `json:"scope"`
	Target          string `json:"target"`
	Provenance      string `json:"provenance"`
}

// RollbackState records whether the engine's own rollback was attempted and
// whether it completed. It never contains file content.
type RollbackState struct {
	Attempted bool   `json:"attempted"`
	Complete  bool   `json:"complete"`
	Error     string `json:"error,omitempty"`
}

// Manifest is the fleet sidecar (format 4).
type Manifest struct {
	Format          int            `json:"format"`
	ID              string         `json:"id"`
	Operation       string         `json:"operation"`
	RestoredFrom    string         `json:"restoredFrom,omitempty"`
	State           string         `json:"state"`
	CreatedAt       string         `json:"createdAt"`
	Home            string         `json:"home"`
	BackupRoot      string         `json:"backupRoot"`
	BackupPath      string         `json:"backupPath"`
	TargetRoot      string         `json:"targetRoot"`
	Scope           string         `json:"scope"`
	SourceRev       string         `json:"sourceRev,omitempty"`
	ConfigDigest    string         `json:"configDigest,omitempty"`
	RendererVersion string         `json:"rendererVersion,omitempty"`
	Rollback        *RollbackState `json:"rollback,omitempty"`
	Files           []FileRecord   `json:"files"`
}

// ReadManifestFile loads a format-4 fleet sidecar. A missing file reports
// (nil, nil) so callers can distinguish "no fleet provenance" from a
// malformed manifest without treating absence as corruption.
func ReadManifestFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if m.Format != ManifestFormat {
		return nil, fmt.Errorf("unsupported fleet manifest format %d in %s (supported: %d)", m.Format, path, ManifestFormat)
	}
	return &m, nil
}

// ReadLiveManifest loads <home>/.agentic-sdd/fleet/current.json. A missing
// pointer reports (nil, nil): no file has recorded fleet ownership.
// It refuses non-absolute home paths and symlink state directories.
func ReadLiveManifest(home string) (*Manifest, error) {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	path := LivePointerPath(absHome)
	if err := checkNoSymlinkDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	return ReadManifestFile(path)
}

// checkNoSymlinkDir inspects every existing component of path and refuses symlinks.
func checkNoSymlinkDir(path string) error {
	cleaned := filepath.Clean(path)
	for p := cleaned; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil {
			if os.IsNotExist(err) {
				parent := filepath.Dir(p)
				if parent == p {
					return nil
				}
				continue
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink state directory: %s", filepath.Clean(p))
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
	}
	return nil
}

// validateLiveManifest refuses a live pointer that belongs to another home or
// target root, so a copied or stale pointer can never direct writes elsewhere.
func validateLiveManifest(m *Manifest, home, targetRoot string) error {
	if m == nil {
		return nil
	}
	if m.Home != "" && filepath.Clean(m.Home) != filepath.Clean(home) {
		return fmt.Errorf("live fleet manifest belongs to a different home (%s)", m.Home)
	}
	if m.TargetRoot != "" && filepath.Clean(m.TargetRoot) != filepath.Clean(targetRoot) {
		return fmt.Errorf("live fleet manifest target root %s does not match %s", m.TargetRoot, targetRoot)
	}
	return nil
}

// WriteLiveManifest atomically replaces the live pointer with the merged view
// of managed files.
func WriteLiveManifest(home string, m *Manifest) error {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	path := LivePointerPath(absHome)
	dir := filepath.Dir(path)
	if err := checkNoSymlinkDir(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := marshalManifest(m)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".current-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// writeManifestFile writes a sidecar into an existing backup directory.
func writeManifestFile(dir string, m *Manifest) error {
	data, err := marshalManifest(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ManifestFileName), data, 0600)
}

func marshalManifest(m *Manifest) ([]byte, error) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// RebuildLiveManifest reconstructs the live ownership view by replaying the
// fleet sidecar of every completed transaction under the backup root, oldest
// first. It exists so a lost or deleted live pointer cannot silently turn
// managed files into unmanaged collisions. Only applied transactions count; a
// failed or in-flight transaction is never replayed.
func RebuildLiveManifest(home string) (*Manifest, error) {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	targetRoot := TargetRootForHome(absHome)
	backupRoot := skillsync.BackupRootFor("", absHome)
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() || skillsync.ValidateBackupID(e.Name()) != nil {
			continue
		}
		ids = append(ids, e.Name())
	}
	sort.Strings(ids) // timestamp backup names sort oldest first

	var live *Manifest
	for _, id := range ids {
		m, err := ReadManifestFile(filepath.Join(backupRoot, id, ManifestFileName))
		if err != nil {
			return nil, err
		}
		if m == nil || m.State != ManifestStateApplied {
			continue
		}
		if err := m.Validate(id, absHome, backupRoot, targetRoot); err != nil {
			return nil, fmt.Errorf("backup %s fleet manifest: %w", id, err)
		}
		live = MergeLive(live, m)
	}
	return live, nil
}

// verifyBackupCopies confirms that every saved previous file in a completed
// backup really carries the digest the transaction recorded before any
// destination is touched. A changed file between the plan's snapshot and the
// engine's copy therefore aborts the transaction with the backup intact
// rather than committing a mismatched provenance claim.
func verifyBackupCopies(backupDir string, files []FileRecord) error {
	for _, r := range files {
		if r.BackupRef == "" {
			continue
		}
		want := filepath.Join("opencode-agents", filepath.Base(r.Target))
		if filepath.Clean(r.BackupRef) != filepath.Clean(want) {
			return fmt.Errorf("%w: unexpected backup reference %q for %s", ErrUnsafeTarget, r.BackupRef, r.Target)
		}
		path := filepath.Join(backupDir, want)
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("verify backup copy %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: refusing special file or symlink %s", ErrUnsafeTarget, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("verify backup copy %s: %w", path, err)
		}
		if r.BackupDigest == "" || digestBytes(data) != r.BackupDigest {
			return fmt.Errorf("%w: backup copy for %s failed digest verification", ErrUnsafeTarget, r.Target)
		}
	}
	return nil
}

// MergeLive merges one completed transaction into the previous live view.
// Installed and adopted records replace older records for the same target;
// removed records delete them. Every other managed record is preserved, so an
// apply or restore that touches a subset never silently loses ownership.
func MergeLive(previous, txn *Manifest) *Manifest {
	out := *txn
	if out.ConfigDigest == "" && previous != nil {
		out.ConfigDigest = previous.ConfigDigest
	}
	if out.SourceRev == "" && previous != nil {
		out.SourceRev = previous.SourceRev
	}
	if out.RendererVersion == "" && previous != nil {
		out.RendererVersion = previous.RendererVersion
	}
	records := map[string]FileRecord{}
	if previous != nil {
		for _, r := range previous.Files {
			records[r.Target] = r
		}
	}
	for _, r := range txn.Files {
		if r.Provenance == ProvenanceRemoved {
			delete(records, r.Target)
			continue
		}
		records[r.Target] = r
	}
	out.Files = make([]FileRecord, 0, len(records))
	for _, r := range records {
		out.Files = append(out.Files, r)
	}
	sort.Slice(out.Files, func(i, j int) bool { return out.Files[i].Target < out.Files[j].Target })
	return &out
}

// Validate checks the manifest's identity and every target it claims. It
// refuses traversal, symlinked or special targets, duplicate records and a
// manifest for another home, backup root or target root.
func (m *Manifest) Validate(backupID, home, backupRoot, targetRoot string) error {
	if m == nil {
		return fmt.Errorf("fleet manifest is nil")
	}
	if m.Format != ManifestFormat {
		return fmt.Errorf("unsupported fleet manifest format %d (supported: %d)", m.Format, ManifestFormat)
	}
	if m.ID != backupID {
		return fmt.Errorf("fleet manifest id %q does not match backup %s", m.ID, backupID)
	}
	if m.Scope != ScopeGlobal {
		return fmt.Errorf("unsupported fleet scope %q", m.Scope)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	if filepath.Clean(m.Home) != absHome {
		return fmt.Errorf("fleet manifest was made for a different home (%s, current %s)", m.Home, absHome)
	}
	if filepath.Clean(m.TargetRoot) != filepath.Clean(targetRoot) {
		return fmt.Errorf("fleet manifest target root %s does not match current target root %s", m.TargetRoot, targetRoot)
	}
	if filepath.Clean(m.BackupRoot) != filepath.Clean(backupRoot) {
		return fmt.Errorf("fleet manifest backup root %s does not match current backup root %s", m.BackupRoot, backupRoot)
	}
	seen := map[string]bool{}
	for i, r := range m.Files {
		want, err := targetPathFor(targetRoot, filepath.Base(r.Target))
		if err != nil {
			return fmt.Errorf("fleet manifest file %d: %w", i+1, err)
		}
		if filepath.Clean(r.Target) != want {
			return fmt.Errorf("fleet manifest file %d target %q is not directly under %s", i+1, r.Target, targetRoot)
		}
		if seen[r.Target] {
			return fmt.Errorf("fleet manifest has duplicate target %s", r.Target)
		}
		seen[r.Target] = true
		if r.Scope != ScopeGlobal {
			return fmt.Errorf("fleet manifest target %s has unsupported scope %q", r.Target, r.Scope)
		}
		if r.InstanceID == "" || r.SourceRef == "" {
			return fmt.Errorf("fleet manifest target %s is missing instance or source identity", r.Target)
		}
		switch r.Provenance {
		case ProvenanceManaged, ProvenanceAdopted, ProvenanceRestored:
			if r.DesiredDigest == "" {
				return fmt.Errorf("fleet manifest target %s claims managed provenance without a desired digest", r.Target)
			}
		case ProvenanceRemoved, ProvenanceLegacy, ProvenanceUnknown:
		default:
			return fmt.Errorf("fleet manifest target %s has unknown provenance %q", r.Target, r.Provenance)
		}
	}
	return nil
}

// RecordFor returns the live record for a target path.
func (m *Manifest) RecordFor(target string) (FileRecord, bool) {
	if m == nil {
		return FileRecord{}, false
	}
	clean := filepath.Clean(target)
	for _, r := range m.Files {
		if filepath.Clean(r.Target) == clean {
			return r, true
		}
	}
	return FileRecord{}, false
}

// IsManaged reports whether the record represents live fleet ownership.
func (r FileRecord) IsManaged() bool {
	switch r.Provenance {
	case ProvenanceManaged, ProvenanceAdopted, ProvenanceRestored:
		return r.DesiredDigest != ""
	default:
		return false
	}
}

// TargetRootForHome is the one global agent target root in v1.
func TargetRootForHome(home string) string {
	return filepath.Join(home, ".config", "opencode", "agents")
}

// targetPathFor validates a bare agent file name and returns its path under
// root. It refuses traversal, separators, non-markdown names and AGENTS.md;
// no fleet operation may ever write the user's AGENTS.md.
func targetPathFor(root, file string) (string, error) {
	if file == "" || file == "." || file == ".." {
		return "", fmt.Errorf("invalid agent file name %q", file)
	}
	if file != filepath.Base(file) || strings.ContainsAny(file, `/\`) || strings.Contains(file, "..") {
		return "", fmt.Errorf("invalid agent file name %q: path separators and traversal are not allowed", file)
	}
	if !strings.HasSuffix(file, ".md") {
		return "", fmt.Errorf("invalid agent file name %q: want a .md profile", file)
	}
	if strings.EqualFold(file, "AGENTS.md") {
		return "", fmt.Errorf("refusing to manage AGENTS.md")
	}
	return filepath.Join(root, file), nil
}

// onDiskDigest returns the SHA-256 of a regular target file. A missing file is
// ("", false, nil); a symlink or special file is refused, never followed.
func onDiskDigest(path string) (string, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("refusing special file or symlink %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	return digestBytes(data), true, nil
}
