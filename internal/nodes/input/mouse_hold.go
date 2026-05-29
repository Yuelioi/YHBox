package input

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() {
	node.Register(&MouseHoldStart{})
	node.Register(&MouseHoldStop{})
}

// MouseHoldStart 在 (xRatio, yRatio) 客户区坐标按下鼠标 (不松开). 配对 MouseHoldStop.
type MouseHoldStart struct{}

const (
	mhStartInExec   = "In"
	mhStartInXRatio = "XRatio"
	mhStartInYRatio = "YRatio"
	mhStartInButton = "Button"
	mhStartOutOut   = "Done"
)

func (MouseHoldStart) Spec() node.Spec {
	return node.Spec{
		Kind:     "MouseHoldStart",
		Category: "Input",
		Inputs: []node.InputSpec{
			{Name: mhStartInExec, Type: "Exec"},
			{Name: mhStartInXRatio, Type: "Number", Default: json.Number("0.5"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: mhStartInYRatio, Type: "Number", Default: json.Number("0.5"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: mhStartInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: mhStartOutOut, Type: "Exec"},
		},
	}
}

func (MouseHoldStart) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	btn := in.String(mhStartInButton)
	if btn == "" {
		btn = "left"
	}
	// 老 runtime 在 Run 内对 button 二次校验 (Validate 已覆盖, 但保留显式 err 防 wire 跳过 Validate)
	if btn != "left" && btn != "right" && btn != "middle" {
		return nil, fmt.Errorf("MouseHoldStart: invalid button %q", btn)
	}
	x := in.Float64(mhStartInXRatio)
	y := in.Float64(mhStartInYRatio)
	if err := ctx.Input().MouseDown(x, y, btn); err != nil {
		return nil, fmt.Errorf("MouseHoldStart (%.3f,%.3f) %s: %w", x, y, btn, err)
	}
	return ctx.Out(mhStartOutOut).Fire(), nil
}

func (MouseHoldStart) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("mouse ▼ %s @ (%.2f,%.2f)",
		in.String(mhStartInButton), in.Float64(mhStartInXRatio), in.Float64(mhStartInYRatio))
}

func (MouseHoldStart) Validate(in node.Inputs) []node.ValidationError {
	return validateMouseButtonField(in.String(mhStartInButton), mhStartInButton)
}

// MouseHoldStop 松开鼠标按键 (跟 MouseHoldStart 配对). 老 runtime: hold_nodes.go::execMouseHoldStop.
type MouseHoldStop struct{}

const (
	mhStopInExec   = "In"
	mhStopInButton = "Button"
	mhStopOutOut   = "Done"
)

func (MouseHoldStop) Spec() node.Spec {
	return node.Spec{
		Kind:     "MouseHoldStop",
		Category: "Input",
		Inputs: []node.InputSpec{
			{Name: mhStopInExec, Type: "Exec"},
			{Name: mhStopInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: mhStopOutOut, Type: "Exec"},
		},
	}
}

func (MouseHoldStop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	btn := in.String(mhStopInButton)
	if btn == "" {
		btn = "left"
	}
	if btn != "left" && btn != "right" && btn != "middle" {
		return nil, fmt.Errorf("MouseHoldStop: invalid button %q", btn)
	}
	if err := ctx.Input().MouseUp(btn); err != nil {
		return nil, fmt.Errorf("MouseHoldStop %s: %w", btn, err)
	}
	return ctx.Out(mhStopOutOut).Fire(), nil
}

func (MouseHoldStop) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("mouse ▲ %s", in.String(mhStopInButton))
}

func (MouseHoldStop) Validate(in node.Inputs) []node.ValidationError {
	return validateMouseButtonField(in.String(mhStopInButton), mhStopInButton)
}

// validateMouseButtonField 复刻老 validator.go::validateMouseHold:
// button 必须 in {left, right, middle}. 空字符串视为有效 (run 时 fallback "left").
//
// 本 helper 仅本文件内 Start/Stop 复用, 不算 cross-node abstraction (跟 key_hold.go::validateVKField 同理).
func validateMouseButtonField(btn, field string) []node.ValidationError {
	if btn == "" {
		return nil // empty → Run fallback "left"
	}
	if btn != "left" && btn != "right" && btn != "middle" {
		return []node.ValidationError{{
			Code:    "INVALID_MOUSE_BUTTON",
			Message: fmt.Sprintf("button %q not in left/right/middle", btn),
			Field:   field,
		}}
	}
	return nil
}
