// validator_deps.go — dependency 存在性检查 (TEMPLATE_NOT_FOUND / CLIP_NOT_FOUND).
// 调用方注入 hasTemplate / hasClip 让 validator 查 asset 是否存在于容器.
package container

// ValidateContainerWithDeps 在 base validation 之上, 加 dependency 存在性检查.
// hasTemplate / hasClip 任一为 nil 时跳过对应 kind 检查.
func ValidateContainerWithDeps(
	c *Container,
	hasTemplate func(key string) bool,
	hasClip func(id string) bool,
) []ValidationError {
	errs := ValidateContainer(c)
	if hasTemplate == nil && hasClip == nil {
		return errs
	}
	for _, sg := range c.Subgraphs {
		for _, n := range sg.Graph.Nodes {
			switch n.Kind {
			case "CheckTemplate", "ClickTemplate", "WaitTemplate":
				if hasTemplate == nil {
					continue
				}
				key, _ := n.Config["template"].(string)
				if key == "" {
					continue
				}
				if !hasTemplate(key) {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeTemplateNotFound,
						GraphPath: []string{"subgraph", sg.ID},
						NodeID:    n.ID,
						Params:    map[string]any{"key": key},
					})
				}
			case "PlayClip":
				if hasClip == nil {
					continue
				}
				id, _ := n.Config["clipID"].(string)
				if id == "" {
					continue
				}
				if !hasClip(id) {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeClipNotFound,
						GraphPath: []string{"subgraph", sg.ID},
						NodeID:    n.ID,
						Params:    map[string]any{"id": id},
					})
				}
			}
		}
	}
	return errs
}
