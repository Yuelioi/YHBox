package appbootstrap

import (
	"os"
	"path/filepath"
	"strings"
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

func TestResolveWorkspaceRootMigratesLegacyDirectory(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, legacyWorkspaceDirectory)
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
	if _, err := os.Stat(filepath.Join(got, "marker")); err != nil {
		t.Fatalf("migrated marker: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy directory still exists: %v", err)
	}
}

func TestResolveWorkspaceRootRejectsAmbiguousRoots(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{workspaceDirectory, legacyWorkspaceDirectory} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	_, err := resolveWorkspaceRoot(root)
	if err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("resolveWorkspaceRoot error = %v", err)
	}
}

func TestResolveWorkspaceRootRejectsFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, workspaceDirectory), []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resolveWorkspaceRoot(root)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("resolveWorkspaceRoot error = %v", err)
	}
}
