// internal/nodes/system/subgraph_output.go
// SubgraphOutput — 子图出口 marker. Spec.IsGraphMarker=true; runner 在
// dispatchInRegion / runRegionBody 里 pop frame + 切回父图 + 找父图调用方
// declID 边的下游, 不调任何执行接口. 零 capability.
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
		Description: "子图出口节点 — pop frame 回父图, 走父图调用方 declID 对应的下游. framework special-route.",
		Inputs: []node.InputSpec{
			{Name: soInExec, Type: "Exec"},
			{Name: soInDeclID, Type: "String", Default: "",
				DisplayName: "出口 ID",
				Doc:         "对应 Subgraph.OutputPins[].declID, 父图调用方按此 declID 路由下游边.",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — sub-runner 处理 frame pop + parent resume
		IsGraphMarker: true,
	}
}
