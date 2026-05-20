// install-fishing-v2 把 bin/data/library/subgraphs/fishing-v2/ 下的 fishing-v2.json 主容器
// + 13 个 helper/state subgraph 安装到 bin/data/containers/fishing-v2/, GUI 启动后直接显示
// "fishing-v2" 容器可运行.
//
// 用法:
//   go run ./cmd/install-fishing-v2          # 仅替换 fishing-v2/, 保留其他 user container
//   go run ./cmd/install-fishing-v2 --force  # 额外清空 bin/data/containers/* 所有其他容器
//
// 流程 (validate-first):
//   1. 从 library 读 main + sibling subgraphs, 合并成 in-memory Container 并 validate
//   2. validate 失败 → 立刻退出, 不动 bin/data/containers/
//   3. validate 通过 → 删 fishing-v2/ (或 --force 时删全部), 再写新版
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yhbox/internal/services/container"
)

const (
	libraryDir    = "bin/data/library/subgraphs/fishing-v2"
	containersDir = "bin/data/containers"
	mainID        = "fishing-v2"
)

func main() {
	force := flag.Bool("force", false, "wipe ALL bin/data/containers/* (default: only fishing-v2)")
	flag.Parse()
	if err := run(*force); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func run(force bool) error {
	// Step 1: load library main + sibling subgraphs into memory
	mainObj, subgraphPayloads, err := loadLibrary()
	if err != nil {
		return fmt.Errorf("load library: %w", err)
	}

	// Step 2: validate BEFORE touching bin/data/containers/
	if err := validateInMemory(mainObj, subgraphPayloads); err != nil {
		return fmt.Errorf("pre-install validation: %w", err)
	}
	fmt.Printf("✅ library validation passed (%d subgraphs)\n", len(subgraphPayloads))

	// Step 3: wipe (selective by default, --force = global)
	if err := wipeContainers(force); err != nil {
		return err
	}

	// Step 4: write container.json + subgraphs/
	now := time.Now().UTC().Format(time.RFC3339)
	mainObj["createdAt"] = now
	mainObj["updatedAt"] = now

	containerDir := filepath.Join(containersDir, mainID)
	subgraphsDir := filepath.Join(containerDir, "subgraphs")
	if err := os.MkdirAll(subgraphsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	containerJSON := filepath.Join(containerDir, "container.json")
	out, err := json.MarshalIndent(mainObj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal container: %w", err)
	}
	if err := os.WriteFile(containerJSON, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", containerJSON, err)
	}
	fmt.Printf("✅ %s\n", containerJSON)

	for sgID, payload := range subgraphPayloads {
		dst := filepath.Join(subgraphsDir, sgID+".json")
		if err := os.WriteFile(dst, payload, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		fmt.Printf("✅ %s\n", dst)
	}

	fmt.Printf("\n🎉 Installed fishing-v2 container with %d subgraphs.\n", len(subgraphPayloads))
	fmt.Printf("   重启 GUI / task dev 即可看到 'fishing-v2' 容器.\n")
	return nil
}

// loadLibrary 读 library main + sibling subgraphs, 返回 main map + (sgID → raw bytes) map.
// raw bytes 用于 step 4 写盘 (避免 unmarshal → marshal 丢字段顺序).
func loadLibrary() (map[string]any, map[string][]byte, error) {
	mainPath := filepath.Join(libraryDir, "fishing-v2.json")
	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read main %s: %w", mainPath, err)
	}
	var mainObj map[string]any
	if err := json.Unmarshal(mainBytes, &mainObj); err != nil {
		return nil, nil, fmt.Errorf("parse main: %w", err)
	}
	delete(mainObj, "subgraphs") // sibling files are SoT

	subgraphs := make(map[string][]byte)
	libEntries, err := os.ReadDir(libraryDir)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", libraryDir, err)
	}
	for _, ent := range libEntries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		if ent.Name() == "fishing-v2.json" {
			continue
		}
		src := filepath.Join(libraryDir, ent.Name())
		b, err := os.ReadFile(src)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", src, err)
		}
		var sg map[string]any
		if err := json.Unmarshal(b, &sg); err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", src, err)
		}
		sgID, _ := sg["id"].(string)
		if sgID == "" {
			return nil, nil, fmt.Errorf("%s: missing id", src)
		}
		subgraphs[sgID] = b
	}
	return mainObj, subgraphs, nil
}

// validateInMemory 把 main + subgraphs 合并成 container.Container 跑 validator.
// 出现 SeverityError → 返 error (上层报错退出, 不动 bin/data/containers/).
func validateInMemory(mainObj map[string]any, subgraphPayloads map[string][]byte) error {
	mainBytes, err := json.Marshal(mainObj)
	if err != nil {
		return fmt.Errorf("re-marshal main: %w", err)
	}
	var c container.Container
	if err := json.Unmarshal(mainBytes, &c); err != nil {
		return fmt.Errorf("parse main as Container: %w", err)
	}
	for sgID, b := range subgraphPayloads {
		var sg container.Subgraph
		if err := json.Unmarshal(b, &sg); err != nil {
			return fmt.Errorf("parse subgraph %s: %w", sgID, err)
		}
		c.Subgraphs = append(c.Subgraphs, sg)
	}
	errs := container.ValidateContainer(&c)
	hasErr := false
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			fmt.Printf("  ❌ %s @ %s/%s: %s\n", e.Code, strings.Join(e.GraphPath, "/"), e.NodeID, e.Message)
			hasErr = true
		}
	}
	if hasErr {
		return fmt.Errorf("validation has errors (see above)")
	}
	return nil
}

// wipeContainers --force → 删 bin/data/containers/*; 否则仅删 bin/data/containers/fishing-v2/.
func wipeContainers(force bool) error {
	if force {
		entries, err := os.ReadDir(containersDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read %s: %w", containersDir, err)
		}
		for _, ent := range entries {
			path := filepath.Join(containersDir, ent.Name())
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("rm %s: %w", path, err)
			}
			fmt.Printf("🗑  rm %s\n", path)
		}
		return nil
	}
	// default: only fishing-v2
	target := filepath.Join(containersDir, mainID)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("stat %s: %w", target, err)
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("rm %s: %w", target, err)
	}
	fmt.Printf("🗑  rm %s (other user containers preserved; use --force to wipe all)\n", target)
	return nil
}
