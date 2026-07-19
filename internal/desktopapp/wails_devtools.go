//go:build !production

package desktopapp

import "github.com/wailsapp/wails/v3/pkg/application"

func webviewDebugKeyBindings() map[string]func(application.Window) {
	return map[string]func(application.Window){
		"Ctrl+Shift+I": func(window application.Window) {
			window.OpenDevTools()
		},
	}
}
