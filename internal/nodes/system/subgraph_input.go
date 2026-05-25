// internal/nodes/system/subgraph_input.go
// SubgraphInput — 子图入口 marker. Spec.IsGraphMarker=true; runner 在
// dispatchInRegion / runRegionBody 里 special-route 跳过执行, 直通 out edge.
// 零 capability (无 Run/RunRegion/Evaluate).
package system

import "yhbox/internal/node"

func init() { node.Register(&SubgraphInput{}) }

type SubgraphInput struct{}

const (
	siOutOut = "out"
)

func (SubgraphInput) Spec() node.Spec {
	return node.Spec{
		Kind:        "SubgraphInput",
		Category:    "System",
		DisplayName: "子图入口",
		Description: "子图入口节点 — sub-runner 从这里开跑. framework special-route, 不调任何执行接口.",
		// no Inputs — entry point
		Outputs: []node.OutputSpec{
			{Name: siOutOut, Type: "Exec", DisplayName: "out"},
		},
		IsGraphMarker: true,
	}
}
