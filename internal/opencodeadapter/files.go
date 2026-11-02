package opencodeadapter

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func checkPath(path string, missing bool) (string, error) {
	if path == "" {
		return "", fmt.Errorf("explicit path required")
	}
	for _, p := range strings.Split(filepath.ToSlash(path), "/") {
		if p == ".." {
			return "", fmt.Errorf("path traversal rejected")
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}
	current := filepath.VolumeName(abs) + string(os.PathSeparator)
	parts := strings.Split(strings.TrimPrefix(abs, current), string(os.PathSeparator))
	for i, p := range parts {
		if p == "" {
			continue
		}
		current = filepath.Join(current, p)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) && missing {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("path unavailable")
		}
		if info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) || (!info.IsDir() && !info.Mode().IsRegular()) || (!info.IsDir() && info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0) {
			return "", fmt.Errorf("symlink or special path rejected")
		}
	}
	return abs, nil
}

// ReadInput refuses symlinks, devices/FIFOs and oversized files before reading.
// Storage must be quiescent/trusted; this is not an adversarial filesystem sandbox.
func ReadInput(path string) ([]byte, error) {
	base := strings.ToLower(filepath.Base(path))
	if base == "auth.json" || base == "credentials.json" || strings.Contains(base, ".db") || strings.HasPrefix(base, ".env") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") {
		return nil, fmt.Errorf("credential-store paths are not accepted")
	}
	abs, err := checkPath(path, false)
	if err != nil {
		return nil, err
	}
	before, err := os.Lstat(abs)
	if err != nil || !before.Mode().IsRegular() || before.Size() > MaxInput {
		return nil, fmt.Errorf("bounded regular input required")
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("input unavailable")
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, fmt.Errorf("input changed")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxInput+1))
	if err != nil || len(data) > MaxInput {
		return nil, fmt.Errorf("input exceeds limit or cannot be read")
	}
	return data, nil
}

func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func activePath(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == ".opencode" || p == ".agents" || p == ".claude" || p == ".codex" || p == ".git" || p == "skills-files" || (p == ".config" && i+1 < len(parts) && parts[i+1] == "opencode") {
			return true
		}
	}
	return false
}

func configState(root string) ([]byte, os.FileInfo, error) {
	dest, err := checkPath(filepath.Join(root, filepath.FromSlash(destination)), true)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Lstat(dest)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("destination unavailable")
	}
	data, err := ReadInput(dest)
	if err != nil {
		return nil, nil, err
	}
	if _, err := parseConfig(data); err != nil {
		return nil, nil, err
	}
	return data, info, nil
}

func projectRoot(root string) (string, error) {
	root, err := checkPath(root, false)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() || root == filepath.VolumeName(root)+string(os.PathSeparator) {
		return "", fmt.Errorf("explicit project directory required")
	}
	return root, nil
}

func Preview(root, candidate string) (PreviewResult, error) {
	root, err := projectRoot(root)
	if err != nil {
		return PreviewResult{}, err
	}
	candidate, err = checkPath(candidate, false)
	if err != nil {
		return PreviewResult{}, err
	}
	if inside(root, candidate) || activePath(candidate) {
		return PreviewResult{}, fmt.Errorf("candidate must be external to project and active discovery")
	}
	data, err := ReadInput(candidate)
	if err != nil {
		return PreviewResult{}, err
	}
	if _, err := validateCandidate(data); err != nil {
		return PreviewResult{}, err
	}
	old, info, err := configState(root)
	if err != nil {
		return PreviewResult{}, err
	}
	original := "absent"
	if info != nil {
		original = digest(old)
	}
	return PreviewResult{Destination: destination, Original: original, Candidate: digest(data), Noop: info != nil && bytes.Equal(data, old)}, nil
}

func privateDirectory(path string) (string, error) {
	path, err := checkPath(path, false)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("existing owner-private directory required")
	}
	return path, nil
}

