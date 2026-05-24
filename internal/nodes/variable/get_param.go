// internal/nodes/variable/get_param.go
// GetParam pure-data — pull-based evaluator reads subgraph input param.
// Phase 4 stub; Phase 5 subgraph runtime impl wires this.
package variable

import "yhbox/internal/node"

func init() { node.Register(&GetParam{}) }

type GetParam struct{}

const (
	gpInParamName = "paramName"
	gpOutValue    = "value"
)

func (GetParam) Spec() node.Spec {
	return node.Spec{
		Kind:        "GetParam",
		Version:     1,
		Category:    "Variable",
		DisplayName: "读子图参数",
		Description: "pure-data 节点 — 读 Subgraph 调用时传入的 input param. 仅子图内有效.",
		Inputs: []node.InputSpec{
			{Name: gpInParamName, Type: "String", Required: true,
				DisplayName: "参数名",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: gpOutValue, Type: "*", DisplayName: "值"},
		},
		IsPureData: true,
	}
}

func (GetParam) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errPureDataNotEvaluatable
}
