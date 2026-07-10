//go:build !windows

package calibration

import "github.com/yottaapp/yotta/pkg/platform"

// Start reports that raw-input DPI calibration is unavailable on this host.
func Start() error {
	return platform.NewUnsupportedError("mouse DPI calibration")
}

// Stop is idempotent when calibration could not be started on this host.
func Stop() (State, error) {
	return Get(), nil
}

// HotkeyHook preserves the service lifecycle contract on unsupported hosts.
type HotkeyHook struct{}

// NewHotkeyHook creates an unsupported host adapter.
func NewHotkeyHook(uint32, func()) *HotkeyHook { return &HotkeyHook{} }

// Start reports that the global calibration hotkey is unavailable.
func (*HotkeyHook) Start() error {
	return platform.NewUnsupportedError("calibration global hotkey")
}

// Stop is an idempotent no-op for an unsupported hook.
func (*HotkeyHook) Stop() {}
