//go:build !windows

package calibration

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/pkg/platform"
)

func TestServiceReportsUnsupportedNativeCapabilities(t *testing.T) {
	service := NewService(nil, nil)
	if err := service.Start(); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("Start() error = %v, want platform.ErrUnsupported", err)
	}
	if err := service.StartHotkeyWatch(); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("StartHotkeyWatch() error = %v, want platform.ErrUnsupported", err)
	}
	if state, err := service.Stop(); err != nil || state.Active {
		t.Fatalf("Stop() = (%#v, %v), want inactive state and nil error", state, err)
	}
	service.StopHotkeyWatch()
}
