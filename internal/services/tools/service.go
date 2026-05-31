package tools

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/lxn/win"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"yhbox/pkg/winutil"
)

// GameProvider 由 main.go 注入，返当前游戏 hwnd + client 尺寸。
type GameProvider interface {
	GameHWND() (hwnd win.HWND, w int, h int, ok bool)
}

// Service wails3 RPC 入口。
type Service struct {
	game GameProvider

	mu             sync.Mutex
	app            *application.App // 后注入（wailsApp 创建后才有）
	hud            *application.WebviewWindow
	recordingHUD   *application.WebviewWindow
	calibratorHUD  *application.WebviewWindow
	// onCalibratorClose: 校准 HUD 窗关闭时的兜底清理 (main.go 注入 → 卸 F8 钩 + 停 session)。
	// 覆盖 ESC / Alt+F4 / 崩溃 等不走前端正常关闭的路径。
	onCalibratorClose func()
	// pickerWindows: requestID → window，方便复用（同 id 重开聚焦旧窗口）
	pickerWindows map[string]*application.WebviewWindow
	// captureHotkey 返当前「窗口捕获」键的 (mods, vk)。main.go 从 hotkey registry
	// (tools.window-capture 条目) 注入；nil 或返 vk==0 时 StartWindowTargetCapture 回退 F9。
	captureHotkey func() (mods, vk uint32)
}

func NewService(game GameProvider) *Service {
	return &Service{
		game:          game,
		pickerWindows: map[string]*application.WebviewWindow{},
	}
}

// SetApp main.go wailsApp 创建后注入。
func (s *Service) SetApp(app *application.App) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.app = app
}

// SetCaptureHotkeyGetter main.go 注入「窗口捕获」键读取器（从 hotkey registry 读
// tools.window-capture 当前绑定）。让捕获键统一走热键中心、可 rebind，不再硬编 F9。
func (s *Service) SetCaptureHotkeyGetter(fn func() (mods, vk uint32)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captureHotkey = fn
}

func (s *Service) wailsApp() *application.App {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.app
}

// MousePos 当前鼠标在屏幕 + 游戏客户区的位置。HUD 高频 poll。
func (s *Service) MousePos() MousePosInfo {
	sx, sy, ok := readCursor()
	info := MousePosInfo{ScreenX: sx, ScreenY: sy}
	if !ok {
		return info
	}
	hwnd, cw, ch, hasGame := s.game.GameHWND()
	if !hasGame || hwnd == 0 || cw <= 0 || ch <= 0 {
		return info
	}
	info.HasGame = true
	info.ClientW, info.ClientH = cw, ch
	cx, cy, ok2 := screenToClient(hwnd, sx, sy)
	if !ok2 {
		return info
	}
	info.ClientX, info.ClientY = cx, cy
	if cw > 0 {
		info.XRatio = float64(cx) / float64(cw)
	}
	if ch > 0 {
		info.YRatio = float64(cy) / float64(ch)
	}
	return info
}

// OpenMouseHUD 打开鼠标位置 HUD 小窗口。已开则 focus。
func (s *Service) OpenMouseHUD() error {
	app := s.wailsApp()
	if app == nil {
		return fmt.Errorf("wails app 未初始化")
	}
	s.mu.Lock()
	if s.hud != nil {
		s.mu.Unlock()
		s.hud.Focus()
		return nil
	}
	s.mu.Unlock()
	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:           "鼠标位置",
		Width:           320,
		Height:          240,
		MinWidth:        260,
		MinHeight:       180,
		URL:             "/#/tools/mouse-hud",
		Frameless:       true,
		AlwaysOnTop:     true,
		BackgroundColour: application.NewRGB(18, 18, 18),
	})
	s.mu.Lock()
	s.hud = w
	s.mu.Unlock()
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		s.mu.Lock()
		s.hud = nil
		s.mu.Unlock()
	})
	return nil
}

