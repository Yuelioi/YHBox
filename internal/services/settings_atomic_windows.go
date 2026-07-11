//go:build windows

package services

import "golang.org/x/sys/windows"

func replaceSettingsFile(oldPath, newPath string) error {
	oldPtr, err := windows.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPtr, err := windows.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(
		oldPtr,
		newPtr,
		windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH,
	)
}

// MoveFileEx with MOVEFILE_WRITE_THROUGH does not return until the replace is
// flushed to disk, providing the Windows equivalent of rename + directory fsync.
func syncSettingsDir(string) error { return nil }
