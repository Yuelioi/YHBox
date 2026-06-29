package catalog

import (
	"fmt"
	"strings"
)

// Markdown 把 BuildWithI18n() 的目录渲染成人读 Markdown 速查表 (按 category 分组,
// 每节点列输入/输出表, 出口表的"携带数据"列展开 exec-data 字段)。纯字符串拼接,
// 顺序稳定 (Build 已按 category→kind 排序)。供 cmd/node-catalog --md 与 task nodes 用。
func Markdown(nodes []Node) string {
	var b strings.Builder
	b.WriteString("# 节点速查表\n\n")
	fmt.Fprintf(&b, "共 %d 节点。源: `go run ./cmd/node-catalog export --md`（= `task nodes`）。\n", len(nodes))

	var cat string
	for _, n := range nodes {
		if n.Category != cat {
			cat = n.Category
			b.WriteString("\n## " + cat + "\n")
		}
		writeNode(&b, n)
	}
	return b.String()
}

func writeNode(b *strings.Builder, n Node) {
	b.WriteString("\n### " + n.Kind)
	if n.Label != "" {
		b.WriteString(" — " + n.Label)
	}
	b.WriteString("\n")

	if marks := nodeMarks(n); marks != "" {
		b.WriteString("`" + marks + "`\n\n")
	}
	if n.Description != "" {
		b.WriteString(n.Description + "\n\n")
	}
	if n.Example != "" {
		b.WriteString("示例: " + n.Example + "\n\n")
	}

	writeInputs(b, n.Inputs)
	writeOutputs(b, n.Outputs)
}

func nodeMarks(n Node) string {
	var m []string
	if n.NeedsTarget {
		m = append(m, "NeedsTarget")
	}
	if len(n.TargetCapabilities) > 0 {
		m = append(m, "Caps:"+strings.Join(n.TargetCapabilities, ","))
	}
	if len(n.SupportedTargets) > 0 {
		m = append(m, "Supports:"+strings.Join(n.SupportedTargets, ","))
	}
	if n.NeedsWindow {
		m = append(m, "NeedsWindow")
	}
	if n.IsPureData {
		m = append(m, "PureData")
	}
	return strings.Join(m, " · ")
}

func writeInputs(b *strings.Builder, pins []Pin) {
	if len(pins) == 0 {
		return
	}
	b.WriteString("输入:\n\n")
	b.WriteString("| pin | 类型 | 必填 | 默认 | 说明 |\n|---|---|---|---|---|\n")
	for _, p := range pins {
		req := ""
		if p.Required {
			req = "✓"
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
			p.Name, p.Type, req, fmtDefault(p.Default), cellEscape(p.Hint))
	}
	b.WriteString("\n")
}

func writeOutputs(b *strings.Builder, pins []Pin) {
	if len(pins) == 0 {
		return
	}
	b.WriteString("输出:\n\n")
	b.WriteString("| 出口 | 类型 | 携带数据 |\n|---|---|---|\n")
	for _, p := range pins {
		fmt.Fprintf(b, "| %s | %s | %s |\n", p.Name, p.Type, fmtPinData(p.Data))
	}
	b.WriteString("\n")
}

// fmtPinData 渲染出口携带的 Data 字段为 "Center(Point), Count(Number)"; 空则 "—"。
func fmtPinData(data []PinData) string {
	if len(data) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(data))
	for _, d := range data {
		s := d.Name + "(" + d.Type + ")"
		if d.Optional {
			s += "?"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func fmtDefault(v any) string {
	if v == nil {
		return "—"
	}
	return cellEscape(fmt.Sprintf("%v", v))
}

// cellEscape 把会破坏表格的字符转义 (管道符 / 换行)。
func cellEscape(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
