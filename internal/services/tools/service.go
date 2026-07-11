package tools

import (
	"fmt"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/automation/target"
)

// WindowResolver 由 main.go 注入 (concrete *container.Service)，按 containerID 解析目标窗口。
// 本包不 import container 包以保持解耦。
type WindowResolver interface {
	ResolveWindowForNode(containerID, nodeID string) (target.WindowHandle, error)
	ResolveEditorTargetForNode(containerID, nodeID string) (target.Target, error)
	CaptureBackendFor(containerID string) string
}

// cachedWindow gameWindowFor 的短期缓存条目 (MousePos 高频 poll，不能每帧 EnumWindows)。
// 注意: wh.HWND 在 2s TTL 内若游戏关了又开会变陈旧失效，调用方 (screenToClient / capture.Frame) 需容错。
type cachedWindow struct {
	wh target.WindowHandle
	at time.Time
}

// Service is the tools RPC entry point.
type Service struct {
	resolver  WindowResolver
	presenter Presenter

	mu              sync.Mutex
	winCache        map[string]cachedWindow // containerID → 解析结果 (2s TTL)
	hud             windowSlot
	recordingHUD    windowSlot
	calibratorHUD   windowSlot
	launcher        windowSlot
	launcherVisible bool // 只反映本 feature 的 Show/Hide，不强保证跟 OS 同步
	// onCalibratorClose: 校准 HUD 窗关闭时的兜底清理 (main.go 注入 → 卸 F8 钩 + 停 session)。
	// 覆盖 ESC / Alt+F4 / 崩溃 等不走前端正常关闭的路径。
	onCalibratorClose func()
	// pickerWindows: requestID → lifecycle slot，方便复用（同 id 重开聚焦旧窗口）
	pickerWindows map[string]*windowSlot
	targetTools   targetToolRouter
	// captureHotkey 返当前「窗口捕获」键的 (mods, vk)。main.go 从 hotkey registry
	// (tools.window-capture 条目) 注入；nil 或返 vk==0 时 StartWin32WindowTargetCapture 回退 F9。
	captureHotkey func() (mods, vk uint32)
}

func NewService(resolver WindowResolver, presenter Presenter) *Service {
	s := &Service{
		resolver:      resolver,
		presenter:     presenter,
		winCache:      map[string]cachedWindow{},
		pickerWindows: map[string]*windowSlot{},
	}
	s.targetTools = newTargetToolRouter(map[string]TargetToolAdapter{
		target.KindWin32Window: win32TargetToolAdapter{service: s},
		target.KindAndroidADB:  androidTargetToolAdapter{service: s},
	})
	return s
}

// gameWindowFor 按 containerID + nodeID 解析目标窗口，带 2s 缓存 (MousePos 高频 poll 不能每帧 EnumWindows)。
// nodeID 为空时回落容器主 Win32WindowTarget。
func (s *Service) gameWindowFor(containerID, nodeID string) (target.WindowHandle, bool) {
	cacheKey := containerID + "|" + nodeID
	s.mu.Lock()
	if c, ok := s.winCache[cacheKey]; ok && time.Since(c.at) < 2*time.Second {
		wh := c.wh
		s.mu.Unlock()
		return wh, wh.HWND != 0
	}
	s.mu.Unlock()

	// 锁外调阻塞解析 (winutil.ResolveWindow 窗口没开时最长等 3s), 不持锁, 不卡其它 RPC.
	wh, err := s.resolver.ResolveWindowForNode(containerID, nodeID)

	s.mu.Lock()
	defer s.mu.Unlock()
	// double-check: 期间并发 poll 可能已写缓存
	if c, ok := s.winCache[cacheKey]; ok && time.Since(c.at) < 2*time.Second {
		return c.wh, c.wh.HWND != 0
	}
	if err != nil {
		// 负缓存: 失败也存零值 2s, 避免游戏关着时高频 poll 反复阻塞解析
		s.winCache[cacheKey] = cachedWindow{at: time.Now()}
		return target.WindowHandle{}, false
	}
	s.winCache[cacheKey] = cachedWindow{wh: wh, at: time.Now()}
	return wh, true
}

