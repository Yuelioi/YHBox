// internal/nodes/control/foreach.go
// ForEach — RegionRunner 节点. 遍历 List, 每轮 Capture 元素+下标跑 Body.
// Break/Continue sentinel 同 Loop. 非列表/空列表 → 0 轮直接 Done (不算错).
// Category "List" (palette 与列表节点同组; 机制归 control 包).
package control

import (
	"errors"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&ForEach{}) }

type ForEach struct{}

const (
	feInExec = "In"
	feInList = "List"

	feOutBody = "Body"
	feOutDone = "Done"

	// Body 出口 Data 字段 — 每轮迭代的元素 / 序号, 供路径② ctx.CaptureOutput 绑变量 (Spec C)
	feDataItem  = "Item"
	feDataIndex = "Index"
)

func (ForEach) Spec() node.Spec {
	return node.Spec{
		Kind:     "ForEach",
		Category: "List",
		Inputs: []node.InputSpec{
			{Name: feInExec, Type: "Exec"},
			{Name: feInList, Type: "List"},
		},
		Outputs: []node.OutputSpec{
			{Name: feOutBody, Type: "Exec", Data: []node.DataField{
				{Name: feDataItem, Type: "Any"},
				{Name: feDataIndex, Type: "Number"},
			}},
			{Name: feOutDone, Type: "Exec"},
			{Name: "Fail", Type: "Exec", Semantic: "error",
				Data: []node.DataField{
					{Name: "Error", Type: "String"},
					{Name: "Code", Type: "String"},
				}},
		},
	}
}

// RunRegion — items 由 dataWire 在 ForEach 自身 dispatch 入口拉取一次 (稳定性来自单次拉取;
// per-dispatch 缓存只负责同 dispatch 内多 pin 引用同值). 快照仅冻结切片头: 元素是引用,
// body 改其内容后续轮可见.
func (ForEach) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) (string, error)) (node.Outputs, error) {
	items := in.List(feInList)
	for i, el := range items {
		ctx.CaptureOutput(feDataItem, el)
		ctx.CaptureOutput(feDataIndex, i)
		if _, err := body(ctx); err != nil {
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
