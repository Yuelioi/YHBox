// Package catalog 把 node.All() 注册表提炼成对 LLM / 文档友好的节点目录 (只结构,
// 展示文本 i18n 是 fast-follow)。cmd/node-catalog (CLI) 和 cmd/yotta-mcp (MCP) 共享本包。
package catalog

import (
	_ "embed"
	"encoding/json"
	"sort"

	"yotta/internal/node"
)

type Pin struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Exec     bool   `json:"exec,omitempty"`
	Required bool   `json:"required,omitempty"`
	Advanced bool   `json:"advanced,omitempty"`
	Default  any    `json:"default,omitempty"`
	// 展示文案 (仅 BuildWithI18n 填充; Build 结构-only 时为空, omitempty 不输出)。
	Label string `json:"label,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

type Node struct {
	Kind        string `json:"kind"`
	Category    string `json:"category"`
	NeedsWindow bool   `json:"needsWindow,omitempty"`
	IsPureData  bool   `json:"isPureData,omitempty"`
	Inputs      []Pin  `json:"inputs"`
	Outputs     []Pin  `json:"outputs"`
	// 展示文案 (仅 BuildWithI18n 填充)。
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// Build 读全注册表, 返按 category→kind 稳定排序的目录。
// 调用方负责匿名 import 全 internal/nodes/* 触发注册。
func Build() []Node {
	regs := node.All()
	out := make([]Node, 0, len(regs))
	for _, rn := range regs {
		s := rn.Spec
		cn := Node{Kind: s.Kind, Category: s.Category, NeedsWindow: s.NeedsWindow, IsPureData: s.IsPureData}
		for _, in := range s.Inputs {
			cn.Inputs = append(cn.Inputs, Pin{
				Name: in.Name, Type: in.Type, Exec: in.Type == node.TypeExec,
				Required: in.Required, Advanced: in.Advanced, Default: in.Default,
			})
		}
		for _, o := range s.Outputs {
			cn.Outputs = append(cn.Outputs, Pin{Name: o.Name, Type: o.Type, Exec: o.Type == node.TypeExec})
		}
		out = append(out, cn)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

// node-i18n.json 由 frontend `pnpm gen:node-i18n` 从 zh.ts 的 node.* 块抽取生成。
//
//go:embed node-i18n.json
var nodeI18nJSON []byte

type pinI18n struct {
	Label string `json:"label"`
	Hint  string `json:"hint"`
}

type nodeI18n struct {
	Label       string             `json:"label"`
	Description string             `json:"description"`
	Input       map[string]pinI18n `json:"input"`
	Output      map[string]pinI18n `json:"output"`
}

// BuildWithI18n 在 Build() 结构基础上按 kind JOIN 进展示文案 (label/description/pin label+hint),
// 供 list_nodes 给 LLM 更全语境。zh.ts 缺某 kind 的文案则该字段留空 (drift guard 测试会兜)。
func BuildWithI18n() []Node {
	out := Build()
	var i18n map[string]nodeI18n
	if err := json.Unmarshal(nodeI18nJSON, &i18n); err != nil {
		panic("catalog: node-i18n.json 解析失败: " + err.Error())
	}
	for i := range out {
		n := &out[i]
		t, ok := i18n[n.Kind]
		if !ok {
			continue
		}
		n.Label = t.Label
		n.Description = t.Description
		for j := range n.Inputs {
			if p, ok := t.Input[n.Inputs[j].Name]; ok {
				n.Inputs[j].Label = p.Label
				n.Inputs[j].Hint = p.Hint
			}
		}
		for j := range n.Outputs {
			if p, ok := t.Output[n.Outputs[j].Name]; ok {
				n.Outputs[j].Label = p.Label
			}
		}
	}
	return out
}
