// Yotta desktop process entrypoint.
package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/yottaapp/yotta/internal/desktopapp"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

func main() {
	if err := desktopapp.Run(desktopapp.Config{Assets: assets, TrayIcon: trayIcon}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Yotta startup failed: %v\n", err)
		os.Exit(1)
	}
}
