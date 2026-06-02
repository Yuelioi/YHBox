// Package catalog 把 node.All() 注册表提炼成对 LLM / 文档友好的节点目录 (只结构,
// 展示文本 i18n 是 fast-follow)。cmd/node-catalog (CLI) 和 cmd/yotta-mcp (MCP) 共享本包。
package catalog

import (
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
}

type Node struct {
	Kind        string `json:"kind"`
	Category    string `json:"category"`
	NeedsWindow bool   `json:"needsWindow,omitempty"`
	IsPureData  bool   `json:"isPureData,omitempty"`
	Inputs      []Pin  `json:"inputs"`
	Outputs     []Pin  `json:"outputs"`
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
