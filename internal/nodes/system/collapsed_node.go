// internal/nodes/system/collapsed_node.go
// CollapsedNode — Subgraph 的折叠 (anonymous) 表示. 老 spec ExecIn=[in], config
// {subgraphId, label}; runtime/nodes.go::case "Subgraph", "CollapsedNode" 同
// dispatch 路径 (CollapsedNode = isAnonymous Subgraph, runtime 路径相同).
//
// 新框架 Phase 4 stub: Run 返 errSubgraphNodeStub (sentinels.go). Phase 5
// sub-runner 接管后跟 Subgraph 共享 dispatch.
package system

import "yhbox/internal/node"

func init() { node.Register(&CollapsedNode{}) }

type CollapsedNode struct{}

const (
	cnInExec       = "in"
	cnInSubgraphID = "subgraphId"
	cnInLabel      = "label"
)

func (CollapsedNode) Spec() node.Spec {
	return node.Spec{
		Kind:        "CollapsedNode",
		Version:     1,
		Category:    "System",
		DisplayName: "折叠子图",
		Description: "(Phase 5 stub) Subgraph 的折叠 (isAnonymous) 表示 — runtime 跟 Subgraph 同 dispatch. 当前 Run 返 sentinel.",
		Inputs: []node.InputSpec{
			{Name: cnInExec, Type: "Exec"},
			{Name: cnInSubgraphID, Type: "String", Required: true,
				DisplayName: "子图 ID",
				Doc:         "目标 isAnonymous Subgraph 标识符",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: cnInLabel, Type: "String", Default: "Collapsed",
				DisplayName: "标签",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — exec-out 来自被调子图的 OutputPins, 动态决定 (跟 Subgraph 同).
	}
}

func (CollapsedNode) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errSubgraphNodeStub
}
