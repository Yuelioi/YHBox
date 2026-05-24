// internal/nodes/io/toast.go
// Toast — 弹 GUI toast (frontend 提示). 老 runtime: nodes.go::execToast emit
// "container:toast" wails event. 新框架 Phase 4 用 ctx.Log().Info() 兜底
// (FE 见不到, 但有 trace), Phase 5 wire 时 main.go 注入真 emit-toast service
// (或扩 LogService 加 Toast(level, title, message) 方法).
package io

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Toast{}) }

type Toast struct{}

const (
	toastInExec    = "in"
	toastInTitle   = "Title"
	toastInMessage = "Message"
	toastInColor   = "Color"
	toastOutDone   = "Done"
)

func (Toast) Spec() node.Spec {
	return node.Spec{
		Kind:        "Toast",
		Version:     1,
		Category:    "IO",
		DisplayName: "Toast",
		Description: "弹 GUI toast 提示. Phase 4 兜底走 LogService.Info; Phase 5 wire 真 emit 到 frontend.",
		Inputs: []node.InputSpec{
			{Name: toastInExec, Type: "Exec"},
			{Name: toastInTitle, Type: "*", Default: "提示",
				DisplayName: "标题",
				Doc:         "任意类型, 自动 stringify",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: toastInMessage, Type: "*", Required: true,
				DisplayName: "消息",
				Doc:         "任意类型, 自动 stringify",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: toastInColor, Type: "String", Default: "primary",
				DisplayName: "颜色",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "primary", Label: "Primary"},
							{Value: "success", Label: "Success"},
							{Value: "warning", Label: "Warning"},
							{Value: "danger", Label: "Danger"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: toastOutDone, Type: "Exec", DisplayName: "完成"},
		},
	}
}

func (Toast) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	title := fmt.Sprint(in.Raw(toastInTitle))
	msg := fmt.Sprint(in.Raw(toastInMessage))
	color := in.String(toastInColor)
	// Phase 4: log fallback. Phase 5: wire real wails event emit.
	ctx.Log().Info("[toast/%s] %s: %s", color, title, msg)
	return ctx.Out(toastOutDone).Fire(), nil
}

func (Toast) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("toast %s: %s", fmt.Sprint(in.Raw(toastInTitle)), fmt.Sprint(in.Raw(toastInMessage)))
}
