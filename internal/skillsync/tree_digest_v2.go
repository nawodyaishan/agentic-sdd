package skillsync

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// TreeDigestVersion2 is the digest discriminator recorded for a tree hashed by
// TreeDigestV2. New backup entries carry this version; absent or zero keeps the
// legacy TreeDigest semantics for already-written data.
const TreeDigestVersion2 = 2

// treeDigestV2Header is the fixed K2 domain separator. It makes a v2 digest
// unambiguous even if the same bytes are hashed under another scheme.
const treeDigestV2Header = "agentic-sdd/tree/v2\x00"

// Frozen kind bytes. The numeric values are part of the K2 stream and must not
// be reassigned.
const (
	treeDigestV2KindDirectory byte = 0x01
	treeDigestV2KindFile      byte = 0x02
)

// TreeDigestV2 returns the hex-encoded SHA-256 identity of the tree rooted at
// root under the version 2 framing:
//
//	header   "agentic-sdd/tree/v2\x00"
//	entry    kind byte (0x01 directory, 0x02 file)
//	         uint64 big-endian length of the relative path in bytes
//	         relative path bytes, '/' separated, no Unicode normalisation
//	         uint64 big-endian content length (zero for a directory)
//	         content bytes (none for a directory)
//
// The root itself is excluded and an empty directory hashes as the header
// alone. Entries are ordered by the full relative path compared byte-wise, so
// the framing cannot be confused by content that happens to spell a path line
// and cannot depend on per-directory walk order. Permissions and timestamps are
// not part of the identity.
//
// Unlike the legacy treeDigest, v2 walks the real directory with lstat, rejects
// symbolic links and special files, and takes each file's content length from
// fstat of the opened file: it reads one byte past the recorded length and
// fails rather than hashing different bytes. It is an identity for detecting
// changed or truncated copies, not a signature.
func TreeDigestV2(root string) (string, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("tree v2: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("tree v2: refusing symlink root %s", root)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("tree v2: root %s is not a directory", root)
	}

	entries, err := collectTreeDigestV2Entries(root, "", nil)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })

	h := sha256.New()
	if _, err := h.Write([]byte(treeDigestV2Header)); err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.dir {
			if err := writeTreeDigestV2Path(h, treeDigestV2KindDirectory, entry.rel, 0); err != nil {
				return "", err
			}
			continue
		}
		if err := writeTreeDigestV2File(h, entry.rel, entry.full); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// treeDigestV2Entry is one directory or regular file discovered under the
// root. rel is the '/'-separated path used in the stream; full is the path the
// content is read from.
type treeDigestV2Entry struct {
	rel  string
	full string
	dir  bool
}

// collectTreeDigestV2Entries walks dir without following links. Directory
// order is irrelevant because TreeDigestV2 sorts the full set of paths.
func collectTreeDigestV2Entries(dir, prefix string, entries []treeDigestV2Entry) ([]treeDigestV2Entry, error) {
	children, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("tree v2: %w", err)
	}
	for _, child := range children {
		rel := child.Name()
		if prefix != "" {
			rel = prefix + "/" + child.Name()
		}
		full := filepath.Join(dir, child.Name())
		info, err := os.Lstat(full)
		if err != nil {
			return nil, fmt.Errorf("tree v2: %w", err)
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			return nil, fmt.Errorf("tree v2: refusing symlink %s", rel)
		case mode.IsDir():
			entries = append(entries, treeDigestV2Entry{rel: rel, full: full, dir: true})
			entries, err = collectTreeDigestV2Entries(full, rel, entries)
			if err != nil {
				return nil, err
			}
		case mode.IsRegular():
			entries = append(entries, treeDigestV2Entry{rel: rel, full: full})
		default:
			return nil, fmt.Errorf("tree v2: refusing special file %s", rel)
		}
	}
	return entries, nil
}

// writeTreeDigestV2Path writes the kind, path and content-length frame shared
// by directory and file entries.
func writeTreeDigestV2Path(h io.Writer, kind byte, rel string, size uint64) error {
	if _, err := h.Write([]byte{kind}); err != nil {
		return err
	}
	if err := writeTreeDigestV2Uint64(h, uint64(len(rel))); err != nil {
		return err
	}
	if _, err := h.Write([]byte(rel)); err != nil {
		return err
	}
	return writeTreeDigestV2Uint64(h, size)
}

func writeTreeDigestV2Uint64(h io.Writer, v uint64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	_, err := h.Write(buf[:])
	return err
}

// writeTreeDigestV2File frames one regular file using the length reported by
// fstat of the opened descriptor.
func writeTreeDigestV2File(h io.Writer, rel, full string) error {
	f, err := os.Open(full)
	if err != nil {
		return fmt.Errorf("tree v2: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("tree v2: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("tree v2: refusing special file %s", rel)
	}
	size := info.Size()
	if size < 0 {
		return fmt.Errorf("tree v2: negative size for %s", rel)
	}
	if err := writeTreeDigestV2Path(h, treeDigestV2KindFile, rel, uint64(size)); err != nil {
		return err
	}
	return copyTreeDigestV2Content(h, f, uint64(size))
}

// copyTreeDigestV2Content copies exactly size bytes into h and fails if the
// source yields more or fewer, so a file changed during the walk is reported
// instead of being silently hashed at a different length.
func copyTreeDigestV2Content(h io.Writer, r io.Reader, size uint64) error {
	if size > 0 {
		if _, err := io.CopyN(h, r, int64(size)); err != nil {
			return fmt.Errorf("tree v2: content shorter than the recorded %d bytes: %w", size, err)
		}
	}
	var probe [1]byte
	n, err := r.Read(probe[:])
	if n > 0 {
		return fmt.Errorf("tree v2: content longer than the recorded %d bytes", size)
	}
	if err != nil && err != io.EOF {
		return fmt.Errorf("tree v2: %w", err)
	}
	return nil
}
