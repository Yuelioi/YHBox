// Yotta desktop process entrypoint.
package main

import (
	"embed"
	"fmt"
	"io"
	"os"

	"github.com/yottaapp/yotta/internal/desktopapp"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

func main() {
	desktopMain(desktopapp.Run, os.Stderr, os.Exit)
}

func desktopMain(start func(desktopapp.Config) error, stderr io.Writer, exit func(int)) {
	if err := start(desktopapp.Config{Assets: assets, TrayIcon: trayIcon}); err != nil {
		_, _ = fmt.Fprintf(stderr, "Yotta startup failed: %v\n", err)
		exit(1)
	}
}
