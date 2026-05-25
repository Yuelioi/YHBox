// internal/nodes/system/throw.go
// Throw — 显式抛 error. 老 runtime throw_nodes.go::execThrow 返 *errThrow 让最近 Try
// 节点截获走 error 出口; 没 Try 包就冒泡到主 runner 当 container:error.
//
// Phase 5: 返 typed ThrowError. Try region 用 errors.As(err, &te) 抽 message 注入
// catch 出口 data field; 也可 errors.Is(err, errThrow) 判断是否 throw (Try 当前接受
// 任何 error, 不强区分, 留给未来需求).
package system

import (
	"yhbox/internal/node"
)

func init() { node.Register(&Throw{}) }

type Throw struct{}

const (
	thInExec    = "in"
	thInMessage = "message"
)

func (Throw) Spec() node.Spec {
	return node.Spec{
		Kind:        "Throw",
		Category:    "System",
		DisplayName: "抛错",
		Description: "立刻抛 ThrowError, 由最近的 Try 区域截获走 catch 出口; 没 Try 包就冒泡到主 runner 报 container:error. errors.As 可抽原 message.",
		Inputs: []node.InputSpec{
			{Name: thInExec, Type: "Exec"},
			{Name: thInMessage, Type: "String", Default: "",
				DisplayName: "消息",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — terminal node
	}
}

func (Throw) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, &ThrowError{Message: in.String(thInMessage)}
}
