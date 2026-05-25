// internal/nodes/variable/get_var.go
// GetVar pure-data — pull-based evaluator reads variable. Phase 4 stub Run returns sentinel.
// Phase 5 wire 加 pull-eval, GetVar.Run 永不调.
package variable

import "yhbox/internal/node"

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

func (GetVar) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, node.ErrPureDataMustEvaluate
}
