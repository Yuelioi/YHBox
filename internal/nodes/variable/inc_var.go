// internal/nodes/variable/inc_var.go
// IncVar — 按 scope=auto/local/global 给容器变量加一个数.
package variable

import (
	"encoding/json"
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&IncVar{}) }

type IncVar struct{}

const (
	ivInExec    = "In"
	ivInVarName = "VarName"
	ivInScope   = "Scope"
	ivInDelta   = "Delta"
	ivOutOut    = "Done"
)

func (IncVar) Spec() node.Spec {
	return node.Spec{
		Kind:     "IncVar",
		Category: "Variable",
		Inputs: []node.InputSpec{
			{Name: ivInExec, Type: "Exec"},
			{Name: ivInVarName, Type: "String", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: ivInScope, Type: "String", Default: "auto",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto"},
							{Value: "local"},
							{Value: "global"},
						}})}},
			{Name: ivInDelta, Type: "Number", Default: json.Number("1"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: ivOutOut, Type: "Exec"},
		},
	}
}

func (IncVar) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	name := in.String(ivInVarName)
	if name == "" {
		return nil, fmt.Errorf("IncVar: missing varName")
	}
	delta := in.Float64(ivInDelta)
	scope := in.String(ivInScope)
	if scope == "" {
		scope = "auto"
	}
	ctx.Vars().IncScoped(name, scope, delta)
	return ctx.Out(ivOutOut).Fire(), nil
}

func (IncVar) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("%s += %v", in.String(ivInVarName), in.Float64(ivInDelta))
}
