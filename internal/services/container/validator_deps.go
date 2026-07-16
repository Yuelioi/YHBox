// validator_deps.go — template dependency existence checks for legacy containers.
package container

import nodepkg "github.com/yottaapp/yotta/internal/node"

// ValidateContainerWithDeps 在 base validation 之上, 加 dependency 存在性检查.
func ValidateContainerWithDeps(
	c *Container,
	sgs []Subgraph,
	hasTemplate func(key string) bool,
) []ValidationError {
	return ValidateContainerWithDepsAndRegistry(c, sgs, hasTemplate, nodepkg.DefaultRegistrySnapshot())
}

func ValidateContainerWithDepsAndRegistry(
	c *Container,
	sgs []Subgraph,
	hasTemplate func(key string) bool,
	registry nodepkg.RegistryReader,
) []ValidationError {
	errs := ValidateContainerWithRegistry(c, sgs, registry)
	if hasTemplate == nil {
		return errs
	}
	// 主图 + 每个子图都扫: template 节点在顶层图也是合法的, 漏扫主图会让
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
			}
		}
	}
	checkNodes(c.Graph.Nodes, []string{"main"})
	for _, sg := range sgs {
		checkNodes(sg.Graph.Nodes, []string{"subgraph", sg.ID})
	}
	return errs
}
