//go:build !windows

package main

// Feature 022 Batch B9 (T19): FIFO record inputs. A named pipe must be refused
// from its file type alone, so the reader never opens it and therefore never
// blocks waiting for a writer.

import (
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestB9ReadExecutionRecordRefusesFIFOWithoutBlocking(t *testing.T) {
	dir := t.TempDir()
	fifo := dir + "/record.fifo"
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}

	type outcome struct {
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		_, err := readExecutionRecord(fifo, strings.NewReader(""))
		done <- outcome{err: err}
	}()

	select {
	case got := <-done:
		b9AssertUsageError(t, got.err, "")
		if !strings.Contains(got.err.Error(), "regular file") {
			t.Fatalf("FIFO refusal %v does not state the regular-file rule", got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading a FIFO blocked; the file type must be checked before open")
	}
}
