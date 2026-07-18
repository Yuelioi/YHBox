//go:build !windows

package durablefs

import (
	"os"
	"path/filepath"
)

func replaceFile(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func publishNewFile(oldPath, newPath string) error {
	if err := os.Link(oldPath, newPath); err != nil {
		return err
	}
	if err := os.Remove(oldPath); err != nil {
		return &committedError{err: err}
	}
	return nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func removeFile(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return &committedError{err: err}
	}
	return nil
}
