// Win32WindowTarget — 运行时解析 title/class/processName 切换当前活动窗口; 普通 exec 节点,
// 可放多个切换不同窗口. 有 exec-in "In" 和 exec-out "Done".
package system

import (
	"errors"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/pkg/winutil"
)

func init() { node.Register(&Win32WindowTarget{}) }

type Win32WindowTarget struct{}

const (
	wtInExec        = "In"
	wtInTitle       = "Title"
	wtInClass       = "Class"
	wtInProcessName = "ProcessName"
	wtInTitleMatch  = "TitleMatch"
	wtOutDone       = "Done"
)

func (Win32WindowTarget) Spec() node.Spec {
	return node.Spec{
		Kind:                "Win32WindowTarget",
		Category:            "Target",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityWindow},
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
			{Name: wtOutDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
			{Name: "Fail", Type: "Exec", Semantic: "error",
				Data: []node.DataField{
					{Name: "Error", Type: "String"},
					{Name: "Code", Type: "String"},
				}},
		},
	}
}

func (Win32WindowTarget) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if err := ctx.Window().SetActive(ctx.Context(),
		in.String(wtInTitle), in.String(wtInClass), in.String(wtInProcessName), in.String(wtInTitleMatch)); err != nil {
		if errors.Is(err, winutil.ErrWindowNotFound) {
			return nil, node.Failf(node.CodeNotFound, err, "Win32WindowTarget: %v", err)
		}
		return nil, err
	}
	w, err := ctx.Window().Snapshot()
	if err != nil {
		return nil, err
	}
	return ctx.Out(wtOutDone).Set("Window", w).Fire(), nil
}
