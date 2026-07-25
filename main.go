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
	desktopMainWithReporter(desktopapp.Run, os.Stderr, showStartupError, os.Exit)
}

func desktopMain(start func(desktopapp.Config) error, stderr io.Writer, exit func(int)) {
	desktopMainWithReporter(start, stderr, func(string) {}, exit)
}

func desktopMainWithReporter(start func(desktopapp.Config) error, stderr io.Writer, report func(string), exit func(int)) {
	if err := start(desktopapp.Config{Assets: assets, TrayIcon: trayIcon}); err != nil {
		message := fmt.Sprintf("Yotta startup failed: %v", err)
		_, _ = fmt.Fprintln(stderr, message)
		report(message)
		exit(1)
	}
}
