// closure.go — 依赖解析单一咽喉的便利出口 (2026-06-12 全局化).
//
// 两层结构: 节点级依赖提取 (scanner.go 的 emit, 单一实现) 是 primitive; 其上:
//   - Closure (正向): 根 → 子图 ID 集 + 资产 GUID 集. 运行时 bundle / validator /
//     未来跨机打包 共用, 不许各自内联递归.
//   - 反向 referrer (谁引用了 X) 在 wire 层 (有容器 store 访问权), 用同一 primitive
//     遍历全部根后匹配, 返回引用位置列表 — 不经 ClosureResult.
//
// 接口不变量: 每个 SubgraphID 至多访问一次 (visited 集是契约不是实现细节 —
// 防未来改实现导致导出/校验在循环引用上无限递归).
package dependency

// ClosureResult 正向闭包结果. 结构体而非裸集合 — 未来加维度 (vars/blob...) 不改签名.
type ClosureResult struct {
	SubgraphIDs []string // 传递闭包内全部子图 ID (BFS 序, 去重)
	AssetGUIDs  []string // template + clip GUID 统一 (去重)
}

// Closure 从一组根节点出发解析正向依赖闭包.
// getNodes: sgID → 该子图节点集; 不存在返 (nil, nil) — 跳过 (悬空引用由 validator 报).
func Closure(rootNodes []NodeInfo, getNodes func(sgID string) ([]NodeInfo, error)) (ClosureResult, error) {
	deps, err := ScanContainerDependencies(rootNodes, getNodes)
	if err != nil {
		return ClosureResult{}, err
	}
	var out ClosureResult
	for _, d := range deps {
		switch d.Kind {
		case KindSubgraph:
			out.SubgraphIDs = append(out.SubgraphIDs, d.Key)
		case KindTemplate, KindClip:
			out.AssetGUIDs = append(out.AssetGUIDs, d.Key)
		}
	}
	return out, nil
}
