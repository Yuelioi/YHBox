//go:build !windows

package input

import (
	"fmt"

	"github.com/yottaapp/yotta/pkg/platform"
)

// NewBackend rejects known Windows adapters with a classified platform error.
func NewBackend(name string) (Backend, error) {
	switch name {
	case "", "postmessage", "sendinput":
		if name == "" {
			name = "sendinput"
		}
		return nil, platform.NewUnsupportedError("input backend " + name)
	default:
		return nil, fmt.Errorf("unknown input backend %q (supported: postmessage, sendinput)", name)
	}
}
