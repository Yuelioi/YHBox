//go:build windows

package container

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func replaceContainerFile(oldPath, newPath string) error {
	oldPtr, err := windows.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPtr, err := windows.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(oldPtr, newPtr,
		windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func syncContainerDir(string) error { return nil }

func removeContainerFileDurable(path string) error {
	tombstone, err := os.CreateTemp(filepath.Dir(path), ".container-rollback-*")
	if err != nil {
		return err
	}
	tombstonePath := tombstone.Name()
	if err := tombstone.Close(); err != nil {
		_ = os.Remove(tombstonePath)
		return err
	}
	if err := replaceContainerFile(path, tombstonePath); err != nil {
		_ = os.Remove(tombstonePath)
		return err
	}
	// The write-through rename durably removes the authoritative name. A stale
	// tombstone is harmless and ignored by the container loader.
	_ = os.Remove(tombstonePath)
	return nil
}
