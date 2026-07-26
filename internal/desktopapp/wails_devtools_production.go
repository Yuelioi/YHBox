//go:build production

package desktopapp

import "github.com/wailsapp/wails/v3/pkg/application"

func webviewDebugKeyBindings() map[string]func(application.Window) {
	return nil
}
