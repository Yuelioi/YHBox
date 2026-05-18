package container

// validateCollapsedReferences (v4 §9.2 spec): two related checks.
//
// 1. COLLAPSED_REFERENCED_BY_SUBGRAPH_CALL: a regular `Subgraph` call node refers to a
//    subgraph that has isAnonymous=true (which is reserved for `CollapsedNode` use).
//    isAnonymous subgraphs MUST only be invoked through a `CollapsedNode` kind in the
//    same graph that owns the CollapsedNode.
//
// 2. COLLAPSED_PIN_BROKEN: a `CollapsedNode` references a subgraphId that doesn't exist,
//    OR the referenced subgraph is NOT isAnonymous (CollapsedNode must wrap an anonymous
//    one — if user toggled isAnonymous=false they should upgrade to Subgraph kind).
//
// Note: COLLAPSED_PIN_BROKEN per spec §9.2 also covers "internal SubgraphInput/Output
// markers don't line up with external pins". That's a deeper structural check — deferred
// here to validateInvalidPins which catches dangling edges on the call node by other means.
func validateCollapsedReferences(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	sgIndex := make(map[string]*Subgraph, len(c.Subgraphs))
	for i := range c.Subgraphs {
		sgIndex[c.Subgraphs[i].ID] = &c.Subgraphs[i]
	}

	walk := func(g Graph, path []string) []ValidationError {
		var errs []ValidationError
		for _, n := range g.Nodes {
			sgID, _ := n.Config["subgraphId"].(string)

			switch n.Kind {
			case "Subgraph":
				if sgID == "" {
					continue // MISSING_SUBGRAPH reported elsewhere
				}
				sg := sgIndex[sgID]
				if sg == nil {
					continue
				}
				if sg.IsAnonymous {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedReferencedBySubgraphCall,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"subgraphId": sgID},
					})
				}

			case "CollapsedNode":
				if sgID == "" {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedPinBroken,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"reason": "missing subgraphId"},
					})
					continue
				}
				sg := sgIndex[sgID]
				if sg == nil {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedPinBroken,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"reason": "subgraph not found", "subgraphId": sgID},
					})
					continue
				}
				if !sg.IsAnonymous {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedPinBroken,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"reason": "subgraph is not anonymous — should upgrade kind to Subgraph", "subgraphId": sgID},
					})
				}
			}
		}
		return errs
	}

	var all []ValidationError
	all = append(all, walk(c.Graph, []string{"main"})...)
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		all = append(all, walk(sg.Graph, []string{"subgraph", sg.ID})...)
	}
	return all
}
