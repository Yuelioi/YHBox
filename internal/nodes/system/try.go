// internal/nodes/system/try.go
// Try — region 节点. body 跑完无 error → normal 出口; body 返 error → catch 出口,
// error.Error() 字符串挂在 catch.error data field.
//
// Phase 5 简化: 截获 *任何* body error. Loop 的 Break/Continue sentinel 如果穿过
// Try 边界会被当 catchable error 处理 → 不 break 上游 Loop. 这是 Phase 6 refinement
// (Try 应仅截获 ThrowError + 透传 Break/Continue), 当前接受 — fishing-v2 redraw 不
// 会出现 Try 包 Loop body 内 Break 跨 Try 边界的拓扑.
package system

import (
	"yhbox/internal/node"
)

func init() { node.Register(&Try{}) }

type Try struct{}

const (
	tryInExec       = "In"
	tryInSubgraphID = "SubgraphID"
	tryOutNormal    = "Out"
	tryOutCatch     = "Catch"
	tryDataError    = "Error"
)

func (Try) Spec() node.Spec {
	return node.Spec{
		Kind:        "Try",
		Category:    "System",
		DisplayName: "Try Catch",
		Description: "捕获 body 子图内的 error (含 Throw). 正常完成走 out; 出错走 catch, error 字符串挂在 catch.error 字段. body 子图由 SubgraphID 指定 (镜像老 runtime config.subgraphId).",
		Inputs: []node.InputSpec{
			{Name: tryInExec, Type: "Exec"},
			{Name: tryInSubgraphID, Type: "String", Semantic: "SubgraphID", Required: true,
				DisplayName: "Body 子图",
				Doc:         "Try 包裹的子图 ID; runner 端 push frame 跑该子图, body return error 触发 catch.",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "subgraphIDs"})}},
		},
		Outputs: []node.OutputSpec{
			{Name: tryOutNormal, Type: "Exec", DisplayName: "正常"},
			{Name: tryOutCatch, Type: "Exec", DisplayName: "捕获",
				Data: []node.DataField{
					{Name: tryDataError, Type: "String", Doc: "捕获的 error.Error() 字符串. ThrowError 抛出时即 Throw 节点的 Message."},
				}},
		},
	}
}

// Dependencies — 子图分享 / library import 时 BFS 抽 callee 引用.
func (Try) Dependencies(in node.Inputs) []node.Dependency {
	id := in.String(tryInSubgraphID)
	if id == "" {
		return nil
	}
	return []node.Dependency{{Kind: "subgraph", Key: id}}
}

// Run — 防御性. 正常路径走 RunNodeAsRegion → RunRegion.
func (Try) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, node.ErrMustUseRegion
}

// RunRegion — body 跑完 error → catch.error; 无 error → normal.
func (Try) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	err := body(ctx)
	if err != nil {
		return ctx.Out(tryOutCatch).Set(tryDataError, err.Error()).Fire(), nil
	}
	return ctx.Out(tryOutNormal).Fire(), nil
}
