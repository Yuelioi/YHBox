// internal/nodes/input/swipe.go
// Swipe — 从 Begin 拖到 End (按住 Button, 历时 DurationMs 毫秒).
// pins: In, Begin(Point), End(Point), DurationMs(Number,200), Button(下拉 left/right/middle)
// out: Done。NeedsWindow。Begin/End 为 Point pin，可从检测节点(ClickTemplate/DetectColor 等)输出直连。
package input

import (
	"encoding/json"
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&Swipe{}) }

// Swipe 按住鼠标从起点拖到终点。
type Swipe struct{}

const (
	swInExec       = "In"
	swInBegin      = "Begin"
	swInEnd        = "End"
	swInButton     = "Button"
	swInDurationMs = "DurationMs"
	swOutDone      = "Done"
)

func (Swipe) Spec() node.Spec {
	return node.Spec{
		Kind:        "Swipe",
		Category:    "Input",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: swInExec, Type: "Exec"},
			{Name: swInBegin, Type: "Point"},
			{Name: swInEnd, Type: "Point"},
			{Name: swInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
			{Name: swInDurationMs, Type: "Number", Default: json.Number("200"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swOutDone, Type: "Exec"},
		},
	}
}

func (Swipe) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	begin := in.Point(swInBegin)
	end := in.Point(swInEnd)
	btn := in.String(swInButton)
	if btn == "" {
		btn = "left"
	}
	dur := in.Int(swInDurationMs)
	if dur <= 0 {
		dur = 200
	}
	if err := ctx.Input().Drag(begin.X, begin.Y, end.X, end.Y, btn, dur); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Swipe (%.3f,%.3f)→(%.3f,%.3f) %s: %v", begin.X, begin.Y, end.X, end.Y, btn, err)
	}
	return ctx.Out(swOutDone).Fire(), nil
}

func (Swipe) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	btn := in.String(swInButton)
	if btn != "" && btn != "left" && btn != "right" && btn != "middle" {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_MOUSE_BUTTON",
			Message: fmt.Sprintf("button %q not in left/right/middle", btn),
			Field:   swInButton,
		})
	}
	return errs
}
