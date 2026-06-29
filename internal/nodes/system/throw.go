// internal/nodes/system/throw.go
// Throw — 显式抛 error. 返 typed ThrowError (实现 node.Coded) 让最近一个接线的 region
// Fail 出口截获走失败分支; 没 region 接就冒泡到主 runner 当 container:error.
//
// dispatch 失败路由用 errors.As(err, &coded) 抽 Code + message 注入 Fail 出口 data field.
package system

import (
	"yotta/internal/node"
)

func init() { node.Register(&Throw{}) }

type Throw struct{}

const (
	thInExec    = "In"
	thInMessage = "Message"
	thInCode    = "Code"
)

func (Throw) Spec() node.Spec {
	return node.Spec{
		Kind:     "Throw",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: thInExec, Type: "Exec"},
			{Name: thInMessage, Type: "String", Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: thInCode, Type: "String", Default: "", Advanced: true,
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — terminal node
	}
}

func (Throw) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, &ThrowError{Message: in.String(thInMessage), Code: in.String(thInCode)}
}
