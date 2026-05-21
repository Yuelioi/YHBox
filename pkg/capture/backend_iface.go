package capture

import (
	"errors"
	"fmt"
	"image"
	"sync"

	"github.com/lxn/win"
)

// IBackend per-container instance interface. 与现有的 Backend enum + 包级 Frame/FrameROI
// 并存 — Phase A 删 bot 时旧 caller 随之退役, 那时再清理重名.
//
// impl 必须 IsWindow(hwnd) 前置 + defer recover(), invalid hwnd 返 (nil, error) 永不 panic.
// WGC impl 额外捕 WGC Session 错误 (Direct3D11CaptureFramePool 在窗口刚关闭瞬间会抛 Closed).
type IBackend interface {
	Name() string
	Frame(hwnd win.HWND) (*image.RGBA, error)
	FrameROI(hwnd win.HWND, x, y, w, h int) (*image.RGBA, error)
	ClientSize(hwnd win.HWND) (int, int, error)
	Close() error
}

// 各 backend init 函数 (initWGC / initMock) 是 process-global 单次初始化.
// 同一 process 多 container 拿 IBackend 时, 这些 init 只跑一次. 用 once 保证.
var (
	wgcInitOnce  sync.Once
	wgcInitErr   error
	mockInitOnce sync.Once
	mockInitErr  error
)

func ensureWGCInit() error {
	wgcInitOnce.Do(func() {
		wgcInitErr = initWGC()
	})
	return wgcInitErr
}

func ensureMockInit() error {
	mockInitOnce.Do(func() {
		mockInitErr = initMock()
	})
	return mockInitErr
}

// NewIBackend 区分 4 种语义:
//
//	"auto" → 用包级 AutoBackend() 同款 OS 选择 (build>=20348 才用 WGC); fail fallback GDI
//	"wgc"  → unsupported OS → warning + fallback GDI, GDI 也 fail → err
//	"gdi"  → fail → err (无 fallback)
//	"mock" → 失败 → err
//
// warning 是 string (空 = 没事); err 是 fatal (caller 应该停 container 启动).
//
// 阈值跟 AutoBackend() 保持一致 (Win11/Server2022 才默选 WGC). Win10 下 WGC 对 DX 游戏
// 容易抓黑帧 / stale buffer, 之前 v2 这里写 18362 (Win10 1903) 跟包级 20348 不一致 →
// fishing v2 实测时 ColorBarTrack 抓到的帧 yellow/green 像素全 0, v1 (走 AutoBackend GDI)
// 同窗口同 ROI 跑得通. 统一到 AutoBackend() 避免再分叉.
func NewIBackend(name string) (b IBackend, warning string, err error) {
	switch name {
	case "", "auto":
		switch AutoBackend() {
		case BackendWGC:
			if wb, e := newWGCBackend(); e == nil {
				return wb, "", nil
			}
			if gb, e := newGDIBackend(); e == nil {
				return gb, "", nil
			}
			return nil, "", errors.New("auto: WGC + GDI 都初始化失败")
		default:
			if gb, e := newGDIBackend(); e == nil {
				return gb, "", nil
			}
			return nil, "", errors.New("auto: GDI init 失败")
		}

	case "wgc":
		if WindowsBuild() >= 18362 {
			if wb, e := newWGCBackend(); e == nil {
				return wb, "", nil
			}
			// 支持的 OS 上 WGC 还失败 — fallback GDI + warning
			if gb, e := newGDIBackend(); e == nil {
				return gb, "WGC 初始化失败, fallback 到 GDI", nil
			}
			return nil, "", errors.New("wgc: WGC + GDI 都失败")
		}
		// 不支持的 OS, fallback GDI + warning
		if gb, e := newGDIBackend(); e == nil {
			return gb, fmt.Sprintf("WGC 要求 Windows 10 1903+, 当前 build %d, fallback 到 GDI", WindowsBuild()), nil
		}
		return nil, "", errors.New("wgc: unsupported OS 且 GDI fallback 也失败")

	case "gdi":
		if gb, e := newGDIBackend(); e == nil {
			return gb, "", nil
		}
		return nil, "", errors.New("gdi: init 失败 (无 fallback for explicit gdi)")

	case "mock":
		if mb, e := newMockBackend(); e == nil {
			return mb, "", nil
		}
		return nil, "", errors.New("mock: init 失败")

	default:
		return nil, "", fmt.Errorf("unknown capture backend %q (supported: auto/wgc/gdi/mock)", name)
	}
}
