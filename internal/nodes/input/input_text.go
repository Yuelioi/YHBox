// internal/nodes/input/input_text.go
// InputText — 向当前窗口注入一串文字 (unicode, 逐 rune SendInput KEYEVENTF_UNICODE).
// pins: In(Exec), Text(String, Required, text widget). out: Done.
// NeedsWindow.
package input

import "github.com/yottaapp/yotta/internal/node"

func init() { node.Register(&InputText{}) }

// InputText 向当前目标窗口输入一段文字。
type InputText struct{}

const (
	itInExec  = "In"
	itInText  = "Text"
	itOutDone = "Done"
)

func (InputText) Spec() node.Spec {
	return node.Spec{
		Kind:                "InputText",
		Category:            "Input",
		NeedsTarget:         true,
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityInput},
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityText,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: itInExec, Type: "Exec"},
			{Name: itInText, Type: "String", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: itOutDone, Type: "Exec"},
		},
	}
}

func (InputText) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	text := in.String(itInText)
	if err := ctx.Services().Input.TypeText(text); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "InputText: %v", err)
	}
	return ctx.Out(itOutDone).Fire(), nil
}

func (InputText) Validate(in node.Inputs) []node.ValidationError {
	if in.String(itInText) == "" {
		return []node.ValidationError{{
			Code:    "MISSING_TEXT",
			Message: "Text 不能为空",
			Field:   itInText,
		}}
	}
	return nil
}
