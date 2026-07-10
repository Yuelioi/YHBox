//go:build !windows

package capture

import (
	"fmt"

	"github.com/yottaapp/yotta/pkg/platform"
)

// NewIBackend keeps the portable mock available and classifies Windows-only adapters.
func NewIBackend(name string) (IBackend, string, error) {
	switch name {
	case "mock":
		backend, err := newMockBackend()
		if err != nil {
			return nil, "", err
		}
		return backend, "", nil
	case "", "auto", "wgc", "gdi":
		if name == "" {
			name = "auto"
		}
		return nil, "", platform.NewUnsupportedError("capture backend " + name)
	default:
		return nil, "", fmt.Errorf("unknown capture backend %q (supported: auto/wgc/gdi/mock)", name)
	}
}
