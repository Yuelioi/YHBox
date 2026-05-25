// internal/nodes/variable/get_var.go
// GetVar pure-data — Evaluator capability. ctx.Vars() 在 EvaluatePureData 入口
// 已被 framework wrap 成 snapshot-mode (scope="global"/auto-fallback 走 frozen view).
package variable

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&GetVar{}) }

type GetVar struct{}

const (
	gvInVarName = "VarName"
	gvInScope   = "Scope"
	gvOutValue  = "Value"
)

func (GetVar) Spec() node.Spec {
	return node.Spec{
		Kind:        "GetVar",
		Category:    "Variable",
		DisplayName: "读变量",
		Description: "pure-data 节点 — 读容器变量供 data edge 下游消费. scope=auto/local/global.",
		Inputs: []node.InputSpec{
			{Name: gvInVarName, Type: "String", Required: true,
				DisplayName: "变量名",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: gvInScope, Type: "String", Default: "auto",
				DisplayName: "作用域",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto", Label: "auto"},
							{Value: "local", Label: "local"},
							{Value: "global", Label: "global"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: gvOutValue, Type: "*", DisplayName: "值"},
		},
		IsPureData: true,
	}
}

func (GetVar) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	name := in.String(gvInVarName)
	if name == "" {
		return nil, fmt.Errorf("GetVar: missing VarName")
	}
	scope := in.String(gvInScope)
	if scope == "" {
		scope = "auto"
	}
	v, _ := ctx.Vars().GetScoped(name, scope)
	return v, nil
}

