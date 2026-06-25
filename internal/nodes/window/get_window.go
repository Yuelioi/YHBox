package window

import (
	"errors"
	"time"

	"yotta/internal/node"
	"yotta/pkg/winutil"
)

func init() { node.Register(&GetWindow{}) }

// resolveWindowFn 测试可替换; 默认真 Win32 解析。
var resolveWindowFn = winutil.ResolveWindow

const (
	selInTitle      = "Title"
	selInClass      = "Class"
	selInProcess    = "ProcessName"
	selInTitleMatch = "TitleMatch"
)

// windowSelectorInputs — 窗口匹配输入。与 system.WindowTarget 的选择器口径一致(防漂移);
// 后续同包控制节点不需要选择器(它们靠活动窗口/Window 输入), 故此 helper 暂仅 GetWindow 用。
func windowSelectorInputs() []node.InputSpec {
	return []node.InputSpec{
		{Name: selInTitle, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: selInClass, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: selInProcess, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: selInTitleMatch, Type: "String", Default: "exact",
			Widget: node.WidgetSpec{Kind: "dropdown", Props: node.MarshalProps(node.DropdownProps{
				Options: []node.EnumOption{{Value: "exact"}, {Value: "contains"}, {Value: "prefix"}, {Value: "suffix"}, {Value: "regex"}},
			})}},
	}
}

func matchSpecFrom(in node.Inputs) winutil.MatchSpec {
	return winutil.MatchSpec{
		Title:       in.String(selInTitle),
		Class:       in.String(selInClass),
		ProcessName: in.String(selInProcess),
		TitleMatch:  in.String(selInTitleMatch),
	}
}

func windowFromHandle(wh winutil.WindowHandle) node.Window {
	return node.Window{
		HWND: wh.HWND, Title: wh.Title, Class: wh.Class,
		Process: wh.ProcessName, PID: wh.PID, ClientW: wh.ClientW, ClientH: wh.ClientH,
	}
}

// GetWindow 解析匹配窗口为 Window 对象, 不改活动窗口。
type GetWindow struct{}

const (
	gwInExec = "In"
	gwDone   = "Done"
	gwFail   = "Fail"
)

func (GetWindow) Spec() node.Spec {
	return node.Spec{
		Kind:     "GetWindow",
		Category: "Window",
		Inputs:   append([]node.InputSpec{{Name: gwInExec, Type: "Exec"}}, windowSelectorInputs()...),
		Outputs: []node.OutputSpec{
			{Name: gwDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
			{Name: gwFail, Type: "Exec", Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"}, {Name: "Code", Type: "String"}}},
		},
	}
}

func (GetWindow) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	wh, err := resolveWindowFn(ctx.Context(), matchSpecFrom(in), 3*time.Second, 500*time.Millisecond)
	if err != nil {
		if errors.Is(err, winutil.ErrWindowNotFound) {
			return nil, node.Failf(node.CodeNotFound, err, "GetWindow: %v", err)
		}
		return nil, err
	}
	return ctx.Out(gwDone).Set("Window", windowFromHandle(wh)).Fire(), nil
}
