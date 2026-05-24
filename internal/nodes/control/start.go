// internal/nodes/control/start.go
// Start — graph entry node. Every graph has exactly 1 Start; framework dispatch
// begins from it. No exec-in pin (you can't wire INTO an entry point).
package control

import "yhbox/internal/node"

func init() { node.Register(&Start{}) }

type Start struct{}

const (
	startOutOut = "out"
)

func (Start) Spec() node.Spec {
	return node.Spec{
		Kind:        "Start",
		Version:     1,
		Category:    "Control",
		DisplayName: "起点",
		Description: "图入口. 框架启动时从 Start 节点开始执行. 每图恰好 1 个.",
		// 不要 exec-in — Start 是入口
		Outputs: []node.OutputSpec{
			{Name: startOutOut, Type: "Exec", DisplayName: "开始"},
		},
	}
}

func (Start) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return ctx.Out(startOutOut).Fire(), nil
}
