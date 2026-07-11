//go:build !windows

package container

import (
	"os"
	"path/filepath"
)

func replaceContainerFile(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func syncContainerDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func removeContainerFileDurable(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncContainerDir(filepath.Dir(path))
}
