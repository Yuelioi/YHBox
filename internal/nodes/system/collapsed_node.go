// internal/nodes/system/collapsed_node.go
// CollapsedNode — Subgraph 的折叠 (anonymous) 表示. 老 spec ExecIn=[in], config
// {subgraphId, label}; runtime/nodes.go::case "Subgraph", "CollapsedNode" 同
// dispatch 路径 (CollapsedNode = isAnonymous Subgraph, runtime 路径相同).
//
// Phase 5.5b: 跟 Subgraph 共享 RegionRunner 实现 — body() 调一次 → Done 出口.
// runtime 端 makeBodyFor switch 已含 "CollapsedNode" case (复用 makeBodyForSubgraph).
package system

import "yhbox/internal/node"

func init() { node.Register(&CollapsedNode{}) }

type CollapsedNode struct{}

const (
	cnInExec       = "In"
	cnInSubgraphID = "SubgraphID"
	cnInLabel      = "Label"
	cnOutDone      = "Done"
)

func (CollapsedNode) Spec() node.Spec {
	return node.Spec{
		Kind:        "CollapsedNode",
		Category:    "System",
		DisplayName: "折叠子图",
		Description: "Subgraph 的折叠 (isAnonymous) 表示 — runtime 跟 Subgraph 同 dispatch (RegionRunner: body 调一次 → Done).",
		Inputs: []node.InputSpec{
			{Name: cnInExec, Type: "Exec"},
			{Name: cnInSubgraphID, Type: "String", Semantic: "SubgraphID", Required: true,
				DisplayName: "子图 ID",
				Doc:         "目标 isAnonymous Subgraph 标识符",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "subgraphIDs"})}},
			{Name: cnInLabel, Type: "String", Default: "Collapsed",
				DisplayName: "标签",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: cnOutDone, Type: "Exec", DisplayName: "完成"},
		},
	}
}

// RunRegion — body() 调一次, 跑 callee subgraph. error 透传; 无 error → Done.
func (CollapsedNode) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	if err := body(ctx); err != nil {
		return nil, err
	}
	return ctx.Out(cnOutDone).Fire(), nil
}

// Dependencies — 子图分享 / library import 时 BFS 抽 callee 引用.
func (CollapsedNode) Dependencies(in node.Inputs) []node.Dependency {
	id := in.String(cnInSubgraphID)
	if id == "" {
		return nil
	}
	return []node.Dependency{{Kind: "subgraph", Key: id}}
}
