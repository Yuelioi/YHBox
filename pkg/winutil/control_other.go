//go:build !windows

package winutil

import "github.com/yottaapp/yotta/pkg/platform"

func IsWindow(uintptr) bool { return false }

func Maximize(uintptr) error { return platform.NewUnsupportedError("maximize native window") }
func Minimize(uintptr) error { return platform.NewUnsupportedError("minimize native window") }
func Restore(uintptr) error  { return platform.NewUnsupportedError("restore native window") }

func MoveResize(uintptr, int, int, int, int) error {
	return platform.NewUnsupportedError("move or resize native window")
}

func CloseWindow(uintptr) error { return platform.NewUnsupportedError("close native window") }

type WindowState struct {
	State      string
	Foreground bool
	X          int
	Y          int
	Width      int
	Height     int
}

func InspectWindowState(uintptr) (WindowState, error) {
	return WindowState{}, platform.NewUnsupportedError("inspect native window")
}
