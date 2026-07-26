//go:build windows

package storage

import (
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestResolveDefaultsToWindowsLocalAppData(t *testing.T) {
	t.Setenv(EnvironmentRoot, "")
	roots, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		t.Fatalf("KnownFolderPath: %v", err)
	}
	want := filepath.Join(local, vendorDirectory, productDirectory)
	if !strings.EqualFold(roots.Root, want) {
		t.Fatalf("root = %q, want %q", roots.Root, want)
	}
}
