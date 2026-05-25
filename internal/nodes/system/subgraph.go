// internal/nodes/system/subgraph.go
// Subgraph — 调用容器内 SubgraphID 指定的子图. Phase 5 RegionRunner 实现.
//
// 节点本身只 wrap region 边界 — body 回调由 Phase 5 runner 构造, 内含 "切换
// dispatch table 到目标子图 + 推 ExecFrame + 跑下游 + 返回" 全套. Subgraph.RunRegion
// 调一次 body, body 返 error 透传, 无 error → 走 Done 出口.
//
// Phase 5 简化: 静态 SubgraphID 输入 + 单 Params JSON. dynamic DataIn pin (跟 callee.
// inputParams 1:1) 留 Phase 6+ — 当前所有 fishing-v2 helper 走 default config 都可调通.
package system

import (
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
		Kind:        "Subgraph",
		Category:    "System",
		DisplayName: "调用子图",
		Description: "调用容器内 SubgraphID 指定的子图. body 回调由 runner 构造 + 跑完返回, 无 error → 走 Done. Phase 5: 静态 ID + 简化 Params JSON, Phase 6 加 dynamic InputParams.",
		Inputs: []node.InputSpec{
			{Name: sgInExec, Type: "Exec"},
			{Name: sgInSubgraphID, Type: "String", Semantic: "SubgraphID", Required: true,
				DisplayName: "子图 ID",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "subgraphIDs"})}},
			{Name: sgInParams, Type: "JSON", Default: map[string]any{},
				DisplayName: "参数",
				Doc:         "调用参数 (Phase 5 透传给 runner — runner 决定如何注入 callee 的 SubgraphInput).",
				Widget:      node.WidgetSpec{Kind: "json"}},
		},
		Outputs: []node.OutputSpec{
			{Name: sgOutDone, Type: "Exec", DisplayName: "完成"},
		},
	}
}

// Run — 防御性. 正常路径走 RunNodeAsRegion → RunRegion.
func (Subgraph) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errSubgraphMustUseRegion
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
