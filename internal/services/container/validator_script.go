package container

import (
	"fmt"

	scriptsvc "yotta/internal/services/script"
)

// validateScriptNodes — Script 节点编辑期校验: JS 语法 (goja 编译) + 动态输入重名.
func validateScriptNodes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError
	check := func(nodes []GraphNode, graphPath []string) {
		for i := range nodes {
			n := &nodes[i]
			if n.Kind != "Script" {
				continue
			}
			if code := PinString(n, "Code"); code != "" {
				if err := scriptsvc.CheckSyntax(code); err != nil {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeScriptParseError,
						GraphPath: graphPath, NodeID: n.ID,
						Params: map[string]any{"error": scriptsvc.AdjustWrapLine(err.Error())},
					})
				}
			}
			seen := map[string]bool{}
			for _, d := range ParseDynamicInputDecls(n) {
				if seen[d.Name] {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeScriptDuplicateInput,
						GraphPath: graphPath, NodeID: n.ID,
						Params: map[string]any{"name": d.Name},
					})
				}
				seen[d.Name] = true
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		check(sg.Graph.Nodes, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)})
	}
	return errs
}
