//go:build !windows

package services

import "os"

func replaceSettingsFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func syncSettingsDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
