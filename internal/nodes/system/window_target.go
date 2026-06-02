// WindowTarget — 声明式窗口目标节点. config 用 6 个顶级 String 字段描述目标
// (title/class/processName/titleMatch + inputBackend/captureBackend), widget 选 dropdown
// (titleMatch / inputBackend / captureBackend). 无 exec-in, 单 Fire exec-out 表达 passthrough.
// runtime/nodes.go::case "WindowTarget" 是 no-op — RuntimeContext 启动期由 wire_container.go
// 读这些字段解析 hwnd + 选 backend.
package system

import (
	"encoding/json"

	"yotta/internal/node"
)

func init() { node.Register(&WindowTarget{}) }

type WindowTarget struct{}

const (
	wtInTitle          = "Title"
	wtInClass          = "Class"
	wtInProcessName    = "ProcessName"
	wtInTitleMatch     = "TitleMatch"
	wtInInputBackend   = "InputBackend"
	wtInCaptureBackend = "CaptureBackend"
	wtInScaleTolerance = "ScaleTolerance"
	wtOutFire          = "Fire"
)

func (WindowTarget) Spec() node.Spec {
	return node.Spec{
		Kind:     "WindowTarget",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: wtInTitle, Type: "String", Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wtInClass, Type: "String", Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wtInProcessName, Type: "String", Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wtInTitleMatch, Type: "String", Default: "exact",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "exact"},
							{Value: "contains"},
							{Value: "prefix"},
							{Value: "suffix"},
							{Value: "regex"},
						}})}},
			{Name: wtInInputBackend, Type: "String", Default: "postmessage",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "postmessage"},
							{Value: "sendinput"},
						}})}},
			{Name: wtInCaptureBackend, Type: "String", Default: "auto",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto"},
							{Value: "gdi"},
							{Value: "wgc"},
							{Value: "mock"},
						}})}},
			{Name: wtInScaleTolerance, Type: "Number", Default: json.Number("2.0"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 1, Max: 4, Step: 0.1})}},
		},
		Outputs: []node.OutputSpec{
			{Name: wtOutFire, Type: "Exec"},
		},
	}
}

func (WindowTarget) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// declarative node — runtime 已在启动期读 title/match config 解析 hwnd, 这里 no-op.
	return ctx.Out(wtOutFire).Fire(), nil
}
