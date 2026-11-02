//go:build linux || darwin

package opencodeadapter

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSpecialFilesAndSymlinkDestinationsAreRejected(t *testing.T) {
	root, candidate, _, _ := fixture(t)
	fifo := filepath.Join(filepath.Dir(root), "input-fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadInput(fifo); err == nil {
		t.Fatal("FIFO read admitted")
	}
	dir := filepath.Join(root, ".opencode")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(candidate, filepath.Join(dir, "opencode.jsonc")); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(root, candidate); err == nil {
		t.Fatal("symlink destination admitted")
	}
}