func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if _, err := checkPath(path, true); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".sdd-config-")
	if err != nil {
		return fmt.Errorf("cannot stage config file")
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(mode.Perm()); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("cannot persist staged file")
	}
	if err = os.Rename(name, path); err != nil {
		return fmt.Errorf("cannot replace config file")
	}
	if err = syncDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("replacement durability uncertain")
	}
	return nil
}

func saveTransaction(dir string, tx Transaction) error {
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0600)
}

func unchanged(root string, old []byte, info os.FileInfo) error {
	current, now, err := configState(root)
	if err != nil {
		return err
	}
	if (info == nil) != (now == nil) || !bytes.Equal(current, old) {
		return fmt.Errorf("destination changed; reconcile instead of overwriting")
	}
	if info != nil && (!os.SameFile(info, now) || info.Mode() != now.Mode()) {
		return fmt.Errorf("destination identity changed")
	}
	return nil
}

func Apply(root, candidate, backupRoot, original, installed string) (Transaction, error) {
	return apply(root, candidate, backupRoot, original, installed, nil)
}

// beforeReplace is an in-package deterministic failure/race test seam, never a
// caller-controlled CLI hook or external executable.
func apply(root, candidate, backupRoot, original, installed string, beforeReplace func() error) (Transaction, error) {
	preview, err := Preview(root, candidate)
	if err != nil {
		return Transaction{}, err
	}
	if original == "" || installed == "" || preview.Original != original || preview.Candidate != installed {
		return Transaction{}, fmt.Errorf("preview identity pins missing or changed")
	}
	root, err = projectRoot(root)
	if err != nil {
		return Transaction{}, err
	}
	candidate, err = checkPath(candidate, false)
	if err != nil {
		return Transaction{}, err
	}
	data, err := ReadInput(candidate)
	if err != nil || digest(data) != installed {
		return Transaction{}, fmt.Errorf("candidate changed after preview")
	}
	old, info, err := configState(root)
	if err != nil {
		return Transaction{}, err
	}
	if (info == nil && original != "absent") || (info != nil && digest(old) != original) {
		return Transaction{}, fmt.Errorf("destination changed after preview")
	}
	tx := Transaction{Version: 1, Kind: "opencode-project-config", Root: root, Original: original, Installed: installed, Phase: "noop", InstalledMode: 0600}
	if info != nil {
		tx.OriginalMode = uint32(info.Mode().Perm())
		tx.InstalledMode = tx.OriginalMode
	}
	if preview.Noop {
		return tx, nil
	}
	backupRoot, err = privateDirectory(backupRoot)
	if err != nil {
		return Transaction{}, err
	}
	if inside(root, backupRoot) || inside(backupRoot, root) || activePath(backupRoot) || inside(backupRoot, candidate) {
		return Transaction{}, fmt.Errorf("backup storage must be separate from project, candidate and discovery")
	}
	backup, err := os.MkdirTemp(backupRoot, "config-")
	if err != nil {
		return Transaction{}, fmt.Errorf("cannot reserve backup")
	}
	tx.Backup = backup
	tx.Phase = "prepared"
	if info != nil {
		f, err := os.OpenFile(filepath.Join(backup, "original"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return tx, fmt.Errorf("cannot reserve original backup")
		}
		_, err = f.Write(old)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return tx, fmt.Errorf("original backup failed")
		}
	}
	if err = saveTransaction(backup, tx); err != nil {
		return tx, err
	}
	if err = syncDir(backupRoot); err != nil {
		return tx, fmt.Errorf("backup durability uncertain")
	}
	if beforeReplace != nil {
		if err = beforeReplace(); err != nil {
			return tx, err
		}
	}
	if err = unchanged(root, old, info); err != nil {
		return tx, err
	}
	dir := filepath.Join(root, ".opencode")
	if err = os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
		return tx, fmt.Errorf("cannot create project config directory")
	}
	if _, err = checkPath(dir, false); err != nil {
		return tx, err
	}
	if err = unchanged(root, old, info); err != nil {
		return tx, err
	}
	if err = atomicWrite(filepath.Join(dir, "opencode.jsonc"), data, os.FileMode(tx.InstalledMode)); err != nil {
		return tx, err
	}
	tx.Phase = "installed"
	if err = saveTransaction(backup, tx); err != nil {
		return tx, fmt.Errorf("config installed; manifest persistence uncertain; preserve backup")
	}
	return tx, nil
}

