package main

// Feature 022 Batch B9 (T19): bounded record ingestion for
// `run record --result FILE`. These tests pin the reader contract before the
// implementation: the size bound must be enforced before allocation, only
// regular files are accepted, unsafe file types are refused with a bounded,
// content-free diagnostic, and bounded stdin keeps working.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"agentic-sdd/internal/execution"
)

// b9WriteSizedFile writes size bytes of payload into path.
func b9WriteSizedFile(t *testing.T, path string, size int, payload byte) {
	t.Helper()
	if err := os.WriteFile(path, bytes.Repeat([]byte{payload}, size), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

// b9AssertUsageError asserts the bounded reader refusal: a usage error that
// names the problem without echoing record content.
func b9AssertUsageError(t *testing.T, err error, marker string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a refusal, got nil error")
	}
	if !errors.Is(err, execution.ErrUsage) {
		t.Fatalf("error %v is not a usage error", err)
	}
	if marker != "" && strings.Contains(err.Error(), marker) {
		t.Fatalf("error echoed record content %q: %v", marker, err)
	}
}

// TestB9ReadExecutionRecordRegularFileBounds covers the limit/limit+1 rule for
// a regular file: the bound is checked from the opened file's own metadata
// before any allocation proportional to the file size.
func TestB9ReadExecutionRecordRegularFileBounds(t *testing.T) {
	dir := t.TempDir()

	atLimit := filepath.Join(dir, "at-limit.json")
	b9WriteSizedFile(t, atLimit, execution.MaxRecordBytes, 'a')
	data, err := readExecutionRecord(atLimit, strings.NewReader(""))
	if err != nil {
		t.Fatalf("record at the %d-byte limit was refused: %v", execution.MaxRecordBytes, err)
	}
	if len(data) != execution.MaxRecordBytes {
		t.Fatalf("read %d bytes, want %d", len(data), execution.MaxRecordBytes)
	}

	overLimit := filepath.Join(dir, "over-limit.json")
	b9WriteSizedFile(t, overLimit, execution.MaxRecordBytes+1, 'b')
	_, err = readExecutionRecord(overLimit, strings.NewReader(""))
	b9AssertUsageError(t, err, "")
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("limit error %v does not name the byte bound", err)
	}
}

// TestB9ReadExecutionRecordRejectsBeforeAllocating proves the bound is applied
// from file metadata, not after a full read: refusing a 32 MiB fixture must not
// allocate 32 MiB.
func TestB9ReadExecutionRecordRejectsBeforeAllocating(t *testing.T) {
	const size = 32 << 20
	path := filepath.Join(t.TempDir(), "large.json")
	b9WriteSizedFile(t, path, size, 'h')

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := readExecutionRecord(path, strings.NewReader(""))
	runtime.ReadMemStats(&after)
	b9AssertUsageError(t, err, "")

	allocated := after.TotalAlloc - before.TotalAlloc
	if allocated > size/8 {
		t.Fatalf("refusing a %d-byte file allocated %d bytes; the bound must be checked before allocation", size, allocated)
	}
}

