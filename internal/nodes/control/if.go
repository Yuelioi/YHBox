// internal/nodes/control/if.go
// If — boolean branch. 老 runtime nodes.go::execIf 1:1.
// Condition true → True 出口; false → False 出口.
package control

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&If{}) }

type If struct{}

const (
	ifInExec   = "in"
	ifInCond   = "Condition"
	ifOutTrue  = "True"
	ifOutFalse = "False"
)

func (If) Spec() node.Spec {
	return node.Spec{
		Kind:        "If",
		Version:     1,
		Category:    "Control",
		DisplayName: "条件分支",
		Description: "Condition true → True 出口; false → False 出口. 老 v4 If.",
		Inputs: []node.InputSpec{
			{Name: ifInExec, Type: "Exec"},
			{Name: ifInCond, Type: "Bool", Default: true,
				DisplayName: "条件",
				Widget:      node.WidgetSpec{Kind: "checkbox"}},
		},
		Outputs: []node.OutputSpec{
			{Name: ifOutTrue, Type: "Exec", DisplayName: "True"},
			{Name: ifOutFalse, Type: "Exec", DisplayName: "False"},
		},
	}
}

func (If) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if in.Bool(ifInCond) {
		return ctx.Out(ifOutTrue).Fire(), nil
	}
	return ctx.Out(ifOutFalse).Fire(), nil
}

func (If) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("if %v → %s", in.Bool(ifInCond), exitName)
}