func validDigest(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == sha256.Size && strings.ToLower(s) == s
}

func Rollback(root, backup string) (Transaction, error) {
	root, err := projectRoot(root)
	if err != nil {
		return Transaction{}, err
	}
	backup, err = privateDirectory(backup)
	if err != nil {
		return Transaction{}, err
	}
	if inside(root, backup) || inside(backup, root) || activePath(backup) {
		return Transaction{}, fmt.Errorf("backup overlaps project/discovery")
	}
	manifest := filepath.Join(backup, "manifest.json")
	mi, err := os.Stat(manifest)
	if err != nil || mi.Mode().Perm()&0077 != 0 {
		return Transaction{}, fmt.Errorf("private config manifest required")
	}
	data, err := ReadInput(manifest)
	if err != nil {
		return Transaction{}, err
	}
	if _, err = parseConfig(data); err != nil {
		return Transaction{}, err
	}
	var tx Transaction
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(&tx); err != nil || tx.Version != 1 || tx.Kind != "opencode-project-config" || tx.Root != root || tx.Backup != backup || !validDigest(tx.Installed) || (tx.Original != "absent" && !validDigest(tx.Original)) || tx.OriginalMode > 0777 || tx.InstalledMode > 0777 || tx.InstalledMode == 0 {
		return Transaction{}, fmt.Errorf("invalid or foreign config manifest")
	}
	if tx.Phase != "installed" && tx.Phase != "prepared" && tx.Phase != "rolled_back" {
		return Transaction{}, fmt.Errorf("unknown transaction phase")
	}
	var old []byte
	if tx.Original != "absent" {
		originalPath := filepath.Join(backup, "original")
		oi, err := os.Stat(originalPath)
		if err != nil || oi.Mode().Perm()&0077 != 0 {
			return tx, fmt.Errorf("private original backup required")
		}
		old, err = ReadInput(originalPath)
		if err != nil || digest(old) != tx.Original {
			return tx, fmt.Errorf("original backup digest mismatch")
		}
		if _, err = parseConfig(old); err != nil {
			return tx, err
		}
	}
	current, info, err := configState(root)
	if err != nil {
		return tx, err
	}
	alreadyOriginal := (tx.Original == "absent" && info == nil) || (info != nil && digest(current) == tx.Original && uint32(info.Mode().Perm()) == tx.OriginalMode)
	if tx.Phase == "rolled_back" {
		if !alreadyOriginal {
			return tx, fmt.Errorf("rolled-back destination was changed")
		}
		return tx, nil
	}
	if alreadyOriginal {
		tx.Phase = "rolled_back"
		return tx, saveTransaction(backup, tx)
	}
	if info == nil || digest(current) != tx.Installed || uint32(info.Mode().Perm()) != tx.InstalledMode {
		return tx, fmt.Errorf("installed config changed; rollback refused")
	}
	if err = unchanged(root, current, info); err != nil {
		return tx, err
	}
	dest := filepath.Join(root, filepath.FromSlash(destination))
	if tx.Original == "absent" {
		if err = os.Remove(dest); err == nil {
			err = syncDir(filepath.Dir(dest))
		}
	} else {
		err = atomicWrite(dest, old, os.FileMode(tx.OriginalMode))
	}
	if err != nil {
		return tx, fmt.Errorf("rollback incomplete; preserve manifest and files")
	}
	tx.Phase = "rolled_back"
	return tx, saveTransaction(backup, tx)
}
