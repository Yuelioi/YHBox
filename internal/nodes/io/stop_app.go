package io

import (
	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/pkg/platform"
)

func init() { node.Register(&StopApp{}) }

// StopApp 按进程名、exe 路径或 PID 强制结束进程。
type StopApp struct{}

// killProcess 可注入，测试时替换为 mock。
var killProcess = platform.KillProcess

const (
	saInExec   = "In"
	saInTarget = "Target"
	saOutDone  = "Done"
)

func (StopApp) Spec() node.Spec {
	return node.Spec{
		Kind:            "StopApp",
		Category:        "IO",
		PlatformTargets: []string{node.SupportedTargetWin32Window},
		Inputs: []node.InputSpec{
			{Name: saInExec, Type: node.TypeExec},
			{Name: saInTarget, Type: "String", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: saOutDone, Type: node.TypeExec},
			{Name: "Fail", Type: "Exec", Semantic: "error",
				Data: []node.DataField{
					{Name: "Error", Type: "String"},
					{Name: "Code", Type: "String"},
				}},
		},
	}
}

func (StopApp) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	target := in.String(saInTarget)
	if target == "" {
		return nil, node.Failf(node.CodeError, nil, "StopApp: Target 不能为空")
	}
	if err := killProcess(target); err != nil {
		return nil, node.Failf(node.CodeError, err, "StopApp 结束进程 %q 失败: %v", target, err)
	}
	return ctx.Out(saOutDone).Fire(), nil
}