// ConfigureCaptureHotkeyGetter main.go 注入「窗口捕获」键读取器（从 hotkey registry 读
// tools.window-capture 当前绑定）。让捕获键统一走热键中心、可 rebind，不再硬编 F9。
func ConfigureCaptureHotkeyGetter(s *Service, getter func() (mods, vk uint32)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captureHotkey = getter
}

func (s *Service) windowPresenter() Presenter {
	if s.presenter == nil || !s.presenter.Ready() {
		return nil
	}
	return s.presenter
}

// MousePos 当前鼠标在 containerID 目标窗口客户区 + 屏幕的位置。HUD 高频 poll。
// nodeID 指定当前编辑节点（按最近上游 Win32WindowTarget 解析窗口）；无节点上下文传 ""。
func (s *Service) MousePos(containerID, nodeID string) (MousePosInfo, error) {
	sx, sy, err := readCursor()
	info := MousePosInfo{ScreenX: sx, ScreenY: sy}
	if err != nil {
		return info, err
	}
	wh, hasGame := s.gameWindowFor(containerID, nodeID)
	if !hasGame {
		return info, nil
	}
	hwnd, cw, ch := wh.HWND, wh.ClientW, wh.ClientH
	if hwnd == 0 || cw <= 0 || ch <= 0 {
		return info, nil
	}
	info.HasGame = true
	info.ClientW, info.ClientH = cw, ch
	cx, cy, err := screenToClient(hwnd, sx, sy)
	if err != nil {
		return info, err
	}
	info.ClientX, info.ClientY = cx, cy
	if cw > 0 {
		info.XRatio = float64(cx) / float64(cw)
	}
	if ch > 0 {
		info.YRatio = float64(cy) / float64(ch)
	}
	return info, nil
}

// OpenMouseHUD 打开鼠标位置 HUD 小窗口 (按 containerID 解析目标窗口)。已开则 focus。
func (s *Service) OpenMouseHUD(containerID string) error {
	presenter := s.windowPresenter()
	if presenter == nil {
		return apperr.New(apperr.CodeWailsNotReady, nil)
	}
	w, opened, err := s.openWindow(presenter, &s.hud, WindowRequest{
		Kind:        WindowMouseHUD,
		ContainerID: containerID,
	}, nil)
	if err != nil {
		return err
	}
	if !opened && w != nil {
		w.Focus()
	}
	return nil
}

// OpenRecordingHUD 打开录制控制悬浮窗 — 实心盒子窗 frameless + AlwaysOnTop (跟校准窗同风格).
// 内容: 标题栏(X) / 大号 REC 计时 + 模式 / 暂停·继续·停止按钮 + 热键 hint. 解决 "录制时切回 Yotta 不方便" 痛点.
// 已开则 focus. 用户关闭窗口 / 录制结束都触发自动关.
func (s *Service) OpenRecordingHUD() error {
	presenter := s.windowPresenter()
	if presenter == nil {
		return apperr.New(apperr.CodeWailsNotReady, nil)
	}
	w, opened, err := s.openWindow(presenter, &s.recordingHUD, WindowRequest{Kind: WindowRecordingHUD}, nil)
	if err != nil {
		return err
	}
	if !opened && w != nil {
		w.Focus()
	}
	return nil
}

