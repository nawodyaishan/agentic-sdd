package skillsync

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// treeV2TestHeader is the frozen K2 domain header, written literally so a
// change to the production constant cannot silently redefine the stream.
const treeV2TestHeader = "agentic-sdd/tree/v2\x00"

// treeV2TestStream builds the expected v2 byte stream independently of the
// implementation: header, then one framed entry per call.
type treeV2TestStream struct{ bytes.Buffer }

func (s *treeV2TestStream) entry(kind byte, rel, content string) {
	s.WriteByte(kind)
	s.writeUint64(uint64(len(rel)))
	s.WriteString(rel)
	s.writeUint64(uint64(len(content)))
	if kind == 0x02 {
		s.WriteString(content)
	}
}

func (s *treeV2TestStream) writeUint64(v uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	s.Write(b[:])
}

func (s *treeV2TestStream) digest() string {
	sum := sha256.Sum256(append([]byte(treeV2TestHeader), s.Bytes()...))
	return hex.EncodeToString(sum[:])
}

func treeV2WriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func treeV2Digest(t *testing.T, root string) string {
	t.Helper()
	got, err := TreeDigestV2(root)
	if err != nil {
		t.Fatalf("TreeDigestV2(%s): %v", root, err)
	}
	return got
}

func TestTreeDigestV2EmptyDirectoryIsHeaderOnly(t *testing.T) {
	root := t.TempDir()
	sum := sha256.Sum256([]byte(treeV2TestHeader))
	want := hex.EncodeToString(sum[:])
	if got := treeV2Digest(t, root); got != want {
		t.Fatalf("empty directory digest = %s, want header-only %s", got, want)
	}
}

func TestTreeDigestV2ExactFraming(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	treeV2WriteFile(t, filepath.Join(root, "d", "f"), "hi")

	want := &treeV2TestStream{}
	want.entry(0x01, "d", "")
	want.entry(0x02, "d/f", "hi")
	if got := treeV2Digest(t, root); got != want.digest() {
		t.Fatalf("digest = %s, want framed stream %s", got, want.digest())
	}
}

func TestTreeDigestV2NestedEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "d", "e"), 0o755); err != nil {
		t.Fatal(err)
	}

	want := &treeV2TestStream{}
	want.entry(0x01, "d", "")
	want.entry(0x01, "d/e", "")
	if got := treeV2Digest(t, root); got != want.digest() {
		t.Fatalf("digest = %s, want %s", got, want.digest())
	}
}

// TestTreeDigestV2SortsOverFullPath pins byte-wise ordering of the complete
// relative path: '.' (0x2e) sorts before '/' (0x2f), so "a.b" precedes "a/b"
// and a per-directory walk order would produce a different stream.
func TestTreeDigestV2SortsOverFullPath(t *testing.T) {
	root := t.TempDir()
	treeV2WriteFile(t, filepath.Join(root, "a", "b"), "")
	treeV2WriteFile(t, filepath.Join(root, "a.b"), "")

	want := &treeV2TestStream{}
	want.entry(0x01, "a", "")
	want.entry(0x02, "a.b", "")
	want.entry(0x02, "a/b", "")
	got := treeV2Digest(t, root)
	if got != want.digest() {
		t.Fatalf("digest = %s, want full-path order %s", got, want.digest())
	}

	walkOrder := &treeV2TestStream{}
	walkOrder.entry(0x01, "a", "")
	walkOrder.entry(0x02, "a/b", "")
	walkOrder.entry(0x02, "a.b", "")
	if got == walkOrder.digest() {
		t.Fatal("digest matches per-directory walk order, not full-path byte order")
	}
}

