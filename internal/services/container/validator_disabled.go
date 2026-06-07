package container

// validateDisabledNodes checks rules for nodes marked disabled=true:
//  1. Container-level terminals (Start/MouseCalibration/EventTick) disabled → error
//     (invalid: no entry / calibration lost / listener 永不启动)
//     WindowTarget 是普通可执行节点, 允许禁用.
//  2. Loop/Switch/Race/Parallel disabled → warn (passthrough behavior is opinionated;
//     user might want to delete instead)
func validateDisabledNodes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	branchKinds := map[string]bool{"Loop": true, "Switch": true, "Race": true, "Parallel": true}
	terminalKinds := map[string]bool{"Start": true, "MouseCalibration": true, "EventTick": true}

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
					Params:    map[string]any{"nodeID": n.ID, "kind": n.Kind},
				})
				continue
			}
			if branchKinds[n.Kind] {
				errs = append(errs, ValidationError{
					Severity:  SeverityWarning,
					Code:      CodeDisabledBranchNodeWarn,
					GraphPath: append([]string(nil), path...),
					NodeID:    n.ID,
					Params:    map[string]any{"nodeID": n.ID, "kind": n.Kind},
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
