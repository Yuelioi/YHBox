// cmd/migrate-templates — 一次性数据迁移: 模板节点的单 key 字段 Template → 列表 Templates.
//
// 背景: WaitTemplate/ClickTemplate/CheckTemplate 的模板字段从单 key 升成多模板
// 列表 (config.literal.Template: "x" → config.literal.Templates: ["x"]) + MatchMode(any/all).
// MatchMode 不写入数据 (节点 Spec Default="any" 兜底). 跑一次即可, 之后删本 cmd.
//
// 用法: go run ./cmd/migrate-templates [dataDir]   (默认 bin/data)
//   - 递归扫 <dataDir>/containers/**/*.json
//   - 改动的文件原地重写 (2-space indent)
//   - 幂等: 已是 Templates 的节点跳过
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var templateKinds = map[string]bool{
	"WaitTemplate": true, "ClickTemplate": true, "CheckTemplate": true,
}

func main() {
	dataDir := "bin/data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}
	root := filepath.Join(dataDir, "containers")

	var scanned, changedFiles, migratedNodes int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		scanned++
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			fmt.Printf("  skip (not JSON object): %s\n", path)
			return nil
		}
		n := migrate(doc)
		if n == 0 {
			return nil
		}
		out, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %s: %w", path, err)
		}
		out = append(out, '\n')
		if err := os.WriteFile(path, out, info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		changedFiles++
		migratedNodes += n
		fmt.Printf("  migrated %d node(s): %s\n", n, path)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("done. scanned %d files, changed %d files, migrated %d template node(s).\n",
		scanned, changedFiles, migratedNodes)
}

// migrate 递归找每个有 "kind" 的对象, 模板节点就把 config.literal.Template → Templates.
// 返回本 doc 内迁移的节点数.
func migrate(v any) int {
	count := 0
	switch t := v.(type) {
	case map[string]any:
		if kind, ok := t["kind"].(string); ok && templateKinds[kind] {
			count += migrateNode(t)
		}
		for _, child := range t {
			count += migrate(child)
		}
	case []any:
		for _, child := range t {
			count += migrate(child)
		}
	}
	return count
}

// migrateNode 把单节点 config.literal.Template(string) 转成 Templates([]string).
func migrateNode(node map[string]any) int {
	cfg, ok := node["config"].(map[string]any)
	if !ok {
		return 0
	}
	lit, ok := cfg["literal"].(map[string]any)
	if !ok {
		return 0
	}
	if _, already := lit["Templates"]; already {
		return 0 // 幂等: 已迁移
	}
	tpl, ok := lit["Template"].(string)
	if !ok {
		return 0 // 没有单值 Template (或非 string) — 无可迁
	}
	if tpl == "" {
		lit["Templates"] = []any{}
	} else {
		lit["Templates"] = []any{tpl}
	}
	delete(lit, "Template")
	return 1
}
