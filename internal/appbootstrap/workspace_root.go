package appbootstrap

import "path/filepath"

const workspaceDirectory = "workspace"

func resolveWorkspaceRoot(dataRoot string) (string, error) {
	return filepath.Join(dataRoot, workspaceDirectory), nil
}
