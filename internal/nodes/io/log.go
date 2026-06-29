// Package io IO 节点 (Log / Toast / PlayClip).
package io

import (
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&Log{}) }

type Log struct{}

const (
	logInExec    = "In"
	logInMessage = "Message"
	logInLevel   = "Level"
	logOutDone   = "Done"
)

func (Log) Spec() node.Spec {
	return node.Spec{
		Kind:     "Log",
		Category: "IO",
		Inputs: []node.InputSpec{
			{Name: logInExec, Type: "Exec"},
			{Name: logInMessage, Type: "*", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: logInLevel, Type: "String", Default: "info",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "debug"},
							{Value: "info"},
							{Value: "warn"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: logOutDone, Type: "Exec"},
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
