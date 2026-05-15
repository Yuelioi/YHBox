// wire_misc.go 杂项 wiring：tools service 的 game provider，schedule
// daemon 的 hotkey registrar。各自一两行不值得单独成文件。
package main

import (
	"github.com/lxn/win"

	"yhbox/internal/hotkey"
	"yhbox/internal/services"
)

// ---- ToolsService GameProvider 适配 ----
//
// tools.Service 需要 (hwnd, w, h, ok) 四元组拿当前游戏窗口；这里从 services.App
// 转换 win.HWND 类型。

type toolsGameAdapter struct {
	app *services.App
}

func (t *toolsGameAdapter) GameHWND() (win.HWND, int, int, bool) {
	g := t.app.Game()
	if g == nil || !g.OK {
		return 0, 0, 0, false
	}
	return win.HWND(g.HWND), g.W, g.H, true
}

// ---- Schedule HotkeyRegistrar: *hotkey.HotkeyRegistry → schedule.HotkeyRegistrar ----
//
// schedule 包用 string 参数解耦 services 类型；这里把 source 字符串转 HotkeySource。

type scheduleHotkeyRegistrar struct {
	reg *hotkey.HotkeyRegistry
}

func (a *scheduleHotkeyRegistrar) Register(key, source, label, hotkeyStr, readonly string, onFire func()) error {
	return a.reg.Register(key, hotkey.HotkeySource(source), label, hotkeyStr, readonly, onFire)
}

func (a *scheduleHotkeyRegistrar) Unregister(key string) error {
	return a.reg.Unregister(key)
}
