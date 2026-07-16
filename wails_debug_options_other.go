//go:build !windows || production

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func wailsWindowsOptions() application.WindowsOptions {
	return application.WindowsOptions{}
}
