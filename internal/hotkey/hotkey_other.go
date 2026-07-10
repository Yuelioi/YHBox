//go:build !windows

package hotkey

import (
	"context"

	"github.com/yottaapp/yotta/pkg/platform"
)

func runHotkeyLoop(_ context.Context, _ []HotkeySpec, _ func(int), ready chan<- error, done chan<- struct{}) {
	defer close(done)
	ready <- platform.NewUnsupportedError("global hotkeys")
}
