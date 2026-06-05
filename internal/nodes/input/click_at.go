package input

import (
	"encoding/json"
	"fmt"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&ClickAt{}) }

// ClickAt 在客户区坐标 (xRatio, yRatio) 单击鼠标按键, 按下时长 durationMs.
type ClickAt struct{}

const (
	caInExec       = "In"
	caInXRatio     = "XRatio"
	caInYRatio     = "YRatio"
	caInButton     = "Button"
	caInMoveMs     = "MoveMs"
	caInDurationMs = "DurationMs"
	caInJitterPct  = "JitterPct"
	caOutDone      = "Done"
)

func (ClickAt) Spec() node.Spec {
	return node.Spec{
		Kind:        "ClickAt",
		Category:    "Input",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: caInExec, Type: "Exec"},
			{Name: caInXRatio, Type: "Number", Default: json.Number("0.5"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: caInYRatio, Type: "Number", Default: json.Number("0.5"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: caInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
			{Name: caInMoveMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: caInDurationMs, Type: "Number", Default: json.Number("50"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: caInJitterPct, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: caOutDone, Type: "Exec"},
		},
	}
}

func (ClickAt) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	x := in.Float64(caInXRatio)
	y := in.Float64(caInYRatio)
	btn := in.String(caInButton)
	if btn == "" {
		btn = "left"
	}
	dur := in.Int(caInDurationMs)
	if dur <= 0 {
		dur = 50
	}
	dur = node.JitterInt(dur, in.Int(caInJitterPct)) // ±% 近正态抖动 (pct=0 → 原值)
	// 先 (可选) 滑到目标 + 发 hover, 再按下. MoveMs=0 → 单帧瞬移 hover (恢复 #4 丢掉的 hover).
	// moveCursor 可被 ctx 取消; 此时还没按下, 直接返回无需释放.
	if err := moveCursor(ctx, x, y, in.Int(caInMoveMs)); err != nil {
		return nil, err
	}
	// 拆 down→可取消 sleep→up: cancel 时立即松键返回 (修长按停不下)。
	if err := ctx.Input().MouseDown(x, y, btn); err != nil {
		return nil, fmt.Errorf("ClickAt down (%.3f,%.3f) %s: %w", x, y, btn, err)
	}
	select {
	case <-ctx.Context().Done():
		_ = ctx.Input().MouseUp(btn)
		return nil, ctx.Context().Err()
	case <-time.After(time.Duration(dur) * time.Millisecond):
	}
	if err := ctx.Input().MouseUp(btn); err != nil {
		return nil, fmt.Errorf("ClickAt up %s: %w", btn, err)
	}
	return ctx.Out(caOutDone).Fire(), nil
}

func (ClickAt) Validate(in node.Inputs) []node.ValidationError {
	btn := in.String(caInButton)
	if btn != "" && btn != "left" && btn != "right" && btn != "middle" {
		return []node.ValidationError{{
			Code:    "INVALID_MOUSE_BUTTON",
			Message: fmt.Sprintf("button %q not in left/right/middle", btn),
			Field:   caInButton,
		}}
	}
	return nil
}
