package hotkey

import (
	"fmt"
	"strings"
)

// HotkeyService 是 wails3 binding RPC 入口，前端通过它读/改热键。
// 所有 mutation 都走 HotkeyRegistry，由它做 reserved / conflict / 持久化 / emit。
type HotkeyService struct {
	reg            *HotkeyRegistry
	systemDefaults map[string]string // 内置热键出厂默认 (key→hotkeyStr)，给 ResetSystemDefaults 用
}

func NewHotkeyService(reg *HotkeyRegistry) *HotkeyService {
	return &HotkeyService{reg: reg}
}

// SetSystemDefaults 注入内置热键的出厂默认值 (key→hotkeyStr)。main.go 启动期调一次。
// 「重置默认」只覆盖这些 key — 容器 / action / schedule 是用户数据，不在内。
func (s *HotkeyService) SetSystemDefaults(d map[string]string) { s.systemDefaults = d }

// ResetSystemDefaults 把所有内置热键恢复出厂默认 (强停 / 校准 / 录制停止 / 录制暂停)。
// 两遍：先全清再逐条设默认 —— 避免新旧默认值跟当前自定义值瞬时冲突 (e.g. 用户把两个键对调)。
// 逐条走 registry.Update → 各自持久化 + emit。容器热键不动。
func (s *HotkeyService) ResetSystemDefaults() error {
	for key := range s.systemDefaults {
		_ = s.reg.Update(key, "")
	}
	var errs []string
	for key, def := range s.systemDefaults {
		if err := s.reg.Update(key, def); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分重置失败: %s", strings.Join(errs, "; "))
	}
	return nil
}

// List 返回所有 hotkey entry 给前端"快捷键" tab 渲染。
func (s *HotkeyService) List() []HotkeyEntry {
	return s.reg.List()
}

// Update 改某 entry 的热键绑定。hotkeyStr="" 表示清空。
// error message 带 [conflict] / [reserved] / [invalid] 前缀让前端做分类 toast。
func (s *HotkeyService) Update(key, hotkeyStr string) error {
	return s.reg.Update(key, hotkeyStr)
}

// Pause 临时反注册所有 OS hotkey。前端 HotkeyCaptureInput 进入捕获模式时调，
// 不暂停的话 Win32 会拦截已注册组合（如 Ctrl+Shift+1）直接派发给原 action，
// webview 收不到 keystroke。
func (s *HotkeyService) Pause() error {
	return s.reg.Pause()
}

// Resume Pause 的反操作。前端 stopListening 时调。
func (s *HotkeyService) Resume() error {
	return s.reg.Resume()
}

// RegisterEditor 暴露给 FE — useEditorHotkeys onActivated 时调。
// label / readonlyReason 是 i18n key 串 (FE t() 渲染).
// conflict / parse / reserved 失败不返 err — entry 进表 status=failed + lastError 描述,
// FE 表里 lastError 渲染让用户看见冲突源 (SettingsHotkeys.vue:30-36).
func (s *HotkeyService) RegisterEditor(key, label, hotkeyStr, readonlyReason string) error {
	return s.reg.RegisterEditor(key, label, hotkeyStr, readonlyReason)
}

// Unregister 暴露给 FE — useEditorHotkeys onDeactivated 清理.
// 不存在的 key noop, 返 nil.
func (s *HotkeyService) Unregister(key string) error {
	return s.reg.Unregister(key)
}
