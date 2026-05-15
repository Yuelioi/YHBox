// pkg/capture 提供异环客户区帧抓取。三条后端可选：
//   - BackendGDI: PrintWindow + GDI（默认，所有 Windows 都能跑，但游戏在后台时
//     有概率拿到冻结/黑帧 — DX 游戏失焦时尤其常见）
//   - BackendWGC: Windows Graphics Capture（Win10 1903+，后台抓帧更稳，需要外部
//     capture_wgc.dll，启动期 SetBackend(BackendWGC) 才会尝试加载）
//   - BackendMock: 从磁盘 PNG 序列伪装抓帧，给离线调参 / 回放调试用。
//     看 mock.go 里 mockDir() 的搜索路径。游戏不需要打开。
//
// 调用方一律走 capture.Frame / capture.FrameROI / capture.NewCapturer 这三个包级 API；
// 选 backend 是 main.go 启动期 SetBackend 的一次性决定（GUI 切语言/换 backend 后
// 提示用户重启 exe）。
package capture

import (
	"fmt"
	"image"

	"github.com/lxn/win"
)

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

// Capturer 是高频抓多 ROI 的会话句柄。GDI/WGC 各自实现。
//
// 调用约定：
//   - Frame() 必须串行调用（同一 capturer 内不并发）
//   - Close() 可从任意 goroutine 调用，重复调用安全
//   - 返回的 imgs[i].Pix 是 capturer 内部复用 buffer，下次 Frame 会被覆盖
type Capturer interface {
	Frame() ([]*image.RGBA, error)
	Close() error
}

// active 是当前生效的后端。默认 GDI，SetBackend 切换。
var active Backend = BackendGDI

// LastInitWarning 是 backend 初始化时收集的次要警告（不致命，主功能仍可用）。
// 比如 WGC 的 SetIsBorderRequired 在 Win10 老版本上返回 Err 但 capture 仍能工作。
// main.go 启动期读一次写日志面板，方便诊断。
var LastInitWarning string

// SetBackend 切换截屏后端。WGC 后端会尝试加载 capture_wgc.dll，加载失败返错。
// 启动期 main.go 根据 settings 调一次；GUI 改设置需重启 exe 才生效（避免运行中
// 切后端导致 Capturer 资源混乱）。
func SetBackend(b Backend) error {
	switch b {
	case BackendGDI:
		active = b
		return nil
	case BackendWGC:
		if err := initWGC(); err != nil {
			return err
		}
		active = b
		return nil
	case BackendMock:
		if err := initMock(); err != nil {
			return err
		}
		active = b
		return nil
	default:
		return fmt.Errorf("unknown backend %d", b)
	}
}

// Active 返回当前生效的后端。给前端/日志显示用。
func Active() Backend { return active }

// Frame 抓一帧客户区全图。后端由 active 决定。
func Frame(hwnd win.HWND) (*image.RGBA, error) {
	switch active {
	case BackendWGC:
		return wgcFrame(hwnd)
	case BackendMock:
		return mockFrame(hwnd)
	default:
		return gdiFrame(hwnd)
	}
}

// FrameROI 抓客户区指定矩形区域。返回图的 (0,0) 对应 (roiX, roiY)。
func FrameROI(hwnd win.HWND, roiX, roiY, roiW, roiH int) (*image.RGBA, error) {
	switch active {
	case BackendWGC:
		return wgcFrameROI(hwnd, roiX, roiY, roiW, roiH)
	case BackendMock:
		return mockFrameROI(hwnd, roiX, roiY, roiW, roiH)
	default:
		return gdiFrameROI(hwnd, roiX, roiY, roiW, roiH)
	}
}

// NewCapturer 创建高频多 ROI 会话。失败返回错误（窗口尺寸异常、ROI 越界、
// WGC 加载失败等）。
func NewCapturer(hwnd win.HWND, rois []image.Rectangle) (Capturer, error) {
	switch active {
	case BackendWGC:
		return newWGCCapturer(hwnd, rois)
	case BackendMock:
		return newMockCapturer(hwnd, rois)
	default:
		return newGDICapturer(hwnd, rois)
	}
}
