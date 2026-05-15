package hotkey

// HotkeyService 是 wails3 binding RPC 入口，前端通过它读/改热键。
// 所有 mutation 都走 HotkeyRegistry，由它做 reserved / conflict / 持久化 / emit。
type HotkeyService struct {
	reg *HotkeyRegistry
}

func NewHotkeyService(reg *HotkeyRegistry) *HotkeyService {
	return &HotkeyService{reg: reg}
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
