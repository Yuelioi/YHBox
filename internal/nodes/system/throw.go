// internal/nodes/system/throw.go
// Throw — 显式抛 error. 返 typed ThrowError 让最近 Try 节点截获走 error 出口;
// 没 Try 包就冒泡到主 runner 当 container:error.
//
// Try region 用 errors.As(err, &te) 抽 message 注入 catch 出口 data field.
package system

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Throw{}) }

type Throw struct{}

const (
	thInExec    = "In"
	thInMessage = "Message"
)

func (Throw) Spec() node.Spec {
	return node.Spec{
		Kind:     "Throw",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: thInExec, Type: "Exec"},
			{Name: thInMessage, Type: "String", Default: "",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — terminal node
	}
}

func (Throw) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, &ThrowError{Message: in.String(thInMessage)}
}

func (Throw) Display(in node.Inputs, exitName string, out node.OutputData) string {
	msg := in.String(thInMessage)
	if msg == "" {
		return "throw (no message)"
	}
	return fmt.Sprintf("throw: %s", msg)
}
