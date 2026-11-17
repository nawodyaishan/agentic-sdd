package skillsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// B6 (T12) direct coverage of the bounded regular-file reader. Limits are
// passed explicitly so the boundary is exact and cheap to test.

func TestB6BoundedRegularFileLimitIsExact(t *testing.T) {
	dir := t.TempDir()
	const limit = 64
	atLimit := filepath.Join(dir, "at-limit")
	overLimit := filepath.Join(dir, "over-limit")
	if err := os.WriteFile(atLimit, []byte(strings.Repeat("a", limit)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overLimit, []byte(strings.Repeat("a", limit+1)), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readBoundedRegularFile(atLimit, limit)
	if err != nil || len(got) != limit {
		t.Fatalf("read at limit = %d bytes, %v; want %d, nil", len(got), err, limit)
	}
	if _, err := readBoundedRegularFile(overLimit, limit); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("read at limit+1 error = %v, want an exceeds refusal", err)
	}
}

func TestB6BoundedRegularFileRefusesUnsafeObjects(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{
		"symlink":   link,
		"directory": subdir,
		"missing":   filepath.Join(dir, "missing"),
	} {
		if _, err := readBoundedRegularFile(path, 1024); err == nil {
			t.Fatalf("%s accepted by the bounded reader", name)
		}
	}
}
