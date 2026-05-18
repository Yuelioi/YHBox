package container

// validateGetParamNodes scans subgraphs for GetParam nodes; flags paramName values
// not declared in the containing subgraph's InputParams (GETPARAM_UNKNOWN_PARAM).
//
// Main-graph GetParam is meaningless (main has no inputParams) — flag as unknown too.
func validateGetParamNodes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	checkNodes := func(g Graph, paramSet map[string]bool, path []string) []ValidationError {
		var errs []ValidationError
		for _, n := range g.Nodes {
			if n.Kind != "GetParam" {
				continue
			}
			name, _ := n.Config["paramName"].(string)
			if name == "" || !paramSet[name] {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeGetParamUnknownParam,
					GraphPath: path, NodeID: n.ID,
					Params: map[string]any{"name": name},
				})
			}
		}
		return errs
	}

	// Main graph: empty paramSet (any GetParam → unknown)
	var all []ValidationError
	all = append(all, checkNodes(c.Graph, map[string]bool{}, []string{"main"})...)

	// Each subgraph: its own InputParams.
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		paramSet := make(map[string]bool, len(sg.InputParams))
		for _, p := range sg.InputParams {
			paramSet[p.Name] = true
		}
		all = append(all, checkNodes(sg.Graph, paramSet, []string{"subgraph", sg.ID})...)
	}
	return all
}
