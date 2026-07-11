package input

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/node"
)

func init() {
	node.Register(&MouseHoldStart{})
	node.Register(&MouseHoldStop{})
}

// MouseHoldStart 在坐标位置按下鼠标 (不松开). 配对 MouseHoldStop.
type MouseHoldStart struct{}

const (
	mhStartInExec   = "In"
	mhStartInPoint  = "Point"
	mhStartInButton = "Button"
	mhStartOutOut   = "Done"
)

func (MouseHoldStart) Spec() node.Spec {
	return node.Spec{
		Kind:                "MouseHoldStart",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityInput, node.RuntimeCapabilityWindow},
		Category:            "Input",
		NeedsTarget:         true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityMouseButton,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: mhStartInExec, Type: "Exec"},
			{Name: mhStartInPoint, Type: "Point", Default: node.Point{X: 0.5, Y: 0.5},
				Schema: node.PointSchema()},
			{Name: mhStartInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
		}, node.WindowInputSpec()),
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
	// Run 内对 button 二次校验 (Validate 已覆盖, 但显式 err 防 wire 跳过 Validate)
	if btn != "left" && btn != "right" && btn != "middle" {
		return nil, fmt.Errorf("MouseHoldStart: invalid button %q", btn)
	}
	x, y, err := node.ResolvePoint(ctx, in.Point(mhStartInPoint))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "MouseHoldStart resolve point: %v", err)
	}
	if err := ctx.Services().Input.MouseDown(x, y, btn); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "MouseHoldStart (%.3f,%.3f) %s: %v", x, y, btn, err)
	}
	return ctx.Out(mhStartOutOut).Fire(), nil
}

func (MouseHoldStart) Validate(in node.Inputs) []node.ValidationError {
	return validateMouseButtonField(in.String(mhStartInButton), mhStartInButton)
}

// MouseHoldStop 松开鼠标按键 (跟 MouseHoldStart 配对).
type MouseHoldStop struct{}

const (
	mhStopInExec   = "In"
	mhStopInButton = "Button"
	mhStopOutOut   = "Done"
)

func (MouseHoldStop) Spec() node.Spec {
	return node.Spec{
		Kind:                "MouseHoldStop",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityInput},
		Category:            "Input",
		NeedsTarget:         true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityMouseButton,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: mhStopInExec, Type: "Exec"},
			{Name: mhStopInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
		}, node.WindowInputSpec()),
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
	if err := ctx.Services().Input.MouseUp(btn); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "MouseHoldStop %s: %v", btn, err)
	}
	return ctx.Out(mhStopOutOut).Fire(), nil
}

func (MouseHoldStop) Validate(in node.Inputs) []node.ValidationError {
	return validateMouseButtonField(in.String(mhStopInButton), mhStopInButton)
}

// validateMouseButtonField:
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