// OpenRecordingHUD 打开录制控制悬浮窗 — 双行卡片 frameless + AlwaysOnTop + 透明背景.
// 内容: REC 红点 + 计时 + 模式 / 暂停·继续·停止按钮 / F12 hint. 解决 "录制时切回 YHBox 不方便" 痛点.
// BackgroundTypeTransparent: 窗口本体透明, 圆角外露桌面 (修黑角); 卡片视觉由前端半透明玻璃画.
// 已开则 focus. 用户关闭窗口 / 录制结束都触发自动关.
func (s *Service) OpenRecordingHUD() error {
	app := s.wailsApp()
	if app == nil {
		return fmt.Errorf("wails app 未初始化")
	}
	s.mu.Lock()
	if s.recordingHUD != nil {
		s.mu.Unlock()
		s.recordingHUD.Focus()
		return nil
	}
	s.mu.Unlock()
	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "录制控制",
		Width:          260,
		Height:         96,
		URL:            "/#/tools/recording-hud",
		Frameless:      true,
		AlwaysOnTop:    true,
		DisableResize:  true,
		BackgroundType: application.BackgroundTypeTransparent,
	})
	s.mu.Lock()
	s.recordingHUD = w
	s.mu.Unlock()
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		s.mu.Lock()
		s.recordingHUD = nil
		s.mu.Unlock()
	})
	return nil
}

// SetCalibratorCloseHandler main.go 注入: 校准 HUD 窗关闭时卸 F8 钩 + 停 session。
func (s *Service) SetCalibratorCloseHandler(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCalibratorClose = fn
}

// OpenCalibratorHUD 打开独立置顶校准窗 (frameless + AlwaysOnTop), 并自动把目标游戏窗口置前
// (用户不用自己 alt-tab)。检测不到游戏窗口 → 返 error 让前端提示。requestID 透到窗口 URL,
// 校准完成时窗口 emit 'calibration:result' 带 id 给调用方匹配。已开则 focus。
func (s *Service) OpenCalibratorHUD(requestID string) (bool, error) {
	app := s.wailsApp()
	if app == nil {
		return false, fmt.Errorf("wails app 未初始化")
	}
	hwnd, _, _, ok := s.game.GameHWND()
	if !ok || hwnd == 0 {
		return false, fmt.Errorf("未检测到目标游戏窗口, 先在容器里捕获游戏窗口再校准")
	}
	s.mu.Lock()
	if s.calibratorHUD != nil {
		w := s.calibratorHUD
		s.mu.Unlock()
		w.Focus()
		winutil.BringToFront(hwnd) // 仍把游戏置前
		return true, nil
	}
	s.mu.Unlock()

	hashURL := "/#/tools/calibration-hud?id=" + url.QueryEscape(requestID)
	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "鼠标校准",
		Width:            360,
		Height:           220,
		URL:              hashURL,
		Frameless:        true,
		AlwaysOnTop:      true,
		DisableResize:    true,
		BackgroundColour: application.NewRGB(18, 18, 18),
	})
	s.mu.Lock()
	s.calibratorHUD = w
	s.mu.Unlock()
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		s.mu.Lock()
		s.calibratorHUD = nil
		cb := s.onCalibratorClose
		s.mu.Unlock()
		if cb != nil {
			cb() // 兜底: 卸 F8 钩 + 停 session (ESC/Alt+F4/崩溃都走这)
		}
	})
	// 窗口建好后把游戏置前 — HUD 是 AlwaysOnTop 仍浮在游戏上, 但焦点给游戏 (鼠标转动才进游戏)。
	winutil.BringToFront(hwnd)
	return true, nil
}

// CloseCalibratorHUD 前端校准完成/取消时调, 让后端关窗 (前端拿不到 self handle)。幂等。
func (s *Service) CloseCalibratorHUD() error {
	s.mu.Lock()
	w := s.calibratorHUD
	s.calibratorHUD = nil
	s.mu.Unlock()
	if w != nil {
		w.Close()
	}
	return nil
}

