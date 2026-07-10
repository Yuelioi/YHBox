//go:build !windows

package runtime

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/pkg/platform"
)

func TestNewWin32ControllerProviderReportsUnsupportedPlatform(t *testing.T) {
	_, err := newWin32ControllerProvider(nil)
	if !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want platform.ErrUnsupported", err)
	}
}
