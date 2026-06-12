// validate-fishing-v2 加载 bin/data/containers/fishing-v2/container.json + 全局子图池
// bin/data/subgraphs/*.json (2026-06-12 全局化布局) 并跑 container ValidateContainer.
//
// 用法: go run ./cmd/validate-fishing-v2
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"yotta/internal/services/container"

	// Anonymous imports — 触发 nodepkg 节点注册, 让 validator nodepkg union 在 validate path 生效.
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/event"     // EventTick (listener-driven 定时触发)
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/collection" // Split/Join/List* 列表节点
	_ "yotta/internal/nodes/random" // RandomInt/RandomFloat/RandomBool
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

const fishingV2Dir = "bin/data/containers/fishing-v2"

func main() {
	containerPath := filepath.Join(fishingV2Dir, "container.json")
	data, err := os.ReadFile(containerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read container: %v\n", err)
		os.Exit(2)
	}
	var c container.Container
	if err := json.Unmarshal(data, &c); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal container: %v\n", err)
		os.Exit(2)
	}

	// 全局子图池 (粗粒度: 全池都给 validator — known 集合语义, 工具不必精确闭包).
	subgraphsDir := "bin/data/subgraphs"
	entries, err := os.ReadDir(subgraphsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read subgraphs dir: %v\n", err)
		os.Exit(2)
	}
	var subgraphs []container.Subgraph
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(subgraphsDir, e.Name())
		sgData, rerr := os.ReadFile(path)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, rerr)
			os.Exit(2)
		}
		var sg container.Subgraph
		if err := json.Unmarshal(sgData, &sg); err != nil {
			fmt.Fprintf(os.Stderr, "unmarshal subgraph %s: %v\n", path, err)
			os.Exit(2)
		}
		container.NormalizeSubgraph(&sg)
		subgraphs = append(subgraphs, sg)
	}
	sort.Slice(subgraphs, func(i, j int) bool { return subgraphs[i].ID < subgraphs[j].ID })

	c.Normalize()

	errs := container.ValidateContainer(&c, subgraphs)
	if len(errs) == 0 {
		fmt.Println("✅ All clean — 0 validation errors")
		return
	}
	// 按 severity 排
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
		path := strings.Join(e.GraphPath, "/")
		if path == "" {
			path = "(root)"
		}
		var msg string
		if len(e.Params) > 0 {
			b, _ := json.Marshal(e.Params)
			msg = string(b)
		}
		fmt.Printf("- [%s] %s @ %s/%s — %s\n", e.Severity, e.Code, path, e.NodeID, msg)
	}
	if hasError {
		os.Exit(1)
	}
}
