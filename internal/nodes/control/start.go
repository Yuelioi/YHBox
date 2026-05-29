// internal/nodes/control/start.go
// Start — graph entry node. Every graph has exactly 1 Start; framework dispatch
// begins from it. No exec-in pin (you can't wire INTO an entry point).
package control

import "yhbox/internal/node"

func init() { node.Register(&Start{}) }

type Start struct{}

const (
	startOutOut = "Done"
)

func (Start) Spec() node.Spec {
	return node.Spec{
		Kind:     "Start",
		Category: "Control",
		// 不要 exec-in — Start 是入口
		Outputs: []node.OutputSpec{
			{Name: startOutOut, Type: "Exec"},
		},
	}
}

func (Start) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return ctx.Out(startOutOut).Fire(), nil
}
