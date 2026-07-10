//go:build !windows

package hotkey

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/pkg/platform"
)

func TestHotkeyManagerRegisterReportsUnsupportedPlatform(t *testing.T) {
	manager := NewHotkeyManager()
	_, err := manager.Register(HotkeySpec{Mods: MOD_CONTROL, VK: VK_1, Name: "Ctrl+1"}, OwnerAction, func() {})
	if !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("Register() error = %v, want platform.ErrUnsupported", err)
	}
	if bindings := manager.Bindings(); len(bindings) != 0 {
		t.Fatalf("failed registration left bindings: %#v", bindings)
	}
}
