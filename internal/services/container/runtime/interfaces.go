package runtime

import (
	"context"
	"image"

	"yhbox/internal/services/expr"
	"yhbox/internal/services/inputclip"
)

// ClipResolver PlayClip 节点用: clipID → InputClip. main.go 注入 inputclip.Service 适配.
type ClipResolver interface {
	Resolve(clipID string) (*inputclip.InputClip, bool)
}

// TemplateMatcher Wait/Check/ClickTemplate 节点用。注入实现见 main.go 适配器。
// v2.1 加 containerID — 模板按容器隔离, 每容器自己的 templates/ 目录.
// ctx 用于 timeout/cancel, frame 由 caller 抓好传入.
type TemplateMatcher interface {
	// Detect 单次检测. frame 由 caller 抓好传入 (nil → 无帧, 返 false).
	// region [r,r,r,r]（0..1 比例），nil → 全屏. containerID 定位模板目录.
	// 返 found + 命中位置（屏幕比例坐标）+ 命中 region + 实际匹配度 conf.
	// conf 即便 found=false 也返真实最高匹配度 (供超时/miss 诊断: 看「差多少」). 无帧/无 variant → 0.
	Detect(ctx context.Context, containerID string, frame *image.RGBA, templateKey string, threshold float64, region []float64) (found bool, point expr.Point, regionOut [4]float64, conf float64, err error)
}

// NoopMatcher：测试 + 启动前没注入实现时的默认。
type NoopMatcher struct{}

func (NoopMatcher) Detect(ctx context.Context, containerID string, frame *image.RGBA, k string, th float64, region []float64) (bool, expr.Point, [4]float64, float64, error) {
	return false, expr.Point{}, [4]float64{}, 0, nil
}

// GameProvider 提供跨进程窗口置前能力。main.go 启动时注入适配器。
// hwnd 解析已落在 rt.Window (WindowTarget 节点填), GameProvider 只剩 BringToForeground。
type GameProvider interface {
	// BringToForeground 尝试把 hwnd 置前。返 true 表示 OS 接受调用；
	// 注意：成功 ≠ 一定到前台（OS 可能延迟切换），但 false 一定是失败。
	BringToForeground(hwnd uintptr) bool
}
