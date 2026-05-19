package container

import "fmt"

// validateDisabledNodes checks rules for nodes marked disabled=true:
//  1. Container-level nodes (Start/WindowTarget/MouseCalibration/OnEvent) disabled → error
//     (invalid: no entry / target lost)
//  2. Loop/Switch/Race/Parallel/Try disabled → warn (passthrough behavior is opinionated;
//     user might want to delete instead)
//
// Spec: editor-v2-quick-actions-design.md §6.3.
func validateDisabledNodes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	branchKinds := map[string]bool{"Loop": true, "Switch": true, "Race": true, "Parallel": true, "Try": true}
	terminalKinds := map[string]bool{"Start": true, "WindowTarget": true, "MouseCalibration": true, "OnEvent": true}

	var errs []ValidationError
	check := func(g Graph, path []string) {
		for _, n := range g.Nodes {
			if !n.Disabled {
				continue
			}
			if terminalKinds[n.Kind] {
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeInvalidDisabledTerminal,
					GraphPath: append([]string(nil), path...),
					NodeID:    n.ID,
					Message:   fmt.Sprintf("容器级节点 %s (kind=%s) 不允许禁用 — 禁用 Start = 容器没入口; 禁用 WindowTarget = 无目标窗口", n.ID, n.Kind),
					Params:    map[string]any{"kind": n.Kind},
				})
				continue
			}
			if branchKinds[n.Kind] {
				errs = append(errs, ValidationError{
					Severity:  SeverityWarning,
					Code:      CodeDisabledBranchNodeWarn,
					GraphPath: append([]string(nil), path...),
					NodeID:    n.ID,
					Message:   fmt.Sprintf("分支/异步节点 %s (kind=%s) 禁用走 passthrough exit pin — 行为非确定, 建议删而非禁", n.ID, n.Kind),
					Params:    map[string]any{"kind": n.Kind},
				})
			}
		}
	}
	check(c.Graph, []string{"main"})
	for _, sg := range c.Subgraphs {
		check(sg.Graph, []string{"subgraph", sg.ID})
	}
	return errs
}
