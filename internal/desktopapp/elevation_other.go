//go:build !windows

package desktopapp

import "github.com/yottaapp/yotta/internal/platform"

func launchElevated() error { return platform.ErrUnsupported }
