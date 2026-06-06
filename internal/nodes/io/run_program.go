package io

import (
	"fmt"

	"yotta/internal/node"
	"yotta/pkg/platform"
)

func init() { node.Register(&RunProgram{}) }

// RunProgram 用系统默认关联程序打开 Target —— 可执行文件直接运行、URL 走默认浏览器、
// 文档/文件夹走默认程序。启动即走 (fire-and-forget)，不等程序退出。
type RunProgram struct{}

const (
	rpInExec    = "In"
	rpInTarget  = "Target"
	rpInArgs    = "Args"
	rpInWorkDir = "WorkingDir"
	rpInWindow  = "WindowState"
	rpOutDone   = "Done"
)

func (RunProgram) Spec() node.Spec {
	return node.Spec{
		Kind:     "RunProgram",
		Category: "IO",
		Inputs: []node.InputSpec{
			{Name: rpInExec, Type: node.TypeExec},
			{Name: rpInTarget, Type: "String", Required: true, Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: rpInArgs, Type: "String", Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: rpInWorkDir, Type: "String", Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: rpInWindow, Type: "String", Default: "normal",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "normal"},
							{Value: "minimized"},
							{Value: "maximized"},
							{Value: "hidden"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: rpOutDone, Type: node.TypeExec},
		},
	}
}

func (RunProgram) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	target := in.String(rpInTarget)
	if target == "" {
		return nil, fmt.Errorf("RunProgram: Target 不能为空")
	}

	showCmd := platform.SWShowNormal
	switch in.String(rpInWindow) {
	case "minimized":
		showCmd = platform.SWShowMinNoActive
	case "maximized":
		showCmd = platform.SWShowMaximized
	case "hidden":
		showCmd = platform.SWHide
	}

	if err := platform.ShellOpen(target, in.String(rpInArgs), in.String(rpInWorkDir), showCmd); err != nil {
		return nil, fmt.Errorf("RunProgram 启动 %q 失败: %w", target, err)
	}
	return ctx.Out(rpOutDone).Fire(), nil
}
