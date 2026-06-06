// internal/nodes/system/try.go
// Try — region 节点. body 跑完无 error → normal 出口; body 返 error → catch 出口,
// error.Error() 字符串挂在 catch.error data field.
//
// 坑: Try 截获 *任何* body error. Loop 的 Break/Continue sentinel 如果穿过 Try
// 边界会被当 catchable error 处理 → 不 break 上游 Loop.
package system

import (
	"yotta/internal/node"
)

func init() { node.Register(&Try{}) }

type Try struct{}

const (
	tryInExec       = "In"
	tryInSubgraphID = "SubgraphID"
	tryOutNormal    = "Out"
	tryOutCatch     = "Catch"
	tryDataError    = "Error"
	tryCapError     = "CaptureError"
)

func (Try) Spec() node.Spec {
	return node.Spec{
		Kind:     "Try",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: tryInExec, Type: "Exec"},
			{Name: tryInSubgraphID, Type: "String", Semantic: "SubgraphID", Required: true,
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "subgraphIDs"})}},
			{Name: tryCapError, Type: "String", Advanced: true, Semantic: "capture",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: tryOutNormal, Type: "Exec"},
			{Name: tryOutCatch, Type: "Exec",
				Data: []node.DataField{
					{Name: tryDataError, Type: "String"},
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

// RunRegion — body 跑完 error → catch.error; 无 error → normal.
func (Try) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	err := body(ctx)
	if err != nil {
		node.Capture(ctx, in, tryCapError, err.Error())
		return ctx.Out(tryOutCatch).Set(tryDataError, err.Error()).Fire(), nil
	}
	node.Capture(ctx, in, tryCapError, "")
	return ctx.Out(tryOutNormal).Fire(), nil
}
