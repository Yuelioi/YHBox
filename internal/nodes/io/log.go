// Package io contains legacy IO nodes.
package io

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/node"
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
		Kind:                "Log",
		Category:            "IO",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityLog},
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
		ctx.Services().Log.Debug("%s", msg)
	case "warn":
		ctx.Services().Log.Warn("%s", msg)
	default:
		ctx.Services().Log.Info("%s", msg)
	}
	return ctx.Out(logOutDone).Fire(), nil
}
