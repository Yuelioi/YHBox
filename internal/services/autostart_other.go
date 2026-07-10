//go:build !windows

package services

import (
	"fmt"
	"runtime"
)

func ApplyAutostart(enabled bool) error {
	if !enabled {
		return nil
	}
	return fmt.Errorf("autostart is not supported on %s", runtime.GOOS)
}
