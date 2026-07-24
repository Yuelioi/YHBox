package main

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/yottaapp/yotta/internal/desktopapp"
)

func TestDesktopManifestRequiresAdministrator(t *testing.T) {
	manifest, err := os.ReadFile("build/windows/wails.exe.manifest")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifest, []byte(`requestedExecutionLevel level="requireAdministrator" uiAccess="false"`)) {
		t.Fatal("desktop manifest must require administrator privileges")
	}
}

func TestDesktopDevManifestRunsAsInvoker(t *testing.T) {
	manifest, err := os.ReadFile("build/windows/wails.dev.manifest")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifest, []byte(`requestedExecutionLevel level="asInvoker" uiAccess="false"`)) {
		t.Fatal("development manifest must run as invoker so Wails can supervise the process")
	}
	if bytes.Contains(manifest, []byte(`requestedExecutionLevel level="requireAdministrator"`)) {
		t.Fatal("development manifest must not request elevation")
	}
}

func TestDesktopMainDelegatesEmbeddedResourcesAndReportsStartupFailure(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := 0
	desktopMain(func(config desktopapp.Config) error {
		if len(config.TrayIcon) == 0 {
			t.Error("tray icon was not delegated")
		}
		if _, err := config.Assets.ReadFile("frontend/dist/index.html"); err != nil {
			t.Errorf("frontend assets were not delegated: %v", err)
		}
		return errors.New("boom")
	}, &stderr, func(code int) { exitCode = code })
	if exitCode != 1 || stderr.String() != "Yotta startup failed: boom\n" {
		t.Fatalf("unexpected startup failure: exit=%d stderr=%q", exitCode, stderr.String())
	}
}
