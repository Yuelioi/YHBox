package dependency

import "fmt"

// Extractor 节点种类的依赖提取器. specs/*.go init() 通过 dependency.RegisterExtractor 注册.
// cfg 是节点的 Config map (map[string]any). 用 Config 而不是整个 GraphNode
// 避免 specs → container → specs 循环导入.
type Extractor interface {
	Extract(cfg map[string]any) []Dependency
}

// NodeInfo 节点的最小描述, 供 ScanSubgraphDependencies 迭代. 不引用 container 包
// (container → specs → dependency → container 会循环).
type NodeInfo struct {
	Kind   string
	Config map[string]any
}

// ScanSubgraphDependencies 递归扫子图依赖. BFS, dedupe, cyclic-safe.
// getNodes: 给定 sgID 返回该子图的全部节点; 不存在时返 nil, nil — 跳过, 不报错.
// getExtractor: 节点 kind → Extractor (生产代码用 dependency.GetExtractor).
//
// 返 []Dependency 含 root subgraph 自身 + 所有 transitively 引用的 subgraph / template / clip.
func ScanSubgraphDependencies(
	rootSgID string,
	getNodes func(sgID string) ([]NodeInfo, error),
	getExtractor func(kind string) Extractor,
) ([]Dependency, error) {
	visited := map[string]bool{}
	seenDeps := map[string]bool{}
	var allDeps []Dependency
	queue := []string{rootSgID}

	for len(queue) > 0 {
		sgID := queue[0]
		queue = queue[1:]
		if visited[sgID] {
			continue
		}
		visited[sgID] = true

		// 自身也是 KindSubgraph 依赖
		selfDep := Dependency{Kind: KindSubgraph, Key: sgID}
		if !seenDeps[selfDep.String()] {
			seenDeps[selfDep.String()] = true
			allDeps = append(allDeps, selfDep)
		}

		nodes, err := getNodes(sgID)
		if err != nil {
			return nil, fmt.Errorf("get subgraph %q: %w", sgID, err)
		}
		for _, n := range nodes {
			ext := getExtractor(n.Kind)
			if ext == nil {
				continue
			}
			for _, d := range ext.Extract(n.Config) {
				if seenDeps[d.String()] {
					continue
				}
				seenDeps[d.String()] = true
				allDeps = append(allDeps, d)
				if d.Kind == KindSubgraph {
					queue = append(queue, d.Key)
				}
			}
		}
	}
	return allDeps, nil
}

// ScanSubgraphDependenciesWithExtractors 测试 helper, 用 map 替代 getExtractor 函数.
func ScanSubgraphDependenciesWithExtractors(
	rootSgID string,
	getNodes func(sgID string) ([]NodeInfo, error),
	extractors map[string]Extractor,
) ([]Dependency, error) {
	return ScanSubgraphDependencies(rootSgID, getNodes, func(k string) Extractor {
		return extractors[k]
	})
}
