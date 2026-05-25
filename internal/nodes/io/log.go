// Package io IO 节点 (Log / Toast / PlayClip).
package io

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Log{}) }

type Log struct{}

const (
	logInExec    = "in"
	logInMessage = "Message"
	logInLevel   = "Level"
	logOutDone   = "Done"
)

func (Log) Spec() node.Spec {
	return node.Spec{
		Kind:        "Log",
		Category:    "IO",
		DisplayName: "日志",
		Description: "写一条日志到 framework LogService. Message 接 wildcard (任意类型, 自动 fmt.Sprint).",
		Inputs: []node.InputSpec{
			{Name: logInExec, Type: "Exec"},
			{Name: logInMessage, Type: "*", Required: true,
				DisplayName: "消息",
				Doc:         "任意类型 — 字符串 / 数字 / Point / Rect 等, framework 自动 stringify",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: logInLevel, Type: "String", Default: "info",
				DisplayName: "级别",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "debug", Label: "Debug"},
							{Value: "info", Label: "Info"},
							{Value: "warn", Label: "Warn"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: logOutDone, Type: "Exec", DisplayName: "完成"},
		},
	}
}

func (Log) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	msg := fmt.Sprint(in.Raw(logInMessage))
	level := in.String(logInLevel)
	switch level {
	case "debug":
		ctx.Log().Debug("%s", msg)
	case "warn":
		ctx.Log().Warn("%s", msg)
	default:
		ctx.Log().Info("%s", msg)
	}
	return ctx.Out(logOutDone).Fire(), nil
}

func (Log) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("[%s] %s", in.String(logInLevel), fmt.Sprint(in.Raw(logInMessage)))
}
