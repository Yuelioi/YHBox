// Package collection 列表纯数据节点 (7 个): Split/Join/ListLength/ListGet/ListContains/
// ListAppend/ListSlice. 见 specs/2026-06-10-collection-nodes.md C 节. Category "List".
// 元素不分类型 ([]any); 越界/非列表一律安全值 (nil/空/0), 从不 error.
package collection

import (
	"encoding/json"
	"strings"

	"github.com/yottaapp/yotta/internal/node"
)

func init() {
	for _, n := range []node.Node{
		&Split{}, &Join{}, &ListLength{}, &ListGet{},
		&ListContains{}, &ListAppend{}, &ListSlice{},
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

// ===== ListContains =====

type ListContains struct{}

func (ListContains) Spec() node.Spec {
	return listSpec("ListContains", []node.InputSpec{
		listIn(),
		{Name: "Value", Type: "*"},
	}, "Bool")
}

// Evaluate — 与 Eq 节点完全同语义 (node.LooseEqual): 同类型直比、跨类型串比.
func (ListContains) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	val := in.Raw("Value")
	for _, el := range in.List("List") {
		if node.LooseEqual(el, val) {
			return true, nil
		}
	}
	return false, nil
}

// ===== ListAppend =====

type ListAppend struct{}

func (ListAppend) Spec() node.Spec {
	return listSpec("ListAppend", []node.InputSpec{
		listIn(),
		{Name: "Item", Type: "*"},
	}, "List")
}

// Evaluate — 返回新列表, 必 copy: 防 append 原地改写上游 Evaluate 返回的切片 (底层数组别名).
// 浅拷贝 — 嵌套 map/子 list 与原列表共享引用 (值语义同 Python list.copy(), 非 bug, i18n 写明).
func (ListAppend) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	out := make([]any, 0, len(items)+1)
	out = append(out, items...)
	return append(out, in.Raw("Item")), nil
}

// ===== ListSlice =====

type ListSlice struct{}

func (ListSlice) Spec() node.Spec {
	return listSpec("ListSlice", []node.InputSpec{
		listIn(),
		{Name: "Start", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		// Count 默认 -1 = 取到末尾 (与 Substring.Length 同约定: 负=到末尾/0=空/正=N).
		{Name: "Count", Type: "Integer", Default: json.Number("-1"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "List")
}

// Evaluate — 返回新列表 (copy 防别名). Start clamp [0,len], Start>=len → 恒空.
func (ListSlice) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	start := in.Int("Start")
	if start < 0 {
		start = 0
	}
	if start >= len(items) {
		return []any{}, nil
	}
	count := in.Int("Count")
	if count == 0 {
		return []any{}, nil
	}
	end := len(items)
	if count > 0 {
		end = start + count
		if end > len(items) {
			end = len(items)
		}
	}
	out := make([]any, end-start)
	copy(out, items[start:end])
	return out, nil
}
