package skillsync

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// maxManifestFileBytes is the per-file limit for backup manifests read by this
// package (plan K2 §4.2). Oversize input is refused, never truncated. Legacy
// manifests larger than this are refused by documented compatibility policy.
const maxManifestFileBytes int64 = 4 << 20

// ErrUnsafeInput reports a file that is not a regular file, changed while it
// was opened, or exceeds its byte limit. Messages name the path only; they
// never include file content.
var ErrUnsafeInput = errors.New("unsafe or oversized input file")

// readBoundedRegularFile reads path, which must be a regular file no larger than
// limit bytes. A symbolic link, FIFO, device or directory is refused before the
// object can block or allocate. At most limit+1 bytes are read, so an oversize
// file is detected without buffering it whole.
func readBoundedRegularFile(path string, limit int64) ([]byte, error) {
	pre, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !pre.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file", ErrUnsafeInput, path)
	}
	if pre.Size() > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrUnsafeInput, path, limit)
	}
	f, err := openBoundedNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	post, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !post.Mode().IsRegular() || !os.SameFile(pre, post) {
		return nil, fmt.Errorf("%w: %s changed or is not a regular file while it was opened", ErrUnsafeInput, path)
	}
	if post.Size() > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrUnsafeInput, path, limit)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrUnsafeInput, path, limit)
	}
	return data, nil
}
