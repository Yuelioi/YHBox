package adbexec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvPathWins(t *testing.T) {
	got := Resolve(ResolveOptions{
		EnvPath:       `C:\custom\adb.exe`,
		ExecutableDir: t.TempDir(),
		WorkingDir:    t.TempDir(),
		GOOS:          "windows",
	})
	if got != `C:\custom\adb.exe` {
		t.Fatalf("Resolve() = %q, want env path", got)
	}
}

func TestResolveBundledPlatformToolsNextToExecutable(t *testing.T) {
	exeDir := t.TempDir()
	adbPath := filepath.Join(exeDir, "platform-tools", "adb.exe")
	writeEmptyFile(t, adbPath)

	got := Resolve(ResolveOptions{
		ExecutableDir: exeDir,
		WorkingDir:    t.TempDir(),
		GOOS:          "windows",
	})
	if got != adbPath {
		t.Fatalf("Resolve() = %q, want %q", got, adbPath)
	}
}

func TestResolveBundledPlatformToolsUnderWorkingBin(t *testing.T) {
	wd := t.TempDir()
	adbPath := filepath.Join(wd, "bin", "platform-tools", "adb.exe")
	writeEmptyFile(t, adbPath)

	got := Resolve(ResolveOptions{
		ExecutableDir: t.TempDir(),
		WorkingDir:    wd,
		GOOS:          "windows",
	})
	if got != adbPath {
		t.Fatalf("Resolve() = %q, want %q", got, adbPath)
	}
}

func TestResolveFallsBackToPath(t *testing.T) {
	got := Resolve(ResolveOptions{
		ExecutableDir: t.TempDir(),
		WorkingDir:    t.TempDir(),
		GOOS:          "windows",
	})
	if got != "adb" {
		t.Fatalf("Resolve() = %q, want PATH fallback", got)
	}
}

func writeEmptyFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o755); err != nil {
		t.Fatal(err)
	}
}
