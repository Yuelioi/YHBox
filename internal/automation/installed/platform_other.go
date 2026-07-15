//go:build !windows

package installed

import "github.com/yottaapp/yotta/pkg/platform"

func PlatformSupported() bool { return false }

func newPlatformDriver(Profile) (driver, error) {
	return nil, failure(CodeUnsupportedHost, platform.NewUnsupportedError("installed automation target"))
}
