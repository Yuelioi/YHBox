package appbootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspaceRootUsesStableDirectory(t *testing.T) {
	root := t.TempDir()
	got, err := resolveWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, workspaceDirectory); got != want {
		t.Fatalf("resolveWorkspaceRoot = %q, want %q", got, want)
	}
}

func TestResolveWorkspaceRootDoesNotInspectOrMigrateLegacyDirectories(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "workspace-3.1")
	if err := os.Mkdir(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(legacy, "marker")
	if err := os.WriteFile(marker, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, workspaceDirectory) {
		t.Fatalf("resolveWorkspaceRoot = %q", got)
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "kept" {
		t.Fatalf("legacy data changed: %q, %v", raw, err)
	}
}
