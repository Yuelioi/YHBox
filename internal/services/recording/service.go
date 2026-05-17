package recording

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lxn/win"

	"yhbox/internal/services/inputclip"
)

// GameHwndProvider 给录制器拿当前游戏窗口 hwnd。
// main.go 启动时注入（services.App.Game() 适配）。
type GameHwndProvider interface {
	HWND() (uintptr, bool)
}

// HotkeySettingsProvider 给 Service 拿停录热键 VK + mouseMode.
// main.go 注入 settings adapter. nil = 默认 F12 + relative.
type HotkeySettingsProvider interface {
	GetStopHotkeyVK() uint32 // 0x7B = F12 默认
	GetMouseMode() string    // 'relative' / 'absolute'
}

// Service wails3 RPC 入口.
//
// 前端流程:
//  1. Start({filterMode}) → 立即返 tempID; hook 已挂上但用户还没停
//  2. (用户在 UI / F12 全局热键触发停止)
//  3. Stop() → 落盘 InputClip 到 clipSvc.Store, 拿 clip 元数据
//
// 停录热键: LL hook 直接检测 → return 1 拦截不透传游戏 → 异步调 StopAsync.
// StopAsync 完成时 emit 'recording:completed' {clipID} | {error} 给前端订阅.
type Service struct {
	rec     *Recorder
	game    GameHwndProvider
	hkProv  HotkeySettingsProvider
	clipSvc *inputclip.Service
	emit    func(name string, data any)
	mu      sync.Mutex // 防 F12 stop callback 跟 UI Stop 同时跑
}

func NewService(rec *Recorder, game GameHwndProvider, hkProv HotkeySettingsProvider, clipSvc *inputclip.Service) *Service {
	return &Service{rec: rec, game: game, hkProv: hkProv, clipSvc: clipSvc}
}

// SetEmit main.go 启动期注入. wails3 application.Event.Emit 包一层.
func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

// StartArgs 前端传入的录制开关.
type StartArgs struct {
	FilterMode string `json:"filterMode"` // 'precise' | 'simple'
}

// Start 启动录制(非阻塞). 返回临时录制 ID (前端订阅事件流过滤用).
//
// Start 后会 atomic 设 LL hook 的 stopHotkeyVK + callback —
// 用户在游戏前台按 F12 时, hook 直接拦截 (不透传游戏) 并异步触发 StopAsync.
func (s *Service) Start(args StartArgs) (string, error) {
	if s.game == nil {
		return "", errors.New("game provider 未注入")
	}
	hwnd, ok := s.game.HWND()
	if !ok || hwnd == 0 {
		return "", errors.New("游戏窗口未就绪")
	}
	mouseMode := "relative"
	stopVK := uint32(0x7B)
	if s.hkProv != nil {
		mouseMode = s.hkProv.GetMouseMode()
		stopVK = s.hkProv.GetStopHotkeyVK()
	}
	filterMode := args.FilterMode
	if filterMode == "" {
		filterMode = "precise"
	}
	meta := inputclip.ClipMeta{
		MouseMode:      mouseMode,
		FilterMode:     filterMode,
		StopHotkeyVK:   stopVK,
		BaseResolution: [2]int{1920, 1080}, // TODO: 真实读取游戏窗口尺寸
		WindowMode:     "fullscreen",
	}
	id, err := s.rec.Start(win.HWND(hwnd), meta)
	if err != nil {
		return "", fmt.Errorf("recorder.Start: %w", err)
	}
	SetActiveStopHotkey(stopVK, func() {
		s.StopAsync()
	})
	return id, nil
}

// Stop 同步停止录制 — 取出 clip, 落盘到 clipSvc.Store, 返回 clip (含 events).
//
// 注意: 用 internal mutex 防 F12 stop callback 跟 UI Stop 重入 (Recorder.Stop 不可重入).
func (s *Service) Stop() (*inputclip.InputClip, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.rec.Active() {
		return nil, errors.New("recorder not active")
	}
	clip, err := s.rec.Stop()
	// 不管成败都清 stopHotkey, 避免悬挂 callback 引发误触
	SetActiveStopHotkey(0, nil)
	if err != nil {
		return nil, err
	}
	if s.clipSvc != nil {
		if err := s.clipSvc.Save(clip); err != nil {
			return nil, fmt.Errorf("save clip: %w", err)
		}
	}
	return clip, nil
}

// StopAsync 异步停录 — 跑后 emit 'recording:completed' {clipID} 或 {error}.
// F12 hook callback 走这条 (callback 不能阻塞 hook 线程 50ms+).
// 前端 RecordingHUD 子窗口主动调它 (拿不到 RPC 返回值, 靠订阅事件).
func (s *Service) StopAsync() {
	go func() {
		clip, err := s.Stop()
		if s.emit == nil {
			return
		}
		if err != nil {
			s.emit("recording:completed", map[string]any{"error": err.Error()})
			return
		}
		s.emit("recording:completed", map[string]any{"clipID": clip.ID})
	}()
}

// IsRecording 状态查询.
func (s *Service) IsRecording() bool {
	return s.rec.Active()
}
