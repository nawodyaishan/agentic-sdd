//go:build linux || darwin

package skillsync

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestB6FIFOManifestRejectedWithoutBlocking: a FIFO named manifest.json is
// refused. Reading it must not block waiting for a writer.
func TestB6FIFOManifestRejectedWithoutBlocking(t *testing.T) {
	root, id, home := b6BareBackupRoot(t)
	path := filepath.Join(root, id, "manifest.json")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, _, err := loadBackup(root, id, home)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("loadBackup accepted a FIFO manifest")
		}
	case <-time.After(10 * time.Second):
		// Release the blocked reader so the test process does not leak it.
		if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
			_ = f.Close()
		}
		t.Fatal("reading a FIFO manifest blocked")
	}
}
