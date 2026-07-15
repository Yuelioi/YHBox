//go:build !windows

package appcontrol

import (
	"context"
	"errors"
)

var errUnsupportedHost = errors.New("installed application lifecycle is unavailable on this host")

type unavailableController struct{}

func newPlatformController() platformController { return unavailableController{} }
func PlatformSupported() bool                   { return false }
func (unavailableController) Launch(context.Context, Profile) (uint32, error) {
	return 0, errUnsupportedHost
}
func (unavailableController) Terminate(context.Context, Profile) (int, error) {
	return 0, errUnsupportedHost
}
