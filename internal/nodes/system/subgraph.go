// internal/nodes/system/subgraph.go
// Subgraph — 调用容器内 SubgraphID 指定的子图.
//
// 节点本身只 wrap region 边界 — body 回调由 runner 构造, 内含 "切换 dispatch
// table 到目标子图 + 推 ExecFrame + 跑下游 + 返回" 全套. Subgraph.RunRegion
// 调一次 body, body 返 error 透传, 无 error → 走 Done 出口.
//
// 只支持静态 SubgraphID 输入 + 单 Params JSON, 无 dynamic DataIn pin.
package system

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Subgraph{}) }

type Subgraph struct{}

const (
	sgInExec       = "In"
	sgInSubgraphID = "SubgraphID"
	sgInParams     = "Params"
	sgOutDone      = "Done"
)

func (Subgraph) Spec() node.Spec {
	return node.Spec{
		Kind:     "Subgraph",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: sgInExec, Type: "Exec"},
			{Name: sgInSubgraphID, Type: "String", Semantic: "SubgraphID", Required: true,
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "subgraphIDs"})}},
			{Name: sgInParams, Type: "JSON", Default: map[string]any{},
				Widget: node.WidgetSpec{Kind: "json"}},
		},
		Outputs: []node.OutputSpec{
			{Name: sgOutDone, Type: "Exec"},
		},
	}
}

func (Subgraph) Display(in node.Inputs, exitName string, out node.OutputData) string {
	id := in.String(sgInSubgraphID)
	if id == "" {
		return "call (?)"
	}
	return fmt.Sprintf("call %s → Done", id)
}

// RunRegion — body() 调一次, 跑 callee 子图. error 透传; 无 error → Done.
func (Subgraph) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	if err := body(ctx); err != nil {
		return nil, err
	}
	return ctx.Out(sgOutDone).Fire(), nil
}

// Dependencies — 子图分享 / library import 时 BFS 抽 callee 引用.
func (Subgraph) Dependencies(in node.Inputs) []node.Dependency {
	id := in.String(sgInSubgraphID)
	if id == "" {
		return nil
	}
	return []node.Dependency{{Kind: "subgraph", Key: id}}
}
