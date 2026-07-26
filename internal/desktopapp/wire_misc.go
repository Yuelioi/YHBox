// wire_misc.go 杂项 wiring：schedule daemon 的 hotkey registrar。
package desktopapp

import (
	"github.com/yottaapp/yotta/internal/hotkey"
)

// ---- Schedule HotkeyRegistrar: *hotkey.HotkeyRegistry → schedule.HotkeyRegistrar ----
//
// schedule 包用 string 参数解耦 services 类型；这里把 source 字符串转 HotkeySource。

type scheduleHotkeyRegistrar struct {
	reg *hotkey.HotkeyRegistry
}

func (a *scheduleHotkeyRegistrar) Register(key, source, label string, labelParams map[string]string, hotkeyStr, readonly string, onFire func()) error {
	return a.reg.Register(key, hotkey.HotkeySource(source), label, labelParams, hotkeyStr, readonly, onFire)
}

func (a *scheduleHotkeyRegistrar) Unregister(key string) error {
	return a.reg.Unregister(key)
}
