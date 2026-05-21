package dependency

import (
	"fmt"

	"yhbox/internal/services/container"
)

// Extractor 节点种类的依赖提取器. 由 nodekind.Spec 持. 测试时也可注入 mock.
type Extractor interface {
	Extract(node *container.GraphNode) []Dependency
}

// ScanSubgraphDependencies 递归扫子图依赖. BFS, dedupe, cyclic-safe.
// getSubgraph: 给定 sgID 返 Subgraph (nil 或 not-found 时返 nil, nil — 跳过, 不报错).
// getExtractor: 节点 kind → Extractor (生产代码从 nodekind.GetSpec(kind).DependencyExtractor 取).
//
// 返 []Dependency 含 root subgraph 自身 + 所有 transitively 引用的 subgraph / template / clip.
func ScanSubgraphDependencies(
	rootSgID string,
	getSubgraph func(string) (*container.Subgraph, error),
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

		sg, err := getSubgraph(sgID)
		if err != nil {
			return nil, fmt.Errorf("get subgraph %q: %w", sgID, err)
		}
		if sg == nil {
			continue
		}
		for i := range sg.Graph.Nodes {
			n := &sg.Graph.Nodes[i]
			ext := getExtractor(n.Kind)
			if ext == nil {
				continue
			}
			for _, d := range ext.Extract(n) {
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
	getSubgraph func(string) (*container.Subgraph, error),
	extractors map[string]Extractor,
) ([]Dependency, error) {
	return ScanSubgraphDependencies(rootSgID, getSubgraph, func(k string) Extractor {
		return extractors[k]
	})
}
