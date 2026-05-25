// internal/nodes/system/window_target.go
// WindowTarget — 声明式窗口目标节点. 老 spec (nodekind/specs/system.go) 用嵌套
// config.match{title,class,processName,titleMatch} + config.runtime{inputBackend,
// captureBackend} 描述目标; runtime/nodes.go::case "WindowTarget" 是 no-op (runtime
// 在 RuntimeContext 启动期通过 wire_container.go 读取这些字段解析 hwnd + 选 backend).
//
// 新框架 declarative 形态保持 (no exec-in), 单 Fire exec-out 表达 passthrough.
// 字段拆 6 个顶级 String 来对应原 match.*/runtime.* 子字段, widget 选 dropdown
// (titleMatch / inputBackend / captureBackend).
package system

import "yhbox/internal/node"

func init() { node.Register(&WindowTarget{}) }

type WindowTarget struct{}

const (
	wtInTitle          = "Title"
	wtInClass          = "Class"
	wtInProcessName    = "ProcessName"
	wtInTitleMatch     = "TitleMatch"
	wtInInputBackend   = "InputBackend"
	wtInCaptureBackend = "CaptureBackend"
	wtOutFire          = "Fire"
)

func (WindowTarget) Spec() node.Spec {
	return node.Spec{
		Kind:        "WindowTarget",
		Category:    "System",
		DisplayName: "目标窗口",
		Description: "声明式 — runtime 启动期读 title/class/processName 解析 hwnd, 读 inputBackend/captureBackend 选 backend. 节点本体 no-op, 走 Fire 出口.",
		Inputs: []node.InputSpec{
			{Name: wtInTitle, Type: "String", Default: "",
				DisplayName: "窗口标题",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: wtInClass, Type: "String", Default: "",
				DisplayName: "窗口类名",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: wtInProcessName, Type: "String", Default: "",
				DisplayName: "进程名",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: wtInTitleMatch, Type: "String", Default: "exact",
				DisplayName: "标题匹配方式",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "exact", Label: "exact"},
							{Value: "contains", Label: "contains"},
							{Value: "prefix", Label: "prefix"},
							{Value: "suffix", Label: "suffix"},
							{Value: "regex", Label: "regex"},
						}})}},
			{Name: wtInInputBackend, Type: "String", Default: "postmessage",
				DisplayName: "输入后端",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "postmessage", Label: "postmessage"},
							{Value: "sendinput", Label: "sendinput"},
						}})}},
			{Name: wtInCaptureBackend, Type: "String", Default: "auto",
				DisplayName: "截屏后端",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto", Label: "auto"},
							{Value: "bitblt", Label: "bitblt"},
							{Value: "wgc", Label: "wgc"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: wtOutFire, Type: "Exec", DisplayName: "Fire"},
		},
	}
}

func (WindowTarget) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// declarative node — runtime 已在启动期读 title/match config 解析 hwnd, 这里 no-op.
	return ctx.Out(wtOutFire).Fire(), nil
}