// TestB9ReadExecutionRecordUnsafeTypes covers directory, device and symlink
// inputs. Each is refused before any content read, so the diagnostic can never
// carry record bytes and a special file can never block the reader.
func TestB9ReadExecutionRecordUnsafeTypes(t *testing.T) {
	dir := t.TempDir()

	t.Run("directory", func(t *testing.T) {
		sub := filepath.Join(dir, "records-dir")
		if err := os.Mkdir(sub, 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := readExecutionRecord(sub, strings.NewReader(""))
		b9AssertUsageError(t, err, "")
	})

	t.Run("device", func(t *testing.T) {
		_, err := readExecutionRecord(os.DevNull, strings.NewReader(""))
		b9AssertUsageError(t, err, "")
		if !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("device refusal %v does not state the regular-file rule", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		target := filepath.Join(dir, "real-record.json")
		b9WriteSizedFile(t, target, 32, 'c')
		link := filepath.Join(dir, "record-link.json")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		_, err := readExecutionRecord(link, strings.NewReader(""))
		b9AssertUsageError(t, err, "")
		if !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlink refusal %v does not name the symlink", err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		_, err := readExecutionRecord(filepath.Join(dir, "absent.json"), strings.NewReader(""))
		b9AssertUsageError(t, err, "")
	})
}

// TestB9ReadExecutionRecordBoundedStdin keeps the pipeline form bounded: the
// limit plus one byte is read, detected and refused.
func TestB9ReadExecutionRecordBoundedStdin(t *testing.T) {
	atLimit := io.LimitReader(bytes.NewReader(bytes.Repeat([]byte{'d'}, execution.MaxRecordBytes)), execution.MaxRecordBytes)
	data, err := readExecutionRecord("-", atLimit)
	if err != nil {
		t.Fatalf("stdin record at the limit was refused: %v", err)
	}
	if len(data) != execution.MaxRecordBytes {
		t.Fatalf("read %d bytes, want %d", len(data), execution.MaxRecordBytes)
	}

	over := bytes.NewReader(bytes.Repeat([]byte{'e'}, execution.MaxRecordBytes+1))
	if _, err := readExecutionRecord("-", over); err == nil {
		t.Fatal("oversize stdin record was accepted")
	} else {
		b9AssertUsageError(t, err, "")
	}

	// An endless stream must not be read past the bound: the reader is allowed
	// to stop after limit+1 bytes rather than draining the source.
	endless := &b9EndlessReader{}
	if _, err := readExecutionRecord("-", endless); err == nil {
		t.Fatal("endless stdin record was accepted")
	}
	if endless.read > int64(execution.MaxRecordBytes)+1 {
		t.Fatalf("stdin reader consumed %d bytes, want at most %d", endless.read, execution.MaxRecordBytes+1)
	}
}

// b9EndlessReader yields payloads forever and records how much was consumed.
type b9EndlessReader struct {
	read int64
}

func (r *b9EndlessReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'f'
	}
	r.read += int64(len(p))
	return len(p), nil
}

// TestB9RunRecordUnsafeResultFiles covers the command contract: an unsafe or
// oversize --result exits 2 with empty stdout, one error line and no record
// artifact, while a bounded regular file still records.
func TestB9RunRecordUnsafeResultFiles(t *testing.T) {
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)
	recordsDir := filepath.Join(identity.RunDir, "records")

	recordCount := func() int {
		t.Helper()
		entries, err := os.ReadDir(recordsDir)
		if err != nil {
			t.Fatalf("read records dir: %v", err)
		}
		return len(entries)
	}
	before := recordCount()

	assertRefused := func(t *testing.T, resultPath string) {
		t.Helper()
		code, stdout, stderr := runExecutionCLI(t, dir, "run", "record", "--run", identity.RunID, "--task", "T01", "--result", resultPath, "--format", "json")
		if code != 2 {
			t.Fatalf("exit %d, want 2; stdout %q stderr %q", code, stdout, stderr)
		}
		if stdout != "" {
			t.Fatalf("stdout = %q, want empty", stdout)
		}
		lines := strings.Split(strings.TrimSpace(stderr), "\n")
		if len(lines) != 1 || !strings.HasPrefix(lines[0], "error:") {
			t.Fatalf("stderr = %q, want a single error line", stderr)
		}
		if got := recordCount(); got != before {
			t.Fatalf("a refused record wrote an artifact: %d -> %d", before, got)
		}
	}

	oversize := filepath.Join(t.TempDir(), "oversize.json")
	b9WriteSizedFile(t, oversize, execution.MaxRecordBytes+1, 'g')
	t.Run("oversize", func(t *testing.T) { assertRefused(t, oversize) })

	sub := filepath.Join(t.TempDir(), "record-dir")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Run("directory", func(t *testing.T) { assertRefused(t, sub) })

	t.Run("device", func(t *testing.T) { assertRefused(t, os.DevNull) })

	t.Run("symlink", func(t *testing.T) {
		// A symlink is refused even when its target is a valid record: the
		// reader never resolves an unverified second path.
		target := filepath.Join(t.TempDir(), "valid.json")
		payload := `{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"fixture"}`
		if err := os.WriteFile(target, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(t.TempDir(), "record-link.json")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		assertRefused(t, link)
	})

	t.Run("regular file control", func(t *testing.T) {
		valid := filepath.Join(t.TempDir(), "valid.json")
		payload := `{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"fixture report"}`
		if err := os.WriteFile(valid, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		code, _, stderr := runExecutionCLI(t, dir, "run", "record", "--run", identity.RunID, "--task", "T01", "--result", valid, "--format", "json")
		if code != 0 {
			t.Fatalf("regular record file exit %d: %s", code, stderr)
		}
	})
}
