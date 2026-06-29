// internal/nodes/variable/get_param.go
// GetParam pure-data — Evaluator capability. ctx.Params() 读当前 frame.LocalParams
// (subgraph 入参). frame-private state, snapshot wrap 不包.
package variable

import (
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&GetParam{}) }

type GetParam struct{}

const (
	gpInParamName = "ParamName"
	gpOutValue    = "Value"
)

func (GetParam) Spec() node.Spec {
	return node.Spec{
		Kind:     "GetParam",
		Category: "Variable",
		Inputs: []node.InputSpec{
			{Name: gpInParamName, Type: "String", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: gpOutValue, Type: "*"},
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
