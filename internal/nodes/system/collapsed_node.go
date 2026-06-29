// internal/nodes/system/collapsed_node.go
// CollapsedNode — Subgraph 的折叠 (anonymous) 表示, 跟 Subgraph 共享 RegionRunner
// 语义: body() 调一次, fire body 回报的出口 decl ID (DynamicOutputs).
package system

import "yotta/internal/node"

func init() { node.Register(&CollapsedNode{}) }

type CollapsedNode struct{}

const (
	cnInExec       = "In"
	cnInSubgraphID = "SubgraphID"
	cnInLabel      = "Label"
)

func (CollapsedNode) Spec() node.Spec {
	return node.Spec{
		Kind:     "CollapsedNode",
		Category: "System",
		// 出口 = callee OutputPins 的 decl ID, 随后备子图动态变 — 静态只声明 Fail.
		DynamicOutputs: true,
		Inputs: []node.InputSpec{
			{Name: cnInExec, Type: "Exec"},
			{Name: cnInSubgraphID, Type: "String", Semantic: "SubgraphID", Required: true,
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "subgraphIDs"})}},
			{Name: cnInLabel, Type: "String", Default: "Collapsed",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: "Fail", Type: "Exec", Semantic: "error",
				Data: []node.DataField{
					{Name: "Error", Type: "String"},
					{Name: "Code", Type: "String"},
				}},
		},
	}
}

// RunRegion — body() 调一次, 跑 callee subgraph. error 透传; exit = 到达的出口
// decl ID, "" → 不 fire.
func (CollapsedNode) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) (string, error)) (node.Outputs, error) {
	exit, err := body(ctx)
	if err != nil {
		return nil, err
	}
	if exit == "" {
		return nil, nil
	}
	return ctx.Out(exit).Fire(), nil
}

// Dependencies — 子图分享 / library import 时 BFS 抽 callee 引用.
func (CollapsedNode) Dependencies(in node.Inputs) []node.Dependency {
	id := in.String(cnInSubgraphID)
	if id == "" {
		return nil
	}
	return []node.Dependency{{Kind: "subgraph", Key: id}}
}
