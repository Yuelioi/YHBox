package runtime

import (
	"context"

	"yhbox/internal/services/expr"
)

// TemplateMatcher Wait/Check/ClickTemplate 节点用。注入实现见 main.go 适配器。
// v1 仅 template_appeared 检测。
type TemplateMatcher interface {
	// Detect 单次检测。region [r,r,r,r]（0..1 比例），nil → 全屏。
	// 返 found + 命中位置（屏幕比例坐标）+ 命中 region。
	Detect(ctx context.Context, templateKey string, threshold float64, region []float64) (found bool, point expr.Point, regionOut [4]float64, err error)
}

// ActionInvoker InvokeAction 节点用。
type ActionInvoker interface {
	Invoke(ctx context.Context, actionID string, params map[string]expr.Value) error
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
	Detect(ctx context.Context, region [4]float64, mode string, rng [6]int) (count int, cx, cy float64, err error)
}

// NoopColorDetector：测试 + 启动前没注入实现时的默认。
type NoopColorDetector struct{}

func (NoopColorDetector) Detect(context.Context, [4]float64, string, [6]int) (int, float64, float64, error) {
	return 0, 0, 0, nil
}

// InputDriver 容器级输入原语：ClickAt / KeyPress / MouseMoveRel / Scroll。
// 实现解析当前游戏 hwnd + 调 pkg/input；测试用 NoopInputDriver 占位。
//
// 坐标用 0..1 比例；driver 自己换算到客户区像素（每次 capture.ClientSize）。
type InputDriver interface {
	Click(ctx context.Context, xRatio, yRatio float64, button string, durationMs int) error
	KeyPress(ctx context.Context, vk string, durationMs int) error
	MouseMoveRel(ctx context.Context, dx, dy int, durationMs int) error
	Scroll(ctx context.Context, xRatio, yRatio float64, delta int) error
}

// NoopMatcher / NoopInvoker / NoopInputDriver：测试 + 启动前没注入实现时的默认。
type NoopMatcher struct{}

func (NoopMatcher) Detect(ctx context.Context, k string, th float64, region []float64) (bool, expr.Point, [4]float64, error) {
	return false, expr.Point{}, [4]float64{}, nil
}

type NoopInvoker struct{}

func (NoopInvoker) Invoke(ctx context.Context, id string, params map[string]expr.Value) error {
	return nil
}

type NoopInputDriver struct{}

func (NoopInputDriver) Click(context.Context, float64, float64, string, int) error { return nil }
func (NoopInputDriver) KeyPress(context.Context, string, int) error                { return nil }
func (NoopInputDriver) MouseMoveRel(context.Context, int, int, int) error          { return nil }
func (NoopInputDriver) Scroll(context.Context, float64, float64, int) error        { return nil }
