package dependency

import (
	"fmt"

	nodepkg "yotta/internal/node"
)

// NodeInfo 节点的最小描述, 供 ScanSubgraphDependencies 迭代. 跟 container.GraphNode 解耦
// (避免 dependency → container → … 闭环), 由 caller 适配.
type NodeInfo struct {
	Kind   string
	Config map[string]any
}

// ScanSubgraphDependencies 递归扫子图依赖. BFS, dedupe, cyclic-safe.
//
// getNodes: 给定 sgID 返回该子图的全部节点; 不存在时返 nil, nil — 跳过, 不报错.
// 节点 kind → deps 走 nodepkg.Get(kind).Dependencies(NewInputsFromConfig(cfg)) — 单一源
// 跟 framework Dependencies(in) 接口对齐. 节点未注册 (kind 未知) 或未实现 Dependencer 接口
// → 0 依赖.
//
// 返 []Dependency 含 root subgraph 自身 + 所有 transitively 引用的 subgraph / template / clip.
func ScanSubgraphDependencies(
	rootSgID string,
	getNodes func(sgID string) ([]NodeInfo, error),
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
			rn, ok := nodepkg.Get(n.Kind)
			if !ok || rn.Dependencies == nil {
				continue
			}
			in := nodepkg.NewInputsFromConfig(n.Config)
			for _, raw := range rn.Dependencies(in) {
				d := Dependency{Kind: Kind(raw.Kind), Key: raw.Key}
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
