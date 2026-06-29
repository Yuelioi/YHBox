package runtime

import (
	"context"
	"image"

	"yotta/internal/node"
	"yotta/internal/services/expr"
	"yotta/internal/services/inputclip"
)

// ClipResolver PlayClip 节点用: clipID → InputClip. main.go 注入 inputclip.Service 适配.
type ClipResolver interface {
	Resolve(clipID string) (*inputclip.InputClip, bool)
}

// TemplateMatcher Wait/Check/ClickTemplate 节点用。注入实现见 main.go 适配器。
// 资产全局 (asset.Store): 节点按 guid 引用, 不再 per-container 隔离.
// ctx 用于 timeout/cancel, frame 由 caller 抓好传入.
type TemplateMatcher interface {
	// Detect 单次检测. frame 由 caller 抓好传入 (nil → 无帧, 返 false).
	// guid 定位全局资产记录; region [r,r,r,r]（0..1 比例），nil → 全屏.
	// scaleTolerance 由 caller (有容器上下文) 传入 — 跨分辨率缩放兜底的容差 [1/k, k]; <=0 走默认.
	// 返 found + 命中位置（屏幕比例坐标）+ 命中 region + 实际匹配度 conf.
	// conf 即便 found=false 也返真实最高匹配度 (供超时/miss 诊断: 看「差多少」). 无帧/无 variant → 0.
	Detect(ctx context.Context, frame *image.RGBA, guid string, threshold float64, region []float64, scaleTolerance float64) (found bool, point expr.Point, regionOut [4]float64, conf float64, err error)

	// DetectAll 单帧返回某模板的所有命中 (3×3 局部极大 + 单模板内 NMS)。
	// 坐标全帧归一化 0..1; 每命中带 BBox + TemplateKey(=guid)。语义参数同 Detect。spec §节点2。
	DetectAll(ctx context.Context, frame *image.RGBA, guid string, threshold float64, region []float64, scaleTolerance float64) ([]node.TemplateMatch, error)
}

// NoopMatcher：测试 + 启动前没注入实现时的默认。
type NoopMatcher struct{}

func (NoopMatcher) Detect(ctx context.Context, frame *image.RGBA, guid string, th float64, region []float64, scaleTolerance float64) (bool, expr.Point, [4]float64, float64, error) {
	return false, expr.Point{}, [4]float64{}, 0, nil
}

func (NoopMatcher) DetectAll(ctx context.Context, frame *image.RGBA, guid string, th float64, region []float64, scaleTolerance float64) ([]node.TemplateMatch, error) {
	return nil, nil
}

// GameProvider 提供跨进程窗口置前能力。main.go 启动时注入适配器。
// hwnd 解析已落在当前活动窗口（经 rt.WindowHandle()/SetActiveWindow() 访问，Win32WindowTarget 节点运行时填入）, GameProvider 只剩 BringToForeground。
type GameProvider interface {
	// BringToForeground 尝试把 hwnd 置前。返 true 表示 OS 接受调用；
	// 注意：成功 ≠ 一定到前台（OS 可能延迟切换），但 false 一定是失败。
	BringToForeground(hwnd uintptr) bool
}
