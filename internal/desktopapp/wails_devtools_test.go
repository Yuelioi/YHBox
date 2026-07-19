//go:build !production

package desktopapp

import "testing"

func TestDevelopmentMainWindowExposesDevToolsShortcut(t *testing.T) {
	options := mainWindowOptions(1400, 900)
	if options.KeyBindings["Ctrl+Shift+I"] == nil {
		t.Fatal("development main window does not expose the DevTools shortcut")
	}
}
