// internal/node/resolve.go
package node

import "fmt"

// ResolvePoint 把 Point 按 Unit 归一成 0-1 客户区比例.
// UnitRatio → 原样(不碰 Window);UnitPx → 取 ClientSize 后 px÷宽高.
// 不复用 ResolveScalar — 那带 |v|<=1⇒比例 启发, 对显式 px 小值会误判.
func ResolvePoint(ctx Ctx, p Point) (xRatio, yRatio float64, err error) {
	if p.Unit != UnitPx {
		return p.X, p.Y, nil
	}
	w, h, err := ctx.Services().Window.ClientSize()
	if err != nil {
		return 0, 0, err
	}
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("client size %dx%d invalid for px point", w, h)
	}
	return p.X / float64(w), p.Y / float64(h), nil
}
