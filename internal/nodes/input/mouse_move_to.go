package input

import (
	"encoding/json"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&MouseMoveTo{}) }

// MouseMoveTo 把光标 (可见地) 滑到客户区坐标 (Point), 不点击.
// Point 支持 ratio(0-1) 与 px 两种单位; px 由 ResolvePoint 换算为比例.
// MoveMs=0 → 瞬移; >0 → 在 MoveMs 内从当前光标位置直线分帧滑过去 (~60fps), 可被 ctx 取消.
type MouseMoveTo struct{}

const (
	mmtInExec      = "In"
	mmtInPoint     = "Point"
	mmtInMoveMs    = "MoveMs"
	mmtInJitterPct = "JitterPct"
	mmtOutDone     = "Done"
)

func (MouseMoveTo) Spec() node.Spec {
	return node.Spec{
		Kind:        "MouseMoveTo",
		Category:    "Input",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: mmtInExec, Type: "Exec"},
			{Name: mmtInPoint, Type: "Point", Default: node.Point{X: 0.5, Y: 0.5},
				Schema: node.PointSchema()},
			{Name: mmtInMoveMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: mmtInJitterPct, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: mmtOutDone, Type: "Exec"},
		},
	}
}

func (MouseMoveTo) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	x, y, err := node.ResolvePoint(ctx, in.Point(mmtInPoint))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "MouseMoveTo resolve point: %v", err)
	}
	moveMs := node.JitterInt(in.Int(mmtInMoveMs), in.Int(mmtInJitterPct)) // ±% 抖滑行时长 (pct=0 → 原值)
	if err := moveCursor(ctx, x, y, moveMs); err != nil {
		return nil, err
	}
	return ctx.Out(mmtOutDone).Fire(), nil
}

// moveCursor 把光标滑到客户区比例 (xR,yR), 耗时 durMs. MouseMoveTo 与 ClickAt(MoveMs>0) 共用.
//
// 刻意放在本文件而非单独 helpers.go: input 包约定「1 文件 1 节点无共享 helper」, 但分帧+ctx
// 取消逻辑非平凡且被 2 节点用, DRY 优先共用. 由「主」节点 MouseMoveTo 持有.
//
// durMs<=0 → 单帧瞬移 (MoveTo 同时发 hover WM_MOUSEMOVE). >0 → 从当前光标位置直线分帧 (~16ms/帧),
// 帧间 select ctx.Done 可被 停止/强停 立即打断 (跟 Sleep/KeyPress 同范式, 不重蹈裸 sleep 停不下).
// 直线轨迹; 贝塞尔/随机控制点留后续拟人化批次 (D1.2).
func moveCursor(ctx node.Ctx, xR, yR float64, durMs int) error {
	if durMs <= 0 {
		if err := ctx.Input().MoveTo(xR, yR); err != nil {
			return node.Failf(node.CodeSendFailed, err, "move cursor: %v", err)
		}
		return nil
	}
	sx, sy, err := ctx.Input().CursorRatio()
	if err != nil {
		// 拿不到起点 (client rect 为空等) → 退化为瞬移, 别卡死.
		if err := ctx.Input().MoveTo(xR, yR); err != nil {
			return node.Failf(node.CodeSendFailed, err, "move cursor: %v", err)
		}
		return nil
	}
	frames := durMs / 16
	if frames < 1 {
		frames = 1
	}
	frameDur := time.Duration(durMs) * time.Millisecond / time.Duration(frames)
	for i := 1; i <= frames; i++ {
		t := float64(i) / float64(frames)
		if err := ctx.Input().MoveTo(sx+t*(xR-sx), sy+t*(yR-sy)); err != nil {
			return node.Failf(node.CodeSendFailed, err, "move cursor: %v", err)
		}
		select {
		case <-ctx.Context().Done():
			return ctx.Context().Err()
		case <-time.After(frameDur):
		}
	}
	return nil
}
