// WindowTarget — 运行时解析 title/class/processName 切换当前活动窗口; 普通 exec 节点,
// 可放多个切换不同窗口. 有 exec-in "In" 和 exec-out "Fire".
package system

import (
	"yotta/internal/node"
)

func init() { node.Register(&WindowTarget{}) }

type WindowTarget struct{}

const (
	wtInExec        = "In"
	wtInTitle       = "Title"
	wtInClass       = "Class"
	wtInProcessName = "ProcessName"
	wtInTitleMatch  = "TitleMatch"
	wtOutFire       = "Fire"
)

func (WindowTarget) Spec() node.Spec {
	return node.Spec{
		Kind:     "WindowTarget",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: wtInExec, Type: "Exec"},
			{Name: wtInTitle, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wtInClass, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wtInProcessName, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
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
		},
		Outputs: []node.OutputSpec{
			{Name: wtOutFire, Type: "Exec"},
		},
	}
}

func (WindowTarget) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if err := ctx.Window().SetActive(ctx.Context(),
		in.String(wtInTitle), in.String(wtInClass), in.String(wtInProcessName), in.String(wtInTitleMatch)); err != nil {
		return nil, err
	}
	return ctx.Out(wtOutFire).Fire(), nil
}