// CloseRecordingHUD 录制 service 停录时调, 主动关 HUD.
// 已关或没开时 idempotent.
func (s *Service) CloseRecordingHUD() {
	s.mu.Lock()
	w := s.recordingHUD
	s.recordingHUD = nil
	s.mu.Unlock()
	if w != nil {
		w.Close()
	}
}

// OpenScreenPicker 打开屏幕选择器。mode: "point" | "rect" | "template_save"。
// requestID 调用方生成（UUID），picker 完成时通过 emit 事件 "tools:picker-result"
// 带上 id 给调用方匹配。containerID 仅 template_save 模式需要（空字符串则保存失败）。
func (s *Service) OpenScreenPicker(mode, requestID, containerID string) error {
	app := s.wailsApp()
	if app == nil {
		return fmt.Errorf("wails app 未初始化")
	}
	if mode != "point" && mode != "rect" && mode != "template_save" {
		return fmt.Errorf("unsupported mode %q", mode)
	}
	if requestID == "" {
		return fmt.Errorf("requestID 不能为空")
	}
	s.mu.Lock()
	if existing, ok := s.pickerWindows[requestID]; ok {
		s.mu.Unlock()
		existing.Focus()
		return nil
	}
	s.mu.Unlock()

	hashURL := "/#/tools/screen-picker?mode=" + url.QueryEscape(mode) + "&id=" + url.QueryEscape(requestID) + "&containerID=" + url.QueryEscape(containerID)
	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "选择屏幕位置",
		Width:     1280,
		Height:    800,
		MinWidth:  720,
		MinHeight: 480,
		URL:       hashURL,
		Frameless: true,
	})
	s.mu.Lock()
	s.pickerWindows[requestID] = w
	s.mu.Unlock()
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		s.mu.Lock()
		delete(s.pickerWindows, requestID)
		s.mu.Unlock()
	})
	return nil
}

// ClosePicker 由 picker 完成后或取消时调，前端拿不到 self window handle，
// 让后端清理。
func (s *Service) ClosePicker(requestID string) error {
	s.mu.Lock()
	w, ok := s.pickerWindows[requestID]
	if ok {
		delete(s.pickerWindows, requestID)
	}
	s.mu.Unlock()
	if !ok || w == nil {
		return nil
	}
	w.Close()
	return nil
}

// --- WindowTarget capture (F9 global hotkey, async via event) ---

// StartWindowTargetCapture 注册「窗口捕获」热键 (默认 F9, 走热键中心可 rebind),
// 用户按下后:
//  1. 查前台窗口 metadata
//  2. emit "windowtarget:captured" event {title, class, processName, clientW, clientH}
//  3. 自动反注册热键
//
// 键来源 = SetCaptureHotkeyGetter 注入的 registry 绑定值 (mods+vk); 未注入回退 F9。
// 同时只能一个 capture session. 用户多次开启需要先 CancelWindowTargetCapture.
// 返 captureID 给前端用来 cancel.
//
// 流程: 前端 NodeInspector 点 "捕获" → 调本 RPC → 用户 Alt-Tab 到游戏 → 按该键
// → 前端收 event 填表. 取代旧 CaptureForegroundWindow (用户在游戏前台时无法点按钮).
func (s *Service) StartWindowTargetCapture() (string, error) {
	var mods, vk uint32
	s.mu.Lock()
	getter := s.captureHotkey
	s.mu.Unlock()
	if getter != nil {
		mods, vk = getter()
	}
	if vk == 0 {
		mods, vk = 0, 0x78 // VK_F9 回退
	}
	app := s.wailsApp()
	if app == nil {
		return "", errors.New("wails app 未初始化, 无法 emit event")
	}
	return startWindowTargetCapture(mods, vk, func(name string, data any) {
		app.Event.Emit(name, data)
	})
}

// CancelWindowTargetCapture 主动 cancel 一个等待中的 capture session.
// captureID 必须匹配 — 不匹配 / 无活跃 session 都返 nil (idempotent).
// 前端组件 unmount / 用户再点按钮取消都调本 RPC.
func (s *Service) CancelWindowTargetCapture(captureID string) error {
	return cancelWindowTargetCapture(captureID)
}
