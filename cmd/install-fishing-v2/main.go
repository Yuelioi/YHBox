// install-fishing-v2 把 bin/data/library/subgraphs/fishing-v2/ 下的 fishing-v2.json 主容器
// + 13 个 helper/state subgraph 安装到 bin/data/containers/fishing-v2/, GUI 启动后直接显示
// "fishing-v2" 容器可运行.
//
// 用法: go run ./cmd/install-fishing-v2
//
// 行为 (破坏性):
//   1. 删 bin/data/containers/* 所有现有 user container (清理 fishing v1 / 历史损坏数据)
//   2. 创建 bin/data/containers/fishing-v2/container.json (从 library fishing-v2.json 复制)
//   3. 创建 bin/data/containers/fishing-v2/subgraphs/<sg-id>.json (13 个 helper/state subgraph)
package main

import (
	"encoding/json"
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
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Step 1: wipe existing user containers
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

	// Step 2: read library fishing-v2.json + sibling subgraphs
	mainPath := filepath.Join(libraryDir, "fishing-v2.json")
	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("read main %s: %w", mainPath, err)
	}
	var mainObj map[string]any
	if err := json.Unmarshal(mainBytes, &mainObj); err != nil {
		return fmt.Errorf("parse main: %w", err)
	}
	// strip "subgraphs":[] from container.json (it's loaded from sibling files at runtime)
	delete(mainObj, "subgraphs")
	now := time.Now().UTC().Format(time.RFC3339)
	mainObj["createdAt"] = now
	mainObj["updatedAt"] = now

	// Step 3: write container.json + subgraphs/
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

	// copy each sibling subgraph
	libEntries, err := os.ReadDir(libraryDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", libraryDir, err)
	}
	copied := 0
	for _, ent := range libEntries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		if ent.Name() == "fishing-v2.json" {
			continue // main, already written
		}
		src := filepath.Join(libraryDir, ent.Name())
		b, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read %s: %w", src, err)
		}
		// parse → check id matches filename (sanity), keep payload as-is
		var sg map[string]any
		if err := json.Unmarshal(b, &sg); err != nil {
			return fmt.Errorf("parse %s: %w", src, err)
		}
		sgID, _ := sg["id"].(string)
		if sgID == "" {
			return fmt.Errorf("%s: missing id", src)
		}
		dst := filepath.Join(subgraphsDir, sgID+".json")
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		copied++
		fmt.Printf("✅ %s\n", dst)
	}

	fmt.Printf("\n🔍 验证安装的 container...\n")
	if err := validateInstalled(containerDir); err != nil {
		return fmt.Errorf("post-install validation: %w", err)
	}
	fmt.Printf("\n🎉 Installed fishing-v2 container with %d subgraphs, 0 validation errors.\n", copied)
	fmt.Printf("   重启 GUI / task dev 即可看到 'fishing-v2' 容器.\n")
	return nil
}

func validateInstalled(containerDir string) error {
	b, err := os.ReadFile(filepath.Join(containerDir, "container.json"))
	if err != nil {
		return fmt.Errorf("read container.json: %w", err)
	}
	var c container.Container
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("parse container.json: %w", err)
	}
	sgDir := filepath.Join(containerDir, "subgraphs")
	entries, err := os.ReadDir(sgDir)
	if err != nil {
		return fmt.Errorf("read subgraphs/: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		sgB, err := os.ReadFile(filepath.Join(sgDir, e.Name()))
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		var sg container.Subgraph
		if err := json.Unmarshal(sgB, &sg); err != nil {
			return fmt.Errorf("parse %s: %w", e.Name(), err)
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
