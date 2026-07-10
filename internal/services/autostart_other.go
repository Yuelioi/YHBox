//go:build !windows

package services

import "github.com/yottaapp/yotta/pkg/platform"

func ApplyAutostart(enabled bool) error {
	if !enabled {
		return nil
	}
	return platform.NewUnsupportedError("autostart")
}
