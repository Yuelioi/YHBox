// node-catalog — LLM-facing node catalog export + 任意 container.json 校验.
//
// spike 工具: 验证"给 LLM 节点目录 + 图 schema → 生成 container.json → 校验修复回路"是否成立.
//
// 用法:
//
//	go run ./cmd/node-catalog export            # 导出全节点目录 (JSON, 带大白话+出口Data, stdout)
//	go run ./cmd/node-catalog export --md       # 同上但渲染成人读 Markdown 速查表
//	go run ./cmd/node-catalog pins              # 全节点 pin 名按名合并 → 命名参考表 (Markdown) + 命名分裂告警
//	go run ./cmd/node-catalog validate <path>   # 对一个 container.json 跑 ValidateContainer
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"yotta/internal/catalog"
	"yotta/internal/services/container"

	// Anonymous imports — 触发 init() 节点注册.
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/event" // EventTick (listener-driven 定时触发)
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/collection" // Split/Join/List* 列表节点
	_ "yotta/internal/nodes/random" // RandomInt/RandomFloat/RandomBool
	_ "yotta/internal/nodes/script" // Script (内嵌 JS, goja)
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: node-catalog export [--md] | pins | validate <path>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "export":
		md := len(os.Args) > 2 && os.Args[2] == "--md"
		doExport(md)
	case "pins":
		doPins()
	case "validate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: node-catalog validate <container.json>")
			os.Exit(2)
		}
		doValidate(os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(2)
	}
}

func doExport(md bool) {
	if md {
		fmt.Print(catalog.Markdown(catalog.BuildWithI18n()))
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(catalog.BuildWithI18n()); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(2)
	}
}

func doValidate(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(2)
	}
	var c container.Container
	if err := json.Unmarshal(data, &c); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal: %v\n", err)
		os.Exit(2)
	}
	c.Normalize()
	errs := container.ValidateContainer(&c)
	if len(errs) == 0 {
		fmt.Println("✅ All clean — 0 validation errors")
		return
	}
	sort.SliceStable(errs, func(i, j int) bool {
		if errs[i].Severity != errs[j].Severity {
			return errs[i].Severity > errs[j].Severity
		}
		return errs[i].Code < errs[j].Code
	})
	hasError := false
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			hasError = true
		}
		gp := strings.Join(e.GraphPath, "/")
		if gp == "" {
			gp = "(root)"
		}
		var msg string
		if len(e.Params) > 0 {
			b, _ := json.Marshal(e.Params)
			msg = string(b)
		}
		fmt.Printf("- [%s] %s @ %s/%s — %s\n", e.Severity, e.Code, gp, e.NodeID, msg)
	}
	if hasError {
		os.Exit(1)
	}
}

// pinAgg — 一个 pin 名在全节点里的合并视图: 见过的类型集 / 用它的节点集 / 首个非空文案。
type pinAgg struct {
	types map[string]struct{}
	nodes map[string]struct{}
	label string
}

func newPinMap() map[string]*pinAgg { return map[string]*pinAgg{} }

func addPin(m map[string]*pinAgg, name, typ, label, kind string) {
	a := m[name]
	if a == nil {
		a = &pinAgg{types: map[string]struct{}{}, nodes: map[string]struct{}{}}
		m[name] = a
	}
	if typ != "" {
		a.types[typ] = struct{}{}
	}
	if label != "" && a.label == "" {
		a.label = label
	}
	a.nodes[kind] = struct{}{}
}

func sortedSet(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedPinKeys(m map[string]*pinAgg) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// doPins — 遍历全节点 spec, 把 pin 名按用途分组、按名合并, 输出命名参考表 + 命名分裂告警。
func doPins() {
	nodes := catalog.BuildWithI18n()

	inParam := newPinMap() // 输入参数 (非 exec): 真正填值的 config pin
	inExec := newPinMap()  // 输入 exec 口 (流程入口, 通常 In)
	outExec := newPinMap() // 出口 exec (分支: Found/Done/Fail/Timeout…)
	outVal := newPinMap()  // 纯数据输出 (Expr/Add… 的 Result 等)
	data := newPinMap()    // exec 出口携带的数据字段 (exec-data)

	for _, n := range nodes {
		for _, p := range n.Inputs {
			if p.Exec {
				addPin(inExec, p.Name, p.Type, p.Label, n.Kind)
			} else {
				addPin(inParam, p.Name, p.Type, p.Label, n.Kind)
			}
		}
		for _, p := range n.Outputs {
			if p.Exec {
				addPin(outExec, p.Name, "", p.Label, n.Kind)
			} else {
				addPin(outVal, p.Name, p.Type, p.Label, n.Kind)
			}
			for _, d := range p.Data {
				addPin(data, d.Name, d.Type, "", n.Kind)
			}
		}
	}

	var b strings.Builder
	b.WriteString("# 节点 pin 命名参考 (合并)\n\n")
	fmt.Fprintf(&b, "> `task nodes:pins` 生成, 共 %d 个节点。加新节点时先查此表对齐命名, 别造同义新词。\n\n", len(nodes))

	render := func(title, note string, m map[string]*pinAgg, showType bool) {
		fmt.Fprintf(&b, "## %s\n\n", title)
		if note != "" {
			b.WriteString(note + "\n\n")
		}
		if len(m) == 0 {
			b.WriteString("(无)\n\n")
			return
		}
		if showType {
			b.WriteString("| pin | 类型 | 文案 | 用量 | 用于节点 |\n|---|---|---|---|---|\n")
		} else {
			b.WriteString("| pin | 文案 | 用量 | 用于节点 |\n|---|---|---|---|\n")
		}
		for _, name := range sortedPinKeys(m) {
			a := m[name]
			ns := sortedSet(a.nodes)
			if showType {
				fmt.Fprintf(&b, "| `%s` | %s | %s | %d | %s |\n",
					name, strings.Join(sortedSet(a.types), " / "), a.label, len(ns), strings.Join(ns, ", "))
			} else {
				fmt.Fprintf(&b, "| `%s` | %s | %d | %s |\n",
					name, a.label, len(ns), strings.Join(ns, ", "))
			}
		}
		b.WriteString("\n")
	}

	render("输入参数 (config)", "填值的参数 pin —— 新节点的同类参数请复用这里已有的名。", inParam, true)
	render("输入 exec 口", "流程入口 (约定统一 `In`)。", inExec, false)
	render("出口 exec", "分支出口 (成功/失败/超时/循环体…)。", outExec, false)
	render("出口数据字段 (exec-data)", "exec 出口携带、下游同名 data wire 接收的字段。", data, true)
	render("纯数据输出", "纯函数节点的值输出。", outVal, true)

	// 命名分裂告警 — 检测逻辑在 catalog.DetectNameSplits (guard 测试 TestNoPinNameSplit 共用)。
	b.WriteString("## ⚠ 命名分裂告警\n\n")
	if splits := catalog.DetectNameSplits(); len(splits) == 0 {
		b.WriteString("(无)\n")
	} else {
		for _, s := range splits {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}

	fmt.Print(b.String())
}
