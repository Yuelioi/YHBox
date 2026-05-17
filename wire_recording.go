package main

import (
	"yhbox/internal/hotkey"
	"yhbox/internal/services"
	"yhbox/internal/services/inputclip"
	"yhbox/internal/services/recording"
)

// recordingGameHwndAdapter 把 services.App.Game() 适配到 recording.GameHwndProvider
type recordingGameHwndAdapter struct {
	app *services.App
}

func (a *recordingGameHwndAdapter) HWND() (uintptr, bool) {
	g := a.app.Game()
	if g == nil || !g.OK {
		return 0, false
	}
	return uintptr(g.HWND), true
}

// recordingHkAdapter 拿停录热键 VK + mouseMode.
// Phase 5: GetStopHotkeyVK 读 settings.UI.RecordingStopHotkey 解析; mouseMode 仍硬编 relative
// (后续容器配置维度落地后再接管).
type recordingHkAdapter struct {
	app *services.App
}

func (a *recordingHkAdapter) GetStopHotkeyVK() uint32 {
	hk := a.app.Settings().UI.RecordingStopHotkey
	if hk == "" {
		return 0x7B // F12 fallback
	}
	_, vk, err := hotkey.ParseHotkey(hk)
	if err != nil || vk == 0 {
		return 0x7B
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
// 异步调 service.StopAsync → emit 'recording:completed'. 没了 HotkeyRegistry 备路
// — LL hook 直接走够稳, 不需要 OS RegisterHotKey 备份 (那个反而经常被游戏 reserve).
func newRecordingService(app *services.App, clipSvc *inputclip.Service) *recording.Service {
	rec := recording.NewRecorder()
	rec.SetMouseCounts360Getter(func() int {
		return app.Settings().UI.MouseCounts360
	})
	return recording.NewService(rec, &recordingGameHwndAdapter{app: app}, &recordingHkAdapter{app: app}, clipSvc)
}
