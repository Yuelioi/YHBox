package calibration

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// VKF8 是 DPI 校准热键的出厂默认 (VK_F8); vkGetter 拿不到绑定时兜这个.
const VKF8 = 0x77

// Service 暴露给 wails 前端。raw-input 累积 session 管在包级平台 adapter；
// F8 全局热键的 LL hook 由本 Service 持有 (校准窗开关时 Start/StopHotkeyWatch).
type Service struct {
	emit     func(name string, data any) // 广播 'calibration:toggle' 给前端
	vkGetter func() uint32               // 读当前 F8 绑定的 vk (热键中心可 rebind)

	hookMu       sync.Mutex
	hook         *HotkeyHook
	lifecycleMu  sync.Mutex
	closed       atomic.Bool
	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shutdownErr  error
}

// NewService — emit 广播事件给前端, vkGetter 读当前校准热键 vk (可为 nil → 用 VK_F8).
func NewService(emit func(name string, data any), vkGetter func() uint32) *Service {
	return &Service{emit: emit, vkGetter: vkGetter, shutdownDone: make(chan struct{})}
}

// Start 启动校准（清零累积值，开始监听 raw mouse）。
func (s *Service) Start() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.closed.Load() {
		return errors.New("calibration service is closed")
	}
	Reset()
	return Start()
}

// Stop 停止校准并返回累积状态。
func (s *Service) Stop() (State, error) { return Stop() }

// Status 当前累积状态（前端 200ms poll 用）。
func (s *Service) Status() State { return Get() }

// StartHotkeyWatch 校准窗打开时调: 装 F8 LL hook, 命中 emit 'calibration:toggle'。
// 装钩失败返 error (杀软拦截等)。先停旧的再装, 防快速开关双钩。
func (s *Service) StartHotkeyWatch() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.closed.Load() {
		return errors.New("calibration service is closed")
	}
	s.hookMu.Lock()
	defer s.hookMu.Unlock()
	if s.hook != nil {
		s.hook.Stop()
		s.hook = nil
	}
	vk := uint32(VKF8)
	if s.vkGetter != nil {
		if v := s.vkGetter(); v != 0 {
			vk = v
		}
	}
	h := NewHotkeyHook(vk, func() {
		if s.emit != nil {
			s.emit("calibration:toggle", nil)
		}
	})
	if err := h.Start(); err != nil {
		return err
	}
	s.hook = h
	return nil
}

// StopHotkeyWatch 校准窗关闭时调: 卸 F8 LL hook。幂等。
func (s *Service) StopHotkeyWatch() {
	s.hookMu.Lock()
	defer s.hookMu.Unlock()
	if s.hook != nil {
		s.hook.Stop()
		s.hook = nil
	}
}

// Shutdown releases the service-owned LL hook before stopping raw-input capture.
// It stays outside the method set to avoid exposing lifecycle wiring as RPC.
func Shutdown(ctx context.Context, s *Service) error {
	s.shutdownOnce.Do(func() {
		s.closed.Store(true)
		go func() {
			s.lifecycleMu.Lock()
			s.StopHotkeyWatch()
			_, err := s.Stop()
			s.shutdownErr = err
			s.lifecycleMu.Unlock()
			close(s.shutdownDone)
		}()
	})
	select {
	case <-s.shutdownDone:
		s.lifecycleMu.Lock()
		defer s.lifecycleMu.Unlock()
		return s.shutdownErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
