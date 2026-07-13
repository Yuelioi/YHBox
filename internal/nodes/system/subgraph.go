// internal/nodes/system/subgraph.go
// Subgraph — 调用容器内 SubgraphID 指定的子图.
//
// 节点本身只 wrap region 边界 — body 回调由 runner 构造, 内含 "切换 dispatch
// table 到目标子图 + 推 ExecFrame + 跑下游 + 返回" 全套. body 回报 callee 到达的
// 出口 decl ID, 节点原样 fire — 父图边以 decl ID 为动态 output pin.
//
// 只支持静态 SubgraphID 输入 + 单 Params JSON, 无 dynamic DataIn pin.
package system

import (
	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&Subgraph{}) }

type Subgraph struct{}

const (
	sgInExec       = "In"
	sgInSubgraphID = "SubgraphID"
	sgInParams     = "Params"
)

func (Subgraph) Spec() node.Spec {
	return node.Spec{
		Kind:     "Subgraph",
		Category: "System",
		// 出口 = callee OutputPins 的 decl ID, 随绑定子图动态变 — 静态只声明 Fail.
		DynamicPorts: []node.DynamicPortSpec{{
			Role: node.DynamicPortOutput, Shape: node.DynamicPortGraphInterface,
			MaxItems: 4096,
		}},
		Inputs: []node.InputSpec{
			{Name: sgInExec, Type: "Exec"},
			{Name: sgInSubgraphID, Type: "String", Semantic: "SubgraphID", Required: true,
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "subgraphIDs"})}},
			{Name: sgInParams, Type: "JSON", Default: map[string]any{},
				Widget: node.WidgetSpec{Kind: "json"}},
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

// RunRegion — body() 调一次, 跑 callee 子图. error 透传; exit = 到达的出口 decl ID,
// "" (没到达任何出口) → 不 fire, 父图流到此为止.
func (Subgraph) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) (string, error)) (node.Outputs, error) {
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
func (Subgraph) Dependencies(in node.Inputs) []node.Dependency {
	id := in.String(sgInSubgraphID)
	if id == "" {
		return nil
	}
	return []node.Dependency{{Kind: "subgraph", Key: id}}
}
