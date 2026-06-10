// internal/nodes/control/foreach.go
// ForEach — RegionRunner 节点. 遍历 List, 每轮 Capture 元素+下标跑 Body.
// Break/Continue sentinel 同 Loop. 非列表/空列表 → 0 轮直接 Done (不算错).
// Category "List" (palette 与列表节点同组; 机制归 control 包).
package control

import (
	"errors"

	"yotta/internal/node"
)

func init() { node.Register(&ForEach{}) }

type ForEach struct{}

const (
	feInExec   = "In"
	feInList   = "List"
	feCapItem  = "CaptureItem"
	feCapIndex = "CaptureIndex"

	feOutBody = "Body"
	feOutDone = "Done"
)

func (ForEach) Spec() node.Spec {
	return node.Spec{
		Kind:     "ForEach",
		Category: "List",
		Inputs: []node.InputSpec{
			{Name: feInExec, Type: "Exec"},
			{Name: feInList, Type: "List"},
			{Name: feCapItem, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "any", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: feCapIndex, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: feOutBody, Type: "Exec"},
			{Name: feOutDone, Type: "Exec"},
			{Name: "Fail", Type: "Exec", Semantic: "error",
				Data: []node.DataField{
					{Name: "Error", Type: "String"},
					{Name: "Code", Type: "String"},
				}},
		},
	}
}

// RunRegion — items 在 ForEach 自身 dispatch 入口取一次 (上游非确定节点由 per-dispatch
// 缓存保证同值 → 列表对整轮循环稳定). 快照仅冻结切片头: 元素是引用, body 改其内容后续轮可见.
func (ForEach) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	items := in.List(feInList)
	for i, el := range items {
		node.Capture(ctx, in, feCapItem, el)
		node.Capture(ctx, in, feCapIndex, i)
		if err := body(ctx); err != nil {
			if errors.Is(err, errBreakRequested) {
				return ctx.Out(feOutDone).Fire(), nil
			}
			if errors.Is(err, errContinueRequested) {
				continue
			}
			return nil, err
		}
	}
	return ctx.Out(feOutDone).Fire(), nil
}
