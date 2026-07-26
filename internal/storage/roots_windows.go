//go:build windows

package storage

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func localDataRoot() (string, error) {
	root, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", fmt.Errorf("resolve Windows LocalAppData: %w", err)
	}
	return root, nil
}
