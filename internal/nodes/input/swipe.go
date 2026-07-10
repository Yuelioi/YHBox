// internal/nodes/input/swipe.go
// Swipe — 从 Begin 拖到 End (按住 Button, 历时 DurationMs 毫秒).
// pins: In, Begin(Point), End(Point), DurationMs(Number,200), Button(下拉 left/right/middle)
// out: Done。NeedsTarget。Begin/End 为 Point pin，可从检测节点(ClickTemplate/DetectColor 等)输出直连。
package input

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/node"
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
		NeedsTarget: true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityDrag,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: swInExec, Type: "Exec"},
			{Name: swInBegin, Type: "Point", Schema: node.PointSchema()},
			{Name: swInEnd, Type: "Point", Schema: node.PointSchema()},
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
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: swOutDone, Type: "Exec"},
		},
	}
}

func (Swipe) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	bx, by, err := node.ResolvePoint(ctx, in.Point(swInBegin))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Swipe resolve begin: %v", err)
	}
	ex, ey, err := node.ResolvePoint(ctx, in.Point(swInEnd))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Swipe resolve end: %v", err)
	}
	btn := in.String(swInButton)
	if btn == "" {
		btn = "left"
	}
	dur := in.Int(swInDurationMs)
	if err := ctx.Input().Drag(bx, by, ex, ey, btn, dur); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Swipe drag: %v", err)
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
