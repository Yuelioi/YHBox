// validator_template_key.go — CheckTemplate / ClickTemplate / WaitTemplate 节点的
// template key 格式校验 (INVALID_TEMPLATE_KEY).
package container

import (
	"fmt"

	"yhbox/internal/services/template"
)


// validateTemplateKeyConfig 校验单个节点的 config["template"] 是否符合 namespace 格式.
// 只处理 CheckTemplate / ClickTemplate / WaitTemplate; 其它 kind 返回 nil.
func validateTemplateKeyConfig(n *GraphNode) []ValidationError {
	switch n.Kind {
	case "CheckTemplate", "ClickTemplate", "WaitTemplate":
	default:
		return nil
	}
	key := PinString(n, "Template")
	if key == "" {
		return nil // 未配 template，由其它规则负责
	}
	if err := template.ValidateKey(key); err != nil {
		return []ValidationError{{
			Severity: SeverityError,
			Code:     CodeInvalidTemplateKey,
			NodeID:   n.ID,
			Params:   map[string]any{"key": key, "error": err.Error()},
		}}
	}
	return nil
}

// validateTemplateKeyNodes 扫主图 + 所有子图，对每个 template 节点调 validateTemplateKeyConfig.
func validateTemplateKeyNodes(c *Container) []ValidationError {
	var errs []ValidationError

	check := func(nodes []GraphNode, graphPath []string) {
		for i := range nodes {
			n := &nodes[i]
			nodeErrs := validateTemplateKeyConfig(n)
			for j := range nodeErrs {
				nodeErrs[j].GraphPath = graphPath
			}
			errs = append(errs, nodeErrs...)
		}
	}

	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range c.Subgraphs {
		sgPath := []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}
		check(sg.Graph.Nodes, sgPath)
	}
	return errs
}
