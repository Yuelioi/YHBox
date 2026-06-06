// node-catalog — LLM-facing node catalog export + 任意 container.json 校验.
//
// spike 工具: 验证"给 LLM 节点目录 + 图 schema → 生成 container.json → 校验修复回路"是否成立.
//
// 用法:
//
//	go run ./cmd/node-catalog export            # 导出全节点目录 (JSON, 带大白话+出口Data, stdout)
//	go run ./cmd/node-catalog export --md       # 同上但渲染成人读 Markdown 速查表
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
	_ "yotta/internal/nodes/event"     // EventTick (listener-driven 定时触发)
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: node-catalog export [--md] | validate <path>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "export":
		md := len(os.Args) > 2 && os.Args[2] == "--md"
		doExport(md)
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
