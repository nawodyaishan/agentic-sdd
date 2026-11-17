//go:build !windows

package fleet

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// B6 (T13): a FIFO sidecar must be refused without blocking the reader.

func b6Mkfifo(path string) error {
	return syscall.Mkfifo(path, 0600)
}

func TestB6ManifestReaderRejectsFIFOWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), ManifestFileName)
	if err := b6Mkfifo(path); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	type result struct{ err error }
	done := make(chan result, 1)
	go func() {
		_, err := ReadManifestFile(path)
		done <- result{err}
	}()
	select {
	case r := <-done:
		if !errors.Is(r.err, ErrUnsafeManifest) {
			t.Fatalf("FIFO sidecar error = %v, want ErrUnsafeManifest", r.err)
		}
	case <-time.After(5 * time.Second):
		// Unblock the reader's open so the goroutine does not leak, then fail.
		if w, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
			_ = w.Close()
		}
		t.Fatal("reading a FIFO sidecar blocked")
	}
}
