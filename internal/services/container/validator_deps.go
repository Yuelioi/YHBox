// validator_deps.go — dependency 存在性检查 (TEMPLATE_NOT_FOUND / CLIP_NOT_FOUND).
// 调用方注入 hasTemplate / hasClip 让 validator 查 asset 是否存在于容器.
package container

// ValidateContainerWithDeps 在 base validation 之上, 加 dependency 存在性检查.
// hasTemplate / hasClip 任一为 nil 时跳过对应 kind 检查.
func ValidateContainerWithDeps(
	c *Container,
	sgs []Subgraph,
	hasTemplate func(key string) bool,
	hasClip func(id string) bool,
) []ValidationError {
	errs := ValidateContainer(c, sgs)
	if hasTemplate == nil && hasClip == nil {
		return errs
	}
	// 主图 + 每个子图都扫: template/clip 节点在顶层图也是合法的, 漏扫主图会让
	// "引用了不存在的 GUID" 在主图上不报错 (运行时静默 miss).
	checkNodes := func(nodes []GraphNode, graphPath []string) {
		for _, n := range nodes {
			switch n.Kind {
			case "CheckTemplate", "ClickTemplate", "WaitTemplate":
				if hasTemplate == nil {
					continue
				}
				for _, key := range PinStringList(&n, "Templates") {
					if key == "" {
						continue
					}
					if !hasTemplate(key) {
						errs = append(errs, ValidationError{
							Severity:  SeverityError,
							Code:      CodeTemplateNotFound,
							GraphPath: graphPath,
							NodeID:    n.ID,
							Params:    map[string]any{"key": key},
						})
					}
				}
			case "PlayClip":
				if hasClip == nil {
					continue
				}
				id := PinString(&n, "ClipID")
				if id == "" {
					continue
				}
				if !hasClip(id) {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeClipNotFound,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Params:    map[string]any{"id": id},
					})
				}
			}
		}
	}
	checkNodes(c.Graph.Nodes, []string{"main"})
	for _, sg := range sgs {
		checkNodes(sg.Graph.Nodes, []string{"subgraph", sg.ID})
	}
	return errs
}
