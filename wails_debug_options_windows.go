//go:build windows && !production

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	webviewDebugPortEnv     = "YOTTA_WEBVIEW_DEBUG_PORT"
	webviewDebugProfileEnv  = "YOTTA_WEBVIEW_DEBUG_PROFILE"
	minimumWebviewDebugPort = 1024
)

func wailsWindowsOptions() application.WindowsOptions {
	var options application.WindowsOptions
	if profile := strings.TrimSpace(os.Getenv(webviewDebugProfileEnv)); profile != "" && filepath.IsAbs(profile) {
		options.WebviewUserDataPath = filepath.Clean(profile)
	}

	rawPort := strings.TrimSpace(os.Getenv(webviewDebugPortEnv))
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < minimumWebviewDebugPort || port > 65535 {
		return options
	}
	options.AdditionalBrowserArgs = []string{fmt.Sprintf("--remote-debugging-port=%d", port)}
	return options
}
