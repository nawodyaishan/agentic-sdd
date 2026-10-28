package harnesscheck

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && relative(filepath.ToSlash(rel))
}

func canonicalDirectory(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	return abs, nil
}

func checkDirectory(workspace, cwd string) (string, error) {
	dir, err := canonicalDirectory(filepath.Join(workspace, filepath.FromSlash(cwd)))
	if err != nil {
		return "", err
	}
	if !within(workspace, dir) {
		return "", fmt.Errorf("cwd escapes workspace")
	}
	return dir, nil
}

// prepareOutput only accepts a new or empty private directory. Evidence is
// outside the checkout. Existing symlinks in the supplied output path are refused.
// Like the command itself, this requires no concurrent untrusted filesystem writer.
func prepareOutput(workspace, output string) (string, error) {
	if output == "" {
		return "", fmt.Errorf("explicit external --output required")
	}
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	if within(workspace, abs) || within(abs, workspace) {
		return "", fmt.Errorf("output and workspace must not overlap")
	}
	var missing []string
	for p := abs; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if os.IsNotExist(err) {
			missing = append(missing, p)
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("unsafe output component: %s", p)
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0700); err != nil {
			return "", err
		}
	}
	canonical, err := canonicalDirectory(abs)
	if err != nil {
		return "", err
	}
	if within(workspace, canonical) || within(canonical, workspace) {
		return "", fmt.Errorf("output overlaps workspace")
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if info.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("existing output directory must be owner-only")
	}
	entries, err := os.ReadDir(canonical)
	if err != nil {
		return "", err
	}
	if len(entries) != 0 {
		return "", fmt.Errorf("output directory must be empty; refusing to overwrite evidence")
	}
	return canonical, nil
}

const truncationMarker = "\n[TRUNCATED: log byte limit reached]\n"

type boundedLog struct {
	mu        sync.Mutex
	closed    bool
	file      *os.File
	remaining int64
	truncated bool
	err       error
}

func (b *boundedLog) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	if b.closed {
		return n, nil
	}
	if int64(len(p)) > b.remaining {
		p = p[:int(b.remaining)]
		b.truncated = true
	}
	if len(p) > 0 {
		written, err := b.file.Write(p)
		b.remaining -= int64(written)
		if err != nil {
			b.err = err
			return written, err
		}
	}
	return n, nil // Drain discarded data so a verbose child cannot deadlock.
}

func (b *boundedLog) close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	first := b.err
	if b.truncated {
		if _, err := io.WriteString(b.file, truncationMarker); first == nil {
			first = err
		}
	}
	if err := b.file.Sync(); first == nil {
		first = err
	}
	if err := b.file.Close(); first == nil {
		first = err
	}
	return first
}

type moduleFile struct {
	exists bool
	data   []byte
}

func modules(dir string) (map[string]moduleFile, error) {
	result := map[string]moduleFile{}
	for _, name := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			result[name] = moduleFile{}
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("module file is not regular: %s", name)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		result[name] = moduleFile{true, data}
	}
	return result, nil
}

func sameModules(a, b map[string]moduleFile) bool {
	for name, file := range a {
		if file.exists != b[name].exists || !bytes.Equal(file.data, b[name].data) {
			return false
		}
	}
	return true
}

func verifyEvidence(path string, original os.FileInfo) error {
	current, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("evidence missing: %w", err)
	}
	if !current.Mode().IsRegular() || !os.SameFile(original, current) || current.Mode().Perm()&0077 != 0 || current.Mode().Perm()&0600 != 0600 {
		return fmt.Errorf("evidence replaced or no longer private: %s", path)
	}
	return nil
}
