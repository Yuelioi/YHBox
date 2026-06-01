package main

import (
	"yhbox/internal/hotkey"
	"yhbox/internal/services"
	"yhbox/internal/services/inputclip"
	"yhbox/internal/services/recording"
)

// recordingHkAdapter 拿停录/暂停热键 VK (读 hotkey registry) + mouseMode (读 settings)。
// 录制热键现进了热键中心 (recording.stop/pause, ll-hook 机制), registry 是权威;
// 用户在「快捷键」页 rebind → 下次录制 Start 读到新值 (无需重启)。
type recordingHkAdapter struct {
	app *services.App
	reg *hotkey.HotkeyRegistry
}

func (a *recordingHkAdapter) GetStopHotkeyVK() uint32 {
	return a.hotkeyVK("recording.stop", 0x7B) // F12 fallback
}

func (a *recordingHkAdapter) GetPauseHotkeyVK() uint32 {
	return a.hotkeyVK("recording.pause", 0x7A) // F11 fallback
}

func (a *recordingHkAdapter) hotkeyVK(key string, fallback uint32) uint32 {
	if a.reg == nil {
		return fallback
	}
	e, ok := a.reg.Get(key)
	if !ok || e.HotkeyStr == "" {
		return fallback
	}
	_, vk, err := hotkey.ParseHotkey(e.HotkeyStr)
	if err != nil || vk == 0 {
		return fallback
	}
	return vk
}

func (a *recordingHkAdapter) GetMouseMode() string {
	m := a.app.Settings().UI.RecordingMouseMode
	if m != "absolute" {
		return "relative"
	}
	return m
}

// newRecordingService 构造完整 recording service 链路.
//
// F12 停录路径 (v2): LL hook 检测 vk == activeStopHotkeyVK → return 1 不透传游戏 →
// 异步调 service.StopAsync → emit 'recording:completed'. 停录/暂停热键值从 hotkey
// registry 取 (recording.stop/pause, ll-hook 机制), 不再读 settings raw 字段。
func newRecordingService(app *services.App, clipSvc *inputclip.Service, reg *hotkey.HotkeyRegistry) *recording.Service {
	rec := recording.NewRecorder()
	rec.SetMouseCounts360Getter(func() int {
		return app.Settings().ActiveMouseCounts360()
	})
	return recording.NewService(rec, &recordingHkAdapter{app: app, reg: reg}, clipSvc)
}
