// Package collection 列表纯数据节点 (7 个): Split/Join/ListLength/ListGet/ListContains/
// ListAppend/ListSlice. 见 specs/2026-06-10-collection-nodes.md C 节. Category "List".
// 元素不分类型 ([]any); 越界/非列表一律安全值 (nil/空/0), 从不 error.
package collection

import (
	"encoding/json"
	"strings"

	"yotta/internal/node"
)

func init() {
	for _, n := range []node.Node{
		&Split{}, &Join{}, &ListLength{}, &ListGet{},
	} {
		node.Register(n)
	}
}

// listSpec 单 Result 出口的 List 分类 pure-data Spec (同 purefunc.specBuilder 思路, 包内自持).
func listSpec(kind string, inputs []node.InputSpec, resultType string) node.Spec {
	return node.Spec{
		Kind: kind, Category: "List",
		Inputs:     inputs,
		Outputs:    []node.OutputSpec{{Name: "Result", Type: resultType}},
		IsPureData: true,
	}
}

func listIn() node.InputSpec {
	return node.InputSpec{Name: "List", Type: "List"}
}

func sepIn() node.InputSpec {
	return node.InputSpec{Name: "Separator", Type: "String", Default: ",", Widget: node.WidgetSpec{Kind: "text"}}
}

// ===== Split =====

type Split struct{}

func (Split) Spec() node.Spec {
	return listSpec("Split", []node.InputSpec{
		{Name: "Text", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		sepIn(),
	}, "List")
}

// Evaluate — Text=="" → 空列表 (刻意偏离 Go Split("",sep) 的 [""], 更直觉);
// Separator=="" → 按 UTF-8 字符逐个拆 (Go strings.Split 语义, rune 边界).
func (Split) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	text := in.String("Text")
	if text == "" {
		return []any{}, nil
	}
	parts := strings.Split(text, in.String("Separator"))
	out := make([]any, len(parts))
	for i, p := range parts {
		out[i] = p
	}
	return out, nil
}

// ===== Join =====

type Join struct{}

func (Join) Spec() node.Spec {
	return listSpec("Join", []node.InputSpec{listIn(), sepIn()}, "String")
}
func (Join) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	parts := make([]string, len(items))
	for i, el := range items {
		parts[i] = node.FormatValue(el)
	}
	return strings.Join(parts, in.String("Separator")), nil
}

// ===== ListLength =====

type ListLength struct{}

func (ListLength) Spec() node.Spec {
	return listSpec("ListLength", []node.InputSpec{listIn()}, "Number")
}
func (ListLength) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return float64(len(in.List("List"))), nil
}

// ===== ListGet =====

type ListGet struct{}

func (ListGet) Spec() node.Spec {
	return listSpec("ListGet", []node.InputSpec{
		listIn(),
		{Name: "Index", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "*")
}

// Evaluate — 越界 (含负) → nil (不做负索引). nil 歧义 (元素=nil vs 越界) spec 已声明: 要区分先 ListLength.
func (ListGet) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	idx := in.Int("Index")
	if idx < 0 || idx >= len(items) {
		return nil, nil
	}
	return items[idx], nil
}
