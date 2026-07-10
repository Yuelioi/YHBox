//go:build !windows

package platform

import (
	"fmt"
	"runtime"
)

const (
	SWHide            = 0
	SWShowNormal      = 1
	SWShowMaximized   = 3
	SWShowMinNoActive = 7
)

func ShellOpen(_, _, _ string, _ int) error {
	return fmt.Errorf("shell open is not supported on %s", runtime.GOOS)
}
