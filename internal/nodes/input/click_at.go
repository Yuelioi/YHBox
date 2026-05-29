package input

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&ClickAt{}) }

// ClickAt 在客户区坐标 (xRatio, yRatio) 单击鼠标按键, 按下时长 durationMs.
type ClickAt struct{}

const (
	caInExec       = "In"
	caInXRatio     = "XRatio"
	caInYRatio     = "YRatio"
	caInButton     = "Button"
	caInDurationMs = "DurationMs"
	caOutDone      = "Done"
)

func (ClickAt) Spec() node.Spec {
	return node.Spec{
		Kind:     "ClickAt",
		Category: "Input",
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
			{Name: caInDurationMs, Type: "Number", Default: json.Number("50"),
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
	if err := ctx.Input().Click(x, y, btn, dur); err != nil {
		return nil, fmt.Errorf("ClickAt (%.3f,%.3f) %s: %w", x, y, btn, err)
	}
	return ctx.Out(caOutDone).Fire(), nil
}

func (ClickAt) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("click %s @ (%.2f,%.2f)", in.String(caInButton),
		in.Float64(caInXRatio), in.Float64(caInYRatio))
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
