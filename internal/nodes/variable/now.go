// internal/nodes/variable/now.go
// Now — pure-data，返当前 unix 毫秒（live 墙钟，不进快照）。取代旧 $sys.now_ms。
// 配 VarLastChange 算"距今": Now - VarLastChange(x)。
package variable

import "yotta/internal/node"

func init() { node.Register(&Now{}) }

type Now struct{}

const nowOutValue = "Value"

func (Now) Spec() node.Spec {
	return node.Spec{
		Kind:       "Now",
		Category:   "Variable",
		Outputs:    []node.OutputSpec{{Name: nowOutValue, Type: "Number"}},
		IsPureData: true,
	}
}

func (Now) Evaluate(ctx node.Ctx, _ node.Inputs) (any, error) {
	return float64(ctx.Now().UnixMilli()), nil
}
