package tools

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/lxn/win"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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
	// pickerWindows: requestID → window，方便复用（同 id 重开聚焦旧窗口）
	pickerWindows map[string]*application.WebviewWindow
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

// OpenRecordingHUD 打开录制控制悬浮窗 — 200×60 frameless + AlwaysOnTop.
// 内容: REC 红点 + 计时 + 停止按钮. 解决 "录制时切回 YHBox 不方便" 痛点.
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
		Title:                             "录制控制",
		Width:                             220,
		Height:                            60,
		URL:                               "/#/tools/recording-hud",
		Frameless:        true,
		AlwaysOnTop:      true,
		DisableResize:    true,
		BackgroundColour: application.NewRGB(24, 24, 27),
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
// 带上 id 给调用方匹配。
func (s *Service) OpenScreenPicker(mode, requestID string) error {
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

	hashURL := "/#/tools/screen-picker?mode=" + url.QueryEscape(mode) + "&id=" + url.QueryEscape(requestID)
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
