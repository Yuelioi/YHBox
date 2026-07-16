// pkg/capture 提供客户区帧抓取。三条后端可选：
//   - BackendGDI: PrintWindow + GDI（所有 Windows 都能跑，但游戏在后台时
//     有概率拿到冻结/黑帧 — DX 游戏失焦时尤其常见）
//   - BackendWGC: Windows Graphics Capture（Win10 1903+，后台抓帧更稳，需要外部
//     capture_wgc.dll）
//   - BackendMock: 从磁盘 PNG 序列伪装抓帧，给离线调参 / 回放调试用。
//     看 mock.go 里 mockDir() 的搜索路径。游戏不需要打开。
//
// 调用方为每个 admitted Run 创建 IBackend；Backend enum + 常量供配置/日志用。
package capture

// Backend 标识截屏后端的选择。
type Backend int

const (
	BackendGDI Backend = iota
	BackendWGC
	BackendMock
)

// String 给日志/事件用。
func (b Backend) String() string {
	switch b {
	case BackendWGC:
		return "wgc"
	case BackendMock:
		return "mock"
	default:
		return "gdi"
	}
}
