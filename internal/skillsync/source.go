package skillsync

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	fspath "path"
	"path/filepath"

	skillsfiles "agentic-sdd"
)

// installedDirMode and installedFileMode are applied to everything copied
// from source into a target skill directory, regardless of whether the
// source is the embedded tree (whose files report a fixed, non-writable
// synthetic mode) or an on-disk checkout.
const (
	installedDirMode  = 0755
	installedFileMode = 0644
)

// sourceFS resolves the tree of skill definitions to install, rooted so
// that "agentic-sdd-<name>" entries sit at its top level. An explicit repo
// path (e.g. --repo, a checkout during local development) takes precedence;
// otherwise the skills-files embedded in the binary are used so an installed
// binary (e.g. via Homebrew) works without a repository checkout.
func sourceFS(repo string) (fs.FS, error) {
	if repo == "" {
		return fs.Sub(skillsfiles.Files, "skills-files")
	}
	return fs.Sub(os.DirFS(repo), "skills-files")
}

func checkTreeFS(source fs.FS, root string) error {
	return fs.WalkDir(source, root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file or symlink %s", p)
		}
		return nil
	})
}

// sameSourceTarget reports whether the source subtree at srcPath and the
// on-disk target subtree at dstPath hold the same directory structure and
// file content. Permission bits are not compared: the embedded source
// reports a fixed synthetic mode that carries no meaningful information.
func sameSourceTarget(source fs.FS, srcPath, dstPath string) (bool, error) {
	srcInfo, err := fs.Stat(source, srcPath)
	if err != nil {
		return false, err
	}
	dstInfo, err := os.Lstat(dstPath)
	if err != nil {
		return false, err
	}
	if srcInfo.IsDir() != dstInfo.IsDir() {
		return false, nil
	}
	if srcInfo.IsDir() {
		srcEntries, err := fs.ReadDir(source, srcPath)
		if err != nil {
			return false, err
		}
		dstEntries, err := os.ReadDir(dstPath)
		if err != nil {
			return false, err
		}
		if len(srcEntries) != len(dstEntries) {
			return false, nil
		}
		for i := range srcEntries {
			if srcEntries[i].Name() != dstEntries[i].Name() {
				return false, nil
			}
			same, err := sameSourceTarget(source, fspath.Join(srcPath, srcEntries[i].Name()), filepath.Join(dstPath, dstEntries[i].Name()))
			if err != nil || !same {
				return same, err
			}
		}
		return true, nil
	}
	if srcInfo.Size() != dstInfo.Size() {
		return false, nil
	}
	srcFile, err := source.Open(srcPath)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()
	dstFile, err := os.Open(dstPath)
	if err != nil {
		return false, err
	}
	defer dstFile.Close()
	srcHash, dstHash := sha256.New(), sha256.New()
	if _, err := io.Copy(srcHash, srcFile); err != nil {
		return false, err
	}
	if _, err := io.Copy(dstHash, dstFile); err != nil {
		return false, err
	}
	return bytes.Equal(srcHash.Sum(nil), dstHash.Sum(nil)), nil
}

// copyFromSource copies the subtree at srcPath in source into the existing
// real directory dst, refusing anything that isn't a regular file or
// directory. Every copied file and directory gets installedFileMode /
// installedDirMode, independent of any mode reported by source.
func copyFromSource(source fs.FS, srcPath, dst string) error {
	entries, err := fs.ReadDir(source, srcPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := fspath.Join(srcPath, entry.Name())
		to := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(to, installedDirMode); err != nil {
				return err
			}
			if err := copyFromSource(source, from, to); err != nil {
				return err
			}
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file %s", from)
		}
		in, err := source.Open(from)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, installedFileMode)
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
