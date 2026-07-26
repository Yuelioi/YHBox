//go:build !windows

package scriptengine

import "errors"

func verifyWorkerConfinement() error {
	return errors.New("script worker confinement is unavailable on this platform")
}
