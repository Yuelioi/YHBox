// internal/nodes/io/toast.go
// Toast — 弹 GUI toast (frontend 提示). 当前无 toast service, 退化为
// ctx.Log().Info() — FE 见不到, 但有 trace.
package io

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Toast{}) }

type Toast struct{}

const (
	toastInExec    = "In"
	toastInTitle   = "Title"
	toastInMessage = "Message"
	toastInColor   = "Color"
	toastOutDone   = "Done"
)

func (Toast) Spec() node.Spec {
	return node.Spec{
		Kind:     "Toast",
		Category: "IO",
		Inputs: []node.InputSpec{
			{Name: toastInExec, Type: "Exec"},
			{Name: toastInTitle, Type: "*", Default: "提示",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: toastInMessage, Type: "*", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: toastInColor, Type: "String", Default: "primary",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "primary"},
							{Value: "success"},
							{Value: "warning"},
							{Value: "danger"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: toastOutDone, Type: "Exec"},
		},
	}
}

func (Toast) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	title := fmt.Sprint(in.Raw(toastInTitle))
	msg := fmt.Sprint(in.Raw(toastInMessage))
	color := in.String(toastInColor)
	ctx.Log().Info("[toast/%s] %s: %s", color, title, msg)
	return ctx.Out(toastOutDone).Fire(), nil
}

func (Toast) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("toast %s: %s", fmt.Sprint(in.Raw(toastInTitle)), fmt.Sprint(in.Raw(toastInMessage)))
}
