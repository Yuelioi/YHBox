package platform

import (
	"errors"
	"fmt"
	"runtime"
)

// ErrUnsupported classifies capabilities that the current host OS cannot provide.
var ErrUnsupported = errors.New("platform capability is not supported")

// UnsupportedError preserves the unavailable capability and host OS for diagnostics.
type UnsupportedError struct {
	Capability string
	OS         string
}

// NewUnsupportedError describes a capability unavailable on the current host OS.
func NewUnsupportedError(capability string) *UnsupportedError {
	return &UnsupportedError{Capability: capability, OS: runtime.GOOS}
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("%s is not supported on %s", e.Capability, e.OS)
}

func (e *UnsupportedError) Unwrap() error { return ErrUnsupported }
