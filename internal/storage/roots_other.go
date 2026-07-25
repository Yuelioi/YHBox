//go:build !windows

package storage

import (
	"fmt"
	"os"
)

// Non-Windows GUI support remains preview-grade. UserConfigDir gives the
// platform application-support location while preserving an explicit override.
func localDataRoot() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user application data: %w", err)
	}
	return root, nil
}