// OpenLauncher 打开（或显示+聚焦）悬浮窗启动器。frameless + AlwaysOnTop，仿 OpenRecordingHUD。
// 入口按钮调；关闭按钮走 HideLauncher（隐藏不销毁）。
func (s *Service) OpenLauncher() error {
	presenter := s.windowPresenter()
	if presenter == nil {
		return apperr.New(apperr.CodeWailsNotReady, nil)
	}
	w, opened, err := s.openWindow(presenter, &s.launcher, WindowRequest{Kind: WindowLauncher}, func() {
		s.mu.Lock()
		s.launcherVisible = false
		s.mu.Unlock()
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.launcherVisible = true
	s.mu.Unlock()
	if !opened && w != nil {
		w.Show()
		w.Focus()
	}
	return nil
}

// ToggleLauncher 呼出/隐藏（toggle 全局热键调）。可见 → 隐；否则 → 开/显示。
func (s *Service) ToggleLauncher() error {
	s.mu.Lock()
	visible := s.launcherVisible && s.launcher.window != nil
	w := s.launcher.window
	s.mu.Unlock()
	if visible {
		s.mu.Lock()
		s.launcherVisible = false
		s.mu.Unlock()
		w.Hide()
		return nil
	}
	return s.OpenLauncher()
}

// HideLauncher 标题栏 × 按钮调（FE）：隐藏不销毁，状态保留。
func (s *Service) HideLauncher() error {
	s.mu.Lock()
	w := s.launcher.window
	s.launcherVisible = false
	s.mu.Unlock()
	if w != nil {
		w.Hide()
	}
	return nil
}

// SetLauncherAlwaysOnTop 图钉切换：运行时改启动器窗口置顶（默认开窗即置顶）。
func (s *Service) SetLauncherAlwaysOnTop(on bool) error {
	w := s.currentWindow(&s.launcher)
	if w != nil {
		w.SetAlwaysOnTop(on)
	}
	return nil
}

// SetLauncherSize 程序化设启动器窗口尺寸（前端按内容自适应高度 / 右下角手柄拖拽时调用）。
func (s *Service) SetLauncherSize(width, height int) error {
	w := s.currentWindow(&s.launcher)
	if w != nil && width > 0 && height > 0 {
		w.SetSize(width, height)
	}
	return nil
}

// ConfigureCalibratorCloseHandler main.go 注入: 校准 HUD 窗关闭时卸 F8 钩 + 停 session。
func ConfigureCalibratorCloseHandler(s *Service, handler func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCalibratorClose = handler
}

// OpenCalibratorHUD 打开独立置顶校准窗 (frameless + AlwaysOnTop)。
// 校准测量走全局 raw-input (INPUTSINK, 不挑前台窗口), F8 走自治 LL 钩 —— 所以这里不绑任何游戏:
// 开窗后用户自己切到想校准的游戏/软件按 F8 即可。通用框架不假设"唯一的那个游戏", 也不强制把某个
// 检测到的游戏置前 (那是 v1 单游戏残留)。requestID 透到窗口 URL, 校准完成时窗口 emit
// 'calibration:result' 带 id 给调用方匹配。已开则 focus。
//
// 返 (opened, error) 而非纯 error: wails3 纯 error 方法经 FE invoke 后成功/失败都返 undefined,
// 调用方无法区分 (见 incident wails-error-only-rpc-invoke-undefined)。
func (s *Service) OpenCalibratorHUD(requestID string) (bool, error) {
	presenter := s.windowPresenter()
	if presenter == nil {
		return false, apperr.New(apperr.CodeWailsNotReady, nil)
	}
	w, opened, err := s.openWindow(presenter, &s.calibratorHUD, WindowRequest{
		Kind:      WindowCalibratorHUD,
		RequestID: requestID,
	}, func() {
		s.mu.Lock()
		cb := s.onCalibratorClose
		s.mu.Unlock()
		if cb != nil {
			cb()
		}
	})
	if err != nil {
		return false, err
	}
	if !opened && w != nil {
		w.Focus()
	}
	if !opened && w == nil {
		return false, nil
	}
	return true, nil
}

// CloseCalibratorHUD 前端校准完成/取消时调, 让后端关窗 (前端拿不到 self handle)。幂等。
func (s *Service) CloseCalibratorHUD() error {
	w := s.invalidateWindow(&s.calibratorHUD)
	if w != nil {
		w.Close()
		s.mu.Lock()
		cb := s.onCalibratorClose
		s.mu.Unlock()
		if cb != nil {
			cb()
		}
	}
	return nil
}

// CloseRecordingHUD 录制 service 停录时调, 主动关 HUD.
// 已关或没开时 idempotent.
func (s *Service) CloseRecordingHUD() {
	w := s.invalidateWindow(&s.recordingHUD)
	if w != nil {
		w.Close()
	}
}

// OpenScreenPicker 打开屏幕选择器。mode: "point" | "rect" | "template_save" | "template_recapture" | "color"。
// requestID 调用方生成（UUID），picker 完成时通过 emit 事件 "tools:picker-result"
// 带上 id 给调用方匹配。containerID 仅 template_save / template_recapture 模式需要（空字符串则保存失败）。
// nodeID 指定当前编辑节点（按最近上游 Win32WindowTarget 截图）；无节点上下文传 ""。
// colorSpace 仅 color 模式需要（"hsv" | "rgb"），其他模式传 ""。
// guid 仅 template_recapture 模式需要（重拍目标资产 GUID，存成同 GUID 的新分辨率档）；其他模式传 ""。
func (s *Service) OpenScreenPicker(mode, requestID, containerID, nodeID, colorSpace, guid string) error {
	if mode != "point" && mode != "rect" && mode != "template_save" && mode != "template_recapture" && mode != "color" {
		return fmt.Errorf("unsupported mode %q", mode)
	}
	if requestID == "" {
		return fmt.Errorf("requestID 不能为空")
	}
	tg := target.Target{Kind: target.KindWin32Window}
	if s.resolver != nil {
		resolved, err := s.resolver.ResolveEditorTargetForNode(containerID, nodeID)
		if err != nil {
			return err
		}
		if resolved.Kind != "" {
			tg = resolved
		}
	}
	return s.targetTools.OpenPicker(tg, PickerRequest{
		Mode:        mode,
		RequestID:   requestID,
		ContainerID: containerID,
		NodeID:      nodeID,
		ColorSpace:  colorSpace,
		GUID:        guid,
	})
}

// ClosePicker 由 picker 完成后或取消时调，前端拿不到 self window handle，
// 让后端清理。
func (s *Service) ClosePicker(requestID string) error {
	s.mu.Lock()
	slot, ok := s.pickerWindows[requestID]
	if ok {
		delete(s.pickerWindows, requestID)
	}
	s.mu.Unlock()
	if !ok || slot == nil {
		return nil
	}
	if w := s.invalidateWindow(slot); w != nil {
		w.Close()
	}
	return nil
}

// --- Win32WindowTarget capture (F9 global hotkey, async via event) ---

// StartWin32WindowTargetCapture 注册「窗口捕获」热键 (默认 F9, 走热键中心可 rebind),
// 用户按下后:
//  1. 查前台窗口 metadata
//  2. emit "win32windowtarget:captured" event {title, class, processName, clientW, clientH}
//  3. 自动反注册热键
//
// 键来源 = ConfigureCaptureHotkeyGetter 注入的 registry 绑定值 (mods+vk); 未注入回退 F9。
// 同时只能一个 capture session. 用户多次开启需要先 CancelWin32WindowTargetCapture.
// 返 captureID 给前端用来 cancel.
//
// 流程: 前端 NodeInspector 点 "捕获" → 调本 RPC → 用户 Alt-Tab 到游戏 → 按该键
// → 前端收 event 填表. 取代旧 CaptureForegroundWindow (用户在游戏前台时无法点按钮).
func (s *Service) StartWin32WindowTargetCapture() (string, error) {
	if err := win32WindowCaptureSupported(); err != nil {
		return "", err
	}
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
	presenter := s.windowPresenter()
	if presenter == nil {
		return "", apperr.New(apperr.CodeWailsNotReady, nil)
	}
	return startWin32WindowTargetCapture(mods, vk, func(name string, data any) {
		presenter.Emit(name, data)
	})
}

// CancelWin32WindowTargetCapture 主动 cancel 一个等待中的 capture session.
// captureID 必须匹配 — 不匹配 / 无活跃 session 都返 nil (idempotent).
// 前端组件 unmount / 用户再点按钮取消都调本 RPC.
func (s *Service) CancelWin32WindowTargetCapture(captureID string) error {
	return cancelWin32WindowTargetCapture(captureID)
}
