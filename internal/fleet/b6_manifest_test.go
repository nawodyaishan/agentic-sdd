package fleet

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// B6 (T13) reader and atomic-write tests for fleet manifests (plan §4.2, §4.3).

const b6ManifestLimit = 4 << 20

// b6ManifestBytes returns a valid format-4 sidecar padded with trailing spaces
// to exactly size bytes.
func b6ManifestBytes(t *testing.T, size int) []byte {
	t.Helper()
	body := []byte(`{"format":4,"id":"x","state":"applied","files":[]}`)
	if size < len(body) {
		t.Fatalf("size %d is smaller than the manifest", size)
	}
	return append(body, bytes.Repeat([]byte(" "), size-len(body))...)
}

func b6WriteSidecar(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ManifestFileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestB6ManifestReaderAcceptsExactLimit(t *testing.T) {
	path := b6WriteSidecar(t, b6ManifestBytes(t, b6ManifestLimit))
	m, err := ReadManifestFile(path)
	if err != nil || m == nil {
		t.Fatalf("manifest at exactly the limit: m=%v err=%v", m, err)
	}
}

func TestB6ManifestReaderRejectsLimitPlusOne(t *testing.T) {
	path := b6WriteSidecar(t, b6ManifestBytes(t, b6ManifestLimit+1))
	if _, err := ReadManifestFile(path); !errors.Is(err, ErrUnsafeManifest) {
		t.Fatalf("limit+1 manifest error = %v, want ErrUnsafeManifest", err)
	}
}

func TestB6ManifestReaderRejectsOversizeWithoutContent(t *testing.T) {
	path := b6WriteSidecar(t, bytes.Repeat([]byte("x"), b6ManifestLimit+1024))
	_, err := ReadManifestFile(path)
	if !errors.Is(err, ErrUnsafeManifest) {
		t.Fatalf("oversize error = %v, want ErrUnsafeManifest", err)
	}
	if bytes.Contains([]byte(err.Error()), []byte("xxxx")) {
		t.Fatalf("diagnostic leaked manifest content: %v", err)
	}
}

func TestB6ManifestReaderRejectsDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ManifestFileName)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifestFile(dir); !errors.Is(err, ErrUnsafeManifest) {
		t.Fatalf("directory sidecar error = %v, want ErrUnsafeManifest", err)
	}
}

func TestB6ManifestReaderRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixtures need elevated rights on windows")
	}
	real := b6WriteSidecar(t, b6ManifestBytes(t, 128))
	link := filepath.Join(t.TempDir(), ManifestFileName)
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ReadManifestFile(link); !errors.Is(err, ErrUnsafeManifest) {
		t.Fatalf("symlink sidecar error = %v, want ErrUnsafeManifest", err)
	}
}

func TestB6AtomicWriteReplacesWholeFileAndLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(path, []byte("old sidecar bytes that are longer than the new one\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(path, []byte("new\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := readString(t, path); got != "new\n" {
		t.Fatalf("atomic write left %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("atomic write left temporary files: %v", entries)
	}
}

func TestB6AtomicWriteFailureLeavesNoTempAndKeepsDestination(t *testing.T) {
	dir := t.TempDir()
	// A non-empty directory at the destination makes the final rename fail
	// after the temporary file is complete; the temporary file must go.
	dest := filepath.Join(dir, ManifestFileName)
	if err := os.Mkdir(dest, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "keep"), []byte("kept"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(dest, []byte("new\n"), 0600); err == nil {
		t.Fatal("expected the rename onto a directory to fail")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != ManifestFileName {
		t.Fatalf("failed atomic write left temporary files: %v", entries)
	}
	if got := readString(t, filepath.Join(dest, "keep")); got != "kept" {
		t.Fatalf("destination was modified: %q", got)
	}
}
