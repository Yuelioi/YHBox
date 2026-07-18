//go:build windows

package durablefs

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func replaceFile(oldPath, newPath string) error {
	oldPtr, err := windows.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPtr, err := windows.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(oldPtr, newPtr, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func publishNewFile(oldPath, newPath string) error {
	oldPtr, err := windows.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPtr, err := windows.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(oldPtr, newPtr, windows.MOVEFILE_WRITE_THROUGH)
}

func syncDir(string) error { return nil }

func removeFile(path string) error {
	tombstone, err := os.CreateTemp(filepath.Dir(path), ".removed-*.tmp")
	if err != nil {
		return err
	}
	tombstonePath := tombstone.Name()
	if err := tombstone.Close(); err != nil {
		_ = os.Remove(tombstonePath)
		return err
	}
	if err := replaceFile(path, tombstonePath); err != nil {
		_ = os.Remove(tombstonePath)
		return err
	}
	_ = os.Remove(tombstonePath)
	return nil
}
