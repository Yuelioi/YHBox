package appbootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	workspaceDirectory       = "workspace"
	legacyWorkspaceDirectory = "workspace-3.1"
)

func resolveWorkspaceRoot(dataRoot string) (string, error) {
	current := filepath.Join(dataRoot, workspaceDirectory)
	legacy := filepath.Join(dataRoot, legacyWorkspaceDirectory)
	currentExists, err := directoryExists(current)
	if err != nil {
		return "", fmt.Errorf("inspect workspace root: %w", err)
	}
	legacyExists, err := directoryExists(legacy)
	if err != nil {
		return "", fmt.Errorf("inspect legacy workspace root: %w", err)
	}
	if currentExists && legacyExists {
		return "", fmt.Errorf(
			"workspace migration requires one data root; both %q and %q exist",
			current,
			legacy,
		)
	}
	if currentExists || !legacyExists {
		return current, nil
	}
	if err := os.Rename(legacy, current); err != nil {
		return "", fmt.Errorf("migrate workspace root from %q to %q: %w", legacy, current, err)
	}
	return current, nil
}

func directoryExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%q is not a directory", path)
	}
	return true, nil
}
