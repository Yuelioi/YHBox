// internal/nodes/system/subgraph_output.go
// SubgraphOutput — 子图出口 marker. 老 spec ExecIn=[in]/ExecOut=nil + declID
// config 字段; runtime/nodes.go::execSubgraphOutput pop frame + 切回父图 +
// 找父图调用方 declID 边的下游.
//
// 新框架 Phase 4 stub: Run 返 errSubgraphNodeStub (sentinels.go). Phase 5
// sub-runner 截 sentinel, pop frame + resume 父 frame.
package system

import "yhbox/internal/node"

func init() { node.Register(&SubgraphOutput{}) }

type SubgraphOutput struct{}

const (
	soInExec   = "In"
	soInDeclID = "DeclID"
)

func (SubgraphOutput) Spec() node.Spec {
	return node.Spec{
		Kind:        "SubgraphOutput",
		Category:    "System",
		DisplayName: "子图出口",
		Description: "(Phase 5 stub) 子图出口节点 — pop frame 回父图, 走父图调用方 declID 对应的下游. 当前 Run 返 sentinel.",
		Inputs: []node.InputSpec{
			{Name: soInExec, Type: "Exec"},
			{Name: soInDeclID, Type: "String", Default: "",
				DisplayName: "出口 ID",
				Doc:         "对应 Subgraph.OutputPins[].declID, 父图调用方按此 declID 路由下游边.",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — sub-runner 处理 frame pop + parent resume
	}
}

func (SubgraphOutput) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errSubgraphNodeStub
}
