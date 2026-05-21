package runtime

import (
	"context"

	"yhbox/internal/services/expr"
	"yhbox/internal/services/inputclip"
)

// ClipResolver PlayClip 节点用: clipID → InputClip. main.go 注入 inputclip.Service 适配.
type ClipResolver interface {
	Resolve(clipID string) (*inputclip.InputClip, bool)
}

// TemplateMatcher Wait/Check/ClickTemplate 节点用。注入实现见 main.go 适配器。
// v2.1 加 containerID — 模板按容器隔离, 每容器自己的 templates/ 目录.
// ctx 用于 timeout/cancel, hwnd 从 rt.Window.HWND 拿.
type TemplateMatcher interface {
	// Detect 单次检测。region [r,r,r,r]（0..1 比例），nil → 全屏。
	// containerID 用于定位该容器的模板目录. hwnd 0 表示 noop.
	// 返 found + 命中位置（屏幕比例坐标）+ 命中 region。
	Detect(ctx context.Context, containerID string, hwnd uintptr, templateKey string, threshold float64, region []float64) (found bool, point expr.Point, regionOut [4]float64, err error)
}

// ColorDetector DetectColor 节点用：在 ROI 内统计落在颜色范围内的像素。
// 实现解析当前游戏窗口截屏 + 抠 ROI + 遍历像素 + HSV/RGB 判定。
//
//	region: 客户区比例 [x, y, w, h]，全 0 = 全屏。
//	mode:   "hsv" | "rgb"。
//	rng:    6 元 — hsv: [hMin,hMax,sMin,sMax,vMin,vMax]；rgb: [rMin,rMax,gMin,gMax,bMin,bMax]
//
// 返：命中像素数 / 命中中心客户区比例坐标 (cx, cy)。无命中时 cx/cy = 0。
type ColorDetector interface {
	Detect(ctx context.Context, hwnd uintptr, region [4]float64, mode string, rng [6]int) (count int, cx, cy float64, err error)
}

// NoopColorDetector：测试 + 启动前没注入实现时的默认。
type NoopColorDetector struct{}

func (NoopColorDetector) Detect(context.Context, uintptr, [4]float64, string, [6]int) (int, float64, float64, error) {
	return 0, 0, 0, nil
}

// NoopMatcher：测试 + 启动前没注入实现时的默认。
type NoopMatcher struct{}

func (NoopMatcher) Detect(ctx context.Context, containerID string, hwnd uintptr, k string, th float64, region []float64) (bool, expr.Point, [4]float64, error) {
	return false, expr.Point{}, [4]float64{}, nil
}

// GameProvider runtime 需要知道当前游戏窗口 HWND + 能调 BringToForeground。
// main.go 启动时注入适配器（1.22 wire）。
//
// v3 Phase B: WindowTarget 已经把 hwnd 解析放在 rt.Window, 这里只剩 BringToForeground
// 是仍由 GameProvider 提供的能力 (跨进程窗口置前). HWND() 字段已经无 caller, 待后续清理.
type GameProvider interface {
	// BringToForeground 尝试把 hwnd 置前。返 true 表示 OS 接受调用；
	// 注意：成功 ≠ 一定到前台（OS 可能延迟切换），但 false 一定是失败。
	BringToForeground(hwnd uintptr) bool
}
