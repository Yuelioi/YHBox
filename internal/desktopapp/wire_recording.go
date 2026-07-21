package desktopapp

import (
	"context"

	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/hotkey"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/services/recording"
)

type recordingTargetAcquirer interface {
	AcquireRecordingTarget(context.Context, string) (target.WindowHandle, int, func(), error)
}

// recordingCalibrationTargets keeps target-specific playback calibration as
// the explicit override, while precise authoring follows the user's active
// mouse calibration profile when the target has no override of its own.
type recordingCalibrationTargets struct {
	targets         recordingTargetAcquirer
	activeCounts360 func() int
}

func (targets *recordingCalibrationTargets) AcquireRecordingTarget(ctx context.Context, slot string) (target.WindowHandle, int, func(), error) {
	window, counts360, release, err := targets.targets.AcquireRecordingTarget(ctx, slot)
	if err != nil || counts360 > 0 || targets.activeCounts360 == nil {
		return window, counts360, release, err
	}
	return window, targets.activeCounts360(), release, nil
}

// recordingHkAdapter 拿开始/停录/暂停热键 VK (读 hotkey registry) + mouseMode (读 settings)。
// 录制热键现进了热键中心 (recording.start/stop/pause, ll-hook 机制), registry 是权威;
// 用户在「快捷键」页 rebind → 下次录制 Start 读到新值 (无需重启)。
type recordingHkAdapter struct {
	app *services.App
	reg *hotkey.HotkeyRegistry
}

func (a *recordingHkAdapter) GetStartHotkeyVK() uint32 {
	return a.hotkeyVK("recording.start", 0x79) // F10 fallback
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
// registry 取 (recording.start/stop/pause, ll-hook 机制), 不再读 settings raw 字段。
func newRecordingService(app *services.App, clipSvc *inputclip.Service, macroSvc *macro.Service, reg *hotkey.HotkeyRegistry, targets automationinstalled.AuthoringTargets, emit ...func(name string, data any)) *recording.Service {
	rec := recording.NewRecorder()
	calibratedTargets := &recordingCalibrationTargets{
		targets: targets,
		activeCounts360: func() int {
			if app == nil {
				return 0
			}
			return app.Settings().ActiveMouseCounts360()
		},
	}
	return recording.NewService(rec, &recordingHkAdapter{app: app, reg: reg}, clipSvc, macroSvc, calibratedTargets, emit...)
}
