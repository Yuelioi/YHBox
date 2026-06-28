// Package catalog 把 node.All() 注册表提炼成对 LLM / 文档友好的节点目录 (只结构,
// 展示文本 i18n 是 fast-follow)。cmd/node-catalog (CLI) 和 cmd/yotta-mcp (MCP) 共享本包。
package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"yotta/internal/node"
)

// PinData — exec 出口携带的一个数据字段 (下游 exec-data wire 收同名字段)。
// 镜像 node.DataField; 只有 output 出口会填, input pin 留空。
type PinData struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional,omitempty"`
}

type Pin struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Exec     bool   `json:"exec,omitempty"`
	Required bool   `json:"required,omitempty"`
	Advanced bool   `json:"advanced,omitempty"`
	Default  any    `json:"default,omitempty"`
	// Data 仅 output exec 出口填 (input pin 留空 → omitempty 不输出)。
	Data []PinData `json:"data,omitempty"`
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
	Example     string `json:"example,omitempty"`
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
			p := Pin{Name: o.Name, Type: o.Type, Exec: o.Type == node.TypeExec}
			for _, d := range o.Data {
				p.Data = append(p.Data, PinData{Name: d.Name, Type: d.Type, Optional: d.Optional})
			}
			cn.Outputs = append(cn.Outputs, p)
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
	Example     string             `json:"example"`
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
		n.Example = t.Example
		for j := range n.Inputs {
			if p, ok := t.Input[n.Inputs[j].Name]; ok {
				n.Inputs[j].Label = p.Label
				n.Inputs[j].Hint = p.Hint
			}
			if n.Inputs[j].Label == "" && n.Inputs[j].Exec {
				n.Inputs[j].Label = "执行"
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

// DetectNameSplits 扫全节点 pin 名, 返回"命名分裂"的人读描述 (空 slice = 命名干净):
//   - 形近撞名: 规范化 (小写去符号) 后同 key 却有多种拼写 (Roi vs ROI)。
//   - 同名不同具体类型: 一个 pin 名 (限输入参数 / exec-out Data 字段) 出现 ≥2 种具体类型;
//     带 `*` (通配) 的是有意泛型 pin (Add/Eq 的 A/B 等), 不算。
//
// cmd/node-catalog pins 的告警段 + guard 测试 TestNoPinNameSplit 共用本函数 —— 单一真相源。
func DetectNameSplits() []string {
	nodes := Build()
	names := map[string]struct{}{}                 // 所有 pin 名 → 形近撞名
	inputTypes := map[string]map[string]struct{}{} // 输入参数: 名 → 类型集 (含 *)
	dataTypes := map[string]map[string]struct{}{}  // exec-out Data 字段: 名 → 类型集 (含 *)
	note := func(m map[string]map[string]struct{}, name, typ string) {
		if typ == "" {
			return
		}
		if m[name] == nil {
			m[name] = map[string]struct{}{}
		}
		m[name][typ] = struct{}{}
	}
	for _, n := range nodes {
		for _, p := range n.Inputs {
			names[p.Name] = struct{}{}
			if !p.Exec {
				note(inputTypes, p.Name, p.Type)
			}
		}
		for _, o := range n.Outputs {
			names[o.Name] = struct{}{}
			for _, d := range o.Data {
				names[d.Name] = struct{}{}
				note(dataTypes, d.Name, d.Type)
			}
		}
	}

	var out []string

	// 形近撞名。
	byNorm := map[string][]string{}
	for name := range names {
		k := normPinName(name)
		byNorm[k] = append(byNorm[k], name)
	}
	for _, k := range sortedStrSliceKeys(byNorm) {
		v := byNorm[k]
		if len(v) > 1 {
			sort.Strings(v)
			for i := range v {
				v[i] = "`" + v[i] + "`"
			}
			out = append(out, "形近撞名: "+strings.Join(v, " vs ")+" —— 选一个统一")
		}
	}

	// 同名不同具体类型: 按角色 (输入参数 / Data 字段) 各自查, 不跨角色 —— Loop.Count(输入) 与
	// DetectColor.Count(数据) 是两个概念不算撞。含 `*` (通配) 的是有意泛型 pin, 跳过。
	for _, grp := range []struct {
		label string
		m     map[string]map[string]struct{}
	}{{"输入参数", inputTypes}, {"数据字段", dataTypes}} {
		keys := make([]string, 0, len(grp.m))
		for name := range grp.m {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			ts := grp.m[name]
			if _, wild := ts["*"]; wild {
				continue
			}
			if len(ts) > 1 {
				out = append(out, fmt.Sprintf("同名不同类型 [%s] `%s`: %s —— 确认是否该统一",
					grp.label, name, strings.Join(sortedSetKeys(ts), " / ")))
			}
		}
	}
	return out
}

// normPinName — 小写 + 去非字母数字, 用于把 Roi / ROI / R_O_I 归到同一 key 比对拼写。
func normPinName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func sortedSetKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedStrSliceKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
