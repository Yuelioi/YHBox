// If — boolean branch.
// Condition true → True 出口; false → False 出口.
package control

import (
	"yotta/internal/node"
)

func init() { node.Register(&If{}) }

type If struct{}

const (
	ifInExec   = "In"
	ifInCond   = "Condition"
	ifOutTrue  = "True"
	ifOutFalse = "False"
)

func (If) Spec() node.Spec {
	return node.Spec{
		Kind:     "If",
		Category: "Control",
		Inputs: []node.InputSpec{
			{Name: ifInExec, Type: "Exec"},
			{Name: ifInCond, Type: "Bool", Default: true,
				Widget: node.WidgetSpec{Kind: "checkbox"}},
		},
		Outputs: []node.OutputSpec{
			{Name: ifOutTrue, Type: "Exec"},
			{Name: ifOutFalse, Type: "Exec"},
		},
	}
}

func (If) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if in.Bool(ifInCond) {
		return ctx.Out(ifOutTrue).Fire(), nil
	}
	return ctx.Out(ifOutFalse).Fire(), nil
}