// TestTreeDigestV2SeparatesLegacyCollision is the ambiguity case the legacy
// stream cannot distinguish: a single file "a" holding "f b\nchanged" frames
// exactly like an empty "a" plus "b" holding "changed".
func TestTreeDigestV2SeparatesLegacyCollision(t *testing.T) {
	root := t.TempDir()
	first, second := filepath.Join(root, "first"), filepath.Join(root, "second")
	treeV2WriteFile(t, filepath.Join(first, "a"), "f b\nchanged")
	treeV2WriteFile(t, filepath.Join(second, "a"), "")
	treeV2WriteFile(t, filepath.Join(second, "b"), "changed")

	v1First, err := TreeDigest(os.DirFS(first), ".")
	if err != nil {
		t.Fatalf("legacy TreeDigest(first): %v", err)
	}
	v1Second, err := TreeDigest(os.DirFS(second), ".")
	if err != nil {
		t.Fatalf("legacy TreeDigest(second): %v", err)
	}
	if v1First != v1Second {
		t.Fatalf("legacy framing no longer collides: %s != %s", v1First, v1Second)
	}

	if treeV2Digest(t, first) == treeV2Digest(t, second) {
		t.Fatal("v2 digest collides for the ambiguous legacy pair")
	}
}

func TestTreeDigestV2RejectsSymlinkEntry(t *testing.T) {
	root := t.TempDir()
	treeV2WriteFile(t, filepath.Join(root, "target"), "x")
	if err := os.Symlink("target", filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := TreeDigestV2(root)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("err = %v, want symlink refusal", err)
	}
}

func TestTreeDigestV2RejectsSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := TreeDigestV2(link)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("err = %v, want symlink root refusal", err)
	}
}

func TestTreeDigestV2RejectsFIFO(t *testing.T) {
	root := t.TempDir()
	fifo := filepath.Join(root, "fifo")
	if out, err := exec.Command("mkfifo", fifo).CombinedOutput(); err != nil {
		t.Skipf("mkfifo unavailable (%v: %s)", err, strings.TrimSpace(string(out)))
	}

	type result struct {
		digest string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		digest, err := TreeDigestV2(root)
		done <- result{digest: digest, err: err}
	}()
	select {
	case got := <-done:
		if got.err == nil || !strings.Contains(got.err.Error(), "special file") {
			t.Fatalf("err = %v, want special-file refusal", got.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("TreeDigestV2 blocked on a FIFO entry")
	}
}

func TestTreeDigestV2RejectsNonDirectoryRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	treeV2WriteFile(t, file, "x")

	if _, err := TreeDigestV2(file); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("err = %v, want not-a-directory refusal", err)
	}
	if _, err := TreeDigestV2(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing root accepted")
	}
}

func TestTreeDigestV2RejectsContentLengthMismatch(t *testing.T) {
	var out bytes.Buffer
	if err := copyTreeDigestV2Content(&out, strings.NewReader("hello"), 5); err != nil {
		t.Fatalf("exact-size content: %v", err)
	}
	if out.String() != "hello" {
		t.Fatalf("content = %q, want %q", out.String(), "hello")
	}

	out.Reset()
	if err := copyTreeDigestV2Content(&out, strings.NewReader("hello"), 3); err == nil {
		t.Fatal("content longer than the recorded size accepted")
	}

	out.Reset()
	if err := copyTreeDigestV2Content(&out, strings.NewReader("hello"), 8); err == nil {
		t.Fatal("content shorter than the recorded size accepted")
	}
}

func TestTreeDigestV2IgnoresModeAndTime(t *testing.T) {
	root := t.TempDir()
	treeV2WriteFile(t, filepath.Join(root, "f"), "same bytes")
	before := treeV2Digest(t, root)

	if err := os.Chmod(filepath.Join(root, "f"), 0o600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1234567890, 0)
	if err := os.Chtimes(filepath.Join(root, "f"), stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if after := treeV2Digest(t, root); after != before {
		t.Fatalf("digest changed with permissions/timestamps: %s != %s", after, before)
	}
}

func TestTreeDigestV2IsRootPathIndependent(t *testing.T) {
	root := t.TempDir()
	first, second := filepath.Join(root, "first"), filepath.Join(root, "second")
	treeV2WriteFile(t, filepath.Join(first, "d", "f"), "x")
	treeV2WriteFile(t, filepath.Join(second, "d", "f"), "x")

	if treeV2Digest(t, first) != treeV2Digest(t, second) {
		t.Fatal("digest depends on the absolute root path")
	}
}
