//go:build windows && !production

package desktopapp

import (
	"path/filepath"
	"testing"
)

func TestWailsWindowsOptionsEnableExplicitDevCDP(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "webview2")
	t.Setenv(webviewDebugPortEnv, "9225")
	t.Setenv(webviewDebugProfileEnv, profile)

	options := wailsWindowsOptions()
	if len(options.AdditionalBrowserArgs) != 1 || options.AdditionalBrowserArgs[0] != "--remote-debugging-port=9225" {
		t.Fatalf("additional browser args = %v", options.AdditionalBrowserArgs)
	}
	if options.WebviewUserDataPath != profile {
		t.Fatalf("WebView profile = %q, want %q", options.WebviewUserDataPath, profile)
	}
}

func TestWailsWindowsOptionsIgnoreInvalidDevCDP(t *testing.T) {
	t.Setenv(webviewDebugPortEnv, "80")
	t.Setenv(webviewDebugProfileEnv, "relative-profile")

	options := wailsWindowsOptions()
	if len(options.AdditionalBrowserArgs) != 0 || options.WebviewUserDataPath != "" {
		t.Fatalf("invalid debug options were accepted: %+v", options)
	}
}
