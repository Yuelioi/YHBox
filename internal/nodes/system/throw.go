// internal/nodes/system/throw.go
// Throw — 显式抛 error. 老 runtime throw_nodes.go::execThrow 返 *errThrow 让
// 最近 Try 节点截获走 error 出口, 没 Try 包就冒泡到主 runner 当 container:error.
//
// 新框架 Phase 4 stub: 单 exec-in, 无 exec-out, Run 返 errors.New("throw: " + msg).
// Phase 5 Try region 实做后再换成 typed sentinel + errors.As 抽 message.
package system

import (
	"errors"

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
		Version:     1,
		Category:    "System",
		DisplayName: "抛错",
		Description: "立刻抛 error, 由最近的 Try 区域截获走 error 出口; 没 Try 包就冒泡到主 runner 报 container:error.",
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
	return nil, errors.New("throw: " + in.String(thInMessage))
}
