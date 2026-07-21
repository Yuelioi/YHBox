package recording

import "github.com/yottaapp/yotta/internal/services/calibration"

func newStartHotkeyWatch(vk uint32, callback func()) startHotkeyWatch {
	return calibration.NewHotkeyHook(vk, callback)
}
