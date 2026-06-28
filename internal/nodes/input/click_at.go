package input

import (
	"encoding/json"
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&ClickAt{}) }

// ClickAt 在客户区坐标 pt 处点击鼠标按键, 按下时长 durationMs.
// Point 支持 ratio(0-1) 与 px 两种单位; px 由 ResolvePoint 换算为比例.
// 可选组合键 Keys (e.g. "ctrl+shift") 和连点次数 ClickCount (默认 1).
type ClickAt struct{}

const (
	caInExec       = "In"
	caInPoint      = "Point"
	caInButton     = "Button"
	caInMoveMs     = "MoveMs"
	caInDurationMs = "DurationMs"
	caInJitterPct  = "JitterPct"
	caInKeys       = "Keys"
	caInClickCount = "ClickCount"
	caOutDone      = "Done"
)

func (ClickAt) Spec() node.Spec {
	return node.Spec{
		Kind:            "ClickAt",
		Category:        "Input",
		NeedsTarget:     true,
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: caInExec, Type: "Exec"},
			{Name: caInPoint, Type: "Point", Default: node.Point{X: 0.5, Y: 0.5},
				Schema: node.PointSchema()},
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
			{Name: caInKeys, Type: "String", Default: "", Advanced: true,
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: caInClickCount, Type: "Integer", Default: json.Number("1"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: caOutDone, Type: "Exec"},
		},
	}
}

func (ClickAt) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	pt := in.Point(caInPoint)
	x, y, err := node.ResolvePoint(ctx, pt)
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "ClickAt resolve point: %v", err)
	}
	btn := in.String(caInButton)
	if btn == "" {
		btn = "left"
	}
	dur := in.Int(caInDurationMs)
	if dur <= 0 {
		dur = 50
	}
	dur = node.JitterInt(dur, in.Int(caInJitterPct))
	if err := moveCursor(ctx, x, y, in.Int(caInMoveMs)); err != nil {
		return nil, err
	}
	modKeys := in.String(caInKeys)
	clickCount := in.Int(caInClickCount)
	if clickCount < 1 {
		clickCount = 1
	}
	if err := node.ClickWithMods(ctx, node.Point{X: x, Y: y}, btn, modKeys, clickCount, dur); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "ClickAt (%.3f,%.3f) %s: %v", x, y, btn, err)
	}
	return ctx.Out(caOutDone).Fire(), nil
}

func (ClickAt) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	btn := in.String(caInButton)
	if btn != "" && btn != "left" && btn != "right" && btn != "middle" {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_MOUSE_BUTTON",
			Message: fmt.Sprintf("button %q not in left/right/middle", btn),
			Field:   caInButton,
		})
	}
	if _, ok := node.ParseMods(in.String(caInKeys)); !ok {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_MODIFIER_KEY",
			Message: "Keys 含非法修饰键 (仅 ctrl/shift/alt/win)",
			Field:   caInKeys,
		})
	}
	if in.Int(caInClickCount) < 1 {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_CLICK_COUNT",
			Message: "ClickCount 必须 >= 1",
			Field:   caInClickCount,
		})
	}
	return errs
}
