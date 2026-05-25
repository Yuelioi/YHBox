// internal/nodes/variable/inc_var.go
// IncVar — 给容器变量加一个值. 老 runtime nodes.go::execIncVar 1:1.
package variable

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&IncVar{}) }

type IncVar struct{}

const (
	ivInExec    = "in"
	ivInVarName = "VarName"
	ivInScope   = "Scope"
	ivInDelta   = "Delta"
	ivOutOut    = "out"
)

func (IncVar) Spec() node.Spec {
	return node.Spec{
		Kind:        "IncVar",
		Category:    "Variable",
		DisplayName: "变量自增",
		Description: "给容器变量加 Delta (默认 1). scope=auto/local/global. 老 runtime 同款.",
		Inputs: []node.InputSpec{
			{Name: ivInExec, Type: "Exec"},
			{Name: ivInVarName, Type: "String", Required: true,
				DisplayName: "变量名",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: ivInScope, Type: "String", Default: "auto",
				DisplayName: "作用域",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto", Label: "auto"},
							{Value: "local", Label: "local"},
							{Value: "global", Label: "global"},
						}})}},
			{Name: ivInDelta, Type: "Number", Default: json.Number("1"),
				DisplayName: "增量",
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: ivOutOut, Type: "Exec", DisplayName: "完成"},
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
