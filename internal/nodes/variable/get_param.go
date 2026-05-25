// internal/nodes/variable/get_param.go
// GetParam pure-data — Evaluator capability. ctx.Params() 读当前 frame.LocalParams
// (subgraph 入参). frame-private state, snapshot wrap 不包.
package variable

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&GetParam{}) }

type GetParam struct{}

const (
	gpInParamName = "ParamName"
	gpOutValue    = "Value"
)

func (GetParam) Spec() node.Spec {
	return node.Spec{
		Kind:        "GetParam",
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

func (GetParam) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	name := in.String(gpInParamName)
	if name == "" {
		return nil, fmt.Errorf("GetParam: missing ParamName")
	}
	v, _ := ctx.Params().Get(name)
	return v, nil
}

