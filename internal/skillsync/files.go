package skillsync

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func checkParents(path, home string) error {
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && !info.IsDir() {
			return fmt.Errorf("refusing non-directory path component %s", p)
		}
		if p == home {
			break
		}
	}
	return nil
}

func checkTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file or symlink %s", path)
		}
		return nil
	})
}

func sameTree(left, right string) (bool, error) {
	leftInfo, err := os.Lstat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Lstat(right)
	if err != nil {
		return false, err
	}
	if leftInfo.IsDir() != rightInfo.IsDir() || (!leftInfo.IsDir() && leftInfo.Mode().Perm() != rightInfo.Mode().Perm()) {
		return false, nil
	}
	if leftInfo.IsDir() {
		leftEntries, err := os.ReadDir(left)
		if err != nil {
			return false, err
		}
		rightEntries, err := os.ReadDir(right)
		if err != nil {
			return false, err
		}
		if len(leftEntries) != len(rightEntries) {
			return false, nil
		}
		for i := range leftEntries {
			if leftEntries[i].Name() != rightEntries[i].Name() {
				return false, nil
			}
			same, err := sameTree(filepath.Join(left, leftEntries[i].Name()), filepath.Join(right, rightEntries[i].Name()))
			if err != nil || !same {
				return same, err
			}
		}
		return true, nil
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}
	leftFile, err := os.Open(left)
	if err != nil {
		return false, err
	}
	defer leftFile.Close()
	rightFile, err := os.Open(right)
	if err != nil {
		return false, err
	}
	defer rightFile.Close()
	leftHash, rightHash := sha256.New(), sha256.New()
	if _, err := io.Copy(leftHash, leftFile); err != nil {
		return false, err
	}
	if _, err := io.Copy(rightHash, rightFile); err != nil {
		return false, err
	}
	return bytes.Equal(leftHash.Sum(nil), rightHash.Sum(nil)), nil
}

func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	return copyContents(src, dst)
}

func copyContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from, to := filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := copyTree(from, to); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file %s", from)
		}
		in, err := os.Open(from)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(file, in)
		closeErr := file.Close()
		_ = in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
