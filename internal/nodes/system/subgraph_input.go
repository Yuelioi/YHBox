// internal/nodes/system/subgraph_input.go
// SubgraphInput — 子图入口 marker. 老 spec ExecIn=nil/ExecOut=[out], runtime
// nodes.go::case "SubgraphInput" 直通到 .out edges (本质是 entry point, sub-runner
// 从这个节点起跑).
//
// 新框架 Phase 4 stub: Run 返 errSubgraphNodeStub (sentinels.go). Phase 5
// sub-runner / RegionRunner 接管后, 这节点不再走 Run, framework 在 push frame
// 时直接 dispatch 它的 out edge.
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
		Description: "(Phase 5 stub) 子图入口节点 — sub-runner 从这里开跑. 当前 Run 返 sentinel, Phase 5 sub-runner 接管后不再走 Run.",
		// no Inputs — entry point
		Outputs: []node.OutputSpec{
			{Name: siOutOut, Type: "Exec", DisplayName: "out"},
		},
	}
}

func (SubgraphInput) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errSubgraphNodeStub
}
